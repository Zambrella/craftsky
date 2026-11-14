---
{
  "id": "2026-11-13-per-did-subscriptions",
  "title": "Account-owned subscriptions, Stripe checkout and promotional access",
  "source": "docs/changes/2026-11-13-per-did-subscriptions",
  "implementation": "implemented",
  "archive": "active",
  "components": ["app", "appview", "web-client"],
  "topics": ["subscriptions", "billing", "authentication"],
  "keywords": ["RevenueCat", "Stripe Managed Payments", "promotional grants", "eager identity", "per-DID", "native ownership", "webCheckoutPending", "app_user_id"],
  "summary": "Subscriptions and promotional access belong to individual accounts, with guarded native identity and durable Stripe Managed Payments checkout.",
  "currentDocs": ["docs/adr/027-per-did-subscriptions.md", "docs/architecture.md", "docs/operations/subscription-billing-recovery.md"],
  "code": ["appview/internal/subscriptions", "appview/internal/integrations/revenuecat", "appview/internal/integrations/stripe", "app/lib/subscriptions", "web-client/src/lib/features/settings/Billing.svelte", "web-client/src/lib/features/shell/billing-identity.ts"],
  "evidence": ["docs/changes/2026-11-13-per-did-subscriptions/validation.md", "docs/changes/2026-11-13-per-did-subscriptions/06-implementation-review.md", "docs/changes/2026-11-13-per-did-subscriptions/eager-identity.md", "docs/changes/2026-11-13-per-did-subscriptions/promotional-grants.md", "docs/changes/2026-11-13-per-did-subscriptions/provider-setup.md", "docs/changes/2026-11-13-per-did-subscriptions/main-merge.md"],
  "supersededBy": [],
  "followUps": ["Production deployment and integration activation remain separate release steps.", "Real-device native purchases, cross-provider ownership and live application checkout require separate acceptance evidence."]
}
---

# Account-owned subscriptions, Stripe checkout and promotional access

ADR 027 replaces the separate billing-holder and cross-DID licence assignment
model: each account owns its private RevenueCat UUID and subscription. Optional
identity synchronization runs on account activation without blocking sign-in;
native operations retain store-ownership, session and final-dispatch guards.

Stripe web checkout requires Managed Payments and uses durable, account-scoped
idempotency and reconciliation before authorizing access. Promotional grants
are separate access sources with expiry/revocation and valid paid overlap. Both
clients allow an early subscription and explain immediate billing; remaining
complimentary time is not added to the paid term.

The implementation review corrected malformed-provider access loss, stale shared
access, disposed-operation handling and native duplicate-purchase hazards.
Local backend, Flutter and web checks establish implementation behavior, while
provider sandbox configuration and native compilation remain separate from real
device/store or production acceptance. Follow the maintained ADR and recovery
guide for current behavior; historical catalogue and provider notes are evidence
of the recorded setup rather than permission to activate billing.
