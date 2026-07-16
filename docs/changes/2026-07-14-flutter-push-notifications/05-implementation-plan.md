# TDD Implementation Plan: Flutter Push Notifications

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with notes
- Coding plan: `04-coding-plan.md`
- Implementation approval: explicit `implement-tdd` invocation on 2026-07-15
- Correction input: `06-implementation-review.md` — Changes required
- Correction approval: explicit `Address required changes` selection on 2026-07-15
- Simplification input: `06-implementation-review.md` — Approved with notes, IR-007–IR-012
- Simplification approval: explicit `Implement all those things` instruction on 2026-07-16

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Keep Firebase types inside the approved adapter/bootstrap/background boundary.
- Do not run physical-device delivery until automated verification is green and the bounded sender gate starts from `PUSH_ENABLED=false`.
- Do not commit or push during this stage unless the user explicitly asks.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 1 | UT-002 | FR-008, FR-009, NFR-001, RULE-003 | AC-007, AC-008, AC-018 | Fails |
| 2 | UT-003, UT-015 | FR-004, FR-009, RULE-003, RULE-006 | AC-004, AC-008 | Fails |
| 3 | UT-004 | FR-010, RULE-003 | AC-009, AC-025 | Fails |
| 4 | UT-001 | FR-003, FR-006, RULE-001 | AC-003 | Fails |
| 5 | UT-013 | FR-005, FR-006, NFR-003 | AC-005 | Fails |
| 6 | IT-001, IT-002, AT-002 | BR-001, FR-003–FR-006, NFR-003, RULE-001, RULE-006 | AC-003–AC-005 | Fails |
| 7 | UT-012, IT-010, AT-009 | FR-002, FR-019, FR-020, NFR-002 | AC-015, AC-016 | Fails |
| 8 | UT-014, AT-011 | FR-022 | AC-026 | Fails |
| 9 | AT-001, REG-001, REG-005, REG-006 | BR-001, FR-001, FR-002, FR-020, FR-023, NFR-002 | AC-001, AC-002, AC-015, AC-016, AC-027 | Fails |
| 10 | UT-016, AT-012 | FR-023, FR-025 | AC-027 | Fails |
| 11 | UT-018, AT-003, IT-007 | BR-002, FR-007, FR-012, NFR-004, RULE-002, RULE-007 | AC-006, AC-010, AC-019, AC-029 | Fails |
| 12 | AT-004, AT-005, IT-004, IT-012 | BR-002, FR-008–FR-010, FR-022, RULE-003, RULE-006 | AC-007–AC-009, AC-025, AC-026 | Fails |
| 13 | UT-005, IT-003, AT-006 | FR-011, FR-012 | AC-010, AC-028, AC-029 | Fails |
| 14 | UT-006, UT-007, IT-008 | BR-003, FR-013, NFR-004 | AC-011, AC-020 | Fails |
| 15 | UT-008, IT-005, AT-007, REG-008 | BR-003, FR-014, RULE-005 | AC-012, AC-021 | Fails |
| 16 | UT-009, UT-011, IT-006 | FR-015, FR-016 | AC-013, AC-014, AC-022, AC-023 | Fails |
| 17 | UT-010 | FR-016 | AC-014 | Fails |
| 18 | AT-008, IT-009 | BR-004, FR-015–FR-018, FR-024, NFR-004, RULE-004, RULE-008 | AC-013, AC-014, AC-022, AC-023 | Fails |
| 19 | UT-019, AT-010, IT-011 | FR-021 | AC-017, AC-024 | Fails |
| 20 | UT-017 | NFR-004 | AC-006, AC-011, AC-013 | Fails |
| 21 | REG-002, REG-003, REG-004, REG-009 | FR-007, FR-010–FR-014, FR-019, FR-021, NFR-001 | AC-005, AC-010, AC-012, AC-015, AC-017–AC-021, AC-025 | Fails |
| 22 | IT-013 | FR-020 | AC-016 | Fails |
| 23 | IT-014, REG-007 | FR-025 | AC-027 | Fails |
| 24 | IT-015, REG-010 | FR-026 | AC-030 | Fails |
| 25 | MAN-001–MAN-005 | FR-001, FR-005, FR-007, FR-008, FR-018, FR-021, FR-023–FR-026 | AC-001, AC-003–AC-007, AC-013, AC-017, AC-024, AC-027, AC-030 | Manual / external prerequisites |

