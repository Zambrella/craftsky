# TDD Implementation Plan: Onboarding Experience

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Coding plan: `04-coding-plan.md`
- High-risk implementation approval: Granted by the user on 2026-08-31 for OAuth projection, authenticated onboarding status, migration/private persistence, per-DID authorization, and account-deletion cleanup.

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability and executed commands updated after every loop.
- Preserve OAuth/Tap fallback, authenticated-DID ownership, atomic profile semantics, and exact account-lease boundaries.
- Do not modify lexicons, profile wire routes, OAuth wire shapes, or add durable Flutter completion markers.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-009 | FR-024, RULE-007 | AC-041, AC-042 | Fails: fetched Bluesky profile and CID are discarded |
| 2 | IT-009, AT-019, REG-008 | FR-024, NFR-004, RULE-007 | AC-041, AC-042, AC-043 | Fails: no canonical row exists before handoff |
| 3 | IT-001 | FR-018, RULE-003, RULE-004 | AC-030, AC-031, AC-038 | Fails: completion store and handlers do not exist |
| 4 | IT-002 | FR-018 | AC-030, AC-031 | Fails: authenticated routes and policies do not exist |
| 5 | IT-005 | FR-018, RULE-003 | AC-030, AC-031 | Fails: migration, cleanup, and terminal inventory are absent |
| 6 | UT-004, IT-008, AT-018 | FR-009, FR-019, FR-020 | AC-032, AC-033, AC-034 | Fails: completion is synchronous device-local state |
| 7 | UT-006, AT-013, REG-003, REG-006, REG-007 | FR-018, FR-020, RULE-004, NFR-003 | AC-030, AC-034, AC-038 | Fails: startup/router do not await server authority |
| 8 | UT-001, UT-002, UT-005 | FR-003, FR-006, FR-011, FR-012, FR-022, RULE-005 | AC-006, AC-010, AC-016, AC-017, AC-027, AC-028, AC-039 | Fails: no action, progress, or flow-state model exists |
| 9 | UT-003, AT-016, IT-004 | FR-014, FR-021 | AC-023, AC-035, AC-036 | Fails: profile prefill blocks the editable onboarding page |
| 10 | UT-007, AT-017, IT-003, REG-001, REG-004 | FR-005, RULE-001, RULE-006 | AC-009, AC-040 | Fails: no onboarding payload composer exists |
| 11 | AT-003, AT-004, MAN-001 | FR-002, FR-003, FR-014, FR-023 | AC-004, AC-005, AC-006, AC-023, AC-037 | Fails: placeholder has no profile controls |
| 12 | AT-005, REG-005 | FR-004 | AC-007 | Fails: placeholder has no craft controls |
| 13 | AT-006, AT-007, AT-008 | BR-002, FR-005, FR-007, FR-010, FR-011, FR-017, FR-022, RULE-003, RULE-005 | AC-003, AC-008, AC-011, AC-015, AC-016, AC-027, AC-028, AC-029, AC-039 | Fails: required save/navigation behavior is absent |
| 14 | AT-009, AT-010, AT-011, AT-012, UT-008, IT-007, REG-002 | FR-008, FR-009, FR-013, FR-015, FR-016 | AC-012, AC-013, AC-014, AC-018, AC-024, AC-025, AC-026 | Fails: reusable Instagram sections are absent |
| 15 | AT-001, AT-002, AT-014, AT-015, IT-006, MAN-002 | BR-001, FR-001, FR-006, FR-012, FR-017, NFR-001, NFR-002, NFR-003, RULE-002, RULE-003 | AC-001, AC-002, AC-010, AC-017, AC-019, AC-020, AC-021, AC-022, AC-029 | Fails: full responsive and account-safe flow is absent |
| 16 | REG-006 | FR-020 | AC-034 | Fails: OAuth handoff can precede canonical CraftSky membership projection |
| 17 | REG-006 follow-up | FR-020, RULE-007 | AC-034, AC-043 | Fails: repeated identical accepted profile writes remain permanently source-order uncertain |

## Implementation Steps

