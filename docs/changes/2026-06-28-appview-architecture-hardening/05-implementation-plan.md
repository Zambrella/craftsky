# TDD Implementation Plan: AppView Architecture Hardening

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
- Preserve bare `/v1/*` success response bodies and standard error envelopes.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-006 / IT-004 | BR-002, FR-006, RULE-004 | AC-006 | Fails: no explicit route policy registry exists. |
| 2 | UT-008 | FR-004, FR-012, NFR-001 | AC-011, AC-012, AC-015 | Fails: no default body-limit middleware exists. |
| 3 | UT-009 | FR-005, RULE-004 | AC-013 | Fails: no body-limit override resolution exists. |
| 4 | UT-010 | FR-013 | AC-016 | Fails: no no-body policy enforcement exists. |
| 5 | IT-006 | FR-004, FR-005, FR-012, FR-013, NFR-001 | AC-011, AC-012, AC-013, AC-016 | Fails: body policy is not wired through mux/logging order. |
| 6 | UT-007 | BR-002, FR-007, FR-008, FR-009, RULE-006, RULE-007 | AC-007, AC-008, AC-019, AC-020 | Fails: no route-class limiter exists. |
| 7 | IT-005 | BR-002, FR-007, FR-008, FR-009, RULE-006, RULE-007 | AC-007, AC-008, AC-019, AC-020 | Fails: routes do not reject with 429 before handler work. |
| 8 | UT-005 | BR-003, FR-010, FR-011, FR-014, RULE-003 | AC-009, AC-010, AC-017 | Fails: CORS headers/config/order are incomplete. |
| 9 | IT-007 / IT-008 | BR-003, FR-010, FR-014, RULE-003 | AC-009, AC-010, AC-017 | Fails: production/dev CORS and preflight limiter ordering not proven. |
| 10 | UT-012 | NFR-003, RULE-003 | AC-014 | Fails: config lacks new limit defaults/validation. |
| 11 | IT-009 | NFR-003, RULE-008 | AC-014, AC-021 | Fails: process-local limiter warning artifact absent. |
| 12 | UT-011 | NFR-001, NFR-004 | AC-015 | Fails: logging may expose raw headers/body and reads before body policy. |
| 13 | UT-001 / UT-003 | BR-001, FR-002, RULE-001 | AC-001, AC-002, AC-003, AC-004 | Fails only if helper/response contract regresses. |
| 14 | AT/REG sweep | BR-001, FR-001, RULE-001, RULE-002 | AC-001, AC-002, AC-003, AC-004 | Existing regressions should remain green after changes. |
| 15 | MAN-001 / MAN-002 | NFR-001, NFR-002, NFR-004, RULE-006, RULE-008 | AC-015, AC-019, AC-021 | Manual review records final middleware/logging/limiter guidance. |

## Implementation Steps

### Step 1: UT-006 / IT-004
- Write failing test: Added `TestV1RoutePoliciesCoverRegisteredRoutes` in `appview/internal/routes/routes_test.go` to inspect dev/prod `/v1/*` policy metadata, validate explicit rate/body classes, assert representative auth/read/search/write/upload/dev policies, and ensure dev-only routes are absent from prod policy output.
- Run command: `cd appview && go test ./internal/routes`
- Confirmed failure: Build failed with undefined `V1RoutePolicies`, `RoutePolicy`, `RateClass`, `BodyKind`, and constants, proving no route policy registry existed.
- Implement: Added `appview/internal/routes/policy.go` with `RateClass`, `BodyKind`, `RoutePolicy`, validity helpers, and `V1RoutePolicies` backed by an explicit table for all currently registered `/v1/*` routes plus dev-only conditionals.
- Run command: `cd appview && go test ./internal/routes`
- Refactor: None beyond keeping the policy registry in a standalone file for later route wiring.
- Notes: Covers the first guardrail for BR-002 / FR-006 / RULE-004 and AC-006. The policy table is explicit but not yet used to compose middleware; later IT steps will wire enforcement through registration.

