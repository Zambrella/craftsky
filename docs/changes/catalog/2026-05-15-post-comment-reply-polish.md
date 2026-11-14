---
{
  "id": "2026-05-15-post-comment-reply-polish",
  "title": "Post, comment and reply polish proposals",
  "source": "docs/changes/2026-05-15-post-comment-reply-polish",
  "implementation": "proposed",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "threads",
    "profiles",
    "navigation"
  ],
  "keywords": [
    "profile Comments tab",
    "focus",
    "oldest-first replies",
    "descendant count",
    "timestamp tooltip"
  ],
  "summary": "This package records draft plans for separating profile posts from comments and polishing focused thread navigation, reply ordering, counts and controls.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md",
    "appview/README.md"
  ],
  "code": [
    "app/lib/feed/pages",
    "app/lib/feed/widgets/post_card.dart",
    "appview/internal/api"
  ],
  "evidence": [
    "docs/changes/2026-05-15-post-comment-reply-polish/plan.md#verification-commands",
    "docs/changes/2026-05-15-post-comment-reply-polish/amendments-2026-05-15.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Reconcile the draft and amendments with implementation history before assigning implemented status."
  ]
}
---

# Post, comment and reply polish proposals

## Outcome and decisions

This package records draft plans for separating profile posts from comments and polishing focused thread navigation, reply ordering, counts and controls. The amendments propose temporary focus highlighting, oldest-first reply branches, full timestamp tooltips and presentation-only hiding of reply counts/repost controls. Recursive descendant counts are intended to stay batched and depth-bounded.

## Evidence and limits

Both source documents are explicitly draft-for-review plans. They contain intended verification commands and risks, but this package has no execution or validation record establishing what landed. Similar present-day UI is not completion evidence for this historical proposal.

Historical evidence: [plan.md — Verification Commands](../../../docs/changes/2026-05-15-post-comment-reply-polish/plan.md#verification-commands); [amendments-2026-05-15.md](../../../docs/changes/2026-05-15-post-comment-reply-polish/amendments-2026-05-15.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [app/README.md](../../../app/README.md); [appview/README.md](../../../appview/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
