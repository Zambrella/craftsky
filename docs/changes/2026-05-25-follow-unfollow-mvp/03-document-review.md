# Document Review: Follow / Unfollow MVP

## Verdict

Status: Changes required

Reviewer: document-reviewer

Date: 2026-05-25

Risk level: High

## Summary

The requirements and acceptance-test specification are broadly traceable and cover the amended interoperable follow/unfollow scope, including non-Craftsky profile navigation, AppView-mediated PDS writes, graph indexing, Flutter UI behavior, and security constraints. Every Must requirement has linked acceptance criteria and at least one test or explicit gap.

One blocking inconsistency remains: `01-requirements.md` still marks Tap historical follow delivery verification as a blocking question to be confirmed during test design, but `02-acceptance-tests.md` does not resolve it. Instead, it leaves live Tap verification as `GAP-001`/`MAN-001`, with different timing language. Before coding planning, the documents should be aligned so the next agent knows whether Tap verification is a prerequisite to implementation planning, the first implementation/preflight step, or an acceptance/completion check.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Critical | Risk / Readiness | The Tap historical delivery question is still marked blocking in requirements but unresolved in test design. The requirements say to confirm during test design; the test spec says isolated tests cannot prove it and leaves a manual check before implementation is accepted as complete. This creates conflicting stage-gate timing for coding planning. | `01-requirements.md` §21, ASM-003, RISK-007, Handoff lines 456-457; `02-acceptance-tests.md` §1 Tap note, MAN-001, GAP-001, Handoff blocking gaps | Revise the documents to one consistent position: either (a) resolve the Tap capability now with documented evidence and remove/downgrade the blocking question, or (b) explicitly make MAN-001 the first coding-plan/preflight task and change the requirement open question from “confirm during test design” to “verify before accepting implementation.” |
| DR-002 | Important | Requirements / Tests | Error code names for some new follow validation failures are intentionally left to implementation. This is workable for coding planning, but acceptance tests must lock names before handler implementation proceeds too far. | `01-requirements.md` AC-012, AC-013, EC-001, EC-002; `02-acceptance-tests.md` UT-003, GAP-003 | Coding planner should include an early API-contract decision for exact error codes, then make the first handler tests assert those codes. |
| DR-003 | Important | Requirements / Architecture | Non-Craftsky profile hydration is required, but the documents leave the implementation mechanism open between indexing, cache, or AppView-side PDS hydration. This is probably acceptable for coding planning, but it is a design hotspot because it broadens the current membership-gated `bluesky_profiles` model. | `01-requirements.md` FR-011, FR-012, ASM-005, RISK-008, Data/Persistence Impact; `02-acceptance-tests.md` UT-012, UT-013, IT-007, IT-011 | Coding planner should make profile hydration architecture an explicit early design step before API handler work, including failure behavior and storage/cache ownership. |
| DR-004 | Suggestion | Requirements / Edge Cases | Duplicate PDS follow records are assumed rare/invalid, but requirements only define collapsed count/viewer semantics, not canonical unfollow behavior when multiple active URIs exist for the same follower-target pair. | `01-requirements.md` RULE-003, RISK-002, RISK-004; `02-acceptance-tests.md` GAP-004, UT-011 | Coding planner should either document canonical-delete behavior or keep this as a known limitation while ensuring counts/state collapse duplicates. |

## Traceability Review

- Planning to requirements: The amended direction is preserved. Requirements reflect the shift from Craftsky-only follows to interoperable atproto follows, non-Craftsky profile navigation, response-driven optimistic UI, self-follow/self-unfollow rejection, and MVP exclusion of non-Craftsky counts.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR`, and `RULE` links to at least one acceptance criterion. Acceptance criteria are generally externally verifiable.
- Acceptance criteria to tests: Every acceptance criterion is represented in `02-acceptance-tests.md` by automated tests, manual checks, or explicit test gaps. AC-016/AC-017/AC-025 are only partially automatable because live Tap backfill behavior needs a real stack check.

## Coverage Review

- Must requirements covered: Yes. The coverage matrix maps all Must requirements to test IDs or explicit gaps.
- Missing or weak coverage: No unacknowledged missing coverage. Weak areas are explicitly documented as `GAP-001` through `GAP-005`.
- Manual-only coverage: Live Tap historical delivery (`MAN-001`) and query-plan validation (`MAN-003`) are manual. Manual coverage is justified, but `MAN-001` conflicts with the requirement document’s unresolved blocking-question wording.

## Risk And Approval Review

- Risk level: High, correctly carried forward.
- Review requirement: Required, correctly carried forward.
- Approval notes: The documents should not proceed to coding planning until DR-001 is resolved or explicitly accepted by the user as a coding-plan preflight rather than a test-design blocker.

## Coding Plan Readiness

- Ready for coding planning: No
- Recommended first step: Resolve DR-001 by aligning the Tap historical delivery verification language across `01-requirements.md` and `02-acceptance-tests.md`. If the user accepts deferring live verification, make MAN-001 the first coding-plan/preflight task and keep `UT-004` as the first failing implementation test.
- Blocking issues: DR-001

## Notes For Next Stage

- Once DR-001 is addressed, the recommended first failing test remains `UT-004`: follow indexer create is idempotent and stores one active relationship from an `app.bsky.graph.follow` create event.
- The coding planner should explicitly decide follow error codes early (`GAP-003`) and include profile hydration architecture before follow handler/UI work.
- Do not start source-code implementation until the Tap verification gate is either resolved, downgraded, or accepted as a preflight task by the user.