### Step 17: REG-006 Follow-up
- Write failing test: Reproduce two accepted onboarding profile writes with the same URI, record fingerprint, and CID across lifecycle generations, followed by a live uncertain source and an authoritative PDS read.
- Red command: `cd appview && TEST_DATABASE_URL='postgres://craftsky:dev@127.0.0.1:16008/craftsky_dev?sslmode=disable' go test ./internal/ingestion -run TestAuthoritativeReconciliationSelectsNewestProvenAcceptedIdenticalProfile -count=1`.
- Confirmed failure: Yes. Authoritative reconciliation returned retryable `source_order_uncertain` with `PDS effect source match is ambiguous`, leaving the profile projection absent and account initialization endpoints on `profile_not_found`.
- Implement: Permit an authoritative observation to select the newest duplicate candidate only when that candidate independently records `accepted` with a non-empty result CID exactly matching the authoritative PDS CID. Unknown outcomes and A-to-B-to-A histories remain fail-closed.
- Green commands: Focused reconciliation and existing provenance tests passed; `GOFLAGS='-p=1' TEST_DATABASE_URL='postgres://craftsky:dev@127.0.0.1:16008/craftsky_dev?sslmode=disable' go test ./... -count=1` passed.
- Live verification: Rebuilt the AppView and queued the supported read-only reconciliation for `craftskymod1.bsky.social`. The durable job completed, the source became authoritative and linked to the newest accepted operation, `craftsky_profiles` was restored, startup requests succeeded, and the connected app reached onboarding with no runtime errors.
- Notes: A concurrent first verification attempt exhausted PostgreSQL shared lock memory; the serial full suite passed. A later isolated `just appview-check` exceeded the ten-minute tool timeout before completion, so the new correction relies on the green serial full suite plus live durable-recovery evidence.

### Step 16: REG-006
- Write failing test: Verify OAuth initialization projects the fetched or newly written CraftSky profile through the canonical idempotent indexer before handoff-dependent current-member requests can begin.
- Red command: `cd appview && go test ./internal/app ./internal/auth -run 'TestOAuth.*Craftsky.*Projection|TestInitializeProfileAndIdentityCache.*Craftsky'`
- Confirmed failure: Yes. The focused auth test failed to compile because the durable onboarding writer discarded the authoritative CID and OAuth initialization had no CraftSky profile projector boundary.
- Implement: Return the authoritative profile CID from the durable onboarding effect, retain fetched/new CraftSky record data, project it through the canonical idempotent indexer before Bluesky projection and auxiliary effects, and fail callback finalization if this required membership projection fails. Threaded the narrow projector through app and route composition.
- Green commands: `cd appview && go test ./internal/auth -run 'TestInitializeProfileAndIdentityCacheProjectsNewCraftskyProfileBeforeAuxiliaryEffects'`; `cd appview && go test ./...`; `just appview-check` (all pass).
- Refactor: Reused the existing Tap event/indexer boundary rather than adding direct membership SQL. Added event-shape, authoritative-CID, fail-closed ordering, and real-Postgres pre-handoff assertions while green.
- Notes: Manual registration logs showed the Tap source event arrive before handoff exchange while `/v1/languages/preferences` and `/v1/onboarding/status` still returned the shared `profile_not_found` membership response. The real registration flow now verifies `craftsky_profiles` exists before creating the handoff exchange; Tap remains the durable fallback and idempotent replay path.

### Step 1: UT-009
- Write failing test: Verify present/missing profile branching, DID/CID/body forwarding, best-effort failure, safe warning, and projection before subsequent effects.
- Run command: `cd appview && go test ./internal/auth -run 'TestInitializeProfile|TestOAuth.*Projection'`
- Confirmed failure: Yes. The first focused test failed to compile because `InitializeProfileAndIdentityCache` had no projector dependency and treated the projector as the identity-cache argument.
- Implement: Added the narrow `BlueskyProfileProjector` interface, retained the fetched record/CID, invoked projection under a bounded child context before repository tracking and identity-cache publication, and kept projection failure warning-only with redacted logging.
- Run command: `cd appview && go test ./internal/auth -run 'TestInitializeProfileAndIdentityCache(Project|Projection)'`; `cd appview && go test ./internal/auth` (pass).
- Refactor: Removed the redundant exported `InitializeProfile` entry point and moved existing tests to the sole production initializer while green.
- Notes: Missing Bluesky profile skips projection; a projection error does not stop later effects or OAuth initialization. Application dependency wiring remains Step 2.

