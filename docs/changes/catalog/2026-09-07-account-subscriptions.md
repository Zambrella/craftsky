---
{
  "id": "2026-09-07-account-subscriptions",
  "title": "AppView account-assigned subscription foundation",
  "source": "docs/changes/2026-09-07-account-subscriptions",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview"
  ],
  "topics": [
    "subscriptions",
    "billing",
    "account-deletion"
  ],
  "keywords": [
    "RevenueCat",
    "billing identity",
    "license assignment",
    "will_not_renew",
    "pending payment",
    "ADR 014"
  ],
  "summary": "The recorded backend foundation delegates store lifecycle interpretation to RevenueCat while AppView owns private billing identities, licenses, DID assignment and access.",
  "currentDocs": [
    "docs/adr/027-per-did-subscriptions.md",
    "docs/operations/subscription-billing-recovery.md"
  ],
  "code": [
    "appview/internal/subscriptions",
    "appview/internal/integrations/revenuecat"
  ],
  "evidence": [
    "docs/changes/2026-09-07-account-subscriptions/05-implementation-plan.md#ir-012-and-ir-013-final-verification",
    "docs/changes/2026-09-07-account-subscriptions/06-implementation-review.md"
  ],
  "supersededBy": ["2026-11-13-per-did-subscriptions"],
  "followUps": [
    "Production RevenueCat/store configuration and billing recovery evidence are separate from local implementation approval."
  ]
}
---

# AppView account-assigned subscription foundation

ADR 027 replaces this package's separate payer/beneficiary assignment model with
account-owned subscriptions. The provider reconciliation and private billing
foundation remain; consult the replacement change for current ownership,
checkout and promotional-access behavior.

## Outcome and decisions

The recorded backend foundation delegates store lifecycle interpretation to RevenueCat while AppView owns private billing identities, licenses, DID assignment and access. The payer and beneficiary account are separate identities. Reconciliation and owner lifecycle transitions fail closed on missing/unknown/pending evidence. The final deletion correction permits active or expired access only when every relevant persisted row is non-pending and exactly will_not_renew; renewable/resumable evidence blocks deletion atomically.

## Evidence and limits

The final focused review is Approved and closes exported-type documentation plus the revised deletion boundary. It is narrower than proving store/RevenueCat operation in production. Consult the execution record and maintained billing recovery runbook for complete evidence and incident actions.

Historical evidence: [05-implementation-plan.md — IR-012 And IR-013 Final Verification](../../../docs/changes/2026-09-07-account-subscriptions/05-implementation-plan.md#ir-012-and-ir-013-final-verification); [06-implementation-review.md](../../../docs/changes/2026-09-07-account-subscriptions/06-implementation-review.md).

## Current entry points

[docs/adr/027-per-did-subscriptions.md](../../../docs/adr/027-per-did-subscriptions.md); [docs/operations/subscription-billing-recovery.md](../../../docs/operations/subscription-billing-recovery.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