## Implementation Review Correction Order

The correction pass follows `06-implementation-review.md` and keeps the original requirement/test IDs. Each row is a new red-green-refactor loop; passing pre-existing tests are not treated as evidence for the missing behavior.

| Correction Step | Review Finding | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|---|
| C1 | IR-001 | IT-002 | FR-003, FR-004, FR-018 | AC-003, AC-004, AC-013; EC-013 | Fails: resume does not re-read permission |
| C2 | IR-002 | UT-019, AT-010, IT-011 | FR-021 | AC-017, AC-024 | Fails: later same-DID cleanup is skipped |
| C3 | IR-003 | AT-006, IT-007 | FR-011, FR-012 | AC-010, AC-028, AC-029 | Fails: generic/tombstone taps return silently |
| C4 | IR-004 | IT-008, REG-004 | FR-013, RULE-005 | AC-020 | Fails: signed-out resume creates count request |
| C5 | IR-005 | IT-004, IT-012 | FR-008–FR-010, FR-022, RULE-003, RULE-006 | AC-007–AC-009, AC-025, AC-026 | Missing public-boundary coverage |
| C6 | IR-005 | IT-005, REG-008 | FR-014, RULE-005 | AC-012, AC-021 | Missing public-boundary coverage |
| C7 | IR-005 | IT-006 | FR-016 | AC-014, AC-023 | Missing HTTP-contract coverage |
| C8 | IR-005 | IT-007 | FR-007, NFR-004, RULE-002 | AC-006, AC-019 | Missing root presentation coverage |
| C9 | IR-005 | IT-008 | FR-013, NFR-004 | AC-011, AC-020 | Missing shell integration coverage |
| C10 | IR-005 | IT-009 | FR-015–FR-018, FR-024, RULE-004, RULE-008 | AC-013, AC-014, AC-022, AC-023 | Missing typed-route coverage |
| C11 | IR-005 | REG-002 | NFR-001, NFR-002 | AC-018 | Missing notification sentinel coverage |
| C12 | IR-005 | REG-004, REG-009 | FR-007, FR-010, FR-013, FR-019, RULE-007 | AC-015, AC-019, AC-020, AC-025 | Missing structural guards |

## Implementation Steps

Each automated step follows the same loop: add only the named focused test(s), run the smallest command, record the meaningful red result, implement the minimum linked behavior, rerun to green, run nearby regressions, refactor only while green, and record the exact evidence below.

### Approved Simplification Pass Order

This pass changes internal layout only. It keeps the original behavior requirements and public test IDs while implementing IR-007–IR-012 from the approved follow-up review.

| Simplification Step | Review Findings | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|---|
| S1 | IR-008 | IT-003, IT-006 | FR-011, FR-013–FR-016, NFR-002 | AC-010, AC-012–AC-014, AC-023 | Fails: capability providers construct separate forwarding adapters |
| S2 | IR-009 | UT-007, IT-008, REG-004 | FR-013 | AC-020 | Fails: callers require trigger taxonomy for identical refresh work |
| S3 | IR-007, IR-011 | UT-012, UT-018, IT-007, IT-010, REG-009 | FR-002, FR-007, FR-019, NFR-002 | AC-006, AC-015, AC-019 | Fails: runtime does not directly own service lifecycle and foreground effects |
| S4 | IR-010 | UT-001, UT-013, IT-002 | FR-003–FR-006, FR-018 | AC-003–AC-005, AC-013 | Fails: permission and token registration use separate lifecycle coordinators |
| S5 | IR-012 | UT-003, UT-008, UT-016, IT-005, IT-012, AT-012 | FR-009, FR-014, FR-023, RULE-003, RULE-005 | AC-008, AC-012, AC-021, AC-027 | Fails: one-use policies remain separate from their observable owner flows |

