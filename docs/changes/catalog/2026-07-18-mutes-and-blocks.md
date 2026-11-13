---
{
  "id": "2026-07-18-mutes-and-blocks",
  "title": "Mute, block and hidden comment branches",
  "source": "docs/changes/2026-07-18-mutes-and-blocks",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "relationships",
    "privacy",
    "moderation"
  ],
  "keywords": [
    "muted_ancestor_uri",
    "explicit reveal",
    "block propagation",
    "duplicate unblock",
    "IT-029"
  ],
  "summary": "Implemented private mutes and public blocks, including muted ancestry filtering before reply pagination.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md"
  ],
  "code": [
    "appview/internal/relationships",
    "appview/internal/api/post_store.go",
    "app/lib/feed"
  ],
  "evidence": [
    "docs/changes/2026-07-18-mutes-and-blocks/05-implementation-plan.md#step-36-ir-007--ut-005-it-029",
    "docs/changes/2026-07-18-mutes-and-blocks/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "The package leaves its MAN-001/MAN-002 device checks as external evidence gaps."
  ]
}
---

# Mute, block and hidden comment branches

Mutes are private AppView preferences; blocks use the public graph and enforce stricter visibility across replies, mentions, quotes and notifications. The correction pass covers PDS-only duplicate unblock records, transactional mute/delivery cancellation and account-owned Flutter relationship state.

The older review's final blocker concerned muted ancestors outside the current reply page or focused window. Step 36 records a real-Postgres failing fixture, propagation of `muted_ancestor_uri` through recursive ordinary/focused/continuation queries, exclusion before LIMIT and passing broader Go/Flutter verification. Explicit temporary reveal remains the client path for opening a hidden branch. Do not repeat the older page-local-only implementation from the initial plan.

The recorded manual device gates remain distinct from those automated results. Current lifecycle, visibility and account-isolation code must be consulted for present behavior; this package explains why hidden-branch ancestry must survive pagination boundaries.

Evidence: [05-implementation-plan.md](../2026-07-18-mutes-and-blocks/05-implementation-plan.md#step-36-ir-007--ut-005-it-029); [06-implementation-review.md](../2026-07-18-mutes-and-blocks/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
