# Acceptance Test Specification: Local Post Drafts And Submit-Time Media Uploads

## 1. Test Strategy

This specification converts the reviewed local-drafts requirements into a test-first Flutter implementation path. The carried-forward risk level is **Medium**. The highest-risk boundaries are durable file updates across interruption, unpublished-media privacy, the removal of every eager-upload path, account fencing, exact pre-submit persistence for existing drafts, and release of screen-awake state on every terminal path.

Coverage is split as follows:

- Unit tests cover pure draft eligibility, manifest/version rules, path and owner validation, update planning, ordering, schedule restoration, transfer selection, retry-reference invalidation, timeout behavior, overlay state, and safe error projection.
- Integration tests use temporary directories, injected filesystem faults, Riverpod containers, widget harnesses, recording repositories/API adapters, fake clocks, and an injected screen-awake service.
- Acceptance scenarios describe the member-visible behavior and are automated through the nearest stable widget/provider/repository seam rather than a separate device end-to-end framework.
- Regression tests protect current standard, quote, reply/comment, project, scheduled-post, routing, media-validation, API-contract, and account-switch behavior.
- Manual checks are limited to real iOS/Android storage lifecycle, foreground screen-sleep behavior, platform accessibility/rendering, and process-kill realism that widget tests cannot prove.

Automation conventions discovered in the repository:

- Flutter tests use `package:flutter_test`, `ProviderContainer.test`, provider overrides, widget fakes, recording messengers/repositories, and `http_mock_adapter` where HTTP shape matters.
- Existing feature tests live under `app/test/<feature>/`; draft-specific tests should live under `app/test/drafts/` while shared composer behavior remains in the existing feed, project, scheduled-post, router, settings, and auth suites.
- Repository tests should inject an explicit temporary root. They must not use a developer's real application-support directory or include private test canaries in screenshots or failure output.
- Time, UUIDs, filesystem operations, upload clients, account leases, and screen-awake behavior must be injectable so failure boundaries are deterministic.
- A network-spy adapter must fail a test immediately if local preparation or draft operations invoke public blob upload or private scheduled staging before explicit submission.
- Device-local draft storage is a deliberate product exception to the repository's broad AppView/Postgres draft guidance. Tests enforce the approved local-only boundary and must not introduce server persistence merely to reconcile that guidance.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-004, AC-006 | AT-001, AT-004, AT-006, IT-008, IT-009 | Acceptance / Integration | Yes |
| BR-002 | AC-002, AC-019 | AT-002, AT-019, IT-001, IT-016 | Acceptance / Integration | Yes |
| BR-003 | AC-009, AC-010, AC-011 | AT-009, AT-010, AT-011, IT-010, IT-011, IT-012 | Acceptance / Integration | Yes |
| BR-004 | AC-013, AC-014 | AT-013, AT-014, IT-013 | Acceptance / Integration | Yes; manual platform check also |
| FR-001 | AC-001 | AT-001, UT-001, IT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-004, AC-006, AC-024 | AT-004, AT-006, AT-024, UT-002, UT-005 | Acceptance / Unit | Yes |
| FR-003 | AC-002, AC-004, AC-025 | AT-002, AT-004, AT-025, UT-003, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-005 | AT-005, UT-014, IT-006, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-006 | AT-006, IT-008, IT-009 | Acceptance / Integration | Yes |
| FR-006 | AC-007, AC-016, AC-017 | AT-007, AT-016, AT-017, UT-006, IT-002, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-008 | AT-008, IT-006 | Acceptance / Integration | Yes |
| FR-008 | AC-012, AC-014 | AT-012, AT-014, IT-014 | Acceptance / Integration | Yes |
| FR-009 | AC-014, AC-015, AC-026 | AT-014, AT-015, AT-026, UT-007, UT-011, IT-011, IT-012, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-009, AC-025 | AT-009, AT-025, UT-008, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-010, AC-026 | AT-010, AT-026, UT-009, UT-010, UT-011, IT-011 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-011, AC-026 | AT-011, AT-026, UT-009, UT-011, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-013 | AT-013, UT-012, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-018 | AT-018, UT-016, IT-005, IT-015 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-015, AC-016 | AT-015, AT-016, UT-002, UT-007, IT-002, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-015, AC-017 | AT-015, AT-017, UT-007, IT-003, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-002, AC-005 | AT-002, AT-005, IT-001, IT-006, IT-007 | Acceptance / Integration | Yes |
| FR-018 | AC-003, AC-025 | AT-003, AT-025, IT-008, IT-009 | Acceptance / Integration | Yes |
| FR-019 | AC-024 | AT-024, UT-005 | Acceptance / Unit | Yes |
| FR-020 | AC-023 | AT-023, IT-005 | Acceptance / Integration | Yes |
| FR-021 | AC-025 | AT-025, UT-008, IT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-022 | AC-027 | AT-027, UT-013, IT-014 | Acceptance / Unit / Integration | Yes |
| FR-023 | AC-026 | AT-026, UT-010, IT-011, IT-012 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-007, AC-016, AC-017 | AT-007, AT-016, AT-017, UT-006, IT-002, IT-003 | Acceptance / Unit / Integration | Yes; manual process-kill check also |
| NFR-002 | AC-019 | AT-019, UT-015, IT-016 | Acceptance / Unit / Integration | Yes |
| NFR-003 | AC-013 | AT-013, IT-013 | Acceptance / Integration | Yes; manual platform check also |
| NFR-004 | AC-020 | AT-020, UT-017, IT-018 | Acceptance / Unit / Integration | Yes; manual responsiveness check also |
| NFR-005 | AC-014, AC-026 | AT-014, AT-026, UT-012, IT-013 | Acceptance / Unit / Integration | Yes; manual platform check also |
| RULE-001 | AC-002, AC-009 | AT-002, AT-009, IT-001, IT-010 | Acceptance / Integration | Yes |
| RULE-002 | AC-001, AC-021 | AT-001, AT-021, UT-001, IT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-003, AC-027 | AT-003, AT-027, UT-013, IT-008, IT-009, IT-014 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-001, AC-006, AC-012 | AT-001, AT-006, AT-012, UT-001, IT-008, IT-009, IT-014 | Acceptance / Unit / Integration | Yes |
| RULE-005 | AC-022 | AT-022, IT-012 | Acceptance / Integration | Yes |
| RULE-006 | AC-008, AC-023 | AT-008, AT-023, IT-005, IT-006 | Acceptance / Integration | Yes |
| RULE-007 | AC-005, AC-007 | AT-005, AT-007, UT-004, IT-006 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Save meaningful incomplete eligible work

