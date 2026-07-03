# TDD Implementation Plan: AppView Observability

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
- Keep telemetry fields on the approved safe allowlist.
- Keep `/metrics` unauthenticated in AppView code and document production network/proxy restriction later in this stage.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 / AT-002 | FR-003, FR-004, FR-015, RULE-001, RULE-011 | AC-002, AC-011, AC-017 | Fails: `/metrics` is not registered |
| 2 | REG-002 | FR-003, FR-015, RULE-001 | AC-002, AC-011 | Fails until `/metrics` bypasses v1 auth and representative v1 behavior is preserved |
| 3 | UT-010 / UT-011 | FR-016, NFR-005, RULE-009, RULE-011 | AC-002, AC-017, AC-018 | Fails until registry exposes `craftsky_appview` metrics without first-slice non-goals |
| 4 | UT-008 / AT-010 | FR-012, FR-013, FR-017, RULE-003, RULE-010 | AC-007, AC-016 | Fails until Sentry config defaults and sampling validation exist |
| 5 | UT-002 / AT-005 | BR-002, NFR-002, RULE-002, RULE-005 | AC-005, AC-012 | Fails until redaction helpers and body logging guard exist |
| 6 | UT-001 / AT-001 | FR-001, FR-002 | AC-001, AC-018 | Fails until request log field construction uses stable fields and route patterns |
| 7 | UT-003 / AT-006 | FR-004, FR-018, NFR-001 | AC-006, AC-014 | Fails until route pattern fallback and allowlist helpers are implemented |
| 8 | IT-002 | FR-004, FR-018, NFR-001 | AC-006, AC-014 | Fails until middleware telemetry uses route patterns end-to-end |
| 9 | UT-012 / AT-012 | FR-006, FR-020 | AC-019 | Fails until DB operation timing helpers support `search.posts` |
| 10 | IT-008 | BR-001, FR-006, FR-020 | AC-019 | Fails until search route emits comparable HTTP and DB timings |
| 11 | UT-009 / AT-009 | FR-009, FR-010 | AC-004 | Fails until HTTP panic recovery emits safe envelope/log/capture |
| 12 | UT-005 | FR-006, FR-008, FR-019 | AC-010, AC-013 | Fails until PDS stage/category vocabulary exists |
| 13 | UT-007 | FR-006, FR-019 | AC-010, AC-013 | Fails until every current PDS write operation has a bounded name |
| 14 | IT-004 / AT-008 | FR-006, FR-019 | AC-010, AC-013, AC-017 | Fails until PDS/OAuth write paths emit bounded telemetry |
| 15 | UT-006 / AT-007 | FR-005 | AC-002, AC-017 | Fails until Tap/indexer label classification exists |
| 16 | GAP-001 / IT-003 | FR-011 | AC-004 | Fails until Tap/indexer panic boundary is explicit and tested |
| 17 | UT-004 / AT-004 | BR-002, FR-009, RULE-004, RULE-007 | AC-004, AC-015 | Fails until Sentry event allowlist exists |
| 18 | IT-005 / AT-003 | FR-002, FR-006, FR-008 | AC-003, AC-010, AC-018 | Fails until Sentry trace/span correlation exists |
| 19 | IT-007 | FR-007, FR-009, NFR-004 | AC-003, AC-004, AC-016 | Fails until Sentry lifecycle and flush behavior exist |
| 20 | AT-011 / REG-004 | NFR-003, RULE-006 | AC-009 | Should pass after instrumentation preserves existing behavior |
| 21 | MAN-003 | FR-014 | AC-008 | Manual dev Docker check pending |
| 22 | MAN-004 | FR-015, RULE-008 | AC-011 | Manual documentation review pending |
| 23 | MAN-005 | FR-016, NFR-005 | AC-002 | Manual metric documentation review pending |

## Implementation Steps

## Post-Review Fix Plan

The implementation review in `06-implementation-review.md` found four Must-level gaps and one manual evidence gap. This pass continues the same TDD stage and keeps the original requirements/test IDs as the source of truth.

| Fix Step | Review ID | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| R1 | IR-003 | AT-002, IT-001, IT-002 | FR-004 | AC-002, AC-006, AC-014 | Fails: in-flight gauge is zero during an active request |
| R2 | IR-001 | AT-004, AT-009, IT-007 | FR-002, FR-009, FR-010, FR-011 | AC-004, AC-015, AC-016 | Fails: Sentry events are sanitized locally but not captured by a real SDK-backed client |
| R3 | IR-002 | AT-007, UT-006, IT-003 | FR-005, FR-011 | AC-002, AC-004, AC-017 | Fails: Tap/indexer metrics are not emitted |
| R4 | IR-004 | AT-003, AT-008, IT-004, IT-005 | FR-006, FR-008, FR-019 | AC-010, AC-013, AC-017 | Fails: PDS/OAuth telemetry is metrics-only and lacks logs/spans plus handler-path evidence |
| R5 | IR-005 | MAN-003 | FR-014 | AC-008 | Pending: dev Docker `/metrics` scrape not executed |

