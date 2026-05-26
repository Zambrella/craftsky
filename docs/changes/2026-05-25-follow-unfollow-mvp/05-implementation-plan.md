# TDD Implementation Plan: Follow / Unfollow MVP

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Stop and ask for explicit approval before touching high-risk areas (auth, permissions, billing, payments, migrations, destructive actions, privacy, security, compliance).

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-004 | FR-001, FR-002, NFR-002, RULE-003 | AC-006 | Fails |
| 2 | UT-005 | FR-002, NFR-002 | AC-006 | Fails |
| 3 | UT-006 | FR-001, FR-002, RULE-007 | AC-007, AC-019 | Fails |
| 4 | IT-001 | FR-001, RULE-003, RULE-007 | AC-005, AC-006, AC-007, AC-019 | Fails |
| 5 | IT-002 | FR-001, FR-002, RULE-005 | AC-003, AC-004, AC-007 | Fails |
| 6 | UT-007 | FR-001, BR-003 | AC-005 | Fails |
| 7 | UT-008 | FR-006, RULE-005, RULE-006 | AC-003, AC-004, AC-018 | Fails |
| 8 | UT-009 | RULE-009 | AC-026 | Fails |
| 9 | IT-012 | FR-002 | AC-006, AC-007, AC-025 | Fails |
| 10 | UT-012 | BR-005, FR-006, FR-011 | AC-020, AC-021 | Fails |
| 11 | UT-013 | FR-012, RULE-008 | AC-021, AC-023 | Fails |
| 12 | IT-007 | BR-005, FR-011, FR-012, RULE-001, RULE-008 | AC-020, AC-021, AC-023 | Fails |
| 13 | IT-011 | FR-012 | AC-020, AC-021 | Fails |
| 14 | UT-010 | FR-003, FR-005, RULE-003 | AC-001 | Fails |
| 15 | UT-011 | FR-004, FR-005, RULE-004 | AC-002, AC-009 | Fails |
| 16 | IT-004 | FR-003, FR-005, RULE-001, RULE-002, RULE-003 | AC-001, AC-012, AC-013, AC-020, AC-022 | Fails |
| 17 | IT-005 | FR-004, FR-005, RULE-001, RULE-002, RULE-004 | AC-002, AC-009, AC-012, AC-013, AC-020, AC-022 | Fails |
| 18 | IT-009 | NFR-001 | AC-008, AC-012, AC-013 | Fails |
| 19 | IT-010 | FR-005, NFR-003 | AC-014 | Fails |
| 20 | UT-014 | FR-007, NFR-003 | AC-014 | Fails |
| 21 | UT-015 | FR-007 | AC-011, AC-021 | Fails |
| 22 | UT-016 | FR-007, FR-008, RULE-008 | AC-010, AC-011, AC-021, AC-023 | Fails |
| 23 | UT-017 | FR-008 | AC-024 | Fails |
| 24 | UT-018 | FR-008, FR-009 | AC-015 | Fails |
| 25 | IT-008 | FR-010 | AC-016 | Fails |
| 26 | AT-001 | BR-001, FR-003, FR-005, FR-007, FR-008, RULE-001, RULE-003 | AC-001, AC-010, AC-011, AC-022, AC-024 | Fails |
| 27 | AT-002 | BR-001, FR-004, FR-005, FR-007, FR-008, RULE-004 | AC-002, AC-009, AC-022, AC-024 | Fails |
| 28 | AT-003 | BR-001, BR-005, FR-003, FR-004, FR-011, FR-012, RULE-001 | AC-001, AC-002, AC-020, AC-021, AC-022 | Fails |
| 29 | AT-004 | BR-002, FR-001, FR-006, RULE-005 | AC-003, AC-004, AC-011 | Fails |
| 30 | AT-005 | BR-005, FR-006, FR-011, RULE-008 | AC-021, AC-023 | Fails |
| 31 | AT-006 | FR-008, FR-009 | AC-015 | Fails |
| 32 | AT-007 | FR-003, FR-004, FR-006, RULE-002, RULE-006 | AC-013, AC-018 | Fails |
| 33 | AT-008 | BR-004, FR-002, FR-010, RULE-005 | AC-016, AC-017, AC-025 | Fails |
| 34 | MAN-001 | BR-004, FR-010 | Live Tap historical delivery smoke check | Pending manual |
| 35 | MAN-002 | BR-001, BR-005, FR-003, FR-004 | End-to-end follow/unfollow smoke check | Pending manual |
| 36 | MAN-003 | NFR-004 | Profile query-plan check | Pending manual |

