# Coding Plan: Useful Logging and Error Reporting

## Sentry-led implementation amendment approved 2026-10-07

Use the existing worktree and implement SDK-T01–SDK-T06 in order. Flutter: register supported SDK cause extractors for app/API wrappers; reviewed static StateError subtype retains its message without permitting arbitrary StateError prose; native exceptions carry chain/stack instead of JSON summaries. Configure official logging integration with issue threshold off and a small context enrichment bridge. Local root console emitter remains independent. Install local-only error callbacks before Sentry integration setup; leave SDK handlers installed afterwards and preserve UI fallback. Go: use SDK SetException conversion with original cause hint and existing bounded cycle detection; preserve extracted stacks, omit synthetic capture-time stacks for stackless errors; reviewed static wrapper explanations and safe runtime adapters remain source controlled. Use official slog handler for remote export after source selection, keep local sanitizing handler and targeted worker suppression. Both final filters preserve selected breadcrumbs and standard metadata, continue rejecting attachments/private enrichment. Simplify ErrorReporter to explicit exception capture; feature code continues to avoid Sentry imports. Add dependencies only for official integrations. Do not enable tracing or alter production/business/API behavior.


## Refined coding plan approved 2026-10-07

Implement in order SIM-T01 → SIM-T02 → SIM-T03 → SIM-T04 → SIM-T05 → SIM-T06. Each behavior receives a meaningful red before its implementation, then affected regression checks.

1. Remove client message/route catalogues and resource-path classification. Slim ApiFailureDetails/AppError diagnostics to stable codes/status/method/requestId.
2. Remove automatic log-to-issue promotion and ownership properties; keep explicit provider/unhandled capture owners and audit consumed storage failures.
3. Replace the aggregate/chunk console pipeline with a fixed brief-record selector; remove generic Flutter retry limiter, preserve original business retries and Go targeted limiter.
4. Use SDK captureException with original throwable/stack, simplify beforeSend/Log/Breadcrumb callbacks and safe context selection; remove selected-event/log Expandos and event reconstruction.
5. Remove Go message catalogue and log nonce registry; preserve typed selection, private worker provenance, vetted storage causes and independent local filtering.
6. Update canonical guide/traceability; verify final serialized sinks, focused/full suites, analyzer, diff. No commit or production action enabled.


## 1. Inputs

- Requirements: `01-requirements.md`, Reviewed; user-confirmed public/private boundary and grilling decisions.
- Tests: `02-acceptance-tests.md`, reviewed together with the requirements; 49 proposed cases covering 25 requirements and 24 acceptance criteria.
- Document review: `03-document-review.md`, Approved with notes; DR-001–DR-005 addressed below.
- Architecture: root `AGENTS.md`, `atproto-craft-social-app-reference.md` public-PDS/private-AppView split and server-side OAuth boundary, and the AppView API architecture spec. This plan changes diagnostics, not record ownership or API behavior.
- Status: Draft for approval to implement. Risk: High. This stage performed read-only inspection and wrote this plan only. No source/tests/dependencies/configuration changed, application tests run, commit created or production action performed.

## 2. Implementation Strategy

Keep Go slog, the existing Observer, Flutter package:logging, Riverpod and Sentry adapters. Add one shared diagnostic contract per runtime, supported source adapters and final emission protection. Restore useful causes and public references through that contract, then migrate the audited feature boundaries. Separate log export, issue capture and trace sampling; use existing classification and stable metric vocabulary without restricting event context to metric labels.

Go local and remote logs will share a sanitizing slog handler. Observer.Log will use it instead of separately exporting a second copy. Flutter will use a single root-log pipeline with a structured diagnostic message carried by `LogRecord.object`, plus explicit issue capture owners. Existing raw/third-party records get safe source-specific treatment at the boundary rather than permission to stringify arbitrary values.

Implement all Must requirements and include the Should public diagnostic summaries. Do not defer the private worker, provider, storage, Tap, contributor guidance or validation work after repairing central logging. The F-01–F-20 call-site inventory is a completion artifact.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Acceptance Criteria | Test IDs |
|---|---|---|---|---|---|
| Shared diagnostics | Restrictive context allowlists, generic error classes | Typed source/context policy, retained cause descriptions, bounded safe fields and final SDK scrubbers | FR-001, FR-002, FR-010, FR-011, NFR-002, RULE-001, RULE-002, RULE-003 | AC-001, AC-002, AC-006, AC-012, AC-013, AC-021 | UT-001–UT-005, UT-012, UT-013, AT-001, AT-002, AT-007, IT-011 |
| AppView logger and wiring | JSON slog plus separate Observer log export | One sanitizing handler; bind export sink after Observer construction; direct slog covered; no duplicate export | FR-013, FR-015 | AC-015, AC-022 | IT-012, IT-013, REG-004 |
| HTTP diagnostics | Arrival lacks path; completion DEBUG; health paths bypass observed branch | Pre-arrival route/privacy classification, INFO lifecycle records, independent request correlation, probe severity exceptions | BR-001, FR-003, FR-004, RULE-004 | AC-003, AC-004, AC-020 | AT-003, AT-004, UT-006, UT-007, IT-001–IT-004 |
| Cause capture and classification | CaptureError/Panic generic events, 5xx fallback, repeated reporting owners | Preserve safe original type/cause/stack, recovery-time stacks, per-occurrence ownership; expected failures remain contextual | BR-001, FR-001, FR-007, FR-008 | AC-001, AC-002, AC-008, AC-009, AC-021 | UT-001–UT-003, UT-010, UT-011, IT-003, IT-009 |
| API/auth/dependencies/Tap | Helpers drop cause or callers log raw errors | Source adapters and operation context; public record references; vetted dependency fields; terminal outcomes | FR-002, FR-009, FR-011 | AC-006, AC-010 | IT-006, AT-001, AT-002 |
| Private workers/integrations | Content-free classes and occasional raw upstream text | Vetted cause, account/stage/opaque workflow reference on operational failure; keep private target/activity protected | FR-009, RULE-002 | AC-011 | AT-006, IT-007 |
| Flutter logging/reporting | Root debug print; SEVERE/promoted WARNING forwarding; placeholder provider failure | Structured diagnostic input, independent Logs, breadcrumbs, safe platform fallback and explicit capture owner | FR-005, FR-007, FR-008, FR-013, FR-016 | AC-005, AC-008, AC-009, AC-015, AC-023 | AT-005, UT-008, UT-010, UT-011, IT-005, IT-009, IT-012, IT-015, MAN-001 |
| Flutter API and feature context | Narrow envelope and endpoint mapper; initialization fixed message | Safe envelope details through AppError; registered route classification; operation-account attribution and feature causes | FR-003, FR-006, FR-009, FR-012, NFR-003 | AC-003, AC-007, AC-010, AC-014, AC-018 | IT-004, IT-008, IT-010, UT-009, UT-014, REG-001, REG-002 |
| Volume and aggregation | Existing bounded metrics; no shared enrichment budget | Diagnostic budgets and scoped retry suppression; preserve stable grouping and terminal cause | NFR-001, NFR-002 | AC-013, AC-016, AC-017 | UT-013, UT-015, UT-016, IT-016 |
| Guidance/evidence | Root contributor rules, no canonical logging guide | Mandatory AGENTS.md section, practical guide and validation evidence with pending rollout checklist | FR-014, FR-017 | AC-019, AC-024 | IT-014, MAN-002–MAN-004 |

