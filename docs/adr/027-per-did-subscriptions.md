# ADR 027: Account-owned subscriptions and Stripe web checkout

Date: 2026-11-13
Status: Accepted by the maintainer for implementation
Supersedes: ADR 014's payer, assignable-licence and separate native tier-group decisions

## Provider eligibility update (13 November 2026)

Paddle rejected craftsky.social after domain review, classifying it as Personal
Websites/Social Networking. The maintainer selected Stripe Managed Payments
through RevenueCat's Stripe Billing integration as the replacement. Three
provider-only sandbox purchases verified both tier mappings and independent
customers using the same payer details. They do not prove production merchant
eligibility or application checkout behavior. Live activation remains a separate
release gate. See [provider setup](../changes/2026-11-13-per-did-subscriptions/provider-setup.md).

## Context

CraftSky supports several independent signed-in DIDs. Supporting separate native
Plus and Business purchases for different DIDs under one store account led to a
billing-holder identity, licence assignment and reassignment controls. The
maintainer now accepts native-store purchase limits and chooses independent web
subscriptions for additional DIDs. There are no active app users requiring
compatibility with the old product behaviour, but existing production billing
records and subscriptions still require explicit migration controls.

## Decision

Each DID owns its own subscription and is a separate RevenueCat customer. AppView
maintains a stable random UUID for that customer, rather than using the public
DID as RevenueCat's externally queryable identifier. Reinstallation and another
device recover the same UUID through authenticated AppView requests. Activating
an account synchronizes SDK identity when ownership checks allow; it never
transfers purchases or access.

When the subscription integration is enabled, eligible accounts that have
completed onboarding synchronize billing identity when they become active:
after sign-in, account switching and session restoration on launch. This is
optional background work outside account-critical initialization. AppView
creates or recovers the DID's UUID and durably queues initial RevenueCat
customer registration through reconciliation. The browser uses this server
registration path outside checkout. Opening a web paywall configures the Web SDK
with that same identified UUID. Native clients also attempt SDK
configuration/login through the serialized ownership guard. Store conflicts or
provider failures must not prevent sign-in or account switching; native billing
continues to require the exact current identity before any operation. Guests
remain unconfigured, and clients never call SDK logout to create an anonymous
customer between identified accounts. Integration enablement remains explicit.

Plus and Business are alternative tiers, with Business including Plus benefits.
RevenueCat promotional grants are a separate source of access, not a paid
subscription. AppView recognizes the configured project's `plus` and `business`
entitlements on verified `promotional` subscription snapshots, which need no
product ID. Grants belong to the same customer-owner DID and never create an
assignable licence. Effective access is the highest valid paid or promotional
tier; Business includes Plus. Grant expiry or revocation must not remove access
provided independently by another grant or paid subscription.

An active promotion does not block buying a paid subscription or count as a
duplicate paid purchase. Both clients display complimentary access and its
expiry separately from actual paid billing. Buying during a promotion uses the
ordinary store or Stripe purchase flow: paid billing starts immediately, and
unused complimentary time is not added to the paid term. It does not revoke the
independent promotion. Scheduling paid billing to begin at grant expiry is not
part of this flow. Actual paid subscriptions and pending payment operations
retain the existing duplicate-purchase protections and management rules.

Apple products belong in one subscription group, Business above Plus. Google
Play uses native replacement flows for tier changes. A store account cannot be
treated as multiple independent customers merely by switching DIDs. Restores
must retain the original identified customer; anonymous purchase/redemption and
automatic purchase transfers are not supported product flows.

### Shared RevenueCat paywalls (15 November 2026)

The maintainer accepts RevenueCat's Stripe Billing fallback when Managed Payments
is unavailable. This supersedes the earlier requirement to reject all non-MoR
transactions. Enable **Use Managed Payments when available** in the RevenueCat
Stripe configuration: eligible purchases use Stripe as merchant of record;
ordinary Stripe Billing may proceed otherwise, with CraftSky retaining merchant
and tax responsibility for those transactions.

The Svelte client uses `@revenuecat/purchases-js` and the same published `plus`
and `business` offering paywalls as Flutter. Each offering's `$rc_monthly` package
contains the matching native, Test Store and Stripe products. Runtime selection
comes from the environment-specific Stripe app SDK key; never use the mobile
Test Store key on web. The SDK renders the paywall in a bottom sheet and switches to a larger
modal for Stripe payment. Cancelling provider checkout restores the sheet. Provider authentication may still redirect out of the modal.

`POST /v1/billing/web-paywall` accepts only `tier`, derives the private customer
UUID from the authenticated DID and checks fresh reconciled billing state under
the owner lifecycle fence. It returns the environment-matched public SDK key,
private App User ID and offering with `Cache-Control: no-store`. The client
repeats preflight before every SDK purchase, including retries, and discards
operations after switching accounts or disposing the view. A browser account
lock prevents concurrent paywalls in cooperating tabs. No anonymous customer,
redemption or identity transfer is used. SDK purchase results trigger AppView
reconciliation; they never grant feature access directly.

