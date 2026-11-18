---
{
  "id": "2026-10-02-logging-error-reporting",
  "title": "Protected logging and native Sentry diagnostics",
  "source": "docs/changes/2026-10-02-logging-error-reporting",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview",
    "web-client"
  ],
  "topics": [
    "logging",
    "errors",
    "sentry"
  ],
  "keywords": [
    "CaptureDiagnostic",
    "ReportRequestFailure",
    "ObservePrivateFailure",
    "ReportContext",
    "WithDiagnosticOccurrence",
    "Riverpod ownership",
    "requestId",
    "Sentry",
    "error reporting",
    "expired session",
    "offline access",
    "permission denied",
    "rate limits",
    "request correlation"
  ],
  "summary": "This evolving package records the protected diagnostic pipeline and later mobile, web and AppView error-handling audits through 13 November 2026.",
  "currentDocs": [
    "docs/development/logging-error-reporting.md",
    "docs/architecture.md"
  ],
  "code": [
    "appview/internal/observability",
    "app/lib/shared/observability",
    "web-client/src/lib"
  ],
  "evidence": [
    "docs/changes/2026-10-02-logging-error-reporting/06-validation-evidence.md#appview-error-handling-audit-13-november-2026",
    "docs/changes/2026-10-02-logging-error-reporting/06-validation-evidence.md#client-error-handling-audit-13-november-2026",
    "docs/changes/2026-10-02-logging-error-reporting/06-validation-evidence.md#repository-retry-escalation--2026-10-08-err-003"
  ],
  "supersededBy": [],
  "followUps": [
    "Post-release hosted collection/symbolication and separately listed device/provider checks remain unverified by local fixtures."
  ]
}
---

# Protected logging and native Sentry diagnostics

## Outcome and decisions

This evolving package records the protected diagnostic pipeline and later mobile, web and AppView error-handling audits through 13 November 2026. Preserve native typed causes/stacks, select public versus private context at the operation boundary and keep one issue owner per occurrence. Supporting logs never implicitly create issues. Worker attempts isolate sibling ownership; expected retries remain logs, with diagnostic escalation independent of durable retry behavior. Raw credentials, private payloads, recipient routing and capabilities remain excluded.

## Evidence and limits

The review file is a historical correction input. Later validation sections record actual serialized SDK/local canary checks, client/browser suites and backend race coverage, including affected-package reruns. Hosted symbolication/collection and device-specific rollout proof are explicitly separate; folder date and opening verdict cannot summarize this package.

Historical evidence: [latest AppView audit](../../../docs/changes/2026-10-02-logging-error-reporting/06-validation-evidence.md#appview-error-handling-audit-13-november-2026); [client audit](../../../docs/changes/2026-10-02-logging-error-reporting/06-validation-evidence.md#client-error-handling-audit-13-november-2026); [05-implementation-plan.md](../../../docs/changes/2026-10-02-logging-error-reporting/05-implementation-plan.md); [06-implementation-review.md](../../../docs/changes/2026-10-02-logging-error-reporting/06-implementation-review.md).

## Current entry points

[docs/development/logging-error-reporting.md](../../../docs/development/logging-error-reporting.md); [docs/architecture.md](../../../docs/architecture.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
