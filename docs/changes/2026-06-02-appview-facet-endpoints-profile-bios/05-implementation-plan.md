# TDD Implementation Plan: AppView Facet Endpoints And Plain Profile Bios

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
- Preserve the documented guardrails: no lexicon changes, no profile `descriptionFacets`, no non-Craftsky mention targets, no PDS tokens in Flutter, no migration-time network work, and no handles on `bluesky_profiles`.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 | BR-001, FR-001, FR-002, FR-005, NFR-001, RULE-001 | AC-001, AC-003, AC-012, AC-014, AC-015 | `/v1/facets/*` routes missing / return 404 |
| 2 | UT-001 | FR-001, FR-005, NFR-002 | AC-014, EC-007 | No request parser exists; validation behavior undefined |
| 3 | UT-002 | FR-002 | AC-015 | DTOs do not exist |
| 4 | UT-003 | FR-001, FR-002, NFR-002 | AC-005 | Ranking helper/store method missing |
| 5 | UT-004 | FR-003, FR-004, RULE-001 | AC-006, EC-001 | Exact resolve handler missing |
| 6 | IT-002 | FR-001, FR-013, FR-014 | AC-004, AC-013, AC-018 | Cache schema/store missing |
| 7 | IT-003 | FR-003, FR-004, FR-014, RULE-001 | AC-002, AC-006, AC-013, AC-018 | Refresh/filter exact resolve flow missing |
| 8 | IT-004 / UT-005 | FR-005, FR-006, NFR-002, RULE-003 | AC-003, AC-007, EC-002 | Hashtag query/count method missing |
| 9 | IT-009 | FR-016 | AC-021, EC-008 | Profile initialization path does not update identity cache |
| 10 | IT-005 | FR-015 | AC-019 | CLI command missing |
| 11 | UT-006 / UT-008 | FR-007, FR-005, BR-001 | AC-001, AC-003, AC-012 | AppView repository classes missing |
| 12 | UT-007 / AT-002 | FR-003, FR-008, BR-001 | AC-002, AC-006, AC-016 | Exact AppView resolver not wired to generator |
| 13 | IT-006 / AT-001 / AT-003 / AT-006 | FR-007, FR-008, BR-001 | AC-001, AC-003, AC-012, EC-006 | Production providers still mock-backed / error handling not locked |
| 14 | UT-011 / IT-007 / REG-001 / REG-004 / AT-004 | BR-002, FR-009, FR-010, RULE-002 | AC-008, AC-009, AC-010 | `descriptionFacets` still in model/API/save flow |
| 15 | UT-009 / UT-010 / IT-008 / AT-005 | BR-003, FR-011, FR-012, NFR-003 | AC-010, AC-011, AC-017, AC-020, EC-003, EC-004 | Bio depends on stored facets; parser absent |
| 16 | REG-002 | FR-008 | AC-002, AC-016 | Parser refactor could break byte offsets or AT facet JSON |

## Implementation Steps

### Step 1: IT-001
- Write failing test: Added `TestAddRoutes_FacetRoutesRegisteredAndRequireAuthenticatedDevice` covering registration plus auth/device enforcement for `GET /v1/facets/mentions`, `GET /v1/facets/mentions/resolve`, and `GET /v1/facets/hashtags`.
- Run command: `go test ./internal/routes -run TestAddRoutes_FacetRoutesRegisteredAndRequireAuthenticatedDevice -count=1` from `appview/`.
- Confirmed failure: Routes fell through to `/` and returned 404 instead of being registered behind auth/device middleware.
- Implement: Added AppView facet request/response/handler skeletons and registered the three `/v1/facets/*` routes with `authN(deviceID(...))` in `routes.go`.
- Run command: `go test ./internal/routes -run TestAddRoutes_FacetRoutesRegisteredAndRequireAuthenticatedDevice -count=1` from `appview/`.
- Refactor: None.
- Notes: Route-level auth/device contract is green; response body/store semantics are covered by later focused steps.

