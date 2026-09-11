# TDD Implementation Plan: Moderation Cases, Strikes, Appeals, And Suspension

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`)
- Coding plan: `04-coding-plan.md`
- Explicit implementation approval: Received from the user on 2026-09-10.
- Risk: High. The approval covers the migration, authentication, authorization, privacy, and account-enforcement work specified by the approved documents. No lexicon or PDS-owned record changes are authorized.

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update a focused failing test before implementation.
- Run the smallest relevant test first and record a meaningful red result.
- Implement only enough behavior to make the active test green.
- Refactor only while tests are green.
- Keep traceability and command evidence updated after each slice.
- Use the scoped inline-`pgx` exception only inside the moderation store and transaction bridges approved in `04-coding-plan.md`.
- Keep stored enforcement projections authoritative until an atomic worker commit.
- Keep notification intent in the authoritative transaction and provider preference enforcement in dispatch.
- Preserve all legacy reports and moderation outputs without synthetic cases or owner history.

## Test Order

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State | Status |
|---:|---|---|---|---|---|
| 1 | UT-005 | FR-032 | AC-036 | Reference package does not exist. | Complete |
| 2 | IT-020, REG-008 | FR-001, FR-006, FR-011, FR-013, FR-019, FR-024, FR-026, FR-036, NFR-001, RULE-003 | AC-002, AC-004, AC-018, AC-021, AC-023, AC-029, AC-045 | Migration and constraints do not exist. | Complete |
| 3 | IT-001, IT-002, REG-001 | FR-001, FR-002, FR-029, RULE-007 | AC-002, AC-003, AC-034, AC-036 | Accepted reports do not create or join cases. | Complete |
| 4 | UT-002, UT-009 | FR-012, FR-026, FR-027, FR-028, FR-033, RULE-001, RULE-012 through RULE-015 | AC-004, AC-029, AC-032, AC-035, AC-037, AC-042 | Decision and visibility policy types do not exist. | Complete |
| 5 | IT-003, IT-004, IT-015, IT-022 | FR-005, FR-012 through FR-014, FR-024, FR-026, NFR-001, NFR-002, RULE-001, RULE-003, RULE-004, RULE-015 | AC-004, AC-006, AC-021, AC-029, AC-037, AC-038 | Atomic adjudication service does not exist. | Complete |
| 6 | UT-001, UT-003 | FR-013, FR-034, RULE-002 through RULE-004 | AC-004 through AC-009, AC-038, AC-039 | Expiry arithmetic and standing derivation do not exist. | Complete |
| 7 | AT-007, IT-005, IT-006, IT-007, IT-008 | BR-002, BR-006, FR-006, FR-013 through FR-019, FR-029, FR-030, FR-034, RULE-001 through RULE-004, RULE-006, RULE-009, RULE-012 | AC-005 through AC-010, AC-023, AC-038, AC-039, AC-042 | Standing transitions and expiry processor do not exist. | Complete |
| 8 | UT-008, AT-006, IT-009 | BR-003, FR-006, FR-010, FR-011, FR-029, FR-031, RULE-005, RULE-016, RULE-017 | AC-017, AC-018, AC-040, AC-043 | Appeal lifecycle does not exist. | Complete |
| 9 | UT-013, IT-019 | BR-005, FR-024, FR-025 | AC-021, AC-022 | Source-neutral trusted adapter does not exist. | Complete |
| 10 | UT-011, AT-005, IT-010, IT-011 | BR-004, FR-003 through FR-005, FR-014, FR-023, FR-024, NFR-001 through NFR-004, RULE-001 | AC-019 through AC-021, AC-027, AC-030 | Production moderator auth and admin routes do not exist. | Complete |
| 11 | UT-006, IT-012, IT-023, IT-024, IT-025 | BR-001, FR-007, FR-009, FR-012, FR-022, FR-027, FR-032, FR-033, NFR-003, NFR-004, RULE-006, RULE-009, RULE-011, RULE-013 through RULE-015 | AC-001, AC-014, AC-020, AC-026, AC-030, AC-032, AC-033, AC-036, AC-037 | Owner-safe presentation and routes do not exist. | Complete |
| 12 | UT-004, AT-003, IT-013, IT-014, REG-003, REG-006, REG-007 | BR-007, FR-017, FR-018, FR-035, NFR-005, RULE-008 | AC-010 through AC-013, AC-028, AC-041, AC-044 | Routes have no exhaustive suspension capability classification. | Complete |
| 13 | UT-012, IT-016 | BR-006, FR-019, FR-020, NFR-006, RULE-009, RULE-010 | AC-023, AC-024, AC-033, AC-038, AC-039 | Complete notification eligibility and atomic intent do not exist. | Complete |
| 14 | UT-007, IT-017, IT-018, REG-004 | BR-006, FR-020, FR-021, NFR-006 | AC-023 through AC-025 | Moderation category/payload/dispatch do not exist. | Complete |
| 15 | AT-011, IT-027 | FR-020, FR-035 | AC-024, AC-044 | API and Flutter settings lack the moderation preference. | Complete |
| 16 | UT-010, AT-001, AT-002, IT-025 | BR-001, BR-003, FR-007 through FR-010, FR-032, NFR-007, RULE-005, RULE-006 | AC-001, AC-014 through AC-017, AC-031, AC-036 | Flutter owner models, page, and appeal behavior do not exist. | Complete |
| 17 | AT-004, IT-026 | BR-006, FR-019, FR-021 | AC-023, AC-025, AC-048 | Notification destination inference cannot open moderation history. | Complete |
| 18 | UT-014, IT-021 | FR-023, NFR-002, NFR-003, NFR-008 | AC-021, AC-027, AC-046, AC-047 | Moderation alert predicates and safe observations do not exist. | Complete |
| 19 | REG-002, REG-005 | FR-026, NFR-004, RULE-008 | AC-029, AC-030 | Existing behavior must remain green after integration. | Complete |
| 20 | MAN-001 through MAN-004 | BR-004, FR-003 through FR-005, FR-008 through FR-010, FR-021, FR-023, NFR-002, NFR-006, NFR-007 | AC-015 through AC-021, AC-025, AC-027, AC-031, AC-036, AC-048 | External Retool/device/accessibility evidence is unavailable in automated tests. | Blocked: external systems/devices unavailable |

## Implementation Steps

### Step 1: UT-005

- Write failing test: Add public-reference table tests for canonical, mixed-case, malformed, non-v4, and alternate/internal identifiers.
- Run command: `cd appview && go test ./internal/moderation -run TestCaseReference`
- Confirmed failure: Build failed only because `FormatCaseReference`, `ParseCaseReference`, and `ErrInvalidCaseReference` were undefined.
- Implement: Added an opaque `CaseReference` around `uuid.UUID`, strict `MOD-<UUIDv4>` parsing, canonical lowercase formatting, and UUID access for persistence boundaries.
- Run command: `cd appview && gofmt -w internal/moderation/reference.go internal/moderation/reference_test.go && go test ./internal/moderation -run TestCaseReference` passed.
- Refactor: Kept syntax/version validation in one private helper while green.
- Notes: Bare UUIDs, alternate prefixes, nil UUIDs, malformed UUIDs, and non-v4 UUIDs are rejected; mixed-case canonical forms resolve to one public value.

### Step 2: IT-020 And REG-008

- Write failing test: Add migration/legacy-preservation test against a pre-feature schema and explicit database invariants.
- Run command: `just dev-d && just test`
- Confirmed failure: Focused test failed because `000072_moderation_cases.up.sql` did not exist.
- Implement: Added paired migration files for report-origin cases, append-only commands/decisions/effects, active projections, one logical strike, independent account-standing bases, correspondence/appeals, and actorless moderation notification/category constraints.
- Run command: `TEST_DATABASE_URL="postgres://craftsky:dev@localhost:15720/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/db -run TestModerationCasesMigrationPreservesLegacyDataAndEnforcesInvariants -count=1` passed. `just dev-d` also built the app and applied the full migration chain through `000072` successfully.
- Refactor: Kept lifecycle invariants in named tables/indexes and reversible notification constraints while green.
- Notes: The test proves old reports, outputs, and Instagram notifications survive without synthetic cases/effects/standing/appeals; down/up remains valid.

