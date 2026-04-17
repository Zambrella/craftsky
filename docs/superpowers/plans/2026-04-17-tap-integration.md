# Tap Integration Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the appview scaffold's `firehose.NotImplemented` stub with a real Tap WebSocket-with-acks consumer, stand up a `docker-compose` dev stack (postgres + migrate + tap + tap-bootstrap + appview), and prove the end-to-end pipe by indexing `app.bsky.feed.post` events from four allowlisted Bluesky accounts into a throwaway `bluesky_posts_sample` table.

**Architecture:** Tap runs as a sibling container, consuming the AT Protocol Relay and handing filtered JSON events to the appview over `ws://tap:2480/channel`. The appview's `tap.WSConsumer` dispatches each event to `index.Indexer.Handle`, which writes to Postgres. Identity events are dropped. Migrations and DID bootstrap run as one-shot compose services so `just dev` from a fresh clone produces a fully-configured running system.

**Tech Stack:** Go 1.23, `pgx/v5`, `golang-migrate/v4`, stdlib `net/http`, `slog`, `github.com/coder/websocket` (or `gorilla/websocket` — pick in Chunk 3); Postgres 16; Docker Compose; `just`; Tap (`ghcr.io/bluesky-social/indigo/tap:<pinned-tag>`).

**Spec:** [docs/superpowers/specs/2026-04-17-tap-integration-design.md](../specs/2026-04-17-tap-integration-design.md) — read before starting. All decisions referenced by "per spec §N" refer to that document.

**Working directory:** `/Users/douglastodd/Projects/craftsky` (the repo root). Go commands run from `appview/` unless noted.

---

## Background for a fresh contributor

If you're picking this up cold:

- The appview is a Go HTTP service that indexes atproto records into Postgres and serves JSON to a Flutter client. It currently exists only as a scaffold — connection plumbing, auth stub, HTTP router, no real event ingestion.
- Tap is a Go service (part of the indigo monorepo) that sits between the AT Protocol Relay and your app. It handles firehose connection management, CBOR/MST decoding, signature verification, backfill, and per-repo ordering, and hands you simple JSON events. Upstream README is at https://github.com/bluesky-social/indigo/tree/main/cmd/tap — read it.
- WebSocket-with-acks means: we read JSON frames from `ws://tap:2480/channel`, index each one into Postgres, then send an ack back. Tap redelivers any event we don't ack within `TAP_RETRY_TIMEOUT` (upstream default 60s). The exact ack frame format isn't documented in the README; Task 3.2 reads the `indigo/cmd/tap` source to figure it out.
- Bluesky handles (`@bsky.app`, etc.) resolve to DIDs (`did:plc:...`). Tap only knows DIDs. We resolve handles to DIDs once at setup time, paste them into the `tap-bootstrap` service in the compose file, and forget about it.

Project-wide rules live in [AGENTS.md](../../../AGENTS.md). Key rules that bind this work:

- `sqlc` for SQL → Go codegen (but NOT used yet in the scaffold; we will not introduce sqlc in this plan either — the single `bluesky_posts_sample` table is throwaway, and adding sqlc machinery for one table is YAGNI). Raw `pgxpool` calls are fine for the indexer.
- `slog` for logging, stdlib `net/http` for routing, `pgx/v5` for Postgres.
- **Integration tests must hit a real database, not mocks** — this is a binding user-memory rule. `BlueskyPostsSample` tests use real Postgres via per-test schemas.

---

## File structure

Files this plan creates or modifies, with responsibilities:

**Create:**
- `docker-compose.yml` (repo root) — Five services: `postgres`, `migrate` (one-shot), `tap`, `tap-bootstrap` (one-shot), `appview`.
- `justfile` (repo root) — Dev-workflow recipes.
- `appview/Dockerfile` — Multi-stage alpine build; produces both `/app/appview` and `/app/cli` binaries in the final image.
- `appview/.dockerignore` — Excludes `.git`, test binaries, editor artifacts from the build context.
- `appview/internal/tap/consumer.go` — The `Consumer` interface, `Event` struct, `ConnState` struct, `WSConsumer` implementation with reconnect + ack + poison-pill guard.
- `appview/internal/tap/consumer_test.go` — Tests against an `httptest.Server` hosting a fake WS.
- `appview/internal/index/bluesky_posts_sample.go` — `BlueskyPostsSample` implementation of `Indexer`.
- `appview/internal/index/bluesky_posts_sample_test.go` — Real-Postgres tests via per-test schemas.
- `appview/internal/api/health.go` — `GET /healthz` handler combining DB ping + Tap state.
- `appview/internal/api/health_test.go` — Table-driven tests of JSON response shape and `status` transitions.
- `appview/cmd/cli/tap.go` — `cli tap status` subcommand.
- `appview/cmd/cli/tap_test.go` — HTTP-fake-based tests of exit-code logic.
- `appview/migrations/000NN_bluesky_posts_sample.up.sql` + `.down.sql` — Creates/drops the sample table.

**Modify:**
- `appview/internal/index/indexer.go` — Replace `Backfill(ctx, did)` with `Handle(ctx, ev tap.Event)`.
- `appview/internal/app/config.go` — Add `TapWSURL`, `TapAckTimeout`, `TapReconnectMax`, `TapMaxRetries` fields; parse from env.
- `appview/internal/app/config_test.go` — Cover new keys.
- `appview/internal/app/deps.go` — Replace `Firehose firehose.Subscriber` with `Consumer tap.Consumer`; retype `Indexer` to match new interface.
- `appview/internal/app/deps_test.go` — Adjust for new fields.
- `appview/cmd/appview/server.go` — Start the Tap consumer goroutine alongside the HTTP server; cancel cleanly on shutdown.
- `appview/cmd/appview/server_test.go` (if present) — Adjust.
- `appview/internal/routes/` — Register `/healthz` route.
- `appview/cmd/cli/main.go` — Remove `firehose` and `backfill` subcommand registrations; add `tap` subcommand registration.
- `appview/environments/dev.env` — Add new TAP_ config keys.
- `appview/environments/prod.env.example` — Same.
- `appview/README.md` — Repo-layout tree, CLI table, Makefile promise → justfile.
- `README.md` (repo root) — New "Getting started" section.
- `AGENTS.md` — Dev-workflow line, repo-layout touch-up.
- `docs/superpowers/specs/2026-04-16-appview-server-scaffold-design.md` — Add "Status: partially superseded" header.

**Delete:**
- `appview/internal/firehose/subscriber.go` + `subscriber_test.go`.
- `appview/internal/firehose/` (empty dir).
- `appview/cmd/cli/firehose.go`.
- `appview/cmd/cli/backfill.go`.

---

## Chunk 1: Prep & infrastructure

Goal: Resolve the four handles to DIDs, pin the Tap image tag, scaffold the `docker-compose.yml`, `justfile`, and `appview/Dockerfile`. At the end of this chunk, `just dev` brings up postgres + tap + a stock appview (still with the `firehose.NotImplemented` stub — real consumer lands in Chunk 3). The compose stack is fully wired except for the new Go code.

Create a feature branch for this work at the start of the chunk: `git checkout -b feat/tap-integration`.

### Task 1.1: Resolve the four Bluesky handles to DIDs

**Files:** None modified — output is four DID strings for later tasks.

- [ ] **Step 1: Resolve each handle**

Run from any terminal with network access:

```bash
for h in bsky.app jay.bsky.team dougtodd.dev eurosky.social; do
  did=$(curl -fsS "https://public.api.bsky.app/xrpc/com.atproto.identity.resolveHandle?handle=$h" | jq -r .did)
  echo "$h -> $did"
done
```

Expected: four lines of the form `handle -> did:plc:...` (or possibly `did:web:...` for `dougtodd.dev` / `eurosky.social` if they use domain DIDs — both forms are valid and both work with Tap).

- [ ] **Step 2: Record the DIDs**

Save the four DIDs into this plan file itself under a "Resolved DIDs" section at the bottom (see Appendix A in this plan), so they're captured in git history and Task 1.3 can reference them by name. If any handle fails to resolve (network error, account deleted), note the error, pick a substitute popular account from Bluesky (any account with recent posts is fine; `@pfrazee.com`, `@why.bsky.team` are good candidates), and record both the failure and the replacement.

- [ ] **Step 3: Commit**

```bash
cd /Users/douglastodd/Projects/craftsky
git add docs/superpowers/plans/2026-04-17-tap-integration.md
git commit -m "plan: record resolved DIDs for tap integration"
```

### Task 1.2: Pin the Tap image tag

**Files:** None yet — output is a version string used in Task 1.3.

- [ ] **Step 1: Find the latest Tap release**

Check https://github.com/bluesky-social/indigo/pkgs/container/indigo%2Ftap for available tags, or run:

```bash
curl -fsS "https://ghcr.io/v2/bluesky-social/indigo/tap/tags/list" | jq .
```

(The unauthenticated endpoint may require a token; falling back to the web UI is fine.)

Pick the latest non-`latest` tag. Record the full image reference (`ghcr.io/bluesky-social/indigo/tap:<tag>`) in Appendix A of this plan.

- [ ] **Step 2: Verify the image pulls**

```bash
docker pull ghcr.io/bluesky-social/indigo/tap:<tag>
```