## Implementation Steps

### Step 1: UT-004 (FR-001, FR-002, NFR-002, RULE-003)

- Write failing test:
  - Create `appview/internal/index/bluesky_follow_test.go` with `TestBlueskyFollow_CreateIdempotent` covering duplicate create delivery for identical `(URI, CID)`.
  - Use a focused test schema for `atproto_follows` and assert one active row.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/index -run TestBlueskyFollow_CreateIdempotent`
- Confirmed failure:
  - Red: compile failure `undefined: index.NewBlueskyFollow` from `bluesky_follow_test.go`.
- Implement:
  - Add `appview/internal/index/bluesky_follow.go` with create-path handling and idempotent upsert semantics.
- Run command:
  - Green focused: `go test ./internal/index -run TestBlueskyFollow_CreateIdempotent` passes.
  - Nearby suite: `go test ./internal/index` currently fails at pre-existing `TestCraftskyPost_Create_WithImages_StoresSizeAndAspectRatio` CID fixture parsing; failure is outside follow-indexer scope.
- Refactor:
  - None.
- Notes:
  - Added `BlueskyFollow` indexer with `app.bsky.graph.follow` collection gating and upsert semantics keyed by `uri`.
  - Delete action support was added minimally to keep action handling consistent and prepare for UT-006.

### Subsequent Steps

- Execute steps 2 through 36 in the listed order with strict red-green-refactor loops.
- Update this file after each loop with:
  - failing assertion/error summary,
  - minimal implementation applied,
  - focused command and green result,
  - any nearby regression command run.

### Step 4: IT-001 (FR-001, RULE-003, RULE-007)

- Write failing test:
  - Added `TestFollowStore_ActiveGraphSemantics` in `appview/internal/api/follow_store_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestFollowStore_ActiveGraphSemantics`
- Confirmed failure:
  - Red: compile failure `undefined: api.NewFollowStore` and `undefined: api.FollowRow`.
- Implement:
  - Added `appview/internal/api/follow_store.go` with:
    - `FollowRow`
    - `NewFollowStore`
    - `UpsertActive`
    - `DeleteActiveByURI`
    - `ListActiveFollowedDIDs`
  - Added migrations:
    - `appview/migrations/000012_atproto_follows.up.sql`
    - `appview/migrations/000012_atproto_follows.down.sql`
- Run command:
  - Green focused command above.
  - Green nearby command: `go test ./internal/api -run TestFollowStore_`.
- Refactor:
  - None.
- Notes:
  - User explicitly approved proceeding with migration changes before this loop.

### Step 5: IT-002 (FR-001, FR-002, RULE-005)

- Write failing test:
  - Added `TestProfileStore_ReadByDID_CraftskyOnlyCounts` in `appview/internal/api/profile_store_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestProfileStore_ReadByDID_CraftskyOnlyCounts`
- Confirmed failure:
  - Red: compile failure `ProfileRow` missing `FollowerCount` and `FollowingCount`.
- Implement:
  - Extended `ProfileRow` in `profile_store.go` with count fields.
  - Updated `ProfileStore.Read` query to compute Craftsky-account-only counts using joins from `atproto_follows` to `craftsky_profiles`.
- Run command:
  - Green focused command above.
  - Green nearby command: `go test ./internal/api -run TestProfileStore_`.
- Refactor:
  - None.
- Notes:
  - Count semantics now match RULE-005 for Craftsky profiles in store integration coverage.

### Step 6: UT-007 (FR-001, BR-003)

- Write failing test:
  - Added `TestFollowStore_ListActiveFollowedDIDs_OnlyActiveUnique` in `appview/internal/api/follow_store_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestFollowStore_ListActiveFollowedDIDs_OnlyActiveUnique`
- Confirmed failure:
  - No red state; test was green on first run because list/collapse behavior was already provided by step-4 implementation.
- Implement:
  - No code changes required for this loop.
- Run command:
  - Green focused command above.
- Refactor:
  - None.
- Notes:
  - Explicitly verifies active followed DID lookup for future feed work (AC-005).

### Step 7: UT-008 (FR-006, RULE-005, RULE-006)

- Write failing test:
  - Added `TestBuildProfileResponse_IncludesFollowStateAndCraftskyCounts` in `appview/internal/api/profile_response_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestBuildProfileResponse_IncludesFollowStateAndCraftskyCounts`
- Confirmed failure:
  - Red: compile failure due missing `ProfileRow`/`ProfileResponse` follow-state and count fields.
- Implement:
  - Extended `ProfileRow` with `ViewerIsFollowing` and `IsCraftskyProfile`.
  - Extended `ProfileResponse` with `viewerIsFollowing`, `isCraftskyProfile`, `followerCount`, `followingCount`.
  - Updated `BuildProfileResponse` mapping.
- Run command:
  - Green focused command above.
  - Green nearby command: `go test ./internal/api -run TestBuildProfileResponse_`.
- Refactor:
  - None.

### Step 8: UT-009 (RULE-009)

- Write failing test:
  - Added `TestGetProfile_CountsUnavailable` in `appview/internal/api/profile_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestGetProfile_CountsUnavailable`
- Confirmed failure:
  - Red expectation not previously supported by handler contract.
- Implement:
  - Added `ErrProfileCountsUnavailable` sentinel.
  - Mapped count-calculation failures to `profile_counts_unavailable` in `writeProfileResponse`.
  - Added count-query error wrapping in `ProfileStore.Read` when `atproto_follows` resolution fails.
- Run command:
  - Green focused command above.

### Step 9: IT-012 (FR-002)

- Write failing test:
  - Added `TestNewIndexerDispatcherRegistersBlueskyFollow` in `appview/internal/app/indexer_wiring_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/app -run TestNewIndexerDispatcherRegistersBlueskyFollow`
- Confirmed failure:
  - Red: dispatcher returned `indexer: not yet implemented` for follow NSID.
- Implement:
  - Registered `app.bsky.graph.follow` in `newIndexerDispatcher` with `index.NewBlueskyFollow(pool)`.
  - Extended test DDL with `atproto_follows` table.
  - Updated `docker-compose.yml` `TAP_COLLECTION_FILTERS` to include `app.bsky.graph.follow`.
- Run command:
  - Green focused command above.
  - Green nearby command: `go test ./internal/app -run TestNewIndexerDispatcher`.

### Step 10: UT-012 (BR-005, FR-006, FR-011)

- Write failing test:
  - Added `TestBuildProfileResponse_NonCraftskyProfileHasNilCounts` in `appview/internal/api/profile_response_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestBuildProfileResponse_NonCraftskyProfileHasNilCounts`
- Confirmed failure:
  - No red state; test was green on first run after step-7 response-field implementation.
- Implement:
  - No code changes required for this loop.

### Step 11: UT-013 (FR-012, RULE-008)

- Write failing test:
  - Updated `TestBlueskyProfile_DropsForNonMember` to `TestBlueskyProfile_CreatesForNonMember` in `appview/internal/index/bluesky_profile_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/index -run TestBlueskyProfile_CreatesForNonMember`
- Confirmed failure:
  - Red: count remained 0 due membership gating.
- Implement:
  - Removed Craftsky-membership gate in `appview/internal/index/bluesky_profile.go` so non-member `app.bsky.actor.profile` events are indexed.
- Run command:
  - Green focused command above.
  - Green nearby command: `go test ./internal/index -run TestBlueskyProfile_`.

### Step 12: IT-007 (BR-005, FR-011, FR-012, RULE-001, RULE-008)

- Write failing test:
  - Added `TestProfileStore_ReadByDID_NonCraftskyFromBlueskyCache` in `appview/internal/api/profile_store_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestProfileStore_ReadByDID_NonCraftskyFromBlueskyCache`
- Confirmed failure:
  - Red: `profile: not found` for non-member DID despite cached `bluesky_profiles` row.
- Implement:
  - Added non-Craftsky fallback read path in `ProfileStore` (`readNonCraftsky`) using `bluesky_profiles`.
  - Set `isCraftskyProfile=false`, empty crafts, nil counts for non-member rows.
  - Updated profile GET response to include `createdAt` only for Craftsky profiles.
- Run command:
  - Green focused command above.
  - Green nearby command: `go test ./internal/api -run TestProfileStore_`.

### Step 13: IT-011 (FR-012)

- Coverage source:
  - Satisfied by non-member indexer behavior validated in step-11 (`TestBlueskyProfile_CreatesForNonMember`).
- Notes:
  - `bluesky_profiles` can now contain rows without corresponding `craftsky_profiles` membership.

### Step 2: UT-005 (FR-002, NFR-002)

- Write failing test:
  - Added `TestBlueskyFollow_UpdateUpsertsByURI` in `appview/internal/index/bluesky_follow_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/index -run TestBlueskyFollow_UpdateUpsertsByURI`
- Confirmed failure:
  - No red state; test was green on first run because step-1 implementation already performs URI-keyed upsert for create/update.
- Implement:
  - No code changes required for this loop.
- Run command:
  - Green focused command above.
  - Green nearby command: `go test ./internal/index -run TestBlueskyFollow_`.
- Refactor:
  - None.
- Notes:
  - Behavior now explicitly covered: update with changed CID/subject updates existing URI row without duplicating rows.

### Step 3: UT-006 (FR-001, FR-002, RULE-007)

- Write failing test:
  - Added `TestBlueskyFollow_DeleteRemovesRowAndUnknownDeleteIsNoop` in `appview/internal/index/bluesky_follow_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/index -run TestBlueskyFollow_DeleteRemovesRowAndUnknownDeleteIsNoop`
- Confirmed failure:
  - No red state; test was green on first run because delete handling was included in step-1 implementation.
- Implement:
  - No additional code changes required for this loop.
- Run command:
  - Green focused command above.
  - Green nearby command: `go test ./internal/index -run TestBlueskyFollow_`.
- Refactor:
  - None.
- Notes:
  - Behavior covered: delete removes active row by URI; unknown delete is safe no-op.

### Step 15: UT-011 (FR-004, FR-005, RULE-004)

- Write failing test:
  - Added `TestUnfollowProfileHandler_DeletesActiveRecordAndReturnsProfile` in `appview/internal/api/follow_test.go`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestUnfollowProfileHandler_DeletesActiveRecordAndReturnsProfile`
