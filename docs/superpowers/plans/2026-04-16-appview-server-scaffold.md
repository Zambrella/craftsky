# App View Server Scaffold Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the Craftsky App View's Go server and CLI binary with dependency-injection and lifecycle patterns modelled on `stash_hub/server`, ending in a running `GET /health` + `GET /whoami` with a smoke-testable CLI.

**Architecture:** Single Go module under `appview/`. Two `main` packages (`cmd/appview`, `cmd/cli`) share a library package `internal/app` that loads config and wires `*Deps`. Stdlib `net/http` + middleware stack (logging, CORS, auth). Stub interfaces for firehose/indexer so day-one CLI subcommands compile and return "not yet implemented" cleanly.

**Tech Stack:**
- Go 1.22+ (uses stdlib method/path routing; repo has Go 1.25 installed)
- `github.com/jackc/pgx/v5` + `pgxpool` for Postgres
- `github.com/spf13/cobra` for the CLI
- `github.com/joho/godotenv` for env file loading
- `github.com/golang-migrate/migrate/v4` (with `postgres` driver and `file` source) for migrations
- `github.com/google/uuid` for request IDs
- Standard library `log/slog` for logging

**Reference spec:** [docs/superpowers/specs/2026-04-16-appview-server-scaffold-design.md](../specs/2026-04-16-appview-server-scaffold-design.md) — authoritative. If this plan and the spec disagree, the spec wins; stop and reconcile.

**Working directory for all commands:** `appview/` (i.e. `cd appview` once before starting; all `go` commands assume that cwd). Commit messages use the `feat(appview):` / `chore(appview):` prefix so they're greppable.

**Prerequisites for running the final server locally:**
- A local Postgres instance the dev `DATABASE_URL` points at. The scaffold pings it; it does not create a database. Spin one up via `docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=dev -e POSTGRES_USER=craftsky -e POSTGRES_DB=craftsky_dev postgres:16` if you need one.
- No other services required.

**Testing philosophy for this scaffold:**
- Every task that adds code has a test. Tests run first (TDD).
- Unit tests use the stdlib `testing` package. No testing framework dependency.
- Integration-style tests that need a live Postgres are marked with `//go:build integration` and are only invoked via `go test -tags=integration ./...`. The acceptance criteria at the end of each chunk describe which commands to run.
- `httptest.NewServer` is used for any handler/middleware test that benefits from a real request path.

---

## Chunk 1: Foundation — module deps, config, db, environments

**Scope:** Nothing HTTP yet. Set up the Go module's dependencies, write `internal/app/config.go`, `internal/db/db.go`, and the environment files. Each can be tested in isolation. End of chunk: `go build ./...` passes, `go test ./...` passes, but there's no binary behaviour yet.

### Task 1.1: Add module dependencies

**Files:**
- Modify: `appview/go.mod`
- Modify: `appview/go.sum` (generated)

- [ ] **Step 1: Confirm we're in the right directory**

Run: `pwd && cat go.mod`
Expected: ends in `/craftsky/appview`; `go.mod` shows `module social.craftsky/appview`. The `go` directive version is whatever the existing file sets — do not change it in this task. No `require` block yet.

- [ ] **Step 2: Add all five dependencies at once**

Run:
```bash
go get github.com/jackc/pgx/v5@latest
go get github.com/spf13/cobra@latest
go get github.com/joho/godotenv@latest
go get github.com/golang-migrate/migrate/v4@latest
go get github.com/google/uuid@latest
```

Expected: each command prints a `go: added ... v…` line. `go.mod` now has a `require` block with these five entries (plus transitive `go.sum` entries).

- [ ] **Step 3: Tidy**

Run: `go mod tidy`
Expected: no output, or it prints some `indirect` trims. `go.mod` and `go.sum` both updated.

- [ ] **Step 4: Verify build still works with just the stub main**

