# Acceptance Test Specification: Scheduled Posts

## 1. Test Strategy

Risk level: High. Scheduled posts combine private unpublished content, cross-device mutation, durable background work, private object storage, delayed OAuth-authenticated PDS writes, and an external side effect that cannot share a transaction with AppView state. The suite therefore uses several complementary levels:

- Flutter widget/provider tests cover the `When` control, eligible composers, private-staging progress and recovery, capacity UX, Settings management, full-composer editing, account isolation, manual refresh, accessibility, and preservation of the immediate-post path.
- Go unit tests cover schedule-time validation, status/capacity rules, retry timing, failure classification, frozen publication identity/body, cleanup deadlines, safe object metadata, and HTTP-independent worker construction.
- Go integration tests use `httptest`, fakes, and `internal/testdb.WithSchema` for the migration contract, API contracts, ownership, transactional capacity, last-write-wins edits, leases, stale-worker fencing, session selection, PDS crash recovery, cleanup, route registration, operational signals, and privacy canaries.
- S3-compatible adapter tests run against MinIO for real private-object behavior; fake object storage remains the fast default for worker tests.
- Regression tests protect existing immediate posts, eager PDS image upload, quote/reply publication, project validation, cache behavior, and the absence of new notification production.
- Manual checks are limited to real-device presentation, real AppView/PDS/Tap execution, multi-device behavior, and deployed storage/alert controls that cannot be proven by repository tests.

Time-dependent tests must inject clocks/timers; they must not wait five minutes, 30 minutes, 24 hours, or 30 days in wall-clock time. PDS and object-storage fakes must expose deterministic barriers around upload, record write, and completion persistence so every crash/race boundary is testable.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-016 | AT-001, AT-009, IT-008, MAN-003 | Acceptance / Integration / Manual | Yes, except real-stack confirmation |
| BR-002 | AC-006, AC-007 | AT-005, IT-003 | Acceptance / Integration | Yes |
| BR-003 | AC-008, AC-009, AC-010 | AT-006, AT-007, AT-008, IT-004, IT-005, IT-006, REG-007 | Acceptance / Integration / Regression | Yes |
| BR-004 | AC-012, AC-013 | AT-004, AT-013, IT-009, IT-014, IT-024 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-025 | AT-001, AT-003, UT-007 | Acceptance / Unit | Yes |
| FR-002 | AC-002, AC-003 | AT-002, UT-001, IT-002, REG-003 | Acceptance / Unit / Integration / Regression | Yes |
| FR-003 | AC-004, AC-025 | AT-003, UT-002, UT-003, IT-002, MAN-001 | Acceptance / Unit / Integration / Manual | Yes |
| FR-004 | AC-001 | AT-001, REG-001, REG-003, REG-004, REG-008, REG-009 | Acceptance / Regression | Yes |
| FR-005 | AC-005, AC-012, AC-025 | AT-004, UT-009, UT-019, IT-001, IT-022, IT-024, IT-025, MAN-002 | Acceptance / Unit / Integration / Manual | Yes |
| FR-006 | AC-006, AC-007, AC-026 | AT-005, UT-004, UT-012, IT-003, IT-026 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-005, AC-008, AC-009, AC-010, AC-011 | AT-006, AT-007, AT-008, AT-013, IT-001, IT-002, IT-004, IT-005, IT-006, IT-019 | Acceptance / Integration | Yes |
| FR-008 | AC-008, AC-011, AC-027, AC-031, AC-032 | AT-006, UT-008, UT-014, UT-017, IT-004, IT-021, REG-007, REG-010 | Acceptance / Unit / Integration / Regression | Yes |
| FR-009 | AC-009, AC-028 | AT-007, UT-007, UT-009, IT-005, MAN-004 | Acceptance / Unit / Integration / Manual | Yes |
| FR-010 | AC-010, AC-014, AC-029 | AT-008, IT-006 | Acceptance / Integration | Yes |
| FR-011 | AC-014, AC-015, AC-029 | AT-008, AT-009, AT-011, UT-006, IT-006, IT-007, IT-008, IT-020, IT-026 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-013, AC-016, AC-017, AC-018, AC-020 | AT-009, AT-010, AT-011, UT-009, UT-010, IT-009, IT-012, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-017 | AT-011, UT-010, IT-007, IT-009, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-016, AC-017, AC-030, AC-032 | AT-009, AT-011, AT-012, UT-020, IT-010, IT-016, IT-017, IT-026, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| FR-015 | AC-018, AC-019, AC-020 | AT-010, UT-005, UT-006, UT-011, IT-011, IT-012, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-005 | AT-004, IT-022, REG-008 | Acceptance / Integration / Regression | Yes |
| FR-017 | AC-012, AC-013, AC-014 | AT-004, AT-013, UT-019, IT-014, IT-015, IT-024, IT-025, REG-002, REG-009, MAN-002 | Acceptance / Unit / Integration / Regression / Manual | Yes |
| FR-018 | AC-018, AC-021 | AT-010, UT-018, IT-013, MAN-004 | Acceptance / Unit / Integration / Manual | Yes |
| FR-019 | AC-022 | UT-016, IT-020 | Unit / Integration | Yes |
| FR-020 | AC-030 | AT-012, UT-013, UT-020, IT-016, IT-017, IT-026 | Acceptance / Unit / Integration | Yes |
| FR-021 | AC-031, AC-032 | AT-006, AT-012, UT-014, IT-023, REG-005, REG-006 | Acceptance / Unit / Integration / Regression | Yes |
| NFR-001 | AC-015, AC-016, AC-017 | AT-009, AT-011, IT-007, IT-008, IT-010, IT-026, MAN-003 | Acceptance / Integration / Manual | Yes |
| NFR-002 | AC-015, AC-016 | AT-009, IT-008, MAN-003 | Acceptance / Integration / Manual | Yes |
| NFR-003 | AC-023 | AT-013, UT-015, IT-018 | Acceptance / Unit / Integration | Yes |
| NFR-004 | AC-012, AC-023, AC-030 | AT-004, AT-012, AT-013, UT-015, IT-014, IT-015, IT-016, MAN-005 | Acceptance / Unit / Integration / Manual | Partial: production controls are a release gate |
| NFR-005 | AC-024 | AT-014, MAN-001 | Acceptance / Manual | Partial: semantics automated, device presentation manual |
| NFR-006 | AC-033 | IT-027, MAN-006 | Integration / Manual | Yes, except deployed alert confirmation |
| RULE-001 | AC-006, AC-007, AC-019 | AT-005, AT-010, UT-004, IT-003, IT-011 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-002, AC-003 | AT-002, UT-001, IT-002 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-004, AC-025 | AT-003, UT-002, UT-003, IT-002 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-011, AC-012 | AT-006, AT-013, IT-001, IT-004, IT-014 | Acceptance / Integration | Yes |
| RULE-005 | AC-014, AC-028, AC-029 | AT-007, AT-008, AT-011, UT-006, IT-005, IT-006, IT-007 | Acceptance / Unit / Integration | Yes |
| RULE-006 | AC-019, AC-020 | AT-010, UT-005, UT-011, IT-011, IT-012 | Acceptance / Unit / Integration | Yes |
| RULE-007 | AC-015 | AT-009, IT-008 | Acceptance / Integration | Yes |
| RULE-008 | AC-018, AC-019 | AT-007, AT-010, UT-006, UT-007, IT-011, IT-013 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Posting now remains the default

