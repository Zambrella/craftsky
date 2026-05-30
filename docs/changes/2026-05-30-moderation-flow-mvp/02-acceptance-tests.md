# Acceptance Test Specification: Moderation Flow MVP Without Live Ozone/PDS Report Submission

## 1. Test Strategy

This specification covers the moderation-flow MVP described in `01-requirements.md`. The feature is high risk because it introduces private report storage, dev-only moderation mutation controls, and read-path content suppression. Test design therefore emphasizes server-side enforcement, privacy boundaries, route/config gating, and Flutter user-visible behavior.

Recommended approach:

- Start with AppView store and migration tests for private reports, forwarding metadata, and indexed moderation outputs.
- Add AppView handler and route tests for report intake, validation, auth/device requirements, minimal response bodies, and synthetic endpoint gating.
- Add AppView read-path tests at the store/query and handler level for timeline, profile posts/comments, direct post, direct profile, thread/comment, and notification enforcement.
- Add Flutter API client, repository/provider, and widget tests for report submission, validation, retry states, duplicate-submit prevention, report action visibility, and warning rendering.
- Add regression tests proving unmoderated content and existing response compatibility remain unchanged.
- Keep manual checks limited to local-dev smoke, accessibility/localization review, and privacy/log review where automation is not sufficient.

Risk-based review recommendation: **Required before implementation continues.** This carries forward the high-risk status from the requirements because failures can leak private moderation data or incorrectly hide/show user content.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-021 | AT-001, AT-002, AT-012, IT-004, IT-013, UT-001, UT-012 | Acceptance / Integration / Unit | Yes |
| BR-002 | AC-003, AC-004, AC-022 | AT-003, IT-001, IT-004, IT-005, UT-004, REG-007, MAN-003 | Acceptance / Integration / Unit / Manual | Yes + Manual review |
| BR-003 | AC-005, AC-006, AC-035 | AT-003, IT-003, IT-005, UT-004, REG-007 | Acceptance / Integration / Unit / Regression | Yes |
| BR-004 | AC-012, AC-013, AC-014, AC-015, AC-016, AC-033, AC-040 | AT-007, AT-009, AT-010, IT-009, IT-010, IT-011, IT-012, IT-014, UT-005, UT-006, REG-001, REG-004 | Acceptance / Integration / Unit / Regression | Yes |
| BR-005 | AC-017, AC-018, AC-030, AC-039 | AT-008, IT-015, UT-007, UT-013, REG-003, MAN-002 | Acceptance / Integration / Unit / Regression / Manual | Yes + Manual review |
| FR-001 | AC-001, AC-003, AC-007 | AT-001, IT-004, IT-006, UT-001 | Acceptance / Integration / Unit | Yes |
| FR-002 | AC-002, AC-003, AC-008 | AT-002, IT-004, IT-007, UT-001 | Acceptance / Integration / Unit | Yes |
| FR-003 | AC-009 | AT-004, IT-008, REG-006 | Acceptance / Integration / Regression | Yes |
| FR-004 | AC-010, AC-011, AC-034, AC-041 | AT-004, AT-012, IT-006, IT-007, UT-001, UT-002, UT-003, UT-012 | Acceptance / Integration / Unit | Yes |
| FR-005 | AC-007, AC-043 | AT-011, IT-006, IT-016 | Acceptance / Integration | Yes |
| FR-006 | AC-008, AC-044 | AT-011, IT-007, IT-016 | Acceptance / Integration | Yes |
| FR-007 | AC-003, AC-004, AC-042 | AT-003, IT-001, IT-004, UT-004 | Acceptance / Integration / Unit | Yes |
| FR-008 | AC-005, AC-006, AC-035 | AT-003, IT-003, IT-005, UT-004 | Acceptance / Integration / Unit | Yes |
| FR-009 | AC-019, AC-020, AC-036, AC-037 | AT-005, AT-006, IT-002, IT-017, UT-011, REG-006 | Acceptance / Integration / Unit / Regression | Yes |
| FR-010 | AC-019, AC-023 | AT-006, IT-002, UT-009 | Acceptance / Integration / Unit | Yes |
| FR-011 | AC-019, AC-024, AC-038 | AT-006, AT-009, IT-002, IT-018, UT-005, UT-009 | Acceptance / Integration / Unit | Yes |
| FR-012 | AC-023, AC-024, AC-038 | AT-006, AT-009, IT-002, IT-018, UT-005, UT-009 | Acceptance / Integration / Unit | Yes |
| FR-013 | AC-012, AC-014 | AT-007, IT-009, IT-011, REG-001 | Acceptance / Integration / Regression | Yes |
| FR-014 | AC-013, AC-014 | AT-007, IT-009, IT-010, IT-011, REG-001 | Acceptance / Integration / Regression | Yes |
| FR-015 | AC-015 | AT-007, IT-012, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-016 | AC-016 | AT-007, IT-010, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-017 | AC-017, AC-039 | AT-008, IT-015, UT-007, UT-013, REG-003 | Acceptance / Integration / Unit / Regression | Yes |
| FR-018 | AC-018, AC-039 | AT-008, IT-015, UT-007, UT-013, REG-003 | Acceptance / Integration / Unit / Regression | Yes |
| FR-019 | AC-001, AC-002, AC-025 | AT-001, AT-002, AT-012, UT-012, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-020 | AC-026 | AT-012, UT-012 | Acceptance / Unit | Yes |
| FR-021 | AC-027, AC-028, AC-029, AC-045 | AT-012, UT-002, UT-003, UT-014, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-022 | AC-017, AC-018, AC-030, AC-039 | AT-008, UT-007, UT-013, MAN-002 | Acceptance / Unit / Manual | Yes + Manual review |
| FR-023 | AC-024, AC-038 | AT-009, IT-018, UT-005 | Acceptance / Integration / Unit | Yes |
| FR-024 | AC-024 | AT-009, IT-018, UT-005 | Acceptance / Integration / Unit | Yes |
| FR-025 | AC-033 | AT-010, IT-014, REG-005 | Acceptance / Integration / Regression | Yes |
| FR-026 | AC-046 | AT-001, AT-002, IT-004, UT-008 | Acceptance / Integration / Unit | Yes |
| FR-027 | AC-011, AC-041 | AT-004, AT-012, IT-004, UT-002 | Acceptance / Integration / Unit | Yes |
| NFR-001 | AC-020, AC-037 | AT-005, IT-017, UT-011, REG-006 | Acceptance / Integration / Unit / Regression | Yes |
| NFR-002 | AC-030 | AT-003, AT-008, UT-007, UT-013, MAN-003 | Acceptance / Unit / Manual | Yes + Manual review |
| NFR-003 | AC-012, AC-013, AC-014, AC-015, AC-016, AC-033 | AT-007, AT-010, IT-009, IT-010, IT-011, IT-012, IT-014 | Acceptance / Integration | Yes |
| NFR-004 | AC-031 | IT-019, MAN-004 | Integration / Manual | Yes + Manual review |
| NFR-005 | AC-035 | AT-003, IT-003, IT-005, UT-004 | Acceptance / Integration / Unit | Yes |
| RULE-001 | AC-032 | AT-011, IT-020 | Acceptance / Integration | Yes |
| RULE-002 | AC-029 | AT-012, UT-014 | Acceptance / Unit | Yes |
| RULE-003 | AC-012, AC-013, AC-015, AC-016, AC-040 | AT-007, AT-009, IT-009, IT-010, IT-012, UT-005, UT-006 | Acceptance / Integration / Unit | Yes |
| RULE-004 | AC-006, AC-022 | AT-003, IT-005, REG-007, MAN-003 | Acceptance / Integration / Regression / Manual | Yes + Manual review |
| RULE-005 | AC-043, AC-044 | AT-011, IT-016 | Acceptance / Integration | Yes |
| RULE-006 | AC-023, AC-036 | AT-005, AT-006, IT-002, IT-017, UT-009, UT-011 | Acceptance / Integration / Unit | Yes |

