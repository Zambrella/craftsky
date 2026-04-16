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
- **No sqlc output, queries, or migrations.** Directories stay empty,
  `internal/models/` reserved.
- **No Dockerfile, `build.sh`, or CI configuration.** Go code only.
- **No observability service.** `slog` is sufficient until there's a concrete
  need.
- **No `service/` aggregate package** (stash_hub's `MagicInputService`
  equivalent). Add one if/when a reason exists.

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
`*app.Deps` is passed into `NewServer`, `routes.AddRoutes`, and CLI
subcommand handlers. It is **never** passed into individual HTTP handlers —
each handler factory takes only the specific dependencies it uses
(`api.HealthHandler(db)`, not `api.HealthHandler(deps)`). This prevents
handlers from silently growing dependencies.

### Router: stdlib `net/http`
Go 1.22+'s `ServeMux` supports method routing (`"GET /path"`) and path
parameters (`"/path/{id}"`) natively. No `chi` dependency. (Updates the
"`chi` or stdlib" note in `AGENTS.md` — we pick stdlib.)

### Auth interface: `(ctx, token) -> (did, error)`
`AuthService.Authenticate` is transport-agnostic. The `Authenticated`
middleware handles HTTP-specific concerns (bearer header parsing, `X-Dev-DID`
override) and calls the service. This keeps `AuthService` usable from both
HTTP and CLI contexts.

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

## Architecture

```
appview/
├── cmd/
│   ├── appview/               # server binary
│   │   ├── main.go            # signal-aware run(), env parsing, http.Server lifecycle
│   │   └── server.go          # NewServer(ctx, *app.Deps) http.Handler
│   └── cli/                   # ops / smoke-test binary
│       ├── main.go            # cobra root command, --env persistent flag
│       ├── migrate.go         # wraps golang-migrate against deps.Config.DatabaseURL
│       ├── ping.go            # db pool ping + pool stats
│       ├── request.go         # authenticated request to running server
│       ├── firehose.go        # stub subcommand(s)
│       ├── backfill.go        # stub subcommand
│       └── did.go             # stub subcommand (did-resolve)
├── internal/
│   ├── app/
│   │   ├── config.go          # Config struct, Env type, LoadConfig, ParseEnv
│   │   └── deps.go            # Deps struct, NewDevDeps, NewProdDeps
│   ├── auth/
│   │   ├── service.go         # AuthService interface, context helpers
│   │   ├── oauth.go           # NotImplementedAuthService (prod stub)
│   │   └── mock.go            # MockAuthService (dev)
│   ├── middleware/
│   │   ├── auth.go            # Authenticated(authService, next)
│   │   ├── cors.go            # CORSConfig, CORS middleware
│   │   └── logging.go         # Logging middleware (runID injection)
│   ├── routes/
│   │   └── routes.go          # AddRoutes(ctx, mux, *app.Deps)
│   ├── api/
│   │   ├── health.go          # HealthHandler(db) — public
│   │   └── whoami.go          # WhoAmIHandler() — authenticated
│   ├── db/
│   │   └── db.go              # Connect(ctx, url) (*pgxpool.Pool, error)
│   ├── firehose/
│   │   └── subscriber.go      # Subscriber interface + NotImplemented impl
│   ├── index/
│   │   └── indexer.go         # Indexer interface + NotImplemented impl
│   └── models/                # reserved for sqlc output (empty file)
├── environments/
│   ├── dev.env                # DATABASE_URL, CRAFTSKY_DEV_DID
│   └── prod.env               # DATABASE_URL, (future: PDS_OAUTH_CLIENT_ID…)
├── migrations/                # unchanged, still empty
└── queries/                   # unchanged, still empty
```

### Dependency flow

```
main.go (cmd/appview)
  ├─ parses env from os.Args[1]
  ├─ app.LoadConfig(env)                  → Config (reads environments/<env>.env)
  ├─ app.NewDevDeps / NewProdDeps(ctx, cfg)
  │    └─ returns (*Deps, cleanup, error)
  ├─ NewServer(ctx, deps)
  │    └─ routes.AddRoutes(ctx, mux, deps)
  │         └─ api.FooHandler(deps.X)    ← individual deps only
  │    └─ middleware.CORS(deps.Config.AllowedOrigins)
  │    └─ middleware.Logging(deps.Logger)
  └─ http.Server.ListenAndServe + graceful shutdown on ctx.Done()

main.go (cmd/cli)
  ├─ cobra root with persistent --env flag
  ├─ each subcommand Run:
  │    ├─ app.LoadConfig(env)
  │    ├─ app.NewDevDeps / NewProdDeps
  │    └─ executes its op using the subset of Deps it needs
```

### `Config` shape (day one)

