# Coding Plan: Settings Page And Lean Account Deletion

> **AppView audit correction (2026-08-14):** Section 13 is the authoritative implementation design for remote outcome uncertainty. It replaces the earlier “one operation table / finalize after first empty scan” assumptions with the approved temporary exact-key safety tombstones while preserving every lean user-facing and final-minimization boundary.

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

## 13. AppView Audit Correction Plan: Crash-Safe Exact-Key Convergence

### 13.1 Inputs and implementation strategy

- Requirements amendment: `01-requirements.md` §24 (Approved, High risk, product-owner decision 2026-08-14).
- Acceptance amendment: `02-acceptance-tests.md` §12 (AT-007–AT-009, UT-025–UT-026, IT-035–IT-038, REG-017).
- Document re-review: `03-document-review.md` §7 (`Approved`).
- Audit design: `docs/2026-appview-code-audit-plan/AV-002_AV-003_AV-006_AV-007-account-lifecycle.md`.

Keep the existing client contract and lean private-cleanup replay. Correct only the server's cross-system completion proof:

```text
ordinary PDS/object effect
  persist deterministic attempt + owner generation + exact remote key
  cross remote acceptance boundary
  settle locally OR remain outcome-uncertain after crash

accepted deletion, under owner/job fence
  freeze new ordinary effects
  adopt every unresolved registered-collection/object attempt as an exact-key tombstone
  replay private cleanup
  reconcile each exact key without repeating its original write
  run registered-collection deletion/final scan
  if any key lacks settlement proof: keep operation reconciling
  if all keys settled: atomically delete tombstones + operation + deletion-only OAuth
```

An AppView advisory lock disappearing, an outbound client deadline passing, an absent `HEAD`, or an empty first PDS scan is not settlement proof. The PDS/object adapter may mark a tombstone settled only from proved pre-acceptance failure or from a tested server-side settlement boundary followed by a final exact-key delete/absence check. Otherwise bounded reconciliation continues.

### 13.2 Migration and storage contract

Reserve the next audit migration number centrally and add:

- `appview/migrations/<next>_account_deletion_safety_tombstones.up.sql`
- `appview/migrations/<next>_account_deletion_safety_tombstones.down.sql`

Create `account_deletion_safety_tombstones` with this minimized contract:

| Column / constraint | Purpose |
|---|---|
| `operation_id`, `owner_did`, `owner_generation` | Exact accepted deletion owner/job/generation; operation FK cascades only during verified finalization. |
| `kind` | Constrained to `pds_record` or `scheduled_object`. |
| `exact_key` | Canonical typed AT URI for PDS or immutable private object key; never a wildcard/prefix selector. |
| `upload_generation` | Null for PDS; positive and required for scheduled objects. |
| `source_attempt_id` | Non-secret deterministic reference to the durable PDS/upload attempt that created the uncertainty. |
| `state` | Constrained to `pending`, `reconciling`, or `settled`; no user-visible progress semantics. |
| `remote_deadline`, `settlement_not_before` | Optional adapter evidence; `settlement_not_before` is populated only from a configured/tested server-side guarantee. |
| `attempts`, `next_attempt_at`, `lease_token`, `lease_expires_at`, `last_result_category` | Bounded worker claiming/retry with lease-field consistency checks and coarse errors. |
| `created_at`, `updated_at`, `settled_at` | Operational reconciliation timestamps; not an audit narrative. |

Use `UNIQUE NULLS NOT DISTINCT (operation_id, kind, exact_key, upload_generation)` plus owner/job/exact-key claim indexes. Checks require registered-owner AT URIs for PDS rows, immutable key + positive generation for object rows, positive owner generation, non-negative attempts, valid state/timestamp combinations, and paired lease fields. Store no handle, OAuth/PDS token, record body/CID payload, media metadata/content, status capability, or client recovery data.

Coordinate rather than duplicate the grouped audit schema:

- `owner_lifecycles` supplies owner state/generation and the exclusive effect fence.
- The ordinary owner-effect attempt relation supplies deterministic PDS URI and remote-boundary outcome.
- Scheduled-media upload attempts supply immutable object key/generation and remote deadline.
- `oauth_sessions.lifecycle_state = deletion_only` and credential generation supply the only PDS deletion capability.
- Final migration numbers and foreign keys must be allocated in the shared lifecycle/auth migration series; the semantic migration name above is fixed even if its numeric prefix changes.

### 13.3 Files and modules