## 3. Acceptance Scenarios

### AT-001: User reports another user's post

Requirement IDs: BR-001, FR-001, FR-019, FR-026
Acceptance Criteria: AC-001, AC-003, AC-046
Priority: Must
Level: Acceptance
Automation Target: Flutter widget/provider tests in `app/test/feed/widgets/`, Flutter API tests in `app/test/feed/data/post_api_client_test.dart`, and AppView handler tests in `appview/internal/api/report_test.go`

```gherkin
Feature: Post reporting
  Scenario: Signed-in user reports another user's post
    Given a signed-in user is viewing a post authored by another account
    And the report dialog offers the approved reason taxonomy
    When the user selects an approved reason and submits the report
    Then Flutter calls POST /v1/posts/{did}/{rkey}/reports with reasonType and optional details
    And AppView persists a private report row for the canonical post subject
    And the response body contains only reportId and status "accepted"
    And the user sees a transient success confirmation
```

### AT-002: User reports another user's profile/account

Requirement IDs: BR-001, FR-002, FR-019, FR-026
Acceptance Criteria: AC-002, AC-003, AC-046
Priority: Must
Level: Acceptance
Automation Target: Flutter widget/provider tests in `app/test/profile/`, Flutter API tests in `app/test/profile/data/profile_api_client_test.dart`, and AppView handler tests in `appview/internal/api/report_test.go`

```gherkin
Feature: Profile reporting
  Scenario: Signed-in user reports another user's profile
    Given a signed-in user is viewing another user's profile
    When the user opens report actions, selects an approved reason, and submits
    Then Flutter calls POST /v1/profiles/{handleOrDid}/reports
    And AppView resolves the account to a canonical DID before storing the report
    And the response body contains only reportId and status "accepted"
    And the user sees a transient success confirmation
```

### AT-003: Report privacy and placeholder forwarding seam

Requirement IDs: BR-002, BR-003, FR-007, FR-008, NFR-002, NFR-005, RULE-004
Acceptance Criteria: AC-003, AC-004, AC-005, AC-006, AC-022, AC-035
Priority: Must
Level: Acceptance
Automation Target: AppView store/handler tests in `appview/internal/api/report_store_test.go` and `appview/internal/api/report_test.go`

