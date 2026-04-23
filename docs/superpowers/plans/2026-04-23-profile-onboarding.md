# Profile Onboarding & Endpoints Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the server-side profile experience from [`2026-04-23-profile-onboarding-design.md`](../specs/2026-04-23-profile-onboarding-design.md): two new indexers (`craftsky_profiles`, `bluesky_profiles`) with membership-gated Bluesky indexing, onboarding-on-login in the OAuth callback, and three `/v1/profiles/*` endpoints. Drops the throwaway `bluesky_posts_sample` indexer.

**Architecture:** Everything fits into the existing AppView Go service. Two new firehose indexers register via the `index.Dispatcher` pattern already in place. Onboarding happens synchronously in the OAuth callback handler: we resume the freshly-created session, `getRecord` the user's two profile records, write an empty `social.craftsky.actor.profile` if missing, then return the token. Three new HTTP handlers under `/v1/profiles/` serve reads and writes; the PUT handler does a read-before-write merge on the Bluesky side to preserve avatar/banner (which are read-only in v1).

**Tech Stack:** Go 1.22+, `pgx/v5`, `indigo` (`atproto/syntax`, `atproto/auth/oauth`, `atproto/atclient`, `atproto/identity`), `golang-migrate/v4`, stdlib `net/http`. Tests run via `just test` against the compose Postgres.

---

## Background reading for the implementer

Read these before starting. They're short but load-bearing.

- [`docs/superpowers/specs/2026-04-23-profile-onboarding-design.md`](../specs/2026-04-23-profile-onboarding-design.md) — the spec this plan implements. **Primary source of truth.**
- [`docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`](../specs/2026-04-21-appview-api-architecture-design.md) — `/v1/` URL conventions, `X-Craftsky-Device-Id`, error envelope, partial-success handling for PDS writes.
- [`docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`](../specs/2026-04-22-api-wire-alignment-design.md) — camelCase JSON key rule on `/v1/*`.
- [`docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md`](../specs/2026-04-18-appview-oauth-bff-design.md) §3.2–3.4 — callback flow, `ResumeSession`, `APIClient` pattern.
- [`lexicon/social/craftsky/actor/profile.json`](../../../lexicon/social/craftsky/actor/profile.json) — the record shape (unchanged).
- [`appview/internal/index/dispatcher.go`](../../../appview/internal/index/dispatcher.go) and [`bluesky_posts_sample.go`](../../../appview/internal/index/bluesky_posts_sample.go) — patterns for indexers and the sample we're replacing.
- [`appview/internal/api/whoami.go`](../../../appview/internal/api/whoami.go) and [`whoami_test.go`](../../../appview/internal/api/whoami_test.go) — handler + test patterns to copy.
- [`appview/internal/api/envelope/envelope.go`](../../../appview/internal/api/envelope/envelope.go) — error-response helper.
- [`appview/internal/api/handle_resolver.go`](../../../appview/internal/api/handle_resolver.go) — interface being extended.
- [`appview/internal/auth/handlers_oauth.go`](../../../appview/internal/auth/handlers_oauth.go) — callback being modified.
- [`AGENTS.md`](../../../AGENTS.md) — project rules. Coding conventions, Go toolchain, testing posture.

## Conventions this plan follows

- **TDD.** Every task writes the test first, confirms it fails, writes the minimal code to pass, confirms it passes, commits. This is rigid — don't batch.
- **One commit per task.** Tasks are small. Frequent commits make reverts cheap.
- **`just test` is the only test runner.** Requires `just dev-d` running so integration tests can hit the compose Postgres via `localhost:5433`.
- **`just fmt` after every non-trivial change.** Do not commit Go files that haven't been `gofmt`'d.
- **Naming.** camelCase for JSON fields on `/v1/*`, snake_case for error codes, UpperCamelCase for Go exports, snake_case for SQL identifiers. Matches what the codebase already does.
- **No emojis in code or commit messages.**

## File structure

All paths are relative to repo root.

**New files:**

- `appview/migrations/000007_drop_bluesky_posts_sample.up.sql`
- `appview/migrations/000007_drop_bluesky_posts_sample.down.sql`
- `appview/migrations/000008_craftsky_profiles.up.sql`
- `appview/migrations/000008_craftsky_profiles.down.sql`
- `appview/migrations/000009_bluesky_profiles.up.sql`
- `appview/migrations/000009_bluesky_profiles.down.sql`
- `appview/internal/index/craftsky_profile.go` — `CraftskyProfile` indexer.
- `appview/internal/index/craftsky_profile_test.go`
- `appview/internal/index/bluesky_profile.go` — `BlueskyProfile` indexer.
- `appview/internal/index/bluesky_profile_test.go`
- `appview/internal/auth/initialize_profile.go` — `InitializeProfile` function called from OAuth callback.
- `appview/internal/auth/initialize_profile_test.go`
- `appview/internal/api/profile.go` — three handler factories.
- `appview/internal/api/profile_request.go` — PUT request decoding + validation.
- `appview/internal/api/profile_response.go` — response type + avatar/banner URL synthesis.
- `appview/internal/api/profile_store.go` — Postgres reads/writes used by the profile handlers.
- `appview/internal/api/profile_test.go` — handler tests.
- `appview/internal/api/profile_store_test.go` — store tests.

**Modified files:**

- `appview/internal/api/handle_resolver.go` — typed with `syntax.DID`/`syntax.Handle`; add `ResolveDID`.
- `appview/internal/api/handle_resolver_test.go` — update signature; add `ResolveDID` tests.
- `appview/internal/api/whoami.go` — parse `syntax.DID` at call site.
- `appview/internal/api/whoami_test.go` — update `stubResolver` to new signature.
- `appview/internal/auth/handlers_oauth.go` — call `InitializeProfile` after `ProcessCallback`.
- `appview/internal/auth/handlers_test.go` — add test for initialisation-error → error page.
- `appview/internal/app/deps.go` — register new indexers, drop sample registration, expose handlers' deps.
- `appview/internal/app/deps_test.go` — adjust.
- `appview/internal/routes/routes.go` — register 3 new routes.
- `appview/internal/routes/routes_test.go` — add route assertions.
- `AGENTS.md` — short note on indexer extension pattern.

**Deleted files:**

- `appview/internal/index/bluesky_posts_sample.go`
- `appview/internal/index/bluesky_posts_sample_test.go`

## Chunk boundaries

- **Chunk 1:** Verify migration numbers and drop the sample indexer (code + migration).
- **Chunk 2:** New migrations for `craftsky_profiles` and `bluesky_profiles`.
- **Chunk 3:** `CraftskyProfile` indexer.
- **Chunk 4:** `BlueskyProfile` indexer (with membership gate + cascading delete wired from Chunk 3).
- **Chunk 5:** `HandleResolver` extension (typed, with `ResolveDID`).
- **Chunk 6:** `InitializeProfile` and callback integration.
- **Chunk 7:** Profile store (Postgres reads/writes) shared by the handlers.
- **Chunk 8:** `GET /v1/profiles/@{handleOrDid}` + `GET /v1/profiles/me` handlers.
- **Chunk 9:** `PUT /v1/profiles/me` handler (parallel writes, read-before-write merge).
- **Chunk 10:** Route registration + wiring in `deps.go` + AGENTS.md update.

---

## Chunk 1: Drop the sample indexer

The `bluesky_posts_sample` indexer was always explicitly temporary. Its comment said "MUST be deleted when the first social.craftsky.* indexer lands." We're about to add that indexer, so drop the sample now to keep the code surface clean.

### Task 1.1: Verify current migration head

**Files:**
- Inspect: `appview/migrations/`

- [ ] **Step 1:** List migrations to confirm the current head.

```bash
ls appview/migrations/
```

Expected: highest prefix is `000006`. If a new migration has landed since this plan was written, use the next-in-sequence numbers throughout — the sample-drop becomes `NNN+1`, `craftsky_profiles` becomes `NNN+2`, `bluesky_profiles` becomes `NNN+3`. Keep them contiguous.

### Task 1.2: Write the sample-drop migration

**Files:**
- Create: `appview/migrations/000007_drop_bluesky_posts_sample.up.sql`
- Create: `appview/migrations/000007_drop_bluesky_posts_sample.down.sql`

- [ ] **Step 1:** Create the up migration.

`appview/migrations/000007_drop_bluesky_posts_sample.up.sql`:

```sql
-- Drop the throwaway sample table that validated the Tap pipeline.
-- See docs/superpowers/specs/2026-04-23-profile-onboarding-design.md §3.4.
DROP TABLE IF EXISTS bluesky_posts_sample;
```

- [ ] **Step 2:** Create the down migration that restores the original schema.

`appview/migrations/000007_drop_bluesky_posts_sample.down.sql`:

```sql
-- Restore bluesky_posts_sample at its original shape.
-- Matches migration 000001_bluesky_posts_sample.up.sql.
CREATE TABLE bluesky_posts_sample (
    uri        TEXT PRIMARY KEY,
    cid        TEXT NOT NULL,
    did        TEXT NOT NULL,
    rkey       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    record     JSONB NOT NULL
);
CREATE INDEX ON bluesky_posts_sample (did);
```

Cross-check the columns/indexes against `appview/migrations/000001_bluesky_posts_sample.up.sql`. If the original differs, adjust the down migration to match it exactly — the down migration must reverse history faithfully.

- [ ] **Step 3:** Apply the migration.

```bash
just migrate up
```

Expected: `migration applied`.

- [ ] **Step 4:** Roll back to verify the down migration.

```bash
just migrate down 1
just migrate up
```

Expected: both succeed.

- [ ] **Step 5:** Commit.

```bash
git add appview/migrations/000007_drop_bluesky_posts_sample.up.sql \
        appview/migrations/000007_drop_bluesky_posts_sample.down.sql
git commit -m "db: drop bluesky_posts_sample in prep for profile indexers"
```

### Task 1.3: Remove the sample indexer code

**Files:**
- Delete: `appview/internal/index/bluesky_posts_sample.go`
- Delete: `appview/internal/index/bluesky_posts_sample_test.go`
- Modify: `appview/internal/app/deps.go`

- [ ] **Step 1:** Delete the indexer files.

```bash
git rm appview/internal/index/bluesky_posts_sample.go \
       appview/internal/index/bluesky_posts_sample_test.go
```

- [ ] **Step 2:** Remove the sample registration from `deps.go`.

In `appview/internal/app/deps.go`, locate:

```go
blueskySample := index.NewBlueskyPostsSample(pool)

dispatcher := index.NewDispatcher(index.NotImplemented{})
dispatcher.Register("app.bsky.feed.post", blueskySample)
```

