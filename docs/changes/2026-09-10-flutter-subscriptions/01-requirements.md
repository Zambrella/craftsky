# Requirements: Flutter Account Subscriptions

## 1. Initial Request

Now that the AppView account-subscription foundation is implemented, add the
Flutter portion of the subscription experience. This slice should complete the
subscription flow end to end while deliberately leaving paid-feature enforcement
for later work.

## 2. Current Codebase Findings

- Relevant files:
  - `adr/014-account-assigned-subscription-licenses.md` separates the private
    billing owner from the DID receiving a license.
  - `docs/changes/2026-09-07-account-subscriptions/01-requirements.md` and the
    implemented `appview/internal/api/subscriptions.go` define the AppView
    billing, access, reconciliation, and assignment contracts.
  - `app/lib/auth/models/session_registry.dart` retains up to five independent
    account sessions and one active account on an installation.
  - `app/lib/settings/pages/settings_page.dart`,
    `app/lib/auth/widgets/account_switcher_content.dart`, and
    `app/lib/router/router.dart` are the established settings, account-switching,
    and typed-routing surfaces.
  - `app/lib/shared/api/providers/dio_provider.dart` and the shared API error
    mapping are the established AppView transport boundary.
  - `app/lib/bootstrap.dart` separates native initialization from web startup and
    requires fallible initialization to have explicit loading/error handling.
  - `app/pubspec.yaml` does not yet include `purchases_flutter` or
    `purchases_ui_flutter`.
- Existing patterns:
  - Flutter uses Riverpod providers, repository/API-client boundaries,
    `dart_mappable` models, typed `go_router` routes, generated localization, and
    account-boundary invalidation.
  - AppView session requests are scoped to a DID and device. An inactive retained
    account has its own session token but must not be confused with the active
    account.
  - User-facing failures use the shared error mapper and messenger rather than
    exposing transport or provider errors directly.
- Current behavior:
  - Every Flutter account appears free because the app does not read
    `GET /v1/subscriptions/access`.
  - There is no billing-owner selection, RevenueCat identity, paywall, purchase,
    restore, license assignment, subscription management, or tier badge UI.
  - The owner billing response returns subscriptions and licenses as separate
    arrays, but a license does not currently expose the internal subscription ID
    needed to correlate those rows safely.
- Constraints discovered:
  - AppView, not RevenueCat `CustomerInfo`, is authoritative for a DID's effective
    tier and future paid-feature authorization.
  - RevenueCat must use the billing owner's AppView-issued opaque UUID, not an
    active DID, handle, device ID, email, or anonymous customer.
  - Ordinary CraftSky account switching must not switch or merge the RevenueCat
    billing customer.
  - The launch catalog has separate Plus and Business offerings/products so one
    payer can own both and assign them to different DIDs.
  - RevenueCatUI paywalls and Customer Center support native iOS and Android, not
    the app's web or desktop targets. Current RevenueCatUI requires iOS 15 or
    later and Android API 21 or later.
  - The Xcode project already targets iOS 15. Android inherits the Flutter SDK
    minimum, and this project requires Flutter 3.44 or later, satisfying the
    RevenueCatUI Android API 21 minimum.
  - RevenueCatUI owns purchases initiated from its paywall; Flutter must not issue
    a second manual purchase around that UI.
  - AppView creates a newly reconciled license as unassigned. Restore never
    implies beneficiary intent.
- Test/build commands discovered:
  - `just app-analyze`
  - `just app-test`
  - `just app-build-ios`
  - `just app-build-apk`
  - `just app-run-ios`
  - `just app-run-android`

## 3. Clarifying Questions And Decisions

### Q1: What is included in this Flutter slice?

Answer: Implement most subscription work and make the subscription flow
completable end to end, apart from locking paid features behind the paywall.

Decision / implication: The slice includes SDK setup, billing-owner identity,
tier-specific paywalls, purchase and restore completion, AppView reconciliation,
explicit license assignment, tier visibility, and provider management. It does
not add authorization checks to existing Plus or Business capabilities.

### Q2: Which state grants account-level access?

Answer: AppView effective access, as established by ADR 014 and the completed
AppView requirements.

Decision / implication: RevenueCat purchase results trigger synchronization but
never directly grant a DID a tier. Flutter displays and later gates from
`GET /v1/subscriptions/access`.

### Q3: How is a successful purchase assigned?

Answer: Explicitly after AppView has reconciled the purchase.

Decision / implication: A completed purchase is not presented as fully activated
until the resulting license is visible and the owner assigns it to an eligible
retained account. A purchase must not be silently assigned to the active or
billing-owner DID.

### Q4: How does RevenueCat identity interact with multi-account switching?

Answer: RevenueCat represents the selected billing account, not the active DID.

Decision / implication: Billing setup explicitly selects the current DID as the
billing owner. That owner's AppView-issued UUID remains the RevenueCat App User
ID through ordinary switches among retained accounts. Billing mutations require
the owner account to be active; beneficiary accounts may see only their own
effective access.

### Q5: Which platforms complete purchases in this slice?

Answer: The native-store launch direction covers iOS and Android.

Decision / implication: Web and desktop continue to display account access where
supported by AppView, but do not initialize native RevenueCatUI or offer purchase,
restore, or Customer Center actions.

### Q6: What happens when the selected billing-owner session is removed?

Answer: Pin the original billing owner.

Decision / implication: Flutter retains a disabled private binding to the
original owner DID and RevenueCat UUID, does not call RevenueCat `logOut`, and
does not permit another DID to become billing owner. Billing resumes only after
the same DID is reauthenticated and AppView returns the same UUID. Owner transfer,
identity reset, and post-deletion recovery remain separate production work.

### Q7: How will Flutter correlate owner-visible subscriptions and licenses?

Answer: Add a narrow AppView response enhancement.

Decision / implication: Each `BillingLicense` returned by owner billing routes
will include the internal `subscriptionId` of its provider subscription. Flutter
may join rows only on that field, never by array order or provider identifiers.
No persistence, reconciliation, authorization, webhook, or assignment behavior
changes.

### Q8: How will Flutter correlate an owner-triggered reconciliation?

Answer: Keep the existing reconciliation route and use a client baseline
algorithm.

