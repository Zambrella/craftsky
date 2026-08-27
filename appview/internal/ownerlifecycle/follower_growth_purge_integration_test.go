package ownerlifecycle

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/followergrowth"
	"social.craftsky/appview/internal/testdb"
)

const (
	followerGrowthCaptureLockIDForTest int64 = 0x6372616674736b79
	followerGrowthPauseLockID          int64 = 420060
)

func TestTerminalPurgeRemovesOnlyOwnersFollowerGrowthSnapshots(t *testing.T) {
	pool, store, processor, owner, generation := newTerminalPurgeProcessorTest(t, 10)
	ctx := context.Background()
	other := syntax.DID("did:plc:follower-growth-other")
	if _, err := pool.Exec(ctx, `
		INSERT INTO follower_growth_snapshots (
			profile_did, snapshot_date, follower_count, captured_at
		) VALUES
			($1, '2026-08-24', 8, '2026-08-24T00:00:02Z'),
			($1, '2026-08-25', 9, '2026-08-25T00:00:02Z'),
			($2, '2026-08-25', 4, '2026-08-25T00:00:02Z')
	`, owner, other); err != nil {
		t.Fatalf("seed follower growth purge rows: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_follows (
			uri, did, rkey, cid, subject_did, record, created_at
		) VALUES (
			'at://did:plc:follower-growth-other/app.bsky.graph.follow/owner',
			$2, 'owner', 'follow-cid', $1, '{}', now()
		)
	`, owner, other); err != nil {
		t.Fatalf("seed public follow row: %v", err)
	}

	claim := claimSpecificTerminalComponent(
		t, store, owner, generation, "follower_growth_snapshots", "owner",
	)
	result, err := processor.ProcessClaim(ctx, claim)
	if err != nil {
		t.Fatalf("purge follower growth snapshots: %v", err)
	}
	if result.RowsAffected != 2 || !result.Complete {
		t.Fatalf("purge result = %+v, want two rows and completion", result)
	}

	var ownerRows, otherRows, followRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM follower_growth_snapshots WHERE profile_did=$1`, owner).Scan(&ownerRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM follower_growth_snapshots WHERE profile_did=$1`, other).Scan(&otherRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_follows WHERE did=$1 AND subject_did=$2`, other, owner).Scan(&followRows); err != nil {
		t.Fatal(err)
	}
	if ownerRows != 0 || otherRows != 1 || followRows != 1 {
		t.Fatalf("rows after purge: owner=%d other=%d public follows=%d", ownerRows, otherRows, followRows)
	}
}

func TestTerminalPurgeWaitsForFollowerGrowthCaptureBeforeCompleting(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllTerminalInventoryMigrations(t, pool)
	ctx := context.Background()
	fencer, err := NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycleStore, err := NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:follower-growth-capture-race")
	lifecycle, err := lifecycleStore.EnsureOnboardingOwner(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycleStore.Transition(ctx, TransitionRequest{
		Owner: owner, ExpectedGeneration: lifecycle.Generation,
		To: StateActive, Reason: "profileCreated",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'growth-race-cid')
	`, owner); err != nil {
		t.Fatalf("seed capture owner: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION pause_follower_growth_run()
		RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(420060);
			RETURN NEW;
		END;
		$$
	`); err != nil {
		t.Fatalf("create capture pause function: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE TRIGGER pause_follower_growth_run
		BEFORE INSERT ON follower_growth_snapshot_runs
		FOR EACH ROW EXECUTE FUNCTION pause_follower_growth_run()
	`); err != nil {
		t.Fatalf("prepare paused capture: %v", err)
	}

	blocker, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Release()
	if _, err := blocker.Exec(ctx, `SELECT pg_advisory_lock($1)`, followerGrowthPauseLockID); err != nil {
		t.Fatal(err)
	}
	locked := true
	defer func() {
		if !locked {
			return
		}
		var unlocked bool
		if err := blocker.QueryRow(context.Background(), `SELECT pg_advisory_unlock($1)`, followerGrowthPauseLockID).Scan(&unlocked); err != nil || !unlocked {
			t.Errorf("release capture pause lock: unlocked=%t err=%v", unlocked, err)
		}
	}()

	snapshotDate := time.Date(2026, time.August, 26, 0, 0, 0, 0, time.UTC)
	captureDone := make(chan error, 1)
	go func() {
		_, err := followergrowth.NewStore(pool).Capture(ctx, snapshotDate, snapshotDate.Add(2*time.Second))
		captureDone <- err
	}()
	waitForAdvisoryWaiter(t, pool, followerGrowthPauseLockID)

	terminal, err := lifecycleStore.Terminalize(ctx, TerminalizeRequest{
		Owner: owner, Reason: "identityDeleted",
	})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewTerminalPurgeProcessor(TerminalPurgeProcessorConfig{
		Store: lifecycleStore, WorkerID: "growth-capture-race", ComponentLimit: 100,
		RowBatchSize: 10, LeaseDuration: time.Minute, RetryDelay: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	claim := claimSpecificTerminalComponent(
		t, lifecycleStore, owner, terminal.Generation, "follower_growth_snapshots", "owner",
	)
	type purgeOutcome struct {
		result PurgeBatchResult
		err    error
	}
	purgeDone := make(chan purgeOutcome, 1)
	go func() {
		result, err := processor.ProcessClaim(ctx, claim)
		purgeDone <- purgeOutcome{result: result, err: err}
	}()

	purgeReturnedEarly := false
	deadline := time.Now().Add(2 * time.Second)
	for {
		select {
		case outcome := <-purgeDone:
			if outcome.err != nil {
				t.Fatalf("purge while capture paused: %v", outcome.err)
			}
			purgeReturnedEarly = true
		default:
		}
		if purgeReturnedEarly || hasAdvisoryWaiter(t, pool, followerGrowthCaptureLockIDForTest) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("purge neither completed nor waited for follower-growth capture")
		}
		time.Sleep(5 * time.Millisecond)
	}

	var unlocked bool
	if err := blocker.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, followerGrowthPauseLockID).Scan(&unlocked); err != nil || !unlocked {
		t.Fatalf("release capture pause lock: unlocked=%t err=%v", unlocked, err)
	}
	locked = false
	if err := <-captureDone; err != nil {
		t.Fatalf("complete paused capture: %v", err)
	}
	if purgeReturnedEarly {
		t.Fatal("terminal purge completed while a follower-growth capture could still commit private rows")
	}
	outcome := <-purgeDone
	if outcome.err != nil {
		t.Fatalf("purge after capture: %v", outcome.err)
	}
	if !outcome.result.Complete || outcome.result.RowsAffected != 1 {
		t.Fatalf("purge result = %+v, want one row and completion", outcome.result)
	}
	var remaining int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM follower_growth_snapshots WHERE profile_did=$1
	`, owner).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("owner snapshots after terminal purge = %d, want 0", remaining)
	}
}

func waitForAdvisoryWaiter(t *testing.T, pool *pgxpool.Pool, key int64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !hasAdvisoryWaiter(t, pool, key) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for advisory lock %d", key)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func hasAdvisoryWaiter(t *testing.T, pool *pgxpool.Pool, key int64) bool {
	t.Helper()
	var waiting bool
	if err := pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM pg_locks
			WHERE locktype='advisory'
			  AND classid=(($1::bigint >> 32) & 4294967295)::oid
			  AND objid=($1::bigint & 4294967295)::oid
			  AND NOT granted
		)
	`, key).Scan(&waiting); err != nil {
		t.Fatal(err)
	}
	return waiting
}
