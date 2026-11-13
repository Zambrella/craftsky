---
{
  "id": "2026-08-25-follower-growth-metrics",
  "title": "Follower-growth observations and charts",
  "source": "docs/changes/2026-08-25-follower-growth-metrics",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "profiles",
    "growth",
    "metrics"
  ],
  "keywords": [
    "follower_growth",
    "daily snapshot",
    "availability window",
    "latest metadata",
    "IR-018"
  ],
  "summary": "Added private follower-growth observations with bounded availability metadata and charts grounded in actual observation ranges.",
  "currentDocs": [
    "app/README.md",
    "docs/development/logging-error-reporting.md"
  ],
  "code": [
    "appview/internal/followergrowth",
    "appview/internal/api/follower_growth.go",
    "app/lib/profile"
  ],
  "evidence": [
    "docs/changes/2026-08-25-follower-growth-metrics/05-implementation-plan.md#step-44-boundary-corrected-regression-pass",
    "docs/changes/2026-08-25-follower-growth-metrics/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Release-device chart/accessibility checks and the historical capacity-qualified gate need separate current evidence."
  ]
}
---

# Follower-growth observations and charts

Follower-growth reporting captures observations rather than inventing missing historical data. Range availability and latest metadata must identify the actual first and last in-range observations, so sparse samples and boundary dates do not imply unsupported coverage.

The final review accepts the IR-018 correction and finds no remaining Must-level implementation defect. The execution record includes domain, persistence, chart, range and metadata correction tests, together with Go/Flutter verification. Some broader AppView evidence was affected by shared capacity, and representative release-device checks remain distinct from automated chart assertions.

Use this package to trace observation boundaries, account access and chart-state decisions. Current capture-worker source and diagnostic policy govern scheduling, retry escalation and operational reporting; the old package is not a promise of backfilled history, live deployment or full physical-device verification.

Evidence: [05-implementation-plan.md](../2026-08-25-follower-growth-metrics/05-implementation-plan.md#step-44-boundary-corrected-regression-pass); [06-implementation-review.md](../2026-08-25-follower-growth-metrics/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
