# TDD Implementation Plan: Lean Account Deletion Simplification

> **Current status (2026-08-20):** The lean simplification and the approved
> AppView audit exact-key safety correction are implemented. The correction
> execution log and checklist in the final section supersede the 2026-08-14
> pending snapshot; external destructive/remote settlement checks remain
> release gates.

## Inputs

- Requirements: `01-requirements.md` (Approved, High risk)
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Coding plan: `04-coding-plan.md`
- Product-owner approval: 2026-08-11 — explicitly approved the lean asynchronous/no-status design and directed implementation.
- Prior implementation: commit `8372c0ca` contains the superseded receipt/status/checkpoint/audit design and remains the red baseline for removal tests.

## Implementation Rules

- Do not weaken fresh OAuth, captured-account fencing, exact-handle confirmation, owner/collection scope, membership-last deletion, final empty PDS scan, private cleanup, or server-only credentials.
- Write/update one focused test before simplifying each subsystem.
- Run the smallest relevant test first and record a meaningful failure.
- Remove superseded dependencies before deleting their source files.
- Refactor/delete only while focused and nearby tests are green.
- Do not modify Lexicons or add eager AppView deletion.
- Never run destructive tests against a personal, production, or real non-disposable PDS account.
- Do not commit or push unless explicitly requested after this simplification.

## Test Order

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-029 | FR-021, FR-025, RULE-010 | AC-042, AC-046 | Migration still creates seven superseded tables. |
| 2 | UT-010, IT-010, IT-011 | FR-015, NFR-002, RULE-002, RULE-009 | AC-015, AC-017, AC-037, AC-048 | PDS deleter requires per-record registrar/marker persistence. |
| 3 | IT-012, IT-013, IT-031 | FR-015, FR-026, NFR-002 | AC-016, AC-017, AC-037, AC-047 | Private cleaner requires checkpoints/artifacts and executable manifest machinery. |
| 4 | IT-027, IT-030 | FR-015, FR-020, FR-023, FR-025, RULE-007, RULE-008 | AC-023, AC-037, AC-044, AC-046 | Lifecycle waits for convergence receipts/gates and finalizes through an audit. |
| 5 | UT-007, IT-016, IT-032 | FR-017, FR-025, NFR-002, NFR-006 | AC-023, AC-039, AC-046 | Retry can enter needs-attention/manual state and telemetry/audit remain. |
| 6 | IT-033, REG-008, REG-010, REG-013 | FR-021, FR-023, FR-027, RULE-008 | AC-038, AC-039, AC-042, AC-044 | Status/retry/recovery routes and Tap receipt observer remain wired. |
| 7 | UT-012, UT-014, IT-034, REG-014–REG-016 | FR-016, FR-022, FR-027 | AC-018, AC-036, AC-038, AC-043 | Flutter persists/status-polls deleting rows and special-cases former bearers. |
| 8 | UT-009, UT-024, IT-006, IT-009, IT-020, REG-001–REG-007, REG-009, REG-012 | FR-001–FR-021, FR-024, RULE-001–RULE-006, RULE-009, RULE-011 | Security and Settings ACs | Fresh auth, settings, namespace, pending-login, and sign-out behavior must remain green. |

## Implementation Steps

### Step 1: IT-029 — Minimal Migration

- Write/update failing test: Require only OAuth request metadata and `account_deletion_operations`; assert status/recovery/expected/receipt/checkpoint/artifact/audit tables are absent.
- Focused command: `cd appview && TEST_DATABASE_URL=... go test ./internal/db -run TestAccountDeletionMigration -count=1`
- Implement: Rewrite migration `000037` in place and update down migration.
- Refactor: Remove config dependencies that exist solely for status credentials after downstream callers are ready.

### Step 2: UT-010 / IT-010 / IT-011 — Stateless Record Convergence

- Update PDS deleter tests so construction needs only the narrow PDS client and batch size.
- Preserve repeated full scans, owner/collection validation, membership-last ordering, pagination, not-found idempotency, and non-CraftSky/blob prohibitions.
- Remove expected-record registrar and delete-request marker calls.

