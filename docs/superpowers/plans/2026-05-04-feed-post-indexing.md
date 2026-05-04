# Feed Post Indexing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the storage half of the Craftsky feed: a `CraftskyPost` indexer that consumes `social.craftsky.feed.post` events from Tap and writes a new `craftsky_posts` Postgres table, per [`2026-05-04-feed-post-indexing-design.md`](../specs/2026-05-04-feed-post-indexing-design.md).

**Architecture:** A new `Indexer` registered with the existing `index.Dispatcher` ([appview/internal/index/dispatcher.go](../../../appview/internal/index/dispatcher.go)). One Postgres migration adds `craftsky_posts` with a foreign key to `craftsky_profiles(did) ON DELETE CASCADE` so leaving Craftsky removes the user's posts atomically. The indexer is gated on `craftsky_profiles` membership (matches `BlueskyProfile`), idempotent on `(URI, CID)` per the existing replay-skip pattern, materialises text + non-project fields (`facets`, `images`, `reply_*`, `quote_*`) plus a hashtag `tags TEXT[]` extracted from facet tag features, and stores the full record as `JSONB` for forward-compat. Like/repost indexing, project-field materialisation, and any read endpoint are out of scope.

**Tech Stack:** Go 1.22+, `pgx/v5`, `indigo` (`atproto/syntax`, `lexutil`, `bsky.RichtextFacet*`, `comatproto.RepoStrongRef`), generated lexicon types in `appview/internal/lexicon/craftsky/`, `golang-migrate/v4`. Tests run via `just test` against the compose Postgres.

---

## Background reading for the implementer

Read these before starting. Short but load-bearing.

- [`docs/superpowers/specs/2026-05-04-feed-post-indexing-design.md`](../specs/2026-05-04-feed-post-indexing-design.md) — the spec this plan implements. **Primary source of truth.**
- [`docs/superpowers/specs/2026-04-23-post-lexicon-fields-design.md`](../specs/2026-04-23-post-lexicon-fields-design.md) — the locked lexicon shape; explains why `tags` is a multi-source search column and why facet-derived tags don't enforce the kebab-case pattern.
- [`docs/superpowers/specs/2026-04-17-tap-integration-design.md`](../specs/2026-04-17-tap-integration-design.md) — the Tap consumer's at-least-once delivery semantics and the `MaxRetries` poison-pill cap.
- [`appview/internal/index/dispatcher.go`](../../../appview/internal/index/dispatcher.go) — how indexers are wired in.
- [`appview/internal/index/craftsky_profile.go`](../../../appview/internal/index/craftsky_profile.go) — the established indexer pattern (idempotent upsert, `record_cid IS DISTINCT FROM` replay filter, in-transaction delete).
- [`appview/internal/index/bluesky_profile.go`](../../../appview/internal/index/bluesky_profile.go) — the established membership-gated pattern.
- [`appview/internal/index/craftsky_profile_test.go`](../../../appview/internal/index/craftsky_profile_test.go) and [`bluesky_profile_test.go`](../../../appview/internal/index/bluesky_profile_test.go) — test idioms (`testdb.WithSchema`, inline DDL, `tap.Event` literals).
- [`appview/internal/lexicon/craftsky/feedpost.go`](../../../appview/internal/lexicon/craftsky/feedpost.go) — the generated `FeedPost` types this indexer decodes into. **Do not hand-roll a narrow record struct in the indexer file** — `AGENTS.md` says use the generated types.
- [`appview/internal/tap/consumer.go`](../../../appview/internal/tap/consumer.go) `Event` struct — what every indexer receives.
- [`AGENTS.md`](../../../AGENTS.md) — project rules (Go conventions, indexer registration contract, `syntax.*` typed wrappers).
- [`docker-compose.yml`](../../../docker-compose.yml) — the `TAP_COLLECTION_FILTERS` env var that decides which NSIDs Tap forwards.

## Conventions this plan follows

- **TDD.** Every task writes the test first, confirms it fails, writes minimal code to pass, confirms it passes, commits. Don't batch.
- **One commit per task.** Tasks are small. Frequent commits make reverts cheap.
- **`just test` is the only test runner.** Requires `just dev-d` running so integration tests can hit the compose Postgres at `localhost:5433`.
- **`just fmt` after every non-trivial change.** Don't commit Go files that haven't been `gofmt`'d.
- **Naming.** snake_case for SQL identifiers, UpperCamelCase for Go exports, lowerCamelCase for Go locals. Matches the rest of the codebase.
- **No emojis in code, comments, or commit messages.**
- **Comments only when the *why* is non-obvious.** Don't restate what the code does.

## File structure

All paths are relative to repo root.

**New files:**

- `appview/migrations/000010_craftsky_posts.up.sql` — schema.
- `appview/migrations/000010_craftsky_posts.down.sql` — rollback.
- `appview/internal/index/craftsky_post.go` — the `CraftskyPost` indexer.
- `appview/internal/index/craftsky_post_test.go` — its tests.

**Modified files:**

- `appview/internal/app/deps.go` — register the new indexer with the dispatcher.
- `docker-compose.yml` — extend `TAP_COLLECTION_FILTERS` to include `social.craftsky.feed.post`.

**No deletes.**

## Chunk boundaries

Each chunk is self-contained: it ends with passing tests and a single commit (or two if test + impl are split for visibility). Chunks build linearly — Chunk N depends on Chunks 1..N-1.

- **Chunk 1: State check.** Verify migration head is `000009` and confirm `craftsky_profiles(did)` exists (the FK target).
- **Chunk 2: Migration.** Add `000010_craftsky_posts.up.sql` + `.down.sql`. Run the up migration in dev. Verify the schema.
- **Chunk 3: Indexer skeleton + plain text upsert.** New file with `Handle` dispatching on action, plain text upsert (no facets/images/reply/quote), membership gate. Covers the 80% case end-to-end.
- **Chunk 4: Facets, tags, images.** Extend upsert to materialise the rendering-relevant fields. Includes the tag-extraction helper.
- **Chunk 5: Reply and quote pointers.** Extend upsert to materialise structural pointers.
- **Chunk 6: Update idempotency.** Replay-skips-update test (same CID) and update-replaces test (new CID).
- **Chunk 7: Delete and cascade.** Direct delete + FK cascade via parent profile delete.
- **Chunk 8: Wiring.** Register the indexer in `deps.go`, extend `TAP_COLLECTION_FILTERS`, smoke test in compose.

---

## Chunk 1: State check

Quick verification that the codebase still matches what this plan was written against. Bail out early if migration numbers have drifted.

### Task 1.1: Verify current migration head

**Files:**
- Inspect: `appview/migrations/`

- [ ] **Step 1:** List migrations to confirm the current head.

```bash
ls appview/migrations/
```

Expected: highest numeric prefix is `000009`. If a migration has landed since this plan was written, increment the new migration's prefix to be `head+1` and use that prefix everywhere in this plan instead of `000010`. Migration numbers must stay contiguous.

