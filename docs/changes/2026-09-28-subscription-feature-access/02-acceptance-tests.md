# Acceptance Test Specification: Subscription Feature Access

## 1. Test Strategy

The product owner approved `01-requirements.md` for test design. Risk remains **High**: existing free capabilities and public profile projections become paid. Design tests against AppView-effective access for the authenticated DID, never billing-owner identity or device state. Start with a failing backend feature authorization test, then cover each write/read and public projection, lapse/restoration, worker behavior, and Flutter upgrade messaging. Use Go integration tests with isolated Postgres schemas for persistence and route checks; Flutter widget/provider tests for account-scoped UI and navigation. Mock provider reconciliation at the established boundary rather than running store transactions. Existing product-feature tests that assert unrestricted access must be updated to cover paid and unlicensed cases, not removed wholesale.

Existing suites: `appview/internal/subscriptions/subscription_acceptance_test.go`, `appview/internal/api/{profile_pin_test,profile_customisation_handler_test,saved_post_folder_request_test,follower_growth_response_test,business_account_type_acceptance_test,business_event_acceptance_test,scheduled_post_http_test}.go`, `appview/internal/scheduledposts/worker_acceptance_test.go`, `app/test/subscriptions/providers/subscription_access_provider_test.dart`, `app/test/saved_posts/pages/saved_posts_page_test.dart`, `app/test/settings/{profile_customisation_page_test,follower_growth_page_test}.dart`, `app/test/business/business_accessibility_test.dart`, and `app/test/scheduled_posts/scheduled_posts_page_test.dart`. The currently supported profile-customisation choices are colour/background; the retired border selector is out of scope (resolved GAP-001).

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-003, AC-015 | AT-001, AT-002, AT-005, IT-002, IT-003, IT-004, IT-005 | Acceptance / Integration | Yes |
| BR-002 | AC-001, AC-004 | AT-001, AT-003, IT-006 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-005 | AT-001, UT-001, IT-001, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| FR-002 | AC-002, AC-003, AC-006, AC-015 | AT-002, AT-005, UT-001, IT-002, IT-003, IT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-007, AC-008 | AT-007, UT-005, REG-003 | Acceptance / Unit / Regression | Yes |
| FR-004 | AC-004, AC-009 | AT-003, AT-008, IT-006, IT-007 | Acceptance / Integration | Yes |
| FR-005 | AC-010, AC-011 | AT-004, IT-007, IT-008, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-006 | AC-015, AC-016, AC-017 | AT-005, AT-006, UT-003, IT-003, IT-009, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-012, AC-018 | AT-009, UT-004, IT-011, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| FR-008 | AC-004, AC-011, AC-019 | AT-003, AT-004, UT-002, IT-006, IT-007, IT-012 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-013 | AT-007, MAN-001 | Acceptance / Manual | Partial |
| NFR-002 | AC-005, AC-014 | IT-001, REG-001, REG-002 | Integration / Regression | Yes |
| RULE-001 | AC-001, AC-004, AC-019 | AT-001, AT-003, UT-001, UT-002, IT-006 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-011 | AT-004, IT-008 | Acceptance / Integration | Yes |
| RULE-003 | AC-020 | AT-010, UT-001, IT-009, REG-006 | Acceptance / Unit / Integration / Regression | Yes |

Every AC-001–AC-020 is exercised by at least one AT/IT/REG/MAN below. GAP-001 and GAP-002 are resolved by the clarified current customisation scope and Business owner-route policy. A migration test covers removal of unused account-type storage.

## 3. Acceptance Scenarios

### AT-001: Per-account tier inheritance
Requirement IDs: BR-001, BR-002, FR-001, RULE-001
Acceptance Criteria: AC-001, AC-005
Priority: Must · Level: Acceptance · Automation Target: `app/test/subscriptions/subscription_feature_access_test.dart`

```gherkin
Feature: Account-specific benefits
  Scenario: Different DIDs on the same device have different paid features
    Given a billing owner has assigned Plus to Alice and Business to Bob
    And Carol on the same device has no active assigned licence
    When each account becomes active and opens a paid feature entry point
    Then Alice may use Plus features including follower growth but not Business features
    And Bob may use both Plus and Business features
    And Carol is offered an upgrade instead of access
    And switching accounts does not transfer any licence
```