### Step 3: IT-012 / IT-013 / IT-031 — Replayable Private Cleanup

- Update the component interface to `Purge(ctx, owner)` and require every component to run on every attempt.
- Inject a failure after one component and prove a second full run safely converges with Alice/Bob/shared controls.
- Remove cleanup checkpoint/artifact persistence and executable manifest code/tests.
- Scheduled cleanup continues to enqueue existing idempotent object cleanup jobs without deletion-job artifacts.

### Step 4: IT-027 / IT-030 — Lean Lifecycle

- Update acceptance fixture so no Tap/indexer events or receipts are delivered.
- Collapse lifecycle to private cleanup → repeated PDS deletion/final scan → terminal finalization.
- Simplify store to intent/active operation, lease/retry fields, bound OAuth, and atomic final delete.
- Prove restart, duplicate acceptance, owner/namespace preservation, and zero retained deletion state.

### Step 5: UT-007 / IT-016 / IT-032 — Server-Owned Retry And Finalization

- Make transient/OAuth failures always schedule another attempt using capped backoff; remove needs-attention/manual decisions.
- Remove deletion audit/sweeper and deletion-specific metrics/telemetry.
- Keep ordinary redacted structured logging for worker errors/completion.

### Step 6: IT-033 / REG-013 — Routes And Indexer Independence

- Remove status/Retry/recovery endpoints, status middleware/signing, and corresponding dependency interfaces.
- Remove deletion receipt observer and must-replay sentinel from dispatcher/Tap consumer.
- Run ordinary dispatcher/indexer/Tap tests unchanged.

### Step 7: IT-034 / REG-014–REG-016 — Immediate Flutter Handoff

- Narrow repository/model/controller to intent/cancel/accept only; acceptance has no status capability DTO.
- On `202`, clear local product data, remove ordinary account, and activate MRU/Sign in.
- Remove deletion status registry/storage/providers, status/pending pages, global polling, switcher row branches, status routes, and special 401 recovery.
- Update localized boundary/acceptance/pending copy and regenerate localization/routes.

### Step 8: Preserved Security And Settings Regression Pass

- Re-run fresh OAuth atomic persistence, pending-login denial, collection drift, owner/namespace/blob, Settings/About/Account, switcher, router, cache, sign-out, localization, and accessibility tests.
- Run full AppView suite, focused/full Flutter suite, `dart analyze`, `gofmt -l`, and `git diff --check`.

## Execution Log

| Test IDs | Red Evidence | Green Evidence | Status |
|---|---|---|---|
| IT-029 | Focused migration test reported all seven superseded status/recovery/expected/receipt/checkpoint/artifact/audit tables still existed | Rewritten `000037` creates only `account_deletion_operations` plus atomic OAuth request metadata; owner uniqueness, bound-session FK, down migration, and sentinel preservation pass against disposable Postgres | Completed |
| UT-010, IT-010, IT-011 | Focused tests failed to compile because `NewPDSDeleter` and `DeleteAll` still required an expected-record registrar and job ID | Stateless constructor and repeated owner scan pass owner/namespace/membership-last/pagination/not-found controls; an injected connection loss after the PDS side effect converges on replay without any marker table | Completed |
| IT-012, IT-013, IT-031 | Focused cleanup tests failed to compile while components still required job/checkpoint/artifact arguments | Every private component now replays by owner on every attempt; partial-failure replay, Alice/Bob isolation, scheduled object cleanup enqueueing, and removal of checkpoint/artifact/manifest machinery pass | Completed |
| IT-027, IT-030 | Lifecycle fixtures still required convergence receipts and audit-backed finalization | Lifecycle now runs replayable private cleanup, bound-OAuth PDS deletion through a full empty scan, then atomically removes the operation and bound OAuth; restart and duplicate-acceptance tests pass without Tap events | Completed |
| UT-007, IT-016, IT-032 | Retry tests still expected a needs-attention/manual-retry terminal branch and deletion telemetry | Retry uses capped automatic backoff indefinitely, worker failures retain only coarse categories and ordinary redacted logs, and successful completion leaves no deletion operation or credential | Completed |
| IT-033, REG-008, REG-010, REG-013 | Route and dependency tests referenced status/retry/recovery endpoints, signed capabilities, and the Tap receipt observer | The HTTP surface is start/cancel/accept only; status credentials, middleware, routes, metrics, dispatcher receipts, and Tap replay coupling are absent; route/index/Tap suites pass | Completed |
| UT-012, UT-014, IT-034, REG-014–REG-016 | Flutter tests referenced persisted deletion status, polling pages, deleting switcher rows, and former-bearer recovery | `202` now triggers immediate local cleanup and MRU/Sign-in handoff; repository/controller expose only intent/cancel/accept; status storage/pages/polling/routes and special 401 handling are removed; focused Flutter tests pass | Completed |
| Preserved regressions / broad verification | Focused copy tests caught ambiguous combined PDS-account wording, and AuthComplete success tests lacked a router-safe pending harness | Boundary copy now states each non-deletion guarantee explicitly; `dart analyze` reports no issues, `go test ./... -count=1` passes all AppView packages, `flutter test --reporter compact` passes all 1,449 tests, changed Go files produce no `gofmt -l` output, and `git diff --check` is clean | Completed |

