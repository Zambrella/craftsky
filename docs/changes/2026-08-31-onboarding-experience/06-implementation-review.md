# Implementation Review: Onboarding Experience

## Verdict
Status: Changes required
Reviewer: OpenCode
Date: 2026-08-31
Risk level: Medium

## Summary
The implementation now covers the three-step flow, server-backed completion, optimistic retry, profile/craft persistence, shared Instagram content, startup gating, and OAuth-time Bluesky projection. The full Flutter suite and focused AppView packages pass. It is not ready for handoff because several Must behaviors remain incorrect at loading/account boundaries, and the approved high-risk route, OAuth, cleanup, and acceptance tests are not fully implemented. The release-equivalent AppView gate is also incomplete because an unrelated fixed-clock credential-cleanup test fails before migration rollback/reapply.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior | Skip is absent while profile prefill is loading or has failed. The page renders a spinner or retry-only scaffold instead of the explicit flow exit, although Skip may be disabled only while a submitted profile save is in flight. | FR-010, FR-022; AC-003, AC-027; AT-007, AT-016; `app/lib/onboarding/pages/onboarding_page.dart:29-50`, `:93-107` | Keep an account-scoped Skip action available during prefill loading/error and add widget coverage for both states. |
| IR-002 | Important | Behavior | The five-second Bluesky prefill bound counts only fixed retry delays, not elapsed request time. Slow successful reads can keep initialization blocked beyond five seconds. The test replaces all delays with zero and does not verify an elapsed-clock deadline. | FR-021; AC-035, AC-036; UT-003, AT-016; `app/lib/onboarding/providers/onboarding_flow_provider.dart:38-46`; `app/test/onboarding/onboarding_profile_prefill_test.dart:22-67` | Enforce a true elapsed deadline including request duration, inject the clock/deadline seam, and test slow responses plus the five-second stop. |
| IR-003 | Important | Behavior | The shared Instagram import composer watches import state but leaves create controls enabled while the provider is initially loading or failed. A create can race the initial list and be overwritten by its late result, contrary to the explicit coding-plan guard. | FR-008, FR-015; AC-024; AT-009, IT-007; CPQ-006; `app/lib/instagram_migration/pages/instagram_migration_page.dart:592-656`; `app/lib/instagram_migration/providers/instagram_imports_provider.dart:13-20`, `:63-77` | Gate import creation on provider readiness, expose retry/error behavior, and add a composer-without-history race test. |
| IR-004 | Important | Behavior | Router redirect logic uses any available initialization value without verifying that its lease matches the registry's exact active lease. Retained account-A completion can transiently redirect account B before B initializes. Existing gate checks protect content but not route mutation. | FR-018, FR-020, NFR-003, RULE-003; AC-021, AC-034; AT-013, AT-015, UT-006; `app/lib/router/router.dart:124-140` | Require exact active-lease equality before redirects and add switch/re-auth tests with stale resolved initialization plus an onboarding-status failure Retry test. |
| IR-005 | Important | Tests | The declared Instagram, account-isolation, and atomic-profile acceptance IDs are represented by narrow render/unit cases rather than the approved observable scenarios. Missing cases include import actions/readiness, suggestion follow/dismiss/load-more, linked/reactivation parity, completion/Instagram stale operations, profile-read/avatar stale operations, and real identity-only/crafts-only update preservation. | AT-009-AT-012, AT-015, AT-017; IT-003, IT-006, IT-007; `app/test/onboarding/onboarding_instagram_step_test.dart`, `onboarding_account_isolation_test.dart`, `onboarding_profile_payload_test.dart` | Add the omitted observable scenarios or document a justified test gap for each acceptance criterion. |
| IR-006 | Important | Tests | AppView tests do not establish the approved authenticated route contract or OAuth-before-handoff path. The route test checks policy metadata only; real-flow tests inject a nil projector; canonical replay invokes the OAuth adapter twice rather than delivering the second event through Tap. | IT-001, IT-002, AT-019, IT-009, REG-008; AC-030, AC-031, AC-041-AC-043; `appview/internal/routes/onboarding_route_test.go:5-21`; `appview/internal/app/federated_real_flow_integration_test.go:2079-2088`, `:2432-2441`; `oauth_bluesky_profile_projection_test.go:48-52` | Add full-mux auth/device/body/query/lifecycle tests, POST/envelope cases, and a real projector-before-recording-handoff plus Tap replay integration test. |
| IR-007 | Important | Tests | The migration/private-cleanup contract lacks feature-specific evidence. Cleanup tests do not seed/assert onboarding rows, and store tests hand-create the table instead of validating migration 000065. `just appview-check` then fails on an unrelated fixed-clock credential expiry before down-to-zero/reapply. | FR-018, RULE-003, RULE-004; IT-005; `appview/internal/accountdeletion/private_cleanup_test.go:97-166`; `appview/internal/api/onboarding_test.go:26-44`; `appview/internal/auth/session_cleanup_processor_test.go:115-140` | Extend Alice/Bob cleanup and migration tests, correct or isolate the unrelated fixed-clock gate failure, then rerun `just appview-check` through rollback/reapply. |
| IR-008 | Suggestion | Accessibility | A saving primary action replaces its text with an unlabeled progress indicator, so assistive technology loses the action purpose and receives only a generic loading announcement. | NFR-002; AC-020; AT-014; `app/lib/onboarding/widgets/onboarding_bottom_action.dart:18-38` | Preserve the localized action label in busy semantics and test label, busy, and disabled state together. |
| IR-009 | Suggestion | Code Quality | Generated files outside the feature contain hash/EOF-only churn, and twelve mapper files make `git diff --check` fail with new blank lines at EOF. | Coding-plan generated-output guardrail; `app/lib/feed/**.mapper.dart`, `app/lib/moderation/**.mapper.dart`, `app/lib/shared/widgets/post_summary.mapper.dart` | Regenerate deterministically or remove unrelated generated churn without hand-editing generated files. |
| IR-010 | Suggestion | Observability | Completion read/write failures return redacted envelopes but emit no non-sensitive diagnostic log as required by the requirements observability section. | `01-requirements.md` section 18; `appview/internal/api/onboarding.go:55-58`; `appview/internal/routes/routes_onboarding.go:5-9` | Inject the route logger and log operation/category/request ID without DID, timestamp, or raw error data. |

