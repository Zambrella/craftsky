# Acceptance Test Specification: Moderation Cases, Strikes, Appeals, And Suspension

## 1. Test Strategy

Risk level: **High**. Document review and explicit approval are required before implementation.

Testing follows the domain boundary described in `01-requirements.md`:

- Unit tests cover decision validation, strike expiry arithmetic, suspension capability classification, public references, appeal transitions, notification eligibility, payload construction, and Flutter model parsing.
- PostgreSQL integration tests cover concurrent case grouping, append-only adjudication, replay and revision protection, atomic projections, strike and suspension state, expiry recovery, appeals, privacy, notification outbox work, and migration constraints.
- HTTP route tests cover admin authentication, owner history, queue pagination, error envelopes, field-level disclosure, and the complete suspended-user capability matrix.
- Flutter widget and repository tests cover settings navigation, standing/history states, appeal fallbacks, notification routing, accessibility, localization use, and responsive layouts.
- Regression tests preserve existing report intake, moderation-output visibility, account deletion, API, and push behavior.
- Manual checks are limited to Retool configuration and external application/device behavior that this repository cannot automate.

Acceptance scenarios are automated at the nearest reliable boundary rather than requiring a new cross-process Flutter integration suite. PostgreSQL tests must run with `TEST_DATABASE_REQUIRED=true`; skipped database tests are not release evidence. Time-sensitive tests use an injected clock and deterministic worker scheduling rather than wall-clock sleeps.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-014, AC-015 | AT-001, IT-012, UT-010 | Acceptance / Integration / Unit | Yes |
| BR-002 | AC-006, AC-009 | AT-007, AT-008, IT-005, IT-006 | Acceptance / Integration | Yes |
| BR-003 | AC-016, AC-018 | AT-002, AT-006, IT-009 | Acceptance / Integration | Yes |
| BR-004 | AC-019, AC-020, AC-021 | AT-005, IT-003, IT-010, IT-011, MAN-001 | Acceptance / Integration / Manual | Mixed |
| BR-005 | AC-022 | UT-013, IT-019 | Unit / Integration | Yes |
| BR-006 | AC-023, AC-025, AC-038, AC-039 | AT-004, AT-007, AT-010, IT-008, IT-016, IT-018, IT-026 | Acceptance / Integration | Yes |
| BR-007 | AC-028 | IT-014, REG-006 | Integration / Regression | Yes |
| FR-001 | AC-002, AC-034 | AT-009, IT-001, IT-002 | Acceptance / Integration | Yes |
| FR-002 | AC-003 | AT-009, IT-002 | Acceptance / Integration | Yes |
| FR-003 | AC-019, AC-030 | AT-005, IT-010, UT-011 | Acceptance / Integration / Unit | Yes |
| FR-004 | AC-020 | AT-005, IT-011 | Acceptance / Integration | Yes |
| FR-005 | AC-021 | AT-005, IT-003 | Acceptance / Integration | Yes |
| FR-006 | AC-008, AC-018, AC-038, AC-039 | AT-006, AT-007, AT-010, IT-007, IT-008, IT-009 | Acceptance / Integration | Yes |
| FR-007 | AC-001, AC-014, AC-030, AC-033 | AT-001, IT-012, IT-024 | Acceptance / Integration | Yes |
| FR-008 | AC-001, AC-014, AC-015, AC-031 | AT-001, UT-010, MAN-004 | Acceptance / Unit / Manual | Mixed |
| FR-009 | AC-016, AC-036 | AT-002, IT-025, MAN-002 | Acceptance / Integration / Manual | Mixed |
| FR-010 | AC-017 | AT-002, UT-008, MAN-002 | Acceptance / Unit / Manual | Mixed |
| FR-011 | AC-018, AC-040, AC-043 | AT-006, UT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-004, AC-032, AC-037 | AT-008, UT-002, IT-004, IT-023 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-005, AC-007, AC-008 | AT-007, UT-003, IT-005, IT-007, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-006, AC-021 | AT-007, IT-003, IT-005, IT-022 | Acceptance / Integration | Yes |
| FR-015 | AC-009, AC-042 | AT-008, IT-006 | Acceptance / Integration | Yes |
| FR-016 | AC-007, AC-008, AC-038 | AT-007, AT-010, IT-007, IT-008 | Acceptance / Integration | Yes |
| FR-017 | AC-010 | AT-007, IT-006, IT-007 | Acceptance / Integration | Yes |
| FR-018 | AC-011, AC-012, AC-013, AC-041 | AT-003, UT-004, IT-013, IT-014 | Acceptance / Unit / Integration | Yes |
| FR-019 | AC-023, AC-038, AC-039 | AT-004, AT-007, AT-010, UT-012, IT-006, IT-008, IT-016 | Acceptance / Unit / Integration | Yes |
| FR-020 | AC-024, AC-044 | AT-011, UT-007, UT-012, IT-017, IT-027 | Acceptance / Unit / Integration | Yes |
| FR-021 | AC-025, AC-048 | AT-004, UT-007, IT-018, IT-026, MAN-003 | Acceptance / Unit / Integration / Manual | Mixed |
| FR-022 | AC-026 | UT-006, IT-012 | Unit / Integration | Yes |
| FR-023 | AC-027 | AT-005, IT-011, IT-021, MAN-001 | Acceptance / Integration / Manual | Mixed |
| FR-024 | AC-021, AC-022 | AT-005, UT-013, IT-003, IT-019 | Acceptance / Unit / Integration | Yes |
| FR-025 | AC-022 | UT-013, IT-019 | Unit / Integration | Yes |
| FR-026 | AC-029 | UT-009, IT-015, REG-002 | Unit / Integration / Regression | Yes |
| FR-027 | AC-032, AC-033 | AT-008, UT-002, IT-024 | Acceptance / Unit / Integration | Yes |
| FR-028 | AC-035 | AT-008, UT-009, IT-015 | Acceptance / Unit / Integration | Yes |
| FR-029 | AC-003, AC-034 | AT-009, IT-002, IT-008, IT-009 | Acceptance / Integration | Yes |
| FR-030 | AC-039 | AT-010, IT-008 | Acceptance / Integration | Yes |
| FR-031 | AC-040 | AT-006, UT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-032 | AC-036 | AT-002, UT-005, IT-002, IT-025 | Acceptance / Unit / Integration | Yes |
| FR-033 | AC-037 | AT-008, UT-002, IT-023 | Acceptance / Unit / Integration | Yes |
| FR-034 | AC-005, AC-007, AC-038 | AT-007, UT-001, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-035 | AC-044 | AT-011, IT-027 | Acceptance / Integration | Yes |
| FR-036 | AC-045 | IT-020, REG-008 | Integration / Regression | Yes |
| NFR-001 | AC-021 | AT-005, IT-003, IT-022 | Acceptance / Integration | Yes |
| NFR-002 | AC-021, AC-027 | AT-005, IT-003, IT-011, IT-021 | Acceptance / Integration | Yes |
| NFR-003 | AC-014, AC-020, AC-027, AC-046 | UT-006, IT-011, IT-012, IT-021, IT-023 | Unit / Integration | Yes |
| NFR-004 | AC-030 | UT-011, IT-010, IT-012, REG-005 | Unit / Integration / Regression | Yes |
| NFR-005 | AC-012 | AT-003, UT-004, IT-013, IT-014 | Acceptance / Unit / Integration | Yes |
| NFR-006 | AC-023, AC-024 | UT-007, UT-012, IT-016, IT-017, REG-004 | Unit / Integration / Regression | Yes |
| NFR-007 | AC-015, AC-031 | AT-001, UT-010, MAN-004 | Acceptance / Unit / Manual | Mixed |
| NFR-008 | AC-047 | UT-014, IT-021 | Unit / Integration | Yes |
| RULE-001 | AC-004, AC-005, AC-027 | UT-002, UT-003, IT-004, IT-007, IT-011 | Unit / Integration | Yes |
| RULE-002 | AC-005, AC-038 | UT-001, IT-007 | Unit / Integration | Yes |
| RULE-003 | AC-004, AC-039 | UT-003, IT-004, IT-008 | Unit / Integration | Yes |
| RULE-004 | AC-006, AC-009, AC-042 | UT-003, IT-005, IT-006, IT-022 | Unit / Integration | Yes |
| RULE-005 | AC-017, AC-018 | AT-002, AT-006, UT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| RULE-006 | AC-007, AC-014 | AT-001, AT-007, IT-007, IT-012 | Acceptance / Integration | Yes |
| RULE-007 | AC-002, AC-003 | AT-009, IT-001, IT-002 | Acceptance / Integration | Yes |
| RULE-008 | AC-028, AC-029 | IT-014, IT-015, REG-002, REG-006 | Integration / Regression | Yes |
| RULE-009 | AC-023, AC-033, AC-038, AC-039 | UT-012, IT-006, IT-007, IT-008, IT-016, IT-024 | Unit / Integration | Yes |
| RULE-010 | AC-024 | UT-012, IT-017 | Unit / Integration | Yes |
| RULE-011 | AC-033 | IT-012, IT-024 | Integration | Yes |
| RULE-012 | AC-042 | AT-008, UT-002, IT-006 | Acceptance / Unit / Integration | Yes |
| RULE-013 | AC-037 | UT-002, IT-023 | Unit / Integration | Yes |
| RULE-014 | AC-037 | UT-002, IT-023 | Unit / Integration | Yes |
| RULE-015 | AC-004, AC-037 | AT-008, UT-002, IT-004, IT-023 | Acceptance / Unit / Integration | Yes |
| RULE-016 | AC-040, AC-043 | AT-006, UT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| RULE-017 | AC-043 | AT-006, UT-008, IT-009 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Affected User Understands Account Standing And History
Requirement IDs: BR-001, FR-007, FR-008, NFR-007, RULE-006
Acceptance Criteria: AC-001, AC-014, AC-015, AC-031
Priority: Must
Level: Acceptance
Automation Target: `app/test/moderation/pages/account_standing_page_test.dart`

