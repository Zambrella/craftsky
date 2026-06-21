# Requirements: Search Refinements Before UI Slice

## 1. Initial Request

Plan a non-UI refinement slice for Craftsky search before the rendered search UI is built. The desired future app behavior includes a blank search page with recent searches and top/trending hashtags by craft type, a way to view/manage all recent searches, typeahead suggestions for profiles and hashtags, submitted search results split across posts/projects/profiles/hashtags tabs, exact hashtag result screens, and a separate Projects bottom-nav surface where project browsing/filtering happens by craft type with chronological/popular sorting. The work may touch both the AppView and Flutter data/logic layers, but should not implement UI-specific behavior in this slice.

## 2. Current Codebase Findings

- Relevant files:
  - Prior search requirements/contracts: `docs/changes/2026-06-19-appview-search-foundation/01-requirements.md`, `docs/changes/2026-06-20-flutter-search-data-layer/01-requirements.md`.
  - AppView route registration: `appview/internal/routes/routes.go`.
  - AppView search handlers/store/contracts: `appview/internal/api/search.go`, `search_request.go`, `search_response.go`, `search_store.go`, `search_recent_store.go`, `search_cursor.go`, `search_ranking.go`.
  - AppView facet suggestions: `appview/internal/api/facet.go`, `facet_store.go`, `facet_request.go`, `facet_response.go`.
  - AppView project materialization and project API helpers: `appview/internal/api/post_project.go`, `appview/internal/index/craftsky_post.go`, `appview/migrations/000016_project_posts.up.sql`, `appview/migrations/000019_search_foundation.up.sql`.
  - Flutter search data layer: `app/lib/search/data/*`, `app/lib/search/models/*`, `app/lib/search/providers/*`, `app/lib/search/pages/search_page.dart`.
  - Flutter facet suggestion data layer: `app/lib/shared/rich_text/data/appview_facet_suggestion_repository.dart`, `app/lib/shared/rich_text/data/facet_suggestion_repository.dart`, `app/lib/shared/rich_text/providers/facet_suggestion_providers.dart`.
  - Flutter project discovery data layer: `app/lib/projects/data/project_api_client.dart`, `project_repository.dart`, `app/lib/projects/providers/project_feed_provider.dart`, `app/lib/projects/pages/projects_page.dart`.
  - Flutter navigation: `app/lib/router/router.dart`, `app/lib/router/app_shell.dart`.
- Existing patterns:
  - Flutter reads come through AppView JSON/HTTP using shared authenticated `Dio`; the app does not read craft data from PDS directly.
  - `/v1/*` AppView APIs use authenticated session + `X-Craftsky-Device-Id`, camelCase JSON, standard error envelopes, and opaque cursors for paginated lists.
  - Search result endpoints and recent-search mutations are explicit; result fetching does not automatically save recents.
  - Existing Flutter feature data layers use API client + repository + Riverpod provider seams with fake repositories in tests.
  - Existing facet autocomplete repositories are under shared rich-text code and currently call `/v1/facets/mentions`, `/v1/facets/mentions/resolve`, and `/v1/facets/hashtags`.
- Current behavior:
  - AppView has `GET /v1/search/hashtags/{tag}/posts`, `/v1/search/profiles`, `/v1/search/posts`, `/v1/search/projects`, `/v1/search/hashtags/top`, `/v1/search/recent`, `POST /v1/search/recent`, `DELETE /v1/search/recent/{id}`, and `GET /v1/projects`.
  - AppView exact hashtag result search already returns top-level posts, including project posts, matching a normalized exact tag.
  - AppView profile search and facet mention suggestions both prioritize followed profiles, but the ranking/query logic is implemented separately.
  - AppView has no paginated committed hashtag-query result endpoint for a future Hashtags search tab.
  - Flutter has non-UI search clients/repositories/providers for posts, projects, profiles, exact hashtag posts, top hashtags, and recents.
  - Flutter has a separate project feed data layer for `/v1/projects`, but `ProjectsPage` is a stub.
  - Flutter `SearchPage` is a stub and does not yet consume recents, top hashtags, suggestions, exact hashtag results, or result tabs.
  - The app shell already includes a Projects bottom-nav branch and a Search bottom-nav branch.
- Constraints discovered:
  - No lexicon change is needed for this slice.
  - Recent/saved searches are private user behavior and should remain AppView Postgres state, not PDS records.
  - `/v1/facets/*` behavior should not be broken for project composer/profile rich-text autocomplete while introducing search suggestions.
  - Project records use full craft tokens such as `social.craftsky.feed.defs#knitting`, while some current search/project/top-hashtag code and tests use bare values such as `knitting`; this must be normalized before the UI depends on it.
  - Project browsing/filtering should remain distinct from committed full-text search even if AppView shares implementation helpers internally.
- Test/build commands discovered:
  - AppView tests: `just test` from repo root after `just dev-d`, or focused `cd appview && go test ./internal/api ./internal/routes -count=1`.
  - Flutter code generation: `cd app && dart run build_runner build --delete-conflicting-outputs`.
  - Flutter focused tests: `cd app && flutter test test/search test/shared/rich_text test/projects`.
  - Flutter broader checks: `cd app && flutter analyze` and `cd app && flutter test`.

## 3. Clarifying Questions And Decisions

### Q1: For the “view and manage all their saved searches” requirement, should this slice create a distinct saved-search feature, or should it extend/rename the existing server-backed recent-search history management?

Answer: Saved/recent are one and the same.

Decision / implication: This slice shall not introduce a second saved-search persistence model. Existing AppView-backed recent searches remain the single managed search-history surface; future UI can present them as recent/saved searches as product copy requires.

### Q2: Which overall API/data-layer direction should the requirements use?

Answer: Option A — shared suggestion core, unified typeahead contract, separate paginated result APIs, and `/v1/projects` for project browse/filtering.

Decision / implication: Requirements should plan a shared suggestion API/core, keep result-search and project-browse surfaces distinct, and preserve compatibility for existing facet autocomplete while making future search UI data straightforward.

### Q3: Should submitted-search Posts and Projects tabs be disjoint?

Answer: Yes — keep them disjoint.

Decision / implication: Submitted text-search Posts results should exclude project posts, and submitted text-search Projects results should contain project posts. Exact hashtag results remain a combined feed of matching regular posts and projects.

