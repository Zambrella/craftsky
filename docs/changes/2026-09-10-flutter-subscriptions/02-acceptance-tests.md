# Acceptance Test Specification: Flutter Account Subscriptions

## 1. Test Strategy

This specification tests the Flutter subscription slice at five levels:

- Acceptance/widget tests exercise owner setup, purchase coordination,
  reconciliation, assignment, tier presentation, account switching, restore,
  management, failure, and accessibility through replaceable AppView and
  RevenueCat boundaries.
- Unit tests cover strict wire models, billing-owner state, product eligibility,
  result mapping, reconciliation polling, assignment candidates, stale-operation
  fencing, and privacy-safe diagnostics.
- Integration tests cover exact AppView routes and camelCase payloads, retained
  account session scoping, secure owner persistence, router/settings wiring, and
  coordination across the RevenueCat adapter and AppView repository.
- Regression tests protect free feature availability, multi-account isolation,
  web startup, normal cancellation, and existing error/privacy behavior.
- Manual sandbox checks are limited to behavior that Flutter tests cannot
  credibly emulate: native RevenueCatUI rendering, real App Store and Google Play
  transactions, restore ownership, external provider management, and native
  lifecycle behavior.

The RevenueCat SDK is never the test authority for a DID's paid access. Automated
tests must establish that only AppView self-access can publish an effective tier.
Tests must also distinguish store purchase completion from CraftSky account
activation: a new license requires reconciliation, explicit assignment, and a
matching AppView access refresh; a same-subscription recovery preserves its
existing assignment but still requires matching AppView access.

Risk level remains High. The product owner approved `01-requirements.md` on
2026-09-10. Workflow document review is required before implementation.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001 | AT-002, IT-006, IT-008, MAN-001, MAN-002 | Acceptance / Integration / Manual | Partial |
| BR-002 | AC-002, AC-003 | AT-003, UT-001, IT-010, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| BR-003 | AC-004 | AT-012, REG-001, REG-003 | Acceptance / Regression | Yes |
| BR-004 | AC-005 | AT-009, AT-010, IT-007, IT-009, MAN-003, MAN-004 | Acceptance / Integration / Manual | Partial |
| FR-001 | AC-006 | UT-003, IT-012, MAN-001, MAN-002 | Unit / Integration / Manual | Partial |
| FR-002 | AC-004, AC-006 | AT-012, UT-003, REG-003 | Acceptance / Unit / Regression | Yes |
| FR-003 | AC-007, AC-011 | AT-001, AT-011, AT-017, UT-002, UT-013, IT-001, IT-004, IT-014 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-008 | AT-001, AT-017, UT-013, IT-003, IT-005, IT-007, IT-009, IT-014 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-008 | UT-002, IT-003, REG-006 | Unit / Integration / Regression | Yes |
| FR-006 | AC-003, AC-009, AC-011 | AT-003, AT-011, UT-002, IT-004, IT-010, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| FR-007 | AC-009, AC-010 | AT-004, IT-002, IT-011 | Acceptance / Integration | Yes |
| FR-008 | AC-011 | AT-011, AT-017, UT-002, UT-008, UT-013, IT-004, IT-010, IT-014, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| FR-009 | AC-002, AC-012 | AT-004, AT-013, UT-001, UT-012, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-013 | AT-015, IT-002, IT-011 | Acceptance / Integration | Yes |
| FR-011 | AC-013 | AT-015, IT-002, IT-010, REG-002 | Acceptance / Integration / Regression | Yes |
| FR-012 | AC-014, AC-029 | AT-013, UT-010, UT-012, IT-001, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-015 | AT-005, UT-004 | Acceptance / Unit | Yes |
| FR-014 | AC-016, AC-022 | AT-006, AT-014, UT-006, IT-005, MAN-001, MAN-002 | Acceptance / Unit / Integration / Manual | Partial |
| FR-015 | AC-016, AC-017 | AT-006, UT-006, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-017 | AT-006, REG-004 | Acceptance / Regression | Yes |
| FR-017 | AC-001, AC-018 | AT-002, AT-007, UT-005, UT-011, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-018 | AC-001, AC-002 | AT-002, AT-003, IT-006 | Acceptance / Integration | Yes |
| FR-019 | AC-001, AC-019 | AT-002, AT-008, UT-007, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-020 | AC-019, AC-020 | AT-008, UT-008, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-021 | AC-020 | AT-008, IT-008 | Acceptance / Integration | Yes |
| FR-022 | AC-005, AC-018 | AT-009, IT-007, MAN-003 | Acceptance / Integration / Manual | Partial |
| FR-023 | AC-005, AC-021 | AT-010, IT-009, MAN-004 | Acceptance / Integration / Manual | Partial |
| FR-024 | AC-018, AC-022 | AT-006, AT-007, AT-014, UT-006, UT-010 | Acceptance / Unit | Yes |
| FR-025 | AC-014 | AT-013 | Acceptance | Yes |
| FR-026 | AC-010, AC-011, AC-013, AC-025 | AT-003, AT-004, AT-011, AT-015, AT-017, UT-008, UT-013, IT-010, IT-014, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| FR-027 | AC-029 | AT-013, UT-012, IT-001, IT-013 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-023 | UT-012, IT-001, IT-002 | Unit / Integration | Yes |
| NFR-002 | AC-024 | UT-009, REG-006 | Unit / Regression | Yes |
| NFR-003 | AC-006 | UT-003, IT-012 | Unit / Integration | Yes |
| NFR-004 | AC-020, AC-025 | AT-008, AT-011, AT-017, UT-008, UT-013, IT-010, IT-014, MAN-005 | Acceptance / Unit / Integration / Manual | Partial |
| NFR-005 | AC-026 | AT-016, MAN-005 | Acceptance / Manual | Partial |
| NFR-006 | AC-027 | REG-007 | Regression | Yes |
| NFR-007 | AC-027, AC-028 | REG-007, MAN-001, MAN-002, MAN-003, MAN-004 | Regression / Manual | Partial |
| NFR-008 | AC-018, AC-025 | AT-007, UT-011, IT-010 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-002 | AT-003, UT-001, IT-006 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-007, AC-009, AC-011 | AT-001, AT-003, AT-011, UT-002, UT-013, IT-004, IT-014 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-019 | AT-008, UT-007 | Acceptance / Unit | Yes |
| RULE-004 | AC-001, AC-019 | AT-002, AT-008, UT-004, UT-007 | Acceptance / Unit | Yes |
| RULE-005 | AC-001, AC-005 | AT-002, AT-009, IT-006, IT-007 | Acceptance / Integration | Yes |
| RULE-006 | AC-018, AC-022 | AT-007, AT-014, UT-005 | Acceptance / Unit | Yes |
| RULE-007 | AC-006 | AT-012, UT-003, REG-003 | Acceptance / Unit / Regression | Yes |
| RULE-008 | AC-009, AC-011 | AT-003, AT-009, AT-011, IT-004, IT-010 | Acceptance / Integration | Yes |
| RULE-009 | AC-004 | AT-012, REG-001 | Acceptance / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Explicit Billing Owner Setup