- [ ] **Step 2:** Confirm `craftsky_profiles` exists in `000008` and has `did` as its primary key.

```bash
grep -A 6 "CREATE TABLE craftsky_profiles" appview/migrations/000008_craftsky_profiles.up.sql
```

Expected output begins with `CREATE TABLE craftsky_profiles (` and `did TEXT NOT NULL PRIMARY KEY`. If either is missing, stop and update the plan — this is the FK target.

### Task 1.2: Confirm Tap collection filter location

**Files:**
- Inspect: `docker-compose.yml`

- [ ] **Step 1:** Find the env var.

```bash
grep -n "TAP_COLLECTION_FILTERS" docker-compose.yml
```

Expected: one match around line 53, currently set to `"social.craftsky.actor.profile,app.bsky.actor.profile"`. Note the line number; you'll edit it in Chunk 8.

---

## Chunk 2: Migration

Adds the `craftsky_posts` table with FK to `craftsky_profiles(did) ON DELETE CASCADE`, the partial indexes for reply/quote columns, and the GIN index on `tags`.

### Task 2.1: Create the up migration

**Files:**
- Create: `appview/migrations/000010_craftsky_posts.up.sql`

- [ ] **Step 1:** Write the file.

```sql
-- appview/migrations/000010_craftsky_posts.up.sql
-- See docs/superpowers/specs/2026-05-04-feed-post-indexing-design.md.
CREATE TABLE craftsky_posts (
    uri              TEXT        NOT NULL PRIMARY KEY,
    did              TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey             TEXT        NOT NULL,
    cid              TEXT        NOT NULL,

    text             TEXT        NOT NULL,
    facets           JSONB,
    images           JSONB,

    reply_root_uri   TEXT,
    reply_root_cid   TEXT,
    reply_parent_uri TEXT,
    reply_parent_cid TEXT,

    quote_uri        TEXT,
    quote_cid        TEXT,

    tags             TEXT[]      NOT NULL DEFAULT '{}',

    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (did, rkey)
);

CREATE INDEX craftsky_posts_indexed_at_desc
    ON craftsky_posts (indexed_at DESC);
CREATE INDEX craftsky_posts_did_indexed_at_desc
    ON craftsky_posts (did, indexed_at DESC);
CREATE INDEX craftsky_posts_reply_parent_uri
    ON craftsky_posts (reply_parent_uri) WHERE reply_parent_uri IS NOT NULL;
CREATE INDEX craftsky_posts_reply_root_uri
    ON craftsky_posts (reply_root_uri)   WHERE reply_root_uri   IS NOT NULL;
CREATE INDEX craftsky_posts_quote_uri
    ON craftsky_posts (quote_uri)        WHERE quote_uri        IS NOT NULL;
CREATE INDEX craftsky_posts_tags_gin
    ON craftsky_posts USING GIN (tags);
```

### Task 2.2: Create the down migration

**Files:**
- Create: `appview/migrations/000010_craftsky_posts.down.sql`

- [ ] **Step 1:** Write the file. Postgres drops the indexes when the table goes; explicit `DROP TABLE IF EXISTS` is enough.

```sql
-- appview/migrations/000010_craftsky_posts.down.sql
DROP TABLE IF EXISTS craftsky_posts;
```

### Task 2.3: Run the up migration in compose

**Files:**
- (none — runs against the running stack)

- [ ] **Step 1:** Make sure compose is up.

```bash
just dev-d
```

Expected: containers start; no migrate errors in the logs.

- [ ] **Step 2:** Apply the new migration.

```bash
just migrate up
```

Expected: `1/u craftsky_posts (...)` line in stdout, no errors.

- [ ] **Step 3:** Verify the table exists with the right shape.

```bash
just psql -c '\d craftsky_posts'
```

Expected: every column listed in the up migration is present, with `uri` flagged `not null` and primary key, `did` flagged `not null` and foreign key to `craftsky_profiles(did) ON DELETE CASCADE`, the six indexes listed at the bottom.

- [ ] **Step 4:** Verify the cascade direction.

```bash
just psql -c "
INSERT INTO craftsky_profiles (did, record_cid) VALUES ('did:plc:cascadetest', 'seed');
INSERT INTO craftsky_posts (uri, did, rkey, cid, text, record, created_at)
VALUES ('at://did:plc:cascadetest/social.craftsky.feed.post/k1', 'did:plc:cascadetest', 'k1', 'cidA',
        'hello', '{}'::jsonb, now());
DELETE FROM craftsky_profiles WHERE did = 'did:plc:cascadetest';
SELECT count(*) FROM craftsky_posts WHERE did = 'did:plc:cascadetest';
"
```

Expected: final `count` is `0` (cascade removed the post).

- [ ] **Step 5:** Verify rollback works on a fresh stack.

```bash
just migrate down 1
just psql -c '\d craftsky_posts'
```

Expected: the second command prints `Did not find any relation named "craftsky_posts"`.

- [ ] **Step 6:** Re-apply the up migration so the rest of the plan can run against it.

```bash
just migrate up
```

### Task 2.4: Commit the migration

- [ ] **Step 1:** Stage and commit.

```bash
git add appview/migrations/000010_craftsky_posts.up.sql appview/migrations/000010_craftsky_posts.down.sql
git commit -m "feat(appview): add craftsky_posts migration"
```

---

## Chunk 3: Indexer skeleton + plain text upsert

The 80% case end to end. After this chunk, a plain text `social.craftsky.feed.post` event from Tap produces one row in `craftsky_posts`; an event whose author isn't in `craftsky_profiles` is dropped silently; events for other collections are ignored without error; an unknown action errors.

The indexer file ends this chunk with a working `Handle`, `handleUpsert` (text-only path), and `handleDelete` stub. Later chunks extend `handleUpsert` for facets/tags/images/reply/quote, and replace the delete stub with the real implementation.

### Task 3.1: Add the test fixture DDL helper

**Files:**
- Create: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Write the file with the test fixture DDL and a small helper to seed a `craftsky_profiles` row. The constant must be a *snapshot* of the schema (not a `\i`-include of the migration), to keep tests free of file dependencies.

```go
// appview/internal/index/craftsky_post_test.go
package index_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// craftskyPostsDDL mirrors appview/migrations/000010_craftsky_posts.up.sql.
// craftsky_profiles is needed because craftsky_posts has a FK into it.
const craftskyPostsDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE craftsky_posts (
    uri              TEXT        NOT NULL PRIMARY KEY,
    did              TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey             TEXT        NOT NULL,
    cid              TEXT        NOT NULL,

    text             TEXT        NOT NULL,
    facets           JSONB,
    images           JSONB,

    reply_root_uri   TEXT,
    reply_root_cid   TEXT,
    reply_parent_uri TEXT,
    reply_parent_cid TEXT,

    quote_uri        TEXT,
    quote_cid        TEXT,

    tags             TEXT[]      NOT NULL DEFAULT '{}',

    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (did, rkey)
);
`

