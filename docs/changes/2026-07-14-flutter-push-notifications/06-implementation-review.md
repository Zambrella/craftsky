# Implementation Review: Flutter Push Notifications

## Verdict

Status: Changes required  
Reviewer: Codex  
Date: 2026-07-15  
Risk level: High

## Summary

The implementation establishes most of the planned native Firebase configuration, provider-neutral service boundary, device registration, notification models, new-count and seen state, preferences, typed routes, sign-out hooks, and AppView sound/config follow-up. Focused tests and static analysis are green.

The change is not ready to merge or hand off, however. App resume does not re-read a previously denied OS permission, the long-lived sign-out cleanup permanently suppresses later cleanup for the same DID, durable generic rows never use AppView resolution, unavailable rows provide no tap feedback, and signed-out resume can create an authenticated new-count request. Required integration and privacy/ownership regression coverage is also materially incomplete despite `05-implementation-plan.md` recording it as complete.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior | Resume retries registration only from the coordinator's existing eligibility state. If permission was denied, registration is ineligible; granting permission in OS settings does not cause `getPermission()` to run again, so no registration occurs. The settings permission `FutureProvider` is also not invalidated, leaving the denied-device warning stale. | FR-003, FR-004, FR-018; AC-003, AC-004, AC-013; EC-013; IT-002; MAN-004; `app/lib/notifications/widgets/notification_effect_host.dart:39`; `app/lib/notifications/services/notification_runtime.dart:68`; `app/lib/notifications/services/notification_coordinator.dart:43`; `app/lib/notifications/pages/notification_settings_page.dart:11`; `app/test/notifications/providers/notification_coordinator_test.dart:62` | Start with a failing denied-to-authorized resume test. On eligible resume, re-read OS permission without re-prompting, update registration eligibility, register exactly once when authorization became available, and refresh/invalidate the settings permission state. |
| IR-002 | Important | Risk | `NotificationSignOutCleanup` records each DID in `_completed` forever, while its provider is long-lived. After Alice signs out, signs in again, and later signs out or receives a 401, cleanup is skipped, so the new local routing binding can remain and unconfirmed cleanup does not attempt token deletion. The current test explicitly locks in once-per-provider-lifetime behavior rather than once-per-sign-out behavior. | FR-021; AC-017, AC-024; EC-017; UT-019, AT-010, IT-011, MAN-003; `app/lib/notifications/services/notification_sign_out_cleanup.dart:14`; `app/lib/notifications/providers/notification_lifecycle_provider.dart:14`; `app/test/notifications/services/notification_sign_out_cleanup_test.dart:5` | Add a failing same-DID re-authentication test covering two distinct sign-out cycles, including an unconfirmed second cycle. Preserve concurrent single-flight coalescing, but allow every later sign-out cycle to execute its required token/binding policy. |
| IR-003 | Important | Behavior | Both `GenericNotification` and `UnavailableNotification` taps return immediately. Unknown/everythingElse rows therefore never resolve their stable notification ID through AppView, and unavailable tombstones show no brief localized feedback. Existing page navigation tests cover only known available rows. | FR-011, FR-012; AC-010, AC-028, AC-029; EC-009, EC-010; AT-006, IT-003, IT-007; `app/lib/notifications/widgets/notification_row.dart:49`; `app/test/notifications/notifications_page_test.dart:102` | Add failing widget/integration tests for unknown/everythingElse and unavailable rows. Resolve supported generic rows through the owner-scoped AppView resolution repository using the stable notification ID and navigate only from the returned target; keep tombstones non-navigable and show localized unavailable feedback. |
| IR-004 | Important | Behavior | The root effect host is mounted for the ready app even when signed out, and resume unconditionally reads `notificationNewCountProvider.notifier`. Constructing that notifier immediately calls the authenticated `new-count` endpoint, so returning to a signed-out Welcome flow can make an unauthorized request. | FR-013; AC-020; RULE-005; IT-008, REG-004; `app/lib/app.dart:64`; `app/lib/notifications/widgets/notification_effect_host.dart:39`; `app/lib/notifications/providers/notification_new_count_provider.dart:27` | Add a failing root lifecycle test proving signed-out or not-onboarded resume performs zero count calls, then gate the resume refresh on authenticated/onboarded readiness while retaining exactly one refresh for a ready account. |
| IR-005 | Important | Tests | Several planned Must integration/regression seams have no equivalent automated coverage: full open/routing flow (IT-004/IT-012), seen render flow (IT-005/REG-008), preferences HTTP contract (IT-006), foreground host presentation (IT-007), compact/rail badge integration (IT-008), typed settings route (IT-009), auth-controller cleanup integration (IT-011), redaction sentinels (REG-002), forbidden polling/persistence guards (REG-004), and one-owner import/ownership protection (REG-009). `05-implementation-plan.md` nevertheless says these steps and all planned Must automation passed. | NFR-001, NFR-002; AC-006–AC-009, AC-011–AC-018, AC-020–AC-026; IT-004–IT-009, IT-011, IT-012; REG-002, REG-004, REG-008, REG-009; `02-acceptance-tests.md` sections 5–6; `05-implementation-plan.md:109`; `05-implementation-plan.md:119`; `05-implementation-plan.md:146` | Add public-boundary integration/regression tests with the planned observable scope (exact filenames are not required), including the new tests from IR-001–IR-004. Extend the existing observability scans instead of relying on code inspection. Update `05-implementation-plan.md` so its traceability and final evidence match what was actually run and covered. |
| IR-006 | Suggestion | Tests | The reported canonical Flutter suite remains red in an unchanged feed repost test. It reproduces alone and no feed files are changed by this slice, so it is not evidence that the notification implementation caused a regression, but the repository is not at a fully green merge baseline. | `05-implementation-plan.md:134`; `test/feed/pages/post_comment_section_page_test.dart` | Resolve the baseline failure separately or record explicit maintainer acceptance before merge; do not describe the full Flutter suite as passing. |