Requirement IDs: FR-003, FR-004, RULE-002

Acceptance Criteria: AC-007, AC-008

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/billing_owner_setup_test.dart`

```gherkin
Feature: Billing owner setup
  Scenario: The current account explicitly becomes the billing owner
    Given Alice is the active authenticated account
    And no billing owner is reserved on the installation
    When Alice opens subscriptions but has not confirmed setup
    Then no billing-account ensure request is made
    When Alice confirms that this account will own billing
    Then Flutter durably reserves Alice's DID before any network call
    And Flutter ensures Alice's AppView billing account exactly once
    And Flutter durably adds the returned UUID to Alice's reservation
    And only then RevenueCat is identified with that UUID
    And billing actions remain disabled until that identity is confirmed
```

### AT-002: Purchase, Reconcile, And Assign

Requirement IDs: BR-001, FR-017, FR-018, FR-019, RULE-004, RULE-005

Acceptance Criteria: AC-001

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_purchase_flow_test.dart`

```gherkin
Feature: Account-assigned subscription purchase
  Scenario Outline: The owner assigns a newly purchased license
    Given Alice is the identified billing owner
    And Alice and Bob have current retained sessions on the same device
    And the <tier> offering is available and not already owned
    When Alice completes the <tier> paywall purchase
    Then Flutter requests AppView reconciliation
    And no account is treated as <tier> from RevenueCat CustomerInfo
    When AppView returns a new unassigned <tier> license
    And Alice selects Bob and confirms assignment
    Then Flutter assigns that license to Bob
    And Bob's refreshed AppView self-access displays <tier>
    And Alice is not automatically assigned the license

    Examples:
      | tier     |
      | Plus     |
      | Business |

  Scenario: The same provider subscription regains access
    Given Alice owns an inaccessible Plus subscription whose license is assigned to Bob
    And the owner state is fresh, fully reconciled, non-pending, will_not_renew, and anomaly-free
    When Alice repurchases Plus and RevenueCat restores access to that subscription
    And the post-baseline AppView generation reconciles
    Then Flutter preserves the existing license and Bob assignment
    And Flutter does not wait for a new license or open assignment
    And activation completes only when Bob's AppView self-access displays Plus

  Scenario: Repurchase produces a new provider subscription
    Given Alice owns an inaccessible Plus subscription whose license is assigned to Bob
    And the owner state is fresh, fully reconciled, non-pending, will_not_renew, and anomaly-free
    When Alice repurchases Plus and AppView exposes a different subscription and license ID
    Then the old dormant assignment remains unchanged
    And the new license remains unassigned
    And Flutter requires explicit assignment before any DID is activated
```

### AT-003: AppView Authority Survives Account Switching

Requirement IDs: BR-002, FR-006, FR-018, FR-026, RULE-001, RULE-008

Acceptance Criteria: AC-002, AC-003, AC-009

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_identity_boundary_test.dart`

```gherkin
Feature: Billing identity and access authority
  Scenario: Switching to a beneficiary does not switch RevenueCat identity
    Given RevenueCat is identified with billing owner Alice's UUID
    And the billing customer's RevenueCat CustomerInfo has a Business entitlement
    But AppView reports Bob as free
    When the active account switches from Alice to Bob
    Then Flutter does not call RevenueCat login or logout
    And Bob displays the AppView effective tier free
    When AppView later reports Bob as Business
    Then Bob displays Business without changing RevenueCat identity
```

### AT-004: Beneficiary Access And Billing Privacy

Requirement IDs: FR-007, FR-009, FR-026

Acceptance Criteria: AC-010, AC-012

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/beneficiary_subscription_view_test.dart`

```gherkin
Feature: Beneficiary subscription view
  Scenario: A non-owner sees only its account access
    Given Bob is active and receives an inaccessible dormant Plus assignment
    And Alice is the retained billing owner
    When Bob opens subscriptions
    Then Bob sees effective tier free and assigned tier Plus
    And Bob sees no billing UUID, product, provider status, other assignment, restore, purchase, or management data
    And Bob can choose to switch to Alice for billing management
```

### AT-005: Safe Purchase And Repurchase Eligibility

Requirement IDs: FR-013, RULE-004

Acceptance Criteria: AC-015

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_purchase_eligibility_test.dart`

```gherkin
Feature: Purchase eligibility
  Scenario Outline: Unsafe state cannot start another purchase
    Given Alice is the billing owner
    And the requested tier has <state>
    When the owner page renders
    Then the purchase action for that tier is unavailable
    And the app does not present a RevenueCat paywall for that tier

    Examples:
      | state                                                   |
      | an accessible subscription                              |
      | a pending-payment subscription                          |
      | an inaccessible subscription with omitted renewal      |
      | an inaccessible subscription with empty renewal        |
      | an inaccessible subscription that will renew           |
      | an inaccessible subscription that will change product  |
      | an inaccessible subscription that will pause           |
      | an inaccessible subscription awaiting price consent    |
      | an inaccessible subscription that already renewed      |
      | an inaccessible subscription whose renewal is unknown  |
      | a stale or pending reconciliation                      |
      | a subscription or license anomaly                      |
      | an unknown subscription or license anomaly             |

  Scenario: A fully reconciled lapsed tier can be purchased again
    Given Alice is the billing owner
    And every subscription correlated to Plus is inaccessible
    And none has pending payment
    And every auto-renewal status is exactly will_not_renew
    And no correlated subscription or license has an anomaly
    When the owner page renders
    Then the Plus purchase action is available
    And any dormant assignment is disclosed and remains unchanged

  Scenario: A tier with no owned subscription can be purchased
    Given Alice is the billing owner
    And the fresh owner state has no subscription correlated to Business
    When the owner page renders
    Then the Business purchase action is available
```

### AT-006: Tier-Specific Paywall Outcomes

Requirement IDs: FR-014, FR-015, FR-016, FR-024

Acceptance Criteria: AC-016, AC-017, AC-022

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_paywall_test.dart`

