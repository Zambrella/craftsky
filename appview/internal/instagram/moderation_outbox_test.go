package instagram

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

const moderationOutboxRelayBaseDDL = `
CREATE TABLE moderation_outputs (
    id TEXT PRIMARY KEY,
    source_did TEXT NOT NULL,
    subject_did TEXT NOT NULL
);
CREATE TABLE owner_lifecycles (
    owner_did TEXT PRIMARY KEY,
    state TEXT NOT NULL,
    generation BIGINT NOT NULL,
    auth_epoch BIGINT NOT NULL,
    transition_reason TEXT NOT NULL,
    transitioned_at TIMESTAMPTZ NOT NULL,
    terminal_at TIMESTAMPTZ,
    purge_completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE owner_effect_attempts (
    owner_did TEXT NOT NULL,
    owner_generation BIGINT NOT NULL,
    remote_outcome TEXT NOT NULL,
    projection_disposition TEXT NOT NULL,
    repeat_forbidden BOOLEAN NOT NULL,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE instagram_reconciliation_jobs (
    id UUID PRIMARY KEY,
    owner_did TEXT NOT NULL,
    target_did TEXT,
    link_id UUID,
    import_id UUID,
    reason TEXT NOT NULL,
    status TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    lease_token UUID,
    lease_expires_at TIMESTAMPTZ,
    terminal_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE instagram_account_links (
    id UUID PRIMARY KEY,
    owner_did TEXT NOT NULL,
    state TEXT NOT NULL,
    discoverable BOOLEAN NOT NULL,
    conflict_pending BOOLEAN NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
`

func TestModerationRestorationRelayPromotesPendingAtomically(t *testing.T) {
	pool := testdb.WithSchema(t, moderationOutboxRelayDDL(t))
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	seedModerationOutbox(t, pool, now, true, true)
	relay := newModerationRestorationRelay(t, pool, now)
	relay.newID = func() uuid.UUID {
		return uuid.MustParse("30000000-0000-4000-8000-000000000001")
	}

	processed, err := relay.PromotePending(context.Background(), 10)
	if err != nil {
		t.Fatalf("PromotePending: %v", err)
	}
	if processed != 2 {
		t.Fatalf("processed = %d, want 2", processed)
	}
	assertOutboxStatus(t, pool, "output-linked", "queued", true, true)
	assertOutboxStatus(t, pool, "output-unlinked", "no_work", false, true)

	var (
		jobCount int
		reason   string
	)
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)::int, min(reason)
		FROM instagram_reconciliation_jobs
	`).Scan(&jobCount, &reason); err != nil {
		t.Fatalf("read promoted job: %v", err)
	}
	if jobCount != 1 || reason != "moderationCleared:output-linked" {
		t.Fatalf("promoted jobs = %d reason=%q", jobCount, reason)
	}

	processed, err = relay.PromotePending(context.Background(), 10)
	if err != nil || processed != 0 {
		t.Fatalf("second PromotePending = %d, %v; want 0, nil", processed, err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM instagram_reconciliation_jobs`); err != nil {
		t.Fatalf("delete promoted job: %v", err)
	}
	assertOutboxStatus(t, pool, "output-linked", "queued", false, true)
}

func TestModerationRestorationRelayRollsBackJobAndStatusTogether(t *testing.T) {
	pool := testdb.WithSchema(t, moderationOutboxRelayDDL(t))
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	seedModerationOutbox(t, pool, now, true, false)
	wantErr := errors.New("injected status failure")
	relay := newModerationRestorationRelay(t, pool, now)
	relay.beforeStatusUpdate = func(string) error { return wantErr }

	if _, err := relay.PromotePending(context.Background(), 10); !errors.Is(err, wantErr) {
		t.Fatalf("PromotePending error = %v, want injected error", err)
	}
	assertModerationRelayTableCount(t, pool, "instagram_reconciliation_jobs", 0)
	assertOutboxStatus(t, pool, "output-linked", "pending", false, false)
}

