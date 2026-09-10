# Document Review: AppView Account Subscriptions

## Verdict

Status: Approved

Reviewer: OpenCode

Date: 2026-09-10

Risk level: High

## Summary

The revised requirements and acceptance tests consistently implement the
product owner's approved reconciliation-first MVP. The documents preserve ADR
014's payer/beneficiary separation and local AppView authorization while making
RevenueCat's complete API v2 subscription snapshot and `gives_access` value the
lifecycle authority.

The revised scope removes the disproportionate launch machinery for webhook
lifecycle reduction, purchase intents, ownership transfer, automatic duplicate
failover, cross-tier arbitration, raw payload quarantine/retention, and runtime
merchandising validation. Every remaining Must requirement has automated
coverage or an explicit later Flutter evidence gap.

## Findings

The document review incorporates the validated 2026-09-10 account-deletion
policy correction alongside the four resolved provider-model inconsistencies:

- Billing-period timestamps no longer override RevenueCat `gives_access`,
  including during provider outages; reconciliation staleness is observable.
- A post-lapse or paid-product-change subscription ID creates a new unassigned
  license rather than inheriting the prior subscription's assignment.
- Owner deletion requests reconciliation and remains blocked until every row in
  a post-intent complete snapshot is non-pending and exactly `will_not_renew`.
  Active/access-granting and expired/inaccessible canceled rows are both
  eligible; assigned non-owner deletion clears target assignments atomically.
- Every reconciliation trigger advances requested generation, so a trigger
  arriving during an in-flight fetch remains pending after the older claim
  applies.

No unresolved document findings remain.

## Superseded Review Decisions

The prior review approved a larger contract. The following former decisions are
intentionally superseded by the product owner's 2026-09-09 Option A approval:

- Event types no longer drive a local lifecycle reducer. Every safely mapped
  event queues a current RevenueCat snapshot fetch.
- Purchase-assignment intents and billing-ownership transfer are deferred.
- Duplicate and cross-tier anomalies fail closed instead of invoking canonical
  promotion, inheritance, or destination-first assignment rules.
- Assignments persist on their own license through inaccessible periods and can
  reactivate only for the same RevenueCat subscription ID; new IDs are
  unassigned.
- Exact webhook bodies, malformed-body quarantine, collision payloads, and
  one-year raw retention are removed.
- Deletion is blocked before post-intent reconciliation and whenever any row is
  pending or lacks exact `will_not_renew` evidence. Access, status, timestamps,
  and webhook types do not establish cancellation. Once every row has affirmative
  non-renewal evidence, including rows still granting access, closed accounts
  retain only a DID-free identity marker and are never reconciled again; assigned
  non-owner deletion clears target assignments.
- Prices, territories, offerings, packages, trials, Family Sharing, and current
  offering selection move to the external activation checklist.

## Traceability Review

- Planning to requirements: Option A and the explicit merchandising decision
  appear in sections 3, 5, 8, 12, and 21 of `01-requirements.md`.
- Requirements to acceptance criteria: Every Must requirement links to at least
  one of AC-001 through AC-026. The two Should requirements also have coverage.
- Acceptance criteria to tests: Every acceptance criterion has an acceptance,
  unit, integration, regression, or justified later-stage manual target.
- Retired IDs: `FR-021`, `FR-023` through `FR-025`, `FR-033`, `RULE-009`, and
  `RULE-010` are explicitly retired rather than silently omitted.

## Coverage Review

- Must requirements covered: All.
- Missing or weak AppView coverage: None identified.
- Manual-only coverage: None for the AppView implementation. `MAN-001` completes
  Flutter RevenueCat identity evidence later. `MAN-002` verifies external
  merchandising and store configuration and is intentionally not an AppView
  runtime test.

## Risk And Approval Review

- Risk level: High because the feature controls paid authorization, accepts a
  public provider callback, and participates in account deletion.
- Review requirement: Complete for coding-plan readiness. Explicit user approval
  of Option A was provided on 2026-09-09, and validated manual feedback approved
  the revised deletion policy on 2026-09-10.
- Production blockers: Real provider credentials/catalog resources, Flutter SDK
  identity tests, and an account-recovery procedure for `Keep with original App
  User ID` remain required before launch.
- Risk reduction: Trusting RevenueCat `gives_access` and reconciling complete
  snapshots removes most local lifecycle disagreement and event-order risk.
- Concurrency reduction: Separate requested, claimed, and reconciled generations
  prevent an apply from erasing a trigger received during provider fetch.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Implement and test the five-field effective-access
  projection, then the four-table core schema and snapshot mapper.
- Blocking issues: None for AppView implementation.

## Notes For Next Stage

- Use one reconciliation processor for webhook, owner-refresh, restore-like, and
  scheduled triggers.
- Increment requested generation on every trigger; applying a claim must leave a
  newer requested generation due.
- Do not add event-specific subscription mutations.
- Store only sanitized webhook metadata; never persist raw bodies.
- Keep provider subscription and license rows separate as required by ADR 014.
- Derive app identity and tier from configured product mappings; the RevenueCat
  subscription resource does not supply app identity.
- Keep billing-period timestamps informational; never locally expire
  `gives_access`.
- Treat simultaneous accessible conflicts as owner-visible, non-assignable
  anomalies; a normal new subscription ID remains unassigned but is not
  anomalous solely because it is new.
- Block billing-owner deletion until every persisted subscription in a
  post-intent complete snapshot has `pending_payment=false` and exact
  `auto_renewal_status='will_not_renew'`; do not require expiry or loss of access,
  infer cancellation from other fields, or call the provider. Clear target
  assignments when an assigned non-owner is deleted.
- Keep Flutter and external RevenueCat/store configuration outside this coding
  plan.
