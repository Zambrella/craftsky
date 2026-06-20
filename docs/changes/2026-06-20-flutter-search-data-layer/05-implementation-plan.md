# TDD Implementation Plan: Flutter Search Data Layer

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Keep this slice Flutter-only and non-UI.
- Do not add local recent-search persistence, PDS calls, dependencies, AppView code, lexicon changes, route changes, or rendered UI behavior.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-002 | BR-002, FR-001, FR-002, FR-005, FR-006, FR-007, RULE-004 | AC-002, AC-010, AC-011, AC-013 | Fails: no SearchApiClient/models |
| 2 | UT-001 | FR-001, FR-007, NFR-005 | AC-014, AC-023 | Fails: no SearchSort |
| 3 | UT-002 | BR-002, FR-001, FR-002, NFR-005 | AC-002, AC-011, AC-023 | Fails: no SearchPostPage |
| 4 | UT-007 | FR-001, FR-006, RULE-004, NFR-005 | AC-010, AC-013, AC-023 | Fails: no search pagination state |
| 5 | UT-009 | FR-016, NFR-005 | AC-022, AC-023 | Fails: mappers not registered/generated |
| 6 | IT-003 | BR-002, FR-001, FR-003, FR-005, FR-006, FR-007, RULE-004 | AC-003, AC-010, AC-013, AC-023 | Fails: no profile search API/model |
| 7 | IT-004 | BR-002, FR-001, FR-002, FR-005, FR-006, FR-007, RULE-004 | AC-004, AC-010, AC-011, AC-013, AC-023 | Fails: no post search API method |
| 8 | IT-005 | BR-002, FR-001, FR-002, FR-005, FR-006, FR-007, FR-008, RULE-004 | AC-004, AC-010, AC-011, AC-013, AC-015, AC-023 | Fails: no project search filters/API |
| 9 | IT-006 | BR-002, FR-001, FR-005, FR-014, NFR-005 | AC-005, AC-023 | Fails: no top hashtag API/model |
| 10 | IT-007 | BR-002, FR-001, FR-005, FR-014, FR-015, NFR-005 | AC-006, AC-023 | Fails: no recent list API/model |
| 11 | IT-008 | BR-003, FR-005, FR-007, FR-008, FR-009, FR-014, FR-015, NFR-005 | AC-007, AC-014, AC-023 | Fails: no save recent API/payloads |
| 12 | IT-009 | BR-003, FR-005, FR-014, RULE-002, NFR-005 | AC-008, AC-023 | Fails: no delete recent API |
| 13 | IT-010 | FR-004, FR-006, NFR-002, NFR-003, RULE-004, NFR-005 | AC-012, AC-013, AC-023 | Fails until API methods wrap `unwrapApi` |
| 14 | IT-011 | FR-010, NFR-005 | AC-017, AC-023 | Fails: no SearchRepository |
| 15 | IT-001 | FR-004, FR-011, NFR-002, NFR-005 | AC-009, AC-012, AC-017, AC-023 | Fails: no production providers |
| 16 | IT-012 | FR-010, FR-012, NFR-004, NFR-005 | AC-017, AC-018, AC-023 | Fails: no result providers |
| 17 | IT-013 | FR-006, FR-012, FR-013, RULE-004, NFR-005 | AC-010, AC-013, AC-019, AC-020, AC-023 | Fails: no loadMore accumulation |
| 18 | UT-008 | FR-013, NFR-005 | AC-020, AC-023 | Fails: no merge helpers |
| 19 | IT-014 | FR-010, FR-014, RULE-001, NFR-005 | AC-017, AC-021, AC-023 | Fails: no top/recent providers |
| 20 | UT-004 | BR-002, FR-001, FR-014, NFR-005 | AC-005, AC-023 | Fails until top models complete |
| 21 | UT-005 | BR-003, FR-001, FR-009, FR-015, NFR-005 | AC-006, AC-007, AC-014, AC-023 | Fails until recent models complete |
| 22 | UT-006 | FR-008, NFR-005 | AC-004, AC-015, AC-023 | Fails until filter models complete |
| 23 | REG-001..REG-005 | BR-001, NFR-001, RULE-003 | AC-001, AC-016, AC-023 | Existing behavior should remain green |
| 24 | MAN-001..MAN-004 | FR-016, BR-001, NFR-002, NFR-004 | AC-001, AC-008, AC-009, AC-016, AC-018, AC-022 | Manual diff/static checks required |

## Implementation Steps

