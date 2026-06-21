# TDD Implementation Plan: Search Refinements Before UI Slice

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated after each TDD loop.
- Preserve the non-UI scope: no rendered Search/Projects UI, tabs, cards, layouts, route/navigation behavior, or local persistent search-history storage.
- Preserve `/v1/` API conventions: authenticated session, `X-Craftsky-Device-Id`, camelCase JSON, standard error envelopes, bounded limits, and opaque cursors.
- Preserve `/v1/facets/*` compatibility and keep project browsing/filtering under `/v1/projects` and `app/lib/projects/**`.

## Approved Test Order
Mirrors `04-coding-plan.md` §9. Each loop will be executed red-green-refactor and updated below.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001 | FR-001, NFR-003, RULE-007 | AC-002, AC-014, AC-017 | `ParseSearchSuggestionsRequest` does not exist. |
| 2 | UT-004 | FR-008, NFR-003, RULE-004 | AC-008, AC-014 | Exact hashtag request tests miss invalid/safe normalization cases. |
| 3 | UT-013 | FR-008, FR-012, NFR-003, RULE-004 | AC-008, AC-014, AC-021 | Exact hashtag sort tests miss chronology/popular/invalid cases. |
| 4 | UT-005 | FR-009, FR-014, FR-015 | AC-009, AC-013 | No shared full-token craft canonicalizer/defaults. |
| 5 | AT-008 | BR-002, FR-014, FR-015 | AC-013 | Craft-token normalization acceptance coverage not yet passing. |
| 6 | UT-007 | BR-006, FR-013, FR-016, RULE-001, RULE-009 | AC-011, AC-015, AC-020 | `query` recent and selected-profile payloads are unsupported. |
| 7 | UT-008 | FR-009, FR-010, NFR-003, RULE-005 | AC-009, AC-010, AC-019 | Search-project parser accepts rich browse filters; `/v1/projects` lacks filter parser support. |
| 8 | UT-002 | FR-002, RULE-003 | AC-003, AC-004, AC-017 | Profile suggestion ranking is duplicated and summaries lack crafts. |
| 9 | IT-002 | FR-002, FR-004, RULE-003 | AC-003, AC-004 | Search and facet profile suggestion ranking/crafts not unified. |
| 10 | UT-003 | FR-003, FR-005, RULE-004 | AC-002, AC-004, AC-005 | Hashtag ranking/query helper not implemented. |
| 11 | IT-003 | FR-003, FR-004 | AC-002, AC-004, AC-005 | Hashtag suggestion core not shared with facet compatibility path. |
| 12 | IT-004 | FR-005, NFR-004, RULE-004 | AC-005, AC-014, AC-016 | Committed hashtag-query endpoint/cursor not implemented. |
| 13 | UT-006 | FR-006, FR-007, RULE-006, RULE-008 | AC-006, AC-007, AC-018 | Submitted post/project search is not relevance-first/disjoint. |
| 14 | IT-005 | FR-006, FR-007, RULE-006, RULE-008 | AC-006, AC-007, AC-018 | Store tests do not prove relevance-first disjoint tabs. |
| 15 | UT-011 | BR-002, FR-015 | AC-012, AC-013 | Top hashtags use bare/default craft behavior and omit empty supported groups. |
| 16 | IT-007 | BR-002, FR-014, FR-015 | AC-012, AC-013 | Top hashtag route/store does not expose canonical full craft groups. |
| 17 | IT-001 | FR-001, RULE-007 | AC-002, AC-014, AC-017 | Unified suggestions route does not exist. |
| 18 | IT-011 | NFR-002, NFR-003 | AC-014 | New/changed route auth/device/validation coverage absent. |
| 19 | IT-006 | FR-008, NFR-003, RULE-004 | AC-008, AC-014 | Exact hashtag feed tests lack combined regular/project exact matching coverage. |
| 20 | IT-017 | FR-008, FR-012, NFR-003, RULE-004 | AC-008, AC-021 | Exact hashtag chronological/popular sort integration coverage absent. |
| 21 | REG-002 | FR-008, FR-012, NFR-003, RULE-004 | AC-008, AC-021 | Exact hashtag regression coverage does not include new sort contract. |
| 22 | IT-008 | FR-009, FR-010, FR-014, RULE-005 | AC-009, AC-010, AC-013, AC-014 | `/v1/projects` lacks browse filter families/craft alias parity. |
| 23 | IT-009 | FR-010, RULE-005 | AC-009, AC-019 | `/v1/search/projects` browse-filter rejection not enforced. |
| 24 | IT-010 | BR-006, FR-013, FR-016, RULE-001, RULE-002, RULE-009 | AC-011, AC-015, AC-020 | Recent persistence cannot store `query` refined payloads. |
| 25 | REG-004 | BR-006, FR-013, FR-016, RULE-001, RULE-002, RULE-009 | AC-011, AC-015, AC-020 | Recent regression coverage needs refined payloads and explicit mutations. |
| 26 | UT-009 | FR-012, FR-015, NFR-002 | AC-001, AC-002, AC-005, AC-006, AC-008, AC-012, AC-013, AC-014 | Flutter models/client mapping for suggestions/hashtags/full tokens missing. |
| 27 | IT-012 | FR-001, FR-005, FR-012, FR-016, NFR-002 | AC-002, AC-005, AC-006, AC-008, AC-014, AC-020 | Flutter SearchApiClient/repository lacks new contracts. |
| 28 | UT-010 | FR-006, FR-012 | AC-006, AC-016 | Flutter independent pagination providers missing for hashtag results/suggestions/blank state. |
| 29 | IT-013 | BR-001, BR-002, FR-006, FR-012, RULE-008 | AC-006, AC-012, AC-016, AC-018 | Flutter search providers lack blank-search and independent tab states. |
| 30 | IT-014 | BR-005, FR-011, RULE-005, RULE-009 | AC-010, AC-011, AC-014 | Project browse provider lacks filter model and boundary assertions. |
| 31 | REG-007 | BR-005, FR-010, FR-011, RULE-005 | AC-009, AC-010, AC-019 | Project/search boundary regression coverage absent. |
| 32 | IT-015 | FR-004 | AC-004 | Existing rich-text facet compatibility must remain green after refactor. |
| 33 | REG-001 | BR-003, FR-002, FR-003, FR-004, RULE-003 | AC-003, AC-004 | Facet autocomplete regression suite must remain green. |
| 34 | AT-001 | BR-001, BR-002, FR-012, NFR-001 | AC-001, AC-012 | Blank-search non-UI providers missing. |
| 35 | REG-005 | BR-001, FR-012, NFR-001 | AC-001 | No-rendered-UI regression must be verified by tests/source review. |
| 36 | MAN-001 | BR-001, FR-012, NFR-001 | AC-001 | Source diff review required. |
| 37 | MAN-002 | NFR-002, RULE-001 | AC-014, AC-015 | Architecture/privacy source review required. |
| 38 | MAN-003 | FR-009, NFR-004 | AC-009, AC-016 | Query/index boundedness review required. |

## Execution Log

