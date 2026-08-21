# Implementation Review: AT Protocol Identity Freshness

## Verdict

Status: Changes required
Reviewer: Codex
Date: 2026-08-20
Risk level: Medium

## Summary

The implementation correctly separates cached presentation resolution from
fresh authoritative resolution, wires Tap identity events to a durable refresh
trigger, keeps Tap-provided handles out of the verified index, and invalidates
Indigo's local DID/handle entries without remote work in the ACK path. The
reported PostgreSQL, race, compile, vet, Staticcheck, formatting, and diff gates
are green.

One ordering race remains in the new event-driven refresh path. A refresh that
selected an older Tap trigger can finish after a newer identity event commits
and then either delay or delete the newer trigger. That contradicts the Must
requirement that newer events schedule immediate work and means the periodic
scan, rather than the event-driven path, may be required to repair the mapping.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior | Refresh finalization is not fenced to the trigger that was selected. `refreshCandidates` does not return `tap_event_id`; after resolution, `deferRefresh` updates whatever state row currently exists and `clearRefreshState` deletes it unconditionally. If event 700 is in flight and event 701 commits, failure of 700 changes 701 from immediate `pending` to delayed `retry`, while success of 700 deletes 701's trigger and may write the older resolution result first. | ID-010, ID-012, AC-007; IT-ID-007; `appview/internal/api/identity_cache_refresh.go:63-102`; `appview/internal/api/identity_cache_store.go:222-279`; `appview/internal/ingestion/store.go:501-517` | Carry the selected refresh provenance/version through the candidate. Make successful write-through plus trigger completion conditional on that unchanged provenance, and make failure deferral conditional as well. If a newer event replaced the row, the older worker must not write, clear, or delay it. Add deterministic required-PostgreSQL barriers for both success-after-newer-event and failure-after-newer-event. |

## Requirement And Test Traceability

- Requirements implemented: ID-001 through ID-009 and ID-011 are implemented
  with matching tests and production wiring. ID-010 and ID-012 are implemented
  for transaction atomicity, ordinary redelivery, local invalidation, and
  authoritative write-through, but are incomplete under an in-flight refresh
  racing a newer Tap event.
- Tests implemented: UT-ID-001, UT-ID-002, IT-ID-003, UT-ID-004, IT-ID-005,
  REG-ID-006, IT-ID-007, UT-ID-008, and REG-ID-009.
- Unplanned behavior: none identified. Migrations 000053 and 000054, refresh
  configuration, lifecycle inventory, observations, and cache invalidation all
  map to approved requirements.
- Remaining gaps: IT-ID-007 covers same-event redelivery and a newer event after
  a completed attempt, but not a newer event that commits while an older
  resolver call is paused.

## Test Evidence

- Commands reviewed:
  - Required-PostgreSQL `go test` for `internal/ingestion`, `internal/tap`,
    `internal/api`, `internal/app`, and `internal/db`.
  - The same package set with `go test -race`.
  - Whole-module compile-only `go test ./... -run '^$' -count=1`.
  - `go vet` and repository-pinned Staticcheck 2026.1 over the affected
    packages.
  - `gofmt -l` over the affected files and repository `git diff --check`.
- Passing evidence: every command above passed. Required-PostgreSQL tests used
  the local PostgreSQL test instance at port 15747.
- Failing or skipped tests: no existing implementation test remained failing
  or skipped. The missing in-flight event-order barriers are the blocking test
  gap identified by this review. The first sandboxed PostgreSQL and Staticcheck
  attempts were environment-blocked; their approved reruns passed.

## Risk Review

- Risk level: Medium.
- Risk notes: security-sensitive handle-selected operations still use fresh
  authoritative resolution, so this race does not make stale PostgreSQL data
  authoritative for deletion, OAuth, mentions, or mutations. It can keep
  presentation/search identity stale and defeats the requested immediate
  event-driven convergence until periodic repair.
- Approval notes: apply migrations 000053 and 000054 before starting the
  updated AppView. Approval is blocked until refresh completion and deferral
  are generation/provenance safe under a newer Tap event.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: no Flutter or user-facing visual surface changed.
- Suggested polish notes: none.

## Handoff Back To TDD Builder

- Required fixes: close IR-001 with trigger-provenance conditional write,
  completion, and retry deferral.
- Suggested next failing test: pause event 700's authoritative resolver after
  candidate selection, ingest event 701, then separately release event 700 as
  success and failure. In both cases assert event 701 remains immediate
  `pending`, its attempt count is unchanged, and event 700 cannot overwrite or
  clear the newer state or verified mapping.
- Verification to rerun: focused new PostgreSQL barriers; full required-
  PostgreSQL ingestion/API/app/migration packages; the same set with `-race`;
  whole-module compile-only; vet; Staticcheck; gofmt; diff check.
