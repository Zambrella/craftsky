# Implementation Review: Flutter Push Notifications

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-07-16
Risk level: Medium

## Summary

The corrected Flutter notification slice is behaviorally well covered: all 68 focused notification tests pass and targeted static analysis reports no issues. No correctness, privacy, ownership, or test-coverage blocker was identified in this simplification follow-up.

The implementation is nevertheless more fragmented than it needs to be. The largest cost is not the Firebase boundary or the stateful race handling; it is the chain of one-purpose orchestration objects and pass-through adapters around them. A behavior-preserving cleanup can remove several files and substantially reduce test setup while retaining the provider-neutral `NotificationService`, Riverpod override seams, public-boundary widget/integration tests, typed/redacted identifiers, secure routing storage, pending-open state, and optimistic-edit generation handling.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-007 | Suggestion | Code Quality | Runtime ownership is split across `NotificationRuntime`, `NotificationServiceOwner`, and `ForegroundNotificationHandler`. The provider then creates a `late final` callback cycle so the owner can call back into the runtime. These objects are never independently reused, and tests must reconstruct the same graph manually. | NFR-001, NFR-002; UT-012, UT-018, IT-004, IT-007, IT-010; `app/lib/notifications/providers/notification_runtime_provider.dart:28`; `app/lib/notifications/services/notification_runtime.dart:16`; `app/lib/notifications/services/notification_service_owner.dart:13`; `app/lib/notifications/services/foreground_notification_handler.dart:9`; `app/test/notifications/notification_open_flow_test.dart` | Non-blocking: let `NotificationRuntime` own service initialization/subscriptions/disposal and the three foreground effects directly. Keep `NotificationService` as the fakeable provider boundary and move the exactly-once and ordered-effect assertions into runtime tests. |
| IR-008 | Suggestion | Code Quality | The data path has a pure forwarding layer: `NotificationApiClient` performs every operation, `ApiNotificationRepository` forwards every method, and five capability providers each construct a new wrapper around the same client. The narrow capability interfaces are useful test seams; the forwarding implementation is not. | NFR-001; IT-001, IT-004–IT-006, IT-012; `app/lib/notifications/data/notification_api_client.dart:9`; `app/lib/notifications/data/api_notification_repository.dart:9`; `app/lib/notifications/providers/notification_repository_provider.dart:7` | Non-blocking: make the HTTP adapter implement the existing capability interfaces directly (renaming it to `ApiNotificationRepository` if preferred), remove the forwarding class, and have all capability providers expose the same adapter instance. Existing provider overrides remain unchanged. |
| IR-009 | Suggestion | Code Quality | `NotificationNewCountTrigger` names five call sites that all execute identical behavior, plus two impossible production triggers used only to prove that callers do not exist. This pushes architecture-test concerns into the production API without adding a behavioral decision. | FR-013, NFR-001; UT-007, IT-008, REG-004; `app/lib/notifications/providers/notification_new_count_provider.dart:4`; `app/test/notifications/notification_architecture_test.dart:75` | Non-blocking: replace `refreshFor(trigger)` with `refresh()`. Keep refresh timing covered at the effect-host, foreground-flow, seen-flow, and shell boundaries, and retain the static no-`Timer` guard. |
| IR-010 | Suggestion | Code Quality | Permission readiness and device registration form one lifecycle but are split between `NotificationCoordinator`, `NotificationPermissionPolicy`, and `NotificationRegistrationCoordinator`. The outer coordinator repeats state already held by the inner coordinator and delegates every terminal path to it. | FR-003, FR-004, FR-018; UT-001, UT-013, IT-002; `app/lib/notifications/services/notification_coordinator.dart:6`; `app/lib/notifications/models/notification_permission.dart`; `app/lib/notifications/services/notification_registration_coordinator.dart:20` | Non-blocking: consolidate these into one registration-lifecycle controller while preserving the in-flight fence, latest-token behavior, permission recheck on resume, and DID recheck before saving. Test the combined public lifecycle rather than each internal hop. |
| IR-011 | Suggestion | Tests | The ownership regression test asserts constructor names and exact source-file locations. That protects the current decomposition and will fail when the implementation is simplified even if exactly-one ownership remains behaviorally correct. The Firebase-confinement and forbidden-storage/timer scans remain valuable. | NFR-002; REG-004, REG-009; `app/test/notifications/notification_architecture_test.dart:32`; `app/test/notifications/providers/notification_service_owner_test.dart` | Non-blocking: replace the `NotificationServiceOwner` source-layout assertion with a behavioral runtime test proving one initialization, one subscription set, one initial-open read, and no delivery after disposal. Keep only source scans that enforce a dependency/privacy boundary. |
| IR-012 | Suggestion | Code Quality | A few very small policy types expose no reusable decision: `NotificationSeenGate` is a set around one coordinator, `NotificationRoutingPolicy` is an exact equality expression, and `NotificationPresentationOptions` carries `vibration` and `localNotification` fields the Firebase adapter never reads. Each is testable, but their tests mirror the implementation rather than a broader observable boundary. | UT-003, UT-008, UT-016; `app/lib/notifications/services/notification_seen_policy.dart`; `app/lib/notifications/services/notification_routing_policy.dart`; `app/lib/notifications/services/notification_presentation_policy.dart` | Non-blocking: fold the seen set into `NotificationSeenCoordinator`, inline the binding equality inside the owner-scoped open flow, and reduce presentation configuration to the three Firebase fields actually consumed. Preserve the seen-flow, stale-binding/no-HTTP, and Firebase configuration tests. |

