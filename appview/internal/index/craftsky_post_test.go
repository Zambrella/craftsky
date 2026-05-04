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
	if uri != string(ev.URI) {
		t.Errorf("uri = %q, want %q", uri, ev.URI)
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

func TestCraftskyPost_Create_WithProjectPayload_StoredInRecordOnly(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:p")
	idx := index.NewCraftskyPost(pool, testLogger())

	const projectJSON = `{
		"$type": "social.craftsky.feed.post",
		"text": "finished the shawl!",
		"createdAt": "` + fixedCreatedAt + `",
		"project": {
			"common": {
				"craftType": "social.craftsky.feed.defs#knitting",
				"status":    "social.craftsky.feed.defs#finished",
				"title":     "Hitchhiker Shawl",
				"materials": ["merino"],
				"tags":      ["fair-isle"]
			}
		}
	}`
	ev := tap.Event{
		URI:        "at://did:plc:p/social.craftsky.feed.post/r",
		CID:        "bafyP",
		DID:        "did:plc:p",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(projectJSON),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// The materialised columns must NOT carry project data this pass —
	// `tags` is from facets only, the read endpoint will not see project
	// fields until the project-fields spec lands.
	var (
		tags   []string
		recRaw string
	)
	if err := pool.QueryRow(context.Background(),
		`SELECT tags, record::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&tags, &recRaw); err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("tags = %v, want empty (project tags are not yet materialised)", tags)
	}

	// The raw record column must round-trip the project payload byte-for-meaning.
	var got map[string]any
	if err := json.Unmarshal([]byte(recRaw), &got); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	project, ok := got["project"].(map[string]any)
	if !ok {
		t.Fatalf("record.project missing or not an object; got %T", got["project"])
	}
	common, ok := project["common"].(map[string]any)
	if !ok {
		t.Fatalf("record.project.common missing")
	}
	if common["craftType"] != "social.craftsky.feed.defs#knitting" {
		t.Errorf("craftType = %v", common["craftType"])
	}
	if common["title"] != "Hitchhiker Shawl" {
		t.Errorf("title = %v", common["title"])
	}
}

func TestCraftskyPost_MalformedCreatedAt_Errors(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:bad")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:bad/social.craftsky.feed.post/r",
		CID:        "bafyBAD",
		DID:        "did:plc:bad",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"x","createdAt":"not-a-timestamp"}`),
	}
	if err := idx.Handle(context.Background(), ev); err == nil {
		t.Error("want error for unparseable createdAt; got nil")
	}

	var count int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM craftsky_posts WHERE uri = $1`, ev.URI).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 (malformed event must not insert a row)", count)
	}
}
