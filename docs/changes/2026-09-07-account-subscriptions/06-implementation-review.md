# Implementation Review: AppView Account Subscriptions

## Verdict
Status: Approved
Reviewer: OpenCode
Date: 2026-09-10
Risk level: High

## Summary
This focused follow-up reviewed only the latest requested corrections: useful Go
documentation for every exported type in `internal/subscriptions/types.go`, and
the revised account-deletion policy in FR-022/AC-020. No scoped implementation,
test, or traceability gap was identified.

All fifteen exported types in `types.go` have comments that identify their
domain purpose rather than merely repeat the declaration. The deletion boundary
now permits both active/access-granting and expired/inaccessible subscriptions
when every persisted row is non-pending and exactly `will_not_renew`. Missing,
empty, unknown, renewable, resumable, already-renewed, and pending-payment
evidence blocks the lifecycle transition atomically.

## Findings
None identified.

## Requirement And Test Traceability
- Requirements implemented: Revised FR-022 and FR-031 satisfy AC-020. The
  exported-type documentation correction is complete in
  `appview/internal/subscriptions/types.go`.
- Tests implemented: IT-008 and AT-009 coverage includes pre-reconciliation
  blocking, active/access-granting success, expired/inaccessible success, mixed
  blocker sets, warning/error behavior, assignment cleanup, closure cleanup,
  and delayed-event handling.
- Document consistency: `01-requirements.md` through `04-coding-plan.md` state
  the same exact non-pending/`will_not_renew` rule. The final manual-feedback
  correction in `05-implementation-plan.md` explicitly replaces its historical
  pre-correction TDD entries and records the matching implementation and tests.
- Unplanned behavior: None identified. In particular, no provider cancellation
  operation or dependency was introduced.
- Remaining gaps: None within this focused follow-up scope. Previously deferred
  Flutter, live-provider activation, account-recovery, and paid-capability work
  remains outside this review.

## Test Evidence
- Independently run: focused uncached race tests for
  `TestSubscriptionDeletionParticipantBlocksClosesAndUnassigns`,
  `TestSubscriptionDeletionParticipantBlocksMixedProviderBilling`,
  `TestAcceptedSelfAssignedDeletionSerializesWithSnapshotApply`, and
  `TestAppServiceOwnsDeletionCredentialAcrossLifecycle` passed in
  `internal/accountdeletion`.
- Independently run: `go doc ./internal/subscriptions` completed and exposed the
  documented exported subscription API.
- Independently run: `git diff --check` passed with no output.
- Supplied fresh evidence: focused race tests, `go vet ./...`,
  `just appview-check`, and `git diff --check` all passed.
- Failing or skipped tests: None in scope.
- Vulnerability evidence: Known transitive `GO-2026-5932` remains acknowledged;
  no upstream fix is available and the release gate passed.

## Behavioral Verification
- Fresh post-intent reconciliation: `BeginDeletion` advances and records
  `deletion_requested_generation`; `ConfirmDeletion` requires the reconciled
  generation to reach that fence. Snapshot application accepts only a complete,
  generation-and-lease-fenced snapshot before advancing reconciliation.
- Exact deletion predicate: The destructive transaction blocks on any
  `pending_payment=true` row or any renewal value distinct from exact
  `will_not_renew`; it does not inspect status, access, timestamps, or webhook
  type as cancellation evidence.
- Active and expired success: The participant and full-service tests prove
  active/access-granting closure, while the accepted-deletion/snapshot race
  proves expired/inaccessible closure under the same predicate.
- Atomic mixed blockers: Each mixed-set case retains the active owner, both
  provider rows, and the assignment after rejection.
- Closure and assignment cleanup: Accepted owner deletion removes live provider
  subscriptions and cascading licenses/assignments and leaves a DID-free closed
  marker. Accepted non-owner deletion clears only assignments targeting that
  DID. Reversible intent creation and failed confirmation preserve assignments.
- Delayed-event non-resurrection: A delayed event for the retained RevenueCat
  UUID is stored as `ignored_closed` without scheduling reconciliation. The
  deterministic deletion/snapshot race returns `ErrStaleSnapshot` and leaves no
  live subscription or assignment row.
- Provider behavior and warning: The deletion participant is database-only and
  cannot cancel provider billing. The warning remains
  `provider_billing_not_canceled` with the accurate statement that deleting the
  CraftSky account does not cancel provider billing and may end CraftSky access.

## Risk Review
- Risk level: High
- Risk notes: This is an irreversible account-deletion boundary over paid-access
  state. Its affirmative provider-evidence rule, reconciliation fence, lock
  order, rollback behavior, and non-resurrection path are covered by focused
  PostgreSQL race tests.
- Approval notes: The revised policy is fail-closed for all missing and
  non-terminal evidence while no longer incorrectly requiring expiry or loss of
  access. Prior assignment-preservation, lock-order, closure-cleanup, and
  delayed-event protections remain intact.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: This follow-up changes Go documentation and AppView deletion behavior;
  it contains no user-facing UI.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes: None.
- Suggested next failing test: None; the focused follow-up is approved.
- Verification to rerun: Preserve the focused deletion race tests, `go vet
  ./...`, `just appview-check`, and `git diff --check` if this boundary changes.