Decision / implication: Flutter reads a baseline `requestedGeneration`, sends the
reconciliation request, observes the first later owner state whose
`requestedGeneration` exceeds the baseline, and treats that value as the target.
It waits until `reconciledGeneration` reaches at least that target and then
evaluates the expected billing state. Concurrent webhook or scheduled requests
may advance generations further but cannot cause premature activation.

### Q9: When is first-time owner selection made durable?

Answer: Reserve the confirming DID before the first network call.

Decision / implication: Explicit confirmation first persists an installation
owner reservation containing the DID and no UUID. Only that DID may resume setup.
The same DID may retry idempotent AppView ensure until a UUID is returned and
durably added to the reservation. RevenueCat identity work occurs only after UUID
persistence. A failure or crash leaves the same owner reserved and retryable; it
never reopens setup to another DID.

## 4. Candidate Approaches

### Option A: AppView-Authoritative Native Billing Coordinator

Summary: Keep one explicitly selected billing owner, identify RevenueCat with its
AppView UUID, use offering-specific RevenueCatUI paywalls and Customer Center,
then reconcile and assign through AppView.

Pros:

- Preserves the payer/beneficiary split and the implemented AppView contract.
- Supports one Plus and one Business license under one store payer.
- Reuses provider-owned purchase and management UI.
- Gives the user a deterministic path from purchase through assignment.

Cons:

- Purchase completion requires an asynchronous reconciliation wait before
  assignment.
- Billing management requires switching back to the billing-owner account.
- Native integration and sandbox verification are needed on both stores.

Risks:

- A wrong RevenueCat identity can attach a receipt to the wrong billing account.
- Provider or reconciliation delays can leave a successful purchase temporarily
  unassignable.

### Option B: Active-DID RevenueCat Identity And Entitlement Gating

Summary: Log RevenueCat into whichever DID is active and use `CustomerInfo`
entitlements directly.

Pros:

- Conventional single-account RevenueCat integration.
- Immediate client-side entitlement result after purchase.

Cons:

- Cannot correctly express one payer assigning Plus and Business to different
  DIDs.
- Conflicts with ADR 014 and AppView's authority.

Risks:

- Purchases may move, merge, or appear on the wrong CraftSky account.

### Option C: Custom Store And Subscription UI

Summary: Fetch offerings and build custom paywall, purchase, restore, and
management interfaces around lower-level RevenueCat APIs.

Pros:

- Maximum presentation control.

Cons:

- Duplicates dashboard-configurable RevenueCatUI behavior and increases billing
  UI scope.

Risks:

- More platform-specific purchase and management edge cases become app-owned.

## 5. Recommended Direction

Recommended approach: Option A, the AppView-authoritative native billing
coordinator.

Why: It is the only approach that completes native purchases without violating
the accepted separation between the payer and beneficiary. RevenueCat remains
responsible for store transactions and management UI, AppView remains responsible
for billing identity and DID assignment, and Flutter coordinates those systems
without inventing a second access model. One narrow owner-response field links a
license to its internal subscription row; the existing reconciliation mutation
contract remains unchanged.

## 6. Problem / Opportunity

The server can now represent account-assigned subscriptions, but members cannot
buy, restore, assign, view, or manage them from the app. Flutter must bridge the
native store and RevenueCat lifecycle to AppView's private billing model while
remaining safe under multiple retained CraftSky accounts.

## 7. Goals

- G-001: Let an authenticated billing owner complete a Plus or Business sandbox
  or production purchase on supported native platforms.
- G-002: Convert a successful purchase or restore into visible AppView billing
  state and an explicit DID assignment.
- G-003: Keep RevenueCat billing identity independent of ordinary active-account
  switching.
- G-004: Show each account's AppView-authoritative effective tier without using
  RevenueCat entitlements as account authorization.
- G-005: Let the billing owner restore and manage provider subscriptions.
- G-006: Handle cancellation, delay, failure, unsupported platforms, and account
  changes without false access or cross-account leakage.

## 8. Non-Goals

- NG-001: Locking or unlocking any existing feature based on paid tier.
- NG-002: Defining the final Plus or Business benefit list, usage limits, or
  expired-data behavior.
- NG-003: Modifying AppView reconciliation behavior, database schema, webhook
  handling, assignment policy, authorization, or account-deletion policy. The
  sole AppView change in scope is adding `subscriptionId` to each owner-visible
  license response.
- NG-004: Creating or changing RevenueCat projects, apps, products,
  entitlements, offerings, packages, paywall templates, Customer Center
  configuration, prices, trials, or store listings.
- NG-005: Web checkout, desktop purchases, annual plans, plan replacement,
  bundles, multiple same-tier licenses, or billing-owner transfer/recovery.
- NG-006: Remote assignment invitations or assignment to an account that is not
  currently authenticated on the installation.
- NG-007: Inferring assignment from the active account, the store account, a
  RevenueCat entitlement, or a restored receipt.
- NG-008: PDS, Lexicon, Tap, feed, ranking, search, moderation, reach, or public
  record changes.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Billing owner | Authenticated DID that explicitly owns the private billing account. | Purchase, restore, reconcile, assign, inspect, and manage subscriptions. |
| Assigned account | Retained authenticated DID receiving a Plus or Business license. | Correct tier display without payer or provider details. |
| Free account | DID with no accessible assignment. | Unchanged free behavior and clear optional upgrade entry points. |
| RevenueCat | Native purchase and subscription lifecycle provider. | Stable AppView-issued billing UUID and platform public SDK key. |
| AppView | Authority for billing ownership, assignment, and effective tier. | Authenticated, device-scoped requests and explicit synchronization triggers. |
| Apple App Store / Google Play | Native payment provider. | Correct products, sandbox account, and platform configuration. |

## 10. Current Behavior

The Flutter app has no subscription domain or RevenueCat SDK. Settings and the
account switcher do not show tiers. No UI calls the implemented AppView billing
routes, and there is no native purchase, restore, assignment, or management path.

## 11. Desired Behavior

On iOS or Android, a signed-in member opens a subscription settings page. The
page reads the active DID's effective access. If no installation billing owner is
reserved, the member can explicitly make the current account the billing owner.
Flutter first persists that DID as the permanent installation reservation, then
calls `PUT /v1/billing/account`, persists the returned stable RevenueCat App User
ID into the reservation, and only then identifies the native SDK with that UUID
before enabling billing actions.

