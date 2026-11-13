---
{
  "id": "2026-05-28-timeline-feed-appview",
  "title": "AppView chronological timeline foundation",
  "source": "docs/changes/2026-05-28-timeline-feed-appview",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview"
  ],
  "topics": [
    "feeds",
    "pagination",
    "relationships"
  ],
  "keywords": [
    "ListTimeline",
    "indexed_at",
    "opaque cursor",
    "follow graph",
    "exact-full final page"
  ],
  "summary": "Introduced the indexed chronological timeline and recorded fixes for final-page cursors and changing follows between pages.",
  "currentDocs": [
    "docs/development/home-feeds.md",
    "docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md"
  ],
  "code": [
    "appview/internal/api/post_store.go",
    "appview/internal/api"
  ],
  "evidence": [
    "docs/changes/2026-05-28-timeline-feed-appview/05-implementation-plan.md#remediation-verification-results",
    "docs/changes/2026-05-28-timeline-feed-appview/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# AppView chronological timeline foundation

The server timeline selects the viewer's own posts and currently followed authors from AppView projections, excludes comment/reply/repost rows in this original slice, and orders by indexed time plus URI. It reuses normal post and engagement hydration and does not read craft data directly from PDSes.

The initial review identified an extra cursor on an exactly full final page and absent follow/unfollow-between-pages coverage. The later remediation section records both corrected behaviors, their tests and the completed execution checklist. Read that section together with the older review rather than reopening its original findings automatically.

Later home-feed and repost work extends this foundation. Current home-feed guidance and current visibility predicates control present behavior. The package is useful for opaque cursor eligibility and current-follow-graph semantics, not a frozen description of every current feed item.

Evidence: [05-implementation-plan.md](../2026-05-28-timeline-feed-appview/05-implementation-plan.md#remediation-verification-results); [06-implementation-review.md](../2026-05-28-timeline-feed-appview/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
