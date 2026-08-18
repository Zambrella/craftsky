package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
)

type authRequestSweepStep struct {
	stats  auth.AuthRequestSweepStats
	err    error
	cancel context.CancelFunc
}

type scriptedAuthRequestSweeper struct {
	steps   []authRequestSweepStep
	batches []int
}

func (s *scriptedAuthRequestSweeper) SweepAuthRequests(_ context.Context, batch int) (auth.AuthRequestSweepStats, error) {
	step := s.steps[len(s.batches)]
	s.batches = append(s.batches, batch)
	if step.cancel != nil {
		step.cancel()
	}
	return step.stats, step.err
}

func TestRunAuthRequestSweeperRunsImmediatelyBacksOffAndDrains(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	oldest := time.Now().Add(-5 * time.Minute)
	sweeper := &scriptedAuthRequestSweeper{steps: []authRequestSweepStep{
		{err: errors.New("database unavailable")},
		{stats: auth.AuthRequestSweepStats{Deleted: 1, Pending: 2, OldestPendingCreatedAt: &oldest}},
		{stats: auth.AuthRequestSweepStats{Pending: 2, OldestPendingCreatedAt: &oldest}, cancel: cancel},
	}}
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{MetricRecorder: recorder})
	done := make(chan struct{})
	go func() {
		defer close(done)
		runAuthRequestSweeper(
			ctx,
			sweeper,
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			observer,
			17,
			time.Millisecond,
		)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("auth request sweeper did not stop after cancellation")
	}
	if len(sweeper.batches) != 3 {
		t.Fatalf("sweep calls=%d, want 3", len(sweeper.batches))
	}
	for _, batch := range sweeper.batches {
		if batch != 17 {
			t.Fatalf("sweep batch=%d, want 17", batch)
		}
	}

	var failures, deletions, pending, oldestAge bool
	for _, call := range recorder.Calls() {
		switch call.Name {
		case "craftsky_appview_auth_request_sweep_failures_total":
			failures = call.Value == 1
		case "craftsky_appview_auth_request_sweep_deleted_total":
			deletions = call.Value == 1
		case "craftsky_appview_auth_requests_pending":
			pending = call.Value == 2
		case "craftsky_appview_auth_requests_oldest_pending_age_seconds":
			oldestAge = call.Value > 0
		}
	}
	if !failures || !deletions || !pending || !oldestAge {
		t.Fatalf("missing sweeper signals failures=%v deletions=%v pending=%v oldestAge=%v; calls=%#v",
			failures, deletions, pending, oldestAge, recorder.Calls())
	}
}

func TestRunAuthRequestSweeperRejectsUnsafeLoopConfiguration(t *testing.T) {
	sweeper := &scriptedAuthRequestSweeper{}
	runAuthRequestSweeper(context.Background(), sweeper, nil, nil, 0, time.Second)
	runAuthRequestSweeper(context.Background(), sweeper, nil, nil, 10, 0)
	if len(sweeper.batches) != 0 {
		t.Fatalf("invalid configuration executed %d sweeps", len(sweeper.batches))
	}
}