```gherkin
Feature: Owner moderation transparency
  Scenario: Affected user reviews current standing and consequential history
    Given an authenticated user has warning, visibility, active-strike, expired, and overturned decisions
    When the user opens Account standing from Settings
    Then the page identifies the current enforcement state, active strike count, and threshold
    And it presents every consequential decision chronologically with localized user-safe explanations
    And it does not expose reports, no-action outcomes, internal evidence, or moderator data
    And its wording does not frame unused strikes as permission to violate policy
```

### AT-002: User Starts An Appeal Without Creating One Automatically
Requirement IDs: BR-003, FR-009, FR-010, FR-032, RULE-005
Acceptance Criteria: AC-016, AC-017, AC-036
Priority: Must
Level: Acceptance
Automation Target: `app/test/moderation/pages/account_standing_page_test.dart`

```gherkin
Feature: Email appeal contact
  Scenario: User invokes the appeal action
    Given an appealable decision with public reference MOD-<case UUID>
    When the user selects Appeal
    Then CraftSky launches a mailto addressed to moderation@craftsky.social
    And the subject contains the public case reference
    And copyable address and reference fallbacks remain available
    And returning from or abandoning the composer does not mark an appeal pending or change enforcement
```

### AT-003: Suspended User Retains Only Approved Capabilities
Requirement IDs: FR-018, NFR-005
Acceptance Criteria: AC-011, AC-012, AC-013, AC-041
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/suspended_capability_matrix_test.go`

```gherkin
Feature: Restricted CraftSky participation
  Scenario Outline: Server enforces the suspended capability matrix
    Given an authenticated suspended user using an untrusted client
    When they invoke <operation>
    Then AppView returns <result>
    And denied operations create no local or PDS side effect

    Examples:
      | operation | result |
      | read public content | allowed |
      | view moderation history | allowed |
      | submit a report | allowed |
      | manage blocks or mutes | allowed |
      | manage notification, language, device, recent-search, or saved-content state | allowed |
      | cancel or delete existing private scheduled or migration work | allowed |
      | delete owned public content | allowed |
      | delete account or sign out | allowed |
      | create or edit public content | denied with standard error |
      | ordinary social interaction | denied with standard error |
      | upload, prepare, schedule, publish, onboard, or start an import | denied with standard error |
      | invoke an unclassified mutation | denied with standard error |
