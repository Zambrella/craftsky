# Promotional access and early subscriptions

RevenueCat promotional entitlements can provide Plus or Business to an account
without a store purchase. This change consumes verified grants for existing
account UUIDs; it does not allocate a launch campaign, grant anyone access, or
enable a production integration.

## Provider and storage

- Reconciliation recognizes `store: promotional` with the documented nullable
  `product_id`. A complete entitlement list must identify the configured project
  and `plus` or `business` lookup key; the shared `paid` key alone grants no tier.
- Nested entitlement pagination is authenticated only against the same origin
  and exact subscription endpoint, bounded, and validated before replacing any
  known access. A malformed or partial snapshot leaves the old projection intact.
- Both expiry fields must be explicitly numeric or null. The earliest known end
  bounds access; explicit null bounds permit a lifetime grant. Expired grants
  cannot retain access simply because the provider snapshot has not refreshed.
- Migration 000126 stores grants separately from paid subscriptions and licences.
  Canonical access queries combine valid grants and paid licences, retain
  production/sandbox isolation and owner lifecycle checks, and choose the highest
  tier. Business includes Plus. Legitimate paid/promotion overlap is not a paid
  duplicate; actual paid duplicates retain their existing anomaly handling.
- Complete snapshots reconcile revocation/removal. Grants participate in access
  fencing and feature checks, including scheduling, pins and business features.
  A grant is not a recurring billing obligation during account deletion.

RevenueCat records promotional transactions in its production environment even
when a client uses a sandbox SDK. This implementation preserves the existing
environment boundary; it does not mirror production gifts into sandbox access
or accept synthetic sandbox promotional transactions as access sources.

## API and clients

Billing state adds an optional, default-empty `promotionalGrants` array. Each item
has a private grant UUID, `tier`, `givesAccess` and nullable/optional `expiresAt`.
The existing `subscriptions` and `licenses` arrays describe paid billing only.
The access response continues to describe the highest effective tier.

Flutter and Svelte show complimentary access and expiry separately. A grant-only
account can subscribe to either tier before the grant ends. The ordinary paid
flow starts billing immediately; unused complimentary time is not added to the
paid term. The grant itself retains its original expiry. A real paid subscription
or pending checkout still prevents an independent duplicate purchase.

Native plan changes resolve real paid purchase evidence rather than whichever
entitlement is currently dominant in RevenueCat, since a promotional Business
grant can temporarily overshadow paid Plus. Paid-purchase completion requires
the new paid licence to reconcile even when the effective tier stays higher due
to a grant.

## Validation

- RevenueCat integration package passed under `-race`, including complete JSON
  decoding through reconciliation, promotional Business plus paid Plus overlap,
  malformed snapshots preserving known access, and revocation falling back to
  paid Plus. A separate read-only provider-decoder review found no objections.
- Flutter subscription/auth/AppShell suite: 225 passed, five existing skips.
  Final affected suites after cleanup: 61 passed. Analysis found no issues;
  generated mapper/localizations and formatting checks passed.
- Svelte: 29 focused unit tests and 19 browser fixtures passed (five promotion
  cases plus existing billing/identity regressions). Check reported no errors or
  warnings; lint, production build and formatting passed.
- Business grant tests passed under `-race` against full migrated schemas:
  account type, profile and event visibility, expiry/revocation and paid overlap.
- Full subscription and Business packages passed under `-race`; focused API and
  route regressions passed for subscriptions, checkout/management, profile pins,
  saved folders and Business behavior. Scheduling and accepted-deletion fence
  tests, grant ownership/environment isolation and migration rollback checks
  also passed. A separate backend review found no objections. The final global
  diff whitespace check passed.

No running debug application was available for hot reload. No production
migration, deployment, grant creation or real store purchase is included in local
verification. Migration 000126 and the existing integration activation gates must
be completed through the normal release process. Production acceptance must
also verify grant/revoke event delivery and effective access through the
authenticated RevenueCat-to-AppView receiver; local fixtures do not prove that
provider delivery.

## Sources

- [RevenueCat customer grants](https://www.revenuecat.com/docs/dashboard-and-metrics/customer-profile#granting-promotional-entitlements)
- [RevenueCat subscription schema](https://www.revenuecat.com/docs/api-v2/subscription-data-model)
- [Subscription entitlements endpoint](https://www.revenuecat.com/docs/api-v2/subscription#tag/Subscription/operation/list-subscription-entitlements)
