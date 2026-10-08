# TDD Implementation Plan: Useful Logging and Error Reporting

## Flutter console visibility 2026-10-08

CON-001 authorized 2026-10-08. CON-T01 completed: meaningful red for missing INFO (`console-t01-red.log`), then green for default INFO/WARNING, FINE suppression and sanitized debug opt-in (`console-t01-green.log`); provider-value changes removed from scope by explicit maintainer preference; Final focused console/native logging suites: seven passed (`console-final-focused.log`). Full Flutter suite: 2,710 passed, 38 skipped, two obsolete WARN-only bridge assertions failed (`console-flutter-full.log`); both were updated for INFO+ and passed in the focused rerun, retaining independent Sentry gate/canary checks. Analyzer clean (`console-flutter-analyze-final.log`). Guide and validation evidence updated; final diff check passed and plan read back. Implementation complete, review pending stage choice. Full suite was not rerun after assertion-only updates; no production/device verification claimed. No Go/business/Sentry capture policy changes; no commit/push.

## Repository retry escalation correction 2026-10-08

ERR-003 amendment approved 2026-10-08: escalation at existing attempt threshold while recovery continues. ERR-T03 completed: behavioral red for expired/mixed/source-change ERROR and missing explanation (`retry-t03-red.log`), then green (`retry-t03-green.log`). ERR-T04 completed: real migrated worker red at threshold with zero issues (`retry-t04-red.log`), then green for WARN/no issue followed by ERROR/one issue on each subsequent failed attempt (`retry-t04-green.log`). Serialized local/native output retains public owner, operation and original exception type; private prose absent and durable pending retry state preserved. Full `just test` Go 1.27.1 race verification against compose PostgreSQL/MinIO passed (`retry-go-full.log`), including final native source-explanation/canary assertions. Guide, requirements, acceptance tests, coding plan and investigation updated; final diff check passed and plan read back. No Flutter source changed, so prior Flutter verification is retained. Implementation complete; review awaits stage choice. No commit, push or runtime restart authorized.

## Approved completion metadata correction 2026-10-07

| Step | Test / requirement | Status |
|---|---|---|
| R1 | REL-T01 / REL-001: follower completion outcome across sinks | completed |
| R2 | REL-T02 / REL-001: known release and absent release across sinks | completed |
| R3 | Verification, guide/evidence, plan readback | completed |

R1 confirmed behavioral red for `already_complete` becoming `unknown` in local/native SDK output (`rel-t01-red.log`); same test green after admitting the existing legitimate completion outcome (`rel-t01-green.log`). R2 confirmed red for embedded release becoming `[OMITTED]` and empty release being emitted (`rel-t02-red.log`); same test green after shared release selection and omission handling (`rel-t02-green.log`). Existing technical release names remain accepted; only the known embedded AppView format adds `@` support. Private release/outcome canaries remain excluded. No metrics/business/release policy changes. Focused observability/follower-growth/config/startup race suites passed (`rel-focused-final.log`). Full `just test` PostgreSQL/MinIO race verification finished; every package except API passed (`rel-go-full.log`). API hit `TestScheduledImageReleaseConcurrentUploads`' five-second admission timeout; isolated rerun with the same Go 1.27.1 race toolchain passed (`rel-image-recheck.log`). No image code/test change was made. Guide/evidence updated, plan read back and `git diff --check` passed. Full-run timing failure is documented, not claimed as a clean full-suite pass. Implementation review remains pending the stage choice; no Flutter changes or commit/push/restart/deploy.

## Dev error follow-up 2026-10-07

Approved ERR-001/002 scope is recorded in 01/02/04. Using implement-tdd for this continuation.

| Step | Test / requirement | Status |
|---|---|---|
| E1 | ERR-T01 / ERR-001: typed validation details, WARN/no issue, unchanged quarantine | completed |
| E2 | ERR-T02 / ERR-002: per-job cause/context and conservative lease classification | completed |
| E3 | Full Go verification, guide/evidence and plan readback | completed |

E1 confirmed behavioral red: dispatcher rejection emitted ERROR, generic cause, a public-text excerpt and one issue (`err-t01-red.log`). Green: typed validator details survive transient outcome data; WARN/no issue and no body (`err-t01-green.log`). Native SDK Logs regression separately confirmed red for generic remote body (`err-remote-red.log`) and green with safe missing-field explanation (`err-remote-green.log`). Oversized-blob regression retains original cause and numeric size. Initial new durable reason proved incompatible with the database quarantine constraint; no migration was added. Durable `malformed_record` and source `invalid_lexicon` remain unchanged; diagnostic reason is `invalid_lexicon`. Real projection-worker regression confirms quarantine commit and transient cause retention.

E2 confirmed red: RunOnce logged no per-job cause/context (`err-t02-red.log`). Green: public owner, job operation/kind, attempt and typed joined causes retained; read-only lease inspection distinguishes supersession/expiry without changing conditional fencing. Mixed errors remain ERROR; a further red demonstrated repeated supersession stayed WARN, now ERROR at the configured attempt threshold (`err-durability-red.log`). Batch-only duplicate logging removed; original returned causes still match errors.Is. No queue/lease duration/reconciliation/PDS policy changes.

Affected Go race suites passed (`err-focused-final.log`), followed by native SDK Log explanation verification (`err-remote-green.log`). Full `just test` passed (`err-go-full.log`), followed by final observability/Tap/schema race checks (`err-final-diagnostics.log`) and both missing-field/oversized-business-blob native WARN Log cases (`err-lexicon-last.log`). Exhausted validation guardrail confirmed red for WARN (`err-terminal-red.log`) then green after limiting the downgrade to quarantine. Guide/evidence updated, implementation plan read back and `git diff --check` passed. Implementation review remains pending the stage choice. A Sentry import-boundary failure was fixed by moving the mock SDK test into the already-approved Tap test package, without broadening the boundary. No Flutter changes; no commit/push/restart/deploy.

## Dev diagnostic usability correction 2026-10-07

Maintainer authorized removal of the stale dev Sentry release override and the noisy unsupported-field marker, retaining reviewed operational metadata. Existing SDK-003/005/006, SIM-005/006 and NFR-002 cover the source selection and bounds. Error investigation is read-only; business/schema/record fixes are recommendations only. No commit/push/deploy requested for this pass.

| Step | Test / requirement | Status |
|---|---|---|
| D1 | SDK-T05: real protected startup log retains listener/version metadata | completed |
| D2 | SDK-T05: supported Tap envelope metadata retained; unsupported/private fields quietly excluded | completed |
| D3 | SDK-T06: actual budget limits still mark truncation; dev release follows embedded version | completed |
| D4 | Go focused/full verification, guide/evidence, error investigation and plan readback | completed |

D1 red: protected startup test lost `addr` (`devdiag-startup-red.log`); same test green after reviewed literal listener-address and semantic-version selection (`devdiag-startup-green.log`). D2 red: Tap ID/action/recordBytes missing and unsupported fields add marker (`devdiag-metadata-red.log`); same test green with typed public envelope metadata retained and exclusions quiet (`devdiag-metadata-green.log`). Actual logs: 568 of 587 had the marker, including 512 record-event DEBUG messages.

D3 verification: stale dev release override cleared (embedded release precedence remains covered by existing config tests); actual count/byte budget markers retained with correlation and private-frame canaries. An initial byte-budget fixture used an invalid long release and was sanitized before it reached the budget; corrected fixture uses a real large typed panic stack. Final diagnostics/startup/config race suite passed (`devdiag-focused-final.log`). Full `just test` PostgreSQL/MinIO race suite passed (`devdiag-go-full.log`). Guide, validation evidence and read-only dev error investigation updated; implementation plan read back and `git diff --check` passed. Temporary validation probe removed. Implementation review remains pending the stage choice. Running dev container has not been rebuilt; changes remain uncommitted.

## IR-007 correction 2026-10-07

User selected the required-correction stage. Scope: SDK-T04 / SDK-001, SDK-005, SIM-002, FR-008 / AC-009; preserve SDK ownership and explicit terminal/reportability policy. Optional IR-008 is outside this correction. No commit authorized.

| Step | Test / requirements | Status |
|---|---|---|
| C1 | SDK-T04 / AC-009: actual SDK framework/platform/zone expected-outcome matrix | completed |
| C2 | SDK-T04 / AC-009: explicit terminal and cancellation compatibility | completed |
| C3 | SDK-T06: Flutter regression suite, analysis, evidence and plan readback | completed |

Confirmed red: `/private/tmp/ir007-red.log` exports eight issues instead of two. Green: all four SDK-T04 handler tests pass (`/private/tmp/ir007-green.log`), including framework/platform/zone expected outcomes, both reportability overrides, original cause/stack, protected canary and local output. Implemented the existing classifier at the SDK automatic boundary using the original typed throwable; native SDK ThrowableMechanism metadata distinguishes automatic from explicit capture without app ownership flags. Cause/stack selection and local callbacks are unchanged. C2 verifies explicit terminal network/expiry export with request correlation, plus direct/wrapped cancellation suppression even with terminal/reportable overrides. Focused diagnostics/provider/error suites: 99 passed, 1 skipped (`/private/tmp/ir007-focused.log`). Analyzer: No issues found (`/private/tmp/ir007-analyze.log`). Full Flutter verification: 2,711 passed, 38 skipped (`/private/tmp/ir007-full.log`). `git diff --check` passed. Go source/tests are unchanged in this correction; the previous full Go evidence still applies. Guide, acceptance-test detail and validation evidence updated; implementation plan read back. IR-007 implementation is complete, with implementation review pending the stage choice. Device/live symbolication remains separately pending. No commit, push or deployment.

## Sentry-led execution 2026-10-07

Inputs: approved SDK amendment in 01–04. User authorized implementation including described privacy refinement; no stage commit authorized for this new pass.

| Step | Test / requirements | Status |
|---|---|---|
| 1 | SDK-T01 / SDK-002, SDK-003 | completed |
| 2 | SDK-T02 / SDK-002, SDK-003 | completed |
| 3 | SDK-T03 / SDK-004 | completed |
| 4 | SDK-T04 / SDK-001 | completed |
| 5 | SDK-T05 / SDK-005 | completed |
| 6 | SDK-T06 / SDK-006 | completed |

