// Package testdb holds shared test helpers for Postgres-backed tests.
//
// The helpers here are intended for test use only (they call t.Skip /
// t.Fatal), but live in a normal, importable package because cross-package
// fixtures need a stable import path.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	setupTimeout   = 30 * time.Second
	cleanupTimeout = 10 * time.Second
	scopedPoolMax  = int32(4)
)

var extensionInitializations sync.Map

type extensionInitialization struct {
	once sync.Once
	err  error
}

// WithSchema creates a focused, isolated Postgres schema for one test, runs
// ddl inside it, and returns a deliberately small pool scoped to that schema.
// Focused fixtures receive all-active owner lifecycle predicates when their DDL
// does not define the production predicates.
//
// TEST_DATABASE_URL is preferred. DATABASE_URL is ignored unless
// TEST_DATABASE_ALLOW_DATABASE_URL=true, and every selected URL must name a
// database that looks explicitly test- or development-only. If no URL is
// selected, the test skips unless TEST_DATABASE_REQUIRED=true.
func WithSchema(t *testing.T, ddl string) *pgxpool.Pool {
	t.Helper()
	pool := withIsolatedSchema(t, false)
	ctx, cancel := context.WithTimeout(t.Context(), setupTimeout)
	defer cancel()

	if ddl != "" {
		if _, err := pool.Exec(ctx, ddl); err != nil {
			t.Fatalf("create focused test schema: %v", err)
		}
	}
	installFocusedLifecyclePredicates(t, ctx, pool)
	return pool
}

// WithMigratedSchema creates an isolated schema and applies every production
// up migration without rewriting it. It deliberately requires PostgreSQL 16 so
// callers can use it as production-schema compatibility evidence.
func WithMigratedSchema(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := withIsolatedSchema(t, true)
	ctx, cancel := context.WithTimeout(t.Context(), setupTimeout)
	defer cancel()

	if err := requirePostgreSQL16(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := ApplyAllUpMigrations(ctx, pool); err != nil {
		t.Fatalf("apply exact production migrations: %v", err)
	}
	return pool
}

func withIsolatedSchema(t *testing.T, includePublic bool) *pgxpool.Pool {
	t.Helper()
	url, skip, err := resolveDatabaseURL()
	if err != nil {
		t.Fatal(err)
	}
	if skip {
		t.Skip("TEST_DATABASE_URL is unset; skipping real-pg test")
	}

	ctx, cancel := context.WithTimeout(t.Context(), setupTimeout)
	defer cancel()

	bootstrapConfig, err := testPoolConfig(url, 1)
	if err != nil {
		t.Fatalf("parse bootstrap database config: %v", err)
	}
	bootstrap, err := pgxpool.NewWithConfig(ctx, bootstrapConfig)
	if err != nil {
		t.Fatalf("create bootstrap pool: %v", err)
	}

	if err := ensurePGTrgmOnce(ctx, bootstrapConfig, bootstrap); err != nil {
		closePool(t, bootstrap, "bootstrap pool after extension failure")
		t.Fatalf("bootstrap pg_trgm: %v", err)
	}

	schema, err := newSchemaName()
	if err != nil {
		closePool(t, bootstrap, "bootstrap pool after schema-name failure")
		t.Fatalf("generate test schema name: %v", err)
	}
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := bootstrap.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		closePool(t, bootstrap, "bootstrap pool after schema-create failure")
		t.Fatalf("create schema %s: %v", schema, err)
	}

	scopedConfig, err := testPoolConfig(url, scopedPoolMax)
	if err != nil {
		dropSchema(t, bootstrap, quotedSchema)
		closePool(t, bootstrap, "bootstrap pool after scoped-config failure")
		t.Fatalf("parse scoped database config: %v", err)
	}
	scopedConfig.ConnConfig.RuntimeParams["search_path"] = schema
	if includePublic {
		scopedConfig.ConnConfig.RuntimeParams["search_path"] += ",public"
	}
	pool, err := pgxpool.NewWithConfig(ctx, scopedConfig)
	if err != nil {
		dropSchema(t, bootstrap, quotedSchema)
		closePool(t, bootstrap, "bootstrap pool after scoped-pool failure")
		t.Fatalf("create scoped pool: %v", err)
	}

	t.Cleanup(func() {
		closePool(t, pool, "scoped pool")
		dropSchema(t, bootstrap, quotedSchema)
		closePool(t, bootstrap, "bootstrap pool")
	})
	return pool
}

