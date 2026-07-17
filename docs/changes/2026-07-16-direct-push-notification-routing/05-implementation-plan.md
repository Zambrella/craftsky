# TDD Implementation Plan: Direct Push Notification Routing

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
- Preserve binding validity independently from fact validity.
- Keep destination inference provider-neutral and keep authenticated AppView reads authoritative.
- Do not delete the resolver until the replacement AppView producer and Flutter consumer are green.
- Do not create a stage commit unless the user explicitly enables commits.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 1 | UT-001 | FR-001, FR-005, FR-006, FR-011, FR-012, FR-014 | AC-002, AC-005, AC-006, AC-011, AC-012, AC-014 | Fails |
| 2 | UT-002, UT-003, REG-009 | FR-002, FR-005, FR-012, FR-019 | AC-002, AC-005, AC-012, AC-021 | Fails |
| 3 | UT-006, AT-002 | BR-002, FR-006, RULE-003 | AC-006 | Fails |
| 4 | UT-004, UT-005, AT-003 | FR-002, FR-003, FR-011, FR-012, FR-016, FR-019, RULE-004 | AC-003, AC-011, AC-012, AC-016, AC-021 | Fails |
| 5 | UT-008, AT-008 | FR-018 | AC-022 | Fails |
| 6 | UT-011, UT-012 | FR-001, FR-002, FR-004, FR-014, FR-017, NFR-003, RULE-001 | AC-002, AC-004, AC-014, AC-016, AC-018 | Fails |
| 7 | IT-001 | FR-001, FR-002, FR-003, FR-017 | AC-002, AC-003, AC-016 | Fails |
| 8 | IT-002, REG-001 | FR-001, FR-004, FR-014, NFR-003, RULE-001 | AC-002, AC-004, AC-014, AC-018, AC-020 | Fails |
| 9 | UT-007, IT-003, AT-001 | BR-001, FR-007, FR-011, FR-018, NFR-001 | AC-001, AC-003, AC-007, AC-011, AC-022 | Fails |
| 10 | IT-004, AT-004, REG-010 | BR-002, FR-006, FR-013, RULE-003, RULE-006 | AC-006, AC-013, AC-020 | Fails |
| 11 | UT-014, AT-005, IT-009 | FR-016 | AC-003, AC-016 | Fails |
| 12 | UT-009 | FR-010 | AC-010 | Fails |
| 13 | IT-005 | BR-002, FR-008, RULE-002, RULE-005 | AC-008 | Verify or extend |
| 14 | AT-006, IT-006 | FR-009, NFR-005, RULE-005 | AC-009 | Fails |
| 15 | AT-007, IT-007, REG-006 | FR-010, NFR-005, RULE-006 | AC-010, AC-020 | Fails |
| 16 | UT-013, AT-009, IT-011, REG-005 | FR-015, RULE-006 | AC-015, AC-020 | Fails |
| 17 | IT-008, REG-008 | FR-001, FR-014, FR-015 | AC-014, AC-015 | Fails |
| 18 | UT-010, IT-010, REG-007 | FR-005, NFR-002, NFR-004, RULE-001 | AC-005, AC-017, AC-019 | Fails |
| 19 | UT-015 | NFR-004 | AC-019 | Fails if boundary drifts |
| 20 | REG-002 | RULE-006 | AC-020 | Passes unless delivery semantics drift |
| 21 | REG-003 | RULE-006 | AC-020 | Passes unless lifecycle semantics drift |
| 22 | REG-004 | RULE-006 | AC-020 | Passes unless list/newness semantics drift |
| 23 | MAN-001–MAN-005 | FR-006, FR-009, FR-013, NFR-001, RULE-003, RULE-005 | AC-006, AC-007, AC-009, AC-013 | Manual release gate; unavailable in host automation |

## Implementation Steps

### Step 1: UT-001

- Write failing test: Added structured-attempt coverage for independent binding/fact validity, ignored legacy `notificationId`, source preservation, and redacted diagnostics.
- Run command: `cd app && flutter test test/notifications/models/notification_open_event_test.dart`
- Confirmed failure: Yes. The test failed to compile because `NotificationOpenAttempt`, `ValidNotificationFacts`, and `InvalidNotificationFacts` did not exist.
- Implement: Added the provider-neutral structured attempt, independently nullable binding parse, bounded common version/type classification, sealed fact outcomes, and identifier-free diagnostics while retaining the old event temporarily for the ordered cutover.
- Run command: `cd app && flutter test test/notifications/models/notification_open_event_test.dart`
- Refactor: Formatted the source/test and reran the focused suite while green.
- Notes: Passed 6 tests. The old `NotificationOpenEvent` remains temporarily because later ordered steps migrate consumers before the resolver removal slice.

### Step 2: UT-002, UT-003, REG-009

- Write failing test: Added one UT-002 matrix test for every known category and one UT-003/REG-009 boundary test for malformed DIDs, arbitrary URLs, non-post collections, invalid authorities/record keys, non-ASCII, and over-bound facts.
- Run command: `cd app && flutter test test/notifications/models/notification_open_event_test.dart`
- Confirmed failure: Yes. UT-002 first failed because typed fact getters did not exist. After the minimum category matrix passed, UT-003 failed with an uncaught `InvalidDidError`, proving the provider boundary was not safely rejecting invalid identifiers.
- Implement: Added private valid-fact construction with category-specific typed DID/AT-URI fields, exact required-field parsing, ignored extras, 1024-byte ASCII bounds, strict AT-URI validation, exact post collection/DID authority checks, and record-key validation.
- Run command: `cd app && flutter test test/notifications/models/notification_open_event_test.dart`
- Refactor: Kept raw facts out of `toString`, formatted while green, and reran the focused suite.
- Notes: Passed 5 tests. The parser now rejects literal URLs and non-post AT-URIs without throwing, while preserving a valid binding for the later fallback policy.

### Step 3: UT-006, AT-002

