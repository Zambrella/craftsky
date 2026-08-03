# TDD Implementation Plan: Scheduled Posts

**Date started:** 2026-07-31
**Status:** Implementation complete; awaiting implementation review
**Risk:** High
**Implementation approval:** Granted when the user explicitly invoked `implement-tdd` on 2026-07-31
**Review correction approval:** Granted when the user selected `Address required changes` on 2026-07-31
**Second review correction approval:** Granted when the user selected `Address required changes` on 2026-08-01
**Third review correction approval:** Granted when the user selected `Address required changes` for IR-018 through IR-022 on 2026-08-01
**Fifth review correction approval:** Granted when the user selected `Address required changes` for IR-023 through IR-028 on 2026-08-01

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`)
- Coding plan: `04-coding-plan.md` (`Ready for implementation approval`)

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one failing test before its implementation.
- Run the smallest relevant test first and confirm a meaningful red result.
- Refactor only while tests are green.
- Keep traceability and this execution record updated after each loop.
- Use raw parameterized pgx queries; do not introduce sqlc.
- Keep unpublished data owner-private and content-free in telemetry and errors.
- Do not change `lexicon/`, immediate posting defaults, eager PDS upload, or notification behavior.
- Use injected clocks, jitter, and barriers; never wait through product deadlines in tests.
- Do not commit or push unless the user explicitly asks.

## Confirmed Codebase Seams

- `appview/internal/instagram/automatic_follow_store.go` demonstrates raw pgx transactions, leases, retries, and `FOR UPDATE SKIP LOCKED`.
- `appview/internal/instagram/automatic_follow_worker.go` demonstrates HTTP-independent workers with injected time and bounded batches.
- `appview/internal/auth/background_session_selector.go` provides owner-scoped active OAuth-session selection.
- `appview/internal/testdb/testdb.go` provides real-Postgres isolated-schema tests.
- `app/lib/saved_posts/providers/` and `app/lib/auth/providers/account_boundary_provider.dart` provide account-keyed Flutter state patterns.
- Migration `000034` is unoccupied at stage start.

## Test Order

The coding plan overrides the acceptance specification's numeric summary by requiring `UT-002` as the first failing implementation test. Within each later slice, tests follow the dependency order in `04-coding-plan.md`.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 1 | UT-002 | FR-003, RULE-003 | AC-004, AC-025 | Fails: scheduled-post validation package does not exist |
| 2 | UT-001 | FR-002, RULE-002 | AC-002, AC-003 | Fails: eligibility rules absent |
| 3 | UT-004 | FR-006, RULE-001 | AC-006, AC-007, AC-019 | Fails: countable statuses absent |
| 4 | UT-005 | FR-015, RULE-006 | AC-019 | Fails: retry schedule absent |
| 5 | UT-006 | FR-011, FR-015, RULE-005, RULE-006, RULE-008 | AC-018, AC-019, AC-020, AC-029 | Fails: state machine absent |
| 6 | UT-009 | FR-005, FR-009, FR-012 | AC-005, AC-009, AC-020 | Fails: canonical payload absent |
| 7 | UT-010 | FR-012, FR-013 | AC-017 | Fails: publication freeze absent |
| 8 | UT-011 | FR-015, RULE-006 | AC-019, AC-020 | Fails: failure classifier absent |
| 9 | UT-013 | FR-020 | AC-030 | Fails: retention rules absent |
| 10 | UT-015 | NFR-003, NFR-004 | AC-023 | Fails: safe object metadata absent |
| 11 | UT-016 | FR-019 | AC-022 | Fails: scheduled worker absent |
| 12 | UT-018 | FR-018 | AC-018, AC-021 | Fails: scheduled session policy absent |
| 13 | UT-020 | FR-014, FR-020 | AC-030, AC-032 | Fails: tombstone projection absent |
| 14 | UT-003 | FR-003, RULE-003 | AC-004, AC-025 | Fails: Flutter schedule-time model absent |
| 15 | UT-007 | FR-001, FR-009, RULE-008 | AC-001, AC-009, AC-025 | Fails: composer timing state absent |
| 16 | UT-008 | FR-008 | AC-008 | Fails: scheduled row model absent |
| 17 | UT-012 | FR-006 | AC-026 | Fails: capacity UI state absent |
| 18 | UT-014 | FR-008, FR-021 | AC-031, AC-032 | Fails: public status mapping absent |
| 19 | UT-017 | FR-008 | AC-008, AC-027 | Fails: account-keyed list state absent |
| 20 | UT-019 | FR-005, FR-017 | AC-005, AC-012, AC-025 | Fails: staging state absent |
| 21 | IT-026 | FR-006, FR-011, FR-014, FR-020, NFR-001 | AC-006, AC-014, AC-015, AC-017, AC-030 | Fails: migration absent |
| 22 | IT-003 | BR-002, FR-006, RULE-001 | AC-006, AC-007 | Fails: durable capacity store absent |
| 23 | IT-005 | FR-009, RULE-005 | AC-009, AC-028 | Fails: LWW/version fences absent |
| 24 | IT-006 | FR-010, FR-011, RULE-005 | AC-010, AC-014, AC-029 | Fails: mutation/publication serialization absent |
| 25 | IT-007 | FR-011, FR-013, NFR-001, RULE-005 | AC-014, AC-015, AC-017 | Fails: due claims and lease recovery absent |
| 26 | IT-016 | FR-014, FR-020, NFR-004 | AC-030 | Fails: cleanup store absent |
| 27 | IT-017 | FR-014, FR-020 | AC-016, AC-030, AC-032 | Fails: finalization/tombstone lifecycle absent |
| 28 | IT-025 | FR-005, FR-017 | AC-005, AC-030 | Fails: orphan staging lifecycle absent |
| 29 | IT-015 | FR-017, NFR-004 | AC-012, AC-030 | Fails: S3 adapter/MinIO contract absent |
| 30 | IT-014 | FR-017, NFR-004, RULE-004 | AC-011, AC-012 | Fails: private media API absent |
| 31 | IT-024 | BR-004, FR-005, FR-017 | AC-012, AC-013 | Fails: private-copy publication absent |
| 32 | IT-018 | NFR-003 | AC-023 | Fails: end-to-end privacy boundary absent |
| 33 | IT-001 | FR-005, FR-007, RULE-004 | AC-005, AC-011 | Fails: scheduled create API absent |
| 34 | IT-002 | FR-002, FR-003, FR-007, RULE-002, RULE-003 | AC-003, AC-004 | Fails: strict HTTP validation absent |
| 35 | IT-004 | BR-003, FR-007, FR-008, RULE-004 | AC-008, AC-011 | Fails: owner-scoped list/get absent |
| 36 | IT-019 | FR-007 | AC-005, AC-011 | Fails: routes/policies absent |
| 37 | IT-008 | BR-001, FR-011, NFR-001, NFR-002, RULE-007 | AC-015, AC-016 | Fails: due worker behavior absent |
| 38 | IT-009 | BR-004, FR-012, FR-013 | AC-013, AC-017 | Fails: partial-upload recovery absent |
| 39 | IT-010 | FR-013, FR-014, NFR-001 | AC-017 | Fails: ambiguous PDS reconciliation absent |
| 40 | IT-011 | FR-015, RULE-001, RULE-006, RULE-008 | AC-019 | Fails: retry lifecycle absent |
| 41 | IT-012 | FR-012, FR-015, RULE-006 | AC-020 | Fails: current-policy revalidation absent |
| 42 | IT-013 | FR-012, FR-015, FR-018, RULE-008 | AC-018, AC-021 | Fails: account/session lifecycle absent |
| 43 | IT-020 | FR-011, FR-019 | AC-015, AC-022 | Fails: AppView worker wiring absent |
| 44 | IT-023 | FR-021 | AC-031, AC-032 | Fails: notification boundary unproven |
| 45 | IT-027 | NFR-006 | AC-033 | Fails: operational signals absent |
| 46 | AT-009 | BR-001, FR-011, FR-012, FR-014, NFR-001, NFR-002, RULE-007 | AC-013, AC-015, AC-016 | Fails: complete due workflow absent |
| 47 | AT-010 | FR-012, FR-015, FR-018, RULE-001, RULE-006, RULE-008 | AC-018, AC-019, AC-020, AC-021 | Fails: complete failure workflow absent |
| 48 | AT-011 | FR-011, FR-012, FR-013, FR-014, NFR-001, RULE-005 | AC-017 | Fails: complete crash workflow absent |
| 49 | AT-012 | FR-014, FR-020, FR-021, NFR-004 | AC-030, AC-032 | Fails: complete retention workflow absent |
| 50 | AT-013 | BR-004, FR-007, FR-017, NFR-003, NFR-004, RULE-004 | AC-011, AC-012, AC-023 | Fails: complete privacy workflow absent |
| 51 | IT-021 | FR-008 | AC-008, AC-027 | Fails: Flutter account isolation absent |
| 52 | IT-022 | FR-005, FR-016 | AC-005 | Fails: Flutter scheduled submission absent |
| 53 | AT-001 | BR-001, FR-001, FR-004 | AC-001 | Fails: When control absent |
| 54 | AT-002 | FR-002, RULE-002 | AC-002, AC-003 | Fails: composer eligibility UI absent |
| 55 | AT-003 | FR-001, FR-003, RULE-003 | AC-004, AC-025 | Fails: schedule picker absent |
| 56 | AT-004 | BR-004, FR-005, FR-016, FR-017, NFR-004, RULE-004 | AC-005, AC-012, AC-025 | Fails: submit-time staging UI absent |
| 57 | AT-005 | BR-002, FR-006, RULE-001 | AC-006, AC-007, AC-026 | Fails: capacity UI absent |
| 58 | AT-006 | BR-003, FR-007, FR-008, FR-021, RULE-004 | AC-008, AC-011, AC-027, AC-031 | Fails: management UI absent |
| 59 | AT-007 | BR-003, FR-007, FR-009, RULE-005, RULE-008 | AC-009, AC-028 | Fails: scheduled editing UI absent |
| 60 | AT-008 | BR-003, FR-007, FR-010, FR-011, RULE-005 | AC-010, AC-014, AC-029 | Fails: delete/lock UI absent |
| 61 | AT-014 | NFR-005 | AC-024 | Fails: accessibility behavior absent |
| 62 | REG-001 | FR-004 | AC-001 | Existing test should remain green |
| 63 | REG-002 | FR-017 | AC-012 | Existing test should remain green |
| 64 | REG-003 | FR-002, FR-004 | AC-001, AC-003 | Existing test should remain green |
| 65 | REG-004 | FR-004 | AC-001 | Existing test should remain green |
| 66 | REG-005 | FR-014, FR-021 | AC-032 | New regression proof initially absent |
| 67 | REG-006 | FR-021 | AC-031, AC-032 | New regression proof initially absent |
| 68 | REG-007 | BR-003, FR-008 | AC-008 | Existing Settings tests should remain green |
| 69 | REG-008 | FR-004, FR-016 | AC-001, AC-005 | Side-by-side cache proof initially absent |
| 70 | REG-009 | FR-004, FR-017 | AC-001, AC-012 | Existing wire tests should remain green |
| 71 | REG-010 | FR-008, RULE-004 | AC-008, AC-011 | Account-boundary regression initially absent |

## Implementation Steps

### Step 1: UT-002 — Schedule-time validation

- Write failing test: table-driven public validation tests for whole/non-whole-minute values at 4:59, 5:00, 28 days, and 28 days plus one minute using an injected server time.
- Run command: `cd appview && go test ./internal/scheduledposts -run TestValidateScheduledAt`
- Confirmed failure: Build failed because `ValidateScheduledAt` and `ErrInvalidScheduledAt` did not exist.
- Implement: Added `validation.go` with the five-minute and 28-day inclusive limits, whole-minute validation, and the stable domain error.
- Run command after implementation: `cd appview && go test ./internal/scheduledposts -run '^TestValidateScheduledAt$'` and `cd appview && go test ./internal/scheduledposts` both passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: This is the mandatory first red test from the approved coding plan.

### Step 2: UT-001 — Scheduled-post eligibility

- Write failing test: classify original standard, standalone project, quote, and reply/comment shapes.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestScheduleEligibility$'`
- Confirmed failure: After bypassing a sandbox-only Go cache denial, the build failed because the eligibility shape, kinds, validator, and stable error did not exist.
- Implement: Added standard/project domain kinds, the minimal payload shape, and quote/reply/kind rejection.
- Run command after implementation: The focused test and full `./internal/scheduledposts` package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Keep the API boundary and existing immediate-post behavior out of this unit loop.

