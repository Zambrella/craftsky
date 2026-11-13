# Native store catalog change preview

Prepared 13 November 2026; explicitly approved and applied later that day. This document retains the original reviewed changes. See [native-store-results.md](native-store-results.md) for actual provider read-back and remaining work.

## Apple

- Keep the existing Plus monthly product and all current prices.
- Replace its assignable-account description with "Plus features for your CraftSky account."
- Set the existing group's English (U.K.) customer-visible name to "CraftSky Subscriptions". Its internal reference name remains "CraftSky Plus" so the replacement explicitly reuses that group.
- Create `social.craftsky.app.business.monthly.v2` in that same group, monthly, using all 175 current Business territory prices and availability. GB remains GBP 8.99; US remains USD 9.99. Any territory not explicitly listed is equalized by Apple from US at apply time.
- The new internal reference is "Business Monthly Account" and RevenueCat display name "CraftSky Business Monthly (Account)" to avoid reserved-name conflicts. Its customer-facing name remains "CraftSky Business Monthly".
- Set its description to "Plus benefits and business tools for your account."
- Add privacy URL and accurate account-owned subscription review notes to Plus and the replacement Business product.
- Follow creation with Business-above-Plus ranking and RevenueCat entitlement/package mapping. Retain the legacy Business product and purchase history; retire it from new sales only after verifying the replacement.

The current Apple plan has one product added, one modified, and no blockers. Each product has one warning: no review screenshot was supplied, so RevenueCat will add a blank placeholder. This does not provide an acceptable real purchase-flow screenshot for final App Review; replace placeholders before submission. No submission is included in this work.

## Google Play

The requested final availability is all supported countries. The current empty console requires product creation before it exposes worldwide price generation. This first preview therefore creates both expected subscriptions with `monthly` auto-renewing base plans in **DRAFT**, using en-GB names and account-owned descriptions. They cannot be sold while in DRAFT.

| Product | Initial GB price | Initial US price |
| --- | --- | --- |
| `social.craftsky.app.plus:monthly` | GBP 2.49 | USD 2.99 |
| `social.craftsky.app.business:monthly` | GBP 8.99 | USD 9.99 |

GB/US are initial price seeds, not a change to the worldwide availability decision. After creation, use Play Console's supported-country pricing controls, verify every supported region and preserve these GB/US prices. Review the resulting worldwide pricing before activation. Do not copy Apple's region map wholesale or use Apple's equalization directive for Google.

The Google creation preview has two products added and no blockers. Each item warns that its base plan stays in DRAFT and cannot be sold. Each has an informational notice that RevenueCat writes Google prices using regions version 2026/01, its newest supported version. Grace and account-hold settings are omitted to retain Google defaults.

## Review artifacts and approval

`native-store-preview.json` contains the complete provider-computed diffs, desired states and warnings for both previews. RevenueCat allows only one active plan and one store per plan; the active dashboard plan now contains Apple. The Google preview was computed and saved before replacing the active draft with Apple. Recreate the exact approved Google creation states only after the Apple plan finishes, compare its fresh diffs with this preview, and stop if material changes appear.

The store-state tool requires confirmation after these actual diffs are shown and before apply. Store purchases, Apple review submission, production AppView configuration, migrations and deployment remain separate.

