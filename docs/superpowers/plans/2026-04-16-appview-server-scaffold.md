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

### Task 1.1: Note on module dependencies (no-op)

No action for this task. Explanation for the implementer:

We cannot `go get` the five scaffold dependencies up-front because `go mod tidy` (which `go get` runs implicitly in module mode) strips any dependency that nothing imports. Attempting `go get ...@latest` in a module whose only code is the stub `main.go` ends with an empty `go.mod`.

Instead, each downstream task that first imports a package adds it with `go get <pkg>@latest` as its own initial step. The distribution is:

| Task | `go get` |
|------|----------|
| 1.3 (Config+LoadConfig) | `github.com/joho/godotenv@latest` |
| 1.4 (db.Connect) | `github.com/jackc/pgx/v5@latest` |
| 2.4 (Logging middleware) | `github.com/google/uuid@latest` |
| 4.1 (cobra root) | `github.com/spf13/cobra@latest` |
| 4.4 (migrate subcommand) | `github.com/golang-migrate/migrate/v4@latest` |

**Go directive heads-up for Task 4.4:** `golang-migrate/v4` requires a newer `go` directive than the repo's current `go 1.23`. `go get` will bump `go.mod`'s `go` directive to whatever migrate/v4 needs (likely `go 1.25.x`). This is expected — Task 4.4's first step calls it out explicitly. Do not revert.

No commit in this task. Proceed to Task 1.2.

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
- Modify: `appview/go.mod`, `appview/go.sum` (via `go get`)

- [ ] **Step 0: Add the godotenv dependency**

Run: `go get github.com/joho/godotenv@latest`
Expected: `go: added github.com/joho/godotenv v1.x.x`. `go.mod` gains a require entry; `go.sum` gains hashes. The dep will stick because the implementation in Step 3 imports it.

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
// It UNSETS the relevant env vars before the test — not just sets them to
// empty. godotenv.Load treats a set-but-empty variable as "present" and
// skips the file's value, which would be the opposite of what we want in
// every test here. Unsetting keeps the file as the source of truth, and
// the t.Cleanup block restores any prior value after the test.
func testConfigFile(t *testing.T, contents string) string {
	t.Helper()
	for _, k := range []string{"DATABASE_URL", "ALLOWED_ORIGINS", "CRAFTSKY_DEV_DID"} {
		prior, had := os.LookupEnv(k)
		_ = os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(k, prior)
			} else {
				_ = os.Unsetenv(k)
			}
		})
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
- Modify: `appview/go.mod`, `appview/go.sum` (via `go get`)

- [ ] **Step 0: Add the pgx/v5 dependency**

Run: `go get github.com/jackc/pgx/v5@latest`
Expected: `go: added github.com/jackc/pgx/v5 v5.x.x`. The `pgxpool` subpackage comes along for free.

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
- Modify: `appview/go.mod`, `appview/go.sum` (via `go get`)

- [ ] **Step 0: Add the uuid dependency**

Run: `go get github.com/google/uuid@latest`
Expected: `go: added github.com/google/uuid v1.x.x`.

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

## Chunk 3: Server binary — stubs, routes, handlers, `Deps`, `main.go`

**Scope:** Wire a running HTTP server. Introduces stub `firehose.Subscriber` and `index.Indexer`, the `api` handlers (`HealthHandler`, `WhoAmIHandler`), `routes.AddRoutes`, `internal/app/deps.go` with `NewDevDeps`/`NewProdDeps`, then `cmd/appview/server.go` and `cmd/appview/main.go`. End of chunk: `go run ./cmd/appview dev` serves `GET /health` and `GET /whoami`.

### Task 3.1: Stub `firehose.Subscriber` interface and NotImplemented impl

**Files:**
- Create: `appview/internal/firehose/subscriber.go`
- Create: `appview/internal/firehose/subscriber_test.go`

- [ ] **Step 1: Write failing test**

Create `appview/internal/firehose/subscriber_test.go`:
```go
package firehose

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNotImplemented_ReplayErrors(t *testing.T) {
	var s Subscriber = NotImplemented{}
	err := s.Replay(context.Background(), time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "firehose") || !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("err = %q, want containing 'firehose' and 'not yet implemented'", err.Error())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/firehose/...`
Expected: compile error — `undefined: Subscriber`, `undefined: NotImplemented`.

- [ ] **Step 3: Implement the stub**

Create `appview/internal/firehose/subscriber.go`:
```go
// Package firehose defines the contract for consuming the atproto Relay
// firehose. Day one contains only the interface and a NotImplemented stub
// so the CLI's firehose-replay subcommand compiles and returns a clean
// error. Real subscription logic lands in a later commit.
package firehose

import (
	"context"
	"errors"
	"time"
)

// Subscriber replays firehose events into the indexer.
type Subscriber interface {
	// Replay re-indexes firehose events since the given timestamp.
	Replay(ctx context.Context, since time.Time) error
}

// NotImplemented is the day-one stub. Every method returns a descriptive
// error; the CLI surfaces this to stdout with exit code 1.
type NotImplemented struct{}

var _ Subscriber = NotImplemented{}

func (NotImplemented) Replay(ctx context.Context, since time.Time) error {
	return errors.New("firehose: not yet implemented")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/firehose/...`
Expected: `ok  social.craftsky/appview/internal/firehose  0.0Xs`

- [ ] **Step 5: Commit**

```bash
git add internal/firehose/subscriber.go internal/firehose/subscriber_test.go
git commit -m "feat(appview): add firehose.Subscriber interface and NotImplemented stub"
```

### Task 3.2: Stub `index.Indexer` interface and NotImplemented impl

**Files:**
- Create: `appview/internal/index/indexer.go`
- Create: `appview/internal/index/indexer_test.go`

- [ ] **Step 1: Write failing test**

Create `appview/internal/index/indexer_test.go`:
```go
package index

import (
	"context"
	"strings"
	"testing"
)

func TestNotImplemented_BackfillErrors(t *testing.T) {
	var idx Indexer = NotImplemented{}
	err := idx.Backfill(context.Background(), "did:plc:abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "indexer") || !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("err = %q, want containing 'indexer' and 'not yet implemented'", err.Error())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/index/...`
Expected: compile error — `undefined: Indexer`, `undefined: NotImplemented`.

- [ ] **Step 3: Implement the stub**

Create `appview/internal/index/indexer.go`:
```go
// Package index defines the contract for writing atproto records into
// Postgres. Day one contains only the interface and a NotImplemented stub
// so the CLI's backfill subcommand compiles.
package index

import (
	"context"
	"errors"
)

// Indexer writes records into the application's Postgres store.
type Indexer interface {
	// Backfill re-indexes all records for the given DID from its PDS.
	Backfill(ctx context.Context, did string) error
}

// NotImplemented is the day-one stub.
type NotImplemented struct{}

var _ Indexer = NotImplemented{}

func (NotImplemented) Backfill(ctx context.Context, did string) error {
	return errors.New("indexer: not yet implemented")
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/index/...`
Expected: `ok  social.craftsky/appview/internal/index  0.0Xs`

- [ ] **Step 5: Commit**

```bash
git add internal/index/indexer.go internal/index/indexer_test.go
git commit -m "feat(appview): add index.Indexer interface and NotImplemented stub"
```

### Task 3.3: Placeholder `internal/models/doc.go`

**Files:**
- Create: `appview/internal/models/doc.go`

No test — this file exists only so the package is importable once sqlc output starts landing. It has no exported symbols.

- [ ] **Step 1: Create the file**

Create `appview/internal/models/doc.go`:
```go
// Package models holds sqlc-generated types for Postgres rows and query
// results. The generated files live alongside this one; nothing is
// hand-written in here. Day one: empty.
package models
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/models/...`
Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add internal/models/doc.go
git commit -m "chore(appview): reserve internal/models for sqlc output"
```

### Task 3.4: `api.HealthHandler`

**Files:**
- Create: `appview/internal/api/health.go`
- Create: `appview/internal/api/health_test.go`

The Ping-error path needs a pool whose `Ping` can be made to fail without a real DB. We do that by creating a pool with a bogus config that parses successfully but points at an unreachable address.

Actually — `pgxpool.Pool` has no interface we can mock and `Ping` uses the real network stack. For unit-testing the 200 path we'd need an integration test. We split:
- Unit test: the 200 path is verified by using a pool pointed at a socket that IS reachable (see Task 3.11 acceptance — the full-server integration run).
- Unit test here: the 503 path, using `pgxpool.New` with a valid URL that refuses connections fast (`postgres://127.0.0.1:1/doesnotexist?connect_timeout=1`).

- [ ] **Step 1: Write failing test (503 path)**

Create `appview/internal/api/health_test.go`:
```go
package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// newUnreachablePool returns a pool whose Ping fails quickly. Used to test
// the 503 path without needing a live DB.
func newUnreachablePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	// 127.0.0.1:1 is a reserved port; connect will refuse immediately.
	cfg, err := pgxpool.ParseConfig("postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return pool
}

func TestHealth_ReturnsServiceUnavailableWhenDBDown(t *testing.T) {
	pool := newUnreachablePool(t)
	defer pool.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := HealthHandler(pool, logger)
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "db unreachable") {
		t.Errorf("body = %q, want 'db unreachable'", rec.Body.String())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/api/...`