### Step 3: UT-004 — Countable capacity statuses

- Write failing test: enumerate Scheduled, Publishing, Retrying, Needs attention, published, deleted, and unknown states.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestStatusCountsTowardCapacity$'`
- Confirmed failure: Build failed because the status vocabulary and capacity classifier did not exist.
- Implement: Added the six domain statuses and counted only Scheduled, Publishing, Retrying, and Needs attention.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Database capacity serialization remains IT-003.

### Step 4: UT-005 — Bounded retry plan

- Write failing test: verify due/+1/+3/+7/+15/+30 eligibility with deterministic jitter extremes and a hard +30 cap.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestRetryAttemptAt$'`
- Confirmed failure: Build failed because the retry-attempt function did not exist.
- Implement: Added the six approved offsets, deterministic intermediate jitter, exhausted-attempt signaling, and due/+30 clamps.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Worker state transitions remain later loops.

### Step 5: UT-006 — Lifecycle transitions and mutation locks

- Write failing test: cover valid/invalid status transitions, Publishing mutation locks, automatic claim eligibility, and stale worker versions.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestLifecycleStateRules$'`
- Confirmed failure: Build failed because transition validation, mutation/claim classification, and worker-version fencing did not exist.
- Implement: Added the exact allowed lifecycle edges, member mutation locks, automatic claimable statuses, and stale-version rejection.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Database transaction ordering remains IT-005 through IT-007.

### Step 6: UT-009 — Canonical editable payload

- Write failing test: round-trip complete standard/project payloads with facets, languages, project data, ordered media, alt text, and aspect ratios.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestPayloadRoundTrip$'`
- Confirmed failure: Build failed because the canonical payload/media types and encoding functions did not exist.
- Implement: Added strict canonical JSON encoding/decoding for kind, text, facets, languages, project JSON, and ordered media metadata.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: HTTP request mapping and publication-time PDS assembly remain later loops.

### Step 7: UT-010 — Frozen publication identity and body

- Write failing test: freeze a typed TID/rkey, first-attempt `createdAt`, intended URI, body bytes, and hash, then attempt recovery with different inputs.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestFreezePublicationIdentityAndBody$'`
- Confirmed failure: Build failed because publication-freeze request/result types and logic did not exist.
- Implement: Added typed TID-to-record-key conversion, intended AT URI construction, first-attempt UTC time, body hash/copy, and immutable reuse of an existing freeze.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Current Indigo exposes `syntax.NewTIDFromTime`; the stored TID string is parsed as a typed `syntax.RecordKey` at the boundary.

### Step 8: UT-011 — Publication failure classification

- Write failing test: classify timeout, auth/PDS/object unavailability, invalid payload/media/policy, and record conflict into retry or immediate Needs attention with safe codes.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestClassifyPublicationFailure$'`
- Confirmed failure: Build failed because failure dispositions, safe sentinels, and classification did not exist.
- Implement: Added wrapped-error-aware retry/Needs attention classification with only approved content-free codes.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Attempt timing/state persistence remains worker/store integration work.

### Step 9: UT-013 — Retention deadlines

- Write failing test: calculate unclaimed-media, Needs attention, published tombstone, and immediate-success cleanup eligibility.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestRetentionDeadlines$'`
- Confirmed failure: Build failed because retention deadline functions did not exist.
- Implement: Added exact UTC 24-hour, 30-day, 30-day, and immediate-success deadline functions.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Cleanup querying/deletion remains IT-016/IT-017/IT-025.

### Step 10: UT-015 — Opaque object keys and safe diagnostics

- Write failing test: construct an object key and diagnostic attributes beside DID/handle/filename/alt/time/token canaries and assert only opaque/allowlisted output.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestPrivateMetadataIsOpaqueAndDiagnosticFieldsAreSafe$'`
- Confirmed failure: Build failed because opaque-key and allowlisted-diagnostic helpers did not exist.
- Implement: Added UUID-only scheduled-media keys plus allowlisted operation/result/error-class attributes that reduce unknown values to `unknown`.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Full telemetry scanning remains IT-018.

### Step 11: UT-016 — HTTP-independent bounded worker

- Write failing test: construct a worker from interfaces and injected time, claim a configured bounded batch, and process it without route/request state.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestWorkerProcessesBoundedBatchWithoutHTTP$'`
- Confirmed failure: Build failed because the work item, interfaces, worker options, constructor, and batch method did not exist.
- Implement: Added interface-only claim/process orchestration, injected UTC time, a default batch, and a hard maximum of 100.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Publication semantics and AppView lifecycle wiring remain IT-008 through IT-013 and IT-020.

### Step 12: UT-018 — Active owner publication session

- Write failing test: use an owner-scoped selector fake for another active device, absent last session, and expired/revoked selection outcomes.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestSelectPublicationSession$'`
- Confirmed failure: Build failed because the scheduled publication session adapter did not exist.
- Implement: Added an owner-typed selector interface that reuses `auth.ErrNoUsableBackgroundSession` and maps absence/empty selection to `ErrAuthUnavailable`.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Real database session/account lifecycle behavior remains IT-013.

### Step 13: UT-020 — Content-free publication tombstone

- Write failing test: project a completed schedule containing payload/media canaries into the exact allowed identity/idempotency/URI/CID/timestamp fields.
- Run command: `cd appview && go test ./internal/scheduledposts -run '^TestNewPublicationTombstoneDropsPrivateContent$'`
- Confirmed failure: Build failed because completed-publication/tombstone types and projection did not exist.
- Implement: Added the exact allowed reconciliation fields, UTC published/expiry timestamps, ignored private inputs, and redacted diagnostic formatting.
- Run command after implementation: The focused test and full package passed.
- Refactor: Ran `gofmt`; no further refactor was needed.
- Notes: Persistence and expiry remain IT-017.

### Step 14: UT-003 — Absolute instant and local display

- Write failing test: retain one UTC instant while rendering it with two device timezone/offset views, including a DST/travel change.
- Run command: `cd app && flutter test test/scheduled_posts/schedule_time_test.dart`
- Confirmed failure: Flutter compilation failed because the schedule-time model and `ScheduledInstant` did not exist.
- Implement: Added UTC normalization/wire encoding plus a deterministic current-zone name/offset display view.
- Run command after implementation: The focused Flutter test passed.
- Refactor: Ran `dart format`; no further refactor was needed.
- Notes: Picker widgets and server validation remain AT-003/IT-002.

### Step 15: UT-007 — Composer timing initialization

- Write failing test: initialize a new composer, a future scheduled edit, and a Needs attention edit with a missed time.
- Run command: `cd app && flutter test test/scheduled_posts/schedule_composer_state_test.dart`
- Confirmed failure: Flutter compilation failed because composer timing choice/state did not exist.
- Implement: Added pure factories for new, future-edit, and Needs attention edit timing state.
- Run command after implementation: The focused test and neighboring schedule-time test passed.
- Refactor: Ran `dart format`; no further refactor was needed.
- Notes: Widget wiring remains AT-001/AT-003/AT-007.

### Step 16: UT-008 — Bounded management row mapping

- Write failing test: map long text-only and image/project summaries to first-media/type/title/bounded-preview/time/status rows without full payload exposure.
- Run command: `cd app && flutter test test/scheduled_posts/scheduled_post_row_model_test.dart`
- Confirmed failure: Flutter compilation failed because scheduled summary/status/kind and row presentation models did not exist.
- Implement: Added redacted summary/row models with first-media selection, Unicode-code-point preview bounding, project title, current-zone time view, and status.
- Run command after implementation: The focused test and all neighboring scheduled pure-state tests passed.
- Refactor: Ran `dart format`; no further refactor was needed.
- Notes: Management widgets and authenticated thumbnail fetching remain AT-006.

### Step 17: UT-012 — Full-cap composer actions

- Write failing test: derive schedule/post-now actions for counts zero through three under both timing choices.
- Run command: `cd app && flutter test test/scheduled_posts/schedule_capacity_state_test.dart`
- Confirmed failure: Flutter compilation failed because capacity/action state did not exist.
- Implement: Added pure count validation and derived visibility/enabled/manage/count state while keeping Post now enabled.
- Run command after implementation: The focused test and all neighboring scheduled pure-state tests passed.
- Refactor: Ran `dart format`; no further refactor was needed.
- Notes: Provider refresh and widget management link remain later loops.

### Step 18: UT-014 — Public status vocabulary

- Write failing test: map all internal/wire lifecycle values to exactly four unpublished UI statuses and suppress published/deleted/unknown values.
- Run command: `cd app && flutter test test/scheduled_posts/scheduled_post_status_test.dart`
- Confirmed failure: Flutter compilation failed because wire-to-public status decoding did not exist.
- Implement: Added an exhaustive four-value decoder that returns no management status for published/deleted/internal/unknown values.
- Run command after implementation: The focused test and all neighboring scheduled pure-state tests passed.
- Refactor: Ran `dart format`; no further refactor was needed.
- Notes: Settings count/list widgets remain AT-006.

### Step 19: UT-017 — Account-keyed sort and refresh

