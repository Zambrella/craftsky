# Coding Plan: Flutter Search Data Layer

## 1. Stage Status

- Status: Ready for TDD implementation
- Source requirements: `01-requirements.md`
- Source acceptance tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — approved with notes, no blocking gaps
- Scope: Flutter-only non-UI search data layer: services/API client, repositories, models, Riverpod providers/notifiers, generated Dart files, and tests.

Do **not** implement rendered search UI, route behavior, AppView handlers, lexicons, migrations, dependencies, local persistence, analytics, or PDS calls in this slice.

## 2. Existing Architecture To Match

Observed implementation patterns that the TDD builder should reuse:

- API clients live under feature `data/`, accept a shared authenticated `Dio`, call `/v1/*`, and wrap calls in `unwrapApi()` so callers receive existing `ApiException` subtypes.
- Production repositories adapt API clients behind abstract interfaces, e.g. `PostRepository` + `ApiPostRepository`.
- Riverpod dependency providers use `@Riverpod(keepAlive: true)` and read the shared `dioProvider`.
- Cursor-accumulating providers follow `Timeline`, `UserPosts`, and `UserProjects`: initial `build()` fetches page 1; `loadMore()` uses `state.value`/`state.requireValue`, guards concurrent loads, passes the opaque cursor, and appends results.
- Models use `dart_mappable`; new mappable models need generated `*.mapper.dart` and `initializeMappers()` registration in `app/lib/bootstrap.dart` when decoded through mappers.
- API-client tests use `Dio(BaseOptions(baseUrl: ...))`, `ErrorMappingInterceptor`, `http_mock_adapter`, and `setUpAll(initializeMappers)`.
- Provider tests use `ProviderContainer.test(overrides: [...])` and fake repositories.

## 3. AppView Contract Summary

Consume existing AppView routes exactly as implemented:

| Route | Method | Request shape | Response model |
|---|---:|---|---|
| `/v1/search/hashtags/{tag}/posts` | GET | path `tag`; query `sort`, `limit`, `cursor` | `{hashtag?, items: Post[], cursor?}` |
| `/v1/search/profiles` | GET | query `q`, `limit`, `cursor`; **no `sort`** | `{items: ProfileSearchSummary[], cursor?}` |
| `/v1/search/posts` | GET | query `q`, `sort`, `limit`, `cursor` | `{items: Post[], cursor?}` |
| `/v1/search/projects` | GET | query `q?`, `sort`, `limit`, `cursor`, repeated project filters | `{items: Post[], cursor?}` |
| `/v1/search/hashtags/top` | GET | repeated `craftTypes`, `limit` | `{groups: [{craftType, items: [{tag, count}]}]}` |
| `/v1/search/recent` | GET | none | `{items: RecentSearch[]}` |
| `/v1/search/recent` | POST | `{type, displayLabel, payload}` | `RecentSearch` |
| `/v1/search/recent/{id}` | DELETE | path `id` | `204 No Content` |

Contract guardrails:

- Use the shared authenticated `dioProvider`; never create an unauthenticated Dio for production search providers.
- Keep cursors and recent IDs as opaque strings.
- Use camelCase JSON keys.
- Encode hashtag path values safely as a single path segment; do not allow `/` or spaces in caller input to change the route shape.
- Preserve repeated query values for `craftTypes` and project filter families (`craftType`, `projectType`, `patternDifficulty`, `color`, `material`, `designTag`, `projectTag`). Verify Dio list serialization in tests and set the Dio request/list format explicitly if needed.

## 4. Implementation Files

### 4.1 Create Search Models

Create under `app/lib/search/models/`:

- `search_sort.dart`
  - `enum SearchSort { chronological, popular }`
  - expose wire string as `name` or `wireValue` only for these two values.
  - Covers `FR-001`, `FR-007`, `AC-014`, `UT-001`.

- `search_post_page.dart`
  - `@MappableClass(ignoreNull: true)`
  - `List<Post> items`, `String? cursor`, `String? hashtag`
  - Reuse existing `Post` for all hashtag/post/project search result items.
  - Covers `FR-001`, `FR-002`, `AC-002`, `AC-011`, `UT-002`.

