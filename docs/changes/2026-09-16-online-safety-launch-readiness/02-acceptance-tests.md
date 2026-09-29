# Acceptance Test Specification: Pre-Launch Online Safety Readiness

## 1. Test Strategy

Risk level: High.

The release needs evidence at five levels:

- Unit tests cover scan state, visibility, retries, policy-version cache keys,
  taxonomy mappings, redaction, permissions, retention, deadlines, and age rules.
- Go integration tests use PostgreSQL and MinIO to cover Tap ingestion, worker
  restart/replay, every read surface, incidents, holds, deletion, moderation, and
  restricted storage. Provider tests use only a protocol-neutral deterministic
  scanner stub until IWF supplies an approved contract.
- Flutter widget tests cover reporting, 16+ onboarding, safety guidance, owner
  notices, eligibility restrictions, disabled video affordances, and accessibility.
- Acceptance checks cover launch gates, policy publication, public-site consent,
  production configuration, and end-to-end operational workflows.
- Manual exercises are limited to provider/cloud/mailbox controls, specialist
  judgments, accessibility technologies, and human operational procedures that
  cannot be established by repository tests alone.

Production acceptance is blocked until every gap in Section 9 is closed. Bounded
coding planning may cover provider-neutral infrastructure and the deterministic
stub, but must exclude the production IWF adapter. A stub test proves CraftSky
behavior around an adapter; it does not prove compatibility with or approval by IWF
Image Intercept.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001 | AT-001, MAN-001 | Acceptance / Manual | Partial |
| BR-002 | AC-002, AC-003 | AT-002, IT-003, IT-007 | Acceptance / Integration | Yes |
| BR-003 | AC-013, AC-014, AC-015 | AT-004, AT-005, AT-006 | Acceptance | Partial |
| BR-004 | AC-010, AC-011, AC-012 | AT-007, AT-008, AT-009, MAN-002 | Acceptance / Manual | Partial |
| BR-005 | AC-019, AC-020 | AT-010, AT-011, MAN-010 | Acceptance / Manual | Partial |
| BR-006 | AC-026 | AT-012, MAN-014 | Acceptance / Manual | Partial |
| FR-001 | AC-002, AC-004, AC-038 | AT-002, IT-002, IT-004 | Acceptance / Integration | Yes |
| FR-002 | AC-004, AC-005 | UT-003, IT-005 | Unit / Integration | Yes |
| FR-003 | AC-003, AC-006 | UT-001, IT-003, IT-006 | Unit / Integration | Yes |
| FR-004 | AC-002, AC-007 | UT-002, IT-003 | Unit / Integration | Yes |
| FR-005 | AC-006, AC-008 | AT-008, IT-006 | Acceptance / Integration | Yes |
| FR-006 | AC-003, AC-009 | UT-004, IT-007 | Unit / Integration | Yes |
| FR-007 | AC-027, AC-028 | UT-014, UT-015, IT-008 | Unit / Integration | Yes |
| FR-008 | AC-028 | UT-014, IT-008 | Unit / Integration | Yes |
| FR-009 | AC-008, AC-043 | AT-008, IT-009 | Acceptance / Integration | Yes |
| FR-010 | AC-010, AC-021 | AT-007, AT-013, IT-010, IT-027 | Acceptance / Integration | Partial |
| FR-011 | AC-011, AC-012, AC-022, AC-042 | AT-008, AT-009, IT-011, IT-012, IT-022 | Acceptance / Integration | Partial |
| FR-012 | AC-012 | AT-009, IT-012 | Acceptance / Integration | Yes |
| FR-013 | AC-010, AC-011 | AT-007, IT-010, IT-011 | Acceptance / Integration | Partial |
| FR-014 | AC-013, AC-046 | AT-004, UT-006 | Acceptance / Unit | Yes |
| FR-015 | AC-016 | UT-006, UT-007 | Unit | Yes |
| FR-016 | AC-016, AC-017 | UT-007, IT-014 | Unit / Integration | Yes |
| FR-017 | AC-013 | AT-004 | Acceptance | Yes |
| FR-018 | AC-014, AC-015 | AT-005, IT-015 | Acceptance / Integration | Partial |
| FR-019 | AC-014 | UT-018, IT-015 | Unit / Integration | Yes |
| FR-020 | AC-015, AC-018 | AT-006, IT-016 | Acceptance / Integration | Yes |
| FR-021 | AC-021 | AT-013, IT-027, MAN-004 | Acceptance / Manual | Partial |
| FR-022 | AC-023, AC-048 | AT-004, AT-014, IT-028, MAN-005 | Acceptance / Manual | Partial |
| FR-023 | AC-019 | AT-010, IT-017 | Acceptance / Integration | Yes |
| FR-024 | AC-020, AC-049 | AT-011, UT-012, IT-018 | Acceptance / Unit / Integration | Yes |
| FR-025 | AC-007, AC-024 | AT-002, IT-019 | Acceptance / Integration | Yes |
| FR-026 | AC-005, AC-024 | IT-005, IT-019 | Integration | Yes |
| FR-027 | AC-025 | AT-015, IT-020, REG-007 | Acceptance / Integration / Regression | Yes |
| FR-028 | AC-029 | AT-016, IT-021 | Acceptance / Integration | Yes |
| FR-029 | AC-022 | UT-009, IT-022 | Unit / Integration | Partial |
| FR-030 | AC-030 | AT-017, IT-023 | Acceptance / Integration | Yes |
| FR-031 | AC-031 | AT-018, MAN-007, MAN-008 | Acceptance / Manual | Partial |
| FR-032 | AC-039 | AT-003, UT-017, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-033 | AC-040 | REG-001 | Regression | Yes |
| FR-034 | AC-041 | UT-016, IT-005 | Unit / Integration | Yes |
| FR-035 | AC-044 | AT-008, IT-011 | Acceptance / Integration | Yes |
| FR-036 | AC-045 | UT-008, IT-009 | Unit / Integration | Yes |
| FR-037 | AC-047 | AT-005, IT-024, MAN-006 | Acceptance / Manual | Partial |
| FR-038 | AC-046 | AT-004, UT-023 | Acceptance / Unit | Yes |
| NFR-001 | AC-004, AC-005, AC-009 | IT-005, IT-007 | Integration | Yes |
| NFR-002 | AC-011, AC-032 | IT-011, IT-025 | Integration | Yes |
| NFR-003 | AC-033 | AT-019, MAN-012 | Acceptance / Manual | Partial |
| NFR-004 | AC-009, AC-029 | IT-007, IT-021 | Integration | Yes |
| NFR-005 | AC-027 | UT-015, MAN-009 | Unit / Manual | Partial |
| NFR-006 | AC-018, AC-021, AC-023 | IT-016, IT-027, IT-028 | Integration | Yes |
| NFR-007 | AC-050 | IT-011, MAN-003 | Integration / Manual | Partial |
| RULE-001 | AC-006, AC-017 | UT-007, IT-006, IT-014 | Unit / Integration | Yes |
| RULE-002 | AC-024, AC-034 | IT-019, REG-002 | Integration / Regression | Yes |
| RULE-003 | AC-032, AC-034 | IT-025, REG-003 | Integration / Regression | Yes |
| RULE-004 | AC-027, AC-028 | UT-014, UT-015, IT-008 | Unit / Integration | Yes |
| RULE-005 | AC-002, AC-035 | IT-002, MAN-011 | Integration / Manual | Partial |
| RULE-006 | AC-014, AC-015 | AT-005, AT-006, MAN-006 | Acceptance / Manual | Partial |
| RULE-007 | AC-029, AC-036 | AT-016, MAN-013 | Acceptance / Manual | Partial |
| RULE-008 | AC-037 | REG-004 | Regression | Yes |
| RULE-009 | AC-050, AC-051 | AT-008, IT-011, IT-013 | Acceptance / Integration | Partial |
| RULE-010 | AC-049 | AT-010, AT-011, UT-012, REG-008 | Acceptance / Unit / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Block Launch With Any Open P0
Requirement IDs: BR-001
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `scripts/online-safety-readiness` and release workflow