Expected: successful pull. If the image requires authentication, note that and fall back to building from source via `FROM golang:1.25-alpine AS tap-build / RUN go install github.com/bluesky-social/indigo/cmd/tap@<commit>` (a multi-stage Dockerfile addition). Not expected to be necessary — ghcr.io is typically public for Bluesky's images.

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/plans/2026-04-17-tap-integration.md
git commit -m "plan: pin tap image tag"
```

### Task 1.3: Create docker-compose.yml

**Files:**
- Create: `docker-compose.yml`

- [ ] **Step 1: Write the compose file**

Create `/Users/douglastodd/Projects/craftsky/docker-compose.yml` with the following content. Replace `<TAP_TAG>` with the tag from Task 1.2 and `<DID1>`, `<DID2>`, `<DID3>`, `<DID4>` with the DIDs from Task 1.1.

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

  migrate:
    build:
      context: ./appview
      dockerfile: Dockerfile
    # Override the image's ENTRYPOINT (/app/appview) so `command:` runs
    # the CLI binary directly instead of being appended as arguments.
    entrypoint: []
    command: ["/app/cli", "migrate", "up"]
    environment:
      DATABASE_URL: postgres://craftsky:dev@postgres:5432/craftsky_dev?sslmode=disable
      APPVIEW_ENV: dev
      ALLOWED_ORIGINS: "*"
      CRAFTSKY_DEV_DID: did:plc:craftsky-dev-user
    depends_on:
      postgres:
        condition: service_healthy
    restart: "no"

  tap:
    image: ghcr.io/bluesky-social/indigo/tap:<TAP_TAG>
    restart: unless-stopped
    environment:
      TAP_RELAY_URL: https://relay1.us-east.bsky.network
      TAP_DATABASE_URL: sqlite:///data/tap.db
      TAP_COLLECTION_FILTERS: "app.bsky.feed.post"
      TAP_LOG_LEVEL: info
      TAP_NO_REPLAY: "true"
    volumes:
      - tapdata:/data
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:2480/health || exit 1"]
      interval: 5s
      timeout: 3s
      retries: 10

  tap-bootstrap:
    image: curlimages/curl:8.9.1
    depends_on:
      tap:
        condition: service_healthy
    restart: "no"
    entrypoint: ["/bin/sh", "-c"]
    command:
      - |
        curl -fsS -X POST http://tap:2480/repos/add \
          -H "Content-Type: application/json" \
          -d '{"dids":["<DID1>","<DID2>","<DID3>","<DID4>"]}'

  appview:
    build:
      context: ./appview
      dockerfile: Dockerfile
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
      tap:
        condition: service_healthy
      tap-bootstrap:
        condition: service_completed_successfully
    environment:
      APPVIEW_ENV: dev
      DATABASE_URL: postgres://craftsky:dev@postgres:5432/craftsky_dev?sslmode=disable
      TAP_WS_URL: ws://tap:2480/channel
      TAP_ACK_TIMEOUT: "10s"
      TAP_RECONNECT_MAX: "30s"
      TAP_MAX_RETRIES: "5"
      ALLOWED_ORIGINS: "*"
      CRAFTSKY_DEV_DID: did:plc:craftsky-dev-user
    ports:
      - "8080:8080"

volumes:
  pgdata:
  tapdata:
```

- [ ] **Step 2: Validate compose syntax**

```bash
cd /Users/douglastodd/Projects/craftsky
docker compose config > /dev/null
```

Expected: exits 0 with no output. Any error means YAML or schema issue — fix before continuing.

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "feat(infra): add docker-compose stack for tap integration"
```

### Task 1.4: Create appview/Dockerfile and .dockerignore

**Files:**
- Create: `appview/Dockerfile`
- Create: `appview/.dockerignore`

- [ ] **Step 1: Write .dockerignore**

Create `/Users/douglastodd/Projects/craftsky/appview/.dockerignore`:

```
.git
*.test
*.out
coverage.*
.idea
.vscode
*.swp
```

- [ ] **Step 2: Write Dockerfile**

Create `/Users/douglastodd/Projects/craftsky/appview/Dockerfile`:

```dockerfile
# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS build
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

- [ ] **Step 3: Build the image**

```bash
cd /Users/douglastodd/Projects/craftsky
docker compose build appview
```

Expected: successful build. Fix any compilation errors (these would indicate pre-existing scaffold issues, not new problems).

- [ ] **Step 4: Commit**

```bash
git add appview/Dockerfile appview/.dockerignore
git commit -m "feat(infra): add appview Dockerfile"
```

### Task 1.5: Create justfile

**Files:**
- Create: `justfile`

- [ ] **Step 1: Verify just is installed**

```bash
just --version
```

If not present, install with `brew install just` (macOS) or see https://just.systems/man/en/chapter_4.html for alternatives.

- [ ] **Step 2: Write the justfile**

Create `/Users/douglastodd/Projects/craftsky/justfile`:

```
set dotenv-load := false

default:
    @just --list

# Start the full compose stack in the foreground.
dev:
    docker compose up --build

# Start detached.
dev-d:
    docker compose up -d --build

# Stop and remove containers. Volumes are preserved.
down:
    docker compose down

# Follow logs across all services.
logs:
    docker compose logs -f

# Run the CLI inside the appview container.
migrate *ARGS:
    docker compose exec appview /app/cli migrate {{ARGS}}

ping:
    docker compose exec appview /app/cli ping

tap-status:
    docker compose exec appview /app/cli tap status

# Open a psql session against the dev database.
psql:
    docker compose exec postgres psql -U craftsky craftsky_dev

# Run the Go test suite with the race detector enabled. The container is
# one-shot (--rm) so it does not leave artifacts behind, and it reaches
# postgres via the compose network.
test:
    docker compose run --rm appview go test -race ./...

# Format and vet Go code on the host.
fmt:
    cd appview && gofmt -w . && go vet ./...
```

- [ ] **Step 3: Verify just parses the file**

```bash
cd /Users/douglastodd/Projects/craftsky
just --list
```

Expected: lists every recipe above with no syntax errors.

- [ ] **Step 4: Commit**

```bash
git add justfile
git commit -m "feat(infra): add justfile with dev workflow recipes"
```

### Task 1.6: Smoke-test the compose stack

**Files:** None — this is a verification step.

- [ ] **Step 1: Cold start**

```bash
cd /Users/douglastodd/Projects/craftsky
docker compose down -v   # wipe any prior state
just dev
```

Expected (within ~60 seconds of start):
- `postgres` reaches `healthy`.
- `migrate` exits 0 (it has nothing to migrate yet — the sample-table migration lands in Chunk 4 — but `cli migrate up` against zero migration files is a no-op, not an error).
- `tap` reaches `healthy`.
- `tap-bootstrap` exits 0 (HTTP 200 from `/repos/add`).
- `appview` starts and serves `GET /health` (scaffold-level — `/healthz` lands in Chunk 4).

Wait 2 minutes with the stack up. In another terminal:

```bash
docker compose ps
docker compose logs tap | head -50
docker compose logs tap-bootstrap
```

Look for: tap logs should show the relay connection established and the four DIDs being backfilled. tap-bootstrap log should show `HTTP/1.1 200 OK` or similar.

If `migrate` fails because there are no migrations in the `migrations/` directory, that's actually correct behavior for `golang-migrate` — it will succeed with "no migrations found." If it fails with an error, inspect: `docker compose logs migrate`.

If tap-bootstrap fails with a 4xx/5xx, inspect the error body. Common issue: the `/repos/add` endpoint might require auth if `TAP_ADMIN_PASSWORD` is set — we don't set it, so it should be open.

- [ ] **Step 1a: Verify `cli migrate up` treats no-change as success**

`golang-migrate`'s `Up()` returns `migrate.ErrNoChange` when the DB is already at the latest migration. If the existing `cli migrate up` subcommand (from the scaffold spec) propagates this as an error, AC #3 ("running `just migrate up` again is a no-op") will fail. Read `appview/cmd/cli/migrate.go` and confirm `errors.Is(err, migrate.ErrNoChange)` is treated as success (exit 0). If not, patch it — a one-liner:

```go
if err != nil && !errors.Is(err, migrate.ErrNoChange) {
    return err
}
```

Commit as `fix(cli): treat ErrNoChange as success in migrate up` if a patch was needed. If the scaffold already handles this, no commit.

- [ ] **Step 2: Tear down**

```bash
just down
```

Expected: all containers stopped cleanly; volumes (`pgdata`, `tapdata`) preserved for next run.

- [ ] **Step 3: Commit any compose adjustments**

If Step 1 revealed a problem you had to fix (wrong env var name, wrong endpoint, etc.), commit that fix now with `fix(infra): <description>`. Otherwise no commit — this task is verification-only.

---

## Chunk 2: Go-side deletions and interface reshape

Goal: Delete the old firehose package and CLI subcommands, reshape `index.Indexer` to the new interface, and scaffold an empty `internal/tap/` package. The repo must still compile and pass existing tests at the end of this chunk. This is a "negative-space" commit that doesn't add new behavior but clears the way for Chunks 3 and 4.

### Task 2.1: Delete the firehose package

**Files:**
- Delete: `appview/internal/firehose/subscriber.go`
- Delete: `appview/internal/firehose/subscriber_test.go`
- Delete: `appview/internal/firehose/` (directory)

- [ ] **Step 1: Check for references**

```bash
cd /Users/douglastodd/Projects/craftsky
grep -rn "internal/firehose\|firehose\.Subscriber\|firehose\.NotImplemented" appview/
```

Expected references: `appview/internal/app/deps.go` (two places — import + field), `appview/cmd/cli/firehose.go` (import + usage). Record the list; these are the files we'll patch in Task 2.3 and 2.4.

- [ ] **Step 2: Delete the package**

```bash
rm -rf appview/internal/firehose
```

- [ ] **Step 3: Verify compile fails as expected**

```bash
cd appview
go build ./...
```

Expected: fails with "package ... internal/firehose: cannot find package" and/or "undefined: firehose". That's exactly what we're about to fix.

- [ ] **Step 4: No commit yet** — compile is broken; we'll commit at the end of Chunk 2 after the repo builds green.

### Task 2.2: Delete cli firehose and backfill subcommands

**Files:**
- Delete: `appview/cmd/cli/firehose.go`
- Delete: `appview/cmd/cli/backfill.go`
- Modify: `appview/cmd/cli/main.go` (remove `firehoseCmd` / `backfillCmd` registrations)

- [ ] **Step 1: Read cmd/cli/main.go**

Identify the lines that register `firehoseCmd` and `backfillCmd` (look for `rootCmd.AddCommand(firehoseCmd)` or similar, and any inline subcommand definitions).

- [ ] **Step 2: Delete the subcommand files**

```bash
rm appview/cmd/cli/firehose.go appview/cmd/cli/backfill.go
```

- [ ] **Step 3: Remove registrations from main.go**

Edit `appview/cmd/cli/main.go`: delete the `rootCmd.AddCommand(firehoseCmd)` / `rootCmd.AddCommand(backfillCmd)` lines, and any helper vars that are now orphaned. Run `go build ./cmd/cli` to surface remaining references and fix them.

- [ ] **Step 4: Delete any matching tests**

```bash
ls appview/cmd/cli/*_test.go
```

If there's a `firehose_test.go` or `backfill_test.go`, delete it.

- [ ] **Step 5: No commit yet** — still chained with Chunk 2.

### Task 2.3: Reshape index.Indexer interface

**Files:**
- Modify: `appview/internal/index/indexer.go`
- Modify: `appview/internal/index/indexer_test.go`

- [ ] **Step 1: Rewrite indexer.go**