### Step 3: IT-001, IT-002, And REG-001

- Write failing test: Add concurrent open-case grouping and resolved-case/new-report behavior through accepted report intake.
- Run command: Focused PostgreSQL package tests, then `just test`.
- Confirmed failure: `IT-001` failed to compile because `NewStore`, `AcceptedReport`, and `AttachAcceptedReportTx` were absent. The ReportStore regression then failed to compile because its constructor had no case-attacher seam.
- Implement: Added typed accepted-report/case models, canonical account/post/event subject keys, safe snapshot serialization, conflict-safe open-case creation, immutable report association, and production ReportStore wiring inside its existing transaction.
- Run command: Focused real-PostgreSQL moderation and API tests passed, including concurrent grouping, post-resolution new-case behavior, duplicate report persistence, and terminal-owner rejection. `go test ./internal/app -run '^$'` also passed dependency compilation.
- Refactor: Kept SQL in `internal/moderation`; retained an optional constructor attacher solely for existing focused pre-feature fixtures while production wiring always supplies it.
- Notes: Duplicate reports remain separate immutable evidence rows. `IT-002` was already green after the `IT-001` conflict-safe open-only primitive and needed no additional production change.

### Step 4: UT-002 And UT-009

- Write failing test: Add complete policy tables for disposition/reason/consequence and visibility mapping.
- Run command: `cd appview && go test ./internal/moderation -run 'Test(DecisionPolicy|VisibilityEffects)'`
- Confirmed failure: The focused visibility test failed to compile because the mapping discarded the apply/negate operation for formal warnings. The pre-existing unrecorded policy tests were already green when this session resumed.
- Implement: Added closed disposition/reason/effect validation, explicit strike and severe-category eligibility, required violation evidence and severe rationale, at-most-one viewer visibility effect, and an operation-preserving visibility plan for formal warnings and existing `warn`/`hide`/`takedown` outputs.
- Run command: `cd appview && go test ./internal/moderation -run 'Test(ValidateDecision|EveryNamedReason|OnlyApprovedSevereReasons|MapVisibilityEffects)' -count=1` passed; `cd appview && go test ./internal/moderation -count=1` passed.
- Refactor: Kept policy validation pure, returned a copy of the approved reason set, and represented a missing formal-warning effect with the zero `EffectAction` value.
- Notes: Formal warning and viewer `warn` remain independent. Tests cover every approved reason for ordinary-strike and severe-suspension eligibility, formal-warning reversal, malformed actions/effects, and independent severe-plus-strike selection.

