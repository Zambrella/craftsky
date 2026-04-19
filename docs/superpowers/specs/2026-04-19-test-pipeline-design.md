# Test Pipeline: End-to-End Validation via `social.craftsky.test.post`

**Date:** 2026-04-19
**Status:** Approved design, ready for implementation planning
**Author:** brainstormed with Claude

## Summary

Build a disposable, end-to-end validation of the appview's async pipeline: firehose event → Tap consumer → indexer → Postgres → HTTP read API. All of this is wrapped around a throwaway lexicon (`social.craftsky.test.post`) and a throwaway Postgres table (`test_posts`), so we can prove the pipeline works without committing to the shape of the real `social.craftsky.feed.post` record — which we are not yet confident enough to index.

The work ships as a single vertical slice. The one structural piece that survives is the collection-based `Dispatcher` in `internal/index/`; everything else (lexicon, migration, indexer, handler, route) is quarantined in `internal/testpipeline/` and deleted when the real post indexer lands.

## Motivation

OAuth v1 is merged. The next bottleneck is that `internal/index.NotImplemented` errors on every Tap event, so nothing the firehose delivers lands in Postgres, and therefore no read API can return anything meaningful. Before building the real social graph (posts, follows, likes, blocks), we need confidence that the pipeline itself works: Tap's at-least-once delivery, the indexer contract, idempotent upserts, and a read endpoint that serves the indexed data.

The real `social.craftsky.feed.post` lexicon already exists and is fairly rich (text, facets, project details, images, quote embeds, reply refs). Committing to that shape by indexing it now is premature — we don't yet know whether the field set is right, and once records are written to real PDSes under a given NSID, the lexicon is effectively frozen (per AGENTS.md rule #4). A deliberately disposable test NSID sidesteps this while still validating everything mechanical.

## Non-goals

- **Not** designing the real post indexer. That's a separate slice with its own ADR.
- **Not** a BFF write path. Seeding test records is done externally (e.g. via `atcli`).
- **Not** exercising blob handling. No images on the test record.
- **Not** auth, pagination with cursors, or any production-grade read ergonomics on `/test/feed`.
- **Not** migrating or removing the existing `bluesky_posts_sample` table; it's already marked for deletion alongside the first real indexer.

## Design

### Lexicon

New file: `lexicon/social/craftsky/test/post.json`.

NSID `social.craftsky.test.post` — a dedicated `test` namespace (not `feed.testPost`) so anything under `social.craftsky.test.*` is obviously quarantined.

Fields:
- `text` — string, `maxLength: 3000`, `maxGraphemes: 300` (matches real post lexicon).
- `createdAt` — datetime, required.

The `description` on `main` explicitly marks the record as disposable and points at `internal/testpipeline/` for context. No facets, no images, no embed, no reply, no project metadata.

No ADR is written. The rationale for the NSID's existence lives in this design doc; the record is not intended to appear on production PDSes.

### Database

New migration `appview/migrations/000004_test_posts.up.sql` (and matching `.down.sql`):

```sql
-- DELETE ME: part of the disposable test pipeline. Drop this migration
-- (and add a drop-table migration) when the real social.craftsky.feed.post
-- indexer lands. See internal/testpipeline/ for context.
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

- `uri` is the primary key — atproto's natural record identifier, and it lets us upsert idempotently on replay.
- `cid` is stored but not part of the key. On record update the URI stays the same and the CID changes; the upsert replaces the row.
- `did` is denormalized into the row so `/test/feed` filter/query is a single-table read (no URI parsing).
- `created_at` is indexed DESC — the only query shape we serve is reverse-chronological.
- `indexed_at` is included for pipeline-lag debugging. Cheap, useful.
- No foreign keys — the table is a leaf and `DROP TABLE test_posts` is a clean removal.

The existing `bluesky_posts_sample` table stays untouched. This migration does not bundle unrelated cleanup.

### Dispatcher (survivable)

New file: `appview/internal/index/dispatcher.go`.

A `Dispatcher` implementing `index.Indexer`, routing events by collection NSID to registered handlers.

```go
type Dispatcher struct {
    handlers map[string]Indexer
    fallback Indexer
}

