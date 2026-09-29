# UI Polish Notes: Flutter Account Subscriptions

## Summary

Refined the subscription page hierarchy and reused CraftSky's established account and control treatments.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | Remove debug information | Hid reconciliation timestamps and raw store/product identifiers while retaining user-relevant access timing. | `app/lib/subscriptions/pages/subscription_page.dart` | Done |
| UIP-002 | Reduce restore emphasis | Rendered Restore purchases as a text button while keeping Manage subscription prominent. | `app/lib/subscriptions/pages/subscription_page.dart` | Done |
| UIP-003 | Theme purchase and assignment actions | Replaced subscription page filled-tonal actions and confirmation actions with existing CraftSky chunky controls. | `app/lib/subscriptions/pages/subscription_page.dart` | Done |
| UIP-004 | Theme assignment modal and account rows | Replaced the Material dialog with `CraftskyDialog` and reused the extracted account-switcher row, including avatar, display name, handle, and customisation. | `app/lib/auth/widgets/account_switcher_content.dart`, `app/lib/subscriptions/pages/subscription_page.dart` | Done |
| UIP-005 | Hide free account labels | Free access now renders no tier badge; paid, loading, and unavailable badges remain visible. | `app/lib/subscriptions/widgets/subscription_tier_badge.dart` | Done |
| UIP-006 | Keep chunky shadows visible | Added bottom layout clearance beneath subscription action buttons so their hard-offset shadows are not clipped. | `app/lib/subscriptions/pages/subscription_page.dart` | Done |
| UIP-007 | Localize async progress | Replaced the operation card during active work with a spinner in the purchase, restore, manage, assignment, or unassignment button that initiated it. The status card remains only when reconciliation times out and needs an explicit refresh. | `app/lib/subscriptions/pages/subscription_page.dart` | Done |
| UIP-008 | Clarify tier choices and management | Added distinct Plus and Business card treatments, localized monthly prices from each RevenueCat offering, concise tier copy, and consistent View details actions. Purchase-eligible tiers open their paywall; existing subscriptions open Customer Center. Manage subscription now uses the standard primary button. | `app/lib/subscriptions/models/subscription_page_model.dart`, `app/lib/subscriptions/pages/subscription_page.dart`, `app/lib/subscriptions/providers/subscription_page_model_provider.dart`, `app/lib/subscriptions/services/revenuecat_service.dart`, `app/lib/subscriptions/services/revenuecat_service_native.dart` | Done |
| UIP-009 | Remove redundant tier metadata and improve contrast | Inactive and available tiers now omit status, empty assignment, assignability, and period metadata. Active tiers use concise renewal or access-end copy and show assignment only when assigned. Added a dark-gold light-mode Business accent with automated 4.5:1 contrast coverage. | `app/lib/l10n/app_en.arb`, `app/lib/subscriptions/pages/subscription_page.dart`, `app/lib/theme/brand_colors.dart` | Done |

## Verification

- Commands run: focused widget suite, complete `test/subscriptions` suite, Dart MCP analysis, `git diff --check`, and a live-app hot restart after the model shape changed.
- Passing evidence: 53 focused widget tests passed; all 178 subscription tests passed; analysis and diff validation reported no errors; hot restart completed with no runtime errors.
- Skipped checks and reason: Full cross-platform builds were not repeated because this pass changes only Flutter presentation and widget composition.

## Scope Guardrails

- Requirement behavior changed: No.
- Business logic changed: Tier-card destinations now consistently route eligible purchases to paywalls and existing subscription states to Customer Center.
- APIs, migrations, permissions, or dependencies changed: No. The internal RevenueCat offering adapter and page presentation model now expose localized package prices.
- Notes: Billing eligibility, assignment candidates, confirmations, and mutation orchestration are unchanged. A failed offering price lookup does not discard a successfully loaded price for the other tier.

## Follow-ups

- [ ] None.
