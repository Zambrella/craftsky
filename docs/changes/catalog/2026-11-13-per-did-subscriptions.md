---
{
  "id": "2026-11-13-per-did-subscriptions",
  "title": "Account-owned subscriptions, Stripe checkout and promotional access",
  "source": "docs/changes/2026-11-13-per-did-subscriptions",
  "implementation": "implemented",
  "archive": "active",
  "components": [
    "app",
    "appview",
    "web-client"
  ],
  "topics": [
    "subscriptions",
    "billing",
    "authentication"
  ],
  "keywords": [
    "RevenueCat",
    "Stripe Managed Payments",
    "promotional grants",
    "eager identity",
    "per-DID",
    "native ownership",
    "webCheckoutPending",
    "app_user_id",
    "Test Store local development",
    "web feature upgrade prompts",
    "direct feature paywalls",
    "Plus action icons",
    "composer schedule icon CSS isolation",
    "free save folder loading",
    "web catalog prices",
    "embedded Checkout modal",
    "RevenueCat web paywalls",
    "Managed Payments fallback"
  ],
  "summary": "Subscriptions and promotional access belong to individual accounts, with guarded native identity and durable Stripe Managed Payments checkout.",
  "currentDocs": [
    "docs/adr/027-per-did-subscriptions.md",
    "docs/architecture.md",
    "docs/operations/subscription-billing-recovery.md"
  ],
  "code": [
    "appview/internal/subscriptions",
    "appview/internal/integrations/revenuecat",
    "appview/internal/integrations/stripe",
    "app/lib/subscriptions",
    "web-client/src/lib/features/settings/Billing.svelte",
    "web-client/src/lib/features/shell/billing-identity.ts",
    "scripts/revenuecat-webhook-dev",
    "web-client/src/lib/features/shell/subscription-feature.ts",
    "appview/internal/subscriptions/web_prices.go",
    "web-client/src/lib/features/settings/RevenueCatPaywall.svelte",
    "web-client/src/lib/features/shell/FeaturePaywall.svelte",
    "app/lib/subscriptions/widgets/plus_feature_lock.dart",
    "appview/internal/subscriptions/web_paywall.go",
    "appview/internal/subscriptions/web_checkout_cancel.go"
  ],
  "evidence": [
    "docs/changes/2026-11-13-per-did-subscriptions/validation.md",
    "docs/changes/2026-11-13-per-did-subscriptions/06-implementation-review.md",
    "docs/changes/2026-11-13-per-did-subscriptions/eager-identity.md",
    "docs/changes/2026-11-13-per-did-subscriptions/promotional-grants.md",
    "docs/changes/2026-11-13-per-did-subscriptions/provider-setup.md",
    "docs/changes/2026-11-13-per-did-subscriptions/main-merge.md",
    "docs/changes/2026-11-13-per-did-subscriptions/local-development.md",
    "docs/changes/2026-11-13-per-did-subscriptions/web-feature-prompts.md",
    "docs/changes/2026-11-13-per-did-subscriptions/subscription-page-parity.md",
    "docs/changes/2026-11-13-per-did-subscriptions/embedded-checkout.md",
    "docs/changes/2026-11-13-per-did-subscriptions/revenuecat-web-paywalls.md",
    "docs/changes/2026-11-13-per-did-subscriptions/feature-paywall-entry.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Production deployment and integration activation remain separate release steps.",
    "Real-device native purchases, cross-provider ownership and live application checkout require separate acceptance evidence."
  ]
}
---

# Account-owned subscriptions, Stripe checkout and promotional access

ADR 027 replaces the separate billing-holder and cross-DID licence assignment
model: each account owns its private RevenueCat UUID and subscription. Optional
identity synchronization runs on account activation without blocking sign-in;
native operations retain store-ownership, session and final-dispatch guards.

Locked paid features open their required paywall in place on Flutter and web.
Web paywalls open as bottom sheets, expanding to the larger Stripe checkout
modal on purchase selection and returning to the sheet on provider cancellation.
The web sheet dismisses on outside clicks and uses only provider chrome; the larger checkout retains its own
Close checkout control and allows up to 1200px for Stripe desktop checkout.
There is no outer sheet header or manual refresh
and last-checked footer.
Web checkout uses the same published RevenueCat paywalls as native, with Stripe
Managed Payments preferred and ordinary billing fallback accepted. Both shared
paywalls published corrected account-ownership and billing copy on 15 November
2026 with their existing layout and fonts preserved. Reconciliation
remains authoritative. Durable checkout recovery applies to retained direct
Stripe operations. Promotional grants
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
