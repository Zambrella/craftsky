# Feed Post Indexing Design

- **Status:** Draft
- **Date:** 2026-05-04
- **Related:**
  - [post-lexicon-fields](2026-04-23-post-lexicon-fields-design.md) — locks the lexicon shape and names the fields the indexer will materialise (project fields are deferred to a later spec).
  - [appview-server-scaffold](2026-04-16-appview-server-scaffold-design.md) — the dispatcher/indexer pattern this spec extends.
  - [profile-onboarding](2026-04-23-profile-onboarding-design.md) — establishes the membership table (`craftsky_profiles`) the post indexer gates on.
  - [tap-integration](2026-04-17-tap-integration-design.md) — the firehose-via-Tap consumer that delivers events to the dispatcher.

## Summary

Index `social.craftsky.feed.post` records into a new `craftsky_posts` Postgres table so the AppView has a queryable feed substrate. This is the storage half of the feed feature; the read endpoint and Flutter client are separate, sequenced specs.

This pass materialises a deliberately narrow slice: post identity (URI, DID, rkey, CID), `text`, `createdAt`, the structural relations needed for thread/quote queries (`reply_*`, `quote_*`), the rendering-relevant non-project fields (`facets`, `images`), and a hashtag-search column (`tags`) extracted from facet tag features. The full record is preserved as `JSONB` so future passes can materialise project fields, image dimensions, etc. without re-fetching from PDSes.

Like and repost indexing, project-field materialisation, and any read endpoint are explicitly out of scope.

## Goals

