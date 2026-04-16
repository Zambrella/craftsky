# App View Server Scaffold — Design

**Date:** 2026-04-16
**Status:** Approved, ready for implementation planning
**Scope:** `appview/` (Go service)

## Summary

Stand up the minimum viable shape of the Craftsky App View binary and a
companion CLI tool, modelled on the dependency-injection and lifecycle
patterns used in `stash_hub/server`. The scaffold compiles and runs end-to-end
(`GET /health`, `GET /whoami`, graceful shutdown, env-switched config) without
yet implementing the firehose subscriber, indexer, or real atproto OAuth.

Structural parity with stash_hub only — no Firebase, no GCP Cloud
Logging/Cloud Tasks, no Dockerfile or Cloud Run scaffolding.

## Motivation

`appview/` currently has empty `internal/*` packages and a one-line `main.go`.
Before firehose / indexer / OAuth work can start, the server needs:

- A single, consistent shape for loading config, wiring dependencies, and
  swapping real/mock implementations per environment.
- A CLI entry point for smoke testing and iteration against the server and its
  database.
- A middleware stack (CORS, logging, auth) that handlers can rely on from day
  one.

The stash_hub pattern (`run()` + typed `NewServer` constructor + inline
dev/prod swaps) is a proven baseline. This spec adapts it to Craftsky's stack
(Postgres + pgx + sqlc, stdlib `net/http`, atproto OAuth) and fixes the one
readability smell — stash_hub's `NewServer` takes 11 positional args — by
introducing a shared `*app.Deps` struct.

## Non-Goals

- **No firehose subscription logic.** Stub package + interface only.
- **No indexer logic.** Stub package + interface only.
- **No real atproto OAuth implementation.** A `NotImplementedAuthService`
  returns an error in prod mode until real OAuth lands.
- **No sqlc output, queries, or migrations.** Directories stay empty.
  `internal/models/` contains only a `doc.go` declaring `package models`.
- **No Dockerfile, `build.sh`, or CI configuration.** Go code only.
- **No observability service.** `slog` is sufficient until there's a concrete
  need.
- **No `service/` aggregate package** (stash_hub's `MagicInputService`
  equivalent). Add one if/when a reason exists.

## In-Scope Touch-Ups to Supporting Docs

Implementation of this spec will also:

- Update `appview/README.md` to reflect: (a) stdlib `net/http` instead of
  `chi`, (b) the new `cmd/cli/` binary, (c) the updated `internal/` layout.
- Update `AGENTS.md` line 29 so "`chi` or stdlib `net/http`" becomes just
  "stdlib `net/http` (Go 1.22+ method/path routing is sufficient)".
- Add `environments/prod.env` to `.gitignore` now, before real secrets land.
  Check in a `environments/prod.env.example` template so contributors know
  the required variable names.

## Design Decisions

### Environment switching: positional arg
The server binary takes `dev` or `prod` as `os.Args[1]` (matches stash_hub).
Missing/invalid arg exits with a clear error. Rationale: self-documenting on
the command line, forces every invocation to be explicit, and provides a
single hook point in code for dev/prod divergence (mock auth, log level, CORS
permissiveness).

The CLI uses `cobra` with a persistent `--env` flag instead — subcommand is
the primary UX concern, env is configuration.