### Setup
- Workflow documents read: `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, `04-coding-plan.md`.
- Existing `05-implementation-plan.md`: none found; this file created.
- Document review status: Approved with notes; no blocking gaps.
- Relevant AppView and Flutter search/facet/project files inspected before code changes.
- Current step: continue approved AppView parser/normalization loops.

### Step 1: UT-001 / FR-001, NFR-003, RULE-007
- Write failing test: Added `TestParseSearchSuggestionsRequest` in `appview/internal/api/search_request_test.go` for trimmed non-empty `q`, default/selected types, per-section limits, invalid cursor, blank query, unknown types, over-limit values, and overlong query.
- Run command: `go test ./internal/api -run TestParseSearchSuggestionsRequest -count=1`
- Confirmed failure: build failed because `api.ParseSearchSuggestionsRequest`, `api.SearchSuggestionType`, `api.SearchSuggestionTypeProfiles`, and `api.SearchSuggestionTypeHashtags` were undefined.
- Implement: Added `SearchSuggestionsRequest`, `SearchSuggestionType` constants, default/max suggestion limits, `ParseSearchSuggestionsRequest`, and type parsing to `appview/internal/api/search_request.go`.
- Run command: `go test ./internal/api -run TestParseSearchSuggestionsRequest -count=1` → passed.
- Nearby command: `go test ./internal/api -run 'TestParse(SearchSuggestions|Post|Profile|Project|Top)|TestDecodeSaveRecent|TestNormalizeHashtag' -count=1` → passed.
- Refactor: None beyond the small parser helper.
- Notes: This loop anchors the strict validation behavior from DR-001 and AC-014; endpoint/handler integration remains for IT-001/IT-011.

### Step 2: UT-004 / FR-008, NFR-003, RULE-004
- Write failing test: Added `TestParseExactHashtagPostsRequestNormalizesSafePathTag` in `appview/internal/api/search_request_test.go` for optional leading `#`, mixed case, trimming, and invalid safe path-segment values including spaces, slashes, repeated `#`, empty values, and controls.
- Run command: `go test ./internal/api -run TestParseExactHashtagPostsRequestNormalizesSafePathTag -count=1`
- Confirmed failure: build failed because `api.ParseExactHashtagPostsRequest` was undefined.
- Implement: Added `ExactHashtagPostsRequest` with normalized `Tag` and `ParseExactHashtagPostsRequest` delegating to `NormalizeHashtagPathValue`.
- Test setup fix: Adjusted the test URL fixture to avoid raw spaces in the synthetic request path and set the path value directly.
- Run command: `go test ./internal/api -run TestParseExactHashtagPostsRequestNormalizesSafePathTag -count=1` → passed.
- Nearby command: `go test ./internal/api -run 'Test(ParseExactHashtagPostsRequest|NormalizeHashtagPathValue)' -count=1` → passed.
- Refactor: None.
- Notes: Handler still parses inline; later exact-hashtag sort loop will extend this parser and wire it into the handler.

### Step 3: UT-013 / FR-008, FR-012, NFR-003, RULE-004
- Write failing test: Added `TestParseExactHashtagPostsRequestSortLimitAndCursor` for omitted/chronological/popular sorts, bounded limits, opaque cursor preservation, invalid sort, invalid limit, and invalid cursor.
- Run command: `go test ./internal/api -run TestParseExactHashtagPostsRequestSortLimitAndCursor -count=1`
- Confirmed failure: build failed because `ExactHashtagPostsRequest` had no `Sort`, `Limit`, or `Cursor` fields.
- Implement: Extended `ExactHashtagPostsRequest` and `ParseExactHashtagPostsRequest` to parse `sort`, bounded `limit`, and opaque `cursor` while preserving normalized exact-tag behavior.
- Run command: `go test ./internal/api -run TestParseExactHashtagPostsRequestSortLimitAndCursor -count=1` → passed.
- Nearby command: `go test ./internal/api -run 'TestParseExactHashtagPostsRequest|TestNormalizeHashtagPathValue' -count=1` → passed.
- Refactor: Updated `SearchHashtagPostsHandler` to use `ParseExactHashtagPostsRequest`, preserving invalid-cursor vs validation-error envelopes.
- Refactor check: `go test ./internal/api -run 'TestParseExactHashtagPostsRequest|TestNormalizeHashtagPathValue' -count=1` → passed.
- Notes: Store-level exact matching and sort ordering remain for IT-006/IT-017.

### Step 4: UT-005 / FR-009, FR-014, FR-015
- Write failing test: Added `TestCanonicalCraftTypes` for full tokens, supported bare aliases, case/space normalization, de-duplication, default supported craft order, and unknown/blank craft rejection.
- Run command: `go test ./internal/api -run TestCanonicalCraftTypes -count=1`
- Confirmed failure: build failed because `api.CanonicalCraftType` and `api.CanonicalCraftTypes` were undefined.
- Implement: Added `appview/internal/api/craft_type.go` with canonical full-token defaults for knitting, crochet, sewing, embroidery, and quilting; accepted full tokens plus bare aliases; de-duplicated canonical outputs.
- Formatting: `gofmt -w internal/api/search_request.go internal/api/search_request_test.go internal/api/search.go internal/api/craft_type.go`.
- Run command: `go test ./internal/api -run TestCanonicalCraftTypes -count=1` → passed.
- Nearby command: `go test ./internal/api -run 'TestCanonicalCraftTypes|TestParse(ProjectList|ProjectSearch|TopHashtags)' -count=1` → passed.
- Refactor: None.
- Notes: Parser/store adoption of canonical tokens is covered by AT-008/IT-007/IT-008.

### Step 5: AT-008 / BR-002, FR-014, FR-015
- Write failing test: Added `TestCraftTypeRequestParsersUseCanonicalFullTokens` for project browse and top-hashtag request parsers accepting full tokens plus aliases, de-duping equivalent values, returning full tokens, including default full-token top-hashtag groups, and rejecting unknown tokens.
- Run command: `go test ./internal/api -run TestCraftTypeRequestParsersUseCanonicalFullTokens -count=1`
- Confirmed failure: project browse parser returned `[]string{"knitting", "social.craftsky.feed.defs#knitting", "crochet"}` instead of canonical de-duplicated full tokens.
- Implement: Updated `ParseProjectListRequest` and `ParseTopHashtagsRequest` to use `CanonicalCraftTypes`; updated existing parser expectations to canonical full-token outputs.
- Formatting: `gofmt -w internal/api/search_request.go internal/api/search_request_test.go internal/api/craft_type.go`.
- Run command: `go test ./internal/api -run TestCraftTypeRequestParsersUseCanonicalFullTokens -count=1` → passed.
- Nearby command: `go test ./internal/api -run 'Test(CanonicalCraftTypes|CraftTypeRequestParsersUseCanonicalFullTokens|ParseProjectListRequest|ParseTopHashtagsRequest)' -count=1` → passed.
- Refactor: None.
- Notes: Store response grouping still needs full-token and empty-group assertions in UT-011/IT-007.

### Step 6: UT-007 / BR-006, FR-013, FR-016, RULE-001, RULE-009
- Write failing test: Added `TestDecodeSaveRecentSearchRequestSupportsFutureSearchPayloads` for `query` payloads containing `q` only, hashtag payloads containing canonical `tag` only, and selected-profile payloads containing stable `did` plus normalized display metadata.
- Run command: `go test ./internal/api -run TestDecodeSaveRecentSearchRequestSupportsFutureSearchPayloads -count=1`
- Confirmed failure: `query` type was rejected, hashtag payloads added a `sort` field, and selected-profile identity payloads were rejected.
- Implement: Allowed `query` recent type; normalized query payloads to `q` only; changed hashtag recents to tag-only; replaced query-shaped profile recents with direct selected-profile payload normalization (`did`, normalized `handle`, optional trimmed `displayName`/`avatar`); retained legacy `post`/`project` normalization.
- Test updates: Updated existing typed-payload expectations to the refined hashtag/profile payload shapes and added invalid cases for blank query, hashtag extra sort, profile query-shape, and profile missing DID.
- Formatting: `gofmt -w internal/api/search_request.go internal/api/search_request_test.go`.
- Run command: `go test ./internal/api -run 'TestDecodeSaveRecentSearchRequest(SupportsFutureSearchPayloads|NormalizesTypedPayloads|RejectsInvalidTypedPayloads)' -count=1` → passed.
- Refactor: None.
- Notes: Persistence/check-constraint support for `query` remains for IT-010/REG-004.

