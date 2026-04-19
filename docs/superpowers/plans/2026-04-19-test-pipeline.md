# Test Pipeline Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a disposable end-to-end validation of the firehose → index → Postgres → HTTP read API loop, using a throwaway `social.craftsky.test.post` lexicon and `test_posts` table. The surviving artifact is a collection-based `Dispatcher` in `internal/index/`; everything else gets deleted when the real `social.craftsky.feed.post` indexer lands.

**Architecture:** A new `index.Dispatcher` routes `tap.Event`s by collection NSID to registered indexers. A new `internal/testpipeline/` package contains the disposable test-post indexer, SQL queries, and HTTP handler. `GET /test/feed` is registered only when `deps.Config.Env == app.EnvDev` so the route cannot leak to production. The existing `BlueskyPostsSample` indexer stays in place (also disposable) and gets registered against the dispatcher for `app.bsky.feed.post`.

**Tech Stack:** Go 1.22+, `pgx/v5` (hand-written SQL; no sqlc in this repo), `log/slog`, stdlib `net/http`, `golang-migrate/v4` migrations. Tests run via `just test` against the compose Postgres.

**Spec:** [docs/superpowers/specs/2026-04-19-test-pipeline-design.md](../specs/2026-04-19-test-pipeline-design.md)

**Prerequisites to verify before starting:**
- `just dev` comes up cleanly (stack is healthy).
- `just test` passes on the current branch.
- `appview/internal/index/indexer.go` still defines `Indexer` with `Handle(ctx, tap.Event) error`.
- `tap.Event` still has fields `Collection`, `Action`, `URI`, `CID`, `DID`, `Rkey`, `Record`.

---

## File Structure

### New files

- `lexicon/social/craftsky/test/post.json` — throwaway record schema.
- `appview/migrations/000004_test_posts.up.sql` — creates `test_posts` table.
- `appview/migrations/000004_test_posts.down.sql` — drops the table.
- `appview/internal/index/dispatcher.go` — collection-NSID → `Indexer` router (survivable).
- `appview/internal/index/dispatcher_test.go` — dispatcher unit tests.
- `appview/internal/testpipeline/doc.go` — package-level DELETE ME notice.
- `appview/internal/testpipeline/indexer.go` — implements `index.Indexer` for `social.craftsky.test.post`.
- `appview/internal/testpipeline/indexer_test.go` — indexer unit tests against compose Postgres.
- `appview/internal/testpipeline/handler.go` — `GET /test/feed` handler.
- `appview/internal/testpipeline/handler_test.go` — handler unit tests.
- `appview/internal/testpipeline/integration_test.go` — end-to-end test (event → dispatcher → Postgres → HTTP).

### Modified files

- `appview/internal/app/deps.go` — construct a `Dispatcher`, register existing `BlueskyPostsSample` and new `testpipeline.Indexer` against it, wire it into `WSConsumer`.
- `appview/internal/routes/routes.go` — register `GET /test/feed` behind a `deps.Config.Env == app.EnvDev` guard.

### Files intentionally untouched

- `appview/internal/index/bluesky_posts_sample.go` — stays for now. Gets re-wired through the dispatcher in the deps change, but its internals don't change. It still no-ops on non-matching collections (the dispatcher will also no-op, making this belt-and-braces but harmless).
- `appview/internal/index/indexer.go` — `Indexer` interface and `NotImplemented` stub are already correct.
- `appview/internal/tap/consumer.go` — consumer consumes `HandlerIndexer`; dispatcher satisfies it. No changes needed.

---

## Chunk 1: Lexicon + migration

Ships the disposable data shape, nothing executable.

### Task 1.1: Add the `social.craftsky.test.post` lexicon

**Files:**
- Create: `lexicon/social/craftsky/test/post.json`

- [ ] **Step 1: Write the lexicon file**

```json
{
  "lexicon": 1,
  "id": "social.craftsky.test.post",
  "defs": {
    "main": {
      "type": "record",
      "description": "DISPOSABLE. Throwaway test post used to validate the firehose → index → Postgres → read API pipeline. Delete this NSID and all associated code (appview/internal/testpipeline/, test_posts table, /test/feed route) once the real social.craftsky.feed.post indexer lands. See docs/superpowers/specs/2026-04-19-test-pipeline-design.md.",
      "key": "tid",
      "record": {
        "type": "object",
        "required": ["text", "createdAt"],
        "properties": {
          "text": {
            "type": "string",
            "maxLength": 3000,
            "maxGraphemes": 300,
            "description": "The post text."
          },
          "createdAt": {
            "type": "string",
            "format": "datetime",
            "description": "Client-declared creation timestamp."
          }
        }
      }
    }
  }
}
```

- [ ] **Step 2: Verify JSON is valid**