### Dev/prod service wiring: factory functions
`app.NewDevDeps(ctx, cfg)` and `app.NewProdDeps(ctx, cfg)` return a fully
wired `*app.Deps` + a `func()` cleanup closure. `run()` calls one based on the
parsed env. Rationale: keeps `run()` short (~50 lines vs stash_hub's ~300),
collocates all per-env variations in one file, makes it trivial for the CLI
to reuse the same wiring.

### `Deps` location: `internal/app/deps.go`
Shared by both `cmd/appview` and `cmd/cli`. Neither binary's `main` package
can be imported, so the wiring has to live in a library package. Chosen over
duplicating factories in each binary because drift between them is a known
failure mode ("works on server, broken in migrate").

### `Deps` usage: assembly-only
`*app.Deps` is passed into `NewServer`, `routes.AddRoutes`, middleware
constructors, and CLI subcommand handlers. It is **never** passed into
individual HTTP handlers — each handler factory takes only the specific
dependencies it uses (`api.HealthHandler(db)`, not `api.HealthHandler(deps)`).
This prevents handlers from silently growing dependencies.

Middleware constructors (e.g. `middleware.Logging(deps.Logger)`) are
considered "assembly" for this rule: they run at startup to produce a wrapped
handler, not at request time.

### Router: stdlib `net/http`
Go 1.22+'s `ServeMux` supports method routing (`"GET /path"`) and path
parameters (`"/path/{id}"`) natively. No `chi` dependency.

### Migration tooling: `golang-migrate/migrate/v4`
Used programmatically (not as a shelled-out CLI) so the CLI subcommand reuses
`deps.Config.DatabaseURL` and doesn't depend on a separately installed
binary. Migrations directory URI is `file://appview/migrations` when running
from the repo root, resolved relative to the CLI's working directory.

CLI subcommands and their `golang-migrate` equivalents:
- `cli migrate up --env dev` → `m.Up()`.
- `cli migrate down [N] --env dev` → `m.Steps(-N)`. Default `N` is `1` when
  argv omits it. No "down all" escape hatch; rolling back the whole schema
  must be done by repeated invocation on purpose.
- `cli migrate status --env dev` → `m.Version()`; prints current version and
  dirty flag, or "no migrations applied" when `ErrNilVersion`.
- `cli migrate redo --env dev` → `m.Steps(-1)` then `m.Steps(1)`.

When `migrations/` is empty (day one), `cli migrate status` prints "no
migrations applied (migrations directory is empty)" and exits 0. `up` / `down`
/ `redo` in that state also exit 0 with the same message. The implementation
should create the `schema_migrations` tracking table only on first real
migration run, not on `status`.

### Auth interface: `(ctx, token) -> (did, error)`
`AuthService.Authenticate` is transport-agnostic. The `Authenticated`
middleware handles HTTP-specific concerns (bearer header parsing, `X-Dev-DID`
override) and calls the service. This keeps `AuthService` usable from both
HTTP and CLI contexts.

The concrete mechanism for the `X-Dev-DID` override:
1. Middleware reads the `X-Dev-DID` header. If present, it calls
   `auth.WithDevDID(ctx, devDID)` which stores the value at an unexported
   context key inside the `auth` package.
2. Middleware then calls `authService.Authenticate(ctx, token)`.
3. `MockAuthService.Authenticate` calls `auth.DevDIDFromContext(ctx)`. If a
   value is present, it returns it; otherwise it returns `m.DefaultDID`.
4. `NotImplementedAuthService.Authenticate` ignores the context and returns
   an error.

The context key lives in the `auth` package (not `middleware`) so `auth`
implementations can read it without importing `middleware` — that would be a
layering cycle. The helpers `auth.WithDevDID` and `auth.DevDIDFromContext`
are exported; the key itself is unexported.

### Dev auth: `MockAuthService` with `X-Dev-DID` header override
Mock always authenticates. If the request carries `X-Dev-DID`, that DID is
returned; otherwise falls back to `cfg.DevDID`. Enables local multi-user
testing without standing up real OAuth. Costs ~3 lines over a single-fixed-
DID mock.

### CLI framework: `cobra`
Justified over stdlib `flag` by the expected command tree size (migrate,
ping, request, firehose replay, backfill, did-resolve — ~6 commands at day
N). Consistent help output and persistent-flag inheritance matter more than
the dependency cost.

## Module Dependencies

The implementation adds exactly these modules to `appview/go.mod`:

| Module | Version pin | Used by |
|---|---|---|
| `github.com/jackc/pgx/v5` | `v5.x` (latest) | `internal/db`, `internal/app` |
| `github.com/spf13/cobra` | `v1.x` (latest) | `cmd/cli` only |
| `github.com/joho/godotenv` | `v1.x` (latest) | `internal/app/config.go` |
| `github.com/golang-migrate/migrate/v4` | `v4.x` (latest) | `cmd/cli/migrate.go` only; includes the `postgres` driver and `file` source |
| `github.com/google/uuid` | `v1.x` (latest) | `internal/middleware/logging.go` |

Exact minor versions are chosen by `go get` at implementation time. No other
modules are added in this pass.

## Architecture

```
appview/
├── cmd/
│   ├── appview/               # server binary (package main)
│   │   ├── main.go            # signal-aware run(), env parsing, http.Server lifecycle
│   │   └── server.go          # NewServer(ctx, *app.Deps) http.Handler (main.go owns *http.Server)
│   └── cli/                   # ops / smoke-test binary (package main)
│       ├── main.go            # cobra root command, --env persistent flag
│       ├── migrate.go         # wraps golang-migrate against deps.Config.DatabaseURL
│       ├── ping.go            # db pool ping + pool stats
│       ├── request.go         # authenticated request to running server
│       ├── firehose.go        # stub subcommand(s)
│       ├── backfill.go        # stub subcommand
│       └── did.go             # stub subcommand (did-resolve)
├── internal/
│   ├── app/                   # package app
│   │   ├── config.go          # Config struct, Env type, LoadConfig, ParseEnv
│   │   └── deps.go            # Deps struct, NewDevDeps, NewProdDeps
│   ├── auth/                  # package auth
│   │   ├── service.go         # AuthService interface, WithDevDID, DevDIDFromContext
│   │   ├── oauth.go           # NotImplementedAuthService (prod stub)
│   │   └── mock.go            # MockAuthService (dev)
│   ├── middleware/            # package middleware
│   │   ├── auth.go            # Authenticated(authService, next), GetDID(ctx)
│   │   ├── cors.go            # CORSConfig, CORS middleware
│   │   └── logging.go         # Logging middleware (runID injection), GetRunID
│   ├── routes/                # package routes
│   │   └── routes.go          # AddRoutes(ctx, mux, *app.Deps) — ctx is startup scope, for any route-time validation; handlers use r.Context()
│   ├── api/                   # package api
│   │   ├── health.go          # HealthHandler(db) — public
│   │   └── whoami.go          # WhoAmIHandler() — authenticated
│   ├── db/                    # package db
│   │   └── db.go              # Connect(ctx, url) (*pgxpool.Pool, error)
│   ├── firehose/              # package firehose
│   │   └── subscriber.go      # Subscriber interface + NotImplemented impl
│   ├── index/                 # package index
│   │   └── indexer.go         # Indexer interface + NotImplemented impl
│   └── models/                # package models
│       └── doc.go             # only contents: `package models` (reserved for sqlc)
├── environments/
│   ├── dev.env                # checked in; no secrets
│   ├── prod.env               # gitignored
│   └── prod.env.example       # checked-in template listing required keys
├── migrations/                # unchanged, empty, tracked by a `.gitkeep`
└── queries/                   # unchanged, empty, tracked by a `.gitkeep`
```

### Dependency flow

```
main.go (cmd/appview)
  ├─ signal.NotifyContext(ctx, Interrupt, SIGTERM)  ← wraps the entire run()
  ├─ parses env from os.Args[1]
  ├─ app.LoadConfig(env)                  → Config (reads environments/<env>.env)
  ├─ app.NewDevDeps / NewProdDeps(ctx, cfg)
  │    └─ returns (*Deps, cleanup, error)
  ├─ defer cleanup()                      ← runs even if NewServer / Listen fails
  ├─ httpServer := &http.Server{ Handler: NewServer(ctx, deps) }
  │    └─ NewServer(ctx, deps):
  │       └─ routes.AddRoutes(ctx, mux, deps)
  │            └─ api.FooHandler(deps.X)  ← individual deps only
  │       └─ middleware.CORS(deps.Config.AllowedOrigins)
  │       └─ middleware.Logging(deps.Logger)
  └─ shutdown on ctx.Done():
       1. httpServer.Shutdown(10s timeout)   ← waits for in-flight requests
       2. (on return from run()) defer cleanup() fires
          → closes DB pool, flushes anything else

main.go (cmd/cli)
  ├─ cobra root with persistent --env flag
  ├─ each subcommand Run:
  │    ├─ app.LoadConfig(env)
  │    ├─ app.NewDevDeps / NewProdDeps
  │    ├─ defer cleanup()
  │    └─ executes its op using the subset of Deps it needs
```

### `Config` shape

```go
type Env string

const (
    EnvDev  Env = "dev"
    EnvProd Env = "prod"
)

type Config struct {
    Env            Env
    DatabaseURL    string   // required in all envs
    AllowedOrigins []string // required in all envs (prod: explicit list; dev: may include "*")
    DevDID         string   // required in dev; ignored in prod
}
```

**Required keys by env:**

| Env var | dev | prod | Notes |
|---|---|---|---|
| `DATABASE_URL` | required | required | Postgres connection string. |
| `ALLOWED_ORIGINS` | required | required | Comma-separated list. Dev may be `*`. |
| `CRAFTSKY_DEV_DID` | required | ignored | Default DID for `MockAuthService`. |

`LoadConfig` reads `environments/<env>.env` via `godotenv`, then falls through
to `os.Getenv` (so env vars set outside the file still win). Missing required
keys cause `LoadConfig` to return an error naming the specific missing key.

`ParseEnv(s string) (Env, error)` is a trivial string → `Env` converter used
by both `main.go` (for `os.Args[1]`) and the CLI's `--env` flag PreRun hook.

### `Deps` shape

```go
type Deps struct {
    Config      Config
    Logger      *slog.Logger
    DB          *pgxpool.Pool
    AuthService auth.AuthService

    // Stub interfaces with NotImplemented impls on day one.
    // Present so the Deps shape is stable and CLI stubs compile.
    Firehose firehose.Subscriber
    Indexer  index.Indexer
}
```

### Stub interfaces

Day-one stubs define just enough surface for the CLI subcommands to compile
and return clear errors:

```go
// internal/firehose/subscriber.go
package firehose

import "context"

type Subscriber interface {
    // Replay re-indexes firehose events since the given timestamp.
    // Returns a descriptive error from NotImplemented until real impl lands.
    Replay(ctx context.Context, since time.Time) error
}

type NotImplemented struct{}

func (NotImplemented) Replay(ctx context.Context, since time.Time) error {
    return errors.New("firehose: not yet implemented")
}
```

```go
// internal/index/indexer.go
package index

import "context"

type Indexer interface {
    // Backfill re-indexes all records for the given DID from its PDS.
    Backfill(ctx context.Context, did string) error
}

type NotImplemented struct{}

func (NotImplemented) Backfill(ctx context.Context, did string) error {
    return errors.New("indexer: not yet implemented")
}
```

The CLI surfaces these errors to stdout and exits 1. The error message and
exit code are the spec's stability contract; no panic, no silent success.

### Middleware stack (outside-in)

```
request
  ↓ Logging   (assigns runID to ctx, logs method + path at Info)
  ↓ CORS      (origin check, preflight handling)
  ↓ mux       (method + path routing)
  ↓ Authenticated  (only on authenticated routes, per-route wrap)
  ↓ handler
```

`Logging` and `CORS` are applied globally in `NewServer`. `Authenticated` is
applied per-route inside `routes.AddRoutes` (so `/health` stays public).

`Logging` runs outside `Authenticated` deliberately: 401 responses are logged
with a `run_id` so failed auth attempts can be correlated with client errors.
Do not move `Logging` inside the auth check.

**Logging middleware specifics** (port of stash_hub's equivalent):
- Generates `uuid.New().String()` per request, stores at `runIDKey` in ctx.
- Logs `slog.Info("Request received", "method", r.Method, "path", r.URL.Path,
  "run_id", runID)` at entry.
- Exports `middleware.GetRunID(ctx) string` for handlers that want to log
  with it.
- Uses `deps.Logger` (not `slog.Default()`), so test instances can capture
  output.

**Auth middleware specifics:**
- Exports `middleware.GetDID(ctx) (string, bool)`.
- Uses an unexported `didKey` of type `contextKey` to avoid collisions.
- On success, stores the DID at `didKey` and calls `next.ServeHTTP` with the
  augmented request.
- On failure (missing/malformed Authorization header, empty token, or
  `authService.Authenticate` error), writes `http.Error(w, "Unauthorized",
  401)` and returns.

### `HealthHandler` specifics

- `GET /health` only. Other methods fall through to `http.NotFoundHandler`.
- Calls `db.Ping(r.Context())` with a 2-second per-request timeout (derived
  via `context.WithTimeout`).
- On success: `200` with `Content-Type: application/json` and body
  `{"status":"ok"}`.
- On ping error: `503` with `Content-Type: text/plain` and body
  `db unreachable`. The underlying error is logged at `slog.LevelError` via
  `deps.Logger` but not returned to the client.

### `/whoami` in prod

The `/whoami` route is wired behind `Authenticated` regardless of environment.
In prod, `NotImplementedAuthService.Authenticate` always errors, so `/whoami`
returns 401 until real OAuth lands. This is intentional (keeps the auth
middleware path exercised in prod) and has no dedicated acceptance criterion.

### Auth flow

```
HTTP request with `Authorization: Bearer <token>`
  ↓ middleware.Authenticated
  │    ├─ parses bearer header (401 on missing/malformed)
  │    ├─ if X-Dev-DID header present: ctx = auth.WithDevDID(ctx, devDID)
  │    └─ calls authService.Authenticate(ctx, token)
  │         ├─ MockAuthService: auth.DevDIDFromContext(ctx) or cfg.DevDID
  │         └─ NotImplementedAuthService: returns errors.New("atproto OAuth not implemented yet")
  ↓ on success: ctx = context.WithValue(ctx, didKey, did)
  ↓ handler calls middleware.GetDID(ctx) to read it
```

### Startup log lines

At the end of `NewDevDeps` / `NewProdDeps`, before returning, emit exactly
these log lines at the specified levels via the freshly constructed
`deps.Logger`:

- `slog.Debug("log level", "level", "debug")` — dev only; its presence in
  stdout is how AC #1 confirms debug logging is actually active.
- `slog.Info("deps initialised", "env", cfg.Env)` — both envs. Used by AC
  #2 as the signal that prod started.

### Lifecycle & shutdown semantics

- **Signal scope.** `signal.NotifyContext(ctx, Interrupt, SIGTERM)` wraps the
  entire `run()` body, so Ctrl-C during `NewDevDeps` (e.g. slow DB connect)
  cancels cleanly.
- **Cleanup idempotency.** The `cleanup` closure returned by the factories
  must be safe to call multiple times (no-op on second call). Pattern: wrap
  the inner `sync.Once` call.
- **`defer cleanup()` placement.** In `run()`, called immediately after deps
  init succeeds. If `ListenAndServe` errors (port in use), `cleanup()` still
  fires on function return.
- **Shutdown ordering.** All log lines below are at `slog.LevelInfo` via
  `deps.Logger`.
  1. `ctx.Done()` fires.
  2. Log `"shutdown: received signal"`.
  3. `httpServer.Shutdown(shutdownCtx)` with a 10-second timeout. This blocks
     until in-flight requests complete or the timeout fires. On return, log
     `"shutdown: http server stopped"`.
  4. `run()` returns; deferred `cleanup()` fires, closing the DB pool and
     logging `"shutdown: db pool closed"`.
  5. `loggingClient.Close()` (if we had one — not day one).
- The DB pool stays open during `httpServer.Shutdown` so in-flight requests
  can complete their DB calls. This is why cleanup is deferred, not called
  inline before `Shutdown`.

### CLI `request` subcommand specifics

- Syntax: `cli request [--env dev] [-d BODY] [-H "Key: Value"...] METHOD PATH`.
  `METHOD` and `PATH` are positional. `BODY` is optional; `-H` may be repeated.
- Base URL is hardcoded: `http://localhost:8080` in dev, `https://<prod-url>` in
  prod. Prod URL comes from a new optional `APPVIEW_BASE_URL` env var; if
  unset, `request` errors in prod mode ("set APPVIEW_BASE_URL to hit prod").
- Authentication:
  - Dev: sets `Authorization: Bearer dev` and `X-Dev-DID: <cfg.DevDID>`.
    Override with `--did` flag to test multi-user.
  - Prod: does not add auth headers (because there's no token mechanism the
    CLI has access to). Hits public endpoints only. If the endpoint requires
    auth, server returns 401 and CLI prints it.
- Output: first stdout line is `<status-code> <status-text>` (e.g. `200 OK`);
  subsequent lines are the response body verbatim. Exits 0 on 2xx, 1 on
  4xx/5xx, 2 on transport error (connection refused etc).
- HTTP client is constructed ad-hoc inside the subcommand, not injected via
  `Deps`. 30-second timeout.

## Acceptance Criteria

1. `go run ./cmd/appview dev` starts successfully and listens on `:8080`.
   `deps.Logger` is constructed with `slog.LevelDebug`; a startup-time
   `slog.Debug("log level", "level", "debug")` line confirms the level is
   active in stdout (visible only if level is debug, so its presence is the
   evidence).
2. `go run ./cmd/appview prod` with a valid `environments/prod.env` starts
   successfully; `deps.Logger` is constructed with `slog.LevelInfo` and the
   equivalent `slog.Debug` line from AC #1 is NOT emitted (evidencing the
   prod level). With a missing required var, exits non-zero with an error
   naming the missing variable.
3. `go run ./cmd/appview` (no arg) or with an invalid arg exits non-zero with
   a clear error message.
4. `curl http://localhost:8080/health` returns 200 `{"status":"ok"}` when
   Postgres is reachable; 503 when it is not.
5. `curl -H "Authorization: Bearer anything" \
     -H "X-Dev-DID: did:plc:test123" \
     http://localhost:8080/whoami` returns 200 `{"did":"did:plc:test123"}` in
   dev mode.
6. `curl http://localhost:8080/whoami` (no Authorization header) returns 401.
7. SIGINT/SIGTERM triggers graceful shutdown within 10s; in-flight requests
   complete before the process exits. Stdout contains, in order, the lines
   `"shutdown: received signal"`, `"shutdown: http server stopped"`, and
   `"shutdown: db pool closed"` before the process exits 0.
8. `go run ./cmd/cli ping --env dev` exits 0 when DB is up, non-zero when
   down; prints pool stats (acquired, idle, total) on success.
9. `go run ./cmd/cli migrate status --env dev` prints "no migrations applied
   (migrations directory is empty)" and exits 0 when `migrations/` has no
   `.sql` files.
10. `go run ./cmd/cli request GET /whoami --env dev` (with the server running)
    prints status `200 OK` and body `{"did":"<cfg.DevDID>"}`, exits 0.
11. `go run ./cmd/cli firehose replay --env dev` exits 1 with stderr line
    `firehose: not yet implemented`.
12. `go run ./cmd/cli backfill did:plc:abc --env dev` exits 1 with stderr
    line `indexer: not yet implemented`.
13. `go vet ./...` and `gofmt -l .` (both from `appview/`) produce no output.
14. `go build ./...` succeeds from `appview/`.

## Risks / Open Questions

- **`environments/prod.env` gitignore is in scope but non-obvious** — an
  implementer could forget to add it. The "In-Scope Touch-Ups" section calls
  it out explicitly, and the gitignore change must ship in the same commit
  as `environments/prod.env.example` to prevent accidental secret commits.
- **`pgxpool` vs standalone `pgx.Conn`** — going with `pgxpool` because both
  the server and the CLI's `ping` benefit from pool lifecycle semantics, and
  sqlc-generated code works with either.
- **Cobra is a CLI-only dependency** — imported only from `cmd/cli`. Server
  binary builds without it.
- **`cli request` cannot hit prod with auth** — stated above. Acceptable for a
  smoke-test tool; the workaround is `curl` with a real bearer token. Revisit
  when real OAuth lands and the CLI can acquire its own token.
- **`golang-migrate` directory URI is path-sensitive** — the CLI resolves
  `migrations/` relative to its working directory. Document in the CLI's
  `migrate` subcommand `--help` that it must be run from the `appview/`
  directory, or make the path configurable via `--migrations-dir`.

## References

- `stash_hub/server/cmd/server/main.go` — source pattern being adapted.
- `stash_hub/server/internal/middleware/` — `Authenticated`, `CORSMiddleware`,
  `LoggingMiddleware` being ported with minor adjustments.
- `craftsky/AGENTS.md` — architectural rules (Postgres + sqlc, atproto OAuth,
  stdlib or chi).
- `craftsky/appview/README.md` — target layout this spec refines.
