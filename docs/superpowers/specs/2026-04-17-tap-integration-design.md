# Tap integration — design

**Date:** 2026-04-17
**Status:** proposed
**Supersedes pieces of:** [2026-04-16-appview-server-scaffold-design.md](./2026-04-16-appview-server-scaffold-design.md) (the `firehose` package and `cli firehose replay` / `cli backfill` subcommands are removed; see §2 and §3)

## Summary

Replace the day-one `firehose.NotImplemented` stub in the appview scaffold with a real [Tap](https://github.com/bluesky-social/indigo/tree/main/cmd/tap) WebSocket client. Run Tap as a sibling container in a top-level `docker-compose.yml` alongside Postgres and the appview. Index `app.bsky.feed.post` events from four allowlisted Bluesky accounts into a throwaway `bluesky_posts_sample` table to prove the end-to-end pipe from Relay → Tap → appview → Postgres.

This is deliberately the **minimum viable Tap integration**. No `social.craftsky.*` lexicon or indexer is introduced here; that's a follow-up spec. The sample table exists solely to validate the wire and is deleted when the first Craftsky indexer lands.

## Goals

1. Replace the firehose stub with a real Tap-backed event consumer.
2. Ship a `docker compose`-based dev workflow so a cold contributor runs one command.
3. Prove events flow end-to-end by writing one row per live/backfilled `app.bsky.feed.post` into a sample Postgres table.
4. Keep the scaffold's `Deps` abstraction intact: the consumer is just another dependency injected into `Deps`, not a new global.

## Non-goals

- Any `social.craftsky.*` lexicon, record type, or indexer. Follow-up spec.
- Optimistic-write reconciliation (write to PDS → pending row → reconcile on Tap event). Deferred until a real record type lands.
- OAuth, DPoP, session management, Token Mediating Backend. `NotImplementedAuthService` remains as-is.
- Per-NSID dispatch registry / multi-indexer architecture. The sample indexer guards on `Event.Collection == "app.bsky.feed.post"` and drops everything else.
- GitHub Actions / CI. Separate task.
- Production deployment (SSL termination, `compose.prod.yaml`, secrets). `prod.env.example` stays theoretical.
- Jetstream fallback. One delivery path only.
- Admin CLI for Tap ops (adding/removing tracked DIDs at runtime). The compose env-var list covers day-one needs.
- Host-run `go run ./cmd/appview dev`. The canonical dev workflow is `just dev` (= `docker compose up --build`).

## 1. Architecture & topology

```
wss://bsky.network (Relay)
          │
          ▼
    ┌──────────┐   SQLite (named volume: tapdata)
    │   tap    │◄──────── /data/tap.sqlite
    │container │
    └────┬─────┘
         │ WebSocket-with-acks (ws://tap:2480/subscribe)
         ▼
    ┌────────────┐      ┌──────────────┐
    │  appview   │─────▶│   postgres   │
    │ container  │  SQL │   (pgdata)   │
    └────────────┘      └──────────────┘
         ▲
         │ HTTP :8080
         │
     (Flutter app / cli)
```

Three containers, each with one job:

- **`tap`** — pinned `ghcr.io/bluesky-social/indigo/tap:<pinned-tag>`. Connects to `wss://bsky.network`, tracks four allowlisted DIDs, filters for `app.bsky.feed.post`, stores cursor and repo metadata in SQLite on the `tapdata` named volume. Exposes its WS on `:2480` inside the compose network only — no host port.
- **`postgres`** — `postgres:16`. Named volume `pgdata`. Exposes `:5432` to the host for `just psql`.
- **`appview`** — built from `./appview/Dockerfile`. Depends on both via healthchecks. Exposes `:8080` to the host.

### Dev workflow

One canonical entry point: `just dev` → `docker compose up --build`. Foreground by default (terminal is the live log stream; Ctrl-C stops the stack). A detached variant (`just dev-d` + `just logs`) is available for when you want another terminal.

There is no `go run ./cmd/appview dev` path. All `cli` subcommands run inside the container (`just ping`, `just migrate up`, `just tap-status`), and Postgres is reachable from the host at `localhost:5432` for external tools like `psql` or a GUI.

### Log format & levels

- **`appview`** — `APPVIEW_ENV=dev` triggers `slog.LevelDebug` with the existing JSON handler. Unchanged from the scaffold.
- **`tap`** — info level (upstream default). Bumped to debug only if diagnosing connection issues.
- **`postgres`** — default (warnings and above). Noisy query logging not enabled.

Interleaved JSON + plain-text in foreground mode is acceptable; no pretty-printer dependency is introduced. If this becomes annoying in practice, switching appview to `slog.NewTextHandler` in dev is a one-line change for a future PR.

## 2. Go-side package changes

### Deletions

- **`internal/firehose/`** — entire package deleted (both `subscriber.go` and `subscriber_test.go`). No shim, no re-export.
- **`cmd/cli/firehose.go`** — deleted along with its `firehose replay` subcommand registration.
- **`cmd/cli/backfill.go`** — deleted along with its `backfill` subcommand registration. Backfill is Tap's job now.
- **`index.Indexer.Backfill`** method — removed from `internal/index/indexer.go`. Replaced, see below.

### Additions

**`internal/tap/` package.**

```go
// Package tap consumes atproto events from a Tap sidecar over WebSocket.
package tap

type Event struct {
    URI        string          // at://did/collection/rkey
    CID        string
    DID        string
    Collection string          // e.g. "app.bsky.feed.post"
    Rkey       string
    Action     string          // "create" | "update" | "delete"
    Record     json.RawMessage // opaque JSON; indexer decodes per-collection
    Live       bool            // false during backfill, true for steady-state
    Seq        uint64          // Tap's per-event sequence, used for acking
}

type Consumer interface {
    Run(ctx context.Context) error
    State() ConnState
}

type ConnState struct {
    Connected        bool
    LastEventAt      time.Time
    LastError        string
    ReconnectAttempt int
}
```

Raw-JSON `Record` keeps the consumer collection-agnostic. Per-collection typing happens inside whichever `Indexer` handles the event.

**Concrete type `WSConsumer`** (in `internal/tap/consumer.go`). Fields: WS URL, indexer handle, `state` struct guarded by a mutex, reconnect policy. `Run(ctx)`:

1. Dial the WS.
2. On connect: set `state.Connected = true`, reset `ReconnectAttempt`.
3. For each frame received: decode → `indexer.Handle(ctx, event)` → on success, send ack frame with `event.Seq`; on indexer error, log and continue without acking (Tap will redeliver).
4. On any WS-level error: set `state.Connected = false`, log the error, sleep `min(1s << ReconnectAttempt, 30s)`, increment `ReconnectAttempt`, retry.
5. On `ctx.Done()`: close WS cleanly, return `ctx.Err()`.

`State()` returns a snapshot by copying the mutex-guarded struct. Called by the health handler.

**`Indexer` interface reshape** (in `internal/index/indexer.go`):

```go
type Indexer interface {
    Handle(ctx context.Context, ev tap.Event) error
}
```

First implementation: **`BlueskyPostsSample`** (new file `internal/index/bluesky_posts_sample.go`). Holds a `*pgxpool.Pool`. `Handle` switches on `ev.Collection`:

- `app.bsky.feed.post` with action `create` or `update`: `INSERT ... ON CONFLICT (uri) DO UPDATE SET cid = EXCLUDED.cid, record = EXCLUDED.record`. Idempotency is natural on `(uri, cid)`.
- `app.bsky.feed.post` with action `delete`: `DELETE FROM bluesky_posts_sample WHERE uri = $1`. Missing rows are not an error.
- Any other collection: log at debug and return `nil`. (We shouldn't see these given Tap's filter, but be defensive.)

File carries a prominent header comment marking both the table and the indexer as throwaway, to be deleted when the first `social.craftsky.*` indexer lands.

### Wiring

**`internal/app/deps.go`.** `Deps` gains a `Consumer tap.Consumer` field in place of `Firehose firehose.Subscriber`, and `Indexer index.Indexer` is re-typed to match the new interface. `newDeps` constructs `tap.NewWSConsumer(cfg.TapWSURL, indexer)` and `index.NewBlueskyPostsSample(pool)`. The consumer goroutine is **not** started inside `newDeps` — construction stays pure.

**`cmd/appview/server.go`** starts the consumer goroutine alongside the HTTP server:

```go
go func() {
    if err := deps.Consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
        deps.Logger.Error("tap consumer exited", slog.Any("err", err))
    }
}()
```

Shutdown cancels the root context; both the HTTP server and the consumer drain.

### Health endpoint

New handler `internal/api/health.go`, registered on `GET /healthz` (unauthenticated). Response:

```json
{
  "status": "ok",
  "db": "ok",
  "tap": {
    "connected": true,
    "last_event_at": "2026-04-17T14:23:11Z",
    "reconnect_attempt": 0,
    "last_error": ""
  }
}
```

Rules:

- `db` field is `"ok"` if `pool.Ping(ctx)` succeeds, else `"error"`.
- `tap` block is populated from `consumer.State()`.
- Overall `status`: `"ok"` when both `db == "ok"` and `tap.connected == true`; `"degraded"` otherwise. HTTP 200 in both cases — reads don't require Tap to work. HTTP 503 only if the handler itself panics.

### Config additions

In `internal/app/config.go`, three new keys:

- `TAP_WS_URL` (required) — e.g. `ws://tap:2480/subscribe`
- `TAP_ACK_TIMEOUT` (default `10s`) — matches Tap's default ack window
- `TAP_RECONNECT_MAX` (default `30s`) — cap for exponential backoff

Added to `environments/dev.env` and `environments/prod.env.example`.

## 3. Docker Compose, Dockerfile, migrations, CLI

### `docker-compose.yml` (repo root)

```yaml
services:
  postgres:
    image: postgres:16
    restart: unless-stopped
    environment:
      POSTGRES_USER: craftsky
      POSTGRES_PASSWORD: dev
      POSTGRES_DB: craftsky_dev
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U craftsky -d craftsky_dev"]
      interval: 2s
      timeout: 2s
      retries: 15

  tap:
    # Tap is under active development. Bump this tag intentionally, not on :latest.
    # See README's "Updating Tap" section for the bump process.
    image: ghcr.io/bluesky-social/indigo/tap:<pinned-tag>
    restart: unless-stopped
    environment:
      TAP_RELAY_HOST: wss://bsky.network
      # Four allowlisted accounts for dev. DIDs are resolved from handles at
      # implementation time: @bsky.app, @jay.bsky.team, @dougtodd.dev,
      # @eurosky.social.
      TAP_TRACKED_DIDS: "<did:plc:...>,<did:plc:...>,<did:plc:...>,<did:plc:...>"
      TAP_COLLECTION_FILTERS: "app.bsky.feed.post"
    volumes:
      - tapdata:/data
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:2480/healthz || exit 1"]
      interval: 5s
      timeout: 3s
      retries: 10

  appview:
    build:
      context: ./appview
      dockerfile: Dockerfile
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      tap:
        condition: service_healthy
    environment:
      APPVIEW_ENV: dev
      DATABASE_URL: postgres://craftsky:dev@postgres:5432/craftsky_dev?sslmode=disable
      TAP_WS_URL: ws://tap:2480/subscribe
      ALLOWED_ORIGINS: "*"
      CRAFTSKY_DEV_DID: did:plc:craftsky-dev-user
    ports:
      - "8080:8080"

volumes:
  pgdata:
  tapdata:
```

**Implementation-time confirmations** (do not block design approval; document what was found when implementing):

1. Exact Tap env var names (`TAP_TRACKED_DIDS`, `TAP_COLLECTION_FILTERS`, healthcheck path). If Tap only accepts a DID allowlist via its admin HTTP API, add a one-shot `tap-bootstrap` init container that POSTs the DID list after Tap is healthy.
2. Confirm Tap exposes a healthcheck endpoint on `:2480`. If not, replace the healthcheck with a TCP probe on that port.
3. Pinned tag — pick the latest stable Tap release at the time of implementation; record the version in the compose file comment.
4. Resolve the four handles to DIDs via `did-resolve` or a one-off lookup; paste into `TAP_TRACKED_DIDS`.

### `appview/Dockerfile`

Alpine base (friendlier to poke at than distroless during early dev):

```dockerfile
# syntax=docker/dockerfile:1.7
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/appview ./cmd/appview \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cli     ./cmd/cli

FROM alpine:3.19
RUN addgroup -S app && adduser -S -G app app \
 && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/appview /app/appview
COPY --from=build /out/cli     /app/cli
COPY migrations /app/migrations
USER app
EXPOSE 8080
ENTRYPOINT ["/app/appview"]
CMD ["dev"]
```

Two binaries in one image lets `docker compose exec appview /app/cli …` work without a second build or image.

### Migration

New `appview/migrations/<NNNN>_bluesky_posts_sample.up.sql` (next sequential prefix), with matching `.down.sql`:

```sql
-- SAMPLE TABLE — throwaway. Delete this migration (up + down) and every
-- reference to it when the first social.craftsky.* indexer lands.
-- See docs/superpowers/specs/2026-04-17-tap-integration-design.md.

CREATE TABLE bluesky_posts_sample (
    uri        TEXT PRIMARY KEY,
    cid        TEXT NOT NULL,
    did        TEXT NOT NULL,
    rkey       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    record     JSONB NOT NULL
);
CREATE INDEX bluesky_posts_sample_did_idx ON bluesky_posts_sample (did);
```

Down-migration drops table and index.

### CLI changes

- **Remove** `cmd/cli/firehose.go`, `cmd/cli/backfill.go` and their subcommand registrations in `cmd/cli/main.go`.
- **Add** `cmd/cli/tap.go` with `cli tap status`. It hits `http://localhost:8080/healthz` (or `APPVIEW_URL` from env) and pretty-prints the `tap` block:

  ```
  $ just tap-status
  connected:         true
  last_event_at:     2026-04-17T14:23:11Z (8s ago)
  reconnect_attempt: 0
  ```

  Exit codes: 0 when connected, 1 when disconnected, 2 on transport error. Mirrors the existing convention from `cli request`.
- **Keep** `ping`, `migrate`, `request`, `did-resolve` unchanged.

### `justfile` (repo root)

```
set dotenv-load := false

default:
    @just --list

dev:
    docker compose up --build

dev-d:
    docker compose up -d --build

down:
    docker compose down

logs:
    docker compose logs -f

migrate *ARGS:
    docker compose exec appview /app/cli migrate {{ARGS}}

ping:
    docker compose exec appview /app/cli ping

tap-status:
    docker compose exec appview /app/cli tap status

psql:
    docker compose exec postgres psql -U craftsky craftsky_dev

test:
    cd appview && go test ./...

fmt:
    cd appview && gofmt -w . && go vet ./...
```

`just` is a new dependency for contributors. Install via `brew install just` or equivalent; flagged in `README.md` and `AGENTS.md`.

### Documentation updates

- **`appview/README.md`** — remove the `make` section referenced in the scaffold spec (it never existed yet, just a promise); replace with the `just` table. Remove `cli firehose replay` and `cli backfill` from the subcommand list; add `cli tap status`.
- **`AGENTS.md`** — add a "Dev workflow" line pointing at `just dev`; update the repo layout to mention top-level `docker-compose.yml` and `justfile`.
- **`README.md`** (repo root) — add a "Getting started" section with the two-step flow: install `just` and Docker, run `just dev`, wait for healthchecks, open `http://localhost:8080/healthz`.

## 4. Acceptance criteria, tests, risks

### Acceptance criteria

The change lands when all are true:

1. `just dev` (= `docker compose up --build`) brings all three containers to healthy state. `docker compose ps` shows `healthy` for postgres, tap, and appview.
2. `curl localhost:8080/healthz` returns HTTP 200 with `tap.connected: true` within 60 seconds of a cold `just dev`.
3. `just migrate up` applies the `bluesky_posts_sample` migration cleanly; running it again is a no-op (no errors, no duplicate schema).
4. After `just dev` has been up for at least 2 minutes — or after any of the four tracked accounts has posted to Bluesky during that window — `just psql` followed by `SELECT count(*) FROM bluesky_posts_sample;` returns a non-zero row count. Row shape matches migration types; `record` column round-trips as valid JSON when `SELECT record FROM bluesky_posts_sample LIMIT 1;` is piped to `jq`.
5. **Idempotency.** `docker compose stop appview` mid-stream, then `docker compose start appview`. No duplicate-key errors in logs; `count(*)` resumes climbing (not resetting, not stalling).
6. **Degradation.** `docker compose stop tap`. Within 10 seconds, `curl localhost:8080/healthz` shows `tap.connected: false` and `status: "degraded"`, HTTP 200. Appview logs show reconnect attempts with backoff. `docker compose start tap`. Within 30 seconds `/healthz` flips back to `connected: true`, `status: "ok"`.
7. `just test` — all tests pass (existing plus the new ones in §Test strategy).
8. `just fmt` produces no changes; `go vet ./...` is clean.
9. `just tap-status` inside the container prints connection state and exits 0 when connected.
10. A cold contributor can clone the repo, install `just` + Docker, run `just dev`, and reach AC #2 using only the README. The README gives them every command they need.

### Test strategy

- **`internal/tap/consumer_test.go`** — `httptest.Server` with a fake WS endpoint. Cases:
  - Happy path: three events sent, three `Handle` calls, three acks received.
  - Indexer error: `Handle` returns error → no ack sent for that event, connection stays open, next event still flows.
  - WS close mid-stream: consumer reconnects; `State().Connected` transitions `true → false → true`; `ReconnectAttempt` increments.
  - Context cancel: `Run` returns `context.Canceled`, WS closed with normal-closure code.
- **`internal/index/bluesky_posts_sample_test.go`** — **real Postgres** (per AGENTS.md; per the project feedback memory "integration tests must hit a real database, not mocks"). Each test gets its own schema (`CREATE SCHEMA test_<uuid>; SET search_path = test_<uuid>`) for isolation; torn down on completion. Cases:
  - Create → row present with expected fields.
  - Create same URI twice → single row (idempotent via `ON CONFLICT`).
  - Update → existing row's `cid` and `record` refreshed, `created_at` unchanged.
  - Delete → row absent.
  - Delete non-existent URI → no error.
  - Unknown collection → no row, no error.
- **`internal/api/health_test.go`** — stub `Consumer` returning canned `State()` values; assert JSON response shape and overall `status` transitions (`ok` when both db and tap ok; `degraded` otherwise).
- **Postgres for tests.** Preferred: `docker compose run --rm appview go test ./...` (uses the same Postgres as dev). Also supported: a developer-supplied `TEST_DATABASE_URL` env var for host-side runs. Both documented in README.
- **No Go-based end-to-end test.** The acceptance criteria above are the E2E — running `just dev` against the real Relay proves the whole wire. Recreating that in a test would mean mocking Tap and the Relay, which duplicates effort without testing anything real.

### Risks

- **Tap is young software.** Upstream may change env-var names, event schema, or delivery semantics. Mitigated by the pinned image tag and by the `WSConsumer`'s malformed-payload path (decode error → log → no ack → redelivery → loop visible in logs). A catastrophic schema change shows up as an acceptance-criteria failure, not silent data corruption.
- **Handle → DID resolution.** The four Bluesky handles are resolved to DIDs at implementation time. If any handle fails to resolve (deleted account, typo), implementation fills in a working DID from the remaining three and flags it. Dev should not rely on a specific single DID.
- **Tap env-var naming.** The compose `TAP_TRACKED_DIDS` name is provisional. If upstream uses a different name or requires admin-API bootstrapping, implementation adopts whatever's correct without requiring a design re-review. Flagged explicitly under "Implementation-time confirmations" above.
- **Backfill volume for four popular accounts.** `@bsky.app` and `@jay.bsky.team` have posted many times; first-run backfill may take several minutes and produce hundreds to low-thousands of rows. That's the intended sanity-test. If it turns out to be genuinely overwhelming, the mitigation is to trim the DID list — a compose-file edit, not a design change.
- **Log noise in foreground mode.** Interleaved JSON and plain-text logs in `just dev` may be hard to read under load. Mitigated by the per-service log-level defaults (postgres at warn, tap at info). If this becomes a real problem, switching appview's dev handler to `slog.NewTextHandler` is a follow-up, not scope here.

## Appendix — relationship to the scaffold spec

The scaffold spec ([2026-04-16-appview-server-scaffold-design.md](./2026-04-16-appview-server-scaffold-design.md)) designed a `firehose.Subscriber` interface with a single `Replay(ctx, since)` method, and an `index.Indexer.Backfill(ctx, did)` method, both as `NotImplemented` stubs. Both method shapes assumed direct Relay consumption plus separate backfill — the exact responsibilities Tap subsumes.

This spec deletes both interfaces in their original form and replaces them with the `tap.Consumer` / `index.Indexer.Handle` pair that fits Tap's single-stream model. The `cli firehose replay` and `cli backfill` subcommands go with them. Per AGENTS.md ("don't add backwards-compatibility shims like renaming unused vars … if you are certain that something is unused, you can delete it completely"), the deletion is clean — no aliases, no deprecated re-exports.