Expected: compile error — `undefined: HealthHandler`.

- [ ] **Step 3: Implement `HealthHandler`**

Create `appview/internal/api/health.go`:
```go
// Package api holds HTTP handler factories. Each handler factory takes
// only the specific dependencies it needs — never the full *app.Deps —
// so handlers can't silently grow dependencies over time.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler returns a handler that pings the DB pool and reports
// 200 on success or 503 on failure. The ping is given a 2-second
// per-request timeout so a hung DB doesn't hang health checks.
//
// The response contract:
//   - 200 + application/json + {"status":"ok"}
//   - 503 + text/plain + "db unreachable"
// The underlying error is logged at Error via logger but not returned
// to the client.
func HealthHandler(pool *pgxpool.Pool, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			logger.Error("health: db ping failed", slog.String("err", err.Error()))
			// http.Error sets Content-Type: text/plain; charset=utf-8 itself.
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/api/...`
Expected: `ok  social.craftsky/appview/internal/api`. The unreachable-port probe should fail fast (under ~2s).

- [ ] **Step 5: Commit**

```bash
git add internal/api/health.go internal/api/health_test.go
git commit -m "feat(appview): add api.HealthHandler"
```

### Task 3.5: `api.WhoAmIHandler`

**Files:**
- Create: `appview/internal/api/whoami.go`
- Create: `appview/internal/api/whoami_test.go`

- [ ] **Step 1: Write failing test**

Rationale: `middleware.Authenticated` uses an unexported `didKey` to store the DID in context, so the test can't bypass the middleware to pre-populate it. We instead run the full `Authenticated(mock, logger)(WhoAmIHandler())` chain — which mirrors exactly how the route is wired in production.

Create `appview/internal/api/whoami_test.go`:
```go
package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/middleware"
)

// authTestMock is an inline AuthService that returns a fixed DID. It lets
// whoami_test.go inject a DID into the request context via the real
// Authenticated middleware, without depending on internal/auth.
type authTestMock struct{ did string }

func (m *authTestMock) Authenticate(ctx context.Context, token string) (string, error) {
	return m.did, nil
}

func TestWhoAmI_ReturnsDIDFromContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	outer := middleware.Authenticated(&authTestMock{did: "did:plc:alice"}, logger)(WhoAmIHandler())

	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	outer.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body struct {
		DID string `json:"did"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DID != "did:plc:alice" {
		t.Errorf("did = %q, want did:plc:alice", body.DID)
	}
}

func TestWhoAmI_WithoutDIDInContextReturns500(t *testing.T) {
	// Call the handler directly without running Authenticated — a routing
	// bug that's worth failing loudly on rather than silently returning
	// {"did":""}.
	h := WhoAmIHandler()
	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/api/...`
Expected: compile error — `undefined: WhoAmIHandler`.

- [ ] **Step 3: Implement `WhoAmIHandler`**

Create `appview/internal/api/whoami.go`:
```go
package api

import (
	"encoding/json"
	"net/http"

	"social.craftsky/appview/internal/middleware"
)

// WhoAmIHandler returns the caller's authenticated DID as JSON. It
// assumes middleware.Authenticated has run — if not, it returns 500
// with a "no did in context" body, which would be a routing bug.
func WhoAmIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			http.Error(w, "no did in context", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"did": did})
	})
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/api/...`
Expected: both tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/api/whoami.go internal/api/whoami_test.go
git commit -m "feat(appview): add api.WhoAmIHandler"
```

### Task 3.6: `internal/app/deps.go` — `Deps` struct + factories

**Files:**
- Create: `appview/internal/app/deps.go`
- Create: `appview/internal/app/deps_test.go`

- [ ] **Step 1: Write failing test**

The factories need a reachable DB to succeed. For unit tests we use the same unreachable-pool trick to assert they fail cleanly, plus verify the struct fields for a happy-path test driven via `testing.T`'s tempdir. A real happy-path run lives in Chunk 3 acceptance (Task 3.11).

Create `appview/internal/app/deps_test.go`:
```go
package app

import (
	"context"
	"strings"
	"testing"

	"social.craftsky/appview/internal/auth"
)

func TestNewDevDeps_UnreachableDBReturnsError(t *testing.T) {
	cfg := Config{
		Env:            EnvDev,
		DatabaseURL:    "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1",
		AllowedOrigins: []string{"*"},
		DevDID:         "did:plc:test",
	}
	deps, cleanup, err := NewDevDeps(context.Background(), cfg)
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		if deps != nil && deps.DB != nil {
			deps.DB.Close()
		}
		t.Fatal("expected error for unreachable DB, got nil")
	}
	if !strings.Contains(err.Error(), "db") && !strings.Contains(err.Error(), "ping") {
		t.Errorf("err = %v, expected db/ping context", err)
	}
	if deps != nil {
		t.Errorf("deps = %v, want nil on error", deps)
	}
}

func TestNewProdDeps_UnreachableDBReturnsError(t *testing.T) {
	cfg := Config{
		Env:            EnvProd,
		DatabaseURL:    "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1",
		AllowedOrigins: []string{"https://craftsky.social"},
	}
	_, _, err := NewProdDeps(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Covers the "which auth service gets wired" contract without touching
// the network: we construct Deps by hand and assert the field types match
// what each factory would have produced. This pins the behaviour even
// when a reachable DB isn't available.
func TestDepsAuthServiceShape(t *testing.T) {
	// Dev: MockAuthService
	devDeps := &Deps{
		Config:      Config{Env: EnvDev, DevDID: "did:plc:default"},
		AuthService: &auth.MockAuthService{DefaultDID: "did:plc:default"},
	}
	if _, ok := devDeps.AuthService.(*auth.MockAuthService); !ok {
		t.Errorf("dev: AuthService = %T, want *auth.MockAuthService", devDeps.AuthService)
	}

	// Prod: NotImplementedAuthService
	prodDeps := &Deps{
		Config:      Config{Env: EnvProd},
		AuthService: auth.NotImplementedAuthService{},
	}
	if _, ok := prodDeps.AuthService.(auth.NotImplementedAuthService); !ok {
		t.Errorf("prod: AuthService = %T, want auth.NotImplementedAuthService", prodDeps.AuthService)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/app/...`
Expected: compile error — `undefined: NewDevDeps`, `undefined: NewProdDeps`, `undefined: Deps`.

- [ ] **Step 3: Implement `deps.go`**

Create `appview/internal/app/deps.go`:
```go
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/firehose"
	"social.craftsky/appview/internal/index"
)

// Deps is the fully-wired set of dependencies for one Craftsky App View
// process. NewDevDeps and NewProdDeps build it; cmd/appview and cmd/cli
// both consume it.
//
// Deps is passed into NewServer, routes.AddRoutes, and CLI subcommand
// entry points — it is never passed into an individual HTTP handler.
// Handler factories in internal/api take only the specific dependencies
// they use.
type Deps struct {
	Config      Config
	Logger      *slog.Logger
	DB          *pgxpool.Pool
	AuthService auth.AuthService

	// Day-one stubs. Shape is stable so CLI subcommands compile.
	Firehose firehose.Subscriber
	Indexer  index.Indexer
}

// NewDevDeps wires the dev variant: debug-level logger, MockAuthService,
// NotImplemented firehose+indexer.
func NewDevDeps(ctx context.Context, cfg Config) (*Deps, func(), error) {
	return newDeps(ctx, cfg, slog.LevelDebug, &auth.MockAuthService{DefaultDID: cfg.DevDID})
}

// NewProdDeps wires the prod variant: info-level logger,
// NotImplementedAuthService, NotImplemented firehose+indexer.
func NewProdDeps(ctx context.Context, cfg Config) (*Deps, func(), error) {
	return newDeps(ctx, cfg, slog.LevelInfo, auth.NotImplementedAuthService{})
}

// newDeps is the shared core of NewDevDeps and NewProdDeps. Keeping the
// divergence to just the three parameters (log level, auth service, and
// cfg) makes the env-conditional surface easy to audit.
func newDeps(ctx context.Context, cfg Config, level slog.Level, authSvc auth.AuthService) (*Deps, func(), error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	// Third-party libs that reach for slog.Default should get our logger.
	slog.SetDefault(logger)

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("db connect: %w", err)
	}

	deps := &Deps{
		Config:      cfg,
		Logger:      logger,
		DB:          pool,
		AuthService: authSvc,
		Firehose:    firehose.NotImplemented{},
		Indexer:     index.NotImplemented{},
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			deps.DB.Close()
			deps.Logger.Info("shutdown: db pool closed")
		})
	}

	// Startup log lines per spec. AC #1/#2 check for presence/absence.
	if cfg.Env == EnvDev {
		logger.Debug("log level", slog.String("level", "debug"))
	}
	logger.Info("deps initialised", slog.String("env", string(cfg.Env)))

	return deps, cleanup, nil
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/app/...`
Expected: all four tests pass (TestParseEnv + Load* + Deps unreachable tests + shape test).

- [ ] **Step 5: Verify no other package broke**

