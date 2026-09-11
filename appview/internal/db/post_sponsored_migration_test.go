package db_test

import (
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestPostSponsoredMigrationDefaultsExistingRowsFalse(t *testing.T) {
	pool := testdb.WithSchema(t, projectPostsMigrationPreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, record, created_at)
		VALUES ('at://did:plc:migrate/social.craftsky.feed.post/r', 'did:plc:migrate', 'r', 'c', 't', '{}', now())
	`); err != nil {
		t.Fatalf("insert existing post: %v", err)
	}
	migration, err := os.ReadFile("../../migrations/000069_post_sponsored.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	var sponsored bool
	if err := pool.QueryRow(ctx, `SELECT sponsored FROM craftsky_posts`).Scan(&sponsored); err != nil {
		t.Fatalf("read sponsored: %v", err)
	}
	if sponsored {
		t.Fatal("existing post sponsored = true, want false")
	}
	assertColumn(t, pool, "craftsky_posts", "sponsored", "boolean")
}
