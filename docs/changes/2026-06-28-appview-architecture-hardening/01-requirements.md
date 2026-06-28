# Requirements: AppView Architecture Hardening

## 1. Initial Request

Tackle the AppView roadmap items that are all related to API architecture hardening before launch:

- Request body size limits.
- Cross-cutting envelope helpers and device-id middleware.
- Rate limiting per token and per device ID.
- CORS policy.
- Success response envelope decision.

The requester asked for guidance on best practices and confirmed these decisions during discovery:

- Keep successful `/v1/*` JSON responses as bare endpoint-specific bodies for v1; do not wrap them in `{ "data": ... }`.
- Use route-class rate limiting rather than a single global limit.
- Configure production CORS for the known Craftsky web app origin because a web version may arrive soon.
- Use a global JSON request-body size limit with explicit endpoint overrides.
- Use process-local AppView rate limiting for v1, with single-instance deployment as an explicit constraint.
- Use initial configurable rate/body defaults recorded in this document.

## 2. Current Codebase Findings

- Relevant files:
  - `docs/roadmap.md` lines 25-29 list the requested AppView architecture items.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` establishes `/v1/`, auth headers, device ID, error envelope, pagination, and marks body limits/rate limiting/CORS/success envelope as follow-up work.
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md` specifies camelCase JSON, error-envelope helpers, and device-id middleware expectations.
  - `appview/internal/routes/routes.go` composes `Authenticated` and `DeviceID` middleware on `/v1/*` routes.
  - `appview/internal/api/envelope/envelope.go` provides the canonical error envelope helper.
  - `appview/internal/api/envelope/cursor.go` provides opaque cursor helpers for paginated bare success bodies.
  - `appview/internal/middleware/device_id.go` enforces `X-Craftsky-Device-Id` format and stores it in context.
  - `appview/internal/middleware/cors.go` implements an allow-list CORS middleware.
  - `appview/internal/middleware/logging.go` currently reads JSON request bodies for debug logging and restores them.
  - `appview/internal/app/config.go` already loads `ALLOWED_ORIGINS` and media upload size limits.
- Existing patterns:
  - Middleware constructors return `func(http.Handler) http.Handler`.
  - `/v1/*` errors use `{error, message, requestId}` with optional `fields`.
  - Successful responses currently use endpoint-specific bare JSON shapes.
  - Device ID validation accepts `^[A-Za-z0-9_-]{1,128}$`.
  - CORS supports exact origins plus `*` for dev-like wildcard behavior.
- Current behavior:
  - Device ID middleware and error helpers exist, but the roadmap still needs the architecture contract finalized and any gaps covered by tests/implementation.
  - General JSON request body size limits are not visible as a cross-cutting policy.
  - Rate limiting is not visible in the inspected code.
  - CORS exists, but the production policy needs to be made launch-ready for near-term web clients.
  - Success envelope policy is not documented as a final v1 decision.
- Constraints discovered:
  - Flutter app talks to AppView over JSON/HTTP and never directly to the PDS for normal reads.
  - `/oauth/*`, `/health`, and `/healthz` are not part of the `/v1/*` app API surface.
  - `/v1/*` JSON uses camelCase.
  - API breaking changes after launch require `/v2/`; success response shape must be locked before launch.
  - Implementation should use Go stdlib `net/http` patterns and existing middleware style.
- Test/build commands discovered:
  - Repository guidance says `just test` runs Go tests against the compose Postgres.
  - Existing Go tests live under `appview/internal/**` and include middleware, routes, auth, and API handler coverage.

## 3. Clarifying Questions And Decisions

### Q1: Should successful `/v1/*` JSON responses be wrapped in `{ "data": ... }`?

Answer: Leave as bare success bodies.

Decision / implication: v1 shall retain endpoint-specific success shapes such as `{ "did": "..." }`, `{ "items": [...] }`, and created-resource objects. Only errors use the standard error envelope. This avoids churn in existing AppView and Flutter contracts and must be documented as the launch contract to avoid a later `/v2/` just for wrapping.

### Q2: What rate-limiting model should v1 use?

Answer: Route-class policy.

Decision / implication: Rate limits shall be configurable by route class, including at least auth, read, write, expensive/search, and upload classes, while enforcing both per-token and per-device counters where identity is available. V1 shall use shared buckets per route class rather than per-endpoint buckets.

### Q3: What production CORS posture should v1 use?

