// appview/internal/api/post_store_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
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
CREATE TABLE craftsky_likes (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_uri TEXT        NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    subject_cid TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    UNIQUE (did, rkey)
);
CREATE TABLE craftsky_reposts (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_uri TEXT        NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    subject_cid TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    UNIQUE (did, rkey)
);
CREATE TABLE moderation_outputs (
    id                  TEXT        NOT NULL PRIMARY KEY,
    source_did          TEXT        NOT NULL,
    subject_type        TEXT        NOT NULL CHECK (subject_type IN ('post', 'account')),
    subject_did         TEXT        NOT NULL,
    subject_collection  TEXT,
    subject_rkey        TEXT,
    subject_uri         TEXT,
    value               TEXT        NOT NULL CHECK (value IN ('hide', 'takedown', 'warn')),
    action              TEXT        NOT NULL CHECK (action IN ('apply', 'negate')),
    internal_reason     TEXT,
    expires_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    indexed_at          TIMESTAMPTZ NOT NULL DEFAULT now()
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

func seedReplyPost(t *testing.T, pool *pgxpool.Pool, did, rkey, text, rootURI, parentURI string, createdAt time.Time) string {
	t.Helper()
	uri := "at://" + did + "/social.craftsky.feed.post/" + rkey
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_posts (
			uri, did, rkey, cid, text, record, created_at, indexed_at,
			reply_root_uri, reply_root_cid, reply_parent_uri, reply_parent_cid
		)
		VALUES ($1, $2, $3, 'bafycid-' || $3, $4, '{}'::jsonb, $5, $5, $6, 'rootcid', $7, 'parentcid')`,
		uri, did, rkey, text, createdAt, rootURI, parentURI); err != nil {
		t.Fatalf("seed reply post: %v", err)
	}
	return uri
}

func seedInteraction(t *testing.T, pool *pgxpool.Pool, table, did, rkey, subjectURI string, deleted bool) string {
	t.Helper()
	uri := "at://" + did + "/social.craftsky.feed." + table + "/" + rkey
	var deletedAt any
	if deleted {
		deletedAt = time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_`+table+`s (uri, did, rkey, cid, subject_uri, subject_cid, record, created_at, indexed_at, deleted_at)
		VALUES ($1, $2, $3, 'bafy' || $3, $4, 'subjectcid', '{}'::jsonb, $5, $5, $6)`,
		uri, did, rkey, subjectURI, time.Date(2026, 5, 10, 11, 0, 0, 0, time.UTC), deletedAt); err != nil {
		t.Fatalf("seed %s: %v", table, err)
	}
	return uri
}

