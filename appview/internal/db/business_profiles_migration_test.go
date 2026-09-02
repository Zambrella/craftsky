package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const businessAccountTypesMigrationPreStateDDL = `
CREATE TABLE craftsky_profiles (
    did        TEXT NOT NULL PRIMARY KEY,
    record_cid TEXT NOT NULL
);
CREATE TABLE migration_sentinel (
    id    INTEGER NOT NULL PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:alice', 'alice-cid');
INSERT INTO migration_sentinel (id, value) VALUES (1, 'preserve-me');
`

func TestBusinessAccountTypesMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000061_business_account_types.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000061_business_account_types.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, businessAccountTypesMigrationPreStateDDL)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertBusinessAccountTypesSchema(t, pool)

	for _, accountType := range []string{"regular", "business"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_account_types (owner_did, account_type)
			VALUES ($1, $2)
		`, "did:plc:"+accountType, accountType); err != nil {
			t.Errorf("insert account type %q: %v", accountType, err)
		}
	}
	for _, accountType := range []string{"", "pro", "Business"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_account_types (owner_did, account_type)
			VALUES ($1, $2)
		`, "did:plc:invalid-"+accountType, accountType); err == nil {
			t.Errorf("insert invalid account type %q succeeded", accountType)
		}
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_account_types (owner_did, account_type)
		VALUES ('did:plc:alice', 'business')
	`); err != nil {
		t.Fatalf("insert member account type: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = 'did:plc:alice'`); err != nil {
		t.Fatalf("delete membership: %v", err)
	}
	var retained string
	if err := pool.QueryRow(ctx, `
		SELECT account_type FROM craftsky_account_types WHERE owner_did = 'did:plc:alice'
	`).Scan(&retained); err != nil {
		t.Fatalf("read retained account type: %v", err)
	}
	if retained != "business" {
		t.Fatalf("retained account type = %q, want business", retained)
	}

	apply("down", down)
	if tableExists(t, pool, "craftsky_account_types") {
		t.Fatal("craftsky_account_types remained after down migration")
	}
	assertBusinessMigrationSentinel(t, pool)

	apply("second up", up)
	assertBusinessAccountTypesSchema(t, pool)
}

func assertBusinessAccountTypesSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if !tableExists(t, pool, "craftsky_account_types") {
		t.Fatal("craftsky_account_types table missing")
	}
	for _, constraint := range []string{
		"craftsky_account_types_pkey",
		"craftsky_account_types_account_type_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
	for _, column := range []string{"owner_did", "account_type"} {
		if !columnExists(t, pool, "craftsky_account_types", column) {
			t.Errorf("column craftsky_account_types.%s missing", column)
		}
	}

	var foreignKeys int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM pg_constraint c
		JOIN pg_class t ON t.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = current_schema()
		  AND t.relname = 'craftsky_account_types'
		  AND c.contype = 'f'
	`).Scan(&foreignKeys); err != nil {
		t.Fatalf("count account type foreign keys: %v", err)
	}
	if foreignKeys != 0 {
		t.Fatalf("account type foreign key count = %d, want 0", foreignKeys)
	}
}

func assertBusinessMigrationSentinel(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var value string
	if err := pool.QueryRow(context.Background(), `
		SELECT value FROM migration_sentinel WHERE id = 1
	`).Scan(&value); err != nil {
		t.Fatalf("read migration sentinel: %v", err)
	}
	if value != "preserve-me" {
		t.Fatalf("migration sentinel = %q, want preserve-me", value)
	}
}