Answer: Allow the known Craftsky web app origin because a web version may come very soon.

Decision / implication: Production CORS shall be explicit allow-list based and support `https://app.craftsky.social` as the initial authenticated web app origin. `https://craftsky.social` is the marketing/landing origin and shall not be allowed to call authenticated AppView APIs unless a future decision changes that. Production CORS shall not allow arbitrary origins, wildcard subdomains, or preview-domain patterns.

### Q4: How should request body size limits be scoped?

Answer: Use a global JSON limit plus explicit endpoint overrides.

Decision / implication: AppView shall enforce a conservative default body limit for JSON requests and allow route-specific overrides for endpoints with different needs, especially image/blob upload.

### Q5: Should rate limiting be shared across instances or process-local for v1?

Answer: Use process-local AppView rate limiting for v1 and explicitly constrain that to single-AppView-instance deployment.

Decision / implication: V1 may use in-memory/process-local counters while production runs one AppView instance. Multi-instance deployment requires shared limiter storage or equivalent edge enforcement before scaling horizontally.

### Q6: What initial body-size and rate-limit defaults should be documented?

Answer: Lock initial configurable defaults.

Decision / implication: Defaults shall be: JSON body limit `1 MiB`; auth/login `10 requests/minute per device ID`; read routes `300 requests/minute per token` and `600 requests/minute per device ID`; write routes `60 requests/minute per token` and `120 requests/minute per device ID`; expensive/search routes `60 requests/minute per token` and `120 requests/minute per device ID`; upload routes `100 requests/hour per token` and `200 requests/hour per device ID`.

### Q7: What exact body-limit error should be used?

Answer: Use HTTP 413 with error code `request_body_too_large`.

Decision / implication: Oversized body responses shall use the standard error envelope and message `request body exceeds the configured limit`.

### Q8: Should production CORS use exact origins only?

Answer: Yes.

Decision / implication: Production CORS shall use exact-origin matching only. Wildcard subdomains and preview-domain patterns are out of scope; additional origins must be explicitly configured.

### Q9: Should v1 CORS send `Access-Control-Allow-Credentials: true`?

Answer: No, not unless a future cookie-based web auth design requires it.

Decision / implication: Browser API calls shall rely on `Authorization` headers, not cookie credentials, for v1.

### Q10: What should 429 responses expose?

Answer: Use `rate_limited` plus `Retry-After`, without public quota/bucket details.

Decision / implication: V1 shall not expose public `X-RateLimit-*` headers or bucket names. Internal logs may include route class and key type without raw token/device values.

### Q11: How should dev-only `/v1/dev/*` routes be handled?

Answer: They still require explicit classification.

Decision / implication: Dev-only routes may use a documented dev-only exemption or relaxed class, and tests should verify those routes cannot appear in production unexpectedly.

### Q12: How trustworthy is `X-Craftsky-Device-Id`?

Answer: Treat it as an untrusted client-supplied signal, not identity.

Decision / implication: Device ID may contribute to abuse controls but shall not be used as authorization evidence. Token-based limits remain authoritative for authenticated routes.

### Q13: What middleware ordering is required for body limits?

Answer: Body-size limiting must run before any middleware that reads or copies request bodies, including debug logging.

Decision / implication: Oversized body protection must prevent large in-memory reads and must not log full rejected bodies.

### Q14: Should no-body routes reject unexpected bodies?

Answer: Yes.

Decision / implication: `/v1/*` routes that do not declare a request body, including GET and default DELETE routes, shall reject non-empty bodies with error code `request_body_not_allowed` unless the endpoint explicitly allows a body.

### Q15: Should CORS preflight requests be rate limited by route-class limits?

Answer: No.

Decision / implication: Successful CORS preflight handling is exempt from normal AppView route-class rate limits. Exact-origin checks, preflight caching, and future edge/proxy controls handle preflight abuse.

### Q16: Should AppView v1 use IP-based rate-limit keys?

Answer: No.

Decision / implication: AppView v1 shall avoid IP-based rate-limit keys. Login IP throttling is a deployment edge/proxy responsibility outside this AppView requirements scope.

### Q17: Should upload limits count attempts or only successes?

Answer: Count attempts.

Decision / implication: Upload rate limits count attempted upload requests, not just successful uploads.

## 4. Candidate Approaches

### Option A: Minimal Hardening Around Current Behavior