Requirement IDs: BR-001, FR-001, RULE-002, RULE-004
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/composer/draft_save_eligibility_test.dart`

```gherkin
Feature: Explicit local drafts
  Scenario Outline: Save a meaningful standard or project draft without publication validation
    Given a new eligible <kind> composer containing a deliberate text, media, alt-text, language, project-field, or scheduling change
    When the member chooses Save draft
    Then one account-owned local draft is saved without network access or publication validation
    And untouched default-only composers do not enable Save draft

    Examples:
      | kind     |
      | standard |
      | project  |
```

### AT-002: Reopen private app-owned media offline

Requirement IDs: BR-002, FR-003, FR-017, RULE-001
Acceptance Criteria: AC-002
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/data/file_draft_repository_test.dart`, `app/test/drafts/composer/draft_composer_hydrator_test.dart`

```gherkin
Feature: Durable local draft media
  Scenario: Reopen after the original picker asset becomes unavailable
    Given a signed-in member saved prepared images into private app-owned draft storage
    And the original picker assets are removed and AppView connectivity is unavailable
    When the app restarts and the member reopens the draft
    Then its prepared previews load from the app-owned copies
    And no original asset, AppView request, PDS request, or scheduled-media request is used
```

### AT-003: Offer explicit close choices without autosaving

Requirement IDs: FR-018, RULE-003
Acceptance Criteria: AC-003
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_composer_sheet_discard_test.dart`, `app/test/projects/widgets/project_composer_discard_test.dart`

```gherkin
Feature: Explicit draft retention
  Scenario: Leave a dirty eligible composer
    Given a dirty eligible composer
    When the member attempts to close it
    Then a new composer offers Save draft, Discard, and Keep editing
    And an existing draft offers Save changes, Discard changes, and Keep editing
    And editing alone performs no durable write while Discard changes preserves the prior snapshot
```

### AT-004: Round-trip the complete editable snapshot

Requirement IDs: BR-001, FR-002, FR-003
Acceptance Criteria: AC-004
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/composer/draft_composer_hydrator_test.dart`

```gherkin
Feature: Draft snapshot fidelity
  Scenario Outline: Reopen every saved field and image in order
    Given a saved <kind> draft containing text, languages, scheduling intent, kind-specific fields, ordered prepared images, and alt text
    When the draft is reopened
    Then every saved value and publishable image is restored in the same editable order
    And the untouched source asset is neither read nor retained

    Examples:
      | kind     |
      | standard |
      | project  |
```

### AT-005: Manage active-account drafts from Settings

Requirement IDs: FR-004, FR-017, RULE-007
Acceptance Criteria: AC-005
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/pages/drafts_page_test.dart`, `app/test/settings/settings_page_test.dart`, `app/test/router/settings_routes_test.dart`

```gherkin
Feature: Draft management
  Scenario: List, open, and delete drafts while offline
    Given the active account has multiple drafts and another account also has drafts
    When Settings > Drafts opens without AppView connectivity
    Then only active-account rows appear newest-updated first with deterministic ties
    And each row shows its thumbnail or draft icon, kind, preview, and last-saved time
    And edit, confirmed delete, empty, error, and retry states work without badge, search, filter, folder, bulk action, or manual sort
```

### AT-006: Open the matching incomplete composer

Requirement IDs: BR-001, FR-002, FR-005, RULE-004
Acceptance Criteria: AC-006
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/pages/drafts_page_test.dart`, `app/test/drafts/composer/draft_composer_hydrator_test.dart`

```gherkin
Feature: Resume a local draft
  Scenario Outline: Open an incomplete draft in its correct composer
    Given an incomplete saved <kind> draft row
    When the member taps the row
    Then the matching composer opens with the stable saved draft ID and all recoverable state
    And publication validation is deferred until Post or Schedule

    Examples:
      | kind     |
      | standard |
      | project  |
```

### AT-007: Update one draft atomically without duplication

Requirement IDs: FR-006, NFR-001, RULE-007
Acceptance Criteria: AC-007
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/data/file_draft_repository_test.dart`, `app/test/drafts/pages/drafts_page_test.dart`

```gherkin
Feature: Safe draft updates
  Scenario: Save changes to an existing draft
    Given an existing draft with unchanged and changed media
    When the member saves the edit
    Then its ID and created time remain unchanged and no duplicate row appears
    And unchanged immutable files are reused while changed media receives new immutable files
    And the manifest switches atomically before old unreferenced files are cleaned
    And updatedAt advances and list ordering follows it
```

### AT-008: Delete deliberately and idempotently

Requirement IDs: FR-007, RULE-006
Acceptance Criteria: AC-008
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/pages/drafts_page_test.dart`, `app/test/drafts/data/file_draft_repository_test.dart`

```gherkin
Feature: Draft deletion
  Scenario: Confirm deletion and handle its result honestly
    Given a saved draft and its unshared media
    When the member confirms deletion
    Then the row, manifest, thumbnail, and unshared media are removed
    And repeating the repository delete is harmless
    But a failed delete remains visible and retryable without a success claim
```

### AT-009: Perform all pre-submit media work locally

Requirement IDs: BR-003, FR-010, RULE-001
Acceptance Criteria: AC-009
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/providers/composer_images_provider_test.dart`, `app/test/drafts/network/draft_network_boundary_test.dart`

```gherkin
Feature: No optimistic media upload
  Scenario: Prepare and edit selected media before submission
    Given a member selects supported media
    When CraftSky inspects, strips metadata, resizes, re-encodes, prepares, reorders, edits alt text, saves, closes, reopens, or discards it
    Then local prepared bytes and early local errors are available
    And recorded traffic contains no public blob upload or private scheduled-media staging request
```

### AT-010: Upload immediate media only at Post or Reply

Requirement IDs: BR-003, FR-011
Acceptance Criteria: AC-010
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/composer/composer_submission_coordinator_test.dart`, `app/test/projects/widgets/project_composer_submit_test.dart`

```gherkin
Feature: Immediate submit-time upload
  Scenario Outline: Materialize current media before creating the post
    Given a valid <kind> composer with locally prepared ordered images
    When the member taps <action>
    Then current prepared media is revalidated and uploaded through POST /v1/blobs/images only after that tap
    And POST /v1/posts is called afterward with the returned references in image order

    Examples:
      | kind     | action |
      | standard | Post   |
      | quote    | Post   |
      | reply    | Reply  |
      | project  | Post   |
```

