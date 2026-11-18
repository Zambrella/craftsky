---
{
  "id": "2026-07-10-appview-push-notifications",
  "title": "Durable notification delivery and account-wide newness",
  "source": "docs/changes/2026-07-10-appview-push-notifications",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview"
  ],
  "topics": [
    "notifications",
    "push",
    "durability"
  ],
  "keywords": [
    "Firebase",
    "FCM",
    "activation revision",
    "new-count",
    "seen",
    "lease",
    "notification reference roles"
  ],
  "summary": "The recorded backend implementation adds durable notification activation/delivery and later account-wide new-count and seen acknowledgement.",
  "currentDocs": [
    "docs/development/notification-previews.md",
    "docs/development/notification-category-filtering.md",
    "appview/README.md"
  ],
  "code": [
    "appview/internal/push",
    "appview/internal/notifications",
    "appview/internal/api/notifications.go"
  ],
  "evidence": [
    "docs/changes/2026-07-10-appview-push-notifications/05-implementation-plan.md#re-review-verification",
    "docs/changes/2026-07-10-appview-push-notifications/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical Firebase/provider/device gates MAN-001 and MAN-002 are separate from local queue tests."
  ]
}
---

# Durable notification delivery and account-wide newness

## Outcome and decisions

The recorded backend implementation adds durable notification activation/delivery and later account-wide new-count and seen acknowledgement. Activation revisions distinguish genuine reactivation/source replacement from exact replay. Seen state is monotonic and account-scoped across devices; list/count eligibility shares moderation boundaries. Push payload privacy, leased retries and deletion/moderation of every reference role receive dedicated coverage.

## Evidence and limits

The final review is Approved for the newness follow-up. The execution record reports seeded API tests, race suites, Go vet and the canonical backend suite passing. Automated tests did not contact Firebase, so provider acceptance/device receipt remained external gates.

Historical evidence: [05-implementation-plan.md — Re-review Verification](../../../docs/changes/2026-07-10-appview-push-notifications/05-implementation-plan.md#re-review-verification); [06-implementation-review.md](../../../docs/changes/2026-07-10-appview-push-notifications/06-implementation-review.md).

## Current entry points

[docs/development/notification-previews.md](../../../docs/development/notification-previews.md); [docs/development/notification-category-filtering.md](../../../docs/development/notification-category-filtering.md); [appview/README.md](../../../appview/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