Run: `go build ./...`
Expected: exits 0 with no output.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit -m "feat(appview): add module dependencies for scaffold"
```

### Task 1.2: `internal/app/config.go` — failing test for ParseEnv

**Files:**
- Create: `appview/internal/app/config.go`
- Create: `appview/internal/app/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `appview/internal/app/config_test.go`:
```go
package app

import (
	"testing"
)

func TestParseEnv(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Env
		wantErr bool
	}{
		{"dev", "dev", EnvDev, false},
		{"prod", "prod", EnvProd, false},
		{"empty", "", "", true},
		{"unknown", "staging", "", true},
		{"caps", "DEV", "", true}, // case-sensitive by design
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnv(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseEnv(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/...`
Expected: compile error — `undefined: ParseEnv`, `undefined: Env`, `undefined: EnvDev`, `undefined: EnvProd`.

- [ ] **Step 3: Write the minimal `Env` and `ParseEnv`**

Create `appview/internal/app/config.go`:
```go
// Package app wires configuration and dependencies for both the server and
// CLI binaries. It is the single source of truth for how a Craftsky App View
// process is assembled.
package app

import "fmt"

// Env identifies which deployment environment the process is running in.
// It drives which .env file is loaded and which real/mock dependencies are
// wired by the Deps factories.
type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

// ParseEnv converts the string form of an environment (as supplied by the
// server's positional arg or the CLI's --env flag) into an Env.
// Case-sensitive: "DEV" returns an error.
func ParseEnv(s string) (Env, error) {
	switch s {
	case string(EnvDev):
		return EnvDev, nil
	case string(EnvProd):
		return EnvProd, nil
	default:
		return "", fmt.Errorf("unknown env %q: expected %q or %q", s, EnvDev, EnvProd)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/...`
Expected: `ok  social.craftsky/appview/internal/app  0.0Xs`

- [ ] **Step 5: Commit**

```bash
git add internal/app/config.go internal/app/config_test.go
git commit -m "feat(appview): add app.Env and ParseEnv"
```

### Task 1.3: `Config` struct and `LoadConfig` — test first

**Files:**
- Modify: `appview/internal/app/config.go`
- Modify: `appview/internal/app/config_test.go`

- [ ] **Step 1: Extend test with LoadConfig cases**

Replace `appview/internal/app/config_test.go` with:
```go
package app

import (
	"os"
	"strings"
	"testing"
)

func TestParseEnv(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Env
		wantErr bool
	}{
		{"dev", "dev", EnvDev, false},
		{"prod", "prod", EnvProd, false},
		{"empty", "", "", true},
		{"unknown", "staging", "", true},
		{"caps", "DEV", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnv(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseEnv(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// testConfigFile writes a temporary .env-style file and returns its path.
// It also clears the relevant env vars before the test so LoadConfig's
// os.Getenv fallback doesn't leak in from the host environment.
func testConfigFile(t *testing.T, contents string) string {
	t.Helper()
	for _, k := range []string{"DATABASE_URL", "ALLOWED_ORIGINS", "CRAFTSKY_DEV_DID"} {
		t.Setenv(k, "")
	}
	f, err := os.CreateTemp(t.TempDir(), "test-*.env")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	if _, err := f.WriteString(contents); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp: %v", err)
	}
	return f.Name()
}

func TestLoadConfig_DevValid(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\n")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Env != EnvDev {
		t.Errorf("Env = %q, want %q", cfg.Env, EnvDev)
	}
	if cfg.DatabaseURL != "postgres://dev" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "*" {
		t.Errorf("AllowedOrigins = %v", cfg.AllowedOrigins)
	}
	if cfg.DevDID != "did:plc:test" {
		t.Errorf("DevDID = %q", cfg.DevDID)
	}
}

func TestLoadConfig_ProdValid(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://prod\nALLOWED_ORIGINS=https://a.example,https://b.example\n")
	cfg, err := LoadConfig(EnvProd, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got := cfg.AllowedOrigins; len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Errorf("AllowedOrigins = %v", got)
	}
	// DevDID is ignored in prod; unset is fine.
	if cfg.DevDID != "" {
		t.Errorf("DevDID = %q, want empty in prod", cfg.DevDID)
	}
}

func TestLoadConfig_MissingDatabaseURL(t *testing.T) {
	path := testConfigFile(t, "ALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\n")
	_, err := LoadConfig(EnvDev, path)
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error should name DATABASE_URL, got %v", err)
	}
}

func TestLoadConfig_MissingDevDIDInDev(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://dev\nALLOWED_ORIGINS=*\n")
	_, err := LoadConfig(EnvDev, path)
	if err == nil {
		t.Fatal("expected error for missing CRAFTSKY_DEV_DID in dev")
	}
	if !strings.Contains(err.Error(), "CRAFTSKY_DEV_DID") {
		t.Errorf("error should name CRAFTSKY_DEV_DID, got %v", err)
	}
}

func TestLoadConfig_OSEnvUsedWhenFileAbsent(t *testing.T) {
	// File lacks DATABASE_URL, but os.Getenv has it.
	path := testConfigFile(t, "ALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\n")
	t.Setenv("DATABASE_URL", "postgres://fromenv")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DatabaseURL != "postgres://fromenv" {
		t.Errorf("DatabaseURL = %q, want postgres://fromenv", cfg.DatabaseURL)
	}
}

func TestLoadConfig_OSEnvWinsOnConflict(t *testing.T) {
	// File and os.Getenv both set DATABASE_URL; os.Getenv must win.
	path := testConfigFile(t, "DATABASE_URL=postgres://fromfile\nALLOWED_ORIGINS=*\nCRAFTSKY_DEV_DID=did:plc:test\n")
	t.Setenv("DATABASE_URL", "postgres://fromenv")
	cfg, err := LoadConfig(EnvDev, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DatabaseURL != "postgres://fromenv" {
		t.Errorf("DatabaseURL = %q, want postgres://fromenv (os.Getenv must win over .env file)", cfg.DatabaseURL)
	}
}

func TestLoadConfig_DevDIDIgnoredInProd(t *testing.T) {
	path := testConfigFile(t, "DATABASE_URL=postgres://p\nALLOWED_ORIGINS=https://a.example\nCRAFTSKY_DEV_DID=did:plc:leaked\n")
	cfg, err := LoadConfig(EnvProd, path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DevDID != "" {
		t.Errorf("DevDID = %q, want empty in prod (leaked from .env)", cfg.DevDID)
	}
}
```