- Write failing test: load out-of-order rows, replace/de-duplicate on refresh, and prove Alice/Bob provider-family instances never share cached rows.
- Run command: `cd app && flutter test test/scheduled_posts/scheduled_posts_provider_test.dart`
- Confirmed failure: Flutter compilation failed because the repository interface/family and scheduled-list family did not exist.
- Implement: Added an injectable repository family plus an account-keyed async notifier that sorts by UTC time, de-duplicates by resource ID, and replaces state on explicit refresh.
- Run command after implementation: Generated only the two new Riverpod files; the focused provider test passed.
- Refactor: Ran `dart format`; no further refactor was needed.
- Notes: The production HTTP repository and account-boundary generation are IT-021.

### Step 20: UT-019 — Submit-time private staging state

- Write failing test: retain local bytes and stable operation/media IDs across visible progress, staging failure, create failure, and retry; do nothing for Post now.
- Run command: `cd app && flutter test test/scheduled_posts/private_staging_state_test.dart`
- Confirmed failure: Flutter compilation failed because private staging source/progress/phase/failure state did not exist.
- Implement: Added immutable, redacted staging state that begins only for Schedule, preserves byte objects/operation/media IDs, exposes progress, and resumes at staging or creation as appropriate.
- Run command after implementation: The focused test and all seven scheduled pure-state/provider tests passed.
- Refactor: Ran `dart format`; no further refactor was needed.
- Notes: Media processing/upload and repository orchestration remain IT-022/AT-004.

### Step 21: IT-026 — Scheduled-post migration contract

