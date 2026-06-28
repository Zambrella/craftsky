# Acceptance Test Specification: AppView Architecture Hardening

## 1. Test Strategy

This specification verifies AppView v1 cross-cutting API hardening before launch: bare success responses, standard error envelopes, device ID enforcement, body-size limits, route-class rate limits, CORS policy, safe configuration, and logging redaction. The requirements are **high risk** because they affect API contracts, browser security posture, abuse controls, middleware ordering, and launch deployment constraints. Explicit review approval is recommended before implementation continues.

Primary automation targets are Go tests in `appview/internal/**` using existing middleware, route, config, and API test conventions. `just test` is the full discovered command; focused package commands can be run under `appview/` with the same `TEST_DATABASE_URL` when compose Postgres is available.

Discovered commands:

- `just dev-d` from the repo root to start compose services before integration tests.
- `just test` from the repo root; runs `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...`.
- `just fmt` from the repo root for Go formatting and vetting.

Existing relevant test conventions:

- Route auth/device wrapping tests live in `appview/internal/routes/routes_test.go`.
- Middleware tests live in `appview/internal/middleware/*_test.go`, including CORS, device ID, auth, and logging.
- Error envelope tests live in `appview/internal/api/envelope/envelope_test.go`.
- Config tests live in `appview/internal/app/config_test.go`.
- API response-shape tests live in `appview/internal/api/*_response_test.go` and handler tests in `appview/internal/api/*_test.go`.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, UT-001, IT-001, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| BR-002 | AC-006, AC-007, AC-008 | AT-004, AT-005, UT-007, IT-004, IT-005, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| BR-003 | AC-009, AC-010 | AT-006, UT-005, IT-007, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| FR-001 | AC-003, AC-004 | AT-002, UT-002, IT-002, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| FR-002 | AC-003, AC-004 | AT-002, UT-002, UT-003, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-005, AC-007 | AT-003, UT-004, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-011, AC-012 | AT-007, UT-008, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-013 | AT-008, UT-009, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-006 | AT-004, UT-006, IT-004, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-007 | AC-007, AC-008 | AT-005, UT-007, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-007, AC-008 | AT-003, AT-005, UT-007, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-008 | AT-005, UT-007, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-009, AC-010 | AT-006, UT-005, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-010 | UT-005, IT-007 | Unit / Integration | Yes |
| FR-012 | AC-011 | AT-007, UT-008, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-016 | AT-009, UT-010, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-017 | AT-010, UT-005, IT-008 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-011, AC-012, AC-015 | AT-007, UT-008, UT-011, IT-006, MAN-001 | Acceptance / Unit / Integration / Manual | Mixed |
| NFR-002 | AC-007, AC-008 | AT-005, UT-007, IT-005, MAN-001 | Acceptance / Unit / Integration / Manual | Mixed |
| NFR-003 | AC-014 | UT-012, IT-009 | Unit / Integration | Yes |
| NFR-004 | AC-015 | UT-011, MAN-001 | Unit / Manual | Mixed |
| RULE-001 | AC-001, AC-002 | AT-001, UT-001, IT-001, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-002 | AC-003 | AT-002, UT-002, IT-002, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-003 | AC-009 | AT-006, UT-005, IT-007, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-004 | AC-006, AC-013 | AT-004, AT-008, UT-006, UT-009, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-005 | AC-018 | AT-011, UT-013, IT-003 | Acceptance / Unit / Integration | Yes |
| RULE-006 | AC-019 | UT-007, IT-005, MAN-002 | Unit / Integration / Manual | Mixed |
| RULE-007 | AC-020 | AT-012, UT-007, IT-005 | Acceptance / Unit / Integration | Yes |
| RULE-008 | AC-021 | IT-009, MAN-002 | Integration / Manual | Mixed |

## 3. Acceptance Scenarios

### AT-001: Successful V1 Responses Remain Bare

