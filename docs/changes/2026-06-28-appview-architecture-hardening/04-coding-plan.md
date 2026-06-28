# Coding Plan: AppView Architecture Hardening

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — approved with notes.
- Related architecture context read during planning:
  - `AGENTS.md`
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
  - `appview/cmd/appview/server.go`
  - `appview/internal/routes/routes.go`
  - `appview/internal/app/config.go`
  - `appview/internal/app/deps.go`
  - `appview/internal/middleware/{auth,cors,device_id,logging}.go`
  - `appview/internal/api/envelope/envelope.go`
  - Existing route, middleware, API, and config tests under `appview/internal/**`.

## 2. Implementation Strategy

Implement launch-ready hardening by introducing explicit `/v1/*` route policy metadata first, then deriving body-limit and rate-limit middleware composition from that metadata. This matches the document-review recommendation to start with `UT-006` / `IT-004` and prevents new routes from silently bypassing body or rate policy.

Keep the existing stdlib `net/http` routing style and middleware constructor shape (`func(http.Handler) http.Handler`). Preserve bare success response bodies and the existing `envelope.WriteError` contract. Add a small success JSON helper only if needed for consistency, and ensure it does not wrap success bodies in `{ "data": ... }`.

Server middleware ordering should become:

```text
Logging/run_id assignment, with no full request-body copy for oversized bodies
  -> CORS (preflight short-circuits before normal route limits)
    -> mux route policies, where each /v1 route applies:
       body policy before any body-copying/debug inspection or handler parsing
       auth/device/rate-limit ordering appropriate to route class
       endpoint handler
```

Because current `Logging` reads JSON request bodies before `CORS`/mux, implementation must either split run-id assignment from debug payload copying or make logging body capture bounded and policy-aware. The hard guardrail is that body-size enforcement prevents large in-memory reads and logs never include full rejected bodies or bearer tokens.