- Confirmed failure:
  - Red: compile failure `undefined: api.UnfollowProfileHandler`.
- Implement:
  - Added `UnfollowProfileHandler` in `appview/internal/api/follow.go`.
  - Mirrors follow-target validation and identity resolution behavior from POST.
  - Deletes active follow record through PDS using stored `rkey` when present.
  - Treats `auth.ErrRecordNotFound` as idempotent success and removes local active graph row.
  - Returns updated profile response for active and no-active paths.
- Run command:
  - Green focused command above.
  - Green nearby command: `go test ./internal/api -run 'Test(FollowProfileHandler|UnfollowProfileHandler)_'`.

### Step 16: IT-004 (FR-003, FR-005, RULE-001, RULE-002, RULE-003)

- Write failing/contract tests:
  - Added to `appview/internal/api/follow_test.go`:
    - `TestFollowProfileHandler_InvalidIdentifier`
    - `TestFollowProfileHandler_SelfRejected`
    - `TestFollowProfileHandler_AlreadyFollowingIsIdempotent`
    - `TestFollowProfileHandler_AllowsNonCraftskyTarget`
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'Test(FollowProfileHandler|UnfollowProfileHandler)_'`
- Confirmed failure:
  - Initial invalid-identifier test setup used malformed request URL and panicked in `httptest.NewRequest`; test fixture was corrected to use a valid URL with invalid path value.
- Implement:
  - No additional handler logic required after UT-010 baseline; tests now lock endpoint behavior for invalid/self/non-Craftsky/idempotent follow cases.
- Run command:
  - Green focused command above.

### Step 17: IT-005 (FR-004, FR-005, RULE-001, RULE-002, RULE-004)

- Write/extend tests:
  - Added to `appview/internal/api/follow_test.go`:
    - `TestUnfollowProfileHandler_NoActiveIsIdempotent`
    - `TestUnfollowProfileHandler_InvalidIdentifier`
    - `TestUnfollowProfileHandler_SelfRejected`
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'Test(FollowProfileHandler|UnfollowProfileHandler)_'`
- Confirmed failure:
  - Red for active-delete path was captured in Step 15 (`undefined: api.UnfollowProfileHandler`).
