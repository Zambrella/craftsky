# Implementation Review: Account-owned subscriptions

## Verdict

Status: Approved with notes
Reviewer: Codex with independent AppView, Flutter/native and Svelte reviews
Date: 2026-11-13
Risk level: High

## Summary

Reviewed the subscription stack against ADR 027 and the current recovery runbook.
All seven findings below are corrected with regression coverage. The review
covered provider response decoding, reconciliation, webhooks, catalog
mapping, checkout/portal recovery, migrations, account deletion, paid-feature
authorization, native identity and ownership, purchase/restore/management,
account/session fences, and web billing/proxy/UI state. No production mutations
or real purchases form part of this review. The user explicitly authorized
implementation of confirmed findings after this review.

## Findings

| ID | Severity | Area | Finding | References | Required action |
| --- | --- | --- | --- | --- | --- |
| IR-001 | Important (P1) | Provider state | HTTP 200 `null`, `{}`, or `items:null` is accepted as a complete empty RevenueCat snapshot. Missing access/payment booleans become false. Applying malformed provider state can revoke valid access and clear pins. | `appview/internal/integrations/revenuecat/client.go`, `fetchPage`, `subscriptionResource`, `ListCustomerSubscriptions`; ADR 027 authoritative reconciliation | Validate required consumed response fields before accepting a complete snapshot. Retain additive-field compatibility. Prove invalid responses preserve persisted access and pins. |
| IR-002 | Important (P1) | Native ownership | Bootstrap configures RevenueCat without an App User ID, before store ownership checks. Android's automatic pending-purchase sync can post an interrupted purchase under an anonymous ID, which a later login can alias to another DID. | `app/lib/subscriptions/services/revenuecat_service_native.dart`, `NativeRevenueCatConfigurator.configure`; `billing_account_setup_coordinator.dart`; ADR 027 forbids anonymous purchase associations | Defer SDK activation until a server-issued customer UUID is available, verify store ownership before activation/identity changes, and configure with that UUID explicitly. Preserve AppView access display when local store ownership prevents native operations. |
| IR-003 | Important (P2) | Flutter state | Purchase, restore and management completion invalidate only the page model. Its watched shared access provider can retain Free or paid state, leaving feature gates stale despite completed reconciliation. | `app/lib/subscriptions/pages/subscription_page.dart`, `_run`, `_refresh`; `subscription_page_model_provider.dart`; `subscription_access_provider.dart` | Refresh the account-scoped shared access provider after billing operations and reconciliation, with current-session fencing. |
| IR-004 | Important (P2) | Web state | Billing updates local access, while AppShell exposes a separate access value loaded only at boot. Reconciliation can show Plus while SPA scheduling, folders and pins remain locked. | `web-client/src/lib/features/settings/Billing.svelte`, `apply`; `web-client/src/lib/features/shell/AppShell.svelte`, `boot` | Publish verified account-scoped billing access to the shared shell state. Test purchase activation, access loss and wrong-account/late responses. |
| IR-005 | Important (P2) | Native lifecycle | Global native-operation state is started before asynchronous work, but terminal cleanup reads a widget Ref. Disposing the page during a slow dependency/provider operation can prevent cleanup and leave all billing controls blocked until restart. The correction review also found that delayed ownership/offering work could still open native UI after navigation. | `app/lib/subscriptions/pages/subscription_page.dart`, operation result handlers; `subscription_pending_operation_provider.dart`; purchase/restore/Customer Center coordinators | Make cleanup independent of widget lifetime and specific to its operation, retain confirmed pending reconciliation, and recheck page/session validity immediately before native effects after asynchronous ownership/offering work. Test leaving/reopening at these boundaries. |
| IR-006 | Important (P2) | Database capacity | WithPlusAccess holds a pooled connection for its session fence then calls SelfAccess through the pool. The access check waits for a second connection and stalls with a one-connection or saturated pool. | `appview/internal/subscriptions/feature_access.go`, `WithPlusAccess`; `store.go`, `SelfAccess` | Perform the fenced access read using the already-held connection. Test allowed/denied/error paths with one pool connection and preserve the access-loss fence. |
| IR-007 | Important (P1) | Native restart/payment | After a store payment and process death before RevenueCat indexing, in-memory pending state is lost and AppView can still have a fresh empty snapshot. Same-owner store receipts pass ownership checks, so a different tier opens a plain paywall instead of replacement. Google Play can create two subscriptions; asynchronous SDK sync and optional paywall product-change configuration do not guarantee protection. | `subscription_purchase_controller.dart`, `PaywallCoordinator.present`, native ownership channel; locked Android `PaywallViewModel.performPurchase` and `ProductChangeCalculator` | Require explicit native-store evidence that no current subscription exists before a new independent purchase. Keep ownership-verified replacement, restore and management paths available. Test AppView Free/store active after restart and retain fail-closed pending/unverified behavior. |