### Step 2: UT-008
- Write failing test: Added `TestBodyLimitDefaultJSONRejectsOversizedBeforeHandler` and `TestBodyLimitDefaultJSONAllowsAtLimit` in `appview/internal/middleware/body_limit_test.go`.
- Run command: `cd appview && go test ./internal/middleware -run TestBodyLimitDefaultJSON`
- Confirmed failure: Build failed with undefined `BodyLimit`, `BodyLimitConfig`, and `BodyDefaultJSON`, proving no default JSON body-limit middleware existed.
- Implement: Added `appview/internal/middleware/body_limit.go` with `BodyLimitConfig`, body-kind constants, and `BodyLimit` middleware that reads up to `limit+1`, restores the body when within limit, rejects over-limit bodies before the handler with HTTP 413 `request_body_too_large`, and logs bounded rejection metadata only.
- Run command: `cd appview && go test ./internal/middleware -run TestBodyLimitDefaultJSON`
- Refactor: None.
- Notes: Covers FR-004 / FR-012 / NFR-001 for the focused default JSON limit unit behavior. Later steps will add overrides, no-body behavior, and route/logging integration.

### Step 3: UT-009
- Write failing test: Added `TestBodyLimitUploadUsesUploadOverride` covering upload bodies over the default JSON limit but within upload override, and bodies over the upload override.
- Run command: `cd appview && go test ./internal/middleware -run TestBodyLimitUploadUsesUploadOverride`
- Confirmed failure: Oversized upload override case called the handler because `BodyUpload` was not enforced.
- Implement: Extended `BodyLimit` to enforce `cfg.UploadBytes` for `BodyUpload`.
- Run command: `cd appview && go test ./internal/middleware -run TestBodyLimitUploadUsesUploadOverride`
- Refactor: None.
- Notes: Covers FR-005 / RULE-004 and AC-013 at the focused middleware level.

### Step 4: UT-010
- Write failing test: Added `TestBodyLimitNoBodyRejectsNonEmptyBodies` covering absent, whitespace-only, and non-empty bodies on a no-body route.
- Run command: `cd appview && go test ./internal/middleware -run TestBodyLimitNoBodyRejectsNonEmptyBodies`
- Confirmed failure: Non-empty no-body request still called the handler.
- Implement: Added `BodyNoBody` enforcement that permits absent/empty bodies and rejects non-empty bodies with HTTP 400 `request_body_not_allowed` standard envelope before handler work.
- Run command: `cd appview && go test ./internal/middleware -run TestBodyLimitNoBodyRejectsNonEmptyBodies`
- Refactor: None.
- Notes: Covers FR-013 and AC-016 at focused middleware level.

### Step 5: IT-006
- Write failing test: Added `TestAddRoutes_BodyPolicyRunsThroughMux` covering a default JSON route (`POST /v1/posts`) rejected at a configured low limit and a no-body route (`GET /v1/whoami`) rejecting an unexpected body through the real mux before auth/handler behavior.
- Run command: `cd appview && go test ./internal/routes -run TestAddRoutes_BodyPolicyRunsThroughMux`
- Confirmed failure: Build failed because `app.Config` did not expose `JSONBodyLimitBytes`, and body policy was not wired through route registration.
- Implement: Added `JSONBodyLimitBytes` to `app.Config`; wired `middleware.BodyLimit` into representative real routes for `POST /v1/auth/login` (default JSON), `GET /v1/whoami` (no-body), `POST /v1/blobs/images` (upload override), and `POST /v1/posts` (default JSON), with a 1 MiB fallback default when config is unset.
- Run command: `cd appview && go test ./internal/routes -run TestAddRoutes_BodyPolicyRunsThroughMux`; nearby command `cd appview && go test ./internal/routes ./internal/middleware`.
- Refactor: None; full policy-driven registration remains for later expansion.
- Notes: Covers IT-006 for representative default JSON/no-body mux behavior and starts route-level body policy wiring. Coverage gap remains: not every route uses body policy middleware yet; the route policy table from Step 1 keeps that visible for later completion.

