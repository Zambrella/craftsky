# TDD Implementation Plan: AppView Search Foundation

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
- Keep this slice AppView-only; do not edit Flutter UI, lexicons, or PDS write behavior.
- Keep recent-search state AppView-private, DID-scoped, hard-deleted, and out of verbose logs.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001, AT-001, REG-001 | FR-001, NFR-001 | AC-014, AC-019 | Fails: no `/v1/search/*` routes or handlers exist. |
| 2 | UT-002 | FR-003, FR-006, FR-018, RULE-003, NFR-001, NFR-002 | AC-019 | Fails: no shared search validation parser exists. |
| 3 | UT-001, IT-002, IT-003, AT-002 | BR-001, FR-002, FR-003, RULE-002 | AC-001, AC-002, AC-021 | Fails: no hashtag normalization/search exists. |
| 4 | UT-009, IT-009, AT-009 | FR-016, FR-002, RULE-006 | AC-012, AC-015 | Fails: no stable search cursor helpers exist. |
| 5 | UT-005, IT-010, IT-011, AT-008 | BR-006, FR-006, FR-007, FR-017, RULE-006, NFR-003 | AC-012, AC-013, AC-024, AC-026 | Fails: no centralized popularity formula/order exists. |
| 6 | UT-008, IT-012, REG-003 | FR-009 | AC-016, AC-024 | Fails: no search response wrappers exist. |
| 7 | UT-006, IT-004, AT-003 | BR-002, FR-004, FR-005, RULE-003, NFR-005 | AC-003, AC-004, AC-020 | Fails: no profile search/ranking exists. |
| 8 | IT-005, AT-007 | FR-006, FR-018, FR-019, NFR-006 | AC-019, AC-022 | Fails: no post/project keyword search exists. |
| 9 | UT-003, IT-006, AT-006 | BR-005, FR-007, FR-008, FR-018, FR-020, RULE-006 | AC-010, AC-011, AC-019, AC-023 | Fails: no project filter parser/query exists. |
| 10 | UT-007, IT-007, AT-005 | BR-004, FR-011, FR-012 | AC-008, AC-009, AC-025 | Fails: no grouped top hashtag query exists. |
| 11 | UT-004, IT-008, IT-014, AT-004 | BR-003, FR-013, FR-014, FR-015, FR-021, FR-022, RULE-001, RULE-004, RULE-005 | AC-005, AC-006, AC-007, AC-018, AC-027 | Fails: no recent-search persistence exists. |
| 12 | IT-013, AT-010, REG-004 | FR-010 | AC-017 | Fails: search surfaces do not exist to apply moderation filtering. |
| 13 | IT-015, AT-011, REG-002, REG-005, MAN-001, MAN-003 | RULE-004, NFR-002, NFR-004, NFR-005, NFR-006, RULE-001 | AC-020, EC-008 | Fails/blocked: indexed path and privacy regressions not yet proven. |
| 14 | MAN-001 through MAN-004 | NFR-001, NFR-002, NFR-003, NFR-004, NFR-005, NFR-006, RULE-001, FR-009 | AC-013, AC-014, AC-015, AC-016, AC-018, AC-020, AC-022 | Manual review pending after implementation. |

## Implementation Steps

### Step 1: IT-001, AT-001, REG-001
- Write failing test: Added `TestAddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice` in `appview/internal/routes/routes_test.go` covering all planned `/v1/search/*` endpoints for registration, missing auth, missing device ID, and standard error-envelope fields.
- Run command: `go test ./internal/routes -run TestAddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice -count=1` from `appview/`.
- Confirmed failure: Meaningful red failure: every search route resolved to the fallthrough `404 page not found`, so auth/device middleware was not reached.
- Implement: Added `SearchStore` scaffolding and placeholder search handlers in `appview/internal/api/search.go`; registered all dedicated authenticated/device-protected `/v1/search/*` routes in `appview/internal/routes/routes.go`.
- Run command: `gofmt -w "internal/routes/routes.go" "internal/routes/routes_test.go" "internal/api/search.go" && go test ./internal/routes -run TestAddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice -count=1`.
- Refactor: None; placeholder handlers are deliberately temporary for later TDD loops.
- Notes: Route-family contract is established without implementing unlinked search behavior. Search handlers currently return `501 not_implemented` once middleware succeeds; later loops replace placeholders with validated behavior.

