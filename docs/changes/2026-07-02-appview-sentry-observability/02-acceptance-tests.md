# Acceptance Test Specification: AppView Sentry Observability Consolidation

## 1. Test Strategy

This change is medium risk because it replaces Prometheus-first AppView observability with Sentry-backed errors, logs, traces, and metrics, while also tightening privacy boundaries and removing the public `/metrics` route. The test strategy should be test-first at three levels:

- Unit tests for config defaults, interface boundaries, safe normalization, sentinel error classification, panic redaction, Sentry log filtering, metric method behavior, span attributes, and import/dependency guards.
- Integration tests using `httptest`, existing AppView fakes, and Sentry test transports or in-memory observers for HTTP, storage, PDS/OAuth, Tap/indexer, and startup wiring behavior.
- Regression tests around unchanged `/v1/*` API behavior, auth/device enforcement, Tap ack/retry behavior, PDS write behavior, local stdout logs, and removal of the old Prometheus route/dependencies.

Manual checks are limited to documentation review and optional hosted Sentry verification because deterministic behavior should be covered with local fakes and Sentry test transports.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-003, AC-004 | AT-001, AT-002, AT-003, IT-001, IT-002, IT-003 | Acceptance / Integration | Yes |
| BR-002 | AC-005 | UT-002, UT-006, UT-007, IT-003, IT-004 | Unit / Integration | Yes |
| BR-003 | AC-006, AC-007 | AT-003, UT-003, UT-004, UT-005, IT-002, IT-005, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-001 | AC-001 | AT-001, UT-001, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-002, AC-007 | AT-003, UT-004, UT-005, IT-002, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-003, AC-014 | AT-002, UT-009, REG-004 | Acceptance / Unit / Regression | Yes |
| FR-004 | AC-004, AC-005 | AT-003, UT-006, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-005, AC-014 | AT-002, UT-001, UT-006, IT-001, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| FR-006 | AC-005 | UT-002 | Unit | Yes |
| FR-007 | AC-008 | AT-003, UT-007, IT-002, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-009, AC-010 | AT-003, IT-002, IT-004, IT-005, IT-006 | Acceptance / Integration | Yes |
| FR-009 | AC-009, AC-011 | UT-008, IT-004 | Unit / Integration | Yes |
| FR-010 | AC-010 | IT-005 | Integration | Yes |
| FR-011 | AC-010 | IT-006, UT-010, REG-003 | Integration / Unit / Regression | Yes |
| FR-012 | AC-012 | UT-003, UT-005, IT-002, IT-005, IT-006 | Unit / Integration | Yes |
| FR-013 | AC-007 | AT-003, UT-004, IT-002, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-012 | UT-005, IT-002, IT-005, IT-006 | Unit / Integration | Yes |
| FR-015 | AC-014 | AT-001, AT-002, UT-001, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-015 | AT-004, REG-002, REG-003, REG-005 | Acceptance / Regression | Yes |
| FR-017 | AC-013 | AT-002, UT-011, IT-007, REG-001, MAN-001 | Acceptance / Unit / Integration / Regression / Manual | Yes, plus manual docs check |
| FR-018 | AC-003, AC-006 | UT-009, MAN-002 | Unit / Manual | Yes, plus manual docs check |
| NFR-001 | AC-006, AC-012, AC-016 | UT-003, UT-005, UT-007, UT-008, IT-003 | Unit / Integration | Yes |
| NFR-002 | AC-001, AC-014 | AT-001, AT-002, UT-001, UT-010 | Acceptance / Unit | Yes |
| NFR-003 | AC-014 | AT-002, UT-006, REG-006 | Acceptance / Unit / Regression | Yes |
| NFR-004 | AC-005 | UT-006, GAP-001 | Unit | Yes, with provider-risk gap |
| NFR-005 | AC-003, AC-006 | UT-009 | Unit | Yes |
| NFR-006 | AC-010, AC-014 | UT-010, IT-006 | Unit / Integration | Yes |
| RULE-001 | AC-006 | UT-001, IT-001 | Unit / Integration | Yes |
| RULE-002 | AC-006 | AT-003, UT-003, UT-007, IT-002, IT-003, IT-004, IT-005, IT-006 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-007 | UT-004, IT-002, IT-006 | Unit / Integration | Yes |
| RULE-004 | AC-006 | UT-003, UT-006, UT-007, IT-003 | Unit / Integration | Yes |
| RULE-005 | AC-008 | UT-007, IT-002 | Unit / Integration | Yes |
| RULE-006 | AC-006, AC-012 | UT-003, UT-005, IT-002, IT-005, IT-006 | Unit / Integration | Yes |
| RULE-007 | AC-003, AC-006 | UT-009, REG-004 | Unit / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Explicit Sentry Startup Gates

