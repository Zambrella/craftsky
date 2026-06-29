# Acceptance Test Specification: AppView Observability

## 1. Test Strategy

This specification verifies the AppView-only observability baseline across structured logs, Prometheus `/metrics`, Sentry error capture/tracing configuration, HTTP middleware, Tap/indexer background work, DB/PDS dependency telemetry, and privacy-safe field handling. The requirements are **medium risk** because the work is cross-cutting and security-sensitive, even though it should not change product behavior.

Primary automation targets are Go tests under `appview/internal/**` and `appview/cmd/appview/**`. Unit tests should cover configuration parsing, allowlisted fields, redaction, route-pattern labeling, bounded operation/category vocabularies, and telemetry helper behavior. Middleware/server tests should cover request logs, correlation IDs, panic recovery, `/metrics`, and unchanged route behavior. Integration-style tests should cover Prometheus output from HTTP, Tap/indexer, DB health, and PDS/OAuth write paths where existing fake clients and stores make that practical.

Discovered commands:

- `just dev-d` from the repo root to start compose services before integration tests.
- `just test` from the repo root; runs `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...`.
- Focused AppView tests: `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index -count=1`.
- `just fmt` from the repo root for Go formatting and vetting.
- Manual dev scrape: start `just dev-d`, then request the documented AppView host port `/metrics`.

Existing relevant test conventions:

- Route policy and auth/device wrapping tests live in `appview/internal/routes/routes_test.go`.
- Middleware tests live in `appview/internal/middleware/*_test.go`, including request logging, CORS, device ID, auth, body limit, and rate limiting.
- Config tests live in `appview/internal/app/config_test.go`.
- App dependency wiring tests live in `appview/internal/app/deps_test.go` and `indexer_wiring_test.go`.
- HTTP handler and store tests live in `appview/internal/api/*_test.go`.
- Auth/PDS/OAuth write path tests live in `appview/internal/auth/*_test.go` and related `appview/internal/api/*_test.go` suites for profiles, posts, blobs, follows, reports, likes, and reposts.
- Tap consumer tests live in `appview/internal/tap/consumer_test.go`; indexer tests live in `appview/internal/index/*_test.go`.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-003, AC-004, AC-010, AC-019 | AT-001, AT-002, AT-003, AT-004, AT-012, IT-001, IT-002, IT-003, IT-004, IT-008 | Acceptance / Integration | Mixed |
| BR-002 | AC-005, AC-006, AC-015 | AT-005, AT-006, UT-002, UT-003, UT-004, IT-006, MAN-002 | Acceptance / Unit / Integration / Manual | Mixed |
| FR-001 | AC-001, AC-005, AC-014 | AT-001, AT-006, UT-001, UT-003, REG-001 | Acceptance / Unit / Regression | Yes |
| FR-002 | AC-001, AC-003, AC-004, AC-018 | AT-001, AT-003, AT-004, UT-001, IT-005 | Acceptance / Unit / Integration | Mixed |
| FR-003 | AC-002 | AT-002, IT-001, REG-002 | Acceptance / Integration / Regression | Yes |
| FR-004 | AC-002, AC-006, AC-014 | AT-002, AT-006, UT-003, IT-001, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-002, AC-017 | AT-007, UT-006, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-002, AC-006, AC-010, AC-013, AC-017, AC-019 | AT-008, AT-012, UT-005, UT-007, UT-012, IT-004, IT-005, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-003, AC-016 | AT-003, UT-008, IT-007, MAN-001 | Acceptance / Unit / Integration / Manual | Mixed |
| FR-008 | AC-003, AC-006, AC-010, AC-013 | AT-003, AT-008, UT-005, IT-004, IT-007 | Acceptance / Unit / Integration | Mixed |
| FR-009 | AC-004, AC-015 | AT-004, AT-009, UT-004, UT-009, IT-007, MAN-001 | Acceptance / Unit / Integration / Manual | Mixed |
| FR-010 | AC-004 | AT-009, UT-009, REG-003 | Acceptance / Unit / Regression | Yes |
| FR-011 | AC-004 | AT-004, AT-007, IT-003, GAP-001 | Acceptance / Integration / Gap | Mixed |
| FR-012 | AC-007 | AT-010, UT-008, REG-004 | Acceptance / Unit / Regression | Yes |
| FR-013 | AC-007 | AT-010, IT-001, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-014 | AC-008 | MAN-003 | Manual | No |
| FR-015 | AC-002, AC-011 | AT-002, IT-001, REG-002, MAN-004 | Acceptance / Integration / Regression / Manual | Mixed |
| FR-016 | AC-002 | UT-010, MAN-005 | Unit / Manual | Mixed |
| FR-017 | AC-016 | AT-003, UT-008, MAN-001 | Acceptance / Unit / Manual | Mixed |
| FR-018 | AC-006, AC-014, AC-015 | AT-006, UT-003, UT-004, IT-002, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-019 | AC-010, AC-013 | AT-008, UT-005, UT-007, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-020 | AC-019 | AT-012, UT-012, IT-008 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-006, AC-014 | AT-006, UT-003, UT-004, IT-002 | Acceptance / Unit / Integration | Yes |
| NFR-002 | AC-005, AC-006, AC-012, AC-015 | AT-005, AT-006, UT-002, UT-004, MAN-002 | Acceptance / Unit / Manual | Mixed |
| NFR-003 | AC-009 | AT-011, REG-001, REG-002, REG-003, REG-004 | Acceptance / Regression | Yes |
| NFR-004 | AC-009 | UT-008, IT-007, MAN-001 | Unit / Integration / Manual | Mixed |
| NFR-005 | AC-002, AC-003, AC-017 | UT-010, IT-001, IT-002, IT-003, IT-004, IT-005 | Unit / Integration | Yes |
| RULE-001 | AC-002, AC-011 | AT-002, REG-002 | Acceptance / Regression | Yes |
| RULE-002 | AC-005 | AT-005, UT-002, UT-004, MAN-002 | Acceptance / Unit / Manual | Mixed |
| RULE-003 | AC-007 | AT-010, UT-008, REG-004 | Acceptance / Unit / Regression | Yes |
| RULE-004 | AC-004, AC-006, AC-015 | AT-004, AT-006, UT-004, IT-007 | Acceptance / Unit / Integration | Mixed |
| RULE-005 | AC-005, AC-012 | AT-005, UT-002, REG-005, MAN-002 | Acceptance / Unit / Regression / Manual | Mixed |
| RULE-006 | AC-009 | AT-011, REG-004 | Acceptance / Regression | Yes |
| RULE-007 | AC-015 | AT-004, UT-004, MAN-001 | Acceptance / Unit / Manual | Mixed |
| RULE-008 | AC-011 | MAN-004 | Manual | No |
| RULE-009 | AC-018 | UT-011, IT-001 | Unit / Integration | Yes |
| RULE-010 | AC-016 | AT-003, UT-008, REG-006 | Acceptance / Unit / Regression | Yes |
| RULE-011 | AC-002, AC-017 | AT-002, IT-001, UT-011 | Acceptance / Integration / Unit | Yes |

## 3. Acceptance Scenarios

### AT-001: HTTP Request Logs Are Structured And Correlated

Requirement IDs: BR-001, FR-001, FR-002
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/logging_test.go`, `appview/internal/routes/routes_test.go`

```gherkin
Feature: AppView request logging
  Scenario: Completed HTTP requests emit stable JSON log fields with a shared run_id
    Given AppView request logging is configured with the JSON slog handler
    When a representative /v1 request completes and the handler logs an error
    Then the request log is valid JSON
    And it includes timestamp, level, message, environment, service, run_id, method, route pattern, status, and duration
    And the same run_id is available to downstream handlers and appears in handler error logs
    And the log uses the registered route pattern rather than the raw URL path
```

### AT-002: Metrics Endpoint Exposes Prometheus Text Outside V1 Auth

Requirement IDs: BR-001, FR-003, FR-004, FR-015, RULE-001, RULE-011
Acceptance Criteria: AC-002, AC-011
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/routes_test.go`, `appview/cmd/appview/server_test.go`

