# Coding Plan: Flutter Account Subscriptions

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`, 2026-09-10)
- Architecture: `adr/014-account-assigned-subscription-licenses.md`
- API conventions:
  `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- Wire conventions:
  `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`

## 2. Implementation Strategy

Implement the approved contract in two boundaries, in dependency order:

1. Add the internal subscription/license relationship to the existing AppView
   owner response. This is a response projection only: add
   `BillingLicense.subscriptionId`, select the existing
   `provider_subscriptions.id`, and extend the registered-route contract test.
   No migration, route, request, reconciliation, assignment, or authorization
   change is needed.
2. Add a Flutter `subscriptions` feature that treats AppView self-access as the
   only account-access authority and RevenueCat as a replaceable native billing
   service. Persist one owner reservation with a required DID and optional UUID
   at the top level of the secure session registry, read access through each
   account's existing `accountDioProvider`,
   and serialize owner actions through one lease-fenced controller.

The Flutter implementation will not persist `CustomerInfo`, receipts,
transactions, offerings, or owner billing state. The RevenueCat service may be
configured on supported native startup, including its temporary SDK-managed
anonymous state, but every billing action stays disabled until the exact AppView
UUID is confirmed. Active-account changes never drive RevenueCat login/logout.

Keep the first implementation narrow:

- One owner controller rather than separate production controllers for every
  button. Test files may isolate purchase, restore, assignment, and Customer
  Center behavior by method.
- One consolidated owner wire model and one self-access model rather than a file
  per DTO.
- Pure helpers for purchase eligibility, identity readiness, candidate selection,
  and reconciliation state only where their state machines need direct unit
  tests.
- Small page-local widgets first; extract only tier badges and reusable status or
  confirmation components.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| AppView owner projection | `subscriptions.BillingState` assembled transactionally in `Store.OwnerState` and serialized by existing GET/PUT handlers | Add non-null internal `subscriptionId` to each license and test shuffled response arrays | FR-012, FR-027, NFR-001 | IT-013, IT-001, UT-012, AT-013 |
| Secure account registry | Strict versioned `SessionRegistry` snapshot in `flutter_secure_storage` | Persist one top-level redacted owner reservation; reserve DID before ensure, add UUID before RevenueCat work, migrate v2 snapshots to v3 with no reservation, and never remove it with a session | FR-003, FR-006, FR-008, RULE-002, RULE-008 | AT-001, AT-011, UT-002, IT-004, REG-005 |
| AppView Flutter client | Feature API clients use account-specific Dio and standard error mapping | Add self-access, owner, reconciliation, assignment, and unassignment methods | FR-009, FR-012, FR-017, FR-020, FR-021, FR-027 | IT-001, IT-002, IT-008 |
| RevenueCat boundary | Native integrations are initialized behind replaceable providers | Add platform/config resolver, native adapter, and unavailable adapter; never expose SDK `CustomerInfo` as access | FR-001, FR-002, FR-004, FR-005, FR-014 through FR-016, FR-022, FR-023 | UT-003, UT-006, IT-003, IT-005, IT-007, IT-009, IT-012 |
| Account access state | Families use `AccountSessionLease` and `accountDioProvider` | Add lease-keyed self-access provider for active settings and inactive switcher rows | BR-002, FR-007, FR-009 through FR-011, FR-026 | AT-003, AT-004, AT-015, UT-001, IT-002, IT-010, IT-011 |
| Owner operations | Existing mutations capture `ActiveAccountLease`, recheck it after awaits, and invalidate on boundaries | Add exact reserved-owner guard, owner page controller, baseline reconciliation coordinator, and serialized operation state | FR-004, FR-006 through FR-008, FR-013, FR-017 through FR-024, FR-026 | AT-002, AT-005 through AT-011, AT-014, AT-017, UT-004 through UT-008, UT-010, UT-011, UT-013, IT-003 through IT-010, IT-014 |
| Settings and routing | Typed `go_router` routes under `/profile/settings`; descriptor-backed settings rows | Add `/profile/settings/subscriptions`, a settings row, and typed route generation | FR-009, FR-025 | AT-004, AT-013, IT-011 |
| Account switcher | `LiveAccountSwitcherContent` renders retained rows without changing active account while loading | Add an account-scoped tier badge per row with loading/unavailable isolation | FR-010, FR-011, FR-026 | AT-015, IT-002, IT-010, REG-002 |
| Localization and UI | ARB localization, Material theme, semantics and narrow-screen widget tests | Add owner/beneficiary, purchase, pending, assignment, restore, management, and fallback copy | FR-024, FR-025, NFR-005 | AT-012 through AT-016, UT-010, MAN-005 |
| Privacy and diagnostics | Redacted model strings, `ProviderLogger`, `AppErrorMapper`, Sentry sanitizer, secret scans | Redact billing DTOs, configure an explicit release-safe RevenueCat log level, install no raw SDK forwarding handler, sanitize provider exceptions, and report only operation/tier/platform/outcome classifications | FR-005, NFR-002 | UT-009, REG-006 |
| Native/build configuration | `--dart-define-from-file`, generated plugin files, pub lock, iOS 15 target, inherited Android minimum | Add both SDK packages, public key names, explicit iOS Podfile 15 target, and build assertions | FR-001, FR-002, NFR-003, NFR-007 | UT-003, IT-012, REG-003, REG-007, MAN-001 through MAN-004 |

