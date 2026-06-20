# Requirements: Flutter Search Data Layer

## 1. Initial Request

Wire the newly implemented AppView search foundation into the Flutter app for a non-UI slice of work: services, repositories, providers, models, and related tests only. The search UI itself should remain out of scope.

## 2. Current Codebase Findings

- Relevant files:
  - Prior AppView search requirements and contracts: `docs/changes/2026-06-19-appview-search-foundation/01-requirements.md`, `02-acceptance-tests.md`, `05-implementation-plan.md`, `06-implementation-review.md`.
  - AppView search handlers/contracts: `appview/internal/routes/routes.go`, `appview/internal/api/search.go`, `appview/internal/api/search_request.go`, `appview/internal/api/search_response.go`.
  - Flutter search stub: `app/lib/search/pages/search_page.dart`, `app/test/search/search_page_test.dart`.
  - Existing API-client/repository/provider patterns: `app/lib/feed/data/post_api_client.dart`, `api_post_repository.dart`, `post_repository.dart`, `app/lib/feed/providers/*`; `app/lib/profile/data/profile_api_client.dart`, `profile_repository.dart`, `app/lib/profile/providers/*`; `app/lib/notifications/data/notification_api_client.dart`, `notifications_provider.dart`.
  - Existing shared AppView HTTP plumbing: `app/lib/shared/api/providers/dio_provider.dart`, `app/lib/shared/api/api_unwrap.dart`, `app/lib/shared/api/api_exception.dart`.
  - Existing models to reuse: `app/lib/feed/models/post.dart`, `post_page.dart`, `app/lib/profile/models/profile_account_summary.dart`, `app/lib/projects/models/project.dart`, `app/lib/projects/options/project_option_catalogs.dart`.
  - Mapper initialization: `app/lib/bootstrap.dart`.
- Existing patterns:
  - Feature API clients accept a shared authenticated `Dio`, call `/v1/*` endpoints, and wrap calls with `unwrapApi`.
  - Production repositories adapt API clients behind testable interfaces.
  - Repository and API-client providers use Riverpod codegen with `@Riverpod(keepAlive: true)`.
  - Paginated data providers use cursor-accumulating `AsyncNotifier` state with `loadMore()` and rely on Riverpod 3 preserving previous data during loading/error transitions.
  - Wire/data models use `dart_mappable`; new mappable classes require generated `*.mapper.dart` files and registration in `initializeMappers()` when needed by app startup/tests.
  - Tests commonly use `http_mock_adapter` for API-client request/response coverage and fake repositories for provider tests.
- Current behavior:
  - `SearchPage` renders only a stub title/body and optional hashtag text.
  - The Flutter app has no `SearchApiClient`, `SearchRepository`, search models, or search providers.
  - Hashtag facet taps can route to `/search?tag=...`, but no search result data is fetched.
  - Existing `/v1/facets/*` suggestion repositories remain autocomplete-only and are separate from committed search result endpoints.
- Constraints discovered:
  - This slice is Flutter-only. No AppView, lexicon, migration, dependency, or UI/page behavior changes are needed.
  - Reads must continue to use AppView JSON/HTTP via the existing authenticated Dio stack; the app must not call PDS directly or hold PDS tokens.
  - Recent searches are private AppView state. The app should call AppView recent-search endpoints, not store search history as public PDS records.
  - AppView search endpoints require authenticated session and `X-Craftsky-Device-Id`; the existing `dioProvider` supplies these through interceptors/providers.
  - AppView returns camelCase JSON, opaque cursors, and standard error envelopes; Flutter should keep cursors opaque and surface mapped `ApiException`s.
- Test/build commands discovered:
  - From `app/`: `dart run build_runner build --delete-conflicting-outputs` after new mappable/provider files.
  - From `app/`: focused `flutter test test/search` for the new slice.
  - From `app/`: likely broader verification `flutter test` and `flutter analyze`.

## 3. Clarifying Questions And Decisions

### Q1: Confirm the scope for this non-UI Flutter search slice.

Answer: Option A recommended.

Decision / implication: Build a dedicated search data layer under `app/lib/search` with models, API client, repository, Riverpod providers/notifiers, and tests. Do not wire rendered search UI, tabs, filter controls, or route behavior in this slice.

## 4. Candidate Approaches

### Option A: Dedicated `app/lib/search` data layer

