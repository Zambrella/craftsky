package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const savedPostsMigrationPreStateDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    record_cid  TEXT        NOT NULL
);
CREATE TABLE craftsky_posts (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL
);
CREATE TABLE craftsky_likes (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
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
    did         TEXT        NOT NULL,
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
CREATE TABLE actor_mutes (
    owner_did   TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    subject_did TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_did, subject_did)
);
CREATE TABLE atproto_blocks (
    uri         TEXT        NOT NULL PRIMARY KEY,
    blocker_did TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_did TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (blocker_did, rkey)
);
CREATE TABLE notification_events (
    id                         UUID        NOT NULL PRIMARY KEY,
    recipient_did              TEXT        NOT NULL,
    actor_did                  TEXT        NOT NULL,
    category                   TEXT        NOT NULL CHECK (category IN ('like', 'follow', 'reply', 'mention', 'quote', 'repost', 'everythingElse')),
    subject_key                TEXT        NOT NULL,
    source_uri                 TEXT        NOT NULL,
    source_cid                 TEXT        NOT NULL,
    source_rkey                TEXT        NOT NULL,
    subject_uri                TEXT,
    subject_cid                TEXT,
    parent_uri                 TEXT,
    parent_cid                 TEXT,
    root_uri                   TEXT,
    root_cid                   TEXT,
    quoted_uri                 TEXT,
    quoted_cid                 TEXT,
    eligibility_scope          TEXT        NOT NULL CHECK (eligibility_scope IN ('everyone', 'peopleIFollow')),
    recipient_followed_actor   BOOLEAN     NOT NULL,
    push_enabled_snapshot      BOOLEAN     NOT NULL,
    state                      TEXT        NOT NULL CHECK (state IN ('active', 'retracted')),
    first_activity_at          TIMESTAMPTZ NOT NULL,
    activity_at                TIMESTAMPTZ NOT NULL,
    indexed_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    initial_push_evaluated_at  TIMESTAMPTZ NOT NULL,
    retracted_at               TIMESTAMPTZ,
    retraction_reason          TEXT,
    UNIQUE (recipient_did, actor_did, category, subject_key)
);
CREATE TABLE migration_sentinel (
    id          INTEGER     NOT NULL PRIMARY KEY
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:carol', 'carol-cid'), ('did:plc:dave', 'dave-cid');
INSERT INTO craftsky_posts (uri, did, rkey, cid)
VALUES ('at://did:plc:dave/social.craftsky.feed.post/sentinel', 'did:plc:dave', 'sentinel', 'post-sentinel-cid');
INSERT INTO craftsky_likes (uri, did, rkey, cid, subject_uri, subject_cid, record, created_at)
VALUES ('at://did:plc:carol/social.craftsky.feed.like/sentinel', 'did:plc:carol', 'sentinel-like', 'like-cid', 'at://did:plc:dave/social.craftsky.feed.post/sentinel', 'post-sentinel-cid', '{}', '2026-07-19T10:00:00Z');
INSERT INTO craftsky_reposts (uri, did, rkey, cid, subject_uri, subject_cid, record, created_at)
VALUES ('at://did:plc:carol/social.craftsky.feed.repost/sentinel', 'did:plc:carol', 'sentinel-repost', 'repost-cid', 'at://did:plc:dave/social.craftsky.feed.post/sentinel', 'post-sentinel-cid', '{}', '2026-07-19T10:01:00Z');
INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at)
VALUES ('at://did:plc:carol/app.bsky.graph.follow/sentinel', 'did:plc:carol', 'sentinel-follow', 'follow-cid', 'did:plc:dave', '{}', '2026-07-19T10:02:00Z');
INSERT INTO actor_mutes (owner_did, subject_did, created_at, updated_at)
VALUES ('did:plc:carol', 'did:plc:dave', '2026-07-19T10:03:00Z', '2026-07-19T10:03:00Z');
INSERT INTO atproto_blocks (uri, blocker_did, rkey, cid, subject_did, record, created_at)
VALUES ('at://did:plc:carol/app.bsky.graph.block/sentinel', 'did:plc:carol', 'sentinel-block', 'block-cid', 'did:plc:dave', '{}', '2026-07-19T10:04:00Z');
INSERT INTO notification_events (
    id, recipient_did, actor_did, category, subject_key, source_uri, source_cid, source_rkey,
    subject_uri, subject_cid, eligibility_scope, recipient_followed_actor, push_enabled_snapshot,
    state, first_activity_at, activity_at, initial_push_evaluated_at
) VALUES (
    '00000000-0000-4000-8000-000000000099', 'did:plc:dave', 'did:plc:carol', 'like',
    'at://did:plc:dave/social.craftsky.feed.post/sentinel',
    'at://did:plc:carol/social.craftsky.feed.like/sentinel', 'like-cid', 'sentinel-like',
    'at://did:plc:dave/social.craftsky.feed.post/sentinel', 'post-sentinel-cid', 'everyone', false, true,
    'active', '2026-07-19T10:00:00Z', '2026-07-19T10:00:00Z', '2026-07-19T10:00:01Z'
);
INSERT INTO migration_sentinel (id) VALUES (1);
`

func TestSavedPostsMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000024_saved_posts.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000024_saved_posts.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, savedPostsMigrationPreStateDDL)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}
	apply("up", up)
	assertUnrelatedSavedPostMigrationState(t, pool)

	for _, table := range []string{"saved_post_folders", "saved_posts"} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, constraint := range []string{
		"saved_post_folders_pkey",
		"saved_post_folders_owner_did_fkey",
		"saved_post_folders_owner_did_id_key",
		"saved_post_folders_name_check",
		"saved_posts_pkey",
		"saved_posts_owner_did_fkey",
		"saved_posts_post_uri_fkey",
		"saved_posts_owner_did_folder_id_fkey",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
	for _, index := range []string{
		"saved_post_folders_owner_name_idx",
		"saved_posts_owner_saved_at_idx",
		"saved_posts_owner_folder_saved_at_idx",
		"saved_posts_owner_unfiled_saved_at_idx",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s missing", index)
		}
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
		INSERT INTO craftsky_posts (uri, did, rkey, cid)
		VALUES
			('at://did:plc:bob/social.craftsky.feed.post/one', 'did:plc:bob', 'one', 'post-one'),
			('at://did:plc:bob/social.craftsky.feed.post/two', 'did:plc:bob', 'two', 'post-two'),
			('at://did:plc:bob/social.craftsky.feed.post/three', 'did:plc:bob', 'three', 'post-three');
	`); err != nil {
		t.Fatalf("insert pre-state fixtures: %v", err)
	}

	const (
		aliceFolderOne = "00000000-0000-4000-8000-000000000001"
		aliceFolderTwo = "00000000-0000-4000-8000-000000000002"
		bobFolder      = "00000000-0000-4000-8000-000000000003"
	)
	if _, err := pool.Exec(ctx, `
		INSERT INTO saved_post_folders (id, owner_did, name, created_at, updated_at)
		VALUES
			($1, 'did:plc:alice', 'Ideas', '2026-07-20T10:00:00Z', '2026-07-20T10:00:00Z'),
			($2, 'did:plc:alice', 'Ideas', '2026-07-20T10:01:00Z', '2026-07-20T10:01:00Z'),
			($3, 'did:plc:bob', 'Ideas', '2026-07-20T10:02:00Z', '2026-07-20T10:02:00Z')
	`, aliceFolderOne, aliceFolderTwo, bobFolder); err != nil {
		t.Fatalf("insert duplicate-name folders: %v", err)
	}

	const savedAt = "2026-07-20T11:00:00Z"
	if _, err := pool.Exec(ctx, `
		INSERT INTO saved_posts (owner_did, post_uri, folder_id, saved_at)
		VALUES ('did:plc:alice', 'at://did:plc:bob/social.craftsky.feed.post/one', $1, $2)
	`, aliceFolderOne, savedAt); err != nil {
		t.Fatalf("insert foldered save: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO saved_posts (owner_did, post_uri, folder_id, saved_at)
		VALUES ('did:plc:alice', 'at://did:plc:bob/social.craftsky.feed.post/one', $1, $2)
	`, aliceFolderTwo, savedAt); err == nil {
		t.Fatal("duplicate owner/post save succeeded")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO saved_posts (owner_did, post_uri, folder_id, saved_at)
		VALUES ('did:plc:alice', 'at://did:plc:bob/social.craftsky.feed.post/two', $1, $2)
	`, bobFolder, savedAt); err == nil {
		t.Fatal("cross-owner folder assignment succeeded")
	}

	if _, err := pool.Exec(ctx, `DELETE FROM saved_post_folders WHERE id = $1`, aliceFolderOne); err != nil {
		t.Fatalf("delete non-empty folder: %v", err)
	}
	var folderID *string
	var gotSavedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT folder_id::text, saved_at
		FROM saved_posts
		WHERE owner_did = 'did:plc:alice'
		  AND post_uri = 'at://did:plc:bob/social.craftsky.feed.post/one'
	`).Scan(&folderID, &gotSavedAt); err != nil {
		t.Fatalf("read save after folder delete: %v", err)
	}
	if folderID != nil {
		t.Fatalf("folder delete left assignment %q", *folderID)
	}
	wantSavedAt, err := time.Parse(time.RFC3339, savedAt)
	if err != nil {
		t.Fatalf("parse fixture savedAt: %v", err)
	}
	if !gotSavedAt.Equal(wantSavedAt) {
		t.Fatalf("savedAt changed after folder delete: got %s want %s", gotSavedAt, wantSavedAt)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO saved_posts (owner_did, post_uri, folder_id, saved_at)
		VALUES
			('did:plc:alice', 'at://did:plc:bob/social.craftsky.feed.post/two', NULL, '2026-07-20T11:01:00Z'),
			('did:plc:bob', 'at://did:plc:bob/social.craftsky.feed.post/three', $1, '2026-07-20T11:02:00Z')
	`, bobFolder); err != nil {
		t.Fatalf("insert lifecycle saves: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_posts WHERE uri = 'at://did:plc:bob/social.craftsky.feed.post/two'`); err != nil {
		t.Fatalf("delete exact saved post: %v", err)
	}
	assertSavedPostCount(t, pool, "did:plc:alice", 1)
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = 'did:plc:alice'`); err != nil {
		t.Fatalf("delete saved-state owner membership: %v", err)
	}
	assertSavedPostCount(t, pool, "did:plc:alice", 0)
	assertSavedFolderCount(t, pool, "did:plc:alice", 0)
	assertSavedPostCount(t, pool, "did:plc:bob", 1)
	assertSavedFolderCount(t, pool, "did:plc:bob", 1)

	apply("down", down)
	for _, table := range []string{"saved_posts", "saved_post_folders"} {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	if !tableExists(t, pool, "craftsky_profiles") || !tableExists(t, pool, "craftsky_posts") || !tableExists(t, pool, "migration_sentinel") {
		t.Fatal("down migration removed unrelated schema")
	}
	var sentinel int
	if err := pool.QueryRow(ctx, `SELECT id FROM migration_sentinel`).Scan(&sentinel); err != nil || sentinel != 1 {
		t.Fatalf("down migration changed unrelated data: id=%d err=%v", sentinel, err)
	}
	assertUnrelatedSavedPostMigrationState(t, pool)

	apply("second up", up)
	assertUnrelatedSavedPostMigrationState(t, pool)
	for _, table := range []string{"saved_post_folders", "saved_posts"} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing after second up", table)
		}
	}
}

func assertUnrelatedSavedPostMigrationState(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{name: "like", query: `SELECT uri || '|' || subject_uri FROM craftsky_likes`, want: "at://did:plc:carol/social.craftsky.feed.like/sentinel|at://did:plc:dave/social.craftsky.feed.post/sentinel"},
		{name: "repost", query: `SELECT uri || '|' || subject_uri FROM craftsky_reposts`, want: "at://did:plc:carol/social.craftsky.feed.repost/sentinel|at://did:plc:dave/social.craftsky.feed.post/sentinel"},
		{name: "follow", query: `SELECT uri || '|' || subject_did FROM atproto_follows`, want: "at://did:plc:carol/app.bsky.graph.follow/sentinel|did:plc:dave"},
		{name: "mute", query: `SELECT owner_did || '|' || subject_did FROM actor_mutes`, want: "did:plc:carol|did:plc:dave"},
		{name: "block", query: `SELECT uri || '|' || subject_did FROM atproto_blocks`, want: "at://did:plc:carol/app.bsky.graph.block/sentinel|did:plc:dave"},
		{name: "notification", query: `SELECT id::text || '|' || category || '|' || state || '|' || source_uri FROM notification_events`, want: "00000000-0000-4000-8000-000000000099|like|active|at://did:plc:carol/social.craftsky.feed.like/sentinel"},
	}
	for _, test := range tests {
		t.Run("preserves unrelated "+test.name, func(t *testing.T) {
			var got string
			if err := pool.QueryRow(context.Background(), test.query).Scan(&got); err != nil {
				t.Fatalf("read unrelated %s state: %v", test.name, err)
			}
			if got != test.want {
				t.Fatalf("unrelated %s state = %q, want %q", test.name, got, test.want)
			}
		})
	}
}

func tableExists(t *testing.T, pool *pgxpool.Pool, table string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
		t.Fatalf("lookup table %s: %v", table, err)
	}
	return exists
}

func assertSavedPostCount(t *testing.T, pool *pgxpool.Pool, owner string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM saved_posts WHERE owner_did = $1`, owner).Scan(&got); err != nil {
		t.Fatalf("count saved posts: %v", err)
	}
	if got != want {
		t.Fatalf("saved post count for owner = %d, want %d", got, want)
	}
}

func assertSavedFolderCount(t *testing.T, pool *pgxpool.Pool, owner string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM saved_post_folders WHERE owner_did = $1`, owner).Scan(&got); err != nil {
		t.Fatalf("count saved folders: %v", err)
	}
	if got != want {
		t.Fatalf("saved folder count for owner = %d, want %d", got, want)
	}
}
