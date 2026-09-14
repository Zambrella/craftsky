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

func TestListenAddressUsesRenderPortWithLocalDefault(t *testing.T) {
	for _, test := range []struct {
		name    string
		port    string
		want    string
		wantErr bool
	}{
		{name: "local default", want: "0.0.0.0:8080"},
		{name: "Render port", port: "10000", want: "0.0.0.0:10000"},
		{name: "maximum port", port: "65535", want: "0.0.0.0:65535"},
		{name: "not numeric", port: "http", wantErr: true},
		{name: "zero", port: "0", wantErr: true},
		{name: "too large", port: "65536", wantErr: true},
		{name: "surrounding whitespace", port: " 8080", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := listenAddress(test.port)
			if (err != nil) != test.wantErr {
				t.Fatalf("listenAddress(%q) error = %v, wantErr %t", test.port, err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("listenAddress(%q) = %q, want %q", test.port, got, test.want)
			}
		})
	}
}