```gherkin
Feature: Tier-specific RevenueCat paywalls
  Scenario Outline: Each provider result has one deterministic outcome
    Given Alice is identified as the billing owner
    And the selected tier's named offering is available
    When the RevenueCat paywall returns <result>
    Then the named offering is presented exactly once with dismissal available
    And Flutter records the <handling> outcome
    And Flutter never issues a second manual purchase

    Examples:
      | result       | handling                              |
      | purchased    | begin AppView reconciliation          |
      | restored     | begin AppView reconciliation          |
      | cancelled    | return with no state mutation         |
      | error        | show a retryable provider failure     |

  Scenario: An unavailable named offering never presents RevenueCatUI
    Given Alice is identified as the billing owner
    And the selected tier's named offering or required package is missing
    When Alice attempts to open its paywall
    Then Flutter displays pre-presentation unavailability
    And RevenueCatUI is not called
    And Flutter does not report purchase success

  Scenario: An invalid attached paywall fails during presentation
    Given Alice is identified as the billing owner
    And the selected tier's named offering and required package are available
    But its attached paywall is missing or invalid
    When Alice attempts to open its paywall
    Then RevenueCatUI presentation returns or throws a provider failure
    And Flutter displays retryable provider failure without purchase success
    And Flutter never issues a manual purchase
```

### AT-007: Bounded Reconciliation Wait

Requirement IDs: FR-017, FR-024, NFR-008, RULE-006

Acceptance Criteria: AC-018, AC-022, AC-025

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_reconciliation_flow_test.dart`

```gherkin
Feature: Post-provider reconciliation
  Scenario: A delayed AppView result remains pending without repurchase
    Given RevenueCat has returned purchased
    And Flutter records requested generation 7 before requesting reconciliation
    And AppView accepts the reconciliation request
    But Flutter does not observe a requested generation above 7 within the bounded wait
    When the wait reaches its timeout
    Then polling stops
    And the app displays pending billing rather than activated access
    And the app offers refresh without reopening checkout

  Scenario: A concurrent reconciliation trigger cannot cause early activation
    Given Flutter records requested generation 7 before requesting reconciliation
    When a webhook advances requested generation to 8
    And the owner reconciliation advances it to 9 before Flutter reads again
    Then Flutter uses the first observed post-baseline generation 9 as its target
    And generation 8 being reconciled does not complete the flow
    When reconciled generation reaches at least 9
    Then purchase still waits for a recognized new-license or same-subscription state

  Scenario: Restore may reconcile without changing billing state
    Given Flutter records the owner baseline and requests reconciliation after restore
    When a post-baseline requested generation is observed and reconciled
    But AppView billing state is unchanged
    Then restore completes as no change
    And purchase would remain pending under the same responses
```

### AT-008: Assignment, Reassignment, And Unassignment

Requirement IDs: FR-019, FR-020, FR-021, NFR-004, RULE-003, RULE-004

Acceptance Criteria: AC-019, AC-020

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_assignment_flow_test.dart`

```gherkin
Feature: License assignment management
  Scenario Outline: AppView assignment outcomes preserve server truth
    Given Alice owns an assignable Plus license
    And the current server assignment is Bob
    When Alice confirms an assignment operation that returns <result>
    Then the app displays <message>
    And duplicate taps produce only one mutation
    And the refreshed server assignment, not an optimistic local value, is displayed

    Examples:
      | result                         | message                               |
      | success to Carol               | Carol is assigned                     |
      | assignment_target_ineligible   | target account is unavailable         |
      | assignment_conflict            | target already has a paid assignment  |
      | assignment_cooldown            | assignment cannot yet be changed      |
      | billing_license_not_found      | license state must be refreshed       |
      | successful unassignment        | license is unassigned                 |

  Scenario Outline: Exact errors are preserved for both mutation routes
    Given Alice owns a Plus license
    When the <operation> route returns <error>
    Then Flutter maps that exact error rather than only its HTTP status
    And refreshed server assignment remains authoritative

    Examples:
      | operation    | error                          |
      | assignment   | billing_license_not_found      |
      | assignment   | assignment_target_ineligible   |
      | assignment   | assignment_conflict            |
      | assignment   | assignment_cooldown            |
      | unassignment | billing_license_not_found      |

  Scenario: Owner authorization failure uses normal session invalidation
    Given Alice owns an assignable Plus license
    When an assignment request returns owner HTTP 401
    Then Alice's existing session invalidation path runs
    And Flutter does not classify the response as target ineligibility
```

### AT-009: Restore Uses The Existing Owner

Requirement IDs: BR-004, FR-022, RULE-005, RULE-008

Acceptance Criteria: AC-005, AC-018

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_restore_flow_test.dart`

```gherkin
Feature: Restore subscriptions
  Scenario: Restore discovers licenses without moving assignments
    Given Alice is active and RevenueCat is identified with Alice's existing billing UUID
    And a known Plus license is assigned to Bob
    When Alice explicitly restores purchases
    Then no anonymous or replacement RevenueCat identity is used
    And Flutter requests AppView reconciliation
    And Bob's known assignment is preserved
    And any newly discovered Business license remains unassigned
```

### AT-010: Customer Center Management Refresh

Requirement IDs: BR-004, FR-023

Acceptance Criteria: AC-005, AC-021

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_management_flow_test.dart`

```gherkin
Feature: Provider subscription management
  Scenario Outline: Returning from provider management refreshes AppView
    Given Alice is active and identified as the billing owner
    When Alice opens Customer Center and <action>
    Then Customer Center uses Alice's billing UUID
    And Flutter requests AppView reconciliation on return
    And the owner billing page refreshes

    Examples:
      | action                                      |
      | completes the Customer Center restore       |
      | selects a management option                 |
      | returns from external store management      |
      | dismisses after a management action         |
```

### AT-011: Owner Session Loss Fences Active Work

Requirement IDs: FR-003, FR-008, FR-026, NFR-004, RULE-002, RULE-008

