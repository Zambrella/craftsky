package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestStartFollowerGrowthWorkerRunsImmediatelyAndStops(t *testing.T) {
	started := make(chan struct{})
	runner := recordingFollowerGrowthRunner{run: func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	done := startFollowerGrowthWorker(
		ctx,
		runner,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("follower growth worker did not start immediately")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("follower growth worker did not join shutdown")
	}
}

type recordingFollowerGrowthRunner struct {
	run func(context.Context) error
}

func (r recordingFollowerGrowthRunner) Run(ctx context.Context) error {
	return r.run(ctx)
}
