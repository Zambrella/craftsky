package index_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestTransactionalPipelineRollsBackServingMutationWhenJobCompletionFails(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	migration, err := os.ReadFile("../../migrations/000045_tap_ingestion_durability.up.sql")
	if err != nil {
		t.Fatalf("read Tap durability migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply Tap durability migration: %v", err)
	}
	renameMigration, err := os.ReadFile("../../migrations/000058_tap_projection_generation_column.up.sql")
	if err != nil {
		t.Fatalf("read Tap projection generation migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(renameMigration)); err != nil {
		t.Fatalf("apply Tap projection generation migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
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
		)
	`); err != nil {
		t.Fatalf("create lifecycle authority: %v", err)
	}
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	event := tap.Event{
		ID: 70, URI: "at://did:plc:atomic/social.craftsky.actor.profile/self",
		DID: "did:plc:atomic", Collection: "social.craftsky.actor.profile", Rkey: "self",
		Rev: "3aaaaaaaaaaa2", CID: "bafy-atomic", Action: "create",
		Record: json.RawMessage(`{"crafts":["sewing"]}`),
	}
	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatalf("ingest profile source: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',now(),now(),now())
	`, event.DID); err != nil {
		t.Fatalf("seed source lifecycle authority: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE tap_source_records SET projection_generation=1 WHERE uri=$1
	`, event.URI); err != nil {
		t.Fatalf("bind source lifecycle generation: %v", err)
	}
	claims, err := store.ClaimProjectionJobs(ctx, ingestion.ProjectionClaimRequest{
		Worker: "atomic-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION reject_tap_projection_completion() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.state = 'complete' THEN
				RAISE EXCEPTION 'injected completion failure';
			END IF;
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER reject_tap_projection_completion
		BEFORE UPDATE ON tap_projection_jobs
		FOR EACH ROW EXECUTE FUNCTION reject_tap_projection_completion();
	`); err != nil {
		t.Fatalf("install failure trigger: %v", err)
	}

	dispatcher := index.NewTransactionalDispatcher()
	dispatcher.Register("social.craftsky.actor.profile", index.NewCraftskyProfile(pool, noopBackfiller{}, testLogger()))
	if err := store.Project(ctx, claims[0], dispatcher.Project); err == nil {
		t.Fatal("projection unexpectedly succeeded")
	}
	var profiles int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_profiles WHERE did=$1`, event.DID).Scan(&profiles); err != nil {
		t.Fatalf("count profiles: %v", err)
	}
	if profiles != 0 {
		t.Fatalf("serving profile rows=%d; outer projection transaction did not roll back", profiles)
	}
	job, err := store.ProjectionJob(ctx, event.URI)
	if err != nil || job.State != "pending" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
}
