# TDD Implementation Plan: Profile Social Summary

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
- Preserve AppView-read / Flutter-render architecture; Flutter must not query the PDS for profile/social summary data.
- Keep `followerCount` and `followingCount` in the API contract while removing visible profile/settings-entry display.

## Test Order
Mirrors `04-coding-plan.md` §9.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 | FR-005, FR-006, RULE-003, RULE-005 | AC-006, AC-007, AC-020 | Fails: `ProfileRow` lacks post summary fields and store does not count root posts/project count. |
| 1 | UT-002 | FR-006, RULE-005 | AC-007 | Fails: total post count is not exposed/filterable. |
| 1 | UT-003 | FR-005, RULE-003 | AC-006 | Fails: recent activity root-post filter is not implemented. |
| 2 | IT-002 | BR-002, FR-003, RULE-001 | AC-003 | Fails: no mutual follower count query/field. |
| 2 | UT-001 | RULE-001, FR-003 | AC-003 | Fails: no mutual predicate helper/query behavior. |
| 3 | IT-003 | FR-012, FR-015 | AC-016, AC-019 | Fails: JSON lacks new mutual count or response contract tests. |
| 3 | UT-005 | FR-012, FR-015 | AC-016, AC-019 | Fails: response DTO lacks scalar mutual count/no-preview assertion. |
| 3 | REG-008 | FR-012 | AC-016 | Fails if old count fields are removed or not serialized. |
| 4 | IT-004 | FR-010, FR-011, FR-016, NFR-001 | AC-011, AC-012, AC-015, AC-019 | Fails: no mutual followers list method/endpoint. |
| 5 | IT-005 | BR-003, FR-008, FR-010, FR-011, RULE-004 | AC-009, AC-011, AC-012 | Fails: no followers list endpoint/order/cursor support. |
| 5 | IT-006 | BR-003, FR-009, FR-010, FR-011, RULE-004 | AC-010, AC-011, AC-012 | Fails: no following list endpoint/order/cursor support. |
| 6 | IT-007 | NFR-001 | AC-013 | Fails: new graph endpoints not registered/protected. |
| 6 | REG-007 | NFR-001 | AC-013 | Fails if route auth/device conventions regress. |
| 7 | IT-008 | NFR-002 | AC-014 | Fails: default/max bounded list behavior absent. |
| 8 | UT-011 | FR-010, FR-015 | AC-011, AC-019 | Fails: Flutter models do not decode new fields/list rows. |
| 9 | UT-012 | FR-011, FR-016, NFR-001 | AC-012, AC-015 | Fails: Flutter API client lacks list endpoints and cursor query support. |
| 10 | UT-004 | FR-004, RULE-002 | AC-005 | Fails: account-age formatter absent. |
| 10 | UT-006 | FR-001, FR-004, FR-006, FR-017 | AC-001, AC-005, AC-007, AC-018, AC-020 | Fails: stats widget renders old follower/following/project behavior. |
| 10 | AT-001 | BR-001, FR-001, FR-012 | AC-001, AC-016 | Fails: profile page shows follower/following stats. |
| 10 | AT-004 | FR-004, FR-005, FR-006, RULE-002, RULE-003, RULE-005 | AC-005, AC-006, AC-007, AC-020 | Fails: profile page lacks joined/recent/total/data-driven project stats. |
| 10 | AT-009 | FR-017, FR-004 | AC-018 | Fails: non-Craftsky age-hiding not covered/implemented. |
| 11 | UT-007 | FR-002, FR-013 | AC-002, AC-015 | Fails: mutual link widget absent. |
| 11 | AT-002 | BR-002, FR-002 | AC-002, AC-004 | Fails: visitor mutual count/self absence behavior absent. |
| 11 | AT-003 | BR-002, FR-013, FR-016 | AC-015, AC-019 | Fails: mutual bottom sheet/list load absent. |
| 12 | UT-009 | FR-007, FR-012 | AC-008 | Fails: settings entries absent. |
| 12 | AT-005 | BR-003, FR-007, FR-012 | AC-008 | Fails: settings links absent or not tested for no counts. |
| 12 | REG-005 | FR-007 | AC-008 | Fails if existing settings tiles regress. |
| 13 | UT-008 | FR-014 | AC-017 | Fails: empty state copy helper/widget absent. |
| 13 | UT-010 | FR-008, FR-009, RULE-004 | AC-009, AC-010, AC-017 | Fails: follow list pages/providers absent. |
| 13 | AT-006 | BR-003, FR-008, FR-010, RULE-004 | AC-009, AC-011 | Fails: followers page absent. |
| 13 | AT-007 | BR-003, FR-009, FR-010, RULE-004 | AC-010, AC-011 | Fails: following page absent. |
| 13 | AT-008 | FR-014 | AC-017 | Fails: empty graph states/zero mutuals not covered. |
| 14 | REG-001 | FR-001, FR-004, FR-006 | AC-001, AC-005, AC-007 | Existing profile identity/bio/crafts/avatar/banner must remain green. |
| 14 | REG-002 | NG-005 | N/A | Follow/unfollow UI behavior must remain green. |
| 14 | REG-003 | FR-017 | AC-018 | Non-Craftsky marker must remain green. |
| 14 | REG-004 | FR-005, RULE-005 | AC-006, AC-007 | Profile posts/comments tabs must remain green. |
| 14 | REG-006 | FR-003, NG-005 | AC-003 | Follow write endpoints and `viewerIsFollowing` must remain green. |

