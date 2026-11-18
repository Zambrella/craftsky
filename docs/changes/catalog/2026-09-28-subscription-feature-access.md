---
{
  "id": "2026-09-28-subscription-feature-access",
  "title": "Subscription-aware feature access",
  "source": "docs/changes/2026-09-28-subscription-feature-access",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "subscriptions",
    "access-control",
    "lifecycle"
  ],
  "keywords": [
    "SelfAccess",
    "Free re-save",
    "Business lapse",
    "cached owner event",
    "MAN-001"
  ],
  "summary": "Applied server-authoritative subscription access to saved collections, Business controls and cached owner-event presentation.",
  "currentDocs": [
    "docs/adr/014-account-assigned-subscription-licenses.md",
    "docs/operations/subscription-billing-recovery.md"
  ],
  "code": [
    "appview/internal/subscriptions",
    "app/lib/subscriptions",
    "app/lib/business"
  ],
  "evidence": [
    "docs/changes/2026-09-28-subscription-feature-access/05-implementation-plan.md#re-review-remediation-2026-09-29",
    "docs/changes/2026-09-28-subscription-feature-access/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "MAN-001 VoiceOver/TalkBack and focus order remains a signed-in physical-device check in this package."
  ]
}
---

# Subscription-aware feature access

The feature-access slice separates ordinary free actions from paid feature entry points and protects current-account state during refresh, lapse and account changes. Subscription eligibility remains server-authoritative; cached presentation cannot continue exposing privileged controls while fresh access is uncertain.

Initial review corrections cover Free re-save responses, mounted Business detail, management-route lapse and missing lifecycle sequences. The later re-review remediation explicitly fixes owner-event detail and Business action refresh gaps. The final implementation checklist records passing automated/focused/full checks and leaves only the physical-device accessibility/focus check as a documented external follow-up.

Use this package for lapse and uncertain-access behavior rather than as a store billing integration guide. Current entitlement, assignment and recovery code determine present state. Historical review wording should be read with the later correction section; it is not evidence that those refresh gaps still exist.

Evidence: [05-implementation-plan.md](../2026-09-28-subscription-feature-access/05-implementation-plan.md#re-review-remediation-2026-09-29); [06-implementation-review.md](../2026-09-28-subscription-feature-access/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