Replace with (we'll register real indexers in Chunk 10):

```go
dispatcher := index.NewDispatcher(index.NotImplemented{})
```

- [ ] **Step 3:** Verify `go build` still passes.

```bash
cd appview && go build ./... && cd -
```

Expected: no errors.

- [ ] **Step 4:** Run the full test suite to confirm nothing broke.

```bash
just test
```

Expected: all tests pass. (The sample's own tests are gone; nothing else referenced it.)

- [ ] **Step 5:** Commit.

```bash
git add appview/internal/app/deps.go
git commit -m "index: remove bluesky_posts_sample indexer"
```

---

## Chunk 2: Profile-table migrations

Two migrations, one per table. Apply in numerical order.

### Task 2.1: `craftsky_profiles` migration

**Files:**
- Create: `appview/migrations/000008_craftsky_profiles.up.sql`
- Create: `appview/migrations/000008_craftsky_profiles.down.sql`

- [ ] **Step 1:** Write the up migration.

```sql
-- appview/migrations/000008_craftsky_profiles.up.sql
-- See docs/superpowers/specs/2026-04-23-profile-onboarding-design.md §2.1.
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

- [ ] **Step 2:** Write the down migration.

```sql
-- appview/migrations/000008_craftsky_profiles.down.sql
DROP TABLE IF EXISTS craftsky_profiles;
```

- [ ] **Step 3:** Apply the migration.

```bash
just migrate up
```

- [ ] **Step 4:** Verify schema.

```bash
just psql -c '\d craftsky_profiles'
```

Expected output includes columns `did`, `crafts`, `record_cid`, `indexed_at`, `created_at` with the declared types; `did` marked as primary key; `crafts` default `'{}'`.

- [ ] **Step 5:** Roll back and re-apply to verify the down migration.

```bash
just migrate down 1
just migrate up
```

Expected: both succeed.

- [ ] **Step 6:** Commit.

```bash
git add appview/migrations/000008_craftsky_profiles.up.sql \
        appview/migrations/000008_craftsky_profiles.down.sql
git commit -m "db: add craftsky_profiles table"
```

### Task 2.2: `bluesky_profiles` migration

**Files:**
- Create: `appview/migrations/000009_bluesky_profiles.up.sql`
- Create: `appview/migrations/000009_bluesky_profiles.down.sql`

- [ ] **Step 1:** Write the up migration.

```sql
-- appview/migrations/000009_bluesky_profiles.up.sql
-- See docs/superpowers/specs/2026-04-23-profile-onboarding-design.md §2.2.
CREATE TABLE bluesky_profiles (
    did          TEXT        NOT NULL PRIMARY KEY,
    display_name TEXT,
    description  TEXT,
    avatar_cid   TEXT,
    avatar_mime  TEXT,
    banner_cid   TEXT,
    banner_mime  TEXT,
    record_cid   TEXT        NOT NULL,
    indexed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

- [ ] **Step 2:** Write the down migration.

```sql
-- appview/migrations/000009_bluesky_profiles.down.sql
DROP TABLE IF EXISTS bluesky_profiles;
```

- [ ] **Step 3:** Apply.

```bash
just migrate up
```

- [ ] **Step 4:** Verify schema.

```bash
just psql -c '\d bluesky_profiles'
```

Expected: nine columns; `did` primary key; `display_name`, `description`, `avatar_cid`, `avatar_mime`, `banner_cid`, `banner_mime` nullable.

- [ ] **Step 5:** Roll back and re-apply.

```bash
just migrate down 1
just migrate up
```

- [ ] **Step 6:** Commit.

```bash
git add appview/migrations/000009_bluesky_profiles.up.sql \
        appview/migrations/000009_bluesky_profiles.down.sql
git commit -m "db: add bluesky_profiles table"
```

---

## Chunk 3: `CraftskyProfile` indexer

Handles `social.craftsky.actor.profile` firehose events. Follow the test scaffolding pattern from `bluesky_posts_sample_test.go` (before it's deleted? — it's already deleted in Chunk 1, so copy the `withSchema` helper from git history, or re-create an equivalent helper inline in the new test file).

### Task 3.1: Re-create the `withSchema` test helper

**Files:**
- Create: `appview/internal/index/testhelpers_test.go`

- [ ] **Step 1:** Add a per-test schema helper. This replaces the one lost with the sample test file.

```go
// appview/internal/index/testhelpers_test.go
package index_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// withSchema creates an isolated schema for one test and returns a pool
// whose default search_path points at it. Dropped via t.Cleanup.
// ddlStatements is run inside the fresh schema before the pool is returned.
func withSchema(t *testing.T, ddlStatements string) *pgxpool.Pool {
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

	if ddlStatements != "" {
		if _, err := pool.Exec(ctx, ddlStatements); err != nil {
			t.Fatalf("create test tables: %v", err)
		}
	}
	return pool
}
```

- [ ] **Step 2:** Run it to confirm the file compiles on its own.

```bash
cd appview && go vet ./internal/index/... && cd -
```

Expected: no errors. (`go vet` is enough to flag syntax; full coverage comes when we use the helper next.)

- [ ] **Step 3:** Commit.

```bash
git add appview/internal/index/testhelpers_test.go
git commit -m "test: add per-schema index test helper"
```

### Task 3.2: Write the failing `create`/`update` test

**Files:**
- Create: `appview/internal/index/craftsky_profile_test.go`

- [ ] **Step 1:** Write the test.

```go
// appview/internal/index/craftsky_profile_test.go
package index_test

import (
	"context"
	"encoding/json"
	"testing"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
)

const craftskyProfilesDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE bluesky_profiles (
    did          TEXT        NOT NULL PRIMARY KEY,
    display_name TEXT,
    description  TEXT,
    avatar_cid   TEXT,
    avatar_mime  TEXT,
    banner_cid   TEXT,
    banner_mime  TEXT,
    record_cid   TEXT        NOT NULL,
    indexed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func TestCraftskyProfile_Create(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool)

	ev := tap.Event{
		URI:        "at://did:plc:abc/social.craftsky.actor.profile/self",
		CID:        "bafy1",
		DID:        "did:plc:abc",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["knitting","sewing"]}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var crafts []string
	var cid string
	err := pool.QueryRow(context.Background(),
		`SELECT crafts, record_cid FROM craftsky_profiles WHERE did = $1`, ev.DID).
		Scan(&crafts, &cid)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if cid != "bafy1" {
		t.Errorf("record_cid = %q, want bafy1", cid)
	}
	if len(crafts) != 2 || crafts[0] != "knitting" || crafts[1] != "sewing" {
		t.Errorf("crafts = %v, want [knitting sewing]", crafts)
	}
}
```

- [ ] **Step 2:** Run it to confirm it fails.

```bash
just test
```

Expected: `go build` error — `undefined: index.NewCraftskyProfile`. Good.

### Task 3.3: Write the minimal `CraftskyProfile` indexer

**Files:**
- Create: `appview/internal/index/craftsky_profile.go`

- [ ] **Step 1:** Write the indexer type + `NewCraftskyProfile` constructor + the `create`/`update` branch of `Handle`.

```go
// appview/internal/index/craftsky_profile.go
package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

// CraftskyProfile indexes social.craftsky.actor.profile events into the
// craftsky_profiles table. Required invariant: idempotent on (DID, CID).
// Tap delivers events at least once.
//
// A delete cascades into bluesky_profiles — the user leaving Craftsky
// (by deleting their social.craftsky.actor.profile record) removes their
// Bluesky mirror, since membership is defined by presence in
// craftsky_profiles.
type CraftskyProfile struct {
	pool *pgxpool.Pool
}

var _ Indexer = (*CraftskyProfile)(nil)

// NewCraftskyProfile builds an indexer backed by the given pool.
func NewCraftskyProfile(pool *pgxpool.Pool) *CraftskyProfile {
	return &CraftskyProfile{pool: pool}
}

const craftskyProfileNSID = "social.craftsky.actor.profile"

// craftskyProfileRecord mirrors the subset of social.craftsky.actor.profile
// that the indexer cares about.
type craftskyProfileRecord struct {
	Crafts []string `json:"crafts"`
}

func (c *CraftskyProfile) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != craftskyProfileNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		var rec craftskyProfileRecord
		if err := json.Unmarshal(ev.Record, &rec); err != nil {
			return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
		}
		// Defensive: TEXT[] NOT NULL in the column; never let a nil slice go in.
		if rec.Crafts == nil {
			rec.Crafts = []string{}
		}
		const q = `
			INSERT INTO craftsky_profiles (did, crafts, record_cid)
			VALUES ($1, $2, $3)
			ON CONFLICT (did) DO UPDATE SET
				crafts = EXCLUDED.crafts,
				record_cid = EXCLUDED.record_cid,
				indexed_at = now()
			WHERE craftsky_profiles.record_cid IS DISTINCT FROM EXCLUDED.record_cid
		`
		if _, err := c.pool.Exec(ctx, q, ev.DID, rec.Crafts, ev.CID); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		return c.handleDelete(ctx, ev.DID)
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

// handleDelete removes the craftsky_profiles row and its bluesky_profiles
// mirror in a single transaction. See spec §3.1.
func (c *CraftskyProfile) handleDelete(ctx context.Context, did string) error {
	return pgx.BeginFunc(ctx, c.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`DELETE FROM craftsky_profiles WHERE did = $1`, did); err != nil {
			return fmt.Errorf("delete craftsky_profiles %s: %w", did, err)
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM bluesky_profiles WHERE did = $1`, did); err != nil {
			return fmt.Errorf("delete bluesky_profiles %s: %w", did, err)
		}
		return nil
	})
}
```

- [ ] **Step 2:** Run the test.

```bash
just test -run TestCraftskyProfile_Create ./internal/index/...
```

Expected: PASS.

- [ ] **Step 3:** `just fmt`.

```bash
just fmt
```

- [ ] **Step 4:** Commit.

```bash
git add appview/internal/index/craftsky_profile.go \
        appview/internal/index/craftsky_profile_test.go
git commit -m "index: add CraftskyProfile indexer with create support"
```

### Task 3.4: Add tests for update + idempotency

**Files:**
- Modify: `appview/internal/index/craftsky_profile_test.go`

- [ ] **Step 1:** Append tests.

```go
func TestCraftskyProfile_UpdateReplacesCrafts(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool)

	create := tap.Event{
		URI:        "at://did:plc:x/social.craftsky.actor.profile/self",
		CID:        "c1",
		DID:        "did:plc:x",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["knitting"]}`),
	}
	update := create
	update.CID = "c2"
	update.Action = "update"
	update.Record = json.RawMessage(`{"crafts":["knitting","quilting"]}`)

	if err := idx.Handle(context.Background(), create); err != nil {
		t.Fatal(err)
	}
	if err := idx.Handle(context.Background(), update); err != nil {
		t.Fatal(err)
	}

	var crafts []string
	var cid string
	_ = pool.QueryRow(context.Background(),
		`SELECT crafts, record_cid FROM craftsky_profiles WHERE did = $1`, create.DID).
		Scan(&crafts, &cid)
	if cid != "c2" {
		t.Errorf("record_cid = %q, want c2", cid)
	}
	if len(crafts) != 2 || crafts[1] != "quilting" {
		t.Errorf("crafts = %v, want [knitting quilting]", crafts)
	}
}

func TestCraftskyProfile_ReplayedEventPreservesIndexedAt(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool)

	ev := tap.Event{
		URI:        "at://did:plc:y/social.craftsky.actor.profile/self",
		CID:        "c1",
		DID:        "did:plc:y",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["crochet"]}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}

	var firstIndexedAt string
	_ = pool.QueryRow(context.Background(),
		`SELECT indexed_at::text FROM craftsky_profiles WHERE did = $1`, ev.DID).
		Scan(&firstIndexedAt)

	// Re-deliver identical event.
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}

	var secondIndexedAt string
	_ = pool.QueryRow(context.Background(),
		`SELECT indexed_at::text FROM craftsky_profiles WHERE did = $1`, ev.DID).
		Scan(&secondIndexedAt)

	if firstIndexedAt != secondIndexedAt {
		t.Errorf("indexed_at changed on replay: %q -> %q", firstIndexedAt, secondIndexedAt)
	}
}
```

- [ ] **Step 2:** Run the tests.

```bash
just test -run TestCraftskyProfile ./internal/index/...
```

Expected: all PASS. The idempotency test relies on the `WHERE record_cid IS DISTINCT FROM` guard in the SQL.

- [ ] **Step 3:** Commit.

```bash
git add appview/internal/index/craftsky_profile_test.go
git commit -m "test: add CraftskyProfile update and idempotency coverage"
```

### Task 3.5: Add tests for delete cascade + unknown-action error

**Files:**
- Modify: `appview/internal/index/craftsky_profile_test.go`

- [ ] **Step 1:** Append tests.

```go
func TestCraftskyProfile_DeleteRemovesBothRows(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool)
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:z/social.craftsky.actor.profile/self",
		CID:        "c1",
		DID:        "did:plc:z",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["sewing"]}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}
	// Seed a bluesky_profiles row for the same DID.
	if _, err := pool.Exec(ctx,
		`INSERT INTO bluesky_profiles (did, display_name, record_cid) VALUES ($1, $2, $3)`,
		create.DID, "alice", "bskyCID"); err != nil {
		t.Fatal(err)
	}

	del := tap.Event{
		URI: create.URI, DID: create.DID, Rkey: "self",
		Collection: "social.craftsky.actor.profile", Action: "delete",
	}
	if err := idx.Handle(ctx, del); err != nil {
		t.Fatalf("delete Handle: %v", err)
	}

	var crCount, bsCount int
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM craftsky_profiles WHERE did = $1`, del.DID).Scan(&crCount)
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM bluesky_profiles WHERE did = $1`, del.DID).Scan(&bsCount)
	if crCount != 0 || bsCount != 0 {
		t.Errorf("post-delete counts = (craftsky:%d, bluesky:%d), want (0,0)", crCount, bsCount)
	}
}

func TestCraftskyProfile_UnknownAction(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool)
	ev := tap.Event{
		URI:        "at://did:plc:a/social.craftsky.actor.profile/self",
		CID:        "c",
		DID:        "did:plc:a",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "weird",
		Record:     json.RawMessage(`{"crafts":[]}`),
	}
	if err := idx.Handle(context.Background(), ev); err == nil {
		t.Error("want error for unknown action; got nil")
	}
}

func TestCraftskyProfile_OtherCollectionIgnored(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool)
	ev := tap.Event{
		URI: "at://did:plc:b/app.bsky.feed.post/k", CID: "c", DID: "did:plc:b", Rkey: "k",
		Collection: "app.bsky.feed.post", Action: "create",
		Record: json.RawMessage(`{}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("want nil for other collection; got %v", err)
	}
}
```

- [ ] **Step 2:** Run the tests.

```bash
just test -run TestCraftskyProfile ./internal/index/...
```

Expected: all PASS.

- [ ] **Step 3:** Commit.

```bash
git add appview/internal/index/craftsky_profile_test.go
git commit -m "test: add CraftskyProfile delete-cascade and action coverage"
```

---

## Chunk 4: `BlueskyProfile` indexer

Mirrors the structure of Chunk 3. Key differences: (a) membership-gated on `craftsky_profiles`, (b) parses the much richer `app.bsky.actor.profile` record including nested blob refs.

### Task 4.1: Write the failing `create` test with membership seeded

**Files:**
- Create: `appview/internal/index/bluesky_profile_test.go`

- [ ] **Step 1:** Write the test.

```go
// appview/internal/index/bluesky_profile_test.go
package index_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
)

// Reuses craftskyProfilesDDL from craftsky_profile_test.go.

func seedMember(t *testing.T, pool *pgxpool.Pool, did string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
		did, "seed"); err != nil {
		t.Fatal(err)
	}
}

