---
{
  "id": "2026-09-17-unified-pds-record-mutations",
  "title": "Tap-authoritative command and projection lifecycle",
  "source": "docs/changes/2026-09-17-unified-pds-record-mutations",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "mutations",
    "ingestion",
    "idempotency"
  ],
  "keywords": [
    "PDS command",
    "Tap-authoritative",
    "same-key retry",
    "ambiguous outcome",
    "source revision",
    "legacy winner tables"
  ],
  "summary": "The recorded redesign unifies public record mutations around PDS authority, durable command identity and Tap-derived projections.",
  "currentDocs": [
    "docs/architecture.md",
    "AGENTS.md",
    "docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md"
  ],
  "code": [
    "appview/internal/pdscommands",
    "appview/internal/pdseffects",
    "appview/internal/ingestion",
    "app/lib/shared/api"
  ],
  "evidence": [
    "docs/changes/2026-09-17-unified-pds-record-mutations/05-implementation-plan.md#post-commit-legacy-table-removal-2026-09-28",
    "docs/changes/2026-09-17-unified-pds-record-mutations/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Tap-authoritative command and projection lifecycle

## Outcome and decisions

The recorded redesign unifies public record mutations around PDS authority, durable command identity and Tap-derived projections. Lost or malformed responses must retain the same frozen command key instead of automatically duplicating a write. External writes, replay, resync, source revisions and accepted overlays converge through guarded command/effect and projection boundaries. A later cleanup removes winner-era physical set tables while historical rollback recreates empty compatibility tables only.

## Evidence and limits

The final review is Approved with notes after follow-up corrections and legacy removal. Execution records real-PostgreSQL tests, complete backend/release gates and client regression/analysis checks. The package describes a coherent lifecycle, not permission for generic direct PDS reads/writes or deleting unrelated record namespaces.

Historical evidence: [05-implementation-plan.md — Post-commit legacy-table removal (2026-09-28)](../../../docs/changes/2026-09-17-unified-pds-record-mutations/05-implementation-plan.md#post-commit-legacy-table-removal-2026-09-28); [06-implementation-review.md](../../../docs/changes/2026-09-17-unified-pds-record-mutations/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [AGENTS.md](../../../AGENTS.md); [docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md](../../../docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md). This card preserves the recorded change outcome; it does not replace current code or maintained guidance.
