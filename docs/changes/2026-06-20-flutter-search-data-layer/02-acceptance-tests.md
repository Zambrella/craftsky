# Acceptance Test Specification: Flutter Search Data Layer

## 1. Test Strategy

This specification verifies the Flutter-only, non-UI search data-layer slice. The goal is to make the later search UI able to consume AppView search safely through typed models, a Dio-backed API client, repository abstractions, and Riverpod providers without changing rendered widgets, routes, AppView code, lexicons, dependencies, or local persistence.

The slice remains **medium risk** because it adds generated Flutter model/provider code, covers eight `/v1/search/*` endpoint contracts, introduces cursor-accumulating providers, and handles private recent-search mutations. Review is recommended before implementation continues, but the risk level does not require blocking approval.

Test design emphasizes Flutter automation under `app/test/search/`:

1. Model/unit tests for Dart decoding/serialization, supported enum values, project-filter query shape, recent-search typed payloads, opaque cursors, mapper registration, and de-duplication helpers.
2. API-client tests with `http_mock_adapter` for every `/v1/search/*` endpoint, query/path/body serialization, response decoding, no unsupported profile sort parameter, `204 No Content` delete handling, and mapped `ApiException` errors via `unwrapApi`.
3. Repository/provider tests with fake repositories and `ProviderContainer.test()` for provider construction, overrideability, initial loads, `loadMore()`, cursor accumulation, previous-data preservation, duplicate suppression, top hashtags, recent search list/save/delete state, and no implicit recent-save side effect from result fetches.
4. Regression tests protecting the existing `SearchPage` placeholder behavior, tag route context, and `/v1/facets/*` autocomplete repositories.
5. Manual checks limited to static review of generated-file scope, mapper initialization, no local/PDS recent-search persistence, UI-agnostic provider contracts, and no unintended source changes outside the Flutter search data layer.

Discovered commands:

- From `app/`: `dart run build_runner build --delete-conflicting-outputs` after adding `dart_mappable` and Riverpod files.
- From `app/`: focused `flutter test test/search` for the new slice.
- From `app/`: broader `flutter test` and `flutter analyze` before handoff.

Existing relevant test conventions:

