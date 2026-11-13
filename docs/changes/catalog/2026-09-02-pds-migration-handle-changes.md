---
{
  "id": "2026-09-02-pds-migration-handle-changes",
  "title": "DID-first identity across handle and PDS migrations",
  "source": "docs/changes/2026-09-02-pds-migration-handle-changes",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "identity",
    "auth",
    "migration"
  ],
  "keywords": [
    "SessionCoordinator",
    "PDS authority",
    "metadata cache TTL",
    "same DID",
    "endpoint migration",
    "handle change"
  ],
  "summary": "The recorded change keeps account authority and content operations tied to stable DID identity across handle changes and PDS migrations.",
  "currentDocs": [
    "docs/architecture.md",
    "appview/README.md",
    "app/README.md"
  ],
  "code": [
    "appview/internal/auth/session_coordinator.go",
    "app/lib/auth"
  ],
  "evidence": [
    "docs/changes/2026-09-02-pds-migration-handle-changes/05-implementation-plan.md#execution-log",
    "docs/changes/2026-09-02-pds-migration-handle-changes/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical physical/provider migration and handle-change staging checks remain separate release evidence."
  ]
}
---

# DID-first identity across handle and PDS migrations

## Outcome and decisions

The recorded change keeps account authority and content operations tied to stable DID identity across handle changes and PDS migrations. Effect-time authority checks cover persisted OAuth endpoints, repository verification, scheduling/deletion and retained client sessions. Bounded OAuth metadata caches enforce constructor-level TTL and abandoned-loader capacity rules; telemetry separates cache outcomes without exposing credentials.

## Evidence and limits

The final review is Approved after IR-001 through IR-003 and a later cache-constructor evidence pass. It records broad automated coverage and repository release gates passing. Device and external migration/provider staging checks remained required before release, so local passing tests are not a live migration claim.

Historical evidence: [05-implementation-plan.md — Execution Log](../../../docs/changes/2026-09-02-pds-migration-handle-changes/05-implementation-plan.md#execution-log); [06-implementation-review.md](../../../docs/changes/2026-09-02-pds-migration-handle-changes/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [appview/README.md](../../../appview/README.md); [app/README.md](../../../app/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
