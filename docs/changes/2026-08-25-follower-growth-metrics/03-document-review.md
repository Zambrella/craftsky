# Document Review: Follower Growth Metrics

## Verdict

Status: Changes required

Reviewer: OpenCode

Date: 2026-08-25

Risk level: Medium

## Summary

The requirements and acceptance-test specification consistently preserve the selected daily-snapshot design, owner-only access, canonical Craftsky-member count semantics, explicit missing dates, and account-bound Flutter state. Every Must requirement links to acceptance criteria and proposed tests, and the proposed database-outward TDD order is practical for this repository.

Coding planning should not begin yet because three API/calendar details remain ambiguous enough to produce incompatible implementations: the successful no-history response contract, the scope and nullability of `availableFrom`, and the one-year start date when the current UTC date is February 29. The test specification also needs stronger evidence for the full non-interference rule after those requirement details are resolved.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Requirements | The required endpoint behavior for an account with no snapshots is not defined. FR-010 and AC-017 require latest snapshot metadata, latest count, and `availableFrom`, but only net change is explicitly nullable. AC-015 promises success only when data is available, while FR-018 requires a no-history UI state. The API could therefore return success with null fields, omit fields, or return an error, and all would fit part of the current wording. | `01-requirements.md` FR-010, FR-018, AC-015, AC-017, AC-026; `02-acceptance-tests.md` UT-004, UT-007, IT-007 | Specify that a valid owner request succeeds even with no snapshots; define exact nullability/presence for `availableFrom`, latest snapshot date, capture timestamp, latest count, and net change; update the linked acceptance criteria and tests without renumbering existing IDs. |
| DR-002 | Important | Requirements | `availableFrom` has no precise scope. It may mean the owner's earliest snapshot globally, the first non-null point inside the selected period, or the selected range start when a snapshot exists there. These produce different responses for established accounts with a leading in-range gap and for partial-history accounts. | `01-requirements.md` FR-010, FR-015, AC-017, AC-022; `02-acceptance-tests.md` UT-002, IT-008, TD-002 | Define `availableFrom` precisely, including its value for no history and for a selected period whose first date is missing despite older history. Align UT-002 and IT-008 with that definition. |
| DR-003 | Important | Requirements | The one-year boundary is not deterministic when the current UTC date is February 29. “One UTC calendar year before” does not state whether 2028-02-29 starts at 2027-02-28 or 2027-03-01, yet AC-018 explicitly requires deterministic leap-year behavior. | `01-requirements.md` FR-011, AC-018, EC-011; `02-acceptance-tests.md` UT-001, IT-008, TD-003 | Choose and state the February 29 anniversary rule, then add the exact expected start date and point count to UT-001 and IT-008. |
| DR-004 | Important | Tests | RULE-005 and AC-032 cover feed ordering, recommendations, discovery visibility, moderation outcomes, and advertising, but REG-003 exercises only feed/search and REG-004 exercises public-profile exposure. The matrix currently marks the whole rule automated even though several named effects have no concrete check or explicit absence assertion. | `01-requirements.md` RULE-005, AC-032; `02-acceptance-tests.md` coverage matrix, REG-003, REG-004 | Expand regression/static-boundary coverage to every named domain, or explicitly document which systems do not exist and test that follower-growth code is not imported or queried by the existing feed, discovery, moderation, and ranking boundaries. |
| DR-005 | Suggestion | Tests | IT-003 combines a useful set-based SQL assertion with an index-plan expectation that may be unstable for an all-member aggregate, where PostgreSQL can correctly choose a sequential scan. The requirement is no per-profile query loop, not mandatory index use for every fixture. | `01-requirements.md` FR-004, AC-010, RISK-002; `02-acceptance-tests.md` IT-003, GAP-001 | Make the primary automated assertion one capture statement/no per-profile query pattern. Keep plan inspection focused on pathological plans and move cardinality-specific index expectations to the benchmark/load follow-up. |
| DR-006 | Suggestion | Risk | Worker cadence, successful empty-run tracking, and the source of the latest-successful-snapshot-age metric are intentionally deferred, but they directly shape freshness and observability tests. This is correctly listed as non-blocking only if coding design resolves it before implementation. | `01-requirements.md` Open Questions, NFR-002, NFR-007; `02-acceptance-tests.md` UT-005, IT-005, IT-013, GAP-004 | Carry GAP-004 into the coding plan as an explicit design decision before worker tests are written. |

## Traceability Review

- Planning to requirements: The confirmed Option A daily canonical snapshot direction is preserved. The ledger, lazy capture, live overlay, global graph, event-driven writes, timezone rebucketing, and historical reconstruction alternatives remain excluded.
- Requirements to acceptance criteria: All 3 business requirements, 20 functional requirements, 5 business rules, and 7 non-functional requirements link to acceptance criteria. Must coverage is complete by ID, but DR-001 through DR-003 identify criteria whose expected values are not yet precise enough to implement consistently.
- Acceptance criteria to tests: AC-001 through AC-038 appear in the coverage matrix and concrete test cases. Test IDs are stable and monotonic. DR-004 identifies weaker behavioral evidence for AC-032 rather than a missing ID link.

## Coverage Review

- Must requirements covered: Yes by traceability ID. BR-001 through BR-003, FR-001 through FR-020, RULE-001 through RULE-005, and required NFRs all map to automated tests or justified mixed automated/manual coverage.
- Missing or weak coverage: No Must requirement is absent. AC-032 has weak breadth, and the no-history, `availableFrom`, and leap-day tests cannot have definitive expected values until DR-001 through DR-003 are resolved.
- Manual-only coverage: None is exclusively manual. NFR-005 and NFR-004/NFR-006 have automated widget/semantics coverage supplemented by justified real screen-reader and chart-legibility checks.
- Test data: Fixtures cover mixed membership, sparse history, leap dates, concurrency, authorization, account switching, deletion, retention, and telemetry privacy.
- Automation practicality: Proposed targets match existing `testdb.WithSchema`, route-policy, Dio mock, Riverpod provider, router, responsive widget, and semantics test conventions.

## Risk And Approval Review

- Risk level: Medium. The cross-stack scope, atomic snapshot transaction, concurrency, private account state, owner deletion integration, and accessible chart justify the existing rating.
- Review requirement: Document review is required before coding planning because DR-001 through DR-003 affect the API contract and deterministic range behavior.
- Approval notes: Resolve DR-001 through DR-004 in `01-requirements.md` and `02-acceptance-tests.md`, preserving existing requirement, acceptance-criteria, and test IDs. DR-005 and DR-006 may be carried as coding-plan notes.

## Coding Plan Readiness

- Ready for coding planning: No
- Recommended first step: Revise the requirements to define the empty-history response, `availableFrom`, and February 29 behavior, then align the existing test cases and rerun document review.
- Blocking issues: DR-001, DR-002, DR-003, and DR-004.
- Recommended first failing test after approval: IT-001 remains the correct first implementation test because it establishes the snapshot persistence invariant independently of the unresolved API details.

## Notes For Next Stage

- Preserve the database-outward implementation sequence from `02-acceptance-tests.md` after the document set is approved.
- Treat the owner DID as authentication-derived throughout the route and storage interfaces.
- Design database coordination before writing concurrency code; uniqueness alone does not prove one complete logical all-member result.
- Define worker clock/timer injection and successful-run observability before implementing UT-005, IT-005, or IT-013.
- Keep no-history and sparse-history response construction in one deterministic server-side series-shaping path so Flutter does not infer missing dates.
- Reassess chart dependencies only during coding planning, with semantics, RTL, maintenance, and supported-platform behavior as selection gates.
