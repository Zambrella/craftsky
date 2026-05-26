// appview/internal/index/bluesky_follow_test.go
package index_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const atprotoFollowsDDL = `
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

func TestBlueskyFollow_CreateIdempotent(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, atprotoFollowsDDL)
	idx := index.NewBlueskyFollow(pool)

	ev := tap.Event{
		URI:        "at://did:plc:alice/app.bsky.graph.follow/follow1",
		CID:        "bafyfollow1",
		DID:        "did:plc:alice",
		Rkey:       "follow1",
		Collection: "app.bsky.graph.follow",
		Action:     "create",
		Record: json.RawMessage(`{
			"subject": "did:plc:bob",
			"createdAt": "2026-05-25T12:00:00Z"
		}`),
	}

	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("first Handle: %v", err)
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("second Handle duplicate delivery: %v", err)
	}

	var count int
	var cid, subject string
	var createdAt time.Time
	err := pool.QueryRow(context.Background(), `
		SELECT count(*), max(cid), max(subject_did), max(created_at)
		FROM atproto_follows
	`).Scan(&count, &cid, &subject, &createdAt)
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if cid != "bafyfollow1" {
		t.Errorf("cid = %q, want bafyfollow1", cid)
	}
	if subject != "did:plc:bob" {
		t.Errorf("subject_did = %q, want did:plc:bob", subject)
	}
	wantCreatedAt := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	if !createdAt.Equal(wantCreatedAt) {
		t.Errorf("created_at = %v, want %v", createdAt, wantCreatedAt)
	}
}

func TestBlueskyFollow_UpdateUpsertsByURI(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, atprotoFollowsDDL)
	idx := index.NewBlueskyFollow(pool)

	create := tap.Event{
		URI:        "at://did:plc:alice/app.bsky.graph.follow/follow1",
		CID:        "bafyfollow1",
		DID:        "did:plc:alice",
		Rkey:       "follow1",
		Collection: "app.bsky.graph.follow",
		Action:     "create",
		Record: json.RawMessage(`{
			"subject": "did:plc:bob",
			"createdAt": "2026-05-25T12:00:00Z"
		}`),
	}
	update := create
	update.Action = "update"
	update.CID = "bafyfollow2"
	update.Record = json.RawMessage(`{
		"subject": "did:plc:carol",
		"createdAt": "2026-05-25T12:00:00Z"
	}`)

	if err := idx.Handle(context.Background(), create); err != nil {
		t.Fatalf("create Handle: %v", err)
	}
	if err := idx.Handle(context.Background(), update); err != nil {
		t.Fatalf("update Handle: %v", err)
	}

	var count int
	var cid, subject string
	err := pool.QueryRow(context.Background(), `
		SELECT count(*), max(cid), max(subject_did)
		FROM atproto_follows
		WHERE uri = $1
	`, create.URI).Scan(&count, &cid, &subject)
	if err != nil {
		t.Fatalf("select: %v", err)
	}

	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if cid != "bafyfollow2" {
		t.Errorf("cid = %q, want bafyfollow2", cid)
	}
	if subject != "did:plc:carol" {
		t.Errorf("subject_did = %q, want did:plc:carol", subject)
	}
}

func TestBlueskyFollow_DeleteRemovesRowAndUnknownDeleteIsNoop(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, atprotoFollowsDDL)
	idx := index.NewBlueskyFollow(pool)

	create := tap.Event{
		URI:        "at://did:plc:alice/app.bsky.graph.follow/follow1",
		CID:        "bafyfollow1",
		DID:        "did:plc:alice",
		Rkey:       "follow1",
		Collection: "app.bsky.graph.follow",
		Action:     "create",
		Record: json.RawMessage(`{
			"subject": "did:plc:bob",
			"createdAt": "2026-05-25T12:00:00Z"
		}`),
	}
	if err := idx.Handle(context.Background(), create); err != nil {
		t.Fatalf("create Handle: %v", err)
	}

	del := tap.Event{
		URI:        create.URI,
		DID:        create.DID,
		Rkey:       create.Rkey,
		Collection: "app.bsky.graph.follow",
		Action:     "delete",
	}
	if err := idx.Handle(context.Background(), del); err != nil {
		t.Fatalf("delete existing Handle: %v", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM atproto_follows`).Scan(&count); err != nil {
		t.Fatalf("count after delete: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}

	unknown := tap.Event{
		URI:        "at://did:plc:alice/app.bsky.graph.follow/does-not-exist",
		DID:        create.DID,
		Rkey:       "does-not-exist",
		Collection: "app.bsky.graph.follow",
		Action:     "delete",
	}
	if err := idx.Handle(context.Background(), unknown); err != nil {
		t.Fatalf("delete unknown Handle: %v", err)
	}
}

func TestBlueskyFollow_HistoricalEventCreatesActiveRow(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, atprotoFollowsDDL)
	idx := index.NewBlueskyFollow(pool)

	historical := tap.Event{
		URI:        "at://did:plc:alice/app.bsky.graph.follow/follow-historical",
		CID:        "bafyfollow-historical",
		DID:        "did:plc:alice",
		Rkey:       "follow-historical",
		Collection: "app.bsky.graph.follow",
		Action:     "create",
		Live:       false,
		Record: json.RawMessage(`{
			"subject": "did:plc:bob",
			"createdAt": "2026-05-25T12:00:00Z"
		}`),
	}

	if err := idx.Handle(context.Background(), historical); err != nil {
		t.Fatalf("Handle historical: %v", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM atproto_follows WHERE uri = $1`, historical.URI).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