- `profile_search_page.dart`
  - `ProfileSearchResult` matching the flat AppView wire keys plus `viewerIsFollowing`.
  - Reuse profile-summary semantics by exposing `ProfileAccountSummary get summary` or an equivalent adapter, while decoding the flat response shape.
  - Use existing DID/handle mapper conventions if fields are typed as `Did`/`Handle`.
  - `ProfileSearchPage { List<ProfileSearchResult> items, String? cursor }`.
  - Covers `FR-001`, `FR-003`, `AC-003`, `UT-003`.

- `top_hashtags.dart`
  - `TopHashtagsResponse { List<TopHashtagGroup> groups }`
  - `TopHashtagGroup { String craftType, List<TopHashtagItem> items }`
  - `TopHashtagItem { String tag, int count }`
  - Covers `FR-001`, `FR-014`, `AC-005`, `UT-004`.

- `project_search_filters.dart`
  - `ProjectSearchFilters` with lists for exactly the supported families:
    `craftType`, `projectType`, `patternDifficulty`, `color`, `material`, `designTag`, `projectTag`.
  - Provide helpers:
    - `Map<String, List<String>> toQueryParameters()` for repeated GET query values.
    - `Map<String, List<String>> toPayloadMap()` for recent-search payloads, omitting empty families and producing deterministic key ordering for tests.
  - Do not include unsupported keys or collapse values into comma-separated strings.
  - Covers `FR-008`, `AC-015`, `UT-006`.

- `search_queries.dart`
  - Immutable provider/API query parameter models with value equality:
    - `HashtagSearchQuery { String tag, SearchSort sort }`
    - `ProfileSearchQuery { String q }` — deliberately no `sort` field.
    - `PostSearchQuery { String q, SearchSort sort }`
    - `ProjectSearchQuery { String? q, SearchSort sort, ProjectSearchFilters filters }`
    - `TopHashtagsQuery { List<String> craftTypes, int? limit }`
  - These models are UI-agnostic provider keys, not visual tab state.
  - Covers `FR-007`, `FR-008`, `NFR-004`, `AC-018`.

- `recent_search.dart`
  - `RecentSearchType` limited to `hashtag`, `profile`, `post`, `project`.
  - `RecentSearchPayload` sealed/base type with concrete payloads:
    - `HashtagRecentSearchPayload { String tag, SearchSort sort }`
    - `ProfileRecentSearchPayload { String q }` — no sort.
    - `PostRecentSearchPayload { String q, SearchSort sort }`
    - `ProjectRecentSearchPayload { String? q, SearchSort sort, ProjectSearchFilters filters }`
  - `SaveRecentSearchRequest { RecentSearchType type, String displayLabel, RecentSearchPayload payload }` with `toMap()` producing `{type, displayLabel, payload}`.
  - `RecentSearchItem { String id, RecentSearchType type, String displayLabel, RecentSearchPayload payload, DateTime updatedAt }`.
  - `RecentSearchPage { List<RecentSearchItem> items }`.
  - If `dart_mappable` cannot cleanly decode payload by sibling `type`, use a small custom factory around mapper-generated payload classes; still keep the public representation typed.
  - Covers `FR-001`, `FR-009`, `FR-015`, `AC-006`, `AC-007`, `AC-014`, `UT-005`.

- `search_result_state.dart`
  - UI-agnostic provider state classes:
    - `SearchPostResultsState { List<Post> items, String? cursor, String? hashtag }`
    - `ProfileSearchResultsState { List<ProfileSearchResult> items, String? cursor }`
  - Both expose `bool get hasMore => cursor != null` and concise `toString()` for logging.
  - Keep loading/error state on the surrounding `AsyncValue`, matching existing providers.
  - Covers `FR-012`, `FR-013`, `AC-018`, `AC-019`, `UT-007`.

Generated model artifacts expected:

- `app/lib/search/models/*.mapper.dart` for each mappable model file that declares a part.
- Register all new mapper classes in `initializeMappers()` in `app/lib/bootstrap.dart`.

### 4.2 Create Search API Client

Create `app/lib/search/data/search_api_client.dart`:

```dart
class SearchApiClient {
  const SearchApiClient(this._dio);
  final Dio _dio;

  Future<SearchPostPage> searchHashtagPosts(
    String tag, {
    SearchSort? sort,
    int? limit,
    String? cursor,
  });

  Future<ProfileSearchPage> searchProfiles({
    required String q,
    int? limit,
    String? cursor,
  });

  Future<SearchPostPage> searchPosts({
    required String q,
    SearchSort? sort,
    int? limit,
    String? cursor,
  });

  Future<SearchPostPage> searchProjects({
    String? q,
    SearchSort? sort,
    ProjectSearchFilters? filters,
    int? limit,
    String? cursor,
  });

  Future<TopHashtagsResponse> topHashtags({
    List<String>? craftTypes,
    int? limit,
  });

  Future<RecentSearchPage> listRecentSearches();
  Future<RecentSearchItem> saveRecentSearch(SaveRecentSearchRequest request);
  Future<void> deleteRecentSearch(String id);
}
```

Implementation rules:

- Wrap every method in `unwrapApi(() async { ... })`.
- Build query maps with null-aware entries; serialize `limit` consistently with existing clients (`limit?.toString()` is acceptable).
- For hashtag paths, use a private safe segment encoder such as `Uri.encodeComponent(tag)` before interpolating the path. Add tests for values from `TD-008`; if exact `http_mock_adapter` matching is impractical, document the limitation in the test comments as allowed by `GAP-002`.
- For repeated query parameters, prefer a helper that merges `Map<String, List<String>>` into Dio query parameters and sets/request-tests the Dio list format that produces repeated keys.
- Treat `DELETE 204` as success without decoding a body.

Covers `FR-004` through `FR-009`, `NFR-002`, `NFR-003`, `RULE-004`, `AC-002` through `AC-015`, `IT-002` through `IT-010`.

### 4.3 Create Repository Boundary

Create under `app/lib/search/data/`:

- `search_repository.dart`
  - Abstract interface mirroring the API-client methods at repository level.
  - Method names should be future-UI friendly and not expose Dio details.

- `api_search_repository.dart`
  - `class ApiSearchRepository implements SearchRepository`
  - Delegate each method to `SearchApiClient` without changing arguments, cursors, IDs, or payloads.

Covers `FR-010`, `AC-017`, `IT-011`.

### 4.4 Create Dependency Providers

Create under `app/lib/search/providers/`:

- `search_api_client_provider.dart`
  ```dart
  @Riverpod(keepAlive: true)
  SearchApiClient searchApiClient(Ref ref) =>
      SearchApiClient(ref.watch(dioProvider));
  ```

- `search_repository_provider.dart`
  ```dart
  @Riverpod(keepAlive: true)
  SearchRepository searchRepository(Ref ref) =>
      ApiSearchRepository(ref.watch(searchApiClientProvider));
  ```

Generated artifacts expected:

- `search_api_client_provider.g.dart`
- `search_repository_provider.g.dart`

Covers `FR-011`, `AC-009`, `AC-017`, `IT-001`.

### 4.5 Create Search Result Providers

Create under `app/lib/search/providers/`:

- `hashtag_search_provider.dart`
- `profile_search_provider.dart`
- `post_search_provider.dart`
- `project_search_provider.dart`

Recommended shape:

```dart
const searchResultsPageLimit = 25;

@riverpod
class HashtagSearch extends _$HashtagSearch {
  @override
  Future<SearchPostResultsState> build(HashtagSearchQuery query) async { ... }
  Future<void> loadMore() async { ... }
}
```

Provider rules:

- Each initial `build()` reads `searchRepositoryProvider` and calls the matching method with `limit: searchResultsPageLimit`.
- `loadMore()` no-ops when there is no current value, `hasMore == false`, or `state.isLoading`.
- During load-more, set `state = const AsyncLoading<...>()` and rely on Riverpod 3 previous-data preservation, as existing providers do.
- On success, append only new items and update cursor.
- De-duplicate posts by `Post.uri` and profiles by DID first, falling back to handle if needed.
- Do **not** call `saveRecentSearch()` from any result provider.

Helper sketch:

```dart
List<Post> appendUniquePosts(List<Post> current, List<Post> next) { ... }
List<ProfileSearchResult> appendUniqueProfiles(
  List<ProfileSearchResult> current,
  List<ProfileSearchResult> next,
) { ... }
```

