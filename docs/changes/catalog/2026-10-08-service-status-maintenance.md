---
{
  "id": "2026-10-08-service-status-maintenance",
  "title": "Service-status polling and maintenance presentation",
  "source": "docs/changes/2026-10-08-service-status-maintenance",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "service-status",
    "web"
  ],
  "topics": [
    "service-status",
    "maintenance",
    "recovery"
  ],
  "keywords": [
    "expiry grant",
    "announcement revision",
    "fake R2",
    "maintenance modal",
    "irreversible expiry"
  ],
  "summary": "Added independently fetched service-status grants and maintenance/announcement UI with bounded expiry and explicit dismissal.",
  "currentDocs": [
    "docs/operations/service-status.md",
    "service-status/README.md"
  ],
  "code": [
    "app/lib/service_status",
    "service-status",
    "web"
  ],
  "evidence": [
    "docs/changes/2026-10-08-service-status-maintenance/05-implementation-plan.md#final-outcome-2026-10-09"
  ],
  "supersededBy": [],
  "followUps": [
    "Hosted status publication/cache verification and physical assistive-technology checks require separate current evidence."
  ]
}
---

# Service-status polling and maintenance presentation

Status polling is independent of authentication and does not grant business authority. Validated snapshots carry bounded expiry; a wall-clock correction cannot revive a grant already expired. Local development tooling and Worker fixtures exercise polling, validation and state replacement without publishing hosted status.

The later presentation amendment replaces the initial announcement banner with a dismissible modal over the retained application. Button, outside tap, Escape and root back persist the dismissed revision; dismissal does not pop the underlying route, and a persistence failure retains a retryable modal. Maintenance behavior remains separate.

The execution log records full/focused Flutter, loopback iPhone simulator, browser adapter, Python/Worker and website checks. Fake R2, simulator and local browser evidence do not establish Cloudflare provisioning, hosted cache behavior, physical assistive technology or real provider journeys. The operations runbook controls current publication and recovery procedures.

Evidence: [05-implementation-plan.md](../2026-10-08-service-status-maintenance/05-implementation-plan.md#final-outcome-2026-10-09). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