- Requirement IDs: BR-001, FR-001, FR-004
- Acceptance Criteria: AC-001
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/scheduled_posts/post_composer_scheduling_test.dart`

```gherkin
Feature: Choose when to publish
  Scenario: Publish immediately without changing When
    Given an eligible new-post composer is open
    Then the When row shows Now and the primary action shows Post
    When the member submits valid content
    Then the existing immediate-post request runs
    And no scheduled-post resource is created
```

### AT-002: Only eligible original posts can be scheduled

- Requirement IDs: FR-002, RULE-002
- Acceptance Criteria: AC-002, AC-003
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/scheduled_posts/scheduling_eligibility_test.dart`

```gherkin
Feature: Scheduled-post eligibility
  Scenario Outline: Show scheduling only for eligible composers
    Given a <composer> composer
    Then Schedule for later is <availability>

    Examples:
      | composer                    | availability |
      | original standard top-level | available    |
      | project top-level          | available    |
      | quote                      | unavailable  |
      | reply or comment           | unavailable  |
```

### AT-003: Choose a bounded absolute publication time

- Requirement IDs: FR-001, FR-003, RULE-003
- Acceptance Criteria: AC-004, AC-025
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/scheduled_posts/schedule_time_picker_test.dart`

```gherkin
Feature: Schedule time selection
  Scenario: Select a valid whole-minute time
    Given an eligible composer whose When row shows Now
    When the member chooses Schedule for later
    Then the picker permits whole-minute choices from five minutes through 28 days
    And the relevant timezone or offset is visible
    When a valid time is selected
    Then the primary action shows Schedule
    And later timezone changes do not change the stored UTC instant
```

### AT-004: Stage private media only when Schedule is submitted

- Requirement IDs: BR-004, FR-005, FR-016, FR-017, NFR-004, RULE-004
- Acceptance Criteria: AC-005, AC-012, AC-025
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/scheduled_posts/scheduled_post_submission_test.dart`

```gherkin
Feature: Submit an image schedule
  Scenario: Private staging succeeds before schedule creation
    Given an eligible composer image was eagerly uploaded to the PDS
    And its local source bytes remain available
    When the member taps Schedule
    Then private-staging progress is shown
    And retained local bytes are reprocessed and uploaded to owner-scoped private staging
    And the schedule is created only after staging succeeds
    And no public post or optimistic public-cache item is created
    And the initial PDS blob is not referenced by the schedule

  Scenario: Private staging or schedule creation fails
    Given valid scheduled content is ready
    When private staging or schedule creation fails
    Then the full composer remains open and unchanged
    And the member can retry without a duplicate schedule
```

### AT-005: Enforce and explain the three-item capacity

- Requirement IDs: BR-002, FR-006, RULE-001
- Acceptance Criteria: AC-006, AC-007, AC-026
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/scheduled_posts/scheduled_post_capacity_test.dart`

```gherkin
Feature: Scheduled-post capacity
  Scenario: All three slots are occupied
    Given the account has three retained items in countable statuses
    Then Schedule for later remains visible but disabled
    And the composer warns that another post cannot be scheduled
    And a management link is shown
    And Post now remains enabled
    When one item is published or deleted and state is refreshed
    Then Schedule becomes available again
```

### AT-006: Manage unpublished schedules from Settings

- Requirement IDs: BR-003, FR-007, FR-008, FR-021, RULE-004
- Acceptance Criteria: AC-008, AC-011, AC-027, AC-031
- Priority: Must
- Level: Acceptance
- Automation Targets: `app/test/settings/settings_page_test.dart`,
  `app/test/scheduled_posts/scheduled_posts_page_test.dart`, and
  `app/test/scheduled_posts/scheduled_posts_provider_test.dart`

```gherkin
Feature: Scheduled-post management
  Scenario: View and refresh the current account's schedules
    Given the active account has Scheduled, Retrying, Publishing, and Needs attention items
    When Settings is opened
    Then Scheduled posts shows the Needs attention count
    When the management screen opens
    Then unpublished rows are ordered by scheduled time
    And each row shows its permitted thumbnail, type, preview, local time and status
    And no other account or published-history item is shown
    And status does not poll automatically
    When the member pulls to refresh
    Then the latest status is fetched
    And no push or in-app failure notification is created
```

Implementation mapping: `settings_page_test.dart` renders the real Settings
page against account-keyed Alice and Bob repositories, asserts Alice's non-zero
Needs attention badge, switches to Bob and asserts the zero-count badge is
omitted, and follows the typed Scheduled posts route. The management page test
covers row content and Publishing locks; the provider test covers ordering,
manual refresh, and account-boundary state replacement.

### AT-007: Edit using the full composer

- Requirement IDs: BR-003, FR-007, FR-009, RULE-005, RULE-008
- Acceptance Criteria: AC-009, AC-028
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/scheduled_posts/scheduled_post_edit_test.dart`

```gherkin
Feature: Edit a scheduled post
  Scenario: Edit a future schedule
    Given a future schedule is listed
    When its row is tapped
    Then the matching full composer opens with every field and media preview
    And Schedule and its existing time remain selected

  Scenario: Recover a Needs attention item
    Given a Needs attention item is listed
    When its row is tapped
    Then the missed time is shown and Post now is selected
    And the member may post now, choose a new valid schedule, or delete
    And merely signing in never resumes publication
```

### AT-008: Delete before publication and lock during Publishing

