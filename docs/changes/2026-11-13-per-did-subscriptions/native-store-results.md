# Native store catalog results

Date: 13 November 2026. The maintainer approved the prepared Apple changes and Google monthly DRAFT creation; both provider plans reached **applied**, with every item applied and no apply errors. This is store configuration evidence, not native purchase or production deployment proof.

## Apple

- Kept Plus and verified all 175 existing prices unchanged, including GBP 2.49 / USD 2.99 per month.
- Created `social.craftsky.app.business.monthly.v2`; all 175 prices and availability entries match the legacy Business product, including GBP 8.99 / USD 9.99.
- Plus and replacement Business share group `22404620`. Its internal reference remains CraftSky Plus; its en-GB display name is CraftSky Subscriptions.
- Saved and reloaded the group order: **Business level 1; Plus level 2**.
- Updated account-owned descriptions, review notes and privacy URL. Both products read `READY_TO_SUBMIT`; the supplied review images are the explicitly approved blank placeholders, so replace them with genuine submitted-build purchase UI before App Review. Nothing was submitted.
- Attached the replacement RevenueCat product `prod21eeb1c1b6` to **business** and **paid**, and replaced the old Apple association in the Business monthly package. Google and Test Store associations remain intact. Independent read-back passed.
- Retained legacy Apple Business product `social.craftsky.app.business.monthly`, its entitlement links and history. Its store availability was not changed. A retirement-only preview was discarded without applying it; retiring legacy availability remains a separate decision after existing subscription effects are assessed.

## Google Play

Both expected products now exist with en-GB account-owned descriptions and monthly auto-renewing base plans:

| Product | Base-plan state | GB | US |
| --- | --- | --- | --- |
| `social.craftsky.app.plus:monthly` | ACTIVE | GBP 2.49 | USD 2.99 |
| `social.craftsky.app.business:monthly` | ACTIVE | GBP 8.99 | USD 9.99 |

The first approved batch created these as drafts with GB/US seeds. The maintainer then explicitly approved the complete worldwide price maps and activation. The second plan applied both items without errors. Independent live reads confirmed **ACTIVE**, P1M billing, 174 named available regions per product and an exact match for every approved regional amount/currency. The GB/US prices remain unchanged. No purchase was submitted. Seven-day grace remains; earlier Console inspection showed automatic account hold of 53 days, totalling 60 with grace, and no grace/hold override was included in the plan.

Google's all-supported-countries policy is now configured: **174 named regions plus future markets**. The Console's count of 175 includes its future-market option. Google generated the prices from USD 2.99 / USD 9.99; the proposed and applied Plus GB override retains GBP 2.49 instead of the generated GBP 2.69. Business GB remains GBP 8.99. Future-market defaults are Plus USD 2.99 / EUR 2.71 and Business USD 9.99 / EUR 9.05, with new-subscriber availability enabled. Live read-back matched those defaults exactly.

RevenueCat's initial validation caught seven obsolete country aliases in the name-to-code preview mapping; these were replaced with current ISO codes before approval or apply. The final plan had no blockers or warning-severity messages, only informational notices that RevenueCat uses Google regions version 2026/01. No Apple price map was used for Google. See [the full price CSV](google-worldwide-prices.csv), [the approved plan snapshot](google-worldwide-plan.json) and [independent verification](google-worldwide-results.json).

## Application release handoff

Add this entry **additively** to AppView's runtime `REVENUECAT_PRODUCT_MAPPINGS`:

```json
"prod21eeb1c1b6": {"appId":"app118e6a032e","tier":"business"}
```

Retain `prod65a34336b0` and all existing mappings. The RevenueCat App Store app and client public key do not change. Local examples intentionally target Test Store and were not rewritten with production IDs. No runtime credentials, Render configuration, migrations or releases were changed.

Real iOS/Android acceptance remains: offerings/prices, same-group StoreKit ownership, purchase/restore to the original account UUID, account switching, Business upgrade and deferred Plus downgrade. SDK/provider configuration alone does not prove these paths.

Full compact identifiers and verification flags are in [native-store-results.json](native-store-results.json); the approved original diffs remain in [native-store-preview.json](native-store-preview.json).