Summary: Keep bare success bodies, add a global JSON body limit, configure existing CORS, and add simple global rate limits.

Pros:
- Lowest implementation churn.
- Easy to explain and test.
- Preserves current API shapes.

Cons:
- Simple rate limits may not protect expensive endpoints well.
- Less future-proof for web clients and abuse controls.
- May need rapid rework before public launch.

Risks:
- Attackers or buggy clients may exhaust resources through search, uploads, or writes before hitting a broad global limit.

### Option B: Launch-Ready Route-Class Hardening

Summary: Keep bare success bodies, define a global JSON body limit with overrides, formalize device-id/error-envelope helper expectations, implement process-local shared route-class rate limits per token and per device, and lock production CORS to the exact web app origin.

Pros:
- Aligns with existing API shape while addressing launch risks.
- Matches different abuse profiles for auth, reads, writes, search, and uploads.
- Supports a near-term web client without permissive production CORS.
- Gives test-design clear route classes and expected errors.

Cons:
- More configuration and test matrix than a single global limiter.
- Requires careful route classification to avoid accidental unprotected routes.
- Requires client-facing error behavior for 413 and 429 to be consistent.

Risks:
- Misconfigured limits could block legitimate usage or fail open.

### Option C: Success Envelope And Broad API Normalization

Summary: Wrap all success responses in `{ "data": ... }` while also adding limits, CORS, and rate limiting.

Pros:
- Highly consistent top-level success shape.
- Leaves obvious room for future metadata.
- Good moment to do it before public launch if wanted.

Cons:
- Conflicts with user-confirmed direction.
- Requires changing existing AppView handlers, Flutter models, and tests.
- Adds nesting to simple responses and paginated lists.

Risks:
- Large contract churn could delay launch and introduce regressions.

## 5. Recommended Direction

Recommended approach: Option B, launch-ready route-class hardening while preserving bare success bodies.

Why: This direction addresses the actual launch risks — oversized bodies, abuse limits, browser-origin policy, and cross-cutting response/header consistency — without creating unnecessary API-shape churn. It also supports a likely near-term web client through exact production CORS allow-listing for `https://app.craftsky.social` rather than a permissive policy.

## 6. Problem / Opportunity

The AppView has enough endpoint surface that cross-cutting API behavior must be locked before public launch. Without a documented and tested contract for body limits, rate limits, CORS, device IDs, and response envelope shape, clients may depend on inconsistent behavior, abusive traffic may be harder to control, and later fixes could require a breaking `/v2/` API change.

## 7. Goals

- G-001: Lock the v1 success response contract before public clients depend on it.
- G-002: Protect AppView from oversized request bodies in a consistent, testable way.
- G-003: Add launch-ready AppView rate limiting that distinguishes route risk classes and keys by token/device where possible, with documented v1 single-instance constraints.
- G-004: Define a production CORS policy that supports `https://app.craftsky.social` without allowing arbitrary origins.
- G-005: Make cross-cutting helper and middleware expectations clear enough that future endpoints do not bypass them accidentally.

## 8. Non-Goals

- NG-001: Do not change public record lexicons.
- NG-002: Do not change OAuth protocol behavior under `/oauth/*` except insofar as outer server middleware may handle generic HTTP concerns like CORS preflight where applicable.
- NG-003: Do not wrap successful v1 responses in `{ "data": ... }`.
- NG-004: Do not design a full observability/metrics/tracing system; only require minimal logging/metrics hooks needed for rate-limit/body-limit/CORS behavior.
- NG-005: Do not add user-facing Flutter UI changes beyond whatever client compatibility is needed for unchanged bare success bodies and standard errors.
- NG-006: Do not rely solely on reverse proxy/CDN enforcement for body size or rate limiting.
- NG-007: Do not implement AppView IP-based rate-limit keys in v1; IP throttling is an edge/proxy responsibility.
- NG-008: Do not allow `https://craftsky.social` to call authenticated AppView APIs unless a future requirements decision changes that.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Flutter client | Current first-party mobile app using `/v1/*` JSON APIs. | Stable success shapes, standard errors, predictable body/rate-limit failures. |
| Future Craftsky web client | Anticipated browser-based first-party client. | Explicitly allowed production origins and credential/header support. |
| AppView operator | Person deploying and maintaining AppView. | Configurable limits, safe defaults, and logs/metrics for rejected requests. |
| AppView developer | Contributor adding or changing handlers. | Clear middleware/helper requirements and tests that catch bypasses. |
| Abusive or buggy client | Client sending too many or too-large requests. | Should be throttled or rejected without degrading service. |

