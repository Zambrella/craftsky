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