```gherkin
Feature: AppView Prometheus metrics endpoint
  Scenario: Metrics are scrapeable without Craftsky app-session middleware
    Given the AppView mux is registered
    When GET /metrics is requested without Authorization or X-Craftsky-Device-Id
    Then the response status is 200
    And the content type is Prometheus text exposition
    And the body includes HTTP metrics and at least one craftsky_appview service metric
    And the request is not handled under /v1/*
    And the response is not a Craftsky JSON auth error envelope
```

### AT-003: Sentry Tracing Is Optional, Bounded, And Flushes On Shutdown

Requirement IDs: BR-001, FR-002, FR-007, FR-008, FR-017, RULE-010
Acceptance Criteria: AC-003, AC-016
Priority: Should
Level: Acceptance
Automation Target: `appview/internal/app/config_test.go`, `appview/internal/observability/*_test.go`, `appview/cmd/appview/server_test.go`

```gherkin
Feature: Optional Sentry tracing
  Scenario: Configured Sentry tracing creates safe spans without requiring OpenTelemetry
    Given Sentry is configured with tracing enabled and a conservative production sample rate
    When HTTP requests, PDS/OAuth write-proxy operations, and Tap/indexer operations execute
    Then spans are created with service metadata, run_id linkage where available, duration, status, operation, component, and bounded attributes
    And Sentry trace and span IDs are available for log/event correlation where practical
    And graceful shutdown flushes pending Sentry telemetry
    And no OpenTelemetry or OTLP exporter is required
```

### AT-004: Sentry Captures Actionable Errors With Redacted Context