### Step 7: UT-008 / FR-009, FR-010, NFR-003, RULE-005
- Write failing test: Added `TestProjectBrowseFiltersStayUnderProjectsAPI` for `/v1/projects` accepting craft/filter families and `/v1/search/projects` remaining text-only with required `q`, no sort override, and no rich filters.
- Run command: `go test ./internal/api -run TestProjectBrowseFiltersStayUnderProjectsAPI -count=1`
- Confirmed failure: build failed because `ProjectListRequest` had no `Filters` field for project browse filters.
- Implement: Added `ProjectListRequest.Filters`; updated `/v1/projects` parsing to accept supported browse filter families, canonicalize `craftType`, normalize/dedupe other filters, and reject unknown keys; updated `/v1/search/projects` parsing to require non-empty `q` and reject sort/filter params; wired `ListProjectsHandler` to pass `req.Filters` to the store.
- Test updates: Replaced old search-project filter parser expectations with text-only expectations and allowed `color` under `/v1/projects`.
- Formatting: `gofmt -w internal/api/search_request.go internal/api/search_request_test.go internal/api/search.go`.
- Run command: `go test ./internal/api -run TestProjectBrowseFiltersStayUnderProjectsAPI -count=1` → passed.
- Nearby command: `go test ./internal/api -run 'Test(ProjectBrowseFiltersStayUnderProjectsAPI|ParseProjectSearchRequestTextOnly|ParseProjectListRequest)' -count=1` → passed.
- Refactor: None beyond reusing the parsed browse-filter map in the handler.
- Notes: Store-level filtering/pagination/popular behavior remains for IT-008; route-level rejection remains for IT-009/IT-011.

### Step 8: UT-002 / FR-002, RULE-003
- Write failing test: Added `TestBuildProfileSearchSummaryIncludesCrafts` and `TestRankMentionSuggestionRowsUsesSharedProfileRelevance` to require profile summaries to carry `crafts` and facet mention ranking to use the same profile relevance order (display-name matches before description matches, after handle match tiers).
- Run command: `go test ./internal/api -run 'Test(BuildProfileSearchSummaryIncludesCrafts|RankMentionSuggestionRowsUsesSharedProfileRelevance)' -count=1`
- Confirmed failure: build failed because `ProfileSearchRow`/`ProfileSearchSummary` lacked `Crafts`, and `MentionSuggestionRow` lacked `Description`.
- Implement: Added `Crafts` to profile search rows/summaries with defensive copy, selected `craftsky_profiles.crafts` in profile search, added `Description`/`Crafts` to mention suggestion rows, and updated mention ranking to use `ProfileRelevanceRank` plus stable handle/DID tie-breakers.
- Formatting: `gofmt -w internal/api/search_response.go internal/api/facet_response.go internal/api/facet_store.go internal/api/search_store.go internal/api/search_profile_rank_test.go internal/api/facet_suggestion_test.go`.
- Run command: `go test ./internal/api -run 'Test(BuildProfileSearchSummaryIncludesCrafts|RankMentionSuggestionRowsUsesSharedProfileRelevance|RankMentionSuggestionRowsFollowedPrefixThenHandle|Profile)' -count=1` → passed.
- Nearby command: `go test ./internal/api -run 'Test(ProfileRelevanceRank|ProfileSearchRankTuple|BuildProfileSearchSummaryIncludesCrafts|RankMentionSuggestionRows)' -count=1` → passed.
- Refactor: Reused the profile relevance helper in facet ranking.
- Notes: Store/route equivalence for search vs facet ranking remains for IT-002.

### Step 9: IT-002 / FR-002, FR-004, RULE-003
- Write failing test: Added `TestSearchAndFacetProfileSuggestionsShareRankingAndCrafts` in `appview/internal/api/search_store_test.go`, seeding matching profile display-name and description rows and asserting search/facet overlapping order plus `crafts` in search summary.
- Initial command without DB: `go test ./internal/api -run TestSearchAndFacetProfileSuggestionsShareRankingAndCrafts -count=1 -v` skipped because `TEST_DATABASE_URL`/`DATABASE_URL` were unset.
- Dependency setup: `just dev-d` attempted; command timed out during image build after 120s, but `docker compose ps` showed the Postgres/AppView/Tap services already running.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchAndFacetProfileSuggestionsShareRankingAndCrafts -count=1`
- Confirmed failure: facet suggestions returned only the display-name match and omitted the description match, so the shared search/facet ranking overlap could not be established.
- Implement: Updated `FacetStore.SearchMentionSuggestions` SQL to include description matches and to order by the same followed/exact-handle/prefix/substring/display/description/handle/DID rank tiers used by profile search.
- Formatting: `gofmt -w internal/api/facet_store.go`.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchAndFacetProfileSuggestionsShareRankingAndCrafts -count=1` → passed.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(SearchAndFacetProfileSuggestionsShareRankingAndCrafts|SearchStore_SearchProfilesPaginatesByRankTuple|RankMentionSuggestionRows)' -count=1` → passed.
- Refactor: Replaced a panic-prone test assertion with explicit length assertions after the first DB-backed red run.
- Notes: Unified search suggestion endpoint still remains for IT-001; existing facet response shape remains unchanged.

### Step 10: UT-003 / FR-003, FR-005, RULE-004
- Write failing test: Added `TestRankHashtagResultsNormalizesAggregatesAndRanks` for leading-`#`/case normalization, duplicate count aggregation, negative-count clamping, substring filtering, and ranking exact first, prefix next by count/tag, substring last.
- Run command: `go test ./internal/api -run TestRankHashtagResultsNormalizesAggregatesAndRanks -count=1`
- Confirmed failure: build failed because `api.RankHashtagResults` was undefined.
- Implement: Added `RankHashtagResults`, `normalizeHashtagSearchTerm`, and `hashtagMatchRank` in `search_ranking.go`.
- Formatting: `gofmt -w internal/api/search_ranking.go internal/api/search_ranking_test.go`.
- Run command: `go test ./internal/api -run TestRankHashtagResultsNormalizesAggregatesAndRanks -count=1` → passed.
- Nearby command: `go test ./internal/api -run 'Test(RankHashtagResults|NormalizeHashtagSuggestionRows|EscapeFacetLikePattern)' -count=1` → passed.
- Refactor: None.
- Notes: Store-backed counts/cursors remain for IT-003/IT-004.

### Step 11: IT-003 / FR-003, FR-004
- Write failing test: Added `TestFacetHashtagSuggestionsUseHashtagResultRanking` to seed exact, prefix, and substring hashtag matches and assert facet suggestions normalize leading `#` and rank exact before prefix before substring despite higher substring counts.
- Run command: `gofmt -w internal/api/search_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestFacetHashtagSuggestionsUseHashtagResultRanking -count=1`
- Confirmed failure: facet suggestions returned no rows for `#Sock`, proving hashtag suggestion input normalization was missing (and the existing count-first SQL ordering would not satisfy the ranked contract).
- Implement: Updated `FacetStore.SearchHashtagSuggestions` to normalize leading `#`, use escaped substring matching, and order exact matches first, prefix matches next, then 28-day count descending and tag ascending.
- Formatting: `gofmt -w internal/api/facet_store.go`.
- Run command: `gofmt -w internal/api/facet_store.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestFacetHashtagSuggestionsUseHashtagResultRanking -count=1` → passed.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(FacetHashtagSuggestionsUseHashtagResultRanking|NormalizeHashtagSuggestionRows|EscapeFacetLikePattern)' -count=1` → passed.
- Refactor: None.
- Notes: A committed paginated hashtag-query endpoint/cursor remains for IT-004.