## 4. Files And Modules

### AppView

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/subscriptions/types.go` | Change | Add required `SubscriptionID uuid.UUID` with JSON key `subscriptionId` to `BillingLicense` | FR-027 | IT-013 |
| `appview/internal/subscriptions/store.go` | Change | Select and scan `subscription.id` in the existing owner-license query | FR-012, FR-027 | IT-013 |
| `appview/internal/routes/subscription_routes_test.go` | Change | Extend `TestOwnerBillingStateUsesPrivateCamelCaseContract` for two shuffled subscription/license pairs on GET and existing-account PUT; retain provider-ID leak assertions | FR-027, NFR-001, NFR-002 | IT-013 |

Do not change AppView migrations, routes, handler request shapes, reconciliation
logic, assignment logic, or generated lexicon code. The acceptance target named
`appview/internal/api/subscriptions_test.go` maps to the stronger existing
registered-route suite above rather than creating a duplicate database test.

### Flutter Production

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/auth/models/billing_owner_binding.dart` | Create | Strict, redacted owner reservation with required DID and optional UUID | FR-003, FR-006, FR-008 | UT-002, IT-004 |
| `app/lib/auth/models/session_registry.dart` | Change | Add top-level reservation, first-DID-only transition, same-DID UUID completion, v2-to-v3 decode, v3 encode, and preservation on session removal | FR-003, FR-006, FR-008, RULE-002 | AT-001, AT-011, IT-004 |
| `app/lib/auth/providers/session_registry_provider.dart` | Change | Serialize durable DID reservation and UUID completion; expose no clear/replace-owner mutation | FR-003, FR-006, FR-008 | IT-004 |
| `app/lib/subscriptions/models/subscription_access.dart` | Create | Strict `free`/`plus`/`business` self-access model with dormant assigned tier | FR-009, NFR-001, RULE-001 | UT-001, UT-012 |
| `app/lib/subscriptions/models/billing_state.dart` | Create | Owner account, subscription, license, assignment, and reconciliation DTOs; validate UUID relationships; retain provider strings for fallback display | FR-012, FR-027, NFR-001 | UT-012, UT-010, IT-001 |
| `app/lib/subscriptions/models/subscription_operation_state.dart` | Create | Redacted UI/coordinator states for idle, unavailable, running, pending, completed, and failed outcomes | FR-017, FR-024, NFR-002 | UT-005, UT-010, AT-014 |
| `app/lib/subscriptions/models/subscription_rules.dart` | Create | Pure tier correlation, purchase eligibility, assignment-candidate, and presentation classification helpers | FR-013, FR-019, FR-024 | UT-004, UT-007, UT-010 |
| `app/lib/subscriptions/data/subscription_api_client.dart` | Create | Exact `/v1/subscriptions/access` and `/v1/billing/*` JSON/HTTP methods | FR-009, FR-012, FR-017, FR-020, FR-021 | IT-001 |
| `app/lib/subscriptions/data/subscription_repository.dart` | Create | Replaceable AppView interface scoped to one account Dio | NFR-006 | IT-002, IT-006 through IT-010 |
| `app/lib/subscriptions/data/api_subscription_repository.dart` | Create | Thin API-backed repository implementation | FR-009, FR-012, FR-017, FR-020, FR-021 | IT-001, IT-002 |
| `app/lib/subscriptions/services/revenuecat_service.dart` | Create | SDK-free interface, opaque offering handle, direct paywall result, restore, identity, and Customer Center callback contracts | FR-001, FR-004, FR-014 through FR-016, FR-022, FR-023, NFR-006 | UT-003, UT-006, IT-003, IT-005, IT-007, IT-009 |
| `app/lib/subscriptions/services/revenuecat_service_native.dart` | Create | iOS/Android adapter over `Purchases` and `RevenueCatUI`; catch and sanitize platform failures, model attachment failures at presentation, and discard `CustomerInfo` as authority | FR-001, FR-004, FR-014 through FR-016, FR-022, FR-023, NFR-002 | UT-006, UT-009, MAN-001 through MAN-004 |
| `app/lib/subscriptions/services/revenuecat_service_unavailable.dart` | Create | Safe no-op/error boundary for web, desktop, blank keys, and failed native initialization | FR-002 | AT-012, UT-003, REG-003 |
| `app/lib/subscriptions/services/revenuecat_bootstrap.dart` | Create | Resolve platform public keys, set explicit debug/release-safe SDK logging without raw forwarding, configure once on native platforms, and return unavailable service on failure | FR-001, FR-002, NFR-002, NFR-003 | UT-003, UT-009, IT-012 |
| `app/lib/subscriptions/services/billing_owner_guard.dart` | Create | Capture exact reservation plus active lease; check before every approved external call and fence completions | FR-004, FR-026, NFR-004 | UT-008, UT-013, IT-003, IT-010, IT-014 |
| `app/lib/subscriptions/services/reconciliation_coordinator.dart` | Create | Baseline-generation bounded poll and purchase/restore outcome classification with injectable delay | FR-017, FR-018, FR-022, NFR-008 | UT-005, UT-011, IT-006, IT-007 |
| `app/lib/subscriptions/providers/subscription_repository_provider.dart` | Create | Build repository from `accountDioProvider(account)` | FR-011, FR-026 | IT-002 |
| `app/lib/subscriptions/providers/subscription_access_provider.dart` | Create | Lease-keyed self-access family with post-await lease verification | FR-009 through FR-011, FR-026 | AT-003, AT-004, AT-015, IT-002, IT-010 |
| `app/lib/subscriptions/providers/revenuecat_service_provider.dart` | Create | Expose bootstrap-selected native or unavailable service for overrides | FR-001, FR-002, NFR-006 | UT-003, IT-003, IT-005 |
| `app/lib/subscriptions/providers/owner_billing_controller.dart` | Create | Build page state and serialize setup, refresh, purchase, restore, manage, reconcile, assign, and unassign under one owner lease | FR-003 through FR-008, FR-012 through FR-024, FR-026 | AT-001, AT-002, AT-005 through AT-011, AT-017, IT-004 through IT-010, IT-014 |
| `app/lib/subscriptions/pages/subscription_page.dart` | Create | Render self-access for all accounts and owner controls only for the active exact owner | FR-007, FR-009, FR-012, FR-024, FR-025 | AT-004, AT-013, AT-014, AT-016 |
| `app/lib/subscriptions/widgets/subscription_tier_badge.dart` | Create | Reusable free/Plus/Business loading/unavailable badge for settings and switcher | FR-010 | AT-015 |
| `app/lib/subscriptions/widgets/subscription_license_card.dart` | Create | Owner tier/status/assignment actions; keep confirmations local unless reused | FR-012, FR-019 through FR-021, FR-025 | AT-008, AT-013 |
| `app/lib/auth/providers/account_boundary_provider.dart` | Change | Invalidate access and owner subscription providers during account boundaries | FR-026 | IT-010, REG-005 |
| `app/lib/auth/widgets/account_switcher_content.dart` | Change | Render a lease-keyed tier badge for each retained row without activation | FR-010, FR-011 | AT-015, REG-002 |
| `app/lib/bootstrap.dart` | Change | Initialize subscription mappers and override the RevenueCat service on native startup without making bootstrap throw | FR-001, FR-002, NFR-001 | IT-012, REG-003 |
| `app/lib/router/route_locations.dart` | Change | Add `subscriptionsChild` and canonical settings location | FR-009 | IT-011 |
| `app/lib/router/router.dart` | Change | Add typed `SubscriptionsRoute` under `SettingsRoute` on the authenticated shell navigator | FR-009 | IT-011 |
| `app/lib/settings/models/settings_row.dart` | Change | Add `SettingsRowId.subscriptions` in the General section | FR-009, FR-025 | IT-011 |
| `app/lib/settings/pages/settings_page.dart` | Change | Add localized subscriptions row and current account tier subtitle | FR-009, FR-010 | AT-004, IT-011 |
| `app/lib/l10n/app_en.arb` and other committed ARBs | Change | Add all subscription, identity, pending, error, confirmation, and accessibility copy | FR-024, FR-025, NFR-005 | AT-014, AT-016 |
| `app/pubspec.yaml`, `app/pubspec.lock` | Change | Add compatible `purchases_flutter` and `purchases_ui_flutter` versions | FR-001 | IT-012, REG-007 |
| `app/config/*.env.example`, `app/config/README.md` | Change | Add blank platform public-key names and setup documentation; never commit real values | FR-001, FR-002, NFR-003 | IT-012 |
| `app/ios/Podfile`, lock/generated native plugin files as produced | Change | Declare iOS 15 explicitly and accept dependency-resolution output only | FR-001, NFR-007 | IT-012, MAN-001 |
| Android/generated plugin files as produced | Verify / Change only if generated | Confirm inherited min SDK satisfies API 21; do not raise it without build evidence | FR-001, NFR-007 | IT-012, MAN-002 |

