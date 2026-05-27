// appview/internal/api/follow_store_test.go
package api_test

import (
	"context"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const followStoreDDL = `
CREATE TABLE atproto_follows (
	uri         TEXT        NOT NULL PRIMARY KEY,
	did         TEXT        NOT NULL,
	rkey        TEXT        NOT NULL,
	cid         TEXT        NOT NULL,
	subject_did TEXT        NOT NULL,
	record      JSONB       NOT NULL,
	created_at  TIMESTAMPTZ NOT NULL,
	indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (did, rkey),
	UNIQUE (did, subject_did)
);
CREATE INDEX atproto_follows_did_idx ON atproto_follows (did);
CREATE INDEX atproto_follows_subject_did_idx ON atproto_follows (subject_did);
CREATE INDEX atproto_follows_did_subject_did_idx ON atproto_follows (did, subject_did);
`

func TestFollowStore_ActiveGraphSemantics(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, followStoreDDL)
	store := api.NewFollowStore(pool)
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	row := api.FollowRow{
		URI:        "at://did:plc:alice/app.bsky.graph.follow/f1",
		DID:        "did:plc:alice",
		Rkey:       "f1",
		CID:        "bafyfollow1",
		SubjectDID: "did:plc:bob",
		CreatedAt:  createdAt,
	}

	// Insert active follow.
	if err := store.UpsertActive(ctx, row, []byte(`{"subject":"did:plc:bob"}`)); err != nil {
		t.Fatalf("UpsertActive create: %v", err)
	}

	// Re-deliver same event (idempotent).
	if err := store.UpsertActive(ctx, row, []byte(`{"subject":"did:plc:bob"}`)); err != nil {
		t.Fatalf("UpsertActive replay: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_follows`).Scan(&count); err != nil {
		t.Fatalf("count follows: %v", err)
	}
	if count != 1 {
		t.Fatalf("follow row count = %d, want 1", count)
	}

	followed, err := store.ListActiveFollowedDIDs(ctx, "did:plc:alice")
	if err != nil {
		t.Fatalf("ListActiveFollowedDIDs: %v", err)
	}
	if len(followed) != 1 || followed[0] != "did:plc:bob" {
		t.Fatalf("followed = %v, want [did:plc:bob]", followed)
	}

	if err := store.DeleteActiveByURI(ctx, row.URI); err != nil {
		t.Fatalf("DeleteActiveByURI: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_follows`).Scan(&count); err != nil {
		t.Fatalf("count after delete: %v", err)
	}
	if count != 0 {
		t.Fatalf("follow row count after delete = %d, want 0", count)
	}

	followed, err = store.ListActiveFollowedDIDs(ctx, "did:plc:alice")
	if err != nil {
		t.Fatalf("ListActiveFollowedDIDs after delete: %v", err)
	}
	if len(followed) != 0 {
		t.Fatalf("followed after delete = %v, want []", followed)
	}
}

func TestFollowStore_ListActiveFollowedDIDs_OnlyActiveUnique(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, followStoreDDL)
	store := api.NewFollowStore(pool)
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	row1 := api.FollowRow{
		URI:        "at://did:plc:alice/app.bsky.graph.follow/f1",
		DID:        "did:plc:alice",
		Rkey:       "f1",
		CID:        "c1",
		SubjectDID: "did:plc:bob",
		CreatedAt:  createdAt,
	}
	row2 := api.FollowRow{
		URI:        "at://did:plc:alice/app.bsky.graph.follow/f2",
		DID:        "did:plc:alice",
		Rkey:       "f2",
		CID:        "c2",
		SubjectDID: "did:plc:carol",
		CreatedAt:  createdAt,
	}
	row1Duplicate := api.FollowRow{
		URI:        "at://did:plc:alice/app.bsky.graph.follow/f3",
		DID:        "did:plc:alice",
		Rkey:       "f3",
		CID:        "c3",
		SubjectDID: "did:plc:bob",
		CreatedAt:  createdAt,
	}

	if err := store.UpsertActive(ctx, row1, []byte(`{"subject":"did:plc:bob"}`)); err != nil {
		t.Fatalf("upsert row1: %v", err)
	}
	if err := store.UpsertActive(ctx, row2, []byte(`{"subject":"did:plc:carol"}`)); err != nil {
		t.Fatalf("upsert row2: %v", err)
	}
	if err := store.UpsertActive(ctx, row1Duplicate, []byte(`{"subject":"did:plc:bob"}`)); err != nil {
		t.Fatalf("upsert duplicate subject: %v", err)
	}

	followed, err := store.ListActiveFollowedDIDs(ctx, "did:plc:alice")
	if err != nil {
		t.Fatalf("ListActiveFollowedDIDs: %v", err)
	}
	if len(followed) != 2 {
		t.Fatalf("followed len = %d, want 2 (%v)", len(followed), followed)
	}
	if followed[0] != "did:plc:bob" || followed[1] != "did:plc:carol" {
		t.Fatalf("followed = %v, want [did:plc:bob did:plc:carol]", followed)
	}

	var countBob int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM atproto_follows WHERE did = $1 AND subject_did = $2`,
		"did:plc:alice", "did:plc:bob",
	).Scan(&countBob); err != nil {
		t.Fatalf("count bob follows: %v", err)
	}
	if countBob != 1 {
		t.Fatalf("bob follow count = %d, want 1", countBob)
	}
}
