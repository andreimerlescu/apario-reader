package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
)

func closeLogFiles() {
	for name, logFile := range log_files {
		err := logFile.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "closeLogFiles %s caused err: %+v\n", name, err)
		}
	}
}

// StackFrame represents a single stack frame with its metadata
type StackFrame struct {
	File     string
	Line     int
	Function string
	Content  string
}

// getStackFrames returns stack frames starting from skip
func getStackFrames(skip, maxFrames int) []StackFrame {
	frames := make([]StackFrame, 0, maxFrames)

	for i := skip; i < skip+maxFrames; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		frame := StackFrame{
			File:     file,
			Line:     line,
			Function: fn.Name(),
		}

		// Try to get line content, but don't fail if we can't
		content, err := getLineContent(file, line)
		if err == nil {
			frame.Content = content
		}

		frames = append(frames, frame)
	}

	return frames
}

// getLineContent safely retrieves a specific line from a file
func getLineContent(filepath string, lineNum int) (string, error) {
	if lineNum < 1 {
		return "", fmt.Errorf("invalid line number: %d", lineNum)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentLine := 0

	for scanner.Scan() {
		currentLine++
		if currentLine == lineNum {
			return strings.TrimSpace(scanner.Text()), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning file: %w", err)
	}

	return "", fmt.Errorf("line %d not found in file", lineNum)
}

// formatStackTrace formats stack frames into a readable string
func formatStackTrace(frames []StackFrame) string {
	var b strings.Builder

	for _, frame := range frames {
		// Basic frame info will always be present
		fmt.Fprintf(&b, "\n\tat %s:%d (%s)", frame.File, frame.Line, frame.Function)

		// Only add the content preview if we have it
		if frame.Content != "" {
			fmt.Fprintf(&b, "\n\t  → %s", frame.Content)
		}
	}

	return b.String()
}

// CustomLogger extends the standard logger with trace capabilities
type CustomLogger struct {
	*log.Logger
	file     *os.File
	maxDepth int // configurable stack depth
}

// NewCustomLogger creates a new CustomLogger with the specified configuration
func NewCustomLogger(w io.Writer, prefix string, flag int, maxDepth int) *CustomLogger {
	if maxDepth <= 0 {
		maxDepth = 10 // sensible default
	}
	return &CustomLogger{
		Logger:   log.New(w, prefix, flag),
		maxDepth: maxDepth,
	}
}

func (l *CustomLogger) TraceReturn(v ...interface{}) error {
	msg := fmt.Sprint(v...)
	frames := getStackFrames(2, l.maxDepth) // Skip Trace() and runtime.Caller
	trace := formatStackTrace(frames)
	strace := fmt.Sprintf("%s\nStack Trace:%s\n", msg, trace)
	return l.Return(strace)
}

func (l *CustomLogger) TraceReturnf(format string, v ...interface{}) error {
	msg := fmt.Sprintf(format, v...)
	frames := getStackFrames(2, l.maxDepth) // Skip Trace() and runtime.Caller
	trace := formatStackTrace(frames)
	strace := fmt.Sprintf("%s\nStack Trace:%s\n", msg, trace)
	return l.Return(strace)
}

func (l *CustomLogger) Return(v ...interface{}) error {
	msg := fmt.Sprint(v...)
	err := l.Output(2, msg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "l.Output() err: %+v", err)
	}
	return errors.New(msg)
}

func (l *CustomLogger) Returnf(format string, v ...interface{}) error {
	msg := fmt.Sprintf(format, v...)
	err := l.Output(2, msg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "l.Output() err: %+v", err)
	}
	return errors.New(msg)
}

// Trace logs a message with a stack trace
func (l *CustomLogger) Trace(v ...interface{}) {
	msg := fmt.Sprint(v...)
	frames := getStackFrames(2, l.maxDepth) // Skip Trace() and runtime.Caller
	trace := formatStackTrace(frames)
	strace := fmt.Sprintf("%s\nStack Trace:%s\n", msg, trace)
	err := l.Output(2, strace)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "l.Output() err: %+v", err)
	}
}

// Tracef logs a formatted message with a stack trace
func (l *CustomLogger) Tracef(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	frames := getStackFrames(2, l.maxDepth) // Skip Tracef() and runtime.Caller
	trace := formatStackTrace(frames)
	err := l.Output(2, fmt.Sprintf("%s\nStack Trace:%s\n", msg, trace))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "l.Output() err: %+v", err)
	}
}
