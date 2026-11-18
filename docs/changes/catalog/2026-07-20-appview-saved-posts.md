---
{
  "id": "2026-07-20-appview-saved-posts",
  "title": "Private saved posts and folders",
  "source": "docs/changes/2026-07-20-appview-saved-posts",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview"
  ],
  "topics": [
    "saved-posts",
    "privacy",
    "pagination"
  ],
  "keywords": [
    "SavedPostService",
    "tri-state saves",
    "folder assignment",
    "keyset pagination",
    "saved-state indexes",
    "viewer state"
  ],
  "summary": "The package records private AppView saved-post state, folder CRUD, viewer-state hydration and filtered saved-list pagination.",
  "currentDocs": [
    "docs/architecture.md",
    "appview/README.md"
  ],
  "code": [
    "appview/internal/api/saved_post_store.go",
    "appview/internal/api/saved_post.go"
  ],
  "evidence": [
    "docs/changes/2026-07-20-appview-saved-posts/05-implementation-plan.md#c4-ir-003--it-014",
    "docs/changes/2026-07-20-appview-saved-posts/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Use the later correction execution record to assess historical test coverage; the initial review is not a current defect inventory."
  ]
}
---

# Private saved posts and folders

## Outcome and decisions

The package records private AppView saved-post state, folder CRUD, viewer-state hydration and filtered saved-list pagination. Saves remain private rather than PDS records. Folder assignment, canonical viewer state and removal behavior are owner-scoped and respect current visibility. Later corrections strengthen real lifecycle tests, every production query shape and migration preservation of unrelated interaction/private data.

## Evidence and limits

The review retains its initial Changes required findings; the later execution log records C1–C5 corrections. Its representative 1,200-row query matrix demonstrates intended forward/backward indexes across twelve list shapes, and seeded migration up/down/up checks preserve unrelated rows.

Historical evidence: [05-implementation-plan.md — C4: IR-003 / IT-014](../../../docs/changes/2026-07-20-appview-saved-posts/05-implementation-plan.md#c4-ir-003--it-014); [06-implementation-review.md](../../../docs/changes/2026-07-20-appview-saved-posts/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [appview/README.md](../../../appview/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
