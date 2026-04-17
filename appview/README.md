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
│   ├── tap/                 # WebSocket-with-acks consumer for the Tap sidecar
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
cli tap status --env dev           # prints tap connection state (exit 0 connected, 1 disconnected, 2 transport)
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

Local development runs through Docker Compose, driven by the `justfile` at the repo root. See the root `README.md` for a getting-started walkthrough; the compose stack (`postgres`, `migrate`, `tap`, `tap-bootstrap`, `appview`) is brought up with:

```
just dev      # foreground
just dev-d    # detached
just down     # stop (volumes preserved)
```

Common recipes:

```
just test         # go test -race ./... on the host against the compose Postgres
just fmt          # gofmt -w . && go vet ./... on the host
just ping         # cli ping inside the appview container
just tap-status   # cli tap status inside the appview container
just psql         # psql shell against the dev database
just migrate up   # wraps golang-migrate/v4 via the CLI
```

Tests run on the **host** (the appview image has no Go toolchain), so Go must be installed locally and `just dev-d` must already be running — the integration tests connect to the compose Postgres at `localhost:5432` via the `TEST_DATABASE_URL` the recipe sets.

## Why Go

See the reference doc's "Tech Stack" section. TL;DR: contributor accessibility, ecosystem maturity, single static binary deploys, alignment with the atproto Go ecosystem (`indigo`).
