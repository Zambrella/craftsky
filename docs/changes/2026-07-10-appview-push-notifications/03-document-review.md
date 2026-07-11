# Document Review: AppView Push Notifications

## Verdict

Status: Approved with notes  
Reviewer: Codex  
Date: 2026-07-11  
Risk level: High

## Summary

The documents consistently select durable notification events plus a transactional per-account-subscription outbox, preserve the AppView/PDS privacy boundary, and provide broad automated coverage for all 44 acceptance criteria. The scope, lifecycle policy, preference timing, multi-account routing, provider isolation, and at-least-once limitation are well defined.

The three blocking findings from the initial review are resolved. Regression and manual cases now link directly to acceptance criteria; FR-033 and AC-044 define atomic, non-transferring token rebinding across device IDs; and AC-005 plus IT-004 explicitly cover rollback during both creation and deletion lifecycle changes. The documents are ready for coding planning. Remaining notes concern TDD sequencing and the separate explicit approval required before implementation.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-004 | Suggestion | Coding readiness | The revised handoff now sequences UT-001, migration/index invariants in IT-028, and one producer's creation/deletion atomicity cases in IT-004. The coding plan should retain this vertical slice before expanding to every producer. | `02-acceptance-tests.md` §11; UT-001, IT-004, IT-028 | Preserve the stated sequence and avoid implementing all producers before one end-to-end transactional path is green. |
| DR-005 | Suggestion | Risk / Approval | The requirements correctly retain a High-risk explicit approval gate. The user's direct request authorized test design and document review, but it is not approval to implement. The current handoff mentions this, though the next stage should keep the gate visible. | `01-requirements.md` §22; `02-acceptance-tests.md` §§1, 11 | Keep explicit implementation approval as a stage gate after coding-plan review. Coding planning may proceed now. |

## Traceability Review

- Planning to requirements: The recommended Option A is carried through FR-001 through FR-016 and the desired-behavior section. Confirmed decisions on prospective preferences, retraction/reactivation, six-hour expiry, payload minimization, and multi-account installations remain intact. No initial-prompt artifact exists; the initial request and discovery context are embedded in `01-requirements.md`.
- Requirements to acceptance criteria: Every Must BR, FR, NFR, and RULE references at least one AC. AC-001 through AC-044 are externally verifiable. FR-033 and AC-044 now specify the cross-device token collision invariant without requiring the coding planner to invent an authorization policy.
- Acceptance criteria to tests: AC-001 through AC-044 appear in the coverage matrix and in automated unit, integration, acceptance, or regression targets. Every acceptance, unit, integration, regression, and manual test case links requirement and acceptance-criteria IDs. AC-005 and IT-004 cover rollback on both creation and deletion.

## Coverage Review

- Must requirements covered: BR-001 through BR-003, FR-001 through FR-033, NFR-001 through NFR-004, and RULE-001 through RULE-006 all have linked automated tests in the matrix. The Should requirements NFR-005 and NFR-006 are also covered.
- Missing or weak coverage: None blocking. Query-count instrumentation and the concrete FCM client remain explicit coding-plan decisions with fallback verification strategies documented as GAP-003 and GAP-005.
- Manual-only coverage: Only real Android FCM and iOS FCM/APNs delivery are manual. This is justified because provider credentials, OS delivery, and device state are nondeterministic; payload construction, TTL, retry, routing, and redaction remain automated with a fake sender.

## Risk And Approval Review

- Risk level: High. The feature stores private device routing data, introduces an external provider and worker, changes the source of truth for the notification feed, and must coordinate lifecycle changes transactionally.
- Review requirement: Satisfied for coding planning. The document set has been re-reviewed after resolving the initial blockers.
- Approval notes: The user explicitly initiated test design and document review. No implementation approval has been given. After an approved coding plan, request explicit approval before implementation begins.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Design the category model and persistence schema around UT-001 and IT-028, then define the shared transaction-aware ingestion/retraction seam needed to drive IT-004 red-green.
- Blocking issues: None for coding planning. Explicit approval remains required before implementation.

## Notes For Next Stage

- Keep notification creation/retraction behind a shared transaction-aware service so producer indexers cannot accidentally commit source and notification state independently.
- Design installation ownership, account authorization, and send routing as separate concepts. The token collision rule must not transfer account subscriptions implicitly or expose one account's installation state to another.
- Use an injected clock, jitter source, and sender; automated tests must not contact FCM.
- Preserve stable notification identity across source record recreation while retaining the original “push already enqueued” boundary.
- Make migration/index design precede the first Postgres transaction test in the TDD sequence.
- Keep the implementation approval gate separate from approval to write the coding plan.
