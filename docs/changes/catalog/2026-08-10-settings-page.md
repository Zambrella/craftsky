---
{
  "id": "2026-08-10-settings-page",
  "title": "Settings and owner-authorized account deletion",
  "source": "docs/changes/2026-08-10-settings-page",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "settings",
    "account-deletion",
    "durability"
  ],
  "keywords": [
    "exact handle confirmation",
    "fresh OAuth",
    "membership last",
    "safety tombstones",
    "late PDS commit",
    "uncertain object"
  ],
  "summary": "The package records Settings expansion and the explicitly owner-authorized permanent CraftSky deletion flow with later exact-key safety corrections.",
  "currentDocs": [
    "docs/architecture.md",
    "AGENTS.md",
    "docs/operations/production-render.md"
  ],
  "code": [
    "app/lib/settings",
    "appview/internal/accountdeletion",
    "appview/internal/ownerlifecycle"
  ],
  "evidence": [
    "docs/changes/2026-08-10-settings-page/05-implementation-plan.md",
    "docs/changes/2026-08-10-settings-page/06-implementation-review.md#evidence-and-traceability"
  ],
  "supersededBy": [],
  "followUps": [
    "Production/provider settlement and physical account-deletion behavior require separate authorized evidence."
  ]
}
---

# Settings and owner-authorized account deletion

## Outcome and decisions

The package records Settings expansion and the explicitly owner-authorized permanent CraftSky deletion flow with later exact-key safety corrections. Deletion is limited to the authenticated owner’s registered CraftSky record collections and matching private object generations. It does not delete the DID/PDS account, unrelated namespaces or blobs. Minimal non-secret tombstones retain uncertain effects until convergence; unknown remote settlement must never fabricate data-free completion.

## Evidence and limits

The review’s opening verdict is historical: its later Section 9 is the approved correction verdict. Execution records real PostgreSQL/MinIO late-effect scenarios, migration and broad client/backend checks. A finite remote settlement bound was not demonstrated, so unknown object outcomes remain reconciliation work.

Historical evidence: [05-implementation-plan.md](../../../docs/changes/2026-08-10-settings-page/05-implementation-plan.md); [06-implementation-review.md — Evidence and traceability](../../../docs/changes/2026-08-10-settings-page/06-implementation-review.md#evidence-and-traceability).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [AGENTS.md](../../../AGENTS.md); [docs/operations/production-render.md](../../../docs/operations/production-render.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
