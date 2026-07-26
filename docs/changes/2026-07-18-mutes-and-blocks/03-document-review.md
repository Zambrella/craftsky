# Document Review: Account Mutes And Blocks

## Verdict

Status: Approved with notes  
Reviewer: Codex  
Date: 2026-07-19  
Risk level: High

## Summary

The requirements and acceptance-test specification are consistent with the revised direction: private AppView mutes, public interoperable `app.bsky.graph.block` records, `craftsky_profiles` as the sole membership predicate, PDS-confirmed public writes, account-scoped optimistic Flutter state, ordinary Tap convergence for server policy, non-destructive record retention, and one shared indexed-state relationship policy.

Traceability remains complete. All stable requirement, acceptance-criterion, and test IDs are preserved. The revised tests explicitly distinguish immediate Flutter behavior from delayed AppView enforcement, require no synchronous `atproto_blocks` projection, cover rapid unblock before indexing, and verify both membership/backfill record-owner directions without inventing an activation state machine.

No blocking contradiction, missing Must coverage, or unresolved product question was found. The findings below are coding-plan guardrails. Approval is for coding-plan work; implementation still requires explicit approval.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Contract / Tests | The accepted Tap window means Flutter can show a PDS-confirmed block while raw AppView reads still show the prior indexed state. An immediate refresh must not reverse the successful client state. | `01-requirements.md` FR-006, FR-025, AC-014, AC-036; `02-acceptance-tests.md` IT-009, UT-011 | Model a confirmed account-scoped client overlay that wins over stale pre-Tap responses and retires when matching indexed state is observed. |
| DR-002 | Important | API / PDS | A rapid unblock can arrive before the create event exists in `atproto_blocks`, so a local-only lookup cannot identify the record to delete. | `01-requirements.md` FR-003, FR-006, EC-015; `02-acceptance-tests.md` IT-006 | Return URI/CID/rkey from block creation and add a narrow authenticated PDS record lookup fallback by subject when the local index misses; delete the exact owned rkey idempotently. |
| DR-003 | Suggestion | Membership / Backfill | Removing activation persistence intentionally allows joining-owned historical blocks to converge after profile visibility, while already-indexed inbound blocks apply immediately when the profile row appears. | `01-requirements.md` FR-005, AC-013, AC-059, RISK-012; `02-acceptance-tests.md` IT-008, IT-025, GAP-005 | Keep `craftsky_profiles EXISTS` as the only membership predicate, request ordinary Tap tracking/backfill at join/rejoin, and observe eventual backfill failures without adding readiness state. |

## Traceability Review

- Planning to requirements: The user-approved consistency contract is recorded in Q11, Option A, G-008, FR-005–FR-006, FR-025, the acceptance criteria, edge cases, persistence impact, risks, and test handoff.
- Requirements to acceptance criteria: Every Must requirement retains at least one acceptance criterion. AC-010 and AC-014 now specify PDS-confirmed responses without synchronous projection; AC-013 and AC-059 use profile-row membership plus ordinary Tap convergence.
- Acceptance criteria to tests: IT-005 proves no synchronous block upsert, IT-006 covers rapid pre-index unblock, IT-009 and UT-011 cover stale-refresh-resistant Flutter state, and IT-008/IT-025 cover membership/backfill convergence.

## Coverage Review

- Must requirements covered: 48 of 48.
- Additional Should coverage: NFR-004 remains covered by IT-030 and IT-035.
- Defined verification: 15 acceptance scenarios, 16 unit tests, 36 integration tests, 9 regression tests, and 2 manual checks.
- Missing or weak coverage: None blocking. GAP-005 accurately records the deliberately accepted join/backfill convergence window.
- Manual-only coverage: None. MAN-001 and MAN-002 supplement automated accessibility and live interoperability coverage.

## Risk And Approval Review

- Risk level: High.
- Review requirement: Satisfied for progression to coding planning.
- Approval notes: The plan must not reintroduce `block_write_intents`, synchronous block projection, `craftsky_membership_activations`, or a second membership predicate.
- Principal risks retained: omitted policy paths, private mute leakage, stale optimistic UI replacement, Tap lag/failure, rapid pre-index unblock, historical-block convergence, dense pagination, notification delivery races, indirect-reference leakage, lifecycle cleanup, and multi-account late completion.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Define the minimal reversible `actor_mutes` and `atproto_blocks` schema, then test the PDS-confirmed/no-local-upsert block handler contract and Tap indexer independently.
- Blocking issues: None.
- Required planning clarifications: Specify overlay retirement, exact PDS lookup fallback, and join/rejoin Tap resync without persisted activation.

## Notes For Next Stage

- Keep one central AppView policy over indexed mute/block state and enumerate every read, write, notification, newness, badge, push, deep-link, quote/repost, and third-party reference consumer.
- Keep canonical PDS writes, indexed AppView state, and confirmed Flutter overlays as three explicit consistency layers.
- Use only `actor_mutes` and `atproto_blocks` for new persistence; membership remains `craftsky_profiles EXISTS`.
- Preserve DID/account-generation ownership through optimistic mutation, stale refresh, branch reveal, cached content, notification counts, and late completion.
- Add bounded Tap lag/backfill failure observability without target identifiers.
- Do not add or change anything under `lexicon/`; use canonical `app.bsky.graph.block` types.