- Write failing test: Replaced the resolver-oriented coordinator test with one binding matrix covering absent payload binding, absent local binding, mismatch, and match.
- Run command: `cd app && flutter test test/notifications/providers/notification_open_coordinator_test.dart`
- Confirmed failure: Yes. The coordinator had no structured-attempt callback and still required the resolution repository path.
- Implement: Changed the coordinator to accept `NotificationOpenAttempt`, load the secure DID-keyed binding first, emit only generic unavailable feedback for every invalid binding state, and release validated facts only on an exact match.
- Run command: `cd app && flutter test test/notifications/providers/notification_open_coordinator_test.dart`
- Refactor: Removed resolver/model/policy imports from the coordinator, formatted while green, and reran the focused test.
- Notes: Passed 1 test. Destination inference is deliberately the next red-green loop; the binding gate itself has no resolver or network callback.

### Step 4: UT-004, UT-005, AT-003

- Write failing test: Added a UT-004 category-to-destination matrix and a separate UT-005/AT-003 invalid-versus-unknown fallback test.
- Run command: `cd app && flutter test test/notifications/services/notification_destination_inference_test.dart`
- Confirmed failure: Yes. UT-004 failed because the destination model/inference service did not exist. After its minimum mapping passed, UT-005 failed because invalid facts had no `unableToOpen` feedback.
- Implement: Added provider-neutral Notifications/profile/post destinations, optional reply focus, open outcomes, and pure inference for all valid, unknown, and invalid fact outcomes.
- Run command: `cd app && flutter test test/notifications/services/notification_destination_inference_test.dart test/notifications/providers/notification_open_coordinator_test.dart test/notifications/models/notification_open_event_test.dart`
- Refactor: Changed the green coordinator seam from validated facts to final inferred outcomes, keeping binding as the mandatory first gate.
- Notes: Passed 8 focused tests. Inference imports no Firebase, network, UI context, or GoRouter types; invalid facts request Notifications with feedback and unknown valid types do so quietly.

### Step 5: UT-008, AT-008

- Write failing test: Updated the latest-only/sign-in-discard test to use structured attempts with distinct callback sources.
- Run command: `cd app && flutter test test/notifications/services/pending_notification_open_test.dart`
- Confirmed failure: Yes. `PendingNotificationOpen` still accepted and returned the obsolete `NotificationOpenEvent` shape.
- Implement: Changed the one-slot in-memory pending state to accept and return `NotificationOpenAttempt` without changing readiness semantics.
- Run command: `cd app && flutter test test/notifications/services/pending_notification_open_test.dart`
- Refactor: Formatted while green; no new queue, persistence, timer, or deduplication state was added.
- Notes: Passed 1 test. The newest transient attempt is released once, and `requiresSignIn` clears it permanently.

### Step 6: UT-011, UT-012

- Write failing test: Replaced the old payload test with an exact UT-011 category matrix, then added UT-012 for maximum reply data and over-bound fact rejection.
- Run command: `cd appview && go test ./internal/push -count=1`
- Confirmed failure: Yes. UT-011 failed because `RoutingFacts` and the version 1 builder signature did not exist. After that passed, UT-012 failed because a 1025-byte subject URI entered provider data.
- Implement: Added typed `RoutingFacts`, the exact version 1 common/category data map, and an ASCII 1024-byte routing-fact guard. Updated the Firebase sender call site to the new builder while retaining the delivery lifecycle for later integration tests.
- Run command: `cd appview && go test ./internal/push -run 'TestBuildPayloadUT01(1|2)' -count=1`
- Refactor: Ran `gofmt` and reran both focused tests while green.
- Notes: Passed 2 focused tests. The maximum two-URI reply data is below 4096 bytes; over-bound values are omitted so they cannot create an unbounded provider map. The test required shared Go build-cache access after the sandbox denied the cache path.

### Step 7: IT-001

- Write failing test: Added a real-Postgres dispatcher test with distinct actor/source/subject facts and assertions for unchanged token/platform/TTL/binding/copy inputs.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/push -run TestDispatcherIT001ProjectsCanonicalRoutingFacts -count=1`
- Confirmed failure: Yes. The first configured run failed because the documented local Postgres stack was stopped; after starting `just dev-d`, the meaningful red result showed all three `RoutingFacts` fields were empty.
- Implement: Extended the existing claim projection with typed actor/source and nullable subject values, then populated `SendRequest.RoutingFacts` without changing queue, lease, token, platform, TTL, or visible-copy inputs.
- Run command: Same focused real-Postgres command.
- Refactor: Ran `gofmt` and kept the canonical-fact projection inside the existing claim/send seam.
- Notes: Passed 1 real-Postgres test. The detached repository stack is now running to support subsequent integration tests.

### Step 8: IT-002, REG-001

- Write failing test: Updated the Firebase sender test to assert the exact serialized version 1 message and structurally reject `NotificationID` on `SendRequest`.
- Run command: `cd appview && go test ./internal/push -run TestFirebaseSenderBuildsCombinedMessageWithBoundedTTL -count=1`
- Confirmed failure: Yes. The provider request still exposed a `NotificationID` field.
- Implement: Removed provider-facing notification ID from `SendRequest` and dispatcher construction, kept typed routing facts, and updated the existing observability test call site to the new payload builder.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/push -run 'Test(FirebaseSenderBuildsCombinedMessageWithBoundedTTL|BuildPayloadUT01(1|2)|DispatcherIT001ProjectsCanonicalRoutingFacts)' -count=1`
- Refactor: Ran `gofmt` and reran the producer, serializer, and dispatcher contract tests together.
- Notes: Passed 4 focused tests. Serialized data is exact and bounded; visible copy, token, TTL, APNs expiration, and sound remain unchanged.

### Step 9: UT-007, IT-003, AT-001

- Write failing test: Replaced the HTTP-resolver flow test with a provider-neutral runtime test for direct post navigation, matching-bound legacy fallback, and stale-binding rejection.
- Run command: `cd app && flutter test test/notifications/notification_open_flow_test.dart`
- Confirmed failure: Yes. The runtime and service interface still required the old event shape and resolution repository, and the effect model still carried resolution outcomes.
- Implement: Migrated the runtime/service seam to structured attempts and inferred open outcomes, removed the resolution dependency and failure policy from the runtime, and emitted navigation immediately after the binding-first coordinator returns.
- Run command: Same focused Flutter test.
- Refactor: Removed Dio/AppView resolver fakes from the test and formatted the runtime/effect/service files while green.
- Notes: Passed 1 test. Direct and legacy-fallback outcomes emit without any notification-specific or destination network dependency; stale bindings emit only unavailable feedback.