The owner can choose Plus or Business. Flutter loads and presents that tier's
configured RevenueCat offering. After purchase or restore succeeds, Flutter asks
AppView to reconcile and performs a baseline-generation bounded refresh until the
resulting billing state is current or the wait times out. A genuinely new license
remains unassigned, and the owner explicitly selects one eligible retained
account and confirms assignment. If the same provider subscription regains
access, its existing license and assignment are preserved; an unassigned restored
license still requires explicit assignment. The app then refreshes the assigned
account's AppView access and shows its effective tier.

The owner page displays both owned subscriptions, assignment state, safe provider
status, anomalies, and stale reconciliation state. Provider management opens the
RevenueCat Customer Center under the same billing UUID. Returning from a provider
operation requests another AppView reconciliation. Switching to a beneficiary or
free account does not change RevenueCat identity and does not expose owner-only
billing information; management requires switching to the retained owner account.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | A billing owner shall be able to complete the supported native Plus or Business subscription flow from product selection through explicit assignment to an eligible CraftSky DID. | Delivers the requested end-to-end outcome. | Prompt | AC-001 |
| BR-002 | Business | Must | Paid account state shall follow AppView's assigned DID and shall not follow the installation, active DID, handle, or combined RevenueCat customer entitlements. | Prevents cross-account access. | ADR 014; AppView requirements | AC-002, AC-003 |
| BR-003 | Business | Must | Free accounts and existing feature behavior shall remain unchanged by this slice. | Feature gating is explicitly deferred. | Prompt; direction | AC-004 |
| BR-004 | Business | Must | The billing owner shall be able to restore and manage provider subscriptions without receiving implied assignment or client-authoritative access. | Covers normal billing lifecycle operations safely. | Prompt; discovery | AC-005 |
| FR-001 | Functional | Must | The Flutter app shall integrate the supported `purchases_flutter` and `purchases_ui_flutter` SDKs for iOS and Android and load platform-specific public SDK keys from the existing environment configuration mechanism. | Enables native RevenueCat flows without embedding secret server credentials. | Codebase; RevenueCat docs | AC-006 |
| FR-002 | Functional | Must | Missing, blank, or unsupported-platform RevenueCat configuration shall disable purchase, restore, and management actions without preventing sign-in, social use, or AppView access reads. | Billing configuration must fail safely and remain optional to free use. | Codebase; scope decision | AC-004, AC-006 |
| FR-003 | Functional | Must | On first explicit owner confirmation, Flutter shall durably reserve the active DID before any AppView or RevenueCat call. While that reservation has no UUID, only the same active DID may call idempotent `PUT /v1/billing/account`; the returned UUID shall be persisted into the reservation before RevenueCat identity work. Once the UUID exists, startup, sign-in, access reads, switching, and recovery shall never call ensure and may use only `GET /v1/billing/account`. | Makes permanent owner selection crash-safe across AppView, RevenueCat, and local storage. | AppView FR-001; decisions Q4, Q6, and Q9; DR-010 | AC-007, AC-011 |
| FR-004 | Functional | Must | Before any purchase, restore, or Customer Center operation, Flutter shall obtain the owner state from AppView and configure or identify RevenueCat with exactly its `revenueCatAppUserId`. | Maintains the accepted payer identity. | ADR 014; AppView RULE-008 | AC-008 |
| FR-005 | Functional | Must | Flutter shall never use a DID, handle, email, device ID, session token, locally generated UUID, or anonymous RevenueCat ID as the billing App User ID and shall prohibit anonymous purchase and restore. | Avoids identity leakage and receipt misassociation. | ADR 014; RevenueCat identity guidance | AC-008 |
| FR-006 | Functional | Must | The selected billing owner DID, plus the RevenueCat UUID once known, shall remain pinned across setup failures, app restarts, ordinary account switches, and owner-session loss; none of those events shall call RevenueCat `logOut` or identify RevenueCat as another DID or UUID. | Active account and payer are independent concepts. | ADR 014; decisions Q4, Q6, and Q9 | AC-003, AC-009, AC-011 |
| FR-007 | Functional | Must | Billing mutations and owner details shall be available only while the retained billing-owner account is active; another active account shall be offered a switch-to-owner action and shall receive no owner billing details. | AppView owner routes authenticate the owner DID. | AppView API; privacy boundary | AC-009, AC-010 |
| FR-008 | Functional | Must | If the reserved owner's session is removed, deleted, invalid, or unavailable, Flutter shall retain the reserved DID and optional UUID as an unusable private binding, disable billing operations, reject every other DID, and ignore stale completions. Reauthenticated same-DID setup may retry PUT only while UUID is absent. Once UUID exists, recovery is GET-only: a match resumes billing, while 404 or mismatch retains the lock without ensure, RevenueCat identity mutation, or replacement setup. | Prevents stale or accidental payer changes, including at every crash boundary and after AppView account closure. | ADR 014; decisions Q6 and Q9; DR-010 | AC-011 |
| FR-009 | Functional | Must | The subscription page shall read `GET /v1/subscriptions/access` for the active account and represent only the wire tiers `free`, `plus`, and `business`, including an inaccessible dormant assignment as effective `free` with its assigned tier where returned. | Mirrors AppView's canonical access projection. | Implemented AppView API | AC-002, AC-012 |
| FR-010 | Functional | Must | The account switcher and account-facing settings shall display each retained account's latest AppView-authoritative effective tier, with loading or unavailable state that does not reuse another account's tier. | Makes multi-account assignment understandable. | Direction section 7; prompt | AC-013 |
| FR-011 | Functional | Must | Access and billing data loaded for an inactive retained account shall use that account's own current session scope and shall not change the active account or route as a side effect. | Prevents account-boundary leakage. | Multi-account architecture | AC-013 |
| FR-012 | Functional | Must | The owner billing page shall call `GET /v1/billing/account`, correlate each license to its subscription only by `license.subscriptionId == subscription.id`, and display owned Plus and Business license state including assignment, access, safe status/timing, assignability, anomaly, and reconciliation staleness. It shall never correlate rows by array order. | Store UIs cannot show CraftSky assignment context. | User answer Q7; ADR 014 | AC-014, AC-029 |
| FR-013 | Functional | Must | Purchase eligibility shall be evaluated per correlated tier from a fresh, fully reconciled owner state. Purchase shall be blocked while any correlated subscription gives access, has pending payment, lacks exact `autoRenewalStatus='will_not_renew'`, has a subscription/license anomaly, or while reconciliation is stale or pending. A tier with no subscription is eligible. A tier whose correlated subscriptions are all inaccessible, non-pending, exactly `will_not_renew`, and anomaly-free is eligible for post-lapse repurchase; any dormant assignment is preserved and may need explicit unassignment before the new license can target the same DID. | Prevents duplicate payment without permanently blocking normal repurchase. | Direction; AppView FR-026 and FR-029; DR-001 | AC-015 |
| FR-014 | Functional | Must | Selecting Plus or Business shall fetch the matching named RevenueCat offering, reject a missing offering or missing required package before presentation, and otherwise present its dashboard paywall directly. Flutter shall not use a global current offering or entitlement-based `presentPaywallIfNeeded`. A missing/invalid attached paywall discovered only during presentation is a provider error, not pre-presentation unavailability. Dashboard activation shall verify a working dismiss affordance; `displayCloseButton` alone is not sufficient for every paywall version. | Uses only availability facts exposed by the SDK while preserving independent account licenses. | Activation checklist; RevenueCatUI docs; DR-011; coding-plan review | AC-016, AC-022 |
| FR-015 | Functional | Must | RevenueCatUI shall own the transaction initiated from its directly presented paywall; Flutter shall not issue a duplicate manual purchase and shall distinguish the direct results purchased, restored, cancelled, and error. | Prevents duplicate transactions and excludes `notPresented`, which belongs to `presentPaywallIfNeeded`. | RevenueCatUI docs; DR-011 | AC-016, AC-017 |
| FR-016 | Functional | Must | A cancelled or dismissed paywall shall leave AppView billing and assignment state unchanged and return the user to a usable owner page without an error success message. | Cancellation is normal user behavior. | RevenueCatUI docs | AC-017 |
| FR-017 | Functional | Must | After a purchased or restored result, Flutter shall read the owner state's baseline `requestedGeneration`, request `POST /v1/billing/reconciliation`, and perform bounded, cancel-safe refreshes. The first observed `requestedGeneration` greater than baseline becomes the minimum target. Flutter shall not evaluate completion before `reconciledGeneration >= target`; purchase completes only when a recognized expected state also appears, while restore may complete with no state change after the target reconciles. Timeout or failure remains pending with retry. | Provider completion and AppView visibility are asynchronous, and other reconciliation triggers may race. | Implemented AppView API; user answer Q8 | AC-001, AC-018 |
| FR-018 | Functional | Must | Flutter shall not report account activation from RevenueCat `CustomerInfo`. A new unassigned license requires successful assignment and matching self-access. When the same correlated subscription regains access, its existing assignment shall be preserved and activation completes only after that DID's self-access reflects the effective tier; if its license is unassigned, explicit assignment is still required. | Preserves AppView authority and same-subscription recovery. | ADR 014; AppView FR-029; decision Q2 | AC-001, AC-002 |
| FR-019 | Functional | Must | A newly reconciled unassigned license shall prompt the owner to select from eligible retained accounts and explicitly confirm the tier and target account before calling the assignment endpoint. | Assignment must express beneficiary intent. | AppView FR-009 through FR-011 | AC-001, AC-019 |
| FR-020 | Functional | Must | Assignment shall call `PUT /v1/billing/licenses/{licenseId}/assignment` with the selected DID and distinguish the exact AppView errors `billing_license_not_found`, `assignment_target_ineligible`, `assignment_conflict`, and `assignment_cooldown`; target-session/device races use `assignment_target_ineligible`, while ordinary owner `401` uses session invalidation. Assignment and unassignment failures shall not replace refreshed server truth with optimistic state. | Aligns Flutter handling with AppView's implemented errors. | Implemented AppView API; DR-007; DR-013 | AC-019, AC-020 |
| FR-021 | Functional | Must | The owner shall be able to unassign a license with explicit confirmation through `DELETE /v1/billing/licenses/{licenseId}/assignment`; reassignment shall explain the seven-day target-change cooldown before confirmation. | Makes assignment management complete and avoids surprising loss of access. | AppView FR-030 | AC-020 |
| FR-022 | Functional | Must | Restore shall be an explicit owner-only action under the existing billing UUID; after RevenueCat restore completes, Flutter shall use the same reconciliation and explicit-assignment flow as purchase, preserving known assignments and leaving new licenses unassigned. | Restore proves provider ownership, not beneficiary intent. | AppView FR-032; RevenueCat docs | AC-005, AC-018 |
| FR-023 | Functional | Must | Provider management shall open RevenueCat Customer Center under the owner UUID; restore or management callbacks, dismissal after a management action, and return from an external store-management destination shall request AppView reconciliation and refresh owner state. | Keeps AppView synchronized after provider-side changes. | RevenueCat Customer Center docs | AC-005, AC-021 |
| FR-024 | Functional | Must | Provider failures, reconciliation delay, stale state, unsupported offerings, and assignment anomalies shall have distinct localized loading, empty, retry, and support-oriented states and shall never be shown as successful activation. | Billing errors can otherwise mislead users after a charge. | Prompt; codebase messaging pattern | AC-018, AC-022 |
| FR-025 | Functional | Should | The subscription surface should explain that Plus and Business are independently billed account licenses, identify the current billing owner, show the assigned account, and state that switching accounts does not move a subscription. | The native store cannot communicate CraftSky assignment. | Direction; ADR 014 | AC-014 |
| FR-026 | Functional | Must | Account-level subscription providers and cached projections shall be invalidated or keyed by account lease so switching, removal, reauthentication, or deletion cannot display or mutate stale account data. Each owner-only flow shall require the exact reserved owner to be active at entry and revalidate its captured lease immediately before: owner billing GET/PUT; offering retrieval; RevenueCat identity inspection/configuration; direct paywall, restore, or Customer Center presentation; reconciliation POST and every polling GET; assignment; and unassignment. First setup shall revalidate before durable DID reservation and before ensure. A stale lease stops dispatch/polling and fences completion. | Defines the complete owner-operation boundary without affecting beneficiary self-access reads. | Codebase; DR-006; DR-012; decision Q9 | AC-010, AC-011, AC-013, AC-025 |
| FR-027 | Functional | Must | AppView owner billing responses shall add `subscriptionId` to every `BillingLicense`, referencing the internal owner-visible `BillingSubscription.id`; the field shall be present on ensure/read responses and shall not expose the provider subscription identifier. | Enables safe subscription/license correlation without changing billing behavior. | User answer Q7; DR-003 | AC-029 |
| NFR-001 | Non-functional | Must | Subscription models and AppView requests shall follow existing camelCase JSON, strict shape/type parsing, session/device authentication, API exception mapping, repository, and Riverpod conventions. Tier values shall be closed to `free`, `plus`, and `business`; unknown provider-owned status, store, renewal, or anomaly strings shall use a safe fallback presentation rather than fail the complete owner response. | Maintains the app architecture while tolerating additive provider state. | Codebase; API architecture; DR-008 | AC-023 |
| NFR-002 | Non-functional | Must | RevenueCat secret API keys, AppView session tokens, receipts, payment details, full `CustomerInfo`, billing UUIDs, DIDs, and provider identifiers shall not be logged, reported to Sentry as values, or displayed outside their authorized UI purpose. Flutter shall set an explicit release-safe RevenueCat log policy, shall not install a raw SDK log-forwarding handler, and shall map provider exceptions without forwarding raw messages/details. | Billing and account relationships are sensitive, including in native SDK diagnostics. | ADR 014; privacy requirements; coding-plan review | AC-024 |
| NFR-003 | Non-functional | Must | RevenueCat public SDK keys shall be platform-specific configuration values and shall not be treated as AppView server credentials or reused across the wrong app/store. | Prevents configuration confusion. | RevenueCat SDK guidance | AC-006 |
| NFR-004 | Non-functional | Must | Purchase, restore, assignment, and management actions shall prevent duplicate submission while in progress and remain safe if the page is dismissed, the app backgrounds, or the active account changes. | Native provider flows are asynchronous. | Mobile lifecycle risk | AC-020, AC-025 |
| NFR-005 | Non-functional | Must | Subscription UI and controls shall support screen readers, text scaling, platform back/dismiss behavior, and narrow mobile layouts without clipped critical status or actions. | Billing decisions require accessible, legible UI. | Existing Flutter quality standard | AC-026 |
| NFR-006 | Non-functional | Must | Unit and widget tests shall use replaceable RevenueCat and AppView boundaries and shall not require live purchases, network access, or real SDK initialization. | Keeps automated tests deterministic. | Codebase test pattern | AC-027 |
| NFR-007 | Non-functional | Must | Release verification shall include Flutter analysis, automated subscription tests, iOS and Android builds, and manual sandbox purchase/restore/management checks on both supported platforms before production activation. | Billing correctness cannot be established from mocks alone. | RevenueCat verification guidance | AC-027, AC-028 |
| NFR-008 | Non-functional | Should | Reconciliation polling shall use a documented bounded timeout and avoid unbounded background work or repeated provider calls after disposal. | Protects battery, bandwidth, and provider capacity. | Mobile lifecycle risk | AC-018, AC-025 |
| RULE-001 | Business rule | Must | Only the AppView `effectiveTier` and `givesAccess` response may represent a DID's current paid access; RevenueCat entitlements represent billing-customer purchases only. | Separates payer state from beneficiary authorization. | ADR 014 | AC-002 |
| RULE-002 | Business rule | Must | One installation shall reserve at most one billing-owner DID. The reservation becomes permanent immediately after explicit active-owner confirmation, before ensure; setup failure or owner-session loss shall not make the slot available to another DID. The AppView UUID is added when known. | One store receipt must remain with one private billing identity across crashes. | Direction; decisions Q4, Q6, and Q9 | AC-007, AC-009, AC-011 |
| RULE-003 | Business rule | Must | The billing owner need not be assigned either owned license. | Payer and beneficiary are separate. | ADR 014 | AC-019 |
| RULE-004 | Business rule | Must | Plus and Business may be owned concurrently and assigned to different DIDs, but the UI shall not offer both assignments to one DID. | Business includes Plus and AppView rejects overlap. | Direction; AppView RULE-003 | AC-001, AC-019 |
| RULE-005 | Business rule | Must | Purchase and restore shall never auto-assign a license. | Store ownership does not prove beneficiary intent. | AppView requirements | AC-001, AC-005 |
| RULE-006 | Business rule | Must | A successful store transaction with delayed, failed, or anomalous AppView reconciliation is a pending billing state, not successful CraftSky account activation. | Prevents false access claims after charge. | Authority boundary | AC-018, AC-022 |
| RULE-007 | Business rule | Must | Web and desktop shall not expose native purchase, restore, or Customer Center actions in this slice. | RevenueCatUI is unsupported and web billing is deferred. | Decision Q5 | AC-006 |
| RULE-008 | Business rule | Must | Ordinary CraftSky sign-out or switching of a beneficiary shall not move purchases to another RevenueCat customer. Loss or deletion of the owner session disables billing but retains the reserved DID and optional UUID; Flutter shall neither log RevenueCat out nor permit a replacement owner in this slice. | Restore behavior is configured to keep the original App User ID. | ADR 014; decisions Q6 and Q9 | AC-009, AC-011 |
| RULE-009 | Business rule | Must | This slice shall not change availability or behavior of existing features based on `free`, `plus`, or `business`. | Feature locking is explicitly excluded. | Prompt | AC-004 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-017, FR-018, FR-019, RULE-004, RULE-005 | Given an identified owner purchases an eligible Plus or Business offering, baseline-correlated AppView reconciliation produces one of two valid outcomes: a genuinely new license remains unassigned until explicit assignment and matching self-access, or the same subscription regains access while preserving its existing assignment and completes only when that DID's self-access reflects the tier. An unassigned recovered license still requires explicit assignment. |
| AC-002 | BR-002, FR-009, FR-018, RULE-001 | Given RevenueCat reports purchases for the billing customer, a DID's displayed access changes only when AppView self-access reports the corresponding effective tier and access. |
| AC-003 | BR-002, FR-006 | Given RevenueCat is identified as billing owner A, when the active CraftSky account switches to B and back, then RevenueCat remains identified as A and no purchase state is applied directly to B. |
| AC-004 | BR-003, FR-002, RULE-009 | With billing unavailable, unconfigured, or with any tier state, every existing feature remains reachable and behaves as it did before this slice. |
| AC-005 | BR-004, FR-022, FR-023, RULE-005 | Restore and Customer Center run only under the owner UUID; afterward AppView reconciliation preserves known assignments and presents any newly discovered license as unassigned. |
| AC-006 | FR-001, FR-002, NFR-003, RULE-007 | Correct public keys initialize RevenueCat on supported iOS/Android builds; missing or mismatched configuration disables billing safely, and web/desktop initializes no native billing UI. |
| AC-007 | FR-003, RULE-002 | Startup, sign-in, access reads, and switching create no billing account. First explicit confirmation durably reserves the active DID before one ensure. A crash/failure before UUID persistence permits only that DID to retry ensure; a returned UUID is persisted before RevenueCat identity work, and another DID can never claim the slot. |
| AC-008 | FR-004, FR-005 | Every purchase, restore, or Customer Center call observes `Purchases.appUserID` equal to AppView's owner UUID, never an anonymous ID or CraftSky identifier. |
| AC-009 | FR-006, FR-007, RULE-002, RULE-008 | Restart and beneficiary switching preserve the selected owner while retained; beneficiary UI exposes no management action except switching to that owner, and switching does not call RevenueCat identity mutation. |
| AC-010 | FR-007, FR-026 | A beneficiary can read only its self-access; owner subscriptions, UUID, products, provider state, assignments to other DIDs, and management actions are absent. |
| AC-011 | FR-003, FR-006, FR-008, FR-026, RULE-002, RULE-008 | Setup failure or owner-session loss disables billing and stale work while retaining the reserved DID and optional UUID. A different DID cannot ensure, select, or identify. The same DID may retry ensure only before UUID persistence; afterward recovery performs GET only, where exact UUID resumes and 404/mismatch stays locked with zero ensure/logout/reidentification calls. |
| AC-012 | FR-009 | Free, accessible Plus, accessible Business, and dormant assigned-license responses render the exact AppView effective and assigned tier semantics. |
| AC-013 | FR-010, FR-011, FR-026 | Each retained account row displays only its own effective tier or its own loading/unavailable state, and refreshing inactive rows neither activates them nor changes routing. |
| AC-014 | FR-012, FR-025 | The owner page joins subscriptions and licenses only through `subscriptionId`, distinguishes Plus and Business ownership, provider/access status, assignment, staleness, and anomalies, and clearly identifies the billing owner and assigned account. |
| AC-015 | FR-013 | A fresh tier with no subscription is purchasable. A fresh correlated tier is also purchasable when every prior subscription is inaccessible, non-pending, exactly `will_not_renew`, and anomaly-free. Accessible, pending-payment, renewable/unknown renewal, stale/pending reconciliation, and anomalous states cannot launch purchase. A dormant assignment is preserved and disclosed before repurchase. |
| AC-016 | FR-014, FR-015 | Selecting each available tier presents its matching dashboard paywall with dismissal, and one interaction results in at most one RevenueCat purchase request. |
| AC-017 | FR-015, FR-016 | Paywall cancellation/dismissal returns to the owner page with unchanged assignment state, while provider errors show retryable localized failure rather than purchase success. |
| AC-018 | FR-017, FR-022, FR-024, NFR-008, RULE-006 | Purchase or restore captures a baseline, requests reconciliation, observes a requested generation above baseline, and waits for at least that generation to reconcile. Concurrent generation advances cannot cause premature activation. Purchase also requires a recognized new-license, same-subscription recovery, or bounded anomaly outcome; unchanged state remains pending, while restore may complete unchanged. Timeout or failure offers retry without claiming activation. |
| AC-019 | FR-019, FR-020, RULE-003, RULE-004 | Assignment requires explicit target selection and confirmation, permits owner or non-owner targets that AppView accepts, and does not offer a DID already holding another paid assignment. |
| AC-020 | FR-020, FR-021, NFR-004 | Assignment, reassignment, and unassignment serialize user actions and preserve refreshed server state on failure. Assignment maps exact `billing_license_not_found`, `assignment_target_ineligible`, `assignment_conflict`, and `assignment_cooldown`; unassignment maps `billing_license_not_found`; owner 401 remains ordinary session invalidation and generic server failures remain non-success. |
| AC-021 | FR-023 | Customer Center displays the identified owner's subscriptions; restore and management-return paths request AppView reconciliation and refresh the owner page. |
| AC-022 | FR-014, FR-024, RULE-006 | Missing named offering or required package fails before RevenueCatUI; attached-paywall/presentation error, other provider error, reconciliation delay, stale state, and anomaly each render a distinct non-success state with an appropriate retry, management, or support path. |
| AC-023 | NFR-001 | Subscription API fixtures round-trip the implemented camelCase contracts, map standard errors, and send the correct active or explicitly scoped retained-account session and device headers. |
| AC-024 | NFR-002 | Sensitive canaries are absent from CraftSky and forwarded native logs, Sentry event values, raw provider exception messages/details, generic errors, unauthorized UI, and persisted non-secret preferences; release configuration uses the approved RevenueCat log level and no raw SDK log forwarding. |
| AC-025 | FR-026, NFR-004, NFR-008 | Owner reservation writes, billing GET/PUT, offering retrieval, RevenueCat identity work, paywall, restore, Customer Center, reconciliation request/polls, assignment, and unassignment stop before their next external call when the exact reserved-owner lease is not current; dismissal, backgrounding, or switching also causes no duplicate action, stale navigation, unbounded poll, or cross-account publication. |
| AC-026 | NFR-005 | Subscription selection, status, assignment, retry, restore, and management remain operable with screen readers, large text, platform back behavior, and narrow supported screens. |
| AC-027 | NFR-006, NFR-007 | `just app-analyze` and `just app-test` pass with deterministic fakes covering identity, purchase outcomes, reconciliation, assignment, account boundaries, privacy, and lifecycle races. |
| AC-028 | NFR-007 | iOS and Android builds pass, and on each platform a sandbox owner can render both configured paywalls, complete one purchase, reconcile and assign it, restore, open management, and cancel/dismiss without purchase. |
| AC-029 | FR-012, FR-027 | Every owner billing response supplies each license's internal `subscriptionId`; it matches exactly one subscription in the same response, does not expose a RevenueCat subscription identifier, and supports correlation independent of array order. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | No billing owner is reserved. | Show self-access and an explicit owner-setup action; on confirmation reserve the DID durably before any network call and do not initialize an anonymous purchase flow. | FR-003, FR-005 |
| EC-002 | Active account is a beneficiary, not the billing owner. | Show only beneficiary access and a switch-to-owner action for management. | FR-007 |
| EC-003 | Owner buys while target account is inactive. | Keep owner active, reconcile, and permit explicit assignment using the retained target's DID. | FR-019 |
| EC-004 | Purchase succeeds but reconciliation times out. | Show a pending state and retry; do not repeat the purchase or claim activation. | FR-017, RULE-006 |
| EC-005 | App backgrounds during native purchase. | Resume from provider result or refresh owner state without issuing another purchase. | NFR-004 |
| EC-006 | User dismisses or cancels the paywall. | Return normally with no assignment or false error/success. | FR-016 |
| EC-007 | Named offering, required package, or attached paywall is missing/unusable. | Reject missing offering/package before RevenueCatUI. If attachment/template failure is discoverable only by presentation, map the thrown/error result to provider failure. Retain other app behavior. | FR-014, FR-024 |
| EC-008 | Restored receipt belongs to another RevenueCat App User ID under keep-with-original behavior. | Show a recovery/support failure, preserve the current owner identity, and do not transfer or create access. | FR-022, RULE-008 |
| EC-009 | AppView returns a duplicate or cross-tier anomaly. | Show bounded owner-visible anomaly state and do not offer purchase or assignment as a repair. | FR-013, FR-024 |
| EC-010 | Assignment or unassignment fails. | Map exact AppView error codes, refresh without changing active account or local assignment optimistically, and send owner `401` through normal session invalidation. | FR-020 |
| EC-011 | Existing assignment is inaccessible but retained. | Display effective free and assigned tier; do not treat the DID or license as unassigned. | FR-009 |
| EC-012 | Owner attempts reassignment inside seven days. | Explain cooldown and preserve current assignment. | FR-021 |
| EC-013 | Owner session is removed/deleted or setup crashes. | Ignore stale completion, retain the reserved DID/optional UUID without RevenueCat logout, and reject replacement setup. The same DID may resume PUT only with no UUID; after UUID persistence recovery is GET-only and 404/mismatch never ensures replacement. | FR-003, FR-008, NFR-004, RULE-008 |
| EC-014 | Web or desktop opens subscription settings. | Show effective access and an unavailable-native-purchase explanation without invoking RevenueCatUI. | FR-002, RULE-007 |
| EC-015 | Same provider subscription regains access. | Preserve its license and assignment; refresh the beneficiary's self-access instead of waiting for or creating a new license. | FR-017, FR-018 |
| EC-016 | Expired same-tier subscription is fully reconciled. | Permit repurchase only when inaccessible, non-pending, exactly `will_not_renew`, and anomaly-free; preserve any dormant assignment until explicitly removed. | FR-013 |
| EC-017 | Concurrent trigger advances reconciliation generation. | Wait for the observed post-baseline target and expected state; never activate from generation movement alone. | FR-017 |