func TestBlueskyProfile_CreateForMember(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	seedMember(t, pool, "did:plc:m")
	idx := index.NewBlueskyProfile(pool)

	ev := tap.Event{
		URI:        "at://did:plc:m/app.bsky.actor.profile/self",
		CID:        "bafbluesky",
		DID:        "did:plc:m",
		Rkey:       "self",
		Collection: "app.bsky.actor.profile",
		Action:     "create",
		Record: json.RawMessage(`{
			"displayName": "Mallory",
			"description": "sews things",
			"avatar":   {"$type":"blob","ref":{"$link":"bafkavatar"},"mimeType":"image/jpeg","size":1},
			"banner":   {"$type":"blob","ref":{"$link":"bafkbanner"},"mimeType":"image/png","size":1}
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var displayName, description, avatarCID, avatarMime, bannerCID, bannerMime, recordCID string
	err := pool.QueryRow(context.Background(), `
		SELECT display_name, description, avatar_cid, avatar_mime,
		       banner_cid, banner_mime, record_cid
		FROM bluesky_profiles WHERE did = $1`, ev.DID).
		Scan(&displayName, &description, &avatarCID, &avatarMime,
			&bannerCID, &bannerMime, &recordCID)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if displayName != "Mallory" {
		t.Errorf("display_name = %q", displayName)
	}
	if description != "sews things" {
		t.Errorf("description = %q", description)
	}
	if avatarCID != "bafkavatar" || avatarMime != "image/jpeg" {
		t.Errorf("avatar = (%q, %q)", avatarCID, avatarMime)
	}
	if bannerCID != "bafkbanner" || bannerMime != "image/png" {
		t.Errorf("banner = (%q, %q)", bannerCID, bannerMime)
	}
	if recordCID != "bafbluesky" {
		t.Errorf("record_cid = %q", recordCID)
	}
}

func TestBlueskyProfile_DropsForNonMember(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewBlueskyProfile(pool)

	ev := tap.Event{
		URI:        "at://did:plc:nm/app.bsky.actor.profile/self",
		CID:        "c",
		DID:        "did:plc:nm",
		Rkey:       "self",
		Collection: "app.bsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"displayName":"bob"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("Handle should drop non-members without error; got %v", err)
	}
	var count int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM bluesky_profiles WHERE did = $1`, ev.DID).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 (non-member must not be indexed)", count)
	}
}
```

- [ ] **Step 2:** Run, confirm it fails (`undefined: index.NewBlueskyProfile`).

```bash
just test -run TestBlueskyProfile ./internal/index/...
```

Expected: build error or FAIL.

### Task 4.2: Write the `BlueskyProfile` indexer

**Files:**
- Create: `appview/internal/index/bluesky_profile.go`

- [ ] **Step 1:** Write the indexer.

```go
// appview/internal/index/bluesky_profile.go
package index

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

// BlueskyProfile indexes app.bsky.actor.profile events into the
// bluesky_profiles table, gated on Craftsky membership (presence in
// craftsky_profiles). Required invariant: idempotent on (DID, CID).
type BlueskyProfile struct {
	pool *pgxpool.Pool
}

var _ Indexer = (*BlueskyProfile)(nil)

// NewBlueskyProfile builds an indexer backed by the given pool.
func NewBlueskyProfile(pool *pgxpool.Pool) *BlueskyProfile {
	return &BlueskyProfile{pool: pool}
}

const blueskyProfileNSID = "app.bsky.actor.profile"

// blueskyBlobRef is the atproto blob-reference shape carried inside an
// app.bsky.actor.profile record. We only need the CID link and MIME type.
type blueskyBlobRef struct {
	Ref      struct {
		Link string `json:"$link"`
	} `json:"ref"`
	MimeType string `json:"mimeType"`
}

type blueskyProfileRecord struct {
	DisplayName *string         `json:"displayName,omitempty"`
	Description *string         `json:"description,omitempty"`
	Avatar      *blueskyBlobRef `json:"avatar,omitempty"`
	Banner      *blueskyBlobRef `json:"banner,omitempty"`
}

func (b *BlueskyProfile) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != blueskyProfileNSID {
		return nil
	}
	isMember, err := b.isMember(ctx, ev.DID)
	if err != nil {
		return fmt.Errorf("membership check %s: %w", ev.DID, err)
	}
	if !isMember {
		// Drop silently — the user isn't on Craftsky, so we don't mirror
		// their Bluesky profile.
		return nil
	}

	switch ev.Action {
	case "create", "update":
		var rec blueskyProfileRecord
		if err := json.Unmarshal(ev.Record, &rec); err != nil {
			return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
		}
		const q = `
			INSERT INTO bluesky_profiles
				(did, display_name, description,
				 avatar_cid, avatar_mime, banner_cid, banner_mime, record_cid)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (did) DO UPDATE SET
				display_name = EXCLUDED.display_name,
				description  = EXCLUDED.description,
				avatar_cid   = EXCLUDED.avatar_cid,
				avatar_mime  = EXCLUDED.avatar_mime,
				banner_cid   = EXCLUDED.banner_cid,
				banner_mime  = EXCLUDED.banner_mime,
				record_cid   = EXCLUDED.record_cid,
				indexed_at   = now()
			WHERE bluesky_profiles.record_cid IS DISTINCT FROM EXCLUDED.record_cid
		`
		var (
			avatarCID, avatarMime *string
			bannerCID, bannerMime *string
		)
		if rec.Avatar != nil && rec.Avatar.Ref.Link != "" {
			avatarCID = &rec.Avatar.Ref.Link
			avatarMime = &rec.Avatar.MimeType
		}
		if rec.Banner != nil && rec.Banner.Ref.Link != "" {
			bannerCID = &rec.Banner.Ref.Link
			bannerMime = &rec.Banner.MimeType
		}
		if _, err := b.pool.Exec(ctx, q,
			ev.DID, rec.DisplayName, rec.Description,
			avatarCID, avatarMime, bannerCID, bannerMime, ev.CID); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		if _, err := b.pool.Exec(ctx,
			`DELETE FROM bluesky_profiles WHERE did = $1`, ev.DID); err != nil {
			return fmt.Errorf("delete %s: %w", ev.URI, err)
		}
		return nil
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

func (b *BlueskyProfile) isMember(ctx context.Context, did string) (bool, error) {
	var exists bool
	err := b.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1)`, did).
		Scan(&exists)
	return exists, err
}
```

- [ ] **Step 2:** Run the tests.

```bash
just test -run TestBlueskyProfile ./internal/index/...
```

Expected: the two tests PASS.

- [ ] **Step 3:** `just fmt`.

- [ ] **Step 4:** Commit.

```bash
git add appview/internal/index/bluesky_profile.go \
        appview/internal/index/bluesky_profile_test.go
git commit -m "index: add BlueskyProfile indexer with membership gate"
```

### Task 4.3: Idempotency, update, delete, and non-member-delete tests

**Files:**
- Modify: `appview/internal/index/bluesky_profile_test.go`

- [ ] **Step 1:** Append tests.

```go
func TestBlueskyProfile_UpdateReplacesFields(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	seedMember(t, pool, "did:plc:u")
	idx := index.NewBlueskyProfile(pool)
	ctx := context.Background()

	create := tap.Event{
		URI: "at://did:plc:u/app.bsky.actor.profile/self", CID: "c1",
		DID: "did:plc:u", Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "create",
		Record: json.RawMessage(`{"displayName":"old"}`),
	}
	update := create
	update.CID = "c2"
	update.Action = "update"
	update.Record = json.RawMessage(`{"displayName":"new"}`)

	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}
	if err := idx.Handle(ctx, update); err != nil {
		t.Fatal(err)
	}
	var dn, cid string
	_ = pool.QueryRow(ctx,
		`SELECT display_name, record_cid FROM bluesky_profiles WHERE did = $1`, create.DID).
		Scan(&dn, &cid)
	if dn != "new" || cid != "c2" {
		t.Errorf("after update: display_name=%q record_cid=%q; want new, c2", dn, cid)
	}
}

func TestBlueskyProfile_ReplayedEventPreservesIndexedAt(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	seedMember(t, pool, "did:plc:r")
	idx := index.NewBlueskyProfile(pool)
	ctx := context.Background()

	ev := tap.Event{
		URI: "at://did:plc:r/app.bsky.actor.profile/self", CID: "c1",
		DID: "did:plc:r", Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "create",
		Record: json.RawMessage(`{"displayName":"alice"}`),
	}
	if err := idx.Handle(ctx, ev); err != nil {
		t.Fatal(err)
	}

	var first string
	_ = pool.QueryRow(ctx,
		`SELECT indexed_at::text FROM bluesky_profiles WHERE did = $1`, ev.DID).Scan(&first)

	if err := idx.Handle(ctx, ev); err != nil {
		t.Fatal(err)
	}

	var second string
	_ = pool.QueryRow(ctx,
		`SELECT indexed_at::text FROM bluesky_profiles WHERE did = $1`, ev.DID).Scan(&second)

	if first != second {
		t.Errorf("indexed_at changed on replay: %q -> %q", first, second)
	}
}

func TestBlueskyProfile_DeleteRemovesRow(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	seedMember(t, pool, "did:plc:d")
	idx := index.NewBlueskyProfile(pool)
	ctx := context.Background()

	create := tap.Event{
		URI: "at://did:plc:d/app.bsky.actor.profile/self", CID: "c1",
		DID: "did:plc:d", Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "create",
		Record: json.RawMessage(`{"displayName":"x"}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}
	del := tap.Event{
		URI: create.URI, DID: create.DID, Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "delete",
	}
	if err := idx.Handle(ctx, del); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var count int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM bluesky_profiles WHERE did = $1`, del.DID).
		Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestBlueskyProfile_DeleteNonMemberIsNoop(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewBlueskyProfile(pool)
	del := tap.Event{
		URI: "at://did:plc:gone/app.bsky.actor.profile/self",
		DID: "did:plc:gone", Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "delete",
	}
	if err := idx.Handle(context.Background(), del); err != nil {
		t.Errorf("delete on non-member should be silent; got %v", err)
	}
}
```

- [ ] **Step 2:** Run.

```bash
just test -run TestBlueskyProfile ./internal/index/...
```

Expected: all PASS.

- [ ] **Step 3:** Commit.

```bash
git add appview/internal/index/bluesky_profile_test.go
git commit -m "test: add BlueskyProfile idempotency/update/delete coverage"
```

---

## Chunk 5: `HandleResolver` extension

Switch the interface to use `syntax.DID` / `syntax.Handle` and add `ResolveDID`.

### Task 5.1: Add a failing `ResolveDID` test

