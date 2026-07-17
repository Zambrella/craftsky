# Implementation Review: Direct Push Notification Routing

## Verdict

Status: Superseded — re-review required

Reviewer: Codex

Date: 2026-07-17

Risk level: High

Post-review note: Manual Android and in-app notification testing found two additional defects after this approval. The TDD correction and evidence are recorded as BUG-001 and BUG-002 in `05-implementation-plan.md`; the current diff requires a new implementation review before this approval can be treated as current.

## Summary

The implementation now satisfies the approved direct-routing contract and is ready for merge or handoff from an implementation-review perspective. The versioned minimal payload, provider-neutral parsing and inference, binding-first navigation, reply focus, destination authorization boundary, resolver removal, privacy controls, and regression coverage remain coherent. The correction pass resolved all three original findings with meaningful red-green-refactor evidence and introduced no new finding.

Permanent profile refresh errors now suppress retained data, retryable post/profile refresh errors expose destination-scoped Retry while retaining authenticated content, and in-flight notification opens cannot emit navigation or fallback effects after sign-out, onboarding change, account switch, disposal, or an away-and-back readiness transition.

## Findings

None identified.

### Resolved Findings

| ID | Resolution | Evidence |
|---|---|---|
| IR-001 | Resolved. Named permanent profile errors take precedence over any retained Riverpod value, so cached profile content is not rendered after `profile_not_found`. | FR-009, AC-009; `app/lib/profile/pages/profile_page.dart:78-85`; `app/test/profile/profile_page_test.dart:233` |
| IR-002 | Resolved. Retryable refresh errors render the shared localized Retry state above retained authenticated post/profile content, and Retry invalidates only the current destination provider. | FR-010, NFR-005, AC-010; `app/lib/feed/pages/post_thread_page.dart:161-212`; `app/lib/profile/pages/profile_page.dart:88-108`; refresh tests at `app/test/feed/pages/post_thread_page_test.dart:140` and `app/test/profile/profile_page_test.dart:336` |
| IR-003 | Resolved. Each open captures a readiness revision and emits effects only while the runtime remains ready, undisposed, on the same DID, and on the same revision after asynchronous binding work. | BR-002, FR-006, RULE-003, AC-006; `app/lib/notifications/services/notification_runtime.dart:77-134`; `app/test/notifications/notification_open_flow_test.dart:71` |

## Requirement And Test Traceability

- Requirements implemented: BR-001 through BR-002, FR-001 through FR-019, NFR-001 through NFR-005, and RULE-001 through RULE-006 map to the implemented payload, parser, binding, inference, navigation, destination, cutover, privacy, and regression surfaces. IR-001 through IR-003 close the only gaps found in the first review.
- Tests implemented: The planned UT-001 through UT-015, AT-001 through AT-009, IT-001 through IT-011, and regression surfaces have automated coverage or verified existing suites. The correction pass adds observable refresh-state and in-flight account-transition coverage through public page/runtime interfaces.
- Unplanned behavior: None identified. The readiness revision is narrowly scoped to the approved account-isolation requirement, and retained-content Retry presentation follows the approved coding plan.
- Remaining gaps: MAN-001 through MAN-005 remain documented physical-device/provider release-readiness checks. They are not substitutes for automated coverage and do not block this implementation review.

## Test Evidence

- Commands reviewed: All red-green commands and final verification recorded in `05-implementation-plan.md`; the current diff and status; correction-focused destination/runtime tests; broader notification/destination/router tests; `just app-analyze`; `just app-test`; the unchanged AppView verification recorded by the initial implementation; and `git diff --check`.
- Passing evidence: This re-review reran the post, profile, and notification-open-flow files and passed all 28 tests. The correction pass also passed 26 combined destination tests, 8 runtime/coordinator/effect tests, the 111-test feature gate, clean `just app-analyze`, and `git diff --check`. The unchanged AppView implementation retains its recorded clean `just test` result.
- Failing or skipped tests: Full `just app-test` completed 862 tests with only the same reproducible pre-existing failure in `post_comment_section_page_test.dart` (`wires repost action for the root post`) and no new failures. That test is outside this workflow's requirements and changed paths. MAN-001 through MAN-005 were not run because they require configured provider credentials and physical devices.

## Risk Review

- Risk level: High.
- Risk notes: The provider-data privacy relaxation and account-binding boundary remain high-risk by design, but the data contract is minimal and bounded, destination reads remain authenticated, telemetry remains identifier-free, and readiness revisions now close the asynchronous cross-account gap.
- Approval notes: Approved for merge or handoff from this workflow stage. The unrelated repost-action test failure should continue to be reported separately until fixed, and the five manual checks remain release-readiness gates before physical push behavior is claimed.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: The localized permanent and transient states, accessible labeled actions, retained-content behavior, and route-preserving layout are coherent. No polish-only defect was identified.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None; the implementation review is approved.
- Verification to rerun: No correction rerun is required before handoff. Complete MAN-001 through MAN-005 before release readiness, and continue to report the unrelated repost-action suite failure separately if it remains reproducible.