- Guardrails: retain the provider-neutral `NotificationService`, Firebase import boundary, exact once-only subscription behavior, latest-token and DID fencing, secure routing storage, AppView-only resolution authority, pending-open readiness behavior, seen-after-render behavior, and silent foreground presentation.
- Verification cadence: run the smallest focused test for each red-green loop, then the complete notification suite after each consolidation. Run linked auth/router/observability and canonical checks after S5.
- S1 — IR-008 / IT-003 / IT-006: added a provider-container test requiring all five narrow repository providers to expose one shared HTTP adapter. Red failed because `notificationApiRepositoryProvider` did not exist and each capability constructed a separate forwarding wrapper. Moved the Dio route implementations directly into `ApiNotificationRepository`, removed `NotificationApiClient`, and retained every narrow interface/provider override while sharing one adapter instance. The focused provider/API/device/preferences run passed 6 tests.
- S2 — IR-009 / UT-007 / IT-008 / REG-004: replaced the trigger-classification unit test with a provider test requiring one fetch for each explicit refresh request. Red failed because the notifier exposed only `refreshFor(trigger)`. Removed the seven-value trigger enum and policy, changed the notifier to `refresh()`, and updated the existing resume, foreground, and mark-seen call sites. The focused count/lifecycle/seen/shell/architecture run passed 10 tests; the existing no-`Timer` and no-icon-badge guards remain in place.
- S3 — IR-007 / IR-011 / UT-012 / UT-018 / IT-007 / IT-010 / REG-009: added a behavioral runtime test requiring idempotent start, one listener per provider-neutral stream, one initial-open read, one banner/list/count sequence per duplicate foreground callback, and complete cancellation on disposal. Red failed because `NotificationRuntime` did not accept or own the service. Moved initialization, subscriptions, initial-open consumption, foreground effects, and disposal into the runtime; removed `NotificationServiceOwner`, `ForegroundNotificationHandler`, their callback cycle, and the source-layout ownership assertion. The focused runtime/effect/open/architecture run passed 8 tests, then the complete notification suite passed 68 tests.
- S4 — IR-010 / UT-001 / UT-013 / IT-002: replaced the separate permission and token tests with one registration-lifecycle suite. Red failed because `NotificationRegistrationCoordinator` accepted only token callbacks and exposed no readiness or permission-recheck API. Moved permission checking/requesting, readiness, resume recovery, token retrieval, latest-token retention, single-flight registration, and DID-before-save fencing into one coordinator; removed `NotificationCoordinator` and `NotificationPermissionPolicy`. The focused registration/runtime/effect/open run passed 11 tests, then the complete notification suite passed 66 tests.
- S4 concurrency follow-up — UT-013 / FR-005 / FR-006: a final public-lifecycle audit added two race tests for token refresh and DID replacement during an in-flight registration. The latest-token test failed because the first consolidated version returned the existing future without draining the changed token. The coordinator now serializes registrations and drains only revisions that arrive during the active attempt; unchanged failures still stop until a later explicit trigger, and old-DID results are discarded before saving. The combined lifecycle suite passed 8 tests.
- S5 — IR-012 / UT-003 / UT-008 / UT-016 / IT-005 / IT-012 / AT-012: added an adapter-boundary test requiring the exact foreground and permission presentation values. Red failed because those values were hidden behind an intermediate presentation object. Inlined the three Firebase presentation fields in the adapter, folded the render-token set into `NotificationSeenCoordinator`, and inlined the missing/stale binding check in `NotificationOpenCoordinator`; removed the presentation, seen, and routing policy files and their implementation-mirroring unit tests. The retained seen-after-render, stale/missing-binding-with-no-HTTP, Firebase configuration, open-flow, and seen-flow tests passed, then the complete notification suite passed 65 behavioral tests. The lower count reflects deletion of structure-coupled tests, not lost behavior coverage.
- Generated-provider follow-up: added an architecture test requiring every notification provider source to include its generated part, use `@riverpod`/`@Riverpod`, and avoid manual provider constructors. Red failed on the existing manual declarations. Converted all nine provider source files to generated declarations, preserved every public provider name and `isAutoDispose: false` lifetime, declared the effect stream as `Raw<Stream<NotificationEffect>>`, retained all override APIs, moved pagination to the rule-prescribed `state.requireValue` pattern, and updated provider tests to `ProviderContainer.test()`. Build generation completed successfully; the architecture/provider slice passed 13 tests and the complete notification suite passed 68 tests.
- Identifier-model follow-up: moved `NotificationId` and `AccountSubscriptionId` out of `notification_open_event.dart` into dedicated model files. API resolution/durable rows now import only `NotificationId`; registration/secure routing import only `AccountSubscriptionId`; open-event consumers import the event plus identifiers only when their own signatures require them. Added direct parsing/equality/wire/redaction tests for each identifier while retaining the provider-data rejection matrix. The focused identifier/routing/open-flow slice passed 18 tests, the complete notification suite passed 72 tests, and full Flutter analysis remained clean.