**Files:**
- Modify: `appview/internal/api/handle_resolver_test.go` (if it doesn't exist, create).

- [ ] **Step 1:** Inspect the current test file.

```bash
cat appview/internal/api/handle_resolver_test.go
```

Note whether it uses `identity.Directory` directly (meaning we update the stub too). If the file doesn't exist, create it in Step 2.

- [ ] **Step 2:** Write/extend the test.

If the file exists, add to it. If it doesn't, replace with:

```go
// appview/internal/api/handle_resolver_test.go
package api_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
)

// stubDirectory lets us inject canned directory results.
type stubDirectory struct {
	lookupDID    func(syntax.DID) (*identity.Identity, error)
	lookupHandle func(syntax.Handle) (*identity.Identity, error)
}

func (s stubDirectory) LookupDID(_ context.Context, did syntax.DID) (*identity.Identity, error) {
	return s.lookupDID(did)
}
func (s stubDirectory) LookupHandle(_ context.Context, h syntax.Handle) (*identity.Identity, error) {
	return s.lookupHandle(h)
}
func (s stubDirectory) Lookup(_ context.Context, _ syntax.AtIdentifier) (*identity.Identity, error) {
	panic("not used")
}
func (s stubDirectory) Purge(_ context.Context, _ syntax.AtIdentifier) error { return nil }

func TestDirectoryHandleResolver_ResolveDID_HappyPath(t *testing.T) {
	t.Parallel()
	r := api.DirectoryHandleResolver{Directory: stubDirectory{
		lookupHandle: func(h syntax.Handle) (*identity.Identity, error) {
			return &identity.Identity{DID: syntax.DID("did:plc:xyz"), Handle: h}, nil
		},
	}}
	got, err := r.ResolveDID(context.Background(), syntax.Handle("alice.example"))
	if err != nil {
		t.Fatal(err)
	}
	if got != syntax.DID("did:plc:xyz") {
		t.Errorf("got %q", got)
	}
}

func TestDirectoryHandleResolver_ResolveDID_LookupError(t *testing.T) {
	t.Parallel()
	r := api.DirectoryHandleResolver{Directory: stubDirectory{
		lookupHandle: func(h syntax.Handle) (*identity.Identity, error) {
			return nil, errors.New("plc down")
		},
	}}
	_, err := r.ResolveDID(context.Background(), syntax.Handle("alice.example"))
	if !errors.Is(err, api.ErrHandleUnavailable) {
		t.Errorf("want ErrHandleUnavailable; got %v", err)
	}
}
```

Note: `identity.Directory` is an interface; if the stub doesn't satisfy it after you write it, run `go build` and mirror the real signatures — the three methods above are the minimum the resolver uses; the rest can `panic` in tests.

- [ ] **Step 3:** Run — should fail to compile (`ResolveDID` doesn't exist yet).

```bash
just test -run TestDirectoryHandleResolver_ResolveDID ./internal/api/...
```

### Task 5.2: Add `ResolveDID` and switch to `syntax.DID`/`syntax.Handle`

**Files:**
- Modify: `appview/internal/api/handle_resolver.go`

- [ ] **Step 1:** Rewrite the file.

```go
// appview/internal/api/handle_resolver.go
package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// HandleResolver resolves between atproto DIDs and handles. Production
// impl wraps indigo's identity.Directory; tests commonly stub the
// interface directly.
type HandleResolver interface {
	ResolveHandle(ctx context.Context, did syntax.DID) (syntax.Handle, error)
	ResolveDID(ctx context.Context, handle syntax.Handle) (syntax.DID, error)
}

// DirectoryHandleResolver is the indigo-backed implementation.
type DirectoryHandleResolver struct {
	Directory identity.Directory
}

var _ HandleResolver = DirectoryHandleResolver{}

// ErrHandleUnavailable wraps every failure mode (directory error, empty
// handle, etc). Handlers convert this to 502 identity_unavailable.
var ErrHandleUnavailable = errors.New("handle unavailable")

// ResolveHandle returns the handle for did.
func (r DirectoryHandleResolver) ResolveHandle(ctx context.Context, did syntax.DID) (syntax.Handle, error) {
	id, err := r.Directory.LookupDID(ctx, did)
	if err != nil {
		return "", fmt.Errorf("%w: lookup: %v", ErrHandleUnavailable, err)
	}
	if id.Handle == "" || id.Handle == syntax.HandleInvalid {
		return "", fmt.Errorf("%w: empty handle for %s", ErrHandleUnavailable, did)
	}
	return id.Handle, nil
}

// ResolveDID returns the DID for handle.
func (r DirectoryHandleResolver) ResolveDID(ctx context.Context, handle syntax.Handle) (syntax.DID, error) {
	id, err := r.Directory.LookupHandle(ctx, handle)
	if err != nil {
		return "", fmt.Errorf("%w: lookup: %v", ErrHandleUnavailable, err)
	}
	return id.DID, nil
}
```

- [ ] **Step 2:** Run the resolver tests.

```bash
just test -run TestDirectoryHandleResolver ./internal/api/...
```

At this point `whoami.go` and `whoami_test.go` will fail to compile because `ResolveHandle` now takes `syntax.DID`. That's expected — we fix in Task 5.3.

### Task 5.3: Update `whoami.go` and its test to pass `syntax.DID`

**Files:**
- Modify: `appview/internal/api/whoami.go`
- Modify: `appview/internal/api/whoami_test.go`

- [ ] **Step 1:** Update `whoami.go`.

In the handler, after pulling `did` from context:

```go
// before
handle, err := resolver.ResolveHandle(r.Context(), did)

// after
parsed, perr := syntax.ParseDID(did)
if perr != nil {
    logger.Warn("whoami: invalid DID in context",
        slog.String("did", did),
        slog.String("err", perr.Error()),
        slog.String("run_id", middleware.GetRunID(r.Context())))
    envelope.WriteError(w, http.StatusInternalServerError,
        "internal_error", "invalid did in context",
        middleware.GetRunID(r.Context()), nil)
    return
}
handle, err := resolver.ResolveHandle(r.Context(), parsed)
```

And in the response:

```go
// before
_ = json.NewEncoder(w).Encode(WhoAmIResponse{DID: did, Handle: handle})

// after
_ = json.NewEncoder(w).Encode(WhoAmIResponse{DID: did, Handle: handle.String()})
```

Add the import: `"github.com/bluesky-social/indigo/atproto/syntax"`.

- [ ] **Step 2:** Update `whoami_test.go` stub signature.

```go
// before
func (s stubResolver) ResolveHandle(ctx context.Context, did string) (string, error) {
    return s.handle, s.err
}

// after
type stubResolver struct {
    handle syntax.Handle
    err    error
}

func (s stubResolver) ResolveHandle(_ context.Context, _ syntax.DID) (syntax.Handle, error) {
    return s.handle, s.err
}

func (s stubResolver) ResolveDID(_ context.Context, _ syntax.Handle) (syntax.DID, error) {
    return "", nil
}
```

And update the test initialisers: `stubResolver{handle: "alice.example"}` becomes `stubResolver{handle: syntax.Handle("alice.example")}`. Add the import.

- [ ] **Step 3:** Run.

```bash
just test ./internal/api/...
```

Expected: all PASS.

- [ ] **Step 4:** `just fmt`.

- [ ] **Step 5:** Commit.

```bash
git add appview/internal/api/handle_resolver.go \
        appview/internal/api/handle_resolver_test.go \
        appview/internal/api/whoami.go \
        appview/internal/api/whoami_test.go
git commit -m "api: add ResolveDID; type HandleResolver with syntax.DID/Handle"
```

---

## Chunk 6: `InitializeProfile` and callback integration

Runs after `ProcessCallback`. Three PDS calls: get Bluesky profile, get Craftsky profile, and (if the latter 404s) put an empty Craftsky profile.

### Task 6.1: Define a `PDSClient` interface so tests can mock

**Files:**
- Create: `appview/internal/auth/pds_client.go`

Indigo's `*atclient.APIClient` is a concrete type with many methods. We want a minimal interface the initialiser uses, to keep tests independent of indigo's client.

- [ ] **Step 1:** Write the interface.

```go
// appview/internal/auth/pds_client.go
package auth

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ErrRecordNotFound is the canonical "getRecord returned 404" sentinel
// used across this package. PDSClient implementations wrap whatever
// their upstream library raises into this value.
var ErrRecordNotFound = errors.New("pds: record not found")

// PDSClient is the minimal surface InitializeProfile uses against the
// user's PDS. In production it's an adapter over indigo's
// atclient.APIClient; in tests it's a hand-rolled mock.
//
// All record bodies are passed and returned as already-decoded Go values
// (typically map[string]any) — the adapter handles JSON encoding.
type PDSClient interface {
	GetRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, out any) error
	PutRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, record any) error
}
```

- [ ] **Step 2:** Compile check.

```bash
cd appview && go build ./... && cd -
```

- [ ] **Step 3:** Commit.

```bash
git add appview/internal/auth/pds_client.go
git commit -m "auth: add PDSClient interface for profile initialiser"
```

### Task 6.2: Write the failing `InitializeProfile` tests

**Files:**
- Create: `appview/internal/auth/initialize_profile_test.go`

- [ ] **Step 1:** Write tests with a mock `PDSClient`.

```go
// appview/internal/auth/initialize_profile_test.go
package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

type mockPDS struct {
	getCalls []getCall
	putCalls []putCall

	getRecord func(collection, rkey string, out any) error
	putRecord func(collection, rkey string, record any) error
}

type getCall struct{ Collection, Rkey string }
type putCall struct {
	Collection, Rkey string
	Record           any
}

func (m *mockPDS) GetRecord(_ context.Context, _ syntax.DID, collection, rkey string, out any) error {
	m.getCalls = append(m.getCalls, getCall{collection, rkey})
	return m.getRecord(collection, rkey, out)
}
func (m *mockPDS) PutRecord(_ context.Context, _ syntax.DID, collection, rkey string, record any) error {
	m.putCalls = append(m.putCalls, putCall{collection, rkey, record})
	return m.putRecord(collection, rkey, record)
}

const (
	bskyNSID = "app.bsky.actor.profile"
	cskyNSID = "social.craftsky.actor.profile"
)

func TestInitializeProfile_ReturningUserBothPresent(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) error {
			switch coll {
			case bskyNSID:
				*(out.(*map[string]any)) = map[string]any{"displayName": "Alice"}
				return nil
			case cskyNSID:
				*(out.(*map[string]any)) = map[string]any{
					"$type":  cskyNSID,
					"crafts": []any{"sewing"},
				}
				return nil
			}
			t.Fatalf("unexpected get collection %q", coll)
			return nil
		},
		putRecord: func(_, _ string, _ any) error {
			t.Fatalf("PutRecord should not be called for returning user")
			return nil
		},
	}
	if err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:a")); err != nil {
		t.Fatalf("InitializeProfile: %v", err)
	}
	if len(m.getCalls) != 2 {
		t.Errorf("getCalls = %d, want 2", len(m.getCalls))
	}
	if len(m.putCalls) != 0 {
		t.Errorf("putCalls = %d, want 0", len(m.putCalls))
	}
}

func TestInitializeProfile_NewUserWritesEmptyCraftsky(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) error {
			switch coll {
			case bskyNSID:
				*(out.(*map[string]any)) = map[string]any{"displayName": "Alice"}
				return nil
			case cskyNSID:
				return auth.ErrRecordNotFound
			}
			return nil
		},
		putRecord: func(coll, rkey string, record any) error {
			if coll != cskyNSID {
				t.Errorf("put collection = %q, want %q", coll, cskyNSID)
			}
			if rkey != "self" {
				t.Errorf("put rkey = %q, want self", rkey)
			}
			body, _ := record.(map[string]any)
			if body["$type"] != cskyNSID {
				t.Errorf("put $type = %v", body["$type"])
			}
			c, _ := body["crafts"].([]string)
			if len(c) != 0 {
				t.Errorf("put crafts = %v, want empty", c)
			}
			return nil
		},
	}
	if err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:b")); err != nil {
		t.Fatalf("InitializeProfile: %v", err)
	}
	if len(m.putCalls) != 1 {
		t.Errorf("putCalls = %d, want 1", len(m.putCalls))
	}
}

func TestInitializeProfile_NoBlueskyProfileIsOK(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, _ any) error {
			return auth.ErrRecordNotFound
		},
		putRecord: func(coll, _ string, _ any) error {
			if coll != cskyNSID {
				t.Errorf("put collection = %q, want %q", coll, cskyNSID)
			}
			return nil
		},
	}
	if err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:c")); err != nil {
		t.Fatalf("InitializeProfile: %v", err)
	}
}

func TestInitializeProfile_BlueskyReadErrorFails(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	m := &mockPDS{
		getRecord: func(coll, _ string, _ any) error {
			if coll == bskyNSID {
				return boom
			}
			return nil
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:d"))
	if err == nil {
		t.Fatal("want error; got nil")
	}
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Errorf("want ErrProfileInitFailed; got %v", err)
	}
}

func TestInitializeProfile_CraftskyReadErrorFails(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) error {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{}
				return nil
			}
			return errors.New("boom")
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:e"))
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Errorf("want ErrProfileInitFailed; got %v", err)
	}
}

func TestInitializeProfile_MalformedCraftskyRecord(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) error {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{}
				return nil
			}
			// crafts expected to be []string; return wrong type.
			*(out.(*map[string]any)) = map[string]any{
				"$type":  cskyNSID,
				"crafts": "not an array",
			}
			return nil
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:f"))
	if !errors.Is(err, auth.ErrProfileDataInvalid) {
		t.Errorf("want ErrProfileDataInvalid; got %v", err)
	}
}

func TestInitializeProfile_PutRecordFailure(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) error {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{}
				return nil
			}
			return auth.ErrRecordNotFound
		},
		putRecord: func(_, _ string, _ any) error { return errors.New("pds down") },
	}
	err := auth.InitializeProfile(context.Background(), m, syntax.DID("did:plc:g"))
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Errorf("want ErrProfileInitFailed; got %v", err)
	}
}
```

- [ ] **Step 2:** Run — build should fail (`InitializeProfile`, sentinels undefined).

```bash
just test -run TestInitializeProfile ./internal/auth/...
```

### Task 6.3: Implement `InitializeProfile`

**Files:**
- Create: `appview/internal/auth/initialize_profile.go`

- [ ] **Step 1:** Write the implementation.

```go
// appview/internal/auth/initialize_profile.go
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ErrProfileInitFailed wraps any non-404 PDS failure during onboarding-
// on-login. Callers surface this as a profile_init_failed error page.
var ErrProfileInitFailed = errors.New("profile: init failed")

// ErrProfileDataInvalid indicates the fetched social.craftsky.actor.profile
// record fails lexicon validation. Callers surface this as a
// profile_data_invalid error page.
var ErrProfileDataInvalid = errors.New("profile: data invalid")

const (
	blueskyProfileNSID    = "app.bsky.actor.profile"
	craftskyProfileNSID   = "social.craftsky.actor.profile"
	profileRecordKey      = "self"
)

// InitializeProfile performs onboarding-on-login side effects against
// the user's PDS:
//
//  1. Fetch app.bsky.actor.profile (non-404 errors fail).
//  2. Fetch social.craftsky.actor.profile.
//     - If present, validate it.
//     - If missing, write an empty {crafts: []} record.
//
// Called by the OAuth callback after ProcessCallback + SaveSession and
// before the Craftsky session token is returned. Per
// docs/superpowers/specs/2026-04-23-profile-onboarding-design.md §4, on
// any failure we fail the whole callback — the user is sent to an error
// page, their Craftsky session is not created.
func InitializeProfile(ctx context.Context, client PDSClient, did syntax.DID) error {
	// 1. Bluesky profile: presence is optional; only non-404 errors fail.
	var bskyRecord map[string]any
	if err := client.GetRecord(ctx, did, blueskyProfileNSID, profileRecordKey, &bskyRecord); err != nil {
		if !errors.Is(err, ErrRecordNotFound) {
			return fmt.Errorf("%w: get %s: %v", ErrProfileInitFailed, blueskyProfileNSID, err)
		}
	}

	// 2. Craftsky profile: present → validate; missing → write empty.
	var cskyRecord map[string]any
	err := client.GetRecord(ctx, did, craftskyProfileNSID, profileRecordKey, &cskyRecord)
	switch {
	case err == nil:
		if vErr := validateCraftskyProfile(cskyRecord); vErr != nil {
			return fmt.Errorf("%w: %v", ErrProfileDataInvalid, vErr)
		}
		return nil
	case errors.Is(err, ErrRecordNotFound):
		empty := map[string]any{
			"$type":  craftskyProfileNSID,
			"crafts": []string{},
		}
		if putErr := client.PutRecord(ctx, did, craftskyProfileNSID, profileRecordKey, empty); putErr != nil {
			return fmt.Errorf("%w: put %s: %v", ErrProfileInitFailed, craftskyProfileNSID, putErr)
		}
		return nil
	default:
		return fmt.Errorf("%w: get %s: %v", ErrProfileInitFailed, craftskyProfileNSID, err)
	}
}

// validateCraftskyProfile does a minimal shape check against
// social.craftsky.actor.profile. Stricter lexicon validation is future
// work; for now we just confirm crafts, if present, is an array of strings.
func validateCraftskyProfile(rec map[string]any) error {
	raw, ok := rec["crafts"]
	if !ok {
		return nil // crafts is optional per the lexicon.
	}
	arr, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("crafts is not an array (got %T)", raw)
	}
	for i, item := range arr {
		if _, ok := item.(string); !ok {
			return fmt.Errorf("crafts[%d] is not a string (got %T)", i, item)
		}
	}
	return nil
}
```

- [ ] **Step 2:** Run the tests.

```bash
just test -run TestInitializeProfile ./internal/auth/...
```

Expected: all PASS.

- [ ] **Step 3:** `just fmt`.

- [ ] **Step 4:** Commit.

```bash
git add appview/internal/auth/initialize_profile.go \
        appview/internal/auth/initialize_profile_test.go
git commit -m "auth: add InitializeProfile for onboarding-on-login"
```

### Task 6.4: Indigo-backed `PDSClient` adapter

**Files:**
- Create: `appview/internal/auth/pds_client_indigo.go`

Adapts indigo's `*atclient.APIClient` to our interface. XRPC naming: `com.atproto.repo.getRecord` is a query (GET); `com.atproto.repo.putRecord` is a procedure (POST).

- [ ] **Step 1:** Write the adapter.

```go
// appview/internal/auth/pds_client_indigo.go
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// IndigoPDSClient adapts indigo's *atclient.APIClient to our PDSClient
// interface.
type IndigoPDSClient struct {
	Client *atclient.APIClient
}

var _ PDSClient = (*IndigoPDSClient)(nil)

// GetRecord calls com.atproto.repo.getRecord on the user's PDS. A 404
// error response is translated to ErrRecordNotFound so callers can
// switch on presence.
func (i *IndigoPDSClient) GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) error {
	nsid, err := syntax.ParseNSID("com.atproto.repo.getRecord")
	if err != nil {
		return fmt.Errorf("parse nsid: %w", err)
	}
	var resp struct {
		URI   string `json:"uri"`
		CID   string `json:"cid"`
		Value any    `json:"value"`
	}
	params := map[string]any{
		"repo":       repo.String(),
		"collection": collection,
		"rkey":       rkey,
	}
	if err := i.Client.Get(ctx, nsid, params, &resp); err != nil {
		// indigo wraps HTTP 404 into an *atclient.APIError with StatusCode.
		var apiErr *atclient.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return ErrRecordNotFound
		}
		return err
	}
	// We want the record body in `out`, which is typically *map[string]any.
	// Re-assign via reflection-free assertion.
	if m, ok := out.(*map[string]any); ok {
		if v, ok := resp.Value.(map[string]any); ok {
			*m = v
			return nil
		}
		return fmt.Errorf("getRecord value has unexpected type %T", resp.Value)
	}
	return fmt.Errorf("unsupported out type %T", out)
}

// PutRecord calls com.atproto.repo.putRecord on the user's PDS.
func (i *IndigoPDSClient) PutRecord(ctx context.Context, repo syntax.DID, collection, rkey string, record any) error {
	nsid, err := syntax.ParseNSID("com.atproto.repo.putRecord")
	if err != nil {
		return fmt.Errorf("parse nsid: %w", err)
	}
	body := map[string]any{
		"repo":       repo.String(),
		"collection": collection,
		"rkey":       rkey,
		"record":     record,
	}
	var resp any
	return i.Client.Post(ctx, nsid, body, &resp)
}
```

**Cross-check:** the field on `atclient.APIError` is typically `StatusCode`. Verify by running `go doc github.com/bluesky-social/indigo/atproto/atclient.APIError`. If the field has a different name, adjust accordingly — the point is to translate an HTTP 404 into `ErrRecordNotFound`.

- [ ] **Step 2:** Compile check.

```bash
cd appview && go build ./... && cd -
```

If compilation fails because of `atclient.APIError` field naming, adjust — the adapter's public surface doesn't change.

- [ ] **Step 3:** Commit.

```bash
git add appview/internal/auth/pds_client_indigo.go
git commit -m "auth: add indigo-backed PDSClient adapter"
```

### Task 6.5: Wire `InitializeProfile` into the OAuth callback

**Files:**
- Modify: `appview/internal/auth/handlers_oauth.go`
- Modify: `appview/internal/auth/handlers_test.go`

- [ ] **Step 1:** Update `HTTPHandlers` to hold a `PDSClient` factory.

The callback handler needs a way to build a `PDSClient` from the DID and OAuth session ID it just processed. Add this as a field on `HTTPHandlers`:

```go
// In handlers_oauth.go, extend HTTPHandlers:
type HTTPHandlers struct {
    OAuth            *oauth.ClientApp
    CraftskySessions *CraftskySessionStore
    Pool             *pgxpool.Pool
    Logger           *slog.Logger
    DevMode          bool
    // NewPDSClient builds a PDSClient scoped to the given OAuth session.
    // Injected so tests can supply a mock without standing up indigo.
    NewPDSClient func(ctx context.Context, did syntax.DID, oauthSessionID string) (PDSClient, error)
}
```

And update the constructor:

```go
func NewHTTPHandlers(
    oauthApp *oauth.ClientApp,
    craftskyStore *CraftskySessionStore,
    pool *pgxpool.Pool,
    logger *slog.Logger,
    devMode bool,
    newPDSClient func(ctx context.Context, did syntax.DID, oauthSessionID string) (PDSClient, error),
) *HTTPHandlers {
    return &HTTPHandlers{
        OAuth:            oauthApp,
        CraftskySessions: craftskyStore,
        Pool:             pool,
        Logger:           logger,
        DevMode:          devMode,
        NewPDSClient:     newPDSClient,
    }
}
```

Add the import: `"github.com/bluesky-social/indigo/atproto/syntax"`.

- [ ] **Step 2:** Insert the initialiser call inside `CallbackHandler`, after `h.OAuth.ProcessCallback` succeeds but before `h.CraftskySessions.Create`.

```go
// Right after ProcessCallback success:
pdsClient, err := h.NewPDSClient(r.Context(), sessData.AccountDID, sessData.SessionID)
if err != nil {
    h.Logger.Error("NewPDSClient failed",
        slog.String("did", sessData.AccountDID.String()),
        slog.String("err", err.Error()))
    renderErrorHTML(w, http.StatusBadGateway,
        "Sign-in succeeded but we couldn't initialise your profile. Please try again.")
    return
}
if err := InitializeProfile(r.Context(), pdsClient, sessData.AccountDID); err != nil {
    h.Logger.Warn("InitializeProfile failed",
        slog.String("did", sessData.AccountDID.String()),
        slog.String("err", err.Error()))
    switch {
    case errors.Is(err, ErrProfileDataInvalid):
        renderErrorHTML(w, http.StatusBadGateway,
            "Your Craftsky profile record is in an unexpected format. Contact support.")
    default:
        renderErrorHTML(w, http.StatusBadGateway,
            "Sign-in succeeded but we couldn't initialise your profile. Please try again.")
    }
    return
}
```

Make sure `errors` is imported.

- [ ] **Step 3:** Add a handler test for "initialiser returns error → error page."

In `handlers_test.go`, add a test like:

```go
func TestCallbackHandler_InitializeProfileFailureRendersErrorPage(t *testing.T) {
    // Build an HTTPHandlers with NewPDSClient returning a mock whose
    // GetRecord always errors. Drive ProcessCallback via the existing
    // test harness (copy from an existing happy-path test in this file).
    // Assert rr.Code == http.StatusBadGateway and the body mentions the
    // expected user-facing message. Do NOT assert on CraftskySessions
    // being untouched — simpler to assert on the response only.
}
```

Look at existing tests in `handlers_test.go` for the harness shape; the test should follow the same setup as any other callback test. The point is: plug in an `InitializeProfile`-failing mock and verify the response is the error page, not the happy-path callback HTML.

- [ ] **Step 4:** Run.

```bash
just test ./internal/auth/...
```

Expected: all PASS. If the existing `handlers_test.go` tests fail because they don't supply `NewPDSClient`, update their constructors to pass a no-op mock (getRecord returns ErrRecordNotFound, putRecord returns nil).

- [ ] **Step 5:** `just fmt`.

- [ ] **Step 6:** Commit.

```bash
git add appview/internal/auth/handlers_oauth.go \
        appview/internal/auth/handlers_test.go
git commit -m "auth: call InitializeProfile in OAuth callback"
```

---

## Chunk 7: Profile store

The HTTP handlers read `craftsky_profiles` ⋈ `bluesky_profiles` by DID, and the PUT handler needs helpers for merging and writing. Isolate this in its own package-internal file to keep handlers slim.

### Task 7.1: Write the failing store tests

**Files:**
- Create: `appview/internal/api/profile_store_test.go`

- [ ] **Step 1:** Write tests.

```go
// appview/internal/api/profile_store_test.go
package api_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
)

const profileStoreDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE bluesky_profiles (
    did          TEXT        NOT NULL PRIMARY KEY,
    display_name TEXT,
    description  TEXT,
    avatar_cid   TEXT,
    avatar_mime  TEXT,
    banner_cid   TEXT,
    banner_mime  TEXT,
    record_cid   TEXT        NOT NULL,
    indexed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// Duplicated from internal/index/testhelpers_test.go to avoid cross-package
// test deps. Small enough to paste; if a third copy appears, extract.
func withSchema(t *testing.T, ddl string) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("no database URL")
	}
	ctx := context.Background()
	bootstrap, _ := pgxpool.New(ctx, url)
	schema := fmt.Sprintf("test_%d", rand.Uint32())
	if _, err := bootstrap.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = bootstrap.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		bootstrap.Close()
	})
	cfg, _ := pgxpool.ParseConfig(url)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, _ := pgxpool.NewWithConfig(ctx, cfg)
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestProfileStore_ReadByDID_MemberWithBothRows(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, profileStoreDDL)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, $2, $3)`,
		"did:plc:a", []string{"sewing"}, "cid1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO bluesky_profiles (did, display_name, avatar_cid, avatar_mime, record_cid)
		VALUES ($1, $2, $3, $4, $5)`,
		"did:plc:a", "Alice", "bafav", "image/jpeg", "cid2")
	if err != nil {
		t.Fatal(err)
	}

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:a")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.DID != "did:plc:a" {
		t.Errorf("DID = %q", got.DID)
	}
	if got.DisplayName == nil || *got.DisplayName != "Alice" {
		t.Errorf("DisplayName = %v", got.DisplayName)
	}
	if got.AvatarCID == nil || *got.AvatarCID != "bafav" {
		t.Errorf("AvatarCID = %v", got.AvatarCID)
	}
	if len(got.Crafts) != 1 || got.Crafts[0] != "sewing" {
		t.Errorf("Crafts = %v", got.Crafts)
	}
	if got.CreatedAt.IsZero() {
		t.Errorf("CreatedAt is zero")
	}
}

func TestProfileStore_ReadByDID_NonMember(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, profileStoreDDL)
	store := api.NewProfileStore(pool)
	_, err := store.Read(context.Background(), "did:plc:nobody")
	if err == nil {
		t.Fatal("want error; got nil")
	}
	if err != api.ErrProfileNotFound {
		t.Errorf("want ErrProfileNotFound; got %v", err)
	}
}

func TestProfileStore_ReadByDID_MemberWithoutBlueskyRow(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, profileStoreDDL)
	ctx := context.Background()
	_, _ = pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, $2, $3)`,
		"did:plc:b", []string{}, "cid1")

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:b")
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != nil {
		t.Errorf("DisplayName should be nil; got %v", *got.DisplayName)
	}
	if len(got.Crafts) != 0 {
		t.Errorf("Crafts = %v, want empty", got.Crafts)
	}
}
```

- [ ] **Step 2:** Run — should fail to build (`api.NewProfileStore`, `api.ErrProfileNotFound`, etc.).

```bash
just test -run TestProfileStore ./internal/api/...
```

### Task 7.2: Implement the profile store

**Files:**
- Create: `appview/internal/api/profile_store.go`

- [ ] **Step 1:** Write the store.

```go
// appview/internal/api/profile_store.go
package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrProfileNotFound is returned by ProfileStore.Read when the DID has
// no Craftsky membership row. Handlers translate this into 404.
var ErrProfileNotFound = errors.New("profile: not found")

