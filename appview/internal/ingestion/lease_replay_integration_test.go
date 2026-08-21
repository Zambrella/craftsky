package ingestion_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestTombstoneRedeliveryAndExpiredProjectionLeaseAreIdempotent(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	now := time.Date(2026, 8, 14, 15, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	owner := syntax.DID("did:plc:lease-owner")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'profile')`, owner); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	uri := syntax.ATURI("at://did:plc:lease-owner/social.craftsky.feed.post/one")
	create := tap.Event{
		ID: 100, URI: uri, DID: owner, Collection: "social.craftsky.feed.post",
		Rkey: "one", Rev: "3aaaaaaaaaaa2", CID: "bafy-create", Action: "create",
		Record: json.RawMessage(`{"text":"one","createdAt":"2026-08-14T15:00:00Z"}`),
	}
	if _, err := store.IngestRecord(ctx, create); err != nil {
		t.Fatalf("ingest create: %v", err)
	}
	deleted := create
	deleted.ID = 102
	deleted.Rev = "3aaaaaaaaaaa4"
	deleted.Action = "delete"
	deleted.CID = ""
	deleted.Record = nil
	if _, err := store.IngestRecord(ctx, deleted); err != nil {
		t.Fatalf("ingest delete: %v", err)
	}
	if _, err := store.IngestRecord(ctx, create); err != nil {
		t.Fatalf("redeliver stale create: %v", err)
	}
	assertTapSourceAction(t, store, uri, "delete")

	firstToken := uuid.New()
	first, err := store.ClaimProjectionJobs(ctx, ingestion.ProjectionClaimRequest{
		Worker: "worker-old", LeaseToken: firstToken, LeaseDuration: time.Second, Limit: 1,
	})
	if err != nil || len(first) != 1 {
		t.Fatalf("first claims=%+v err=%v", first, err)
	}
	now = now.Add(2 * time.Second)
	second, err := store.ClaimProjectionJobs(ctx, ingestion.ProjectionClaimRequest{
		Worker: "worker-new", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(second) != 1 {
		t.Fatalf("reclaimed jobs=%+v err=%v", second, err)
	}
	oldProjectorCalled := false
	err = store.Project(ctx, first[0], func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
		oldProjectorCalled = true
		return tap.Applied(), nil
	})
	if !errors.Is(err, ingestion.ErrProjectionLeaseLost) || oldProjectorCalled {
		t.Fatalf("stale project err=%v called=%t", err, oldProjectorCalled)
	}
	if err := store.Project(ctx, second[0], func(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
		if source.Action != "delete" {
			t.Fatalf("reclaimed source action=%s", source.Action)
		}
		_, err := tx.Exec(ctx, `DELETE FROM craftsky_posts WHERE uri=$1`, source.URI)
		return tap.Applied(), err
	}); err != nil {
		t.Fatalf("complete reclaimed tombstone: %v", err)
	}
	job, err := store.ProjectionJob(ctx, uri)
	if err != nil || job.State != "complete" || job.Attempts != 2 {
		t.Fatalf("completed job=%+v err=%v", job, err)
	}
	var sources, receipts int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tap_source_records WHERE uri=$1`, uri).Scan(&sources); err != nil {
		t.Fatalf("count sources: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tap_ingestion_receipts WHERE source_uri=$1`, uri).Scan(&receipts); err != nil {
		t.Fatalf("count receipts: %v", err)
	}
	if sources != 1 || receipts != 2 {
		t.Fatalf("sources=%d receipts=%d, want 1 current source and create/delete receipts", sources, receipts)
	}
}