Run: `python3 -m json.tool lexicon/social/craftsky/test/post.json > /dev/null`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add lexicon/social/craftsky/test/post.json
git commit -m "feat(lexicon): add disposable social.craftsky.test.post"
```

### Task 1.2: Add migration 000004 for `test_posts`

**Files:**
- Create: `appview/migrations/000004_test_posts.up.sql`
- Create: `appview/migrations/000004_test_posts.down.sql`

- [ ] **Step 1: Write the up migration**

```sql
-- DELETE ME: part of the disposable test pipeline. Drop this table
-- (with a follow-up drop migration) when the real social.craftsky.feed.post
-- indexer lands. See appview/internal/testpipeline/ and
-- docs/superpowers/specs/2026-04-19-test-pipeline-design.md.
CREATE TABLE test_posts (
    uri         TEXT PRIMARY KEY,
    cid         TEXT NOT NULL,
    did         TEXT NOT NULL,
    text        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX test_posts_created_at_idx ON test_posts (created_at DESC);
```

- [ ] **Step 2: Write the down migration**

```sql
DROP INDEX IF EXISTS test_posts_created_at_idx;
DROP TABLE IF EXISTS test_posts;
```

- [ ] **Step 3: Run the migration against the dev Postgres**

If the stack isn't already up:
```bash
just dev-d
```

Then apply migrations:
```bash
just migrate up
```

(The `migrate` recipe runs `docker compose exec appview /app/cli migrate up` under the hood.) Expected: migration 000004 reported as applied.

- [ ] **Step 4: Confirm the table exists**

Run: `just psql -c "\d test_posts"`
Expected: table definition prints with columns `uri`, `cid`, `did`, `text`, `created_at`, `indexed_at` and the `test_posts_created_at_idx` index.

- [ ] **Step 5: Verify the down migration reverses cleanly**

```bash
just migrate down 1
just psql -c "\d test_posts"
```
Expected second command: `Did not find any relation named "test_posts"`.

Then re-apply and confirm the table is back:
```bash
just migrate up
just psql -c "\d test_posts"
```

- [ ] **Step 6: Commit**

```bash
git add appview/migrations/000004_test_posts.up.sql appview/migrations/000004_test_posts.down.sql
git commit -m "feat(db): add disposable test_posts table (migration 000004)"
```

---

**CHUNK 1 REVIEW GATE:** Dispatch plan-document-reviewer on Chunk 1 before proceeding. Fix issues, re-dispatch, iterate until approved.

---

## Chunk 2: Dispatcher (the survivable piece)

This is the one piece of code from this slice that outlives the cleanup. It gets scrutinized harder than the rest.

### Task 2.1: Write the dispatcher test suite (TDD)

**Files:**
- Create: `appview/internal/index/dispatcher_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package index

import (
	"context"
	"errors"
	"testing"

	"social.craftsky/appview/internal/tap"
)

// fakeIndexer records every event it's asked to handle and returns a
// configurable error.
type fakeIndexer struct {
	name   string
	events []tap.Event
	err    error
}

func (f *fakeIndexer) Handle(_ context.Context, ev tap.Event) error {
	f.events = append(f.events, ev)
	return f.err
}

func TestDispatcher_RoutesByCollection(t *testing.T) {
	a := &fakeIndexer{name: "a"}
	b := &fakeIndexer{name: "b"}
	fallback := &fakeIndexer{name: "fallback"}

	d := NewDispatcher(fallback)
	d.Register("social.craftsky.test.post", a)
	d.Register("app.bsky.feed.post", b)

	if err := d.Handle(context.Background(), tap.Event{Collection: "social.craftsky.test.post", URI: "at://x"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := d.Handle(context.Background(), tap.Event{Collection: "app.bsky.feed.post", URI: "at://y"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(a.events); got != 1 {
		t.Errorf("indexer a: got %d events, want 1", got)
	}
	if got := len(b.events); got != 1 {
		t.Errorf("indexer b: got %d events, want 1", got)
	}
	if got := len(fallback.events); got != 0 {
		t.Errorf("fallback: got %d events, want 0", got)
	}
}

func TestDispatcher_UnregisteredGoesToFallback(t *testing.T) {
	fallback := &fakeIndexer{}
	d := NewDispatcher(fallback)

	if err := d.Handle(context.Background(), tap.Event{Collection: "com.example.unknown"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(fallback.events); got != 1 {
		t.Fatalf("fallback: got %d events, want 1", got)
	}
}

func TestDispatcher_PropagatesDownstreamError(t *testing.T) {
	boom := errors.New("boom")
	a := &fakeIndexer{err: boom}
	d := NewDispatcher(NotImplemented{})
	d.Register("x.y.z", a)

	err := d.Handle(context.Background(), tap.Event{Collection: "x.y.z"})
	if !errors.Is(err, boom) {
		t.Fatalf("got %v, want boom", err)
	}
}

func TestDispatcher_NilFallbackPanicsOnMiss(t *testing.T) {
	// A nil fallback is a wiring bug — prefer a loud panic at boot over a
	// silent drop in prod. We don't test the happy path needing a fallback;
	// this test documents the contract: every dispatcher must have one.
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil fallback")
		}
	}()
	NewDispatcher(nil)
}
```

- [ ] **Step 2: Run tests to verify they fail (dispatcher not written yet)**

Run: `just test ./appview/internal/index/...` (or the project equivalent — check `justfile`)
Expected: FAIL — `NewDispatcher` undefined.

### Task 2.2: Implement the dispatcher

**Files:**
- Create: `appview/internal/index/dispatcher.go`

- [ ] **Step 1: Write the implementation**

```go
package index

import (
	"context"

	"social.craftsky/appview/internal/tap"
)

// Dispatcher routes tap.Events to indexers keyed by the event's atproto
// collection NSID. It itself implements Indexer so it can be handed to
// the Tap consumer in place of a single concrete indexer.
//
// Dispatcher is NOT safe for concurrent Register calls. Register is
// expected to be called once during startup wiring, before Run on the
// consumer. Handle is called serially by the Tap consumer (one event at
// a time per connection), so no locking is needed on the read path.
type Dispatcher struct {
	handlers map[string]Indexer
	fallback Indexer
}

// NewDispatcher returns a Dispatcher with the given fallback for events
// whose collection has no registered handler. fallback must be non-nil;
// a wiring mistake that passes nil is a loud panic (we prefer that to a
// silent drop in prod).
func NewDispatcher(fallback Indexer) *Dispatcher {
	if fallback == nil {
		panic("index.NewDispatcher: fallback must not be nil")
	}
	return &Dispatcher{
		handlers: map[string]Indexer{},
		fallback: fallback,
	}
}

// Register associates collection (e.g. "social.craftsky.feed.post") with
// idx. A later Register for the same collection replaces the previous
// handler; this is convenient in tests and startup-only in prod.
func (d *Dispatcher) Register(collection string, idx Indexer) {
	d.handlers[collection] = idx
}

// Handle routes ev to the indexer registered for ev.Collection, or to
// the fallback if none matches. Downstream errors propagate unchanged.
func (d *Dispatcher) Handle(ctx context.Context, ev tap.Event) error {
	if h, ok := d.handlers[ev.Collection]; ok {
		return h.Handle(ctx, ev)
	}
	return d.fallback.Handle(ctx, ev)
}

var _ Indexer = (*Dispatcher)(nil)
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `just test ./appview/internal/index/...`
Expected: all four dispatcher tests PASS. `bluesky_posts_sample_test.go` and `indexer_test.go` keep passing.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/index/dispatcher.go appview/internal/index/dispatcher_test.go
git commit -m "feat(index): add collection-NSID Dispatcher"
```

---

**CHUNK 2 REVIEW GATE:** Dispatch plan-document-reviewer on Chunk 2. Fix, re-dispatch, iterate.

---

## Chunk 3: Testpipeline indexer

### Task 3.1: Create the package with a DELETE ME doc.go

**Files:**
- Create: `appview/internal/testpipeline/doc.go`

- [ ] **Step 1: Write the package doc**

```go
// Package testpipeline is DISPOSABLE. It exists to validate the appview's
// firehose → index → Postgres → HTTP read API loop end to end using the
// throwaway lexicon social.craftsky.test.post and the test_posts table.
//
// When the real social.craftsky.feed.post indexer lands, DELETE the
// entire package (rm -rf appview/internal/testpipeline/) along with:
//
//   - lexicon/social/craftsky/test/post.json
//   - appview/migrations/000004_test_posts.up.sql + .down.sql
//     (plus a follow-up drop migration)
//   - the GET /test/feed route registration in internal/routes/routes.go
//   - the Dispatcher.Register call for social.craftsky.test.post in
//     internal/app/deps.go
//
// The Dispatcher itself stays.
//
// See docs/superpowers/specs/2026-04-19-test-pipeline-design.md.
package testpipeline
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./appview/internal/testpipeline/...`
Expected: no output, exit 0.

### Task 3.2: TDD the indexer — create/update

**Files:**
- Create: `appview/internal/testpipeline/indexer_test.go`

- [ ] **Step 1: Write the first failing tests**

The test file uses the external test package (`testpipeline_test`) so it matches the existing idiom in `appview/internal/index/bluesky_posts_sample_test.go`. It includes a file-local `withSchema(t)` helper — copied from `bluesky_posts_sample_test.go` but with DDL adapted to the `test_posts` schema — that creates a fresh Postgres schema per test and returns a `search_path`-scoped pool. Dropped in `t.Cleanup`. Every test uses `t.Parallel()` because each gets its own schema.

```go
package testpipeline_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testpipeline"
)

// withSchema creates an isolated schema for one test and returns a pool
// whose default search_path points at it. Dropped via t.Cleanup.
// Adapted from appview/internal/index/bluesky_posts_sample_test.go.
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

	ddl := `
		CREATE TABLE ` + schemaName + `.test_posts (
			uri        TEXT PRIMARY KEY,
			cid        TEXT NOT NULL,
			did        TEXT NOT NULL,
			text       TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX ON ` + schemaName + `.test_posts (created_at DESC);
	`
	if _, err := bootstrap.Exec(ctx, ddl); err != nil {
		t.Fatalf("create test table: %v", err)
	}

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

func TestIndexer_CreateInsertsRow(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)

	rec, _ := json.Marshal(map[string]any{
		"text":      "hello pipeline",
		"createdAt": "2026-04-19T10:00:00Z",
	})
	ev := tap.Event{
		URI:        "at://did:plc:abc/social.craftsky.test.post/3kxaaa",
		CID:        "bafyaaa",
		DID:        "did:plc:abc",
		Collection: "social.craftsky.test.post",
		Rkey:       "3kxaaa",
		Action:     "create",
		Record:     rec,
	}

	if err := ix.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var (
		gotCID, gotDID, gotText string
		gotCreatedAt            time.Time
	)
	err := pool.QueryRow(context.Background(),
		`SELECT cid, did, text, created_at FROM test_posts WHERE uri = $1`,
		ev.URI,
	).Scan(&gotCID, &gotDID, &gotText, &gotCreatedAt)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if gotCID != "bafyaaa" || gotDID != "did:plc:abc" || gotText != "hello pipeline" {
		t.Errorf("row mismatch: cid=%s did=%s text=%s", gotCID, gotDID, gotText)
	}
	wantT, _ := time.Parse(time.RFC3339, "2026-04-19T10:00:00Z")
	if !gotCreatedAt.Equal(wantT) {
		t.Errorf("created_at: got %v want %v", gotCreatedAt, wantT)
	}
}

func TestIndexer_UpdateReplacesRow(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)

	rec1, _ := json.Marshal(map[string]any{"text": "v1", "createdAt": "2026-04-19T10:00:00Z"})
	rec2, _ := json.Marshal(map[string]any{"text": "v2", "createdAt": "2026-04-19T10:00:00Z"})
	evBase := tap.Event{
		URI: "at://did:plc:abc/social.craftsky.test.post/3kxaaa",
		DID: "did:plc:abc", Collection: "social.craftsky.test.post", Rkey: "3kxaaa",
	}
	create := evBase
	create.CID = "bafy1"
	create.Action = "create"
	create.Record = rec1
	update := evBase
	update.CID = "bafy2"
	update.Action = "update"
	update.Record = rec2

	if err := ix.Handle(context.Background(), create); err != nil {
		t.Fatal(err)
	}
	if err := ix.Handle(context.Background(), update); err != nil {
		t.Fatal(err)
	}

	var count int
	pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM test_posts WHERE uri = $1`, evBase.URI,
	).Scan(&count)
	if count != 1 {
		t.Errorf("row count: got %d want 1", count)
	}
	var cid, text string
	pool.QueryRow(context.Background(),
		`SELECT cid, text FROM test_posts WHERE uri = $1`, evBase.URI,
	).Scan(&cid, &text)
	if cid != "bafy2" || text != "v2" {
		t.Errorf("post-update: cid=%s text=%s, want bafy2/v2", cid, text)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `just test ./appview/internal/testpipeline/...`
Expected: FAIL — `NewIndexer` undefined.

### Task 3.3: Implement the indexer create/update path

**Files:**
- Create: `appview/internal/testpipeline/indexer.go`

- [ ] **Step 1: Write the minimal implementation**

```go
package testpipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

// Indexer writes social.craftsky.test.post records into test_posts.
// Delete me with the rest of the package; see doc.go.
type Indexer struct {
	pool *pgxpool.Pool
}

// NewIndexer returns an indexer backed by pool.
func NewIndexer(pool *pgxpool.Pool) *Indexer { return &Indexer{pool: pool} }

const testPostNSID = "social.craftsky.test.post"

// testPostRecord is the decoded shape of a social.craftsky.test.post.
// Fields not defined in the lexicon are ignored.
type testPostRecord struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

// Handle upserts on create/update and deletes on delete. Events for any
// other collection are ignored (the Dispatcher already routes by NSID,
// so this is belt-and-braces — same idiom as BlueskyPostsSample).
//
// createdAt is required because test_posts.created_at is NOT NULL. Empty
// text is allowed through (the lexicon treats it as valid; the table
// column is NOT NULL but empty string satisfies it).
func (i *Indexer) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != testPostNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		var rec testPostRecord
		if err := json.Unmarshal(ev.Record, &rec); err != nil {
			return fmt.Errorf("unmarshal test post %s: %w", ev.URI, err)
		}
		if rec.CreatedAt.IsZero() {
			return fmt.Errorf("test post %s: missing createdAt", ev.URI)
		}
		const q = `
			INSERT INTO test_posts (uri, cid, did, text, created_at, indexed_at)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (uri) DO UPDATE SET
				cid        = EXCLUDED.cid,
				text       = EXCLUDED.text,
				created_at = EXCLUDED.created_at,
				indexed_at = now()
		`
		if _, err := i.pool.Exec(ctx, q, ev.URI, ev.CID, ev.DID, rec.Text, rec.CreatedAt); err != nil {
			return fmt.Errorf("upsert %s: %w", ev.URI, err)
		}
		return nil
	case "delete":
		// Implemented in Task 3.4.
		return fmt.Errorf("delete not yet implemented")
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}
```

- [ ] **Step 2: Run tests to verify create+update pass**

Run: `just test ./appview/internal/testpipeline/...`
Expected: both TestIndexer_CreateInsertsRow and TestIndexer_UpdateReplacesRow PASS.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/testpipeline/doc.go appview/internal/testpipeline/indexer.go appview/internal/testpipeline/indexer_test.go
git commit -m "feat(testpipeline): indexer upserts on create/update"
```

### Task 3.4: TDD + implement delete

**Files:**
- Modify: `appview/internal/testpipeline/indexer_test.go`
- Modify: `appview/internal/testpipeline/indexer.go`

- [ ] **Step 1: Add the delete test**

Append to `indexer_test.go`:

```go
func TestIndexer_DeleteRemovesRow(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)

	rec, _ := json.Marshal(map[string]any{"text": "bye", "createdAt": "2026-04-19T10:00:00Z"})
	uri := "at://did:plc:abc/social.craftsky.test.post/3kxbbb"
	create := tap.Event{
		URI: uri, CID: "bafy1", DID: "did:plc:abc",
		Collection: "social.craftsky.test.post", Rkey: "3kxbbb",
		Action: "create", Record: rec,
	}
	del := tap.Event{
		URI: uri, DID: "did:plc:abc",
		Collection: "social.craftsky.test.post", Rkey: "3kxbbb",
		Action: "delete",
	}

	if err := ix.Handle(context.Background(), create); err != nil {
		t.Fatal(err)
	}
	if err := ix.Handle(context.Background(), del); err != nil {
		t.Fatalf("delete: %v", err)
	}

	var count int
	pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM test_posts WHERE uri = $1`, uri,
	).Scan(&count)
	if count != 0 {
		t.Errorf("row count after delete: got %d want 0", count)
	}
}

