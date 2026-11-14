---
{
  "id": "2026-07-08-reposts-and-quotes",
  "title": "Reposts and compact quote previews",
  "source": "docs/changes/2026-07-08-reposts-and-quotes",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview",
    "lexicon"
  ],
  "topics": [
    "feeds",
    "reposts",
    "quotes"
  ],
  "keywords": [
    "quoteView",
    "quote CID",
    "feed item wrapper",
    "repost reason",
    "quote hydration"
  ],
  "summary": "Added repost feed reasons and reusable one-level quote hydration across post read and create surfaces.",
  "currentDocs": [
    "docs/adr/023-quote-preview-image-selection.md",
    "docs/development/home-feeds.md"
  ],
  "code": [
    "appview/internal/api",
    "app/lib/feed"
  ],
  "evidence": [
    "docs/changes/2026-07-08-reposts-and-quotes/05-implementation-plan.md#verification-log",
    "docs/changes/2026-07-08-reposts-and-quotes/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Reposts and compact quote previews

The change introduces repost/quote behavior with interoperable record references, outer feed-item identity and compact `quoteView` hydration. Quote previews are shared across the home timeline, details, authored lists, comments, branch replies, search and quote-create responses rather than implemented as unrelated response shapes.

The final review approves fixes for timeline wrappers, repost reasons, quote-CID canonicalization and missing hydration on read/create paths. The execution log records successive review corrections, backend tests and focused Flutter API/create-provider regressions. Its older unchecked review box does not override the later approved review.

Quote context remains bounded; the implementation does not recursively expand arbitrary embedded threads. Current quote-image-selection and post-editing ADRs refine the present behavior. Use this package for the distinction between the outer post, repost reason and quoted target, and consult current source for visibility, moderation and record-version policies.

Evidence: [05-implementation-plan.md](../2026-07-08-reposts-and-quotes/05-implementation-plan.md#verification-log); [06-implementation-review.md](../2026-07-08-reposts-and-quotes/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