## Manual Release Gate

- Not run in this implementation environment: exercise the flow with a disposable real PDS account and fresh OAuth, verify every registered `social.craftsky.*` collection reaches an empty scan, verify other namespaces and blobs remain, and confirm ordinary login returns only `account_deletion_pending` while the worker is active.
- This is a release/smoke gate, not a remaining code-completion dependency.

## Implementation Review Correction Pass

Input: `06-implementation-review.md` (`Changes required`, IR-001–IR-004).

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| C1 | UT-010, IT-010 | FR-015, RULE-005 | AC-015, AC-030, AC-037 | A deletion-mutated paginated scan can delete membership before a skipped earlier-collection record. |
| C2 | UT-014 | FR-022 | AC-036, AC-043 | The existing test exercises only an unused cleanup-plan model, not the production cleaner. |
| C3 | IT-016, IT-032 | FR-017, FR-021, FR-025, NFR-006 | AC-023, AC-039–AC-042, AC-046 | Worker retry and OAuth authority claims are partly covered by deleted or test-only targets. |
| C4 | IT-033 | FR-021, FR-027 | AC-038, AC-039, AC-042 | Removed status/retry/recovery paths are not requested by a surviving route test. |
| C5 | Structural cleanup | NG-006, FR-025 | AC-046 | Deletion-specific metric APIs and unused policy helpers remain despite the lean contract. |

Correction rules:

- Add one focused failing test before each behavioral correction.
- Keep the one-table/no-status/no-receipt/no-checkpoint architecture unchanged.
- Bind tests to production paths rather than declarative test-only models.
- Update `02-acceptance-tests.md` and this execution log to name the surviving evidence.
- Do not commit or push unless explicitly requested.

### Correction Execution Log

| Test IDs | Red Evidence | Green Evidence | Status |
|---|---|---|---|
| UT-010, IT-010 | Paginated deletion-order test failed with `[post, post, like, profile, post]`, proving membership was removed before a skipped post | Non-membership collections now converge through empty passes before membership is deleted; the complete registry then receives a final empty scan, and focused PDS/restart tests pass | Completed |
| UT-014 | Focused test failed to compile because no production-bound injectable cleaner existed; the prior test exercised only an unused enum/set | The provider now executes `AccountProductDataCleaner`; its focused test proves all draft/Instagram/cache steps run after an earlier failure, while the coordinator test proves the accepted account is still removed and routed | Completed |
| IT-016, IT-032 | Review found the surviving suite lacked direct production-bound assertions for capped Worker scheduling and cross-job/cross-owner OAuth denial | Focused Worker test now proves attempt 100 schedules at the capped delay through `WorkerStore`; Store acceptance test proves only the matching job/owner can resolve bound OAuth and terminal finalization leaves no residue | Completed |
| IT-033 | Review found no surviving requests against the removed status/retry/recovery surface | Route coverage now proves GET status is not allowed, retry/reauth subpaths are 404, and a former status credential cannot authorize the removed recovery endpoint | Completed |
| Structural cleanup | Review found unused deletion metric implementations plus OAuth/local-cleanup/Instagram policy objects exercised only by their own tests | Deletion-specific recorder APIs and metric names are absent; OAuth authorization is tested through Store, local cleanup through the production cleaner, and Instagram deletion through the real terminal owner-purge boundary; focused Go packages pass | Completed |
| Broad correction verification | `dart analyze` initially found one unnecessary null assertion in the new cleaner boundary | Assertion was simplified; `go test ./... -count=1` passes all AppView packages, `flutter test --reporter compact` passes all 1,449 tests, `dart analyze` reports no issues, changed Go files produce no `gofmt -l` output, targeted residue search returns no matches, and `git diff --check` is clean | Completed |