### Step 5: IT-003, IT-004, IT-015, And IT-022

- Write failing test: Add atomic command/replay/revision/visibility/concurrency behavior one case at a time. On resumption, the atomicity/replay tests and implementation were already present but unrecorded; added the missing service-level apply/negate matrix for all three existing visibility outputs and independent formal warnings.
- Run command: Focused moderation PostgreSQL tests, followed by `just test`.
- Confirmed failure: Historical red evidence for the unrecorded atomic service work is unavailable. The added IT-015 regression passed immediately against that implementation rather than producing a new red result.
- Implement: Existing resumed code provides the standing-before-case lock order, durable replay fingerprints, optimistic revision rejection, append-only decisions/effects, transaction-aware legacy visibility outputs, standing projection, and atomic notification insertion. No production change was needed for the added visibility matrix.
- Run command: `TEST_DATABASE_URL="postgres://craftsky:dev@localhost:15720/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/moderation -count=1` and `just test` passed.
- Refactor: None; the resumed service remained green.
- Notes: Failure injection covers every coupled decision write. Replay, conflicting replay, stale revision, no implicit strike, and visibility apply/negate are PostgreSQL-backed. The review correction replaced the nondeterministic race with bounded PostgreSQL lock barriers for both decision/expiry and reversal/expiry acquisition orders, exact terminal history/projection assertions, and idempotent retries.

### Step 6: UT-001 And UT-003