```go
type Env string

const (
    EnvDev  Env = "dev"
    EnvProd Env = "prod"
)

type Config struct {
    Env            Env
    DatabaseURL    string
    AllowedOrigins []string
    DevDID         string  // dev only; default DID for MockAuthService
    // Reserved for later commits:
    //   PDSOAuthClientID, PDSOAuthClientSecret, RelayURL, etc.
}
```

`LoadConfig(env)` reads `environments/<env>.env` via `godotenv` and validates
required fields. Missing required fields → `LoadConfig` returns an error
naming the missing key. `DevDID` is required when `env == EnvDev`, otherwise
optional.

### `Deps` shape (day one)

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

### Middleware stack (outside-in)

```
request
  ↓ Logging   (assigns runID, logs method + path)
  ↓ CORS      (origin check, preflight handling)
  ↓ mux       (method + path routing)
  ↓ Authenticated  (only on authenticated routes, per-route wrap)
  ↓ handler
```

`Logging` and `CORS` are applied globally in `NewServer`. `Authenticated` is
applied per-route inside `routes.AddRoutes` (so `/health` stays public).

### Auth flow

```
HTTP request with `Authorization: Bearer <token>`
  ↓ middleware.Authenticated
  │    ├─ parses bearer header (401 on missing/malformed)
  │    ├─ dev only: if X-Dev-DID header present, injects into ctx
  │    └─ calls authService.Authenticate(ctx, token)
  │         ├─ MockAuthService: returns X-Dev-DID from ctx or cfg.DevDID
  │         └─ NotImplementedAuthService: returns error (prod stub)
  ↓ on success: ctx has didKey populated
  ↓ handler calls middleware.GetDID(ctx) to read it
```

## Acceptance Criteria

1. `go run ./cmd/appview dev` starts successfully, logs at debug level,
   listens on `:8080`.
2. `go run ./cmd/appview prod` with a valid `environments/prod.env` starts
   successfully; with a missing required var, exits non-zero with an error
   naming the missing variable.
3. `go run ./cmd/appview` (no arg) or with an invalid arg exits non-zero with
   a clear error message.
4. `curl http://localhost:8080/health` returns 200 `{"status":"ok"}` when
   Postgres is reachable; 503 when it is not.
5. `curl -H "Authorization: Bearer anything" \
     -H "X-Dev-DID: did:plc:test123" \
     http://localhost:8080/whoami` returns 200 `{"did":"did:plc:test123"}` in
   dev mode.
6. `curl http://localhost:8080/whoami` (no auth header) returns 401.
7. SIGINT/SIGTERM triggers graceful shutdown within 10s; in-flight requests
   complete before the process exits.
8. `go run ./cmd/cli ping --env dev` exits 0 when DB is up, non-zero when
   down; prints pool stats on success.
9. `go run ./cmd/cli migrate status --env dev` reports migration state
   against the dev DB.
10. `go run ./cmd/cli request GET /whoami --env dev` hits the running server
    and prints the DID returned.
11. `go run ./cmd/cli firehose replay --env dev` exits non-zero with a "not
    yet implemented" error (not a panic, not a compile failure).
12. `go run ./cmd/cli backfill did:plc:abc --env dev` exits non-zero with a
    "not yet implemented" error.
13. `go vet ./...` and `gofmt -l .` produce no output.

## Risks / Open Questions

- **`environments/*.env` files contain no secrets on day one** (just a
  `DATABASE_URL` pointing at localhost and a dev DID). When real OAuth lands,
  `prod.env` will hold client secrets — we need a `.gitignore` entry for
  `environments/prod.env` before that commit, or to move prod secrets to an
  external source (Secret Manager, etc.). Flag at the time, not now.
- **`pgxpool` vs standalone `pgx.Conn`** — going with `pgxpool` because both
  the server and the CLI's `ping` benefit from pool lifecycle semantics, and
  sqlc-generated code works with either. No expected issue.
- **Cobra adds a dependency** the server doesn't need. Acceptable — it's a
  CLI-only dependency imported only from `cmd/cli`.
- **`AGENTS.md` says "`chi` or stdlib `net/http`"** — this spec picks stdlib.
  If a strong case for chi emerges later (middleware composition, sub-router
  ergonomics), revisit. Not blocking.

## References

- `stash_hub/server/cmd/server/main.go` — source pattern being adapted.
- `stash_hub/server/internal/middleware/` — `Authenticated`, `CORSMiddleware`,
  `LoggingMiddleware` being ported with minor adjustments.
- `craftsky/AGENTS.md` — architectural rules (Postgres + sqlc, atproto OAuth,
  stdlib or chi).
- `craftsky/appview/README.md` — target layout this spec refines.