### AT-002: Direct Plus action gates and free baseline
Requirement IDs: BR-001, FR-002
Acceptance Criteria: AC-002, AC-003, AC-006
Priority: Must · Level: Acceptance · Automation Target: `appview/internal/api/subscription_feature_access_acceptance_test.go`

```gherkin
Feature: Plus API permissions
  Scenario Outline: Paid actions depend on the account's effective tier
    Given a current member has effective <tier> access
    When they directly request <operation>
    Then the request <result> under the standard /v1 error contract when denied
    And any denied request has no side effects
    Examples:
      | tier     | operation                     | result   |
      | free     | create scheduled post         | fails    |
      | free     | create or modify saved folder | fails    |
      | free     | pin a post                    | fails    |
      | free     | read own follower growth      | fails    |
      | plus     | create scheduled post         | succeeds |
      | plus     | create or modify saved folder | succeeds |
      | plus     | pin a post                    | succeeds |
      | plus     | read own follower growth      | succeeds |
      | business | create scheduled post         | succeeds |
      | business | create or modify saved folder | succeeds |
      | business | pin a post                    | succeeds |
      | business | read own follower growth      | succeeds |
  Scenario: Ordinary activity remains free
    Given a Free member
    When they publish an ordinary post or view their flat saved-post list
    Then both operations remain available under existing rules
```

### AT-003: Business type and tools come from assigned licence
Requirement IDs: BR-002, FR-004, FR-008, RULE-001
Acceptance Criteria: AC-004, AC-019
Priority: Must · Level: Acceptance · Automation Target: `appview/internal/api/business_subscription_access_acceptance_test.go` and `app/test/business/business_subscription_gate_test.dart`

```gherkin
Feature: Paid business identity
  Scenario: Account type is derived without the former switch or table
    Given a member previously used the testing-only business switch and now has only Plus access
    When they visit account settings, attempt the former account-type mutation directly, or submit a business declaration or event
    Then their effective account type is regular
    And no account-type toggle or mutation route is available
    And products, CTA, details, and event writes are denied without changing records
    When a Business licence is assigned to that DID and gives access
    Then their effective account type becomes business without a manual switch
    And eligible business writes work under existing ownership and validation rules
    And attempting the former mutation to set regular is still unavailable
    And no live account-type table read is used to determine either status
```

### AT-004: Business lapse, restore, and source retention
Requirement IDs: FR-005, FR-008, RULE-002
Acceptance Criteria: AC-010, AC-011
Priority: Must · Level: Acceptance · Automation Target: `appview/internal/api/business_subscription_serving_acceptance_test.go`

```gherkin
Feature: CraftSky business serving
  Scenario: Existing business records are hidden until access returns
    Given a Business DID has a declaration, products, CTA, details, and events on its PDS
    When effective Business access ends
    Then its effective account type becomes regular
    And all CraftSky profile, event-list, and event-detail surfaces hide paid business content, including for the owner
    And the owner cannot reach Business management pages or management API routes
    And the authored source records remain intact
    When effective Business access returns
    Then the business identity, owner management routes, and eligible public content return without reauthoring
    And Free and Plus visitors may read the licensed public business content
    And existing membership and moderation exclusions still apply
```

### AT-005: Customisation is entirely Plus-only
Requirement IDs: BR-001, FR-002, FR-006
Acceptance Criteria: AC-015
Priority: Must · Level: Acceptance · Automation Target: `appview/internal/api/profile_customisation_subscription_test.go` and `app/test/settings/profile_customisation_page_test.dart`

```gherkin
Feature: Paid profile customisation
  Scenario: Saved choices survive a lapse but are not served without access
    Given Alice selected non-default profile colour and background while subscribed
    When Alice has no effective Plus or Business access
    Then an owner mutation is rejected and visitors see standard defaults on every visible identity surface
    And the saved choices are retained
    When access returns
    Then the saved choices are displayed again without another save
```

### AT-006: Saved posts flatten and pins clear on lapse
Requirement IDs: FR-006
Acceptance Criteria: AC-016, AC-017
Priority: Must · Level: Acceptance · Automation Target: `appview/internal/api/paid_feature_lapse_acceptance_test.go` and `app/test/saved_posts/pages/saved_posts_page_test.dart`

