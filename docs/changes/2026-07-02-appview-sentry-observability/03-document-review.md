# Document Review: AppView Sentry Observability Consolidation

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-07-02
Risk level: Medium

## Summary

The requirements and acceptance-test specification are consistent and ready for coding-plan work. The documents preserve the confirmed direction: consolidate AppView observability into Sentry, remove Prometheus `/metrics`, keep local stdout logs, add bounded business/work spans, and protect privacy through local interfaces, allowlists, and sentinel classification.

No blocking issues were found. The notes below are implementation-planning cautions, not required document rewrites.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Risk | Sentry Application Metrics remains the largest provider risk because local tests can prove call shape and privacy, but not hosted Sentry ingestion/search behavior. | `01-requirements.md` RISK-001, ASM-001; `02-acceptance-tests.md` GAP-001, MAN-003 | In the coding plan, keep the metrics interface small and make hosted Sentry metric verification a post-implementation/manual validation step, not a blocker for local TDD. |
| DR-002 | Suggestion | Requirements | The approved `sentry-go` import boundary intentionally allows "narrowly approved startup/log-handler wiring where unavoidable"; the exact files are not enumerated yet. | `01-requirements.md` FR-006, Q13; `02-acceptance-tests.md` UT-002 | In the coding plan, name the approved startup/log-handler files so the import-boundary test has an explicit allowlist. |
| DR-003 | Suggestion | Tests | The acceptance tests correctly require validation-focused failures for unsafe telemetry attributes, but the implementation will need to distinguish runtime normalization from strict test-mode validation. | `01-requirements.md` NFR-001, Q19; `02-acceptance-tests.md` UT-003, UT-007, UT-008 | In the coding plan, define the validation helper or in-memory observer behavior before adding instrumentation call sites. |

## Traceability Review

- Planning to requirements: The recommended Option A from `01-requirements.md` is carried into goals, requirements, risks, and non-goals. Decisions from the clarifying questions are preserved, including Sentry as the primary backend, removal of `/metrics`, metrics behind AppView-domain interfaces, no `sentrysql`, DSN-only errors/panics, and bounded child spans.
- Requirements to acceptance criteria: Every Must requirement has at least one acceptance criterion. The criteria are externally verifiable through config parsing, Sentry test transports/fakes, route tests, import guards, privacy assertions, and existing regression suites.
- Acceptance criteria to tests: All 16 acceptance criteria appear in `02-acceptance-tests.md`, with test IDs and automation targets. Manual checks are limited to documentation and hosted Sentry behavior that is impractical to fully prove locally.

## Coverage Review

- Must requirements covered: Yes. A mechanical check found 34 requirement IDs in `01-requirements.md`, no Must requirements missing acceptance-criteria links, and no Must requirements missing test references in `02-acceptance-tests.md`.
- Missing or weak coverage: None blocking. Hosted Sentry metric/log product behavior is intentionally recorded as a gap because local tests should not depend on external Sentry state.
- Manual-only coverage: No Must behavior is manual-only. Manual checks cover docs, breadcrumb/log distinction, and optional non-production hosted Sentry metric-name verification.

## Risk And Approval Review

- Risk level: Medium, unchanged from requirements.
- Review requirement: Review was recommended because the change touches cross-cutting observability, privacy, provider architecture, feature flags, and removal of `/metrics`.
- Approval notes: Coding planning may proceed. The planner should preserve the test-first order and avoid expanding scope into dashboards, alerts, OpenTelemetry, database migrations, client observability, or Sentry SQL instrumentation.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Start with `UT-001`, config defaults and Sentry pillar gating, then define the narrow observability interfaces and in-memory validation behavior before replacing Prometheus call sites.
- Blocking issues: None.

## Notes For Next Stage

- Keep implementation sequencing aligned with the acceptance-test order: config gates, metrics interface, sanitization/validation, error classification, panic redaction, tracing boundaries, integration paths, then `/metrics` and Prometheus dependency removal.
- Name the Sentry import allowlist early so `UT-002` can prevent direct SDK usage from spreading.
- Treat runtime normalization and test-mode validation as separate behaviors: production should degrade safely, while validation tests should fail loudly.
- Update the existing `/metrics` tests rather than deleting their intent; they should become regression coverage proving the old unauthenticated metrics surface is gone.
- Keep local stdout logging behavior explicit while filtering Sentry-bound logs with the same classifier used for Sentry events.
