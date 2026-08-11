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
	"social.craftsky/appview/internal/testdb"
)

func TestPrivateDataManifestCoversMigratedOwnerSurfaces(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)

	manifest := PrivateDataManifest()
	classified := make(map[string]ManifestEntry, len(manifest))
	for _, entry := range manifest {
		if entry.Name == "" || !entry.Policy.Valid() || !entry.Kind.Valid() || entry.Rule == "" ||
			entry.OwnershipPath == "" || entry.CleanupComponent == "" || entry.VerificationQuery == "" {
			t.Fatalf("invalid manifest entry: %+v", entry)
		}
		key := string(entry.Kind) + ":" + entry.Name
		if previous, exists := classified[key]; exists {
			t.Fatalf("duplicate manifest entry %s: %+v and %+v", key, previous, entry)
		}
		classified[key] = entry
		if entry.Kind == ManifestTable {
			var count int
			if err := pool.QueryRow(context.Background(), entry.VerificationQuery).Scan(&count); err != nil {
				t.Errorf("manifest verification query for %s is not executable: %v", entry.Name, err)
			}
		} else if !strings.HasPrefix(entry.VerificationQuery, "gate:") {
			t.Errorf("service %s has no executable gate descriptor: %q", entry.Name, entry.VerificationQuery)
		}
	}

	rows, err := pool.Query(context.Background(), `
		WITH RECURSIVE owner_tables(table_name) AS (
			SELECT DISTINCT table_name::text
			FROM information_schema.columns
			WHERE table_schema=current_schema()
			  AND (column_name='did' OR column_name LIKE '%\_did' ESCAPE '\')
			UNION
			SELECT child.relname
			FROM owner_tables parent
			JOIN pg_class parent_class ON parent_class.relname=parent.table_name
			JOIN pg_namespace parent_namespace ON parent_namespace.oid=parent_class.relnamespace
			  AND parent_namespace.nspname=current_schema()
			JOIN pg_constraint fk ON fk.contype='f' AND fk.confrelid=parent_class.oid
			JOIN pg_class child ON child.oid=fk.conrelid
			JOIN pg_namespace child_namespace ON child_namespace.oid=child.relnamespace
			  AND child_namespace.nspname=current_schema()
		)
		SELECT table_name FROM owner_tables ORDER BY table_name
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var discovered []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatal(err)
		}
		discovered = append(discovered, table)
		if _, exists := classified[string(ManifestTable)+":"+table]; !exists {
			t.Errorf("migrated owner surface %s has no explicit manifest policy", table)
		}
	}
	if len(discovered) == 0 {
		t.Fatal("schema discovery returned no owner surfaces")
	}

	for _, required := range []string{
		"scheduledPostObjectStore",
		"instagramPrivateDataService",
		"craftskyPublicIndexers",
		"sharedIdentityAndBlueskyCaches",
	} {
		if _, exists := classified[string(ManifestService)+":"+required]; !exists {
			t.Errorf("required non-schema manifest entry %s missing", required)
		}
	}
}

func TestPrivateDataManifestNamesExecutableProductionComponents(t *testing.T) {
	t.Parallel()

	executable := map[string]bool{
		"databasePrivate":                true,
		"instagramPrivate":               true,
		"scheduledPosts":                 true,
		"auditSweeper":                   true,
		"terminalFinalization":           true,
		"craftskyPublicIndexers":         true,
		"sharedIdentityAndBlueskyCaches": true,
	}
	for _, entry := range PrivateDataManifest() {
		if !executable[entry.CleanupComponent] {
			t.Errorf("manifest entry %s:%s names non-executable component %q", entry.Kind, entry.Name, entry.CleanupComponent)
		}
	}
}

func applyAllAccountDeletionTestMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public`); err != nil {
		t.Fatalf("ensure pg_trgm extension: %v", err)
	}
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
		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply %s: %v", filepath.Base(path), err)
		}
	}
}