- API-client tests use `Dio(BaseOptions(baseUrl: ...))`, `ErrorMappingInterceptor`, `http_mock_adapter`, and `setUpAll(initializeMappers)` where mapper-backed models are decoded.
- Provider tests use fake repositories plus `ProviderContainer.test(overrides: [...])`, following `app/test/feed/providers/timeline_provider_test.dart`.
- Existing search regression coverage starts in `app/test/search/search_page_test.dart`.
- Existing facet autocomplete tests live in `app/test/shared/rich_text/facet_suggestion_repository_test.dart` and must stay separate from committed search result tests.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-016 | AT-001, REG-001, REG-002, REG-003, MAN-002 | Acceptance / Regression / Manual | Mixed |
| BR-002 | AC-002, AC-003, AC-004, AC-005, AC-006 | AT-002, AT-003, AT-004, AT-005, IT-002, IT-003, IT-004, IT-005, IT-006, IT-007, UT-002, UT-003, UT-004 | Acceptance / Integration / Unit | Yes |
| BR-003 | AC-007, AC-008 | AT-006, IT-008, IT-009, IT-014, UT-005, MAN-003 | Acceptance / Integration / Unit / Manual | Mixed |
| FR-001 | AC-002, AC-003, AC-004, AC-005, AC-006, AC-007, AC-010 | AT-002, AT-003, AT-004, AT-005, AT-006, UT-001, UT-002, UT-003, UT-004, UT-005, UT-007, IT-002, IT-003, IT-004, IT-005, IT-006, IT-007, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-002, AC-011 | AT-002, UT-002, IT-002, IT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-003 | AT-003, UT-003, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-009, AC-012 | AT-001, IT-001, IT-010, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-005 | AC-002, AC-003, AC-004, AC-005, AC-006, AC-007, AC-008 | AT-002, AT-003, AT-004, AT-005, AT-006, IT-002, IT-003, IT-004, IT-005, IT-006, IT-007, IT-008, IT-009 | Acceptance / Integration | Yes |
| FR-006 | AC-010, AC-013 | AT-002, AT-003, AT-004, UT-007, IT-002, IT-003, IT-004, IT-005, IT-010, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-002, AC-003, AC-004, AC-014 | AT-002, AT-003, AT-004, UT-001, UT-005, IT-002, IT-003, IT-004, IT-005, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-004, AC-015 | AT-004, UT-006, IT-005, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-007, AC-014 | AT-006, UT-005, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-017 | AT-001, IT-011, IT-012, IT-014 | Acceptance / Integration | Yes |
| FR-011 | AC-017 | AT-001, IT-001 | Acceptance / Integration | Yes |
| FR-012 | AC-018, AC-019 | AT-007, IT-012, IT-013 | Acceptance / Integration | Yes |
| FR-013 | AC-019, AC-020 | AT-007, UT-008, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-005, AC-006, AC-007, AC-008, AC-021 | AT-005, AT-006, IT-006, IT-007, IT-008, IT-009, IT-014 | Acceptance / Integration | Yes |
| FR-015 | AC-006, AC-007, AC-014 | AT-005, AT-006, UT-005, IT-007, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-022 | UT-009, IT-015, MAN-001 | Unit / Integration / Manual | Mixed |
| NFR-001 | AC-001, AC-016 | AT-001, REG-001, REG-002, REG-003, REG-004, MAN-002 | Acceptance / Regression / Manual | Mixed |
| NFR-002 | AC-009, AC-012 | AT-001, IT-001, IT-010, MAN-003 | Acceptance / Integration / Manual | Mixed |
| NFR-003 | AC-012 | IT-010 | Integration | Yes |
| NFR-004 | AC-018 | AT-007, IT-012, MAN-004 | Acceptance / Integration / Manual | Mixed |
| NFR-005 | AC-023 | UT-001, UT-002, UT-003, UT-004, UT-005, UT-006, UT-007, UT-008, UT-009, IT-001, IT-002, IT-003, IT-004, IT-005, IT-006, IT-007, IT-008, IT-009, IT-010, IT-011, IT-012, IT-013, IT-014, IT-015, REG-001, REG-002, REG-003 | Unit / Integration / Regression | Yes |
| RULE-001 | AC-021 | AT-006, IT-014 | Acceptance / Integration | Yes |
| RULE-002 | AC-008 | AT-006, IT-009, MAN-003 | Acceptance / Integration / Manual | Mixed |
| RULE-003 | AC-016 | REG-003, MAN-002 | Regression / Manual | Mixed |
| RULE-004 | AC-010, AC-013 | AT-002, AT-003, AT-004, UT-007, IT-002, IT-003, IT-004, IT-005, IT-010 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Search Data Layer Is Non-UI And Uses Authenticated AppView Providers

Requirement IDs: BR-001, FR-004, FR-010, FR-011, NFR-001, NFR-002
Acceptance Criteria: AC-001, AC-009, AC-012, AC-016, AC-017
Priority: Must
Level: Acceptance
Automation Target: `app/test/search/data/search_api_client_test.dart`, `app/test/search/data/search_repository_test.dart`, `app/test/search/providers/search_repository_provider_test.dart`, `app/test/search/search_page_test.dart`

```gherkin
Feature: Flutter search data-layer boundary
  Scenario: Production search providers expose authenticated AppView data access without changing UI
    Given the Flutter app has the existing shared authenticated dioProvider
    And the existing SearchPage renders only its placeholder title/body and optional hashtag context
    When a provider container reads searchApiClientProvider and searchRepositoryProvider
    Then the SearchApiClient is constructed from the shared dioProvider
    And the SearchRepository is backed by ApiSearchRepository
    And both providers can be overridden by tests
    And no PDS client or unauthenticated Dio is constructed by the search feature
    And SearchPage widgets, routes, and app-shell navigation remain unchanged
```

### AT-002: Hashtag Search Fetches Post Pages With Opaque Pagination

Requirement IDs: BR-002, FR-001, FR-002, FR-005, FR-006, FR-007, RULE-004
Acceptance Criteria: AC-002, AC-010, AC-011, AC-013
Priority: Must
Level: Acceptance
Automation Target: `app/test/search/data/search_api_client_test.dart`, `app/test/search/providers/hashtag_search_provider_test.dart`