- Write failing test: apply the migration under `internal/testdb.WithSchema`, inspect all four tables/columns/named constraints/indexes, reject invalid lifecycle/reference rows, and verify down migration preserves unrelated sentinel data.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/db -run '^TestScheduledPostsMigration$'`
- Confirmed failure: The focused test failed because migration `000034_scheduled_posts.up.sql` did not exist. The first database run then exposed that a nullable ordinal passed a CHECK through SQL unknown semantics.
- Implement: Added migration `000034` up/down SQL with private schedule/media/tombstone/cleanup tables, owner/idempotency/lifecycle/reference constraints, and named due/lease/capacity/cleanup indexes; tightened media attachment checks to require an explicit ordinal.
- Run command after implementation: The focused test passed against the repository-managed Postgres container on its checkout-specific published port.
- Refactor: Kept all storage SQL in the migration and used no sqlc artifacts.
- Notes: Rechecked that `000034` remained available; no lexicon or public-post schema changed. The local Postgres service was started without the rest of the stack for this loop.

### Step 22: IT-003 — Transactional three-item capacity

- Write failing test: create two countable rows, race two owner-scoped creates from separate transactions, and verify exactly one final slot commits before proving publish/delete releases capacity.
- Run command: `cd appview && TEST_DATABASE_URL=<checkout Postgres URL> go test ./internal/scheduledposts -run '^TestStoreEnforcesCapacityTransactionally$'`
- Confirmed failure: Build failed because the raw-pgx store, create contract, capacity error, and three-item constant did not exist.
- Implement: Added an owner-profile row lock, count-and-insert transaction, and strict three-active-row rejection using raw parameterized pgx.
- Run command after implementation: The focused concurrent test and full scheduled-post package passed against checkout Postgres.
- Refactor: Moved explicit SQL constants into `store_queries.go` while the focused test remained green.
- Notes: No sqlc or in-memory capacity authority was introduced; deleting the active winner proved the slot becomes immediately reusable.

### Step 23: IT-005 — Last-write-wins edits and worker version fence

- Write failing test: race two owner edits without an expected client version, assert the highest committed internal version owns the stored payload, and reject a worker mutation carrying the pre-edit payload version.
- Run command: `cd appview && TEST_DATABASE_URL=<checkout Postgres URL> go test ./internal/scheduledposts -run '^TestStoreEditsAreLastWriteWinsAndFenceStaleWorkers$'`
- Confirmed failure: Build failed because update results/params, owner update, and frozen-record worker mutation did not exist.
- Implement: Added row-locked last-write-wins edits that increment the private payload version, plus a lease/status/version-fenced frozen-record mutation.
- Run command after implementation: The focused concurrent test and full scheduled-post package passed against checkout Postgres.
- Refactor: Kept all new SQL as explicit raw-query constants and all public transaction orchestration in `store.go`.
- Notes: No payload version is accepted from or exposed to the member edit contract; it remains an internal worker fence.

### Step 24: IT-006 — Serialize edit/delete against Publishing

- Write failing test: exercise update/delete before the Publishing boundary and while a worker holds the matching per-schedule effect lock, verifying commit order decides and no member mutation crosses the external-write boundary.
- Run command: `cd appview && TEST_DATABASE_URL=<checkout Postgres URL> go test ./internal/scheduledposts -run '^TestStoreSerializesMutationsAgainstPublishing$'`
- Confirmed failure: Build failed because the Publishing effect contract, guard, and owner delete mutation did not exist.
- Implement: Added matching transaction/session advisory locks, a dedicated-connection Publishing guard, atomic status/lease/identity transition, and owner-scoped locked delete.
- Run command after implementation: The focused three-order serialization test and full scheduled-post package passed.
- Refactor: Kept advisory-key derivation in one SQL surface and made guard release idempotent.
- Notes: The session lock spans the external-effect window; member mutations acquire the matching transaction lock before inspecting state.

### Step 25: IT-007 — Exclusive due claims and lease recovery

- Write failing test: race two workers over due/future rows, recover exactly at lease expiry, preserve the first publication identity, and reject the old lease's worker write.
- Run command: `cd appview && TEST_DATABASE_URL=<checkout Postgres URL> go test ./internal/scheduledposts -run '^TestStoreClaimsDueWorkWithExclusiveRecoverableLeases$'`
- Confirmed failure: Build failed because the durable due-claim method did not exist.
- Implement: Added expired-lease recovery and bounded `FOR UPDATE SKIP LOCKED` claims that atomically enter Publishing, allocate/reuse TID identity, and issue a new lease token.
- Run command after implementation: The focused concurrent/recovery test and full scheduled-post package passed.
- Refactor: Reused the existing worker lease/version errors and typed Indigo record-key boundary.
- Notes: Future rows remain Scheduled; the old lease is fenced after recovery.

### Step 26: IT-016 — Cleanup lifecycle store

- Write failing test: persist live/shared/unclaimed/replaced/deleted/success/expired fixtures, claim eligible cleanup safely, retry an object-delete failure, and preserve every live reference.
- Run command: `cd appview && TEST_DATABASE_URL=<checkout Postgres URL> go test ./internal/scheduledposts -run '^TestStoreCleansEligiblePrivateArtifactsSafely$'`
- Confirmed failure: The focused test failed to compile because lifecycle sweep,
  cleanup claim, completion, and retry operations did not exist.
- Implement: Added transactional expiry for unclaimed media, Needs attention
  schedules/media, and tombstones; content-free deduplicated cleanup jobs;
  reference cancellation; exclusive recoverable cleanup leases; fenced completion;
  and allowlisted retry metadata.
- Run command after implementation: The focused real-Postgres cleanup lifecycle
  test passed.
- Refactor: Centralized every new raw parameterized query in `store_queries.go`.
- Notes: Object deletion remains outside the database transaction and can retry;
  live object references are never returned as cleanup claims.

### Step 27: IT-017 — Atomic publication finalization and tombstone

- Write failing test: finalize a correctly fenced Publishing claim, insert the
  content-free 30-day tombstone, enqueue its private media, remove active private
  rows, release capacity, and make an identical repeated completion idempotent.
- Run command: `cd appview && TEST_DATABASE_URL=<checkout Postgres URL> go test ./internal/scheduledposts -run '^TestStoreFinalizesPublicationAtomically$'`
- Confirmed failure: The focused test failed to compile because the finalization
  contract and store operation did not exist.
- Implement: Added one owner/status/lease/version-fenced transaction that records
  the content-free result and 30-day expiry, removes active private rows, queues
  immediate media cleanup, and treats an identical tombstone as idempotent while
  rejecting a conflicting result.
- Run command after implementation: The focused finalization and cleanup lifecycle
  tests passed against real Postgres.
- Refactor: Kept result parsing typed as AT URI/CID and all SQL centralized.
- Notes: Finalization deliberately does not reacquire the publication effect lock;
  its caller retains the dedicated guard through commit.

### Step 28: IT-025 — Unclaimed staging after create failure

- Write failing test: fail schedule creation at capacity after private media is
  Ready, release a slot, retry the identical operation to claim one object, and
  expire only the still-unclaimed object after 24 hours.
- Run command: `cd appview && TEST_DATABASE_URL=<checkout Postgres URL> go test ./internal/scheduledposts -run '^TestStoreExpiresOnlyTrulyUnclaimedStagingAfterCreateRetry$'`
- Confirmed failure: The focused test failed to compile because create requests
  could not carry media identities; the store also lacked media claiming and an
  active-operation idempotency lookup.
- Implement: Create now validates up to four distinct media IDs, returns the same
  active resource for an identical owner operation/request hash, rejects changed
  operation reuse, and atomically attaches only same-owner unclaimed Ready media
  after the capacity check.
- Run command after implementation: The orphan-staging test plus neighboring
  capacity and finalization tests passed against real Postgres.
- Refactor: Added a small UUID uniqueness helper and centralized operation/media
  lock and attach queries.
- Notes: Capacity rejection rolls back before attachment; the lifecycle sweep
  later removes only media that remained unclaimed for the full 24 hours.

## Implementation Review Correction Pass

The user selected `Address required changes` after the `Changes required`
verdict in `06-implementation-review.md`. The correction order temporarily
preempts Step 26 because IR-002 and IR-003 affect the claim/effect-lock contract
on which later publisher and cleanup work depends.

### Correction 1: IR-002 — Compose IT-006 and IT-007 claim/effect fencing

- Requirement IDs: FR-011, FR-012, FR-013, RULE-005
- Acceptance Criteria: AC-014, AC-015, AC-017
- Write failing test: claim a due row, acquire its publication effect guard with
  the returned owner/lease/version, prove owner mutation cannot cross the guard,
  and expose the fake external-write boundary only after the recheck succeeds.
- Confirmed failure: The focused test failed to compile because
  `AcquirePublishingEffect` did not exist; the only effect API attempted a second
  Publishing transition and rejected an already-claimed row.
- Implement: Added a dedicated-connection effect acquisition that verifies the
  claimed row's owner, Publishing status, lease token, and payload version.
  Removed the duplicate pre-claim transition API and converted existing IT-006
  scenarios to the single `ClaimDue` then `AcquirePublishingEffect` protocol.
- Verification: The new correction test plus IT-006/IT-007 focused real-Postgres
  tests passed.

### Correction 2: IR-003 — Never pool a session with an uncertain advisory unlock

- Requirement IDs: FR-010, FR-011, FR-012, FR-013, NFR-001, RULE-005
- Acceptance Criteria: AC-014, AC-017
- Write failing test: release a held effect guard with a cancelled context, then
  prove a later caller can acquire the same schedule lock.
- Confirmed failure: Release used the already-cancelled caller context, returned
  `context canceled`, and would have returned the session to the pool without a
  confirmed advisory unlock.
- Implement: Release now uses a bounded non-cancelled cleanup context. A
  confirmed unlock returns the connection normally; an error or false unlock
  hijacks and closes the connection so it cannot re-enter the pool.
- Verification: The cancellation test proved a simultaneously held independent
  connection could immediately acquire the same advisory key; all neighboring
  effect/lease tests passed.

### Correction 3: IR-006 — Collision-safe first-attempt TIDs

- Requirement IDs: FR-012, FR-013, NFR-001
- Acceptance Criteria: AC-016, AC-017
- Write failing test: claim two due schedules for one owner whose UUIDs share the
  low ten bits and assert both receive distinct persisted record keys.
- Confirmed failure: The real-Postgres claim failed with the named unique index
  because both rows received the same timestamp/clock-ID TID.
- Implement: First-attempt allocation now takes a non-blocking transaction-level
  owner lock, probes up to all 1,024 TID clock IDs from the deterministic starting
  point, and excludes active and tombstoned owner keys. A busy owner is left for
  another bounded claim rather than blocking or aborting the batch.
- Verification: The deterministic collision test and neighboring claim/effect
  integration tests passed.

### Correction 4: IR-007 — Recover or terminate an expired sixth attempt

- Requirement IDs: FR-013, FR-014, FR-015, FR-020, RULE-006
- Acceptance Criteria: AC-017, AC-019, AC-030
- Write failing test: expire a sixth-attempt Publishing lease and prove it is not
  stranded in unclaimable Retrying state.
- Confirmed failure: Recovery returned no claim; the row was changed to
  Retrying while the due query excluded its attempt count of six.
- Implement: Ordinary expired attempts still return to Retrying. An expired
  sixth attempt stays Publishing and is preferentially reclaimed with a fresh
  lease, the same attempt count, and the same frozen identity so the worker can
  reconcile the ambiguous effect. The later definite-failure worker path remains
  responsible for entering Needs attention with its bounded retention deadline.
- Verification: The sixth-attempt case and all neighboring exclusive/recoverable
  lease, collision, and claim/effect tests passed.

### Correction 5: IR-004 — Text-only schedules skip media staging

- Requirement IDs: FR-005
- Acceptance Criteria: AC-005, AC-025
- Write failing test: begin Schedule with zero media sources and expect the
  creation phase immediately while Post now remains idle.
- Confirmed failure: A zero-source Schedule entered Staging even though no media
  event could advance it.
- Implement: Schedule now enters Creating immediately when the retained media
  source list is empty; Post now still returns the unchanged idle state.
- Verification: Both zero-media and retained-media UT-019 cases passed.

### Correction 6: IR-005 — Ready media requires a predicted blob CID

- Requirement IDs: FR-012, FR-017
- Acceptance Criteria: AC-013
- Write failing test: reject ready or attached media without a non-empty
  predicted CID while preserving resumable uploading metadata.
- Confirmed failure: The migration contract reported the lifecycle constraint
  missing and accepted attached Ready media with a null predicted CID.
- Implement: Added the named media lifecycle CHECK. Uploading rows remain
  resumable only while unclaimed with no predicted CID; Ready rows require a
  non-null, non-empty predicted CID and may then be attached.
- Verification: The complete IT-026 migration contract passed, including null
  and empty CID rejection and the valid Uploading control.

### Correction 7: IR-008 — Restore a green Flutter analysis gate

- Requirement IDs: NFR-005 and the repository quality gate
- Acceptance Criteria: AC-024
- Failing evidence: `flutter analyze` reports four issues in new scheduled-post
  files.
- Implement: Documented the intentionally growing repository boundary, split two
  long expressions into named locals, and added the explicit test-helper counter
  type without changing runtime behavior.
- Verification: Full `flutter analyze` completed with `No issues found`.

## Second Implementation Review Correction Pass

The user selected `Address required changes` after the 2026-08-01
`Changes required` verdict in `06-implementation-review.md`. This correction
pass addresses the findings in review order before continuing the main test
sequence:

1. IR-009 / IT-017+IT-025 — tombstone-aware create idempotency.
2. IR-010 / IT-006+IT-016 — atomic schedule-delete media cleanup enqueueing.
3. IR-011 / IT-016 — real object deletion with retry and a final live-reference
   check.
4. IR-012 / IT-014 — reject invalid staged-image content and size.
5. IR-013 / IT-014 — idempotent staged-media deletion.

Each correction will record its own meaningful red result, minimum
implementation, focused green result, and neighboring verification below.

### Correction 8: IR-009 — Tombstone-aware create idempotency

- Requirement IDs: FR-005, FR-013, FR-014, NFR-001
- Acceptance Criteria: AC-005, AC-016, AC-017
- Test IDs: IT-017, IT-025
- Write failing test: finalize a schedule, replay its original owner operation
  and request hash, require the completed URI/CID/timestamp without recreating
  an active row, then reject a changed hash.
- Confirmed failure: The focused test failed to compile because the create
  result could not represent completed publication identity and `Store.Create`
  queried only active schedules.
- Implement: Added a typed completed create projection and an owner/operation
  tombstone lookup after the active-operation lookup but before capacity or
  media claims. Matching hashes return Published identity; changed hashes keep
  the stable operation-conflict behavior.
- Verification: The focused correction test and neighboring finalization,
  orphan-staging, and transactional-capacity tests passed against real
  Postgres.
- Refactor: Kept the raw parameterized tombstone query centralized in
  `store_queries.go`; no API or active-resource behavior changed.

### Correction 9: IR-010 — Schedule deletion queues attached private media

- Requirement IDs: FR-010, FR-017, FR-020, NFR-004
- Acceptance Criteria: AC-010, AC-014, AC-030
- Test IDs: IT-006, IT-016
- Write failing test: attach Ready private media to a schedule, delete it,
  require the schedule/media rows to disappear with exactly one cleanup job,
  prove no later claim, and repeat deletion safely.
- Confirmed failure: The focused real-Postgres test found zero cleanup jobs
  after deletion because the media foreign key cascaded metadata without first
  preserving its object key.
- Implement: Under the existing per-schedule effect lock and row lock, Delete
  now locks/collects attached object keys, deletes the active schedule/media,
  and inserts deduplicated cleanup jobs in the same transaction. An absent row
  is treated as an idempotent completed delete.
- Verification: The focused correction test and neighboring Publishing race,
  effect-fence, lifecycle cleanup, and finalization tests passed.
- Refactor: Reused the finalization media-lock query and existing content-free
  cleanup-job insert contract.

### Correction 10: IR-011 — Execute object cleanup and fence re-upload races

- Requirement IDs: FR-014, FR-020, NFR-004
- Acceptance Criteria: AC-030
- Test ID: IT-016
- First failing test: drive one queued object through an injected delete failure,
  verify its retry deadline/attempt, then succeed and remove both the object and
  cleanup job.
- First confirmed failure: The test failed to compile because no cleanup
  processor/options contract existed; only manual queue state methods were
  implemented.
- Implement: Added an HTTP-independent bounded cleanup processor with injected
  clock, lease, and retry delay. It claims jobs, performs a final database
  reference check, invokes idempotent object deletion, and completes or retries
  with the allowlisted safe error class.
- Second failing test: pause a claimed object deletion, attempt a same-ID private
  re-upload, then finish cleanup and retry the upload.
- Second confirmed failure: The concurrent re-upload succeeded while deletion
  was in progress, allowing the cleanup processor to remove the newly live
  object.
- Implement: Private-media reservation now serializes on any cleanup row for its
  opaque object key. It cancels Pending cleanup before reuse and rejects while a
  Deleting lease owns the key; retry after completion creates the object safely.
- Verification: Both focused processor tests and the race-enabled neighboring
  lifecycle, media service, and schedule-delete suites passed against real
  Postgres with the fake object store.
- Remaining IT-016 scope: Replacement and account-deletion producers remain
  tied to their later scheduled-update/account-lifecycle slices. IT-016 stays In
  progress rather than overstating those paths as implemented.

### Correction 11: IR-012 — Reject invalid staged-image bodies

- Requirement IDs: FR-017, NFR-004
- Acceptance Criteria: AC-012
- Test ID: IT-014
- Write failing test: submit empty, arbitrary non-image, declared-MIME mismatch,
  and configured-oversize bodies through the authenticated scheduled-media PUT;
  require 422 `scheduled_media_invalid` and no service call.
- Confirmed failure: Empty, malformed, and MIME-mismatched inputs reached the
  service and returned 200 because the shared immediate-upload decoder validated
  only the declared MIME allowlist and byte limit.
- Implement: Added scheduled-staging-only content sniffing after the bounded
  body decode and before durable reservation. Empty bodies and bytes whose
  detected image type does not match the declared canonical type return the
  standard content-free validation envelope; the existing immediate blob path
  remains unchanged.
- Verification: The focused invalid-body table and neighboring scheduled-media
  privacy plus shared immediate blob validation/decoding tests passed under the
  race detector.
- Refactor: Updated the successful handler fixture to use a detected JPEG
  signature rather than arbitrary bytes.

### Correction 12: IR-013 — Idempotent non-disclosing staged-media deletion

- Requirement IDs: FR-017, FR-020, RULE-004
- Acceptance Criteria: AC-011, AC-030
- Test ID: IT-014
- Write failing test: issue a foreign delete without changing owner media, then
  delete as owner twice and require both owner attempts to succeed with one
  cleanup job.
- Confirmed failure: The focused real-Postgres service test returned
  `scheduled media not found` for the non-owner no-op and would return the same
  error on the repeated owner delete.
- Implement: The private-media service now treats the store's missing/foreign
  sentinel as an idempotent successful no-op. Attached owner media still returns
  the existing conflict and no foreign row can be removed.
- Verification: The service test passed with real Postgres; the handler test now
  performs two DELETE requests and observes 204 twice. Race-enabled neighboring
  cleanup/re-upload and scheduled-media validation/privacy tests all passed.
- Refactor: Kept non-disclosure at the service boundary so handlers retain the
  standard error mapping for real dependency failures.

### Second correction pass verification

- Correction status: Completed. IR-009 through IR-013 now have focused red/green
  evidence and passing neighboring regression coverage; the overall scheduled
  posts implementation remains In progress.
- Go aggregate gate: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:15557/craftsky_dev?sslmode=disable go test -race -count=1 ./internal/scheduledposts ./internal/api ./internal/db ./internal/app ./cmd/appview` passed.
- Flutter feature gate: `flutter test test/scheduled_posts` passed all eight
  tests.