### Step 1: IT-002
- Write failing test: Added `app/test/search/data/search_api_client_test.dart` coverage for hashtag path/query/decoding, safe path-segment encoding, and invalid-cursor error mapping.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Confirmed failure: Meaningful compile failure because `SearchApiClient` and `SearchSort` did not exist.
- Implement: Added `SearchSort`, `SearchPostPage`, and `SearchApiClient.searchHashtagPosts()` using `unwrapApi`, `Uri.encodeComponent`, opaque `cursor`, supported `sort`, `limit`, and `Post` item decoding.
- Run command: `dart run build_runner build --delete-conflicting-outputs`; `flutter test test/search/data/search_api_client_test.dart`
- Refactor: None.
- Notes: Safe path encoding coverage uses `fiber art` and verifies mapped `ApiBadRequest` from AppView validation rather than client-side parsing/recovery.

### Steps 2-24
- Each step will be executed with the same red/green/refactor fields updated after the focused loop.

### Step 2: UT-001
- Write failing test: Added `app/test/search/models/search_sort_test.dart` for supported sort values and profile query sort exclusion.
- Run command: `flutter test test/search/models/search_sort_test.dart`
- Confirmed failure: Meaningful compile failure because `search_queries.dart` and `ProfileSearchQuery` did not exist.
- Implement: Added UI-agnostic query models and initial project filter model; `ProfileSearchQuery` exposes only `q` and no sort field.
- Run command: `flutter test test/search/models/search_sort_test.dart`
- Refactor: Formatted new model/test files.
- Notes: `SearchSort` remains limited to chronological/popular wire values.

### Step 3: UT-002
- Write failing test: Added `app/test/search/models/search_post_page_test.dart` for `SearchPostPageMapper` decoding `items` as existing `Post` objects with hashtag/cursor preservation.
- Run command: `flutter test test/search/models/search_post_page_test.dart`
- Confirmed failure: The focused unit test started green because `SearchPostPage` was already created in the IT-002 vertical slice to decode hashtag API responses.
- Implement: No additional code required.
- Run command: `flutter test test/search/models/search_post_page_test.dart`
- Refactor: None.
- Notes: This test records the narrower model-level coverage for the behavior first driven by IT-002.

### Step 4: UT-007
- Write failing test: Added `app/test/search/models/search_pagination_state_test.dart` asserting exact opaque cursor preservation and `hasMore` derivation.
- Run command: `flutter test test/search/models/search_pagination_state_test.dart`
- Confirmed failure: Meaningful compile failure because `SearchPostResultsState` did not exist.
- Implement: Added `SearchPostResultsState` with `items`, `cursor`, optional `hashtag`, `hasMore`, and `copyWith`.
- Run command: `flutter test test/search/models/search_pagination_state_test.dart`
- Refactor: Formatted new state/test files.
- Notes: Cursor values are stored as unparsed strings.

### Step 5: UT-009
- Write failing test: Added `app/test/search/models/search_mapper_registration_test.dart` for mapper initialization and representative search page decode.
- Run command: `flutter test test/search/models/search_mapper_registration_test.dart`
- Confirmed failure: The focused test started green because `SearchPostPageMapper` had already been generated for IT-002; startup registration was still missing.
- Implement: Added search mapper imports and `SearchPostPageMapper.ensureInitialized()` to `initializeMappers()`; later added profile/top hashtag mapper registration as those models were introduced.
- Run command: `flutter test test/search/models/search_mapper_registration_test.dart`
- Refactor: Formatted `bootstrap.dart`.
- Notes: Generated mapper files are produced by build_runner.

### Step 6: IT-003
- Write failing test: Expanded `search_api_client_test.dart` for profile search `q`/`limit`/`cursor`, follow state, and no sort parameter.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Confirmed failure: Covered by missing profile search models/client method before implementation.
- Implement: Added `ProfileSearchResult`, `ProfileSearchPage`, mapper registration, and `SearchApiClient.searchProfiles()`.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Refactor: Formatted search files.
- Notes: `ProfileSearchQuery` remains sort-free.

### Step 7: IT-004
- Write failing test: Added post search API request/decoding coverage.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Confirmed failure: Missing `searchPosts()` before endpoint expansion.
- Implement: Added `SearchApiClient.searchPosts()` with supported sort, limit, and opaque cursor parameters.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Refactor: None beyond formatting.
- Notes: Results reuse `SearchPostPage` and `Post`.

### Step 8: IT-005
- Write failing test: Added project search API coverage for repeated filters and browse-all request.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Confirmed failure: Missing project search API/filter serialization before endpoint expansion.
- Implement: Added `ProjectSearchFilters` query/payload helpers and `SearchApiClient.searchProjects()` with repeated query list format.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Refactor: None beyond formatting.
- Notes: Unsupported project filter keys are not modeled.

