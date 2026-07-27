package db_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/testdb"
)

// IT-009, REG-002: migration 000032 is reversible and preserves existing
// ordinary rows while installing the profile chronology indexes.
func TestInstagramImportMigrationUpDownUp(t *testing.T) {
	t.Parallel()

	up, err := os.ReadFile("../../migrations/000032_instagram_post_imports.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000032_instagram_post_imports.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	var schema string
	if err := pool.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatalf("read isolated schema: %v", err)
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire migration connection: %v", err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, fmt.Sprintf(
		"SET search_path TO %s, public",
		pgx.Identifier{schema}.Sanitize(),
	)); err != nil {
		t.Fatalf("include extension schema in migration search path: %v", err)
	}
	migrationPaths, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil {
		t.Fatalf("list predecessor migrations: %v", err)
	}
	for _, path := range migrationPaths {
		if strings.HasPrefix(filepath.Base(path), "000032_") {
			break
		}
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read predecessor migration %s: %v", path, err)
		}
		if _, err := conn.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply predecessor migration %s: %v", path, err)
		}
	}
	if _, err := conn.Exec(ctx, `
		INSERT INTO craftsky_posts (
			uri, did, rkey, cid, text, record, created_at, indexed_at
		) VALUES (
			'at://did:plc:member/social.craftsky.feed.post/existing',
			'did:plc:member',
			'existing',
			'bafyexisting',
			'existing',
			'{}',
			'2019-01-02T03:04:05Z',
			'2026-07-23T10:11:12Z'
		)
	`); err != nil {
		t.Fatalf("seed version-24 row: %v", err)
	}
	for _, step := range []struct {
		name string
		sql  []byte
	}{
		{name: "up", sql: up},
		{name: "down", sql: down},
		{name: "up again", sql: up},
	} {
		if _, err := conn.Exec(ctx, string(step.sql)); err != nil {
			t.Fatalf("apply %s migration: %v", step.name, err)
		}
	}

	var source *string
	var profileSortAt time.Time
	if err := conn.QueryRow(ctx, `
		SELECT external_import_source, profile_sort_at
		FROM craftsky_posts
		WHERE rkey = 'existing'
	`).Scan(&source, &profileSortAt); err != nil {
		t.Fatalf("read migrated row: %v", err)
	}
	if source != nil {
		t.Fatalf("existing source = %v, want NULL", source)
	}
	wantIndexedAt := time.Date(2026, 7, 23, 10, 11, 12, 0, time.UTC)
	if !profileSortAt.Equal(wantIndexedAt) {
		t.Fatalf("existing profile_sort_at = %s, want indexed_at %s", profileSortAt, wantIndexedAt)
	}

	if _, err := conn.Exec(ctx, `
		INSERT INTO craftsky_posts (
			uri, did, rkey, cid, text, record, created_at, external_import_source
		) VALUES (
			'at://did:plc:member/social.craftsky.feed.post/imported',
			'did:plc:member',
			'imported',
			'bafyimported',
			'imported',
			'{}',
			'2018-01-02T03:04:05Z',
			'instagram'
		)
	`); err != nil {
		t.Fatalf("insert recognized source: %v", err)
	}
	if _, err := conn.Exec(ctx, `
		INSERT INTO craftsky_posts (
			uri, did, rkey, cid, text, record, created_at, external_import_source
		) VALUES (
			'at://did:plc:member/social.craftsky.feed.post/unknown',
			'did:plc:member',
			'unknown',
			'bafyunknown',
			'unknown',
			'{}',
			now(),
			'future-source'
		)
	`); err == nil {
		t.Fatal("unknown persisted source unexpectedly accepted")
	}

	for _, index := range []string{
		"craftsky_posts_profile_posts_sort_idx",
		"craftsky_posts_profile_projects_sort_idx",
		"craftsky_posts_profile_comments_sort_idx",
	} {
		var exists bool
		if err := conn.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = current_schema() AND indexname = $1
			)
		`, index).Scan(&exists); err != nil {
			t.Fatalf("inspect index %s: %v", index, err)
		}
		if !exists {
			t.Errorf("expected profile index %s", index)
		}
	}
}
