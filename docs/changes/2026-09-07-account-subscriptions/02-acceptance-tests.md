# Acceptance Test Specification: AppView Account Subscriptions

## 1. Test Strategy

This high-risk billing boundary uses focused unit, PostgreSQL integration, route,
and regression tests. Tests treat RevenueCat customer subscription snapshots as
authoritative provider input and do not reproduce store lifecycle event
sequences. Webhook tests prove authentication, sanitized durable deduplication,
and reconciliation queueing; they do not retain or test raw provider payloads.

Provider calls use a deterministic fake API v2 client. Time-dependent cooldown,
staleness, lease, and deletion-state behavior uses an injected clock. Concurrent
creation, assignment, snapshot application, and work claiming use separate
database connections.

Risk level: **High**. Document review and explicit approval remain required
before implementation.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-002, IT-001 | Acceptance / Integration | Yes |
| BR-002 | AC-003 | AT-003, REG-004 | Acceptance / Regression | Yes |
| BR-003 | AC-003, AC-006 | AT-003, AT-006, UT-002 | Acceptance / Unit | Yes |
| BR-004 | AC-004 | AT-010, REG-001, REG-003 | Acceptance / Regression | Yes |
| BR-005 | AC-005 | IT-001 | Integration | Yes |
| FR-001 | AC-007 | AT-001, IT-002 | Acceptance / Integration | Yes |
| FR-002 | AC-008 | AT-001, AT-004, REG-005 | Acceptance / Regression | Yes |
| FR-003 | AC-001, AC-005, AC-009 | AT-002, UT-002, IT-001, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-001, AC-009 | AT-002, IT-001, IT-005 | Acceptance / Integration | Yes |
| FR-005 | AC-006, AC-010 | AT-006, AT-008, UT-001, UT-002 | Acceptance / Unit | Yes |
| FR-006 | AC-003, AC-010 | AT-003, AT-008, UT-001 | Acceptance / Unit | Yes |
| FR-007 | AC-003, AC-011 | AT-003, UT-001, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-007, AC-008, AC-012 | AT-001, AT-004 | Acceptance | Yes |
| FR-009 | AC-002, AC-013 | AT-002, AT-005, IT-003 | Acceptance / Integration | Yes |
| FR-010 | AC-013 | AT-005, IT-003 | Acceptance / Integration | Yes |
| FR-011 | AC-013 | AT-005, IT-003 | Acceptance / Integration | Yes |
| FR-012 | AC-002, AC-014 | AT-002, AT-005, IT-003 | Acceptance / Integration | Yes |
| FR-013 | AC-003, AC-004 | AT-003, AT-010, UT-001 | Acceptance / Unit | Yes |
| FR-014 | AC-015 | AT-007, UT-004 | Acceptance / Unit | Yes |
| FR-015 | AC-016, AC-017 | AT-007, IT-006 | Acceptance / Integration | Yes |
| FR-016 | AC-006, AC-017 | AT-006, AT-007, IT-006 | Acceptance / Integration | Yes |
| FR-017 | AC-006, AC-009, AC-018 | AT-006, IT-005, IT-007 | Acceptance / Integration | Yes |
| FR-018 | AC-018 | AT-006, IT-007 | Acceptance / Integration | Yes |
| FR-019 | AC-019 | AT-006, UT-003, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-020 | AC-004, AC-010 | AT-008, AT-010, IT-004 | Acceptance / Integration | Yes |
| FR-022 | AC-020 | AT-009, IT-008, REG-002 | Acceptance / Integration / Regression | Yes |
| FR-026 | AC-021 | AT-011, UT-005, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-027 | AC-010 | AT-008, UT-001, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-028 | AC-011 | AT-003, UT-001 | Acceptance / Unit | Yes |
| FR-029 | AC-010, AC-011 | AT-003, AT-008, IT-005 | Acceptance / Integration | Yes |
| FR-030 | AC-022 | AT-005, UT-006, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-031 | AC-020 | AT-009, IT-008 | Acceptance / Integration | Yes |
| FR-032 | AC-018 | AT-006, IT-007 | Acceptance / Integration | Yes |
| NFR-001 | AC-004 | AT-010, REG-003 | Acceptance / Regression | Yes |
| NFR-002 | AC-012, AC-013 | AT-004, AT-005, REG-005 | Acceptance / Regression | Yes |
| NFR-003 | AC-008, AC-015, AC-023 | AT-001, AT-007, AT-012, REG-006 | Acceptance / Regression | Yes |
| NFR-004 | AC-016, AC-018 | AT-006, AT-007, IT-006, IT-007 | Acceptance / Integration | Yes |
| NFR-005 | AC-016 | AT-007, IT-006 | Acceptance / Integration | Yes |
| NFR-006 | AC-003 | AT-003, IT-004 | Acceptance / Integration | Yes |
| NFR-007 | AC-024 | AT-013, IT-009 | Acceptance / Integration | Yes |
| NFR-008 | AC-024 | AT-013, REG-001 through REG-006 | Acceptance / Regression | Yes |
| RULE-001 | AC-007, AC-008 | AT-001, IT-001, IT-002 | Acceptance / Integration | Yes |
| RULE-002 | AC-001, AC-003 | AT-002, AT-003, IT-001 | Acceptance / Integration | Yes |
| RULE-003 | AC-002, AC-014 | AT-002, AT-005, IT-003 | Acceptance / Integration | Yes |
| RULE-004 | AC-002 | AT-002 | Acceptance | Yes |
| RULE-005 | AC-003 | AT-003, REG-004 | Acceptance / Regression | Yes |
| RULE-006 | AC-019 | AT-006, UT-003 | Acceptance / Unit | Yes |
| RULE-007 | AC-025 | UT-007 | Unit | Yes |
| RULE-008 | AC-026 | AT-004, IT-010, MAN-001 | Acceptance / Integration / Manual | Partial; Flutter evidence deferred |

