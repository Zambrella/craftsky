package db_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const mutesBlocksMigrationPreStateDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

const mutesBlocksVersion22PublicFKDDL = `
CREATE TABLE craftsky_profiles (
    did TEXT NOT NULL PRIMARY KEY,
    record_cid TEXT NOT NULL
);
CREATE TABLE craftsky_posts (
    uri TEXT NOT NULL PRIMARY KEY,
    did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL
);
CREATE TABLE craftsky_likes (
    uri TEXT NOT NULL PRIMARY KEY,
    did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL,
    subject_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE
);
CREATE TABLE craftsky_reposts (
    uri TEXT NOT NULL PRIMARY KEY,
    did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL,
    subject_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
INSERT INTO craftsky_posts (uri, did, rkey, cid)
VALUES
    ('at://did:plc:alice/social.craftsky.feed.post/alice', 'did:plc:alice', 'alice', 'post-alice'),
    ('at://did:plc:bob/social.craftsky.feed.post/target', 'did:plc:bob', 'target', 'post-target');
INSERT INTO craftsky_likes (uri, did, rkey, cid, subject_uri)
VALUES ('at://did:plc:alice/social.craftsky.feed.like/like', 'did:plc:alice', 'like', 'like-cid', 'at://did:plc:bob/social.craftsky.feed.post/target');
INSERT INTO craftsky_reposts (uri, did, rkey, cid, subject_uri)
VALUES ('at://did:plc:alice/social.craftsky.feed.repost/repost', 'did:plc:alice', 'repost', 'repost-cid', 'at://did:plc:bob/social.craftsky.feed.post/target');
`

func TestMutesBlocksMigrationUpgradesVersion22PublicMembershipFKs(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000023_mutes_blocks.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000023_mutes_blocks.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	pool := testdb.WithSchema(t, mutesBlocksVersion22PublicFKDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply up migration to version-22 schema: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did='did:plc:alice'`); err != nil {
		t.Fatalf("delete membership after upgrade: %v", err)
	}
	assertRowCount(t, pool, "craftsky_posts", 2)
	assertRowCount(t, pool, "craftsky_likes", 1)
	assertRowCount(t, pool, "craftsky_reposts", 1)

	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply down migration with retained public rows: %v", err)
	}
	for _, constraint := range []string{
		"craftsky_posts_did_fkey",
		"craftsky_likes_did_fkey",
		"craftsky_reposts_did_fkey",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Fatalf("down migration did not restore %s", constraint)
		}
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("reapply up migration after down: %v", err)
	}
}

func TestMutesBlocksMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000023_mutes_blocks.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000023_mutes_blocks.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	for _, forbidden := range []string{"block_write_intents", "craftsky_membership_activations"} {
		if strings.Contains(string(up), forbidden) {
			t.Fatalf("migration introduced forbidden table %s", forbidden)
		}
	}

	pool := testdb.WithSchema(t, mutesBlocksMigrationPreStateDDL)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}
	apply("up", up)

	for _, table := range []string{"actor_mutes", "atproto_blocks"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatalf("lookup table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s missing", table)
		}
	}

	for _, constraint := range []string{
		"actor_mutes_pkey",
		"actor_mutes_owner_did_fkey",
		"atproto_blocks_pkey",
		"atproto_blocks_blocker_did_rkey_key",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}

	for _, index := range []string{
		"actor_mutes_owner_list_idx",
		"atproto_blocks_blocker_subject_idx",
		"atproto_blocks_subject_blocker_idx",
		"atproto_blocks_owner_list_idx",
	} {
		if !indexExists(t, pool, index) {
			t.Errorf("index %s missing", index)
		}
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid')
	`); err != nil {
		t.Fatalf("insert profiles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO actor_mutes (owner_did, subject_did)
		VALUES ('did:plc:alice', 'did:plc:bob')
	`); err != nil {
		t.Fatalf("insert mute: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO actor_mutes (owner_did, subject_did)
		VALUES ('did:plc:alice', 'did:plc:bob')
	`); err == nil {
		t.Fatal("duplicate mute pair succeeded")
	}

	const firstBlock = `
		INSERT INTO atproto_blocks (
			uri, blocker_did, rkey, cid, subject_did, record, created_at
		) VALUES (
			'at://did:plc:alice/app.bsky.graph.block/one',
			'did:plc:alice', 'one', 'cid-one', 'did:plc:bob',
			'{"subject":"did:plc:bob","createdAt":"2026-07-19T00:00:00Z"}',
			'2026-07-19T00:00:00Z'
		)
	`
	if _, err := pool.Exec(ctx, firstBlock); err != nil {
		t.Fatalf("insert block: %v", err)
	}
	if _, err := pool.Exec(ctx, firstBlock); err == nil {
		t.Fatal("duplicate block URI succeeded")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_blocks (
			uri, blocker_did, rkey, cid, subject_did, record, created_at
		) VALUES (
			'at://did:plc:alice/app.bsky.graph.block/other-uri',
			'did:plc:alice', 'one', 'cid-other', 'did:plc:bob',
			'{"subject":"did:plc:bob","createdAt":"2026-07-19T00:00:01Z"}',
			'2026-07-19T00:00:01Z'
		)
	`); err == nil {
		t.Fatal("duplicate blocker/rkey succeeded")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_blocks (
			uri, blocker_did, rkey, cid, subject_did, record, created_at
		) VALUES (
			'at://did:plc:alice/app.bsky.graph.block/two',
			'did:plc:alice', 'two', 'cid-two', 'did:plc:bob',
			'{"subject":"did:plc:bob","createdAt":"2026-07-19T00:00:02Z"}',
			'2026-07-19T00:00:02Z'
		)
	`); err != nil {
		t.Fatalf("insert compatible duplicate block pair: %v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = 'did:plc:bob'`); err != nil {
		t.Fatalf("delete mute subject profile: %v", err)
	}
	assertRowCount(t, pool, "actor_mutes", 1)
	assertRowCount(t, pool, "atproto_blocks", 2)

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = 'did:plc:alice'`); err != nil {
		t.Fatalf("delete mute owner profile: %v", err)
	}
	assertRowCount(t, pool, "actor_mutes", 0)
	assertRowCount(t, pool, "atproto_blocks", 2)

	apply("down", down)
	for _, table := range []string{"actor_mutes", "atproto_blocks"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatalf("lookup dropped table %s: %v", table, err)
		}
		if exists {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	var profilesExist bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass(current_schema() || '.craftsky_profiles') IS NOT NULL`).Scan(&profilesExist); err != nil {
		t.Fatalf("lookup preserved profiles table: %v", err)
	}
	if !profilesExist {
		t.Fatal("down migration removed pre-existing craftsky_profiles")
	}

	apply("second up", up)
	for _, table := range []string{"actor_mutes", "atproto_blocks"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatalf("lookup table %s after second up: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s missing after second up", table)
		}
	}
}

func assertRowCount(t *testing.T, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s row count = %d, want %d", table, got, want)
	}
}