### Generated Files

Regenerate rather than manually edit:

- `app/lib/**/*.mapper.dart` for new `dart_mappable` DTOs.
- `app/lib/**/*.g.dart` for Riverpod providers/controllers.
- `app/lib/router/router.g.dart` for the typed route.
- `app/lib/l10n/generated/*` for ARB changes.
- Flutter plugin registrants and CocoaPods locks produced by dependency
  resolution, retaining only deterministic expected changes.

## 5. Services, Interfaces, And Data Flow

### 5.1 AppView Response Projection

The database already enforces:

```text
billing_licenses.provider_subscription_id
  -> provider_subscriptions.id
```

`BillingSubscription.id` is already `provider_subscriptions.id`. Extend the
owner-license query to select that same UUID:

```text
BillingLicense {
  id,
  subscriptionId, // internal provider_subscriptions.id
  tier,
  assignedDid,
  assignedAt,
  assignable,
  anomaly
}
```

The existing owner transaction and GET/PUT handlers remain unchanged. The route
test will seed inverse subscription/license ordering and assert ID-based joins,
required camelCase output, same-owner privacy, and absence of raw RevenueCat IDs.

### 5.2 AppView Flutter Repository

Partial interface:

```text
abstract interface class SubscriptionRepository {
  Future<SubscriptionAccess> getAccess();
  Future<BillingState> ensureBillingAccount(); // first setup only
  Future<BillingState> getBillingAccount();    // UUID reservation/recovery
  Future<void> requestReconciliation();
  Future<Assignment> assign(licenseId, targetDid);
  Future<void> unassign(licenseId);
}
```

Construct one repository per `AccountKey` from that account's
`accountDioProvider`. This preserves fixed token/device headers and existing
lease-aware 401 invalidation for active and inactive retained accounts. The API
client maps exact AppView envelope values, especially the four assignment errors,
instead of branching only on HTTP 409/422.