### Implementation Review Correction Pass

- Status: complete; ready for follow-up implementation review.
- First failing test: C1 / IT-002 / EC-013, denied permission changed to authorized in OS settings before app resume.
- Guardrails: no new product behavior, routes, persistence, polling, Firebase boundary, or AppView contract; source changes must map to IR-001–IR-005 and the IDs above.
- Manual state: MAN-001–MAN-005 remain blocked on the documented external prerequisites and are not implied by automated correction evidence.
- C1 — IT-002 / IR-001: added a denied-to-authorized resume case. Red failed because resume did not call `getPermission()` again (`permissionChecks` stayed at 1). `NotificationCoordinator` now retains only current readiness inputs and re-checks permission without prompting on resume before updating registration eligibility. The focused coordinator suite passed 4 tests. Settings-state invalidation remains paired with the root lifecycle loop in C4.
- C2 — UT-019/AT-010/IT-011 / IR-002: replaced the lifetime-idempotence expectation with one test that proves overlapping calls coalesce while a later same-DID unconfirmed sign-out runs again. Red failed because the second cycle made zero token-deletion calls. Removed the permanent `_completed` set while retaining per-DID in-flight coalescing. The cleanup and existing 401 interceptor suites passed 4 tests.
- C3 — AT-006/IT-007 / IR-003: added a widget test with one future generic row and one unavailable tombstone. Red failed because the generic tap made zero resolution calls. `NotificationRow` now resolves generic/everythingElse rows by stable ID through the owner-scoped repository and uses only the authorized resolution outcome; tombstones remain non-navigable and emit localized warning feedback. Navigation outcome rendering was refactored into one shared helper used by both the root effect host and durable rows while green. The complete Notifications page suite passed 5 tests.
- C4 — IT-008/REG-004 / IR-001/IR-004: added a root lifecycle test with signed-out auth, a recording newness repository, and a cached denied permission. Red observed two unauthorized count calls on resume. `NotificationEffectHost` now tracks current authenticated/onboarded readiness, invalidates the shared permission provider on resume, and refreshes count only for a ready account. The permission provider moved to its planned standalone module. The lifecycle and coordinator suites passed 5 tests, including denied-to-authorized cache recovery.
- C5 — IT-004/IT-012 / IR-005: added a public-boundary runtime test using real secure DID-keyed routing storage, mocked Dio resolution, an unknown normalized category, and the provider-neutral effect stream. The new test was green on first execution: one matching binding made one owner-scoped GET and emitted only the server post target; a stale binding emitted unavailable with no second GET; signed-out readiness discarded the open. No production change was needed or manufactured for this coverage-only gap.
- C6 — IT-005/REG-008 / IR-005: added a widget/provider flow covering controlled first-page loading, a retained failure state, rendered content, and rendered empty success. The first run stalled because the test used `pumpAndSettle` across Riverpod's automatic provider retry; disabling retry in the harness produced the intended observable states and was a test-setup correction, not a product failure. The final test proves zero seen/count work for loading and failure and one seen then count refresh for each successful render token. A mocked-Dio API test also proves the seen request is a bodyless POST. Both tests were green once the harness represented the specified states; no production change was needed.
- C7 — IT-006 / IR-005: added mocked-Dio GET/PATCH contract tests at the HTTP boundary. GET proves a future category and its unknown fields survive decoding outside the closed UI enum; PATCH proves the exact path and a body containing one known category with one changed field, while the returned future category remains preserved. Both tests passed on first execution, confirming the existing API/model seam already implemented the contract; no product change was manufactured for this coverage-only gap.
- C8 — IT-007 / IR-005: extended the root effect-host test through a real `NotificationServiceOwner`, `NotificationRuntime`, foreground handler, effect stream, and recording messenger. After correcting the test runtime's initially omitted banner-to-effect callback, two identical provider-neutral foreground receipts each produced visible localized action feedback, one list invalidation, and one count refresh. The three-test lifecycle/foreground host suite passed; production behavior was unchanged because the missing link was confined to the new test harness.
- C9 — IT-008 / IR-004/IR-005: added real indexed-shell tests at compact and large form factors with a fake count repository. The initial red proved the visual `99+` badge rendered but its actual-count semantics were lost; the large rail was entirely suppressed by the nested branch navigator's route semantics. `AppShell` now supplies the localized actual count through each destination label and orders the large branch navigator before the rail in semantics while reversing only the Row layout direction, preserving the rail on the leading visual edge in LTR and RTL. Both shell layouts expose `137 new activities`, render `99+`, and the root lifecycle suite additionally proves signed-out resume makes zero count calls while ready resume adds exactly one. The combined shell/lifecycle run passed 5 tests.
- C10 — IT-009 / IR-005: added a production-router widget test with signed-in/onboarded auth and fake notification repositories. It asserts the generated typed location, invokes the real Notifications settings action, verifies the matched `/notifications/settings` page is full-screen above the shell, then uses Back to restore the Notifications branch without reloading its first page. The initial harness allowed the auto-disposed router provider to release between interactions; retaining the provider as the production app does fixed the harness, and the test passed without a product change.
- C11 — REG-002 / IR-005: extended the existing observability secret scan with a notification-directory guard against direct logging, analytics, breadcrumb, and Sentry sinks plus runtime stringification sentinels for notification ID, routing ID, title, and body. Extended the Sentry sanitizer test with token, routing ID, notification ID, DID, handle, AT-URI, provider payload, and credential fields while retaining only bounded classification/endpoint categories. The combined scan/sanitizer run passed 6 tests without a product change.
- C12 — IT-011/REG-004/REG-009 / IR-005: added auth-controller integration assertions for confirmed and failed logout ordering: confirmed logout retains the provider token, while failed logout best-effort deletes it; both remove the current DID binding before clearing the local session. Added structural guards that constrain Firebase listener APIs to the adapter, one service-owner construction to the runtime provider, one effect-host mount to the root app, and prohibit count timers, platform icon-badge APIs, persisted pending opens, notification receipt storage, or notification IDs in the routing-binding store. The first structural run caught an overbroad test token that matched the provider-neutral registration callback; narrowing it to the Firebase adapter receiver produced the intended guard. Auth/cleanup passed 13 tests and architecture/config/owner passed 7 tests; no product change was needed.

