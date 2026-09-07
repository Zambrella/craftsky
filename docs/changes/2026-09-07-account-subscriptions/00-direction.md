# Direction: Account-Assigned Pro And Business Subscriptions

Status: Chosen direction before detailed discovery and requirements

Date: 2026-09-07

## 1. Purpose

CraftSky will offer paid benefits for individual CraftSky accounts while
continuing to support multiple signed-in accounts on one installation. A member
must be able to keep free, Pro, and Business identities on the same device
without paid access following the device or whichever account is currently
active.

This document records the selected product and billing direction. It is not the
complete requirements, store configuration, API design, schema design, or
implementation plan.

## 2. Product Principles

- The free version remains a complete social experience.
- Paid access adds convenience, customisation, publishing tools, higher service
  limits, and business tools.
- Payment never increases distribution, ranking, feed placement, search
  placement, or visibility.
- Paid access applies to an explicitly assigned CraftSky account, identified by
  DID, rather than to a device.
- Free accounts do not consume a paid license.
- Subscription and account-assignment data is private AppView data and is not
  written to a PDS.

The current draft benefit direction is maintained:

| Tier | Draft benefits |
|---|---|
| Pro | Scheduled posts, saved drafts, saved-post folders, enhanced profile customisation, additional pinned posts, and higher-quality or longer video uploads. |
| Business | Pro benefits plus follower growth metrics, featured products, a profile call to action, business information, and upcoming events. |

The exact benefits, limits, names, prices, and launch sequencing remain subject
to later requirements work.

## 3. Chosen Commercial Shape

At launch, one payer may hold two independent recurring subscriptions:

| Subscription | Allowance |
|---|---|
| Pro account license | Grants Pro to one assigned DID. |
| Business account license | Grants Business, including all Pro benefits, to one assigned DID. |

A payer may hold neither subscription, either subscription, or both
subscriptions at the same time. When both are active, they must be assignable to
different DIDs. The intended supported example is:

```text
Personal account       Free
Creator account        Pro
Shop account           Business
Other accounts         Free
```

The initial native-store model deliberately limits a payer to one Pro account
license and one Business account license. It does not attempt to sell the same
native-store subscription repeatedly for an arbitrary number of DIDs.

CraftSky may support any number of free accounts at the service level. The
existing Flutter installation limit of five retained signed-in accounts remains
a separate product constraint and is not changed by this direction.

## 4. Billing Identity And Account Assignment

The RevenueCat customer will represent a private CraftSky billing account, not
the active DID and not the installation.

```text
CraftSky billing account
RevenueCat App User ID: private stable UUID

Pro subscription
  -> Pro license
  -> assigned DID A

Business subscription
  -> Business license
  -> assigned DID B
```

The billing account has an authenticated owner DID. A private, stable UUID is
used as its RevenueCat App User ID. The UUID must not change when the owner
switches active CraftSky accounts or devices.

AppView maps each active subscription to a license and maps each license to at
most one DID. AppView derives a DID's effective tier from its active license
assignment. Flutter must not grant account access directly from the billing
account's combined RevenueCat `CustomerInfo`, because that object can contain
both subscriptions without expressing which DID receives each one.

Assigning a license creates an intentional private relationship between the
billing account and the target DID. CraftSky must not infer this relationship
from accounts retained on a device. The target account must be authenticated or
must explicitly accept an assignment invitation. The launch experience may be
limited to assigning accounts currently authenticated on the same installation;
remote invitations are deferred.

## 5. Store And RevenueCat Direction

### Apple App Store

Pro and Business must be separate subscription groups if one Apple Account is
to hold both subscriptions concurrently. Monthly and any future annual product
for the same account license belong in that license's group.

```text
CraftSky Pro group
  - Pro Monthly
  - Pro Annual (future)

CraftSky Business group
  - Business Monthly
  - Business Annual (future)
```

Apple recommends one group when products are ordinary tiers of one service, but
separate groups are required when a customer may legitimately buy both. Store
copy and in-app explanations must therefore present these as two independently
assignable account licenses, not merely two mutually exclusive tiers for one
account.