## 4. Files And Modules

Paths marked Create are proposals. Existing modules will be changed only where needed for the diagnostic contract, preserving their business interfaces except for diagnostic parameters/adapters.

| Path / Module | Create / Change | Purpose | Requirement IDs | Acceptance Criteria | Test IDs |
|---|---|---|---|---|---|
| `appview/internal/observability/diagnostic.go`, `diagnostic_policy.go`, `diagnostic_bounds.go` | Create | Immutable selected diagnostic description; approved source adapters, field policy and budget/suppression helpers | FR-001, FR-002, FR-010, FR-011, NFR-002, RULE-001–RULE-003 | AC-001, AC-006, AC-012, AC-013, AC-017, AC-021 | UT-001–UT-005, UT-012, UT-013, UT-016, IT-011, IT-016 |
| `appview/internal/observability/sentry.go`, `observer.go`, `logs.go`, `validation.go`, `error_classifier.go` | Change | Concrete cause/stack event construction, final SDK hooks, independent sink dispatch, correlation and separate context/metric policies | FR-001, FR-003, FR-007, FR-008, FR-013, FR-015, NFR-001 | AC-001–AC-003, AC-008, AC-009, AC-015, AC-016, AC-022 | AT-001–AT-003, UT-001–UT-003, UT-006, UT-010, UT-011, UT-015, IT-003, IT-012, IT-013 |
| `appview/internal/observability/log_handler.go`, `request_path.go` | Create | Sanitizing slog handler and request path contract, without importing routes | FR-004, FR-010, FR-015, RULE-004 | AC-004, AC-012, AC-020, AC-022 | UT-007, IT-001, IT-011, IT-013 |
| `appview/internal/app/deps_foundation.go`, `deps_observability.go`, `deps.go`; `appview/cmd/appview/server.go` | Change | Install protected process logger before DB connection; attach remote sink after validated Observer; wire classifier and health logging | FR-003, FR-004, FR-013, FR-015 | AC-003, AC-004, AC-015, AC-022 | IT-001–IT-004, IT-012, IT-013 |
| `appview/internal/routes/policy.go`, `catalogue.go`; proposed `diagnostic_policy.go` in routes | Change / Create | Diagnostic metadata aligned with actual route catalogue; non-mutating pre-routing resolver; route manifest for Dart tests | FR-004, FR-006, RULE-004 | AC-004, AC-007, AC-020 | UT-007, IT-001, IT-008 |
| `appview/internal/middleware/logging.go`, `recovery.go`, `metrics.go` | Change | Lifecycle fields/probe levels/recovery frames/capture marker; keep response semantics and metric behavior | FR-001, FR-003, FR-004, FR-007, NFR-003 | AC-002–AC-004, AC-008, AC-018 | IT-001–IT-004, REG-003 |
| `appview/internal/api/telemetry_log.go`, `auth/telemetry_log.go`, API/auth call sites; observability `db.go`, `pds.go`; `tap/consumer.go`, `internal/index/`, ingestion callers | Change | Source-aware error adapters and cause/context propagation without capturing at every wrapper layer | FR-001, FR-002, FR-009, FR-011 | AC-001, AC-002, AC-006, AC-010, AC-021 | IT-006, AT-001, AT-002 |
| `appview/internal/scheduledposts/`, `accountdeletion/`, `push/`, `instagram/` and relevant private API handlers | Change | Failure-only account/stage/workflow attribution; vetted provider/DB cause; safe stored error summaries where those reach diagnostics | FR-009, NFR-002, RULE-002 | AC-011, AC-017 | AT-006, IT-007, IT-016, REG-003 |
| `app/lib/shared/observability/diagnostic.dart`, `diagnostic_policy.dart`, `diagnostic_emitter.dart`, `platform_log.dart`, `diagnostic_emitter_provider.dart` | Create | Selected input/record and safe adapters; root pipeline, injectable device sink and simple DI provider | FR-001, FR-002, FR-005, FR-010, FR-013, FR-016, NFR-002, RULE-001–RULE-003 | AC-001, AC-005, AC-012, AC-013, AC-015, AC-021, AC-023 | UT-001–UT-003, UT-008, UT-012, UT-013, IT-005, IT-011, IT-012, IT-015 |
| `app/lib/shared/observability/error_reporter.dart`, `log_forwarder.dart`, `sentry_error_reporter.dart`, `sentry_sanitizer.dart`, `observability_bootstrap.dart`; `app/lib/main.dart` | Change | Separate emitLog from issue capture, guarded interfaces/final callbacks, single root subscription and safe startup/framework/platform paths | FR-005, FR-007, FR-008, FR-013, FR-016, NFR-001 | AC-005, AC-008, AC-009, AC-015, AC-016, AC-023 | IT-005, IT-009, IT-012, IT-015, UT-015, REG-004, REG-006 |
| `app/lib/shared/api/api_exception.dart`, `providers/error_mapping_interceptor.dart`; proposed `api_diagnostic_routes.dart`; `app/lib/shared/errors/app_error_mapper.dart` | Change / Create | Method/route/safe path/status/code/message/field reasons/request ID and network cause selection through both mapping layers | FR-003, FR-006, NFR-003, RULE-004 | AC-003, AC-007, AC-018, AC-020 | UT-006, UT-007, UT-009, IT-004, IT-008, REG-001, REG-002 |
| `app/lib/bootstrap.dart`; `auth/providers/active_account_initialization_provider.dart`, `auth/widgets/active_account_initialization_gate.dart`; auth/storage/media/video/feature mutation call sites | Change | Provider errors retain cause; initialization helper receives failed stage/lease; supporting gate log has no second capture owner; operation-account attribution | FR-001, FR-007, FR-008, FR-009 | AC-001, AC-008–AC-010 | IT-009, IT-010, UT-011, REG-002 |
| `app/lib/observability/video_diagnostics.dart`; public identity/post/event summary methods and mixed private/auth model diagnostic adapters | Change | Useful phase/cause/provider status and public references; selected model summaries while private toString protections remain | FR-009, FR-012, RULE-001, RULE-002 | AC-010, AC-012, AC-014 | UT-014, IT-010, REG-005 |
| Existing Go/Dart suites plus proposed diagnostic, SDK-envelope, correlation, path, platform and doc-example tests listed in `02-acceptance-tests.md` | Change / Create during TDD | Implement all 49 cases using existing fixtures; Go app tests and cmd server wiring verify real chain | BR-001, FR-001–FR-017, NFR-001–NFR-003, RULE-001–RULE-004 | AC-001–AC-024 | AT-001–AT-007, UT-001–UT-016, IT-001–IT-016, REG-001–REG-006, MAN-001–MAN-004 |
| Root `AGENTS.md`; `docs/engineering/logging-and-error-reporting.md` | Change / Create | Mandatory policy plus one canonical Go/Flutter practical guide | FR-014 | AC-019 | IT-014, MAN-002 |
| `docs/changes/2026-10-02-logging-error-reporting/06-validation-evidence.md` | Create during implementation | Audit inventory, positive/negative sink evidence, investigations, device results and separately pending production checklist | FR-017, FR-014, FR-016 | AC-019, AC-023, AC-024 | MAN-001–MAN-004 |