### Step 10: IT-004, AT-004, REG-010

- Write failing test: Replaced the lifecycle test with initial/background/duplicate/foreground-attempt coverage, then tightened it to require `ForegroundNotificationEvent.openAttempt`.
- Run command: `cd app && flutter test test/notifications/services/notification_runtime_lifecycle_test.dart`
- Confirmed failure: Yes. The tightened callback contract failed because foreground events still exposed the obsolete `openEvent` constructor/property.
- Implement: Migrated foreground events, Firebase background/initial parsing, and the root effect-host banner action to `NotificationOpenAttempt`; malformed provider facts are no longer filtered before the binding policy. Updated runtime provider wiring and effect-host fakes to the resolver-free runtime.
- Run command: `cd app && flutter test test/notifications/services/notification_runtime_lifecycle_test.dart test/notifications/notification_effect_host_test.dart`
- Refactor: Kept one runtime owner/root host and added only a temporary named legacy navigation bridge so generic rows can remain untouched until their ordered resolver-removal step.
- Notes: Passed 4 tests. Initial, background, duplicate, and foreground attempts use the same outcome path; duplicate callbacks retain at-least-once behavior with no dedupe storage.

### Step 11: UT-014, AT-005, IT-009

- Write failing test: Added a pure typed-route construction test for reply subject path and source focus query.
- Run command: `cd app && flutter test test/router/notification_open_routing_test.dart`
- Confirmed failure: Yes. The navigation layer had no typed route-construction seam and its existing post branch omitted focus.
- Implement: Added `postThreadRouteForNotification`, which parses only the validated subject post URI into `PostThreadRoute` and carries optional source focus; navigation uses that typed route and safely falls back if parsing fails.
- Run command: `cd app && flutter test test/router/notification_open_routing_test.dart test/feed/pages/post_comment_section_page_test.dart`
- Refactor: Replaced a brittle full-router harness with a direct typed-route test, then ran the existing focused-comment integration suite.
- Notes: The new route test passed. The nearby comment-section suite had 19 passes and one unrelated existing failure in `wires repost action for the root post`; rerunning that test alone reproduced the failure (`repostCalls` stayed empty), and no files in that repost path were changed by this slice.

### Step 12: UT-009

- Write failing test: Added a pure matrix for named 404s, network, representative 500, 502 `identity_unavailable`, 401, and unexpected failures.
- Run command: `cd app && flutter test test/shared/errors/notification_destination_error_test.dart`
- Confirmed failure: Yes. The classifier module and three-way error kind did not exist.
- Implement: Added identifier-free destination error classification: only named 404 post/profile misses are permanent, 401 is authentication loss, and all transient/unexpected failures are retryable.
- Run command: Same focused test.
- Refactor: Kept the classifier pure and exhaustive over the observable classes without stringifying errors or route identifiers.
- Notes: Passed 1 test. The classifier creates no retry scheduler or persisted open state.

### Step 13: IT-005

- Write failing test: Not added; the approved observable behavior was already covered at handler and real-Postgres store levels.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'Test(PostStore_ReadOne_(NotFound|HiddenPostOrHiddenAuthorReturnsNotFound)|ProfileStore_ReadByDID_Hidden.*|GetPost_NotFound_404|GetProfile_NonMember)$' -count=1`
- Confirmed failure: Not applicable; this was the coding plan's verify-or-extend step and no gap remained.
- Implement: None.
- Run command: Same focused real-Postgres/handler command.
- Refactor: None.
- Notes: Passed. Direct post/profile identifiers still resolve through authenticated AppView handlers/stores; deleted, hidden, and taken-down records return named not-found outcomes rather than content.

### Step 14: AT-006, IT-006

- Write failing test: Added a post-thread widget test for a named `404 post_not_found` response before adding the shared destination state.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'notification destination 404 shows permanent recovery actions'` (the test was then moved to its planned dedicated suite).
- Confirmed failure: Yes. The page rendered no permanent-unavailable title and exposed only its old generic Retry action.
- Implement: Added the shared localized `NotificationDestinationErrorState`; wired post/profile named 404s to permanent Back/View notifications actions; kept the destination scaffold visible; and made a permanent post error override cached `_lastSection` content.
- Run command: `flutter test test/feed/pages/post_thread_page_test.dart test/profile/profile_page_test.dart`; focused stale-refresh and route-action tests also passed independently.
- Refactor: Replaced Riverpod runtime-subclass matching with `AsyncValue.error` inspection so permanent refresh failures with retained previous data are classified correctly.
- Notes: Passed 21 focused tests. Permanent states remain on the intended route, hide stale post content, expose accessible labeled recovery actions, and exercise both the current back stack and typed notifications route. The pre-existing unrelated repost-action failure remains isolated to the older comment-section suite.

### Step 15: AT-007, IT-007, REG-006

- Write failing test: Added post and profile transient-failure tests that disable Riverpod's automatic retry, tap the localized Retry action, and require the destination content to load on exactly the second repository call; added a page-level 401 assertion alongside the existing interceptor regression.
- Run command: Focused single-test runs for each new destination scenario, followed by the combined destination/classifier/interceptor suite.
- Confirmed failure: No separate production red remained after Step 14 introduced the approved shared state with both permanent and transient branches. One profile assertion was narrowed after it detected an unrelated Retry button below the successfully loaded profile rather than a lingering notification-destination state.
- Implement: The Step 14 shared state already classified network/`5xx`/`502` failures as retryable in place and rendered nothing for `ApiUnauthorized`; existing `SignOutOn401Interceptor` remained the global sign-out owner.
- Run command: `flutter test test/feed/pages/post_thread_page_test.dart test/profile/profile_page_test.dart test/shared/errors/notification_destination_error_test.dart test/shared/api/providers/sign_out_on_401_interceptor_test.dart`.
- Refactor: Used a shared post-section fixture in the dedicated page suite and scoped the successful profile assertion to `NotificationDestinationErrorState` so unrelated tab retry affordances do not create a false failure.
- Notes: Passed 26 focused tests. Retry invalidates only the current destination provider, no notification-open retry is scheduled, and 401 produces no notification-specific copy or action while the interceptor signs out and clears storage.