```gherkin
Feature: Hashtag search data access
  Scenario: Hashtag results decode as existing Post items and paginate by opaque cursor
    Given AppView returns a hashtag search page with hashtag "sockkal", two Post-shaped items, and cursor "opaque:next"
    When future UI code requests hashtag results for "SockKAL" with sort "popular", limit 20, and cursor "opaque:start"
    Then the API client calls GET /v1/search/hashtags/SockKAL/posts with sort, limit, and cursor query parameters
    And the response exposes canonical hashtag "sockkal"
    And items decode as existing Post objects rather than a duplicate post-result DTO
    And the cursor is stored and resent as an opaque string without parsing
    When AppView returns an invalid-cursor error envelope
    Then the mapped ApiException is surfaced and the client does not attempt cursor recovery
```

### AT-003: Profile Search Exposes Summary Results With Follow State And No Sort

Requirement IDs: BR-002, FR-001, FR-003, FR-005, FR-006, FR-007, RULE-004
Acceptance Criteria: AC-003, AC-010, AC-013
Priority: Must
Level: Acceptance
Automation Target: `app/test/search/data/search_api_client_test.dart`, `app/test/search/providers/profile_search_provider_test.dart`

```gherkin
Feature: Profile search data access
  Scenario: Profile search sends query pagination and decodes viewer follow state
    Given AppView returns profile search items with ProfileAccountSummary fields, viewerIsFollowing, and cursor "opaque:profiles"
    When future UI code requests profiles with q "ali", limit 25, and cursor "opaque:start"
    Then the API client calls GET /v1/search/profiles with q, limit, and cursor
    And no chronological or popular sort parameter is exposed or sent
    And each result includes the profile summary fields plus viewerIsFollowing
    And the cursor is stored and resent as an opaque string without parsing
```

### AT-004: Post And Project Search Send Supported Sorts And Project Filters

Requirement IDs: BR-002, FR-001, FR-002, FR-005, FR-006, FR-007, FR-008, RULE-004
Acceptance Criteria: AC-004, AC-010, AC-011, AC-013, AC-015
Priority: Must
Level: Acceptance
Automation Target: `app/test/search/data/search_api_client_test.dart`, `app/test/search/providers/post_search_provider_test.dart`, `app/test/search/providers/project_search_provider_test.dart`

```gherkin
Feature: Post and project search data access
  Scenario: Search requests serialize documented parameters and decode Post pages
    Given future UI code has a post search query "alpaca" sorted by chronological order
    When it requests general post search with limit and cursor
    Then the API client calls GET /v1/search/posts with q, sort, limit, and cursor
    And the page decodes existing Post items with an optional cursor
    Given future UI code has a project search with q, popular sort, and repeated craftType, material, designTag, and projectTag filters
    When it requests project search
    Then the API client calls GET /v1/search/projects with q, sort, limit, cursor, and repeated query values per filter family
    And repeated filter values are not collapsed into one comma-separated string
    And unsupported profile sort or unknown project filter keys are not produced by typed request models
```

### AT-005: Top Hashtags And Recent List Decode Private Search Metadata

Requirement IDs: BR-002, FR-001, FR-005, FR-014, FR-015
Acceptance Criteria: AC-005, AC-006
Priority: Must
Level: Acceptance
Automation Target: `app/test/search/data/search_api_client_test.dart`, `app/test/search/providers/top_hashtags_provider_test.dart`, `app/test/search/providers/recent_searches_provider_test.dart`

```gherkin
Feature: Top hashtags and recent searches
  Scenario: Search support providers expose grouped hashtags and typed recents
    Given AppView returns top hashtag groups for knitting and crochet with tag counts
    When top hashtags are requested with repeated craftTypes and limit
    Then the API client calls GET /v1/search/hashtags/top with repeated craftTypes and limit
    And each craft group and tag count is decoded
    Given AppView returns recent searches for hashtag, profile, post, and project types
    When recent searches are listed
    Then the API client calls GET /v1/search/recent
    And each item exposes an opaque id, type, displayLabel, updatedAt, and rerunnable typed payload
```

### AT-006: Recent Searches Change Only Through Explicit Save And Delete Mutations

Requirement IDs: BR-003, FR-005, FR-009, FR-014, FR-015, RULE-001, RULE-002
Acceptance Criteria: AC-007, AC-008, AC-014, AC-021
Priority: Must
Level: Acceptance
Automation Target: `app/test/search/data/search_api_client_test.dart`, `app/test/search/providers/recent_searches_provider_test.dart`

