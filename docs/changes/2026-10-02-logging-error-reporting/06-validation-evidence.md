# Logging and Error Reporting Validation Evidence

## Login timeline lifecycle — 2026-10-11

Timeline initial loading now checks `ref.mounted` after fetching, before reading overlays. Pagination checks the captured account operation at the same point. The late-fetch regression first reproduced `UnmountedRefException`, then passed with `ApiCanceled` and no issue capture.

Riverpod `StateError` and `UnmountedRefException` reporting remains unchanged. A regression reproduces a loading auto-dispose provider whose future was retained by a read-only consumer, requires its original `StateError` to remain reportable, and verifies retained type/origin stack plus excluded provider-prose canary in serialized local output. Auth startup's manual providers now have stable names to identify previously anonymous failures; the original unnamed loading-disposal warning's application source has not been established.

- Focused timeline, account-boundary, client, provider-logger and observability suites: **106 passed / 1 skipped** (`/private/tmp/craftsky-login-disposal-focused.log`). Analysis: **No issues found** (`/private/tmp/craftsky-login-disposal-analyze.log`).
- Full Flutter suite: **2,844 passed / 38 skipped** (`/private/tmp/craftsky-login-disposal-full.log`). Diff whitespace check passed.
- No iPhone simulator relaunch, physical release-device check, deployment or production mutation was performed. Backend code is unchanged; the full Go race suite passed earlier in this session, as recorded below.

## Account-client startup disposal — 2026-10-11

Account-client initialization now checks whether its build is still mounted after registry and device-ID loading. A disposed build or removed session exits with `ApiCanceled`, using the existing expected-cancellation classifier rather than reporting an unexpected provider failure.

- The session-invalidation regression first failed with `DiagnosticStateError: Account session unavailable`, then passed after the correction. Regressions cover delayed registry loading and session removal during device-ID loading, the replacement client's bearer header, no issue capture, retained public actor DID, and token/device-ID canaries excluded from serialized local output.
- Focused client, provider-logger, session-validation and language-provider suites: **23 passed**.
- Full Flutter suite: **2,842 passed / 38 skipped** (`/private/tmp/craftsky-disposal-flutter.log`). Flutter analysis: **No issues found** (`/private/tmp/craftsky-disposal-analyze.log`). Diff whitespace check passed.
- Full Go `just test`: passed all packages with race detection against local PostgreSQL/MinIO (`/private/tmp/craftsky-disposal-go.log`).
- iPhone simulator relaunch, physical release-device diagnostics and production retrieval were not performed for this correction. No deployment or production mutation was performed.

## PR 127 CI repair — 2026-10-08

GitHub run 37761501772 on 37294149: Flutter failed only SDK-T05's enabled-log case (expected three serialized Logs, received zero); AppView release gate failed Staticcheck S1034 for repeated type assertions and U1000 for the unused capture wrapper. Aggregate PR check failed because these jobs failed. Read run/job logs and check annotations before changes.

Minimal repair: type-switch binding removes redundant assertions; unused private capture helper removed. Flutter transport test waits for the actual serialized three-log batch before SDK shutdown instead of one event-loop turn, preserving real integration, native transport, issue gate, local output and canary assertions. No SDK/product behavior or workflow changes. Narrow verification: exact Staticcheck 2026.2.1 across AppView passed; observability race tests with isolated local PostgreSQL passed; both Flutter bridge cases passed; scoped Flutter analysis and git diff check clean. Logs: `/private/tmp/pr127-staticcheck.log`, `/private/tmp/pr127-go-repair.log`, `/private/tmp/pr127-flutter-repair.log`, `/private/tmp/pr127-flutter-analysis.log`. Remote rerun pending after push.

## PR handoff and merged-tree verification — 2026-10-08

Maintainer authorized committing all changes, pushing and opening a PR. Current main was merged to resolve five conflicts. Retained main's AppView version/changelog (1.0.12), business-profile replacement validation and UI-only 4xx field details; preserved this branch's original diagnostic cause/stack and removal of server prose/route catalogues from client telemetry. No route/envelope/lexicon schema changes from this integration.

Final stable merged-tree checks: `just test` passed (Go 1.27.1, race, local PostgreSQL/MinIO; `/private/tmp/pr-go-final-full.log`); full Flutter suite 2,761 passed, 38 skipped (`/private/tmp/pr-flutter-final-full.log`); `flutter analyze` reported No issues found (`/private/tmp/pr-flutter-analyze.log`); working/staged `git diff --check` passed. These results supersede the earlier assertion-only Flutter failures. The first handoff Flutter run was invalidated by merging while it ran and was stopped; only the fresh resolved-tree run above is final evidence. No live production verification or deployment performed.

## Flutter console visibility — 2026-10-08 (CON-001)

CON-T01 red: INFO disappeared behind the hard-coded WARNING threshold (`/private/tmp/console-t01-red.log`). Green: root forwarding emits INFO/WARNING by default, omits FINE, and opt-in debug FINE retains credential scrubbing (`/private/tmp/console-t01-green.log`). `CRAFTSKY_DEBUG_LOGS=true` is gated by `kDebugMode`; release/profile retain INFO+. Full provider formatting remains unchanged per explicit maintainer preference.

Full Flutter suite: 2,710 passed, 38 skipped and two outdated WARN-only bridge assertions failed (`/private/tmp/console-flutter-full.log`). Updated both assertions to retain INFO, still omit FINE by default, preserve protected canaries and keep native Sentry issue/log gates unchanged. Final focused platform/bridge rerun: seven passed (`/private/tmp/console-final-focused.log`). Analyzer: No issues found (`/private/tmp/console-flutter-analyze-final.log`). Full suite was not repeated after assertion-only updates. `git diff --check` passed. No Go changes in this correction; prior Go evidence applies. Correct-branch device relaunch and live console retrieval remain manual; no running Flutter app was restarted.

## Repository retry escalation — 2026-10-08 (ERR-003)

- ERR-T03: real PostgreSQL lease/source regression red for expired/mixed/source-change severity and hidden source explanation (`/private/tmp/retry-t03-red.log`); green after retry severity and reviewed cause selection (`/private/tmp/retry-t03-green.log`). Original handler/lease errors remain discoverable with errors.Is; stale claims cannot complete jobs.
- ERR-T04: real migrated repository worker + protected local logger + official SDK transport red at threshold (no issue, `/private/tmp/retry-t04-red.log`); green (`/private/tmp/retry-t04-green.log`). Final full-suite assertions additionally retain the reviewed source-change message/reason on native exceptions, original cause type and public actor/operation while excluding protected prose. Before threshold: WARN/no issue. At/after threshold: ERROR/one issue per failed attempt. Durable pending state, attempts and backoff remain unchanged.
- Full `just test` passed with Go 1.27.1, race detector, isolated compose PostgreSQL and MinIO fixtures (`/private/tmp/retry-go-full.log`). `git diff --check` passed. No Flutter edits; prior Flutter evidence applies. No live synthetic Sentry failure, production change or container restart performed.
- Maintainer explicitly chose the existing attempt alert threshold as diagnostic escalation while retries continue. Default is five. `result=exhausted` is diagnostic vocabulary, not a stopped queue state; claim failures are retried on the next poll and stay WARN.

## Completion metadata correction, 2026-10-07

REL-T01 / REL-T02 / REL-001: actual protected follower-growth output and native mock Sentry Logs retain `already_complete` and `craftsky-appview@1.0.11`; unavailable empty release is absent in both sinks. Unknown outcome and unreviewed release canaries stay excluded. Confirmed red/green transcripts: `/private/tmp/rel-t01-red.log`, `/private/tmp/rel-t01-green.log`, `/private/tmp/rel-t02-red.log`, `/private/tmp/rel-t02-green.log`. Shared selection fixes repeated filtering without loosening generic private prose handling. Focused `go test -race ./internal/observability ./internal/followergrowth ./internal/app ./cmd/appview -count=1` passed (`/private/tmp/rel-focused-final.log`). Full `just test` PostgreSQL/MinIO race run completed (`/private/tmp/rel-go-full.log`): every package except API passed. API image admission test `TestScheduledImageReleaseConcurrentUploads` timed out after five seconds; isolated same-toolchain/race rerun passed (`/private/tmp/rel-image-recheck.log`). Full-run timeout is retained as a validation limitation; no image code/test change was made. `git diff --check` passed, guide/evidence updated and implementation plan read back. No Flutter changes, metric/business/release policy changes, live post-change verification or commit/push/restart/deploy.

## Dev error follow-up, 2026-10-07

- ERR-T01 / ERR-001: actual dispatcher + protected sink + mock SDK tests prove WARN, safe required-field explanation, public URI/event ID and `invalid_lexicon`; no issue or body/private canary exported. Native Log serialization initially lost the explanation, then passed after placing reviewed typed rule detail in the log message. Business oversized-blob test retains 11,189,477-byte actual size and original validator cause. Durable projection-worker regression proves the existing `malformed_record` quarantine is still committed with transient cause details available to diagnostics.
- ERR-T02 / ERR-002: local PostgreSQL tests reproduce pure supersession, expiry, mixed handler/lease failures and supersession at the configured alert-attempt threshold. Original handler and lease causes survive errors.Is; stale claims never complete jobs. Selected owner/job operation/attempt survive protected JSON output; private cause prose stays absent. WARN applies only to confirmed pure supersession below the threshold; other cases remain ERROR.
- Focused race suites passed: ingestion/index/schema/tap/observability (`/private/tmp/err-focused-final.log`); subsequent native Log detail regression passed (`/private/tmp/err-remote-green.log`). Full `just test` passed (`/private/tmp/err-go-full.log`), followed by final observability/Tap/schema race suites (`/private/tmp/err-final-diagnostics.log`) and both native WARN Log rejection cases (`/private/tmp/err-lexicon-last.log`). Exhausted validation remains ERROR and exports an issue; protected validator prose canary remains absent. `git diff --check` passed; implementation plan read back. Flutter source unchanged, tests not repeated.
- No migrations, lexicon schema changes, existing PDS record writes, authoring-policy/lease-duration/queue changes, commit/push/restart/deploy. Native SDK export is mock-tested; changed live dev WARN behavior awaits a rebuild. Earlier read-only live error investigation is historical evidence, not post-change live verification.

## Dev diagnostic usability correction, 2026-10-07