Strict loop: one behavioral failing test, confirm meaningful red, minimum code, same test green, then refactor/nearby coverage. Finish with full suites/analyzer, diff review, guide/evidence update and plan readback. Native/device/production retrieval remains pending; no deployment or push.


### Executed SDK loops

Each red below was a behavioral failure, followed by the same focused test passing. Temporary transcripts are under `/private/tmp`; the permanent source tests are the durable evidence.

| ID | Confirmed red | Minimum implementation and green evidence |
|---|---|---|
| SDK-T01 | Flutter wrappers emitted one flattened exception; reviewed video transition explanation was generic | Supported AppError/API/Dio cause extractors; native throwable/stack privacy selection; `DiagnosticStateError` preserves StateError catches. `sdk-t01-red.log`, `sdk-t01-message-red.log` → `sdk-t01-all-green.log`. |
| SDK-T02 | Go discarded attached origin stacks/native mechanisms and reviewed wrapper explanation | SDK SetException for bounded normal graphs; preserve attached origin stacks, remove fabricated capture-time stacks; reviewed static WrapError. `sdk-t02-red.log`, `sdk-t02-message-red.log` → `sdk-t02-all-green.log`. Panic recovery exceptions remain separate after a focused regression caught replacement of their stack. |
| SDK-T03 | Safe connectivity/navigation and Go operation/HTTP history were removed | Final hooks retain selected SDK categories/scalars; navigation observers wired to root, authenticated shell and all five branches, arguments removed before formatting, transactions disabled. `sdk-t03-flutter-red.log`, `sdk-t03-go-red.log` → corresponding green logs. Actual native observer route-pattern and hyphenated-name tests also confirmed red (`sdk-navigation-red.log`, `sdk-navigation-names-red.log`) before selection was added. |
| SDK-T04 | Local callbacks still replaced SDK callbacks/owned duplicate explicit capture | Local-only callbacks installed before Sentry initialization; SDK appRunner owns enabled automatic capture, consumed startup errors explicit; local zone fallback only without available SDK. Actual SentryFlutter integrations serialize exactly two distinct framework/platform issues with mechanisms/stacks. `sdk-t04-red.log` → `sdk-t04-green.log`. |
| SDK-T05 | Official logging produced no native SDK Logs/history; request timelines were missing without tracing | Matching-version sentry_logging and sentry-go/slog integrations; Dart log issue threshold OFF, console independent; exception-only reporter and removal of custom LogForwarder/duplicate context cause fields. Request hub clones isolate history without tracing. `sdk-t05-flutter-red.log`, `sdk-t05-go-red.log`, `sdk-t05-isolation-red.log` → corresponding green logs. |
| SDK-T06 | Frame sanitization removed symbolication metadata; SDK formatter failure lost typed local fallback; cyclic Go graphs lost the issue | Retain native image/symbol/package/platform addresses while stripping locals/source; typed protected fallback; bounded fallback exceptions plus available attached stacks. `sdk-t06-frames-red.log`, `sdk-t06-fallback-red.log`, `sdk-cycle-red.log` → passing focused regressions. Full verification recorded in 06-validation-evidence. |

Refactor while green: migrate reviewed literal-only StateError constructions across feature boundaries to the subtype (no input/interpolation, same catch semantics); remove obsolete reporter fake APIs and JSON-summary assertions; regenerate affected providers and router observer wiring, restore unrelated generation drift. No authorization, purchase, account lifecycle, retry, ACK, API or lexicon policy changes.

Broad-run corrections: the previous manual navigation-message expectation now verifies the fixed safe native message. A scheduled object-open retry intentionally creates no issue; its log now retains scalar error.type and vetted HTTP status/SQLSTATE, with no duplicate chain/stack JSON. The existing permanent media test retains positive type/status, private canaries and unchanged retry/attention policy assertions.

Finalization: full Go PostgreSQL/MinIO race suite, full Flutter suite, analyzer, diff review and contributor guide/evidence update. Independent implementation review is pending this stage's exit gate. Final SDK-T06 refinements: native Go status/byte scalar test confirmed meaningful red (`sdk-native-scalars-red.log`) before retaining numeric attributes at the final hook; HTTP int64 status normalization preserves the existing safe status contract. Flutter stack snapshot/language retention confirmed red (`sdk-stack-snapshot-red.log`) and green with the final 19-test SDK suite (`sdk-flutter-last-focused.log`). Native/device/live release symbolication remains pending; no commit, push or deployment.


### SDK completion checklist

- [x] SDK-T01–SDK-T06 implemented with confirmed behavioral red/green loops and permanent regressions.
- [x] Full Flutter suite: 2,707 passed, 38 skipped; final stack metadata focused suite: 19 passed.
- [x] Full Go `just test`: PostgreSQL/MinIO, race detector, all packages passed (`/private/tmp/sdk-go-all-green.log`). Includes final retry cause/type/status and numeric log scalar corrections.
- [x] Flutter analyzer: No issues found (`/private/tmp/sdk-analyze-last.log`).
- [x] Source/diff/privacy check and `git diff --check`; all changes map to SDK amendment, static explanation selection, SDK wiring or affected test/fake/generation cleanup.
- [x] Guide and validation evidence updated; implementation plan read back.
- [x] Independent implementation review explicitly pending; physical device/live symbolication validation deferred, not claimed.
- [x] No new commit/push/deploy authorized or performed.


## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Review: `03-document-review.md` (Approved with notes)
- Coding plan: `04-coding-plan.md`
- Implementation authorized by the user selecting implement-tdd on 2026-10-05. No commit, push, production inspection or mutation authorized.

## Implementation Rules
- Link each behavior to its requirement and test ID.
- One focused failing behavior at a time, then minimum implementation and green verification.
- Refactor only while green; preserve business behavior.
- Follow source-aware selection before sanitization and bounds; never stringify unknown errors wholesale.
- Device/production checks remain Pending until actual evidence exists.

## Progress
- [x] Read workflow documents.
- [x] Create implementation plan.
- [x] Inspect relevant code and tests.
- [x] Run final verification (device gap documented).
- [x] Update and read back implementation notes.

## Test Order
Mirror coding-plan section 9; each ID remains a separate loop. Both runtime cases are required before a cross-runtime ID is completed.

| Step | Test ID | Requirement IDs / Acceptance Criteria | Status |
|---|---|---|---|
| 1 | UT-003 | FR-001, RULE-003 / AC-001, AC-021 | completed |
| 2 | UT-001 | FR-001, RULE-003 / AC-001, AC-021 | completed |
| 3 | UT-002 | BR-001, FR-001, FR-002, FR-011 / AC-001, AC-002, AC-006 | completed |
| 4 | UT-004 | BR-001, FR-001, FR-002, FR-011 / AC-001, AC-002, AC-006 | completed |
| 5 | UT-005 | BR-001, FR-001, FR-002, FR-011 / AC-001, AC-002, AC-006 | completed |
| 6 | AT-001 | BR-001, FR-001, FR-002, FR-011 / AC-001, AC-002, AC-006 | completed |
| 7 | AT-002 | BR-001, FR-001, FR-002, FR-011 / AC-001, AC-002, AC-006 | completed |
| 8 | UT-007 | FR-004, RULE-004 / AC-004, AC-020 | completed |
| 9 | AT-004 | FR-004, RULE-004 / AC-004, AC-020 | completed |
| 10 | IT-001 | FR-004, RULE-004 / AC-004, AC-020 | completed |
| 11 | IT-002 | FR-004, RULE-004 / AC-004, AC-020 | completed |
| 12 | UT-012 | FR-010, FR-011, NFR-002, RULE-001–RULE-004 / AC-012, AC-013, AC-020, AC-021 | completed |
| 13 | UT-013 | FR-010, FR-011, NFR-002, RULE-001–RULE-004 / AC-012, AC-013, AC-020, AC-021 | completed |
| 14 | AT-007 | FR-010, FR-011, NFR-002, RULE-001–RULE-004 / AC-012, AC-013, AC-020, AC-021 | completed |
| 15 | IT-011 | FR-010, FR-011, NFR-002, RULE-001–RULE-004 / AC-012, AC-013, AC-020, AC-021 | completed |
| 16 | UT-008 | FR-005, FR-015 / AC-005, AC-022 | completed |
| 17 | AT-005 | FR-005, FR-015 / AC-005, AC-022 | completed |
| 18 | IT-005 | FR-005, FR-015 / AC-005, AC-022 | completed |
| 19 | IT-013 | FR-005, FR-015 / AC-005, AC-022 | completed |
| 20 | UT-006 | BR-001, FR-003 / AC-003 | completed |
| 21 | AT-003 | BR-001, FR-003 / AC-003 | completed |
| 22 | IT-004 | BR-001, FR-003 / AC-003 | completed |
| 23 | UT-010 | FR-001, FR-007, FR-008 / AC-002, AC-008, AC-009 | completed |
| 24 | UT-011 | FR-001, FR-007, FR-008 / AC-002, AC-008, AC-009 | completed |
| 25 | IT-003 | FR-001, FR-007, FR-008 / AC-002, AC-008, AC-009 | completed |
| 26 | IT-009 | FR-001, FR-007, FR-008 / AC-002, AC-008, AC-009 | completed |
| 27 | UT-009 | FR-006, NFR-003 / AC-007, AC-018 | completed |
| 28 | IT-008 | FR-006, NFR-003 / AC-007, AC-018 | completed |
| 29 | REG-001 | FR-006, NFR-003 / AC-007, AC-018 | completed |
| 30 | REG-002 | FR-006, NFR-003 / AC-007, AC-018 | completed |
| 31 | IT-006 | FR-001, FR-002, FR-008, FR-009, FR-011, FR-012 / AC-001, AC-006, AC-009, AC-010, AC-014 | completed |
| 32 | IT-010 | FR-001, FR-002, FR-008, FR-009, FR-011, FR-012 / AC-001, AC-006, AC-009, AC-010, AC-014 | completed |
| 33 | UT-014 | FR-001, FR-002, FR-008, FR-009, FR-011, FR-012 / AC-001, AC-006, AC-009, AC-010, AC-014 | completed |
| 34 | AT-006 | FR-009, NFR-003, RULE-001, RULE-002 / AC-011, AC-012, AC-018 | completed |
| 35 | IT-007 | FR-009, NFR-003, RULE-001, RULE-002 / AC-011, AC-012, AC-018 | completed |
| 36 | REG-003 | FR-009, NFR-003, RULE-001, RULE-002 / AC-011, AC-012, AC-018 | completed |
| 37 | REG-005 | FR-009, NFR-003, RULE-001, RULE-002 / AC-011, AC-012, AC-018 | completed |
| 38 | UT-015 | NFR-001, NFR-002 / AC-013, AC-016, AC-017 | completed |
| 39 | UT-016 | NFR-001, NFR-002 / AC-013, AC-016, AC-017 | completed |
| 40 | IT-016 | NFR-001, NFR-002 / AC-013, AC-016, AC-017 | completed |
| 41 | IT-012 | FR-010, FR-013, FR-016, RULE-001, RULE-002 / AC-012, AC-015, AC-023 | completed |
| 42 | IT-015 | FR-010, FR-013, FR-016, RULE-001, RULE-002 / AC-012, AC-015, AC-023 | completed |
| 43 | REG-004 | FR-010, FR-013, FR-016, RULE-001, RULE-002 / AC-012, AC-015, AC-023 | completed |
| 44 | REG-006 | FR-010, FR-013, FR-016, RULE-001, RULE-002 / AC-012, AC-015, AC-023 | completed |
| 45 | IT-014 | FR-014 / AC-019 | completed |
| 46 | MAN-002 | FR-014 / AC-019 | completed: implementer source review |
| 47 | MAN-001 | FR-016, FR-017 / AC-023, AC-024 | blocked: no physical iOS/Android device |
| 48 | MAN-003 | FR-016, FR-017 / AC-023, AC-024 | completed: local synthetic investigations |
| 49 | MAN-004 | FR-016, FR-017 / AC-023, AC-024 | completed: checklist written; live execution pending |

