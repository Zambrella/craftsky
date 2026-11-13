# Subscription identity and recovery

Subscriptions belong to one CraftSky DID. AppView stores one stable private
RevenueCat customer UUID for that DID. Each installation may cache UUIDs for
its signed-in DIDs, but AppView is the identity and paid-access authority.
There is no billing-holder selection or licence assignment. See
[ADR 027](../adr/027-per-did-subscriptions.md).

## New installation and account switching

Sign in with the original CraftSky account. `PUT /v1/billing/account` is
idempotent by DID and returns the original customer UUID. The client identifies
RevenueCat with that UUID, serializing SDK identity changes with purchase,
restore and management operations. An account switch invalidates delayed UI
results; it never reassigns provider transactions. Keep RevenueCat configured
with **Keep with original App User ID** for production and sandbox. Do not use
anonymous checkout, redemption links or receipt transfers.

The native preflight reads store transaction ownership before dispatching a
purchase or restore. A conflicting or unverifiable transaction prevents the
operation. This is a safety gate, not proof that a store account cannot change
after the query; test it with the real supported stores before release. Show an
account handle only when ownership has been securely verified. Generic receipt
errors do not identify another DID. Do not collect receipts, tokens or private
customer IDs in support messages.

## Web checkout

Use Stripe Managed Payments with RevenueCat's **Stripe Billing** integration.
Paddle rejected CraftSky and is not an available provider. Keep Stripe server
credentials out of both client build configurations.

Configure these AppView settings together, or leave all absent to disable web
checkout:

| Setting | Purpose |
| --- | --- |
| `STRIPE_SECRET_KEY` | Server-only Stripe key for the environment selected by `REVENUECAT_ENVIRONMENT` |
| `STRIPE_ACCOUNT_ID` | Expected direct Stripe merchant account, verified before session creation |
| `STRIPE_PLUS_PRICE_ID` | Approved recurring Plus price |
| `STRIPE_BUSINESS_PRICE_ID` | Approved recurring Business price |
| `STRIPE_CHECKOUT_SUCCESS_URL` | Fixed web destination after checkout completion |
| `STRIPE_CHECKOUT_CANCEL_URL` | Fixed web destination after leaving checkout |
| `STRIPE_BILLING_RETURN_URL` | Fixed web destination after leaving the Customer Portal |

`STRIPE_PORTAL_CONFIGURATION_ID` optionally selects an approved Customer Portal
configuration. Keep upgrades, downgrades and quantity changes disabled while web
tier changes remain deferred.

Add the RevenueCat Stripe app to `REVENUECAT_APP_IDS` and map its **internal
RevenueCat product IDs** (`prod...`) to that app and tier in
`REVENUECAT_PRODUCT_MAPPINGS`. AppView reconciles the v2 subscription's
`product_id`. Imported Stripe products use a Stripe price ID (`price_...`) as
their store identifier; that identifier belongs in the corresponding
`STRIPE_*_PRICE_ID` setting, not the RevenueCat product mapping. Use actual
provider read-back for both identifiers.
Attach each tier's entitlement and shared `paid` entitlement. Configure external
purchase tracking to read `app_user_id` metadata, recognize paid invoices and
reject incomplete, unpaid/delinquent and paused imports. The same UUID must be
present in Checkout Session metadata and subscription metadata. Disable anonymous
redemption and use the six documented Stripe events plus its endpoint signing
secret. The separate RevenueCat-to-AppView production webhook requires matching
authorization and signing configuration; inspect its environment filters and
restrict development receivers before accepting live events. See
[provider setup](../changes/2026-11-13-per-did-subscriptions/provider-setup.md).

AppView creates sessions with Managed Payments explicitly enabled, quantity one
and server-selected prices/return URLs. It returns a checkout URL only when the
session reports Managed Payments enabled and Stripe as tax-liability owner.
Never retry as ordinary Stripe Billing. RevenueCat's **Managed Payments when
available** toggle permits fallback and cannot enforce this requirement by
itself. The returned Stripe URL can contain a private fragment; preserve it for
navigation and exclude the entire URL from diagnostics and support messages.

Migration `000125` adds durable pending web-checkout operations. Work is fenced
per billing account and environment. Retries recover the existing session or
reuse its stored request and Stripe idempotency key after an uncertain response.
The migration's rollback refuses a nonempty checkout-operation table; recovery
records for potentially payable sessions must not disappear during rollback.
Checkout and portal dispatch serialize with suspension changes through a
moderation-standing lock bounded to 30 seconds; the pending operation commits
separately before provider I/O. Read-only reconciliation of a known session
remains available while suspended or awaiting deletion.
A different tier while an existing checkout remains payable returns
`web_checkout_pending`. Only a provider-observed expired session releases an
unused checkout. After completed payment, another checkout also requires
Stripe's subscription to be canceled and fresh RevenueCat state to establish
that the paid period no longer blocks checkout. The first canceled observation
queues and records a new reconciliation generation; that generation must finish
before cleanup. A recently completed older generation cannot satisfy this gate.
Neither local expiry time nor
a browser cancel redirect is sufficient.