### Q4: What profiles are eligible for profile suggestions/search?

Answer: Craftsky profiles only for v1.

Decision / implication: Profile suggestions and profile search results should use indexed Craftsky profiles, matching the existing facet suggestion posture. External atproto account fallback is future work.

### Q5: How should exact hashtag selections behave?

Answer: Selecting a hashtag opens one combined exact-hashtag feed of top-level matching regular posts and projects.

Decision / implication: Exact hashtag screens should not split into Posts/Projects tabs in this slice and should continue to exclude replies/comments.

### Q6: How should blank-search top/trending hashtags be sourced and ranked?

Answer: Use project posts only for craft-grouped top hashtags, support all current craft tokens by default, and rank by distinct visible project-post count in the last 28 days.

Decision / implication: Craft grouping is based on project `common.craftType`; regular posts are not inferred into craft groups. Default groups include knitting, crochet, sewing, embroidery, and quilting as full craft tokens, even when a group is empty.

### Q7: What craft-type contract should project/top-hashtag APIs use?

Answer: Return full craft tokens and accept full tokens plus supported bare aliases as inputs.

Decision / implication: AppView canonicalizes to `social.craftsky.feed.defs#...` values for comparisons and response groups. Flutter maps tokens to labels.

### Q8: Should typeahead suggestions paginate?

Answer: No — typeahead suggestions are top-N only, with per-section `hasMore` metadata.

Decision / implication: Suggestion dropdowns remain fast and bounded. “View all” navigates to submitted result tabs rather than paginating inside typeahead.

### Q9: How should submitted-search Hashtags tab matching/ranking work?

Answer: Normalize away a leading `#`, match tags by case-insensitive substring, rank exact match first, prefix matches next, then 28-day count descending, then tag ascending.

Decision / implication: Hashtag entity search remains lexical and deterministic. Counts use distinct visible top-level regular posts and projects from the last 28 days.

### Q10: Which recent-search item types should the future Search recents list contain?

Answer: Free-text submitted searches, selected hashtags, and selected profiles. Project browse/filter combinations should not be added to Search recents in this slice.

Decision / implication: Add a generic `query` recent type for free-text all-tabs searches. Hashtag recents open exact hashtag results. Profile recents open the selected profile directly. Flutter should not generate project-filter recents for the Projects bottom-nav surface.

### Q11: What payloads should selected profile and hashtag recents store?

Answer: Profile recents store stable selected-profile identity and navigate directly to that profile; hashtag recents store canonical tag and open exact hashtag results.

Decision / implication: Profile recents should not rerun a profile search query. Existing recent payload validation/modeling should be refined from query-shaped profile recents to selected-profile recents.

### Q12: What payload should a generic free-text query recent store?

Answer: Store the query text only.

Decision / implication: Selecting a query recent reopens the all-tabs submitted search screen at its default tab/sort. Active-tab/sort memory is future work.

### Q13: Where should rich project filters live?

Answer: Move rich project filter support to `/v1/projects`; submitted-search Projects tab should be text-search-only.

Decision / implication: `/v1/search/projects` remains for committed text-search Projects results. `/v1/projects` owns craft tabs, project filter families, chronological/popular sort, and project browse pagination.

### Q14: How should submitted free-text search results rank?

Answer: Posts and Projects tabs should default to text relevance, not chronology.

Decision / implication: Free-text search is relevance-first with explicit ordering by relevance score descending, then `createdAt` descending, then URI descending. Chronological/popular sorting belongs primarily to project browse; additional search sorts can be future work.

### Q15: How should Projects bottom-nav popular sort work?

Answer: Reuse the existing deterministic engagement plus recency-decay popularity formula.

Decision / implication: Project browse `popular` remains testable and consistent with the previous search foundation formula.

### Q16: What profile suggestion fields are needed for the mockup subtitle?

Answer: Include profile `crafts` in profile suggestion/search summary data.

Decision / implication: AppView/Flutter profile suggestion summaries should carry craft tokens so future UI can render context such as “Sophie • Knitter”.

## 4. Candidate Approaches

### Option A: Shared suggestion core plus separate result and browse APIs

Summary: Add a unified search suggestion contract for profile/hashtag typeahead, route both search suggestions and facet autocomplete through shared ranking/query logic, add a paginated hashtag-query result surface, keep committed result tabs as separate providers/endpoints, and keep project browsing/filtering under `/v1/projects`.

Pros:
- Reuses profile/hashtag suggestion ranking for both composer facets and search typeahead.
- Keeps exact search results, typeahead suggestions, recent history, and project browse semantics distinct and testable.
- Preserves existing `/v1/facets/*` compatibility while allowing the search UI to consume one cohesive suggestion response.
- Matches the new Projects bottom-nav direction and avoids making project filtering feel like search.
- Allows AppView to share internal query builders between `/v1/search/projects` and `/v1/projects` without exposing one catch-all API.

Cons:
- Requires coordinated AppView and Flutter data-layer changes.
- Adds at least one new AppView API contract and new Flutter suggestion/hashtag-result models/providers.
- May require updating existing tests that assumed bare craft-type values or project-including post search.

Risks:
- Suggestion and search result semantics can be confused if naming and tests do not clearly separate them.
- Craft-token normalization mistakes could make project tabs/top hashtags appear empty once real project records are indexed.
- Hashtag-query pagination can become inefficient without careful indexed query design.

### Option B: Keep endpoints separate and centralize only internals

Summary: Leave `/v1/facets/*`, `/v1/search/*`, and `/v1/projects` endpoint shapes mostly unchanged while refactoring AppView internals so facet and search suggestions share ranking logic.

Pros:
- Lower API churn.
- Easier to land as a narrow backend refactor.
- Existing Flutter facet autocomplete code can remain almost untouched.

Cons:
- Future search typeahead still has to orchestrate multiple suggestion calls or reuse rich-text-specific repositories.
- Does not give the Flutter search feature a cohesive profile+hashtag suggestion contract.
- Leaves the missing paginated Hashtags result tab unaddressed unless added separately.

Risks:
- Endpoint duplication may continue to drift as search UI grows.
- The app may accidentally couple search UI to composer-specific facet abstractions.

### Option C: Broad generic search API

Summary: Add one generic endpoint for suggestions, committed search result tabs, recents, and project browse/filtering.

