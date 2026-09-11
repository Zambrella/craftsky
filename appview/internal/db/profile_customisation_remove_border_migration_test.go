package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestProfileCustomisationRemoveBorderMigrationUpDownUp(t *testing.T) {
	create, err := os.ReadFile("../../migrations/000036_profile_customisation.up.sql")
	if err != nil {
		t.Fatalf("read profile customisation migration: %v", err)
	}
	up, err := os.ReadFile("../../migrations/000070_profile_customisation_remove_border.up.sql")
	if err != nil {
		t.Fatalf("read remove-border up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000070_profile_customisation_remove_border.down.sql")
	if err != nil {
		t.Fatalf("read remove-border down migration: %v", err)
	}

	pool := testdb.WithSchema(t, profileCustomisationMigrationPreStateDDL+string(create))
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_customisations (
			owner_did, colour, profile_border, profile_background
		) VALUES
			('did:plc:alice', 'orchid', 'thin', 'skewdark'),
			('did:plc:bob', 'teal', 'thick', 'x2')
	`); err != nil {
		t.Fatalf("seed profile customisations: %v", err)
	}

	apply := func(label string, migration []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	if columnExists(t, pool, "profile_customisations", "profile_border") {
		t.Fatal("profile_customisations.profile_border remained after up migration")
	}
	assertProfileCustomisationValuesPreserved(t, pool)

	apply("down", down)
	if !columnExists(t, pool, "profile_customisations", "profile_border") {
		t.Fatal("profile_customisations.profile_border missing after down migration")
	}
	rows, err := pool.Query(ctx, `SELECT profile_border FROM profile_customisations ORDER BY owner_did`)
	if err != nil {
		t.Fatalf("read restored borders: %v", err)
	}
	count := 0
	for rows.Next() {
		var border string
		if err := rows.Scan(&border); err != nil {
			t.Fatalf("scan restored border: %v", err)
		}
		if border != "medium" {
			t.Fatalf("restored border = %q, want medium", border)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read restored border rows: %v", err)
	}
	rows.Close()
	if count != 2 {
		t.Fatalf("restored border row count = %d, want 2", count)
	}
	var defaultValue *string
	if err := pool.QueryRow(ctx, `
		SELECT column_default
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'profile_customisations'
		  AND column_name = 'profile_border'
	`).Scan(&defaultValue); err != nil {
		t.Fatalf("read restored border default: %v", err)
	}
	if defaultValue != nil {
		t.Fatalf("restored border default = %q, want none", *defaultValue)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:charlie', 'charlie-cid')
	`); err != nil {
		t.Fatalf("seed owner for no-default check: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_customisations (owner_did, colour, profile_background)
		VALUES ('did:plc:charlie', 'cobalt', 'none')
	`); err == nil {
		t.Fatal("insert without restored profile_border succeeded")
	}

	apply("second up", up)
	if columnExists(t, pool, "profile_customisations", "profile_border") {
		t.Fatal("profile_customisations.profile_border remained after second up migration")
	}
	assertProfileCustomisationValuesPreserved(t, pool)
}

func assertProfileCustomisationValuesPreserved(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var colour, background string
	if err := pool.QueryRow(context.Background(), `
		SELECT colour, profile_background
		FROM profile_customisations
		WHERE owner_did = 'did:plc:alice'
	`).Scan(&colour, &background); err != nil {
		t.Fatalf("read preserved customisation: %v", err)
	}
	if colour != "orchid" || background != "skewdark" {
		t.Fatalf("preserved customisation = %q/%q, want orchid/skewdark", colour, background)
	}
}