```gherkin
Feature: Private report persistence
  Scenario: Accepted reports are private and not submitted to PDS/Ozone
    Given a valid post or profile report includes optional private details
    When AppView accepts the report
    Then AppView stores the private report and safe forwarding metadata in Postgres
    And the placeholder forwarder prepares future subject/reason payload data
    But no PDS network submission occurs
    And no report record is written to a user PDS repository
    And no full prepared forwarding payload is persisted
    And the user-facing response excludes private details and forwarding payload data
```

### AT-004: Invalid report requests are rejected without persistence

Requirement IDs: FR-003, FR-004, FR-027
Acceptance Criteria: AC-009, AC-010, AC-011, AC-034, AC-041
Priority: Must
Level: Acceptance
Automation Target: AppView handler/route tests in `appview/internal/api/report_test.go` and `appview/internal/routes/routes_test.go`, Flutter dialog tests in `app/test/feed/widgets/` and `app/test/profile/`

```gherkin
Feature: Report validation
  Scenario Outline: Invalid report requests do not create report rows
    Given a signed-in user attempts to submit a report
    When the request has <invalid condition>
    Then AppView returns a standard error envelope
    And no report row is persisted

    Examples:
      | invalid condition |
      | missing auth or device ID |
      | malformed JSON |
      | malformed path identifier |
      | unsupported reasonType |
      | details longer than 1,000 characters |
      | direct self-report target |
```

### AT-005: Synthetic endpoint cannot be used outside the explicit dev gate

Requirement IDs: FR-009, NFR-001, RULE-006
Acceptance Criteria: AC-020, AC-036, AC-037
Priority: Must
Level: Acceptance
Automation Target: AppView config and route tests in `appview/internal/app/config_test.go` and `appview/internal/routes/routes_test.go`

```gherkin
Feature: Dev moderation endpoint gating
  Scenario: Synthetic moderation route is unavailable or rejected unless fully enabled
    Given AppView is running in production or dev without the explicit moderation flag
    When a caller requests POST /v1/dev/moderation/ozone-events
    Then the route is unavailable and moderation state is unchanged

  Scenario: Half-enabled dev moderation fails safely
    Given APPVIEW_ENV is dev and APPVIEW_ENABLE_DEV_MODERATION is true
    But APPVIEW_DEV_MODERATION_TOKEN is empty
    When AppView starts
    Then startup fails with a clear configuration error

  Scenario: Token and auth are required when the route is registered
    Given the route is registered in dev with a configured token
    When a request lacks auth, device ID, or a valid X-Craftsky-Dev-Moderation-Token
    Then AppView rejects the request and moderation state is unchanged
```

### AT-006: Synthetic moderation output is ingested for trusted post/account subjects

Requirement IDs: FR-009, FR-010, FR-011, FR-012, RULE-006
Acceptance Criteria: AC-019, AC-023, AC-036, AC-038
Priority: Must
Level: Acceptance
Automation Target: AppView handler/store tests in `appview/internal/api/moderation_test.go` and `appview/internal/api/moderation_store_test.go`

```gherkin
Feature: Synthetic moderation ingestion
  Scenario Outline: Dev caller ingests one trusted moderation output
    Given AppView is in dev with synthetic moderation enabled
    And the request includes valid auth, device ID, and dev moderation token
    When the caller submits one <subject type> output with <value> and <action> from a trusted source DID
    Then AppView persists the output with source DID, subject identity, value, action/negation state, timestamps, and optional internal reason

    Examples:
      | subject type | value    | action |
      | post         | hide     | apply  |
      | post         | takedown | apply  |
      | post         | warn     | apply  |
      | account      | hide     | apply  |
      | account      | takedown | apply  |
      | account      | warn     | apply  |
      | post         | hide     | negate |
      | account      | warn     | negate |
```

### AT-007: Hide/takedown outputs suppress product reads

Requirement IDs: BR-004, FR-013, FR-014, FR-015, FR-016, NFR-003, RULE-003
Acceptance Criteria: AC-012, AC-013, AC-014, AC-015, AC-016, AC-040
Priority: Must
Level: Acceptance
Automation Target: AppView store and handler tests in `appview/internal/api/timeline_store_test.go`, `post_store_test.go`, `profile_store_test.go`, `post_test.go`, and `profile_test.go`

```gherkin
Feature: Moderated read enforcement
  Scenario: Hidden posts and hidden authors are omitted from list surfaces
    Given active hide or takedown outputs exist for a post and for an author account
    When timeline, profile-post, profile-comment, and thread/comment list APIs are requested
    Then the hidden post is omitted
    And posts by the hidden author are omitted
    And pagination remains deterministic

  Scenario: Direct reads do not reveal hidden-vs-missing state
    Given a post or author account has active hide or takedown moderation
    When direct post or profile APIs are requested
    Then direct post reads return 404 post_not_found
    And direct profile reads return a not-found-style 404 response
```

### AT-008: Warn outputs keep content visible with generic UI copy

Requirement IDs: BR-005, FR-017, FR-018, FR-022, NFR-002
Acceptance Criteria: AC-017, AC-018, AC-030, AC-039
Priority: Must
Level: Acceptance
Automation Target: AppView response tests in `appview/internal/api/post_response_test.go` and `profile_response_test.go`; Flutter widget/model tests in `app/test/feed/` and `app/test/profile/`