### AT-011: Stage scheduled media only at Schedule

Requirement IDs: BR-003, FR-012
Acceptance Criteria: AC-011
Priority: Must
Level: Acceptance
Automation Target: `app/test/scheduled_posts/scheduled_post_submission_test.dart`

```gherkin
Feature: Scheduled submit-time staging
  Scenario: Stage private bytes before accepting a schedule
    Given a valid new or edited scheduled composer with locally prepared images
    When the member taps Schedule
    Then current bytes are uploaded to owner-private scheduled staging using stable media identifiers
    And create or update waits for staging success
    And the composer never uploads those images to the PDS blob endpoint
```

### AT-012: Remove an originating draft only after authoritative success

Requirement IDs: FR-008, RULE-004
Acceptance Criteria: AC-012
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/composer/draft_submission_lifecycle_test.dart`

```gherkin
Feature: Draft submission cleanup
  Scenario Outline: Preserve or delete the source draft according to the authoritative result
    Given a composer opened from a local draft
    When <target> submission returns <result>
    Then the source draft is <retention>
    And successful submission is performed once

    Examples:
      | target    | result            | retention                 |
      | immediate | confirmed success | deleted with local media  |
      | scheduled | confirmed success | deleted with local media  |
      | immediate | validation failure| retained                   |
      | scheduled | API failure       | retained                   |
```

### AT-013: Show the exact blocking overlay after validation

Requirement IDs: BR-004, FR-013, NFR-003
Acceptance Criteria: AC-013
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/submission_overlay_test.dart`, existing standard/project/scheduled composer widget suites

```gherkin
Feature: Blocking submission feedback
  Scenario Outline: Block a confirmed valid submission
    Given local validation and any missing-alt confirmation have completed successfully
    When a member starts a valid <target> submission
    Then a full-screen non-dismissible overlay appears before upload or final API work
    And taps, back navigation, duplicate submission, and cancellation are blocked
    And a centered spinner and accessible busy text show exactly <copy>

    Examples:
      | target    | copy                       |
      | immediate | Publishing your post…      |
      | scheduled | Scheduling your post…      |
```

### AT-014: End overlay and screen-awake ownership on every terminal path

Requirement IDs: BR-004, FR-008, FR-009, NFR-005
Acceptance Criteria: AC-014
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/composer/composer_submission_coordinator_test.dart`, `app/test/feed/widgets/submission_overlay_test.dart`

```gherkin
Feature: Submission lifecycle cleanup
  Scenario Outline: Release blocking state on a terminal outcome
    Given a valid submission owns the overlay and screen-awake state
    When it ends through <outcome>
    Then the overlay exits and screen-awake state is disabled exactly once
    And failures preserve editable state and any applicable saved draft

    Examples:
      | outcome                  |
      | confirmed success        |
      | transfer or API failure  |
      | one-minute image timeout |
      | stale account completion |
      | thrown exception         |
      | overlay disposal         |
```

### AT-015: Keep damaged drafts recoverable and private

Requirement IDs: FR-009, FR-015, FR-016
Acceptance Criteria: AC-015
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/data/file_draft_repository_recovery_test.dart`, `app/test/drafts/pages/drafts_page_test.dart`

```gherkin
Feature: Damaged local draft recovery
  Scenario: Load corrupt or incomplete local data safely
    Given healthy drafts plus a corrupt or unsupported manifest or a missing media file
    When Drafts loads and the damaged item is opened
    Then healthy drafts remain available and recoverable authored fields and media order remain
    And missing media shows Image unavailable and can be removed, replaced, or deleted
    And submission is blocked until missing bytes are resolved
    And no private content or path is logged
```

### AT-016: Recover at every atomic file boundary

Requirement IDs: FR-006, FR-015, NFR-001
Acceptance Criteria: AC-016
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/data/file_draft_repository_recovery_test.dart`

```gherkin
Feature: Interrupted draft save recovery
  Scenario Outline: Restart around each update boundary
    Given a prior complete draft and a save with changed media
    When execution is interrupted <boundary>
    And repository startup reconciliation runs
    Then the manifest references only complete immutable media from the prior or new snapshot
    And the usable draft survives while unreferenced temporary or orphan artifacts are cleaned

    Examples:
      | boundary                          |
      | before each changed-media write   |
      | after each changed-media write    |
      | before manifest replacement       |
      | after manifest replacement        |
      | during old-media cleanup          |
```

### AT-017: Preserve the prior version when storage fails

Requirement IDs: FR-016, NFR-001
Acceptance Criteria: AC-017
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/data/file_draft_repository_failure_test.dart`

```gherkin
Feature: Local storage failure
  Scenario Outline: Fail a new or updated draft save safely
    Given a composer and, for an update, a prior complete saved version
    When private storage becomes <condition> during save
    Then the save reports failure and the composer remains open and intact
    And the last good version remains complete
    And no partial replacement is listed

    Examples:
      | condition   |
      | full        |
      | unavailable |
```

### AT-018: Fence operations to the captured account

Requirement IDs: FR-014
Acceptance Criteria: AC-018
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/providers/drafts_account_boundary_test.dart`, `app/test/feed/composer/composer_submission_coordinator_test.dart`

```gherkin
Feature: Account-safe private work
  Scenario Outline: Ignore stale completion after an account change
    Given an active-account <operation> is in flight
    When another account becomes active before completion
    Then the stale result cannot expose or mutate the new account's drafts
    And it cannot publish or schedule as the new account
    And no obsolete success message is shown

    Examples:
      | operation |
      | save      |
      | open      |
      | delete    |
      | submit    |
```

### AT-019: Keep unpublished canaries out of diagnostics

Requirement IDs: BR-002, NFR-002
Acceptance Criteria: AC-019
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/privacy/draft_privacy_test.dart`

```gherkin
Feature: Draft privacy
  Scenario: Exercise local and submission failures with privacy canaries
    Given unique canaries in draft text, project fields, alt text, account data, paths, filenames, and image bytes
    When save, list, open, delete, recovery, upload, and submission paths succeed or fail
    Then no canary appears in logs, analytics, traces, metrics, crash/error output, or model diagnostics
    And content remains only in private local storage until explicit submission
```

### AT-020: Keep maximum-media file work asynchronous

Requirement IDs: NFR-004
Acceptance Criteria: AC-020
Priority: Should
Level: Acceptance
Automation Target: `app/test/drafts/composer/draft_async_state_test.dart`