func TestIndexer_DuplicateCreateIsIdempotent(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)

	rec, _ := json.Marshal(map[string]any{"text": "once", "createdAt": "2026-04-19T10:00:00Z"})
	ev := tap.Event{
		URI: "at://did:plc:abc/social.craftsky.test.post/3kxccc",
		CID: "bafy1", DID: "did:plc:abc",
		Collection: "social.craftsky.test.post", Rkey: "3kxccc",
		Action: "create", Record: rec,
	}

	if err := ix.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if err := ix.Handle(context.Background(), ev); err != nil {
		t.Fatalf("second create: %v", err)
	}

	var count int
	pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM test_posts WHERE uri = $1`, ev.URI,
	).Scan(&count)
	if count != 1 {
		t.Errorf("duplicate create produced %d rows, want 1", count)
	}
}

func TestIndexer_MalformedRecordErrors(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)

	ev := tap.Event{
		URI: "at://did:plc:abc/social.craftsky.test.post/3kxddd",
		CID: "bafy1", DID: "did:plc:abc",
		Collection: "social.craftsky.test.post", Rkey: "3kxddd",
		Action: "create", Record: []byte(`{"not valid json`),
	}
	if err := ix.Handle(context.Background(), ev); err == nil {
		t.Fatal("expected error on malformed record, got nil")
	}
}
```

- [ ] **Step 2: Run tests; delete test should fail, malformed+duplicate tests should pass**

Run: `just test ./appview/internal/testpipeline/...`
Expected: TestIndexer_DeleteRemovesRow FAIL ("delete not yet implemented"); duplicate + malformed PASS (already covered by the existing implementation).

- [ ] **Step 3: Implement delete**

Replace the `case "delete":` block in `indexer.go`:

```go
	case "delete":
		if _, err := i.pool.Exec(ctx,
			`DELETE FROM test_posts WHERE uri = $1`, ev.URI); err != nil {
			return fmt.Errorf("delete %s: %w", ev.URI, err)
		}
		return nil