Run: `go build ./...` and `go test ./...`
Expected: both exit 0.

- [ ] **Step 6: Commit**

```bash
git add internal/app/deps.go internal/app/deps_test.go
git commit -m "feat(appview): add app.Deps with NewDevDeps/NewProdDeps factories"
```

### Task 3.7: `internal/routes/routes.go`

**Files:**
- Create: `appview/internal/routes/routes.go`
- Create: `appview/internal/routes/routes_test.go`

- [ ] **Step 1: Write failing test**

The routes test doesn't need a live DB — it verifies route registration by making HTTP requests and checking the router's dispatch behaviour. For `/health` we still need a DB, so we skip that assertion here (covered by the end-to-end run in Task 3.11). For `/whoami` and the `/` catch-all, we can assert.

Create `appview/internal/routes/routes_test.go`:
```go
package routes

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
)

func testDeps() *app.Deps {
	return &app.Deps{
		Config:      app.Config{Env: app.EnvDev, AllowedOrigins: []string{"*"}, DevDID: "did:plc:test"},
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService: &auth.MockAuthService{DefaultDID: "did:plc:test"},
		// DB, Firehose, Indexer left nil — routes doesn't need them and
		// /health isn't tested here.
	}
}

func TestAddRoutes_WhoAmIAuthenticatedReturnsDID(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/whoami", nil)
	req.Header.Set("Authorization", "Bearer anything")
	req.Header.Set("X-Dev-DID", "did:plc:from-header")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "did:plc:from-header") {
		t.Errorf("body = %q, want containing 'did:plc:from-header'", rec.Body.String())
	}
}

func TestAddRoutes_WhoAmIWithoutAuthReturns401(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/whoami", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAddRoutes_UnknownPathReturns404(t *testing.T) {
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDeps())

	req := httptest.NewRequest("GET", "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/routes/...`
Expected: compile error — `undefined: AddRoutes`.

- [ ] **Step 3: Implement `AddRoutes`**

Create `appview/internal/routes/routes.go`:
```go
// Package routes wires the App View's HTTP routes onto a *http.ServeMux.
// Each handler factory in internal/api takes only the specific deps it
// needs; this package owns the mapping from URL → handler + middleware.
package routes

import (
	"context"
	"net/http"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/middleware"
)

// AddRoutes registers all App View routes on mux.
//
// ctx is the startup-scope context (used by future route-time validation,
// e.g. checking that a required table exists at boot). Per-request work
// inside handlers uses r.Context(), not this ctx.
func AddRoutes(ctx context.Context, mux *http.ServeMux, deps *app.Deps) {
	// Public.
	mux.Handle("GET /health", api.HealthHandler(deps.DB, deps.Logger))

	// Authenticated.
	authN := middleware.Authenticated(deps.AuthService, deps.Logger)
	mux.Handle("GET /whoami", authN(api.WhoAmIHandler()))

	// Fallthrough.
	mux.Handle("/", http.NotFoundHandler())
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/routes/...`
Expected: all three tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/routes/routes.go internal/routes/routes_test.go
git commit -m "feat(appview): add routes.AddRoutes for /health and /whoami"
```

### Task 3.8: `cmd/appview/server.go`

**Files:**
- Create: `appview/cmd/appview/server.go`

`server.go` is trivially thin — no dedicated test, it's exercised through the end-to-end run in Task 3.11. The existing `main.go` from the pre-scaffold state is overwritten in the next task.

- [ ] **Step 1: Create `server.go`**

Create `appview/cmd/appview/server.go`:
```go
package main

import (
	"context"
	"net/http"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/routes"
)

// NewServer constructs the App View's HTTP handler. main.go wraps it in
// a *http.Server; this function stays focused on routing and middleware.
//
// Middleware stack (outside-in):
//   Logging  (assigns run_id, logs every request)
//   CORS     (origin check, preflight handling)
//   mux      (routing — Authenticated is applied per-route)
func NewServer(ctx context.Context, deps *app.Deps) http.Handler {
	mux := http.NewServeMux()
	routes.AddRoutes(ctx, mux, deps)

	var h http.Handler = mux
	h = middleware.CORS(deps.Config.AllowedOrigins)(h)
	h = middleware.Logging(deps.Logger)(h)
	return h
}
```

- [ ] **Step 2: Verify build still works**

Run: `go build ./...`
Expected: compiles cleanly (existing `main.go` stub still there; `server.go` is unused but the `NewServer` symbol is internal to `package main`, Go's unused-function check doesn't apply at file scope).

- [ ] **Step 3: Commit**

```bash
git add cmd/appview/server.go
git commit -m "feat(appview): add NewServer wiring middleware stack onto mux"
```

### Task 3.9: `cmd/appview/main.go` — replace the stub

**Files:**
- Modify: `appview/cmd/appview/main.go`

No dedicated unit test — `main` packages are awkward to unit-test and the behaviour is covered by the end-to-end run in Task 3.11.

- [ ] **Step 1: Replace `main.go`**

Replace `appview/cmd/appview/main.go` entirely with:
```go
// Command appview runs the Craftsky App View HTTP server.
//
// Usage:
//   appview dev
//   appview prod
//
// The positional argument selects the environment file under
// environments/ and the dev/prod divergent wiring (log level, auth
// service, CORS permissiveness).
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"social.craftsky/appview/internal/app"
)

func main() {
	if err := run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	// Signal handling wraps the whole run so Ctrl-C during deps init
	// (e.g. slow DB connect) exits cleanly.
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// SIGPIPE is sent when a client disconnects mid-write. Go's net/http
	// already surfaces this as an error return; we don't need the signal.
	signal.Ignore(syscall.SIGPIPE)

	if len(args) <= 1 {
		return fmt.Errorf("expected argument of either 'dev' or 'prod'")
	}
	env, err := app.ParseEnv(args[1])
	if err != nil {
		return err
	}

	cfg, err := app.LoadConfig(env, fmt.Sprintf("environments/%s.env", env))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var deps *app.Deps
	var cleanup func()
	switch env {
	case app.EnvDev:
		deps, cleanup, err = app.NewDevDeps(ctx, cfg)
	case app.EnvProd:
		deps, cleanup, err = app.NewProdDeps(ctx, cfg)
	default:
		// ParseEnv should have rejected anything else, but defense in depth.
		return fmt.Errorf("unreachable: unknown env %q after ParseEnv", env)
	}
	if err != nil {
		return fmt.Errorf("build deps: %w", err)
	}
	defer cleanup()

	httpServer := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", "8080"),
		Handler: NewServer(ctx, deps),
	}

	go func() {
		deps.Logger.Info("listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			deps.Logger.Error("server error", "err", err.Error())
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		deps.Logger.Info("shutdown: received signal")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			deps.Logger.Error("shutdown error", "err", err.Error())
		}
		deps.Logger.Info("shutdown: http server stopped")
	}()
	wg.Wait()
	return nil
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: exits 0. `appview/bin/` or the build cache now holds a compiled binary for `cmd/appview`.

- [ ] **Step 3: Verify the full test suite still passes**

Run: `go test ./...`
Expected: every package's tests pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/appview/main.go
git commit -m "feat(appview): replace main stub with full signal-aware server lifecycle"
```

### Task 3.10: Manual smoke of the server binary (no Postgres needed yet)

**Files:** none modified.

This task catches regressions before the DB-dependent acceptance run.

- [ ] **Step 1: Run server with no args**

Run: `go run ./cmd/appview`
Expected: exits non-zero, prints `expected argument of either 'dev' or 'prod'` to stderr.

- [ ] **Step 2: Run with invalid arg**

Run: `go run ./cmd/appview staging`
Expected: exits non-zero, prints `unknown env "staging": expected "dev" or "prod"` (or similar) to stderr.

- [ ] **Step 3: Run dev with DATABASE_URL cleared**

Override the file's `DATABASE_URL` with an empty shell env var (os.Getenv wins over the file per `LoadConfig`'s contract):
```bash
DATABASE_URL= go run ./cmd/appview dev
```
Expected: exits non-zero with `load config: DATABASE_URL is required`.

This avoids any file shuffling that could leave a `.bak` around if interrupted.

- [ ] **Step 4: Commit: nothing (no files changed)**

Nothing to commit — this task is pure verification. Proceed.

### Task 3.11: Chunk 3 acceptance — live DB run

**Files:** none modified.

- [ ] **Step 1: Ensure a Postgres is running**

If you don't already have one on port 5432:
```bash
docker run --rm -d --name craftsky-dev-pg \
  -p 5432:5432 \
  -e POSTGRES_USER=craftsky \
  -e POSTGRES_PASSWORD=dev \
  -e POSTGRES_DB=craftsky_dev \
  postgres:16
```

Wait 3 seconds, then verify: `docker exec craftsky-dev-pg pg_isready -U craftsky`. Expected: `/var/run/postgresql:5432 - accepting connections`.

- [ ] **Step 2: Build the binary, start it in the background, wait for readiness**

From `appview/`:
```bash
go build -o /tmp/appview ./cmd/appview
/tmp/appview dev > /tmp/appview.log 2>&1 &
APPVIEW_PID=$!
# Poll /health until the server is up (max ~10s).
for i in {1..50}; do
  if curl -sS -o /dev/null http://localhost:8080/health 2>/dev/null; then break; fi
  sleep 0.2