## Implementation Steps

### Step 1: IT-001 / UT-002 / UT-003
- Write failing test: Added `TestProfileStore_ReadByDID_ProfileSummaryCountsRootPosts` in `appview/internal/api/profile_store_test.go`, seeding root posts inside/outside the trailing 7-day window, a reply row, and an explicit zero project-count expectation.
- Run command: `go test ./internal/api -run TestProfileStore_ReadByDID_ProfileSummaryCountsRootPosts -count=1` from `appview/`.
- Confirmed failure: Build failed because `api.ProfileRow` had no `PostCount`, `PostsLast7Days`, or `ProjectCount` fields.
- Implement: Added summary fields to `ProfileRow`; extended `ProfileStore.Read` to count top-level authored `craftsky_posts` rows, count recent top-level rows with `created_at >= now() - interval '7 days'`, and set explicit data-driven `ProjectCount` to `0`.
- Run command: `go test ./internal/api -run TestProfileStore_ReadByDID_ProfileSummaryCountsRootPosts -count=1` passed; nearby `go test ./internal/api -run 'TestProfileStore_ReadByDID' -count=1` passed.
- Refactor: None.
- Notes: This covers IT-001 plus UT-002/UT-003 semantics in one AppView store-level test; quote/root posts remain top-level because only reply fields define comments/replies in the current `craftsky_posts` model.

### Step 2: IT-002 / UT-001
- Write failing test: Added `TestProfileStore_ReadByDID_MutualFollowerCountUsesViewerGraph`, seeding `viewer -> mutual`, `mutual -> profile`, and two non-qualifying one-sided follow relationships.
- Run command: `go test ./internal/api -run TestProfileStore_ReadByDID_MutualFollowerCountUsesViewerGraph -count=1` from `appview/`.
- Confirmed failure: Build failed because `api.ProfileRow` had no `MutualFollowerCount` field.
- Implement: Added `MutualFollowerCount` to `ProfileRow`; extended `ProfileStore.Read` with a viewer-scoped follow-graph count for visitor Craftsky profiles, omitted for self profiles.
- Run command: Focused test passed; nearby `go test ./internal/api -run 'TestProfileStore_ReadByDID' -count=1` passed.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: This covers IT-002 plus UT-001 predicate semantics at store level. Mutual count uses AppView indexed `atproto_follows` only.

### Step 3: IT-003 / UT-005 / REG-008
- Write failing test: Added `TestBuildProfileResponse_IncludesSummaryCountsWithoutMutualPreview` in `appview/internal/api/profile_response_test.go` asserting old counts remain, new scalar counts serialize, and no mutual preview array is emitted.
- Run command: `go test ./internal/api -run TestBuildProfileResponse_IncludesSummaryCountsWithoutMutualPreview -count=1` from `appview/`.
- Confirmed failure: Build failed because `api.ProfileResponse` had no `MutualFollowerCount`, `PostCount`, `PostsLast7Days`, or `ProjectCount` fields.
- Implement: Added those fields to `ProfileResponse` with camelCase JSON tags and populated them from `ProfileRow` in `BuildProfileResponse`.
- Run command: Focused test passed; nearby `go test ./internal/api -run 'TestBuildProfileResponse|TestGetProfile' -count=1` passed.
- Refactor: Ran `gofmt` on touched response files.
- Notes: Covers IT-003, UT-005, and REG-008 response-contract preservation for existing follower/following fields.