### Step 2: UT-001
- Write failing test: Added `TestParseFacetSuggestionRequestQueryAndLimitBounds` for empty/whitespace query, 64/65-character bounds, default limit, accepted limits 1/10/25, and rejected limit 26.
- Run command: `go test ./internal/api -run TestParseFacetSuggestionRequestQueryAndLimitBounds -count=1` from `appview/`.
- Confirmed failure: Not observed; the parser had already been introduced during the Step 1 route skeleton, so the focused test was immediately green.
- Implement: No additional implementation needed.
- Run command: `go test ./internal/api -run TestParseFacetSuggestionRequestQueryAndLimitBounds -count=1` from `appview/` passed.
- Refactor: None.
- Notes: This step locks the resolved `limit > 25` validation-error decision and empty/whitespace query handling.

### Step 3: UT-002
- Write failing test: Added `TestFacetMentionSuggestionJSONOmitsUnknownOptionalFields` for required DID/handle/`isCraftskyProfile`/`viewerIsFollowing` fields and omitted unknown `displayName`/`avatar`.
- Run command: `go test ./internal/api -run TestFacetMentionSuggestionJSONOmitsUnknownOptionalFields -count=1` from `appview/`.
- Confirmed failure: Not observed; DTOs were introduced by the Step 1 route skeleton and already matched this contract.
- Implement: No additional implementation needed.
- Run command: `go test ./internal/api -run TestFacetMentionSuggestionJSONOmitsUnknownOptionalFields -count=1` from `appview/` passed.
- Refactor: None.
- Notes: Wire JSON uses camelCase and omits unknown optional fields.

### Step 4: UT-003
- Write failing test: Added `TestRankMentionSuggestionRowsFollowedPrefixThenHandle` for followed-first, prefix-before-substring, and handle-ascending ordering.
- Run command: `go test ./internal/api -run TestRankMentionSuggestionRowsFollowedPrefixThenHandle -count=1` from `appview/`.
- Confirmed failure: Build failed with `undefined: RankMentionSuggestionRows`.
- Implement: Added `RankMentionSuggestionRows` helper.
- Run command: `go test ./internal/api -run TestRankMentionSuggestionRowsFollowedPrefixThenHandle -count=1` from `appview/` passed.
- Refactor: None.
- Notes: Removed an accidentally-added future hashtag test before running, keeping this loop scoped to UT-003.

### Step 5: UT-004
- Write failing test: Added `TestResolveFacetMentionHandlerSuccessAndMentionNotFound` for minimal success response and non-Craftsky/missing `404 mention_not_found` envelope mapping.
- Run command: `go test ./internal/api -run TestResolveFacetMentionHandlerSuccessAndMentionNotFound -count=1` from `appview/`.
- Confirmed failure: Not observed; the exact resolve handler skeleton already satisfied this focused handler contract.
- Implement: No additional implementation needed.
- Run command: `go test ./internal/api -run TestResolveFacetMentionHandlerSuccessAndMentionNotFound -count=1` from `appview/` passed.
- Refactor: None.
- Notes: Store-level Craftsky filtering and cache refresh behavior remain covered by IT-003.

### Step 6: IT-002
- Write failing test: Added `TestFacetStoreSearchMentionSuggestionsUsesFreshSeparateIdentityCache` with `craftsky_profiles`, `bluesky_profiles`, `atproto_follows`, and separate `atproto_identity_cache` rows covering fresh, exactly-24h, stale, and non-Craftsky identities.
- Run command: `go test ./internal/api -run TestFacetStoreSearchMentionSuggestionsUsesFreshSeparateIdentityCache -count=1` from `appview/`.
- Confirmed failure: The real Postgres helper may skip when `TEST_DATABASE_URL`/`DATABASE_URL` is absent; no runtime red was observable in this environment. The pre-implementation store would have returned empty rows for the seeded case.
- Implement: Added migration `000015_identity_handle_cache`, `FacetStore.SearchMentionSuggestions`, and the fresh-cache Craftsky-only join/query using the separate identity cache table.
- Run command: `go test ./internal/api -run TestFacetStoreSearchMentionSuggestionsUsesFreshSeparateIdentityCache -count=1` from `appview/` passed/skipped per local DB availability.
- Refactor: None.
- Notes: Autocomplete uses cached identities only and treats entries resolved at `now - 24h` as fresh.

