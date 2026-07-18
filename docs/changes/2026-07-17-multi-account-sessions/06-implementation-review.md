# Implementation Review: Multi-Account Sessions And Notification Routing

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-07-18
Risk level: High

## Summary

The implementation is behaviorally sound, and no account-isolation defect was identified in this complexity review. Fixed-session clients, account-scoped `401` invalidation, stale-completion fencing, exact notification-recipient resolution, and per-account cache boundaries are load-bearing complexity and should remain while their corresponding behavior remains in scope.

The implementation is nevertheless more complex than the product needs to be if some recovery and presentation requirements can be relaxed. There is also dead single-account compatibility code and redundant model/provider ceremony that can be removed without changing behavior. The highest-value simplification is a coherent package: require confirmed online sign-out, remove cleanup-only credentials, and replace the verified two-slot journal with one fail-closed secure snapshot. Smaller independent relaxations can remove eager inactive-session validation, inactive-account count badges, and the global identity transition overlay.

These are non-blocking product/maintenance choices, not correctness findings. The current implementation remains aligned with the approved requirements and is acceptable as-is if offline sign-out recovery, partial secure-storage recovery, inactive badges, eager validation, and the richer transition experience are all still desired.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-007 | Suggestion | Code Quality | The superseded single-account storage path remains in production sources even though the registry replaced it and the project explicitly has no legacy-install migration requirement. `SecureTokenStorage`, `secureTokenStorageProvider`, the generated `StoredSession` mapper/bootstrap registration, and `AuthSession.setSignedOut()` have no production caller; their remaining consumers are legacy-shaped tests. | `app/lib/auth/providers/secure_token_storage.dart:126`; `app/lib/auth/providers/auth_session_provider.dart:25`; `app/lib/auth/models/stored_session.dart:2`; `app/lib/bootstrap.dart:280`; FR-001 | Delete the legacy storage/provider and compatibility method, make `StoredSession` a plain model, remove its mapper output/bootstrap registration, and update tests to use the registry storage boundary. No requirement relaxation is needed. |
| IR-008 | Suggestion | Code Quality | Several abstractions encode no behavior: `AccountActivationSource` is accepted but never read; `AccountSwitcherAction.actions` is test-only; `addAccountHelper` stores an English sentinel merely to choose already-localized UI copy; and three registry-provider methods duplicate the existing serialized `_mutate` pipeline. Registry mutations also repeat full-object reconstruction extensively. | `app/lib/auth/providers/account_activation_coordinator.dart:6`; `app/lib/auth/models/account_switcher_state.dart:7`; `app/lib/auth/providers/session_registry_provider.dart:19`; `app/lib/auth/models/session_registry.dart:270` | Remove the inert source/action/helper state, route all ordinary registry writes through one serialized mutation helper, and add a private explicit copy/rebuild helper for immutable registry updates. Preserve the lease and generation semantics. |
| IR-009 | Suggestion | Risk | The verified two-slot journal, tolerant per-entry repair, persisted revision/counter repair, and cleanup-only credential queue provide strong interruption recovery, but they account for much of the registry complexity. Because local sessions are recoverable credentials rather than user data, a simpler fail-closed policy is reasonable if offline sign-out is also relaxed. | `app/lib/auth/models/session_registry.dart:39`; `app/lib/auth/providers/secure_token_storage.dart:46`; `app/lib/auth/models/pending_session_cleanup.dart:5`; `app/lib/notifications/services/notification_sign_out_recovery.dart:7`; FR-001, FR-016, FR-026, NFR-002 | Recommended requirements relaxation: sign-out succeeds only after AppView confirms logout (keep the account and show retry on network failure); treat authoritative `401` as locally removable; store one versioned secure snapshot; on corrupt/unreadable storage, fail signed out and invalidate the provider token best-effort. Then remove the A/B journal, read-back verification, partial-entry recovery, pending cleanup credential, and recovery coordinator. Do not simplify only half of this package. |
| IR-010 | Suggestion | Behavior | Startup eagerly validates every inactive session with a concurrency coordinator and a separate ownership launch guard. Inactive accounts are already exercised by fixed clients during notification registration/count work and can be authoritatively invalidated on `401`; proactive `whoami` mostly makes expired rows disappear earlier. | `app/lib/auth/services/session_validation_coordinator.dart:17`; `app/lib/auth/providers/auth_session_provider.dart:19`; FR-024, AC-027 | Recommended requirements relaxation: validate the active account at startup and validate inactive accounts lazily when selected or first used. Remove the inactive worker pool and ownership launch guard. Accept that an expired inactive account can remain visible until use. |
| IR-011 | Suggestion | Behavior | Numeric unread badges for every inactive switcher row require account-family network state, independent refresh behavior, failure isolation, and foreground-recipient refresh wiring. They are useful but not necessary for account switching or safe notification opens. | `app/lib/auth/models/account_switcher_state.dart:52`; `app/lib/router/app_shell.dart:269`; `app/lib/notifications/providers/notification_new_count_provider.dart`; FR-020, AC-023, AC-031 | Recommended requirements relaxation: keep the active account's normal navigation badge, but omit counts from inactive switcher rows and fetch notifications after activation. Remove the inactive-count family behavior and its switcher/runtime wiring. |
| IR-012 | Suggestion | UI | The full-screen identity transition uses a dedicated transition model, provider, overlay, and publication callback. Its `AccountActivationSource` distinction is unused, and notification-triggered activation currently supplies a no-op publisher, so the richer barrier is not consistently part of the account boundary. The actual isolation comes from durable activation, provider invalidation, fixed clients, and generation fences. | `app/lib/auth/providers/account_transition_provider.dart:7`; `app/lib/auth/widgets/account_transition_overlay.dart:8`; `app/lib/auth/providers/account_activation_coordinator.dart:20`; NFR-001, NFR-003, AC-008 | Optional requirements relaxation: keep the switcher open and disable its actions while activation commits, then navigate to Home; remove the global identity overlay/transition provider and unused source enum. Preserve the lease checks, invalidation, and stale-result fences. |
| IR-013 | Suggestion | UI | Persisting display name/avatar identity and hydrating the active profile solely for the Profile destination and switcher adds storage fields, a lease-fenced identity provider, image fallback behavior, and broad widget coverage. Cached handles already provide a stable fallback identity. | `app/lib/auth/providers/active_account_identity_provider.dart:19`; `app/lib/auth/models/stored_session.dart:18`; `app/lib/auth/models/session_registry.dart:480`; FR-005, FR-021, FR-022 | Optional product relaxation: use cached handles and generic account icons in the switcher/Profile destination/recipient line. Remove cached profile metadata and the eager active-identity hydration path. Keep this richer identity UI if visual account recognition is important; its complexity is internally coherent. |