Replace the contents of `appview/internal/index/indexer.go` with:

```go
// Package index defines the contract for writing atproto records into
// Postgres. Implementations are dispatched by the Tap consumer, one event
// at a time. Implementations MUST be idempotent on (URI, CID) because Tap
// delivers events at least once.
package index

import (
	"context"
	"errors"

	"social.craftsky/appview/internal/tap"
)

// Indexer writes records into the application's Postgres store.
type Indexer interface {
	// Handle processes a single Tap event. Returns nil on success;
	// any non-nil error causes the Tap consumer to skip the ack, so
	// Tap will redeliver the event after TAP_RETRY_TIMEOUT.
	Handle(ctx context.Context, ev tap.Event) error
}

// NotImplemented is a stub indexer that errors on every event.
// Used during construction before the real indexer is wired in.
type NotImplemented struct{}

var _ Indexer = NotImplemented{}

func (NotImplemented) Handle(ctx context.Context, ev tap.Event) error {
	return errors.New("indexer: not yet implemented")
}
```

Note: this imports `internal/tap`, which doesn't exist yet — Task 2.4 creates the skeleton.

- [ ] **Step 2: Rewrite indexer_test.go**

Replace `appview/internal/index/indexer_test.go` with a minimal test that confirms the `NotImplemented` stub errors:

```go
package index

import (
	"context"
	"testing"

	"social.craftsky/appview/internal/tap"
)

func TestNotImplementedHandleErrors(t *testing.T) {
	t.Parallel()
	err := NotImplemented{}.Handle(context.Background(), tap.Event{})
	if err == nil {
		t.Fatal("expected error from NotImplemented.Handle, got nil")
	}
}
```

- [ ] **Step 3: No compile/test yet** — still waiting on the `tap` package skeleton. Continue to Task 2.4.

### Task 2.4: Create internal/tap package skeleton

**Files:**
- Create: `appview/internal/tap/consumer.go`

- [ ] **Step 1: Write the package skeleton**

Create `/Users/douglastodd/Projects/craftsky/appview/internal/tap/consumer.go`:

```go
// Package tap consumes atproto events from a Tap sidecar over WebSocket.
//
// The real WSConsumer lands in a later commit. This file defines the public
// types so other packages (internal/index, internal/app, internal/api) can
// compile against them.
package tap

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Event is one decoded record-event from Tap's /channel WebSocket.
// Identity events are consumed internally by the consumer and are not
// surfaced to indexers.
type Event struct {
	URI        string          // at://did/collection/rkey
	CID        string          // content identifier of the record
	DID        string          // repo owner
	Collection string          // e.g. "app.bsky.feed.post"
	Rkey       string          // record key
	Action     string          // "create" | "update" | "delete"
	Record     json.RawMessage // opaque JSON; nil or empty on Action == "delete"
	Live       bool            // false during backfill, true for steady-state
	ID         uint64          // Tap's per-event "id" field from the envelope
	Rev        string          // repo rev at time of event
}

// Consumer is the interface the appview uses to consume events from Tap.
type Consumer interface {
	// Run blocks until ctx is cancelled, continuously connecting to Tap
	// and dispatching events to the configured indexer. It always returns
	// a non-nil error; on graceful shutdown the error is ctx.Err().
	Run(ctx context.Context) error

	// State returns a snapshot of the consumer's current connection state.
	// Safe to call concurrently with Run.
	State() ConnState
}

// ConnState describes the consumer's current connection state; used by the
// /healthz handler and the `cli tap status` command.
type ConnState struct {
	Connected        bool
	LastEventAt      time.Time
	LastError        string
	ReconnectAttempt int
}

// NotImplemented is a stub consumer used until WSConsumer lands.
// Run returns an error immediately; State reports disconnected.
type NotImplemented struct{}

var _ Consumer = NotImplemented{}

func (NotImplemented) Run(ctx context.Context) error {
	return errors.New("tap: consumer not yet implemented")
}

func (NotImplemented) State() ConnState {
	return ConnState{LastError: "not implemented"}
}
```

- [ ] **Step 2: Verify `internal/index` compiles against the skeleton**

```bash
cd /Users/douglastodd/Projects/craftsky/appview
go build ./internal/tap ./internal/index
```

Expected: success.

- [ ] **Step 3: No commit yet** — still chained with deps.go/server.go updates.

### Task 2.5: Update internal/app/deps.go and tests

**Files:**
- Modify: `appview/internal/app/deps.go`
- Modify: `appview/internal/app/deps_test.go`

- [ ] **Step 1: Patch deps.go**

Edit `appview/internal/app/deps.go`:

- Replace `"social.craftsky/appview/internal/firehose"` import with `"social.craftsky/appview/internal/tap"`.
- Change the `Firehose firehose.Subscriber` field on `Deps` to `Consumer tap.Consumer`.
- In `newDeps`, change `Firehose: firehose.NotImplemented{}` to `Consumer: tap.NotImplemented{}`.
- Leave the `Indexer` field in place; the type still compiles because `index.NotImplemented` still satisfies the new interface.

- [ ] **Step 2: Update deps_test.go**

Find any assertions on `deps.Firehose` and rename them to `deps.Consumer`. If the test asserts the concrete type is `firehose.NotImplemented`, change it to `tap.NotImplemented`.

- [ ] **Step 3: Build and test**

```bash
cd /Users/douglastodd/Projects/craftsky/appview
go build ./...
```

Expected: success.

```bash
go test ./...
```

Expected: all tests pass. If there are more call sites for `deps.Firehose` (likely in `cmd/appview/server.go` or `internal/routes/`), the build error from Step 3 surfaces them; rename each to `deps.Consumer` and retest.

- [ ] **Step 4: Commit the full Chunk 2 change as one unit**

```bash
git add -A
git status   # review: should show firehose deletions, indexer reshape, tap skeleton, deps rename
git commit -m "$(cat <<'EOF'
refactor(appview): remove firehose stub; introduce internal/tap skeleton

Deletes internal/firehose, cli firehose replay, and cli backfill — all
superseded by Tap's combined backfill-plus-live model. Reshapes
index.Indexer to a single Handle(ctx, Event) method consuming tap.Event.
Adds an internal/tap package skeleton with Consumer interface, Event
type, ConnState type, and NotImplemented stub so the rest of the code
compiles. No new behavior: repo still uses NotImplemented stubs for
both Indexer and Consumer.

Refs: docs/superpowers/specs/2026-04-17-tap-integration-design.md
EOF
)"
```

---

## Chunk 3: Config additions and real Tap consumer

Goal: Add the four `TAP_*` config keys, implement the real `WSConsumer` with TDD, and swap it into `Deps`. The consumer is tested against a fake WS server but not yet started from `server.go` (that wiring + integration with the real indexer lands in Chunk 4).

### Task 3.1: Add config keys

**Files:**
- Modify: `appview/internal/app/config.go`
- Modify: `appview/internal/app/config_test.go`
- Modify: `appview/environments/dev.env`
- Modify: `appview/environments/prod.env.example`

- [ ] **Step 1: Write failing test for new Config fields**

Append to `appview/internal/app/config_test.go`:

```go
func TestLoadConfig_TapFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	envPath := dir + "/test.env"
	contents := "DATABASE_URL=postgres://x\n" +
		"ALLOWED_ORIGINS=*\n" +
		"CRAFTSKY_DEV_DID=did:plc:test\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n" +
		"TAP_ACK_TIMEOUT=7s\n" +
		"TAP_RECONNECT_MAX=45s\n" +
		"TAP_MAX_RETRIES=3\n"
	if err := os.WriteFile(envPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	// Clear env so file wins.
	for _, k := range []string{"DATABASE_URL", "ALLOWED_ORIGINS", "CRAFTSKY_DEV_DID",
		"TAP_WS_URL", "TAP_ACK_TIMEOUT", "TAP_RECONNECT_MAX", "TAP_MAX_RETRIES"} {
		t.Setenv(k, "")
	}

	cfg, err := LoadConfig(EnvDev, envPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.TapWSURL != "ws://tap:2480/channel" {
		t.Errorf("TapWSURL = %q", cfg.TapWSURL)
	}
	if cfg.TapAckTimeout != 7*time.Second {
		t.Errorf("TapAckTimeout = %v", cfg.TapAckTimeout)
	}
	if cfg.TapReconnectMax != 45*time.Second {
		t.Errorf("TapReconnectMax = %v", cfg.TapReconnectMax)
	}
	if cfg.TapMaxRetries != 3 {
		t.Errorf("TapMaxRetries = %d", cfg.TapMaxRetries)
	}
}

func TestLoadConfig_TapDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	envPath := dir + "/test.env"
	contents := "DATABASE_URL=postgres://x\n" +
		"ALLOWED_ORIGINS=*\n" +
		"CRAFTSKY_DEV_DID=did:plc:test\n" +
		"TAP_WS_URL=ws://tap:2480/channel\n"
	if err := os.WriteFile(envPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"TAP_ACK_TIMEOUT", "TAP_RECONNECT_MAX", "TAP_MAX_RETRIES"} {
		t.Setenv(k, "")
	}

	cfg, err := LoadConfig(EnvDev, envPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TapAckTimeout != 10*time.Second {
		t.Errorf("default TapAckTimeout = %v", cfg.TapAckTimeout)
	}
	if cfg.TapReconnectMax != 30*time.Second {
		t.Errorf("default TapReconnectMax = %v", cfg.TapReconnectMax)
	}
	if cfg.TapMaxRetries != 5 {
		t.Errorf("default TapMaxRetries = %d", cfg.TapMaxRetries)
	}
}
```

Also add missing imports: `"time"`, `"os"` if not already present.

- [ ] **Step 2: Run tests, confirm failure**

```bash
cd /Users/douglastodd/Projects/craftsky/appview
go test ./internal/app -run TestLoadConfig_Tap
```

Expected: compilation failure (`cfg.TapWSURL undefined`). That's the red test.

- [ ] **Step 3: Add fields + parse logic**

Edit `appview/internal/app/config.go`:

Add to the `Config` struct:

```go
TapWSURL        string
TapAckTimeout   time.Duration
TapReconnectMax time.Duration
TapMaxRetries   int
```

Add `import "time"` and `import "strconv"` at the top.

Add parsing in `LoadConfig` (between the existing `cfg` initialization and the "Required everywhere" block):

