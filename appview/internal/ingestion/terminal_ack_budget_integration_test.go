package ingestion_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
)

func TestTerminalIdentityDeadlineRollsBackEveryPreAckSecurityWrite(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:terminal-deadline")
	service, err := ingestion.NewService(ingestion.ServiceConfig{
		Store: store, Lifecycles: lifecycles,
		ProfileParticipant: func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error {
			return nil
		},
		TerminalParticipant: func(ctx context.Context, _ pgx.Tx, _ *ownerlifecycle.Lifecycle, _ ownerlifecycle.Lifecycle) error {
			<-ctx.Done()
			return ctx.Err()
		},
		TerminalCommitTimeout: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	outcome, err := service.IngestIdentity(ctx, tap.IdentityEvent{
		ID: 901, DID: owner, Status: "deleted",
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("terminal identity error=%v, want deadline exceeded", err)
	}
	if outcome.Kind != tap.OutcomeRetryable || outcome.Reason != tap.ReasonStorageUnavailable {
		t.Fatalf("terminal identity outcome=%+v, want retryable storage failure", outcome)
	}
	if _, err := lifecycles.Get(context.Background(), owner); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("deadline left a lifecycle tombstone behind: %v", err)
	}
	assertIdentityReceiptCount(t, pool, 0)
	var components, cleanupJobs int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM owner_purge_components WHERE owner_did=$1
	`, owner).Scan(&components); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM auth_auxiliary_cleanup_jobs WHERE owner_did=$1
	`, owner).Scan(&cleanupJobs); err != nil {
		t.Fatal(err)
	}
	if components != 0 || cleanupJobs != 0 {
		t.Fatalf("deadline committed partial security work: components=%d cleanupJobs=%d", components, cleanupJobs)
	}
}

func TestTerminalIdentityPreAckCommitDoesNotTouchUnboundedServingRows(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:terminal-high-cardinality")
	onboarding, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: onboarding.Generation,
		To: ownerlifecycle.StateActive, Reason: "profileCreated",
	}); err != nil {
		t.Fatal(err)
	}
	const servingRows = 2001
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_posts(uri,did,cid)
		SELECT 'at://' || $1 || '/social.craftsky.feed.post/' || value::text,
		       $1,
		       'cid-' || value::text
		FROM generate_series(1,$2) AS value
	`, owner, servingRows); err != nil {
		t.Fatalf("seed serving rows: %v", err)
	}

	// Hold one serving row locked while terminalization commits. Any pre-ACK
	// scan/update/delete of the owner's physical inventory would block here;
	// the fixed tombstone/auth/ledger transaction must not touch it.
	locked, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Rollback(context.Background())
	var lockedURI string
	if err := locked.QueryRow(context.Background(), `
		SELECT uri FROM craftsky_posts WHERE did=$1 ORDER BY uri LIMIT 1 FOR UPDATE
	`, owner).Scan(&lockedURI); err != nil {
		t.Fatal(err)
	}

	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	outcome, err := service.IngestIdentity(ctx, tap.IdentityEvent{
		ID: 902, DID: owner, Status: "deleted",
	})
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("terminal identity outcome=%+v err=%v with locked row %s", outcome, err, lockedURI)
	}
	state, err := lifecycles.Get(context.Background(), owner)
	if err != nil || state.State != ownerlifecycle.StateTerminal {
		t.Fatalf("terminal lifecycle=%+v err=%v", state, err)
	}
	var retained, components int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_posts WHERE did=$1`, owner).Scan(&retained); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM owner_purge_components
		WHERE owner_did=$1 AND owner_generation=$2
	`, owner, state.Generation).Scan(&components); err != nil {
		t.Fatal(err)
	}
	if retained != servingRows || components != len(ownerlifecycle.TerminalPurgeCatalogue()) {
		t.Fatalf("pre-ACK work retained=%d/%d fixedComponents=%d/%d",
			retained, servingRows, components, len(ownerlifecycle.TerminalPurgeCatalogue()))
	}
}