Requirement IDs: BR-001, FR-001, FR-015, NFR-002
Acceptance Criteria: AC-001, AC-014
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/app/config_test.go`, `appview/internal/app/deps_test.go`, `appview/internal/observability/sentry_test.go`

```gherkin
Feature: Sentry observability startup
  Scenario: AppView enables only explicitly configured Sentry pillars
    Given AppView is configured with SENTRY_DSN, release, environment, and explicit logs/tracing/metrics flags
    When AppView dependencies are initialized and later shut down
    Then Sentry is initialized with default PII disabled
    And errors and recovered panics are enabled when a DSN exists
    And logs, tracing, and metrics follow only their explicit enablement flags
    And shutdown flushes the configured Sentry client
```

### AT-002: Disabled Sentry Remains Functional Without Metrics Endpoint

Requirement IDs: BR-001, FR-003, FR-005, FR-015, FR-017, NFR-002, NFR-003
Acceptance Criteria: AC-003, AC-013, AC-014
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/routes_test.go`, `appview/internal/observability/metrics_test.go`, `appview/internal/middleware/logging_test.go`

```gherkin
Feature: Local/dev observability defaults
  Scenario: AppView runs without Sentry or a replacement metrics endpoint
    Given no SENTRY_DSN is configured
    When AppView starts and serves representative HTTP and background work
    Then observability calls use no-op or test-safe implementations without external export
    And local structured JSON stdout logs remain available
    And GET /metrics no longer exposes Prometheus or replacement metrics output
```

### AT-003: Representative Work Emits Safe Sentry Telemetry

