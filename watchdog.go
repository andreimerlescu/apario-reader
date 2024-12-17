package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func watch_for_signal(watchdog chan os.Signal, cancel context.CancelFunc) {
	<-watchdog
	fmt.Println("watchdog signal received")
	wg_active_tasks.PreventAdd()
	err := ch_webserver_done.Write(struct{}{})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "cant close ch_webserver_done")
	}
	closeLogFiles()
	cancel()

	if !sem_analyze_pages.IsEmpty() {
		_, _ = fmt.Fprintf(os.Stderr, "sem_analyze_pages has %d items left inside it\n", sem_analyze_pages.Len())
	}

	if !sem_db_directories.IsEmpty() {
		_, _ = fmt.Fprintf(os.Stderr, "sem_db_directories has %d items left inside it\n", sem_db_directories.Len())
	}

	ch_db_directories.Close()
	ch_cert_reloader_cancel.Close()
	ch_webserver_done.Close()

	fmt.Printf("Completed running in %d\n", time.Since(startedAt))

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq apario-reader.exe")
	default:
		cmd = exec.Command("pgrep", "apario-reader")
	}

	output, err := cmd.Output()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		return
	}

	pids := find_process_identifiers(string(output))

	for _, pid := range pids {
		terminate_process_identifier(pid)
	}

	os.Exit(0)
}

func find_process_identifiers(output string) []int {
	var pids []int

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pid, err := strconv.Atoi(fields[1])
			if err == nil {
				pids = append(pids, pid)
			} else {
				_, _ = fmt.Fprintln(os.Stderr, err)
			}
		}
	}

	return pids
}

func terminate_process_identifier(pid int) {
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error terminating PID %v: %+v", pid, err)
	}
}