## 10. Current Behavior

AppView already has `/v1/*` route registration, error-envelope helpers, device-id middleware, and CORS middleware. Successful responses are currently endpoint-specific bare JSON bodies. Image uploads have media-specific size configuration. However, there is no finalized requirements contract for the requested architecture-hardening items, no visible cross-cutting JSON body limit policy, no visible AppView rate limiter, and CORS/success-envelope decisions remain on the roadmap rather than locked for launch.

## 11. Desired Behavior

AppView v1 should have a documented, testable cross-cutting API contract: successful responses remain bare endpoint-specific JSON, errors use the standard envelope, body sizes are limited by default with explicit overrides, route-class rate limits enforce both per-token and per-device controls where identity is available, CORS allows only the configured Craftsky web app origin in production, and all v1 routes follow consistent middleware/helper expectations.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | AppView v1 shall lock successful JSON responses as bare endpoint-specific bodies rather than `{ "data": ... }` wrappers. | Avoids a later breaking `/v2/` change and preserves existing client/server shape. | User answer Q1; API architecture open question | AC-001, AC-002 |
| BR-002 | Business | Must | AppView shall provide launch-ready abuse protection for `/v1/*` through route-class rate limits. | Public launch requires protection against abusive or buggy clients. | User answer Q2; roadmap | AC-006, AC-007, AC-008 |
| BR-003 | Business | Must | AppView production CORS shall support `https://app.craftsky.social` without allowing arbitrary origins or the marketing origin `https://craftsky.social`. | A web version may come soon, but permissive CORS and marketing-origin API access are not launch-safe. | User answers Q3, Q8 | AC-009, AC-010 |
| FR-001 | Functional | Must | The system shall continue returning standard error envelopes with `error`, `message`, and `requestId`, plus optional `fields`, for `/v1/*` error responses. | Existing API contract and client error handling depend on this shape. | Codebase; API architecture spec | AC-003, AC-004 |
| FR-002 | Functional | Must | The system shall expose and require shared response helpers for `/v1/*` handlers so JSON success and error responses consistently set status and `Content-Type`. | Prevents handlers from drifting or bypassing error-envelope conventions. | Codebase; roadmap cross-cutting helpers item | AC-003, AC-004 |
| FR-003 | Functional | Must | The system shall enforce `X-Craftsky-Device-Id` on applicable `/v1/*` routes using the established validation rule of 1-128 characters from `[A-Za-z0-9_-]`. | Device ID is needed for session instrumentation and per-device rate limiting. | Existing middleware; API architecture spec | AC-005, AC-007 |
| FR-004 | Functional | Must | The system shall enforce a default `1 MiB` request body size limit for JSON request bodies. | Protects memory and CPU from oversized JSON bodies. | User answers Q4, Q6 | AC-011, AC-012 |
| FR-005 | Functional | Must | The system shall support explicit per-route body size overrides for endpoints whose expected payloads differ from the default, including blob/image upload routes. | Uploads and other large-body endpoints need limits suited to their purpose. | User answer Q4; existing media limits | AC-013 |
| FR-006 | Functional | Must | The system shall classify `/v1/*` routes into shared rate-limit classes including at least auth, read, write, expensive/search, and upload. | Different route types have different cost and abuse profiles. Shared class buckets avoid endpoint rotation bypasses. | User answers Q2, Q6 | AC-006 |
| FR-007 | Functional | Must | The system shall enforce rate limits per Craftsky session token for authenticated requests. | A valid token should not be able to abuse the API independently of device ID. | Roadmap; user answer Q2 | AC-007, AC-008 |
| FR-008 | Functional | Must | The system shall enforce rate limits per `X-Craftsky-Device-Id` where the device ID is available. | Device-level throttling is needed for login and multi-token abuse scenarios. | Roadmap; user answer Q2 | AC-007, AC-008 |
| FR-009 | Functional | Must | When a request is rate limited, the system shall return HTTP 429 with error code `rate_limited`, use the standard error envelope, and include `Retry-After`. | Clients need predictable throttling behavior without exposing bucket details to attackers. | User answer Q10; roadmap | AC-008 |
| FR-010 | Functional | Must | Production CORS shall allow only exact configured origins and shall include the headers/methods needed by first-party web clients, including `Authorization`, `Content-Type`, and `X-Craftsky-Device-Id`. | Browser clients need CORS support for authenticated API calls. | User answers Q3, Q8; existing CORS code | AC-009, AC-010 |
| FR-011 | Functional | Should | Development CORS may allow localhost or wildcard origins when explicitly configured. | Keeps local development convenient without weakening production. | Existing CORS code; discovery recommendation | AC-010 |
| FR-012 | Functional | Must | Oversized request bodies shall be rejected with HTTP 413, error code `request_body_too_large`, and message `request body exceeds the configured limit`. | Clients and tests need a stable oversize error contract. | User answer Q7 | AC-011 |
| FR-013 | Functional | Must | Routes that do not declare a request body shall reject non-empty bodies with error code `request_body_not_allowed`, unless an endpoint explicitly allows a body. | Prevents clients from depending on ignored or ambiguous request bodies. | User answer Q14 | AC-016 |
| FR-014 | Functional | Must | CORS preflight requests that pass the CORS policy shall be handled before normal route-class rate limiting. | Browser preflights are unauthenticated and should not consume token/device route buckets. | User answer Q15 | AC-017 |
| NFR-001 | Non-functional | Must | Body-limit enforcement shall occur before handlers parse request bodies or perform expensive work, and before any middleware reads or copies request bodies. | Limits must protect resource usage, not merely validate after work is done; debug logging currently reads JSON bodies. | User answer Q13; codebase finding | AC-011, AC-012, AC-015 |
| NFR-002 | Non-functional | Must | Rate-limit checks shall occur early in the middleware chain after enough identity information is available to choose the correct key. | Prevents expensive handler work for requests that should be throttled. | Discovery | AC-007, AC-008 |
| NFR-003 | Non-functional | Should | Rate-limit and body-limit behavior shall be configurable with safe defaults suitable for local development and production deployment. | Operators need tuning without source changes. | Discovery; app config pattern | AC-014 |
| NFR-004 | Non-functional | Should | Limit rejections shall avoid logging full sensitive request bodies or bearer tokens. | Prevents sensitive data exposure during abuse events. | Security review | AC-015 |
| RULE-001 | Business rule | Must | Successful `/v1/*` responses shall not be wrapped in `{ "data": ... }` for v1. | User explicitly chose bare success bodies. | User answer Q1 | AC-001, AC-002 |
| RULE-002 | Business rule | Must | Error responses shall remain enveloped even though success responses are bare. | Distinguishes operational errors from domain resources while preserving support diagnostics. | Existing API architecture | AC-003 |
| RULE-003 | Business rule | Must | Production CORS shall not use `*` or broad origin reflection. | Prevents arbitrary browser origins from calling credentialed APIs. | User answer Q3; security best practice | AC-009 |
| RULE-004 | Business rule | Must | Every registered `/v1/*` route shall have an explicit body-limit and rate-limit classification, even if its limit class is `none` or `no-body`. | Prevents new routes from silently bypassing cross-cutting policy. | Discovery | AC-006, AC-013 |
| RULE-005 | Business rule | Must | `X-Craftsky-Device-Id` shall be treated as an untrusted client-supplied abuse-control signal, not as identity or authorization evidence. | Device IDs can be spoofed or rotated. | User answer Q12 | AC-018 |
| RULE-006 | Business rule | Must | AppView v1 shall not use IP addresses as rate-limit keys. | Avoids privacy and proxy-header complexity in AppView v1. | User answer Q16 | AC-019 |
| RULE-007 | Business rule | Must | Upload rate limits shall count attempted upload requests, not only successful uploads. | Failed uploads still consume resources and can be abusive. | User answer Q17 | AC-020 |
| RULE-008 | Business rule | Must | V1 route-class rate limiting may be process-local only while AppView runs as a single instance. | Process-local limits are not globally enforceable across replicas. | User answer Q5 | AC-021 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, RULE-001 | Given any successful `/v1/*` JSON endpoint response, when the response body is decoded, then the top-level body is the endpoint-defined resource/page/action shape and is not wrapped in a top-level `data` field solely for envelope purposes. |
| AC-002 | BR-001, RULE-001 | Given a paginated successful response, when the body is decoded, then pagination fields such as `items` and `cursor` remain in the endpoint-defined bare shape unless the endpoint's own contract says otherwise. |
| AC-003 | FR-001, FR-002, RULE-002 | Given a `/v1/*` handler returns a client or server error, when the response is received, then it has `Content-Type: application/json` and a body containing `error`, `message`, and `requestId`, with `fields` only when applicable. |
| AC-004 | FR-001, FR-002 | Given representative `/v1/*` handlers, when tests inspect emitted JSON errors, then handlers use or conform to the shared envelope helper behavior rather than custom incompatible shapes. |
| AC-005 | FR-003 | Given an applicable `/v1/*` request without `X-Craftsky-Device-Id` or with a malformed value, when it is processed, then the system rejects it with HTTP 400 and the standard error envelope before the endpoint handler runs. |
| AC-006 | BR-002, FR-006, RULE-004 | Given the route table, when route classifications are inspected by tests or review, then each `/v1/*` route has an explicit rate-limit class of auth, read, write, expensive/search, upload, or an intentionally documented exempt/no-body class. |
| AC-007 | BR-002, FR-003, FR-007, FR-008, NFR-002 | Given authenticated requests with a Craftsky token and device ID, when the client exceeds the configured class limit for either key, then further requests in that class are rejected before handler work proceeds. |
| AC-008 | BR-002, FR-007, FR-008, FR-009, NFR-002 | Given a request is rejected by rate limiting, when the response is received, then it is HTTP 429, uses error code `rate_limited`, uses the standard error envelope, includes `Retry-After`, and does not expose public bucket/quota details. |
| AC-009 | BR-003, FR-010, RULE-003 | Given production configuration, when a browser request comes from `https://app.craftsky.social`, then CORS headers allow it; when it comes from `https://craftsky.social`, wildcard subdomains, preview patterns, or any unconfigured origin, then the response does not allow that origin. |
| AC-010 | BR-003, FR-010, FR-011 | Given a CORS preflight from an allowed web origin, when it requests supported methods and headers including `Authorization`, `Content-Type`, and `X-Craftsky-Device-Id`, then the preflight succeeds with appropriate CORS headers. |
| AC-011 | FR-004, FR-012, NFR-001 | Given a JSON request body larger than the default `1 MiB` limit to a default-limited route, when the request is processed, then the system rejects it with HTTP 413, error code `request_body_too_large`, message `request body exceeds the configured limit`, and the standard error envelope before JSON parsing, endpoint work, or body-copying debug logging. |
| AC-012 | FR-004, NFR-001 | Given a JSON request body at or below the default configured limit, when the request is otherwise valid, then body-limit middleware does not reject it. |
| AC-013 | FR-005, RULE-004 | Given an endpoint with an explicit body-size override, when requests are below and above that override, then the endpoint accepts/rejects according to the override rather than the default JSON limit. |
| AC-014 | NFR-003 | Given AppView starts in dev or prod, when limit-related configuration is omitted or supplied, then safe defaults are applied and invalid configuration fails startup with a clear error. |
| AC-015 | NFR-001, NFR-004 | Given a request is rejected for size or rate limiting, when logs are inspected, then logs include enough context for diagnosis without logging bearer tokens or full rejected request bodies. |
| AC-016 | FR-013 | Given a `/v1/*` no-body route receives a non-empty body, when the request is processed, then the system rejects it with error code `request_body_not_allowed` unless the endpoint explicitly allows a body. |
| AC-017 | FR-014 | Given an allowed CORS preflight request, when it is processed, then it is handled by CORS policy and does not consume normal route-class rate-limit buckets. |
| AC-018 | RULE-005, FR-003, FR-008 | Given a request includes a valid device ID, when authorization decisions are made, then the device ID is not used as proof of identity or permission. |
| AC-019 | RULE-006 | Given AppView rate-limit keys are inspected, when v1 limits are configured, then no limiter key is based on client IP address. |
| AC-020 | RULE-007 | Given upload requests exceed the upload class limit through failed or successful attempts, when another upload is attempted, then it is rate limited even if prior attempts did not succeed. |
| AC-021 | RULE-008 | Given the v1 limiter is process-local, when deployment guidance/configuration is reviewed, then it explicitly states that multi-instance AppView deployments require shared limiter storage or equivalent edge enforcement before horizontal scaling. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Request has no body on a no-body route. | Body-limit middleware does not reject solely due to missing body. | FR-004, RULE-004 |
| EC-002 | Request has `Content-Type` missing or non-JSON but targets a JSON endpoint. | Existing endpoint validation handles content semantics; size limits still protect the raw body where configured. | FR-004 |
| EC-003 | Body exceeds the limit before JSON decoding. | Return HTTP 413 with standard error envelope; do not attempt handler-level parsing. | FR-004, NFR-001 |
| EC-004 | Authenticated route has valid token but missing device ID. | Return HTTP 400 `missing_device_id` before rate-limit keys that require device ID are evaluated. | FR-003, FR-008 |
| EC-005 | Login route has no Craftsky token yet. | Apply auth-route and per-device rate limiting using device ID and any other configured pre-auth key; do not require token. | FR-006, FR-008 |
| EC-006 | Multiple valid tokens use the same device ID. | Per-device limit can throttle aggregate traffic even when per-token limits are not individually exceeded. | FR-008 |
| EC-007 | Same token appears from multiple device IDs. | Per-token limit can throttle aggregate session traffic even when per-device limits are not individually exceeded. | FR-007 |
| EC-008 | CORS request has no `Origin` header, such as mobile/native or server-to-server traffic. | Do not add allow-origin headers solely for CORS; process according to normal auth and route rules. | FR-010 |
| EC-009 | Production config accidentally contains wildcard origin. | Startup should reject it or runtime should refuse wildcard behavior in production. | RULE-003, NFR-003 |
| EC-010 | New `/v1/*` route is added without policy classification. | Tests or registration helpers fail, forcing explicit body-limit and rate-limit classification. | RULE-004 |
| EC-011 | Marketing origin `https://craftsky.social` calls authenticated API. | Production CORS does not allow the origin unless a future requirements change explicitly adds it. | BR-003, FR-010 |
| EC-012 | CORS preflight flood. | Normal route-class limits do not apply; exact-origin checks and future edge/proxy controls are the mitigation. | FR-014 |
| EC-013 | Client rotates device IDs to avoid login limits. | AppView v1 does not use IP keys; edge/proxy IP throttling is documented as out-of-scope responsibility. | RULE-005, RULE-006 |
| EC-014 | Upload request fails validation but reaches the upload limiter. | The attempt counts against upload route-class limits. | RULE-007 |

