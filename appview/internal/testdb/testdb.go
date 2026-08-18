// Package testdb holds shared test helpers for Postgres-backed tests.
//
// The helpers here are intended for test use only (they call t.Skip /
// t.Fatal), but live in a normal, importable package — Go doesn't have a
// "test fixtures" convention beyond that, and a cross-package shared
// helper needs a stable import path.
package testdb

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WithSchema creates an isolated Postgres schema for one test, runs the
// given ddl inside it, and returns a pool whose default search_path is
// scoped to that schema. The schema is dropped and both pools are closed
// via t.Cleanup.
//
// If TEST_DATABASE_URL and DATABASE_URL are both unset the test is skipped,
// unless TEST_DATABASE_REQUIRED=true. Required mode fails instead so a full
// test job cannot report success after silently skipping database coverage.
// An empty ddl argument is allowed and runs no statements; the caller can
// issue CREATE TABLE manually against the returned pool.
func WithSchema(t *testing.T, ddl string) *pgxpool.Pool {
	t.Helper()
	url, skip, err := resolveDatabaseURL()
	if err != nil {
		t.Fatal(err)
	}
	if skip {
		t.Skip("TEST_DATABASE_URL and DATABASE_URL both unset; skipping real-pg test")
	}

	ctx := context.Background()
	bootstrap, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("bootstrap pool: %v", err)
	}
	if err := ensurePGTrgm(ctx, bootstrap); err != nil {
		bootstrap.Close()
		t.Fatalf("bootstrap pg_trgm: %v", err)
	}
	schema := fmt.Sprintf("test_%d", rand.Uint32())
	if _, err := bootstrap.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = bootstrap.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		bootstrap.Close()
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("scoped pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if ddl != "" {
		if _, err := pool.Exec(ctx, ddl); err != nil {
			t.Fatalf("create test tables: %v", err)
		}
	}
	// Most focused store fixtures intentionally model pre-lifecycle schemas.
	// Give those fixtures the production terminal-predicate signature with an
	// all-active implementation. Migration 39 uses CREATE OR REPLACE and takes
	// over when a test applies the real lifecycle schema.
	var hasTerminalPredicate bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regprocedure('appview_owner_is_terminal(text)') IS NOT NULL
	`).Scan(&hasTerminalPredicate); err != nil {
		t.Fatalf("inspect terminal predicate fixture: %v", err)
	}
	if !hasTerminalPredicate {
		if _, err := pool.Exec(ctx, `
			CREATE FUNCTION appview_owner_is_terminal(candidate_did TEXT)
			RETURNS BOOLEAN
			LANGUAGE SQL
			IMMUTABLE
			PARALLEL SAFE
			AS $$ SELECT false $$
		`); err != nil {
			t.Fatalf("create terminal predicate fixture: %v", err)
		}
	}
	// Focused pre-lifecycle store fixtures likewise receive an all-active
	// membership predicate. Migration 39 replaces it with lifecycle authority
	// when the complete schema is under test.
	var hasActivePredicate bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regprocedure('appview_owner_is_active(text)') IS NOT NULL
	`).Scan(&hasActivePredicate); err != nil {
		t.Fatalf("inspect active owner predicate fixture: %v", err)
	}
	if !hasActivePredicate {
		if _, err := pool.Exec(ctx, `
			CREATE FUNCTION appview_owner_is_active(candidate_did TEXT)
			RETURNS BOOLEAN
			LANGUAGE SQL
			IMMUTABLE
			PARALLEL SAFE
			AS $$ SELECT true $$
		`); err != nil {
			t.Fatalf("create active owner predicate fixture: %v", err)
		}
	}
	return pool
}

// ensurePGTrgm serializes the database-global extension bootstrap across test
// packages and processes. PostgreSQL extensions are not scoped to the isolated
// schemas created by WithSchema.
func ensurePGTrgm(ctx context.Context, pool *pgxpool.Pool) error {
	const lockKey int64 = 0x435346505447524d // "CSFPTGRM"

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin extension bootstrap: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, lockKey); err != nil {
		return fmt.Errorf("lock extension bootstrap: %w", err)
	}
	if _, err := tx.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public`); err != nil {
		return fmt.Errorf("create pg_trgm extension: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit extension bootstrap: %w", err)
	}
	return nil
}

func resolveDatabaseURL() (url string, skip bool, err error) {
	required := false
	if raw := os.Getenv("TEST_DATABASE_REQUIRED"); raw != "" {
		required, err = strconv.ParseBool(raw)
		if err != nil {
			return "", false, fmt.Errorf("TEST_DATABASE_REQUIRED: parse boolean: %w", err)
		}
	}

	url = os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url != "" {
		return url, false, nil
	}
	if required {
		return "", false, fmt.Errorf("TEST_DATABASE_REQUIRED=true but TEST_DATABASE_URL and DATABASE_URL are unset")
	}
	return "", true, nil
}