### Step 6: UT-007
- Write failing test: Added `appview/internal/middleware/rate_limit_test.go` covering per-device bucket rejection, per-token bucket rejection without IP-derived keys, and middleware 429 behavior with `Retry-After`, standard envelope, no public `X-RateLimit-*` headers, and handler short-circuiting.
- Run command: `cd appview && go test ./internal/middleware -run 'TestRateLimiter|TestRateLimitMiddleware'`
- Confirmed failure: Build failed with undefined rate limiter types/functions (`NewLocalRateLimiter`, `RateLimitConfig`, `RateClass`, `ClassLimit`, `RateKeys`). After initial implementation, retry-after used real `time.Until` instead of the fake clock and produced a negative duration.
- Implement: Added `appview/internal/middleware/rate_limit.go` with process-local fixed-window limiter, token/device keys only, fake-clock-friendly `Allow`, `DebugKeys`, and `RateLimit` middleware that emits HTTP 429 `rate_limited` with `Retry-After` and no quota headers.
- Run command: `cd appview && go test ./internal/middleware -run 'TestRateLimiter|TestRateLimitMiddleware'`
- Refactor: Changed retry-after calculation to use the limiter's fake-clock `now` value.
- Notes: Covers UT-007 for BR-002 / FR-007 / FR-008 / FR-009 / RULE-006 / RULE-007. Upload attempt counting is structurally covered by limiter-before-handler behavior and will be integrated with upload route in IT-005.

### Step 7: IT-005
- Write failing test: Added `TestAddRoutes_RateLimitRejectsBeforeHandlerWork` in `appview/internal/routes/routes_test.go`, using a low read-class per-device limit through the real mux on `GET /v1/whoami`. The test proves the first authenticated/device request succeeds, the second request returns HTTP 429 with `rate_limited` and `Retry-After`, no public `X-RateLimit-*` headers are exposed, and handler output is not present in the throttled response.
- Run command: `cd appview && go test ./internal/routes -run TestAddRoutes_RateLimitRejectsBeforeHandlerWork`
- Confirmed failure: Build failed with `deps.RateLimiter undefined`, proving the process-local limiter existed only as middleware and was not integrated into app deps/routes.
- Implement: Added `RateLimiter *middleware.LocalRateLimiter` to `app.Deps`; wired `middleware.RateLimit` for the representative read route `GET /v1/whoami` after auth/device middleware so both authenticated session context and device ID are available before limiter evaluation. Routes without an injected limiter keep prior behavior for now.
- Run command: `cd appview && go test ./internal/routes -run TestAddRoutes_RateLimitRejectsBeforeHandlerWork`; nearby command `cd appview && go test ./internal/routes ./internal/middleware`.
- Refactor: None beyond keeping the route-level limiter optional so existing tests and partial wiring continue to pass until config/deps defaults are added.
- Notes: Covers IT-005 for a real read route and verifies 429 behavior before handler work for BR-002 / FR-007 / FR-008 / FR-009 / RULE-006. Coverage gap remains: limiter is not yet wired to every route class, upload attempt counting is not yet proven through the upload route, and `newDeps` does not yet construct the default process-local limiter/config.

### Step 8: UT-005
- Write failing test: Added `TestCORS_PreflightAllowsCraftskyHeadersWithoutCredentials` in `appview/internal/middleware/cors_test.go`, covering allowed app origin preflight support for `Authorization`, `Content-Type`, and `X-Craftsky-Device-Id`, with no `Access-Control-Allow-Credentials` header and no downstream handler call.
- Run command: `cd appview && go test ./internal/middleware -run TestCORS`
- Confirmed failure: `Access-Control-Allow-Headers` lacked `X-Craftsky-Device-Id`.
- Implement: Updated CORS middleware to include `X-Craftsky-Device-Id` in allowed headers and clarified comments that v1 uses bearer `Authorization` headers without cookie credential CORS.
- Run command: `cd appview && go test ./internal/middleware -run TestCORS`
- Refactor: None.
- Notes: Covers UT-005 for FR-010 / FR-011 / FR-014 / RULE-003 header behavior and credential posture. Production wildcard origin validation is covered in Step 10.