### Step 1: IT-001 / AT-002
- Write failing test: `TestAddRoutes_MetricsIsPublicOpsEndpoint` in `appview/internal/routes/routes_test.go`.
- Run command: `cd appview && go test ./cmd/appview ./internal/routes -count=1`
- Confirmed failure: `GET /metrics` returned `404 page not found`.
- Implement: Added `appview/internal/observability` with an isolated Prometheus registry and registered `GET /metrics` as a public ops route.
- Run command: `cd appview && go test ./internal/routes -run TestAddRoutes_MetricsIsPublicOpsEndpoint -count=1`; `cd appview && go test ./cmd/appview ./internal/routes -count=1`.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: `/metrics` returns Prometheus text exposition and includes `craftsky_appview_build_info`; route is outside `/v1/*` and does not use the Craftsky JSON auth envelope.

### Step 2: REG-002
- Write failing test: `TestAddRoutes_MetricsBypassesV1AuthButV1RoutesDoNot` in `appview/internal/routes/routes_test.go`.
- Run command: `cd appview && go test ./internal/routes -run TestAddRoutes_MetricsBypassesV1AuthButV1RoutesDoNot -count=1`.
- Confirmed failure: First version assumed `missing_device_id` when both auth and device headers were absent; current middleware correctly returned `unauthorized` first.
- Implement: Corrected the regression setup to supply `Authorization` and omit only `X-Craftsky-Device-Id`.
- Run command: `cd appview && go test ./internal/routes -run TestAddRoutes_MetricsBypassesV1AuthButV1RoutesDoNot -count=1`.
- Refactor: Ran `gofmt` on `routes_test.go`.
- Notes: `/metrics` remains public without app headers; `/v1/whoami` still requires the existing v1 headers.

### Step 3: UT-010 / UT-011
- Write failing test: `TestMetricsUseCraftskyAppViewNamesAndUnits` in `appview/internal/observability/metrics_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestMetricsUseCraftskyAppViewNamesAndUnits -count=1`.
- Confirmed failure: Compile failed because `Observer.ObserveHTTPRequest` did not exist.
- Implement: Added HTTP request count, duration, response-size, and in-flight collectors under the `craftsky_appview` prefix plus `ObserveHTTPRequest`.
- Run command: `cd appview && go test ./internal/observability -run TestMetricsUseCraftskyAppViewNamesAndUnits -count=1`; `cd appview && go test ./internal/routes ./internal/observability -count=1`.
- Refactor: Ran `gofmt` on touched observability files.
- Notes: Registry exposes internal Prometheus metrics only; no Sentry Application Metrics or Prometheus exemplar support is initialized.

### Step 4: UT-008 / AT-010
- Write failing test: `TestLoadConfig_ObservabilityDefaultsAndValidation` in `appview/internal/app/config_test.go`.
- Run command: `cd appview && go test ./internal/app -run TestLoadConfig_ObservabilityDefaultsAndValidation -count=1`.
- Confirmed failure: Compile failed because Sentry and unsafe response-body logging fields did not exist on `app.Config`.
- Implement: Added `SENTRY_DSN`, `SENTRY_RELEASE`, `SENTRY_TRACING_ENABLED`, `SENTRY_TRACES_SAMPLE_RATE`, and `APPVIEW_UNSAFE_LOG_RESPONSE_BODIES` parsing with disabled defaults, sample-rate validation, prod conservative trace sampling, and prod-forced unsafe body logging off.
- Run command: `cd appview && go test ./internal/app -run TestLoadConfig_ObservabilityDefaultsAndValidation -count=1`; `cd appview && go test ./internal/app ./internal/routes ./internal/observability -count=1`.
- Refactor: Ran `gofmt` on touched config files.
- Notes: Sentry remains disabled when DSN is absent; no OpenTelemetry/OTLP configuration is required.

### Step 5: UT-002 / AT-005
- Write failing test: `TestRedactHeadersRemovesSensitiveTelemetryValues` in `appview/internal/observability/redaction_test.go`, then `TestLogging_DoesNotLogResponseBodyByDefault` in `appview/internal/middleware/logging_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestRedactHeadersRemovesSensitiveTelemetryValues -count=1`; `cd appview && go test ./internal/middleware -run TestLogging_DoesNotLogResponseBodyByDefault -count=1`.
- Confirmed failure: Header helper was undefined; after helper implementation, response-body test showed debug logs included `json_payload`.
- Implement: Added shared header redaction for Authorization, Cookie, DPoP, Craftsky device/session token headers, switched logging to the helper, and removed default response-body capture from request logging.
- Run command: `cd appview && go test ./internal/middleware -run 'TestLogging_(DoesNotLogResponseBodyByDefault|RedactsAuthorizationAndDoesNotLogRequestBody)' -count=1`; `cd appview && go test ./internal/middleware ./internal/observability -count=1`.
- Refactor: Canonicalized redacted header keys so `http.Header.Get` works reliably; ran `gofmt` on touched files.
- Notes: Request and response bodies are omitted by default; prod unsafe response-body logging is already forced off by config.

### Step 6: UT-001 / AT-001
- Write failing test: `TestLogging_CompletionUsesStableFieldsAndRoutePattern` in `appview/internal/middleware/logging_test.go`.
- Run command: `cd appview && go test ./internal/middleware -run TestLogging_CompletionUsesStableFieldsAndRoutePattern -count=1`.
- Confirmed failure: Completion logs used raw `path`, request detail logs used raw query, and no `route_pattern` field was emitted.
- Implement: Removed raw path/query fields from request logs, retained run_id propagation, read the mux pattern from the downstream request, normalized the method prefix, and logged `route_pattern` on completion with `unmatched` fallback.
- Run command: `cd appview && go test ./internal/middleware -run TestLogging_CompletionUsesStableFieldsAndRoutePattern -count=1`; `cd appview && go test ./internal/middleware -count=1`.
- Refactor: Updated the legacy logging test to assert `route_pattern` fallback instead of raw path; ran `gofmt`.
- Notes: Handler logs can share the same `run_id`; completion logs include method, route pattern, status, bytes, duration, and run_id without raw identifiers or query strings.

### Step 7: UT-003 / AT-006
- Write failing test: `TestRoutePatternUsesMuxPatternOrUnmatched` in `appview/internal/observability/route_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestRoutePatternUsesMuxPatternOrUnmatched -count=1`.
- Confirmed failure: Compile failed because shared `RoutePattern` did not exist.
- Implement: Added `observability.RoutePattern` to normalize Go ServeMux method-qualified patterns and return `unmatched` when no pattern exists; switched logging to use the shared helper.
- Run command: `cd appview && go test ./internal/observability -run TestRoutePatternUsesMuxPatternOrUnmatched -count=1`; `cd appview && go test ./internal/middleware -count=1`.
- Refactor: Removed the local middleware-only route pattern helper; ran `gofmt`.
- Notes: Raw paths and query strings are not needed for route labels; unmatched requests get a bounded fallback.

### Step 8: IT-002
- Write failing test: `TestNewServer_HTTPMetricsUseRoutePattern` in `appview/cmd/appview/server_test.go`.
- Run command: `cd appview && go test ./cmd/appview -run TestNewServer_HTTPMetricsUseRoutePattern -count=1`.
- Confirmed failure: `/metrics` had only `craftsky_appview_build_info`; no HTTP request metrics were recorded.
- Implement: Added `middleware.HTTPMetrics`, recording method, route pattern, status, duration, and response bytes through `Observer.ObserveHTTPRequest`; wired it into `NewServer`.
- Run command: `cd appview && go test ./cmd/appview -run TestNewServer_HTTPMetricsUseRoutePattern -count=1`; `cd appview && go test ./cmd/appview ./internal/middleware ./internal/routes ./internal/observability -count=1`.
- Refactor: Ran `gofmt` on touched server and middleware files.
- Notes: HTTP metrics use `/v1/posts/{did}/{rkey}` for identifier-bearing paths and omit raw DID, rkey, and query values.

### Step 9: UT-012 / AT-012
- Write failing test: `TestObserveDBOperationRecordsBoundedComparableTelemetry` in `appview/internal/observability/db_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestObserveDBOperationRecordsBoundedComparableTelemetry -count=1`.
- Confirmed failure: Compile failed because `DBOperation` and `Observer.ObserveDB` did not exist.
- Implement: Added `craftsky_appview_db_operation_duration_seconds` and `ObserveDB` with bounded `operation`, `route_pattern`, and `result` labels.
- Run command: `cd appview && go test ./internal/observability -run TestObserveDBOperationRecordsBoundedComparableTelemetry -count=1`; `cd appview && go test ./internal/observability -count=1`.
- Refactor: Ran `gofmt` on touched observability files.
- Notes: DB metrics omit SQL, query text, raw identity, and request payloads. `RunID` is retained in the operation struct for future log/span correlation, not used as a Prometheus label.

### Step 10: IT-008
- Write failing test: `TestSearchStore_SearchPostsEmitsDBOperationTelemetry` in `appview/internal/api/search_store_test.go`.
- Run command: `cd appview && go test ./internal/api -run TestSearchStore_SearchPostsEmitsDBOperationTelemetry -count=1`.
- Confirmed failure: Compile failed because `SearchStore.WithObserver` did not exist.
- Implement: Added optional `SearchStore` observer wiring, wrapped `SearchPosts` as bounded DB operation `search.posts`, and passed the AppView observer into the route-level search store.
- Run command: `cd appview && go test ./internal/api -run TestSearchStore_SearchPostsEmitsDBOperationTelemetry -count=1`; `cd appview && go test ./internal/api ./internal/routes ./internal/observability -count=1`.
- Refactor: Kept existing `NewSearchStore(pool)` compatibility; moved the original public `SearchPosts` body into a private helper; ran `gofmt`.
- Notes: `search.posts` DB duration uses route pattern `/v1/search/posts`, result labels, and no raw query text.

### Step 11: UT-009 / AT-009
- Write failing test: `TestRecovery_ReturnsV1EnvelopeAndContinuesServing` in `appview/internal/middleware/recovery_test.go`.
- Run command: `cd appview && go test ./internal/middleware -run TestRecovery_ReturnsV1EnvelopeAndContinuesServing -count=1`.
- Confirmed failure: Compile failed because `middleware.Recovery` did not exist.
- Implement: Added `Recovery` middleware that catches panics, logs safe HTTP context, writes the standard v1 error envelope before the response is committed, and wired it into `NewServer`.
- Run command: `cd appview && go test ./internal/middleware -run TestRecovery_ReturnsV1EnvelopeAndContinuesServing -count=1`; `cd appview && go test ./cmd/appview ./internal/middleware -count=1`.
- Refactor: Used the shared `observability.RoutePattern`; ran `gofmt`.
- Notes: The test verifies a second request still succeeds after a panic. Sentry capture is deferred to the Sentry event loop later in this implementation plan.

### Step 12: UT-005
- Write failing test: `TestClassifyPDSErrorUsesBoundedCategoriesAndStages` in `appview/internal/observability/pds_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestClassifyPDSErrorUsesBoundedCategoriesAndStages -count=1`.
- Confirmed failure: Compile failed because PDS stage/category vocabulary and classifier did not exist.
- Implement: Added bounded PDS stages, categories, `NormalizePDSStage`, and `ClassifyPDSError` mapping context timeouts, network errors, existing auth sentinels, record-not-found, and atclient status codes.
- Run command: `cd appview && go test ./internal/observability -run TestClassifyPDSErrorUsesBoundedCategoriesAndStages -count=1`; `cd appview && go test ./internal/observability -count=1`.
- Refactor: Ran `gofmt` on PDS observability files.
- Notes: Classifier returns only approved categories: timeout, network, auth, rate_limited, validation, not_found, forbidden, server, unexpected, or none.

### Step 13: UT-007
- Write failing test: `TestPDSWriteOperationRegistryCoversCurrentWritePaths` in `appview/internal/observability/pds_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestPDSWriteOperationRegistryCoversCurrentWritePaths -count=1`.
- Confirmed failure: Compile failed because PDS operation constants and registry did not exist.
- Implement: Added bounded PDS operation constants for OAuth session resume, profile writes, post create/delete, blob upload, follow/unfollow, like/unlike, and repost/unrepost, plus `KnownPDSOperation`.
- Run command: `cd appview && go test ./internal/observability -run TestPDSWriteOperationRegistryCoversCurrentWritePaths -count=1`; `cd appview && go test ./internal/observability -count=1`.
- Refactor: Ran `gofmt` on PDS observability files.
- Notes: Raw or identifier-bearing operation names are rejected by the registry.

### Step 14: IT-004 / AT-008
- Write failing test: `TestWrapPDSFactoryRecordsWriteTelemetry` in `appview/internal/observability/pds_wrapper_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestWrapPDSFactoryRecordsWriteTelemetry -count=1`.
- Confirmed failure: Compile failed because `Observer.WrapPDSFactory` did not exist.
- Implement: Added `craftsky_appview_pds_write_duration_seconds`, factory/client wrappers, session-resume timing, write method timing, bounded operation mapping, stage/result/category labels, and wrapped the real dependency PDS factory in `newDeps`.
- Run command: `cd appview && go test ./internal/observability -run TestWrapPDSFactoryRecordsWriteTelemetry -count=1`; `cd appview && go test ./internal/observability ./internal/app -count=1`.
- Refactor: Kept `GetRecord` uninstrumented as a read path; ran `gofmt`.
- Notes: Metrics omit DID, OAuth session ID, record payloads, and blob bytes.

### Step 15: UT-006 / AT-007
- Write failing test: `TestSafeNSIDLabelUsesKnownNSIDsOrBoundedFallbacks` in `appview/internal/observability/tap_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestSafeNSIDLabelUsesKnownNSIDsOrBoundedFallbacks -count=1`.
- Confirmed failure: Compile failed because `SafeNSIDLabel` did not exist.
- Implement: Added known NSID allowlist and bounded `unsupported` / `malformed` fallback labels.
- Run command: `cd appview && go test ./internal/observability -run TestSafeNSIDLabelUsesKnownNSIDsOrBoundedFallbacks -count=1`; `cd appview && go test ./internal/observability -count=1`.
- Refactor: Ran `gofmt` on Tap observability files.
- Notes: Raw path-like values such as `did:plc:raw/rkey123` are classified as `malformed`, not emitted as labels.

### Step 16: GAP-001 / IT-003
- Write failing test: `TestWSConsumer_IndexerPanicDoesNotCrashConsumer` in `appview/internal/tap/consumer_test.go`.
- Run command: `cd appview && go test ./internal/tap -run TestWSConsumer_IndexerPanicDoesNotCrashConsumer -count=1`.
- Confirmed failure: Panic escaped `handleWithTimeout` and crashed the test process.
- Implement: Added per-event panic recovery in `handleWithTimeout`, increments retry count, and returns a bounded panic error so the event is not acked on ordinary retryable failure.
- Run command: `cd appview && go test ./internal/tap -run TestWSConsumer_IndexerPanicDoesNotCrashConsumer -count=1`; `cd appview && go test ./internal/tap -count=1`.
- Refactor: Ran `gofmt` on Tap consumer files.
- Notes: Panic recovery lives at the approved per-event indexer boundary. The error text records only the panic value type, not raw record payloads or identifiers.

### Step 17: UT-004 / AT-004
- Write failing test: `TestSanitizeEventContextKeepsOnlyAllowedTechnicalFields` in `appview/internal/observability/sentry_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestSanitizeEventContextKeepsOnlyAllowedTechnicalFields -count=1`.
- Confirmed failure: Compile failed because `EventContext` and `SanitizeEventContext` did not exist.
- Implement: Added backend-neutral event context allowlist for approved technical fields only.
- Run command: `cd appview && go test ./internal/observability -run TestSanitizeEventContextKeepsOnlyAllowedTechnicalFields -count=1`; `cd appview && go test ./internal/observability -count=1`.
- Refactor: Ran `gofmt` on Sentry observability files.
- Notes: Raw DID, handle, token, request body, and raw path fields are dropped before any future external event capture.

### Step 18: IT-005 / AT-003
- Write failing test: `TestStartSpanAddsTraceIDsOnlyWhenTracingEnabled` in `appview/internal/observability/sentry_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestStartSpanAddsTraceIDsOnlyWhenTracingEnabled -count=1`.
- Confirmed failure: Compile failed because tracing config, span context, start-span, and trace ID APIs did not exist.
- Implement: Added optional tracing flag to `observability.Config`, context-carried trace/span IDs, no-op disabled spans, enabled spans with finish result, and wired observer construction to AppView Sentry tracing config and release.
- Run command: `cd appview && go test ./internal/observability -run TestStartSpanAddsTraceIDsOnlyWhenTracingEnabled -count=1`; `cd appview && go test ./internal/app ./internal/observability -count=1`.
- Refactor: Ran `gofmt` on observability and deps files.
- Notes: This is a backend-neutral tracing abstraction for safe correlation. It does not add OpenTelemetry/OTLP or Prometheus exemplars.

### Step 19: IT-007
- Write failing test: `TestFlushUsesConfiguredExternalTelemetryHook` in `appview/internal/observability/sentry_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestFlushUsesConfiguredExternalTelemetryHook -count=1`.
- Confirmed failure: Compile failed because `SentryDSN`, `FlushFunc`, and `Observer.Flush` did not exist.
- Implement: Added a testable external telemetry flush hook, no-op disabled flush behavior, Sentry DSN propagation into observer config, and shutdown cleanup flushing before DB close.
- Run command: `cd appview && go test ./internal/observability -run TestFlushUsesConfiguredExternalTelemetryHook -count=1`; `cd appview && go test ./internal/app ./internal/observability -count=1`.
- Refactor: Ran `gofmt` on observer and deps files.
- Notes: This provides lifecycle plumbing for optional external telemetry; the hot path remains non-blocking.

### Step 20: AT-011 / REG-004
- Write failing test: Existing route, health, middleware, and broader tests were reused as the regression suite.
- Run command: `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`.
- Confirmed failure: No code failure; broader `just test` initially failed only because the sandbox blocked local listeners and localhost Postgres connections.
- Implement: Added README/env documentation for `/metrics` production restriction and Sentry/metrics configuration.
- Run command: `just fmt`; `just test` with normal local permissions.
- Refactor: Promoted `github.com/prometheus/client_golang` to a direct dependency.
- Notes: Existing `/healthz` semantics and representative API/auth route behavior are preserved by passing route/middleware/API suites.

### Step 21: MAN-003
- Manual check: executed during the post-review fix pass.
- Run command: `just dev-d`; `curl -fsS http://localhost:18080/metrics | sed -n '1,80p'`.
- Notes: The scrape returned Prometheus text with `craftsky_appview_build_info`, `craftsky_appview_http_requests_in_flight`, and Tap metrics including `craftsky_appview_tap_connected`.

### Step 22: MAN-004
- Manual check: completed by document update/review.
- Notes: `appview/README.md` and `appview/environments/prod.env.example` state production `/metrics` must be restricted by network policy, reverse proxy, or platform ingress.

### Step 23: MAN-005
- Manual check: completed by document update/review.
- Notes: `appview/README.md` lists metric names/units and states names are internal ops details for this slice.

### Post-Review R1: IR-003 / AT-002 / IT-001 / IT-002
- Write failing test: `TestHTTPMetrics_InFlightGaugeIsNonZeroDuringActiveRequest` in `appview/internal/middleware/metrics_test.go`.
- Run command: `cd appview && go test ./internal/middleware -run TestHTTPMetrics_InFlightGaugeIsNonZeroDuringActiveRequest -count=1`.
- Confirmed failure: `/metrics` had no in-flight gauge sample while a handler was blocked.
- Implement: Split HTTP request metrics into `BeginHTTPRequest`, `EndHTTPRequest`, and completed request observation; middleware now increments before `next.ServeHTTP` and defers decrement.
- Run command: `cd appview && go test ./internal/middleware -run TestHTTPMetrics_InFlightGaugeIsNonZeroDuringActiveRequest -count=1`.
- Notes: Completion metrics still use the registered route pattern; the begin-time in-flight label uses bounded `unmatched` because the outer middleware runs before `ServeMux` has populated `r.Pattern`.

### Post-Review R2: IR-001 / AT-004 / AT-009 / IT-007
- Write failing tests: `TestCaptureErrorUsesSentryTransportWithSanitizedContext` and extended `TestRecovery_ReturnsV1EnvelopeAndContinuesServing`.
- Run command: `cd appview && go test ./internal/observability -run TestCaptureErrorUsesSentryTransportWithSanitizedContext -count=1`; `cd appview && go test ./internal/middleware -run TestRecovery_ReturnsV1EnvelopeAndContinuesServing -count=1`.
- Confirmed failure: Observer had no SDK transport/client or capture API; recovery accepted no observer.
- Implement: Added `github.com/getsentry/sentry-go`, SDK client initialization when `SENTRY_DSN` is set, test transport support, sanitized `CaptureError`/`CapturePanic`, SDK flush, and recovery capture with safe HTTP tags.
- Run command: `cd appview && go test ./internal/observability ./internal/middleware -count=1`.
- Notes: Event messages and exception values are bounded; allowlisted tags carry component, route/operation, result/category/stage, run_id, and trace/span IDs where available.

### Post-Review R3: IR-002 / AT-007 / IT-003
- Write failing tests: `TestTapMetricsExposeIngestionAndIndexerSignals` and `TestWSConsumer_EmitsTapMetricsAndCapturesIndexerErrors`.
- Run command: `cd appview && go test ./internal/observability -run TestTapMetricsExposeIngestionAndIndexerSignals -count=1`; `cd appview && go test ./internal/tap -run TestWSConsumer_EmitsTapMetricsAndCapturesIndexerErrors -count=1`.
- Confirmed failure: Tap metrics APIs and `WSConsumerConfig.Observer` did not exist.
- Implement: Added Tap connected/reconnect/last-event-age/event-received/event-acked/ack-failure/indexer-record/indexer-duration collectors, wired them into the Tap consumer, and captured indexer errors through Sentry with bounded NSID labels.
- Run command: `cd appview && go test ./internal/observability ./internal/tap -count=1`.
- Notes: Touched Tap logs no longer emit raw DID/rkey/URI/record payloads for record received, indexer failed, or poison-pill dropped paths.

### Post-Review R4: IR-004 / AT-003 / AT-008 / IT-004 / IT-005
- Write failing test: `TestWrapPDSFactoryEmitsLogsSpansAndSentryForUnexpectedFailures` in `appview/internal/observability/pds_wrapper_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestWrapPDSFactoryEmitsLogsSpansAndSentryForUnexpectedFailures -count=1`.
- Confirmed failure: Observer config had no logger, and the PDS wrapper emitted metrics only.
- Implement: Added observer logger support; PDS/OAuth wrapper now starts bounded spans when tracing is enabled, emits structured safe logs, records metrics, and captures timeout/network/server/unexpected failures in Sentry.
- Run command: `cd appview && go test ./internal/observability -run TestWrapPDSFactoryEmitsLogsSpansAndSentryForUnexpectedFailures -count=1`.
- Notes: Expected auth/validation/not-found/forbidden/rate-limited categories remain logs/metrics/span-only and are not captured as Sentry error events by default.

### Post-Review R4 Coverage: IR-004 / AT-008
- Write test: `TestPDSWriteHandlersEmitObservedOperations` in `appview/internal/api/observability_pds_test.go`.
- Run command: `cd appview && go test ./internal/api -run TestPDSWriteHandlersEmitObservedOperations -count=1`.
- Implement: Exercised real handler paths for profile writes, post create/delete, blob upload, follow/unfollow, like/unlike, repost/unrepost, and session-resume factory calls through `observer.WrapPDSFactory`.
- Notes: Metrics include all bounded operation names and omit raw DID, rkey, session ID, request text, and media bytes.

### Re-Review R5: IR-001 / AT-004
- Write failing test: `TestHTTPMetrics_CapturesNonPanic5xxResponseInSentry` in `appview/internal/middleware/metrics_test.go`.
- Run command: `cd appview && go test ./internal/middleware -run TestHTTPMetrics_CapturesNonPanic5xxResponseInSentry -count=1`.
- Confirmed failure: Handler-written non-panic 500 response captured zero Sentry events.
- Implement: `HTTPMetrics` now installs an observability capture marker in the request context, records completed request metrics, and captures uncaptured HTTP 5xx responses with sanitized Sentry context. Recovery/PDS captures mark the same request context so panic and selected PDS captures are not double-reported by the generic HTTP 5xx path.
- Run command: `cd appview && go test ./internal/middleware -run 'TestHTTPMetrics_CapturesNonPanic5xxResponseInSentry|TestRecovery_ReturnsV1EnvelopeAndContinuesServing' -count=1`.
- Notes: The new test also serves a 404 response and asserts it is not captured. Captured 5xx events include component, route pattern, method, status/status class, category, duration, and run_id; raw DID, query string, body text, and 4xx error names are omitted.

### Re-Review R6: IR-002 / AT-003
- Action: Clarified the current tracing scope in `appview/README.md`, `appview/environments/dev.env`, and `appview/environments/prod.env.example`.
- Notes: This deferral note was later superseded by Re-Review R9, which wires bounded observer spans to real `sentry-go` transactions/spans.

### Re-Review R7: IR-001 / AC-015 / AT-008
- Write failing test: `TestHTTPMetrics_DoesNotCaptureExpectedPDSFailureReturnedAs502` in `appview/internal/middleware/metrics_test.go`.
- Run command: `cd appview && go test ./internal/middleware -run TestHTTPMetrics_DoesNotCaptureExpectedPDSFailureReturnedAs502 -count=1`.
- Confirmed failure: A validation-class PDS error observed through `WrapPDSFactory` was not captured by the PDS observer, but the later handler-written `502 pds_write_failed` response was captured by the generic HTTP 5xx Sentry fallback.
- Implement: Expected PDS categories are now marked as handled on the request capture marker without sending a Sentry error event. The regression covers validation, auth, forbidden, not-found, and rate-limited PDS classes returning a fallback AppView 502. High-severity PDS timeout/network/server/unexpected categories still capture through the PDS path, and generic uncaptured AppView 5xx responses still capture through `HTTPMetrics`.
- Run command: `cd appview && go test ./internal/middleware -run TestHTTPMetrics_DoesNotCaptureExpectedPDSFailureReturnedAs502 -count=1`; `cd appview && go test ./internal/middleware ./internal/observability -count=1`.
- Notes: Metrics/logs still classify expected PDS categories. The handling marker only prevents the generic HTTP fallback from turning expected user/actionable PDS 4xx classes into Sentry error events.

### Re-Review R8: IR-001 / AC-005 / AC-010 / AT-005 / AT-008
- Write failing test: `TestExpirePDSSessionLogsBoundedContextWithoutRawIdentityOrSession` in `appview/internal/app/deps_test.go`.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/app -run TestExpirePDSSessionLogsBoundedContextWithoutRawIdentityOrSession -count=1 -v`.
- Confirmed failure: The normal expiry warning and the cleanup-error logs included raw `did:plc:alice` and `session-secret` fields.
- Implement: `expirePDSSession` now logs bounded PDS/OAuth context only: component, operation, failure stage, result, error category, and run_id when present. Raw DID and OAuth session ID are still used for DB cleanup calls but are no longer emitted in logs.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/app -run TestExpirePDSSessionLogsBoundedContextWithoutRawIdentityOrSession -count=1`.
- Notes: The regression covers both successful cleanup and forced cleanup-error branches.

### Re-Review R9: IR-002 / AC-003 / AC-010 / AT-003
- Write failing test: `TestStartSpanExportsSentryTransactionAndChildSpan` in `appview/internal/observability/sentry_test.go`.
- Run command: `cd appview && go test ./internal/observability -run TestStartSpanExportsSentryTransactionAndChildSpan -count=1`.
- Confirmed failure: `StartSpan` produced local trace/span IDs but captured zero Sentry transaction events.
- Implement: `Observer` now owns a Sentry hub when Sentry is configured. `StartSpan` creates real `sentry-go` spans/transactions with bounded component/operation/result data, preserves trace/span IDs in context, and finishes the SDK span when the local wrapper finishes. Tracing without a configured Sentry DSN still uses local trace/span IDs only.
- Run command: `cd appview && go test ./internal/observability -run TestStartSpanExportsSentryTransactionAndChildSpan -count=1`; `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/app ./internal/middleware ./internal/observability ./internal/auth -count=1`.
- Notes: `TestWrapPDSFactoryEmitsLogsSpansAndSentryForUnexpectedFailures` now filters for exactly one Sentry error event because configured tracing also emits transaction events. README and env examples now describe real Sentry SDK transaction/span export rather than a deferral.

### Re-Review R10: IR-001 / AC-005 / AC-006 / AC-010 / AT-005 / AT-006 / AT-008
- Write failing tests: `TestPDSWriteHandlerLogsUseBoundedContextWithoutRawIdentitySessionOrContent` in `appview/internal/api/observability_pds_test.go` and `TestDispatcher_LogOmitsRawRecordIdentity` in `appview/internal/index/dispatcher_test.go`.
- Run command: `cd appview && go test ./internal/api -run TestPDSWriteHandlerLogsUseBoundedContextWithoutRawIdentitySessionOrContent -count=1`; `cd appview && go test ./internal/index -run TestDispatcher_LogOmitsRawRecordIdentity -count=1`.
- Confirmed failure: PDS write-handler debug/warn logs included raw DID, OAuth session ID, request/record structs, record URI/rkey/CID, and user/content bytes. The dispatcher debug log included the full AT URI.
- Implement: Added bounded PDS log helper attributes and moved post create/delete, like/unlike, repost/unrepost, blob upload, and profile put PDS/write logs onto component/operation/stage/result/category/run_id fields only. Removed raw error strings from write-path store/identity failure logs so wrapped errors cannot leak identifiers. Removed the dispatcher's raw URI log field while keeping collection/action/indexer/fallback routing context.
- Run command: `cd appview && go test ./internal/api -run TestPDSWriteHandlerLogsUseBoundedContextWithoutRawIdentitySessionOrContent -count=1`; `cd appview && go test ./internal/index -run TestDispatcher_LogOmitsRawRecordIdentity -count=1`.
- Notes: Tests cover representative profile put, post create/delete, like create, blob upload, and index dispatch logs with `did:plc:alice`, `session-secret`, `post1`, AT URI/CID values, request/record text, and media bytes asserted absent from captured JSON logs.

### Re-Review R11: IR-001 / AC-005 / AC-006 / AT-006
- Write failing test: `TestReadHandlerLogsUseBoundedContextWithoutRawIdentityOrContent` in `appview/internal/api/observability_pds_test.go`.
- Run command: `cd appview && go test ./internal/api -run TestReadHandlerLogsUseBoundedContextWithoutRawIdentityOrContent -count=1`.
- Confirmed failure: Read-route debug and prod-level logs included raw DID, viewer DID, handle input, rkey, AT URI, CID, cursor, response objects, row objects, and wrapped error strings containing user/content values.
- Implement: Added bounded API log helper attributes and moved get-post, comment/reply lists, author post/project/comment lists, profile get/me/list, timeline, notifications, facets, reports, search, whoami, health, and dev moderation logs onto component/operation/result/category/run_id fields. Removed raw path/header/client-address logging from middleware, raw token/session/handle/request URI/loopback URI logging from auth, raw Tap/indexer error objects from background logs, raw Tap URL logging, and raw Tap `last_error` retention in health state.
- Audit command: `rg -n 'slog\\.(String|Any)\\(\"(did|viewer_did|profile_did|subject_did|target_did|handle|input|rkey|uri|.*_uri|cid|cursor|next_cursor|focus|parent|row|response|request|record|err|.*err|blob|device|session|token|query|path|headers|remote_addr|url|state)\"|slog\\.String\\(\"err\", .*\\.Error\\(\\)\\)|slog\\.Any' appview/internal appview/cmd` returned no matches.
- Run command: `cd appview && go test ./internal/api -run 'TestReadHandlerLogsUseBoundedContextWithoutRawIdentityOrContent|TestPDSWriteHandlerLogsUseBoundedContextWithoutRawIdentitySessionOrContent' -count=1`; `cd appview && go test ./internal/middleware -count=1`; `cd appview && go test ./internal/auth -count=1`.
- Notes: The focused regression covers `GET /v1/posts/{did}/{rkey}`, reply/comment lists, author list, profile get, prod-level post/profile/mutual-follower failures, and forbidden raw values including DID, handle, rkey, AT URI/CID, cursor, row/response objects, and user content.

### Re-Review R12: IR-001 Suggestion / FR-007 / FR-008 / AC-003 / AT-003
- Write failing tests: `TestHTTPMetrics_ExportsSentryTransactionWithRoutePattern` in `appview/internal/middleware/metrics_test.go` and `TestWSConsumer_ExportsSentryConsumeAndIndexerSpans` in `appview/internal/tap/consumer_test.go`.
- Run command: `cd appview && go test ./internal/middleware -run TestHTTPMetrics_ExportsSentryTransactionWithRoutePattern -count=1`; `cd appview && go test ./internal/tap -run TestWSConsumer_ExportsSentryConsumeAndIndexerSpans -count=1`.
- Confirmed failure: HTTP handlers received no Sentry trace/span IDs and no HTTP transaction was exported; Tap processing exported zero Sentry transaction events for the consume loop or indexer handling.
- Implement: `HTTPMetrics` now starts a bounded `http.server` Sentry transaction before dispatch, exposes trace/span IDs to handlers through request context, then names and finishes the transaction with the registered route pattern, status class, result, duration, method, and run_id after the mux returns. `WSConsumer.runOnce` now starts a bounded `tap.consume` transaction, and `handleWithTimeout` starts a `tap.indexer.handle` child span with safe NSID/result/component context.
- Run command: `cd appview && go test ./internal/middleware -run TestHTTPMetrics_ExportsSentryTransactionWithRoutePattern -count=1`; `cd appview && go test ./internal/tap -run TestWSConsumer_ExportsSentryConsumeAndIndexerSpans -count=1`; `cd appview && go test ./internal/observability ./internal/middleware ./internal/tap -count=1`.
- Notes: The new tests assert transaction/span export through `sentry-go` test transport and assert raw DID, rkey, cursor, CID, and record text are absent from transaction names and child span data. This implements the prior Should-level tracing suggestion without adding OpenTelemetry/OTLP export.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [ ] Review completed or explicitly skipped

## Final Verification
- `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1` passed.
- `just fmt` passed.
- `just test` passed: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...`.
- `just dev-d` passed and `curl -fsS http://localhost:18080/metrics | sed -n '1,80p'` returned Prometheus text from the Docker AppView.
- Re-review fix verification passed: `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/tap ./internal/index ./internal/observability -count=1`; `just fmt && just test`.
- Latest re-review fix verification passed: `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/tap ./internal/index ./internal/observability -count=1`; `just fmt`; `just test`; `git diff --check`.
- OAuth expiry log and Sentry span review fix verification passed: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/app ./internal/middleware ./internal/observability ./internal/auth -count=1`; `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/tap ./internal/index ./internal/observability -count=1`; `just fmt`; `just test`.
- PDS/write handler and dispatcher log privacy re-review verification passed: `cd appview && go test ./internal/api -run TestPDSWriteHandlerLogsUseBoundedContextWithoutRawIdentitySessionOrContent -count=1`; `cd appview && go test ./internal/index -run TestDispatcher_LogOmitsRawRecordIdentity -count=1`; `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/app ./internal/middleware ./internal/observability ./internal/auth ./internal/index -count=1`; `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`; `just fmt`; `just test`; `git diff --check`.
- Read-route and broad logging privacy re-review verification passed: `cd appview && go test ./internal/api -run 'TestReadHandlerLogsUseBoundedContextWithoutRawIdentityOrContent|TestPDSWriteHandlerLogsUseBoundedContextWithoutRawIdentitySessionOrContent' -count=1`; `cd appview && go test ./internal/middleware -count=1`; `cd appview && go test ./internal/auth -count=1`; `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/app ./internal/middleware ./internal/observability ./internal/auth ./internal/index -count=1`; `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`; `just fmt`; `just test`; `git diff --check`.
- HTTP/Tap tracing suggestion verification passed: `cd appview && go test ./internal/middleware -run TestHTTPMetrics_ExportsSentryTransactionWithRoutePattern -count=1`; `cd appview && go test ./internal/tap -run TestWSConsumer_ExportsSentryConsumeAndIndexerSpans -count=1`; `cd appview && go test ./internal/observability ./internal/middleware ./internal/tap -count=1`; `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`; `just fmt`; `just test`; `git diff --check`.
- The first sandboxed `just test` attempt failed because local listener and Postgres connections were blocked by sandbox networking (`operation not permitted`); rerunning the same command outside the sandbox passed.
- `git diff` reviewed for requirement traceability. Changes are limited to AppView observability code/tests/docs/config examples and the implementation-plan artifact.

## Documented Gaps
- Hosted Sentry project smoke testing (`MAN-001`) was not executed because no real project DSN was provided in this workflow. SDK initialization, capture, transaction/span export, and flush are covered with `sentry-go` test transport.