- Requirement IDs: BR-003, FR-007, FR-010, FR-011, RULE-005
- Acceptance Criteria: AC-010, AC-014, AC-029
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/scheduled_posts/scheduled_post_delete_test.dart`

```gherkin
Feature: Delete a scheduled post
  Scenario: Delete before Publishing
    Given an unpublished item has not entered Publishing
    When the owner confirms Delete from the row or editor
    Then it disappears, releases its slot, cannot publish, and cleanup begins

  Scenario: Publication has started
    Given the item is Publishing
    Then Edit and Delete are locked with clear copy
    And a manual refresh reveals its later published or recoverable state
```

### AT-009: Publish a healthy due item exactly once

- Requirement IDs: BR-001, FR-011, FR-012, FR-014, NFR-001, NFR-002, RULE-007
- Acceptance Criteria: AC-013, AC-015, AC-016
- Priority: Must
- Level: Acceptance
- Automation Target: `appview/internal/scheduledposts/worker_acceptance_test.go`

```gherkin
Feature: Due publication
  Scenario: App is closed when a schedule becomes due
    Given a valid durable schedule and an active usable owner session
    And Flutter is closed or offline
    Then no PDS upload or record write occurs before scheduledAt
    When scheduledAt arrives under healthy dependencies
    Then one worker enters Publishing within 60 seconds
    And allocates the stable TID and freezes createdAt
    And uploads current private media to the owner's PDS
    And writes one record with that identity and body
    And removes the item from management and capacity
```

Implementation mapping: `worker_acceptance_test.go` drives a durable real-
Postgres schedule before due, at due plus 30 seconds, and through a repeated
batch. It asserts zero early PDS effects, one stable record write, management
removal/capacity release, and no replay.

### AT-010: Bound retries and require deliberate recovery

- Requirement IDs: FR-012, FR-015, FR-018, RULE-001, RULE-006, RULE-008
- Acceptance Criteria: AC-018, AC-019, AC-020, AC-021
- Priority: Must
- Level: Acceptance
- Automation Target: `appview/internal/scheduledposts/failure_acceptance_test.go`

```gherkin
Feature: Publication failure
  Scenario: A transient dependency remains unavailable
    Given a due schedule encounters a transient failure
    Then attempts are eligible at due and approximately plus 1, 3, 7, 15 and 30 minutes
    And bounded jitter never schedules an attempt beyond 30 minutes
    And the final failure becomes Needs attention and still counts toward three
    And it receives no automatic attempt after that

  Scenario: Current policy makes content permanently invalid
    Given validation, media, mention authorization, or block state rejects the content
    When publication is attempted
    Then it enters Needs attention immediately
    And no member-authored content is altered
```

Implementation mapping: `failure_acceptance_test.go` drives the complete due,
+1, +3, +7, +15, and +30 minute lifecycle, verifies the retained byte-identical
payload and final Needs attention state, rejects automatic work beyond the
window, and separately proves immediate policy-invalid preservation without a
PDS write.

### AT-011: Recover the same record across crashes and leases

- Requirement IDs: FR-011, FR-012, FR-013, FR-014, NFR-001, RULE-005
- Acceptance Criteria: AC-017
- Priority: Must
- Level: Acceptance
- Automation Target: `appview/internal/scheduledposts/recovery_acceptance_test.go`

```gherkin
Feature: Publication recovery
  Scenario Outline: Worker stops around the PDS record write
    Given Publishing has persisted a stable TID, createdAt and record body
    When the worker stops <boundary>
    And its lease expires or AppView restarts
    Then no post record is written until every current image is available
    And content-addressed PDS blob uploads may be safely reused or repeated
    And recovery never attempts to delete PDS blobs
    And recovery reconciles the same record identity and body
    And at most one visible post exists
    And a stale worker cannot overwrite the recovered outcome

    Examples:
      | boundary                                                       |
      | before any PDS media upload                                    |
      | after one or more PDS media uploads but before the record write |
      | after PDS record success but before local completion            |
```

Implementation mapping: `recovery_acceptance_test.go` covers failure before any
upload, after a partial upload, after all media but before the record write, and
after an ambiguously committed record. Every case compares the frozen body and
predicted blob references and permits at most one record write.

### AT-012: Remove private content on bounded lifecycle events

- Requirement IDs: FR-014, FR-020, FR-021, NFR-004
- Acceptance Criteria: AC-030, AC-032
- Priority: Must
- Level: Acceptance
- Automation Target: `appview/internal/scheduledposts/retention_acceptance_test.go`

```gherkin
Feature: Scheduled-content retention
  Scenario: Lifecycle deadlines are reached
    Given unclaimed, Needs attention, and published-tombstone fixtures
    When the injected clock crosses each retention deadline
    Then unclaimed staging is removed after 24 hours
    And Needs attention payload and media are removed after its visible 30-day date
    And published tombstones are removed after 30 days
    And successful publication starts private cleanup immediately
    And referenced live objects are never removed as orphans
    And no success notification or management-history row is created
```

Implementation mapping: `retention_acceptance_test.go` proves inclusive
deadline behavior for unclaimed staging, Needs attention content, tombstones,
and live references. Immediate post-publication cleanup/capacity release is
covered by `finalization_test.go`; replacement, delete, account deletion,
idempotent cleanup, and retry safety remain in `cleanup_test.go`.

### AT-013: Keep unpublished content owner-private and out of telemetry

- Requirement IDs: BR-004, FR-007, FR-017, NFR-003, NFR-004, RULE-004
- Acceptance Criteria: AC-011, AC-012, AC-023
- Priority: Must
- Level: Acceptance
- Automation Target: `appview/internal/scheduledposts/privacy_acceptance_test.go`

```gherkin
Feature: Private scheduled content
  Scenario: Another account guesses private identifiers
    Given Alice owns a schedule and staged image containing privacy canaries
    When Bob attempts get, update, delete, or preview operations
    Then neither existence nor content is disclosed or changed
    And Alice can access the authenticated preview
    And logs, traces, metrics, analytics, errors and object keys contain none of the canaries or credentials