### Step 4: IT-004
- Write failing test: Added `TestProfileStore_ListMutualFollowers_PaginatesDisplayRows` for mutual list ordering, display fields, total count, and opaque cursor paging; added `TestGetMutualFollowers_ReturnsPaginatedAccountRows` for handler JSON shape and query parameters.
- Run command: `go test ./internal/api -run TestProfileStore_ListMutualFollowers_PaginatesDisplayRows -count=1` and `go test ./internal/api -run TestGetMutualFollowers_ReturnsPaginatedAccountRows -count=1` from `appview/`.
- Confirmed failure: First failed because `ProfileStore.ListMutualFollowers` was undefined; second failed because `GetMutualFollowersHandler` and `ProfileAccountPage` were undefined.
- Implement: Added `ProfileAccountRow`, store mutual list query with keyset cursor, `ProfileAccountSummary`/`ProfileAccountPage`, mutual handler, account-row handle resolution, and cursor/error handling.
- Run command: `go test ./internal/api -run 'TestProfileStore_ListMutualFollowers_PaginatesDisplayRows|TestGetMutualFollowers_ReturnsPaginatedAccountRows' -count=1` passed.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: Route registration/auth protection is deferred to Step 6 as planned; this step covers store and handler contract for mutual list data.

### Step 5: IT-005 / IT-006
- Write failing test: Added `TestProfileStore_ListFollowersAndFollowing_OrderNewestFirst` for newest-first self follower/following lists and `TestGetMeFollowersAndFollowing_ReturnPaginatedAccountRows` for self graph handlers.
- Run command: Focused store and handler test commands from `appview/`.
- Confirmed failure: Store test failed because `ListFollowers` / `ListFollowing` did not exist; handler test failed because `GetMeFollowersHandler` / `GetMeFollowingHandler` did not exist.
- Implement: Added follower/following store methods with keyset cursor and total count; added self graph handlers that read signed-in DID, parse `limit`/`cursor`, resolve account handles, and return `ProfileAccountPage`.
- Run command: `go test ./internal/api -run 'TestProfileStore_ListFollowersAndFollowing_OrderNewestFirst|TestGetMeFollowersAndFollowing_ReturnPaginatedAccountRows' -count=1` passed.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: Covers IT-005 and IT-006 at store and handler level; route auth/device registration remains Step 6.

### Step 6: IT-007 / REG-007
- Write failing test: Added `TestRoutes_ProfileSocialGraphEndpointsRequireAuthenticatedDevice` covering mutual followers, followers, and following routes for unauthenticated and missing-device requests.
- Run command: `go test ./internal/routes -run TestRoutes_ProfileSocialGraphEndpointsRequireAuthenticatedDevice -count=1` from `appview/`.
- Confirmed failure: All new endpoint requests returned 404 because routes were not registered.
- Implement: Registered `GET /v1/profiles/@{handleOrDid}/mutual-followers`, `GET /v1/profiles/me/followers`, and `GET /v1/profiles/me/following` behind the same `authN(deviceID(...))` stack as existing profile routes.
- Run command: Focused route auth test passed.
- Refactor: Ran `gofmt` on route files.
- Notes: Covers IT-007 and REG-007 for route-level auth/device conventions.

### Step 7: IT-008
- Write failing test: No separate red test added; bounded list behavior was implemented while adding the new graph handlers by reusing existing `parseLimit` default/max cap and store keyset queries.
- Run command: Covered by focused graph handler/store tests in Steps 4-6.
- Confirmed failure: Not applicable; this is a `Should` performance/boundedness test and the required bound was inherited from existing API pagination helper during prior red/green loops.
- Implement: Handlers cap `limit` using existing default 50/max 100 semantics; store queries require caller-provided `LIMIT` and use keyset cursor predicates.
- Run command: `go test ./internal/api -run 'TestProfileStore_ListMutualFollowers_PaginatesDisplayRows|TestProfileStore_ListFollowersAndFollowing_OrderNewestFirst|TestGetMutualFollowers_ReturnsPaginatedAccountRows|TestGetMeFollowersAndFollowing_ReturnPaginatedAccountRows' -count=1` passed as part of graph work.
- Refactor: None.
- Notes: No index migration added because adding migrations is high-risk and current tests verify bounded behavior rather than planner use. Migration/index review remains a performance follow-up if large-list manual checks show issues.

### Step 8: UT-011
- Write failing test: Extended `app/test/profile/models/profile_test.dart` to decode `mutualFollowerCount`, `postCount`, `postsLast7Days`, `projectCount`, and new `ProfileAccountPage` / `ProfileAccountSummary` list rows.
- Run command: `flutter test test/profile/models/profile_test.dart` from `app/`.
- Confirmed failure: Compilation failed because new profile fields and account-page model files/mappers did not exist.
- Implement: Added fields to `Profile`; created `ProfileAccountSummary` and `ProfileAccountPage` models; registered mappers in bootstrap; ran build runner to generate mapper artifacts.
- Run command: Focused model test passed.
- Refactor: None.
- Notes: Covers UT-011 model decoding for profile scalar fields and graph list account rows.

### Step 9: UT-012
- Write failing test: Added `ProfileApiClient` tests for mutual followers, self followers, and self following endpoints with `limit`/opaque `cursor` query parameters and page decoding.
- Run command: `flutter test test/profile/data/profile_api_client_test.dart` from `app/`.
- Confirmed failure: Compilation failed because `listMutualFollowers`, `listFollowersMe`, and `listFollowingMe` did not exist.
- Implement: Added client methods and repository interface/production/fake implementations for the three graph list endpoints.
- Run command: Focused API client test passed.
- Refactor: Ran `dart format` on touched profile data/model/test files.
- Notes: Covers UT-012 and locks concrete route paths from the coding plan.

### Step 10: UT-004 / UT-006 / AT-001 / AT-004 / AT-009
- Write failing test: Added `profile_stats_test.dart` for `formatJoinedAge`, new profile stat rendering, follower/following hiding, data-driven project count, and non-Craftsky age hiding; updated `profile_page_test.dart` expectations to assert profile-page stats and no old count labels.
- Run command: `flutter test test/profile/widgets/profile_stats_test.dart` and then `flutter test test/profile/profile_page_test.dart test/profile/widgets/profile_stats_test.dart` from `app/`.
- Confirmed failure: Formatter was missing; widget tests then failed because `ProfileStats(profile:)` did not exist and old follower/following stat cells were still rendered. Initial exact-copy test was corrected to match `timeago`'s `about a year ago` wording per coding plan.
- Implement: Added `timeago`, `formatJoinedAge`, rewrote `ProfileStats` to render joined/recent/total/projects stats only, updated `ProfileMetaSection` to pass the full profile, and removed the hardcoded project count.
- Run command: Focused profile stats and profile page tests passed.
- Refactor: Ran `dart format` on touched Flutter profile files.
- Notes: Covers UT-004, UT-006, AT-001, AT-004, and AT-009. Follower/following API fields remain decoded but are not rendered in profile stats.

### Step 11: UT-007 / AT-002 / AT-003
- Write failing test: Extended `profile_page_test.dart` to expect visitor mutual text and to tap it, opening a bottom sheet that renders mutual account rows from the repository.
- Run command: `flutter test test/profile/profile_page_test.dart` from `app/`.
- Confirmed failure: Mutual text was absent and tapping `12 mutual followers` failed because no widget existed.
- Implement: Added `ProfileMutualFollowersLink`, `ProfileMutualFollowersSheet`, visitor-only composition in `ProfileMetaSection`, and passed `isOwnProfile` from `ProfilePage`.
- Run command: Focused profile page tests passed.
- Refactor: Ran `dart format` on touched profile UI files.
- Notes: Covers UT-007, AT-002, and AT-003. The bottom sheet uses `FractionallySizedBox(heightFactor: 0.9)` and fetches the separate paginated mutual endpoint through the repository.

### Step 12: UT-009 / AT-005 / REG-005
- Write failing test: Extended `settings_page_test.dart` to expect tappable Followers/Following entries with no numeric counts while preserving existing settings tiles.
- Run command: `flutter test test/settings/settings_page_test.dart` from `app/`.
- Confirmed failure: Settings page had no Followers or Following entries.
- Implement: Added Followers and Following `ListTile`s without counts; existing Clear Image Cache and Sign Out tiles remain.
- Run command: Focused settings page test passed.
- Refactor: Ran `dart format` on touched settings files.
- Notes: Covers UT-009, AT-005, and REG-005.

### Step 13: UT-008 / UT-010 / AT-006 / AT-007 / AT-008
- Write failing test: Added `follow_list_page_test.dart` for Followers title count, row ordering preservation, Following title count, and empty following copy.
- Run command: `flutter test test/settings/follow_list_page_test.dart` from `app/`.
- Confirmed failure: `FollowListPage` / `FollowListKind` did not exist.
- Implement: Added shared `FollowListPage` for followers/following, loading first pages through the profile repository, app-bar counts, ordered rows, and empty copy. Wired settings tiles to push these pages.
- Run command: `flutter test test/settings/settings_page_test.dart test/settings/follow_list_page_test.dart` passed.
- Refactor: Ran `dart format` on touched settings test/page files.
- Notes: Covers UT-008, UT-010, AT-006, AT-007, and the follower/following empty-state portions of AT-008. Zero-mutuals absence is covered by `ProfileMutualFollowersLink` returning shrink for count 0 and profile tests where no mutual count is rendered.

### Step 14: REG-001 / REG-002 / REG-003 / REG-004 / REG-006
- Write failing test: No new regression-only test added; existing profile, follow, post-tab, AppView API, and route suites were run after feature loops.
- Run command: `go test ./internal/api ./internal/routes` from `appview/`; `flutter test test/profile/profile_page_test.dart test/profile/widgets/profile_posts_tab_test.dart test/profile/widgets/profile_comments_tab_test.dart test/settings/settings_page_test.dart test/settings/follow_list_page_test.dart` from `app/`.
- Confirmed failure: Not applicable; these were regression verification commands after green feature loops.
- Implement: No regression-specific code changes required.
- Run command: Both regression commands passed.
- Refactor: None.
- Notes: Covers REG-001 through REG-006 within the available focused regression suites.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped

## Execution Notes
- Created initial plan from approved requirements/test/coding-plan documents on 2026-05-27.
- Final verification on 2026-05-27:
  - `go test ./...` from `appview/` passed.
  - `flutter test` from `app/` passed.
  - Dart analyzer via MCP on `app/lib`, `app/test/profile`, and `app/test/settings` reported no errors.
- Coverage notes:
  - Review fix RF-3 added an approved index-only migration for ordered follow and root-post query shapes.
  - Settings follower/following pages are implemented with local `MaterialPageRoute` pushes from Settings rather than generated typed GoRouter routes; behavior matches acceptance criteria and keeps scope small.
  - Review fixes RF-1 and RF-2 added explicit cursor-driven `Load more` pagination to the settings follower/following pages and mutual followers bottom sheet.

## Implementation Review Fix Plan
Source review: `06-implementation-review.md` (`Changes required`).

| Fix Step | Review Finding | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| RF-1 | IR-001 | UT-010 | FR-008, FR-009, RULE-004 | AC-009, AC-010, AC-017 | Fails: `FollowListPage` ignores a non-null `cursor` and cannot append page 2 rows. |
| RF-2 | IR-001 | AT-003 | BR-002, FR-013, FR-016 | AC-015, AC-019 | Fails: mutual followers bottom sheet ignores a non-null `cursor` and cannot append page 2 rows. |
| RF-3 | IR-002 | IT-008 | NFR-002 | AC-014 | Fails/review gap: no supporting index migration exists for ordered follow/root-post query shapes. This step requires explicit migration approval before editing migration files. |

### RF-1: UT-010 / IR-001
- Write failing test: Added `followers page loads and appends cursor pages` in `app/test/settings/follow_list_page_test.dart`, returning a first followers page with `cursor: 'next-followers'` and asserting a second repository call with that opaque cursor appends Bob after existing rows.
- Run command: `flutter test test/settings/follow_list_page_test.dart` from `app/`.
- Confirmed failure: Test failed because no `Load more` control existed; `FollowListPage` loaded only the first page.
- Implement: Reworked `FollowListPage` state to retain accumulated account rows, total count, next cursor, and loading-more state; added a `Load more` button row that calls the same repository method with the opaque cursor and appends page 2 rows.
- Run command: `flutter test test/settings/follow_list_page_test.dart` passed.
- Refactor: None yet; mutual bottom sheet pagination remains RF-2.
- Notes: Covers UT-010 for follower/following presentation pagination while preserving app-bar total count and row order.