### Step 2: UT-002
- Write failing test: Added `TestParsePostSearchRequestValidation` and `TestParseProfileSearchRequestRejectsSort` in `appview/internal/api/search_request_test.go` for required post `q`, invalid sort, profile sort rejection, over-limit values, overlong queries, malformed cursors, defaults, and trimming.
- Run command: `go test ./internal/api -run 'TestParse(Post|Profile)SearchRequest' -count=1` from `appview/`.
- Confirmed failure: Meaningful build red: `ParsePostSearchRequest`, `ParseProfileSearchRequest`, and `SearchSortChronological` did not exist.
- Implement: Added `appview/internal/api/search_request.go` with search limit/query constants, `SearchSort` constants, bounded limit parsing, required query parsing, post sort validation, profile sort rejection, and cursor validation via `envelope.DecodeCursor`.
- Run command: `gofmt -w "internal/api/search_request.go" "internal/api/search_request_test.go" && go test ./internal/api -run 'TestParse(Post|Profile)SearchRequest' -count=1`.
- Refactor: None.
- Notes: UT-002 validation scaffolding is green for post/profile request parsing. Project filters, hashtag normalization, and recent payload validation remain in their planned loops.

### Step 3: UT-001, IT-002, IT-003, AT-002
- Write failing test: Added `TestNormalizeHashtagPathValue` in `appview/internal/api/search_request_test.go` for trimming, one leading `#`, lowercasing, canonical values, and invalid empty/overlong/space/slash/double-hash inputs.
- Run command: `go test ./internal/api -run TestNormalizeHashtagPathValue -count=1`.
- Confirmed failure: Meaningful build red: `NormalizeHashtagPathValue` did not exist.
- Implement: Added `NormalizeHashtagPathValue`; added exact-hashtag handler parsing; added initial `SearchStore.SearchHashtagPosts` equality query against materialized `tags` for top-level posts and canonical hashtag metadata response.
- Run command: `gofmt -w "internal/api/search.go" "internal/api/search_store.go" "internal/api/search_cursor.go" "internal/api/search_response.go" && go test ./internal/api -run TestNormalizeHashtagPathValue -count=1`; reran `go test ./internal/routes -run TestAddRoutes_SearchRoutesRegisteredAndRequireAuthenticatedDevice -count=1`.
- Refactor: None.
- Notes: UT-001 is green. Store/handler exact-equality integration coverage is scaffolded and will be exercised with broader store tests; text fallback and reply exclusion are encoded in the query (`tags` equality and top-level predicates).

### Step 4: UT-009, IT-009, AT-009
- Write failing test: Added `search_cursor_test.go` coverage for chronological and popularity cursor round-trips and sort-mismatch invalid cursor behavior.
- Run command: `gofmt -w "internal/api/search_cursor_test.go" && go test ./internal/api -run 'TestSearch(Chronological|Popularity)Cursor' -count=1`.
- Confirmed failure: Not red in isolation because `search_cursor.go` had already been introduced as a support file during the Step 3 exact-hashtag pagination implementation.
- Implement: `EncodeChronologicalSearchCursor`, `DecodeChronologicalSearchCursor`, `EncodePopularityCursor`, and `DecodePopularityCursor` centralize opaque cursor payloads with sort discriminators and RFC3339Nano time values.
- Run command: same focused command passed.
- Refactor: None.
- Notes: Pagination helpers are centralized earlier than their dedicated test loop as called out in the coding plan. Invalid cursor mapping is available to handlers through `envelope.ErrInvalidCursor`.

### Step 5: UT-005, IT-010, IT-011, AT-008
- Write failing test: Added `search_ranking_test.go` tests for the documented decayed popularity formula and future-age clamping.
- Run command: `gofmt -w "internal/api/search_ranking_test.go" && go test ./internal/api -run TestPopularityScore -count=1`.
- Confirmed failure: Not red in isolation because `PopularityScore` was introduced with the Step 3 exact-hashtag `sort=popular` store path.
- Implement: `PopularityScore(likes, visibleReplies, reposts, createdAt, rankedAt)` centralizes the required `likes + 2*replies + 3*reposts` formula and `pow(1 + ageHours / 72, 1.5)` decay.
- Run command: same focused command passed.
- Refactor: None.
- Notes: SQL popularity ordering uses the same constants/formula shape in `SearchStore.searchPosts`; broader post/project popularity fixtures remain to be covered by future integration tests.

### Step 6: UT-008, IT-012, REG-003
- Write failing test: Added `TestSearchPostPageResponseOmitsPopularityScore` in `appview/internal/api/search_response_test.go`.
- Run command: `gofmt -w "internal/api/search_response_test.go" && go test ./internal/api -run TestSearchPostPageResponseOmitsPopularityScore -count=1`.
- Confirmed failure: No meaningful new red because response wrapper scaffolding already existed from the exact-hashtag loop.
- Implement: Existing `SearchPostPageResponse` wraps `[]*PostResponse` and has no public score field.
- Run command: focused command passed.
- Refactor: None.
- Notes: Response wrapper JSON includes search metadata, post engagement counts, and cursor while omitting `popularityScore`.