## Completion Checklist

- [x] All retained Must requirements covered by passing tests or documented gaps.
- [x] Seven superseded deletion tables removed; only minimal operation state remains.
- [x] No per-record receipt/expected/marker or Tap acknowledgement coupling remains.
- [x] No cleanup checkpoint/artifact/manifest runtime machinery remains.
- [x] No status/recovery credential, route, Flutter registry/page/polling/switcher row, or manual Retry remains.
- [x] No deletion audit/sweeper or deletion-specific metric adapter remains.
- [x] Fresh OAuth, exact-handle, owner/namespace/blob, private cleanup, durable retry, final empty scan, and pending-login denial remain green.
- [x] Relevant focused and broad verification completed.
- [x] `05-implementation-plan.md` updated with actual red/green evidence and read back.
- [x] Manual real-OAuth/PDS gates recorded without claiming they ran.
- [x] Implementation review completed; IR-001–IR-004 correction evidence is recorded above and is ready for re-review.

## AppView Audit Exact-Key Safety Correction (Approved 2026-08-14)

### Correction inputs and rules

- Authoritative requirements: `01-requirements.md` §24.
- Authoritative tests: `02-acceptance-tests.md` §12.
- Approved re-review: `03-document-review.md` §7.
- Coding plan: `04-coding-plan.md` §13.
- Product decision: permit minimal non-secret, exact-owner/job/key safety tombstones while PDS/object outcomes remain uncertain; remove them with operation/OAuth state after convergence.
- Preserve the completed no-status, no-recovery-credential, no-manual-Retry, no-index-receipt, no-component-checkpoint, no-audit, no-detailed-metrics, namespace/blob/DID-account, and final artifact-free boundaries.
- Do not implement Go or migration changes from this documentation update; begin the code stage with a meaningful red IT-035.

### Correction test order

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected initial state |
|---|---|---|---|---|
| S1 | IT-035 | FR-028, FR-029, RULE-013 | AC-050 | Current worker can finalize after an initially empty PDS scan and retains no exact-URI key for a delayed commit. |
| S2 | UT-025 | FR-028, FR-029, NFR-007, RULE-012 | AC-049, AC-054 | No minimized tombstone migration/store/type or exact-scope constraint exists. |
| S3 | UT-026 | FR-028, FR-030, NFR-007, RULE-012 | AC-049, AC-054 | Scheduled media does not fully encode immutable upload generation and tested settlement proof. |
| S4 | IT-036 | FR-028, FR-030, RULE-013 | AC-051 | An early object delete/absence can discard tracking before a delayed accepted `Put` materializes. |
| S5 | IT-037 | FR-029, FR-030, RULE-013 | AC-052, AC-054 | Retry exhaustion/elapsed time lacks the amended “remain reconciling” barrier. |
| S6 | IT-038, AT-009 | FR-031, NFR-007 | AC-053 | Finalization does not atomically prove/remove the approved temporary state because that state does not yet exist. |
| S7 | REG-017 and broad regressions | FR-031, NFR-007, RULE-012, RULE-013 | AC-053, AC-054 | The corrected persistence must not revive deleted status/audit/receipt/checkpoint/metrics or broaden deletion authority. |

### Planned red-green-refactor loops

#### S1 — IT-035 delayed PDS commit barrier

