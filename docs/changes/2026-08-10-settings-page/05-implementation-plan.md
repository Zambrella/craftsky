# TDD Implementation Plan: Lean Account Deletion Simplification

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
