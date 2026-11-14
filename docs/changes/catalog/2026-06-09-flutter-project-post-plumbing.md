---
{
  "id": "2026-06-09-flutter-project-post-plumbing",
  "title": "Flutter project models, feeds and mutation caches",
  "source": "docs/changes/2026-06-09-flutter-project-post-plumbing",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "projects",
    "serialization",
    "cache"
  ],
  "keywords": [
    "ProjectDetailsMapper",
    "userProjectsProvider",
    "unknown project details",
    "rollback",
    "DID handle cache keys"
  ],
  "summary": "The Flutter-only slice connects project-post models and create payloads to profile project lists and mutation cache updates.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md"
  ],
  "code": [
    "app/lib/projects/models",
    "app/lib/feed/providers",
    "app/lib/profile/providers"
  ],
  "evidence": [
    "docs/changes/2026-06-09-flutter-project-post-plumbing/05-implementation-plan.md#review-fix-verification",
    "docs/changes/2026-06-09-flutter-project-post-plumbing/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Serialize project details through the reviewed Project/ProjectDetailsMapper boundary, not concrete generated detail mappers."
  ]
}
---

# Flutter project models, feeds and mutation caches

## Outcome and decisions

The Flutter-only slice connects project-post models and create payloads to profile project lists and mutation cache updates. Known and unknown project details retain the wire shape through Project/ProjectDetailsMapper. Create/delete/like/repost changes reconcile both DID-keyed and handle-keyed project caches, preserve the separate profile Posts list, and roll back failures.

## Evidence and limits

The re-review is Approved with notes after provider-cache tests cover all mutation directions and both live key forms. The execution record reports whole-app analysis, the complete 527-test Flutter suite and focused regressions passing. A non-blocking mapper misuse risk is documented.

Historical evidence: [05-implementation-plan.md — Review Fix Verification](../../../docs/changes/2026-06-09-flutter-project-post-plumbing/05-implementation-plan.md#review-fix-verification); [06-implementation-review.md](../../../docs/changes/2026-06-09-flutter-project-post-plumbing/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [app/README.md](../../../app/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
