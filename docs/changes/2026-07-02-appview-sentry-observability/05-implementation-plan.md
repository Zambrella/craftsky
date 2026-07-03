# TDD Implementation Plan: AppView Sentry Observability Consolidation

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Keep production `sentry-go` imports inside `appview/internal/observability`.
- Preserve existing `/v1/*`, auth, PDS write, and Tap behavior except for observability side effects.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001 | FR-001, FR-015, NFR-002, RULE-001 | AC-001, AC-006, AC-014 | Fails: new Sentry flags and DSN-only semantics missing |
| 2 | UT-002 | BR-002, FR-006 | AC-005 | Fails: import boundary not encoded |
| 3 | UT-006 | BR-002, FR-004, FR-005, NFR-003, NFR-004, RULE-004 | AC-004, AC-005, AC-014 | Fails: metrics remain Prometheus-shaped |
| 4 | UT-003 | BR-003, FR-012, NFR-001, RULE-002, RULE-004, RULE-006 | AC-006, AC-012, AC-016 | Fails: shared sanitizer/validator incomplete |
| 5 | UT-005 | FR-012, FR-014, NFR-001, RULE-006 | AC-012, AC-016 | Fails: no shared sentinel classifier |
| 6 | UT-004 | BR-003, FR-002, FR-013, RULE-003 | AC-007 | Fails: panic redaction assertions not complete |
| 7 | UT-007 | FR-007, FR-008, NFR-001, RULE-002, RULE-004, RULE-005 | AC-006, AC-008, AC-009, AC-016 | Fails: tracing normalization incomplete |
| 8 | IT-002 | BR-001, BR-003, FR-002, FR-007, FR-008, FR-012, FR-013, RULE-002, RULE-005, RULE-006 | AC-002, AC-006, AC-007, AC-008, AC-009, AC-012 | Fails: HTTP telemetry lacks safe child spans/events |
| 9 | IT-003 | BR-001, BR-002, FR-004, NFR-001, RULE-002, RULE-004 | AC-004, AC-005, AC-006, AC-016 | Fails: Sentry/in-memory metric implementation not wired broadly |
| 10 | UT-008 | FR-009, NFR-001 | AC-011, AC-016 | Fails: DB helper has no span/result-class behavior |
| 11 | IT-004 | BR-002, FR-008, FR-009, RULE-002 | AC-009, AC-011 | Fails: selected storage paths lack DB spans |
| 12 | IT-005 | BR-003, FR-008, FR-010, FR-012, RULE-002, RULE-006 | AC-006, AC-010, AC-012 | Fails: PDS/OAuth telemetry incomplete |
| 13 | UT-009 | FR-003, FR-018, NFR-005, RULE-007 | AC-003, AC-006 | Fails: Sentry log sink/filter missing |
| 14 | UT-010 | FR-011, NFR-006 | AC-010, AC-014 | Fails: Tap trace controls missing |
| 15 | IT-006 | FR-008, FR-011, FR-012, FR-013, NFR-006, RULE-002, RULE-003, RULE-006 | AC-006, AC-007, AC-010, AC-012, AC-014 | Fails: Tap/indexer spans and sampling incomplete |
| 16 | IT-007 | FR-017 | AC-013 | Fails: `/metrics` still registered |
| 17 | UT-011 | FR-017 | AC-013 | Fails: Prometheus references still present |
| 18 | AT-004 / REG suite | FR-016 | AC-015 | May fail if instrumentation changes product behavior |

## Implementation Steps