- SDK-T05 / SDK-003, SDK-005: protected startup regression confirmed red for missing listener/version; same test green after reviewed address/version selection. Tap metadata regression confirmed red for missing ID/action/recordBytes and misleading omission marker; same test green after typed public metadata selection and quiet exclusion. Transcripts: `/private/tmp/devdiag-startup-red.log`, `/private/tmp/devdiag-startup-green.log`, `/private/tmp/devdiag-metadata-red.log`, `/private/tmp/devdiag-metadata-green.log`.
- SDK-T06 / SDK-006, NFR-002: real count and byte exhaustion retain the explicit budget marker, correlation and parseable bounded JSON. Unsupported private metadata and stack-local canaries remain absent. Initial byte fixture was sanitized before reaching the budget and was corrected to exercise a large typed panic stack.
- Final focused Go race verification passed: `go test -race ./internal/observability ./cmd/appview ./internal/app -count=1`; `/private/tmp/devdiag-focused-final.log`. Full `just test` PostgreSQL/MinIO race verification passed; `/private/tmp/devdiag-go-full.log`. `git diff --check` passed; implementation plan read back.
- Removed stale dev release override; default release follows embedded `appview/VERSION` (currently `craftsky-appview@1.0.11`). Running dev container has not been rebuilt/recreated; live application of these changes is pending `just dev-d`.
- Read-only dev Sentry Logs returned four ERROR entries, matching local logs. Issue APP-VIEW-4 contains two quarantined public records. Read-only local database inspection and a temporary validation probe identified oversized business images and a missing required post sponsorship declaration. Two lease failures recovered; the exact triggering cause is not fully recorded. Details and proposed follow-ups: [07-dev-error-investigation.md](07-dev-error-investigation.md).
- Flutter source is unchanged; Flutter suites were not repeated. No production/device symbolication claim, record mutation, restart, deployment, commit or push. Error/worker fixes are recommendations only.

## IR-007 automatic expected-failure correction, 2026-10-07

User authorized the required review correction. The actual SDK platform regression initially exported eight issues instead of two (`/private/tmp/ir007-red.log`). The final event hook now classifies the original typed throwable when the SDK wraps it with native ThrowableMechanism metadata. Explicit owners still classify with their own operation outcome before capture; no custom ownership flag or forwarding layer was added.

- Actual framework/platform/zone tests suppress cancellation, offline, session expiry, not-found, validation and explicit non-reportability; reportability overrides and unexpected native cause/stack chains export. Protected local output retains all eight occurrences. Privacy canaries remain absent from serialized local/remote output.
- Explicit terminal offline/expiry failures export with request correlation and supplied frames. Direct/wrapped cancellation remains suppressed even with terminal/reportability overrides.
- Focused SDK-T04 automatic tests: 4 passed (`/private/tmp/ir007-green.log`).
- Observability/provider/error suites: 99 passed, 1 skipped (`/private/tmp/ir007-focused.log`).
- Full `cd app && flutter test --no-pub`: 2,711 passed, 38 skipped (`/private/tmp/ir007-full.log`).
- `cd app && flutter analyze --no-pub`: No issues found (`/private/tmp/ir007-analyze.log`).
- `git diff --check`: passed.

Go source is unchanged by this correction, so its previous full PostgreSQL/MinIO race evidence remains applicable. Optional IR-008 breadcrumb severity is not implemented. The prior review remains historical evidence of the pre-correction defect; implementation review of this correction is pending. No commit, push, deployment or production mutation. Device/live symbolication checks remain separately pending; synthetic Dart SDK transports do not prove native/device behavior.

## SDK-led refinement validation, 2026-10-07

This section supersedes conflicting earlier evidence about custom forwarding, flattened exception summaries and SDK integration ownership. The approved SDK-001–SDK-006 amendment is the contract for this pass. Results below are local synthetic SDK serialization/regressions, not live collector/device proof.

- Native Flutter AppError → API → parsing chains retain concrete types, caught decode frames and returned request correlation, with no custom failure context or cause JSON in Logs.
- Actual Flutter SDK framework/platform integrations capture each occurrence once, preserving mechanism and stacks while local callbacks remain protected and independent.
- Go normal native chains retain attached origin stacks/mechanisms; stackless ordinary errors acquire no invented origin stack. Recovered panic stacks remain intact. Cyclic/wide graphs use a bounded fallback with an explicit omission marker.
- Reviewed static application explanations survive both sinks through DiagnosticStateError/WrapError. Unknown payload-bearing dependency prose stays generic; no per-message registry. StateError and Go Unwrap catch/classification semantics are preserved.
- Matching SDK-version official logging integrations emit Logs without issues. Flutter operation/request/workflow references remain on native issues and supporting breadcrumbs. Go retry Logs keep a scalar concrete error type and vetted HTTP status/SQLSTATE; native issues carry full chains/stacks. No duplicate chain/stack log JSON.
- Native navigation observer tests retain static route patterns and hyphenated screen names while removing route arguments (including private draft and auth proof canaries). Root, authenticated-shell and tab observers are configured with transactions disabled. Selected lifecycle/connectivity and operation/HTTP breadcrumbs survive final filtering. Go request histories remain isolated without tracing.
- Privacy positives and negative canaries cover credentials, dependency prose/private payloads, private route arguments, request/header/body enrichment, frame locals/source, attachments and native symbolication metadata. Image/symbol addresses, native/package/platform fields and stack snapshot/language survive stripping; release symbolication itself remains a separate live check.

Final command results:

- `cd app && flutter test --no-pub`: Passed, **2,707 passed / 38 skipped** (`/private/tmp/sdk-flutter-full-final.log`). The final two-field stack metadata correction subsequently passed all **19 SDK capture/sanitizer tests** (`sdk-flutter-last-focused.log`).
- Flutter observability + router + legacy breadcrumb regression suite: **243 passed / 8 skipped** (`sdk-flutter-final-focused.log`).
- `cd app && flutter analyze --no-pub`: Passed, **No issues found** (`sdk-analyze-complete.log`); final source analysis also passed with **No issues found** in `sdk-analyze-last.log`.
- Go final focused race suite for observability/middleware: Passed (`sdk-go-complete-focused.log`). The final complete PostgreSQL/MinIO `just test` rerun **passed all packages with the race detector** (`/private/tmp/sdk-go-all-green.log`), including final scalar normalization and scheduled retry regressions.
- `git diff --check`: Passed. No unrelated generated changes retained; affected provider hashes and router observer generation are included.

Earlier failed broad runs exposed one legacy navigation-message expectation, the loss of brief retry cause fields after duplicate chain JSON removal, and SDK-native int64 HTTP status normalization; all have permanent regression corrections. A filtered generator run temporarily omitted other generated parts; complete generation restored them, and unrelated generated drift was removed. No unresolved setup workaround is being committed.

No API/envelope, schema, lexicon, auth/purchase policy, retry/disposition, production gates/access/retention or infrastructure change. Dependencies add only matching-version official Dart logging and Go slog packages. No commit, push, deployment, native/device retrieval or production synthetic failure performed.


## IR-005–IR-006 correction verification (2026-10-07)

Authoritative results for the final correction pass; earlier refined results and prior execution below remain history.

- IR-005 permanent Go SDK capture/serialization regression failed on thread locals/source/path canaries, then passed after thread payload exclusion. Native exception type, useful frame/file and release survive. Transcripts: `/private/tmp/ir005-red.log`, `/private/tmp/ir005-green.log`.
- IR-006 permanent actual Flutter issue/transaction envelope regression failed with four items (two attachment payloads), then passed with exactly the two permitted payloads. Exception type/supplied frame/release and transaction operation survive. Scope attachment loaders remain uncalled; hint attachments/screenshots/view hierarchy and exception-prose canaries are absent. Transcripts: `/private/tmp/ir006-red.log`, `/private/tmp/ir006-green.log`.
- The Flutter SDK integration wraps the HTTP/native transport selected by SDK initialization and removes attachment items before loading or delegation. No event reconstruction, catalogue, provenance registry or ownership flag was introduced. Go removes threads at its existing final callback.
- Focused Go race suites: passed (`/private/tmp/ir005-006-go-focused.log`). Focused Flutter: **91 passed / 1 skipped** (`/private/tmp/ir005-006-flutter-focused.log`).
- Full Go `just test`: passed, local PostgreSQL/MinIO and race detection (`/private/tmp/ir005-006-go-full.log`). Full Flutter: **2709 passed / 38 skipped** (`/private/tmp/ir005-006-flutter-full.log`). Analysis: **No issues found** (`/private/tmp/ir005-006-analyze.log`). Diff whitespace check passed.
- Test scope/hint attachments and transports use synthetic canaries and `example.invalid`; no network telemetry or production action. Physical release-device and production retrieval/access/retention checks remain pending. Implementation is complete; review approval is not claimed.


## Refined implementation validation (2026-10-07)

This section is authoritative for the approved SIM-001–SIM-006 refinement. Earlier results below describe the preceding implementation, including superseded client catalogue, ownership-marker, exact chunk-budget and generic retry-manager behavior. They are retained as history, not current requirements or current sink guarantees.

- Full Flutter `flutter test --no-pub`: **2708 passed / 38 skipped**. The opt-in Go/Flutter correlation fixture is separately verified; ordinary skips remain visible. Transcript: `/private/tmp/refine-flutter-final.log`.
- Final Flutter `flutter analyze --no-pub`: **No issues found**. Transcript: `/private/tmp/refine-analyze.log`.
- Actual paired Go error envelope → Flutter API/AppError mapping → single local console JSON → Sentry Logs and issue: **passed**, using synthetic local/mock transport data. Client issue correlation is in its small `operation` context. Fixture/evidence: `evidence/refined-2026-10-07/`. Transcripts: `/private/tmp/refine-correlation-go.log`, `/private/tmp/refine-correlation-flutter.log`.
- Focused Go observability suite with `-race`: **passed**, including static developer text, typed numeric/boolean SDK attributes, typed cause/status, private worker context, local fallback and SDK enrichment canaries. Transcript: `/private/tmp/refine-go-final-focused.log`.
- Full Go `just test` with race detection and local PostgreSQL/MinIO: **passed**. An earlier broad run identified the storage-specific HTTP label regression, corrected by deriving the label from the typed cause rather than restoring a message registry. Transcript: `/private/tmp/refine-go-final.log`.
- The broad Flutter run exposed obsolete field fixtures after removing client server-prose/path/category properties. Updated them to stable code/status/method/request correlation; retained private notification/saved-post canaries. A formatting fix raced with an edit and damaged three identifiers; corrected and reran analysis/full Flutter.
- Removed Flutter server-envelope/route mirror, route generator, diagnostic-message registry, generic retry limiter, SDK Expando provenance and ownership properties; removed Go global message registry, nonce registry, selected-event snapshots and protected-log marker. The scoped diagnostic runtime count fell from 9068 to 7220 lines (1848 removed; 20.4%); Flutter diagnostic modules fell from 3130 to 1636 lines (47.7%). Source-owned explicit captures outside those modules add a few small injected reporter dependencies.
- Developer static messages are trusted by code review and scrubbed/bounded at sinks. Arbitrary private prose is excluded at the source; unknown exception prose remains generic. Native SDK exception identity deduplication remains enabled. SDK issue stacks/standard metadata remain richer than the brief console record. No exact console/SDK reconstruction parity or total SDK-envelope budget is claimed.
- `git diff --check`: **passed**. Guide and callable Go/Flutter examples reflect the refined contract. Physical release-device collection and live production retrieval/access/retention checks remain separately pending; no production mutation, live telemetry, API/lexicon/schema change, commit or push occurred.