### Step 12: IT-004 / FR-005, NFR-004, RULE-004
- Write failing test: Added `TestSearchStore_SearchHashtagsRanksAndPaginates` for committed hashtag-query ranking, two-page cursor pagination, leading-`#` normalization, and invalid cursor handling.
- Run command: `gofmt -w internal/api/search_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_SearchHashtagsRanksAndPaginates -count=1`
- Confirmed failure: build failed because `SearchStore.SearchHashtags`, `HashtagSearchRequest`, and `HashtagSearchResult` did not exist.
- Implement: Added `HashtagSearchRequest`, `HashtagSearchResult`, `HashtagSearchPageResponse`, hashtag offset cursor helpers, and `SearchStore.SearchHashtags` with exact/prefix/count/tag ranking and opaque cursor pagination.
- Store run command: `gofmt -w internal/api/search_request.go internal/api/search_response.go internal/api/search_cursor.go internal/api/search_store.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_SearchHashtagsRanksAndPaginates -count=1` → passed.
- Additional failing route test: Added `hashtag search` to `TestAddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice`; focused route run failed because `GET /v1/search/hashtags` was not registered.
- Implement route/API: Added `ParseHashtagSearchRequest`, `SearchHashtagsHandler`, and registered authenticated/device-gated `GET /v1/search/hashtags`.
- Route command: `gofmt -w internal/api/search_request.go internal/api/search.go internal/routes/routes.go internal/routes/routes_test.go && go test ./internal/routes -run TestAddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice/hashtag_search_registered -count=1` → passed.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/routes -run 'Test(SearchStore_SearchHashtagsRanksAndPaginates|AddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice)' -count=1` → passed.
- Refactor: None.
- Notes: New route follows `/v1/` auth/device route registration; broader validation/error envelope coverage remains for IT-011.

### Step 13: UT-006 / FR-006, FR-007, RULE-006, RULE-008
- Write failing test: Updated the post/project store search test to `TestSearchStore_SearchPostsAndProjectsUseRelevanceAndDisjointTabs`, requiring submitted Posts to exclude projects/replies and rank an older stronger text match before a newer weak match, while Projects returns project posts only.
- Run command: `gofmt -w internal/api/search_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_SearchPostsAndProjectsUseRelevanceAndDisjointTabs -count=1`
- Confirmed failure: Posts search returned newer weak regular post first and included project posts in the Posts results.
- Implement: Added relevance-search branches for submitted post and project text search. Posts now filter `p.is_project = false` and order by `ts_rank_cd` score descending, then `created_at` descending, then URI descending. Projects now use a weighted project text vector and the same deterministic tie-breakers while preserving browse filters when internally supplied.
- Formatting: `gofmt -w internal/api/search_store.go`.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_SearchPostsAndProjectsUseRelevanceAndDisjointTabs -count=1` → passed.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'TestSearchStore_Search(PostsAndProjectsUseRelevanceAndDisjointTabs|ProjectsAppliesFilterSemantics|ProjectsPopularOrdersBrowseAllAndFilteredProjects)' -count=1` → passed.
- Refactor: None.
- Notes: Relevance pagination cursor support remains limited and should be covered or documented in IT-005 if not completed.

### Step 14: IT-005 / FR-006, FR-007, RULE-006, RULE-008
- Write failing test: Extended `TestSearchStore_SearchPostsAndProjectsUseRelevanceAndDisjointTabs` to page submitted Posts and Projects independently with `Limit: 1`, requiring opaque cursors to return the second relevance-ranked item and then exhaust.
- Run command: `gofmt -w internal/api/search_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_SearchPostsAndProjectsUseRelevanceAndDisjointTabs -count=1`
- Confirmed failure: page 1 returned the correct top relevance match but no cursor, so page 2 repeated page 1 and could not paginate independently.
- Implement: Added relevance cursor helpers encoding kind/query/score/createdAt/URI, wired submitted post and project relevance searches to decode cursor kind/query, apply keyset pagination over `(relevance_score, created_at, uri)`, and return next cursors when more rows exist.
- Formatting: `gofmt -w internal/api/search_cursor.go internal/api/search_store.go internal/api/search_store_test.go`.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_SearchPostsAndProjectsUseRelevanceAndDisjointTabs -count=1` → passed.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'TestSearchStore_Search(PostsAndProjectsUseRelevanceAndDisjointTabs|ProjectsAppliesFilterSemantics|ProjectsPopularOrdersBrowseAllAndFilteredProjects)|TestParse(Post|Project)SearchRequest' -count=1` → passed.
- Refactor: None.
- Notes: This completes the AppView store coverage for relevance-first ordering, disjoint Posts/Projects tabs, and independent post/project cursor pagination for IT-005.

### Step 15: UT-011 / BR-002, FR-015
- Write failing test: Updated `TestSearchStore_TopHashtagsGroupsDistinctProjectsAndEmptyCrafts` to seed project records with full craft tokens, include a regular post and hidden/old project rows that must not count, and require default top-hashtag groups for knitting, crochet, sewing, embroidery, and quilting as full tokens with empty groups included.
- Run command: `gofmt -w internal/api/search_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_TopHashtagsGroupsDistinctProjectsAndEmptyCrafts -count=1`
- Confirmed failure: `TopHashtags` defaulted to bare craft values, omitted embroidery, and returned empty counts for full-token project records.
- Implement: Updated `SearchStore.TopHashtags` to canonicalize requested craft types and use default supported full craft tokens when omitted.
- Formatting: `gofmt -w internal/api/search_store.go`.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_TopHashtagsGroupsDistinctProjectsAndEmptyCrafts -count=1` → passed.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(SearchStore_TopHashtagsGroupsDistinctProjectsAndEmptyCrafts|CraftTypeRequestParsersUseCanonicalFullTokens|CanonicalCraftTypes|ParseTopHashtagsRequest)' -count=1` → passed.
- Refactor: None.
- Notes: Route/client top-hashtag response coverage remains for IT-007 and Flutter model tests.