Covers `FR-012`, `FR-013`, `RULE-001`, `AC-018` through `AC-021`, `IT-012`, `IT-013`, `UT-008`.

### 4.6 Create Top Hashtags And Recent Search Providers

Create under `app/lib/search/providers/`:

- `top_hashtags_provider.dart`
  - A FutureProvider-family or AsyncNotifier keyed by `TopHashtagsQuery` is sufficient.
  - Fetch through `SearchRepository.topHashtags(...)`.
  - Keep returned data as grouped hashtag metadata, not UI chips/tabs.

- `recent_searches_provider.dart`
  - AsyncNotifier with `build()` fetching `SearchRepository.listRecentSearches()`.
  - Mutation methods:
    - `Future<RecentSearchItem> save(SaveRecentSearchRequest request)`
    - `Future<void> delete(String id)`
  - After successful save/delete, either refresh from AppView or update local state deterministically. Refresh is safer if AppView de-duplicates or reorders saved recents.
  - Do not write local persistent storage and do not call PDS APIs.

Covers `FR-014`, `FR-015`, `RULE-001`, `RULE-002`, `AC-005` through `AC-008`, `AC-021`, `IT-014`.

## 5. Test Implementation Plan

### 5.1 Required Test Support

Create `app/test/search/fakes/fake_search_repository.dart` mirroring `FakePostRepository`:

- One optional callback per repository method.
- Unstubbed methods throw `UnimplementedError` so missing dependencies fail loudly.

Create shared fixtures/helpers only under `app/test/search/` if they reduce duplication:

- Post-shaped map matching existing `PostMapper` requirements.
- Profile search map with `viewerIsFollowing`.
- Top hashtags groups.
- Recent payload maps for all four types.
- Error envelopes for `invalid_cursor`, validation, unauthorized, and server errors.

### 5.2 Explicit TDD Sequence

To resolve `DR-001`, use this single sequence:

1. Start with failing `IT-002` in `app/test/search/data/search_api_client_test.dart` for hashtag search path/query/decoding/error behavior, including one safe path-encoding case from `TD-008` (`DR-003`).
2. Add the minimal model tests needed to make that pass: `UT-001`, `UT-002`, `UT-007`, and mapper registration coverage from `UT-009`.
3. Expand API-client tests endpoint by endpoint: `IT-003` through `IT-010`.
4. Add repository delegation and provider construction tests: `IT-011`, `IT-001`.
5. Add provider initial-load and pagination tests: `IT-012`, `IT-013`, plus duplicate helper `UT-008`.
6. Add recent/top-hashtag lifecycle tests: `IT-014`, `UT-004`, `UT-005`, `UT-006`.
7. Run regressions and manual scope checks: `REG-001` through `REG-005`, `MAN-001` through `MAN-004`.

### 5.3 Expected Test Files

Create or update only Flutter search/test-support files:

- `app/test/search/models/search_sort_test.dart`
- `app/test/search/models/search_post_page_test.dart`
- `app/test/search/models/profile_search_page_test.dart`
- `app/test/search/models/top_hashtags_test.dart`
- `app/test/search/models/recent_search_test.dart`
- `app/test/search/models/project_search_filters_test.dart`
- `app/test/search/models/search_pagination_state_test.dart`
- `app/test/search/models/search_mapper_registration_test.dart`
- `app/test/search/data/search_api_client_test.dart`
- `app/test/search/data/search_api_client_error_test.dart`
- `app/test/search/data/search_repository_test.dart`
- `app/test/search/providers/search_repository_provider_test.dart`
- `app/test/search/providers/hashtag_search_provider_test.dart`
- `app/test/search/providers/profile_search_provider_test.dart`
- `app/test/search/providers/post_search_provider_test.dart`
- `app/test/search/providers/project_search_provider_test.dart`
- `app/test/search/providers/top_hashtags_provider_test.dart`
- `app/test/search/providers/recent_searches_provider_test.dart`
- `app/test/search/providers/search_pagination_merge_test.dart`
- Keep `app/test/search/search_page_test.dart` behavior unchanged except for adding assertions that protect the placeholder if needed.

Verification commands from `app/`:

```sh
dart run build_runner build --delete-conflicting-outputs
flutter test test/search
flutter test
flutter analyze
```