```

- [ ] **Step 4: Run tests — all five should pass**

Run: `just test ./appview/internal/testpipeline/...`
Expected: all five indexer tests PASS.

- [ ] **Step 5: Commit**

```bash
git add appview/internal/testpipeline/indexer.go appview/internal/testpipeline/indexer_test.go
git commit -m "feat(testpipeline): handle delete + idempotent create"
```

---

**CHUNK 3 REVIEW GATE:** Dispatch plan-document-reviewer on Chunk 3. Fix, re-dispatch, iterate.

---

## Chunk 4: HTTP handler

### Task 4.1: TDD the handler

**Files:**
- Create: `appview/internal/testpipeline/handler_test.go`

- [ ] **Step 1: Write the failing tests**

This file lives in the same external test package as `indexer_test.go` (`testpipeline_test`), so it reuses the `withSchema` helper defined there. Tests use `t.Parallel()` for the same schema-per-test reason.

```go
package testpipeline_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testpipeline"
)

func TestHandler_EmptyTableReturnsEmptyArray(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	h := testpipeline.NewHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/test/feed", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var body struct {
		Posts []any `json:"posts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Posts == nil {
		t.Error("posts field should be non-nil empty array, not null")
	}
	if len(body.Posts) != 0 {
		t.Errorf("posts: got %d items want 0", len(body.Posts))
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content-type: got %q", ct)
	}
}

func TestHandler_ReturnsReverseChronological(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)
	h := testpipeline.NewHandler(pool)

	for _, row := range []struct {
		uri, text, ts string
	}{
		{"at://did:plc:a/social.craftsky.test.post/1", "first",  "2026-04-19T10:00:00Z"},
		{"at://did:plc:a/social.craftsky.test.post/2", "second", "2026-04-19T11:00:00Z"},
		{"at://did:plc:a/social.craftsky.test.post/3", "third",  "2026-04-19T12:00:00Z"},
	} {
		rec, _ := json.Marshal(map[string]any{"text": row.text, "createdAt": row.ts})
		ev := tapEvent(row.uri, "bafy", "did:plc:a", "create", rec) // helper below
		if err := ix.Handle(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test/feed", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var body struct {
		Posts []struct {
			Text string `json:"text"`
		} `json:"posts"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Posts) != 3 {
		t.Fatalf("got %d posts want 3", len(body.Posts))
	}
	if body.Posts[0].Text != "third" || body.Posts[2].Text != "first" {
		t.Errorf("order wrong: %+v", body.Posts)
	}
}

