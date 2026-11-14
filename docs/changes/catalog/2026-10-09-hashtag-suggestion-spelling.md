---
{
  "id": "2026-10-09-hashtag-suggestion-spelling",
  "title": "Representative hashtag spelling with case-insensitive identity",
  "source": "docs/changes/2026-10-09-hashtag-suggestion-spelling",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "hashtags",
    "search",
    "indexing"
  ],
  "keywords": [
    "MeMadeMay",
    "most-used spelling",
    "case insensitive",
    "ambiguous source fields",
    "migration 000090",
    "facet autocomplete"
  ],
  "summary": "The recorded implementation displays the most-used observed hashtag spelling while combining identity and counts case-insensitively.",
  "currentDocs": [
    "appview/README.md",
    "docs/architecture.md"
  ],
  "code": [
    "appview/internal/postutil",
    "appview/internal/api/facet_store.go",
    "app/lib/shared/rich_text"
  ],
  "evidence": [
    "docs/changes/2026-10-09-hashtag-suggestion-spelling/05-implementation-plan.md#main-merge-verification-2026-10-09",
    "docs/changes/2026-10-09-hashtag-suggestion-spelling/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Production migration and deployed spelling behavior require release-specific verification."
  ]
}
---

# Representative hashtag spelling with case-insensitive identity

## Outcome and decisions

The recorded implementation displays the most-used observed hashtag spelling while combining identity and counts case-insensitively. Each surface selects spelling from its own eligible posts, preserving visibility, ranking, pagination and composer insertion. The owner explicitly approved rejecting ambiguous recognized source aliases/duplicate keys. Source validation, migration checks and a database constraint retain that exception rather than silently choosing conflicting input.

## Evidence and limits

The final review is Approved for local implementation readiness. The execution log records merge reconciliation, the hashtag migration renumbering and passing service-backed race/client checks. Production migration execution and deployment were not authorized or demonstrated by that approval.

Historical evidence: [05-implementation-plan.md — Main Merge Verification (2026-10-09)](../../../docs/changes/2026-10-09-hashtag-suggestion-spelling/05-implementation-plan.md#main-merge-verification-2026-10-09); [06-implementation-review.md](../../../docs/changes/2026-10-09-hashtag-suggestion-spelling/06-implementation-review.md).

## Current entry points

[appview/README.md](../../../appview/README.md); [docs/architecture.md](../../../docs/architecture.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
