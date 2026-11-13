---
{
  "id": "2026-07-16-direct-push-notification-routing",
  "title": "Direct notification destinations and account binding",
  "source": "docs/changes/2026-07-16-direct-push-notification-routing",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "notifications",
    "navigation",
    "account-isolation"
  ],
  "keywords": [
    "notification payload v1",
    "binding-first",
    "reply focus",
    "BUG-001",
    "BUG-002"
  ],
  "summary": "Replaced resolver-driven notification opens with minimal typed destinations guarded by the active account and readiness lifecycle.",
  "currentDocs": [
    "docs/development/notification-previews.md",
    "docs/development/notification-category-filtering.md"
  ],
  "code": [
    "app/lib/notifications",
    "appview/internal/notifications",
    "appview/internal/push"
  ],
  "evidence": [
    "docs/changes/2026-07-16-direct-push-notification-routing/05-implementation-plan.md#follow-notification-action-final-verification",
    "docs/changes/2026-07-16-direct-push-notification-routing/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical approval was superseded after later fixes; a fresh review/device result should accompany any release claim."
  ]
}
---

# Direct notification destinations and account binding

Notification opens parse a versioned minimal payload, infer provider-neutral destinations, establish the bound account first and let destination reads enforce authorization. Replies carry comment focus; permanent access failures suppress retained content, while retryable refresh failures provide destination-scoped recovery.

Later fixes fence navigation after sign-out, account changes, onboarding changes, disposal and away-and-back readiness transitions. Additional execution entries cover Android/in-app defects, notification-row presentation and follow actions, with Flutter notification and PostgreSQL/race evidence.

The review file explicitly marks its earlier approval superseded and requiring re-review after BUG-001/BUG-002. That is a review-evidence limit, not proof the implemented fixes are absent. Its historical direct-routing rationale remains useful, but current notification preview/filtering guides and source govern today's payload, content and presentation rules. OS delivery and signed-device navigation require separate release evidence.

Evidence: [05-implementation-plan.md](../2026-07-16-direct-push-notification-routing/05-implementation-plan.md#follow-notification-action-final-verification); [06-implementation-review.md](../2026-07-16-direct-push-notification-routing/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
