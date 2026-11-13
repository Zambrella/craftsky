---
{
  "id": "2026-07-03-flutter-error-handling-sentry",
  "title": "Original Flutter error mapping and Sentry reporter",
  "source": "docs/changes/2026-07-03-flutter-error-handling-sentry",
  "implementation": "superseded",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "observability",
    "errors",
    "sentry"
  ],
  "keywords": [
    "ErrorReporter",
    "LogRecord bridge",
    "Riverpod observer",
    "AppError",
    "localized errors",
    "breadcrumbs"
  ],
  "summary": "The original Flutter slice introduces centralized error mapping, localized recovery messages and an app-owned Sentry reporter boundary.",
  "currentDocs": [
    "docs/development/logging-error-reporting.md",
    "app/README.md"
  ],
  "code": [
    "app/lib/shared/observability",
    "app/lib/shared/api"
  ],
  "evidence": [
    "docs/changes/2026-07-03-flutter-error-handling-sentry/05-implementation-plan.md#final-verification",
    "docs/changes/2026-07-03-flutter-error-handling-sentry/06-implementation-review.md"
  ],
  "supersededBy": [
    "2026-10-02-logging-error-reporting"
  ],
  "followUps": []
}
---

# Original Flutter error mapping and Sentry reporter

## Outcome and decisions

The original Flutter slice introduces centralized error mapping, localized recovery messages and an app-owned Sentry reporter boundary. It records framework/platform/root-zone handling, provider failures, severe logging, safe HTTP diagnostics and protected breadcrumbs. The later logging redesign changes ownership rules: supporting logs do not implicitly own issues, and native SDK chains/automatic integrations remain authoritative under the current guide.

## Evidence and limits

The historical re-review is Approved with notes; 766 Flutter tests and analysis passed after corrections. Real Sentry ingestion and native-crash behavior remained staging/device checks. Do not revive retired severe-log capture behavior from this package.

Historical evidence: [05-implementation-plan.md — Final Verification](../../../docs/changes/2026-07-03-flutter-error-handling-sentry/05-implementation-plan.md#final-verification); [06-implementation-review.md](../../../docs/changes/2026-07-03-flutter-error-handling-sentry/06-implementation-review.md).

## Current entry points

[docs/development/logging-error-reporting.md](../../../docs/development/logging-error-reporting.md); [app/README.md](../../../app/README.md). This card preserves the recorded change outcome; it does not replace current code or maintained guidance.