```gherkin
Scenario: An unresolved P0 blocks release
  Given the compliance register contains a P0 without approved linked evidence
  When the public-release readiness check runs
  Then the check fails and identifies the unresolved control without exposing restricted data
```

### AT-002: Keep Image Records Hidden Until All Blobs Clear
Requirement IDs: BR-002, FR-001, FR-004, FR-025, RULE-005
Acceptance Criteria: AC-002, AC-003, AC-007, AC-038
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/index/image_scan_acceptance_test.go`

```gherkin
Scenario: One non-clear image hides the complete parent
  Given Tap delivers a post or business event with multiple rendered images
  And at least one required blob is pending, unavailable, errored, or matched
  When any public AppView surface requests the record
  Then the complete record and every unsafe derivative are absent
  And the source PDS record is not deleted
```

### AT-003: Preserve A Safe Profile Image
Requirement IDs: FR-032
Acceptance Criteria: AC-039
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/index/bluesky_profile_test.go`

```gherkin
Scenario: A replacement profile image is not clear
  Given a profile has a previously cleared avatar or banner
  When its replacement is pending, matched, unavailable, or errored
  Then CraftSky continues to render the previously cleared image
  And a profile with no cleared image renders a neutral placeholder
```

### AT-004: Route A Signed-In Safety Report
Requirement IDs: BR-003, FR-014, FR-017, FR-022, FR-038
Acceptance Criteria: AC-013, AC-046, AC-048
Priority: Must
Level: Acceptance
Automation Target: `app/test/moderation/widgets/report_flow_test.dart`

```gherkin
Scenario: The reporter chooses the correct specialist route
  Given a signed-in member opens the report flow
  When they select a broad group and then a precise allegation
  Then the approved reason is submitted with user-safe guidance
  And child-safety guidance says not to download, attach, forward, or redistribute material
  And immediate-danger guidance directs them to local emergency services first
  And intellectual-property concerns use the dedicated email route without a generic report
```

### AT-005: Safely Intake An External Email
Requirement IDs: BR-003, FR-018, FR-019, FR-037, RULE-006
Acceptance Criteria: AC-014, AC-015, AC-047
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/safetyintake/email_acceptance_test.go`

```gherkin
Scenario: A non-user emails a safety concern
  Given an external email identifies a canonical CraftSky URL or AT URI
  When authorized intake accepts the message
  Then receipt, reference, urgency, ownership, provenance, and minimized contact details are recorded
  And no reporter DID is invented
  And any image or video attachment is rejected or quarantined outside ordinary case storage without preview or forwarding
```

### AT-006: Resolve An External Appeal Without Dropping Enforcement
Requirement IDs: BR-003, FR-018, FR-020, RULE-006
Acceptance Criteria: AC-015, AC-018
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/moderation/appeal_service_integration_test.go`

```gherkin
Scenario: An owner appeals an action by email
  Given correspondence is bound to an owner-visible decision and the appellant is proportionately verified
  When the appeal is reviewed and resolved
  Then enforcement remains active until an authorized outcome changes it
  And acknowledgement, correspondence, reconsideration, outcome, reversals, and actors are append-only and auditable
```

### AT-007: Exercise The CSEA Reporting Workflow
Requirement IDs: BR-004, FR-010, FR-013
Acceptance Criteria: AC-010, AC-011
Priority: Must
Level: Acceptance / Tabletop
Automation Target: operational evidence pack plus `appview/internal/safetyincident/csea_workflow_test.go`