## 15. Data / Persistence Impact

- New fields: A private installation-local owner reservation containing a DID
  and optional RevenueCat UUID may be added to the existing secure
  account/session snapshot or an equivalently protected store. The DID is
  persisted before ensure; the UUID is persisted before RevenueCat identity
  work; the reservation remains disabled after owner-session loss. Persist no RevenueCat
  `CustomerInfo`, receipt, transaction, or payment details.
- Changed fields: Account switcher presentation state gains an account-scoped
  effective-tier projection; this is cacheable display data, not authorization.
- Migration required: No AppView database migration. The owner API response adds
  `BillingLicense.subscriptionId`. Any local secure-storage schema change must
  retain strict versioned decoding and a safe migration path that does not clear
  an existing owner pin implicitly.
- Backwards compatibility: Existing installations begin with no selected billing
  owner and all existing sessions/features continue to work.

## 16. UI / API / CLI Impact

- UI: Add subscription entry points, owner setup, a subscription settings page,
  Plus and Business status/purchase cards, post-purchase assignment, tier badges,
  restore, unassign/reassign confirmation, pending/error states, and Customer
  Center launch.
- API: Consume existing `GET /v1/subscriptions/access`, `PUT` and `GET
  /v1/billing/account`, `POST /v1/billing/reconciliation`, and license assignment
  `PUT`/`DELETE` routes. Add `subscriptionId` to the `BillingLicense` objects
  returned by owner account routes; no route, request, or behavior changes.
