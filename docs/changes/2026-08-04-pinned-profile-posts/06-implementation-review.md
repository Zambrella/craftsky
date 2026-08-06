# Implementation Review: Pinned Profile Posts

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-08-05
Risk level: Medium

## Summary

The implementation satisfies the approved AppView-only two-slot design and is ready for handoff from an automated correctness perspective. The private schema, owner-scoped transactional mutations, target-specific stale-safe unpinning, structural cleanup, policy-aware profile promotion, pin-bound cursor invalidation, authoritative Flutter state, account fencing, approved owner-action surfaces, and profile-only attribution all match the requirements. The implementation introduces no payment, plan, tier, entitlement, access-gating, PDS, lexicon, Tap, public-post, or non-profile ranking behavior.

No blocking behavior, migration, privacy, pagination, state-management, or accessibility defect was identified. The notes below concern test traceability, one Should-level observability edge, and the two already-documented manual UI gates.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Suggestion | Tests / Traceability | The approved-surface behavior is covered compositionally rather than with every concrete AT-002 and AT-012 example. The current tests prove the shared `PostCard` type/eligibility matrix, a standard timeline action, a standard thread-root action, profile project attribution, and default-off presentation. They do not directly drive the specified top-level quote timeline action, project thread-root action, project Projects-tab action, or each named negative surface. Code inspection confirms the generic opt-in/default-off wiring is correct, so this does not block approval, but `05-implementation-plan.md` describes the acceptance scenarios more completely than the literal widget coverage. | FR-001, FR-009, RULE-002, RULE-003; AT-002, AT-012, REG-002; `app/test/feed/feed_page_test.dart`; `app/test/feed/pages/post_thread_page_test.dart`; `app/test/profile/widgets/profile_projects_tab_test.dart`; `app/test/feed/widgets/post_card_test.dart` | None for approval. If exact scenario-level traceability is desired, add focused widget cases for the missing positive shape/surface combinations and one actual default-off surface matrix test. |
| IR-002 | Suggestion | Risk / Code Quality | Another-author PUT/DELETE requests are rejected in the handlers before `ProfilePinStore`, while the feature-specific bounded observer is invoked only by store mutations. These production rejections still receive generic HTTP telemetry, but they do not emit the proposed profile-pin operation/slot/result/error-class observation. Store-level tests exercise the bounded `forbidden` class, so the test does not expose this handler-path distinction. NFR-005 is Should-level and the omission does not affect behavior or privacy. | NFR-005, AC-019, IT-014; `appview/internal/api/profile_pin.go`; `appview/internal/api/profile_pin_store.go`; `appview/internal/api/profile_pin_observability_test.go` | None for approval. Optionally route handler-prevalidated rejections through a bounded observer seam and protect it with a handler-level telemetry test. |
| IR-003 | Suggestion | Tests / Risk | Physical VoiceOver/TalkBack, keyboard/focus behavior, and final light/dark/narrow/maximum-text-scale visual parity have not been run. Automated semantics and large-text widget coverage pass, and the implementation log correctly leaves these as external manual gates. | BR-002, FR-009, NFR-004, NFR-006; AC-010, AC-018, AC-020; MAN-001, MAN-002; `app/test/feed/widgets/post_card_test.dart`; `05-implementation-plan.md` | Run MAN-001 and MAN-002 before declaring device-level UI verification complete; record any result or follow-up separately. |

## Requirement And Test Traceability

- Requirements implemented: BR-001–BR-003, FR-001–FR-013, NFR-001–NFR-006, and RULE-001–RULE-004 are represented in the implementation. NFR-005 has the non-blocking handler-path telemetry note in IR-002.
- Tests implemented: IT-001–IT-016, UT-001–UT-009, AT-001–AT-012, and REG-001–REG-008 have direct or explicitly composite automated evidence. IR-001 records where the acceptance examples are proven through shared seams rather than literal end-surface combinations.
- Unplanned behavior: None material. Cursor helpers were kept beside the existing profile-list handler in `post.go` instead of the coding plan's proposed separate `profile_pin_cursor.go`; this is a localized file-placement deviation with no contract or behavior change. The shared 4xx diagnostic mapping change is linked to UT-009 and redacts 5xx server text and dynamic endpoint identifiers.
- Remaining gaps: MAN-001 and MAN-002 are pending. IR-001 and IR-002 are optional hardening work, not required corrections.

## Test Evidence

- Commands reviewed:
  - Current review rerun: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:16430/craftsky_dev?sslmode=disable go test ./internal/db ./internal/api ./internal/index ./internal/routes -run 'ProfilePin|ProfilePins|Pinned' -count=1`.
  - Current review rerun: `flutter test` across the pin page/model/client/provider, standard/project pagination providers, shared card, feed, thread, and profile-tab targets.
  - Implementation evidence: full AppView `go test ./... -count=1` against real Postgres; full Flutter `flutter test` with 1,311 tests; `dart analyze`; formatting and `git diff --check`.
- Passing evidence: All four focused AppView packages passed during review. The current Flutter review selection passed 168 tests. The implementation log records 176 tests in its broader consolidated selection, 1,311 tests in the full Flutter run, a passing full AppView run, clean static analysis, and clean formatting/whitespace checks. The final client shape-classifier refinement was separately rerun with its focused `PostCard` tests and static analysis.
- Failing or skipped tests: No failing automated test is recorded or reproduced. MAN-001 and MAN-002 were skipped and remain explicit manual gates. A full-suite rerun was not repeated during this review because the implementation stage had already run it and the current focused suites cover the final changed feature paths.

## Risk Review

- Risk level: Medium.
- Risk notes: The material risks remain owner isolation, transaction ordering, permanent-versus-temporary lifecycle handling, policy-before-limit pagination, cursor invalidation, shared-card presentation leakage, and stale active-account completion. Real-Postgres integration tests, deterministic transaction barriers, query-plan/privacy sentinels, Flutter provider tests, explicit default-off card inputs, and current focused reruns provide proportionate coverage. Manual device accessibility and visual rendering remain the only unverified user-facing risk.
- Approval notes: The implementation preserves private AppView ownership and the exact bodyless authoritative `200 OK` API contract. It does not modify `lexicon/`, PDS data, public post models, `profile_sort_at`, notifications, interaction counts, search/discovery ranking, or commercial/access behavior.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The implemented row reuses the repost attribution position, typography, spacing, subdued colour, and informational semantics, and automated narrow-width/large-text coverage passes. No visible polish defect is established from code or widget tests, but a small polish pass may be useful if MAN-002 reveals device-specific spacing, wrapping, contrast, or visual-state differences.
- Suggested polish notes: Keep any pass limited to icon alignment, spacing, colour/contrast, text wrapping, and semantics. Do not change pin eligibility, mutation behavior, pagination, API state, or copy.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: If taking the optional observability hardening, add a handler-level test proving another-author PUT and DELETE rejections emit one bounded `forbidden` profile-pin observation without a DID, URI, or state value. If taking the optional traceability hardening, start with a Projects-tab owner action test that pins a project and proves the standard slot remains unchanged.
- Verification to rerun: Run the focused AppView profile-pin suites and the affected Flutter widget/provider targets after any optional change. Rerun `dart analyze`, `git diff --check`, and the full relevant suite before merge. Run and record MAN-001 and MAN-002 for complete device-level UI verification.
