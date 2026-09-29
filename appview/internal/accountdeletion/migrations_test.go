package accountdeletion

import (
	"bytes"
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func applyAllAccountDeletionTestMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	migrations, err := testdb.UpMigrationNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range migrations {
		sql, err := testdb.ReadMigration(name)
		if err != nil {
			t.Fatal(err)
		}
		sql = adaptAccountDeletionMigrationForPostgreSQL14(name, sql)
		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
}

func adaptAccountDeletionMigrationForPostgreSQL14(name string, sql []byte) []byte {
	switch name {
	case "000019_search_foundation.up.sql":
		sql = bytes.ReplaceAll(sql, []byte("gin_trgm_ops"), []byte("public.gin_trgm_ops"))
	case "000024_saved_posts.up.sql":
		// PostgreSQL 14 (used by some local test environments) cannot parse
		// the PostgreSQL 15 subset-column SET NULL syntax. Account-deletion
		// tests do not exercise this unrelated saved-post foreign key.
		sql = bytes.ReplaceAll(sql, []byte("ON DELETE SET NULL (folder_id)"), []byte("ON DELETE NO ACTION"))
	case "000041_account_deletion_safety_tombstones.up.sql":
		sql = bytes.ReplaceAll(sql, []byte("UNIQUE NULLS NOT DISTINCT ("), []byte("UNIQUE ("))
		sql = append(sql, []byte(`
				CREATE UNIQUE INDEX account_deletion_safety_tombstones_null_upload_test_idx
				ON account_deletion_safety_tombstones(operation_id,kind,exact_key)
				WHERE upload_generation IS NULL;
			`)...)
	}
	return sql
}