### Step 2: IT-009, AT-019, REG-008
- Write failing test: Verify the real canonical projection is visible before handoff and same-CID Tap replay remains idempotent.
- Run command: `cd appview && go test ./internal/app ./internal/index -run 'TestOAuth.*Bluesky.*Projection|TestBluesky(Profile|Backfiller)'`
- Confirmed failure: Yes. `TestOAuthBlueskyProfileProjectionUsesCanonicalTapEvent` failed to compile because the app-layer adapter did not exist.
- Implement: Added the app adapter, constructed it with `index.NewBlueskyProfile`, and threaded the narrow projector through app, routes, and OAuth handler dependencies.
- Run command: `cd appview && go test ./internal/app -run TestOAuthBlueskyProfileProjection`; `go test ./internal/auth`; `go test ./internal/routes`; `go test ./internal/index -run 'TestBluesky(Profile|Backfiller)'` (pass).
- Refactor: Kept event handling behind a narrow local interface so adapter tests do not duplicate index parsing.
- Notes: The real-Postgres same-CID test is present but skipped locally because no database URL is configured; `just appview-check` remains required. The adapter calls `BlueskyProfile.Handle`, and existing tracking/backfill tests remain green.

### Step 3: IT-001
- Write failing test: Verify incomplete status, permanent/idempotent completion, original timestamp, per-DID isolation, lifecycle guard, and envelopes.
- Run command: `cd appview && go test ./internal/api -run TestOnboarding`
- Confirmed failure: Yes. The handler test first failed to compile without the status model/handler; the persistence test then failed without `NewOnboardingStatusStore`.
- Implement: Added camelCase status/completion handlers and a guarded store with missing-row incomplete semantics, idempotent insert, and authoritative timestamp re-read.
- Run command: `cd appview && go test ./internal/api -run 'Test(GetOnboardingStatus|OnboardingStatusStore)'`; `go test ./internal/api` (pass; database-backed case skips without a configured database).
- Refactor: Kept one shared handler path for read/complete while preserving separate public constructors.
- Notes: The authenticated context is the only DID source; no DID selector is decoded.

### Step 4: IT-002
- Write failing test: Verify auth, device, current-member, no-body, no-query, method, and route policy behavior.
- Run command: `cd appview && go test ./internal/routes -run TestOnboarding`
- Confirmed failure: Yes. The focused policy test panicked because `/v1/onboarding/status` had no registered policy.
- Implement: Registered both bodyless current-member policies and routes, constructing the store route-locally.
- Run command: `cd appview && go test ./internal/routes -run 'TestOnboarding|TestAddRoutesRegistersExpectedInventory'`; `go test ./internal/routes` (pass).
- Refactor: Added a narrow onboarding route bundle and updated the exact route inventory.
- Notes: Existing middleware supplies bearer, device, current-member lifecycle, body, and rate enforcement.

### Step 5: IT-005
- Write failing test: Verify migration shape/up-down behavior, Alice-only private cleanup, terminal inventory, and terminal purge.
- Run command: `cd appview && go test ./internal/db ./internal/accountdeletion ./internal/ownerlifecycle -run 'Test.*Onboarding|TestTerminalDIDInventory'`
- Confirmed failure: The approved schema/inventory was absent; migration and inventory coverage would fail once the new store table became part of the migrated schema.
- Implement: Added migration 000062, explicit private cleanup, and terminal DID inventory registration.
- Run command: `cd appview && go test ./internal/accountdeletion`; `go test ./internal/ownerlifecycle`; `go test ./internal/db -run TestMigration` (pass; database-backed cases require final release gate).
- Refactor: None.
- Notes: No foreign key or cascade was added; the primary key provides the role-leading unique purge key.