## 3. Acceptance Scenarios

### AT-001: Billing Identity Is Explicit, Stable, And Private
Requirement IDs: FR-001, FR-002, FR-008, RULE-001, NFR-003
Acceptance Criteria: AC-007, AC-008
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/subscription_routes_test.go`

```gherkin
Feature: Private billing ownership
  Scenario: A member deliberately establishes one billing identity
    Given a current member has no billing account
    When login, self-access, and ordinary settings requests are made
    Then no billing account is created
    When the member explicitly ensures billing ownership repeatedly
    Then one account and one stable opaque RevenueCat UUID are returned
    And unrelated and assigned members cannot read or manage it
```

### AT-002: Plus And Business Licenses Are Independently Assigned
Requirement IDs: BR-001, FR-003, FR-004, FR-009, FR-012, RULE-002, RULE-003, RULE-004
Acceptance Criteria: AC-001, AC-002
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/subscriptions/subscription_acceptance_test.go`

```gherkin
Feature: Independent licenses
  Scenario: One payer funds separate Plus and Business accounts
    Given a complete RevenueCat snapshot contains accessible Plus and Business subscriptions
    When the snapshot is applied repeatedly
    Then one subscription and license exist for each tier
    And both licenses are initially unassigned
    When the owner assigns Plus to DID A and Business to DID B
    Then each DID receives only its assigned tier
    And assigning both licenses to one DID fails atomically
```

### AT-003: Self Access Is Local, DID-Bound, And Minimal
Requirement IDs: BR-002, BR-003, FR-006, FR-007, FR-013, FR-028, FR-029, NFR-006, RULE-002, RULE-005
Acceptance Criteria: AC-003, AC-011
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/subscription_routes_test.go`

```gherkin
Feature: Account-scoped paid access
  Scenario: An account reads only its assigned access
    Given an accessible or dormant assignment exists for a member DID
    And RevenueCat is unavailable
    When the DID reads self access from another valid session and device
    Then the result comes from local state without a provider call
    And accessible assignment returns its tier
    And dormant assignment returns effective free and its assigned tier
    And no payer, product, provider, anomaly, or management fact is present
    When the DID changes handle or another local account becomes active
    Then the result is unchanged