- Implement:
  - Reused Step-15 implementation and locked behavior for active/no-active/invalid/self targets.
- Run command:
  - Green focused command above.

### Step 18: IT-009 (NFR-001)

- Write failing tests:
  - Added route tests in `appview/internal/routes/routes_test.go`:
    - `TestRoutes_PostProfileFollowRequiresAuth`
    - `TestRoutes_PostProfileFollowRequiresDeviceID`
    - `TestRoutes_DeleteProfileFollowRequiresAuth`
    - `TestRoutes_DeleteProfileFollowRequiresDeviceID`
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/routes -run 'TestRoutes_(PostProfileFollowRequiresAuth|PostProfileFollowRequiresDeviceID|DeleteProfileFollowRequiresAuth|DeleteProfileFollowRequiresDeviceID)'`
- Confirmed failure:
  - Red: all four tests returned 404 (follow routes not registered).
- Implement:
  - Registered follow routes in `appview/internal/routes/routes.go`:
    - `POST /v1/profiles/{handleOrDid}/follows`
    - `DELETE /v1/profiles/{handleOrDid}/follows`
  - Wired dependencies by adding `FollowStore` to `appview/internal/app/deps.go` and initializing it with `api.NewFollowStore(pool)`.
  - Strengthened device-ID error envelope assertion (`error`, `message`, `requestId`) in `TestRoutes_PostProfileFollowRequiresDeviceID`.
- Run command:
  - Green focused command above.

### Step 19: IT-010 (FR-005, NFR-003)

- Write/extend tests:
  - Extended `TestFollowProfileHandler_WritesFollowRecordAndReturnsProfile` to assert:
    - AppView builds PDS client with server-side OAuth session ID (`sess-alice`).
    - Follow response body contains no `accessToken` or `refreshToken` fields.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'Test(FollowProfileHandler|UnfollowProfileHandler)_'`