// ProfileRow is the joined read view of craftsky_profiles and
// bluesky_profiles for a single DID. Nullable bluesky fields are pointers
// so "present but empty string" and "absent entirely" are distinguishable.
type ProfileRow struct {
	DID         string
	Crafts      []string
	CreatedAt   time.Time
	DisplayName *string
	Description *string
	AvatarCID   *string
	AvatarMime  *string
	BannerCID   *string
	BannerMime  *string
}

// ProfileStore is the Postgres-backed read/write surface used by the
// /v1/profiles/* handlers. It owns no business logic; merges, validation,
// and URL synthesis live in the handler layer.
type ProfileStore struct {
	pool *pgxpool.Pool
}

func NewProfileStore(pool *pgxpool.Pool) *ProfileStore {
	return &ProfileStore{pool: pool}
}

// Read returns the joined profile for did. Returns ErrProfileNotFound if
// the DID is not a Craftsky member (absent from craftsky_profiles).
func (s *ProfileStore) Read(ctx context.Context, did string) (*ProfileRow, error) {
	const q = `
		SELECT
			cp.did, cp.crafts, cp.created_at,
			bp.display_name, bp.description,
			bp.avatar_cid, bp.avatar_mime,
			bp.banner_cid, bp.banner_mime
		FROM craftsky_profiles cp
		LEFT JOIN bluesky_profiles bp ON bp.did = cp.did
		WHERE cp.did = $1
	`
	row := s.pool.QueryRow(ctx, q, did)
	out := &ProfileRow{}
	err := row.Scan(
		&out.DID, &out.Crafts, &out.CreatedAt,
		&out.DisplayName, &out.Description,
		&out.AvatarCID, &out.AvatarMime,
		&out.BannerCID, &out.BannerMime,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("profile read %s: %w", did, err)
	}
	return out, nil
}
```

- [ ] **Step 2:** Run the tests.

```bash
just test -run TestProfileStore ./internal/api/...
```

Expected: all PASS.

- [ ] **Step 3:** `just fmt`.

- [ ] **Step 4:** Commit.

```bash
git add appview/internal/api/profile_store.go \
        appview/internal/api/profile_store_test.go