Pros:
- One apparent client entry point.
- Maximum flexibility in one surface.

Cons:
- Combines typeahead, result search, private history, and project browse into one ambiguous API.
- Harder to validate and test precisely because profiles, posts, projects, hashtags, recents, and project filters have different semantics.
- More likely to require churn as UI decisions evolve.

Risks:
- Catch-all endpoint becomes a product-policy bottleneck and obscures ranking/filtering rules.

## 5. Recommended Direction

Recommended approach: Option A — shared suggestion core plus separate result and browse APIs.

Why: It matches the confirmed direction, keeps project filtering under the Projects feature, gives search typeahead a cohesive profile+hashtag data contract, preserves existing composer facet behavior, and keeps result-tab pagination/ranking testable per result type.

## 6. Problem / Opportunity

The initial AppView and Flutter search data layers cover foundational result fetching, recents, and project search, but the desired search UX has sharpened. Before UI work begins, the backend and Flutter logic should align around the future product shape: blank search discovery, shared typeahead suggestions, disjoint result tabs, exact hashtag results, and a separate project-browse/filter surface. Refining these contracts now reduces UI churn and prevents search-specific code from absorbing project browsing or duplicating facet-suggestion logic.

## 7. Goals

- G-001: Provide non-UI AppView and Flutter data contracts for the future blank search page: recent searches plus top hashtags by craft type.
- G-002: Provide a shared profile+hashtag suggestion surface that can power both search typeahead and composer/profile rich-text facets with consistent ranking.
- G-003: Support submitted search results as four separately paginated result types: posts, projects, profiles, and hashtags.
- G-004: Preserve exact hashtag navigation/results for selected hashtag suggestions or hashtag taps.
- G-005: Keep project browsing/filtering in the Projects feature/API, with craft-type tabs, filters, and chronological/popular sorting supported by non-UI data code.
- G-006: Resolve craft-type token normalization before UI code depends on project/top-hashtag contracts.
- G-007: Keep the slice limited to AppView and Flutter data/logic layers, not rendered UI.

## 8. Non-Goals

- NG-001: Do not implement rendered search UI, search tabs, search box widgets, recent-search management screens, project tab UI, filter controls, cards, or scrolling layouts.
- NG-002: Do not change visual navigation or add new rendered routes/pages in this slice, except for compile/build compatibility if generated route code already exists.
- NG-003: Do not create a separate saved-search feature distinct from existing recent searches.
- NG-004: Do not change atproto lexicons or write recent/saved search state to a PDS.
- NG-005: Do not add semantic search, embeddings, typo-tolerant search, recommendations, or algorithmic ranking beyond the existing explicit relevance/popularity rules.
- NG-006: Do not remove existing `/v1/facets/*` routes without preserving composer/profile autocomplete compatibility.
- NG-007: Do not add unauthenticated public search endpoints in this slice.
- NG-008: Do not implement analytics, telemetry, push notifications, or background polling for search.
- NG-009: Do not make the Flutter app read craft data from a PDS directly or store PDS tokens.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in Craftsky user | A user who will later use search and project discovery in the Flutter app. | Relevant suggestions, clear result tabs, exact hashtag navigation, private recent-search management, and project browsing/filtering. |
| Future search UI | Flutter widgets to be implemented after this slice. | Stable providers/models for blank state, typeahead, result tabs, exact hashtag results, and recent-search actions. |
| Future project UI | Flutter widgets for the Projects bottom-nav branch. | Stable providers/models for craft tabs, project filters, sort choice, pagination, and project list data. |
| Composer/profile rich-text autocomplete | Existing Flutter rich-text/facet code paths. | Same or better mention/hashtag suggestions without regressions. |
| AppView API | Go HTTP API and Postgres query layer. | Clear authenticated endpoints, shared ranking logic, normalized craft filters, bounded pagination, and testable contracts. |
| Test designer / implementer | The next workflow agents. | Traceable requirements and acceptance criteria that distinguish suggestions, search results, recents, and project browsing. |

## 10. Current Behavior

AppView has search result, top-hashtag, recent-search, facet suggestion, and project-list endpoints, but their shapes reflect the earlier search foundation. Search suggestions and facet suggestions are separate; there is no unified profile+hashtag typeahead response. Profile suggestion ranking exists in multiple places, and profile suggestion/search summaries do not currently expose Craftsky `crafts` for UI subtitles. There is no paginated committed hashtag-query endpoint for a Hashtags result tab. `/v1/search/posts` currently searches top-level posts and may include project posts, while the future UI expects separate Posts and Projects tabs. Free-text post/project search currently defaults to chronological/popular behavior from the previous foundation, but the confirmed search UX expects text relevance by default. `/v1/projects` is separate from search but currently supports only craft type, sort, limit, and cursor; richer project filters are under `/v1/search/projects`, which conflicts with the confirmed project-browse boundary. Existing recent-search payload types include query-shaped post/profile/project searches, but the confirmed recents behavior needs a generic all-tabs `query` recent, direct selected-profile recents, exact-hashtag recents, and no project-filter recents generated by Flutter. Some current search/project code uses bare craft values (`knitting`) even though project records and Flutter project option catalogs use full lexicon tokens (`social.craftsky.feed.defs#knitting`). Flutter has non-UI search and project data layers, but no search-specific suggestion provider, no hashtag-query result provider, no combined blank-search state provider, and no UI consumption.

## 11. Desired Behavior