```gherkin
Feature: Responsive local draft work
  Scenario: Save and open the maximum supported media set
    Given a draft with four maximum-rule images
    When the member saves or opens it through delayed injected file/image work
    Then bounded saving or loading progress is rendered before completion
    And the Flutter UI can continue pumping frames instead of waiting synchronously for the entire operation
```

### AT-021: Exclude quote and reply/comment drafts

Requirement IDs: RULE-002
Acceptance Criteria: AC-021
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_composer_sheet_discard_test.dart`, `app/test/feed/widgets/post_composer_sheet_test.dart`

```gherkin
Feature: First-release draft eligibility
  Scenario Outline: Keep target-bound composers in memory only
    Given a dirty <kind> composer
    When draft actions and close choices are inspected
    Then Save draft is unavailable and no local draft is created
    But submission still uses submit-time media upload and the common blocking overlay where applicable

    Examples:
      | kind    |
      | quote   |
      | reply   |
      | comment |
```

### AT-022: Keep local drafts outside scheduled capacity

Requirement IDs: RULE-005
Acceptance Criteria: AC-022
Priority: Must
Level: Acceptance
Automation Target: `app/test/scheduled_posts/scheduled_post_capacity_test.dart`, `app/test/drafts/composer/draft_submission_lifecycle_test.dart`

```gherkin
Feature: Scheduled capacity boundary
  Scenario: Save locally before scheduling
    Given an account below its server-enforced scheduled capacity
    And any number of local drafts
    When another draft is saved
    Then the scheduled count is unchanged and no capacity is consumed
    And the existing capacity rule is evaluated only when Schedule is tapped
```

### AT-023: Retain until an explicit terminal removal event

Requirement IDs: FR-020, RULE-006
Acceptance Criteria: AC-023
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/data/draft_retention_test.dart`, `app/test/settings/sign_out_tile_test.dart`

```gherkin
Feature: Draft retention
  Scenario Outline: Apply the agreed lifecycle event
    Given an account owns saved local drafts
    When <event> occurs
    Then the drafts are <result> on that installation
    And no server terminal-deletion signal is polled or required to manage local drafts

    Examples:
      | event                                      | result   |
      | ordinary sign-out then same-account sign-in| retained |
      | time passes                                | retained |
      | explicit delete                            | removed  |
      | confirmed publish or schedule success      | removed  |
      | app-data removal                           | removed  |
```

### AT-024: Restore only a still-valid Later time

Requirement IDs: FR-002, FR-019
Acceptance Criteria: AC-024
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/composer/draft_schedule_restoration_test.dart`

```gherkin
Feature: Draft schedule intent
  Scenario Outline: Reopen saved Later intent
    Given a draft saved with a Later time that is <validity> under current scheduling rules
    When the draft reopens
    Then the composer selects <choice>
    And <effect>

    Examples:
      | validity      | choice | effect                                                        |
      | still valid   | Later  | the same instant is preserved                                 |
      | past/invalid  | Now    | an explanation appears with no submission or capacity change |
```

### AT-025: Save only locally ready media and report the save result

Requirement IDs: FR-003, FR-010, FR-018, FR-021
Acceptance Criteria: AC-025
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/composer/draft_media_readiness_test.dart`, standard/project composer widget suites

```gherkin
Feature: Draft media readiness
  Scenario Outline: Gate Save draft or Save changes on local preparation
    Given an eligible composer whose attached image is <state>
    When the member inspects the save action
    Then <result>

    Examples:
      | state      | result                                                        |
      | preparing  | save is unavailable and local progress is visible             |
      | failed     | save is unavailable until retry succeeds or the image is removed |
      | locally ready | save can run; success closes with Draft saved               |
      | save failure  | the composer stays open and intact with safe feedback        |
```

### AT-026: Retry only missing or changed uploads with one-minute image budgets

Requirement IDs: FR-009, FR-011, FR-012, FR-023
Acceptance Criteria: AC-026
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/composer/composer_submission_coordinator_test.dart`, `app/test/scheduled_posts/private_staging_state_test.dart`

```gherkin
Feature: Submit-time media retry
  Scenario Outline: Retry after one image fails or exceeds one minute
    Given some media transfers succeeded before another <failure>
    When the overlay exits and the same open composer retries
    Then immediate submission reuses only unchanged successful in-memory references and transfers missing or changed images
    And closing the composer discards immediate remote references
    And scheduled submission reuses its existing idempotent media identifiers

    Examples:
      | failure                  |
      | returns an error         |
      | exceeds one minute       |
```

### AT-027: Snapshot an existing draft before network submission only

Requirement IDs: FR-022, RULE-003
Acceptance Criteria: AC-027
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/composer/draft_submission_lifecycle_test.dart`