// All records share createdAt; order among ties is unspecified. This
// test only checks count, not order — don't "fix" it.
func TestHandler_LimitRespected(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)
	h := testpipeline.NewHandler(pool)

	for i := 0; i < 5; i++ {
		rec, _ := json.Marshal(map[string]any{"text": "x", "createdAt": "2026-04-19T10:00:00Z"})
		ev := tapEvent(
			fmt.Sprintf("at://did:plc:a/social.craftsky.test.post/%d", i),
			"bafy", "did:plc:a", "create", rec,
		)
		if err := ix.Handle(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test/feed?limit=2", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var body struct{ Posts []any `json:"posts"` }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Posts) != 2 {
		t.Errorf("got %d want 2", len(body.Posts))
	}
}

// Seeds 201 rows so the clamp is actually observable in the response.
// Without the clamp the handler would return 201; with the clamp it
// returns 200.
func TestHandler_LimitClampedTo200(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	ix := testpipeline.NewIndexer(pool)
	h := testpipeline.NewHandler(pool)

	for i := 0; i < 201; i++ {
		rec, _ := json.Marshal(map[string]any{"text": "x", "createdAt": "2026-04-19T10:00:00Z"})
		ev := tapEvent(
			fmt.Sprintf("at://did:plc:a/social.craftsky.test.post/%d", i),
			"bafy", "did:plc:a", "create", rec,
		)
		if err := ix.Handle(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test/feed?limit=999", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 (limit should clamp, not 400)", rec.Code)
	}
	var body struct{ Posts []any `json:"posts"` }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Posts) != 200 {
		t.Errorf("got %d posts, want 200 (clamp)", len(body.Posts))
	}
}

