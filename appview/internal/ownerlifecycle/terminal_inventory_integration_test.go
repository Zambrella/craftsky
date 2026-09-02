package ownerlifecycle

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestTerminalDIDInventoryIncludesBusinessState(t *testing.T) {
	want := map[string]TerminalDIDAction{
		"craftsky_account_types.owner_did":              TerminalDeleteRow,
		"craftsky_business_profiles.owner_did":          TerminalDeleteRow,
		"craftsky_business_events.owner_did":            TerminalDeleteRow,
		"craftsky_business_record_tombstones.owner_did": TerminalDeleteRow,
	}
	for _, entry := range TerminalDIDInventory() {
		key := entry.Table + "." + entry.Column
		if action, exists := want[key]; exists {
			if entry.Action != action {
				t.Fatalf("%s action = %q, want %q", key, entry.Action, action)
			}
			delete(want, key)
		}
	}
	if len(want) != 0 {
		t.Fatalf("terminal DID inventory is missing business state: %v", want)
	}
}

// TestTerminalDIDInventoryCoversMigratedSchema is the fail-closed contract for
// terminal visibility and purge work. A migration cannot add a persisted DID
// role without declaring whether the row is purged, anonymized, or retained as
// part of the small security/cleanup ledger.
func TestTerminalDIDInventoryCoversMigratedSchema(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllTerminalInventoryMigrations(t, pool)

	rows, err := pool.Query(context.Background(), `
		SELECT columns.table_name,columns.column_name
		FROM information_schema.columns columns
		JOIN information_schema.tables tables
		  ON tables.table_schema=columns.table_schema
		 AND tables.table_name=columns.table_name
		WHERE columns.table_schema=current_schema()
		  AND tables.table_type='BASE TABLE'
		  AND columns.data_type='text'
		  AND (columns.column_name='did' OR columns.column_name LIKE '%\_did' ESCAPE '\')
		ORDER BY columns.table_name,columns.column_name
	`)
	if err != nil {
		t.Fatalf("query persisted DID roles: %v", err)
	}
	defer rows.Close()

	migrated := make(map[string]struct{})
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			t.Fatalf("scan persisted DID role: %v", err)
		}
		migrated[table+"."+column] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate persisted DID roles: %v", err)
	}
	primaryKeys := migratedPrimaryKeys(t, pool)

	declared := make(map[string]struct{})
	for _, entry := range TerminalDIDInventory() {
		key := entry.Table + "." + entry.Column
		if _, duplicate := declared[key]; duplicate {
			t.Errorf("duplicate terminal DID inventory entry %s", key)
		}
		declared[key] = struct{}{}
		if _, exists := migrated[key]; !exists {
			t.Errorf("terminal DID inventory declares missing migrated column %s", key)
		}
	}

	var missing []string
	for key := range migrated {
		if _, ok := declared[key]; !ok {
			table, _, _ := strings.Cut(key, ".")
			missing = append(missing, key+" key="+strings.Join(primaryKeys[table], "+"))
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("persisted DID roles missing from terminal inventory: %s", strings.Join(missing, ", "))
	}
}

