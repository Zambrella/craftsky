---
{
  "id": "2026-06-19-appview-search-foundation",
  "title": "AppView search foundation",
  "source": "docs/changes/2026-06-19-appview-search-foundation",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview"
  ],
  "topics": [
    "search",
    "pagination",
    "query-performance"
  ],
  "keywords": [
    "SearchStore",
    "FTS",
    "trigram",
    "exact hashtags",
    "recent searches",
    "project filters",
    "EXPLAIN"
  ],
  "summary": "The recorded backend slice implements hashtag, profile, post and project search, private recent searches and craft-specific top hashtags.",
  "currentDocs": [
    "appview/README.md",
    "docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md"
  ],
  "code": [
    "appview/internal/api/search_store.go",
    "appview/internal/api/search_post_store.go"
  ],
  "evidence": [
    "docs/changes/2026-06-19-appview-search-foundation/05-implementation-plan.md#final-verification",
    "docs/changes/2026-06-19-appview-search-foundation/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical representative EXPLAIN/query-plan review was not completed against the applied search migration."
  ]
}
---

# AppView search foundation

## Outcome and decisions

The recorded backend slice implements hashtag, profile, post and project search, private recent searches and craft-specific top hashtags. It separates exact hashtag identity from fuzzy/profile and keyword matching, preserves visibility/moderation boundaries and uses opaque bounded cursor pages. Review corrections cover project popularity, profile pagination, recent-query normalization and avatar hydration before client UI consumption.

## Evidence and limits

The final execution record reports full Go/race verification and correction suites passing. The review is Approved with notes. Representative EXPLAIN verification was attempted before the required search migration was applied, leaving planner evidence separate from passing seeded correctness tests.

Historical evidence: [05-implementation-plan.md — Final Verification](../../../docs/changes/2026-06-19-appview-search-foundation/05-implementation-plan.md#final-verification); [06-implementation-review.md](../../../docs/changes/2026-06-19-appview-search-foundation/06-implementation-review.md).

## Current entry points

[appview/README.md](../../../appview/README.md); [docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md](../../../docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