```

### AT-004: Owner Billing State Uses The Private API Contract
Requirement IDs: FR-002, FR-008, NFR-002, RULE-008
Acceptance Criteria: AC-008, AC-012, AC-026
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/subscription_routes_test.go`

```gherkin
Feature: Owner billing state
  Scenario: The owner reads billing and identity readiness
    Given the owner has subscriptions, licenses, assignments, and an anomaly
    When the owner calls the billing-state endpoint
    Then the camelCase response contains only permitted billing context
    And billing readiness names only the stable AppView-issued UUID
    When a non-owner makes the request
    Then a standard non-leaking error is returned
```

### AT-005: Assignment Enforces Same-Device Proof And Simple Cooldown
Requirement IDs: FR-009, FR-010, FR-011, FR-012, FR-030, RULE-003
Acceptance Criteria: AC-013, AC-014, AC-022
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/subscription_assignment_test.go`

```gherkin
Feature: Safe assignment
  Scenario: An owner changes an assignment
    Given owner and target have current sessions on the request device
    When the owner assigns an assignable license
    Then assignment succeeds even when the target is not the active Flutter account
    When owner, license, membership, session, device, or uniqueness checks fail
    Then the prior assignment remains unchanged
    When a target change succeeds
    Then another target change before seven days is rejected
    And a target change at or after seven days succeeds
```

### AT-006: Reconciliation Uses RevenueCat Snapshot Truth
Requirement IDs: BR-003, FR-005, FR-016 through FR-019, FR-032, NFR-004, RULE-006
Acceptance Criteria: AC-006, AC-009, AC-018, AC-019
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/subscriptions/reconciliation_test.go`

```gherkin
Feature: Reconciliation-first provider state
  Scenario: Different triggers converge through one snapshot path
    Given webhooks of known, unknown, and reordered types arrive for one customer
    When webhook, owner-refresh, restore-like, and scheduled work runs
    Then each trigger fetches complete current RevenueCat subscription state
    And event type never directly determines access
    And repeated snapshots update the same rows
    And known assignments remain unchanged
    And new licenses remain unassigned
    And sandbox or unknown products, including mappings to unconfigured apps, grant no production access

  Scenario: A trigger arrives while a claimed snapshot is being fetched
    Given a worker has claimed requested generation 4 and is fetching RevenueCat
    When another reconciliation trigger advances requested generation to 5
    And the worker successfully applies its complete generation 4 snapshot
    Then generation 5 remains due for a later claim

  Scenario: A stale lease tries to finish after recovery
    Given a replacement worker holds the current lease and generation
    When the superseded worker tries to apply its snapshot
    Then the stale apply is rejected without mutating subscription state
```

### AT-007: Webhook Ingress Authenticates And Queues Sanitized Work
Requirement IDs: FR-014 through FR-016, NFR-003 through NFR-005
Acceptance Criteria: AC-015, AC-016, AC-017
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/integrations/revenuecat/webhook_handler_test.go`

```gherkin
Feature: RevenueCat webhook ingress
  Scenario: Verified events become durable reconciliation signals
    Given the webhook is configured with Authorization and HMAC secrets
    And an ingress deadline no greater than ten seconds
    When a bounded request has valid credentials and customer/event identity
    Then sanitized event work is committed once before a successful response
    And no provider fetch or subscription mutation happens in ingress
    When the same event ID is retried or its type is unknown
    Then it remains an idempotent reconciliation signal
    When credentials, signature age, body size, or structure are invalid
    Then no raw body is persisted and no access is granted
```

### AT-008: Reconciled Access Does Not Infer Local Expiry
Requirement IDs: FR-005, FR-006, FR-020, FR-027, FR-029
Acceptance Criteria: AC-010
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/subscriptions/access_acceptance_test.go`

```gherkin
Feature: Subscription access boundaries
  Scenario: Provider failure preserves the latest accepted access
    Given an assigned license has givesAccess true and a billing-period end
    And RevenueCat becomes unavailable
    When the billing-period end passes and reconciliation becomes stale
    Then effective access remains paid from the latest accepted givesAccess
    And staleness is observable
    And the assignment remains unchanged
    When a later complete snapshot for the same subscription reports givesAccess false
    Then effective access becomes free without deleting the assignment
    When another snapshot for that same subscription reports givesAccess true
    Then the same assignment supplies access again
```