## 15. Data / Persistence Impact

- New fields: None required by this requirements stage. Device ID persistence already exists in the inspected code via session touch behavior.
- Changed fields: None required.
- Migration required: None expected for the documented requirements unless implementation chooses persistent/distributed rate-limit storage.
- Backwards compatibility:
  - Successful response bodies stay compatible with the current bare response shape.
  - New 413 and 429 errors add standardized failure modes that clients should handle through existing API error handling.

## 16. UI / API / CLI Impact

- UI:
  - No direct UI requirement.
  - Flutter/web clients may need to gracefully handle 413 and 429 standard error envelopes.
- API:
  - `/v1/*` success responses remain bare.
  - `/v1/*` errors remain enveloped.
  - Oversized requests return 413.
  - Rate-limited requests return 429 with retry guidance.
  - CORS allows configured first-party web origins and headers.
  - No-body routes reject unexpected bodies with `request_body_not_allowed`.
- CLI:
  - No direct CLI impact.
- Background jobs:
  - No direct background job impact.

## 17. Security / Privacy / Permissions

- Authentication:
  - Authenticated requests continue to require `Authorization: Bearer <craftsky-session-token>`.
  - Rate limiting shall use token identity after authentication where available.
- Authorization:
  - No endpoint authorization semantics change in this scope.