## Requirement And Test Traceability

- Requirements implemented: Substantial implementation exists for FR-001–FR-026, including native Firebase identity/configuration, the provider-neutral adapter, registration and routing storage, foreground and open coordinators, durable models, new-count/seen state, settings/preferences, sign-out hooks, and the AppView APNs sound/non-production gate.
- Tests implemented: Unit and focused integration coverage exists for parsing, permission policy, registration policy, routing storage/policy, service ownership, pending opens, resolution policy, durable decoding/pagination, badge formatting/trigger classification, preference model/provider races, settings page controls, sign-out cleanup, native configuration, background handler, AppView sender/config, and related auth/app paths.
- Unplanned behavior: Signed-out resume can instantiate the authenticated count provider and issue `GET /v1/notifications/new-count` (IR-004). No unrelated product behavior or lexicon/API route change was identified in the reviewed diff.
- Remaining gaps: IR-001–IR-005 are blocking. Physical-device delivery and OS behavior remain unverified because MAN-001–MAN-005 were correctly reported as blocked on external prerequisites.

## Test Evidence

- Commands reviewed:
  - `flutter test test/notifications/providers/notification_coordinator_test.dart test/notifications/services/notification_sign_out_cleanup_test.dart test/notifications/notifications_page_test.dart`
  - `dart analyze`
  - `git diff --check`
  - Implementation-reported focused notification, notification/auth/app, AppView, repository Go, and full Flutter commands in `05-implementation-plan.md`.
- Passing evidence:
  - Review-focused Flutter command passed 9 tests.
  - `dart analyze` passed with no issues.
  - `git diff --check` passed.
  - `05-implementation-plan.md` reports 48 focused notification tests, a 74-test linked harness, focused AppView tests, and repository `just test` passing.
- Failing or skipped tests:
  - The implementation-reported full Flutter suite passed 820 of 821 tests; `wires repost action for the root post` failed and reproduced alone.
  - MAN-001–MAN-005 were not run because physical devices and APNs/FCM credentials were unavailable.
  - The planned Must integration/regression coverage listed in IR-005 is absent or not equivalent to the acceptance-test scope.

## Risk Review

- Risk level: High
- Risk notes: This slice changes authentication cleanup, secure routing state, native Firebase configuration, permission lifecycle, external delivery, and user navigation. IR-002 can leave a later session's routing state behind after sign-out, and IR-001 can leave a user opted in at the OS level but unregistered. Missing privacy and ownership guards make regressions at those boundaries harder to detect.
- Approval notes: Address IR-001–IR-005 with a new strict red-green-refactor pass, then rerun the focused notification/auth/app suites, static analysis, diff checks, relevant AppView tests, and the canonical broader suites. Keep manual delivery explicitly blocked until its external prerequisites are available.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The blocking visible issues are behavioral and belong in TDD. After they are corrected, a small device-level polish pass could improve confidence in the settings warning transition, tombstone feedback, banner layout, and compact/rail badge presentation, but no separate polish change should precede the required fixes.
- Suggested polish notes: Check long localized banner copy, settings cards at large text sizes, unavailable feedback timing, and badge semantics/placement on both compact navigation and the large rail.

## Handoff Back To TDD Builder

- Required fixes:
  - IR-001: re-read permission and recover registration/settings state on resume.
  - IR-002: scope sign-out idempotency to concurrent work rather than the DID's entire provider lifetime.
  - IR-003: resolve generic durable rows through AppView and provide tombstone feedback.
  - IR-004: prevent signed-out/not-onboarded resume count calls.
  - IR-005: complete Must integration/privacy/ownership coverage and correct implementation evidence.
- Suggested next failing test: Start with IT-002/EC-013: denied permission at readiness, authorization changed in OS settings, then resume must re-read permission and make exactly one authenticated registration while updating the settings warning state.
- Verification to rerun: Focused tests for each red-green cycle; the full notification, auth, router, app/bootstrap, observability, and related feed suites; `just app-analyze`; focused AppView push/config/API tests; repository `just test`; `git diff --check`; then MAN-001–MAN-005 when prerequisites are available.
