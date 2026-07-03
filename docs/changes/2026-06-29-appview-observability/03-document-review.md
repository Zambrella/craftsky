# Document Review: AppView Observability

## Verdict
Status: Approved with notes
Reviewer: Codex
Date: 2026-06-29
Risk level: Medium

## Summary
The requirements and acceptance-test specification are consistent with the selected Option A direction: AppView-only structured logs, Prometheus `/metrics`, Sentry error capture, optional Sentry tracing, safe configuration, and coverage for HTTP, Tap/indexer, DB, and PDS/OAuth write paths. Must-level requirements have linked acceptance criteria and test coverage. The remaining issues are implementation-planning notes, not blockers.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Traceability | Search DB timing is correctly covered, but the coding plan still needs to choose the exact first search route/store operation used for the red-green loop so `FR-020` does not expand into broad DB instrumentation work. | `01-requirements.md` FR-020, AC-019; `02-acceptance-tests.md` AT-012, UT-012, IT-008 | In `04-coding-plan.md`, name the initial search route and bounded DB operation, then treat additional request-backed DB operations as follow-on implementation steps. |
| DR-002 | Suggestion | Tests | Background panic capture is acknowledged as a test gap, and implementation may need a new worker boundary to test it cleanly. | `01-requirements.md` FR-011, AC-004; `02-acceptance-tests.md` GAP-001, AT-007, IT-003 | In `04-coding-plan.md`, identify the Tap/indexer wrapper or worker boundary where panic recovery/capture will live before implementation starts. |
| DR-003 | Suggestion | Risk | Production `/metrics` access restriction is necessarily outside AppView code and is covered by manual documentation review, but the target documentation/config file is not yet identified. | `01-requirements.md` FR-015, RULE-008, AC-011, RISK-005; `02-acceptance-tests.md` MAN-004, GAP-003 | In `04-coding-plan.md`, specify where the production restriction note will be added and keep code-level `/metrics` unauthenticated as required. |

## Traceability Review
- Planning to requirements: The requirements preserve the confirmed direction from the discovery section: include Sentry, structured JSON logs, and Prometheus `/metrics`; defer OpenTelemetry/OTLP export and Sentry Application Metrics; keep `/metrics` outside `/v1/`; protect tokens, bodies, raw identity, and high-cardinality fields.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR`, and `RULE` has at least one linked acceptance criterion. Should-level requirements also have criteria where implementation behavior needs verification.
- Acceptance criteria to tests: The coverage matrix maps each acceptance criterion to acceptance, unit, integration, regression, or manual checks. Manual-only coverage is justified for hosted Sentry smoke checks and production network/proxy restrictions.

## Coverage Review
- Must requirements covered: `BR-001`, `BR-002`, Must `FR-*`, Must `NFR-*`, and Must `RULE-*` requirements are covered by linked ACs and tests or explicit manual/gap treatment.
- Missing or weak coverage: No blocking missing Must coverage identified. The weakest areas are background panic recovery, real hosted Sentry behavior, production `/metrics` restriction, and performance overhead; all are explicitly listed as test gaps or manual checks.
- Manual-only coverage: `MAN-001` covers real Sentry smoke behavior, `MAN-003` covers dev Docker `/metrics` reachability, `MAN-004` covers production `/metrics` restriction documentation, and `MAN-005` covers metric name/unit documentation. These are acceptable for this stage.

## Risk And Approval Review
- Risk level: Medium.
- Review requirement: Coding plan should preserve the security/privacy allowlist, avoid raw path/query/user identifiers, keep hosted telemetry disabled unless explicitly configured, and avoid synchronous vendor calls on request hot paths.
- Approval notes: No architecture conflict found. `/metrics` outside `/v1/` aligns with the API architecture spec's ops endpoint model. No lexicon, schema, Flutter, or API response contract change is requested.

## Coding Plan Readiness
- Ready for coding planning: Yes.
- Recommended first step: Start from `AT-002` / `IT-001`: prove unauthenticated `GET /metrics` returns Prometheus text outside `/v1/*` while existing `/v1/*` auth/device behavior remains intact.
- Blocking issues: None.

## Notes For Next Stage
- Keep the first implementation slice small enough to land safely: metrics route/registry wiring first, then safe config/redaction, then request logging/correlation, then DB/PDS/Tap instrumentation, then Sentry lifecycle and tracing.
- Define the telemetry helper boundary before touching many handlers so labels, redaction, and Sentry attributes stay consistent.
- Make the allowlist explicit in code and tests; any field outside the allowlist should require a deliberate coding-plan callout.
- Keep production body logging forced off even if the unsafe local flag is set.
