// appview/internal/index/craftsky_post_test.go
package index_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// craftskyPostsDDL mirrors appview/migrations/000010_craftsky_posts.up.sql.
// craftsky_profiles is needed because craftsky_posts has a FK into it.
const craftskyPostsDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE craftsky_posts (
    uri              TEXT        NOT NULL PRIMARY KEY,
    did              TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey             TEXT        NOT NULL,
    cid              TEXT        NOT NULL,

    text             TEXT        NOT NULL,
    facets           JSONB,
    images           JSONB,

    reply_root_uri   TEXT,
    reply_root_cid   TEXT,
    reply_parent_uri TEXT,
    reply_parent_cid TEXT,

    quote_uri        TEXT,
    quote_cid        TEXT,

    tags             TEXT[]      NOT NULL DEFAULT '{}',

    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (did, rkey)
);
`

// seedCraftskyMember inserts a craftsky_profiles row so a post for did
// can pass the membership check.
func seedCraftskyMember(t *testing.T, pool *pgxpool.Pool, did string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
		did, "seed"); err != nil {
		t.Fatalf("seed craftsky_profiles: %v", err)
	}
}

// fixedCreatedAt is a constant timestamp used in test events so assertions
// on `created_at` are exact. Chosen arbitrarily; not load-bearing.
const fixedCreatedAt = "2026-05-04T12:00:00Z"

// testTime parses fixedCreatedAt for comparisons that need a time.Time.
func testTime(t *testing.T) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, fixedCreatedAt)
	if err != nil {
		t.Fatalf("parse fixed time: %v", err)
	}
	return tt
}

func TestCraftskyPost_OtherCollectionIgnored(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:b/app.bsky.feed.post/k",
		CID:        "c",
		DID:        "did:plc:b",
		Rkey:       "k",
		Collection: "app.bsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("want nil for other collection; got %v", err)
	}
}

func TestCraftskyPost_UnknownAction(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:a")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:a/social.craftsky.feed.post/k",
		CID:        "c",
		DID:        "did:plc:a",
		Rkey:       "k",
		Collection: "social.craftsky.feed.post",
		Action:     "weird",
		Record:     json.RawMessage(`{"text":"hi","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err == nil {
		t.Error("want error for unknown action; got nil")
	}
}

func TestCraftskyPost_Create_NonMember_DroppedSilently(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:nm/social.craftsky.feed.post/k",
		CID:        "c",
		DID:        "did:plc:nm",
		Rkey:       "k",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"hi","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("Handle should drop non-members without error; got %v", err)
	}

	var count int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM craftsky_posts WHERE did = $1`, ev.DID).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 (non-member must not be indexed)", count)
	}
}

func TestCraftskyPost_Create_PlainText(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:m")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:m/social.craftsky.feed.post/r1",
		CID:        "bafy1",
		DID:        "did:plc:m",
		Rkey:       "r1",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "first post",
			"createdAt": "` + fixedCreatedAt + `"
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var (
		uri, did, rkey, cid, text string
		facets, images            *string
		replyRoot, replyParent    *string
		quoteURI, quoteCID        *string
		tags                      []string
		createdAt                 time.Time
	)
	err := pool.QueryRow(context.Background(), `
		SELECT uri, did, rkey, cid, text,
		       facets::text, images::text,
		       reply_root_uri, reply_parent_uri,
		       quote_uri, quote_cid,
		       tags, created_at
		FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&uri, &did, &rkey, &cid, &text,
			&facets, &images,
			&replyRoot, &replyParent,
			&quoteURI, &quoteCID,
			&tags, &createdAt)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if did != "did:plc:m" || rkey != "r1" || cid != "bafy1" {
		t.Errorf("ident = (%q,%q,%q)", did, rkey, cid)
	}
	if text != "first post" {
		t.Errorf("text = %q", text)
	}
	if facets != nil || images != nil {
		t.Errorf("facets/images should be NULL on plain text post; got facets=%v images=%v", facets, images)
	}
	if replyRoot != nil || replyParent != nil || quoteURI != nil || quoteCID != nil {
		t.Errorf("reply/quote columns should be NULL on plain text post")
	}
	if len(tags) != 0 {
		t.Errorf("tags = %v, want empty", tags)
	}
	if !createdAt.Equal(testTime(t)) {
		t.Errorf("created_at = %v, want %v", createdAt, testTime(t))
	}
}