### 5.3 RevenueCat Boundary

Keep SDK types behind the interface so tests never initialize platform channels:

```text
abstract interface class RevenueCatService {
  BillingAvailability get availability;
  Future<RevenueCatIdentity> currentIdentity();
  Future<void> identify(String appViewUuid);
  Future<RevenueCatOffering?> getOffering(String identifier);
  Future<DirectPaywallResult> presentPaywall(RevenueCatOffering offering);
  Future<void> restorePurchases();
  Future<void> presentCustomerCenter(CustomerCenterCallbacks callbacks);
}

enum DirectPaywallResult { purchased, restored, cancelled, error }
```

`RevenueCatOffering` is an opaque interface/reference implemented privately by
the native adapter; fake tests provide a trivial implementation. Do not expose
or persist `CustomerInfo` or use entitlements to determine DID access.

Native startup rules:

- Resolve the key for iOS or Android only.
- Blank/missing key, web/desktop, or caught configuration failure returns an
  unavailable service and leaves the rest of the app usable.
- Configure `Purchases` once. The SDK may initially own an anonymous identifier,
  but no purchase, restore, offering, or Customer Center call is reachable then.
- Set an explicit SDK log level: verbose/debug diagnostics may be enabled only in
  debug builds; release uses a non-verbose level. Do not install an SDK log
  handler that forwards raw messages to CraftSky logs or Sentry. Map
  `PlatformException` and SDK failures to closed internal classifications without
  retaining raw messages/details.
- Before owner work, accept an already matching UUID. An anonymous SDK identity
  may call `Purchases.logIn(exactAppViewUuid)`. A different identified UUID fails
  locked: never call `logOut`, never directly `logIn` another identified UUID,
  and never use a DID/handle/device ID.
- Resolve the tier's named offering and required package before presentation.
  Directly present that offering with `displayCloseButton: true`; never call
  `presentPaywallIfNeeded` or `purchasePackage`. The SDK does not expose a
  dependable attached-paywall preflight, so catch/map presentation failures as
  provider errors. Require manual verification of a real close affordance for
  every configured paywall version because the flag alone is insufficient for
  all template versions.
- Customer Center callbacks set refresh intent only; they do not grant access.

### 5.4 Owner Reservation Lifecycle

`BillingOwnerBinding` stores the owner DID and an optional AppView-generated
RevenueCat UUID. It is top-level registry state, not a `StoredSession` field, so
`SessionRegistry.remove` cannot erase it.

```text
never reserved + active confirmer:
  recheck active lease
  durably reserve DID with UUID absent
  recheck active lease and matching DID-only reservation
  PUT ensure (idempotent retry is allowed for this DID only)
  verify and durably add returned UUID to the same reservation
  recheck active lease and UUID reservation
  identify RevenueCat only from anonymous/matching state

DID-only reservation + same DID reauthenticated:
  recheck active lease and matching reservation
  retry PUT ensure
  persist returned UUID before any RevenueCat identity call

UUID reservation + same DID reauthenticated:
  recheck active lease
  GET owner state only
  if UUID matches: verify/identify safe SDK state, enable billing
  if 404 or mismatch: remain locked; no PUT, logout, or reidentification

reserved + another active DID:
  self-access only + switch-to-owner/reauth guidance
  no owner GET, PUT, RevenueCat call, or owner details
```

The two durable writes are ordering barriers, not optimistic state updates. If
DID persistence fails, make no network call. If PUT fails or the process stops,
retain the DID-only reservation and permit only that DID to retry. If UUID
persistence fails, make no RevenueCat call; the same DID may retry idempotent PUT.
After UUID persistence, PUT is unreachable and recovery is GET-only. Decode
existing schema-v2 snapshots as v3 with no reservation and write only schema v3
thereafter. Malformed reservation data fails the whole secure snapshot closed,
matching current behavior.

### 5.5 Purchase And Reconciliation Flow

```text
capture active exact-owner lease
GET fresh owner state
correlate licenses by license.subscriptionId == subscription.id
evaluate tier eligibility fail-closed
recheck lease -> get named offering
recheck lease -> verify/identify exact UUID
recheck lease -> present direct paywall

if cancelled/error: stop without AppView mutation
if purchased/restored:
  retain pre-provider state for outcome comparison
  recheck lease -> GET owner state; capture baseline requestedGeneration
  recheck lease -> POST reconciliation
  poll with owner recheck before every GET:
    first requestedGeneration > baseline becomes target
    wait until reconciledGeneration >= target
    purchase: also require new license, same-subscription recovery, or anomaly
    restore: unchanged state may complete
```

Use a documented 30-second maximum and an injectable one-second delay rather than
adding a fake-clock dependency. If no post-baseline generation or expected
purchase state appears, publish a pending timeout with refresh/retry, never a
purchase retry. Disposal or stale owner lease stops the delay/poll and fences the
completion.

Outcome comparison uses internal IDs from the pre-provider state:

- Existing subscription changes from inaccessible to accessible: preserve its
  license and assignment. Refresh the assigned DID's self-access; if unassigned,
  prompt assignment.
- New subscription/license ID: leave it unassigned and prompt explicit target
  selection. Do not replace a dormant old assignment.
- Reconciled anomaly or unresolved relationship: support-oriented non-success;
  no activation or second checkout.
