# Coding Plan: Search Refinements Before UI Slice

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Codebase context inspected:
  - AppView search/facet/routes/store/request/response code under `appview/internal/api/search*.go`, `facet*.go`, and `appview/internal/routes/routes.go`.
  - AppView migrations `000016_project_posts.*`, `000019_search_foundation.*`, and `000008_craftsky_profiles.*`.
  - Flutter search data/models/providers/tests under `app/lib/search/**` and `app/test/search/**`.
  - Flutter rich-text facet compatibility code under `app/lib/shared/rich_text/**` and `app/test/shared/rich_text/**`.
  - Flutter project data/providers/options under `app/lib/projects/**` and existing project tests under `app/test/projects/**`.
  - API architecture specs for `/v1/`, auth/device headers, error envelopes, camelCase JSON, and opaque cursors.

Document-review constraints to carry into implementation:

- `DR-001`: Empty or whitespace-only `GET /v1/search/suggestions?q=` is a standard validation error; Flutter providers should avoid noisy blank suggestion calls.
- `DR-002`: Rich project browse filters sent to `/v1/search/projects` are rejected with standard validation errors; they are not silently ignored.
- `DR-004`: Submitted post/project text search must use deterministic relevance-score ordering before tests are written.

## 2. Implementation Strategy

Land the slice as a non-UI contract refinement across AppView and Flutter data layers:

1. First tighten AppView request parsing and route contracts so failing tests anchor the new API shape: unified suggestions, committed hashtag query results, text-only submitted project search, project browse filters under `/v1/projects`, craft-token canonicalization, exact hashtag sort validation, and refined recent payloads.
2. Refactor AppView store logic by separating four modes that currently overlap:
   - bounded typeahead suggestions,
   - submitted text-search tabs,
   - exact hashtag post feeds,
   - project browse/filter feeds.
3. Reuse ranking/counting helpers for search suggestions and existing `/v1/facets/*` endpoints instead of coupling Flutter search UI data to rich-text facet repositories.
4. Add Flutter models, clients, repositories, and Riverpod providers for the new contracts without changing rendered pages, visual navigation, widgets, or route behavior.
5. Move active project browse/filter data contracts to the `projects` feature boundary while keeping submitted text project search under `search`.