## Implementation Steps
### Step 1: UT-003 / FR-001, RULE-003 / AC-021
- Write failing test: unknown/private opaque prose excluded while concrete causes, stage and approved SQLSTATE remain.
- Run command: `cd appview && go test ./internal/observability -run TestDiagnosticUnknownPrivateMessage -count=1`.
- Confirmed failure: Go captured `AppViewError/db.error` instead of concrete SQL cause; Flutter emitted `opaque linen sentence` from the unknown error. Setup/cache/compiler failures were corrected before establishing these meaningful red assertions.
- Implement: Go selects concrete cause types with vetted messages and validated SQLSTATE; Dart selects a safe explanation before SDK formatting and restores concrete type with a scoped processor.
- Green verification: both focused commands passed (2026-10-05). Go retains no fabricated origin stack; Dart preserves `loadPublicRecord` supplied frame. No refactor. Final cross-sink/bounds remain later loops.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned automated Must tests passing
- [ ] MAN-001 physical release-device verification (GAP-003; no attached physical devices)
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Implementation review explicitly deferred to stage exit choice

Automated implementation and local evidence complete; physical release-device verification remains blocked/Pending, and production checklist execution remains separately authorized/Pending. No commit, push or deployment performed.

### Step 2: UT-001 / FR-001 / AC-001
- Red: Go lost exact static `pds: record not found`; Dart replaced `ApiUnauthorized: unauthorized` with generic text.
- Implementation: exact reviewed static sentinel adapter in Go; only sealed static ApiUnauthorized/ApiCanceled explanations in Dart. Arbitrary wrapped/private text stays excluded.
- Green: full Go observability suite and focused Flutter sanitizer/reporter tests passed. Joined/wrapped Go types retained. Updated only the obsolete generic exception-value expectation, preserving its canaries.

### Step 3: UT-002 / FR-001 / AC-002
- Red: Go recovery stack missing, then absolute `/Users/` paths leaked.
- Implementation: recovery-time Sentry stack; strip absolute path to filename. Selected runtime type-assertion/nil-panic explanations retained; unknown panic prose excluded.
- Green: Go observability suite; Flutter supplied-frame/local-path tests. Flutter local-path case passed initially with installed SDK; no invented failure or extra change.
- Commands: `GOCACHE=/private/tmp/craftsky-logging-go-cache go test ./internal/observability -count=1`; `just app-test --no-pub test/observability/diagnostic_sanitizer_test.dart test/observability/sentry_error_reporter_test.dart`.
- Final SDK/native, cross-sink, bounds and HTTP/Tap wiring assertions remain later tests.

### Step 4: UT-004 / FR-002 / AC-001, AC-006
- Red: actual Go/Flutter events omitted actor/target/URI/handle/CID/NSID/record key; attempted DID adapter omitted the useful malformed value. New interfaces were first scaffolded conservatively to establish behavioral failures beyond missing-interface compilation.
- Implementation: sealed/reviewed workflow contexts; public references go in event context rather than identity tags. Attempted DID uses a bounded lexical admission policy, explicit invalid marker and fixed validation reason, never claims a successful parse.
- Green: Go observability suite and Flutter diagnostic/reporter tests passed. Unknown prose remains absent beside public references. Nearby Go middleware and Tap regression suites passed after sandbox escalation for local listeners.

### Step 5: UT-005 / FR-002, FR-011 / AC-006
- Red: both runtime events omitted the specifically selected published failure excerpt.
- Implementation: published-record parse failure adapter only; selected text is credential-sanitized before UTF-8 truncation to 2 KiB. Unknown/draft/failed-publication prose never passes this adapter. Nil Go error emits no event.
- Green verification: Go observability suite passed; Flutter verification passed.
- No diagnostic network fetches or full record/media serialization added. Final sink coverage, broader canaries and bounds tests remain pending.

### Step 6: AT-001 / BR-001, FR-001, FR-002 / AC-001
- Red: real post-read handler emitted generic fallback with no cause/public target; local logger lacked SQLSTATE and target.
- Implementation: request-scoped existing Observer made available by HTTPMetrics; read-store failure describes/logs selected error and captures it as the issue owner. Actual record key is parsed at the request boundary for diagnostic construction only; invalid keys do not change API behavior.
- Green: `go test ./internal/api -run 'Test(GetPost|ReadHandlerLogs|LoggingErrorReporting)' -count=1`.
- Revised only obsolete public actor/target/URI exclusions in existing read telemetry test; retained content/cursor/other protected assertions and added public positives.
- No API response or auth/record-write behavior changed.

### Step 7: AT-002 / BR-001, FR-001 / AC-002
- Red: HTTP and real WS Tap panic events had frames, but local recovery logs omitted concrete cause and stack.
- Implementation: selected panic log helper in each recovery boundary, independent of remote availability. Green-only refactor shares construction across local and issue paths.
- Green: `go test ./internal/observability ./internal/middleware ./internal/tap -count=1`; Tap acceptance uses barriers and verifies no acknowledgement of retryable panic. Flutter supplied exception/frame cases from UT-002 remain passing.

### Step 8: UT-007 / FR-004, RULE-004 / AC-004, AC-020
- In progress: read-only catalogue resolver before arrival, conservative method/encoding/unmatched policy, and companion-context protection.

### Steps 8–11: UT-007, AT-004, IT-001, IT-002 / FR-004, RULE-004
- Red: missing route-aware safe path at arrival and companion-field privacy; held-handler test established arrival before completion. Fixed auth/membership fixture setup before meaningful server red.
- Implementation: read-only catalogue diagnostic resolver, context-scoped public-target selection, INFO ordinary arrival/completion, local unowned 5xx fallback; probe logging wraps existing metrics/trace bypass.
- Green: Go/Dart path matrices, held arrival, ordinary lifecycle success/rejection/unmatched/cancel/5xx, actual-server arrival and probe tests pass. Probe 200 at DEBUG, 4xx WARN, 5xx ERROR; recovered panic after started200 ERROR without rewriting response.
- Existing /healthz returns200 with degraded body, retained. Corrected test expectation; /health dependency failure503 retains ERROR completion.
- Commands: focused Go middleware/server tests and Flutter diagnostic suite. Wider route drift and final envelope protection remain later loops.

### Step 12: UT-012 / FR-010, RULE-001–003
- Red: Go and Dart admitted published text leaked cookie, email, userinfo/signed capability URL, encoded OAuth code/state, local-path and private-key canaries.
- Implementation: supported percent-decoding before sanitization; key-block, generic HTTP URL, email/local-path and reviewed credential replacements before UTF-8 bounds. Generic HTTP URLs have no public provenance, so capabilities/path/query/userinfo are removed. Explicit AT references survive.
- Green: Go observability suite and Flutter diagnostic/reporter suites pass. Nested final-envelope selection is verified separately in IT-011; unknown objects are never admitted by this helper.

### Step 13: UT-013 / FR-010, FR-011, NFR-002
- Red: Go cyclic cause graphs hung inside errors.Is/As classification; Flutter percent-decoder rejected unescaped Unicode. Final event budget pressure is covered with the IT-011 transport fixture below.
- Implementation: bounded independent cause traversal, cycle/repetition detection and explicit omission cause; no unknown Is/As/Stringer traversal. Decode only encoded input and handle invalid encoding safely. UTF-8 truncation occurs after sanitization.
- Green: Go observability/middleware/Tap suites and Flutter boundary tests. Stack/field/record budgets are instantiated in final transport tests. Unsupported nested models/collections are omitted entirely rather than recursively admitting unknown fields.

### Step 14: AT-007 / FR-010, RULE-001–003
- Red: actual SDK scope enrichment added private nested data, arbitrary prose, request/cookie/user details and local stack paths in both runtimes.
- Implementation: final SDK event selection after enrichment, with private in-process selected-event provenance; spoofable tags cannot authorize arbitrary SDK fields. Concrete type and useful sanitized frames remain for unowned SDK exceptions. Unknown SDK prose uses a vetted explanation. Selected public targets and static cause explanations survive.
- Green: Go MockTransport final event serialization and Dart actual SentryEnvelope item transport serialization. Original scope metadata is discarded; occurrence-selected metadata wins over scope overwrites.

### Step 15: IT-011 / FR-010, RULE-001–004
- Red: direct slog/With/localOnlyAttrs emitted unknown prose/models/secrets; SDK Logs/traces exposed arbitrary enrichment; Flutter root platform path emitted no diagnostic when Sentry disabled. Final size tests found over-budget Go (132001 bytes) and Dart (396666 bytes) records. Redacted invalid public segments still authorized companion targets in both runtimes.
- Implementation: protected slog handler installed in shared process initialization (AppView/CLI) and Observer local path; exact reviewed fixed-message catalogue, selected error/cause/workflow fields and conservative unknown-object omission. SDK final Logs/transactions policy, selected platform/root emitter and sanitized framework presentation; invalid path selection denies companion targets. No raw print/developer exception interpolation remains in main.
- Budgets: 2KiB admitted text/path, 32 selected root fields, unsupported nested structures omitted, 8 Go causes, 64 frames/event; frame function/module/filename limits160/120/80bytes. Cause types256bytes. Application error data64KiB, local log16KiB, platform <=8 chunks each<=2KiB. Platform stack aggregate10KiB reserves core fields; identifier overflow is marked, never truncated into a purported valid ID. Full SDK envelopes include SDK metadata outside the owned-record budget.
- Green: Go observability/middleware/Tap suites and shared app initialization tests; Flutter SDK transport, platform/root/framework, diagnostic sanitizer and existing error-handler tests (16 passing in the combined command). Per-record JSON remains parseable, unknown nested content is omitted, public positives survive, paths cannot reappear through companion contexts.
- Sink export binding, full route drift, occurrence ownership, retry suppression, broader call-site migration and F-01–F-20 completion inventory remain later loops. No claim of completed feature coverage.

### Step 16: UT-008 / FR-005 / AC-005
- Red: ordinary warning, significant INFO and debug-opt-in records emitted no independent Logs. Fixed LogRecord object fixture placement before confirming behavioral failure.
- Implementation: emitLog interface in Noop/Guarded/Sentry reporters and compatible test doubles; forwarding chooses Logs, breadcrumbs and issues independently. WARNING/ERROR Logs, explicit significant INFO Logs+breadcrumb, unselected INFO omitted; FINE remote requires opt-in. Supporting structured records can declare occurrence ownership. Known safe messages retained; arbitrary interpolated prose omitted.
- Green: forwarding, reporter, bootstrap and Sentry reporter tests. Revised only obsolete warning-local-only/App-log expectations to match FR-005, preserving cause/stack/context assertions. Full serialized root-path evidence follows AT-005/IT-005.

### Steps 17–18: AT-005, IT-005 / FR-005
- Red: actual root forwarding exported the distinct message-only/exception/warning/significant INFO Logs but final issue omitted the relevant bootstrap breadcrumb.
- Implementation: reviewed fixed/coarse breadcrumb source policy in SDK callbacks and final events, bounded50entries; arbitrary SDK messages/private data remain excluded.
- Green: actual SentryEnvelope transport shows five distinct selected Logs, one later unexpected issue, relevant breadcrumb and named stack; FINE absent. SDK Logs disabled leaves one safe local warning and no remote payload. Forwarding and existing breadcrumb suites pass.

### Step 19: IT-013 / FR-015
- Red: configured direct slog record never exported (2 exported records instead of3); Observer records did export separately.
- Implementation: protected handler export binding shared by parent/With loggers; Observer.Log uses that same path with no duplicate dispatch. Private short-lived SDK provenance snapshots are consumed synchronously (max256 active calls, no stored history), ignoring SDK-added fields and removing internal tokens before serialization. Unsupported/no-provenance SDK records keep conservative final policy. Selected lifecycle/significant INFO and all WARNING/ERROR export; successful probe DEBUG does not.
- Green: configured local/export Logs match and have no issues/traces, Logs-off retains local records; actual middleware lifecycle export works with tracing off and enabled-but-zero-rate. Go observability/middleware/app initialization tests passed. Revised only obsolete local-only safe trace exclusion; protection of secret/private attrs stays asserted. Correlation IDs themselves remain the next loop.

### Step 20: UT-006 / FR-003
- Red: request-context ID disappeared from local/export logs and issue context without sampling; Flutter mapper retained server ID but final issue lost it as correlation context.
- Implementation: request/trace context extraction shared by diagnostic logs/capture and protected handler; valid run_id retained unchanged. High-cardinality request/trace/case IDs go in issue correlation context rather than generic issue tags. Flutter mapped appViewRequestId reaches platform/Logs and final event context; missing IDs remain absent.
- Green: Go no-DSN/DSN-unsampled matrix; Dart actual serialized event retains ID/release/environment after mapping, and missing ID is absent. Neighboring Go middleware/PDS active-span and Dart mapper/platform tests passed. Replaced obsolete trace-tag assertions with positive correlation-context assertions.

### Steps 21–22: AT-003, IT-004 / FR-003
- Paired integration: real Go GetPost failure emits requestId, local/export/cause/issue trail and optional generated fixture; actual Dart Dio interceptor and AppError mapper consume that same envelope, followed by root platform/SDK log and owned issue. Both harnesses passed using CRAFTSKY_CORRELATION_FIXTURE=/private/tmp/craftsky-logging-correlation.json. No static replacement ID was substituted.
- Red extensions: process metadata absent from local lifecycle records; sampled child span omitted run_id; tracing config without DSN fabricated UUID trace/span IDs.
- Implementation: protected process logger metadata binding; all actual SDK spans receive request correlation and their own trace/span IDs; no backend leaves spans unavailable and no IDs fabricated. Real-backend test setup replaces obsolete no-backend populated-span expectation.
- Green: no-DSN/unsampled/sampled Go envelope matrix and paired Flutter local/export/event test. Missing server IDs were already verified absent in UT-006. Wider Go API/observability/middleware verification is running next.
- The paired Dart integration skips with a documented reason when its generated Go fixture env is not supplied; final verification must run the explicit paired commands.

### Step 23: UT-010 / FR-007
- Red: two Go captures per marked occurrence and ownership absent without DSN; Flutter callback plus root log captured twice (4 versus2).
- Implementation: atomic per-occurrence claim before backend availability; CaptureError delegates immutable selected diagnostics. Callback supporting records carry explicit issue ownership. Bounded cause matching replaces potentially cyclic errors.Is traversal in capture. No global error/text deduplication.
- Green: focused Go ownership and Flutter UT-010 tests pass; independently reused objects retain two occurrences.

### Step 24: UT-011 / FR-008
- Red: Go expected conditions produced6 issues; Flutter severe expected API errors produced5.
- Implementation: bounded expected/terminal classifier; cancellation always expected, exhausted/terminal/quarantine failures remain reportable. Go ordinary network conditions differ from actionable server dependency failures. Flutter supporting Logs survive independently; explicit outcome permits terminal retry ownership.
- Green: focused classification tests and nearby Go observability/middleware/Tap plus Flutter ownership/forwarding suites. Selected-cause/enrichment tests explicitly identify their terminal occurrence, retaining public/cause positives and all privacy canaries.
- Direct SDK-owner and provider integration remain IT-009; no claim those callers are migrated yet.

### Step 25: IT-003 / FR-001, FR-007
- Actual HTTP Logging/metrics/recovery matrix covers deeper capture, uncaptured5xx fallback, panic before write and panic after started200, each repeated independently. Passed initially after UT-010; no production change was necessary and no failure was invented. Ordinary errors remain stackless; panic frames present; started response untouched; two issues for two occurrences.
- Existing real Tap panic/no-ack and middleware suites also pass.

### Step 26: IT-009 / FR-001, FR-007, FR-008
- Red: actual SDK provider+expected-owner fixture produced5 issues instead of3; provider type replaced by placeholder and no cause/stack in platform output.
- Implementation: provider supporting selected log carries original cause/stack plus explicit issue ownership; provider owner passes original cause to protected reporter. Sentry owner applies expected/terminal classification.
- Green: two independently repeated StateErrors retain supplied readPublishedPost frame and safe local output; expected expiry/offline absent, terminal offline retained. Serialized canaries absent. Updated obsolete raw LogRecord/error string assertion to inspect selected actual platform output and cause fields instead; preserved privacy negatives and added original-type positive.

### Step 27: UT-009 / FR-006
- Red: AppError omitted safe message/field reasons/method/route/path; underlying FormatException mapped as expected network failure.
- Implementation: selected static AppView message and validation vocabulary; bounded allowed-field reasons; method/route/safe actual path through both mapping layers. ApiFailureDetails/AppError carry underlying cause and supplied stack without rendering it. Underlying unexpected parse failures remain reportable. Final issue keeps high-cardinality API diagnostics in context; platform/Logs retain selected diagnostics. Technical context now requires stable token syntax, not arbitrary plain prose.
- Green: mapper suites and actual SDK regression fixtures pass; malformed/private envelope text excluded, representative status matrix retains IDs. Full route coverage/drift is next.

### Step 28: IT-008 / FR-006, RULE-004
- Red: all actual route policies lacked explicit metadata; Flutter account-deletion route mapped unknown. Invalid caption CID granted public companion context.
- Implementation: reviewed explicit JSON manifest covers every production/dev/admin method+path with stable category and public/private diagnostic target rule; Go embeds it and Flutter generated constants consume it. Drift tests detect missing/orphan policies and generated mismatch. Read-only resolver follows literal specificity and known methods; unknown routes remain conservative. Strict CID library used directly (existing model Cid adapter accepts legacy fake values).
- Green: actual catalogue metadata test plus every-route Flutter interceptor/resolver matrix, existing path and mapper regressions. Obsolete /v1/feed fixture replaced with actual /v1/feed/timeline; no invented route added.
- Regenerate Flutter via python3 scripts/generate-diagnostic-routes after reviewed manifest changes, then run both drift tests.

### Steps 29–30: REG-001, REG-002 / NFR-003
- Existing Go envelope/API/routes regression command passed. Existing Dart localized copy/actions, mapper and active-account initialization gate widget suites passed (35tests). Passed initially after enrichment; no artificial red or production behavior change.
- Public responses retain camelCase envelope and existing recovery actions. Internal operator detail stays in selected diagnostics.

### Step 31: IT-006 / FR-001, FR-002, FR-008, FR-009, FR-011 (completed)
- Completed red/green boundary slices: real auth registration lost pgconn cause (now request-owned cause capture via neutral ctxkeys callback); shared API author list lost cause/public target (now error-helper call sites carry actual enclosing errors and catalogue-selected public workflow); PDS completion omitted cause with DSN disabled (now selected causes); DB wrapper had no local/issue failure (now same selected cause owner).
- Tap retry initially created an unexpected issue, now selected cause/URI/CID/event/outcome/no-ack remains without issue. Durable terminal rejection initially had no issue, now selected published FeedPost text/record context and durably_committed/quarantine outcome; credentials removed. Typed generated lexicon used, no schema change. Existing actual WS ACK/retry behavior unchanged.
- Commands: focused TestAuthDiagnosticRetainsUnderlyingCause, TestAuthorListDiagnosticRetainsCauseAndPublicTarget, TestPDSCompletionDiagnosticRetainsCauseInLocalLog, TestDBDiagnosticRetainsSQLStateAndCauseWithoutRowText, TestWSConsumer_EmitsTapMetricsAndCapturesIndexerErrors, TestTapTerminalRejectionRetainsPublishedFailureExcerpt.
- Auth SDK transport fixture permitted by exact test-file import-boundary exception; production SDK import restriction unchanged. Actual durable projection/indexer and remaining API/helper coverage audit still pending before completing IT-006.

- Additional IT-006 red/green slices: real durable projection worker and profile backfiller retained source/event/outcome; onboarding projection wrapping now preserves errors.Is cause; PDS failure retains repository/collection and vetted HTTP status. Public references positive assertions replace obsolete broad DID bans, while session/body/provider negatives remain.
- Bare cause-bearing API 5xx branches now use ReportRequestFailure; shared command-error response accepts initiating context and production callers provide it. Real event-read and command-response tests retain PG SQLSTATE, exclude private intent, and keep envelopes unchanged. Caption fetch retains concrete cause/stage and content-validation failure without logging caption bytes.
- Actual malformed-input WS quarantine now emits durable terminal outcome/event/validated companion identifiers; identity panic captures concrete runtime type and recovery frames. Mixed metrics fixture now expects its two actual terminal quarantines and explicitly denies an issue for event2's expected retry.
- Green commands: go test ./internal/tap ./internal/api ./internal/observability ./internal/auth -count=1; focused local-DB projection/backfill/dispatcher checks. No production action. F-01–F-20 completion inventory, private failures and retry-volume refinements remain later planned loops.

### Step 32: IT-010 / FR-009 / AC-010 (completed)
- Next: actual initialization, storage, video/media, mutation and mention/device boundaries; retain initiating account and original cause without private bytes or input.

- IT-010 executed red/green slices: initialization gate retains original error/stack/account with supporting ownership; provider-family failure snapshots typed initiating lease; source-owned post mutation keeps actor A after B activates and one issue whether current or stale. Synchronous scoped failure metadata preserves the original AsyncError and does not retain history.
- Secure snapshot read/write, auth handoff wrappers and device storage retain original typed causes/stack and selected native codes/stages without snapshots/tokens/device IDs. ServerUnavailable retains original API cause; bounded app-owned wrapper mapping keeps HTTP status/request ID and expected offline behavior. Auth regression suite: 25 passing.
- Video coordinator keeps original exceptions/stacks, phase, selected provider HTTP status/code and initiating account; limits/authorization/upload/polling/publication fault matrix passed initially after shared adapters (no invented red). Actual Dio upload failure retains original Dio cause/status and JWT confinement tests pass. No ephemeral JWT/job/proof/media/alt text is diagnostic context.
- Image preparation and picker/camera errors retain causes/stages; permission denial is a warning with explicit expected outcome. Picker account is captured before awaiting native UI, and no diagnostic-only registry provider is initialized. Optional caption fetch/resource failure now retains concrete cause and stage while playback stays available. Caption resource/text/label remain excluded.
- Facet API/Dio failures preserve safe envelope correlation and static operation; partial response decode aggregates one cause/count, preserving valid items. Query/handle/item payload/results are excluded. Decode red first showed duplicate warnings, then dropped count; both corrected and all 5 facet tests pass.
- Combined nearby auth/mapper/root bridge/provider/image regressions: 57 passing; initialization/storage/device/facet/video/mutation regressions: 61 passing; native player: 4 passing. Earlier image preparation timing failure did not recur. Fixed deterministic visibility timer teardown in the new caption widget fixture; no business change.
- Remaining global/private sink refinements and exhaustive finding inventory are later explicit loops, not feature completion claims. No commits, production inspection or mutations.

### Step 33: UT-014 / FR-012 / AC-014 (completed)
- Explicit selected public model summaries; mixed private representations remain protected.

- UT-014 red: selected public model contexts were empty, losing public DID. Added closed ModelDiagnosticSummary with enum model/state, bounded count and flags. Explicit extensions select SettingsIdentity, AuthState, ActiveAccountIdentity, PostSummaryData and business event/detail/list references; no UI labels, prose, images/URLs, cursors, errors or sessions serialized. Public references require an explicit publicWorkflow declaration; default summaries have no identity/target/membership. Existing redacted toString methods remain. Unit green: just app-test --no-pub test/observability/diagnostic_summary_test.dart.

### Step 34: AT-006 / FR-009, RULE-002 / AC-011 (completed)
- Failure-only private operational identity/workflow references, no private target membership or account-linked success history.

- AT-006 Go red: private-route selection removed initiating DID/workflow reference despite retained PG SQLSTATE/stage. Closed constructor now requires a non-nil failure, accepts only initiating typed DID and UUID worker reference, exposes no targets/payload; focused Go local/mock SDK test green.
- Paired Dart red confirmed missing initiating DID in private operational failure. Conservative context scaffold remains empty. Automatic approval review twice rejected the proposed edit, requiring direct trusted user authorization for operation account DID and workflow reference in local/Sentry output despite reviewed FR-009/AC-011. Requested that exact authorization asynchronously. No rejected edit ran, no live telemetry or production action performed. AT-006 stays in_progress pending that answer; later dependent private-worker loops not started. Read-only analysis and completed-boundary regression checks proceed independently.

- Independent verification checkpoint while waiting for AT-006 authorization: Go observability/API/auth/Tap/middleware/routes passed. Flutter analysis failed (1020 infos, 9 warnings, no errors); detailed output is in /private/tmp/craftsky-logging-analysis.txt. Formatting/lint cleanup waits until the active red loop is resolved. No dependency files changed; git diff --check passed.
- Full just test race/integration run exposed two obsolete expectations so far: TestIR017ScheduledSaveLifecycleExcludesCanariesFromApplicableSinks requires zero exported lifecycle logs (now required by FR-015); TestPDSMigrationProductionPathsAreCrossSinkSecretFree searches correlation IDs in event tags (now correlation context under NFR-001). Both need positive contract-aligned assertions preserving original canary protection, in the later regression loop. Full run result pending; no claim it passed.

- Full `just test` completed with six failures: scheduled-save and link-preview external-log-zero expectations; PDS migration event correlation tag expectation; moderation operation export count; push moderation alert attributes; MinIO private object-store availability. Remaining packages passed with race/integration checks. These are outstanding regression/environment work, not silently accepted. Flutter lint cleanup and remaining steps 34–49 are incomplete. No active test processes remain.

- User directly approved the exact failure-only fields/destinations and instructed continuation. Rejected action retried with that approval and succeeded. AT-006 Dart platform/export selection and fully serialized injected SDK event tests now pass; successes exclude private activity identifiers. No live transmission. Go paired test already green. Approval applies to the documented operation-account/non-capability reference boundary only.

### Step 35: IT-007 / FR-009, RULE-002 / AC-011 (completed)
- Actual private workers/integration faults, persisted safe summaries and unchanged business retry/authorization behavior.

- IT-007 scheduled publication red lost original PG cause through ambiguous command result. Fresh append-command results now carry an in-memory DiagnosticCause excluded from JSON and persistence. Worker retains attempt/account/job/stage/retry or terminal outcome; classifier/persisted decision unchanged. Actual DB retry test and command lost-response reconciliation test green.
- Account deletion capped retry red lost account/job/PG cause. Source failure report retains those selected fields without lease/private prose; six-hour capped retry and category unchanged. Focused production-boundary test green.
- Push send red discarded provider cause. Optional failure observer retains owning account/delivery UUID/provider stage/attempt/cause without actor, notification, subscription, routing, install, token or payload. Actual DB status remains retry; original sentinel test green.
- Instagram provider retry/permanent and redemption/membership/candidate storage faults red had no cause diagnostics. Worker now retains actual source cause, known initiating owner and work reference; owner absent before successful redemption. Private username/IGSID/lease/attempt capability omitted. Existing retry, terminal, cancellation, completion and optional-reply tests green. Production worker gets the existing Observer dependency. Other worker stages, finalization and persisted-summary/source audits remain in this loop.

- IT-007 additional red/green: Instagram owner inactivation/rate-limit storage, optional reply and durable completion failures; deletion safety retry/finalization/persistence failure, including both original and persistence SQLSTATEs; private HTTP cause boundary selects authenticated operation owner only; scheduled returned snapshot fault and concrete record_write stage. Existing business transitions/cancellation/fencing remain unchanged. Instagram real HTTP client now retains vetted HTTP status (429 fixture); selected cause explains closed provider classification without response prose. Production Instagram worker receives existing Observer. Tests use synthetic accounts/buffers only.
- Focused IT-007 fixtures for all four worker classes and private HTTP boundary are green. The global F-01–F-20 caller inventory/final emission and retry-volume audits still apply before feature completion; this is not release approval.

### Step 36: REG-003 / NFR-003 / AC-018 (completed)
- Run real local worker/recovery/authorization/fencing/ACK regressions after private failure enrichment.