```gherkin
Scenario: A synthetic detection requires authority reporting
  Given an approved synthetic CSEA incident and authorized safety administrator
  When the CSEA tabletop runs from detection through closure
  Then priority, assignment, initial report, supplement, duplicate handling, authority reference, retention, follow-up, and resolution are recorded
```

### AT-008: Review A Match Without Automatic Guilt Or Byte Exposure
Requirement IDs: FR-005, FR-009, FR-011, FR-035, FR-036, RULE-001, RULE-009
Acceptance Criteria: AC-006, AC-008, AC-042, AC-043, AC-044, AC-045, AC-051
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/safetyincident/detection_acceptance_test.go`

```gherkin
Scenario: A scanner result matches
  Given the deterministic scanner returns match for an indexed blob
  When the detection is processed
  Then the subject remains hidden and a restricted incident is created without a report or sanction
  And an ordinary moderator reviews safe metadata without rendered bytes
  And only an authorized human confirmation can create a systemDetected case
  And any owner notice says Child safety violation without provider, evidence, or reporting details
```

### AT-009: Apply Deletion And Holds Without Indefinite Retention
Requirement IDs: BR-004, FR-011, FR-012, FR-029
Acceptance Criteria: AC-012, AC-022
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/accountdeletion/legal_hold_acceptance_test.go`

```gherkin
Scenario: Account deletion overlaps restricted evidence
  Given unrelated private data and evidence with no hold, an active scoped hold, or an expired hold
  When account cleanup and retention jobs run
  Then unrelated and unheld data are deleted
  And only active in-scope held evidence remains
  And expired held evidence is automatically deleted with non-sensitive completion evidence
```

### AT-010: Record Explicit 16+ Acceptance
Requirement IDs: BR-005, FR-023, RULE-010
Acceptance Criteria: AC-019
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_age_declaration_test.dart`

```gherkin
Scenario: A person completes registration
  Given the applicable minimum age is 16
  When onboarding completes
  Then the person explicitly declares they meet the threshold
  And the server records the accepted policy version
  And no date of birth or age band is requested
```

### AT-011: Apply A Reversible Under-16 Eligibility Restriction
Requirement IDs: BR-005, FR-024, RULE-010
Acceptance Criteria: AC-020, AC-049
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/moderation/age_eligibility_acceptance_test.go`

```gherkin
Scenario: Specific credible evidence indicates a member may be under 16
  Given an authorized moderator has evidence other than appearance, interests, writing, location, or report volume
  When the moderator completes eligibility review
  Then any restriction is reversible and appealable
  And sign-in, privacy, removal, deletion, block, and mute controls remain available
  And the workflow uses no facial-age estimation, age band, or full date of birth
```

### AT-012: Publish Only Proven Policies
Requirement IDs: BR-001, BR-006
Acceptance Criteria: AC-001, AC-026
Priority: Must
Level: Acceptance
Automation Target: policy publication release gate

```gherkin
Scenario: Public policies are approved for publication
  Given all five policy routes are staged with version and effective metadata
  When atomic publication is requested
  Then every present-tense control has approved implementation and test evidence
  And publication fails if any linked P0 or claimed control remains unproved
```

### AT-013: Handle A Qualifying Intimate-Image Complaint
Requirement IDs: FR-010, FR-021, NFR-006
Acceptance Criteria: AC-021
Priority: Must
Level: Acceptance / Tabletop
Automation Target: `appview/internal/safetyincident/intimate_image_test.go` plus exercise evidence

```gherkin
Scenario: An eligible complainant reports an intimate image
  Given the approved response target and standing rules are configured
  When the complaint is adjudicated
  Then receipt, standing, declarations, deadline, judgment, same-image search, substantially-same search, action, exception, outcome, and actors are recorded
```

### AT-014: Handle Immediate Danger And Authority Requests
Requirement IDs: FR-022, NFR-006
Acceptance Criteria: AC-023, AC-048
Priority: Must
Level: Acceptance / Tabletop
Automation Target: `appview/internal/safetyincident/authority_workflow_test.go` plus exercise evidence

```gherkin
Scenario: A credible threat and later authority request are received
  Given user guidance identifies local emergency services as the first route
  When internal escalation and authority handling proceed
  Then preservation, independent verification, lawful-process approval, minimized disclosure, and post-incident review are attributable
  And credentials and unrelated data are never included
```

### AT-015: Disable Production Video
Requirement IDs: FR-027
Acceptance Criteria: AC-025
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/video_launch_gate_test.go` and Flutter video eligibility tests

```gherkin
Scenario: A production client attempts to publish video
  Given CraftSky is configured for public launch
  When the client requests video authorization or publication
  Then the server refuses the operation
  And the production UI offers no video creation path
```

### AT-016: Prioritize Urgent Operational Work
Requirement IDs: FR-028, NFR-004, RULE-007
Acceptance Criteria: AC-029, AC-036
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/safety_admin_test.go`

```gherkin
Scenario: Urgent and overdue work exists while the founder is unavailable
  Given queues contain untriaged, urgent, detected, appealed, dead-lettered, and expiring items
  When an authorized named helper views assigned work
  Then priority, age, deadline, owner or cover, and alert state are visible without restricted content
  And actions outside the helper's assigned permission are denied
```

### AT-017: Withhold PostHog Before Consent
Requirement IDs: FR-030
Acceptance Criteria: AC-030
Priority: Must
Level: Acceptance
Automation Target: public-site browser test suite

```gherkin
Scenario: A visitor has not consented to analytics
  Given no approved alternative PECR route applies
  When a public page loads
  Then no PostHog script, request, cookie, or identifier is created
```

### AT-018: Exercise Privacy Operations
Requirement IDs: FR-031
Acceptance Criteria: AC-031
Priority: Must
Level: Acceptance / Tabletop
Automation Target: operational evidence pack

```gherkin
Scenario: Synthetic rights, privacy complaint, and breach cases are opened
  Given approved procedures and applicable clocks exist
  When each exercise runs to completion
  Then identity checks, searches, decisions, escalation, notifications, and outcomes are recorded
```

