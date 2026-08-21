# TDD Implementation Plan: AT Protocol Identity Freshness

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation rules

- Every change links to ID-001 through ID-012.
- Write one focused failing test before each behaviour.
- Use the hardened federated boundary for both directory modes.
- Never fall back from an authoritative lookup to stale cached data.
- Preserve unrelated notification changes in the shared worktree.

## Test order

| Step | Test ID | Requirement IDs | Expected initial state |
|---|---|---|---|
| 1 | UT-ID-001 | ID-001, ID-002, ID-006 | Fails: only cached directory exists |
| 2 | UT-ID-002 | ID-003, ID-009 | Fails: fresh-looking DB row bypasses resolution |
| 3 | IT-ID-003 | ID-004, ID-009 | Fails: deletion trusts Postgres handle |
| 4 | UT-ID-004 | ID-005 | Fails: all routes share cached resolver |
| 5 | IT-ID-005 | ID-007, ID-008, ID-009 | Fails: no bounded refresh processor |
| 6 | REG-ID-006 | ID-001..ID-009 | Passes: focused, PostgreSQL, race, vet, Staticcheck, formatting, and compile gates are green |
| 7 | IT-ID-007 | ID-010, ID-012 | Fails: ordinary identity events only persist receipts |
| 8 | UT-ID-008 | ID-011 | Fails: Tap does not invalidate Indigo's process-local cache |
| 9 | REG-ID-009 | ID-010..ID-012 | Passes: required-PostgreSQL, race, vet, Staticcheck, formatting, diff, and compile gates are green |
| 10 | IT-ID-010 | ID-010, ID-012; AC-007; IR-001 | Fails: an older in-flight refresh can clear or delay a newer Tap trigger |

## Execution log

### UT-ID-001
- Status: passed.
- RED: the composition test failed because `federatedClients` exposed only the
  cached directory.
- GREEN: the cached directory now wraps the same hardened uncached base used by
  the authoritative capability; an architecture test also fixes OAuth to the
  authoritative directory.

### UT-ID-002
- Status: passed.
- RED: a fresh-looking persistent mapping for a reassigned handle returned the
  old DID without consulting the directory.
- GREEN: exact mention resolution always performs fresh handle-to-DID and
  DID-to-handle verification, then lifecycle-fenced write-through. Definitive
  misses remain 404; transient authority failures return retryable 503.

### IT-ID-003
- Status: passed against PostgreSQL.
- RED: deletion intent creation selected the confirmation handle directly from
  `atproto_identity_cache`.
- GREEN: intent creation resolves the authenticated DID authoritatively, stores
  the canonical mapping, and creates no intent/OAuth flow/lifecycle transition
  during an identity outage.

### UT-ID-004
- Status: passed.
- RED: follow, unfollow, mute, unmute, block, unblock, and report route
  constructors all received the cached display resolver.
- GREEN: those mutation capabilities and the operator backfill receive the
  authoritative resolver; presentation/read routes retain the cached resolver.

### IT-ID-005
- Status: passed against PostgreSQL.
- RED: no durable bounded refresh processor or retry state existed.
- GREEN: migration 000053, the refresh processor, configuration, startup and
  shutdown wiring, lifecycle inventory, retry deferral, non-starvation, and
  bounded non-PII metrics are implemented. The test covers stale, missing,
  failed, successful, and terminal candidates.

### REG-ID-006
- Status: passed.
- Required-PostgreSQL full affected packages passed for API, account deletion,
  app composition, routes, migrations, owner lifecycle, and observability.
- The same package set passed with `-race`.
- Full CLI, AppView startup, and auth regression packages passed.
- `go vet` and repository-pinned Staticcheck 2026.1 passed for all affected
  packages.
- Whole-module compile-only `go test ./... -run '^$'`, `gofmt -l`, and
  `git diff --check` passed.

### IT-ID-007
- Status: passed against PostgreSQL.
- RED: an ordinary identity event still succeeded and committed its receipt
  when the durable refresh-state table was unavailable; no immediate refresh
  trigger existed.
- GREEN: migration 000054 adds ordered Tap-event provenance and `pending`
  refresh state. Ordinary identity ingestion now commits the receipt and
  immediate refresh trigger in one transaction. Same-event redelivery cannot
  reset retry state, while a newer event schedules immediately. The existing
  WebSocket consumer contract continues to ACK only after the durable ingestor
  returns.

### UT-ID-008
- Status: passed.
- RED: the composed identity directory retained its DID, old-handle, and
  hinted new-handle entries after a Tap identity event.
- GREEN: Tap ingestion purges those process-local Indigo entries only after
  the durable transaction commits and performs no resolution in the ACK path.
  The authoritative refresh worker repeats invalidation after its verified
  write, closing the crash window between enqueue and local invalidation.
- A PostgreSQL regression proves a Tap-triggered refresh bypasses the normal
  24-hour freshness threshold, verifies the new mapping authoritatively, and
  replaces the old persistent handle.

### REG-ID-009
- Status: passed.
- Required-PostgreSQL package tests passed for ingestion, Tap, API, app
  composition, and migrations.
- The same package set passed with `-race`.
- `go vet` and repository-pinned Staticcheck 2026.1 passed for all affected
  packages.
- Whole-module compile-only `go test ./... -run '^$'`, `gofmt`, and
  `git diff --check` passed.

### IT-ID-010
- Status: passed against PostgreSQL.
- Review correction: pause an older authoritative refresh after candidate
  selection, commit a newer Tap identity event and a newer verified mapping,
  then release the older attempt as success and failure. The newer trigger must
  remain immediate and the older attempt must not overwrite, clear, or defer
  it.
- RED: success deleted the newer event's refresh row and overwrote its verified
  mapping; failure changed the newer `pending` row to delayed `retry` with an
  incremented attempt count.
- GREEN: candidate selection now durably ensures a refresh-state row and
  carries its nullable Tap event provenance. Failure deferral updates only the
  selected provenance. Successful verified write-through, provenance recheck,
  and state deletion share one lifecycle-fenced transaction; a superseded
  attempt performs none of those mutations.
- Focused required-PostgreSQL success/failure barrier passed. Nearby identity
  refresh, invalidation, receipt, redelivery, and terminal identity tests also
  passed.

### Post-review correction verification

- Required-PostgreSQL full affected packages passed for ingestion, Tap, API,
  app composition, and migrations.
- The same package set passed with `-race`.
- Whole-module compile-only `go test ./... -run '^$' -count=1` passed.
- `go vet` and repository-pinned Staticcheck 2026.1 passed over all affected
  packages.
- `gofmt -l` and repository `git diff --check` passed.
- The existing `06-implementation-review.md` retains its `Changes required`
  verdict as the historical correction input; a fresh implementation-review
  pass is required to close IR-001.

## Pre-production migration note

- Apply migrations 000053 and 000054 before starting the updated AppView.
- Existing refresh rows remain retryable. Rows created by ordinary identity
  events after 000054 carry the newest Tap event ID; no historical Tap handle
  is trusted or backfilled into the verified index.
- The periodic bounded scan intentionally remains enabled so pre-migration
  identities and any missed local invalidation still converge.

## Completion checklist

- [x] All Must requirements covered
- [x] All planned tests passing
- [x] Relevant regressions passing
- [x] Documentation and configuration examples updated
- [x] No unlinked behaviour implemented
- [ ] Implementation review completed or explicitly skipped
