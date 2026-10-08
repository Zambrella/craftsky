# Implementation Review: Flutter Account Subscriptions

## Verdict

Status: Approved with notes
Reviewer: OpenCode, independent implementation review
Date: 2026-09-14
Risk level: High

## Summary

The implementation is ready for merge and implementation handoff. The eighth
correction fully resolves `IR-020` and `IR-021`: active assignments no longer
receive dormant wording, and purchase/restore reconciliation now fails closed on
unresolved or inaccessible subscription/license relationships.

No blocking or Important findings remain. Production activation remains
separately blocked by `MAN-001` through `MAN-005` and their external staging,
RevenueCat, store, sandbox, and device prerequisites.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-022 | Suggestion | Traceability / UI | The owner page does not display the retained billing-owner account label. This remains a non-blocking Should-level gap. | `FR-025` (Should), `AC-014`, `AT-013`; `app/lib/subscriptions/providers/subscription_page_model_provider.dart`; `app/lib/subscriptions/pages/subscription_page.dart`; `app/test/subscriptions/pages/subscription_settings_page_test.dart` | Optional follow-up: display the authorized owner handle, never the RevenueCat billing UUID, and add an `AT-013` assertion. |

## Correction Verification

- `IR-020`: Resolved. Dormant assignment copy requires both an assigned tier and
  `givesAccess == false`. Canonical active Plus and Business cases prove dormant
  wording is absent while effective-Free dormant cases remain covered.
- `IR-021`: Resolved. `billingRelationshipsResolved` validates complete one-to-one
  subscription/license relationships for purchase eligibility and reconciliation.
  Newly introduced expected-tier purchase licenses must correlate to accessible
  subscriptions, and purchase/restore assignment selectors independently enforce
  the same access relationship.

## Requirement And Test Traceability

- Requirements implemented: All reviewed Must-level requirements and automated
  acceptance paths are implemented. AppView remains authoritative for access,
  ownership, assignment, and reconciliation.
- Tests implemented: Purchase, restore, assignment, Customer Center, pending-flow
  recovery, account/privacy boundaries, optional-platform behavior, accessibility,
  and diagnostics have deterministic coverage. The eighth pass adds orphan and
  inaccessible relationship cases at the state-machine and controller boundaries.
- Unplanned behavior: None identified.
- Remaining gaps: No Must-level implementation gaps remain. `IR-022` is an
  optional Should-level follow-up.
- Architecture: No unplanned route, lexicon, PDS-write, migration, or authorization
  changes were identified.

## Test Evidence

- Commands reviewed: Focused correction suites, complete subscription suite, full
  Flutter suite, Dart MCP analysis, `just test`, `just appview-check`, iOS
  simulator build, Android debug build, web build, and `git diff --check`.
- Passing evidence: The independent focused correction suite passed 61 tests. The
  complete subscription suite passed 173 tests and the full Flutter suite passed
  2,467 tests. Dart analysis, backend/release checks, all platform builds, and diff
  validation passed.
- Failing automated tests: None.
- Skipped external tests: `MAN-001` through `MAN-005`, for the documented native
  catalog, credentials, sandbox, staging, and manual-verification prerequisites.
- Advisory: `GO-2026-5932` remains an existing unrelated no-fix advisory.

## Risk Review

- Risk level: High.
- Risk notes: The feature crosses billing ownership, RevenueCat identity,
  entitlement assignment, account switching, privacy, and asynchronous
  reconciliation boundaries. The implementation fail-closes reviewed identity and
  relationship anomalies, and automated evidence is proportionate for handoff.
- Approval notes: Approved for merge and implementation handoff. This verdict does
  not approve production activation before the manual prerequisites pass.

## UI Polish Recommendation

- Recommendation: Optional.
- Reason: The UI is behaviorally coherent, but hierarchy between aggregate access,
  individual licenses, and payer/beneficiary status could be refined.
- Suggested polish notes: Keep any polish non-behavioral and do not alter billing
  state, authorization, or action eligibility.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None.
- Verification to rerun: No correction rerun required. Run the standard gates if
  the implementation changes after this review.