- Write failing test: Add clamped UTC calendar arithmetic and standing projection tables. These tests and their implementation were present but unrecorded when work resumed.
- Run command: `cd appview && go test ./internal/moderation -run 'Test(StrikeDeadline|Standing)'`, then the PostgreSQL-required moderation package and `just test`.
- Confirmed failure: Historical red evidence is unavailable for the resumed implementation.
- Implement: `StrikeDeadline` normalizes issuance to UTC, adds one calendar year, and clamps leap day; `DeriveStanding` counts one active logical strike per case and keeps threshold and severe bases independent. A due strike remains active until its processed projection changes.
- Run command: The focused tests, PostgreSQL-required moderation package, and `just test` passed.
- Refactor: None required during reconciliation.
- Notes: Due time and effective expiry are distinct. Worker timing and restart behavior remain Step 7 work.

### Step 7: AT-007 And IT-005 Through IT-008

- Write failing test: Reconciled existing third-strike, severe-basis, partial-reversal, strike-reapplication, and expiry tests. Added a focused explicit-restoration test requiring the exact active severe logical effect, then added pre-deadline/restart convergence coverage.
- Run command: `TEST_DATABASE_URL="postgres://craftsky:dev@localhost:15720/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/moderation -run '^TestExplicitSevereRestorationTargetsActiveEffectAndAppendsAuditEvent$' -count=1` and focused restart/expiry tests.
- Confirmed failure: The restoration test failed to compile because `SevereRestorationCommand` and `RestoreSevereSuspension` did not exist. The admin route test then failed strict JSON decoding because `effectId` was not accepted.
- Implement: Added a dedicated restoration transaction with standing-before-case locking, revision/replay checks, exact active-effect validation, `severeRestored` case event, `restore` effect event, standing recomputation, case revision, and notification intent committed together. Routed the admin restoration endpoint to it with required camelCase `effectId`. Existing expiry processing remained unchanged.
- Run command: Dedicated domain and admin restoration tests passed. PostgreSQL restart coverage passed with no processing one microsecond before the deadline, stored threshold suspension while the worker was absent, one commit 59 minutes after the deadline, and no duplicate work on retry.
- Refactor: Kept the owner as `syntax.DID` inside the restoration transaction and isolated the restoration replay fingerprint.
- Notes: Worker commit is authoritative and idempotent. Existing tests also cover threshold entry, severe-plus-strike independence, quiet severe-overlap expiry, enforcement-changing expiry notification, partial reversal, one logical strike reapplication, and expiry/third-strike serialization.

### Step 8: UT-008, AT-006, And IT-009

- Write failing test: Reconciled existing correspondence, confirmation, resolution, and policy tests; added an explicit no-action case test proving private outcomes cannot create an appeal lifecycle.
- Run command: `TEST_DATABASE_URL="postgres://craftsky:dev@localhost:15720/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/moderation -run '^(TestAppealCorrespondenceRequiresConfirmationAndUsesOneLifecycle|TestChangedAppealRecordsEffectChangeWithoutReopening|TestNoActionCaseCannotCreateAppealLifecycle|TestAppealLifecyclePolicy)$' -count=1`.
- Confirmed failure: Historical red evidence is unavailable for the resumed appeal implementation. The added no-action eligibility test passed immediately against the existing owner-visible-decision check.
- Implement: Existing resumed code stores hashed untrusted correspondence separately, creates no pending state until moderator confirmation, permits one pending-to-upheld/changed lifecycle, keeps enforcement active for status-only changes, and appends selected effect negations for changed outcomes without reopening the case. No production change was required for the added test.
- Run command: Focused appeal policy and PostgreSQL integration tests passed; the full AppView suite had already passed with the same appeal implementation after Step 7.
- Refactor: None required during reconciliation.
- Notes: Email sender identity never confers authority. No-action cases cannot consume the single lifecycle, repeated correspondence remains separate intake, and upheld outcomes create no notification work.

### Step 9: UT-013 And IT-019

- Write failing test: Add equivalent admin/future trusted-adapter normalization and service invocation tests.
- Run command: Focused adapter tests.
- Confirmed failure: Adapter tests failed before the separate normalization boundary existed and authenticated attribution could override spoofed payload fields.
- Implement: Split normalization from invocation in `TrustedInputAdapter`; authenticated attribution replaces payload actor/source fields and all admin decisions cross the shared adapter boundary.
- Run command: Focused moderation and API tests passed with PostgreSQL required; combined serial package verification also passed.
- Refactor: Kept the adapter source-neutral and exposed no persistence seam.
- Notes: The adapter exposes no table mutation seam and does not claim Ozone payload compatibility.

