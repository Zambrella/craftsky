# Document Review: Follow / Unfollow MVP

## Verdict

Status: Approved with notes

Reviewer: document-reviewer

Date: 2026-05-25

Risk level: High

## Summary

The requirements and acceptance-test specification are traceable and cover the amended interoperable follow/unfollow scope, including non-Craftsky profile navigation, AppView-mediated PDS writes, graph indexing, Flutter UI behavior, and security constraints. Every Must requirement has linked acceptance criteria and at least one test or explicit gap.

Document-review follow-up resolved the only blocking inconsistency: the user confirmed that Tap will deliver historical data. `01-requirements.md` and `02-acceptance-tests.md` now treat live Tap verification as an end-to-end smoke check rather than a blocking unknown.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Requirements / Tests | Error code names for some new follow validation failures are intentionally left to implementation. This is workable for coding planning, but acceptance tests must lock names before handler implementation proceeds too far. | `01-requirements.md` AC-012, AC-013, EC-001, EC-002; `02-acceptance-tests.md` UT-003, GAP-002 | Coding planner should include an early API-contract decision for exact error codes, then make the first handler tests assert those codes. |
| DR-002 | Important | Requirements / Architecture | Non-Craftsky profile hydration is required, but the documents leave the implementation mechanism open between indexing, cache, or AppView-side PDS hydration. This is probably acceptable for coding planning, but it is a design hotspot because it broadens the current membership-gated `bluesky_profiles` model. | `01-requirements.md` FR-011, FR-012, ASM-005, RISK-008, Data/Persistence Impact; `02-acceptance-tests.md` UT-012, UT-013, IT-007, IT-011 | Coding planner should make profile hydration architecture an explicit early design step before API handler work, including failure behavior and storage/cache ownership. |
| DR-003 | Suggestion | Requirements / Edge Cases | Duplicate PDS follow records are assumed rare/invalid, but requirements only define collapsed count/viewer semantics, not canonical unfollow behavior when multiple active URIs exist for the same follower-target pair. | `01-requirements.md` RULE-003, RISK-002, RISK-004; `02-acceptance-tests.md` GAP-003, UT-011 | Coding planner should either document canonical-delete behavior or keep this as a known limitation while ensuring counts/state collapse duplicates. |

## Traceability Review

- Planning to requirements: The amended direction is preserved. Requirements reflect the shift from Craftsky-only follows to interoperable atproto follows, non-Craftsky profile navigation, response-driven optimistic UI, self-follow/self-unfollow rejection, and MVP exclusion of non-Craftsky counts.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR`, and `RULE` links to at least one acceptance criterion. Acceptance criteria are generally externally verifiable.
- Acceptance criteria to tests: Every acceptance criterion is represented in `02-acceptance-tests.md` by automated tests, manual checks, or explicit test gaps. AC-016/AC-017/AC-025 have automated indexer/dispatcher coverage, with `MAN-001` retained as an end-to-end Tap smoke check.

## Coverage Review

- Must requirements covered: Yes. The coverage matrix maps all Must requirements to test IDs or explicit gaps.
- Missing or weak coverage: No unacknowledged missing coverage. Weak areas are explicitly documented as `GAP-001` through `GAP-004`.
- Manual-only coverage: Live Tap smoke verification (`MAN-001`) and query-plan validation (`MAN-003`) are manual. Manual coverage is justified and no longer blocks coding planning because Tap historical delivery has been confirmed.

## Risk And Approval Review

- Risk level: High, correctly carried forward.
- Review requirement: Required, correctly carried forward.
- Approval notes: The user confirmed Tap historical delivery during document review follow-up. Coding planning may proceed with the notes above.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Start from `UT-004`: follow indexer create is idempotent and stores one active relationship from an `app.bsky.graph.follow` create event.
- Blocking issues: None

## Notes For Next Stage

- The recommended first failing test is `UT-004`: follow indexer create is idempotent and stores one active relationship from an `app.bsky.graph.follow` create event.
- The coding planner should explicitly decide follow error codes early (`GAP-002`) and include profile hydration architecture before follow handler/UI work.
- Keep `MAN-001` as an end-to-end smoke check after implementation wiring, not as a blocker to coding planning.