Evidence date: 2026-10-05. Reviewer: implementing agent. This records local implementation verification, not production or release approval. The authoritative ordered TDD execution and red/green notes are in `05-implementation-plan.md`.

All transport artifacts use synthetic fixtures, local buffers, injected platform sinks or SDK test transports. No live Sentry/Render inspection, error publication, production mutation, deployment, commit or push occurred. The user explicitly approved failure-only initiating account DID and existing non-capability workflow reference on private operational failures; private payload/recipient/target membership and success history remain excluded.

## Final verification

- Full Flutter `just app-test --no-pub`: Passed, **2708 passed / 38 skipped**. The paired correlation fixture is opt-in and separately executed, not treated as a missing ordinary suite assertion. Existing environment/platform/optional integration skips remain visible in test output.
- Final SDK/correlation/bootstrap command after the last measurement-selection change: Passed, 16 tests including the opt-in paired correlation fixture. Account-initialization UI/evidence command: Passed, 11 tests.
- Full Go `just test`: Passed, race detection and local PostgreSQL/MinIO integration enabled. The final unsupported-module test also passed separately.
- `cd app && flutter analyze --no-pub`: Passed, **No issues found**. The normal analysis recipe attempted dependency refresh and failed; resolved-dependency analysis is the verification command. No dependency workaround committed.
- `git diff --check`: Passed at final cleanup checkpoint. `pubspec.yaml`, lockfiles, migrations, lexicons and production infrastructure/configuration unchanged.
- Earlier broad-run failures were resolved with positive retained-field assertions and existing privacy canaries: lifecycle Logs export, correlation moved from tags to context, required service metadata and reviewed static API pin messages. Local MinIO private-bucket bootstrap was required. The final tests preserve existing auth/fencing/retry/ACK/UI behavior.

## Source finding inventory

Selection before formatting, sanitization and bounds applies in every sink. Public identity never grants permission to emit arbitrary error prose. Listed evidence includes actual source/SDK payload tests, not just central-helper coverage.

| Finding | Verified implemented behavior | Evidence |
| --- | --- | --- |
| F-01 | Concrete/wrapped Go cause, vetted explanation/SQLSTATE/status; unknown prose excluded, no invented origin stack | UT-001/003, AT-001, IT-006 |
| F-02 | HTTP/Tap recovery retains concrete panic type, safe runtime explanation and recovery frames; failed quarantine persistence remains unacknowledged | UT-002, AT-002, IT-003, Tap quarantine-store panic regression |
| F-03 | Go run/request ID survives local/Logs/issue/API/Dio/AppError/Flutter without sampled transaction; available trace/span and process metadata retained | UT-006, AT-003, IT-004 paired artifacts |
| F-04 | Ordinary and probe HTTP receipt/completion include method, classified actual path, pattern/status/bytes/duration/ID; private/query/capability segments excluded | UT-007, AT-004, IT-001/002 |
| F-05 | Shared HTTP/auth 5xx helpers receive original cause and selected source workflow; marker prevents fallback duplicate | AT-001, IT-006/007 |
| F-06 | DB/request cause and SQLSTATE retained; no SQL/row detail, same issue ownership marker | IT-003/006 |
| F-07 | Real Riverpod provider preserves error/stack and typed initiating lease; source-owned swallowed/stale mutations retain their source | IT-009/010 and serialized provider fixture |
| F-08 | Root bridge retains selected fixed message/severity/operation/cause/stack; Logs independent from issue owner | UT-008, AT-005, IT-005/012 |
| F-09 | Closed public/attempted/published-parse/private-failure adapters carry permitted source context outside aggregate tags | UT-004/005/007/015, AT-006, IT-011 |
| F-10 | API route manifest and vetted envelope retain code/message/field reasons/status/method/path/request ID through mapping; raw response excluded | UT-009, IT-008, REG-001/002, pin API regressions |
| F-11 | Account initialization/provider/gate retain original cause/stack/stage with recovery UI unchanged | IT-010, account initialization fixture and stale-lease regressions |
| F-12 | Protected direct slog, root/platform and final SDK selectors omit unsupported enrichment; bounded cause/context/frames/log output | UT-012/013, AT-007, IT-011; unsupported transaction measurements excluded |
| F-13 | Callback/provider/source owners emit one issue; expected overrides/cancellation preserved, supporting severe records issue-owned | UT-010/011, IT-009; startup/reporter regressions |
| F-14 | PDS reads/writes/direct list, Tap projection/backfill and terminal published parse retain available public references and durable outcome | IT-006, IT-016; unavailable dependency origin omitted, no enrichment fetch |
| F-15 | Video preparation/upload/poll/caption/playback and image/picker failures retain cause/stack/stage; credential/media/path excluded; soft UI policy unchanged | IT-010 and caption handler/player regressions |
| F-16 | Scheduling/deletion/push/Instagram source failures retain failure owner/job/stage/attempt/cause; claim omits unavailable owner; persistence joins preserve originating cause | AT-006, IT-007; scheduling finalization, deletion safety/claim, push finalization, Instagram claim/retry persistence tests |
| F-17 | Auth handoff/storage/device/facets retain selected cause/native/API code/stage without private values or query/model dump | IT-010, storage/handoff/facet/video privacy regressions |
| F-18 | Explicit model summaries retain public references only under public-workflow declaration; coarse count/state/flags otherwise | UT-014, serialized SDK summary and documentation examples |
| F-19 | Private models/session/handoff/video/device/recipient representations remain protected across selected sinks | REG-005, AT-006/007 and protected model/video regression matrix |
| F-20 | One startup root subscription; local output independent of DSN/export; retry admission shared; capture/export/flush/local-sink faults isolated without recursion | IT-012/013/015/016, REG-004/006; physical release output remains MAN-001 Pending |

## MAN-002 contributor guide review

Status: Passed implementer manual source/usability review; maintainer implementation review not yet performed.

Start at root `AGENTS.md`, follow its mandatory canonical `docs/development/logging-error-reporting.md` link. For a public API DB failure choose the existing request context and `ReportRequestFailure` or cause-bearing `LogDiagnostic` plus designated `CaptureDiagnostic`; for a private worker choose failure-only `ObservePrivateFailure`; for a Flutter provider let `ProviderLogger` own the issue and make the supporting log issue-owned. Explicit feature capture keeps the caught exception/stack and fixed `DiagnosticMessage`.

Reviewed the severity/expected-outcome table, correct/incorrect Go/Dart examples, public/private/credential boundary, correlation, classified actual path versus route pattern, URL origin treatment, source context/stack, bounds/retry policy, sink gates and verification checklist against the final interfaces. IT-014 executes equivalent examples and negative canaries. Numeric spacing corrected during review; no optional skill is needed to follow the guide.

## MAN-003 local investigation evidence

Status: Passed local synthetic investigation. Toolchain: Go 1.27.1; Flutter 3.44.9 / Dart 3.12.2; Sentry Go 0.49 / Flutter SDK 9.23. Collector: JSON slog buffers, injected platform sink and SDK mock/serialized transports. Physical console/live collector not used.

Commands (Go commands run in `appview/`, Flutter command from repo root):

```sh
CRAFTSKY_DIAGNOSTIC_EVIDENCE=1 go test ./internal/api ./internal/tap ./internal/scheduledposts -run 'TestAuthorListDiagnosticRetainsCauseAndPublicTarget|TestTapTerminalRejectionRetainsPublishedFailureExcerpt|TestIT007ScheduledSnapshotFailureRetainsReturnedCause' -count=1 -v
CRAFTSKY_CORRELATION_FIXTURE=/absolute/path/to/evidence/paired-correlation-go.json go test ./internal/api -run TestLoggingErrorReportingRequestCorrelation -count=1
CRAFTSKY_CORRELATION_FIXTURE=/absolute/path/to/evidence/paired-correlation-go.json CRAFTSKY_DIAGNOSTIC_EVIDENCE_DIR=/absolute/path/to/evidence just app-test --no-pub test/observability/request_correlation_test.dart test/observability/sdk_emission_test.dart test/observability/sentry_bootstrap_test.dart
CRAFTSKY_DIAGNOSTIC_EVIDENCE_DIR=/absolute/path/to/evidence just app-test --no-pub test/app_test.dart
```

The opt-in artifact path is a test output directory, never a telemetry field. Assertions run before evidence write. Test transport uses a synthetic `example.invalid` DSN and never sends network telemetry. Local tests vary Logs on/off and sampled/unsampled gates independently; fixture-level SDK gates are explicit in the tests.

| Artifact | Gates / retained excerpt | Investigation from emitted data |
| --- | --- | --- |
| [Go public read, Tap, scheduled failure](evidence/go-source-diagnostics.txt) | Public-read/Tap: Logs and issue mock enabled, test environment. Scheduled snapshot: DSN absent, local-only. Public read keeps SQLSTATE40001 and distinct author target; Tap keeps published excerpt with credential redacted, event1/record URI/NSID and durably_committed acknowledgement; schedule keeps SQLSTATE08006, snapshot stage, initiating did:plc:owner and workflow38a405b6-0a2e-4706-8e69-21ce5b218e41 | Locate cause, then selected diagnostic target or safe event/job ref. Public read is query failure; Tap is malformed_record quarantine with durable ACK; schedule is private snapshot access failure. No private payload/lease/recipient fields or invented HTTP trail |
| [Paired Go request](evidence/paired-correlation-go.json) | Local, mock issue and Logs enabled, environment test/release test-release, zero sampled transactions. requestId14636b35-17df-421a-8c8e-e7b413e47fda; incoming public path/vetted target; SQLSTATE23505/query/post.get | Read API envelope requestId; find matching receipt/error/completion run_id and issue correlation context. Available trace/span references are diagnostic context, not grouping tags |
| [Paired Flutter request](evidence/paired-correlation-flutter.json) | Actual Dio mapping and final serialized local/Log/issue; server status500, internal_error/post read failed, same request ID, classified posts route | Follow the exact ID from Go into appViewRequestId; find supplied loadPublicRecord frame and ApiServerError. Request classification retains public path; no body/query/header copied |
| [Flutter provider](evidence/flutter-provider.json) | Local and serialized SDK, independent StateError occurrences with readPublishedPost frame; expected auth/offline failures generate no unexpected issue | Locate provider stage/coarse classification, original type and named source frame; two independent failures survive. No guessed account/target when unavailable |
| [Flutter account initialization](evidence/flutter-account-initialization.json) | Local severe account initialization, DSN absent; StateError, initialization stage and bounded64frames in3chunks | Join ordered diagnostic chunk strings, decode JSON, locate initialization stage and source frame. Original secret/DID/preferences prose excluded; account-recovery UI confirmed by11passing tests |
| [Telemetry initialization fallback](evidence/flutter-initialization.json) | SDK initialization throws; safe local warning/no-op reporter, no export | Locate StateError and _FakeSentryBootstrapAdapter.initialize / ObservabilityBootstrap.initialize frames; fixed Telemetry reporting failed message, no recursion/private prose |

