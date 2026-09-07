package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestPreferredPronounsMigrationUpDownUp(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000068_preferred_pronouns.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000068_preferred_pronouns.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE bluesky_profiles (
			did TEXT PRIMARY KEY,
			record_cid TEXT NOT NULL
		);
		INSERT INTO bluesky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'bafyprofile');
	`)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertNullablePronounsColumn(t, pool)
	var pronouns *string
	if err := pool.QueryRow(ctx, `SELECT pronouns FROM bluesky_profiles WHERE did = 'did:plc:alice'`).Scan(&pronouns); err != nil {
		t.Fatalf("read existing row: %v", err)
	}
	if pronouns != nil {
		t.Fatalf("existing-row pronouns = %q, want NULL", *pronouns)
	}
	if _, err := pool.Exec(ctx, `UPDATE bluesky_profiles SET pronouns = 'she/her' WHERE did = 'did:plc:alice'`); err != nil {
		t.Fatalf("store pronouns: %v", err)
	}

	apply("down", down)
	if columnExists(t, pool, "bluesky_profiles", "pronouns") {
		t.Fatal("pronouns column remained after down migration")
	}
	apply("second up", up)
	assertNullablePronounsColumn(t, pool)
}

func assertNullablePronounsColumn(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var nullable string
	if err := pool.QueryRow(context.Background(), `
		SELECT is_nullable
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'bluesky_profiles'
		  AND column_name = 'pronouns'
	`).Scan(&nullable); err != nil {
		t.Fatalf("read pronouns column: %v", err)
	}
	if nullable != "YES" {
		t.Fatalf("pronouns is_nullable = %q, want YES", nullable)
	}
}