Summary: Add `SearchApiClient`, `SearchRepository`, `ApiSearchRepository`, search models, and Riverpod providers/notifiers under the search feature boundary.

Pros:
- Keeps AppView search integration cohesive and discoverable.
- Matches existing Flutter architecture for API client, repository abstraction, models, and providers.
- Gives the future search UI a stable provider surface without touching widgets now.
- Avoids mixing committed search behavior into facet-autocomplete repositories.

Cons:
- Requires several new files plus `dart_mappable` and Riverpod generated files.
- Some result models wrap existing `Post` and `ProfileAccountSummary` shapes, so care is needed to avoid duplicating source-of-truth fields.

Risks:
- Provider contracts could overfit an imagined UI if they expose too much UI state.
- Recent-search payload typing must match AppView validation or the future UI will see avoidable 400s.

### Option B: Fold search methods into existing post/profile repositories

Summary: Add post/project/hashtag searches to `PostRepository`, profile search to `ProfileRepository`, and create small helpers for recent searches and top hashtags.

Pros:
- Reuses existing repository owners for post and profile models.
- Fewer new top-level abstractions.

Cons:
- Splits one search feature across unrelated repositories.
- Leaves recent searches and top hashtags without a natural home.
- Makes future search UI orchestration harder because result sources are scattered.

Risks:
- Scope creep into feed/profile provider behavior and tests.

### Option C: API client only

Summary: Add a `SearchApiClient` and models, but defer repository interfaces and Riverpod providers to the UI slice.

Pros:
- Smallest immediate implementation.
- Useful for proving endpoint decoding before state management.

Cons:
- Does not satisfy the requested “services, providers, etc.” scope.
- Defers pagination, recent-search mutations, and provider contracts into the UI work.

Risks:
- The later UI slice may need to revise the client/model choices once providers are added.

## 5. Recommended Direction

Recommended approach: Option A — dedicated `app/lib/search` data layer.

Why: It matches the confirmed user choice, keeps search-specific AppView integration cohesive, mirrors established app data-layer conventions, and produces a UI-ready provider/repository contract without implementing UI.

## 6. Problem / Opportunity

The AppView can now serve search results, top hashtags, and private recent searches, but the Flutter app has no data-layer surface to consume those endpoints. A non-UI data-layer slice lets the app safely decode, fetch, paginate, save, and delete search data before visual search components are built.

## 7. Goals

- G-001: Provide Flutter models for all AppView search response and request payloads needed by the future search UI.
- G-002: Provide a Dio-backed search API client that covers every AppView search endpoint implemented in the prior slice.
- G-003: Provide a repository abstraction and production AppView-backed implementation for testable search consumption.
- G-004: Provide Riverpod providers/notifiers for paginated search results, top hashtags, and recent-search lifecycle operations.
- G-005: Keep the slice non-UI and compatible with existing AppView auth, error, pagination, mapper, and testing conventions.

## 8. Non-Goals

- NG-001: Do not implement rendered search UI, tabs, input fields, filters, blank-state widgets, result cards, or scrolling screens.
- NG-002: Do not modify `SearchPage` behavior except where unavoidable for compile-time provider imports; no user-visible UI change is expected.
- NG-003: Do not change Flutter routes, deep links, app shell navigation, or hashtag facet navigation behavior.
- NG-004: Do not change AppView search endpoints, migrations, SQL, handlers, tests, or API contracts.
- NG-005: Do not change atproto lexicons or write search/recent-search records to a PDS.
- NG-006: Do not replace or alter existing `/v1/facets/*` autocomplete repositories.
- NG-007: Do not add local persistent search history in Flutter.
- NG-008: Do not add analytics, telemetry events, or a new dependency in this slice.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Future search UI | Flutter widgets that will be implemented in a later slice. | Needs stable models/providers for result pages, top hashtags, filters, and recent searches. |
| Signed-in Craftsky user | A user who will later interact with search. | Needs their future app experience to use authenticated AppView search and private recent-search state. |
| Flutter developer/test designer | Maintains app data flows and tests. | Needs clear API-client, repository, model, and provider contracts with fakeable abstractions. |
| AppView API | Existing backend search surface. | Expects authenticated requests, supported query parameters, typed recent payloads, and opaque cursor reuse. |

## 10. Current Behavior

