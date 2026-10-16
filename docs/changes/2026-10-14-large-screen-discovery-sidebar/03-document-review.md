# Document Review: Large-screen discovery sidebar

## Verdict

Status: Approved with notes

Reviewer: Codex

Date: 2026-10-14

Risk level: Medium

## Summary

The document set is ready for coding planning. It preserves the confirmed direction: preferred-craft links opening the selected Projects tab, followed by five newest eligible CraftSky members in the existing desktop sidebar.

All 13 Must requirements link to acceptance criteria and planned tests. All 11 acceptance criteria have concrete verification paths. No contradictory requirements or blocking product questions were identified. Tests are explicitly planned rather than claimed as implemented or passing.

This verdict approves readiness for coding-plan work. It does not authorize implementation, production changes, deployment, or backfills.

Reviewed inputs: `01-requirements.md`, `02-acceptance-tests.md`, and the preceding conversation. There is no separate initial-prompt artifact; requirements section 1 and conversation preserve the request. Earlier documents are unchanged by this review.

## Findings

All findings below are non-blocking for coding planning.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Risk | The first-onboarding-completion definition is documented and covered, but timestamp immutability and legacy coverage remain unverified. The planner must distinguish an implementation dependency from a completed check. | Requirements sections 15, 19–21; ASM-002, RISK-001; Tests IT-004, GAP-001 | Inspect onboarding persistence and lifecycle behavior during coding planning. Record whether existing metadata meets the approved semantics; if it does not, resolve that discrepancy before implementing member selection. Preserve the omission policy for missing metadata; do not invent join dates or silently add a production backfill. |
| DR-002 | Suggestion | Tests | The behavioral tests are sufficient, but deterministic tie order and the public summary schema are intentionally not fixed yet. IT-001 and IT-006 need precise expected results once interfaces are designed. | Requirements RULE-001, RULE-003; Tests IT-001, IT-005, IT-006, GAP-002 | Choose and document the tie-breaker, authenticated read operation and allowed response fields in the coding plan. Connect the tests to those contracts while retaining stable IDs. |
| DR-003 | Suggestion | Traceability | Requirements section 22 still says Draft and not approved, despite the subsequent user approval recorded in the test specification. This is a historical status mismatch, not missing authorization for planning. | Requirements section 22; Tests sections 1 and 11; user message “LGTM. Move to $write-acceptance-tests” | Use the recorded user approval as the planning baseline. Update the earlier status only during an authorized document revision; this review does not rewrite it. |
| DR-004 | Suggestion | Tests | Several scenario IDs intentionally group variants. Implementing only the headline Gherkin would miss mounted-page reselection, both async completion orders, and some cardinality/exclusion cases. | Tests AT-003, AT-006, AT-007, IT-002, IT-003 | Carry the accompanying variant notes and table setups into the implementation checklist. Keep expected identities/query values independent of production selection logic. Use separate parameterized cases where a single failure would otherwise hide the cause. |

## Traceability Review

- Planning to requirements: The user's chosen craft links and newest members map to BR-001/BR-002 and FR-002/FR-003/FR-005. Two-section order, profile-row presentation and empty craft editing come from the accepted preceding proposal. Deferred projects, drafts, conversations and ranking are explicit non-goals.
- Requirements to acceptance criteria: All 13 Must requirements have at least one AC reference; all AC references resolve. AC-001–AC-011 link back to relevant requirements.
- Acceptance criteria to tests: Each AC appears in the coverage matrix and specific scenario/table entries. All AT, UT, IT, REG and MAN cases name requirement and AC IDs. EC-001–EC-007 have test paths through loading, empty craft results, invalid navigation input, empty member results, unavailable profiles, late responses and constrained layouts.
- Assumptions: Existing sidebar scope, first completed onboarding, eligibility including followed members, and absence of polling are carried consistently into the test specification. User approval of the requirements supplies the baseline; verification of membership metadata is still outstanding.

## Coverage Review

- Must requirements covered: BR-001–BR-002, FR-001–FR-007, NFR-001, RULE-001–RULE-003 — 13 of 13 with planned coverage.
- Acceptance criteria covered: AC-001–AC-011 — 11 of 11 with planned coverage.
- Missing or weak coverage: No missing Must or AC coverage identified. GAP-001/GAP-002 identify unresolved data/interface details rather than unassigned behavior. DR-004 calls out the need to implement all specified variants.
- Test levels: Router/widget tests observe selected tabs and real project query inputs; real-Postgres tests verify ordering and visibility before limiting; route tests verify authentication and serialized contracts. Optional pure unit coverage avoids forcing unnecessary helpers. Existing shell, Projects and account-switcher regressions are included.
- Manual-only coverage: Device visual quality and VoiceOver presentation remain manual. Automated widget layout, keyboard and semantics checks cover their testable counterparts, so core accessibility behavior is not manual-only.
- Evidence limits: No feature tests have been implemented or executed. Existing suite locations and commands are planning context, not evidence that the new feature passes.

## Risk And Approval Review

- Risk level: Medium, unchanged. Main concerns are membership semantics, server-side visibility, mounted-route selection and account isolation.
- Review requirement: Document review was recommended and explicitly requested by the user's choice “1.” This artifact completes that review.
- Approval notes: Requirements were approved in conversation. The test specification has now been reviewed as part of the document set. No high-risk production work is within scope; production changes and backfills retain their separate change controls.
- Risk verification paths: RISK-001 → IT-001/IT-004 and GAP-001; RISK-002 → IT-002/IT-003/IT-006; RISK-003 → AT-003/REG-003; RISK-004 → AT-007/IT-003.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Verify the onboarding-completion write/lifecycle semantics and identify existing profile visibility and relationship filters. Then design the authenticated member read and craft-selecting navigation contract.
- Recommended first failing implementation test: AT-003 for a craft link selecting the correct tab and query while Projects is already mounted, as specified in the test handoff.
- Blocking issues: None for coding planning. DR-001 must be resolved before implementing the member selection path; DR-002 is normal interface-design work for the coding plan.

## Notes For Next Stage

- Create `04-coding-plan.md` only after the user chooses to advance; do not implement during planning.
- Keep the shared desktop scope and current breakpoint. Derive a preferred-only link list rather than directly displaying the existing provider's preferred-first full catalog.
- Define how explicit craft navigation updates an already mounted Projects page and survives late preference reordering without persisting a preference change.
- Reuse existing public profile summaries, visibility policy and profile/edit navigation where suitable. Apply eligibility before the five-member limit on the server.
- Specify account-scoped state, cache behavior and disposal/late-response handling. Both sections must remain independently usable during load/error/retry.
- Resolve metadata semantics, deterministic ordering, response fields and final automation targets in the plan. If new facts require a product change, surface it as a proposed revision rather than silently diverging from approved requirements.
- Preserve all requirement, AC and test IDs. Avoid adding polling, follow controls, editorial features, migrations or infrastructure without a demonstrated need and appropriate authorization.
- Retain the focused Flutter tests and required backend gate for later validation. Record manual device evidence separately from automated results.
