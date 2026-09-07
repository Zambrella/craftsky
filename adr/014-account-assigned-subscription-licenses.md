## Architecture Decision Record

- Status: Accepted
- Aspect: Subscription identity, multi-account billing, paid access authority
- Date: 2026-09-07
- Decision: Represent native-store subscriptions as independently assignable account licenses owned by a private CraftSky billing account

### Why I needed to decide this

CraftSky intends to offer Pro and Business benefits while allowing one
installation to retain several independent CraftSky accounts. A member may use a
free personal account, a Pro creator account, and a Business shop account on the
same device. Paid access must follow the intended CraftSky DID across devices;
it must not follow the installation or whichever local account is currently
active.

Apple and Google own the payment relationship through the device's store
account. RevenueCat can associate purchases with a custom App User ID, but it
cannot make one store account behave like several independent store accounts.
In particular, Apple returns the Apple Account's purchases together on its
receipt, and one Apple Account can hold only one subscription in a subscription
group at a time. Google likewise does not provide a general way to buy the same
subscription product repeatedly for different in-app accounts.

This creates two separate identities that the architecture must not conflate:

- the payer whose store account or web payment method owns subscriptions; and
- the CraftSky DID that receives the product benefits.

The decision also needs to preserve a path to selling additional Pro or Business
licenses through the web without replacing account authorization later.

### Options I considered

**Option 1: Make each DID a separate RevenueCat customer - not chosen**

Configure RevenueCat with the active DID, or a UUID mapped one-to-one to that DID,
and attach every purchase directly to that account.

Pros:

- Gives each DID an apparently simple subscription status.
- Makes client-side RevenueCat entitlement checks straightforward for the active
  account.
- Works well when one store account funds only one paid CraftSky account.

Cons:

- Does not reliably support a Pro purchase for one DID and a Business purchase
  for another DID under the same Apple Account and receipt.
- Restore behavior can reject the second account, move purchases between
  accounts, or associate a later purchase with the receipt's existing owner.
- Treats RevenueCat identity switching as if it changed store ownership when it
  does not.
- Creates avoidable purchase and support failure modes after a user has already
  been charged.

**Option 2: Attach paid access to the installation - not chosen**

Treat a successful purchase as unlocking paid features for accounts retained on
that device.

Pros:

- Requires little account-assignment UI.
- Matches the local availability of a store receipt.

Cons:

- Paid access would not reliably follow a DID to another device.
- Reinstallation, device replacement, and concurrent devices become ambiguous.
- Every account on a shared device could inherit benefits accidentally.
- Conflicts with CraftSky's account-scoped sessions and server-authorized paid
  capabilities.

**Option 3: Sell a fixed multi-account bundle - not chosen for launch**

Sell products such as a Pro bundle with three Pro assignments or a Business
bundle with one Business assignment and five Pro assignments.

Pros:

- One store transaction can fund several DIDs without splitting a receipt.
- A single mutually exclusive plan group can provide a conventional bundle
  upgrade and downgrade path.
- Fits members who predictably operate several accounts.

Cons:

- Forces customers to pay for a predefined allowance rather than the one or two
  licenses they need.
- Requires product decisions about seat counts, mixed tiers, unused capacity,
  and downgrade selection before demand is known.
- A six-account bundle exceeds the current five-account local session limit if
  all assigned accounts are expected to remain signed in on one installation.
- Still requires a billing owner and private DID assignments, so it does not
  remove the core architecture work.

This remains a compatible future product because the chosen design represents
licenses as rows rather than fixed tier columns.

**Option 4: Duplicate native-store products or groups as numbered slots - not chosen**

Create several copies of each subscription and allocate the next unused product
or Apple subscription group whenever a payer adds an account.

Pros:

- Could approximate several simultaneous native-store licenses under one store
  account.

Cons:

- Produces a finite, duplicated, and difficult-to-evolve store catalog.
- Exposes confusing duplicate subscriptions in Apple and Google management UI.
- Requires fragile slot-selection and restore logic.
- Does not provide a credible path to arbitrary quantities.
- Increases the chance of charging a user without assigning the expected access.

**Option 5: Use one billing customer with one Pro and one Business account license - chosen**