### Step 1: UT-002
- Write failing test: added the known/future/malformed/extra-key provider payload matrix.
- Run command: `cd app && flutter test test/notifications/models/notification_open_event_test.dart`
- Confirmed failure: yes — compilation failed because the provider-neutral event boundary did not exist.
- Implement: added validated, redacted notification/routing identifiers, bounded type parsing, known-category mapping, unknown-category normalization, and extra-key ignoring.
- Run command: `flutter test test/notifications/models/notification_open_event_test.dart` — 3 tests passed.
- Refactor: formatted the focused source and test; the focused test remained green.
- Notes: no Firebase import or destination-shaped provider value entered the domain model.

### Step 2: UT-003, UT-015
- Write failing test: added exact-match routing policy plus two-DID replace/remove and corruption storage cases.
- Run command: focused routing policy/storage tests
- Confirmed failure: yes — compilation failed because routing policy/storage seams did not exist.
- Implement: added exact typed binding comparison and a secure-storage-backed DID map with replace-one, remove-one, and corruption cleanup.
- Run command: focused tests — 3 tests passed; UT-002 plus focused routing tests — 6 tests passed.
- Refactor: formatted source/tests while green.
- Notes: removing Alice preserves Bob; routing data uses the dedicated secure-storage backend and never shared preferences.