func seedModerationOutput(t *testing.T, pool *pgxpool.Pool, subjectType, subjectDID, subjectURI, value string, createdAt time.Time) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO moderation_outputs (
			id, source_did, subject_type, subject_did, subject_collection,
			subject_rkey, subject_uri, value, action, created_at, indexed_at
		)
		VALUES (
			$1, 'did:plc:labeler', $2, $3,
			CASE WHEN $2 = 'post' THEN 'social.craftsky.feed.post' ELSE NULL END,
			CASE WHEN $2 = 'post' THEN split_part($4, '/', 5) ELSE NULL END,
			NULLIF($4, ''), $5, 'apply', $6, $6
		)`, "mod-"+subjectType+"-"+subjectDID+"-"+value+"-"+createdAt.Format("150405.000000000"), subjectType, subjectDID, subjectURI, value, createdAt); err != nil {
		t.Fatalf("seed moderation output: %v", err)
	}
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

func TestPostStore_ReadOne_AttachesWarningMetadataWithoutRawReason(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		subjectType string
		subjectURI  string
		wantKind    string
	}{
		{name: "post warn", subjectType: "post", wantKind: "post"},
		{name: "author warn", subjectType: "account", wantKind: "author"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pool := testdb.WithSchema(t, postStoreDDL)
			seedMember(t, pool, "did:plc:bob")
			uri := seedPost(t, pool, "did:plc:bob", "rk1", "warned but visible", now)
			subjectURI := ""
			if tc.subjectType == "post" {
				subjectURI = uri
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO moderation_outputs (
					id, source_did, subject_type, subject_did, subject_collection,
					subject_rkey, subject_uri, value, action, internal_reason, created_at, indexed_at
				)
				VALUES (
					$1, 'did:plc:labeler', $2, 'did:plc:bob',
					CASE WHEN $2 = 'post' THEN 'social.craftsky.feed.post' ELSE NULL END,
					CASE WHEN $2 = 'post' THEN 'rk1' ELSE NULL END,
					NULLIF($3, ''), 'warn', 'apply', 'raw unsafe reason fixture', $4, $4
				)`, "warn-"+tc.subjectType, tc.subjectType, subjectURI, now); err != nil {
				t.Fatalf("seed warn output: %v", err)
			}

			row, err := api.NewPostStore(pool).ReadOne(ctx, "did:plc:bob", "rk1")
			if err != nil {
				t.Fatalf("ReadOne: %v", err)
			}
			if row.ModerationWarningKind == nil || *row.ModerationWarningKind != tc.wantKind {
				t.Fatalf("ModerationWarningKind = %v, want %q", row.ModerationWarningKind, tc.wantKind)
			}
			resp := api.BuildPostResponse(row, syntax.Handle("bob.example"))
			data, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}
			if !strings.Contains(string(data), `"warningKind":"`+tc.wantKind+`"`) {
				t.Fatalf("response missing warning kind %q: %s", tc.wantKind, data)
			}
			if strings.Contains(string(data), "raw unsafe reason fixture") || strings.Contains(string(data), "internalReason") {
				t.Fatalf("response leaked raw moderation reason: %s", data)
			}
		})
	}
}

