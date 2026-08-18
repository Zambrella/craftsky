package db_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const indexMaintenancePreStateDDL = `
CREATE TABLE craftsky_posts (
    uri TEXT PRIMARY KEY
);
CREATE TABLE saved_posts (
    owner_did TEXT NOT NULL,
    post_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    saved_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (owner_did, post_uri)
);
CREATE INDEX saved_posts_owner_saved_at_idx
    ON saved_posts (owner_did, saved_at DESC, post_uri DESC);

CREATE TABLE craftsky_likes (
    uri TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    subject_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX craftsky_likes_did_subject_uri_active_unique
    ON craftsky_likes (did, subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_likes_active_subject_uri
    ON craftsky_likes (subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_likes_active_did_subject_uri
    ON craftsky_likes (did, subject_uri) WHERE deleted_at IS NULL;

CREATE TABLE craftsky_reposts (
    uri TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    subject_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX craftsky_reposts_did_subject_uri_active_unique
    ON craftsky_reposts (did, subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_reposts_active_subject_uri
    ON craftsky_reposts (subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_reposts_active_did_subject_uri
    ON craftsky_reposts (did, subject_uri) WHERE deleted_at IS NULL;

CREATE TABLE atproto_follows (
    uri TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    UNIQUE (did, subject_did)
);
CREATE INDEX atproto_follows_did_subject_did_idx
    ON atproto_follows (did, subject_did);

CREATE TABLE push_account_subscriptions (
    id UUID PRIMARY KEY
);
CREATE TABLE push_deliveries (
    id UUID PRIMARY KEY,
    account_subscription_id UUID NOT NULL
        REFERENCES push_account_subscriptions(id) ON DELETE CASCADE
);

INSERT INTO craftsky_posts(uri) VALUES ('at://did:plc:author/social.craftsky.feed.post/one');
INSERT INTO saved_posts(owner_did, post_uri, saved_at)
VALUES ('did:plc:saver', 'at://did:plc:author/social.craftsky.feed.post/one', now());
INSERT INTO craftsky_likes(uri, did, subject_uri, deleted_at) VALUES
    ('at://did:plc:one/social.craftsky.feed.like/active', 'did:plc:one', 'at://did:plc:author/social.craftsky.feed.post/one', NULL),
    ('at://did:plc:two/social.craftsky.feed.like/deleted', 'did:plc:two', 'at://did:plc:author/social.craftsky.feed.post/one', now());
INSERT INTO craftsky_reposts(uri, did, subject_uri, deleted_at) VALUES
    ('at://did:plc:one/social.craftsky.feed.repost/active', 'did:plc:one', 'at://did:plc:author/social.craftsky.feed.post/one', NULL),
    ('at://did:plc:two/social.craftsky.feed.repost/deleted', 'did:plc:two', 'at://did:plc:author/social.craftsky.feed.post/one', now());
INSERT INTO atproto_follows(uri, did, subject_did)
VALUES ('at://did:plc:one/app.bsky.graph.follow/one', 'did:plc:one', 'did:plc:target');
INSERT INTO push_account_subscriptions(id)
VALUES ('10000000-0000-4000-8000-000000000001');
INSERT INTO push_deliveries(id, account_subscription_id)
VALUES ('20000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001');
`

func TestIndexMaintenanceMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000043_index_maintenance.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000043_index_maintenance.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, indexMaintenancePreStateDDL)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertIndexMaintenanceUp(t, pool)
	assertIndexPlans(t, pool)
	assertIndexMaintenanceCascades(t, pool)

	apply("down", down)
	assertIndexMaintenanceDown(t, pool)

	apply("second up", up)
	assertIndexMaintenanceUp(t, pool)
}

func assertIndexMaintenanceUp(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for indexName, column := range map[string]string{
		"saved_posts_post_uri_idx":                    "post_uri",
		"push_deliveries_account_subscription_id_idx": "account_subscription_id",
		"craftsky_likes_subject_uri_idx":              "subject_uri",
		"craftsky_reposts_subject_uri_idx":            "subject_uri",
	} {
		assertOrdinaryLeadingIndex(t, pool, indexName, column)
	}
	for _, indexName := range []string{
		"craftsky_likes_active_did_subject_uri",
		"craftsky_reposts_active_did_subject_uri",
		"atproto_follows_did_subject_did_idx",
	} {
		if indexExists(t, pool, indexName) {
			t.Errorf("redundant index %s still exists", indexName)
		}
	}
	for _, indexName := range []string{
		"craftsky_likes_did_subject_uri_active_unique",
		"craftsky_reposts_did_subject_uri_active_unique",
		"atproto_follows_did_subject_did_key",
	} {
		assertUniqueIndex(t, pool, indexName)
	}
}