```

Implementation mapping: `privacy_acceptance_test.go` exercises owner/foreign
schedule and media access against real Postgres plus the private object store
and scans emitted safe metadata for payload, media, DID, and credential
canaries. The authenticated HTTP non-disclosure envelopes are exercised by
`internal/api/scheduled_post_test.go` and `scheduled_media_test.go`;
`internal/api/scheduled_privacy_test.go` captures scheduled-handler logs and
error envelopes around canary-bearing store and media failures. Complete
publication/retry/Needs-attention/recovery/stale-worker/cleanup metric capture
is exercised by `observability_test.go`; concrete metric names and bounded
attributes are checked by `internal/observability/scheduled_posts_test.go`.
`internal/middleware/metrics_test.go` proves HTTP traces use route patterns
rather than identifiers or query values, and
`internal/observability/error_classifier_test.go` proves captured errors retain
only classified sentinels rather than raw provider details.

### AT-014: Scheduling controls remain accessible

- Requirement IDs: NFR-005
- Acceptance Criteria: AC-024
- Priority: Should
- Level: Acceptance
- Automation Target: `app/test/scheduled_posts/scheduled_posts_accessibility_test.dart`

```gherkin
Feature: Accessible scheduling
  Scenario: Use scheduling with assistive presentation
    Given large text and semantics inspection are enabled
    When a member selects a time, stages media, manages a status, edits, or confirms deletion
    Then every required action remains reachable without clipping
    And controls have meaningful localized labels
    And validation, progress, lock and failure status changes are announced
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-002, RULE-002 | AC-002, AC-003 | Classify schedulable post kinds. | Original standard, project, quote embed, reply/comment reference. | Only original standard and valid standalone project return eligible. | `appview/internal/scheduledposts/validation_test.go` |
| UT-002 | FR-003, RULE-003 | AC-004, AC-025 | Validate schedule boundaries using injected server time. | Whole/non-whole minutes at 4:59, 5:00, 28 days, and 28 days plus one minute. | Only whole-minute values within the inclusive bounds pass. | `appview/internal/scheduledposts/validation_test.go` |
| UT-003 | FR-003, RULE-003 | AC-004, AC-025 | Preserve the absolute instant while formatting local timezone/offset. | One UTC instant rendered before/after timezone and DST changes. | Encoded UTC remains identical; display reflects current local zone and offset. | `app/test/scheduled_posts/schedule_time_test.dart` |
| UT-004 | FR-006, RULE-001 | AC-006, AC-007, AC-019 | Count capacity statuses. | Scheduled, Publishing, Retrying, Needs attention, published, deleted. | First four count; published/deleted do not. | `appview/internal/scheduledposts/state_test.go` |
| UT-005 | FR-015, RULE-006 | AC-019 | Generate the bounded retry plan. | Due time, attempt number, deterministic jitter extremes. | Eligibility follows due/+1/+3/+7/+15/+30 and never crosses due+30. | `appview/internal/scheduledposts/retry_test.go` |
| UT-006 | FR-011, FR-015, RULE-005, RULE-006, RULE-008 | AC-018, AC-019, AC-020, AC-029 | Validate lifecycle transitions and mutation locks. | Every status/event pair, including stale versions. | Only allowed transitions occur; Publishing locks member mutations; terminal Needs attention never auto-runs. | `appview/internal/scheduledposts/state_test.go` |
| UT-007 | FR-001, FR-009, RULE-008 | AC-001, AC-009, AC-025 | Initialize composer timing state. | New composer, future edit, Needs attention edit with missed time. | New defaults Now; future edit preserves time; Needs attention defaults Post now and retains missed-time display. | `app/test/scheduled_posts/schedule_composer_state_test.dart` |
| UT-008 | FR-008 | AC-008 | Map scheduled resources to bounded management rows. | Text-only and image/project fixtures with long text. | First image, type/title, bounded preview, localized time, and exact status are produced without full payload leakage. | `app/test/scheduled_posts/scheduled_post_row_model_test.dart` |
| UT-009 | FR-005, FR-009, FR-012 | AC-005, AC-009, AC-020 | Round-trip the complete editable payload without mutation. | Standard/project payloads with text, facets, languages, project fields, alt text, and media order. | Create/edit/publication snapshots retain every field and byte-significant ordering. | `appview/internal/scheduledposts/payload_test.go` |
| UT-010 | FR-012, FR-013 | AC-017 | Freeze publication identity and body once. | First attempt followed by transient failures and recovery. | TID, createdAt, serialized body, and intended URI remain identical. | `appview/internal/scheduledposts/publication_test.go` |
| UT-011 | FR-015, RULE-006 | AC-019, AC-020 | Classify publication failures. | Timeouts, retryable PDS/storage responses, invalid payload/media, mention/block denial. | Transient errors retry; permanent errors enter Needs attention immediately with safe codes. | `appview/internal/scheduledposts/failure_test.go` |
| UT-012 | FR-006 | AC-026 | Derive full-cap composer actions. | Counts 0 through 3 and schedule/now selection. | At 3, Schedule is visible/disabled with exact count and Manage link; Post now remains enabled. | `app/test/scheduled_posts/schedule_capacity_state_test.dart` |
| UT-013 | FR-020 | AC-030 | Calculate cleanup deadlines. | Unclaimed upload, Needs attention transition, publication time. | Deadlines are exactly 24 hours, 30 days, and 30 days respectively; success cleanup is immediately eligible. | `appview/internal/scheduledposts/retention_test.go` |
| UT-014 | FR-008, FR-021 | AC-031, AC-032 | Limit public UI status vocabulary and visibility. | All internal lifecycle states, published result. | UI emits only four agreed unpublished statuses; published is absent from management. | `app/test/scheduled_posts/scheduled_post_status_test.dart` |
| UT-015 | NFR-003, NFR-004 | AC-023 | Produce opaque object keys and safe diagnostic fields. | DID, handle, filename, alt text, exact time, tokens, safe IDs. | Keys/attributes contain only opaque IDs and approved low-cardinality fields. | `appview/internal/scheduledposts/privacy_test.go` |
| UT-016 | FR-019 | AC-022 | Construct and invoke the worker without HTTP context. | Store, clock, session selector, PDS and object-store interfaces. | One batch can run independently of routes/request context and enforces a bounded batch. | `appview/internal/scheduledposts/worker_test.go` |
| UT-017 | FR-008 | AC-008, AC-027 | Sort and refresh account-keyed scheduled state. | Out-of-order rows, duplicate refresh, changed account key. | Rows sort by scheduledAt; refresh replaces state; account keys never share cached rows. | `app/test/scheduled_posts/scheduled_posts_provider_test.dart` |
| UT-018 | FR-018 | AC-018, AC-021 | Decide whether publication has an active usable owner session. | Multiple devices, one sign-out, last sign-out, expired/revoked sessions. | Another active owner session may be selected; absent active authorization returns retryable auth-unavailable. | `appview/internal/scheduledposts/session_test.go` |
| UT-019 | FR-005, FR-017 | AC-005, AC-012, AC-025 | Drive submit-time private-staging state. | Retained local bytes, stage progress, stage failure, create failure, retry. | Staging starts only on Schedule; progress is visible; content/source survive failure; retries reuse idempotency safely. | `app/test/scheduled_posts/private_staging_state_test.dart` |
| UT-020 | FR-014, FR-020 | AC-030, AC-032 | Project published state into a content-free tombstone. | Completed schedule with payload/media and URI/CID. | Tombstone retains only allowed identity/idempotency/timestamps and no previewable content. | `appview/internal/scheduledposts/tombstone_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-005, FR-007, RULE-004 | AC-005, AC-011 | Create is strict, owner-derived, and idempotent. | Authenticated `httptest` handler, fake staged-media ownership, operation key. | Repeat the same valid camelCase request, then vary owner/body. | Same operation returns one resource/slot; owner comes from auth; conflicting reuse is safely rejected. | `appview/internal/api/scheduled_post_test.go` |
| IT-002 | FR-002, FR-003, FR-007, RULE-002, RULE-003 | AC-003, AC-004 | Reject excluded kinds and invalid times at the HTTP boundary. | Injected server clock and authenticated handler. | Submit quote/reply/project-with-quote and every time boundary violation. | Standard error envelope/machine code is returned and no schedule/media claim is retained. | `appview/internal/api/scheduled_post_request_test.go` |
| IT-003 | BR-002, FR-006, RULE-001 | AC-006, AC-007 | Enforce the final slot transactionally in real Postgres. | `testdb.WithSchema`, two connections, two existing countable rows. | Concurrently create the third and fourth contenders. | Exactly one commits; count never exceeds three; later publish/delete releases capacity. | `appview/internal/scheduledposts/store_test.go` |
| IT-004 | BR-003, FR-007, FR-008, RULE-004 | AC-008, AC-011 | List/get are owner-scoped, ordered, and content-shaped. | Alice/Bob fixtures in all statuses plus published tombstone. | Alice and Bob list/get, including guessed IDs. | Each sees only owned unpublished rows in order; published and foreign content are not disclosed. | `appview/internal/api/scheduled_post_test.go` |
| IT-005 | FR-009, RULE-005 | AC-009, AC-028 | Member updates are last-write-wins while worker versions are fenced. | Real Postgres schedule and two device requests plus a claimed older worker version. | Commit two valid full updates, then let stale worker attempt publication. | Last accepted edit is authoritative and stale claim cannot cross the write boundary. | `appview/internal/scheduledposts/store_test.go` |
| IT-006 | FR-010, FR-011, RULE-005 | AC-010, AC-014, AC-029 | Serialize delete/edit against Publishing. | Barrier-controlled store transaction and fake external writer. | Race update/delete with Publishing transition in both commit orders. | Mutation wins before Publishing; otherwise conflict is returned before external write and UI-safe lock status is exposed. | `appview/internal/scheduledposts/store_test.go`, `appview/internal/api/scheduled_post_test.go` |
| IT-007 | FR-011, FR-013, NFR-001, RULE-005 | AC-014, AC-015, AC-017 | Claim due work with exclusive expiring leases. | Real Postgres, multiple workers, due/future rows, injected clock. | Claim concurrently, expire one lease, and submit stale completion. | Each row has one valid lease; future rows remain untouched; recovered owner wins and stale completion is rejected. | `appview/internal/scheduledposts/store_test.go` |
| IT-008 | BR-001, FR-011, NFR-001, NFR-002, RULE-007 | AC-015, AC-016 | Never publish early and meet the healthy start target. | Fake clock/ticker, durable store, healthy fakes, restarted worker. | Advance to before due, due, and due+60 seconds. | No early side effect; one Publishing transition begins within the target after restart. | `appview/internal/scheduledposts/worker_test.go` |
| IT-009 | BR-004, FR-012, FR-013 | AC-013, AC-017 | Recover after a partial PDS media upload while preserving stable record identity/body. | Fake private store, recording PDS, four-image/project fixture, and a barrier after one or more blob uploads. | Stop before the final record write, expire the lease/restart, and recover the attempt. | No post record exists until all current media is available; content-addressed blobs may be safely reused/re-uploaded; every record write uses the identical TID/createdAt/body; no draft payload is written early; and Craftsky never requests PDS blob deletion. | `appview/internal/scheduledposts/recovery_acceptance_test.go` |
| IT-010 | FR-013, FR-014, NFR-001 | AC-017 | Reconcile a PDS success not recorded locally. | Fake PDS commits then returns ambiguous error; local success barrier fails. | Recover after lease expiry/restart. | Worker finds/replaces the same URI as required, records URI/CID once, and creates no second visible post. | `appview/internal/scheduledposts/recovery_acceptance_test.go` |
| IT-011 | FR-015, RULE-001, RULE-006, RULE-008 | AC-019 | Execute the complete transient retry lifecycle. | Injected clock, deterministic jitter, always-transient fake. | Advance through all six eligible attempts and beyond +30. | Six bounded attempts occur; item becomes Needs attention, counts toward capacity, and never runs again automatically. | `appview/internal/scheduledposts/worker_test.go` |
| IT-012 | FR-012, FR-015, RULE-006 | AC-020 | Revalidate current policy without rewriting content. | Previously valid fixture made invalid by media/mention/block/validation state. | Run due publication. | No PDS record is written; Needs attention is immediate; payload/media/order remain byte-equivalent. | `appview/internal/scheduledposts/publication_test.go` |
| IT-013 | FR-012, FR-015, FR-018, RULE-008 | AC-018, AC-021 | Enforce active-owner-session and account lifecycle rules. | Multiple owner sessions, other-account session, last sign-out, account-deletion barrier. | Publish across each session state and race deletion with claim. | Another active owner session works; no anonymous/foreign write occurs; last sign-out exhausts to Needs attention; account deletion fences work and removes private data. | `appview/internal/scheduledposts/session_test.go`, `failure_acceptance_test.go`, `cleanup_test.go`; production composition in `appview/internal/app/instagram_lifecycle_test.go` |
| IT-014 | FR-017, NFR-004, RULE-004 | AC-011, AC-012 | Authenticate private staging upload and preview. | `httptest` API, fake S3 adapter, Alice/Bob sessions. | Alice uploads/previews; Bob guesses media ID; invalid/oversized content is submitted. | Only valid owner operations succeed; safe errors reveal no foreign existence; object remains private. | `appview/internal/api/scheduled_media_test.go` |
| IT-015 | FR-017, NFR-004 | AC-012, AC-030 | Exercise the S3-compatible adapter against MinIO. | Disposable MinIO bucket with private policy and test credentials. | Put/get/delete/checksum an object and attempt anonymous/cross-prefix access. | Owner service path works over configured transport; unauthorized access fails; metadata/key rules hold; cleanup is idempotent. | `appview/internal/scheduledposts/objectstore_minio_test.go` |
| IT-016 | FR-014, FR-020, NFR-004 | AC-030 | Clean orphans, replacements, deletes, account deletion, and expiries safely. | Real Postgres metadata, fake object store, shared/reference fixtures, injected clock. | Run cleanup at each deadline and retry injected delete failures. | Eligible objects/payloads are removed, live references survive, slots release, and failures remain retryable without intentional retention extension. | `appview/internal/scheduledposts/cleanup_test.go` |
| IT-017 | FR-014, FR-020 | AC-016, AC-030, AC-032 | Finalize success and tombstone lifecycle atomically enough for recovery. | Successful fake PDS write and cleanup queue. | Finalize, list management, rerun completion, then advance 30 days. | Item/capacity disappear once; cleanup is immediately eligible; hidden content-free tombstone dedupes replay then expires. | `appview/internal/scheduledposts/store_test.go` |
| IT-018 | NFR-003 | AC-023 | Scan telemetry and errors for privacy canaries. | Canary payload/media/filename/token/signed URL; in-memory logs, metrics, traces. | Exercise create/edit/preview/publish/retry/delete/cleanup failures. | No raw canary, token, object key, signed URL, request body, or full provider response is captured. | `appview/internal/api/scheduled_privacy_test.go`, `appview/internal/scheduledposts/observability_test.go`, `appview/internal/middleware/metrics_test.go`, `appview/internal/observability/error_classifier_test.go` |
| IT-019 | FR-007 | AC-005, AC-011 | Register all routes with standard policies and envelopes. | Construct full server and route-policy registry. | Call each scheduled-post/media method unauthenticated, invalid, foreign, and valid. | `/v1/` camelCase contracts, auth policy, method handling, request ID, and safe envelopes are consistent. | `appview/internal/routes/routes_test.go`, `appview/internal/api/scheduled_post_test.go`, `appview/internal/api/scheduled_media_test.go`, `appview/cmd/appview/server_test.go` |
| IT-020 | FR-011, FR-019 | AC-015, AC-022 | Wire the in-process worker without coupling it to HTTP. | App deps/config and scripted processor. | Start/cancel server lifecycle; simulate backlog and processor error. | Worker starts once, uses bounded batches, drains backlog, survives errors, stops on context cancellation, and remains separately constructible. | `appview/cmd/appview/server_test.go`, `appview/internal/scheduledposts/worker_test.go`, `appview/internal/app/config_test.go`, `appview/internal/app/deps.go` |
| IT-021 | FR-008 | AC-008, AC-027 | Keep Flutter scheduled state account-keyed. | Riverpod container with Alice/Bob repositories and delayed refreshes. | Switch account while Alice refresh is in flight, then pull to refresh Bob. | Alice data never renders/mutates Bob state; Bob receives only refreshed Bob rows. | `app/test/scheduled_posts/scheduled_posts_provider_test.dart` |
| IT-022 | FR-005, FR-016 | AC-005 | Submit complete standard/project schedules without public cache insertion. | Flutter repository fakes, composer images/local bytes, timeline/profile/project provider observers. | Schedule each eligible composer and inject success/failure. | Complete payload reaches scheduled repository; success refreshes schedules only; failure preserves composer; no public cache is changed. | `app/test/scheduled_posts/scheduled_post_submission_test.dart` |
| IT-023 | FR-021 | AC-031, AC-032 | Prove lifecycle transitions do not enqueue notifications. | Seeded notification-store sentinels, static producer-boundary scan, and schedule fixtures. | Retry, enter Needs attention, and publish successfully. | No push delivery, in-app notification event, or new notification preference/category is created. | `appview/internal/scheduledposts/notification_boundary_test.go` |
| IT-024 | BR-004, FR-005, FR-017 | AC-012, AC-013 | Use the private copy rather than the eager PDS blob. | Distinct byte/checksum markers for eager PDS upload and private staging. | Publish the due schedule. | Worker reads/uploads the private marker; record never references the initial eager blob CID. | `appview/internal/scheduledposts/publication_test.go` |
| IT-025 | FR-005, FR-017 | AC-005, AC-030 | Expire staging left unclaimed by schedule-creation failure. | Successful private upload followed by rejected/failed create and injected clock. | Retry create idempotently, then advance 24 hours for truly unclaimed media. | No phantom slot appears; claimed retry is safe; remaining orphan is removed once. | `appview/internal/scheduledposts/cleanup_test.go` |
| IT-026 | FR-006, FR-011, FR-014, FR-020, NFR-001 | AC-006, AC-014, AC-015, AC-017, AC-030 | Verify the scheduled-post migration creates the complete private durable schema. | Apply the migration in `internal/testdb.WithSchema` and inspect the catalog. | Query tables, columns, constraints, references, and indexes; exercise invalid lifecycle/identity rows. | Schedule/media/tombstone tables exist; status/lifecycle and required-reference constraints hold; owner/idempotency uniqueness exists; due, expired-lease, owner/capacity, and cleanup queries have named supporting indexes; and no public post/lexicon schema is changed. | `appview/internal/db/scheduled_posts_migration_test.go` |
| IT-027 | NFR-006 | AC-033 | Emit complete, content-free scheduled-worker operational signals. | Injected clock, deterministic lifecycle fixtures, in-memory metric recorder, and privacy canaries. | Create due/overdue rows; publish, retry, exhaust auth, recover a lease, reject a stale worker, and succeed/fail cleanup. | Required queue depth/age, start latency/duration, attempt/retry/Needs attention, fence/recovery, and cleanup signals are emitted with bounded low-cardinality attributes and no content/account identifiers. | `appview/internal/scheduledposts/observability_test.go`; concrete metric validation in `appview/internal/observability/scheduled_posts_test.go` |

Implementation mapping for IT-006: `TestStoreSerializesMutationsAgainstPublishing`
drives both update-before-claim and claim-before-update/delete commit orders;
the scheduled-post API tests assert the resulting Publishing conflict is
returned through the standard safe envelope before any caller can retry the
mutation.

Implementation mapping for IT-009:
`TestScheduledPublicationRecoversTheSameFrozenRecordAcrossCrashBoundaries`
stops the worker before any upload, after a partial upload, after all media but
before the record lookup/write, and after PDS record success but before local
completion. Every boundary leaves a Publishing row with frozen identity/body,
expires and reclaims its lease through a newly constructed worker, rejects the
stale worker, and proves at most one visible record with the same TID,
createdAt, body, and private-media references.

Implementation mapping for IT-019:
`TestScheduledPostRoutesRequireAuthenticationAndUseDeclaredBodyPolicies` and
the full route-policy coverage tests prove registration, authentication,
methods, and body policies. The scheduled post/media handler suites cover
valid, invalid, and foreign-owner requests plus camelCase responses and safe
request-ID envelopes; `TestNewServer_HTTPMetricsUseRoutePattern` constructs the
server middleware around those registered patterns.

Implementation mapping for IT-020:
`TestScheduledWorkerLoopRunsImmediatelyDrainsAndStopsOnCancellation` proves the
separately constructed worker loop runs immediately, drains a backlog,
continues after a processor error, and stops on cancellation. The scheduled
worker package tests prove its configured batch bound and claim behavior.
`TestLoadConfig_ScheduledPostObjectStoreIsCompleteAndProductionUsesTLS` covers
the production worker/object-store configuration, while `internal/app/deps.go`
is the composition root exercised by the package and server tests.

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | New standard posts publish immediately by default. | FR-004 | AC-001 | Existing `createPostProvider`/repository path runs when When remains Now and returns the normal post result. |
| REG-002 | Image selection continues eager PDS preparation/upload. | FR-017 | AC-012 | Selecting/removing/retrying composer images retains current `POST /v1/blobs/images` behavior before Schedule is submitted. |
| REG-003 | Quote and reply/comment composers still publish immediately but cannot schedule. | FR-002, FR-004 | AC-001, AC-003 | Existing quote/reply payloads pass the immediate route while schedule UI/API rejects them. |
| REG-004 | Project composer validation and complete fields remain intact for Post now. | FR-004 | AC-001 | Existing three-page project submission, image requirement, metadata, languages, facets, and feedback tests remain green. |
| REG-005 | Published posts use normal surfaces, not scheduled history or notifications. | FR-014, FR-021 | AC-032 | A completed schedule is absent from management and visible through normal indexed post surfaces only. |
| REG-006 | Scheduling introduces no notification category, preference, event, or delivery. | FR-021 | AC-031, AC-032 | Static/import and behavioral checks fail if scheduled-post lifecycle writes to notification producers. |
| REG-007 | Existing Settings destinations and typed routing remain available. | BR-003, FR-008 | AC-008 | Adding Scheduled posts preserves current Settings entries and route behavior. |
| REG-008 | Immediate posts update public caches; scheduled posts do not. | FR-004, FR-016 | AC-001, AC-005 | Side-by-side provider test asserts only successful Post now performs optimistic public insertion. |
| REG-009 | Existing immediate post and eager blob wire contracts remain valid. | FR-004, FR-017 | AC-001, AC-012 | Current `/v1/posts` and `/v1/blobs/images` request/response tests remain unchanged and green. |
| REG-010 | Account switching cannot leak private feature state. | FR-008, RULE-004 | AC-008, AC-011 | Existing account-boundary/router tests plus scheduled provider tests show no cross-account rows, counts, previews, or mutations. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Eligible standard schedule | Text, languages, mention/link/tag facets, four ordered images with alt text, whole-minute UTC time. | AT-003, AT-004, IT-001, IT-009, IT-022 |
| TD-002 | Complete project schedule | Craft type, title, pattern, materials, colours, design tags, body, language, facets, required image. | AT-002, AT-007, IT-009, IT-022 |
| TD-003 | Excluded payloads | Quote embed, reply root/parent, comment form, project with quote, malformed kind. | AT-002, UT-001, IT-002, REG-003 |
| TD-004 | Time boundary table | Injected server now plus 4:59, 5:00, non-zero seconds, 27d23h59m, 28d, 28d+1m; DST/travel zones. | AT-003, UT-002, UT-003, IT-002, MAN-001 |
| TD-005 | Lifecycle/capacity set | One row for Scheduled, Publishing, Retrying, Needs attention, published tombstone, deleted; multiple owner DIDs. | AT-005, AT-006, UT-004, IT-003, IT-004 |
| TD-006 | Retry/error table | Timeout, network, 429/5xx-like transient errors; invalid media/payload, mention denial, block denial; deterministic jitter. | AT-010, UT-005, UT-011, IT-011, IT-012 |
| TD-007 | Crash/race barriers | Pauses before/after Publishing commit, after one or more media uploads, before/after PDS record success, local completion, lease expiry, account deletion. | AT-008, AT-011, IT-006, IT-007, IT-009, IT-010, IT-013 |
| TD-008 | Privacy canaries | Unique text, alt text, facet URI/tag, project values, filename, DID/handle, token, object key, signed URL, provider body. | AT-013, UT-015, IT-018 |
| TD-009 | Retention fixtures | Unclaimed upload at 24-hour boundary, replaced/shared objects, Needs attention expiry, published tombstone expiry, injected delete failures. | AT-012, UT-013, UT-020, IT-016, IT-017, IT-025 |
| TD-010 | Session fixtures | Two active owner sessions, signed-out device, expired/revoked last session, foreign active session, terminal account deletion. | AT-010, UT-018, IT-013, MAN-004 |
| TD-011 | Operational signal fixtures | Safe status counts, due/overdue ages, start/completion times, retry exhaustion, auth unavailability, stale fence, lease recovery, cleanup success/failure, and privacy canaries. | IT-027, MAN-006 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-003, NFR-005, RULE-003 | AC-004, AC-024, AC-025 | Real-device picker, timezone, DST, accessibility, and large text. | On iOS and Android, inspect schedule selection near both bounds; change device zone; enable screen reader and largest supported text; exercise validation and delete confirmation. | The instant remains fixed, timezone/offset is understandable, semantics are announced, and all controls remain reachable without clipping. |
| MAN-002 | FR-005, FR-017 | AC-005, AC-012, AC-025 | Real-device image reprocessing and failure recovery. | Select images, observe eager upload, tap Schedule, interrupt/restore the network during private staging and schedule creation, then retry. | Staging begins only after Schedule, progress is clear, composer content/local source survives failure, one schedule is created, and the eager blob is not used as durable media. |
| MAN-003 | BR-001, FR-012, FR-014, NFR-001, NFR-002 | AC-013, AC-015, AC-016, AC-032 | End-to-end closed-app publication against dev AppView/PDS/Tap. | Schedule a text and image/project post, close Flutter, cross due time, restart AppView during a separate run, and inspect PDS then feed/profile after Tap catches up. | Nothing publishes early; publication starts within the healthy target, creates one normal post, management/capacity clear, and no notification/history entry appears. |
| MAN-004 | FR-008, FR-009, FR-018, RULE-005, RULE-008 | AC-009, AC-018, AC-021, AC-027, AC-028, AC-029 | Multi-device edits, manual refresh, and session lifecycle. | Open the same schedule on two devices, save competing edits, sign out one then all sessions, cross due time, sign in again, and pull to refresh. | Last accepted edit wins, stale worker content does not publish, one active session suffices, last sign-out leads to Needs attention, sign-in does not resume it, and refresh reveals current status. |
| MAN-005 | NFR-004 | AC-012, AC-023, AC-030 | Production S3-compatible security and lifecycle configuration. | For the selected managed provider, inspect TLS endpoint, bucket public-access block, platform encryption-at-rest setting, least-privilege service credentials, logs, and lifecycle/cleanup behavior with non-sensitive fixtures. | Anonymous/public access fails, transport and provider encryption are enabled, credentials are scoped, no sensitive identifiers appear, and application cleanup works; no custom encryption layer is required. |
| MAN-006 | NFR-006 | AC-033 | Production overdue/worker/auth/cleanup alert configuration. | Inspect deployed metric names/attributes and alert rules; inject or safely simulate sustained overdue depth, worker/claim errors, exhausted retry/auth failures, and cleanup backlog. | Each sustained condition triggers the intended operational alert with bounded, content-free context; healthy recovery clears it according to policy; no DID, handle, schedule ID, media ID, or member content is exposed. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Acceptance Criteria | Reason | Follow-Up |
|---|---|---|---|---|---|
| GAP-001 | Managed production S3-compatible controls cannot be verified until a provider/bucket is selected. | NFR-004 | AC-012, AC-023, AC-030 | Repository and MinIO tests cannot prove the deployed provider's TLS, encryption-at-rest, public-access, IAM, or retention configuration. | Treat MAN-005 as a blocking production-release gate and record provider evidence during deployment review. |
| GAP-002 | Fake-clock/metric-recorder tests cannot prove real deployment consistently begins work within 60 seconds or that deployed alert delivery is healthy. | NFR-002, NFR-006 | AC-015, AC-016, AC-033 | Polling, scheduling, database load, metrics export, and alert delivery are environmental. | Require IT-008, IT-027, MAN-003, and MAN-006 before release. |
| GAP-003 | The PDS's short-window deletion of unreferenced eager blobs is external behavior. | FR-017 | AC-012 | Craftsky must not depend on a provider-specific cleanup duration, and deterministic deletion may not be exposed by a PDS test harness. | Verify the integration leaves the blob unreferenced, retain the AT Protocol contract as an assumption, and optionally observe cleanup on the dev PDS without making it a publication dependency. |
| GAP-004 | A real process kill at the exact PDS-success/local-failure boundary is difficult to make deterministic. | FR-013, NFR-001 | AC-017 | Automated barrier fakes prove the protocol, but a live PDS crash drill is slower and less repeatable. | Require deterministic IT-010 before implementation approval and include one live recovery drill in pre-release verification if the dev PDS supports record lookup/replacement. |

Risk remains High. The DR-001 through DR-004 revisions are present, but implementation must not begin until the revised requirements and test design pass re-review and receive explicit approval. GAP-001 and the applicable portions of GAP-002 through GAP-004 remain release gates rather than reasons to mislabel coverage as fully automated.

## 10. Out Of Scope

- Scheduling quote posts, comments, replies, reposts, or non-post records.
- Recurrence, bulk scheduling, calendars, templates, approvals, shared/team editing, or queues larger than three.
- Draft autosave or general-purpose cross-device drafts.
- Notifications for upcoming, successful, retrying, or Needs attention schedules beyond the agreed Settings count/status surfaces.
- Scheduled-post history after publication or management-based editing/deletion of published posts.
- Lexicon changes or a public `scheduledAt` field.
- Selecting or load-testing a final managed object-storage vendor during test design.
- Extracting the worker into a separate service in this release.
- Testing Craftsky-driven deletion of unreferenced PDS blobs; the app intentionally performs no such deletion.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-31-scheduled-posts/`
- Risk level: High; document review and explicit approval are required before implementation.
- Recommended first failing test for implementation: `UT-002` in `appview/internal/scheduledposts/validation_test.go`, defining the inclusive five-minute/28-day whole-minute server-time boundary without infrastructure dependencies.
- Suggested test order for implementation:
  1. `UT-001` through `UT-020` for post eligibility, time, capacity, lifecycle, retry, payload, identity, retention, privacy, and Flutter state rules.
  2. `IT-003`, `IT-005` through `IT-007`, `IT-016`, `IT-017`, `IT-025`, and `IT-026` for the migration contract, store invariants, concurrency, claims, and cleanup.
  3. `IT-001`, `IT-002`, `IT-004`, `IT-014`, and `IT-019` for private API/media contracts and ownership.
  4. `IT-008` through `IT-013`, `IT-018`, `IT-020`, `IT-023`, `IT-024`, and `IT-027` for worker publication, recovery, sessions, wiring, notification boundary, observability, and privacy.
  5. `AT-001` through `AT-008`, `AT-014`, `IT-021`, `IT-022`, and `REG-001` through `REG-010` for Flutter workflows and existing behavior.
  6. `AT-009` through `AT-013`, MinIO `IT-015`, full AppView/Flutter suites, then MAN-001 through MAN-006 release checks.
- Commands discovered:
  - `just app-test`
  - `just app-analyze`
  - `cd appview && go test ./...`
  - `just test` with the Compose Postgres running
  - Focused targets after the planned suites exist: `just app-test test/scheduled_posts` and `cd appview && go test ./internal/scheduledposts ./internal/api ./internal/db ./internal/app ./cmd/appview`
- Blocking gaps:
  - None for re-review; DR-001 through DR-004 have been addressed without renumbering existing IDs.
  - Implementation is blocked on the required high-risk re-review and explicit approval.
  - Production release remains blocked on GAP-001 and the applicable live checks in GAP-002 through GAP-004.