- SDK: Add current compatible `purchases_flutter` and `purchases_ui_flutter`
  dependencies and required native platform configuration.
- CLI: None.
- Background jobs: No new Flutter background service. Bounded foreground refresh
  coordinates with AppView's existing reconciliation worker.

## 17. Security / Privacy / Permissions

- Authentication: Every AppView call uses the intended retained account's valid
  session and device scope. RevenueCat operations require the active billing
  owner and its AppView UUID.
- Authorization: AppView remains authoritative. Flutter owner UI is a convenience
  boundary and does not replace server owner checks.
- Sensitive data: Treat the billing UUID and payer/beneficiary relationship as
  private. Public SDK keys may ship in the app; RevenueCat secret keys may not.
- Abuse cases: Prevent duplicate taps, duplicate same-tier checkout, implicit
  assignment, assignment overlap, identity switching, and optimistic access.
- Store permissions: Purchases use the native store account. Flutter never
  receives AppView OAuth credentials, RevenueCat secret credentials, or raw
  receipts for custom validation.

## 18. Observability

- Events: Record bounded outcomes for SDK configuration, owner identification,
  paywall presentation/result, restore, reconciliation wait, assignment,
  management presentation, and unsupported configuration.
- Logs: Use event names, tier, platform, safe error classification, and outcome;
  omit DIDs, handles, billing UUIDs, session tokens, provider subscription IDs,
  transactions, receipts, and payment details.