Requirement IDs: BR-001, RULE-001
Acceptance Criteria: AC-001, AC-002
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/routes_test.go`, representative `appview/internal/api/*_response_test.go`

```gherkin
Feature: AppView v1 success response shape
  Scenario: Successful endpoint responses are not wrapped for envelope purposes
    Given representative successful /v1 JSON endpoints are registered
    When an authenticated device calls simple and paginated endpoints successfully
    Then each response body uses the endpoint-defined top-level shape
    And no response is wrapped solely in a top-level data field
    And paginated responses keep endpoint-defined items and cursor fields at the top level
```

### AT-002: V1 Errors Use The Standard Envelope

Requirement IDs: FR-001, FR-002, RULE-002
Acceptance Criteria: AC-003, AC-004
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/envelope/envelope_test.go`, `appview/internal/routes/routes_test.go`

```gherkin
Feature: AppView v1 error response shape
  Scenario: Handler and middleware errors use the canonical envelope
    Given representative /v1 handlers and middleware can return client or server errors
    When an error response is emitted
    Then the status matches the failure
    And Content-Type is application/json
    And the JSON body contains error, message, and requestId
    And fields is present only when field-level validation details apply
```

### AT-003: Applicable V1 Routes Require Valid Device IDs

Requirement IDs: FR-003, FR-008, RULE-005
Acceptance Criteria: AC-005, AC-007, AC-018
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/device_id_test.go`, `appview/internal/routes/routes_test.go`

```gherkin
Feature: Device ID middleware
  Scenario Outline: Device ID is required but not trusted as identity
    Given an applicable /v1 route is registered
    When a request includes <device_header>
    Then the request is <result>
    And rejected requests use HTTP 400 and the standard error envelope before the endpoint handler runs
    And accepted device IDs are available only as an abuse-control signal, not authorization proof

    Examples:
      | device_header | result |
      | missing | rejected with missing_device_id |
      | has spaces | rejected with invalid_device_id |
      | 129 allowed characters | rejected with invalid_device_id |
      | valid A-Z a-z 0-9 _ - value up to 128 chars | passed to downstream auth/rate-limit logic |
```

### AT-004: Every V1 Route Has Explicit Limit Classifications

Requirement IDs: BR-002, FR-006, RULE-004
Acceptance Criteria: AC-006, AC-013
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/routes_test.go` or a new route-policy contract test suite

```gherkin
Feature: Route hardening policy registration
  Scenario: Route table cannot silently bypass body or rate-limit policy
    Given the AppView route table is built
    When tests inspect all registered /v1/* routes
    Then every route has an explicit rate-limit class of auth, read, write, expensive/search, upload, exempt, or dev-only relaxed
    And every route has an explicit body policy of no-body, default JSON limit, or named override
    And dev-only routes cannot appear in production without an intentional documented classification
```

### AT-005: Route-Class Rate Limiting Rejects Excess Token Or Device Traffic

Requirement IDs: BR-002, FR-007, FR-008, FR-009, NFR-002
Acceptance Criteria: AC-007, AC-008
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/rate_limit_test.go`, `appview/internal/routes/routes_test.go`

```gherkin
Feature: Process-local route-class rate limiting
  Scenario Outline: Requests exceeding either token or device bucket are throttled early
    Given a route in class <class> has configured limits
    And a client sends requests with Craftsky token <token> and device ID <device>
    When the client exceeds either the per-token or per-device bucket for that class
    Then additional requests in that class are rejected before endpoint handler work proceeds
    And the response is HTTP 429 with error rate_limited
    And the response includes Retry-After
    And the response does not include public X-RateLimit-* quota or bucket details

    Examples:
      | class | token | device |
      | read | token-a | device-a |
      | write | token-a | device-a |
      | expensive/search | token-a | device-a |
      | upload | token-a | device-a |
      | auth | none | device-a |
```

### AT-006: Production CORS Allows Only Exact Configured Web Origins

Requirement IDs: BR-003, FR-010, FR-011, RULE-003
Acceptance Criteria: AC-009, AC-010
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/cors_test.go`, `appview/internal/app/config_test.go`

```gherkin
Feature: Production CORS policy
  Scenario Outline: Origin handling follows exact allow-list policy
    Given AppView runs with production CORS configuration allowing https://app.craftsky.social
    When a browser request has Origin <origin>
    Then Access-Control-Allow-Origin is <allowed_origin>
    And credentialed cookie CORS is not enabled for v1

    Examples:
      | origin | allowed_origin |
      | https://app.craftsky.social | https://app.craftsky.social |
      | https://craftsky.social | empty |
      | https://preview.craftsky.social | empty |
      | https://evil.example | empty |
      | no Origin header | empty |
```

### AT-007: Default JSON Body Limit Rejects Oversized Requests Early

Requirement IDs: FR-004, FR-012, NFR-001
Acceptance Criteria: AC-011, AC-012, AC-015
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/body_limit_test.go`, `appview/internal/middleware/logging_test.go`

```gherkin
Feature: Default JSON body size limit
  Scenario: Oversized default-limited JSON body is rejected before parsing or logging copies it
    Given a default-limited JSON /v1 route with a 1 MiB limit
    When a request body larger than 1 MiB is sent
    Then the response is HTTP 413
    And the error code is request_body_too_large
    And the message is request body exceeds the configured limit
    And the standard error envelope is used
    And the endpoint handler and body-copying debug logger do not read the full rejected body
```

### AT-008: Body Limit Overrides Supersede The Default

Requirement IDs: FR-005, RULE-004
Acceptance Criteria: AC-013
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/body_limit_test.go`, `appview/internal/api/blob_test.go`

```gherkin
Feature: Per-route body-size overrides
  Scenario: Upload endpoint uses its explicit upload limit rather than the JSON default
    Given an upload route has an explicit body-size override
    When a request is larger than 1 MiB but within the upload override
    Then body-limit middleware does not reject it for the default JSON limit
    When a request exceeds the upload override
    Then the request is rejected according to the override with a standard error envelope
```

### AT-009: No-Body Routes Reject Unexpected Bodies

Requirement IDs: FR-013
Acceptance Criteria: AC-016
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/body_limit_test.go`, `appview/internal/routes/routes_test.go`

```gherkin
Feature: No-body route enforcement
  Scenario: GET route rejects a non-empty request body
    Given a /v1 route declares no request body
    When a client sends a non-empty body to that route
    Then the request is rejected before endpoint handler work proceeds
    And the error code is request_body_not_allowed
    And a request with no body is not rejected solely by the body policy
```

### AT-010: Allowed CORS Preflights Do Not Consume Route Buckets

Requirement IDs: FR-014
Acceptance Criteria: AC-017
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/cors_test.go`, `appview/internal/middleware/rate_limit_test.go`

```gherkin
Feature: CORS preflight handling order
  Scenario: Successful preflight short-circuits before route-class rate limiting
    Given an allowed Origin requests a supported method and headers by OPTIONS preflight
    And the matching route has a very low route-class rate limit
    When multiple successful preflight requests are sent
    Then CORS handles them without invoking the route handler
    And the normal route-class token/device buckets are not consumed
```

### AT-011: Device ID Is Not Authorization Evidence

Requirement IDs: RULE-005, FR-003, FR-008
Acceptance Criteria: AC-018
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/routes_test.go`, `appview/internal/middleware/auth_test.go`

```gherkin
Feature: Device ID trust boundary
  Scenario: Valid device ID cannot authorize an authenticated route without a valid session
    Given an authenticated /v1 route is registered
    When a request includes a syntactically valid X-Craftsky-Device-Id but no valid Authorization token
    Then the request is unauthorized
    And the device ID is not treated as proof of user identity or permission
```

### AT-012: Upload Rate Limits Count Attempts

Requirement IDs: RULE-007, BR-002
Acceptance Criteria: AC-020
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/middleware/rate_limit_test.go`, `appview/internal/api/blob_test.go`

```gherkin
Feature: Upload attempt throttling
  Scenario: Failed upload attempts consume the upload route-class bucket
    Given the upload class has a configured request limit
    When a client sends upload attempts that fail validation after reaching the limiter
    And the number of attempts exceeds the upload class limit
    Then a subsequent upload attempt is rejected with HTTP 429 rate_limited
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | BR-001, RULE-001 | AC-001, AC-002 | Assert success helper/response encoders do not add envelope-only `data`. | Representative simple object and paginated object. | JSON has endpoint fields at top level; no synthetic `data`. | `appview/internal/api/*_response_test.go` |
| UT-002 | FR-001, FR-002, RULE-002 | AC-003, AC-004 | Verify canonical error helper writes status, JSON content type, error/message/requestId, optional fields. | Error code/message/requestId with nil and populated fields. | Existing standard envelope shape preserved. | `appview/internal/api/envelope/envelope_test.go` |
| UT-003 | FR-002 | AC-003, AC-004 | Verify success helper, if added, sets status and `Content-Type: application/json` without changing body contract. | Status 200/201 and response struct. | Header/status correct; bare body. | New helper tests near `appview/internal/api/envelope/` or API package |
| UT-004 | FR-003 | AC-005 | Validate device ID accepted/rejected by existing rule. | Missing, empty, spaces, 129 chars, valid `[A-Za-z0-9_-]{1,128}`. | 400 envelope for invalid; valid value stored in context. | `appview/internal/middleware/device_id_test.go` |
| UT-005 | BR-003, FR-010, FR-011, RULE-003, FR-014 | AC-009, AC-010, AC-017 | Verify CORS exact allow-list, wildcard dev behavior, supported headers/methods, no credentials, and preflight short-circuit. | Origins: app, marketing, preview, evil, localhost/wildcard dev; preflight headers. | Only allowed origins echoed; preflight succeeds; next/rate limiter not called. | `appview/internal/middleware/cors_test.go` |
| UT-006 | FR-006, RULE-004 | AC-006 | Validate route policy registry rejects unclassified `/v1/*` routes. | Route table with classified and unclassified routes. | Test/helper fails unclassified route; allows documented exempt/dev-only class. | `appview/internal/routes/routes_test.go` |
| UT-007 | BR-002, FR-007, FR-008, FR-009, RULE-006, RULE-007 | AC-007, AC-008, AC-019, AC-020 | Verify limiter keys and counters for token/device route classes, no IP keys, Retry-After calculation, upload attempt counting. | Low limits, fake clock, token/device combinations, failed upload attempts. | Correct key throttled; 429 envelope; Retry-After present; no quota headers; no IP key used. | `appview/internal/middleware/rate_limit_test.go` |
| UT-008 | FR-004, FR-012, NFR-001 | AC-011, AC-012, AC-015 | Verify default body-limit middleware rejects over 1 MiB before handler reads and allows at/below limit. | Bodies of 1 MiB, 1 MiB + 1 byte. | At limit passes; over limit gets 413 envelope and handler not called. | `appview/internal/middleware/body_limit_test.go` |
| UT-009 | FR-005, RULE-004 | AC-013 | Verify body-limit override resolution. | Default route and upload override route with below/above sizes. | Override route uses override, not default. | `appview/internal/middleware/body_limit_test.go` |
| UT-010 | FR-013 | AC-016 | Verify no-body policy rejects non-empty bodies and allows absent bodies. | GET/DELETE with nil, empty, and non-empty bodies. | Non-empty body gets request_body_not_allowed; nil/empty passes policy. | `appview/internal/middleware/body_limit_test.go` |
| UT-011 | NFR-001, NFR-004 | AC-015 | Verify rejection logging redacts bearer tokens and avoids full rejected body. | Request with Authorization and oversized sensitive payload. | Logs include route/status/limit context but not token or body content. | `appview/internal/middleware/logging_test.go`, body/rate limiter tests with test logger |
| UT-012 | NFR-003 | AC-014 | Verify config defaults and invalid values for body/rate/CORS limits. | Missing env values, invalid duration/count/byte values, prod wildcard origin. | Safe defaults applied; invalid config fails startup with clear env name. | `appview/internal/app/config_test.go` |
| UT-013 | RULE-005 | AC-018 | Verify middleware/context helpers never derive DID/session identity from device ID. | Valid device ID without auth context. | Device ID is context-only abuse signal; auth context remains absent. | `appview/internal/middleware/device_id_test.go`, `auth_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, RULE-001 | AC-001, AC-002 | Representative route responses stay bare through real mux. | Build mux with `routes.AddRoutes` and fake deps. | Call `/v1/whoami` and one paginated list route successfully. | Bodies expose endpoint fields directly and no synthetic `data`. | `appview/internal/routes/routes_test.go` |
| IT-002 | FR-001, FR-002, RULE-002 | AC-003, AC-004 | Middleware/handler errors through mux use standard envelope. | Build mux with fake deps. | Trigger missing auth, missing device ID, validation error. | JSON content type and canonical envelope with requestId. | `appview/internal/routes/routes_test.go`, API handler tests |
| IT-003 | FR-003, RULE-005 | AC-005, AC-018 | Device ID runs before applicable handlers and does not bypass auth. | Mux with authenticated route and test handler spy. | Send missing/malformed/valid-device-without-auth requests. | Invalid device rejected before handler; valid device without auth remains unauthorized. | `appview/internal/routes/routes_test.go` |
| IT-004 | BR-002, FR-006, RULE-004 | AC-006 | Full route table policy coverage. | Build production and dev route tables/policy registry. | Inspect every `/v1/*` registration. | Every route has rate class and body policy; dev-only routes classified or absent in prod. | `appview/internal/routes/routes_test.go` |
| IT-005 | BR-002, FR-007, FR-008, FR-009, RULE-006, RULE-007 | AC-007, AC-008, AC-019, AC-020 | End-to-end limiter behavior across token/device buckets. | Low test limits, fake clock, mux with handler counter. | Exceed read/write/search/upload/auth limits by token and device. | 429 envelope, Retry-After, no quota headers; handler counter stops; upload failures counted. | `appview/internal/middleware/rate_limit_test.go`, route integration tests |
| IT-006 | FR-004, FR-005, FR-012, FR-013, NFR-001 | AC-011, AC-012, AC-013, AC-016 | Body policy through mux before logging/handler. | Mux with default JSON route, upload override route, no-body route, body-reading logger spy. | Send at-limit, over-limit, override-size, and no-body-with-body requests. | Correct pass/reject decisions and standard 413/request_body_not_allowed envelopes before expensive reads. | `appview/internal/middleware/body_limit_test.go`, `routes_test.go` |
| IT-007 | BR-003, FR-010, FR-011, RULE-003 | AC-009, AC-010 | CORS policy with production/dev config. | Load prod config with `https://app.craftsky.social`; dev config with wildcard/localhost. | Send actual and preflight browser requests. | Exact prod origin allowed; marketing/preview/evil not allowed; supported headers include Authorization, Content-Type, X-Craftsky-Device-Id; no credentials. | `appview/internal/middleware/cors_test.go`, `appview/internal/app/config_test.go` |
| IT-008 | FR-014 | AC-017 | Preflight does not consume route-class limiter. | CORS before limiter with low route limit and test counters. | Send several allowed OPTIONS preflights, then a normal route request. | Preflights succeed/short-circuit; normal request still has unused bucket. | `appview/internal/middleware/cors_test.go`, `rate_limit_test.go` |
| IT-009 | NFR-003, RULE-008 | AC-014, AC-021 | Deployment/config guidance for process-local limiter and defaults. | Load config/docs generated for dev/prod. | Omit limit envs; supply invalid envs; inspect documented deployment note. | Defaults match requirements; invalid config fails; single-instance/shared-storage warning exists. | `appview/internal/app/config_test.go`, docs/config test if available |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Existing `/v1/*` successful response bodies are endpoint-specific bare JSON. | BR-001, RULE-001 | Extend existing response tests so `whoami`, timeline/list, profile/post/facet responses do not gain a synthetic `data` wrapper. |
| REG-002 | Existing error envelopes use `{error, message, requestId}` with optional `fields`. | FR-001, RULE-002 | Keep `envelope.WriteError` tests and route error tests passing after helper/middleware changes. |
| REG-003 | Existing device ID validation accepts `[A-Za-z0-9_-]{1,128}` and rejects invalid values. | FR-003 | Preserve `device_id_test.go` cases while integrating rate-limit device keys. |
| REG-004 | Existing registered authenticated routes remain behind auth/device middleware and are not accidentally exempt from rate limiting. | BR-002, FR-006, RULE-004 | Route table contract test fails if any `/v1/*` route lacks explicit policy. |
| REG-005 | Existing CORS exact-origin behavior stays allow-list based while production forbids wildcard/broad reflection. | BR-003, RULE-003 | Expand `cors_test.go` for app origin allowed, marketing origin denied, and prod wildcard rejected. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Success response contract samples | `/v1/whoami` fake DID/handle, representative paginated response with `items`/`cursor`, created-resource response. | AT-001, IT-001, REG-001 |
| TD-002 | Error envelope samples | Validation error with fields, missing auth, missing/invalid device, oversize body, rate limit rejection. | AT-002, IT-002, REG-002 |
| TD-003 | Device IDs | Valid: `dev_ABC-123`, 128-character value; invalid: missing, empty, `has spaces`, 129-character value. | AT-003, UT-004, IT-003, REG-003 |
| TD-004 | CORS origins | `https://app.craftsky.social`, `https://craftsky.social`, `https://preview.craftsky.social`, `https://evil.example`, localhost/dev wildcard, no Origin. | AT-006, UT-005, IT-007, REG-005 |
| TD-005 | Body sizes | Empty body, 1 MiB JSON body, 1 MiB + 1 byte JSON body, upload-sized body below override, body above upload override. | AT-007, AT-008, AT-009, UT-008, UT-009, IT-006 |
| TD-006 | Rate-limit identities | Tokens `token-a`, `token-b`; devices `device-a`, `device-b`; fake clock windows for minute/hour limits. | AT-005, AT-012, UT-007, IT-005 |
| TD-007 | Route policy fixture | All registered `/v1/*` methods/paths with expected auth/read/write/expensive/search/upload/dev/exempt class and body policy. | AT-004, UT-006, IT-004, REG-004 |
| TD-008 | Config defaults | JSON body limit `1 MiB`; auth `10/min/device`; read `300/min/token`, `600/min/device`; write/search `60/min/token`, `120/min/device`; upload `100/hour/token`, `200/hour/device`. | UT-007, UT-012, IT-005, IT-009 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | NFR-001, NFR-002, NFR-004 | Middleware ordering and log redaction review. | Review final route/middleware composition and test logs for oversized/rate-limited requests. | Body limits run before debug body-copying; rate limits run before expensive handler work; logs omit bearer tokens and full rejected bodies. |
| MAN-002 | RULE-006, RULE-008 | Deployment constraint and no-IP-key review. | Review config/docs/deployment notes after implementation. | AppView v1 limiter is documented as process-local/single-instance; multi-instance requires shared limiter or edge enforcement; no AppView limiter key uses client IP. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Full production traffic suitability of numeric rate-limit defaults cannot be proven by unit tests. | BR-002, NFR-003 | Defaults are launch guesses and may false-positive or be too loose under real usage. | Keep values configurable; monitor 429s; revisit before/after beta launch. |
| GAP-002 | Preflight-flood abuse is not mitigated by AppView route-class limiter by requirement. | FR-014 | AC-017 explicitly exempts successful preflights from normal buckets. | Document reliance on exact-origin checks and future edge/proxy controls. |
| GAP-003 | Process-local limiter behavior across multiple replicas is intentionally not automated. | RULE-008 | V1 assumes single AppView instance; multi-instance needs different architecture. | Require explicit approval/shared limiter before horizontal scaling. |
| GAP-004 | Log redaction assertions may not cover all production logger sinks/formatters. | NFR-004 | Tests can capture known logger output but not every deployment sink. | Manual review plus structured logging convention check. |

## 10. Out Of Scope

- Implementing shared/distributed rate-limit storage for multi-instance AppView deployments.
- IP-based AppView rate-limit keys or login IP throttling; these remain edge/proxy responsibilities.
- Changing `/oauth/*`, `/health`, or `/healthz` endpoint contracts except generic CORS preflight behavior where outer middleware applies.
- Wrapping successful `/v1/*` responses in `{ "data": ... }`.
- Flutter UI changes beyond client compatibility with existing bare success bodies and standard 413/429 errors.
- Lexicon changes, migrations, or persistent rate-limit storage unless a later implementation decision changes scope.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-06-28-appview-architecture-hardening/01-requirements.md`
- Test specification: `docs/changes/2026-06-28-appview-architecture-hardening/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this agent: `docs/changes/2026-06-28-appview-architecture-hardening/`
- Risk level: High
- Review recommendation: Required before implementation because the scope affects API contracts, CORS/security posture, body limits, logging safety, and abuse controls.
- Recommended first failing test for implementation: `UT-006`/`IT-004` route policy contract requiring every `/v1/*` route to declare rate-limit class and body policy; this creates the guardrail for all later middleware behavior.
- Suggested test order for implementation:
  1. `UT-006` and `IT-004` route policy classification contract.
  2. `UT-008`, `UT-009`, `UT-010`, then `IT-006` body-limit/no-body behavior.
  3. `UT-007`, then `IT-005` rate-limit keying and 429 behavior.
  4. `UT-005`, `IT-007`, and `IT-008` CORS and preflight ordering.
  5. `UT-012` and `IT-009` config defaults, invalid config, and deployment warnings.
  6. `AT-001`/`REG-001` and `AT-002`/`REG-002` final success/error contract regression sweep.
  7. Manual checks `MAN-001` and `MAN-002` before implementation sign-off.
- Commands discovered:
  - `just dev-d`
  - `just test`
  - `just fmt`
- Blocking gaps: None. Non-blocking gaps are recorded in Section 9.