- Flutter static analysis: `flutter analyze` completed with `No issues found`.
- Whitespace gate: `git diff --check` passed.
- Remaining implementation gates: IT-015 still requires MinIO-backed object
  verification. IT-016 remains In progress until the later replacement and
  account-deletion paths enqueue cleanup. The later main-sequence tests and the
  full `just test` / `just app-test` gates have not yet been run.
- Manual and external gates: MAN-001 through MAN-006 and GAP-001 through GAP-004
  remain unexecuted as documented in `02-acceptance-tests.md`.

## Third Implementation Review Correction Pass

The user selected `Address required changes` after the 2026-08-01 fresh
`Changes required` verdict in `06-implementation-review.md`. This approval
authorizes the high-risk privacy and validation corrections in review order:

1. IR-014 / IT-016 — fence an object deletion across cleanup-lease expiry so a
   stale deleter cannot remove a newly live same-key re-upload.
2. IR-015 / IT-014 — reject structurally corrupt JPEG, PNG, and WebP bodies even
   when their signatures match the declared MIME type.

Each correction will begin with one deterministic failing test, implement only
the linked behavior, rerun neighboring race/handler coverage, and record its
red/green evidence below. The remaining IR-001 feature sequence stays In
progress after this bounded correction pass.

### Correction 13: IR-014 — Fence cleanup effects across lease expiry

- Requirement IDs: FR-017, FR-020, NFR-004
- Acceptance Criteria: AC-030
- Test ID: IT-016
- Write failing test: block worker A inside object deletion, expire and recover
  its cleanup lease through worker B, begin a same-ID re-upload, then release
  both workers and require the recreated bytes to remain readable.
- Confirmed failure: The focused test failed to compile because the cleanup
  store/processor had no effect-guard contract; its only fence ended when
  `PrepareCleanupDelete` committed before the object-store call.
- Implement: Added a per-object session advisory guard around final reference
  check, object deletion, and cleanup completion/retry. Private-media reservation
  takes the matching transaction advisory lock before inspecting or cancelling
  cleanup work, so recovery may replace a lease but cannot overlap a stale
  deleter with a newly live same-key object.
- Verification: The focused expired-lease race passed under the race detector
  against real Postgres. The first neighboring run exposed an obsolete test
  barrier that waited for re-upload before allowing the now-serialized delete;
  the barrier order was corrected, then all cleanup lifecycle, schedule-delete,
  ordinary concurrent re-upload, retry, and private-media service cases passed.
- Refactor: Extracted one claim-processing helper so every cleanup path releases
  its external-effect guard before the batch advances or returns.

### Correction 14: IR-015 — Decode staged images before durable reservation

- Requirement IDs: FR-017, NFR-004
- Acceptance Criteria: AC-012
- Test ID: IT-014
- Write failing test: submit truncated JPEG, PNG, and WebP bodies whose prefixes
  identify the declared MIME type; require 422 `scheduled_media_invalid` and no
  durable media-service call.
- Confirmed failure: The truncated JPEG and PNG cases returned 200 because
  `http.DetectContentType` accepted their signatures without verifying that the
  image streams could be decoded.
- Implement: Scheduled staging now fully decodes the bounded body through Go's
  image registry and requires the decoded format to match the declared canonical
  MIME type. The standard JPEG/PNG decoders and the official
  `golang.org/x/image/webp` decoder cover the existing allowlist; the immediate
  PDS blob path remains unchanged.
- Verification: The corrupt-signature table and the scheduled-media success
  handler passed under the race detector using a genuinely encoded JPEG fixture.
  Neighboring scheduled and immediate image-upload validation/handler tests also
  passed.
- Refactor: Ran `go mod tidy`; `golang.org/x/image` is a direct dependency and
  its compatible module graph updates `golang.org/x/text` and `golang.org/x/sync`.

### Third correction pass verification

- Correction status: Completed. IR-014 and IR-015 now have meaningful red
  evidence, focused green results, and passing neighboring regression coverage;
  the overall scheduled-post feature remains In progress under IR-001.