```gherkin
Feature: Warning labels
  Scenario: Warned content remains visible with neutral localized warning copy
    Given a post, profile, or author has an active warn output
    And no active hide or takedown output applies
    When AppView returns the subject to Flutter
    Then response metadata identifies that a generic warning should be shown
    And Flutter keeps the content visible
    And Flutter shows one approved localized inline warning
    And Flutter does not display raw report details or raw internal reason text
```

### AT-009: Moderation policy precedence, negation, and expiry are deterministic

Requirement IDs: BR-004, FR-011, FR-012, FR-023, FR-024, RULE-003
Acceptance Criteria: AC-024, AC-038, AC-040
Priority: Should / Must where tied to RULE-003
Level: Acceptance
Automation Target: AppView policy unit tests in `appview/internal/api/moderation_policy_test.go` and store tests in `appview/internal/api/moderation_store_test.go`

```gherkin
Feature: Moderation policy computation
  Scenario: Hide or takedown wins over warn
    Given active warn and hide/takedown outputs apply to the same subject
    When AppView computes effective moderation policy
    Then the subject is hidden or returned as 404 instead of shown with a warning

  Scenario: Same-source negation cancels prior matching active outputs
    Given prior apply outputs exist for a subject, source DID, and value
    When a negate output with the same source DID, subject type, subject identity, and value is ingested
    Then matching prior outputs from that source are inactive
    But matching outputs from other trusted source DIDs remain active

  Scenario: Expired outputs are inactive
    Given a moderation output has expired
    When AppView computes effective moderation policy
    Then the expired output does not enforce visibility or warning behavior
```

### AT-010: Notifications omit hidden/taken-down subjects and actors

Requirement IDs: BR-004, FR-025, NFR-003
Acceptance Criteria: AC-033
Priority: Must
Level: Acceptance
Automation Target: AppView notification tests in `appview/internal/api/notification_store_test.go` and `notifications_test.go`

```gherkin
Feature: Notification moderation enforcement
  Scenario: Notifications do not leak hidden subjects or actors
    Given active hide or takedown outputs apply to a notification actor or subject post/account
    When the notification list API is requested
    Then notifications involving the hidden/taken-down actor or subject are omitted
    And no unavailable placeholder notification is returned
```

### AT-011: Duplicate reports and hidden-target reports remain eligible

Requirement IDs: FR-005, FR-006, RULE-001, RULE-005
Acceptance Criteria: AC-032, AC-043, AC-044
Priority: Must
Level: Acceptance
Automation Target: AppView report store/handler tests in `appview/internal/api/report_store_test.go` and `report_test.go`

```gherkin
Feature: Report eligibility
  Scenario: Multiple reports for the same indexed subject are accepted
    Given an indexed post or resolvable profile has already been reported
    When a different user or the same user later submits another valid report
    Then AppView accepts the report and persists a separate row

  Scenario: Already hidden targets can still be reported from stale/race flows
    Given an indexed post or resolvable profile has active hide/takedown moderation
    When another non-author user submits a valid report for that subject
    Then AppView accepts and persists the report based on target existence/resolution
```

### AT-012: Flutter report UX validates, retries, and avoids accidental duplicate submit

Requirement IDs: BR-001, FR-019, FR-020, FR-021, RULE-002
Acceptance Criteria: AC-021, AC-025, AC-026, AC-027, AC-028, AC-029, AC-045
Priority: Must
Level: Acceptance
Automation Target: Flutter widget/provider tests in `app/test/feed/widgets/`, `app/test/profile/`, and provider/repository tests under `app/test/feed/providers/` and `app/test/profile/providers/`

