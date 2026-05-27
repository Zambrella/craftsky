# Document Review: Profile Social Summary

## Verdict
Status: Approved with notes
Reviewer: OpenCode document-reviewer
Date: 2026-05-27
Risk level: Medium

## Summary
The requirements and acceptance-test documents are consistent and ready for coding-plan work. The documents preserve the confirmed product decisions: profile pages hide follower/following counts, profile responses keep those counts available, mutuals use `mutualFollowerCount` plus a separate paginated endpoint, recent and total posts count top-level authored posts only, non-Craftsky profiles hide account age, and follower/following list counts move one tap deeper to app-bar titles.

No blocking issues were found. The remaining findings are non-blocking planning notes that should be resolved during coding-plan/API-design work, primarily final endpoint paths, final JSON field names, and exact Flutter age-format thresholds.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | API design | The requirements intentionally leave exact new route names unspecified, while tests mark this as GAP-001. This is acceptable for document review but must be settled before implementation tests are written. | `01-requirements.md` §16, FR-016, NFR-001; `02-acceptance-tests.md` GAP-001, IT-004-IT-007 | Coding planner should choose and document concrete `/v1/` endpoint paths before coding. |
| DR-002 | Suggestion | API contract | The requirements define post/project count semantics but not exact response field names beyond camelCase; tests use example names (`postCount`, `postsLast7Days`, `projectCount`) and mark this as GAP-002. | `01-requirements.md` §15, FR-005, FR-006; `02-acceptance-tests.md` TD-005, GAP-002 | Coding planner should finalize response field names and align tests to them. |
| DR-003 | Suggestion | UI logic | `Joined <age> ago` is specified, but age unit thresholds are not. This is appropriately recorded as a test gap, not a blocker. | `01-requirements.md` FR-004, RULE-002; `02-acceptance-tests.md` UT-004, GAP-003 | Coding planner should define formatter thresholds before Flutter unit tests are authored. |
| DR-004 | Suggestion | Risk / performance | Large-list performance and index quality are recognized but only partly automatable. This is acceptable for the current stage. | `01-requirements.md` RISK-001, NFR-002; `02-acceptance-tests.md` IT-008, MAN-002, GAP-004 | Coding planner should consider indexes and query shapes explicitly. |

## Traceability Review
- Planning to requirements: The confirmed Profile Summary API direction is preserved in `01-requirements.md` Q1/Q2, the recommended direction, requirements FR-012 through FR-016, and API/UI impact sections. Follow-up product decisions from grilling are captured explicitly in Q2 and reflected throughout the requirements.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR-001`, and `RULE` links to at least one acceptance criterion. Acceptance criteria AC-001 through AC-020 are externally verifiable and align with the requirement IDs.
- Acceptance criteria to tests: Every acceptance criterion has automated test coverage or a justified manual check. AC-014/NFR-002 has partial automation plus manual/performance notes, which is appropriate for query performance risk.

## Coverage Review
- Must requirements covered: Yes. All Must business, functional, non-functional, and rule requirements are represented in the coverage matrix and test cases.
- Missing or weak coverage: None blocking. The only weak areas are intentionally listed as GAP-001 through GAP-004: concrete route names, exact JSON field names, age-format thresholds, and portable verification of database index use.
- Manual-only coverage: Bottom-sheet visual height/scroll behavior (`MAN-001`) and perceived large-list performance (`MAN-002`) are reasonable manual complements to automated widget/integration tests.

## Risk And Approval Review
- Risk level: Medium.
- Review requirement: Review recommended before coding planning; this document review satisfies that workflow gate.
- Approval notes: Proceed to coding planning with the non-blocking findings above. No high-risk auth, privacy, persistence migration, or lexicon blocker was identified. Auth/device behavior is explicitly covered by NFR-001 and IT-007.

## Coding Plan Readiness
- Ready for coding planning: Yes.
- Recommended first step: Start with `IT-001` from `02-acceptance-tests.md`: AppView profile summary counts top-level posts only and exposes a data-driven project count. This locks down the central server-side summary semantics before Flutter UI work.
- Blocking issues: None.

## Notes For Next Stage
- Choose final endpoint paths for mutual followers, followers, and following before writing route tests.
- Choose final camelCase response field names for `mutualFollowerCount`, top-level total posts, posts in the last 7 days, and project count before writing model/client tests.
- Define Flutter age-format thresholds for `Joined <age> ago` before implementing the formatter.
- Consider SQL indexes/query plans for mutuals and recency-sorted graph lists during implementation planning.
- Preserve the architectural rule that Flutter reads social/profile data from AppView and never queries PDS directly.