| Path / module | Create / change | Planned correction | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/accountdeletion/state.go` | Change | Add internal reconciling/tombstone value types without exposing a client phase model. | FR-028–FR-031, NFR-007 | UT-025, IT-037 |
| `appview/internal/accountdeletion/store.go` | Change | Atomically adopt exact-key attempts at acceptance/claim, lease tombstones, compare-and-set outcomes, block finalization on unsettled keys, and delete settled rows with operation/OAuth. | FR-028, FR-031, NFR-007 | UT-025, IT-035, IT-038 |
| `appview/internal/accountdeletion/lifecycle.go` | Change | Order private cleanup, exact-key reconciliation, final collection scan, and artifact-free finalization under the owner/job fence. | FR-028–FR-031, RULE-013 | IT-035–IT-038 |
| `appview/internal/accountdeletion/worker.go` | Change | Claim bounded tombstone reconciliation, retain operations without a finite proof, and never promote elapsed timeout to success. | FR-029, FR-030, RULE-013 | IT-035, IT-037, IT-038 |
| `appview/internal/accountdeletion/pds_deleter.go` | Change | Add exact typed URI read/delete/absence reconciliation using only the matching `deletion_only` session; retain full registered collection scan and membership-last ordering. | FR-029, RULE-012 | UT-025, IT-035 |
| `appview/internal/accountdeletion/app_service.go` | Change | On acceptance, fence the owner and bind/adopt all unresolved exact-key attempts before ordinary authority is revoked. | FR-028, RULE-013 | IT-035, IT-038 |
| `appview/internal/accountdeletion/private_cleanup.go` | Preserve/narrow integration | Continue whole-component idempotent cleanup; do not turn safety tombstones into component checkpoints. | FR-031 | REG-017 |
| `appview/internal/scheduledposts/media_service.go` | Change | Persist immutable upload attempt/key/generation/deadline before `Put`; hold owner/object fences through ready CAS. | FR-028, FR-030 | UT-026, IT-036 |
| `appview/internal/scheduledposts/account_deletion.go` | Change | Expose only matching owner/job/generation unresolved object attempts for adoption/reconciliation. | FR-028, RULE-012 | IT-036 |
| `appview/internal/scheduledposts/cleanup_processor.go` | Change | Repeat exact-key delete/absence checks, fence stale leases, and require tested settlement evidence before settled CAS. | FR-030, RULE-013 | IT-036, IT-037 |
| `appview/internal/scheduledposts/store.go`, `store_queries.go`, `tombstone.go` | Change | Persist/claim generation-specific attempt and deletion-tombstone state; prevent key reuse across owner generations. | FR-028, FR-030, NFR-007 | UT-026, IT-036, IT-037 |
| `appview/internal/auth/account_deletion_reauth.go`, `store.go` | Change | Keep exact job/owner/credential-generation `deletion_only` lookup; finalization removes it only with converged operation/tombstones. | FR-029, FR-031, RULE-012 | IT-035, IT-038 |
| `appview/internal/app/deps.go`, `appview/cmd/appview/main.go` | Change | Wire bounded reconcilers and validated settlement configuration; do not add status/audit/metrics services. | FR-029–FR-031, NFR-007 | IT-037, REG-017 |
| `appview/internal/accountdeletion/migrations_test.go` | Change | Assert exact table/columns/checks/indexes and absence of superseded status/receipt/checkpoint/audit tables. | NFR-007, RULE-012 | UT-025, IT-038, REG-017 |

No Flutter, route, response-body, or localization change is required. Acceptance remains `202`; the confirming account is removed locally and pending login remains non-ordinary.

### 13.4 Service boundaries and partial signatures

```go
// Partial signatures only. syntax.ATURI is parsed at the boundary.
type SafetyKey struct {
    Kind             SafetyKind
    URI              syntax.ATURI // pds_record only
    ObjectKey        string       // scheduled_object only
    UploadGeneration int64        // scheduled_object only
}

type SafetyStore interface {
    AdoptUncertainAttempts(ctx context.Context, op ClaimedOperation) error
    ClaimDueSafety(ctx context.Context, leaseDuration time.Duration) (SafetyClaim, bool, error)
    RecordSafetyRetry(ctx context.Context, claim SafetyClaim, next time.Time, category ErrorCategory) error
    MarkSafetySettled(ctx context.Context, claim SafetyClaim, evidence SettlementEvidence) error
    FinalizeIfConverged(ctx context.Context, op ClaimedOperation) (bool, error)
}

type ExactPDSReconciler interface {
    ReconcileDelete(ctx context.Context, deletionCapability DeletionCapability, uri syntax.ATURI, settlement SettlementPolicy) (SettlementEvidence, error)
}