## 5. Services, Interfaces, And Data Flow

### Diagnostic selection and provenance (DR-001)

Represent workflow privacy separately from error-message provenance. PublicRecordContext permits already-published target references; PrivateOperationalFailureContext permits operation account DID, stage and a verified non-capability workflow ID, and excludes target/membership/recipient/content. Unknown context has no permissive default. PublicRecordContext does not authorize arbitrary raw exception text or whole records.

KnownSafeError is assigned by reviewed adapters for specific application-authored static explanations or validated public-boundary validation failures. Do not add a generic `safe: true` switch which dumps err.Error/toString. For approved wrapped errors, retain vetted wrapper explanation and describe each cause independently; never derive a safe wrapper by splitting an arbitrary rendered error string. Approved dependency adapters select network failure kind, credential-free origin, provider status/code, SQLSTATE and an approved constraint identifier where safe. Unknown/private error messages use a specific vetted explanation plus concrete type, stage, available stack and approved fields. Do not serialize SQL row detail, arbitrary provider HTML/body, headers or storage objects.

Use typed Go atproto wrappers already parsed at business boundaries; logging does not reparse trusted IDs. Public attempted identifiers from a failed parse remain bounded attempted values with the validation reason. Dart diagnostic fields distinguish actor/target and preserve immutable operation context. A selected public excerpt requires an explicit failure-diagnostic purpose and a source already known published; publishing request bodies, drafts and schedules never qualify. Enrichment uses data already available to the operation and does no extra fetch.

Partial interface sketches, names subject to normal implementation refinement:

```text
Go: DiagnosticInput {operation, stage, workflowContext, sourceAdapter, err, suppliedStack, occurrence}
Go: DescribeError(input) -> DiagnosticRecord       // original concrete types; selected causes/frames
Go: DiagnosticHandler implements slog.Handler    // local protection + independent Logs dispatch
Go: RequestDiagnosticResolver.Resolve(method, url) -> RequestDiagnosticContext
Go: Observer.CaptureDiagnostic(ctx, record)       // issue only; owner/expected policy applies

Dart: DiagnosticInput(operationContext, sourceAdapter, error, stack, occurrence, captureOwner)
Dart: DiagnosticPolicy.select(input) -> DiagnosticRecord
Dart: ErrorReporter.emitLog(DiagnosticRecord record)
Dart: DiagnosticEmitter.emit(LogRecord record)    // select -> local/remote/breadcrumb branches
Dart: DiagnosticEmitter.reportFailure(input)     // supporting log + explicit issue capture
```

Do not hand raw exception objects to Sentry for automatic stringification after selection. Construct error events with retained concrete exception types, selected cause messages and available stacks. Go recovery supplies a recovery-time stack; stackless ordinary Go errors do not acquire a pretend origin stack. Dart events retain the supplied stack. Reduce local user path components in frame filenames while preserving module/function/frame usefulness.

### Go logging and bootstrapping

Build a shared handler state containing policy, budgets, local handler and an initially no-op remote LogSink. Foundation installs it around the JSON stdout handler and sets the same logger as slog.Default before DB connection. After newObservabilityDependencies succeeds, bind its export-only sink to that shared state once, before workers start. WithAttrs/WithGroup clones share sink state; synchronization and closure are safe. No initialization-time logs are replayed or persistently buffered.

Handle sees raw record message, attributes, group/LogValuer values and request context, and builds a safe record before invoking either sink. Evaluate only supported scalars/approved diagnostic values; do not stringify arbitrary objects or expand them recursively. Unknown third-party messages get type/source/technical context and vetted text; approved logger/source adapters preserve safe messages. Every migration of a call site supplies actual error and source adapter rather than a pre-redacted category alone. Replace unsafe localOnlyAttrs with the same selected policy; local output has no private-data exemption.

Enabled must allow records needed by either configured sink. Handle checks local threshold and remote eligibility independently. WARNING/ERROR are export eligible; selected INFO is eligible for ordinary HTTP arrival/completion and explicit significant public/operational outcome records. DEBUG remote output defaults off. Observer.Log delegates to the handler; it must not also call LogSink.Emit. The handler's export-only sink never calls the process logger. Export errors use a sanitized local-only fallback with a recursion guard, not a global log path that re-enters export.