git commit -m "api: add ProfileStore.Read joining craftsky+bluesky profiles"
```

---

## Chunk 8: `GET /v1/profiles/@{handleOrDid}` and `GET /v1/profiles/me`

Both handlers share the response composition, so we build the response type first, then the two handlers.

### Task 8.1: Failing tests for response composition

**Files:**
- Create: `appview/internal/api/profile_response_test.go`

- [ ] **Step 1:** Write tests for `BuildProfileResponse` and avatar/banner URL synthesis.

```go
// appview/internal/api/profile_response_test.go
package api_test

import (
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

func strPtr(s string) *string { return &s }

func TestBuildProfileResponse_FullRow(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{
		DID:         "did:plc:xyz",
		Crafts:      []string{"knitting", "sewing"},
		CreatedAt:   time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC),
		DisplayName: strPtr("Alice"),
		Description: strPtr("textile person"),
		AvatarCID:   strPtr("bafav"),
		AvatarMime:  strPtr("image/jpeg"),
		BannerCID:   strPtr("bafbn"),
		BannerMime:  strPtr("image/png"),
	}
	out := api.BuildProfileResponse(row, "alice.example", true)
	if out.DID != "did:plc:xyz" || out.Handle != "alice.example" {
		t.Errorf("did/handle mismatch: %+v", out)
	}
	if out.DisplayName == nil || *out.DisplayName != "Alice" {
		t.Errorf("displayName = %v", out.DisplayName)
	}
	if out.Avatar == nil ||
		*out.Avatar != "https://cdn.bsky.app/img/avatar/plain/did:plc:xyz/bafav@jpeg" {
		t.Errorf("avatar = %v", out.Avatar)
	}
	if out.Banner == nil ||
		*out.Banner != "https://cdn.bsky.app/img/banner/plain/did:plc:xyz/bafbn@png" {
		t.Errorf("banner = %v", out.Banner)
	}
	if out.CreatedAt == nil {
		t.Errorf("createdAt should be present for GET")
	}
}

func TestBuildProfileResponse_UnknownMimeOmitsAvatar(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{
		DID:        "did:plc:xyz",
		Crafts:     []string{},
		CreatedAt:  time.Now(),
		AvatarCID:  strPtr("baf"),
		AvatarMime: strPtr("image/tiff"), // not in supported set.
	}
	out := api.BuildProfileResponse(row, "h", true)
	if out.Avatar != nil {
		t.Errorf("avatar should be omitted; got %v", *out.Avatar)
	}
}

func TestBuildProfileResponse_NoCreatedAtForPut(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{
		DID:       "did:plc:x",
		Crafts:    []string{},
		CreatedAt: time.Now(),
	}
	out := api.BuildProfileResponse(row, "h", false)
	if out.CreatedAt != nil {
		t.Errorf("createdAt should be omitted for PUT; got %v", *out.CreatedAt)
	}
}

func TestBuildProfileResponse_EmptyCraftsStaysArray(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{DID: "did:plc:x", Crafts: nil, CreatedAt: time.Now()}
	out := api.BuildProfileResponse(row, "h", true)
	if out.Crafts == nil {
		t.Fatal("crafts must never be nil (must serialise as [])")
	}
	if len(out.Crafts) != 0 {
		t.Errorf("crafts = %v", out.Crafts)
	}
}
```

- [ ] **Step 2:** Run — should fail to build.

### Task 8.2: Implement `profile_response.go`

**Files:**
- Create: `appview/internal/api/profile_response.go`

- [ ] **Step 1:** Write the response type + builder.

```go
// appview/internal/api/profile_response.go
package api

import (
	"time"
)