- Go aggregate gate: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:15557/craftsky_dev?sslmode=disable go test -race -count=1 ./internal/scheduledposts ./internal/api ./internal/db ./internal/app ./cmd/appview` passed.
- Flutter feature gate: `flutter test test/scheduled_posts` passed all eight
  tests.
- Flutter static analysis: `flutter analyze` completed with `No issues found`.
- Remaining implementation gates: IT-015 still requires concrete MinIO
  verification. IT-016 remains In progress for replacement and account-deletion
  producers. The 41 untouched automated/regression IDs, full `just test`, full
  `just app-test`, MAN-001 through MAN-006, and GAP-001 through GAP-004 remain
  unexecuted or incomplete as already documented.

Later step details will be appended after each preceding loop becomes green. If execution order changes, record the reason here before continuing.

## Verification Commands

Focused commands will be recorded per loop. The final automated gate is:

```text
cd appview && go test ./internal/scheduledposts ./internal/api ./internal/db ./internal/app ./cmd/appview
just app-test test/scheduled_posts
just app-analyze
just test
just app-test
git diff --check
```

## Manual And External Release Gates

- MAN-001 through MAN-006 remain unexecuted release checks unless explicitly run in a suitable device/deployed environment.
- GAP-001 through GAP-004 remain external or live-system limitations exactly as documented in `02-acceptance-tests.md`.
- These gates do not block local implementation, but they must not be reported as automated passes.

## Execution Log

| Test ID | Red evidence | Green evidence | Refactor / notes | Status |
|---|---|---|---|---|
| UT-002 | Missing validator and error symbols | Focused and package tests passed | Added only injected-time validation | Completed |
| UT-001 | Missing eligibility contract | Focused and package tests passed | Added only domain eligibility | Completed |
| UT-004 | Missing status vocabulary/classifier | Focused and package tests passed | Added only domain status capacity rules | Completed |
| UT-005 | Missing retry timing function | Focused and package tests passed | Added six bounded attempt offsets | Completed |
| UT-006 | Missing transition/lock/fence rules | Focused and package tests passed | Added pure lifecycle state rules | Completed |
| UT-009 | Missing canonical payload contract | Focused and package tests passed | Added strict deterministic payload round-trip | Completed |
| UT-010 | Missing publication freeze contract | Focused and package tests passed | Added typed immutable identity/body freeze | Completed |
| UT-011 | Missing safe failure classifier | Focused and package tests passed | Added content-free retry/permanent classification | Completed |
| UT-013 | Missing retention functions | Focused and package tests passed | Added exact bounded deadlines | Completed |
| UT-015 | Missing opaque metadata helpers | Focused and package tests passed | Added allowlisted content-free metadata | Completed |
| UT-016 | Missing bounded worker contract | Focused and package tests passed | Added HTTP-independent batch orchestration | Completed |
| UT-018 | Missing publication session adapter | Focused and package tests passed | Reused owner-scoped background selector contract | Completed |
| UT-020 | Missing tombstone projection | Focused and package tests passed | Added content-free 30-day tombstone | Completed |
| UT-003 | Missing Flutter absolute-time model | Focused Flutter test passed | Added invariant UTC/current-zone display model | Completed |
| UT-007 | Missing composer timing state | Focused and neighboring Flutter tests passed | Added new/future/Needs attention defaults | Completed |
| UT-008 | Missing row presentation models | Focused and neighboring Flutter tests passed | Added bounded redacted row mapping | Completed |
| UT-012 | Missing capacity action state | Focused and neighboring Flutter tests passed | Added full-cap schedule disable/manage state | Completed |
| UT-014 | Missing public status decoder | Focused and neighboring Flutter tests passed | Limited management to four unpublished statuses | Completed |
| UT-017 | Missing account-keyed list providers | Focused provider test passed | Added sorted/de-duplicated refresh-replace state | Completed |
| UT-019 | Missing staging state machine | Focused and all pure Flutter tests passed | Preserved bytes and stable retry identity | Completed |
| IT-026 | Missing migration; then nullable ordinal escaped CHECK via SQL unknown | Focused real-Postgres migration test passed | Added only migration `000034` private durable schema | Completed |
| IT-003 | Missing raw-pgx store/capacity contract | Focused concurrent and package tests passed | Owner-row lock serializes the final slot; SQL centralized | Completed |
| IT-005 | Missing LWW update and frozen-record fence | Focused concurrent and package tests passed | Internal version increments; member contract has no version token | Completed |
| IT-006 | Missing shared effect lock/guard/delete | Focused ordering and package tests passed | Dedicated session guard fences external effects | Completed |
| IT-007 | Missing due claim/lease recovery path | Focused concurrent/recovery and package tests passed | SKIP LOCKED claims preserve frozen identity | Completed |
| IR-002 / IT-006+IT-007 | Missing post-claim effect guard | New and existing focused real-Postgres tests passed | One claim-to-effect protocol | Completed |
| IR-003 / IT-006 | Cancelled caller prevented advisory unlock | Cancellation and neighboring effect tests passed | Confirm unlock or discard physical connection | Completed |
| IR-006 / IT-007 | Unique-index violation for matching UUID clock bits | Collision and neighboring claim tests passed | Owner-serialized bounded TID allocation | Completed |
| IR-007 / IT-007+IT-011 | Sixth attempt became unclaimable Retrying | Lease and neighboring claim tests passed | Reclaim ambiguous final attempt without increment | Completed |
| IR-004 / UT-019 | Text-only Schedule remained Staging | Focused Flutter state tests passed | Zero sources proceed directly to Creating | Completed |
| IR-005 / IT-026 | Missing constraint; null Ready CID accepted | Complete migration contract passed | Named media lifecycle constraint | Completed |
| IR-008 | `flutter analyze` reported four issues | Full analyzer passed with no issues | Behavior-neutral lint cleanup | Completed |
| IR-009 / IT-017+IT-025 | Completed operations were absent from create idempotency | Focused replay/conflict plus neighboring Postgres tests passed | Tombstone lookup precedes capacity/media claims | Completed |
| IR-010 / IT-006+IT-016 | Schedule delete cascaded media metadata without cleanup work | Focused delete plus neighboring race/lifecycle tests passed | Delete captures keys and enqueues cleanup atomically | Completed |
| IR-011 / IT-016 | Cleanup jobs had no object-deletion processor or destructive-boundary re-upload fence | Focused delete/retry and concurrent re-upload tests passed under race | HTTP-independent processor plus object-key cleanup/reservation serialization | Completed |
| IR-012 / IT-014 | Empty/malformed/MIME-mismatched staged bodies reached durable media service | Focused invalid/oversize handler table and neighboring race tests passed | Scheduled-only content sniff before reservation | Completed |
| IR-013 / IT-014 | Repeated owner media DELETE returned 404 instead of succeeding idempotently | Real-Postgres service and repeated 204 handler tests passed | Missing/foreign delete is a non-mutating no-op | Completed |
| IR-014 / IT-016 | An expired cleanup lease could let a stale deleter overlap a newly live same-key upload | Focused expired-lease race and neighboring cleanup/media suites passed under race | Matching per-object session/transaction advisory fence | Completed |
| IR-015 / IT-014 | Signature-only sniffing accepted truncated images as durable staged media | Corrupt JPEG/PNG/WebP table and neighboring scheduled/immediate image suites passed under race | Full decode plus declared-format match before service | Completed |
| IT-016 | Missing cleanup lifecycle store API; second review found no object-deletion processor or delete-path coverage | Full lifecycle, replacement, account-deletion, cleanup retry, and race suites passed | Transactional expiry, complete cleanup producers, and leased object deletion | Completed |
| IT-017 | Missing finalization contract/store operation | Finalization and cleanup tests passed | Fenced atomic tombstone/private cleanup | Completed |
| IT-025 | Missing create media/idempotency contract | Orphan, capacity, and finalization tests passed | Atomic media claim and true-orphan expiry | Completed |
| IT-015 | Missing S3-compatible adapter | Signed protocol and concrete Compose MinIO contract passed in `just test` | AWS SDK v2 path-style private adapter plus private bucket bootstrap | Completed |
| IT-014 | Missing private media service/handlers; later reviews found invalid-body, idempotent-delete, and structural-image gaps | Service/handler, invalid/oversize/corrupt-image, owner privacy, and repeated-delete race suites passed | Owner-private validated PUT/GET and idempotent DELETE | Completed |

### Step 29: IT-015 — S3-compatible object-store adapter

- Write failing test: configure an S3-compatible endpoint, require signed
  path-style bucket requests, PUT/GET/DELETE private bytes without an ACL, and
  reject plain HTTP configuration outside dev/test.
- Confirmed failure: The test failed to compile because no S3 adapter or
  configuration contract existed.
- Implement: Added the current AWS SDK for Go v2 adapter with static provider
  credentials, custom base endpoint, path-style addressing, content-free
  dependency errors, opaque-key validation, and production HTTPS enforcement.
- Verification: The focused signed protocol contract passed.
- Completed: Added the Compose MinIO/private-bucket bootstrap and passed the same
  operations against that concrete service. Production provider evidence
  remains MAN-005/GAP-001.

### Step 30: IT-014 — Owner-private staged media API

- Order note: The route-independent service and handler loop initially proceeded
  against the adapter contract and an in-memory object-store fake; the later
  concrete MinIO gate is now complete.
- Write failing tests: stage identical/changed bytes against real Postgres and a
  fake object store; then exercise PUT/GET/DELETE handlers with authenticated
  owner/foreign contexts and inspect response fields and cache/sniffing headers.
- Confirmed failure: The service test lacked the media service/contracts; the API
  test lacked all three handlers.
- Implement: Added uploading-before-object-write metadata, raw-CID prediction,
  idempotent Ready completion, same-owner reads/deletes, checksum/size validation,
  content-free cleanup enqueueing, safe camelCase responses, standard envelopes,
  and `private, no-store`/`nosniff` preview headers.
- Verification: Focused service and handler tests passed, followed by
  `go test -race ./internal/scheduledposts ./internal/api ./internal/db`.
- Notes: Route policy/dependency registration intentionally remains IT-019.

### Steps 31–36: Complete API, storage, lifecycle, and worker wiring

- Completed IT-015 with pinned MinIO server/client images, a private bucket
  bootstrap, health checks, an isolated volume, checkout-specific port support,
  and the concrete S3-compatible PUT/GET/DELETE contract in `just test`.
- Completed IT-016, IT-017, and IT-025 for replacement, explicit deletion,
  successful-publication cleanup, lifecycle expiry, account/DID deletion,
  cleanup leasing/retry, and same-key re-upload serialization.
- Completed IT-001, IT-002, IT-004, and IT-019 with authenticated owner-scoped
  create/list/get/update/delete/manual-publication routes, strict time/kind/body
  validation, camelCase envelopes, idempotency, capacity serialization, bounded
  summaries, full detail, and private preview/download policy.
- Completed IT-020 by wiring the scheduled store, S3 adapter, publication and
  cleanup processors, bounded background workers, account-deletion service, and
  lifecycle shutdown through AppView dependencies.
- Verification: focused real-Postgres API/store/race tests passed; Compose
  configuration validation passed; the full race-enabled `just test` gate
  passed against local Postgres and MinIO.

### Steps 37–45: Publication, recovery, current policy, and operations

- Completed IT-008 through IT-013 and AT-009 through AT-013 with stable TID,
  `createdAt`, frozen record bytes/hash, private-copy PDS blob uploads, pre-PUT
  reconciliation, ambiguous-write recovery, bounded retry/Needs attention,
  active owner-session selection, stale lease/version fences, and content-free
  tombstone finalization.
- Manual Post now atomically applies the full last-write-wins edit, claims
  Publishing, and immediately attempts publication. Definite failures retain a
  Needs attention schedule; ambiguous PDS writes remain Publishing for
  reconciliation.
- Publication revalidates current membership, payload/media limits, post and
  project policy, facets/mentions, and block authorization before any PDS write.
- Completed IT-027 with content-free queue depth/due/overdue/age, publication
  latency/duration/attempt, retry/Needs attention/recovery/stale-worker, and
  cleanup depth/age/outcome signals. Added the exact metric families and initial
  alert thresholds to `appview/README.md`.
- Publication effect-lock release errors now propagate after safe connection
  disposal instead of being silently dropped.

### Steps 46–53: Flutter scheduling, management, editing, and accessibility

- Completed IT-021, IT-022, AT-001 through AT-008, AT-014, and the scheduling
  regression slice for original standard and project composers. Now remains the
  default; quotes, replies, and comments expose no scheduling control.
- Added account-keyed repository/state, private-media staging and authenticated
  previews, visible per-image/creation progress, capacity feedback, Settings
  destination with Needs attention badge, pull-to-refresh management, exact
  status rows, Publishing locks, and confirmed deletion.
- Both full composers support future and Needs attention edits, existing private
  thumbnails, missed-time copy, Delete, reschedule, and manual Post now.
- Project editing hydrates every common/craft-specific field including nested
  gauge values and preserves unchanged body, pattern, and material facets.
- Accessibility coverage verifies large text, reachable Edit/Delete controls,
  localized lock semantics, and live-region status/staging announcements.
- Verification: all scheduled-post tests plus affected composer/project tests
  passed, and `flutter analyze` completed with no issues.

### Step 54: Final verification and remaining release gates

- Passed: `./scripts/compose-dev config --quiet`.
- Passed: `just test` (all Go packages with `-race`, real Postgres, concrete
  MinIO contract).
- Passed: focused Flutter scheduled-post, project composer, and post composer
  suites, including AT-014 accessibility.
- Passed: `flutter analyze` with no issues.
- Passed: `git diff --check`.
- Repository-wide `just app-test` ran 1,190 tests and reported one unrelated
  pre-existing Instagram migration widget failure: the test expects the nested
  RichText fragment `Notification settings` to be a standalone discoverable
  Text widget. Neither that source nor its test is changed by this feature; the
  focused affected regression suites are green.
- Still manual/release-only by specification: MAN-001 through MAN-006, including
  real-device presentation, live closed-app/PDS/Tap drills, multi-device LWW,
  managed-storage controls, and deployed alert delivery/recovery.

### Review correction pass: IR-018 through IR-022

The fourth implementation review found four Must-level Flutter integration
defects and a corresponding acceptance-coverage gap. The approved correction
order is deliberately behavior-first:

1. AT-003 / FR-003 / AC-004 and AC-025: add the missing picker acceptance
   target and enforce the exact inclusive five-minute-to-28-day window in both
   composers.
2. AT-007 / FR-009 / AC-009 and AC-028: add full standard scheduled-media edit
   coverage, then hydrate existing private media into the editable attachment
   state and serialize the final media set for reschedule and Post now.
3. AT-007 / FR-009 / AC-009 and AC-028: add full project-editor round-trip
   coverage, then initialize every conditional craft/pattern state value and
   apply the same editable scheduled-media behavior.
4. AT-006 / FR-008 / AC-008: extend the management acceptance target to assert
   type, optional project title, bounded text preview, localized absolute time
   with timezone/offset, thumbnail, and status; wire the page to the tested row
   presentation model.
5. AT-001, AT-002, AT-004, AT-005, and AT-008: restore the approved named
   acceptance targets or equivalent consolidated end-to-end targets for Now,
   eligibility, scheduled submission/failure preservation, capacity, and
   delete/Publishing locks.
6. Rerun affected Flutter suites, analysis, the repository-wide Flutter gate,
   `git diff --check`, and Go gates if a server/media contract must change.

No backend, migration, lexicon, encryption, notification, or publication-policy
change is approved unless a failing acceptance test proves it is required by
these findings. The unrelated Instagram migration baseline failure remains
separate from scheduled-post behavior.

### Review correction execution: IR-018 through IR-022

- IR-021 / AT-003 red: the approved picker target did not exist and the two
  composers accepted first-day and last-day times outside the exact window.
  Green: both composers use one injected-clock picker helper that accepts only
  whole-minute values in the inclusive `now + 5 minutes` through
  `now + 28 days` window and reports the localized range error otherwise.
- IR-018 / AT-004 and AT-007 red: scheduled media were static previews outside
  `ComposerImagesState`; editing, replacement, reordering, rescheduling, and
  Post now could silently submit a different media set. Green: authenticated
  owner-private bytes hydrate into normal editable attachments, existing media
  IDs remain stable, only new images are reprocessed/staged, and both schedule
  and Post now serialize the exact final ordered media set. A narrow injectable
  materialization seam makes staging-before-create, progress, failure
  preservation, and retry identity deterministic at widget level while the
  production seam still invokes the real media preparation service.
- IR-019 / AT-007 red: the live project editor did not initialize the state
  variables controlling craft-specific and pattern fields. The first unchanged
  save assertion then found a second loss path: shared form fields supplied
  empty local initial values that overrode `FormBuilder.initialValue`, dropping
  title, pattern, materials, colours, and design tags. Green: conditional state
  is initialized before first build, form adapters preserve parent initial
  values, materials hydrate as typed values with facets, and unchanged widget
  saves round-trip every common and conditional field for knitting, sewing,
  crochet, and quilting without restaging existing media.
- IR-020 / AT-006 red: the management widget omitted the bounded project body
  preview and timezone/offset because it bypassed the tested row model. Green:
  rows now use `ScheduledPostRowModel` and render type, optional project title,
  bounded preview, first private thumbnail, localized absolute time with zone
  and UTC offset, status, Needs attention expiry, and Publishing lock.
- IR-022 / AT-001 through AT-008 green: every named Flutter acceptance target
  now exists. The restored targets exercise immediate posting, eligibility,
  exact time bounds, private staging and retry, live three-item capacity,
  management presentation, full-composer edits, confirmed deletion, Publishing
  locks, and manual pull-to-refresh recovery.

### Fourth correction pass verification

- `dart analyze` from `app/`: passed with `No issues found`.
- Focused Flutter gate covering `test/scheduled_posts`, affected standard-post
  language behavior, project hydration, project widget behavior, and submit
  adaptation: 43 tests passed.
- Repository-wide `flutter test`: 1,206 tests passed and reproduced only the
  unrelated pre-existing IT-023 Instagram migration assertion that expects the
  nested RichText fragment `Notification settings` as a standalone `Text`.
- `just test`: passed all Go packages under the race detector against the local
  Compose Postgres and MinIO services.
- `git diff --check`: passed.
- MAN-001 through MAN-006 and GAP-001 through GAP-004 remain manual/external
  release gates and were not represented as automated passes.

## Fifth Implementation Review Correction Pass

The user selected `Address required changes` after the 2026-08-01 review found
IR-023 through IR-028. This approval covers the reviewed privacy, account
lifecycle, publication-recovery, account-isolation, capacity, payload-
preservation, and test-traceability corrections. The TDD order follows the
review's risk ordering:

1. IR-023 / IT-013+IT-016 — compose scheduled cleanup into ordinary Craftsky
   profile deletion and prove cleanup is queued in the production lifecycle.
2. IR-024 / IT-009+IT-010+AT-011 — persist the body built from predicted blob
   references before any PDS effect, verify uploaded blob identity, and cover
   deterministic crash/recovery boundaries.
3. IR-026 / IT-021+REG-010 — capture and enforce `ActiveAccountLease`
   generations across delayed list, detail, media, staging, update, and delete
   work.
4. IR-025 / AT-005+AT-007 — retain the three-item capacity display while
   permitting an existing non-Publishing schedule to save or reschedule.
5. IR-027 / UT-009+AT-007 — preserve stored standard-post facets when text is
   unchanged for both reschedule and Post now.
6. IR-028 / AT-009 through AT-013, IT-023, and IT-027 — restore scenario-level
   acceptance evidence for retry, recovery, retention, privacy, notification,
   and operational boundaries.

Each item begins with one focused meaningful failure, receives only its linked
minimum implementation, and records focused and neighboring green evidence
below. The completion checklist remains open until all correction tests and
aggregate gates have been rerun.

### Correction 15: IR-023 — Compose scheduled cleanup into profile deletion

- Requirement IDs: FR-018, FR-020
- Acceptance Criteria: AC-021, AC-030
- Test IDs: IT-013, IT-016
- Write failing test: require `profileMembershipDeletion` to invoke scheduled
  account cleanup inside its existing actor-deletion transaction between
  notification deletion and Instagram membership inactivation.
- Confirmed failure: the focused lifecycle test failed to compile because the
  production composition had no scheduled-deletion dependency.
- Implement: constructed one `scheduledposts.AccountDeletion` in AppView deps,
  composed it into ordinary Craftsky profile deletion, and retained the same
  service in the terminal DID-deletion handler list.
- Integration verification: a real-Postgres profile-indexer delete now removes
  the owner's active schedule, staged-media metadata, and publication tombstone
  while committing the opaque media cleanup job before the profile cascade.
  Focused lifecycle and terminal-deletion tests passed.

### Correction 16: IR-024 — Freeze predicted media before PDS effects

- Requirement IDs: FR-012, FR-013
- Acceptance Criteria: AC-013, AC-017
- Test IDs: IT-009, IT-010, AT-011
- Write failing test: pause at the first PDS upload and require the database to
  already contain canonical record bytes referencing the staged media's stored
  predicted raw CID.
- Confirmed failure: the upload-boundary test found no frozen record because the
  live processor uploaded media first and assembled the record from the returned
  PDS blob map afterward.
- Implement: publication snapshots now load `blob_cid`; production invokes the
  tested freeze helper to persist the canonical body/hash from predicted
  CID/MIME/size metadata before session or PDS work. Every attempt then uploads
  the private bytes and rejects any PDS blob response whose typed identity or
  canonical blob map differs from the frozen prediction.
- Verification: focused freeze and mismatched-response tests passed. The restored
  `recovery_acceptance_test.go` deterministically covers failure before any PDS
  upload, after a partial upload, after media but before record write, and after
  record commit but before local completion. Every recovery reused identical
  TID/`createdAt`/record bytes/media refs and produced at most one record write.
  The full scheduled-post backend package also passed.

### Correction 17: IR-026 — Fence Flutter private work by active lease

- Requirement IDs: FR-008, RULE-004
- Acceptance Criteria: AC-008, AC-011, AC-027
- Test IDs: IT-021, REG-010
- Write failing test: start Alice's list load behind a completer, activate Bob,
  then complete Alice's response and require it not to become provider data.
- Confirmed failure: the delayed Alice future resolved with private rows after
  the activation generation changed.
- Implement: list builds/refreshes capture and validate `ActiveAccountLease`;
  management detail/delete closures recheck the same lease after every await and
  pass it into the editor. Both full composers bind repository, media staging,
  hydration, update, publication, and deletion to that captured owner, discard
  stale completions, and close a scheduled editor safely after an account
  activation change.
- Verification: completer-controlled delayed list, detail, private-media, and
  mutation tests passed. Existing full standard/project scheduled-edit tests
  remained green, and focused static analysis reported no issues.

### Correction 18: IR-025 — Let an editor retain its existing capacity slot

- Requirement IDs: BR-002, BR-003, FR-006, FR-009
- Acceptance Criteria: AC-007, AC-009
- Test IDs: UT-012, AT-005, AT-007
- Write failing test: derive capacity at three retained rows for an editor that
  already owns one row and require scheduling to remain enabled while the count
  and management affordance still report full capacity.
- Confirmed failure: the capacity model had no retained-slot input, so every
  Later action was disabled at three rows, including updates to one of those
  rows.
- Implement: `ScheduleCapacityState` now distinguishes a new post from an
  existing scheduled post. Both full composers pass retained-slot ownership for
  edit flows; new composers retain the server-aligned three-item limit.
- Verification: focused model and widget suites passed. A new standard post is
  still blocked at three, refresh still unlocks it after deletion, and both
  standard and project editors can reschedule while displaying `3 of 3
  scheduled`.

### Correction 19: IR-027 — Preserve standard facets for unchanged text

- Requirement IDs: FR-009
- Acceptance Criteria: AC-009
- Test IDs: UT-009, AT-007
- Write failing test: open a standard scheduled post containing a stored mention
  facet, inject a resolver that cannot recreate that mention, submit unchanged
  text, and require both reschedule and Post now to send the stored facet map.
- Confirmed failure: the standard composer regenerated facets unconditionally;
  the unresolved stored mention was omitted from the outgoing payload.
- Implement: unchanged scheduled text now reuses the stored raw facet list.
  Edited text and every new/immediate post continue through the normal final-
  text generator.
- Verification: the full scheduled-edit widget suite passed, including exact
  facet preservation for reschedule and Post now plus the neighboring media,
  account-generation, capacity, and project round-trip cases.

### Correction 20: IR-028 — Restore backend acceptance traceability

- Requirement IDs: BR-001, BR-004, FR-011 through FR-021, NFR-001 through
  NFR-004, NFR-006, RULE-001, RULE-004 through RULE-008
- Acceptance Criteria: AC-011 through AC-023, AC-030 through AC-033
- Test IDs: AT-009 through AT-013, IT-009, IT-010, IT-013, IT-018, IT-023,
  IT-027
- Coverage red: the approved acceptance document named five backend acceptance
  targets and several high-risk integration targets that did not exist, while
  the implementation log claimed the entire Must suite complete.
- Implement: added durable real-Postgres acceptance targets for healthy due
  publication, the complete six-attempt retry window and permanent invalidity,
  deterministic crash recovery, inclusive retention cutoffs, owner-private
  access, notification isolation, and content-free operational signals. Updated
  the acceptance document with individual-scenario mappings where existing
  focused API, finalization, cleanup, lifecycle, or concrete-metric tests form
  the consolidated evidence.
- Verification: the full `internal/scheduledposts` package passed. The new
  targets prove zero early effects and one final record, no seventh automatic
  attempt, byte-identical retained content, stable frozen publication across
  every TD-007 boundary, safe deadline inclusion, live-reference retention,
  opaque owner-scoped access, no notification producer dependency, queue/
  retry/recovery/stale-worker/cleanup signals, and privacy-canary exclusion.

### Fifth correction pass verification

- `dart analyze` from `app/`: passed with `No issues found`.
- Focused Flutter gate covering all `test/scheduled_posts`, standard-post facet
  submission, and the neighboring project composer: 41 tests passed.
- Repository-wide `flutter test`: 1,212 tests passed and reproduced only the
  unrelated pre-existing Instagram migration assertion that searches for the
  nested RichText fragment `Notification settings` as a standalone `Text`.
- Full `internal/scheduledposts` backend package: passed against the local test
  Postgres, including AT-009 through AT-013 and IT-023/IT-027.
- `just test`: passed every Go package under the race detector against the local
  Compose Postgres and MinIO services.
- `./scripts/compose-dev config --quiet`: passed.
- `git diff --check`: passed.
- MAN-001 through MAN-006 and GAP-001 through GAP-004 remain the documented
  manual/external release gates; no manual or deployed-environment pass is
  claimed here.

## Sixth Implementation Review Correction Pass

The user selected `Address required changes` after the 2026-08-02 review found
IR-029 through IR-033. This authorizes the reviewed high-risk test work for
publication recovery, private-data telemetry, notification isolation, and
operational signals. Production behavior will change only if a new acceptance
test exposes a requirement-linked defect.

### Correction order

1. **IR-029 / AT-011, IT-009, IT-010 / FR-011 through FR-014 / AC-017** —
   replace handled-error retry scenarios with deterministic worker-stop
   barriers that leave the row Publishing, expire and reclaim its lease, recover
   with a new worker, and reject a stale prior outcome.
2. **IR-031 / IT-027 / NFR-006 / AC-033** — drive publish success, transient
   retry, sixth auth failure into Needs attention, lease recovery, stale-worker
   rejection, cleanup success, and cleanup failure through production-wired
   observers.
3. **IR-030 / IT-018 / NFR-003 / AC-023** — reuse the complete lifecycle
   capture from step 2, then add production logs and API error surfaces for
   canary-bearing create/edit/preview/publish/retry/delete/cleanup paths and scan
   every captured surface.
4. **IR-032 / IT-023 / FR-021 / AC-031 and AC-032** — retain the static import
   boundary and add behavioral before/after notification-store evidence for
   retry, Needs attention, and successful publication.
5. **IR-033 / IT-006, IT-019, IT-020** — update the acceptance specification to
   map the existing consolidated store, route/API, and server-worker tests at
   scenario level.
6. Rerun the focused scheduled-post backend and Flutter account-boundary suites,
   then `dart analyze`, `just test`, the repository-wide Flutter suite,
   Compose configuration validation, and `git diff --check`.

### TDD execution record

Each correction below must record its focused red/green evidence before it is
marked complete.

#### Correction 21: IR-029 — real crash/lease recovery acceptance

- Write failing test: assert a boundary interruption leaves the schedule in
  Publishing rather than converting it to a normal Retrying failure.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:15557/craftsky_dev?sslmode=disable go test ./internal/scheduledposts -run TestScheduledPublicationRecoversTheSameFrozenRecordAcrossCrashBoundaries -count=1`
