# Coding Plan: AppView Search Foundation

## 1. Inputs

- Requirements: `docs/changes/2026-06-19-appview-search-foundation/01-requirements.md`
- Acceptance tests: `docs/changes/2026-06-19-appview-search-foundation/02-acceptance-tests.md`
- Document review: `docs/changes/2026-06-19-appview-search-foundation/03-document-review.md` (`Approved`)
- Additional review notes supplied by the user for this stage:
  - Treat the requirements, acceptance tests, and review as source of truth.
  - Keep this AppView-only; do not infer Flutter UI work.
  - Use documented response examples and canonical no-`#` or URL-encoded `%23` hashtag paths in tests.
  - Centralize popularity scoring and cursor seek values early.
  - Plan recent-search persistence and search-supporting migrations deliberately with `MAN-001` in mind.
  - Keep recent searches AppView-private, DID-scoped, hard-deleted, idempotent on nonexistent/not-owned delete, excluded from PDS writes, and excluded from verbose logs.
- Repository references inspected:
  - `atproto-craft-social-app-reference.md`
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
  - `appview/internal/routes/routes.go`, `routes_test.go`
  - `appview/internal/api/post.go`, `post_store.go`, `post_response.go`
  - `appview/internal/api/timeline.go`, `timeline_store.go`
  - `appview/internal/api/profile.go`, `profile_store.go`, `profile_response.go`
  - `appview/internal/api/facet.go`, `facet_store.go`, `facet_response.go`
  - `appview/internal/api/envelope/cursor.go`, `cursor_test.go`
  - `appview/internal/api/moderation_policy.go`, `moderation_store.go`
  - `appview/internal/app/deps.go`
  - migrations `000008` through `000018`, especially posts, profiles, follows, interactions, identity cache, project posts, and pattern facets

## 2. Implementation Strategy

Add a dedicated AppView search API family under authenticated `/v1/search/*` routes. Implement it as Go AppView work only: new handler/request/response/store files under `appview/internal/api`, new route registrations in `appview/internal/routes/routes.go`, and one deliberate migration for recent-search persistence plus search-supporting indexes/columns. Do not edit Flutter UI, Flutter repositories, Flutter routes, lexicons, or PDS write behavior.

Use existing AppView patterns rather than introducing a new framework:

- `net/http` handler factories in `internal/api`.
- Existing auth/device middleware in `routes.go`.
- Existing `envelope.WriteError`, `envelope.EncodeCursor`, and camelCase JSON conventions.
- Existing `PostResponse` and profile summary builders for result items.
- Raw `pgx` SQL in stores; this repo currently has no `appview/queries/*.sql` sqlc files.
- Existing moderation predicates (`postVisibleModerationPredicate`, `profileVisibleModerationPredicate`) for hide/takedown filtering.

Front-load three shared building blocks before broad endpoint work:

1. `search_request.go` for normalization/validation constants and typed query parsing.
2. `search_ranking.go` for the v1 popularity formula and SQL/order-by tuple definitions.
3. `search_cursor.go` for centralized chronological, popularity, and profile-relevance cursor payloads.