### Step 16: IT-007 / BR-002, FR-014, FR-015
- Write focused assertion: Extended the top-hashtag store test to call `TopHashtags` with mixed bare/full duplicate craft inputs (`knitting`, full crochet token, full knitting token), requiring canonical full-token groups in de-duplicated request order.
- Run command: `gofmt -w internal/api/search_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_TopHashtagsGroupsDistinctProjectsAndEmptyCrafts -count=1` → passed.
- Confirmed failure: The default/full-token behavior for IT-007 had already failed and been fixed in Step 15; the added mixed-alias assertion was satisfied by the Step 15 canonicalization implementation, so no additional code change was required in this loop.
- Implement: None beyond Step 15's `TopHashtags` canonicalization.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(SearchStore_TopHashtagsGroupsDistinctProjectsAndEmptyCrafts|CraftTypeRequestParsersUseCanonicalFullTokens|CanonicalCraftTypes|ParseTopHashtagsRequest)' -count=1` → passed.
- Refactor: None.
- Notes: AppView store/parser coverage now proves full-token defaults, empty supported groups, project-only counts, visible/recent filtering, and full/bare alias de-duplication. Flutter top-hashtag model/client coverage remains for UT-009/IT-012.

### Step 17: IT-001 / FR-001, RULE-007
- Write failing test: Added `TestSearchSuggestionsHandlerReturnsGroupedTopNSections` to seed matching Craftsky profiles and hashtags, call `GET /v1/search/suggestions` through the handler with per-section limit 1, and require grouped profile/hashtag sections, `hasMore: true`, profile crafts, normalized hashtag tag, and no pagination cursor in the response.
- Run command: `gofmt -w internal/api/search_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchSuggestionsHandlerReturnsGroupedTopNSections -count=1`
- Confirmed failure: build failed because `api.SearchSuggestionsHandler` and `api.SearchSuggestionsResponse` did not exist.
- Implement: Added `SearchSuggestionsResponse` with profile and hashtag sections, implemented `SearchSuggestionsHandler` using the existing parser plus profile/hashtag search helpers with per-section `hasMore`, initialized empty unrequested sections, and registered authenticated/device-gated `GET /v1/search/suggestions`.
- Route coverage: Added `search suggestions` to `TestAddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice`.
- Formatting: `gofmt -w internal/api/search.go internal/api/search_response.go internal/api/search_store_test.go internal/routes/routes.go internal/routes/routes_test.go`.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchSuggestionsHandlerReturnsGroupedTopNSections -count=1` → passed.
- Route command: `go test ./internal/routes -run TestAddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice/search_suggestions -count=1` → passed.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/routes -run 'Test(SearchSuggestionsHandlerReturnsGroupedTopNSections|ParseSearchSuggestionsRequest|AddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice)' -count=1` → passed.
- Refactor: None.
- Notes: This route returns bounded top-N sections only and intentionally exposes `hasMore` instead of any cursor.

### Step 18: IT-011 / NFR-002, NFR-003
- Write failing test: The `GET /v1/search/suggestions` auth/device route assertion added in Step 17 initially failed before route registration, covering the new-route portion of IT-011. Existing parser tests cover bounded limits, blank required queries, unsupported params, and invalid cursor envelopes for changed search routes.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/routes -run 'Test(Parse(SearchSuggestions|Hashtag|Post|Project|ExactHashtagPosts)Request|AddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice)' -count=1`
- Confirmed failure: No new failure after Step 17; the IT-011 route registration/auth/device failure was resolved by registering `GET /v1/search/suggestions`.
- Implement: No additional code changes required in this loop.
- Run command: same focused command → passed.
- Refactor: None.
- Notes: This verifies the current AppView route/parser contract surface for auth/device gating and standard validation/error-envelope paths. Flutter shared-Dio validation remains for IT-012.

### Step 19: IT-006 / FR-008, NFR-003, RULE-004
- Write failing test: Exact hashtag parser failures were covered in Steps 2–3, and store exact-equality coverage exists in `TestSearchStore_SearchHashtagPostsUsesStoredTagEqualityOnly` for combined regular/project results while excluding substring tags, text-only mentions, and replies.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(SearchStore_SearchHashtagPostsUsesStoredTagEqualityOnly|ParseExactHashtagPostsRequest)' -count=1`
- Confirmed failure: No new failure in this loop; prior parser/store red-green work already satisfied the IT-006 exact-feed contract.
- Implement: No additional code changes required.
- Run command: same focused command → passed.
- Refactor: None.
- Notes: Exact hashtag result mode remains separate from substring hashtag search/suggestions and returns top-level regular/project posts by stored exact tag equality.

### Step 20: IT-017 / FR-008, FR-012, NFR-003, RULE-004
- Write focused test: Added `TestSearchStore_SearchHashtagPostsSortsChronologicalAndPopular` for exact hashtag chronological pagination over regular/project posts and popular ordering using the existing deterministic engagement plus recency-decay formula.
- Run command: `gofmt -w internal/api/search_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_SearchHashtagPostsSortsChronologicalAndPopular -count=1` → passed.
- Confirmed failure: No additional failure; existing exact hashtag sort implementation already satisfied the new focused integration test.
- Implement: No code changes required.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(SearchStore_SearchHashtagPosts(SortsChronologicalAndPopular|UsesStoredTagEqualityOnly)|ParseExactHashtagPostsRequest|Decode.*Cursor)' -count=1` → passed.
- Refactor: None.
- Notes: Chronological exact hashtag pagination and popular exact hashtag ordering are now explicitly covered at the store level; Flutter exact hashtag client/provider sort coverage remains in later Flutter loops.

### Step 21: REG-002 / FR-008, FR-012, NFR-003, RULE-004
- Write failing test: Regression behavior is covered by the exact hashtag parser/store tests from Steps 19–20, including substring exclusion, reply exclusion, text-only mention exclusion, combined regular/project inclusion, chronological pagination, and popular ordering.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(SearchStore_SearchHashtagPosts(SortsChronologicalAndPopular|UsesStoredTagEqualityOnly)|ParseExactHashtagPostsRequest)' -count=1`
- Confirmed failure: No new failure; regression suite is green with the new exact-hashtag sort coverage.
- Implement: No additional code changes required.
- Run command: same focused command → passed.
- Refactor: None.
- Notes: Exact hashtag regression coverage now protects exact matching and sort semantics pending Flutter client/provider coverage later.

### Step 22: IT-008 / FR-009, FR-010, FR-014, RULE-005
- Write failing test: Updated project browse store tests to seed full craft tokens while using bare `craftType=knitting` filters, requiring `/v1/projects`-style browse filters to match canonical full-token records for chronological and popular browse paths. Added chronological cursor pagination over a filtered craft browse.
- Run command: `gofmt -w internal/api/search_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'TestSearchStore_SearchProjects(PopularOrdersBrowseAllAndFilteredProjects|AppliesFilterSemantics)' -count=1`
- Confirmed failure: filtered project browse returned empty results because store-level craft filters compared bare aliases directly against full-token project records.
- Implement: Updated `projectFilterValues` to canonicalize `craftType` filter values with `CanonicalCraftTypes` before SQL comparison, while preserving existing normalized non-craft filters.
- Formatting: `gofmt -w internal/api/search_store.go internal/api/search_store_test.go`.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'TestSearchStore_SearchProjects(PopularOrdersBrowseAllAndFilteredProjects|AppliesFilterSemantics)' -count=1` → passed.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/routes -run 'Test(SearchStore_SearchProjects(PopularOrdersBrowseAllAndFilteredProjects|AppliesFilterSemantics)|ProjectBrowseFiltersStayUnderProjectsAPI|ParseProjectListRequest|AddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice/project_list)' -count=1` → passed for API; routes package had no matching subtest in that focused pattern.
- Refactor: None.
- Notes: Project browse now handles bare/full craft filters against full-token records, filter families, chronological pagination, and popular sorting in store coverage. Route-level browse-filter rejection remains IT-009.