### AT-009: Account Deletion Closes Local Billing Authority
Requirement IDs: FR-022, FR-031
Acceptance Criteria: AC-020
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/accountdeletion/subscription_acceptance_test.go`

```gherkin
Feature: Subscription-aware account deletion
  Scenario: Any uncertain, renewable, resumable, or pending subscription blocks final deletion
    Given a complete post-intent snapshot contains a subscription with pending payment or an auto-renewal status other than exact will_not_renew
    When deletion starts
    Then AppView warns that provider billing is not canceled
    And requests a new reconciliation generation
    When deletion is confirmed
    Then deletion is blocked without removing billing state

  Scenario: Explicitly canceled active provider billing permits deletion
    Given a complete post-intent reconciliation shows every subscription has pending_payment false and exact auto_renewal_status will_not_renew
    And at least one subscription is active and currently gives access
    When warned deletion is confirmed
    Then live ownership, subscriptions, licenses, and assignments are removed
    And only a DID-free billing ID, RevenueCat UUID, and closure time remain
    And no provider cancellation request occurs
    And later events are acknowledged without reconciliation or access recreation

  Scenario: Explicitly canceled expired provider billing permits deletion
    Given a complete post-intent reconciliation shows every expired inaccessible subscription has pending_payment false and exact auto_renewal_status will_not_renew
    When warned deletion is confirmed
    Then local billing closure succeeds under the same rule as an active canceled subscription

  Scenario: An assigned non-owner is deleted
    Given a member DID is assigned another owner's license
    When that member completes account deletion
    Then every assignment targeting the deleted DID is removed atomically
    And billing ownership and provider subscription state are unchanged
```

### AT-010: Subscription State Never Alters Free Social Data
Requirement IDs: BR-004, FR-013, FR-020, NFR-001
Acceptance Criteria: AC-004
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/subscriptions/isolation_acceptance_test.go`

```gherkin
Feature: Public and free-state isolation
  Scenario: Billing changes authorization only
    Given free and paid members have existing social data
    When snapshots, assignments, access changes, anomalies, and deletion are processed
    Then accounts without accessible assignment remain free
    And no PDS, Tap, feed, search, ranking, moderation, reach, classification, or retained-data mutation occurs
```

### AT-011: Provider Anomalies Fail Closed
Requirement IDs: FR-026
Acceptance Criteria: AC-021
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/subscriptions/anomaly_acceptance_test.go`

```gherkin
Feature: Provider anomaly containment
  Scenario: Unexpected simultaneously accessible state appears
    Given a snapshot contains simultaneously accessible same-tier subscriptions or an accessible paid cross-tier transition
    When the snapshot is applied
    Then every provider subscription remains owner-visible
    And safe existing assignment is not moved
    And no extra license becomes assignable
    And no automatic promotion, inheritance, or cross-tier arbitration occurs

  Scenario: RevenueCat creates a new subscription after lapse or product change
    Given the prior subscription is inaccessible and its assignment remains dormant
    When a complete snapshot introduces a new accessible subscription ID
    Then a new license is created unassigned
    And the prior assignment is not inherited or moved
    And the new subscription is not classified as anomalous solely because it is new
```

### AT-012: Provider Secrets And Payloads Are Never Retained
Requirement IDs: NFR-003
Acceptance Criteria: AC-023
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/subscriptions/privacy_acceptance_test.go`

```gherkin
Feature: Billing data minimization
  Scenario: Sensitive canaries pass through every boundary
    Given webhook bodies, provider responses, and credentials contain unique canaries
    When success, duplicate, malformed, retry, API, logging, metrics, and operator paths run
    Then raw bodies, credentials, receipts, payment details, and private identifiers are absent from prohibited outputs and persistence
```

### AT-013: Release Verification Covers The Revised Boundary
Requirement IDs: NFR-007, NFR-008
Acceptance Criteria: AC-024
Priority: Must
Level: Acceptance
Automation Target: `just appview-check`