That sequencing protects deterministic pagination (`AC-012`, `AC-013`, `AC-015`, `AC-026`) and keeps the popularity formula from being copied into endpoint-specific code.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Routes | `routes.AddRoutes` wires authenticated + device-id routes directly on `http.ServeMux` | Register `GET /v1/search/hashtags/{tag}/posts`, `GET /v1/search/profiles`, `GET /v1/search/posts`, `GET /v1/search/projects`, `GET /v1/search/hashtags/top`, `GET/POST/DELETE /v1/search/recent` | FR-001, NFR-001 | AT-001, IT-001, REG-001 |
| Request parsing | Per-feature parse helpers in `api/*_request.go`; cursor helpers in `api/envelope` | Add bounded search query parsing, project filter parsing, hashtag normalization, sort parsing, recent payload validation | FR-003, FR-008, FR-018, NFR-002, RULE-003 | UT-001, UT-002, UT-003, UT-004 |
| Result ranking/cursors | Existing seek cursors are endpoint-local maps | Centralize chronological and popularity seek tuples; add deterministic profile relevance cursor | FR-016, FR-017, NFR-003, RULE-006 | UT-005, UT-009, IT-009, IT-010, IT-011 |
| Post/project search store | `PostStore` owns timeline/profile post queries and `EngagementSummaries` | Add `SearchStore` query methods for exact hashtags, post keyword search, project filter/browse, profile search, top hashtags; reuse `PostResponse` hydration | FR-002, FR-006, FR-007, FR-009, FR-010, FR-019, FR-020 | AT-002, AT-006, AT-007, AT-008, AT-009, AT-010, IT-002, IT-005, IT-006, IT-012, IT-013 |
| Profile search | Facet mention autocomplete uses identity cache + Craftsky profiles | Add committed profile result search over handle/display name/description with followed-first relevance and deterministic tie-breakers | FR-004, FR-005, RULE-003, NFR-005 | AT-003, UT-006, IT-004 |
| Top hashtags | `/v1/facets/hashtags` returns substring autocomplete counts | Add separate grouped top-hashtag endpoint sourced from project posts only, distinct-project counted, 28-day default window | FR-011, FR-012 | AT-005, UT-007, IT-007, REG-002 |
| Recent searches | No recent-search persistence exists | Add AppView-private `craftsky_recent_searches` table and save/list/delete store + handlers | FR-013, FR-014, FR-015, FR-021, FR-022, RULE-001, RULE-004, RULE-005 | AT-004, UT-004, IT-008, IT-014, IT-015, REG-005 |
| Migrations/indexes | Posts/projects already have tag/project indexes; no FTS/recent table | Add next migration (`000019_*` if still current) for recent table, FTS/search columns, trigram/profile indexes, and project-filter search indexes | NFR-002, NFR-005, NFR-006 | AT-011, MAN-001 |
| Logging/privacy | Handlers log endpoint/run data; some existing debug logs include request objects | Search handlers must avoid logging full recent payloads and long free-text queries; log bounded metadata only | RULE-001, FR-015 | MAN-003, REG-005 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000019_search_foundation.up.sql` | Create | Recent-search table, search vectors/indexes, lower-case generated filter arrays/indexes, profile trigram support | FR-013, FR-017, NFR-002, NFR-005, NFR-006, RULE-001 | AT-004, AT-011, IT-008, MAN-001 |
| `appview/migrations/000019_search_foundation.down.sql` | Create | Drop indexes/columns/table/function/extension-created artifacts in safe reverse order | FR-013, NFR-002 | IT-008, MAN-001 |
| `appview/internal/api/search_request.go` | Create | Constants and parsers for limits, sorts, hashtag path normalization, profile/post q validation, project filters, top hashtag params, recent payload validation | FR-003, FR-008, FR-018, NFR-002, RULE-003 | UT-001, UT-002, UT-003, UT-004 |
| `appview/internal/api/search_cursor.go` | Create | Centralized cursor encode/decode for chronological, popularity, and profile relevance seek tuples | FR-016, FR-017 | UT-009, IT-009 |
| `appview/internal/api/search_ranking.go` | Create | V1 popularity formula, rank tuple constants/helpers, profile relevance classifier | FR-005, FR-017, NFR-003 | UT-005, UT-006, MAN-002 |
| `appview/internal/api/search_response.go` | Create | Search-specific wrapper DTOs and builders for post pages, hashtag metadata, profile summaries, top hashtag groups, recents | FR-009, FR-011, FR-013, NFR-001 | UT-008, IT-012, MAN-004 |
| `appview/internal/api/search_store.go` | Create | `SearchStore` for hashtag/post/project/profile/top-hashtag result queries and engagement aggregation for ranking | FR-002, FR-004, FR-006, FR-007, FR-010, FR-012, FR-019, FR-020 | IT-002 through IT-007, IT-009 through IT-013 |
| `appview/internal/api/search_recent_store.go` | Create | Recent-search persistence, de-duplication, pruning, DID-scoped list, idempotent hard delete | FR-013, FR-014, FR-015, FR-021, FR-022 | UT-004, IT-008, IT-014 |
| `appview/internal/api/search.go` | Create | HTTP handlers for all `/v1/search/*` endpoints, response hydration, errors, auth DID extraction, privacy-safe logs | FR-001 through FR-022, NFR-001 | AT-001 through AT-011, IT-001, IT-015 |
| `appview/internal/routes/routes.go` | Change | Instantiate `searchStore := api.NewSearchStore(deps.DB)` and register all search routes through `authN(deviceID(...))` | FR-001, NFR-001 | AT-001, IT-001, REG-001 |
| `appview/internal/routes/routes_test.go` | Change | Add search route auth/device coverage; preserve existing non-search wrapping tests | FR-001, NFR-001 | AT-001, IT-001, REG-001 |
| `appview/internal/api/search_request_test.go` | Create | Unit tests for normalization, validation, project filters, recent payloads | FR-003, FR-008, FR-018, NFR-002 | UT-001, UT-002, UT-003, UT-004 |
| `appview/internal/api/search_cursor_test.go` or `envelope/cursor_test.go` | Create/Change | Search cursor tuple round-trip and invalid cursor mapping | FR-016 | UT-009 |
| `appview/internal/api/search_ranking_test.go` | Create | Popularity formula and tie-breakers | FR-017, NFR-003 | UT-005, MAN-002 |
| `appview/internal/api/search_profile_rank_test.go` | Create | Profile match class ordering and followed-first behavior | FR-005, NFR-005 | UT-006 |
| `appview/internal/api/search_top_hashtags_test.go` | Create | Distinct project counting and empty craft groups | FR-011, FR-012 | UT-007 |
| `appview/internal/api/search_response_test.go` | Create | Wrapper JSON shapes, no `popularityScore`, existing post/profile nested shape reuse | FR-009, NFR-001 | UT-008, MAN-004 |
| `appview/internal/api/search_store_test.go` | Create | Store/integration coverage for hashtag equality, profile search, post/project search, top hashtags, pagination, popularity, moderation | BR-001, BR-002, BR-004, BR-005, BR-006, FR-010 | IT-002 through IT-007, IT-009 through IT-013 |
| `appview/internal/api/search_recent_store_test.go` | Create | Recent save/list/delete privacy, de-dupe, pruning, hard delete | BR-003, FR-013 through FR-015, FR-021, FR-022 | IT-008, IT-014 |
| Existing facet/post/profile tests | Change only as needed | Add regression assertions without changing `/v1/facets/*` semantics or core `PostResponse`/profile contracts | NG-004, FR-009 | REG-002, REG-003, REG-004, REG-005 |

## 5. API Contract And Response Shapes

All `/v1/search/*` endpoints are authenticated and device-id protected. Use camelCase JSON, bare success bodies, standard error envelopes, and object-wrapped lists with optional `cursor` omitted when absent.

### 5.1 Exact hashtag posts

Route:

```text
GET /v1/search/hashtags/{tag}/posts?sort=chronological|popular&limit=25&cursor=...
```

Path examples for tests must be canonical no-`#` or URL-encoded if explicitly testing a leading hash:

```text
GET /v1/search/hashtags/SockKAL/posts
GET /v1/search/hashtags/%23SockKAL/posts
```

Never use `GET /v1/search/hashtags/#SockKAL/posts` in tests or docs because raw `#` is an HTTP fragment and will not reach the server path.

Response:

```json
{
  "hashtag": "sockkal",
  "items": [
    { "uri": "at://did:plc:alice/social.craftsky.feed.post/abc", "text": "..." }
  ],
  "cursor": "opaque"
}
```

`items` are full existing `PostResponse` objects. `hashtag` is the canonical normalized tag with no leading `#`. Internal popularity scores must not appear.

### 5.2 Profile search

Route:

```text
GET /v1/search/profiles?q=ali&limit=25&cursor=...
```

No `sort` query parameter is supported. `sort=popular` or `sort=chronological` returns validation error.

Response:

```json
{
  "items": [
    {
      "did": "did:plc:alice",
      "handle": "alice.craftsky.social",
      "displayName": "Alice",
      "description": "Knitter and sock mender",
      "avatar": "https://cdn.bsky.app/img/avatar/plain/did:plc:alice/baf...@jpeg",
      "isCraftskyProfile": true,
      "viewerIsFollowing": true
    }
  ],
  "cursor": "opaque"
}
```

This should reuse `ProfileAccountSummary` conventions but may add `viewerIsFollowing` in a search-specific summary because ranking and UI display need that state. Results are Craftsky profiles only.

### 5.3 General post search

Route:

```text
GET /v1/search/posts?q=alpaca&sort=chronological|popular&limit=25&cursor=...
```

`q` is required and must be non-empty after trim.

Response:

```json
{
  "items": [ { "uri": "at://...", "project": { "common": { "craftType": "knitting" } } } ],
  "cursor": "opaque"
}
```

Items are existing `PostResponse` objects and include project data when present.

### 5.4 Project search / browse

Route:

```text
GET /v1/search/projects?q=sock&craftType=knitting&projectType=socks&color=blue&material=alpaca&designTag=cables&projectTag=kal&patternDifficulty=intermediate&sort=chronological|popular&limit=25&cursor=...
```

`q` is optional. With no `q` and no filters, the endpoint returns all visible top-level projects in default chronological order; `sort=popular` is valid for browse-all.

Filter semantics:

- OR within repeated values of the same filter family.
- AND across different filter families.
- User-facing values match case-insensitively.
- Unsupported filter keys or invalid values return validation errors rather than broadening results.
- At most 10 values per filter family and 50 total filter values.

Response shape matches general post search: `{ "items": [PostResponse], "cursor": "opaque" }`.

### 5.5 Top hashtags grouped by craft type

Route:

```text
GET /v1/search/hashtags/top?craftTypes=knitting&craftTypes=crochet&limit=10
```

When no `craftTypes` are supplied, return all supported craft groups. Count only distinct top-level project posts inside the v1 28-day window.

Response:

```json
{
  "groups": [
    {
      "craftType": "knitting",
      "items": [
        { "tag": "sock", "count": 12 },
        { "tag": "sweater", "count": 8 }
      ]
    },
    {
      "craftType": "quilting",
      "items": []
    }
  ]
}
```

### 5.6 Recent searches

Routes:

```text
GET    /v1/search/recent
POST   /v1/search/recent
DELETE /v1/search/recent/{id}
```

Recent searches are AppView-private and capped at the latest 50 per authenticated DID. They are not PDS records and result endpoints never auto-save them.

Save request examples:

```json
{ "type": "hashtag", "displayLabel": "#Sock", "payload": { "tag": "sock", "sort": "chronological" } }
{ "type": "profile", "displayLabel": "ali", "payload": { "q": "ali" } }
{ "type": "post", "displayLabel": "alpaca", "payload": { "q": "alpaca", "sort": "popular" } }
{ "type": "project", "displayLabel": "Knitting socks", "payload": { "q": "sock", "sort": "chronological", "filters": { "craftType": ["knitting"], "projectTag": ["sock"] } } }
```

List/save response item:

```json
{
  "id": "recent_01J...",
  "type": "project",
  "displayLabel": "Knitting socks",
  "payload": {
    "q": "sock",
    "sort": "chronological",
    "filters": { "craftType": ["knitting"], "projectTag": ["sock"] }
  },
  "updatedAt": "2026-06-19T12:34:56Z"
}
```

Duplicate saves are keyed by authenticated DID + type + normalized payload hash. A duplicate refreshes `updatedAt` and moves the row to the top while preserving the existing stored `displayLabel`.

Delete response should be an idempotent success for owned, not-owned, nonexistent, or already-deleted IDs. Prefer `204 No Content` to align with the API architecture's DELETE convention; `200 {}` is acceptable only if existing tests in this repo require JSON success bodies.

## 6. Services, Interfaces, And Data Flow

### 6.1 Route wiring

Sketch:

```text
postStore := api.NewPostStore(deps.DB)      // already exists for post routes
searchStore := api.NewSearchStore(deps.DB)  // new, may internally reuse PostStore helpers

mux.Handle("GET /v1/search/hashtags/{tag}/posts",
  authN(deviceID(api.SearchHashtagPostsHandler(searchStore, deps.HandleResolver, deps.Logger))))
mux.Handle("GET /v1/search/profiles",
  authN(deviceID(api.SearchProfilesHandler(searchStore, deps.Logger))))
mux.Handle("GET /v1/search/posts",
  authN(deviceID(api.SearchPostsHandler(searchStore, deps.HandleResolver, deps.Logger))))
mux.Handle("GET /v1/search/projects",
  authN(deviceID(api.SearchProjectsHandler(searchStore, deps.HandleResolver, deps.Logger))))
mux.Handle("GET /v1/search/hashtags/top",
  authN(deviceID(api.TopHashtagsHandler(searchStore, deps.Logger))))
mux.Handle("GET /v1/search/recent",
  authN(deviceID(api.ListRecentSearchesHandler(searchStore, deps.Logger))))
mux.Handle("POST /v1/search/recent",
  authN(deviceID(api.SaveRecentSearchHandler(searchStore, deps.Logger))))
mux.Handle("DELETE /v1/search/recent/{id}",
  authN(deviceID(api.DeleteRecentSearchHandler(searchStore, deps.Logger))))
```

No new `app.Deps` field is required unless implementation chooses to separate `RecentSearchStore` from `SearchStore`. If separated, construct it from `deps.DB` inside `routes.go` rather than adding process-wide state.

### 6.2 Store interfaces

Use small interfaces in handlers so tests can inject fakes:

```text
type SearchReader interface {
  ListHashtagPosts(ctx, viewerDID, tag string, sort SearchSort, limit int, cursor string, now time.Time) ([]*PostRow, string, error)
  SearchPosts(ctx, viewerDID string, req PostSearchRequest, limit int, cursor string, now time.Time) ([]*PostRow, string, error)
  SearchProjects(ctx, viewerDID string, req ProjectSearchRequest, limit int, cursor string, now time.Time) ([]*PostRow, string, error)
  SearchProfiles(ctx context.Context, viewerDID string, req ProfileSearchRequest, limit int, cursor string) ([]ProfileSearchRow, string, error)
  TopHashtags(ctx context.Context, req TopHashtagsRequest, now time.Time) ([]TopHashtagGroupRow, error)
  EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error)
}

type RecentSearchStore interface {
  ListRecentSearches(ctx context.Context, viewerDID string) ([]RecentSearchRow, error)
  SaveRecentSearch(ctx context.Context, viewerDID string, req SaveRecentSearchRequest, now time.Time) (RecentSearchRow, error)
  DeleteRecentSearch(ctx context.Context, viewerDID, id string) error
}
```

`SearchStore` can implement both interfaces. Internally, it may hold `postStore *PostStore` and delegate `EngagementSummaries` to avoid duplicating existing response engagement code.

### 6.3 Post response hydration flow

All post/project/hashtag result handlers should share one response-building helper:

```text
func buildSearchPostResponses(ctx, rows, viewerDID, store, resolver) ([]*PostResponse, error):
  collect row.URI
  summaries := store.EngagementSummaries(ctx, viewerDID, uris)
  handles := resolveHandlesForRows(ctx, rows, resolver)
  for each row:
    resp := BuildPostResponse(row, handles[row.DID])
    applyEngagementSummary(resp, summaries[row.URI])
    append
```

Guardrails:

- Do not expose `popularityScore`.
- Do not make per-result PDS calls; handle resolution uses the existing resolver/cache path as current post/timeline handlers do.
- Moderation filtering happens in store queries before ranking/limiting.
- Replies/comments are excluded from hashtag, post, project, and top-hashtag result sets unless a future requirement changes that.

### 6.4 Popularity ranking

Central helper:

```text
const popularityReplyWeight = 2
const popularityRepostWeight = 3
const popularityDecayHours = 72.0
const popularityDecayExponent = 1.5

func PopularityScore(likes, visibleReplies, reposts int, createdAt, rankedAt time.Time) float64
```

Formula:

```text
weightedEngagement = likes + (2 * visibleReplies) + (3 * reposts)
score = weightedEngagement / pow(1 + ageHours / 72, 1.5)
```

SQL/store ranking must use the same expression and tie-breakers:

```text
ORDER BY popularity_score DESC, p.created_at DESC, p.uri DESC
```

For `sort=popular`, freeze `rankedAt` for the first page and carry it in the cursor so subsequent pages use the same age/decay basis. This avoids page drift while a user follows a cursor.

Popularity counts:

- active likes only (`craftsky_likes.deleted_at IS NULL`)
- active reposts only (`craftsky_reposts.deleted_at IS NULL`)
- visible descendant replies/comments only; hidden/takedown descendants must be excluded from ranking counts

Response engagement counts can continue to use existing `EngagementSummaries` unless the builder chooses to update it to exclude hidden descendants globally. The requirement that hidden descendants are excluded is mandatory for ranking.

### 6.5 Cursor payloads

Centralize cursor encoding/decoding in `search_cursor.go`; do not hand-roll endpoint-specific maps in handlers.

Chronological result cursor:

```json
{ "sort": "chronological", "createdAt": "2026-06-19T12:00:00Z", "uri": "at://..." }
```

Popularity result cursor:

```json
{ "sort": "popular", "rankedAt": "2026-06-19T12:34:56Z", "score": 1.2345, "createdAt": "2026-06-19T12:00:00Z", "uri": "at://..." }
```

Profile relevance cursor:

```json
{ "kind": "profile", "followedRank": 0, "relevanceRank": 1, "handleLower": "alice.craftsky.social", "did": "did:plc:alice" }
```

Handlers map malformed or mismatched cursor payloads to `400 invalid_cursor`.

### 6.6 Hashtag normalization

Sketch:

```text
func NormalizeHashtagPath(raw string) (string, error):
  s := strings.TrimSpace(raw)           // PathValue is already URL-decoded by net/http
  s = strings.TrimPrefix(s, "#")        // remove one leading # only
  s = strings.TrimSpace(s)
  s = strings.ToLower(s)
  reject empty, over 128, whitespace/control/slash, or values not representable as a stored tag
```

`##sock` becomes `#sock` after removing one leading hash and should be rejected unless existing tag validation explicitly allows `#` inside stored tags. This preserves the acceptance-test expectation that one leading hash is stripped, not arbitrary hash cleanup.

### 6.7 Profile relevance

Classify matching profiles explicitly; do not rely on opaque trigram similarity for final ordering.

Rank tuple:

```text
followedRank: 0 if viewer follows profile else 1
relevanceRank:
  0 exact handle match
  1 handle prefix match
  2 handle substring match
  3 display name match
  4 description/bio match
tie-breakers: handleLower ASC, did ASC
```

`pg_trgm` may be used to find candidate rows efficiently, but the final `ORDER BY` must preserve this explicit tuple.

## 7. Migration And Persistence Plan

Use the next migration number after the current highest (`000018` at planning time): `000019_search_foundation`. Implementer must verify the current migration number before creating files.

### 7.1 Recent-search table

Planned table:

```text
craftsky_recent_searches
  id TEXT PRIMARY KEY                         -- opaque server-generated ID, e.g. recent_<ULID>
  viewer_did TEXT NOT NULL                    -- authenticated DID; no FK to public profile required
  search_type TEXT NOT NULL CHECK (...)
  display_label TEXT NOT NULL                 -- max 120 after trim
  normalized_payload JSONB NOT NULL           -- max 4096 bytes encoded
  normalized_payload_hash TEXT NOT NULL       -- sha256 hex of canonical normalized payload
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()

UNIQUE (viewer_did, search_type, normalized_payload_hash)
INDEX (viewer_did, updated_at DESC, id DESC)
```

Do not add `deleted_at`; recent-search delete is a hard delete.

Save flow:

```text
canonicalPayload := normalizeRecentPayload(type, payload)
hash := sha256(canonical JSON bytes)
INSERT ... ON CONFLICT (viewer_did, search_type, normalized_payload_hash)
  DO UPDATE SET updated_at = excluded.updated_at
  -- intentionally do NOT overwrite display_label or normalized_payload on duplicate
prune rows where row_number() over (partition by viewer_did order by updated_at desc, id desc) > 50
```

### 7.2 FTS/search-supporting columns and indexes

Decide in the migration rather than deferring vaguely to implementation:

1. Add local FTS vectors for document-like post/project search:
   - `craftsky_posts.search_vector` generated from `text`.
   - `craftsky_project_posts.search_vector` generated from common project fields: `common_title`, `pattern_name`, `materials`, `project_tags`, `design_tags`.
   - GIN indexes on both vectors, scoped where possible to top-level visible/browseable rows.
2. Add `pg_trgm` support immediately for profile candidate lookup:
   - `CREATE EXTENSION IF NOT EXISTS pg_trgm;`
   - trigram indexes on `atproto_identity_cache.handle_lower`, `lower(bluesky_profiles.display_name)`, and `lower(bluesky_profiles.description)` where useful.
   - Preserve explicit deterministic ranking in SQL/tests; trigram is only a scalable candidate-finding aid.
3. Add deterministic chronology indexes for search result surfaces:
   - top-level posts: `(created_at DESC, uri DESC)` with root-post predicates.
   - top-level projects: `(created_at DESC, uri DESC)` where `is_project=true` and standalone project predicates hold.
4. Add case-insensitive project filter support:
   - btree expression indexes on lower scalar fields such as `common_craft_type`, `pattern_difficulty`, and craft-specific project type columns or a generated `project_types_search` array.
   - generated lower-case array columns plus GIN indexes for `materials`, `colors`, `design_tags`, and `project_tags`, if existing materialized arrays preserve author casing.
   - If implementation proves existing arrays are already normalized lowercase, the builder may keep current GIN indexes and document that finding in `MAN-001`; otherwise add generated lower arrays to avoid query-time unindexed `lower(unnest(...))` scans.

Potential helper for generated lower arrays:

```text
CREATE FUNCTION craftsky_lower_text_array(input TEXT[]) RETURNS TEXT[] IMMUTABLE ...
ALTER TABLE craftsky_project_posts ADD COLUMN materials_search TEXT[] GENERATED ALWAYS AS (craftsky_lower_text_array(materials)) STORED;
CREATE INDEX ... USING GIN (materials_search);
```

The down migration must drop dependent generated columns/indexes before dropping helper functions.

### 7.3 Manual check hooks

After implementation, `MAN-001` should review representative `EXPLAIN` output for:

- exact hashtag equality (`tags @> ARRAY[...]` or equivalent)
- chronological and popularity post/project result queries
- post/project FTS `q` search
- project filters across scalar and array fields
- profile search candidate lookup
- recent-search list by DID
- top hashtag grouping by craft type and 28-day window

## 8. Query Semantics

### 8.1 Exact hashtag search

Filter:

```text
p.reply_root_uri IS NULL
p.reply_parent_uri IS NULL
lower(tag) equality against canonical tag using materialized p.tags only
```

Do not parse raw `p.text` for visual hashtags. Do not include replies/comments. Include both regular and project posts.

### 8.2 General post search

Filter:

```text
q required
top-level only
visible posts only
matches p.search_vector OR project search vector when a project row exists
```

Search fields: post text plus project title, pattern name, material text/tags, project tags, design tags. Craft-specific detail fields remain filters or future work.

### 8.3 Project search

Base filter:

```text
p.is_project = true
p.reply_root_uri IS NULL
p.reply_parent_uri IS NULL
p.quote_uri IS NULL
visible post/account only
```

Supported query parameters:

```text
q
craftType
projectType
patternDifficulty
color
material
designTag
projectTag
sort
limit
cursor
```

Use repeated query params for repeated values, e.g. `craftType=knitting&craftType=crochet`. Keep validation strict; unknown keys should return a standard validation error.

### 8.4 Top hashtags

Source project posts only:

```text
p.is_project = true
p.created_at >= now - 28 days
p is top-level standalone visible project
group by pp.common_craft_type and lower(trim(tag))
count distinct p.uri per craft/tag
```

The tag source should include materialized `craftsky_posts.tags` because it already merges text/project tag sources. Count each project once per tag even if the same tag appears in multiple materialized sources.

Return requested craft groups even when no rows match.

## 9. Error, Empty, And Edge States

| Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Missing auth/device header | Existing middleware returns standard error envelope | FR-001, NFR-001 | AT-001, IT-001 |
| Raw `#` in URL path example | Do not use in tests/docs; use `/Sock` or `/%23Sock` | FR-003 | DR-002, UT-001, IT-002 |
| Hashtag empty after trim/one `#` removal | 400/422 validation error, never all posts | FR-003, NFR-002 | UT-001, UT-002 |
| Hashtag text-only appearance with no stored tag | Not returned | RULE-002 | AT-002, IT-003 |
| Profile `sort` supplied | Validation error; profile relevance order only | RULE-003 | AT-003, UT-002 |
| `/v1/search/posts` missing/blank `q` | Validation error; not a discovery feed | FR-006 | AT-007, UT-002 |
| `/v1/search/projects` with no filters and no `q` | Browse all visible projects chronological by default | FR-020, RULE-006 | AT-006, IT-006 |
| Unsupported project filter key/value | Validation error; do not silently broaden | FR-018 | AT-006, UT-003 |
| Hidden/taken-down result would rank first | Filter before ranking/limiting | FR-010 | AT-010, IT-013 |
| Popularity score tie | `created_at DESC, uri DESC` | FR-017 | UT-005, IT-010, IT-011 |
| Invalid cursor | 400 `invalid_cursor` | FR-016 | UT-009, IT-009 |
| Duplicate recent save with new label | Refresh `updatedAt`; preserve existing label | FR-021 | AT-004, UT-004, IT-008 |
| Recent delete not owned/nonexistent | Hard-delete if owned; otherwise idempotent success without leakage | FR-015, RULE-005 | AT-004, IT-014 |
| Search result endpoints called while typing | Return results but do not save recents | RULE-004 | IT-015 |
| Recent/search payload logging | Log endpoint/type/count/run ID only; no full payload or long query in verbose logs | RULE-001 | MAN-003 |

## 10. Test Implementation Plan

Follow the acceptance-test suggested order, with shared ranking/cursor work moved early.

| Order | Test IDs | Target | Initial Expected Failure |
|---|---|---|---|
| 1 | IT-001, AT-001, REG-001 | `routes_test.go`, `search_test.go` | No `/v1/search/*` routes; auth/device cases 404 |
| 2 | UT-002 | `search_request_test.go` | No shared validation constants/parser |
| 3 | UT-001, IT-002, IT-003, AT-002 | `search_request_test.go`, `search_store_test.go` | Hashtag route/search not implemented; normalization absent |
| 4 | UT-009, IT-009, AT-009 | `search_cursor_test.go`, `search_store_test.go` | No stable search cursor helpers |
| 5 | UT-005, IT-010, IT-011, AT-008 | `search_ranking_test.go`, `search_store_test.go` | Popularity formula/order not centralized/implemented |
| 6 | UT-008, IT-012, REG-003 | `search_response_test.go`, post response tests | Search wrappers absent; post/project response reuse untested |
| 7 | UT-006, IT-004, AT-003 | `search_profile_rank_test.go`, `search_store_test.go` | No profile search/ranking |
| 8 | IT-005, AT-007 | `search_store_test.go`, `search_test.go` | No post/project FTS keyword search; missing post `q` validation |
| 9 | UT-003, IT-006, AT-006 | `search_request_test.go`, `search_store_test.go` | No project filter parser/query semantics |
| 10 | UT-007, IT-007, AT-005 | `search_top_hashtags_test.go`, `search_store_test.go` | No grouped top hashtag query |
| 11 | UT-004, IT-008, IT-014, AT-004 | `search_recent_store_test.go`, `search_test.go` | No recent table/store/handlers |
| 12 | IT-013, AT-010, REG-004 | `search_store_test.go`, moderation tests | Moderation not applied across all search surfaces |
| 13 | IT-015, AT-011, REG-002, REG-005 | Handler/store tests plus manual review | Search endpoints may auto-save; facet regression/privacy/index checks missing |
| 14 | MAN-001 through MAN-004 | Manual review | Query plans/log redaction/API contracts not reviewed |

Commands:

```text
just dev-d
just test
just fmt
```

For faster loops, run targeted Go tests from `appview/` with `TEST_DATABASE_URL` set after the compose Postgres is available.

## 11. Sequencing And Guardrails

1. Add route tests and empty handler scaffolding first (`IT-001`/`AT-001`).
2. Add request parsing, ranking, and cursor helpers before store queries.
3. Add migration/schema support before integration store tests requiring recent-search persistence or FTS/indexed paths.
4. Implement exact hashtag search before broader FTS; it is equality against materialized tags, not autocomplete.
5. Implement response wrappers and hydration once, then reuse for hashtag/post/project endpoints.
6. Implement profile search with explicit followed-first relevance; keep `pg_trgm` as candidate support, not ranking source of truth.
7. Implement project filters with strict validation and documented OR/AND semantics.
8. Implement top hashtags from project posts only and keep requested empty craft groups.
9. Implement recent searches last or after migration is stable; keep privacy/logging review close to that code.
10. Run regression tests for facets, moderation, post response shape, and PDS-write absence.

Hard guardrails for the TDD builder:

- No Flutter UI, state, repository, route, or widget work in this slice.
- No lexicon changes and no ADR needed for lexicons.
- No PDS writes for search or recent searches.
- No raw `#` in request URLs in tests.
- No `popularityScore` in public JSON.
- No unbounded query strings, filters, payloads, cursors, or result limits.
- No verbose logs containing full recent-search payloads or long free-text queries.
- No per-result PDS/network lookups in normal search result hydration.

## 12. Risks And Open Implementation Questions

| ID | Risk / Question | Impact | Planned Handling |
|---|---|---|---|
| RISK-001 | Broad AppView slice touches many endpoints and query paths | Implementation could become large and hard to review | Keep files grouped under `search*`, follow staged TDD order, avoid Flutter/PDS/lexicon scope creep |
| RISK-002 | FTS/generated-column migration details may need adjustment for PostgreSQL function immutability or existing data casing | Migration could fail or indexes might not support `MAN-001` | Write migration tests early; if lower-array generated columns are impractical, use an immutable helper function or document normalized-storage proof |
| RISK-003 | Popularity SQL and Go formula can drift | Pagination and tests become flaky | Centralize constants and test both Go helper and SQL ordering fixtures with controlled data |
| RISK-004 | Existing `EngagementSummaries` counts descendant replies without moderation filtering | Public response counts may differ from ranking counts | Mandatory: ranking excludes hidden/takedown replies. Optional: update global summary helper only if existing response tests are adjusted intentionally |
| RISK-005 | Profile search may depend on stale identity cache rows | Some profiles missing until cache/backfill refresh | Use local indexed data only per requirements; do not add network discovery to search path |
| RISK-006 | Recent searches are private user behavior | Privacy/logging mistakes could leak sensitive intent | DID-scope every query, hard delete, idempotent not-owned delete, avoid full-payload logs, add `MAN-003` review |

Open implementation questions: none blocking. The plan chooses immediate FTS and `pg_trgm` support in the migration to satisfy `MAN-001` deliberately; the builder may refine exact index shapes during implementation if tests and manual query-plan review document the final choice.

## 13. Traceability Summary

| Work Item | Requirement IDs | Acceptance / Test IDs |
|---|---|---|
| Authenticated route family | FR-001, NFR-001 | AC-014, AT-001, IT-001, REG-001 |
| Exact hashtag posts | BR-001, FR-002, FR-003, RULE-002 | AC-001, AC-002, AC-021, AT-002, UT-001, IT-002, IT-003 |
| Profile search | BR-002, FR-004, FR-005, RULE-003, NFR-005 | AC-003, AC-004, AT-003, UT-006, IT-004 |
| Recent searches | BR-003, FR-013, FR-014, FR-015, FR-021, FR-022, RULE-001, RULE-004, RULE-005 | AC-005, AC-006, AC-007, AC-018, AC-027, AT-004, UT-004, IT-008, IT-014, IT-015 |
| Top hashtag groups | BR-004, FR-011, FR-012 | AC-008, AC-009, AC-025, AT-005, UT-007, IT-007 |
| Project search/filter/browse | BR-005, FR-007, FR-008, FR-018, FR-020 | AC-010, AC-011, AC-019, AC-023, AT-006, UT-003, IT-006 |
| General post/project keyword search | FR-006, FR-019, NFR-006 | AC-022, AT-007, IT-005 |
| Chronological/popular sort and cursors | BR-006, FR-016, FR-017, NFR-003, RULE-006 | AC-012, AC-013, AC-015, AC-026, AT-008, AT-009, UT-005, UT-009, IT-009, IT-010, IT-011 |
| Response contracts | FR-009, NFR-001 | AC-016, AC-024, UT-008, IT-012, MAN-004, REG-003 |
| Moderation filtering | FR-010 | AC-017, AT-010, IT-013, REG-004 |
| Bounds/indexed local paths | NFR-002, NFR-004, NFR-005, NFR-006 | AC-020, AT-011, MAN-001 |
