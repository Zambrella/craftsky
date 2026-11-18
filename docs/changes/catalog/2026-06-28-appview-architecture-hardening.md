---
{
  "id": "2026-06-28-appview-architecture-hardening",
  "title": "AppView HTTP architecture hardening",
  "source": "docs/changes/2026-06-28-appview-architecture-hardening",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview"
  ],
  "topics": [
    "http",
    "security",
    "admission"
  ],
  "keywords": [
    "RoutePolicy",
    "body limits",
    "rate_limited",
    "Retry-After",
    "CORS"
  ],
  "summary": "Established policy-driven route wrapping, body limits, rate classifications and protected HTTP diagnostics.",
  "currentDocs": [
    "docs/architecture.md",
    "docs/development/configuration.md",
    "docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md"
  ],
  "code": [
    "appview/internal/routes",
    "appview/internal/middleware"
  ],
  "evidence": [
    "docs/changes/2026-06-28-appview-architecture-hardening/05-implementation-plan.md",
    "docs/changes/2026-06-28-appview-architecture-hardening/06-implementation-review.md#test-evidence"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# AppView HTTP architecture hardening

The hardening pass introduces shared route-policy metadata and middleware application for the registered `/v1/` surface. Success bodies remain bare JSON; errors use the standard envelope. No-body/default/upload admission, validated device IDs, CORS headers and rate-limit responses become consistent across routes.

The final review accepts correction of earlier policy-enforcement gaps, with all-policy route integration tests and passing focused/full Go package sweeps. Numeric rate defaults and production tuning remain operational matters. The historical review did not execute the canonical race-enabled Compose gate, so its plain Go sweep should not be presented as equivalent evidence.

Later federated-HTTP, lifecycle and admission audits strengthen this foundation. Consult current architecture, configuration and API guidance for today's isolation and scaling rules. In particular, old single-process limiter assumptions are historical context, not permission to alter production replicas or replace established network boundaries.

Evidence: [05-implementation-plan.md](../2026-06-28-appview-architecture-hardening/05-implementation-plan.md); [06-implementation-review.md](../2026-06-28-appview-architecture-hardening/06-implementation-review.md#test-evidence). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