After this slice, the AppView and Flutter data/logic layers are aligned with the desired future search and project-discovery UX. A unified top-N suggestion contract returns Craftsky-only profile suggestions and hashtag suggestions using shared ranking/count logic, per-section `hasMore`, and profile craft metadata for future subtitles, while existing facet autocomplete remains compatible. Submitted free-text searches can independently page posts, projects, profiles, and hashtags; the Posts and Projects tabs are disjoint for this slice and text results default to relevance. Exact hashtag selections fetch one combined top-level feed of regular posts and projects with that exact normalized tag, sortable by chronology or popularity. Recent/saved searches remain one private AppView-backed history surface, now shaped around generic free-text query recents, exact hashtag recents, and direct selected-profile recents; project browse/filter combinations do not appear in Search recents. Project browsing and filtering are served by the Projects feature/API (`/v1/projects`) rather than being treated as search UI, while AppView may share backend query code internally. Craft-type inputs are normalized to full lexicon tokens so top hashtags, project tabs, project filters, and project records use one consistent contract.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | The system shall provide non-UI AppView and Flutter contracts for the refined search experience before rendered search UI work begins. | The user explicitly wants planning and data/logic refinements before UI implementation. | Prompt | AC-001, AC-006, AC-012 |
| BR-002 | Business | Must | The system shall support blank search-page data needs: private recent searches and top/trending hashtags grouped by craft type. | The desired landing search page shows recents and craft-grouped hashtags. | Prompt | AC-011, AC-012, AC-013 |
| BR-003 | Business | Must | The system shall support search typeahead suggestions for profiles and hashtags using ranking/count semantics that are reusable by composer/profile facet autocomplete. | Profile suggestions should rank the same in search and project composer/facet contexts. | Prompt, Q2 | AC-002, AC-003, AC-004 |
| BR-004 | Business | Must | The system shall support submitted search results as four independently page-able result categories: posts, projects, profiles, and hashtags. | The desired submitted search screen has four tabs. | Prompt | AC-005, AC-006, AC-007, AC-014 |
| BR-005 | Business | Must | The system shall keep project browsing/filtering as a Projects feature rather than folding it into the search flow. | The user now has a separate Projects bottom-nav surface where filtering occurs. | Prompt, Q2 | AC-009, AC-010 |
| BR-006 | Business | Must | Recent and saved searches shall be treated as the same managed private search-history surface for this slice. | The user clarified that “saved” was a misstatement. | Q1 | AC-011 |
| FR-001 | Functional | Must | The AppView shall expose a unified authenticated top-N suggestion contract for profile and hashtag typeahead, recommended as `GET /v1/search/suggestions`, requiring a trimmed non-empty `q`, optional type selection, bounded per-type limits, and returning per-section `hasMore` metadata; empty or whitespace-only `q` is a standard validation error. | Future search UI should not have to call separate composer-specific facet endpoints for one typeahead panel, and typeahead should not paginate internally. | Q2, Q8, Discovery | AC-002, AC-014, AC-017 |
| FR-002 | Functional | Must | Profile suggestions shall be Craftsky-profile-only and produced by shared ranking logic used by both the unified suggestion contract and existing facet mention/autocomplete behavior; profile suggestion/search summary data shall include craft metadata needed for future subtitles. | Ranking consistency is a stated product requirement, and the mockup shows profile craft context. | Prompt, Q4, Q16, Discovery | AC-003, AC-004, AC-017 |
| FR-003 | Functional | Must | Hashtag suggestions shall be produced by shared normalization/count logic used by both the unified suggestion contract and existing facet hashtag/autocomplete behavior, using distinct visible top-level regular posts and projects from the last 28 days for global suggestion counts. | Search and composer hashtag suggestions should not drift, and global hashtag discovery should not exclude regular posts. | Prompt, Q9, Discovery | AC-002, AC-004, AC-005 |
| FR-004 | Functional | Must | Existing `/v1/facets/mentions`, `/v1/facets/mentions/resolve`, and `/v1/facets/hashtags` shall remain compatible for current Flutter rich-text/facet callers, either as wrappers over the shared suggestion core or by migrating those callers without behavior regressions. | The slice must not break composer/profile autocomplete while adding search suggestions. | Discovery | AC-004 |
| FR-005 | Functional | Must | The AppView shall expose a paginated committed hashtag-query result endpoint for the future Hashtags search tab, recommended as `GET /v1/search/hashtags?q=...`, returning normalized hashtag items with 28-day counts, opaque cursor pagination, and deterministic ranking: exact match first, prefix matches next, then count descending, then tag ascending. | The desired submitted search screen has a Hashtags tab, which is distinct from exact hashtag post results and typeahead suggestions. | Prompt, Q9, Discovery | AC-005, AC-014 |
| FR-006 | Functional | Must | The AppView and Flutter search data layer shall provide separate paginated fetch paths/providers for submitted query results: posts, projects, profiles, and hashtags; submitted free-text post/project results shall default to an explicit relevance score ordered by score descending, then `createdAt` descending, then URI descending. The relevance score shall use PostgreSQL full-text rank over the searchable fields or an equivalent explicit scoring helper; chronology/popularity must not be the default free-text ordering. | Future tab UI needs independent pagination and loading states per tab, and text search should behave like search rather than a browse feed. | Prompt, Q2, Q14 | AC-006, AC-014, AC-018 |
| FR-007 | Functional | Must | For submitted query results in this slice, the Posts fetch path shall return top-level non-project posts, and the Projects fetch path shall return top-level project posts. | Separate tabs should not duplicate the same project item in both Posts and Projects by default. | Prompt interpretation, Discovery | AC-007 |
| FR-008 | Functional | Must | Exact hashtag selection shall use `/v1/search/hashtags/{tag}/posts` or an equivalent exact normalized hashtag result fetch path that returns one combined feed of top-level regular posts and project posts tagged with that exact hashtag, supports chronological and popular sorting, and is not substring hashtag suggestions and not replies/comments. | Clicking a hashtag suggestion/option should show all top-level posts/projects with that exact hashtag, with future UI able to present chronological or popular exact-tag feeds. | Prompt, Q5, Prior search requirements, User feedback | AC-008, AC-021 |
| FR-009 | Functional | Must | The Projects API/data layer shall support project browsing by all current supported craft types, chronological/popular sort, opaque cursor pagination, and project filter families needed by the project filtering UI; popular sort shall reuse the existing deterministic engagement plus recency-decay formula. | Project filtering now belongs in the Projects bottom-nav surface. | Prompt, Q6, Q13, Q15 | AC-009, AC-010, AC-014 |
| FR-010 | Functional | Must | `/v1/projects` shall be the canonical AppView route family for project browsing/filtering, while `/v1/search/projects` remains available for committed text-search Projects tab results only; rich filter families shall move out of the public `/v1/search/projects` contract. | Keeps project browse separate from search without duplicating backend logic unnecessarily. | Prompt, Q13 | AC-009, AC-010, AC-019 |
| FR-011 | Functional | Must | Flutter shall keep project browse/filter state under the project feature data layer, not under the search repository/provider boundary. | Future Projects UI should consume project providers directly. | Discovery | AC-010 |
| FR-012 | Functional | Must | Flutter shall add or adapt search data-layer models/providers for unified suggestions, paginated hashtag-query results, submitted query result tabs, blank-search data, and exact hashtag result state without rendering UI. | Future UI needs stable provider surfaces after this non-UI slice. | Prompt, Discovery | AC-001, AC-002, AC-005, AC-006, AC-008, AC-012, AC-021 |
| FR-013 | Functional | Must | Recent-search list, save, and delete behavior shall remain explicit AppView-backed behavior, with no automatic recent mutation from typeahead or result fetches. | Recents are private history and existing search foundation chose explicit saves. | Q1, Q10, Prior search requirements | AC-011 |
| FR-014 | Functional | Must | AppView shall canonicalize supported craft-type inputs for project browse, project search, and top-hashtag grouping to full `social.craftsky.feed.defs#...` tokens while accepting bare aliases for backwards/developer convenience. | Existing records use full tokens, but some current query code/tests use bare values. | Discovery | AC-013 |
| FR-015 | Functional | Must | AppView responses that identify project craft groups, including top-hashtag groups and project browse filters where applicable, shall expose canonical full craft-type tokens rather than bare aliases. | Flutter can map tokens to labels through existing project option catalogs. | Discovery | AC-013 |
| FR-016 | Functional | Must | Recent-search payload contracts for the future Search recents list shall support generic free-text `query` recents with `q` only, exact `hashtag` recents with canonical tag, and direct selected `profile` recents with stable profile identity; Flutter shall not generate project browse/filter recents in this slice. | The mockup includes free-text, hashtag, and profile recents, while project filtering has moved to Projects. | Q10, Q11, Q12, Q13, Q14, Q15 | AC-011, AC-020 |
| NFR-001 | Non-functional | Must | This slice shall not implement rendered UI or visual route/navigation behavior. | The user explicitly asked to avoid UI-specific work. | Prompt | AC-001 |
| NFR-002 | Non-functional | Must | Flutter search/project/facet calls shall continue to use authenticated AppView HTTP via the shared Dio stack and shall not call PDS directly. | Project architecture requires AppView reads and no PDS tokens on device. | AGENTS.md, Discovery | AC-014, AC-015 |
| NFR-003 | Non-functional | Must | New or changed AppView APIs shall follow existing `/v1/` conventions: authenticated session, device ID, camelCase JSON, standard error envelopes, bounded limits, and opaque cursors for paginated lists. | Maintains API consistency and client expectations. | API specs, Discovery | AC-014, AC-021 |
| NFR-004 | Non-functional | Should | Search/suggestion/project queries should remain bounded and index-aware, using existing materialized columns/indexes where possible and adding supporting indexes only if needed. | Search and hashtag queries can degrade as indexed data grows. | Discovery | AC-016 |
| NFR-005 | Non-functional | Should | Tests should cover AppView route/request/response contracts, shared ranking/normalization helpers, Flutter API clients, repositories, providers, pagination, and compatibility regressions. | The test-design stage needs clear coverage targets. | Discovery | AC-016 |
| RULE-001 | Business rule | Must | Recent/saved searches are one private AppView-backed history surface in this slice; there shall be no separate saved-search table, route family, repository, or provider. | The user clarified saved/recent were the same. | Q1 | AC-011, AC-015 |
| RULE-002 | Business rule | Must | Typeahead suggestions and result fetches shall not automatically save recent searches; only explicit recent-search save calls mutate recents. | Avoids noisy recent history from typing/pagination. | Prior search requirements | AC-011 |
| RULE-003 | Business rule | Must | Profile suggestion ranking shall be equivalent for search typeahead and facet mention autocomplete. | The user explicitly wants profile suggestion ranking to match between composer and search. | Prompt | AC-003, AC-004 |
| RULE-004 | Business rule | Must | Hashtag typeahead/search-tab query matching and exact hashtag result matching are different modes: typeahead/query tabs may use substring matching, but exact hashtag result screens must match one normalized tag exactly; chronological/popular exact-feed sorting must not change exact-match semantics. | Prevents confusing suggestion semantics with exact hashtag navigation. | Prompt, Prior search requirements, User feedback | AC-005, AC-008, AC-021 |
| RULE-005 | Business rule | Must | Project browse/filtering belongs to the Projects API/data-layer boundary, not to search UI state, even if AppView reuses search-store internals. | The user moved project filtering to the Projects bottom nav. | Prompt, Q2 | AC-009, AC-010 |
| RULE-006 | Business rule | Must | Submitted-query Posts and Projects result tabs shall be disjoint for this slice. | Separate tabs should avoid duplicate result cards and make acceptance testing deterministic. | Prompt interpretation | AC-007 |
| RULE-007 | Business rule | Must | Typeahead suggestions shall be bounded top-N previews, not paginated result lists; “View all” behavior belongs to submitted result tabs. | Keeps suggestions fast and separates transient typeahead from committed search. | Q8 | AC-002, AC-017 |
| RULE-008 | Business rule | Must | Submitted free-text search shall default to explicit relevance-score ordering: score descending, then `createdAt` descending, then URI descending. Chronological/popular sorting is part of project browse unless explicitly added to search later. | Text search should prioritize query relevance and remain deterministic. | Q14 | AC-006, AC-018 |
| RULE-009 | Business rule | Must | Project browse/filter combinations from the Projects bottom-nav surface shall not be generated as Search recent/saved search items in this slice. | Prevents project browse state from polluting Search recents. | Q10 | AC-011, AC-020 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-012, NFR-001 | Given this slice is implemented, when Flutter UI/page/route files are inspected, then there is no rendered search UI, project UI, search tab UI, recent-management page UI, or visual navigation behavior added by this slice. |
| AC-002 | BR-003, FR-001, FR-003, FR-012, RULE-007 | Given an authenticated user and a non-empty typeahead query, when the unified suggestion contract is requested for profiles and hashtags, then AppView returns bounded top-N grouped profile and hashtag suggestions with camelCase fields, normalized hashtag values, per-section `hasMore`, and no pagination cursor. |
| AC-003 | BR-003, FR-002, RULE-003 | Given the same viewer, indexed Craftsky profiles, follows, crafts, and query text, when profile suggestions are requested through search typeahead and through the facet/autocomplete path, then the relative ranking of returned profile suggestions is equivalent for the overlapping result set and returned profile suggestion/search summary data includes craft metadata. |
| AC-004 | FR-004, RULE-003 | Given existing Flutter rich-text/facet autocomplete code requests mention or hashtag suggestions, when the suggestion refactor is complete, then existing facet autocomplete tests still pass and the response fields expected by composer/profile code remain compatible. |
| AC-005 | BR-004, FR-005, RULE-004 | Given indexed hashtags matching a submitted query, when the committed hashtag-query endpoint is requested with `q`, `limit`, and optional `cursor`, then it returns hashtag result items ranked exact match first, prefix matches next, then 28-day count descending, then tag ascending, with an opaque next cursor when more results exist. |
| AC-006 | BR-001, BR-004, FR-006, FR-012, RULE-008 | Given a submitted query, when Flutter data-layer providers for posts, projects, profiles, and hashtags are read, then each fetches through the appropriate repository/API path, exposes UI-agnostic async state, can paginate independently, and post/project text results use relevance-score ordering by default. |
| AC-007 | FR-007, RULE-006 | Given indexed regular posts and project posts that both match a submitted query, when Posts and Projects result providers/endpoints are fetched, then regular posts appear only in Posts results and project posts appear only in Projects results. |
| AC-008 | FR-008, FR-012, RULE-004 | Given a selected hashtag value with optional leading `#` or mixed casing, when exact hashtag results are requested, then the request is normalized safely and returns top-level regular posts and project posts matching that exact canonical tag only. |
| AC-009 | BR-005, FR-009, FR-010, RULE-005 | Given project browse parameters with craft type, sort, limit, cursor, and supported project filter families, when `/v1/projects` is requested, then it returns paginated project posts matching those browse/filter parameters without requiring use of a search UI endpoint, and `sort=popular` uses the existing deterministic popularity formula. |
| AC-010 | BR-005, FR-010, FR-011, RULE-005 | Given future Projects UI code reads project browse providers, when craft type, sort, filters, and pagination state are supplied, then Flutter calls the project repository/data layer rather than the search repository/data layer. |
| AC-011 | BR-002, BR-006, FR-013, FR-016, RULE-001, RULE-002, RULE-009 | Given typeahead suggestions and result fetches are performed, when recent searches are listed afterward, then recents are unchanged unless an explicit save mutation was called; explicit saves support free-text query, selected hashtag, and selected profile recents, and project browse/filter interactions do not generate Search recents in this slice. |
| AC-012 | BR-001, BR-002, FR-012 | Given future blank `SearchPage` logic consumes non-UI data providers, when blank-search data is requested, then recent searches and craft-grouped top hashtags for the supported default craft tokens are fetchable without requiring rendered UI code. |
| AC-013 | BR-002, FR-014, FR-015 | Given project records store full craft tokens and callers provide either full tokens or supported bare aliases, when project browse/search/top-hashtag requests are handled, then comparisons use canonical full tokens and responses expose canonical full craft tokens for craft groups, including empty supported groups when explicitly/default requested. |
| AC-014 | BR-004, FR-001, FR-005, FR-006, FR-009, NFR-002, NFR-003 | Given new or changed AppView endpoints are called without auth/device headers, with invalid limits, or with invalid cursors, then they follow existing `/v1/` authentication, validation, error-envelope, and opaque-cursor behavior; given valid requests, Flutter uses shared authenticated Dio and preserves cursors opaquely. |
| AC-015 | NFR-002, RULE-001 | Given recent/saved search behavior is implemented, when code paths are inspected, then no recent/saved search state is written to PDS records or local persistent Flutter search-history storage in this slice. |
| AC-016 | NFR-004, NFR-005 | Given focused AppView and Flutter tests run for search, facets, and projects, when this slice is complete, then tests cover ranking consistency, craft-token normalization, hashtag-query pagination, project browse filters, provider pagination, and compatibility regressions or explicitly document any gaps. |
| AC-017 | FR-001, FR-002, RULE-007 | Given more profile or hashtag suggestion matches exist than the requested top-N limit, when unified suggestions are requested, then each affected section returns only the bounded items plus `hasMore: true`; given no extra matches exist, `hasMore` is false. |
| AC-018 | FR-006, RULE-008 | Given multiple post/project text-search matches with different textual relevance and creation times, when submitted search results are requested without an explicit future sort override, then higher textual relevance scores rank ahead of newer-but-less-relevant matches, and equal relevance scores tie by `createdAt` descending and then URI descending. |
| AC-019 | FR-010 | Given a request to `/v1/search/projects` includes rich project browse filter parameters such as `craftType`, `color`, `material`, `designTag`, `projectTag`, `patternDifficulty`, or `projectType`, when the refined contract is implemented, then AppView rejects those unsupported filter parameters with the standard validation error, while `/v1/projects` accepts supported browse filters. |
| AC-020 | FR-016, RULE-009 | Given recent-search save requests for `query`, `hashtag`, and `profile`, when AppView stores and returns them, then `query` payloads contain `q` only, `hashtag` payloads contain canonical `tag`, and `profile` payloads contain stable selected-profile identity for direct navigation; Flutter does not serialize project browse/filter recents. |
| AC-021 | FR-008, FR-012, NFR-003, RULE-004 | Given exact hashtag posts exist with different creation times and popularity scores, when `/v1/search/hashtags/{tag}/posts` is requested with `sort=chronological` or `sort=popular`, then AppView returns only top-level regular posts and project posts matching that exact canonical tag ordered by the selected sort, preserves opaque cursor pagination for that sort, and rejects unsupported sort values with the standard validation error. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Unified suggestions are requested with an empty or whitespace-only query. | AppView returns the standard validation error; Flutter does not issue repeated noisy calls for blank input. | FR-001 |
| EC-002 | Suggestion query starts with `#` or `@`. | Hashtag/profile matching normalizes the prefix where appropriate without leaking the prefix into canonical hashtag tags or handles. | FR-001, FR-003 |
| EC-003 | Existing facet endpoint callers encounter AppView suggestion errors. | Existing tolerant autocomplete behavior is preserved where applicable; composer/profile autocomplete should not crash or block post composition on suggestion failures. | FR-004 |
| EC-004 | Hashtag-query result endpoint receives a query that matches no tags. | Returns an empty `items` list and no cursor. | FR-005 |
| EC-005 | Exact hashtag result receives a tag containing spaces, slashes, control characters, or only `#`. | AppView returns the standard validation error; Flutter surfaces mapped API errors and does not construct invalid path segments. | FR-008, NFR-003 |
| EC-006 | Project browse receives both full token and bare alias craft-type inputs. | Both are canonicalized to the same supported full token for filtering; duplicate equivalent values do not duplicate results. | FR-014 |
| EC-007 | Project browse receives an unknown craft type or unsupported filter key. | AppView returns a standard validation error rather than silently returning all projects. | FR-009, FR-014, NFR-003 |
| EC-008 | A project has a future craft token not currently in the supported alias list. | Existing indexed project data remains readable, but filter/top-hashtag canonicalization only promises supported current tokens unless future-token support is explicitly added. | FR-014, FR-015 |
| EC-009 | Posts and Projects tabs both match the same project text. | The project appears only in Projects results for this slice. | FR-007, RULE-006 |
| EC-010 | A user deletes an already-deleted or not-owned recent-search ID. | Existing idempotent delete semantics remain: AppView does not reveal ownership/existence and Flutter treats success as success. | FR-013 |
| EC-011 | A load-more call is issued while another page load is in progress. | Flutter provider state avoids duplicate concurrent pagination requests, following existing search/project pagination patterns. | FR-006, FR-011, FR-012 |
| EC-012 | Top hashtags are requested for a craft type with no recent tags. | The craft group is returned with an empty `items` list when that craft group was explicitly requested or included by default. | BR-002, FR-015 |
| EC-013 | A free-text query recent is saved with blank or overlong `q`. | AppView rejects it with standard validation; Flutter should not intentionally serialize blank query recents. | FR-016, NFR-003 |
| EC-014 | A selected profile recent's handle later changes. | The recent can still navigate by stable DID, while display handle/label can be refreshed by the profile surface later. | FR-016 |
| EC-015 | A project browse filter is accidentally sent to `/v1/search/projects`. | AppView rejects it with the standard validation error under the refined contract; callers should use `/v1/projects` for filters. | FR-010, RULE-005 |
| EC-016 | A suggestion section has exactly the requested limit of matches but no additional match. | `hasMore` is false; implementations should fetch limit+1 internally or otherwise avoid falsely showing View all. | FR-001, RULE-007 |
| EC-017 | A submitted text-search match is very new but weakly relevant. | It does not outrank a substantially more relevant match solely because it is newer under the default relevance sort; recency applies only after relevance scores tie. | FR-006, RULE-008 |
| EC-018 | Exact hashtag results are requested with no `sort`, with `sort=chronological`, with `sort=popular`, or with an unsupported sort. | Missing sort uses the existing/default exact-hashtag ordering, chronological and popular sorts are accepted and stable, and unsupported sort values return the standard validation error without broadening hashtag matches. | FR-008, NFR-003, RULE-004 |