## Requirement And Test Traceability

- Requirements implemented: FR-001–FR-026 remain represented in the current implementation. The proposed simplifications do not change notification eligibility, permission timing, registration, provider-data validation, owner-scoped resolution, foreground effects, newness/seen semantics, preferences, navigation, sign-out cleanup, or native delivery configuration.
- Tests implemented: The focused suite covers parsing, Firebase configuration and background entry point, registration races, routing storage and ownership, open resolution, foreground effects, new-count/badge behavior, seen-after-render behavior, preferences, settings UI, sign-out cleanup, and architecture/privacy guards.
- Unplanned behavior: None identified in this follow-up.
- Remaining gaps: MAN-001–MAN-005 remain external physical-device/provider checks. They are not made riskier by the proposed internal refactors, but must still be run separately when prerequisites are available.

## Test Evidence

- Commands reviewed:
  - `cd app && flutter test test/notifications`
  - `cd app && dart analyze lib/notifications test/notifications`
- Passing evidence:
  - Focused notification suite passed all 68 tests.
  - Targeted notification production/test analysis completed with no issues.
- Failing or skipped tests:
  - No focused failures.
  - The repository-wide Flutter suite, linked auth/router suites, AppView tests, and manual physical-device checks were not rerun for this read-only simplification review.

## Risk Review

- Risk level: Medium
- Risk notes: The behavior is sensitive to app lifecycle, asynchronous stream ownership, authentication, and secure owner-scoped routing. Simplification is safe only if it preserves the existing provider-neutral service boundary and public lifecycle/integration coverage. A single large rewrite would be harder to validate than the staged refactor order below.
- Approval notes: Current behavior is approved. Treat IR-007–IR-012 as optional maintainability work, not release blockers.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: This review concerns internal notification orchestration and test structure. No visible UI issue was identified.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: If simplification is approved, start with the data adapter and new-count API because they are low-risk mechanical changes. Then add one runtime lifecycle test that expresses the combined ownership behavior before merging `NotificationServiceOwner` and `ForegroundNotificationHandler` into `NotificationRuntime`.
- Verification to rerun: After each refactor, run the focused notification tests and targeted analysis. After the runtime/registration consolidation, rerun linked auth/router/app lifecycle tests, canonical `just app-test`, `just app-analyze`, `git diff --check`, and the relevant AppView notification tests. For manual delivery, confirm credential-aware development startup reports push enabled and record the intended local device/account before generating events.