### Step 16: UT-013, AT-009, IT-011, REG-005

- Write failing test: Replaced the legacy generic-resolution expectation with generic and unknown-category rows whose `ListTile.onTap` must be null; retained a recording resolver and route/messenger checks to catch side effects.
- Run command: `flutter test test/notifications/notifications_page_test.dart --plain-name 'AT-009 generic and unknown rows are inert while tombstones warn'`.
- Confirmed failure: Yes. Both informational tiles still exposed non-null tap callbacks.
- Implement: Made `NotificationRow` provider-neutral and assigns no `onTap` to `GenericNotification`; removed its generic resolver branch while preserving typed known-row navigation and the unavailable-row warning.
- Run command: `flutter test test/notifications/notifications_page_test.dart`.
- Refactor: Converted `NotificationRow` from `ConsumerWidget` to `StatelessWidget` and removed now-unused resolution imports/async method.
- Notes: Passed all 5 page tests. Generic and unknown rows are inert with no resolution/navigation/messenger side effects; known rows still navigate and unavailable rows still warn.

### Step 17: IT-008, REG-008

- Write failing test: Added a Flutter architecture assertion requiring all resolution-only files/types/providers to be absent, then added an AppView route-registry assertion requiring the former notification-ID path to fall through.
- Run command: Focused `notification_architecture_test.dart`; then `go test ./internal/routes -run TestAddRoutes_NotificationResolutionRouteIsRemoved -count=1`.
- Confirmed failure: Yes. Flutter still contained the notification-ID model first; after the client removal, AppView still matched `GET /v1/notifications/{notificationId}`.
- Implement: Deleted the Flutter resolution ID/model/policy, repository interface/call/provider, legacy event/navigation bridge, and obsolete tests; regenerated the provider output. Deleted the AppView handler/store and tests, route registration, and route policy. Updated remaining service fakes and redaction fixtures to the structured attempt type.
- Run command: Combined Flutter architecture/repository/page/registration/secret-scan suite (22 tests); focused AppView route and policy tests.
- Refactor: Removed unrelated generator drift from files outside this change, leaving only the intended generated provider deletion.
- Notes: Clean cutover is complete. Only structural test sentinel strings mention the removed symbols; no resolver producer, consumer, model, policy, provider, handler, store, or registered path remains.

### Step 18: UT-010, IT-010, REG-007

- Write failing test: Ran the existing real registration/enqueue/retry/success privacy integration after adding routing facts; expanded Flutter parser, sanitizer, and stringification sentinels for binding, DID, subject/focus URIs, raw payload, provider error, title, and body.
- Run command: Real-Postgres `go test ./internal/observability -run TestPushPrivacySentinelsAcrossRegistrationEnqueueDispatchAndTelemetry -count=1`; focused Flutter parser/Sentry/architecture/secret-scan suite.
- Confirmed failure: Yes. The AppView test treated the intentionally public provider `subjectUri` as if it were telemetry and reported the new routing fact as a leak.
- Implement: Separated the required provider-boundary payload assertions from logs/Sentry/metrics assertions. Switched the integration fixture to reply so distinct source/focus and subject URI facts are both covered; verified provider data contains only required public routing facts while telemetry excludes them, the generated binding, token, private content, raw payload, credentials, and provider-error sentinels.
- Run command: Updated real-Postgres integration passed; all 17 focused Flutter privacy/architecture tests passed.
- Refactor: Documented the privacy boundary in the integration test: registration responses/provider data intentionally carry binding/public facts, but observability must never copy them.
- Notes: Diagnostics expose only bounded category/result/classification values. Parser and attempt stringification, Sentry sanitizer output, logs, metrics, and Sentry evidence contain no identifiers or raw payloads.

### Step 19: UT-015

- Write failing test: Added a structural import scan over the new parser, destination, and inference seams plus an exact inventory of Firebase Messaging importers.
- Run command: `flutter test test/notifications/notification_architecture_test.dart --plain-name 'UT-015 routing domain stays provider, UI, and network neutral'`.
- Confirmed failure: Yes. The new destination value model imported Flutter solely for `@immutable`; the first Firebase inventory expectation also exposed two existing adapter-layer bootstrap/background files.
- Implement: Removed the unnecessary Flutter dependency/annotations from the final immutable-by-construction destination classes. Defined the existing Firebase service, background handler, and bootstrap as the complete adapter-layer importer set.
- Run command: `flutter test test/notifications/notification_architecture_test.dart`.
- Refactor: Sorted the structural importer inventory for deterministic results.
- Notes: Passed all 6 architecture tests. Parsing/inference/domain files import no Firebase, Flutter UI, Dio, repositories, providers, or widgets; Firebase remains confined to its three adapter-layer files.

### Step 20: REG-002

- Write failing test: Existing regression suite; no new behavior expected.
- Run command: Existing dispatcher/retry/Firebase sender suites.
- Confirmed failure: No.
- Implement: Only approved regression fixes caused by this change.
- Run command: Real-Postgres `go test ./internal/push -count=1`.
- Refactor: None.
- Notes: Passed. Claiming, fencing, cancellation, invalid-token cleanup, TTL, provider retry, and sender semantics remain green with canonical routing facts.

### Step 21: REG-003

- Write failing test: Existing regression suite; no new behavior expected.
- Run command: Existing notification lifecycle/index suites.
- Confirmed failure: No.
- Implement: Only approved regression fixes caused by this change.
- Run command: Real-Postgres `go test ./internal/notifications -count=1` and `go test ./internal/index -run Notification -count=1`.
- Refactor: None.
- Notes: Passed. Eligibility, preferences, coalescing, index behavior, and preference snapshots remain unchanged.

### Step 22: REG-004

