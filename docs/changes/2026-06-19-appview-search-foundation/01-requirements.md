# Requirements: AppView Search Foundation

## 1. Initial Request

Build the AppView search foundation for Craftsky. The first search slice should support exact hashtag result searches, semi-fuzzy profile search, AppView-backed recent searches with delete, blank-search top hashtags grouped by craft type, general post search, project-specific filtered search, and result ordering by chronology or popularity where applicable. This is AppView work only, but the API contracts should be shaped for later Flutter search UI consumption.

## 2. Current Codebase Findings

- Relevant files:
  - AppView route registration: `appview/internal/routes/routes.go`.
  - API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`, `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`.
  - Existing facet autocomplete: `appview/internal/api/facet.go`, `appview/internal/api/facet_store.go`, `appview/internal/api/facet_response.go`.
  - Post/query storage and response assembly: `appview/internal/api/post.go`, `appview/internal/api/post_store.go`, `appview/internal/api/post_response.go`, `appview/internal/api/timeline.go`, `appview/internal/api/timeline_store.go`.
  - Profile storage and response assembly: `appview/internal/api/profile.go`, `appview/internal/api/profile_store.go`, `appview/internal/api/profile_response.go`.
  - Search-adjacent migrations: `appview/migrations/000008_craftsky_profiles.up.sql`, `000009_bluesky_profiles.up.sql`, `000010_craftsky_posts.up.sql`, `000011_craftsky_interactions.up.sql`, `000015_identity_handle_cache.up.sql`, `000016_project_posts.up.sql`.
  - Flutter search entry point: `app/lib/search/pages/search_page.dart`, `app/lib/router/router.dart`, `app/lib/shared/rich_text/facet_action_handler.dart`.
- Existing patterns:
  - Every `/v1/*` AppView endpoint other than auth/ops requires Craftsky session auth and `X-Craftsky-Device-Id`.
  - Successful list responses use object-wrapped `items` and optional opaque `cursor` fields.
  - Errors use the standard camelCase envelope: `{error, message, requestId}` plus optional `fields`.
  - Existing post and timeline handlers hydrate `PostResponse` objects and apply engagement summaries from likes, reposts, and replies.
  - Existing moderation predicates hide posts/accounts with active `hide` or `takedown` labels.
  - Existing facet endpoints are autocomplete endpoints, not full search-result endpoints.
- Current behavior:
  - Search endpoints are explicitly out of scope in the v1 API architecture document and are not implemented yet.
  - `GET /v1/facets/hashtags` returns hashtag suggestions by substring query and 28-day root-post counts.
  - `GET /v1/facets/mentions` returns Craftsky profile mention suggestions from `atproto_identity_cache` plus `craftsky_profiles` and `bluesky_profiles`.
  - `craftsky_posts.tags` stores normalized hashtag/tag values and has a GIN index.
  - `craftsky_project_posts` materializes project fields such as craft type, status, title, pattern difficulty, materials, colors, design tags, project tags, and craft-specific project type/subtype fields.
  - `craftsky_likes` and `craftsky_reposts` store active/deleted interactions, and replies are represented as `craftsky_posts` rows with reply pointers.
  - Flutter `SearchPage` is currently a stub that can receive a `tag` query parameter from hashtag facet taps.
- Constraints discovered:
  - No lexicon change is needed for this slice; search reads existing indexed public data and stores private recent-search state in AppView Postgres.
  - Private-by-intent data should not be written to the PDS under the project privacy rule.
  - Exact hashtag result search must not reuse substring hashtag autocomplete semantics.
  - AppView search must return client-shaped REST/JSON responses under `/v1/`, not XRPC.
  - Project filtering can start from materialized fields already present in `craftsky_project_posts`, but additional indexes or generated search columns may be needed for acceptable query performance.
- Test/build commands discovered:
  - AppView tests: `just test` from the repo root after the compose database is available.
  - Formatting: `just fmt`.
  - Local stack: `just dev`.

## 3. Clarifying Questions And Decisions

### Q1: Should this requirements artifact cover the AppView search foundation as one slice?

Answer: Yes — use the recommended AppView search foundation direction.

Decision / implication: The requirements cover the AppView endpoints, persistence, ranking semantics, filtering semantics, and Flutter-facing API contracts for hashtag search, profile search, post search, project search, top hashtags, and recent searches. Flutter UI implementation remains out of scope.

### Q2: Where should recent searches be stored?

Answer: AppView, not PDS.

Decision / implication: Recent searches are private user behavior and must live in AppView Postgres keyed to the authenticated user DID. They must not be written as atproto records.

### Q3: How should recent searches be added?

Answer: Explicit save.

Decision / implication: AppView must expose an explicit authenticated recent-search save endpoint that the app calls only when the user commits/selects a search. Search-result endpoints must not automatically save every request, avoiding noisy recents from typing or autocomplete.

### Q4: How should profile search be ordered?

Answer: Prefer matches against handle, then display name, then bio/description.

Decision / implication: Profile ranking should make exact/prefix handle matches strongest, then handle substring, then display-name matches, then profile-description matches. Viewer-following can be a secondary boost but must not override stronger textual relevance.

### Q5: What does popularity mean for post/project searches?

Answer: Number of interactions with decay based on age.

Decision / implication: Popularity sorting must use active likes, replies, and reposts/comments as engagement inputs, adjusted by a deterministic recency decay. The v1 formula must be documented in implementation/test specs and centralized so it can evolve later without changing endpoint names.

### Q6: What endpoint family should search use?

Answer: Dedicated `/v1/search/*` endpoints.

Decision / implication: Existing `/v1/facets/*` endpoints remain autocomplete/resolve surfaces. Search-result and search-state APIs should live under `/v1/search/*`.

## 4. Candidate Approaches

### Option A: AppView search foundation with dedicated `/v1/search/*` APIs

Summary: Add dedicated authenticated AppView search endpoints, exact hashtag search, profile search, general post search, project filter search, grouped top hashtags, and AppView-backed recent-search persistence.

Pros:
- Matches the confirmed direction and keeps this chunk focused on AppView logic.
- Gives Flutter a stable API contract before the full search UI is built.
- Keeps autocomplete endpoints separate from committed search-result endpoints.
- Treats recent searches as private AppView state.
- Reuses existing post/profile/project/interaction indexes and response builders where appropriate.

Cons:
- Larger AppView slice with multiple endpoints and query paths.
- Requires careful pagination/ranking contracts so later Flutter work does not need endpoint churn.
- Likely requires at least one migration for recent searches and possibly supporting indexes/search vectors.

Risks:
- Search performance may degrade as indexed data grows if filters and ranking are not indexed deliberately.
- Popularity sorting can become product-sensitive if the formula is hard-coded in many places.

### Option B: Decompose into hashtag/profile search first

Summary: Start with exact hashtag results and profile search only, then add recent searches, top hashtags, project filtering, and popularity sorting in later requirement documents.

Pros:
- Smaller implementation and test surface.
- Lower immediate risk.
- Can validate endpoint patterns before expanding.

Cons:
- Does not satisfy the requested blank search page, project filtering, or ordering needs.
- More likely to create response/endpoint churn across follow-up slices.
- Delays key app-facing contracts the Flutter search UI will need.

Risks:
- Early endpoint shapes may be too narrow for project filters and popularity sorting.

### Option C: One generic search endpoint

Summary: Add one broad `/v1/search` endpoint with type, filter, and sort parameters for profiles, posts, projects, hashtags, and recents.

Pros:
- One entry point for the app.
- Flexible query parameter surface.

Cons:
- Harder to test precisely.
- Profiles, posts, projects, top hashtags, and recent searches have different response shapes and ranking semantics.
- Encourages over-generalization before search behavior is proven.

Risks:
- Endpoint becomes a catch-all with ambiguous validation and weak client contracts.

## 5. Recommended Direction

Recommended approach: Option A — AppView search foundation with dedicated `/v1/search/*` APIs.

Why: It satisfies the confirmed product direction, preserves the API architecture's REST/client-shaped conventions, keeps private recent-search state in AppView, avoids overloading the existing facet-autocomplete endpoints, and gives the future Flutter search UI a clear contract for each search surface.

## 6. Problem / Opportunity

Craftsky needs search that works for the craft community: exact hashtags, discoverable profiles, project filtering, and useful blank-search discovery. The AppView already indexes much of the public data needed to power search, but there is no committed search API surface, no private recent-search state, and no defined popularity ordering. This slice turns the indexed AppView data into stable search contracts without implementing the Flutter UI.

## 7. Goals

- G-001: Let users find posts/projects by an exact hashtag, without substring or approximate tag matches.
- G-002: Let users find Craftsky profiles using semi-fuzzy text search over handle, display name, and bio/description.
- G-003: Let users revisit and manage their own recent searches.
- G-004: Let the blank search page show currently active hashtags grouped by craft type.
- G-005: Let users search posts and filtered projects with chronological or popularity ordering.
- G-006: Provide stable AppView API contracts for later Flutter search UI work.

## 8. Non-Goals

- NG-001: Do not implement Flutter search UI, state management, or repository code in this slice.
- NG-002: Do not change atproto lexicons or write search/recent-search records to a PDS.
- NG-003: Do not implement typo-tolerant search, semantic search, embeddings, recommendations, or algorithmic feed ranking.
- NG-004: Do not replace or remove existing `/v1/facets/*` autocomplete endpoints.
- NG-005: Do not expose unauthenticated public search endpoints in this slice.
- NG-006: Do not add third-party XRPC search APIs.
- NG-007: Do not implement moderation/admin search tooling.
- NG-008: Do not support destructive deletion of public PDS data; recent-search delete only affects private AppView recent-search state.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in Craftsky user | A user browsing or searching from the Craftsky app. | Needs exact hashtag results, relevant profile results, searchable posts/projects, useful discovery suggestions, and control over recent searches. |
| Flutter client | The app consuming AppView search APIs. | Needs authenticated, paginated, camelCase JSON contracts with stable error semantics and response shapes. |
| AppView operator/developer | Maintains indexed search data and AppView performance. | Needs bounded queries, clear indexes, testable ranking rules, and no private search state on PDS. |

## 10. Current Behavior

Search is not implemented as an AppView result surface. The app has a stub search page and hashtag facet taps can navigate to `/search?tag=...`, but the page has no AppView-backed results. Existing `/v1/facets/*` endpoints support mention/hashtag autocomplete while composing text. AppView stores indexed post, profile, project, hashtag, and interaction data that can be queried, but there are no `/v1/search/*` routes, no recent-search persistence, no grouped top-hashtag endpoint, and no defined popularity sort.

## 11. Desired Behavior

The AppView exposes authenticated `/v1/search/*` endpoints for exact hashtag post results, profile search, general post search, project filter search, grouped top hashtags, and recent-search management. Search list endpoints paginate with opaque cursors, use standard AppView error envelopes, apply moderation visibility rules, and return response shapes the Flutter search UI can consume later. Recent searches are saved explicitly by the app and can be listed or deleted by the owning user. Post/project searches can sort chronologically or by a documented popularity score; profile search uses relevance ordering only.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky users shall be able to search for a specific hashtag and receive only visible top-level posts/projects that carry that exact normalized hashtag. | Exact hashtag search is a core requested behavior and differentiates committed search from autocomplete. | Prompt, Discovery | AC-001, AC-002, AC-016 |
| BR-002 | Business | Must | Craftsky users shall be able to discover profiles through semi-fuzzy profile search. | Users need profile discovery beyond exact handle lookup. | Prompt | AC-003, AC-004 |
| BR-003 | Business | Must | Craftsky users shall be able to view and delete their own recent searches. | Recent search history is a requested user-facing search feature. | Prompt, Q2, Q3 | AC-005, AC-006, AC-007 |
| BR-004 | Business | Must | Craftsky users shall be able to see current top hashtags grouped by craft type for blank-search discovery. | Blank search should be useful before a query is typed. | Prompt | AC-008, AC-009 |
| BR-005 | Business | Must | Craftsky users shall be able to search and filter project posts by craft-relevant fields. | Project discovery needs progressive app-side filters backed by AppView query semantics. | Prompt, Codebase | AC-010, AC-011 |
| BR-006 | Business | Must | Craftsky users shall be able to order post and project search results by chronology or popularity. | The user requested sort options for searches except profile searches. | Prompt, Q5 | AC-012, AC-013 |
| FR-001 | Functional | Must | The system shall register dedicated authenticated `/v1/search/*` routes for search results, top hashtags, and recent-search management. | Keeps result search separate from facet autocomplete and follows API architecture. | Q6, Codebase | AC-014, AC-015 |
| FR-002 | Functional | Must | The system shall expose an exact hashtag results endpoint, such as `GET /v1/search/hashtags/{tag}/posts`, accepting `sort`, `limit`, and `cursor`. | Provides a clear API for hashtag facet taps and typed hashtag searches. | Prompt, Q6 | AC-001, AC-002, AC-012, AC-015 |
| FR-003 | Functional | Must | The system shall normalize hashtag inputs by trimming whitespace, removing one leading `#` if present, and matching case-insensitively against stored normalized tag values. | Users may arrive from typed text or facet taps, while storage is normalized. | Codebase, Prompt | AC-001, EC-001, EC-002 |
| FR-004 | Functional | Must | The system shall expose profile search, such as `GET /v1/search/profiles?q=...`, searching Craftsky profiles by cached handle, Bluesky display name, and Bluesky description/bio. | Supports semi-fuzzy profile discovery with existing indexed profile data. | Prompt, Q4, Codebase | AC-003, AC-004, EC-003 |
| FR-005 | Functional | Must | Profile search shall order results by textual relevance: exact/prefix handle matches first, then handle substring, display-name matches, and description/bio matches, with stable deterministic tie-breakers. | Matches the user's preferred profile ordering and makes results predictable. | Q4 | AC-004 |
| FR-006 | Functional | Must | The system shall expose general post search, such as `GET /v1/search/posts?q=...`, over visible top-level posts, including project posts, with chronological and popularity sort support. | Provides broad search that includes projects without forcing users into project-only mode. | Prompt | AC-012, AC-013, AC-016 |
| FR-007 | Functional | Must | The system shall expose project filter search, such as `GET /v1/search/projects`, filtering visible top-level project posts by supported materialized project fields. | Provides AppView support for the app's future progressive project-filter UI. | Prompt, Codebase | AC-010, AC-011, AC-013 |
| FR-008 | Functional | Must | Project filter search shall support at least craft type, craft-specific project type, pattern difficulty, color, material text/tag, design tag, project tag, and optional keyword query where data is materialized; v1 filter semantics shall be OR within repeated values for the same filter family and AND across different filter families. | Captures the filter fields called out by the user and current project materialization. | Prompt, Codebase | AC-010, AC-011 |
| FR-009 | Functional | Must | Search endpoints returning posts/projects shall build the same core post response shape used by existing timeline/profile post endpoints, including author data, project data when present, and engagement summary fields. | Flutter should consume consistent post cards across timeline, profile, and search surfaces. | Codebase, Discovery | AC-016 |
| FR-010 | Functional | Must | Search endpoints shall apply existing post and account moderation visibility rules before limiting, ranking, or returning results. | Hidden/takedown content must not reappear through search. | Codebase | AC-017, EC-004 |
| FR-011 | Functional | Must | The system shall expose grouped top hashtags, such as `GET /v1/search/hashtags/top?craftTypes=...`, returning craft-grouped tags and recent counts. | Supports blank search discovery grouped by craft interest. | Prompt | AC-008, AC-009 |
| FR-012 | Functional | Must | Top hashtag results shall use a bounded recent window, defaulting to 28 days for v1, and shall only include hashtags associated with the requested craft groups or all supported craft groups when no craft filter is provided. | Matches existing 28-day hashtag count behavior and app need to pass craft interests. | Prompt, Codebase | AC-008, AC-009 |
| FR-013 | Functional | Must | The system shall expose recent-search list, save, and delete endpoints, such as `GET /v1/search/recent`, `POST /v1/search/recent`, and `DELETE /v1/search/recent/{id}`. | Recent searches need explicit user-controlled persistence. | Prompt, Q2, Q3 | AC-005, AC-006, AC-007 |
| FR-014 | Functional | Must | Recent-search saves shall accept a bounded typed payload representing committed searches, including at least hashtag search, profile query, post query, and project filter search. | The app must be able to save the types of searches introduced in this slice. | Q3, Prompt | AC-005, AC-006, EC-005 |
| FR-015 | Functional | Must | Recent-search list and delete operations shall be scoped to the authenticated viewer DID. | Search history is private per user. | Q2, AGENTS privacy rule | AC-006, AC-007, AC-018 |
| FR-016 | Functional | Must | Search list endpoints shall support `limit` and opaque `cursor` pagination following existing AppView v1 conventions. | Consistent pagination is required for app consumption and API architecture compliance. | Codebase | AC-015 |
| FR-017 | Functional | Must | Post/project popularity sort shall use active likes, active reposts, and descendant replies/comments as engagement inputs and apply deterministic age decay based on the result post's creation time. | Defines popularity in terms discussed by the user while preventing a pure all-time ranking. | Q5, Codebase | AC-013 |
| FR-018 | Functional | Should | Search query parsing should reject or ignore unsupported filter fields with clear validation behavior instead of silently broadening results. | Avoids app bugs causing misleading or overly broad searches. | Discovery | AC-019, EC-006 |
| NFR-001 | Non-functional | Must | Search APIs shall use `/v1/*` camelCase JSON, standard AppView error envelopes, and authenticated session/device middleware. | Required by existing AppView API conventions. | Codebase | AC-014, AC-015, AC-019 |
| NFR-002 | Non-functional | Must | Search queries shall be bounded by default and maximum limits, bounded query string lengths, and indexed access paths appropriate for expected AppView growth. | Search can become expensive without guardrails. | Discovery | AC-015, AC-020 |
| NFR-003 | Non-functional | Should | Popularity scoring should be centralized and documented so future scoring changes do not require changing endpoint names or response shapes. | The formula is product-sensitive and likely to evolve. | Q5 | AC-013 |
| NFR-004 | Non-functional | Should | Search endpoints should avoid per-result network calls to PDS or identity services in the normal result path. | Search should be responsive and rely on indexed AppView data. | Codebase | AC-020 |
| NFR-005 | Non-functional | Should | Profile search should use PostgreSQL `pg_trgm` as the preferred scalable local indexed strategy once data size requires indexed substring or similarity matching, while preserving explicit deterministic v1 ranking. | Keeps profile search scalable without making ranking opaque. | Review feedback | AC-004, AC-020 |
| RULE-001 | Business rule | Must | Recent searches are private AppView data and must not be written to the PDS or exposed to other users. | Search history is private-by-intent behavior. | Q2, AGENTS privacy rule | AC-018 |
| RULE-002 | Business rule | Must | Exact hashtag result search shall match stored tag equality, not substring, prefix, full-text, or display-text-only matches. | User explicitly requested only posts with the exact hashtag. | Prompt | AC-001, AC-002 |
| RULE-003 | Business rule | Must | Profile searches shall not support chronology or popularity sorting in v1; unsupported profile sort parameters shall return validation errors rather than changing relevance ordering. | User excluded profile searches from chronology/popularity ordering. | Prompt | AC-004, EC-007 |
| RULE-004 | Business rule | Must | Search result endpoints shall not auto-save recents; only the explicit recent-search save endpoint records a search. | Prevents noisy recent searches from intermediate typing/autocomplete requests. | Q3 | AC-005, EC-008 |
| RULE-005 | Business rule | Must | Deleting a recent search shall hard delete that recent-search row rather than soft deleting it. | The user explicitly chose hard delete for private recent-search removal. | Review feedback | AC-007, AC-018 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-002, FR-003, RULE-002 | Given visible top-level posts tagged `sock`, `sockknitting`, and `SOCK`, when an authenticated user searches the exact hashtag `#sock`, then only posts whose normalized stored tag equals `sock` are returned. |
| AC-002 | BR-001, FR-002, RULE-002 | Given posts whose text visually contains `#sock` but whose indexed tags do not contain `sock`, when exact hashtag search is requested, then those posts are not returned. |
| AC-003 | BR-002, FR-004 | Given Craftsky profiles with matching handle, display name, and bio text, when `GET /v1/search/profiles?q=ali` is called, then matching Craftsky profiles are returned with profile summary fields and no non-Craftsky profiles. |
| AC-004 | BR-002, FR-004, FR-005, RULE-003 | Given profile matches of different strengths, when profile search results are returned, then exact/prefix handle matches rank before weaker handle, display-name, and bio matches; when chronology/popularity sort parameters are sent to profile search, then the endpoint returns a validation error. |
| AC-005 | BR-003, FR-013, FR-014, RULE-004 | Given a user commits a hashtag, profile, post, or project-filter search in the app, when the app explicitly calls `POST /v1/search/recent`, then the search is saved or refreshed in that user's recent-search list. |
| AC-006 | BR-003, FR-013, FR-014, FR-015 | Given a user has saved recent searches, when `GET /v1/search/recent` is called, then recent searches are returned newest-first with stable IDs, type metadata, display labels, and enough payload for the app to rerun the search. |
| AC-007 | BR-003, FR-013, FR-015, RULE-005 | Given a user deletes one of their recent searches, when `DELETE /v1/search/recent/{id}` succeeds, then the row is hard deleted and subsequent list responses for that user no longer include that search. |
| AC-008 | BR-004, FR-011, FR-012 | Given recent top-level project posts across knitting and crochet with hashtag tags, when top hashtags are requested for those craft types, then the response contains separate craft groups with counts for tags used in the 28-day window. |
| AC-009 | BR-004, FR-011, FR-012 | Given a requested craft type has no recent hashtag activity, when top hashtags are requested, then the response includes that requested craft group with an empty `items` list so the app can render stable craft sections. |
| AC-010 | BR-005, FR-007, FR-008 | Given indexed project posts with different craft types, project types, difficulties, colors, materials, design tags, and project tags, when project search filters are provided, then only projects matching the requested filter combination are returned. |
| AC-011 | BR-005, FR-007, FR-008 | Given multiple values for supported project filters, when project search is requested, then the endpoint applies OR semantics within repeated values for the same filter family, AND semantics across different filter families, and rejects unsupported filter values or fields according to validation rules. |
| AC-012 | BR-006, FR-002, FR-006 | Given matching hashtag or post search results with different creation/index times, when `sort=chronological` is requested, then results are ordered newest-first with deterministic tie-breakers and opaque cursors continue that order. |
| AC-013 | BR-006, FR-006, FR-007, FR-017, NFR-003 | Given matching post/project results with different engagement counts and ages, when `sort=popular` is requested, then results are ordered by the documented decayed popularity score with deterministic tie-breakers. |
| AC-014 | FR-001, NFR-001 | Given any `/v1/search/*` endpoint other than none explicitly public, when the request is missing auth or device ID, then it fails through the existing authenticated/device middleware and standard error envelope. |
| AC-015 | FR-001, FR-002, FR-016, NFR-001, NFR-002 | Given a search list endpoint receives valid `limit` and `cursor` parameters, when results span multiple pages, then it returns an `items` array and an opaque `cursor` only when more results are available; invalid cursors return `400 invalid_cursor`. |
| AC-016 | BR-001, FR-006, FR-009 | Given post or hashtag search returns regular and project posts, when the app decodes the response, then each item uses the same core post response contract as existing timeline/profile post list items, including project fields when present. |
| AC-017 | FR-010 | Given matching content or authors are actively hidden/taken down by moderation outputs, when any search endpoint is called, then those rows are filtered before result limiting and ranking. |
| AC-018 | FR-015, RULE-001, RULE-005 | Given two authenticated users have different recent searches, when either user lists or deletes recents, then they can only see/delete their own entries and cannot infer the other's recent-search contents. |
| AC-019 | FR-018, NFR-001 | Given malformed query parameters, unsupported sort values, unsupported filter fields, or invalid recent-search payloads, when the endpoint handles the request, then it returns a documented 400/422 standard error envelope rather than silently broadening the query. |
| AC-020 | NFR-002, NFR-004, NFR-005 | Given a representative seeded data set, when search endpoints are exercised in tests or local development, then they use bounded limits and indexed/local AppView data paths without per-result PDS network calls in the normal path; profile search can adopt PostgreSQL `pg_trgm` indexes when substring/similarity performance requires it without changing relevance-order semantics. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Hashtag input has a leading `#`, casing differences, or surrounding whitespace. | Normalize to the canonical stored tag value before equality matching. | FR-003, RULE-002 |
| EC-002 | Hashtag input is empty after trimming or only `#`. | Return validation error rather than all posts. | FR-003, NFR-002 |
| EC-003 | Profile query matches stale/missing identity-cache handles. | Search uses locally available indexed/cache data; it does not do broad network discovery. Missing stale candidates may be absent until cache/backfill updates. | FR-004, NFR-004 |
| EC-004 | Moderated post would otherwise be high-ranking by popularity. | Moderation filtering wins; hidden/takedown rows are absent before ranking. | FR-010 |
| EC-005 | Recent-search save duplicates an existing recent search for the same user and payload. | Refresh/update the existing entry rather than creating noisy duplicates, or otherwise de-duplicate according to the documented contract. | FR-014 |
| EC-006 | App sends unknown project filter key or invalid enum/value. | Return documented validation error; do not ignore if ignoring would broaden results. | FR-018 |
| EC-007 | App sends `sort=popular` or `sort=chronological` to profile search. | Return a validation error; profile results remain relevance-ordered only when no unsupported sort is requested. | RULE-003 |
| EC-008 | App calls search endpoints while typing without saving. | Results may be returned, but recent searches are not changed unless `POST /v1/search/recent` is called. | RULE-004 |
| EC-009 | Top hashtags requested with no craft filters. | Return all supported craft groups using the same grouped shape. | FR-011, FR-012 |
| EC-010 | Popularity sort sees equal scores. | Use deterministic tie-breakers such as created/indexed time and URI so pagination is stable. | FR-017, FR-016 |

## 15. Data / Persistence Impact

- New fields/tables:
  - A new AppView recent-search persistence table is expected, keyed by authenticated user DID and a server-generated recent-search ID.
  - The table should store search type, display label, normalized payload/filter JSON, and timestamps such as `created_at`/`updated_at`.
  - Recent-search delete is a hard delete; no `deleted_at` column is required for v1 recent-search removal.
- Changed fields:
  - No existing public record fields need to change.
  - Supporting search indexes, generated columns, or materialized text vectors may be added to existing AppView tables if needed.
- Migration required:
  - Yes, for recent-search persistence and any search-supporting indexes/materialized columns.
  - Migrations must not perform PDS/network calls.
- Backwards compatibility:
  - Existing `/v1/facets/*`, profile, timeline, post, and project endpoints must continue to work.
  - Additive `/v1/search/*` endpoints do not require a `/v2/` API bump.

## 16. UI / API / CLI Impact

- UI:
  - No Flutter UI implementation in this slice.
  - API contracts must support a future search page with typed tabs/sections, blank-state top hashtags, recent searches, hashtag-result pages, profile result rows, post result lists, and progressive project filters.
- API:
  - Add authenticated `/v1/search/*` endpoints for exact hashtag post results, profile search, general post search, project filter search, top hashtags, and recent-search list/save/delete.
  - Use camelCase JSON and existing AppView pagination/error conventions.
  - Existing `/v1/facets/*` autocomplete endpoints remain separate and unchanged unless shared helpers are refactored internally.
- CLI:
  - No CLI is required by the product behavior.
  - If search indexes or materialized data need backfill, a bounded operational backfill command may be planned by later implementation stages.
- Background jobs:
  - No new background job is required by the requirements.
  - Existing firehose indexers continue to maintain posts, projects, profiles, and interactions.

## 17. Security / Privacy / Permissions

- Authentication:
  - All `/v1/search/*` endpoints are authenticated and require `X-Craftsky-Device-Id`, matching current v1 conventions.
- Authorization:
  - Recent-search list/delete/save operations are scoped to the authenticated viewer DID.
  - Search results expose only public indexed content already visible through AppView surfaces and filtered by moderation policy.
- Sensitive data:
  - Recent searches are private-by-intent AppView data and must not be stored in PDS records or returned to other users.
  - Recent-search logs should avoid recording full sensitive query payloads at high log levels unless redacted/bounded.
- Abuse cases:
  - Query length, result limits, filter count, and pagination must be bounded to reduce scraping and expensive-query abuse.
  - No unauthenticated public search endpoint is introduced in this slice.

## 18. Observability

- Events:
  - None required for product analytics in this requirements slice.
- Logs:
  - Log endpoint errors with request ID/run ID and endpoint name.
  - Avoid logging full recent-search payloads or long free-text queries unless truncated/redacted.
- Metrics:
  - Recommended implementation metrics: request count/latency by endpoint, validation failures, result counts, and query errors.
- Alerts:
  - None required for this slice, but high error rates or latency on `/v1/search/*` would be useful operational signals later.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Search scope is broad for one AppView slice. | Requirements or tests may become too large, slowing implementation. | Keep Flutter UI, lexicon changes, semantic search, and recommendations out of scope; split implementation tasks later if needed. |
| RISK-002 | Popularity formula may not match user expectations. | Users may see stale or surprising popular results. | Centralize and document the v1 formula, test clear ranking examples, and keep endpoint shape independent of formula changes. |
| RISK-003 | Search queries may be slow on larger data sets. | Poor app responsiveness and database load. | Require bounded limits, validation, local indexed data paths, and migration/index review before implementation. |
| RISK-004 | Recent searches are private user behavior. | Storing or logging them incorrectly could violate privacy expectations. | Store only in AppView scoped by DID; do not write PDS records; avoid verbose logging of payloads. |
| RISK-005 | Top hashtags grouped by craft may omit relevant non-project posts without craft metadata. | Blank-search discovery may feel incomplete. | Start from materialized craft/project data and document the limitation; revisit author-craft inference or richer tagging in a later slice. |
| RISK-006 | Exact hashtag matching depends on indexer tag normalization. | Search could miss posts if tags were not materialized from all intended project/text fields. | Test text facets and project tag materialization paths; document any known gaps. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | “Semi-fuzzy” profile search means case-insensitive exact/prefix/substring style matching, not typo-tolerant edit-distance matching. | Requirements would need to add trigram/typo-tolerant behavior and likely different indexes. |
| ASM-002 | Top hashtags grouped by craft can initially use materialized project craft type / project hashtag data rather than inferring craft type for every regular post. Future regular post craft-type properties are expected to broaden this source later. | The top-hashtag endpoint may need broader craft attribution rules once regular posts carry craft type. |
| ASM-003 | The future Flutter UI can explicitly save committed recent searches and does not need AppView to infer commitment from every search request. | Recent-search API would need auto-save or a different event contract. |
| ASM-004 | Existing `PostResponse` and profile-summary response patterns are acceptable for search result items. | Search may require new response DTOs or adapter fields for the app. |
| ASM-005 | 28 days is a suitable v1 recency window for top hashtags because existing hashtag suggestions already use 28-day counts. | The top-hashtag endpoint would need a different default/window parameter. |
| ASM-006 | Search remains authenticated in v1. | Public search would require separate rate limiting, privacy, and caching requirements. |

## 21. Open Questions

- [ ] Non-blocking: Pick exact v1 popularity formula constants during implementation planning. The implementer should choose a reasonable centralized default that can be tweaked later without changing endpoint names or response shapes.

## 22. Review Status

Status: Reviewed
Risk level: Medium
Review recommended: Yes
Reviewer: Douglas Todd
Date: 2026-06-19
Notes: Initial review feedback addressed. Semi-fuzzy profile search, project-backed top hashtag grouping, and 28-day window were confirmed. Future regular-post craft-type properties are expected to broaden top-hashtag craft grouping. PostgreSQL `pg_trgm` is the preferred scalable profile-search strategy when needed. Recent-search delete is hard delete. Popularity constants remain a non-blocking implementation-planning choice.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-19-appview-search-foundation/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - Business: `BR-001` through `BR-006`
  - Functional: `FR-001` through `FR-017`
  - Non-functional: `NFR-001`, `NFR-002`, `NFR-005`
  - Rules: `RULE-001` through `RULE-005`
- Suggested test levels:
  - AppView API handler tests for auth/device enforcement, validation, response shapes, and error envelopes.
  - AppView store/integration tests for exact hashtag matching, profile ranking, post/project filters, top hashtag grouping, recent-search persistence, moderation filtering, pagination, and popularity ordering.
  - Regression tests ensuring existing `/v1/facets/*`, timeline/profile post response contracts, and moderation filters continue to behave.
- Blocking open questions: None.