### Step 10: UT-011, AT-005, IT-010, And IT-011

- Write failing test: Add request parsing/pagination, moderator authentication, queue/detail, and command route tests sequentially.
- Run command: Focused API and route tests.
- Confirmed failure: Strict DTO tests initially accepted missing revisions, extra JSON values, and endpoint-inappropriate fields.
- Implement: Added strict command DTOs, mandatory `expectedRevision`, curated errors, opaque pagination/filter validation, stable queue reads, detailed case context, separate moderator authentication, and production environment variables.
- Run command: PostgreSQL-required API, middleware, and race suites passed; `go vet` passed.
- Refactor: Kept actor/source authority in authenticated context.
- Notes: Actor/source values come only from authenticated context.

### Step 11: UT-006 And IT-012 Through IT-025

- Write failing test: Add presentation privacy first, then owner standing/history/detail route behavior and `noAction` privacy.
- Run command: Focused moderation and API tests.
- Confirmed failure: Presentation tests failed before the explicit subject allowlist existed; integration tests exposed inherited reapplication deadlines and malformed cursor handling.
- Implement: Added strict owner allowlist DTOs, authenticated-owner scoping, canonical case-reference lookup, stable opaque chronology, deleted/edited subject fallback, immutable historical effect facts, and separate safe reason/detail fields.
- Run command: PostgreSQL-required moderation/API suites and focused race tests passed.
- Refactor: Removed current projection state from immutable historical events; strike `dueAt` is emitted only for its apply event.
- Notes: Owner wire DTOs are strict allowlists; only public case references leave the boundary. Review correction removed Flutter's unsupported current-state `active` field and added an exact AppView response fixture without that field; the generated mapper and owner UI now consume immutable `type`, `action`, and optional `dueAt` facts consistently.

### Step 12: UT-004, AT-003, IT-013, IT-014, REG-003, REG-006, And REG-007

- Write failing test: Add a catalogue completeness test, then middleware behavior and PDS-side-effect tests. Review correction added a production `AddRoutes` invocation matrix; it failed to compile until an innermost package-private handler probe seam existed.
- Run command: Focused route, moderation, owner lifecycle, and report tests.
- Confirmed failure: The catalogue test identified unclassified authenticated mutations.
- Implement: Added exhaustive mutation capabilities; unknown mutations remain `SuspensionUnspecified` and fail catalogue construction. Preserved reports/account deletion and proved no suspended PDS side effects. Review correction invokes every registered authenticated production route for a stored suspended member and asserts allowed handler entry or canonical denial before body/handler/PDS work. PostgreSQL integrations drive threshold, severe, expiry, reversal, and restoration through the real standing reader, and the final IT-014 correction invokes actual `AddRoutes` post handlers: retained owner deletion reaches a recording `NewPDSEffects` executor with the stored owner generation while denied post creation performs no body parsing or PDS work.
- Run command: PostgreSQL-required routes/API/account-deletion/moderation suites and `go vet` passed. The full registered-route matrix and combined stored-standing, real-handler, recording-PDS integration pass without skipped database coverage.
- Refactor: Kept suspension policy centralized in the route catalogue.
- Notes: Unknown mutations fail closed; owner-requested removal remains distinct.

### Step 13: UT-012 And IT-016

- Write failing test: Add complete event eligibility table and atomic outbox behavior.
- Run command: Focused notification policy and moderation PostgreSQL tests.
- Confirmed failure: Eligibility coverage exposed missing event cases and notification intent ownership.
- Implement: Made `notifications.ShouldNotifyModeration` authoritative and exhaustive, including silent status-only appeals/quiet expiry and notifying enforcement-changing expiry; retained transaction rollback behavior.
- Run command: PostgreSQL-required moderation and notifications suites passed.
- Refactor: Moved the intent DTO into notifications to avoid a dependency cycle.
- Notes: Appeal status-only and quiet expiry remain notification-free.

### Step 14: UT-007, IT-017, IT-018, And REG-004