### Step 23: IT-009 / FR-010, RULE-005
- Write route test: Added `TestSearchProjectsRouteRejectsBrowseFilters` to call `/v1/search/projects?q=sock&craftType=knitting&material=alpaca` with auth/device headers and require a standard `400` error envelope.
- Run command: `gofmt -w internal/routes/routes_test.go && go test ./internal/routes -run TestSearchProjectsRouteRejectsBrowseFilters -count=1` → passed.
- Confirmed failure: No new failure; parser/handler validation from UT-008 already rejects browse filters before the store is reached.
- Implement: No additional code changes required.
- Nearby command: `go test ./internal/api ./internal/routes -run 'Test(ProjectBrowseFiltersStayUnderProjectsAPI|ParseProjectSearchRequestTextOnly|SearchProjectsRouteRejectsBrowseFilters)' -count=1` → passed.
- Refactor: None.
- Notes: `/v1/search/projects` is now covered as text-search-only at parser and route/error-envelope levels; `/v1/projects` owns browse filters.

### Step 24: IT-010 / BR-006, FR-013, FR-016, RULE-001, RULE-002, RULE-009
- Write failing test: Extended `TestSearchStore_RecentSearchLifecycleDedupesPrunesAndHardDeletes` to save a refined `query` recent with `q` only, verify normalized query payload persistence, and updated hashtag duplicate coverage to tag-only payloads.
- Run command: `gofmt -w internal/api/search_recent_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_RecentSearchLifecycleDedupesPrunesAndHardDeletes -count=1`
- Confirmed failure: `SaveRecentSearch query` failed with PostgreSQL check-constraint violation because `craftsky_recent_searches.search_type` allowed only `hashtag`, `profile`, `post`, and `project`.
- Implement: Added migration `000020_search_refinements` to widen the recent-search `search_type` check constraint to include `query` while retaining legacy `post`/`project`; updated the DB-backed test schema accordingly.
- Test adjustment: Decode JSONB when asserting normalized query payload because PostgreSQL may reformat JSONB whitespace.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSearchStore_RecentSearchLifecycleDedupesPrunesAndHardDeletes -count=1` → passed.
- Nearby command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(SearchStore_RecentSearchLifecycleDedupesPrunesAndHardDeletes|DecodeSaveRecentSearchRequest)' -count=1` → passed.
- Refactor: None.
- Notes: Migration was added but not run manually. Down migration deletes `query` rows before restoring the old check constraint, matching rollback constraints.

### Step 25: REG-004 / BR-006, FR-013, FR-016, RULE-001, RULE-002, RULE-009
- Write failing test: Regression behavior is covered by the refined recent payload parser tests plus the DB-backed lifecycle test extended in Step 24. Together they cover explicit save/list/delete, viewer scoping, idempotent not-owned/already-deleted deletion, refined `query`/`hashtag`/selected-profile payloads, and retained legacy `post`/`project` support.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(SearchStore_RecentSearchLifecycleDedupesPrunesAndHardDeletes|DecodeSaveRecentSearchRequest)' -count=1`
- Confirmed failure: No new failure; regression coverage is green after the `query` persistence migration and refined payload updates.
- Implement: No additional code changes required.
- Run command: same focused command → passed.
- Refactor: None.
- Notes: Recent/saved searches remain one explicit private AppView-backed history surface; no automatic recent mutation was added to search/suggestion/result fetch paths.

### Step 26: UT-009 / FR-012, FR-015, NFR-002
- Write failing test: Added `test/search/models/search_refinement_models_test.dart` requiring Flutter to decode unified suggestions with profile crafts, committed hashtag-query pages with opaque cursors, and refined `query`/tag-only `hashtag`/selected-profile recent payloads.
- Run command: `flutter test test/search/models/search_refinement_models_test.dart`
- Confirmed failure: compilation failed because `hashtag_search_page.dart`, `search_suggestions.dart`, `SearchSuggestionsMapper`, `HashtagSearchPageMapper`, `QueryRecentSearchPayload`, and `RecentSearchType.query` did not exist.
- Implement: Added Flutter search models for `HashtagSearchPage`/`HashtagSearchResult` and `SearchSuggestions`; added `crafts` to `ProfileSearchResult`; added `query` recent type, `QueryRecentSearchPayload`, tag-only hashtag recents, and selected-profile recent payloads; registered new mappers in `bootstrap.dart`; regenerated mapper files with build_runner.
- Test updates: Updated existing recent-search model tests to the refined future payload shapes while retaining legacy `post`/`project` decode coverage.
- Run command: `flutter test test/search/models/search_refinement_models_test.dart` → passed.
- Nearby command: `flutter test test/search/models/search_refinement_models_test.dart test/search/models/recent_search_test.dart test/search/models/top_hashtags_test.dart test/search/models/search_mapper_registration_test.dart` → passed.
- Refactor: None.
- Notes: Flutter model mapping now preserves full craft tokens as strings, keeps cursors opaque, and does not serialize project browse/filter recents as future Search recents.

### Step 27: IT-012 / FR-001, FR-005, FR-012, FR-016, NFR-002
- Write failing test: Updated `search_api_client_test.dart` to require `SearchApiClient.searchSuggestions`, committed `searchHashtags`, text-only `searchProjects`, and refined future Search recent payloads; added repository delegation coverage for suggestions and hashtags.
- Run command: `flutter test test/search/data/search_api_client_test.dart`
- Confirmed failure: compilation failed because `SearchSuggestionType`, `SearchApiClient.searchSuggestions`, and `SearchApiClient.searchHashtags` were undefined.
- Implement: Added `SearchSuggestionType`, client methods for `/v1/search/suggestions` and `/v1/search/hashtags`, repository interface/delegation methods, and made submitted `searchProjects` text-only (required `q`, no sort/filter query params). Updated `ProjectSearchQuery` and `project_search_provider` to match the submitted-search contract, then regenerated Dart mapper/provider files.
- Test updates: Updated search client recent payload expectations to `query`, tag-only `hashtag`, and selected-profile payloads. Retained legacy `post`/`project` decode in model tests but stopped generating project browse/filter recents from the search client test.
- Run command: `flutter test test/search/data/search_api_client_test.dart` → passed.
- Nearby command: `flutter test test/search/data/search_api_client_test.dart test/search/data/search_repository_test.dart test/search/data/search_api_client_error_test.dart` → passed.
- Refactor: None.
- Notes: Flutter search client/repository now uses shared Dio paths only, preserves cursors opaquely, and keeps project browse filters out of `/v1/search/projects`.

### Step 28: UT-010 / FR-006, FR-012
- Write failing test: Added `hashtag_result_search_provider_test.dart` for submitted Hashtags-tab pagination, requiring an independent `HashtagResultSearchQuery`/provider, opaque cursor load-more, duplicate suppression by tag, and no-op when exhausted.
- Run command: `flutter test test/search/providers/hashtag_result_search_provider_test.dart`
- Confirmed failure: compilation failed because `hashtag_result_search_provider.dart`, `HashtagResultSearchQuery`, the provider, and fake repository hooks for `searchHashtags`/`searchSuggestions` were missing.
- Implement: Added `HashtagResultSearchQuery`, `HashtagSearchResultsState`, `appendUniqueHashtags`, `hashtag_result_search_provider.dart`, fake repository support for suggestions/hashtags, and regenerated mapper/provider files.
- Test updates: Updated project search provider tests and fake repository signatures for the text-only `searchProjects` contract.
- Run command: `flutter test test/search/providers/hashtag_result_search_provider_test.dart` → passed.
- Nearby command: `flutter test test/search/providers/hashtag_result_search_provider_test.dart test/search/providers/hashtag_search_provider_test.dart test/search/providers/post_search_provider_test.dart test/search/providers/project_search_provider_test.dart test/search/providers/profile_search_provider_test.dart test/search/providers/search_pagination_merge_test.dart` → passed.
- Refactor: None.
- Notes: Submitted Hashtags tab pagination is now independent from exact hashtag post feeds; post/project/profile/exact hashtag pagination remained green.