Observer construction must also support tests or shared callers that supply a plain logger, an injected LogSink or no local logger: adopt an existing DiagnosticHandler or wrap the injected local handler exactly once; use a discard local handler if absent. Both paths share the same export-only sink and policy, so Observer.Log retains independent export without requiring application foundation wiring. Validate standalone Observer and configured application logger cases.

Retain current startup configuration validation and cleanup semantics. Telemetry runtime failure must not change business results. A missing DSN keeps the protected local logger functional. Shared CLI initialization uses the same protection; CLI config/credential dumps are not added.

### Request path classification and correlation (DR-002)

Extend V1 route diagnostic metadata alongside V1RoutePolicies; use V1Catalogue's matching/canonical-path logic through a read-only diagnostic resolver supplied to middleware. Its API must not invoke a handler, perform auth, reject a request or alter request URL/routing decisions. Middleware imports an observability-owned resolver interface, avoiding routes -> middleware -> routes cycles. Add explicit safe metadata for existing non-v1 OAuth metadata/JWKS/callback and health paths; no new HTTP route.

Classify before arrival. Match both method and path; for method mismatch use the route's safe conservative path policy, never infer public data from access class. Record matched route literals and approved public DID/handle/rkey segments; private save/report/folder/media targets and hashtag/search terms become markers. Unclassified paths preserve only known safe structural literals and marker positions, within bounds. Reject ambiguous/non-canonical decoding for diagnostics by using conservative markers; do not transform the actual request. Omit query/fragment entirely. Initial operational-query allowlist is empty: retain no query values until a reviewed typed adapter is justified. Public references are allowed only in companion context with the same workflow policy, preventing reintroduction of a private target.

Keep local `run_id`, API `requestId` and Flutter `appViewRequestId` as sink-specific names for the same unchanged ID. Attach trace/span IDs if available; use request ID independently of tracing. Preserve existing response envelopes. Generate correlation before arrival and any host/rate/body rejection. Completion keeps the stored sanitized actual path and updates route pattern from the existing recorder; patterns remain stable aggregation keys. Do not derive metrics or transactions from actual paths.

Wrap both health branches in Logging, while preserving their current bypass of HTTPMetrics and their existing recovery/host/admission handling. Normal arrivals/completions INFO including 4xx/unmatched; separate 5xx cause ERROR. Health arrivals DEBUG; successful completions DEBUG; failed completions WARNING for 4xx or ERROR for 5xx/unexpected. A stalled request produces arrival immediately. Keep existing canceled/499 internal outcome and ResponseWriter semantics.

### Issue ownership and final SDK boundaries

Go capture occurs at the layer with the actionable cause and necessary approved context; helpers describe/enrich rather than every DB/PDS/API layer capturing independently. Preserve the existing request capture marker, setting occurrence ownership regardless of remote availability; HTTP fallback reports only when no owner captured the unexpected 5xx. Panic recovery owns its occurrence. Tap owns terminal/quarantined events, jobs own terminal actionable failures; recoverable expected retries have logs/context without unexpected issue capture. Preserve separate genuine occurrences even when error text/object is reused.

Flutter root-zone/framework/platform handlers own their occurrences; their supporting structured log carries an owner/occurrence marker so root forwarding exports logs without recapturing. ProviderLogger owns reportable provider failures; feature helpers enrich that cause and avoid duplicate generic capture. Explicit mutation handlers own mutations they handle, and unowned severe records use the root fallback owner after expected-failure classification. Carry occurrence identity through the reporting path; do not deduplicate by message or permanent raw error-object identity. Do not suppress the linked AppView event. Check SDK-installed FlutterError/OnError/zone integrations during real bootstrap tests and avoid registering a second owner for app-owned callbacks; retain appropriate native crash handling.

Installed dependencies were inspected locally: sentry-go v0.49.0 and sentry/sentry_flutter v9.23.0 expose error, transaction and log before-send hooks; Dart Transport sends SentryEnvelope. Use Go BeforeSend/BeforeSendTransaction/BeforeSendLog/BeforeBreadcrumb and Dart beforeSend/beforeSendTransaction/beforeSendLog/beforeBreadcrumb for final selection, sanitization and bounds. Sanitized record metadata reaching callbacks must not be a spoofable permission for unknown SDK fields. Keep default PII/raw-body/replay/screenshot/failed-request collection protections; suppress unsupported unsafe automatically collected fields. In-memory test transports assert final serialized logs, exceptions, breadcrumbs, contexts and transactions. No package upgrade is planned.

### Budgets and suppression (DR-004)

Adopt the following starting implementation budgets as named constants in both runtime policies. Measure UTF-8 serialized output, including redaction/truncation markers and structured encoding overhead. Test actual sink envelopes separately from application-owned diagnostic budgets. An SDK envelope may include SDK metadata and batching overhead; its application-owned record must still obey these limits and SDK limits. If a platform or SDK needs a lower bound, lower its sink budget, document why, and rerun positive usefulness and boundary tests rather than silently truncating correlation/core fields.

| Budget | Planned value | Treatment |
|---|---|---|
| Text field and sanitized path | 2 KiB each | Sanitize/select first; UTF-8-safe truncation and marker. Omit/mark an over-budget identifier rather than falsely present a truncated valid ID. |
| Selected public payload excerpt | 4 KiB total | Failure purpose + published provenance; selected fields only; first item dropped under record pressure. |
| Context fields / nested structures | 32 root fields, depth 4, 32 entries per collection; 16 KiB aggregate | Approved fields only; stop cycles and mark omitted counts. No full models, bodies or media. |
| Cause tree / stack | 8 causes, 64 frames total | Keep concrete types and most useful origin/recovery frames; repeated/cyclic links marked; sanitized filenames. |
| Diagnostic error record | 64 KiB | Type/operation/correlation and useful frames precede excerpts; never log whole raw event on budget failure. |
| Log/platform diagnostic record | 16 KiB | Same priorities; platform output chunks at <=2 KiB with bounded <=8 chunks and correlation/chunk metadata included in budget. No application-managed persistent buffer. |
| Relevant application breadcrumbs | 50 entries maximum | Select navigation/lifecycle/operation categories; SDK ring bounds, no extra persistent history. |