- Metrics: Suggested counters/timings are paywall result by tier/platform,
  reconciliation completion/timeout, assignment outcome, restore outcome, and
  Customer Center presentation outcome. Do not use account identifiers as labels.
- Alerts: None required in Flutter for this slice. Existing AppView provider and
  reconciliation alerts remain authoritative for backend failures.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Flutter identifies RevenueCat as the active DID instead of the billing UUID. | Purchases can merge, move, or appear on the wrong billing account. | Centralize owner identity, prohibit anonymous billing, and test identity across switches/restarts. |
| RISK-002 | A store charge completes before AppView reflects it. | User believes payment failed or retries purchase. | Show a distinct pending state, bounded reconciliation wait, refresh/retry, and no duplicate checkout. |
| RISK-003 | RevenueCat/dashboard/store catalog is incomplete or mismatched. | Paywall, product, purchase, restore, or management flow cannot complete. | Keep activation external, fail closed in-app, and require per-platform sandbox verification. |
| RISK-004 | Account-scoped access caches cross a switch or stale async completion. | Wrong tier or billing data is displayed. | Key state by account lease, invalidate at account boundaries, and discard stale completions. |
| RISK-005 | Restore under keep-with-original identity cannot recover a payer who lost the owner account. | Legitimate purchase cannot be restored. | Show support-oriented failure and keep production activation blocked on a recovery procedure; do not transfer identity in this slice. |
| RISK-006 | Separate Plus and Business groups/products confuse users expecting an upgrade. | Accidental concurrent billing or poor cancellation expectations. | Explain independent account licenses, show both statuses/assignments, and fail closed on duplicate/unresolved purchase state. |
| RISK-007 | Customer Center behavior differs by platform. | Cancellation, refund, or external management experience is inconsistent. | Use RevenueCatUI's platform behavior, localize expectations, and verify both stores manually. |
| RISK-008 | A future RevenueCatUI release raises native minimum versions. | Some supported devices may no longer build or install after an SDK update. | Pin a compatible SDK range and recheck native targets during dependency updates. |
| RISK-009 | Owner session loss is mistaken for an empty billing-owner slot. | A second DID may alias or move the store receipt. | Retain an unusable pinned DID/UUID, prohibit replacement and logout, and require exact same-owner recovery. |
| RISK-010 | Subscription and license arrays are correlated by position. | Tier, assignment, or repurchase state can be attributed to the wrong purchase. | Add internal `subscriptionId` to license responses and join only by ID. |
| RISK-011 | The process stops between AppView ensure, RevenueCat identity, and local owner persistence. | AppView/RevenueCat may retain one owner while Flutter permits another. | Persist the confirming DID before PUT and the returned UUID before RevenueCat identity; only the reserved DID may resume. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Launch uses offering identifiers `plus` and `business`, each with one monthly package and an attached RevenueCat paywall. | Offering selection/configuration requirements must change. |
| ASM-002 | Platform public SDK keys will be supplied through the existing per-environment config files before sandbox verification. | Native purchase paths remain disabled. |
| ASM-003 | RevenueCat Customer Center will be configured before end-to-end verification. | Management renders a minimal or incomplete provider UI. |
| ASM-004 | Explicitly selecting the active account as owner and permanently pinning that installation to the owner until future recovery is acceptable launch UX. | A transfer/reset design is required before another owner can use billing on the installation. |
| ASM-005 | Inactive retained sessions may perform their own scoped self-access read without activating that account. | Account-switcher badges must be limited to cached or active-account data. |
| ASM-006 | The RevenueCat SDK version selected during implementation retains requirements no higher than iOS 15 and Android API 21. | Platform support, deployment targets, or SDK/UI approach must be reviewed. |
| ASM-007 | Observing the first `requestedGeneration` above a pre-request baseline is sufficient as the minimum reconciliation target even when other triggers race. | The reconciliation mutation may need to return an exact generation later. |