- REG-003 real local worker/recovery regression command passed for scheduling, deletion, Instagram, Tap, middleware and command reconciliation. Push failed only its obsolete exact3attrs assumption: required service/environment/release now accompanies the3selected operation fields. Replaced exact count with positive service assertion; original operation/outcome and all owner/device/token canaries retained. Push Dispatcher and IT-007 regression command green. No retry/fencing/authorization/ACK changes.

### Step 37: REG-005 / RULE-001, RULE-002 / AC-012 (completed)
- Preserve protected model/session/private activity and video credential confinement tests.

- REG-005 Go red: old push/scheduled privacy fixtures prohibited the explicitly approved failure owner. Updated with narrow structural positive checks: exactly one failed local/exported push diagnostic; exactly one scheduled issue diagnostic; owning DID only in operation_account_did. Remove only that proven allowed field for the existing all-canary scan. Success, metrics, transactions and all other private routing/payload/credential values still deny the owner and original canaries. Scheduled lifecycle now positively requires exported Logs under FR-015. Go observability/auth/API protected-value command green; Flutter secret/video/storage/protected-model matrix:25passing.

### Step 38: UT-015 / NFR-001 / AC-016 (completed)
- Verify many changing request/workflow identifiers retain distinct diagnostic context while aggregation/grouping dimensions remain stable in both runtimes.

- UT-015 Go actual SDK:20distinct account/workflow/request IDs,20issues+20sampled transactions; issue tags/transaction names/metric dimensions stable, diagnostic/correlation contexts distinct, no grouping override. Dart actual serialized SDK:20StateError occurrences keep stable tags with20distinct correlations/public actors; separate FormatException retains its concrete type. Both passed initially after earlier central selection (no artificial red/production change).

### Step 39: UT-016 / NFR-002 / AC-017 (completed)
- Chosen retry-only suppression:30second window,first emission retained,summary suppressed count on next window/terminal,maximum128ephemeral scopes; capacity overflow emits unsuppressed rather than losing independent failures. Unexpected/terminal occurrences always emitted. Scope uses reviewed operation/stage plus non-capability workflow reference when present; no payload/account history. Fake-clock boundary fixtures next. Caller integration remains IT-016.

- UT-016 Go red:15burst logs, no summary. Protected handler now coalesces explicit retry records before both local/export sinks using a shared mutex-protected limiter. First/window/terminal output3records with0/9/3suppressed; terminal PG08006 still one issue. Dart root applies one retry decision to both local/remote dispatch, retaining original record/error/stack; same fake-clock0/9/3fixture green.
- Capacity/idle-expiry red:135versus134logs under128scopes plus overflow. Both runtimes now lazily expire idle scopes after60seconds (and reset on backward clock), preserve overflow occurrences unsuppressed, and always emit independent unexpected/terminal occurrences. Core/neighboring Go tests green; Flutter platform/root/serialized SDK regression suite18passing. Scope counters retain no error/payload/account objects. Summaries apply to active window counters; expired idle counters are discarded.

### Step 40: IT-016 / NFR-002 / AC-017 (completed)
- Verify actual caller retry bursts preserve metrics, business outcomes and final selected cause across local/export sinks.

- IT-016 real Tap WS red:terminal parse stage failed to drain projection retry counter. Reviewed event/workflow references now span stage changes within the same operation.10actual deliveries retain10metric attempts,no retry ACK,one durable terminal ACK,2selected local/exported diagnostic logs,terminal8suppressed and1issue. Fixture corrected to the existing durable contract:terminal outcome with non-nil error is never ACKed; durable terminal success returns nil and reports selected published rejection context. No business change.
- Next red found9duplicate generic ERROR Logs after the owned retry warnings. Existing generic supporting Tap line now DEBUG; actionable selected first/terminal cause records remain. WS metrics, ACK and terminal regressions green.
- Dart actual root/SDK Logs on/off stage-transition red lost summary. Shared reviewed workflow scope now spans stages; local2records,8suppressed terminal,one original StateError/named stack issue,remote2or0dependingLogs. No credential/private prose. Flutter startup subscription integration remains the explicit IT-015 loop.

### Step 41: IT-012 / FR-013 / AC-015 (completed)
- Throwing reporter/exporter/failed flush must retain safe local diagnostics without recursion or business changes.

- IT-012 red/green: Go throwing exporter, synchronous capture, asynchronous SDK batch transport panic, failed/panicking flush and local sink panic. Local fallbacks bypass exporter/retry gate; original selected cause retained where available. Injected transport wrapper preserves default SDK transport selection. Child-process async fixture proves process survives rather than masking a goroutine panic. Focused Go tests green.
- Flutter red/green: throwing capture/log/breadcrumb/state, initialization and actual root forwarding. Direct selected local fallback retains original cause/stack/request ID, never reporter recursion/private prose; initialization remains no-op on failure. Reporter and platform/root/bootstrap regression suite green. No live telemetry.

### Step 42: IT-015 / FR-016 / AC-023 (completed)
- Consolidate actual startup into one root subscription before SDK initialization; keep local release-policy emission and one retry admission across initialization.

- IT-015 red: startup root subscription stayed bound to the initial no-op reporter. It now resolves the current reporter per record, keeps a default platform emitter and one retry limiter, and is installed once before SDK initialization. Local output before/after/without SDK retains cause, named frame and request ID; initialized reporter receives exactly one later log. Nearby root/reporter/platform tests:17passed. Actual device validation remains MAN-001 Pending.

### Step 43: REG-004 / FR-013 / AC-015 (completed)
- Existing no-DSN, failed initialization, framework callback and platform/reporter fault suites pass. Original caller behavior/no-op fallback preserved. No artificial red or business change.

### Step 44: REG-006 / FR-010, RULE-001, RULE-002 / AC-012 (completed)
- Existing Go SDK configuration (automatic user/header/cookie/query/body collection disabled), Flutter effective options/import-boundary and release configuration regressions pass. Flutter combined command15passed; Go focused command passed. No SDK collection broadened. No artificial red.

### Step 45: IT-014 / FR-014 / AC-019 (completed)
- Add one canonical guide linked by mandatory root AGENTS policy, with executable examples and positive/negative source-context assertions.

- IT-014 red: canonical guide file/link absent. Added mandatory root policy and one practical guide covering source selection, severity, ownership, cause/stack, path/URL, correlation, sink gates, bounds/retry suppression, correct/incorrect Go/Flutter examples, commands and checklist. Callable Go and Dart example tests retain SQLSTATE/type/stack/public target/failure owner/job/request IDs and exclude private canaries; both pass. Manual usability review MAN-002 remains separate.

## Final source/regression audit (completed; historical checkpoints below)
- Canonical guide examples green. Earlier broad-suite gaps remain open; no completion or release claim.
- Local test services were no longer running; focused DB test failed setup with connection refused, not a code regression. Restart only Postgres/MinIO and checked-in private bucket bootstrap for integration checks.

- Final audit red/green refinements tied to existing IDs: IT-010 playback-open and stream causes (soft UI/grace policy unchanged); UT-011 outer AppError expected override; UT-003 closed reviewed API codes; UT-013 omission marker inside stack count/byte limits; UT-006 local environment/release without SDK; AT-006 malformed/oversized private owner omitted; UT-014 typed parser errors and serialized public/private summaries; IT-006 direct PDS listing; IT-007 deletion claim and safety persistence, push finalization, Instagram retry persistence retain source causes. Focused regressions green.
- Broad Go race/PostgreSQL/MinIO run passed before the final source refinements. Final affected Go packages require race rerun. Full Flutter run:2701passed,38skipped,2failures; both obsolete initialization/pin-message expectations corrected and focused61test command passed. Final full rerun required after cleanup.
- File-scoped Dart fixes/formatting completed68files. Analysis recipe attempted dependency refresh and failed resolution; `flutter analyze --no-pub` ran against resolved dependencies with104issues (no errors), down from1029. Remaining changed-code cleanup and baseline comparison pending.


### Step 46: MAN-002 / FR-014 / AC-019 (completed)
- Manual implementer source review on 2026-10-05: begin at root AGENTS policy, follow canonical guide, choose public HTTP ReportRequestFailure/LogDiagnostic and issue owner, private worker ObservePrivateFailure and Flutter provider/source-owned DiagnosticMessage and reporter. Interfaces match executable IT-014 examples. Severity, expected outcomes, ownership, correlation, path/URL, public/private provenance, bounds/retry and sink gates covered.
- Corrected numeric spacing during usability review. Guide is usable without an optional skill. This is implementer review, not a claim that the maintainer reviewed it; implementation review remains the stage exit choice.

### Step 47: MAN-001 / FR-016 / AC-023 (blocked verification)
- `cd app && flutter devices --machine` completed on 2026-10-05: iOS simulator, macOS and web, zero physical iOS/Android devices. Device identifiers/names excluded from checked-in evidence.
- Release-policy platform/startup/failure tests pass but cannot prove compiled release console retrieval. GAP-003 remains Pending for supported attached devices. No persistent buffer/support UI added.

### Step 48: MAN-003 / FR-017 / AC-024 (completed local evidence)
- Generated selected local/mock SDK artifacts for public read, Tap durable malformed-record rejection, private scheduled snapshot failure, real Riverpod provider, cold-start account initialization, telemetry initialization fallback and paired Go/API/Dio/AppError/Flutter request trail.
- Commands, gates, excerpts, references, cause-location steps and limits are in `06-validation-evidence.md`; `evidence/` contains synthetic data only. Checked artifacts for secrets/private canaries/user paths. Jobs/Tap have no manufactured HTTP trail. No live/production evidence claimed.

### Step 49: MAN-004 / FR-017 / AC-024 (completed checklist; live execution Pending)
- Separate pending checklist in `06-validation-evidence.md`: effective environment/release/Logs gates, ordinary retrieval/access, unsampled correlation and retention/access policy.
- Execution requires separately authorized rollout. No production inspection/mutation, deploy, push or synthetic production failure performed.