### Step 3: UT-004
- Write failing test: added active post/profile, notifications/retracted/malformed, and not-found/network/timeout cases.
- Run command: focused resolution policy test
- Confirmed failure: yes — compilation failed because resolution model/policy types did not exist.
- Implement: added AppView resolution decoding and a pure navigation outcome policy using only active server targets.
- Run command: focused test — 3 tests passed; all completed pure notification tests — 9 tests passed.
- Refactor: formatted source/tests while green.
- Notes: safe failures always select Notifications; only network/timeout adds feedback; no outcome supports retry persistence.

### Step 4: UT-001
- Write failing test: added the signed-in/onboarded/undetermined-authorized-denied matrix.
- Run command: focused permission policy test
- Confirmed failure: yes — compilation failed because the permission policy model did not exist.
- Implement: added provider-neutral permission state and explicit `none`/`request`/`register` policy actions.
- Run command: focused test — 1 test passed.
- Refactor: formatted source/test while green.
- Notes: only signed-in + onboarded + undetermined requests; authorized registers and denied does nothing.

### Step 5: UT-013
- Write failing test: added deferred latest-token, empty-token, transient failure/retry, and ineligible readiness cases.
- Run command: focused registration coordinator test
- Confirmed failure: yes — compilation failed because the registration coordinator did not exist.
- Implement: added provider-neutral platform/registration callbacks, in-memory latest-token retention, eligibility gating, single-flight attempts, and failure swallowing.
- Run command: focused test — 3 tests passed.
- Refactor: formatted source/test while green.
- Notes: no unauthenticated registration, empty tokens are skipped, and transient failure waits for another explicit trigger.