- Write failing test: Existing regression suite; no new behavior expected.
- Run command: Existing AppView and Flutter list/newness/seen/badge suites.
- Confirmed failure: No.
- Implement: Only approved regression fixes caused by this change.
- Run command: Real-Postgres focused AppView list/new-count/seen tests; Flutter notifications-page, seen-flow, and app-shell badge suites.
- Refactor: None.
- Notes: Passed the focused AppView package tests and all 8 Flutter widget/flow tests. List rendering, new-count, seen-after-success, sound contract, and accessible capped badge behavior remain unchanged.

### Step 23: MAN-001–MAN-005

- Write failing test: Not automated; physical-device and real-provider checks.
- Run command: Manual Android/iOS delivery and debug request trace after automated gates are green.
- Confirmed failure: Not run in host implementation stage.
- Implement: No implementation outside approved automated findings.
- Run command: Pending release-readiness environment and physical devices.
- Refactor: Not applicable.
- Notes: Not run. Real iOS/Android delivery, stale-target interaction, retained multi-account OS notification behavior, and a terminated-app debug request trace require configured provider credentials/physical devices. These remain explicit release gates rather than host-test claims.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [ ] Review completed or explicitly skipped

## Final Verification

- `dart analyze` over the changed notification/destination/profile/router surfaces: passed with no issues.
- `just app-analyze`: passed with no issues after rerunning with shared FVM-cache access.
- `just test`: passed the canonical race-enabled AppView suite across all packages.
- Real-Postgres focused AppView push, route, API, lifecycle, index, list/newness/seen, and privacy suites: passed.
- Focused Flutter notification, destination, profile, router, privacy, architecture, seen, and badge tests: passed.
- Broad feature command completed 126 tests with one unrelated failure: `post_comment_section_page_test.dart` — `wires repost action for the root post`. The same failure reproduced before the destination-page implementation and remains outside every requirement/test ID in this workflow.
- Canonical `just app-test` / full `flutter test` likewise completed 858 tests with that same single unrelated failure and no additional failures.
- `git diff --check`: passed.
- Generated Riverpod and localization outputs were regenerated. Unrelated generator-version drift was removed from the worktree.
- No commit or push was created because stage commits were not enabled.

## Execution Notes

- 2026-07-17: The user explicitly invoked `implement-tdd`, satisfying CPQ-001. No workflow document has a blocking gap or Changes required verdict.
- 2026-07-17: Stage commits are not enabled. The current worktree will be preserved and only approved files will be edited.
- 2026-07-17: `06-implementation-review.md` returned `Changes required` with IR-001 through IR-003. The user explicitly selected Address required changes, authorizing a focused TDD correction pass including the account-isolation race.

## Implementation Review Correction Pass

| Step | Finding / Test | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 24 | IR-001 / profile permanent-refresh acceptance | FR-009, RULE-005 | AC-009 | Fails because a retained profile value wins over the refresh error |
| 25 | IR-002 / post and profile transient-refresh acceptance | FR-010, NFR-005 | AC-010 | Fails because retained destination content suppresses Retry |
| 26 | IR-003 / in-flight account-transition runtime test | BR-002, FR-006, RULE-003 | AC-006 | Fails because an old binding read can emit after readiness changes |

### Step 24: IR-001 / FR-009

- Write failing test: Added a profile refresh acceptance test that first renders a sentinel display name, invalidates `userProfileProvider`, then returns named `404 profile_not_found` and requires permanent UI with the sentinel absent.
- Run command: `cd app && flutter test test/profile/profile_page_test.dart --plain-name 'ProfilePage permanent refresh error hides previously loaded profile'`
- Confirmed failure: Yes. The permanent title was absent because `_ProfileScaffold` selected the retained non-null value before examining the refresh error.
- Implement: Gave named permanent destination errors precedence over retained profile values, matching the protected post-thread behavior.
- Run command: Same focused command.
- Refactor: Deferred shared error-scaffold extraction until the transient-refresh loop is green.
- Notes: Passed 1 focused test. A permanent profile refresh now hides previously authenticated cached profile content.

### Step 25: IR-002 / FR-010

- Write failing test: Added post and profile refresh-after-success acceptance tests requiring retained authenticated content plus Retry inside the shared destination error state.
- Run command: Focused plain-name commands for `transient refresh error keeps destination Retry available` and `ProfilePage transient profile refresh keeps destination Retry available`.
- Confirmed failure: Yes. The post test found no Retry. The first profile assertion was a false green caused by an unrelated tab Retry; scoping it beneath `NotificationDestinationErrorState` produced the meaningful red failure.
- Implement: When a retryable refresh error retains post/profile data, render the shared localized Retry state above that authenticated content. Retry continues to invalidate only the current destination provider; permanent errors still replace cached content.
- Run command: Both focused commands, followed by `cd app && flutter test test/feed/pages/post_thread_page_test.dart test/profile/profile_page_test.dart`.
- Refactor: Extracted the duplicated profile destination-error callbacks into one helper and kept the ordinary successful profile layout unchanged.
- Notes: Passed all 26 combined destination page tests, including initial, permanent refresh, transient refresh, recovery actions, and 401 behavior.

### Step 26: IR-003 / FR-006

- Write failing test: Added one public-runtime test with a blocked secure-routing read. It covers both readiness changing to `requiresSignIn` and switching from Alice to Bob before the old Alice binding read completes.
- Run command: `cd app && flutter test test/notifications/notification_open_flow_test.dart --plain-name 'IR-003 discards an in-flight open across account readiness changes'`
- Confirmed failure: Yes. The sign-in-required case received a `NotificationNavigationEffect` from the old matching binding after readiness changed.
- Implement: Added a monotonically increasing readiness revision. Each open captures its revision, and navigation/unavailable effects are emitted only if the runtime is still undisposed, ready, on the same DID, and on the same revision after asynchronous binding work. This also rejects an away-and-back transition to the same DID.
- Run command: Same focused command, followed by the open-flow, runtime-lifecycle, coordinator, pending-open, and effect-host suites.
- Refactor: Centralized the full current-open predicate in `_isCurrentOpen` so both effect classes use the same account/readiness rule.
- Notes: Passed the focused race test and all 8 nearby runtime/coordinator/effect tests. No cancellation queue, persistence, or dedupe state was added.

## Correction Pass Final Verification