### Final audit supplement
- IT-007 Instagram claim red: absent SQLSTATE08006 source log. Claim diagnostics now have no invented owner/job before success; worker regressions pass.
- IT-007 scheduling finalization red: SQLSTATE08006 retained but stage command_setup. Stage now tracks effect acquisition/media upload/record write/finalization/release, preserving transitions. Local PostgreSQL publication regressions pass.
- IT-011 measurement red: arbitrary private measurement name survived SDK trace. Cleared unsupported measurements, curated Observer metrics unaffected; focused SDK tests pass. AT-007 unsupported Go module sentinel passed initially because SDK integration replaces caller module input; no artificial red/change.
- Mechanical Dart cleanup briefly produced invalid cascades; corrected before handoff. File-scoped fixes also proposed unneeded any dependencies; reverted and used existing Sentry Flutter re-exports. Dependency files unchanged. Final analysis/regression outcome follows after completion.


## Final completion record (2026-10-05)
- 45 automated test-ID loops completed in approved order with meaningful red/green slices or explicitly recorded already-passing regressions. Final source refinements remain linked to those IDs.
- `just test`: Passed, complete Go race suite with local PostgreSQL/MinIO integration. Final unsupported SDK module sentinel also passed separately.
- `just app-test --no-pub`: Passed, 2708passed/38skipped. After final SDK measurement selection, affected SDK/correlation/bootstrap16test command and account-initialization11test command passed. Paired request fixture executed explicitly.
- `cd app && flutter analyze --no-pub`: Passed, No issues found. `git diff --check`: Passed. Dependencies/lockfiles, migrations, lexicons and production infrastructure/configuration unchanged.
- MAN-002 implementer guide source/usability review and MAN-003 local synthetic investigations complete. MAN-004 checklist written, live execution Pending. MAN-001 blocked by absent physical iOS/Android test devices, GAP-003 remains Pending; adapter tests do not close it.
- Read back this final plan and validation evidence. Implementation review not run; offered at stage exit. No stage commit/push/deployment/live telemetry/production action. Changes remain in the current worktree for review.

## Review correction pass (2026-10-06)

User selected Address required changes for the four concrete findings in `06-implementation-review.md`. This authorizes correction within the already approved private/public boundary. No commit, deployment or live telemetry operation is enabled.

| Order | Test ID / requirement | Review finding | Status |
|---|---|---|---|
| C1 | IT-007 / FR-009, FR-010, FR-011, RULE-002 / AC-011, AC-012 | IR-002: private background PDS propagation | completed |
| C2 | IT-015 / FR-013, FR-016 / AC-015, AC-023 | IR-001: default product sink | completed |
| C3 | UT-013 / FR-010, NFR-002 / AC-013 | IR-003: aggregate core/chunk budget | completed |
| C4 | IT-007 / FR-001, FR-009, RULE-002 / AC-010, AC-011, AC-021 | IR-004: scheduled media I/O causes | completed |
| C5 | Final verification and evidence/read-back | Focused regressions, broad suites, guide alignment | completed |

Order follows the review handoff: the confirmed privacy defect comes first; each behavior gets its own red/green loop. Physical-device MAN-001 and live MAN-004 remain separate pending verification.

### C1: IT-007 / FR-009, FR-010, FR-011, RULE-002 / AC-011, AC-012

- Red: full migrated-schema scheduled-publication fixture through observed fenced command/PDS callback exposed unpublished record key. Initial fixture/schema failures were corrected before accepting the behavioral red.
- Green: private context propagates from processor through commands/effects; lower PDS failures select permitted operation-account/job references only. Unclassified background PDS operations omit inferred public context; public positive fixtures now explicitly declare request provenance. Dependency failure and expected missing-record cases pass, original worker outcomes unchanged.
- Narrow test-only Sentry import exception permits this serialized sink fixture; production import boundary stays unchanged.

### C2: IT-015 / FR-013, FR-016 / AC-015, AC-023

- Red: permanent product AOT probe compiled the real default adapter and emitted only its completion marker, failing the WARNING/cause/correlation output assertion.
- Green: default sink uses Dart print, retained in product builds and routed by Flutter to its platform console. Product AOT probe passes; injected platform sanitization and bootstrap checks remain separate. Physical device retrieval still MAN-001 Pending. A mistyped bootstrap suite path was corrected before interpreting the combined command result.

### C3: UT-013 / FR-010, NFR-002 / AC-013

- Red: combined accepted 8-cause/64-frame/context/validation inputs emitted 14400 bytes of incomplete JSON; permanent fixture failed decoding.
- Green: whole selected platform record (including process/suppression metadata) is structurally bounded to 6500 bytes before encoding, allowing JSON-string escaping overhead within eight complete <=2048-byte wrappers. Optional enrichment is removed before useful stack frames; explicit omission markers retain cause/correlation/frame positives. Chunk boundaries remain UTF-8-safe and account for wrapper bytes.
- Nearby platform/product/bootstrap/serialized SDK command: 21 passed. Lower sink budget is deliberate to guarantee complete wrappers under escaping, as permitted by DR-004.

- C3 follow-up red/green: strengthened maximum-record fixture with private job correlation. Old optional workflow admission dropped the UUID before core reduction. Core reduction now precedes workflow admission; workflow correlation is prioritized during the final metadata-aware pass. Positive request ID, original cause type, named frame and private job reference all survive alongside bounded complete JSON. Nearby/full Flutter verification rerun after this refinement.

### C4: IT-007 / FR-001, FR-009, RULE-002 / AC-010, AC-011, AC-021

- Red: worker-level Open/Read/Close fixtures lost original `*http.ResponseError`, `*net.OpError` and `*fs.PathError` in local and serialized mock exported output. The initial Open-status expectation was corrected to the existing retrying policy; the missing types/status remained meaningful behavioral failures.
- Green: Open joins its original cause with ErrObjectUnavailable; Read/Close join their causes with ErrMediaInvalid, keeping size/hash mismatch classification separate. The selector retains only vetted Smithy HTTP status (503 fixture), never response/header/provider prose. Three fixtures pass, account/job/stage positives survive, private object/header/session/media canaries remain absent. Persisted Open retrying/object_unavailable and Read/Close needs_attention/media_invalid outcomes are unchanged.

### C5: Final correction verification (completed)

- Affected Go observability/scheduledposts/pdscommands/middleware suites passed with race detection and required local PostgreSQL integration.
- Full `just test` passed with race detection and local PostgreSQL/MinIO integration. Transcript: `/private/tmp/craftsky-logging-corrections-go-full.txt`.
- Full `cd app && flutter test --no-pub` on final sources: **2710 passed / 38 skipped**. Transcript: `/private/tmp/craftsky-logging-corrections-flutter-full.txt`. Full rerun followed the C3 workflow-correlation refinement; the product probe imports the actual default adapter via package resolution.
- `cd app && flutter analyze --no-pub`: **No issues found**. Transcript: `/private/tmp/craftsky-logging-corrections-analysis.txt`. Seven initial lint infos were corrected; the logging framework's deliberate platform print has a narrow justified lint exclusion.
- Exact vetted storage HTTP explanation asserted in both sinks; the two new scheduled regression tests passed again with race detection after strengthening that positive oracle.
- `git diff --check` passed. Guide now describes private propagation, product-safe platform output and complete serialized chunk/whole-record bounds. No dependencies, native platform configuration, lockfiles, migrations, lexicons or production configuration changed.
- IR-001–IR-004 corrections implemented and regression-tested; the existing review verdict remains historical **Changes required** until a new implementation review. This is implementation completion, not review approval.
- MAN-001 physical attached-device retrieval and MAN-004 live production execution remain Pending. No commit, push, deployment or live telemetry operation performed. All correction progress items complete; requested implementation review remains the exit choice.

## Second review correction pass (2026-10-06)

User selected Address required changes for remaining IR-003. Existing provenance/privacy authorization applies; this pass only reduces selected platform output and does not widen data permissions. No commit or production action enabled.

| Order | Test ID / requirement / acceptance | Work | Status |
|---|---|---|---|
| C6 | UT-013 / FR-010, NFR-002 / AC-013 | Permanent accepted long-cause/path/private-account combination; final structural fallback | completed |
| C7 | IT-011, IT-015 / FR-010, FR-016 / AC-012, AC-023 | Nearby platform/SDK/root regressions, broader Flutter suite, analyzer, guide/evidence/read-back | completed |

- Setup completed: reloaded authoritative workflow documents, second review, mandatory guide, current emitter and permanent/scratch fixtures.
- Plan: one UT-013 behavior loop; vary absent/minimum stack and final process/suppression metadata within that bound invariant. Confirm meaningful red before changing source; preserve normal-sized request/job IDs and useful cause/frame positives.

### C6: UT-013 / FR-010, NFR-002 / AC-013

- Red: permanent admitted long-type/cause/path/private-owner fixture failed the 6500-byte assertion at **7626 bytes**, including production/release/suppression metadata. Joined JSON decoded successfully, matching the narrowed second-review defect.
- Green: optional path detail yields as a whole; intermediary wrappers yield before ordinary correlation, retaining outer/deepest causes and an explicit omission node. A final compact schema bounds scalar fields, retains one concrete cause/useful frame, and marks oversized workflow identifiers rather than presenting partial IDs. Both minimum-stack and no-stack variants pass with positive operation/request/job/cause/frame/process metadata and private-cause absence checks.
- Follow-up red: an oversized multibyte custom Level name forced the compact fallback but still caused incomplete JSON. Green: compact scalar strings now have a 160-byte admission limit and explicit omission. The same test passes, preserving normal owner/job/request references, cause/frame, operation and suppression count. This is the same UT-013 aggregate-bound invariant; it adds no business behavior or data permission.
- Nearby platform/SDK/root/sanitizer/correlation suites before the follow-up: 97 passed / 1 skipped. Final sources are being verified again below.

### C7: Final verification and read-back (completed)