Requirement IDs: BR-001, FR-002, FR-009, FR-011, RULE-004, RULE-007
Acceptance Criteria: AC-004, AC-015
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/observability/*_test.go`, `appview/internal/middleware/*_test.go`, `appview/internal/tap/consumer_test.go`

```gherkin
Feature: Sentry error capture
  Scenario Outline: Actionable failures are captured with bounded technical context
    Given Sentry is configured with a test transport
    When <failure> occurs
    Then Sentry captures an event with component, environment, release if configured, run_id where available, route pattern or operation, status or error category, and bounded failure stage where applicable
    And the event is flushed during graceful shutdown
    And the event does not include raw DID, handle, device ID, session ID, token, AT-URI, CID, rkey, email, request body, response body, or uploaded media

    Examples:
      | failure |
      | an HTTP handler panic |
      | an AppView 5xx/internal error |
      | an unexpected PDS timeout or server failure |
      | a background Tap/indexer error |
```

### AT-005: Secrets And Payloads Are Omitted From Logs And Sentry

Requirement IDs: BR-002, FR-001, NFR-002, RULE-002, RULE-005
Acceptance Criteria: AC-005, AC-012
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/logging_test.go`, `appview/internal/observability/*_test.go`, `appview/internal/app/config_test.go`

```gherkin
Feature: Telemetry payload safety
  Scenario Outline: Sensitive headers, tokens, and bodies are not exported by default
    Given a request or error includes <sensitive_input>
    When logs and Sentry events are emitted in prod
    Then the sensitive value is redacted or omitted
    And request and response bodies are not included
    And dev-only response-body logging remains disabled unless the explicit unsafe local flag is set
    And response bodies are never exported to Sentry

    Examples:
      | sensitive_input |
      | Authorization header |
      | Cookie header |
      | OAuth access or refresh token |
      | PDS token |
      | DPoP proof or key material |
      | Craftsky session token |
      | JSON request body |
      | JSON response body |
      | uploaded media bytes |
```

### AT-006: Route Pattern And Allowlist Rules Prevent High Cardinality

Requirement IDs: BR-002, FR-004, FR-008, FR-018, NFR-001, NFR-002, RULE-004
Acceptance Criteria: AC-006, AC-014, AC-015
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/routes_test.go`, `appview/internal/middleware/logging_test.go`, `appview/internal/observability/*_test.go`

```gherkin
Feature: Safe telemetry attribute allowlist
  Scenario: Identifier-bearing routes and queries produce bounded telemetry fields
    Given matched routes contain DID, handle, rkey, hashtag, search text, cursor, or query parameters
    And an unmatched route contains raw identifiers
    When metrics, logs, and Sentry spans/events are emitted
    Then matched routes use the registered route pattern
    And unmatched routes use a bounded fallback such as unmatched
    And labels and attributes are limited to the approved safe allowlist
    And raw paths, full query strings, DIDs, handles, AT-URIs, CIDs, rkeys, tokens, device IDs, and user content are not used
```

### AT-007: Tap And Indexer Telemetry Covers Background Ingestion

Requirement IDs: BR-001, FR-005, FR-011, NFR-005
Acceptance Criteria: AC-002, AC-004, AC-017
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/tap/consumer_test.go`, `appview/internal/index/*_test.go`

```gherkin
Feature: Tap and indexer observability
  Scenario: Firehose ingestion emits metrics and safe error context
    Given a fake Tap server sends supported, skipped, malformed, and indexer-error frames
    When the Tap consumer processes frames, reconnects, acknowledges events, or fails to acknowledge
    Then metrics record connected state, reconnect attempts, last-event age, events received, events acknowledged, ack failures, records indexed, records skipped, indexing errors, and handling duration
    And per-NSID labels use known registered NSIDs or bounded fallbacks
    And background errors are logged and captured in Sentry when configured without raw record payloads or user identifiers
```

### AT-008: PDS And DB Dependency Telemetry Uses Bounded Outcomes

Requirement IDs: BR-001, FR-006, FR-008, FR-019
Acceptance Criteria: AC-010, AC-013, AC-017
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/auth/*_test.go`, `appview/internal/api/*_test.go`, `appview/internal/db/*_test.go`, `appview/internal/observability/*_test.go`

```gherkin
Feature: Dependency and PDS write telemetry
  Scenario Outline: AppView write-proxy paths emit operation, result, stage, category, and duration
    Given a fake PDS/OAuth client or DB operation can return success and categorized failures
    When <operation> succeeds or fails
    Then logs and metrics include bounded operation, result, duration, failure stage where applicable, and safe error category
    And Sentry spans include the same bounded context when tracing is enabled
    And expected validation, unauthorized, forbidden, not-found, and rate-limited PDS responses are not captured as Sentry error events by default
    And no token, DPoP material, request payload, response payload, raw DID, handle, AT-URI, CID, or rkey is exported

    Examples:
      | operation |
      | OAuth session resume |
      | profile write |
      | post create |
      | post delete |
      | blob upload |
      | follow |
      | unfollow |
      | like |
      | unlike |
      | repost |
      | unrepost |
      | selected DB health or manual operation |
```

### AT-009: HTTP Panics Are Recovered And Reported

Requirement IDs: FR-009, FR-010
Acceptance Criteria: AC-004
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/*_test.go`, `appview/internal/routes/routes_test.go`

```gherkin
Feature: HTTP panic recovery
  Scenario: A panicking v1 handler does not crash the AppView process
    Given a /v1 handler panics before writing a response
    When the request is served
    Then the panic is logged with run_id and safe route context
    And Sentry captures the panic when configured
    And the response uses the standard AppView error envelope with error, message, and requestId where possible
    And the process continues serving subsequent requests
```

### AT-010: Missing Sentry Configuration Keeps AppView Local And Functional

Requirement IDs: FR-012, FR-013, RULE-003
Acceptance Criteria: AC-007
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/app/config_test.go`, `appview/internal/app/deps_test.go`, `appview/cmd/appview/server_test.go`

```gherkin
Feature: Safe disabled defaults for hosted observability
  Scenario: Sentry is absent in dev or tests
    Given no Sentry DSN or tracing configuration is present
    When AppView configuration and dependencies are initialized
    Then startup succeeds
    And logs remain local JSON
    And GET /metrics remains available
    And no external Sentry telemetry export is attempted
    And request handling does not emit per-request Sentry errors
```

### AT-011: Existing AppView Behavior Is Preserved

Requirement IDs: NFR-003, NFR-004, RULE-006
Acceptance Criteria: AC-009
Priority: Must
Level: Acceptance
Automation Target: existing `appview/internal/routes/*_test.go`, `appview/internal/api/*_test.go`, `appview/internal/middleware/*_test.go`

```gherkin
Feature: Observability side effects do not change product behavior
  Scenario: Existing route behavior remains stable with instrumentation enabled or disabled
    Given representative existing /v1, /oauth, /health, and /healthz route tests run
    When observability instrumentation is enabled and when optional backends are disabled
    Then response status codes, response bodies, auth requirements, device ID behavior, rate-limit behavior, route availability, and /healthz HTTP 200 semantics remain unchanged
```

### AT-012: Search Request Telemetry Shows Whether DB Time Dominates

Requirement IDs: BR-001, FR-006, FR-020
Acceptance Criteria: AC-019
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_test.go`, `appview/internal/api/search_store_test.go`, `appview/internal/observability/*_test.go`

```gherkin
Feature: Request-backed DB operation timing
  Scenario: Search telemetry can be compared against total request duration
    Given an authenticated device calls an AppView search route
    And the backing search store operation is instrumented as a bounded DB operation
    When the request completes
    Then telemetry records total HTTP request duration for the registered search route pattern
    And telemetry records bounded DB operation duration for the corresponding search operation
    And the HTTP and DB operation telemetry share enough route and correlation context to compare the durations
    And maintainers can determine whether most response time was spent in database work
    And telemetry does not include raw SQL parameters, search query text, raw DIDs, handles, tokens, or request bodies
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-002 | AC-001, AC-018 | Validate request log field construction and run_id propagation helpers. | Request context with run_id, method, route pattern, status, duration, optional trace IDs. | Stable JSON-safe field set includes run_id and optional trace/span IDs; no raw path when route pattern exists. | `appview/internal/middleware/logging_test.go` |
| UT-002 | BR-002, NFR-002, RULE-002, RULE-005 | AC-005, AC-012 | Verify redaction/omission helper for sensitive headers, tokens, cookies, DPoP, and bodies. | Header map, body marker, token-like values. | Sensitive values are redacted or omitted; bodies are absent unless dev unsafe local logging is explicitly allowed. | `appview/internal/observability/*_test.go`, `appview/internal/middleware/logging_test.go` |
| UT-003 | FR-004, FR-018, NFR-001 | AC-006, AC-014 | Verify route-pattern resolution and unmatched fallback for telemetry labels. | Registered route, unmatched route, path with DID/rkey/query. | Matched routes use pattern; unmatched routes use bounded fallback; raw query/path identifiers are excluded. | `appview/internal/routes/routes_test.go`, `appview/internal/observability/*_test.go` |
| UT-004 | BR-002, FR-009, RULE-004, RULE-007 | AC-015 | Validate Sentry event context allowlist. | Event context with allowed and disallowed fields. | Only service, environment, release, component, operation, route pattern, status, run_id, trace IDs, category/stage, result, and duration are retained. | `appview/internal/observability/*_test.go` |
| UT-005 | FR-006, FR-008, FR-019 | AC-010, AC-013 | Validate PDS write operation/stage/category vocabulary. | Success, timeout, network, auth, rate_limited, validation, not_found, forbidden, server, unexpected errors across stages. | Errors map to bounded categories and stages; unexpected values use bounded fallback. | `appview/internal/auth/*_test.go`, `appview/internal/observability/*_test.go` |
| UT-006 | FR-005 | AC-002, AC-017 | Validate Tap/indexer metric label classification. | Known Craftsky NSID, known Bluesky NSID, unsupported NSID, malformed event. | Labels use registered NSID or bounded fallback and result/reason values are low-cardinality. | `appview/internal/tap/consumer_test.go`, `appview/internal/index/*_test.go` |
| UT-007 | FR-006, FR-019 | AC-010, AC-013 | Verify every current PDS write operation has an instrumentation name. | Enumerated operation registry. | Profile, post create/delete, blob upload, follow/unfollow, like/unlike, repost/unrepost, OAuth/session-resume paths are all covered. | `appview/internal/auth/*_test.go`, `appview/internal/api/*_test.go` |
| UT-008 | FR-007, FR-012, FR-017, RULE-003, RULE-010 | AC-007, AC-016 | Validate Sentry config defaults and sampling rules. | Empty DSN, dev DSN, prod DSN, invalid sample rate, explicit sample rate, unsafe response-body flag. | Empty DSN disables export; prod defaults conservatively; sample rate is validated; unsafe body logging is rejected or ignored in prod. | `appview/internal/app/config_test.go` |
| UT-009 | FR-009, FR-010 | AC-004 | Verify panic recovery helper captures/logs safely and preserves API envelope when possible. | Panic before response write, panic after partial response write. | Before-write panic returns standard envelope; after-write panic is logged/captured best-effort without process crash. | `appview/internal/middleware/*_test.go` |
| UT-010 | FR-016, NFR-005 | AC-002, AC-017 | Validate metric names, units, and help text follow the internal service convention. | Registered collectors. | Names use `craftsky_appview` convention and document counter/gauge/histogram units. | `appview/internal/observability/*_test.go` |
| UT-011 | RULE-009, RULE-011 | AC-017, AC-018 | Guard first-slice non-goals in metrics setup. | Metrics registry configuration. | No Prometheus exemplars or Sentry Application Metrics are required or initialized. | `appview/internal/observability/*_test.go` |
| UT-012 | FR-006, FR-020 | AC-019 | Validate request-backed DB operation naming and correlation helpers. | Route pattern, run_id, bounded DB operation name such as `search.posts`, HTTP duration, DB duration. | HTTP and DB telemetry can be compared by route/correlation context without raw SQL, query text, or per-query span data. | `appview/internal/observability/*_test.go`, `appview/internal/api/search_store_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-003, FR-004, FR-015, RULE-001, RULE-011 | AC-002, AC-011, AC-017 | `/metrics` returns Prometheus output and HTTP metrics. | Build AppView mux/server with test registry. | Call `/metrics` before and after representative requests. | Endpoint is 200 without app auth; output includes HTTP counters/gauges/histograms and service metrics. | `appview/internal/routes/routes_test.go`, `appview/cmd/appview/server_test.go` |
| IT-002 | FR-004, FR-018, NFR-001 | AC-006, AC-014 | HTTP metrics/logs use route patterns. | Register representative routes with path variables and query values. | Send matched and unmatched requests. | Telemetry uses route patterns or `unmatched`; raw identifiers and query strings are absent. | `appview/internal/routes/routes_test.go`, `appview/internal/middleware/logging_test.go` |
| IT-003 | FR-005, FR-011 | AC-002, AC-004, AC-017 | Tap/indexer path emits ingestion metrics and errors. | Fake Tap server and fake indexer with success, skip, and error cases. | Run consumer until events are handled. | Metrics capture connected/reconnect/ack/indexer outcomes; Sentry test transport captures configured background errors safely. | `appview/internal/tap/consumer_test.go`, `appview/internal/index/*_test.go` |
| IT-004 | FR-006, FR-019 | AC-010, AC-013, AC-017 | PDS/OAuth write-proxy operations emit bounded metrics/logs. | Fake PDS/OAuth clients returning success and categorized failures. | Exercise profile/post/blob/follow/like/repost/session-resume paths. | Each path emits operation, result, duration, stage/category where applicable, without secrets or payloads. | `appview/internal/auth/*_test.go`, `appview/internal/api/*_test.go` |
| IT-005 | FR-002, FR-006, FR-008 | AC-003, AC-010, AC-018 | Sentry traces correlate with logs for HTTP and write operations. | Sentry configured with test transport and tracing enabled. | Serve request that performs a fake PDS write. | Logs include run_id and trace/span IDs where practical; spans use bounded operation attributes. | `appview/internal/observability/*_test.go`, `appview/internal/api/*_test.go` |
| IT-006 | BR-002, NFR-002 | AC-005, AC-006, AC-012, AC-015 | End-to-end telemetry redaction for representative API requests. | Request with auth headers, device ID, body, and response. | Serve request with logging and Sentry test transport enabled. | Telemetry excludes tokens, raw identity, body content, and uploaded media; route pattern and bounded categories remain. | `appview/internal/middleware/logging_test.go`, `appview/internal/observability/*_test.go` |
| IT-007 | FR-007, FR-009, NFR-004 | AC-003, AC-004, AC-016 | Sentry initialization, non-blocking runtime behavior, and flush lifecycle. | Sentry test transport plus failing/unavailable transport simulation. | Start server, emit error/span, shut down. | Startup succeeds; telemetry failures do not block requests; shutdown flush is called. | `appview/internal/app/deps_test.go`, `appview/cmd/appview/server_test.go` |
| IT-008 | BR-001, FR-006, FR-020 | AC-019 | Search request emits comparable HTTP and DB operation durations. | Authenticated test search request with fake or instrumented store timing. | Serve search request. | Telemetry exposes total route duration and bounded search DB operation duration under shared route/correlation context, making DB-dominated latency visible without per-query spans. | `appview/internal/api/search_test.go`, `appview/internal/api/search_store_test.go`, `appview/internal/observability/*_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Existing request logging injects `run_id` and omits request bodies. | FR-001, NFR-003 | Extend `appview/internal/middleware/logging_test.go` without removing current `run_id` and redaction assertions. |
| REG-002 | `/metrics` is an ops endpoint, not a `/v1/*` route requiring Craftsky auth/device middleware. | FR-003, FR-015, RULE-001 | Route test proves `/metrics` is unauthenticated while representative `/v1/*` routes still require auth/device headers. |
| REG-003 | HTTP panics do not crash the process and v1 errors use the standard envelope where possible. | FR-010, NFR-003 | Panic recovery test followed by a second healthy request. |
| REG-004 | `/healthz` continues returning HTTP 200 with `status` `ok` or `degraded`. | RULE-006, NFR-003 | Keep existing `appview/internal/api/healthz_test.go` expectations unchanged. |
| REG-005 | Dev response-body logging remains disabled by default and cannot be enabled in prod. | RULE-005, NFR-002 | Config and logging tests for default dev behavior and prod unsafe flag handling. |
| REG-006 | OpenTelemetry/OTLP export remains out of scope for this slice. | RULE-010 | Config/deps tests ensure no OTLP exporter configuration is required for startup or tracing tests. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | HTTP route telemetry safety | Requests to matched `/v1/posts/{rkey}`-style routes, search routes with query text, unmatched identifier-bearing paths, and `/metrics`. | AT-001, AT-002, AT-006, IT-001, IT-002 |
| TD-002 | Sensitive data redaction | Authorization, Cookie, OAuth/PDS token strings, DPoP proof/key placeholders, Craftsky session token, JSON body, response body, uploaded media marker, raw DID/handle/device ID/email. | AT-005, AT-006, IT-006, MAN-002 |
| TD-003 | Tap/indexer telemetry | Fake Tap frames for supported create/delete, malformed record, unsupported NSID, indexer error, reconnect, and ack failure. | AT-007, IT-003 |
| TD-004 | PDS/OAuth write telemetry | Fake operations for profile write, post create/delete, blob upload, follow/unfollow, like/unlike, repost/unrepost, session resume, plus timeout/network/auth/rate-limit/validation/not-found/forbidden/server/unexpected failures. | AT-008, UT-005, UT-007, IT-004 |
| TD-005 | Sentry lifecycle | Empty DSN, configured test DSN, tracing enabled/disabled, dev/prod envs, valid/invalid sample rates, failing test transport. | AT-003, AT-004, AT-010, IT-005, IT-007 |
| TD-006 | Metrics documentation | Registered collector names, help strings, label names, and units for HTTP, Tap/indexer, DB, PDS/OAuth, process/runtime metrics. | UT-010, MAN-005 |
| TD-007 | Search DB timing comparison | Authenticated search request, route pattern, run_id, bounded search operation name, controlled fake/store delay, HTTP duration, DB operation duration. | AT-012, UT-012, IT-008 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | FR-007, FR-009, FR-017, RULE-007 | Optional Sentry smoke check with a real configured project or local SDK transport. | Configure Sentry in a non-production environment, trigger a test internal error and trace, then inspect the captured event/span. | Event/span contains safe technical context and no raw identity, tokens, bodies, or user content; sampling follows config; shutdown flushes events. |
| MAN-002 | BR-002, NFR-002, RULE-002, RULE-005 | Human review of telemetry payloads for privacy. | Inspect representative logs, Prometheus output, and Sentry test events generated by automated tests. | No secrets, request/response bodies, uploaded media, raw user identity, raw paths with identifiers, or full query strings are present. |
| MAN-003 | FR-014 | Dev Docker `/metrics` reachability. | Run `just dev-d`, request the documented AppView host port `/metrics`, and inspect output. | Metrics are reachable and include at least one `craftsky_appview` metric. |
| MAN-004 | FR-015, RULE-008 | Production deployment restriction note. | Review implementation docs/config notes for `/metrics` exposure. | Documentation states production `/metrics` must be restricted by network policy, reverse proxy, or platform ingress; AppView itself does not add Craftsky user auth. |
| MAN-005 | FR-016, NFR-005 | Metric name/unit documentation review. | Compare implemented metric names/help/labels against the implementation docs. | Metric names use consistent `craftsky_appview` convention, units are clear, and names are documented as internal ops details for this slice. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Background panic capture may be hard to prove without restructuring worker boundaries. | FR-011 | Current Tap consumer tests can verify returned errors and indexer failures, but panic recovery may require new wrapper boundaries. | Coding plan should identify the exact boundary for worker panic recovery and add a focused test there. |
| GAP-002 | Real Sentry hosted behavior cannot be fully proven in automated tests. | FR-007, FR-009, FR-017 | CI should not depend on a hosted vendor project or network access. | Use test transport automation plus MAN-001 smoke check in a non-production environment. |
| GAP-003 | Production network restriction for `/metrics` is outside AppView code. | FR-015, RULE-008 | AppView can document and keep endpoint unauthenticated, but ingress/network policy lives in deployment infrastructure. | Implementation docs must call out the restriction; deployment work can add infrastructure checks later. |
| GAP-004 | Overhead is only partially measurable in this stage. | NFR-004 | Unit/integration tests can assert non-blocking design and bounded labels but not production performance. | Coding plan should avoid synchronous vendor calls on hot paths and consider lightweight benchmarks only if implementation introduces expensive wrappers. |

## 10. Out Of Scope

- Flutter client observability, product analytics, user tracking, funnels, or ranking telemetry.
- Production dashboards, alert rules, hosted Prometheus/Grafana, hosted OpenTelemetry collector, or Sentry project provisioning.
- OpenTelemetry/OTLP export, OTLP/gRPC, multiple trace exporters, and DB auto-instrumentation spans.
- Sentry Application Metrics; Prometheus `/metrics` is the first-slice metric contract.
- Prometheus exemplars or trace-linked metric exemplars.
- Database schema changes, lexicon changes, API response contract changes, app auth changes, or Flutter behavior changes.
- Automated proof of production network/proxy restrictions for `/metrics`; this remains deployment documentation/manual review in this slice.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-06-29-appview-observability/01-requirements.md`
- Test specification: `docs/changes/2026-06-29-appview-observability/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-06-29-appview-observability/`
- Recommended first failing test for implementation: `AT-002` / `IT-001`, proving unauthenticated `GET /metrics` returns Prometheus text outside `/v1/*` while existing `/v1/*` auth/device behavior remains intact.
- Suggested test order for implementation: `AT-002`, `UT-008`, `AT-005`, `AT-001`, `AT-006`, `AT-012`, `AT-009`, `AT-008`, `AT-007`, `AT-003`, `AT-004`, `AT-011`, then manual checks.
- Commands discovered: `just dev-d`; `just test`; `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index -count=1`; `just fmt`.
- Blocking gaps: None. Non-blocking gaps are listed in Section 9.