// ProfileResponse is the JSON shape returned by all three profile
// endpoints. Fields tagged `omitempty` are omitted from the wire when nil.
type ProfileResponse struct {
	DID         string     `json:"did"`
	Handle      string     `json:"handle"`
	DisplayName *string    `json:"displayName,omitempty"`
	Description *string    `json:"description,omitempty"`
	Avatar      *string    `json:"avatar,omitempty"`
	Banner      *string    `json:"banner,omitempty"`
	Crafts      []string   `json:"crafts"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
}

// mimeExt maps the MIME types we know Bluesky's CDN serves into the
// extension suffix it expects in the URL. Unknown MIME types cause the
// avatar/banner field to be omitted rather than produce a broken URL.
// See docs/superpowers/specs/2026-04-23-profile-onboarding-design.md §5.4.
var mimeExt = map[string]string{
	"image/jpeg": "jpeg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// BuildProfileResponse composes a ProfileResponse from a row and a
// freshly-resolved handle. When includeCreatedAt is false, CreatedAt is
// nil — used by the PUT response path, which must not emit this field
// (see §5.3 of the spec).
func BuildProfileResponse(row *ProfileRow, handle string, includeCreatedAt bool) ProfileResponse {
	crafts := row.Crafts
	if crafts == nil {
		crafts = []string{}
	}
	out := ProfileResponse{
		DID:         row.DID,
		Handle:      handle,
		DisplayName: row.DisplayName,
		Description: row.Description,
		Crafts:      crafts,
	}
	if avatar := synthBlobURL("avatar", row.DID, row.AvatarCID, row.AvatarMime); avatar != "" {
		out.Avatar = &avatar
	}
	if banner := synthBlobURL("banner", row.DID, row.BannerCID, row.BannerMime); banner != "" {
		out.Banner = &banner
	}
	if includeCreatedAt {
		t := row.CreatedAt
		out.CreatedAt = &t
	}
	return out
}

func synthBlobURL(kind, did string, cid, mime *string) string {
	if cid == nil || mime == nil {
		return ""
	}
	ext, ok := mimeExt[*mime]
	if !ok {
		return ""
	}
	return "https://cdn.bsky.app/img/" + kind + "/plain/" + did + "/" + *cid + "@" + ext
}
```

- [ ] **Step 2:** Run.

```bash
just test -run TestBuildProfileResponse ./internal/api/...
```

Expected: all PASS.

- [ ] **Step 3:** `just fmt`.

- [ ] **Step 4:** Commit.

```bash
git add appview/internal/api/profile_response.go \
        appview/internal/api/profile_response_test.go
git commit -m "api: add ProfileResponse builder with CDN URL synthesis"
```

### Task 8.3: Failing tests for `GET /v1/profiles/@{handleOrDid}`

**Files:**
- Create: `appview/internal/api/profile_test.go`

- [ ] **Step 1:** Write initial GET tests. They need a fake store and resolver, and they use `httptest`.

```go
// appview/internal/api/profile_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// fakeStore implements the subset of ProfileStore that handlers call.
// Injected into handlers via the ProfileReader / ProfileWriter interfaces
// defined in profile.go.
type fakeStore struct {
	row *api.ProfileRow
	err error
}

func (f *fakeStore) Read(_ context.Context, _ string) (*api.ProfileRow, error) {
	return f.row, f.err
}

// fakeResolver implements api.HandleResolver.
type fakeResolver struct {
	handleFor syntax.Handle
	didFor    syntax.DID
	err       error
}

func (f fakeResolver) ResolveHandle(_ context.Context, _ syntax.DID) (syntax.Handle, error) {
	return f.handleFor, f.err
}
func (f fakeResolver) ResolveDID(_ context.Context, _ syntax.Handle) (syntax.DID, error) {
	return f.didFor, f.err
}

func nilLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGetProfile_ByDIDHappyPath(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{
		DID: "did:plc:xyz", Crafts: []string{"sewing"},
		CreatedAt: time.Now(),
	}
	h := api.GetProfileHandler(
		&fakeStore{row: row},
		fakeResolver{handleFor: "alice.example"},
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@did:plc:xyz", nil)
	req.SetPathValue("handleOrDid", "did:plc:xyz")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body api.ProfileResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body.DID != "did:plc:xyz" || body.Handle != "alice.example" {
		t.Errorf("%+v", body)
	}
}

func TestGetProfile_ByHandleHappyPath(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{DID: "did:plc:xyz", Crafts: []string{}, CreatedAt: time.Now()}
	resolver := fakeResolver{
		didFor:    syntax.DID("did:plc:xyz"),
		handleFor: syntax.Handle("alice.example"),
	}
	h := api.GetProfileHandler(&fakeStore{row: row}, resolver, nilLogger())
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@alice.example", nil)
	req.SetPathValue("handleOrDid", "alice.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestGetProfile_InvalidIdentifier(t *testing.T) {
	t.Parallel()
	h := api.GetProfileHandler(&fakeStore{}, fakeResolver{}, nilLogger())
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@NOT VALID", nil)
	req.SetPathValue("handleOrDid", "NOT VALID")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "invalid_identifier" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestGetProfile_NonMember(t *testing.T) {
	t.Parallel()
	h := api.GetProfileHandler(
		&fakeStore{err: api.ErrProfileNotFound},
		fakeResolver{},
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@did:plc:gone", nil)
	req.SetPathValue("handleOrDid", "did:plc:gone")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "profile_not_found" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestGetProfile_ResolveDIDError(t *testing.T) {
	t.Parallel()
	h := api.GetProfileHandler(
		&fakeStore{},
		fakeResolver{err: errors.New("plc down")},
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/@alice.example", nil)
	req.SetPathValue("handleOrDid", "alice.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "identity_unavailable" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestGetMeProfile_HappyPath(t *testing.T) {
	t.Parallel()
	row := &api.ProfileRow{DID: "did:plc:me", Crafts: []string{}, CreatedAt: time.Now()}
	h := api.GetMeProfileHandler(
		&fakeStore{row: row},
		fakeResolver{handleFor: "alice.example"},
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/me", nil)
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestGetMeProfile_NoDIDInContext(t *testing.T) {
	t.Parallel()
	h := api.GetMeProfileHandler(&fakeStore{}, fakeResolver{}, nilLogger())
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/me", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rr.Code)
	}
}
```

- [ ] **Step 2:** Run — should fail to build.

### Task 8.4: Implement `GetProfileHandler` and `GetMeProfileHandler`

**Files:**
- Create: `appview/internal/api/profile.go`

- [ ] **Step 1:** Write the handlers.

```go
// appview/internal/api/profile.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// ProfileReader is the read surface the profile GET handlers use. The
// concrete production implementation is *ProfileStore. Tests inject a
// fake.
type ProfileReader interface {
	Read(ctx context.Context, did string) (*ProfileRow, error)
}

// GetProfileHandler serves GET /v1/profiles/@{handleOrDid}.
//
// The "{handleOrDid}" path segment arrives URL-decoded by net/http's
// routing; this handler does not strip the leading "@" (the mux pattern
// includes the "@" literally).
func GetProfileHandler(store ProfileReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.PathValue("handleOrDid")
		runID := middleware.GetRunID(r.Context())
		did, err := resolveToDID(r.Context(), raw, resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				logger.Warn("profile: ResolveDID failed",
					slog.String("input", raw),
					slog.String("err", err.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		writeProfileResponse(w, r, store, resolver, did, logger)
	})
}

// GetMeProfileHandler serves GET /v1/profiles/me.
func GetMeProfileHandler(store ProfileReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		didStr, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		did, err := syntax.ParseDID(didStr)
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "invalid did in context", runID, nil)
			return
		}
		writeProfileResponse(w, r, store, resolver, did, logger)
	})
}

// writeProfileResponse loads the row, resolves the current handle, and
// emits the JSON response. Used by both GET handlers.
func writeProfileResponse(
	w http.ResponseWriter,
	r *http.Request,
	store ProfileReader,
	resolver HandleResolver,
	did syntax.DID,
	logger *slog.Logger,
) {
	runID := middleware.GetRunID(r.Context())
	row, err := store.Read(r.Context(), did.String())
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"profile_not_found", "profile not found", runID, nil)
			return
		}
		logger.Error("profile: store read failed",
			slog.String("did", did.String()),
			slog.String("err", err.Error()),
			slog.String("run_id", runID))
		envelope.WriteError(w, http.StatusInternalServerError,
			"internal_error", "profile read failed", runID, nil)
		return
	}
	handle, err := resolver.ResolveHandle(r.Context(), did)
	if err != nil {
		logger.Warn("profile: ResolveHandle failed",
			slog.String("did", did.String()),
			slog.String("err", err.Error()),
			slog.String("run_id", runID))
		envelope.WriteError(w, http.StatusBadGateway,
			"identity_unavailable", "could not resolve handle", runID, nil)
		return
	}
	resp := BuildProfileResponse(row, handle.String(), true)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// errInvalidIdentifier is used to signal a path-parsing failure back up
// to the handler. Not surfaced beyond this file.
var errInvalidIdentifier = errors.New("invalid identifier")

// resolveToDID parses raw as either a DID (starts with "did:") or a
// handle, and returns the DID either directly or via handle resolution.
func resolveToDID(ctx context.Context, raw string, resolver HandleResolver) (syntax.DID, error) {
	if strings.HasPrefix(raw, "did:") {
		did, err := syntax.ParseDID(raw)
		if err != nil {
			return "", errInvalidIdentifier
		}
		return did, nil
	}
	handle, err := syntax.ParseHandle(raw)
	if err != nil {
		return "", errInvalidIdentifier
	}
	return resolver.ResolveDID(ctx, handle)
}
```

- [ ] **Step 2:** Run the tests.

```bash
just test -run "TestGetProfile|TestGetMeProfile" ./internal/api/...
```

Expected: all PASS.

- [ ] **Step 3:** `just fmt`.

- [ ] **Step 4:** Commit.

```bash
git add appview/internal/api/profile.go \
        appview/internal/api/profile_test.go
git commit -m "api: add GET /v1/profiles handlers"
```

---

## Chunk 9: `PUT /v1/profiles/me`

Parallel writes, read-before-write merge on the Bluesky side, four outcome branches.

### Task 9.1: Request type + validation

**Files:**
- Create: `appview/internal/api/profile_request.go`
- Create: `appview/internal/api/profile_request_test.go`

- [ ] **Step 1:** Write the failing test.

```go
// appview/internal/api/profile_request_test.go
package api_test

import (
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestDecodeProfilePut_HappyPath(t *testing.T) {
	t.Parallel()
	body := `{"displayName":"Alice","description":"textile","crafts":["sewing"]}`
	req, err := api.DecodeProfilePut(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if req.DisplayName == nil || *req.DisplayName != "Alice" {
		t.Errorf("displayName = %v", req.DisplayName)
	}
	if req.Crafts == nil || len(req.Crafts) != 1 || req.Crafts[0] != "sewing" {
		t.Errorf("crafts = %v", req.Crafts)
	}
}

func TestDecodeProfilePut_RejectsAvatar(t *testing.T) {
	t.Parallel()
	_, err := api.DecodeProfilePut(strings.NewReader(`{"avatar":"blob:..."}`))
	var fe *api.FieldError
	if err == nil || !asFieldErr(err, &fe) {
		t.Fatalf("want FieldError; got %v", err)
	}
	if _, ok := fe.Fields["avatar"]; !ok {
		t.Errorf("fields = %v", fe.Fields)
	}
}

func TestDecodeProfilePut_RejectsBanner(t *testing.T) {
	t.Parallel()
	_, err := api.DecodeProfilePut(strings.NewReader(`{"banner":"blob:..."}`))
	var fe *api.FieldError
	if err == nil || !asFieldErr(err, &fe) {
		t.Fatal("want FieldError")
	}
	if _, ok := fe.Fields["banner"]; !ok {
		t.Errorf("fields = %v", fe.Fields)
	}
}

func TestValidateProfilePut_OversizeDisplayName(t *testing.T) {
	t.Parallel()
	dn := strings.Repeat("x", 641) // 641 bytes > 640.
	req := api.ProfilePutRequest{DisplayName: &dn}
	err := api.ValidateProfilePut(req)
	var fe *api.FieldError
	if !asFieldErr(err, &fe) {
		t.Fatalf("want FieldError; got %v", err)
	}
	if _, ok := fe.Fields["displayName"]; !ok {
		t.Errorf("fields = %v", fe.Fields)
	}
}

func TestValidateProfilePut_TooManyCrafts(t *testing.T) {
	t.Parallel()
	crafts := make([]string, 11)
	for i := range crafts {
		crafts[i] = "a"
	}
	req := api.ProfilePutRequest{Crafts: crafts}
	err := api.ValidateProfilePut(req)
	var fe *api.FieldError
	if !asFieldErr(err, &fe) || fe.Fields["crafts"] == "" {
		t.Fatalf("want FieldError on crafts; got %v", err)
	}
}

// asFieldErr is a tiny helper mirroring errors.As for our concrete type.
func asFieldErr(err error, out **api.FieldError) bool {
	if err == nil {
		return false
	}
	if fe, ok := err.(*api.FieldError); ok {
		*out = fe
		return true
	}
	return false
}
```

- [ ] **Step 2:** Run — should fail to build.

### Task 9.2: Implement request decoder and validator

**Files:**
- Create: `appview/internal/api/profile_request.go`

- [ ] **Step 1:** Write it.

```go
// appview/internal/api/profile_request.go
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// ProfilePutRequest is the decoded request body for PUT /v1/profiles/me.
// Avatar and banner are deliberately absent — the handler rejects bodies
// that carry them. See spec §5.3.
type ProfilePutRequest struct {
	DisplayName *string   `json:"displayName,omitempty"`
	Description *string   `json:"description,omitempty"`
	Crafts      []string  `json:"crafts,omitempty"`
}

// FieldError is returned by DecodeProfilePut and ValidateProfilePut when
// the request body has per-field problems. Handlers translate it into
// either 400 unexpected_field or 422 validation_failed per spec §5.3.
type FieldError struct {
	Code   string
	Fields map[string]string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%s: %v", e.Code, e.Fields)
}

// DecodeProfilePut reads a JSON body into ProfilePutRequest, rejecting
// any unknown keys and any occurrence of "avatar" or "banner" (which
// are deliberately not writable in v1). Returns a *FieldError with
// Code = "unexpected_field" in the latter case.
func DecodeProfilePut(body io.Reader) (ProfilePutRequest, error) {
	// Sniff the raw body once so we can reject avatar/banner with a
	// specific error code before touching strict-unmarshal.
	var raw map[string]json.RawMessage
	dec := json.NewDecoder(body)
	if err := dec.Decode(&raw); err != nil {
		return ProfilePutRequest{}, &FieldError{
			Code:   "malformed_body",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	rejected := map[string]string{}
	for _, k := range []string{"avatar", "banner"} {
		if _, present := raw[k]; present {
			rejected[k] = "not writable in v1"
		}
	}
	if len(rejected) > 0 {
		return ProfilePutRequest{}, &FieldError{
			Code:   "unexpected_field",
			Fields: rejected,
		}
	}
	// Re-marshal through a strict decoder to catch other unknown keys.
	out := ProfilePutRequest{}
	// Round-trip raw back to JSON so DisallowUnknownFields has a full body.
	pretty, err := json.Marshal(raw)
	if err != nil {
		return ProfilePutRequest{}, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": err.Error()}}
	}
	strict := json.NewDecoder(bytesReader(pretty))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&out); err != nil {
		return ProfilePutRequest{}, &FieldError{
			Code:   "unexpected_field",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	return out, nil
}

// bytesReader is a tiny helper to turn a []byte into an io.Reader.
func bytesReader(b []byte) io.Reader { return &byteReader{b: b} }

type byteReader struct {
	b   []byte
	pos int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.pos:])
	r.pos += n
	return n, nil
}

// ValidateProfilePut enforces the length/count constraints from spec §5.3.
func ValidateProfilePut(req ProfilePutRequest) error {
	fields := map[string]string{}
	if req.DisplayName != nil {
		if len(*req.DisplayName) > 640 || utf8.RuneCountInString(*req.DisplayName) > 64 {
			fields["displayName"] = "exceeds 64 graphemes / 640 bytes"
		}
	}
	if req.Description != nil {
		if len(*req.Description) > 2560 || utf8.RuneCountInString(*req.Description) > 256 {
			fields["description"] = "exceeds 256 graphemes / 2560 bytes"
		}
	}
	if req.Crafts != nil {
		if len(req.Crafts) > 10 {
			fields["crafts"] = "exceeds maximum of 10 entries"
		}
		for i, c := range req.Crafts {
			if len(c) > 50 || utf8.RuneCountInString(c) > 50 {
				fields[fmt.Sprintf("crafts[%d]", i)] = "exceeds 50 graphemes / 50 bytes"
			}
		}
	}
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}
```

> **Note on graphemes:** strictly, "graphemes" are unicode extended-grapheme-clusters, which the stdlib can't count on its own. `utf8.RuneCountInString` counts runes, which is a slightly stricter upper bound (most graphemes are one rune; combining marks count as extra runes). For v1 we accept that imprecision — this matches how other atproto clients ship. If a pedantic validation becomes necessary, add a `golang.org/x/text/unicode/norm`-based count later.

- [ ] **Step 2:** Run.

```bash
just test -run "TestDecodeProfilePut|TestValidateProfilePut" ./internal/api/...
```

Expected: all PASS.

- [ ] **Step 3:** `just fmt`.

- [ ] **Step 4:** Commit.

```bash
git add appview/internal/api/profile_request.go \
        appview/internal/api/profile_request_test.go
git commit -m "api: add ProfilePutRequest decode + validate"
```

### Task 9.3: Failing tests for `PUT /v1/profiles/me`

**Files:**
- Modify: `appview/internal/api/profile_test.go`

- [ ] **Step 1:** Add a fake PDS client and handler tests.

```go
// fakePDSForPut is a lightweight mock scoped to PUT tests.
type fakePDSForPut struct {
	getBsky      func() (map[string]any, error)
	putBsky      func(body map[string]any) error
	putCraftsky  func(body map[string]any) error
	putBskyCalls []map[string]any
}

// We cast this into the minimal PDSClient surface the PUT handler uses,
// declared via an interface on the api package (ProfilePDSClient).

func (f *fakePDSForPut) GetRecord(_ context.Context, _ syntax.DID, collection, _ string, out any) error {
	if collection == "app.bsky.actor.profile" {
		rec, err := f.getBsky()
		if err != nil {
			return err
		}
		*(out.(*map[string]any)) = rec
		return nil
	}
	return errors.New("unexpected get collection: " + collection)
}
func (f *fakePDSForPut) PutRecord(_ context.Context, _ syntax.DID, collection, _ string, body any) error {
	m, _ := body.(map[string]any)
	switch collection {
	case "app.bsky.actor.profile":
		f.putBskyCalls = append(f.putBskyCalls, m)
		return f.putBsky(m)
	case "social.craftsky.actor.profile":
		return f.putCraftsky(m)
	}
	return errors.New("unexpected put collection: " + collection)
}

// newPutHandler wires a fake store, resolver, and PDS client.
func newPutHandler(
	t *testing.T,
	store *fakeStore,
	pds *fakePDSForPut,
	resolver fakeResolver,
) http.Handler {
	t.Helper()
	return api.PutMeProfileHandler(
		store,
		resolver,
		func(_ context.Context, _ syntax.DID, _ string) (api.ProfilePDSClient, error) {
			return pds, nil
		},
		nilLogger(),
	)
}

func TestPutProfile_HappyPathMergesBlueskyExtras(t *testing.T) {
	t.Parallel()
	captured := map[string]any{}
	pds := &fakePDSForPut{
		getBsky: func() (map[string]any, error) {
			return map[string]any{
				"displayName": "old",
				"avatar": map[string]any{
					"$type":    "blob",
					"ref":      map[string]any{"$link": "bafav"},
					"mimeType": "image/jpeg",
					"size":     1,
				},
			}, nil
		},
		putBsky: func(body map[string]any) error {
			for k, v := range body {
				captured[k] = v
			}
			return nil
		},
		putCraftsky: func(_ map[string]any) error { return nil },
	}
	row := &api.ProfileRow{DID: "did:plc:me", Crafts: []string{"sewing"}, CreatedAt: time.Now()}
	h := newPutHandler(t,
		&fakeStore{row: row},
		pds,
		fakeResolver{handleFor: "alice.example"},
	)
	body := `{"displayName":"new","crafts":["sewing","quilting"]}`
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(body))
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if captured["displayName"] != "new" {
		t.Errorf("bluesky displayName = %v", captured["displayName"])
	}
	if _, ok := captured["avatar"]; !ok {
		t.Error("avatar must be preserved from existing record")
	}
}

func TestPutProfile_RejectsAvatar(t *testing.T) {
	t.Parallel()
	pds := &fakePDSForPut{}
	h := newPutHandler(t, &fakeStore{}, pds, fakeResolver{})
	body := `{"avatar":{"ref":{"$link":"x"}}}`
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(body))
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "unexpected_field" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestPutProfile_PartialSuccessReturns502(t *testing.T) {
	t.Parallel()
	pds := &fakePDSForPut{
		getBsky:     func() (map[string]any, error) { return map[string]any{}, nil },
		putBsky:     func(_ map[string]any) error { return nil },
		putCraftsky: func(_ map[string]any) error { return errors.New("pds down") },
	}
	row := &api.ProfileRow{DID: "did:plc:me", Crafts: []string{}, CreatedAt: time.Now()}
	h := newPutHandler(t,
		&fakeStore{row: row},
		pds,
		fakeResolver{handleFor: "alice.example"},
	)
	body := `{"displayName":"x"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(body))
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "pds_write_partial" {
		t.Errorf("code = %q", env.Error)
	}
	if env.Fields["craftsky"] != "failed" || env.Fields["bsky"] != "ok" {
		t.Errorf("fields = %v", env.Fields)
	}
}

func TestPutProfile_BothFailsReturns502(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	pds := &fakePDSForPut{
		getBsky:     func() (map[string]any, error) { return map[string]any{}, nil },
		putBsky:     func(_ map[string]any) error { return boom },
		putCraftsky: func(_ map[string]any) error { return boom },
	}
	row := &api.ProfileRow{DID: "did:plc:me", Crafts: []string{}, CreatedAt: time.Now()}
	h := newPutHandler(t, &fakeStore{row: row}, pds, fakeResolver{handleFor: "alice.example"})
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"x"}`))
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "pds_write_failed" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestPutProfile_ReadBeforeWriteFailure(t *testing.T) {
	t.Parallel()
	pds := &fakePDSForPut{
		getBsky: func() (map[string]any, error) { return nil, errors.New("pds down") },
	}
	row := &api.ProfileRow{DID: "did:plc:me", Crafts: []string{}, CreatedAt: time.Now()}
	h := newPutHandler(t, &fakeStore{row: row}, pds, fakeResolver{handleFor: "alice.example"})
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"x"}`))
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:me"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "pds_read_failed" {
		t.Errorf("code = %q", env.Error)
	}
}
```

Add the imports at the top of the file: `"strings"` and `"sync"`.

- [ ] **Step 2:** Run — should fail to build.

### Task 9.4: Implement `PutMeProfileHandler`

**Files:**
- Modify: `appview/internal/api/profile.go`

- [ ] **Step 1:** Add the PUT handler and its helpers.

```go
// Add to profile.go.

