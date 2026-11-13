---
{
  "id": "2026-06-07-appview-project-posts",
  "title": "AppView project-post indexing and reads",
  "source": "docs/changes/2026-06-07-appview-project-posts",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview"
  ],
  "topics": [
    "projects",
    "indexing",
    "search"
  ],
  "keywords": [
    "details.$type",
    "projectCount",
    "project materialization",
    "craft-specific columns"
  ],
  "summary": "Added project-bearing post writes, craft-specific indexed projections and profile project reads, with recorded review corrections.",
  "currentDocs": [
    "docs/adr/020-project-evolution-search-and-multiple-crafts.md",
    "docs/adr/021-reserved-post-craft-metadata.md"
  ],
  "code": [
    "appview/internal/projectcatalog",
    "appview/internal/index",
    "appview/internal/api"
  ],
  "evidence": [
    "docs/changes/2026-06-07-appview-project-posts/05-implementation-plan.md#verification",
    "docs/changes/2026-06-07-appview-project-posts/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "The historical execution record qualified its database-dependent verification; consult newer backend gate evidence before relying on old test totals."
  ]
}
---

# AppView project-post indexing and reads

Project-bearing post requests are validated and written through AppView, while Tap projects project metadata into queryable storage. Shared response hydration exposes project content, profile project counts and authored-project listing without adding a second public record format.

The initial review found detail values copied into unrelated craft column families and gaps in route/update/delete tests. Later execution steps record discriminator-sensitive materialization, route and convergence regressions, plus clarification-driven corrections. The final execution checklist distinguishes tests runnable without Compose from PostgreSQL-dependent evidence; a checked box alone is not proof that every database scenario ran at that point.

This is the foundation rather than today's complete craft model. Later evolution/search and reserved-field ADRs govern current field intent. Use the package to understand indexing boundaries and removal/update convergence, then inspect current generated types and project queries.

Evidence: [05-implementation-plan.md](../2026-06-07-appview-project-posts/05-implementation-plan.md#verification); [06-implementation-review.md](../2026-06-07-appview-project-posts/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
