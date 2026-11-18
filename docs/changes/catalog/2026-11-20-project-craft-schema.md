---
{
  "id": "2026-11-20-project-craft-schema",
  "title": "Project craft schema expansion",
  "source": "docs/changes/2026-11-20-project-craft-schema",
  "implementation": "proposed",
  "archive": "active",
  "components": ["app", "appview", "lexicon", "web-client"],
  "topics": ["projects", "composer", "search", "taxonomy"],
  "keywords": ["designTags", "needleSizesMm", "hookSizesMm", "techniques", "machine knitting", "needle felting"],
  "summary": "Reviewed requirements for two release phases covering existing craft improvements and six new primary crafts.",
  "currentDocs": [
    "docs/adr/020-project-evolution-search-and-multiple-crafts.md",
    "docs/development/search-fields.md"
  ],
  "code": ["appview/internal/projectcatalog", "lexicon/social/craftsky/project"],
  "evidence": [
    "docs/changes/2026-11-20-project-craft-schema/01-requirements.md",
    "docs/changes/2026-11-20-project-craft-schema/02-acceptance-tests.md",
    "docs/changes/2026-11-20-project-craft-schema/03-document-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Use the revised acceptance test specification for coding planning once the maintainer requests it.",
    "Verify scalar-size publishing and editing with legacy mobile and web builds before enabling plural-list writes."
  ]
}
---

# Project craft schema expansion

Phase 1 retains shared design tags, adds craft-specific suggestions and techniques, and defines additive string lists for needle and hook sizes with scalar-size compatibility. Phase 2 adds six primary crafts and extends the editors, profile interests, onboarding and discovery across AppView, Flutter and Svelte.

The [requirements](../2026-11-20-project-craft-schema/01-requirements.md) record the maintainer's decisions, starter vocabulary and remaining rollout checks. Scheduling covers the current authoring and publication flow. The scalar-size compatibility requirement applies to public posts and saved local drafts. The [acceptance test specification](../2026-11-20-project-craft-schema/02-acceptance-tests.md) maps those decisions to client scenarios, AppView integration coverage and rollout checks.

The [document review](../2026-11-20-project-craft-schema/03-document-review.md) identified five gaps. The revised acceptance test specification adds the assertions and maps each finding to its updated test.