```gherkin
Feature: Flutter report user experience
  Scenario: Report actions are shown only for other users' content
    Given Flutter renders post cards and profiles
    When the signed-in user opens action menus
    Then report actions are present for other users' posts and visitor profiles
    And report actions are absent for the signed-in user's own posts and own profile

  Scenario: Dialog validation and retry behavior
    Given the report dialog is open
    When no reason is selected or details exceed 1,000 characters
    Then submit is blocked with accessible localized feedback
    When a valid submit fails due to API or network error
    Then the dialog preserves input and offers retry

  Scenario: In-flight submit disables repeated requests
    Given the user submits a valid report
    When the request is in flight and the user taps submit repeatedly
    Then Flutter sends only one request for that submission attempt
    When the request succeeds
    Then Flutter shows a transient success confirmation without persistently marking the subject as reported
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-002, FR-004, BR-001 | AC-001, AC-002, AC-010, AC-021 | Validate approved report reason taxonomy for post and profile reports. | Each allowed `reasonType`; unsupported values; missing reason. | Allowed values pass; missing/unsupported reason returns validation error and no persistence call. | `appview/internal/api/report_request_test.go` |
| UT-002 | FR-004, FR-027, FR-021 | AC-011, AC-027, AC-041 | Normalize optional details as plain text. | Omitted details, empty string, whitespace-only, trimmed text, 1,000 chars, 1,001 chars, reason `other` without details. | Trimmed text is stored; empty/whitespace details become omitted; 1,001 chars fails; `other` does not require details. | `appview/internal/api/report_request_test.go`, Flutter dialog validator tests |
| UT-003 | FR-004, FR-021 | AC-027, AC-034 | Self-report validation uses `invalid_report_target`. | Authenticated reporter DID equals post author DID or profile subject DID. | Validation returns standard envelope code `invalid_report_target`; no store call. | `appview/internal/api/report_test.go` |
| UT-004 | BR-002, BR-003, FR-007, FR-008, NFR-005 | AC-004, AC-005, AC-035, AC-046 | Build forwarding metadata without persisting full future payload. | Valid report with reason/details and canonical subject. | Forwarder receives enough inputs to prepare future payload; persisted forwarding row has status/schema/timestamps only; response excludes details/payload. | `appview/internal/api/report_forwarder_test.go` |
| UT-005 | FR-011, FR-012, FR-023, FR-024, RULE-003 | AC-024, AC-038, AC-040 | Compute active moderation policy with apply, negate, expiry, and precedence. | Multiple outputs for same/different source, post/account subjects, warn/hide/takedown, expired outputs. | Expired and same-source-negated outputs inactive; other trusted source outputs remain active; hide/takedown dominates warn. | `appview/internal/api/moderation_policy_test.go` |
| UT-006 | BR-004, RULE-003 | AC-012, AC-013, AC-015, AC-016, AC-040 | Treat hide and takedown as same user-visible enforcement. | Effective policy value `hide` and `takedown` for post/account. | Both map to list omission and direct not-found behavior. | `appview/internal/api/moderation_policy_test.go` |
| UT-007 | BR-005, FR-017, FR-018, FR-022, NFR-002 | AC-017, AC-018, AC-030, AC-039 | Map warn policy to response metadata without raw reason text. | Warn outputs with internal reason/details. | Metadata indicates generic warning; raw reason/details are absent from response DTOs. | `appview/internal/api/post_response_test.go`, `profile_response_test.go` |
| UT-008 | FR-026 | AC-046 | Serialize accepted report response minimally. | Accepted post/profile report result. | JSON contains only `reportId` and `status: "accepted"`; excludes details, forwarding payload, moderation state, counts. | `appview/internal/api/report_response_test.go` |
| UT-009 | FR-010, FR-011, FR-012, RULE-006 | AC-019, AC-023, AC-038 | Validate synthetic moderation request shape and trusted source DID. | Post/account subject, hide/takedown/warn, apply/negate, untrusted source, batch payload. | Valid single trusted output passes; untrusted source and batch requests fail without persistence. | `appview/internal/api/moderation_request_test.go` |
| UT-010 | FR-005, FR-006, RULE-005 | AC-007, AC-008, AC-043, AC-044 | Canonicalize report subject identities. | Post DID/rkey, malformed identifiers, profile handle/DID, hidden-but-indexed targets. | Valid subjects resolve to canonical post/account identities; malformed/unresolvable targets fail; hidden indexed targets remain eligible. | `appview/internal/api/report_target_test.go` |
| UT-011 | FR-009, NFR-001, RULE-006 | AC-020, AC-036, AC-037 | Validate dev moderation config. | Env prod/dev, enable flag true/false, token empty/non-empty, source DID allowlist. | Route config only valid in dev with flag and token; missing token errors; trusted source defaults and allowlist are enforced. | `appview/internal/app/config_test.go` |
| UT-012 | FR-019, FR-020, FR-021, BR-001 | AC-021, AC-025, AC-026, AC-027 | Flutter report action visibility and reason validation logic. | Current user DID, post author/profile DID, reason selection state, localized reason list. | Actions appear for other users only; all approved reasons are selectable; submit disabled until valid. | `app/test/feed/widgets/post_card_test.dart`, `app/test/profile/profile_page_test.dart` |
| UT-013 | FR-022, BR-005, NFR-002 | AC-030, AC-039 | Flutter warning copy renders exact generic strings only. | Post warning, profile warning, author warning metadata with raw reason present in test fixture. | UI shows approved generic localized copy and never shows raw reason. | `app/test/feed/widgets/post_card_test.dart`, `app/test/profile/profile_page_test.dart` |
| UT-014 | FR-021, RULE-002 | AC-028, AC-029, AC-045 | Flutter provider/repository submit state prevents accidental double-submit and supports retry. | Slow API future, repeated taps, failing API future, retry success. | One request while in-flight; input preserved on failure; retry sends a new request; success shows transient confirmation only. | `app/test/feed/providers/report_post_provider_test.dart`, `app/test/profile/providers/report_profile_provider_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-002, FR-007 | AC-003, AC-004, AC-042 | Migration/store persists private report rows with canonical subject snapshots. | Apply new moderation migrations in test DB; seed reporter, post/profile, CID where available. | Insert accepted post and profile reports through store. | Rows include reporter DID, subject type, canonical identities, reason, normalized details, device ID, timestamps, forwarding status, safe snapshots. | `appview/internal/api/report_store_test.go` |
| IT-002 | FR-010, FR-011, FR-012, RULE-006 | AC-019, AC-023, AC-038 | Migration/store persists synthetic moderation outputs. | Test DB with moderation output table; trusted source config. | Store post/account apply and negate outputs. | Rows contain trusted source DID, subject identity, value, action/negation, optional expiry/reason, created/indexed timestamps. | `appview/internal/api/moderation_store_test.go` |
| IT-003 | BR-003, FR-008, NFR-005 | AC-005, AC-035 | Forwarding status metadata is stored without full payload. | Accepted report with details and canonical subject. | Invoke report service/handler. | Forwarding status is `prepared_not_submitted` or equivalent; only safe metadata/schema marker persisted; full future payload absent. | `appview/internal/api/report_store_test.go` |
| IT-004 | FR-001, FR-002, FR-007, FR-026, BR-002 | AC-001, AC-002, AC-003, AC-004, AC-046 | Report endpoints accept valid requests. | Handler with authenticated DID/device ID and seeded target post/profile. | POST valid post/profile report JSON. | 200/201-style success with only reportId/status; report row persisted privately. | `appview/internal/api/report_test.go` |
| IT-005 | BR-003, RULE-004, BR-002 | AC-006, AC-022, AC-035 | Report intake does not submit to PDS or mutate PDS records. | Inject fake PDS/forwarder that records network/write calls. | Submit valid report. | PDS submission/write call count remains zero; only AppView moderation tables change. | `appview/internal/api/report_test.go` |
| IT-006 | FR-001, FR-004, FR-005 | AC-007, AC-010, AC-034, AC-043 | Post report validation and hidden-target eligibility. | Seed post author and reporter; include unknown target and hidden indexed target fixtures. | Submit malformed, unknown, self-report, and hidden-indexed post reports. | Invalid cases return standard error envelopes and no row; hidden indexed non-author report succeeds. | `appview/internal/api/report_test.go` |
| IT-007 | FR-002, FR-004, FR-006 | AC-008, AC-010, AC-034, AC-044 | Profile report validation and hidden-target eligibility. | Seed profile resolver/store; include unresolvable, malformed, self, and hidden indexed account fixtures. | Submit profile report requests. | Invalid cases return standard envelopes and no row; hidden/resolvable non-self target succeeds. | `appview/internal/api/report_test.go` |
| IT-008 | FR-003 | AC-009 | Report routes require auth and device middleware. | Register routes with normal deps. | Request report endpoints without auth and without device ID. | Existing auth/device error-envelope behavior is returned. | `appview/internal/routes/routes_test.go` |
| IT-009 | BR-004, FR-013, FR-014, NFR-003, RULE-003 | AC-012, AC-013, AC-014 | Timeline list filters hidden posts and hidden authors. | Seed timeline with visible post, hidden post, hidden author post, cursors. | Call timeline store/handler. | Hidden rows omitted; visible rows remain; cursor behavior deterministic. | `appview/internal/api/timeline_store_test.go`, `timeline_test.go` |
| IT-010 | BR-004, FR-014, FR-016, NFR-003 | AC-013, AC-016 | Profile direct read and profile-authored posts enforce account hide/takedown. | Seed profile with authored posts and active account hide/takedown. | Call `GET /v1/profiles/{handleOrDid}` and author post/comment lists. | Profile returns not-found-style 404; authored post/comment lists omit hidden-author rows. | `appview/internal/api/profile_store_test.go`, `profile_test.go`, `post_store_test.go` |
| IT-011 | BR-004, FR-013, FR-014, NFR-003 | AC-012, AC-013, AC-014 | Thread/comment read paths filter hidden posts and hidden authors. | Seed root, comments, replies, hidden comment, hidden reply author. | Call comments/replies/branch APIs. | Hidden post and hidden-author rows omitted with deterministic pagination. | `appview/internal/api/post_store_test.go`, `post_test.go` |
| IT-012 | BR-004, FR-015, RULE-003 | AC-015, AC-040 | Direct post fetch returns `404 post_not_found` for hidden/taken-down post or hidden author. | Seed direct post cases with active post/account hide/takedown and warn+hide combinations. | GET direct post endpoint. | Hidden/taken-down cases return `404 post_not_found`; warn-only remains visible. | `appview/internal/api/post_store_test.go`, `post_test.go` |
| IT-013 | FR-019, FR-021 | AC-001, AC-002, AC-028, AC-045 | Flutter API/repository sends report requests and handles success/failure. | Dio mock adapter for post/profile report endpoints; fake repositories/providers. | Submit report success and API/network failure. | Correct route/body sent; success confirmation path receives accepted response; failure preserves retryable state. | `app/test/feed/data/post_api_client_test.dart`, `app/test/profile/data/profile_api_client_test.dart` |
| IT-014 | BR-004, FR-025, NFR-003 | AC-033 | Notification list filters hidden/taken-down subjects and actors. | Seed notifications for visible subject, hidden subject, hidden actor, hidden account subject. | Call notification store/handler. | Only visible notifications returned; no placeholder leak. | `appview/internal/api/notification_store_test.go`, `notifications_test.go` |
| IT-015 | BR-005, FR-017, FR-018, FR-022 | AC-017, AC-018, AC-030, AC-039 | AppView response metadata supports Flutter generic warnings. | Seed warn-only post/account outputs and internal reasons. | Fetch post/profile/timeline response DTOs. | Content visible; metadata present; raw reason/details absent; account warn appears on profile and authored post card data. | `appview/internal/api/post_response_test.go`, `profile_response_test.go` |
| IT-016 | FR-005, FR-006, RULE-005 | AC-043, AC-044 | Report eligibility uses indexed existence/resolution, not current read visibility. | Seed indexed post/profile with active hide/takedown. | Submit non-self post/profile reports. | Reports accepted and persisted despite moderated read visibility. | `appview/internal/api/report_test.go` |
| IT-017 | FR-009, NFR-001, RULE-006 | AC-020, AC-036, AC-037 | Synthetic endpoint registration and token gate. | Route config variants: prod, dev flag off, dev flag on/token missing, dev flag on/token valid. | Start/load config and request synthetic endpoint. | Route unavailable except fully enabled dev; missing token fails startup; bad/missing token rejects without mutation. | `appview/internal/app/config_test.go`, `appview/internal/routes/routes_test.go` |
| IT-018 | FR-011, FR-012, FR-023, FR-024 | AC-024, AC-038 | Active-policy computation over stored outputs handles negate and expiry. | Store apply, negate, expired, and cross-source outputs. | Query active policy for subject. | Same-source prior outputs inactive after negate; expired inactive; cross-source active output still enforced. | `appview/internal/api/moderation_store_test.go` |
| IT-019 | NFR-004 | AC-031 | Read-path moderation lookups avoid per-row remote calls and N+1 query shape. | Instrument store/query test with multiple posts/profiles and local moderation rows. | Execute paginated list reads. | Enforcement uses local indexed state and bounded query pattern; no per-row remote calls. | `appview/internal/api/*_store_test.go` |
| IT-020 | RULE-001 | AC-032 | Multiple report rows for the same subject are allowed. | Seed one existing report for a post/profile. | Submit another valid report from different user and same user later. | Request is not rejected solely due to existing row; separate report rows are persisted. | `appview/internal/api/report_store_test.go`, `report_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Unmoderated timeline/profile/thread list reads continue returning existing visible posts with existing cursor behavior. | FR-013, FR-014, BR-004 | Seed no moderation outputs and verify timeline, author posts, comments, and replies match pre-feature expectations. |
| REG-002 | Existing report-free post/profile APIs maintain camelCase JSON and tolerate omitted `moderation` metadata. | FR-017, FR-018 | Decode responses with no `moderation` field in AppView and Flutter model tests. |
| REG-003 | Warn-only subjects remain visible and do not become soft-hidden/interstitial-gated. | BR-005, FR-017, FR-018, FR-022 | Verify warned post/profile content is rendered inline with one warning banner and normal actions/content remain present. |
| REG-004 | Direct missing post/profile behavior remains not-found and hidden subjects use the same not-found style. | FR-015, FR-016 | Compare unknown target and hidden target response status/error codes. |
| REG-005 | Notification list retains existing camelCase response shape and viewer handling when no moderation applies. | FR-025 | Existing notification tests pass with moderation absent; visible notifications still include actor/subject data. |
| REG-006 | Existing authenticated/device middleware behavior remains consistent for product routes. | FR-003, FR-009, NFR-001 | Report and synthetic routes use existing error-envelope behavior for missing auth/device while existing routes still pass current route tests. |
| REG-007 | No PDS write/delete/report submission occurs for moderation-report MVP. | BR-002, BR-003, RULE-004 | Fake PDS call counters remain zero during report and synthetic-output flows. |
| REG-008 | Flutter post/profile pages continue showing existing action menus, follow/share/edit controls, and engagement counts for unmoderated content. | FR-019, FR-020, FR-021 | Existing widget tests continue to pass, with added report actions only where expected. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Baseline users | `did:plc:alice` signed-in reporter, `did:plc:bob` post/profile subject, `did:plc:carol` alternate reporter, `did:plc:labeler` trusted source. | AT-001, AT-002, AT-006, AT-011, IT-001, IT-020 |
| TD-002 | Post subject identity | `at://did:plc:bob/social.craftsky.feed.post/3lf2abc`, collection `social.craftsky.feed.post`, rkey `3lf2abc`, CID `bafy-post-v1`. | AT-001, IT-001, IT-006, IT-012 |
| TD-003 | Profile subject identity | Submitted handle `bob.craftsky.social`, canonical DID `did:plc:bob`, optional handle snapshot. | AT-002, IT-001, IT-007, IT-010 |
| TD-004 | Approved reasons | `harassment`, `hate`, `spam`, `misleading`, `suspected_ai_generated`, `adult_or_graphic`, `impersonation`, `off_topic`, `intellectual_property`, `other`. | AT-001, AT-002, AT-012, UT-001 |
| TD-005 | Invalid report inputs | Malformed JSON, unsupported `reasonType: "not_allowed"`, details length 1,001, malformed DID/rkey, self-report target. | AT-004, IT-006, IT-007, UT-001, UT-002, UT-003 |
| TD-006 | Moderation values | `hide`, `takedown`, `warn` with actions `apply` and `negate`; one output per request. | AT-006, AT-009, IT-002, IT-018 |
| TD-007 | Warning metadata | Warned post, warned profile/account, account-warned author post with internal reason `raw unsafe reason fixture`. | AT-008, IT-015, UT-007, UT-013 |
| TD-008 | Hidden read fixtures | Visible post, post-level hidden post, account-level hidden author with authored post/comment/reply, warning plus hide precedence case. | AT-007, AT-009, IT-009, IT-010, IT-011, IT-012 |
| TD-009 | Notification fixtures | Like/reply/follow notifications with visible actor, hidden actor, hidden subject post, hidden account subject. | AT-010, IT-014, REG-005 |
| TD-010 | Flutter UI states | Report dialog no reason, valid reason with optional details, 1,001-char details, in-flight submit, API failure, retry success, success confirmation. | AT-012, UT-012, UT-014, IT-013 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | BR-001, FR-019, FR-021 | Local end-to-end report UX smoke. | Run app against local AppView, sign in, report another user's post/profile, exercise success and retry flows. | Report actions, validation, loading, retry, and success feedback are understandable and non-persistent. |
| MAN-002 | BR-005, FR-022, NFR-002 | Localized warning copy and accessibility review. | Render warned post, warned profile, and warned author post card. Inspect copy, semantics, color contrast, and screen-reader labels. | Exact generic copy is visible, neutral, localized, accessible, and raw reason text is absent. |
| MAN-003 | BR-002, RULE-004, NFR-002 | Privacy/log spot check. | Exercise report and synthetic flows with private details/internal reasons; inspect API response bodies and development logs. | Responses and normal logs do not expose private details, raw internal reasons, dev moderation token values, or full forwarding payloads. |
| MAN-004 | NFR-004 | Query/performance sanity review. | Run focused read-path tests with moderate fixture volume or inspect query plans/logged SQL if available. | Moderation enforcement uses local indexed state and does not introduce obvious per-row remote calls or unbounded query loops. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Full live Ozone and PDS report-submission behavior is not tested. | BR-003, FR-008 | Live Ozone operation and `com.atproto.moderation.createReport` submission are explicit non-goals for this MVP. | Add separate acceptance tests when live Ozone/PDS submission enters scope. |
| GAP-002 | Search moderation coverage is absent. | BR-004, NFR-003 | Search is not implemented and is explicitly out of scope. | Add search read-path tests if search ships concurrently or later. |
| GAP-003 | Rate limiting and report-spam abuse controls are not covered. | RULE-001 | MVP explicitly allows duplicate/multiple reports and does not require rate limiting. | Define abuse/rate-limit requirements in a later safety iteration. |
| GAP-004 | Automated accessibility coverage is partial. | FR-021, FR-022 | Widget tests can assert labels/visibility but cannot fully validate human UX, contrast, and screen-reader quality. | Keep MAN-002 as required review before implementation acceptance. |