- Write failing test: Add category/payload privacy, current-preference dispatch, and retry/account-binding behavior. The first resumed test asserted that moderation provider work is queued even when the category is disabled at enqueue time, leaving current-preference suppression to dispatch.
- Run command: `TEST_DATABASE_URL="postgres://craftsky:dev@localhost:15720/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/notifications -run '^TestModerationWriterQueuesDeliveryWhenPreferenceIsDisabledUntilDispatchRecheck$' -count=1`.
- Confirmed failure: Two durable moderation events existed but only one push delivery was queued; the event created while moderation push was disabled had no delivery work.
- Implement: Removed enqueue-time delivery suppression from `ModerationWriter`. The event still records its preference snapshot, while every eligible active account subscription receives durable delivery work for the dispatcher to evaluate against the current preference.
- Run command: The focused test passed. PostgreSQL-backed moderation, notifications, push, and API suites also passed with `TEST_DATABASE_REQUIRED=true`.
- Refactor: Renamed the snapshot variable to make clear that it is recorded state, not the provider-send authority.
- Notes: Actorless moderation work is claimable/listable/countable and retryable. Dispatch rechecks current preferences and active routing. Payloads contain only a safe public case reference and opaque account binding, never a recipient DID or internal identifier.

### Step 15: AT-011 And IT-027

- Write failing test: Add backend preference persistence, then Flutter settings toggle/reload behavior.
- Run command: Focused Go preferences tests and `just app-test test/notifications/pages/notification_settings_page_test.dart`.
- Confirmed failure: Flutter had no moderation category or fixed-scope preference row.
- Implement: Added moderation as the ninth fixed-`everyone` notification category with an independent push toggle, localization, icon, persistence, and reload behavior.
- Run command: Backend preference tests passed; all notification tests passed in focused and full Flutter runs.
- Refactor: Reused the existing preference model and settings page.
- Notes: Moderation has fixed scope and an independent push toggle.

### Step 16: UT-010, AT-001, AT-002, And IT-025

- Write failing test: Add additive JSON model tests, standing/history widget behavior, appeal launch/fallback, and typed route lookup.
- Run command: `just app-test test/moderation test/router/moderation_routes_test.dart`.
- Confirmed failure: Owner models, providers, routes, page, and appeal actions were absent.
- Implement: Added additive-tolerant mapped models, UUIDv4 public references, repository, account-keyed Riverpod providers, responsive history/detail UI, settings navigation, typed routes, mailto appeal, and copy fallbacks.
- Run command: Focused moderation/router/settings tests passed; the final full Flutter suite passed all 2,336 tests and analysis found no issues.
- Refactor: Kept launch and clipboard effects injectable and all private state account-keyed.
- Notes: Account-key all private state and use generated localization. Review corrections key history cards by canonical case reference, event type, and immutable occurrence time; restrict appeal actions to unappealed `decision` entries; and label immutable effect actions as `Applied`, `Expired`, `Overturned`, or `Access restored` rather than inferring current state. Widget tests cover duplicate event types, the appeal eligibility matrix, and a complete historical action sequence.

### Step 17: AT-004 And IT-026

- Write failing test: Extend existing notification-open tests for exact-account activation/recheck before moderation navigation and invalid binding failures.
- Run command: Focused Flutter notification/router tests.
- Confirmed failure: Moderation payload facts, destination type, and route dispatch were absent.
- Implement: Required and canonicalized UUIDv4 case references, added `ModerationHistoryDestination`, and routed through `AccountStandingCaseRoute` only after the existing coordinator activates the exact account and rechecks its lease.
- Run command: Thirty focused notification/router tests passed; full Flutter suite and analysis passed.
- Refactor: Extended the single existing coordinator and generated polymorphic destination mapping.
- Notes: Extend destination inference; do not add a second coordinator.

### Step 18: UT-014 And IT-021

- Write failing test: Add exact alert-boundary predicates and safe operation observation tests.
- Run command: Focused observability tests.
- Confirmed failure: Observability tests failed before moderation operation/auth/work-age observations existed.
- Implement: Added closed-label observations and exact alerts: five admin-auth failures in an inclusive five-minute window, strike expiry more than one hour overdue, and eligible moderation delivery at least fifteen minutes old. Work-age excludes disabled preferences and inactive routing; sensitive values/raw DIDs are rejected. Review correction exercises production admin-route authentication, real decision/appeal/effect-change/expiry service hooks, and `Dispatcher.ProcessBatch`, with exact metric-label and cross-output sentinel checks.
- Run command: PostgreSQL-required moderation, observability, middleware, push, routes, and app suites passed normally and under focused `-race`; `go vet` passed.
- Refactor: Wired observations at committed operation boundaries with hashed request identifiers.
- Notes: Metrics and logs must exclude all sensitive sentinels.