### Step 9: IT-006
- Write failing test: Added top-hashtag request/decoding coverage.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Confirmed failure: Missing top hashtag models/API before implementation.
- Implement: Added `TopHashtagsResponse`, group/item models, mapper registration, and `SearchApiClient.topHashtags()`.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Refactor: None beyond formatting.
- Notes: Repeated `craftTypes` are sent as repeated query values.

### Step 10: IT-007
- Write failing test: Added recent-search list decoding coverage with typed payloads.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Confirmed failure: Missing recent-search models/API before implementation.
- Implement: Added `RecentSearchType`, typed payloads, `RecentSearchItem`, `RecentSearchPage`, and `SearchApiClient.listRecentSearches()`.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Refactor: None beyond formatting.
- Notes: Recent IDs are stored as opaque strings.

### Step 11: IT-008
- Write failing test: Added explicit save-recent request body and response decoding coverage.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Confirmed failure: Missing save request modeling/client method before implementation.
- Implement: Added `SaveRecentSearchRequest`, payload `toMap()` methods, and `SearchApiClient.saveRecentSearch()`.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Refactor: None beyond formatting.
- Notes: Profile payload model has no sort; project filters use supported keys only.

### Step 12: IT-009
- Write failing test: Added delete-recent 204 success coverage.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Confirmed failure: Missing delete-recent client method before implementation.
- Implement: Added `SearchApiClient.deleteRecentSearch()` with encoded opaque ID path segment and no local/PDS persistence.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Refactor: None beyond formatting.
- Notes: Treats 204 No Content as success.

### Step 13: IT-010
- Write failing test: Added `app/test/search/data/search_api_client_error_test.dart` for unauthorized and server error mapping.
- Run command: `flutter test test/search/data/search_api_client_error_test.dart`
- Confirmed failure: The focused tests started green because endpoint expansion already wrapped every method in `unwrapApi`.
- Implement: No additional code required.
- Run command: `flutter test test/search/data/search_api_client_error_test.dart`
- Refactor: Formatted the new test file.
- Notes: Representative cursor error mapping is also covered by IT-002; no client-side cursor recovery is implemented.

### Step 14: IT-011
- Write failing test: Added `app/test/search/data/search_repository_test.dart` for `ApiSearchRepository` argument/result pass-through via mocked HTTP.
- Run command: `flutter test test/search/data/search_repository_test.dart`
- Confirmed failure: The focused test started green because repository files had already been added during endpoint expansion.
- Implement: No additional code required after `SearchRepository`/`ApiSearchRepository` creation.
- Run command: `flutter test test/search/data/search_repository_test.dart`
- Refactor: Formatted the test file.
- Notes: Repository boundary is fakeable and hides Dio details.

### Step 15: IT-001
- Write failing test: Added `app/test/search/providers/search_repository_provider_test.dart` plus `FakeSearchRepository` for production provider construction and overrideability.
- Run command: `flutter test test/search/providers/search_repository_provider_test.dart`
- Confirmed failure: The focused test started green because providers had already been generated during implementation expansion.
- Implement: No additional code required after adding keep-alive `searchApiClientProvider` and `searchRepositoryProvider`.
- Run command: `flutter test test/search/providers/search_repository_provider_test.dart`
- Refactor: Formatted provider test/fake files.
- Notes: Production API client reads shared `dioProvider`.

### Step 16: IT-012
- Write failing test: Added provider initial-load tests for hashtag, profile, post, and project providers using `FakeSearchRepository`.
- Run command: `flutter test test/search/providers/hashtag_search_provider_test.dart test/search/providers/profile_search_provider_test.dart test/search/providers/post_search_provider_test.dart test/search/providers/project_search_provider_test.dart`
- Confirmed failure: Provider implementation already existed from the expansion step; tests verified behavior green.
- Implement: Result providers fetch through `SearchRepository`, expose UI-agnostic state with items/cursor/hasMore, and use `searchResultsPageLimit`.
- Run command: focused provider tests listed above.
- Refactor: Added remaining profile/post/project provider tests to avoid hashtag-only coverage.
- Notes: Providers contain no tab/layout/widget concepts.

### Step 17: IT-013
- Write failing test: Added `loadMore()` coverage in `hashtag_search_provider_test.dart` for opaque cursor pass-through, append, duplicate suppression, no-more state, and concurrent no-op.
- Run command: `flutter test test/search/providers/hashtag_search_provider_test.dart`
- Confirmed failure: Provider implementation already existed; tests verified behavior green.
- Implement: `loadMore()` guards missing state/no cursor/loading, uses previous `state.value`, appends unique posts, and preserves opaque cursor strings.
- Run command: `flutter test test/search/providers/hashtag_search_provider_test.dart`
- Refactor: None beyond formatting.
- Notes: Representative pagination behavior is shared by post/project; profile uses profile-specific de-duplication helper.