```

### AT-004: Moderation Notification Activates The Correct Account And Opens History
Requirement IDs: BR-006, FR-019, FR-021
Acceptance Criteria: AC-023, AC-025, AC-048
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/notification_open_flow_test.dart`, `app/test/router/moderation_routes_test.dart`

```gherkin
Feature: Moderation system notification
  Scenario: User taps a moderation notice
    Given a durable actorless CraftSky moderation notification for a public case reference and opaque account binding
    When the notification is opened while another retained account is active
    Then the app activates and rechecks the exact intended account lease
    And it navigates to the matching moderation-history entry only after activation
    And the notice is not rendered as activity from another user
    And an invalid, ambiguous, removed, or stale binding reveals no case data and does not navigate
```

### AT-005: Moderator Reviews And Resolves A Case Safely
Requirement IDs: BR-004, FR-003, FR-004, FR-005, FR-014, FR-023, FR-024, NFR-001, NFR-002
Acceptance Criteria: AC-019, AC-020, AC-021, AC-027, AC-030
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/moderation_admin_handler_test.go`

```gherkin
Feature: Trusted moderation workflow
  Scenario: Authenticated moderator resolves an open case
    Given a scoped moderator requests a filtered cursor-paginated queue and case detail
    When the moderator submits a valid decision for the current case revision
    Then the decision, selected effects, projections, audit attribution, and notification intent commit atomically
    And retrying the source replay identity creates no duplicate effect
    And a stale revision is rejected without side effects
    And member, development, missing, and invalid credentials cannot use the endpoint
```

### AT-006: One Appeal Lifecycle Records Review Without Trusting Email Identity
Requirement IDs: BR-003, FR-006, FR-011, FR-031, RULE-005, RULE-016, RULE-017
Acceptance Criteria: AC-018, AC-040, AC-043
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/moderation/appeal_service_integration_test.go`

```gherkin
Feature: Appeal lifecycle
  Scenario: Staff records and resolves correspondence for an owner-visible decision
    Given correspondence from any sender contains a valid public case reference
    When trusted staff receives but has not confirmed the correspondence
    Then no appeal lifecycle or owner-visible pending state exists
    When an authenticated moderator confirms the appeal and later resolves it as upheld or changed
    Then the case has one appeal lifecycle progressing through pending to the outcome
    And repeated correspondence attaches without creating another lifecycle
    And the sender is not treated as authenticated and enforcement remains active until an authorized change
    And changed effects are represented by attributable append-only events
```

### AT-007: Strike Expiry And Reversal Restore Threshold-Only Access
Requirement IDs: BR-002, BR-006, FR-006, FR-013, FR-014, FR-016, FR-017, FR-019, FR-034, RULE-002, RULE-004, RULE-006, RULE-009
Acceptance Criteria: AC-005, AC-006, AC-007, AC-008, AC-010, AC-038
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/moderation/expiry_processor_integration_test.go`

```gherkin
Feature: Strike-based enforcement
  Scenario: Threshold-only suspension converges after one strike expires
    Given an account is threshold-suspended by three active strikes and has no severe basis
    When one strike reaches its clamped 12-calendar-month UTC deadline while the worker is unavailable
    Then the strike is due but the stored suspension projection remains authoritative
    When restart-safe processing resumes within one hour
    Then expiry, active count two, and restored participation are persisted atomically once
    And the expired strike remains in history
    And exactly one moderation notification is created because effective enforcement changed

  Scenario: Severe suspension survives strike changes
    Given an account has an active severe suspension basis
    When strikes expire, are overturned, or the active count falls below three
    Then severe suspension remains until an authorized explicit restoration or reversal
```

### AT-008: Decision Dimensions Apply Only Explicit Eligible Consequences
Requirement IDs: FR-012, FR-015, FR-027, FR-028, FR-033, RULE-003, RULE-012, RULE-013, RULE-014, RULE-015
Acceptance Criteria: AC-004, AC-009, AC-032, AC-035, AC-037, AC-042
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/moderation/adjudication_service_integration_test.go`

```gherkin
Feature: Moderation decision policy
  Scenario Outline: Decision validates disposition, reason, and explicit effects
    Given a case has any number of attached reports
    When a moderator submits <decision>
    Then the service produces <result>

    Examples:
      | decision | result |
      | violation without consequences | rejection without side effects |
      | noAction with a consequence | rejection without side effects |
      | formalWarning only | private owner warning only |
      | viewer warn only | existing visibility warning only |
      | formalWarning and viewer warn | both independent effects |
      | other with strike or severe suspension | rejection without side effects |
      | eligible severe reason without severity rationale | rejection without side effects |
      | eligible severe reason with severe only | severe suspension and no strike |
      | eligible severe reason with severe and strike | both independent effects |
```

### AT-009: Reports Form Stable Non-Reopening Cases
Requirement IDs: FR-001, FR-002, FR-029, FR-032, RULE-007
Acceptance Criteria: AC-002, AC-003, AC-034, AC-036
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/moderation/adjudication_service_integration_test.go`

```gherkin
Feature: Moderation case lifecycle
  Scenario: Concurrent reports target the same canonical subject
    Given no case is open for a canonical subject
    When two accepted reports arrive concurrently
    Then exactly one UUIDv4-backed case is open and both reports are attached

  Scenario: A later report follows a resolved case
    Given all cases for a canonical subject are resolved
    When another report is accepted
    Then a new case and public reference are created
    And prior case history is unchanged and the prior case does not reopen
```

### AT-010: Partial Reversal And Reapplication Preserve Audit History
Requirement IDs: FR-006, FR-016, FR-019, FR-030, RULE-003, RULE-009
Acceptance Criteria: AC-008, AC-039
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/moderation/adjudication_service_integration_test.go`

```gherkin
Feature: Effect-level correction
  Scenario: Moderator reverses and later reapplies selected effects
    Given a resolved decision has an active strike, takedown, and severe suspension
    When an authorized command reverses only the strike
    Then the takedown and severe suspension remain active
    And the original decision and reversal remain auditable
    When a later command reapplies the strike with a rationale and distinct replay identity
    Then the case's one logical strike is active with a new issuance timestamp and expiry
    And no second strike is created
    And each changed consequence creates one durable notification intent
```

