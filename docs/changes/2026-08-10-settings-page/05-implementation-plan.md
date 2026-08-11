# TDD Implementation Plan: Settings Page And Account Management

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`, High risk)
- Coding plan: `04-coding-plan.md`
- High-risk implementation approval: explicitly granted by the product owner on 2026-08-10.

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before its implementation.
- Run the smallest relevant test first and confirm that its failure is meaningful.
- Refactor only while the focused and nearby tests are green.
- Keep traceability and actual command evidence updated in this file.
- Do not enable a destructive route until the repository guidance/reference amendment is complete and the namespace/account/blob safety regressions pass.
- Do not edit a Lexicon for this feature.
- Never run destructive tests against a personal or production AT/PDS account.
- Do not commit or push unless the user explicitly asks.

## Test Order

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-009, IT-024 | FR-015, RULE-005 | AC-015, AC-030 | No compiled exact collection registry or Lexicon drift gate exists. |
| 2 | UT-001–UT-005, UT-020–UT-023 | FR-001, FR-003, FR-004, FR-008–FR-014, FR-018, NFR-001, NFR-003, NFR-004, RULE-003, RULE-004, RULE-009 | AC-003, AC-010, AC-012, AC-014, AC-019, AC-020, AC-022, AC-024, AC-028, AC-031–AC-036 | Settings identity, row model, About helpers, confirmation policy/copy, and semantics are absent. |
| 3 | REG-001 | BR-002, FR-003 | AC-003, AC-004, AC-005, AC-032 | The adjacent Customisation route is not present in this checkout. |
| 4 | UT-006–UT-008, UT-017, UT-019 | FR-015, FR-017, FR-020, FR-023, FR-025, FR-027, NFR-006, RULE-006, RULE-007, RULE-010 | AC-023, AC-037, AC-039, AC-041, AC-044, AC-046 | No deletion state machine, retry policy, terminal gate, audit projection, or redaction policy exists. |
| 5 | UT-010, UT-011, UT-018, UT-024 | FR-013, FR-015, FR-017, FR-019–FR-021, FR-026, NFR-001, NFR-002, NFR-006, RULE-001, RULE-002, RULE-010 | AC-015, AC-017, AC-023, AC-025, AC-035, AC-040–AC-042, AC-047 | PDS deletion planning, credential separation, explicit Instagram cleanup, and OAuth proof binding are absent. |
| 6 | IT-006, IT-007 | BR-003, FR-013, FR-015, FR-019–FR-021, NFR-001, NFR-002, RULE-001, RULE-002 | AC-015, AC-017, AC-025, AC-026, AC-035, AC-040, AC-041 | OAuth callbacks cannot produce deletion proofs and no owner-scoped acceptance route exists. |
| 7 | IT-008, IT-009, IT-017, IT-021, IT-023 | BR-003, FR-015, FR-020, FR-021, FR-025, NFR-002, NFR-006, RULE-004, RULE-010 | AC-016, AC-023, AC-040–AC-042, AC-046 | Durable minimized storage, bind-before-revoke, restricted status access, exact audit expiry, and coarse telemetry are absent. |
| 8 | IT-010, IT-011, IT-022 | FR-015, FR-020, FR-021, FR-023, NFR-002, RULE-002, RULE-009 | AC-015, AC-017, AC-023, AC-036, AC-037, AC-041, AC-044, AC-048 | No expected-before-delete pagination/recovery loop or blob boundary exists. |
| 9 | IT-012, IT-013 | BR-003, FR-015, FR-026, RULE-010 | AC-016, AC-037, AC-047 | Private owner-data coverage and explicit Instagram deletion are incomplete and lack a drift gate. |
| 10 | IT-016 | FR-017, FR-027, RULE-006 | AC-023, AC-039 | Bounded retry, attention status, and manual Retry are absent. |
| 11 | IT-018, REG-008 | FR-023, RULE-008 | AC-044 | Existing indexers have no post-handler/pre-ack deletion receipt observer. |
| 12 | IT-019 | FR-023, FR-027, RULE-007, RULE-008 | AC-037, AC-044 | Terminal state cannot wait for receipt-backed AppView convergence. |
| 13 | IT-027 | BR-003, FR-015, FR-020, FR-021, FR-023, RULE-007 | AC-015, AC-016, AC-037, AC-040, AC-044 | The complete durable server deletion lifecycle is absent. |
| 14 | IT-001–IT-005, REG-002–REG-007 | BR-001, BR-002, BR-004, FR-001–FR-014, FR-018, NFR-001, NFR-003–NFR-005, RULE-003, RULE-004 | AC-001–AC-014, AC-019–AC-022, AC-024, AC-027–AC-036 | The new Settings/About/Account surfaces and routes are absent. |
| 15 | UT-012–UT-016, IT-014, IT-015, IT-025, IT-026, IT-028 | FR-016, FR-020–FR-022, FR-024, FR-027, NFR-001, NFR-002, NFR-003, NFR-006, RULE-001, RULE-011 | AC-018, AC-025, AC-027, AC-036, AC-038, AC-041–AC-043, AC-045 | Flutter has no separate secure deletion status, cleanup, fallback, or recovery lifecycle. |
| 16 | IT-020, REG-011, REG-012 | FR-019, FR-024, RULE-011 | AC-035, AC-040, AC-045 | Pending deletion login still produces ordinary membership/session behavior and rejoin boundaries are unproved. |
| 17 | REG-009, REG-010 | BR-003, FR-023, RULE-002, RULE-008, RULE-009 | AC-015, AC-017, AC-036, AC-044, AC-048 | Whole-account/blob/namespace and eager-purge prohibitions are not regression-protected. |
| 18 | MAN-001–MAN-004 | BR-003, FR-021–FR-023, NFR-003, NFR-005, RULE-002, RULE-007, RULE-009 | AC-015–AC-017, AC-022, AC-027, AC-029, AC-036, AC-037, AC-040, AC-043, AC-044, AC-048 | Manual release gates remain pending after automated implementation and review. |

## Implementation Steps

### Step 1: UT-009 / IT-024 — CraftSky Record Collection Boundary

- Write failing test: Walk `lexicon/social/craftsky/`, select only schemas whose `defs.main.type` is `record`, and require an exact match with the compiled registry. Verify membership profile is ordered last.
- Run command: `cd appview && go test ./internal/accountdeletion -run 'TestCraftskyRecordCollections'`
- Confirmed failure: `go test ./internal/accountdeletion -run '^TestCraftskyRecordCollections$'` failed to compile because `CraftskyRecordCollections` did not exist. This was the expected missing-behavior failure after rerunning with access to the existing Go build cache.
- Implement: Add only the typed compiled registry needed by the test.
- Run command: `go test ./internal/accountdeletion -run '^TestCraftskyRecordCollections$'` passed.
- Refactor: The exported function returns a copy of a package-owned typed array so callers cannot mutate the safety registry. No unrelated refactor was made.
- Notes: No PDS client, deletion method, OAuth change, route, migration, or Lexicon edit is permitted in this increment.

### Step 2: Guidance Amendment And Non-Destructive Settings Unit Loops

- Begin only after Step 1 is green.
- Amend `AGENTS.md` and `atproto-craft-social-app-reference.md` with the approved narrow exception before any destructive route is registered.
- Execute the tests in Step 2 one at a time in the order named by `04-coding-plan.md` and record each red/green command here.

### Steps 3–18

- Follow the order in the table above and section 9 of `04-coding-plan.md`.
- Add a separate executed-loop subsection for every individual test ID as it becomes active.
- If order changes, record the reason before changing implementation scope.

### Migration Number Adjustment

- The approved coding plan named `000036_account_deletion`, but current `main` now contains `000036_profile_customisation` from the required adjacent work.
- Implementation therefore uses `000037_account_deletion`; table/constraint behavior and test traceability are unchanged.

## Execution Log

| Test ID | Requirement IDs | Red Evidence | Green Evidence | Status |
|---|---|---|---|---|
| UT-009 | FR-015, RULE-005 | Missing `CraftskyRecordCollections` compile failure | Focused test passed | Completed |
| IT-024 | RULE-005 | Exact registry is compared with every primary `record` Lexicon under `lexicon/social/craftsky/` | Focused drift-gate test passed | Completed |
| UT-001 | FR-001, NFR-001 | Missing `settings_identity.dart` and `projectSettingsIdentity` compile failure | `flutter test test/settings/settings_identity_test.dart` passed (3 tests) | Completed |
| UT-002 | FR-003, FR-004, RULE-003 | Missing `settings_row.dart`, canonical inventory, and trailing-icon policy compile failure | `flutter test test/settings/settings_row_model_test.dart` passed (2 tests) | Completed |
| UT-003 | FR-011 | Missing `about_version.dart` and `buildVersionLabel` compile failure | `flutter test test/settings/about_version_test.dart` passed (2 tests) | Completed |
| UT-004 | FR-013 | Missing confirmation model and exact-handle matcher compile failure | `flutter test test/settings/delete_account_confirmation_test.dart` passed | Completed |
| UT-005 | FR-014, RULE-009 | Missing localized `deleteAccountBoundary` method compile failure | `flutter test test/settings/delete_account_copy_test.dart` passed after `flutter gen-l10n` | Completed |
| UT-020 | FR-008, FR-009, FR-010 | Missing canonical Settings links and safe launcher wrapper compile failure | `flutter test test/settings/about_action_error_test.dart` passed (2 tests) | Completed |
| UT-021 | FR-018, RULE-004 | Missing distinct Settings action-policy model compile failure | `flutter test test/settings/settings_action_policy_test.dart` passed | Completed |
| UT-022 | NFR-004 | Generated localizations lacked the complete Settings/About/Account/deletion string surface | `flutter test test/settings/settings_localizations_test.dart` passed after `flutter gen-l10n` | Completed |
| UT-023 | FR-004, NFR-003 | Missing reusable semantic Settings row widget compile failure | `flutter test test/settings/settings_accessibility_test.dart` passed (3 tests) | Completed |
| REG-001 | BR-002, FR-003 | This checkout initially lacked the adjacent Customisation implementation | Current `main` aligned; focused Settings/route suite passed (10 tests), including Customisation row/location guards | Completed |
| UT-006 | FR-017, FR-020, FR-027, RULE-006 | Missing deletion states, events, transition function, and point-of-no-return policy compile failure | `go test ./internal/accountdeletion -run '^TestDeletionStateMachine$'` passed | Completed |
| UT-007 | FR-017, FR-021, FR-027, NFR-002 | Missing retry schedule, failure classification, deterministic jitter, and exhaustion policy compile failure | `go test ./internal/accountdeletion -run '^TestRetryPolicy$'` passed | Completed |
| UT-008 | FR-015, FR-020, FR-023, RULE-007 | Missing explicit terminal-success gate contract compile failure | `go test ./internal/accountdeletion -run '^TestTerminalEligibility$'` passed, including blob-GC exclusion | Completed |
| UT-017 | FR-025, NFR-006, RULE-010 | Missing minimized audit projection, exact expiry predicate, and rejoin policy compile failure | `go test ./internal/accountdeletion -run '^TestDeletionAuditProjectionAndExpiry$'` passed | Completed |
| UT-019 | FR-025, NFR-006, RULE-010 | Missing operational allow-list, terminal projection, and coarse telemetry contract compile failure | `go test ./internal/accountdeletion -run '^TestOperationalMinimizationAndTelemetryRedaction$'` passed | Completed |
| UT-010 | FR-015, NFR-002, RULE-002 | Missing deletion-only PDS capability and owner/namespace-closed convergent scan compile failure | `go test ./internal/accountdeletion -run '^TestPDSDeleterIsOwnerScopedAndNamespaceClosed$'` passed | Completed |
| UT-011 | FR-021, NFR-006 | Missing status action authorization, redacted status projection, and worker-only bound OAuth access compile failure | `go test ./internal/accountdeletion -run '^(TestStatusCapabilityAuthorizationIsNarrow|TestBoundOAuthSessionIsWorkerOnly)$'` passed | Completed |
| UT-018 | FR-026, RULE-010 | Missing explicit-deletion Instagram hard-delete plan and PurgeOwner-only capability compile failure | `go test ./internal/accountdeletion -run '^TestInstagramExplicitDeletionPlanAlwaysHardDeletes$'` passed | Completed |
| UT-024 | FR-013, FR-019, FR-020, FR-021, NFR-001, RULE-001 | Missing one-way fresh proof, exact owner/handle/session binding, replay protection, and idempotent/replacement binding compile failure | Focused `internal/auth` reauthentication and `internal/accountdeletion` OAuth-binding tests passed | Completed |
| IT-006 | FR-013, FR-019, FR-020, FR-021, NFR-001, RULE-001 | OAuth callback always entered ordinary profile/session initialization; no pre-consumption deletion purpose branch existed | `go test ./internal/auth -run '^TestOAuthCallbackUsesDeletionOnlyPurposeWithoutMintingOrdinaryAccess$'` passed; database bind-before-revoke ordering remains mapped to IT-009 | Completed |
| IT-007 | BR-003, FR-015, NFR-002, RULE-002 | Missing owner-derived deletion intent/accept policies, strict DTO, handler, and canonical errors compile failure | `go test ./internal/routes -run '^TestAccountDeletionAcceptanceRouteIsAuthenticatedOwnerScopedAndStrict$'` passed | Completed |
| IT-008 | FR-020, NFR-002, NFR-006, RULE-010 | Missing migration/store API, durable operation, restart load, duplicate acceptance, and terminal cleanup compile/file failures | Migration contract and `TestStoreDurableAcceptanceAndTerminalMinimization` passed | Completed |
| IT-009 | FR-015, FR-020, FR-021, RULE-004 | No transaction bound the fresh OAuth row before recovery-hash capture, subscription removal, bearer deletion, and unbound OAuth deletion | Durable store test passed with Alice/Bob controls and FK-protected bound session | Completed |
| IT-017 | FR-021, NFR-006 | Missing signed/hash-backed capability, dedicated status middleware, and status/Retry/reauth route policies compile failure | Signed capability and restricted status-route tests passed | Completed |
| IT-021 | FR-025, NFR-006, RULE-010 | No terminal transaction removed operational/OAuth/status/URI/receipt state and inserted the exact 30-day minimal audit | Durable store test and migration closed-column test passed | Completed |
| IT-023 | NFR-006 | Missing deletion lifecycle logger/metric adapter compile failure | `go test ./internal/accountdeletion -run '^TestDeletionTelemetryEmitsOnlyCoarseLifecycleSignals$'` passed with coarse-only fields | Completed |
| IT-010 | FR-015, FR-023, RULE-002 | Store did not implement durable expected-URI registration for the narrow PDS deleter | Disposable-Postgres `TestPDSDeleterPersistsExpectedRecordsAndPreservesOtherData` passed; other namespace/blob controls preserved | Completed |
| IT-011 | FR-015, FR-020, FR-021, FR-023, NFR-002 | Disposable-Postgres test showed an uncertain post-side-effect failure left `delete_requested_at` unset | After marker implementation, `TestPDSDeletionRestartConvergesAfterUncertainSideEffect` passed against disposable Postgres | Completed |
| IT-022 | RULE-009 | Cross-test blob/account boundary needed an explicit regression target | `go test ./internal/accountdeletion -run '^TestBlobBoundaryIsReferenceOnlyAndNeverTerminalGate$'` passed | Completed |
| IT-012 | BR-003, FR-015, FR-026, RULE-010 | No executable private-owner cleanup manifest or checkpointed cleanup service existed | Disposable-Postgres private-manifest and owner-isolation tests passed in the full AppView suite | Completed |
| IT-013 | FR-026, RULE-010 | Explicit deletion had no Instagram hard-delete override | Instagram deletion-plan test and existing Instagram retention suites passed | Completed |
| IT-016 | FR-017, FR-027, RULE-006 | No worker retry exhaustion/manual-resume integration existed | `TestWorkerUsesBoundedRetryThenManualRetryResetsSameJob` and deletion-status widget test passed | Completed |
| IT-018 | FR-023, RULE-008 | Dispatcher had no post-handler/pre-ack deletion receipt observer | Dispatcher ordering/error tests and receipt-observer tests passed | Completed |
| REG-008 | FR-023, RULE-008 | Receipt wiring could have changed existing indexer replay behavior | Full `internal/index` suite passed, including existing dispatcher/indexer behavior | Completed |
| IT-019 | FR-023, FR-027, RULE-007, RULE-008 | Terminal processing did not wait for receipts plus absent/retracted indexed effects | Convergence and durable lifecycle tests passed against disposable Postgres | Completed |
| IT-027 | BR-003, FR-015, FR-020, FR-021, FR-023, RULE-007 | No complete restartable lifecycle joined acceptance, cleanup, PDS deletion, convergence, OAuth removal, and audit projection | Durable lifecycle, store, worker-failure, PDS, convergence, and app-service suites passed together; MAN-003 remains the real-stack gate | Completed (automated) |
| IT-001 | BR-001, BR-002, BR-004, FR-001, FR-003, FR-004, NFR-003–NFR-005 | Expanded production Settings surface was absent | Settings widget/identity/row/accessibility suites passed | Completed |
| IT-002 | BR-002, FR-003, FR-005–FR-007, FR-012, NFR-005 | New/existing Settings destinations were not wired through canonical routes | Settings route, notification route, Back-selection, and typed-navigation policy tests passed | Completed |
| IT-003 | BR-001, FR-002, FR-005, NFR-001 | Settings had no entry to the shared responsive switcher | Account switcher, account-routing, lease, and MRU behavior remained green | Completed |
| IT-004 | BR-004, FR-004, FR-007–FR-011, NFR-003 | About page and moved cache action were absent | About, external-link failure, version, cache, and semantics tests passed | Completed |
| IT-005 | BR-004, FR-012–FR-014, NFR-003, NFR-004, RULE-003 | Account page and destructive warning/exact-handle confirmation were absent | Account widget, exact-handle, copy, lease, and fresh-reauth server tests passed | Completed |
| REG-002 | BR-002, FR-006 | Notifications could have been duplicated or behaviorally replaced | Existing notification settings route/page coverage passed in the full Flutter run until unrelated baseline failures | Completed |
| REG-003 | BR-004, FR-010 | Moving cache clearing could have changed its two-cache behavior | Existing clear-image-cache suite passed (10 tests in the focused settings run) | Completed |
| REG-004 | FR-018, RULE-004 | Error styling could have coupled Sign out to deletion | Sign-out suite and Settings action-policy tests passed | Completed |
| REG-005 | FR-002 | Shared switcher limits, MRU, guard, and presentation could regress | Existing account switcher/model/routing coverage passed | Completed |
| REG-006 | FR-011 | About could drift from shell build-version formatting | Shared version formatter tests passed | Completed |
| REG-007 | FR-008, FR-009 | Legal-link failures could navigate away or expose errors | About and shell safe-launch tests passed | Completed |
| UT-012 | FR-016, FR-022 | No acceptance transition for ordinary versus status-only registries existed | Session-registry deletion tests passed for MRU fallback, status-primary, and terminal removal | Completed |
| UT-013 | FR-016, FR-027, NFR-003 | Switcher had no disabled deletion-status row projection | Account-switcher state/widget coverage passed for pending, attention, and terminal states | Completed |
| UT-014 | FR-022, NFR-006 | No explicit device cleanup/preservation plan existed | Local-cleanup unit test passed | Completed |
| UT-015 | NFR-001, RULE-001 | Late deletion work was not fenced to the captured account lease | Lease and controller tests passed, including late Alice completion after Bob activation | Completed |
| UT-016 | FR-024, RULE-011 | Post-login destination did not account for pending or completed deletion | Deletion-login policy tests passed | Completed |
| IT-014 | FR-016, FR-022 | Client acceptance did not persist restricted status before cleanup/fallback | Controller coordinator test passed with exact effect ordering and Bob fallback | Completed |
| IT-015 | FR-016, FR-021, FR-022 | No-last-account acceptance had no status-only primary path | Session-registry, controller, status-page, and router redirect coverage passed | Completed |
| IT-025 | FR-022 | A former bearer received generic 401 invalidation rather than next-contact status recovery | Former-bearer recovery, pending-deletion interceptor, and cleanup coordinator tests passed; two-device offline timing remains MAN-004 | Completed (automated boundary) |
| IT-026 | NFR-001, RULE-001 | Stale Alice acceptance could affect active Bob | Controller and lease tests passed with Bob preserved and no active-account navigation | Completed |
| IT-028 | FR-020, NFR-002 | An uncertain accepted response could create a second local job or restore ordinary access | Client same-job resolution plus one-time former-bearer server recovery/cancellation tests passed | Completed |
| IT-020 | FR-019, FR-024, RULE-011 | Ordinary OAuth login could initialize membership while deletion was pending | OAuth pending-login test passed: new OAuth session is rejected and status-only handoff occurs before membership initialization | Completed |
| REG-011 | FR-019 | Client deletion DTO/storage could accidentally receive PDS authority | Status-only client/header and recovery projection tests passed; client models expose no PDS token fields | Completed |
| REG-012 | FR-024, RULE-011 | Retained audit/pending state could block or restore a fresh membership | Login-policy, pending-login, audit-rejoin policy, and terminal minimization tests passed | Completed (automated) |
| REG-009 | BR-003, RULE-002, RULE-009 | Namespace/account/blob prohibitions lacked executable controls | PDS deleter and blob-boundary fakes passed; no whole-account, other-namespace, or blob delete API is used | Completed (automated) |
| REG-010 | FR-023, RULE-008 | New deletion logic could eagerly hide AppView data | Receipt/convergence tests pass with public removal owned by existing indexers only | Completed |

## Verification Summary

- `cd app && dart analyze` — passed with `No issues found!`.
- Focused Flutter verification — 105 tests passed across Settings, deletion models/storage/controllers, routing, account switcher, 401 recovery, and Dio provider coverage.
- Full Flutter attempt — 1,459 tests passed and two failed. The two remaining failures are unchanged `auth_complete_page_test.dart` harness failures: the direct `MaterialApp` fixture supplies no `GoRouter` even though the unchanged production page navigates to `FeedRoute` after successful completion. They were not changed as part of this feature.
- `cd appview && TEST_DATABASE_URL=... go test ./...` — every AppView package passed, including disposable-Postgres account-deletion coverage.
- `git diff --check` — passed after implementation.
- No destructive test was run against a personal, production, or real PDS account.
- MAN-001 through MAN-004 remain explicit release gates; MAN-003 must use only a disposable development AT/PDS account.

## 2026-08-11 Implementation Review Correction Pass

The implementation review in `06-implementation-review.md` returned `Changes required`. The product owner selected **Address required changes**, explicitly authorizing this high-risk auth/privacy/destructive-data correction pass. The following loops supersede the affected prior completion claims until each new focused test and its broader regression evidence are green.

### Correction Test Order

| Order | Review Finding / Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IR-001 / IT-006 | FR-013, FR-017, FR-020 | AC-013, AC-023, AC-035 | Abandoned or expired owner-unique intents block a later attempt and leave OAuth/status residue. |
| 2 | IR-002 / IT-018 | FR-023, FR-027, RULE-007, RULE-008 | AC-037, AC-044 | Receipt-observer errors count toward poison-drop and can be acknowledged without a receipt. |
| 3 | IR-003 / IT-011, IT-018, IT-019 | FR-020, FR-023, RULE-007 | AC-037, AC-044 | A matching delete observed before expected-record registration is acknowledged and lost. |
| 4 | IR-004 / IT-011 | FR-015, FR-020, FR-023, NFR-002 | AC-023, AC-037, AC-044 | A successful PDS delete followed by marker persistence failure leaves an unrecoverable null marker. |
| 5 | IR-005 / IT-012 | BR-003, FR-015, FR-026, RULE-010 | AC-016, AC-037, AC-047 | The manifest test classifies schema surfaces but does not execute every delete/retain/orphan policy with Alice/Bob/shared controls. |
| 6 | IR-005 / IT-027 | BR-003, FR-015, FR-020, FR-021, FR-023, RULE-007 | AC-015, AC-016, AC-037, AC-040, AC-044 | Component tests do not yet provide the specified full lifecycle fixture, restart, reordered event, and preservation controls. |
| 7 | IR-006 / UT-013, IT-014, IT-016 | FR-016, FR-017, FR-027 | AC-018, AC-038, AC-039 | Status refresh runs only on the status page; a switcher row and app resume can remain stale. |
| 8 | IR-007 / IT-023 | NFR-006 | AC-046 | The redacted telemetry adapter is not invoked by real worker/lifecycle/audit paths. |
| 9 | IR-008 / UT-024, IT-006 | FR-019, FR-020, FR-021, NFR-001, RULE-001 | AC-035, AC-040, AC-041 | Auth requests are first stored as login requests and rebound in a second SQL update, with incomplete failure cleanup. |

### Correction Loop Rules

- Execute one row at a time in the order above: focused red test, minimum implementation, focused green, nearby regressions, then update this execution log.
- Do not treat an existing component test as the planned acceptance test when its fixture omits a stated barrier, owner/shared control, or terminal gate.
- Preserve the owner/DID/`social.craftsky.*`/blob boundaries and the rule that public AppView deletion remains indexer-owned.
- No destructive real-PDS test is authorized by this correction pass; MAN-003 remains a later disposable-development-account release gate.

### Correction Execution Log

| Review Finding / Test ID | Requirement IDs | Red Evidence | Green Evidence | Status |
|---|---|---|---|---|
| IR-001 / IT-006 | FR-013, FR-017, FR-020 | `TestAppServiceReplacesExpiredIntentForSameOwner` failed on the owner-unique operation constraint; the Flutter Back test failed to compile because the completion page had no cancellation boundary | Expired intent replacement passed against disposable Postgres; the completion-page Back test and nearby Account/controller/lease tests passed | Completed |
| IR-002 / IT-018 | FR-023, FR-027, RULE-007, RULE-008 | Consumer test failed to compile because `tap.ErrMustReplay` did not exist; ordinary errors were the only poison classification | Must-replay redelivery test and existing ordinary poison-drop test passed together; dispatcher test proves receipt failures carry the classification while retaining the original error | Completed |
| IR-003 / IT-011, IT-018, IT-019 | FR-020, FR-023, RULE-007 | Delete-before-registration test found zero receipts before the expected row existed | Observer now stores owner/job-scoped CraftSky delete observations for the active job; late registration plus delete marking converges, and duplicate/extra observation regressions pass | Completed |
| IR-004 / IT-011 | FR-015, FR-020, FR-023, NFR-002 | Failure injection proved `DeleteRecord` ran and removed the record before `delete_requested_at` persistence succeeded | PDS deletion now begins only after the durable request marker; marker-failure retry and uncertain post-side-effect restart tests pass | Completed |
| IR-005 / IT-012 | BR-003, FR-015, FR-026, RULE-010 | The production-component contract test failed because the manifest named service labels rather than executable cleaner component names; the full Alice/Bob/shared fixture then exposed an Alice-only push installation surviving its last subscription deletion | The manifest now carries ownership, executable component, and executable verification metadata; schema queries, production-component coverage, actual database/Instagram/scheduled cleanup, replay, orphan/shared controls, and queued object deletion pass against disposable Postgres | Completed |
| IR-005 / IT-027 | BR-003, FR-015, FR-020, FR-021, FR-023, RULE-007 | Strengthening durable acceptance with installation controls proved the acceptance transaction left an owner-only installation behind; the specified full worker fixture was absent | `TestWorkerAcceptanceRunsCompleteDeletionLifecycleAcrossRestart` now passes with real acceptance/bind-before-revoke, production private components, a worker/store restart, paged multi-collection PDS deletion, scheduled-object completion, reversed plus duplicate post-indexer receipts, final rescan/minimization, and Bob/shared/non-CraftSky controls | Completed |
| IR-006 / UT-013, IT-014, IT-016 | FR-016, FR-017, FR-027 | The new switcher/resume tests failed to compile because no registry-level refresh host existed; status polling belonged only to the mounted status page | The app-level refresh host uses status-only clients, bounded 2s/5s/15s/30s backoff, overlap prevention, and resume refresh; visible-row terminal removal, attention-stop, status-page, and shell switcher regressions pass | Completed |
| IR-007 / IT-023 | NFR-006 | Production-path tests failed to compile because store/worker/lifecycle/audit types had no telemetry injection points and production dependencies never constructed the adapter | One coarse telemetry instance is now wired through production acceptance, phase, automatic/manual retry, attention, convergence delay, terminal-success, and audit-expiry paths; real transition tests, metric-sink attributes, and prohibited-data redaction pass | Completed |
| IR-008 / UT-024, IT-006 | FR-019, FR-020, FR-021, NFR-001, RULE-001 | The atomic-store test failed to compile because no authenticated request-metadata context existed; the AppService residue test then proved the second update could upgrade an ordinary login request and let deletion reauthentication proceed | `SaveAuthRequestInfo` now inserts deletion purpose, owner, and job metadata atomically; initial and replacement AppService flows verify that exact persisted binding and delete rejected residue, while ordinary login remains login-purpose; full `internal/auth` and `internal/accountdeletion` package tests pass | Completed |

### Correction Verification

- `cd appview && TEST_DATABASE_URL=... go test ./... -count=1` — every AppView package passed after all eight corrections.
- Focused Flutter correction regression — 93 tests passed across Settings, deletion status refresh, account switching, routing, storage, login policy, and former-bearer recovery.
- `cd app && flutter test` — 1,459 tests passed and the two unchanged `auth_complete_page_test.dart` fixture tests failed because their direct `MaterialApp` contains no `GoRouter`; neither the production page nor this test file was modified.
- `cd app && dart analyze` — passed with `No issues found!` after correcting the refresh-host lints.
- Go formatting and `git diff --check` — passed.
- No destructive test was run against a personal, production, or real PDS account.

### 2026-08-10 — Mandatory Guidance Gate

- Amended `AGENTS.md` and `atproto-craft-social-app-reference.md` after UT-009/IT-024 passed.
- Preserved the prohibitions on whole-account/DID deletion, other namespaces, direct blob deletion, Flutter-held PDS credentials, and a reusable generic PDS-delete API.
- No destructive route, PDS deleter, OAuth mutation, or migration exists at this point.

## Completion Checklist

- [x] All Must requirements covered by passing tests or documented gaps.
- [ ] All planned Must tests passing (manual gates and two unchanged full-suite harness failures remain).
- [x] Relevant task-focused regression tests passing.
- [x] No unlinked behavior implemented.
- [x] Repository guidance/reference amendment completed before destructive route enablement.
- [x] `05-implementation-plan.md` updated with actual red/green evidence.
- [x] Full Flutter and AppView verification completed; the two unchanged Flutter harness failures are documented above.
- [x] Manual release gates recorded without claiming they ran.
- [ ] Implementation review completed or explicitly skipped.