The Flutter app has a placeholder `SearchPage` and no search data-layer code. Existing feed/profile/notification surfaces demonstrate API client, repository, model, and provider patterns, but none call `/v1/search/*`. Facet suggestion repositories call `/v1/facets/*` for autocomplete only. Search recents, top hashtags, profile search, post search, project search, and exact hashtag result search are inaccessible from Flutter code.

## 11. Desired Behavior

The Flutter app exposes a dedicated non-UI search data layer. Feature code can call a `SearchRepository` or Riverpod providers to fetch and paginate exact hashtag, profile, post, and project search results; fetch grouped top hashtags; list recent searches; explicitly save committed searches; and delete recent searches. The implementation uses existing authenticated Dio/error handling, decodes AppView camelCase JSON into typed Dart models, keeps cursors opaque, and leaves all rendered UI behavior unchanged.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | The Flutter app shall expose AppView search data to future UI code without implementing the search UI in this slice. | The user requested non-UI search wiring. | Prompt, Q1 | AC-001, AC-016 |
| BR-002 | Business | Must | Future UI code shall be able to fetch exact hashtag, profile, general post, project, top-hashtag, and recent-search data through a cohesive search feature boundary. | Search is one product feature even though result types vary. | Prompt, Discovery | AC-002, AC-003, AC-004, AC-005, AC-006 |
| BR-003 | Business | Must | Future UI code shall be able to explicitly save and delete private recent searches through AppView. | AppView recent searches were designed as private explicit-save state. | Prior AppView requirements | AC-007, AC-008 |
| FR-001 | Functional | Must | The system shall add typed Dart models for search post pages, profile search pages/items, top hashtag groups/items, recent-search items/pages, supported search sort values, project search filters, and recent-search save payloads. | The app needs typed request/response contracts for search providers and tests. | Discovery | AC-002, AC-003, AC-004, AC-005, AC-006, AC-007, AC-010 |
| FR-002 | Functional | Must | Search post/project/hashtag result page models shall reuse the existing `Post` model for `items` rather than defining a parallel post-result model. | AppView search returns the same core post response shape as timelines/profiles. | Prior AppView requirements, Codebase | AC-002, AC-011 |
| FR-003 | Functional | Must | Profile search result models shall reuse or embed the existing `ProfileAccountSummary` shape and include the AppView `viewerIsFollowing` field. | AppView profile search rows extend profile summary fields with followed state. | AppView `search_response.go` | AC-003 |
| FR-004 | Functional | Must | The system shall add a Dio-backed `SearchApiClient` using the shared authenticated `dioProvider` and `unwrapApi` error handling. | Existing AppView clients use this pattern for session/device headers and mapped exceptions. | Codebase | AC-009, AC-012 |
| FR-005 | Functional | Must | `SearchApiClient` shall implement methods for `GET /v1/search/hashtags/{tag}/posts`, `GET /v1/search/profiles`, `GET /v1/search/posts`, `GET /v1/search/projects`, `GET /v1/search/hashtags/top`, `GET /v1/search/recent`, `POST /v1/search/recent`, and `DELETE /v1/search/recent/{id}`. | Covers the full AppView search foundation needed by the app. | Prior AppView implementation | AC-002, AC-003, AC-004, AC-005, AC-006, AC-007, AC-008 |
| FR-006 | Functional | Must | Search API methods shall pass optional `limit` and `cursor` parameters opaquely for paginated endpoints and shall not inspect or synthesize cursor contents. | AppView cursors are opaque and existing app patterns pass them through. | API conventions, Codebase | AC-010, AC-013 |
| FR-007 | Functional | Must | Search API methods shall support `sort=chronological|popular` only for hashtag, post, and project result endpoints; profile search shall not expose chronology/popularity sort. | Mirrors AppView validation and avoids unsupported profile sorts. | Prior AppView requirements | AC-002, AC-003, AC-004, AC-014 |
| FR-008 | Functional | Must | Project search request modeling shall support optional `q`, optional sort, and supported filter families: `craftType`, `projectType`, `patternDifficulty`, `color`, `material`, `designTag`, and `projectTag`, including repeated values per family. | Future project filter UI needs these AppView-backed filters. | Prior AppView requirements, AppView `search_request.go` | AC-004, AC-015 |
| FR-009 | Functional | Must | Recent-search save modeling shall support AppView types `hashtag`, `profile`, `post`, and `project` with rerunnable typed payloads and a display label. | The app must save committed searches in the shape AppView validates. | Prior AppView requirements, AppView `search_request.go` | AC-007, AC-014 |
| FR-010 | Functional | Must | The system shall add a `SearchRepository` interface and `ApiSearchRepository` production implementation that delegates to `SearchApiClient`. | Existing app features depend on fakeable repositories. | Codebase, Q1 | AC-017 |
| FR-011 | Functional | Must | The system shall add keep-alive Riverpod providers for `SearchApiClient` and `SearchRepository`. | Matches app dependency-injection conventions. | Codebase | AC-017 |
| FR-012 | Functional | Must | The system shall add Riverpod provider/notifier surfaces for initial fetch and `loadMore()` pagination of hashtag, profile, post, and project search result pages. | Future UI needs ready-to-consume paginated search state. | Prompt, Codebase | AC-018, AC-019 |
| FR-013 | Functional | Must | Paginated search providers shall accumulate pages, preserve existing data during load-more transitions per Riverpod 3 behavior, and de-duplicate appended results by stable identity where a stable identity exists. | Prevents duplicate result cards across cursor pages and follows existing timeline patterns. | Codebase | AC-019, AC-020 |
| FR-014 | Functional | Must | The system shall add Riverpod provider/notifier surfaces for top hashtags and recent searches, including explicit save and delete mutations that refresh or update recent-search state after successful mutation. | Future blank search and recents UI need non-UI state surfaces. | Prompt, Prior AppView requirements | AC-005, AC-006, AC-007, AC-008, AC-021 |
| FR-015 | Functional | Must | Recent-search result models shall preserve AppView `payload` contents in a rerunnable typed representation for all supported recent types. | The UI must be able to rerun saved searches. | Prior AppView requirements | AC-006, AC-007, AC-014 |
| FR-016 | Functional | Must | New mappable models and generated mapper files shall be integrated with app mapper initialization where required by decoding/tests. | Existing model decoding depends on `dart_mappable` registration. | Codebase | AC-022 |
| NFR-001 | Non-functional | Must | The slice shall be limited to Flutter search data-layer files, generated Dart files, and search tests; it shall not change source under `appview/`, `lexicon/`, or UI widgets/routes. | Maintains requested non-UI scope and avoids backend churn. | Prompt, Q1 | AC-001, AC-016 |
| NFR-002 | Non-functional | Must | Search data-layer calls shall use existing authenticated Dio interceptors and shall not bypass AppView or call a PDS directly. | Project architecture requires app reads through AppView and no PDS tokens on device. | AGENTS.md, Codebase | AC-009, AC-012 |
| NFR-003 | Non-functional | Must | API-client methods shall surface AppView errors as existing sealed `ApiException` subtypes rather than raw `DioException`s. | Existing app callers expect mapped API exceptions. | Codebase | AC-012 |
| NFR-004 | Non-functional | Should | Provider/state types should remain UI-agnostic and avoid embedding visual tab, widget, or layout concerns. | Keeps this slice reusable by multiple future UI layouts. | Q1 | AC-018 |
| NFR-005 | Non-functional | Should | Tests should cover API request paths/query/body shapes, response decoding, repository delegation, provider pagination, and recent-search mutation state. | The test-design stage needs clear non-UI verification targets. | Discovery | AC-023 |
| RULE-001 | Business rule | Must | Search result endpoints shall not automatically save recent searches; only explicit recent-search save calls shall mutate recents. | AppView search foundation requires explicit save to avoid noisy recents. | Prior AppView requirements | AC-021 |
| RULE-002 | Business rule | Must | Flutter shall not persist recent-search history locally or write it to PDS records in this slice. | Recent searches are private AppView state. | AGENTS.md, Prior AppView requirements | AC-008 |
| RULE-003 | Business rule | Must | Facet autocomplete repositories and `/v1/facets/*` behavior shall remain separate from committed search-result repositories/providers. | Autocomplete and committed search have different semantics. | Prior AppView requirements, Codebase | AC-016 |
| RULE-004 | Business rule | Must | Opaque recent-search IDs and cursors returned by AppView shall be treated as server-owned strings and not parsed for client logic. | Avoids coupling Flutter to AppView implementation internals. | API conventions | AC-010, AC-013 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, NFR-001 | Given the slice is implemented, when app widgets/pages/routes are inspected, then no user-visible search UI behavior, app-shell navigation, or route semantics have changed. |
| AC-002 | BR-002, FR-001, FR-002, FR-005, FR-007 | Given a mocked AppView hashtag search response with `hashtag`, `items`, and `cursor`, when the search API client/repository requests hashtag results with tag, sort, limit, and cursor, then it calls `/v1/search/hashtags/{tag}/posts`, decodes existing `Post` items, preserves the canonical hashtag and cursor, and passes supported sort values. |
| AC-003 | BR-002, FR-001, FR-003, FR-005, FR-007 | Given a mocked profile search response with profile summary fields, `viewerIsFollowing`, and `cursor`, when profile search is requested with `q`, limit, and cursor, then it calls `/v1/search/profiles`, decodes profile summaries with followed state, and exposes no chronology/popularity sort parameter. |
| AC-004 | BR-002, FR-001, FR-005, FR-007, FR-008 | Given a project search request with query, sort, repeated filters, limit, and cursor, when the API client sends the request, then it calls `/v1/search/projects` with the documented query parameters and decodes a `Post` item page. |
| AC-005 | BR-002, FR-001, FR-005, FR-014 | Given a mocked top-hashtags response with craft groups and tag counts, when top hashtags are requested with optional craft types and limit, then it calls `/v1/search/hashtags/top` and decodes every group and item. |
| AC-006 | BR-002, FR-001, FR-005, FR-014, FR-015 | Given a mocked recent-search list response, when recent searches are listed, then it calls `/v1/search/recent` and decodes items with opaque id, type, displayLabel, typed payload, and updatedAt. |
| AC-007 | BR-003, FR-005, FR-009, FR-014, FR-015 | Given a committed hashtag/profile/post/project search payload and display label, when the app explicitly saves the recent search, then it POSTs `/v1/search/recent` with the expected type, displayLabel, and rerunnable payload and decodes the saved item. |
| AC-008 | BR-003, FR-005, FR-014, RULE-002 | Given a recent-search ID, when delete is requested, then the client calls `DELETE /v1/search/recent/{id}`, treats `204 No Content` as success, and does not modify PDS or local persistent search-history storage. |
| AC-009 | FR-004, NFR-002 | Given the production providers are used, when `SearchApiClient` is constructed, then it uses the shared authenticated `dioProvider` rather than creating an unauthenticated Dio or PDS client. |
| AC-010 | FR-001, FR-006, RULE-004 | Given paginated search responses include opaque cursors, when pages are decoded and subsequent pages are requested, then cursors are stored and resent as strings without client parsing. |
| AC-011 | FR-002 | Given post/project/hashtag search items include fields already supported by `PostMapper`, when decoded through search models, then they produce existing `Post` objects without a separate duplicate post DTO. |
| AC-012 | FR-004, NFR-002, NFR-003 | Given AppView returns a validation, unauthorized, server, network, or cancel error through Dio, when a search API method is called, then callers receive the existing mapped `ApiException` subtype behavior. |
| AC-013 | FR-006, RULE-004 | Given an invalid cursor causes AppView to return an error envelope, when the API client receives it, then it surfaces the mapped API error and does not attempt cursor recovery or parsing. |
| AC-014 | FR-007, FR-009, FR-015 | Given typed recent-search payload models for hashtag, profile, post, and project searches, when serialized for save, then they match AppView-supported payload keys and do not include unsupported profile sort or unknown project filter keys. |
| AC-015 | FR-008 | Given repeated project filter values are supplied, when serialized as query parameters, then repeated values are preserved for each supported filter family rather than collapsed into one comma-separated string unless AppView explicitly supports that shape. |
| AC-016 | BR-001, NFR-001, RULE-003 | Given existing facet suggestion tests and search page stub tests run, when the search data layer is added, then existing `/v1/facets/*` autocomplete and `SearchPage` placeholder behavior remain compatible. |
| AC-017 | FR-010, FR-011 | Given a provider container, when search repository providers are read, then `searchApiClientProvider` and `searchRepositoryProvider` produce the production API client/repository and can be overridden in tests. |
| AC-018 | FR-012, NFR-004 | Given a future UI watches search result providers, when initial provider builds occur, then they fetch through `SearchRepository` and expose UI-agnostic async state objects for results, cursor, and `hasMore`. |
| AC-019 | FR-012, FR-013 | Given a search result provider has loaded a first page with a next cursor, when `loadMore()` succeeds, then the provider appends the next page, updates the cursor, and preserves prior items during the loading transition. |
| AC-020 | FR-013 | Given a load-more response repeats a result already present in the provider state, when stable identity is available (`Post.uri`, profile DID/handle, or recent/top item identity as applicable), then the accumulated state avoids duplicate entries. |
| AC-021 | FR-014, RULE-001 | Given result search providers are fetched or paginated, when recent searches are listed afterward, then recents have not changed unless the explicit save mutation provider was called; after save/delete succeeds, recent-search state is refreshed or updated. |
| AC-022 | FR-016 | Given new mappable search models exist, when mapper initialization and code generation are run, then generated mapper files are present and search model decoding works in tests without missing mapper registration errors. |
| AC-023 | NFR-005 | Given focused search tests are run, when the test suite exercises API clients, models, repositories, providers, pagination, and recent mutations, then Must requirements are covered by automated tests or explicitly documented test gaps. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Hashtag value contains `#`, spaces, mixed casing, or characters that require URL encoding. | Client preserves caller-provided display/search value enough to call the path safely; AppView performs canonical validation/normalization and mapped errors surface through `ApiException`. | FR-005, NFR-003 |
| EC-002 | Profile search is requested with an empty/blank query by future UI code. | Data layer may send the request only if provider/model validation allows it; AppView validation errors surface as mapped API errors. | FR-004, FR-005, NFR-003 |
| EC-003 | Project search has no query and no filters. | Request is allowed and maps to AppView browse-all project semantics. | FR-008 |
| EC-004 | AppView omits `cursor` on a final page. | Model state treats `cursor == null` as no more pages. | FR-006, FR-012 |
| EC-005 | `loadMore()` is called while an initial load or another load-more is in progress. | Provider does not issue duplicate concurrent pagination requests. | FR-012, FR-013 |
| EC-006 | AppView returns an item with malformed JSON shape. | Decoding fails through normal model/API error paths; the data layer does not silently drop malformed committed search results unless a provider explicitly documents a tolerant behavior. | FR-001, NFR-003 |
| EC-007 | Recent-search delete targets an already-deleted or not-owned ID. | Client treats AppView's idempotent success response as success and avoids exposing ownership assumptions. | FR-014, RULE-004 |
| EC-008 | Recent-search payload is `type=project` with filter key ordering differences. | Typed payload serialization should be deterministic enough for tests and compatible with AppView normalization; AppView remains source of truth for de-duplication. | FR-009, FR-015 |
| EC-009 | Top hashtags are requested without craft types. | Request omits `craftTypes` and decodes AppView's default all-supported-groups response. | FR-005, FR-014 |
| EC-010 | Provider is disposed/recreated for the same search parameters. | New provider instance fetches from repository using the same parameters; no local persistent cache is required. | FR-012, RULE-002 |

