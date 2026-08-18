package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStopBackgroundWorkersCancelsBeforeBoundedDrain(t *testing.T) {
	cancelled := make(chan struct{})
	done := make(chan struct{})
	cancel := func() {
		close(cancelled)
		close(done)
	}

	if err := stopBackgroundWorkers(cancel, time.Second, done); err != nil {
		t.Fatalf("stopBackgroundWorkers: %v", err)
	}
	select {
	case <-cancelled:
	default:
		t.Fatal("worker context was not cancelled")
	}
}

func TestStopBackgroundWorkersReturnsAtDeadline(t *testing.T) {
	done := make(chan struct{})
	start := time.Now()
	err := stopBackgroundWorkers(func() {}, 20*time.Millisecond, done)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stopBackgroundWorkers error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("bounded drain took %v", elapsed)
	}
}

func TestStopBackgroundWorkersRejectsInvalidTimeout(t *testing.T) {
	if err := stopBackgroundWorkers(func() {}, 0); err == nil {
		t.Fatal("expected invalid timeout error")
	}
}