func TestTerminalDIDInventoryHasRoleLeadingKeysetIndexes(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllTerminalInventoryMigrations(t, pool)

	type indexColumns struct {
		name    string
		columns []string
	}
	indexesByTable := make(map[string][]indexColumns)
	rows, err := pool.Query(context.Background(), `
		SELECT table_rel.relname,index_rel.relname,
		       array_agg(attribute.attname ORDER BY key.ordinality)
		FROM pg_index index_def
		JOIN pg_class table_rel ON table_rel.oid=index_def.indrelid
		JOIN pg_namespace namespace ON namespace.oid=table_rel.relnamespace
		JOIN pg_class index_rel ON index_rel.oid=index_def.indexrelid
		CROSS JOIN LATERAL unnest(index_def.indkey)
		     WITH ORDINALITY AS key(attnum,ordinality)
		JOIN pg_attribute attribute
		  ON attribute.attrelid=table_rel.oid AND attribute.attnum=key.attnum
		WHERE namespace.nspname=current_schema()
		  AND index_def.indisvalid
		  AND key.ordinality <= index_def.indnkeyatts
		GROUP BY table_rel.relname,index_rel.relname
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var table, name string
		var columns []string
		if err := rows.Scan(&table, &name, &columns); err != nil {
			t.Fatal(err)
		}
		indexesByTable[table] = append(indexesByTable[table], indexColumns{name: name, columns: columns})
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	var missing []string
	for _, entry := range TerminalDIDInventory() {
		required := []string{entry.Column}
		for _, key := range entry.KeyColumns {
			if !slices.Contains(required, key) {
				required = append(required, key)
			}
		}
		covered := false
		for _, index := range indexesByTable[entry.Table] {
			if len(index.columns) < len(required) {
				continue
			}
			if slices.Equal(index.columns[:len(required)], required) {
				covered = true
				break
			}
		}
		if !covered {
			missing = append(missing, fmt.Sprintf(
				"%s.%s (%s/%s) needs index prefix (%s)",
				entry.Table, entry.Column, entry.Component, entry.Role,
				strings.Join(required, ","),
			))
		}
	}
	if len(missing) != 0 {
		slices.Sort(missing)
		t.Fatalf("terminal purge keyset indexes missing:\n%s", strings.Join(missing, "\n"))
	}
}

func TestTerminalDIDInventoryUsesSafeUniqueBatchKeys(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllTerminalInventoryMigrations(t, pool)

	type uniqueKey struct {
		name    string
		columns []string
	}
	uniqueByTable := make(map[string][]uniqueKey)
	rows, err := pool.Query(context.Background(), `
		SELECT table_rel.relname,index_rel.relname,
		       array_agg(attribute.attname ORDER BY key.ordinality)
		FROM pg_index index_def
		JOIN pg_class table_rel ON table_rel.oid=index_def.indrelid
		JOIN pg_namespace namespace ON namespace.oid=table_rel.relnamespace
		JOIN pg_class index_rel ON index_rel.oid=index_def.indexrelid
		CROSS JOIN LATERAL unnest(index_def.indkey)
		     WITH ORDINALITY AS key(attnum,ordinality)
		JOIN pg_attribute attribute
		  ON attribute.attrelid=table_rel.oid AND attribute.attnum=key.attnum
		WHERE namespace.nspname=current_schema()
		  AND index_def.indisvalid
		  AND index_def.indisunique
		  AND index_def.indpred IS NULL
		  AND index_def.indexprs IS NULL
		  AND key.ordinality <= index_def.indnkeyatts
		GROUP BY table_rel.relname,index_rel.relname
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var table, name string
		var columns []string
		if err := rows.Scan(&table, &name, &columns); err != nil {
			t.Fatal(err)
		}
		uniqueByTable[table] = append(uniqueByTable[table], uniqueKey{name: name, columns: columns})
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	identifier := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	var unsafe []string
	for _, entry := range TerminalDIDInventory() {
		if !identifier.MatchString(entry.Table) || !identifier.MatchString(entry.Column) ||
			!identifier.MatchString(entry.Component) || !identifier.MatchString(entry.Role) ||
			len(entry.KeyColumns) == 0 {
			unsafe = append(unsafe, entry.Table+"."+entry.Column+" has invalid identifiers or no batch key")
			continue
		}
		for _, column := range entry.KeyColumns {
			if !identifier.MatchString(column) {
				unsafe = append(unsafe, entry.Table+"."+entry.Column+" has invalid key "+column)
			}
		}
		if entry.Action != TerminalDeleteRow && strings.TrimSpace(entry.Rationale) == "" {
			unsafe = append(unsafe, entry.Table+"."+entry.Column+" retained/anonymized without rationale")
		}
		unique := false
		for _, key := range uniqueByTable[entry.Table] {
			if slices.Equal(key.columns, entry.KeyColumns) {
				unique = true
				break
			}
		}
		if !unique {
			unsafe = append(unsafe, fmt.Sprintf(
				"%s.%s batch key (%s) is not an unconditional unique key",
				entry.Table, entry.Column, strings.Join(entry.KeyColumns, ","),
			))
		}
	}
	if len(unsafe) != 0 {
		slices.Sort(unsafe)
		t.Fatalf("unsafe terminal purge inventory:\n%s", strings.Join(unsafe, "\n"))
	}
}

func TestTerminalPurgeClassifiesEveryCascadingDIDParent(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyAllTerminalInventoryMigrations(t, pool)
	rows, err := pool.Query(context.Background(), `
		SELECT DISTINCT parent.relname
		FROM pg_constraint AS con
		JOIN pg_class AS parent ON parent.oid=con.confrelid
		JOIN pg_namespace AS namespace ON namespace.oid=parent.relnamespace
		WHERE con.contype='f'
		  AND con.confdeltype IN ('c','n')
		  AND namespace.nspname=current_schema()
		ORDER BY parent.relname
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	cascadingParents := make(map[string]struct{})
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatal(err)
		}
		cascadingParents[table] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	var missing []string
	for _, entry := range TerminalDIDInventory() {
		if entry.Action != TerminalDeleteRow {
			continue
		}
		if _, cascades := cascadingParents[entry.Table]; !cascades {
			continue
		}
		policy := terminalCascadePolicies[entry.Table]
		if policy != "drain" && policy != "dependency" && policy != "fixed" {
			missing = append(missing, entry.Table+"."+entry.Column)
		}
	}
	if len(missing) != 0 {
		slices.Sort(missing)
		t.Fatalf("unclassified cascading terminal parents: %s", strings.Join(missing, ", "))
	}
}

func TestTerminalPurgeRoleIndexesMigrationUpDownUp(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	applyTerminalInventoryMigrationsBefore(t, pool, "000046_")
	up, err := os.ReadFile("../../migrations/000046_terminal_purge_role_indexes.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000046_terminal_purge_role_indexes.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	assertMigrationIndexes := func(want int) {
		t.Helper()
		var got int
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*)::int
			FROM pg_class index_rel
			JOIN pg_namespace namespace ON namespace.oid=index_rel.relnamespace
			WHERE namespace.nspname=current_schema()
			  AND index_rel.relkind='i'
			  AND index_rel.relname LIKE 'terminal_purge_%'
		`).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("terminal purge index count=%d, want %d", got, want)
		}
	}

	if _, err := pool.Exec(context.Background(), string(up)); err != nil {
		t.Fatalf("terminal purge indexes up: %v", err)
	}
	assertMigrationIndexes(43)
	if _, err := pool.Exec(context.Background(), string(down)); err != nil {
		t.Fatalf("terminal purge indexes down: %v", err)
	}
	assertMigrationIndexes(0)
	if _, err := pool.Exec(context.Background(), string(up)); err != nil {
		t.Fatalf("terminal purge indexes second up: %v", err)
	}
	assertMigrationIndexes(43)
}