### Step 1: UT-001
- Write failing test: config matrix for no DSN, DSN-only, explicit logs/tracing/metrics flags, invalid sample rates, and Tap trace controls.
- Run command: `cd appview && go test ./internal/app ./internal/observability -run 'Test.*Sentry|Test.*Config' -count=1`
- Confirmed failure: `go test ./internal/app -run TestLoadConfig_ObservabilityDefaultsAndValidation -count=1` failed because `Config` did not expose logs, metrics, or Tap tracing flags. `go test ./internal/observability -run TestSentryPillarGates -count=1` then failed because observer construction did not expose or apply those gates.
- Implement: Added `SENTRY_LOGS_ENABLED`, `SENTRY_METRICS_ENABLED`, `SENTRY_TAP_TRACING_ENABLED`, and `SENTRY_TAP_TRACES_SAMPLE_RATE` parsing with DSN-disabled fallbacks; passed the gates into `observability.New`; stored effective gate state on `Observer`; set Sentry `DisableLogs` and `DisableMetrics` from explicit flags.
- Run command: `cd appview && go test ./internal/app ./internal/observability -run 'TestLoadConfig_ObservabilityDefaultsAndValidation|TestSentryPillarGates' -count=1`
- Refactor: Ran `gofmt` on touched Go files.
- Notes: DSN-only behavior now enables Sentry client/error capture while keeping logs, tracing, metrics, and Tap tracing disabled by default.

### Step 2: UT-002
- Write failing test: repository import scan that permits production `sentry-go` imports only in `appview/internal/observability`.
- Run command: `cd appview && go test ./internal/observability -run TestSentryImportBoundary -count=1`
- Confirmed failure: The new guard passed immediately because production `sentry-go` imports were already limited to `internal/observability`.
- Implement: Added `import_boundary_test.go` to preserve that boundary for the rest of this implementation.
- Run command: `cd appview && go test ./internal/observability -run TestSentryImportBoundary -count=1`
- Refactor: Ran `gofmt` on the new test file.
- Notes: This loop did not require product-code changes; it locks the approved import allowlist before broader instrumentation work.

### Step 3: UT-006
- Write failing test: no-op, in-memory, and Sentry metric recorder behavior through AppView-domain methods.
- Run command: `cd appview && go test ./internal/observability -run Test.*Metric -count=1`
- Confirmed failure: `go test ./internal/observability -run 'TestInMemoryMetrics|TestObserverSentryMetrics' -count=1` failed because `NewInMemoryMetricRecorder` and `MetricCall` did not exist.
- Implement: Added the `MetricRecorder` interface, no-op recorder, in-memory recorder, and Sentry recorder using `sentry.NewMeter`; connected existing HTTP, DB, PDS, and Tap metric facade methods to the recorder while keeping Prometheus collectors temporarily for later removal loops.
- Run command: `cd appview && go test ./internal/observability -run 'TestInMemoryMetrics|TestObserverSentryMetrics' -count=1`; then `cd appview && go test ./internal/observability -count=1`
- Refactor: Ran `gofmt` on touched observability files.
- Notes: DSN-only observers emit no Sentry metrics. `MetricsEnabled` observers emit reused `craftsky_appview_*` names through the Sentry SDK. Prometheus remains present only as transitional compatibility until IT-007/UT-011.

### Step 4: UT-003
- Write failing test: sanitizer and strict validator reject forbidden values and normalize runtime values.
- Run command: `cd appview && go test ./internal/observability -run 'Test.*Redact|Test.*Validat|Test.*Sanit' -count=1`
- Confirmed failure: `go test ./internal/observability -run 'TestSanitizeEventContext|TestCaptureErrorUsesSentryTransport|TestValidateMetricCall|TestMetricRecorderNormalizes' -count=1` failed because `ValidateMetricCall` did not exist.
- Implement: Added metric validation helpers; removed `run_id` from Sentry-bound event context; tightened runtime metric normalization for unsafe route, operation, stage, category, status, and result values.
- Run command: `cd appview && go test ./internal/observability -run 'TestSanitizeEventContext|TestCaptureErrorUsesSentryTransport|TestValidateMetricCall|TestMetricRecorderNormalizes' -count=1`; then `cd appview && go test ./internal/observability -count=1`
- Refactor: Ran `gofmt` on touched observability files.
- Notes: Runtime metrics normalize unsafe values to bounded fallbacks; validation-focused tests fail loudly on forbidden/high-cardinality metric attributes.