- Restore with no changes: completed restore, unchanged access.

### 5.6 Purchase Eligibility

Build a validated relation by internal ID, never array order. An orphan license,
duplicate relationship, or unresolved owner subscription fails checkout closed.
For the requested tier:

- Eligible if no correlated subscription exists.
- Eligible after lapse only if every correlated subscription is inaccessible,
  not pending payment, has exact `autoRenewalStatus == 'will_not_renew'`, both
  subscription and license anomalies are empty, and reconciliation is current.
- Block for null/omitted/empty, every known non-terminal renewal value, every
  unknown renewal value, and every known/unknown anomaly while still rendering
  unknown provider values with safe fallback text.

Plus and Business are evaluated independently, except candidate assignment
excludes a DID already holding either paid tier.

### 5.7 Assignment And Access Refresh

Build candidates from current retained leases and current owner licenses. The UI
may filter obvious conflicts, but AppView remains final authority. After explicit
tier/target confirmation:

- Recheck the exact owner lease before PUT/DELETE.
- Serialize mutations and disable duplicate taps.
- Refresh owner state after every non-401 response.
- On assignment success, refresh the target's lease-keyed self-access and only
  then show the effective tier.
- Map target-session races to `assignment_target_ineligible`.
- Send owner 401 through existing lease-scoped invalidation.
- Preserve server assignment on conflict, cooldown, missing license, or generic
  failure; unassignment only expects missing-license or generic/401 failures.

### 5.8 Customer Center And Lifecycle

The owner controller presents Customer Center through the service under the same
identity guard. A small coordinator coalesces refresh intent from restore,
management-option callbacks, dismissal after an action, and app resume after an
external management destination. Reconciliation still follows the baseline
algorithm. Plain dismissal with no management action may refresh the owner GET
but does not claim provider mutation or account activation.

## 6. State, Providers, Controllers, Or DI

### Provider Graph

```text
sessionRegistryProvider
  -> billingOwnerBindingProvider (derived DID + optional UUID reservation)
  -> accountDioProvider(AccountKey)
       -> subscriptionRepositoryProvider(AccountKey)
            -> subscriptionAccessProvider(AccountSessionLease)
            -> ownerBillingControllerProvider (active exact owner only)

revenueCatServiceProvider (bootstrap override; fake in tests)
  -> ownerBillingControllerProvider

ownerBillingControllerProvider
  -> billingOwnerGuard
  -> reconciliationCoordinator
  -> subscriptionAccessProvider(target lease) invalidation/refresh
```

Provider choices:

- `billingOwnerBindingProvider`: synchronous `Provider`, derived from the loaded
  registry; UI handles registry loading separately.
- `subscriptionRepositoryProvider(AccountKey)`: `FutureProvider.family` because
  `accountDioProvider` is asynchronous.
- `subscriptionAccessProvider(AccountSessionLease)`: auto-dispose
  `FutureProvider.family`; the lease in the key prevents cross-generation reuse.
- `revenueCatServiceProvider`: keep-alive `Provider<RevenueCatService>` whose
  default is unavailable and whose native bootstrap value is injected at the
  root `ProviderScope`.
- `OwnerBillingController`: keep-alive only while the authenticated app/session
  registry is alive; use an `AsyncNotifier` with explicit operation substate and
  methods. It must not stringify DTO contents in provider logs.

The controller captures a `BillingOwnerOperation` record containing the
reservation and `ActiveAccountLease`. The guard has no permissive null path:
unlike the general existing helper, subscription owner operations fail when
registry/lease state is absent. Recheck immediately before every external call
listed by FR-026 and after each await before publishing.

Add subscription providers to `accountStateInvalidatorProvider`. Family access
providers also self-fence by confirming that `registry.leaseFor(account)` still
equals their captured lease before returning data.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Route And Entry Point

- Add typed `SubscriptionsRoute` at
  `/profile/settings/subscriptions`, lifted to the authenticated shell navigator
  like other settings children.
- Add `SettingsRowId.subscriptions` under General before Account. The subtitle
  shows the active account's own effective tier or loading/unavailable, not owner
  billing state.
- Add the same lease-keyed compact tier badge to each account switcher row.

### Subscription Page Composition

```text
SubscriptionPage
  Scaffold/AppBar
  ActiveAccountAccessSection
    effective tier
    dormant assigned tier when present
  BillingAvailabilitySection
    unsupported/missing configuration explanation
  switch (owner relationship)
    neverReserved -> explicit permanent-owner confirmation
    reservedOwnerIncomplete -> same-owner setup retry or switch/reauth guidance
    reservedOwnerInactive -> beneficiary view + switch/reauth owner action
    reservedOwnerActive -> OwnerBillingSection
      owner identity explanation
      PlusLicenseCard
      BusinessLicenseCard
      restore/manage controls
      pending/retry/support state
```

Each license card shows only safe provider status/timing strings, assignment,
access, reconciliation freshness, anomaly fallback, and eligible actions. Unknown
provider values use localized “Status unavailable” text while their raw values
remain out of UI/logs.

Assignment uses an explicit confirmation sheet/dialog naming tier and target.
Unassignment and reassignment use explicit confirmation; reassignment copy states
the seven-day cooldown before dispatch. Owner setup copy states that the payer is
permanently pinned on this installation and switching accounts will not move
purchases.