func TestModerationRestorationRelayUsesSkipLockedAcrossWorkers(t *testing.T) {
	pool := testdb.WithSchema(t, moderationOutboxRelayDDL(t))
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	seedModerationOutbox(t, pool, now, true, false)
	selected := make(chan struct{})
	release := make(chan struct{})
	first := newModerationRestorationRelay(t, pool, now)
	first.beforeStatusUpdate = func(string) error {
		close(selected)
		<-release
		return nil
	}
	second := newModerationRestorationRelay(t, pool, now)

	type result struct {
		count int
		err   error
	}
	results := make(chan result, 2)
	go func() {
		count, err := first.PromotePending(context.Background(), 1)
		results <- result{count: count, err: err}
	}()
	<-selected
	go func() {
		count, err := second.PromotePending(context.Background(), 1)
		results <- result{count: count, err: err}
	}()

	secondResult := <-results
	if secondResult.err != nil || secondResult.count != 0 {
		close(release)
		t.Fatalf("second worker = %+v, want no locked work", secondResult)
	}
	close(release)
	firstResult := <-results
	if firstResult.err != nil || firstResult.count != 1 {
		t.Fatalf("first worker = %+v, want one promoted row", firstResult)
	}
	assertModerationRelayTableCount(t, pool, "instagram_reconciliation_jobs", 1)
	assertOutboxStatus(t, pool, "output-linked", "queued", true, true)
}

