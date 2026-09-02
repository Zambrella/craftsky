package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/testdb"
)

type authRequestSweepStep struct {
	stats  auth.AuthRequestSweepStats
	err    error
	cancel context.CancelFunc
}

type scriptedAuthRequestSweeper struct {
	steps            []authRequestSweepStep
	batches          []int
	reconcileBatches []int
}

func (s *scriptedAuthRequestSweeper) SweepAuthRequests(_ context.Context, batch int) (auth.AuthRequestSweepStats, error) {
	step := s.steps[len(s.batches)]
	s.batches = append(s.batches, batch)
	if step.cancel != nil {
		step.cancel()
	}
	return step.stats, step.err
}

func (s *scriptedAuthRequestSweeper) ReconcileStaleRegistrationExchanges(
	_ context.Context,
	batch int,
) (auth.RegistrationExchangeReconciliationStats, error) {
	s.reconcileBatches = append(s.reconcileBatches, batch)
	return auth.RegistrationExchangeReconciliationStats{}, nil
}

type restartAuthRequestSweeper struct {
	*auth.PostgresAuthStore
	t      *testing.T
	pool   *pgxpool.Pool
	cancel context.CancelFunc
	sweeps int
}

func (s *restartAuthRequestSweeper) SweepAuthRequests(ctx context.Context, batch int) (auth.AuthRequestSweepStats, error) {
	if s.sweeps == 0 {
		var unreconciled int
		if err := s.pool.QueryRow(ctx, `
			SELECT count(*) FROM oauth_auth_requests WHERE request_state='exchange_started'
		`).Scan(&unreconciled); err != nil {
			s.t.Fatal(err)
		}
		if unreconciled != 0 {
			s.t.Fatalf("ordinary sweep started with %d unreconciled registration exchanges", unreconciled)
		}
	}
	stats, err := s.PostgresAuthStore.SweepAuthRequests(ctx, batch)
	s.sweeps++
	if s.sweeps == 2 {
		s.cancel()
	}
	return stats, err
}

func TestRunAuthRequestSweeperReconcilesStaleRegistrationsBeforeOrdinarySweepAfterRestart(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE oauth_auth_request_reservations (
			id UUID PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			expires_at TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE oauth_auth_requests (
			state TEXT PRIMARY KEY,
			data JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			purpose TEXT NOT NULL,
			request_state TEXT NOT NULL,
			exchange_attempt_id UUID,
			exchange_started_at TIMESTAMPTZ,
			exchange_finished_at TIMESTAMPTZ,
			consumed_at TIMESTAMPTZ
		);
		CREATE TABLE oauth_unverified_credentials (
			request_state TEXT PRIMARY KEY REFERENCES oauth_auth_requests(state) ON DELETE CASCADE,
			data JSONB NOT NULL,
			status TEXT NOT NULL,
			eligible_at TIMESTAMPTZ NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	seedStaleRegistrationExchanges(t, pool)

	store := auth.NewPostgresAuthStore(pool, auth.StoreConfig{
		AuthRequestExpiry:            5 * time.Millisecond,
		AuthRequestExchangeExpiry:    5 * time.Millisecond,
		PendingAuthRequestCapacity:   2,
		AuthRequestTerminalRetention: time.Hour,
	})
	ctx, cancel := context.WithCancel(context.Background())
	sweeper := &restartAuthRequestSweeper{PostgresAuthStore: store, t: t, pool: pool, cancel: cancel}

	runAuthRequestSweeper(
		ctx,
		sweeper,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		observability.New(observability.Config{}),
		17,
		10*time.Millisecond,
	)

	var ambiguousState, cleanupState, credentialStatus string
	if err := pool.QueryRow(context.Background(), `SELECT request_state FROM oauth_auth_requests WHERE state='without-quarantine'`).Scan(&ambiguousState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT request.request_state,credential.status
		FROM oauth_auth_requests request
		JOIN oauth_unverified_credentials credential ON credential.request_state=request.state
		WHERE request.state='with-quarantine'
	`).Scan(&cleanupState, &credentialStatus); err != nil {
		t.Fatal(err)
	}
	if ambiguousState != string(auth.AuthRequestExchangeAmbiguous) {
		t.Fatalf("stale exchange without quarantine=%q, want exchange_ambiguous", ambiguousState)
	}
	if cleanupState != string(auth.AuthRequestCleanupPending) || credentialStatus != "pending" {
		t.Fatalf("quarantined stale exchange=%s/%s, want cleanup_pending/pending", cleanupState, credentialStatus)
	}

	if _, err := store.ReserveAuthRequestCapacity(context.Background()); err != nil {
		t.Fatalf("stale registration ambiguity did not release shared capacity: %v", err)
	}
	if _, err := store.ReserveAuthRequestCapacity(context.Background()); !errors.Is(err, auth.ErrAuthRequestCapacity) {
		t.Fatalf("second reservation error=%v, want cleanup-pending exchange to retain one shared slot", err)
	}
	var retained int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_auth_requests`).Scan(&retained); err != nil {
		t.Fatal(err)
	}
	if retained != 2 {
		t.Fatalf("retained auth-request evidence=%d, want 2", retained)
	}
}

func seedStaleRegistrationExchanges(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_auth_requests(
			state,purpose,request_state,exchange_attempt_id,created_at,exchange_started_at,consumed_at
		) VALUES
			('without-quarantine','registration','exchange_started','10000000-0000-4000-8000-000000000001',now()-interval '1 minute',now()-interval '1 minute',now()-interval '1 minute'),
			('with-quarantine','registration','exchange_started','10000000-0000-4000-8000-000000000002',now()-interval '1 minute',now()-interval '1 minute',now()-interval '1 minute');
		INSERT INTO oauth_unverified_credentials(request_state,data,status,eligible_at,expires_at)
		VALUES('with-quarantine','{}','held',now()+interval '1 hour',now()+interval '2 hours');
	`); err != nil {
		t.Fatalf("seed stale registration exchanges: %v", err)
	}
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