## 6. Requirement / Acceptance / Test Mapping

| Work item | Requirement IDs | Acceptance criteria | Test IDs |
|---|---|---|---|
| Search models and mapper registration | `FR-001`, `FR-002`, `FR-003`, `FR-016` | `AC-002`, `AC-003`, `AC-010`, `AC-011`, `AC-022` | `UT-001`, `UT-002`, `UT-003`, `UT-007`, `UT-009` |
| Sort, project filter, and query parameter modeling | `FR-007`, `FR-008`, `RULE-004` | `AC-004`, `AC-014`, `AC-015` | `UT-001`, `UT-006`, `IT-005`, `IT-008` |
| Recent-search typed payloads | `BR-003`, `FR-009`, `FR-015`, `RULE-002` | `AC-006`, `AC-007`, `AC-008`, `AC-014` | `UT-005`, `IT-007`, `IT-008`, `IT-009` |
| `SearchApiClient` endpoint coverage | `BR-002`, `FR-004`, `FR-005`, `FR-006`, `NFR-002`, `NFR-003` | `AC-002` through `AC-013` | `IT-002` through `IT-010` |
| Repository abstraction and production provider | `FR-010`, `FR-011` | `AC-017` | `IT-001`, `IT-011` |
| Result providers and pagination | `FR-012`, `FR-013`, `NFR-004`, `RULE-001` | `AC-018`, `AC-019`, `AC-020`, `AC-021` | `AT-007`, `IT-012`, `IT-013`, `UT-008` |
| Top hashtags and recent mutations | `FR-014`, `FR-015`, `RULE-001`, `RULE-002` | `AC-005`, `AC-006`, `AC-007`, `AC-008`, `AC-021` | `AT-005`, `AT-006`, `IT-014` |
| Non-UI scope and autocomplete separation | `BR-001`, `NFR-001`, `RULE-003` | `AC-001`, `AC-016` | `REG-001`, `REG-002`, `REG-003`, `REG-004`, `MAN-002` |

## 7. Guardrails For The TDD Builder

- Do not edit `appview/`, `lexicon/`, migrations, SQL, Go code, `pubspec.yaml`, or `pubspec.lock`.
- Do not implement result cards, search fields, tabs, filter UI, empty states, or route/deep-link behavior.
- Avoid touching `app/lib/search/pages/search_page.dart` or `app/lib/router/*`; if a compile-time import change is unavoidable, keep it behavior-neutral and document why in the implementation review. This resolves `DR-002`.
- Keep `/v1/facets/*` autocomplete repositories separate from committed `/v1/search/*` result repositories/providers.
- Do not persist recent searches locally and do not write recent-search records to PDS.
- Do not log full recent-search payloads.
- Do not parse or synthesize AppView cursors/recent IDs.
- Profile search must not expose or send `sort`.
- Result fetching must not auto-save recent searches.
- Generated files should be committed only when produced by `build_runner` for the new models/providers.

## 8. Risks And Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Dio list serialization might not match AppView repeated-query expectations. | Project filters/top hashtags could fail despite valid models. | Add request-shape tests for repeated values; set Dio list format explicitly if needed. |
| Recent payload decoding by sibling `type` is more complex than simple mapper decoding. | Runtime decode failures for recents. | Use a small custom factory if needed; cover all four types in `UT-005` and `IT-007`/`IT-008`. |
| Provider contracts may overfit future UI. | Later UI slice churn. | Keep query-in/state-out shape; no tab/layout/widget concepts. |
| Generated mapper/provider files can be stale. | Build/test failures. | Run build_runner and mapper registration test. |
| Hashtag path encoding can be implementation-sensitive. | Route injection or brittle tests. | Use safe path-segment encoding and include focused `TD-008` coverage from `DR-003`. |

## 9. Completion Criteria

Implementation is ready for review when:

- All planned models, API client, repository, providers, generated files, and search tests exist.
- `initializeMappers()` includes new search mappers.
- Focused `flutter test test/search` passes.
- Broader `flutter test` and `flutter analyze` have been run or any failures are documented as unrelated.
- Diff review confirms no AppView, lexicon, route behavior, UI behavior, dependency, or local persistence changes were introduced.