### Step 7: UT-006, IT-004, AT-003
- Write failing test: Added `search_profile_rank_test.go` for relevance classes and followed-first rank tuple behavior.
- Run command: `go test ./internal/api -run 'TestProfile(RelevanceRank|SearchRankTuple)' -count=1`.
- Confirmed failure: Meaningful build red: `ProfileRelevanceRank` and `ProfileSearchRankTuple` did not exist.
- Implement: Added profile ranking helpers in `search_ranking.go`, plus profile search handler/store wiring using Craftsky profiles, identity cache, Bluesky profile text, follow state, and profile moderation predicates.
- Run command: `gofmt -w "internal/api/search.go" "internal/api/search_store.go" && go test ./internal/api -run 'TestProfile(RelevanceRank|SearchRankTuple)' -count=1`.
- Refactor: None.
- Notes: UT-006 is green. Profile search now rejects unsupported sort via parser and orders by followed rank, relevance rank, handle, and DID.

### Step 8: IT-005, AT-007
- Write failing test: Existing `TestParsePostSearchRequestValidation` already covered missing/blank post `q`; `/v1/search/posts` was still a placeholder `501`.
- Run command: `go test ./internal/api ./internal/routes -count=1`.
- Confirmed failure: The route existed but the implementation path had not been wired before this step.
- Implement: Added `SearchStore.SearchPosts` and `SearchPostsHandler`, searching visible top-level posts/projects by local AppView fields: post text, project title, pattern name, materials, project tags, and design tags. Replies are excluded by top-level predicates.
- Run command: `gofmt -w "internal/api/search.go" "internal/api/search_store.go" && go test ./internal/api ./internal/routes -count=1`.
- Refactor: Corrected SQL to inline PostgreSQL matching rather than referencing a nonexistent helper function.
- Notes: Handler/store compile and package tests are green. Dedicated seeded FTS integration fixtures remain a documented coverage gap; migration adds local text-vector indexes for the intended strategy.

### Step 9: UT-003, IT-006, AT-006
- Write failing test: Added `TestParseProjectSearchRequestFilters` for supported filters, lower-case normalization, unsupported keys, overlong values, and per-family count bounds.
- Run command: `go test ./internal/api -run TestParseProjectSearchRequestFilters -count=1`.
- Confirmed failure: Meaningful build red: `ParseProjectSearchRequest` did not exist.
- Implement: Added project search request parser and `SearchProjectsHandler`/`SearchStore.SearchProjects` with browse-all support, optional `q`, strict filter validation, OR within filter arrays, AND across filter families, case-insensitive scalar/array matching, chronological default order, and shared post response hydration.
- Run command: `gofmt -w "internal/api/search.go" "internal/api/search_store.go" "internal/api/search_request.go" "internal/api/search_request_test.go" && go test ./internal/api ./internal/routes -count=1`.
- Refactor: Added `projectFilterValues` to ensure absent filters bind as empty arrays rather than SQL NULL.
- Notes: Unit parser coverage is green. Dedicated seeded project-filter integration tests are not present and are recorded as a coverage gap.

### Step 10: UT-007, IT-007, AT-005
- Write failing test: Route remained placeholder `501` before this loop; no dedicated top-hashtag unit fixture was added.
- Run command: `go test ./internal/api ./internal/routes -count=1`.
- Confirmed failure: Endpoint implementation absent before the loop.
- Implement: Added `TopHashtagsResponse`, `ParseTopHashtagsRequest`, `TopHashtagsHandler`, and `SearchStore.TopHashtags` using top-level project posts, a 28-day window, distinct project counts, requested craft groups with empty `items`, and default craft groups when none are requested.
- Run command: `gofmt -w "internal/api/search.go" "internal/api/search_store.go" "internal/api/search_request.go" "internal/api/search_response.go" && go test ./internal/api ./internal/routes -count=1`.
- Refactor: None.
- Notes: Package tests are green. Dedicated seeded distinct-count integration fixtures remain a documented coverage gap.