## 10. Out Of Scope

- Live Ozone deployment or live Ozone WebSocket label ingestion.
- PDS/Ozone report submission via `com.atproto.moderation.createReport`.
- Lexicon additions or changes.
- Moderator dashboard, appeals, legal workflow, email workflow, blocks, mutes, and Ozone UI.
- Deleting or mutating user PDS records as a moderation side effect.
- Search moderation, because search is not implemented.
- Persisted cross-refresh/cross-device “already reported” UI state.
- Report-spam rate limiting.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this agent: `docs/changes/2026-05-30-moderation-flow-mvp/`
- Risk level: High
- Review recommendation: Required before implementation continues
- Recommended first failing test for implementation: `IT-001` — migration/store persists private post/profile report rows with canonical subject snapshots and safe forwarding metadata.
- Suggested test order for implementation:
  1. `IT-001`, `UT-002`, `UT-004` for report storage, detail normalization, and forwarding metadata privacy.
  2. `IT-004`, `IT-006`, `IT-007`, `IT-008`, `UT-001`, `UT-003`, `UT-008`, `UT-010` for AppView report endpoints and validation.
  3. `UT-011`, `IT-017`, `IT-002`, `UT-009` for synthetic endpoint config, route gating, and persistence.
  4. `UT-005`, `UT-006`, `IT-018` for moderation active-policy semantics.
  5. `IT-009`, `IT-010`, `IT-011`, `IT-012`, `IT-014`, `IT-019` for AppView read-path enforcement.
  6. `IT-015`, `UT-007` for response metadata and raw-reason privacy.
  7. `IT-013`, `UT-012`, `UT-013`, `UT-014`, `AT-012` for Flutter API/repository/provider/widget behavior.
  8. `REG-001` through `REG-008` before final acceptance.
- Commands discovered:
  - AppView focused tests from `appview/`: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app`
  - Full AppView test recipe from repo root: `just test`
  - Flutter focused tests from `app/`: `flutter test <paths>`
  - Likely Flutter moderation-focused command after tests exist: `flutter test test/feed test/profile test/notifications`
- Blocking gaps: None for test design. Implementation must still receive required high-risk document review before coding proceeds.
