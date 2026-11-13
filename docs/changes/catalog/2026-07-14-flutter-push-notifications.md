---
{
  "id": "2026-07-14-flutter-push-notifications",
  "title": "Flutter push lifecycle, routing and preferences",
  "source": "docs/changes/2026-07-14-flutter-push-notifications",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "notifications",
    "account-isolation",
    "push"
  ],
  "keywords": [
    "NotificationService",
    "NotificationRuntime",
    "pending notification open",
    "Firebase ADC",
    "new-count",
    "optimistic preferences"
  ],
  "summary": "The Flutter push slice wires provider-neutral notification services, account-bound routing and notification preferences to the existing AppView delivery model.",
  "currentDocs": [
    "app/README.md",
    "docs/development/notification-previews.md"
  ],
  "code": [
    "app/lib/notifications",
    "appview/internal/push"
  ],
  "evidence": [
    "docs/changes/2026-07-14-flutter-push-notifications/05-implementation-plan.md#final-verification",
    "docs/changes/2026-07-14-flutter-push-notifications/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Physical notification permission, delivery/open and device lifecycle checks remain external release evidence."
  ]
}
---

# Flutter push lifecycle, routing and preferences

## Outcome and decisions

The Flutter push slice wires provider-neutral notification services, account-bound routing and notification preferences to the existing AppView delivery model. Secure routing state and pending opens must retain the exact recipient/session; stale completions cannot relabel a new account. Follow-up simplification removes pass-through orchestration while preserving Riverpod overrides, optimistic generation handling and public-boundary tests.

## Evidence and limits

The review is Approved with notes and its cleanup suggestions are followed by a recorded simplification pass, 72 notification tests, clean analysis and backend race verification. ADC sender-selection checks demonstrate configuration only, not live Firebase invocation or phone delivery.

Historical evidence: [05-implementation-plan.md — Final Verification](../../../docs/changes/2026-07-14-flutter-push-notifications/05-implementation-plan.md#final-verification); [06-implementation-review.md](../../../docs/changes/2026-07-14-flutter-push-notifications/06-implementation-review.md).

## Current entry points

[app/README.md](../../../app/README.md); [docs/development/notification-previews.md](../../../docs/development/notification-previews.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
