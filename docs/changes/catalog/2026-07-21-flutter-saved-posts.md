---
{
  "id": "2026-07-21-flutter-saved-posts",
  "title": "Flutter saved posts and private collections",
  "source": "docs/changes/2026-07-21-flutter-saved-posts",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "saved-content",
    "collections",
    "state"
  ],
  "keywords": [
    "Saved overview",
    "folder move",
    "explicit save",
    "collection refresh",
    "restart retry"
  ],
  "summary": "Added Flutter saved-post collections and corrected retry, empty-refresh and mounted destination reconciliation paths.",
  "currentDocs": [
    "app/README.md",
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/saved_posts",
    "appview/internal/api/saved_post.go"
  ],
  "evidence": [
    "docs/changes/2026-07-21-flutter-saved-posts/05-implementation-plan.md#second-correction-verification-2026-07-22",
    "docs/changes/2026-07-21-flutter-saved-posts/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Consult current release evidence for the package’s manual saved-collection device scenarios."
  ]
}
---

# Flutter saved posts and private collections

Saved posts and folders use private AppView APIs with account-scoped state, opaque identifiers and explicit collection operations. The Flutter surface covers overview/list ordering, save/unsave, moving between folders, navigation and error recovery rather than creating public PDS bookmark records.

Several review rounds matter. Earlier restart-retry, cancellation/error projection and private DTO formatting issues were corrected. Later execution entries address mounted destination collections using a different sort and make completely empty collections refreshable. The final plan records second-correction verification, user-feedback conventions and a production-router navigation regression. Its stale review findings must be read alongside those follow-ups.

Recorded automation includes focused saved-post/PostCard tests, Flutter analysis/full suites and database-backed race checks. Manual scenarios remain separately identified. Current subscription gating and local draft policy should be consulted before reusing this package's original UI/access assumptions.

Evidence: [05-implementation-plan.md](../2026-07-21-flutter-saved-posts/05-implementation-plan.md#second-correction-verification-2026-07-22); [06-implementation-review.md](../2026-07-21-flutter-saved-posts/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
