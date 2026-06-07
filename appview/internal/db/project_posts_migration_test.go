package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const projectPostsMigrationPreStateDDL = `
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

func TestProjectPostsMigrationCreatesSchemaAndIndexes(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/000016_project_posts.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	pool := testdb.WithSchema(t, projectPostsMigrationPreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	assertColumn(t, pool, "craftsky_posts", "is_project", "boolean")
	assertColumn(t, pool, "craftsky_posts", "project_craft_type", "text")
	assertColumn(t, pool, "craftsky_project_posts", "uri", "text")
	assertColumn(t, pool, "craftsky_project_posts", "raw_project", "jsonb")
	assertColumn(t, pool, "craftsky_project_posts", "common_craft_type", "text")
	assertColumn(t, pool, "craftsky_project_posts", "project_tags", "ARRAY")
	assertColumn(t, pool, "craftsky_project_posts", "knitting_project_type", "text")

	if !constraintExists(t, pool, "craftsky_project_posts_pkey") {
		t.Fatalf("craftsky_project_posts primary key missing")
	}
	if !constraintExists(t, pool, "craftsky_project_posts_uri_fkey") {
		t.Fatalf("craftsky_project_posts FK missing")
	}

	for _, name := range []string{
		"craftsky_posts_profile_projects_idx",
		"craftsky_posts_project_craft_type_idx",
		"craftsky_project_posts_common_craft_type_idx",
		"craftsky_project_posts_project_tags_gin",
	} {
		if !indexExists(t, pool, name) {
			t.Fatalf("index %s missing", name)
		}
	}

	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles (did, record_cid) VALUES ('did:plc:migrate', 'c')`); err != nil {
		t.Fatalf("insert profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, record, created_at)
		VALUES ('at://did:plc:migrate/social.craftsky.feed.post/r', 'did:plc:migrate', 'r', 'c', 't', '{"text":"t","createdAt":"2026-06-07T00:00:00Z"}', now())
	`); err != nil {
		t.Fatalf("insert post: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_project_posts (uri, raw_project, common_craft_type)
		VALUES ('at://did:plc:migrate/social.craftsky.feed.post/r', '{"common":{"craftType":"social.craftsky.feed.defs#knitting"}}', 'social.craftsky.feed.defs#knitting')
	`); err != nil {
		t.Fatalf("insert project post: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_posts WHERE uri = 'at://did:plc:migrate/social.craftsky.feed.post/r'`); err != nil {
		t.Fatalf("delete post: %v", err)
	}
	var remaining int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM craftsky_project_posts`).Scan(&remaining); err != nil {
		t.Fatalf("count project posts: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("cascade remaining rows = %d, want 0", remaining)
	}
}

func assertColumn(t *testing.T, pool *pgxpool.Pool, table, column, dataType string) {
	t.Helper()
	var got string
	if err := pool.QueryRow(context.Background(), `
		SELECT data_type
		FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2
	`, table, column).Scan(&got); err != nil {
		t.Fatalf("column %s.%s missing: %v", table, column, err)
	}
	if got != dataType {
		t.Fatalf("column %s.%s data_type = %s, want %s", table, column, got, dataType)
	}
}

func constraintExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints
			WHERE table_schema = current_schema() AND constraint_name = $1
		)
	`, name).Scan(&exists); err != nil {
		t.Fatalf("constraint lookup %s: %v", name, err)
	}
	return exists
}

func indexExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE schemaname = current_schema() AND indexname = $1
		)
	`, name).Scan(&exists); err != nil {
		t.Fatalf("index lookup %s: %v", name, err)
	}
	return exists
}