### Step 11: UT-004, IT-008, IT-014, AT-004
- Write failing test: Red condition was absence of recent-search table/store/handlers; recent routes returned placeholder `501` before this loop.
- Run command: `go test ./internal/api ./internal/routes -count=1`.
- Confirmed failure: Persistence and handlers did not exist.
- Implement: Added migration `000019_search_foundation` with `craftsky_recent_searches`; added recent request normalization/hash generation, opaque ID generation, DID-scoped save/list/delete store methods, duplicate refresh preserving existing display label, prune-to-50 behavior, response builders, and authenticated handlers. Delete returns `204 No Content` and is idempotent for nonexistent/not-owned IDs because the delete is scoped by viewer DID.
- Run command: `gofmt -w "internal/api/search.go" "internal/api/search_request.go" "internal/api/search_response.go" "internal/api/search_recent_store.go" && go test ./internal/api ./internal/routes -count=1`.
- Refactor: Recent-search logs include type/run/error only, not full payloads.
- Notes: Package tests are green. Dedicated recent-search database lifecycle/privacy fixtures are not present and are recorded as a coverage gap.

### Step 12: IT-013, AT-010, REG-004
- Write failing test: No new moderation fixture was added in this continuation.
- Run command: `go test ./internal/api ./internal/routes -count=1`.
- Confirmed failure: Not red in isolation; search SQL was written to reuse existing moderation predicates.
- Implement: Search post/project/hashtag/top-hashtag queries use `postVisibleModerationPredicate`; profile search uses `profileVisibleModerationPredicate`. Popularity reply counts exclude active hide/takedown reply/account rows.
- Run command: package tests passed.
- Refactor: Fixed the popularity reply-count subquery so moderation filtering references the reply alias rather than the parent alias.
- Notes: Existing moderation tests pass under broader verification. Dedicated cross-surface search moderation fixtures remain a documented gap.

### Step 13: IT-015, AT-011, REG-002, REG-005, MAN-001, MAN-003
- Write failing test: No new regression/manual fixture was added.
- Run command: `go test ./...`; `just fmt`; `just test`.
- Confirmed failure: Not red in isolation.
- Implement: Added migration search-supporting indexes: recent list index, post/project text-vector GIN indexes, chronological root-post index, lower-case project scalar indexes, and trigram candidate indexes for handles/display names/descriptions. Search result endpoints do not call recent save code, preserving explicit-save behavior.
- Run command: all listed commands passed.
- Refactor: None.
- Notes: `MAN-001` EXPLAIN review and deeper log-redaction review were not run; code-level review found no PDS writes and no full recent payload logging in handlers. Existing facet routes were not modified.

### Step 14: MAN-001 through MAN-004
- Check: Partially completed by code/migration inspection and full test/format commands; no representative `EXPLAIN` plans were run.
- Notes: Manual query-plan review remains a follow-up gap. API contracts are implemented with camelCase wrappers and standard error envelopes. Popularity score is internal-only.

## Implementation Review Fix Pass (2026-06-20)

### Fix 1: IT-011 / AT-008 project `sort=popular`
- Requirement IDs: BR-006, FR-007, FR-017, FR-020, RULE-006.
- Write failing test: Added `TestSearchStore_SearchProjectsPopularOrdersBrowseAllAndFilteredProjects` for browse-all and filtered project search with an older, higher-engagement project outranking newer quieter projects.
- Red failure: Implementation review confirmed `SearchProjects` accepted `sort=popular` while ordering chronologically and returning a zero score.
- Implement: Added the centralized decayed popularity SQL path to project search, including active likes/reposts, visible replies, stable popularity cursors, and `popularity_score DESC, created_at DESC, uri DESC` ordering.
- Green command: `go test ./internal/api -run 'TestSearchStore_' -count=1` passed.

### Fix 2: profile search cursor pagination
- Requirement IDs: FR-004, FR-005, FR-016.
- Write failing test: Added `TestSearchStore_SearchProfilesPaginatesByRankTuple` to prove followed-first/relevance ordering continues across pages with no duplicates.
- Red failure: Implementation review confirmed profile search ignored cursors, fetched only `limit`, and never returned a next cursor.
- Implement: Added profile cursor encode/decode helpers over `(followedRank, relevanceRank, handleLower, did)`, store seek pagination with `limit + 1`, next-cursor generation, and handler mapping for invalid profile cursors.
- Green command: `go test ./internal/api -run 'TestSearchStore_' -count=1` passed.