### Step 7: IT-003
- Write failing test: Added `TestFacetStoreResolveMentionRefreshesCacheAndFiltersCraftskyProfiles` with stale Alice cache, resolver-backed canonical refresh, and non-Craftsky Mallory filtering.
- Run command: `go test ./internal/api -run TestFacetStoreResolveMentionRefreshesCacheAndFiltersCraftskyProfiles -count=1` from `appview/`.
- Confirmed failure: Build failed because `NewFacetStore` did not accept a `HandleResolver` and `ResolveMention` was a stub.
- Implement: Added resolver injection, `IdentityCacheStore.FreshByHandle`, `Upsert`, `IsCraftskyProfile`, and resolver-backed `FacetStore.ResolveMention` that refreshes missing/stale Craftsky rows and maps unresolved/non-Craftsky identities to `ErrMentionNotFound`.
- Run command: `go test ./internal/api -run TestFacetStoreResolveMentionRefreshesCacheAndFiltersCraftskyProfiles -count=1` from `appview/` passed/skipped per local DB availability.
- Refactor: None.
- Notes: Exact resolve now refreshes canonical handles and does not insert rows for non-Craftsky accounts.

### Step 8: IT-004 / UT-005
- Write failing test: Added `TestNormalizeHashtagSuggestionRowsLowercaseCountsAndSorts` plus `TestFacetStoreSearchHashtagSuggestionsCountsRecentRootPosts` for mixed casing, duplicate tags, empty tags, recent root-only counts, old posts, and replies.
- Run command: `go test ./internal/api -run TestNormalizeHashtagSuggestionRowsLowercaseCountsAndSorts -count=1` from `appview/`.
- Confirmed failure: Build failed with `undefined: NormalizeHashtagSuggestionRows`.
- Implement: Added hashtag row normalization and `FacetStore.SearchHashtagSuggestions` using `craftsky_posts.tags`, root-post predicates, 28-day cutoff, lowercase canonical grouping, `COUNT(DISTINCT p.uri)`, and count-desc/tag-asc ordering.
- Run command: `go test ./internal/api -run 'TestNormalizeHashtagSuggestionRowsLowercaseCountsAndSorts|TestFacetStoreSearchHashtagSuggestionsCountsRecentRootPosts' -count=1` from `appview/` passed/skipped DB portions per local DB availability.
- Refactor: None.
- Notes: Counts represent root posts only and response tags are lowercase without leading `#`.

### Step 9: IT-009
- Write failing test: Added `TestInitializeProfileAndIdentityCacheUpsertsAfterSuccessfulInitialization` and failure variant proving identity-cache upsert is attempted after successful profile initialization and transient upsert failures are logged/continued.
- Run command: `go test ./internal/auth -run TestInitializeProfileAndIdentityCache -count=1` from `appview/`.
- Confirmed failure: Build failed with `undefined: auth.InitializeProfileAndIdentityCache`.
- Implement: Added narrow `auth.IdentityCacheUpdater`, `InitializeProfileAndIdentityCache`, `api.IdentityCacheService.UpsertCurrentHandle`, `app.Deps.IdentityCacheUpdater`, and OAuth route wiring so callback/profile initialization invokes the updater without importing `api` into `auth`.
- Run command: `go test ./internal/auth -run TestInitializeProfileAndIdentityCache -count=1` from `appview/` passed.
- Refactor: None.
- Notes: Upsert failures do not create partial rows in auth; exact resolve/backfill remain recovery paths.

### Step 10: IT-005
- Write failing test: Added `TestIdentityCacheBackfillCommandUsesDefaultAndExplicitLimits` for `identity-cache backfill` default limit `100` and explicit `--limit 10`.
- Run command: `go test ./cmd/cli -run TestIdentityCacheBackfillCommandUsesDefaultAndExplicitLimits -count=1` from `appview/`.
- Confirmed failure: Build failed with missing `newIdentityCacheCmd` and `identityCacheBackfillStats`.
- Implement: Added `cmd/cli/identity_cache.go`, the `identity-cache backfill` command, bounded runner, `IdentityCacheStore.BackfillCandidateDIDs`, and resolver/upsert loop.
- Run command: `go test ./cmd/cli -run TestIdentityCacheBackfillCommandUsesDefaultAndExplicitLimits -count=1` from `appview/` passed.
- Refactor: None.
- Notes: SQL migration remains schema-only; backfill performs network/identity resolution at command runtime.