- IR-001 profile permanent-refresh test: passed after a meaningful red failure.
- IR-002 post/profile transient-refresh tests: passed after meaningful red failures; the profile test was first corrected to exclude an unrelated tab-level Retry false positive.
- IR-003 sign-out/account-switch in-flight runtime test: passed after a meaningful red navigation effect.
- Combined post/profile destination suites: passed all 26 tests.
- Combined runtime/open-flow/coordinator/pending/effect suites: passed all 8 tests.
- Broader notification/destination/router command: passed all 111 tests.
- `just app-analyze`: passed with no issues after the shared FVM-cache permission rerun and one mechanical test-field lint correction.
- `just app-test`: completed 862 tests with the same single unrelated pre-existing failure in `post_comment_section_page_test.dart` — `wires repost action for the root post`; no additional failure appeared.
- AppView tests were not rerun during the correction pass because IR-001 through IR-003 changed only Flutter source/tests and the implementation record; the original correction input already records canonical `just test` success for the unchanged AppView diff.
- MAN-001 through MAN-005 remain physical-device/provider release gates and were not run.
- No commit or push was created because stage commits remain disabled.

## Manual Device Finding Correction Pass

| Step | Finding / Test | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 27 | BUG-001 / root effect-host router-context test | FR-007, FR-013 | AC-001, AC-013 | Fails because the `MaterialApp.router.builder` context is above GoRouter |
| 28 | BUG-002 / liked-comment root-thread routing tests | FR-002, FR-003, FR-007, FR-017 | AC-003, AC-007, AC-016 | Fails because a comment `subjectUri` is used as the root thread |

### Step 27: BUG-001 / FR-007

- Write failing test: Mounted the root effect host through `MaterialApp.router.builder`, emitted a post navigation effect, and required the thread route to render.
- Run command: `cd app && flutter test test/notifications/notification_effect_host_test.dart --plain-name 'BUG-001 root effect host navigates from MaterialApp.router builder'`.
- Confirmed failure: Yes. The test raised `No GoRouter found in context` through the same `NotificationEffectHost` and generated `PostThreadRoute.push` stack as the physical Android background-open report.
- Implement: Pass the provider-owned `GoRouter` instance to notification navigation and call `router.go` / `router.push` with typed route locations; retain the widget context only for localized feedback.
- Run command: Same focused command.
- Refactor: Kept typed route construction in `notification_navigation.dart`; no second navigator key or listener was added.
- Notes: Passed. Root effects no longer depend on the builder context being below GoRouter's inherited scope.

### Step 28: BUG-002 / FR-003

- Write failing test: Added a notification-row widget test for a liked comment, a provider-neutral fact inference test, an interaction-index test for durable canonical root storage, a dispatcher projection assertion, and an exact provider payload assertion.
- Run command: Focused plain-name Flutter commands for the row and inference tests; real-Postgres focused Go commands for `TestInteractionNotificationStoresCommentRoot` and `TestDispatcherIT001ProjectsCanonicalRoutingFacts`; focused `TestBuildPayloadUT011ExactNotificationFactMatrix`.
- Confirmed failure: Yes. The row routed to the comment author's DID/rkey instead of the root; inference returned the subject destination instead of root plus focus; the durable `root_uri` was `NULL`; the dispatcher request lacked `RootURI`; and like/repost payload data lacked `rootUri`.
- Implement: In-app rows derive the canonical thread root from the hydrated subject post and focus a differing comment. Interaction activation stores `COALESCE(reply_root_uri, uri)` and its CID. Dispatch carries that root as a typed routing fact, like/repost provider data requires bounded `subjectUri` plus `rootUri`, and Flutter routes to the root while focusing a differing subject.
- Run command: All focused commands above, followed by combined parser/inference/open-flow/effect/list/router Flutter suites and complete AppView push/index packages.
- Refactor: Centralized hydrated-post thread routing in `NotificationRow._openPost`; kept payload inference pure and preserved the existing single root effect host and authenticated destination read.
- Notes: Passed. Live AppView logs showed the failing comments request returning `400`, and read-only database inspection proved the selected notification subject was a comment whose `reply_root_uri` pointed to another post. No migration or backfill was added; newly generated like/repost pushes use the corrected clean-cutover contract, while existing in-app rows route correctly from hydrated post data.

## Manual Device Finding Final Verification

- BUG-001 reproduced the physical Android `No GoRouter found in context` stack, then passed after navigation used the provider-owned router instance.
- BUG-002 row, parser/inference, interaction-index, dispatcher, and exact-payload tests each produced a meaningful red before their minimum implementation and now pass.
- Combined Flutter notification/open/list/router set: passed 22 tests.
- Complete Flutter notification suite plus notification router test: passed 77 tests.
- Broader notification/destination/router feature gate: passed 114 tests.
- Complete AppView push and index packages: passed.
- Canonical race-enabled `just test`: passed across every AppView package.
- Canonical `just app-analyze`: passed with no issues.
- Canonical `just app-test`: completed 865 tests with only the same unrelated pre-existing failure in `post_comment_section_page_test.dart` — `wires repost action for the root post`; no new failure appeared.
- `git diff --check`: passed.
- The running AppView logs and development database were inspected read-only. No Flutter terminal session was attached to this Codex task, so Flutter runtime evidence came from the user-provided logs and the exact widget reproduction.
- Fresh physical background/terminated opens and a newly generated like/repost-on-comment notification remain manual verification gates after rebuilding AppView and the Flutter app. Previously delivered version-1 like/repost payloads do not contain the newly required `rootUri` and intentionally use the clean-cutover fallback.
- No commit or push was created because stage commits remain disabled.

## Notification Language Correction Pass

| Step | Test | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 29 | UT-016 | FR-020 | AC-023 | Fails because in-app rows hard-code post wording and call every response a reply to a post |
| 30 | UT-017 | FR-020 | AC-023 | Fails because OS-visible copy is selected only from category and hard-codes post wording |
| 31 | IT-012 | FR-020 | AC-023 | Fails because the dispatcher does not project indexed target role into the send request |

### Step 29: UT-016 / FR-020

