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

Answer: Sort profiles the user follows first, then prefer matches against handle, then display name, then bio/description.

Decision / implication: Profile ranking should prioritize profiles the viewer follows first, then order by textual relevance: exact/prefix handle matches strongest, then handle substring, then display-name matches, then profile-description matches.

### Q5: What does popularity mean for post/project searches?

Answer: Number of interactions with decay based on age.

Decision / implication: Popularity sorting must use active likes, replies, and reposts/comments as engagement inputs, adjusted by a deterministic recency decay. The v1 formula must be documented in implementation/test specs and centralized so it can evolve later without changing endpoint names.

### Q6: What endpoint family should search use?

Answer: Dedicated `/v1/search/*` endpoints.

Decision / implication: Existing `/v1/facets/*` endpoints remain autocomplete/resolve surfaces. Search-result and search-state APIs should live under `/v1/search/*`.

### Q7: Should exact hashtag search include replies/comments or parse raw post text as a fallback?

Answer: No; exact hashtag search is top-level posts/projects only and matches indexed/materialized tags only.

Decision / implication: Replies/comments are excluded from exact hashtag search in v1, and visual hashtag text that was not materialized into indexed tags is treated as an indexing/composer issue rather than a search fallback.

### Q8: How should top hashtags grouped by craft type be sourced in v1?

Answer: Use project posts only for craft-grouped counts, return requested empty craft groups, do not add an uncategorized group, and count distinct posts.

Decision / implication: Top hashtag groups use project craft metadata and project/post tag materialization. Regular non-project posts are not included until future regular-post craft-type properties exist. If a requested craft has no recent tags, it still appears with `items: []`. Counts represent distinct posts/projects using the tag, not repeated occurrences.

### Q9: Can project search be used as a browse-all project endpoint?

Answer: Yes.

Decision / implication: `GET /v1/search/projects` may be called with no filters or keyword query and returns all visible top-level projects, default sorted chronologically. `sort=popular` is valid even when no filters are present.

### Q10: Can general post search be used with an empty `q`?

Answer: No.

Decision / implication: `GET /v1/search/posts` requires a non-empty query; it is not a global all-posts discovery feed in v1.

### Q11: What fields should keyword post/project search cover?

Answer: Search post text plus core project common fields.

Decision / implication: General post search and project keyword search use post text plus materialized project common fields such as project title, pattern name, material text, project tags, and design tags. Craft-specific detail fields remain primarily filters in v1.

### Q12: What popularity formula and sort tie-breakers should v1 use?

Answer: Use `weightedEngagement = likes + (2 * replies) + (3 * reposts)`, then `popularityScore = weightedEngagement / pow(1 + ageHours / 72, 1.5)`, sorted by score descending, then created time descending, then URI descending.

Decision / implication: Popularity is deterministic and centralized. It counts only active likes, active reposts, and visible descendant replies/comments. The score is internal; API responses return normal engagement counts, not `popularityScore`.

### Q13: What chronological ordering should search use?

Answer: Use authored creation time, not indexing time.

Decision / implication: Chronological search sorting is `created_at DESC, uri DESC`; default search sort is chronological unless the app explicitly requests `sort=popular`.

### Q14: What recent-search lifecycle rules should v1 use?

Answer: Use opaque server-generated IDs, store both normalized payload and display label, de-duplicate by normalized type/payload per user, move duplicates to the top, keep the latest 50 entries per user, and make delete idempotent hard delete.

Decision / implication: Recent search persistence supports user-friendly display labels and deterministic reruns/de-duplication. Saving an existing search refreshes `updatedAt` while preserving the existing stored display label for that normalized search; older entries beyond 50 are pruned. `DELETE` returns success even if the row is already gone or not owned by the caller, avoiding existence leaks.

### Q15: What matching/indexing strategies should v1 prefer?

Answer: Use PostgreSQL full-text search for post/project keyword search and `pg_trgm` as the preferred scalable profile-search strategy when needed.

Decision / implication: Post/project keyword search is document-like and should be backed by PostgreSQL FTS with deterministic tie-breakers. Profile search keeps explicit followed-first and relevance ranking, using `pg_trgm` only as a scalable local indexed strategy without making ranking opaque.

