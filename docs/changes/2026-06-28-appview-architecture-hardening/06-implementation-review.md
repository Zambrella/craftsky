# Implementation Review: AppView Architecture Hardening

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation-reviewer
Date: 2026-06-28
Risk level: High

## Summary
The implementation adds useful foundations for route policy metadata, body-limit middleware, a process-local limiter, CORS header updates, config defaults, and logging redaction. However, it does not yet satisfy the Must requirements for launch-ready `/v1/*` hardening because most registered routes are still not enforced by the policy metadata: body policy and rate limiting are wired only to a small representative subset of routes. This leaves many `/v1/*` reads, writes, searches, and no-body routes able to bypass the new protections despite having entries in `V1RoutePolicies`.

The focused AppView test sweep passes, but the tests currently encode the partial implementation rather than the full acceptance contract. The change should return to TDD for full policy-driven route registration/enforcement and stronger route-table tests that fail when metadata and actual middleware composition drift.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Critical | Behavior / Traceability | The route policy registry is not the source of truth for actual `/v1/*` middleware composition. `V1RoutePolicies` lists every route, but `AddRoutes` applies `BodyLimit` only to `POST /v1/auth/login`, `GET /v1/whoami`, `POST /v1/blobs/images`, and `POST /v1/posts`, and applies `RateLimit` only to those plus `GET /v1/search/posts`. Most registered `/v1/*` routes therefore bypass body policy and/or rate limits while appearing classified in the metadata. | `01-requirements.md` FR-004 through FR-009, FR-013, RULE-004; `02-acceptance-tests.md` AT-004, AT-005, AT-009, IT-004, IT-005, IT-006; `04-coding-plan.md` route policy registry and middleware data flow; `appview/internal/routes/routes.go`; `appview/internal/routes/policy.go`; `05-implementation-plan.md` Steps 5, 7, 14 notes | Make route registration policy-driven or otherwise guarantee every registered `/v1/*` route applies its declared body policy and rate class. Add tests that inspect/verify actual route enforcement, not just metadata existence. |
| IR-002 | Critical | Behavior / Tests | No-body enforcement is missing for most GET/DELETE and bodyless write routes. For example routes such as `GET /v1/feed/timeline`, `GET /v1/search/posts`, `GET /v1/posts/{did}/{rkey}`, and bodyless POST/DELETE follow/like/repost/logout routes still call through auth/device middleware without `BodyLimit(..., BodyNoBody, ...)`. These routes can accept or ignore unexpected request bodies, contradicting the requirement that no-body routes reject non-empty bodies. | `01-requirements.md` FR-013, AC-016, EC-010; `02-acceptance-tests.md` AT-009, UT-010, IT-006; `appview/internal/routes/routes.go`; `appview/internal/routes/policy.go` | Apply `BodyNoBody` to every route declared no-body in the policy table and add representative route-table tests beyond `/v1/whoami` to prove all no-body policies are enforced. |
| IR-003 | Critical | Behavior / Risk | Route-class rate limiting is not launch-ready because only a few route classes/routes are actually protected in `AddRoutes`. Reads such as timeline/notifications/profile/post reads, most search routes, and many writes can still bypass the limiter. This fails the abuse-protection requirement and the shared route-class bucket model. | `01-requirements.md` BR-002, FR-006, FR-007, FR-008, FR-009, NFR-002, RULE-007; `02-acceptance-tests.md` AT-005, AT-012, IT-005; `appview/internal/routes/routes.go`; `05-implementation-plan.md` Step 7 and Step 14 gap notes | Wire `RateLimit` for every applicable route class, including read/write/search/upload/auth/dev policies. Add tests that exceed limits on at least one route per class and verify shared class buckets across multiple endpoints in the same class. |
| IR-004 | Important | Tests / Traceability | The implementation plan records known gaps but the completion checklist is left unchecked, and the automated tests do not yet cover all planned Must tests. Examples include missing full route-table enforcement tests, missing upload attempt counting through the upload route, missing automated or documented manual review for server-level CORS ordering, and no assertion of the AC-021 startup warning. | `05-implementation-plan.md` Steps 7, 9, 11, 14, 15 and checklist; `02-acceptance-tests.md` IT-005, IT-008, IT-009, MAN-001, MAN-002 | Complete Step 15 manual review notes/checklist and either automate or explicitly document accepted gaps. Blocking Must behavior gaps must be fixed rather than documented as partial implementation. |

## Requirement And Test Traceability
- Requirements implemented: Partially. Bare success response regressions, standard error envelope preservation, CORS allowed-header/no-cookie-credential posture, production wildcard rejection, default JSON body-limit middleware, upload body-limit middleware, no-body middleware, limiter primitive, config defaults, and logging redaction have focused coverage.
- Tests implemented: Focused Go tests were added/updated for route policy metadata, representative body-limit behavior, limiter primitives, representative route-level rate limiting, CORS headers, config defaults, logging redaction, and success response shape.
- Unplanned behavior: No unrelated feature work, Flutter UI changes, lexicon changes, migrations, or dependency changes were observed.
- Remaining gaps: Full `/v1/*` enforcement is missing for body policies and rate classes. Route metadata can drift from actual route registration. Several acceptance/manual checks remain incomplete or only partially evidenced.

## Test Evidence
- Commands reviewed:
  - `git status --short`
  - `git diff --stat`
  - `git diff --name-only`
  - Targeted diffs for routes, middleware, config, deps, and tests
  - `cd appview && go test ./...`
- Passing evidence:
  - `cd appview && go test ./...` passed locally during review.
- Failing or skipped tests:
  - No failing automated tests observed.
  - `just test` / race-enabled full repo command was not run during review.
  - Manual checks `MAN-001` and `MAN-002` are not completed in `05-implementation-plan.md`.

## Risk Review
- Risk level: High
- Risk notes: The highest risk is false confidence from a complete-looking policy table that is not used to enforce most routes. In production this would leave many endpoints outside the intended abuse controls and no-body protections.
- Approval notes: Do not approve for merge/handoff until actual route enforcement is complete and tests prove metadata and middleware cannot drift.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: No user-facing Flutter UI changes were made. Changes are server/API behavior and tests only.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes:
  - Make `V1RoutePolicies` drive or verify actual route registration so every `/v1/*` route has enforced body and rate policy.
  - Apply no-body/default/upload body limits to all declared routes.
  - Apply route-class rate limiting to all auth/read/write/search/upload routes, with shared buckets per class.
  - Add route-table enforcement tests that fail when a policy exists but the registered route lacks its middleware.
  - Complete/manual-review `MAN-001` and `MAN-002`, including middleware ordering, no IP keys, no sensitive logging, and process-local limiter guidance.
- Suggested next failing test: Expand `TestAddRoutes_BodyPolicyRunsThroughMux` or add a new route policy integration test that iterates representative routes from `V1RoutePolicies` and proves no-body/default/upload policies are enforced through the real mux for every body kind and class.
- Verification to rerun:
  - `cd appview && go test ./internal/routes ./internal/middleware ./internal/app ./internal/api`
  - `cd appview && go test ./...`
  - `just test` when compose Postgres is available.