- Write failing test: Added a widget matrix with like, repost, and response rows targeting a root post, direct comment, and nested reply.
- Run command: `cd app && flutter test test/notifications/notifications_page_test.dart --plain-name 'UT-016 uses post, comment, and reply language in rows'`.
- Confirmed failure: Yes. The test found three `Alice liked your post` rows because all subject roles used the same hard-coded copy.
- Implement: Classified each hydrated subject from its reply root/parent structure and selected localized post/comment/reply copy. A response to a root now says `commented on your post`; deeper responses say `replied to your comment` or `replied to your reply`.
- Run command: Regenerated localization output, formatted the touched Dart files, and reran the same focused test.
- Refactor: Kept the role classifier private to the row presentation boundary and reused the existing `Post.reply` model; no notification API field was added.
- Notes: Passed 1 focused widget test after the meaningful red failure. Neutral mention copy and root-only quote copy remain unchanged.

### Step 30: UT-017 / FR-020

- Write failing test: Added a table-driven AppView payload test covering like, repost, reply/comment, and quote bodies for post, comment, and reply targets while comparing each data map to its role-neutral baseline.
- Run command: `cd appview && go test ./internal/push -run TestBuildPayloadUT017UsesConversationRoleInVisibleCopy -count=1`.
- Confirmed failure: Yes. The test did not compile because no bounded content-role type or payload input existed.
- Implement: Added the internal `ContentRole` enum to push facts and selected content-free visible bodies from category plus role. Unknown/absent roles retain the root-post fallback; mentions, follows, and generic activity remain neutral.
- Run command: Formatted the Go files and reran the focused test with shared Go build-cache access.
- Refactor: Centralized visible action selection in `visibleBody`; the role is not serialized into provider data.
- Notes: Passed the 12-case copy matrix. Exact provider data remained unchanged for every role.

### Step 31: IT-012 / FR-020