Saved artifacts were checked for fixture private/secret canaries and absolute user paths; none remain. The Tap published excerpt deliberately contains a redaction marker. Selected cause/type/code/stack and applicable references are actionable; arbitrary unknown prose is intentionally unavailable. SDK/public module metadata is build provenance, not a private data payload. These are synthetic test findings, not evidence of operator access or production retention.

## MAN-001 attached release-device verification

Status: **Pending / blocked by unavailable attached physical test devices (GAP-003)**. Read-only `flutter devices --machine` found iOS simulator, macOS and web, zero physical iOS/Android devices. No identifiers/names saved here. Adapter/startup tests prove selected release-policy dispatch; they do not prove a compiled release app's console retrieval.

Maintainer must build/run supported iOS and Android release test configurations, record build/platform/collector, and trigger synthetic WARNING/ERROR before initialization, with SDK disabled and unavailable transport. Capture bounded cause/frames/correlation and repeat with canaries. Confirm no persisted app buffer/support UI. Leave this Pending until actual sanitized device evidence exists.

## MAN-004 separate production checklist

Checklist authoring: Complete. **Every execution item below is Pending/unverified (GAP-004)**. Perform only during a separately authorized rollout, using ordinary deployed diagnostics. Do not inject synthetic production failures, mutate configuration or infer authorization from this implementation.

- [ ] Record deployed semantic version, immutable release, environment and effective DSN/Logs/error/tracing gates for Go and Flutter; do not expose DSN credentials.
- [ ] Retrieve an ordinary request's receipt/completion from the actual configured log sink; verify classified incoming path versus route pattern and protected-value exclusion.
- [ ] Verify the unchanged API requestId/run_id through an ordinary unsampled request and available Flutter diagnostic; record available trace/span context without inventing it.
- [ ] Retrieve ordinary exported Logs and an existing actionable issue using intended operator access; verify cause/source/stage/permitted reference and capture ownership. Logs retrieval remains distinct from issue/tracing gates.
- [ ] Confirm production operator permissions, retention/access policy and suitability for public identifiers and narrowly permitted private failure owner/job context. Record policy/access gaps; do not change either implicitly.
- [ ] Record date/build/environment/status, collector, safe correlation, sanitized excerpts, investigation steps and reviewer. Failed retrieval remains Failed/Pending, not source-level proof of production behavior.

No rollout/release approval is implied. Physical release console validation and actual production retrieval remain the material external verification gaps.

## Review correction validation (2026-10-06)

The four first-review findings were addressed in separate red/green corrections. The second review closed IR-001, IR-002 and IR-004 but demonstrated another accepted aggregate-budget combination for IR-003. The following table records the first correction pass; the second-pass evidence follows below.

| Finding | Permanent regression and observed correction | Result |
|---|---|---|
| IR-001 / IT-015 / FR-016 | `app/test/observability/product_platform_log_test.dart` compiles and runs `test/fixtures/platform_log_product_probe.dart` against the real default adapter. Original developer.log emitted only the completion marker. Dart print now retains WARNING, cause and correlation in product AOT output. | Passed; actual iOS/Android attached-device retrieval remains MAN-001 Pending. |
| IR-002 / IT-007 / RULE-002 | `TestIT007ScheduledPrepublicationPDSOmitsPrivateTargetAcrossSinks` uses a full migrated local schema and observed fenced command/PDS chain. Private dependency and expected not-found diagnostics omit the unpublished key/URI while retaining permitted failure account/job/stage/cause fields. Classified public PDS positive fixtures remain green; unclassified background PDS context is conservative. | Passed, including race run. |
| IR-003 / UT-013 / NFR-002 | Combined cause/stack/context/validation fixture now decodes the joined JSON and asserts <=6500 selected bytes, <=8 chunks and <=2048 bytes per complete wrapper, explicit omissions, cause, useful frame, request ID and private workflow UUID. Whole core reduction precedes workflow admission; final metadata participates in the budget. | Passed. |
| IR-004 / IT-007 / FR-001, FR-009 | `TestIT007ScheduledMediaIOCausesSurviveAcrossSinks` independently injects Open/Read/Close faults, retaining Smithy ResponseError, net.OpError and fs.PathError. Local and mock exported sinks retain the vetted `Storage request failed (HTTP 503)` explanation, exclude object/header/session/media canaries, and preserve persisted retrying/object_unavailable versus needs_attention/media_invalid decisions. | Passed, including race run. |

Final verification:

- Full Go `just test`: passed race/local PostgreSQL/MinIO integration; transcript `/private/tmp/craftsky-logging-corrections-go-full.txt`.
- Full Flutter `cd app && flutter test --no-pub`: **2710 passed / 38 skipped** on final corrected sources; transcript `/private/tmp/craftsky-logging-corrections-flutter-full.txt`. Existing optional/environment/platform skips remain visible; no device coverage is inferred from this count.
- Flutter `flutter analyze --no-pub`: **No issues found**; transcript `/private/tmp/craftsky-logging-corrections-analysis.txt`.
- Affected Go packages passed separately with race/local PostgreSQL integration; transcript `/private/tmp/craftsky-logging-corrections-go-focused.txt`.
- Canonical guide updated to match corrected entry points, release sink and whole-record/wrapper budgets. `git diff --check` passed.
- Tests use synthetic local/mock data. No physical device output, effective live Sentry/Render gates, production retrieval, deployment or retention/access verification is claimed. Existing MAN-001/MAN-004 pending statuses remain unchanged.

## Second review aggregate-budget correction (2026-10-06)

IR-003 / UT-013 / FR-010, NFR-002 / AC-013:

- Permanent priority-field regression: long bounded concrete cause types, maximum classified path/private owner, ordinary request/job correlation, production/release/suppression metadata, minimum/absent stack. Initial red: **7626 selected bytes >6500**, with complete decoded JSON. Final focused green retains useful type/operation/correlation/frame and process fields, omits private-cause canary, and checks complete JSON/wrappers/chunk payloads/count.
- Permanent compact-fallback regression: oversized multibyte custom Level name initially caused unterminated joined JSON. Bounded compact scalar admission fixes this; positive ordinary owner/job/request/cause/frame/operation/suppression fields survive and arbitrary cause prose stays absent.
- Structural selection yields whole path detail, then intermediary cause wrappers, then a compact schema with explicit omission markers. No partial identifier is emitted as a complete ID. Normal-sized request/job references remain protected by positive oracles.
- Final affected command: `cd app && flutter test --no-pub test/observability test/bootstrap/provider_logger_test.dart test/shared/api/providers/error_mapping_interceptor_test.dart test/shared/errors/app_error_mapper_test.dart`: **98 passed / 1 skipped**. Transcript: `/private/tmp/craftsky-logging-budget-focused.txt`.
- Full `cd app && flutter test --no-pub`: **2712 passed / 38 skipped**. Transcript: `/private/tmp/craftsky-logging-budget-flutter-full.txt`. Runtime source was unchanged after this run; equivalent generic fixture typedefs/style cleanup were followed by a final platform-file rerun: **7 passed**, transcript `/private/tmp/craftsky-logging-budget-platform-final.txt`.
- Final `flutter analyze --no-pub`: **No issues found**. Transcript: `/private/tmp/craftsky-logging-budget-analysis.txt`. Three initial test-only style infos were corrected. `git diff --check` passed.
- Go source/tests are unchanged in this pass; preceding full Go/race and second-review scheduled regression results remain applicable. MAN-001 physical device and MAN-004 live production checks remain Pending; no commit/push/deployment/live telemetry action. Implementation completion awaits another review; no review approval is claimed.

## Recent Sentry account-lifecycle corrections (2026-10-11)

- Read recent issues and latest events using the authenticated Sentry CLI. Flutter issues FLUTTER-APP-P/Q/S/T reported ended account authority or provider disposal as StateErrors; FLUTTER-APP-R accessed timeline overlays through a disposed Ref.
- AccountOperationCanceled retains StateError catch semantics and original typed cause, but is expected cancellation in explicit and automatic reporting. Timeline build/pagination check account authority before reading overlays. Account Dio checks disposal after registry loading; profile error recovery checks mounting before reading its fence. Subscription response identity mismatches remain reportable.
- Focused Flutter provider and observability suites: **104 passed / 1 skipped**. Transcript: `/private/tmp/craftsky-sentry-focused.log`. Explicit serialized SDK tests and the automatic event hook suppress account cancellation while existing positive/error and protected-value checks remain green.
- Full Flutter suite: **2847 passed / 38 skipped**. Transcript: `/private/tmp/craftsky-sentry-full.log`.
- Flutter analysis: **No issues found**. Transcript: `/private/tmp/craftsky-sentry-analyze.log`.
- AppView issue APP-VIEW-6 was a development repository reconciliation attempt with `reason=source_changed`, `result=exhausted`, `attempt=10`; the checked-in retry escalation policy intentionally retains these issues. APP-VIEW-7 was a production Firebase push failure with `result=expected`. A regression reproduced one unwanted issue for that outcome. ExpectedDiagnostic now suppresses `expected` as well as `retry`, retaining local failure logs and terminal issue capture. APP-VIEW-5 was a production PDS repository-head failure without a retained specific cause/status, so no speculative backend fix was made.
- Physical-device validation of these fixes and post-release Sentry recurrence checks remain **Pending**. No production failures were injected. No deployment, production configuration change or Sentry issue-status mutation was performed. The backend classifier regression failed before the fix (`issues = 1, want 0 for expected`); the complete observability package passed with the race detector (`go test -race ./internal/observability -count=1`, transcript `/private/tmp/craftsky-sentry-go-green.log`). The complete `./scripts/appview-check` gate passed, including full PostgreSQL/MinIO integration and race runs, module/codegen/lint/source and binary vulnerability checks, Docker builds, migration-failure checks and service smoke tests. Transcript: `/private/tmp/craftsky-sentry-appview-check.log`; artifacts: `/private/tmp/craftsky-sentry-appview-check`. Pinned Tap handshake passed; deterministic event injection remains unavailable in Tap v0.1.10, as recorded by the gate.

- Main synchronization: fast-forwarded to `1d897590` and resolved four conflicts restoring local work. Retained main's ApiCanceled behavior and earlier session check in account Dio, both sets of lifecycle regression tests, and the additional account-authority fence in timeline build. Post-integration provider/observability suites: **188 passed / 1 skipped** (`/private/tmp/craftsky-main-sync-tests.log`); analysis: **No issues found** (`/private/tmp/craftsky-main-sync-analysis.log`); `git diff --check` passed. Backend source and tests did not change during integration; the preceding complete AppView gate remains the backend evidence. The earlier full Flutter result predates this synchronization; the full Flutter suite was not repeated.

## Device error-reporting consent (2026-10-11)

