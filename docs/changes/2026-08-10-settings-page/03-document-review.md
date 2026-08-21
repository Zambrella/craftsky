# Document Review: Settings Page And Lean Account Deletion

> **AppView audit re-review (2026-08-14):** Section 7 is the current review verdict for the exact-key safety-tombstone amendment. It supersedes the earlier one-table/no-checkpoint readiness note but preserves the lean UI, permission, and final-minimization boundaries.

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

## 7. AppView Audit Re-review: Exact-Key Safety Tombstones

### Verdict

Status: Approved

Reviewer: Codex workflow review

Date: 2026-08-14

Risk level: High

### Summary

The product decision that blocked the lifecycle plan is now explicit and consistently testable. The amended requirements permit only temporary, non-secret, exact-owner/job/key tombstones for outcome-uncertain registered CraftSky PDS writes and scheduled-object writes. They forbid a false terminal/data-free result after an initially empty scan, require indefinite reconciliation when no finite remote settlement guarantee is proven, and restore the original artifact-free state after convergence.

The amendment does not reintroduce deletion progress/status UI, a status/recovery credential, manual Retry, per-component private-cleanup checkpoints, index receipts, a deletion audit/sweeper, detailed deletion metrics, or broader PDS/object authority. No product question remains for coding-plan work.

### Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-003 | Important | Correctness | A remote PDS/object call may be accepted before AppView crashes and commit after the worker's first absence check; the original one-table contract could therefore report success while data later appears. | `01-requirements.md` §24, FR-028–FR-031, AC-049–AC-053; `02-acceptance-tests.md` §12 | Implement the approved exact-key tombstones and make IT-035/IT-036 deterministic release barriers. This is fully specified and is not a planning blocker. |
| DR-004 | Important | Minimization | Temporary tombstones are acceptable only if their schema cannot drift into a payload, audit, user-status, or reusable deletion store and terminal finalization removes them. | NFR-007, RULE-012, AC-049, AC-053, REG-017 | Enforce field/state/check constraints, typed exact-key APIs, final residue tests, and capability-boundary tests. |
| DR-005 | Suggestion | Operations | Without a tested server-side settlement bound, a tombstone and operation may remain reconciling indefinitely. | RULE-013, AC-052, IT-037 | Keep claims/backoff bounded and observable through shared redacted lifecycle signals; do not invent a timeout or surface a user status API. |

### Traceability and coverage review

- Planning to requirements: The audit plan's recommended safety branch is recorded as the product owner's 2026-08-14 decision. Superseded clauses are named precisely, while the final lean boundaries remain explicit.
- Requirements to acceptance criteria: Every new Must requirement (FR-028–FR-031, NFR-007, RULE-012, RULE-013) links to AC-049–AC-054.
- Acceptance criteria to tests: AT-007–AT-009, UT-025–UT-026, IT-035–IT-038, and REG-017 cover both cross-system crash races, scope/minimization, indefinite reconciliation, and final artifact removal.
- Manual-only coverage: No new Must behavior is manual-only. MAN-003 remains the disposable real-PDS smoke gate, not the race-safety evidence.

### Risk and approval review

- Risk level: High because the work coordinates destructive PDS deletion, private-object deletion, OAuth authority, and durable migrations across crashes.
- Review requirement: Satisfied. The product owner selected the exact-key safety-tombstone branch on 2026-08-14.
- Approval notes: Implementation may proceed with the strict test order. An empty first scan, elapsed client timeout, or vanished AppView advisory lock is never a convergence proof.

### Coding-plan readiness

- Ready for coding planning: Yes.
- Recommended first step: Add the minimized tombstone migration/store contract and write IT-035 as the first end-to-end failing barrier.
- Blocking issues: None.
- Required coordination: Reserve its migration in the shared audit sequence and reuse the owner lifecycle/effect attempt, deletion-only session, owner/object fence, and scheduled-media generation contracts rather than creating parallel authority models.