Requirement IDs: BR-001, BR-003, FR-002, FR-004, FR-007, FR-008, FR-012, FR-013, RULE-002, RULE-005, RULE-006
Acceptance Criteria: AC-002, AC-004, AC-006, AC-007, AC-008, AC-009, AC-012
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/metrics_test.go`, `appview/internal/observability/sentry_test.go`, `appview/internal/api/*_test.go`

```gherkin
Feature: Safe Sentry telemetry for AppView work
  Scenario: HTTP request work emits bounded events, metrics, and spans
    Given Sentry errors, tracing, logs, and metrics are enabled with test transports
    And an HTTP request includes raw DIDs, rkeys, query strings, request bodies, and token-like values
    When the request performs auth/session, handler, DB, and error-producing work
    Then Sentry receives only bounded route-pattern transaction names and child spans
    And Sentry metrics use AppView-domain methods with bounded tags
    And non-panic errors use safe category/code/stage fields
    And no Sentry event, log, span, or metric includes forbidden raw values
```

### AT-004: Observability Does Not Change Product Behavior

Requirement IDs: FR-016
Acceptance Criteria: AC-015
Priority: Must
Level: Acceptance
Automation Target: Existing AppView suites under `appview/internal/api`, `appview/internal/auth`, `appview/internal/routes`, `appview/internal/tap`

```gherkin
Feature: Product behavior preservation
  Scenario: Existing AppView behavior is unchanged after observability consolidation
    Given the Sentry consolidation is implemented
    When the existing focused AppView Go test suite runs
    Then /v1 API responses keep their existing JSON contracts
    And auth/device enforcement remains compatible
    And PDS write behavior remains compatible
    And Tap consumer ack, retry, drop, and indexing behavior remains compatible
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-015, NFR-002, RULE-001 | AC-001, AC-006, AC-014 | Verify config parsing for `SENTRY_DSN`, release, explicit log/tracing/metrics flags, trace sample rate validation, metrics volume controls, Tap trace volume controls, and DSN-only behavior. | Temp `.env` files covering no DSN, DSN only, each explicit flag, invalid sample rates, prod/dev defaults. | No DSN disables external export; DSN-only enables errors/panics only; logs/tracing/metrics require explicit flags; sample/volume controls validate; default PII remains disabled in observer options. | `appview/internal/app/config_test.go`, `appview/internal/observability/sentry_test.go` |
| UT-002 | BR-002, FR-006 | AC-005 | Guard direct `sentry-go` imports to approved packages only. | Repository file scan from a Go test or script-backed test. | Direct imports appear only in `appview/internal/observability` and narrowly approved startup/log-handler wiring/tests; business logic, storage, route handlers, and indexers use local interfaces. | `appview/internal/observability/import_boundary_test.go` |
| UT-003 | BR-003, FR-012, NFR-001, RULE-002, RULE-004, RULE-006 | AC-006, AC-012, AC-016 | Validate event/log/span/metric attribute sanitization and runtime normalization. | Context containing tokens, DIDs, handles, AT-URIs, CIDs, rkeys, emails, raw paths, request bodies, raw error strings, `run_id` as metric dimension, and invalid enum values. | Forbidden fields are dropped or normalized; metric dimensions exclude high-cardinality values; production mode maps invalid values to `unknown`/`other`; validation test implementation reports invalid values. | `appview/internal/observability/redaction_test.go`, `appview/internal/observability/validation_test.go` |
| UT-004 | BR-003, FR-002, FR-013, RULE-003 | AC-007 | Verify recovered panic capture never exports recovered value strings. | Panic values containing token-like strings, raw DIDs, handles, and record payload text. | Sentry event includes safe component/operation/route/result and recovered type only; recovered value string is absent. | `appview/internal/observability/sentry_test.go`, `appview/internal/middleware/recovery_test.go`, `appview/internal/tap/consumer_test.go` |
| UT-005 | FR-012, FR-014, NFR-001, RULE-006 | AC-012, AC-016 | Verify sentinel/enum error classifier covers common AppView categories and does not fall back to raw `err.Error()` for Sentry-bound output. | Auth/session, validation, rate limit, not found, forbidden, timeout, network, PDS server, Tap/indexer, DB, and unexpected errors with wrapped raw details. | Classifier returns bounded category/code/stage/result; Sentry event values/messages and Sentry-bound log attrs use safe sentinels; raw wrapped details are absent. | `appview/internal/observability/error_classifier_test.go` |
| UT-006 | BR-002, FR-004, FR-005, NFR-003, NFR-004, RULE-004 | AC-004, AC-005, AC-014 | Verify metrics are emitted through AppView-domain methods with no-op, in-memory, and Sentry implementations. | Representative HTTP, DB, PDS, Tap/indexer metric calls with bounded and unbounded tags. | Callers do not use Prometheus labels/collectors; disabled metrics are no-op; in-memory metrics record deterministic calls; Sentry implementation maps to permitted `craftsky_appview_*` names and bounded tags. | `appview/internal/observability/metrics_test.go` |
| UT-007 | FR-007, FR-008, NFR-001, RULE-002, RULE-004, RULE-005 | AC-006, AC-008, AC-009, AC-016 | Verify tracing interface enforces bounded transaction/span names and safe attributes. | HTTP route patterns, unknown routes, raw paths, operation names, result/status values, user identifiers, token-like attributes. | Transactions use method plus route pattern or bounded operation names; unknown paths normalize to bounded fallback; child spans record component/operation/result/status only; forbidden values are absent. | `appview/internal/observability/sentry_test.go`, `appview/internal/observability/route_test.go` |
| UT-008 | FR-009, NFR-001 | AC-011, AC-016 | Verify DB operation span helpers use named storage operations and bounded result classes. | `search.posts`, `feed.list`, `profile.get`, `session.lookup`, write mutation operations with row counts and SQL/query text candidates. | DB spans include bounded operation/result class/duration; exact row counts, SQL text, query text, and user content are absent; invalid operation names fail validation tests. | `appview/internal/observability/db_test.go`, selected store tests |
| UT-009 | FR-003, FR-018, NFR-005, RULE-007 | AC-003, AC-006 | Verify local stdout logs remain separate from Sentry logs and use safe filtering for Sentry-bound attrs. | Structured log records containing raw local error strings, route patterns, request IDs, headers, bodies, and safe classifier fields. | Local logs preserve existing allowed behavior; Sentry logs are emitted only when enabled; Sentry-bound attrs use bounded classifier output and omit raw bodies/secrets/identifiers; breadcrumbs are absent or low-cardinality only. | `appview/internal/middleware/logging_test.go`, `appview/internal/observability/logs_test.go` |
| UT-010 | FR-011, NFR-006 | AC-010, AC-014 | Verify Tap/indexer trace volume controls and sampling decisions. | Tap success events, Tap errors, indexer panics, reconnects, enabled/disabled tracing, sample rates or operation-aware config. | Success-path Tap spans are sampled/configurable; errors and panics remain captured; disabled tracing produces no export; no per-record identifiers are emitted. | `appview/internal/tap/consumer_test.go`, `appview/internal/observability/tap_test.go` |
| UT-011 | FR-017 | AC-013 | Guard Prometheus removal in code and dependency metadata. | `go.mod`, `go.sum`, route registration, observability package files, README snippets. | Prometheus collectors and dependencies are absent; `MetricsHandler`/Prometheus exposition helpers are removed or test-only; no production `/metrics` handler is registered. | `appview/internal/observability/prometheus_removal_test.go` or focused repository guard |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-001, FR-005, FR-015, RULE-001 | AC-001, AC-014 | Verify AppView dependency wiring initializes and flushes the configured Sentry observer safely. | Build `app.Deps` with Sentry test transport, explicit flags, and no real network transport. | Initialize dependencies and call shutdown/flush path. | Observer has expected enabled pillars, default PII disabled, no startup failure for optional Sentry export, and flush hook is invoked. | `appview/internal/app/deps_test.go`, `appview/internal/observability/sentry_test.go` |
| IT-002 | BR-001, BR-003, FR-002, FR-007, FR-008, FR-012, FR-013, RULE-002, RULE-005, RULE-006 | AC-002, AC-006, AC-007, AC-008, AC-009, AC-012 | Verify HTTP middleware and representative handlers export safe Sentry events and route-pattern traces. | `httptest` server/mux, Sentry mock transport, tracing enabled, handlers for success, 5xx, expected 4xx, and panic paths. | Serve requests containing raw identifiers, query strings, bodies, and token-like values. | Only actionable non-panic errors and recovered panics are captured; transactions use route patterns; child spans exist for handler/auth/DB where instrumented; raw forbidden values are absent. | `appview/internal/middleware/metrics_test.go`, `appview/internal/middleware/recovery_test.go`, selected `appview/internal/api/*_test.go` |
| IT-003 | BR-001, BR-002, FR-004, NFR-001, RULE-002, RULE-004 | AC-004, AC-005, AC-006, AC-016 | Verify Sentry metrics cover representative HTTP, DB, PDS, and Tap/indexer operations through AppView-domain methods. | Sentry or in-memory metric implementation with validation enabled. | Exercise representative operation metric calls. | Metrics use bounded names/tags, reuse `craftsky_appview_*` names where permitted, avoid Prometheus label concepts in callers, and reject/normalize invalid dimensions. | `appview/internal/observability/metrics_test.go`, `appview/internal/api/observability_pds_test.go`, `appview/internal/tap/consumer_test.go` |
| IT-004 | BR-002, FR-008, FR-009, RULE-002 | AC-009, AC-011 | Verify selected storage operations create manual DB spans without SQL/query/user content. | Test database or store fakes for search, feed, profile, session lookup, and write-side storage operations; tracing enabled with test transport. | Execute selected store methods. | Trace contains named DB/storage spans with bounded result classes and no SQL text, exact row counts, search text, DIDs, handles, AT-URIs, CIDs, rkeys, or record payloads. | `appview/internal/api/search_store_test.go`, `profile_store_test.go`, `timeline_store_test.go`, `appview/internal/auth/store_test.go` |
| IT-005 | BR-003, FR-008, FR-010, FR-012, RULE-002, RULE-006 | AC-006, AC-010, AC-012 | Verify PDS/OAuth and blob upload paths produce safe spans/events/log attrs. | Fake PDS client/factory, Sentry mock transport, tracing/logging enabled. | Run session resume, request build, PDS request/response, write operations, and blob upload success/error cases. | Child spans exist for meaningful PDS/OAuth boundaries; expected classified failures use sentinel fields; unexpected failures capture safe Sentry events; tokens, DIDs, handles, record payloads, and raw upstream text are absent. | `appview/internal/observability/pds_wrapper_test.go`, `appview/internal/api/blob_test.go`, `appview/internal/auth/*_test.go` |
| IT-006 | FR-008, FR-011, FR-012, FR-013, NFR-006, RULE-002, RULE-003, RULE-006 | AC-006, AC-007, AC-010, AC-012, AC-014 | Verify Tap consumer and indexer tracing, errors, panic capture, ack/reconnect spans, and sampling controls. | Fake WebSocket/Tap events, dispatcher/indexer fakes, Sentry mock transport, tracing enabled and disabled cases. | Process receive/decode/classify/handle/ack/reconnect success, error, and panic scenarios. | Top-level/background transactions and child spans cover configured boundaries; success sampling is honored; errors/panics remain visible; no raw repo identifiers or record content is emitted. | `appview/internal/tap/consumer_test.go`, `appview/internal/index/*_test.go` |
| IT-007 | FR-017 | AC-013 | Verify public `/metrics` route is removed and no replacement local metrics endpoint exists. | AppView routes registered through `routes.AddRoutes` and server middleware stack. | Request `GET /metrics` with and without v1 auth/device headers. | `/metrics` no longer returns Prometheus text or app metrics; no replacement metrics endpoint is documented or registered; `/v1/*` auth/device behavior remains intact. | `appview/internal/routes/routes_test.go`, `appview/cmd/appview/server_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Removing `/metrics` must not introduce a new public metrics surface or auth bypass. | FR-005, FR-017 | Update existing `/metrics` route/server tests to expect the route is absent and verify no public replacement endpoint is registered. |
| REG-002 | Existing `/v1/*` API responses, camelCase JSON, and error envelopes remain compatible. | FR-016 | Run existing route/API tests and add focused assertions around representative success, validation error, auth error, and server error responses after instrumentation. |
| REG-003 | Tap consumer ack/retry/drop/indexer behavior remains compatible. | FR-011, FR-016 | Run existing Tap consumer/indexer tests after adding spans and error capture; assert ack/retry/drop outcomes are unchanged. |
| REG-004 | Local structured stdout logging remains available for Docker/dev workflows. | FR-003, RULE-007 | Keep existing logging middleware tests and add a case proving Sentry logs disabled does not remove local completion/error logs. |
| REG-005 | PDS write and blob upload behavior remains compatible. | FR-016 | Run existing PDS, blob, post, interaction, and auth PDS client tests after adding PDS/OAuth spans and Sentry classifications. |
| REG-006 | Disabled Sentry path remains low overhead and test-friendly. | NFR-003 | Run focused observability and middleware tests with no DSN and assert no network transport is required, no spans/events/metrics are externally exported, and no nil observer panics occur. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Sentry config matrix | Temp `.env` values for no DSN, DSN only, logs enabled, tracing enabled, metrics enabled, Tap sampling enabled, invalid sample rates, dev/prod environments. | AT-001, AT-002, UT-001, IT-001 |
| TD-002 | Forbidden telemetry values | `secret-token`, OAuth/PDS token-like strings, DPoP-like key text, `did:plc:raw`, raw handles, raw AT-URIs, raw CIDs, rkeys, emails, query strings, request bodies, record payload snippets, raw upstream error text. | AT-003, UT-003, UT-004, UT-005, UT-007, IT-002, IT-003, IT-004, IT-005, IT-006 |
| TD-003 | Safe enum/sentinel values | Components `http`, `auth`, `db`, `pds`, `tap`, `indexer`; operations `search.posts`, `feed.list`, `profile.get`, `session.lookup`, `post.create`; categories `validation`, `auth`, `rate_limited`, `timeout`, `network`, `server`, `unexpected`; results `success`, `error`, `canceled`; status classes `2xx`-`5xx`. | UT-003, UT-005, UT-007, UT-008, IT-002, IT-004, IT-005 |
| TD-004 | Representative HTTP requests | `/v1/posts/did:plc:alice/post1?cursor=secret`, `/v1/search/posts?q=secret`, `/v1/whoami`, `/v1/blobs/images`, `/v1/internal/{did}` with auth/device headers and request bodies. | AT-003, IT-002, IT-007, REG-002 |
| TD-005 | PDS/OAuth failure fixtures | Fake PDS clients returning validation, auth, forbidden, not found, rate limited, timeout, network, PDS server, unexpected, and wrapped upstream-response errors. | UT-005, IT-005, REG-005 |
| TD-006 | Tap/indexer fixtures | Fake Tap events for known Craftsky NSIDs, unknown collections, decode failures, handler errors, handler panics, ack failures, reconnects, and success-path high-volume events. | UT-010, IT-006, REG-003 |
| TD-007 | DB span fixtures | Store calls returning none/one/some/many result classes plus attempted SQL/query/search/user-content attributes. | UT-008, IT-004 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | FR-017 | Documentation no longer presents Prometheus `/metrics` as AppView's runtime metrics path. | Review `appview/README.md`, relevant docs, and recipe comments after implementation. | Prometheus route/collector docs are removed or rewritten for Sentry metrics; no replacement local metrics endpoint is promised. |
| MAN-002 | FR-018 | Logs and breadcrumbs are documented as distinct concepts. | Review observability README/update notes. | Sentry logs are the primary searchable log sink when enabled; breadcrumbs are optional, low-cardinality, and not a required deliverable. |
| MAN-003 | FR-004, NFR-004 | Sentry metric-name continuity is reasonable with the current Sentry metrics API. | In a configured non-production Sentry project, emit representative metrics once and inspect names/tags. | Names are as close as Sentry permits to `craftsky_appview_*`; any required normalization is documented. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Hosted Sentry Application Metrics behavior cannot be fully proven by local unit tests. | FR-004, NFR-004 | Local fakes and SDK transports can verify calls, names, and tags, but final product behavior depends on hosted Sentry's current metrics support. | Keep the metrics interface narrow and run MAN-003 before relying on production metrics. |
| GAP-002 | Sentry logs support may require SDK-specific handler behavior that is harder to inspect than local slog output. | FR-003, NFR-005 | Unit tests can verify filtering and handler calls, but hosted log indexing/search display is external. | Use fakes for automated safety tests and optionally verify one non-production Sentry log event manually. |
| GAP-003 | Performance overhead is not benchmarked in this acceptance stage. | NFR-003, NFR-006 | Requirements ask for minimal overhead, but exact overhead depends on implementation and load. | Add microbenchmarks or load tests only if implementation review finds measurable overhead risk. |

## 10. Out Of Scope

- Flutter client observability tests.
- Hosted Sentry dashboard, alert, issue ownership, uptime, and notification tests.
- OpenTelemetry/OTLP exporter tests.
- Database schema migration tests.
- Lexicon validation tests.
- Product analytics, user tracking, conversion funnel, or ranking telemetry tests.
- Full Sentry `sentrysql` / `database/sql` instrumentation tests.
- Exhaustive SQL-query-level span tests.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-02-appview-sentry-observability/`
- Recommended first failing test for implementation: `UT-001`, starting with config defaults that prove `SENTRY_DSN` alone enables only errors/recovered panics and new logs/tracing/metrics flags are explicitly gated.
- Suggested test order for implementation: `UT-001`, `UT-006`, `UT-003`, `UT-005`, `UT-004`, `UT-007`, `IT-002`, `IT-003`, `IT-004`, `IT-005`, `UT-010`, `IT-006`, `IT-007`, regression suite.
- Commands discovered: `just dev-d`; `just test`; `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`; `just fmt`.
- Blocking gaps: None.