- Confirmed result:
  - Green; token-boundary behavior now explicitly locked in handler tests.
- Notes:
  - Flutter-side API client token-boundary coverage remains scheduled under `UT-014`/`REG-004`.

### Step 20: UT-014 (FR-007, NFR-003)

- Write failing test:
  - Extended `app/test/profile/data/profile_api_client_test.dart` with:
    - `POST follow uses Craftsky endpoint and no token fields`
    - `DELETE unfollow uses Craftsky endpoint and no token fields`
- Run command:
  - `cd app && flutter test test/profile/data/profile_api_client_test.dart`
- Confirmed failure:
  - Red: `ProfileApiClient` missing `followProfile` and `unfollowProfile` methods.
- Implement:
  - Added methods in `app/lib/profile/data/profile_api_client.dart`:
    - `followProfile(String handleOrDid)` -> `POST /v1/profiles/@$handleOrDid/follows`
    - `unfollowProfile(String handleOrDid)` -> `DELETE /v1/profiles/@$handleOrDid/follows`
- Run command:
  - Green focused command above.

### Step 21: UT-015 (FR-007)

- Write failing test:
  - Added `app/test/profile/models/profile_test.dart` with cases for:
    - `viewerIsFollowing`, `isCraftskyProfile`, `followerCount`, `followingCount`
    - non-Craftsky null/unknown count handling.
- Run command:
  - `cd app && flutter test test/profile/models/profile_test.dart`
- Confirmed failure:
  - Red: `Profile` model missing new follow/count fields.
- Implement:
  - Extended `app/lib/profile/models/profile.dart` with:
    - `viewerIsFollowing bool`
    - `isCraftskyProfile bool`
    - `followerCount int?`
    - `followingCount int?`
  - Regenerated mapper/provider outputs via `dart run build_runner build`.
- Run command:
  - Green focused command above.