Do not rate-limit ordinary request arrival/completion, first/terminal unexpected errors or distinct issue occurrences. Limit repetitive recoverable worker/dependency retry logs only: first 3 emissions per 60-second window for (stable operation, stage, class, safe opaque workflow reference when available), followed by a count/outcome summary at window rollover or workflow end. Terminal cause bypasses suppression and includes pending count. Bound in-memory suppression keys to 256; flush a safe summary on eviction rather than grow memory. Use injected clock and reset on scope completion. Account/record IDs and payloads never become metric labels/grouping dimensions; ephemeral suppression scope is not an aggregation label or persistent history.

### Flutter API, local and remote data flow

ApiFailureDetails gains method, stable route pattern/category, safe target fields, vetted/sanitized diagnostic message for both 4xx/5xx, safe validation field reasons and selected network/parse cause metadata. Preserve request ID unchanged. Only typed AppView envelope fields selected; registration/provider exceptions remain subject to source policy. Retaining a safe 5xx message does not display it directly to the user. AppErrorMapper copies these fields to safeDiagnostics while preserving existing kind/localization/recovery actions. Dio RequestOptions/Response remain needed by networking but never enter the diagnostic record wholesale.

Maintain a shared synthetic route/response fixture under this workflow's test fixtures, generated/verified from actual Go registrations, consumed by Dart route tests. It is test data, not a new wire contract or runtime Flutter dependency on Go code. Compare configured dev/admin variants as well as ordinary routes. Dart route policy mirrors the approved method/pattern/segment classification; drift fails IT-008. No HTTP fetch to discover routes.

One root subscription calls DiagnosticEmitter synchronously for local output before guarded remote dispatch, preserving logs even if Sentry throws/hangs or bootstrap has not initialized. Structured diagnostic message objects carry selected operation/provenance/owner metadata through LogRecord.object; their toString returns only the safe diagnostic message. Unstructured raw records get conservative approved-source policy. Remove raw print and developer.log exception interpolation from owned emission paths, including FlutterError.presentError output where necessary; maintain the existing framework handling and release error widget UI while presenting sanitized debug exception details. Configure root selection according to severity; platform WARNING/ERROR runs in release regardless of Sentry, and safe debug output remains available.

Extend ErrorReporter with emitLog for severity-aware independent Sentry Logs; update Noop/Guarded/Sentry implementations and test doubles. Preserve captureMessage compatibility as safe log emission rather than making warnings into issues. LogForwarder/DiagnosticEmitter preserve meaningful messages, exception operation context and relevant breadcrumbs; selected INFO outcomes use explicit source metadata, never generic state dumps. High-cardinality identities/request IDs go in context/log attributes, not metric labels, transaction names, fingerprint or generic issue tags used as grouping keys. No new persistent storage, buffers or support screens.

## 6. State, Providers, Controllers, Or DI

Use a simple Provider<DiagnosticEmitter> named diagnosticEmitterProvider; this is a dependency adapter, not application state and does not need an AsyncNotifier. Retain errorReporterProvider and the existing bootstrap overrides. Construct the emitter with a no-op remote reporter and live safe platform sink before async initialization; after initialization attach the GuardedErrorReporter once. Sentry disabled/failing leaves the emitter's local behavior intact. Bootstrap injects the same emitter/reporter instance into ProviderLogger and ProviderScope; root error callbacks use it before/after bootstrap.

```text
main: safe platform sink + DiagnosticPolicy -> DiagnosticEmitter (remote initially Noop)
ObservabilityBootstrap -> GuardedErrorReporter -> emitter remote binding
ProviderScope: errorReporterProvider override + diagnosticEmitterProvider override
ProviderLogger -> mapper + diagnostic emitter -> local / Logs / owned issue
sessionRegistryProvider.activeLease -> accountLanguagePreferencesProvider(lease)
                                     -> onboardingStatusProvider(lease.session)
                                     -> activeAccountInitializationProvider
operation lease captured at start -> feature diagnostics (never current-account lookup at emit time)
```

ProviderDidFail retains the actual exception/cause description and supplied stack rather than RedactedProviderFailure. Keep current retry and expected-error classification. Selected provider summaries expose public identity/post/event references and count/state; retain private draft/save/auth representation protections. Do not log complete provider state or family arguments whose provenance is unknown.

Active-account initialization captures the exact lease and failure stage where dependencies fail; extend logActiveAccountInitializationFailure to receive the selected cause/stack/account/stage. If the registry fails before a lease is available, retain its cause/stage without guessing an account. Capture dependency-stage context at the provider boundary while preserving the original error and supplied stack. The gate's existing listener may emit a supporting owned log or rely on the enriched provider log; it must not create a second issue. It must not attach the registry's newer active account after an operation switches accounts. Existing initialization/auth/session state machines and generated providers remain unchanged unless signature wiring requires their normal regeneration; no provider architecture migration is planned.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