import (
    // existing imports
    "sync"
)

// ProfilePDSClient is the subset of PDS operations the PUT handler uses.
// Defined here (rather than in internal/auth) so handlers depend on a
// local abstraction and tests don't drag in the auth package's mock.
type ProfilePDSClient interface {
	GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) error
	PutRecord(ctx context.Context, repo syntax.DID, collection, rkey string, record any) error
}

// PDSClientFactory produces a per-request PDS client scoped to the caller's
// OAuth session.
type PDSClientFactory func(ctx context.Context, did syntax.DID, oauthSessionID string) (ProfilePDSClient, error)

// PutMeProfileHandler serves PUT /v1/profiles/me.
func PutMeProfileHandler(
	store ProfileReader,
	resolver HandleResolver,
	newPDS PDSClientFactory,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())

		didStr, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		did, derr := syntax.ParseDID(didStr)
		if derr != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "invalid did in context", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())

		reqBody, err := DecodeProfilePut(r.Body)
		if err != nil {
			if fe, ok := err.(*FieldError); ok {
				status := http.StatusBadRequest
				envelope.WriteError(w, status, fe.Code, "request body rejected", runID, fe.Fields)
				return
			}
			envelope.WriteError(w, http.StatusBadRequest,
				"malformed_body", "could not parse body", runID, nil)
			return
		}
		if err := ValidateProfilePut(reqBody); err != nil {
			if fe, ok := err.(*FieldError); ok {
				envelope.WriteError(w, http.StatusUnprocessableEntity,
					fe.Code, "validation failed", runID, fe.Fields)
				return
			}
			envelope.WriteError(w, http.StatusUnprocessableEntity,
				"validation_failed", "validation failed", runID, nil)
			return
		}

		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("profile: newPDS failed", slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}

		// Read-before-write on Bluesky so we preserve avatar/banner.
		var bsky map[string]any
		if err := pds.GetRecord(r.Context(), did, blueskyProfileNSID, profileRecordKey, &bsky); err != nil {
			logger.Warn("profile: bluesky getRecord failed", slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_read_failed", "could not read current bluesky profile", runID, nil)
			return
		}
		mergedBsky := mergeBlueskyRecord(bsky, reqBody)
		cskyBody := map[string]any{
			"$type":  craftskyProfileNSID,
			"crafts": nonNilStrings(reqBody.Crafts),
		}

		type writeResult struct {
			err error
		}
		var wg sync.WaitGroup
		wg.Add(2)
		bskyRes := make(chan writeResult, 1)
		cskyRes := make(chan writeResult, 1)
		go func() {
			defer wg.Done()
			err := pds.PutRecord(r.Context(), did, blueskyProfileNSID, profileRecordKey, mergedBsky)
			bskyRes <- writeResult{err: err}
		}()
		go func() {
			defer wg.Done()
			err := pds.PutRecord(r.Context(), did, craftskyProfileNSID, profileRecordKey, cskyBody)
			cskyRes <- writeResult{err: err}
		}()
		wg.Wait()
		close(bskyRes)
		close(cskyRes)
		bskyErr := (<-bskyRes).err
		cskyErr := (<-cskyRes).err

		switch {
		case bskyErr == nil && cskyErr == nil:
			// Compose response from the bodies we wrote, without a DB round-trip.
			handle, herr := resolver.ResolveHandle(r.Context(), did)
			if herr != nil {
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			row := syntheticRow(did.String(), mergedBsky, reqBody.Crafts)
			resp := BuildProfileResponse(row, handle.String(), false)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		case bskyErr != nil && cskyErr != nil:
			logger.Error("profile: both PDS writes failed",
				slog.String("bsky_err", bskyErr.Error()),
				slog.String("csky_err", cskyErr.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_failed", "both profile writes failed", runID, nil)
		default:
			logger.Warn("profile: partial PDS write",
				slog.Any("bsky_err", bskyErr), slog.Any("csky_err", cskyErr))
			fields := map[string]string{
				"bsky":     okOrFailed(bskyErr),
				"craftsky": okOrFailed(cskyErr),
			}
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_partial", "partial profile write", runID, fields)
		}
	})
}

// constants repeated from internal/auth/initialize_profile.go to avoid an
// import cycle; if either set drifts, keep them in sync.
const (
	blueskyProfileNSID  = "app.bsky.actor.profile"
	craftskyProfileNSID = "social.craftsky.actor.profile"
	profileRecordKey    = "self"
)

// mergeBlueskyRecord returns a fresh record body formed from `existing`
// (preserving avatar/banner/etc.) with displayName and description
// overridden by the request. If the request field is nil, the existing
// value is cleared from the output, matching PUT-clears-missing semantics.
func mergeBlueskyRecord(existing map[string]any, req ProfilePutRequest) map[string]any {
	out := map[string]any{"$type": blueskyProfileNSID}
	// Carry over everything except the client-managed fields.
	for k, v := range existing {
		switch k {
		case "$type", "displayName", "description":
			// either we set $type explicitly or these come from req
			continue
		default:
			out[k] = v
		}
	}
	if req.DisplayName != nil {
		out["displayName"] = *req.DisplayName
	}
	if req.Description != nil {
		out["description"] = *req.Description
	}
	return out
}

// syntheticRow constructs a ProfileRow from the bodies we just wrote,
// used to render the PUT response without a DB round-trip.
func syntheticRow(did string, bsky map[string]any, crafts []string) *ProfileRow {
	row := &ProfileRow{DID: did, Crafts: nonNilStrings(crafts)}
	if dn, ok := bsky["displayName"].(string); ok {
		row.DisplayName = &dn
	}
	if desc, ok := bsky["description"].(string); ok {
		row.Description = &desc
	}
	if av, ok := bsky["avatar"].(map[string]any); ok {
		if cid := blobCID(av); cid != "" {
			row.AvatarCID = &cid
		}
		if mime, ok := av["mimeType"].(string); ok && mime != "" {
			row.AvatarMime = &mime
		}
	}
	if bn, ok := bsky["banner"].(map[string]any); ok {
		if cid := blobCID(bn); cid != "" {
			row.BannerCID = &cid
		}
		if mime, ok := bn["mimeType"].(string); ok && mime != "" {
			row.BannerMime = &mime
		}
	}
	return row
}

func blobCID(blob map[string]any) string {
	ref, ok := blob["ref"].(map[string]any)
	if !ok {
		return ""
	}
	link, _ := ref["$link"].(string)
	return link
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func okOrFailed(err error) string {
	if err == nil {
		return "ok"
	}
	return "failed"
}
```

- [ ] **Step 2:** Run.

```bash
just test -run TestPutProfile ./internal/api/...
```

Expected: all PASS.

- [ ] **Step 3:** `just fmt`.

- [ ] **Step 4:** Commit.

```bash
git add appview/internal/api/profile.go \
        appview/internal/api/profile_test.go
git commit -m "api: add PUT /v1/profiles/me handler"
```

---

## Chunk 10: Wiring + routes + docs

Bring everything together in the route registration and dependency wiring, and update AGENTS.md.

### Task 10.1: Update `deps.go` to build handlers and register indexers

**Files:**
- Modify: `appview/internal/app/deps.go`

- [ ] **Step 1:** Register both new indexers and expose a `ProfileStore` + `NewPDSClient` factory.

In `newDeps`, after `dispatcher := index.NewDispatcher(index.NotImplemented{})`:

```go
dispatcher.Register("social.craftsky.actor.profile", index.NewCraftskyProfile(pool))
dispatcher.Register("app.bsky.actor.profile", index.NewBlueskyProfile(pool))
```

Add a field on `Deps`:

```go
// ProfileStore serves the /v1/profiles endpoints.
ProfileStore *api.ProfileStore
// NewPDSClient produces a PDSClient bound to an OAuth session.
NewPDSClient func(ctx context.Context, did syntax.DID, oauthSessionID string) (api.ProfilePDSClient, error)
```

Wire them in `newDeps`:

```go
deps.ProfileStore = api.NewProfileStore(pool)
deps.NewPDSClient = func(ctx context.Context, did syntax.DID, sid string) (api.ProfilePDSClient, error) {
    sess, err := oauthApp.ResumeSession(ctx, did, sid)
    if err != nil {
        return nil, err
    }
    return &auth.IndigoPDSClient{Client: sess.APIClient()}, nil
}
```

Add the `syntax` import.

Also update `NewHTTPHandlers` invocation to pass the new factory. Add an `auth.PDSClient` adapter for the callback that wraps the same factory — but the callback uses `auth.PDSClient`, not `api.ProfilePDSClient`. Two options: (a) make them the same interface in a shared package, (b) write a small adapter.

For this plan we keep them separate (they live in different packages). Add a parallel factory in `newDeps`:

```go
newAuthPDSClient := func(ctx context.Context, did syntax.DID, sid string) (auth.PDSClient, error) {
    sess, err := oauthApp.ResumeSession(ctx, did, sid)
    if err != nil {
        return nil, err
    }
    return &auth.IndigoPDSClient{Client: sess.APIClient()}, nil
}
```

And pass it into `NewHTTPHandlers`.

- [ ] **Step 2:** Compile check.

```bash
cd appview && go build ./... && cd -
```

Fix any wiring errors. Run the full test suite.

```bash
just test
```

Existing tests pass; new tests from previous chunks still green.

- [ ] **Step 3:** Commit.

```bash
git add appview/internal/app/deps.go appview/internal/app/deps_test.go
git commit -m "app: wire profile indexers, store, and PDS client factories"
```

### Task 10.2: Register the three routes

**Files:**
- Modify: `appview/internal/routes/routes.go`
- Modify: `appview/internal/routes/routes_test.go`

- [ ] **Step 1:** Write a failing test first. In `routes_test.go`, add assertions that `GET /v1/profiles/@{handleOrDid}`, `GET /v1/profiles/me`, and `PUT /v1/profiles/me` all exist and reject unauthenticated requests with 401.

Copy the pattern from an existing route test in the file. Each test should:
- Build routes via `routes.AddRoutes` with a minimal `*app.Deps`.
- Send an unauth request to each new route.
- Assert 401 (because `Authenticated` middleware is first).

- [ ] **Step 2:** Register the routes. In `routes.go`, after the existing `POST /v1/auth/logout` line:

```go
mux.Handle("GET /v1/profiles/@{handleOrDid}",
    authN(deviceID(api.GetProfileHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger))))
mux.Handle("GET /v1/profiles/me",
    authN(deviceID(api.GetMeProfileHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger))))
mux.Handle("PUT /v1/profiles/me",
    authN(deviceID(api.PutMeProfileHandler(deps.ProfileStore, deps.HandleResolver, deps.NewPDSClient, deps.Logger))))
```

- [ ] **Step 3:** Run.

```bash
just test ./internal/routes/...
```

Expected: all PASS.

- [ ] **Step 4:** `just fmt` + commit.

```bash
git add appview/internal/routes/routes.go appview/internal/routes/routes_test.go
git commit -m "routes: register /v1/profiles endpoints"
```

### Task 10.3: Final end-to-end sanity check

- [ ] **Step 1:** Run the full suite.

```bash
just test
```

Expected: all PASS.

- [ ] **Step 2:** Restart dev compose to exercise migrations from clean.

```bash
just down
just dev-d
just migrate up
```

Expected: migrations apply cleanly. `just psql -c '\d'` shows `craftsky_profiles` and `bluesky_profiles`, no `bluesky_posts_sample`.

- [ ] **Step 3:** Manual smoke test of onboarding-on-login if possible. This requires a real OAuth login flow against a dev PDS; skip if the dev harness isn't set up. Document the skipped check in the completion message.

### Task 10.4: Update AGENTS.md

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1:** Add a line to the Architectural Rules or Coding Conventions section (pick whichever flows better when you read the current file) noting the indexer extension pattern. Something like:

> **New firehose indexers** register via `dispatcher.Register(nsid, idx)` in `appview/internal/app/deps.go`. One indexer per NSID; `Handle` must be idempotent on `(URI, CID)`.

- [ ] **Step 2:** Commit.

```bash
git add AGENTS.md
git commit -m "docs: note indexer registration pattern in AGENTS.md"
```

---

## Done

At this point:
- The sample indexer is gone.
- Two real firehose indexers exist, with the membership gate on `bluesky_profiles`.
- OAuth callback seeds both tables via the firehose after writing an empty `social.craftsky.actor.profile` for new users.
- Three profile endpoints are live and tested.
- `HandleResolver` is typed with `syntax.DID`/`syntax.Handle` and supports both directions.

End-to-end, a user can:
1. Call `POST /v1/auth/login`, complete OAuth on their PDS → their `craftsky_profiles` and (if present) `bluesky_profiles` rows appear via firehose.
2. `GET /v1/profiles/me` → sees their (possibly empty) profile.
3. `PUT /v1/profiles/me` with `crafts` and `displayName` → writes propagate to both atproto records on their PDS, the firehose catches up, subsequent GETs reflect the change.

The Flutter app can use this surface when it lands onboarding UX in a future spec.