For accessibility, keep controls at least 48 logical pixels, add semantic labels
that include tier/status/action, permit text wrapping, use scrolling content at
320-pixel width and 2x text scale, and preserve platform back/dismiss behavior.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Registry/access loading | Show account-local skeleton/progress; never reuse another row's tier | FR-010, FR-011 | AT-015, IT-002 |
| Access unavailable | Show localized unavailable badge/section; retain free app behavior | FR-002, FR-010 | AT-012, AT-015 |
| Never-reserved owner | Show explicit first-owner confirmation; confirmation durably reserves DID before PUT | FR-003 | AT-001, AT-011, UT-002 |
| DID-only reservation | Only the same active DID may retry idempotent PUT; no other DID or RevenueCat call | FR-003, FR-008 | AT-011, IT-004, IT-014 |
| Reserved owner inactive/missing | Hide owner details and mutations; offer switch/reauth; retain DID and optional UUID | FR-007, FR-008 | AT-004, AT-011 |
| UUID recovery GET 404/mismatch | Remain locked with support/recovery message; zero PUT/logout/reidentify | FR-003, FR-008 | AT-011, IT-004, IT-014 |
| SDK anonymous before owner setup | Billing disabled; identify only with exact AppView UUID during guarded setup | FR-004, FR-005 | IT-003 |
| SDK different identified UUID | Fail locked; never alias via login or logout | FR-004 through FR-006 | UT-002, IT-003 |
| Missing/blank key or unsupported platform | Use unavailable service; no native method channel call | FR-002 | AT-012, UT-003, REG-003 |
| Missing named offering/package | Pre-presentation unavailable state; do not call RevenueCatUI | FR-014, FR-024 | AT-006, AT-014, UT-006 |
| Missing/invalid attached paywall | Catch presentation return/throw and show retryable provider failure | FR-014, FR-024 | AT-006, AT-014, UT-006, IT-005 |
| Direct paywall cancelled | Normal return; no reconciliation, error, or success | FR-015, FR-016 | AT-006, REG-004 |
| Direct paywall error | Retryable provider failure; no AppView activation | FR-015, FR-024 | AT-006, AT-014 |
| Accessible/pending/renewable/unknown/anomalous tier | Hide/disable purchase, explain status, and refresh/support as appropriate | FR-013 | AT-005, UT-004 |
| Fully lapsed exact terminal tier | Permit paywall; disclose preserved dormant assignment | FR-013 | AT-002, AT-005, UT-004 |
| No post-baseline generation | Stop bounded poll and show pending refresh, never repurchase | FR-017, RULE-006 | AT-007, UT-005, UT-011 |
| Generation reconciled without purchase state | Continue until bound, then pending; restore may complete unchanged | FR-017, FR-022 | AT-007, IT-006, IT-007 |
| Same subscription recovers | Preserve assignment and refresh assigned DID access | FR-018 | AT-002, IT-006 |
| New post-lapse subscription | Preserve old dormant assignment; new license starts unassigned | FR-018, FR-019 | AT-002, IT-006 |
| Assignment target race | Map exact `assignment_target_ineligible`, refresh server state | FR-020 | AT-008, IT-008 |
| Conflict/cooldown/not found | Map exact envelope; preserve refreshed assignment | FR-020, FR-021 | AT-008, IT-001, IT-008 |
| Owner 401 | Use existing lease invalidation; disable owner work but retain reservation | FR-008, FR-020, FR-026 | AT-008, REG-005 |
| Account switch/removal during work | Stop before next call; ignore completion/navigation; cancel poll | FR-026, NFR-004 | AT-011, AT-017, UT-008, UT-013, IT-010, IT-014 |
| Unknown provider status/store/renewal/anomaly | Render safe fallback; block purchase where state affects eligibility | NFR-001, FR-013 | UT-004, UT-010, UT-012 |

## 9. Test Implementation Plan

Automated “integration” targets remain ordinary in-process `flutter test` suites;
do not create Flutter Driver or `integration_test/` infrastructure for this slice.
Use `ProviderContainer.test`, controlled completers, `http_mock_adapter`, and
small handwritten fakes. Do not add Mockito or a fake-clock package.

