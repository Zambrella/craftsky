---
{
  "id": "2026-05-29-notifications-mvp",
  "title": "Read-only in-app notifications foundation",
  "source": "docs/changes/2026-05-29-notifications-mvp",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "notifications",
    "pagination",
    "navigation"
  ],
  "keywords": [
    "reply focus",
    "NotificationRow",
    "newness",
    "opaque cursor",
    "loading retry"
  ],
  "summary": "The package records the initial read-only Notifications tab backed by AppView-derived social activity and cursor pagination.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md",
    "appview/README.md"
  ],
  "code": [
    "app/lib/notifications",
    "appview/internal/api/notification_store.go",
    "appview/internal/notifications"
  ],
  "evidence": [
    "docs/changes/2026-05-29-notifications-mvp/05-implementation-plan.md#review-fix-verification",
    "docs/changes/2026-05-29-notifications-mvp/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Use later push/newness records for delivery and seen-state policy; this MVP was read-only."
  ]
}
---

# Read-only in-app notifications foundation

## Outcome and decisions

The package records the initial read-only Notifications tab backed by AppView-derived social activity and cursor pagination. Follow notifications navigate to profiles; like/repost subjects open threads; replies preserve focus identity. Corrections align camelCase reply focus fields, loading/error controls and route-intent coverage. Later push and new-count packages extend this foundation without making this early plan current notification policy.

## Evidence and limits

The original Changes required review is followed by explicit review-fix execution and focused real-database/API and Flutter tests. Its analyzer limitation concerned an unrelated dummy profile repository; treat that as a dated test boundary, not a current defect.

Historical evidence: [05-implementation-plan.md — Review Fix Verification](../../../docs/changes/2026-05-29-notifications-mvp/05-implementation-plan.md#review-fix-verification); [06-implementation-review.md](../../../docs/changes/2026-05-29-notifications-mvp/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [app/README.md](../../../app/README.md); [appview/README.md](../../../appview/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