### Q16: What other result-safety and filter rules apply?

Answer: Apply existing moderation only for profile safety in v1; block/mute filtering is out of scope unless already indexed/enforced. Project filter values match case-insensitively. Exact hashtag responses identify the normalized lowercase canonical tag without `#`.

Decision / implication: Search does not invent block/mute behavior in this slice. App-provided filter casing should not affect project matches, but returned post/project data preserves stored display values. Hashtag result metadata is canonicalized even if recent-search display labels preserve user-entered casing.

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
| FR-002 | Functional | Must | The system shall expose a path-based exact hashtag results endpoint, `GET /v1/search/hashtags/{tag}/posts`, accepting `sort`, `limit`, and `cursor`. | Provides a clear API for hashtag facet taps and typed hashtag searches. | Prompt, Q6, Q7 | AC-001, AC-002, AC-012, AC-015, AC-021 |
| FR-003 | Functional | Must | The system shall normalize hashtag inputs by trimming whitespace, removing one leading `#` if present, and matching case-insensitively against stored normalized tag values. | Users may arrive from typed text or facet taps, while storage is normalized. | Codebase, Prompt | AC-001, EC-001, EC-002 |
| FR-004 | Functional | Must | The system shall expose profile search, such as `GET /v1/search/profiles?q=...`, searching Craftsky profiles by cached handle, Bluesky display name, and Bluesky description/bio. | Supports semi-fuzzy profile discovery with existing indexed profile data. | Prompt, Q4, Codebase | AC-003, AC-004, EC-003 |
| FR-005 | Functional | Must | Profile search shall order followed profiles before profiles the viewer does not follow, then order within each followed/non-followed group by textual relevance: exact/prefix handle matches first, then handle substring, display-name matches, and description/bio matches, with stable deterministic tie-breakers. | Matches the user's preferred profile ordering, adds the reviewed followed-first behavior, and makes results predictable. | Q4, Review feedback | AC-004 |
| FR-006 | Functional | Must | The system shall expose general post search, such as `GET /v1/search/posts?q=...`, over visible top-level posts, including project posts, with chronological and popularity sort support; `q` is required and must be non-empty. | Provides broad search that includes projects without turning post search into a global discovery feed. | Prompt, Q10 | AC-012, AC-013, AC-016, AC-022 |
| FR-007 | Functional | Must | The system shall expose project filter search as a `GET` endpoint with query parameters, such as `GET /v1/search/projects`, filtering visible top-level project posts by supported materialized project fields. | Provides AppView support for the app's future progressive project-filter UI while keeping search read operations URL-addressable. | Prompt, Codebase, Q9 | AC-010, AC-011, AC-013, AC-023 |
| FR-008 | Functional | Must | Project filter search shall support at least craft type, craft-specific project type, pattern difficulty, color, material text/tag, design tag, project tag, and optional keyword query where data is materialized; v1 filter semantics shall be OR within repeated values for the same filter family and AND across different filter families, with case-insensitive matching for user-facing string filter values. | Captures the filter fields called out by the user and current project materialization. | Prompt, Codebase, Q11, Q16 | AC-010, AC-011 |
| FR-009 | Functional | Must | Search endpoints returning posts/projects shall build the same core post response shape used by existing timeline/profile post endpoints, including author data, project data when present, and engagement summary fields, but not the internal decayed popularity score. | Flutter should consume consistent post cards across timeline, profile, and search surfaces while ranking internals remain tunable. | Codebase, Discovery, Q12 | AC-016, AC-024 |
| FR-010 | Functional | Must | Search endpoints shall apply existing post and account moderation visibility rules before limiting, ranking, or returning results. | Hidden/takedown content must not reappear through search. | Codebase | AC-017, EC-004 |
| FR-011 | Functional | Must | The system shall expose grouped top hashtags, such as `GET /v1/search/hashtags/top?craftTypes=...`, returning craft-grouped tags and recent counts. | Supports blank search discovery grouped by craft interest. | Prompt | AC-008, AC-009 |
| FR-012 | Functional | Must | Top hashtag results shall use a bounded recent window, defaulting to 28 days for v1, count distinct project posts using each tag, and shall only include project-post hashtags associated with the requested craft groups or all supported craft groups when no craft filter is provided. | Matches existing 28-day hashtag count behavior and uses reliable v1 craft metadata. | Prompt, Codebase, Q8 | AC-008, AC-009, AC-025 |
| FR-013 | Functional | Must | The system shall expose recent-search list, explicit save, and idempotent hard-delete endpoints, such as `GET /v1/search/recent`, `POST /v1/search/recent`, and `DELETE /v1/search/recent/{id}`. | Recent searches need explicit user-controlled persistence and private deletion. | Prompt, Q2, Q3, Q14 | AC-005, AC-006, AC-007 |
| FR-014 | Functional | Must | Recent-search saves shall accept a bounded typed payload representing committed searches, including at least hashtag search, profile query, post query, and project filter search; saves shall store both normalized payload and display label. | The app must be able to save the types of searches introduced in this slice while preserving display labels. | Prompt, Q3, Q14 | AC-005, AC-006, EC-005 |
| FR-015 | Functional | Must | Recent-search list and delete operations shall be scoped to the authenticated viewer DID. | Search history is private per user. | Q2, AGENTS privacy rule | AC-006, AC-007, AC-018 |
| FR-016 | Functional | Must | Search list endpoints shall support `limit` and opaque `cursor` pagination following existing AppView v1 conventions. | Consistent pagination is required for app consumption and API architecture compliance. | Codebase | AC-015 |
| FR-017 | Functional | Must | Post/project popularity sort shall use the centralized v1 formula `score = (likes + (2 * replies) + (3 * reposts)) / pow(1 + ageHours / 72, 1.5)`, counting active likes, active reposts, and visible descendant replies/comments only. | Defines popularity in terms discussed by the user while preventing a pure all-time ranking. | Q5, Q12, Q13, Codebase | AC-013, AC-026 |
| FR-018 | Functional | Should | Search query parsing should reject or ignore unsupported filter fields with clear validation behavior instead of silently broadening results. | Avoids app bugs causing misleading or overly broad searches. | Discovery | AC-019, EC-006 |
| FR-019 | Functional | Must | General post search and project keyword search shall search post text plus core materialized project common fields: project title, pattern name, material text, project tags, and design tags. | Users expect keyword search to find project metadata such as titles and materials without searching every raw craft-specific field. | Q11, Q12 | AC-022 |
| FR-020 | Functional | Must | Project search shall allow no filters or keyword query and return all visible top-level projects, defaulting to chronological order unless `sort=popular` is explicitly requested. | Supports the future progressive project browse/filter UI. | Q9, Q13 | AC-023 |
| FR-021 | Functional | Must | Recent-search saves shall de-duplicate by authenticated user, search type, and normalized payload, update `updatedAt`/move duplicates to the top while preserving the existing stored display label, keep only the latest 50 entries per user, and prune older entries. | Keeps recent searches useful and bounded. | Q14 | AC-027 |
| FR-022 | Functional | Must | Recent-search IDs returned to clients shall be opaque server-generated identifiers, not derived from query payloads or exposed composite keys. | Avoids leaking query payloads in URLs/logs and keeps the API flexible. | Q14 | AC-006, AC-007 |
| NFR-001 | Non-functional | Must | Search APIs shall use `/v1/*` camelCase JSON, standard AppView error envelopes, and authenticated session/device middleware. | Required by existing AppView API conventions. | Codebase | AC-014, AC-015, AC-019 |
| NFR-002 | Non-functional | Must | Search queries shall be bounded by default and maximum limits, bounded query string lengths, and indexed access paths appropriate for expected AppView growth. V1 defaults shall use `limit=25` and `maxLimit=100` for paginated result lists, `topLimit=10` and `maxTopLimit=50` per top-hashtag craft group, a maximum free-text query length of 256 Unicode scalar values after trimming, a maximum hashtag path value length of 128 after normalization, a maximum recent-search display label length of 120, a maximum recent-search normalized payload size of 4096 bytes, at most 10 repeated values per project filter family, and at most 50 total project-filter values per request. | Search can become expensive without guardrails. | Discovery | AC-015, AC-020 |
| NFR-003 | Non-functional | Should | Popularity scoring should be centralized and documented so future scoring changes do not require changing endpoint names or response shapes. | The formula is product-sensitive and likely to evolve. | Q5, Q14 | AC-013 |
| NFR-004 | Non-functional | Should | Search endpoints should avoid per-result network calls to PDS or identity services in the normal result path. | Search should be responsive and rely on indexed AppView data. | Codebase | AC-020 |
| NFR-005 | Non-functional | Should | Profile search should use PostgreSQL `pg_trgm` as the preferred scalable local indexed strategy once data size requires indexed substring or similarity matching, while preserving explicit deterministic v1 ranking. | Keeps profile search scalable without making ranking opaque. | Review feedback | AC-004, AC-020 |
| NFR-006 | Non-functional | Should | Post/project keyword search should use PostgreSQL full-text search as the intended v1 local search strategy, with deterministic relevance and sort tie-breakers. | Post and project keyword search is document-like and should not rely on slow raw substring scans. | Q15 | AC-020, AC-022 |
| RULE-001 | Business rule | Must | Recent searches are private AppView data and must not be written to the PDS or exposed to other users. | Search history is private-by-intent behavior. | Q2, AGENTS privacy rule | AC-018 |
| RULE-002 | Business rule | Must | Exact hashtag result search shall match stored tag equality, not substring, prefix, full-text, or display-text-only matches. | User explicitly requested only posts with the exact hashtag. | Prompt | AC-001, AC-002 |
| RULE-003 | Business rule | Must | Profile searches shall not support chronology or popularity sorting in v1; unsupported profile sort parameters shall return validation errors rather than changing relevance ordering. | User excluded profile searches from chronology/popularity ordering. | Prompt | AC-004, EC-007 |
| RULE-004 | Business rule | Must | Search result endpoints shall not auto-save recents; only the explicit recent-search save endpoint records a search. | Prevents noisy recent searches from intermediate typing/autocomplete requests. | Q3 | AC-005, EC-008 |
| RULE-005 | Business rule | Must | Deleting a recent search shall hard delete that recent-search row rather than soft deleting it. | The user explicitly chose hard delete for private recent-search removal. | Review feedback | AC-007, AC-018 |
| RULE-006 | Business rule | Must | Search result list endpoints shall default to chronological order (`created_at DESC, uri DESC`) unless the endpoint has relevance-only semantics or the app explicitly requests `sort=popular`. | Craftsky's product principles favor chronological ordering by default. | Q13, Q15 | AC-012, AC-023 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-002, FR-003, RULE-002 | Given visible top-level posts tagged `sock`, `sockknitting`, and `SOCK`, when an authenticated user searches the exact hashtag `#sock`, then only posts whose normalized stored tag equals `sock` are returned. |
| AC-002 | BR-001, FR-002, RULE-002 | Given posts whose text visually contains `#sock` but whose indexed tags do not contain `sock`, when exact hashtag search is requested, then those posts are not returned. |
| AC-003 | BR-002, FR-004 | Given Craftsky profiles with matching handle, display name, and bio text, when `GET /v1/search/profiles?q=ali` is called, then matching Craftsky profiles are returned with profile summary fields and no non-Craftsky profiles. |
| AC-004 | BR-002, FR-004, FR-005, RULE-003 | Given matching profiles include accounts the viewer follows and does not follow, when profile search results are returned, then followed profiles rank before non-followed profiles; within each followed/non-followed group, exact/prefix handle matches rank before weaker handle, display-name, and bio matches; when chronology/popularity sort parameters are sent to profile search, then the endpoint returns a validation error. |
| AC-005 | BR-003, FR-013, FR-014, RULE-004 | Given a user commits a hashtag, profile, post, or project-filter search in the app, when the app explicitly calls `POST /v1/search/recent`, then the search is saved or refreshed in that user's recent-search list. |
| AC-006 | BR-003, FR-013, FR-014, FR-015 | Given a user has saved recent searches, when `GET /v1/search/recent` is called, then recent searches are returned newest-first with stable IDs, type metadata, display labels, and enough payload for the app to rerun the search. |
| AC-007 | BR-003, FR-013, FR-015, RULE-005 | Given a user deletes one of their recent searches, when `DELETE /v1/search/recent/{id}` succeeds, then the row is hard deleted and subsequent list responses for that user no longer include that search. |
| AC-008 | BR-004, FR-011, FR-012 | Given recent top-level project posts across knitting and crochet with hashtag tags, when top hashtags are requested for those craft types, then the response contains separate craft groups with counts for tags used in the 28-day window. |
| AC-009 | BR-004, FR-011, FR-012 | Given a requested craft type has no recent hashtag activity, when top hashtags are requested, then the response includes that requested craft group with an empty `items` list so the app can render stable craft sections. |
| AC-010 | BR-005, FR-007, FR-008 | Given indexed project posts with different craft types, project types, difficulties, colors, materials, design tags, and project tags, when project search filters are provided, then only projects matching the requested filter combination are returned. |
| AC-011 | BR-005, FR-007, FR-008 | Given multiple values for supported project filters, when project search is requested, then the endpoint applies OR semantics within repeated values for the same filter family, AND semantics across different filter families, and rejects unsupported filter values or fields according to validation rules. |
| AC-012 | BR-006, FR-002, FR-006 | Given matching hashtag or post search results with different creation times, when `sort=chronological` is requested, then results are ordered newest-first by `created_at DESC, uri DESC` and opaque cursors continue that order. |
| AC-013 | BR-006, FR-006, FR-007, FR-017, NFR-003 | Given matching post/project results with different engagement counts and ages, when `sort=popular` is requested, then results are ordered by the documented decayed popularity score with deterministic tie-breakers. |
| AC-014 | FR-001, NFR-001 | Given any `/v1/search/*` endpoint other than none explicitly public, when the request is missing auth or device ID, then it fails through the existing authenticated/device middleware and standard error envelope. |
| AC-015 | FR-001, FR-002, FR-016, NFR-001, NFR-002 | Given a search list endpoint receives valid `limit` and `cursor` parameters, when results span multiple pages, then it returns an `items` array and an opaque `cursor` only when more results are available; invalid cursors return `400 invalid_cursor`. |
| AC-016 | BR-001, FR-006, FR-009 | Given post or hashtag search returns regular and project posts, when the app decodes the response, then each item uses the same core post response contract as existing timeline/profile post list items, including project fields when present. |
| AC-017 | FR-010 | Given matching content or authors are actively hidden/taken down by moderation outputs, when any search endpoint is called, then those rows are filtered before result limiting and ranking. |
| AC-018 | FR-015, RULE-001, RULE-005 | Given two authenticated users have different recent searches, when either user lists or deletes recents, then they can only see/delete their own entries and cannot infer the other's recent-search contents. |
| AC-019 | FR-018, NFR-001 | Given malformed query parameters, unsupported sort values, unsupported filter fields, or invalid recent-search payloads, when the endpoint handles the request, then it returns a documented 400/422 standard error envelope rather than silently broadening the query. |
| AC-020 | NFR-002, NFR-004, NFR-005 | Given a representative seeded data set, when search endpoints are exercised in tests or local development, then they use bounded limits and indexed/local AppView data paths without per-result PDS network calls in the normal path; profile search can adopt PostgreSQL `pg_trgm` indexes when substring/similarity performance requires it without changing relevance-order semantics. |
| AC-021 | FR-002, FR-003 | Given `GET /v1/search/hashtags/SockKAL/posts`, when the endpoint succeeds, then response metadata identifies the searched hashtag as the normalized canonical `sockkal` without a leading `#`. |
| AC-022 | FR-006, FR-019, NFR-006 | Given posts/projects with matches in post text, project title, pattern name, material text, project tags, or design tags, when general post or project keyword search is requested with a non-empty `q`, then matching visible top-level records are found through the documented PostgreSQL full-text/local indexed strategy and deterministic tie-breakers. |
| AC-023 | FR-007, FR-020, RULE-006 | Given no filters and no keyword query are passed to `GET /v1/search/projects`, when the endpoint is called, then all visible top-level projects are returned in chronological order by default; when `sort=popular` is explicitly passed, the same browse-all project set is popularity-ordered. |
| AC-024 | FR-009 | Given `sort=popular` orders search results by decayed score, when the response is encoded, then items include engagement counts but do not expose `popularityScore`. |
| AC-025 | FR-011, FR-012 | Given one project repeats the same hashtag across multiple materialized tag sources, when top hashtags are counted, then that tag contributes one distinct-project count for that craft group. |
| AC-026 | FR-017 | Given active likes/reposts and visible replies differ from deleted or hidden/takedown interactions, when popularity is calculated, then only active likes, active reposts, and visible descendant replies/comments contribute; ties sort by `created_at DESC, uri DESC`. |
| AC-027 | FR-021, FR-014 | Given a user saves the same normalized search more than once with a different submitted display label and later exceeds 50 saved searches, when recent searches are listed, then the duplicate appears once at the top with refreshed `updatedAt`, the existing stored display label unchanged, and older entries beyond the latest 50 pruned. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Hashtag input has a leading `#`, casing differences, or surrounding whitespace. | Normalize to the canonical stored tag value before equality matching. | FR-003, RULE-002 |
| EC-002 | Hashtag input is empty after trimming or only `#`. | Return validation error rather than all posts. | FR-003, NFR-002 |
| EC-003 | Profile query matches stale/missing identity-cache handles. | Search uses locally available indexed/cache data; it does not do broad network discovery. Missing stale candidates may be absent until cache/backfill updates. | FR-004, NFR-004 |
| EC-004 | Moderated post would otherwise be high-ranking by popularity. | Moderation filtering wins; hidden/takedown rows are absent before ranking. | FR-010 |
| EC-005 | Recent-search save duplicates an existing recent search for the same user and payload. | Refresh `updatedAt` and move the existing entry to the top without creating a duplicate and without replacing the existing stored display label. | FR-014, FR-021 |
| EC-006 | App sends unknown project filter key or invalid enum/value. | Return documented validation error; do not ignore if ignoring would broaden results. | FR-018 |
| EC-007 | App sends `sort=popular` or `sort=chronological` to profile search. | Return a validation error; profile results remain relevance-ordered only when no unsupported sort is requested. | RULE-003 |
| EC-008 | App calls search endpoints while typing without saving. | Results may be returned, but recent searches are not changed unless `POST /v1/search/recent` is called. | RULE-004 |
| EC-009 | Top hashtags requested with no craft filters. | Return all supported craft groups using the same grouped shape. | FR-011, FR-012 |
| EC-010 | Popularity sort sees equal scores. | Use deterministic tie-breakers `created_at DESC, uri DESC` so pagination is stable. | FR-017, FR-016 |
| EC-011 | `GET /v1/search/posts` receives missing, empty, or whitespace-only `q`. | Return validation error rather than a global all-posts feed. | FR-006 |
| EC-012 | Project search receives no filters and no query. | Return all visible top-level projects using default chronological order unless `sort=popular` is explicitly requested. | FR-020, RULE-006 |
| EC-013 | Recent-search delete targets an already-deleted, nonexistent, or not-owned opaque ID. | Return idempotent success without revealing whether another user's row exists. | FR-013, FR-015, RULE-005 |
| EC-014 | Project filter values differ only by case from stored normalized values. | Match case-insensitively while preserving stored/display casing in returned post/project data. | FR-008 |