- Sensitive data:
  - Logs for rejected requests must not include bearer tokens or full oversized bodies.
  - Device IDs are client-generated identifiers and should be treated as operational identifiers, not as proof of user identity.
- Abuse cases:
  - Oversized JSON bodies.
  - High-volume reads.
  - High-volume writes.
  - Expensive search/facet traffic.
  - Upload attempts.
  - Login/pre-auth hammering by device ID.
  - Cross-origin browser calls from unapproved origins.
- Device IDs are spoofable and must not be treated as authentication or authorization evidence.
- AppView v1 intentionally avoids IP-based rate-limit keys; deployment edge/proxy controls own IP throttling.

## 18. Observability

- Events:
  - No analytics events required.
- Logs:
  - Log body-limit rejections with route/method/status/run ID and limit class where available.
  - Log rate-limit rejections with route class, key type, status, and run ID, without raw token values.
  - Log CORS denials at debug or low-noise level if useful for diagnosing web rollout.
- Metrics:
  - Should expose or make straightforward to add counters for 413 and 429 by route class.
  - Should expose or make straightforward to add counters for CORS preflight/denial outcomes if the project already has metrics infrastructure available.
- Alerts:
  - None required in this scope.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Rate limits are too strict for legitimate users. | Users may see false 429s during normal use. | Use the documented configurable defaults, route classes, retry guidance, and monitoring. |