## Requirement And Test Traceability

- Requirements implemented: All currently approved Must requirements and linked acceptance criteria are represented in the implementation plan. IR-009 through IR-013 identify explicit requirement changes that must be agreed before removing their implementations.
- Tests implemented: IR-005 is covered by `UT-004`, `IT-003`, and `REG-004` production-shape read/error/rollback evidence. IR-006 is covered by `AT-003`, `UT-018`, `IT-011`, and `REG-002` responsive interaction, selected-state, keyboard, and fallback evidence. Earlier registry, OAuth, fixed-client, notification, sign-out/recovery, privacy, and AppView contract tests remain recorded in `05-implementation-plan.md`.
- Unplanned behavior: None identified. No dependency, migration, lexicon, new AppView route, linked-account API, bulk logout, inactive direct removal, per-account navigation history, or OS-visible recipient-copy change was added.
- Remaining gaps: `MAN-001` through `MAN-003` require supported physical devices, provider delivery, platform secure-storage inspection, and active assistive technology before release sign-off. The simplifications above have not been approved or implemented.

## Test Evidence

- Commands reviewed: Recent commit history and diff statistics; current source and call-site inspection; workflow requirements, acceptance tests, coding plan, implementation plan, and prior review; previously recorded `dart analyze`, Flutter, Go, generated-output, and diff-check evidence.
- Passing evidence: The prior correction re-review reran the corrected boundary and switcher suites (8 tests), `dart analyze`, and `git diff --check`; all passed. The finalized implementation record reports 24 focused boundary/interaction tests, 5 switcher tests, 108 expanded provider/router regressions, all 910 Flutter tests, and the complete race-enabled Go suite passing. Generated outputs were current at commit time.
- Failing or skipped tests: No automated failure is known. Tests were not rerun for this documentation-only complexity review. `MAN-001` through `MAN-003` were not run for the documented environment reasons and remain required before production release.

## Risk Review

- Risk level: High
- Risk notes: Authentication credentials, account-scoped state, notification recipient activation, and shared-installation cleanup remain intrinsically high-risk. The implemented fixed-session clients, generation ownership checks, verified two-slot journal, exact routing resolution, quarantine ordering, sentinel redaction tests, and production-shape stale-completion coverage address the identified code risks.
- Approval notes: Approved with non-blocking simplification notes. The implementation may remain as-is. If simplifying, change the relevant requirements and acceptance tests first, then remove behavior test-first. Complete the three documented manual checks before release if the existing implementation is retained. No commit, push, or pull request was created by this review.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: This pass concerns structural and product-scope simplification. Polishing UI that may be removed would be premature.
- Suggested polish notes: Decide IR-011 through IR-013 before any visual polish pass.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: If simplification is approved, start with IR-007 because it has no behavior tradeoff. For the larger package, first revise FR-001, FR-016, FR-026, NFR-002, and their acceptance tests, then write the new online-sign-out and fail-closed-storage tests before removing recovery code.
- Verification to rerun: After any simplification, regenerate outputs, run `dart analyze`, affected focused Flutter suites, complete `flutter test --reporter compact`, `just test`, and `git diff --check`. Before release of retained behavior, complete `MAN-001` through `MAN-003`.

## Post-Approval Field Verification

Two field reports exposed narrow integration gaps after the approval above. Both were corrected test-first on 2026-07-18 without changing the verdict:

- Initial sign-in now hydrates the active account's own profile identity from the shell, so the Profile destination updates without requiring a visit to Profile. The update is fenced to the captured session lease.
- A signed-in `/auth/complete` callback now reaches `AuthCompletePage`; the production controller retains A, activates B, and navigates successfully to B's Home or onboarding route.

The new end-to-end widget tests passed along with all 73 auth tests, all 21 router tests, all 912 Flutter tests, `dart analyze`, generated-output verification, the race-enabled Go suite, and `git diff --check`. No new correctness finding was identified; the later complexity audit changes the current verdict to Approved with notes. `MAN-001` through `MAN-003` remain required before release.