No source files, tests, dependencies, migrations, or lexicons are changed by this planning stage.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Route registration | `routes.AddRoutes` calls `mux.Handle` directly and wraps per-route `authN(deviceID(handler))`. | Introduce route policy registration helpers that record every `/v1/*` method/path with rate class and body policy, then compose middleware from policy. | BR-002, FR-006, RULE-004, FR-005 | AT-004, UT-006, IT-004, REG-004, AT-008 |
| Body limits | No visible cross-cutting JSON body limit; media upload has separate `api.MediaLimits`. | Add body policy middleware with `no-body`, default JSON `1 MiB`, and explicit upload/media override behavior. | FR-004, FR-005, FR-012, FR-013, NFR-001 | AT-007, AT-008, AT-009, UT-008, UT-009, UT-010, IT-006 |
| Rate limiting | No visible AppView limiter. | Add process-local route-class limiter keyed by token identity and device ID where available; no IP keys. | BR-002, FR-006, FR-007, FR-008, FR-009, RULE-006, RULE-007, RULE-008 | AT-005, AT-012, UT-007, IT-005, MAN-002 |
| Auth/device middleware | Existing `Authenticated` stores DID/session ID; `DeviceID` validates and stores untrusted device ID. | Preserve validation/trust boundary; arrange auth class routes to rate-limit per device without requiring token, and authenticated routes to rate-limit after token identity is available. | FR-003, RULE-005, FR-007, FR-008 | AT-003, AT-011, UT-004, UT-013, IT-003 |
| CORS | Existing allow-list middleware with dev `*`, but headers include `X-Dev-DID` and not `X-Craftsky-Device-Id`; comments mention credentials. | Keep exact-origin allow-list, support `Authorization`, `Content-Type`, `X-Craftsky-Device-Id`, avoid `Access-Control-Allow-Credentials`, reject/prohibit prod wildcard, and ensure allowed preflights short-circuit before route limits. | BR-003, FR-010, FR-011, FR-014, RULE-003 | AT-006, AT-010, UT-005, IT-007, IT-008, REG-005 |
| Response helpers | `envelope.WriteError` exists; successes are handler-specific bare JSON. | Preserve error envelope. Add optional bare `WriteJSON` helper for status/content-type consistency, without a success envelope. | BR-001, FR-001, FR-002, RULE-001, RULE-002 | AT-001, AT-002, UT-001, UT-002, UT-003, IT-001, IT-002, REG-001, REG-002 |
| Config/deployment guidance | `Config` loads env, allowed origins, media limits; `Deps` logs startup. | Add config structs/env parsing for body/rate defaults; validate prod CORS wildcard; choose startup operator warning as AC-021 artifact. | NFR-003, RULE-008, BR-003 | UT-012, IT-009, MAN-002 |
| Logging | `Logging` currently logs headers and full JSON body at debug, including possible sensitive headers. | Redact `Authorization`, do not log full rejected bodies, avoid unbounded body reads before body-limit enforcement. | NFR-001, NFR-004 | UT-011, MAN-001 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/routes/policy.go` | Create | Define `RateClass`, `BodyPolicy`, `RoutePolicy`, policy registry, registration helper, and policy inspection surface for tests. | FR-006, RULE-004, FR-005 | UT-006, IT-004, AT-004 |
| `appview/internal/routes/routes.go` | Change | Replace direct `/v1/*` `mux.Handle` calls with policy-aware registration; classify every route. Keep `/health`, `/healthz`, `/oauth/*` outside `/v1` policy scope. | RULE-004, FR-006, FR-014 | IT-004, REG-004 |
| `appview/internal/middleware/body_limit.go` | Create | Enforce no-body/default JSON/override body policies, write 413 and `request_body_not_allowed` errors. | FR-004, FR-005, FR-012, FR-013, NFR-001 | UT-008, UT-009, UT-010, IT-006 |
| `appview/internal/middleware/rate_limit.go` | Create | Process-local route-class token/device limiter with fake-clock-friendly internals and 429 envelope. | BR-002, FR-007, FR-008, FR-009, RULE-006, RULE-007 | UT-007, IT-005, AT-005, AT-012 |
| `appview/internal/middleware/cors.go` | Change | Update allowed headers, remove credential language/behavior, validate preflight short-circuit behavior. | FR-010, FR-011, FR-014, RULE-003 | UT-005, IT-007, IT-008 |
| `appview/internal/middleware/logging.go` | Change | Redact sensitive headers and avoid unbounded JSON request body logging before body policy. | NFR-001, NFR-004 | UT-011, MAN-001 |
| `appview/internal/middleware/auth.go` | Change if needed | Expose safe token/session identifier for limiter without logging/storing raw bearer token. Prefer using authenticated session ID/DID from context after auth. | FR-007, RULE-006 | UT-007, IT-005 |
| `appview/internal/middleware/device_id.go` | Change if needed | Preserve validation; ensure comments/tests clarify device ID is not authorization evidence. | FR-003, RULE-005 | UT-004, UT-013, IT-003 |
| `appview/internal/api/envelope/envelope.go` | Change | Optionally add `WriteJSON`/success helper that sets status and `Content-Type` while encoding the supplied bare body. | FR-002, BR-001, RULE-001 | UT-001, UT-003, REG-001 |
| `appview/internal/app/config.go` | Change | Add default body/rate limit config parsing, production CORS wildcard rejection, and clear env-name errors. | NFR-003, RULE-003 | UT-012, IT-009 |
| `appview/internal/app/deps.go` | Change | Wire shared process-local limiter into deps and log startup warning that limiter is process-local/single-instance only. | RULE-008, NFR-003 | IT-009, MAN-002 |
| `appview/cmd/appview/server.go` | Change | Adjust middleware comments/composition so CORS preflight is before route limits and logging/body-limit ordering is safe. | FR-014, NFR-001 | IT-006, IT-008, MAN-001 |
| `appview/internal/routes/routes_test.go` | Change tests during TDD | Route policy contract, success/error shape, route-level integration tests. | Multiple | UT-006, IT-001 through IT-004, REG-004 |
| `appview/internal/middleware/body_limit_test.go` | Create tests during TDD | Unit/integration body policy coverage. | FR-004, FR-005, FR-012, FR-013 | UT-008, UT-009, UT-010, IT-006 |
| `appview/internal/middleware/rate_limit_test.go` | Create tests during TDD | Limiter keying, Retry-After, no IP keys, upload attempts, fake clock. | BR-002, FR-007, FR-008, FR-009, RULE-006, RULE-007 | UT-007, IT-005 |
| `appview/internal/middleware/cors_test.go` | Change | Production exact origins, dev wildcard, headers, no credentials, preflight before limiter. | BR-003, FR-010, FR-014 | UT-005, IT-007, IT-008 |
| `appview/internal/app/config_test.go` | Change | Limit defaults, invalid values, prod wildcard rejection. | NFR-003, RULE-003, RULE-008 | UT-012, IT-009 |
| `appview/environments/*.env.example` or existing env docs if present | Change if present | Document new env vars and process-local limiter warning. Startup log is still the required AC-021 artifact if no env docs exist. | RULE-008, NFR-003 | IT-009, MAN-002 |

## 5. Services, Interfaces, And Data Flow

### Route policy registry

Add explicit route metadata near route registration so the policy table and handler registration cannot drift.

```text
type RateClass string
const (
  RateClassAuth RateClass = "auth"
  RateClassRead RateClass = "read"
  RateClassWrite RateClass = "write"
  RateClassSearch RateClass = "expensive_search"
  RateClassUpload RateClass = "upload"
  RateClassExempt RateClass = "exempt"
  RateClassDevOnly RateClass = "dev_only_relaxed"
)

type BodyKind string
const (
  BodyNoBody BodyKind = "no_body"
  BodyDefaultJSON BodyKind = "default_json"
  BodyUpload BodyKind = "upload"
  BodyExempt BodyKind = "exempt"
)

type RoutePolicy struct {
  Method string
  PathPattern string
  RateClass RateClass
  BodyKind BodyKind
  AuthRequired bool
  DevOnly bool
}

func registerV1(mux *http.ServeMux, policy RoutePolicy, handler http.Handler, deps *app.Deps)
```

Tests should call an inspection function such as `V1Policies(env app.Env, cfg app.Config) []RoutePolicy` or inspect the registry populated by `AddRoutes`. The builder should choose the least invasive approach, but route registration and policy inspection must share source data.

Initial classification sketch:

```text
auth:
  POST /v1/auth/login                BodyDefaultJSON or endpoint-specific auth body policy

read:
  GET /v1/whoami
  GET /v1/profiles..., followers/following/mutual-followers
  GET /v1/feed/timeline
  GET /v1/notifications
  GET /v1/posts..., replies/comments
  GET /v1/projects
  GET /v1/search/recent

expensive/search:
  GET /v1/facets/*
  GET /v1/search/hashtags...
  GET /v1/search/profiles
  GET /v1/search/posts
  GET /v1/search/projects
  GET /v1/profiles/{handleOrDid}/posts|projects|comments if implementation considers these search-backed

write:
  POST /v1/auth/logout
  PUT /v1/profiles/me
  POST/DELETE follows, likes, reposts, reports, recent search saves/deletes, posts create/delete

upload:
  POST /v1/blobs/images

dev-only relaxed/exempt:
  GET /v1/dev/media/{name}
  POST /v1/dev/moderation/ozone-events
```

The implementation may adjust borderline read/search classifications, but every `/v1/*` route must have an explicit class and tests must capture the final table.

### Body-limit middleware

```text
type BodyLimitConfig struct {
  DefaultJSONBytes int64 // 1 MiB
  UploadBytes int64      // from existing MaxImageUploadBytes unless stricter override is configured
}

func BodyPolicy(policy routes.BodyPolicy, cfg BodyLimitConfig, logger *slog.Logger) func(http.Handler) http.Handler
```

Planned behavior:

- `BodyNoBody`: allow nil/`http.NoBody` and effectively empty bodies; reject non-empty bodies with HTTP 400 or another existing client-error status chosen by implementation, error code `request_body_not_allowed`, standard envelope.
- `BodyDefaultJSON`: cap raw request body at `1 MiB`; reject `> limit` with HTTP 413, code `request_body_too_large`, message `request body exceeds the configured limit`.
- `BodyUpload`: use explicit upload override, initially tied to existing image upload max (`MaxImageUploadBytes`) unless config introduces a separate upload body limit.
- Enforcement must happen before handler JSON decode and before any debug body-copying.

Implementation hint: `http.MaxBytesReader` is appropriate when a `ResponseWriter` is available, but tests must verify rejection happens before handler reads large payloads. If using `MaxBytesReader`, ensure the middleware itself performs a bounded read/check or that handlers consistently translate `MaxBytesError` to the required envelope.

### Process-local rate limiter

Use a small in-memory limiter rather than adding dependencies. A fixed-window or token-bucket implementation is acceptable if tests can use a fake clock and `Retry-After` is deterministic enough.

```text
type RateLimitConfig struct {
  Classes map[RateClass]ClassLimit
}

type ClassLimit struct {
  Window time.Duration
  PerToken int
  PerDevice int
}

type Limiter interface {
  Allow(now time.Time, class RateClass, keys RateKeys) Decision
}

type RateKeys struct {
  TokenKey string // Prefer authenticated session ID or DID/session-derived non-secret key, never raw bearer token in logs.
  DeviceID string
}

type Decision struct {
  Allowed bool
  RetryAfter time.Duration
  KeyType string // "token" or "device" for internal logs only
}
```

Default class config from requirements:

```text
auth:    10/min per device, no token requirement
read:    300/min per token, 600/min per device
write:   60/min per token, 120/min per device
search:  60/min per token, 120/min per device
upload:  100/hour per token, 200/hour per device
```

429 behavior:

- Status: `429 Too Many Requests`
- Error code: `rate_limited`
- Standard envelope via `envelope.WriteError`
- Include `Retry-After`
- Do not emit public `X-RateLimit-*` headers or bucket names.
- Logs may include route class and key type, not raw token/device values.

### Middleware data flow

Recommended composition by route category:

```text
POST /v1/auth/login:
  bodyPolicy(default JSON)
  deviceID
  rateLimit(auth, device key only)
  login handler

Authenticated /v1 reads/writes/search/upload:
  bodyPolicy(route policy)
  authN                 // validates bearer token and stores DID/session ID
  deviceID              // validates untrusted device ID and touches session if auth context exists
  rateLimit(class, token key + device key)
  handler

Dev-only /v1/dev/*:
  bodyPolicy(explicit dev policy)
  optional dev auth/token behavior already required by handler where present
  rateLimit(dev-only relaxed or documented exempt)
  handler
```

Current `routes.go` wraps `authN(deviceID(handler))`, meaning `Authenticated` runs before `DeviceID`. Keeping that ordering is acceptable for authenticated routes because device ID must not authorize. If implementation needs rate limiting before auth for a route, it must still avoid token-based decisions before authentication and must not trust a raw bearer token as identity.

### Config and AC-021 guidance surface

Add validated config fields and env parsing in `app.Config`, using existing helper style (`durationEnv`, bounded int parsing, clear env names). Suggested fields:

```text
Config.BodyLimits.DefaultJSONBytes
Config.RateLimits.Classes[RateClass]
Config.RateLimits.ProcessLocalSingleInstanceWarning bool or implicit startup log
```

Concrete AC-021 artifact: add an operator-facing startup log in `newDeps` such as:

```text
logger.Warn("rate limiter is process-local; run one AppView instance or configure shared/edge enforcement before horizontal scaling")
```

This startup log is the required concrete artifact for `IT-009`/`MAN-002`. If env example/docs already exist and are touched during implementation, also document the same warning there, but do not make docs the only verification surface unless tests can inspect it reliably.

## 6. State, Providers, Controllers, Or DI

No Flutter/Riverpod state is involved.

AppView dependency injection changes:

```text
app.Config
  ├─ BodyLimits
  └─ RateLimits

app.Deps
  ├─ Config
  ├─ Logger
  └─ RateLimiter (new process-local limiter instance)

routes.AddRoutes
  ├─ route policy registry
  ├─ middleware.BodyPolicy(policy.BodyKind, deps.Config.BodyLimits, deps.Logger)
  ├─ middleware.Authenticated(...)
  ├─ middleware.DeviceID(...)
  └─ middleware.RateLimit(deps.RateLimiter, policy.RateClass, deps.Logger)
```

If adding a `RateLimiter` field to `app.Deps` complicates tests, the TDD builder may initialize a default limiter inside `routes.AddRoutes`, but shared process-wide class buckets are easier to reason about when limiter construction lives in `newDeps` and tests can inject a fake/low-limit limiter.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

No Flutter UI/widgets/routes are planned.

User-facing HTTP surfaces:

- `/v1/*` successful JSON responses remain endpoint-specific bare bodies.
- `/v1/*` error responses remain `{error, message, requestId}` with optional `fields`.
- New standardized failure responses:
  - `413 request_body_too_large` with message `request body exceeds the configured limit`.
  - `429 rate_limited` with `Retry-After`.
  - `request_body_not_allowed` for unexpected bodies on no-body routes.
- CORS preflight supports bearer-token `Authorization`, `Content-Type`, and `X-Craftsky-Device-Id`; it does not enable cookie credential CORS through `Access-Control-Allow-Credentials`.

Routes/navigation impact is limited to server route registration internals. Do not add `/v2/`, `/xrpc/`, Flutter client behavior, lexicons, migrations, or OAuth protocol changes.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Successful `/v1/*` JSON | Encode endpoint-defined bare body; no synthetic top-level `data`. | BR-001, RULE-001 | AT-001, UT-001, IT-001, REG-001 |
| Handler/middleware client/server error | Use `envelope.WriteError` with JSON content type and `requestId`. | FR-001, FR-002, RULE-002 | AT-002, UT-002, IT-002, REG-002 |
| Missing/invalid device ID | Existing 400 `missing_device_id` / `invalid_device_id`; handler not called. | FR-003, RULE-005 | AT-003, UT-004, IT-003 |
| Device ID without auth | Device ID remains context-only abuse signal; authenticated route returns 401 without valid token. | RULE-005 | AT-011, UT-013, IT-003 |
| Oversized default JSON body | Reject early with 413 `request_body_too_large`, standard envelope, no full body logging. | FR-004, FR-012, NFR-001, NFR-004 | AT-007, UT-008, UT-011, IT-006 |
| Body at or below default limit | Body policy passes; downstream validation/handler decides semantics. | FR-004 | UT-008, IT-006 |
| Upload body above JSON default but within upload override | Upload body policy passes; upload handler/media validation runs. | FR-005, RULE-004 | AT-008, UT-009, IT-006 |
| Upload body above override | Reject with standard size-limit envelope according to override. | FR-005, FR-012 | AT-008, UT-009, IT-006 |
| Non-empty body on no-body route | Reject before handler with `request_body_not_allowed`; absent body passes. | FR-013 | AT-009, UT-010, IT-006 |
| Token or device route-class bucket exhausted | Reject before endpoint work with 429 `rate_limited`, `Retry-After`, no public quota headers. | FR-007, FR-008, FR-009, NFR-002 | AT-005, UT-007, IT-005 |
| Auth/login route pre-auth | Apply auth-class per-device limit without requiring Craftsky token. | FR-006, FR-008, DR-001 | AT-005, UT-007, IT-005 |
| Upload failed attempts | Count attempts once they reach upload limiter, regardless of later validation failure. | RULE-007 | AT-012, UT-007, IT-005 |
| Allowed CORS preflight | CORS short-circuits before route limiter and does not consume buckets. | FR-014 | AT-010, UT-005, IT-008 |
| Production wildcard origin | Config validation fails or wildcard behavior disabled in prod; exact origins only. | RULE-003, NFR-003 | UT-012, IT-007, REG-005 |
| Process-local limiter with horizontal scaling risk | Startup/operator-facing warning states single-instance/shared-or-edge requirement. | RULE-008 | IT-009, MAN-002 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-006 / IT-004 | `appview/internal/routes/routes_test.go` or new route-policy tests | Build dev/prod route tables; inspect `TD-007` policy fixture. | No route policy registry exists; routes lack explicit body/rate classes. |
| 2 | UT-008 | `appview/internal/middleware/body_limit_test.go` | 1 MiB and 1 MiB+1 byte default JSON bodies, handler spy. | No body-limit middleware. |
| 3 | UT-009 | `body_limit_test.go` and upload route fixture | Default vs upload override sizes. | No override resolution. |
| 4 | UT-010 | `body_limit_test.go` | GET/DELETE with nil, empty, non-empty bodies. | No no-body enforcement. |
| 5 | IT-006 | `routes_test.go` / `body_limit_test.go` | Mux with default JSON, upload, and no-body routes plus logging/body-read spy. | Policy not wired through mux; logging may read full body first. |
| 6 | UT-007 | `appview/internal/middleware/rate_limit_test.go` | Low limits, fake clock, token/device keys, auth route with no token, upload attempts. | No limiter implementation. |
| 7 | IT-005 | `rate_limit_test.go` / route integration | Mux with handler counter and low class limits. | Route middleware does not reject 429 before handler. |
| 8 | UT-005 | `appview/internal/middleware/cors_test.go` | Origins `app`, marketing, preview, evil, localhost/dev wildcard; preflight headers. | Existing CORS lacks `X-Craftsky-Device-Id` header and prod validation. |
| 9 | IT-007 / IT-008 | `cors_test.go`, `rate_limit_test.go` | CORS before limiter with low route bucket. | Preflight ordering/bucket behavior not proven. |
| 10 | UT-012 | `appview/internal/app/config_test.go` | Missing env values, invalid byte/count/duration values, prod wildcard. | Config lacks limit fields and prod wildcard rejection. |
| 11 | IT-009 | `config_test.go` or startup/deps test | Capture startup log or inspect selected guidance surface. | No AC-021 process-local warning exists. |
| 12 | UT-011 | `logging_test.go`, body/rate limiter log tests | Authorization header and oversized sensitive payload. | Existing logging emits raw headers and JSON payload. |
| 13 | UT-001 / UT-003 | `appview/internal/api/*_response_test.go` or envelope helper tests | Representative simple and paginated success bodies. | If helper added incorrectly, `data` wrapper appears. |
| 14 | AT/REG sweep | Existing API/route response tests | `/v1/whoami`, paginated list, missing auth/device, validation error. | Regressions in bare success or error envelope shape. |
| 15 | MAN-001 / MAN-002 | Manual review | Review final middleware order, logs, limiter key construction, startup warning. | Any unbounded body logging, IP key, or missing single-instance warning blocks sign-off. |

Focused commands:

```text
cd appview && go test ./internal/routes ./internal/middleware ./internal/app ./internal/api
just test
just fmt
```

Per `02-acceptance-tests.md`, `just dev-d` may be needed before full integration tests that require compose Postgres.

## 10. Sequencing And Guardrails

- First TDD step: write `UT-006` / `IT-004` route policy classification contract before implementing middleware internals.
- Dependencies between work items:
  1. Route policy metadata must exist before route-wide body/rate enforcement can be wired safely.
  2. Body policy must be implemented before resolving logging body-copy behavior.
  3. Rate limiter needs auth/device context decisions settled by route class.
  4. CORS preflight ordering should be verified after route policy/limiter composition exists.
  5. Config defaults and startup warning should be finalized before full integration sweep.
- Guardrails:
  - Do not wrap successful `/v1/*` JSON in `{ "data": ... }`.
  - Do not change `/oauth/*`, `/health`, or `/healthz` contracts except outer generic CORS behavior if already applied by server middleware.
  - Do not use IP addresses as AppView rate-limit keys.
  - Do not treat `X-Craftsky-Device-Id` as authentication or authorization evidence.
  - Do not log raw bearer tokens, raw limiter token keys, or full rejected request bodies.
  - Do not allow `*`, wildcard subdomains, preview patterns, or broad origin reflection in prod CORS.
  - Do not add persistent/distributed limiter storage in this scope.
  - Count upload attempts at the upload limiter before later upload validation can fail.
  - Auth/login limiter tests must model pre-auth requests without requiring a Craftsky token, per DR-001.
  - CORS notes must distinguish bearer-token `Authorization` headers from cookie credential CORS, per DR-002.
- Out of scope:
  - Flutter UI changes.
  - Lexicon changes or ADRs.
  - Database migrations.
  - Shared Redis/Postgres rate limiter.
  - Edge/proxy IP throttling implementation.
  - Success envelope migration or `/v2/` API.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Exact classification for some search-backed read routes may be read vs expensive/search. | Rate limits could be too strict or too loose for those routes. | TDD builder should choose final class in policy fixture and keep it explicit/reviewable. |
| CPQ-002 | Non-blocking | Whether default JSON auth/login body limit needs a smaller override than 1 MiB. | Large login bodies may be allowed up to default limit. | Use default JSON unless tests/implementation discover an existing narrower login contract; keep configurable. |
| CPQ-003 | Non-blocking | Limiter algorithm choice: fixed window vs token bucket. | Affects `Retry-After` precision and burst behavior. | Use simple fake-clock-friendly process-local implementation; tests should assert behavior, not undocumented algorithm internals. |
| CPQ-004 | Non-blocking | Token limiter key should avoid raw bearer token. | Raw token in memory/logs is sensitive; hashing/session ID strategy affects tests. | Prefer authenticated session ID or DID+session key from `Authenticated`; never log raw token or token key value. |
| CPQ-005 | Non-blocking | Logging currently captures request headers and JSON payload before route body policy. | Could violate NFR-001/NFR-004 if not adjusted. | Split/adjust logging so run ID is available early but body capture is bounded/redacted and cannot read oversized rejected bodies. |
| CPQ-006 | Non-blocking | Config env variable names are not specified in requirements. | Tests need stable names. | Choose clear names during TDD, e.g. `APPVIEW_JSON_BODY_LIMIT_BYTES`, `APPVIEW_RATE_READ_TOKEN_PER_MINUTE`; errors must name env vars. |

No blocking open questions remain from document review.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-06-28-appview-architecture-hardening/04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-006` / `IT-004` route policy classification contract in `appview/internal/routes/routes_test.go` or a new route-policy contract test suite.
- Focused command: `cd appview && go test ./internal/routes ./internal/middleware ./internal/app ./internal/api`
- Full command: `just test` after `just dev-d` when compose-backed tests need Postgres.
- Notes:
  - Treat `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this coding plan as source of truth.
  - Preserve bare success bodies and standard error envelopes.
  - Use route policy metadata as the implementation spine.
  - Satisfy AC-021 with a concrete startup/operator-facing process-local limiter warning, preferably tested by capturing startup logs or inspecting config/deps behavior.
