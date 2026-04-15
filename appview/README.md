# appview

The Craftsky App View — a Go service that subscribes to the atproto Relay firehose, indexes Craftsky records into Postgres, and serves a JSON/HTTP API to the Flutter client.

Also acts as the **Token Mediating Backend (TMB)** for OAuth with user PDSes.

## Layout

```
appview/
├── cmd/
│   └── appview/        # main entrypoint
├── internal/
│   ├── firehose/       # relay subscription & filtering
│   ├── index/          # writing records into postgres
│   ├── api/            # HTTP handlers (chi)
│   ├── auth/           # OAuth TMB + session management
│   └── models/         # sqlc-generated types
├── migrations/         # SQL migration files (golang-migrate / goose)
└── queries/            # SQL files consumed by sqlc
```

## Key Dependencies

- [`github.com/bluesky-social/indigo`](https://github.com/bluesky-social/indigo) — atproto SDK (firehose, XRPC, OAuth)
- [`pgx`](https://github.com/jackc/pgx) — Postgres driver
- [`sqlc`](https://sqlc.dev) — SQL → Go codegen
- [`chi`](https://github.com/go-chi/chi) — HTTP router
- `slog` — standard library logging

## Development

Not yet set up. Planned `make dev`, `make migrate`, `make generate`, `make test` targets.

Run Postgres via `docker compose up postgres`, then the Go binary with `air` or `go run ./cmd/appview`.

## Why Go

See the reference doc's "Tech Stack" section. TL;DR: contributor accessibility, ecosystem maturity, single static binary deploys, alignment with the atproto Go ecosystem (`indigo`).