Existing current, pending or anomalous paid subscriptions block a new purchase;
complimentary grants do not. Web tier changes remain deferred until expiry.
RevenueCat owns new SDK Checkout creation and its retries. AppView does not
persist the SDK's Checkout Session in `billing_web_checkouts`, so the older
server-created session idempotency, cancellation and pending-operation fences
do not describe this SDK flow. A tab lock is not a cross-device payment lock.
RevenueCat duplicate-purchase protection and server reconciliation remain
necessary; validate uncertain payments and cross-device races in sandbox before
production activation.

The direct Stripe checkout endpoints remain for recovery of already-created
operations. Those operations retain their original Managed Payments validation,
merchant binding, durable idempotency and provider-confirmed expiration rules.
`webCheckoutPending` continues to block native purchases and new web paywalls
while such an operation exists, across environments. **Cancel pending purchase**
expires only the authenticated owner's verified open session; closing a
RevenueCat modal disposes its UI without calling that recovery endpoint.

Management still uses authenticated AppView resolution of a retained direct
checkout's Stripe customer. SDK-created purchases without that mapping use the
existing receipt-management fallback. Do not substitute a static email-login
portal: multiple CraftSky customers may share a payer email. New customer-specific
portal support needs verified provider ownership evidence, not a caller-selected
Stripe customer or an unverified configured management URL.

Mobile apps contain no web purchase promotion, instructions or checkout links.
A native conflict explains that the store subscription belongs to another
CraftSky account. A DID/handle is shown only with verified ownership evidence;
the app must not guess from local accounts or disclose arbitrary customers.
Receipt associations can survive expiry, so conflict copy says "linked" rather
than always claiming an active subscription. Native preflight/account-switch
behaviour requires real store sandbox validation before enabling purchases.

## Implementation and data boundaries

AppView remains authoritative for paid features, including scheduling and
business access. Provider events, reconciliation, refunds, expiry and deletion
remain server workflows. Billing stays private in Postgres; no Lexicon/PDS data
changes are needed. Internal billing/customer and licence projection rows may
remain as implementation details, but a projection can grant access only to its
customer's owner DID and cannot be reassigned.

Remove cross-DID assignment routes and client controls. Preserve customer UUIDs
and subscription history. Forward migrations must refuse ambiguous historical
cross-DID allocations rather than silently moving benefits or provider records.
Deletion still requires provider billing to be resolved and does not cancel a
provider subscription simply by removing local state.
Promotional grants carry no recurring payment obligation and do not block
deletion. Their local access is disabled when the billing account closes.
Any unresolved web-checkout operation, including one from another environment,
blocks account deletion. Reconciliation may resolve known sessions only after
observing provider terminal state; it must not create an unknown session while
the account is pending deletion.

## Consequences

- Removes billing-holder selection, user-controlled licence assignment and
  reassignment cooldowns from the product.
- Keeps ordinary authenticated account-session fences and serializes native
  identity changes with purchase/restore operations.
- Retains provider lifecycle complexity; a checkout UI success cannot grant
  privileges, and an unknown/stale provider state cannot authorize checkout.
- Stripe live Managed Payments eligibility, products, prices, RevenueCat
  tracking, webhook configuration and application purchase validation are
  separate release gates. Sandbox configuration is not production approval.
- The existing separate Apple groups need a reviewed store-catalog transition;
  code changes cannot move an existing product between groups automatically.
- No production database migration, deployment or app release is implied by
  completing local implementation.

## References

- [RevenueCat customer identity](https://www.revenuecat.com/docs/customers/identifying-customers)
- [RevenueCat restore behaviour](https://www.revenuecat.com/docs/projects/restore-behavior)
- [RevenueCat promotional grants](https://www.revenuecat.com/docs/dashboard-and-metrics/customer-profile#granting-promotional-entitlements)
- [RevenueCat subscription schema](https://www.revenuecat.com/docs/api-v2/subscription-data-model)
- [RevenueCat Stripe Billing integration](https://www.revenuecat.com/docs/web/integrations/stripe)
- [RevenueCat Managed Payments fallback](https://www.revenuecat.com/docs/web/integrations/stripe/stripe-managed-payments)
- [RevenueCat Stripe identity and notifications](https://www.revenuecat.com/docs/platform-resources/server-notifications/stripe-server-notifications)
- [Stripe Managed Payments Checkout](https://docs.stripe.com/payments/managed-payments/set-up)
- [Stripe Customer Portal sessions](https://docs.stripe.com/customer-management/integrate-customer-portal)
- [Stripe no-code portal email behavior](https://docs.stripe.com/customer-management/activate-no-code-customer-portal)
- [Apple subscription groups](https://developer.apple.com/help/app-store-connect/manage-subscriptions/offer-auto-renewable-subscriptions/)