- No new screen, widget, route, navigation, developer-detail UI or persistent support-log feature. Preserve loading/empty/localized error/retry/sign-in/switch-account behavior. The existing initialization gate's diagnostic call is the only planned widget wiring adjustment; validate its state/action regressions.
- HTTP handlers keep their routes, status codes, response envelopes and authorization. Change telemetry helpers and supported diagnostic metadata only. Read the API architecture spec during any handler touch; camelCase wire errors and optional fields remain intact.
- Background jobs retain lease/fencing, scheduling/retry/quarantine/ack and deletion decisions. Add safe failure-stage/workflow context at operation boundaries, not account-linked success histories.
- Shared CLI/process logging uses protected initialization and source adapters; no new CLI command or config dump.
- Contributors start from a concise mandatory AGENTS.md logging/reporting policy and follow `docs/engineering/logging-and-error-reporting.md`. The guide includes supported entry points, classifications, severity/sink matrix, actual path/correlation handling, cause/stack preservation, expected failures, capture owners and duplicate avoidance, budgets, correct/incorrect Go/Dart examples, commands and review checklist. Use actual final interfaces; no new `.agent/rules` or mandatory skill.
- No persistence schema, query, lexicon, AT Protocol record format, media transport or external dependency changes. Private workflows remain AppView-private; ordinary writes remain PDS-mediated through AppView.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Acceptance Criteria | Test IDs |
|---|---|---|---|---|
| Known-safe error vs unknown/private error | Retain selected original type/cause for both; safe message only through approved adapter; unknown uses vetted explanation plus stage/code | FR-001, RULE-003 | AC-001, AC-021 | UT-001, UT-003, IT-011 |
| HTTP/Tap panic; response already started | Recovery-time stack and safe value; owner capture; do not rewrite started response | FR-001, FR-007, NFR-003 | AC-002, AC-008, AC-018 | AT-002, IT-003, REG-003 |
| No origin stack or request ID | Retain cause; indicate absent correlation/frame origin rather than fabricate values | FR-001, FR-003, FR-013 | AC-002, AC-003, AC-015 | UT-002, UT-006, IT-012 |
| Handler hangs; early rejection/unmatched/canceled request | Immediate safe arrival with ID; INFO completion if reached; route fallback and internal canceled/499 behavior preserved | FR-004, RULE-004, NFR-003 | AC-004, AC-020, AC-018 | AT-004, IT-001, REG-003 |
| Health probe success/failure | Logging wraps existing branch; DEBUG arrival/success; WARNING 4xx/ERROR 5xx or unexpected completion, same underlying handling | FR-004, FR-015 | AC-004, AC-022 | IT-002, IT-013 |
| Same public target in private action | Public target/URI/search term redacted in path and every companion field; safe account attribution only for operational failure | FR-009, RULE-002, RULE-004 | AC-011, AC-020 | UT-007, IT-007, IT-011 |
| Draft/failed publication versus published parse failure | Excerpt only from selected published failure data; draft/media fields excluded; no blocked/deleted-content fetch | FR-011, NFR-002 | AC-006, AC-013, AC-017 | UT-005, UT-013, IT-006, IT-016 |
| Account switched before failure finishes | Carry initiating lease/account through diagnostic input, never resolve a new account for reporting | FR-002, FR-009, RULE-001 | AC-001, AC-010, AC-012 | IT-010, IT-011 |
| Expected cancellation/offline/expiry/not-found/validation/retry | Appropriate contextual log/breadcrumb, no unexpected issue; terminal actionable error reportable | FR-007, FR-008 | AC-008, AC-009 | UT-010, UT-011, IT-009 |
| Missing/malformed envelope, network/parse exception | Concrete cause metadata; no HTML/full Response/RequestOptions; preserve useful route/method/request ID when available | FR-006, NFR-003 | AC-007, AC-018 | UT-009, IT-008, REG-001 |
| Telemetry disabled/throws/unavailable before initialization | Safe local path independent; guarded remote; no recursion/business-state mutation or buffer | FR-013, FR-016 | AC-015, AC-023 | IT-012, IT-015, REG-004, MAN-001 |
| Oversized/deep/cyclic/multibyte diagnostics | Selection before sanitization/truncation; UTF-8/encoded-byte bounds, explicit omissions, no credential fragments or whole media | FR-010, NFR-002, RULE-001 | AC-012, AC-013 | UT-012, UT-013, IT-011 |
| Repeated recoverable worker retries; terminal failure | Scoped bounded suppression and count/outcome summary; terminal full cause always retained; retry/ack decisions unchanged | FR-009, NFR-002, NFR-003 | AC-010, AC-017, AC-018 | UT-016, IT-016, REG-003 |
| Loading/empty/UI failure state | Diagnostic context remains separate from localized UI; existing widget layout/navigation/recovery behavior preserved | NFR-003 | AC-018 | REG-001, REG-002 |

## 9. Test Implementation Plan

The rows below cover every designed case. Existing suite locations and proposed file names are listed in `02-acceptance-tests.md`; new tests must exercise real callers/final payloads rather than reproduce implementation internals. Do one failing test/behavior slice at a time, implement the smallest supported contract, then refactor and move to the next slice. Do not add all speculative test scaffolding before proving the first failure.

