# Document Review: Timeline Feed AppView

## Verdict
Status: Approved with notes
Reviewer: gpt-5.5 document reviewer
Date: 2026-05-28
Risk level: Medium

## Summary

`01-requirements.md` and `02-acceptance-tests.md` are consistent and ready for coding-plan work. The requirements preserve the chosen direct joined AppView timeline approach, keep Flutter/client and future feed variants out of scope, and carry the clarified decisions into testable business rules and acceptance criteria. The test specification provides traceable automated coverage for every Must requirement and documents the main residual risk around query-plan/index performance.

No blocking findings were identified. The notes below are implementation-planning reminders to make the first TDD slice sharper, not prerequisites for moving forward.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Tests | The test plan covers requested-limit pagination, invalid cursors, and unknown query parameters, but it does not explicitly call out default-limit and max-limit assertions from the “existing default/max limit behavior” wording. | `01-requirements.md` FR-007, AC-006, AC-009, AC-017; `02-acceptance-tests.md` IT-003, IT-011, IT-012, UT-003 | Non-blocking: during coding planning, either fold default/max-limit assertions into the handler/query tests or confirm existing shared limit-parser tests already cover this behavior for new endpoints. |
| DR-002 | Suggestion | Tests / Routing | Negative auth/device route coverage is clear, but the “valid auth + device reaches the timeline handler” portion is implicit rather than named as its own positive route assertion. | `01-requirements.md` AC-001; `02-acceptance-tests.md` AT-001, IT-008, IT-009, IT-010 | Non-blocking: consider adding a cheap positive route-wiring assertion if the route test harness can observe handler invocation. |
| DR-003 | Suggestion | Risk | Query boundedness/index suitability is appropriately documented as partial review/manual coverage, because production-scale query plans may not be proven by fixtures alone. | `01-requirements.md` NFR-002, RISK-001; `02-acceptance-tests.md` IT-016, MAN-001, GAP-001 | Non-blocking: coding planning should include an index/query-plan review step and add a narrow supporting index only if evidence shows existing indexes are insufficient. |

## Traceability Review

- Planning to requirements: The requirements carry forward the documented decisions from Q1–Q15, including direct joined querying, AppView-indexed data only, own-post inclusion, current follow graph per page, quote strong references only, `indexed_at DESC, uri DESC` ordering, no filters except `limit`/`cursor`, and unknown-query-parameter tolerance.
- Requirements to acceptance criteria: Every Must business, functional, non-functional, and business-rule requirement has linked acceptance criteria. Should requirements also have appropriate acceptance criteria where behavior matters (`FR-011`, `FR-012`, `NFR-002`, `NFR-003`).
- Acceptance criteria to tests: Every acceptance criterion is represented in the acceptance, unit, integration, regression, or manual coverage plan. The only weak spots are the non-blocking explicitness notes in DR-001 and DR-002.

## Coverage Review

- Must requirements covered: Yes. `BR-001`, `BR-002`, `FR-001` through `FR-010`, `FR-013`, `NFR-001`, and `RULE-001` through `RULE-005` all have acceptance criteria and planned automated tests.
- Missing or weak coverage: No blocking gaps. Weak explicitness exists around default/max-limit assertions and positive route-to-handler wiring, both of which can be handled in the coding plan without changing product intent.
- Manual-only coverage: None for Must behavior. `NFR-002` has partial manual/review coverage through `MAN-001` and `GAP-001`, which is acceptable for a Should-level performance/query-plan concern.

## Risk And Approval Review

- Risk level: Medium.
- Review requirement: Review before coding planning is satisfied by this artifact.
- Approval notes: Medium risk is justified because this adds a user-visible API endpoint and establishes feed semantics that later Flutter work will consume. The risk is mitigated by the direct joined query scope, bounded pagination, explicit non-goals, and a store/query-boundary requirement for future feed variants.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Start with `IT-001` — `TestTimelineStore_ListTimeline_ReturnsOwnAndFollowedEligiblePostsOnly` in `appview/internal/api/timeline_store_test.go`.
- Blocking issues: None.

## Notes For Next Stage

- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this review as the source of truth.
- Keep the first implementation slice focused on a timeline store/query boundary before route wiring.
- Preserve the response contract by reusing existing `PostResponse`, author hydration, engagement summary, cursor, and error-envelope patterns.
- Use current indexed AppView state only: `craftsky_posts`, active `atproto_follows`, indexed profile/display data, and engagement tables. Do not introduce PDS read-through or synthetic just-created rows.
- Preserve future-feed flexibility by avoiding a one-off SQL string embedded directly in route registration.
