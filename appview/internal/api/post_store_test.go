// appview/internal/api/post_store_test.go
package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/testdb"
)

const postStoreDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE bluesky_profiles (
    did          TEXT        NOT NULL PRIMARY KEY,
    display_name TEXT,
    description  TEXT,
    avatar_cid   TEXT,
    avatar_mime  TEXT,
    banner_cid   TEXT,
    banner_mime  TEXT,
    record_cid   TEXT        NOT NULL,
    indexed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
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

func seedMember(t *testing.T, pool *pgxpool.Pool, did string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, 'seed')`, did); err != nil {
		t.Fatalf("seed member: %v", err)
	}
}

func seedBskyProfile(t *testing.T, pool *pgxpool.Pool, did, displayName, avatarCID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO bluesky_profiles (did, display_name, avatar_cid, record_cid)
		 VALUES ($1, $2, $3, 'seed')`, did, displayName, avatarCID); err != nil {
		t.Fatalf("seed bsky profile: %v", err)
	}
}

func seedPost(t *testing.T, pool *pgxpool.Pool, did, rkey, text string, indexedAt time.Time) string {
	t.Helper()
	uri := "at://" + did + "/social.craftsky.feed.post/" + rkey
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, record, created_at, indexed_at)
		VALUES ($1, $2, $3, 'bafycid', $4, '{}'::jsonb, $5, $5)`,
		uri, did, rkey, text, indexedAt); err != nil {
		t.Fatalf("seed post: %v", err)
	}
	return uri
}

func TestPostStore_ReadOne_HappyPath(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "bafyavatar")
	seedPost(t, pool, "did:plc:alice", "rk1", "hello", time.Now())

	store := api.NewPostStore(pool)
	row, err := store.ReadOne(context.Background(), "did:plc:alice", "rk1")
	if err != nil {
		t.Fatalf("ReadOne: %v", err)
	}
	if row.Text != "hello" || row.DID != "did:plc:alice" || row.Rkey != "rk1" {
		t.Errorf("row mismatch: %+v", row)
	}
	if row.AuthorDisplayName == nil || *row.AuthorDisplayName != "Alice" {
		t.Errorf("displayName = %v", row.AuthorDisplayName)
	}
	if row.AuthorAvatarCID == nil || *row.AuthorAvatarCID != "bafyavatar" {
		t.Errorf("avatarCID = %v", row.AuthorAvatarCID)
	}
}

func TestPostStore_ReadOne_NoBlueskyMirror(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	seedPost(t, pool, "did:plc:alice", "rk1", "hello", time.Now())

	store := api.NewPostStore(pool)
	row, err := store.ReadOne(context.Background(), "did:plc:alice", "rk1")
	if err != nil {
		t.Fatalf("ReadOne: %v", err)
	}
	if row.AuthorDisplayName != nil {
		t.Errorf("expected nil displayName, got %v", *row.AuthorDisplayName)
	}
	if row.AuthorAvatarCID != nil {
		t.Errorf("expected nil avatarCID, got %v", *row.AuthorAvatarCID)
	}
}

func TestPostStore_ReadOne_NotFound(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")

	store := api.NewPostStore(pool)
	_, err := store.ReadOne(context.Background(), "did:plc:alice", "missing")
	if !errors.Is(err, api.ErrPostNotFound) {
		t.Fatalf("want ErrPostNotFound, got %v", err)
	}
}

func TestPostStore_ListByAuthor_OrdersByIndexedAtDesc(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	t1 := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	seedPost(t, pool, "did:plc:alice", "rk1", "first", t1)
	seedPost(t, pool, "did:plc:alice", "rk2", "second", t2)
	seedPost(t, pool, "did:plc:alice", "rk3", "third", t3)

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListByAuthor(context.Background(), "did:plc:alice", 50, "")
	if err != nil {
		t.Fatalf("ListByAuthor: %v", err)
	}
	if cursor != "" {
		t.Errorf("want empty cursor on final page, got %q", cursor)
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	if rows[0].Rkey != "rk3" || rows[1].Rkey != "rk2" || rows[2].Rkey != "rk1" {
		t.Errorf("ordering wrong: %s,%s,%s", rows[0].Rkey, rows[1].Rkey, rows[2].Rkey)
	}
}

func TestPostStore_ListByAuthor_RespectsLimitAndPaginates(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	for i := 0; i < 5; i++ {
		seedPost(t, pool, "did:plc:alice",
			"rk"+string(rune('0'+i)),
			"p", time.Date(2026, 5, 1+i, 12, 0, 0, 0, time.UTC))
	}

	store := api.NewPostStore(pool)
	page1, cursor, err := store.ListByAuthor(context.Background(), "did:plc:alice", 2, "")
	if err != nil || len(page1) != 2 {
		t.Fatalf("page1 err=%v len=%d", err, len(page1))
	}
	if cursor == "" {
		t.Fatal("want non-empty cursor on partial page")
	}
	page2, cursor2, err := store.ListByAuthor(context.Background(), "did:plc:alice", 2, cursor)
	if err != nil || len(page2) != 2 {
		t.Fatalf("page2 err=%v len=%d", err, len(page2))
	}
	if cursor2 == "" {
		t.Fatal("want non-empty cursor after page2")
	}
	page3, cursor3, err := store.ListByAuthor(context.Background(), "did:plc:alice", 2, cursor2)
	if err != nil || len(page3) != 1 {
		t.Fatalf("page3 err=%v len=%d", err, len(page3))
	}
	if cursor3 != "" {
		t.Errorf("want empty cursor on final page, got %q", cursor3)
	}
}

func TestPostStore_ReadAuthor_HydratesFromBlueskyProfile(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "bafyAvatar")

	store := api.NewPostStore(pool)
	got, err := store.ReadAuthor(context.Background(), "did:plc:alice")
	if err != nil {
		t.Fatalf("ReadAuthor: %v", err)
	}
	if got.DisplayName == nil || *got.DisplayName != "Alice" {
		t.Errorf("displayName = %v", got.DisplayName)
	}
	if got.AvatarCID == nil || *got.AvatarCID != "bafyAvatar" {
		t.Errorf("avatarCID = %v", got.AvatarCID)
	}
}

func TestPostStore_ReadAuthor_NoBlueskyMirror_ReturnsNils(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")

	store := api.NewPostStore(pool)
	got, err := store.ReadAuthor(context.Background(), "did:plc:alice")
	if err != nil {
		t.Fatalf("ReadAuthor: %v", err)
	}
	if got.DisplayName != nil || got.AvatarCID != nil {
		t.Errorf("expected nils, got %+v", got)
	}
}

func TestPostStore_ListByAuthor_InvalidCursor(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	store := api.NewPostStore(pool)
	_, _, err := store.ListByAuthor(context.Background(), "did:plc:alice", 50, "!!!not-base64!!!")
	if !errors.Is(err, envelope.ErrInvalidCursor) {
		t.Fatalf("want ErrInvalidCursor, got %v", err)
	}
}

func TestPostStore_ListByAuthor_EmptyForUnknownAuthor(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListByAuthor(context.Background(), "did:plc:nobody", 50, "")
	if err != nil {
		t.Fatalf("want nil err for unknown author, got %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("want 0 rows, got %d", len(rows))
	}
	if cursor != "" {
		t.Errorf("want empty cursor, got %q", cursor)
	}
}

func TestPostStore_ListByAuthor_TiedIndexedAt_TieBreakByURI(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	// Three posts with the same indexed_at; pagination must split cleanly.
	tied := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	seedPost(t, pool, "did:plc:alice", "rkA", "first", tied)
	seedPost(t, pool, "did:plc:alice", "rkB", "second", tied)
	seedPost(t, pool, "did:plc:alice", "rkC", "third", tied)

	store := api.NewPostStore(pool)
	page1, cursor, err := store.ListByAuthor(context.Background(), "did:plc:alice", 2, "")
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1 len = %d", len(page1))
	}
	if cursor == "" {
		t.Fatal("want non-empty cursor after page1")
	}

	page2, cursor2, err := store.ListByAuthor(context.Background(), "did:plc:alice", 2, cursor)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("page2 len = %d", len(page2))
	}
	if cursor2 != "" {
		t.Errorf("want empty cursor on final page, got %q", cursor2)
	}

	// Together the pages should yield exactly the three rkeys, no dupes,
	// no skips.
	seen := map[string]int{}
	for _, r := range page1 {
		seen[r.Rkey]++
	}
	for _, r := range page2 {
		seen[r.Rkey]++
	}
	for _, rkey := range []string{"rkA", "rkB", "rkC"} {
		if seen[rkey] != 1 {
			t.Errorf("rkey %s seen %d times, want 1", rkey, seen[rkey])
		}
	}
}

func TestPostStore_ListByAuthor_CursorContinues_TiedIndexedAt(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	// Three posts, two share the same indexed_at — tie-break by uri DESC
	// must produce a stable, no-duplicate, no-skip ordering across pages.
	tied := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	earlier := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	seedPost(t, pool, "did:plc:alice", "rkA", "tied-A", tied) // uri ends "rkA"
	seedPost(t, pool, "did:plc:alice", "rkB", "tied-B", tied) // uri ends "rkB"
	seedPost(t, pool, "did:plc:alice", "rkC", "earlier", earlier)

	store := api.NewPostStore(pool)
	// Page size 1 forces the cursor to land between the two tied rows.
	page1, cursor1, err := store.ListByAuthor(context.Background(), "did:plc:alice", 1, "")
	if err != nil || len(page1) != 1 {
		t.Fatalf("page1 err=%v len=%d", err, len(page1))
	}
	// uri DESC means rkB sorts before rkA at the same indexed_at.
	if page1[0].Rkey != "rkB" {
		t.Fatalf("page1 rkey = %q, want rkB", page1[0].Rkey)
	}
	if cursor1 == "" {
		t.Fatal("want non-empty cursor after page1")
	}

	page2, cursor2, err := store.ListByAuthor(context.Background(), "did:plc:alice", 1, cursor1)
	if err != nil || len(page2) != 1 {
		t.Fatalf("page2 err=%v len=%d", err, len(page2))
	}
	if page2[0].Rkey != "rkA" {
		t.Errorf("page2 rkey = %q, want rkA (tied-uri-DESC continuation)", page2[0].Rkey)
	}
	if cursor2 == "" {
		t.Fatal("want non-empty cursor after page2")
	}

	page3, cursor3, err := store.ListByAuthor(context.Background(), "did:plc:alice", 1, cursor2)
	if err != nil || len(page3) != 1 {
		t.Fatalf("page3 err=%v len=%d", err, len(page3))
	}
	if page3[0].Rkey != "rkC" {
		t.Errorf("page3 rkey = %q, want rkC", page3[0].Rkey)
	}
	// cursor3 may be non-empty; we need to fetch once more to confirm exhaustion.
	page4, cursor4, err := store.ListByAuthor(context.Background(), "did:plc:alice", 1, cursor3)
	if err != nil {
		t.Fatalf("page4: %v", err)
	}
	if len(page4) != 0 {
		t.Errorf("page4 should be empty, got %d rows", len(page4))
	}
	if cursor4 != "" {
		t.Errorf("want empty cursor on exhausted page, got %q", cursor4)
	}
}
