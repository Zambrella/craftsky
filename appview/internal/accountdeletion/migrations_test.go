package accountdeletion

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func applyAllAccountDeletionTestMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			migrations = append(migrations, filepath.Join("../../migrations", entry.Name()))
		}
	}
	sort.Strings(migrations)
	for _, path := range migrations {
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(path) == "000019_search_foundation.up.sql" {
			sql = bytes.ReplaceAll(sql, []byte("gin_trgm_ops"), []byte("public.gin_trgm_ops"))
		}
		// PostgreSQL 14 (used by some local test environments) cannot parse
		// the PostgreSQL 15 subset-column SET NULL syntax. Account-deletion
		// tests do not exercise this unrelated saved-post foreign key.
		if filepath.Base(path) == "000024_saved_posts.up.sql" {
			sql = bytes.ReplaceAll(sql, []byte("ON DELETE SET NULL (folder_id)"), []byte("ON DELETE NO ACTION"))
		}
		if filepath.Base(path) == "000041_account_deletion_safety_tombstones.up.sql" {
			sql = bytes.ReplaceAll(sql, []byte("UNIQUE NULLS NOT DISTINCT ("), []byte("UNIQUE ("))
			sql = append(sql, []byte(`
				CREATE UNIQUE INDEX account_deletion_safety_tombstones_null_upload_test_idx
				ON account_deletion_safety_tombstones(operation_id,kind,exact_key)
				WHERE upload_generation IS NULL;
			`)...)
		}
		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply %s: %v", filepath.Base(path), err)
		}
	}
}