### AT-019: Meet The Safety Accessibility Gate
Requirement IDs: NFR-003
Acceptance Criteria: AC-033
Priority: Must
Level: Acceptance
Automation Target: Flutter safety widget suites plus manual assistive-technology pass

```gherkin
Scenario: A member uses safety controls with accessibility settings
  Given keyboard navigation, screen reader, large text, and validation errors
  When report, appeal, notice, and eligibility flows are completed
  Then all controls meet the approved WCAG 2.2 AA gate and permit error recovery
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-003 | AC-003, AC-006 | Enumerate valid scan states and terminal/non-terminal behavior. | `pending`, `clear`, `match`, `unavailable`, `error`, unknown | Only `clear` is display-eligible; unknown is rejected/fail-closed. | `appview/internal/imagesafety/state_test.go` |
| UT-002 | FR-004 | AC-002, AC-007 | Calculate parent visibility across all blob states. | Empty, one clear, mixed, duplicate blobs | Image parent is visible only when every required blob is clear. | `appview/internal/imagesafety/visibility_test.go` |
| UT-003 | FR-002 | AC-004, AC-005 | Build and validate result cache keys. | Blob CID, provider, policy/corpus version | Same key reuses a valid result; changed CID/version cannot reuse it silently. | `appview/internal/imagesafety/cache_test.go` |
| UT-004 | FR-006 | AC-003, AC-009 | Calculate bounded exponential retry and exhaustion. | Attempt number, error class, clock | Delays are bounded; exhaustion dead-letters without clearing content. | `appview/internal/imagesafety/retry_test.go` |
| UT-005 | FR-005, FR-009 | AC-006, AC-008 | Deduplicate incident creation while retaining subject links. | Replayed match, shared blob, multiple records | One stable detection is reused with required subject-specific links and no report. | `appview/internal/safetyincident/detection_test.go` |
| UT-006 | FR-014, FR-015 | AC-013, AC-016, AC-046 | Map groups, allegations, internal reasons, legal classes, and safe text separately. | Every approved category | Mapping is exhaustive; user input cannot directly assert a legal judgment. | `appview/internal/moderation/taxonomy_test.go` and Flutter model tests |
| UT-007 | FR-015, FR-016, RULE-001 | AC-016, AC-017 | Require an authorized human decision before effects. | Reports, match, human decisions, severity | Signals alone apply no effect; supported decisions expose proportionate effects. | `appview/internal/moderation/policy_test.go` |
| UT-008 | FR-036 | AC-045 | Render the confirmed-match owner notice. | Decision/action/appeal data plus restricted metadata | Text uses `Child safety violation` and omits provider, match, evidence, and report details. | `appview/internal/moderation/presentation_test.go` |
| UT-009 | FR-029 | AC-022 | Calculate retention and hold precedence. | Data class, created time, hold scope/expiry | Correct deletion time is returned; active scoped hold pauses only covered deletion. | `appview/internal/retention/policy_test.go` |
| UT-010 | RULE-009 | AC-050, AC-051 | Authorize restricted actions by role and assignment. | Moderator, helper, safety administrator | Only safety administrators can access bytes/holds/reports/disclosures; safe metadata follows role. | `appview/internal/safetyincident/authorization_test.go` |
| UT-011 | FR-010, FR-021, FR-022 | AC-010, AC-021, AC-023 | Calculate incident deadlines and escalation. | Incident class, receipt, approved target | Applicable deadline and alert points are deterministic and no unapproved target is invented. | `appview/internal/safetyincident/deadline_test.go` |
| UT-012 | FR-024, RULE-010 | AC-020, AC-049 | Validate under-age evidence and retained capabilities. | Allowed evidence, prohibited inference, restriction | Prohibited signals cannot support action; approved controls remain accessible. | `appview/internal/moderation/age_eligibility_test.go` |
| UT-013 | NFR-002, RULE-003 | AC-032, AC-034 | Redact safety objects for logs, metrics, errors, push, and owner APIs. | Fully populated restricted records | Output contains only allowlisted non-sensitive fields. | `appview/internal/safetyincident/redaction_test.go` |
| UT-014 | FR-007, FR-008, RULE-004 | AC-028 | Validate scanner mode and production readiness. | Environment, adapter identity, credentials | Production rejects stub/missing/invalid IWF configuration and cannot mark clear. | `appview/internal/imagesafety/config_test.go` |
| UT-015 | FR-007, NFR-005, RULE-004 | AC-027 | Enforce a protocol-neutral scanner interface and deterministic fixture stub. | Fixture image IDs/outcomes | Stub is deterministic; interface contains no speculative IWF wire fields and sends only required input. | `appview/internal/imagesafety/scanner_test.go` |
| UT-016 | FR-034 | AC-041 | Decide whether a clear blob needs rescanning. | Clear result, time, policy, targeted request | Time alone never schedules a rescan; an authorized request or approved policy change can. | `appview/internal/imagesafety/rescan_test.go` |
| UT-017 | FR-032 | AC-039 | Select the profile image presented to readers. | Candidate state, previous clear image | Non-clear candidate falls back to last clear image or neutral placeholder. | `appview/internal/index/profile_image_policy_test.go` |
| UT-018 | FR-018, FR-019 | AC-014 | Normalize external-email provenance without member identity. | Email metadata, optional sender details | External source and minimized contact reference are retained; `reporter_did` remains null. | `appview/internal/safetyintake/email_test.go` |
| UT-019 | FR-011, FR-012 | AC-012, AC-022 | Validate hold scope, basis, approver, and expiry. | Complete/incomplete hold requests | Invalid or indefinite holds fail; valid holds cover only explicit evidence. | `appview/internal/safetyincident/hold_test.go` |
| UT-020 | FR-022, FR-028 | AC-023, AC-029 | Rank urgent work without using report volume as guilt. | Threats, deadlines, report counts | Credible urgency/deadline controls priority; count does not establish violation. | `appview/internal/safetyincident/priority_test.go` |
| UT-021 | BR-006 | AC-026 | Validate public-policy claims against control evidence. | Claims, versions, evidence references | Missing/stale evidence blocks publication; approved metadata passes. | policy release-gate tests |
| UT-022 | FR-027 | AC-025 | Evaluate production video feature gates. | Environment and feature flags | Production always returns disabled independent of dormant client code. | `appview/internal/app/video_gate_test.go` |
| UT-023 | FR-038 | AC-046 | Route intellectual-property selections. | Copyright/trade-mark reason | Dedicated email instructions are returned and no generic report payload is built. | `app/test/moderation/models/report_reason_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-002, FR-009, FR-010, FR-011, FR-018, FR-023 | AC-004, AC-008, AC-010, AC-011, AC-014, AC-019 | Apply private scan/safety/intake/age migrations up, down, and up. | Disposable PostgreSQL | Run migration cycle and inspect constraints/indexes | Schema is reversible, referentially sound, and private; generated queries compile. | `appview/internal/db/online_safety_migration_test.go` |
| IT-002 | FR-001, RULE-005 | AC-002, AC-038 | Extract scan dependencies from every in-scope Tap record, including third-party writes. | Post, business event, profile, external embed fixtures | Dispatch Tap events | Every rendered blob CID is queued at index time exactly once. | Existing `appview/internal/index/*_test.go` suites |
| IT-003 | BR-002, FR-003, FR-004 | AC-002, AC-003, AC-007 | Gate multi-image parent visibility transactionally. | Mixed scanner fixtures in PostgreSQL | Query before and after final clear | Parent remains absent until all clear; non-clear transition removes eligibility. | `appview/internal/index/image_scan_acceptance_test.go` |
| IT-004 | FR-001, FR-032 | AC-038, AC-039 | Gate avatars and banners independently with safe fallback. | Profile image history and scan results | Index profile updates | Only last-cleared or placeholder URLs appear in profile APIs. | `appview/internal/index/bluesky_profile_test.go` |
| IT-005 | FR-002, FR-026, FR-034, NFR-001 | AC-004, AC-005, AC-041 | Replay, reconnect, edit, delete, and targeted rescan preserve correct state. | Tap events with stable/changed record and blob CIDs | Replay events and restart workers | Stable valid result is reused; changed blob becomes pending; delete cannot resurrect; no periodic rescan occurs. | `appview/internal/tap/replay_test.go` plus image-safety integration suite |
| IT-006 | FR-003, FR-005, FR-035, RULE-001 | AC-006, AC-008, AC-044 | Process a match without report, decision, effect, or rendered bytes. | Match fixture and ordinary moderator | Run worker and inspect admin response | Restricted incident exists; content hidden; sanctions absent; response is metadata-only. | `appview/internal/safetyincident/detection_acceptance_test.go` |
| IT-007 | FR-006, NFR-001, NFR-004 | AC-003, AC-009 | Survive timeout, retry, restart, exhaustion, and manual retry. | Fake clock, failing scanner, restarted workers | Advance attempts through dead letter and manual retry | No duplicate attempts/effects; visibility stays closed; metrics/alert/dead letter update; authorized retry resumes. | `appview/internal/imagesafety/worker_acceptance_test.go` |
| IT-008 | FR-007, FR-008, RULE-004 | AC-027, AC-028 | Enforce scanner production startup/readiness. | Stub, missing credentials, invalid credentials, approved adapter test double | Start service and call readiness | Unsafe configurations fail readiness and produce no clear results. | `appview/internal/app/imagesafety_config_test.go` |
| IT-009 | FR-009, FR-036 | AC-043, AC-045 | Create `systemDetected` case after human confirmation and use existing enforcement. | Restricted incident and authorized moderator | Confirm, decide, notify, appeal | Safe incident reference links the case; no report exists; effects/history/appeal work; notice is safe. | `appview/internal/moderation/adjudication_service_integration_test.go` |
| IT-010 | FR-010, FR-013 | AC-010 | Persist CSEA lifecycle and duplicate/supplement handling. | Synthetic incident | Record assignment, report, supplement, reference, duplicate, resolution | Append-only chronology is complete and deadline state remains accurate. | `appview/internal/safetyincident/csea_workflow_test.go` |
| IT-011 | FR-011, FR-013, FR-035, NFR-002, NFR-007, RULE-009 | AC-011, AC-042, AC-044, AC-050, AC-051 | Store/access/export required evidence through restricted boundary. | Dedicated MinIO test bucket, KMS/IAM abstractions, role fixtures | Save metadata, conditionally copy bytes, access and export | Default incident has no byte copy; required object is private; access is reasoned/audited; export minimized; unauthorized access denied. | `appview/internal/safetyincident/evidence_store_integration_test.go` |
| IT-012 | FR-011, FR-012 | AC-012 | Make account deletion hold-aware. | Accounts with no, active, expired, and out-of-scope holds | Run private cleanup and expiry | Only valid in-scope held evidence survives; all other eligible private data is deleted. | `appview/internal/accountdeletion/legal_hold_acceptance_test.go` |
| IT-013 | RULE-009 | AC-050, AC-051 | Enforce admin authorization on every restricted route. | Moderator/helper/admin sessions | Call byte, hold, NCA, disclosure, safe-metadata routes | Least privilege is consistent and every permitted restricted action has actor/reason audit. | `appview/internal/api/safety_admin_auth_test.go` |
| IT-014 | FR-016, RULE-001 | AC-016, AC-017 | Apply sanctions only from supported authorized decisions. | Every serious reason and effect combination | Submit decisions and raw signals | Valid decisions apply approved effects; reports/matches alone do not. | Existing moderation policy/enforcement integration suites |
| IT-015 | FR-018, FR-019 | AC-014, AC-015 | Persist/link external intake and correspondence. | Non-user, representative, pseudonymous, repeated email fixtures | Accept and append messages | One canonical item has stable reference/provenance; contact restricted; no fake DID or duplicate guilt signal. | `appview/internal/safetyintake/email_acceptance_test.go` |
| IT-016 | FR-020, NFR-006 | AC-015, AC-018 | Verify appeal ownership, enforcement continuity, and reversals. | Owner decision and email correspondence | Confirm and resolve appeal | Unverified/wrong-owner requests fail; enforcement persists; all transitions are attributed and append-only. | Existing moderation appeal integration suites |
| IT-017 | FR-023 | AC-019 | Persist policy acceptance server-side. | New account and versioned terms | Complete onboarding and fetch status | Missing declaration blocks completion; accepted version/time are stored per account. | Onboarding API and migration tests |
| IT-018 | FR-024 | AC-020, AC-049 | Enforce eligibility restrictions while preserving essential actions. | Reviewed under-16 restriction | Call normal and retained routes | Restricted product use fails; appeal, privacy, removal, deletion, blocks, mutes, sign-out/sign-in remain available. | `appview/internal/middleware/age_eligibility_test.go` |
| IT-019 | FR-025, FR-026, RULE-002 | AC-007, AC-024, AC-034 | Exercise all suppression surfaces and caches. | Pending, matched, hidden, taken-down fixtures | Query feeds, profiles, pins, threads, reads, replies, search, notifications, events, quotes, reposts, previews and replay | No suppressed subject/derivative appears; cache evicted; PDS source untouched. | Cross-package moderation visibility suites |
| IT-020 | FR-027 | AC-025 | Deny all production video server paths. | Production configuration | Call token, upload, publication, and project-post paths | Every video operation is unavailable and no video record is created through CraftSky. | Video API/handler tests |
| IT-021 | FR-028, NFR-004 | AC-009, AC-029 | Aggregate safe queue health and prioritized work. | Synthetic queue ages/states/deadlines/assignments | Query admin status | Ordering and counts are correct; alerts/cover visible; restricted values absent. | `appview/internal/api/safety_admin_test.go` |
| IT-022 | FR-011, FR-029 | AC-022 | Execute retention across implemented storage classes. | Expired/live/held DB rows, S3 objects and operational records | Run retention job repeatedly | Due data is deleted idempotently; held/live data remains; non-sensitive completion evidence exists. | `appview/internal/retention/worker_acceptance_test.go` |
| IT-023 | FR-030 | AC-030 | Test public pages before/after valid analytics consent. | Clean browser contexts | Load all public routes and inspect network/storage | PostHog is absent before consent; approved loading occurs only after consent. | Public-site browser test suite |
| IT-024 | FR-037 | AC-047 | Verify the mailbox ingestion boundary rejects media. | Email fixtures with images/video/URLs | Process inbound notification, not raw preview | Media never enters ordinary intake/case objects and safe handling status is recorded. | `appview/internal/safetyintake/mailbox_boundary_test.go` |
| IT-025 | NFR-002, RULE-003 | AC-032, AC-034 | Leak-test all ordinary outputs and private/public boundaries. | Canary secrets in every restricted field | Run workflows and inspect logs, Sentry sink, analytics, errors, push, APIs and PDS writes | No canary appears outside approved restricted storage/audit; no private PDS record exists. | Privacy sink boundary integration suites |
| IT-026 | FR-014, FR-017, FR-022, FR-038 | AC-013, AC-046, AC-048 | Submit every report route through Flutter repository/API contract. | Category fixtures | Build and send each route | Supported allegations use camelCase API values; IP does not call report API; safety guidance is preserved. | Flutter repository/widget tests and `appview/internal/api/report_test.go` |
| IT-027 | FR-010, FR-021, NFR-006 | AC-021 | Persist intimate-image workflow fields and actor history. | Approved target and complaint fixtures | Process judgment/search/action/outcome | Required chronology is complete and deadline calculations use only approved configuration. | `appview/internal/safetyincident/intimate_image_test.go` |
| IT-028 | FR-022, NFR-006 | AC-023 | Persist verified authority workflow and minimized disclosure. | Genuine/fraudulent authority fixtures | Verify, approve, disclose, close | Fraudulent request is rejected; genuine request requires lawful approval; export excludes credentials/unrelated data; review is recorded. | `appview/internal/safetyincident/authority_workflow_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | The author may see a local synthetic image post before refresh, but AppView does not serve it. | FR-033 | AC-040 | Extend `app/test/feed/providers/create_post_provider_test.dart` and verify another client/direct API cannot read it before clear. |
| REG-002 | Moderation hides CraftSky presentation without deleting PDS records. | RULE-002 | AC-024, AC-034 | Existing moderation PDS-boundary tests assert no delete call for scan/incident/moderation outcomes. |
| REG-003 | Reports, cases, mutes, appeals, and other private data remain AppView-only. | RULE-003 | AC-032, AC-034 | Schema/write tests reject any private safety record construction through PDS adapters. |
| REG-004 | Ozone is absent from this architecture slice. | RULE-008 | AC-037 | Dependency/config/route scan fails if Ozone integration or Ozone-dependent behavior is introduced. |
| REG-005 | Suspended members retain report/appeal/safety/account-maintenance exceptions. | BR-003, FR-020 | AC-013, AC-015 | Extend middleware/report tests for reporting, appeals, blocks/mutes, privacy, removal, sign-out, and deletion. |
| REG-006 | Duplicate reports remain allegations and do not independently prove guilt. | RULE-001 | AC-017 | Repeated report integration test verifies no automatic decision/effect and preserves separate provenance. |
| REG-007 | Dormant video code cannot become callable in production through a flag default. | FR-027 | AC-025 | Production configuration tests cover UI and all authorization/publication endpoints. |
| REG-008 | Launch age collection stays data-minimal. | RULE-010 | AC-049 | Request/schema/UI tests fail if age band, DOB, facial estimate, or age-derived targeting fields are introduced. |
| REG-009 | Standard `/v1/` responses keep camelCase and the approved error envelope. | FR-018, FR-028 | AC-014, AC-029 | New intake/admin API contract tests use existing JSON/error assertions. |
| REG-010 | Existing decision/effect/history/appeal behavior remains the enforcement foundation. | FR-009, FR-016, FR-020 | AC-016, AC-018, AC-043 | Existing moderation integration suite passes for report- and system-detected cases. |

## 7. Test Data

All media fixtures must be benign synthetic bytes. No real CSAM, known illegal
hash, victim image, production IWF response, personal mailbox content, or live
authority credential may enter source control, CI, logs, screenshots, or test
artifacts.

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Scanner outcomes | Deterministic benign image fixtures mapped to all five scan states | AT-002, AT-003, AT-008, IT-002 through IT-008 |
| TD-002 | Blob identity | Valid immutable CIDs reused across records and policy versions | UT-003, UT-005, IT-005 |
| TD-003 | In-scope records | Generated post, business event, profile, and external-embed JSON from CraftSky and third-party clients | IT-002 through IT-004 |
| TD-004 | Safety incidents | Synthetic incident identifiers, provider references, integrity values, deadlines, and histories | AT-007 through AT-009, IT-006 through IT-013 |
| TD-005 | Restricted canaries | Non-harmful unique tokens standing in for hashes, evidence, contacts, credentials, and judgments | IT-011, IT-025 |
| TD-006 | Reports/taxonomy | One benign report per approved group/reason plus IP and immediate danger | AT-004, UT-006, IT-014, IT-026 |
| TD-007 | External email | RFC 5322 fixtures for non-user, representative, pseudonymous, repeated correspondence, and media attachment cases | AT-005, IT-015, IT-024 |
| TD-008 | Appeals | Owner-visible decisions, correct/wrong owners, correspondence and outcomes | AT-006, IT-016 |
| TD-009 | Holds/retention | No hold, valid scoped hold, expired hold, invalid indefinite hold across each implemented class | AT-009, UT-009, UT-019, IT-012, IT-022 |
| TD-010 | Age controls | Accepted policy versions and specific credible/prohibited synthetic age signals | AT-010, AT-011, UT-012, IT-017, IT-018 |
| TD-011 | Authority requests | Clearly synthetic valid, incomplete, and fraudulent requests with no real credentials | AT-014, IT-028 |
| TD-012 | Roles | Founder, ordinary moderator, scoped helper, safety administrator, and unauthorized actor | AT-016, UT-010, IT-013, IT-021 |
| TD-013 | Accessibility | Long localized copy, 200% text scaling, focus order, semantic labels, and validation errors | AT-019, MAN-012 |
| TD-014 | Policy evidence | P0 controls with approved, missing, stale, and mismatched evidence/version metadata | AT-001, AT-012, UT-021 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-001 | AC-001 | Accountable launch review | Review every P0, linked artifact, owner approval, date, and residual condition. | No public launch approval while any P0 is open, stale, unowned, or unsupported. |
| MAN-002 | BR-004, FR-013 | AC-010, AC-011 | Approved CSEA/NCA tabletop | Use only synthetic metadata; execute priority, report, supplement, duplicate, follow-up, retention, and closure with trained roles. | End-to-end record is complete and approved observers sign off without handling prohibited test media. |
| MAN-003 | NFR-007, RULE-009 | AC-050, AC-051 | AWS restricted-bucket control review | Inspect region, account, bucket policy, block-public-access, IAM, KMS, logging, lifecycle, CDN/routes, and break-glass access. | Bucket is Frankfurt-only, private, separately controlled/audited, lifecycle-bound, and inaccessible to ordinary roles/public paths. |
| MAN-004 | FR-021 | AC-021 | Intimate-image complaint tabletop | Run eligible/ineligible standing, approved deadline, same/substantially-same search, exception, outcome, and escalation cases. | Approved target and complete attributable record are met; unresolved legal target blocks launch. |
| MAN-005 | FR-022 | AC-023, AC-048 | Threat and fraudulent-authority tabletop | Exercise emergency guidance, urgent assignment, preservation, independent verification, legal review, refusal/disclosure, and post-review. | CraftSky is not presented as emergency service; fraud is rejected; genuine disclosure is minimized and approved. |
| MAN-006 | FR-037, RULE-006 | AC-014, AC-015, AC-047 | Mailbox readiness | Send benign image/video attachments and URL-only reports through provider configuration; inspect quarantine, previews, forwarding, acknowledgements, references, escalation, and outcomes. | Media is rejected/quarantined without preview/forwarding; URL-only intake remains operational and auditable. |
| MAN-007 | FR-031 | AC-031 | Privacy-rights and complaint exercise | Run synthetic access/deletion/correction/objection and privacy complaint cases. | Identity, search, exemptions, decision, clock, escalation, and response evidence meet approved procedure. |
| MAN-008 | FR-031, NFR-002 | AC-031, AC-032 | Personal-data-breach exercise | Seed a synthetic restricted-data exposure and execute containment, assessment, notification decision, remediation, and review. | Applicable clocks and actors are recorded; exercises and notifications contain no real restricted content or credentials. |
| MAN-009 | FR-007, NFR-005, RULE-004 | AC-027, AC-028 | IWF contract/security validation | With IWF documentation and benign provider fixtures, review request/result/minimization/retention/auth/rate-limit/error semantics and run sandbox validation. | Adapter matches the approved contract, minimizes transfer/retention, and production readiness succeeds only with validated credentials. |
| MAN-010 | BR-005 | AC-019, AC-020 | Cross-channel 16+ consistency | Compare onboarding, help/support scripts, enforcement, store listings, terms, privacy, reporting, and child-readable copy. | Every channel states and applies the approved threshold and local-higher-minimum rule consistently. |
| MAN-011 | RULE-005 | AC-002, AC-035 | Index-only containment approval | Present the PDS-before-Tap data flow and controls to provider/specialist reviewer. | Written approval is linked, or launch remains blocked until a tested pre-upload requirement replaces this boundary. |
| MAN-012 | NFR-003 | AC-033 | WCAG 2.2 AA assistive-technology pass | Test safety flows on supported platforms with keyboard, VoiceOver/TalkBack, 200% text, contrast checks, focus, and error recovery. | No critical/serious failure and all approved WCAG 2.2 AA criteria pass with evidence. |
| MAN-013 | RULE-007 | AC-029, AC-036 | Founder-unavailable coverage exercise | Remove founder access during synthetic urgent work and invoke the rota/cover process. | Named trained helper receives alerts, performs only assigned responsibilities, and records recusal/escalation. |
| MAN-014 | BR-006 | AC-026 | Atomic policy publication review | Compare all five routes against shipped release evidence and verify versions/effective dates/rollback. | Policies change atomically and contain no unsupported present-tense control, stale age statement, or unresolved operator placeholder. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Real IWF adapter contract and benign provider test vectors are unavailable. | FR-007, FR-008, NFR-005, RULE-004 | The private protocol must not be guessed. The bounded coding plan excludes the production adapter. | Obtain IWF access/documentation, revise contract tests, complete DPIA/security review, and pass MAN-009 before production-adapter completion or launch. |
| GAP-002 | Documentary evidence for index-time-only containment approval is pending. | RULE-005, BR-002 | The accountable owner confirmed on 17 September 2026 that specialist/provider approval was obtained, resolving the boundary for planning, but the reviewer/date/reference/conditions have not been recorded. | Attach and verify the approval evidence through MAN-011 before implementation completion or launch; amend requirements if the evidence introduces conditions. |
| GAP-003 | Intimate-image response target and operating coverage are unresolved. | FR-021 | The register's 48-hour value is not authoritative until specialist approval. | Configure only the approved target, then execute AT-013/MAN-004. |
| GAP-004 | Eligible NCA administrator and named cover are not assigned/trained. | BR-004, RULE-007, RULE-009 | Automated tests cannot establish legal eligibility, training, wellbeing, or availability. | Record authorization/training and pass MAN-002/MAN-013. |
| GAP-005 | Safe mailbox provider/configuration is not selected. | FR-018, FR-037, RULE-006 | Repository tests cannot prove provider preview/forwarding/retention behavior. | Select/configure a provider and pass MAN-006 with benign media. |
| GAP-006 | Final operator/controller details and DPO position are unresolved. | BR-006, FR-031 | Policies and rights procedures cannot publish with placeholders. | Obtain approved legal details before AT-012/MAN-014 can pass. |
| GAP-007 | Worldwide jurisdiction review is incomplete. | BR-001, BR-005 | Local higher ages, reporting, privacy, and availability controls may vary. | Approve controls or constrain launch territories; link outcome to P0 gate. |
| GAP-008 | Restricted AWS controls need deployment-level verification. | NFR-007, RULE-009 | MinIO integration tests do not prove AWS region/IAM/KMS/public-access posture. | Provision with reviewed infrastructure and pass MAN-003. |
| GAP-009 | Retention cannot be fully automated for third-party email, Sentry, PostHog, backups, and local drafts until provider/device contracts exist. | FR-029 | AppView jobs do not control every listed system. | Add provider-specific deletion evidence and device tests to IT-022 before release. |
| GAP-010 | Browser suite target for public PostHog behavior is not currently present in the discovered repository tests. | FR-030 | Consent behavior needs real network/storage browser assertions. | Select the public-site test harness and implement IT-023 before policy publication. |
| GAP-011 | WCAG conformance requires supported-device and assistive-technology evidence beyond widget tests. | NFR-003 | Semantics tests do not prove real screen-reader behavior. | Complete MAN-012 and remediate all release-blocking findings. |
| GAP-012 | Public policy claims depend on subscriptions/commercial states being implemented or removed. | BR-006 | Requirements leave this as a publication blocker. | Resolve copy/scope before AT-012 and MAN-014. |

## 10. Out Of Scope

- Real or simulated illegal imagery, production IWF payloads before contract
  approval, and testing that requires staff to view suspected CSAM.
- Ozone, novel-content classifiers, video scanning, DMs, livestreaming, precise
  location, and public moderation/safety lexicon records.
- Highly effective age assurance, age bands, full dates of birth, facial-age
  estimation, and age-based personalization.
- Network-wide deletion or moderation-driven deletion of source PDS records.
- A public web reporting form while the approved email route passes readiness.
- Penetration testing and independent legal/privacy/accessibility opinions; these
  remain separate release evidence rather than substitutes for this specification.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if initiated by the user:
  `docs/changes/2026-09-16-online-safety-launch-readiness/`
- Recommended first failing test for implementation: `UT-001`, followed by
  `UT-002` and `IT-003`; this establishes fail-closed state and visibility before
  adding queues or admin behavior.
- Suggested test order for implementation: schema/migrations; scan states and
  adapter stub; Tap dependency extraction and visibility; retries/replay; incidents
  and evidence; moderation integration; report/email/appeal flows; age controls;
  retention/deletion; admin/observability; policy/production gates; table exercises.
- Commands discovered: `just appview-test-unit`, `just test`, `just appview-check`,
  `just app-test <paths>`, and `just app-analyze`.
- Bounded coding planning is authorized for provider-neutral infrastructure and the
  deterministic stub. It must exclude production IWF adapter implementation.
- Blocking completion/launch gaps: GAP-001 through GAP-012. No implementation
  completion or launch approval is implied by this test design.
