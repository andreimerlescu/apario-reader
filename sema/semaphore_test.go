package sema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBakersDozen(t *testing.T) {
	type args struct {
		maxConcurrency int
	}
	tests := []struct {
		name string
		args args
		want Semaphore
	}{
		{"test one", args{3}, &semaphore{semC: make(chan struct{}, 3)}},
		{"test two", args{6}, &semaphore{semC: make(chan struct{}, 6)}},
		{"test tri", args{9}, &semaphore{semC: make(chan struct{}, 9)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.maxConcurrency); got != nil {
				for i := 0; i < tt.args.maxConcurrency; i++ {
					got.Acquire()
					assert.LessOrEqual(t, i, tt.args.maxConcurrency)
					assert.Equal(t, got.Len(), i+1)
				}
			}
		})
	}
}

func TestSemaphoreAcquireRelease(t *testing.T) {
	sem := New(3)

	for i := 0; i < 10; i++ {
		sem.Acquire()
		sem.Acquire()
		sem.Acquire()

		sem.Release()
		sem.Release()
		sem.Release()
	}
}

func TestSemaphoreEmpty(t *testing.T) {
	sem := New(2)

	if !sem.IsEmpty() {
		t.Error("semaphore should be empty")
	}

	sem.Acquire()

	if sem.IsEmpty() {
		t.Error("semaphore should not be empty")
	}

	sem.Release()

	if !sem.IsEmpty() {
		t.Error("semaphore should be empty")
	}
}