func TestPostStore_ReadOne_PreservesImagesJSON(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, images, record, created_at, indexed_at)
		VALUES (
			'at://did:plc:alice/social.craftsky.feed.post/rk1',
			'did:plc:alice',
			'rk1',
			'bafycid',
			'hello',
			'[{
				"cid":"bafkimage",
				"mime":"image/jpeg",
				"size":253496,
				"alt":"project photo",
				"aspectRatio":{"width":919,"height":2000}
			}]'::jsonb,
			'{}'::jsonb,
			now(),
			now()
		)`); err != nil {
		t.Fatalf("seed post with images: %v", err)
	}

	store := api.NewPostStore(pool)
	row, err := store.ReadOne(context.Background(), "did:plc:alice", "rk1")
	if err != nil {
		t.Fatalf("ReadOne: %v", err)
	}
	if len(row.Images) == 0 {
		t.Fatalf("images missing on row")
	}
	if got := string(row.Images); got == "" || got == "null" {
		t.Fatalf("images json = %q", got)
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

func TestPostStore_ReadOne_HiddenPostOrHiddenAuthorReturnsNotFound(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		moderation func(t *testing.T, pool *pgxpool.Pool, uri string)
	}{
		{name: "post hide", moderation: func(t *testing.T, pool *pgxpool.Pool, uri string) {
			seedModerationOutput(t, pool, "post", "did:plc:bob", uri, "hide", time.Now())
		}},
		{name: "author takedown", moderation: func(t *testing.T, pool *pgxpool.Pool, uri string) {
			seedModerationOutput(t, pool, "account", "did:plc:bob", "", "takedown", time.Now())
		}},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pool := testdb.WithSchema(t, postStoreDDL)
			seedMember(t, pool, "did:plc:bob")
			uri := seedPost(t, pool, "did:plc:bob", "rk1", "hidden", time.Now())
			tc.moderation(t, pool, uri)

			store := api.NewPostStore(pool)
			_, err := store.ReadOne(context.Background(), "did:plc:bob", "rk1")
			if !errors.Is(err, api.ErrPostNotFound) {
				t.Fatalf("want ErrPostNotFound, got %v", err)
			}
		})
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

func TestPostStore_ListByAuthor_ExcludesCommentsAndReplies(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol"} {
		seedMember(t, pool, did)
	}
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := seedPost(t, pool, "did:plc:alice", "root", "root", base)
	comment := seedReplyPost(t, pool, "did:plc:alice", "comment", "comment", root, root, base.Add(time.Minute))
	seedReplyPost(t, pool, "did:plc:alice", "nested", "nested", root, comment, base.Add(2*time.Minute))

	store := api.NewPostStore(pool)
	rows, _, err := store.ListByAuthor(context.Background(), "did:plc:alice", 50, "")
	if err != nil {
		t.Fatalf("ListByAuthor: %v", err)
	}
	if len(rows) != 1 || rows[0].Rkey != "root" {
		t.Fatalf("rows = %v, want only root", replyRkeys(rows))
	}
}

func TestPostStore_ListCommentsByAuthor_ReturnsCommentsAndRepliesOnly(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol"} {
		seedMember(t, pool, did)
	}
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := seedPost(t, pool, "did:plc:alice", "root", "root", base)
	comment := seedReplyPost(t, pool, "did:plc:alice", "comment", "comment", root, root, base.Add(time.Minute))
	seedReplyPost(t, pool, "did:plc:alice", "nested", "nested", root, comment, base.Add(2*time.Minute))
	seedReplyPost(t, pool, "did:plc:bob", "bob-comment", "bob", root, root, base.Add(3*time.Minute))

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListCommentsByAuthor(context.Background(), "did:plc:alice", 50, "")
	if err != nil {
		t.Fatalf("ListCommentsByAuthor: %v", err)
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want empty", cursor)
	}
	if len(rows) != 2 || rows[0].Rkey != "nested" || rows[1].Rkey != "comment" {
		t.Fatalf("rows = %v", replyRkeys(rows))
	}
}

func TestPostStore_ListCommentsByAuthor_Paginates(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := seedPost(t, pool, "did:plc:alice", "root", "root", base)
	for i := 0; i < 3; i++ {
		seedReplyPost(t, pool, "did:plc:alice", "comment"+string(rune('0'+i)), "comment", root, root, base.Add(time.Duration(i+1)*time.Hour))
	}

	store := api.NewPostStore(pool)
	page1, cursor, err := store.ListCommentsByAuthor(context.Background(), "did:plc:alice", 2, "")
	if err != nil || len(page1) != 2 {
		t.Fatalf("page1 err=%v len=%d", err, len(page1))
	}
	if cursor == "" {
		t.Fatal("want non-empty cursor")
	}
	page2, cursor2, err := store.ListCommentsByAuthor(context.Background(), "did:plc:alice", 2, cursor)
	if err != nil || len(page2) != 1 {
		t.Fatalf("page2 err=%v len=%d", err, len(page2))
	}
	if cursor2 != "" {
		t.Fatalf("cursor2 = %q, want empty", cursor2)
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

func TestPostStore_ResolveTargetAndFindActiveInteractions(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	seedMember(t, pool, "did:plc:bob")
	postURI := seedPost(t, pool, "did:plc:alice", "rk1", "hello", time.Now())
	likeURI := seedInteraction(t, pool, "like", "did:plc:bob", "like1", postURI, false)
	seedInteraction(t, pool, "repost", "did:plc:bob", "repost-deleted", postURI, true)

	store := api.NewPostStore(pool)
	target, err := store.ResolvePostTarget(context.Background(), "did:plc:alice", "rk1")
	if err != nil {
		t.Fatalf("ResolvePostTarget: %v", err)
	}
	if target.URI != postURI || target.CID != "bafycid" {
		t.Fatalf("target = %+v", target)
	}

	like, err := store.FindActiveLike(context.Background(), "did:plc:bob", postURI)
	if err != nil {
		t.Fatalf("FindActiveLike: %v", err)
	}
	if like.URI != likeURI || like.SubjectURI != postURI || like.Rkey != "like1" {
		t.Fatalf("like = %+v", like)
	}
	_, err = store.FindActiveRepost(context.Background(), "did:plc:bob", postURI)
	if !errors.Is(err, api.ErrInteractionNotFound) {
		t.Fatalf("want ErrInteractionNotFound for deleted repost, got %v", err)
	}
}

func TestPostStore_EngagementSummaries_ActiveOnlyAndViewerStates(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol", "did:plc:dave"} {
		seedMember(t, pool, did)
	}
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	post1 := seedPost(t, pool, "did:plc:alice", "p1", "one", base)
	post2 := seedPost(t, pool, "did:plc:alice", "p2", "two", base.Add(time.Hour))
	seedInteraction(t, pool, "like", "did:plc:bob", "like-p1", post1, false)
	seedInteraction(t, pool, "like", "did:plc:carol", "like-p1", post1, false)
	seedInteraction(t, pool, "like", "did:plc:dave", "like-p1-deleted", post1, true)
	seedInteraction(t, pool, "like", "did:plc:bob", "like-p2", post2, false)
	seedInteraction(t, pool, "repost", "did:plc:bob", "repost-p1", post1, false)
	seedInteraction(t, pool, "repost", "did:plc:bob", "repost-p2-deleted", post2, true)
	seedInteraction(t, pool, "repost", "did:plc:carol", "repost-p2", post2, false)
	reply1 := seedReplyPost(t, pool, "did:plc:bob", "reply1", "reply", post1, post1, base.Add(2*time.Hour))
	seedReplyPost(t, pool, "did:plc:carol", "reply2", "reply", post1, post1, base.Add(3*time.Hour))
	seedReplyPost(t, pool, "did:plc:dave", "grandchild", "nested", post1, reply1, base.Add(4*time.Hour))

	store := api.NewPostStore(pool)
	summaries, err := store.EngagementSummaries(context.Background(), "did:plc:bob", []string{post1, post2})
	if err != nil {
		t.Fatalf("EngagementSummaries: %v", err)
	}
	if got := summaries[post1]; got.LikeCount != 2 || got.RepostCount != 1 || got.ReplyCount != 3 || !got.ViewerHasLiked || !got.ViewerHasReposted || !got.ViewerHasReplied {
		t.Fatalf("post1 summary = %+v", got)
	}
	if got := summaries[post2]; got.LikeCount != 1 || got.RepostCount != 1 || got.ReplyCount != 0 || !got.ViewerHasLiked || got.ViewerHasReposted || got.ViewerHasReplied {
		t.Fatalf("post2 summary = %+v", got)
	}
}

func TestPostStore_EngagementSummaries_ViewerHasRepliedIsDirectChildOnly(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol"} {
		seedMember(t, pool, did)
	}
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := seedPost(t, pool, "did:plc:alice", "root", "root", base)
	comment := seedReplyPost(t, pool, "did:plc:carol", "comment", "comment", root, root, base.Add(time.Minute))
	reply := seedReplyPost(t, pool, "did:plc:alice", "reply", "reply", root, comment, base.Add(2*time.Minute))
	seedReplyPost(t, pool, "did:plc:bob", "nested", "nested", root, reply, base.Add(3*time.Minute))

	store := api.NewPostStore(pool)
	summaries, err := store.EngagementSummaries(context.Background(), "did:plc:bob", []string{root, comment, reply})
	if err != nil {
		t.Fatalf("EngagementSummaries: %v", err)
	}
	if got := summaries[root]; got.ViewerHasReplied {
		t.Fatalf("root summary = %+v, want viewer reply false for nested-only participation", got)
	}
	if got := summaries[comment]; !got.ViewerHasReplied {
		t.Fatalf("comment summary = %+v, want direct viewer reply true", got)
	}
	if got := summaries[reply]; !got.ViewerHasReplied {
		t.Fatalf("reply summary = %+v, want direct viewer reply true", got)
	}
}

func TestPostStore_ListCommentBranchReplies_PaginatesBranchOldestFirst(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol", "did:plc:dave"} {
		seedMember(t, pool, did)
	}
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := seedPost(t, pool, "did:plc:alice", "root", "root", base)
	comment := seedReplyPost(t, pool, "did:plc:bob", "comment", "comment", root, root, base.Add(time.Minute))
	reply1 := seedReplyPost(t, pool, "did:plc:bob", "reply1", "first", root, comment, base.Add(2*time.Minute))
	seedReplyPost(t, pool, "did:plc:carol", "reply2", "second", root, comment, base.Add(3*time.Minute))
	seedReplyPost(t, pool, "did:plc:dave", "reply3", "third", root, comment, base.Add(4*time.Minute))
	seedReplyPost(t, pool, "did:plc:bob", "grandchild", "nested", root, reply1, base.Add(5*time.Minute))
	otherComment := seedReplyPost(t, pool, "did:plc:bob", "other-comment", "other", root, root, base.Add(6*time.Minute))
	seedReplyPost(t, pool, "did:plc:carol", "other-reply", "other reply", root, otherComment, base.Add(7*time.Minute))

	store := api.NewPostStore(pool)
	page1, cursor, err := store.ListCommentBranchReplies(context.Background(), comment, root, 3, "")
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 3 || page1[0].Rkey != "reply1" || page1[1].Rkey != "reply2" || page1[2].Rkey != "reply3" {
		t.Fatalf("page1 = %v", replyRkeys(page1))
	}
	if cursor == "" {
		t.Fatal("want non-empty cursor")
	}
	if _, err := envelope.DecodeCursor(cursor); err != nil {
		t.Fatalf("cursor is not decodable by envelope helper: %v", err)
	}
	page2, cursor2, err := store.ListCommentBranchReplies(context.Background(), comment, root, 3, cursor)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 1 || page2[0].Rkey != "grandchild" {
		t.Fatalf("page2 = %v", replyRkeys(page2))
	}
	if cursor2 != "" {
		t.Fatalf("want empty final cursor, got %q", cursor2)
	}
}

func TestPostStore_ListCommentBranchRepliesAround_IncludesFocusedReplyAfterFirstPage(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol"} {
		seedMember(t, pool, did)
	}
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := seedPost(t, pool, "did:plc:alice", "root", "root", base)
	comment := seedReplyPost(t, pool, "did:plc:bob", "comment", "comment", root, root, base.Add(time.Minute))
	var focused string
	for i := 0; i < 12; i++ {
		uri := seedReplyPost(t, pool, "did:plc:carol", "reply"+string(rune('a'+i)), "reply", root, comment, base.Add(time.Duration(i+2)*time.Minute))
		if i == 10 {
			focused = uri
		}
	}

	store := api.NewPostStore(pool)
	page, cursor, err := store.ListCommentBranchRepliesAround(context.Background(), comment, root, focused, 10)
	if err != nil {
		t.Fatalf("ListCommentBranchRepliesAround: %v", err)
	}
	if got := replyRkeys(page); len(got) != 10 || got[0] != "replyb" || got[9] != "replyk" {
		t.Fatalf("page = %v", got)
	}
	if !postRowsContainURI(page, focused) {
		t.Fatalf("focused reply missing from page = %v", replyRkeys(page))
	}
	if cursor == "" {
		t.Fatal("want cursor for predictable load-more after focused slice")
	}
	nextPage, nextCursor, err := store.ListCommentBranchReplies(context.Background(), comment, root, 10, cursor)
	if err != nil {
		t.Fatalf("next page: %v", err)
	}
	if got := replyRkeys(nextPage); len(got) != 1 || got[0] != "replyl" {
		t.Fatalf("next page = %v", got)
	}
	if nextCursor != "" {
		t.Fatalf("next cursor = %q, want empty", nextCursor)
	}
}

func TestPostStore_ListRootComments_PaginatesOpaqueCursorOldestFirst(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol"} {
		seedMember(t, pool, did)
	}
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := seedPost(t, pool, "did:plc:alice", "root", "root", base)
	for i := 0; i < 12; i++ {
		did := "did:plc:bob"
		if i%2 == 1 {
			did = "did:plc:carol"
		}
		seedReplyPost(t, pool, did, "comment"+string(rune('a'+i)), "comment", root, root, base.Add(time.Duration(i+1)*time.Minute))
	}
	firstComment := seedReplyPost(t, pool, "did:plc:bob", "nested", "nested", root, root, base.Add(20*time.Minute))
	seedReplyPost(t, pool, "did:plc:carol", "nested-child", "child", root, firstComment, base.Add(21*time.Minute))

	store := api.NewPostStore(pool)
	page1, cursor, err := store.ListRootComments(context.Background(), root, "did:plc:viewer", "oldest", 10, "")
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 10 {
		t.Fatalf("page1 len = %d, want 10", len(page1))
	}
	if page1[0].Rkey != "commenta" || page1[9].Rkey != "commentj" {
		t.Fatalf("page1 rkeys = %v", replyRkeys(page1))
	}
	if cursor == "" {
		t.Fatal("want non-empty cursor after page 1")
	}
	if _, err := envelope.DecodeCursor(cursor); err != nil {
		t.Fatalf("cursor is not opaque envelope cursor: %v", err)
	}
	page2, cursor2, err := store.ListRootComments(context.Background(), root, "did:plc:viewer", "oldest", 10, cursor)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 3 || page2[0].Rkey != "commentk" || page2[1].Rkey != "commentl" || page2[2].Rkey != "nested" {
		t.Fatalf("page2 rkeys = %v", replyRkeys(page2))
	}
	if cursor2 != "" {
		t.Fatalf("want final cursor empty, got %q", cursor2)
	}
}

func TestPostStore_ListRootComments_GroupsViewerAndSortsWithinGroups(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	for _, did := range []string{"did:plc:alice", "did:plc:viewer", "did:plc:bob"} {
		seedMember(t, pool, did)
	}
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := seedPost(t, pool, "did:plc:alice", "root", "root", base)
	seedReplyPost(t, pool, "did:plc:bob", "other-early", "other early", root, root, base.Add(1*time.Minute))
	seedReplyPost(t, pool, "did:plc:viewer", "viewer-mid", "viewer mid", root, root, base.Add(2*time.Minute))
	seedReplyPost(t, pool, "did:plc:bob", "other-late", "other late", root, root, base.Add(3*time.Minute))
	seedReplyPost(t, pool, "did:plc:viewer", "viewer-late", "viewer late", root, root, base.Add(4*time.Minute))

	store := api.NewPostStore(pool)
	for _, tc := range []struct {
		name string
		sort string
		want []string
	}{
		{name: "oldest", sort: "oldest", want: []string{"viewer-mid", "viewer-late", "other-early", "other-late"}},
		{name: "follows", sort: "follows", want: []string{"viewer-mid", "viewer-late", "other-early", "other-late"}},
		{name: "newest", sort: "newest", want: []string{"viewer-late", "viewer-mid", "other-late", "other-early"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, cursor, err := store.ListRootComments(context.Background(), root, "did:plc:viewer", tc.sort, 10, "")
			if err != nil {
				t.Fatalf("ListRootComments: %v", err)
			}
			if cursor != "" {
				t.Fatalf("cursor = %q, want empty final cursor", cursor)
			}
			if got := replyRkeys(rows); !equalStrings(got, tc.want) {
				t.Fatalf("rkeys = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPostStore_ListCommentBranchReplies_RejectsIncompleteCursor(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, postStoreDDL)
	store := api.NewPostStore(pool)
	cursor, err := envelope.EncodeCursor(map[string]any{
		"createdAt": time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	_, _, err = store.ListCommentBranchReplies(context.Background(), "at://did:plc:alice/social.craftsky.feed.post/comment", "at://did:plc:alice/social.craftsky.feed.post/root", 2, cursor)
	if !errors.Is(err, envelope.ErrInvalidCursor) {
		t.Fatalf("want ErrInvalidCursor, got %v", err)
	}
}

func replyRkeys(rows []*api.PostRow) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Rkey)
	}
	return out
}

func postRowsContainURI(rows []*api.PostRow, uri string) bool {
	for _, row := range rows {
		if row.URI == uri {
			return true
		}
	}
	return false
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