### Steps 6–24
- Tests and requirement links: use the ordered table above without reordering.
- Step 6 — IT-001/IT-002/AT-002: exact device-registration request and readiness orchestration tests failed for missing API/service/coordinator seams; added the provider-neutral service contract, device repository/API, and non-blocking permission-to-registration coordinator. Focused tests passed (3), then the completed notification unit/integration slice passed (17) after adding direct concrete-repository imports found by the nearby suite.
- Step 7 — UT-012/IT-010/AT-009: one-owner lifecycle test failed for the missing owner; added single-start initialization, one initial-open consumption, one subscription per provider-neutral stream, callback forwarding, idempotent cancellation, and service disposal. Focused test passed (1).
- Step 8 — UT-014/AT-011: pending-open test failed for the missing readiness slot; added a non-persistent latest-wins transient slot that consumes once on ready and clears permanently when sign-in is required. Focused test passed (1).
- Step 9 — AT-001/REG-001/REG-005/REG-006: static tests failed on legacy identities and missing Firebase/native files. Downloaded the two existing `craftsky-app` client configs, added Firebase/app-settings dependencies and an adapter/bootstrap/background boundary, aligned both app IDs, created the single manifest-bound Android channel, and enabled iOS remote notifications/entitlements. The project plist/entitlements/bundle/capability wiring was applied through native Swift/XcodeProj tooling (no duplicate Firebase SPM dependency). Static tests passed (4); focused analysis has no errors.
- Step 10 — UT-016/AT-012: the presentation-policy test failed because the provider-neutral policy did not exist. Added explicit background permission effects (alert + sound, no badge) and silent foreground effects (no OS alert, sound, badge, vibration, or local notification), then wired the Firebase adapter to the shared policy.
- Step 11 — UT-018/AT-003/IT-007: the repeated-receipt test failed because the foreground handling seam did not exist. Added an ordered, provider-neutral handler that emits a banner, invalidates the first page, and refreshes count for every callback without ID storage, deduplication, or page-visibility suppression.
  Production wiring: mounted one provider-owned service/runtime instance under a single root `NotificationEffectHost`; the runtime alone owns provider subscriptions, readiness, registration, pending opens, secure-binding validation, AppView resolution, invalidation, and provider-neutral effects. The host only presents localized banners/feedback and typed routes, with no Firebase import or subscription. A final traceability red/green pass added app-resume registration retry and the approved resume-only count refresh trigger.
- Step 12 — AT-004/AT-005/IT-004/IT-012: the authorized-open tests failed because no coordinator or resolution endpoint seam existed. Added exact current-DID binding validation before HTTP, owner-scoped AppView resolution by stable ID, and destination mapping solely from the resolved AppView target; stale/missing bindings emit unavailable without resolving. A final red/green failure-mapping test distinguishes AppView 404/unavailable fallback from network/server feedback without retaining retry state.
- Step 13 — UT-005/IT-003/AT-006: the durable-row matrix failed on quote/everythingElse/unknown types, stable IDs, and tombstones. Expanded decoding to seven known categories plus generic unknown, retained stable IDs, converted unavailable actor/content to safe tombstones, and changed pagination deduplication from source URI to stable ID.
- Step 14 — UT-006/UT-007/IT-008: badge/trigger tests failed because no count model or trigger boundary existed. Added 0-hidden/99+ badge formatting, exactly five event-driven refresh triggers, AppView count/seen adapters, and the same accessible badge presentation for bottom navigation and rail; no timer or platform icon badge exists.
- Step 15 — UT-008/IT-005/AT-007/REG-008: the acknowledgement-gate test failed because no successful-render token existed. Added one token per successful first-page load, preserved it through load-more, consumed it post-frame only after content/empty rendering, posted bodyless seen once, and refreshed count only after success; failures release the token for a later render attempt.
- Step 16 — UT-009/UT-011/IT-006: preference decoder/patch tests failed because the account-wide model and API seams did not exist. Added closed seven-category/scope values, raw unknown-entry preservation, exact one-category/one-field PATCH serialization, and GET/PATCH repository adapters.
- Step 17 — UT-010: the overlapping-edit test failed because no optimistic preference provider existed. Added per-category/per-field mutation generations, immediate optimistic state, current-generation server merge, and targeted rollback that cannot overwrite a newer edit.
- Step 18 — AT-008/IT-009: the settings widget test failed because the page/route did not exist. Added `/notifications/settings` on the root navigator, a Notifications action, seven account-wide sections, independent scope/push controls, denied-device guidance with native-settings action, loading/retry/error feedback, and no master or unknown-category controls. Focused widget test passed.
- Step 19 — UT-019/AT-010/IT-011: cleanup tests failed because provider token/routing cleanup was absent. Added idempotent single-flight per-DID cleanup: confirmed logout retains the Firebase token, failed/forced logout best-effort deletes it, and every path removes only the current DID binding before local session clearing. Existing auth/session/401 tests plus focused cleanup tests passed (21).
- Step 20 — UT-017: added generated localization coverage for settings, category/tombstone rows, badge semantics, and action/error copy; replaced notification UI literals with `AppLocalizations` and regenerated localization sources. Focused localization/settings/badge tests passed (3).
- Step 21 — REG-002/REG-003/REG-004/REG-009: the final notification/auth/app harness passed 74 tests, covering the notification slice, explicit logout, background session validation, global 401 cleanup, and root app initialization. The canonical full Flutter suite ran 821 tests and had one isolated failure in the unchanged feed repost test described under Final Verification.
- Step 22 — IT-013: the constrained entry-point test failed because Firebase initialization was not injectable. Kept the top-level retained handler and added only an optional initializer seam; the handler still performs Firebase initialization alone and never reads/logs payload data or touches UI, routing, providers, or navigation.
- Step 23 — IT-014/REG-007: the existing complete-message regression failed on missing APNs payload. Added only `aps.sound = "default"`; token, visible copy, data map, Android TTL/config, APNs expiration, provider call, and result classification remain on the existing path.
- Step 24 — IT-015/REG-010: the expanded configuration test proved dev could enable push without a Firebase project. Validation now requires `FIREBASE_PROJECT_ID` whenever `PUSH_ENABLED=true` in any environment, while the default remains false; `environments/dev.env` records the normal disabled state explicitly.
- Refactors: only while the focused and nearby suites are green.

