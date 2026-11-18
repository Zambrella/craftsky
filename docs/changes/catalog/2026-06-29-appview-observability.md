---
{
  "id": "2026-06-29-appview-observability",
  "title": "Original AppView observability foundation",
  "source": "docs/changes/2026-06-29-appview-observability",
  "implementation": "superseded",
  "archive": "archived",
  "components": [
    "appview"
  ],
  "topics": [
    "observability",
    "logging",
    "sentry"
  ],
  "keywords": [
    "Prometheus removal",
    "HTTP metrics",
    "Tap spans",
    "redaction",
    "Sentry transaction",
    "run_id"
  ],
  "summary": "This package records the original AppView metrics, structured logging, panic/5xx capture and Sentry tracing foundation.",
  "currentDocs": [
    "docs/development/logging-error-reporting.md",
    "docs/architecture.md"
  ],
  "code": [
    "appview/internal/observability",
    "appview/internal/middleware"
  ],
  "evidence": [
    "docs/changes/2026-06-29-appview-observability/05-implementation-plan.md#final-verification",
    "docs/changes/2026-06-29-appview-observability/06-implementation-review.md"
  ],
  "supersededBy": [
    "2026-10-02-logging-error-reporting"
  ],
  "followUps": []
}
---

# Original AppView observability foundation

## Outcome and decisions

This package records the original AppView metrics, structured logging, panic/5xx capture and Sentry tracing foundation. Its approved implementation includes route-pattern telemetry, PDS operation instrumentation, protected read/write logs and shutdown flushing. Historical requirements mention Prometheus and broader identifier omission; those choices must not override the later logging/error-reporting design and current protected diagnostic entry points.

## Evidence and limits

The original review is Approved with notes and full Go verification is recorded. Hosted Sentry smoke testing was not executed in that pass. The later logging package substantially refines ownership, native causes, public/private selection and supported sinks; use it and the maintained guide for current work.

Historical evidence: [05-implementation-plan.md — Final Verification](../../../docs/changes/2026-06-29-appview-observability/05-implementation-plan.md#final-verification); [06-implementation-review.md](../../../docs/changes/2026-06-29-appview-observability/06-implementation-review.md).

## Current entry points

[docs/development/logging-error-reporting.md](../../../docs/development/logging-error-reporting.md); [docs/architecture.md](../../../docs/architecture.md). This card preserves the recorded change outcome; it does not replace current code or maintained guidance.
