# Coding Plan: AppView Sentry Observability Consolidation

## 1. Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`

## 2. Implementation Strategy

Refactor the existing `appview/internal/observability` package in place so it remains the AppView observability boundary, but split the current Prometheus-backed `Observer` into narrow local interfaces for metrics, tracing, error capture, and Sentry-bound logs. Keep existing `slog` stdout logging as the local operator surface, remove `/metrics`, remove Prometheus collectors/dependencies, and add Sentry implementations plus no-op and in-memory validation implementations behind AppView-domain methods.

This fits the current codebase because AppView already wires a single `*observability.Observer` through `app.Deps`, HTTP middleware, PDS wrappers, Tap consumer, and selected stores. The TDD work should evolve that boundary rather than creating a second observability framework. Production direct `sentry-go` imports should be allowed only under `appview/internal/observability`; tests may use `sentry.MockTransport` only in explicitly named integration suites while business, storage, routes, middleware, and Tap code consume local interfaces.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| App config | `app.Config` has DSN, release, tracing flag, trace sample rate | Add explicit Sentry logs, metrics, and Tap trace volume controls; preserve DSN-only errors/panics | FR-001, FR-015, NFR-002, NFR-006, RULE-001 | AT-001, UT-001, IT-001, UT-010 |
| Dependency wiring | `newDeps` creates stdout `slog` logger and `observability.New` | Build observer from config; compose optional Sentry-bound log sink inside observability; flush on cleanup | FR-001, FR-003, FR-005, FR-006 | AT-001, AT-002, IT-001, UT-002, UT-009 |
| Observability package | One `Observer` owns Prometheus collectors plus Sentry client/hub | Introduce narrow metric/tracer/error/log interfaces, validator/sanitizer helpers, no-op/in-memory/Sentry implementations | BR-002, FR-004, FR-005, FR-006, NFR-001 | UT-002, UT-003, UT-006, UT-007, UT-008 |
| HTTP middleware | `HTTPMetrics` starts request span and records Prometheus metrics | Rename/reshape around HTTP observability; emit AppView metric methods, safe events, route-pattern transactions | FR-002, FR-007, FR-012, FR-013, RULE-005, RULE-006 | AT-003, UT-004, UT-007, IT-002 |
| Routes | Public `/metrics` route registered outside `/v1/*` | Remove `/metrics`; update tests to prove no replacement metrics endpoint | FR-017 | AT-002, IT-007, REG-001 |
| PDS/OAuth wrapper | `WrapPDSFactory` records PDS metrics, logs, spans, and some events | Keep wrapper, add session/request/response/blob spans and safe classifier fields through local interfaces | FR-010, FR-012, FR-014, RULE-002 | IT-005, UT-005, REG-005 |
| DB/storage | `ObserveDB` records Prometheus duration for selected operations | Add manual named DB spans and result classes; avoid SQL/query text, exact counts, identifiers | FR-008, FR-009, RULE-002 | UT-008, IT-004 |
| Tap/indexer | Tap consumer records metrics and consume/indexer spans | Add receive/decode/classify/ack/reconnect spans, sampling controls, safe panic/error capture | FR-011, FR-013, NFR-006 | UT-010, IT-006, REG-003 |
| Docs/dependencies | README documents Prometheus; `go.mod` includes Prometheus | Rewrite AppView observability docs; remove Prometheus modules after callers are replaced | FR-017, FR-018, NFR-004 | UT-011, MAN-001, MAN-002, MAN-003 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/app/config.go` | Change | Parse `SENTRY_LOGS_ENABLED`, `SENTRY_METRICS_ENABLED`, `SENTRY_TAP_TRACING_ENABLED` or equivalent approved names, metric/Tap volume controls, and validation defaults | FR-001, FR-015, NFR-002, NFR-006 | UT-001 |
| `appview/internal/app/config_test.go` | Change | Add config matrix for DSN-only and explicit pillar gates | FR-001, FR-015, NFR-002 | UT-001 |
| `appview/internal/app/deps.go` | Change | Pass full observability config; keep Sentry SDK import out of this file unless a documented log-handler bridge makes it unavoidable | FR-001, FR-003, FR-006 | IT-001, UT-002 |
| `appview/internal/observability/config.go` | Create | Local observability config type independent of Prometheus and with Sentry transport/test injection isolated here | FR-001, FR-005, FR-006 | UT-001, IT-001 |
| `appview/internal/observability/observer.go` | Create / Change | Group narrow interfaces and expose existing `Observer` facade for low-churn call sites | BR-002, FR-005, FR-006 | UT-002, UT-006 |
| `appview/internal/observability/metrics.go` | Change | Replace Prometheus collectors with AppView-domain metric methods and Sentry/no-op/in-memory implementations | FR-004, FR-005, FR-017 | UT-006, IT-003, UT-011 |
| `appview/internal/observability/sentry.go` | Change | Sentry client/hub lifecycle, `BeforeSend*` scrubbers, events, spans, metrics bridge, flush | FR-001, FR-002, FR-007, FR-012 | UT-004, UT-007, IT-001 |
| `appview/internal/observability/logs.go` | Create | Sentry-bound structured log sink/filter; prefer SDK slog integration if present, otherwise local handler using Sentry log APIs/client hooks | FR-003, FR-018, NFR-005 | UT-009 |
| `appview/internal/observability/validation.go` | Create | Shared runtime normalization and validation-test failure helpers for telemetry names/attrs | NFR-001, RULE-002, RULE-004, RULE-006 | UT-003, UT-006, UT-007, UT-008 |
| `appview/internal/observability/error_classifier.go` | Create | Bounded sentinel/enum classifier for Sentry events and Sentry-bound logs | FR-012, FR-014, RULE-006 | UT-005 |
| `appview/internal/observability/db.go` | Change | Convert `ObserveDB` into named operation span plus metric emission with bounded result classes | FR-008, FR-009 | UT-008, IT-004 |
| `appview/internal/observability/pds.go` | Change | Add safe span attributes, event classifier, and metric methods for PDS/OAuth/write/blob boundaries | FR-010, FR-012, FR-014 | IT-005 |
| `appview/internal/observability/tap.go` | Change | Add metric methods, safe labels, Tap sampling/trace helpers | FR-011, NFR-006 | UT-010, IT-006 |
| `appview/internal/observability/import_boundary_test.go` | Create | Guard direct production `sentry-go` imports to `appview/internal/observability`; allow only named test files to use `sentry.MockTransport` | FR-006 | UT-002 |
| `appview/internal/observability/prometheus_removal_test.go` | Create | Guard absence of Prometheus dependencies, collectors, and handler names after replacement | FR-017 | UT-011 |
| `appview/internal/middleware/metrics.go` | Change | Keep existing middleware entrypoint or rename later; call observer HTTP methods, trace interface, and safe error classifier | FR-002, FR-007, FR-012 | AT-003, IT-002 |
| `appview/internal/middleware/logging.go` | Change | Preserve stdout logs; optionally fan out safe Sentry-bound attrs through observability without changing local log content | FR-003, RULE-007 | UT-009, REG-004 |
| `appview/internal/middleware/recovery.go` | Change | Ensure panic capture includes recovered type only and no recovered value string | FR-013, RULE-003 | UT-004, IT-002 |
| `appview/internal/routes/routes.go` | Change | Remove `GET /metrics`; keep health endpoints and `/v1/*` stack unchanged | FR-017, FR-016 | IT-007, REG-001 |
| `appview/cmd/appview/server.go` | Change | Keep middleware order unless tests expose a trace/log reason to adjust; update comments from metrics-specific to observability | FR-016, FR-017 | REG-002 |
| `appview/internal/api/search_store.go`, `profile_store.go`, `timeline_store.go`, `post_store.go`, selected auth stores | Change | Add named `ObserveDB` spans around approved storage operations only | FR-008, FR-009 | UT-008, IT-004 |
| `appview/internal/api/blob.go`, post/profile/follow/interaction write handlers, auth handlers | Change | Add business/work spans where the boundary is cleaner in handler code; use local tracer only | FR-008, FR-010 | IT-002, IT-005 |
| `appview/internal/tap/consumer.go` | Change | Add receive/decode/classify/ack/reconnect spans and keep behavior unchanged | FR-011, FR-016 | UT-010, IT-006, REG-003 |
| `appview/README.md` | Change | Remove Prometheus runtime docs; document Sentry flags, DSN-only behavior, local stdout logs, and logs-vs-breadcrumbs distinction | FR-017, FR-018 | MAN-001, MAN-002 |
| `appview/go.mod`, `appview/go.sum` | Change | Remove Prometheus deps; upgrade `sentry-go` only if implementation needs SDK log/metric APIs missing from current module | FR-004, FR-017, NFR-005 | UT-011, MAN-003 |

## 5. Services, Interfaces, And Data Flow

Keep call sites AppView-specific and avoid a generic telemetry framework. The facade can remain `*observability.Observer` for low churn, but internally it should delegate to narrow interfaces.

```text
type Observer struct {
  Metrics MetricRecorder
  Tracer Tracer
  Errors ErrorReporter
  Logs LogSink
  validator TelemetryValidator
}

type MetricRecorder interface {
  HTTPRequestStarted(ctx, method, routePattern)
  HTTPRequestFinished(ctx, method, routePattern, statusClass/status, duration, responseBytes)
  DBOperation(ctx, operation, routePattern, resultClass, duration)
  PDSOperation(ctx, operation, stage, result, category, duration)
  TapConnected(ctx, connected)
  TapEventReceived(ctx, eventType)
  TapEventAcknowledged(ctx, result)
  TapIndexerRecord(ctx, nsid, result, reason, duration)
}

type Tracer interface {
  StartSpan(ctx, SpanContext) (context.Context, Span)
}

type ErrorReporter interface {
  CaptureError(ctx, ClassifiedError, EventContext)
  CapturePanic(ctx, PanicContext)
}

type LogSink interface {
  Emit(ctx, level, message, EventContext)
}
```

Sentry data flow:

```text
app config -> observability.New
  -> sentry.NewClient(ClientOptions{
       Dsn, Environment, Release,
       SendDefaultPII: false,
       EnableTracing: cfg.TracingEnabled,
       TracesSampleRate / TracesSampler,
       DisableLogs: !cfg.LogsEnabled,
       DisableMetrics: !cfg.MetricsEnabled,
       BeforeSend / BeforeSendTransaction / BeforeSendLog / BeforeSendMetric,
     })
  -> hub stored only in observability
  -> local wrappers use sentry.StartSpan, sentry.NewMeter(ctx), client capture APIs
```

The current `sentry-go` v0.47 module exposes `sentry.NewMeter(ctx)` with `Count`, `Gauge`, `Distribution`, `WithUnit`, and `WithAttributes`. The plan should map AppView-domain methods to those APIs only inside the Sentry metric implementation. If the documented `github.com/getsentry/sentry-go/slog` package is unavailable in the selected module version, implement the Sentry log sink locally or upgrade the SDK as a focused dependency step with tests proving the same filtering behavior.

Approved direct Sentry import boundary for `UT-002`:

- Production: `appview/internal/observability/**/*.go` only.
- Startup/log wiring exception: none by default. Add `appview/internal/app/deps.go` to the allowlist only if a later implementation test proves the SDK log handler cannot be hidden inside observability.
- Tests: `appview/internal/observability/*_test.go`, plus selected integration tests in `appview/internal/middleware`, `appview/internal/tap`, and `appview/internal/api` may import `sentry-go` for `MockTransport` until a local test recorder replaces them.

## 6. State, Providers, Controllers, Or DI

AppView uses explicit Go dependency wiring rather than providers or controllers.

```text
app.LoadConfig
  -> app.newDeps
    -> logger := stdout JSON slog handler
    -> observer := observability.New(observability.Config{...})
    -> optional safe Sentry log sink is composed inside observer
    -> deps.Observability passed to server, routes, stores, PDS factory, Tap consumer

cmd/appview.NewServer
  Logging(stdout logger)
  HTTPMetrics/HTTPObservability(observer)
  Recovery(logger, observer)
  CORS
  mux/routes
```

Keep the existing dependency shape: `Deps.Observability` remains a pointer to the concrete `observability.Observer` struct (`*observability.Observer`). The `Observer` struct can internally hold narrower interfaces for metrics, tracing, errors, and logs, but those interfaces should not become four separate `Deps` fields in this slice because that would increase churn across route constructors and tests without improving the approved acceptance criteria.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

No Flutter UI, widgets, or client routes are in scope.

HTTP surfaces:

- Remove `GET /metrics` from `appview/internal/routes/routes.go`.
- Keep `GET /health`, `GET /healthz`, `/oauth/*`, and all `/v1/*` routes unchanged.
- Update route/server tests to expect `/metrics` is absent and does not return Prometheus text or a new metrics endpoint.
- Do not add any replacement local metrics route.

Operational docs:

- Update `appview/README.md` observability section for Sentry errors/logs/traces/metrics flags, DSN-only behavior, stdout logs, and manual hosted Sentry metric verification.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| No `SENTRY_DSN` | Build observer with no-op external exporters; local stdout logs remain | FR-005, FR-015 | AT-002, UT-001, REG-006 |
| `SENTRY_DSN` only | Enable errors/recovered panics; keep logs, tracing, metrics disabled | FR-001, FR-015, NFR-002 | AT-001, UT-001, IT-001 |
| Logs explicitly disabled | Keep stdout `slog`; do not send Sentry logs | FR-003 | AT-002, UT-009, REG-004 |
| Metrics explicitly disabled | Metric recorder is no-op or in-memory in tests; no endpoint exists | FR-005, FR-017 | UT-006, IT-007 |
| Invalid sample rate or volume control | Config load fails with named env var | FR-001, NFR-002, NFR-006 | UT-001, UT-010 |
| Runtime invalid telemetry attrs | Normalize to `unknown`, `other`, `unmatched`, or safe enum bucket | NFR-001, RULE-002, RULE-004 | UT-003, UT-007, UT-008 |
| Validation test invalid attrs | In-memory/validator implementation records a test failure or explicit validation error | NFR-001 | UT-003, UT-006, UT-007 |
| Panic with sensitive recovered value | Capture recovered type and safe context only; omit value string | FR-013, RULE-003 | UT-004, IT-002, IT-006 |
| Non-panic wrapped upstream error | Use bounded classifier category/code/stage; omit raw wrapped text from Sentry | FR-012, FR-014, RULE-006 | UT-005, IT-005 |
| Unknown HTTP path containing identifiers | Use bounded fallback transaction/route name such as `unmatched` | FR-007, RULE-005 | UT-007, IT-002 |
| DB returns many rows | Record result class such as `many`, not exact count | FR-009 | UT-008, IT-004 |
| Tap high-volume success path | Apply Tap-specific tracing control; still capture errors/panics | FR-011, NFR-006 | UT-010, IT-006 |
| Sentry init error | Log safe startup context and continue unless config itself is invalid | FR-001, FR-015 | IT-001 |
| Hosted Sentry metrics behavior differs | Keep local interface stable; document/manual-check Sentry metric names | NFR-004 | GAP-001, MAN-003 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `appview/internal/app/config_test.go`, `appview/internal/observability/sentry_test.go` | Env matrix for no DSN, DSN only, logs/tracing/metrics flags, invalid rates, Tap controls | New flags missing; DSN-only behavior not represented |
| 2 | UT-002 | `appview/internal/observability/import_boundary_test.go` | Repository file scan over production `.go` files and named test allowlist | Prometheus/Sentry boundaries not encoded; possible broad imports |
| 3 | UT-006 | `appview/internal/observability/metrics_test.go` | No-op, in-memory, and Sentry metric recorders with bounded attrs | Current metrics are Prometheus collectors and `/metrics` output |
| 4 | UT-003 | `appview/internal/observability/redaction_test.go`, `validation_test.go` | Forbidden values and invalid enum/tag inputs | Current sanitizer allowlist permits `run_id` and lacks strict validation mode |
| 5 | UT-005 | `appview/internal/observability/error_classifier_test.go` | Sentinel errors and wrapped unsafe details | No shared classifier for all AppView categories |
| 6 | UT-004 | `appview/internal/observability/sentry_test.go`, `middleware/recovery_test.go`, `tap/consumer_test.go` | Panic values with tokens, DIDs, handles, payload snippets | Panic event assertions need recovered type and value exclusion |
| 7 | UT-007 | `appview/internal/observability/sentry_test.go`, `route_test.go` | Route patterns, unknown paths, safe/unsafe attrs | Tracing normalization incomplete |
| 8 | IT-002 | `appview/internal/middleware/metrics_test.go`, selected `api/*_test.go` | `httptest`, Sentry mock/in-memory observer, raw identifiers in request | HTTP telemetry still Prometheus-shaped and lacks child spans |
| 9 | IT-003 | `appview/internal/observability/metrics_test.go`, `api/observability_pds_test.go`, `tap/consumer_test.go` | Representative HTTP/DB/PDS/Tap metric calls | Sentry metric implementation not wired |
| 10 | UT-008 | `appview/internal/observability/db_test.go`, selected store tests | Named store operations and attempted unsafe attrs | `ObserveDB` only emits Prometheus duration and no spans |
| 11 | IT-004 | `search_store_test.go`, `profile_store_test.go`, `timeline_store_test.go`, `auth/store_test.go` | Store calls using test DB/fakes | DB spans absent in selected storage paths |
| 12 | IT-005 | `observability/pds_wrapper_test.go`, `api/blob_test.go`, `auth/*_test.go` | Fake PDS clients and classified failures | PDS/OAuth spans/events incomplete |
| 13 | UT-009 | `middleware/logging_test.go`, `observability/logs_test.go` | Local stdout capture plus Sentry-bound log sink/fake | Sentry logs not implemented; filtering not shared |
| 14 | UT-010 | `tap/consumer_test.go`, `observability/tap_test.go` | Tap success/error/panic/reconnect fixtures and sampling config | Tap trace volume control missing |
| 15 | IT-006 | `tap/consumer_test.go`, `index/*_test.go` | Fake Tap events, dispatcher/indexer fakes, Sentry/in-memory observer | Receive/decode/ack/reconnect spans absent |
| 16 | IT-007 | `routes/routes_test.go`, `cmd/appview/server_test.go` | AppView mux/server with requests to `/metrics` and `/v1/*` | `/metrics` still returns Prometheus text |
| 17 | UT-011 | `observability/prometheus_removal_test.go` | Scan `go.mod`, `go.sum`, routes, observability files, docs snippets | Prometheus dependencies and handler references remain |
| 18 | AT-004 / REG suite | Existing AppView suites | Focused command from handoff | Instrumentation regressions, route/API/Tap/PDS behavior changes |
| 19 | MAN-001, MAN-002, MAN-003 | Manual docs/Sentry check | README review and one non-prod Sentry metrics smoke test | Docs or hosted metric names may need adjustment |

## 10. Sequencing And Guardrails

- First TDD step: `UT-001` in `appview/internal/app/config_test.go`, starting with DSN-only behavior and explicit flags for logs/tracing/metrics.
- Dependencies between work items: config gates before observer construction; observer interfaces before replacing Prometheus call sites; validation/classifier helpers before adding new events/logs/spans; Sentry/no-op/in-memory metrics before removing `/metrics`; route/dependency removal after callers stop using Prometheus.
- Guardrails: keep production direct `sentry-go` imports inside `appview/internal/observability`; keep local stdout logs; do not export raw identifiers/content/tokens/error strings; use route patterns, sentinel enums, status classes, result classes, and operation allowlists; keep AppView API envelopes unchanged; keep Tap ack/retry/drop behavior unchanged.
- Out of scope: Flutter observability, dashboards/alerts/issue ownership, OpenTelemetry/OTLP, database migrations, lexicon changes, product analytics, replacing `pgxpool` with `database/sql`, `sentrysql`, a replacement local metrics endpoint, exhaustive per-query spans.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | `sentry-go` Application Metrics hosted behavior cannot be fully proven locally | Local tests may pass while hosted Sentry names/searchability differ | Keep metrics interface narrow; run MAN-003 in non-production after implementation |
| CPQ-002 | Non-blocking | Documented Sentry slog integration may not exist in the currently cached v0.47 module | Implementation may need local handler or SDK upgrade | During TDD, first inspect selected module; prefer local `LogSink` wrapper if package is unavailable |
| CPQ-003 | Non-blocking | Runtime normalization can hide bad telemetry by grouping under fallback values | Mistakes may be visible only as `unknown`/`other` in production | Use validation-focused in-memory tests that fail loudly on invalid attrs |
| CPQ-004 | Non-blocking | More spans in Tap/indexer paths can increase noise/cost | High-volume ingestion could produce excessive Sentry trace data | Add Tap-specific trace controls before broad success-path instrumentation |
| CPQ-005 | Non-blocking | Removing `/metrics` may break local habits or docs | Developers lose quick curl-based metrics inspection | Update docs and route tests; rely on stdout logs, tests, and explicitly enabled Sentry metrics |

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-001` for config defaults and explicit Sentry pillar gates.
- Focused command: `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`
- Notes: Keep the implementation staged so Prometheus removal happens after Sentry/no-op/in-memory metrics satisfy existing metric intent. Hosted Sentry metric/log UI behavior remains manual validation, not a local TDD blocker.