func assertIndexMaintenanceDown(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, indexName := range []string{
		"saved_posts_post_uri_idx",
		"push_deliveries_account_subscription_id_idx",
		"craftsky_likes_subject_uri_idx",
		"craftsky_reposts_subject_uri_idx",
	} {
		if indexExists(t, pool, indexName) {
			t.Errorf("new index %s remained after down migration", indexName)
		}
	}
	for _, indexName := range []string{
		"craftsky_likes_active_did_subject_uri",
		"craftsky_reposts_active_did_subject_uri",
		"atproto_follows_did_subject_did_idx",
	} {
		if !indexExists(t, pool, indexName) {
			t.Errorf("historical index %s was not restored", indexName)
		}
	}
}

func assertOrdinaryLeadingIndex(
	t *testing.T,
	pool *pgxpool.Pool,
	indexName string,
	wantColumn string,
) {
	t.Helper()
	var (
		isUnique  bool
		predicate *string
		column    string
	)
	err := pool.QueryRow(context.Background(), `
		SELECT index.indisunique,
		       pg_get_expr(index.indpred, index.indrelid),
		       attribute.attname
		FROM pg_index AS index
		JOIN pg_class AS index_class ON index_class.oid = index.indexrelid
		JOIN pg_attribute AS attribute
		  ON attribute.attrelid = index.indrelid
		 AND attribute.attnum = index.indkey[0]
		WHERE index_class.oid = to_regclass($1)
	`, indexName).Scan(&isUnique, &predicate, &column)
	if err != nil {
		t.Fatalf("inspect index %s: %v", indexName, err)
	}
	if isUnique {
		t.Errorf("index %s unexpectedly unique", indexName)
	}
	if predicate != nil {
		t.Errorf("index %s predicate = %q, want none", indexName, *predicate)
	}
	if column != wantColumn {
		t.Errorf("index %s leading column = %q, want %q", indexName, column, wantColumn)
	}
}

func assertUniqueIndex(t *testing.T, pool *pgxpool.Pool, indexName string) {
	t.Helper()
	var unique bool
	if err := pool.QueryRow(context.Background(), `
		SELECT index.indisunique
		FROM pg_index AS index
		JOIN pg_class AS index_class ON index_class.oid = index.indexrelid
		WHERE index_class.oid = to_regclass($1)
	`, indexName).Scan(&unique); err != nil {
		t.Fatalf("inspect unique index %s: %v", indexName, err)
	}
	if !unique {
		t.Errorf("index %s is not unique", indexName)
	}
}

func assertIndexPlans(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `SET enable_seqscan = off`); err != nil {
		t.Fatalf("disable sequential scan for index-usability checks: %v", err)
	}
	queries := map[string]string{
		"saved_posts_post_uri_idx": `SELECT 1 FROM saved_posts
			WHERE post_uri = 'at://did:plc:author/social.craftsky.feed.post/one'`,
		"push_deliveries_account_subscription_id_idx": `SELECT 1 FROM push_deliveries
			WHERE account_subscription_id = '10000000-0000-4000-8000-000000000001'`,
		"craftsky_likes_subject_uri_idx": `SELECT 1 FROM craftsky_likes
			WHERE subject_uri = 'at://did:plc:author/social.craftsky.feed.post/one'`,
		"craftsky_reposts_subject_uri_idx": `SELECT 1 FROM craftsky_reposts
			WHERE subject_uri = 'at://did:plc:author/social.craftsky.feed.post/one'`,
	}
	for indexName, query := range queries {
		rows, err := pool.Query(context.Background(), "EXPLAIN (COSTS OFF) "+query)
		if err != nil {
			t.Fatalf("explain %s: %v", indexName, err)
		}
		var lines []string
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				rows.Close()
				t.Fatalf("scan explain %s: %v", indexName, err)
			}
			lines = append(lines, line)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatalf("iterate explain %s: %v", indexName, err)
		}
		rows.Close()
		if plan := strings.Join(lines, "\n"); !strings.Contains(plan, indexName) {
			t.Errorf("plan does not use %s:\n%s", indexName, plan)
		}
	}
}

func assertIndexMaintenanceCascades(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		DELETE FROM craftsky_posts
		WHERE uri = 'at://did:plc:author/social.craftsky.feed.post/one'
	`); err != nil {
		t.Fatalf("delete referenced post: %v", err)
	}
	for _, table := range []string{"saved_posts", "craftsky_likes", "craftsky_reposts"} {
		assertTableCount(t, pool, table, 0)
	}
	if _, err := pool.Exec(ctx, `
		DELETE FROM push_account_subscriptions
		WHERE id = '10000000-0000-4000-8000-000000000001'
	`); err != nil {
		t.Fatalf("delete push subscription: %v", err)
	}
	assertTableCount(t, pool, "push_deliveries", 0)
}
