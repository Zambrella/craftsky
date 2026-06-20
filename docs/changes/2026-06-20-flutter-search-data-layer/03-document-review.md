# Document Review: Flutter Search Data Layer

## Verdict

Status: Approved with notes
Reviewer: GPT-5.5 document-reviewer
Date: 2026-06-20
Risk level: Medium

## Summary

The requirements and acceptance-test specification are consistent enough for coding planning. The confirmed Option A direction is carried through to Must requirements, non-goals preserve the non-UI Flutter-only scope, and every Must business, functional, non-functional, and rule requirement has acceptance criteria and mapped tests. No blocking contradictions or missing Must coverage were found.

The remaining notes are sequencing/scope clarifications for the coding planner: choose an explicit first failing test sequence, keep any `SearchPage`/route edits exceptional and behavior-neutral, and ensure the already-documented hashtag path-encoding gap receives focused API-client coverage.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Tests / Coding readiness | The test handoff names `IT-002` as the recommended first failing test, while the suggested test order starts with unit model tests (`UT-001`, `UT-002`, `UT-007`, `UT-009`). This is not blocking, but the coding plan should choose a single explicit TDD sequence. | `02-acceptance-tests.md` §11 lines 308-311 | In the coding plan, state whether implementation starts with `IT-002` or with the listed model scaffolding tests before `IT-002`. |
| DR-002 | Suggestion | Requirements / Scope | Scope is clear overall, but `NG-002` allows unavoidable compile-time `SearchPage` provider imports while `NFR-001`, `MAN-002`, and `REG-004` treat widget/route files as out of scope. This is acceptable if no user-visible behavior changes occur, but UI-file changes should be treated as exceptional. | `01-requirements.md` `NG-002`, `NFR-001`; `02-acceptance-tests.md` `MAN-002`, `REG-004` | Coding planner should keep implementation in data-layer/generated/test files where possible and require explicit justification for any widget/route-file touch. |
| DR-003 | Suggestion | Tests / Risk | Hashtag path-encoding edge cases are documented as a known implementation-sensitive risk, but the main acceptance scenario uses a simple path value. Focused API-client coverage should be included so tags requiring URL encoding remain safe and AppView validation errors map correctly. | `01-requirements.md` `EC-001`; `02-acceptance-tests.md` `TD-008`, `GAP-002`, `IT-002` | Include at least one focused hashtag path-encoding/safe-path test under the `IT-002` API-client coverage unless implementation constraints make the exact assertion impractical, in which case document the limitation. |

## Traceability Review

- Planning to requirements: The confirmed Option A dedicated `app/lib/search` data-layer approach is reflected in goals `G-001` through `G-005`, requirements `BR-001` through `BR-003`, `FR-001` through `FR-016`, and the non-goals in §8. Open questions are marked as none, and the medium-risk areas from planning are carried into §19 risks.
- Requirements to acceptance criteria: All Must `BR`, `FR`, `NFR`, and `RULE` requirements link to acceptance criteria. Should requirements `NFR-004` and `NFR-005` also have acceptance criteria and test coverage.
- Acceptance criteria to tests: The coverage matrix maps acceptance criteria to concrete `AT`, `UT`, `IT`, `REG`, and `MAN` IDs. Each listed test case includes requirement IDs and acceptance criteria IDs.

## Coverage Review

- Must requirements covered: Yes. The matrix covers `BR-001` through `BR-003`, `FR-001` through `FR-016`, Must `NFR-001` through `NFR-003`, and `RULE-001` through `RULE-004`.
- Missing or weak coverage: No blocking gaps. Non-blocking documented gaps are appropriate: mocked AppView rather than live backend (`GAP-001`), hashtag path encoding (`GAP-002`), representative malformed JSON only (`GAP-003`), manual privacy/static scope checks (`GAP-004`), and possible future UI-provider ergonomics (`GAP-005`).
- Manual-only coverage: Manual checks for generated-file scope, mapper initialization, privacy/PDS/local-persistence review, and UI-agnostic provider contracts are justified because they are primarily static architectural checks.

## Risk And Approval Review

- Risk level: Medium, consistent across the requirements and acceptance-test documents.
- Review requirement: Review is recommended before implementation; no policy-driven blocking approval remains after this review.
- Approval notes: Proceed with coding planning. Highest-risk areas for implementation review are AppView wire-shape fidelity, generated mapper/provider files, explicit recent-search mutations, provider pagination/de-duplication, and the non-UI scope boundary.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Use the acceptance-test handoff as source of truth and make the first TDD sequencing decision explicit; `IT-002` is the recommended first failing integration/API-client test, with the listed `UT-001`, `UT-002`, `UT-007`, and `UT-009` model tests available as prerequisite scaffolding if the coding planner chooses that order.
- Blocking issues: None.

## Notes For Next Stage

- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this review as source of truth.
- Keep the implementation Flutter-only and non-UI; avoid widget/route changes unless strictly necessary and behavior-neutral.
- Preserve AppView JSON/HTTP conventions: shared authenticated Dio, `unwrapApi`, camelCase JSON, opaque cursors/IDs, and no PDS/local recent-search persistence.
- Prioritize tests around endpoint serialization/decoding, recent-search typed payloads, mapper/codegen integration, and Riverpod pagination behavior.