### Step 6: UT-004, IT-008, AT-018
- Write failing test: Verify API wire calls, immediate optimistic completion, finite silent retries, lease ownership, disposal, and no persistent marker.
- Red command: `cd app && flutter test test/onboarding/providers/onboarding_status_provider_test.dart test/onboarding/data/api_onboarding_repository_test.dart test/onboarding/onboarding_completion_test.dart`
- Confirmed failure: Yes. The focused tests initially had no API completion model/client/repository or session-lease-scoped asynchronous status authority to exercise.
- Implement: Added the completion model, API client/repository, immediate optimistic completion, bounded silent retry, exact `AccountSessionLease` ownership, retry-only retention, and cancellation on disposal/lease change without a persistent Flutter marker.
- Green command: `cd app && flutter test test/onboarding/providers/onboarding_status_provider_test.dart test/onboarding/data/api_onboarding_repository_test.dart test/onboarding/onboarding_completion_test.dart` (`All tests passed!`).
- Refactor: Kept server completion state separate from visible active-account draft state and retained the notifier only while retries are active.
- Notes: Completion remains optimistic for the current session while the AppView is authoritative across sessions.

### Step 7: UT-006, AT-013, REG-003, REG-006, REG-007
- Write failing test: Verify loading/error do not choose a route, completion bypasses onboarding, retry invalidates exact dependencies, and no preferences authority remains.
- Red command: `cd app && flutter test test/auth test/router`
- Confirmed failure: Yes. Startup and routing did not yet wait for server-backed onboarding status and stale router fixtures still supplied the removed device-local authority.
- Implement: Integrated completion into active-account initialization, the initialization gate, router redirects, retry invalidation, and notification readiness. Migrated router fixtures to `activeAccountInitializationProvider` and retained the router provider in tests to prevent auto-disposal.
- Green commands: `cd app && flutter test test/auth test/router`; `cd app && flutter test test/router/router_redirect_test.dart` (20 tests passed in the focused router run).
- Refactor: Removed `onboarding_refresh_listener.dart`; routing now derives only from resolved initialization for the exact active lease.
- Notes: Loading and error remain in the initialization gate and do not choose an authenticated destination.

### Step 8: UT-001, UT-002, UT-005
- Write failing test: Verify action derivation, progress values, sequential draft retention, and reconstruction reset.
- Red command: `cd app && flutter test test/onboarding/onboarding_action_state_test.dart test/onboarding/onboarding_progress_test.dart test/onboarding/onboarding_flow_state_test.dart`
- Confirmed failure: Yes. The action, progress, and process-local flow-state models did not exist.
- Implement: Added pure action derivation, one-based progress values, sequential draft retention, and persisted-profile reconstruction that resets to step one.
- Green command: `cd app && flutter test test/onboarding/onboarding_action_state_test.dart test/onboarding/onboarding_progress_test.dart test/onboarding/onboarding_flow_state_test.dart` (`All tests passed!`).
- Refactor: Kept action/progress derivation pure and flow state process-local under `ActiveAccountLease`.
- Notes: No draft or completion marker is persisted by Flutter.

### Step 9: UT-003, AT-016, IT-004
- Write failing test: Verify the editable onboarding draft appears before profile loading completes, delayed profile data populates untouched fields, successful empty responses retry for at most five seconds in the background, user edits are never overwritten, and read errors leave the draft usable.
- Red command: `cd app && flutter test test/onboarding/onboarding_profile_prefill_test.dart`
- Confirmed failure: Yes. The flow awaited the full empty-profile retry window before publishing state, causing a roughly four-second spinner for legitimately empty accounts.
- Implement: Publish an editable draft immediately from the active session, then run the existing bounded profile retry as best-effort background work. Apply delayed profile data only while the original draft is untouched, synchronize the mounted field controllers when prefill arrives, stop cleanly on disposal or account changes, and leave the draft usable when reads fail.
- Green commands: `cd app && flutter test test/onboarding/onboarding_profile_prefill_test.dart test/onboarding/onboarding_profile_step_test.dart`; `cd app && flutter test test/onboarding` (`All tests passed!`, 39 tests).
- Refactor: Kept retry and lease fencing inside the flow provider while separating initial draft publication from background prefill.
- Notes: Profile read failures no longer replace onboarding with an error screen. User input and navigation take precedence over delayed prefill.