- Write failing test: in `appview/internal/accountdeletion/worker_acceptance_test.go`, persist one outcome-uncertain exact registered CraftSky URI, make the fake PDS accept but delay visibility/completion, crash/recreate AppView services, allow the first deletion scan to return empty, then expose the record.
- Run command: `cd appview && go test ./internal/accountdeletion -run 'Test.*Delayed.*PDS' -count=1`.
- Meaningful red: operation finalizes or loses exact-key authority before the delayed record appears.
- Minimum green: adopt a minimized exact-URI tombstone before finalization, keep the operation reconciling, and delete/reverify that URI without repeating its original write.
- Refactor only while green: extract typed settlement evidence/capability scope; do not generalize PDS deletion.

#### S2 — UT-025 migration/store/scope

- Write failing tests: migration shape/check/index/residue assertions plus typed invalid owner/job/follow/namespace/blob/secret cases.
- Run command: `cd appview && go test ./internal/accountdeletion -run 'Test.*Migration|Test.*SafetyTombstone|Test.*PDS.*Scope' -count=1`.
- Minimum green: centrally numbered `account_deletion_safety_tombstones` migration and store compare-and-set/lease API with the exact minimized schema from `04-coding-plan.md` §13.2.
- Refactor only while green: keep PDS/object variants in one relation unless normalization demonstrably improves constraints without widening scope.

#### S3/S4 — UT-026 and IT-036 delayed object `Put`

- Write failing unit test: settlement decision rejects elapsed client time/first absent HEAD and binds immutable key + upload generation.
- Write failing integration test: new `appview/internal/scheduledposts/account_deletion_race_test.go` barriers accepted `Put`, service crash, early delete/absence, delayed materialization, and later cleanup.
- Run commands: focused scheduled-post tombstone/media tests, then the new race test with PostgreSQL and MinIO-compatible adapter.
- Minimum green: persist attempt/key/generation before `Put`, hold owner/object fence through ready CAS while alive, adopt deletion tombstone, and repeatedly delete/verify that exact generation after crash.
- Refactor only while green: share lease/settlement types without merging ordinary object cleanup into a broad account-deletion capability.

#### S5 — IT-037 honest indefinite reconciliation

- Write failing test: no PDS/object finite settlement guarantee, client deadlines/retry windows elapsed, adapters currently absent.
- Meaningful red: operation/tombstone is finalized, discarded, or moved to a terminal-success state.
- Minimum green: bounded leased/backoff reconciliation remains active; client account stays removed; ordinary login/member authority stays denied; no user status route appears.

#### S6 — IT-038/AT-009 final minimization

- Write failing barriers around settled CAS, worker crash, retry, and finalization.
- Minimum green: after revalidation of every exact key, one owner/job-fenced transaction deletes all temporary tombstones, the operation, and matching deletion-only OAuth generation. Duplicate retry is idempotent and no deletion-specific residue remains.

#### S7 — REG-017 and full gate

- Re-run removed-route/no-status Flutter tests, follow/namespace/blob/owner/job/generation capability tests, no-audit/no-detailed-metrics residue searches, migration up/down/up, focused PostgreSQL/MinIO/race suites, full Go and Flutter suites, formatting/static analysis, and `git diff --check`.
- Record only commands that actually run; keep MAN-003 and real remote settlement checks as explicit unrun release gates unless performed with disposable infrastructure.

### Correction execution log

