# Implementation Review: AppView Sentry Observability Consolidation

## Verdict
Status: Approved with notes
Reviewer: Codex
Date: 2026-07-02
Risk level: Medium

## Summary
The implementation follows the approved Sentry-consolidation direction: AppView-owned Prometheus collectors and `/metrics` are removed, Sentry logs/tracing/metrics are explicitly gated, errors and panics use bounded/redacted Sentry event data, metric calls now go through AppView-domain methods, and focused regression tests pass. No blocking defects were identified.

One non-blocking traceability note remains around Prometheus dependency wording and test coverage: runtime imports and handlers are guarded, but `go.mod` still contains Prometheus modules indirectly through `github.com/bluesky-social/indigo`. The implementation plan documents this, but the acceptance test text expected dependency metadata coverage more literally.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Suggestion | Traceability / Tests | Prometheus runtime code is removed, but the `UT-011` guard scans only Go files and does not check `go.mod`, `go.sum`, or README snippets even though the acceptance test lists dependency metadata and docs as inputs. `go.mod` still includes Prometheus modules as indirect dependencies via `indigo`, which is likely unavoidable without changing the atproto dependency, but the final guard should make that distinction explicit. | FR-017, AC-013, UT-011; `appview/go.mod`; `appview/internal/observability/prometheus_removal_test.go`; `go mod why -m github.com/prometheus/client_golang` | Non-blocking: either tighten the guard to fail only on direct/AppView-owned Prometheus dependencies and production runtime references, or update the workflow docs to clarify that transitive `indigo` Prometheus modules are allowed while AppView-owned Prometheus code is removed. |

## Requirement And Test Traceability
- Requirements implemented: BR-001 through BR-003; FR-001 through FR-018; NFR/RULE privacy, gating, tracing, metrics, logging, and `/metrics` removal requirements are represented in code or tests.
- Tests implemented: Config gating, import boundary, in-memory/Sentry metrics, sanitizer/validator, error classifier, panic redaction, tracing normalization, HTTP/PDS/DB/Tap observability, `/metrics` removal, Prometheus runtime removal, and focused regression coverage.
- Unplanned behavior: None identified.
- Remaining gaps: Hosted Sentry Application Metrics/log indexing behavior remains a planned manual/non-production verification gap. Prometheus transitive dependency semantics should be clarified as noted in IR-001.

## Test Evidence
- Commands reviewed: `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`; `cd appview && go test ./internal/observability -run TestPrometheusRemoval -count=1`; `cd appview && git diff --check`; `cd appview && go mod why -m github.com/prometheus/client_golang`.
- Passing evidence: Focused AppView suite passed across cmd/appview, app, middleware, routes, api, auth, tap, index, and observability packages. `TestPrometheusRemoval` passed. `git diff --check` passed.
- Failing or skipped tests: `just test` was not rerun during this review because the implementation notes state it requires the compose Postgres on `localhost:5433`; the focused suite from the workflow docs was rerun and passed.

## Risk Review
- Risk level: Medium
- Risk notes: The change remains cross-cutting across startup config, HTTP middleware, PDS wrappers, Tap consumer, metrics/log/error/tracing internals, docs, and dependency metadata. Privacy risks are mitigated by sanitizer/classifier tests and the Sentry import-boundary guard.
- Approval notes: Ready to hand off. The only follow-up is documentation/test precision around transitive Prometheus modules, not a runtime observability blocker.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: No user-facing Flutter/UI changes were made.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes: None.
- Suggested next failing test: None required. Optional follow-up: add or update a guard proving AppView has no direct/runtime Prometheus dependency while allowing the current `indigo` transitive dependency.
- Verification to rerun: `cd appview && go test ./cmd/appview ./internal/app ./internal/middleware ./internal/routes ./internal/api ./internal/auth ./internal/tap ./internal/index ./internal/observability -count=1`; `cd appview && go test ./internal/observability -run TestPrometheusRemoval -count=1`; `cd appview && git diff --check`.