### Step 10: UT-007, AT-017, IT-003, REG-001, REG-004
- Write failing test: Verify complete editable snapshots, unknown craft preservation, image omission semantics, and unchanged last-write-wins behavior.
- Red command: `cd app && flutter test test/onboarding/onboarding_profile_payload_test.dart test/profile/edit_profile_dialog_test.dart`
- Confirmed failure: Yes. No onboarding payload composer existed. A later AT-008 red case also showed that rebuilding all fields from one step could overwrite another step's unsaved draft.
- Implement: Added explicit complete-profile payload composition, unknown craft preservation, image omission semantics, shared constraints/cache publication, and step-owned payload rebasing that preserves drafts from other steps.
- Green commands: `cd app && flutter test test/onboarding/onboarding_profile_payload_test.dart test/profile/edit_profile_dialog_test.dart`; `cd app && flutter test test/onboarding/onboarding_draft_navigation_test.dart test/onboarding/onboarding_profile_payload_test.dart` (`All tests passed!`; the focused edit-profile regression run passed 10 tests).
- Refactor: Kept one payload composer with baseline values for fields not owned by the current step.
- Notes: No client CID or pre-save server merge was added; writes retain last-write-wins behavior.

### Step 11: AT-003, AT-004, MAN-001
- Write failing test: Verify Bluesky prefill, read-only handle, gallery-only replacement, dirty state, validation, and upload feedback.
- Red command: `cd app && flutter test test/onboarding/onboarding_profile_step_test.dart`
- Confirmed failure: Yes. The placeholder had no editable identity controls or avatar action. The first AT-004 review loop then failed because upload state changed action enablement without visible progress/failure feedback.
- Implement: Added Bluesky-prefilled editable identity fields, read-only handle, shared gallery image picking, validation/dirty propagation, localized `Uploading photo...` progress, localized retryable failure feedback, and disabled repeat picking while upload/save is active.
- Green command: `cd app && flutter test test/onboarding/onboarding_profile_step_test.dart` (`All tests passed!`).
- Refactor: Kept avatar picking behind the shared profile image picker provider and upload feedback in the profile step.
- Notes: MAN-001 is deferred/unavailable in this environment because supported physical/simulator iOS and Android gallery targets were not available. Automated pending/failure/recovery widget coverage passes, but it is not manual platform evidence.

### Step 12: AT-005, REG-005
- Write failing test: Verify existing known selections, all 22 catalogue items/order, semantics, dirty state, and empty selection.
- Red command: `cd app && flutter test test/onboarding/onboarding_crafts_step_test.dart`
- Confirmed failure: Yes. The placeholder had no craft catalogue, selection callback, or selected-state semantics.
- Implement: Added the crafts step by reusing `EditProfileCraftsPicker` and the stable `Craft.values` catalogue, with toggle/empty-selection support and authoritative non-duplicated chip semantics.
- Green commands: `cd app && flutter test test/onboarding/onboarding_crafts_step_test.dart test/profile/edit_profile_dialog_test.dart` (`All tests passed!`); after correcting a test-only Flutter tri-state matcher, `cd app && flutter test test/onboarding/onboarding_crafts_step_test.dart` (1 test passed).
- Refactor: Excluded descendant chip semantics so each craft has one accessible selected-state announcement.
- Notes: Unknown craft IDs remain hidden in the catalogue and preserved in submitted full-profile snapshots.