```gherkin
Feature: Submission recovery boundary
  Scenario Outline: Decide whether submission persists a recovery snapshot
    Given a valid <origin> composer
    When the member taps Post or Schedule
    Then <persistence> before any network work
    And confirmed success deletes an originating draft while failure or termination retains its exact attempted version

    Examples:
      | origin                         | persistence                                      |
      | edited existing draft          | the exact validated state is atomically saved    |
      | never-saved in-memory composer | no local recovery draft is created               |
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, RULE-002, RULE-004 | AC-001, AC-021 | Classify eligible kinds and meaningful state without publication validation. | Standard/project/quote/reply kinds; untouched defaults; individual text, media, alt, language, schedule, and project-field changes. | Only top-level standard/project are eligible; every deliberate change qualifies; default-only state does not. | `app/test/drafts/composer/draft_save_eligibility_test.dart` |
| UT-002 | FR-002, FR-015 | AC-004, AC-006, AC-015 | Encode/decode the versioned manifest and reject incompatible/corrupt manifests safely. | Complete standard/project v1 payloads, unknown fields, unsupported future version, malformed JSON. | V1 round-trips exactly; future/corrupt versions produce content-free unavailable errors without destructive rewrite. | `app/test/drafts/models/draft_manifest_test.dart` |
| UT-003 | FR-003 | AC-002, AC-004, AC-025 | Validate prepared-media descriptors and private relative references. | Ordered IDs, MIME, dimensions, byte/checksum metadata, alt text, absolute/traversal/out-of-root paths, duplicate IDs, too many items. | Valid prepared metadata round-trips; unsafe/invalid descriptors are rejected without exposing the path. | `app/test/drafts/models/draft_media_descriptor_test.dart` |
| UT-004 | FR-004, RULE-007 | AC-005, AC-007 | Sort and project management rows deterministically. | Mixed updatedAt values, equal timestamps, standard/project previews, text-only and image drafts. | updatedAt descending with stable-ID tie-break; opening alone does not reorder; row fields and thumbnail/icon choice are bounded. | `app/test/drafts/models/draft_row_model_test.dart` |
| UT-005 | FR-002, FR-019 | AC-024 | Restore saved scheduling intent against an injected current-rule validator. | Valid future, past, and newly invalid instants. | Valid Later remains; invalid Later becomes Now with an explanation and no capacity/submission side effect. | `app/test/drafts/composer/draft_schedule_restoration_test.dart` |
| UT-006 | FR-006, NFR-001 | AC-007, AC-016, AC-017 | Plan an immutable-media update. | Unchanged, replaced, reordered, added, and removed media. | Unchanged files are reused; changed bytes get new names; manifest replacement precedes cleanup; referenced files are never overwritten/deleted. | `app/test/drafts/data/draft_update_plan_test.dart` |
| UT-007 | FR-009, FR-015, FR-016 | AC-014, AC-015, AC-017 | Map storage/recovery failures into safe retryable states. | Full disk, permission/unavailable, corrupt manifest/media, cleanup failure, unknown exception. | Prior state is preserved; errors contain no content/path; damaged items remain unavailable/deletable as specified. | `app/test/drafts/data/draft_storage_error_test.dart` |
| UT-008 | FR-010, FR-021 | AC-009, AC-025 | Drive local image preparation and save readiness. | Preparing, ready, failed, retrying, removed images; reordered images and alt edits. | Selection has no transfer phase; Save is enabled only when every retained image is locally ready. | `app/test/feed/providers/composer_image_state_test.dart`, `composer_images_provider_test.dart` |
| UT-009 | FR-011, FR-012 | AC-010, AC-011 | Select submission materialization by target. | Immediate/scheduled targets with current prepared media and no media. | Immediate selects public blob then post; scheduled selects private staging then schedule; no transfer occurs before invocation. | `app/test/feed/composer/submission_materialization_plan_test.dart` |
| UT-010 | FR-011, FR-023 | AC-026 | Cache and invalidate transient immediate blob references. | Same composer, unchanged/changed bytes, reorder, removal, close, retry. | Only byte-identical same-composer results are reused; changed/missing images upload; close clears every reference; nothing is serializable into a draft. | `app/test/feed/composer/immediate_upload_retry_state_test.dart` |
| UT-011 | FR-009, FR-011, FR-012 | AC-014, AC-026 | Apply an independent one-minute timeout per image. | Fake clock with several sequential images completing just before the deadline or remaining unresolved when 60 seconds is reached. | Each image receives its own one-minute budget; completion before the deadline succeeds, reaching the deadline unresolved times out retryably, and images do not share one aggregate submission deadline. | `app/test/feed/composer/media_upload_timeout_test.dart` |
| UT-012 | FR-013, NFR-005 | AC-013, AC-014 | Own overlay and screen-awake state as one lifecycle. | Start, duplicate start, success, failure, timeout, stale result, exception, dispose. | Enable follows overlay start; interaction is blocked; disable occurs on every terminal/dispose branch and never leaks. | `app/test/feed/composer/submission_overlay_state_test.dart` |
| UT-013 | FR-022, RULE-003 | AC-027 | Decide pre-submit persistence by composer origin. | Edited existing draft, unchanged existing draft, never-saved composer; immediate/scheduled target. | Existing draft saves exact validated state before network; new composer is never persisted merely by submission. | `app/test/drafts/composer/draft_submission_policy_test.dart` |
| UT-014 | FR-004 | AC-005 | Build safe draft list rows. | Long/blank standard text, project title/body, first/missing media, saved times. | Kind, bounded useful preview, local updated time, and thumbnail/icon are produced without reading full media into diagnostics. | `app/test/drafts/models/draft_row_model_test.dart` |
| UT-015 | NFR-002 | AC-019 | Keep models, errors, events, and metric attributes content-free. | Privacy canaries in every sensitive field and path. | `toString`, mapped errors, allowed event properties, and metric attributes contain no canary or stable private identifier. | `app/test/drafts/privacy/draft_privacy_test.dart` |
| UT-016 | FR-014 | AC-018 | Validate captured account-lease ownership. | Matching lease, switched account, signed out, changed session/activation generation. | Only the captured current owner may complete; stale completion yields no mutation or success signal. | `app/test/drafts/providers/drafts_account_boundary_test.dart` |
| UT-017 | NFR-004 | AC-020 | Model bounded async save/open state. | Delayed read, copy, manifest replacement, list, thumbnail decode, and failures. | Loading/saving states are emitted before completion and operations do not require synchronous widget-thread work. | `app/test/drafts/providers/draft_async_state_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-002, FR-003, FR-017, RULE-001 | AC-002, AC-004 | Save and reopen from private app-owned prepared bytes only. | Temporary support root, source file with distinct original/prepared canaries, offline recording clients. | Save, remove source, dispose/recreate repository, list/open. | Complete prepared bytes and metadata reopen; original is not retained/read; no HTTP call occurs. | `app/test/drafts/data/file_draft_repository_test.dart` |
| IT-002 | FR-006, FR-015, NFR-001 | AC-007, AC-016 | Fault-inject every immutable-write, manifest-switch, and cleanup boundary. | Prior complete bundle, changed-media update, filesystem adapter with barriers/failpoints. | Interrupt before/after each boundary, reconstruct repository, reconcile. | Either prior or new complete snapshot is usable; referenced media survives; temporary/orphan files are cleaned safely. | `app/test/drafts/data/file_draft_repository_recovery_test.dart` |
| IT-003 | FR-006, FR-016, NFR-001 | AC-017 | Preserve prior state on full/unavailable storage. | New and existing drafts; filesystem adapter failing after partial byte writes or before rename. | Attempt save/update and reload. | Save fails safely; composer-facing error is actionable; prior manifest/media remain; no partial row appears. | `app/test/drafts/data/file_draft_repository_failure_test.dart` |
| IT-004 | FR-015, FR-016 | AC-015, AC-016 | Reconcile corrupt/newer manifests, missing media, unsafe paths, and orphan files. | Healthy and damaged account bundles in one temporary root. | Start repository, list, open, remove/replace damaged media, delete. | Healthy drafts stay visible; damaged row is safe and deletable; Image unavailable preserves authored metadata; unsafe paths are never followed or logged. | `app/test/drafts/data/file_draft_repository_recovery_test.dart` |
| IT-005 | FR-014, FR-020, RULE-006 | AC-018, AC-023 | Enforce account namespaces and local lifecycle retention. | Alice/Bob roots, account leases, delayed operations, recording network clients, and an explicit local purge harness. | Switch/sign out/re-sign in, guess IDs, complete stale calls, explicitly delete, and remove app data. | Accounts never cross-read/write; ordinary sign-out retains; explicit local deletion/app-data removal purges the intended files; no terminal-deletion API is polled; stale calls have no success side effect. | `app/test/drafts/data/draft_retention_test.dart`, `app/test/drafts/providers/drafts_account_boundary_test.dart` |
| IT-006 | FR-004, FR-007, RULE-007 | AC-005, AC-007, AC-008 | Drive newest-first list, refresh, edit, and confirmed deletion through providers/page. | Account-keyed fake repository with mixed rows, delayed errors, equal timestamps, and delete failure. | Open/refresh/edit/delete/retry; switch account. | Correct rows and states render; unchanged open does not reorder; deletion claims success only after repository success; foreign data never renders. | `app/test/drafts/providers/drafts_provider_test.dart`, `app/test/drafts/pages/drafts_page_test.dart` |
| IT-007 | FR-004, FR-017 | AC-005 | Register the localized Settings entry and typed root-navigator route. | Production router widget harness. | Tap Drafts from Settings, navigate back, exercise deep location. | `/profile/settings/drafts` opens the management page on the expected navigator; other Settings routes remain intact. | `app/test/settings/settings_page_test.dart`, `app/test/router/settings_routes_test.dart` |
| IT-008 | BR-001, FR-001, FR-005, FR-018, FR-021, RULE-002, RULE-003, RULE-004 | AC-001, AC-003, AC-004, AC-006, AC-025 | Save, close, reopen, and update a standard draft. | Post composer widget with fake draft repository, local-image gate, recording messenger, hydrator. | Make incomplete changes; exercise app-bar and back choices; prepare/fail/retry media; save/reopen/update/discard changes. | Eligibility, exact choices, readiness gating, complete hydration, stable update, `Draft saved`, and failure preservation match requirements without network. | `app/test/drafts/composer/standard_draft_composer_test.dart`, `app/test/feed/widgets/post_composer_sheet_discard_test.dart` |
| IT-009 | BR-001, FR-001, FR-005, FR-018, FR-021, RULE-002, RULE-003, RULE-004 | AC-001, AC-003, AC-004, AC-006, AC-025 | Save, close, reopen, and update a project draft. | Project composer widget with complete/partial field variants, fake draft repository, local media gate. | Change each project field/default, save incomplete state, reopen across pages, update/discard/fail. | Meaningful project changes qualify; untouched defaults do not; all fields/media hydrate; save result and prior-version behavior are correct. | `app/test/drafts/composer/project_draft_composer_test.dart`, existing project composer suites |
| IT-010 | BR-003, FR-010, RULE-001 | AC-009, AC-021 | Prove no eager network path remains in any composer operation. | Recording public blob and private staging clients set to fail on unexpected calls; standard/quote/reply/project/new-schedule/scheduled-edit harnesses. | Select/prepare/reorder/edit alt/save/open/discard/close without Post/Reply/Schedule. | Zero blob/staging calls; quote/reply expose no draft action; local readiness and preview behavior remain. | `app/test/drafts/network/draft_network_boundary_test.dart`, `app/test/feed/providers/composer_images_provider_test.dart` |
| IT-011 | BR-003, FR-009, FR-011, FR-023 | AC-010, AC-014, AC-026 | Orchestrate immediate ordered transfer, create, partial retry, change invalidation, and timeout. | Recording upload/post clients, fake per-image clock, four local-ready images, injected overlay/wakelock. | Submit; fail/timeout image N; retry unchanged; change one image; close. | Upload starts only at submit; create waits and receives ordered refs; retry transfers only missing/changed bytes; close clears refs; each image owns a one-minute timeout; failures preserve state. | `app/test/feed/composer/composer_submission_coordinator_test.dart` |
| IT-012 | BR-003, FR-009, FR-012, FR-023, RULE-005 | AC-011, AC-022, AC-026 | Orchestrate scheduled staging/create-update and idempotent retry. | Existing scheduled repository/materializer fakes, capacity fixtures, fake per-image clock, new/edit draft variants. | Save locally; tap Schedule; fail/timeout staging or create; retry. | Saving consumes no capacity; staging begins only on Schedule; no PDS upload occurs; create/update waits; media IDs remain idempotent; failure preserves composer/draft. | `app/test/scheduled_posts/scheduled_post_submission_test.dart`, `private_staging_state_test.dart`, `scheduled_post_capacity_test.dart` |
| IT-013 | BR-004, FR-009, FR-013, NFR-003, NFR-005 | AC-013, AC-014, AC-026 | Exercise modal overlay and screen-awake lifecycle across all composer targets/outcomes. | Widget harness with gated validation, alt confirmation, upload/API completers, fake navigator, semantics, recording wakelock. | Start standard/quote/reply/project/scheduled flows; attempt taps/back; complete every terminal/dispose branch. | Validation precedes overlay; exact copy and busy semantics render; UI/back/duplicates are blocked; wakelock enable/disable is scoped and failure restores editing. | `app/test/feed/widgets/submission_overlay_test.dart`, composer widget suites |
| IT-014 | FR-008, FR-022, RULE-003, RULE-004 | AC-012, AC-027 | Snapshot existing draft before network, and never auto-create a new-composer draft. | Recording repository and network clients with ordered event log; immediate/scheduled success, failure, and termination barriers. | Submit edited existing draft and never-saved composer. | Existing event order is validate, overlay, atomic save, transfer, final API, then delete on success only; failure leaves exact attempted snapshot; never-saved submit performs no draft write. | `app/test/drafts/composer/draft_submission_lifecycle_test.dart` |
| IT-015 | FR-014 | AC-018 | Fence delayed local and remote work through active-account providers. | Alice/Bob repositories, account activation coordinator, delayed save/open/delete/upload/final API completions. | Switch account at each delay boundary. | Alice completion never renders/mutates Bob state, reports Bob success, or continues publication under Bob's lease. | `app/test/drafts/providers/drafts_account_boundary_test.dart`, `app/test/router/account_switch_routing_test.dart` |
| IT-016 | BR-002, NFR-002 | AC-019 | Scan diagnostics and telemetry with end-to-end privacy canaries. | In-memory logger/event/metric/error recorders and canaries in manifests, media, account, path, request failure, and model strings. | Run save/list/open/recovery/delete and immediate/scheduled failures. | No canary escapes private temporary storage; only approved coarse content-free attributes appear. | `app/test/drafts/privacy/draft_privacy_test.dart` |
| IT-017 | FR-011, FR-012 | AC-010, AC-011 | Preserve existing AppView wire contracts while moving invocation timing. | `http_mock_adapter` for `/v1/blobs/images`, `/v1/posts`, and existing scheduled-media/schedule routes. | Invoke clients through the submission coordinator with known prepared bytes. | Existing method, path, headers, camelCase body, error mapping, and ordered blob/scheduled payload shapes remain unchanged; no new endpoint is called. | `app/test/feed/data/post_api_client_test.dart`, new `app/test/scheduled_posts/scheduled_post_api_client_test.dart` |
| IT-018 | NFR-004 | AC-020 | Render bounded progress while maximum-media local I/O is delayed. | Widget/provider harness with four maximum-rule prepared images and gated async repository/image operations. | Save and reopen while pumping frames and interacting with non-blocked progress semantics. | Loading/saving feedback appears before completion; frames continue; final state is correct without synchronous whole-operation blocking. | `app/test/drafts/composer/draft_async_state_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Standard, quote, reply/comment, and project posts retain current validation, payloads, facets, languages, media order, and success feedback. | FR-010, FR-011, FR-013 | AC-009, AC-010, AC-013, AC-021 | Existing composer suites remain green with the only media timing change being local preparation at selection and transfer at submit. |
| REG-002 | Current image count, supported type, original/prepared size, orientation, dimensions, metadata stripping, and alt-text rules remain authoritative. | FR-003, FR-010, RULE-004 | AC-004, AC-009, AC-012, AC-025 | Existing media service/state/provider tests plus draft reopen/submission revalidation cover all boundaries and four-image ordering. |
| REG-003 | Existing scheduled posts still hydrate, edit, reschedule, publish now, stage privately, retry idempotently, and obey capacity. | FR-012, FR-019, FR-023, RULE-005 | AC-011, AC-022, AC-024, AC-026 | All `app/test/scheduled_posts/` suites remain green after the shared-media refactor. |
| REG-004 | Settings and typed root-navigator routes remain available. | FR-004, FR-017 | AC-005 | Existing Settings destinations plus Drafts route open/back/deep-link checks remain green. |
| REG-005 | Account changes continue to protect unsaved composers and account-bound feature state. | FR-014 | AC-003, AC-018 | Existing unsaved-work guard, account boundary, activation coordinator, and router account-switch suites remain green alongside draft lease tests. |
| REG-006 | Immediate post and scheduled API contracts remain unchanged. | FR-011, FR-012 | AC-010, AC-011 | Existing API-client request/response tests assert no new route or combined multipart contract is introduced. |
| REG-007 | Quote and reply/comment composers do not gain durable drafts. | RULE-002, RULE-003 | AC-003, AC-021, AC-027 | Widget/provider checks assert no save action or repository write while submission overlay and submit-time upload still apply. |
| REG-008 | Local drafts do not alter public caches, scheduled counts, or notification behavior merely through local operations. | RULE-001, RULE-005 | AC-002, AC-009, AC-022 | Provider observers stay unchanged through save/open/update/delete; only authoritative publish/schedule paths perform their current refreshes. |
| REG-009 | File-backed storage remains the sole runtime draft persistence choice. | FR-003, NFR-001 | AC-002, AC-016 | Dependency/config and repository composition checks fail if drafts use SharedPreferences, cache/temp directories, or runtime SQLite. |
| REG-010 | Generated providers/routes/localizations and static analysis remain clean. | FR-004, FR-013 | AC-005, AC-013 | Regenerate affected code, run focused/full Flutter tests, `just app-analyze`, formatter, and `git diff --check`. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Complete standard draft | Stable ID/owner/timestamps, text, languages, Now/Later variants, four ordered prepared JPEG/PNG images with distinct bytes, dimensions, filenames, and alt text. | AT-002, AT-004, AT-007, IT-001, IT-002, IT-008, IT-011 |
| TD-002 | Complete and incomplete project drafts | Every project field and default, changed/default-only variants, multi-page content, prepared images, languages, scheduling intent. | AT-001, AT-004, AT-006, IT-009, IT-014 |
| TD-003 | Ineligible composer set | Quote, reply, comment, target metadata, locally ready media. | AT-010, AT-021, UT-001, IT-010, REG-007 |
| TD-004 | Manifest and path corruption corpus | Valid v1, malformed JSON, future version, missing manifest/media, duplicate media IDs, excessive count, absolute/traversal/out-of-root references, truncated bytes, orphan/temp files. | AT-015, AT-016, UT-002, UT-003, IT-002, IT-004 |
| TD-005 | Atomic update/failure barriers | Before/after each media write, manifest temp write/fsync/rename as implemented, cleanup, full disk, permission unavailable, cleanup failure. | AT-007, AT-016, AT-017, UT-006, IT-002, IT-003 |
| TD-006 | Submission result table | Validation/alt-confirm failure, upload success/failure/59s/60s/>60s, API success/failure/ambiguous completion, thrown exception, stale lease, route disposal. | AT-012, AT-014, AT-026, AT-027, IT-011, IT-012, IT-013, IT-014 |
| TD-007 | Account lifecycle fixtures | Alice/Bob namespaces, matching/stale leases, ordinary sign-out/re-sign-in, explicit local deletion, app-data removal, guessed IDs, delayed completions, and recording clients proving no terminal-deletion poll. | AT-005, AT-018, AT-023, UT-016, IT-005, IT-015 |
| TD-008 | Schedule intent and capacity boundaries | Valid future time, past time, newly invalid time, counts below/at three, existing schedule edit. | AT-011, AT-022, AT-024, UT-005, IT-012, REG-003 |
| TD-009 | Privacy canaries | Unique text, project values, alt text, account value, absolute/relative paths, filenames, media bytes, draft/media IDs, exact time, API failure body. | AT-015, AT-019, UT-015, IT-004, IT-016 |
| TD-010 | Async and accessibility fixture | Four maximum-rule images, gated async phases, light/dark themes, largest supported text scale, screen-reader semantics, motion-independent status. | AT-013, AT-020, IT-013, IT-018, MAN-002, MAN-003 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-003, FR-017, NFR-001 | AC-002, AC-004, AC-016, AC-020 | Real iOS and Android persistence, source loss, offline restart, and interruption. | On each platform, save standard/project drafts with four images; remove or revoke the source photos; force-stop/restart offline; edit/save while force-stopping at several observed phases; reopen and delete. | Prepared app-owned media survives source loss/restart; no network is required; no half-updated draft is exposed; deletion removes local content. |
| MAN-002 | BR-004, FR-013, NFR-005 | AC-013, AC-014, AC-026 | Actual foreground screen-sleep prevention and release. | On physical iOS and Android devices with a short auto-lock setting, throttle a multi-image immediate and scheduled submission beyond the lock interval; exercise success, network failure, one-minute timeout, background/foreground, and leaving the overlay. | Device remains awake only while the visible overlay owns submission and returns to normal auto-lock behavior after every terminal/dispose path. |
| MAN-003 | NFR-003, NFR-004 | AC-013, AC-020, AC-025 | Accessibility, themes, text scale, and maximum-media responsiveness. | On both platforms, use screen reader, light/dark themes, largest supported text, reduced motion where available, four images, preparation failure/retry, save/open progress, and both overlay messages. | Status is announced without relying on animation, exact copy remains readable, controls and recovery actions remain reachable, and no prolonged UI freeze is observed. |
| MAN-004 | FR-020, RULE-006 | AC-023 | Installation lifecycle and local retention boundaries. | Save drafts, sign out/in, wait across date changes, explicitly delete individual drafts, then clear app data on a disposable installation and inspect Drafts/app-private storage through supported development tooling. | Sign-out and time retain drafts; explicit deletion removes the selected bundle; app-data removal clears all local drafts; no expiry, count limit, or remote terminal-deletion dependency silently removes work. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Acceptance Criteria | Reason | Follow-Up |
|---|---|---|---|---|---|
| GAP-001 | Fault-injected repository tests cannot prove every mobile filesystem/process-kill timing or platform atomic-rename guarantee. | FR-006, FR-015, NFR-001 | AC-007, AC-016, AC-017 | Deterministic adapters prove operation ordering and recovery policy, but a real OS can terminate between lower-level writes. | Require IT-002/IT-003 and MAN-001 before release; document the chosen directory and atomic replacement primitive during coding review. |
| GAP-002 | Widget tests cannot prove the operating system actually prevents sleep or restores auto-lock. | BR-004, NFR-005 | AC-014, AC-026 | Injected services prove ownership calls, not platform power behavior. | Require IT-013 and physical-device MAN-002 on iOS and Android before release. |
| GAP-003 | Pumping frames with delayed fakes is not a quantitative jank benchmark on release hardware. | NFR-004 | AC-020 | Widget tests detect accidental synchronous waiting but not device-specific decode/copy latency. | Treat IT-018 plus MAN-003 with four maximum-rule images on representative devices as the release evidence; profile if visible stalls appear. |
| GAP-004 | Normal OS backup/restore behavior varies by platform and device policy and is not a CraftSky recovery guarantee. | FR-020, RULE-006 | AC-023 | Requirements allow normal private app-data backup but explicitly exclude cross-device sync/recovery promises. | Do not make backup restore a pass/fail acceptance gate; verify only local retention, explicit deletion, successful-submission cleanup, and app-data removal owned by CraftSky. |

No blocking test-design gap is identified. The gaps above are release-verification boundaries, not unresolved product decisions.

## 10. Out Of Scope

- Server-side, PDS-backed, or cross-device draft storage and any sync/conflict tests.
- Device-visible propagation or polling for terminal account deletion elsewhere; the first release makes no remote purge guarantee for local files.
- Quote, reply, or comment draft persistence beyond verifying that it is absent.
- Background autosave, per-keystroke persistence, or crash recovery for never-saved composers.
- New AppView endpoints, database migrations, lexicon changes, combined multipart posting, scheduled-worker redesign, or PDS orphan-blob deletion.
- Draft search, filters, folders, tags, badges, bulk actions, manual sorting, expiry, quotas, or count limits.
- Video, audio, editing/cropping originals, and media-rule changes.
- Web and desktop persistence or screen-awake behavior.
- Custom encryption, passcode, biometric gating, and guaranteed backup restoration.
- Submission cancellation, per-image progress copy, richer animation, or background completion after suspension/termination.
- Exact filesystem class names, manifest field names beyond required semantics, and coordinator/provider class names; those belong in coding planning.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-08-03-local-post-drafts/`
- Risk level: Medium
- Review recommendation: Move to document review before coding planning because atomic file recovery, account fencing, and the shared submission lifecycle cross several existing composer seams.
- Recommended first failing test for implementation: update `app/test/feed/providers/composer_images_provider_test.dart` so selecting a valid image reaches a locally-ready state while a recording blob client receives zero calls. This creates the no-optimistic-upload seam on which durable drafts and the shared submission coordinator depend.
- Suggested test order for implementation:
  1. UT-008 and IT-010: split local media preparation from transfer and prove no pre-submit network calls.
  2. UT-001–UT-007 and IT-001–IT-004: define draft models, safe paths, file repository, atomic updates, and recovery.
  3. UT-004, UT-014, IT-005–IT-007: account-scoped providers, management rows, Settings, and routing.
  4. IT-008–IT-009 plus AT-001–AT-008 and AT-024–AT-025: standard/project save, close, hydration, editing, readiness, and schedule intent.
  5. UT-009–UT-013 and IT-011–IT-014: shared immediate/scheduled submission, retry timeout, overlay, wakelock, and originating-draft lifecycle.
  6. UT-015–UT-017 and IT-015–IT-018: account races, privacy, async behavior, and API-contract protection.
  7. REG-001–REG-010, full app verification, then MAN-001–MAN-004.
- Commands discovered:
  - Focused draft suite: `just app-test test/drafts`
  - Focused current media suite: `just app-test test/feed/providers/composer_images_provider_test.dart test/feed/media/composer_image_media_service_test.dart`
  - Focused composer/scheduled suites: `just app-test test/feed/widgets test/projects/widgets test/scheduled_posts`
  - Full Flutter suite: `just app-test`
  - Static analysis: `just app-analyze`
  - Generated code when providers/routes/mappers change: `cd app && dart run build_runner build --delete-conflicting-outputs`
  - Format changed Dart sources during implementation: `cd app && dart format <changed-paths>`
  - Whitespace validation: `git diff --check`
- Blocking gaps: None.