### Step 29: IT-013 / BR-001, BR-002, FR-006, FR-012, RULE-008
- Write failing test: Added `search_suggestions_provider_test.dart` requiring a UI-agnostic suggestions provider to return empty sections locally for blank input and delegate non-blank trimmed queries, selected types, and per-section limits through `SearchRepository`.
- Run command: `flutter test test/search/providers/search_suggestions_provider_test.dart`
- Confirmed failure: compilation failed because `search_suggestions_provider.dart`, `SearchSuggestionQuery`, and `searchSuggestionsProvider` were missing.
- Implement: Added `SearchSuggestionQuery`, `search_suggestions_provider.dart`, mapper registration, and regenerated Dart mapper/provider files.
- Run command: `flutter test test/search/providers/search_suggestions_provider_test.dart` → passed.
- Write failing test: Added `blank_search_provider_test.dart` requiring blank-search data to fetch AppView-backed recent searches and top hashtags for the default supported full craft tokens from the project option catalog.
- Run command: `flutter test test/search/providers/blank_search_provider_test.dart`
- Confirmed failure: compilation failed because `blank_search_provider.dart`, `BlankSearchData`, and `blankSearchProvider` were missing.
- Implement: Added `BlankSearchData`, `blank_search_provider.dart`, mapper registration, and regenerated Dart mapper/provider files.
- Run command: `flutter test test/search/providers/blank_search_provider_test.dart` → passed.
- Write failing test: Tightened `post_search_provider_test.dart` so submitted post search fakes accept only the relevance-default text-search contract (`q`, `limit`, `cursor`) with no `sort` argument.
- Run command: `flutter test test/search/providers/post_search_provider_test.dart`
- Confirmed failure: compilation failed because `SearchRepository.searchPosts`, `FakeSearchRepository.onSearchPosts`, `PostSearchQuery`, `SearchApiClient.searchPosts`, and `postSearchProvider` still exposed/passed `sort`.
- Implement: Removed submitted-post `sort` from Flutter query, repository, API client, fake repository, and provider; updated nearby client/repository tests to expect text-only submitted post search; regenerated Dart mapper/provider files.
- Run command: `flutter test test/search/providers/post_search_provider_test.dart` → passed.
- Nearby command: `flutter test test/search/providers/blank_search_provider_test.dart test/search/providers/search_suggestions_provider_test.dart test/search/providers/hashtag_result_search_provider_test.dart test/search/providers/hashtag_search_provider_test.dart test/search/providers/post_search_provider_test.dart test/search/providers/project_search_provider_test.dart test/search/providers/profile_search_provider_test.dart test/search/providers/top_hashtags_provider_test.dart test/search/providers/recent_searches_provider_test.dart test/search/providers/search_pagination_merge_test.dart test/search/data/search_api_client_test.dart test/search/data/search_repository_test.dart` → passed.
- Refactor: None beyond narrowing submitted-post search to the approved text/relevance contract.
- Notes: Flutter now has UI-agnostic blank-search and suggestions providers, independent submitted tab pagination providers, and text-only relevance-default submitted Posts/Projects data contracts without rendered UI changes.

### Step 30: IT-014 / BR-005, FR-011, RULE-005, RULE-009
- Write failing test: Added `project_api_client_test.dart` requiring project-owned `ProjectBrowseQuery`/`ProjectBrowseFilters` to send craft types, filter families, sort, limit, and opaque cursor to `/v1/projects`.
- Run command: `flutter test test/projects/data/project_api_client_test.dart`
- Confirmed failure: compilation failed because `project_browse_filters.dart`, `ProjectBrowseQuery`, `ProjectBrowseFilters`, and the `ProjectApiClient.listProjects(query: ...)` contract did not exist.
- Implement: Added `app/lib/projects/models/project_browse_filters.dart`; changed `ProjectApiClient`, `ProjectRepository`, `ApiProjectRepository`, and `ProjectFeed` to consume `ProjectBrowseQuery`; registered/regenerated mappers and Riverpod provider files.
- Run command: `flutter test test/projects/data/project_api_client_test.dart` → passed.
- Provider boundary assertion: Added `FakeProjectRepository` and `project_feed_provider_test.dart` to require project browse pagination to go through `ProjectRepository`, carry filter/sort query state, preserve the opaque cursor, append/de-dupe posts, and not depend on `SearchRepository`/recents.
- Run command: `flutter test test/projects/providers/project_feed_provider_test.dart` → passed.
- Nearby command: `flutter test test/projects/data/project_api_client_test.dart test/projects/providers/project_feed_provider_test.dart test/projects/options/project_option_catalogs_test.dart` → passed.
- Refactor: None.
- Notes: Project browse/filter state now lives under `app/lib/projects/**`; submitted text project search remains under `app/lib/search/**`, and project browse interactions do not serialize Search recents.

### Step 31: REG-007 / BR-005, FR-010, FR-011, RULE-005
- Write failing test: Regression coverage is provided by the Step 30 project-owned API/provider tests plus existing submitted-project search client/provider tests. These assert browse filters live under `/v1/projects`/`app/lib/projects/**` while `/v1/search/projects` remains text-only.
- Run command: `flutter test test/projects/data/project_api_client_test.dart test/projects/providers/project_feed_provider_test.dart test/search/data/search_api_client_test.dart test/search/providers/project_search_provider_test.dart`
- Confirmed failure: No new failure in this loop; the project/search boundary regressions were satisfied by the IT-014 implementation and prior search-project text-only client/provider tests.
- Implement: No additional code changes required.
- Run command: same regression command → passed.
- Refactor: None.
- Notes: Project browse/filtering belongs to the Projects data layer; submitted project text search remains in Search and does not expose rich browse filters or sort.

### Step 32: IT-015 / FR-004
- Write failing test: This compatibility step re-runs the existing Flutter rich-text facet repository/controller tests and AppView facet tests after the shared suggestion changes.
- Run command: `flutter test test/shared/rich_text/facet_suggestion_repository_test.dart test/shared/rich_text/facet_autocomplete_controller_test.dart test/shared/rich_text/mock_account_suggestion_repository_test.dart test/shared/rich_text/mock_hashtag_suggestion_repository_test.dart`
- Confirmed failure: No new failure; existing Flutter facet callers remained compatible.
- AppView command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(Facet|SearchAndFacetProfileSuggestionsShareRankingAndCrafts)' -count=1` → passed.
- Implement: No additional code changes required.
- Run command: Flutter rich-text compatibility command → passed.
- Refactor: None.
- Notes: `/v1/facets/mentions`, `/v1/facets/mentions/resolve`, and `/v1/facets/hashtags` remain compatible for current rich-text autocomplete callers.

### Step 33: REG-001 / BR-003, FR-002, FR-003, FR-004, RULE-003
- Write failing test: This regression step re-runs the broader Flutter rich-text autocomplete suite and AppView facet/ranking tests after the search suggestion changes.
- Run command: `flutter test test/shared/rich_text`
- Confirmed failure: No new failure; the existing autocomplete/editor/repository/facet model suite remained green.
- AppView command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(Facet|RankMentionSuggestionRows|SearchAndFacetProfileSuggestionsShareRankingAndCrafts|FacetHashtagSuggestionsUseHashtagResultRanking)' -count=1` → passed.
- Implement: No additional code changes required.
- Run command: Flutter full rich-text command → passed.
- Refactor: None.
- Notes: Composer/profile rich-text autocomplete compatibility remains protected after shared profile/hashtag suggestion ranking changes.