func TestHandler_InvalidLimitReturns400(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)
	h := testpipeline.NewHandler(pool)

	for _, q := range []string{"abc", "-1", "0"} {
		req := httptest.NewRequest(http.MethodGet, "/test/feed?limit="+q, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("limit=%q: got %d want 400", q, rec.Code)
		}
	}
}

// tapEvent is a tiny test helper kept local to avoid exporting a helper
// that will be deleted with the package.
func tapEvent(uri, cid, did, action string, record []byte) tap.Event {
	return tap.Event{
		URI: uri, CID: cid, DID: did,
		Collection: "social.craftsky.test.post",
		Action:     action, Record: record,
	}
}
```

Note: method filtering (POST /test/feed → 405) is delegated to `http.ServeMux`'s method-aware routing (`mux.Handle("GET /test/feed", ...)`), so no in-handler check is tested here.

- [ ] **Step 2: Run tests to verify they fail**

Run: `just test ./appview/internal/testpipeline/...`
Expected: FAIL — `NewHandler` undefined.

### Task 4.2: Implement the handler

**Files:**
- Create: `appview/internal/testpipeline/handler.go`

- [ ] **Step 1: Write the implementation**

```go
package testpipeline

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler serves GET /test/feed. See doc.go — disposable.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler returns an HTTP handler backed by pool.
func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

const (
	defaultLimit = 50
	maxLimit     = 200
)

type feedPost struct {
	URI       string    `json:"uri"`
	CID       string    `json:"cid"`
	DID       string    `json:"did"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	IndexedAt time.Time `json:"indexedAt"`
}

type feedResponse struct {
	Posts []feedPost `json:"posts"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	limit := defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if n > maxLimit {
			n = maxLimit
		}
		limit = n
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT uri, cid, did, text, created_at, indexed_at
		   FROM test_posts
		  ORDER BY created_at DESC
		  LIMIT $1`,
		limit,
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Non-nil zero-length slice so it serialises as [] not null.
	posts := make([]feedPost, 0)
	for rows.Next() {
		var p feedPost
		if err := rows.Scan(&p.URI, &p.CID, &p.DID, &p.Text, &p.CreatedAt, &p.IndexedAt); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(feedResponse{Posts: posts})
}
```

- [ ] **Step 2: Run tests to verify they all pass**

Run: `just test ./appview/internal/testpipeline/...`
Expected: all indexer + handler tests PASS.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/testpipeline/handler.go appview/internal/testpipeline/handler_test.go
git commit -m "feat(testpipeline): add GET /test/feed handler"
```

---

**CHUNK 4 REVIEW GATE:** Dispatch plan-document-reviewer on Chunk 4. Fix, re-dispatch, iterate.

---

## Chunk 5: Wiring

Dispatcher into `deps.go`; handler into `routes.go`. Plus the end-to-end integration test that justifies the whole slice.

### Task 5.1: Wire the dispatcher into deps

**Files:**
- Modify: `appview/internal/app/deps.go`

- [ ] **Step 1: Update `newDeps` to build a Dispatcher**

Replace the block around the current indexer construction (lines ~98–118) so that:
- `BlueskyPostsSample` is still constructed (unchanged).
- `testpipeline.Indexer` is constructed.
- Both are registered on a `Dispatcher` whose fallback is `index.NotImplemented{}`.
- The dispatcher is assigned to `deps.Indexer` and passed as `Indexer` into `tap.NewWSConsumer`.

Target shape:

```go
	blueskySample := index.NewBlueskyPostsSample(pool)
	testpipelineIdx := testpipeline.NewIndexer(pool)

	dispatcher := index.NewDispatcher(index.NotImplemented{})
	dispatcher.Register("app.bsky.feed.post", blueskySample)
	dispatcher.Register("social.craftsky.test.post", testpipelineIdx)

	deps := &Deps{
		Config:               cfg,
		Logger:               logger,
		DB:                   pool,
		OAuthApp:             oauthApp,
		OAuthStore:           oauthStore,
		CraftskySessionStore: craftskyStore,
		Indexer:              dispatcher,
		Consumer:             tap.NotImplemented{}, // temp, replaced below
	}

	deps.Consumer = tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          cfg.TapWSURL,
		Indexer:      dispatcher,
		AckTimeout:   cfg.TapAckTimeout,
		ReconnectMax: cfg.TapReconnectMax,
		MaxRetries:   cfg.TapMaxRetries,
		Logger:       logger,
	})
