# Implementation Review: AppView Observability

## Verdict
Status: Approved with notes
Reviewer: Codex
Date: 2026-06-30
Risk level: Medium

## Summary
The latest correction resolves the remaining Must-level privacy gap from the prior review. Read-route handler logs for post, comment/reply, author-list, profile, graph-list, timeline, notifications, search, facet, report, moderation, auth, and whoami paths now use bounded `component`, `operation`, `result`, and `error_category` fields instead of raw DID, handle, rkey, AT-URI/CID, cursor/query/input, row/response objects, or `err.Error()` strings.

The new regression `TestReadHandlerLogsUseBoundedContextWithoutRawIdentityOrContent` covers representative read success and prod error/warn paths and asserts raw identity/content/cursor/error fields are absent. Targeted scans found no remaining raw identifier/error structured log fields in the reviewed runtime packages. Focused and full verification passed.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Suggestion | Traceability / Tracing | Sentry SDK transaction/span export is implemented for observer-created spans and covered with a `sentry-go` transport test. This remains narrower than the full Should-level wording for incoming HTTP and Tap consume-loop spans, but it is non-blocking because the docs describe the actual bounded observer span scope and no OTLP/OpenTelemetry exporter is required in this slice. | `01-requirements.md` FR-007, FR-008, AC-003, AC-010, AC-018; `05-implementation-plan.md` Re-Review R9; `appview/internal/observability/sentry.go`; `appview/internal/observability/sentry_test.go`; `appview/README.md` Observability section | Keep the documented scope, or add HTTP/Tap Sentry span wiring in a later observability slice. |

## Requirement And Test Traceability
- Requirements implemented: `/metrics`, HTTP metrics/logging middleware, in-flight request tracking, Sentry capture for panics and uncaptured AppView 5xx responses, expected PDS 4xx Sentry suppression, PDS write metrics/logs/spans, real Sentry SDK transaction/span export for observer spans, DB/search timing, Tap/indexer metrics, redaction helpers, safe route-pattern telemetry for middleware/metrics, config defaults, Sentry flush, production `/metrics` documentation, PDS/write-handler log redaction, and read-route log redaction are materially covered.
- Tests implemented: Route `/metrics` tests, route-pattern logging/HTTP metric tests, in-flight gauge concurrency test, config validation tests, redaction tests, DB/search timing tests, PDS helper/wrapper tests, handler-level PDS operation coverage, bounded write-handler log test, bounded read-handler log test, dispatcher raw-log omission test, HTTP recovery/Sentry panic test, non-panic HTTP 5xx Sentry test, expected-PDS-502 no-Sentry test, OAuth expiry log redaction test, Sentry transaction/span export test, Tap metric/Sentry error tests, and broader package regression tests.
- Unplanned behavior: No new product/API behavior found. `/metrics` remains an unauthenticated ops endpoint outside `/v1`, as approved.
- Remaining gaps: No blocking Must-level gaps identified. The remaining tracing note is Should-level and documented.

## Test Evidence
- Commands reviewed: current `git status`; current diffs and untracked files in observability, middleware, app, Tap, route, API, auth, index, docs, and config examples; updated `05-implementation-plan.md`.
- Passing evidence: `go test ./internal/api ./internal/app ./internal/middleware ./internal/observability ./internal/auth ./internal/index -count=1` passed. `go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1` passed. `just fmt` passed. `just test` passed with `go test -race ./...`. `git diff --check` passed.
- Failing or skipped tests: None in this re-review.

## Risk Review
- Risk level: Medium.
- Risk notes: The change is cross-cutting and privacy-sensitive, but the prior raw read-route log issue is now covered by tests and broad log-field scans. Optional Sentry tracing is intentionally scoped to observer spans.
- Approval notes: No lexicon, Flutter, API contract, or database migration impact was found. Existing response status/body behavior appears preserved by the focused and full test suites.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: This change has no user-facing UI surface.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes: None.
- Suggested next failing test: None required for this slice.
- Verification to rerun: Before merge, rerun `just fmt`, `just test`, and `git diff --check` after any additional edits.