- Confirmed failure: the partial-upload scenario reported `retrying`, proving
  the old fixture had handled an injected error instead of stopping with its
  Publishing lease intact.
- Implement: the fake PDS now exposes deterministic stop barriers before the
  first upload, before a later upload, before record lookup, and after a
  committed record write. Each scenario recovers the panic at the process
  boundary, confirms the row remains Publishing with frozen bytes, expires and
  reclaims the lease through `ClaimDue`, rejects the old lease, and runs the
  recovered claim through a newly constructed worker. Identity, body, media
  references, and the one-visible-record outcome remain stable.
- Green verification: all four
  `TestScheduledPublicationRecoversTheSameFrozenRecordAcrossCrashBoundaries`
  scenarios passed against the checkout-specific Postgres service.

#### Correction 22: IR-031 — complete operational lifecycle

- Write failing test: require publish success, Needs attention exhaustion, and
  cleanup failure signals in addition to the existing retry/recovery coverage.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:15557/craftsky_dev?sslmode=disable go test ./internal/scheduledposts -run TestScheduledOperationsEmitBoundedContentFreeSignals -count=1`.
- Confirmed failure: the production-wired capture contained queue, auth retry,
  lease recovery, stale-worker, and cleanup-success signals but no
  `publish:success`, `needs_attention`, or object-delete cleanup-failure signal.
- Implement: extended the deterministic lifecycle fixture to publish a healthy
  item, run all six auth-unavailable attempts into Needs attention, preserve the
  existing recovery/stale checks, and exercise both successful cleanup and a
  retryable object-delete failure. The observer output is scanned for every
  required outcome and for payload, account, session, provider-response, media,
  and cleanup canaries.
- Green verification: the focused production-wired scheduled observability test
  passed against Postgres with all required lifecycle signals present and no
  canary in the captured attributes.

#### Correction 23: IR-030 — lifecycle privacy capture

- Write failing test: require captured production surfaces for the approved
  lifecycle paths and fail when any privacy canary appears.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:15557/craftsky_dev?sslmode=disable go test ./internal/api ./internal/scheduledposts ./internal/middleware ./internal/observability -run 'TestScheduledHandlerCaptureExcludesPrivateCanaries|TestScheduledOperationsEmitBoundedContentFreeSignals|TestHTTPMetrics_ExportsSentryTransactionWithRoutePattern|TestCaptureErrorUsesClassifiedSentinelWithoutRawErrorDetails|TestUnpublishedScheduleAndPreviewRemainOwnerPrivate' -count=1`.