Both Apple purchases appear under the same Apple Account and receipt. They must
therefore remain under one RevenueCat billing customer. CraftSky must not try to
associate the Pro transaction with one DID-shaped RevenueCat customer and the
Business transaction with another.

### Google Play

Pro and Business are separate subscription products so one Google account may
hold both. Monthly and future annual billing choices are base plans for the
corresponding subscription product where supported by the final catalog design.

### RevenueCat

RevenueCat tracks the billing customer, validates purchases, and supplies
subscription lifecycle events. AppView tracks license identity, assignment, and
effective account access.

The likely RevenueCat entitlement identifiers are conceptually:

```text
pro_subscription
business_subscription
```

These entitlements indicate whether the billing customer has each kind of
access. They do not represent license quantity or DID assignment. Product and
entitlement identifiers will be finalized during catalog design.

The RevenueCat project should use a restore behavior that does not silently move
the payer's purchases to an unrelated CraftSky billing identity. Exact restore,
account recovery, and support behavior must be specified and tested before
release.

## 6. AppView Authority

AppView is authoritative for paid feature access. This is necessary because
many paid capabilities are server-backed, including scheduled publishing,
service limits, follower metrics, and business surfaces.

The durable model should represent licenses as rows rather than fixed
`pro_did`/`business_did` columns. Conceptually:

```text
billing_accounts
  id
  revenuecat_app_user_id
  owner_did

billing_licenses
  id
  billing_account_id
  provider
  provider_subscription_id
  tier
  status
  gives_access
  current_period_ends_at
  assigned_did
```

The final schema may separate provider subscriptions from assignable licenses,
but it must preserve these concepts:

- Every provider subscription has a stable provider identifier.
- Every active subscription produces a distinct license.
- Every license has one tier and at most one assigned DID.
- Feature authorization uses the authenticated request DID and authoritative
  AppView state.
- Cancellation normally retains access until the paid period expires.
- Billing retry and grace-period access follows the normalized RevenueCat
  subscription state.
- Removing access does not delete a member's PDS records or private retained
  data.

RevenueCat webhooks should drive timely state changes, with idempotent event
handling and periodic API reconciliation. Detailed webhook security, ordering,
replay, and failure recovery belong in later requirements.

## 7. Account And Subscription Experience

The account switcher and account-facing settings should display the effective
tier for each retained DID. A billing management surface should separately show
the payer's subscriptions and their assignments.

Conceptually:

```text
Your subscriptions

Pro
Assigned to: @creator.example
[Change account] [Manage subscription]

Business
Assigned to: @shop.example
[Change account] [Manage subscription]
```

Only an authorized billing-account owner may purchase, restore, cancel through
an available provider flow, or change assignments. An assigned account that is
not the billing owner should be able to see that its access is supplied by a
billing account without receiving payment details or authority to manage the
subscription.

CraftSky should prevent accidental double payment by default. In particular, a
single DID should not ordinarily receive both the Pro and Business licenses,
because Business includes Pro.

## 8. Future Web Expansion

The row-based license model is intentionally designed to support additional
web-purchased subscriptions later.

```text
Native stores initially
  - at most one Pro license
  - at most one Business license

Web later
  - additional Pro licenses
  - additional Business licenses
```

Each successful web subscription becomes another `billing_licenses` entry with
its own provider subscription identifier and can be assigned to another DID.
The account authorization path does not change: AppView still resolves the
active DID to an active assigned license.

RevenueCat entitlements are boolean and cannot by themselves express that one
billing customer owns several licenses of the same tier. Future web support must
therefore consume individual subscription objects and lifecycle events rather
than count active entitlement keys.

RevenueCat Billing or Stripe integrated with RevenueCat are candidate web
providers. Exact support for repeated purchases, checkout identity, taxes,
invoices, customer portal behavior, and regional mobile-store linking rules must
be validated when web billing is designed.

## 9. Important Tradeoffs

- Separate Apple subscription groups permit simultaneous Pro and Business
  ownership but do not provide a native Pro-to-Business replacement path.