### Step 25: MAN-001–MAN-005
- Status: blocked on external physical-device and APNs/FCM credential prerequisites; not run.
- Guardrail: verify `PUSH_ENABLED=false` before and after any bounded delivery session.
- Notes: `appview/environments/dev.env` remains `PUSH_ENABLED=false`; no bounded sender session was started, and automated success does not imply live delivery.

## Final Verification

- Simplification, generated-provider, and identifier-model notification suite: `cd app && flutter test test/notifications` passed 72 behavioral/architecture tests, including direct identifier contracts, the in-flight registration race cases, and generated-provider guard.
- Linked auth/router/observability regression suite: `cd app && flutter test test/auth test/router test/observability test/shared/errors` passed 98 tests.
- Targeted analysis: `cd app && dart analyze lib/notifications test/notifications` passed with no issues.
- Focused AppView tests: `cd appview && go test ./internal/push ./internal/app ./internal/api ./internal/routes -count=1` passed all four packages.
- Repository Flutter tests: the last canonical `just app-test`, immediately before the identifier-only file split, ran 846 tests; 845 passed and the unchanged `test/feed/pages/post_comment_section_page_test.dart` case `wires repost action for the root post` failed with the same empty repost-call ledger recorded before this simplification pass. The subsequent identifier split passed all 72 notification tests and full Flutter analysis; no feed implementation or test file is changed by this slice.
- Repository `just app-analyze`: passed — no issues found.
- Repository `just test`: passed with the race detector and local Postgres.
- `git diff --check`: passed.
- Sender gate: verified `appview/environments/dev.env` contains `PUSH_ENABLED=false`; no live sender invocation occurred.
- Diff-to-requirement traceability review: completed; IR-007–IR-012 were implemented without changing the approved observable behavior or Firebase/security boundaries.

## Completion Checklist

- [x] IR-001–IR-005 corrected through red-green-refactor loops
- [x] IR-007–IR-012 implemented through staged red-green-refactor loops
- [x] All notification providers generated under the repository Riverpod rules
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must automated behavior passing in the focused and linked suites
- [x] Relevant notification/auth/AppView regressions passing; unrelated baseline failures documented accurately
- [x] No unlinked behavior implemented
- [x] Docs and final evidence updated
- [x] Manual checks explicitly reported as blocked on external prerequisites
- [ ] Follow-up implementation review completed or explicitly skipped