func TestModerationRestorationRelayCancelsTerminalTargetBeforeQueueing(t *testing.T) {
	pool := testdb.WithSchema(t, moderationOutboxRelayDDL(t))
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	seedModerationOutbox(t, pool, now, true, false)
	if _, err := pool.Exec(context.Background(), `
		UPDATE owner_lifecycles
		SET state='terminal',terminal_at=$2,updated_at=$2
		WHERE owner_did=$1
	`, syntax.DID("did:plc:linked"), now); err != nil {
		t.Fatal(err)
	}
	relay := newModerationRestorationRelay(t, pool, now)

	processed, err := relay.PromotePending(context.Background(), 10)
	if err != nil {
		t.Fatalf("PromotePending: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	assertOutboxStatus(t, pool, "output-linked", "cancelled_target_terminal", false, true)
	assertModerationRelayTableCount(t, pool, "instagram_reconciliation_jobs", 0)
}

func TestModerationRestorationRelayHoldsTargetFenceThroughQueueCommit(t *testing.T) {
	pool := testdb.WithSchema(t, moderationOutboxRelayDDL(t))
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	seedModerationOutbox(t, pool, now, true, false)
	relay, lifecycles := newModerationRestorationRelayAndStore(t, pool, now)
	selected := make(chan struct{})
	release := make(chan struct{})
	relay.beforeStatusUpdate = func(string) error {
		close(selected)
		<-release
		return nil
	}

	promoted := make(chan error, 1)
	go func() {
		_, err := relay.PromotePending(context.Background(), 1)
		promoted <- err
	}()
	<-selected
	transitioned := make(chan error, 1)
	go func() {
		_, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
			Owner:              syntax.DID("did:plc:linked"),
			ExpectedGeneration: 1,
			To:                 ownerlifecycle.StateDeparted,
			Reason:             "test departure",
		})
		transitioned <- err
	}()

	select {
	case err := <-transitioned:
		close(release)
		t.Fatalf("departure crossed an in-flight moderation promotion: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	if err := <-promoted; err != nil {
		t.Fatalf("PromotePending: %v", err)
	}
	if err := <-transitioned; err != nil {
		t.Fatalf("departure after promotion: %v", err)
	}
	assertOutboxStatus(t, pool, "output-linked", "queued", true, true)
}

func moderationOutboxRelayDDL(t *testing.T) string {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/000044_moderation_restoration_outbox.up.sql")
	if err != nil {
		t.Fatalf("read moderation outbox migration: %v", err)
	}
	return moderationOutboxRelayBaseDDL + string(migration)
}

func newModerationRestorationRelay(
	t *testing.T,
	pool *pgxpool.Pool,
	now time.Time,
) *ModerationRestorationRelay {
	t.Helper()
	relay, _ := newModerationRestorationRelayAndStore(t, pool, now)
	return relay
}

func newModerationRestorationRelayAndStore(
	t *testing.T,
	pool *pgxpool.Pool,
	now time.Time,
) (*ModerationRestorationRelay, *ownerlifecycle.Store) {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	relay, err := NewModerationRestorationRelay(pool, lifecycles, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return relay, lifecycles
}

func seedModerationRelayOwner(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	state ownerlifecycle.State,
	now time.Time,
) {
	t.Helper()
	var terminalAt any
	if state == ownerlifecycle.StateTerminal {
		terminalAt = now
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,terminal_at,created_at,updated_at
		) VALUES($1,$2,1,1,'test',$3,$4,$3,$3)
	`, owner, state, now, terminalAt); err != nil {
		t.Fatalf("seed moderation relay owner: %v", err)
	}
}

func seedModerationOutbox(
	t *testing.T,
	pool *pgxpool.Pool,
	now time.Time,
	linked bool,
	unlinked bool,
) {
	t.Helper()
	// Kept as individual statements so the pgx pool's simple test seam does not
	// depend on parameterized multi-statement execution.
	ctx := context.Background()
	if linked {
		seedModerationRelayOwner(t, pool, syntax.DID("did:plc:linked"), ownerlifecycle.StateActive, now)
		if _, err := pool.Exec(ctx, `
			INSERT INTO moderation_outputs(id,source_did,subject_did)
			VALUES ('output-linked','did:web:moderator.example','did:plc:linked')
		`); err != nil {
			t.Fatalf("seed linked output: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO instagram_account_links(
				id,owner_did,state,discoverable,conflict_pending,updated_at
			) VALUES (
				'20000000-0000-4000-8000-000000000001','did:plc:linked',
				'active',true,false,$1
			)
		`, now); err != nil {
			t.Fatalf("seed linked account: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO moderation_restoration_outbox(
				moderation_output_id,target_did,status,created_at
			) VALUES ('output-linked','did:plc:linked','pending',$1)
		`, now); err != nil {
			t.Fatalf("seed linked intent: %v", err)
		}
	}
	if unlinked {
		seedModerationRelayOwner(t, pool, syntax.DID("did:plc:unlinked"), ownerlifecycle.StateActive, now)
		if _, err := pool.Exec(ctx, `
			INSERT INTO moderation_outputs(id,source_did,subject_did)
			VALUES ('output-unlinked','did:web:moderator.example','did:plc:unlinked')
		`); err != nil {
			t.Fatalf("seed unlinked output: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO moderation_restoration_outbox(
				moderation_output_id,target_did,status,created_at
			) VALUES ('output-unlinked','did:plc:unlinked','pending',$1)
		`, now); err != nil {
			t.Fatalf("seed unlinked intent: %v", err)
		}
	}
}

func assertOutboxStatus(
	t *testing.T,
	pool *pgxpool.Pool,
	outputID string,
	wantStatus string,
	wantJob bool,
	wantProcessed bool,
) {
	t.Helper()
	var (
		status       string
		hasJob       bool
		hasProcessed bool
	)
	if err := pool.QueryRow(context.Background(), `
		SELECT status,reconciliation_job_id IS NOT NULL,processed_at IS NOT NULL
		FROM moderation_restoration_outbox
		WHERE moderation_output_id=$1
	`, outputID).Scan(&status, &hasJob, &hasProcessed); err != nil {
		t.Fatalf("read outbox %s: %v", outputID, err)
	}
	if status != wantStatus || hasJob != wantJob || hasProcessed != wantProcessed {
		t.Fatalf(
			"outbox %s = status:%s job:%t processed:%t, want %s/%t/%t",
			outputID,
			status,
			hasJob,
			hasProcessed,
			wantStatus,
			wantJob,
			wantProcessed,
		)
	}
}

func assertModerationRelayTableCount(
	t *testing.T,
	pool *pgxpool.Pool,
	table string,
	want int,
) {
	t.Helper()
	var got int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT count(*)::int FROM "+table,
	).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}