| Test IDs | Red evidence | Green evidence | Status |
|---|---|---|---|
| IT-035 | An accepted ordinary CraftSky PDS write could lose its response, escape the first deletion scan, and commit after operation/OAuth removal. | Durable owner-effect attempts persist exact CraftSky URI/action/fingerprint/CID provenance before dispatch. Departure/deletion adopts unresolved registered-namespace puts into exact-URI safety work; read-only authoritative reconciliation proves the exact current version before projection/finalization, never repeats the write, and keeps ambiguous A→B→A candidates hidden/retryable. Required-PostgreSQL crash/departure/rejoin/proved-not-accepted/terminal tests pass. | Complete |
| UT-025 | No minimized typed store represented exact URI/object uncertainty while rejecting wider deletion authority. | Migrations `000039`, `000041`, `000049`, and `000050` plus store/migration tests enforce exact owner, generation, operation, action, URI/object key, fingerprint/CID, bounded lease, and non-secret scope. Registered CraftSky collection filtering rejects follow, blob, DID, foreign namespace, malformed URI, and cross-owner adoption. | Complete |
| UT-026, IT-036 | Migration/key/settlement tests and the accepted-`Put` crash barrier failed before durable attempts, generation keys, final-absence proof, and owner/object fencing existed. | Scheduled-object PostgreSQL tests, migration `000040`/`000041` up/down/up, and the PostgreSQL+MinIO race suite pass. The barrier retains tracking across early absence and removes delayed materialization on a repeated exact-key cleanup. | Object sub-lane complete |
| IT-037 | The settlement decision initially allowed no durable representation of an unbounded uncertain result; cleanup remote work also outlived its lease. | Object and PDS uncertainty remain leased/backoff reconciliation work indefinitely when no finite backend guarantee exists. Read-only PDS reconciliation and exact-key object cleanup are deadline/lease bounded; neither elapsed client time nor an early absent read fabricates success. | Complete |
| IT-038, AT-009 | Operation completion previously lacked one owner/job-fenced final residue transaction. | Account-deletion lifecycle tests prove only the accepted deletion-only credential survives acceptance; completion requires all exact safety work settled, then atomically transitions the owner terminal, queues credential revocation, and removes operation-scoped authority/residue without restoring a status/manual-retry surface. | Complete |
| REG-017 / broad verification | The amended behavior initially had no aggregate gate. | Required-PostgreSQL/MinIO normal and race suites, migration up/down/up, capability/residue regressions, `go vet`, pinned Staticcheck, vulnerability scanning, `dart analyze`, and all 1,489 Flutter tests pass in the combined audit gate. | Complete |

#### Scheduled-object sub-lane execution (AV-006)

- This sub-lane owns only `UT-026`, `IT-036`, the object half of `IT-037`, and
  the schema needed by the account-deletion lane. `IT-035`, PDS reconciliation,
  operation/OAuth finalization, and the client contract remain outside it.
- Test order is migration shape -> settlement decision -> delayed accepted
  `Put` barrier -> no-finite-bound retention -> focused PostgreSQL/MinIO/race
  regressions.
- Verification on 2026-08-14: database-required `go test
  ./internal/scheduledposts -count=1`; database-required focused migration
  up/down/up; and database-required, MinIO-configured `go test -race
  ./internal/scheduledposts -count=1` pass. `go vet
  ./internal/scheduledposts` passes. The installed Staticcheck binary is built
  with Go 1.25 and cannot analyze the Go 1.26 module, so Staticcheck is not
  claimed.
- No real remote settlement-bound validation ran. The dependency wiring uses
  a zero bound, so unknown object outcomes remain exact-key, lease-claimable
  reconciliation work rather than reporting data-free completion.

### Correction completion checklist

- [x] IT-035 proves accepted-write → AppView crash → empty scan → delayed PDS commit convergence.
- [x] IT-036 proves accepted `Put` → AppView crash → early absent/delete → delayed object creation convergence.
- [x] No finite-settlement object configuration leaves tombstones reconciling rather than fabricating success; PDS coverage remains separate.
- [x] Scheduled-object tombstone fields and typed APIs are exact-owner/job/key/generation and non-secret; PDS store coverage remains separate.
- [x] Only registered `social.craftsky.*` records and matching private object generations can be reconciled; follow/blob/DID/other namespace/owner remain impossible.
- [x] Proven convergence atomically removes tombstones, operation, and matching deletion-only OAuth authority.
- [x] No status/recovery/manual-Retry/receipt/component-checkpoint/audit/detailed-metrics surface is restored.
- [x] Focused PostgreSQL/MinIO/race, full Go/Flutter, migration, formatting/static-analysis, and diff checks are recorded truthfully.
- [x] `06-implementation-review.md` is re-reviewed and approved after implementation.