Acceptance Criteria: AC-011, AC-025

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_owner_session_loss_test.dart`

```gherkin
Feature: Billing owner session lifecycle
  Scenario: Owner removal invalidates a pending billing operation
    Given Alice is the selected billing owner
    And a reconciliation refresh for Alice is pending
    When Alice's retained session is removed before the refresh completes
    Then the owner binding remains pinned to Alice's DID and UUID but is unusable
    And the stale completion publishes no billing state or navigation
    And purchase, restore, reconciliation, assignment, unassignment, and management are disabled
    And RevenueCat login or logout is not called
    And Bob cannot be selected or ensured as a replacement owner

  Scenario: Only the exact original owner can resume billing
    Given Alice's DID and UUID remain pinned after her session was removed
    When Bob is active and confirms billing setup
    Then setup is rejected without an AppView ensure or RevenueCat identity call
    When Alice reauthenticates and a billing-account GET returns the pinned UUID
    Then billing may resume under Alice
    And no billing-account ensure request is made
    When billing-account GET instead returns 404 or a different UUID
    Then billing remains locked for recovery
    And no billing-account ensure or RevenueCat identity mutation is made

  Scenario: Setup stops after the DID reservation but before ensure completes
    Given Alice confirms first-time setup on an unreserved installation
    And Alice's DID is durably reserved before any network call
    When setup stops before a billing UUID is persisted
    Then Alice's DID remains reserved with no UUID after restart
    And Bob cannot ensure or replace the reservation
    And only active Alice may retry the idempotent billing-account PUT

  Scenario: Setup stops after UUID persistence but before RevenueCat identity
    Given Alice's ensure returns UUID U
    And the reservation is durably updated to Alice and U
    When setup stops before RevenueCat identity work completes
    Then Alice and U remain reserved after restart
    And recovery uses billing-account GET only
    And no billing-account PUT or replacement owner selection is permitted
```

### AT-012: Billing Is Optional To Free Use

Requirement IDs: BR-003, FR-002, RULE-007, RULE-009

Acceptance Criteria: AC-004, AC-006

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_optional_feature_test.dart`

```gherkin
Feature: Optional native billing
  Scenario Outline: Unsupported billing does not restrict CraftSky
    Given the app is running with <condition>
    When a signed-in free account uses CraftSky
    Then sign-in and all existing social and publishing features remain available
    And AppView access can still be displayed
    And native purchase, restore, and Customer Center actions are disabled

    Examples:
      | condition                       |
      | a missing RevenueCat public key |
      | a blank RevenueCat public key   |
      | the web platform                |
      | a desktop platform              |
```

### AT-013: Owner Billing State Presentation

Requirement IDs: FR-009, FR-012, FR-025

Acceptance Criteria: AC-012, AC-014

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_settings_page_test.dart`

```gherkin
Feature: Owner billing state
  Scenario: Plus and Business are joined by internal subscription ID
    Given Alice owns subscription sub-business and its unassigned license is first in the response
    And Alice owns subscription sub-plus and its Bob-assigned license is second in the response
    And the subscription array is returned in the opposite order
    And Business has stale reconciliation
    When Alice opens the owner billing page
    Then Plus and Business appear separately
    And each shows its assignment, access, safe provider status, and timing
    And Business shows its stale state
    And the page explains that switching accounts does not move either subscription
    And no license is matched by array position or provider subscription identifier
```

### AT-014: Distinct Non-Success States

Requirement IDs: FR-014, FR-024, RULE-006

Acceptance Criteria: AC-018, AC-022

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_error_states_test.dart`

```gherkin
Feature: Subscription failure states
  Scenario Outline: Billing problems are not shown as activation
    Given the subscription flow encounters <condition>
    When the page renders the result
    Then it displays the localized <response>
    And it does not display successful CraftSky activation

    Examples:
      | condition                 | response                    |
      | missing offering          | unavailable and retry       |
      | provider error            | provider failure and retry  |
      | reconciliation timeout    | pending and refresh         |
      | stale AppView state        | stale and refresh           |
      | assignment anomaly        | support-oriented state      |
```

### AT-015: Account Switcher Tier Isolation

Requirement IDs: FR-010, FR-011, FR-026

Acceptance Criteria: AC-013

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/account_switcher_tier_test.dart`

```gherkin
Feature: Retained account tier badges
  Scenario: Each row owns its access result
    Given Alice, Bob, and Carol are retained on the installation
    And AppView reports Alice as Plus, Bob as Business, and Carol's request as unavailable
    When the account switcher opens while Alice remains active
    Then Alice displays Plus
    And Bob displays Business
    And Carol displays unavailable rather than another account's tier
    And loading these values does not activate Bob or Carol or change the route
```

### AT-016: Accessible Subscription Decisions

Requirement IDs: NFR-005

Acceptance Criteria: AC-026

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_accessibility_test.dart`

```gherkin
Feature: Accessible subscription UI
  Scenario: Critical billing actions remain usable with accessibility settings
    Given a narrow supported screen with large text and screen-reader semantics
    When the owner reviews, purchases, assigns, restores, retries, and manages a subscription
    Then every critical status and action has an understandable semantic label
    And controls remain tappable without clipped or unreachable content
    And platform back or dismiss returns to the owner page
```

### AT-017: Every Billing Operation Requires The Active Owner

Requirement IDs: FR-003, FR-004, FR-008, FR-026, NFR-004

Acceptance Criteria: AC-007, AC-008, AC-011, AC-025

Priority: Must

Level: Acceptance

Automation Target: `app/test/subscriptions/subscription_active_owner_guard_test.dart`