## 15. Data / Persistence Impact

- New fields:
  - New Dart-only search models are expected under `app/lib/search/models/`, such as result page wrappers, top hashtag groups/items, recent-search items, recent-search payload types, project search filters, and search sort enum/value types.
  - New provider state models may be added for paginated search lists if existing `PostPage`/profile page wrappers are not sufficient.
- Changed fields:
  - Existing `Post`, `ProfileAccountSummary`, `Project`, and facet suggestion models should remain compatible and unchanged unless a minimal additive mapper registration/import is required.
  - `bootstrap.initializeMappers()` may need to register new search mappers.
- Migration required:
  - No database or AppView migration.
  - Dart code generation is required for new `dart_mappable` models and Riverpod providers.
- Backwards compatibility:
  - Existing feed/profile/notification/facet/search-page stub tests should continue to pass.
  - Existing AppView API contracts are consumed as-is.

## 16. UI / API / CLI Impact

- UI:
  - No rendered UI behavior change.
  - Future UI will consume the new repository/providers but is not implemented here.
- API:
  - Flutter begins consuming existing AppView `/v1/search/*` endpoints.
  - No AppView route or wire-contract change.
  - Query/body serialization must match AppView's implemented parameter and JSON payload names.
- CLI:
  - No CLI behavior change.