### Step 19: REG-002 And REG-005

- Write failing test: Add only missing regression assertions; otherwise run existing suites unchanged.
- Run command: Focused moderation-output/route suites, then `just appview-check`.
- Confirmed failure: No new product failure. Full Flutter initially found one stale settings-row count, corrected from 13 to 14. A later link-preview timeout occurred only while two heavyweight gates competed for resources and passed alone.
- Implement: Updated the existing settings regression assertion; no second visibility mechanism or API convention was added.
- Run command: Serial PostgreSQL-required `go test -p 1 ./...` passed; `go vet ./...`, all 2,334 Flutter tests, `flutter analyze`, Dart MCP analysis, and `git diff --check` passed. A fresh `just appview-check` passed every release gate with artifact `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.B90dqYg7lA`.
- Refactor: Mechanical analyzer cleanup only.
- Notes: No second visibility mechanism and no API convention drift.

### Step 20: MAN-001 Through MAN-004

- Write failing test: Not applicable; these checks cover external Retool/device/accessibility behavior.
- Run command: Manual checklists from `02-acceptance-tests.md`.
- Confirmed failure: Not applicable.
- Implement: Repository changes only if a finding maps to an approved requirement.
- Run command: Not run. Retool, real provider/device delivery, email-client presentation, and physical screen-reader/accessibility environments are unavailable in this workspace.
- Refactor: Not applicable.
- Notes: Record as blocked rather than complete when external systems/devices are unavailable.

## Completion Checklist

- [x] All Must requirements covered by passing tests or documented gaps.
- [x] All planned Must automated tests passing.
- [x] Merge-blocking PostgreSQL and routing tests pass without skips in serial required-database runs.
- [x] Relevant regression tests pass.
- [x] `just appview-check` passes.
- [x] Focused and full Flutter tests and `flutter analyze` pass.
- [x] No unlinked behavior implemented.
- [x] No lexicon or unauthorized PDS behavior changed.
- [x] Manual gaps are completed or explicitly recorded as blocked.
- [x] Implementation notes and command evidence are current.
- [x] Implementation review completed or explicitly skipped.

## Execution Notes