```gherkin
Feature: Explicit recent-search lifecycle
  Scenario: Result fetching does not auto-save, while save and delete mutate AppView recents
    Given result search providers have fetched hashtag, profile, post, and project results
    When recent searches are listed afterward
    Then the recent-search state has not changed from result fetching alone
    When future UI code explicitly saves a hashtag, profile, post, or project search with a display label
    Then the API client POSTs /v1/search/recent with type, displayLabel, and the rerunnable typed payload
    And the saved item is decoded and recent-search provider state is refreshed or updated
    When future UI code deletes an opaque recent-search ID
    Then the API client calls DELETE /v1/search/recent/{id}
    And 204 No Content is treated as success
    And no local persistent search-history storage or PDS record is written
```

### AT-007: Search Result Providers Are UI-Agnostic Cursor Accumulators

Requirement IDs: FR-012, FR-013, NFR-004
Acceptance Criteria: AC-018, AC-019, AC-020
Priority: Must
Level: Acceptance
Automation Target: `app/test/search/providers/hashtag_search_provider_test.dart`, `app/test/search/providers/profile_search_provider_test.dart`, `app/test/search/providers/post_search_provider_test.dart`, `app/test/search/providers/project_search_provider_test.dart`

```gherkin
Feature: UI-agnostic search provider pagination
  Scenario: Providers fetch initial pages, append more pages, and suppress duplicates
    Given a fake SearchRepository returns a first page with items and cursor "opaque:next"
    When a future UI watches a hashtag, profile, post, or project search provider
    Then the provider fetches through SearchRepository and exposes async state containing items, cursor, and hasMore
    And the state contains no visual tab, widget, layout, or route concepts
    When loadMore() succeeds
    Then the provider passes the opaque cursor to the repository, appends the next page, and updates cursor and hasMore
    And duplicate posts by Post.uri or duplicate profiles by DID/handle are not appended
    When loadMore() is requested while already loading, or hasMore is false
    Then no duplicate pagination request is issued
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-007, NFR-005 | AC-014, AC-023 | Search sort modeling supports only AppView result sorts and prevents profile sort leakage. | `SearchSort.chronological`, `SearchSort.popular`; profile-search request construction. | Hashtag/post/project requests serialize `sort=chronological` or `sort=popular`; profile request types expose no sort field. | `app/test/search/models/search_sort_test.dart` |
| UT-002 | BR-002, FR-001, FR-002, NFR-005 | AC-002, AC-011, AC-023 | Hashtag/post/project page models decode `items` as existing `Post`. | Post-shaped search page maps with `items`, optional `hashtag`, optional `cursor`. | Decoded page contains `Post` objects and preserves hashtag/cursor; no parallel post-result model is required. | `app/test/search/models/search_post_page_test.dart` |
| UT-003 | BR-002, FR-001, FR-003, NFR-005 | AC-003, AC-023 | Profile search models reuse/embed profile summary fields and include follow state. | Profile result map with `did`, `handle`, `displayName`, `description`, `avatar`, `isCraftskyProfile`, `viewerIsFollowing`. | Decoded item exposes `ProfileAccountSummary`-compatible values and `viewerIsFollowing`. | `app/test/search/models/profile_search_page_test.dart` |
| UT-004 | BR-002, FR-001, FR-014, NFR-005 | AC-005, AC-023 | Top hashtag response models decode groups and counts. | `groups` with `craftType`, `items`, `tag`, `count`; empty group. | Groups decode in order, counts default only if explicitly modeled, and empty groups remain empty arrays. | `app/test/search/models/top_hashtags_test.dart` |
| UT-005 | BR-003, FR-001, FR-009, FR-015, NFR-005 | AC-006, AC-007, AC-014, AC-023 | Recent-search item and save payload models serialize/deserialize supported typed payloads. | Hashtag `{tag, sort}`, profile `{q}`, post `{q, sort}`, project `{q, sort, filters}`. | `type`, `displayLabel`, `payload`, `id`, and `updatedAt` round-trip; profile payload has no sort; unknown project filter keys are not generated. | `app/test/search/models/recent_search_test.dart` |
| UT-006 | FR-008, NFR-005 | AC-004, AC-015, AC-023 | Project filter request modeling preserves supported repeated filter families. | `craftType`, `projectType`, `patternDifficulty`, `color`, `material`, `designTag`, `projectTag`, each with multiple values. | Query serialization keeps repeated values per family and does not collapse values into comma-separated strings. | `app/test/search/models/project_search_filters_test.dart` |
| UT-007 | FR-001, FR-006, RULE-004, NFR-005 | AC-010, AC-013, AC-023 | Page state treats cursors and recent IDs as opaque server strings. | Cursor values such as `opaque:abc/+/=` and `null`; recent ID `recent_01HT...`. | State stores/resends exact strings, derives `hasMore == false` when cursor is null, and performs no parsing. | `app/test/search/models/search_pagination_state_test.dart` |
| UT-008 | FR-013, NFR-005 | AC-020, AC-023 | De-duplication helpers suppress duplicates by stable identity. | Existing and next pages with repeated `Post.uri`, repeated profile DID/handle, repeated top/recent identity where modeled. | Merged state keeps first existing item and appends only new identities. | `app/test/search/providers/search_pagination_merge_test.dart` |
| UT-009 | FR-016, NFR-005 | AC-022, AC-023 | Mapper initialization includes new search mappers. | `initializeMappers()` followed by representative search model decode. | Decode works without missing mapper registration errors; generated mapper files are referenced. | `app/test/search/models/search_mapper_registration_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-004, FR-011, NFR-002, NFR-005 | AC-009, AC-012, AC-017, AC-023 | Search API and repository providers use the shared authenticated Dio and are overrideable. | `ProviderContainer.test()` with a fake or overridden `dioProvider`. | Read `searchApiClientProvider` and `searchRepositoryProvider`; override repository in a second container. | Production providers return `SearchApiClient`/`ApiSearchRepository`; tests can override without source changes; no separate unauthenticated Dio or PDS client is needed. | `app/test/search/providers/search_repository_provider_test.dart` |
| IT-002 | BR-002, FR-001, FR-002, FR-005, FR-006, FR-007, RULE-004, NFR-005 | AC-002, AC-010, AC-011, AC-013, AC-023 | Hashtag API client path/query/decoding/error behavior. | `Dio` + `ErrorMappingInterceptor` + `DioAdapter`; mocked hashtag response and error envelope. | Call hashtag search with tag, sort, limit, cursor; call again against invalid cursor error. | Sends `/v1/search/hashtags/{tag}/posts`; preserves sort/limit/cursor; decodes `Post` items/hashtag/cursor; surfaces mapped `ApiException` on error. | `app/test/search/data/search_api_client_test.dart` |
| IT-003 | BR-002, FR-001, FR-003, FR-005, FR-006, FR-007, RULE-004, NFR-005 | AC-003, AC-010, AC-013, AC-023 | Profile API client query/decoding and no sort parameter. | Mocked profile page response with `viewerIsFollowing` and cursor. | Call profile search with `q`, `limit`, `cursor`. | Sends `/v1/search/profiles` with only q/limit/cursor; decodes profile summaries and follow state; no sort query key is sent. | `app/test/search/data/search_api_client_test.dart` |
| IT-004 | BR-002, FR-001, FR-002, FR-005, FR-006, FR-007, RULE-004, NFR-005 | AC-004, AC-010, AC-011, AC-013, AC-023 | General post API client request and decoding. | Mocked post search page response. | Call post search with q, supported sort, limit, cursor. | Sends `/v1/search/posts`; decodes existing `Post` items and opaque cursor. | `app/test/search/data/search_api_client_test.dart` |
| IT-005 | BR-002, FR-001, FR-002, FR-005, FR-006, FR-007, FR-008, RULE-004, NFR-005 | AC-004, AC-010, AC-011, AC-013, AC-015, AC-023 | Project API client request with repeated filters and browse-all support. | Mocked project search response; filters from every supported family. | Call project search with q/sort/limit/cursor/filters, then with no q and no filters. | Sends `/v1/search/projects` with documented query keys and repeated values; decodes `Post` page; omits q/filter keys for browse-all request. | `app/test/search/data/search_api_client_test.dart` |
| IT-006 | BR-002, FR-001, FR-005, FR-014, NFR-005 | AC-005, AC-023 | Top hashtags API client repeated craftTypes and decoding. | Mocked `groups` response. | Call top hashtags with multiple craftTypes and limit; call without craftTypes. | Sends `/v1/search/hashtags/top` with repeated `craftTypes` when provided, omits when absent, and decodes all groups/items. | `app/test/search/data/search_api_client_test.dart` |
| IT-007 | BR-002, FR-001, FR-005, FR-014, FR-015, NFR-005 | AC-006, AC-023 | Recent-search list API client decodes typed payloads. | Mocked `items` response for hashtag/profile/post/project types. | Call list recent searches. | Sends `/v1/search/recent`; decodes opaque IDs, display labels, payloads, and updatedAt values. | `app/test/search/data/search_api_client_test.dart` |
| IT-008 | BR-003, FR-005, FR-007, FR-008, FR-009, FR-014, FR-015, NFR-005 | AC-007, AC-014, AC-023 | Recent-search save API client serializes each supported type. | Mocked POST responses for four payload types. | Save hashtag, profile, post, and project recents. | POST body uses `type`, `displayLabel`, and AppView-compatible payload keys; profile payload excludes sort; project filters use supported keys. | `app/test/search/data/search_api_client_test.dart` |
| IT-009 | BR-003, FR-005, FR-014, RULE-002, NFR-005 | AC-008, AC-023 | Recent-search delete handles `204 No Content`. | Mocked DELETE response with status 204. | Delete an opaque recent-search ID. | Calls `/v1/search/recent/{id}` and completes successfully without local/PDS writes. | `app/test/search/data/search_api_client_test.dart` |
| IT-010 | FR-004, FR-006, NFR-002, NFR-003, RULE-004, NFR-005 | AC-012, AC-013, AC-023 | Search API client maps AppView, network, and cancel errors through existing `ApiException` behavior. | `Dio` with `ErrorMappingInterceptor`; mocked 400/401/500 envelopes plus network/cancel cases where practical. | Call representative search methods. | Callers receive existing `ApiException` subtypes instead of raw `DioException`; cursors are not parsed or recovered locally. | `app/test/search/data/search_api_client_error_test.dart` |
| IT-011 | FR-010, NFR-005 | AC-017, AC-023 | ApiSearchRepository delegates every method to SearchApiClient. | Fake or recording `SearchApiClient` where testable, or mocked HTTP client plus repository. | Invoke each repository method. | Arguments and returned values pass through unchanged, enabling fake repository tests. | `app/test/search/data/search_repository_test.dart` |
| IT-012 | FR-010, FR-012, NFR-004, NFR-005 | AC-017, AC-018, AC-023 | Result providers fetch initial state through SearchRepository and remain UI-agnostic. | `ProviderContainer.test()` overriding `searchRepositoryProvider` with a fake repository. | Read hashtag/profile/post/project provider futures. | Each provider calls the matching repository method with expected parameters and exposes async state with items, cursor, and hasMore only. | `app/test/search/providers/*_search_provider_test.dart` |
| IT-013 | FR-006, FR-012, FR-013, RULE-004, NFR-005 | AC-010, AC-013, AC-019, AC-020, AC-023 | Result providers implement `loadMore()` accumulation, previous-data preservation, de-duplication, and concurrency guards. | Fake repository returns page 1, page 2 with duplicate, failures, null cursor, and a completer-gated in-flight response. | Call `loadMore()` in success, failure, no-more, and concurrent paths. | Opaque cursor is passed through; items append without duplicates; prior value remains available during loading/error; no duplicate requests occur. | `app/test/search/providers/*_search_provider_test.dart` |
| IT-014 | FR-010, FR-014, RULE-001, NFR-005 | AC-017, AC-021, AC-023 | Top hashtags and recent providers handle fetch/save/delete and prove result fetches do not auto-save. | Fake repository records method calls and returns recents before/after mutation. | Fetch result providers, list recents, save recent, delete recent. | Result fetches do not call save; top/recent initial loads work; save/delete refresh or update recent state after success. | `app/test/search/providers/recent_searches_provider_test.dart`, `app/test/search/providers/top_hashtags_provider_test.dart` |
| IT-015 | FR-016, NFR-005 | AC-022, AC-023 | Code generation/build verification for mappable and Riverpod files. | Generated files are present after build_runner. | Run codegen and focused tests. | `*.mapper.dart` and `*.g.dart` files are up to date; focused search tests pass without mapper or provider generation errors. | Commands: `dart run build_runner build --delete-conflicting-outputs`, `flutter test test/search` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | `SearchPage` placeholder still renders the existing title/body and no result UI. | BR-001, NFR-001, NFR-005 | AC-001, AC-016, AC-023 | Keep/extend `app/test/search/search_page_test.dart` to assert title rendering and absence of newly introduced user-visible result widgets in this slice. |
| REG-002 | Search route tag query context remains compatible for hashtag facet taps. | BR-001, NFR-001, NFR-005 | AC-001, AC-016, AC-023 | Keep `SearchRoute(tag: 'SockKAL').location == '/search?tag=SockKAL'` and `SearchPage(tag: ...)` context test unchanged. |
| REG-003 | `/v1/facets/*` autocomplete repositories remain separate from committed search result repositories/providers. | BR-001, NFR-001, RULE-003, NFR-005 | AC-016, AC-023 | Run existing `app/test/shared/rich_text/facet_suggestion_repository_test.dart`; do not retarget facet tests to `/v1/search/*`. |
| REG-004 | Search slice does not change backend, lexicon, route, dependency, or AppView code. | FR-004, NFR-001 | AC-001, AC-009, AC-016 | Review `git diff -- appview lexicon app/pubspec.yaml app/pubspec.lock app/lib/search/pages app/lib/router` and ensure only allowed Flutter data-layer/generated/test docs changed unless explicitly justified. |
| REG-005 | Existing feed/profile post/profile model decoding remains compatible with added search models. | FR-002, FR-003, NFR-005 | AC-011, AC-023 | Run relevant existing model/API tests such as `app/test/feed/data/post_api_client_test.dart` and `app/test/profile/data/profile_api_client_test.dart` or the broader `flutter test`. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Post-shaped search result item reused across hashtag/post/project pages. | `uri`, `cid`, `rkey`, `text`, `tags`, engagement counts, viewer flags, `createdAt`, `indexedAt`, `author`, and optional `project`, matching existing `PostMapper` expectations. | AT-002, AT-004, UT-002, IT-002, IT-004, IT-005, IT-013 |
| TD-002 | Profile search item with follow state. | `did:plc:alice`, `handle: alice.craftsky.social`, `displayName`, `description`, `avatar`, `isCraftskyProfile: true`, `viewerIsFollowing: true`. | AT-003, UT-003, IT-003, IT-013 |
| TD-003 | Project filter request with repeated supported filter families. | `q: alpaca`, `sort: popular`, `craftType: [knitting, crochet]`, `projectType`, `patternDifficulty`, `color`, `material`, `designTag`, `projectTag`, `limit`, `cursor: opaque:projects`. | AT-004, UT-006, IT-005, IT-008 |
| TD-004 | Top hashtag groups and counts. | Groups for `knitting`, `crochet`, and an empty craft group; items like `{tag: sockkal, count: 12}`. | AT-005, UT-004, IT-006, IT-014 |
| TD-005 | Recent-search payloads for all supported types. | Hashtag `{tag: sockkal, sort: chronological}`; profile `{q: alice}`; post `{q: alpaca, sort: popular}`; project `{q: cardigan, sort: chronological, filters: {...}}`; display labels and opaque IDs. | AT-005, AT-006, UT-005, IT-007, IT-008, IT-014 |
| TD-006 | Standard AppView error envelopes. | `400 invalid_cursor`, validation error, `401 unauthorized`, `500 server_error`, each with `error`, `message`, `requestId`. | AT-002, IT-010 |
| TD-007 | Pagination and duplicate suppression fixtures. | Page 1 with cursor `opaque:next`; page 2 containing one repeated `Post.uri` or repeated profile identity plus one new item; final page with no cursor. | AT-007, UT-007, UT-008, IT-013 |
| TD-008 | Hashtag path-encoding edge values. | Caller-provided tag values such as `SockKAL`, `#SockKAL`, `fiber art`, `sock/kal`, and UTF-8 text. | AT-002, IT-002, GAP-002 |
| TD-009 | API exception and malformed response fixtures. | Error-mapped Dio responses and malformed JSON missing required model fields. | IT-010, GAP-003 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-016 | AC-022 | Generated files and mapper initialization are complete. | Inspect generated `*.mapper.dart` and `*.g.dart` files and `initializeMappers()` after codegen. | Search mappers/providers are generated, committed, and initialized where needed; no stale generated output remains. |
| MAN-002 | BR-001, NFR-001, RULE-003 | AC-001, AC-016 | Scope review for non-UI and autocomplete separation. | Inspect changed files and diffs for widgets/routes/facet repositories. | No rendered search UI, routes, app shell behavior, or `/v1/facets/*` autocomplete behavior was changed by this slice. |
| MAN-003 | BR-003, NFR-002, RULE-002 | AC-008, AC-009, AC-012 | Privacy/security review for recent searches. | Inspect code for PDS clients, token handling, local storage APIs, and logs involving recent payloads. | Search calls use AppView via shared Dio; no PDS writes or local persistent recent-search history are added; logs do not dump full private payloads. |
| MAN-004 | NFR-004 | AC-018 | Provider contract review for UI-agnostic shape. | Inspect public provider/state types. | Providers accept search parameters and expose async data/cursor/hasMore/mutation surfaces without visual tab/layout/widget concerns. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Flutter tests mock AppView rather than exercising a live AppView search stack. | BR-002, BR-003, FR-005 | This slice is Flutter-only; AppView behavior was covered by the prior AppView search foundation specification. | Rely on prior AppView tests for backend correctness; consider an end-to-end smoke test in a later UI or release-hardening slice. |
| GAP-002 | Hashtag path encoding edge cases may be implementation-sensitive. | FR-005, NFR-003 | Requirements allow AppView to validate/normalize; Flutter only needs to produce a safe path and mapped errors. | Add focused API-client tests for encoded path values; document any value that `http_mock_adapter` cannot assert exactly. |
| GAP-003 | Malformed successful JSON is only tested through representative decode failures, not every possible bad field shape. | FR-001, NFR-003 | Exhaustive schema-fuzz testing is out of scope for this Flutter data-layer slice. | Cover one malformed required-field case per response family; leave exhaustive fuzzing to future contract tooling if needed. |
| GAP-004 | Static guarantees that no PDS/local persistence is used are partly manual. | NFR-002, RULE-002 | There is no existing architectural lint that blocks new PDS or local storage usage inside feature code. | Manual privacy review in MAN-003; consider a later lint/check if this pattern repeats. |
| GAP-005 | Future UI may require provider contract adjustments despite UI-agnostic design. | FR-012, NFR-004 | The actual UI interaction model is intentionally out of scope. | Keep provider state minimal; revisit provider ergonomics during the UI slice without changing AppView contracts. |