Note: the `TestParseEnv` from Task 1.2 is included in the replacement above so the whole file is coherent — we're rewriting `config_test.go` now that it has imports and more than one test.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/...`
Expected: compile error — `undefined: Config`, `undefined: LoadConfig`.

- [ ] **Step 3: Implement `Config` + `LoadConfig`**

Replace `appview/internal/app/config.go` with:
```go
// Package app wires configuration and dependencies for both the server and
// CLI binaries. It is the single source of truth for how a Craftsky App View
// process is assembled.
package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Env identifies which deployment environment the process is running in.
type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

// ParseEnv converts the string form of an environment into an Env.
// Case-sensitive: "DEV" returns an error.
func ParseEnv(s string) (Env, error) {
	switch s {
	case string(EnvDev):
		return EnvDev, nil
	case string(EnvProd):
		return EnvProd, nil
	default:
		return "", fmt.Errorf("unknown env %q: expected %q or %q", s, EnvDev, EnvProd)
	}
}

// Config is the validated, fully-resolved configuration for one process.
//
// It is produced by LoadConfig, which merges an .env file with os.Getenv
// (with os.Getenv winning on conflicts). Required fields cause LoadConfig
// to fail loudly; the Config that reaches Deps is always complete.
type Config struct {
	Env            Env
	DatabaseURL    string
	AllowedOrigins []string
	DevDID         string // populated in dev only; empty in prod
}

// LoadConfig reads environments/<env>.env from envFilePath, layers os.Getenv
// on top (so shell env vars override the file), and validates that every
// required field is set. Missing required fields produce an error naming
// the specific key.
//
// envFilePath is passed explicitly (not derived from env) so tests can
// point at a temp file. Callers in main code will pass
// "environments/<env>.env".
func LoadConfig(env Env, envFilePath string) (Config, error) {
	// godotenv.Load merges the file into os.Environ without overwriting
	// existing values — exactly the "os.Getenv wins" semantics we want.
	// A missing file is not fatal: os.Getenv alone may have everything.
	_ = godotenv.Load(envFilePath)

	cfg := Config{
		Env:         env,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		DevDID:      os.Getenv("CRAFTSKY_DEV_DID"),
	}

	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins != "" {
		for _, o := range strings.Split(origins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
			}
		}
	}

	// Required everywhere.
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.AllowedOrigins) == 0 {
		return Config{}, fmt.Errorf("ALLOWED_ORIGINS is required (comma-separated list)")
	}

	// Required in dev only.
	if env == EnvDev && cfg.DevDID == "" {
		return Config{}, fmt.Errorf("CRAFTSKY_DEV_DID is required in dev")
	}
	// In prod, DevDID is intentionally ignored; clear it so callers don't
	// accidentally use a leftover value.
	if env == EnvProd {
		cfg.DevDID = ""
	}

	return cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/...`