- Background jobs:
  - No background job or polling behavior is required.

## 17. Security / Privacy / Permissions

- Authentication:
  - Search calls use the existing authenticated `dioProvider`, including session token and device ID behavior.
- Authorization:
  - AppView remains responsible for enforcing authenticated viewer scope, moderation, and recent-search ownership.
- Sensitive data:
  - Recent-search history is private AppView state; Flutter must not write it to PDS or local persistent storage in this slice.
  - Tests/logging should avoid printing full recent-search payloads unnecessarily.
- Abuse cases:
  - AppView enforces query length and result limits. Flutter should pass only supported parameters and avoid client-side cursor parsing or endpoint bypass.

## 18. Observability

- Events:
  - None required.
- Logs:
  - No new diagnostic logging is required. If added, logs should avoid full recent-search payloads.
- Metrics:
  - None required in Flutter for this slice.
- Alerts:
  - None required.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Provider contracts may accidentally encode UI decisions before UI design is finalized. | Future UI slice may need data-layer churn. | Keep providers UI-agnostic: parameters in, async result state out; avoid tabs/layout concepts. |
| RISK-002 | Recent-search payload models may drift from AppView validation. | Save calls could fail at runtime even if the UI appears correct. | Model typed payloads from `appview/internal/api/search_request.go` and cover serialization in API-client tests. |
| RISK-003 | Duplicate model definitions could diverge from existing `Post` and profile summary contracts. | Search results may decode differently than timeline/profile cards. | Reuse existing `Post` and `ProfileAccountSummary` models where possible. |
| RISK-004 | Mappable/Riverpod generated files can be missed or stale. | Build/test failures or runtime mapper errors. | Require codegen and tests; include mapper initialization in acceptance coverage. |
| RISK-005 | Error-handling behavior could bypass existing `ApiException` mapping. | Future UI would need special-case search errors. | Require `unwrapApi` and focused API-client error tests. |
| RISK-006 | Search slice could expand into UI work. | Larger review surface and unclear stage boundaries. | Treat all widgets/routes as out of scope and preserve existing search-page tests. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The AppView search implementation and route contracts from `2026-06-19-appview-search-foundation` are source of truth for Flutter request/response shapes. | Requirements would need to change if AppView contracts are revised before Flutter integration. |
| ASM-002 | Existing `Post` can decode all post/project/hashtag search item fields returned by AppView. | A new or adjusted post model field may be required. |
| ASM-003 | Existing `ProfileAccountSummary` can represent profile search rows when combined with `viewerIsFollowing`. | A dedicated profile search summary model may need additional fields. |
| ASM-004 | Future UI will explicitly call save-recent mutations only for committed searches. | Data-layer provider behavior may need additional helper methods or events if the UI wants auto-save. |
| ASM-005 | Local persistent caching of search results/recents is unnecessary for this slice. | Additional storage requirements would be needed for offline or cache-first UX. |
| ASM-006 | No new package dependency is needed; existing Dio, Riverpod, dart_mappable, and test tools are sufficient. | Dependency review would be required if a new client-side search/cache abstraction is introduced. |

