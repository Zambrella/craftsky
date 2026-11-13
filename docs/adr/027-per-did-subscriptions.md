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
device recover the same UUID through authenticated AppView requests. Switching
accounts changes the SDK identity; it never transfers purchases or access.

Plus and Business are alternative tiers, with Business including Plus benefits.
Apple products belong in one subscription group, Business above Plus. Google
Play uses native replacement flows for tier changes. A store account cannot be
treated as multiple independent customers merely by switching DIDs. Restores
must retain the original identified customer; anonymous purchase/redemption and
automatic purchase transfers are not supported product flows.

Web checkout uses Stripe Managed Payments, with Stripe as merchant of record
and RevenueCat tracking the resulting Stripe Billing purchases.
The initial web release offers new Plus or Business subscriptions, cancellation
and existing-access display. Web tier changes are deferred. An existing current
or pending subscription blocks another web purchase, including a cancelled
subscription retaining access until its period ends. A subscriber can choose a
different web tier after expiry. Cross-provider replacement is not automatic.

The web client requests a Checkout session from AppView. AppView requires fresh
reconciled subscription state, selects a configured tier price, fixes quantity
at one and uses configured return destinations. The authenticated DID's private
RevenueCat UUID is placed in both session and subscription `app_user_id`
metadata. RevenueCat's automatic external purchase tracking reads that field
and recognizes paid invoices. Each tier retains its own entitlement plus the
shared `paid` entitlement. Checkout return/navigation is not evidence of access.

AppView creates every Stripe session with `managed_payments[enabled]=true` and
checks the returned session reports Managed Payments enabled and Stripe as tax
liability owner before returning the hosted checkout URL. Failure or missing
evidence leaves checkout unavailable; no retry may fall back to ordinary Stripe
Billing. RevenueCat-hosted purchase links are unsuitable for this requirement:
their "Managed Payments when available" setting explicitly permits fallback.

Persist a pending checkout operation before contacting Stripe, serialize work
per account and reuse its provider idempotency key for retries. A response lost
after Stripe accepted a request must not authorize a new independently payable
session. A completed payment remains blocked from another checkout while
RevenueCat catches up, and its Stripe subscription must be canceled before a
new non-blocking RevenueCat reconciliation requested after that cancellation
is observed can permit replacement. A recent empty snapshot from before the
payment or cancellation is insufficient. Only
provider-observed expiration may release an unused
session; a browser cancel redirect or local clock alone does not prove it is
unpayable. Retained URLs and concurrent tier requests must obey the same fence.
Attempts are also pinned to the verified Stripe merchant account so rotating
credentials or configuration cannot create a second payable checkout elsewhere.

Checkout and portal effects retain the owner lifecycle fence and take a shared
moderation-standing lock in a separate transaction bounded to 30 seconds. This
serializes dispatch with suspension changes. The pending operation commits
before that transaction, so cancellation or a failed commit cannot lose the
original provider idempotency key. Read-only reconciliation may inspect a known
session while the owner is suspended or awaiting deletion; it never creates an
unknown session in that recovery path.

Billing state includes `webCheckoutPending`, derived from retained checkout
operations across environments even when web checkout configuration is disabled.
Native purchases must respect this flag before RevenueCat has indexed a web
purchase. A web retry may still resume the same tier's existing session rather
than treat this flag as a reason to abandon its recovery operation.

Subscription management also uses an authenticated AppView operation. The server
resolves the purchasing DID's Stripe customer from its persisted checkout
mapping, verifies the completed session and creates a short-lived Customer
Portal session. Clients never choose a Stripe customer ID. A missing mapping
leaves management unavailable rather than guessing. A static email-login portal
is unsuitable: Stripe can choose the most recently active customer when several
customers share the payer email, which is a supported CraftSky use case.

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
- [RevenueCat Stripe Billing integration](https://www.revenuecat.com/docs/web/integrations/stripe)
- [RevenueCat Managed Payments fallback](https://www.revenuecat.com/docs/web/integrations/stripe/stripe-managed-payments)
- [RevenueCat Stripe identity and notifications](https://www.revenuecat.com/docs/platform-resources/server-notifications/stripe-server-notifications)
- [Stripe Managed Payments Checkout](https://docs.stripe.com/payments/managed-payments/set-up)
- [Stripe Customer Portal sessions](https://docs.stripe.com/customer-management/integrate-customer-portal)
- [Stripe no-code portal email behavior](https://docs.stripe.com/customer-management/activate-no-code-customer-portal)
- [Apple subscription groups](https://developer.apple.com/help/app-store-connect/manage-subscriptions/offer-auto-renewable-subscriptions/)