| Order | Test IDs | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-013 | `appview/internal/routes/subscription_routes_test.go` | Two inverse-ordered provider subscriptions/licenses and provider-ID canary | Owner license JSON lacks `subscriptionId` |
| 2 | UT-001, UT-012 | `app/test/subscriptions/models/subscription_wire_models_test.dart` | Golden access/owner fixtures, malformed types, unknown provider strings, shuffled arrays | Models/mappers do not exist |
| 3 | IT-001 | `app/test/subscriptions/data/subscription_api_client_test.dart` | Dio adapter matching exact methods, paths, bodies, and error envelopes | Client/repository methods do not exist |
| 4 | UT-002, IT-004 | Existing `auth/models/session_registry_test.dart`, `auth/providers/account_session_registry_provider_test.dart`, and secure-storage tests | v2/v3 snapshots, DID-only/DID+UUID reservations, failed writes, interruption, removal, restart, same/different DID/UUID | Registry has no crash-safe reservation or migration |
| 5 | AT-001, AT-011 | `app/test/subscriptions/providers/subscription_account_boundary_test.dart` | Registry/provider fakes and controlled persistence/GET/PUT/RevenueCat calls | DID is not persisted before PUT and UUID is not persisted before identity |
| 6 | UT-003, IT-012 | `app/test/subscriptions/revenuecat_platform_configuration_test.dart` | Injectable platform/key cases plus repository file assertions | Dependencies, key resolver, and explicit iOS target are absent |
| 7 | IT-003, UT-013, AT-017, IT-014 | `app/test/subscriptions/services/subscription_active_owner_guard_test.dart` and provider boundary suite | Exact owner, beneficiary active, stale/removed lease, every FR-026 call | External calls are not guarded |
| 8 | UT-004, AT-005 | `app/test/subscriptions/models/subscription_purchase_eligibility_test.dart` and page suite | Complete renewal/payment/anomaly/freshness matrix | No tier correlation or fail-closed rule exists |
| 9 | UT-006, AT-006, IT-005 | `app/test/subscriptions/providers/paywall_coordinator_test.dart` | Recording service, missing offering/package, attached-paywall throw, four direct results | No SDK-free paywall coordinator exists |
| 10 | UT-005, UT-011, AT-007 | `app/test/subscriptions/services/reconciliation_state_test.dart` and `reconciliation_polling_test.dart` | Injected delay, generation races, timeout, disposal | No baseline/target state machine exists |
| 11 | AT-002, IT-006 | `app/test/subscriptions/providers/subscription_purchase_controller_test.dart` | Scripted new license, same-subscription recovery, new post-lapse ID, access responses | Purchase cannot reach AppView-authoritative activation |
| 12 | UT-007 | `app/test/subscriptions/models/assignment_candidates_test.dart` | Retained leases and both-tier assignments | Candidate filtering does not exist |
| 13 | AT-008, IT-008 | `app/test/subscriptions/providers/subscription_assignment_controller_test.dart` | Exact PUT/DELETE envelopes, target access, duplicate taps, owner 401 | Assignment actions/error mapping do not exist |
| 14 | AT-009, IT-007 | `app/test/subscriptions/providers/subscription_restore_controller_test.dart` | Known assignment, new license, unchanged restore, exact owner UUID | Restore/reconciliation path does not exist |
| 15 | AT-010, IT-009 | `app/test/subscriptions/providers/customer_center_coordinator_test.dart` | Callback and app-resume events with coalesced refresh recorder | Customer Center lifecycle coordination does not exist |
| 16 | AT-003, AT-004, AT-013, AT-014, UT-010 | `app/test/subscriptions/pages/subscription_settings_page_test.dart` | Owner, beneficiary, dormant, stale, pending, anomaly, unknown provider states | Subscription page and presentation projection do not exist |
| 17 | IT-002 | `app/test/subscriptions/providers/subscription_access_provider_test.dart` | Active Alice/inactive Bob and recording account repositories | Inactive access is not scoped by its lease |
| 18 | AT-015, REG-002 | Extend `app/test/router/app_shell_account_switcher_test.dart` | Three retained rows with distinct loading/data/error access | Switcher has no isolated tier badges |
| 19 | IT-011 | New `app/test/router/subscription_routes_test.dart`; extend existing settings/router usage tests | Production router and settings harness | Route/row does not exist or typed usage check fails |
| 20 | AT-012, REG-001, REG-003 | Subscription page/config tests plus existing route/feature architecture tests | Unsupported services and all existing feature routes | Billing may appear or affect unsupported/free behavior |
| 21 | UT-009, REG-006 | `app/test/subscriptions/subscription_privacy_test.dart` and existing `observability/secret_scan_test.dart` | Sensitive canaries, raw platform exception, build mode, SDK logger recorder | New DTO/provider/native diagnostics may expose sensitive values or verbose release logs |
| 22 | AT-016, MAN-005 automation portion | `app/test/subscriptions/pages/subscription_accessibility_test.dart` | Semantics, 320px viewport, 2x text, back/dismiss | New page has no tested accessible layout |
| 23 | FR-024 localization support | `app/test/subscriptions/subscription_localizations_test.dart` | Enumerate required English strings and generated getters | Localized subscription copy is absent |
| 24 | REG-004, REG-005 | Existing messenger and session invalidation suites plus subscription controller tests | Cancellation and lease-scoped 401 | New flow may regress normal cancel/invalidations |
| 25 | REG-007, AC-027 | Full analyze/test gates | All generated outputs current | Analysis or deterministic suite fails |
| 26 | MAN-001 through MAN-005, AC-028 | iOS/Android builds and documented sandbox script | Configured test catalog, owner/beneficiary users, Customer Center | Native/store evidence not yet collected |

Test support to add only when at least two suites need it:

- `app/test/subscriptions/fakes/fake_subscription_repository.dart`
- `app/test/subscriptions/fakes/recording_revenuecat_service.dart`
- `app/test/subscriptions/fakes/subscription_test_data.dart`
- `app/test/subscriptions/subscription_widget_harness.dart`

The widget harness should use production localization/theme,
`ProviderContainer.test`, `UncontrolledProviderScope`, `RecordingMessenger`, and
automatic disposal. Extend existing auth, settings, router, account-switcher,
and secret-scan suites instead of duplicating their established coverage.

## 10. Sequencing And Guardrails

- First TDD step: Extend
  `TestOwnerBillingStateUsesPrivateCamelCaseContract` so it fails because
  `subscriptionId` is absent, then make only the AppView type/query changes.