| Order | Test IDs | Target | Setup / Fixture | Initial Expected Failure | Requirement IDs / Acceptance Criteria |
|---|---|---|---|---|---|
| 1 | UT-003, UT-001 | Go redaction/Sentry; Dart diagnostic sanitizer/reporter suites | Opaque private sentence in unknown/private source; separate known-safe wrapped cause with safe public reference beside credential | Existing sanitization either drops useful cause or allows raw unknown/private message; verify type/available stack/stage/code retained in both branches | FR-001, RULE-003 / AC-001, AC-021 |
| 2 | UT-002, UT-004, UT-005, AT-001, AT-002 | Go Sentry/recovery/Tap and application acceptance; Dart reporter/error handlers | TD-001–TD-003, available/missing stacks, published vs draft pairs | Generic types/messages/stacks or lost public actor/target; draft wrongly accepted as public if using value-only policy | BR-001, FR-001, FR-002, FR-011 / AC-001, AC-002, AC-006 |
| 3 | UT-007, AT-004, IT-001, IT-002 | Path policy/middleware plus `appview/cmd/appview/server_test.go` | Real catalogue/server chain, INFO buffer, held handler, public/private/search/unmatched and health routes | Arrival lacks path; completion DEBUG; probe branch has no logging; companion private context leaks | FR-004, RULE-004 / AC-004, AC-020 |
| 4 | UT-012, UT-013, AT-007, IT-011 | Go cross-sink app/log handler tests; Dart SDK emission/platform tests | Final serialized payload recorders; TD-002–TD-004, chosen budget boundaries and SDK-generated fields | Direct/local/SDK bypasses; credentials/private text or generic context; over-budget payload/core diagnostic loss | FR-010, FR-011, NFR-002, RULE-001–RULE-004 / AC-012, AC-013, AC-020, AC-021 |
| 5 | UT-008, AT-005, IT-005, IT-013 | Go logs and app wiring; Dart root log bridge/SDK transport | Logs on with issue sink suppressed in tests, tracing off/zero-rate; distinct messages/warning/selected INFO; disabled Logs control | Direct slog never exports; warnings absent; App log replaces meaningful message; supporting records wrongly create issues | FR-005, FR-015 / AC-005, AC-022 |
| 6 | UT-006, AT-003, IT-004 | Go application acceptance + Dart request-correlation suite | One fixture produced/verified from Go envelope and consumed through Dio/provider; release/env/trace variants | run_id removed, requestId lost between AppError/reporter/local/export sinks | BR-001, FR-003 / AC-003 |
| 7 | UT-010, UT-011, IT-003, IT-009 | Go middleware fallback/recovery; Dart real bootstrap/handlers/provider logger | Same occurrence log+capture, separate same-text/object failures, SDK-installed handlers, expected and exhausted failures | Duplicate issue, no uncaptured fallback, expected issue noise or terminal suppression | FR-001, FR-007, FR-008 / AC-002, AC-008, AC-009 |
| 8 | UT-009, IT-008, REG-001, REG-002 | Dart interceptor/AppError mapper; route manifest checks and initialization gate widget regression | TD-005 registered routes plus safe 4xx/5xx/fields and malformed bodies | Safe 5xx fields lost, newer route unknown, raw diagnostics shown to user | FR-006, NFR-003 / AC-007, AC-018 |
| 9 | IT-006, IT-010, UT-014 | API/auth/PDS/DB/Tap/index/ingestion and Flutter initialization/media/storage/mutation/public summary suites | Fault injection through each audited class, initiating lease A then switch B, video/provider/storage failures | Helpers discard cause/target/stage; initialization placeholder; wrong current-account attribution; public summaries opaque | FR-001, FR-002, FR-008, FR-009, FR-011, FR-012 / AC-001, AC-006, AC-009, AC-010, AC-014 |
| 10 | AT-006, IT-007, REG-003, REG-005 | Scheduledposts/accountdeletion/push/Instagram worker tests and preserved private/auth/video regressions | Real local Postgres/MinIO fixtures; private canaries; failure then routine success | Cause/job/account unavailable or private target/recipient exposed; success gains private history | FR-009, NFR-003, RULE-001, RULE-002 / AC-011, AC-012, AC-018 |
| 11 | UT-015, UT-016, IT-016 | Metrics/grouping/bounds and real retry callers | Many public IDs, fake clock, bounded suppression scope, terminal cause, counting network/media fakes | IDs enter aggregation, retry logs unbounded, terminal cause lost, extra fetch/full-media copy | NFR-001, NFR-002 / AC-013, AC-016, AC-017 |
| 12 | IT-012, IT-015, REG-004, REG-006 | Go log/Sentry and Dart bootstrap/reporter/platform/SDK option tests | No DSN, throwing/unavailable transport, pre-init/release-policy adapter | Local cause missing, recursion or raw debug/platform output; broadened automatic SDK collection | FR-010, FR-013, FR-016, RULE-001, RULE-002 / AC-012, AC-015, AC-023 |
| 13 | IT-014, MAN-002 | Canonical guide examples and contributor walkthrough | Implemented public/private/correlation/URL interfaces, root guide link | Guide missing, wrong interface example or policy contradiction | FR-014 / AC-019 |
| 14 | MAN-001, MAN-003, MAN-004 | `06-validation-evidence.md`; attached-device consoles and local/test investigation | Synthetic public dependency/provider/initialization/Tap/private publication failures; production checklist initially Pending | Cause/target/request trail cannot be found operationally; physical fallback unproven; local evidence incorrectly labeled production | FR-016, FR-017 / AC-023, AC-024 |

Go tests use `httptest`, buffered JSON slog, sentry.MockTransport and the in-memory metric recorder; workers reuse PostgreSQL/MinIO and package stubs. Dart tests use ProviderContainer, reporter doubles, Dio fixtures, injectable platform recorder and a Transport capturing SentryEnvelope. Fake transports must contain final serialized SDK output, not just captureException arguments. Restore global handlers/default loggers, flush queues, dispose containers and use barriers/fake clocks. Native/platform concerns require the release-device check separately.

For DR-003, assert Logs still export while issue capture is deliberately suppressed by an injected/no-op issue sink or test before-send hook. Keep DSN/log configuration intact; do not add a new production feature flag merely to run this test. Go's zero SampleRate is not an errors-off gate in the inspected SDK, so do not use it to simulate disabled issue capture. Include local-only, Logs-only observation, errors with Logs disabled, trace disabled/zero-rate and normally sampled controls. SDK batching may produce one envelope containing several records; assert record/event counts by kind.

Focused commands, to run during implementation:

- First Go slice from repo root: `cd appview && go test ./internal/observability -run 'TestDiagnostic(KnownSafeCause|UnknownPrivateMessage)' -count=1`. These are proposed new test names; establish their failing assertions before claiming red.
- First Dart slice: `just app-test test/observability/diagnostic_sanitizer_test.dart test/observability/sentry_error_reporter_test.dart`.
- Lifecycle/wiring from appview: `go test ./internal/observability ./internal/middleware ./internal/routes ./cmd/appview -run '<focused case>' -count=1`, using the checked-in toolchain. Go tests run on host, AppView dev server stays Docker-hosted.
- Fast checkpoint: `just appview-test-unit` is explicitly incomplete; PostgreSQL/MinIO tests are skipped there.
- Full affected Go validation: `just test` against the existing local compose test infrastructure; includes race detector and required integration services. Do not point tests at production.
- Flutter affected suites: `just app-test test/observability test/bootstrap/provider_logger_test.dart test/shared/api/providers/error_mapping_interceptor_test.dart test/shared/errors/app_error_mapper_test.dart` plus affected auth/feature/widget suites from the specification.
- `just app-analyze` after Dart changes. No code generation/lexgen or migration is required by this design; run normal provider generation only if implementation actually changes annotated declarations.
- The existing device integration command does not prove compiled-release platform console output; MAN-001 records that separately using a test release build.

## 10. Sequencing And Guardrails