`BillingState.webCheckoutPending` reflects any retained checkout operation for
the billing account, across sandbox and production. It remains true when Stripe
configuration is disabled. Native purchase controls use it to block a new store
purchase before RevenueCat has indexed the web payment. Web controls may resume
the existing same-tier checkout through AppView; do not clear the operation to
make the flag disappear.

Web plan changes are deferred. An account with an active, cancelled-but-current,
pending, ambiguous or stale subscription cannot start another checkout. Cancel
through Stripe's Customer Portal, retain access through expiry,
then choose the other plan. Native subscriptions are managed through their
original store. A success redirect does not grant paid access: reconcile with
RevenueCat and read AppView access. Mobile apps contain no web purchase links
or instructions.

### Pending or uncertain Stripe operation

Inspect the billing account's pending operation and Stripe session with
authorized private tooling. Preserve the original customer UUID and operation
identity. A retry with a known session queries Stripe; a lost create response
must reuse the original idempotency key and frozen request. Unknown-session
operations older than 23 hours remain blocked because Stripe's idempotency
retention is bounded. Do not delete the pending row, clear its operation ID or
switch tiers to force a new payment session. Find and resolve the original
provider result and reconcile RevenueCat before a reviewed recovery action.
The persisted attempt is pinned to `STRIPE_ACCOUNT_ID`; do not point an
unresolved attempt at a different merchant when rotating configuration.

Account deletion is blocked by **any** unresolved checkout operation, across
environments. Reconciliation's resolver may read a known provider session and
clear an operation only after its terminal outcome is established. It does not
create a remote session for an unknown-response operation while the account is
`deletion_pending`. An absent provider configuration, unavailable environment
or unknown session keeps the deletion gate closed; it is not evidence that no
payment can occur.

The current web deletion gate is conservative: a completed checkout remains
retained until Stripe reports its subscription canceled and fresh RevenueCat
state no longer shows a current paid period. A subscription canceled at period
end can therefore delay account deletion until that period ends. Do not remove
the checkout record to bypass this gate; validate this behavior in the
application sandbox before live activation.

Use fresh-state reconciliation after completed payment or cancellation. A
checkout return page only reports navigation; AppView's reconciled access is
authoritative. Keep pending state if Stripe or RevenueCat cannot establish a
terminal result.

### Customer-specific subscription management

`POST /v1/billing/web-management` returns `{url}` for a short-lived Stripe
Customer Portal session. AppView derives the Stripe customer from the
authenticated DID's persisted checkout mapping and verifies its completed
session. The browser cannot supply a customer ID. Without a verified mapping,
the operation returns `web_management_unavailable`; do not guess from an email
address or a different local account.

Do not use Stripe's static email-login portal as the per-DID management flow.
When multiple Stripe customers share an email, Stripe's no-code login selects
the most recent active customer rather than the authenticated CraftSky DID.
The static portal URL created during sandbox preparation is therefore a
historical setup artifact, not a live management destination. Remove it from
RevenueCat's management configuration before live activation unless replaced
with an authenticated account-specific flow. See [Stripe's portal login
behavior](https://docs.stripe.com/customer-management/activate-no-code-customer-portal).

The server fixes the portal return URL and optional portal configuration. Portal
URLs are capabilities: do not cache, persist or log them. Account switching must
discard delayed checkout and management responses.

## Triage and recovery boundaries

1. Distinguish an expired CraftSky sign-in (`401`) from missing billing identity.
2. Read the authenticated DID's AppView customer and compare its UUID with the
   correct RevenueCat project/environment using authorized support tools.
3. Check provider ownership, current/pending subscriptions, reconciliation
   generation and AppView effective tier. A different signed-in local account
   is not evidence that it owns the store receipt.
4. Retain request IDs in private support records. Never log checkout URLs,
   provider credentials, payment payloads or private customer UUIDs.
5. If server identity/history is missing or inconsistent, leave purchases
   blocked. Do not create a replacement customer, transfer a receipt or edit
   owner DIDs as an improvised fix. Verify ownership and inspect backup/provider
   history before proposing a reviewed recovery action.

The forward per-DID migration refuses cross-DID legacy allocations and
concurrent paid subscriptions rather than moving access silently. Resolve those
cases explicitly before deploying. Account deletion still requires provider
billing to be resolved; removing the app or a local DID session does not cancel
Apple, Google or Stripe billing. For database loss, use the
[production recovery runbook](production-render.md).