done
```

Why build-then-run rather than `go run`: `go run` spawns a wrapper process, and forwarding SIGTERM to the compiled child depends on the Go runtime version. A standalone binary receives signals directly, so Step 7's shutdown assertions are reliable.

Why redirect to `/tmp/appview.log`: Step 7 asserts the server's stdout contains three specific lines in order. Redirecting keeps that log stream separate from curl output so you can grep it deterministically.

Expected after the poll loop: the server is responding on 8080. `cat /tmp/appview.log` should show JSON lines for `deps initialised` (Info), `log level` (Debug, dev-only), and `listening`.

- [ ] **Step 3: Hit `/health`**

Run: `curl -sS -o /tmp/health-body -w "%{http_code}\n" http://localhost:8080/health`
Expected: prints `200`. `cat /tmp/health-body` → `{"status":"ok"}`.

- [ ] **Step 4: Hit `/whoami` with `X-Dev-DID`**

Run:
```bash
curl -sS -o /tmp/whoami -w "%{http_code}\n" \
  -H "Authorization: Bearer anything" \
  -H "X-Dev-DID: did:plc:test123" \
  http://localhost:8080/whoami
```
Expected: prints `200`. `cat /tmp/whoami` → `{"did":"did:plc:test123"}`.

- [ ] **Step 5: Hit `/whoami` with no auth**

Run: `curl -sS -o /dev/null -w "%{http_code}\n" http://localhost:8080/whoami`
Expected: prints `401`.

- [ ] **Step 6: Hit an unknown path**

Run: `curl -sS -o /dev/null -w "%{http_code}\n" http://localhost:8080/does-not-exist`
Expected: prints `404`.

- [ ] **Step 6a: `/health` with the DB down returns 503**

This exercises the spec's AC #4 negative half while the server is still running.

```bash
docker stop craftsky-dev-pg
# Wait a moment for the server's pool to notice.
sleep 1
curl -sS -o /tmp/health-down -w "%{http_code}\n" http://localhost:8080/health
cat /tmp/health-down
docker start craftsky-dev-pg
```

Expected: `curl` prints `503`. `/tmp/health-down` contains `db unreachable`. After the restart, Postgres takes ~1-2 s to become ready again; a follow-up `curl http://localhost:8080/health` should then show 200.

- [ ] **Step 7: Graceful shutdown**

Run:
```bash
kill -TERM $APPVIEW_PID
wait $APPVIEW_PID
grep -o 'shutdown: [a-z ]*' /tmp/appview.log
```

Expected:
- `wait` exits 0.
- `grep` prints, in order:
  ```
  shutdown: received signal
  shutdown: http server stopped
  shutdown: db pool closed
  ```

If any line is missing or they're out of order, review `/tmp/appview.log` — the full JSON-log context usually pinpoints the broken step.

- [ ] **Step 8: Stop Postgres (if you started it)**

Run: `docker stop craftsky-dev-pg`
(If you used a pre-existing Postgres, skip.)

- [ ] **Step 8a: Prod startup + log-level evidence**

This covers the half of AC #2 that Task 3.10 Step 3 doesn't: verifying prod actually starts cleanly when the env is valid, AND that the dev-only debug line is NOT emitted.

Temporary prod.env (deleted after the check so no secrets linger). The `trap` guards against a Ctrl-C leaving a stale prod.env behind — once real secrets are in play a leftover file could silently override a later `appview prod` invocation.
```bash
trap 'rm -f environments/prod.env' EXIT INT TERM

cat > environments/prod.env <<'EOF'
DATABASE_URL=postgres://craftsky:dev@localhost:5432/craftsky_dev?sslmode=disable
ALLOWED_ORIGINS=https://craftsky.social
EOF

# Restart Postgres if it's down from Step 6a or Step 8.
docker start craftsky-dev-pg >/dev/null 2>&1 || true
until docker exec craftsky-dev-pg pg_isready -U craftsky >/dev/null 2>&1; do sleep 0.3; done

/tmp/appview prod > /tmp/appview-prod.log 2>&1 &
APPVIEW_PROD_PID=$!
for i in {1..50}; do
  if curl -sS -o /dev/null http://localhost:8080/health 2>/dev/null; then break; fi
  sleep 0.2
done

# Assert: "deps initialised" is present, "log level" debug line is ABSENT.
grep -q '"msg":"deps initialised"' /tmp/appview-prod.log && echo "prod: deps initialised OK"
if grep -q '"msg":"log level"' /tmp/appview-prod.log; then
  echo "FAIL: prod emitted the dev-only debug line"
else
  echo "prod: debug line absent OK"
fi

kill -TERM $APPVIEW_PROD_PID
wait $APPVIEW_PROD_PID

# Clean up the temp prod.env so it isn't left behind.
rm environments/prod.env
```

Expected: both echo lines print (the `prod: ...` ones); the `FAIL:` line does NOT print.

- [ ] **Step 9: Final verification suite**

Run from `appview/`:
- [ ] `go vet ./...` — exits 0.
- [ ] `gofmt -l .` — exits 0, no output.
- [ ] `go test ./...` — all tests pass.
- [ ] `go build ./...` — exits 0.
- [ ] `git log --oneline -20` — shows Chunk 1 + 2 + 3 commits (roughly 20).

If everything passes, Chunk 3 is done — you have a running server that serves both endpoints, handles shutdown cleanly, and has a unit test suite covering every new component.

---

## Chunk 4: CLI binary

**Scope:** Everything under `cmd/cli/`. Cobra root with persistent `--env` flag, and the six subcommands: `ping`, `migrate`, `request`, `firehose replay`, `backfill`, `did-resolve` (stub). End of chunk: every acceptance criterion that involves the CLI passes.

**Shared pattern:** Each subcommand's `RunE` calls a helper that:
1. Reads the resolved `--env` flag value (from the persistent flag).
2. Calls `app.LoadConfig` and the appropriate `NewDevDeps` / `NewProdDeps`.
3. Ensures `cleanup()` is deferred.
4. Calls the subcommand-specific logic with `*app.Deps`.