```go
cfg.TapWSURL = os.Getenv("TAP_WS_URL")

ackTimeout := os.Getenv("TAP_ACK_TIMEOUT")
if ackTimeout == "" {
	cfg.TapAckTimeout = 10 * time.Second
} else {
	d, err := time.ParseDuration(ackTimeout)
	if err != nil {
		return Config{}, fmt.Errorf("TAP_ACK_TIMEOUT: %w", err)
	}
	cfg.TapAckTimeout = d
}

reconnectMax := os.Getenv("TAP_RECONNECT_MAX")
if reconnectMax == "" {
	cfg.TapReconnectMax = 30 * time.Second
} else {
	d, err := time.ParseDuration(reconnectMax)
	if err != nil {
		return Config{}, fmt.Errorf("TAP_RECONNECT_MAX: %w", err)
	}
	cfg.TapReconnectMax = d
}

maxRetries := os.Getenv("TAP_MAX_RETRIES")
if maxRetries == "" {
	cfg.TapMaxRetries = 5
} else {
	n, err := strconv.Atoi(maxRetries)
	if err != nil || n < 0 {
		return Config{}, fmt.Errorf("TAP_MAX_RETRIES: must be non-negative integer, got %q", maxRetries)
	}
	cfg.TapMaxRetries = n
}
```

Add to the "Required everywhere" block:

```go
if cfg.TapWSURL == "" {
	return Config{}, fmt.Errorf("TAP_WS_URL is required")
}
```

- [ ] **Step 4: Run tests, confirm green**

```bash
go test ./internal/app -run TestLoadConfig_Tap -v
```

Expected: both tests pass.

```bash
go test ./internal/app
```

Expected: all app tests pass.

- [ ] **Step 5: Update dev.env and prod.env.example**

Append to `appview/environments/dev.env`:

```
TAP_WS_URL=ws://tap:2480/channel
TAP_ACK_TIMEOUT=10s
TAP_RECONNECT_MAX=30s
TAP_MAX_RETRIES=5
```

Append to `appview/environments/prod.env.example` (with `prod.` prefix for namespacing is unnecessary — use the same keys):

```
TAP_WS_URL=ws://tap:2480/channel
TAP_ACK_TIMEOUT=10s
TAP_RECONNECT_MAX=30s
TAP_MAX_RETRIES=5
```

- [ ] **Step 6: Commit**

```bash
git add appview/internal/app/config.go appview/internal/app/config_test.go \
        appview/environments/dev.env appview/environments/prod.env.example
git commit -m "feat(appview): add TAP_* config keys with defaults"
```

### Task 3.2: Read Tap's ack frame format from indigo source

**Files:** None yet — output is a code comment + understanding that informs Task 3.3.

- [ ] **Step 1: Clone or fetch indigo**

```bash
cd /tmp
git clone --depth 1 https://github.com/bluesky-social/indigo.git indigo-read
cd indigo-read/cmd/tap
ls
```

If a specific commit is pinned by the Tap image tag (Task 1.2), check out that commit: `git fetch --depth 1 origin <commit-sha> && git checkout <commit-sha>`. Otherwise use main — the WS protocol is unlikely to have changed in minor versions.

- [ ] **Step 2: Find the WS handler**

Look for the file that serves `/channel`. Likely candidates:

```bash
grep -rn "/channel" .
grep -rn "WebSocket\|websocket\|upgrader" .
```

The server-side handler receives ack frames. Find the code path that reads from the WS and interprets the ack. Typical shape: a JSON frame like `{"ack": <id>}` or `{"id": <id>, "ack": true}`.

- [ ] **Step 3: Record findings**

Document:
- The exact ack frame structure (JSON shape).
- Whether acks are sent as text or binary WS frames.
- Whether the server expects one ack per event or permits batched acks.
- Whether the server tolerates unknown fields in the ack frame.

Save these findings inline as a comment in the Task 3.3 implementation (see below).

### Task 3.3: Implement WSConsumer (TDD)

**Files:**
- Create: `appview/internal/tap/consumer_test.go`
- Modify: `appview/internal/tap/consumer.go` (add the `WSConsumer` type and supporting code)

The WS library choice: use `github.com/coder/websocket` (formerly `nhooyr.io/websocket`). Rationale: stdlib-shaped API, context-first, well-maintained, no cgo. Add to `go.mod` in Step 3.

- [ ] **Step 1: Write the failing test for happy-path event flow**

Create `appview/internal/tap/consumer_test.go`:

```go
package tap_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"social.craftsky/appview/internal/tap"
)

// fakeIndexer records Handle calls and can be configured to fail.
type fakeIndexer struct {
	mu       sync.Mutex
	events   []tap.Event
	failOnce bool // if true, next Handle returns error (then resets to false)
}

func (f *fakeIndexer) Handle(ctx context.Context, ev tap.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failOnce {
		f.failOnce = false
		return errTest
	}
	f.events = append(f.events, ev)
	return nil
}

func (f *fakeIndexer) Events() []tap.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]tap.Event, len(f.events))
	copy(out, f.events)
	return out
}

var errTest = &testErr{msg: "intentional test error"}

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }

// fakeTap is a minimal /channel WS server. It sends the provided frames
// on connect, then listens for ack frames from the client.
type fakeTap struct {
	frames []string
	acks   chan uint64
}

func newFakeTap(frames []string) *fakeTap {
	return &fakeTap{frames: frames, acks: make(chan uint64, 32)}
}

func (f *fakeTap) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("websocket accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		ctx := r.Context()

		// Send all frames up front.
		for _, fr := range f.frames {
			if err := conn.Write(ctx, websocket.MessageText, []byte(fr)); err != nil {
				return
			}
		}

		// Read acks until client closes. Ack shape confirmed against
		// indigo/cmd/tap types.go: {"type": "ack", "id": <uint>}.
		for {
			var ack map[string]any
			if err := wsjson.Read(ctx, conn, &ack); err != nil {
				return
			}
			if ack["type"] == "ack" {
				if id, ok := ack["id"].(float64); ok {
					f.acks <- uint64(id)
				}
			}
		}
	})
}

func TestWSConsumer_HappyPath(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":1,"type":"record","record":{"live":true,"rev":"r1","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k1","action":"create","cid":"bafy1","record":{"text":"hi"}}}`,
		`{"id":2,"type":"record","record":{"live":true,"rev":"r2","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k2","action":"create","cid":"bafy2","record":{"text":"hey"}}}`,
		`{"id":3,"type":"record","record":{"live":false,"rev":"r3","did":"did:plc:b","collection":"app.bsky.feed.post","rkey":"k3","action":"delete","cid":"bafy3"}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	idx := &fakeIndexer{}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:           wsURL,
		Indexer:       idx,
		AckTimeout:    5 * time.Second,
		ReconnectMax:  1 * time.Second,
		MaxRetries:    5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

	// Wait for three events to be indexed.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(idx.Events()) == 3 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	evs := idx.Events()
	if len(evs) != 3 {
		t.Fatalf("indexed %d events, want 3; got %+v", len(evs), evs)
	}

	// Wait for three acks on the server side.
	seenAcks := map[uint64]bool{}
	for i := 0; i < 3; i++ {
		select {
		case id := <-ft.acks:
			seenAcks[id] = true
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for ack #%d; seen so far: %v", i, seenAcks)
		}
	}
	for _, want := range []uint64{1, 2, 3} {
		if !seenAcks[want] {
			t.Errorf("missing ack for id=%d", want)
		}
	}

	// Assert Event field mapping.
	if evs[0].URI != "at://did:plc:a/app.bsky.feed.post/k1" {
		t.Errorf("evs[0].URI = %q", evs[0].URI)
	}
	if evs[0].Action != "create" {
		t.Errorf("evs[0].Action = %q", evs[0].Action)
	}
	if !evs[0].Live {
		t.Errorf("evs[0].Live should be true")
	}
	if string(evs[0].Record) == "" || !json.Valid(evs[0].Record) {
		t.Errorf("evs[0].Record invalid: %q", evs[0].Record)
	}
	if evs[2].Action != "delete" {
		t.Errorf("evs[2].Action = %q", evs[2].Action)
	}

	// Cancel and wait for Run to return.
	cancel()
	select {
	case err := <-done:
		if err != nil && !isContextCanceled(err) {
			t.Errorf("Run returned %v; want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func isContextCanceled(err error) bool {
	return err == context.Canceled || strings.Contains(err.Error(), "context canceled")
}
```

- [ ] **Step 2: Add the websocket dependency**

```bash
cd /Users/douglastodd/Projects/craftsky/appview
go get github.com/coder/websocket@latest
go mod tidy
```

- [ ] **Step 3: Run the test, confirm red**

```bash
go test ./internal/tap -run TestWSConsumer_HappyPath -v
```

Expected: compile failure — `tap.NewWSConsumer` and `tap.WSConsumerConfig` don't exist yet.

- [ ] **Step 4: Implement WSConsumer**

Append to `appview/internal/tap/consumer.go` (after the `NotImplemented` type):