```gherkin
Feature: Operation-level billing owner preconditions
  Scenario Outline: A beneficiary cannot start an owner external call
    Given Alice is the pinned identified billing owner
    And Bob is the active retained account
    When Bob attempts <call>
    Then the operation is rejected before any RevenueCat or AppView billing call
    And Flutter offers to switch to Alice

    Examples:
      | call                                  |
      | owner billing GET                     |
      | RevenueCat offering retrieval         |
      | RevenueCat identity inspection/login  |
      | direct paywall presentation           |
      | restore                               |
      | Customer Center                       |
      | reconciliation POST                   |
      | reconciliation polling GET            |
      | assignment                            |
      | unassignment                          |

  Scenario Outline: Owner loss between confirmation and dispatch cancels the call
    Given Alice is active and confirms <call>
    When Alice's lease becomes stale before dispatch
    Then no RevenueCat or AppView billing call is made
    And no completion can publish owner state

    Examples:
      | call                                  |
      | owner billing GET                     |
      | RevenueCat offering retrieval         |
      | RevenueCat identity inspection/login  |
      | direct paywall presentation           |
      | restore                               |
      | Customer Center                       |
      | reconciliation POST                   |
      | reconciliation polling GET            |
      | assignment                            |
      | unassignment                          |

  Scenario: First setup and pinned recovery use different account methods
    Given Alice is active on a never-reserved installation
    When Alice confirms first setup
    Then Alice's DID is durably reserved before any network call
    And one billing-account PUT may be dispatched only after the reservation succeeds
    When that PUT returns UUID U
    Then U is durably added to Alice's reservation before RevenueCat identity work
    Given Alice's DID and UUID are already pinned but her session was replaced
    When Alice attempts recovery
    Then only billing-account GET may be dispatched
    And GET 404 or UUID mismatch remains locked without PUT
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | BR-002, FR-009, RULE-001 | AC-002, AC-012 | Parse and project canonical self-access tiers without consulting RevenueCat state. | Free, Plus, Business, dormant Plus, invalid tier fixture. | Valid values preserve effective/assigned semantics; invalid tier fails strict decoding. | `app/test/subscriptions/models/subscription_access_test.dart` |
| UT-002 | FR-003, FR-005, FR-006, RULE-002 | AC-007, AC-008, AC-009, AC-011 | Evaluate reserved billing-owner selection and identity rules. | Never-reserved, DID-only reservation, DID/UUID reservation, removed owner, reauthenticated same DID/same UUID, same DID/different UUID, active beneficiary, anonymous SDK ID. | Explicit setup reserves DID before I/O; only the same DID may retry PUT while UUID is absent; UUID persistence makes recovery GET-only; removal disables without clearing; another DID and mismatched/anonymous identities cannot replace it. | `app/test/subscriptions/models/billing_owner_state_test.dart` |
| UT-003 | FR-001, FR-002, NFR-003, RULE-007 | AC-004, AC-006 | Resolve platform billing availability and public key selection. | iOS/Android keys, blank/missing keys, web/desktop platform, mismatched platform key. | Native matching key enables adapter; every unsupported/missing case disables billing without throwing. | `app/test/subscriptions/services/revenuecat_config_test.dart` |
| UT-004 | FR-013, RULE-004 | AC-015, AC-019 | Decide purchase eligibility per correlated tier. | Absent; accessible; pending payment; inaccessible with omitted/null, empty, `will_renew`, `will_not_renew`, `will_change_product`, `will_pause`, `requires_price_increase_consent`, `has_already_renewed`, or unknown renewal; stale/pending reconciliation; known/unknown subscription or license anomaly; dormant assignment. | Only absent and fully reconciled inaccessible/non-pending/exactly `will_not_renew`/anomaly-free tiers are purchasable; unknown anomaly renders through fallback but blocks; opposite tier remains independent; dormant assignment is preserved. | `app/test/subscriptions/models/subscription_purchase_eligibility_test.dart` |
| UT-005 | FR-017, RULE-006 | AC-018, AC-022 | Reduce baseline and observed reconciliation generations into pending/completed/failed states. | Baseline, first/later requested generations, reconciled generations, concurrent trigger, new license, same-subscription recovery, unchanged restore, timeout/error. | First requested generation above baseline becomes target; completion waits for reconciled >= target; purchase also needs recognized state while restore may complete unchanged; generation races never activate alone. | `app/test/subscriptions/services/reconciliation_state_test.dart` |
| UT-006 | FR-014, FR-015, FR-024 | AC-016, AC-017, AC-022 | Map direct paywall results and observable pre-presentation availability to coordinator actions. | Purchased, restored, cancelled, error/throw; missing named offering; missing required package; attached-paywall presentation failure. | Purchased/restored reconcile; cancelled is normal no-op; error/throw is non-success; missing offering/package makes no RevenueCatUI call; attachment failure maps after attempted presentation; none issues manual purchase. | `app/test/subscriptions/services/paywall_result_test.dart` |
| UT-007 | FR-019, RULE-003, RULE-004 | AC-019 | Build eligible assignment candidates from retained sessions and assignments. | Owner/non-owner sessions, existing paid assignment, missing/stale lease, both tiers. | Owner may be a candidate; retained eligible non-owner may be a candidate; assigned or stale targets are excluded. | `app/test/subscriptions/models/assignment_candidates_test.dart` |
| UT-008 | FR-008, FR-020, FR-026, NFR-004 | AC-011, AC-020, AC-025 | Fence asynchronous results and classify assignment errors. | Matching/stale lease, removed owner, duplicate tap, late completion, target 422 `assignment_target_ineligible`, owner 401. | Only current operation and lease can publish; duplicates/stale completions have no effect; target race is typed separately; owner 401 uses normal session invalidation. | `app/test/subscriptions/services/subscription_operation_fence_test.dart` |
| UT-009 | NFR-002 | AC-024 | Sanitize subscription diagnostics and configure native logging. | Canary UUID, DID, token, receipt, transaction, provider ID, raw `PlatformException` message/details, tier/platform/outcome, debug/release mode. | Output retains only approved classification, tier, platform, and outcome fields; release selects the approved RevenueCat log level; no raw SDK log handler or raw exception payload is forwarded. | `app/test/subscriptions/observability/subscription_privacy_test.dart` |
| UT-010 | FR-012, FR-024 | AC-014, AC-022 | Map owner billing data to distinct presentation states. | Empty, active, canceled-accessible, dormant, stale, pending, anomaly, provider error. | Each state has the expected status/actions and no false activation. | `app/test/subscriptions/models/subscription_presentation_test.dart` |
| UT-011 | FR-017, NFR-008 | AC-018, AC-025 | Verify bounded polling schedule and disposal. | Fake clock, completion before deadline, timeout, cancellation, transient read error. | Poll count/delay is bounded, completion stops work, disposal cancels, and retry starts a fresh bounded wait. | `app/test/subscriptions/services/reconciliation_polling_test.dart` |
| UT-012 | FR-009, FR-012, FR-027, NFR-001 | AC-012, AC-023, AC-029 | Decode every self-access, billing-state, subscription, license, and assignment field with the specified compatibility boundary. | Golden camelCase fixtures; missing/wrong-type values; unknown tier; unknown provider status/store/renewal/anomaly; shuffled arrays and `subscriptionId`. | Required shape/type errors and unknown tier fail; unknown provider-owned strings render a safe fallback; licenses correlate only by internal subscription ID independent of order. | `app/test/subscriptions/models/subscription_models_test.dart` |
| UT-013 | FR-003, FR-004, FR-026, NFR-004 | AC-007, AC-008, AC-011, AC-025 | Recheck the active reserved-owner lease and distinguish DID-only setup from UUID recovery. | Owner GET/PUT, offering retrieval, identity inspection/login, paywall, restore, Customer Center, reconciliation POST/poll GET, assignment, unassignment with current/stale states; reservation persistence success/failure; GET match/404/mismatch. | Only a current exact owner reaches calls; every poll rechecks; stale state stops before dispatch; PUT requires a durable matching DID-only reservation; UUID is persisted before identity; UUID recovery is GET-only and stays locked on 404/mismatch. | `app/test/subscriptions/services/subscription_active_owner_guard_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-003, FR-009, FR-012, FR-020, FR-027, NFR-001 | AC-007, AC-012, AC-014, AC-020, AC-023, AC-029 | Verify exact AppView subscription API methods, paths, bodies, owner relationship, and error mapping. | Dio with `http_mock_adapter` and complete shuffled AppView fixtures. | Call self-access, ensure/get account, reconciliation, assign, and unassign. | Methods and camelCase bodies match; every license decodes `subscriptionId`; array order is irrelevant; `billing_license_not_found`, `assignment_target_ineligible`, `assignment_conflict`, and `assignment_cooldown` map by exact error value; owner 401 and 5xx use standard handling. | `app/test/subscriptions/data/subscription_api_client_test.dart` |
| IT-002 | FR-007, FR-010, FR-011, NFR-001 | AC-010, AC-013, AC-023 | Read inactive retained-account access with its own token and device scope. | Registry with active Alice and inactive Bob; recording request adapter. | Load Bob's tier for the switcher. | Request uses Bob's current lease/token and shared device ID without changing active DID or route; owner endpoint is not called. | `app/test/subscriptions/data/account_scoped_access_test.dart` |
| IT-003 | FR-004, FR-005 | AC-008 | Enforce identified RevenueCat precondition at the adapter boundary. | Fake native channel/adapter reporting exact, anonymous, DID-shaped, and mismatched IDs. | Attempt paywall, restore, and Customer Center operations. | Only exact AppView owner UUID reaches native operation; every other identity fails before purchase/restore/manage. | `app/test/subscriptions/services/revenuecat_identity_guard_test.dart` |
| IT-004 | FR-003, FR-006, FR-008, RULE-002, RULE-008 | AC-007, AC-009, AC-011 | Persist, disable, interrupt, and recover one owner reservation without replacement. | Versioned secure registry fixtures with Alice DID-only and DID/UUID reservations plus Bob; controllable PUT; owner GET match, 404, and mismatch. | Interrupt after DID persistence, after PUT, and after UUID persistence; restart, switch/remove, reject Bob setup, then resume as Alice. | DID always persists before PUT; only Alice retries while UUID is absent; UUID persists before identity; once present, exact GET resumes while 404/mismatch stays locked with zero PUT or replacement identity calls. | `app/test/auth/providers/billing_owner_registry_test.dart` |
| IT-005 | FR-004, FR-014, FR-015 | AC-008, AC-016, AC-017, AC-022 | Coordinate named offering presentation under the exact owner identity. | Fake AppView owner repository and recording RevenueCat gateway with Plus/Business offerings, missing offering/package states, and presentation throws/results. | Select each tier and exercise every availability and paywall outcome. | Correct named offering is presented once when observable prerequisites exist; missing offering/package fails before presentation; attached-paywall failure maps from presentation; no current offering or manual purchase path is used. | `app/test/subscriptions/providers/paywall_coordinator_test.dart` |
| IT-006 | BR-001, FR-017, FR-018, RULE-001, RULE-005 | AC-001, AC-002, AC-018 | Coordinate purchase through baseline reconciliation and both post-lapse outcomes. | Fake gateway; scripted concurrent generations; old/new subscriptions, licenses, assignments, access; fake clock. | Complete first purchase, same-subscription recovery, and new-subscription repurchase. | No CustomerInfo grants access; target generation reconciles before evaluation; a new license requires explicit assignment/access, same-subscription recovery preserves assignment/access, and a new post-lapse license does not replace the dormant one. | `app/test/subscriptions/providers/subscription_purchase_controller_test.dart` |
| IT-007 | FR-004, FR-022, RULE-005 | AC-005, AC-008, AC-018 | Restore under existing owner and reconcile known/new licenses. | Existing Plus assignment, scripted restored Business purchase, exact owner UUID. | Invoke restore and complete reconciliation. | Restore runs once under owner UUID, known assignment remains, Business is unassigned, and no automatic access is published. | `app/test/subscriptions/providers/subscription_restore_controller_test.dart` |
| IT-008 | BR-001, FR-019, FR-020, FR-021 | AC-001, AC-019, AC-020 | Integrate assignment UI controller with exact AppView assignment errors and refreshed state. | Retained targets and scripted success, owner 401, plus exact `billing_license_not_found`, `assignment_target_ineligible`, `assignment_conflict`, and `assignment_cooldown` envelopes on applicable PUT/DELETE routes. | Assign, reassign, and unassign with duplicate taps. | Endpoint/body is used once; exact error value, not only 409/422 status, determines presentation; target race and owner invalidation remain distinct; refreshed server truth is preserved. | `app/test/subscriptions/providers/subscription_assignment_controller_test.dart` |
| IT-009 | FR-004, FR-023 | AC-005, AC-008, AC-021 | Coordinate Customer Center callbacks and external management return. | Recording RevenueCat gateway, lifecycle observer, fake reconciliation repository. | Restore, select management, return to foreground, and dismiss. | Customer Center uses exact owner UUID and relevant return paths request one coalesced reconciliation/refresh. | `app/test/subscriptions/providers/customer_center_coordinator_test.dart` |
| IT-010 | BR-002, FR-006, FR-008, FR-011, FR-026, NFR-004, NFR-008 | AC-003, AC-011, AC-013, AC-025 | Reject stale async publication across account switches, reauthentication, removal, and disposal. | Controlled completers for Alice/Bob reads and owner operation; account leases/generations. | Complete operations out of order after each boundary transition. | State remains keyed to the matching lease; no stale navigation, identity call, mutation, or continuing poll occurs. | `app/test/subscriptions/providers/subscription_account_boundary_test.dart` |
| IT-011 | FR-007, FR-010 | AC-009, AC-013 | Verify typed subscription route and settings/account-switcher entry points. | Router widget harness with owner and beneficiary states. | Navigate from settings, switch active account, and open switcher. | Canonical route renders; owner gets management surface; beneficiary gets self-access and switch-to-owner; tier rows remain isolated. | `app/test/router/subscription_routes_test.dart` |
| IT-012 | FR-001, NFR-003 | AC-006 | Verify environment keys and native target declarations. | Read config examples, `pubspec.yaml`, Xcode project, Podfile, and Android Gradle files. | Run architecture/configuration assertions. | iOS/Android keys are distinct named public values, unsupported targets have no key fallback, and native minimums satisfy the pinned SDK. | `app/test/subscriptions/revenuecat_platform_configuration_test.dart` |
| IT-013 | FR-012, FR-027 | AC-014, AC-029 | Verify the AppView owner response exposes the internal subscription/license relationship. | AppView handler/store fixtures with multiple subscriptions and licenses in shuffled order. | Ensure and read the billing account. | Every license has camelCase `subscriptionId` matching exactly one returned internal subscription ID; no provider subscription identifier is exposed through that field; existing behavior is unchanged. | `appview/internal/routes/subscription_routes_test.go` |
| IT-014 | FR-003, FR-004, FR-026, NFR-004 | AC-007, AC-008, AC-011, AC-025 | Integrate the active-owner guard across the complete explicit owner call boundary. | Controller harness with exact owner, beneficiary-active, removed/stale owner, never-reserved setup, DID-only interrupted setup, and DID/UUID recovery. | Attempt owner GET/PUT, offering/identity/paywall, restore, Customer Center, reconciliation POST/polls, assignment, and unassignment; interrupt setup at each persistence boundary. | Only current exact owner calls dispatch; polling stops when stale; DID persists before PUT and UUID before identity; only DID-only same-owner setup may PUT; UUID recovery uses GET only and 404/mismatch cannot create or identify a replacement. | `app/test/subscriptions/providers/subscription_active_owner_guard_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Existing social, publishing, profile, draft, saved-post, scheduled-post, and business surfaces remain reachable without a paid tier. | BR-003, RULE-009 | AC-004 | Extend route/feature architecture coverage to assert no existing route or action checks subscription access in this slice. |
| REG-002 | Multi-account activation remains explicit and one account is active at a time. | BR-002, FR-006, FR-011 | AC-003, AC-013 | Existing account activation/switch tests pass with tier loading and billing-owner state present. |
| REG-003 | Web startup skips unsupported native initialization and remains usable. | FR-002, RULE-007 | AC-004, AC-006 | Bootstrap and route tests prove web builds expose access display but no RevenueCatUI action or native call. |
| REG-004 | Closing a modal or cancelling an optional action is not treated as an application error. | FR-016 | AC-017 | Messenger/error tests assert paywall cancellation emits neither error nor success and leaves navigation usable. |
| REG-005 | AppView 401 handling still invalidates only the affected account session. | FR-008, FR-026 | AC-011, AC-025 | Existing session invalidation tests pass when owner/access providers are registered with the account boundary. |
| REG-006 | Logs, Sentry, secure storage string output, SDK diagnostics, and API/provider errors redact account secrets. | FR-005, NFR-002 | AC-008, AC-024 | Extend canary suites with billing UUID, receipt, transaction, provider ID, and raw `PlatformException` payloads; assert release-safe RevenueCat logging and no raw forwarding handler. |
| REG-007 | Flutter static analysis and deterministic test suite remain clean without native store/network dependencies. | NFR-006, NFR-007 | AC-027 | `just app-analyze` and `just app-test` pass with all RevenueCat behavior behind replaceable boundaries. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Multi-account identity | Alice owner DID/session/UUID, Bob beneficiary DID/session, Carol free DID/session, distinct leases and handles. | AT-001 through AT-004, AT-008 through AT-011, AT-015; IT-002 through IT-010 |
| TD-002 | Canonical access responses | Free; Plus; Business; dormant Plus (`effectiveTier=free`, `givesAccess=false`, `assignedTier=plus`); optional end timestamps. | AT-003, AT-004, AT-015; UT-001, UT-012 |
| TD-003 | Owner billing state | Empty owner; active Plus assigned to Bob; unassigned Business; shuffled subscription/license arrays linked by `subscriptionId`; stale generations; canceled-accessible; lapsed will_not_renew; unknown renewal; anomaly; pending payment. | AT-002, AT-005, AT-013, AT-014; UT-004, UT-005, UT-010, UT-012; IT-013 |
| TD-004 | RevenueCat offerings | Named `plus` and `business` offerings with `$rc_monthly`; missing offering; missing required package; invalid/missing attached paywall discoverable during presentation. | AT-002, AT-005, AT-006, AT-014; IT-005 |
| TD-005 | Provider outcomes | Purchased, restored, cancelled, returned/thrown error, pre-presentation missing offering/package, attached-paywall presentation failure, restore identity conflict, Customer Center callbacks. | AT-006, AT-009, AT-010; UT-006; IT-005, IT-007, IT-009 |
| TD-006 | Reconciliation scripts | Baseline, concurrent requested-generation advances, delayed target reconciliation, target reconciled without purchase state, same-subscription recovery, new license, unchanged restore, timeout, transient failure, stale response after disposal. | AT-002, AT-007, AT-011, AT-014; UT-005, UT-011; IT-006, IT-010 |
| TD-007 | Assignment outcomes | Success, unassignment, ineligible target, existing assignment conflict, cooldown, missing license, target session race. | AT-002, AT-008; UT-007, UT-008; IT-008 |
| TD-008 | Platform configuration | Distinct test public iOS/Android keys; blank/missing keys; web/macOS/Linux/Windows platform values; pinned minimum targets. | AT-012; UT-003; IT-012 |
| TD-009 | Privacy canaries | Fake billing UUID, DID, session token, receipt, transaction ID, provider subscription ID, payment-like text. | UT-009; REG-006 |
| TD-010 | Sandbox accounts | Separate Apple sandbox and Google license-test accounts with clean purchase history plus the matching CraftSky owner/beneficiary sessions. | MAN-001 through MAN-004 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-001, FR-001, FR-014, NFR-007 | AC-001, AC-006, AC-016, AC-028 | iOS sandbox purchase end to end. | On a physical device or supported simulator with an Apple sandbox account, verify each configured paywall version has a visible/operable close affordance, select the explicit CraftSky billing owner, open each named paywall, dismiss one, purchase the other, reconcile, assign it, and refresh that DID. | Both dashboard paywalls can actually be dismissed regardless of template version; dismissal does not mutate state; one charge completes; the license appears unassigned; explicit assignment succeeds; only AppView self-access shows the target tier. |
| MAN-002 | BR-001, FR-001, FR-014, NFR-007 | AC-001, AC-006, AC-016, AC-028 | Android license-test purchase end to end. | On a Play-enabled device/emulator installed from the configured test track, verify each configured paywall version has a visible/operable close affordance, then repeat the Plus/Business dismissal, purchase, reconciliation, assignment, and target-access flow. | Both paywalls can actually be dismissed; Google Play purchase succeeds once; RevenueCat uses the owner UUID; AppView surfaces and assigns the license; no other retained account inherits access. |
| MAN-003 | BR-004, FR-022, NFR-007 | AC-005, AC-018, AC-028 | Restore on both native stores. | Under the original CraftSky owner UUID, reinstall or use a second device, identify the same owner, invoke restore, reconcile, and inspect known and newly discovered licenses. Run the wrong-owner portion only after RevenueCat restore ownership and the approved recovery procedure are configured and documented. | Original owner restores provider purchases; known assignments persist; new licenses remain unassigned. Wrong-owner execution is production-blocked until its prerequisite exists; when run, it must fail safely with guidance and no transfer. |
| MAN-004 | BR-004, FR-023, NFR-007 | AC-005, AC-021, AC-028 | Customer Center and provider management on iOS and Android. | Open Customer Center with an active sandbox subscription, run restore, open cancellation/management, return from any external store destination, and dismiss. | The owner's subscriptions appear; platform-appropriate management opens; callback/return triggers AppView reconciliation; owner status refreshes without exposing details to beneficiaries. |
| MAN-005 | NFR-004, NFR-005 | AC-025, AC-026 | Native lifecycle and accessibility. | During paywall, restore, reconciliation, and Customer Center flows, background/foreground the app, rotate where supported, use platform back, enable screen reader and large text, and test a narrow screen. | No duplicate purchase/mutation or stale navigation occurs; polling stops when appropriate; critical status/actions remain understandable, visible, and operable. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Automated tests cannot validate native RevenueCatUI rendering, actual close affordances, or real store transaction sheets. | BR-001, FR-001, FR-014, NFR-007 | Flutter widget tests cannot faithfully host provider paywall templates and native store UI. | Require MAN-001 and MAN-002 for every configured paywall version before activation and retain screenshots/log-free outcome evidence. |
| GAP-002 | Store restore ownership and keep-with-original conflict behavior are not reproducible with deterministic fakes alone. | BR-004, FR-022, RULE-008 | Receipt/account ownership belongs to Apple, Google, and RevenueCat. | Require MAN-003 and document the production recovery procedure before release. |
| GAP-003 | Customer Center actions differ by store and dashboard configuration. | FR-023, NFR-007 | Refund, cancellation, and external management behavior is native/provider-owned. | Configure Customer Center and complete MAN-004 on both platforms. |
| GAP-004 | End-to-end reconciliation timing depends on live RevenueCat webhook/API and AppView worker configuration. | FR-017, RULE-006 | Automated Flutter tests use scripted AppView generations and cannot prove live provider latency. | During sandbox checks, verify webhook/reconciliation completion and preserve timeout/retry behavior for outages. |
| GAP-005 | Billing-owner recovery remains undefined. | FR-008, FR-022, RULE-008 | Recovery was explicitly excluded from this implementation slice but blocks production support readiness. | Define and review recovery before production activation; tests cover only safe refusal. |
| GAP-006 | Exact visual merchandising copy is pending. | FR-025, NFR-005 | Requirements allow UI polish without changing semantics. | Verify semantics and accessibility now; finalize copy during design/polish and update widget expectations deliberately. |
| GAP-007 | The wrong-owner branch of MAN-003 cannot yet produce valid evidence. | FR-008, FR-022, RULE-008 | RevenueCat restore ownership and the approved recovery/support procedure are not configured. | Mark that branch blocked, not failed or passed, until configuration and documentation are complete. |

No gap blocks deterministic test-first implementation after workflow document
review. `GAP-001` through `GAP-005` and `GAP-007` block production activation evidence or
support readiness as stated above.

## 10. Out Of Scope

- Tests that expect any existing feature to become paid-only.
- Capability-specific Plus or Business authorization tests.
- AppView database schema, webhook, worker, reconciliation behavior,
  assignment-policy, or account-deletion tests already covered by the AppView
  subscription workflow. The owner-response relationship test is in scope.
- RevenueCat dashboard resource creation, store catalog creation, pricing,
  territory, trial, introductory-offer, or Family Sharing tests.
- Annual, web, desktop, bundle, upgrade/downgrade, ownership transfer/reset,
  remote invitation, tax, invoice, or refund-policy tests.
- PDS, Lexicon, Tap, feed ranking, search reach, moderation, or public record
  tests unrelated to the no-gating regression boundary.

## 11. Handoff To Document Review

- Requirements file:
  `docs/changes/2026-09-10-flutter-subscriptions/01-requirements.md`
- Test specification:
  `docs/changes/2026-09-10-flutter-subscriptions/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill:
  `docs/changes/2026-09-10-flutter-subscriptions/`
