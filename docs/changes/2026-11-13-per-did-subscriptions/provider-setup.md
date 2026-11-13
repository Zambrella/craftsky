# Per-DID subscriptions: provider setup

Date: 2026-11-13. This records live read-back during implementation, not a
successful deployment or end-to-end purchase test. Product direction is
[ADR 027](../../adr/027-per-did-subscriptions.md).

## Provider decision: Paddle rejected CraftSky

On 13 November 2026 the maintainer supplied Paddle's domain-review email for
craftsky.social. Paddle identified **Personal Websites/Social Networking**,
rejected the application under its Acceptable Use Policy, and explicitly said
the account cannot be activated for payments. This is a confirmed eligibility
blocker, not merely pending account verification. Do not enable Paddle checkout
or create further Paddle prices/integration credentials. Keep the per-DID
architecture; select and verify eligibility with another web billing provider
before enabling the integration. Stripe sandbox preparation and the replacement
application design are recorded below. No appeal or support message has been
sent.

The public [Paddle AUP](https://www.paddle.com/help/start/intro-to-paddle/what-am-i-not-allowed-to-sell-on-paddle)
mentions community-access restrictions and says the categories are not
exhaustive. Its public wording does not override the explicit domain rejection.

## Stripe sandbox setup (13 November 2026)

Stripe Managed Payments through RevenueCat's **Stripe Billing** integration is
the replacement being prepared. This is sandbox preparation, not live merchant
approval. RevenueCat Billing is a different integration and does not support
Managed Payments. RevenueCat recommends a Stripe Sandbox because legacy Test
Mode does not support Managed Payments; Stripe also recommends general
sandboxes for new integrations. See the [RevenueCat Stripe setup guide](https://www.revenuecat.com/docs/web/integrations/stripe)
and [Stripe Sandbox guidance](https://docs.stripe.com/sandboxes).

### Completed and verified

- Selected the CraftSky Stripe Sandbox. With the maintainer's explicit
  approvals, installed the RevenueCat Stripe App, linked the account and
  created a dedicated **Stripe Sandbox** configuration in RevenueCat.
- Enabled **Use Managed Payments when available** in that configuration.
- Created the maintainer-approved monthly, tax-inclusive sandbox products:
  **Plus, GBP 2.49**, using tax category `txcd_10103000` (SaaS, personal use),
  and **Business, GBP 8.99**, using `txcd_10103001` (SaaS, business use).
  Both categories appear in Stripe's [eligible tax code list](https://docs.stripe.com/payments/managed-payments/eligibility).
- Imported both products/prices into RevenueCat. Read-back returned
  price-specific `store_identifier` values; deployment configuration must use
  the actual RevenueCat product mappings rather than assume a Stripe product
  identifier is also its price or RevenueCat identifier. Attached Plus to
  **plus** and **paid**, and Business to **business** and **paid**.
- An eligible direct Stripe Checkout API session returned
  `managed_payments.enabled=true` and `automatic_tax.liability.type=stripe`.
  A request with an ineligible physical-goods tax category was rejected as
  ineligible for Managed Payments. No extra product was persisted; catalog
  read-back still contained exactly the intended two products. No payment was
  submitted during that initial probe. It proves those sandbox session-creation
  cases; the completed provider purchase is recorded below.
- Created the Stripe-to-RevenueCat sandbox webhook endpoint using API version
  `2025-03-31.basil` and the six documented events:
  `customer.subscription.updated`, `customer.subscription.deleted`,
  `charge.refunded`, `invoice.updated`, `customer.subscription.created`, and
  `checkout.session.completed`.
- Configured a sandbox Stripe Customer Portal with invoice history, payment and
  billing-detail updates, and cancellation at period end. Subscription tier
  changes and pausing are disabled. Saved its hosted login URL in RevenueCat's
  **Subscription management** tab and verified it after reloading. This static
  email-login URL is a historical sandbox setup artifact. Stripe's no-code
  portal can select the most recently active customer when several share an
  email, so it cannot select a CraftSky DID reliably. Replace/remove it before
  live activation; the application creates customer-specific portal sessions
  through authenticated AppView instead. The sandbox preparation also used the
  public homepage as a temporary return destination. See
  [Stripe's email-login behavior](https://docs.stripe.com/customer-management/activate-no-code-customer-portal).

### Connection configuration and provider purchases verified

The maintainer saved the webhook signing secret directly in RevenueCat.
Subsequent dashboard read-back confirmed a saved secret and the **View secret**
control, without revealing the secret. No password or signing secret was written
to repository files. External purchase tracking is now **enabled** with
`USE_METADATA_FIELD` and metadata key `app_user_id`. Save and subsequent
read-back passed. Invoice-paid recognition and the default rejection of
incomplete, unpaid/delinquent and paused imports remain enabled. The existing
Stripe webhook is **enabled**. Configuration save/read-back and the first
payment's delivery and entitlement processing have passed. Use the documented
[Stripe server notification setup](https://www.revenuecat.com/docs/platform-resources/server-notifications/stripe-server-notifications);
do not put signing secrets, account identifiers or credentials in this file.

A first synthetic Plus Checkout session was created directly through Stripe,
using a fresh artificial customer UUID with no real CraftSky DID. Its
`app_user_id` metadata was set on both the session and the subscription data.
Read-back confirmed Managed Payments enabled and Stripe tax liability. The
browser checkout was filled with Stripe's test card and synthetic contact
details. For a UK billing address it displayed **GBP 2.49 including GBP 0.42
VAT**, consistent with the configured tax-inclusive price.

Automatic approval review classified the final subscription submission as a
user-only recurring-subscription transaction even in the sandbox. The maintainer
completed that final click. Subsequent read-back confirmed:

- Stripe Checkout session **complete**, payment status **paid**, and subscription
  **active**. Managed Payments is enabled on both the Checkout session and its
  subscription, with `automatic_tax.liability.type=stripe`. The total was
  GBP 2.49, including GBP 0.42 VAT (249 and 42 minor units respectively).
- RevenueCat automatically created the exact synthetic UUID customer and its
  **sandbox** Stripe subscription: `gives_access=true`, `pending_payment=false`,
  with **plus** and **paid** entitlements and a current period ending
  **2026-12-13**. The dashboard's webhook status showed **Last received
  2026-11-13 3:06 p.m. UTC**.
- No manual receipt submission, entitlement grant or customer alias was used.
  This verifies the Stripe-to-RevenueCat provider flow for this synthetic
  customer. It does not verify any AppView DID mapping, application checkout,
  mobile access, renewal or production billing.

The second Plus purchase also completed using the same test card, email, name
and UK postal details, with a separate fresh artificial UUID. Stripe created
separate customer and subscription records. RevenueCat retained two distinct
original customer UUIDs, each with its own active sandbox subscription and
**plus** and **paid** access. This verifies same-plan purchases by independent
provider customers using the same payer details; real DID flows still need
application validation.

The Business purchase also completed with those same synthetic payer details
and a third fresh UUID. Stripe returned a complete, paid checkout and an active
sandbox subscription with Managed Payments enabled and Stripe tax liability.
Its GBP 8.99 total included GBP 1.50 VAT. RevenueCat automatically created the
correct third customer and active sandbox subscription with
`gives_access=true`, `pending_payment=false`, and **business** and **paid**
entitlements. All three Stripe subscriptions have separate customer records
and the expected `app_user_id` metadata, and all three remain active sandbox
test records. No live charges or actual CraftSky DID subscriptions were created.

The Stripe-to-RevenueCat sandbox connection, both tier mappings, and same-plan
multi-customer ownership with identical test payer details have now passed.
Cancellation, refund, renewal, failure handling and real application flows
remain separate checks.

## Stripe live preparation (13 November 2026)

After the maintainer completed Stripe account verification and copied the
sandbox configuration to live, live-mode read-back confirmed:

- Charges and payouts are enabled, account details are submitted, and no
  currently due or past-due verification requirements remain.
- Exactly two active monthly GBP prices exist: **Plus, GBP 2.49** and
  **Business, GBP 8.99**, both tax-inclusive, with the approved personal/business
  SaaS tax categories above. Stripe marks both products Managed Payments
  eligible. Managed Payments shows **Ready to use**, with the default enabled;
  Stripe says it becomes active after its first qualifying transaction. This
  is readiness evidence, not a completed live Managed Payments purchase.
- With the maintainer's explicit approvals for live account linking, app
  installation and legal terms, installed the RevenueCat app, linked the live
  CraftSky account and created a separate **CraftSky Stripe Live** configuration.
  The sandbox configuration and test purchases remain intact.
- Imported both live prices and attached Plus to **plus** and **paid**, and
  Business to **business** and **paid**. Read-back shows five products on each
  tier entitlement and ten on **paid**, including the prior native and sandbox
  products.
- Enabled Managed Payments and external purchase tracking in the live
  RevenueCat configuration. After saving and reloading, tracking reads
  `USE_METADATA_FIELD` with key `app_user_id`; paid-invoice recognition and
  rejection of incomplete, unpaid/delinquent and paused imports remain enabled.
  The project's **Keep with original App User ID** setting was retained.
- Created a live Customer Portal configuration with billing-detail and payment
  updates, invoice history and cancellation at period end. Tier changes and
  pausing are disabled. Its return destination is the web app's authenticated
  subscription settings. The email-login portal is disabled; AppView must create
  a customer-specific portal session for the authenticated DID.
- Created the live Stripe-to-RevenueCat webhook with API version
  `2025-03-31.basil` and the same six events listed above. After the maintainer
  completed RevenueCat password verification, saved the endpoint's signing secret
  directly in the live configuration. Reloading confirmed the **View secret**
  control and the expected tracking method, metadata key and endpoint. Then
  enabled the existing live Stripe webhook; a separate API read-back confirmed
  live mode, enabled status, the correct destination, API version and six events.
  No credential or signing secret was written to this repository. This verifies
  saved configuration, not a live event delivery or completed purchase.

The separate RevenueCat-to-AppView production webhook is also not configured:
the inspected project contains development receivers, including one without an
environment filter, rather than the production AppView destination. Before
accepting live purchases, configure and verify the authenticated production
receiver and restrict or retire development receivers so they do not receive
live billing events. Preserve any still-needed sandbox receiver deliberately.

No live customer, checkout purchase, subscription or payment was created. No
AppView runtime credentials, database migrations, runtime configuration or
application deployment were changed. Application activation still requires the live
RevenueCat app allowlist and **internal RevenueCat product IDs** mapped to tiers,
the server-only live Stripe key and approved price/portal settings, the separate
RevenueCat-to-AppView webhook credentials, migrations and the normal release
process. Keep these identifiers and credentials in the deployment configuration,
not this document. Real application sandbox acceptance remains a release gate.

## Application checkout design and release requirements

The replacement application path uses server-created Stripe Checkout and
Customer Portal sessions. This section describes that local implementation's
required behavior; the provider-only purchases above did not exercise it.
No production migration or deployment is implied.

RevenueCat's **Use Managed Payments when available** setting explicitly permits
ordinary Stripe Billing fallback for ineligible products or accounts. A
`pay.rev.cat` URL allowlist cannot enforce merchant-of-record responsibility.
Do not enable production hosted links as a substitute for checking each Stripe
session. See [RevenueCat's Managed Payments fallback documentation](https://www.revenuecat.com/docs/web/integrations/stripe/stripe-managed-payments).

The application and its release validation must preserve these boundaries:

1. Keep the authenticated AppView checkout endpoint and account ownership
   checks. Create Stripe Checkout sessions server-side with a configured tier's
   price, quantity one, fixed return URLs and explicit
   `managed_payments[enabled]=true`, following the [Stripe Checkout guide](https://docs.stripe.com/payments/managed-payments/set-up).
   Never retry a failed managed session as ordinary billing. Require both
   observed response fields above before exposing its checkout URL; missing or
   unexpected values must leave checkout unavailable.
2. Resolve the authenticated DID's private RevenueCat UUID server-side and put
   it in both Checkout Session `metadata.app_user_id` and
   `subscription_data.metadata.app_user_id`. Configure RevenueCat to read that
   key when automatically tracking external purchases. Keep paid-invoice
   recognition enabled. RevenueCat remains the entitlement/reconciliation
   source; a browser redirect must not grant access. See the
   [external purchase tracking guide](https://www.revenuecat.com/docs/web/integrations/stripe/track-external-purchases)
   and server notification identity rules linked above.
3. Persist a pending checkout and use provider idempotency under the DID's
   checkout fence. Retries must recover the same pending operation; concurrent
   Plus/Business requests and retained sessions must not create independent
   purchases while reconciliation is catching up. Observe provider expiration
   for an unused session, or a canceled Stripe subscription plus fresh
   non-blocking RevenueCat state after completed payment, before permitting
   another checkout. A browser cancellation redirect is not
   evidence that a retained Stripe session can no longer be paid.
4. Use authenticated `POST /v1/billing/web-management` for management. AppView
   resolves a verified Stripe customer from the DID's completed checkout mapping
   and returns a short-lived portal session; callers cannot supply another
   customer ID. Do not use shared payer email as an account selector. Missing
   mapping must leave management unavailable.
5. Use the actual RevenueCat app/product mappings, enforce account-bound client
   navigation for both Checkout and management responses, and preserve native
   cross-provider purchase protection and the absence of mobile web-purchase
   promotion. Configuration and recovery details are in the
   [billing runbook](../../operations/subscription-billing-recovery.md).
6. Honor the additive billing-state `webCheckoutPending` flag in native purchase
   controls before RevenueCat indexes the web payment. The flag reflects retained
   operations across environments even if web checkout is disabled. Web retries
   may resume the same tier's existing session through AppView.
7. Keep account deletion blocked for any unresolved checkout operation across
   environments. Reconciliation can resolve known provider sessions but must
   not create an unknown-response session while the account is pending deletion.

## Earlier provider and native preparation

- Before the Stripe setup above, RevenueCat's CraftSky project had App Store,
  Play Store and Test Store apps. There was no Paddle/web provider at inspection.
  The CLI runs via
  `npx @revenuecat/cli` but is not authenticated; the connected RevenueCat tools
  are authenticated and were used instead.
- Production restore behaviour is already **Keep with original App User ID**;
  no separate sandbox override was enabled. It was not changed.
- Created RevenueCat entitlement **paid**, attached to all six existing Plus and
  Business catalog products across the three apps, and read back all six links.
  Keep the separate Plus/Business entitlements for tier reporting. AppView's
  catalog and provider reconciliation remain the paid-access authority.
- At the initial native inspection, Apple had separate CraftSky Plus and CraftSky
  Business groups, both monthly products missing metadata. UK prices were GBP
  2.49 and GBP 8.99; US prices USD 2.99 and USD 9.99. The approved replacement
  catalog changes and verified results are recorded below.
- At the initial inspection, Play Console's subscription list was empty.
  RevenueCat's live store-state reads
  returned **not_found** for both registered Play products, rather than merely
  missing prices. Catalog registration alone is not Google Play provisioning.
- Paddle verification was submitted and the dashboard showed **In progress**
  before the rejection email was supplied. Prepared unpriced Plus and Business
  product entries. The maintainer explicitly approved archiving duplicate Plus
  product; it was archived and the active catalog was read back with exactly
  one Plus and one Business product. No
  Paddle prices, purchases, subscriptions or hosted checkout links were created.
- The older RevenueCat Apple metadata plan from 22 September was not applied.
  Its obsolete assignable-licence desired states were replaced on 13 November
  with the account-owned catalog preview described below. RevenueCat allows
  only one active plan per project and one store per plan.

## Native catalog transition and verified first batch

Apple does not offer moving an existing subscription between groups in the
inspected UI. Preserve existing provider history and customer identifiers:

1. Keep the current Plus monthly product and its existing prices.
2. Create a new Business monthly product in that same group (a new store product
   identifier is needed); copy the current Business prices and availability.
3. Rank Business above Plus in the group and give the group a name covering both
   tiers. Update purchase descriptions to refer to the purchasing account,
   removing the old assignable-licence wording.
4. Register the replacement Business product with RevenueCat, attach Business
   and paid entitlements, replace its offering package mapping, and add its
   RevenueCat product-to-tier mapping in AppView configuration.
5. Retain old Business history. Resolve any existing provider subscriptions
   explicitly before retiring the separate-group product from new sales.
6. Create the two missing monthly Google Play subscriptions/base plans with
   reviewed prices and territories, keeping the registered identifiers where
   possible. Configure replacement behaviour and test Plus/Business changes.

Do not restore or apply the old September desired states: their review text
describes the superseded architecture. The replacement previews are in
[native-store-preview.md](native-store-preview.md), with complete provider
diffs and warnings in [native-store-preview.json](native-store-preview.json).
The maintainer explicitly approved both previews, and both store plans reached
**applied**. The verified results and runtime mapping handoff are in
[native-store-results.md](native-store-results.md). Apple review submission is
separate and has not occurred.

The maintainer selected **all supported Google Play countries**. An empty Play
catalog does not expose country-price generation until products exist. The
Google creation batch first created monthly DRAFT base plans with the agreed
GB/US seeds. After reviewing the complete Google-generated price maps, the
maintainer approved worldwide pricing and activation. Both monthly plans now
read **ACTIVE**, with all 174 named supported regions and future-market
availability enabled. Independent reads exactly matched all approved prices,
including unchanged GB/US prices and the USD/EUR future-market defaults.
See the [full price list](google-worldwide-prices.csv) and
[verification results](google-worldwide-results.json). No purchase was submitted.
The Apple batch preserved all existing Plus prices, created the replacement
Business product in the Plus group with all 175 Business territory prices, and
updated account-owned wording, group display name, privacy URL and review notes.
After creation, Business-above-Plus order was saved and verified after reload.
RevenueCat Business and paid entitlements and the current Business offering now
use the replacement, while legacy history remains intact. No genuine Apple
review screenshot was supplied; the approved blank placeholders require
replacement before review submission.

## Historical Paddle setup plan — blocked, do not execute

The following plan is retained as implementation context only. Paddle rejected
CraftSky; these steps are not approved next actions for the current product.

1. Complete Paddle business verification and approve **pay.rev.cat** for hosted
   checkout. Creating an account or completing legally binding verification is
   a maintainer task.
2. Confirm the monthly web prices (proposed UK prices match native: GBP 2.49
   Plus, GBP 8.99 Business), tax-inclusive presentation and intended markets.
   Set quantity to one. Do not add a trial unless separately specified.
3. Connect Paddle to RevenueCat using the documented API-key permissions and
   automatic tracking. Do not paste credentials into chat or repository files.
4. Import Paddle prices into RevenueCat; attach each to its tier entitlement and
   the shared **paid** entitlement. Add the Paddle app and price-to-tier mappings
   to AppView's existing catalog configuration.
5. Create identified hosted purchase links for each tier. Configure returning
   customers with an existing **paid** entitlement to see the success page,
   including when opening the other tier's retained URL. Disable anonymous
   redemption. Return to the authenticated web subscriptions page.
6. Set AppView's `REVENUECAT_PADDLE_PLUS_CHECKOUT_URL` and
   `REVENUECAT_PADDLE_BUSINESS_CHECKOUT_URL` to the two base URLs. Do not include
   a customer suffix; AppView appends its own UUID. Both must be configured
   together. The server RevenueCat API v2 key also needs customer read/write
   permission for first-time web accounts.
7. Disable Paddle abandoned-cart recovery links for RevenueCat-presented
   checkout, as those links cannot reopen the RevenueCat checkout.

Web tier changes remain deferred. Existing customers cancel through Paddle's
receipt management link/customer portal, retain paid access until expiry, then
choose a new tier. The mobile apps contain no web purchase promotion.

## Release gates

- Apply migration 000124 only after its cross-DID/concurrent-subscription
  preconditions succeed against the intended database. No production migration
  was run during implementation.
- Provider-only sandbox purchase checks have passed for two independent Plus
  customers and one Business customer with the same test payer details,
  automatic event delivery and the expected entitlements. Application DID
  flows remain unverified.
- Validate the direct Stripe checkout path above, including rejection
  of missing/false Managed Payments or non-Stripe tax liability responses.
- Verify two separate DIDs can buy the same Stripe tier using the same payer
  details, without merging customers.
- Verify retained links and concurrent browser checkouts cannot create a second
  paid tier, including response-loss retries, provider-observed expiration and
  completed payments waiting for RevenueCat reconciliation. Apply migration
  `000125` only through the intended environment's normal migration flow.
- Verify pending web checkout blocks native purchases before RevenueCat indexing
  and blocks account deletion across environments, including disabled provider
  configuration and unknown create outcomes.
- Verify authenticated management selects the purchasing DID's Stripe customer
  when multiple customers share an email. Remove the historical static
  email-login management URL from provider configuration before live activation.
- Verify account switching, reinstall, wrong-owner restore, expired receipts,
  missing ownership tokens, sandbox interruptions and cross-store existing
  subscriptions on both native stores. Unknown ownership must fail closed.
- Verify native tier replacement charges and access transitions using the actual
  configured catalog. Separate Apple groups must not be treated as upgrades.
- Verify Stripe cancellation, expiry, failure, refund and webhook reconciliation.
  Neither a checkout success redirect nor a local fixture grants access.
- Release AppView and both clients using the documented release flow only after
  provider configuration and sandbox validation. Implementation does not deploy.