```go
// WSConsumerConfig wires a WSConsumer. All fields are required.
type WSConsumerConfig struct {
	URL          string        // ws://tap:2480/channel
	Indexer      HandlerIndexer
	AckTimeout   time.Duration // per-event Handle deadline
	ReconnectMax time.Duration // cap for exponential reconnect backoff
	MaxRetries   int           // poison-pill threshold per event id
	Logger       *slog.Logger  // optional; nil → slog.Default()
}

// HandlerIndexer is the narrow interface the consumer needs. Defined here
// (not imported from internal/index) to avoid an import cycle.
type HandlerIndexer interface {
	Handle(ctx context.Context, ev Event) error
}

// WSConsumer connects to Tap's /channel WebSocket and dispatches events
// to an indexer, sending acks on success.
type WSConsumer struct {
	cfg    WSConsumerConfig
	logger *slog.Logger

	mu         sync.Mutex
	state      ConnState
	retryCount map[uint64]int // event id → how many times it's failed
}

// NewWSConsumer returns a consumer that connects to the given Tap WS URL.
func NewWSConsumer(cfg WSConsumerConfig) *WSConsumer {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &WSConsumer{
		cfg:        cfg,
		logger:     logger,
		retryCount: map[uint64]int{},
	}
}

func (c *WSConsumer) State() ConnState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *WSConsumer) setConnected(connected bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Connected = connected
	if connected {
		c.state.LastError = ""
		c.state.ReconnectAttempt = 0
	}
}

func (c *WSConsumer) recordError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Connected = false
	c.state.LastError = err.Error()
	c.state.ReconnectAttempt++
}

func (c *WSConsumer) recordEvent() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.LastEventAt = time.Now().UTC()
}

// Run loops forever connecting, reading, and reconnecting on error.
// Returns only when ctx is cancelled.
func (c *WSConsumer) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := c.runOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		c.recordError(err)
		backoff := c.backoff()
		c.logger.Warn("tap consumer disconnected",
			slog.Any("err", err),
			slog.Duration("backoff", backoff),
			slog.Int("attempt", c.State().ReconnectAttempt),
		)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *WSConsumer) backoff() time.Duration {
	attempt := c.State().ReconnectAttempt
	if attempt <= 0 {
		// recordError should have incremented attempt to >=1 before
		// backoff() is called, but guard anyway so callers can't panic.
		return time.Second
	}
	// 1s, 2s, 4s, 8s, 16s, 32s... capped at ReconnectMax.
	// A very large attempt would overflow the shift into negative; the
	// <= 0 check below catches that and clamps to ReconnectMax.
	d := time.Second << (attempt - 1)
	if d <= 0 || d > c.cfg.ReconnectMax {
		d = c.cfg.ReconnectMax
	}
	return d
}

// envelope is the outer shape of every frame Tap sends.
type envelope struct {
	ID       uint64          `json:"id"`
	Type     string          `json:"type"`
	Record   *recordPayload  `json:"record,omitempty"`
	Identity json.RawMessage `json:"identity,omitempty"`
}

type recordPayload struct {
	Live       bool            `json:"live"`
	Rev        string          `json:"rev"`
	DID        string          `json:"did"`
	Collection string          `json:"collection"`
	Rkey       string          `json:"rkey"`
	Action     string          `json:"action"`
	CID        string          `json:"cid"`
	Record     json.RawMessage `json:"record,omitempty"`
}

// ackFrame is sent back to Tap after a successful Handle.
//
// Shape confirmed by reading indigo/cmd/tap/types.go (types WsResponse,
// WsResponseAck) and server.go's /channel handler during Task 3.2.
// Tap's server sends outgoing events as raw bytes over TextMessage frames
// containing a MarshallableEvt JSON. The client acks with a WsResponse
// containing {"type": "ack", "id": <id>}.
type ackFrame struct {
	Type string `json:"type"` // always "ack"
	ID   uint64 `json:"id"`
}

// runOnce handles one WS connection lifecycle.
func (c *WSConsumer) runOnce(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, c.cfg.URL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	c.setConnected(true)
	c.logger.Info("tap consumer connected", slog.String("url", c.cfg.URL))

	for {
		var env envelope
		if err := wsjson.Read(ctx, conn, &env); err != nil {
			return fmt.Errorf("read: %w", err)
		}
		c.recordEvent()

		switch env.Type {
		case "record":
			if env.Record == nil {
				c.logger.Warn("record envelope missing record field", slog.Uint64("id", env.ID))
				continue
			}
			ev := Event{
				URI:        fmt.Sprintf("at://%s/%s/%s", env.Record.DID, env.Record.Collection, env.Record.Rkey),
				CID:        env.Record.CID,
				DID:        env.Record.DID,
				Collection: env.Record.Collection,
				Rkey:       env.Record.Rkey,
				Action:     env.Record.Action,
				Record:     env.Record.Record,
				Live:       env.Record.Live,
				ID:         env.ID,
				Rev:        env.Record.Rev,
			}
			if err := c.handleWithTimeout(ctx, ev); err != nil {
				c.logger.Error("indexer handle failed",
					slog.String("uri", ev.URI),
					slog.Uint64("id", ev.ID),
					slog.Any("err", err),
				)
				if c.shouldDrop(ev.ID) {
					c.logger.Error("dropping poison-pill event after retries",
						slog.String("uri", ev.URI),
						slog.Uint64("id", ev.ID),
						slog.String("record", string(ev.Record)),
					)
					if err := c.sendAck(ctx, conn, ev.ID); err != nil {
						return fmt.Errorf("ack: %w", err)
					}
					c.forgetRetry(ev.ID)
				}
				continue // do not ack on ordinary error
			}
			c.forgetRetry(ev.ID)
			if err := c.sendAck(ctx, conn, ev.ID); err != nil {
				return fmt.Errorf("ack: %w", err)
			}
		case "identity":
			// Drop identity events at debug.
			c.logger.Debug("tap identity event received", slog.Uint64("id", env.ID))
			if err := c.sendAck(ctx, conn, env.ID); err != nil {
				return fmt.Errorf("ack: %w", err)
			}
		default:
			c.logger.Warn("unknown tap envelope type", slog.String("type", env.Type), slog.Uint64("id", env.ID))
			// Ack anyway to avoid blocking.
			if err := c.sendAck(ctx, conn, env.ID); err != nil {
				return fmt.Errorf("ack: %w", err)
			}
		}
	}
}

func (c *WSConsumer) handleWithTimeout(ctx context.Context, ev Event) error {
	handleCtx, cancel := context.WithTimeout(ctx, c.cfg.AckTimeout)
	defer cancel()
	if err := c.cfg.Indexer.Handle(handleCtx, ev); err != nil {
		c.mu.Lock()
		c.retryCount[ev.ID]++
		c.mu.Unlock()
		return err
	}
	return nil
}

func (c *WSConsumer) shouldDrop(id uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Drop when we have failed MORE THAN MaxRetries times. With
	// MaxRetries=5: first 5 failures are ignored (Tap redelivers), the
	// 6th failure triggers drop+ack.
	return c.retryCount[id] > c.cfg.MaxRetries
}

func (c *WSConsumer) forgetRetry(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.retryCount, id)
}

func (c *WSConsumer) sendAck(ctx context.Context, conn *websocket.Conn, id uint64) error {
	return wsjson.Write(ctx, conn, ackFrame{Type: "ack", ID: id})
}
```

Add the necessary imports to the top of the file (replacing the existing import block):

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)
```

- [ ] **Step 5: Run the happy-path test, confirm green**

```bash
go test ./internal/tap -run TestWSConsumer_HappyPath -v
```

Expected: pass.

- [ ] **Step 6: Write the indexer-error test**

Append to `consumer_test.go`:

```go
func TestWSConsumer_IndexerErrorDoesNotAck(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":42,"type":"record","record":{"live":true,"rev":"r","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{"text":"x"}}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	idx := &fakeIndexer{failOnce: true}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Indexer:      idx,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
		MaxRetries:   100, // high — we only want to see the first failure
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	// We expect zero acks within 500ms (indexer failed, so no ack sent).
	select {
	case id := <-ft.acks:
		t.Fatalf("unexpected ack for id=%d after indexer error", id)
	case <-time.After(500 * time.Millisecond):
		// good: no ack
	}
}
```

- [ ] **Step 7: Run and confirm green**

```bash
go test ./internal/tap -run TestWSConsumer_IndexerErrorDoesNotAck -v
```

Expected: pass.

- [ ] **Step 8: Write reconnect test**

Append:

```go
func TestWSConsumer_ReconnectsOnWSClose(t *testing.T) {
	t.Parallel()

	var connCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&connCount, 1)
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		// Close the connection immediately.
		conn.Close(websocket.StatusInternalError, "simulated failure")
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Indexer:      &fakeIndexer{},
		AckTimeout:   1 * time.Second,
		ReconnectMax: 200 * time.Millisecond, // tight for fast test
		MaxRetries:   5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&connCount) >= 3 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&connCount); got < 3 {
		t.Fatalf("connected %d times, expected ≥3 reconnects", got)
	}

	st := c.State()
	if st.ReconnectAttempt < 2 {
		t.Errorf("ReconnectAttempt = %d, want ≥2", st.ReconnectAttempt)
	}
}
```

Add `sync/atomic` import.

- [ ] **Step 9: Run and confirm green**

```bash
go test ./internal/tap -v
```

Expected: all three tests pass.

- [ ] **Step 10: Write the poison-pill test**

```go
func TestWSConsumer_PoisonPillIsDroppedAfterMaxRetries(t *testing.T) {
	t.Parallel()

	// This is tricky: without redelivery, we only see "id=99" once.
	// So the poison-pill path is exercised only if Tap re-sends. We
	// emulate that by sending the same id 7 times in a row from the
	// fake server. With MaxRetries=5, the first 5 failures are ignored
	// (no ack), the 6th failure triggers the drop-and-ack path. We
	// send one extra frame as a buffer to avoid relying on exact count.
	sameFrame := `{"id":99,"type":"record","record":{"live":true,"rev":"r","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{"text":"x"}}}`
	frames := []string{sameFrame, sameFrame, sameFrame, sameFrame, sameFrame, sameFrame, sameFrame}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	idx := &alwaysFailIndexer{}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Indexer:      idx,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
		MaxRetries:   5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	// Expect exactly one ack for id=99 after the 5th-failure → 6th-attempt drop.
	select {
	case id := <-ft.acks:
		if id != 99 {
			t.Fatalf("ack id=%d, want 99", id)
		}
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("timeout waiting for poison-pill ack")
	}
}

type alwaysFailIndexer struct{}

func (alwaysFailIndexer) Handle(ctx context.Context, ev tap.Event) error { return errTest }
```

- [ ] **Step 11: Run and confirm green**

```bash
go test ./internal/tap -v
```

Expected: all four tests pass. If the poison-pill test fails, double-check the retry-count semantics: `shouldDrop` uses `retryCount[id] > MaxRetries`, so with `MaxRetries=5` the 6th failure triggers the drop. The test sends 7 identical frames to guarantee we reach the 6th.

- [ ] **Step 12: Extract the indexer into a local variable**

Currently `newDeps` has (from Chunk 2):

```go
deps := &Deps{
    ...
    Indexer:  index.NotImplemented{},
    Consumer: tap.NotImplemented{},
}
```

Before constructing the consumer, pull the indexer out so the consumer can reference it. Edit `appview/internal/app/deps.go` — replace the inline struct literal with:

```go
indexerImpl := index.NotImplemented{}