- Recommended first failing test for implementation: `UT-001` canonical
  self-access tier parsing/projection, followed by `IT-001` AppView API contracts
  and `UT-002` billing-owner identity state.
- Suggested test order for implementation:
  1. `UT-001`, `UT-012`, `IT-001`, `IT-013`: establish AppView models, API, and
     explicit subscription/license relationship.
  2. `UT-002`, `UT-013`, `IT-004`, `IT-014`, `AT-001`, `AT-011`, `AT-017`:
     establish crash-safe DID-first reservation and same-owner recovery.
  3. `UT-003`, `IT-003`, `IT-012`: establish safe
     native SDK configuration and operation-level identity preconditions.
  4. `UT-004`, `UT-006`, `IT-005`, `AT-005`, `AT-006`: establish checkout,
     post-lapse repurchase eligibility, and provider outcomes.
  5. `UT-005`, `UT-011`, `IT-006`, `AT-002`, `AT-007`: establish reconciliation
     and purchase-to-license coordination.
  6. `UT-007`, `UT-008`, `IT-008`, `AT-008`: establish explicit assignment.
  7. `IT-007`, `IT-009`, `AT-009`, `AT-010`: establish restore and management.
  8. `IT-002`, `IT-010`, `IT-011`, `AT-003`, `AT-004`, `AT-011`, `AT-015`:
     establish account isolation and routes.
  9. `UT-009`, `UT-010`, `AT-012` through `AT-014`, `AT-016`, and all regression
     tests: complete privacy, states, accessibility, and no-gating protection.
  10. `MAN-001` through `MAN-005`: collect native sandbox activation evidence,
      leaving the wrong-owner MAN-003 branch blocked until its prerequisites exist.
- Commands discovered: `just app-analyze`, `just app-test`,
  `just app-test test/subscriptions`, `just app-build-ios`,
  `just app-build-apk`, `just app-run-ios`, and `just app-run-android`.
- Blocking gaps: Workflow document review is required before implementation.
  No test-design gap blocks fake-based implementation. Production activation is
  blocked by `GAP-001` through `GAP-005` and `GAP-007` as applicable.