### AT-011: User Manages The Moderation Push Preference
Requirement IDs: FR-020, FR-035
Acceptance Criteria: AC-024, AC-044
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/pages/notification_settings_page_test.dart`

```gherkin
Feature: Moderation push preference
  Scenario: User disables moderation push delivery
    Given an authenticated user opens notification settings
    When they disable the independent moderation category
    Then Flutter persists the preference through the existing notification settings API
    And a reload displays moderation pushes as disabled without changing other categories
    And subsequent moderation history and enforcement still update without provider delivery attempts
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-034, RULE-002 | AC-005, AC-007, AC-038 | Calculate the due deadline and distinguish it from the worker-commit effective time. | Ordinary dates, month ends, leap-day issuance, non-UTC source offsets normalized to UTC, pending/processed projections. | Due deadline is the same UTC wall-clock time 12 calendar months later with Feb 29 clamped to Feb 28; strike remains projected active until one atomic worker commit within the next hour. | `appview/internal/moderation/expiry_test.go` |
| UT-002 | FR-012, FR-027, FR-033, RULE-001, RULE-012, RULE-013, RULE-014, RULE-015 | AC-004, AC-032, AC-037, AC-042 | Validate disposition, consequence, reason, evidence-note, severity-rationale, and independent severe/strike combinations. | Table of every approved reason and consequence combination plus malformed combinations. | Only explicit eligible combinations pass; `violation` needs effects and evidence notes; `noAction` has none; severe rationale is required; `other` cannot punish account. | `appview/internal/moderation/policy_test.go` |
| UT-003 | FR-013, RULE-003, RULE-004 | AC-004, AC-005, AC-006, AC-007, AC-008, AC-009, AC-039 | Derive active count and independent suspension bases from logical strike/effect events. | Active, expired, overturned, reapplied strikes; threshold and severe basis combinations. | Count includes each case at most once; threshold begins at three; bases do not overwrite one another. | `appview/internal/moderation/standing_test.go` |
| UT-004 | FR-018, NFR-005 | AC-011, AC-012, AC-013, AC-041 | Classify every route capability for a suspended account with a fail-closed default. | Current catalogue grouped into reads, private maintenance, safety/account/deletion, owned removal, public participation/preparation, and unknown mutation. | Confirmed retained categories pass; public participation/preparation and unclassified mutations fail closed. | `appview/internal/routes/suspended_capability_matrix_test.go` |
| UT-005 | FR-032 | AC-036 | Parse, format, and compare public case references. | Valid UUIDv4, mixed-case `MOD-` forms, malformed/non-v4 UUIDs, internal IDs. | Canonical `MOD-<lowercase UUID>` output; case-insensitive lookup; invalid values rejected; no alternate public identifier. | `appview/internal/moderation/reference_test.go` |
| UT-006 | FR-022, NFR-003 | AC-014, AC-020, AC-026 | Build owner-safe and moderator-detail representations from report snapshot and decision data. | Sensitive sentinels in reporter, report text, evidence, moderator, source, device, and credential fields; missing/deleted live subject. | Owner output contains only safe snapshot/reason/detail and canonical references; moderator detail contains authorized evidence; no accidental field reuse. | `appview/internal/moderation/presentation_test.go` |
| UT-007 | FR-020, FR-021, NFR-006 | AC-023, AC-024, AC-025 | Classify moderation notices and construct actorless, account-bound provider payloads. | Decision/reversal/expiry events, preference states, case references, internal sensitive IDs. | Eligible notice has no actor and routes by safe public case identity; disabled preference suppresses delivery; sensitive fields are absent. | `appview/internal/notifications/category_test.go`, `appview/internal/push/payload_test.go` |
| UT-008 | FR-010, FR-011, FR-031, RULE-005, RULE-016, RULE-017 | AC-017, AC-018, AC-040, AC-043 | Validate untrusted correspondence, moderator confirmation, and one appeal lifecycle. | No owner-visible decision, unconfirmed/confirmed first correspondence, repeated correspondence, pending/upheld/changed, expiry/restoration event, arbitrary sender. | Unconfirmed correspondence creates no lifecycle; only moderator confirmation for an owner-visible case creates pending; sender remains unauthenticated; terminal outcome and append-only changes are explicit. | `appview/internal/moderation/appeal_test.go` |
| UT-009 | FR-026, FR-028 | AC-029, AC-035 | Map formal warnings and visibility consequences independently. | Formal-only, viewer-warn-only, both, hide, takedown, reversal. | Owner warning and existing moderation outputs match exactly selected effects; no parallel visibility state is introduced. | `appview/internal/moderation/visibility_effects_test.go` |
| UT-010 | BR-001, FR-008, NFR-007 | AC-001, AC-014, AC-015, AC-031 | Parse owner API models and derive display labels for all standing/history states. | Zero through three strikes, severe basis, active/expired/overturned effects, unknown additive JSON fields. | Correct localized display model without allowance wording; additive fields do not break parsing. | `app/test/moderation/models/account_standing_test.dart`, `app/test/moderation/models/moderation_history_test.dart` |
| UT-011 | FR-003, NFR-004 | AC-019, AC-030 | Validate list limits, filters, cursor handling, canonical identifiers, and error serialization. | Minimum/maximum/overflow limits, malformed cursor, DID/AT-URI values, validation errors. | Bounded lists, opaque cursors, canonical IDs, camelCase fields, standard error envelope. | `appview/internal/api/envelope/cursor_test.go`, `appview/internal/api/moderation_request_test.go` |
| UT-012 | FR-019, FR-020, RULE-009, RULE-010 | AC-023, AC-024, AC-033, AC-038, AC-039 | Decide whether each append-only event creates notification work and whether current preference permits provider delivery. | Report, noAction, consequential decision, reversal/reapplication/restoration with and without consequence changes, appeal status/upheld-only changes, expiry with/without enforcement change, preference changes. | Exactly consequence/enforcement-changing events enqueue once; status-only events do not; current disabled preference blocks dispatch only. | `appview/internal/notifications/moderation_policy_test.go` |
| UT-013 | BR-005, FR-024, FR-025 | AC-021, AC-022 | Normalize trusted Retool and future adapter commands into one adjudication input. | Equivalent source events, source/replay identities, spoofed actor fields. | Equivalent policy input and validation; authenticated context supplies actor/source; replay identity remains stable. | `appview/internal/moderation/adapter_test.go` |
| UT-014 | NFR-008 | AC-047 | Evaluate concrete alert threshold predicates at each boundary. | Four/five auth failures inside/outside five minutes; expiry age at/beyond one hour; eligible notification age below/at 15 minutes. | Alert condition activates exactly at each configured threshold and remains inactive below or outside it. | `appview/internal/observability/moderation_alerts_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, RULE-007 | AC-002 | Race two first accepted reports for one canonical subject. | PostgreSQL schema; two report transactions synchronized before case association. | Commit both report intakes concurrently. | One open case exists, both immutable reports attach, and uniqueness/concurrency errors are handled without losing intake. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-002 | FR-001, FR-002, FR-029, FR-032, RULE-007 | AC-003, AC-034, AC-036 | Enforce report-origin and no-reopen case lifecycle. | Resolved case and retained history for one subject. | Attempt reportless creation, append appeal/reversal, then accept a new report. | Reportless request is rejected; append events do not reopen; new report creates a distinct UUIDv4 case/reference without modifying prior history. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-003 | FR-005, FR-014, FR-024, NFR-001, NFR-002 | AC-021 | Verify atomic decision transaction, replay, stale revision, and attribution. | Open case, moderator identity, replay ID; failure triggers for each projection/outbox write. | Submit, retry, submit stale revision, and inject each write failure. | One attributed decision on success; retry returns same result; stale command has no effects; every injected failure rolls back all effects. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-004 | FR-012, RULE-001, RULE-003, RULE-015 | AC-004 | Prove reports and non-strike decisions cannot create strikes and one case cannot contribute twice. | Case with many reports and decisions without strike, then explicit eligible strike. | Resolve and retry/reapply paths. | No implicit strike; one logical case strike only; report count never affects count. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-005 | FR-013, FR-014, RULE-004 | AC-006 | Issue the third active strike transactionally. | Account with two active strikes on distinct cases. | Authorized decision explicitly issues a strike on a third case. | Decision commit stores count three and threshold suspension in the same transaction. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-006 | FR-015, FR-017, FR-019, RULE-004, RULE-009, RULE-012 | AC-009, AC-010, AC-023, AC-042 | Exercise independent severe/strike effects and explicit restoration notification. | Eligible documented severe case below threshold. | Select severe only, severe plus strike, then restore severe basis. | Severe-only creates no strike; combined selection creates both; strike changes do not lift severe; restoration removes only severe basis and creates one notification because enforcement changes. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-007 | FR-006, FR-013, FR-016, FR-017, FR-019, FR-034, RULE-001, RULE-002, RULE-004, RULE-006, RULE-009 | AC-005, AC-007, AC-010, AC-038 | Persist expiry restart-safely under the projection-authoritative effective-time model. | Injected clock; leap-day and ordinary strikes; threshold-only and overlapping severe accounts; worker stopped at deadline. | Run before deadline, at deadline without processing, after restart, and on retry through the one-hour boundary. | No early expiry; due-but-unprocessed projection remains active; one atomic expiry/count/restoration commit occurs by one hour; severe basis remains; notification only for effective enforcement change. | `appview/internal/moderation/expiry_processor_integration_test.go` |
| IT-008 | FR-006, FR-013, FR-016, FR-019, FR-029, FR-030, RULE-003, RULE-009 | AC-008, AC-023, AC-039 | Reverse a subset and reapply an effect in a resolved case with notification eligibility. | Decision with strike, visibility, and severe effects. | Reverse selected effects; retry; reapply with missing/valid rationale and replay identities. | Unselected effects remain; missing rationale or reused identity rejects; valid reapplication reuses one logical strike with new issuance/expiry; full sequence is append-only; each consequence-changing command creates one intent. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-009 | BR-003, FR-006, FR-011, FR-029, FR-031, RULE-005, RULE-016, RULE-017 | AC-018, AC-040, AC-043 | Store untrusted correspondence and one moderator-confirmed appeal lifecycle. | Owner-visible resolved case and arbitrary email senders. | Record unconfirmed correspondence, authenticated confirmation, duplicates/later messages, and upheld/changed resolution. | Unconfirmed intake creates no lifecycle or owner state; moderator confirmation creates one pending lifecycle; sender grants no authority/disclosure; pending preserves enforcement; outcome/effect changes append without reopening. | `appview/internal/moderation/appeal_service_integration_test.go` |
| IT-010 | BR-004, FR-003, NFR-004 | AC-019, AC-030 | Serve admin queue filters and opaque pagination. | Cases across open/resolved states and canonical subject types. | Request supported filters, bounded page sizes, next pages, malformed cursor. | Stable authorized ordering, no duplicate/omitted rows, opaque cursor only when needed, camelCase response, standard malformed-cursor error. | `appview/internal/api/moderation_admin_handler_test.go` |
| IT-011 | BR-004, FR-004, FR-023, NFR-002, NFR-003, RULE-001 | AC-020, AC-027 | Enforce separate admin auth and curated case detail. | Sensitive case sentinels; valid admin, member, dev moderation, missing, invalid, revoked credentials. | Request detail and submit command under each credential. | Only current scoped admin succeeds; identity is server-derived and audited; denials leak no data; owner/member endpoints cannot retrieve private detail. | `appview/internal/api/moderation_admin_handler_test.go` |
| IT-012 | BR-001, FR-007, FR-022, NFR-003, NFR-004, RULE-006, RULE-011 | AC-001, AC-014, AC-026, AC-030, AC-033 | Return private owner standing and paginated chronological history with strict field allowlist. | Mixed consequential/noAction decisions, deleted/edited subjects, report snapshots, sensitive sentinels. | Request pages as affected owner and another member. | Owner receives safe current standing and all consequential entries in order; other member denied; noAction/private fields absent; safe snapshot fallback works; wire contract is canonical. | `appview/internal/api/moderation_history_handler_test.go` |
| IT-013 | FR-018, FR-035, NFR-005 | AC-011, AC-012, AC-013, AC-041, AC-044 | Execute the full registered-route suspended capability matrix. | Suspended member; route catalogue; spies around handlers/stores/PDS client. | Invoke every authenticated route category, including reads, private preferences/search/saves, deletion/cancellation, reporting/safety, publication/engagement, and an unclassified mutation. | Confirmed reads/private maintenance/safety/account/owned-removal operations work; public participation/preparation and unclassified mutations fail before handlers with no side effects. | `appview/internal/routes/suspended_capability_matrix_test.go` |
| IT-014 | BR-007, FR-018, NFR-005, RULE-008 | AC-012, AC-028, AC-041 | Assert denied enforcement and suspension transitions never mutate PDS-owned data. | Recording PDS client; local/PDS state snapshots; threshold, severe, expiry, reversal, denied mutation, and owner-requested deletion cases. | Apply enforcement transitions and capability attempts. | Enforcement performs zero PDS delete/mutation calls; denied actions have no effects; explicitly permitted owner removal remains distinguishable and works. | `appview/internal/moderation/enforcement_store_integration_test.go` |
| IT-015 | FR-026, FR-028, RULE-008 | AC-029, AC-035 | Apply and negate selected existing moderation outputs. | Subjects visible through existing AppView policy. | Decide formal warning, warn, hide, takedown, combinations, and reversals. | Existing append-only moderation outputs alone govern viewer visibility; formal warning remains owner-only; selected reversals restore only relevant visibility. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-016 | BR-006, FR-019, NFR-006, RULE-009 | AC-023, AC-033, AC-038, AC-039 | Atomically create exactly one durable moderation notification intent for the complete event policy. | Decision/reversal/reapplication/restoration/noAction/report/appeal-status/expiry cases; injected transaction failure and replay. | Commit or fail each event and inspect outbox. | Consequence/enforcement-changing state and intent commit together once; rollback leaves neither; report/noAction/status-only/quiet-expiry events create none; retries do not duplicate. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-017 | FR-020, NFR-006, RULE-010 | AC-024 | Recheck moderation preference during durable dispatch. | Eligible installations; preference enabled/disabled before enqueue, after enqueue, and after lease claim. | Dispatch and retry queued work. | Disabled current preference prevents provider call only; enabled preference uses normal retry/account routing; history and enforcement are unchanged. | `appview/internal/push/dispatcher_test.go` |
| IT-018 | BR-006, FR-021 | AC-025 | Serialize the backend actorless moderation system payload. | Moderation intent, account-bound installations, sensitive sentinels. | Fan out and decode provider payload. | Payload identifies CraftSky moderation, contains the existing opaque account-subscription binding and safe destination data, and contains no actor, raw DID, or private/internal IDs. | `appview/internal/push/payload_test.go` |
| IT-019 | BR-005, FR-024, FR-025 | AC-022 | Send equivalent admin and future-adapter inputs through one service. | Two adapters producing equivalent commands and replayed source IDs. | Execute commands and replay each. | Same policy validation and transactional effects; no direct table write seam; each replay is a no-op. | `appview/internal/moderation/adapter_integration_test.go` |
| IT-020 | FR-001, FR-006, FR-011, FR-013, FR-019, FR-024, FR-036, NFR-001, RULE-003 | AC-002, AC-004, AC-018, AC-021, AC-023, AC-045 | Verify migration constraints and deterministic legacy treatment. | Full pre-feature migration chain with existing report and active/negated moderation-output rows. | Apply migrations, enable feature, inspect legacy/new data, then accept a new report. | Legacy rows survive and outputs remain effective, but no legacy-derived case/history/strike/appeal/suspension/notification exists; new intake groups normally; new constraints enforce lifecycle/replay invariants. | `appview/internal/db/moderation_cases_migration_test.go` |
| IT-021 | FR-023, NFR-002, NFR-003, NFR-008 | AC-021, AC-027, AC-046, AC-047 | Verify safe audit/log/metric representations and concrete alert conditions. | Successful/failed admin auth, decision, appeal, expiry, stalled notification work, threshold boundary times, and sensitive sentinels. | Execute operations and capture audit/log/metric/alert sinks. | Safe actor/request/case identifiers and timestamps are present; sensitive values are absent; alert conditions activate at five failures/five minutes, overdue expiry, and 15-minute eligible-notification age only. | `appview/internal/observability/moderation_test.go` |
| IT-022 | FR-013, FR-014, NFR-001, RULE-004 | AC-006, AC-007, AC-021, AC-038 | Serialize a third strike racing expiry or reversal. | Account at two active strikes; synchronized decision and expiry/reversal transactions. | Commit operations in both possible lock orders. | One correct final count and independent enforcement state; no partial or duplicate transition. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-023 | FR-012, FR-033, NFR-003, RULE-013, RULE-014, RULE-015 | AC-014, AC-020, AC-037 | Persist and serialize reason, evidence, severity rationale, and optional user-safe detail separately. | Every reason; unique sensitive sentinels; optional detail absent/present; prohibited combinations. | Internal fields remain admin-only; localized reason and explicitly authored detail are owner-safe; report/evidence text is never copied automatically. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-024 | FR-007, FR-027, RULE-009, RULE-011 | AC-032, AC-033 | Resolve `noAction` privately. | Open reported case and valid private moderator notes. | Submit `noAction`, then request owner history and inspect effects/outbox. | Case resolves and audit remains private; no consequences, owner item, standing change, or notification exists. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-025 | FR-009, FR-032 | AC-016, AC-036 | Verify owner API and deep-link lookup expose only the case reference. | Case with internal decision/event IDs and mixed-case public lookup. | Fetch owner response and resolve history link. | `MOD-<UUIDv4>` is the only exposed moderation identifier, lookup is case-insensitive, and email data uses it. | `appview/internal/api/moderation_history_handler_test.go`, `app/test/router/moderation_routes_test.dart` |
| IT-026 | FR-021 | AC-025, AC-048 | Reuse secure account activation before Flutter moderation navigation. | Two retained accounts with distinct opaque bindings; invalid, ambiguous, removed, and stale lease cases. | Open a moderation notification while the other account is active. | Exact recipient account activates and lease is rechecked before case navigation; invalid/ambiguous/removed/stale cases show no details and do not navigate. | `app/test/notifications/notification_open_flow_test.dart`, `app/test/router/moderation_routes_test.dart` |
| IT-027 | FR-020, FR-035 | AC-024, AC-044 | Read, update, persist, and reload the independent moderation preference. | Existing notification settings with other categories; API fake/store and provider overrides. | Toggle moderation off/on and reload repository/page; dispatch an eligible intent while off. | Wire JSON is camelCase; only moderation value changes; reloaded UI matches storage; disabled value suppresses provider call without changing history/enforcement. | `appview/internal/notifications/preferences_test.go`, `app/test/notifications/pages/notification_settings_page_test.dart` |
| IT-028 | FR-037 | AC-049 | Open current moderated post content from owner history without widening visibility. | Hidden indexed post, affected author, another viewer, valid/mismatched snapshots, and terminal owner. | Read direct post/thread root as each viewer and activate the Flutter history action. | The author reaches the existing post thread; another viewer receives not found; ordinary reads and terminal-owner filtering remain unchanged; invalid snapshots have no action. | `appview/internal/api/post_store_test.go`, `appview/internal/api/post_test.go`, `appview/internal/api/terminal_visibility_integration_test.go`, `app/test/moderation/pages/account_standing_page_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Duplicate private reports remain independently accepted intake evidence. | FR-001, RULE-007 | AC-002 | Existing report submission tests continue to accept duplicates while new assertions prove they attach to one open case rather than deduplicating reports. |
| REG-002 | Existing `warn`, `hide`, and `takedown` outputs remain the visibility authority. | FR-026, RULE-008 | AC-029 | Existing moderation store/read-path tests pass with adjudication-created apply/negate events and no second visibility projection. |
| REG-003 | Owner account deletion remains irreversible and separate from reversible moderation suspension. | FR-017, FR-018 | AC-010, AC-011 | Existing owner lifecycle and account deletion tests pass; moderation restoration cannot alter deletion lifecycle state. |
| REG-004 | Existing push retries, leases, account routing, and non-moderation preferences remain unchanged. | FR-019, FR-020, NFR-006 | AC-023, AC-024 | Existing push dispatcher and notification preference suites pass after adding moderation category behavior. |
| REG-005 | Existing `/v1/*` routes retain authentication and error-envelope conventions. | NFR-004 | AC-030 | Route architecture, envelope, and cursor suites pass with additive moderation routes. |
| REG-006 | Moderator enforcement never deletes user PDS records. | BR-007, RULE-008 | AC-028 | PDS spy/secret-boundary tests prove all moderation transitions make zero destructive PDS calls. |
| REG-007 | Existing report abuse controls still apply to suspended reporters. | FR-018 | AC-013 | Existing report validation/rate-control tests run with suspended identity and preserve normal rejection behavior without changing suspension. |
| REG-008 | Full migration chain preserves existing moderation data without inventing owner/enforcement history. | FR-001, FR-026, FR-036, NFR-001 | AC-002, AC-021, AC-029, AC-045 | Release migration verification reaches latest schema; legacy reports stay private, outputs remain effective, no synthetic case/effect is created, and new reports group normally. |
| REG-009 | Moderation-hidden posts remain absent from public and indirect read surfaces. | FR-026, FR-037, RULE-008 | AC-029, AC-049 | Existing visibility tests continue to cover feeds, search, profiles, conversations, quotes, captions, and ordinary post reads; only author-authenticated direct post/thread-root reads use the owner fallback. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Canonical report subjects | Account DID, post AT URI, and business-event AT URI with matching safe snapshots and later edited/deleted live forms. | AT-009, IT-001, IT-002, IT-012, IT-023 |
| TD-002 | Concurrent intake | Two or more distinct accepted reports, reporters, and idempotency identities targeting one canonical subject. | AT-009, IT-001, IT-004 |
| TD-003 | Decision policy table | `violation`/`noAction`; every approved reason; formal warning, warn/hide/takedown, strike, severe effects; required/absent evidence and severity rationale. | AT-008, UT-002, IT-004, IT-006, IT-023, IT-024 |
| TD-004 | Strike timeline | Three case strikes with injected issuance times, leap-day date, exact pre-deadline/due/pending/worker-commit instants, reversal, expiry, and reapplication events. | AT-007, AT-010, UT-001, UT-003, IT-005, IT-007, IT-008, IT-022 |
| TD-005 | Suspension bases | None, threshold only, severe only, and simultaneous threshold plus severe states. | AT-003, AT-007, IT-006, IT-007, IT-013, IT-014 |
| TD-006 | Sensitive disclosure sentinels | Unique reporter DID, report text, device ID, internal notes/evidence, moderator ID, raw source metadata, admin credential, and push secret strings. | UT-006, UT-007, IT-011, IT-012, IT-018, IT-021, IT-023 |
| TD-007 | Public references | Valid UUIDv4, canonical/mixed-case `MOD-` values, malformed UUIDs, UUIDs of other versions, internal decision/event IDs. | AT-002, UT-005, IT-002, IT-009, IT-025 |
| TD-008 | Appeal lifecycle | Owner-visible and noAction cases; arbitrary senders; unconfirmed and moderator-confirmed correspondence; repeated/later correspondence; pending/upheld/changed outcomes. | AT-006, UT-008, IT-009 |
| TD-009 | Pagination set | More cases/history entries than maximum page size with tied timestamps and stable unique ordering keys. | AT-001, AT-005, UT-011, IT-010, IT-012 |
| TD-010 | Suspended route catalogue | Every registered route classified under reads, private settings/search/saves, safety/account/deletion, owned removal, publication/engagement, upload/preparation, migration lifecycle, or unclassified mutation. | AT-003, UT-004, IT-013, IT-014 |
| TD-011 | Notification states | Consequential decision, reversal, reapplication, restoration, report, noAction, appeal status, upheld-only outcome, quiet/restoring expiry; preference changes; multiple account bindings; invalid/ambiguous/removed/stale bindings; retryable provider errors. | AT-004, AT-011, UT-007, UT-012, IT-006, IT-008, IT-016 through IT-018, IT-026, IT-027 |
| TD-012 | Flutter display matrix | All standing/effect/appeal states under narrow/wide layouts, 1x/2x text scale, light/dark themes, semantics, focus traversal, and RTL where directional UI exists. | AT-001, AT-002, AT-004, UT-010, MAN-004 |
| TD-013 | Migration baseline | Pre-feature database containing existing reports and active/negated moderation outputs with sentinels proving no synthetic cases, history, enforcement, appeals, or notifications are generated. | IT-020, REG-008 |
| TD-014 | Alert boundaries | Four/five auth failures across five-minute boundary, expiry jobs below/over one hour late, and eligible notification work below/at 15 minutes old. | UT-014, IT-021 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-004, FR-003, FR-004, FR-005, FR-023, NFR-002 | AC-019, AC-020, AC-021, AC-027 | Retool uses only scoped AppView admin APIs. | Configure the secret; inspect queue/detail; submit valid, replayed, and stale decisions; record appeal/effect change/restoration; revoke or rotate credential; inspect Retool resources and database roles. | Workflow handles success/conflict safely, attribution is retained, revoked credential fails, and Retool has no database-write credential or direct table mutation. |
| MAN-002 | FR-009, FR-010 | AC-016, AC-017, AC-036 | Real email-app and no-email-app behavior. | On representative iOS/Android devices, open appeal CTA, inspect recipient/subject, cancel/return, then repeat where no handler is available and use copy fallbacks. | Correct prefill when supported; fallback remains usable; no appeal/enforcement state changes from launch, cancel, failure, or edited subject. |
| MAN-003 | FR-021, NFR-006 | AC-025, AC-048 | Real provider and OS notification presentation. | Deliver a sandbox moderation push to representative devices, tap under matching and different retained-account states, and observe provider acceptance without display where reproducible. | Display is a CraftSky system notice; existing account binding automatically activates the exact retained account before navigation; invalid routing reveals no case data; history remains authoritative regardless of OS presentation. |
| MAN-004 | FR-008, NFR-007 | AC-015, AC-031 | Final visual, accessibility, and localized-copy review. | Review narrow/wide layouts, light/dark themes, 2x text, screen reader output, keyboard traversal, touch targets, and approved English copy for every standing/history state. | Content remains understandable and operable, status is not conveyed by color alone, and wording does not imply unused strikes are permitted violations. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Retool UI and secret/database permissions cannot be asserted by repository automation. | BR-004, FR-023 | Retool configuration is external to this repository. | Require MAN-001 evidence and operator documentation before enabling production admin credentials. |
| GAP-002 | Real email composer launch and provider/OS push presentation are not deterministic in unit/widget tests. | FR-009, FR-010, FR-021, NFR-006 | External apps and OS delivery are outside process control; provider acceptance is not presentation proof. | Automate URI/payload/routing boundaries and retain MAN-002/MAN-003 device checks. |
| GAP-003 | Only English localization resources currently exist. | FR-008, NFR-007 | Automated tests can enforce generated localization usage but cannot validate untranslated locales. | Validate English and layout/semantics now; add locale-specific copy tests when translations are introduced. |
| GAP-004 | No Flutter `integration_test/` suite currently covers a real notification-to-page or email handoff. | FR-009, FR-021 | Introducing an integration harness is outside this test-design stage and may not improve deterministic coverage of OS-owned flows. | Use widget/router tests plus manual device checks; reconsider a harness during coding design if existing boundaries prove insufficient. |
| GAP-005 | Concrete route bindings may change before implementation. | FR-018, NFR-005 | Product-level capability categories are now fixed, but coding design must bind every current and queued route to them. | Generate the route decision table from the catalogue during coding design and make tests fail for every unclassified mutation. |
| GAP-006 | Exact admin credential rotation mechanism and route names are undecided. | FR-003, FR-023 | These are non-blocking coding-design decisions. | Keep tests behavior-based; bind concrete route/config targets in `04-coding-plan.md` before implementation. |
| GAP-007 | Future Ozone is represented only by a contract test adapter, not a live Ozone integration. | BR-005, FR-025 | Ozone deployment and ingestion are explicit non-goals. | Test the shared service seam and replay contract; defer live adapter conformance to Stage 3. |

