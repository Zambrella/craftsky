---
{
  "id": "2026-07-02-appview-sentry-observability",
  "title": "Sentry observability consolidation",
  "source": "docs/changes/2026-07-02-appview-sentry-observability",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview"
  ],
  "topics": [
    "observability",
    "logging",
    "metrics"
  ],
  "keywords": [
    "Prometheus",
    "Sentry Logs",
    "metric recorder",
    "trace sampling",
    "/metrics"
  ],
  "summary": "Consolidated AppView observability around gated Sentry sinks and removed its Prometheus runtime endpoint.",
  "currentDocs": [
    "docs/development/logging-error-reporting.md",
    "docs/development/configuration.md"
  ],
  "code": [
    "appview/internal/observability",
    "appview/internal/middleware"
  ],
  "evidence": [
    "docs/changes/2026-07-02-appview-sentry-observability/05-implementation-plan.md",
    "docs/changes/2026-07-02-appview-sentry-observability/06-implementation-review.md#test-evidence",
    "docs/changes/2026-10-02-logging-error-reporting/06-validation-evidence.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Sentry observability consolidation

This package removes AppView-owned Prometheus collectors and `/metrics`, introduces domain metric methods and independently gated Sentry logs/traces/metrics, and places bounded privacy filtering around errors and panics. Prometheus modules remain indirect dependencies of Indigo; runtime removal does not imply an empty transitive dependency graph.

The final review reports no blocking finding and passing focused observability/configuration/HTTP/PDS/Tap tests. Hosted indexing of Sentry Logs and Application Metrics remained a separate manual check. The original race-enabled Compose gate was not rerun during that review.

Exception ownership, original-cause preservation and privacy rules were substantially refined by the later October logging package and November audits. The maintained logging guide is authoritative now; do not copy this early package's classified-placeholder diagnostic approach back into source. Use it for consolidation history, independent collection gates and the absence of a supported `/metrics` surface.

Evidence: [05-implementation-plan.md](../2026-07-02-appview-sentry-observability/05-implementation-plan.md); [06-implementation-review.md](../2026-07-02-appview-sentry-observability/06-implementation-review.md#test-evidence); [06-validation-evidence.md](../2026-10-02-logging-error-reporting/06-validation-evidence.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
