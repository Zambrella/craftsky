# TDD Implementation Plan: Flutter Account Subscriptions

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`, 2026-09-10)
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability and loop evidence updated here.
- Preserve the DID-before-PUT and UUID-before-RevenueCat ordering barriers.
- Keep AppView self-access authoritative and never log sensitive provider data.

## Test Order

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-013 | FR-012, FR-027 | AC-014, AC-029 | Owner license JSON lacks `subscriptionId` |
| 2 | UT-001, UT-012 | FR-009, FR-012, FR-027, NFR-001 | AC-012, AC-014, AC-023, AC-029 | Flutter subscription wire models do not exist |
| 3 | IT-001 | FR-003, FR-009, FR-012, FR-020, FR-027, NFR-001 | AC-007, AC-012, AC-014, AC-020, AC-023, AC-029 | Subscription API client does not exist |
| 4 | UT-002, IT-004 | FR-003, FR-005, FR-006, FR-008, RULE-002, RULE-008 | AC-007 through AC-009, AC-011 | Secure registry has no owner reservation |
| 5 | AT-001, AT-011 | FR-003, FR-004, FR-008, FR-026, RULE-002, RULE-008 | AC-007, AC-008, AC-011, AC-025 | Crash-safe setup lifecycle does not exist |
| 6 | UT-003, IT-012 | FR-001, FR-002, NFR-003 | AC-004, AC-006 | RevenueCat platform configuration does not exist |
| 7 | IT-003, UT-013, AT-017, IT-014 | FR-003, FR-004, FR-005, FR-008, FR-026, NFR-004 | AC-007, AC-008, AC-011, AC-025 | Owner operation guard does not exist |
| 8 | UT-004, AT-005 | FR-013, RULE-004 | AC-015, AC-019 | Purchase eligibility rules do not exist |
| 9 | UT-006, AT-006, IT-005 | FR-004, FR-014 through FR-016, FR-024 | AC-008, AC-016, AC-017, AC-022 | Paywall coordinator does not exist |
| 10 | UT-005, UT-011, AT-007 | FR-017, FR-024, NFR-008, RULE-006 | AC-018, AC-022, AC-025 | Reconciliation state machine does not exist |
| 11 | AT-002, IT-006 | BR-001, BR-002, FR-017 through FR-019, RULE-001, RULE-004, RULE-005 | AC-001, AC-002, AC-018 | Purchase controller cannot establish AppView access |
| 12 | UT-007 | FR-019, RULE-003, RULE-004 | AC-019 | Assignment candidate rules do not exist |
| 13 | AT-008, IT-008 | BR-001, FR-019 through FR-021, NFR-004 | AC-001, AC-019, AC-020 | Assignment controller does not exist |
| 14 | AT-009, IT-007 | BR-004, FR-004, FR-022, RULE-005 | AC-005, AC-008, AC-018 | Restore controller does not exist |
| 15 | AT-010, IT-009 | BR-004, FR-004, FR-023 | AC-005, AC-008, AC-021 | Customer Center coordinator does not exist |
| 16 | AT-003, AT-004, AT-013, AT-014, UT-010 | BR-002, FR-007, FR-009, FR-012, FR-024, FR-025 | AC-003, AC-009, AC-010, AC-012 through AC-014, AC-022 | Subscription page does not exist |
| 17 | IT-002 | FR-007, FR-010, FR-011, NFR-001 | AC-010, AC-013, AC-023 | Scoped access provider does not exist |
| 18 | AT-015, REG-002 | BR-002, FR-006, FR-010, FR-011, FR-026 | AC-003, AC-013 | Account switcher has no tier badges |
| 19 | IT-011 | FR-007, FR-009, FR-010 | AC-009, AC-013 | Subscription route and settings row do not exist |
| 20 | AT-012, REG-001, REG-003 | BR-003, FR-002, RULE-007, RULE-009 | AC-004, AC-006 | Optional-billing fallback is absent |
| 21 | UT-009, REG-006 | FR-005, NFR-002 | AC-008, AC-024 | Billing diagnostics are not covered |
| 22 | AT-016 | NFR-005 | AC-026 | Subscription accessibility is not covered |
| 23 | Localization coverage | FR-024, FR-025, NFR-005 | AC-014, AC-022, AC-026 | Subscription localization keys do not exist |
| 24 | REG-004, REG-005 | FR-008, FR-016, FR-020, FR-026 | AC-011, AC-017, AC-025 | Cancellation and invalidation regressions are unverified |
| 25 | REG-007, AC-027 | NFR-006, NFR-007 | AC-027 | Full automated gates have not run |
| 26 | MAN-001 through MAN-005, AC-028 | FR-001, FR-014, FR-022, FR-023, NFR-004, NFR-005, NFR-007 | AC-001, AC-005, AC-006, AC-016, AC-018, AC-021, AC-025, AC-026, AC-028 | Native catalog and evidence remain externally blocked |

## Implementation Steps

Each numbered step follows the same loop: write the smallest failing test, run its
focused command, confirm the failure represents missing behavior, implement the
minimum production change, rerun to green, run nearby tests, then record evidence
below before advancing.

### Step 1: IT-013

- Write failing test: Extend the existing registered AppView route contract test
  with inverse-ordered subscriptions/licenses and required `subscriptionId`.
- Run command: From `appview/`, run the focused `go test ./internal/routes
  -run '^TestOwnerBillingStateUsesPrivateCamelCaseContract$' -count=1` with the
  local test database environment.
- Confirmed failure: The license JSON had no `subscriptionId`; all other seeded
  owner fields decoded correctly.
- Implement: Add the response field and select/scan the existing internal UUID.
- Refactor: Only while focused and nearby AppView tests are green.
- Notes: Green after adding `BillingLicense.SubscriptionID` and selecting/scanning
  `provider_subscriptions.id`. The focused test and nearby
  `./internal/subscriptions ./internal/routes` suites pass against PostgreSQL.

### Step 2: UT-001, UT-012

- Confirmed failure: Subscription wire models and mappers did not exist.
- Implement: Added strict closed-tier access models and owner billing models,
  including UUID validation and ID-based license/subscription correlation.
- Notes: Focused model tests pass. Explicit required-type checks prevent coercion
  by generated mappers; optional provider fields remain forward-compatible.

### Step 3: IT-001

- Confirmed failure: No Flutter boundary represented the six AppView
  subscription operations.
- Implement: Added the authenticated JSON API client for access, billing account,
  reconciliation, and assignment routes with exact error-code preservation.
- Notes: Both focused API contract tests pass.

### Step 4: UT-002, IT-004

- Confirmed failure: The secure registry had no billing-owner state or mutation
  methods.
- Implement: Migrated secure snapshots from schema v2 to v3 and added a permanent
  DID-first reservation with optional UUID completion. Reservation requires the
  active account, cannot move to another DID or UUID, and survives session removal.
- Notes: Model and provider persistence tests pass, including v2 reconstruction.

### Step 5: AT-001, AT-011

- Confirmed failure: No setup boundary enforced durable reservation ordering or
  fenced owner-session changes around asynchronous calls.
- Implement: Added an SDK-free setup coordinator with exact active-owner lease
  checks around DID persistence, AppView PUT/GET, UUID persistence, and provider
  identity. Completed reservations are GET-only; owner and identity mismatches
  fail locked.
- Notes: Acceptance tests pass for first setup ordering, pending GET invalidation,
  and replacement-owner rejection. The native SDK adapter remains step 6 work.

### Step 6: UT-003, IT-012

- Confirmed failure: RevenueCat packages, platform key selection, explicit native
  targets, and safe unavailable fallback did not exist.
- Implement: Added RevenueCat Flutter/UI 10.12.0, platform-specific public-key
  inputs, iOS 15 target, SDK-free bootstrap/configurator boundaries, sanitized
  native identity calls, and a provider override. Debug builds use SDK debug
  logging; release builds use info without a raw log handler.
- Notes: Configuration tests and touched-path analysis pass. Native debug builds
  pass for iOS simulator and Android APK. Missing keys and configuration failures
  leave billing unavailable without preventing app startup.

### Step 7: IT-003, UT-013, AT-017, IT-014

- Confirmed failure: No reusable exact-owner lease guard or provider identity
  precondition existed across owner operation classes.
- Implement: Added capture/dispatch/completion fencing for the pinned owner and
  UUID, plus exact RevenueCat identity validation. Native dispatch rechecks the
  owner after asynchronous identity inspection to close the switch race.
- Notes: Guard tests cover every planned owner call class, beneficiary rejection,
  stale pre-dispatch state, and an owner change during identity inspection.

### Step 8: UT-004, AT-005

- Confirmed failure: No tier-correlated checkout eligibility rule existed.
- Implement: Added a pure ID-correlated eligibility projection. Checkout is
  allowed only for a fresh absent tier or when every correlated subscription is
  inaccessible, non-pending, anomaly-free, and exactly `will_not_renew`.
- Notes: Unknown/empty renewal and anomaly values fail closed, tiers remain
  independent, and dormant assignment DIDs are disclosed without mutation.

### Step 9: UT-006, AT-006, IT-005

- Confirmed failure: No SDK-free named-offering/paywall outcome coordinator
  existed.
- Implement: Added direct `plus`/`business` offering lookup, required
  `$rc_monthly` preflight, native RevenueCatUI presentation with an explicit
  offering and close-button request, and deterministic result mapping.
- Notes: Missing offering/package stops before presentation; attachment/template
  throws become provider errors after the attempted presentation. Each provider
  call is guarded by the exact owner lease and RevenueCat UUID.

### Step 10: UT-005, UT-011, AT-007

- Confirmed failure: No baseline/target generation reducer or bounded polling
  coordinator existed.
- Implement: Added first-post-baseline target capture, exact target reconciliation,
  purchase change recognition, unchanged restore completion, anomaly failure,
  transient read handling, bounded one-second polling, and cancellation fencing.
- Notes: Generation movement alone never activates purchase. Pre-provider outcome
  comparison is separate from the post-provider generation baseline so fast
  webhook updates cannot hide a new license.

### Step 11: AT-002, IT-006

- Confirmed failure: No purchase orchestration joined eligibility, direct paywall,
  reconciliation, and assignment prompting.
- Implement: Added exact-owner purchase orchestration with pre-provider comparison,
  post-provider generation baseline, safe provider outcomes, and ID-correlated
  assignment detection.
- Notes: New licenses remain unassigned, same-subscription recovery preserves its
  assignment, and a new post-lapse license does not replace a dormant assignment.

### Step 12: UT-007

- Confirmed failure: No pure assignment-candidate projection existed.
- Implement: Added retained-lease candidate filtering against current registry
  leases and all existing paid assignments.
- Notes: Owner and non-owner sessions are eligible; stale, missing, duplicate, and
  already-assigned DIDs are excluded.

### Step 13: AT-008, IT-008

- Confirmed failure: Assignment and unassignment had no serialized controller.
- Implement: Added duplicate suppression, exact AppView error mapping, owner-state
  refresh after non-401 outcomes, target-lease race checks, and post-success target
  access refresh.
- Notes: No optimistic assignment is published; refreshed AppView state remains
  authoritative after success or failure.

### Step 14: AT-009, IT-007

- Confirmed failure: The RevenueCat boundary and owner controller had no explicit
  restore operation.
- Implement: Added sanitized native restore and exact UUID dispatch followed by
  restore-mode reconciliation.
- Notes: Unchanged restore may complete, known assignments persist, and newly
  discovered licenses remain unassigned.

### Step 15: AT-010, IT-009

- Confirmed failure: The RevenueCat boundary exposed no Customer Center lifecycle
  events and no owner coordinator could refresh after management.
- Implement: Added sanitized native Customer Center presentation, exact UUID
  dispatch, management/restore event coalescing, post-dismiss owner refresh, and
  reconciliation after provider mutation callbacks.
- Notes: Plain dismissal performs a GET without claiming mutation; identity mismatch
  prevents native presentation.

### Step 16: Subscription Page

- Confirmed failure: No role-aware subscription UI projected AppView owner and
  self-access state.
- Implement: Added a pure page model/reducer and responsive production page for
  setup, purchase, restore, Customer Center, assignment, unassignment, and refresh.
- Notes: Beneficiaries see only self-access; owner licenses correlate by internal
  ID and unknown provider states fail closed.

### Step 17: Scoped Access

- Confirmed failure: No lease-keyed access provider isolated retained accounts.
- Implement: Added account-keyed repositories and lease-fenced self-access reads.
- Notes: Inactive reads do not activate accounts or reach owner endpoints, and
  replaced leases cannot publish delayed results.

### Step 18: Account Switcher Badges

- Confirmed failure: Retained account rows had no effective-tier projection.
- Implement: Added lease-keyed tier badges with isolated loading/error states.
- Notes: Badge text scaling is bounded inside the compact badge while full
  semantics remain available; account labels flex and truncate in narrow menus.

### Step 19: Settings Route

- Confirmed failure: Settings had no subscription row or canonical route.
- Implement: Added `/profile/settings/subscriptions`, generated route bindings,
  row ordering, navigation, and the active account's self-access subtitle.
- Notes: Production router and Settings tests pass for compact and large shells.

### Step 20: Optional Billing

- Confirmed failure: The UI fallback and dependency isolation had no regression
  coverage.
- Implement: Added unavailable-billing UI and source-boundary tests.
- Notes: Missing/unsupported native billing remains isolated to subscription
  adapters and does not gate existing feature areas.

### Step 21: Diagnostics

- Confirmed failure: Redaction and raw provider-log isolation were unverified.
- Implement: Added canary diagnostics tests for billing models and the native
  adapter boundary.
- Notes: Diagnostics expose state shape but not DIDs, UUIDs, products, payment
  canaries, raw exceptions, or provider log forwarding.

### Step 22: Accessibility

- Confirmed failure: Narrow-screen and large-text owner actions were unverified.
- Implement: Added semantic action labels, live status regions, minimum action
  targets, scrolling, and a 320px/2x text widget test.
- Notes: Owner actions remain reachable without overflow.

### Step 23: Localization

- Confirmed failure: Subscription UI strings were absent from generated l10n.
- Implement: Added English keys for tiers, statuses, explanations, actions,
  confirmation copy, errors, and semantics; regenerated localization output.
- Notes: A getter coverage test prevents literal fallback copy in the feature.

### Step 24: Cancellation And Invalidation

- Confirmed failure: Provider cancellation and account-boundary cleanup lacked
  explicit regression tests.
- Implement: Added cancellation no-op coverage and invalidated subscription
  repository/access/page projections at account boundaries while preserving the
  permanent billing-owner binding.
- Notes: The focused subscription suite passes 154 tests. Correction coverage
  includes tier-scoped purchase reconciliation, matching beneficiary self-access,
  exact assignment outcomes, UUID fences, stale-owner cancellation, and deferred
  foreground reconciliation at production page/controller boundaries.

### Step 25: Automated Gates

- Run commands: `just app-test`, `just app-analyze`, `just test`, and
  `just appview-check` as supported by the local stack.
- Notes: Complete. `flutter test` passed 2,448 tests, `flutter analyze` reported
  no issues, `just test` passed, `just appview-check` passed with the existing
  no-fix `GO-2026-5932` advisory, and `git diff --check` passed.

### Step 26: Native And Manual Evidence

- Run commands: `just app-build-ios staging`, `just app-build-apk staging`, and
  `just app-build-web staging` before manual sandbox checks.
- Notes: Keyless smoke builds passed for iOS simulator, Android debug APK, and
  web. The staging recipes remain blocked by absent `app/config/staging.env`;
  production catalog, credentials, Customer Center, paywall close affordances,
  restore policy, and recovery procedures leave `MAN-001` through `MAN-005`
  explicitly blocked rather than failed.

## Execution Notes

| Step | Result | Focused Evidence | Notes |
|---|---|---|---|
| 1 | Complete | Focused route test passed; nearby subscription and route packages passed against PostgreSQL | Added only the internal subscription/license response relationship; no route, migration, or behavior change. |
| 2 | Complete | Subscription access and billing model tests passed | Added strict JSON contracts, relationship validation, and redacted diagnostics. |
| 3 | Complete | Subscription API client tests passed | Covers all planned AppView routes and exact assignment error preservation. |
| 4 | Complete | Registry model and provider tests passed | Schema v3 persists permanent DID-only or DID-plus-UUID owner reservation; v2 migrates with no owner. |
| 5 | Complete | Billing owner account-boundary acceptance tests passed | Enforces DID-before-PUT, UUID-before-identity, GET-only recovery, and stale-session fencing. |
| 6 | Complete | RevenueCat configuration tests, iOS simulator build, and Android debug APK build passed | Uses public platform keys, non-verbose release logging, and safe unsupported-platform fallback. |
| 7 | Complete | Active-owner and RevenueCat identity guard tests passed | Every owner call class is fenced; anonymous, DID-shaped, and mismatched provider identities fail before native work. |
| 8 | Complete | Purchase eligibility matrix passed | Only absent or fully lapsed safe tiers permit checkout; dormant assignments remain intact. |
| 9 | Complete | Named paywall coordinator and native adapter tests passed | Uses only `plus`/`business` offerings with `$rc_monthly`; purchase/restore reconcile while cancel is a no-op. |
| 10 | Complete | Reconciliation reducer and bounded polling tests passed | Handles concurrent generations, purchase/restore differences, transient reads, timeout, anomaly, and cancellation. |
| 11 | Complete | Purchase controller tests passed | Eligibility, provider result, reconciliation, recovery, and new-license assignment prompting remain AppView-authoritative. |
| 12 | Complete | Assignment candidate tests passed | Filters retained current leases against assignments across both paid tiers. |
| 13 | Complete | Assignment controller tests passed | Serializes mutations, maps exact errors, and refreshes owner/target server truth. |
| 14 | Complete | Restore controller tests passed | Exact owner UUID, unchanged restore, known assignment preservation, and new unassigned license are covered. |
| 15 | Complete | Customer Center coordinator tests passed | Management callbacks reconcile after foreground resume; plain dismissal refreshes; wrong identity never presents native UI. |
| 16 | Complete | Role/presentation provider and widget tests passed | Owner and beneficiary privacy, setup recovery, stale state, and safe status copy are covered. |
| 17 | Complete | Lease-scoped access provider tests passed | Inactive access is isolated and stale completions are fenced. |
| 18 | Complete | Tier badge and app-shell switcher tests passed | Loading/error isolation and narrow-menu layout are covered. |
| 19 | Complete | Settings model/page/router tests passed | Canonical route works in compact and large Settings shells. |
| 20 | Complete | AT-012 and REG-001/REG-003 tests passed | Optional native billing cannot break startup or gate existing features. |
| 21 | Complete | UT-009 and REG-006 tests passed | Sensitive diagnostics and raw provider forwarding are excluded. |
| 22 | Complete | AT-016 widget test passed at 320px and 2x text | Semantics and reachable actions are covered. |
| 23 | Complete | Subscription localization getter test passed | Generated English strings cover all subscription UI states. |
| 24 | Complete | 154 focused subscription tests passed | Cancellation performs no mutation; account invalidation clears projections but preserves ownership; confirmed provider results defer reconciliation until foreground resume. |
| 25 | Complete | 2,448 Flutter tests, clean analysis, AppView tests/release checks, and whitespace validation passed | Existing no-fix `GO-2026-5932` remains outside this feature. |
| 26 | Blocked | iOS simulator, Android debug APK, and web smoke builds passed; manual evidence requires external catalog and credentials | Staging recipes need `app/config/staging.env`; `MAN-001` through `MAN-005` remain externally blocked. |

## Completion Checklist

## Review Correction Pass: 2026-09-11

The repeated implementation review returned `Changes required`. Corrections use
the existing approved requirements and test IDs in this order:

| Step | Status | Test IDs | Requirement IDs | Acceptance Criteria | Correction Evidence |
|---|---|---|---|---|---|
| C1 | Complete | AT-005, IT-008 | FR-013, FR-021 | AC-015, AC-020 | Canonical dormant licenses expose unassignment but not unsupported direct reassignment. Removing the dormant assignment makes its DID eligible for the new replacement license. |
| C2 | Complete | AT-007, IT-006 | FR-017, FR-024, RULE-006 | AC-018, AC-022 | Purchase and restore retain reconciliation-only retry closures after baseline GET, POST, poll, or timeout failures; Refresh never reopens checkout. |
| C3 | Complete | IT-009, IT-010 | FR-023, FR-026, NFR-004 | AC-021, AC-025 | Foreground state is separate from owner/page validity; confirmed purchase and Customer Center results reconcile after resume without duplicate native presentation. |
| C4 | Complete | UT-004 | FR-013, NFR-001 | AC-015, AC-022 | Eligibility validates all subscription/license IDs and rejects orphan, duplicate, or unresolved relationships before evaluating an absent tier. |
| C5 | Complete | IT-002 | FR-010, FR-011, FR-026 | AC-013, AC-023 | Self-access rejects response-DID mismatch and a real `accountDioProvider(account)` family test proves inactive-account request scope. |
| C6 | Complete | AT-011, AT-014 | FR-008, FR-009, FR-024 | AC-011, AC-012, AC-022 | Owner mismatch/404/unauthorized and transient failures preserve known self-access while rendering distinct locked or refreshable management substates. |
| C7 | Complete | AT-008 | FR-021 | AC-020 | Reassignment confirmation has localized, explicit seven-day target-change cooldown copy with widget coverage before dispatch. |
| C8 | Complete | IT-001 | FR-020, NFR-001 | AC-020, AC-023 | HTTP tests cover all four exact assignment errors, DELETE `billing_license_not_found`, request IDs, and 401/redacted 5xx mapping for both mutation methods. |
| C9 | Complete | IT-006 | FR-018 | AC-001, AC-002 | Access decoding rejects contradictory predicates; recovered assignment tests independently reject wrong DID, wrong tier, and inaccessible matching assignment. |
| C10 | Complete | IT-014 | FR-026, NFR-004 | AC-025 | Production purchase, restore, Customer Center, paywall, assignment, and unassignment tests exercise stale-owner boundaries and return cancellation. |
| C11 | Complete | IT-013 | FR-027 | AC-029 | Seeded existing-account PUT independently asserts both inverse-ordered `subscriptionId` relationships. |

Each correction followed a focused red-green-refactor loop. The complete
correction suite and full automated gates are green.

### C1: AT-005, IT-008

- Confirmed failure: Canonical dormant data (`givesAccess=false`,
  `assignable=false`) initially hid assignment management, then a prior correction
  exposed direct reassignment that AppView rejects.
- Implement: Dormant licenses expose unassignment but never direct reassignment.
  After refreshed server truth removes that assignment, the original DID becomes
  eligible for explicit assignment of the new replacement license.
- Notes: Canonical dormant widget and replacement-candidate tests pass.

### Correction Verification

- `flutter test test/subscriptions`: 154 tests passed.
- `flutter test`: 2,448 tests passed.
- `flutter analyze`: no issues.
- `just test`: passed.
- `just appview-check`: all release gates passed; existing no-fix
  `GO-2026-5932` remains outside this feature.
- `flutter build ios --debug --simulator`: passed.
- `flutter build apk --debug`: passed with the existing Kotlin deprecation
  warning.
- `flutter build web`: passed with the existing icon-font warning.
- `git diff --check`: passed.

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned automated Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Documentation and execution evidence updated
- [ ] Implementation review completed or explicitly skipped

## Second Review Correction Pass: 2026-09-13

The latest implementation review identified `IR-001` through `IR-006`. This
pass corrects those findings against the existing approved requirements and test
IDs without expanding the billing contract.

| Step | Status | Test IDs | Requirement IDs | Acceptance Criteria | Correction Evidence |
|---|---|---|---|---|---|
| IR-003 | Complete | IT-006, IT-010 | FR-017, FR-024, FR-026, NFR-004 | AC-018, AC-022, AC-025 | Purchase and restore controller tests reproduce a foreground transition between poll cancellation and outcome mapping; production-page tests hold a polling GET across background/resume. Both operations retain reconciliation-only pending state without repeating checkout or restore. |
| IR-004 | Complete | IT-006, IT-010, IT-014 | FR-017, FR-026, NFR-004 | AC-018, AC-025 | Confirmed purchase and restore results create container-scoped pending intent that survives page disposal and owner changes. Exact-owner and RevenueCat identity checks remain immediately before native dispatch, while post-native staleness defers AppView work instead of rewriting the provider result. |
| IR-005 | Complete | IT-006, IT-010 | FR-006, FR-017, FR-024, FR-026, RULE-006 | AC-003, AC-018, AC-022, AC-025 | Purchase and restore pending operations retain intent data rather than an activation lease. Refresh constructs a fresh controller and reacquires the exact owner after switching back without repeating checkout or restore. |
| IR-002 | Complete | AT-011, IT-004, IT-014 | FR-003, FR-004, FR-008 | AC-007, AC-008, AC-011 | The production page provider detects an anonymous RevenueCat identity only after a valid GET for the persisted UUID and exposes explicit same-owner recovery. Retry uses the coordinator's GET-only path and identifies once; 404 and AppView/RevenueCat UUID mismatches stay locked without ensure or reidentification. |
| IR-001 | Complete | AT-005, AT-008, IT-008 | FR-013, FR-021 | AC-015, AC-020 | Dormant non-assignable licenses expose removal and repurchase but not direct reassignment. Active assignable licenses retain reassignment, and refreshed post-unassignment state enables the old DID as a target for the new replacement license. |
| IR-006 | Complete | IT-001, IT-008 | FR-020, NFR-001 | AC-020, AC-023 | API-client tests cover the DELETE missing-license envelope and request ID plus unauthorized and redacted server-error envelopes for assignment PUT and unassignment DELETE. |

### IR-003: IT-006, IT-010

- Confirmed failure: A deterministic foreground sequence caused purchase and
  restore poll cancellation to be interpreted as final cancellation after the
  app had already resumed.
- Implement: A cancelled reconciliation remains pending whenever the page and
  exact captured owner are still current; stale page or owner cancellation is
  unchanged.
- Verification: Focused purchase, restore, and production-page suites pass 30
  tests. The production-page tests confirm one native presentation/restore call
  while a polling GET spans background and resume.

### IR-004, IR-005: IT-006, IT-010, IT-014

- Confirmed failures: Paywall owner changes rewrote a confirmed purchase as
  cancellation; route disposal lost page-local pending state; purchase and
  restore retry closures rejected a new activation generation after switching
  back to the same owner.
- Implement: Added a container-scoped in-memory pending operation containing the
  reconciliation intent and pre-provider baseline. Native operations retain
  exact identity and owner checks before dispatch, then record confirmed results
  without a post-native stale-owner rewrite. Refresh rebuilds owner-scoped
  dependencies and reacquires a fresh lease before AppView reconciliation.
- Verification: Paywall, purchase, restore, and production-page focused suites
  pass 44 tests, including route disposal, owner switch/removal, switch-back
  retry, and one-call assertions for checkout and restore.

### IR-002: AT-011, IT-004, IT-014

- Confirmed failure: A completed local owner binding with anonymous RevenueCat
  identity rendered ordinary owner controls and provided no path to finish
  identity setup.
- Implement: After a valid owner GET, the page provider checks RevenueCat
  identity. Anonymous identity renders the existing explicit setup-retry state;
  its action invokes the coordinator, which selects GET because the UUID is
  already persisted. AppView UUID mismatch, 404, and identified RevenueCat UUID
  mismatch remain locked.
- Verification: The production widget path identifies exactly once with zero
  ensure calls, while provider tests prove no identity call after AppView 404 or
  UUID mismatch and no reidentification after RevenueCat identity mismatch.

### IR-001: AT-005, AT-008, IT-008

- Confirmed failure: Canonical dormant state rendered “Change assignment” even
  though AppView returns `billing_license_not_found` for a different target when
  `assignable=false`.
- Implement: Restricted reassignment to active or canceled-accessible licenses
  that are explicitly assignable. Dormant licenses retain removal so the user can
  free the DID before assigning the new replacement license.
- Verification: Widget tests distinguish active reassignment from dormant
  removal, and candidate tests prove the old DID is excluded before dormant
  unassignment and included afterward.

### IR-006: IT-001, IT-008

- Initial state: The shared interceptor already mapped these responses, but the
  subscription API-client suite did not exercise the DELETE route or either
  mutation method under 401/5xx responses.
- Implement: Added direct HTTP adapter coverage for DELETE
  `billing_license_not_found` with request ID, plus PUT and DELETE unauthorized
  envelopes and redacted server-error envelopes.
- Verification: All 12 subscription API-client contract tests pass.

## Third Review Correction Pass: 2026-09-13

The repeated implementation review resolved `IR-001` through `IR-006` and found
`IR-007` through `IR-012`. Corrections remain within the approved billing and
lifecycle requirements and will run in this TDD order:

| Step | Status | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| IR-008 | Complete | IT-010, IT-014 | FR-015, FR-017, FR-022, FR-026, NFR-004 | AC-016, AC-018, AC-025 | Owner-bound in-flight purchase and restore reservations survive route disposal and disable a recreated route before the unresolved native result returns. Focused tests prove one native call. |
| IR-009 | Complete | IT-006, IT-007, IT-010 | FR-017, FR-022, FR-024, FR-026, NFR-004, RULE-006 | AC-005, AC-018, AC-022, AC-025 | In-flight and reconciliation-pending state disables purchase, restore, and management. The state controller rejects a second operation and preserves the original purchase tier and baseline. |
| IR-010 | Complete | AT-010, IT-009, IT-010 | FR-023, FR-026, NFR-004 | AC-005, AC-021, AC-025 | Customer Center mutation callbacks persist owner-bound reconciliation intent above the route. Route disposal and owner switch-back reconcile through a fresh owner lease without reopening Customer Center. |
| IR-007 | Complete | AT-002, AT-005, AT-008, AT-013, IT-008 | FR-012, FR-013, FR-018, FR-019, FR-021 | AC-001, AC-014, AC-015, AC-019, AC-020 | Tier UI correlates licenses by subscription ID, exposes both dormant removal and active replacement assignment, and a production flow proves refreshed reassignment to the released DID. |
| IR-011 | Complete | AT-011, IT-004, IT-014 | FR-003, FR-004, FR-008 | AC-007, AC-008, AC-011 | A shared event trace proves persisted-UUID recovery performs owner GET, identity inspection, then identify, with zero ensure calls. |
| IR-012 | Complete | REG-007 | NFR-006, NFR-007 | AC-027 | Current evidence records 154 focused subscription tests and 2,448 full Flutter tests plus all repository and platform gates. |

Each correction writes one focused failing behavior test, confirms that it fails
for the reviewed gap, applies the minimum production change, reruns the focused
test and nearby suite, then records the result here before advancing.

### IR-007: AT-002, AT-005, AT-008, AT-013, IT-008

- Confirmed failure: The tier card selected the first oldest-first license, so a
  dormant assigned license hid the active replacement license.
- Implement: Select the accessible replacement license by exact subscription ID
  and render independent removal actions for older inaccessible assignments.
- Verification: Widget coverage proves both license IDs remain actionable. A
  production page/controller test removes the dormant assignment, consumes the
  refreshed AppView state, and assigns the active replacement to the released DID.

### IR-008, IR-009: IT-006, IT-007, IT-010, IT-014

- Confirmed failures: Native purchase and restore were reserved only by route-local
  state before provider confirmation, and pending Restore or Manage could dispatch
  again or replace a purchase reconciliation baseline.
- Implement: Added owner-bound native-in-flight and reconciliation-pending phases
  above the route for purchase, restore, and Customer Center. Every native billing
  action is disabled while an operation exists, and only that operation's valid
  state transition may update its intent.
- Verification: Production route-disposal tests hold purchase and restore futures
  unresolved across route recreation and prove exactly one native call. Pending
  exclusivity tests prove purchase tier and baseline remain unchanged.

### IR-010: AT-010, IT-009, IT-010

- Confirmed failure: Customer Center retained mutation confirmation only in a
  route-local boolean and returned cancellation after a stale completion.
- Implement: Mutation callbacks immediately persist owner-bound Customer Center
  reconciliation intent. Retry reconstructs the coordinator from the current
  container and reacquires the exact owner's fresh lease without native presentation.
- Verification: The production test confirms mutation, disposes and recreates the
  route, switches to another account, switches back, and reconciles once with one
  Customer Center presentation.

### IR-011: AT-011, IT-004, IT-014

- Implement: Added one ordered test event stream spanning AppView and RevenueCat.
- Verification: Persisted-UUID recovery asserts `billingGet`, `identityRead`, then
  `identify`, with zero ensure calls. Existing mismatch tests retain zero identity
  mutation guarantees.

### IR-012: REG-007

- Implement: Replaced stale focused and full-suite totals throughout current
  implementation evidence and recorded this correction pass's gates.

### Third Correction Verification

- `flutter test test/subscriptions`: 154 tests passed.
- `flutter test`: 2,448 tests passed.
- `flutter analyze`: no issues.
- `just test`: passed.
- `just appview-check`: all release gates passed; existing no-fix
  `GO-2026-5932` remains outside this feature.
- `flutter build ios --debug --simulator`: passed with the existing future Swift
  Package Manager plugin warning.
- `flutter build apk --debug`: passed with existing Kotlin migration warnings.
- `flutter build web`: passed with the existing icon-font warning.
- `git diff --check`: passed.
- Staging recipes and `MAN-001` through `MAN-005` remain blocked by absent
  `app/config/staging.env` and external RevenueCat/store configuration.

## Eighth Review Correction Pass: 2026-09-14

The seventh implementation review identified one access-presentation regression
and one billing-sensitive reconciliation integrity gap. Corrections run in the
review-recommended TDD order:

| Step | Status | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| IR-020 | Complete | AT-004, AT-013 | FR-009, NFR-005 | AC-002, AC-012 | Canonical active paid access incorrectly renders dormant-assignment wording. |
| IR-021 | Complete | UT-005, IT-006, IT-007 | FR-017, FR-018, FR-022, FR-024, RULE-005, RULE-006 | AC-001, AC-002, AC-005, AC-018, AC-022 | Reconciliation can complete or offer assignment for unresolved or inaccessible subscription/license relationships. |

Each loop adds one focused failing public-interface test, confirms the reviewed
failure, applies the minimum production correction, and reruns its nearby suite
before advancing. The non-blocking Should-level `IR-022` remains deferred.

### IR-020: AT-004, AT-013

- Confirmed failure: Canonical active Plus access with a matching assigned tier
  rendered the dormant-assignment message. The initial assertion was corrected to
  match the exact localized wording before production code changed.
- Implement: The account-access card now requires `assignedTier` and
  `givesAccess == false` before rendering dormant copy.
- Verification: One widget case covers canonical active Plus and Business access;
  the existing effective-Free dormant cases remain green across page roles. The
  focused page suite passes 20 tests.

### IR-021: UT-005, IT-006, IT-007

- Confirmed failure: A reconciled orphan subscription completed restore; a newly
  observed inaccessible purchase license completed and requested assignment; and
  an inaccessible restored license was returned as an assignment candidate.
- Implement: A shared relationship-integrity predicate now drives both purchase
  eligibility and reconciliation. Reconciliation fails closed for unresolved
  relationships and for a newly introduced expected-tier purchase license whose
  exact subscription does not give access. Purchase and restore assignment
  selection independently require an accessible correlated subscription.
- Verification: New tests cover orphan purchase/restore subscriptions,
  inaccessible purchase and restored licenses, and the shared reconciliation
  decision. The focused reconciliation/purchase/restore group passes 41 tests.

### Eighth Correction Verification

- `flutter test test/subscriptions`: 173 tests passed.
- `flutter test`: 2,467 tests passed.
- Dart MCP static analysis: no diagnostics.
- `just test`: passed.
- `just appview-check`: all release gates passed; existing no-fix
  `GO-2026-5932` remains outside this feature.
- `flutter build ios --simulator --no-codesign`: passed with the existing future
  Swift Package Manager plugin warning.
- `flutter build apk --debug`: passed with existing Kotlin migration warnings.
- `flutter build web`: passed with the existing icon-font warning.
- `git diff --check`: passed.
- Staging recipes and `MAN-001` through `MAN-005` remain blocked by absent
  `app/config/staging.env` and external RevenueCat/store configuration.
- `IR-022` remains an explicitly deferred, non-blocking Should-level suggestion.

## Seventh Review Correction Pass: 2026-09-14

The sixth implementation review identified four remaining Must-level presentation
and recovery gaps. Corrections run in the review-recommended TDD order:

| Step | Status | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| IR-016 | Complete | AT-013, AT-014 | FR-012, FR-021, FR-024 | AC-014, AC-020, AC-022 | An assigned anomalous primary license cannot be safely unassigned. |
| IR-017 | Complete | AT-004, AT-013 | FR-009 | AC-012 | Dormant assigned-tier semantics render only for the beneficiary role. |
| IR-018 | Complete | AT-005, AT-014 | FR-012, FR-013, FR-024, RULE-006 | AC-014, AC-015, AC-022 | An unresolved subscription/license relationship appears purchasable instead of requiring support. |
| IR-019 | Complete | AT-011 | FR-007, FR-008, FR-024 | AC-009, AC-011 | A reserved but non-retained owner is projected as an ordinary beneficiary, leaving missing-owner guidance unreachable. |

Each loop adds one focused failing public-interface test, confirms the reviewed
failure, applies the minimum production correction, and reruns its nearby suite
before advancing.

### IR-016: AT-013, AT-014

- Confirmed failure: The anomalous multi-license fixture expected two safe removal
  actions but found only the healthy secondary license action.
- Implement: Every assigned primary license now exposes explicit unassignment even
  when its aggregate tier is anomalous. Purchase, assignment, and reassignment
  remain suppressed for the anomalous tier.
- Verification: The widget test invokes both removal actions and proves callbacks
  receive the two exact license IDs.

### IR-017: AT-004, AT-013

- Confirmed failure: A never-reserved account with effective Free and dormant Plus
  assignment did not render the assigned tier.
- Implement: Dormant assigned-tier semantics now render in the account-access card
  for every page role; the beneficiary-only duplicate was removed.
- Verification: Never-reserved, locked-owner, and beneficiary widget states retain
  both effective-access and dormant-assignment information.

### IR-018: AT-005, AT-014

- Confirmed failure: An owner subscription without a matching license rendered
  both tiers as available to purchase and showed no support-required state.
- Implement: A tier whose fresh purchase eligibility reports an anomaly is
  projected as `SubscriptionTierPresentation.anomaly` before actions are built.
- Verification: The orphan-relationship widget state renders two support-required
  cards and no Plus or Business purchase action. The six eligibility model tests
  remain green.

### IR-019: AT-011

- Confirmed failure: A reserved but non-retained billing owner produced the
  beneficiary role instead of the missing-owner recovery role.
- Implement: The page-model provider now returns `inactiveOwner` when the pinned
  owner session is absent.
- Verification: Eight provider tests pass. Widget coverage proves the page shows
  reauthentication guidance while withholding switch, restore, management,
  mutation, and owner-detail surfaces.

### Seventh Correction Verification

- `flutter test test/subscriptions`: 167 tests passed.
- `flutter test`: 2,461 tests passed.
- Focused subscription-page suite: 19 tests passed after formatting cleanup.
- Dart MCP static analysis: no diagnostics.
- `just test`: passed.
- `just appview-check`: all release gates passed; existing no-fix
  `GO-2026-5932` remains outside this feature.
- `flutter build ios --simulator --no-codesign`: passed with the existing future
  Swift Package Manager plugin warning.
- `flutter build apk --debug`: passed with existing Kotlin migration warnings.
- `flutter build web`: passed with the existing icon-font warning.
- `git diff --check`: passed.
- Staging recipes and `MAN-001` through `MAN-005` remain blocked by absent
  `app/config/staging.env` and external RevenueCat/store configuration.

## Sixth Review Correction Pass: 2026-09-14

The fifth implementation review retained one billing-sensitive fail-closed gap:

| Step | Status | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| IR-016 | Complete | AT-013, AT-014 | FR-012, FR-019, FR-024, RULE-006 | AC-014, AC-019, AC-022 | Tier-wide anomalies suppress secondary assignment and reassignment while retaining explicitly safe unassignment. |

The focused widget loop will place an anomalous accessible license before a healthy
assigned secondary license. The tier must render as requiring support, assignment
and reassignment must fail closed, and explicit unassignment remains available.

### IR-016: AT-013, AT-014

- Confirmed failure: A shuffled duplicate-tier state selected the anomalous
  accessible license as primary while the healthy assigned secondary license still
  exposed `Change assignment`.
- Implement: Secondary assignment and reassignment now require the aggregate tier
  presentation to be non-anomalous. Existing unassignment remains available as the
  explicitly safe repair action.
- Verification: Widget coverage proves anomaly-wide fail-closed behavior with an
  anomalous primary and healthy assigned secondary license, and separately renders
  dormant and anomalous unassigned secondary records without mutation actions.

### Sixth Correction Verification

- `flutter test test/subscriptions`: 163 tests passed.
- `flutter test`: 2,457 tests passed.
- Dart MCP static analysis: no diagnostics.
- `just test`: passed.
- `just appview-check`: all release gates passed; existing no-fix
  `GO-2026-5932` remains outside this feature.
- `flutter build ios --simulator --no-codesign`: passed with the existing future
  Swift Package Manager plugin warning.
- `flutter build apk --debug`: passed with existing Kotlin migration warnings.
- `flutter build web`: passed with the existing icon-font warning.
- `git diff --check`: passed.
- The first concurrent Android build collided with Flutter's iOS ephemeral-package
  cleanup; its isolated rerun passed.
- Staging recipes and `MAN-001` through `MAN-005` remain blocked by absent
  `app/config/staging.env` and external RevenueCat/store configuration.

## Fifth Review Correction Pass: 2026-09-14

The repeated implementation review resolved `IR-008` and `IR-010` and retained
three blocking behavioral gaps plus one traceability note. Corrections will run in
the review-recommended TDD order:

| Step | Status | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| IR-013 | Complete | IT-008, IT-010 | FR-020, FR-021, NFR-004 | AC-020, AC-025 | Shared mutation completion now clears the exact reservation and invalidates owner plus affected beneficiary providers without letting the disposed controller issue another call. |
| IR-014 | Complete | UT-005, AT-014, IT-006, IT-007, IT-009 | FR-017, FR-024, RULE-006 | AC-018, AC-022 | Reconciled anomalous state remains distinct from retryable reconciliation failure and renders the support-required notice. |
| IR-007 | Complete | AT-013, IT-008 | FR-012, FR-019, FR-021 | AC-014, AC-019, AC-029 | Every non-primary license renders its exact correlated status, details, and valid assignment actions. |
| IR-015 | Complete | IT-013 | FR-012, FR-027 | AC-014, AC-029 | The acceptance path now names the route suite and the implemented Go test name includes `IT013`. |

Each behavioral correction starts with one focused failing public-interface test,
applies the minimum production change, and reruns its nearby suite before the next
test. `IR-015` is documentation/test-name traceability only and follows the green
behavior loops.

### IR-013: IT-008, IT-010

- Confirmed failure: A completed assignment or unassignment after route disposal
  cleared its shared reservation but left the recreated owner and beneficiary
  providers on stale pre-mutation state.
- Implement: Shared mutation completion now clears only the matching reservation
  and invalidates the owner page model plus affected beneficiary access provider.
- Verification: Production route tests retain recreated provider listeners through
  mutation completion, observe authoritative owner and beneficiary refreshes, and
  prove the disposed controller performs no stale follow-up request.

### IR-014: UT-005, AT-014, IT-006, IT-007, IT-009

- Confirmed failure: Purchase, restore, and Customer Center controllers mapped a
  fully reconciled anomalous state to ordinary retryable pending.
- Implement: Each controller now returns a distinct state-bearing `anomaly`
  outcome when reconciliation fails with authoritative state. Request, read, and
  timeout failures remain retryable pending. The page maps anomaly outcomes to a
  support-required operation notice.
- Verification: Focused controller tests cover all three mutation paths, and a
  widget test proves the support message replaces pending copy.

### IR-007: AT-013, IT-008

- Confirmed failure: `_TierCard` selected one primary license and rendered only
  secondary licenses that were both assigned and inaccessible.
- Implement: Every remaining tier license now renders independently by its exact
  `subscriptionId`, including safe missing-correlation handling and only the
  assignment actions valid for that license state.
- Verification: Widget coverage uses two active unassigned Plus licenses and
  proves both products and both assignment actions are visible.

### IR-015: IT-013

- Updated the acceptance-test path to
  `appview/internal/routes/subscription_routes_test.go` and renamed the test to
  `TestIT013OwnerBillingStateUsesPrivateCamelCaseContract`.

### Fifth Correction Verification

- `flutter test test/subscriptions`: 161 tests passed.
- `flutter test`: 2,455 tests passed.
- Dart MCP static analysis: no diagnostics.
- `go test ./internal/routes -run
  '^TestIT013OwnerBillingStateUsesPrivateCamelCaseContract$'`: passed.
- `just test`: passed.
- `just appview-check`: all release gates passed; existing no-fix
  `GO-2026-5932` remains outside this feature.
- `flutter build ios --debug --simulator`: passed with the existing future Swift
  Package Manager plugin warning.
- `flutter build apk --debug`: passed with existing Kotlin migration warnings.
- `flutter build web`: passed with the existing icon-font warning.
- `git diff --check`: passed.
- Staging recipes and `MAN-001` through `MAN-005` remain blocked by absent
  `app/config/staging.env` and external RevenueCat/store configuration.

## Fourth Review Correction Pass: 2026-09-14

The repeated implementation review found four remaining gaps. Corrections remain
within the approved lifecycle, assignment, and owner-presentation requirements:

| Step | Status | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| IR-008 | Complete | IT-006, IT-007, IT-010, IT-014 | FR-017, FR-022, FR-026, NFR-004, RULE-006 | AC-018, AC-025 | Confirmed purchase and restore keep reconciliation-only intent through post-provider owner races. Production tests switch accounts while the baseline GET is unresolved and prove no second native operation. |
| IR-010 | Complete | AT-010, IT-009, IT-010 | FR-023, FR-026, NFR-004 | AC-005, AC-021, AC-025 | The first Customer Center mutation callback immediately persists owner-bound reconciliation intent before presentation completes. |
| IR-013 | Complete | AT-008, IT-008, IT-010 | FR-020, FR-021, FR-026, NFR-004 | AC-020, AC-025 | Assignment and unassignment reserve the shared operation provider; route disposal cannot permit a duplicate mutation or stale follow-up GET. |
| IR-007 | Complete | AT-013, IT-008 | FR-012, FR-019, FR-021 | AC-014, AC-019, AC-020 | Secondary dormant licenses render their correlated status, assignment, assignability, product, and timing. The production test now follows authoritative initial, released, and reassigned GET states. |

Each correction starts with one focused failing behavior test, applies the minimum
production change, reruns its nearby suite, and records evidence below.

### IR-008: IT-006, IT-007, IT-010, IT-014

- Confirmed failure: A stale owner after provider confirmation returned
  cancellation, allowing the route to clear the pending purchase or restore.
- Implement: Post-confirmation owner and polling races now return a
  reconciliation-only pending result. Pre-provider stale reads still cancel.
- Verification: Controller tests cover the owner race directly. Production page
  tests switch accounts while the post-confirmation baseline GET is unresolved,
  switch back, reconcile once, and retain exactly one purchase or restore call.

### IR-010: AT-010, IT-009, IT-010

- Confirmed failure: Customer Center recorded the mutation callback locally and
  did not persist shared intent until presentation completed.
- Implement: The first provider mutation callback immediately invokes the shared
  confirmation hook. Presentation completion and failures do not duplicate it.
- Verification: The production test leaves presentation unresolved through route
  disposal and an owner switch, then reconciles after switching back without
  reopening Customer Center.

### IR-013: AT-008, IT-008, IT-010

- Confirmed failure: Assignment and unassignment serialization was owned by the
  disposed route/controller instance.
- Implement: Added assignment and unassignment phases to the container-scoped
  pending-operation provider. All billing actions are disabled while either is
  reserved, and the controller checks route validity before its follow-up GET.
- Verification: Production tests hold each mutation unresolved across route
  disposal and recreation, prove one mutation, and prove no stale follow-up GET.

### IR-007: AT-013, IT-008

- Confirmed failure: The secondary dormant license rendered only an unassign
  action, omitting correlated state required to distinguish it from the active
  replacement. The production fake also advanced state from mutation callbacks
  instead of authoritative GET responses.
- Implement: Secondary dormant assignments now render state projected from their
  exact subscription ID, including status, assignment, assignability, store,
  product, and timing. The production fake publishes state only as GETs complete.
- Verification: Widget coverage proves both licenses are independently legible and
  actionable. The production flow consumes initial, released, released-baseline,
  and reassigned states across exactly four owner GETs.

### Fourth Correction Verification

- `flutter test test/subscriptions`: 156 tests passed.
- `flutter test`: 2,450 tests passed.
- Dart MCP static analysis: no diagnostics.
- `just test`: passed.
- `just appview-check`: all release gates passed; existing no-fix
  `GO-2026-5932` remains outside this feature.
- `flutter build ios --no-codesign`: passed with the existing future Swift
  Package Manager plugin warning.
- `flutter build apk`: passed with existing Kotlin migration and icon-font
  warnings.
- `flutter build web`: passed with the existing icon-font warning.
- `git diff --check`: passed.
- Staging recipes and `MAN-001` through `MAN-005` remain blocked by absent
  `app/config/staging.env` and external RevenueCat/store configuration.