### Step 9: IT-007 / IT-008
- Write failing test: Existing `TestCORS_PreflightShortCircuits` already covers preflight short-circuiting before downstream middleware/handlers. The new CORS preflight header test extends that coverage for the launch web-client headers.
- Run command: `cd appview && go test ./internal/middleware -run TestCORS`
- Confirmed failure: Header support gap from Step 8.
- Implement: Same CORS middleware update from Step 8.
- Run command: `cd appview && go test ./internal/middleware -run TestCORS`
- Refactor: None.
- Notes: Covers preflight short-circuit behavior at middleware level. Full server-order verification remains for final sweep/manual review because server-level CORS composition is outside `routes.AddRoutes`.

### Step 10: UT-012
- Write failing test: Added config tests for prod wildcard CORS rejection, default JSON body limit, JSON body limit override, invalid JSON body limit, and documented default read/upload rate limits.
- Run command: `cd appview && go test ./internal/app -run 'TestLoadConfig_(ProdRejectsWildcardOrigin|LimitDefaults|JSONBodyLimit)'`
- Confirmed failure: Prod wildcard was accepted, `JSONBodyLimitBytes` defaulted to `0`, overrides were ignored, and invalid JSON limit did not fail.
- Implement: Added `APPVIEW_JSON_BODY_LIMIT_BYTES` parsing with 1 MiB default and bounded validation, rejected `ALLOWED_ORIGINS=*` in prod, and added `DefaultRateLimitConfig` with auth/read/write/search/upload defaults from requirements.
- Run command: `cd appview && go test ./internal/app -run 'TestLoadConfig_(ProdRejectsWildcardOrigin|LimitDefaults|JSONBodyLimit)'`; nearby command `cd appview && go test ./internal/routes ./internal/middleware ./internal/app`.
- Refactor: None.
- Notes: Covers UT-012 for JSON body config, default rate values, and prod wildcard rejection. Per-class rate env overrides remain incomplete.

### Step 11: IT-009
- Write failing test: Covered the default limiter config surface through `TestLoadConfig_LimitDefaults`; did not add a DB-backed startup-log test because `newDeps` requires a live Postgres pool.
- Run command: `cd appview && go test ./internal/app -run TestLoadConfig_LimitDefaults`
- Confirmed failure: Rate defaults were not present on config.
- Implement: Added `RateLimits` to `app.Config`, construct `deps.RateLimiter` with the process-local limiter in `newDeps`, and added the operator-facing startup warning: `rate limiter is process-local; run one AppView instance or configure shared/edge enforcement before horizontal scaling`.
- Run command: `cd appview && go test ./internal/routes ./internal/middleware ./internal/app`.
- Refactor: None.
- Notes: Satisfies the concrete AC-021 artifact in code via startup warning and default config construction. Automated assertion of the log line remains a gap until deps construction can be tested without a real DB or via compose-backed integration.

### Step 12: UT-011
- Write failing test: Added `TestLogging_RedactsAuthorizationAndDoesNotLogRequestBody` in `appview/internal/middleware/logging_test.go`, proving debug logs do not include raw bearer tokens or request payload contents.
- Run command: `cd appview && go test ./internal/middleware -run TestLogging_RedactsAuthorizationAndDoesNotLogRequestBody`
- Confirmed failure: Logs included raw `Authorization` and full `json_payload` from the request body.
- Implement: Removed request-body reads from logging middleware and added header redaction for `Authorization` in request detail logs.
- Run command: `cd appview && go test ./internal/middleware -run 'TestLogging|TestGetRunID'`.
- Refactor: None.
- Notes: Covers UT-011 for NFR-001 / NFR-004 by avoiding pre-policy request-body copying and redacting bearer tokens. Response payload logging remains unchanged and should be reviewed in MAN-001 if sensitive response bodies become a concern.