func migratedPrimaryKeys(t *testing.T, pool *pgxpool.Pool) map[string][]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT tc.table_name,kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON kcu.constraint_schema=tc.constraint_schema
		 AND kcu.constraint_name=tc.constraint_name
		WHERE tc.constraint_schema=current_schema()
		  AND tc.constraint_type='PRIMARY KEY'
		ORDER BY tc.table_name,kcu.ordinal_position
	`)
	if err != nil {
		t.Fatalf("query migrated primary keys: %v", err)
	}
	defer rows.Close()
	keys := make(map[string][]string)
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			t.Fatalf("scan migrated primary key: %v", err)
		}
		keys[table] = append(keys[table], column)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate migrated primary keys: %v", err)
	}
	return keys
}

func applyAllTerminalInventoryMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	applyTerminalInventoryMigrationsBefore(t, pool, "")
}

func applyTerminalInventoryMigrationsBefore(t *testing.T, pool *pgxpool.Pool, stopBefore string) {
	t.Helper()
	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") &&
			(stopBefore == "" || entry.Name() < stopBefore) {
			migrations = append(migrations, filepath.Join("../../migrations", entry.Name()))
		}
	}
	sort.Strings(migrations)
	for _, path := range migrations {
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		switch filepath.Base(path) {
		case "000019_search_foundation.up.sql":
			sql = bytes.ReplaceAll(sql, []byte("gin_trgm_ops"), []byte("public.gin_trgm_ops"))
		case "000024_saved_posts.up.sql":
			sql = bytes.ReplaceAll(sql, []byte("ON DELETE SET NULL (folder_id)"), []byte("ON DELETE NO ACTION"))
		case "000041_account_deletion_safety_tombstones.up.sql":
			sql = bytes.ReplaceAll(sql, []byte("UNIQUE NULLS NOT DISTINCT ("), []byte("UNIQUE ("))
			sql = append(sql, []byte(`
				CREATE UNIQUE INDEX account_deletion_safety_tombstones_null_upload_terminal_inventory_idx
					ON account_deletion_safety_tombstones(operation_id,kind,exact_key)
					WHERE upload_generation IS NULL;
			`)...)
		}
		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply %s: %v", filepath.Base(path), err)
		}
	}
}
