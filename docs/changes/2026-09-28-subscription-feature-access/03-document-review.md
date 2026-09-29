# Document Review: Subscription Feature Access

## Verdict

Status: Approved with notes
Reviewer: AI document review
Date: 2026-09-28
Risk level: High

## Summary

The revised requirements and acceptance-test specification are consistent and ready for coding planning. Subsequent coding-plan feedback also requires dropping the unused account-type table and hiding every Business owner option/management route when the DID lacks Business access. FR-008 and AC-019 cover migration and derived classification; FR-004/AC-009 cover owner-route denial. Public reads of licensed business content remain available to visitors of any tier. The retired profile-border choice is excluded; interleaved publication and pin-cleanup tests cover the high-risk races.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Requirements / Tests | Resolved: the retired border choice had been named in a paid-customisation criterion. The revised criterion now covers current colour/background choices only; the test spec marks GAP-001 resolved. | `01-requirements.md` Q3, AC-015; `02-acceptance-tests.md` AT-005, GAP-001; `appview/migrations/000070_profile_customisation_remove_border.up.sql` | None for coding planning; do not reintroduce the border as part of this work. |
| DR-002 | Important | Requirements / Tests | Resolved: the product owner superseded the earlier suggestion to protect the manual switch: it must be removed, including its API mutation. AC-019 and AT-003/IT-006 now require the former mutation to be unavailable to all tiers, including a Business DID attempting to switch to `regular`; the account-type UI is absent. | `01-requirements.md` Q4, FR-008, AC-019; `02-acceptance-tests.md` AT-003, IT-006 | Coding plan should inventory all route registrations, UI toggle entry points, and legacy account-type read dependencies. |
| DR-003 | Important | Tests / Risk | Resolved for test design: IT-011 now interleaves worker claim, loss of access, and the final pre-PDS boundary; IT-009 replays cleanup and tests pins cannot return. Concurrency safety still depends on the implementation's actual final authorization boundary. | `01-requirements.md` FR-006, FR-007, RISK-004, RISK-006; `02-acceptance-tests.md` IT-009, IT-011, GAP-003 | Coding plan must identify the final access recheck and document any unavoidable race before external side effects. |
| DR-004 | Suggestion | Tests / API | Resolved by product-owner feedback: without effective Business access, all Business owner-management reads, mutations and pages are unavailable; Business source records remain intact on the PDS. Scheduled-post owner reads remain available for the required error state. | `01-requirements.md` Q7, FR-004/FR-005, AC-009/AC-010; `02-acceptance-tests.md` AT-008, IT-007, GAP-002 | Gate every Business owner route and settings entry, but preserve public visitor reads when the business owner is licensed. |
| DR-005 | Suggestion | Traceability / Risk | Resolved for planning: `01-requirements.md` now records product-owner approval for test design and the subsequent scope clarification. Final approval of the high-risk document set is still required before implementation. | `01-requirements.md` §22–23; `02-acceptance-tests.md` §11; conversation | Seek explicit approval of the final documents/coding plan before starting implementation. |
| DR-006 | Important | Data / Tests | Resolved by product-owner feedback: the testing-only `craftsky_account_types` table is to be dropped, not retained inertly. A forward migration is required after replacing its query and account-deletion references. | `01-requirements.md` FR-008, AC-019, §15; `02-acceptance-tests.md` IT-012 | Plan new up/down migration and tests for no live SQL dependency, successful account deletion, and intact PDS business source. Do not rewrite historical migration files. |

## Traceability Review

- Planning to requirements: `00-direction.md` gave the account-assigned licence boundary, private billing state, free social baseline, and no-paid-reach rule. Later user decisions supersede its provisional follower-growth allocation, self-declared business type, temporary storage and Business-section availability. These decisions are reflected in BR-001, FR-004, FR-008, RULE-001 and the current non-goals.
- Requirements to acceptance criteria: Every Must BR-001–BR-002, FR-001–FR-008, NFR-001–NFR-002, RULE-001–RULE-003 references at least one testable AC. AC-009/AC-010 distinguish hidden Business owner routes from public reads of licensed businesses; AC-015 matches the current colour/background catalogue; AC-019 checks toggle/mutation/table removal and derived status.
- Acceptance criteria to tests: AC-001–AC-020 each map to tests. Every AT/UT/IT/REG/MAN references requirement and AC IDs. IT-007 checks owner-route denial while public visitor reads remain available for licensed businesses; IT-012 verifies migration and removal of legacy dependencies. The critical lapse, account-switch, direct API, profile projection, and scheduling paths have integration coverage; MAN-001 only supplements automated accessibility checks.

## Coverage Review

- Must requirements covered: All mapped to automated tests, with a complementary manual platform accessibility check.
- Missing or weak coverage: No blocking gap. GAP-002 is resolved by the owner's route-hiding decision; GAP-003 remains an implementation race risk with explicit interleaving tests designed.
- Manual-only coverage: None.

## Risk And Approval Review

- Risk level: High. Changes affect existing free capabilities, public identity projections, and asynchronous publication.
- Review requirement: The product owner approved the earlier requirements for test design, explicitly requested revision/re-review, and supplied further coding-plan feedback to hide unlicensed Business owner routes and drop unused account-type storage. Coding planning may proceed on the aligned documents. Explicit approval of the final high-risk document set and coding plan remains required before implementation.
- Approval notes: The further feedback is incorporated as scope guidance, not approval to start coding.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Design AppView DID-scoped authorization; enumerate every business account-type read/write/UI and lifecycle cleanup reference before dropping the table, then design scheduled publication's final access check and lapse-state transitions.
- Blocking issues: None for coding planning. Do not start implementation until final high-risk approval is obtained.

## Notes For Next Stage

- Keep manual business account-type mutation unavailable to **all** tiers; the effective public type derives from the assigned Business licence. Drop the testing-only table in a new migration after removing all live references and cleanup hooks; never delete public business PDS records as a side effect.
- Preserve the distinction between subscription cancellation while `gives_access=true` and actual access loss. Only the latter clears pins or hides Business content.
- During lapse, deny Business owner routes and hide their settings options; do not hide the owner's scheduled-post status/error page (Plus behavior). Ordinary visitor reads of a licensed business remain available irrespective of the visitor's tier.
- Make the final pre-PDS publication access check and the worker-claim race explicit; use AT-009/IT-011 as acceptance targets.