This fits the existing architecture: AppView remains the read API, Flutter continues to use authenticated shared `Dio`, AppView cursors remain opaque, recents remain private AppView Postgres state, and Riverpod provider seams remain fakeable in tests.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| AppView routes | `routes.go` wires authenticated `/v1/search/*`, `/v1/facets/*`, `/v1/projects` handlers through auth + device middleware. | Add `GET /v1/search/suggestions` and `GET /v1/search/hashtags`; keep exact `GET /v1/search/hashtags/{tag}/posts`; update `/v1/search/projects` and `/v1/projects` contracts. | FR-001, FR-005, FR-008, FR-009, FR-010, NFR-003 | IT-001, IT-004, IT-006, IT-008, IT-009, IT-011, IT-017 |
| AppView request parsing | `search_request.go` parses result search, top hashtags, project filters, recents, and exact hashtag path values. | Add strict suggestion and hashtag-query parsers; make submitted post/project search relevance-first and text-only; canonicalize craft tokens; refine recent payload validation. | FR-001, FR-005, FR-006, FR-010, FR-014, FR-016 | UT-001, UT-004, UT-005, UT-007, UT-008, UT-013 |
| AppView responses | `search_response.go` has post/profile pages, top hashtags, recents. | Add suggestion sections, hashtag result page/items, `crafts` on profile summaries, canonical craft tokens in top hashtag groups. | FR-001, FR-002, FR-005, FR-015 | AT-002, AT-005, AT-008, UT-009, IT-001, IT-004 |
| AppView store/ranking | `SearchStore` mixes text search, exact hashtag feeds, project browse filters, and popularity/chronological ordering. `FacetStore` duplicates suggestion ranking. | Split store helpers by mode; add shared profile/hashtag suggestion helpers; add relevance score helper and relevance cursor; keep exact hashtag chronology/popularity; keep `/v1/projects` browse filters. | FR-002, FR-003, FR-006, FR-007, FR-008, FR-009 | UT-002, UT-003, UT-006, UT-011, IT-002, IT-003, IT-005, IT-006, IT-008, IT-017 |
| AppView persistence/migrations | `craftsky_recent_searches.search_type` allows `hashtag`, `profile`, `post`, `project`. Search indexes exist from foundation. | Add migration for `query` recent type while retaining legacy types deliberately; add supporting indexes only if query review/tests show they are needed. | FR-016, NFR-004, RULE-001 | UT-007, IT-010, IT-016, MAN-003 |
| Flutter search data layer | `SearchApiClient` + `SearchRepository` cover posts/projects/profiles/exact hashtags/top hashtags/recents. | Add unified suggestions, committed hashtag-query results, blank-search data, refined recents, relevance-default submitted post/project calls, and remove rich project filters from search-project calls. | FR-001, FR-005, FR-006, FR-012, FR-016, NFR-002 | UT-009, UT-010, UT-012, IT-012, IT-013 |
| Flutter project data layer | `ProjectApiClient` calls `/v1/projects` with craft type, sort, limit, cursor; `ProjectFeed` has simple craft/sort parameters. | Add project browse query/filter models under `projects`, pass filter families to `/v1/projects`, and keep project browse providers out of search. | FR-009, FR-010, FR-011 | AT-006, UT-008, IT-014, REG-007 |
| Flutter rich-text facets | Separate rich-text repositories call `/v1/facets/mentions`, `/resolve`, and `/hashtags`; tolerant error behavior. | Preserve endpoint paths/response compatibility while AppView shares backend ranking/counting logic. No search UI depends on rich-text repositories. | FR-004, RULE-003 | AT-003, IT-015, REG-001 |
| Rendered UI/routes | `SearchPage` and `ProjectsPage` are stubs; router already has Search and Projects branches. | No rendered UI, new tabs, management page, route, card, or visual navigation changes. | BR-001, NFR-001 | AT-001, REG-005, MAN-001 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/routes/routes.go` | Change | Register `GET /v1/search/suggestions` and `GET /v1/search/hashtags`; keep all search/project routes authenticated + device-gated. | FR-001, FR-005, NFR-003 | IT-001, IT-004, IT-011 |
| `appview/internal/api/search.go` | Change | Add handlers for unified suggestions and hashtag-query results; update submitted post/project handlers for new parsers; keep exact hashtag posts sort handling. | FR-001, FR-005, FR-006, FR-008, FR-010 | IT-001, IT-004, IT-005, IT-006, IT-017 |
| `appview/internal/api/search_request.go` | Change | Add `SearchSuggestionsRequest`, `HashtagSearchRequest`, text-only `PostSearchRequest`/`ProjectSearchRequest`, richer `ProjectListRequest`, craft-token normalization use, recent payload validation. | FR-001, FR-005, FR-006, FR-010, FR-014, FR-016 | UT-001, UT-004, UT-005, UT-007, UT-008, UT-013 |
| `appview/internal/api/search_response.go` | Change | Add suggestion response sections, hashtag result page/items, `crafts` on profile summaries. | FR-001, FR-002, FR-005 | AT-002, AT-005, IT-001, IT-004 |
| `appview/internal/api/search_store.go` | Change | Split exact hashtag feeds, relevance text search, project browse, hashtag query, and top hashtag logic; include `crafts` from `craftsky_profiles`. | FR-002, FR-005, FR-006, FR-007, FR-008, FR-009, FR-015 | UT-002, UT-003, UT-006, UT-011, IT-004, IT-005, IT-006, IT-008, IT-017 |
| `appview/internal/api/search_ranking.go` | Change | Add deterministic hashtag rank helpers and text-search relevance helper shape for tests. | FR-003, FR-005, FR-006 | UT-003, UT-006, AC-018 |
| `appview/internal/api/search_cursor.go` | Change | Add relevance cursor and hashtag-query cursor encoders/decoders; preserve chronological/popularity/profile cursors. | FR-005, FR-006, NFR-003 | IT-004, IT-005, IT-011, IT-017 |
| `appview/internal/api/craft_type.go` | Create | Central supported craft tokens, aliases, defaults, and canonicalization helpers. | FR-014, FR-015 | UT-005, AT-008, IT-007, IT-008 |
| `appview/internal/api/search_suggestions.go` or `suggestion_store.go` | Create | Shared profile/hashtag suggestion store helpers used by search suggestions and facet compatibility wrappers. | FR-001, FR-002, FR-003, FR-004 | UT-002, UT-003, IT-001, IT-002, IT-003 |
| `appview/internal/api/facet_store.go` | Change | Delegate mention/hashtag suggestion queries to shared suggestion helpers while preserving current facet response shape and tolerant empty-query behavior. | FR-002, FR-003, FR-004 | AT-003, IT-002, IT-003, REG-001 |
| `appview/internal/api/facet_response.go` | Change if needed | Carry internal `Crafts []string` on rows; keep public facet fields compatible. Additive public `crafts` is optional, not required for rich-text callers. | FR-002, FR-004 | IT-002, IT-015 |
| `appview/migrations/000020_search_refinements.up.sql` / `.down.sql` | Create | Update recent-search check constraint to include `query` while retaining deliberate legacy `post`/`project` support; optional indexes only if implemented. | FR-016, NFR-004 | IT-010, IT-016 |
| `appview/internal/api/*_test.go`, `appview/internal/routes/routes_test.go` | Change/Create | Add and update AppView unit/integration/regression tests from `02-acceptance-tests.md`. | NFR-005 | UT-001 through UT-013, IT-001 through IT-011, IT-017, REG-001 through REG-004 |
| `app/lib/search/models/search_suggestions.dart` | Create | Search-owned unified suggestion models with profile and hashtag sections plus `hasMore`. | FR-001, FR-002, FR-003, FR-012 | UT-009, IT-012, IT-013 |
| `app/lib/search/models/hashtag_search_page.dart` | Create | Paginated committed Hashtags tab result models. | FR-005, FR-012 | UT-009, UT-010, IT-012, IT-013 |
| `app/lib/search/models/blank_search_data.dart` | Create | UI-agnostic aggregate for blank-search recents + top hashtag groups. | BR-002, FR-012 | AT-001, IT-013 |
| `app/lib/search/models/profile_search_page.dart` | Change | Add `crafts: List<String>` to profile search/suggestion summaries. | FR-002 | AT-002, UT-009, IT-012 |
| `app/lib/search/models/recent_search.dart` | Change | Add `query` recent payload; change future `hashtag` payload output to `tag` only; change `profile` payload to stable selected profile identity; retain legacy decode for `post`/`project` if chosen. | FR-013, FR-016 | AT-007, UT-007, IT-010, IT-012, REG-004 |
| `app/lib/search/models/search_queries.dart` | Change | Add suggestion query and hashtag-result query; make submitted post/project search query models relevance-default and text-only. | FR-005, FR-006, FR-012 | UT-010, IT-013 |
| `app/lib/search/models/search_result_state.dart` | Change | Add `HashtagResultSearchState` and any blank/suggestion state helpers needed by providers. | FR-005, FR-012 | UT-010, IT-013 |
| `app/lib/search/data/search_api_client.dart` | Change | Add `/v1/search/suggestions` and `/v1/search/hashtags`; stop sending rich filters/sort to `/v1/search/projects`; keep exact hashtag posts sort. | FR-001, FR-005, FR-006, FR-008, FR-010, NFR-002 | UT-004, UT-009, UT-013, IT-012 |
| `app/lib/search/data/search_repository.dart`, `api_search_repository.dart` | Change | Expose repository methods for suggestions, hashtag-query pages, blank-search support, and refined recents. | FR-012, FR-016 | UT-012, IT-012, IT-013 |
| `app/lib/search/providers/search_suggestions_provider.dart` | Create | Bounded typeahead provider; returns empty local state for blank input rather than calling AppView repeatedly. | FR-001, RULE-007 | AT-002, UT-010, IT-013 |
| `app/lib/search/providers/hashtag_result_search_provider.dart` | Create | Independent paginated provider for submitted Hashtags tab results. | FR-005, FR-012 | AT-004, AT-005, UT-010, IT-013 |
| `app/lib/search/providers/blank_search_provider.dart` | Create | Combines recent searches and top hashtags for default supported full craft tokens; no rendering. | BR-002, FR-012 | AT-001, IT-013 |
| `app/lib/search/providers/post_search_provider.dart`, `project_search_provider.dart`, `profile_search_provider.dart`, `hashtag_search_provider.dart`, `top_hashtags_provider.dart`, `recent_searches_provider.dart` | Change | Update query signatures, pagination state, no auto-recent behavior, exact hashtag sort, and full-token defaults. | FR-006, FR-008, FR-012, FR-013 | AT-004, AT-005, AT-007, AT-008, UT-010, UT-012 |
| `app/lib/projects/models/project_browse_filters.dart`, `project_browse_query.dart` | Create | Project-owned browse/filter query models with craft types, filter families, sort, and pagination-independent equality. | FR-009, FR-011 | AT-006, UT-008, IT-014, REG-007 |
| `app/lib/projects/data/project_api_client.dart`, `project_repository.dart`, `api_project_repository.dart` | Change | Send supported filter families to `/v1/projects`; keep shared authenticated Dio. | FR-009, FR-010, FR-011 | AT-006, IT-014 |
| `app/lib/projects/providers/project_feed_provider.dart` | Change | Consume project browse query/filter models and paginate via `ProjectRepository`, never `SearchRepository`. | FR-011, RULE-005, RULE-009 | AT-006, IT-014, REG-007 |
| `app/lib/projects/options/project_option_catalogs.dart` | Change if needed | Expose default supported craft-token list for blank top hashtags and project browse tests. | FR-014, FR-015 | UT-005, AT-008 |
| `app/lib/bootstrap.dart` | Change | Register new `dart_mappable` mappers after model changes. | FR-012 | UT-009 |
| Generated Dart files `*.mapper.dart`, Riverpod `*.g.dart` | Change via build_runner | Regenerate only as required by model/provider changes. | FR-012 | Flutter focused tests |
| Flutter tests under `app/test/search/**`, `app/test/projects/**`, `app/test/shared/rich_text/**` | Change/Create | Implement test targets listed in acceptance spec. | NFR-005 | UT-009 through UT-013, IT-012 through IT-015, REG-001, REG-005 through REG-007 |

## 5. Services, Interfaces, And Data Flow

### 5.1 AppView route and handler contracts

New routes:

```text
GET /v1/search/suggestions?q=<text>&types=profiles,hashtags&profileLimit=5&hashtagLimit=5
GET /v1/search/hashtags?q=<text>&limit=25&cursor=<opaque>
```

Changed/confirmed routes:

```text
GET /v1/search/posts?q=<text>&limit=25&cursor=<opaque>
GET /v1/search/projects?q=<text>&limit=25&cursor=<opaque>
GET /v1/search/hashtags/{tag}/posts?sort=chronological|popular&limit=25&cursor=<opaque>
GET /v1/projects?craftType=<token-or-alias>&color=...&material=...&sort=chronological|popular&limit=25&cursor=<opaque>
```

Guardrails:

- `GET /v1/search/suggestions` rejects blank `q`, unknown `types`, invalid limits, and any `cursor` parameter.
- `GET /v1/search/hashtags` is committed hashtag-query search, not exact hashtag posts.
- `GET /v1/search/projects` requires non-empty `q` and rejects rich browse filters and unsupported `sort` parameters in this slice.
- `GET /v1/projects` rejects `q` and unknown filter keys; it owns browse filters and chronological/popular sort.
- All success JSON remains camelCase; paginated responses omit `cursor` when exhausted.

### 5.2 AppView partial signatures

```text
// search_request.go
type SearchSuggestionType string // "profiles", "hashtags"

type SearchSuggestionsRequest struct {
  Query        string
  Types        map[SearchSuggestionType]bool
  ProfileLimit int
  HashtagLimit int
}

type HashtagSearchRequest struct {
  Query  string
  Limit  int
  Cursor string
}

type PostSearchRequest struct {
  Query  string
  Limit  int
  Cursor string
}

type ProjectSearchRequest struct {
  Query  string
  Limit  int
  Cursor string
}

type ProjectListRequest struct {
  Sort    SearchSort
  Limit   int
  Cursor  string
  Filters map[string][]string // includes canonical craftType values
}
```

```text
// craft_type.go
const craftTypePrefix = "social.craftsky.feed.defs#"

var defaultSupportedCraftTypes = []string{
  "social.craftsky.feed.defs#knitting",
  "social.craftsky.feed.defs#crochet",
  "social.craftsky.feed.defs#sewing",
  "social.craftsky.feed.defs#embroidery",
  "social.craftsky.feed.defs#quilting",
}

func CanonicalCraftType(raw string) (string, error)
func CanonicalCraftTypes(raw []string, useDefaults bool) ([]string, error)
```

```text
// search_response.go
type SearchSuggestionsResponse struct {
  Profiles SuggestionProfileSection `json:"profiles"`
  Hashtags SuggestionHashtagSection `json:"hashtags"`
}

type SuggestionProfileSection struct {
  Items   []ProfileSearchSummary `json:"items"`
  HasMore bool                   `json:"hasMore"`
}

type SuggestionHashtagSection struct {
  Items   []HashtagSuggestionSummary `json:"items"`
  HasMore bool                       `json:"hasMore"`
}

type ProfileSearchSummary struct {
  ProfileAccountSummary
  ViewerIsFollowing bool     `json:"viewerIsFollowing"`
  Crafts            []string `json:"crafts"`
}

type HashtagSearchPageResponse struct {
  Items  []HashtagSearchResult `json:"items"`
  Cursor string                `json:"cursor,omitempty"`
}

type HashtagSearchResult struct {
  Tag             string `json:"tag"`
  PostsLast28Days int    `json:"postsLast28Days"`
}
```

### 5.3 Shared suggestion core

Use one AppView query/ranking core for search suggestions and facet suggestions. The shared helper should fetch `limit + 1` for the unified search endpoint so `hasMore` is accurate, and fetch exactly `limit` for existing facet endpoints.

```text
type ProfileSuggestionRow struct {
  DID, Handle, HandleLower string
  DisplayName, Description, AvatarCID, AvatarMime *string
  Crafts []string
  ViewerIsFollowing bool
  FollowedRank int
  RelevanceRank int
}

func SearchProfileSuggestions(ctx, pool, viewerDID, query, limit, now) (rows []ProfileSuggestionRow, hasMore bool, err error)
func SearchHashtagSuggestions(ctx, pool, query, limit, now) (rows []HashtagSuggestionRow, hasMore bool, err error)
```

Profile ranking order:

1. followed profiles first,
2. handle exact match,
3. handle prefix match,
4. handle substring match,
5. display name substring match,
6. description substring match,
7. handle lower ascending,
8. DID ascending.

Only indexed Craftsky profiles (`craftsky_profiles`) are eligible for v1. Include `craftsky_profiles.crafts` in rows. Existing `/v1/facets/mentions` should keep tolerant empty-query behavior and current field compatibility while using the shared rank source.

Hashtag suggestion/query count rules:

- normalize away leading `#`, trim, lower-case,
- count distinct visible top-level regular posts and project posts in the last 28 days,
- exclude replies/comments by the same visibility/top-level predicates used elsewhere,
- do not infer craft groups for regular posts.

### 5.4 Submitted search result data flow

```text
Flutter submitted search providers
  -> SearchRepository.searchPosts/searchProjects/searchProfiles/searchHashtags
  -> SearchApiClient /v1/search/posts|projects|profiles|hashtags
  -> AppView request parser
  -> SearchStore mode-specific query
  -> response page + opaque cursor
  -> provider appends unique items in that tab only
```

Posts tab:

- Requires `q`.
- Returns top-level `craftsky_posts` where `p.is_project = false`.
- Uses relevance cursor and orders by `score DESC, created_at DESC, uri DESC`.

Projects tab:

- Requires `q`.
- Returns top-level `craftsky_project_posts` joined to `craftsky_posts`.
- Does not accept browse filters in `/v1/search/projects`.
- Uses relevance cursor and the same stable tie-breakers.

Profiles tab:

- Keeps existing paginated route and profile rank cursor.
- Adds `crafts` to profile summaries.

Hashtags tab:

- Uses `GET /v1/search/hashtags`.
- Ranks exact match first, prefix matches next, then 28-day count descending, then tag ascending.
- Uses an opaque hashtag cursor that encodes the normalized query and rank tuple.

### 5.5 Relevance scoring design

Pin the scoring before implementing store tests:

```text
// AppView SQL sketch, not production code.
tsq := plainto_tsquery('simple', normalized_query)

post_score := ts_rank_cd(
  to_tsvector('simple', coalesce(p.text, '')),
  tsq
)

project_vector :=
  setweight(to_tsvector('simple', coalesce(pp.common_title, '')), 'A') ||
  setweight(to_tsvector('simple', coalesce(pp.pattern_name, '')), 'B') ||
  setweight(to_tsvector('simple', coalesce(array_to_string(pp.materials, ' '), '')), 'C') ||
  setweight(to_tsvector('simple', coalesce(array_to_string(pp.project_tags, ' '), '')), 'C') ||
  setweight(to_tsvector('simple', coalesce(array_to_string(pp.design_tags, ' '), '')), 'C') ||
  setweight(to_tsvector('simple', coalesce(p.text, '')), 'D')

project_score := ts_rank_cd(project_vector, tsq)

ORDER BY score DESC, p.created_at DESC, p.uri DESC
```

Implementation details may inline the SQL expression instead of adding a Go helper, but tests should assert that a higher score outranks newer weak matches and ties use `createdAt` then URI.

### 5.6 Exact hashtag feed data flow

Existing exact hashtag post search remains distinct from the committed Hashtags tab:

```text
selected hashtag / hashtag tap
  -> normalize safe path segment in Flutter and AppView
  -> GET /v1/search/hashtags/{tag}/posts?sort=chronological|popular
  -> SearchStore.SearchHashtagPosts(tag, sort, limit, cursor, now)
  -> combined regular posts + project posts with exact stored tag equality
```

Do not make this endpoint do substring matching. Do not auto-save a recent item from this fetch.

### 5.7 Recent-search payload contract

Server-side migration should allow new `query` recents and retain legacy `post`/`project` rows deliberately unless a separate cleanup is approved.

Future Flutter-generated recent payloads:

```json
{"type":"query","displayLabel":"alpaca socks","payload":{"q":"alpaca socks"}}
{"type":"hashtag","displayLabel":"#sockkal","payload":{"tag":"sockkal"}}
{"type":"profile","displayLabel":"Alice","payload":{"did":"did:plc:alice","handle":"alice.craftsky.social","displayName":"Alice","avatar":"https://..."}}
```

Rules:

- `query` payload contains `q` only.
- `hashtag` payload contains canonical `tag` only.
- `profile` payload contains stable DID plus display metadata for direct navigation; no query rerun is required.
- Typeahead/result/exact hashtag/project browse fetches do not mutate recents.
- Project browse/filter interactions do not generate Search recents.

## 6. State, Providers, Controllers, Or DI

### 6.1 Existing provider pattern

The app uses Riverpod code generation:

- `@Riverpod(keepAlive: true)` for API clients/repositories.
- `@riverpod class ... extends _$...` for paginated async state with `loadMore()`.
- fake repositories override provider seams in tests.

Keep this pattern. Regenerate Riverpod/Dart mapper files with build_runner during implementation.

### 6.2 Search provider graph

```text
dioProvider
  -> searchApiClientProvider
    -> searchRepositoryProvider
      -> searchSuggestionsProvider(SearchSuggestionQuery)
      -> postSearchProvider(PostSearchQuery)
      -> projectSearchProvider(ProjectSearchQuery)
      -> profileSearchProvider(ProfileSearchQuery)
      -> hashtagResultSearchProvider(HashtagResultSearchQuery)
      -> hashtagSearchProvider(HashtagSearchQuery exact tag)
      -> topHashtagsProvider(TopHashtagsQuery)
      -> recentSearchPageProvider
      -> saveRecentSearchProvider / deleteRecentSearchProvider
      -> blankSearchProvider
```

Provider responsibilities:

- `searchSuggestionsProvider`: returns a UI-agnostic `SearchSuggestions` value. If `query.q.trim().isEmpty`, return empty sections locally and avoid calling AppView repeatedly. If non-empty, call `/v1/search/suggestions`.
- `postSearchProvider`: submitted Posts tab only; required text query; no default chronological sort.
- `projectSearchProvider`: submitted Projects tab only; required text query; no browse filters or project browse sort.
- `profileSearchProvider`: existing profile pagination, with `crafts` decoded.
- `hashtagResultSearchProvider`: committed Hashtags tab, paginated independently from exact hashtag post feeds.
- `hashtagSearchProvider`: exact hashtag posts with `SearchSort.chronological`/`popular`.
- `blankSearchProvider`: combines `recentSearchPageProvider` with `topHashtagsProvider` for the default full craft tokens.

### 6.3 Project provider graph

```text
dioProvider
  -> projectApiClientProvider
    -> projectRepositoryProvider
      -> projectFeedProvider(ProjectBrowseQuery)
```

Project browse models belong under `app/lib/projects/models/**`:

```text
class ProjectBrowseQuery {
  const ProjectBrowseQuery({
    this.craftTypes = const [],
    this.filters = const ProjectBrowseFilters(),
    this.sort = SearchSort.chronological,
  });
}

class ProjectBrowseFilters {
  final List<String> projectType;
  final List<String> patternDifficulty;
  final List<String> color;
  final List<String> material;
  final List<String> designTag;
  final List<String> projectTag;
}
```

`ProjectFeed` should call `ProjectRepository.listProjects(query: ..., limit, cursor)` and should not import or call `SearchRepository`.

### 6.4 Repository interface changes

```text
abstract interface class SearchRepository {
  Future<SearchSuggestions> searchSuggestions(SearchSuggestionQuery query);
  Future<HashtagSearchPage> searchHashtags({required String q, int? limit, String? cursor});
  Future<SearchPostPage> searchPosts({required String q, int? limit, String? cursor});
  Future<SearchPostPage> searchProjects({required String q, int? limit, String? cursor});
  Future<ProfileSearchPage> searchProfiles({required String q, int? limit, String? cursor});
  Future<SearchPostPage> searchHashtagPosts(String tag, {SearchSort? sort, int? limit, String? cursor});
  Future<TopHashtagsResponse> topHashtags({List<String>? craftTypes, int? limit});
  Future<RecentSearchPage> listRecentSearches();
  Future<RecentSearchItem> saveRecentSearch(SaveRecentSearchRequest request);
  Future<void> deleteRecentSearch(String id);
}

abstract interface class ProjectRepository {
  Future<PostPage> listProjects({required ProjectBrowseQuery query, int? limit, String? cursor});
}
```

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

No rendered UI work is part of this slice.

Allowed Flutter source changes:

- data-layer models,
- API clients,
- repositories,
- providers/state,
- mapper/provider generated files,
- stub compile compatibility only if required by signatures.

Disallowed Flutter source changes:

- no rendered Search UI,
- no result tabs,
- no recent management page,
- no project filter controls,
- no new cards/layouts/scroll behavior,
- no visual route/navigation behavior changes,
- no local persistent search-history store.

`SearchPage`, `ProjectsPage`, `router.dart`, and `app_shell.dart` should remain visually equivalent unless generated/compile compatibility requires a minimal no-op update.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Blank unified suggestion `q` | AppView returns `400 validation_error`; Flutter suggestion provider short-circuits blank input to empty sections to avoid noisy calls. | FR-001, NFR-003 | UT-001, EC-001, IT-011 |
| Suggestion section has more matches than limit | AppView queries `limit + 1`, returns only `limit` items and `hasMore: true`; no cursor. | FR-001, RULE-007 | AT-002, AC-017, IT-001 |
| Suggestion query starts with `#` or `@` | Match helpers strip the relevant leading prefix for hashtag/profile matching while returning canonical hashtag tags and normal handles. | FR-001, FR-003 | EC-002, UT-001, UT-003 |
| Existing facet endpoint empty query | Preserve existing tolerant empty `items: []` response for `/v1/facets/*`; do not apply unified search blank-query validation there. | FR-004 | AT-003, IT-015, REG-001 |
| Committed Hashtags tab no matches | Return `items: []` and no cursor. Flutter state has `hasMore == false`. | FR-005 | EC-004, IT-004 |
| Invalid exact hashtag path | AppView returns standard validation error; Flutter path-encodes one segment and maps API error. | FR-008, NFR-003 | UT-004, IT-006 |
| Submitted Posts/Projects matching same project | Posts query filters `p.is_project = false`; Projects query joins project posts only. | FR-007, RULE-006 | AT-004, IT-005, REG-003 |
| Higher relevance vs newer weaker text match | Relevance score orders first; `createdAt` and URI only break ties. | FR-006, RULE-008 | UT-006, IT-005, AC-018 |
| `/v1/search/projects` receives browse filter | Reject with standard validation error. Do not ignore. | FR-010, RULE-005 | UT-008, IT-009, AC-019 |
| `/v1/projects` receives unknown craft type/filter key | Reject with standard validation error instead of returning all projects. | FR-009, FR-014 | UT-005, UT-008, IT-008 |
| Craft type omitted for top hashtags | Use default full-token groups: knitting, crochet, sewing, embroidery, quilting; include empty groups. | BR-002, FR-015 | UT-011, IT-007, AC-013 |
| Load more while a tab is loading | Existing provider guard `if (!state.hasValue || state.isLoading) return` remains; each provider owns its own cursor. | FR-006, FR-012 | UT-010, EC-011 |
| Search/fetches before listing recents | No recent mutation occurs unless explicit save/delete provider is called. | FR-013, RULE-002 | AT-007, UT-012, IT-010 |
| Already-deleted/not-owned recent delete | Preserve idempotent delete behavior and viewer scoping. | FR-013 | EC-010, REG-004 |
| Legacy `post`/`project` recent rows | Retain decode/list compatibility if present; do not generate project browse/filter recents from Flutter. | FR-016, RULE-009 | GAP-005, REG-004 |
| Missing auth/device header | Existing middleware returns `/v1/` error envelopes; route tests include new endpoints. | NFR-002, NFR-003 | AT-009, IT-011 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `appview/internal/api/search_request_test.go` | Parse `GET /v1/search/suggestions` with `q`, `types`, limits, blank q, invalid cursor. | `ParseSearchSuggestionsRequest` does not exist. |
| 2 | UT-004, UT-013 | `appview/internal/api/search_request_test.go`, `app/test/search/data/search_api_client_test.dart` | Exact hashtag path normalization and `sort=chronological|popular` plus invalid sort. | Exact request/client tests miss new sort/invalid cases. |
| 3 | UT-005, AT-008 | `appview/internal/api/search_request_test.go`, `app/test/projects/options/project_option_catalogs_test.dart` | Full craft tokens and aliases for default supported crafts. | No shared canonicalizer; defaults use bare/missing tokens. |
| 4 | UT-007 | `appview/internal/api/search_request_test.go`, `app/test/search/models/recent_search_test.dart` | `query`, tag-only `hashtag`, selected-profile payloads; invalid blank/overlong data. | Recent type `query` and direct profile payload do not exist. |
| 5 | UT-008 | `appview/internal/api/search_request_test.go` | `/v1/projects` accepts browse filters; `/v1/search/projects` rejects them and requires text search. | Current parser accepts rich filters on search-projects and `/v1/projects` rejects filters. |
| 6 | UT-002, IT-002 | `appview/internal/api/facet_suggestion_test.go`, `search_profile_rank_test.go`, `facet_test.go` | Viewer follows, Craftsky profiles with `crafts`, non-Craftsky profile, same query via search/facet. | Ranking logic is duplicated and profile summaries lack crafts. |
| 7 | UT-003, IT-003, IT-004 | `appview/internal/api/search_ranking_test.go`, `search_store_test.go`, `facet_test.go`, `routes_test.go` | Mixed-case hashtag tags, exact/prefix/substring, old rows, hidden rows, multiple pages. | No committed hashtag-query endpoint or cursor exists. |
| 8 | UT-006, IT-005 | `appview/internal/api/search_ranking_test.go`, `search_store_test.go` | Regular vs project posts; relevance strength vs recency; equal-score ties. | Existing search defaults to chronological/popular and Posts includes projects. |
| 9 | UT-011, IT-007 | `appview/internal/api/search_store_test.go`, `app/test/search/models/top_hashtags_test.dart` | Project posts with full craft tokens, empty default groups, regular posts with same tags. | Top hashtags use bare default crafts and omit embroidery/full tokens. |
| 10 | IT-001, IT-011 | `appview/internal/routes/routes_test.go` | New route registration plus auth/device/error-envelope cases. | Routes do not exist. |
| 11 | IT-006, IT-017, REG-002 | `appview/internal/api/search_store_test.go`, `routes_test.go`, Flutter client/provider tests | Exact tag regular/project posts with chronological/popular ordering and invalid sort. | Exact feed tests do not cover popular/chronological sort combinations fully. |
| 12 | IT-008, IT-009 | `appview/internal/api/search_store_test.go`, `routes_test.go` | Project browse filters, aliases/full tokens, popular sort; rich filters on search-projects. | `/v1/projects` lacks filter families; `/v1/search/projects` accepts browse filters. |
| 13 | IT-010, REG-004 | `appview/internal/api/search_recent_store_test.go`, migration test path if present | New recent type/check constraint, explicit save/list/delete, no auto-mutation. | Constraint excludes `query`; payload validator uses old profile/post/project shapes. |
| 14 | UT-009, IT-012 | `app/test/search/models/*_test.dart`, `app/test/search/data/search_api_client_test.dart`, `search_repository_test.dart` | Mock JSON for suggestions, hashtag pages, profile crafts, full craft groups, refined recents. | Flutter models/client methods do not exist. |
| 15 | UT-010, IT-013 | `app/test/search/providers/*_provider_test.dart` | Fake repository with independent cursors, duplicate items, blank suggestions, blank-search aggregate. | Providers for suggestions/hashtag results/blank search do not exist; project/posts send old sort/filter args. |
| 16 | IT-014, REG-007 | `app/test/projects/data/project_api_client_test.dart`, `app/test/projects/providers/project_feed_provider_test.dart` | Project browse query/filter model with search repository spy. | Project API/client/provider lacks filter families and tests. |
| 17 | IT-015, REG-001 | Existing `app/test/shared/rich_text/**` plus AppView facet tests | Existing facet response shapes and tolerant errors. | Shared refactor may break `/v1/facets/*` compatibility if not guarded. |
| 18 | AT-001, REG-005, MAN-001 | `app/test/search/search_page_test.dart` and source-diff review | Inspect Search/Projects pages/router. | Any rendered UI change violates scope. |
| 19 | MAN-002, MAN-003 | Manual source/query review | Check no PDS/local history path; inspect new SQL/indexes. | Architecture/performance risks may be undocumented. |

Focused commands for the TDD builder:

```text
cd appview && go test ./internal/api ./internal/routes -count=1
cd app && dart run build_runner build --delete-conflicting-outputs
cd app && flutter test test/search test/shared/rich_text test/projects
```

Use `just dev-d` before AppView integration tests that need Postgres, and `just test` for the broader AppView suite.

## 10. Sequencing And Guardrails

- First TDD step: `UT-001` for `ParseSearchSuggestionsRequest` and route contract assumptions for `GET /v1/search/suggestions`.
- Recommended implementation sequence:
  1. AppView request parsers/craft canonicalizer/recent payload validators.
  2. AppView response structs and route registration for suggestions/hashtag-query endpoints.
  3. Shared suggestion helpers and facet compatibility wrappers.
  4. Hashtag-query store/cursor and route tests.
  5. Relevance text-search split for Posts/Projects and project browse/search boundary changes.
  6. Top hashtag full craft-token defaults and project browse filter support.
  7. Recent-search migration/store behavior.
  8. Flutter model/client/repository updates.
  9. Flutter search providers and blank-search aggregate.
  10. Flutter project browse models/client/provider tests.
  11. Rich-text facet regressions, no-UI source review, and focused commands.
- Dependencies between work items:
  - Flutter model/client work depends on AppView response shapes being pinned.
  - Provider tests depend on repository method signatures and fake repository updates.
  - Project provider tests depend on project browse query/filter model creation.
  - Recent store tests depend on the migration/check constraint decision.
- Guardrails:
  - Do not edit lexicons; no ADR is needed because no lexicon change is planned.
  - Do not add rendered UI, visual navigation, or routes.
  - Do not store recents on PDS or in local persistent Flutter storage.
  - Do not make Flutter call PDS directly or hold PDS tokens.
  - Do not make typeahead/result fetches auto-save recents.
  - Do not parse or construct opaque cursors in Flutter.
  - Do not couple Search UI data providers to rich-text facet repository interfaces.
  - Do not silently ignore invalid `/v1/search/projects` browse filters.
  - Keep `/v1/facets/*` compatible and tolerant.
  - Add migrations only for the recent check constraint or proven supporting indexes.
- Out of scope:
  - Rendered Search UI, Projects UI, tabs, cards, layouts, management pages, filters UI, and navigation behavior.
  - Distinct saved-search feature/table/routes/providers.
  - External atproto account fallback for profile search/suggestions.
  - Semantic search, embeddings, typo tolerance, recommendations, analytics, polling, or push.
  - Public unauthenticated search endpoints.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Legacy `post`/`project` recent payload rows may exist from the pre-UI foundation. | Removing decode support could break existing local/dev data and tests unexpectedly. | Retain legacy server/list and Flutter decode support unless a separate cleanup is approved; do not generate those recents from future Flutter Search/project browse code. |
| CPQ-002 | Non-blocking | Hashtag substring query over unnested tags may need indexes or future materialization for production scale. | Focused tests prove deterministic behavior but not production cardinality. | Keep limits bounded, use existing visibility predicates, review query shape manually, and add an index migration only if implementation/query review shows it is needed. |
| CPQ-003 | Non-blocking | Suggestion default per-section limits are not product-specified. | UI may later want a different default top-N. | Use `5` default and `25` max for unified search suggestions; callers can pass explicit limits. Existing facet endpoints keep their `10` default and `25` max. |
| CPQ-004 | Non-blocking | Profile recents can store stale display metadata after handle/display changes. | Recent list display may be stale until future profile hydration. | Store stable DID plus handle/display/avatar metadata for direct navigation; profile hydration is future work per `GAP-004`. |
| CPQ-005 | Non-blocking | Lowercasing non-craft project filter tokens can be surprising for camelCase lexicon values. | Comparisons still work when SQL lowers both sides, but responses do not echo filter values. | Preserve current lower-comparison pattern for non-craft filters in this slice; only craftType has canonical full-token response requirements. |

Blocking open questions: None identified.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-001` for parsing/validating `GET /v1/search/suggestions` as a bounded, non-paginated, authenticated top-N suggestion request.
- First focused command after adding the first failing test:

  ```text
  cd appview && go test ./internal/api -run TestParseSearchSuggestionsRequest -count=1
  ```

- Broader focused commands after AppView and Flutter slices are in place:

  ```text
  cd appview && go test ./internal/api ./internal/routes -count=1
  cd app && dart run build_runner build --delete-conflicting-outputs
  cd app && flutter test test/search test/shared/rich_text test/projects
  ```

- Notes:
  - Treat `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this file as source of truth.
  - If implementation pressure suggests changing route names, suggestion section shapes, recent payload shapes, or `/v1/search/projects` validation behavior, stop and revise requirements/tests first.
  - Keep manual checks `MAN-001` through `MAN-003` in the implementation notes because this slice intentionally has no rendered UI and cannot fully prove production query performance with focused fixtures.