- Confirmed failure: review identified that the existing privacy acceptance
  fixture inspected selected metadata but did not capture scheduled-handler
  logs/error envelopes or map the real lifecycle metrics and shared HTTP trace
  surfaces. The new handler-capture test passed on its first execution, so no
  production redaction defect was found and no failing result is fabricated.
- Implement: added `TestScheduledHandlerCaptureExcludesPrivateCanaries`, which
  sends canary-bearing create/edit/delete bodies through closed-store failures
  and canary-bearing object keys, signed URLs, filenames, tokens, and provider
  errors through the media handlers. It captures JSON logs and HTTP responses
  and requires only fixed messages, bounded error classes, and safe envelopes.
  IT-018/AT-013 now map this capture together with the complete operational
  lifecycle metric scan, route-pattern trace export, and classified Sentry
  error tests.
- Green verification: the focused API handler privacy test passed; the combined
  handler, owner-private access, publication lifecycle, HTTP route-pattern
  trace, and error-classification privacy suite passed across
  `internal/api`, `internal/scheduledposts`, `internal/middleware`, and
  `internal/observability`.

#### Correction 24: IR-032 — behavioral notification isolation

- Write failing test: seed notification-boundary tables, run retry, Needs
  attention, and successful publication, and require an unchanged snapshot.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:15557/craftsky_dev?sslmode=disable go test ./internal/scheduledposts -run 'TestScheduledLifecycleLeavesNotificationStateUnchanged|TestScheduledLifecycleHasNoNotificationProducerBoundary' -count=1`.
- Confirmed failure: the first run failed to compile because the new fixture
  used the API-facing `Resource` return type where `Store.Create` returns the
  durable `ScheduledPost` type. After correcting the harness, the behavioral
  assertion passed on its first execution; no production notification side
  effect was found.
- Implement: retained the static import/table-name producer-boundary scan and
  added `TestScheduledLifecycleLeavesNotificationStateUnchanged`. It seeds
  notification event, preference, push-delivery, and account-subscription
  sentinels, captures a before snapshot, drives a transient retry, immediate
  policy failure into Needs attention, and successful PDS publication through
  production store/worker/processor code, proves each transition occurred, and
  requires the notification snapshot to remain identical.
- Green verification: both notification-boundary tests passed against the
  checkout-specific Postgres service.

#### Correction 25: IR-033 — consolidated target mappings

- Update `02-acceptance-tests.md` only after the mapped tests are confirmed to
  execute every setup/action/assertion for IT-006, IT-019, and IT-020.
- Verification: IT-006 now maps the real store serialization test and API lock
  envelope coverage; IT-019 maps the route-policy registry plus scheduled
  post/media handler suites and server middleware; IT-020 maps the immediate
  draining/cancellation server loop, bounded worker package behavior, config,
  and production composition root. Focused execution is included in the final
  verification below.

### Sixth correction pass verification

- Focused backend gate across `internal/scheduledposts`, `internal/api`,
  `internal/routes`, `internal/app`, and `cmd/appview`: passed against the
  checkout-specific Postgres service.
- Combined IT-018 privacy gate across scheduled-handler logs/errors,
  owner-private access, complete lifecycle metrics, route-pattern HTTP traces,
  and classified external errors: passed.
- `just test`: passed every AppView Go package against the checkout-specific
  Compose Postgres and MinIO services.
- `dart analyze` from `app/`: passed with `No issues found`.
- Focused Flutter gate covering all `test/scheduled_posts` tests plus the
  modified standard/project composer seams: 65 tests passed.
- Repository-wide `flutter test`: reproduced the same unrelated pre-existing
  Instagram migration assertion at
  `instagram_migration_page_test.dart:256`; an isolated rerun confirms it
  searches for the nested RichText fragment `Notification settings` as a
  standalone `Text`. No scheduled-post test failed.
- `./scripts/compose-dev config --quiet`: passed for the checkout-specific
  Compose project.
- `git diff --check`: passed.
- MAN-001 through MAN-006 and GAP-001 through GAP-004 remain the documented
  manual/external release gates; no manual or deployed-environment pass is
  claimed here.

## Seventh Implementation Review Correction Pass

The user selected `Address required changes` after the follow-up 2026-08-02
review found IR-034 and IR-035. This pass closes the missing Settings acceptance
evidence and corrects the partial-upload recovery traceability target. Production
behavior will change only if the focused acceptance test exposes a
requirement-linked defect.

### Correction order

1. **IR-034 / AT-006 / FR-008 / AC-008 and AC-031** — add a focused Settings
   widget test that proves the Scheduled posts entry, non-zero Needs attention
   badge, zero-count badge omission, active-account isolation, and typed route.
2. **IR-035 / IT-009 / FR-012 and FR-013 / AC-013 and AC-017** — map the
   partial-upload crash/restart scenario to the recovery acceptance test that
   actually drives the four process-stop boundaries and stable frozen record.
3. Rerun the focused Settings and scheduled-management Flutter tests, the real
   partial-upload recovery test, static analysis, aggregate backend and Flutter
   gates, and `git diff --check`.

### TDD execution record

#### Correction 26: IR-034 — Settings management entry acceptance

- Write focused test: added an AT-006 widget test that renders the real Settings
  page with Alice active and two Needs attention items, switches to Bob with no
  items, and follows the typed Scheduled posts route.
- First execution: the new test passed on its first run. The reviewed gap was
  missing executable evidence, not a production defect, so no failing result or
  production change is fabricated.
- Implement or confirm production behavior: confirmed the existing Settings
  page watches the account-keyed scheduled-post provider, derives only the
  active account's Needs attention count, omits a zero badge, and invokes
  `ScheduledPostsRoute`. No production edit was needed.
- Green verification: the focused Settings file passed all three tests; the
  combined Settings, management-page, and provider gate passed seven tests.

#### Correction 27: IR-035 — partial-upload recovery mapping

- Inspect the mapped executable scenario: confirmed
  `TestScheduledPublicationRecoversTheSameFrozenRecordAcrossCrashBoundaries`
  covers worker stops before upload, after a partial upload, after all media but
  before the record lookup/write, and after PDS commit but before local
  completion. It retains Publishing, expires/reclaims the lease with a new
  worker, rejects stale completion, and preserves the frozen identity/body.
- Update `02-acceptance-tests.md`: AT-006 now maps the Settings, management, and
  account-keyed provider seams explicitly; IT-009 now targets
  `recovery_acceptance_test.go` with a scenario-level mapping.
- Verification: the focused real-Postgres recovery test passed all four crash
  boundaries.

### Seventh correction pass verification

- Focused Flutter Settings/management gate: seven tests passed across the
  Settings page, scheduled management page, and account-keyed provider targets.
- Focused backend partial-upload recovery gate: passed against the
  checkout-specific Postgres service.
- `dart analyze`: passed with `No issues found` after correcting two style-only
  findings in the new test.
- `just test`: passed every AppView Go package against the checkout-specific
  Compose Postgres and MinIO services.
- Repository-wide `flutter test`: 1,213 tests passed and the same unrelated
  pre-existing Instagram migration assertion failed at
  `instagram_migration_page_test.dart:256`; it searches for the nested RichText
  fragment `Notification settings` as a standalone `Text`. No scheduled-post
  or Settings test failed.
- `git diff --check`: passed.
- MAN-001 through MAN-006 and GAP-001 through GAP-004 remain the documented
  manual/external release gates; no manual or deployed-environment pass is
  claimed here.

## Completion Checklist

- [x] All Must requirements covered by passing tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Migration, privacy, and security constraints verified
- [x] Documentation and execution log updated
- [x] Full diff checked against requirement/test traceability
- [ ] Implementation review completed or explicitly deferred