```gherkin
Feature: Release verification
  Scenario: The AppView release gate runs subscription coverage
    When just appview-check runs
    Then migration up, down, and up-again pass
    And subscription, reconciliation, webhook, assignment, access, anomaly, privacy, deletion, auth, and social regression suites pass
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-005 through FR-007, FR-013, FR-027 through FR-029 | AC-003, AC-010, AC-011 | Derive effective access from assignment and latest accepted `givesAccess`. | Plus, Business, free, dormant, stale reconciliation, and passed period timestamps. | Correct minimal projection; timestamps do not override `givesAccess`; no provider lifecycle interpretation. | `appview/internal/subscriptions/access_test.go` |
| UT-002 | BR-003, FR-003, FR-005, FR-017 | AC-006, AC-009 | Map complete API v2 subscription resources into sanitized snapshots. | Status variants, `gives_access`, product/store/environment, timestamps, pagination, additive fields. | `gives_access` is copied; subscription identity is stable; app/tier derive from configured product mapping; unknown additive fields are tolerated. | `appview/internal/integrations/revenuecat/client_test.go` |
| UT-003 | FR-019, RULE-006 | AC-019 | Evaluate production allowlisting. | Known/unknown project, environment, and product mappings carrying app and tier. | Only configured production mappings grant a tier. | `appview/internal/subscriptions/catalog_test.go` |
| UT-004 | FR-014 | AC-015 | Verify exact Authorization and timestamped raw-body HMAC. | Missing, duplicate, malformed, stale, valid, and body-mutated signatures. | Only valid bounded requests pass, using constant-time comparison. | `appview/internal/integrations/revenuecat/signature_test.go` |
| UT-005 | FR-026, FR-029 | AC-021 | Classify snapshot anomalies without inventing subscription lineage. | Single subscriptions, simultaneously accessible same-tier duplicates, accessible cross-tier changes, post-lapse new IDs, and existing assignments. | Simultaneous conflicts grant no new assignable access or movement; a normal new ID creates an unassigned non-anomalous license. | `appview/internal/subscriptions/anomaly_test.go` |
| UT-006 | FR-030 | AC-022 | Calculate target-change cooldown. | Initial assignment and changes before/at/after seven days. | Initial assignment is free; only elapsed target changes succeed. | `appview/internal/subscriptions/cooldown_test.go` |
| UT-007 | RULE-007 | AC-025 | Validate tier values. | Free, Plus, Business, unknown. | Only `free`, `plus`, and `business` are accepted in their defined contexts. | `appview/internal/subscriptions/types_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, BR-005, FR-003, FR-004, RULE-001 through RULE-003 | AC-001, AC-002, AC-005 | Enforce core billing, subscription, license, and assignment constraints. | Isolated PostgreSQL schema. | Insert valid and conflicting rows. | Stable uniqueness and one-to-one/cardinality constraints hold with separate subscription/license rows. | `appview/internal/subscriptions/store_integration_test.go` |
| IT-002 | FR-001, RULE-001 | AC-007 | Ensure one billing account under concurrency. | Current member and two connections. | Ensure repeatedly and concurrently. | One account and stable UUID result; ordinary reads create none. | `appview/internal/subscriptions/store_integration_test.go` |
| IT-003 | FR-009 through FR-012, FR-030, RULE-003 | AC-013, AC-014, AC-022 | Commit assignment with same-device revalidation and cooldown. | Owner/license/target session matrix and two connections. | Assign and race revocation/conflicting assignment. | Valid changes serialize; invalid changes roll back completely. | `appview/internal/subscriptions/assignment_integration_test.go` |
| IT-004 | FR-007, FR-020, FR-027, NFR-006 | AC-003, AC-010 | Read indexed effective access without RevenueCat. | Large unrelated fixture, passed billing-period timestamps, stale reconciliation, and failing provider fake. | Read self access before and after timestamps pass. | Latest accepted `givesAccess` controls the local projection and no provider call occurs. | `appview/internal/subscriptions/access_integration_test.go` |
| IT-005 | FR-003, FR-004, FR-017, FR-019, FR-026, FR-029 | AC-009, AC-019, AC-021 | Atomically apply complete snapshots. | Newer/older, sandbox, unknown, simultaneous duplicate/cross-tier, same-ID recovery, and post-lapse new-ID snapshots. | Apply in differing and concurrent orders. | Newest accepted state wins; same-ID assignments persist; new IDs are unassigned; simultaneous anomalies and allowlists fail closed. | `appview/internal/subscriptions/snapshot_integration_test.go` |
| IT-006 | FR-015, FR-016, NFR-004, NFR-005 | AC-016, AC-017 | Deduplicate sanitized webhook work before acknowledgement. | Real handler/store, repeated IDs, commit failures, and configured ingress deadlines. | Deliver concurrently and interrupt persistence. | One work item commits before success within the configured deadline; failures retry; provider fetch never runs in ingress; no raw body is stored. | `appview/internal/subscriptions/webhook_store_integration_test.go` |
| IT-007 | FR-017, FR-018, FR-032, NFR-004 | AC-018 | Run all reconciliation triggers through one retryable processor. | Scripted provider responses, known assignments, new licenses, trigger-during-fetch barrier, and stale work lease. | Trigger webhook, owner, scheduled, and restore-like work while controlling claim/fetch/apply order. | Same snapshot path converges; a newer requested generation remains due after an older apply; stale completion is rejected; assignments remain neutral. | `appview/internal/subscriptions/reconciliation_integration_test.go` |
| IT-008 | FR-022, FR-031 | AC-020 | Integrate account deletion closure. | Existing deletion flow with active/accessible canceled rows, expired/inaccessible canceled rows, mixed uncertain/renewable/resumable/pending rows, owned licenses, and targeted licenses. | Attempt owner deletion before reconciliation; reconcile exact `will_not_renew`/non-pending evidence and close while access remains active; test mixed blocker sets; delete an assigned non-owner; then deliver a later event. | Both active/access-granting and expired/inaccessible canceled rows permit atomic closure; any mixed row lacking exact non-renewal/non-pending evidence blocks intact; target deletion unassigns; later events cannot restore closed state; no provider cancellation occurs. | `appview/internal/accountdeletion/subscription_integration_test.go` |
| IT-009 | NFR-007 | AC-024 | Apply migration up/down/up. | Existing auth, session, and deletion sentinels. | Run migration cycle and constraints. | Existing data and constraints survive; billing objects are reversible. | `appview/internal/db/subscriptions_migration_test.go` |
| IT-010 | RULE-008 | AC-026 | Enforce owner billing identity readiness. | Owner/non-owner sessions and exact/mismatched UUIDs. | Read readiness and request refresh. | Only owner with its stable UUID can invoke billing operations; active DID is never substituted. | `appview/internal/routes/subscription_identity_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Existing members remain free without billing rows. | BR-004, NFR-008 | AC-004, AC-024 | Run representative routes with no billing data and verify unchanged behavior. |
| REG-002 | Account deletion authentication and lifecycle remain intact. | FR-022, NFR-008 | AC-020, AC-024 | Extend deletion suites with subscribed and assigned DIDs. |
| REG-003 | Billing cannot invoke PDS/Tap/social side effects. | BR-004, NFR-001, NFR-008 | AC-004, AC-024 | Use fail-on-call dependencies and social row sentinels. |
| REG-004 | Handles and active local account do not determine access. | BR-002, RULE-005 | AC-003 | Change handle/session/device activation while preserving DID assignment. |
| REG-005 | Routes preserve authentication, device, body, rate, and error policies. | FR-002, NFR-002, NFR-008 | AC-008, AC-012, AC-013, AC-024 | Extend route architecture and near-miss tests. |
| REG-006 | Observability redaction covers billing data. | NFR-003, NFR-008 | AC-023, AC-024 | Add billing canaries to logs, errors, metrics, and configuration formatting tests. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Identities and sessions | Owner, assigned, unrelated, deleted DIDs; device A/B; current/revoked/expired sessions. | AT-001 through AT-005, AT-009, IT-002, IT-003, IT-008, IT-010 |
| TD-002 | Subscription snapshots | API v2 Plus/Business resources with stable and replacement IDs, access/status/time variants, sandbox, unknown products, complete pagination, and additive fields. | AT-002, AT-006, AT-008, AT-011, UT-002, UT-003, UT-005, IT-005, IT-007 |
| TD-003 | Webhooks | Exact raw bytes for valid/invalid HMAC, known/unknown types, duplicate IDs, malformed and oversized bodies. | AT-007, UT-004, IT-006 |
| TD-004 | Anomalies and replacement IDs | Simultaneously accessible same-tier duplicates, accessible cross-tier transitions, post-lapse new IDs, existing/no assignment. | AT-011, UT-005, IT-005 |
| TD-005 | Sensitive canaries | API keys, webhook secrets, UUIDs, subscription IDs, receipts, payment fields, and raw JSON. | AT-001, AT-007, AT-012, REG-006 |
| TD-006 | Clock boundaries | Requested/claimed generation, reconciliation staleness, billing-period timestamps, seven-day cooldown, work lease, and closure. | AT-005, AT-006, AT-008, UT-006, IT-003 through IT-008 |
| TD-007 | Social sentinels | Representative profile, post, feed, search, moderation, and retained private rows. | AT-010, REG-001, REG-003 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | RULE-008 | AC-026 | Verify later Flutter SDK billing identity behavior. | Configure only with an AppView billing UUID; exercise beneficiary switching, explicit billing-owner selection, purchase, restore, and non-owner UI gating. | Ordinary account switching does not substitute active DID or anonymous identity; only selected owner billing operations run. |
| MAN-002 | None; external activation checklist | None | Verify RevenueCat/store merchandising before production. | Inspect prices, products, base plans, offerings, packages, trials, Family Sharing, territories, restore behavior, and webhook configuration. | External configuration matches requirements section 21 without AppView runtime catalog validation. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Acceptance Criteria | Reason | Follow-Up |
|---|---|---|---|---|---|
| GAP-001 | AppView tests cannot prove Flutter SDK identity transitions or store purchase/restore behavior. | RULE-008 | AC-026 | Flutter RevenueCat integration is out of scope. | Complete MAN-001 and Flutter integration tests before production launch. |
| GAP-002 | Real API v2 and webhook payloads may evolve. | FR-003, FR-014 through FR-018 | AC-006, AC-009, AC-015 through AC-018 | Deterministic fixtures are not live provider evidence. | Capture sanitized sandbox fixtures during integration and tolerate additive fields. |
| GAP-003 | No concrete paid feature is selected. | BR-003, FR-020 | AC-003, AC-004 | This stage builds the shared authority only. | Add capability-specific authorization tests when benefits are approved. |
| GAP-004 | `Keep with original App User ID` requires recovery support. | RULE-008 | AC-026 | Ownership transfer is deferred. | Define recovery before production activation. |

## 10. Out Of Scope

- Event-by-event lifecycle reducer tests.
- Purchase-intent and ownership-transfer tests.
- Automatic canonical failover or cross-tier arbitration tests.
- Raw webhook payload, quarantine, collision, or one-year retention tests.
- AppView tests for prices, territories, offerings, packages, trials, Family
  Sharing, or current offering selection.
- Flutter paywall, purchase, restore, Customer Center, and account-switch UI.
- Future web providers, bundles, and repeated same-tier license quantities.

## 11. Handoff To Document Review

- Requirements: `docs/changes/2026-09-07-account-subscriptions/01-requirements.md`
- Test specification: `docs/changes/2026-09-07-account-subscriptions/02-acceptance-tests.md`
- Next artifact: `03-document-review.md`
- First failing test: `UT-001` in
  `appview/internal/subscriptions/access_test.go`.
- Suggested order: `UT-001`, `IT-001`, `IT-002`, `AT-001`, `UT-002`, `UT-003`,
  `IT-005`, `AT-006`, `UT-004`, `IT-006`, `AT-007`, `IT-007`, access/assignment
  routes, anomaly/deletion/privacy regressions, migration verification.
- Commands: focused `go test` packages, then `just test`, then
  `just appview-check`.
- Blocking AppView gaps: None. GAP-001 and GAP-004 block production launch, not
  AppView implementation.
