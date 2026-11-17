# Subscribed-state parity

15 November 2026. Mobile and web now retain the same plan-card structure when
subscribed: both cards show their monthly catalog price and benefits. The paid
plan has an Active pill and Manage action; the alternative has Upgrade or
Downgrade. The top summary shows Current access: Standard/Plus/Business and the
localized access-end date/time when present. No expiry is invented for Standard
or lifetime grants. Complimentary grants remain distinct from paid ownership.
The separate web current-subscription panel and native footer Manage action
were removed; unresolved payment and scheduled-change notices remain visible.

Existing subscribers go to the provider that owns their payment rather than a
second checkout. Native store purchases use RevenueCat Customer Center on mobile
and the store management page on web. Stripe purchases use an authenticated
customer-specific Stripe portal on both clients. Upgrade/Downgrade labels open
that management flow; no plan, payment timing or entitlement changes until the
provider confirms and AppView reconciles. Portal configuration controls which
plan changes are available. Real provider plan changes were not exercised.

The existing POST /v1/billing/web-management route now resolves SDK-created
purchases without an old direct-checkout mapping. AppView obtains a complete
subscription snapshot for the authenticated account's private RevenueCat UUID,
requires one current Stripe subscription in the configured environment, and
uses its store_subscription_identifier. Stripe verifies the actual merchant,
subscription ID, mode and customer before creating a portal session. Missing or
ambiguous identity fails closed. The response remains {url}; no new routes,
wire fields, migrations or lexicon changes are required. Mobile validates the
portal URL and rechecks account/page ownership before launching a late result.
Portal URLs remain ephemeral capabilities and are never logged or persisted.

Validation:

- 40 Flutter page/API tests passed: active Plus/Business cards, prices/benefits,
  expiry, Manage/Upgrade/Downgrade, native Customer Center lifecycle, Stripe
  navigation, leaving before portal response, malformed portal destinations,
  pull-to-refresh, promotions and large-text accessibility.
- 26 web subscription browser cases passed, including both plans at 360px and
  1280px, summary expiry, both portal actions, card prices/benefits and the
  unchanged new-purchase/paywall/reconciliation paths.
- Affected Go subscriptions, Stripe/RevenueCat integration and app dependency
  packages passed. Focused real-Postgres tests passed for SDK portal resolution
  without a legacy checkout and retained direct-checkout ownership. Unit tests
  reject wrong merchant, environment, subscription/customer IDs, incomplete and
  ambiguous snapshots and retain the RevenueCat store identifier.
- Web build/type/lint, targeted Dart analysis, docs checks and diff checks passed.
  Flutter hot reload succeeded. Local AppView was rebuilt and /health returned
  status: ok. No production deployment, provider configuration change or live
  purchase/plan change was performed.

Provider references: [RevenueCat Stripe management](https://www.revenuecat.com/docs/web/integrations/stripe),
[Customer Center](https://www.revenuecat.com/docs/tools/customer-center), and
[RevenueCat subscription data model](https://www.revenuecat.com/docs/api-v2/subscription-data-model).

## Renewal wording and mobile clipping follow-up

The top summary uses Renews on for a single matching paid tier with fresh,
unambiguous billing state and confirmed will_renew or will_change_product.
The next billing period end takes precedence over the entitlement's access end;
Access until remains for cancellation, unknown renewal and promotional-only
access. A paid lower tier never relabels a higher complimentary tier as renewing.
The mobile access card uses Clip.none to avoid rounding/clipping its padded
content; its paper border, rounded outline and shadow remain.

Follow-up validation: 19 Flutter page/accessibility tests passed, including all
four renewal states and clipping configuration. Eight web browser cases passed for the
same renewal states and retained active-card geometry/action coverage. Flutter
hot reload succeeded; web build and targeted static checks were run.

## Sandbox management diagnosis: subscription-item identifier

The actual authorized sandbox RevenueCat response supplies an si_ subscription
item as store_subscription_identifier, not the sub_ parent used by the initial
fixtures. The management resolver now accepts either and retrieves the Stripe
item's verified parent before the existing merchant/environment/customer checks.
Regression tests cover the observed item form, wrong item ID, absent/invalid
parent, wrong merchant, wrong environment and invalid customer. The focused
real-Postgres management and Stripe adapter tests passed; local AppView was
rebuilt. The stale client copy claiming management is unavailable “here yet”
was replaced with a temporary-unavailable/retry message.

The sandbox request passed the former 503 identity rejection after rebuilding,
but subsequently returned 500. End-to-end portal success is not yet verified.
Automatic review allowed the explicitly approved RevenueCat sandbox read but
blocked the subsequent Stripe sandbox read as a separate service destination.
Further Stripe response inspection and portal-session verification await that
specific approval; no payment or plan change has been made.

## Sandbox portal URL verification

With explicit sandbox inspection approval, Stripe confirmed the merchant,
subscription-item parent, customer and environment all matched. Portal creation
succeeded but returned `/p/session?secret=...`, rather than the older
`/p/session/{token}` fixture. Server, web and Flutter now allow both exact
customer-session forms. The new form requires one nonempty URL-safe secret and
rejects duplicate/extra query fields, fragments, foreign hosts and malformed
capabilities. The URL and secret were not persisted or printed.

The authenticated sandbox endpoint now returns 200. The real Manage action
opened “CraftSky sandbox Billing”, which displayed test mode, cancellation and
billing-detail controls. The portal's subscription_update.enabled remains false;
configuration inspection verified the available monthly Plus/Business prices
are GBP249/GBP899 minor units, but no portal setting or subscription was changed.
This supersedes the preceding blocked-inspection/500 status.

Checks passed: focused real-Postgres and Stripe adapter management tests,
15 web portal/navigation unit tests, 6 browser management/card cases, 17 error
copy tests and 5 Flutter API tests. Web build/type/lint and targeted Dart analysis
passed. Local AppView was rebuilt and Flutter hot reload succeeded.

## Sandbox and live portal plan switching enabled

15 November 2026. Following explicit approval to update both portals, enabled
price-only subscription switching for the two verified monthly products:
CraftSky Plus £2.49 and CraftSky Business £8.99. The sandbox portal was updated
through the configured Stripe API, with configuration snapshots kept outside
the repository. An expanded API read-back verified the exact product/price
allowlist, enabled flag and price-only updates. Other feature settings and all
existing subscription-update billing settings were compared and preserved.

CraftSky's live default portal was updated through the signed-in Dashboard.
Its saved configuration was reloaded in a fresh document and verified: plan
switching enabled, both monthly prices present, quantity updates disabled and
No charges or credits selected. Both modes retain no proration and the existing
billing cycle; new recurring prices begin at the next billing period. No
customer subscription, payment, price or product was modified.

The real sandbox customer portal showed Update subscription after reloading.
No plan-change confirmation was submitted. This supersedes the earlier notes
that subscription_update.enabled was false in sandbox/live. Production AppView
activation and end-to-end live purchases remain separate release work.