## Requirement And Test Traceability
- Requirements implemented: Core three-step UI, optional completion, server authority, optimistic retry, profile/craft drafts and saves, bounded-retry structure, shared Instagram composition, localization, notification readiness, private persistence, cleanup registration, and eager canonical OAuth projection.
- Tests implemented: 1,637 Flutter tests pass, including focused onboarding action/progress/draft/profile/widget suites. Focused AppView auth, app, API, routes, index, cleanup, lifecycle, and migration packages pass.
- Unplanned behavior: None identified.
- Remaining gaps: IR-001 through IR-007 are blocking Must behavior/evidence gaps. MAN-001 and MAN-002 remain explicitly deferred.

## Test Evidence
- Commands reviewed: Focused Flutter onboarding/profile/Instagram suite; full `flutter test`; Dart analysis; focused Go package tests; `just appview-check`; `git diff --check`.
- Passing evidence: 108 focused Flutter tests, 1,637 full Flutter tests, clean Dart analysis, and focused AppView packages pass. Database-backed onboarding store/projection/inventory tests executed before the release gate stopped.
- Failing or skipped tests: `just appview-check` fails at `TestProviderRegistrationCredentialCleanupConverges` because its fixed timestamp violates `oauth_unverified_credentials_expiry_check`; migration down/reapply therefore did not execute. MAN-001 and MAN-002 are deferred. `git diff --check` reports generated EOF whitespace.

## Risk Review
- Risk level: Medium.
- Risk notes: Exact-lease routing, optional-flow escape, private completion authorization/cleanup, atomic profile writes, and OAuth projection remain high-consequence boundaries despite broad green unit coverage.
- Approval notes: Approval is blocked by IR-001 through IR-007. IR-008 through IR-010 are non-blocking quality follow-ups once required corrections are green.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: The current visual structure is coherent. Remaining visible issues are primarily behavioral and accessibility findings that belong in TDD correction, not polish.
- Suggested polish notes: After required fixes, visually check sticky-action spacing, long Instagram content, upload/error transitions, keyboard insets, and large-text focus order.

## Handoff Back To TDD Builder
- Required fixes: IR-001 through IR-007.
- Suggested next failing test: Add AT-007 coverage proving Skip remains available during prefill loading/error, then UT-003 coverage proving slow requests cannot exceed the five-second elapsed bound.
- Verification to rerun: New focused tests, full Flutter suite, Dart analysis, full AppView route/OAuth/cleanup integration tests, `git diff --check`, and `just appview-check` through migration rollback/reapply.
