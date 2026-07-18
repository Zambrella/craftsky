# Implementation Review: Multi-Account Sessions And Notification Routing

## Verdict

Status: Approved
Reviewer: Codex
Date: 2026-07-18
Risk level: High

## Summary

The second correction pass closes IR-005 and IR-006, and the implementation is ready for handoff. The account boundary now invalidates the approved feature-provider inventory and fences asynchronous completion by DID, session generation, and activation generation. A production-shape test starts timeline pagination, a user-post read, and an optimistic like under A, activates B through the real registry/coordinator/invalidator, then proves late A success, error, and rollback cannot publish into B. The prior production-Dio test separately proves the rebuilt active client carries only B's bearer.

The responsive Profile destination now preserves ordinary navigation while sharing long-press, semantics, and Alt+Down account-switch activation. Tests cover the compact bottom sheet, large anchored menu, selected avatar state and semantics, successful avatar rendering, failed-image fallback, missing-image fallback, and real account selection. No separate switch button or ordinary-navigation prompt was introduced.

The complete implementation remains aligned with the approved device-local session-registry architecture, independent AppView sessions, account-scoped notification routing and cleanup, privacy/redaction boundaries, and explicit non-goals. The remaining physical-device/platform checks are documented pre-release gates, not missing implementation evidence.

## Findings

None identified.

## Requirement And Test Traceability

- Requirements implemented: All approved Must requirements and linked acceptance criteria are represented in the implementation plan. The second correction specifically closes `FR-007` through `FR-009`, `NFR-001`, `FR-004`, `FR-006`, `FR-022`, and `NFR-004` against `AC-008`, `AC-009`, `AC-006`, and `AC-025`.
- Tests implemented: IR-005 is covered by `UT-004`, `IT-003`, and `REG-004` production-shape read/error/rollback evidence. IR-006 is covered by `AT-003`, `UT-018`, `IT-011`, and `REG-002` responsive interaction, selected-state, keyboard, and fallback evidence. Earlier registry, OAuth, fixed-client, notification, sign-out/recovery, privacy, and AppView contract tests remain recorded in `05-implementation-plan.md`.
- Unplanned behavior: None identified. No dependency, migration, lexicon, new AppView route, linked-account API, bulk logout, inactive direct removal, per-account navigation history, or OS-visible recipient-copy change was added.
- Remaining gaps: `MAN-001` through `MAN-003` require supported physical devices, provider delivery, platform secure-storage inspection, and active assistive technology before release sign-off.

## Test Evidence

- Commands reviewed: `dart run build_runner build`; `dart analyze`; focused IR-005 and switcher suites; expanded feature-provider/router regressions; complete `flutter test --reporter compact`; repository `just test`; `git diff --check`; and forbidden-scope path inspection.
- Passing evidence: This re-review reran the corrected boundary and switcher suites (8 tests), `dart analyze`, and `git diff --check`; all passed. The finalized implementation record reports 24 focused boundary/interaction tests, 5 switcher tests, 108 expanded provider/router regressions, all 910 Flutter tests, and the complete race-enabled Go suite passing. Generated outputs are current.
- Failing or skipped tests: No automated test fails. `MAN-001` through `MAN-003` were not run for the documented environment reasons and remain required before production release.

## Risk Review

- Risk level: High
- Risk notes: Authentication credentials, account-scoped state, notification recipient activation, and shared-installation cleanup remain intrinsically high-risk. The implemented fixed-session clients, generation ownership checks, verified two-slot journal, exact routing resolution, quarantine ordering, sentinel redaction tests, and production-shape stale-completion coverage address the identified code risks.
- Approval notes: Approved for merge or handoff from implementation review. Complete the three documented manual checks before release. No commit, push, or pull request was created by this review.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The responsive switcher behavior and accessibility contracts are coherent and automated. No visible issue blocks approval, but the feature adds enough new UI that a small visual pass could still be useful.
- Suggested polish notes: Inspect anchored-menu placement, compact-sheet spacing, selected-avatar treatment, badge spacing, transition copy, and screen-reader phrasing on supported form factors without changing behavior.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None; the implementation is approved. Any future behavior change should begin from its own linked requirement and failing test.
- Verification to rerun: Before release, complete `MAN-001` through `MAN-003`. Before merge after any further code change, rerun generated output, `dart analyze`, affected focused Flutter suites, complete `flutter test --reporter compact`, `just test`, and `git diff --check`.

## Post-Approval Field Verification

Two field reports exposed narrow integration gaps after the approval above. Both were corrected test-first on 2026-07-18 without changing the verdict:

- Initial sign-in now hydrates the active account's own profile identity from the shell, so the Profile destination updates without requiring a visit to Profile. The update is fenced to the captured session lease.
- A signed-in `/auth/complete` callback now reaches `AuthCompletePage`; the production controller retains A, activates B, and navigates successfully to B's Home or onboarding route.

The new end-to-end widget tests passed along with all 73 auth tests, all 21 router tests, all 912 Flutter tests, `dart analyze`, generated-output verification, the race-enabled Go suite, and `git diff --check`. No new finding was identified, so the implementation remains Approved. `MAN-001` through `MAN-003` remain required before release.