- Moving one DID from Pro to Business may require purchasing Business and
  separately cancelling or reassigning Pro, with possible overlapping billing
  periods.
- Apple and Google subscription-management screens cannot show which CraftSky
  DID holds a license; CraftSky must provide that context.
- A private billing account introduces an explicit server-side relationship
  between its owner and assigned DIDs. This is narrower than, and must remain
  separate from, the device-local multi-account session registry.
- License reassignment needs abuse controls so one active license cannot be
  rapidly shared across many accounts.
- Account deletion must define what happens when the deleted DID owns a billing
  account or holds an assigned license.

## 10. Relationship To Existing Decisions

- Multi-account sessions remain device-local, with one active account at a time.
  Billing relationships are not inferred from that registry and do not merge
  sessions or account data.
- OAuth access tokens, refresh tokens, and DPoP keys remain server-side. Billing
  does not expand the existing token boundaries.
- Paid state is private-by-intent and belongs in Postgres, not in a public PDS
  record.
- Record writes still go through the PDS and reads still come from AppView.
- Paid access does not alter chronological feeds, ranking, moderation, or public
  data ownership.
- Existing business-profile design treats business classification as a
  self-declared account type that does not imply payment or verification. Later
  subscription requirements must explicitly reconcile that classification with
  which business tools require a paid Business license; this direction does not
  silently redefine the existing account type.

## 11. Rejected Directions

### RevenueCat Customer Per Active DID

Rejected because an Apple receipt belongs to the Apple Account and includes its
purchases. RevenueCat cannot reliably split a Pro purchase and a Business
purchase from the same receipt between separate DID customers.

### Subscription Attached To The Installation

Rejected because access would follow a device rather than a CraftSky account,
would not restore cleanly across devices, and would make account switching
ambiguous.

### Unlimited Native-Store Subscription Per DID

Rejected as the launch foundation because Apple and Google do not support
repeated concurrent purchases of one subscription product in the required
general form. Duplicating store products or subscription groups as numbered
slots would create a fragile catalog and poor management experience.

### Fixed Multi-Account Bundles

Deferred rather than permanently rejected. Bundles such as three Pro accounts
or one Business plus five Pro accounts remain possible later, but the selected
launch direction lets customers buy only the two account licenses they need and
keeps allowances simple.

## 12. Deferred Questions

- Final public names: `Pro`, `Plus`, `Business`, or another naming scheme.
- Final benefits, prices, billing periods, trials, introductory offers, and
  regional availability.
- Whether Pro and Business launch together.
- Whether the subscription owner must be assigned one of the licenses.
- Whether assignments initially require both accounts on one installation.
- Whether reassignment is immediate, rate-limited, or limited per billing
  period.
- How account recovery proves control of an existing billing account.
- How billing ownership transfers when the owner DID is deleted or loses access.
- Exact upgrade, downgrade, cancellation, grace-period, refund, and restore UX.
- How existing unrestricted implementations of proposed paid features transition
  to entitlement checks.
- Which business-profile capabilities remain free account classification and
  which require the paid Business license.
- Whether historical paid data becomes read-only, hidden, or editable after
  access expires.
- Whether Family Sharing is disabled for the native subscriptions.
- Web checkout provider, repeated-product behavior, tax handling, customer
  portal design, and mobile storefront compliance.
- Whether the five-account local session limit should change independently.

## 13. Decision Summary

- Use a private CraftSky billing account as the RevenueCat customer.
- Give the billing account a stable private UUID independent of all DIDs and
  devices.
- Offer one independently assignable Pro account subscription and one
  independently assignable Business account subscription through native stores.
- Allow the same payer to own both subscriptions concurrently and assign them to
  different DIDs.
- Keep free accounts independent and unlicensed.
- Make AppView authoritative for subscription lifecycle, license assignment, and
  feature access.
- Model licenses as rows with provider subscription identifiers so additional
  web-purchased Pro and Business licenses can be added later without redesigning
  account authorization.
- Preserve CraftSky's no-paid-reach principle and existing atproto security and
  data-placement boundaries.