### Step 22: UT-016 (FR-007, FR-008, RULE-008)

- Write failing tests:
  - Extended `app/test/profile/profile_page_test.dart` to assert:
    - visitor follow label shows `Follow` / `Unfollow` from profile state,
    - Craftsky counts are rendered from profile model,
    - non-Craftsky marker text `Non Craftsky profile`,
    - unknown counts render as placeholders rather than fake numbers.
- Run command:
  - `cd app && flutter test test/profile/profile_page_test.dart`
- Confirmed failure:
  - Red: page still used placeholder follow state and placeholder stats; marker missing.
- Implement:
  - `app/lib/profile/pages/profile_page.dart`: use `profile.viewerIsFollowing` for visitor action state.
  - `app/lib/profile/widgets/profile_meta_section.dart`: render `Non Craftsky profile` marker and pass real `followingCount`/`followerCount` into `ProfileStats`.
  - `app/lib/l10n/app_en.arb`: changed `profileFollowingAction` label to `Unfollow` and regenerated l10n.
- Run command:
  - Green focused command above.

### Step 23: UT-017 (FR-008)

- Write failing test:
  - Added `app/test/profile/providers/toggle_follow_profile_provider_test.dart` with
    - `sets loading and optimistic cache while request is in flight`.
- Run command:
  - `cd app && flutter test test/profile/providers/toggle_follow_profile_provider_test.dart`
- Confirmed failure:
  - Red compile: missing `toggle_follow_profile_provider.dart`, missing repository follow hooks in fakes.
- Implement:
  - Added follow/unfollow contracts to `ProfileRepository` and implementations:
    - `app/lib/profile/data/profile_repository.dart`
    - `app/lib/profile/data/api_profile_repository.dart`
    - `app/lib/profile/data/dummy_profile_repository.dart`
    - `app/test/profile/fakes/fake_profile_repository.dart`
  - Added `app/lib/profile/providers/toggle_follow_profile_provider.dart` with optimistic cache update + loading state.
  - Regenerated code via build runner.
- Run command:
  - Green focused command above.

### Step 24: UT-018 (FR-008, FR-009)

- Write failing tests:
  - Extended `toggle_follow_profile_provider_test.dart` with
    - `rolls back cache and surfaces error when follow fails`.
    - `unfollow updates optimistic state then confirms server response`.
  - Extended `profile_page_test.dart` with:
    - `failed follow restores previous state and shows error`.
    - `tapping Unfollow updates profile from repository response`.
- Run command:
  - `cd app && flutter test test/profile/providers/toggle_follow_profile_provider_test.dart`
  - `cd app && flutter test test/profile/profile_page_test.dart`
- Confirmed failure:
  - Red (widget): follow failure path kept optimistic `Unfollow` state.
- Implement:
  - Refactored `toggle_follow_profile_provider.dart` to explicit `try/catch` flow:
    - rollback cached profile before setting `AsyncError`.
    - avoid race where listener reset could clear error state before rollback check.
  - `profile_page.dart`: listens to `toggleFollowProfileProvider` errors and shows `profileFollowToggleError`, then resets provider state.
  - Added l10n key `profileFollowToggleError` and regenerated l10n.
  - Added busy-state plumbing to `VisitorProfileActionSet` and button disable behavior in `profile_actions.dart`.
- Run command:
  - Green focused commands above.

### Step 25: IT-008 (FR-010)

- Write test:
  - Added `TestBlueskyFollow_HistoricalEventCreatesActiveRow` in `appview/internal/index/bluesky_follow_test.go` using `tap.Event{Live:false}`.
- Run command:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/index -run TestBlueskyFollow_HistoricalEventCreatesActiveRow`
- Confirmed result:
  - Green on first run; existing indexer behavior already accepted historical events (no live-gate).
- Notes:
  - Added explicit regression coverage for AC-016 historical delivery semantics.

## Completion Checklist

- [ ] All Must requirements covered by tests or documented gaps
- [ ] All planned Must tests passing
- [ ] Relevant regression tests passing
- [ ] No unlinked behavior implemented
- [ ] Docs updated (`05-implementation-plan.md`)
- [ ] Review completed or explicitly skipped
