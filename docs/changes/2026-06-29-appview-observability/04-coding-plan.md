# Coding Plan: AppView Observability

## 1. Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Reference architecture: `atproto-craft-social-app-reference.md`
- API architecture: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`

## 2. Implementation Strategy

Add an AppView-only observability boundary under `appview/internal/observability`, then wire it through the existing explicit dependency graph in `appview/internal/app/deps.go`. Keep `slog`, stdlib `net/http`, route registration in `routes.AddRoutes`, and the current AppView/PDS split intact.

The implementation should land in TDD slices:

1. Prometheus registry and unauthenticated `GET /metrics` outside `/v1/*`.
2. Observability configuration, safe logger fields, response-body logging guard, and redaction helpers.
3. HTTP request correlation, route-pattern telemetry, metrics, and panic recovery.
4. Search DB operation timing, starting with `GET /v1/search/posts` and `SearchStore.SearchPosts` as bounded operation `search.posts`.
5. PDS/OAuth write-proxy instrumentation through `auth.PDSClientFactory` and `auth.PDSClient` wrappers.
6. Tap/indexer metrics and background recovery at the Tap consumer/indexer handling boundary.
7. Optional Sentry lifecycle, error capture, tracing/span hooks, and shutdown flush.
8. Documentation/manual-check updates for `/metrics` production restriction, metric names, and local inspection.

This fits the existing codebase because AppView already uses explicit constructors, interfaces at PDS and Tap boundaries, stdlib mux route patterns, and focused package-level tests. The plan avoids database migrations, Flutter changes, lexicon changes, API response contract changes, OpenTelemetry/OTLP export, Prometheus exemplars, and Sentry Application Metrics.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| App config | `app.Config` loaded by `LoadConfig`; env-specific validation in `config.go` | Add Sentry and unsafe body logging config with disabled defaults and prod guardrails | FR-012, FR-013, FR-017, RULE-003, RULE-005, RULE-010 | AT-005, AT-010, UT-008, REG-005, REG-006 |
| Dependency wiring | `app.Deps` owns logger, DB, auth, Tap, stores, PDS factory | Add `Observability` dependency, Prometheus registry, Sentry/noop client, observed PDS factory, and cleanup flush | FR-003, FR-007, FR-009, FR-012, FR-013 | AT-002, AT-003, AT-004, AT-010, IT-001, IT-007 |
| Routes | `routes.AddRoutes` registers ops routes and `/v1/*`; route policy table is canonical for v1 | Register `GET /metrics` as public ops route; expose route-pattern helper/ops patterns for telemetry tests | FR-003, FR-015, RULE-001, RULE-008, RULE-011 | AT-002, AT-006, IT-001, IT-002, REG-002 |
| HTTP middleware | `middleware.Logging` assigns `run_id`; CORS wraps mux; v1 auth/device/rate/body applied per route | Tighten logging fields, remove default payload logging, add recovery and HTTP metrics middleware | FR-001, FR-002, FR-004, FR-010, FR-018 | AT-001, AT-005, AT-006, AT-009, UT-001, UT-009, IT-006 |
| Error envelopes | `envelope.WriteError` emits `{error,message,requestId}` for v1 | Preserve shape; recovery uses same envelope before response commit | FR-002, FR-010, NFR-003 | AT-009, AT-011, REG-003 |
| Observability helpers | None identified | Create allowlist, redaction, metric registry, Sentry abstraction, operation vocabularies, route/result/category helpers | BR-002, FR-018, FR-019, NFR-001, NFR-002 | AT-004, AT-005, AT-006, UT-002, UT-004, UT-005, UT-010, UT-011 |
| PDS/OAuth write paths | Handlers receive `auth.PDSClientFactory`; `auth.PDSClient` hides indigo | Wrap factory and returned clients for session resume and PDS method metrics/logs/Sentry spans | BR-001, FR-006, FR-008, FR-009, FR-019 | AT-008, UT-005, UT-007, IT-004, IT-005 |
| Search DB timing | `routes.AddRoutes` creates `api.NewSearchStore(deps.DB)`; handlers call concrete store | Add operation-level timing to `SearchStore`, first `search.posts`, then remaining search operations | BR-001, FR-006, FR-020 | AT-012, UT-012, IT-008 |
| Tap consumer | `tap.WSConsumer` handles WS frames, indexer timeout, ack, reconnect state | Add Tap metrics, last event age, ack/reconnect counters, handle-duration histograms, and panic/error capture | FR-005, FR-011, NFR-005 | AT-007, UT-006, IT-003 |
| Index dispatcher | `index.Dispatcher` routes by NSID with fallback | Instrument registered vs fallback outcomes and known NSID labels without raw URI/CID/rkey | FR-005, FR-011, FR-018 | AT-007, UT-006, IT-003 |
| Startup/shutdown | `cmd/appview/main.go` starts HTTP and Tap goroutines, cleanup closes DB | Recover/capture background panics; flush Sentry during graceful shutdown through deps cleanup | FR-007, FR-009, FR-011, NFR-004 | AT-003, AT-004, IT-007 |
| Docs/env examples | `appview/README.md`, `prod.env.example`, compose host port `18080` | Document local `/metrics`, internal metric names/units, Sentry env vars, and production `/metrics` network/proxy restriction | FR-014, FR-015, FR-016, RULE-008 | MAN-003, MAN-004, MAN-005 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/observability/observability.go` | Create | Public package facade: `Observer`, noop implementation, safe field helpers, shared constants | FR-018, NFR-001, NFR-002 | UT-002, UT-004, UT-010, UT-011 |
| `appview/internal/observability/config.go` | Create | Validate Sentry DSN/release/sample-rate and unsafe body logging policy derived from `app.Config` | FR-012, FR-017, RULE-003, RULE-005, RULE-010 | UT-008, REG-005, REG-006 |
| `appview/internal/observability/metrics.go` | Create | Build isolated Prometheus registry and collectors; expose `http.Handler` for `/metrics` | FR-003, FR-004, FR-005, FR-006, FR-016 | AT-002, UT-010, UT-011, IT-001 |
| `appview/internal/observability/sentry.go` | Create | Sentry/noop client abstraction, event capture, span hooks, flush lifecycle | FR-007, FR-008, FR-009, RULE-004, RULE-007 | AT-003, AT-004, UT-004, IT-005, IT-007 |
| `appview/internal/observability/redaction.go` | Create | Header/body/value omission helpers and allowlist enforcement | BR-002, NFR-002, RULE-002, RULE-005 | AT-005, UT-002, IT-006 |
| `appview/internal/observability/pds.go` | Create | Bounded PDS operation/stage/category vocabulary and PDS client/factory wrappers | FR-006, FR-008, FR-019 | AT-008, UT-005, UT-007, IT-004 |
| `appview/internal/observability/db.go` | Create | Operation-level DB duration/result helper for search and health checks | FR-006, FR-020 | AT-012, UT-012, IT-008 |
| `appview/internal/observability/tap.go` | Create | Tap/indexer metrics and safe NSID/result/reason classification helpers | FR-005, FR-011 | AT-007, UT-006, IT-003 |
| `appview/internal/app/config.go` | Change | Add config fields/env parsing for Sentry and unsafe response-body logging | FR-012, FR-017, RULE-003, RULE-005 | UT-008, REG-005 |
| `appview/internal/app/config_test.go` | Change | Cover disabled defaults, prod unsafe body flag forced off, sample rate validation | FR-012, FR-017, RULE-003, RULE-005 | UT-008, REG-005, REG-006 |
| `appview/internal/app/deps.go` | Change | Create observed logger, Prometheus registry, observer, observed PDS factory, cleanup flush | FR-007, FR-009, FR-012, FR-013 | AT-003, AT-004, AT-010, IT-007 |
| `appview/internal/app/deps_test.go` | Change | Prove no Sentry DSN starts locally; cleanup flush called when configured | FR-007, FR-009, FR-013 | AT-010, IT-007 |
| `appview/cmd/appview/server.go` | Change | Add recovery and HTTP metrics middleware in existing stack | FR-004, FR-010, NFR-003 | AT-002, AT-009, AT-011, IT-001 |
| `appview/cmd/appview/server_test.go` | Change | Verify `/metrics` and optional Sentry lifecycle at server level | FR-003, FR-007, FR-015 | AT-002, AT-003, AT-010, IT-001, IT-007 |
| `appview/cmd/appview/main.go` | Change | Wrap Tap goroutine with background recovery/capture and flush on shutdown through cleanup | FR-009, FR-011 | AT-004, AT-007, IT-003, IT-007 |
| `appview/internal/routes/routes.go` | Change | Register `/metrics`; pass observer to stores where needed; preserve ops/v1 split | FR-003, FR-015, RULE-001 | AT-002, REG-002 |
| `appview/internal/routes/routes_test.go` | Change | Assert `/metrics` bypasses v1 auth/device middleware and route patterns stay bounded | FR-003, FR-015, FR-018 | AT-002, AT-006, IT-001, IT-002, REG-002 |
| `appview/internal/middleware/logging.go` | Change | Emit stable safe fields; remove default JSON payload logging; include route pattern/status/duration | FR-001, FR-002, FR-018, RULE-005 | AT-001, AT-005, AT-006, UT-001, REG-001 |
| `appview/internal/middleware/recovery.go` | Create | Recover HTTP panics, log/capture safely, return v1 envelope where possible | FR-009, FR-010 | AT-009, UT-009, REG-003 |
| `appview/internal/middleware/metrics.go` | Create | Record HTTP count, in-flight, duration, response size, panic count | FR-004, FR-018 | AT-002, AT-006, IT-001, IT-002 |
| `appview/internal/middleware/logging_test.go` | Change | Extend log field, redaction, route pattern, response-body logging tests | FR-001, FR-002, NFR-002 | AT-001, AT-005, AT-006, UT-001, IT-006 |
| `appview/internal/middleware/recovery_test.go` | Create | Panic before/after response write, Sentry capture, standard envelope, process continues | FR-009, FR-010 | AT-009, UT-009, REG-003 |
| `appview/internal/api/search.go` | Change | Preserve handler behavior; ensure route/correlation context reaches search store instrumentation | FR-020, NFR-003 | AT-012, IT-008, AT-011 |
| `appview/internal/api/search_store.go` | Change | Add bounded DB operation timing around search methods; first TDD target is `SearchPosts`/`search.posts` | FR-006, FR-020 | AT-012, UT-012, IT-008 |
| `appview/internal/api/search_store_test.go` | Change | Assert `search.posts` duration/result and no query text/raw identity in telemetry | FR-006, FR-020 | UT-012, IT-008 |
| `appview/internal/auth/pds_client.go` | Change only if needed | Keep interface stable if wrappers can live in observability; avoid method signature churn | FR-006, FR-019 | UT-007, IT-004 |
| `appview/internal/auth/pds_errors.go` | Change | Add or reuse classification helpers for timeout/network/auth/rate-limit/validation/not-found/forbidden/server/unexpected | FR-019 | UT-005 |
| `appview/internal/auth/*_test.go` | Change | Exercise session-resume and PDS classification with fakes | FR-006, FR-019 | UT-005, UT-007, IT-004 |
| `appview/internal/api/profile_test.go` | Change | Verify profile `PutRecord` paths emit write operation names without body/identity leakage | FR-006, FR-019 | IT-004 |
| `appview/internal/api/blob_test.go` | Change | Verify blob upload telemetry excludes media bytes and tokens | FR-006, NFR-002 | AT-008, IT-004, IT-006 |
| `appview/internal/api/follow_test.go` | Change | Verify follow/unfollow operation names and categorized failures | FR-006, FR-019 | UT-007, IT-004 |
| `appview/internal/api/post_test.go` | Change | Verify post create/delete, like/unlike, repost/unrepost operation names and failures | FR-006, FR-019 | UT-007, IT-004 |
| `appview/internal/tap/consumer.go` | Change | Record connected/reconnect/last event/ack/indexer metrics; recover per-event indexer panics | FR-005, FR-011 | AT-007, IT-003, GAP-001 |
| `appview/internal/tap/consumer_test.go` | Change | Fake Tap frames cover received/acked/ack failure/reconnect/indexer error/panic metrics | FR-005, FR-011 | AT-007, UT-006, IT-003 |
| `appview/internal/index/dispatcher.go` | Change | Record known/fallback NSID result, skip/error/index durations using safe collection labels | FR-005, FR-018 | AT-007, UT-006, IT-003 |
| `appview/internal/index/*_test.go` | Change | Verify registered NSIDs, skipped/fallback handling, and error metrics | FR-005, FR-011 | AT-007, UT-006 |
| `appview/environments/prod.env.example` | Change | Document Sentry vars and production `/metrics` restriction expectations | FR-012, FR-015, FR-017 | MAN-001, MAN-004 |
| `appview/environments/dev.env` | Change if needed | Add commented local-only observability examples; keep hosted export disabled by default | FR-012, FR-014 | MAN-003 |
| `appview/README.md` | Change | Add Observability section: local `/metrics`, metrics internal status, Sentry config, restriction note | FR-014, FR-015, FR-016 | MAN-003, MAN-004, MAN-005 |
| `appview/go.mod` / `appview/go.sum` | Change during implementation | Promote Prometheus client to direct dependency; add Sentry Go SDK if selected by implementation | FR-003, FR-007, FR-009 | Build/test commands |

## 5. Services, Interfaces, And Data Flow

Create `internal/observability` as the only package that knows metric names, Sentry tags, safe field allowlists, and operation vocabularies. Other packages should call small methods with bounded names rather than construct labels directly.

Partial interface sketch:

```text
type Observer interface {
    MetricsHandler() http.Handler
    HTTPMiddleware(logger *slog.Logger) func(http.Handler) http.Handler
    RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler
    ObserveDB(ctx, op, routePattern string, fn func(context.Context) error) error
    WrapPDSFactory(auth.PDSClientFactory) auth.PDSClientFactory
    Tap() TapObserver
    Indexer() IndexerObserver
    CaptureError(ctx, EventContext, error)
    StartSpan(ctx, SpanContext) (context.Context, Span)
    Flush(timeout time.Duration) bool
}
```

Use a noop implementation when Sentry is disabled. Prometheus metrics remain active even with Sentry disabled.

HTTP flow:

```text
request
  -> Logging(run_id, safe request fields)
  -> Recovery(capture panic, v1 envelope if possible)
  -> HTTPMetrics(count/in-flight/duration/size/status/route_pattern)
  -> CORS
  -> ServeMux
  -> per-route v1 middleware
  -> handler
```

Route pattern policy:

- Matched HTTP telemetry uses registered route pattern where available.
- Prefer `r.Pattern` after `ServeMux` dispatch for middleware completion metrics/logs; fall back to `unmatched` when blank.
- Existing `routes.RoutePolicy.PathPattern` remains the canonical v1 catalog for route tests and explicit handler setup.
- Do not use raw paths or query strings as metric labels, Sentry attributes, or production log fields.

PDS/write-proxy flow:

```text
handler -> observed PDS factory
    stage=session_resume, operation=oauth.session_resume
    -> observed PDS client
        method/collection -> bounded operation name
        result/category/stage/duration -> logs/metrics/span
```

Bounded PDS operations:

- `oauth.session_resume`
- `profile.put_bsky`
- `profile.put_craftsky`
- `post.create`
- `post.delete`
- `blob.upload`
- `follow.create`
- `follow.delete`
- `like.create`
- `like.delete`
- `repost.create`
- `repost.delete`

Bounded failure stages:

- `session_resume`
- `request_build`
- `pds_request`
- `pds_response`
- `post_write_indexing_wait` only if a current handler actually waits for indexing
- `unexpected`

Bounded categories:

- `timeout`
- `network`
- `auth`
- `rate_limited`
- `validation`
- `not_found`
- `forbidden`
- `server`
- `unexpected`

Search DB timing flow:

```text
GET /v1/search/posts
  -> HTTP middleware records route duration for /v1/search/posts
  -> SearchPostsHandler parses request without logging query text
  -> SearchStore.SearchPosts wraps DB work as db_op=search.posts
  -> DB duration metric/log shares route_pattern and run_id context
```

First DB target from document review `DR-001`: `GET /v1/search/posts` backed by `SearchStore.SearchPosts`, operation `search.posts`. After that test passes, extend the same helper to the remaining search store methods with bounded names such as `search.projects`, `search.profiles`, `search.hashtags`, `search.hashtag_posts`, and `search.suggestions`.

Tap/indexer flow:

```text
WSConsumer.Run
  -> connection state metrics and reconnect counters
  -> per-frame received/last_event metrics
  -> decode invalid identifiers: skipped/error metric, ack/drop safely
  -> handleWithTimeout
       -> recover indexer panic per event
       -> observe indexer duration/result/nsid/reason
  -> sendAck result metric
```

Background panic boundary from document review `DR-002`:

- Per-event indexer panics: recover inside `tap.WSConsumer.handleWithTimeout` so one malformed or buggy record cannot crash the consumer goroutine.
- Whole consumer goroutine panics: wrap the goroutine in `cmd/appview/main.go` with observer background recovery/capture, then close `consumerDone` predictably.

Safe attribute allowlist:

```text
service, environment, release, component, operation, route_pattern,
http_method, http_status, http_status_class, error_category,
failure_stage, duration, result, nsid, tap_connected,
reconnect_attempt, run_id, sentry_trace_id, sentry_span_id
```

Do not add attributes outside this list without updating tests and this plan or a manual note.

## 6. State, Providers, Controllers, Or DI

No Flutter/Riverpod state applies.

Use existing Go constructor dependency injection:

```text
LoadConfig
  -> NewDevDeps/NewProdDeps
     -> observability.New(Config, logger, clock/test hooks)
     -> deps.Observability
     -> deps.Logger = logger.With(service, environment)
     -> deps.NewPDSClient = observer.WrapPDSFactory(realFactory)
     -> deps.Consumer = tap.NewWSConsumer(..., Observer: observer.Tap())
     -> NewServer(ctx, deps)
        -> routes.AddRoutes(ctx, mux, deps)
        -> middleware stack uses deps.Observability
```

Keep tests isolated by using a per-deps Prometheus registry rather than package-global default registration.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

No Flutter UI, app route, lexicon, database, or product behavior change is planned.

User-facing/API surfaces:

- Add `GET /metrics` outside `/v1/`.
- Keep it unauthenticated by Craftsky app-session middleware.
- Return Prometheus text exposition, not the Craftsky JSON error envelope.
- Keep `/health` and `/healthz` behavior unchanged, including `/healthz` HTTP 200 degraded semantics.
- Preserve every existing `/v1/*` and `/oauth/*` response body, status, auth, device ID, rate-limit, and body-limit behavior.

Developer/operator docs:

- Add an `appview/README.md` Observability section with `curl http://localhost:18080/metrics` after `just dev-d`.
- Add production warning that `/metrics` must be restricted by network policy, reverse proxy, or platform ingress.
- Add metric names/units as internal ops details, not public compatibility guarantees.
- Add Sentry env var examples in `appview/environments/prod.env.example` with export disabled unless DSN is set.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Sentry DSN empty | Use noop Sentry client; app starts; Prometheus/logging still active; no external export | FR-012, FR-013, RULE-003 | AT-010, UT-008 |
| Sentry tracing absent | No trace export; optional trace/span IDs omitted from logs/events | FR-007, FR-012 | AT-003, AT-010 |
| Invalid Sentry sample rate | `LoadConfig` fails with named env-var validation error | FR-012, FR-017 | UT-008 |
| Prod unsafe response-body flag set | Force the setting off after validation so prod never logs bodies | RULE-005, NFR-002 | AT-005, REG-005 |
| Dev unsafe response-body flag absent | Response body logging remains disabled by default | RULE-005 | AT-005, REG-005 |
| Handler panic before write | Recovery logs/captures safe context and writes standard v1 envelope where possible | FR-009, FR-010 | AT-009, UT-009 |
| Handler panic after partial write | Recovery logs/captures best effort without process crash; does not attempt second envelope | FR-010 | UT-009 |
| Unmatched route | Route label/attribute is `unmatched`; raw path/query not emitted | FR-018, NFR-001 | AT-006, IT-002 |
| Matched identifier route | Use registered route pattern such as `/v1/posts/{did}/{rkey}` | FR-004, FR-018 | AT-006, IT-002 |
| Request has auth/cookie/DPoP/token headers | Redact or omit; no body capture by default | BR-002, NFR-002, RULE-002 | AT-005, UT-002, IT-006 |
| PDS expected 4xx | Metrics/logs classify; do not capture Sentry error event by default | FR-009, FR-019, RULE-007 | AT-008, UT-005 |
| PDS timeout/network/server/unexpected | Metrics/logs classify; Sentry captures actionable error when configured | FR-009, FR-019 | AT-004, AT-008 |
| Tap reconnect loop | Update connected gauge, reconnect counter, last error log, safe attempt number | FR-005 | AT-007, IT-003 |
| Invalid Tap envelope identifiers | Ack/drop as today; emit skipped/error metrics without raw record payload | FR-005, NFR-002 | AT-007, UT-006 |
| Indexer error | Record error count/duration by safe NSID/reason; Sentry capture when configured | FR-005, FR-011 | AT-007, IT-003 |
| Indexer panic | Recover per event, capture safely, avoid raw URI/CID/rkey/record payload | FR-011 | AT-007, GAP-001 |
| Slow search DB operation | Emit `search.posts` DB duration comparable with HTTP route duration by route/run_id | FR-020 | AT-012, UT-012, IT-008 |
| `/metrics` public in production | Code remains unauthenticated; docs/env example state production network/proxy restriction is mandatory | FR-015, RULE-008 | MAN-004 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001 / AT-002 | `appview/internal/routes/routes_test.go`, `appview/cmd/appview/server_test.go` | Build mux/server with test deps and no auth headers; request `/metrics` | 404 or Craftsky auth/device error instead of Prometheus text |
| 2 | REG-002 | `appview/internal/routes/routes_test.go` | Request `/metrics` and representative `/v1/whoami` without device/auth | `/metrics` missing; or `/v1/*` behavior accidentally changes |
| 3 | UT-010 / UT-011 | `appview/internal/observability/metrics_test.go` | Test registry collectors | No `craftsky_appview` metrics, unclear units, or accidental exemplar/Sentry metric setup |
| 4 | UT-008 / AT-010 | `appview/internal/app/config_test.go`, `appview/internal/app/deps_test.go` | Empty DSN, dev/prod env, sample rates, unsafe flag | Missing config fields or startup requires Sentry |
| 5 | UT-002 / AT-005 | `appview/internal/observability/redaction_test.go`, `appview/internal/middleware/logging_test.go` | Sensitive headers, bodies, token-like values | Authorization-only redaction; response JSON payload logged by default |
| 6 | UT-001 / AT-001 | `appview/internal/middleware/logging_test.go` | Captured JSON slog output for representative request | Missing service/env/route/status/duration or run_id not shared |
| 7 | UT-003 / AT-006 | `appview/internal/routes/routes_test.go`, `appview/internal/observability/*_test.go` | Matched identifier routes, query params, unmatched path | Raw path/query or identifier appears in telemetry fields |
| 8 | IT-002 | `appview/internal/middleware/logging_test.go`, `appview/internal/routes/routes_test.go` | Serve requests through mux with telemetry capture | Metrics/logs use raw paths or no route pattern |
| 9 | UT-012 / AT-012 | `appview/internal/observability/db_test.go`, `appview/internal/api/search_store_test.go` | Instrument `SearchStore.SearchPosts` as `search.posts` with fake/captured observer | No comparable DB operation duration or raw search query leaks |
| 10 | IT-008 | `appview/internal/api/search_test.go`, `appview/internal/api/search_store_test.go` | Authenticated `GET /v1/search/posts` request with test DB/fake timing | HTTP duration and DB duration cannot be correlated by route/run_id |
| 11 | UT-009 / AT-009 | `appview/internal/middleware/recovery_test.go` | Panicking handler before and after write; fake Sentry capture | Panic crashes test or wrong error envelope |
| 12 | UT-005 | `appview/internal/observability/pds_test.go`, `appview/internal/auth/pds_errors_test.go` | Table of PDS errors and stages | Errors map to raw/unbounded strings |
| 13 | UT-007 | `appview/internal/observability/pds_test.go`, handler package tests | Enumerate all current write operations | Missing instrumentation name for a write path |
| 14 | IT-004 / AT-008 | `appview/internal/api/profile_test.go`, `blob_test.go`, `follow_test.go`, `post_test.go`, `auth/*_test.go` | Fake PDS/OAuth clients return success and categorized failures | No metrics/logs for operation/result/stage/category |
| 15 | UT-006 / AT-007 | `appview/internal/tap/consumer_test.go`, `appview/internal/index/*_test.go` | Fake Tap frames for success, skip, malformed, indexer error | No Tap/indexer counters/gauges/durations |
| 16 | GAP-001 / IT-003 | `appview/internal/tap/consumer_test.go` | Fake indexer panics inside `handleWithTimeout` | Panic escapes consumer goroutine |
| 17 | UT-004 / AT-004 | `appview/internal/observability/sentry_test.go` | Test transport/event context with allowed and disallowed fields | Sentry context contains raw identity/body/token or misses run_id |
| 18 | IT-005 / AT-003 | `appview/internal/observability/*_test.go`, `appview/internal/api/*_test.go` | Sentry test transport with tracing enabled and fake PDS write | Missing trace/span IDs or unbounded span attributes |
| 19 | IT-007 | `appview/internal/app/deps_test.go`, `appview/cmd/appview/server_test.go` | Failing test transport and shutdown cleanup | Runtime export failure blocks request or flush not called |
| 20 | AT-011 / REG-004 | Existing route, health, and middleware tests | Run representative packages with instrumentation enabled/disabled | Existing response/auth/healthz behavior changes |
| 21 | MAN-003 | Dev stack manual check | `just dev-d`, `curl http://localhost:18080/metrics` | `/metrics` unavailable or no `craftsky_appview` metric |
| 22 | MAN-004 | `appview/README.md`, `prod.env.example` | Human review of docs | No production restriction note |
| 23 | MAN-005 | `appview/README.md` metric list | Compare metric names/help/units | Names/units undocumented or presented as public API |

Focused commands during implementation:

```text
cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/observability -count=1
cd appview && go test ./internal/api ./internal/auth ./internal/tap ./internal/index -count=1
just test
just fmt
```

## 10. Sequencing And Guardrails

- First TDD step: `AT-002` / `IT-001`, proving unauthenticated `GET /metrics` returns Prometheus text outside `/v1/*` while a representative `/v1/*` route still requires the existing auth/device middleware.
- Dependencies between work items: `/metrics` needs the registry before route registration; config/redaction should land before Sentry export; PDS/search/Tap wrappers should use the shared observer vocabulary before handler instrumentation expands.
- Guardrails: use route patterns, not raw paths; use allowlisted attributes only; no request/response bodies in prod; no tokens, DPoP material, raw DID/handle/device/session IDs, AT-URIs, CIDs, rkeys, query text, uploaded media, or raw record payloads in telemetry; no synchronous vendor calls on request hot paths; keep Prometheus registry test-local; keep `/healthz` semantics unchanged.
- Out of scope: Flutter observability, product analytics, dashboards, alert rules, hosted Prometheus/Grafana, OpenTelemetry/OTLP export, Sentry Application Metrics, Prometheus exemplars, DB auto-instrumentation spans, database migrations, lexicon changes, API response contract changes, auth behavior changes, and production ingress/network policy implementation.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Exact Sentry production release value is not defined. | Events may lack deploy-version grouping until release process exists. | Use optional `SENTRY_RELEASE`; omit when unset. Document in `prod.env.example`. |
| CPQ-002 | Non-blocking | Exact conservative production trace sample rate is not specified. | Sampling could be too high or too low if chosen casually. | Default tracing disabled unless explicitly enabled; if enabled without explicit sample rate in prod, use conservative `0.01` and validate `[0,1]`. |
| CPQ-003 | Non-blocking | Sentry Go SDK dependency is absent from `go.mod`. | Implementation must add a new dependency and may need network access. | Add `github.com/getsentry/sentry-go` during implementation only after Sentry tests drive it; keep abstraction thin. |
| CPQ-004 | Non-blocking | `r.Pattern` availability through outer middleware needs a test before relying on it. | Route labels could fall back to `unmatched` too often. | First route-pattern tests should verify `r.Pattern`; if not reliable, add explicit route-pattern context middleware at route registration. |
| CPQ-005 | Non-blocking | `auth.PDSClient` method signatures do not carry operation names. | Some operations using the same collection/method need careful classification. | Classify by method+collection first; use route/operation context only if a collision appears. Keep the public interface unchanged if possible. |
| CPQ-006 | Non-blocking | Background panic recovery can hide a bug if only counted. | Consumer might continue after an invariant-breaking panic. | Capture/log panic with component and safe NSID context, increment metric, and let poison-pill/drop behavior remain explicit in tests. |
| CPQ-007 | Non-blocking | Metric name list may evolve during TDD. | Docs could drift from implementation. | Add metric registry/name tests and update README after collectors stabilize. |
| CPQ-008 | Non-blocking | Production `/metrics` restriction is outside AppView code. | Public exposure remains possible if deployment is misconfigured. | Code endpoint stays unauthenticated per requirement; README and `prod.env.example` must state network/proxy/platform restriction as mandatory. |

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `AT-002` / `IT-001` for unauthenticated `GET /metrics` outside `/v1/*`.
- Focused command: `cd appview && go test ./cmd/appview ./internal/routes -count=1`
- Notes: Create `internal/observability` early and keep it small. Prove the route and registry first, then pull logging, recovery, PDS, DB, Tap, and Sentry instrumentation through that shared boundary. Do not touch lexicons, migrations, Flutter code, or product API behavior for this slice.
