---
{
  "id": "2026-10-08-search-improvements",
  "title": "PostgreSQL search matching and typo correction",
  "source": "docs/changes/2026-10-08-search-improvements",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "search",
    "matching",
    "privacy"
  ],
  "keywords": [
    "English word forms",
    "craft vocabulary",
    "typo correction",
    "reliable-first ranking",
    "tuple cursor",
    "private canaries"
  ],
  "summary": "The recorded search pass improves word matching, bounded typo matching, selected craft equivalences and searchable authored fields while preserving separate Posts and Projects.",
  "currentDocs": [
    "appview/README.md",
    "docs/architecture.md"
  ],
  "code": [
    "appview/internal/api/search_post_store.go",
    "appview/internal/api/search_store.go",
    "app/lib/search"
  ],
  "evidence": [
    "docs/changes/2026-10-08-search-improvements/05-implementation-plan.md#completion-notes",
    "docs/changes/2026-10-08-search-improvements/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# PostgreSQL search matching and typo correction

## Outcome and decisions

The recorded search pass improves word matching, bounded typo matching, selected craft equivalences and searchable authored fields while preserving separate Posts and Projects. Original query wording and existing visibility/hydration contracts remain intact. Reliable matches rank ahead of corrections with stable tuple pagination. The implementation uses PostgreSQL helpers and does not require new infrastructure or rewriting public PDS records.

## Evidence and limits

The final review is Approved after strengthening the privacy fixture to seed and read actual private-content markers through observed handlers. Later execution distinguishes the original missing-test-setup defect from a production privacy defect, and records required-database regression checks. Representative large-scale performance remains a measurement question rather than an implied benchmark result.

Historical evidence: [05-implementation-plan.md — Completion Notes](../../../docs/changes/2026-10-08-search-improvements/05-implementation-plan.md#completion-notes); [06-implementation-review.md](../../../docs/changes/2026-10-08-search-improvements/06-implementation-review.md).

## Current entry points

[appview/README.md](../../../appview/README.md); [docs/architecture.md](../../../docs/architecture.md). This card preserves the recorded change outcome; it does not replace current code or maintained guidance.