- Dependencies between work items:
  1. AppView response contract before Flutter owner models/correlation.
  2. Wire models and repository before providers/controllers.
  3. Crash-safe DID reservation, UUID completion, and RevenueCat configuration
      before any provider action.
  4. Guard and eligibility helpers before paywall presentation.
  5. Reconciliation state machine before purchase/restore controllers.
  6. Assignment and management flows before the complete owner page.
  7. Route/switcher/UI integration after provider behavior is green.
  8. Full static/build/manual evidence last.
- Guardrails:
  - Never expose OAuth/PDS credentials or RevenueCat secret keys to Flutter.
  - Never use RevenueCat entitlements or `CustomerInfo` as DID authorization.
  - Never call RevenueCat `logOut` in this slice.
  - Never call owner ensure before durable DID reservation, for another DID, or
    after the reservation contains a UUID.
  - Never call RevenueCat identity work before the returned UUID is durable.
  - Never correlate owner arrays by order or raw provider identifier.
  - Never retry checkout after a successful provider result; retry only AppView
    reconciliation/refresh.
  - Never auto-assign on purchase or restore.
  - Never publish state/navigation after an owner/account lease changes.
  - Never log DTOs containing DID, UUID, provider, transaction, or receipt values;
    never forward raw RevenueCat SDK logs or raw platform exception payloads.
  - Do not add a migration, sqlc setup, lexicon change, web billing, feature gate,
    or generic RevenueCat identity switcher.
- Focused commands during TDD:
  - `just app-test test/subscriptions`
  - `just app-test test/auth/models/session_registry_test.dart`
  - `just app-test test/router/subscription_routes_test.dart`
  - `just app-test test/router/app_shell_account_switcher_test.dart`
  - `just app-analyze`
  - `just test` with the dev stack running
  - `just appview-check` for release-equivalent AppView evidence
- Final automated/build gates:
  - `just app-test`
  - `just app-analyze`
  - `just app-build-ios staging`
  - `just app-build-apk staging`
  - `just app-build-web staging`

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking production | Real RevenueCat iOS/Android public keys, products, offerings, attached paywalls with verified close affordances, and Customer Center configuration are not available in source | Native sandbox/manual evidence cannot complete | Keep blank config examples and fake-based implementation; configure and verify every paywall version before activation |
| CPQ-002 | Non-blocking production | Original-owner and installation-pin recovery/reset remain deliberately undefined | A permanently unavailable owner leaves billing locked | Preserve the pin and safe refusal; design recovery separately before production |
| CPQ-003 | Non-blocking production | Wrong-owner restore behavior depends on RevenueCat keep-with-original configuration | That MAN-003 branch cannot yet pass/fail meaningfully | Leave explicitly blocked until provider and support prerequisites exist |
| CPQ-004 | Resolved in plan | SDK starts before an owner may be selected | Anonymous SDK identity could be mistaken for usable billing identity | Allow configuration only; guard every billing action and identify exact AppView UUID first |
| CPQ-005 | Resolved in plan | A non-anonymous mismatched SDK identity cannot safely switch without alias/logout risk | Potential receipt movement | Fail locked; do not logout or directly identify another UUID |
| CPQ-006 | Resolved in plan | Test spec's IT-013 path does not match existing AppView test organization | Could duplicate a database integration test | Extend existing registered-route contract test and retain IT-013 traceability in its test name/comments |
| CPQ-007 | Non-blocking implementation | RevenueCat package resolution may update platform-generated files or minimum requirements | Build churn may exceed the intended source changes | Add packages through Pub, inspect generated diff, retain only expected output, and prove all three Flutter builds |
| CPQ-008 | Non-blocking UI | Final merchandising wording and visual polish are not fixed | Widget copy may evolve without changing behavior | Implement required semantics/localization first; defer visual polish to the approved polish stage |
| CPQ-009 | Resolved in plan | Process interruption during first owner setup can occur between local persistence, AppView ensure, and RevenueCat identity | Another DID could otherwise claim an installation whose backend identity already exists | Persist DID before PUT and UUID before RevenueCat work; allow only same-DID PUT retry while UUID is absent, then use GET-only recovery |
| CPQ-010 | Resolved in plan | RevenueCat SDK cannot reliably preflight every attached paywall/template failure | Flutter could promise pre-presentation unavailability it cannot observe | Preflight only named offering/package; map attachment/template failures from presentation and verify close affordances manually |
| CPQ-011 | Resolved in plan | Raw native SDK logs and platform exception payloads may contain sensitive provider/account data | Release logs or Sentry could leak billing relationships | Set an explicit release-safe log level, install no raw forwarding handler, and retain only sanitized error classifications |

No open question blocks TDD implementation. Production activation remains
blocked by CPQ-001 through CPQ-003 and required native evidence.

## 12. Handoff To TDD Builder

- Coding plan:
  `docs/changes/2026-09-10-flutter-subscriptions/04-coding-plan.md`
- Read first: approved `01-requirements.md`, `02-acceptance-tests.md`, and
  `03-document-review.md`; preserve stable requirement/test IDs in test names or
  comments when consolidating physical files.
- Start red-green with IT-013 in the existing AppView route test, then UT-001 /
  UT-012 Flutter wire models and IT-001 API client.
- Create or update `05-implementation-plan.md` through the `implement-tdd`
  workflow before production edits beyond the first failing test.
- Run focused tests after every behavior slice and the full gates only after all
  focused suites are green.
- Do not claim production readiness without MAN-001 through MAN-005 evidence;
  retain the documented block on wrong-owner MAN-003 until prerequisites exist.
