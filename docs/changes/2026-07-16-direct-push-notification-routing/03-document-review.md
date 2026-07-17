# Document Review: Direct Push Notification Routing

## Verdict

Status: Approved

Reviewer: Codex

Date: 2026-07-17

Risk level: High

## Summary

The requirements and acceptance-test specification are complete and aligned with the confirmed direction: AppView supplies minimal canonical facts, Flutter owns destination inference, account binding gates every outcome, and destination APIs remain authoritative. All 30 Must requirements link to acceptance criteria and automated coverage, and all 22 acceptance criteria appear in the test design.

The earlier blocking parser/binding ambiguity is resolved. The test specification now defines a minimal provider-neutral open attempt with independently testable binding and fact validity, scopes AppView minimum-fact assertions to routing metadata, and records the user's requirements-stage approval while retaining the explicit pre-implementation approval gate. The documents are ready for coding-plan work.

## Findings

None identified after revision.

### Resolved Findings

| ID | Resolution |
|---|---|
| DR-001 | `02-acceptance-tests.md` now separates binding validity from fact validity in the strategy, AT-002/AT-003, UT-001/UT-002/UT-005/UT-006, IT-003, test data, and implementation handoff. |
| DR-002 | `01-requirements.md` records user approval for requirements/test-design progression and preserves separate explicit approval before implementation. |
| DR-003 | IT-001 now limits the minimum-facts assertion to routing metadata and explicitly preserves token, platform, TTL, and generic visible-copy inputs. |

## Traceability Review

- Planning to requirements: Pass. No separate `00-initial-prompt.md` exists, but `01-requirements.md` embeds the initial request, codebase findings, clarified decisions, candidate approaches, recommendation, scope, risks, assumptions, and test commands. The confirmed minimal-facts/Flutter-inference direction is preserved.
- Requirements to acceptance criteria: Pass. All 30 Must requirements reference at least one of the 22 stable acceptance criteria. The criteria cover the payload matrix, exact Flutter mapping, account gate, destination authorization, failure classes, app-state parity, resolver removal, privacy, bounded payloads, regressions, and readiness semantics.
- Acceptance criteria to tests: Pass. Every acceptance criterion is represented in the coverage matrix and has an automated verification path. The low-level contract now distinguishes invalid binding from invalid facts beneath AC-006/AC-011/AC-012/AC-021.

## Coverage Review

- Must requirements covered: 30 of 30.
- Acceptance criteria covered: 22 of 22.
- Missing or weak coverage: None identified. The malformed-facts versus malformed-binding split is explicit at acceptance, unit, integration, test-data, and handoff levels.
- Manual-only coverage: None. MAN-001 through MAN-005 supplement automated coverage for physical FCM/APNs lifecycle delivery, true stale accepted pushes, retained multi-account OS notifications, and qualitative cold-start request order.
- Test levels: Appropriate. Pure parsing/inference/error policy is unit-level; AppView fact projection, route registry, runtime flow, and destination authorization are integration-level; user-visible navigation/error behavior is acceptance-level; physical provider/OS behavior remains manual.
- Automation targets: Practical and grounded in existing Flutter notification, router, feed/profile, AppView push/API, and observability suites. Proposed new files are limited to behavior not represented by a suitable existing suite.

## Risk And Approval Review

- Risk level: High, unchanged.
- Review requirement: Required before coding planning because the provider-data trust boundary, cross-account gate, resolver removal, and destination error behavior are authorization-adjacent.
- Approval notes: The user authorized progression from requirements to test design and into this review. Explicit approval is still required before implementation begins, and that gate is now recorded consistently.
- Primary implementation risk: Binding validity and fact validity must remain independent so malformed facts cannot bypass the binding gate or be discarded before the required post-binding fallback. The revised tests make this observable without prescribing a heavy type hierarchy.
- Other high-risk areas have concrete verification paths: strict category facts, typed identifiers, no resolver request, authenticated destination reads, named permanent/transient errors, privacy sentinels, latest-only readiness, and clean resolver removal.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Build the coding plan around `UT-001`, the structured provider-neutral open-attempt boundary, then implement the binding gate before fact inference/fallback as ordered in the acceptance-test handoff.
- Blocking issues: None.
- Re-review result: The revised parser/binding contract, coverage matrix, test data, AppView routing-metadata scope, and recommended first failing test are consistent.

## Notes For Next Stage

- Do not design implementation around the current nullable `NotificationOpenEvent.tryParseProviderData` shape. The revised tests specify outcomes first; the coding planner should choose the smallest provider-neutral result type that preserves a valid binding independently from fact validity.
- Keep Flutter destination inference separate from provider parsing and from GoRouter route execution so the category mapping remains pure and fakeable.
- Scope AppView “minimum facts” assertions to provider routing data, not token/platform/TTL or unchanged generic visible-copy inputs.
- Retain `UT-001` as the first TDD seam, with its expected result defined by the structured binding/fact outcomes in the revised specification.
