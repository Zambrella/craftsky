# Document Review: Notifications MVP

## Verdict

Status: Approved with notes  
Reviewer: gpt-5.5 document reviewer  
Date: 2026-05-29  
Risk level: Medium

## Summary

`01-requirements.md` and `02-acceptance-tests.md` are consistent with the approved direction: a read-only, derived Notifications MVP exposed through `GET /v1/notifications` and rendered in the Flutter Notifications tab. Must requirements have linked acceptance criteria and test coverage. The test specification gives a clear test-first path, beginning with AppView store tests for derived follow notifications scoped to the authenticated viewer.

No blocking issues were found. Coding planning may proceed. The notes below should be handled during coding-plan/API-shape design rather than by rewriting the requirements or test-design documents.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | API / Tests | The exact notification item wire schema is intentionally not fully pinned beyond `type`, `actor`, event timestamps, subject data, reply identity, camelCase JSON, and reuse of post response semantics where practical. This is sufficient for planning, but implementation needs one explicit response-shape decision before tests are written. | `01-requirements.md` §12 `FR-009`-`FR-011`, §16 API; `02-acceptance-tests.md` `UT-006`, `UT-009`, `IT-001`-`IT-004` | In the coding plan, define the concrete `GET /v1/notifications` response structs and field names before implementing handler/model tests. |
| DR-002 | Suggestion | Requirements / Tests | Unavailable subject-post behavior is documented as implementation-defined: omit the notification or return an unavailable-safe shape, but never crash. This is acceptable for MVP, but it must be decided before `IT-013` is implemented. | `01-requirements.md` `EC-004`; `02-acceptance-tests.md` `IT-013`, `GAP-002` | Coding planner should choose and document one behavior, then make `IT-013` assert that behavior. |
| DR-003 | Suggestion | Test Commands | Flutter focused-test command discovery is intentionally approximate because new notification test files do not exist yet. | `02-acceptance-tests.md` §11 Commands discovered | Coding planner should replace the placeholder command with exact new notification test paths once the implementation test files are planned. |

## Traceability Review

- Planning to requirements: The requirements preserve the approved Option A direction: derived read-only notifications from existing indexed AppView data, no push, no unread state, no grouping, no persisted notification table, no new lexicons, and no PDS reads from Flutter. Risks from planning are carried into `RISK-001` through `RISK-005`.
- Requirements to acceptance criteria: All Must requirements in the handoff (`BR-001`, `BR-002`, `FR-001` through `FR-015`, `NFR-001`, and `RULE-001` through `RULE-003`) link to at least one acceptance criterion in `01-requirements.md` §12-§13.
- Acceptance criteria to tests: `02-acceptance-tests.md` covers `AC-001` through `AC-020` through acceptance, unit, integration, regression, or manual checks. No acceptance criterion is missing coverage.

## Coverage Review

- Must requirements covered: Yes. The coverage matrix in `02-acceptance-tests.md` §2 covers every must-cover ID from the requirements handoff.
- Missing or weak coverage: No blocking gaps. Non-blocking gaps are explicitly recorded as `GAP-001` through `GAP-003`, mainly performance benchmarking, unavailable subject behavior, and manual copy/navigation review.
- Manual-only coverage: None for Must behavior. Manual checks `MAN-001` and `MAN-002` supplement automated tests for visual/navigation quality and architecture review; core behavior also has automated coverage.

## Risk And Approval Review

- Risk level: Medium, consistent across requirements and test specification.
- Review requirement: Review was recommended because this touches a new authenticated AppView API, derived SQL, Flutter data/provider state, and user-visible navigation. This document completes the recommended document review.
- Approval notes: Approved with notes. The feature can move into coding planning, but the coding plan should explicitly define the notification response schema and unavailable-subject policy before implementation tests are written.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Start with failing AppView store test `IT-001` in `appview/internal/api/notification_store_test.go`, proving follow notifications are derived from indexed data and scoped to the authenticated viewer.
- Blocking issues: None.

## Notes For Next Stage

- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this review as the source of truth.
- Define the concrete notification response types/field names in the coding plan before implementing handler and Flutter model tests.
- Decide whether unavailable subject posts are omitted or represented with an unavailable-safe shape; then make `IT-013` assert that exact behavior.
- Keep the slice read-only and derived. Do not add notification persistence, PDS writes, push delivery, unread state, grouping, or new lexicons in this MVP.
- Preserve existing API conventions: `/v1/` prefix, authenticated/device middleware, camelCase JSON, standard error envelope, opaque cursor pagination, and omitted cursor at terminal pages.