## 21. Open Questions

- [ ] Production blocking: supply and validate iOS and Android RevenueCat public
  SDK keys plus live app/store catalog credentials.
- [ ] Production blocking: configure and verify both offering paywalls and
  Customer Center in RevenueCat.
- [ ] Production blocking: verify every configured paywall version has a working
  close/dismiss affordance; do not rely only on `displayCloseButton` for V2.
- [ ] Production blocking: define account recovery/support when the original
  billing owner or UUID is unavailable under keep-with-original restore behavior.
- [ ] Production blocking: define the approved reset/transfer path for an
  installation whose pinned owner was permanently deleted or cannot return.
- [ ] Non-blocking: decide final localized merchandising copy and visual treatment
  during UI design/polish without changing the independent-license semantics.

## 22. Review Status

Status: Revised after document review

Risk level: High

Review recommended: Required

Reviewer: Product owner and OpenCode

Date: 2026-09-10

Notes: The product owner approved the initial direction and, after document
review, selected a crash-safe DID-first pinned original owner, a narrow AppView
`BillingLicense.subscriptionId` response enhancement, and client-side baseline
generation correlation. Billing identity, native charges, restore behavior, and
cross-account state keep this high risk. Revised documents require review before
coding planning. Live production activation remains separately blocked by the
catalog, credentials, paywall close affordances, Customer Center, and recovery
questions in section 21.

## 23. Handoff To Test Design

- Requirements file:
  `docs/changes/2026-09-10-flutter-subscriptions/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: All Must requirements in section 12.
- Suggested test levels: Unit tests for models, identity state, eligibility,
  reconciliation coordination, and provider-result mapping; widget tests for
  settings, status, assignment, error, lifecycle, accessibility, and account
  boundaries; API-client tests for every AppView route/error; platform build and
  manual sandbox tests for RevenueCat paywall, purchase, restore, and Customer
  Center behavior.
- Blocking open questions: Revised workflow document approval. Production
  credentials, catalog configuration, Customer Center setup, and both owner and
  installation recovery paths block live activation but do not block fake-based
  test design after approval.