1. **Get the full Craftsky feed flow working end to end as fast as possible** — indexer → DB → endpoint → client. This spec is the first step. To minimise scope creep, it does the smallest amount of indexing that lets a chronological "all Craftsky posts" feed exist; everything beyond rendering a plain text post is deferred.
2. **Match the existing indexer pattern** in [appview/internal/index/](../../../appview/internal/index/) so a contributor reading `craftsky_post.go` next to `craftsky_profile.go` and `bluesky_profile.go` sees the same shape: `Indexer` interface, dispatcher registration in [appview/internal/app/deps.go](../../../appview/internal/app/deps.go), idempotent on `(URI, CID)`, raw SQL via `pgxpool`.
3. **Keep the schema forward-compatible.** Storing the full record as `JSONB` means the next two materialisation passes (project fields; image dimensions / blob URLs) are pure SQL migrations against existing data, not federated re-fetches from PDSes.
4. **Honour the indexer commitments in [post-lexicon-fields §AppView indexer commitments](2026-04-23-post-lexicon-fields-design.md#appview-indexer-commitments)** insofar as they overlap with this pass: hashtag tags are unified from facet `#tag` features into `tags TEXT[]` (with project tag union deferred to the project-fields pass).

## Non-goals

- **Like and repost indexing.** Separate NSIDs (`social.craftsky.feed.like`, `social.craftsky.feed.repost`) with separate indexers and tables. They will land in their own spec; counter columns on `craftsky_posts` are not added pre-emptively.
- **Project field materialisation.** `is_project`, `craftType`, `status`, `materials`, `pattern.*`, sewing `projectType` — all named in [post-lexicon-fields §AppView indexer commitments](2026-04-23-post-lexicon-fields-design.md#appview-indexer-commitments) — are deferred. Project payloads are still preserved in `record JSONB`; a later migration adds the columns and backfills via SQL.
- **Read endpoint.** No `/v1/feed/*` route. The downstream consumer of `craftsky_posts` is a separate spec.
- **Hashtag merging from `project.common.tags`.** This pass populates `tags` from facets only. The belt-and-suspenders union with `project.common.tags` is added when project-field materialisation lands.
- **Cross-post project identity** (the spec it cuts across is already explicit non-goal in [post-lexicon-fields](2026-04-23-post-lexicon-fields-design.md)).
- **Backfill for posts that arrived before their author's `craftsky_profiles` row.** Posts arriving before the profile are dropped silently; a future post-backfiller (mirroring `BlueskyBackfiller`) is the right vehicle for catch-up if it becomes a real problem. Mentioned in [Risks](#risks).
- **Image blob fetching, transcoding, or CDN proxying.** Images are referenced by CID; clients fetch from the originating PDS. AppView image proxying is its own spec.
- **Reply/quote target validation.** A post may reply to or quote a record we haven't indexed (Bluesky post, deleted post, post from non-member). The indexer stores the URI/CID verbatim and lets the read endpoint handle "target not indexed" rendering.

## Context

### What's already in place

- The lexicon `social.craftsky.feed.post` is locked at [lexicon/social/craftsky/feed/post.json](../../../lexicon/social/craftsky/feed/post.json). Generated Go types live at [appview/internal/lexicon/craftsky/feedpost.go](../../../appview/internal/lexicon/craftsky/feedpost.go).
- The dispatcher routes Tap events to per-NSID indexers; one is registered per NSID in [appview/internal/app/deps.go](../../../appview/internal/app/deps.go). `Handle` must be idempotent on `(URI, CID)` because Tap delivers at least once.
- `craftsky_profiles` is the membership table written by `CraftskyProfile` ([appview/internal/index/craftsky_profile.go](../../../appview/internal/index/craftsky_profile.go)). A user "is on Craftsky" iff they have a row in this table.
- Tap is configured to forward only the collections we have indexers for — `TAP_COLLECTION_FILTERS` in [docker-compose.yml:53](../../../docker-compose.yml). Adding a new indexer means adding to that filter list, otherwise events never reach the appview.

### Membership gating

Posts are gated on `craftsky_profiles` (decided during brainstorming). The indexer drops events from non-members silently — same pattern as `BlueskyProfile`. A post arriving before its author's `craftsky_profiles` row is dropped permanently in this pass; if real users hit this, it gets fixed by a post-backfiller later (mirroring `BlueskyBackfiller` for Bluesky profiles).

This decision motivates the `did REFERENCES craftsky_profiles(did) ON DELETE CASCADE` foreign key: when a user leaves Craftsky (deletes their `social.craftsky.actor.profile`), their posts evaporate alongside their `bluesky_profiles` mirror. The cascade fires automatically; `CraftskyProfile.handleDelete` does not need to be modified.

### Feed ordering

Server-side `indexed_at` (the `now()` snapshot when the row is inserted/updated) is the canonical chronological-feed ordering column. `created_at` is preserved from the record but is for display only — not the order key. Decided during brainstorming; matches Bluesky's protection against backdating-attacks. Trade-off: a legitimately-delayed post (PDS retention catch-up, Tap reconnect) appears at the top rather than at its declared time. Acceptable for v1; a future backfiller can clamp via `LEAST(record.createdAt, now())` if the delay-jumps-the-queue behaviour becomes a real complaint.

### Idempotency

Tap delivers events at least once. The indexer must converge on a single row regardless of replay or out-of-order delivery within an `(URI, CID)` pair:

- Replay of the same `(URI, CID)`: no-op. The `ON CONFLICT (uri) DO UPDATE SET ... WHERE cid IS DISTINCT` filter elides the update; `indexed_at` is not re-stamped.
- New CID for an existing URI: row updated, `indexed_at` advanced. atproto `put` produces this; we treat it as semantically equivalent to a `create` action with the new payload.
- `update` action arriving before `create`: handled by `INSERT ... ON CONFLICT` regardless of action label.
- `delete` of a non-existent URI: silent no-op. Tap may redeliver deletes after retention boundaries; an error here would poison-pill the consumer.

## Design

### Schema

New migration `appview/migrations/000010_craftsky_posts.up.sql`:

```sql
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

Down migration drops the indexes and the table.

#### Column rationale

| Column | Source | Notes |
|---|---|---|
| `uri` | `tap.Event.URI` | Canonical atproto identifier; PK. `at://did/social.craftsky.feed.post/rkey`. |
| `did` | `tap.Event.DID` | Author. FK with `ON DELETE CASCADE` enforces membership gate at storage layer. |
| `rkey` | `tap.Event.Rkey` | Redundant with URI but useful for diagnostics. `UNIQUE (did, rkey)` is a sanity check. |
| `cid` | `tap.Event.CID` | Record CID. Drives the `IS DISTINCT FROM` skip-on-replay filter on upsert. |
| `text` | `record.text` | Materialised. Required by the lexicon; NOT NULL is safe. |
| `facets` | `record.facets` | Stored verbatim as `JSONB` array. NULL when absent or empty (avoids storing `[]`). |
| `images` | `record.images`, transformed | Normalised to `[{cid, mime, alt}, ...]` so clients don't need to know the atproto blob ref shape (`{$type, ref: {$link}, mimeType, size}`). NULL when absent or empty. Source of truth stays in `record`. |
| `reply_root_uri`, `reply_root_cid` | `record.reply.root.{uri,cid}` | NULL when not a reply. |
| `reply_parent_uri`, `reply_parent_cid` | `record.reply.parent.{uri,cid}` | NULL when not a reply. |
| `quote_uri`, `quote_cid` | `record.embed.quoteEmbed.record.{uri,cid}` | NULL when not a quote. Open-union `embed` may grow other variants; only `quoteEmbed` is materialised this pass. |
| `tags` | extracted from `facets` | Lowercased, trimmed, deduped. See [Tag extraction](#tag-extraction). `NOT NULL DEFAULT '{}'` so client code never sees a NULL tag set. |
| `record` | `tap.Event.Record` | Verbatim record as `JSONB`. Forward-compat hook for project-field materialisation, image dimensions, etc. — see [The `record` column](#the-record-column). |
| `created_at` | `record.createdAt`, parsed | Display only. Not indexed; the read endpoint may surface it but feeds order by `indexed_at`. |
| `indexed_at` | `now()` on insert/update | Server-side, monotonic, the chronological-feed ordering key. |

#### The `record` column

Storing the full record as `JSONB` is the cheap escape hatch for forward-compat:

1. **Migration backfill of materialised fields.** When the project-field-materialisation spec lands, `ALTER TABLE craftsky_posts ADD COLUMN craft_type TEXT; UPDATE craftsky_posts SET craft_type = record->'project'->'common'->>'craftType'` is a single Postgres pass — no per-record federation calls.
2. **Surface area we don't lose between materialisation passes.** Even before facet-merge, image dimensions, or project fields are materialised, the data is preserved.
3. **Debugging.** Spot-check a malformed-but-spec-valid record without going back to the PDS.

Cost is negligible: the lexicon caps `text` at 20 KB; a fat project post is ~25 KB worst case, a plain text post ~200 bytes. Postgres stores `JSONB` in a parsed binary form so subsequent reads and `->`/`->>`/`@>` operations are cheap. The alternative — re-fetching records from PDSes when we want to materialise more fields — is operationally fragile (PDS uptime, rate limiting, blob-vs-record routing) and was an explicit lesson from Bluesky's appview implementation.

#### Index rationale

- `craftsky_posts_indexed_at_desc` — global chronological feed (every craft feed and the future follow feed both sit on top of this).
- `craftsky_posts_did_indexed_at_desc` — "X's posts" (profile page, follow-feed-by-DID-set).
- Partial indexes on `reply_parent_uri`, `reply_root_uri`, `quote_uri` — most posts aren't replies or quotes, so partial indexes stay small. Used by future thread-fetch and quote-resolution queries.
- GIN on `tags` — `tags @> ARRAY['fair-isle']` style hashtag queries.

### Indexer

New file `appview/internal/index/craftsky_post.go`. Mirrors `craftsky_profile.go` and `bluesky_profile.go`:

```go
package index

type CraftskyPost struct {
    pool   *pgxpool.Pool
    logger *slog.Logger
}

const craftskyPostNSID syntax.NSID = "social.craftsky.feed.post"

func NewCraftskyPost(pool *pgxpool.Pool, logger *slog.Logger) *CraftskyPost { ... }

func (c *CraftskyPost) Handle(ctx context.Context, ev tap.Event) error {
    if ev.Collection != craftskyPostNSID { return nil }
    switch ev.Action {
    case "create", "update":
        return c.handleUpsert(ctx, ev)
    case "delete":
        return c.handleDelete(ctx, ev)
    default:
        return fmt.Errorf("unknown action %q on %s", ev.Action, ev.URI)
    }
}
```

#### Upsert path

1. **Membership check.** `SELECT EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1)`. False → return `nil` (silent drop, matches `BlueskyProfile`). Permanent for this pass; see [Risks](#risks).
2. **Decode.** `json.Unmarshal(ev.Record, &rec)` into `craftskylex.FeedPost` from the generated package.
3. **Parse `created_at`.** ISO-8601 string from the record into `time.Time`. Parse failure → return error (Tap retries; the `MaxRetries` poison-pill cap on the consumer means we don't loop forever on a malformed timestamp).
4. **Derive materialised columns.**
   - `text` directly from `rec.Text`.
   - `facets` — JSONB-marshal `rec.Facets` if non-empty, else SQL NULL.
   - `images` — flatten each `*FeedPost_Image` to `{cid: img.Image.Ref.String(), mime: img.Image.MimeType, alt: img.Alt}` (the `Ref` is a `lexutil.LexLink`, which is `cid.Cid` under the hood — `.String()` produces the canonical CID string). JSONB-marshal the array if non-empty, else SQL NULL.
   - `reply_*` — if `rec.Reply != nil`, pull `rec.Reply.Root.Uri`, `rec.Reply.Root.Cid`, `rec.Reply.Parent.Uri`, `rec.Reply.Parent.Cid`. Else four NULLs.
   - `quote_*` — if `rec.Embed != nil && rec.Embed.FeedPost_QuoteEmbed != nil`, pull `.Record.Uri`, `.Record.Cid`. Else two NULLs.
   - `tags` — see [Tag extraction](#tag-extraction).
5. **Upsert.** Single statement:

   ```sql
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
   ```

   The `WHERE ... IS DISTINCT FROM` clause is the replay filter: a redelivered event with the same CID matches the row but the WHERE predicate is false, so no rows are written and `indexed_at` is preserved.

#### Tag extraction

Walk `rec.Facets`. For each facet, walk its `Features` union. For each feature whose `$type` is `app.bsky.richtext.facet#tag`, take the `Tag` field. Apply:

1. Trim whitespace.
2. Lowercase.
3. Drop empty strings.
4. Deduplicate (preserve first-seen order is fine; downstream consumers don't depend on order).

Result is the value for `tags`. Empty array (not NULL) when no tag features are present — the column is `NOT NULL DEFAULT '{}'`.

This pass deliberately does **not** enforce the kebab-case pattern from [post-lexicon-fields §Tags design rationale](2026-04-23-post-lexicon-fields-design.md#tags-design-rationale): facet-derived tags can't always satisfy the pattern (inline `#hashtag` facets can't carry hyphens), and the spec explicitly accepts that inline tags are uglier than composer-merged ones. Rejecting non-kebab-case facet tags would silently drop valid hashtags.

#### Delete path

Single statement, no transaction:

```sql
DELETE FROM craftsky_posts WHERE uri = $1
```

Returns the count of affected rows in the result; we ignore it (delete-of-non-existent must be a no-op). The FK cascade only runs the other direction (profile delete → post delete); a post delete event does not cascade anywhere.

### Wiring

Three changes to existing code:

1. **Dispatcher registration.** [appview/internal/app/deps.go:118-124](../../../appview/internal/app/deps.go) gains:

   ```go
   dispatcher.Register("social.craftsky.feed.post",
       index.NewCraftskyPost(pool, logger))
   ```

   Placed alongside the existing `social.craftsky.actor.profile` and `app.bsky.actor.profile` registrations.

2. **Tap collection filter.** [docker-compose.yml:53](../../../docker-compose.yml) — extend `TAP_COLLECTION_FILTERS` to include the post NSID:

   ```yaml
   TAP_COLLECTION_FILTERS: "social.craftsky.actor.profile,social.craftsky.feed.post,app.bsky.actor.profile"
   ```

   The comment block immediately above the env var already explains the contract: every NSID listed here must have a registered indexer on the appview side, otherwise events hit `index.NotImplemented{}` and Tap retries forever.

3. **Migration file.** New `appview/migrations/000010_craftsky_posts.up.sql` and `.down.sql` per [Schema](#schema).

No changes to `Tap`, the dispatcher, the consumer, or any auth/OAuth code.

### Tests

Mirror the structure of [appview/internal/index/craftsky_profile_test.go](../../../appview/internal/index/craftsky_profile_test.go): `package index_test`, inline DDL string, `testdb.WithSchema(t, ddl)` per test, table-driven where useful but separate functions for readability.

DDL fixture in the test file declares both `craftsky_profiles` (parent) and `craftsky_posts` (child). The `bluesky_profiles` table is **not** needed in this fixture — the post indexer doesn't touch it.

Cases:

- `Create` — author exists, plain text post → row inserted with expected text, NULL facets/images/reply/quote, empty `tags`, `record` round-trips, `created_at` parsed correctly.
- `Create_NoAuthor_DroppedSilently` — author missing from `craftsky_profiles` → no row inserted, no error returned.
- `Create_WithFacets_TagsExtracted` — facets containing `app.bsky.richtext.facet#tag` features → `tags` column populated with lowercased, deduped values.
- `Create_WithImages` — record with images → `images` JSONB has flattened `[{cid, mime, alt}, ...]` shape.
- `Create_WithReply` — record with `reply.root` and `reply.parent` → all four `reply_*` columns populated.
- `Create_WithQuote` — record with `embed.quoteEmbed` → `quote_uri` and `quote_cid` populated.
- `Create_WithProjectPayload_StoredInRecordOnly` — record with a populated `project` field → no project columns exist yet, but `record->'project'` round-trips intact.
- `Update_NewCID_RowReplaced` — same URI, new CID, new text → row updated, `indexed_at` advanced past pre-update value.
- `Update_SameCID_NoOp` — same URI, same CID → row unchanged, `indexed_at` not advanced (replay idempotency).
- `Update_BeforeCreate_TreatedAsCreate` — `update` action with no existing row → row inserted (Tap retry-ordering safety).
- `Delete` — existing URI → row removed.
- `Delete_Nonexistent_NoOp` — URI not present → no error, no rows changed.
- `MalformedCreatedAt_Errors` — record with unparseable `createdAt` → `Handle` returns an error (Tap retries within `MaxRetries`).
- `CascadeOnProfileDelete` — insert profile, insert post, delete profile → post row removed by FK cascade.
- `MultipleTagFacets_Deduped` — facets with overlapping or whitespace-padded tag features → deduped, lowercased, trimmed.

Coverage target: every branch in `handleUpsert` and `handleDelete`, plus the tag-extraction helper.

## Alternatives considered

### Single materialised JSONB blob instead of typed columns

Considered keeping the schema deliberately thin — `(uri, did, cid, record, indexed_at)` only — and querying everything via `record->...` operators with a GIN index on `record` itself.

**Rejected** because:

- Hashtag queries (`record @> '{"facets": [{"features": [{"tag": "fair-isle"}]}]}'`) are awkward to express and slow even with GIN; a `tags TEXT[]` column with its own GIN is dramatically simpler.
- Reply-thread queries (`reply_parent_uri = $1`) and quote-resolution queries (`quote_uri = $1`) need plain B-tree indexes on scalar columns, not JSON path indexes.
- The schema-as-documentation argument: a contributor reading the table's `\d` output sees the post's structure at a glance; with a JSONB-only schema they must read the lexicon plus the indexer code.

The compromise — typed columns for the *queryable* fields, `record JSONB` for everything else — captures the operational benefits of materialisation without the cost of materialising every field eagerly.

### Cascade vs. explicit delete in `CraftskyProfile.handleDelete`

Considered handling profile→post cascade by extending `CraftskyProfile.handleDelete` with an explicit `DELETE FROM craftsky_posts WHERE did = $1` inside the transaction (mirroring how it handles `bluesky_profiles`).

**Rejected** in favour of the FK-level `ON DELETE CASCADE`. The FK runs in the same Postgres transaction as the parent delete, so atomicity is preserved; declaratively wired cascade can't be forgotten in a future indexer refactor; and the semantic relationship — "a post belongs to a Craftsky profile" — is correctly modelled at the storage layer rather than smuggled into application code.

`bluesky_profiles` is not converted to a FK cascade in this spec because that's a separate change with its own test surface, and the current explicit-DELETE pattern works fine.

### Buffering posts that arrive before the profile

Considered queuing posts with no matching `craftsky_profiles` row in a `pending_posts` side table, replayed when the profile arrives.

**Rejected** for v1: the table is state we don't need yet, the replay path is its own indexer subtlety, and the realistic failure mode (a Craftsky-app user creates a profile then a post in the same session) almost certainly delivers the events in causal order from the same PDS. If real users hit this, a post-backfiller (mirroring `BlueskyBackfiller`) is the right fix — it pulls from the PDS authoritatively rather than relying on Tap's at-least-once buffer.

### `created_at`-clamped ordering (`LEAST(record.createdAt, now())`)

Considered using the lesser of the record's `createdAt` and the current time as both the storage `created_at` and the feed-ordering key.

**Rejected** for v1 in favour of pure `indexed_at`. Pure server time is simpler to reason about, harder to game, and the only case where clamping beats it (legitimate backfill of historical posts during initial onboarding) doesn't apply yet because there is no backfiller. When that lands, switching to clamped ordering is a one-statement migration plus an index swap.

## Consequences

### Code changes required

- New: [appview/migrations/000010_craftsky_posts.up.sql](../../../appview/migrations/) and `.down.sql`.
- New: [appview/internal/index/craftsky_post.go](../../../appview/internal/index/) and `craftsky_post_test.go`.
- Modified: [appview/internal/app/deps.go](../../../appview/internal/app/deps.go) — register the new indexer.
- Modified: [docker-compose.yml](../../../docker-compose.yml) — extend `TAP_COLLECTION_FILTERS`.

No changes to lexicons, generated lexicon types, the dispatcher, the Tap consumer, or any auth/OAuth code.

### Migration path

None for existing data — there are no `craftsky_posts` rows pre-migration. Down migration drops the indexes and the table cleanly; no data loss path because no other migration depends on this one.

### Performance and storage

- Writes: one INSERT per indexer event. JSONB marshalling of `record` and `facets`/`images` is not on a hot path (Tap throughput is bounded by the firehose). Membership check is a single indexed lookup.
- Reads (future, this spec is write-side only): chronological feeds use `craftsky_posts_indexed_at_desc`. Author timeline uses `craftsky_posts_did_indexed_at_desc`. Hashtag queries use `craftsky_posts_tags_gin`.
- Storage: ~25 KB worst case per post (fat project record), ~200 bytes per plain text post. At 100 M posts, < 2.5 TB worst case; realistically <500 GB.

### Risks

- **Posts before profiles.** A post arriving before its author's `craftsky_profiles` row is dropped. Mitigation: in practice the Craftsky composer cannot run before profile creation, so this is rare from the official client. If third-party clients or out-of-order Tap delivery cause real drops, the fix is a post-backfiller (analogue of [bluesky_backfiller.go](../../../appview/internal/index/bluesky_backfiller.go)). Tracked as future work; not a blocker for v1.
- **Malformed `createdAt` poison-pill.** A record with an unparseable timestamp errors out and Tap retries up to `MaxRetries` before giving up. The retry budget is wasted but not catastrophic; the consumer's poison-pill guard covers this.
- **Hashtag tag set diverges from project tag set.** Until project-field materialisation lands, project posts' `project.common.tags` are not in `tags`. A query for `tags @> ['fair-isle']` will miss project posts that declare `fair-isle` as a structured project tag but not as an inline `#hashtag`. Acceptable for this pass; the union is part of the project-fields spec.
- **`embed` union grows.** The lexicon's `embed` is an open union; this pass materialises only `quoteEmbed`. New variants (record-with-media, video, etc., if added) will be ignored at the column level but preserved in `record`. A migration adds columns when needed.
- **Tag extraction tolerates ugly tags.** Facet-derived tags don't satisfy the kebab-case pattern enforced for composer-merged tags. The lexicon-fields spec already accepts this trade-off; surfaced here as a documentation handoff to the future read endpoint.

## Open questions

None at time of writing. Resolved during brainstorming:

- **Scope:** option A — minimum slice (text + non-project fields). Project fields deferred to a later spec.
- **Membership gating:** required (option A from Q2). Posts before profiles are dropped permanently this pass.
- **Feed ordering:** server-side `indexed_at` (option B from Q3).
- **Materialised non-project fields:** all four — `facets`, `images`, `reply_*`, `quote_*` — as columns (option A from the materialisation question), with `tags TEXT[]` extracted from facet hashtag features for indexed search.
- **`record JSONB`:** in. Forward-compat is worth the storage.