## 15. Data / Persistence Impact

- New fields/tables:
  - A new AppView recent-search persistence table is expected, keyed by authenticated user DID and a server-generated recent-search ID.
  - The table should store an opaque server-generated ID, search type, display label, normalized payload/filter JSON, a normalized payload hash or equivalent de-duplication key, and timestamps such as `created_at`/`updated_at`.
  - The recent-search table should enforce or emulate uniqueness on `(viewer_did, search_type, normalized_payload_hash)` and support newest-first listing with an index such as `(viewer_did, updated_at DESC, id DESC)`.
  - Recent-search delete is a hard delete; no `deleted_at` column is required for v1 recent-search removal.
  - Recent-search persistence should enforce or implement per-user de-duplication by search type and normalized payload, and pruning to the latest 50 entries.
- Changed fields:
  - No existing public record fields need to change.
  - Supporting search indexes, generated columns, or materialized text vectors may be added to existing AppView tables if needed.
  - Post/project keyword search should plan a PostgreSQL full-text vector over post text plus materialized project common fields and a GIN index or equivalent local indexed path.
  - Project filter search should plan indexed access for common filter fields such as craft type, project type, difficulty, colors, materials, design tags, and project tags, using existing normalized/materialized columns where possible.
  - Profile search may add `pg_trgm` indexes for cached handles, display names, and descriptions if implementation uses trigram-backed substring/similarity matching in v1.
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
  - Paginated post/project result endpoints return at least `{ "hashtag": "sock", "items": [<PostResponse>], "cursor": "opaque" }` for hashtag search where `hashtag` is present only on hashtag-result responses, and `{ "items": [<PostResponse>], "cursor": "opaque" }` for general post and project search.
  - Profile search returns `{ "items": [<profile summary>], "cursor": "opaque" }` using the existing profile-summary field conventions; profile responses do not include chronology/popularity fields.
  - Top hashtags return `{ "groups": [{ "craftType": "knitting", "items": [{ "tag": "sock", "count": 12 }] }] }`, with requested empty groups represented as `items: []`.
  - Recent-search list returns `{ "items": [{ "id": "opaque", "type": "hashtag|profile|post|project", "displayLabel": "#sock", "payload": { ... }, "updatedAt": "RFC3339" }] }`; save returns the saved/refreshed item; delete returns success with no cross-user existence disclosure.
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
- Blocks/mutes:
  - Block/mute filtering is out of scope unless such state is already indexed and enforced elsewhere; v1 search applies existing moderation visibility rules.
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
| RISK-002 | Popularity formula may not match user expectations. | Users may see stale or surprising popular results. | Centralize and document the v1 formula, test clear ranking examples, do not expose the score publicly, and keep endpoint shape independent of formula changes. |
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
| ASM-007 | Existing moderation visibility rules are the only safety filters needed for v1 search; block/mute search filtering can wait until that state is indexed/enforced elsewhere. | Search requirements would need to add viewer-specific block/mute exclusions. |