```

Add the import: `"social.craftsky/appview/internal/testpipeline"`.

- [ ] **Step 2: Run existing deps tests — they should still pass**

Run: `just test ./appview/internal/app/...`
Expected: PASS. `deps_test.go` does not assert on `deps.Indexer`'s concrete type (verified during plan review), so no edits to the existing tests are required.

**Dependencies this task relies on (must already be merged):** `index.Dispatcher` and `index.NewDispatcher` from Chunk 2; `testpipeline.NewIndexer` from Chunk 3. Verify with `just test ./appview/internal/index/... ./appview/internal/testpipeline/...` before starting this step.

- [ ] **Step 3: Run the whole test suite**

Run: `just test`
Expected: PASS across all packages.

- [ ] **Step 4: Commit**

```bash
git add appview/internal/app/deps.go appview/internal/app/deps_test.go
git commit -m "feat(app): wire index.Dispatcher with bluesky + testpipeline indexers"
```

(Only include `deps_test.go` in the commit if it needed an update.)

### Task 5.2: Register `GET /test/feed` in dev only

**Files:**
- Modify: `appview/internal/routes/routes.go`

- [ ] **Step 1: Add the dev-gated route registration**

Inside `AddRoutes`, *before* the `Fallthrough` block (the `mux.Handle("/", http.NotFoundHandler())` line), add:

```go
	// Disposable test pipeline (GET /test/feed). Dev only — see
	// docs/superpowers/specs/2026-04-19-test-pipeline-design.md.
	if deps.Config.Env == app.EnvDev {
		mux.Handle("GET /test/feed", testpipeline.NewHandler(deps.DB))
	}
```

Add the import: `"social.craftsky/appview/internal/testpipeline"`.

- [ ] **Step 2: Generalize the `testDeps` helper and add the route-registration test**

The existing `testDeps()` helper in `routes_test.go` hard-codes `Env: app.EnvDev`. Generalize it to accept an env, preserving the existing call sites:

```go
func testDeps() *app.Deps { return testDepsEnv(app.EnvDev) }

func testDepsEnv(env app.Env) *app.Deps {
	return &app.Deps{
		Config:      app.Config{Env: env, AllowedOrigins: []string{"*"}, DevDID: "did:plc:test"},
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		AuthService: &auth.MockAuthService{DefaultDID: "did:plc:test"},
	}
}
```

Then add the new test:

```go
func TestAddRoutes_TestFeedDevOnly(t *testing.T) {
	// Dev: route registered.
	mux := http.NewServeMux()
	AddRoutes(context.Background(), mux, testDepsEnv(app.EnvDev))
	req := httptest.NewRequest(http.MethodGet, "/test/feed", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		t.Error("dev: /test/feed should be registered")
	}

	// Prod: route NOT registered.
	mux = http.NewServeMux()
	AddRoutes(context.Background(), mux, testDepsEnv(app.EnvProd))
	req = httptest.NewRequest(http.MethodGet, "/test/feed", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("prod: /test/feed should be 404, got %d", rec.Code)
	}
}
```

The test doesn't hit the handler's DB path — only route registration matters — so the nil `DB` on the fixture is fine. `AddRoutes` calls `auth.NewHTTPHandlers(...)` before the env gate, but the existing tests confirm that constructor tolerates nil OAuth deps at registration time (they only fail if an OAuth handler is invoked).

- [ ] **Step 3: Run tests**

Run: `just test ./appview/internal/routes/...`
Expected: PASS, including the new test.

- [ ] **Step 4: Commit**

```bash
git add appview/internal/routes/routes.go appview/internal/routes/routes_test.go
git commit -m "feat(routes): register GET /test/feed in dev only"
```

### Task 5.3: End-to-end integration test

**Files:**
- Create: `appview/internal/testpipeline/integration_test.go`

- [ ] **Step 1: Write the end-to-end test**

```go
package testpipeline_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testpipeline"
)