## Requirement and test traceability

- Current requirements: ADR 027 and `docs/operations/subscription-billing-recovery.md`.
- Older assignment-workflow acceptance documents are superseded where they
  conflict with per-DID ownership; no legacy product behavior is required.
- Existing tests cover identity/session isolation, reconciled access, provider
  pagination/authentication, durable Stripe attempts, suspension/deletion fences,
  environment isolation, native ownership/replacement and checkout navigation.
- The seven findings identify missing failure/lifecycle/state-propagation and
  native restart coverage. IR-007 was confirmed during the correction review
  before handoff; its fix remains within this user-authorized review task.

## Test evidence at review completion

- Real local PostgreSQL race suites passed for subscriptions, RevenueCat,
  Stripe, account deletion and Business before fixes.
- Web review: 100 focused tests passed for checkout, API/error messages and
  server auth.
- Overlay-only RevenueCat regressions failed for all four malformed response
  cases, confirming IR-001 without changing implementation during review.
- IR-002 was checked against the locked Android SDK 10.26 source and RevenueCat
  customer/restore documentation. Native device acceptance remains outstanding.

## Risk review

- Risk level: High, due to ownership and provider-authority boundaries.
- Confirmed fixes are local code changes; no deployment or billing activation is
  implied. Real Apple/Google and application-to-Stripe sandbox tests remain
  release gates. Provider-only test purchases do not establish application proof.
- Existing no-transfer ownership, Managed Payments requirement, durable checkout
  records and production enablement controls must be preserved.

## UI polish recommendation

- Recommendation: Not needed as a separate pass.
- These findings concern behavior and recovery, rather than cosmetic changes.

## Handoff and correction evidence

- IR-001: provider validation distinguishes absent/null fields from explicit
  false or an empty list. Malformed first or subsequent pages fail before
  persistent state changes. Documented null promotional products, omitted/null
  terminal cursors and additive fields remain accepted. An explicitly empty
  cursor fails rather than silently terminating pagination. Real PostgreSQL
  regressions preserve access, pins and confirmed reconciliation generations.
- IR-006: the fenced access query reuses the connection already holding the
  session lock. The new one-connection regression reproduced four deadline
  failures before the fix and passes for allowed/denied/effect-error/reuse paths
  after it. The existing concurrent access-loss/publication regression passes.
- IR-002: bootstrap returns a deferred service without configuring an anonymous
  SDK. Serialized identity dispatch verifies native store ownership and the
  current account before initial configuration with the exact AppView UUID or
  a later login. A store conflict leaves AppView subscription display usable.
- IR-003: operation completion invalidates the original session's shared access
  provider while preserving session fences. Real-page regressions keep the
  access provider alive as AppShell does and verify purchase/restore activation
  and management access loss without manual refresh.
- IR-004: Billing and AppShell share a reactive, account-scoped access state.
  DID/session validation, read revisions and disposal checks reject wrong-owner
  or stale responses. Real SPA scheduling-control regressions reproduced both
  stale-state directions before the fix and pass afterward.
- IR-005: native operation cleanup uses retained provider state instead of a
  disposed widget Ref; each callback carries an operation identity. Confirmed
  provider effects remain pending for reconciliation. Page/session checks
  propagate through SDK identity, ownership, offering and final-effect
  boundaries so leaving during preparation cannot dispatch native UI later.
  Regressions cover purchase, restore, Customer Center, delayed ownership and
  offerings, stale cleanup and navigation after a confirmed payment.