deps := &Deps{
    ...
    Indexer:  indexerImpl,
    Consumer: tap.NotImplemented{},  // temp, replaced below
}
```

- [ ] **Step 13: Swap `NotImplemented` for `WSConsumer`**

Immediately after the struct literal, construct the real consumer:

```go
deps.Consumer = tap.NewWSConsumer(tap.WSConsumerConfig{
    URL:          cfg.TapWSURL,
    Indexer:      indexerImpl,
    AckTimeout:   cfg.TapAckTimeout,
    ReconnectMax: cfg.TapReconnectMax,
    MaxRetries:   cfg.TapMaxRetries,
    Logger:       logger,
})
```

No wrapper type is needed because `index.Indexer` and `tap.HandlerIndexer` have identical signatures — `index.NotImplemented{}` satisfies both. The real `index.BlueskyPostsSample` lands in Chunk 4 and also satisfies both.

- [ ] **Step 14: Verify build + tests still green**

```bash
go build ./...
go test ./...
```

Expected: everything green. `deps_test.go` assertions should now see `*tap.WSConsumer` instead of `tap.NotImplemented{}`; adjust if needed.

- [ ] **Step 15: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat(appview): implement WSConsumer for tap /channel WebSocket

Real at-least-once consumer with ack-on-success semantics, context
timeout per Handle, exponential reconnect backoff capped at
TAP_RECONNECT_MAX, and a poison-pill guard that drops an event after
TAP_MAX_RETRIES consecutive Handle failures. Identity events are
acked and dropped at debug level.

Ack frame shape confirmed by reading indigo/cmd/tap source.
EOF
)"
```

---

## Chunk 4: Indexer, migration, health endpoint, CLI

Goal: Ship the `BlueskyPostsSample` indexer backed by a real Postgres migration, register `/healthz`, wire the consumer goroutine into `server.go`, and add `cli tap status`. At the end of this chunk, `just dev` produces rows in `bluesky_posts_sample` — acceptance criterion #4 is satisfiable.

### Task 4.1: Migration

**Files:**
- Create: `appview/migrations/<NNNN>_bluesky_posts_sample.up.sql`
- Create: `appview/migrations/<NNNN>_bluesky_posts_sample.down.sql`

- [ ] **Step 1: Find next migration number**

```bash
ls /Users/douglastodd/Projects/craftsky/appview/migrations/
```

Pick the next integer prefix (e.g. if `000001_*.up.sql` is highest, use `000002`). If the directory is empty, use `000001`.

- [ ] **Step 2: Write the up migration**

Create `appview/migrations/<NNNN>_bluesky_posts_sample.up.sql`:

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

- [ ] **Step 3: Write the down migration**

Create `appview/migrations/<NNNN>_bluesky_posts_sample.down.sql`:

```sql
DROP INDEX IF EXISTS bluesky_posts_sample_did_idx;
DROP TABLE IF EXISTS bluesky_posts_sample;
```

- [ ] **Step 4: Smoke-test migration**

```bash
cd /Users/douglastodd/Projects/craftsky
just down
docker volume rm craftsky_pgdata 2>/dev/null || true   # fresh DB
just dev-d
docker compose logs migrate
```

Expected: `migrate` service exits 0, logs show "Applied N migrations" or similar. Then:

```bash
just psql
```

In the psql prompt: `\dt` should list `bluesky_posts_sample`. `\q` to exit.

- [ ] **Step 5: Commit**

```bash
git add appview/migrations/
git commit -m "feat(appview): add bluesky_posts_sample migration (sample table)"
```

### Task 4.2: BlueskyPostsSample indexer (TDD with real Postgres)

**Files:**
- Create: `appview/internal/index/bluesky_posts_sample.go`
- Create: `appview/internal/index/bluesky_posts_sample_test.go`

Note: this task requires a running Postgres. The tests use `os.Getenv("TEST_DATABASE_URL")` falling back to `DATABASE_URL`. When run via `just test` (which invokes `docker compose run --rm appview go test ./...`), `DATABASE_URL` is set by compose. When run on the host, export it manually: `export TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5432/craftsky_dev?sslmode=disable`.

- [ ] **Step 1: Write the failing test harness**

Create `appview/internal/index/bluesky_posts_sample_test.go`:

```go
package index_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
)

// withSchema creates an isolated schema for one test and returns a pool
// whose default search_path points at it. Dropped via t.Cleanup.
func withSchema(t *testing.T) *pgxpool.Pool {
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
	schemaName := fmt.Sprintf("test_%d", rand.Uint32())
	if _, err := bootstrap.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = bootstrap.Exec(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE")
		bootstrap.Close()
	})

	// Copy the table into the fresh schema so we don't pollute public.
	ddl := `
		CREATE TABLE ` + schemaName + `.bluesky_posts_sample (
			uri        TEXT PRIMARY KEY,
			cid        TEXT NOT NULL,
			did        TEXT NOT NULL,
			rkey       TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			record     JSONB NOT NULL
		);
		CREATE INDEX ON ` + schemaName + `.bluesky_posts_sample (did);
	`
	if _, err := bootstrap.Exec(ctx, ddl); err != nil {
		t.Fatalf("create test table: %v", err)
	}

	// Return a pool whose search_path is scoped to the test schema.
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schemaName
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestBlueskyPostsSample_Create(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	ev := tap.Event{
		URI:        "at://did:plc:abc/app.bsky.feed.post/3k1",
		CID:        "bafy1",
		DID:        "did:plc:abc",
		Rkey:       "3k1",
		Collection: "app.bsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"hello"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM bluesky_posts_sample WHERE uri = $1", ev.URI).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestBlueskyPostsSample_CreateTwiceIsIdempotent(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	ev := tap.Event{
		URI:        "at://did:plc:abc/app.bsky.feed.post/3k1",
		CID:        "bafy1",
		DID:        "did:plc:abc",
		Rkey:       "3k1",
		Collection: "app.bsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"v1"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	// Second delivery with same URI+CID — should not error.
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("second Handle: %v", err)
	}
	var count int
	_ = pool.QueryRow(context.Background(),
		"SELECT count(*) FROM bluesky_posts_sample WHERE uri = $1", ev.URI).Scan(&count)
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestBlueskyPostsSample_Update(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	create := tap.Event{
		URI: "at://did:plc:x/app.bsky.feed.post/k", CID: "c1", DID: "did:plc:x", Rkey: "k",
		Collection: "app.bsky.feed.post", Action: "create",
		Record: json.RawMessage(`{"text":"old"}`),
	}
	update := create
	update.CID = "c2"
	update.Action = "update"
	update.Record = json.RawMessage(`{"text":"new"}`)

	if err := idx.Handle(context.Background(), create); err != nil {
		t.Fatal(err)
	}
	if err := idx.Handle(context.Background(), update); err != nil {
		t.Fatal(err)
	}

	var cid string
	var record []byte
	err := pool.QueryRow(context.Background(),
		"SELECT cid, record FROM bluesky_posts_sample WHERE uri = $1", create.URI).
		Scan(&cid, &record)
	if err != nil {
		t.Fatal(err)
	}
	if cid != "c2" {
		t.Errorf("cid = %q, want c2", cid)
	}
	if string(record) != `{"text":"new"}` {
		t.Errorf("record = %q", record)
	}
}

func TestBlueskyPostsSample_Delete(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	create := tap.Event{
		URI: "at://did:plc:x/app.bsky.feed.post/k", CID: "c1", DID: "did:plc:x", Rkey: "k",
		Collection: "app.bsky.feed.post", Action: "create",
		Record: json.RawMessage(`{"text":"bye"}`),
	}
	if err := idx.Handle(context.Background(), create); err != nil {
		t.Fatal(err)
	}

	del := tap.Event{URI: create.URI, DID: create.DID, Rkey: create.Rkey,
		Collection: "app.bsky.feed.post", Action: "delete"}
	if err := idx.Handle(context.Background(), del); err != nil {
		t.Fatalf("delete Handle: %v", err)
	}

	var count int
	_ = pool.QueryRow(context.Background(),
		"SELECT count(*) FROM bluesky_posts_sample WHERE uri = $1", create.URI).Scan(&count)
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestBlueskyPostsSample_DeleteMissingIsNoop(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	del := tap.Event{
		URI: "at://did:plc:z/app.bsky.feed.post/nothing",
		Collection: "app.bsky.feed.post", Action: "delete",
	}
	if err := idx.Handle(context.Background(), del); err != nil {
		t.Errorf("delete-missing Handle: %v", err)
	}
}

func TestBlueskyPostsSample_UnknownCollectionIgnored(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	idx := index.NewBlueskyPostsSample(pool)

	ev := tap.Event{
		URI: "at://did:plc:y/app.bsky.graph.follow/k", CID: "c", DID: "did:plc:y", Rkey: "k",
		Collection: "app.bsky.graph.follow", Action: "create",
		Record: json.RawMessage(`{"subject":"x"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("unknown-collection Handle: %v", err)
	}
	var count int
	_ = pool.QueryRow(context.Background(), "SELECT count(*) FROM bluesky_posts_sample").Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}
```

- [ ] **Step 2: Run, confirm red**

```bash
cd /Users/douglastodd/Projects/craftsky
just test
```

Expected: compile error — `index.NewBlueskyPostsSample` undefined.

- [ ] **Step 3: Implement the indexer**

Create `appview/internal/index/bluesky_posts_sample.go`:

```go
package index

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

// BlueskyPostsSample is a throwaway indexer that writes Bluesky posts
// into the bluesky_posts_sample table. It exists to validate the
// end-to-end Tap → appview → Postgres pipe end-to-end and MUST be
// deleted when the first social.craftsky.* indexer lands.
//
// See docs/superpowers/specs/2026-04-17-tap-integration-design.md.
type BlueskyPostsSample struct {
	pool *pgxpool.Pool
}

var _ Indexer = (*BlueskyPostsSample)(nil)

// NewBlueskyPostsSample builds an indexer backed by the given pool.
func NewBlueskyPostsSample(pool *pgxpool.Pool) *BlueskyPostsSample {
	return &BlueskyPostsSample{pool: pool}
}

const blueskyPostNSID = "app.bsky.feed.post"

