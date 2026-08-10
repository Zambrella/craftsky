package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const profileCustomisationMigrationPreStateDDL = `
CREATE TABLE craftsky_profiles (
    did        TEXT NOT NULL PRIMARY KEY,
    record_cid TEXT NOT NULL
);
CREATE TABLE migration_sentinel (
    id    INTEGER NOT NULL PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
INSERT INTO migration_sentinel (id, value) VALUES (1, 'preserve-me');
`

func TestProfileCustomisationMigrationUpDownUp(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000036_profile_customisation.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000036_profile_customisation.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, profileCustomisationMigrationPreStateDDL)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertProfileCustomisationSchema(t, pool)
	assertProfileCustomisationMigrationSentinel(t, pool)

	var createdAt, updatedAt time.Time
	if err := pool.QueryRow(ctx, `
		INSERT INTO profile_customisations (
			owner_did, colour, profile_border, profile_background
		) VALUES (
			'did:plc:alice', 'future-colour', 'future-border', 'future-background'
		)
		RETURNING created_at, updated_at
	`).Scan(&createdAt, &updatedAt); err != nil {
		t.Fatalf("insert customisation with retired keys: %v", err)
	}
	if createdAt.IsZero() || updatedAt.IsZero() {
		t.Fatalf("timestamps = (%s, %s), want populated", createdAt, updatedAt)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_customisations (
			owner_did, colour, profile_border, profile_background
		) VALUES ('did:plc:alice', 'cobalt', 'medium', 'none')
	`); err == nil {
		t.Fatal("duplicate owner customisation succeeded")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_customisations (
			owner_did, colour, profile_border, profile_background
		) VALUES ('did:plc:missing', 'cobalt', 'medium', 'none')
	`); err == nil {
		t.Fatal("customisation for missing membership succeeded")
	}

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = 'did:plc:alice'`); err != nil {
		t.Fatalf("delete membership: %v", err)
	}
	assertProfileCustomisationCount(t, pool, "did:plc:alice", 0)
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-rejoined-cid')
	`); err != nil {
		t.Fatalf("rejoin member: %v", err)
	}
	assertProfileCustomisationCount(t, pool, "did:plc:alice", 0)

	apply("down", down)
	if tableExists(t, pool, "profile_customisations") {
		t.Fatal("profile_customisations remained after down migration")
	}
	assertProfileCustomisationMigrationSentinel(t, pool)

	apply("second up", up)
	assertProfileCustomisationSchema(t, pool)
	assertProfileCustomisationMigrationSentinel(t, pool)
}

func assertProfileCustomisationSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if !tableExists(t, pool, "profile_customisations") {
		t.Fatal("profile_customisations table missing")
	}
	for _, constraint := range []string{
		"profile_customisations_pkey",
		"profile_customisations_owner_did_fkey",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
	for _, column := range []string{
		"owner_did",
		"colour",
		"profile_border",
		"profile_background",
		"created_at",
		"updated_at",
	} {
		if !columnExists(t, pool, "profile_customisations", column) {
			t.Errorf("column profile_customisations.%s missing", column)
		}
	}
}

func assertProfileCustomisationMigrationSentinel(t *testing.T, pool *pgxpool.Pool) {
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

func assertProfileCustomisationCount(t *testing.T, pool *pgxpool.Pool, ownerDID string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM profile_customisations WHERE owner_did = $1
	`, ownerDID).Scan(&got); err != nil {
		t.Fatalf("count profile customisations: %v", err)
	}
	if got != want {
		t.Fatalf("customisation count for %s = %d, want %d", ownerDID, got, want)
	}
}
