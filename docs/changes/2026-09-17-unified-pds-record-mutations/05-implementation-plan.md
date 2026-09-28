# TDD Implementation Plan: Unified Tap-Authoritative PDS Record Mutations

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`, high risk, no blockers)
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before its implementation.
- Run the smallest relevant test first and confirm that it fails for the missing behavior.
- Refactor only while the focused and nearby tests are green.
- Keep source projection independent from command history.
- Preserve lifecycle, authorization, credential, private-data, and permanent-deletion boundaries.
- Do not remove a legacy path until its replacement slice and no-caller conformance tests pass.
- Keep this file updated after every completed, blocked, skipped, or reordered test loop.

## Starting Baseline
- The worktree currently ends at migration 72; migration 73 does not exist.
- `tap_source_records` retains raw source evidence but still carries effect-origin projection fields and has no durable versioned semantic-validation status.
- `owner_effect_attempts` still accepts `pds_record` effects and combines dispatch state with projection-origin behavior.
- `pds_set_sources`, `pds_set_aggregates`, and all four `pds_command_*` tables do not exist.
- Follow and interaction projectors still implement winner/collapse semantics; blocks retain duplicate rows but have no common normalized fact/aggregate layer.
- The Flutter app has feature-specific optimistic mutation implementations and no shared PDS mutation controller or HTTP response classifier.
- Existing local document changes predate implementation and must not be reverted or folded into unrelated edits.

## Test Order

The order mirrors the approved phases in `04-coding-plan.md`. Umbrella acceptance and regression IDs are verified with their owning loops and again at the final gate.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-013 | FR-036, FR-044, FR-046 | AC-040, AC-052, AC-054 | Fails: migration 73 and final tables are absent |
| 2 | UT-015 | FR-044 | AC-052 | Fails: persistence boundaries are absent |
| 3 | UT-001 | BR-004, FR-001, FR-002, NFR-002, NFR-006, RULE-006 | AC-005, AC-006 | Fails until source-version selection is isolated and conflict-safe |
| 4 | UT-002 | BR-001, FR-003-FR-005, FR-009, FR-011, FR-029, NFR-002, NFR-004, RULE-001, RULE-003, RULE-006 | AC-002, AC-007-AC-009, AC-015, AC-034 | Fails: no shared authoritative validation registry |
| 5 | UT-003 | FR-008, FR-009, NFR-001, RULE-003 | AC-012, AC-013, AC-041 | Fails: no generic set aggregate reducer |
| 6 | UT-004 | FR-010, FR-042 | AC-014, AC-050 | Fails: notifications are source-driven rather than aggregate-edge-driven |
| 7 | UT-005 | FR-003, FR-009, FR-011, NFR-002, RULE-003, RULE-006 | AC-015 | Fails: blocked normalized facts and wake-up are absent |
| 8 | UT-017 | FR-046 | AC-054 | Fails: source/fact/aggregate boundaries are absent |
| 9 | IT-001 | BR-001, BR-004, FR-001-FR-005, FR-019, FR-029, NFR-002, NFR-004, NFR-006, RULE-001, RULE-002, RULE-005, RULE-006 | AC-001, AC-002, AC-005-AC-009, AC-023, AC-034 | Fails until validation and fact lifecycle are durable |
| 10 | IT-003 | FR-008, NFR-001 | AC-012, AC-041 | Fails until projection transaction boundaries are unified |
| 11 | IT-004 | FR-003, FR-008, FR-009, FR-011, NFR-001, NFR-002, RULE-003, RULE-006 | AC-015, AC-041 | Fails until dependency races converge through facts |
| 12 | IT-005 | BR-004, FR-001, FR-002, FR-006, FR-037, NFR-002, NFR-006, RULE-006 | AC-005, AC-006, AC-010, AC-045 | Fails until verified repair rebuilds without commands |
| 13 | UT-006 | FR-013, FR-014, FR-045, RULE-007 | AC-017, AC-018, AC-053 | Fails: scoped command identity/fingerprints are absent |
| 14 | UT-007 | FR-015, FR-016, FR-019, FR-038, FR-044, RULE-007 | AC-019, AC-020, AC-024, AC-046, AC-052 | Fails: command state machine is absent |
| 15 | UT-016 | FR-045 | AC-053 | Fails: no golden command fingerprint encoding |
| 16 | IT-006 | BR-002, FR-012-FR-016, FR-038, FR-044, FR-045, RULE-007 | AC-003, AC-016-AC-020, AC-046, AC-052, AC-053 | Fails: command store and crash recovery are absent |
| 17 | IT-015 | FR-021-FR-023, FR-047 | AC-026-AC-028, AC-055 | Fails: repository-head and applyWrites transport is absent |
| 18 | UT-008 | FR-021-FR-023, FR-041, FR-047 | AC-026-AC-028, AC-049, AC-055 | Fails: authoritative reader and bounded set planner are absent |
| 19 | IT-008 | BR-001, FR-003, FR-022-FR-025, FR-029, FR-041, FR-047, RULE-003, RULE-008 | AC-002, AC-027-AC-030, AC-049, AC-055 | Fails: guarded set commands are absent |
| 20 | UT-009 | FR-016, FR-018, FR-043 | AC-020, AC-022, AC-051 | Fails: shared command response mapper is absent |
| 21 | UT-012 | FR-028, NFR-003, NFR-005 | AC-042, AC-044 | Fails until command telemetry/redaction exists |
| 22 | IT-011 | BR-001, FR-005, FR-015, FR-025, FR-028, FR-029, FR-035, FR-038, NFR-003, NFR-005, RULE-001, RULE-008 | AC-030, AC-033, AC-034, AC-039, AC-042, AC-044, AC-046 | Fails until command lifecycle and purge wiring exists |
| 23 | UT-010 | BR-003, FR-030, FR-032, FR-033, FR-039, FR-040 | AC-004, AC-036, AC-037, AC-047, AC-048 | Fails: shared Flutter operation reducer is absent |
| 24 | UT-014 | FR-043 | AC-051 | Fails: shared bounded retry scheduler is absent |
| 25 | UT-011 | BR-003, FR-018, FR-030, FR-031, FR-039, FR-048 | AC-035, AC-047, AC-056 | Fails: accepted overlay timing is absent |
| 26 | UT-018 | FR-048 | AC-056 | Fails: mutation-class reconciliation predicates are absent |
| 27 | IT-012 | BR-003, FR-018, FR-030-FR-033, FR-039, FR-040, FR-048 | AC-004, AC-035-AC-037, AC-047, AC-048, AC-056 | Fails: controller/account invalidator integration is absent |
| 28 | UT-013 | BR-003, FR-013, FR-020, FR-021, FR-026, FR-027, FR-030, NFR-004, RULE-007 | AC-004, AC-017, AC-025, AC-026, AC-031, AC-032, AC-043 | Fails: declarative mutation matrices are absent |
| 29 | IT-002 | FR-005, FR-007-FR-010, FR-016, FR-019, FR-024, FR-034, FR-042, FR-046, RULE-001, RULE-003-RULE-005 | AC-011-AC-014, AC-023, AC-024, AC-029, AC-038, AC-050, AC-054 | Fails: like/unlike still uses winner semantics |
| 30 | IT-007 | BR-002, FR-005, FR-012, FR-013, FR-016-FR-020, FR-043, RULE-005 | AC-003, AC-020-AC-025, AC-051 | Fails: like endpoint does not use command recovery |
| 31 | IT-016 | BR-003, FR-013, FR-016, FR-018, FR-030-FR-032, FR-043, FR-048, NFR-004, RULE-007 | AC-017, AC-020, AC-022, AC-035, AC-036, AC-043, AC-051, AC-056 | Fails: real Flutter clients/providers do not delegate |
| 32 | Phase 6 row tests | FR-003, FR-005, FR-007-FR-010, FR-013, FR-016, FR-022-FR-025, FR-028-FR-034, FR-039, FR-041-FR-048 | AC-002, AC-004, AC-007, AC-011-AC-015, AC-017, AC-020, AC-027-AC-030, AC-033-AC-038, AC-047-AC-056 | Fails for repost, then follow, block, and suggestion follow until each row migrates |
| 33 | Phase 7 post tests | FR-012-FR-020, FR-028, FR-030-FR-034, FR-039, FR-043-FR-045, FR-048 | AC-003, AC-004, AC-016-AC-025, AC-033, AC-035-AC-038, AC-047, AC-048, AC-051-AC-053, AC-056 | Fails until post create/delete migrate |
| 34 | Phase 8 business tests | FR-013, FR-016, FR-018, FR-020, FR-027, FR-028, FR-030-FR-034, FR-039, FR-043-FR-045, FR-048 | AC-004, AC-017, AC-020, AC-022, AC-025, AC-032, AC-033, AC-035-AC-038, AC-047, AC-048, AC-051-AC-053, AC-056 | Fails until business records migrate |
| 35 | IT-009 | FR-020, FR-021, FR-027 | AC-025, AC-026, AC-032 | Fails: personal profile writes remain separate |
| 36 | IT-010 | FR-026 | AC-031 | Fails: scheduled final publication still uses old effect path |
| 37 | IT-014 | Matrix-linked requirements | AC-004, AC-017, AC-020, AC-022, AC-025, AC-026, AC-031, AC-032, AC-043 | Fails while any mutation row is pending |
| 38 | UT-019 | FR-049 | AC-057 | Fails while legacy callers remain |
| 39 | IT-017 | FR-049 | AC-057 | Fails while final architecture is incomplete |
| 40 | REG-001-REG-009 | Linked requirements in `02-acceptance-tests.md` | AC-005, AC-006, AC-022, AC-025, AC-030, AC-031, AC-033, AC-034, AC-038, AC-039, AC-057 | Existing/new regressions initially expose superseded behavior |
| 41 | AT-001-AT-014 | Linked requirements in `02-acceptance-tests.md` | AC-001-AC-057 | Umbrella conformance is incomplete until all phases pass |

## Implementation Steps

### Phase 1: Final Schema Baseline
- IT-013: Add the first failing real-PostgreSQL migration test, starting with migration/table existence and expanding one assertion at a time to exact ownership, constraints, indexes, and forbidden legacy paths.
- Run command: `TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL=<worktree-dev-db> go test ./internal/db -run '^TestUnifiedPDSRecordMutationsMigration' -count=1`.
- Implement: migration 73 up/down and required owner-purge inventory entries.
- UT-015: Add retention-boundary schema/store tests only after IT-013 is green.
- Gate: focused migration and purge tests plus down-to-zero/reapply evidence.

### Phase 2: Source Validation, Facts, Aggregates, And Rebuild
- Work in the order UT-001, UT-002, UT-003, UT-004, UT-005, UT-017, IT-001, IT-003, IT-004, IT-005.
- Keep non-like production registrations and serving-query conversions deferred.
- Extract reusable pure reducers before adding transactional stores.
- Gate: incremental and verified-snapshot rebuilds match with empty command tables.

### Phase 3: Command Journal And Protocol Transport
- Work in the order UT-006, UT-007, UT-016, IT-006, IT-015, UT-008, IT-008, UT-009, UT-012, IT-011.
- Persist exact command, plan-version, dispatch-attempt, and tombstone boundaries before remote calls.
- Reuse existing lifecycle fencing and verified-snapshot primitives without extending `owner_effect_attempts` into the new public command model.
- Gate: crash recovery, wire contracts, bounded authoritative reads, ambiguity, retention, redaction, and purge pass.

### Phase 4: Flutter Shared Controller Foundation
- Work in the order UT-010, UT-014, UT-011, UT-018, IT-012, UT-013.
- Use injected time, scheduling, jitter, and UUID sources.
- Reset shared mutation state before feature reads at every account boundary.
- Gate: no feature is migrated yet, but operation, retry, overlay, account reset, and pending matrix behavior pass.

### Phase 5: Like/Unlike Reference Slice
- Work one failing behavior at a time through IT-002, IT-007, IT-016 and the existing interaction/read/notification/push/provider/API-client suites.
- Convert all like serving consumers together; command history remains absent from reads.
- Gate: duplicate external sources, aggregate notification edges, guarded removal, ambiguity, overlay overlap, and account switching pass.

### Phase 6: Remaining Set Actions
- Migrate and gate repost, then follow, then block, then Instagram suggestion follow acceptance.
- Keep private suggestion reservation and excluded mute behavior outside shared public-command state.
- Record each row's local tests and matrix status before starting the next row.

### Phase 7: Immediate Posts
- Migrate append create before addressed delete.
- Preserve blob/video boundaries and empty successful DELETE `204`.

### Phase 8: Business Records
- Migrate business event append/addressed commands, then fixed-key business profile commands.
- Keep private account-type/customisation workflows excluded.

### Phase 9: Personal Profile Compound Command
- IT-009 must fail against the current partial-write implementation before the endpoint is changed.
- Use one ordered two-write guarded transaction; never sequential fallback.

### Phase 10: Scheduled Final Publication
- IT-010 must fail before replacing only the final public create with the already-fenced command executor.
- Preserve scheduling, lease, frozen payload, media, blob-effect, and recovery state.

### Phase 11: Final Wiring, Legacy Removal, And Release Evidence
- Complete IT-014 before removing superseded callers.
- Add and pass UT-019, IT-017, REG-009, AT-013, and AT-014.
- Run all regressions and release gates, then recommended Flutter analysis separately.

## Execution Log

### Contract And Baseline
- Confirmed document review: `Approved`; no blocking requirements, test-design, review, or coding-plan gaps.
- Confirmed implementation authorization: the user explicitly invoked the implementation stage for this high-risk workflow.
- Confirmed first test/order: coding plan Phase 1 controls; IT-013 is first.
- Confirmed no existing `05-implementation-plan.md` or prior implementation state.
- Confirmed implementation baseline through repository exploration; no production code was edited during inspection.

### IT-013: Migration Pair
- Write failing test: `TestUnifiedPDSRecordMutationsMigrationFilesExist` requires the exact migration 73 up/down pair.
- Run command: `cd appview && go test ./internal/db -run '^TestUnifiedPDSRecordMutationsMigrationFilesExist$' -count=1`.
- Confirmed failure: both migration files were absent.
- Implement: added the paired migration files with no schema behavior yet.
- Refactor: none; the next IT-013 loop adds one focused schema-baseline assertion.

### IT-013: Ownership Baseline
- Write failing test: `TestUnifiedPDSRecordMutationsMigrationCreatesOwnershipBaseline` requires the two set tables, four command tables, and versioned validation columns on `tap_source_records`.
- Run command: `TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL='postgres://craftsky:dev@localhost:16063/craftsky_dev?sslmode=disable' go test ./internal/db -run '^TestUnifiedPDSRecordMutationsMigrationCreatesOwnershipBaseline$' -count=1`.
- Confirmed failure: all six tables and all four validation columns were absent.
- Implement: added the final-purpose table boundaries, core keys/checks/indexes, and validation columns in migration 73; added reverse-order down migration cleanup.
- Run result: focused real-PostgreSQL test passed without skipping.
- Refactor: removed an incorrect dispatch-attempt-to-step-ordinal foreign key; dispatch attempts retain `plan_version`, while their ordered steps remain keyed independently by `(command_id, plan_version, ordinal)`.
- Added focused red/green loops for migration 73 removing `pds_follow_operations`. The final `owner_effect_attempts` public-record rejection remains a Phase 11 conformance step because later-phase production routes still require the legacy journal during ordered migration.
- Added focused red/green representative-membership coverage. `pds_set_aggregates` now uses a deferred composite foreign key to the exact normalized fact so fact/aggregate replacement can remain atomic.
- Added focused red/green bounds for ineligibility reasons and dependency keys.
- Added exact source/fact/aggregate ownership, required constraint/index, and migration-73 down/up assertions.
- Added all new DID roles to the terminal purge inventory and bounded cascade drains for command children and normalized source facts.
- Verification passed against PostgreSQL 16: `go test ./internal/db -count=1` and `go test ./internal/ownerlifecycle -count=1`, both with required real-database coverage and no skips.
- `just appview-check` was attempted but is currently blocked outside the repository by the host Xcode/LLVM installation: the active toolchain cannot find `MacOSX27.sdk` and fails to link `libresolv`. The focused migration and lifecycle suites above remain green; release-equivalent gate evidence is still pending.
- IT-013 complete. Phase 1 remains active for UT-015.

### UT-015: Retention Persistence Boundaries
- Added a focused red/green predicate proving only accepted/rejected commands at or after their replay deadline may compact; prepared, dispatching, and ambiguous commands remain intact regardless of age.
- Added a focused red/green minimal tombstone model containing only owner DID, operation kind, scoped key hash, and compaction timestamp.
- Added a focused red/green exact 24-hour replay-deadline contract.
- Added exact PostgreSQL column ownership assertions for commands, ordered steps, dispatch attempts, and the four-column tombstone.
- Verification passed: `go test ./internal/pdscommands -count=1` and the focused real-PostgreSQL migration suite.
- UT-015 and the focused Phase 1 gate are complete. Full `appview-check` remains pending on the documented host toolchain repair.

### UT-001: Source Version Selection
- Added a focused red/green pure selector for first, newer, stale, duplicate, delete-duplicate, and equal-revision conflict versions.
- Refactored the existing locked PostgreSQL source-selection path to delegate to the pure selector without changing persistence behavior.
- Verification passed with `CGO_ENABLED=0`: focused UT-001 and the full `internal/ingestion` package with required PostgreSQL coverage.
- Native linking remains blocked by the documented host Xcode/LLVM SDK issue; disabling CGO avoids that external linker for these pure-Go tests.
- UT-001 complete.

### UT-002: Profile Validation Registry Slice
- Repository inventory found nine production indexed collections and confirmed that validation is currently split among Tap decoding, ingestion, dispatcher switches, projectors, and SQL dependency checks.
- Added a focused red/green validator boundary for `social.craftsky.actor.profile` that applies fixed `self` key policy to create, update, and delete before dispatcher mutation.
- Added focused red/green authoritative Lexicon checks for the profile's ten-craft and 50-character limits by extending the existing embedded Indigo validation catalog.
- Verification passed with `CGO_ENABLED=0`: focused source-validator tests and the full `internal/index` and `internal/ingestion` packages with required PostgreSQL coverage.
- Expanded the shared registry to all nine production collections and every create/update/delete key policy. Non-profile validation is no longer reported as pending.
- Added generated-type and collection-semantic validation for posts, likes, reposts, business records, Bluesky profiles, follows, and blocks, including required fields, identifier/timestamp/language checks, and authoritative Craftsky-post/Bluesky-profile string limits.
- Moved the registry to dependency-neutral `internal/sourcevalidation` so ingestion and projection use the same decision.
- Profile ingestion now validates before membership transition. Invalid profile source evidence is retained without activating or departing the owner.
- Winning source versions now persist validation version, structural status, semantic status, and bounded reason; `SourceRecord` reads expose those fields.
- Updated focused integration fixtures to use Lexicon-valid fixed/TID keys and required post fields. Full `internal/index` and `internal/ingestion` suites pass against PostgreSQL with `CGO_ENABLED=0`.
- UT-002 remains in progress only for the normalized fact/dependency/lifecycle matrix completed by UT-003 through UT-005 and IT-001/IT-004.

### UT-003 / UT-004 / UT-005: Generic Set Primitives
- Added pure canonical old/new scope reduction with deduplication and deterministic lock order.
- Added deterministic representative selection by `(activity_at, source_uri)`, stable activation time during representative churn, and exact activated/deactivated edge classification.
- Added valid-but-ineligible fact classification with bounded reason/dependency state and wake-up to eligible without revalidating or replaying a Tap frame.
- Added real-PostgreSQL aggregate replacement coverage: two physical facts produce one logical aggregate; deleting the representative preserves activation and selects the next source; deleting the last eligible fact deactivates the aggregate.
- Focused UT-003, UT-004, UT-005, and aggregate integration tests pass with `CGO_ENABLED=0`.
- Added latest-version integration coverage: valid set facts are removed by a newer invalid version without historical fallback and restored only by a later valid version.
- Added reverse dependency-order coverage: a valid like persists as an ineligible `missing_subject` fact, contributes no aggregate, and becomes eligible from retained source state after the post appears.

### UT-006 / UT-007 / UT-016 / IT-006: Command Identity Foundation
- Added centralized canonical JSON plus exact versioned request and dispatch-plan SHA-256 domains.
- Added owner/operation-kind/UUID scoped identity, canonical-equivalence and changed-intent tests, and committed request/dispatch golden digests.
- Added explicit legal command transitions with immutable accepted/rejected terminal states; existing 24-hour retention and minimal tombstone tests remain green.
- Added the PostgreSQL command store preparation boundary. Eight concurrent callers converge on one command; changed generation or request fingerprint conflicts before dispatch.
- Added the durable dispatch boundary: immutable ordered steps and a dispatch-attempt row commit with `dispatching` before remote work, ambiguous completion replays safely, and terminal acceptance receives an exact 24-hour replay deadline and cannot be rewritten.
- Focused `internal/pdscommands` unit and PostgreSQL integration tests pass with `CGO_ENABLED=0`.

### IT-015 / UT-008 / IT-008: Authoritative Set Command Foundation
- Added real Indigo wire tests and production transport for `com.atproto.sync.getLatestCommit`, ordered `com.atproto.repo.applyWrites`, explicit unsupported/`InvalidSwap` translation, and single-record delete fallback carrying both `swapCommit` and `swapRecord`.
- Added a narrow `PDSTransport` adapter from the purpose-bound authenticated client to command records without exposing OAuth credentials.
- Added a head-before/head-after complete collection reader that fails closed on changed heads, cursor loops, duplicate or out-of-scope URIs, malformed records, missing CIDs, and configurable page/record/byte limits.
- Added authoritative set-removal planning that filters through local semantic matching, preserves invalid apparent duplicates, sorts deletes by canonical URI, and enforces exact 0/1/2/200/201 write behavior with a 200-write cap.
- Added three-attempt repository-swap handling with exact full-jitter windows of 0-25 ms and 0-50 ms, terminal conflict after attempt three, guarded one-write fallback, and no sequential multi-write fallback.
- Focused `internal/auth` and `internal/pdscommands` suites pass with `CGO_ENABLED=0`.
- Connected bounded authoritative reads to the existing unforgeable `ingestion.VerifiedRepositorySnapshot` abstraction as the signed-CAR fallback. A generated signed-CAR fixture proves the fallback returns the expected collection records without introducing a second verifier.
- Remaining IT-008 work: enforce authority endpoint stability at the command boundary and complete endpoint-level guarded set-command coverage.

### UT-009: Shared Command HTTP Responses
- Added the shared mapper for accepted JSON, empty successful `204`, and exact ambiguous `202 {"status":"ambiguous"}` responses.
- Ambiguous responses clamp integer `Retry-After` to 1-5 seconds and never expose a `Location` status resource.
- Focused `internal/api` response-mapper tests pass with `CGO_ENABLED=0`.
- Added table-driven standard error envelopes for rejection, immutable-key conflict, malformed input, repository conflict, atomic cap, and dispatch unavailability. UT-009 is complete.

### UT-012 / IT-011: Command Telemetry And Lifecycle Safety
- Preserved repository-command/list capabilities through `observability.WrapPDSFactory` inside the existing owner/session-fenced `WithActiveEffects` path.
- Added bounded command-head, authoritative-list, apply-writes, and guarded-fallback metrics. A canary test proves operation keys, record URIs, DIDs, and record content are not emitted as labels.
- Added `remote_deadline` to durable dispatch attempts and exact command plans so account deletion can distinguish active remote work from recoverable uncertainty.
- Added account-deletion adoption/finalization inventory for uncertain `pds_commands`, dispatch attempts, and ordered steps. During ordered migration it also retains the legacy `owner_effect_attempts.effect_kind='pds_record'` inventory so unmigrated routes cannot create an account-deletion safety gap.
- Corrected stale projector, migration, and repository-job fixtures to satisfy the authoritative TID, `$type`, and migration-73 validation-column contracts.
- Focused `internal/auth`, `internal/api`, `internal/app`, `internal/db`, `internal/ingestion`, `internal/observability`, `internal/pdscommands`, and `internal/accountdeletion` suites pass with `CGO_ENABLED=0`.
- A broad `go test ./internal/...` run passed all displayed packages except scheduled-post concurrency timeout flakes; `go test ./internal/scheduledposts -count=1` passed immediately in isolation. Full non-flaky package evidence remains pending.

### UT-010 / UT-014 / UT-011: Flutter Shared Mutation Reducer
- Added focused red tests for immutable same-key ambiguity retry, frozen edits, newest-sequence-wins behavior, account reset, late-result rejection, accepted-overlay reconciliation, two-second grace refresh, 30-second expiry, and bounded automatic retry.
- Confirmed the red state: both focused Flutter tests failed because the shared controller and retry policy were absent.
- Implemented a pure account-lease-scoped reducer with immutable operation tokens, per-scope sequencing, ambiguous/failed states, exact retry-input checks, accepted overlays, explicit reconciliation, and reset epochs.
- Implemented the automatic retry policy with server `Retry-After` clamped to 1-5 seconds, local `[1, 2, 4, 5, 5, 5]` backoff, injected 0-250 ms jitter, six-retry cap, and 30-second elapsed cap.
- Verification passed: `flutter test test/shared/mutations/pds_record_operation_controller_test.dart test/shared/mutations/pds_record_operation_retry_test.dart` and focused Dart analysis report no errors.

### UT-018: Mutation-Class Reconciliation
- Added a focused red test covering append create, addressed update/delete, set active/inactive, fixed-key controlled fields, compound-profile agreement, CID/content alternatives, exact identity, and nested canonical content.
- Confirmed the red state: the proposed shared reconciliation module and all class-specific predicates were absent.
- Added strongly typed ordinary-read projections and reconciliation predicates for append, addressed update/delete, set, fixed-key, and compound-profile operations.
- Append and addressed update require exact URI plus accepted CID or canonical controlled-content agreement; addressed delete requires absence; set actions compare only logical activity; fixed-key comparison ignores uncontrolled fields; compound profile requires both fixed projections.
- Verification passed: all 12 tests under `test/shared/mutations` and focused Dart analysis report no errors.

### IT-012: Scheduling And Account-Boundary Foundation
- Added focused red coverage proving definite acceptance immediately invalidates ordinary reads, schedules exact two-second and 30-second checks, refreshes once at each deadline, and production account invalidation clears ambiguous mutation state.
- Confirmed the red state: the controller had no scheduler/provider and the production account invalidator did not know about shared mutation state.
- Added an injected scheduler, immediate acceptance refresh, scheduled grace/expiry checks, and a production Riverpod controller provider. Timer callbacks remain fenced by controller epoch and current overlay state.
- Wired `accountStateInvalidatorProvider` to reset shared mutation state before invalidating feature reads and repositories.
- Verification passed: all 14 tests under `test/shared/mutations` and focused Dart analysis report no errors.
- IT-012 remains open for migrated-feature late-future and class-specific provider integration; its shared scheduling and production account-boundary foundation is complete.

### UT-013: Declarative Flutter Mutation Matrix
- Added a focused red conformance test for every reviewed included, AppView-only, and excluded mutation row.
- Confirmed the red state: no declarative Flutter matrix existed.
- Added eleven included Flutter rows with required operation-key, identity, CAS, command-step, and shared-controller strategies. All remain explicitly `pending` until their real feature providers migrate.
- Added scheduled final publication as AppView-only and ten private/special workflows as excluded with no shared-controller contract.
- During binding-decision review, added a focused failing retry test and corrected the policy to use the greater of clamped server guidance and local backoff before jitter.
- Verification passed: all 18 tests under `test/shared/mutations` and focused Dart analysis report no errors.
- The Phase 4 shared foundation is complete. IT-012 remains partially open by design for row-local provider integration during feature migration; the next implementation slice is Phase 5 like/unlike.

### IT-002: Logical Like Projection And Readers
- Added a production-path red/green integration test proving two valid physical like records persist as two normalized facts and one logical aggregate with width two.
- Corrected the transactional projector boundary to pass the durable `ingestion.SourceRecord` instead of reconstructing a lossy Tap event. Ordering, validation, and source timestamps now reach normalized projectors unchanged.
- Added production-path churn coverage for `2 -> 1 -> 0`: deleting either physical source preserves the aggregate until the final eligible source disappears.
- Moved normalized fact projection ahead of local subject eligibility checks. A valid like with a missing post now persists as an ineligible source fact, returns the durable missing-subject dependency, and activates from retained source state when the dependency wakes.
- Added a neutral notification set-transition adapter and logical retraction identity. Like notifications activate only on `0 -> 1`, remain unchanged across duplicate width and representative churn, and retract/cancel pending delivery work only on `1 -> 0`.
- The production like projector temporarily dual-writes the legacy serving row with a no-op notification lifecycle. This preserves unmigrated compatibility without allowing the legacy physical-record lifecycle to drive notification edges.
- Migrated ordinary like counts, eligible engagement summaries, viewer-like state, interaction-account lists, and post/project popularity ranking to `pds_set_aggregates`. Counts use one row per logical actor/scope and never sum physical source width. Repost reads remain unchanged for Phase 6.
- Updated test fixtures and query-plan assertions to model logical like aggregates while preserving legacy repost coverage.
- Added one-delivery push coverage: duplicate source creation and representative deletion preserve the original pending delivery; final logical deactivation retracts the notification and cancels that delivery without creating another.
- Verification passed with required PostgreSQL and `CGO_ENABLED=0`: full `internal/auth`, `internal/pdscommands`, `internal/index`, `internal/notifications`, `internal/api`, and `internal/db` packages pass together.
- IT-002 aggregate projection, readers, notifications, and push behavior are complete.

### IT-007: Command-Backed Like Routes
- Extended the coordinated OAuth PDS client with repository-head, atomic apply-writes, and repository-swap delete delegation. Every method remains inside the existing owner/session fence; focused tests cover all nine coordinated client operations.
- Added exact command-result loading for accepted/rejected replay and allowed prepared commands to complete as accepted without remote dispatch for stable no-op set commands.
- Added the durable set-command coordinator: immutable identity is persisted before remote work, repository truth is read at a stable head, create reconciles deterministic records, remove deletes every valid matching record in URI order, and repository-swap conflicts retry at most three times.
- Added guarded one-delete fallback only when atomic writes are unsupported. Lost responses remain ambiguous until authoritative reconciliation; definite client rejection and transport/server ambiguity remain distinct.
- Replaced production like/unlike route authority with command-backed handlers. They require canonical lowercase UUID `Idempotency-Key`, preserve accepted `201`/`200` and empty `204`, return exact ambiguous `202` responses, and replay durable terminal responses exactly.
- Production app/route dependencies now inject `SetCommandService` through the coordinated OAuth client. Combined `internal/auth`, `internal/pdscommands`, `internal/index`, `internal/notifications`, `internal/api`, `internal/db`, `internal/app`, and `internal/routes` tests pass with required PostgreSQL and `CGO_ENABLED=0`.
- IT-007 is complete.

### IT-016: Flutter Like/Unlike Migration
- Added the shared Flutter wire contract for canonical UUID operation keys, exact `202 {"status":"ambiguous"}` parsing, integer `Retry-After` clamping, and strict empty accepted delete `204` validation.
- Like/unlike API and repository methods now require and forward `Idempotency-Key`; accepted response shapes remain unchanged and repost transport remains on the Phase 6 path.
- Migrated `ToggleLikePost` to the account-fenced shared controller. Ambiguous responses retry the immutable method/path with the same operation key, use the shared backoff/jitter policy, stop after six automatic retries or 30 seconds, and do not fabricate a failed write when uncertainty remains.
- Definite acceptance installs a logical set overlay, invalidates ordinary post reads immediately, refreshes at the shared grace/expiry deadlines, and updates live caches. Stale timeline/profile/project/comment/single-post reads apply the overlay; logical agreement retires it.
- Definite failures no longer require speculative rollback because visible state changes only after acceptance. Account invalidation resets controller state and existing generation checks discard late completions.
- Marked the declarative `like/unlike` matrix row migrated; all later included rows remain pending.
- Verification passed for all shared mutation tests, the full post API-client suite, the full interaction-provider suite, and focused Dart analysis with no errors. IT-016 is complete for the Phase 5 like/unlike reference slice.
- Full-app verification exposed that accepted-overlay invalidation could refresh a post thread from stale AppView data before its feature listener patched the section. Added logical like-overlay application across thread roots, comments, and replies; the focused thread regressions now prove stale thread reads remain masked after definite acceptance.
- Updated successful widget-mutation tests to await definite acceptance and explicitly advance the bounded overlay expiry timers.
- Phase 5 gate passed: `just app-test` completed with 2,384 passing tests, and `git diff --check` passed.

### IT-003: Logical Repost Projection And Readers
- Added production-path aggregate coverage proving duplicate physical repost records persist as separate source facts but produce one logical actor/post aggregate.
- Repost notifications now activate only on `0 -> 1`, preserve identity, activity time, and pending delivery state across `2 -> 1` representative churn, and retract only on `1 -> 0`.
- Migrated ordinary repost counts, viewer state, account lists, post and project popularity ranking, active representative lookup, and timeline activity from `craftsky_reposts` to `pds_set_aggregates`.
- Timeline repost items now have stable logical actor/post keys, use the aggregate activation time, and hydrate reason identity from the canonical normalized source and Tap source record. Duplicate source records render one activity and representative churn cannot reorder it.
- Updated realistic query-plan coverage to require indexed access through aggregate, normalized-source, and Tap-source tables instead of the legacy physical repost table.
- Verification passed for the complete `internal/api` package with required PostgreSQL and `CGO_ENABLED=0`.

### IT-008: Command-Backed Repost Routes
- Added command-backed repost/unrepost handlers using the shared authoritative set coordinator. Repost creation preserves share-target and reply restrictions; unrepost removes every semantically valid authoritative match atomically.
- Production repost routes now require canonical lowercase UUID `Idempotency-Key`, preserve accepted `201`/`200` and empty `204`, return exact ambiguous `202` responses, and replay durable terminal responses through the shared command contract.
- Focused handler and route tests pass. Legacy immediate-effect handlers remain temporarily reachable only from their lower-level tests while later cleanup verifies no remaining production caller.

### IT-016: Flutter Repost/Unrepost Migration
- Repost/unrepost API and repository methods now require and forward `Idempotency-Key`, parse the exact ambiguous contract, and strictly validate empty accepted delete `204` responses.
- Migrated `ToggleRepostPost` to the account-fenced shared controller with immutable same-key retries, bounded backoff/jitter, definite-acceptance cache updates, and no speculative visible state before acceptance.
- Added logical repost overlays to single-post, timeline, profile post, project, comment-list, and full thread reads. Stale reads remain masked until logical agreement or bounded expiry.
- Marked the declarative `repost/unrepost` matrix row migrated. Focused shared-mutation, API-client, provider, and Dart analysis gates pass.
- Full-app verification exposed one root-thread repost widget test that did not advance the accepted overlay's bounded timers. The test now advances expiry like the corresponding like test; `just app-test` passes all 2,387 tests.
- Full backend verification exposed two stale repository-repair indexer test doubles after the durable `SourceRecord` dispatcher change; both now implement the production interface.
- Migration 73 temporarily retains valid legacy `pds_record` journal rows and constraints until Phases 7-10 migrate their remaining production callers. Account deletion inventories both legacy and command journals during this transition; Phase 11 must remove the legacy allowance and restore the final IT-013 rejection assertion.
- Phase 6 repost gate passed: `CGO_ENABLED=0 just test` passes every backend package, and the isolated callback-deadline integration test passed ten consecutive runs. Native cgo linking remains blocked only by the documented host Xcode/LLVM SDK issue.

### IT-003: Logical Follow Projection And Readers
- Added production-path aggregate coverage proving duplicate physical follow records persist as separate normalized facts but produce one logical actor/target aggregate.
- Follow notifications now activate only on `0 -> 1`, preserve identity and activity time across representative churn, and retract only on `1 -> 0`. The legacy follow projector remains a notification-free transitional dual write.
- Migrated profile counts, viewer relationship state, follower/following lists, mutuals, timeline eligibility, search/facet ranking, notification actor state, event-time notification policy, push revalidation, Instagram eligibility, and follower-growth snapshots to logical follow aggregates.
- Follow list pagination now uses aggregate activation time plus representative source URI; representative record metadata is hydrated only where the legacy `FindActiveFollow` compatibility reader still requires it.
- Added follow-specific aggregate pagination indexes and changed the follower-count view to logical aggregates; the down migration restores its legacy physical-table definition.

### IT-008: Command-Backed Follow Routes
- Added command-backed follow/unfollow handlers using the shared authoritative set coordinator. Follow preserves target membership and block-direction authorization, uses a fresh TID for new records, and treats an existing external match as a stable no-op.
- Unfollow reads the complete authoritative collection and removes every semantically valid matching follow atomically instead of trusting one projected row or deterministic key.
- Production routes require canonical lowercase UUID `Idempotency-Key`, preserve exact full-profile `200` responses, persist those response bytes for replay, and return the shared ambiguous contract.
- Legacy immediate-effect handlers remain only for lower-level compatibility tests and unmigrated non-route callers.

### IT-016: Flutter Follow/Unfollow Migration
- Follow/unfollow API and repository methods now require and forward `Idempotency-Key`, preserve accepted profile bodies, and parse the exact shared ambiguous response.
- Migrated `ToggleFollowProfile` to the account-fenced shared controller with immutable same-key retries, no speculative visible state before acceptance, sequence fencing, accepted cache publication, and target/viewer/timeline invalidation.
- Added account-scoped logical follow overlays to ordinary profile reads. Stale relationship state remains masked until logical agreement or bounded expiry, and follower counts adjust once without dropping below zero.
- Notification follow actions now delegate to the shared follow provider instead of bypassing operation identity and overlay handling.
- Marked the declarative `follow/unfollow` matrix row migrated. Focused Flutter follow and notification suites pass with clean changed-file analysis.

### IT-003: Logical Block Projection And Readers
- Added production-path aggregate coverage for duplicate block sources and the complete `2 -> 1 -> 0` lifecycle. Both physical sources remain normalized facts, representative churn preserves the logical block, and only the final deletion deactivates it.
- Moved block delivery cancellation to the logical `0 -> 1` activation edge so duplicate physical block records cannot repeatedly cancel delivery work.
- Migrated relationship state and blocked-account pagination, profile/post/search/facet/timeline visibility, notification policy, push revalidation, and business authorization from `atproto_blocks` to `pds_set_aggregates kind='block'`.
- Retained `atproto_blocks` only as a transitional physical projection for legacy compatibility and owner-purge behavior. Added the aggregate-backed block pagination index and realistic query-plan coverage.

### IT-008: Command-Backed Block Routes
- Added command-backed block/unblock handlers using the shared authoritative set coordinator. Block creates one fresh TID only when no authoritative valid match exists; unblock removes every semantically valid authoritative match atomically.
- Production block routes require canonical lowercase UUID `Idempotency-Key`, preserve accepted and replayed response bytes, and return the exact shared ambiguous response contract.
- Definite unblock acceptance enqueues relationship-safety restoration. Mute/unmute remain private excluded mutations and continue through their existing path.

### IT-016: Flutter Block/Unblock Migration
- Block/unblock API and repository methods now require and forward `Idempotency-Key` and strictly parse accepted and ambiguous responses.
- Migrated block/unblock actions in profiles, posts, notifications, and Settings to the account-fenced shared controller with immutable retries, account/sequence fencing, and logical block reconciliation.
- Added an account-scoped logical block overlay to profile, post, notification, timeline, and blocked-list reads. Feed and notification suppression begins only after definite acceptance and remains stable across stale reads until agreement or bounded expiry.
- Marked the declarative `block/unblock` matrix row migrated. Focused Flutter block suites passed, full `CGO_ENABLED=0 just test` passed every backend package, `just app-test` passed all 2,404 Flutter tests, and full-root Dart analysis reported no errors (informational lints remain).

### IT-008: Command-Backed Instagram Suggestion Acceptance
- Kept private suggestion lookup, eligibility, reservation, generation fencing, and completion in the Instagram service while delegating the public follow to the shared authoritative set-command coordinator.
- Suggestion acceptance now requires a canonical lowercase UUID `Idempotency-Key`, captures the authenticated session, derives one stable record key, and persists exact accepted/rejected replay responses. Lost responses return the shared bounded ambiguous contract.
- Existing authoritative follows are accepted as no-ops without creating duplicate records. New follows carry both importer and target lifecycle generations through the command boundary before private completion.
- Runtime wiring uses `SetCommandService`; the legacy immediate PDS record effect is no longer reachable from suggestion acceptance.
- Removed a package-level `instagram -> pdscommands` dependency by translating through a narrow Instagram-owned command result at the app/API adapters. This preserves stored HTTP bytes while keeping the projection domain independent of command infrastructure.

### IT-016: Flutter Instagram Suggestion Acceptance Migration
- Instagram suggestion API and repository methods now require and forward `Idempotency-Key` and strictly parse accepted and ambiguous responses.
- Migrated acceptance to the account-fenced shared follow mutation scope with immutable same-key retries, accepted logical-follow overlay state, ordinary-read invalidation, and bounded reconciliation.
- Pending and ambiguous suggestions remain visible. Definite acceptance removes the suggestion only after the public follow is accepted; account changes and newer operations fence late completions, including stale failures.
- Marked the declarative `instagram suggestion follow acceptance` matrix row migrated.
- Phase 6 gate passed: focused backend command/Instagram/API/app/route suites pass, full `CGO_ENABLED=0 just test` passes every backend package, `just app-test` passes all 2,410 Flutter tests, full-root Dart analysis reports no errors (informational lints remain), and `git diff --check` passes.

### Phase 7: Command-Backed Immediate Post Create/Delete
- Added the append-command coordinator after a focused lost-response test proved append identity had to be selected inside the scoped idempotency lock. One canonical operation key now freezes one TID, creation timestamp, record body, selected URI, and dispatch plan across concurrent requests and replay.
- Immediate post create now requires canonical `Idempotency-Key`, uses the owner/session-fenced repository command boundary, captures one repository head, dispatches one atomic create, reconciles by exact selected URI plus accepted CID or controlled content, and durably replays accepted/rejected HTTP bytes. Blob upload and direct video upload remain outside the command; only the final post-record create is journaled.
- Added the addressed-delete coordinator with exact URI/rkey and expected-CID guards. It rejects stale CIDs without dispatch, uses repository-head CAS, reconciles exact absence after a lost response, preserves empty accepted/replayed `204`, and never retries an uncertain destructive write blindly.
- Wired append and addressed services through production app and route dependencies. The production `POST /v1/posts` and `DELETE /v1/posts/{did}/{rkey}` routes now use command services; legacy handlers remain only for transitional lower-level tests until Phase 11 conformance removal.
- Added endpoint-level lost-response coverage. Create reuses one selected URI with one allocation and one PDS write; delete returns ambiguity after a lost response, then resolves exact absence to empty `204` and replays that result without a second write.
- Full vertical-slice verification exposed that `IndigoPDSClient.GetRecord` could not decode the command transport's generic JSON target. Added a focused red/green adapter test and generic JSON decoding so real coordinated-client post-dispatch verification reaches a definite result instead of false ambiguity.

### Phase 7: Flutter Immediate Post Migration
- Immediate create/delete API and repository methods now require and forward canonical operation keys. Delete additionally sends the accepted record CID in exact `If-Match`, preserves empty successful `204`, and both methods parse the shared bounded ambiguity response.
- Migrated create/delete providers to the account-fenced shared controller with frozen requests, same-key retries, six-retry/30-second bounds, active-account and newest-operation fencing, and no visible mutation before definite acceptance.
- Top-level create and delete install exact-URI record overlays. Accepted creates remain visible across stale timeline/profile/project reads until CID or controlled content agrees; accepted deletes remain absent until exact-URI absence agrees or bounded expiry forces the final refresh.
- Replies continue through the same command/retry boundary but complete without a second record overlay because comment/thread providers already publish the accepted reply directly into their structured caches. This avoids competing feed-oriented invalidation while preserving definite-acceptance ordering.
- The shared controller now owns and cancels its production timers on reconciliation/reset. Successful top-level composer widget tests explicitly advance the required bounded overlay expiry, consistent with the earlier set-mutation test contract.
- Marked `immediate post create` and `immediate post delete` migrated in the declarative matrix and updated its conformance expectation.
- Phase 7 gate passed: focused command, route, API-client, provider, stale-read, comment/reply, and vertical-slice suites pass; `CGO_ENABLED=0 just test` passes every backend package with required PostgreSQL coverage; `just app-test` passes all 2,416 Flutter tests; full-root Dart analysis reports no errors (informational lints remain).

### Phase 8: Business Event Append Create
- Added a focused red handler test proving command-backed business-event creation rejects a missing canonical `Idempotency-Key` before reaching any PDS dependency.
- Added a second focused red test proving one event create freezes the owner/session/generation scope, operation key, validated intent, server-owned `createdAt`, record content, and selected append identity, then returns the accepted event URI/CID as `201`.
- Reused the shared append-command and HTTP response boundaries, including the exact ambiguous `202`, clamped `Retry-After`, and no-`Location` contract; added a row-local endpoint fixture for that response.
- Wired the production business route to `PDSAppendCommands`. The legacy event-create path remains available only when no command executor is supplied so existing lower-level transitional tests remain usable until Phase 11 removal.
- Focused business-event API and business-route tests pass with `CGO_ENABLED=0`. Phase 8 remains active for addressed event update/delete, fixed-key business profile put/delete, Flutter adapters, and the broader gate.

### Phase 8: Business Records Completion
- Added coordinated addressed update/delete handling for business events and fixed-key put/delete handling for business profiles. The handlers freeze validated intent and command identity, reject stale CIDs, preserve unknown record extensions and blob references, reconcile lost responses against exact repository state, return empty `204` deletes, and replay definite results without a second PDS write.
- Wired the production business routes to the append and addressed command executors. Transitional legacy handler fallback remains only for callers without an injected executor and is reserved for Phase 11 removal.
- Updated the Flutter business API and repository boundaries to require canonical operation keys, strictly parse only accepted or bounded ambiguous responses, and preserve the empty successful delete contract.
- Migrated business event create/update/delete, business-profile writes, and personal-profile save to `PdsRecordOperationController`. Each operation reuses one immutable request and operation key across bounded retries, publishes no visible state before definite acceptance, and fences late results by account, session, and superseding operation.
- Added `business_record_overlay.dart` as the shared read-side adapter for optimistic append visibility, addressed update/delete masking, fixed-key profile reconciliation, lease expiry, and stale-read detection. Event lists, event detail, and profile reads now retire overlays only after matching authoritative evidence; unrelated operation scopes do not make those reads stale.
- Removed production business mutation/read dependence on the legacy projection overlay. Its remaining references are its provider declaration and the account-boundary reset required until Phase 11 deletes the superseded mechanism.
- Marked all three Phase 8 matrix rows (`business event create`, `business event update/delete`, and `business profile put/delete`) migrated and updated matrix conformance coverage.
- Focused business/profile verification passed with 234 tests. Full `just app-test` passed all 2,425 Flutter tests, and `just app-analyze` reported no findings.
- Full PostgreSQL-backed `CGO_ENABLED=0 just test` passed every backend package without required-database skips. `CGO_ENABLED=0 just appview-check` then passed the complete release gate, including module and generated-code checks, `gofmt`, `go vet`, Staticcheck, source and binary vulnerability scans, non-race and race tests, production and audit container builds, migration checks, smoke tests, and pinned Tap acceptance.
- Phase 8 is complete. No Lexicon schema change was required; `lexgen-check` passed as part of the release gate.

### Phase 9: Personal Profile Compound Command
- Confirmed the IT-009 red state: the personal-profile endpoint could accept the first sequential fixed-key write before the second write failed, leaving the two profile records inconsistent.
- Added `CompoundCommandService` with one immutable two-record request, one captured repository head, one ordered plan, one guarded `applyWrites`, and no sequential fallback. Unsupported atomic dispatch fails before either record changes.
- Migrated `PUT /v1/profiles/me` command mode to the compound executor with canonical idempotency-key enforcement and shared accepted, rejected, and bounded ambiguous responses.
- Added lost-response reconciliation against both fixed `self` records. A same-key retry returns the stored definite result only after both records agree and never redispatches the transaction.
- Added backend coverage for unsupported atomic writes, no partial changes, exact two-record plan construction, compound lost-response reconciliation, and original-endpoint replay without a second dispatch.
- Updated Flutter profile API and repository boundaries to require and forward one operation key and strictly parse bounded ambiguity. `SaveProfile` and onboarding now reuse one frozen body and UUID key through the shared retry controller and remain fenced by account, session, and superseding operation.
- Added a personal-profile overlay whose reconciliation requires both the Bluesky and Craftsky controlled projections to agree. Profile reads retain the overlay across partial or stale projection updates, compose it with concurrent business-profile overlays, and retire it only after compound agreement or bounded expiry.
- Removed the unused direct `UserProfile` patch methods that could generate operation keys outside the shared controller, and marked `personal profile save` migrated in the declarative matrix. Scheduled final publication remains pending for Phase 10.
- Focused backend and Flutter suites passed, including IT-009, endpoint lost-response replay, keyed profile API behavior, shared-controller retries, two-projection reconciliation, onboarding, edit-profile, navigation, cold-start, and mutation-matrix regressions.
- Phase 9 broader gates passed: `CGO_ENABLED=0 just appview-check`, all 2,430 tests under `just app-test`, and `just app-analyze` with no findings. The backend gate included required PostgreSQL coverage; native CGO remains unavailable because the host Xcode SDK sysroot is missing.
- No Lexicon schema or generated Lexicon type changed. Phase 9 is complete.

### Phase 10: Scheduled Final Publication
- Confirmed the IT-010 red state with `TestIT010ScheduledFinalPublicationRecoversThroughCommandJournal`: after a committed remote create and simulated worker stop, no `scheduled_post_publish` command existed in `pds_commands` because final publication still used `owner_effect_attempts`.
- Replaced `PublicationProcessorOptions.NewEffects` with the combined `GuardedCommandCoordinator` seam. One lifecycle/session fence now spans schedule-effect acquisition, private-media verification and durable blob uploads, final command execution, and publication finalization.
- Added an already-fenced append-command executor that accepts the existing active PDS client and expected-owner scope without recursively entering `WithActiveEffects`. Its scoped wrapper rejects retained use after the guarded callback returns. Command-store transactions reuse the active fenced connection, preserving constrained-pool safety and lifecycle serialization.
- Adapted only the final public post create to operation kind `scheduled_post_publish`. Its deterministic operation key, selected URI/rkey, immutable frozen-record digest, and uploaded blob references are persisted in the common command journal; private scheduling state, leases, frozen payload versions, media staging, cleanup, and blob effect attempts remain unchanged.
- A process stop after final create leaves the command `dispatching`; a returned transport loss records `ambiguous` and schedules retry. Both recovery paths read the frozen URI, confirm exact canonical record equality, accept the existing remote record, and finalize the schedule without a second write. The acceptance tests also prove no legacy `pds_record` effect row is created.
- Updated the lifecycle race fixture to use the migrated schema and real command journal. It passes with a constrained pool, exactly one outer active-effect scope, one PDS write, command state `accepted`, and the competing owner departure serialized until finalization completes.
- Review hardening added callback-scope expiry for retained command executors, exact owner/collection/rkey validation for preselected append identities, and canonical frozen-record comparison after successful dispatch. Focused tests cover post-callback use, malformed selected identities, and replacement of the created record before result readback.
- Focused verification passed for `scheduledposts`, `pdscommands`, `pdseffects`, and `ownerlifecycle`, including the full scheduled recovery and failure suites. Dart workspace analysis reported no errors; Phase 10 introduced no Flutter behavior change.
- `CGO_ENABLED=0 just appview-check` passed the complete backend release gate, including required PostgreSQL tests, non-race and race suites, formatting, vet, Staticcheck, generated-code checks, source and binary vulnerability scans, production/audit container builds, migrations, smoke checks, and the pinned Tap handshake.
- No Lexicon schema or generated Lexicon type changed. Phase 10 is complete.

### Phase 11: Final Wiring, Legacy Removal, And Release Evidence
- Completed IT-014/AT-013 mutation-matrix conformance. Every command-backed row is marked migrated, scheduled final publication remains AppView-only, and the ten reviewed private/special workflows remain explicitly excluded rather than acquiring the public command contract.
- Narrowed production route dependencies so migrated post, scheduled-publication, profile-relationship, and business bundles cannot receive `pdseffects.ExecutorFactory`. Scheduled publication receives the common command coordinator, while blob upload retains only the purpose-specific `BlobEffectFactory` capability.
- Added UT-019/IT-017/REG-009 architecture guards across AppView and Flutter. They reject production effect-origin gating, public relationship mutation methods, set winner replacement, legacy PDS-effect route capabilities, the superseded business projection overlay, and direct migrated-provider cache mutation.
- Removed the obsolete public relationship `Block`/`Unblock` service boundary, production effect-origin reconciliation and lifecycle-source machinery, delete-reconciliation wrapping, the Flutter business projection overlay, and the profile cache-publication helper. Legacy low-level handler constructors remain test-only/transitional definitions with no production route-bundle caller; excluded private account-type/customisation writes retain their existing AppView-private cache publication.
- Finalized migration 73 so new `owner_effect_attempts.effect_kind='pds_record'` rows are rejected while historical rows remain readable for cleanup and migration-down compatibility. The final schema retains blob `object_put`/`object_delete` effects and the common PDS command journal for public record mutations.
- Added regression coverage proving a valid foreign-client record is projected independently of historical effect attempts, and normalized indexer wiring assertions around the source/fact/aggregate boundary.
- Focused conformance passed: `CGO_ENABLED=0 go test ./internal/app ./internal/routes -run 'TestLegacyMutationConformance|TestMigratedMutationRouteBundlesDoNotExposePDSEffects' -count=1` and `flutter test test/architecture/pds_mutation_legacy_conformance_test.dart test/shared/mutations/pds_mutation_matrix_test.dart`.
- MAN-003 passed. The execution log records migration/validation/fact/aggregate/rebuild foundations in Phases 1-2, command journal and authoritative-reader foundations in Phase 3, Flutter controller foundations in Phase 4, then like/unlike, remaining set actions, immediate posts, business records, personal profile, scheduled publication, and final legacy removal in the approved order. No feature slice began before its prerequisite phase.
- MAN-002 and the required release evidence passed: `CGO_ENABLED=0 TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL='postgres://craftsky:dev@localhost:16063/craftsky_dev?sslmode=disable' just test`, `CGO_ENABLED=0 just appview-check`, and `just app-test`. Required PostgreSQL tests ran without skips; `appview-check` completed formatting, vet, Staticcheck, generated-code, vulnerability, race, production/audit image, migration, smoke, and pinned Tap checks.
- Recommended separate evidence passed: `just app-analyze` reported no issues after removing one stale import left by relationship cleanup. This result is not counted as AC-043 release-gate evidence.
- `git diff --check` passed. Final source inventory found no pending matrix row or migrated production caller for a superseded mechanism. Native cgo remains unavailable because the host Xcode SDK sysroot is missing; all required backend evidence used the approved `CGO_ENABLED=0` path.
- No Lexicon schema or generated Lexicon type changed. Phase 11, FR-049, AC-057, AT-014, and the implementation work are complete.

### Implementation Review Correction Order
The implementation review in `06-implementation-review.md` returned `Changes required`. Corrections follow this TDD order:

| Step | Finding | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| 1 | IR-001 | IT-006 | FR-016, FR-023 | AC-016, AC-020, AC-028 | A persisted non-ambiguous dispatch outcome can strand a command in `dispatching`. |
| 2 | IR-002 | IT-011 | FR-038, FR-044 | AC-046, AC-052 | Preparation can recreate command state while terminal purge is in progress. |
| 3 | IR-003 | UT-015 | FR-015, FR-038, FR-044 | AC-019, AC-046, AC-052 | Production never invokes terminal command compaction. |
| 4 | IR-004 | UT-014, IT-016 | FR-030, FR-032, FR-043 | AC-036, AC-051 | Retry exhaustion retains ambiguity without an explicit same-key action and can publish success-like state. |
| 5 | IR-005 | IT-012, IT-016 | FR-031, FR-048, FR-049 | AC-035, AC-056, AC-057 | Quotes bypass accepted overlays and patch feature caches directly. |
| 6 | IR-006 | IT-016 | FR-013 | AC-003, AC-017 | Post create permits an internally generated operation key. |
| 7 | IR-007 | UT-019, IT-017, REG-009 | FR-049 | AC-057 | Flutter no-caller conformance does not inventory migrated UI callers broadly enough. |

Each correction begins with one focused failing test, uses the minimum implementation needed to pass, then runs the affected package/provider suite. Full release evidence is rerun only after all seven findings are green.

### Follow-up Audit Correction Order (2026-09-28)
The subsequent review in `06-implementation-review.md` found IR-008 through IR-012. The user authorized their correction, followed by simplification. This is a new red-green sequence after the earlier phases:

| Step | Finding / improvement | Test ID | Requirements | Acceptance | First failing behavior |
|---|---|---|---|---|
| 1 | IR-008 | IT-016 | BR-002, FR-030, FR-032 | AC-003, AC-021, AC-036 | A dropped response is marked failed and reuses a new key. |
| 2 | IR-009 | IT-001 | FR-004, FR-008 | AC-009, AC-012 | Invalid current source leaves a previous serving fact. |
| 3 | IR-010 | IT-007 | FR-017, FR-019, FR-024 | AC-021, AC-029 | Uncertain set create can be redispatched after external deletion. |
| 4 | IR-011 | IT-007 | FR-016, FR-017, FR-019 | AC-021, AC-024 | Uncertain append/addressed command becomes rejected on external overwrite. |
| 5 | IR-012 | IT-008 | FR-003, FR-005, FR-041 | AC-001, AC-049 | Ingestion-valid external set record is skipped by command matcher. |
| 6 | CAR fallback wiring | IT-008 | FR-047 | AC-055 | Production set reads cannot use verified snapshot fallback. |
| 7 | Legacy removal/shared orchestration | UT-019, IT-017 | FR-049, FR-030 | AC-057, AC-036 | Legacy physical paths and duplicated retry loops remain. |

Follow each step with its focused test and neighboring suite. Keep command history out of projection, do not edit Lexicon shapes, and avoid production database or infrastructure mutations.

- IT-016 / IR-008: Dropped-response `PostApiClient.createPost` test failed with `ApiServerError`; added a keyed-mutation-only API wrapper that classifies unknown transport/5xx/response-format outcomes as ambiguous, and applied it to post, profile, business, and Instagram keyed clients. Kept the definite `video_blob_missing` recovery exception; its red/green regression passes. Focused shared API, post API, and create provider suites pass (75 tests).
- IT-001 / IR-009: Production dispatcher regression failed with one old follow fact and aggregate after an invalid current version. Invalid non-membership sources now project a synthetic serving deletion before quarantine without touching raw PDS evidence; focused PostgreSQL test passes. Membership-profile behavior remains subject to separate lifecycle policy and further full-path verification.
- IT-007 / IR-010: Real-PostgreSQL set-command test showed an ambiguous create being redispatched after an external delete, falsely becoming accepted. Uncertain create attempts now retain ambiguity when the selected URI is absent, overwritten, or replaced by another logical match; exact selected content can still reconcile. Focused old/new command tests pass.
- IT-007 / IR-011: Lost addressed delete plus external recreation initially recorded `rejected`; retry now remains ambiguous. Post-dispatch append-result overwrite also changed from permanent rejection to ambiguity, while pre-dispatch CID/content conflict remains a rejection. Focused regressions pass.
- IT-008 / IR-012: A red route test proved a source-valid `$type`-omitting external follow was ignored by set matching. Command matchers for follow, block, like, and repost now reuse source validation, including rkey policy and optional `$type`, and the updated realistic TID fixtures pass route tests.
- IT-008 / FR-047: A red set-service test proved changing repository heads never invoked configured verified snapshots. Production command wiring now injects the existing signed-CAR fetcher via the authoritative directory and bounded HTTP client; service and existing generated-CAR fallback tests pass.
- IT-007 / FR-023: Crash after a definite `InvalidSwap` stranded set creation in ambiguity. A red PostgreSQL interruption test now passes: the shared journal inspects the last dispatch, permits a new head-bound plan only after definite `InvalidSwap`, and retains the original three-attempt budget. Append, addressed, and compound plans use the same recovery check. Uncertain transport outcomes still never redispatch.
- IT-016 / FR-030, FR-043: Added the shared Flutter `runPdsMutation` retry runner and migrated like, repost, follow, and block providers. It owns the bounded timing/key/epoch decision; each provider only dispatches and presents the result. The provider and shared retry suites pass; remaining feature-specific flows retain their more specialized state handling.
- UT-019 / FR-049: Production's four duplicated coordinated PDS boundary closures were reduced to one purpose-bound factory, and duplicated post-dispatch ambiguity handling to `Store.UnresolvedResult`. Retained legacy physical set tables and their low-level indexer/test/purge paths: earlier migrations and rollback/down migration plus numerous fixture/purge contracts still refer to them. Dropping them here would add a large, potentially destructive migration and test rewrite without simplifying the live read path, which already uses only aggregates. This is a deliberate skipped optional cleanup, not a claimed schema removal.
- Additional branch regression: `just test` exposed `TAP_NO_REPLAY: true` reintroduced in Compose, conflicting with the pre-existing durable-cursor acceptance test. Removed it; focused Tap test and the full `just test` gate pass.
- Gate evidence: final `just app-test` passed all 2,427 Flutter tests including the additional malformed-success regression; `CGO_ENABLED=0 TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL=<local-dev-postgres> just test` passed all packages without DB skips; `CGO_ENABLED=0 just appview-check` passed race, build, migration, health, vulnerability, and Tap checks; `just app-analyze` passed with no issues; `git diff --check` passed.
- A final table-driven fixture confirms that all four set collections accept ingestion-valid external records without `$type` and reject invalid record keys; its focused `internal/api` suite passed after the full backend gate.

### Post-commit legacy-table removal (2026-09-28)
- The user requested a commit before this follow-up. Commit `7de1cae7` contains the mutation correction stage; migration 75 and the legacy removal are committed separately afterward.
- IT-013 / FR-046, FR-049: A real-PostgreSQL migration test first failed because the four physical set tables still existed. Migration `000075_retire_legacy_set_projections` now drops them without CASCADE; its down migration recreates their migration-74 schema so migration 73 and older down paths still work. Rollback recreates empty tables, **not their discarded rows**.
- Removed all production Go references to the four tables. In particular, removed winner-era follow/block/like/repost `Handle` writers, physical follow upsert/delete and block identity readers, and obsolete terminal-purge inventory/cascade SQL. The transactional normalized-source/aggregate projector and its notification/push edge handling remain registered. The dev-only demo seeder now writes Tap source evidence, normalized facts, and aggregate rows, and its reset removes only demo-tagged source identities before recomputing active scopes.
- Replaced active-path legacy test fixtures with source/fact/aggregate fixtures for API reads, pagination, terminal visibility, follower growth, query plans, CLI seeding, and source ingestion. Removed tests exclusive to the deleted winner-era indexers; equivalent duplicate/retarget/notification behavior is exercised by the existing `set_aggregate_integration_test.go` and set-source tests. Historical migration files and tests remain as rollback history. Added a production-source conformance scan to prevent reintroducing physical table callers.
- Verified `CGO_ENABLED=0 TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL=<local-dev-postgres> just test` (no required-DB skips) and `CGO_ENABLED=0 just appview-check` (including race, migration down-to-zero/reapply, builds, vulnerability and Tap gates); both pass. Flutter code was unchanged in this follow-up.

## Completion Checklist
- [x] All Must requirements covered by passing tests or documented gaps
- [x] All planned Must tests passing
- [x] Real-PostgreSQL tests ran without skips
- [x] Relevant regression tests passing
- [x] `just test` passing after review corrections
- [x] `just appview-check` passing after review corrections
- [x] `just app-test` passing after review corrections
- [x] `just app-analyze` recorded separately after review corrections
- [x] No unlinked behavior implemented
- [x] No Lexicon files or generated Lexicon types changed
- [x] No superseded mechanism remains reachable by a migrated caller
- [x] Docs and this execution log updated with correction evidence
- [x] Implementation review completed; subsequently requested legacy-schema removal implemented and verified