Create a private CraftSky billing account for the payer. Use its stable UUID as
the RevenueCat App User ID. A Pro subscription produces one Pro license and a
Business subscription produces one Business license. AppView assigns each
license to a DID and authorizes features from that assignment.

Pros:

- Supports one free, one Pro, and one Business account on the same installation
  under one Apple or Google payer identity.
- Keeps both Apple transactions under the one RevenueCat customer that owns the
  receipt.
- Separates payment ownership from benefit assignment explicitly.
- Makes AppView authoritative for server-backed paid capabilities.
- Allows additional web subscriptions to become more license rows later.
- Avoids a large bundle allowance and a duplicated native-store catalog.

Cons:

- Requires a private billing-account concept and assignment UI.
- Creates an intentional server-side relationship between the billing owner and
  assigned DIDs.
- Requires separate Apple subscription groups so one Apple Account can hold both
  products.
- Pro and Business cannot use Apple's ordinary same-group upgrade/downgrade path.
- Store subscription-management screens cannot show the assigned CraftSky DID.
- RevenueCat entitlements alone cannot express assignment or future quantities.

**Option 6: Sell all paid access only through the web - not chosen for launch**

Use web checkout to create an independent subscription for every DID and let the
mobile apps consume the resulting access.

Pros:

- Supports arbitrary license quantities without native-store product limits.
- Allows each web subscription to have an explicit CraftSky billing context.
- Provides greater control over checkout and subscription management.

Cons:

- Adds web checkout, tax, invoicing, payment recovery, and customer-portal scope
  before native subscriptions launch.
- Mobile purchase links and steering behavior vary by storefront, region, and
  applicable Apple or Google program.
- Does not meet the initial preference to make subscriptions available directly
  through the App Store and Google Play.

Web billing remains a planned extension of the chosen license model.

### What I decided

CraftSky will separate the payer from the account receiving benefits.

A private CraftSky billing account will represent the payer and will have:

- a stable internal ID;
- a stable private UUID used as the RevenueCat App User ID; and
- an authenticated owner DID.

The UUID is independent of the active DID, installation, handle, email address,
and store account identifier. It must remain stable across CraftSky account
switches and devices.

At native-store launch, a billing account may own:

- at most one Pro subscription producing one Pro account license; and
- at most one Business subscription producing one Business account license.

The payer may own neither, either, or both subscriptions. Each active license is
assignable to at most one DID. Business includes Pro capabilities, so CraftSky
will prevent assigning both paid licenses to one DID by default rather than
encouraging accidental double payment.

AppView is authoritative for provider subscription state, license state, DID
assignment, and effective tier. Flutter asks AppView for the active DID's access
and must not apply the billing customer's combined RevenueCat `CustomerInfo`
directly to the active account.

Billing data and assignments remain private in Postgres. No subscription,
billing-account, or license-assignment record is written to a PDS.

Assignment is explicit and authorized. CraftSky must not infer a billing group
from the device-local multi-account registry. The target DID must be
authenticated by its controller or explicitly accept an invitation before it
receives a license. Initial requirements may limit assignment to accounts
currently authenticated on one installation.

### Store shape

Apple products use two subscription groups so one Apple Account may own both
licenses concurrently:

```text
CraftSky Pro
  - Pro Monthly
  - Pro Annual (future)

CraftSky Business
  - Business Monthly
  - Business Annual (future)
```

The products must be described as independently assignable account licenses,
because separate groups are justified by simultaneous ownership rather than an
ordinary single-account tier change.

Google Play uses separate Pro and Business subscription products. Billing-period
and offer details remain catalog-design decisions.

RevenueCat validates purchases and emits subscription lifecycle events for the
billing customer. AppView normalizes those provider subscriptions into distinct
licenses. RevenueCat entitlement keys may identify Pro and Business product
access, but they are not the authority for license count or DID assignment.

The RevenueCat restore configuration must not silently transfer one payer's
receipt to an unrelated CraftSky billing identity. Exact restore and account
recovery behavior is deferred to detailed requirements and must be tested against
both stores before release.

### Future web licenses

The durable data model must represent provider subscriptions and licenses as
rows with stable provider subscription identifiers. It must not use fixed
`pro_did` and `business_did` columns or assume that `(billing account, tier)` is
permanently unique.

