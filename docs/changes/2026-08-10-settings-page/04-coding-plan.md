# Coding Plan: Settings Page And Lean Account Deletion

## 1. Inputs

- Requirements: `01-requirements.md` (Approved, High risk, product-owner implementation approval 2026-08-11)
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)

## 2. Implementation Strategy

Retain the completed Settings/About/Account and fresh-reauth UI. Replace the observable multi-phase deletion subsystem with a single replayable owner job:

1. Fresh reauth creates an owner/job-bound intent.
2. Exact-handle acceptance atomically binds the OAuth session and revokes ordinary access.
3. Flutter treats `202 Accepted` as handoff, clears the account locally, and navigates through existing MRU/Sign-in behavior.
4. Every worker attempt reruns all idempotent private cleanup components, repeatedly scans/deletes registered CraftSky PDS records until empty, and then atomically removes the bound OAuth session and operation.
5. Existing Tap/indexers consume delete events independently; the job neither observes nor waits for them.

Because migration `000037` is not deployed and the app has no users, rewrite it in place to one operation table plus OAuth request metadata.

## 3. Affected Areas

| Area | Existing Branch State | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Migration | Eight deletion tables | Keep `account_deletion_operations`; remove status/recovery/expected/receipt/checkpoint/artifact/audit tables | FR-025, RULE-010 | IT-029 |
| Store/job | Multi-phase/status/audit store | Minimal intent/accept/claim/retry/finalize store; terminal deletes operation | FR-017, FR-020, FR-021, FR-025 | IT-009, IT-016, IT-032 |
| PDS deletion | Per-record registrar/marker | Narrow repeated scan/delete with no per-record persistence | FR-015, RULE-002, RULE-005, RULE-009 | UT-009, UT-010, IT-010, IT-011 |
| Private cleanup | Component checkpoints/artifacts/manifest | Replay all explicit idempotent components every attempt; keep current-store isolation tests | FR-015, FR-026, NFR-002 | IT-012, IT-013, IT-031 |
| Lifecycle | Convergence/terminal gates | One attempt: private cleanup → PDS empty → finalize | FR-015, FR-023, RULE-007 | IT-027, IT-030 |
| Tap/index | Receipt observer and must-replay sentinel | Remove deletion-specific coupling; preserve ordinary indexer semantics | FR-023, RULE-008 | REG-008, REG-010, REG-013 |
| API/routes | Status/retry/recovery routes | Keep start intent, cancel unaccepted intent, accept; remove status/retry/recovery | FR-021, FR-027 | IT-033 |
| Auth | Fresh deletion purpose and pending-login branch | Keep atomic metadata and no-ordinary-session branch; no status handoff token | FR-019, FR-024 | UT-024, IT-006, IT-020 |
| Flutter deletion model | Status DTO/registry/capability | Intent/acceptance only; `accept` returns success without job/status DTO | FR-016, FR-027 | UT-012, IT-034 |
| Flutter routes/UI | Status page, pending page, polling, switcher row | Remove; keep Account and reauth completion/confirmation | FR-016, FR-027 | REG-014, REG-015 |
| Flutter 401 | Deletion-specific recovery | Restore ordinary account-scoped invalidation | FR-022 | REG-016 |
| Observability | Detailed deletion metrics and audit sweeper | Ordinary redacted logs only | FR-017, FR-025, NFR-006 | IT-032 |

## 4. Files And Modules

### Keep And Simplify