## 21. Open Questions

None.

## 22. Review Status

Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer:
Date:
Notes: Medium risk because this adds a new generated Flutter data layer, multiple endpoint contracts, paginated providers, and private recent-search mutations. Review is recommended before test design but not required by policy.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-20-flutter-search-data-layer/01-requirements.md`
- Next test specification: `docs/changes/2026-06-20-flutter-search-data-layer/02-acceptance-tests.md`
- Must-cover requirement IDs:
  - Business: `BR-001`, `BR-002`, `BR-003`
  - Functional: `FR-001` through `FR-016`
  - Non-functional Must: `NFR-001`, `NFR-002`, `NFR-003`
  - Rules: `RULE-001`, `RULE-002`, `RULE-003`, `RULE-004`
- Suggested test levels:
  - Model/unit tests for search response/request payload decoding and serialization.
  - API-client tests with `http_mock_adapter` for every `/v1/search/*` endpoint, query parameters, request bodies, response decoding, and error mapping.
  - Repository tests for delegation and fakeability.
  - Riverpod provider tests for initial loads, `loadMore()`, de-duplication, top hashtags, recent save/delete refresh/update behavior, and no auto-save on result fetch.
  - Regression tests for existing `SearchPage` stub and facet autocomplete repository behavior.
  - Codegen/build checks: `dart run build_runner build --delete-conflicting-outputs`, focused `flutter test test/search`, and broader `flutter test` / `flutter analyze` as appropriate.
- Blocking open questions: None.