## 15. Data / Persistence Impact

- New fields:
  - AppView response models for unified suggestions and paginated hashtag-query results are expected.
  - Profile suggestion/search summary response models should include profile craft metadata.
  - Recent-search payload contracts should add a generic `query` type and refine selected `profile` recents to direct-profile payloads.
  - Flutter models may be added for unified suggestions, hashtag result pages, submitted search tab queries/state, project browse filters, profile crafts in suggestions, and refined recent-search payloads.
- Changed fields:
  - Top-hashtag/project craft group responses should use canonical full craft tokens rather than bare aliases.
  - `/v1/search/posts` semantics may change to exclude project posts for disjoint result tabs.
  - `/v1/search/projects` semantics should be text-search-only for the submitted-search Projects tab and should no longer expose rich project browse filters as part of its public contract.
  - `/v1/projects` request parsing should expand from craft type/sort only to supported project browse filter families.
  - Submitted post/project text search ordering should default to relevance rather than chronological ordering.
  - Exact hashtag result requests through `/v1/search/hashtags/{tag}/posts` should support chronological and popular sort values without changing exact-tag matching semantics.
- Migration required:
  - No new persistence table is required for saved searches; existing `craftsky_recent_searches` remains the only recent/saved search storage.
  - A small AppView migration is likely required if the `craftsky_recent_searches.search_type` check constraint must add `query` and/or adjust supported recent payload types.
  - No lexicon migration is required.
  - Supporting database indexes may be added if needed for hashtag-query or project-filter performance, but the requirement is bounded/index-aware query behavior rather than a specific migration.
