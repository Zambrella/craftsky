---
{
  "id": "2026-09-06-post-interaction-details",
  "title": "Post interaction detail lists",
  "source": "docs/changes/2026-09-06-post-interaction-details",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "posts",
    "interactions",
    "pagination"
  ],
  "keywords": [
    "likes list",
    "reposts list",
    "quotes list",
    "PostCard consumers",
    "REG-002"
  ],
  "summary": "Added bounded interaction-detail lists while restricting their summaries and navigation to thread-detail contexts.",
  "currentDocs": [
    "app/README.md",
    "docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md"
  ],
  "code": [
    "appview/internal/api/post_interaction_lists_store.go",
    "app/lib/feed"
  ],
  "evidence": [
    "docs/changes/2026-09-06-post-interaction-details/05-implementation-plan.md#correction-execution-notes",
    "docs/changes/2026-09-06-post-interaction-details/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "The historical final review still requires manual responsive and screen-reader checks before release approval."
  ]
}
---

# Post interaction detail lists

The feature exposes detailed like, repost and quote participation through policy-aware AppView reads and Flutter lists. It keeps interaction summaries and response-list callbacks in thread details rather than adding them indiscriminately to every feed/search/profile card. Quote hydration reuses current identity and visibility boundaries.

Several execution corrections address loading/retry/continuation states, handle hydration and query bounds. The final review accepts the production-wiring regression covering every `PostCard` consumer, including notifications, and tests the permitted root/comment/nested-reply contexts. Approval remains Changes required only for mandatory responsive and screen-reader evidence; it is not a new source-defect verdict.

Use the package to understand where interaction drill-down belongs and how pagination/privacy compose. Automated widget and database evidence remain separate from those manual gates. Consult current card consumers and handlers before extending interaction controls to another surface.

Evidence: [05-implementation-plan.md](../2026-09-06-post-interaction-details/05-implementation-plan.md#correction-execution-notes); [06-implementation-review.md](../2026-09-06-post-interaction-details/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