func NewDispatcher(fallback Indexer) *Dispatcher
func (d *Dispatcher) Register(collection string, idx Indexer)
func (d *Dispatcher) Handle(ctx context.Context, ev tap.Event) error
```

`Handle` looks up `ev.Collection` in `handlers`; on miss, delegates to `fallback`. Downstream errors propagate so Tap skips the ack and redelivers (the existing poison-pill guard in `WSConsumer` eventually drops truly bad events).

Wiring, in `deps.go` (or the existing appview bootstrap):

```go
dispatcher := index.NewDispatcher(index.NotImplemented{})
dispatcher.Register("social.craftsky.test.post", testpipeline.NewIndexer(db))
consumer := tap.NewWSConsumer(..., dispatcher)
```

`NotImplemented` remains the fallback. Unknown collections continue to produce errors that Tap retries; this preserves current behavior for every non-test collection. Replacing the fallback with a silent-drop is a tuning decision for later.

This dispatcher is the single piece of structural code from this slice that is *not* disposable — real indexers plug into the same `Register` call when they land.

### Testpipeline package (disposable)

All throwaway code is quarantined in a dedicated package so cleanup is `rm -rf internal/testpipeline/` plus migration/lexicon/wiring removal:

```
appview/internal/testpipeline/
├── doc.go           // Package-level DELETE ME notice + rationale
├── indexer.go       // Implements index.Indexer
├── indexer_test.go
├── handler.go       // GET /test/feed
├── handler_test.go
└── queries.sql      // sqlc input
```

`doc.go` states that the entire package is disposable, references the design doc, and lists the sibling files/migrations that get deleted together.

`indexer.go` implements `index.Indexer`:
- On create/update: parse `ev.Record` as a test post; `INSERT ... ON CONFLICT (uri) DO UPDATE` setting cid/text/created_at and refreshing indexed_at.
- On delete: `DELETE FROM test_posts WHERE uri = $1`.
- Malformed record payload: return error (Tap retries; poison-pill eventually drops).
- Idempotent on `(uri, cid)` as the `Indexer` interface requires.

Deletes are handled from day one — it's a single SQL statement and leaving deleted records visible in `/test/feed` would be incorrect semantics.

### HTTP endpoint

`GET /test/feed`, registered in `appview/internal/routes/`, handler in `internal/testpipeline/handler.go`.

- **Auth:** none. This is a diagnostic — hittable with `curl` from anywhere.
- **Query params:** `limit` — integer, default 50, clamped to max 200. Invalid values → 400.
- **No cursor/pagination.** If you need older posts, raise `limit`. A real pagination story belongs on the real feed.

Response envelope:

```json
{
  "posts": [
    {
      "uri": "at://did:plc:abc123/social.craftsky.test.post/3kx...",
      "cid": "bafyrei...",
      "did": "did:plc:abc123",
      "text": "hello from the pipeline",
      "createdAt": "2026-04-19T12:34:56Z",
      "indexedAt": "2026-04-19T12:34:58Z"
    }
  ]
}
```

Reverse-chronological by `created_at`. Envelope object rather than bare array for cheap future extension. Empty result → 200 with `{"posts": []}`, not 404. DB error → 500, logged via `slog`. Content-type `application/json; charset=utf-8`.

SQL:

```sql
SELECT uri, cid, did, text, created_at, indexed_at
FROM test_posts
ORDER BY created_at DESC
LIMIT $1;
```

Uses `test_posts_created_at_idx`.

### Seeding / write path

Out of scope for this slice. Test records are written to a PDS externally (e.g. `atcli update social.craftsky.test.post ...`) during local testing. The BFF write path is its own focused slice, built against the real `feed.post` lexicon, not this one.

The loop this slice validates is therefore: **external PDS write → firehose → Tap → dispatcher → testpipeline indexer → Postgres → `/test/feed`**. The first half of "client → appview → PDS" proves itself in the subsequent BFF-write slice.

## Testing

### Unit tests

`internal/testpipeline/indexer_test.go` against the compose Postgres (per the project convention — no sqlmock, no in-memory substitutes):

- Create event → row exists with correct fields.
- Update event (same URI, new CID, new text) → row replaced, not duplicated.
- Delete event → row gone.
- Duplicate create event → one row, no error (idempotency).
- Malformed record payload → returns error.

`internal/testpipeline/handler_test.go`:

- Empty table → 200, `{"posts": []}`.
- Multiple rows → reverse-chronological order.
- `limit` param respected.
- `limit` clamped to 200.
- Invalid `limit` → 400.

`internal/index/dispatcher_test.go`:

- Registered collection → routed to correct handler.
- Unregistered collection → routed to fallback.
- Downstream error → propagated (so Tap skips the ack).

### Integration test

One end-to-end test that justifies the slice:

- Construct a synthetic `tap.Event` for `social.craftsky.test.post` (create op).
- Feed it through the real `Dispatcher` → `testpipeline.Indexer` → Postgres.
- Hit `GET /test/feed` via `httptest`.
- Assert the record comes back with correct fields, correct order.

If this test fails, the pipeline is broken.

### Out of scope for testing

- Real Tap WebSocket connection — covered by existing `tap` package tests.
- Firehose → Tap delivery — Tap sidecar's responsibility.
- OAuth session middleware — `/test/feed` is unauthenticated.

## Cleanup plan

When the real `social.craftsky.feed.post` indexer lands:

1. `rm -rf appview/internal/testpipeline/`.
2. Add migration `00000N_drop_test_posts.up.sql` with `DROP TABLE test_posts;`.
3. `rm -rf lexicon/social/craftsky/test/`.
4. Remove the `/test/feed` route registration.
5. Remove the `dispatcher.Register("social.craftsky.test.post", ...)` line.

The dispatcher itself stays.

## Risks and open questions

- **Dispatcher fallback noise:** `NotImplemented` errors on every non-test event once the firehose is actually carrying traffic. Acceptable for now because Tap's poison-pill guard caps the damage, but if the log noise becomes a problem before real indexers land, swap the fallback for a silent-drop. Tuning decision, not a blocker.
- **Lexicon discoverability:** anyone reading `lexicon/` will see `social.craftsky.test.post` next to the real NSIDs. The `test` namespace + the `DISPOSABLE` description on the record should be enough signal, but if it isn't, a README in `lexicon/social/craftsky/test/` is a cheap follow-up.
- **Tap event shape:** this design assumes `tap.Event` exposes `Collection`, `Op` (create/update/delete), `URI`, `CID`, and a raw `Record` payload. If any of those are missing or named differently, the dispatcher and indexer adapt; no design changes needed.