### Step 13: AT-006, AT-007, AT-008
- Write failing test: Verify single-flight save/advance, failure recovery, Back behavior, immediate isolated Skip, and in-session drafts.
- Red commands: `cd app && flutter test test/onboarding/onboarding_save_navigation_test.dart`; `cd app && flutter test test/onboarding/onboarding_draft_navigation_test.dart test/onboarding/onboarding_profile_payload_test.dart`.
- Confirmed failure: Yes. Save/navigation was not composed initially. The AT-008 regression then proved a dirty craft draft was discarded after Back, an identity edit, and a successful identity save.
- Implement: Added single-flight save, success-only advance, retryable failure state, deterministic Back, immediate completion-only Skip, and per-step rebasing that preserves other-step drafts.
- Green commands: `cd app && flutter test test/onboarding/onboarding_save_navigation_test.dart`; `cd app && flutter test test/onboarding/onboarding_completion_test.dart test/onboarding/onboarding_instagram_step_test.dart`; `cd app && flutter test test/onboarding/onboarding_draft_navigation_test.dart test/onboarding/onboarding_profile_payload_test.dart` (all passed).
- Refactor: Centralized save/advance and completion mutations in the flow controller.
- Notes: Submitted profile save disables Skip and Back; avatar upload alone does not disable Skip.

### Step 14: AT-009, AT-010, AT-011, AT-012, UT-008, IT-007, REG-002
- Write failing test: Verify shared unlinked/linked/reactivation/import/suggestion states, no onboarding history/revoke/navigation, and non-blocking Finish.
- Red command: `cd app && flutter test test/onboarding/onboarding_instagram_step_test.dart test/instagram_migration`
- Confirmed failure: Yes. The Instagram migration page exposed only the settings composition; onboarding had no reusable scoped sections.
- Implement: Extracted shared public unlinked/linked/reactivation/import/suggestion sections, recomposed settings with settings-only history/revoke/navigation, and hosted only approved sections in onboarding. Finish remains independent of Instagram state.
- Green commands: `cd app && flutter test test/onboarding/onboarding_completion_test.dart test/onboarding/onboarding_instagram_step_test.dart`; `cd app && flutter test test/onboarding test/profile/edit_profile_dialog_test.dart test/instagram_migration` (108 tests passed).
- Refactor: Kept settings-only controls outside the shared onboarding composition.
- Notes: The import composer retains its provider without history mounted.

### Step 15: AT-001, AT-002, AT-014, AT-015, IT-006, MAN-002
- Write failing test: Verify the complete sequential flow, optional empty completion, progress semantics, responsive layout, and stale-account rejection.
- Red commands: `cd app && flutter test test/onboarding/onboarding_accessibility_layout_test.dart test/onboarding/onboarding_account_isolation_test.dart`; focused router/page tests were also run while wiring startup entry and completion exit.
- Confirmed failure: Yes. Dedicated compact/large-text progress semantics and stale-account completion coverage did not exist; duplicate descendant semantics were exposed during the accessibility loop.
- Implement: Finished sequential responsive page composition, localized progress/actions, 320x480 layout behavior at 2x text scaling, authoritative progress/craft semantics, and exact-lease rejection of stale account-A work after switching to account B.
- Green commands: `cd app && flutter test test/onboarding/onboarding_accessibility_layout_test.dart test/onboarding/onboarding_account_isolation_test.dart`; `cd app && flutter test test/onboarding test/profile/edit_profile_dialog_test.dart test/instagram_migration` (108 tests passed).
- Refactor: Removed duplicate progress/chip announcements while preserving visible labels and operable actions.
- Notes: MAN-002 is deferred/unavailable in this environment because supported device/simulator visual, keyboard, screen-reader, focus-order, and platform accessibility targets were not available. Automated compact large-text, progress-semantics, and action-operability coverage passes, but it is not manual target evidence.