// seedCraftskyMember inserts a craftsky_profiles row so a post for did
// can pass the membership check.
func seedCraftskyMember(t *testing.T, pool *pgxpool.Pool, did string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
		did, "seed"); err != nil {
		t.Fatalf("seed craftsky_profiles: %v", err)
	}
}

// fixedCreatedAt is a constant timestamp used in test events so assertions
// on `created_at` are exact. Chosen arbitrarily; not load-bearing.
const fixedCreatedAt = "2026-05-04T12:00:00Z"

// testTime parses fixedCreatedAt for comparisons that need a time.Time.
func testTime(t *testing.T) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, fixedCreatedAt)
	if err != nil {
		t.Fatalf("parse fixed time: %v", err)
	}
	return tt
}
```

- [ ] **Step 2:** Format and verify it compiles (it doesn't reference `index.NewCraftskyPost` yet, but it does import the package — that's fine; tests in this file are added in later steps).

```bash
just fmt
cd appview && go build ./internal/index/...
```

Expected: no errors. The unused-import check is satisfied because each imported package is used in the helpers.

- [ ] **Step 3:** Commit.

```bash
git add appview/internal/index/craftsky_post_test.go
git commit -m "test(appview): add craftsky_posts test fixture and helpers"
```

### Task 3.2: Write the failing skeleton tests

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the four skeleton tests below to the existing file.

```go
func TestCraftskyPost_OtherCollectionIgnored(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:b/app.bsky.feed.post/k",
		CID:        "c",
		DID:        "did:plc:b",
		Rkey:       "k",
		Collection: "app.bsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("want nil for other collection; got %v", err)
	}
}

func TestCraftskyPost_UnknownAction(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:a")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:a/social.craftsky.feed.post/k",
		CID:        "c",
		DID:        "did:plc:a",
		Rkey:       "k",
		Collection: "social.craftsky.feed.post",
		Action:     "weird",
		Record:     json.RawMessage(`{"text":"hi","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err == nil {
		t.Error("want error for unknown action; got nil")
	}
}

func TestCraftskyPost_Create_NonMember_DroppedSilently(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:nm/social.craftsky.feed.post/k",
		CID:        "c",
		DID:        "did:plc:nm",
		Rkey:       "k",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"hi","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("Handle should drop non-members without error; got %v", err)
	}

	var count int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM craftsky_posts WHERE did = $1`, ev.DID).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 (non-member must not be indexed)", count)
	}
}