## 21. Open Questions

None.

## 22. Review Status

Status: Reviewed
Risk level: Medium
Review recommended: Yes
Reviewer: Douglas Todd
Date: 2026-06-19
Notes: Review and grilling feedback addressed. Semi-fuzzy profile search, project-backed top hashtag grouping, and 28-day window were confirmed. Future regular-post craft-type properties are expected to broaden top-hashtag craft grouping. PostgreSQL `pg_trgm` is the preferred scalable profile-search strategy when needed. Post/project keyword search should use PostgreSQL full-text search. Recent searches use opaque IDs, display labels plus normalized payloads, de-duplication, hard delete, idempotent delete, and latest-50 pruning. Profile search sorts followed profiles first, then textual relevance within followed/non-followed groups. Popularity uses the documented weighted engagement plus age-decay formula and remains internal to ordering.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-19-appview-search-foundation/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - Business: `BR-001` through `BR-006`
  - Functional: `FR-001` through `FR-022`
  - Non-functional: `NFR-001`, `NFR-002`, `NFR-005`, `NFR-006`
  - Rules: `RULE-001` through `RULE-006`
- Suggested test levels:
  - AppView API handler tests for auth/device enforcement, validation, response shapes, and error envelopes.
  - AppView store/integration tests for exact hashtag matching, profile ranking, post/project filters, top hashtag grouping, recent-search persistence, moderation filtering, pagination, and popularity ordering.
  - Regression tests ensuring existing `/v1/facets/*`, timeline/profile post response contracts, and moderation filters continue to behave.
- Blocking open questions: None.
