# Implementation Review: AppView Architecture Hardening

## Verdict
Status: Approved with notes
Reviewer: gpt-5.5 implementation-reviewer
Date: 2026-06-28
Risk level: Medium

## Summary
The reworked implementation addresses the prior blocking review findings. `/v1/*` route registration now uses shared `RoutePolicy` metadata through `v1Middleware.wrap`, so body policies and route-class rate limits are applied consistently across the registered non-dev v1 route surface. The implementation preserves bare success responses, standard error envelopes, validated device IDs, exact CORS allow-list behavior, no cookie credential CORS, no AppView IP-based limiter keys, and process-local limiter guidance.

Focused and full AppView Go test sweeps passed during review. The remaining notes are non-blocking: production tuning of initial limit values and broader operational validation remain launch/monitoring concerns already captured in the requirements/test gaps.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| None identified | - | - | No blocking implementation findings remain. | - | - |

## Requirement And Test Traceability
- Requirements implemented:
  - Bare `/v1/*` success response contract remains unchanged and has regression assertions.
  - Error responses continue to use the standard envelope.
  - Body policies are implemented for no-body, default JSON, and upload override cases.
  - Route policies classify the v1 route surface and are used by route registration.
  - Route-class rate limiting is process-local, keyed by authenticated session/token context and device ID where available, and avoids IP keys.
  - 429 responses use `rate_limited`, include `Retry-After`, and avoid public quota headers.
  - CORS supports the required headers/origins without `Access-Control-Allow-Credentials`.
  - Logging no longer copies full request bodies and redacts `Authorization`.
  - Process-local single-instance guidance is present as an operator-facing startup warning.
- Tests implemented:
  - Route policy metadata and all-policy mux enforcement tests.
  - Body-limit/no-body/upload middleware tests.
  - Rate limiter unit and route integration tests.
  - CORS header/preflight tests.
  - Config default/validation tests.
  - Logging redaction tests.
  - Bare success response regression tests.
- Unplanned behavior: No Flutter UI, lexicon, migration, dependency, or unrelated feature changes were observed.
- Remaining gaps: No blocking gaps. Production suitability of numeric rate defaults and multi-instance limiter behavior remain intentionally out of scope and documented as operational risks.

## Test Evidence
- Commands reviewed:
  - `git status --short`
  - `git diff --stat`
  - `git diff --name-only`
  - `git log --oneline -10`
  - `git show --stat --oneline 6422735`
  - `git diff --stat 62c0bd3..HEAD`
  - Source review of route policy/wrapping, middleware, config, and implementation plan updates.
  - `cd appview && go test ./internal/routes ./internal/middleware ./internal/app ./internal/api && go test ./...`
- Passing evidence:
  - Focused package tests passed.
  - Full AppView `go test ./...` passed.
- Failing or skipped tests:
  - No failing tests observed.
  - `just test` / race-enabled compose-backed command was not run during this review.

## Risk Review
- Risk level: Medium
- Risk notes: The main launch risks now are operational rather than implementation-blocking: rate defaults may need adjustment under real traffic, and process-local limiting must remain constrained to a single AppView instance or equivalent shared/edge enforcement.
- Approval notes: The prior critical findings are addressed by policy-aware route wrapping and all-policy enforcement tests. Ready to proceed, with operational monitoring/tuning before public launch.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: No user-facing Flutter UI changes were made. Changes are server/API behavior and tests only.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes: None.
- Suggested next failing test: None required for this workflow stage.
- Verification to rerun:
  - `cd appview && go test ./internal/routes ./internal/middleware ./internal/app ./internal/api`
  - `cd appview && go test ./...`
  - `just test` when compose Postgres is available and race/full workflow validation is desired.