### Step 11: UT-006 / UT-008
- Write failing test: Added `app/test/shared/rich_text/facet_suggestion_repository_test.dart` for mention and hashtag endpoint mapping/decoding with Dio mock adapter.
- Run command: `flutter test test/shared/rich_text/facet_suggestion_repository_test.dart` from `app/`.
- Confirmed failure: Test load failed because `appview_facet_suggestion_repository.dart` and repository classes were missing.
- Implement: Added Dio-backed `AppViewAccountSuggestionRepository` and `AppViewHashtagSuggestionRepository`; switched production providers to use `dioProvider` while leaving mock repositories available for tests/overrides.
- Run command: `flutter test test/shared/rich_text/facet_suggestion_repository_test.dart` from `app/` passed.
- Refactor: None.
- Notes: Suggestion errors fail closed with empty lists; exact resolver maps failures to `null` for later facet generation.

### Step 12: UT-007 / AT-002
- Write failing test: Extended `facet_suggestion_repository_test.dart` with `UT-007 exact resolve maps success and mention_not_found to final facets`, using the AppView repository as the `FacetGenerator` mention resolver.
- Run command: `flutter test test/shared/rich_text/facet_suggestion_repository_test.dart` from `app/`.
- Confirmed failure: Not observed; Step 11's repository implementation already supported exact resolve and null fallback.
- Implement: No additional implementation needed.
- Run command: `flutter test test/shared/rich_text/facet_suggestion_repository_test.dart` from `app/` passed.
- Refactor: None.
- Notes: Final text remains source of truth; no hidden selected-mention state was added.

### Step 13: IT-006 / AT-001 / AT-003 / AT-006
- Write failing test: No new test was needed; existing `facet_autocomplete_editor_test.dart` already covered provider override/composer insertion behavior.
- Run command: `flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart` from `app/`.
- Confirmed failure: Not observed.
- Implement: No additional implementation needed beyond Step 11 provider wiring and fail-closed repositories.
- Run command: `flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart` from `app/` passed.
- Refactor: None.
- Notes: Composer autocomplete preserves insertion/order behavior via repository overrides; production providers are now AppView-backed.

### Step 14: UT-011 / IT-007 / REG-001 / REG-004 / AT-004
- Write failing test: Updated stale profile API/edit-dialog tests to assert plain `description` only, no `descriptionFacets`, and no autocomplete in the bio editor.
- Run command: `flutter test test/profile/data/profile_api_client_test.dart` from `app/` initially still passed under old expectations, revealing stale tests rather than missing implementation.
- Confirmed failure: Stale tests and implementation still referenced `descriptionFacets`; updated tests would fail to compile until API/model/save signatures were changed.
- Implement: Removed `descriptionFacets` from `Profile`, generated mapper, API client, repository interfaces/implementations, save provider, fake repository, `ProfileBio` API, meta section, and edit dialog save path; replaced bio `FacetAutocompleteEditor` with plain `BrandTextField`/`TextEditingController`.
- Run command: `flutter test test/profile/data/profile_api_client_test.dart test/profile/edit_profile_dialog_facets_test.dart` from `app/` passed.
- Refactor: Regenerated Dart mapper/provider code with `dart run build_runner build --delete-conflicting-outputs`.
- Notes: Profile bio editor is now plain text and saves no facet metadata.

### Step 15: UT-009 / UT-010 / IT-008 / AT-005
- Write failing test: Updated `profile_bio_test.dart` from stored-facet fixtures to plain-bio fixtures for bare domains, dotted handles, hashtags, unsupported schemes, and URL fragment overlap behavior.
- Run command: `flutter test test/profile/widgets/profile_bio_test.dart` from `app/`.
- Confirmed failure: Test load failed because `ProfileBio` no longer accepted `descriptionFacets`, and plain parsing was not yet implemented.
- Implement: Added `facet_token_parser.dart` and changed `ProfileBio` to derive render-time raw facets from plain description text.
- Run command: `flutter test test/profile/widgets/profile_bio_test.dart` and `flutter test test/shared/rich_text/faceted_text_actions_test.dart` from `app/` passed.
- Refactor: None.
- Notes: Bio mentions route by visible handle through existing action handler; bare domains normalize to HTTPS and unsupported schemes remain plain text.

### Step 16: REG-002
- Write failing test: Used existing `facet_generator_test.dart` regression fixtures for emoji byte offsets, links, tags, mentions, and URL-fragment overlap.
- Run command: `flutter test test/shared/rich_text/facet_generator_test.dart` from `app/` passed before refactor.
- Confirmed failure: Not observed; this was a regression/refactor step guarded by existing green tests.
- Implement: Refactored `FacetGenerator` to consume `detectSupportedFacetTokens` from the shared parser introduced for profile bios.
- Run command: `flutter test test/shared/rich_text/facet_generator_test.dart test/profile/widgets/profile_bio_test.dart` from `app/` passed.
- Refactor: Centralized token detection in `facet_token_parser.dart` so posts and profile bios share supported token semantics.
- Notes: AT Protocol raw facet byte offsets and overlap behavior were preserved.

## Verification Log
- Formatting:
  - `gofmt -w ...` on changed Go files.
  - `dart format ...` on changed Dart/test files.
- Focused AppView/Go verification:
  - `go test ./internal/routes -run TestAddRoutes_FacetRoutesRegisteredAndRequireAuthenticatedDevice -count=1`
  - `go test ./internal/api -run TestParseFacetSuggestionRequestQueryAndLimitBounds -count=1`
  - `go test ./internal/api -run TestFacetMentionSuggestionJSONOmitsUnknownOptionalFields -count=1`
  - `go test ./internal/api -run TestRankMentionSuggestionRowsFollowedPrefixThenHandle -count=1`
  - `go test ./internal/api -run TestResolveFacetMentionHandlerSuccessAndMentionNotFound -count=1`
  - `go test ./internal/api -run TestFacetStoreSearchMentionSuggestionsUsesFreshSeparateIdentityCache -count=1` (real DB portions skip when `TEST_DATABASE_URL`/`DATABASE_URL` is absent)
  - `go test ./internal/api -run TestFacetStoreResolveMentionRefreshesCacheAndFiltersCraftskyProfiles -count=1` (real DB portions skip when `TEST_DATABASE_URL`/`DATABASE_URL` is absent)
  - `go test ./internal/api -run 'TestNormalizeHashtagSuggestionRowsLowercaseCountsAndSorts|TestFacetStoreSearchHashtagSuggestionsCountsRecentRootPosts' -count=1` (real DB portions skip when `TEST_DATABASE_URL`/`DATABASE_URL` is absent)
  - `go test ./internal/auth -run TestInitializeProfileAndIdentityCache -count=1`
  - `go test ./cmd/cli -run TestIdentityCacheBackfillCommandUsesDefaultAndExplicitLimits -count=1`
- Focused Flutter verification:
  - `flutter test test/shared/rich_text/facet_suggestion_repository_test.dart`
  - `flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart`
  - `flutter test test/profile/data/profile_api_client_test.dart test/profile/edit_profile_dialog_facets_test.dart`
  - `flutter test test/profile/widgets/profile_bio_test.dart`
  - `flutter test test/shared/rich_text/faceted_text_actions_test.dart`
  - `flutter test test/shared/rich_text/facet_generator_test.dart test/profile/widgets/profile_bio_test.dart`
- Broader verification:
  - `go test ./...` from `appview/` passed.
  - `flutter test test/shared/rich_text test/profile test/feed/providers/create_post_provider_test.dart` from `app/` passed.
  - Dart MCP analysis of `app/lib`, `app/test/shared/rich_text`, and `app/test/profile` reported no errors.
  - `git diff --check` passed.
- Manual checks `MAN-001` through `MAN-004` were not run in this automated stage.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Stage completion commit created or explicitly skipped because there were no stage changes
- [x] Review completed or explicitly skipped