- Write failing test: Added a real-Postgres dispatcher matrix for root posts, direct comments, nested replies, and a quote whose copy target comes from `quoted_uri`.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/push -run TestDispatcherIT012ProjectsTargetContentRole -count=1 -v`.
- Confirmed failure: Yes. An initial run without the test database was correctly identified as skipped rather than counted as red. With the real database enabled, all four cases received an empty role instead of post/comment/reply.
- Implement: The existing claim query now joins the category-specific target post (`quoted_uri` for quote, otherwise `subject_uri`) and classifies its indexed reply structure into the bounded internal role passed to `BuildPayload`.
- Run command: Formatted the dispatcher and reran the same real-Postgres test.
- Refactor: Kept role projection inside the existing claim/send path; no migration, payload key, provider content, notification-list API field, or extra lookup request was added.
- Notes: Passed all 4 real-Postgres cases after the meaningful red failure.

## Notification Language Correction Final Verification

- UT-016 in-app row matrix: passed after a meaningful red showing all three like targets rendered as `liked your post`.
- UT-017 OS-visible payload matrix: passed all 12 category/role cases after the missing bounded-role compile failure; every role produced the same provider data map as its baseline.
- IT-012 dispatcher projection: passed all 4 real-Postgres cases after the empty-role red failure. The first no-database run was explicitly treated as skipped evidence, not as a pass.
- Complete Flutter notification suite: passed all 77 tests.
- Complete AppView push package: passed after updating the two intentional minimal-schema fixtures to include the newly read `craftsky_posts` table.
- Canonical race-enabled `just test`: passed across every AppView package.
- Canonical `just app-analyze`: passed with no issues.
- Canonical `just app-test`: completed 866 tests with only the same unrelated pre-existing failure in `post_comment_section_page_test.dart` — `wires repost action for the root post`; no new failure appeared.
- `git diff --check`: passed.
- A fresh physical OS notification remains the final visual/device check after rebuilding AppView and Flutter. Previously delivered notifications retain their already-rendered OS copy.
- No commit or push was created because stage commits remain disabled.

## Notification Row Context Follow-Up

| Step | Test | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 32 | UT-018 | FR-021 | AC-024 | Fails because rows render no actor avatar, category icon, or relative timestamp |
| 33 | UT-020 | FR-021 | AC-024 | Fails because the notification actor response has no display-ready avatar URL |

### Step 32: UT-018 / FR-021

- Write failing test: Rendered follow, like, repost, reply, mention, quote, and generic rows and required seven shared profile avatars, the exact category icon matrix, seven compact relative timestamps, and full timestamp tooltips.
- Run command: `cd app && flutter test test/notifications/notifications_page_test.dart --plain-name 'UT-018 rows show actor avatars, action icons, and relative time'`.
- Confirmed failure: Yes. The widget test found zero `ProfileAvatar` widgets.
- Implement: Added a bounded category-to-icon presentation, rendered the existing small `ProfileAvatar`, decoded the actor's display-ready avatar URL, and placed the notification creation time beside the row copy.
- Refactor: Extracted the post card's compact `now`/minute/hour/day timestamp and localized full tooltip into `RelativeTimeText`, then reused it on posts and notification rows.
- Notes: Passed the focused widget matrix and the combined notification-page/post-card suite after adding the standard Craftsky theme and Riverpod scope to the affected test harnesses.

### Step 33: UT-020 / FR-021

- Write failing test: Required the existing notifications handler to return the canonical CDN avatar URL from an indexed actor avatar CID and MIME.
- Run command: `cd appview && go test ./internal/api -run TestNotificationsHandlerUT020IncludesDisplayReadyActorAvatar -count=1`.
- Confirmed failure: Yes. The test failed to compile because `NotificationRow` had no actor avatar MIME and `NotificationActor` had no display-ready `Avatar` field.
- Implement: Selected actor avatar MIME in the existing list query and synthesized the additive `actor.avatar` response with the same canonical helper used by profile/post responses. Unavailable actors clear both avatar URL and CID.
- Refactor: Kept avatar hydration inside the existing notification list request; no per-row API call, migration, endpoint, or client-side CDN assumption was added.
- Notes: Passed the focused handler test and the complete AppView API package.

## Notification Row Context Final Verification

- UT-018 widget matrix: passed after the meaningful zero-avatar red failure.
- UT-020 AppView response: passed after the missing-field compile failure.
- Combined notification-page and post-card widget suites: passed all 52 tests.
- Complete Flutter notification suite: passed all 78 tests after updating one intentional page test harness with the real Craftsky theme.
- Complete AppView API package: passed.
- Canonical race-enabled `just test`: passed across every AppView package.
- Canonical `just app-analyze`: passed with no issues.
- Canonical `just app-test`: completed 867 tests with only the same unrelated pre-existing failure in `post_comment_section_page_test.dart` — `wires repost action for the root post`; no new failure appeared.
- Physical-device visual inspection remains a manual gate after rebuilding AppView and Flutter so the additive actor avatar URL is present.
- No commit or push was created because stage commits remain disabled.

## Notification Row Visual Correction Pass

| Step | Test | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 34 | UT-018 | FR-021 | AC-024 | Fails because rows use filled icons, place the avatar beside the copy, and do not bold the actor name |
| 35 | UT-021 | FR-021 | AC-024 | Fails because responses without the additive actor `avatar` URL render only the initial even when `avatarCid` is present |

### Step 34: UT-018 / FR-021

- Write failing test: Required each category to use the outlined notification-settings icon, required the profile avatar to be vertically above the actor copy, and required the actor span to be bold.
- Run command: `cd app && flutter test test/notifications/notifications_page_test.dart --plain-name 'UT-018 rows show actor avatars, action icons, and relative time'`.
- Confirmed failure: Yes. After increasing the test viewport to expose the complete matrix, the first missing expectation was the outlined follow icon.
- Implement: Replaced the list-tile layout with an accessible ink row whose content column places `ProfileAvatar` above the rich notification copy, then bolded the actor substring without assuming a localization placeholder order.
- Refactor: Extracted `notificationCategoryIcon` and reused its outlined icon matrix from both notification settings and rows; retained category colors and inert generic-row behavior.

### Step 35: UT-021 / FR-021

- Write failing test: Decode notification actor JSON containing only DID plus a public avatar CID and require a canonical display avatar URL.
- Run command: `cd app && flutter test test/notifications/notifications_page_test.dart --plain-name 'UT-021 derives an actor avatar URL from an older CID response'`.
- Confirmed failure: Yes. The decoded actor's display avatar URL was `null`.
- Implement: Added `displayAvatarUrl`, which prefers the additive AppView URL and otherwise derives the canonical public CDN URL from the actor DID and avatar CID.
- Refactor: Development-media CIDs deliberately remain on the initial fallback because their base URL cannot be reconstructed safely; a rebuilt AppView supplies their display-ready URL.
- Runtime evidence: Recent local notification actors had non-empty avatar CID/MIME values in Postgres, and a synthesized canonical CDN URL returned HTTP 200 with `image/jpeg`. The running AppView container predated the additive `actor.avatar` response, explaining the observed initial-only rendering.

## Notification Row Visual Correction Final Verification

- UT-018 outlined-icon/layout/bold-name matrix: passed after the meaningful missing-outlined-icon failure.
- UT-021 older-response avatar compatibility: passed after the meaningful null-avatar failure, including the guarded development-media case.
- Complete Flutter notification suite: passed all 79 tests.
- Canonical `just app-analyze`: passed with no issues.
- `dart format`: all five touched Dart files were already formatted.
- `git diff --check`: passed.
- A Flutter hot restart or rebuild is required to load the new row widget. Rebuilding the local AppView is still recommended so its additive display-ready `actor.avatar` field covers development-media avatars; public CID avatars work with the compatibility fallback immediately.
- No commit or push was created because stage commits remain disabled.

## Follow Notification Action Follow-Up

| Step | Test | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 36 | UT-022, IT-014 | FR-022 | AC-025 | Fails because neither the notification store row nor actor response exposes the viewer-to-actor follow relationship |
| 37 | UT-023 | FR-022 | AC-025 | Fails because a follow-notification row contains no Follow/Unfollow control |

### Step 36: UT-022 / IT-014 / FR-022

- Write failing test: Required a followed actor row to serialize `viewerIsFollowing=true` in the existing notification actor object.
- Run command: `cd appview && go test ./internal/api -run TestNotificationsHandlerUT022IncludesActorFollowState -count=1`.
- Confirmed failure: Yes. The test failed to compile because `NotificationRow` and `NotificationActor` had no follow-state fields.
- Implement: Added an indexed `EXISTS` projection against `atproto_follows` scoped by the authenticated viewer DID, scanned it with each notification row, and emitted the additive camelCase boolean.
- Refactor: Kept relationship hydration inside the existing paginated notification query; no endpoint, migration, or per-row Flutter fetch was introduced.
- Integration evidence: The real-Postgres pagination test seeds viewer-to-actor follow state and requires it on both pages.

### Step 37: UT-023 / FR-022

- Write failing test: Required a follow row to start at Follow/Unfollow from actor state, toggle through the existing profile repository by actor DID, adopt each returned profile state, and avoid row navigation.
- Run command: `cd app && flutter test test/notifications/notifications_page_test.dart --plain-name 'UT-023 follow notification toggles Follow and Unfollow'`.
- Confirmed failure: Yes. No Follow button was present.
- Implement: Added a compact `ChunkyButton` only for available follow rows, with optimistic local state, disabled in-flight behavior, authoritative response adoption, localized rollback feedback, and invalidation of DID- and handle-keyed profile caches.
- Refactor: Reused existing profile Follow/Unfollow labels and repository mutations; nested button gesture handling prevents the containing row from opening the profile.

## Follow Notification Action Final Verification

- UT-022 handler response: passed after the meaningful missing-field compile failure.
- IT-014 real-Postgres relationship projection: passed for both pages of the seeded notification list.
- UT-023 widget mutation and rollback cases: passed after the meaningful missing-button failure.
- Complete Flutter notification suite: passed all 81 tests after enlarging one intentional navigation-test viewport for the taller follow row.
- Canonical race-enabled `just test`: passed across every AppView package.
- Canonical `flutter analyze`: passed with no issues.
- `dart format`, `gofmt`, and `git diff --check`: passed.
- Rebuild both AppView and Flutter before device testing because the button's initial state depends on the additive `actor.viewerIsFollowing` response field.
- No commit or push was created because stage commits remain disabled.
