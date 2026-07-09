# Document Review: Reposts And Quote Posts

## Verdict
Status: Approved with notes
Reviewer: Codex
Date: 2026-07-08
Risk level: Medium

## Summary
The requirements and acceptance-test documents are aligned with the confirmed Option A direction: straight reposts remain interaction records, quote posts remain authored post records with Craftsky quote embeds, and only the home timeline changes to feed-item shape. Must requirements have acceptance criteria and test coverage. The remaining issues are non-blocking design notes for the coding plan, mainly around exact response-model shape and performance verification.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | API Design | The documents intentionally define quote preview behavior at the product/test level but leave the exact wire model for `visible`, `unavailable`, and `hidden` preview states to implementation design. | `01-requirements.md` FR-008, FR-009, AC-011, AC-012, Q15; `02-acceptance-tests.md` UT-003, IT-003, TD-007 | In `04-coding-plan.md`, specify the concrete quote-preview JSON shape, nullable fields, and Flutter model names before implementation starts. |
| DR-002 | Suggestion | Performance | N+1 prevention is covered as a test gap because the repo may not have query-count instrumentation. This is acceptable for planning, but the coding plan should choose a verifiable strategy. | `01-requirements.md` NFR-003, RISK-003; `02-acceptance-tests.md` IT-013, GAP-001 | In `04-coding-plan.md`, name the intended batched queries or bounded query sequence and decide whether to add instrumentation or use query-plan-specific store tests. |
| DR-003 | Suggestion | Timeline Contract | The tests require deterministic mixed post/repost ordering, but the coding plan still needs to define the stable feed-item identity used for cursor tie-breakers and Flutter list keys. | `01-requirements.md` NFR-001, FR-017; `02-acceptance-tests.md` AT-005, IT-004, TD-004, TD-007 | In `04-coding-plan.md`, define feed-item identity for authored posts and repost activities, including cursor encoding inputs and client dedupe/list-key behavior. |

## Traceability Review
- Planning to requirements: The requirements preserve the confirmed Option A direction, ADR waiver, no reply-sharing v1 scope, separate repost/quote counts, one-level quote preview, and home-timeline-only feed-item response.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR`, and `RULE` links to at least one acceptance criterion. Should requirements also have criteria where useful.
- Acceptance criteria to tests: `02-acceptance-tests.md` covers `AC-001` through `AC-040` with acceptance, unit, integration, regression, or manual checks. Each test case links back to requirement IDs and acceptance criteria IDs.

## Coverage Review
- Must requirements covered: Yes. All Must requirements listed in `01-requirements.md` have test IDs or explicit coverage paths in `02-acceptance-tests.md`.
- Missing or weak coverage: No blocking gaps. `GAP-001` records that exact N+1 verification may need instrumentation; `GAP-002` records that real-PDS OAuth behavior is outside this stage while token exposure remains covered by handler/client tests.
- Manual-only coverage: The only manual check is final UX smoke testing for visual fit and Bluesky-familiar interaction. Core behavior is automated.

## Risk And Approval Review
- Risk level: Medium.
- Review requirement: Review was recommended, not mandatory, in `01-requirements.md`; this document completes that review.
- Approval notes: Implementation may proceed to coding planning. The planner should carry forward DR-001 through DR-003 as design tasks, not as blockers.

## Coding Plan Readiness
- Ready for coding planning: Yes.
- Recommended first step: Start from `IT-001` by changing the current timeline-store coverage that excludes repost activity into a feed-item test that includes followed straight repost activity with reason attribution while still excluding replies.
- Blocking issues: None.

## Notes For Next Stage
- Define concrete Go and Flutter types for home timeline feed items, repost reason, and quote preview states before coding.
- Keep profile, search, thread, and post-detail surfaces post-shaped; only the home timeline should return `{post, reason}` items.
- Treat timeline cursor identity as part of the API contract, not an incidental SQL detail.
- Preserve the existing repost endpoints and straight-repost optimistic behavior while adding the quote action path.
- If lexicon files change unexpectedly, invoke the atproto lexicon checklist and run `just lexgen`; no ADR is required for this pre-live feature per the requirements.