### Step 5: UT-005
- Write failing test: sentinel classifier covers auth/session, validation, rate limit, not found, forbidden, timeout, network, PDS server, Tap/indexer, DB, and unexpected categories without raw error details.
- Run command: `cd appview && go test ./internal/observability -run Test.*Classif -count=1`
- Confirmed failure: `go test ./internal/observability -run 'TestClassifyError|TestCaptureErrorUsesClassified' -count=1` failed because `ClassifyError` did not exist.
- Implement: Added `ClassifiedError` and `ClassifyError` with bounded categories/codes for auth/session, PDS not found, timeout, network, DB no rows, validation, Tap, indexer, server, and unexpected cases; updated `CaptureError` to send `AppViewError` with safe `error_code`/category/stage/result fields instead of raw or generic exception values.
- Run command: `cd appview && go test ./internal/observability -run 'TestClassifyError|TestCaptureErrorUsesClassified' -count=1`; then `cd appview && go test ./internal/observability -count=1`
- Refactor: Ran `gofmt` on classifier and Sentry files.
- Notes: Classifier messages and exception values are sentinels and do not include `err.Error()` details.

### Step 6: UT-004
- Write failing test: recovered panic capture includes type and safe context but never recovered value strings.
- Run command: `cd appview && go test ./internal/observability ./internal/middleware ./internal/tap -run 'Test.*Panic|Test.*Recover' -count=1`
- Confirmed failure: `go test ./internal/observability ./internal/middleware -run 'TestCapturePanic|TestRecovery' -count=1` failed because panic capture used the recovered Go type as the exception type and did not emit a bounded `recovered_type` tag.
- Implement: Changed panic events to use fixed exception type `AppViewPanic`, fixed value `redacted`, bounded `recovered_type`, and no Sentry `run_id`; updated HTTP and 5xx middleware expectations to keep `run_id` local-only while requiring safe classifier fields in Sentry.
- Run command: `cd appview && go test ./internal/observability ./internal/middleware -run 'TestCapturePanic|TestRecovery' -count=1`; `cd appview && go test ./internal/tap -run 'Test.*Panic|Test.*panick|Test.*Consumer' -count=1`; `cd appview && go test ./internal/observability ./internal/middleware ./internal/tap -count=1`
- Refactor: Ran `gofmt` on touched observability and middleware test files.
- Notes: Panic Sentry events now expose recovered type only, never recovered value strings or request IDs.

### Step 7: UT-007
- Write failing test: transaction/span names and attributes are bounded and raw paths/identifiers are omitted.
- Run command: `cd appview && go test ./internal/observability -run 'Test.*Trace|Test.*Route|Test.*Span' -count=1`
- Confirmed failure: `go test ./internal/observability -run TestStartSpanNormalizesUnsafeOperationAndAttributes -count=1` failed because `Span.SetTransactionName` exported a raw path containing a DID and query string.
- Implement: Added transaction-name normalization and event/span attribute normalization for operations, components, route patterns, methods, statuses, status classes, categories, stages, result values, and error codes.
- Run command: `cd appview && go test ./internal/observability -run TestStartSpanNormalizesUnsafeOperationAndAttributes -count=1`; then `cd appview && go test ./internal/observability ./internal/middleware -count=1`
- Refactor: Ran `gofmt` on touched observability files.
- Notes: Defensive tracing normalization now maps unsafe transaction names to bounded forms such as `GET unmatched` and unsafe span operations to `unknown`.

### Step 8: IT-002
- Write failing test: HTTP middleware and representative handlers emit safe events and route-pattern traces.
- Run command: `cd appview && go test ./internal/middleware ./internal/api -run 'Test.*Observability|Test.*Telemetry|Test.*Panic' -count=1`
- Confirmed failure: `go test ./internal/middleware -run TestHTTPMetrics_ExportsSentryTransactionWithRoutePattern -count=1` failed because the HTTP transaction had no child span.
- Implement: Added an `http.handler` child span around downstream HTTP handler execution, including safe method/route/status/result attributes and panic-path error finishing.
- Run command: `cd appview && go test ./internal/middleware -run TestHTTPMetrics_ExportsSentryTransactionWithRoutePattern -count=1`; `cd appview && go test ./internal/middleware -count=1`; `cd appview && go test ./internal/api ./internal/middleware ./internal/observability -run 'Test.*Observability|Test.*Sentry|Test.*Telemetry|Test.*Panic|TestHTTPMetrics|TestRecovery' -count=1`
- Refactor: Ran `gofmt` on `internal/middleware/metrics.go` and tests.
- Notes: Request traces now contain a top-level route-pattern transaction plus a bounded `http.handler` work span.

### Step 9: IT-003
- Write failing test: HTTP, DB, PDS, and Tap/indexer metric calls use bounded AppView-domain methods.
- Run command: `cd appview && go test ./internal/observability ./internal/api ./internal/tap -run 'Test.*Metric|Test.*Observability' -count=1`
- Confirmed failure: The representative `Observer` recorder test passed immediately because `UT-006` had already wired the facade paths.
- Implement: Added `TestObserverMetricRecorderCoversRepresentativeOperations` to exercise HTTP, DB, PDS, Tap, and indexer metric calls via an injected in-memory recorder.
- Run command: `cd appview && go test ./internal/observability -run TestObserverMetricRecorderCoversRepresentativeOperations -count=1`; then `cd appview && go test ./internal/observability ./internal/api ./internal/tap -run 'Test.*Metric|Test.*Observability' -count=1`
- Refactor: Ran `gofmt` on the updated metrics test.
- Notes: No additional product-code changes were needed for this loop.

### Step 10: UT-008
- Write failing test: DB helper emits named operation spans with result classes and no SQL/query/user content.
- Run command: `cd appview && go test ./internal/observability -run Test.*DB -count=1`
- Confirmed failure: `go test ./internal/observability -run TestObserveDBCreatesBoundedStorageSpan -count=1` failed because `DBOperation` had no `ResultClass` field and `ObserveDB` did not create spans.
- Implement: Added `DBOperation.ResultClass`; made `ObserveDB` create bounded `db.<operation>` spans with safe operation, route, result, and duration attributes while preserving metric emission and nil-observer behavior.
- Run command: `cd appview && go test ./internal/observability -run TestObserveDBCreatesBoundedStorageSpan -count=1`; then `cd appview && go test ./internal/observability -count=1`
- Refactor: Ran `gofmt` on DB observability files.
- Notes: DB spans omit SQL/query text, exact row counts, `run_id`, identifiers, and raw content.

### Step 11: IT-004
- Write failing test: selected storage operations create DB spans.
- Run command: `cd appview && go test ./internal/api ./internal/auth -run 'Test.*Store|Test.*Search|Test.*Profile|Test.*Timeline|Test.*Session' -count=1`
- Confirmed failure: The focused search-store span assertion passed after `UT-008` because `SearchStore` already used `ObserveDB`.
- Implement: Extended `TestSearchStore_SearchPostsEmitsDBOperationTelemetry` to run under an active Sentry trace and assert the bounded `db.search.posts` child span in addition to existing metric assertions.
- Run command: `cd appview && go test ./internal/api -run TestSearchStore_SearchPostsEmitsDBOperationTelemetry -count=1`; then `cd appview && go test ./internal/api ./internal/auth -run 'Test.*Store|Test.*Search|Test.*Profile|Test.*Timeline|Test.*Session' -count=1`
- Refactor: Ran `gofmt` on the updated search-store test.
- Notes: Scope stayed on the existing observer-aware storage path to avoid broad store constructor churn in this loop.

### Step 12: IT-005
- Write failing test: PDS/OAuth and blob upload paths produce safe spans/events/log attrs.
- Run command: `cd appview && go test ./internal/observability ./internal/api ./internal/auth -run 'Test.*PDS|Test.*Blob|Test.*OAuth|Test.*Session' -count=1`
- Confirmed failure: Existing focused PDS/blob/auth telemetry tests passed after the prior classifier and sanitizer loops.
- Implement: Added an assertion that PDS unexpected failure events include the bounded `error_code` sentinel alongside category/stage/result fields.
- Run command: `cd appview && go test ./internal/observability ./internal/api ./internal/auth -run 'Test.*PDS|Test.*Blob|Test.*OAuth|Test.*Session' -count=1`
- Refactor: Ran `gofmt` on the updated PDS wrapper test.
- Notes: PDS events/log checks continue to exclude DIDs, session IDs, record bodies, and raw upstream text.

### Step 13: UT-009
- Write failing test: local stdout logs remain while Sentry-bound log sink filters unsafe attrs.
- Run command: `cd appview && go test ./internal/middleware ./internal/observability -run 'Test.*Log' -count=1`
- Confirmed failure: `go test ./internal/observability -run TestSentryLogsRequireExplicitGateAndFilterAttributes -count=1` failed because `Observer.EmitLog` did not exist.
- Implement: Added narrow `LogSink`, no-op sink, Sentry sink using `sentry.NewLogger`, and `Observer.EmitLog`; wired it behind `LogsEnabled`; connected PDS write completion logs to the filtered sink while preserving local stdout logs.
- Run command: `cd appview && go test ./internal/observability -run TestSentryLogsRequireExplicitGateAndFilterAttributes -count=1`; then `cd appview && go test ./internal/middleware ./internal/observability -run 'Test.*Log|TestSentryLogs|TestWrapPDSFactory' -count=1`
- Refactor: Ran `gofmt` on log and PDS observability files.
- Notes: The SDK `slog` integration package is not present in current `sentry-go` v0.47, so this uses the approved local filtered sink fallback.

### Step 14: UT-010
- Write failing test: Tap trace volume controls and sampling decisions.
- Run command: `cd appview && go test ./internal/tap ./internal/observability -run 'Test.*Tap|Test.*Sample|Test.*Trace' -count=1`
- Confirmed failure: `go test ./internal/observability -run TestTapTraceControlsSampleSuccessButKeepForcedErrors -count=1` failed because `Observer.StartTapSpan` did not exist.
- Implement: Added `StartTapSpan` with Tap-specific success sampling and forced error/panic tracing bypass.
- Run command: `cd appview && go test ./internal/observability -run TestTapTraceControlsSampleSuccessButKeepForcedErrors -count=1`; then `cd appview && go test ./internal/observability ./internal/tap -run 'Test.*Tap|Test.*Sample|Test.*Trace' -count=1`
- Refactor: Ran `gofmt` on Tap observability files.
- Notes: Success-path Tap spans require `TapTracingEnabled` and sample rate; forced error/panic spans still start when tracing is enabled.

### Step 15: IT-006
- Write failing test: Tap consumer and indexer spans/errors/panics preserve ack/retry/drop behavior.
- Run command: `cd appview && go test ./internal/tap ./internal/index -run 'Test.*Tap|Test.*Index|Test.*Consumer' -count=1`
- Confirmed failure: `go test ./internal/tap -run TestWSConsumer_ExportsSentryConsumeAndIndexerSpans -count=1` failed because the Tap transaction only had the indexer span.
- Implement: Added Tap receive, decode, and ack spans; routed indexer spans through `StartTapSpan`; kept existing panic/error capture and ack/retry/drop behavior.
- Run command: `cd appview && go test ./internal/tap -run TestWSConsumer_ExportsSentryConsumeAndIndexerSpans -count=1`; then `cd appview && go test ./internal/tap ./internal/index -run 'Test.*Tap|Test.*Index|Test.*Consumer|Test.*Panic' -count=1`
- Refactor: Ran `gofmt` on Tap consumer and test files.
- Notes: The span assertion checks required boundary spans by operation rather than exact span count because cancellation can produce an additional final receive span.

### Step 16: IT-007
- Write failing test: `/metrics` route is absent and no replacement metrics endpoint exists.
- Run command: `cd appview && go test ./internal/routes ./cmd/appview -run 'Test.*Metrics|Test.*Route|Test.*Server' -count=1`
- Confirmed failure: `go test ./internal/routes ./cmd/appview -run 'Test.*Metrics|TestNewServer_HTTPMetricsUseRoutePattern' -count=1` failed because `/metrics` still returned Prometheus text.
- Implement: Removed `GET /metrics` registration from `routes.AddRoutes`; updated route/server tests to assert route absence and inspect route-pattern metrics through the in-memory recorder instead of Prometheus exposition.
- Run command: `cd appview && go test ./internal/routes ./cmd/appview -run 'Test.*Metrics|TestNewServer_HTTPMetricsUseRoutePattern' -count=1`; then `cd appview && go test ./internal/routes ./cmd/appview -count=1`
- Refactor: Ran `gofmt` on route/server files.
- Notes: No replacement local metrics endpoint was added.

### Step 17: UT-011
- Write failing test: Prometheus dependencies, collectors, handlers, and docs references are removed from production runtime.
- Run command: `cd appview && go test ./internal/observability -run TestPrometheusRemoval -count=1`
- Confirmed failure: `go test ./internal/observability -run TestPrometheusRemoval -count=1` failed on Prometheus collectors/imports, `MetricsHandler`, and Prometheus-shaped tests.
- Implement: Added the Prometheus removal guard; replaced Prometheus-backed `Observer` with local recorder/Sentry/log lifecycle implementation; removed `MetricsHandler`; converted tests to in-memory metric assertions; removed `/metrics` exposition dependencies from runtime code.
- Run command: `cd appview && go test ./internal/observability -run TestPrometheusRemoval -count=1`; `cd appview && go test ./internal/observability -count=1`; `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`
- Refactor: Ran `gofmt`; ran `go mod tidy`.
- Notes: Prometheus remains only as an indirect module dependency through `github.com/bluesky-social/indigo/atproto/identity`; AppView no longer imports Prometheus, exposes `/metrics`, or owns Prometheus collectors/handlers.

### Step 18: AT-004 / REG Suite
- Write failing test: no new test expected unless a regression gap appears.
- Run command: `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`
- Confirmed failure: No product-behavior regression appeared in the focused AppView regression suite.
- Implement: Updated AppView observability README and environment comments to document Sentry-only external observability, DSN-only errors/panics, independent logs/tracing/metrics/Tap trace gates, local stdout logs, and `/metrics` removal. Added a final regression test and fix for HTTP in-flight metrics so `EndHTTPRequest` records the end signal through the local metric recorder after Prometheus removal.
- Run command: `cd appview && go test ./internal/observability -run 'TestObserverHTTPInFlightRecordsStartAndEndValues|TestInMemoryMetricsRecordAppViewDomainMethods|TestObserverSentryMetricsRequireExplicitMetricsGate|TestObserverMetricRecorderCoversRepresentativeOperations' -count=1`; `just fmt`; `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`; `cd appview && go test ./internal/observability -run TestPrometheusRemoval -count=1`; `git diff --check`
- Refactor: Restored the HTTP in-flight end path through the local recorder before final verification; no additional refactor required after the rerun.
- Notes: `just fmt` passed and ran `go vet ./...`. The full `just test` recipe was not run because it requires the compose Postgres to already be running on `localhost:5433`; the focused regression suite from the planning docs passed.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [ ] Review completed or explicitly skipped