## 10. Out Of Scope

- Rendered search UI, tabs, filters, result cards, empty states, scrolling pages, or route/deep-link changes.
- AppView route, handler, SQL, migration, indexer, lexicon, or API contract changes.
- Local persistent caching/offline search history in Flutter.
- Analytics, telemetry, or new diagnostic logging.
- New Flutter dependencies.
- Exhaustive live end-to-end search testing against Docker/AppView in this stage; use mocked AppView contracts plus prior backend tests.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-06-20-flutter-search-data-layer/01-requirements.md`
- Test specification: `docs/changes/2026-06-20-flutter-search-data-layer/02-acceptance-tests.md`
- Next review artifact: `docs/changes/2026-06-20-flutter-search-data-layer/03-document-review.md`
- External Plannotator review, if the user initiates it outside this agent: `docs/changes/2026-06-20-flutter-search-data-layer/`
- Risk level carried forward: **Medium**.
- Risk-based review recommendation: **Review recommended before implementation**, but implementation may proceed if the user explicitly chooses to skip review.
- Recommended first failing test for implementation: `IT-002` in `app/test/search/data/search_api_client_test.dart` for `SearchApiClient` hashtag search path/query/decoding/error behavior, because it drives initial search models, `unwrapApi`, cursor handling, and `Post` reuse.
- Suggested test order for implementation:
  1. `UT-001`, `UT-002`, `UT-007`, `UT-009` for base sort/page/cursor/mapper model scaffolding.
  2. `IT-002` through `IT-010` for API-client endpoint coverage and errors.
  3. `IT-011` and `IT-001` for repository delegation and provider construction.
  4. `IT-012`, `IT-013`, and `UT-008` for paginated result providers.
  5. `IT-014`, `UT-005`, and `IT-008`/`IT-009` for recent-search lifecycle.
  6. `REG-001` through `REG-005` and manual checks before implementation review.
- Commands discovered:
  - `cd app && dart run build_runner build --delete-conflicting-outputs`
  - `cd app && flutter test test/search`
  - `cd app && flutter test`
  - `cd app && flutter analyze`
- Blocking gaps: None. Documented non-blocking gaps: `GAP-001` through `GAP-005`.