- First TDD step: write UT-003/UT-001 failing behavior in one runtime, establish the source-aware contract, verify actual safe type/message/cause preservation and opaque-private-text exclusion, then mirror the approved contract in the other runtime.
- Dependencies: source policy and immutable record precede final sink adapters; protected handler exists from process initialization; route resolver precedes arrival-path logging; emission hooks precede wider call-site migration; explicit owners precede enabling independent exports; final interfaces precede guide examples. Local platform output is initialized before remote setup.
- Audit inventory: in `06-validation-evidence.md`, create rows F-01–F-20 with affected callers, source/workflow classification, permitted/forbidden context, capture owner, configured sinks and test/evidence ID. Resolve unclassified rows or record an actual blocker; representative tests alone do not close FR-009.
- Validation ownership (DR-005): the implementing agent runs focused/full automated tests, writes local/test investigation evidence and performs the guide example checks. The maintainer reviews the guide walkthrough and operator usefulness. The agent runs physical release checks if devices/tooling are available; otherwise the maintainer owns execution and the evidence remains Pending. Only the maintainer's separate authorization enables production checklist execution. No absent evidence is labeled Passed.
- Evidence schema: for each check record date/build/environment, status (Passed/Failed/Pending), command/setup, sink/gates, synthetic correlation/reference, sanitized excerpts, cause-location steps and reviewer. Include tests' actual results, not only intended assertions. Summarize exact remaining gaps at implementation handoff.
- Local/test investigations cover public read/dependency failure, Flutter provider and account initialization, Tap malformed record/indexer failure and private scheduled publication. API exercise proves Go/Flutter request trail; job/Tap exercises use their own safe event/workflow IDs. Do not manufacture HTTP request IDs for jobs.
- Production checklist: effective release/environment/Logs gates, operator retrieval of ordinary logs/issues, unsampled request correlation where observable, and telemetry access/retention suitability. Mark Pending until authorized rollout. No Render/Sentry mutation, synthetic production error injection, deployment or push is authorized by this plan.
- Guardrails: same secret/private policy in local/exported/debug/platform sinks; no raw object/error formatting escape hatch; no new diagnostic-only network fetches; no unsafe local-only attributes; no identity/payload metric/fingerprint keys; no duplicate log dispatch; no recursive exporter logs; no shared mutable active-account lookup at emit time.
- Preserve operational behavior: existing error envelopes/localized actions, initialization/fencing/auth/Tap acknowledgements, scheduling and worker retry/deletion semantics stay unchanged. Metrics definitions remain bounded. Guide policy supersedes only the diagnostic restrictions explicitly revised by these requirements.
- Out of scope: migrations, lexicons, persistence/schema redesign, telemetry retention/access changes, replay/screenshots, PDS auth architecture changes, additional vendor/runner, support UI or persistent application-managed device buffer.
- No automatic stage commit or push. Only this plan is added in the coding-plan stage; earlier uncommitted test/review documents remain untouched.

## 11. Risks And Open Questions

No blocking product or architectural question remains. Implementation details below have explicit decisions or verification tasks; revise this plan if evidence makes a chosen limit/adapter impractical, without relaxing the requirements.

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Safe-source adapters and workflow contexts can be misassigned | Credential/private prose disclosure or lost cause | DR-001 resolved by approved adapters, independent workflow classification, unknown defaults and paired useful/private assertions. Inventory every audited boundary during migration. |
| CPQ-002 | Non-blocking | Route/method/encoding classification may drift from runtime routing | Arrival leaks private targets or over-redacts public paths | DR-002 resolved by read-only catalogue resolver, conservative unmatched/ambiguous policy and registered-route fixture comparison with Dart. Preserve actual request routing behavior. |
| CPQ-003 | Non-blocking | Final SDK/native integration behavior may differ from reporter doubles | Duplicate issues or unsafe final fields | Local installed hooks/Transport verified; real SDK bootstrap and serialized envelope tests required. Inspect installed native/error integration behavior during implementation, choose one app callback owner, and keep unsupported automatic fields excluded. No dependency upgrade assumed. |
| CPQ-004 | Non-blocking | Proposed byte/frame/context and retry budgets need measured confirmation | Diagnostic loss, console truncation or excessive volume | DR-004 resolved with section 5 starting budgets and fake-clock policy. Prove UTF-8/serialized limits and positive usefulness; adjust lower sink bounds explicitly if needed, rerunning boundary tests. |
| CPQ-005 | Non-blocking | Startup handler/Observer coupling and SDK failure can recurse or double-export | Lost logs, duplicate records or runtime change | One shared state and export-only sink, guarded local-only failure path, race/wiring and failing-transport tests. No telemetry replay buffer. |
| CPQ-006 | Non-blocking | Physical release-device access may be unavailable to the implementing agent | Platform output verification remains incomplete | Run MAN-001 where available; otherwise maintainer owns device checks and Pending is reported in evidence. Adapter test success is not release-device evidence. |
| CPQ-007 | Non-blocking | Effective production Logs/access/retention not verified | Local behavior may not match deployed retrieval | MAN-004 separately pending; authorized maintainer rollout check only. No live production claim from fake transports. |
| CPQ-008 | Non-blocking | Generic SDK grouping may incorporate exception messages containing IDs | Fragmented issues despite stable metric context | Build stable operation/class/type/route fingerprints where necessary; IDs only context; vary identities and safe messages in UT-015 while preserving distinct actionable cause classes. |

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-10-02-logging-error-reporting/04-coding-plan.md`.
- TDD execution plan: `docs/changes/2026-10-02-logging-error-reporting/05-implementation-plan.md`, to be written by implement-tdd after approval; not created in this stage.
- Start with tests: UT-003 paired with UT-001, keeping all existing criterion/test IDs stable. Go proposed names `TestDiagnosticKnownSafeCause` and `TestDiagnosticUnknownPrivateMessage`; mirror the behavior in Dart sanitizer/reporter suites.
- Focused command: `cd appview && go test ./internal/observability -run 'TestDiagnostic(KnownSafeCause|UnknownPrivateMessage)' -count=1`, after adding the failing tests.
- Review-note disposition: DR-001 source policy in section 5; DR-002 catalogue resolver and probe wiring in section 5; DR-003 sink matrix in section 9; DR-004 budgets/suppression in section 5; DR-005 evidence ownership/location in section 10. No review note deferred without a task.
- Completion includes all Must requirements, positive useful diagnostics, final sink protection, call-site inventory, documented guide, local investigations and explicit status of physical-device/production checks. No production deploy or settings mutation follows automatically.
- Approval boundary: user selection of this stage authorizes the plan. Proceed to implement-tdd only after the user explicitly chooses implementation. The workflow's High-risk review policy requires that approval; this draft is not itself approval or implemented behavior.