- Backwards compatibility:
  - Existing `/v1/facets/*` callers must remain compatible.
  - Existing recent-search IDs remain valid. Existing pre-UI `post`/`project` recent payload support may be retained or migrated internally, but future Flutter Search recents should generate `query`, `hashtag`, and selected `profile` recents only.
  - Bare craft-type inputs should continue to work as accepted aliases, while canonical responses use full tokens.
  - The API is still pre-release; search result response/semantics churn is acceptable if Flutter data-layer tests are updated in the same slice.

## 16. UI / API / CLI Impact

- UI:
  - No rendered UI implementation in this slice.
  - Future search UI will consume blank-search, suggestion, result-tab, exact hashtag, and recent-search providers.
  - Future Projects UI will consume project browse/filter providers.
- API:
  - Add or finalize a unified authenticated suggestion endpoint, recommended as `GET /v1/search/suggestions`.
  - Add or finalize a paginated committed hashtag-query endpoint, recommended as `GET /v1/search/hashtags`.
  - Unified suggestions return top-N profile/hashtag sections with `hasMore`; they do not paginate.
  - Preserve `/v1/facets/*` compatibility.
  - Expand `/v1/projects` for project browse/filter semantics, including all current supported craft tokens and chronological/popular sort.
  - Keep `/v1/search/projects` for committed text-search Projects tab semantics only; move rich project filters to `/v1/projects`.
  - Exact hashtag results continue through `/v1/search/hashtags/{tag}/posts`, accept chronological/popular sort values, and return one combined top-level regular-post/project feed unless a later UI/API design intentionally renames it.
  - Refine recent-search save/list payloads to support `query`, exact `hashtag`, and direct selected `profile` recents for the future Search recents list.