- IR-007: the final independent-purchase preflight requires an explicit native
  `hasActiveSubscription:false`. Android includes suspended subscription
  purchases in its query. StoreKit checks current entitlements and unfinished
  transactions while keeping
  full receipt-history ownership checks. Pending/unverified/missing evidence
  blocks purchase. Existing owned subscriptions produce a Refresh/Restore
  notice and request reconciliation without recording a new payment. Explicit
  replacement, restore and management keep ownership checks and remain usable.
  Regressions cover active/missing/malformed/foreign-owner channel results,
  replacement distinction and a fresh-container Free-server/active-store flow
  that recovers through Restore without a second purchase.
- Passed `./scripts/appview-test focused -p=1 -race
  ./internal/integrations/revenuecat` and the focused publication-fence checks.
- Web passed 104 focused Vitest tests, eight authenticated subscription
  Playwright fixture tests, Svelte check (zero errors/warnings), full lint,
  production build and formatting. The two new Playwright tests were observed
  failing against the pre-fix build and passing against the fixed build.
- Before IR-007, native focused checks passed 63 tests and the aggregate
  subscription/AppShell/settings/Business run passed 210 tests with 12 existing
  skips; analysis was clean. After IR-007, the full subscription suite passed
  175 tests with five existing skips, and full Flutter analysis reported no
  issues. The Android debug APK build passed, including the suspended-purchase
  query in the compiled ownership bridge. The iOS simulator build also passed
  (`flutter build ios --simulator --no-codesign --no-pub`); the Android command
  was `flutter build apk --debug --no-pub`. These compile the final Kotlin and
  Swift bridge changes; they do not establish store-device purchase behavior.
- Validation cleanup: the full backend race run found a five-second timeout in
  `TestScheduledImageReleaseConcurrentUploads`. Three isolated repeats also
  failed. A temporary diagnostic overlay with a 30-second wait and an early
  response check passed in 13.775 seconds, establishing slow instrumented decode
  rather than subscription authorization or early handler rejection. The probe
  verifies decoder admission and retained memory, not latency. Its wait is now
  30 seconds and early responses fail explicitly; runtime image limits are
  unchanged. Three race-detector repeats of the corrected probe passed
  (`./scripts/appview-test focused -p=1 -race ./internal/api -run
  '^TestScheduledImageReleaseConcurrentUploads$' -count=3`, 15.649 seconds).
  The broader run passed AppView/CLI, account deletion, configuration/wiring,
  authentication and Business. Its unrelated remainder was stopped during the
  chat package after the final subscription-specific gate passed. This is
  partial broad-suite evidence, not a clean full-repository run; a second full
  API-package run has not been performed.
- Final current-source race gate passed complete `internal/subscriptions`
  (61.434 seconds), `internal/integrations/revenuecat` (16.689 seconds) and
  `internal/integrations/stripe` (1.869 seconds), plus `internal/routes` filtered
  by `Subscription|Billing|RevenueCat` (3.940 seconds). Commands:

  ```sh
  ./scripts/appview-test focused -p=1 -race ./internal/subscriptions ./internal/integrations/revenuecat ./internal/integrations/stripe
  ./scripts/appview-test focused -p=1 -race ./internal/routes -run 'Subscription|Billing|RevenueCat'
  ```

- All seven required corrections are resolved. The approval is for this local
  implementation; native store-device/application sandbox validation and the
  normal release/deployment gates remain outstanding. No commits, deployment or
  provider configuration changes were made by this review pass.
- No running DTD/debug application was available for hot reload. These checks
  do not establish real StoreKit/Google Play or application-to-Stripe sandbox
  behavior, production migration success, deployment, or live activation.

## Primary references

- [RevenueCat subscription response schema](https://www.revenuecat.com/docs/api-v2/subscription-data-model)
- [RevenueCat list response format](https://www.revenuecat.com/docs/api-v2)
- [RevenueCat customer identity and aliases](https://www.revenuecat.com/docs/customers/identifying-customers)
- [RevenueCat restore behavior](https://www.revenuecat.com/docs/projects/restore-behavior)
- [Android SDK automatic sync initialization](https://github.com/RevenueCat/purchases-android/blob/10.26.0/purchases/src/main/kotlin/com/revenuecat/purchases/PurchasesOrchestrator.kt)
- [RevenueCat paywall product changes and parallel-subscription warning](https://www.revenuecat.com/docs/tools/paywalls/creating-paywalls/product-changes)
- [Google Play subscription query options](https://developer.android.com/reference/com/android/billingclient/api/QueryPurchasesParams.Builder)