### Step 34: AT-001 / BR-001, BR-002, FR-012, NFR-001
- Write failing test: Blank-search provider coverage was added in Step 29; existing `search_page_test.dart` remains the compile/stub guard for SearchPage.
- Run command: `flutter test test/search/providers/blank_search_provider_test.dart test/search/search_page_test.dart`
- Confirmed failure: No new failure; the blank-search data/provider contract and existing SearchPage stub tests were already green.
- Implement: No additional code changes required.
- Run command: same acceptance command → passed.
- Refactor: None.
- Notes: Future blank SearchPage logic can fetch AppView-backed recents and craft-grouped top hashtags through UI-agnostic providers; no rendered blank-search UI was added.

### Step 35: REG-005 / BR-001, FR-012, NFR-001
- Write failing test: Existing `search_page_test.dart` remains the automated SearchPage stub/no-expanded-rendering regression for this non-UI slice.
- Run command: `flutter test test/search/search_page_test.dart`
- Confirmed failure: No new failure; SearchPage remains a simple stub with existing hashtag-context compatibility only.
- Implement: No additional code changes required.
- Run command: same stub regression command → passed.
- Refactor: None.
- Notes: No rendered search tabs, recent-management page, project filters UI, cards, or visual navigation behavior were added; source-diff review is recorded separately in MAN-001.

### Step 36: MAN-001 / BR-001, FR-012, NFR-001
- Check: Reviewed page/router diff with `git diff -- app/lib/search/pages app/lib/projects/pages app/lib/router`.
- Result: No diff output; `SearchPage`, `ProjectsPage`, and router/navigation files were not changed by this slice.
- Notes: No rendered Search UI, Projects UI, tabs, management page, cards, layouts, filter controls, or visual navigation behavior were added.

### Step 37: MAN-002 / NFR-002, RULE-001
- Check: Searched `app/lib/search/**` and `app/lib/projects/**` for direct PDS/atproto network reads, `SharedPreferences`/local persistence, and local search-history storage; reviewed `search_repository_provider.dart`, `project_repository_provider.dart`, and `appview/internal/api/search_recent_store.go`.
- Result: Search/project reads flow through `ApiSearchRepository`/`ApiProjectRepository` using shared AppView Dio clients; no local persistent search-history store was added; AppView recents remain persisted only in Postgres `craftsky_recent_searches` and scoped by viewer DID.
- Notes: `app/lib/search/models/profile_search_page.dart` uses typed atproto identifier wrappers for DIDs/handles only; this is not a PDS read path. Existing unrelated AppView PDS write code for blobs/likes/reposts is outside this search-recents slice.

### Step 38: MAN-003 / FR-009, NFR-004
- Check: Reviewed `SearchStore.SearchHashtags`, submitted relevance searches, project browse filters, top-hashtag queries, cursor helpers, and migrations `000019_search_foundation` / `000020_search_refinements`.
- Result: New/refined query paths are bounded by parser-enforced limits and `limit + 1` fetches; submitted post/project relevance uses cursor predicates over relevance/createdAt/URI; project browse chronological/popular paths use cursor predicates and existing materialized project columns; top hashtags are bounded per craft group. Existing indexes cover post text search, project text search, lower craft type, lower pattern difficulty, root chronological ordering, and recent-search listing.
- Notes: No additional performance index migration was added in this slice. Hashtag substring search still unnests `p.tags` and uses `LIKE '%query%'`; it is bounded and covered by focused tests, but production-scale optimization/materialization remains the accepted GAP-002 follow-up if profiling requires it.

### Review Fix: IR-001 / FR-003, FR-004
- Write failing test: Added `TestFacetHashtagSuggestionsUseVisibleSearchHashtagCounts` in `appview/internal/api/search_store_test.go`, seeding visible root posts, duplicate same-post hashtag rows, hidden/takedown rows, old rows, and reply/comment rows. The test asserts `/v1/facets/hashtags` store suggestions use the same visible 28-day distinct top-level counts as `SearchStore.SearchHashtags`.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestFacetHashtagSuggestionsUseVisibleSearchHashtagCounts -count=1`.
- Confirmed failure: `FacetStore.SearchHashtagSuggestions` returned `sockkal=3` and `sockmending=2`, proving hidden/takedown rows were counted; the search hashtag path returned the expected visible counts `sockkal=2` and `sockmending=1`.
- Implement: Updated `FacetStore.SearchHashtagSuggestions` to apply the same top-level/visibility predicates as the search hashtag query path by excluding quote rows and applying `postVisibleModerationPredicate`. Updated the minimal facet test schema to include `moderation_outputs` so the shared predicate is available in facet-only tests.
- Run command: `gofmt -w internal/api/facet_store.go internal/api/search_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestFacetHashtagSuggestionsUseVisibleSearchHashtagCounts -count=1` → passed.
- Nearby command: `gofmt -w internal/api/identity_cache_store_test.go && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run 'Test(FacetHashtagSuggestionsUse|FacetStoreSearchHashtagSuggestions|SearchStore_SearchHashtagsRanksAndPaginates)' -count=1` → passed.
- Refactor: No unrelated refactor; the facet path now reuses the existing moderation predicate used by search result queries.
- Notes: Resolves implementation-review finding `IR-001`; broader requested verification is recorded below.

## Verification Log
- Focused commands:
  - AppView parser/store/route focused commands are recorded in Steps 1–25.
  - Flutter model/client/provider focused commands are recorded in Steps 26–35.
- Broader commands:
  - `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/routes -count=1` → passed.
  - `dart run build_runner build --delete-conflicting-outputs` → passed, wrote 0 outputs on final run.
  - `flutter test test/search test/shared/rich_text test/projects` → passed.
  - `flutter analyze` initially reported directive-ordering/unused-import issues; after fixing imports, `flutter analyze` → passed with no issues.
  - `gofmt -w ...changed AppView Go files...` completed; AppView API/routes tests re-run afterward and passed.
- Implementation-review fix reruns for `IR-001`:
  - `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/routes -count=1` → passed.
  - `flutter test test/search test/shared/rich_text test/projects` → passed.
  - `flutter analyze` → passed with no issues.
- Blocked commands: none.
- Manual checks:
  - MAN-001 no-rendered-UI source diff review → passed.
  - MAN-002 AppView/private-recents architecture review → passed.
  - MAN-003 bounded/index-aware query review → passed with GAP-002 performance caveat retained.

## Coverage Notes And Gaps
- GAP-001 (no rendered UI E2E in this slice) remains accepted by `02-acceptance-tests.md`.
- GAP-002 (production-scale performance cannot be fully proven by focused data) will be addressed by bounded query review and documented follow-up if needed.
- GAP-004 (profile recent display freshness) remains future work.
- GAP-005 (legacy `post`/`project` recent payload migration) will be handled deliberately during recent payload loops.
- Implementation-review finding `IR-001` is resolved by the new DB-backed facet hashtag visibility/count parity regression and by applying the shared moderation/top-level predicates to `FacetStore.SearchHashtagSuggestions`.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing or explicitly documented as blocked/cancelled
- [x] Relevant regression tests passing
- [x] Relevant broader verification passing or documented as blocked
- [x] No unlinked behavior implemented
- [x] No rendered UI/visual navigation behavior added
- [x] `/v1/facets/*` compatibility preserved
- [x] Project browse/filtering remains under Projects API/data layer
- [x] Recents remain explicit private AppView-backed state only
- [x] Docs updated
- [x] Stage completion commit created or no stage changes to commit (this commit records the completed implementation stage)
