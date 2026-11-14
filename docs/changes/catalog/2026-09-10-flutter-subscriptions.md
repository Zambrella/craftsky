---
{
  "id": "2026-09-10-flutter-subscriptions",
  "title": "Flutter subscription purchase, restore and assignment",
  "source": "docs/changes/2026-09-10-flutter-subscriptions",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "subscriptions",
    "billing",
    "account-isolation"
  ],
  "keywords": [
    "RevenueCatUI",
    "purchase reconciliation",
    "restore",
    "dormant licenses",
    "assignment",
    "billing owner"
  ],
  "summary": "The recorded mobile slice adds subscription purchase/restore, reconciliation and account-assignment UI on the AppView billing foundation.",
  "currentDocs": [
    "docs/adr/014-account-assigned-subscription-licenses.md",
    "docs/operations/subscription-billing-recovery.md",
    "app/README.md"
  ],
  "code": [
    "app/lib/subscriptions",
    "appview/internal/subscriptions"
  ],
  "evidence": [
    "docs/changes/2026-09-10-flutter-subscriptions/05-implementation-plan.md#eighth-correction-verification",
    "docs/changes/2026-09-10-flutter-subscriptions/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical MAN-001 through MAN-005 store, sandbox, RevenueCat and device gates remain external activation prerequisites."
  ]
}
---

# Flutter subscription purchase, restore and assignment

## Outcome and decisions

The recorded mobile slice adds subscription purchase/restore, reconciliation and account-assignment UI on the AppView billing foundation. Purchase and restored license relationships must reconcile authoritatively before the UI reports usable access. Active and dormant licenses retain independent IDs, timing, store/product state and actions; an expired predecessor must not borrow the replacement’s entitlement state. Paid-feature enforcement is a separate scope.

## Evidence and limits

The final review is Approved with notes after eight correction passes. Execution records 2,450 Flutter tests, analysis, backend/release gates and unsigned Android/iOS builds. RevenueCat, store/sandbox and physical-device prerequisites still block production activation under MAN-001 through MAN-005.

Historical evidence: [05-implementation-plan.md — Eighth Correction Verification](../../../docs/changes/2026-09-10-flutter-subscriptions/05-implementation-plan.md#eighth-correction-verification); [06-implementation-review.md](../../../docs/changes/2026-09-10-flutter-subscriptions/06-implementation-review.md).

## Current entry points

[docs/adr/014-account-assigned-subscription-licenses.md](../../../docs/adr/014-account-assigned-subscription-licenses.md); [docs/operations/subscription-billing-recovery.md](../../../docs/operations/subscription-billing-recovery.md); [app/README.md](../../../app/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