| RISK-002 | Rate limits are too loose or missing on some routes. | AppView remains vulnerable to abuse or high-cost traffic. | Require explicit route classification and tests covering the route table. |
| RISK-003 | Body limits break legitimate payloads. | Users may be unable to create posts/profiles/uploads. | Use default JSON limit plus explicit overrides and acceptance tests around known body sizes. |
| RISK-004 | Production CORS is misconfigured for the upcoming web client. | Browser clients fail despite API being healthy. | Require explicit allowed origins and preflight tests for supported headers/methods. |
| RISK-005 | Permissive CORS leaks into production. | Untrusted browser origins can call APIs with user credentials/headers. | Forbid wildcard/broad reflection in production and validate config. |
| RISK-006 | Logging oversized requests leaks sensitive data or consumes memory. | Privacy/security issue and resource pressure during abuse. | Do not log full rejected bodies; ensure limit middleware runs before body-copying debug logging. |
| RISK-007 | Success body decision is forgotten in future endpoints. | New endpoints may introduce inconsistent `data` wrappers. | Document RULE-001 and add contract tests/review guidance. |
| RISK-008 | Process-local rate limiting is used with multiple AppView replicas. | Limits can be bypassed by spreading traffic across replicas. | Keep v1 single-instance or add shared limiter storage/equivalent edge enforcement before horizontal scaling. |
| RISK-009 | Device ID rotation bypasses pre-auth limits. | Login abuse may evade AppView device buckets. | Document device ID as untrusted and rely on edge/proxy IP throttling outside AppView v1. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | AppView remains the sole happy-path API for Flutter and near-term first-party web clients. | Third-party API needs might require broader CORS/API contract design. |
| ASM-002 | Bare success bodies are compatible with current Flutter client models. | If client code already expects wrappers somewhere, requirements need revision. |
| ASM-003 | In-memory/process-local rate limiting is acceptable for v1 because production runs a single AppView instance. | Multi-instance production would need shared limiter storage or equivalent edge enforcement before horizontal scaling. |
| ASM-004 | The initial production authenticated web origin is `https://app.craftsky.social`; `https://craftsky.social` remains marketing-only for API CORS. | Additional web origins, including the marketing origin or preview domains, would need to be added explicitly before browser clients on those origins can call the API. |
| ASM-005 | Existing media upload limits remain authoritative for image/blob upload ceilings. | If product upload limits change, body-limit overrides must be revisited. |
| ASM-006 | Initial rate-limit defaults are configurable and can be adjusted without source-code changes. | Hard-coded defaults would make launch tuning slower. |

