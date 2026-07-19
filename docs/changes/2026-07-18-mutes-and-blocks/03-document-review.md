# Document Review: Account Mutes And Blocks

## Verdict

Status: Approved with notes  
Reviewer: Codex  
Date: 2026-07-19  
Risk level: High

## Summary

The requirements and acceptance-test specification are consistent with the confirmed direction: private AppView mutes, public interoperable `app.bsky.graph.block` records, one server-side relationship policy, a hard current-Craftsky-membership boundary, immediate pre-Tap enforcement, non-destructive record retention, and DID-scoped Flutter state.

Traceability is complete. The requirements define 49 requirements (48 Must and one Should) and 60 acceptance criteria. Every requirement has an exact coverage-matrix row, every Must requirement links to one or more defined tests, every acceptance criterion appears in a real test definition, and every test referenced by the matrix is defined. The test levels, data, automation targets, manual checks, gaps, and suggested order are practical for the existing Go and Flutter test structure.

No blocking contradiction, missing Must coverage, or unresolved product question was found. The three findings below are non-blocking coding-plan clarifications for high-risk backfill/recovery behavior and clean TDD ordering. Approval is for coding-plan work; implementation still requires explicit approval after the plan is reviewed.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Risk / Tests | Membership activation and block backfill are correctly fail-closed, but crash/restart recovery is recorded only as a gap follow-up rather than as an explicit subcase of a stable integration-test ID. A pause/failure test does not by itself prove that persisted activation state recovers safely after process restart. | `01-requirements.md` RISK-012; `02-acceptance-tests.md` IT-008, IT-025, GAP-005 | In the coding plan, add process-restart and retry subcases to IT-008 and IT-025 using real persisted activation/backfill state. Schedule them before membership activation is considered implementation-complete. |
| DR-002 | Suggestion | Tests / Traceability | The phrase “joining DID whose repository contains an inbound block” can obscure record ownership. A repository contains blocks authored by its owner; a block inbound to the joining DID is owned by an already-current member and must already be retained/indexed or separately recovered. | `01-requirements.md` FR-005, AC-013, AC-059; `02-acceptance-tests.md` IT-007, IT-008, TD-003 | Define two named fixtures in the coding plan: (1) a joining repository's outbound block targeting a current member, and (2) an already-current member's retained outbound block targeting the joining DID. Assert both before activation completes. |
| DR-003 | Suggestion | Coding-plan readiness | The recommended first failing test, IT-002, depends on schema support whose explicit contract is IT-035. Writing both tests first is sound, but implementing IT-002 before the migration invariant is established could mix schema, store, and immediate-policy failures in one red step. | `02-acceptance-tests.md` IT-002, IT-035, Handoff To Document Review | In the coding plan, write IT-035 and IT-002 together, make the first green step only the minimum reversible schema and owner-scoped store behavior, then add handler-level immediate enforcement. Preserve IT-002 as the first feature-behavior test. |

## Traceability Review

- Planning to requirements: The initial request, discovery findings, confirmed decisions, rejected approaches, recommended Option A, goals, non-goals, risks, assumptions, and open-question status are embedded in `01-requirements.md`; no separate `00-initial-prompt.md` is present or required. The recommended direction is preserved without adding a local lexicon, third-party private-mute dependency, client-only security boundary, or destructive public-record behavior.
- Requirements to acceptance criteria: All 49 requirement rows link to one or more acceptance criteria. The 48 Must requirements and NFR-004 (Should) are represented. The 60 acceptance criteria are externally verifiable and retain the required user, API, persistence, privacy, lifecycle, notification, push, observability, and membership outcomes.
- Acceptance criteria to tests: All 60 acceptance criteria occur in defined acceptance, unit, integration, regression, or manual test entries. The 49 coverage-matrix rows exactly match the acceptance-criteria links in `01-requirements.md`; no matrix test reference is undefined and no test definition lacks requirement and acceptance-criteria IDs.

## Coverage Review

- Must requirements covered: 48 of 48.
- Additional Should coverage: NFR-004 is covered by IT-030 and IT-035 for indexes, query plans, and bounded call counts. GAP-003 correctly records that no numerical performance SLA exists.
- Defined verification: 15 acceptance scenarios, 16 unit tests, 36 integration tests, 9 regression tests, and 2 manual checks.
- Missing or weak coverage: None blocking. DR-001 makes restart recovery explicit; DR-002 removes direction ambiguity from the backfill fixtures. GAP-001 through GAP-005 accurately distinguish platform boundaries and future drift risks from missing functional coverage.
- Manual-only coverage: None. MAN-001 supplements automated localization/semantics tests with real assistive technology. MAN-002 supplements canonical record/indexer tests with a compatible-client/local-PDS smoke test.

## Risk And Approval Review

- Risk level: High.
- Review requirement: Satisfied for progression to coding planning by this review.
- Approval notes: `Approved with notes` does not authorize implementation. The coding plan must incorporate DR-001 and DR-002 explicitly, preserve every security/privacy/membership boundary, and retain a separate explicit approval gate before implementation.
- Principal risks retained for the coding plan: omitted policy paths, mute privacy leakage, PDS/Tap divergence, historical-block recovery, dense-filter pagination, notification delivery races, indirect-reference leakage, membership/backfill races, owner-versus-subject lifecycle cleanup, and multi-account late completion.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Inventory the concrete migration/store/policy seams and write the IT-035 plus IT-002 red tests, then define the shared relationship decision contract before route or Flutter work.
- Blocking issues: None.
- Required planning clarifications: Bind crash/restart cases to IT-008/IT-025 and distinguish joining-owned outbound blocks from existing-member-owned blocks targeting the joining DID.

## Notes For Next Stage

- Keep one central, auditable AppView relationship policy and enumerate every read, write, notification, newness, badge, push, deep-link, quote/repost, and third-party reference consumer.
- Treat current membership as a query/write eligibility predicate, not as a reason to delete public records or another account's private mute preference.
- Separate canonical PDS block state, synchronous pre-Tap enforcement state, and Tap reconciliation generations so late events cannot resurrect an older relationship.
- Make mute ownership and telemetry privacy visible in schema, query, handler, cache/provider, and observability tasks—not only in endpoint tests.
- Preserve DID/account-generation ownership through Flutter mutations, branch reveal, refresh invalidation, cached content, notification counts, and late completions.
- Put notification creation, list/newness filtering, pending-delivery cancellation, final pre-send eligibility, retained-history restoration, and no-push-replay behavior in distinct plan steps with transaction boundaries called out.
- Include explicit query/index and dense-pagination verification before UI polishing.
- Do not add or change anything under `lexicon/`; use the canonical `app.bsky.graph.block` collection and the existing generated external type or a maintained upstream type where available.