## Final Verification
- `cd app && flutter test test/onboarding test/profile/edit_profile_dialog_test.dart test/instagram_migration`: 108 tests passed.
- `cd app && flutter test`: 1,637 tests passed (`All tests passed!`). The first full run exposed one test-only `bool` versus `Tristate.isTrue` matcher mismatch in AT-005; after correcting the assertion, the final full run was green.
- Dart MCP full-project analysis for `app/`: `No errors`.
- `dart format` on the final touched onboarding test files: formatted successfully with 0 files changed on the final runs.
- Initial `just appview-check`: blocked by `TestProviderRegistrationCredentialCleanupConverges` because its fixed clock violated the credential expiry constraint.
- Correction-pass `just appview-check`: passed all release gates, including real PostgreSQL tests, race coverage, migration down-to-zero, and identical reapply. Artifacts: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.tUcNjvUcLT`.
- `git status --short`, `git diff --stat`, and `git diff --check` were inspected. Unrelated generated hash/EOF-only churn was removed; the final diff check is clean.
- MAN-001 and MAN-002: deferred/unavailable as documented in Steps 11 and 15.

## Completion Checklist
- [x] All Flutter Must requirements covered by automated tests or documented manual gaps
- [x] All planned automated Must Flutter tests passing
- [x] Relevant Flutter regression tests passing
- [x] No unlinked Flutter behavior implemented
- [x] Flutter implementation outcomes documented
- [x] MAN-001 explicitly deferred/unavailable
- [x] MAN-002 explicitly deferred/unavailable
- [x] Review completed
- [x] AppView release gate green

## Implementation Review Correction Pass

### IR-001: Skip During Prefill
- Test IDs: AT-007, AT-016
- Confirmed failure: Skip was absent while profile initialization was loading.
- Implemented: Kept account-scoped Skip available during prefill loading and retryable errors.
- Verification: Focused widget tests pass.

### IR-002: Elapsed Prefill Bound
- Test IDs: UT-003, AT-016
- Confirmed failure: The retry policy counted only delays and excluded request duration.
- Implemented: Enforced an injected elapsed deadline that includes in-flight profile reads.
- Verification: Slow-response and five-second-stop tests pass.

### IR-003: Instagram Import Readiness
- Test IDs: AT-009, IT-007
- Confirmed failure: Import creation remained enabled before initial provider readiness.
- Implemented: Gated import actions on ready state and exposed retry behavior without relying on history being mounted.
- Verification: Composer loading/error/readiness tests pass.

### IR-004: Exact-Lease Routing
- Test IDs: AT-013, AT-015, UT-006
- Confirmed failure: Stale resolved initialization could redirect the newly active account.
- Implemented: Router redirects now require equality with the registry's exact active lease; status failure retry coverage was added.
- Verification: Account switch, reauthentication, and retry tests pass.

### IR-005 And IR-008: Flutter Acceptance And Semantics
- Test IDs: AT-009 through AT-012, AT-014, AT-015, AT-017, IT-003, IT-006, IT-007
- Implemented: Added observable Instagram actions/parity, stale-operation isolation, atomic identity/craft update preservation, and labeled busy-action semantics.
- Verification: 46 correction-focused tests pass; the final full Flutter suite passes 1,651 tests; Dart analysis reports no errors.

### IR-006, IR-007, And IR-010: AppView Evidence
- Test IDs: IT-001, IT-002, AT-019, IT-009, REG-008, IT-005
- Confirmed failures: Stale lifecycle reads succeeded; no projection existed before recording handoff; cleanup fixture used the wrong owner column; the fixed credential-cleanup clock violated its database constraint; handlers lacked a logger seam.
- Implemented: Added full-mux API policy cases, lifecycle-fenced reads, real pre-handoff canonical projection and Tap replay, migration up/down/reapply tests, Alice/Bob cleanup and terminal inventory evidence, safe completion-failure logging, and a current test clock.
- Verification: Focused real-PostgreSQL packages pass and `just appview-check` passes all release gates.

### Final Correction Verification
- `cd app && flutter test`: 1,651 tests passed.
- Dart MCP full-project analysis: no errors.
- `just appview-check`: all release gates passed.
- `git diff --check`: clean after removing unrelated generated EOF-only churn.
- MAN-001 and MAN-002 remain deferred as documented; no supported mobile/screen-reader target was available.

### Runtime Registration Regression Verification
- Root cause: Tap had durably received the new `social.craftsky.actor.profile` event, but asynchronous projection had not populated `craftsky_profiles` before Flutter exchanged and confirmed the handoff. Both current-member startup reads therefore returned `profile_not_found`.
- `cd appview && go test ./...`: all packages passed.
- `just appview-check`: all release gates passed; `TestProviderRegistrationCompletesSharedOnboardingAndConfirmedHandoff` passed against real PostgreSQL with the membership row asserted before handoff creation.
- `git diff --check`: clean.