High-risk verification requirements:

- `IT-003`, `IT-007`, `IT-013`, `IT-014`, `IT-016`, `IT-020`, `IT-021`, `IT-022`, `IT-026`, and `IT-027` are merge-blocking.
- Atomicity tests must inject failures across every coupled write, not only the happy path.
- Privacy tests must serialize complete responses/log records and search for unique sensitive sentinels.
- Suspended-route coverage must fail when a new mutation route lacks an explicit classification.
- Release evidence must include `just appview-check`; the incomplete unit-only command is not sufficient.

## 10. Out Of Scope

- Live Ozone deployment, report forwarding, and production Ozone event ingestion; only the reusable adjudication adapter contract is tested.
- Native in-app appeal submission, verified-email ownership, and automated email ingestion.
- Ban-evasion detection, cross-DID identity linkage, or cross-case incident grouping.
- Guarantees that an operating system presents an accepted push notification.
- Moderator CLI behavior; no moderator CLI is required for V1.
- Lexicon validation or migration because no lexicon change is permitted or required.
- Automated policy inference from report counts, reason popularity, or scoring; such behavior is prohibited rather than tested as a feature.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-09-09-moderation-strikes/01-requirements.md`
- Test specification: `docs/changes/2026-09-09-moderation-strikes/02-acceptance-tests.md`
- Next review artifact: `docs/changes/2026-09-09-moderation-strikes/03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-09-09-moderation-strikes/`
- Recommended first failing test for implementation: `IT-020`, migration preserves legacy rows without synthesizing case/history/enforcement state and applies the new constraints. This establishes the persistence contract before concurrent intake.
- Suggested test order for implementation: `UT-005`; `IT-020`; `IT-001`/`IT-002`; `UT-002`/`IT-003`/`IT-004`; `UT-001`/`UT-003`/`IT-005` through `IT-008`/`IT-022`; appeals; admin/owner APIs; suspension matrix; complete notification policy and preference; secure account activation/routing; Flutter models/widgets/routes; observability; regression and manual checks.
- Commands discovered: `just appview-test-unit` for fast incomplete Go feedback; `just dev-d` then `just test` for PostgreSQL/MinIO/race coverage; `just appview-check` for release-equivalent backend evidence; `just app-test test/moderation test/router/moderation_routes_test.dart test/notifications` for focused Flutter tests; `just app-analyze` for Flutter static analysis.
- Blocking gaps: No product questions block test design. High-risk document review and explicit approval remain required before implementation. Concrete route/config names and credential rotation mechanics remain coding-design tasks under fixed behavior contracts.