- Final affected diagnostic/platform/SDK/root/bootstrap/error mapping command: **98 passed / 1 skipped**. Transcript: `/private/tmp/craftsky-logging-budget-focused.txt`.
- Full `cd app && flutter test --no-pub`: **2712 passed / 38 skipped** on the corrected runtime. Transcript: `/private/tmp/craftsky-logging-budget-flutter-full.txt`.
- Green refactor: replaced the verbose nested generic test type with equivalent expanding typedefs and corrected three analyzer style infos. Final platform file: **7 passed**. Transcript: `/private/tmp/craftsky-logging-budget-platform-final.txt`. No runtime change followed the full-suite run.
- Final `flutter analyze --no-pub`: **No issues found**. Transcript: `/private/tmp/craftsky-logging-budget-analysis.txt`. `git diff --check`: passed.
- Canonical guide and validation evidence updated/read back. Only the emitter, its platform test, canonical guide and implementation/validation notes changed in this correction stage. No API, lexicon, dependencies, migrations, Go source or production settings changed. Existing full Go/race and second-review scheduled regressions remain applicable; no redundant Go rerun for this Flutter-only correction.
- All C6/C7 progress items complete. Implementation review is the next offered stage, not yet performed; existing review verdict remains historical input. MAN-001 physical-device and MAN-004 live production checks remain Pending. No commit, push, deployment or live telemetry operation performed.

## Refined implementation pass (2026-10-07)

The user approved and authorized SIM-001–SIM-006. Reloaded the requirements, acceptance cases, document review, coding plan and mandatory guide; retained earlier execution history. New contract amendments take precedence over older strict behavior. No commit, deployment or live telemetry authorized.

| Order | Test / requirement | Status |
|---|---|---|
| R1 | SIM-T01 / SIM-001 / FR-006 | completed |
| R2 | SIM-T02 / SIM-002 / FR-008 | completed |
| R3 | SIM-T03 / SIM-003 / FR-016, NFR-002 | completed |
| R4 | SIM-T04 / SIM-004, SIM-005 / FR-001, FR-010 | completed |
| R5 | SIM-T05 / SIM-005 / FR-010 | completed |
| R6 | SIM-T06 / SIM-006 / FR-014, NFR-003 | completed |

### R1: SIM-T01
- Red: AppError diagnostics still contained server prose, validation explanations and resource paths.
- Green: API mapping admits bounded code/requestId plus status/method and original client cause/stack; AppError diagnostics omit server prose/fields/paths/categories. Both mapper suites: 16 passed. Wire envelope and localized business classifications unchanged. Mirrored catalogue removal follows once remaining sink dependencies are removed in R4.

### R2: SIM-T02
- Red: severe supporting log created an issue; after removal, consumed registry-read and switched-account post failures had no owner.
- Green: root forwarding exports logs only; provider/framework/platform callbacks capture explicitly. Removed ownership flags across wrappers and supporting messages. Registry read uses injected guarded reporter before returning the existing empty snapshot; post creation captures a consumed late failure using the initiating context/reporter captured before await. Expected classifications and auth/state outcomes unchanged. Ownership/storage/provider suites: 21 passed; mutation account-switch fixture passed. Temporary compile errors from obsolete flag/category assertions were corrected, not counted as behavioral reds.

R3 verification: SIM-T03 failed with three console chunks, then passed with one parseable brief JSON record; plain token assignments are scrubbed. Console/product/bridge suites: 11 passed. Obsolete chunk reconstruction and generic Flutter retry-window requirements are superseded by the approved refinement.

### R4: SIM-T04
Red: final-hook fixture showed SDK event/stack identity and ordinary app metadata were discarded by reconstruction. Green: capture original exception through SDK; final hooks scrub prose/private enrichment in place. Removed Flutter request/route/server-message catalogues, message registry, Expando admission/event provenance and generic retry manager. SDK native identity deduplication remains enabled: two provider captures of the same error object produce one SDK issue. Independently created failures retain separate captures. Superseded SDK aggregate-size and client-route fixtures were replaced by native-stack, scalar-bound and client separation checks. Actual SDK serialization cause/stack/correlation/private canary suite: 23 passed.

### R5: SIM-T05
Red: new static worker message was replaced by unknown-message omission. Green: local and SDK message scrubbers accept bounded developer text; removed Go global message registry, active nonce map, selected-event snapshots and protected-log marker. Final hooks retain SDK metadata/stacks, filter small custom contexts, and derive vetted issue explanations from the SDK original-exception hint. Typed source selectors/private worker boundaries and targeted Go retry limiter remain. Go observability suite passed.

### R2 additional consumed owners
Removed implicit severe capture revealed consumed optional-caption, autocomplete, device storage and image-picker/pipeline failures. Added explicit guarded reporters at these source boundaries; provider-propagated/video-publication failures remain provider-owned. Red fixtures observed zero captures for facet, caption, device and camera failures; green fixtures retain one capture and unchanged empty/soft-error/UUID/draft fallback behavior. No ownership flags or message deduplication manager were reintroduced.

### R6: SIM-T06 and final verification
- Rewrote the mandatory guide around source privacy, explicit owners, client/server separation, SDK final hooks and one brief console record. Updated callable guide fixtures and marked the old implementation review/evidence as historical.
- Final full Flutter suite: **2708 passed / 38 skipped** (`/private/tmp/refine-flutter-final.log`). Full Go `just test`: **passed**, race detector and local PostgreSQL/MinIO enabled (`/private/tmp/refine-go-final.log`). Started only local test services when the initial broad run found PostgreSQL stopped.
- Full Go identified a storage-label regression in scheduled media open failures; restored `Storage request failed (HTTP 503)` from the typed cause/code. A final scalar fixture failed because `retryable=true` had become string `false`; source sink and final hook now retain native boolean/numeric attributes. Final Go observability `-race` suite passed (`/private/tmp/refine-go-final-focused.log`).
- Final analysis: **No issues found** (`/private/tmp/refine-analyze.log`); `git diff --check` passed. Corrected obsolete client field fixtures and a formatter/edit race before the successful full Flutter run.
- Paired actual synthetic Go API envelope → Flutter mapping → local console/Sentry Logs/issue passed. Saved sanitized fixture and transport output in `evidence/refined-2026-10-07/`. The opt-in paired test is separately verified rather than counted as an ordinary skipped assertion.
- Scoped diagnostic runtime: **9068 → 7220 lines**, a reduction of **1848 (20.4%)**. Flutter diagnostic modules: **3130 → 1636 (47.7%)**. This count excludes small explicit owner injection additions in feature/storage code outside the diagnostic modules.
- All SIM-T01–SIM-T06 items complete. Privacy canaries, positive cause/stack/correlation, account-switch behavior and unchanged business recovery remain covered. Device/live production collection stays separately pending; no commit, push, deploy, API/lexicon/schema or production-configuration change. Ready for implementation review; no review approval claimed.

## Review corrections IR-005–IR-006 (2026-10-07)

The user selected address required changes after the refined implementation review. This authorizes the two privacy boundary corrections; no new approval, commit or production operation is needed. Approved SIM contracts and the canonical guide remain authoritative.

| Order | Test / requirements | Correction | Status |
|---|---|---|---|
| C8 | SIM-T05 / SIM-004, SIM-005, FR-010 / AC-012 | IR-005: private Go thread stack fields | completed |
| C9 | SIM-T04 / SIM-004, SIM-006, FR-010 / AC-012 | IR-006: Flutter scope/hint attachment envelope items | completed |
| C10 | SIM-T06 / SIM-006, NFR-003 / AC-018, AC-019 | Focused/full tests, analysis, guide and evidence | completed |

Setup: reloaded requirements, acceptance cases, approved document review, coding plan, prior execution, current review and mandatory guide. Correct C8 before C9 with separate meaningful red/green tests; use existing selectors and SDK integration/transport capabilities. No catalogues, provenance registries or event reconstruction.

### C8: SIM-T05 / IR-005
- Red: permanent SDK capture/serialization test retained three thread locals/source/path canaries; exception type/release/useful frame positives passed (`/private/tmp/ir005-red.log`). An initial wrong-directory invocation and sandbox build-cache denial were setup errors, not behavioral reds.
- Green: final Go callback excludes thread payloads while leaving native exception stacks protected and useful (`/private/tmp/ir005-green.log`). No source business behavior changed.

### C9: SIM-T04 / IR-006
- Red: permanent actual SDK envelope fixture emitted four items instead of the two permitted event/transaction payloads; both extra items contained the scope attachment canary (`/private/tmp/ir006-red.log`). Hint attachments/screenshots/view hierarchy remained excluded by the existing hook. An initial repo-root Flutter invocation was a setup error, not behavioral red.
- Green: a small SDK integration wraps the selected HTTP/native transport and removes attachment items before loading/delegation. Scope attachment loaders are never invoked, both event and transaction output remain, and exception type/supplied frame/release/operation positives pass (`/private/tmp/ir006-green.log`). Native exception/stack objects are not rebuilt.

### C10: SIM-T06 final verification
- Affected Go observability/middleware `-race`: passed (`/private/tmp/ir005-006-go-focused.log`). Affected Flutter diagnostic/SDK/provider/API suites: **91 passed / 1 skipped** (`/private/tmp/ir005-006-flutter-focused.log`).
- Full `just test`: passed with race detection and existing local PostgreSQL/MinIO (`/private/tmp/ir005-006-go-full.log`). Full `flutter test --no-pub`: **2709 passed / 38 skipped** (`/private/tmp/ir005-006-flutter-full.log`).
- `flutter analyze --no-pub`: **No issues found** (`/private/tmp/ir005-006-analyze.log`). The initial single cascade style info was corrected without changing behavior. Dart formatter completed; its telemetry timestamp write was sandbox-denied afterwards, which did not undo formatting. No dependency/toolchain changes were committed.
- `git diff --check`: passed. The guide describes Go thread exclusion and Flutter attachment transport filtering, including exclusion before loading/caching. Permanent tests demonstrate original exception type/supplied useful stack/release/operation positives alongside protected-value absence.
- IR-005 and IR-006 correction work complete; current review verdict remains the pre-correction finding record until re-review. Device release-console and live production checks remain pending as separate rollout work. No business/API/lexicon/schema/configuration change, commit, push, deployment or live telemetry action.
- All C8–C10 progress items completed. Next workflow choice is implementation review, manual comment or stop.