### Step 13: UT-001 / UT-003
- Write failing test: Added explicit no-wrapper assertions to `TestAddRoutes_V1WhoAmIAuthenticatedReturnsDIDAndHandle` and `TestSearchPostPageResponseOmitsPopularityScore`, covering a simple successful `/v1/whoami` route response and a representative paginated search response.
- Run command: `cd appview && go test ./internal/routes -run TestAddRoutes_V1WhoAmIAuthenticatedReturnsDIDAndHandle && go test ./internal/api -run TestSearchPostPageResponseOmitsPopularityScore`
- Confirmed failure: No failure was expected because no success helper or success-envelope migration was introduced; the tests serve as regression guardrails for BR-001 / RULE-001.
- Implement: No production implementation needed. The existing handlers and response structs already return bare endpoint-defined shapes; assertions now fail if a synthetic top-level `data` wrapper appears or if paginated `items`/`cursor` move under a wrapper.
- Run command: `cd appview && go test ./internal/routes -run TestAddRoutes_V1WhoAmIAuthenticatedReturnsDIDAndHandle && go test ./internal/api -run TestSearchPostPageResponseOmitsPopularityScore`.
- Refactor: None.
- Notes: Covers UT-001 / UT-003 for BR-001 / FR-002 / RULE-001. No `WriteJSON` success helper was added, so UT-003 is satisfied by preserving current direct bare encoding behavior rather than testing a new helper.

### Step 14: AT/REG sweep
- Write failing test: No new failing test; this step is a regression sweep over the accumulated acceptance/unit/integration tests after Steps 1-13.
- Run command: `cd appview && go test ./...`
- Confirmed failure: None; full AppView package test sweep passed.
- Implement: No production implementation needed.
- Run command: `cd appview && go test ./...`
- Refactor: None.
- Notes: Confirms representative success response shape, error envelope behavior, body-limit middleware, rate-limit middleware, CORS, config, logging redaction, and route integration remain green. Remaining work is Step 15 manual review/sign-off and any future expansion to enforce body/rate policies on every `/v1/*` route rather than the representative routes wired in this slice.

### Step 15: MAN-001 / MAN-002
- Manual review: Implementation review `06-implementation-review.md` found critical gaps IR-001 through IR-003 because body and rate policies were only enforced on representative routes. Returned to TDD with `TestAddRoutes_AllV1PoliciesEnforcedThroughMux` covering all non-dev `V1RoutePolicies` for no-body enforcement and route-class rate-limit enforcement.
- Red failure: `cd appview && go test ./internal/routes -run TestAddRoutes_AllV1PoliciesEnforcedThroughMux` initially failed/panicked when non-wrapped routes reached handlers with unexpected bodies or after one request instead of being stopped by body/rate middleware.
- Implement: Added a policy-aware `v1Middleware.wrap` helper and rewired `/v1/*` route registration to compose body policy, auth/device, and route-class rate limiting from the same `RoutePolicy` metadata used by `V1RoutePolicies`. This closes review findings IR-001, IR-002, and IR-003 for body/rate middleware drift.
- Green focused command: `cd appview && go test ./internal/routes -run TestAddRoutes_AllV1PoliciesEnforcedThroughMux` passed.
- Nearby verification: `cd appview && go test ./internal/routes ./internal/middleware ./internal/app ./internal/api` passed.
- MAN-001 notes: Body-limit middleware is now the outer route-level wrapper before auth/device/rate and handlers, so no-body and size rejections happen before endpoint parsing. Logging middleware no longer copies request bodies and redacts `Authorization`; rate-limit logs include class/key type but not raw token values.
- MAN-002 notes: Rate limiter keys are token/session and device only; no AppView limiter key uses client IP. `newDeps` constructs a process-local limiter and logs the operator warning that AppView must run as one instance or use shared/edge enforcement before horizontal scaling.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped
