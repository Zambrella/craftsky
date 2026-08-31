# Implementation Review: Flutter Business Profiles

## Verdict

Status: Changes required
Reviewer: OpenCode
Date: 2026-08-30
Risk level: Medium

## Summary

Correction loops R9-R13 closed dirty Products Back navigation, profile/detail stale publication, lifecycle conflict overwrite, Edit Profile accessibility evidence, and filtered cursor tie-break evidence. All repository gates pass. One Must-level list reconciliation race remains: a record-generation change can stale an in-flight list without staling its account-generation preflight, allowing incomplete rows and an old cursor to publish. MAN-001 and MAN-002 remain accepted manual release checks and do not block implementation approval.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-020 | Important | Behavior | List preflight checks only account generation. If another surface settles an event overlay and advances that record generation before an older list response completes, per-record reconciliation marks the row stale and omits it, but the provider still publishes the remaining stale rows and response cursor. Current tests leave the overlay installed and do not cover settlement-before-list-completion. | FR-022, NFR-001, AC-034, AC-036, UT-015, IT-010, IT-013; `app/lib/business/providers/business_projection_overlay_provider.dart:295-307,340-344,426-439`; `app/lib/business/providers/profile_business_events_provider.dart:158-172,214-223,277-278`; `app/lib/business/providers/owner_business_events_provider.dart:66-79,128-134,202-204`; `app/test/business/providers/event_list_provider_test.dart:192-262`; `app/test/business/providers/owner_business_events_provider_test.dart:194-257` | Make list reconciliation report aggregate stale status and retain the current rows/cursor whenever any returned record is stale. Add public and owner tests where another read settles the overlay before the older list completion. |

## Requirement And Test Traceability

- Requirements implemented: The full surface is implemented, but FR-022/NFR-001 remain incomplete for the IR-020 settlement ordering.
- Tests implemented: All 1,798 Flutter tests and every AppView package pass, including required real-Postgres coverage.
- Unplanned behavior: None identified.
- Remaining gaps: IR-020 requires one focused correction. MAN-001 and MAN-002 remain manual release checks.

## Test Evidence

- Commands reviewed: Focused R1-R13 commands in `05-implementation-plan.md`; final `just test`; `just app-test`; `just app-analyze`; `git diff --check`.
- Passing evidence: AppView packages pass with real Postgres; all 1,798 Flutter tests pass; analysis and diff checks are clean.
- Failing or skipped tests: No current automated failures. The settlement-before-list-completion sequence is untested. MAN-001 and MAN-002 are manual-only.

## Risk Review

- Risk level: Medium
- Risk notes: IR-020 can temporarily remove an event and advance pagination from an obsolete response after accepted state has already settled. No migration, lexicon, direct-PDS, token, entitlement, or destructive-data regression was found.
- Approval notes: Correct IR-020, rerun gates, and perform a final focused review.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: Automated UI/accessibility coverage is now complete; optional polish should wait until correctness approval and MAN-001/MAN-002 observations.
- Suggested polish notes: Do not alter routing, reconciliation, conflict, or validation behavior during polish.

## Handoff Back To TDD Builder

- Required fixes: Address IR-020.
- Suggested next failing test: Start a public or owner list read, settle its event overlay through another surface, complete the older list response, and assert current rows/cursor remain unchanged.
- Verification to rerun: Focused public/owner list and reconciliation tests, then `just test`, `just app-test`, `just app-analyze`, and `git diff --check`.