```gherkin
Feature: Plus lapse
  Scenario: Save organization survives but public pins do not
    Given saved posts exist in two folders and both pin slots are occupied
    When effective Plus-or-higher access ends
    Then the owner sees all saved posts in one flat list without folder grouping
    And the folders and membership associations remain stored
    And active pins are cleared and neither profile list promotes a pin
    When access returns
    Then folders and their saved posts return to their former positions
    And pin slots remain empty until the owner pins again
```

### AT-007: Locked Plus discovery and accessible upgrade
Requirement IDs: FR-003, NFR-001
Acceptance Criteria: AC-007, AC-008, AC-013
Priority: Must · Level: Acceptance · Automation Target: `app/test/subscriptions/subscription_feature_lock_test.dart`

```gherkin
Feature: Plus upgrade entry points
  Scenario Outline: Free member explores a Plus feature
    Given a Free member sees the <entry> entry point
    When they activate its overlaid, labelled lock affordance
    Then a dialog explains that <feature> requires Plus
    And a keyboard- and screen-reader-accessible learn-more action opens Settings subscriptions
    And no paid mutation occurs
    Examples:
      | entry                    | feature                |
      | scheduling              | scheduled posts        |
      | saved folders           | saved-post folders     |
      | profile customisation   | profile customisation  |
      | pin action              | pinned posts           |
      | follower growth         | follower growth        |
  Scenario: An eligible member uses the normal action
    Given the active DID has Plus or Business access
    When it opens the same entry points
    Then no subscription-required dialog intercepts them
```

### AT-008: Business section and management are licence-only
Requirement IDs: FR-004
Acceptance Criteria: AC-009
Priority: Must · Level: Acceptance · Automation Target: `app/test/business/business_subscription_gate_test.dart`

```gherkin
Feature: Business tools
  Scenario: A non-Business member has no Business management surface
    Given the active DID is Free or Plus
    When they view Settings or attempt to open a Business management route directly
    Then no Business section, menu item, or owner-management page is available
    And Business owner-management APIs deny reads, writes, and deletes without side effects
    And the normal subscription page remains available through Settings
    When the active DID gains effective Business access
    Then Business settings and management routes become available without Plus-style icon overlays
```

### AT-009: Scheduled publishing during lapse
Requirement IDs: FR-007
Acceptance Criteria: AC-012, AC-018
Priority: Must · Level: Acceptance · Automation Target: `appview/internal/scheduledposts/subscription_access_worker_test.go` and `app/test/scheduled_posts/scheduled_posts_page_test.dart`

```gherkin
Feature: Scheduled publication access
  Scenario: Due work is blocked without Plus access
    Given an owner's future scheduled items remain visible during a lapse with a subscription notice
    When an item becomes due while the owner lacks Plus-or-higher access
    Then no PDS publication occurs
    And it remains visible with a subscription-required error needing attention
    When access later returns
    Then the missed item does not publish automatically
    And a different still-future pending item may publish at its original time
```

### AT-010: Cancellation is not expiry
Requirement IDs: RULE-003
Acceptance Criteria: AC-020
Priority: Must · Level: Acceptance · Automation Target: `appview/internal/subscriptions/feature_access_lifecycle_test.go`

```gherkin
Feature: Effective access timing
  Scenario: Non-renewal keeps benefits through an accessible paid period
    Given the provider marks a licence will-not-renew but still gives access
    When the snapshot is reconciled
    Then its assigned DID retains paid actions, active pins, and Business identity if applicable
    When a later reconciled snapshot ends effective access
    Then lapse behavior applies exactly once
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-002, RULE-001, RULE-003 | AC-001, AC-002, AC-005, AC-020 | Effective-tier predicate uses assigned DID and `gives_access`, not purchaser or renewal intent. | Free, Plus, Business; cancelled-but-accessible; dormant assignment; same payer/multiple DIDs. | Plus allowed for Plus/Business; Business only for Business; cancellation alone changes nothing. | `appview/internal/subscriptions/feature_access_test.go` (proposed) |
| UT-002 | FR-008, RULE-001 | AC-004, AC-019 | Derived business label without a manual account-type store. | Free/Plus/Business DIDs with licence snapshots; historical stored flag present only in migration fixture. | Only effective Business resolves `business`; no application read queries the retired table. | `appview/internal/api/profile_account_type_hydrator_test.go` (extend) |
| UT-003 | FR-006 | AC-016, AC-017 | Lapsed saved-post projection and pin cleanup decision. | Posts in/out of folders; one/two pins; lapse and restoration. | Flat saved posts; membership preserved; pin selections cleared once and remain empty. | `appview/internal/api/paid_feature_lapse_test.go` (proposed) |
| UT-004 | FR-007 | AC-012, AC-018 | Scheduling access decision at due boundary. | Before/during/after scheduled time; licence lost/recovered. | Future remains pending; due-without-access becomes needs-attention; missed work never automatically claimed. | `appview/internal/scheduledposts/subscription_policy_test.go` (proposed) |
| UT-005 | FR-003 | AC-007, AC-008 | Account-scoped gate presentation including failed/unavailable access refresh. | Lease and Free/Plus/Business access response; pending/error; switched account. | Only matching active lease grants paid action; otherwise labelled upgrade/retry state, never another account's benefits. | `app/test/subscriptions/subscription_feature_access_test.dart` (proposed) |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, NFR-002 | AC-001, AC-005 | Shared billing owner, independently assigned Plus/Business DIDs and third Free DID. | Postgres licence fixtures with distinct authenticated sessions and same device. | Make paid requests as each DID and payer. | Only beneficiary's tier applies; old owner/membership checks stay intact. | `appview/internal/api/subscription_feature_access_acceptance_test.go` |
| IT-002 | FR-002 | AC-002, AC-006 | Scheduled create/update and manual publish access including direct HTTP. | Free and paid owner sessions; unsent scheduled payload. | Hit each mutation route. | Free gets standard error with no persisted or PDS side effect; paid retains existing validation rules. | `appview/internal/api/scheduled_post_http_test.go` (extend) |
| IT-003 | FR-002, FR-006 | AC-002, AC-006, AC-016 | Folder writes denied, flat saved list, retained folder associations, renewal restoration. | Mixed root/folder saved posts and folder rows. | Reconcile loss/restore and use saved/folder endpoints. | Free sees every saved post once without folder grouping; write denied; stored memberships return on restoration. | `appview/internal/api/saved_post_folder_subscription_test.go` (proposed) |
| IT-004 | FR-002, FR-006 | AC-002, AC-006, AC-017 | Pin mutation and two-slot lapse across profile lists. | Both pin slots populated and one already-rendered profile cursor. | Lose access, GET lists/pin state, retry direct PUT. | Slots removed, no promotion; direct mutation denied; old pin state cannot reappear on restore. | `appview/internal/api/profile_pin_lifecycle_test.go` (extend) |
| IT-005 | FR-002, BR-001 | AC-003, AC-006, AC-015 | Private follower growth and public customisation across embedded actor/profile reads. | Growth history and non-default saved colour/background. | Read own/other's metrics and public identity responses; attempt customisation write. | Free receives no metrics, sees standard customisation defaults, and cannot mutate customisation; paid gets permitted values; invalid access gives standard error. | `appview/internal/api/follower_growth_response_test.go` and `profile_customisation_hydrator_test.go` (extend) |
| IT-006 | BR-002, FR-004, FR-008, RULE-001 | AC-004, AC-019 | Business writes and retired account-type route. | Free/Plus/Business licences; former account-type records exist only before migration. | Call former account-type mutation as Free, Plus, and Business (attempt both `business` and `regular`); PUT declaration and POST/PUT event. | Former mutation route is unavailable for every tier; Free/Plus cannot write Business content; effective Business can under existing ownership/version checks. | `appview/internal/routes/business_routes_test.go` and `appview/internal/api/business_profile_commands_test.go` (extend) |
| IT-007 | FR-004, FR-005, FR-008 | AC-009, AC-010, AC-011 | Business identity, owner routes, public summary and event reads. | Declaration and events, owner/visitor sessions of each tier, distinct moderation states. | Reconcile access loss/restore; call owner list/get, PUT/DELETE, public profile summaries and event list/detail. | Unlicensed owner cannot use management reads/writes/deletes or see paid data; source records persist; Business restoration re-enables owner routes and eligible public content for visitors of any tier. | `appview/internal/api/business_eligibility_acceptance_test.go` and `appview/internal/routes/business_routes_test.go` (extend) |
| IT-008 | FR-005, RULE-002 | AC-011 | Suppressing public projections does not delete indexed source or PDS records. | Existing PDS-backed declaration/event and indexed raw source. | Reconcile lapse then renewal. | No PDS delete and source rows retained; re-serving needs no reauthoring. | `appview/internal/ingestion/business_records_integration_test.go` (extend) |
| IT-009 | FR-006, RULE-003 | AC-016, AC-017, AC-020 | Lifecycle transitions are keyed to effective access, not cancellation intent. | Plus and Business licences that become will-not-renew while `gives_access=true`, then lose access. | Reconcile snapshots in order, replay access-loss reconciliation and restore later. | No early loss, one pin clear on lapse; repeated/later reconciliation cannot resurrect pins; folders/customisation retained. | `appview/internal/subscriptions/feature_access_lifecycle_test.go` (proposed) |
| IT-010 | FR-006 | AC-015, AC-016, AC-017 | Lapse and restore do not leak cached Flutter state from another account. | Retained sessions, non-default customisation, populated folders/pins. | Switch DID during pending access refresh, then restore first DID. | Correct defaults/flat list/no pins per DID; restored folder membership and customisation for correct DID only. | `app/test/subscriptions/subscription_invalidation_regression_test.dart` (extend) |
| IT-011 | FR-007 | AC-012, AC-018 | Worker publication check, claim-time race, and owner-visible error. | Isolated scheduled-post store, frozen clock, recording fake PDS, active/dormant licences and a controllable worker pause. | Process due batch while unlicensed, restore and process again; pause another job after claim, end its owner's access, then resume at the last pre-PDS boundary; process future item after restoration. | No PDS call after access loss (including already-claimed work), no auto-backfill; missed items need attention; eligible future item may publish at due time. | `appview/internal/scheduledposts/worker_acceptance_test.go` (extend) |
| IT-012 | FR-008 | AC-019 | Forward migration drops testing-only account-type storage after read/cleanup replacement. | Apply historical migrations through the new migration; seed legacy `business`/`regular` rows before upgrade; assign a Business licence to another DID. | Migrate up, verify table absent, exercise public profile/summary/event reads and permanent account deletion. | No live application SQL references the dropped table; historical migration tests remain valid; effective type follows licence, PDS business source persists, deletion cleanup still succeeds; down migration follows repository rollback conventions. | `appview/internal/db/business_profiles_migration_test.go` and `appview/internal/accountdeletion/business_records_acceptance_test.go` (extend) |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Assigned licence does not follow device, payer, or active-account switch. | FR-001, NFR-002 | AC-005 | Re-run subscription assignment/access tests with Plus and Business assigned to different DIDs on one device; include inactive owner. |
| REG-002 | Chronological feed/search/discovery and moderation unaffected by payment. | NFR-002 | AC-014 | Re-run `business_non_entitlement_acceptance_test.go` neutrality tests and pin profile-only ordering tests with tier changes. |
| REG-003 | Ordinary posting, saving, and settings navigation remain available to Free. | FR-003 | AC-007, AC-008 | Widget/API tests ensure lock overlay never intercepts baseline actions, paywall CTA reaches existing subscription route. |
| REG-004 | Blocked/moderated/non-member business records are not newly served on restoration. | FR-005, RULE-002 | AC-010, AC-011 | Existing business eligibility/redaction tests with effective Business licence, plus post-lapse restore and Free/Plus visitor checks; unlicensed owner cannot use management routes. |
| REG-005 | Existing scheduled publication safety and state transitions. | FR-007 | AC-012, AC-018 | Preserve worker retries, idempotent PDS publication and owner-visible needs-attention state with tier-check added. |
| REG-006 | Cancellation retains current paid period access. | RULE-003 | AC-020 | Existing subscription reconciliation/assignment tests followed by feature/pin/business-type assertions until `gives_access=false`. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Multi-account/tier isolation | Billing owner DID; distinct Free, Plus, Business DIDs; same device; separate assigned licence IDs and sessions. | AT-001–AT-004, IT-001, IT-006, REG-001 |
| TD-002 | Lifecycle | Snapshot sequences: active, cancelled-but-`gives_access=true`, expired/false, restored same subscription/true, repeated stale snapshot; separate test for grace with access. | AT-004, AT-006, AT-010, IT-007–IT-011 |
| TD-003 | Saved-post/pin restoration | Two folders with multiple saved posts plus root posts, both profile pin slots, old pinned-content cursor. | AT-006, IT-003, IT-004, IT-009, IT-010 |
| TD-004 | Public business serving/migration | Existing PDS declaration with product, CTA, location/hours; future and past events, hidden and moderated events; legacy `business`/`regular` rows seeded before migration; former mutation requests setting both values. | AT-003, AT-004, IT-006–IT-008, IT-012, REG-004 |
| TD-005 | Scheduling boundary | Pending future, due during lapse, and already published items; frozen clock and recording PDS. | AT-009, IT-002, IT-011, REG-005 |
| TD-006 | Customisation/metrics | Non-default saved colour/background; default values; owner-only follower history; profile identity embedded in post/search/notification. | AT-002, AT-005, IT-005, IT-010 |
| TD-007 | UI and access uncertainty | Free/Plus/Business matching leases; pending/error access, switched account; focus and semantics tree. | AT-001, AT-007, AT-008, UT-005, MAN-001 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-001 | AC-013 | Platform assistive technology and focus order. | On iOS VoiceOver/Android TalkBack and keyboard, visit each locked Plus control, open its dialog, activate the CTA; while signed into a Business DID also check the Business section at larger text sizes. | Required tier and feature announced, logical focus/return behavior, usable without interpreting the lock icon alone. Automated semantics/widget assertions remain primary coverage. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | **Resolved:** former border mismatch. | BR-001, FR-002, FR-006 | `01-requirements.md` Q3/AC-015 now limit paid customisation to currently supported colour/background; the retired border remains out of scope. | Test every supported colour/background option; no border test or scope extension. |
| GAP-002 | **Resolved for Business:** owner-management routes (read/write/delete) are inaccessible without effective Business access; source PDS records are retained. Scheduled-post owner reads remain available so users see the required error. | FR-004, FR-005, FR-007 | Product-owner coding-plan feedback fixes the formerly open management policy. | IT-007 covers Business owner route denial and restoration; IT-011 covers owner-visible scheduled-post status. Do not suppress public visitor reads of a valid licensed business. |
| GAP-003 | **Coverage designed, implementation risk remains:** race between reconciliation and an already claimed worker or pin cleanup. | FR-006, FR-007, RULE-003 | A unit-only gate could miss publication or removal during a state transition. | IT-009 replays cleanup and checks no resurrection; IT-011 interleaves claim → access loss → last pre-PDS boundary and asserts no write. Coding plan must identify the final check and any unavoidable race window. |

## 10. Out Of Scope

- Store prices, RevenueCat dashboard configuration, checkout, purchase/restore UI, entitlement creation, and subscription assignment rules: already covered by earlier subscription stages.
- New business PDS schemas or deletion of source records: outside this feature and prohibited on ordinary lapse.
- Manual store sandbox purchase testing: feature access is based on simulated reconciled AppView state and existing purchase tests, not a new purchase flow.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md` (approved for test design by product owner in this conversation)
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if user initiates outside this skill: `docs/changes/2026-09-28-subscription-feature-access/`
- Recommended first failing test for implementation: `UT-001` tier eligibility and cancellation-before-expiry; then `IT-001` direct DID-scoped API enforcement.
- Suggested test order for implementation: UT-001/UT-002 → IT-001/IT-002–IT-006 → IT-007/IT-008 → UT-003/IT-009/IT-010 → UT-004/IT-011 → Flutter AT-001–AT-010 → regression and accessibility checks.
- Commands discovered: `just test`, `just appview-check`, `just app-test`, `just app-analyze`; focused Go: `go test ./internal/api ./internal/subscriptions ./internal/scheduledposts` from `appview/` with test Postgres; focused Flutter: `flutter test test/subscriptions test/settings test/business test/saved_posts test/scheduled_posts` from `app/`.
- Blocking gaps: None identified after resolving GAP-001 and GAP-002. The mandatory account-type table removal is covered by IT-012. High-risk work requires explicit approval before implementation.
