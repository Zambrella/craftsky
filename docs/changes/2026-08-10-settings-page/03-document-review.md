# Document Review: Settings Page And Lean Account Deletion

## Verdict

Status: Approved with notes
Reviewer: Codex workflow review
Date: 2026-08-11
Risk level: High

## Summary

The requirements and acceptance-test specification consistently implement the product owner's approved simplification. Security-critical destructive boundaries remain explicit and automated, while indexer receipts/convergence, deletion status/recovery, manual Retry, checkpoints/artifacts, the audit/sweeper, and deletion-specific metrics are consistently excluded. The reduced terminal definition is testable and the high-risk implementation approval is recorded.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Risk | Removing the executable private-store manifest makes future store coverage dependent on engineering review. This is an accepted simplification but remains the largest privacy regression risk. | `01-requirements.md` RISK-002, ASM-002; `02-acceptance-tests.md` IT-012, IT-031 | Keep explicit current-store owner-isolation fixtures and call out private-data cleanup in future schema review. No blocker for this pre-production implementation. |
| DR-002 | Suggestion | Recovery | A permanently unusable OAuth session has no in-app recovery UI. The contract permits automatic retry, monitoring, and a later sign-in to refresh authority. | FR-017, FR-024; EC-011; GAP-002 | Keep the pending-login branch coarse and verify it never restores ordinary access. Reconsider user-facing recovery before production users. |

## Traceability Review

- Planning to requirements: The approved lean durable option is reflected in goals, non-goals, desired behavior, requirements, data impact, risks, and assumptions.
- Requirements to acceptance criteria: Every Must BR/FR/NFR/RULE has at least one linked AC. AC-038, AC-042, AC-044, and AC-046 explicitly verify removed subsystems stay removed.
- Acceptance criteria to tests: Every AC is covered by an acceptance, unit, integration, regression, or justified manual test. Destructive real-PDS behavior remains MAN-003 only.

## Coverage Review

- Must requirements covered: Yes.
- Missing or weak coverage: No blocking gaps. Future private-store drift and real firehose/PDS integration are explicit risks/gaps.
- Manual-only coverage: Responsive visual QA and disposable real OAuth/PDS boundary checks.

## Risk And Approval Review

- Risk level: High because the feature changes auth, sessions, private-data deletion, migrations, and owner-scoped PDS records.
- Review requirement: Explicit product-owner approval required.
- Approval notes: On 2026-08-11 the product owner accepted no progress/status UI, independent indexer convergence, automatic server-only retry, no checkpoints, no deletion audit, and ordinary operational logging, then explicitly directed implementation.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: IT-029 — shrink migration `000037` to OAuth request metadata plus one minimal operation table and prove superseded tables are absent.
- Blocking issues: None.

## Notes For Next Stage

- Remove dependencies from the outside inward: schema/test contract, per-record persistence, checkpointed cleanup, convergence/terminal gates, routes/telemetry, then Flutter status state.
- Preserve fresh reauth, atomic job/OAuth binding, session revocation, owner/collection typed boundaries, final empty scan, membership-last ordering, private/Instagram/scheduled cleanup, and pending-login denial.
- Existing Tap/indexer behavior must remain unchanged after receipt-observer removal.
- Do not run destructive tests against a personal or production PDS account.