- 2026-09-10: User explicitly approved implementation after the approved coding-plan gate.
- 2026-09-10: Workflow documents reloaded from disk. No blocking question remains. Step 1 is next.
- 2026-09-10: UT-005 completed red-green-refactor; focused Go test passes.
- 2026-09-10: IT-020 and REG-008 completed against PostgreSQL; full compose migration startup also passes.
- 2026-09-10: IT-001, IT-002, and REG-001 completed against PostgreSQL; report and case association share one transaction.
- 2026-09-10: UT-002 and UT-009 completed; the full moderation package passes with decision policy and independent visibility/owner-warning mapping.
- 2026-09-10: Resumption reconciliation found substantial unrecorded backend implementation for later steps. Those steps remain pending or in progress until their full planned behavior and PostgreSQL evidence are recorded; no Flutter implementation was found.
- 2026-09-10: Restored the AppView compile baseline by passing the existing observer into the moderation expiry processor. `go test ./internal/app -run '^$' -count=1` and the full `just dev-d` image build/migration startup pass.
- 2026-09-10: Advanced the IT-017 enqueue/dispatch boundary safeguard out of order. The focused test failed with `events/deliveries = 2/1`, then passed after durable delivery fan-out stopped treating the enqueue-time preference snapshot as the send authority.
- 2026-09-10: PostgreSQL-required moderation, notification, and push packages pass after the resumed changes; `git diff --check` is clean.
- 2026-09-10: Reconciled and completed Steps 5 and 6. Added service-level coverage for apply/negate behavior across `warn`, `hide`, and `takedown`; it passed immediately against resumed code. Atomic rollback/replay/revision/race tests and UTC deadline/standing unit tests pass with PostgreSQL required.
- 2026-09-10: Full `just test` passed across AppView after one infrastructure-only timeout and a successful rerun with a longer command timeout.
- 2026-09-10: Completed Step 7. Added an exact-effect severe restoration command and admin route contract with distinct `severeRestored`/`restore` audit records, replay safety, atomic standing/notification updates, and PostgreSQL restart timing coverage through the one-hour convergence window.
- 2026-09-10: Completed Step 8 reconciliation. Existing appeal policy/integration tests pass with PostgreSQL required, and added coverage proves `noAction` cases cannot create owner-visible appeal state or append an appeal event.
- 2026-09-10: Completed Steps 9 through 18 with strict moderator/owner boundaries, exhaustive suspension and notification policies, actorless privacy-safe delivery, Flutter standing/history/appeal UI, exact-account notification routing, and closed-label moderation observability.
- 2026-09-10: Final serial PostgreSQL-required moderation/API/observability/middleware/push/app/routes/account-deletion suites and `go vet ./...` passed. Full Flutter passed all 2,333 tests and `flutter analyze` reported no issues. `git diff --check` passed.
- 2026-09-10: `just appview-check` remains infrastructure-blocked. Two completed attempts reached tests but disposable PostgreSQL exhausted shared locks in the account-deletion schema suite (`SQLSTATE 53200`); a longer standalone attempt reproduced the same failure. Artifacts: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.ZEmpXtHUvc` and `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.2sxsPDaOZd`. The same account-deletion suite passes serially with PostgreSQL required.
- 2026-09-10: MAN-001 through MAN-004 are blocked because Retool, real provider/device delivery, email-client presentation, and physical accessibility environments are unavailable.
- 2026-09-10: Correction pass resolved prior findings IR-001, IR-002, IR-004, IR-005, and IR-006. Exact AppView history JSON now decodes in Flutter, repeated same-type events render with unique immutable keys, IT-013 invokes all registered authenticated routes, IT-021 executes production telemetry paths with sentinel scans, and IT-022 deterministically forces all four decision/reversal/expiry lock orders. Re-review found IT-014 still needs a combined real-handler/PDS-boundary proof.
- 2026-09-10: Correction verification passed: required-PostgreSQL affected packages, full serial `go test -p 1 ./...`, focused race tests repeated three times, `go vet ./...`, all 2,334 Flutter tests, `flutter analyze`, Dart MCP analysis, and `git diff --check`.
- 2026-09-10: Fresh `just appview-check` passed all release gates. Artifact: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.B90dqYg7lA`.
- 2026-09-10: Final correction pass resolved IR-008 through IR-010. Appeal actions now appear only on unappealed decision events; immutable effect actions use historical labels; and PostgreSQL-backed `AddRoutes` coverage proves real retained deletion reaches `NewPDSEffects.DeleteRecord` with the stored generation while denied post creation reads no body and reaches no PDS boundary.
- 2026-09-10: Final verification passed: focused affected suites, required-PostgreSQL serial `go test -p 1 ./... -count=1`, `go vet ./...`, all 2,336 Flutter tests, `flutter analyze`, Dart MCP analysis, and `git diff --check`. Fresh `just appview-check` passed all release gates with artifact `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.ImGJUeKP8m`; the existing accepted `GO-2026-5932` OpenPGP advisory remains reported.
- 2026-09-11: Corrected in-app moderation notification rendering. Notification-list decoding now preserves the public case reference in a first-class actorless moderation model, renders localized account-standing copy, and opens the full moderation history with the same current-account guard used by other rows. Malformed references remain visible but inert. Focused model, widget, destination-inference, and routing tests pass; Dart MCP analysis and live hot restart report no errors.
- 2026-09-11: Removed the redundant focused-case child page after navigation testing showed it stacked a second `AccountStandingPage`. Settings now exposes one moderation route that always loads the complete paginated history; in-app and push notification opens both target it. Route-structure and one-back navigation regression tests pass.
- 2026-09-11: Added owner-only access to the current indexed version of a moderated post. Valid post snapshots expose a localized View post action to the existing thread route; direct post and thread-root reads use the authenticated author fallback only after normal visibility returns not found. Other viewers, indirect surfaces, malformed/mismatched references, deleted posts, and terminal owners retain existing unavailable behavior.