That helper lives in `cmd/cli/deps.go` (a small file next to the subcommands, NOT a new `internal/` package — it's a private `package main` helper used by the cobra wiring).

### Task 4.1: Cobra root and shared dep loader

**Files:**
- Create: `appview/cmd/cli/main.go`
- Create: `appview/cmd/cli/deps.go`
- Modify: `appview/go.mod`, `appview/go.sum` (via `go get`)

No dedicated unit tests here — cobra wiring and the dep-loader helper are covered by each subcommand's own end-to-end smoke in Tasks 4.3 / 4.4 / 4.6 / 4.9.

- [ ] **Step 0: Add the cobra dependency**

Run: `go get github.com/spf13/cobra@latest`
Expected: `go: added github.com/spf13/cobra v1.x.x`.

- [ ] **Step 1: Create `cmd/cli/main.go`**

Create `appview/cmd/cli/main.go`:
```go
// Command cli is the Craftsky App View's companion CLI: ops tasks,
// smoke tests, and stubs for not-yet-implemented subsystems.
//
// Usage:
//   cli [subcommand] --env <dev|prod> [flags]
//
// --env is a persistent flag on the root command; every subcommand
// inherits it. Default is "dev" so local iteration just works.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

// envFlag is the value of --env for the current invocation, populated
// by cobra before any subcommand's RunE runs.
var envFlag string

var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "Craftsky App View ops and smoke-test CLI",
	Long: `cli is a companion tool to the appview server. It provides:
  * migrate — apply or inspect database migrations
  * ping    — check DB connectivity
  * request — hit the running server as the dev DID
  * firehose replay, backfill, did-resolve — stubs pending real impls`,
}

func main() {
	rootCmd.PersistentFlags().StringVar(&envFlag, "env", "dev", `environment: "dev" or "prod"`)
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints "Error: ..." itself; we just ensure non-zero exit.
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Create the shared dep loader**

Create `appview/cmd/cli/deps.go`:
```go
package main

import (
	"context"
	"fmt"

	"social.craftsky/appview/internal/app"
)

// loadDeps is the boilerplate every subcommand runs: parse the env flag,
// load config, build *app.Deps. Returns the deps and a cleanup function
// the caller must defer. On error, both are nil.
func loadDeps(ctx context.Context) (*app.Deps, func(), error) {
	env, err := app.ParseEnv(envFlag)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := app.LoadConfig(env, fmt.Sprintf("environments/%s.env", env))
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	switch env {
	case app.EnvDev:
		return app.NewDevDeps(ctx, cfg)
	case app.EnvProd:
		return app.NewProdDeps(ctx, cfg)
	default:
		return nil, nil, fmt.Errorf("unreachable: unknown env %q", env)
	}
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: exits 0. (Root-only CLI builds even with no subcommands registered; `cli --help` would print a usage summary but we'll wire actual subcommands next.)

- [ ] **Step 4: Commit**

```bash
git add cmd/cli/main.go cmd/cli/deps.go
git commit -m "feat(appview): add cli root cobra command and shared deps loader"
```

### Task 4.2: `cli ping` subcommand

**Files:**
- Create: `appview/cmd/cli/ping.go`

- [ ] **Step 1: Create `ping.go`**

Create `appview/cmd/cli/ping.go`:
```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping the configured Postgres and print pool stats",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		deps, cleanup, err := loadDeps(ctx)
		if err != nil {
			return err
		}
		defer cleanup()

		if err := deps.DB.Ping(ctx); err != nil {
			return fmt.Errorf("ping failed: %w", err)
		}
		s := deps.DB.Stat()
		fmt.Printf("ok: db up — acquired=%d idle=%d total=%d max=%d\n",
			s.AcquiredConns(), s.IdleConns(), s.TotalConns(), s.MaxConns())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
```

Note: `loadDeps` already pings on construction via `db.Connect`, so the second Ping is redundant in the happy path — but it's cheap, and it protects against a situation where the pool was built OK but the DB has since gone down between `Connect` and this call. Keeps the output honest.

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add cmd/cli/ping.go
git commit -m "feat(appview): add cli ping subcommand"
```

### Task 4.3: Smoke `cli ping`

**Files:** none modified.

- [ ] **Step 1: Build the CLI**

From `appview/`:
```bash
go build -o /tmp/cli ./cmd/cli
```

- [ ] **Step 2: Ping against a running Postgres**

Start Postgres (if not already up — see Task 3.11 Step 1). Then:
```bash
/tmp/cli ping --env dev
```
Expected: exits 0, prints one line like `ok: db up — acquired=0 idle=1 total=1 max=4`. The exact numbers may differ; `total >= 1` and `max > 0` is what to expect.

- [ ] **Step 3: Ping with DB down**

Stop Postgres (`docker stop craftsky-dev-pg`), then:
```bash
/tmp/cli ping --env dev
```
Expected: exits non-zero. Stderr contains the load/connect error (e.g. `Error: db connect: ping: ...`).

Restart Postgres for the next tasks: `docker start craftsky-dev-pg` (or re-run the docker run from Task 3.11).

- [ ] **Step 4: Nothing to commit**

Task is verification-only. Proceed.

### Task 4.4: `cli migrate` subcommand

**Files:**
- Create: `appview/cmd/cli/migrate.go`
- Create: `appview/cmd/cli/migrate_test.go`
- Modify: `appview/go.mod`, `appview/go.sum` (via `go get`)

golang-migrate/v4's `migrate.New` wants two URLs: a source URL (`file://...`) and a database URL. We use the file source and the Postgres driver, imported blank.

- [ ] **Step 0: Add the golang-migrate/v4 dependency**

Run: `go get github.com/golang-migrate/migrate/v4@latest`
Expected: `go: added github.com/golang-migrate/migrate/v4 v4.x.x`.

**Go directive bump (expected):** `golang-migrate/v4` requires a newer Go than the repo's `go 1.23`. The `go get` will rewrite `go.mod`'s `go` directive to whatever the library needs (likely `go 1.25.x`). This is the correct behaviour — let it stand, don't revert. `go build` / `go test` on the full tree will still succeed because none of the earlier code uses features that require < 1.23; we only ever move forward.

- [ ] **Step 1: Write failing test for the "empty migrations dir" case**

The empty-dir behaviour is observable in unit tests via a tempdir and an in-memory test setup. But golang-migrate's `file://` source requires a real directory. We create an empty tempdir and point at it.

Create `appview/cmd/cli/migrate_test.go`:
```go
package main

import (
	"testing"
)

func TestMigrateStatusEmptyDir(t *testing.T) {
	// The test passes a URL pointing at a closed port, proving that the
	// empty-dir short-circuit runs BEFORE any DB connection attempt —
	// which is the AC #9 contract.
	out, err := runMigrateStatus("postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1", t.TempDir())
	if err != nil {
		t.Fatalf("err = %v, want nil for empty migrations dir", err)
	}
	want := "no migrations applied (migrations directory is empty)"
	if out != want {
		t.Errorf("out = %q, want %q", out, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/cli/...`
Expected: compile error — `undefined: runMigrateStatus`.

- [ ] **Step 3: Implement `migrate.go`**

Create `appview/cmd/cli/migrate.go`:
```go
package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
)

// migrationsDir is the directory used by every subcommand in this family.
// Relative to the CLI's working directory.
const migrationsDir = "migrations"

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply, roll back, or inspect database migrations",
}

// migrateCfg loads only Config (no DB pool). Migrate subcommands use
// golang-migrate's own postgres driver rather than our pgxpool, so we
// don't want loadDeps's side effect of opening a second connection.
// This also means `cli migrate status --env dev` against an empty
// migrations/ directory exits 0 even when Postgres is unreachable —
// which is the AC #9 contract.
func migrateCfg() (string, error) {
	env, err := parseEnvFlag()
	if err != nil {
		return "", err
	}
	cfg, err := loadCfgLight(env)
	if err != nil {
		return "", err
	}
	return cfg.DatabaseURL, nil
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all unapplied migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isMigrationsDirEmpty(migrationsDir) {
			fmt.Println("no migrations applied (migrations directory is empty)")
			return nil
		}
		dbURL, err := migrateCfg()
		if err != nil {
			return err
		}
		return runMigrateUp(dbURL, migrationsDir)
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down [N]",
	Short: "Roll back N migrations (default 1)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		n := 1
		if len(args) == 1 {
			v, err := strconv.Atoi(args[0])
			if err != nil || v <= 0 {
				return fmt.Errorf("N must be a positive integer, got %q", args[0])
			}
			n = v
		}
		if isMigrationsDirEmpty(migrationsDir) {
			fmt.Println("no migrations applied (migrations directory is empty)")
			return nil
		}
		dbURL, err := migrateCfg()
		if err != nil {
			return err
		}
		return runMigrateDown(dbURL, migrationsDir, n)
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print current migration version and dirty flag",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isMigrationsDirEmpty(migrationsDir) {
			fmt.Println("no migrations applied (migrations directory is empty)")
			return nil
		}
		dbURL, err := migrateCfg()
		if err != nil {
			return err
		}
		out, err := runMigrateStatus(dbURL, migrationsDir)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

var migrateRedoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Roll back one migration and re-apply it",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isMigrationsDirEmpty(migrationsDir) {
			fmt.Println("no migrations applied (migrations directory is empty)")
			return nil
		}
		dbURL, err := migrateCfg()
		if err != nil {
			return err
		}
		return runMigrateRedo(dbURL, migrationsDir)
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd, migrateDownCmd, migrateStatusCmd, migrateRedoCmd)
	rootCmd.AddCommand(migrateCmd)
}

// isMigrationsDirEmpty returns true if dir contains no .sql files.
// Missing dir → treated as empty (callers get the same "no migrations"
// message rather than a confusing "no such file" error).
func isMigrationsDirEmpty(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			return false
		}
	}
	return true
}

// fileSourceURL produces the file:// URL golang-migrate expects. It
// resolves the dir to an absolute path so the URL is well-formed
// regardless of cwd quirks.
func fileSourceURL(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	u := &url.URL{Scheme: "file", Path: abs}
	return u.String(), nil
}

// newMigrate is the shared construction step. Callers pass control.
func newMigrate(databaseURL, dir string) (*migrate.Migrate, error) {
	src, err := fileSourceURL(dir)
	if err != nil {
		return nil, fmt.Errorf("build source url: %w", err)
	}
	m, err := migrate.New(src, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("migrate.New: %w", err)
	}
	return m, nil
}

func runMigrateUp(databaseURL, dir string) error {
	if isMigrationsDirEmpty(dir) {
		fmt.Println("no migrations applied (migrations directory is empty)")
		return nil
	}
	m, err := newMigrate(databaseURL, dir)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func runMigrateDown(databaseURL, dir string, n int) error {
	if isMigrationsDirEmpty(dir) {
		fmt.Println("no migrations applied (migrations directory is empty)")
		return nil
	}
	m, err := newMigrate(databaseURL, dir)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func runMigrateStatus(databaseURL, dir string) (string, error) {
	if isMigrationsDirEmpty(dir) {
		return "no migrations applied (migrations directory is empty)", nil
	}
	m, err := newMigrate(databaseURL, dir)
	if err != nil {
		return "", err
	}
	defer m.Close()
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return "no migrations applied", nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("version=%d dirty=%v", v, dirty), nil
}

func runMigrateRedo(databaseURL, dir string) error {
	if isMigrationsDirEmpty(dir) {
		fmt.Println("no migrations applied (migrations directory is empty)")
		return nil
	}
	m, err := newMigrate(databaseURL, dir)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Steps(-1); err != nil {
		return err
	}
	if err := m.Steps(1); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 4: Run to verify test passes**

Run: `go test ./cmd/cli/...`
Expected: `ok  social.craftsky/appview/cmd/cli`. The test passes because `isMigrationsDirEmpty` short-circuits `runMigrateStatus` before any DB connection is attempted.

- [ ] **Step 5: Full verification**

Run: `go build ./... && go test ./...`
Expected: both exit 0.

- [ ] **Step 6: Commit**

```bash
git add cmd/cli/migrate.go cmd/cli/migrate_test.go
git commit -m "feat(appview): add cli migrate up/down/status/redo subcommands"
```

### Task 4.5: Smoke `cli migrate` against the empty directory

**Files:** none.

- [ ] **Step 1: Rebuild the CLI**

```bash
go build -o /tmp/cli ./cmd/cli
```

- [ ] **Step 2: Status against empty migrations/**

Run: `/tmp/cli migrate status --env dev`
Expected: exits 0, prints `no migrations applied (migrations directory is empty)`.

- [ ] **Step 3: Up and down against empty migrations/**

Run: `/tmp/cli migrate up --env dev && /tmp/cli migrate down --env dev`
Expected: both exit 0 and print the same `no migrations applied` line. No changes to the DB.

- [ ] **Step 4: Down with a bad N**

Run: `/tmp/cli migrate down 0 --env dev`
Expected: exits non-zero. Stderr: `Error: N must be a positive integer, got "0"`.

- [ ] **Step 5: Status with DB down**

Stop Postgres (`docker stop craftsky-dev-pg`) and run:
```bash
/tmp/cli migrate status --env dev
```
Expected: exits 0, prints `no migrations applied (migrations directory is empty)`. This exercises the empty-dir short-circuit running *before* any DB connection — it's why the migrate subcommands don't call `loadDeps`.

Restart Postgres afterwards (`docker start craftsky-dev-pg`).

- [ ] **Step 6: Nothing to commit**

Task is verification-only. Proceed.

### Task 4.6: `cli request` subcommand

**Files:**
- Create: `appview/cmd/cli/request.go`
- Create: `appview/cmd/cli/request_test.go`

- [ ] **Step 1: Write failing test for happy path against httptest server**

Create `appview/cmd/cli/request_test.go`:
```go
package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDoRequest_200WritesStatusThenBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer dev" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer dev")
		}
		if got := r.Header.Get("X-Dev-DID"); got != "did:plc:test-caller" {
			t.Errorf("X-Dev-DID = %q, want %q", got, "did:plc:test-caller")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"hello":"world"}`)
	}))
	defer srv.Close()

	var out, errOut bytes.Buffer
	code, err := doRequest(requestArgs{
		Method:  "GET",
		Path:    "/x",
		BaseURL: srv.URL,
		DevDID:  "did:plc:test-caller",
		Out:     &out,
		ErrOut:  &errOut,
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	outStr := out.String()
	if !strings.HasPrefix(outStr, "200 OK\n") {
		t.Errorf("out should start with '200 OK\\n', got %q", outStr)
	}
	if !strings.Contains(outStr, `{"hello":"world"}`) {
		t.Errorf("out missing body: %q", outStr)
	}
	if errOut.Len() != 0 {
		t.Errorf("errOut should be empty on success, got %q", errOut.String())
	}
}

func TestDoRequest_4xxReturnsExit1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	var out, errOut bytes.Buffer
	code, err := doRequest(requestArgs{Method: "GET", Path: "/x", BaseURL: srv.URL, Out: &out, ErrOut: &errOut})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if code != 1 {
		t.Errorf("exit code = %d, want 1 for 401 response", code)
	}
}

func TestDoRequest_TransportErrorReturnsExit2(t *testing.T) {
	// Port 1 is reserved; connect will fail.
	var out, errOut bytes.Buffer
	code, err := doRequest(requestArgs{
		Method:  "GET",
		Path:    "/x",
		BaseURL: "http://127.0.0.1:1",
		Out:     &out,
		ErrOut:  &errOut,
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if code != 2 {
		t.Errorf("exit code = %d, want 2 for transport error", code)
	}
	if !strings.Contains(errOut.String(), "transport error:") {
		t.Errorf("errOut should contain 'transport error:', got %q", errOut.String())
	}
	if out.Len() != 0 {
		t.Errorf("out should be empty on transport error, got %q", out.String())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/cli/...`
Expected: compile error — `undefined: doRequest`, `undefined: requestArgs`.

- [ ] **Step 3: Replace `cmd/cli/deps.go` with the extended version**

`request.go` needs four new helpers (`parseEnvFlag`, `loadCfgLight`, `resolveBaseURL`, plus a `devEnvMarker` constant) that belong next to `loadDeps`. Do this BEFORE writing `request.go` so the compiler sees the symbols.

Replace `appview/cmd/cli/deps.go` entirely with:
```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"social.craftsky/appview/internal/app"
)

// loadDeps is the boilerplate a subcommand runs when it genuinely needs
// a DB pool: parse --env, load config, build *app.Deps.
// Subcommands that don't need DB access (request, migrate, stubs) use
// parseEnvFlag + loadCfgLight instead, avoiding a wasted connect.
func loadDeps(ctx context.Context) (*app.Deps, func(), error) {
	env, err := app.ParseEnv(envFlag)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := app.LoadConfig(env, fmt.Sprintf("environments/%s.env", env))
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	switch env {
	case app.EnvDev:
		return app.NewDevDeps(ctx, cfg)
	case app.EnvProd:
		return app.NewProdDeps(ctx, cfg)
	default:
		return nil, nil, fmt.Errorf("unreachable: unknown env %q", env)
	}
}

// devEnvMarker is re-exposed so other cmd/cli files don't need their
// own internal/app import just to compare against EnvDev.
var devEnvMarker = app.EnvDev

// parseEnvFlag returns app.Env for the current --env flag value.
func parseEnvFlag() (app.Env, error) {
	return app.ParseEnv(envFlag)
}

// loadCfgLight loads Config without building Deps (no DB connection).
func loadCfgLight(env app.Env) (app.Config, error) {
	return app.LoadConfig(env, "environments/"+string(env)+".env")
}

// resolveBaseURL picks the URL `cli request` should hit.
//   Dev  → http://localhost:8080
//   Prod → $APPVIEW_BASE_URL, which must start with https://.
// A permissive http:// in prod is disallowed to prevent sending dev DIDs
// or future real tokens over cleartext.
func resolveBaseURL(env app.Env) (string, error) {
	if env == app.EnvDev {
		return "http://localhost:8080", nil
	}
	v := os.Getenv("APPVIEW_BASE_URL")
	if v == "" {
		return "", errors.New("APPVIEW_BASE_URL must be set to hit prod")
	}
	if !strings.HasPrefix(v, "https://") {
		return "", errors.New("APPVIEW_BASE_URL must use https:// in prod")
	}
	return v, nil
}
```

- [ ] **Step 4: Create `cmd/cli/request.go`**

Create `appview/cmd/cli/request.go`:
```go
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// requestArgs is the testable surface of the request subcommand. Passing
// the dependencies in as a struct keeps doRequest pure — no reading from
// cobra flags or environment inside the function.
type requestArgs struct {
	Method  string
	Path    string
	BaseURL string    // e.g. "http://localhost:8080"
	DevDID  string    // empty disables the X-Dev-DID header
	Body    []byte    // nil = no body
	Headers []string  // extra headers as "Key: Value"
	Out     io.Writer // stdout in real runs; bytes.Buffer in tests
	ErrOut  io.Writer // stderr in real runs; bytes.Buffer in tests
}

// doRequest sends one HTTP request using args and writes the status line
// + body to args.Out. Returns (exitCode, internalErr). exitCode follows
// the contract:
//   0 — 2xx response
//   1 — 4xx/5xx response
//   2 — transport error (couldn't reach server)
// internalErr is non-nil only for bugs (bad args, write failures).
func doRequest(args requestArgs) (int, error) {
	body := io.Reader(nil)
	if args.Body != nil {
		body = bytes.NewReader(args.Body)
	}

	req, err := http.NewRequest(args.Method, args.BaseURL+args.Path, body)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	if args.DevDID != "" {
		req.Header.Set("Authorization", "Bearer dev")
		req.Header.Set("X-Dev-DID", args.DevDID)
	}
	for _, h := range args.Headers {
		k, v, ok := strings.Cut(h, ":")
		if !ok {
			return 0, fmt.Errorf("bad header %q (want 'Key: Value')", h)
		}
		req.Header.Add(strings.TrimSpace(k), strings.TrimSpace(v))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Transport errors are non-success outcomes; they go to stderr
		// so scripts piping stdout (`cli request ... | jq`) don't mix
		// body bytes with error diagnostics.
		fmt.Fprintf(args.ErrOut, "transport error: %s\n", err)
		return 2, nil
	}
	defer resp.Body.Close()

	// First line: "<code> <text>\n"
	if _, err := fmt.Fprintf(args.Out, "%d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode)); err != nil {
		return 0, err
	}
	// Body verbatim.
	if _, err := io.Copy(args.Out, resp.Body); err != nil {
		return 0, err
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return 0, nil
	default:
		return 1, nil
	}
}

var (
	reqHeaderFlag []string
	reqBodyFlag   string
	reqDIDFlag    string
	reqBaseURLEnv = "APPVIEW_BASE_URL"
)

var requestCmd = &cobra.Command{
	Use:   "request METHOD PATH",
	Short: "Send an HTTP request to the running appview server",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve env (we need it for auth defaults) but DON'T call
		// loadDeps — we don't want a DB connection for a smoke-test
		// request.
		env, err := parseEnvFlag()
		if err != nil {
			return err
		}
		cfg, err := loadCfgLight(env)
		if err != nil {
			return err
		}

		base, err := resolveBaseURL(env)
		if err != nil {
			return err
		}

		did := reqDIDFlag
		if did == "" && env == devEnvMarker {
			did = cfg.DevDID
		}

		var body []byte
		if reqBodyFlag != "" {
			body = []byte(reqBodyFlag)
		}

		code, err := doRequest(requestArgs{
			Method:  strings.ToUpper(args[0]),
			Path:    args[1],
			BaseURL: base,
			DevDID:  did,
			Body:    body,
			Headers: reqHeaderFlag,
			Out:     os.Stdout,
			ErrOut:  os.Stderr,
		})
		if err != nil {
			return err
		}
		// Cobra's RunE-to-exit-code mapping is binary (nil→0, non-nil→1).
		// This subcommand needs a tri-state exit: 0 = 2xx, 1 = 4xx/5xx,
		// 2 = transport error. os.Exit is safe here because requestCmd
		// holds no resources — no DB pool, no open files. Any deferred
		// cleanup in this goroutine has already run by the time
		// doRequest returns.
		if code != 0 {
			_ = os.Stdout.Sync()
			_ = os.Stderr.Sync()
			os.Exit(code)
		}
		return nil
	},
}

func init() {
	requestCmd.Flags().StringArrayVarP(&reqHeaderFlag, "header", "H", nil, "extra header 'Key: Value' (may repeat)")
	requestCmd.Flags().StringVarP(&reqBodyFlag, "data", "d", "", "request body")
	requestCmd.Flags().StringVar(&reqDIDFlag, "did", "", "override the dev DID sent in X-Dev-DID (dev env only)")
	rootCmd.AddCommand(requestCmd)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/cli/...`
Expected: all tests pass. The transport-error test exercises a real (refused) connection — `http.Client` errors cleanly.

- [ ] **Step 6: Commit**

```bash
git add cmd/cli/request.go cmd/cli/request_test.go cmd/cli/deps.go
git commit -m "feat(appview): add cli request subcommand with dev-DID auth"
```

### Task 4.7: Smoke `cli request` against the live server

**Files:** none.

Requires the server running on `:8080` (start it as in Task 3.11 Step 2).

- [ ] **Step 1: Rebuild CLI**

`go build -o /tmp/cli ./cmd/cli`

- [ ] **Step 2: Request /health**

Run: `/tmp/cli request GET /health --env dev`
Expected first line of stdout: `200 OK`. Body: `{"status":"ok"}`. Exit code 0.

- [ ] **Step 3: Request /whoami with default DID**

Run: `/tmp/cli request GET /whoami --env dev`
Expected: first line `200 OK`. Body: `{"did":"did:plc:craftsky-dev-user"}` (the dev env's default). Exit 0.

- [ ] **Step 4: Request /whoami overriding the DID**

Run: `/tmp/cli request GET /whoami --did did:plc:alice --env dev`
Expected: first line `200 OK`. Body: `{"did":"did:plc:alice"}`. Exit 0.

- [ ] **Step 5: Request a missing path**

Run: `/tmp/cli request GET /does-not-exist --env dev`
Expected: first line `404 Not Found`. Exit code non-zero (1).

- [ ] **Step 6: Request with server down**

Stop the server (`kill $APPVIEW_PID` if it's still running from earlier), then:
```bash
/tmp/cli request GET /health --env dev 2>/tmp/cli-err.log
rc=$?
cat /tmp/cli-err.log
echo "exit=$rc"
```
Capturing `$?` into `rc` immediately after the CLI invocation is deliberate — `$?` gets clobbered by the next command (`cat`), which is why `echo "exit=$?"` on its own would report `cat`'s exit code, not the CLI's.

Expected: stderr (captured in `/tmp/cli-err.log`) contains `transport error: ...connect: connection refused...`. Stdout is empty. The `echo` line prints `exit=2`.

Restart the server for the remaining tasks if needed.

- [ ] **Step 7: Nothing to commit.**

### Task 4.8: Stub subcommands — `firehose replay`, `backfill`, `did-resolve`

**Files:**
- Create: `appview/cmd/cli/firehose.go`
- Create: `appview/cmd/cli/backfill.go`
- Create: `appview/cmd/cli/did.go`

These three call into the stub `firehose.Subscriber` and `index.Indexer` and surface their "not yet implemented" errors. `did-resolve` has no real backing interface yet, so we keep its "not yet implemented" message inline here.

No dedicated unit tests — each subcommand's behaviour is a single passthrough call to an already-tested stub. Covered by the smoke run in Task 4.9.

- [ ] **Step 1: `firehose.go`**

Create `appview/cmd/cli/firehose.go`:
```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/firehose"
)

var firehoseSinceFlag string

var firehoseCmd = &cobra.Command{
	Use:   "firehose",
	Short: "Manage the firehose subscriber",
}

// Day one: stubs call firehose.NotImplemented directly — no DB pool, no
// loadDeps. This guarantees AC #11's "firehose: not yet implemented"
// error surfaces cleanly even if Postgres is down. When the real
// Subscriber impl lands, swap to `loadDeps(ctx)` → `deps.Firehose`.
var firehoseReplayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Re-index firehose events since --since (stub until real subscriber lands)",
	RunE: func(cmd *cobra.Command, args []string) error {
		since := time.Time{}
		if firehoseSinceFlag != "" {
			t, err := time.Parse(time.RFC3339, firehoseSinceFlag)
			if err != nil {
				// Try date-only as a convenience.
				t, err = time.Parse("2006-01-02", firehoseSinceFlag)
				if err != nil {
					return fmt.Errorf("--since %q: want RFC3339 or YYYY-MM-DD", firehoseSinceFlag)
				}
			}
			since = t
		}
		return firehose.NotImplemented{}.Replay(context.Background(), since)
	},
}

func init() {
	firehoseReplayCmd.Flags().StringVar(&firehoseSinceFlag, "since", "", `replay events from this time (RFC3339 or YYYY-MM-DD)`)
	firehoseCmd.AddCommand(firehoseReplayCmd)
	rootCmd.AddCommand(firehoseCmd)
}
```

- [ ] **Step 2: `backfill.go`**

Create `appview/cmd/cli/backfill.go`:
```go
package main

import (
	"context"

	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/index"
)

// Same rationale as firehose.go: calls NotImplemented directly so AC #12
// ("indexer: not yet implemented") surfaces without needing the DB.
var backfillCmd = &cobra.Command{
	Use:   "backfill DID",
	Short: "Re-index all records for a DID (stub until indexer lands)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return index.NotImplemented{}.Backfill(context.Background(), args[0])
	},
}

func init() {
	rootCmd.AddCommand(backfillCmd)
}
```

- [ ] **Step 3: `did.go`**

Create `appview/cmd/cli/did.go`:
```go
package main

import (
	"errors"

	"github.com/spf13/cobra"
)

var didResolveCmd = &cobra.Command{
	Use:   "did-resolve HANDLE",
	Short: "Resolve a handle to a DID (stub until identity resolver lands)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("did-resolve: not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(didResolveCmd)
}
```

- [ ] **Step 4: Build and test**

Run: `go build ./... && go test ./...`
Expected: both exit 0.

- [ ] **Step 5: Commit**

```bash
git add cmd/cli/firehose.go cmd/cli/backfill.go cmd/cli/did.go
git commit -m "feat(appview): add cli firehose/backfill/did-resolve stub subcommands"
```

### Task 4.9: Smoke the stub subcommands

**Files:** none.

- [ ] **Step 1: Rebuild CLI**

`go build -o /tmp/cli ./cmd/cli`

- [ ] **Step 2: `firehose replay`**

Run: `/tmp/cli firehose replay --env dev; echo "exit=$?"`
Expected: exits non-zero (1), stderr contains `firehose: not yet implemented`, echo prints `exit=1`.

- [ ] **Step 3: `backfill`**

Run: `/tmp/cli backfill did:plc:abc --env dev; echo "exit=$?"`
Expected: exits non-zero (1), stderr contains `indexer: not yet implemented`, echo prints `exit=1`.

- [ ] **Step 4: `did-resolve`**

Run: `/tmp/cli did-resolve alice.bsky.social --env dev; echo "exit=$?"`
Expected: exits non-zero (1), stderr contains `did-resolve: not yet implemented`, echo prints `exit=1`.

- [ ] **Step 5: `--since` validation**

Run: `/tmp/cli firehose replay --since not-a-date --env dev`
Expected: exits non-zero, stderr contains `--since "not-a-date": want RFC3339 or YYYY-MM-DD`.

- [ ] **Step 6: Nothing to commit.**

### Chunk 4 acceptance

Run from `appview/`:
- [ ] `go vet ./...` — exits 0.
- [ ] `gofmt -l .` — exits 0, no output.
- [ ] `go test ./...` — all tests pass.
- [ ] `go build ./...` — exits 0.
- [ ] `git log --oneline -30` — Chunks 1 + 2 + 3 + 4 commits visible (~25 commits total).

Cross-check against the spec's acceptance criteria (spec §"Acceptance Criteria"):
- **AC #5** (Task 4.3 Step 2 passes) — `cli ping` happy path.
- **AC #9** (Task 4.5 Step 2 passes) — `cli migrate status` on empty dir.
- **AC #10** (Task 4.7 Step 3 or Step 4 passes) — `cli request GET /whoami` with dev DID round-trip.
- **AC #11** (Task 4.9 Step 2 passes) — `cli firehose replay` stub error.
- **AC #12** (Task 4.9 Step 3 passes) — `cli backfill` stub error.

---

## Chunk 5: Supporting doc updates

**Scope:** Two checked-in docs — `AGENTS.md` and `appview/README.md` — that the spec flagged as needing updates now that chi is out and `cmd/cli/` is in. Both are pure text edits; no tests. This chunk is small (one commit per doc).

The spec's "In-Scope Touch-Ups" section explicitly calls out the AGENTS.md **line 29** (router) edit and the README rewrite. Task 5.1 Step 2 additionally pins AGENTS.md **line 31** to `golang-migrate/v4`; this is an intentional extension because the spec's "Module Dependencies" and "Migration tooling: golang-migrate/v4" sections pin the tool, and leaving AGENTS.md saying "golang-migrate or goose" would contradict what's actually shipping.

### Task 5.1: Update `AGENTS.md`

**Files:**
- Modify: `AGENTS.md` (at the repo root, not `appview/AGENTS.md`).

- [ ] **Step 1: Drop "chi or" from the Go conventions line**

Read `AGENTS.md` first to locate the line:
```bash
grep -n '^- \*\*Go:\*\*' AGENTS.md
```
Expected: one match at line 29 (may drift slightly if the file has been edited since the spec was written).

Apply this change to that line:
- Before: `- **Go:** standard ``gofmt``, ``slog`` for logging, ``sqlc`` for queries (write SQL, not ORMs), ``chi`` or stdlib ``net/http`` for routing, ``pgx`` for Postgres.`
- After: `- **Go:** standard ``gofmt``, ``slog`` for logging, ``sqlc`` for queries (write SQL, not ORMs), stdlib ``net/http`` for routing (Go 1.22+ method/path routing is enough), ``pgx`` for Postgres.`

- [ ] **Step 2: Pin migration tooling to `golang-migrate`**

On the `## SQL` line (around line 31), replace "`golang-migrate` or `goose`" with "`golang-migrate/v4`":
- Before: `- **SQL:** migrations in ``appview/migrations/`` via ``golang-migrate`` or ``goose``. Queries in ``appview/queries/`` consumed by ``sqlc``.`
- After: `- **SQL:** migrations in ``appview/migrations/`` via ``golang-migrate/v4`` (wrapped by ``appview/cmd/cli migrate``). Queries in ``appview/queries/`` consumed by ``sqlc``.`

- [ ] **Step 3: Verify diff**

Run: `git diff AGENTS.md`
Expected: two lines changed, no whitespace-only noise.

- [ ] **Step 4: Commit**

```bash
git add AGENTS.md
git commit -m "docs(agents): pin router to stdlib and migrations to golang-migrate/v4"
```

### Task 5.2: Update `appview/README.md`

**Files:**
- Modify: `appview/README.md`.

- [ ] **Step 1: Replace the README**

Replace `appview/README.md` entirely with:
```markdown
# appview

The Craftsky App View — a Go service that subscribes to the atproto Relay firehose, indexes Craftsky records into Postgres, and serves a JSON/HTTP API to the Flutter client.

Also acts as the **Token Mediating Backend (TMB)** for OAuth with user PDSes.

## Layout

```
appview/
├── cmd/
│   ├── appview/             # server binary (main + NewServer)
│   └── cli/                 # ops & smoke-test CLI (cobra)
├── internal/
│   ├── app/                 # Config, Deps, NewDevDeps/NewProdDeps
│   ├── auth/                # AuthService interface + mock / not-implemented impls
│   ├── db/                  # pgxpool connection wrapper
│   ├── middleware/          # Logging, CORS, Authenticated
│   ├── routes/              # HTTP route registration
│   ├── api/                 # HTTP handler factories
│   ├── firehose/            # Subscriber interface (stub on day one)
│   ├── index/               # Indexer interface (stub on day one)
│   └── models/              # reserved for sqlc-generated types
├── environments/
│   ├── dev.env              # checked in; no secrets
│   └── prod.env.example     # template; real prod.env is gitignored
├── migrations/              # SQL files consumed by golang-migrate/v4
└── queries/                 # SQL files consumed by sqlc
```

## Binaries

### `cmd/appview` — the HTTP server

```
appview dev    # loads environments/dev.env, debug logging, mock auth
appview prod   # loads environments/prod.env, info logging, (future) real OAuth
```

### `cmd/cli` — ops and smoke-test CLI

```
cli ping --env dev              # pings the DB, prints pool stats
cli migrate up|down|status|redo # wraps golang-migrate/v4
cli request GET /whoami --env dev  # hits the running server as the dev DID
cli firehose replay --since 2026-04-01 --env dev  # stub until the subscriber lands
cli backfill did:plc:abc --env dev                 # stub until the indexer lands
cli did-resolve alice.bsky.social --env dev       # stub until the identity resolver lands
```

Exit codes for `cli request`:
- `0` — 2xx response from the server
- `1` — 4xx/5xx response
- `2` — transport error (couldn't reach server)

## Key Dependencies

- [`github.com/bluesky-social/indigo`](https://github.com/bluesky-social/indigo) — atproto SDK (firehose, XRPC, OAuth) — to be adopted once the real subscriber/OAuth land
- [`pgx/v5`](https://github.com/jackc/pgx) — Postgres driver + pool
- [`sqlc`](https://sqlc.dev) — SQL → Go codegen — to be adopted once first queries land
- [`cobra`](https://github.com/spf13/cobra) — CLI framework
- [`golang-migrate/v4`](https://github.com/golang-migrate/migrate) — migrations, wrapped by `cmd/cli`
- [`godotenv`](https://github.com/joho/godotenv) — env file loader
- [`uuid`](https://github.com/google/uuid) — per-request run IDs
- `slog` — standard library logging
- `net/http` — standard library router (Go 1.22+ method/path routing is sufficient)

## Development

Run Postgres:
```
docker run --rm -d --name craftsky-dev-pg \
  -p 5432:5432 \
  -e POSTGRES_USER=craftsky \
  -e POSTGRES_PASSWORD=dev \
  -e POSTGRES_DB=craftsky_dev \
  postgres:16
```

Run the server:
```
go run ./cmd/appview dev
```

Run the CLI (from `appview/`):
```
go run ./cmd/cli ping --env dev
go run ./cmd/cli request GET /health --env dev
```

Run tests and formatters:
```
go test ./...
go vet ./...
gofmt -l .
```

A future commit will add `make` targets (`make dev`, `make migrate`, `make generate`, `make test`).

## Why Go

See the reference doc's "Tech Stack" section. TL;DR: contributor accessibility, ecosystem maturity, single static binary deploys, alignment with the atproto Go ecosystem (`indigo`).
```

- [ ] **Step 2: Verify it reads sensibly**

Run: `git diff appview/README.md | head -80`
Expected: the diff is large (we rewrote most of it). Spot-check that the `Layout` tree still matches the actual filesystem (it should, since we just built what it describes in Chunks 1-4).

- [ ] **Step 3: Commit**

```bash
git add appview/README.md
git commit -m "docs(appview): update README to reflect stdlib router, cmd/cli, and new layout"
```

### Chunk 5 acceptance

- [ ] `git log --oneline -2` shows the two doc commits.
- [ ] `grep -nw 'chi' AGENTS.md appview/README.md` returns nothing (word-boundary match: chi is gone; defends against false positives like "chicken" in some future edit).
- [ ] `grep -n 'golang-migrate' AGENTS.md appview/README.md` returns hits in both.
- [ ] The `appview/` filesystem matches the Layout section of `appview/README.md`.

---

## Final acceptance — whole spec AC checklist

At this point, every spec acceptance criterion should be verifiable. Run through them in one sitting (Postgres up; the server can be started as needed per AC):

| AC | Verified by |
|----|-------------|
| #1 | Task 3.11 Step 2 — server starts dev, debug line in `/tmp/appview.log` |
| #2 | Task 3.10 Step 3 (missing-var exit) + Task 3.11 Step 8a (prod valid-start + debug-line-absent) |
| #3 | Task 3.10 Steps 1–2 |
| #4 | Task 3.11 Step 3 (200) + Task 3.11 Step 6a (503 with DB down) |
| #5 | Task 3.11 Step 4 |
| #6 | Task 3.11 Step 5 |
| #7 | Task 3.11 Step 7 (ordered shutdown log lines) |
| #8 | Task 4.3 Step 2 |
| #9 | Task 4.5 Step 2 and Step 5 |
| #10 | Task 4.7 Step 3 |
| #11 | Task 4.9 Step 2 |
| #12 | Task 4.9 Step 3 |
| #13 | Chunk 5 acceptance (`go vet` + `gofmt -l`) combined with the per-chunk gates |
| #14 | Every chunk's `go build ./...` step |

If any AC isn't green, return to the relevant task and debug before declaring the scaffold done.



