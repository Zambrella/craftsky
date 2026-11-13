---
{
  "id": "2026-05-25-follow-unfollow-mvp",
  "title": "Follow and unfollow foundation",
  "source": "docs/changes/2026-05-25-follow-unfollow-mvp",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "relationships",
    "profiles",
    "federation"
  ],
  "keywords": [
    "app.bsky.graph.follow",
    "viewerIsFollowing",
    "duplicate follows",
    "non-Craftsky profile"
  ],
  "summary": "Implemented interoperable follow records, indexed graph state and authoritative Flutter follow controls.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md"
  ],
  "code": [
    "appview/internal/relationships",
    "appview/internal/api/follow.go",
    "app/lib/profile"
  ],
  "evidence": [
    "docs/changes/2026-05-25-follow-unfollow-mvp/05-implementation-plan.md",
    "docs/changes/2026-05-25-follow-unfollow-mvp/06-implementation-review.md#test-evidence"
  ],
  "supersededBy": [],
  "followUps": [
    "The historical review did not record its live Tap/PDS smoke checks."
  ]
}
---

# Follow and unfollow foundation

The follow foundation writes and deletes standard `app.bsky.graph.follow` records through the server-held PDS session. Tap/indexer convergence owns durable graph state; response overlays make immediate profile interaction responsive without replacing that ownership. Duplicate follow records collapse into one active relationship.

Profile responses gain viewer-follow state, CraftSky membership and bounded social counts. External profiles can be hydrated without pretending unknown counts are zero. Flutter uses optimistic interaction state, then replaces it from the server or rolls back on failure; PDS credentials stay server-side.

The review records closure of graph-persistence, hydration, real viewer-state and duplicate-collapse findings, with passing backend and Flutter profile tests. Unchecked completion boxes in the old plan are not stronger evidence than the execution and review. Live Tap/PDS and query-plan smoke checks were left manual.

Evidence: [05-implementation-plan.md](../2026-05-25-follow-unfollow-mvp/05-implementation-plan.md); [06-implementation-review.md](../2026-05-25-follow-unfollow-mvp/06-implementation-review.md#test-evidence). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