- CLI:
  - No CLI behavior change expected.
- Background jobs:
  - No new background job or polling behavior required.

## 17. Security / Privacy / Permissions

- Authentication:
  - All new/changed `/v1/*` search, suggestion, recent, and project endpoints require the existing authenticated session and device ID unless explicitly already public; no public search is added in this slice.
- Authorization:
  - Recent/saved search list/save/delete remains scoped to the authenticated viewer DID.
  - Search/project result visibility remains governed by existing AppView moderation and visibility predicates.
- Sensitive data:
  - Recent/saved searches are private AppView state and must not be written to PDS records.
  - Flutter must not add local persistent search-history storage in this slice.
- Abuse cases:
  - Suggestion/result endpoints should enforce bounded limits and query length validation.
  - AppView logs should avoid dumping full private recent-search payloads unnecessarily.

## 18. Observability

- Events:
  - No analytics or product events required in this slice.
- Logs:
  - AppView should continue structured error logging with request/run IDs for failed suggestion/search/project requests, avoiding sensitive payload dumps.
- Metrics:
  - No new metrics required, though latency/error-rate metrics for search/suggestion endpoints can be added later.
- Alerts:
  - No new alerts required.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Suggestion, search result, and project browse semantics become blurred. | The UI and tests may call the wrong endpoints or persist the wrong state. | Keep endpoint/provider names distinct and test each behavior separately. |