Expected: `ok  social.craftsky/appview/internal/app  0.0Xs`. All 7 subtests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/app/config.go internal/app/config_test.go
git commit -m "feat(appview): add Config struct and LoadConfig"
```

### Task 1.4: `internal/db/db.go` — Postgres pool connector

**Files:**
- Create: `appview/internal/db/db.go`
- Create: `appview/internal/db/db_test.go`

- [ ] **Step 1: Write a failing test for the "bad URL" path**

A full integration test would need a live Postgres; we keep that for the final acceptance run. The unit test covers the observable error shape for an unparseable URL.

Create `appview/internal/db/db_test.go`:
```go
package db

import (
	"context"
	"testing"
)

func TestConnect_BadURLReturnsError(t *testing.T) {
	pool, err := Connect(context.Background(), "not a valid url")
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("expected error for invalid URL, got nil")
	}
	if pool != nil {
		t.Errorf("pool should be nil on error, got %v", pool)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/db/...`
Expected: compile error — `undefined: Connect`.

- [ ] **Step 3: Implement `Connect`**

Create `appview/internal/db/db.go`:
```go
// Package db owns the Postgres connection pool. It's a thin wrapper around
// pgxpool — other packages receive the resulting *pgxpool.Pool via
// app.Deps and don't import pgx directly.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect parses the given Postgres URL, builds a pool, and verifies the
// connection with a Ping. Returns the pool + nil on success, or nil + a
// wrapping error on parse/connect failure.
//
// Callers own the returned pool and must call pool.Close() when done.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/db/...`
Expected: `ok  social.craftsky/appview/internal/db  0.0Xs`. The bad-URL test confirms parse errors are surfaced.

- [ ] **Step 5: Verify full build still works**

Run: `go build ./...`
Expected: exits 0.

- [ ] **Step 6: Commit**

```bash
git add internal/db/db.go internal/db/db_test.go
git commit -m "feat(appview): add db.Connect wrapping pgxpool"
```

### Task 1.5: Environment files and gitignore

**Files:**
- Create: `appview/environments/dev.env`
- Create: `appview/environments/prod.env.example`
- Modify: `appview/.gitignore` (creating it)
- Create: `appview/migrations/.gitkeep`
- Create: `appview/queries/.gitkeep`

- [ ] **Step 1: Write `environments/dev.env`**

Create `appview/environments/dev.env`:
```
DATABASE_URL=postgres://craftsky:dev@localhost:5432/craftsky_dev?sslmode=disable
ALLOWED_ORIGINS=*
CRAFTSKY_DEV_DID=did:plc:craftsky-dev-user
```

Rationale: `*` origins is fine for local dev. The DB URL assumes the Postgres container in the plan's preamble; contributors on a different setup edit this file (it's checked in, and that's intentional — no secrets here).

- [ ] **Step 2: Write `environments/prod.env.example`**

Create `appview/environments/prod.env.example`:
```
# Copy to prod.env and fill in real values. prod.env is gitignored.
# Required in prod:
DATABASE_URL=postgres://USER:PASS@HOST:5432/DBNAME?sslmode=require
ALLOWED_ORIGINS=https://craftsky.social,https://www.craftsky.social
# CRAFTSKY_DEV_DID is ignored in prod; do not set.
```

- [ ] **Step 3: Create `appview/.gitignore` with prod.env entry**

Create `appview/.gitignore`:
```
# Production env file — contains secrets, never commit.
environments/prod.env
```

(The repo-level `.gitignore` already covers `bin/`, `tmp/`, etc. This file is scoped to appview-specific ignores.)

- [ ] **Step 4: Keep empty migrations/ and queries/ tracked**

Run:
```bash
touch migrations/.gitkeep queries/.gitkeep
```

- [ ] **Step 5: Verify git status**

Run: `git status`
Expected: the three new files in `environments/`, `.gitignore`, and the two `.gitkeep`s are all shown as untracked. No `prod.env` appears (it doesn't exist yet — the `.gitignore` entry is preventive).

- [ ] **Step 6: Commit**

```bash
git add environments/dev.env environments/prod.env.example .gitignore migrations/.gitkeep queries/.gitkeep
git commit -m "chore(appview): add env files and gitignore prod.env"
```

### Chunk 1 acceptance

Run from `appview/`:
- [ ] `go vet ./...` — exits 0, no output.
- [ ] `gofmt -l .` — exits 0, no output (every file formatted).
- [ ] `go test ./...` — all tests pass, 0 failures.
- [ ] `go build ./...` — exits 0 with no output.
- [ ] `git log --oneline -5` — shows the chunk's 5 commits.

Confirm before moving on: **no HTTP behaviour added yet**, but config + DB primitives are ready for Chunk 2.

---

## Chunk 2: Auth and middleware

**Scope:** Everything under `internal/auth/` and `internal/middleware/`. Defines the `AuthService` interface and both implementations (mock, not-implemented), and ports stash_hub's three middleware. No binaries wired yet.

### Task 2.1: `auth.AuthService` interface + dev-DID context helpers

**Files:**
- Create: `appview/internal/auth/service.go`
- Create: `appview/internal/auth/service_test.go`

- [ ] **Step 1: Write failing test for `WithDevDID` / `DevDIDFromContext`**

Create `appview/internal/auth/service_test.go`:
```go
package auth

import (
	"context"
	"testing"
)

func TestDevDIDRoundTrip(t *testing.T) {
	ctx := context.Background()

	got, ok := DevDIDFromContext(ctx)
	if ok {
		t.Errorf("empty ctx: ok=true, did=%q, want ok=false", got)
	}

	ctx = WithDevDID(ctx, "did:plc:abc")
	got, ok = DevDIDFromContext(ctx)
	if !ok {
		t.Fatal("after WithDevDID: ok=false, want true")
	}
	if got != "did:plc:abc" {
		t.Errorf("did = %q, want did:plc:abc", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth/...`
Expected: compile error — `undefined: WithDevDID`, `undefined: DevDIDFromContext`.

- [ ] **Step 3: Implement the interface and helpers**

Create `appview/internal/auth/service.go`:
```go
// Package auth defines the authentication contract the HTTP middleware
// uses and provides the dev (mock) and prod (not-yet-implemented)
// implementations.
//
// The interface is transport-agnostic: it takes a context and a token,
// returns a DID or error. HTTP-specific concerns (bearer header parsing,
// the X-Dev-DID override header) live in internal/middleware.
//
// Context helpers (WithDevDID / DevDIDFromContext) live here, not in
// middleware, so implementations can read from context without importing
// middleware — that would create a cycle.
package auth

import "context"

// AuthService validates a bearer token and returns the authenticated DID.
type AuthService interface {
	Authenticate(ctx context.Context, token string) (did string, err error)
}

// contextKey is unexported to prevent collisions across packages.
type contextKey string

const devDIDKey contextKey = "dev_did"

// WithDevDID returns a derived context carrying the given DID under the
// dev-DID key. Middleware calls this when the X-Dev-DID header is present.
func WithDevDID(ctx context.Context, did string) context.Context {
	return context.WithValue(ctx, devDIDKey, did)
}

// DevDIDFromContext extracts a dev-DID previously stored via WithDevDID.
// Returns ("", false) if none is present.
func DevDIDFromContext(ctx context.Context) (string, bool) {
	did, ok := ctx.Value(devDIDKey).(string)
	return did, ok
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/auth/...`
Expected: `ok  social.craftsky/appview/internal/auth  0.0Xs`

- [ ] **Step 5: Commit**

```bash
git add internal/auth/service.go internal/auth/service_test.go
git commit -m "feat(appview): add AuthService interface and dev-DID context helpers"
```

### Task 2.2: `MockAuthService`

**Files:**
- Create: `appview/internal/auth/mock.go`
- Modify: `appview/internal/auth/service_test.go` (append)

- [ ] **Step 1: Extend the test with mock cases**

Append to `appview/internal/auth/service_test.go`:
```go
func TestMockAuthService_FallsBackToDefaultDID(t *testing.T) {
	m := &MockAuthService{DefaultDID: "did:plc:default"}
	got, err := m.Authenticate(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "did:plc:default" {
		t.Errorf("did = %q, want did:plc:default", got)
	}
}

func TestMockAuthService_PrefersDevDIDFromContext(t *testing.T) {
	m := &MockAuthService{DefaultDID: "did:plc:default"}
	ctx := WithDevDID(context.Background(), "did:plc:override")
	got, err := m.Authenticate(ctx, "any-token")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "did:plc:override" {
		t.Errorf("did = %q, want did:plc:override", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth/...`
Expected: compile error — `undefined: MockAuthService`.

- [ ] **Step 3: Implement `MockAuthService`**

Create `appview/internal/auth/mock.go`:
```go
package auth

import "context"

// MockAuthService is the dev-only AuthService. It always authenticates.
// The returned DID comes from the request context (see WithDevDID) when
// present, otherwise DefaultDID.
type MockAuthService struct {
	DefaultDID string
}

var _ AuthService = (*MockAuthService)(nil)

func (m *MockAuthService) Authenticate(ctx context.Context, token string) (string, error) {
	if did, ok := DevDIDFromContext(ctx); ok {
		return did, nil
	}
	return m.DefaultDID, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/auth/...`
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/mock.go internal/auth/service_test.go
git commit -m "feat(appview): add MockAuthService with X-Dev-DID override"
```

### Task 2.3: `NotImplementedAuthService`

**Files:**
- Create: `appview/internal/auth/oauth.go`
- Modify: `appview/internal/auth/service_test.go` (append)

- [ ] **Step 1: Extend test**

Append to `appview/internal/auth/service_test.go`:
```go
func TestNotImplementedAuthService_AlwaysErrors(t *testing.T) {
	var s AuthService = &NotImplementedAuthService{}
	_, err := s.Authenticate(context.Background(), "any")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth/...`
Expected: compile error — `undefined: NotImplementedAuthService`.

- [ ] **Step 3: Implement**

Create `appview/internal/auth/oauth.go`:
```go
package auth

import (
	"context"
	"errors"
)

// NotImplementedAuthService is the prod AuthService until real atproto
// OAuth lands. It always returns an error. Wiring /whoami behind
// Authenticated in prod deliberately produces 401s, exercising the
// middleware path.
type NotImplementedAuthService struct{}

var _ AuthService = (*NotImplementedAuthService)(nil)

// ErrAuthNotImplemented is returned by NotImplementedAuthService.Authenticate
// so callers can type-check for it if they care (middleware doesn't — it
// returns 401 regardless).
var ErrAuthNotImplemented = errors.New("atproto OAuth not implemented yet")

func (NotImplementedAuthService) Authenticate(ctx context.Context, token string) (string, error) {
	return "", ErrAuthNotImplemented
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/auth/...`
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/oauth.go internal/auth/service_test.go
git commit -m "feat(appview): add NotImplementedAuthService for prod stub"
```

### Task 2.4: Logging middleware

**Files:**
- Create: `appview/internal/middleware/logging.go`
- Create: `appview/internal/middleware/logging_test.go`

- [ ] **Step 1: Write failing test**

Create `appview/internal/middleware/logging_test.go`:
```go
package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging_InjectsRunIDAndLogs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var seenRunID string
	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenRunID = GetRunID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if seenRunID == "" {
		t.Fatal("handler did not see a run_id in context")
	}

	logged := buf.String()
	if !strings.Contains(logged, `"msg":"Request received"`) {
		t.Errorf("log missing 'Request received': %s", logged)
	}
	if !strings.Contains(logged, `"method":"GET"`) {
		t.Errorf("log missing method: %s", logged)
	}
	if !strings.Contains(logged, `"path":"/health"`) {
		t.Errorf("log missing path: %s", logged)
	}
	if !strings.Contains(logged, seenRunID) {
		t.Errorf("log missing run_id %q: %s", seenRunID, logged)
	}
}

func TestGetRunID_EmptyWhenAbsent(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if got := GetRunID(req.Context()); got != "" {
		t.Errorf("GetRunID = %q, want empty", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/middleware/...`
Expected: compile error — `undefined: Logging`, `undefined: GetRunID`.

- [ ] **Step 3: Implement**

Create `appview/internal/middleware/logging.go`:
```go
// Package middleware holds HTTP middleware used by cmd/appview's NewServer.
//
// Every middleware is a constructor function that takes its dependencies
// at startup and returns a func(http.Handler) http.Handler — a shape that
// composes cleanly with standard library routing.
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// contextKey is a named type so middleware values can't collide with
// other packages' context keys.
type contextKey string

const runIDKey contextKey = "run_id"

// GetRunID extracts the per-request ID injected by the Logging middleware.
// Returns "" if no middleware ran (e.g. from a test that skipped it).
func GetRunID(ctx context.Context) string {
	if id, ok := ctx.Value(runIDKey).(string); ok {
		return id
	}
	return ""
}

// Logging returns middleware that assigns every request a UUID run_id,
// logs an Info "Request received" line with method + path + run_id, and
// puts the run_id in the request context for handlers to log against.
//
// It uses the supplied logger (typically deps.Logger), NOT slog.Default,
// so tests can capture output with a buffered handler.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runID := uuid.New().String()
			ctx := context.WithValue(r.Context(), runIDKey, runID)
			logger.Info("Request received",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("run_id", runID),
			)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/middleware/...`
Expected: both tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/logging.go internal/middleware/logging_test.go
git commit -m "feat(appview): add Logging middleware with run_id injection"
```

### Task 2.5: CORS middleware

**Files:**
- Create: `appview/internal/middleware/cors.go`
- Create: `appview/internal/middleware/cors_test.go`

- [ ] **Step 1: Write failing test**

Create `appview/internal/middleware/cors_test.go`:
```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_AllowsListedOrigin(t *testing.T) {
	handler := CORS([]string{"https://a.example"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://a.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://a.example" {
		t.Errorf("ACAO = %q, want https://a.example", got)
	}
}

func TestCORS_BlocksUnlistedOrigin(t *testing.T) {
	handler := CORS([]string{"https://a.example"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO = %q, want empty for unlisted origin", got)
	}
}

func TestCORS_WildcardAllowsAny(t *testing.T) {
	handler := CORS([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://random.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://random.example" {
		t.Errorf("ACAO = %q, want echoed origin under wildcard", got)
	}
}

func TestCORS_PreflightShortCircuits(t *testing.T) {
	var nextCalled bool
	handler := CORS([]string{"https://a.example"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://a.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if nextCalled {
		t.Error("next handler should not be called for OPTIONS preflight")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("preflight status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("preflight should set Access-Control-Allow-Methods")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/middleware/...`
Expected: compile error — `undefined: CORS`.

- [ ] **Step 3: Implement**

Create `appview/internal/middleware/cors.go`:
```go
package middleware

import (
	"net/http"
)

// CORS returns middleware that handles CORS for the given allow-list.
//
// The allow-list is an explicit list of exact origins. The special value
// "*" matches any origin (used in dev); when wildcarded, the request's
// Origin header is echoed back rather than sending a literal "*" so that
// credentialed requests still work.
//
// Preflight (OPTIONS) requests short-circuit with 200 after the headers
// are set. Non-preflight requests pass through to next with the
// Access-Control-Allow-Origin header set iff the origin is allowed.
//
// Day one: only exact-string match and the "*" wildcard. No subdomain
// patterns, no regex — add them to the spec and this function together
// when a concrete case appears.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if isOriginAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Dev-DID")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isOriginAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return false
	}
	for _, a := range allowed {
		if a == "*" {
			return true
		}
		if a == origin {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/middleware/...`
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/cors.go internal/middleware/cors_test.go
git commit -m "feat(appview): add CORS middleware"
```

### Task 2.6: Auth middleware

**Files:**
- Create: `appview/internal/middleware/auth.go`
- Create: `appview/internal/middleware/auth_test.go`

- [ ] **Step 1: Write failing test**

Create `appview/internal/middleware/auth_test.go`:
```go
package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/auth"
)

// discardLogger returns a slog.Logger that drops everything. Used by tests
// that assert HTTP behaviour without caring about log output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// passthroughHandler captures the DID seen in context and responds 200.
func passthroughHandler(didSeen *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*didSeen, _ = GetDID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthenticated_RejectsMissingHeader(t *testing.T) {
	var seen string
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger())(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticated_RejectsMalformedHeader(t *testing.T) {
	var seen string
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger())(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticated_RejectsEmptyBearer(t *testing.T) {
	var seen string
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger())(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticated_MockSuccessUsesDefaultDID(t *testing.T) {
	var seen string
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger())(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if seen != "did:plc:default" {
		t.Errorf("did seen = %q, want did:plc:default", seen)
	}
}

func TestAuthenticated_MockHonoursXDevDID(t *testing.T) {
	var seen string
	h := Authenticated(&auth.MockAuthService{DefaultDID: "did:plc:default"}, discardLogger())(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Dev-DID", "did:plc:override")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if seen != "did:plc:override" {
		t.Errorf("did seen = %q, want did:plc:override", seen)
	}
}

func TestAuthenticated_NotImplementedReturns401(t *testing.T) {
	var seen string
	h := Authenticated(auth.NotImplementedAuthService{}, discardLogger())(passthroughHandler(&seen))
	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "Unauthorized" {
		t.Errorf("body = %q, want Unauthorized", rec.Body.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/middleware/...`
Expected: compile error — `undefined: Authenticated`, `undefined: GetDID`.

- [ ] **Step 3: Implement**

Create `appview/internal/middleware/auth.go`:
```go
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"social.craftsky/appview/internal/auth"
)

const didKey contextKey = "did"

// GetDID extracts the authenticated DID injected by the Authenticated
// middleware. Returns ("", false) if no middleware ran or if the request
// reached the handler without authentication (which shouldn't happen on
// routes wired via Authenticated, but GetDID stays safe either way).
func GetDID(ctx context.Context) (string, bool) {
	did, ok := ctx.Value(didKey).(string)
	return did, ok
}

// Authenticated returns middleware that validates a bearer token via
// authService and injects the authenticated DID into the request context.
//
// Follows the same constructor-returning-wrapper shape as Logging and
// CORS so routing code can compose them uniformly:
//
//	mux.Handle("/whoami", middleware.Authenticated(deps.AuthService, deps.Logger)(handler))
//
// Flow:
//  1. Extract the bearer token from the Authorization header. Missing or
//     malformed → 401.
//  2. If the request carries X-Dev-DID, inject it into the context via
//     auth.WithDevDID. MockAuthService reads this; other impls ignore it.
//  3. Call authService.Authenticate(ctx, token). Error → 401.
//  4. Inject the returned DID into the context under didKey and call next.
//
// The X-Dev-DID sniff is unconditional: in prod, NotImplementedAuthService
// errors regardless, so 401 is the outcome. This keeps the middleware
// free of Env-awareness.
func Authenticated(authService auth.AuthService, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const bearerPrefix = "Bearer "
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				logger.Warn("auth: missing or malformed Authorization header",
					slog.String("run_id", GetRunID(r.Context())))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
			if token == "" {
				logger.Warn("auth: empty bearer token",
					slog.String("run_id", GetRunID(r.Context())))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			if devDID := r.Header.Get("X-Dev-DID"); devDID != "" {
				ctx = auth.WithDevDID(ctx, devDID)
			}

			did, err := authService.Authenticate(ctx, token)
			if err != nil {
				logger.Warn("auth: Authenticate returned error",
					slog.String("err", err.Error()),
					slog.String("run_id", GetRunID(r.Context())))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx = context.WithValue(ctx, didKey, did)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

Note: `http.Error` appends `"\n"` after the message; the test uses `strings.TrimSpace` to account for it.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/middleware/...`
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/auth.go internal/middleware/auth_test.go
git commit -m "feat(appview): add Authenticated middleware"
```

### Chunk 2 acceptance

Run from `appview/`:
- [ ] `go vet ./...` — exits 0.
- [ ] `gofmt -l .` — exits 0, no output.
- [ ] `go test ./...` — all tests pass.
- [ ] `go build ./...` — exits 0.
- [ ] `git log --oneline -10` — shows all chunk 2 commits.

No binaries run yet, but every middleware and auth implementation has a test covering its happy path and at least one failure mode.

---

*(The plan continues with Chunks 3, 4, and 5. Each is written after the preceding chunk has been reviewed and approved, per the plan-review-loop process.)*
