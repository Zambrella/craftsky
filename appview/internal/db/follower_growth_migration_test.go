package db_test

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const followerGrowthMigrationPreStateDDL = `
CREATE TABLE craftsky_profiles (
    did        TEXT        NOT NULL PRIMARY KEY,
    record_cid TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE atproto_follows (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_did TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey),
    UNIQUE (did, subject_did)
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:alice', 'alice-cid');
`

func TestFollowerGrowthMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000060_follower_growth_snapshots.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000060_follower_growth_snapshots.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, followerGrowthMigrationPreStateDDL)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertFollowerGrowthSchema(t, pool)

	snapshotDate := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
	capturedAt := time.Date(2026, time.August, 25, 0, 0, 2, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO follower_growth_snapshots (
			profile_did, snapshot_date, follower_count, captured_at
		) VALUES ($1, $2, $3, $4)
	`, "did:plc:alice", snapshotDate, int64(12), capturedAt); err != nil {
		t.Fatalf("insert valid snapshot: %v", err)
	}

	var (
		gotDID        string
		gotDate       time.Time
		gotCount      int64
		gotCapturedAt time.Time
	)
	if err := pool.QueryRow(ctx, `
		SELECT profile_did, snapshot_date, follower_count, captured_at
		FROM follower_growth_snapshots
	`).Scan(&gotDID, &gotDate, &gotCount, &gotCapturedAt); err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if gotDID != "did:plc:alice" || !gotDate.Equal(snapshotDate) || gotCount != 12 || !gotCapturedAt.Equal(capturedAt) {
		t.Fatalf("snapshot = (%q, %s, %d, %s), want (%q, %s, %d, %s)",
			gotDID, gotDate, gotCount, gotCapturedAt,
			"did:plc:alice", snapshotDate, 12, capturedAt,
		)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO follower_growth_snapshots (
			profile_did, snapshot_date, follower_count, captured_at
		) VALUES ('did:plc:negative', $1, -1, $2)
	`, snapshotDate, capturedAt); err == nil {
		t.Fatal("negative follower count succeeded")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO follower_growth_snapshots (
			profile_did, snapshot_date, follower_count, captured_at
		) VALUES ('did:plc:alice', $1, 13, $2)
	`, snapshotDate, capturedAt); err == nil {
		t.Fatal("duplicate profile/date snapshot succeeded")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO follower_growth_snapshots (
			profile_did, snapshot_date, follower_count, captured_at
		) VALUES ('did:plc:alice', $1, 13, $2)
	`, snapshotDate.AddDate(0, 0, 1), capturedAt.Add(24*time.Hour)); err != nil {
		t.Fatalf("insert next-date snapshot: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO follower_growth_snapshot_runs (
			snapshot_date, completed_at, captured_profile_count
		) VALUES ($1, $2, 0)
	`, snapshotDate, capturedAt); err != nil {
		t.Fatalf("insert valid empty run: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO follower_growth_snapshot_runs (
			snapshot_date, completed_at, captured_profile_count
		) VALUES ($1, $2, -1)
	`, snapshotDate.AddDate(0, 0, 1), capturedAt); err == nil {
		t.Fatal("negative captured profile count succeeded")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO follower_growth_snapshot_runs (
			snapshot_date, completed_at, captured_profile_count
		) VALUES ($1, $2, 1)
	`, snapshotDate, capturedAt); err == nil {
		t.Fatal("duplicate snapshot run date succeeded")
	}

	apply("down", down)
	for _, relation := range []string{
		"craftsky_profile_follower_counts",
		"follower_growth_snapshots",
		"follower_growth_snapshot_runs",
	} {
		if relationExists(t, pool, relation) {
			t.Fatalf("relation %s remained after down migration", relation)
		}
	}
	var profileCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_profiles`).Scan(&profileCount); err != nil {
		t.Fatalf("count unrelated profiles after down migration: %v", err)
	}
	if profileCount != 1 {
		t.Fatalf("profile count after down migration = %d, want 1", profileCount)
	}

	apply("second up", up)
	assertFollowerGrowthSchema(t, pool)
}

func assertFollowerGrowthSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, relation := range []string{
		"craftsky_profile_follower_counts",
		"follower_growth_snapshots",
		"follower_growth_snapshot_runs",
	} {
		if !relationExists(t, pool, relation) {
			t.Errorf("relation %s missing", relation)
		}
	}
	for _, constraint := range []string{
		"follower_growth_snapshots_pkey",
		"follower_growth_snapshots_follower_count_check",
		"follower_growth_snapshot_runs_pkey",
		"follower_growth_snapshot_runs_captured_profile_count_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}

	var keyColumns []string
	if err := pool.QueryRow(context.Background(), `
		SELECT array_agg(a.attname ORDER BY key.ordinality)
		FROM pg_class table_class
		JOIN pg_namespace namespace ON namespace.oid = table_class.relnamespace
		JOIN pg_index i ON i.indrelid = table_class.oid AND i.indisprimary
		JOIN unnest(i.indkey) WITH ORDINALITY AS key(attnum, ordinality) ON true
		JOIN pg_attribute a ON a.attrelid = table_class.oid AND a.attnum = key.attnum
		WHERE namespace.nspname = current_schema()
		  AND table_class.relname = 'follower_growth_snapshots'
	`).Scan(&keyColumns); err != nil {
		t.Fatalf("read snapshot primary key columns: %v", err)
	}
	wantKeyColumns := []string{"profile_did", "snapshot_date"}
	if !reflect.DeepEqual(keyColumns, wantKeyColumns) {
		t.Fatalf("snapshot primary key columns = %v, want %v", keyColumns, wantKeyColumns)
	}
}

func relationExists(t *testing.T, pool *pgxpool.Pool, relation string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `
		SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL
	`, relation).Scan(&exists); err != nil {
		t.Fatalf("check relation %s: %v", relation, err)
	}
	return exists
}
