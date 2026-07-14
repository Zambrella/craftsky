# Document Review: AppView Push Notifications

## Verdict

Status: Approved
Reviewer: Codex
Date: 2026-07-14
Risk level: High

## Summary

The revised documents consistently add account-wide notification newness without expanding into per-item unread state. They preserve read-only GET semantics through `GET /v1/notifications/new-count`, use an explicit bodyless `POST /v1/notifications/seen`, and define a monotonic activation revision plus snapshot high-water acknowledgement so concurrent notifications are not accidentally consumed.

The new scope is isolated from push delivery, device routing, notification preference, hydration, and lexicon contracts. Every new Must requirement and AC-045 through AC-050 has automated coverage; Should-level NFR-007 is also covered. The user explicitly approved both the selected REST shape and implementation on 2026-07-14, so coding planning may proceed without another product decision.

## Findings

None identified.

## Traceability Review

- Planning to requirements: Q12 through Q15 preserve the confirmed account-wide model, explicit acknowledgement operation, high-water definition, and snapshot concurrency rule. BR-004, FR-034 through FR-038, RULE-007, risks, edge cases, persistence, and API impacts all reflect those decisions.
- Requirements to acceptance criteria: Every Must BR, FR, NFR, and RULE references at least one AC. AC-045 through AC-051 are externally verifiable and distinguish first-use, visibility, replay/reactivation, concurrency, and multi-account/device behavior.
- Acceptance criteria to tests: AT-010, IT-032 through IT-035, and REG-009 cover all new acceptance criteria. Existing test IDs remain stable and no previous coverage was removed.

## Coverage Review

- Must requirements covered: BR-001 through BR-004, FR-001 through FR-038, NFR-001 through NFR-004, and RULE-001 through RULE-007 have linked automated tests. Should requirements NFR-005 through NFR-007 are also automated.
- Missing or weak coverage: None blocking. The snapshot race requires a deterministic transaction barrier or store seam rather than timing-based sleeps; the coding plan must make that seam explicit.
- Manual-only coverage: Only real Android FCM and iOS FCM/APNs delivery remain manual. Notification newness requires no manual provider coverage.

## Risk And Approval Review

- Risk level: High overall because the parent feature stores private device routing data and runs an external push worker. The incremental newness feature is medium risk due to schema/API/concurrency behavior.
- Review requirement: Satisfied. Requirements and tests have been re-reviewed after the scope change.
- Approval notes: The user explicitly selected account-wide state, approved the REST shape, and authorized document updates plus AppView implementation on 2026-07-14.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Add the follow-up migration and make IT-032 fail for the revision sequence, acknowledgement row, and active account/revision index. Then implement IT-033 count behavior before the mark-seen write.
- Blocking issues: None.

## Notes For Next Stage

- Add a new migration rather than rewriting already-applied `000021`.
- Allocate a new revision only for an inserted or genuinely updated activation. Exact replay and retraction must retain the current revision.
- Keep count and list actor-visibility predicates aligned.
- Capture the mark-seen high-water revision and upsert the account marker in one transaction; use greatest-value conflict handling.
- Keep both notification GET routes read-only. Only `POST /v1/notifications/seen` may advance acknowledgement state.
- Scope the acknowledgement table by account DID only; device ID is authentication middleware context, not part of the key.
