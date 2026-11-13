---
{
  "id": "2026-06-01-flutter-facets-ui",
  "title": "Flutter facet editing and suggestions",
  "source": "docs/changes/2026-06-01-flutter-facets-ui",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "rich-text",
    "composer",
    "profiles"
  ],
  "keywords": [
    "FacetTextEditingController",
    "mentions",
    "hashtags",
    "descriptionFacets",
    "debounce"
  ],
  "summary": "Added local facet editing, suggestions and rendered rich text with an explicitly accepted widget-local debounce.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md"
  ],
  "code": [
    "app/lib/shared/rich_text",
    "app/lib/feed",
    "app/lib/profile"
  ],
  "evidence": [
    "docs/changes/2026-06-01-flutter-facets-ui/05-implementation-plan.md#step-28-follow-up--persistent-editable-facet-styling",
    "docs/changes/2026-06-01-flutter-facets-ui/05-implementation-plan.md#skipped-review-finding-ir-003-debounceprovider-deviation",
    "docs/changes/2026-06-01-flutter-facets-ui/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Flutter facet editing and suggestions

The Flutter facet work covers mention/hashtag parsing, composer and profile editing, suggestion interaction, rich-text rendering and transport of facet data. Editable tokens use the theme's primary colour through the text-editing controller. No external identity lookup or direct PDS resolution was introduced in the editor.

The updated review accepts fixes for token styling and the historical AppView rejection of `descriptionFacets`. A widget-local injectable Timer deliberately replaces the planned provider-family debounce: the maintainer explicitly accepted that smaller implementation. Do not treat that documented deviation as unfinished work merely because the coding plan describes another pattern.

The evidence records focused facet/editor/profile rejection regressions and broader Flutter checks. The historical unsupported-profile-field behavior describes that server snapshot; current API and profile implementations must be consulted before assuming that rejection still applies.

Evidence: [05-implementation-plan.md](../2026-06-01-flutter-facets-ui/05-implementation-plan.md#step-28-follow-up--persistent-editable-facet-styling); [06-implementation-review.md](../2026-06-01-flutter-facets-ui/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.

Section pointers: [skipped-review-finding-ir-003-debounceprovider-deviation](../2026-06-01-flutter-facets-ui/05-implementation-plan.md#skipped-review-finding-ir-003-debounceprovider-deviation).