This permits a later web flow to add more licenses:

```text
Billing account
  - native-store Pro license -> DID A
  - native-store Business license -> DID B
  - web Pro license -> DID C
  - web Pro license -> DID D
  - web Business license -> DID E
```

Every web subscription is a separate provider subscription and produces a
separate assignable license. AppView's authorization path remains unchanged.
RevenueCat Billing or Stripe integrated with RevenueCat are candidate providers;
their repeated-purchase and customer-portal behavior must be verified during web
billing design.

### Compatibility and evolution

- Existing free accounts remain free and need no billing account or license.
- The number of service-level free accounts is independent of the Flutter
  installation's current five-session limit.
- Existing multi-account sessions remain device-local. Billing relationships do
  not merge sessions, credentials, feeds, drafts, or other account data.
- A billing assignment is an explicit, private, opt-in server relationship and
  does not weaken normal DID authorization.
- Handles may change without affecting subscription assignment because licenses
  bind to DIDs.
- OAuth access tokens, refresh tokens, DPoP keys, and PDS credentials remain
  AppView-only under the existing architecture.
- Paid access does not alter feed chronology, ranking, search placement,
  moderation, reach, or PDS data ownership.
- Fixed bundles can be introduced later by making one provider product produce a
  defined set of license rows or capacities; the account authorization model
  does not change.
- Web purchases can add arbitrary licenses later without changing the native
  store receipt model.

### Consequences and notes

- AppView needs private billing-account, provider-subscription, license, and
  assignment state plus an idempotent RevenueCat webhook boundary and periodic
  reconciliation.
- Server-backed paid operations must authorize the authenticated DID against
  AppView's current effective-tier state.
- Flutter needs separate concepts for the active CraftSky account and the
  RevenueCat billing customer used for purchase and management operations.
- Subscription management must show each provider, current assignment, and the
  correct provider-specific management destination.
- Cancellation normally retains access until period end; refunds, grace periods,
  billing retry, expiration, reassignment, and deletion require explicit state
  transition requirements.
- Apple cannot natively replace the Pro subscription with Business across the
  two groups. Moving one DID from Pro to Business may involve overlapping paid
  periods, cancellation, or reassignment and needs deliberate UX.
- License reassignment needs abuse controls so a single license cannot be rapidly
  shared among many DIDs.
- Billing-owner deletion requires a transfer, cancellation, or other explicit
  resolution; it cannot silently orphan subscriptions.
- Existing business-profile architecture treats `business` as a self-declared
  account classification that does not imply payment or verification. Detailed
  subscription requirements must decide which business tools require a Business
  license without silently changing that classification's meaning.
- Proposed paid benefits that are already implemented without entitlement checks
  need an explicit transition plan. Expiry must not delete public PDS records or
  private member data merely because paid access ended.

### Out of scope for this ADR

- Final tier names, prices, billing periods, trials, and introductory offers.
- The exact list of Pro and Business benefits or service limits.
- Database migrations, API routes, webhook schemas, and Flutter providers.
- Reassignment frequency, invitation UX, and support overrides.
- Detailed upgrade, downgrade, cancellation, refund, restore, and account
  recovery behavior.
- Whether Family Sharing is enabled.
- Web provider selection, taxes, invoices, checkout implementation, and
  storefront-specific purchase links.
- Changing the current five-account local session limit.
- Any Lexicon or public PDS record change.

### Related references

- `docs/changes/2026-09-07-account-subscriptions/00-direction.md` records the
  selected product direction and deferred questions.
- `docs/changes/2026-07-17-multi-account-sessions/01-requirements.md` governs the
  independent device-local session registry and active-account boundary.
- `docs/changes/2026-08-27-business-profiles/01-requirements.md` records the
  existing distinction between self-declared business classification and paid
  access.
- RevenueCat customer identity:
  `https://www.revenuecat.com/docs/customers/identifying-customers`
- RevenueCat restore behavior:
  `https://www.revenuecat.com/docs/projects/restore-behavior`
- Apple subscription groups:
  `https://developer.apple.com/help/app-store-connect/manage-subscriptions/offer-auto-renewable-subscriptions/`
- Google Play subscriptions:
  `https://support.google.com/googleplay/android-developer/answer/12154973`
