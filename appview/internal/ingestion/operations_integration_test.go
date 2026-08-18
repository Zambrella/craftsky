package ingestion_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestOperationalBacklogsExposeBlockedAndRetryableWork(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	owner := syntax.DID("did:plc:operations")
	uri := syntax.ATURI("at://did:plc:operations/social.craftsky.feed.post/blocked")
	if _, err := store.IngestRecord(ctx, tap.Event{
		ID: 301, URI: uri, DID: owner, Collection: "social.craftsky.feed.post",
		Rkey: "blocked", Rev: "3m00000000301", CID: "bafy-blocked", Action: "create",
		Record: json.RawMessage(`{"text":"blocked","createdAt":"2026-08-14T12:00:00Z"}`),
	}); err != nil {
		t.Fatalf("ingest blocked source: %v", err)
	}
	if err := store.EnqueueRepositoryJob(ctx, owner, ingestion.RepositoryJobPDSReconcile); err != nil {
		t.Fatalf("enqueue repository reconciliation: %v", err)
	}

	projection, err := store.ListProjectionBacklog(ctx, 10)
	if err != nil || len(projection) != 1 {
		t.Fatalf("projection backlog=%+v err=%v", projection, err)
	}
	if projection[0].SourceURI != uri || projection[0].State != "blocked" ||
		projection[0].Dependency.Kind != "member_did" || projection[0].Dependency.Key != owner.String() {
		t.Fatalf("projection backlog item=%+v", projection[0])
	}
	repositories, err := store.ListRepositoryBacklog(ctx, 10)
	if err != nil || len(repositories) != 1 {
		t.Fatalf("repository backlog=%+v err=%v", repositories, err)
	}
	if repositories[0].DID != owner || repositories[0].Kind != ingestion.RepositoryJobPDSReconcile || repositories[0].State != "pending" {
		t.Fatalf("repository backlog item=%+v", repositories[0])
	}
}

func TestOperationalBacklogLimitsAreBounded(t *testing.T) {
	store := &ingestion.Store{}
	if _, err := store.ListProjectionBacklog(context.Background(), 0); err == nil {
		t.Fatal("zero projection backlog limit succeeded")
	}
	if _, err := store.ListRepositoryBacklog(context.Background(), 1001); err == nil {
		t.Fatal("oversized repository backlog limit succeeded")
	}
}