// Handle indexes create/update/delete on app.bsky.feed.post into
// bluesky_posts_sample. Other collections are ignored.
func (b *BlueskyPostsSample) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != blueskyPostNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		const q = `
			INSERT INTO bluesky_posts_sample (uri, cid, did, rkey, record)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (uri) DO UPDATE SET
				cid = EXCLUDED.cid,
				record = EXCLUDED.record
		`
		if _, err := b.pool.Exec(ctx, q, ev.URI, ev.CID, ev.DID, ev.Rkey, ev.Record); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		if _, err := b.pool.Exec(ctx,
			`DELETE FROM bluesky_posts_sample WHERE uri = $1`, ev.URI); err != nil {
			return fmt.Errorf("delete %s: %w", ev.URI, err)
		}
		return nil
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}
```

- [ ] **Step 4: Run tests, confirm green**

```bash
just test
```

Expected: all six `TestBlueskyPostsSample_*` tests pass.

- [ ] **Step 5: Commit**

```bash
git add appview/internal/index/bluesky_posts_sample.go appview/internal/index/bluesky_posts_sample_test.go
git commit -m "feat(index): add BlueskyPostsSample indexer for scaffold validation"
```

### Task 4.3: /healthz handler

**Files:**
- Create: `appview/internal/api/health.go`
- Create: `appview/internal/api/health_test.go`
- Modify: `appview/internal/routes/routes.go` (or wherever route registration lives)

- [ ] **Step 1: Inspect existing route registration**

```bash
cat /Users/douglastodd/Projects/craftsky/appview/internal/routes/*.go
```

Note the pattern used to register routes (factory function, `Deps` injection, mux).

- [ ] **Step 2: Write failing test for /healthz**

Create `appview/internal/api/health_test.go`:

```go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/tap"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(ctx context.Context) error { return f.err }

type fakeStater struct{ state tap.ConnState }

func (f *fakeStater) State() tap.ConnState { return f.state }

func TestHealthz_AllOK(t *testing.T) {
	t.Parallel()
	h := api.NewHealthHandler(fakePinger{}, &fakeStater{state: tap.ConnState{Connected: true, LastEventAt: time.Unix(1700000000, 0)}})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))

	if rr.Code != 200 {
		t.Fatalf("code = %d", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v", body["status"])
	}
	if body["db"] != "ok" {
		t.Errorf("db = %v", body["db"])
	}
	tapBlock, ok := body["tap"].(map[string]any)
	if !ok {
		t.Fatalf("tap block missing: %+v", body)
	}
	if tapBlock["connected"] != true {
		t.Errorf("tap.connected = %v", tapBlock["connected"])
	}
}

func TestHealthz_TapDisconnectedDegraded(t *testing.T) {
	t.Parallel()
	h := api.NewHealthHandler(fakePinger{}, &fakeStater{state: tap.ConnState{Connected: false, LastError: "dial timeout"}})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))

	if rr.Code != 200 {
		t.Fatalf("code = %d (degraded should still be 200)", rr.Code)
	}
	var body map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body["status"] != "degraded" {
		t.Errorf("status = %v", body["status"])
	}
}

func TestHealthz_DBErrorDegraded(t *testing.T) {
	t.Parallel()
	h := api.NewHealthHandler(fakePinger{err: errors.New("ping failed")}, &fakeStater{state: tap.ConnState{Connected: true}})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != 200 {
		t.Errorf("code = %d", rr.Code)
	}
	var body map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body["db"] != "error" {
		t.Errorf("db = %v", body["db"])
	}
	if body["status"] != "degraded" {
		t.Errorf("status = %v", body["status"])
	}
}
```

- [ ] **Step 3: Run, confirm red**

```bash
go test ./internal/api
```

Expected: compile error.

- [ ] **Step 4: Implement health handler**

Create `appview/internal/api/health.go`:

```go
// Package api holds HTTP handler factories that take narrow dependencies
// (not the full Deps struct).
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"social.craftsky/appview/internal/tap"
)

// Pinger matches *pgxpool.Pool's Ping signature without depending on pgx.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Stater returns Tap connection state. Matches tap.Consumer.State.
type Stater interface {
	State() tap.ConnState
}

type healthResponse struct {
	Status string         `json:"status"`
	DB     string         `json:"db"`
	Tap    healthTapBlock `json:"tap"`
}

type healthTapBlock struct {
	Connected        bool   `json:"connected"`
	LastEventAt      string `json:"last_event_at"`
	ReconnectAttempt int    `json:"reconnect_attempt"`
	LastError        string `json:"last_error"`
}

// NewHealthHandler returns a handler for GET /healthz.
func NewHealthHandler(pinger Pinger, stater Stater) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		if err := pinger.Ping(r.Context()); err != nil {
			dbStatus = "error"
		}
		tapState := stater.State()

		resp := healthResponse{
			DB: dbStatus,
			Tap: healthTapBlock{
				Connected:        tapState.Connected,
				ReconnectAttempt: tapState.ReconnectAttempt,
				LastError:        tapState.LastError,
			},
		}
		if !tapState.LastEventAt.IsZero() {
			resp.Tap.LastEventAt = tapState.LastEventAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		if dbStatus == "ok" && tapState.Connected {
			resp.Status = "ok"
		} else {
			resp.Status = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
```

- [ ] **Step 5: Run, confirm green**

```bash
go test ./internal/api
```

Expected: pass.

- [ ] **Step 6: Register the /healthz route**

In `appview/internal/routes/routes.go` (or equivalent), register the handler. The exact pattern depends on the scaffold; likely:

```go
mux.Handle("GET /healthz", api.NewHealthHandler(deps.DB, deps.Consumer))
```

(Note: `*pgxpool.Pool` already has a `Ping(ctx)` method, so it satisfies `Pinger`. `tap.Consumer.State()` satisfies `Stater`.)

- [ ] **Step 7: Build and verify**

```bash
go build ./...
go test ./...
```

All green.

- [ ] **Step 8: Commit**

```bash
git add appview/internal/api/ appview/internal/routes/
git commit -m "feat(api): add /healthz endpoint reporting db + tap state"
```

### Task 4.4: Wire consumer into server.go + real indexer into Deps

**Files:**
- Modify: `appview/cmd/appview/server.go`
- Modify: `appview/internal/app/deps.go`

- [ ] **Step 1: Swap `index.NotImplemented` for `BlueskyPostsSample` in deps.go**

In `newDeps`:

```go
indexerImpl := index.NewBlueskyPostsSample(pool)
```

…and use `indexerImpl` wherever `index.NotImplemented{}` was used.

- [ ] **Step 2: Start the consumer goroutine in server.go**

Edit `cmd/appview/server.go`. After `NewServer` is called and before `ListenAndServe`, spin up the consumer:

```go
consumerCtx, consumerCancel := context.WithCancel(ctx)
go func() {
	if err := deps.Consumer.Run(consumerCtx); err != nil && !errors.Is(err, context.Canceled) {
		deps.Logger.Error("tap consumer exited", slog.Any("err", err))
	}
}()
defer consumerCancel()
```

Integrate with the existing shutdown handler so `consumerCancel()` fires when the server receives SIGTERM/SIGINT.

- [ ] **Step 3: Build + test**

```bash
cd /Users/douglastodd/Projects/craftsky/appview
go build ./...
go test ./...
```

All green.

- [ ] **Step 4: Commit**

```bash
git add appview/cmd/appview/server.go appview/internal/app/deps.go
git commit -m "feat(appview): start tap consumer alongside HTTP server"
```

### Task 4.5: cli tap status subcommand

**Files:**
- Create: `appview/cmd/cli/tap.go`
- Create: `appview/cmd/cli/tap_test.go`
- Modify: `appview/cmd/cli/main.go` (register `tap` subcommand)

- [ ] **Step 1: Write failing test**

Create `appview/cmd/cli/tap_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTapStatusExitConnected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","db":"ok","tap":{"connected":true,"last_event_at":"2026-04-17T14:23:11Z","reconnect_attempt":0,"last_error":""}}`))
	}))
	defer srv.Close()

	code := tapStatus(srv.URL, nil)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestTapStatusExitDisconnected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"degraded","db":"ok","tap":{"connected":false,"last_event_at":"","reconnect_attempt":3,"last_error":"dial tcp: ..."}}`))
	}))
	defer srv.Close()

	code := tapStatus(srv.URL, nil)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestTapStatusExitTransport(t *testing.T) {
	// Point at a closed port.
	code := tapStatus("http://127.0.0.1:1", nil)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestTapStatusExitGarbageBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	code := tapStatus(srv.URL, nil)
	if code != 2 {
		t.Errorf("exit code = %d, want 2 (parse error)", code)
	}
}
```

- [ ] **Step 2: Run, confirm red**

```bash
go test ./cmd/cli
```

Expected: compile error.

- [ ] **Step 3: Implement `cli tap status`**

Create `appview/cmd/cli/tap.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var tapCmd = &cobra.Command{
	Use:   "tap",
	Short: "Tap consumer operations",
}

var tapStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print Tap consumer status from the running appview",
	Run: func(cmd *cobra.Command, args []string) {
		url := os.Getenv("APPVIEW_URL")
		if url == "" {
			url = "http://localhost:8080"
		}
		os.Exit(tapStatus(url, cmd.OutOrStdout()))
	},
}

func init() {
	tapCmd.AddCommand(tapStatusCmd)
}

type healthTap struct {
	Connected        bool   `json:"connected"`
	LastEventAt      string `json:"last_event_at"`
	ReconnectAttempt int    `json:"reconnect_attempt"`
	LastError        string `json:"last_error"`
}

type healthDoc struct {
	Tap healthTap `json:"tap"`
}

// tapStatus fetches /healthz and prints the tap block. Returns the shell
// exit code: 0 connected, 1 disconnected, 2 transport/parse error.
func tapStatus(baseURL string, out io.Writer) int {
	if out == nil {
		out = os.Stdout
	}
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/healthz")
	if err != nil {
		fmt.Fprintf(out, "transport error: %v\n", err)
		return 2
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(out, "read error: %v\n", err)
		return 2
	}
	var doc healthDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		fmt.Fprintf(out, "parse error: %v\n", err)
		return 2
	}

	fmt.Fprintf(out, "connected:         %t\n", doc.Tap.Connected)
	fmt.Fprintf(out, "last_event_at:     %s%s\n", doc.Tap.LastEventAt, relSuffix(doc.Tap.LastEventAt))
	fmt.Fprintf(out, "reconnect_attempt: %d\n", doc.Tap.ReconnectAttempt)
	if doc.Tap.LastError != "" {
		fmt.Fprintf(out, "last_error:        %s\n", doc.Tap.LastError)
	}

	if doc.Tap.Connected {
		return 0
	}
	return 1
}

func relSuffix(ts string) string {
	if ts == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(" (%s ago)", time.Since(t).Round(time.Second))
}
```

- [ ] **Step 4: Register in main.go**

In `appview/cmd/cli/main.go`:

```go
rootCmd.AddCommand(tapCmd)
```

- [ ] **Step 5: Run tests**

```bash
go test ./cmd/cli
```

Expected: all three pass.

- [ ] **Step 6: Commit**

```bash
git add appview/cmd/cli/tap.go appview/cmd/cli/tap_test.go appview/cmd/cli/main.go
git commit -m "feat(cli): add tap status subcommand"
```

### Task 4.6: Verify end-to-end locally

**Files:** None — verification task.

- [ ] **Step 1: Rebuild and cold-start**

```bash
cd /Users/douglastodd/Projects/craftsky
just down
docker volume rm craftsky_pgdata craftsky_tapdata 2>/dev/null || true
just dev
```

- [ ] **Step 2: Verify /healthz**

In another terminal:

```bash
curl -sS localhost:8080/healthz | jq
```

Expected: JSON with `status: "ok"`, `db: "ok"`, `tap.connected: true` (may take up to 60s after cold start).

- [ ] **Step 3: Wait for rows**

Wait a few minutes. Then:

```bash
just psql
```

```sql
SELECT count(*) FROM bluesky_posts_sample;
SELECT DISTINCT did FROM bluesky_posts_sample;
SELECT record FROM bluesky_posts_sample LIMIT 1;
```

Expected:
- Count may be 0 immediately after cold start if no tracked account has posted during backfill + startup. If count is 0 after 10 min, manually post from `@dougtodd.dev` on Bluesky and wait 30s.
- `DISTINCT did` returns ONLY the four allowlisted DIDs. If it returns others, Tap is not filtering correctly — inspect tap logs.
- The record column contains valid JSON with `"$type": "app.bsky.feed.post"`, `"text": "..."`, etc.

- [ ] **Step 4: Verify `just tap-status`**

```bash
just tap-status
```

Expected: `connected: true`, recent `last_event_at`, `reconnect_attempt: 0`.

- [ ] **Step 5: Verify degradation**

```bash
docker compose stop tap
sleep 20
curl -sS localhost:8080/healthz | jq
```

Expected: `status: "degraded"`, `tap.connected: false`, HTTP 200.

```bash
docker compose start tap
sleep 30
curl -sS localhost:8080/healthz | jq
```

Expected: `status: "ok"` again.

- [ ] **Step 6: No commit** — verification only. If any step fails, fix the underlying bug and commit as `fix(...)`.

---

## Chunk 5: Documentation and final verification

Goal: Update all docs to match the new reality, mark the scaffold spec partially superseded, then walk through every acceptance criterion.

### Task 5.1: Update appview/README.md

**Files:**
- Modify: `appview/README.md`

- [ ] **Step 1: Apply edits**

Update `appview/README.md`:

- In the repo-layout tree: replace `│   ├── firehose/            # Subscriber interface (stub on day one)` with `│   ├── tap/                 # WebSocket-with-acks consumer for the Tap sidecar`.
- Under "Binaries → `cmd/cli`", remove the `cli firehose replay` and `cli backfill` entries; add:

  ```
  cli tap status --env dev                  # print tap connection state
  ```

- Replace the whole "Development" section with a pointer to the root README/justfile for the compose workflow, plus `just test` and `just fmt` for common local ops. Remove the bare `docker run ... postgres:16` and `go run ./cmd/appview dev` — those paths are deprecated.
- Remove the "future `make` targets" paragraph at the end.

- [ ] **Step 2: Commit**

```bash
git add appview/README.md
git commit -m "docs(appview): update README for tap integration and just workflow"
```

### Task 5.2: Update root README.md and AGENTS.md

**Files:**
- Modify: `README.md` (repo root)
- Modify: `AGENTS.md`

- [ ] **Step 1: Add "Getting started" to root README**

Add a section to `README.md`:

```markdown
## Getting started

Prerequisites:

- Docker (with Docker Compose v2)
- [`just`](https://just.systems) (`brew install just` on macOS)

Clone and run:

```
git clone <repo>
cd craftsky
just dev
```

Once the stack is up (~60s on a cold start), visit `http://localhost:8080/healthz` — you should see `{"status":"ok",...}`. Post-indexing verification:

```
just tap-status
just psql   # then: SELECT count(*) FROM bluesky_posts_sample;
```

See `appview/README.md` for the full list of `just` recipes.
```

- [ ] **Step 2: Update AGENTS.md**

Insert a "Dev workflow" subsection near the top, under "Project Overview":

```markdown
## Dev Workflow

`just dev` (from the repo root) starts the full compose stack: `postgres`, `migrate`, `tap`, `tap-bootstrap`, `appview`. See `justfile` for all recipes. No `go run`; the appview runs only inside Docker in dev.
```

Update the repo layout table: add rows for the new top-level files.

| `docker-compose.yml` | Full local-dev stack: postgres + migrate + tap + tap-bootstrap + appview |
| `justfile` | Task runner recipes (`just dev`, `just test`, `just psql`, etc.) |

- [ ] **Step 3: Commit**

```bash
git add README.md AGENTS.md
git commit -m "docs: add getting-started instructions and dev workflow section"
```

### Task 5.3: Mark scaffold spec partially superseded

**Files:**
- Modify: `docs/superpowers/specs/2026-04-16-appview-server-scaffold-design.md`

- [ ] **Step 1: Add status line**

At the very top of the file, after the title, insert:

```markdown
**Status:** partially superseded by [`2026-04-17-tap-integration-design.md`](./2026-04-17-tap-integration-design.md) — the `firehose` package, `index.Indexer.Backfill`, and `cli firehose replay` / `cli backfill` subcommands described here are removed by that spec. Everything else (HTTP routing, middleware, `Deps` pattern, auth stubs, CLI request/ping/migrate/did-resolve) remains load-bearing.
```

- [ ] **Step 2: Commit**

```bash
git add docs/superpowers/specs/2026-04-16-appview-server-scaffold-design.md
git commit -m "docs: mark scaffold spec partially superseded by tap integration"
```

### Task 5.4: Walk the 12 acceptance criteria

**Files:** None — final gate.

- [ ] **Step 1: Fresh stack**

```bash
cd /Users/douglastodd/Projects/craftsky
just down
docker volume rm craftsky_pgdata craftsky_tapdata 2>/dev/null || true
just dev-d
```

- [ ] **Step 2: AC #1 — All services healthy**

```bash
sleep 60
docker compose ps
```

Expected: `postgres`, `tap`, `appview` show `healthy`; `migrate` and `tap-bootstrap` have exited 0.

- [ ] **Step 3: AC #2 — /healthz 200 with tap.connected: true**

```bash
curl -fsS localhost:8080/healthz | jq .
```

Expected: `status: "ok"`, `tap.connected: true`, within 60s of cold start.

- [ ] **Step 4: AC #3 — migrations applied, idempotent**

Already verified by AC #1 (migrate exited 0). Re-run:

```bash
just migrate up
```

Expected: no errors, prints "no change" or similar.

- [ ] **Step 5: AC #4 — rows appear in bluesky_posts_sample**

Wait up to 10 minutes. Then:

```bash
just psql
```

```sql
SELECT count(*) FROM bluesky_posts_sample;
SELECT record FROM bluesky_posts_sample LIMIT 1;
```

If count is 0 after 10 min, post from `@dougtodd.dev` and wait 30s, then re-check.

- [ ] **Step 6: AC #5 — idempotency across restart**

```bash
docker compose stop appview
sleep 5
docker compose start appview
docker compose logs appview | tail -50
```

Expected: no duplicate-key errors; count in `SELECT count(*)` resumes climbing.

- [ ] **Step 7: AC #6 — graceful degradation**

```bash
docker compose stop tap
sleep 15
curl -fsS localhost:8080/healthz | jq .status,.tap.connected
docker compose start tap
sleep 35
curl -fsS localhost:8080/healthz | jq .status,.tap.connected
```

Expected: `"degraded", false` then `"ok", true`.

- [ ] **Step 8: AC #7 — all Go tests pass**

```bash
just test
```

Expected: all tests pass.

- [ ] **Step 9: AC #8 — fmt + vet clean**

```bash
just fmt
git status   # expect no unstaged diff
```

Expected: no changes. Any diff indicates code wasn't formatted before committing — fix and commit as `style: run gofmt`.

- [ ] **Step 10: AC #9 — cli tap status exits 0**

```bash
just tap-status
echo "exit=$?"
```

Expected: output shows `connected: true`; echo shows `exit=0`.

- [ ] **Step 11: AC #10 — cold-contributor test**

Close your eyes. Pretend you've never seen this repo. Open `README.md`. Does it get you from zero to AC #2? If yes, ✓. If no, fix the README.

- [ ] **Step 12: AC #11 — delete verification**

Pick a URI from `bluesky_posts_sample` authored by `@dougtodd.dev`. Delete that post on Bluesky (via the app / `https://bsky.app`). Wait 60 seconds.

```sql
SELECT count(*) FROM bluesky_posts_sample WHERE uri = '<the deleted URI>';
```

Expected: 0.

- [ ] **Step 13: AC #12 — DID-filter verification**

```sql
SELECT DISTINCT did FROM bluesky_posts_sample;
```

Expected: only the four allowlisted DIDs (from Appendix A). Any other DID is a blocker — inspect `TAP_COLLECTION_FILTERS` and `/repos/add` bootstrap.

- [ ] **Step 14: Tear down and commit**

```bash
just down
```

If AC #12 check passed and the output differed from the "expected" format in any minor way (e.g. one DID missing because the account hasn't posted), note it in the final commit.

- [ ] **Step 15: Final commit & PR prep**

```bash
git log --oneline feat/tap-integration ^main   # review the series
git push -u origin feat/tap-integration
```

Open a PR against `main`. Title: `feat(appview): integrate tap service for relay event ingestion`. Body should summarize the change, link both specs and this plan, and list the 12 ACs with ✓/✗ from the walk.

---

## Appendix A — Resolved DIDs and pinned Tap tag

Resolved 2026-04-17 via `com.atproto.identity.resolveHandle`:

- `@bsky.app` → `did:plc:z72i7hdynmk6r22z27h6tvur`
- `@jay.bsky.team` → `did:plc:oky5czdrnfjpqslsw2a5iclo`
- `@dougtodd.dev` → `did:plc:jt3cxdyrhjhrkpwzdvonheax`
- `@eurosky.social` → `did:plc:ooensn4mr5mhznzypvxelfa3`

Pinned Tap image: `ghcr.io/bluesky-social/indigo/tap:0.1.10` (digest `sha256:5e20bfe416d29fcd215ed8bf99f10b2ab825a6de4e5599846dd33967ade2abeb`).
