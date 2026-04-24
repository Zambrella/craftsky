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
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WithSchema creates an isolated Postgres schema for one test, runs the
// given ddl inside it, and returns a pool whose default search_path is
// scoped to that schema. The schema is dropped and both pools are closed
// via t.Cleanup.
//
// If TEST_DATABASE_URL and DATABASE_URL are both unset the test is
// skipped. An empty ddl argument is allowed and runs no statements; the
// caller can issue CREATE TABLE manually against the returned pool.
func WithSchema(t *testing.T, ddl string) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("TEST_DATABASE_URL and DATABASE_URL both unset; skipping real-pg test")
	}

	ctx := context.Background()
	bootstrap, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("bootstrap pool: %v", err)
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
	return pool
}
