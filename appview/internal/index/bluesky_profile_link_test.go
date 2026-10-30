package index_test

import (
	"context"
	"encoding/json"
	"testing"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestProfileLinkPendingIconRetainsEditableRecordAndClearsOnDelete(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	source := ingestion.SourceRecord{URI: "at://did:plc:links/app.bsky.actor.link/3m5abc2345672", CID: "link-cid", DID: "did:plc:links", Collection: "app.bsky.actor.link", Rkey: "3m5abc2345672", Action: "create",
		Record: json.RawMessage(`{"$type":"app.bsky.actor.link","url":"https://example.com/","createdAt":"2026-10-27T12:00:00Z","icon":{"$type":"blob","ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"image/png","size":100}}`)}
	projector := index.NewImageSafetyBlueskyProfileLink(index.NewBlueskyProfileLink(), imagesafety.ScanKey{ScannerID: "fixture", PolicyVersion: "policy-1", CorpusVersion: "corpus-1"})
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := projector.Project(ctx, tx, source)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if outcome.Kind != tap.OutcomeBlocked {
		t.Fatalf("outcome=%+v", outcome)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var editable, blocked bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bluesky_profile_links WHERE uri=$1),EXISTS(SELECT 1 FROM image_subject_states WHERE subject_uri=$1 AND visibility_state='blocked')`, source.URI).Scan(&editable, &blocked); err != nil || !editable || !blocked {
		t.Fatalf("editable=%v blocked=%v err=%v", editable, blocked, err)
	}
	source.Action = "delete"
	source.Record = nil
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err = projector.Project(ctx, tx, source)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != tap.OutcomeApplied {
		t.Fatal(outcome)
	}
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bluesky_profile_links WHERE uri=$1),EXISTS(SELECT 1 FROM image_subject_states WHERE subject_uri=$1)`, source.URI).Scan(&editable, &blocked); err != nil || editable || blocked {
		t.Fatalf("delete left state: editable=%v blocked=%v err=%v", editable, blocked, err)
	}
}