// TestPipelineEndToEnd wires the real Dispatcher → Indexer → Postgres →
// Handler chain, pushes a synthetic Tap event in one end, and asserts the
// record comes back out of GET /test/feed. If this test fails, the
// indexing half of the pipeline is broken. (The WS consumer → Dispatcher
// leg is covered by existing tap package tests.)
func TestPipelineEndToEnd(t *testing.T) {
	t.Parallel()
	pool := withSchema(t)

	dispatcher := index.NewDispatcher(index.NotImplemented{})
	dispatcher.Register("social.craftsky.test.post", testpipeline.NewIndexer(pool))

	rec, _ := json.Marshal(map[string]any{
		"text":      "end-to-end",
		"createdAt": "2026-04-19T10:00:00Z",
	})
	ev := tap.Event{
		URI:        "at://did:plc:e2e/social.craftsky.test.post/3kxzzz",
		CID:        "bafyzzz",
		DID:        "did:plc:e2e",
		Collection: "social.craftsky.test.post",
		Rkey:       "3kxzzz",
		Action:     "create",
		Record:     rec,
	}
	if err := dispatcher.Handle(context.Background(), ev); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test/feed", nil)
	w := httptest.NewRecorder()
	testpipeline.NewHandler(pool).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", w.Code)
	}
	var body struct {
		Posts []struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"posts"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, p := range body.Posts {
		if p.URI == ev.URI && p.Text == "end-to-end" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("posted record not in feed: %+v", body.Posts)
	}
}
```

- [ ] **Step 2: Run the integration test**

Run: `just test ./appview/internal/testpipeline/ -run TestPipelineEndToEnd -v`
Expected: PASS.

- [ ] **Step 3: Run the full test suite one more time**

Run: `just test`
Expected: PASS across all packages.

- [ ] **Step 4: Commit**

```bash
git add appview/internal/testpipeline/integration_test.go
git commit -m "test(testpipeline): end-to-end dispatcher→indexer→handler test"
```

---

**CHUNK 5 REVIEW GATE:** Dispatch plan-document-reviewer on Chunk 5. Fix, re-dispatch, iterate.

---

## Chunk 6: Manual verification

Prove the pipeline works against the real compose stack, not just the test harness.

### Task 6.1: Smoke-test against a running stack

The real firehose → Tap → indexer path is covered by `TestPipelineEndToEnd` using a synthetic `tap.Event`. The *compose stack cannot* exercise a real-firehose loop for `social.craftsky.test.post`: docker-compose.yml has no local PDS, and `TAP_COLLECTION_FILTERS` is set to `app.bsky.feed.post` only. So this manual smoke test validates the HTTP handler + DB read path against the real running process — not the Tap path.

- [ ] **Step 1: Start the stack and confirm health**

```bash
just dev-d
curl -sS localhost:8080/healthz
```
Expected: non-error JSON response. (Exact shape is defined in `appview/internal/api/healthz.go`; failure modes there aren't what we're validating here.)

- [ ] **Step 2: Confirm migration 000004 ran**

```bash
just psql -c "\d test_posts"
```
Expected: the `test_posts` table description prints. If it doesn't, run `just migrate up` before continuing.

- [ ] **Step 3: Seed a test row directly via psql**

```bash
just psql -c "INSERT INTO test_posts (uri, cid, did, text, created_at) VALUES ('at://did:plc:manual/social.craftsky.test.post/x', 'bafy', 'did:plc:manual', 'manual smoke test', now());"
```

- [ ] **Step 4: Hit the endpoint**

```bash
curl -sS localhost:8080/test/feed | python3 -m json.tool
```
(`python3 -m json.tool` avoids a `jq` dependency; if `jq` is available, use it.)
Expected: JSON response with a `posts` array containing the seeded row (`did:plc:manual`, text `"manual smoke test"`).

- [ ] **Step 5: Confirm prod gate**

The unit test `TestAddRoutes_TestFeedDevOnly` (Task 5.2) proves the route is only registered when `Env == EnvDev`. No additional manual check; just note in the PR description that the dev check in steps 3–4 passed.

- [ ] **Step 6: Stop the stack**

```bash
just down
```

### Task 6.2: Open the PR

- [ ] **Step 1: Verify `gh` is authed**

```bash
gh auth status
```
If not authed, stop and ask the user; don't proceed without auth.

- [ ] **Step 2: Push the branch**

```bash
git push -u origin claude/happy-swirles-d1f882
```

- [ ] **Step 3: Create the PR**

```bash
gh pr create --base main --title "feat(appview): end-to-end test pipeline via social.craftsky.test.post" --body "$(cat <<'EOF'
## Summary

- Adds a **disposable** `social.craftsky.test.post` lexicon + `test_posts` table + `GET /test/feed` endpoint to validate the firehose → index → Postgres → HTTP read API loop end to end.
- Introduces `index.Dispatcher` (the survivable piece), which routes `tap.Event`s by collection NSID to registered indexers. `BlueskyPostsSample` and `testpipeline.Indexer` are both registered through it.
- `/test/feed` is gated behind `deps.Config.Env == app.EnvDev` so it cannot leak to production.

## Test plan

- [ ] `just test` passes.
- [ ] Manual smoke test against `just dev`: seed a record, `curl /test/feed`, see it come back.
- [ ] Dev gate verified by unit test (`TestAddRoutes_TestFeedDevOnly`).

## Cleanup

The entire package `appview/internal/testpipeline/`, the lexicon `lexicon/social/craftsky/test/`, migration 000004, and the two wiring lines (one in `deps.go`, one in `routes.go`) get deleted when the real `social.craftsky.feed.post` indexer lands. The Dispatcher stays.

Spec: [docs/superpowers/specs/2026-04-19-test-pipeline-design.md](docs/superpowers/specs/2026-04-19-test-pipeline-design.md)

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 4: Verify the PR URL**

Expected: gh prints the URL. Share it with the user.

---

## Notes

- **Commit cadence:** one logical change per commit, as per the plan's step breakdown. Don't squash intermediate test-failing commits into the final one — the failing-test-first commits are valuable history.
- **Skill usage:**
  - Use @superpowers:test-driven-development throughout (every indexer/handler task follows write-test-fail-implement-pass).
  - Use @superpowers:verification-before-completion before the Chunk 6 PR step — only claim the PR is ready after `just test` and manual curl both pass.
- **Reality checks baked into the plan:** the spec mentioned sqlc but this repo hand-writes queries with pgx (see `bluesky_posts_sample.go`). The plan follows the real pattern. If you encounter any other mismatch between spec and repo, update the plan before implementing.
- **What NOT to do:**
  - Don't delete `bluesky_posts_sample.go` or its table. That's a separate cleanup.
  - Don't add sqlc config to this PR. The project doesn't use it yet; introducing it is a separate decision.
  - Don't add pagination, auth, or a cursor to `/test/feed`. It's a diagnostic.
  - Don't write to a PDS from the appview in this PR. BFF writes are the next slice.
