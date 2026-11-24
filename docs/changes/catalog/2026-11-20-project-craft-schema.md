---
{
  "id": "2026-11-20-project-craft-schema",
  "title": "Project craft schema expansion",
  "source": "docs/changes/2026-11-20-project-craft-schema",
  "implementation": "implemented",
  "archive": "active",
  "components": [
    "app",
    "appview",
    "lexicon",
    "web-client"
  ],
  "topics": [
    "projects",
    "composer",
    "search",
    "taxonomy"
  ],
  "keywords": [
    "projectItem",
    "projectType",
    "projectSubtype",
    "Other",
    "custom subtype",
    "patterns",
    "loom type",
    "embroidery method",
    "felting method"
  ],
  "summary": "Simplifies project authoring around browseable item types, searchable custom subtypes, multiple patterns, and a small set of craft-specific choices.",
  "currentDocs": [
    "docs/adr/020-project-evolution-search-and-multiple-crafts.md",
    "docs/adr/029-additive-project-craft-details.md",
    "docs/development/search-fields.md"
  ],
  "code": [
    "appview/internal/projectcatalog",
    "lexicon/social/craftsky/project"
  ],
  "evidence": [
    "docs/changes/2026-11-20-project-craft-schema/01-requirements.md",
    "docs/changes/2026-11-20-project-craft-schema/project-type-proposal.md",
    "docs/changes/2026-11-20-project-craft-schema/02-acceptance-tests.md",
    "docs/changes/2026-11-20-project-craft-schema/03-document-review.md",
    "docs/changes/2026-11-20-project-craft-schema/04-coding-plan.md",
    "docs/changes/2026-11-20-project-craft-schema/05-implementation-plan.md#simplification-evidence",
    "docs/changes/2026-11-20-project-craft-schema/05-implementation-plan.md#runtime-implementation-evidence",
    "docs/changes/2026-11-20-project-craft-schema/05-implementation-plan.md#corrections-after-implementation-review",
    "docs/changes/2026-11-20-project-craft-schema/05-implementation-plan.md#corrections-after-the-second-review",
    "docs/changes/2026-11-20-project-craft-schema/05-implementation-plan.md#flutter-pattern-text-reversion",
    "docs/changes/2026-11-20-project-craft-schema/06-implementation-review.md",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#alternative-project-item-names",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#consistent-display-for-older-projects",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#flat-craft-filter-form",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#alphabetical-options-and-custom-names-under-other",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#short-labels-for-equivalent-names",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#project-composer-field-order",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#additional-crafts-and-required-status",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#composer-hints-and-mobile-done-action",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#short-craft-method-labels",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#pattern-details-by-craft",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#project-item-group-and-summary",
    "docs/changes/2026-11-20-project-craft-schema/07-ui-polish-notes.md#combined-project-type-picker"
  ],
  "supersededBy": [],
  "followUps": [
    "Confirm both release-phase candidates with the simplified composer and equivalent browse outcomes."
  ]
}
---

# Project craft schema expansion

The active [requirements](../2026-11-20-project-craft-schema/01-requirements.md#approved-simplification) and [item taxonomy](../2026-11-20-project-craft-schema/project-type-proposal.md) describe the simpler project composer, searchable custom subtypes, and retained legacy data. The [acceptance tests](../2026-11-20-project-craft-schema/02-acceptance-tests.md#acceptance-tests-for-the-approved-simplification) cover the changes across AppView and both clients.

## Original expansion history

Phase 1 retains shared design tags, adds craft-specific suggestions and techniques, and defines additive lists for needle and hook sizes and up to five patterns per project. Phase 2 adds six primary crafts and extends the editors, profile interests, onboarding and discovery across AppView, Flutter and Svelte.

The [requirements](../2026-11-20-project-craft-schema/01-requirements.md) record the maintainer's decisions, starter vocabulary and remaining rollout checks. Scheduling covers the current authoring and publication flow. Single-size and single-pattern compatibility applies to public posts and saved local drafts. The [acceptance test specification](../2026-11-20-project-craft-schema/02-acceptance-tests.md) maps those decisions to client scenarios, AppView integration coverage and rollout checks.

The [document review](../2026-11-20-project-craft-schema/03-document-review.md) records the verified resolution of five gaps. The maintainer approved the generated schema for both phases in the local explorer. Knitting records Hand and Machine as independent optional selections.

The [implementation plan](../2026-11-20-project-craft-schema/05-implementation-plan.md#runtime-implementation-evidence) records AppView and client verification. The shared taxonomy selects the craft offers for either phase while keeping known records readable. The [coding plan](../2026-11-20-project-craft-schema/04-coding-plan.md) records the release order and manual checks.

The [implementation review](../2026-11-20-project-craft-schema/06-implementation-review.md) approved the local implementation after verifying the final Flutter correction. The implementation plan records the [initial corrections](../2026-11-20-project-craft-schema/05-implementation-plan.md#corrections-after-implementation-review), the [follow-up on pattern annotations and scheduled visibility](../2026-11-20-project-craft-schema/05-implementation-plan.md#corrections-after-the-second-review), and the [Flutter regression for restoring exact saved pattern text](../2026-11-20-project-craft-schema/05-implementation-plan.md#flutter-pattern-text-reversion).

The [project browse contract](../../development/search-fields.md#filtering-and-visibility) covers the combined Project type filter and the other current craft-specific choices.

The [composer polish notes](../2026-11-20-project-craft-schema/07-ui-polish-notes.md) record the current field and option ordering, top-level styling and grouping in the combined Project type picker, required Status, consistent pattern actions, and shared web field styling.