- Error reporting is off by default. The persisted device preference is loaded before Sentry initialization and shared between onboarding and Settings → Preferences. A stable reporter applies runtime opt-in/opt-out to existing explicit error owners; SDK shutdown also stops automatic integrations.
- Local SDK transport regressions verify no initialization/export without consent, protected exception export after opt-in, no direct or reporter export after opt-out, persisted choices after restart, re-enabling, missing-DSN configuration, and storage-write rejection without starting the SDK. Shared-control tests verify both onboarding and settings reflect and persist the same preference.
- Onboarding tests verify Finish is disabled until the mandatory age-and-policy declaration is selected, returns to disabled when unchecked, and does not require optional error-reporting consent.
- Full Flutter suite: 2847 passed / 38 skipped (`/private/tmp/craftsky-flutter-full-tests.log`). Final consent-specific tests: 5 passed, covering the final persistence guard (`/private/tmp/craftsky-consent-final-tests.log`). `flutter analyze --no-pub`: no issues found. `git diff --check`: passed.
- Physical iOS/Android SDK startup/shutdown, native crash collection, release console retrieval, and live Sentry delivery remain unexecuted. No production telemetry was injected or inspected, and no deployment, server telemetry configuration, retention, or access controls changed. Go source/tests are unchanged.

- Consent branch main synchronization: merged `origin/main` at `04e144b9`, preserving both independently appended validation sections. Code merged without conflicts. Post-merge onboarding/settings/observability and affected account-lifecycle suites: **267 passed / 5 skipped** (`/private/tmp/craftsky-onboarding-main-merge-tests.log`); Flutter analysis: **No issues found** (`/private/tmp/craftsky-onboarding-main-merge-analysis.log`); Go observability package with race detector: **passed** (`/private/tmp/craftsky-onboarding-main-merge-go.log`). `git diff --check` passed. The full Flutter suite and full AppView gate were not repeated for this documentation-only conflict resolution; preceding full-suite results describe their respective pre-merge states.

## Provider cancellation log severity (2026-10-12)

- Known account/API/Dio cancellations, including app-owned cause wrappers, now log at FINE in the provider observer. Default console output omits them; opt-in debug output retains the original typed cause and stack. Other provider failures retain WARNING and their existing issue classification.
- Focused observer, cancellation, account-client, timeline, sanitizer and SDK emission suites: **62 passed** (`/private/tmp/craftsky-cancellation-tests.log`). Regressions verify default/debug console thresholds, retained cause types, private-value canaries, zero cancellation captures, and continued reporting of framework disposal misuse. Flutter analysis: **No issues found** (`/private/tmp/craftsky-cancellation-analysis.log`).
- Full Flutter suite: **2919 passed / 38 skipped** (`/private/tmp/craftsky-cancellation-full-tests.log`). `git diff --check`: passed.
- Live login/device console verification and production Sentry delivery were not performed. No deployment or telemetry configuration changed; Go source/tests are unchanged and backend checks were not rerun.

## Open Sentry issue review (2026-10-16)

- Used the Sentry CLI to inspect all 16 unresolved issues across both CraftSky projects. Findings, issue links, local repairs and remaining operational uncertainty are recorded in [the triage note](../../development/sentry-triage-2026-10-16.md).
- Expected account-discard signals retain their existing exception types and catch behavior through a shared cancellation marker. Riverpod-wrapped cancellations are recognized by the provider owner and Sentry final hook. Invalid login input remains UI validation; storage/browser/unknown failures remain reportable. Eligibility, identity, author-post and bookmark continuations check their lifetime before using refs.
- Stale OAuth request callbacks retain their existing refusal behavior and log diagnostics without creating unexpected issues. Database-load failures have a distinct reason and keep their original cause. The real HTTP/Sentry boundary test verifies no capture for a consumed callback request, retained diagnostic facts and absence of callback credential canaries; a closed-pool regression verifies storage failures remain distinct from invalid flows.
- Final focused Flutter run: **145 passed / 9 existing skips** (`/tmp/craftsky-sentry-review/flutter-final.log`). Flutter analysis: **No issues found** (`/tmp/craftsky-sentry-review/flutter-analyze.log`). The bookmark regression reproduces the original unmounted-ref error with the guard removed and passes after restoration. Bundled-font tests load every configured asset.
- Complete AppView release gate: **passed**, including full integration and race runs (**6045 passed / 0 skipped** each), module/codegen/vet/Staticcheck, source/binary vulnerability checks, Docker builds, migrations, media memory checks and service smoke tests. The final consumed-callback fixture was also separately rerun against isolated PostgreSQL with the race detector and passed. Artifacts: `/tmp/craftsky-sentry-review/appview-check/`; transcript: `/tmp/craftsky-sentry-review/appview-check.log`. Pinned Tap handshake passed; deterministic event injection remains unavailable in Tap v0.1.10.
- Physical-device validation, production rollout, post-release recurrence checks and production symbolication remain **Pending**. Live existing Sentry events were read; no production failures were injected and no production configuration/data, deployment, or Sentry issue status was changed.

## Production failure diagnostics (2026-10-16 follow-up)

- For APP-VIEW-5/7/8, preserve original profile/Instagram rejection causes before terminal-response serialization; export reviewed PDS/Firebase machine codes and numeric upstream HTTP statuses in protected local logs and Sentry events. Correlate with existing request/work references and traces. Repository-head missing-CID validation has a fixed recognized explanation. PDS failure spans retain provider facts, including root transaction attributes previously lost during filtering.
- Final focused race tests passed across observability, API, auth and app packages (`/tmp/craftsky-production-diagnostics-final-focused.log`). Real Firebase SDK errors come from an injected local transport, with positive FCM/platform code and status assertions, original native causes and protected body/header/routing/wrapper canaries. The actual profile handler regression keeps its 502 envelope, replaces the generic issue with the original rejection, retains supporting logs when a lower layer already owns the issue and verifies no duplicate captures. Instagram adapter, missing-CID client and root-span filtering regressions also pass.
- The initial gate stopped on module tidiness after a test began importing the already-pinned Google API helper directly; `go mod tidy` only promoted that existing dependency from indirect to direct, without a version change. The complete final `GOFLAGS='-p=1' ./scripts/appview-check` gate **passed**: **6059 integration cases / 6059 race cases**, zero skips; module/codegen/vet/Staticcheck, source/binary vulnerability checks, Docker builds, migrations, media memory budgets and service health checks passed. Transcript: `/tmp/craftsky-production-diagnostics-gate-final.log`; artifacts: `/tmp/craftsky-production-diagnostics-gate-final/`. Pinned Tap handshake passed; deterministic event injection remains unavailable in Tap v0.1.10.
- No Flutter code changed. Production deployment, live delivery, sampling/telemetry configuration and post-release recurrence checks remain unexecuted. No synthetic production errors were injected, no production configuration/data was changed and no Sentry issue status was mutated. Development-only repository snapshot failures are outside this change.

## Operator sign-in diagnostics (2026-10-19 follow-up)

- Operator login failures identify identity lookup, operator lookup, provider discovery, provider metadata, authorization request and login-attempt storage separately. Static explanations preserve original causes. The pinned Indigo error formats supply numeric upstream status and recognized OAuth machine codes; unknown provider text stays excluded.
- Observability package tests pass, including serialized local and Sentry sink checks for request correlation, stage, original cause type, selected provider facts, a single issue owner and protected-value canaries. PostgreSQL-backed login regressions pass for discovery, metadata, actual SDK PAR rejection and login-attempt storage failure, preserving the public 503 envelope. The metadata regression also verifies the confidential client publishes its verification key URL.
- `just test` passed across all Go packages with the race detector and local PostgreSQL/MinIO dependencies. `git diff --check` passed. This is test-suite evidence, not a complete AppView release gate.
- No Flutter code changed. Production deployment, live log delivery and successful provider sign-in after release remain unverified. These tests use local fixtures; no production failures were injected for this change.


## CraftSky open-issue recurrence evidence (2026-10-19)

- Instagram account failures now preserve original store causes and SQLSTATE through the request diagnostic owner. Project-read relationship failures identify their operation/stage; identity-resolution 502s retain the original native cause. Regressions verify matching error-envelope/header/Sentry request IDs and a single issue owner in serialized local/Sentry sinks.
- Login-start diagnostics distinguish identity lookup/result, provider discovery, provider metadata, authorization request and persistence without changing authorization or responses. The real coordinator and pinned Indigo client are exercised against local fake provider responses and real isolated PostgreSQL; retained provider facts are fixed machine codes and numeric HTTP status, never provider descriptions or OAuth capabilities.
- AppView emits its generated UUID in `X-Request-ID`, exposed to allowed web origins. Flutter prefers the canonical envelope's ID and otherwise accepts one UUIDv4 header. The closed response-shape diagnostic survives serialized Sentry delivery; malformed/multiple headers, response bodies and cookies remain excluded. Missing server correlation remains missing for failures outside AppView.
- Shared Flutter API/error/observability suites: 168 passed / one existing skip; analysis: no issues found. Transcripts: `/tmp/craftsky-recurrence-flutter-broad.log`, `/tmp/craftsky-recurrence-analyze.log`.
- Attached iOS/Android response-header retrieval, production deployment, live recurrence delivery and operational recovery remain unverified. No production faults were injected, no Sentry issue statuses were changed, and production telemetry gates/access/retention remain unchanged. Both a backend deployment and updated client build are required for all new evidence to appear in live reports.


Follow-up validation completed: shared Flutter API/error/observability suites passed 168 tests with one existing skip; `flutter analyze --no-pub` reported no issues. The complete `GOFLAGS='-p=1' ./scripts/appview-check` gate passed with 6,223 integration cases and 6,223 race cases, zero test skips, plus module/codegen/vet/Staticcheck, source/binary vulnerability checks, Docker builds, migration-failure checks, media-memory checks and service smoke checks. The real login regression also captures Indigo's process logger through the production diagnostic filter, verifying provider descriptions remain excluded. The fixture resides in the existing approved auth diagnostic test file; the Sentry import policy is unchanged. Transcript: `/tmp/craftsky-recurrence-gate-final.log`; artifacts: `/tmp/craftsky-recurrence-gate-final/`. Pinned Tap handshake passed; deterministic event injection remains unavailable in Tap v0.1.10. `git diff --check` passed. No device run, deployment or live recurrence delivery is implied by these local checks.


## iOS console readability (2026-10-28)

- The default platform sink pretty-prints protected JSON and wraps physical lines at 800 UTF-8 bytes without a total stack limit. Tests capture the actual print boundary, retain all 242 stack entries and async gaps, preserve long Unicode values, and exclude credential, local-path and private-exception canaries.
- Structural framework summaries identify missing Overlay, off-screen floating SnackBar and inactive render-object failures without exporting widget dumps or arbitrary exception prose. Rendering fallback no longer duplicates the framework callback log.
- Observability suite: 107 passed, one existing skip (`/tmp/craftsky-log-tests.log`). Final focused adapter, product AOT and handler suite: 11 passed (`/tmp/craftsky-log-final-tests.log`). Product AOT output retains all 240 fixture frames on short lines. Static analysis of all changed Dart files: no issues found. Profile Hero suite: 13 passed; no animation code changed.
- Connected iOS app hot restart succeeded; runtime-error buffer was empty immediately afterward. The original click sequence has not been reproduced, and attached-device error-console retrieval after reproduction, Android/web checks, release-device retrieval and production deployment remain unverified. This change improves diagnostics; it does not claim to repair the underlying UI failure.