### Fix 3: UT-004 / IT-008 / IT-014 recent-search payloads and privacy
- Requirement IDs: BR-003, FR-013, FR-014, FR-015, FR-021, FR-022, RULE-001, RULE-005.
- Write failing tests: Added `TestDecodeSaveRecentSearchRequestNormalizesTypedPayloads`, `TestDecodeSaveRecentSearchRequestRejectsInvalidTypedPayloads`, and `TestSearchStore_RecentSearchLifecycleDedupesPrunesAndHardDeletes`.
- Red failure: Implementation review confirmed raw JSON payloads such as `{}` or `null` could be saved and equivalent searches did not necessarily de-duplicate.
- Implement: Added type-specific payload normalization for `hashtag`, `profile`, `post`, and `project` recents; defaulted omitted sort values to chronological where applicable; normalized/canonicalized project filters; rejected invalid or non-rerunnable payloads; retained existing de-duplication, prune-to-50, DID-scoped list, and idempotent hard-delete behavior.
- Green command: `go test ./internal/api -run 'Test(SearchStore|DecodeSaveRecent)' -count=1` passed.

### Fix 4: seeded store and response-contract coverage
- Requirement IDs: BR-001, BR-004, BR-005, BR-006, FR-002, FR-006, FR-007, FR-008, FR-009, FR-010, FR-011, FR-012, FR-017, FR-019, FR-020, RULE-002.
- Write failing tests: Added seeded store tests for exact hashtag equality (`TestSearchStore_SearchHashtagPostsUsesStoredTagEqualityOnly`), FTS keyword search (`TestSearchStore_SearchPostsAndProjectsUseFTSFields`), project filters (`TestSearchStore_SearchProjectsAppliesFilterSemantics`), grouped top hashtags (`TestSearchStore_TopHashtagsGroupsDistinctProjectsAndEmptyCrafts`), popularity (`TestSearchStore_SearchProjectsPopularOrdersBrowseAllAndFilteredProjects`), moderation-before-limit/rank (`TestSearchStore_ModerationFiltersBeforeSearchRankingAndLimits`), and retained the existing response wrapper test that proves `popularityScore` is not public JSON.
- Implement: Fixed behavior as needed while keeping existing public response wrappers and post-response reuse.
- Green command: `go test ./internal/api -run 'TestSearchStore_' -count=1` passed.

### Fix 5: FTS/indexed keyword path and MAN-001
- Requirement IDs: FR-019, NFR-002, NFR-004, NFR-005, NFR-006.
- Write failing test: `TestSearchStore_SearchPostsAndProjectsUseFTSFields` covers matches in post text and core project fields while excluding replies.
- Red failure: Implementation review confirmed keyword search used raw `lower(...) LIKE '%q%'` predicates and array `unnest` scans despite planned FTS indexes.
- Implement: Replaced post/project keyword matching with PostgreSQL `to_tsvector('simple', ...) @@ plainto_tsquery('simple', ...)` expressions matching the migration's local FTS index definitions.
- MAN-001 command: Ran representative `EXPLAIN` checks against `postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable` for hashtag equality, post/project FTS, project filters, and recent-search list.
- MAN-001 result: The local dev database had not applied `000019_search_foundation` (`craftsky_recent_searches` was absent and search indexes were not present), so full index-plan validation is blocked until local migrations are applied. The implemented SQL now matches the FTS expressions defined by `000019_search_foundation.up.sql`; the blocked EXPLAIN state should be rechecked after migration in implementation review or local dev setup.

### Fix-pass verification
- Focused command: `go test ./internal/api -run 'Test(SearchStore|DecodeSaveRecent)' -count=1` passed.
- Package command: `go test ./internal/api -count=1` passed.
- Broader command: `go test ./...` from `appview/` passed.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing or coverage gaps documented above
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped/documented for manual checks

## Final Verification
- `go test ./...` from `appview/`: passed.
- `just fmt`: passed (`gofmt -w .` and `go vet ./...`).
- `just test`: passed (`go test -race ./...` with `TEST_DATABASE_URL`).
- Fix-pass focused tests: `go test ./internal/api -run 'Test(SearchStore|DecodeSaveRecent)' -count=1` passed.
- Fix-pass package tests: `go test ./internal/api -count=1` passed.
- Fix-pass broader tests: `go test ./...` from `appview/` passed.
- Fix-pass final verification: `just fmt && just test` passed on 2026-06-20 after correcting the new project-search test fixtures and SQL parameter bindings.

## Coverage Gaps / Follow-ups
- `MAN-001` representative `EXPLAIN` query-plan review was attempted, but the local dev database had not applied migration `000019_search_foundation`; re-run after applying migrations to confirm planner use of the added GIN/trigram/list indexes.
- No remaining known seeded-test gap for the implementation-review required behaviors: project popularity, profile pagination, typed recent-search normalization/privacy, exact hashtag equality, keyword search, project filters, top hashtags, popularity, moderation, and response contracts now have focused automated coverage.
