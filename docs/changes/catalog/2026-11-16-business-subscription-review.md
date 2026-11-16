---
{
  "id": "2026-11-16-business-subscription-review",
  "title": "Business parity and subscription lifecycle review",
  "source": "docs/changes/2026-11-16-business-subscription-review",
  "implementation": "implemented",
  "archive": "active",
  "components": ["app", "web-client", "appview"],
  "topics": ["business", "subscriptions", "parity", "billing"],
  "keywords": ["event timezone", "DST", "multilingual alt text", "conflict reload", "subscription cache", "promotional expiry", "restore from paywall", "pending payment", "RevenueCat duplicate pagination", "cross-environment checkout", "CraftSkyTester1", "billing retry loop", "business draft retention"],
  "summary": "Fixes business timezone, validation and account races, refreshes shared subscription access, preserves restore intent, hardens provider/checkout boundaries and verifies the tester's valid sandbox Business access.",
  "currentDocs": ["docs/development/web-client-parity.md", "docs/adr/027-per-did-subscriptions.md"],
  "code": ["app/lib/business", "app/lib/subscriptions", "app/lib/profile/pages/edit_profile_dialog.dart", "web-client/src/lib/features/settings", "web-client/src/lib/features/shell/subscription-access.svelte.ts", "web-client/src/lib/features/social/business-model.ts", "appview/internal/api/business_profile.go", "appview/internal/api/business_event.go", "appview/internal/subscriptions/web_checkout.go", "appview/internal/integrations/revenuecat/client.go"],
  "evidence": ["docs/changes/2026-11-16-business-subscription-review/README.md"],
  "supersededBy": [],
  "followUps": ["Physical store purchase/restore validation and Android device UI remain release checks; this review does not deploy or verify production webhook delivery."]
}
---

# Business parity and subscription lifecycle review

Records concrete fixes and local/sandbox validation from the business and
subscription audit. The tester's expired Plus history is distinct from its
currently valid Business subscription through 15 December 2026.