func TestCraftskyPost_Create_PlainText(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:m")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:m/social.craftsky.feed.post/r1",
		CID:        "bafy1",
		DID:        "did:plc:m",
		Rkey:       "r1",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "first post",
			"createdAt": "` + fixedCreatedAt + `"
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var (
		uri, did, rkey, cid, text string
		facets, images            *string
		replyRoot, replyParent    *string
		quoteURI, quoteCID        *string
		tags                      []string
		createdAt                 time.Time
	)
	err := pool.QueryRow(context.Background(), `
		SELECT uri, did, rkey, cid, text,
		       facets::text, images::text,
		       reply_root_uri, reply_parent_uri,
		       quote_uri, quote_cid,
		       tags, created_at
		FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&uri, &did, &rkey, &cid, &text,
			&facets, &images,
			&replyRoot, &replyParent,
			&quoteURI, &quoteCID,
			&tags, &createdAt)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if did != "did:plc:m" || rkey != "r1" || cid != "bafy1" {
		t.Errorf("ident = (%q,%q,%q)", did, rkey, cid)
	}
	if text != "first post" {
		t.Errorf("text = %q", text)
	}
	if facets != nil || images != nil {
		t.Errorf("facets/images should be NULL on plain text post; got facets=%v images=%v", facets, images)
	}
	if replyRoot != nil || replyParent != nil || quoteURI != nil || quoteCID != nil {
		t.Errorf("reply/quote columns should be NULL on plain text post")
	}
	if len(tags) != 0 {
		t.Errorf("tags = %v, want empty", tags)
	}
	if !createdAt.Equal(testTime(t)) {
		t.Errorf("created_at = %v, want %v", createdAt, testTime(t))
	}
}
```

- [ ] **Step 2:** Verify the tests fail to compile (because `index.NewCraftskyPost` doesn't exist yet).

```bash
just test
```

Expected: build error mentioning `index.NewCraftskyPost`. That's the failing-test signal — `go test` won't even compile without the symbol.

### Task 3.3: Implement the skeleton

**Files:**
- Create: `appview/internal/index/craftsky_post.go`

- [ ] **Step 1:** Write the file. The membership check, decode, parse-`createdAt`, and minimal upsert (text + record + created_at, NULLs everywhere else) are all done here. Facets/tags/images/reply/quote are wired in subsequent chunks.

```go
// appview/internal/index/craftsky_post.go
package index

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/tap"
)

// CraftskyPost indexes social.craftsky.feed.post events into craftsky_posts.
// Required invariant: idempotent on (URI, CID). Tap delivers at-least-once.
//
// Posts are gated on craftsky_profiles membership: events from non-members
// are dropped silently, matching BlueskyProfile's pattern. A post arriving
// before its author's craftsky_profiles row is dropped permanently for now;
// see the design spec for the post-backfiller follow-up.
type CraftskyPost struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

var _ Indexer = (*CraftskyPost)(nil)

func NewCraftskyPost(pool *pgxpool.Pool, logger *slog.Logger) *CraftskyPost {
	if logger == nil {
		logger = slog.Default()
	}
	return &CraftskyPost{pool: pool, logger: logger}
}

const craftskyPostNSID syntax.NSID = "social.craftsky.feed.post"

func (c *CraftskyPost) Handle(ctx context.Context, ev tap.Event) error {
	if ev.Collection != craftskyPostNSID {
		return nil
	}
	switch ev.Action {
	case "create", "update":
		return c.handleUpsert(ctx, ev)
	case "delete":
		return c.handleDelete(ctx, ev)
	default:
		return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
	}
}

func (c *CraftskyPost) handleUpsert(ctx context.Context, ev tap.Event) error {
	isMember, err := c.isMember(ctx, ev.DID)
	if err != nil {
		return fmt.Errorf("membership check %s: %w", ev.DID, err)
	}
	if !isMember {
		return nil
	}

	var rec craftskylex.FeedPost
	if err := json.Unmarshal(ev.Record, &rec); err != nil {
		return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
	}
	createdAt, err := time.Parse(time.RFC3339, rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("parse createdAt %q on %s: %w", rec.CreatedAt, ev.URI, err)
	}

	// Materialised columns. Subsequent chunks fill the nil/empty values
	// from rec.Facets, rec.Images, rec.Reply, rec.Embed.
	const q = `
		INSERT INTO craftsky_posts
			(uri, did, rkey, cid, text, facets, images,
			 reply_root_uri, reply_root_cid, reply_parent_uri, reply_parent_cid,
			 quote_uri, quote_cid, tags, record, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (uri) DO UPDATE SET
			cid              = EXCLUDED.cid,
			text             = EXCLUDED.text,
			facets           = EXCLUDED.facets,
			images           = EXCLUDED.images,
			reply_root_uri   = EXCLUDED.reply_root_uri,
			reply_root_cid   = EXCLUDED.reply_root_cid,
			reply_parent_uri = EXCLUDED.reply_parent_uri,
			reply_parent_cid = EXCLUDED.reply_parent_cid,
			quote_uri        = EXCLUDED.quote_uri,
			quote_cid        = EXCLUDED.quote_cid,
			tags             = EXCLUDED.tags,
			record           = EXCLUDED.record,
			created_at       = EXCLUDED.created_at,
			indexed_at       = now()
		WHERE craftsky_posts.cid IS DISTINCT FROM EXCLUDED.cid
	`
	_, err = c.pool.Exec(ctx, q,
		ev.URI, ev.DID, ev.Rkey, ev.CID,
		rec.Text,
		nil, nil, // facets, images — Chunk 4
		nil, nil, // reply_root_*
		nil, nil, // reply_parent_*
		nil, nil, // quote_*       — Chunk 5
		[]string{}, // tags         — Chunk 4
		ev.Record,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("upsert %s: %w", ev.URI, err)
	}
	return nil
}

func (c *CraftskyPost) handleDelete(ctx context.Context, ev tap.Event) error {
	// Real implementation lands in Chunk 7. Returning nil here is fine
	// for now — no test in this chunk exercises the delete path against
	// a populated row.
	_ = ev
	_ = ctx
	return nil
}

func (c *CraftskyPost) isMember(ctx context.Context, did syntax.DID) (bool, error) {
	var exists bool
	err := c.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1)`, did).
		Scan(&exists)
	return exists, err
}
```

- [ ] **Step 2:** Format.

```bash
just fmt
```

Expected: clean exit.

- [ ] **Step 3:** Run the tests written in Task 3.2.

```bash
just test
```

Expected: all four tests in `craftsky_post_test.go` pass. Existing indexer tests continue to pass.

- [ ] **Step 4:** Commit.

```bash
git add appview/internal/index/craftsky_post.go appview/internal/index/craftsky_post_test.go
git commit -m "feat(appview): add CraftskyPost indexer skeleton"
```

### Task 3.4: Project-payload-in-record round-trip test

The spec promises the `record JSONB` column preserves project payloads losslessly, even though no project columns exist in this pass. This test locks that contract in.

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test.

```go
func TestCraftskyPost_Create_WithProjectPayload_StoredInRecordOnly(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:p")
	idx := index.NewCraftskyPost(pool, testLogger())

	const projectJSON = `{
		"$type": "social.craftsky.feed.post",
		"text": "finished the shawl!",
		"createdAt": "` + fixedCreatedAt + `",
		"project": {
			"common": {
				"craftType": "social.craftsky.feed.defs#knitting",
				"status":    "social.craftsky.feed.defs#finished",
				"title":     "Hitchhiker Shawl",
				"materials": ["merino"],
				"tags":      ["fair-isle"]
			}
		}
	}`
	ev := tap.Event{
		URI:        "at://did:plc:p/social.craftsky.feed.post/r",
		CID:        "bafyP",
		DID:        "did:plc:p",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(projectJSON),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// The materialised columns must NOT carry project data this pass —
	// `tags` is from facets only, the read endpoint will not see project
	// fields until the project-fields spec lands.
	var (
		tags     []string
		recRaw   string
	)
	if err := pool.QueryRow(context.Background(),
		`SELECT tags, record::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&tags, &recRaw); err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("tags = %v, want empty (project tags are not yet materialised)", tags)
	}

	// The raw record column must round-trip the project payload byte-for-meaning.
	var got map[string]any
	if err := json.Unmarshal([]byte(recRaw), &got); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	project, ok := got["project"].(map[string]any)
	if !ok {
		t.Fatalf("record.project missing or not an object; got %T", got["project"])
	}
	common, ok := project["common"].(map[string]any)
	if !ok {
		t.Fatalf("record.project.common missing")
	}
	if common["craftType"] != "social.craftsky.feed.defs#knitting" {
		t.Errorf("craftType = %v", common["craftType"])
	}
	if common["title"] != "Hitchhiker Shawl" {
		t.Errorf("title = %v", common["title"])
	}
}
```

- [ ] **Step 2:** Run. Expect pass — the skeleton already writes `ev.Record` verbatim, and `text` is parsed independently of `project`.

```bash
just test -run TestCraftskyPost_Create_WithProjectPayload
```

### Task 3.5: Malformed-createdAt test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test.

```go
func TestCraftskyPost_MalformedCreatedAt_Errors(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:bad")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:bad/social.craftsky.feed.post/r",
		CID:        "bafyBAD",
		DID:        "did:plc:bad",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"x","createdAt":"not-a-timestamp"}`),
	}
	if err := idx.Handle(context.Background(), ev); err == nil {
		t.Error("want error for unparseable createdAt; got nil")
	}

	var count int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM craftsky_posts WHERE uri = $1`, ev.URI).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 (malformed event must not insert a row)", count)
	}
}
```

- [ ] **Step 2:** Run. Expect pass — the skeleton's `time.Parse` returns an error and `handleUpsert` propagates it before the INSERT runs.

```bash
just test -run TestCraftskyPost_MalformedCreatedAt
```

- [ ] **Step 3:** Commit the two new tests.

```bash
git add appview/internal/index/craftsky_post_test.go
git commit -m "test(appview): cover record-roundtrip and malformed-createdAt cases"
```

---

## Chunk 4: Facets, tags, images

Extend the upsert path so `facets`, `images`, and `tags` are populated. Adds the tag-extraction helper. Tags are case-folded, trimmed, de-duped (preserving first-seen order).

Field-name reference (so you don't have to hunt through generated code):

- `craftskylex.FeedPost.Facets` — `[]*appbsky.RichtextFacet`.
- `appbsky.RichtextFacet.Features` — `[]*appbsky.RichtextFacet_Features_Elem`.
- `RichtextFacet_Features_Elem.RichtextFacet_Tag` — `*RichtextFacet_Tag`, non-nil when this feature is a hashtag.
- `RichtextFacet_Tag.Tag` — the hashtag string (without the `#`).
- `craftskylex.FeedPost.Images` — `[]*FeedPost_Image`.
- `FeedPost_Image.Image` — `*lexutil.LexBlob` with `Ref lexutil.LexLink` (call `.String()` for the CID) and `MimeType string`.
- `FeedPost_Image.Alt` — `string`.

### Task 4.1: Write the failing tag-extraction test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test.

```go
func TestCraftskyPost_Create_WithTagsFromFacets(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:t")
	idx := index.NewCraftskyPost(pool, testLogger())

	// Two #tag features (one duplicate after lowercasing/trimming) and
	// one #link feature that must NOT contribute a tag.
	ev := tap.Event{
		URI:        "at://did:plc:t/social.craftsky.feed.post/r",
		CID:        "bafyT",
		DID:        "did:plc:t",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "with tags #FairIsle #fairisle and a link",
			"createdAt": "` + fixedCreatedAt + `",
			"facets": [
				{
					"index": {"byteStart": 11, "byteEnd": 20},
					"features": [{"$type": "app.bsky.richtext.facet#tag", "tag": "FairIsle"}]
				},
				{
					"index": {"byteStart": 21, "byteEnd": 30},
					"features": [{"$type": "app.bsky.richtext.facet#tag", "tag": "  fairisle "}]
				},
				{
					"index": {"byteStart": 35, "byteEnd": 39},
					"features": [{"$type": "app.bsky.richtext.facet#link", "uri": "https://example.com"}]
				}
			]
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var (
		tags   []string
		facets *string
	)
	if err := pool.QueryRow(context.Background(),
		`SELECT tags, facets::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&tags, &facets); err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(tags) != 1 || tags[0] != "fairisle" {
		t.Errorf("tags = %v, want [fairisle]", tags)
	}
	if facets == nil {
		t.Errorf("facets column should be populated; got NULL")
	}
}
```

- [ ] **Step 2:** Run; expect failure.

```bash
just test -run TestCraftskyPost_Create_WithTagsFromFacets
```

Expected: test runs and fails — current implementation always writes `tags = []` and `facets = NULL`.

### Task 4.2: Add the tag-extraction helper and image flattener

**Files:**
- Modify: `appview/internal/index/craftsky_post.go`

- [ ] **Step 1:** Add imports for `strings` and the indigo bsky package. Replace the existing import block with:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/tap"
)
```

- [ ] **Step 2:** Append the helpers at the end of `craftsky_post.go`.

```go
// extractTags walks facets and pulls hashtag-feature tags. Lowercase,
// trim, drop empties, dedupe (preserve first-seen order). Always returns
// a non-nil slice — the column is NOT NULL DEFAULT '{}'.
func extractTags(facets []*appbsky.RichtextFacet) []string {
	if len(facets) == 0 {
		return []string{}
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, facet := range facets {
		if facet == nil {
			continue
		}
		for _, feat := range facet.Features {
			if feat == nil || feat.RichtextFacet_Tag == nil {
				continue
			}
			t := strings.ToLower(strings.TrimSpace(feat.RichtextFacet_Tag.Tag))
			if t == "" {
				continue
			}
			if _, dup := seen[t]; dup {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}

// flattenImages turns the lexicon's [{image: LexBlob, alt}, ...] array
// into the storage shape [{cid, mime, alt}, ...]. Returns nil when there
// are no images, so the caller can pass nil to the JSONB column for SQL NULL.
func flattenImages(images []*craftskylex.FeedPost_Image) []map[string]string {
	if len(images) == 0 {
		return nil
	}
	out := make([]map[string]string, 0, len(images))
	for _, img := range images {
		if img == nil || img.Image == nil {
			continue
		}
		out = append(out, map[string]string{
			"cid":  img.Image.Ref.String(),
			"mime": img.Image.MimeType,
			"alt":  img.Alt,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
```

- [ ] **Step 3:** Wire facets/tags/images into the upsert. Replace the body of `handleUpsert` with:

```go
func (c *CraftskyPost) handleUpsert(ctx context.Context, ev tap.Event) error {
	isMember, err := c.isMember(ctx, ev.DID)
	if err != nil {
		return fmt.Errorf("membership check %s: %w", ev.DID, err)
	}
	if !isMember {
		return nil
	}

	var rec craftskylex.FeedPost
	if err := json.Unmarshal(ev.Record, &rec); err != nil {
		return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
	}
	createdAt, err := time.Parse(time.RFC3339, rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("parse createdAt %q on %s: %w", rec.CreatedAt, ev.URI, err)
	}

	var facetsJSON []byte
	if len(rec.Facets) > 0 {
		facetsJSON, err = json.Marshal(rec.Facets)
		if err != nil {
			return fmt.Errorf("marshal facets %s: %w", ev.URI, err)
		}
	}

	var imagesJSON []byte
	if flat := flattenImages(rec.Images); flat != nil {
		imagesJSON, err = json.Marshal(flat)
		if err != nil {
			return fmt.Errorf("marshal images %s: %w", ev.URI, err)
		}
	}

	tags := extractTags(rec.Facets)

	const q = `
		INSERT INTO craftsky_posts
			(uri, did, rkey, cid, text, facets, images,
			 reply_root_uri, reply_root_cid, reply_parent_uri, reply_parent_cid,
			 quote_uri, quote_cid, tags, record, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (uri) DO UPDATE SET
			cid              = EXCLUDED.cid,
			text             = EXCLUDED.text,
			facets           = EXCLUDED.facets,
			images           = EXCLUDED.images,
			reply_root_uri   = EXCLUDED.reply_root_uri,
			reply_root_cid   = EXCLUDED.reply_root_cid,
			reply_parent_uri = EXCLUDED.reply_parent_uri,
			reply_parent_cid = EXCLUDED.reply_parent_cid,
			quote_uri        = EXCLUDED.quote_uri,
			quote_cid        = EXCLUDED.quote_cid,
			tags             = EXCLUDED.tags,
			record           = EXCLUDED.record,
			created_at       = EXCLUDED.created_at,
			indexed_at       = now()
		WHERE craftsky_posts.cid IS DISTINCT FROM EXCLUDED.cid
	`
	_, err = c.pool.Exec(ctx, q,
		ev.URI, ev.DID, ev.Rkey, ev.CID,
		rec.Text,
		facetsJSON, imagesJSON,
		nil, nil, // reply_root_*
		nil, nil, // reply_parent_*
		nil, nil, // quote_*       — Chunk 5
		tags,
		ev.Record,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("upsert %s: %w", ev.URI, err)
	}
	return nil
}
```

(The reply/quote NULL placeholders are kept; Chunk 5 fills them in.)

- [ ] **Step 4:** Format and run the new test plus existing tests.

```bash
just fmt
just test -run TestCraftskyPost
```

Expected: all `TestCraftskyPost_*` tests pass, including the new `WithTagsFromFacets`.

- [ ] **Step 5:** Commit.

```bash
git add appview/internal/index/craftsky_post.go appview/internal/index/craftsky_post_test.go
git commit -m "feat(appview): materialise facets, tags, images on craftsky_posts"
```

### Task 4.3: Add the images-only test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test. The image-storage shape must be `[{cid, mime, alt}, ...]`; we assert the JSON round-trips that way.

```go
func TestCraftskyPost_Create_WithImages(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:i")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:i/social.craftsky.feed.post/r",
		CID:        "bafyI",
		DID:        "did:plc:i",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "post with images",
			"createdAt": "` + fixedCreatedAt + `",
			"images": [
				{
					"image": {"$type":"blob","ref":{"$link":"bafkreiimg1"},"mimeType":"image/jpeg","size":12345},
					"alt": "first photo"
				},
				{
					"image": {"$type":"blob","ref":{"$link":"bafkreiimg2"},"mimeType":"image/png","size":54321},
					"alt": "second photo"
				}
			]
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var imagesJSON string
	if err := pool.QueryRow(context.Background(),
		`SELECT images::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&imagesJSON); err != nil {
		t.Fatalf("select: %v", err)
	}
	var images []map[string]string
	if err := json.Unmarshal([]byte(imagesJSON), &images); err != nil {
		t.Fatalf("decode images: %v (raw=%s)", err, imagesJSON)
	}
	if len(images) != 2 {
		t.Fatalf("len(images) = %d, want 2", len(images))
	}
	if images[0]["cid"] != "bafkreiimg1" || images[0]["mime"] != "image/jpeg" || images[0]["alt"] != "first photo" {
		t.Errorf("images[0] = %v", images[0])
	}
	if images[1]["cid"] != "bafkreiimg2" || images[1]["mime"] != "image/png" || images[1]["alt"] != "second photo" {
		t.Errorf("images[1] = %v", images[1])
	}
}
```

- [ ] **Step 2:** Run.

```bash
just test -run TestCraftskyPost_Create_WithImages
```

Expected: pass (the implementation is already in place from Task 4.2; this test just exercises a different shape).

- [ ] **Step 3:** Commit.

```bash
git add appview/internal/index/craftsky_post_test.go
git commit -m "test(appview): add craftsky_posts images materialisation test"
```

---

## Chunk 5: Reply and quote pointers

Materialise the structural columns. After this chunk, replies and quote-posts populate their respective columns; non-replies/non-quotes leave them NULL.

### Task 5.1: Write the failing reply test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test.

```go
func TestCraftskyPost_Create_WithReply(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:r")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:r/social.craftsky.feed.post/reply",
		CID:        "bafyR",
		DID:        "did:plc:r",
		Rkey:       "reply",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "reply text",
			"createdAt": "` + fixedCreatedAt + `",
			"reply": {
				"root":   {"uri": "at://did:plc:author/social.craftsky.feed.post/root",   "cid": "bafyRoot"},
				"parent": {"uri": "at://did:plc:author/social.craftsky.feed.post/parent", "cid": "bafyParent"}
			}
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var rootURI, rootCID, parentURI, parentCID string
	if err := pool.QueryRow(context.Background(), `
		SELECT reply_root_uri, reply_root_cid, reply_parent_uri, reply_parent_cid
		FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&rootURI, &rootCID, &parentURI, &parentCID); err != nil {
		t.Fatalf("select: %v", err)
	}
	if rootURI != "at://did:plc:author/social.craftsky.feed.post/root" || rootCID != "bafyRoot" {
		t.Errorf("root = (%q, %q)", rootURI, rootCID)
	}
	if parentURI != "at://did:plc:author/social.craftsky.feed.post/parent" || parentCID != "bafyParent" {
		t.Errorf("parent = (%q, %q)", parentURI, parentCID)
	}
}
```

- [ ] **Step 2:** Run; expect failure.

```bash
just test -run TestCraftskyPost_Create_WithReply
```

Expected: failure — the four reply columns are still hardcoded to NULL.

### Task 5.2: Write the failing quote test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test.

```go
func TestCraftskyPost_Create_WithQuote(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:q")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:q/social.craftsky.feed.post/quote",
		CID:        "bafyQ",
		DID:        "did:plc:q",
		Rkey:       "quote",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "social.craftsky.feed.post",
			"text": "quoting another post",
			"createdAt": "` + fixedCreatedAt + `",
			"embed": {
				"$type": "social.craftsky.feed.post#quoteEmbed",
				"record": {"uri": "at://did:plc:other/social.craftsky.feed.post/orig", "cid": "bafyOrig"}
			}
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var quoteURI, quoteCID string
	if err := pool.QueryRow(context.Background(), `
		SELECT quote_uri, quote_cid FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&quoteURI, &quoteCID); err != nil {
		t.Fatalf("select: %v", err)
	}
	if quoteURI != "at://did:plc:other/social.craftsky.feed.post/orig" || quoteCID != "bafyOrig" {
		t.Errorf("quote = (%q, %q)", quoteURI, quoteCID)
	}
}
```

- [ ] **Step 2:** Run; expect failure.

```bash
just test -run TestCraftskyPost_Create_WithQuote
```

Expected: failure — `quote_*` columns are still NULL.

### Task 5.3: Implement reply + quote materialisation

**Files:**
- Modify: `appview/internal/index/craftsky_post.go`

- [ ] **Step 1:** Inside `handleUpsert`, just before the `INSERT` block, derive the four reply variables and the two quote variables. Add this block immediately after `tags := extractTags(rec.Facets)`:

```go
	var (
		replyRootURI, replyRootCID     any
		replyParentURI, replyParentCID any
	)
	if rec.Reply != nil {
		if rec.Reply.Root != nil {
			replyRootURI = rec.Reply.Root.Uri
			replyRootCID = rec.Reply.Root.Cid
		}
		if rec.Reply.Parent != nil {
			replyParentURI = rec.Reply.Parent.Uri
			replyParentCID = rec.Reply.Parent.Cid
		}
	}

	var quoteURI, quoteCID any
	if rec.Embed != nil && rec.Embed.FeedPost_QuoteEmbed != nil &&
		rec.Embed.FeedPost_QuoteEmbed.Record != nil {
		quoteURI = rec.Embed.FeedPost_QuoteEmbed.Record.Uri
		quoteCID = rec.Embed.FeedPost_QuoteEmbed.Record.Cid
	}
```

`any` is used here so a missing field passes `nil` to pgx, which writes SQL NULL. Strings would write empty strings, defeating the partial indexes' `WHERE ... IS NOT NULL` predicate.

- [ ] **Step 2:** Replace the four `nil, nil,` reply/quote placeholders in the `pool.Exec` argument list with the new variables:

```go
	_, err = c.pool.Exec(ctx, q,
		ev.URI, ev.DID, ev.Rkey, ev.CID,
		rec.Text,
		facetsJSON, imagesJSON,
		replyRootURI, replyRootCID,
		replyParentURI, replyParentCID,
		quoteURI, quoteCID,
		tags,
		ev.Record,
		createdAt,
	)
```

- [ ] **Step 3:** Format and run all CraftskyPost tests.

```bash
just fmt
just test -run TestCraftskyPost
```

Expected: all `TestCraftskyPost_*` tests pass, including the new `WithReply` and `WithQuote`.

- [ ] **Step 4:** Commit.

```bash
git add appview/internal/index/craftsky_post.go appview/internal/index/craftsky_post_test.go
git commit -m "feat(appview): materialise reply and quote pointers on craftsky_posts"
```

---

## Chunk 6: Update idempotency

Verify that:

- A redelivered event with the same `(URI, CID)` does not bump `indexed_at` — the replay-skip filter works.
- A new CID for an existing URI advances `indexed_at` and replaces the row's text/CID.

The implementation already handles both cases via `INSERT ... ON CONFLICT ... WHERE cid IS DISTINCT FROM EXCLUDED.cid`. This chunk just adds the tests that prove it.

### Task 6.1: Replay-preserves-indexed_at test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test.

```go
func TestCraftskyPost_Replay_PreservesIndexedAt(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:rp")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:rp/social.craftsky.feed.post/r",
		CID:        "bafyRP",
		DID:        "did:plc:rp",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"once","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}

	var firstIndexedAt string
	if err := pool.QueryRow(context.Background(),
		`SELECT indexed_at::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&firstIndexedAt); err != nil {
		t.Fatalf("select first indexed_at: %v", err)
	}

	// Replay identical event.
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}

	var secondIndexedAt string
	if err := pool.QueryRow(context.Background(),
		`SELECT indexed_at::text FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&secondIndexedAt); err != nil {
		t.Fatalf("select second indexed_at: %v", err)
	}

	if firstIndexedAt != secondIndexedAt {
		t.Errorf("indexed_at changed on replay: %q -> %q", firstIndexedAt, secondIndexedAt)
	}
}
```

- [ ] **Step 2:** Run. Expect pass — the `WHERE cid IS DISTINCT` filter already covers this.

```bash
just test -run TestCraftskyPost_Replay_PreservesIndexedAt
```

### Task 6.2: New-CID-replaces-row test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test.

```go
func TestCraftskyPost_Update_NewCID_ReplacesRow(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:u")
	idx := index.NewCraftskyPost(pool, testLogger())
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:u/social.craftsky.feed.post/r",
		CID:        "bafy1",
		DID:        "did:plc:u",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"original","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}

	var firstIndexedAt string
	if err := pool.QueryRow(ctx,
		`SELECT indexed_at::text FROM craftsky_posts WHERE uri = $1`, create.URI).
		Scan(&firstIndexedAt); err != nil {
		t.Fatalf("select first indexed_at: %v", err)
	}

	update := create
	update.CID = "bafy2"
	update.Action = "update"
	update.Record = json.RawMessage(`{"text":"edited","createdAt":"` + fixedCreatedAt + `"}`)
	if err := idx.Handle(ctx, update); err != nil {
		t.Fatal(err)
	}

	var (
		text, cid          string
		secondIndexedAt    string
	)
	if err := pool.QueryRow(ctx,
		`SELECT text, cid, indexed_at::text FROM craftsky_posts WHERE uri = $1`, create.URI).
		Scan(&text, &cid, &secondIndexedAt); err != nil {
		t.Fatalf("select after update: %v", err)
	}
	if text != "edited" {
		t.Errorf("text = %q, want edited", text)
	}
	if cid != "bafy2" {
		t.Errorf("cid = %q, want bafy2", cid)
	}
	if secondIndexedAt == firstIndexedAt {
		t.Errorf("indexed_at did not advance: %q stayed", firstIndexedAt)
	}
}
```

- [ ] **Step 2:** Run. Expect pass.

```bash
just test -run TestCraftskyPost_Update_NewCID_ReplacesRow
```

### Task 6.3: Update-before-create test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test. Models a Tap retry-ordering quirk where `update` arrives before `create`.

```go
func TestCraftskyPost_Update_BeforeCreate_TreatedAsCreate(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:ub")
	idx := index.NewCraftskyPost(pool, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:ub/social.craftsky.feed.post/r",
		CID:        "bafyUB",
		DID:        "did:plc:ub",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "update",
		Record:     json.RawMessage(`{"text":"hi","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM craftsky_posts WHERE uri = $1`, ev.URI).
		Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (update-before-create must insert)", count)
	}
}
```

- [ ] **Step 2:** Run. Expect pass.

```bash
just test -run TestCraftskyPost_Update_BeforeCreate
```

### Task 6.4: Commit idempotency tests

- [ ] **Step 1:** Stage and commit.

```bash
git add appview/internal/index/craftsky_post_test.go
git commit -m "test(appview): add craftsky_posts idempotency tests"
```

---

## Chunk 7: Delete and cascade

Replace the no-op `handleDelete` with a real `DELETE FROM craftsky_posts WHERE uri = $1`. Add a delete-of-non-existent test (must be a no-op) and a cascade test that exercises FK behaviour.

### Task 7.1: Failing direct-delete test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test.

```go
func TestCraftskyPost_Delete(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:d")
	idx := index.NewCraftskyPost(pool, testLogger())
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:d/social.craftsky.feed.post/r",
		CID:        "bafyD",
		DID:        "did:plc:d",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"to delete","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}

	del := tap.Event{
		URI: create.URI, DID: create.DID, Rkey: create.Rkey,
		Collection: "social.craftsky.feed.post",
		Action:     "delete",
	}
	if err := idx.Handle(ctx, del); err != nil {
		t.Fatalf("delete Handle: %v", err)
	}

	var count int
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM craftsky_posts WHERE uri = $1`, create.URI).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 after delete", count)
	}
}
```

- [ ] **Step 2:** Run; expect failure (current `handleDelete` is a no-op).

```bash
just test -run TestCraftskyPost_Delete$
```

Expected: failure — count is 1, not 0.

### Task 7.2: Failing delete-nonexistent test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test.

```go
func TestCraftskyPost_Delete_Nonexistent_NoOp(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	idx := index.NewCraftskyPost(pool, testLogger())

	del := tap.Event{
		URI:        "at://did:plc:none/social.craftsky.feed.post/r",
		DID:        "did:plc:none",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "delete",
	}
	if err := idx.Handle(context.Background(), del); err != nil {
		t.Errorf("delete-of-nonexistent should be no-op; got %v", err)
	}
}
```

- [ ] **Step 2:** Run. The current no-op stub already passes this — the assertion is that an empty DELETE doesn't error.

```bash
just test -run TestCraftskyPost_Delete_Nonexistent
```

Expected: pass even before the implementation lands.

### Task 7.3: Implement handleDelete

**Files:**
- Modify: `appview/internal/index/craftsky_post.go`

- [ ] **Step 1:** Replace the body of `handleDelete` with the real implementation.

```go
func (c *CraftskyPost) handleDelete(ctx context.Context, ev tap.Event) error {
	if _, err := c.pool.Exec(ctx,
		`DELETE FROM craftsky_posts WHERE uri = $1`, ev.URI); err != nil {
		return fmt.Errorf("delete %s: %w", ev.URI, err)
	}
	return nil
}
```

- [ ] **Step 2:** Format and run.

```bash
just fmt
just test -run TestCraftskyPost_Delete
```

Expected: both `TestCraftskyPost_Delete` and `TestCraftskyPost_Delete_Nonexistent_NoOp` pass.

### Task 7.4: Failing cascade test

**Files:**
- Modify: `appview/internal/index/craftsky_post_test.go`

- [ ] **Step 1:** Append the test. Exercises the FK cascade directly with a SQL DELETE on `craftsky_profiles` (not via `CraftskyProfile.Handle` — that would couple this test to another indexer's behaviour).

```go
func TestCraftskyPost_CascadeOnProfileDelete(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyPostsDDL)
	seedCraftskyMember(t, pool, "did:plc:cc")
	idx := index.NewCraftskyPost(pool, testLogger())
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:cc/social.craftsky.feed.post/r",
		CID:        "bafyCC",
		DID:        "did:plc:cc",
		Rkey:       "r",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
		Record:     json.RawMessage(`{"text":"will cascade","createdAt":"` + fixedCreatedAt + `"}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}

	// Delete the parent profile row directly. The FK cascade should fire.
	if _, err := pool.Exec(ctx,
		`DELETE FROM craftsky_profiles WHERE did = $1`, create.DID); err != nil {
		t.Fatalf("delete profile: %v", err)
	}

	var count int
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM craftsky_posts WHERE did = $1`, create.DID).Scan(&count)
	if count != 0 {
		t.Errorf("post count = %d after profile delete, want 0 (cascade missing?)", count)
	}
}
```

- [ ] **Step 2:** Run. Expect pass — the FK cascade is declared in the DDL.

```bash
just test -run TestCraftskyPost_CascadeOnProfileDelete
```

### Task 7.5: Commit delete + cascade

- [ ] **Step 1:** Stage and commit.

```bash
git add appview/internal/index/craftsky_post.go appview/internal/index/craftsky_post_test.go
git commit -m "feat(appview): handle craftsky_posts deletes with FK cascade"
```

### Task 7.6: Run the full suite

- [ ] **Step 1:** Confirm everything still passes.

```bash
just test
```

Expected: zero failures, including all pre-existing tests in the repo.

---

## Chunk 8: Wiring

Register the indexer with the dispatcher and tell Tap to forward `social.craftsky.feed.post` events. Then smoke-test by writing a real post on a dev PDS and seeing it land in the table.

### Task 8.1: Register CraftskyPost in deps.go

**Files:**
- Modify: `appview/internal/app/deps.go`

- [ ] **Step 1:** Add the registration call alongside the existing two. Find the block at [appview/internal/app/deps.go:118-124](../../../appview/internal/app/deps.go) that currently reads:

```go
	dispatcher := index.NewDispatcher(index.NotImplemented{})
	anonPDS := auth.NewAnonymousPDSClient(identityDir, 5*time.Second)
	blueskyIdx := index.NewBlueskyProfile(pool)
	backfiller := index.NewBlueskyBackfiller(anonPDS, blueskyIdx)
	dispatcher.Register("social.craftsky.actor.profile",
		index.NewCraftskyProfile(pool, backfiller, logger))
	dispatcher.Register("app.bsky.actor.profile", blueskyIdx)
```

- [ ] **Step 2:** Insert the new registration after the existing `social.craftsky.actor.profile` line (keeping the bsky one last). The end of the block becomes:

```go
	dispatcher := index.NewDispatcher(index.NotImplemented{})
	anonPDS := auth.NewAnonymousPDSClient(identityDir, 5*time.Second)
	blueskyIdx := index.NewBlueskyProfile(pool)
	backfiller := index.NewBlueskyBackfiller(anonPDS, blueskyIdx)
	dispatcher.Register("social.craftsky.actor.profile",
		index.NewCraftskyProfile(pool, backfiller, logger))
	dispatcher.Register("social.craftsky.feed.post",
		index.NewCraftskyPost(pool, logger))
	dispatcher.Register("app.bsky.actor.profile", blueskyIdx)
```

- [ ] **Step 3:** Format and build.

```bash
just fmt
cd appview && go build ./...
```

Expected: clean build.

### Task 8.2: Extend TAP_COLLECTION_FILTERS

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step 1:** Find the line (verified in Task 1.2):

```yaml
      TAP_COLLECTION_FILTERS: "social.craftsky.actor.profile,app.bsky.actor.profile"
```

- [ ] **Step 2:** Replace with the extended value:

```yaml
      TAP_COLLECTION_FILTERS: "social.craftsky.actor.profile,social.craftsky.feed.post,app.bsky.actor.profile"
```

The order doesn't matter functionally — Tap treats it as a set — but keeping `social.craftsky.*` together and `app.bsky.*` last is a small readability win.

### Task 8.3: Restart compose and run the full suite

**Files:**
- (none — operational steps)

- [ ] **Step 1:** Recreate the Tap container so it picks up the new filter list.

```bash
docker compose up -d --build tap appview
```

Expected: both services come up healthy. `docker compose ps tap` shows `(healthy)` within ~30 seconds.

- [ ] **Step 2:** Confirm the appview log shows the new dispatcher registration. The dispatcher's `slog.Debug` call fires on each event but the registration itself is not logged; the proxy is that `cli tap status` still shows the consumer connected.

```bash
just tap-status
```

Expected: `connected: true`, no `last_error`.

- [ ] **Step 3:** Run the full Go test suite once more against the updated stack.

```bash
just test
```

Expected: all green.

### Task 8.4: Smoke test (optional but recommended)

If you have a dev PDS account that's already onboarded to Craftsky in this stack, write a real post and watch it land.

**Files:**
- (none — operational steps)

- [ ] **Step 1:** Confirm a Craftsky member exists.

```bash
just psql -c 'SELECT did FROM craftsky_profiles LIMIT 5;'
```

Expected: at least one DID. If the table is empty, run the dev login flow first (see [docs/superpowers/specs/2026-04-23-profile-onboarding-design.md](../specs/2026-04-23-profile-onboarding-design.md) for how onboarding writes the row), then continue.

- [ ] **Step 2:** From the Flutter app or a `curl`-based PDS write, publish a `social.craftsky.feed.post` record under that DID with a plain text body (use any onboarded dev account). Wait ~5 seconds for the firehose round-trip.

- [ ] **Step 3:** Confirm the row appeared.

```bash
just psql -c "SELECT uri, text, created_at, indexed_at FROM craftsky_posts ORDER BY indexed_at DESC LIMIT 5;"
```

Expected: the new row at the top, `text` matching what you wrote, `created_at` close to your client clock, `indexed_at` matching server clock.

### Task 8.5: Commit the wiring

- [ ] **Step 1:** Stage and commit.

```bash
git add appview/internal/app/deps.go docker-compose.yml
git commit -m "feat(appview): wire CraftskyPost indexer and Tap filter"
```

---

## Done

After Chunk 8 the appview indexes `social.craftsky.feed.post` events end to end:

- Tap forwards them; the dispatcher routes by NSID; `CraftskyPost` decodes, gates on membership, materialises text + facets/images/reply/quote/tags, stores the raw record in `JSONB`, and is idempotent on `(URI, CID)`.
- The cascade FK guarantees a user leaving Craftsky removes their posts atomically.
- `craftsky_posts` is queryable by `(indexed_at DESC)`, `(did, indexed_at DESC)`, hashtag (`tags @> ARRAY[...]`), and structural pointers (`reply_parent_uri`, `reply_root_uri`, `quote_uri`).

Next specs in the feed sequence:

1. The `/v1/feed/*` read endpoint that consumes this table.
2. Project field materialisation (extends the schema and the indexer; the `record JSONB` column makes this a SQL-only migration).
3. Like/repost indexers (separate NSIDs, separate tables).
4. Flutter feed UI.

None are blockers for this plan; it stands on its own.