### 28 October 2026: current-release Sentry triage

See [the triage record](../../development/sentry-triage-2026-10-28.md) for the live release inventory, per-issue evidence and recurrence-diagnostic changes. Serialized diagnostic tests retain safe transport/callback/provider facts and exclude protected-value canaries. Device reproduction, native hang breadcrumb delivery, native symbolication, live recurrence after a release, and Firebase delivery were not verified. No deployment was performed.

### 29 October 2026: expired account-deletion retry

- Reproduced FLUTTER-APP-2W locally: a persisted expired intent blocked staging a fresh server-issued intent. The regression failed at the same registry guard and controller start boundary before the fix. Expired local intents can now be replaced after server intent creation; live-intent and active-session fencing remain enforced.
- Reauthentication failures identify session/client lookup, intent creation, local staging, state invalidation and browser launch. Best-effort rollback failures retain their original causes in supporting logs. Riverpod remains the only issue owner, with the initiating account and existing job UUID selected through private operational failure context; existing API status/code/request correlation is retained.
- Serialized local and Sentry tests retain cause/stack, stage, account/job and request references, emit exactly one issue for a staging failure with a failed rollback, and exclude credential, OAuth URL/state, response and rollback-payload canaries.
- Focused registry, settings, account-boundary and diagnostic suites: 130 passed, four existing skips (`/tmp/craftsky-deletion-focused.log`). Flutter analysis: no issues (`/tmp/craftsky-deletion-analysis.log`). The required `just test` attempt could not start because this worktree's local Postgres service is not running (`/tmp/craftsky-deletion-go.log`); no Go code changed.
- Full Flutter suite: 3,509 passed, 38 existing skips (`/tmp/craftsky-deletion-full.log`), after preparing the checkout-local Rive host runtime. `git diff --check` passed.
- Physical iOS/Android reproduction, updated client release, production log/Sentry delivery and post-release recurrence remain unverified. No production faults were injected and no deployment or Sentry issue-state change was performed.

### 3 November 2026: web-client Sentry reporting

- Extended the existing consent-gated browser SDK with startup/framework and
  consumed social/settings/composer error ownership. Original error chains stay
  in memory; final event and wire filters retain only selected error structure,
  code coordinates, deployment/debug-ID metadata and safe AppView correlation.
  Cancellation, offline, validation and ambiguous pending writes remain excluded
  from Sentry. UI, authorization and retry semantics are unchanged.
- Svelte check passed with zero errors/warnings; all 284 unit tests across 44
  files passed, including serialized SDK protected-value canaries, positive
  correlation/debug-ID checks and lifecycle/failure-containment regressions.
- All three new browser reporting scenarios passed on Node preview and again on
  Cloudflare with subscriptions disabled. A loopback DSN and intercepted envelopes
  verified opt-in/reload/revocation, consumed failures and export outages without
  hosted Sentry traffic. A normal build was restored after fixture testing.
- Production build and private source-map packaging passed. Browser source maps
  carry matching debug IDs, their paired JS matches the public artifact, and maps
  are absent from both public output directories. Upload is a separate explicit
  `npm run sentry:upload` step using the authenticated CLI.
- Sentry project inventory was inspected via the CLI. Creating
  `craftsky/web-client` was rejected with HTTP403 by the organization's project
  creation policy. Administrator project creation is pending; no DSN was borrowed
  from Flutter/AppView. Hosted ingestion, source-map upload, production
  symbolication and deployment remain unverified. No synthetic production
  failures, release changes, retention changes or Sentry issue mutations occurred.
- This change instruments the browser only. The web BFF/Worker has no new Sentry
  integration; AppView retains its existing server reporting. No Flutter/Go code
  changed, so native/device and backend release gates were not rerun.
- The parallel accessibility audit and fixes are documented in
  [the web accessibility audit](../../development/web-client-accessibility-audit.md).
  `git diff --check` passed. No commit, push or deployment.

### 4 November 2026: web Sentry project configuration

- Administrator creation of `craftsky/web-client` was confirmed through the CLI;
  the project platform is `javascript-svelte`. The active public client DSN was
  retrieved through the authenticated CLI and saved only in ignored
  `web-client/.env.local`, with `PUBLIC_SENTRY_ENVIRONMENT=development`.
- Type checking passed with zero errors/warnings, and the configured build
  passed. The compiled client contains the selected public DSN. Private source
  maps match the public bundles; public deployment files contain no source maps.
  Existing opt-in consent remains unchanged.
- Source-map upload was rejected by automatic approval review because exporting
  source-code-derived artifacts requires explicit authorization. No upload,
  deployment or synthetic hosted error occurred. The local artifact is ready for
  review/upload; hosted ingestion and symbolication remain unverified.

- After explicit user approval, `npm run sentry:upload` completed successfully.
  The CLI confirmed organization `craftsky`, project `web-client`, 56 uploaded
  files and release `craftsky-web@1.1.1+ba372645b606`. The upload used the retained
  matching artifacts with `--no-rewrite`; no rebuild or deployment followed.
  Source-map upload is now verified. Actual hosted error delivery and production
  symbolication still require a real opted-in error; no synthetic error was sent.

### 5 November 2026: Bunny video cleanup worker

- Cleanup retries select operation, stage, attempt, initiating account and the
  durable job UUID. Initial failures log warnings; five or more failures in one
  stage log errors and capture an issue while recovery continues.
- `internal/video/diagnostic_boundary_test.go` verifies serialized local and
  mock Sentry output, including protected exception/URL/key canaries and positive
  retry/exhaustion fields. Its exact test file is permitted by the SDK import
  boundary; production feature code imports only the observability facade.
- The local worker removed the already-deleted test post's Bunny copy and
  purged its CDN prefix. Playlist, thumbnail and MP4 requests returned 404.
  Production worker rollout, production failure delivery and release-device
  retrieval were not exercised. No production faults or issue-state changes
  were introduced. This backend change does not require a new client build.

### 5 November 2026: web search crash and autocomplete recovery

- Read-only inspection of `WEB-CLIENT-1` confirmed that the owner's opted-in
  production reproduction was delivered and symbolicated to `identityCrafts`
  through `CompactIdentity.svelte`. This is evidence for the existing release's
  delivery and symbolication, not deployment evidence for these fixes.
- The shared web renderer now accepts null/absent crafts; AppView search summaries
  serialize empty crafts as arrays. Autocomplete uses the existing `Work` error
  owner and privacy-filtered reporter, with retry UI and canceled/stale-request
  fencing. Reporting consent and SDK privacy gates are unchanged.
- The web unit suite passed all 320 tests. Svelte checking passed with zero
  errors/warnings. Production builds and private source-map preparation passed;
  the normal local build was restored after loopback diagnostic testing.
- All 24 selected browser tests passed: search null/empty/absent crafts, native
  grouping and counts, clearing/cancel/submission, stale requests, retry,
  diagnostic consent/classification/correlation, accessibility and cached-tab
  navigation. Intercepted local Sentry envelopes retained API status/request ID
  and excluded query/server-message canaries; expected 400 errors emitted no
  event, and a 500 error emitted one event. Mobile and desktop fixture screenshots
  were visually inspected. No synthetic hosted Sentry event was sent.
- Flutter/native-device tests were not rerun because no Flutter code changed.
  Hosted verification of the new behavior, source-map upload and deployment have
  not been performed. The existing Sentry issue remains unresolved pending a
  release; no Sentry configuration or issue-state mutation occurred.
- A further light/dark axe scan is scoped to the changed search form. The initial
  whole-page scan also found an existing app-shell Create post button contrast
  warning (3.54:1); that unrelated control is not changed in this search fix.
- `just test` ran the full Go suite with the race detector. All packages passed
  except scheduledposts, whose MinIO integration test found that this newly
  created worktree fixture had no private bucket. After the checked-in
  `minio-bootstrap` service initialized the buckets, the entire affected package
  passed via `./scripts/appview-test focused -race ./internal/scheduledposts`.
  The full command was not repeated; the only failing package was rerun after
  repairing its fixture. The test services started for this run were then stopped.

## Production Sentry cleanup (2026-11-08)

- Authenticated CLI review covered 101 production Flutter/AppView issues and 373 production events across 90 days. [The issue-by-issue triage report](../../development/sentry-triage-2026-11-08.md) distinguishes fixes, existing repairs, upstream failures and diagnostic-only resolutions. All 101 issues have independently verified next-release activity; fresh production queries returned zero unresolved issues. The six dev-only issues remain unchanged at the maintainer's request.
- Complete Flutter suite: **3971 passed / 44 skipped** (`/tmp/craftsky-sentry-flutter-all.log`). Final affected diagnostic/API/language/feedback suites: **130 passed / 1 skipped** (`/tmp/craftsky-sentry-flutter-final-focused.log`). Additional serialized runtime/thread/privacy assertions: **8 passed** (`/tmp/craftsky-sentry-runtime-privacy.log`). Flutter analysis: **No issues found** (`/tmp/craftsky-sentry-analyze-final.log`). Original singular-transform paint callback reproduces `ArgumentError`; the fixed widget test passes.
- New diagnostics retain native provider causes/stacks, fixed picker/draft reasons, runtime accessibility/lifecycle state, sanitized current/crashed thread frames, OAuth quarantine checks and initial-token status/code, plus real chat stages/attempts. SDK/local sink checks retain positive fields and exclude protected-message, thread, register, source and payload canaries. Stale owner-generation callback refusal is classified as expected validation without altering authorization behavior.
- No running DTD/Flutter debug session was available for hot reload. Physical iOS/Android behavior, post-release Sentry delivery/symbolication and production recurrence remain **unverified**. No deployment, synthetic production failure, consent/configuration change or production data mutation was performed; only requested Sentry issue statuses changed. Changes remain local and uncommitted.