## 21. Open Questions

- [ ] Non-blocking: Should login rate limiting also include a non-IP pre-auth key such as normalized handle attempted, or is per-device plus edge/proxy IP throttling enough for v1?

## 22. Review Status

Status: Draft
Risk level: High
Review recommended: Required
Reviewer: Unassigned
Date: 2026-06-28
Notes: High risk because this change covers API contracts, CORS/security posture, request size limits, and abuse controls. Explicit approval is recommended before test design/implementation proceeds.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-28-appview-architecture-hardening/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`, `BR-002`, `BR-003`
  - `FR-001` through `FR-014`
  - `NFR-001`, `NFR-002`, `NFR-003`, `NFR-004`
  - `RULE-001` through `RULE-008`
- Suggested test levels:
  - Middleware unit tests for body limits, CORS, device ID, and rate limiting.
  - Route registration/contract tests proving every `/v1/*` route has limit classifications.
  - Handler/API integration tests for 413, 429, CORS preflight, and success/error envelope shapes.
  - Configuration tests for safe defaults and invalid production CORS wildcard rejection.
  - Regression tests ensuring successful responses are not wrapped in `data`.
- Blocking open questions: None. Numeric defaults and deployment storage choices can be decided during test design/implementation review if kept configurable and covered by acceptance tests.