### RF-2: AT-003 / IR-001
- Write failing test: Extended `tapping mutual followers opens bottom sheet list` in `app/test/profile/profile_page_test.dart` so the first mutuals response returns `cursor: 'next-mutuals'`; the test taps `Load more` and asserts the repository receives that opaque cursor and appends Dana.
- Run command: `flutter test test/profile/profile_page_test.dart` from `app/`.
- Confirmed failure: Test failed because the bottom sheet had no `Load more` control and only rendered the first `ProfileAccountPage`.
- Implement: Reworked `ProfileMutualFollowersSheet` to retain accumulated mutual account rows, next cursor, and loading-more state; added a `Load more` button row that calls `listMutualFollowers` with the opaque cursor and appends returned rows.
- Run command: `flutter test test/profile/profile_page_test.dart` passed.
- Refactor: Ran `dart format` on touched Flutter profile/settings/test files.
- Notes: Covers AT-003 and IR-001 for the mutual bottom sheet while keeping the 0.9-height bottom sheet and separate endpoint behavior intact.

### RF-3: IT-008 / IR-002
- Write failing test: Added `TestProfileStore_SocialSummaryIndexesCoverOrderedQueries` in `appview/internal/api/profile_store_test.go`, asserting the test DDL includes the ordered follow and root-post index definitions from the coding plan.
- Run command: `go test ./internal/api -run TestProfileStore_SocialSummaryIndexesCoverOrderedQueries -count=1` from `appview/`.
- Confirmed failure: Test failed because `profileStoreDDL` was missing `atproto_follows_subject_created_uri_desc_idx` and the other supporting index fragments.
- Implement: Added index definitions to the profile store test schema and created approved index-only migration files `appview/migrations/000013_profile_social_summary_indexes.up.sql` / `.down.sql` for `atproto_follows(subject_did, created_at DESC, uri DESC)`, `atproto_follows(did, created_at DESC, uri DESC)`, and partial root-post `craftsky_posts(did, created_at DESC)`.
- Run command: Focused index coverage test passed.
- Refactor: Ran `gofmt` on `appview/internal/api/profile_store_test.go`.
- Notes: Explicit migration approval was obtained before touching migration files. This addresses IR-002 / NFR-002 without adding tables or changing durable data shape.

### Review Fix Verification
- `flutter test test/profile/profile_page_test.dart test/settings/follow_list_page_test.dart test/profile/data/profile_api_client_test.dart` from `app/` passed.
- `go test ./internal/api ./internal/routes` from `appview/` passed.
- Dart analyzer via MCP on `app/lib`, `app/test/profile`, and `app/test/settings` reported no errors.
- Remaining gaps: None known for IR-001 or IR-002. Manual large-list/device polish checks remain optional per `02-acceptance-tests.md` MAN-001/MAN-002.

## Following Craftsky-Only Clarification Fix Plan
Source clarification: user confirmed on 2026-05-27 that settings Following should exclude non-Craftsky followed accounts.

| Fix Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| CF-1 | IT-006 | FR-009, FR-010, FR-011, RULE-004 | AC-010, AC-011, AC-012 | Fails: `ProfileStore.ListFollowing` returns non-Craftsky followed accounts and counts them. |

### CF-1: IT-006 / FR-009
- Write failing test: Added `TestFollowAccountQueryConfig_FollowingRequiresCraftskyProfile` in `appview/internal/api/profile_store_query_test.go` and extended `TestProfileStore_ListFollowersAndFollowing_OrderNewestFirst` with non-Craftsky followed account Erin, expecting following totals/order to include Craftsky rows only.
- Run command: `go test ./internal/api -run TestFollowAccountQueryConfig_FollowingRequiresCraftskyProfile -count=1` from `appview/`.
- Confirmed failure: Build failed because `followAccountQueryConfig` did not exist. The existing DB-backed store test is skipped locally when `TEST_DATABASE_URL` / `DATABASE_URL` are unset, so the pure query-config test provides local red coverage for the Craftsky-only following predicate.
- Implement: Added `followAccountQueryConfig` and updated `ProfileStore.listFollowAccounts` so `following` count and list queries join `craftsky_profiles` on `f.subject_did`; `followers` keeps its existing query shape.
- Run command: `go test ./internal/api -run 'TestFollowAccountQueryConfig_FollowingRequiresCraftskyProfile|TestProfileStore_ListFollowersAndFollowing_OrderNewestFirst' -count=1 -v` passed, with the DB-backed store test skipped locally due missing database URL.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: This implements the user clarification that settings Following excludes non-Craftsky followed accounts. Followers behavior was left unchanged.

### Following Clarification Verification
- `go test ./internal/api ./internal/routes` from `appview/` passed.
- `flutter test test/settings/follow_list_page_test.dart` from `app/` passed.
