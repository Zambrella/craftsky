---
{
  "id": "2026-06-21-search-refinements",
  "title": "Search suggestions, result separation and recent-query contracts",
  "source": "docs/changes/2026-06-21-search-refinements",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "search",
    "facets",
    "privacy"
  ],
  "keywords": [
    "craftType tokens",
    "search suggestions",
    "project filters",
    "recent search normalization",
    "FacetStore.SearchHashtagSuggestions"
  ],
  "summary": "The non-UI refinement slice aligns backend and Flutter data contracts for suggestions, submitted search, project browsing and private recents.",
  "currentDocs": [
    "app/README.md",
    "appview/README.md",
    "docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md"
  ],
  "code": [
    "appview/internal/api/search_store.go",
    "appview/internal/api/facet_store.go",
    "app/lib/search"
  ],
  "evidence": [
    "docs/changes/2026-06-21-search-refinements/05-implementation-plan.md#verification-log",
    "docs/changes/2026-06-21-search-refinements/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "This package does not establish rendered search UI behavior or production-scale query performance."
  ]
}
---

# Search suggestions, result separation and recent-query contracts

## Outcome and decisions

The non-UI refinement slice aligns backend and Flutter data contracts for suggestions, submitted search, project browsing and private recents. Posts and Projects remain disjoint, project filters belong under the Projects API, and craft tokens use canonical wire values. A later correction applies the same visible/top-level predicates to facet hashtag suggestions, closing the review’s count-parity finding.

## Evidence and limits

The execution checklist records completed correction work and focused/broader verification, while accepting that rendered UI end-to-end coverage is outside this slice. Production-scale performance and profile-recent display freshness remain dated follow-up boundaries rather than claimed outcomes.

Historical evidence: [05-implementation-plan.md — Verification Log](../../../docs/changes/2026-06-21-search-refinements/05-implementation-plan.md#verification-log); [06-implementation-review.md](../../../docs/changes/2026-06-21-search-refinements/06-implementation-review.md).

## Current entry points

[app/README.md](../../../app/README.md); [appview/README.md](../../../appview/README.md); [docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md](../../../docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
