# Requirements: AppView Observability

## 1. Initial Request

Plan the next AppView slice for observability. The initial candidate scope is Sentry, structured JSON logs, OpenTelemetry traces, and a Go Prometheus metrics endpoint at `/metrics`, with openness to other supporting ideas.

## 2. Current Codebase Findings

- Relevant files:
  - AppView process entry point and shutdown: `appview/cmd/appview/main.go`.
  - HTTP server construction and top-level middleware: `appview/cmd/appview/server.go`.
  - Dependency and logger wiring: `appview/internal/app/deps.go`.
  - Configuration loading and validation: `appview/internal/app/config.go`.
  - Route registration: `appview/internal/routes/routes.go`.
  - Request logging middleware: `appview/internal/middleware/logging.go`.
  - Health endpoints: `appview/internal/api/health.go`, `appview/internal/api/healthz.go`.
  - Tap consumer and indexer wiring: `appview/internal/app/deps.go`, `appview/internal/tap/*`, `appview/internal/index/*`.
  - Dev stack: `docker-compose.yml`, `justfile`, `appview/environments/dev.env`, `appview/environments/prod.env.example`.
  - API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`.
- Existing patterns:
  - AppView uses Go stdlib `net/http`, `http.ServeMux`, `slog`, `pgx`, and explicit dependency wiring.
  - `NewDevDeps` uses debug-level JSON `slog`; `NewProdDeps` uses info-level JSON `slog`.
  - `slog.SetDefault(logger)` is already used so libraries using `slog.Default()` inherit the AppView logger.
  - `middleware.Logging` creates a per-request UUID `run_id`, puts it in context, and logs request start plus debug request/response details.
  - `/health` checks DB liveness; `/healthz` reports DB and Tap consumer state and always returns HTTP 200 with `status` set to `ok` or `degraded`.
  - `/v1/*` routes are authenticated except login and use route-specific middleware for auth, device ID, rate limiting, and body limits.
- Current behavior:
  - Logs are already JSON, but the contract for fields, redaction, correlation, and response-body logging is not formalized.
  - There is no committed `/metrics` route.
  - There is no explicit OpenTelemetry provider/exporter setup in AppView startup.
  - There is no explicit Sentry setup or panic/error capture path.
  - Request context has `run_id`, but there is no trace ID/span ID propagation contract.
  - Tap consumer and indexer behavior is logged in places, but there is no formal metric or trace coverage requirement for firehose ingestion.
- Constraints discovered:
  - Ops endpoints such as `/health`, `/healthz`, and `/metrics` are outside `/v1/`, consistent with the API architecture spec.
  - The Flutter app must continue to hold only Craftsky session tokens; observability must not expose PDS tokens, OAuth refresh/access tokens, Craftsky session tokens, DPoP material, or private AppView data.
  - The AppView currently runs inside Docker for dev; Go tests run on the host against compose Postgres.
  - Existing request logging can include JSON response payloads at debug level, which is risky once external log/error/trace systems are wired.
  - `go.sum` already contains Prometheus and OpenTelemetry packages, but requirements should not assume they are fully integrated.
- Test/build commands discovered:
  - Full AppView tests: `just test` after `just dev-d`.
  - Focused AppView tests: `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api -count=1`.
  - Formatting/vet: `just fmt`.
  - Dev stack: `just dev` or `just dev-d`.

## 3. Clarifying Questions And Decisions

### Q1: Which observability pillars should be included in the first AppView observability slice?
Answer: Include Sentry, structured JSON logs, and a minimal Go Prometheus `/metrics` endpoint in the first slice. Defer vendor-neutral OpenTelemetry/OTLP export and Sentry Application Metrics.
Decision / implication: The first slice should be Sentry-first for hosted error monitoring and tracing, with local/vendor-neutral JSON logs and Prometheus metrics retained for operational contracts. Requirements are scoped to AppView startup, HTTP requests, Tap/indexer background work, PDS/OAuth write paths, and safe configuration.

### Q2: Should alerting dashboards, hosted vendor setup, and production infrastructure be in scope?
Answer: Not explicitly provided.
Decision / implication: This requirements slice should expose useful signals and document alert-ready metric/log/error fields, but actual hosted dashboards, alert rules, and production deployment wiring are non-goals unless added later.

### Q3: Should `/metrics` require authentication?
Answer: Recommendation accepted for now: expose `/metrics` as an unauthenticated ops endpoint in AppView, keep the payload free of secrets/user content, and rely on deployment/network restrictions for production scrape access.
Decision / implication: This slice should not add Craftsky session auth or a custom scrape-token scheme to `/metrics`. The coding plan should document that production deployments must restrict network/proxy access to `/metrics`.

### Q4: Should observability capture full request/response bodies?
Answer: The existing middleware can log JSON response payloads at debug level, but no product/security decision authorizes exporting payloads to third parties.
Decision / implication: This slice shall require body and payload capture to be disabled in prod for logs, Prometheus metrics, and Sentry, with explicit redaction rules for headers and sensitive fields. Dev may retain response-payload logging for local testing only if it is gated to dev and never exported externally.

### Q5: What failure mode should observability prioritize first?
Answer: PDS write failures are the most important expected production failure to diagnose.
Decision / implication: PDS/OAuth write-proxy operations should be first-class in logs, metrics, Sentry error capture, and Sentry tracing when enabled, with bounded operation/result/error-category labels and no tokens or request payloads.

### Q6: What Tap/indexer signal is minimally acceptable?
Answer: Include all proposed signals: connected state, last event age, per-NSID indexed/error counts, and ack failures.
Decision / implication: Tap/indexer metrics are a Must-level requirement, not a later enhancement.

### Q7: Should OpenTelemetry include DB spans?
Answer: Defer OpenTelemetry in this slice.
Decision / implication: DB latency should still be visible through operation-level Prometheus duration histograms and safe structured logs. Sentry SDK spans may be used where they fall naturally out of Sentry tracing, but vendor-neutral DB spans and pgx auto-instrumentation are out of scope.

### Q8: What trace sampling posture should be used?
Answer: Sentry tracing may be configured through the Sentry SDK, but OpenTelemetry/OTLP trace export is deferred.
Decision / implication: The requirements should require safe disabled defaults for Sentry export and configurable Sentry trace sampling if Sentry tracing is enabled. No OTLP sampling/export contract is required in the first slice.

### Q9: Which correlation ID is canonical?
Answer: Recommendation: keep the existing `run_id` as the human-facing request ID and include Sentry trace/span IDs when Sentry tracing is active.
Decision / implication: Do not replace `run_id`. Logs, Sentry events, and error envelopes continue to use it; Sentry traces may add trace/span IDs for hosted telemetry correlation.

### Q10: Should `/healthz` semantics change?
Answer: Leave `/healthz` behavior unchanged for now.
Decision / implication: Health endpoint status-code semantics are out of scope; metrics carry alertable degradation signals.

### Q11: Which metric names are stable contracts?
Answer: No external stable contract is needed yet.
Decision / implication: Metric names should be documented for operators and tests, but treated as internal ops implementation details until dashboards/alerts are formalized.

### Q12: Which PDS write paths should be instrumented?
Answer: All existing AppView-to-PDS write paths should be instrumented in the first slice.
Decision / implication: PDS write telemetry must cover every current write path, including profile writes, post creates/deletes, blob uploads, follows/unfollows, likes/unlikes, reposts/unreposts, and OAuth/session-resume work that can fail before the PDS write.

### Q13: How should DB spans be scoped?
Answer: Defer DB span requirements.
Decision / implication: The first slice should use operation-level DB metrics and logs. DB spans may be added later when OpenTelemetry or deeper Sentry tracing is intentionally scoped.

### Q14: Which failures should Sentry capture?
Answer: Sentry should capture panics, AppView 5xx/internal errors, and high-severity unexpected PDS failure categories, but not every expected operational failure.
Decision / implication: Normal validation, unauthorized/expired session, forbidden, not-found, and rate-limited PDS responses should stay in logs/metrics and Sentry spans when tracing is enabled, but should not become Sentry error events by default. Sentry should capture unexpected categories such as timeout, network, PDS 5xx/server, malformed unexpected response, and token/session handling bugs.

### Q15: Should `/metrics` access restrictions be a hard production requirement?
Answer: Yes. AppView should serve `/metrics` without Craftsky user auth, and production deployments must restrict access by network policy, reverse proxy, or platform ingress rules.
Decision / implication: Requirements should treat public production exposure of `/metrics` as a deployment/security violation even though the AppView endpoint itself is unauthenticated.

### Q16: What exact production trace sampling default should be used?
Answer: No OpenTelemetry production trace sampling default is required in this slice.
Decision / implication: If Sentry tracing is enabled, production sampling must be configurable and default conservatively. A specific OTLP parent-based sampling policy is deferred.

### Q17: Should telemetry use an explicit safe attribute allowlist?
Answer: Yes.
Decision / implication: Logs, metrics, Sentry events, and Sentry spans should use an allowlist of bounded fields. New fields outside that allowlist require explicit justification in the coding plan.

### Q18: Should HTTP metrics and spans use route patterns?
Answer: Yes. HTTP metrics, logs, and Sentry context/spans must use registered route patterns rather than raw request paths.
Decision / implication: Implementation should use route-policy/route-pattern lookup where possible. Unknown routes should use a bounded fallback such as `unmatched`.

### Q19: Should dev response-body logging require an explicit unsafe flag?
Answer: Yes.
Decision / implication: Dev response-body logging should be disabled by default and require an explicit unsafe local-only flag such as `APPVIEW_UNSAFE_LOG_RESPONSE_BODIES=true`; it must be forced off in prod and must never export to Sentry or any future external telemetry backend.

### Q20: Should Sentry events include user identity?
Answer: No, not in this slice.
Decision / implication: Sentry must not include raw DID, handle, device ID, session ID, token, AT-URI, CID, rkey, or email. If user grouping is needed later, it should use a separate, explicit design such as irreversible keyed hashing.

### Q21: How should PDS write failures be classified?
Answer: Use bounded stage plus category fields.
Decision / implication: PDS write failures should include bounded stages such as `session_resume`, `request_build`, `pds_request`, `pds_response`, and optionally `post_write_indexing_wait`, plus bounded categories such as `timeout`, `network`, `auth`, `rate_limited`, `validation`, `not_found`, `forbidden`, `server`, and `unexpected`.

### Q22: Should metrics include latency histograms?
Answer: Yes.
Decision / implication: The AppView should emit histograms for HTTP request duration, PDS write duration, selected DB operation duration, and Tap/indexer handling duration where feasible.

### Q23: Should metrics include exemplars or trace links?
Answer: No, not in this first slice.
Decision / implication: Prometheus metrics, logs, and Sentry events/traces should be correlatable where practical through `run_id` and Sentry trace IDs when enabled, but Prometheus exemplars are a non-goal for v1 observability.

### Q24: Which OpenTelemetry exporter should be required first?
Answer: None in the first slice.
Decision / implication: OpenTelemetry/OTLP export is deferred. If added later, it should be designed as a separate vendor-neutral tracing/export slice.

### Q25: Should Sentry Application Metrics be used in the first slice?
Answer: No. Defer Sentry Application Metrics.
Decision / implication: The first slice should use Prometheus `/metrics` as the metric contract. Sentry Application Metrics can be evaluated later as a hosted diagnostic supplement once the core operational metrics are stable.

### Q26: Should request-backed DB operation telemetry answer whether most response time is spent in DB work?
Answer: Yes. The goal is not to trace every SQL query, but maintainers should be able to compare total request duration with bounded DB operation duration for key request-backed operations such as search.
Decision / implication: The first slice should require route/request-correlated DB operation metrics or logs for selected read/write operations, especially search, so maintainers can answer whether a slow response is mostly database time without requiring per-query SQL spans or OpenTelemetry/pgx instrumentation.

## 4. Candidate Approaches

### Option A: Sentry-first AppView observability baseline across HTTP and background workers
Summary: Add safe structured logging conventions, request correlation, a minimal Prometheus `/metrics` endpoint, Sentry error capture and optional Sentry SDK tracing, and health/operation signals across HTTP, Tap consumer, indexers, DB calls where feasible, and startup/shutdown paths. Defer OpenTelemetry/OTLP and Sentry Application Metrics.
Pros:
- Covers the operational paths most likely to fail: API traffic, DB access, OAuth/PDS writes, Tap ingestion, indexers, startup, and shutdown.
- Gives maintainers enough signal to debug local and deployed AppView behavior without waiting for a broader platform project.
- Keeps the scope AppView-only and does not alter product/API behavior.
- Creates a consistent safety contract for secrets and user content before exporting data to third-party systems.
- Avoids implementing multiple tracing and metrics backends before there is a concrete hosted collector/dashboard need.
Cons:
- Touches cross-cutting middleware and dependency wiring.
- Adds configuration and test surface around initialization and shutdown.
- Requires careful cardinality control for metrics and Sentry context/span attributes.
Risks:
- Over-instrumentation could leak sensitive data or create expensive, noisy telemetry.
- Middleware changes could accidentally alter request behavior if not tested.

### Option B: Minimal HTTP-only metrics and logs
Summary: Tighten request logs and expose basic HTTP Prometheus metrics, deferring traces, Sentry, and Tap/indexer metrics.
Pros:
- Lower implementation risk.
- Fastest path to basic production visibility.
- Smaller dependency and configuration surface.
Cons:
- Leaves background ingestion, indexing, and PDS/OAuth failures under-observed.
- Does not address error aggregation or distributed request correlation.
- Likely requires a second cross-cutting pass soon after.
Risks:
- Maintainers may still lack enough context for the AppView-specific failures that matter most.

### Option C: Vendor-only Sentry observability
Summary: Wire Sentry for panics, explicit errors, logs, traces, and application metrics, with local Prometheus metrics deferred.
Pros:
- Captures actionable errors quickly in one hosted UI.
- Minimizes local route and metrics design.
Cons:
- Does not provide a mature vendor-neutral scrapeable contract for service health, latency, throughput, Tap lag, or rate-limit visibility.
- Depends on Sentry Application Metrics for operational metrics even though this project does not yet need hosted metric features in the first slice.
Risks:
- Error capture alone can miss degraded behavior, slow paths, and ingestion stalls.
- A Sentry-only design can make local development and future non-Sentry operations harder than necessary.

## 5. Recommended Direction

Recommended approach: Option A: Sentry-first AppView observability baseline across HTTP and background workers.

Why: AppView reliability depends on user-facing HTTP, background firehose ingestion, DB access, and PDS write-proxy operations. A useful first slice should cover request correlation, local JSON logs, Sentry errors/optional Sentry tracing, a minimal Prometheus metrics contract, and health signals without changing product behavior. Deferring OpenTelemetry/OTLP and Sentry Application Metrics keeps the first implementation focused while preserving the option to add vendor-neutral tracing or hosted metric exploration later.

## 6. Problem / Opportunity

The AppView is becoming the central read model, write proxy, OAuth mediator, moderation/report intake surface, and firehose indexer for Craftsky. Today it has basic JSON logs and health checks, but no explicit metrics endpoint, hosted error aggregation, or formal redaction/correlation contract. This makes it harder to debug user-visible API failures, PDS/OAuth issues, Tap/indexer stalls, degraded DB connectivity, rate-limit behavior, and startup/shutdown problems.

## 7. Goals

- G-001: Provide safe, structured, correlated AppView logs suitable for local Docker and production log aggregation.
- G-002: Expose Prometheus-compatible metrics for HTTP traffic, process health, DB checks, Tap consumer state, indexer activity, rate limiting, PDS write outcomes, and key outbound dependency outcomes.
- G-003: Add Sentry capture for panics and actionable errors without leaking secrets or user content.
- G-004: Allow Sentry SDK tracing/spans where useful for hosted investigation, with safe sampling and bounded attributes, without requiring OpenTelemetry/OTLP export.
- G-005: Preserve existing `/v1/*`, `/oauth/*`, `/health`, and `/healthz` behavior except for added observability side effects.
- G-006: Make observability configurable by environment and safe when optional backends are not configured.
- G-007: Create enough testable requirements for a follow-on acceptance-test stage.

## 8. Non-Goals

- NG-001: Build production dashboards, hosted Prometheus/Grafana, hosted OpenTelemetry collector, or Sentry project configuration.
- NG-002: Add Flutter client observability.
- NG-003: Add product analytics, user tracking, conversion funnels, or algorithmic ranking telemetry.
- NG-004: Change AppView API response contracts, auth semantics, database schema, lexicons, or Flutter behavior.
- NG-005: Persist observability events in AppView Postgres.
- NG-006: Capture full request bodies, response bodies, OAuth tokens, Craftsky session tokens, DPoP keys, PDS tokens, or private AppView data in telemetry.
- NG-007: Add alert rules, SLO definitions, or production dashboards.
- NG-008: Make metric names a public compatibility contract for external consumers in this slice.
- NG-009: Add Prometheus exemplars or trace-linked metric exemplars.
- NG-010: Add OpenTelemetry/OTLP tracing export, OTLP/gRPC, or multiple trace exporters in the first slice.
- NG-011: Add Sentry user identity or user grouping.
- NG-012: Add Sentry Application Metrics in the first slice.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| AppView maintainer | Developer/operator running Craftsky locally or in production | Quickly diagnose failures, slow requests, ingestion stalls, and degraded dependencies |
| Backend contributor | Engineer adding AppView features | Clear conventions for logs, metrics, Sentry traces/spans where enabled, and error capture |
| On-call/operator | Person responsible for service health | Scrapeable metrics, safe logs, actionable errors, and health signals |
| Craftsky user | End user of the Flutter app | More reliable API and ingestion behavior without exposing private credentials or content |

## 10. Current Behavior

AppView starts a JSON `slog` logger, registers routes on a standard `http.ServeMux`, wraps the server with CORS and request logging middleware, starts a Tap consumer alongside the HTTP listener, and exposes `/health` and `/healthz`. Request logs include a generated `run_id`; handlers often include that `run_id` in logs. There is no defined `/metrics` endpoint, no OpenTelemetry lifecycle, no Sentry lifecycle, no documented metric names, and no formal telemetry redaction contract.

## 11. Desired Behavior

After the change, AppView emits consistent JSON logs with safe fields and correlation IDs, exposes a minimal Prometheus metrics contract at `/metrics`, captures panics and selected errors in Sentry when configured, optionally records Sentry SDK traces/spans when configured, and continues running normally when optional Sentry export is not configured. Telemetry must help diagnose AppView reliability without changing functional behavior or exposing secrets/user content.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Maintainers must be able to diagnose AppView request failures, PDS write failures, dependency degradation, DB-dominated slow responses, and firehose ingestion/indexing issues from emitted telemetry. | AppView is the operational center of reads, writes, auth mediation, and indexing; PDS write failures are the highest-priority expected production failure; maintainers also need to know when request latency is mostly database work. | Prompt / User answer / Codebase | AC-001, AC-002, AC-003, AC-004, AC-010, AC-019 |
| BR-002 | Business | Must | Observability must not expose credentials, tokens, DPoP material, private AppView data, raw user identity, or full user-generated content. | Craftsky session and PDS token boundaries are core architecture rules. | AGENTS.md / Codebase / User answer | AC-005, AC-006, AC-015 |
| FR-001 | Functional | Must | The AppView shall emit structured JSON logs with stable field names for timestamp, level, message, environment, service name, run/request ID where applicable, method, route pattern, status, duration, and error details where applicable. | Logs already use JSON but need a stable contract for aggregation and tests. | Prompt / Codebase / User answer | AC-001, AC-005, AC-014 |
| FR-002 | Functional | Must | The AppView shall keep the existing `run_id` as the human-facing request correlation ID, include it in logs, Sentry events, and error responses where an error response already includes a request ID, and include Sentry trace/span IDs in logs and Sentry events when Sentry tracing is active. | Cross-signal correlation is necessary for debugging, and the existing request ID remains useful outside tracing. | Codebase / API spec / User answer | AC-001, AC-003, AC-004, AC-018 |
| FR-003 | Functional | Must | The AppView shall expose a Prometheus-compatible `GET /metrics` endpoint outside `/v1/`. | Ops endpoints are outside `/v1/`, and the prompt explicitly asks for Go Prometheus metrics. | Prompt / API spec | AC-002 |
| FR-004 | Functional | Must | The AppView shall record HTTP server metrics including request count, in-flight requests, response duration histogram, response size where practical, method, registered route pattern, and status code. | Request throughput, latency, and error rates are baseline service metrics; route patterns avoid identifier leakage. | Discovery / User answer | AC-002, AC-006, AC-014 |
| FR-005 | Functional | Must | The AppView shall record Tap consumer and indexer metrics for connected state, reconnect attempts, last-event freshness or age, events received, events acknowledged, ack failures, per-NSID records indexed, per-NSID records skipped, per-NSID indexing errors, and handling duration histograms where feasible. | Firehose ingestion is a critical non-HTTP path, and all listed signals are required for the first slice. | Codebase / Reference doc / User answer | AC-002, AC-017 |
| FR-006 | Functional | Must | The AppView shall record DB/dependency outcome metrics for health pings, selected operation-level manual DB operations, and every existing outbound PDS/OAuth write-proxy path, using low-cardinality operation/result/stage/error-category labels and duration histograms. | PDS write failures are the highest-priority expected production failure; dependency latency can degrade before outright failure. | Codebase / User answer | AC-002, AC-006, AC-010, AC-013, AC-017, AC-019 |
| FR-007 | Functional | Should | The AppView should support Sentry SDK tracing/spans when Sentry is configured, with environment-configured sampling, safe disabled defaults, and graceful shutdown/flush behavior. | Sentry can provide hosted request and operation context without requiring a separate OpenTelemetry/OTLP backend in the first slice. | Prompt / Codebase / User answer | AC-003, AC-016 |
| FR-008 | Functional | Should | When Sentry tracing is enabled, the AppView should create bounded spans for incoming HTTP requests, every existing outbound PDS/OAuth write-proxy path, and important background operations including Tap consume loops and indexer handling; DB visibility in this slice shall be provided by metrics/logs rather than required DB spans. | This connects user-facing requests, PDS failures, and background work to Sentry without expanding the first slice into full vendor-neutral tracing. | Prompt / Codebase / User answer | AC-003, AC-006, AC-010, AC-013 |
| FR-009 | Functional | Must | The AppView shall initialize Sentry when configured, capture unhandled panics, AppView 5xx/internal errors, selected high-severity PDS/server failures, and selected server/background errors, and flush Sentry during graceful shutdown. | Sentry is explicitly requested and should stay actionable rather than capturing expected operational failures. | Prompt / User answer | AC-004, AC-015 |
| FR-010 | Functional | Must | The AppView shall recover from panics in HTTP handling, log/capture the panic, and return the existing standard API error envelope for `/v1/*` where possible. | Panic capture must not bypass API conventions. | API spec / Discovery | AC-004 |
| FR-011 | Functional | Should | The AppView should capture background Tap/indexer panics and errors with enough context to identify component, NSID where applicable, and failure category. | Background failures can stop indexing without an HTTP request. | Codebase | AC-004 |
| FR-012 | Functional | Must | Observability backends shall be controlled through validated environment configuration with safe disabled defaults for optional exports. | Local dev and CI must work without hosted vendors. | Codebase | AC-007 |
| FR-013 | Functional | Must | The AppView shall continue to start and serve HTTP when Sentry is not configured. | Optional hosted telemetry should not block local development. | Discovery | AC-007 |
| FR-014 | Functional | Should | Dev Docker configuration should make `/metrics` reachable on the existing AppView host port and document how to inspect it locally. | Contributors need a low-friction verification path. | Codebase | AC-008 |
| FR-015 | Functional | Must | The AppView shall not require Craftsky user authentication or `/v1/*` middleware for `GET /metrics`; production access control must be handled by network policy, reverse proxy, or platform ingress restrictions documented during implementation. | Prometheus scrape endpoints should avoid app-session coupling while remaining deployable behind restricted ops access. | User answer / Recommendation | AC-002, AC-011 |
| FR-016 | Functional | Should | AppView metric names and units should be documented for contributors and tests, while remaining internal ops implementation details rather than a public compatibility contract in this slice. | Operators need understandable metrics, but dashboards/alerts are not formalized yet. | User answer / Recommendation | AC-002 |
| FR-017 | Functional | Should | If Sentry tracing is enabled, production trace sampling shall default conservatively and be environment-configurable; dev/local tracing may use higher sampling when explicitly configured. | Prevents accidental full production trace export while preserving local debugging utility. | User answer | AC-016 |
| FR-018 | Functional | Must | Telemetry fields shall follow a safe allowlist: service name, environment, version/release if configured, component, operation name, registered route pattern, HTTP method, HTTP status/status class, bounded error category, bounded failure stage, duration, result, known registered NSID/collection, Tap connection state, retry/reconnect attempt number, `run_id`, and Sentry trace/span IDs when Sentry tracing is enabled. | An allowlist is easier to implement and review than only a denylist. | User answer | AC-006, AC-014, AC-015 |
| FR-019 | Functional | Must | PDS write failure telemetry shall distinguish bounded failure stages and categories. | Separating pre-PDS, PDS-request, and response-classification failures makes write failures diagnosable without leaking payloads. | User answer | AC-010, AC-013 |
| FR-020 | Functional | Must | For selected request-backed DB operations, including AppView search routes, telemetry shall make it possible to compare total HTTP request duration with bounded DB operation duration for the same route and correlation context without requiring per-query SQL spans. | Maintainers need to answer whether most of a slow response was spent in database work while keeping this slice lighter than full SQL tracing. | User answer | AC-019 |
| NFR-001 | Non-functional | Must | Metrics and Sentry context/span attributes must avoid high-cardinality values such as raw DIDs, handles, AT-URIs, CIDs, rkeys, query text, device IDs, tokens, raw paths with identifiers, and full URL query strings. | High cardinality causes cost and privacy problems. | Discovery / User answer | AC-006, AC-014 |
| NFR-002 | Non-functional | Must | Logs, Sentry events/spans, and metrics must redact or omit `Authorization`, session tokens, OAuth/PDS tokens, DPoP proofs/keys, cookies, request bodies, response bodies, uploaded media content, and raw user identity by default. | Prevents secret, identity, and content leakage. | AGENTS.md / Codebase / User answer | AC-005, AC-006, AC-012, AC-015 |
| NFR-003 | Non-functional | Must | Observability instrumentation must not change successful response bodies, status codes, auth requirements, rate-limit behavior, or route availability for existing routes. | This slice is operational, not product behavior. | Discovery | AC-009 |
| NFR-004 | Non-functional | Should | Instrumentation overhead should be low enough for local development and small production deployments, with bounded labels and no per-request blocking network calls on the hot path. | Observability should not materially degrade AppView performance. | Discovery | AC-009 |
| NFR-005 | Non-functional | Should | Metric names should use a consistent `craftsky_appview` service naming convention and document units for counters, gauges, and histograms; Sentry span operation names should use the same bounded operation vocabulary where spans are enabled. | Clear naming reduces future dashboard churn. | Discovery | AC-002, AC-003, AC-017 |
| RULE-001 | Business rule | Must | `/metrics` is an ops endpoint and shall not be placed under `/v1/` or use the `/v1/*` JSON error envelope. | Existing API architecture keeps ops endpoints outside app API versioning. | API spec | AC-002, AC-011 |
| RULE-002 | Business rule | Must | Observability must treat PDS tokens, OAuth refresh/access tokens, Craftsky session tokens, DPoP private keys/proofs, and uploaded media as secrets. | This follows the AppView token boundary and privacy model. | AGENTS.md | AC-005 |
| RULE-003 | Business rule | Must | External Sentry telemetry export shall be disabled unless explicitly configured through environment variables. | Avoids accidental vendor traffic in local dev/CI. | Discovery | AC-007 |
| RULE-004 | Business rule | Should | Sentry event context should use stable technical identifiers and bounded tags, not raw user content or unbounded user identifiers. | Maintains usefulness while limiting privacy/cost risk. | Discovery | AC-004, AC-006, AC-015 |
| RULE-005 | Business rule | Must | Production telemetry shall not log or export request or response bodies; dev-only response-payload logging shall be disabled by default, require an explicit unsafe local-only flag, be forced off in prod, and never export to external telemetry systems. | Dev diagnostics can be useful, but body logging is risky and should require deliberate opt-in. | User answer | AC-005, AC-012 |
| RULE-006 | Business rule | Must | `/healthz` shall keep its existing HTTP status semantics in this slice. | Health behavior changes are out of scope. | User answer | AC-009 |
| RULE-007 | Business rule | Must | Sentry events shall not include raw DID, handle, device ID, session ID, token, AT-URI, CID, rkey, or email in user/context fields in this slice. | Error monitoring should not become an identity export path. | User answer | AC-015 |
| RULE-008 | Business rule | Must | Public production exposure of `/metrics` is not allowed. | Metrics can expose operational state even without user content. | User answer | AC-011 |
| RULE-009 | Business rule | Must | Prometheus exemplars or trace-linked metric exemplars are out of scope for the first observability slice. | Exemplars depend on later collector/dashboard wiring and are not needed for the baseline. | User answer | AC-018 |
| RULE-010 | Business rule | Must | OpenTelemetry/OTLP export is out of scope for the first observability slice. | The first slice should avoid maintaining parallel tracing/export systems before there is a concrete collector/backend need. | User answer | AC-016 |
| RULE-011 | Business rule | Must | Sentry Application Metrics are out of scope for the first observability slice. | Prometheus `/metrics` remains the metric contract; Sentry Application Metrics can be evaluated later as a hosted supplement. | User answer | AC-002, AC-017 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-002 | Given an AppView HTTP request, when the request completes, then structured JSON logs include stable request fields and a correlation ID that also appears in handler error logs for that request. |
| AC-002 | BR-001, FR-003, FR-004, FR-005, FR-006, NFR-005, RULE-001, RULE-011 | Given AppView is running, when `GET /metrics` is requested, then it returns Prometheus text exposition containing HTTP metrics and AppView-specific Tap/indexer/dependency metrics without requiring `/v1/` routing or Sentry Application Metrics. |
| AC-003 | BR-001, FR-002, FR-007, FR-008, NFR-005 | Given Sentry tracing is configured, when HTTP requests and Tap/indexer operations execute, then Sentry spans are created with service metadata, correlation linkage, duration, status, and safe low-cardinality attributes; and spans/events are flushed on graceful shutdown. |
| AC-004 | BR-001, FR-002, FR-009, FR-010, FR-011, RULE-004 | Given Sentry is configured, when an HTTP panic, selected HTTP server error, or background Tap/indexer error occurs, then Sentry captures an event with component, environment, release if configured, correlation ID where available, and redacted context; and events are flushed on graceful shutdown. |
| AC-005 | BR-002, FR-001, NFR-002, RULE-002 | Given requests include authorization headers, OAuth/PDS/DPoP data, or request/response bodies, when logs and Sentry events are emitted, then those secrets and payloads are redacted or omitted by default. |
| AC-006 | BR-002, FR-004, FR-006, FR-008, NFR-001, NFR-002, RULE-004 | Given telemetry is emitted for routes with path/query identifiers or user/account data, then metrics labels and Sentry context/span attributes use route patterns or bounded categories rather than raw DIDs, handles, AT-URIs, CIDs, rkeys, query strings, tokens, or user content. |
| AC-007 | FR-012, FR-013, RULE-003 | Given Sentry DSN is absent, when AppView starts in dev or tests, then startup succeeds, logs remain local JSON, `/metrics` remains available, and no external Sentry telemetry export is attempted. |
| AC-008 | FR-014 | Given the dev Docker stack is running, when a contributor requests the AppView `/metrics` endpoint on the documented host port, then metrics are reachable and include at least one AppView service metric. |
| AC-009 | NFR-003, NFR-004, RULE-006 | Given existing AppView route tests and representative HTTP requests, when observability instrumentation is enabled or disabled, then existing response status/body/auth behavior and `/healthz` status-code semantics remain unchanged and tests continue to pass. |
| AC-010 | BR-001, FR-006, FR-008, FR-019 | Given an outbound PDS/OAuth write-proxy operation or selected manual DB operation succeeds or fails, when telemetry is emitted, then logs and metrics include the bounded operation name, result, duration, stage where applicable, and safe error category without tokens or request payloads; Sentry spans include the same bounded context when Sentry tracing is enabled. |
| AC-011 | FR-015, RULE-001, RULE-008 | Given `GET /metrics` is requested without Craftsky session headers, when AppView receives the request, then the endpoint is handled outside `/v1/*` app auth middleware and returns Prometheus metrics rather than a Craftsky JSON auth error; and production deployment documentation states that `/metrics` must be network/proxy restricted. |
| AC-012 | NFR-002, RULE-005 | Given AppView runs in prod, when HTTP requests or errors are logged/captured/exported, then request and response bodies are not included; given AppView runs in dev, any response-payload logging remains local-only and is not exported to Sentry or any future external telemetry backend. |
| AC-013 | FR-006, FR-019 | Given each existing AppView-to-PDS write path executes, when it succeeds or fails, then it emits consistent write telemetry with operation, result, duration, bounded stage, and bounded category. |
| AC-014 | FR-004, FR-018, NFR-001 | Given HTTP metrics, logs, or Sentry spans are emitted for matched and unmatched routes, then matched routes use registered route patterns and unmatched routes use a bounded fallback such as `unmatched`; raw request paths and query strings are not used as labels/attributes. |
| AC-015 | FR-009, NFR-002, RULE-007 | Given Sentry captures an event, then it contains only technical context such as `run_id`, trace ID, route pattern, operation, component, error category, status, environment, and release, and it does not include raw user identity or expected user/actionable PDS 4xx failures by default. |
| AC-016 | FR-007, FR-017, RULE-010 | Given Sentry tracing is configured in dev/local, then higher sampling is allowed only by explicit configuration; given Sentry tracing is configured in prod, then sampling defaults conservatively and remains environment-configurable; and no OpenTelemetry/OTLP exporter is required. |
| AC-017 | FR-004, FR-005, FR-006, RULE-011 | Given HTTP requests, PDS writes, selected DB operations, and feasible Tap/indexer handling operations execute, then Prometheus duration histograms are emitted for those operation classes without requiring Sentry Application Metrics. |
| AC-018 | FR-002, RULE-009 | Given metrics, logs, and Sentry events/spans are emitted, then they can be correlated through `run_id` and Sentry trace/span IDs where practical, but no Prometheus exemplar support is required. |
| AC-019 | BR-001, FR-006, FR-020 | Given a request-backed DB operation such as AppView search executes, when telemetry is emitted, then maintainers can compare total HTTP request duration with bounded DB operation duration for the same route and correlation context to determine whether most response time was spent in database work, without requiring per-query SQL spans or raw SQL parameter logging. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Sentry DSN is empty | Sentry is disabled without startup failure and without per-request errors. | FR-012, FR-013, RULE-003 |
| EC-002 | Sentry tracing is not configured | AppView still starts, logs remain local JSON, Prometheus metrics remain available, and no tracing export is attempted. | FR-007, FR-012, FR-013 |
| EC-003 | Sentry is configured but temporarily unavailable at runtime | AppView continues serving requests; Sentry export failures are logged safely if surfaced by the SDK and do not block the hot path. | FR-009, NFR-004 |
| EC-004 | HTTP handler panics after partially writing a response | Panic is logged/captured; best-effort recovery avoids process crash, recognizing the response may already be committed. | FR-010 |
| EC-005 | Tap consumer reconnects repeatedly | Metrics and logs reflect reconnect attempts and current degraded state without high-cardinality labels. | FR-005, NFR-001 |
| EC-006 | Indexer receives unsupported or malformed records | Metrics/logs categorize skipped or failed records by bounded reason and NSID where safe. | FR-005, FR-011 |
| EC-007 | Request path contains DID/rkey/query text | Metrics, logs, and Sentry spans use registered route patterns or a bounded fallback; raw identifiers and query text are not labels. | FR-004, FR-008, NFR-001 |
| EC-008 | Debug logging is enabled in dev | Debug logs remain structured and omit sensitive headers, tokens, and request bodies; any response-payload logging is local-only and not exported externally. | NFR-002, RULE-005 |
| EC-009 | PDS write fails after request validation succeeds | Logs, metrics, and Sentry error/spans where enabled capture the operation/result/error category without exporting OAuth/PDS tokens, DPoP proof material, or the request body. | FR-006, FR-008, FR-009, NFR-002 |
| EC-010 | A selected DB operation is slow but succeeds | Operation-level duration metrics and structured logs make the slow operation visible without recording raw SQL parameters or user content. | FR-006, NFR-001 |
| EC-011 | `/metrics` is scraped without `Authorization` or `X-Craftsky-Device-Id` | Endpoint returns Prometheus exposition and does not invoke `/v1/*` auth/device middleware. | FR-015, RULE-001 |
| EC-012 | PDS returns an expected validation, unauthorized, forbidden, not-found, or rate-limited response | Metrics/logs classify the failure, Sentry spans may include the bounded category if tracing is enabled, and Sentry does not capture it as an error event by default. | FR-009, FR-019, RULE-007 |
| EC-013 | A route contains DID, handle, rkey, hashtag, search text, or cursor-like values | HTTP metrics, logs, and Sentry spans use a route pattern or bounded fallback instead of the raw path/query. | FR-004, FR-018, NFR-001 |
| EC-014 | Dev operator enables `APPVIEW_UNSAFE_LOG_RESPONSE_BODIES=true` | Response-body logging is allowed only locally in dev, remains redacted for sensitive headers/tokens, and cannot export to Sentry or any future external telemetry backend. | RULE-005, NFR-002 |
| EC-015 | Production operator accidentally sets `APPVIEW_UNSAFE_LOG_RESPONSE_BODIES=true` | AppView ignores or rejects the unsafe setting in prod so response bodies are not logged/exported. | RULE-005 |
| EC-016 | A search request is slow because the backing DB operation dominates total response time | HTTP request duration and bounded DB operation duration can be compared by route and correlation context so the DB-dominated response is visible without per-query spans. | BR-001, FR-006, FR-020 |

## 15. Data / Persistence Impact

- New fields: None expected in AppView Postgres.
- Changed fields: None expected.
- Migration required: No.
- Backwards compatibility: Existing database contents and schema should be unaffected.

## 16. UI / API / CLI Impact

- UI: None.
- API: Add ops endpoint `GET /metrics` outside `/v1/`; no change to existing `/v1/*`, `/oauth/*`, `/health`, or `/healthz` response contracts. `/metrics` should not require Craftsky app-session headers; production access must be restricted by network/proxy/platform ingress configuration.
- CLI: No CLI behavior change required. Documentation may mention `curl` against the dev AppView host port.
- Background jobs: Tap consumer and indexers gain logs, metrics, Sentry error capture, and optional Sentry spans where configured, but functional ingestion/indexing behavior should not change.

## 17. Security / Privacy / Permissions

- Authentication: Existing app API authentication remains unchanged. `/metrics` should not use Craftsky user authentication; production scrape access must be restricted by network/proxy/platform ingress configuration.
- Authorization: No user authorization semantics change.
- Sensitive data: Tokens, DPoP material, cookies, request bodies, response bodies, uploaded media, raw user identity, and raw user-generated content must be omitted or redacted by default.
- Abuse cases: Publicly exposed metrics could reveal service behavior or operational state; production deployment must restrict scrape access at the network/proxy/platform ingress layer.

## 18. Observability

- Events:
  - Startup, dependency initialization, configuration mode, listener start, graceful shutdown, shutdown timeout/error.
  - HTTP request start/completion/error/panic.
  - Tap connection changes, reconnect attempts, ack failures, ingest errors.
  - Indexer success/skip/error by bounded component/NSID/reason.
- Logs:
  - JSON `slog` remains the logging foundation.
  - Stable fields should include service, environment, component, run/request ID, Sentry trace ID/span ID where available, method, route pattern, status, duration, and safe error category/message.
  - No raw request/response body logging in prod.
  - Dev response-body logging must be disabled by default, require an explicit unsafe local-only flag, and never export to Sentry or any future external telemetry backend.
- Metrics:
  - HTTP request count, duration histogram, in-flight, response size where practical, and panic count.
  - Tap connected state, reconnect attempts, last event age/freshness, events received/acked/error count.
  - Indexer records handled/skipped/failed by bounded NSID/reason and handling duration histograms where feasible.
  - DB health ping outcomes, selected operation-level manual DB operation duration histograms/outcomes, and every existing PDS/OAuth write-proxy operation duration histogram/outcome.
  - For selected request-backed DB operations such as search, DB operation duration should be comparable with total HTTP request duration by route and correlation context to identify DB-dominated slow responses without per-query SQL spans.
  - Process/runtime metrics from the Go Prometheus client where appropriate.
- Traces:
  - OpenTelemetry/OTLP export is deferred.
  - Sentry SDK tracing/spans are optional and disabled unless Sentry tracing is explicitly configured.
  - Sentry trace sampling must default conservatively in production and be configurable by environment.
  - DB spans are not required in the first slice; selected DB operation visibility comes from metrics and logs.
  - Prometheus exemplars are not required.
  - Sentry Application Metrics are deferred; Prometheus `/metrics` is the first-slice metric contract.
- Safe attribute allowlist:
  - service name, environment, version/release if configured, component, operation name, registered route pattern, HTTP method, HTTP status/status class, bounded error category, bounded failure stage, duration, result, known registered NSID/collection, Tap connection state, retry/reconnect attempt number, `run_id`, Sentry trace ID, Sentry span ID.
- Alerts:
  - Not implemented in this slice.
  - Metrics should be suitable for future alerts on high HTTP 5xx rate, p95 latency, Tap disconnected/degraded state, indexing error spikes, DB health failures, and panic/Sentry event spikes.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Telemetry leaks secrets, tokens, or user content. | Severe privacy/security issue and possible vendor data exposure. | Require default body omission, header redaction, bounded attributes, tests for redaction, and explicit non-goals for payload capture. |
| RISK-002 | Metric labels or Sentry span/context attributes have high cardinality. | Increased cost, degraded metrics/backend performance, unusable dashboards, or noisy hosted traces. | Require route patterns/bounded labels and prohibit raw identifiers/query text in labels/attributes. |
| RISK-003 | Cross-cutting middleware changes alter existing request behavior. | User-facing regressions across the AppView API. | Require route behavior preservation and focused middleware/server tests. |
| RISK-004 | Optional vendor/exporter failures block requests or startup. | Local dev or production availability suffers due to telemetry backend issues. | Require disabled-by-default external export and non-blocking hot-path behavior. |
| RISK-005 | `/metrics` exposure leaks operational detail if publicly accessible. | Attackers can inspect service health/traffic patterns. | Keep metric contents free of user content/secrets and require production network/proxy/platform ingress restrictions. |
| RISK-006 | Too much is included in the first slice. | Implementation becomes broad and hard to verify. | Keep scope AppView-only, avoid dashboards/alerts/client telemetry, and prioritize HTTP plus Tap/indexer signals. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | AppView observability is the only target for this slice; Flutter observability is future work. | Requirements would need expansion for client logs/crashes/traces. |
| ASM-002 | Prometheus scraping can be handled by deployment/network configuration rather than AppView user auth in the first pass. | `/metrics` may need endpoint-level authentication or binding controls added in a later slice if deployment cannot reliably restrict access. |
| ASM-003 | Optional Sentry export should be disabled unless configured. | Local dev/CI behavior and config requirements would change. |
| ASM-004 | Existing Go dependencies in `go.sum` can be reused where appropriate, but implementation planning must verify exact direct dependency needs. | Coding plan may need dependency additions or version adjustments. |
| ASM-005 | It is acceptable for `/healthz` to remain HTTP 200 with JSON `status` while metrics provide alertable degradation signals. | Health endpoint semantics may need a separate behavior-change requirement. |

## 21. Open Questions

- [ ] Non-blocking: What exact Sentry environment and release values should production use?
- [ ] Non-blocking: What conservative default should be used for Sentry production trace sampling if Sentry tracing is enabled in the first implementation pass?

## 22. Review Status

Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer:
Date: 2026-06-29
Notes: Medium risk because this is cross-cutting operational infrastructure with privacy/security implications, even though it should not change product behavior.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-29-appview-observability/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - BR-001, BR-002
  - FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-009, FR-010, FR-012, FR-013, FR-015, FR-018, FR-019, FR-020
  - NFR-001, NFR-002, NFR-003
  - RULE-001, RULE-002, RULE-003, RULE-005, RULE-006, RULE-007, RULE-008, RULE-009, RULE-010, RULE-011
- Suggested test levels:
  - Unit tests for config parsing, redaction helpers, metric label sanitization, DB operation naming/correlation helpers, and panic/error capture helpers.
  - Middleware/server tests for request logging, correlation IDs, unchanged route behavior, panic recovery, and `/metrics`.
  - Focused integration tests for metrics emitted from HTTP and Tap/indexer paths where feasible.
  - Manual dev-stack verification for `GET /metrics` and optional configured Sentry smoke checks.
- Blocking open questions: None.