| Path / Module | Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000037_account_deletion.*.sql` | Rewrite | Minimal schema | FR-020, FR-025 | IT-029 |
| `appview/internal/accountdeletion/store.go` | Rewrite/narrow | Intent, acceptance, lease, retry, finalization | FR-017, FR-020, FR-021, FR-025 | IT-009, IT-016, IT-032 |
| `appview/internal/accountdeletion/app_service.go` | Narrow | Start/cancel intent and accept only | FR-013, FR-020 | UT-024, IT-006 |
| `appview/internal/accountdeletion/pds_deleter.go` | Remove registrar | Repeated owner/collection scan | FR-015 | UT-010, IT-010 |
| `appview/internal/accountdeletion/private_cleanup.go` | Remove checkpoints | Replay explicit components | FR-015, FR-026 | IT-012, IT-031 |
| `appview/internal/accountdeletion/lifecycle.go` | Collapse phases | Full attempt then finalize | FR-015, FR-023, FR-025 | IT-027, IT-030 |
| `appview/internal/accountdeletion/worker.go`, `retry.go` | Simplify | Automatic capped server retry only | FR-017 | UT-007, IT-016 |
| `appview/internal/api/account_deletion.go` | Narrow | Intent/cancel/accept handlers; `202` no status capability | FR-013, FR-016, FR-021 | IT-033, IT-034 |
| `appview/internal/auth/account_deletion_reauth.go` and handlers | Narrow | Fresh intent completion and pending-login denial | FR-019, FR-024 | UT-024, IT-006, IT-020 |
| `appview/internal/app/deps.go`, `cmd/appview/main.go` | Simplify wiring | One worker; no status signer/audit sweeper/metrics | FR-017, FR-025 | IT-032 |
| `appview/internal/scheduledposts/account_deletion.go` | Remove job artifact coupling | Reuse idempotent scheduled cleanup jobs | FR-015 | IT-031 |
| `app/lib/auth/data/account_deletion_repository.dart` | Narrow | Start/cancel/accept only; void acceptance | FR-013, FR-016 | IT-034 |
| `app/lib/settings/providers/account_deletion_controller.dart` | Narrow | Fresh intent and immediate accepted cleanup | FR-016, FR-022, FR-027 | IT-034 |
| `app/lib/settings/services/account_deletion_acceptance_coordinator.dart` | Simplify | Local cleanup then ordinary account removal/MRU | FR-016, FR-022 | IT-034 |
| `app/lib/router/router.dart`, `route_locations.dart`, generated route | Remove status/pending routes | Keep reauth callback only | FR-027 | REG-015 |
| `app/lib/l10n/*` | Remove obsolete status/retry strings; update boundary copy | Accurate lean UI | FR-014, FR-027 | UT-005, UT-022 |

### Delete As Superseded

- Backend: `audit*`, `convergence*`, `index_receipt*`, `private_manifest*`, `status_capability*`, `status_authorization_test.go`, `terminal*`, deletion-specific observability/metric adapter, and recovery/status middleware/routes/tests that have no lean contract.
- Flutter: deletion status registry/provider/storage, status refresh host, status page, pending-status page, deletion switcher projection/branches, status-specific 401 recovery, and their tests/generated providers.
- Keep removal regressions in surviving migration/route/router/switcher/interceptor tests rather than tests for deleted implementations.

## 5. Services, Interfaces, And Data Flow

```text
POST start-intent → OAuth accountDeletion redirect → proof
POST accept(proof, exactHandle)
  transaction:
    activate one operation + bind fresh OAuth
    revoke/delete ordinary CraftSky sessions
    delete unbound OAuth sessions
  return 202

Flutter on 202:
  clear local product data
  remove ordinary account
  activate MRU retained account or SignedOut

worker claim:
  privateCleaner.Run(owner) // all components, idempotent
  pdsDeleter.DeleteAll(owner) // repeated first/full scans; membership last
  store.FinalizeSuccess(job, owner) // delete bound OAuth + operation
on error:
  schedule next attempt with capped backoff; log coarse category
```

Key partial interfaces:

```go
type PrivateCleanupComponent interface {
    Name() string
    Purge(context.Context, syntax.DID) error
}

type WorkerStore interface {
    ClaimDue(context.Context, string, time.Duration) (ClaimedOperation, bool, error)
    RecordFailure(context.Context, ClaimedOperation, time.Time, ErrorCategory, int) error
    CompleteAttempt(context.Context, ClaimedOperation) error
}
```

`CompleteAttempt` calls terminal finalization after the processor succeeds; no status/audit projection is returned.

## 6. State, Providers, Controllers, Or DI

- Delete `DeletionStatusRegistry`, its secure storage/provider, and app-level polling.
- Restore `SessionRegistry`/account switcher to ordinary accounts only.
- `AccountDeletionController` retains the captured lease and pending OAuth proof but has no durable deletion status state after acceptance.
- The acceptance coordinator reuses existing account removal/MRU initialization and local cleaner. It does not write a deletion status snapshot.
- AppView DI constructs one `Store`, lean `AppService`, replaying `PrivateCleaner`, `LifecycleProcessor`, and `Worker`; no signer, convergence verifier, telemetry adapter, or audit sweeper.

## 7. UI, Routes, And User-Facing Surfaces

- Settings/About/Account layout remains.
- Keep the fresh-reauth callback page only until exact-handle confirmation/acceptance.
- On `202`, navigate according to the newly active ordinary account or Sign in. Show a brief localized acceptance acknowledgement before/while leaving the deleting account.
- Remove deletion status page, progress/attention copy, manual Retry, deletion rows in account switcher, and global refresh host.
- A pending login uses the existing auth completion error/outcome surface with coarse “Deletion is already in progress” copy and no ordinary bearer.

## 8. Error, Loading, Empty, And Edge States

| Case | Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Cancel before exact-handle submission | Delete intent/auth request; ordinary account unchanged | FR-013, RULE-006 | AT-003 |
| Duplicate acceptance | Resume/return accepted for the same owner job when authenticated proof is valid | FR-020 | IT-009 |
| Lost acceptance response | Revoked session eventually follows normal invalid-session cleanup; server job continues | FR-016, FR-022 | REG-016 |
| Transient private/PDS failure | Replay whole attempt after capped delay | FR-017, NFR-002 | IT-016, IT-031 |
| Missing PDS record | Treat as already deleted | FR-015 | UT-010 |
| Tap/index unavailable | No effect on worker completion | FR-023 | IT-030 |
| Pending DID logs in | No ordinary session; coarse pending outcome | FR-024 | IT-020 |
| Terminal success | Delete bound OAuth and operation; retain no audit/status/receipt/checkpoint state | FR-025 | IT-032 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Initial Expected Failure |
|---|---|---|---|
| 1 | IT-029 | migration contract | Seven superseded tables still exist. |
| 2 | UT-010 / IT-010 | PDS deleter | Constructor still requires registrar and persists expected markers. |
| 3 | IT-031 | private cleanup | Cleaner still requires checkpoint store/artifacts. |
| 4 | IT-030 / IT-027 | lifecycle acceptance | Lifecycle requires convergence verifier/receipts and audit finalization. |
| 5 | UT-007 / IT-016 / IT-032 | retry/finalize | Worker can enter attention/manual states and terminal retains audit/metrics. |
| 6 | IT-033 / REG-013 | routes/Tap | Status routes and receipt observer remain registered. |
| 7 | IT-034 / REG-014–REG-016 | Flutter acceptance | Controller writes status registry; switcher/router/interceptor retain status behavior. |
| 8 | Existing security/Settings regressions | focused/broad suites | Must remain green while obsolete code is removed. |

## 10. Sequencing And Guardrails

- First TDD step: IT-029 minimal migration contract.
- Remove dependencies before deleting source files so compile failures stay local and meaningful.
- Keep owner/collection typed validation, membership-last inventory, fresh auth purpose, atomic binding, and private cleanup throughout.
- Do not add eager AppView deletion or alter existing indexer handlers.
- Do not run a destructive personal/production PDS test.
- No Lexicon changes.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Future private store drift lacks executable manifest. | Possible omitted cleanup. | Accepted; explicit current-store integration fixture and future schema review. |
| CPQ-002 | Non-blocking | Persistent OAuth failure lacks user recovery UI. | Operation may wait for operator/later login. | Accepted for pre-production; keep pending-login denial and monitoring. |
| CPQ-003 | Resolved | Is migration rewrite safe? | Would need follow-up migration if deployed. | AGENTS states no production/users; rewrite `000037` in place. |

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with: IT-029
- Focused command: `cd appview && TEST_DATABASE_URL=... go test ./internal/db -run TestAccountDeletionMigration -count=1`
- Notes: The product owner explicitly authorized this high-risk implementation. No blocking questions remain.