type ExactObjectReconciler interface {
    ReconcileDelete(ctx context.Context, owner syntax.DID, key string, uploadGeneration int64, settlement SettlementPolicy) (SettlementEvidence, error)
}
```

`SettlementEvidence` is an internal typed proof, not arbitrary adapter text. It can express proved-not-accepted or tested-bound-plus-final-absence. It cannot express “client timeout elapsed”, “lock vanished”, or “first lookup absent” as terminal evidence.

### 13.5 Error and edge-state handling

| State / case | Planned handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Known uncertain PDS write; first scan empty | Keep tombstone/operation, retry exact URI, do not finalize. | FR-028, FR-029, RULE-013 | AT-007, IT-035 |
| Delayed PDS record appears | Delete exact URI with deletion-only capability, verify post-settlement absence. | FR-029, RULE-012 | IT-035 |
| Early object delete/HEAD absent; delayed `Put` appears | Keep generation tombstone and repeat exact-key delete/absence verification. | FR-030, RULE-013 | AT-008, IT-036 |
| No tested finite server settlement guarantee | Keep bounded, lease-claimed reconciliation indefinitely; ordinary login remains denied. | FR-029, FR-030, RULE-013 | IT-037 |
| Wrong owner/job/namespace/key/generation | Reject before adapter call and leave unrelated data untouched. | NFR-007, RULE-012 | UT-025, UT-026, REG-017 |
| Crash after settled CAS before finalization | Resume, revalidate all exact keys, then atomically remove temporary state. | FR-031, NFR-002 | IT-038 |
| All exact keys converged | Delete tombstones + operation + matching deletion-only OAuth in one fenced transaction. | FR-031 | AT-009, IT-038 |

### 13.6 TDD implementation order

| Order | Test ID | Target | Setup / fixture | Initial expected failure |
|---|---|---|---|---|
| 1 | IT-035 | `accountdeletion/worker_acceptance_test.go` | Barrier PDS adapter plus deterministic effect attempt/URI | Current worker can finalize after the first empty scan and has no retained exact-URI key. |
| 2 | UT-025 | `accountdeletion/store_test.go`, `migrations_test.go` | Typed PDS rows plus invalid follow/namespace/owner cases | No minimized safety relation or typed exact-key guard exists. |
| 3 | UT-026 | `scheduledposts/tombstone_test.go`, `media_service_test.go` | Generation-specific object attempts and settlement table | Current state cannot represent late `Put` uncertainty safely. |
| 4 | IT-036 | new `scheduledposts/account_deletion_race_test.go` plus MinIO-compatible adapter | Barrier accepted `Put`, crash, early delete, delayed materialization | Early absence lets cleanup forget the key. |
| 5 | IT-037 | account-deletion/scheduled cleanup failure suites | No finite settlement bound | Current retry exhaustion/finalization can discard uncertainty. |
| 6 | IT-038 / AT-009 | worker/store/migration suites | All keys settled plus crash-at-finalize barriers | No atomic post-convergence artifact-removal contract exists. |
| 7 | REG-017 | existing route/Flutter/boundary/residue suites | Full corrected system | Must prove no user status/audit/broader deletion capability returned. |

First TDD command:

`cd appview && go test ./internal/accountdeletion -run 'Test.*Delayed.*PDS' -count=1`

Then run focused PostgreSQL/MinIO packages, `go test -race` for both deterministic barriers, the full AppView test suite, Flutter Settings/auth/router regressions, formatting/static analysis, migration up/down/up, and `git diff --check`. Do not claim MAN-003 or real remote settlement verification unless it actually ran against a disposable development account/backend.

### 13.7 Guardrails and resolved questions

- The product decision is resolved: use temporary minimized exact-key safety tombstones until convergence; do not require an unverified finite settlement/staging assumption.
- Keep the registered `social.craftsky.*` manifest and exact-owner/job/generation deletion-only capability. Never grant account/terminal cleanup authority over `app.bsky.graph.follow`, blobs, another namespace, or a DID/PDS account.
- Safety rows are not per-component private-cleanup checkpoints, Tap/index receipts, a deletion audit, detailed metrics, user status, or recovery credentials.
- Keep PDS/object I/O outside SQL transactions but inside the prescribed owner/object/session advisory fences; use short compare-and-set transactions before/after remote calls.
- No Lexicon change is required.
- No additional product input blocks implementation. Deployment may later supply tested settlement bounds, but absence of one is already specified as continued reconciliation rather than success.

### 13.8 Updated handoff

- Start with: IT-035.
- Migration: centrally numbered `<next>_account_deletion_safety_tombstones.up.sql` / `.down.sql`.
- Focused packages: `appview/internal/accountdeletion`, `appview/internal/scheduledposts`, `appview/internal/auth`, `appview/internal/routes`.
- Completion means both crash barriers pass, indefinite reconciliation is honest, final residue is empty after proven convergence, and every pre-amendment no-status/no-audit/narrow-authority regression stays green.
