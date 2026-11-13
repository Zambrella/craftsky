---
{
  "id": "2026-07-31-scheduled-posts",
  "title": "Durable scheduled publication and private media staging",
  "source": "docs/changes/2026-07-31-scheduled-posts",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "scheduling",
    "media",
    "durability"
  ],
  "keywords": [
    "Publishing lease",
    "Needs attention",
    "frozen record identity",
    "S3 staging",
    "IT-009"
  ],
  "summary": "Implemented the private scheduled-publication queue with stable PDS effects, recoverable media staging and account-isolated management.",
  "currentDocs": [
    "docs/architecture.md",
    "docs/development/logging-error-reporting.md",
    "docs/operations/production-render.md"
  ],
  "code": [
    "appview/internal/scheduledposts",
    "app/lib/scheduled_posts"
  ],
  "evidence": [
    "docs/changes/2026-07-31-scheduled-posts/05-implementation-plan.md#seventh-correction-pass-verification",
    "docs/changes/2026-07-31-scheduled-posts/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Retain separate real-device, provider and private-object-store release evidence for scheduled publication."
  ]
}
---

# Durable scheduled publication and private media staging

Scheduling keeps unpublished payloads and media private in AppView/PostgreSQL and S3-compatible storage. Owner-authenticated management uses the full composer; workers publish through stable PDS record identity, bounded retry policy and cleanup rather than treating a database insert as public publication.

Seven review-correction passes close behavior and evidence gaps. The final accepted record includes real Settings navigation and Needs attention badge state across account switches, plus recovery tests that interrupt four real publication boundaries, retain/reclaim Publishing leases, reject stale completion and preserve the frozen body/record key. Recovery must not silently create a new public record.

The approved review leaves only non-blocking scenario-consolidation and manual UI notes. Local automated recovery is not live PDS/S3 or device proof. Current architecture, subscription policy and worker diagnostic guide govern present authority, eligibility and operational behavior.

Evidence: [05-implementation-plan.md](../2026-07-31-scheduled-posts/05-implementation-plan.md#seventh-correction-pass-verification); [06-implementation-review.md](../2026-07-31-scheduled-posts/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