| RISK-002 | Craft-type token mismatch persists. | Project tabs/top hashtags may appear empty for real project records. | Canonicalize to full `social.craftsky.feed.defs#...` tokens and accept bare aliases only as inputs. |
| RISK-003 | `/v1/search/posts` behavior change surprises existing data-layer tests. | Current Flutter/AppView tests may fail because project posts are excluded. | Treat as intentional pre-UI contract refinement; update tests and document disjoint tabs. |
| RISK-004 | Hashtag-query result pagination is slow on larger data. | Hashtags tab could become expensive or time out. | Use bounded limits, indexed/materialized tag data, deterministic ordering, and add indexes if profiling/tests require. |
| RISK-005 | Facet autocomplete compatibility regresses during suggestion unification. | Project composer/profile rich-text flows lose mention/hashtag suggestions. | Preserve `/v1/facets/*` compatibility and include regression tests for existing rich-text suggestion providers. |
| RISK-006 | Provider contracts overfit imagined UI. | Later UI may need data-layer churn. | Keep providers UI-agnostic: query params, items, cursor, loading/error state, and mutations only. |
| RISK-007 | Recent-search payload refinement conflicts with the existing pre-UI `post`/`project` recent types. | AppView migrations/tests and Flutter recent models may need more churn than expected. | Add explicit `query`/selected-entity recents while retaining or migrating legacy types intentionally; do not generate project recents from Flutter Search. |
| RISK-008 | Relevance ranking for post/project text search is implemented inconsistently with the explicit relevance-score contract. | Result ordering may be inconsistent or hard to test. | Test score-descending ordering with `createdAt` and URI tie-breakers using PostgreSQL text rank or an equivalent explicit scoring helper. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | “Saved searches” and “recent searches” are the same product surface for this slice. | A distinct saved-search feature would require new persistence, endpoints, models, providers, and requirements. |
| ASM-002 | The future submitted-search Posts tab excludes project posts because Projects has its own tab. | If product later wants Posts to include all post types, FR-007/RULE-006 and related acceptance tests must change. |
| ASM-003 | Future UI can map full craft tokens to human-readable labels using existing project option catalogs. | If the API must return display labels too, response contracts need additive label fields. |
| ASM-004 | The canonical route names recommended here (`/v1/search/suggestions`, `/v1/search/hashtags`) are accepted for test design. | If route names change, requirements and tests must be updated together before implementation. |
| ASM-005 | Project browse filters reuse the filter families previously implemented for project search unless product adds more filter dimensions later. | New filter dimensions would need additional requirements and likely indexing work. |
| ASM-006 | A generic query recent should reopen the all-tabs submitted search screen at its default tab with query text only. | If UI later needs tab/sort memory, recent payloads need additive fields and tests. |
| ASM-007 | Profile recents can store stable DID plus handle/display metadata without requiring a fresh profile fetch at list time. | If stale handle/display labels are unacceptable, recents listing may need profile hydration. |

## 21. Open Questions

- None identified as blocking for test design.

## 22. Review Status

Status: Approved with notes
Risk level: Medium
Review recommended: Yes
Reviewer: OpenAI gpt-5.5 document reviewer
Date: 2026-06-21
Notes: Document review completed in `03-document-review.md`; review notes were folded into this artifact by making blank suggestion-query validation explicit, requiring validation errors for unsupported `/v1/search/projects` browse filters, and pinning submitted post/project relevance ordering to score descending with `createdAt` and URI tie-breakers.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-21-search-refinements/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - Business: `BR-001`, `BR-002`, `BR-003`, `BR-004`, `BR-005`, `BR-006`
  - Functional: `FR-001`, `FR-002`, `FR-003`, `FR-004`, `FR-005`, `FR-006`, `FR-007`, `FR-008`, `FR-009`, `FR-010`, `FR-011`, `FR-012`, `FR-013`, `FR-014`, `FR-015`, `FR-016`
  - Non-functional: `NFR-001`, `NFR-002`, `NFR-003`
  - Rules: `RULE-001`, `RULE-002`, `RULE-003`, `RULE-004`, `RULE-005`, `RULE-006`, `RULE-007`, `RULE-008`, `RULE-009`
- Suggested test levels:
  - AppView unit tests for request parsing, craft-token normalization, ranking helpers, relevance ordering, exact hashtag sort parsing, hashtag-query pagination/cursors, refined recent-search payload validation, recent-search non-mutation, and project browse filters.
  - AppView handler/route tests for new/changed endpoint auth, validation, response shape, and compatibility wrappers.
  - AppView integration/store tests for Craftsky-only profile suggestions with crafts, hashtag suggestions/counts, posts/projects disjoint relevance search, exact hashtag results with chronological/popular sorting, top hashtags with full craft tokens, and project browse filtering.
  - Flutter model/API-client tests for unified suggestions, hashtag result pages, exact hashtag sort requests, project browse filters, refined recent payloads, recent preservation, and error/cursor handling.
  - Flutter repository/provider tests for independent tab pagination, blank-search data, project browse provider boundaries, direct selected-profile/hashtag/query recents, and facet autocomplete regressions.
- Blocking open questions: None.