- Final complete AppView gate: **passed**, including **7246 integration cases / 7261 final race cases**, zero test skips; module/codegen/vet/Staticcheck, source/binary vulnerability scans, production/media image builds, migration checks, media memory checks and service smoke tests. The initial run found an unsupported sensitive stage label in a new fixture; it was corrected to the production `finalization` stage and verified before the final gate. Transcript: `/tmp/craftsky-sentry-appview-check-final.log`; artifacts: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.yRiLxQLFta/`. Pinned Tap handshake passed; deterministic event injection remains unavailable.

## Client error-handling audit (13 November 2026)

- Reviewed Flutter and Svelte user messages, consumed failure ownership, native
  cause chains, request correlation and final SDK privacy filtering. Both clients
  retain reviewed input copy and distinguish offline, expired-session,
  permission, missing-content, rate-limit and server failures. Account recovery
  now explains the underlying failure; pre-`runApp` failures have localized retry
  UI instead of a blank screen.
- Consumed billing, media/composer, saved-post, business, relationship and
  notification failures report their original cause and caught stack. Provider
  acquisition and already-published `AsyncError` controls verify one issue owner.
  Retired account operations remain cancellation; exhausted retries report once
  without changing the retry budget or pending/ambiguous state.
- Actual SDK-serialized fixtures retain native causes, app frames, operation,
  status/method and valid AppView UUIDv4 correlation. Protected payload, payment,
  path, credential and private-prose canaries are absent. Browser diagnostics
  remain opt-in; no consent, replay, tracing or production collection gate changed.
- Web validation: **541 unit tests**, Svelte checking with zero errors/warnings,
  ESLint and Prettier passed. **100 browser checks** passed: 22 authenticated
  diagnostics/field/action cases, 12 public-page cases and 16 guest-report cases
  on each of Node preview and local Cloudflare Worker. Diagnostic envelopes were
  intercepted on loopback. The normal build was restored afterward; source maps
  remained private and no upload occurred. Logs: `/tmp/craftsky-error-web-unit.log`,
  `/tmp/craftsky-error-web-browser-node-final.log`,
  `/tmp/craftsky-error-web-browser-worker.log` and the corresponding public/guest
  browser logs.
- Complete Flutter suite: **4240 passed / 44 skipped**
  (`/tmp/craftsky-error-app-tests-clean.log`). Strict whole-app Dart/Riverpod
  analysis passed (`/tmp/craftsky-error-app-analysis-final2.log`). The initial run exposed
  four older positive-correlation fixtures using non-UUID IDs; those fixtures and
  the shared Instagram corpus now use valid UUIDv4 values. All 96 affected tests
  passed before the final complete rerun. Malformed-ID privacy controls remain.
- Backend race verification covered 53 tested packages. The full run found only
  stale release tests from the earlier switch to strict Dart analysis. Their
  fixtures now stub both Flutter and Dart, verify `dart analyze --fatal-infos`,
  and propagate analyzer failures without invoking the host SDK. The complete
  affected `internal/app` package then passed with `-race`; all other packages
  passed in the full run. Logs: `/tmp/craftsky-error-appview-tests-final.log` and
  `/tmp/craftsky-error-appview-app-final.log`. Worktree PostgreSQL/MinIO fixtures
  were stopped afterward, preserving their volumes.
- The iOS simulator debug build passed. On this thread's iPad simulator, a hot
  restart applied the final code; the offline recovery message and Retry action
  were visually and interactively verified. Flutter tooling reported no runtime
  errors. Screenshot: `/tmp/craftsky-error-account-offline.png`. The test build
  used local API access and an empty Sentry DSN.
- Production Sentry inspection was read-only: eight unresolved Flutter issues and
  one web issue were listed, with representative existing events inspected for
  diagnostic gaps. No issue status, production data/configuration, deployment,
  hosted synthetic event or source-map upload changed. Physical iOS/Android and
  post-release hosted ingestion/symbolication of these changes remain unverified.

## AppView error-handling audit (13 November 2026)

- Reviewed request handlers/middleware, command rejection callbacks, provider
  adapters, ingestion, private workers, issue ownership and final local/SDK
  output against the mandatory logging guide. Checked Sentry Go's error-chain
  [documentation](https://docs.sentry.io/platforms/go/usage/) and the installed
  `sentry-go`/`sentry-go/slog` 0.49.0 implementations; no SDK upgrade was needed.
- Added original causes and operation/stage context to missing API failure
  owners, including notification, Instagram, saved-content, scheduling, billing,
  moderation, deletion and safety paths. Durable command rejection callbacks
  report before the terminal response is persisted. Private supporting logs
  select the initiating account only on actual failures.
- Corrected two API classifications: invalid notification preferences still
  return 400 before storage, while storage outages return 500; malformed manual
  scheduled-publication input now returns 400 instead of 500. No routes,
  Lexicons, migrations or wire fields changed.
- Preserved RevenueCat/Instagram transport, read and decode causes and safe
  upstream status; link-preview upstream rejection now retains its status.
  Supporting Sentry Logs no longer overwrite AppView response status with a
  provider status. Removed raw Instagram webhook-body logging and obsolete
  response-body configuration, including current reference/Blueprint entries.
  This is a source/configuration change, not a live Blueprint sync.
- Workers retain consumed failures and original causes when failure persistence
  also fails. Job/attempt scopes isolate sibling ownership and breadcrumb
  timelines. Manual publication keeps the HTTP issue owner; periodic retries
  remain WARN, exhausted failures and unexpected exits remain actionable.
  Retry timing, queue persistence, fencing and acknowledgement rules are intact.
- Serialized local and SDK fixtures verify native cause types, SQLSTATE,
  provider status, request/job references and initiating identity, with protected
  payload/credential/session/recipient/URL canaries absent. Independent review
  identified manual-publication duplication, premature safety-email capture and
  missing chat scope wiring; all three were corrected.
- Complete focused observability, middleware, Tap and link-preview suites passed
  (`/tmp/craftsky-appview-error-focused.log`). Go vet, pinned Staticcheck
  2026.2.1, configuration lint, generated catalogue/reference checks and all 18
  Python configuration tests passed. Logs:
  `/tmp/craftsky-appview-error-vet-final.log`,
  `/tmp/craftsky-appview-error-staticcheck-final.log` and
  `/tmp/craftsky-appview-error-config.log`.
- Backend race/integration coverage passed across **53 tested packages** against
  worktree PostgreSQL and MinIO. The full `just test` run passed 52 packages and
  found one older API fixture that permitted the initiating account on its issue
  but excluded it from supporting failure logs. The fixture now validates that
  account reference only on cause-bearing failure records; payload/token/device
  canaries and request/metric/transaction exclusions remain checked. Its focused
  race rerun passed, followed by the complete API package (194.144 seconds) and
  final Instagram Meta package (2.298 seconds), both with `-race`. Logs:
  `/tmp/craftsky-appview-error-race-full.log`,
  `/tmp/craftsky-appview-error-api-privacy-rerun.log` and
  `/tmp/craftsky-appview-error-api-final.log`. No race failures were reported.
  Worktree PostgreSQL/MinIO fixtures were stopped afterward, preserving volumes.
- Production Sentry inspection was read-only: 20 unresolved issues under the
  actual `prod` environment; `APP-VIEW-2F` retained a typed PDS 502 cause and
  provider code on release `craftsky-appview@1.3.2`. That upstream failure does
  not establish an AppView business-logic defect. No issue status, production
  data/configuration, deployment or hosted synthetic event changed. The full
  Docker release-image/Tap acceptance gate (`just appview-check`) was not run;
  post-release hosted collection of the new diagnostics remains unverified.
  Flutter and web code were unchanged by this backend pass.

### Profile save rejection diagnostics

AppView emits an expected warning for profile saves rejected during decoding
or validation. Search for `operation=profile.update` with the request UUID or
initiating account DID. The warning records the HTTP status, API error code,
stage and up to eight field paths. Link URL rejections distinguish duplicate
URLs from invalid URLs. Submitted values and arbitrary input keys are excluded.

The handler fixture passed for local logs and serialized Sentry Logs with
private-value canaries. The full AppView race and integration suite, 16 Flutter
profile tests, and Dart analysis passed. Hosted collection remains unverified
until release.

## Android composer playback (24 November 2026)

The original H.264 clip produced poster and cover images and a black playback
surface on the Android 17 emulator. MediaKit's GPU output logged
`EGL_BAD_ATTRIBUTE` after attaching a surface. The direct MediaCodec output
experiment also showed black frames.

Local Android composer previews now use the Flutter `video_player` Android
backend with the existing Flutter playback controls and lifecycle coordination.
The original clip rendered moving frames inline and in landscape fullscreen,
then returned to Video details. A 90-degree rotation fixture displayed moving
frames in portrait. Android debug and iOS simulator builds compiled, and
37 focused Flutter video tests passed. Physical-device playback remains
unverified for this change.

## Android native video probe (24 November 2026)

Android selection now reads local video metadata and a bounded JPEG from
`MediaMetadataRetriever` through the same channel contract as iOS. The
background probe reuses the existing cover-frame decoder and returns
display-oriented dimensions plus duration when available. The existing
selection reporter receives a stable platform error code on probe failure.
This replaces the MediaKit screenshot step recorded below.

On the Android 17 emulator, the original 17-second H.264 clip failed on the
previous build. The rebuilt debug APK attached the same clip, opened Video
details with a poster and four cover frames, and returned to the composer
with the attachment selected. A synthetic 640×360 H.264 clip with 90-degree
rotation metadata opened with the expected portrait poster.

Twenty focused Flutter tests and Dart analysis passed. Hosted Sentry
delivery and a physical-device release build remain unverified.

## Android video poster diagnostics (24 November 2026)

Sentry issue FLUTTER-APP-5Q recorded three Android selection failures after
MediaKit returned no poster. The existing selection reporter now includes the
capture result, allowlisted codec when there is a single source video track,
dimensions, available duration and rotation, and whether MediaKit reported a
hardware pixel format. MediaKit's screenshot API does not expose the native
error code.

The focused Flutter suites passed with 179 tests and one skip. The serialized
Sentry event retains the exception and stack alongside the new metadata. The
event and local diagnostic exclude filename and picker URI canaries. Dart
analysis passed. The full Flutter suite passed with 4,389 tests and 50 skips.
The AppView race and integration suite passed against the local PostgreSQL
and MinIO services.

The maintainer reproduced the failure on a local Android emulator. The local
record includes a null capture result for a 1920×1080 H.264 clip (17,493 ms,
rotation 0) with no reported hardware pixel format. The pinned MediaKit Android
controller completes its first-frame wait after sizing the surface, which does
not establish that a frame is ready for capture. This makes frame readiness a
plausible cause. The console trace does not contain the native failure code.

The emulator logcat shows two attempts with the same order. MediaKit published
a nonzero surface reference, then published `wid: 0` from surface cleanup, then
the selection controller logged the failure. The probe awaits player disposal
before the selection controller receives the error. The zero reference belongs
to cleanup and does not establish the video output state at capture time.

Physical Android reproduction, hosted Sentry collection and release
symbolication remain unverified for the enriched diagnostics.

## Android video selection diagnostics (18 November 2026)

- The shared Flutter selection controller reports unexpected failures for both
  composers. A local serialized-envelope test confirms one issue with the
  Android platform, picker code and caught stack, and checks that the local
  log and Sentry event exclude the selected file's name and URI.
- The original Android preview-build failure was not reproduced. Delivery
  from a rebuilt Android preview, hosted Sentry collection and release
  symbolication remain unverified.

## Sentry issue triage (20 November 2026)

The Sentry CLI returned 58 unresolved groups over the 365-day lookback, with
no further pages. The inventory covered all environments across 31 AppView
groups, 26 Flutter groups and one web group. The resolved and archived
inventory contained another 142 groups over the same lookback. The
investigation examined 493 retained events across the open groups.

| Sentry groups | Evidence and disposition |
| --- | --- |
| APP-VIEW-P, APP-VIEW-1Z | The pinned Indigo resolver could lose a failed HTTPS well-known lookup and proceed with an empty DID when DNS was skipped for `.bsky.social`. A local reproduction confirmed this path. The resolver now carries the error through and still verifies the declared handle against the DID document. Future reports distinguish handle well-known, PLC and DID well-known HTTP failures. The three 1Z follow failures were target DID lookups with no preserved provider status. |
| APP-VIEW-2C, APP-VIEW-29 | The historical chat errors lacked their provider details. Chat diagnostics now distinguish a failed OAuth refresh from a failed chat RPC and identify DID lookup, protected-resource discovery and authorization-server metadata, with reviewed status and codes. |
| APP-VIEW-21, APP-VIEW-22, APP-VIEW-25, APP-VIEW-26, APP-VIEW-27, APP-VIEW-28, APP-VIEW-2A, APP-VIEW-2B, APP-VIEW-2D, APP-VIEW-2E, APP-VIEW-2F, APP-VIEW-2W, APP-VIEW-2X, APP-VIEW-2Y, APP-VIEW-1A, APP-VIEW-1B, APP-VIEW-1C | Occurrence histories showed provider 502/503, network, TLS and deadline failures. The three oldest groups came from dev. Existing stages and retry metadata identify the affected chat operations. |
| APP-VIEW-8, APP-VIEW-2V | The generic 5xx group crossed multiple routes in dev and production. Most production occurrences were prior link-preview failures, before the current handler retained the specific fetch cause. The HTTP fallback now records the v1 error code when it owns the issue. 2V already identifies DNS NXDOMAIN in a link-preview fetch. |
| APP-VIEW-N, APP-VIEW-Q, APP-VIEW-13, APP-VIEW-16, APP-VIEW-20, APP-VIEW-24, APP-VIEW-H | PDS, transport and Firebase dependency failures. Existing provider status and error types establish the boundary. New AppView diagnostics add the classified request route and distinguish dial, read and write failures and known TLS, response-header and response-body timeout paths. |
| APP-VIEW-1Y, FLUTTER-APP-4G, FLUTTER-APP-4H | Post reporting returned HTTP 500. Both Flutter groups had a request UUID match with an AppView report-store check constraint failure. Migration 000120 corrected the constraint in AppView 1.3.0. |
| FLUTTER-APP-4Y | A PDS write returned 401 and cleared the active session while the composer could still build. A regression reproduced the widget failure. Both native composer sheets now tolerate the expected session cancellation. |
| FLUTTER-APP-51, FLUTTER-APP-57, FLUTTER-APP-58, FLUTTER-APP-59 | Nine iOS occurrences while paused showed `HttpException` without an HTTP response. The original connection failure phase remains unknown. The protected console and native issue now select the exact reviewed Dart HTTP close phase or retain a generic transport explanation. |
| FLUTTER-APP-52, FLUTTER-APP-53, FLUTTER-APP-54, FLUTTER-APP-55, FLUTTER-APP-4Z, FLUTTER-APP-50, FLUTTER-APP-4M, FLUTTER-APP-4N, FLUTTER-APP-4P | Login or chat failed with an AppView 502/503, a stable response code where available and an AppView request UUID. The provider and route findings above cover the server-side investigations. |
| FLUTTER-APP-4J, FLUTTER-APP-4S, FLUTTER-APP-4T, FLUTTER-APP-4V, FLUTTER-APP-4W, FLUTTER-APP-4Q | Five 502 responses lacked JSON and AppView correlation. The like action in FLUTTER-APP-4Q timed out waiting for a response and had no AppView request UUID. The reports already distinguish these cases. The responding edge and cause of the delayed write response were not established. |
| FLUTTER-APP-4R, FLUTTER-APP-4X, FLUTTER-APP-4K | The seven video assertions predate the native player disposal guard shipped in Flutter app 1.12.0+18. The guard and regression test are present in the current checkout. |
| FLUTTER-APP-56 | The single Android ANR retains both native threads and the lifecycle timeline. The main thread waits during Flutter surface destruction while Impeller's Vulkan raster thread is blocked in `vkDeviceWaitIdle`. The pinned engine still contains this path. |
| WEB-CLIENT-2 | The source-mapped event ended at the prior settings sign-out handler's generic non-2xx exception. The current browser already uses the typed API transport. The Worker now logs sign-out with its own operation, status, stable code and validated request UUID when available. The historic response status is unknown. |

Local regressions reproduce the identity lookup and composer failures, and
check retained fields and privacy canaries in serialized diagnostic sinks.
Flutter analysis was clean and the full suite passed 4,366 tests with 50
skipped. The web type check and lint passed, and all 615 web tests passed.
AppView race and integration coverage passed across the full run and two
package reruns. The full run picked up a chat test that moved after package
discovery, and the scheduled-posts check found the new MinIO bucket had
not been initialized. The final chat race suite and the scheduled-posts
race suite after repository-provided MinIO initialization both passed.
Device recurrence, native release symbolication and hosted delivery of the
new fields remain unverified. The worktree contains the changes and live
Sentry was accessed read-only.

## Sentry issue triage (27 November 2026)

The Sentry CLI returned 92 unresolved groups, with no further issue pages.
Event pagination yielded 707 occurrences across 40 AppView groups, 51 Flutter
groups and one web group. The inventory included three dev-only AppView groups
and three groups containing both production and development events.

### AppView

| APP-VIEW suffixes | Disposition |
| --- | --- |
| 1Y | Post-report constraint failures ended before AppView 1.3.0, which shipped migration 000120. |
| P, 1Z | The current identity resolver propagates a failed HTTPS well-known lookup and records its source boundary. The original upstream response in the follow failures is unknown. |
| 2C, 29 | AppView 1.5.0 and later identify chat OAuth refresh and metadata discovery stages and retain reviewed provider status and codes. |
| 8 | The generic 5xx group contains several routes and environments. The current link-preview handler retains the fetch cause. Public profile reads now preserve the resolver cause and classify its stage, covering the latest production occurrence. |
| 30 | Five PostgreSQL deadlines affected public browsing. All four occurrences with route classification were on guest project search. Diagnostics now identify candidate lookup, author hydration and post enrichment steps. The reason PostgreSQL exceeded the deadline remains unknown. |
| 25, 26, 32, 37, 38 | Protected causes now include reviewed socket errors and the federated HTTP purpose and failure kind. |
| 21, 28, 2F, 2W, N, Q | The current events identify upstream PDS failures. All 61 occurrences in 2F were HTTP 502 during chat status fetch, including four on AppView 1.6.1. |
| 13, 16, 1A, 1B, 1C, 20, 22, 24, 27, 2A, 2B, 2D, 2E, 2X, 2Y, 2Z, 31, 33, 34, 35, 36 | Network, TLS or deadline events. Existing operation, stage and transport diagnostics cover these paths. The three oldest dev-only groups are 1A, 1B and 1C. |
| 2V | Link-preview diagnostics already identify the DNS name-not-found result. |

### Flutter

| FLUTTER-APP suffixes | Disposition |
| --- | --- |
| 5G | Project-feed decoding rejected a DID returned as the author's handle after AppView identity lookup failed. The shared AppView search response now uses the standard `handle.invalid` sentinel and keeps the author DID. |
| K | Registration previously replaced the underlying failure with a generic outcome. The existing provider owner now receives the original cause, caught stack and a bounded request or browser-launch stage. The UI still uses the existing localized recovery copy. |
| 5E | The native video player now records whether the source was local or published, the known playback backend and fullscreen state when available. Native error text and source URLs stay out of the reports. |
| 4G, 4H | The reported HTTP 500s correlate with APP-VIEW-1Y. AppView migration 000120 fixed the underlying constraint. |
| 4K, 4R, 4X | The playback assertions all came from builds before the native player lifecycle guard in 1.12.0+18. The November 26 occurrence in 4R was from 1.10.0+15. |
| 4Y | The shared composer language provider now handles the expected account cancellation during composer teardown. |
| 5Q | Android local video preview moved to native playback in 1.14.1+22. The previously failing H.264 clip passed the local emulator check. |
| 56 | The Android ANR already includes lifecycle breadcrumbs and native stacks showing a Vulkan wait during surface teardown. The pinned Flutter engine owns that path. |
| 4J, 4S, 4T, 4V, 4W, 5S, 4Q | The recorded 502 payloads were non-JSON and the like timeout had no response. The existing response-kind and correlation diagnostics cover these cases. |
| 4M, 4N, 4P, 4Z, 50, 52, 53, 54, 55, 5A, 5F, 5H, 5J, 5K, 5N, 5P, 60, 61 | AppView returned a 502 or 503 with a request UUID. Correlation linked 60 to a PDS 502 on chat read, 5F and 61 to mention identity lookup failures, and eight events in 5H, 5J, 5K and 5N to timeline author-handle resolution failures. The server already records their operation and cause. |
| 51, 57, 58, 59, 5B, 5C, 5D, 5M, 5R, 5T, 5V, 5W, 5X, 5Y, 5Z | The transport failed before an AppView response was available. The current client records reviewed connection-close phases. Additional reviewed socket messages now distinguish refusal, route failure, read/write failure and timeout where the native cause exposes them. |

### Web

WEB-CLIENT-2 occurred once on 1.11.4. The current settings sign-out flow already
reports the operation, HTTP status, stable code and validated AppView request
UUID when available. The exact failure response from November 9 is unknown.
Sentry has the web 1.15.1 debug-ID artifact bundle. Its newest web Release record
is still 1.11.4. The pipeline uploads source maps independently of Release record
creation, and Sentry can create the record on ingest.

### Sentry status

All 92 groups were marked resolved in the next release. A separate read of
each issue verified the release-scoped activity, and a fresh unresolved search
returned zero groups across all three projects. Sentry recorded AppView 1.6.1,
Flutter 1.15.1+24 and web 1.11.4 as the current release anchors. A matching
event from the already deployed web 1.15.1 can therefore qualify as a
regression once Sentry sees that release.

### Local validation

AppView's full `just test` race and integration suite passed with the
worktree's PostgreSQL and MinIO. Focused tests cover a failed author lookup,
the typed public-profile resolver cause and a real PostgreSQL candidate-query
deadline. The serialized diagnostics retain the selected stage and correlation
with private-value canaries excluded.

Flutter static analysis passed. The full Flutter suite passed 4,432 tests
with 51 skips. Focused tests verify the original registration cause and stack,
playback context and selected socket failures in serialized Sentry and local
diagnostics with private-value canaries.

Web type checking and all 657 unit tests passed. The settings sign-out
regression passed in both Vite preview and the local Cloudflare Worker.
The test verifies the serialized failure context, recovery message and
exclusion of account identifiers and upstream text.

No Flutter debug process was available for hot reload. The new changes are
local, and hosted diagnostic delivery remains unverified.