func installFocusedLifecyclePredicates(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	predicates := []struct {
		name string
		body string
	}{
		{"appview_owner_is_terminal", "false"},
		{"appview_owner_is_active", "true"},
	}
	for _, predicate := range predicates {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regprocedure($1) IS NOT NULL`, predicate.name+"(text)").Scan(&exists); err != nil {
			t.Fatalf("inspect %s focused fixture: %v", predicate.name, err)
		}
		if exists {
			continue
		}
		statement := fmt.Sprintf(`
			CREATE FUNCTION %s(candidate_did TEXT)
			RETURNS BOOLEAN
			LANGUAGE SQL
			IMMUTABLE
			PARALLEL SAFE
			AS $$ SELECT %s $$
		`, pgx.Identifier{predicate.name}.Sanitize(), predicate.body)
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("create %s focused fixture: %v", predicate.name, err)
		}
	}
}

func testPoolConfig(url string, maxConns int32) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = maxConns
	cfg.MinConns = 0
	return cfg, nil
}

func newSchemaName() (string, error) {
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return fmt.Sprintf("test_%d_%s", os.Getpid(), hex.EncodeToString(random)), nil
}

func closePool(t *testing.T, pool *pgxpool.Pool, description string) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		pool.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(cleanupTimeout):
		t.Errorf("cleanup %s exceeded %s; a connection may still be acquired", description, cleanupTimeout)
	}
}

func dropSchema(t *testing.T, pool *pgxpool.Pool, quotedSchema string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	if _, err := pool.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
		t.Errorf("cleanup schema %s: %v", quotedSchema, err)
	}
}

// ensurePGTrgmOnce avoids repeating database-global extension setup for every
// schema in this process. The advisory transaction lock keeps first-use setup
// safe when multiple test packages or go test processes share the database.
func ensurePGTrgmOnce(ctx context.Context, cfg *pgxpool.Config, pool *pgxpool.Pool) error {
	key := strings.Join([]string{
		cfg.ConnConfig.Host,
		strconv.FormatUint(uint64(cfg.ConnConfig.Port), 10),
		cfg.ConnConfig.Database,
	}, "\x00")
	value, _ := extensionInitializations.LoadOrStore(key, &extensionInitialization{})
	initialization := value.(*extensionInitialization)
	initialization.once.Do(func() {
		initialization.err = ensurePGTrgm(ctx, pool)
	})
	return initialization.err
}

func ensurePGTrgm(ctx context.Context, pool *pgxpool.Pool) error {
	const lockKey int64 = 0x435346505447524d // "CSFPTGRM"
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm')`).Scan(&exists); err != nil {
		return fmt.Errorf("inspect pg_trgm extension: %w", err)
	}
	if exists {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin extension bootstrap: %w", err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()

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

func requirePostgreSQL16(ctx context.Context, pool *pgxpool.Pool) error {
	var rawVersion string
	if err := pool.QueryRow(ctx, `SHOW server_version_num`).Scan(&rawVersion); err != nil {
		return fmt.Errorf("read PostgreSQL server version: %w", err)
	}
	version, err := strconv.Atoi(rawVersion)
	if err != nil {
		return fmt.Errorf("parse PostgreSQL server_version_num %q: %w", rawVersion, err)
	}
	if version/10000 != 16 {
		return fmt.Errorf("exact migration fixtures require PostgreSQL 16, got server_version_num=%d", version)
	}
	return nil
}

func resolveDatabaseURL() (url string, skip bool, err error) {
	required, err := booleanEnvironment("TEST_DATABASE_REQUIRED")
	if err != nil {
		return "", false, err
	}
	allowFallback, err := booleanEnvironment("TEST_DATABASE_ALLOW_DATABASE_URL")
	if err != nil {
		return "", false, err
	}

	url = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if url == "" && allowFallback {
		url = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if url == "" {
		if required {
			return "", false, fmt.Errorf("TEST_DATABASE_REQUIRED=true but TEST_DATABASE_URL is unset and no opted-in DATABASE_URL fallback is available")
		}
		return "", true, nil
	}
	if err := validateTestDatabaseURL(url); err != nil {
		return "", false, err
	}
	return url, false, nil
}

func booleanEnvironment(name string) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s: parse boolean: %w", name, err)
	}
	return value, nil
}

func validateTestDatabaseURL(url string) error {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return fmt.Errorf("parse test database URL: %w", err)
	}
	database := strings.ToLower(strings.TrimSpace(cfg.ConnConfig.Database))
	parts := strings.FieldsFunc(database, func(r rune) bool { return r == '_' || r == '-' })
	for _, part := range parts {
		if part == "test" || part == "testing" || part == "dev" {
			return nil
		}
	}
	return fmt.Errorf("refusing database %q: test database name must contain test/testing or an explicit dev marker", cfg.ConnConfig.Database)
}

func migrationsDirectory() string {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("testdb: locate source file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "migrations"))
}
