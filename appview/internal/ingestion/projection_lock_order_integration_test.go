package ingestion

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestProjectionActorFencePrecedesSourceRowLock(t *testing.T) {
	pool := testdb.WithSchema(t, projectionLockLifecycleDDL)
	migration, err := os.ReadFile("../../migrations/000045_tap_ingestion_durability.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply Tap durability migration: %v", err)
	}
	renameMigration, err := os.ReadFile("../../migrations/000058_tap_projection_generation_column.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(renameMigration)); err != nil {
		t.Fatalf("apply Tap projection generation migration: %v", err)
	}
	store, err := NewStore(pool, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	actor := syntax.DID("did:plc:projection-lock-order")
	uri := syntax.ATURI("at://did:plc:projection-lock-order/social.craftsky.actor.profile/self")
	if _, err := store.IngestRecord(ctx, tap.Event{
		ID: 1, URI: uri, DID: actor, Collection: "social.craftsky.actor.profile", Rkey: "self",
		Rev: "3aaaaaaaaaaa2", CID: "bafy-lock-order", Action: "create",
		Record: json.RawMessage(`{"crafts":["sewing"]}`),
	}); err != nil {
		t.Fatalf("ingest source: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',now(),now(),now())
	`, actor); err != nil {
		t.Fatalf("seed lifecycle authority: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE tap_source_records SET projection_generation=1 WHERE uri=$1
	`, uri); err != nil {
		t.Fatalf("bind source lifecycle generation: %v", err)
	}
	claims, err := store.ClaimProjectionJobs(ctx, ProjectionClaimRequest{
		Worker: "lock-order-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
	store.lifecycleAware = true

	rowBlocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rowBlocker.Exec(ctx, `SELECT 1 FROM tap_source_records WHERE uri=$1 FOR UPDATE`, uri); err != nil {
		t.Fatal(err)
	}

	projectorEntered := make(chan struct{})
	releaseProjector := make(chan struct{})
	var releaseProjectorOnce sync.Once
	release := func() { releaseProjectorOnce.Do(func() { close(releaseProjector) }) }
	projectDone := make(chan error, 1)
	t.Cleanup(func() {
		_ = rowBlocker.Rollback(context.Background())
		release()
	})
	go func() {
		projectDone <- store.Project(ctx, claims[0], func(context.Context, pgx.Tx, SourceRecord) (tap.Outcome, error) {
			close(projectorEntered)
			<-releaseProjector
			return tap.Applied(), nil
		})
	}()

	key, err := ownerlifecycle.FenceKey(actor)
	if err != nil {
		t.Fatal(err)
	}
	probe, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Release()
	sharedHeld := false
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var acquiredExclusive bool
		if err := probe.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&acquiredExclusive); err != nil {
			t.Fatal(err)
		}
		if !acquiredExclusive {
			sharedHeld = true
			break
		}
		var unlocked bool
		if err := probe.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, key).Scan(&unlocked); err != nil || !unlocked {
			t.Fatalf("release projection fence probe unlocked=%t err=%v", unlocked, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !sharedHeld {
		_ = rowBlocker.Rollback(context.Background())
		release()
		<-projectDone
		t.Fatal("projection waited on its source row before acquiring the actor fence")
	}

	fencer, err := ownerlifecycle.NewFencer(pool, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	transitionEntered := make(chan struct{})
	transitionDone := make(chan error, 1)
	go func() {
		transitionDone <- fencer.WithExclusive(ctx, []syntax.DID{actor}, func(context.Context) error {
			close(transitionEntered)
			return nil
		})
	}()
	select {
	case <-transitionEntered:
		t.Fatal("exclusive transition passed a projection actor fence")
	case <-time.After(100 * time.Millisecond):
	}

	if err := rowBlocker.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-projectorEntered:
	case <-time.After(time.Second):
		t.Fatal("projection did not reach its projector after the source row was released")
	}
	select {
	case <-transitionEntered:
		t.Fatal("exclusive transition completed before the projection transaction")
	case <-time.After(100 * time.Millisecond):
	}
	release()
	if err := <-projectDone; err != nil {
		t.Fatalf("project: %v", err)
	}
	select {
	case <-transitionEntered:
	case <-time.After(time.Second):
		t.Fatal("exclusive transition did not enter after projection commit")
	}
	if err := <-transitionDone; err != nil {
		t.Fatalf("exclusive transition: %v", err)
	}
}

const projectionLockLifecycleDDL = `
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
`