### Step 18: UT-008
- Write failing test: Added `app/test/search/providers/search_pagination_merge_test.dart` for duplicate post suppression.
- Run command: `flutter test test/search/providers/search_pagination_merge_test.dart`
- Confirmed failure: Helper implementation already existed; test verified behavior green.
- Implement: `appendUniquePosts()` and `appendUniqueProfiles()` helpers.
- Run command: `flutter test test/search/providers/search_pagination_merge_test.dart`
- Refactor: None.
- Notes: Posts de-dupe by `Post.uri`; profiles de-dupe by DID/handle.

### Step 19: IT-014
- Write failing test: Added top-hashtags and recent-search provider lifecycle tests.
- Run command: `flutter test test/search/providers/top_hashtags_provider_test.dart test/search/providers/recent_searches_provider_test.dart`
- Confirmed failure: Providers already existed from expansion; tests verified behavior green.
- Implement: `topHashtagsProvider` fetches grouped data; `recentSearchesProvider` lists, explicitly saves/deletes, and refreshes state after mutation; result providers do not auto-save recents.
- Run command: focused top/recent provider tests listed above.
- Refactor: None beyond formatting.
- Notes: No local persistence or PDS writes added.

### Step 20: UT-004
- Write failing test: Added `app/test/search/models/top_hashtags_test.dart` for group/count decoding.
- Run command: `flutter test test/search/models/top_hashtags_test.dart`
- Confirmed failure: Models already existed; test verified behavior green.
- Implement: `TopHashtagsResponse`, `TopHashtagGroup`, and `TopHashtagItem` via dart_mappable.
- Run command: `flutter test test/search/models/top_hashtags_test.dart`
- Refactor: None.
- Notes: Empty groups remain empty arrays.

### Step 21: UT-005
- Write failing test: Added `app/test/search/models/recent_search_test.dart` for supported save payload serialization and item typed payload deserialization.
- Run command: `flutter test test/search/models/recent_search_test.dart`
- Confirmed failure: Models already existed; test verified behavior green.
- Implement: Typed recent payload classes for hashtag/profile/post/project and save/list models.
- Run command: `flutter test test/search/models/recent_search_test.dart`
- Refactor: None.
- Notes: Profile payload excludes sort.

### Step 22: UT-006
- Write failing test: Added `app/test/search/models/project_search_filters_test.dart` for repeated supported filter families.
- Run command: `flutter test test/search/models/project_search_filters_test.dart`
- Confirmed failure: Filter model already existed; test verified behavior green.
- Implement: `ProjectSearchFilters` with supported family lists and deterministic query/payload maps.
- Run command: `flutter test test/search/models/project_search_filters_test.dart`
- Refactor: None.
- Notes: Values remain repeated lists, not comma-collapsed strings.

## Verification Log
- Codegen: `dart run build_runner build --delete-conflicting-outputs` — passed, wrote 0 outputs on final run.
- Focused search/regression command: `flutter test test/search test/shared/rich_text/facet_suggestion_repository_test.dart` — passed, 38 tests.
- Analyzer: `flutter analyze` — passed, no issues after style fixes.
- Broader command: `flutter test` — failed in two existing non-search feed composer tests:
  - `test/feed/pages/feed_page_composer_entry_test.dart: IT-001 feed New post opens chooser and project branch`
  - `test/feed/widgets/post_type_chooser_test.dart: AT-001 compact chooser opens project composer`
  - Both failures expected `Craft type` in feed/project composer UI. This slice did not touch feed widgets/pages, project composer UI, routes, AppView, lexicons, dependencies, or search pages; focused search and required regressions passed.
- Regression checks:
  - `SearchPage` placeholder and tag context tests passed in `flutter test test/search ...`.
  - Facet autocomplete regression passed via `test/shared/rich_text/facet_suggestion_repository_test.dart`.
  - `git diff -- appview lexicon app/pubspec.yaml app/pubspec.lock app/lib/search/pages app/lib/router` produced no scoped diff.
- Manual checks:
  - MAN-001: Generated search mapper/provider files are present; `initializeMappers()` registers search mappers.
  - MAN-002: No rendered search UI, routes, app shell, AppView, lexicon, dependency, or `/v1/facets/*` behavior changed.
  - MAN-003: Search code uses shared AppView Dio provider; no PDS client, token storage, local persistent recent-search storage, or logging of recent payloads was added.
  - MAN-004: Providers are query-in/state-out and expose data/cursor/hasMore/mutation methods without visual tab/layout/widget concepts.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped
