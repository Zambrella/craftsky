# Requirements: AppView Sentry Observability Consolidation

## 1. Initial Request

After a first AppView observability pass, consolidate AppView observability into Sentry for errors, logs, tracing, and metrics. Keep the implementation simple for a single developer while placing the major observability pillars, especially metrics, behind interfaces so future provider changes remain practical. Revisit Sentry Go documentation for usage, tracing, logs, and metrics. Relax error-event redaction where safe by using bounded sentinel/enum values instead of redacting every value, while keeping panic reporting highly redacted. Add spans at business-logic and general-work boundaries rather than relying mostly on one request-level span.

## 2. Current Codebase Findings

- Relevant files:
  - Prior requirements and implementation trace: `docs/changes/2026-06-29-appview-observability/01-requirements.md`, `docs/changes/2026-06-29-appview-observability/05-implementation-plan.md`.
  - AppView observability package: `appview/internal/observability/metrics.go`, `appview/internal/observability/sentry.go`, `appview/internal/observability/pds.go`, `appview/internal/observability/db.go`, `appview/internal/observability/tap.go`.
  - HTTP observability middleware: `appview/internal/middleware/metrics.go`, `appview/internal/middleware/logging.go`, `appview/internal/middleware/recovery.go`.
  - Tap consumer instrumentation: `appview/internal/tap/consumer.go`.
  - Route registration: `appview/internal/routes/routes.go`.
  - AppView configuration and dependency wiring: `appview/internal/app/config.go`, `appview/internal/app/deps.go`.
  - AppView README observability section: `appview/README.md`.
  - API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`, `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`.
- Existing patterns:
  - AppView uses explicit dependency wiring, stdlib `net/http`, `slog`, and an `observability.Observer` object.
  - Current metrics are Prometheus collectors exposed from an isolated registry via unauthenticated `GET /metrics`.
  - Sentry is optional and disabled unless `SENTRY_DSN` is set.
  - Sentry tracing is controlled by `SENTRY_TRACING_ENABLED` and `SENTRY_TRACES_SAMPLE_RATE`.
  - Current Sentry capture uses `sentry.NewClient`, `sentry.NewHub`, safe event context tags, `SendDefaultPII: false`, and `Flush`.
  - Current spans exist for `http.server`, PDS write-proxy work, `tap.consume`, and `tap.indexer.handle`.
  - DB operation observability currently records Prometheus metrics only for bounded operations such as `search.posts`.
- Current behavior:
  - Logs are structured JSON locally with route patterns and `run_id`, but they are not sent to Sentry structured logs as the canonical log backend.
  - Metrics are Prometheus-first through `/metrics`; Sentry Application Metrics are not used.
  - Most HTTP requests get one root request transaction/span, with some deeper spans only where prior code added them.
  - Sentry error events currently redact exception values broadly and preserve only a tight context allowlist.
  - PDS failures are classified into bounded categories such as `timeout`, `network`, `auth`, `rate_limited`, `validation`, `not_found`, `forbidden`, `server`, and `unexpected`.
- Constraints discovered:
  - The Flutter app must continue to hold only Craftsky session tokens; observability must not expose PDS tokens, OAuth refresh/access tokens, Craftsky session tokens, DPoP material, or private AppView data.
  - AppView reads come from Postgres/AppView; writes are mediated through the PDS by AppView.
  - `/v1/*` JSON body and error-envelope conventions must remain unchanged.
  - Sentry Go docs describe errors/panics as captured events, tracing as transactions and spans, logs as structured searchable log entries, and Application Metrics as counters, gauges, and distributions.
  - Sentry tracing docs require explicit sampling via a sample rate or sampler and recommend performance testing before high-throughput tracing deployment.
  - Sentry setup docs show SDK initialization early in application lifecycle, `SendDefaultPII`, `EnableTracing`, `TracesSampleRate`, `EnableLogs`, and process shutdown flushing as relevant configuration points.
  - Sentry metrics docs currently mark Application Metrics as new/beta, so metrics should have a provider interface to reduce future migration cost.
  - Sentry custom instrumentation docs recommend adding child spans inside handlers when a transaction already exists and finishing spans deliberately.
  - Sentry request-module custom instrumentation docs are relevant for outbound HTTP-style work, including PDS/OAuth requests, where AppView needs bounded external-call spans.
  - Sentry `slog` docs provide a direct `sentryslog` handler with level filtering, context attributes, source inclusion controls, and `ReplaceAttr` filtering.
  - Sentry sensitive-data docs recommend SDK-side scrubbing with `BeforeSend` and `BeforeSendTransaction`, disabling default PII when desired, and avoiding sensitive data in breadcrumbs, HTTP context, transaction names, and spans.
  - Sentry SQL instrumentation wraps `database/sql` drivers and creates SQL query/exec spans only under an active transaction. AppView uses `pgxpool` directly across the codebase rather than `database/sql`, so `sentrysql` is not a natural fit for this slice without a DB access-layer change.
- Test/build commands discovered:
  - Full test path: `just dev-d` then `just test`.
  - Focused AppView path: `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`.
  - Formatting/vet: `just fmt`.

## 3. Clarifying Questions And Decisions

### Q1: Should this second pass update the existing observability change folder or create a new one?
Answer: Create a new folder.
Decision / implication: This requirements artifact lives at `docs/changes/2026-07-02-appview-sentry-observability/01-requirements.md` and treats the 2026-06-29 observability work as the existing baseline.

### Q2: Should AppView observability move entirely to Sentry?
Answer: Yes. Move errors, logs, tracing, and metrics to Sentry.
Decision / implication: Sentry becomes the primary external observability backend. Prometheus `/metrics` should not remain the operational metrics contract for AppView after this change.

### Q3: Should provider flexibility still be preserved?
Answer: Yes, especially for metrics; an interface is acceptable.
Decision / implication: Metrics must be behind an interface. Error capture, logging, and tracing should also avoid spreading direct Sentry SDK calls through business code where a small local abstraction fits the existing `Observer` pattern.

### Q4: Should error redaction remain as strict as the current implementation?
Answer: No. Panic values should stay highly redacted, but ordinary captured errors can include bounded sentinel/enum values that improve diagnosis.
Decision / implication: Error telemetry should distinguish panic reporting from classified operational errors. Non-panic events may include safe fields such as operation, component, route pattern, status/status class, failure stage, category, and sentinel error code. Raw user identifiers, request bodies, OAuth material, tokens, record payloads, raw AT-URIs, CIDs, rkeys, handles, and emails remain forbidden.

### Q5: Should spans remain mostly request-level?
Answer: No. Create spans at the borders of business logic and general work.
Decision / implication: HTTP transactions should contain child spans for meaningful work boundaries such as auth/session validation, route handler business operations, DB operations, PDS/OAuth calls, blob upload, indexer handling, Tap receive/ack work, and other long-running or failure-prone units.

### Q6: Should `/metrics` be removed immediately, hidden behind config, or left temporarily for local compatibility?
Answer: Remove it.
Decision / implication: The Sentry consolidation shall remove the public Prometheus `/metrics` route and update docs/tests accordingly. Sentry metrics become the only primary metrics path for AppView.

### Q7: Which exact Sentry metric names should be used for the first implementation?
Answer: Reuse the metric names created for Prometheus.
Decision / implication: Sentry metrics should retain the existing `craftsky_appview_*` names where Sentry's API allows, changing only what is required by Sentry naming/type constraints.

### Q8: Which route/handler business operations should be included in the first span-boundary allowlist?
Answer: Include auth/session, DB, PDS/OAuth, Tap/indexer, search, feed, profile, and write handlers.
Decision / implication: The first span-boundary pass should explicitly cover those operation families and may add similarly bounded operations where the implementation naturally exposes them.

### Q9: Should AppView use Sentry SQL instrumentation?
Answer: Not in this slice.
Decision / implication: Sentry's SQL integration targets `database/sql`, while AppView uses `pgxpool` directly. This slice should keep manual bounded DB operation spans behind the AppView observability interface rather than changing the DB layer to fit `sentrysql`.

### Q10: Should removing `/metrics` leave any default local runtime metrics surface?
Answer: No.
Decision / implication: With no Sentry DSN or with metrics disabled, metrics calls should use no-op or in-memory test implementations. Local troubleshooting should rely on stdout structured logs and tests, not a replacement metrics endpoint.

### Q11: Should Sentry logs replace local JSON logs?
Answer: No. Keep local JSON logs and add Sentry log export when explicitly enabled.
Decision / implication: Docker/dev workflows continue to receive stdout structured logs. Sentry logs are an additional filtered sink, not the only log destination.

### Q12: Should observability use one combined interface or separate narrow interfaces?
Answer: Use separate narrow interfaces, optionally grouped by a small `Observer` struct for wiring.
Decision / implication: Metrics, tracing, errors, and logs should be independently testable and replaceable without creating a broad generic framework.

### Q13: Should Sentry SDK imports be forbidden outside observability and startup/wiring?
Answer: Yes.
Decision / implication: Direct `sentry-go` imports should be limited to `appview/internal/observability` and narrowly approved startup/log-handler wiring where unavoidable.

### Q14: Where should spans be created?
Answer: Spans may be created inside handlers, storage, and indexers through the tracing interface when that is the cleanest work boundary.
Decision / implication: The boundary rule is no direct Sentry SDK usage plus bounded span names/attributes, not forcing all spans into middleware decorators.

### Q15: Should non-panic errors send raw Go error strings to Sentry?
Answer: Use a hybrid allowlist. Do not send arbitrary raw `err.Error()` values by default.
Decision / implication: Sentry error values should be safe sentinel codes/messages for explicitly classified errors. Wrapped upstream errors, DB errors, parse errors, validation details, and user-provided text are unsafe unless explicitly classified as safe.

### Q16: Should local JSON logs include raw error strings when Sentry logs do not?
Answer: Yes, but only for local stdout logs where the current code already treats them as acceptable.
Decision / implication: Sentry-bound log attributes should use the same safe classifier as Sentry error events, while local logs can preserve existing debuggability.

### Q17: Should the requirements distinguish Sentry logs from breadcrumbs?
Answer: Yes.
Decision / implication: Sentry logs are searchable log entries. Breadcrumbs are optional context attached to future events. The first implementation should prioritize logs and traces, adding breadcrumbs only where they are low-cardinality and clearly useful.

### Q18: Should metric callers use Prometheus-style labels or AppView-domain methods?
Answer: Use AppView-domain metric methods.
Decision / implication: The Sentry metrics implementation should map those methods to reused `craftsky_appview_*` metric names where Sentry permits. Callers should not depend on Prometheus collector or label concepts.

### Q19: How should invalid metric and span attributes be handled?
Answer: Normalize safely at runtime and fail in tests.
Decision / implication: Production telemetry should degrade to bounded fallback values such as `unknown` or `other`, while in-memory test implementations and validation helpers should catch unbounded tags, operation names, routes, identifiers, and token-like values.

### Q20: How broad should DB spans be?
Answer: Span named storage operations, not every SQL call.
Decision / implication: DB spans should represent useful AppView operations such as `feed.list`, `profile.get`, `search.posts`, `session.lookup`, and write-side storage mutations. They should avoid sqlc query granularity unless a broader operation must be intentionally split.

### Q21: Should DB spans include row counts?
Answer: Only bounded result classes, not exact counts.
Decision / implication: Use values such as `none`, `one`, `some`, `many`, or configured buckets rather than exact result counts.

### Q22: Should HTTP status be exact or class-only?
Answer: Include exact HTTP status codes and status classes.
Decision / implication: HTTP status codes are bounded and useful, but raw paths, queries, bodies, and user identifiers remain forbidden.

### Q23: Which Sentry features should be enabled by `SENTRY_DSN` alone?
Answer: Errors and panics only.
Decision / implication: Logs, tracing, and metrics each require explicit enablement flags and safe defaults.

### Q24: Should Tap/indexer tracing have independent volume control?
Answer: Yes.
Decision / implication: Tap/indexer tracing should have operation-aware sampling or independent configuration. Success-path Tap spans should be sampled; errors and panics should remain visible.

### Q25: Should Prometheus dependencies and collectors be removed immediately?
Answer: Yes.
Decision / implication: Removing `/metrics` also means removing Prometheus collectors and dependencies in this slice once Sentry/no-op metrics implementations replace them.

## 4. Candidate Approaches

### Option A: Sentry-backed observability with local interfaces
Summary: Make Sentry the concrete external backend for errors, logs, tracing, and metrics while keeping AppView code behind narrow local interfaces for metrics, tracing, errors, and Sentry-bound log emission. Keep local stdout JSON logs for dev/Docker workflows. Remove Prometheus as the runtime metrics path. Add business/work-boundary spans and a tiered privacy policy for panic versus classified error events.
Pros:
- Matches the single-platform simplicity goal.
- Keeps most business code independent of direct Sentry SDK calls.
- Makes metrics provider changes practical if Sentry Application Metrics are not sufficient later.
- Allows better traces without changing API behavior.
- Improves error usefulness through safe sentinel/enum values.
Cons:
- Requires cross-cutting changes in observability, middleware, config, routes, and docs.
- Sentry Application Metrics are currently a newer/beta feature.
- Requires disciplined cardinality and privacy review across logs, spans, metrics, and events.
Risks:
- Over-instrumentation can create noise or cost.
- Relaxed redaction could accidentally expose sensitive values if not enforced by tests and allowlists.

### Option B: Sentry-only direct SDK calls
Summary: Replace Prometheus and local observers with direct `sentry-go` calls wherever errors, logs, spans, and metrics are emitted.
Pros:
- Lowest abstraction overhead.
- Fastest conceptual path to "all in Sentry."
Cons:
- Couples business and middleware code tightly to Sentry.
- Makes future provider changes expensive.
- Makes consistent redaction and cardinality control harder.
Risks:
- Direct SDK usage can spread inconsistent telemetry conventions through the codebase.

### Option C: Keep Prometheus metrics and add Sentry logs/metrics in parallel
Summary: Keep `/metrics` and Prometheus as a first-class metrics backend while also emitting Sentry logs, traces, errors, and metrics.
Pros:
- Retains the prior operational contract.
- Provides an easier rollback for metrics.
Cons:
- Does not meet the simplicity goal of consolidating observability into one platform.
- Maintains two metrics systems and doubles the surface area for labels/cardinality.
Risks:
- The single-developer workflow remains more complex than necessary.

## 5. Recommended Direction

Recommended approach: Option A: Sentry-backed observability with local interfaces.

Why: The user wants one operational platform, and Sentry now has documented support for error capture, structured logs, tracing, and Application Metrics in Go. A local interface boundary keeps the codebase simple today while protecting the project from a costly rewrite if Sentry metrics or another pillar needs to change later. This direction also lets the second pass fix the trace-depth issue by making spans part of the AppView business/work boundary contract.

## 6. Problem / Opportunity

The first observability pass made AppView safer and more visible, but it still leaves the maintainer operating multiple observability concepts: Prometheus metrics for operational signals, local JSON logs, and partial Sentry error/tracing export. That is more platform surface than a single developer needs right now. Consolidating into Sentry can simplify day-to-day debugging, but the implementation should preserve enough internal abstraction to avoid locking every call site to Sentry SDK details.

## 7. Goals

- G-001: Make Sentry the primary AppView backend for errors, logs, tracing, and metrics.
- G-002: Keep AppView observability calls behind separate narrow local interfaces for metrics, tracing, errors, and logs, optionally grouped for wiring.
- G-003: Replace mostly request-level traces with transactions and child spans at meaningful business and work boundaries.
- G-004: Improve Sentry error usefulness with bounded sentinel/enum fields while preserving strict privacy and panic redaction.
- G-005: Preserve existing AppView API, auth, PDS, Tap, and database behavior except for observability side effects.
- G-006: Keep Sentry optional and safe by default in local/dev environments, with DSN-only behavior limited to errors and panics.
- G-007: Leave enough traceability for acceptance-test design.

## 8. Non-Goals

- NG-001: Add Flutter client observability.
- NG-002: Build Sentry dashboards, alerts, issue ownership rules, uptime checks, or production notification routing.
- NG-003: Add OpenTelemetry/OTLP exporters or a vendor-neutral collector in this slice.
- NG-004: Create a general-purpose observability framework beyond the interfaces needed by AppView.
- NG-005: Store observability data in AppView Postgres.
- NG-006: Change `/v1/*`, `/oauth/*`, AppView auth semantics, lexicons, database schema, or Flutter behavior.
- NG-007: Capture request bodies, response bodies, raw record payloads, OAuth tokens, PDS tokens, Craftsky session tokens, DPoP keys, private AppView data, raw handles, raw DIDs, raw AT-URIs, raw CIDs, raw rkeys, emails, or device/session identifiers in Sentry.
- NG-008: Add product analytics, user tracking, conversion funnels, or algorithmic ranking telemetry.
- NG-009: Guarantee permanent compatibility for current Prometheus metric names.
- NG-010: Adopt Sentry `sentrysql` / `database/sql` instrumentation or migrate AppView away from direct `pgxpool` usage in this slice.
- NG-011: Add a replacement local metrics endpoint after removing `/metrics`.
- NG-012: Capture every individual SQL query as its own span.
- NG-013: Make breadcrumbs a primary observability deliverable in this slice.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| AppView maintainer | Single developer/operator running Craftsky locally and in production | One place to inspect errors, logs, traces, and metrics without managing a separate metrics stack |
| Backend contributor | Developer adding AppView routes, indexers, or PDS write paths | Simple local interfaces and clear rules for spans, metrics, logs, and error fields |
| Craftsky user | End user of the Flutter app | More reliable service behavior without credential, identity, or content leakage |

## 10. Current Behavior

AppView currently emits structured local JSON logs and exposes Prometheus metrics at `GET /metrics`. Sentry is optional and covers selected errors/panics plus tracing when configured. HTTP middleware starts one `http.server` span/transaction per request. Some deeper spans exist for PDS writes and Tap/indexer work, but other business and DB work boundaries are metrics-only or untraced. Error events use a narrow context allowlist and redact exception values broadly.

## 11. Desired Behavior

AppView should use Sentry as the single primary external observability platform for errors, logs, tracing, and metrics while continuing to write local structured JSON logs to stdout. Application code should call local observability interfaces rather than direct Sentry SDK APIs outside the observability package and narrowly approved startup/log-handler wiring. Each request or background unit should produce a useful transaction with child spans around meaningful work boundaries when tracing is enabled. Metrics should be emitted through AppView-domain metric methods with Sentry and no-op/test implementations, reusing the existing `craftsky_appview_*` metric names inside the Sentry implementation where Sentry permits. Error events and Sentry-bound logs should provide bounded diagnostic context through sentinel/enum values, while arbitrary raw Go error strings and panic values stay out of Sentry by default. The public Prometheus `/metrics` route, collectors, and dependencies should be removed.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | AppView observability shall consolidate onto Sentry as the primary backend for errors, logs, tracing, and metrics. | Reduces operational complexity for a single developer. | Prompt | AC-001, AC-002, AC-003, AC-004 |
| BR-002 | Business | Must | The implementation shall preserve future flexibility through separate narrow local interfaces for metrics, tracing, errors, and logs, with metrics exposed as AppView-domain methods. | Avoids hard-locking business code to Sentry, especially while metrics are newer/beta, without creating a generic framework. | Prompt, Sentry docs review, grilling answers | AC-005 |
| BR-003 | Business | Must | AppView telemetry shall remain privacy-preserving and shall not expose credentials, private data, raw user identifiers, or record content to Sentry. | Observability must not conflict with AppView/PDS token and privacy rules. | AGENTS.md, codebase findings | AC-006, AC-007 |
| FR-001 | Functional | Must | The system shall initialize Sentry early in AppView startup when configured, including environment, release, error/panic capture, optional logs, optional tracing, optional metrics support, and shutdown flushing. | Sentry Go setup expects early initialization and flushing before process exit, while logs/tracing/metrics should be independently enabled. | Sentry Go setup docs, codebase findings, grilling answers | AC-001 |
| FR-002 | Functional | Must | The system shall send actionable AppView errors and recovered panics to Sentry events. | Errors and panics are core Sentry event use cases. | Sentry usage docs, prompt | AC-002, AC-007 |
| FR-003 | Functional | Must | The system shall keep local structured JSON logs on stdout and shall send safe filtered AppView structured logs to Sentry logs only when logs are explicitly enabled. | Docker/dev workflows need local logs, while Sentry logs should be searchable beside errors and traces when deliberately enabled. | Prompt, Sentry logs docs, grilling answers | AC-003, AC-014 |
| FR-004 | Functional | Must | The system shall emit AppView operational metrics to Sentry Application Metrics through AppView-domain metric methods, reusing the existing `craftsky_appview_*` metric names inside the Sentry implementation where Sentry permits. | Metrics must move to Sentry while preserving provider flexibility and continuity with the first observability pass without leaking Prometheus collector/label concepts to callers. | Prompt, Sentry metrics docs, user answer, grilling answers | AC-004, AC-005 |
| FR-005 | Functional | Must | The system shall provide no-op runtime implementations and in-memory test implementations for disabled Sentry, including metrics support, without adding a replacement local metrics endpoint. | AppView must remain runnable without Sentry and tests must verify behavior deterministically. | Codebase pattern, prompt, grilling answers | AC-005, AC-014 |
| FR-006 | Functional | Must | The system shall avoid direct Sentry SDK calls and `sentry-go` imports from business logic, storage, indexer, and route handler code except through approved observability interfaces; direct imports shall be limited to `appview/internal/observability` and narrowly approved startup/log-handler wiring where unavoidable. | Maintains abstraction, testability, and consistent privacy rules. | Prompt, codebase findings, grilling answers | AC-005 |
| FR-007 | Functional | Must | The system shall create Sentry transactions for top-level HTTP requests and background processing loops, using bounded names such as route patterns and operation names. | Sentry tracing models requests/work as transactions and spans. | Sentry tracing docs, user span note | AC-008 |
| FR-008 | Functional | Must | The system shall create child spans at business-logic and general-work boundaries inside HTTP requests and background work, including auth/session, DB, PDS/OAuth, Tap/indexer, search, feed, profile, and write-handler operations, using the tracing interface where spans are created inside handlers, storage, or indexers. | One request span is insufficient for diagnosing where time/failure occurs, and the cleanest boundary may be inside the component doing the work. | User span note, user answer, grilling answers | AC-009, AC-010 |
| FR-009 | Functional | Must | The system shall create manual DB operation spans for selected named AppView storage operations using bounded operation names, bounded result classes, and no SQL text, query text, exact row counts, or user content. | DB time must be visible inside traces without exposing sensitive data or high-cardinality values, and AppView currently uses `pgxpool` rather than `database/sql`. | User span note, codebase findings, Sentry SQL docs review, grilling answers | AC-009, AC-011 |
| FR-010 | Functional | Must | The system shall create PDS/OAuth spans for session resume, request build, PDS request/response, blob upload, and existing PDS write operations. | PDS/OAuth failures are critical AppView operational paths. | Prior observability requirements, codebase findings | AC-010 |
| FR-011 | Functional | Must | The system shall create Tap and indexer spans for receive, decode/classify, handler execution, ack, reconnect, and panic/error boundaries where applicable. | Firehose ingestion is background critical path work. | Codebase findings, user span note | AC-010 |
| FR-012 | Functional | Must | The system shall enrich non-panic Sentry error events with safe bounded diagnostic fields, including component, operation, route pattern where applicable, exact HTTP status and status class where applicable, failure stage, error category, result, and sentinel error code where known. | Improves diagnosis without exposing raw values. | Prompt, grilling answers | AC-012 |
| FR-013 | Functional | Must | The system shall keep panic events highly redacted while preserving safe context such as component, route pattern or operation, result, and recovered value type. | Panic values may contain arbitrary sensitive data. | Prompt, privacy constraints | AC-007 |
| FR-014 | Functional | Should | The system should define sentinel/enum error codes for common AppView categories, including auth/session, validation, rate limit, not found, forbidden, timeout, network, PDS server, Tap/indexer, DB, and unexpected failures, and should use those codes as Sentry exception values/messages instead of arbitrary raw Go error strings by default. | More useful grouping and filtering than a generic redacted error while avoiding accidental export of sensitive wrapped error text. | Prompt, grilling answers | AC-012 |
| FR-015 | Functional | Must | The system shall keep Sentry disabled by default when no DSN is configured and shall continue running without failed Sentry initialization causing AppView startup failure unless configuration is invalid; with `SENTRY_DSN` alone, only errors and recovered panics shall be sent by default. | Local/dev should not require Sentry; optional backend failures should not break AppView unnecessarily; logs/tracing/metrics need explicit volume control. | Codebase pattern, grilling answers | AC-014 |
| FR-016 | Functional | Must | The system shall preserve existing AppView API responses, route behavior, auth enforcement, Tap behavior, and PDS write behavior apart from observability side effects. | Observability must not change product behavior. | AGENTS.md, codebase findings | AC-015 |
| FR-017 | Functional | Must | The implementation shall remove the public Prometheus `/metrics` route, Prometheus collectors, and Prometheus dependencies once Sentry/no-op metrics implementations replace them. | Aligns with single-platform simplification and the user's decision to remove `/metrics` rather than hide a second metrics system. | Prompt, current codebase findings, user answer, grilling answers | AC-013 |
| FR-018 | Functional | Should | The system should distinguish Sentry logs from breadcrumbs and should only add breadcrumbs where they are low-cardinality and clearly useful context for future events. | Logs and breadcrumbs serve different purposes; breadcrumbs should not become a second uncontrolled telemetry stream. | Sentry usage/logs docs, grilling answers | AC-003, AC-006 |
| NFR-001 | Non-functional | Must | Telemetry fields shall be low-cardinality and bounded by allowlists or enum normalization; invalid metric, log, and span attributes shall normalize safely at runtime and fail in validation-focused tests. | Prevents Sentry noise, cost, and unqueryable dimensions without breaking production for telemetry mistakes. | Prior observability requirements, Sentry docs review, grilling answers | AC-006, AC-012, AC-016 |
| NFR-002 | Non-functional | Must | Sentry logs, tracing, and metrics shall have independent enablement, and tracing/metrics shall have configurable sampling or volume controls with safe disabled defaults. | Sentry tracing docs call out sample-rate configuration and high-throughput performance testing; logs and metrics can also be high volume. | Sentry tracing docs, grilling answers | AC-001, AC-014 |
| NFR-003 | Non-functional | Should | Added observability should introduce minimal overhead when Sentry is disabled. | Local/dev and tests should stay fast and simple. | Codebase pattern | AC-014 |
| NFR-004 | Non-functional | Should | The requirements and follow-on tests should treat Sentry Application Metrics beta status as a design risk mitigated by the metrics interface. | Avoids overcommitting to a new backend API. | Sentry metrics docs review | AC-005 |
| NFR-005 | Non-functional | Should | The implementation should prefer Sentry's `slog` integration or an equivalent local handler composition that preserves local logs while sending only safe, filtered structured logs to Sentry. | AppView already uses `slog`, and Sentry provides a supported `slog` integration with filtering hooks. | Sentry slog docs, codebase findings | AC-003, AC-006 |
| NFR-006 | Non-functional | Must | Tap/indexer tracing shall have independent or operation-aware volume control, and success-path Tap spans shall be sampled while errors and panics remain visible. | Tap/indexer work can be much higher volume than HTTP traffic. | User span note, Sentry tracing docs, grilling answers | AC-010, AC-014 |
| RULE-001 | Business rule | Must | Sentry must be initialized with default PII collection disabled. | Prevents accidental IP/header/user collection. | Sentry setup docs, privacy constraints | AC-006 |
| RULE-002 | Business rule | Must | Raw request/response bodies, tokens, OAuth material, PDS credentials, DPoP keys, raw identity values, record payloads, raw paths containing identifiers, and private AppView data must not be emitted as Sentry event data, log attributes, span data, or metric tags. | Maintains security and privacy boundaries. | AGENTS.md, prompt | AC-006 |
| RULE-003 | Business rule | Must | Panic captured values shall not be emitted verbatim to Sentry. | Panic payloads are arbitrary and unsafe. | Prompt | AC-007 |
| RULE-004 | Business rule | Must | Sentry metric tags and log/span attributes shall not include unbounded IDs such as `run_id` unless explicitly justified for non-metric correlation and excluded from metric dimensions. | Prevents high-cardinality telemetry. | Prior observability requirements, Sentry docs review | AC-006 |
| RULE-005 | Business rule | Must | Sentry traces shall use route patterns for HTTP transaction/span names, not raw request paths. | Avoids leaking path identifiers and controls cardinality. | API conventions, prior observability requirements | AC-008 |
| RULE-006 | Business rule | Must | Sentry-bound events and logs shall not include arbitrary raw Go error strings by default; only explicitly safe sentinel errors/messages may be exported as event values, log attributes, or breadcrumbs. | Wrapped errors can contain upstream response text, identifiers, SQL details, URLs, or user-provided values. | Grilling answers, privacy constraints | AC-006, AC-012 |
| RULE-007 | Business rule | Must | Local stdout logs may preserve raw error strings only where existing local logging already considers them acceptable, but Sentry-bound log data must use the safe classifier. | Keeps local debugging useful without exporting raw details to a third party. | Grilling answers | AC-003, AC-006 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, NFR-002 | Given Sentry configuration is present, when AppView starts and stops, then Sentry is initialized with environment/release/error capture plus optional logs/tracing/metrics according to explicit config and flushes on shutdown. |
| AC-002 | BR-001, FR-002 | Given an actionable non-panic AppView error occurs, when Sentry is configured, then one Sentry error event is captured with safe diagnostic context. |
| AC-003 | BR-001, FR-003, FR-018, RULE-007 | Given AppView emits structured logs, when Sentry logs are disabled or no DSN is configured, then local JSON stdout logs remain available; when Sentry logs are explicitly enabled, then safe filtered log entries are sent to Sentry logs with bounded attributes. |
| AC-004 | BR-001, FR-004 | Given representative HTTP, DB, PDS, and Tap/indexer operations occur, when Sentry metrics are enabled, then counters, gauges, or distribution metrics are emitted to Sentry through AppView-domain metric methods using bounded names and tags, reusing the existing `craftsky_appview_*` metric names where Sentry permits. |
| AC-005 | BR-002, FR-004, FR-005, FR-006, NFR-004 | Given business logic, handlers, storage, and indexer code are inspected, when they record metrics/errors/logs/spans, then they use separate local observability interfaces rather than direct Sentry SDK calls, and `sentry-go` imports are limited to approved observability/startup/log-handler wiring. |
| AC-006 | BR-003, NFR-001, RULE-001, RULE-002, RULE-004, RULE-006, RULE-007 | Given telemetry is emitted to Sentry, when payloads are inspected via test transport or interface fakes, then no forbidden raw secrets, identifiers, content, request/response bodies, arbitrary raw Go error strings, or high-cardinality metric/log/span attributes are present. |
| AC-007 | BR-003, FR-002, FR-013, RULE-003 | Given a panic is recovered in HTTP or Tap/indexer work, when Sentry captures the event, then the event includes only safe bounded context and recovered value type, not the recovered value string. |
| AC-008 | FR-007, RULE-005 | Given HTTP requests and background loops run with tracing enabled, when traces are inspected, then top-level transactions use bounded route-pattern or operation names and include safe result/status context. |
| AC-009 | FR-008, FR-009 | Given representative auth/session, search, feed, profile, write-handler, and DB work occurs during HTTP requests, when tracing is enabled, then the request transaction contains child spans for those work boundaries with bounded operation names and attributes created through the tracing interface. |
| AC-010 | FR-008, FR-010, FR-011, NFR-006 | Given PDS/OAuth and Tap/indexer workflows run, when tracing is enabled, then child spans exist for meaningful external-call, handler, ack, reconnect, and error/panic boundaries where applicable, with Tap/indexer success-path tracing subject to sampling or operation-aware volume control. |
| AC-011 | FR-009 | Given selected named DB storage work such as `search.posts`, `feed.list`, `profile.get`, `session.lookup`, or write-side mutations executes, when tracing is enabled, then manual DB spans report duration/result class using bounded operation names and omit SQL/query/user content and exact row counts without requiring Sentry `sentrysql` or `database/sql` migration. |
| AC-012 | FR-012, FR-014, NFR-001, RULE-006 | Given non-panic errors from known categories occur, when captured or logged in Sentry, then they include bounded sentinel/enum category/code/stage fields and safe event values/messages that distinguish expected categories without arbitrary raw error details. |
| AC-013 | FR-017 | Given Sentry/no-op metrics implementations are in place, when AppView routes, dependencies, collectors, tests, and docs are reviewed, then the public Prometheus `/metrics` route, Prometheus collectors/dependencies, and primary runtime documentation are removed. |
| AC-014 | FR-005, FR-015, NFR-002, NFR-003, NFR-006 | Given no Sentry DSN is configured, when AppView starts and representative requests/background work run, then AppView remains functional and observability calls use no-op/test-safe behavior without external export or replacement metrics endpoint; given only `SENTRY_DSN` is configured, then only errors and recovered panics are externally exported by default. |
| AC-015 | FR-016 | Given the observability consolidation is implemented, when existing AppView tests run, then API behavior, auth behavior, Tap behavior, and PDS write behavior remain compatible with the prior implementation. |
| AC-016 | NFR-001 | Given invalid or unbounded telemetry attribute values are supplied to metrics or tracing interfaces, when running in production/runtime mode, then they normalize to bounded fallback values; when running validation-focused tests, then the invalid values are reported as test failures. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Sentry DSN absent in local/dev | AppView starts normally; telemetry interfaces no-op or keep local behavior without external export. | FR-005, FR-015 |
| EC-002 | Sentry SDK initialization returns an error | AppView handles it according to config policy, logs safe startup context, and does not panic unless config is invalid. | FR-001, FR-015 |
| EC-003 | Panic value contains a token, DID, handle, or record payload | Sentry event redacts the value and emits only the recovered type plus safe context. | FR-013, RULE-003 |
| EC-004 | Non-panic error wraps raw upstream response text | Sentry event emits sentinel category/code/stage and omits raw upstream text. | FR-012, RULE-002 |
| EC-005 | Unknown route path contains identifiers | Transaction/log/span context uses `unmatched` or another bounded fallback instead of the raw path. | RULE-005 |
| EC-006 | Unknown metric, log, or span attribute value is supplied | Runtime observability normalizes or maps it to a bounded fallback before export; validation-focused tests fail so the caller can be fixed. | FR-004, FR-008, NFR-001 |
| EC-007 | Sentry metrics API changes or is unsuitable | Metrics interface localizes provider replacement without rewriting business logic. | BR-002, NFR-004 |
| EC-008 | High-throughput Tap ingestion with tracing enabled | Sampling/config controls volume; traces remain bounded and avoid per-record raw IDs/content. | NFR-002, RULE-002 |
| EC-009 | Developer considers `sentrysql` for DB spans | Implementation keeps the existing `pgxpool` DB layer and uses manual bounded DB spans instead. | FR-009, NG-010 |
| EC-010 | Only `SENTRY_DSN` is configured | AppView exports errors and recovered panics only; Sentry logs, tracing, and metrics remain disabled until their explicit flags are enabled. | FR-001, FR-015, NFR-002 |
| EC-011 | DB operation returns many rows | DB span reports a bounded result class or bucket such as `many`, not an exact row count. | FR-009 |
| EC-012 | Local logs include an existing raw error string | The raw string remains local stdout only when acceptable by current logging policy; Sentry-bound logs/events use the safe classifier. | RULE-006, RULE-007 |
| EC-013 | Breadcrumb candidate includes raw identifiers or high-cardinality values | Breadcrumb is omitted or normalized; logs/traces remain the primary implementation target. | FR-018, RULE-002 |

## 15. Data / Persistence Impact

- New fields: None in Postgres.
- Changed fields: None.
- Migration required: No.
- Backwards compatibility: No data migration expected. Existing Prometheus metric names should be reused inside the Sentry metrics implementation where Sentry permits, but `/metrics`, Prometheus collectors, and Prometheus dependencies are removed as runtime contracts.

## 16. UI / API / CLI Impact

- UI: None.
- API: No `/v1/*` or `/oauth/*` contract changes. `GET /metrics` is removed as part of moving metrics to Sentry, and no replacement local metrics endpoint is added.
- CLI: No required CLI behavior changes. Existing ops docs may need updates if they reference `/metrics`.
- Background jobs: Tap consumer and indexer paths gain Sentry logs/metrics/spans but must preserve ack/retry/drop behavior.

## 17. Security / Privacy / Permissions

- Authentication: No changes to AppView auth. The Flutter app continues to hold only the Craftsky session token.
- Authorization: No changes.
- Sensitive data: Sentry must not receive tokens, DPoP keys, OAuth material, raw DIDs, handles, AT-URIs, CIDs, rkeys, device/session IDs, emails, record payloads, request/response bodies, private AppView data, or arbitrary raw Go error strings.
- Abuse cases: Telemetry must not create a path for reconstructing user behavior or private credentials through logs, spans, metrics, or error events. Metric/log/span dimensions must be bounded to prevent cardinality abuse.

## 18. Observability

- Events: Sentry captures panics and actionable non-panic errors. Panic values stay redacted; non-panic events include bounded diagnostic enum/sentinel fields and safe event values/messages rather than arbitrary raw Go error strings.
- Logs: AppView local structured JSON logs remain on stdout. Safe filtered structured logs are sent to Sentry logs only when explicitly enabled. Sentry-bound logs use the same safe classifier as error events.
- Breadcrumbs: Breadcrumbs are optional and secondary in this slice; add them only for low-cardinality context that is clearly useful for future error events.
- Metrics: AppView metrics are emitted through AppView-domain metric methods with Sentry and no-op/test implementations; Prometheus `/metrics`, collectors, and dependencies are removed. Metric names should reuse the current `craftsky_appview_*` names inside the Sentry implementation where Sentry permits.
- Traces: HTTP and background transactions contain child spans for business and work boundaries, including auth/session, DB, PDS/OAuth, Tap/indexer, search, feed, profile, and write-handler work. DB spans are manual AppView spans for named storage operations rather than every SQL call or Sentry `sentrysql` spans. Tap/indexer tracing has independent or operation-aware volume control.
- Alerts: No alert rules in this slice.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Sentry Application Metrics are documented as new/beta. | Metrics API or product behavior may change, or the feature may not fit AppView needs. | Put metrics behind an interface with no-op/test and Sentry implementations. |
| RISK-002 | Relaxed redaction leaks sensitive data. | Credentials, identifiers, or content could reach a third-party service. | Keep allowlists, sentinel enums, panic redaction, and tests that inspect exported payloads. |
| RISK-003 | More spans create noise or overhead. | Sentry costs/noise increase and traces become harder to read. | Use sampling, bounded span names, and only span meaningful business/work boundaries. |
| RISK-004 | Removing `/metrics` breaks existing local ops habits. | Existing docs or scripts that curl `/metrics` stop working. | Update docs and tests to make Sentry metrics the documented metrics path. |
| RISK-005 | Interface abstraction becomes too broad. | Code complexity increases instead of decreasing. | Keep interfaces narrow and AppView-specific; avoid a generic observability framework. |
| RISK-006 | Sentry SQL instrumentation is tempting but does not match AppView's `pgxpool` architecture. | Adding it could force unnecessary DB-layer churn or parallel DB access patterns. | Keep manual DB spans through the local observability interface; revisit only if AppView intentionally adopts `database/sql` or a pgx-compatible Sentry integration appears. |
| RISK-007 | DSN-only behavior sends fewer signals than expected. | A developer may expect logs/traces/metrics to appear after setting only `SENTRY_DSN`. | Document that DSN-only means errors and recovered panics; require explicit flags for logs, tracing, and metrics. |
| RISK-008 | No local metrics endpoint makes local performance investigation less convenient. | Developers lose quick `curl /metrics` visibility in local/dev. | Preserve local stdout logs, focused tests, and in-memory metric assertions; rely on Sentry metrics only when explicitly enabled. |
| RISK-009 | Runtime normalization could hide telemetry mistakes. | Invalid attributes might be silently grouped under `unknown` or `other` in production. | Make in-memory/validation tests fail on invalid metric/log/span values. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Sentry Application Metrics are acceptable to use despite beta/new status. | The implementation may need to keep Prometheus longer or choose another metrics backend. |
| ASM-002 | The current `github.com/getsentry/sentry-go` dependency is sufficient or can be upgraded within normal dependency-management scope. | Requirements may need an explicit dependency-upgrade task. |
| ASM-003 | Hosted Sentry project configuration, dashboarding, alerting, and notification setup are out of scope for this requirements pass. | Requirements would need additional deployment/operator acceptance criteria. |
| ASM-004 | It is acceptable for non-panic errors to expose bounded categories and sentinel codes, but not raw error strings from untrusted sources. | Redaction rules would need to become stricter or more permissive. |
| ASM-005 | Sentry permits metric names close enough to the current `craftsky_appview_*` Prometheus names to preserve continuity. | Metric names may need minimal Sentry-specific normalization during implementation. |
| ASM-006 | Local stdout logs are an acceptable place to preserve existing raw error strings where current logging already does so. | Local logging policy would need to become stricter and more Sentry-like. |
| ASM-007 | Separate explicit flags for logs, tracing, and metrics are acceptable operational overhead. | Startup/config requirements would need to simplify enablement or choose different defaults. |

## 21. Open Questions

None identified.

## 22. Review Status

Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer:
Date: 2026-07-02
Notes: Medium risk because the change touches cross-cutting observability, privacy, provider architecture, explicit feature enablement, and removal of `/metrics` plus Prometheus internals, and because Sentry Application Metrics are documented as new/beta. No blocking questions are identified for acceptance-test design.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001, BR-002, BR-003, FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, FR-011, FR-012, FR-013, FR-015, FR-016, FR-017, FR-018, NFR-001, NFR-002, NFR-006, RULE-001, RULE-002, RULE-003, RULE-004, RULE-005, RULE-006, RULE-007
- Suggested test levels: unit tests for interface boundaries, Sentry import boundaries, safe field normalization, metric/log/span payloads, sentinel error codes, raw-error exclusion from Sentry, local-log preservation, `/metrics` route and Prometheus dependency removal, explicit enablement defaults, and panic redaction; integration tests with Sentry test transport/fakes for HTTP, PDS/OAuth, DB, and Tap/indexer paths; regression tests for existing API/auth/Tap/PDS behavior; manual review for docs and Sentry metric-name continuity.
- Blocking open questions: None.
