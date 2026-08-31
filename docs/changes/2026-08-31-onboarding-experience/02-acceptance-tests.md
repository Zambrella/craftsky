# Acceptance Test Specification: Onboarding Experience

## 1. Test Strategy
Use a test-first split aligned with the existing repository:

- Flutter widget acceptance tests exercise the full three-step flow, form state, sequential navigation, accessibility, responsive layouts, shared Instagram sections, and router outcomes using Riverpod overrides and existing fake repositories.
- Dart unit/provider tests cover action-state derivation, bounded prefill retries, optimistic completion, in-memory retry ownership, and session-scoped drafts with fake time where timing is involved.
- Go integration tests cover the authenticated `/v1/` completion contract, private Postgres persistence, idempotency, per-DID isolation, migration behavior, route policy, owner-deletion cleanup, and OAuth-time Bluesky profile projection before Flutter handoff.
- Existing profile and Instagram suites provide regression coverage for behavior moved into shared widgets.
- Manual checks are limited to real operating-system gallery behavior and visual/touch behavior that widget tests cannot fully represent.

Risk remains **Medium**. The highest-risk paths are atomic profile preservation, optimistic completion reconciliation, startup gating, account isolation, private cleanup, canonical OAuth/Tap projection idempotency, and regression of the established Instagram settings page.

## 2. Requirement Coverage Matrix
| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, AT-002 | Acceptance | Yes |
| BR-002 | AC-003 | AT-007 | Acceptance | Yes |
| FR-001 | AC-001 | AT-001 | Acceptance | Yes |
| FR-002 | AC-004, AC-005 | AT-003 | Acceptance | Yes |
| FR-003 | AC-006 | AT-004, UT-001 | Acceptance / Unit | Yes |
| FR-004 | AC-007 | AT-005 | Acceptance | Yes |
| FR-005 | AC-008, AC-009 | AT-006, AT-017, IT-003 | Acceptance / Integration | Yes |
| FR-006 | AC-010 | AT-002, UT-001 | Acceptance / Unit | Yes |
| FR-007 | AC-011 | AT-006 | Acceptance | Yes |
| FR-008 | AC-012, AC-013, AC-024, AC-025 | AT-009, AT-010, AT-011, REG-002 | Acceptance / Regression | Yes |
| FR-009 | AC-014, AC-026 | AT-012, AT-018 | Acceptance | Yes |
| FR-010 | AC-003, AC-015, AC-027 | AT-006, AT-007 | Acceptance | Yes |
| FR-011 | AC-016, AC-028 | AT-008, UT-005 | Acceptance / Unit | Yes |
| FR-012 | AC-017 | AT-001, UT-002 | Acceptance / Unit | Yes |
| FR-013 | AC-018 | AT-009 | Acceptance | Yes |
| FR-014 | AC-004, AC-023 | AT-003, IT-004 | Acceptance / Integration | Yes |
| FR-015 | AC-024 | AT-009, REG-002 | Acceptance / Regression | Yes |
| FR-016 | AC-025 | AT-011 | Acceptance | Yes |
| FR-017 | AC-029 | AT-001, AT-008 | Acceptance | Yes |
| FR-018 | AC-030, AC-031 | AT-013, IT-001, IT-002 | Acceptance / Integration | Yes |
| FR-019 | AC-032, AC-033 | AT-018, UT-004 | Acceptance / Unit | Yes |
| FR-020 | AC-034 | AT-013, UT-006 | Acceptance / Unit | Yes |
| FR-021 | AC-035, AC-036 | AT-016, UT-003 | Acceptance / Unit | Yes |
| FR-022 | AC-027 | AT-006, UT-001 | Acceptance / Unit | Yes |
| FR-023 | AC-037 | AT-003, MAN-001 | Acceptance / Manual | Partial |
| FR-024 | AC-041, AC-042 | AT-019, IT-009 | Acceptance / Integration | Yes |
| NFR-001 | AC-019 | AT-014, MAN-002 | Acceptance / Manual | Partial |
| NFR-002 | AC-020 | AT-014 | Acceptance | Yes |
| NFR-003 | AC-021 | AT-015, IT-006 | Acceptance / Integration | Yes |
| NFR-004 | AC-041, AC-043 | AT-019, IT-009, REG-008 | Acceptance / Integration / Regression | Yes |
| RULE-001 | AC-009 | AT-017, IT-003, REG-001 | Acceptance / Integration / Regression | Yes |
| RULE-002 | AC-002, AC-022 | AT-002 | Acceptance | Yes |
| RULE-003 | AC-015, AC-021, AC-031 | AT-007, AT-015, IT-001, IT-006 | Acceptance / Integration | Yes |
| RULE-004 | AC-038 | AT-013, IT-001 | Acceptance / Integration | Yes |
| RULE-005 | AC-039 | AT-008, UT-005 | Acceptance / Unit | Yes |
| RULE-006 | AC-040 | AT-017, REG-001 | Acceptance / Regression | Yes |
| RULE-007 | AC-042, AC-043 | AT-019, UT-009, IT-009, REG-008 | Acceptance / Unit / Integration / Regression | Yes |

## 3. Acceptance Scenarios
### AT-001: Incomplete Account Enters Sequential Three-Step Flow
Requirement IDs: BR-001, FR-001, FR-012, FR-017
Acceptance Criteria: AC-001, AC-017, AC-029
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_page_test.dart`

```gherkin
Feature: Onboarding structure
  Scenario: An incomplete account starts onboarding
    Given the active authenticated DID has incomplete server onboarding state
    And its profile is available
    When routing settles
    Then the full-screen profile identity step is shown
    And the page announces "Step 1 of 3" with a non-interactive progress bar
    When the member proceeds through clean steps
    Then crafts is step 2 and Instagram is step 3
    And Back returns to the previous step from steps 2 and 3
    And Back on step 1 does not leave or complete onboarding
```

### AT-002: All Optional Values May Remain Empty
Requirement IDs: BR-001, FR-006, RULE-002
Acceptance Criteria: AC-002, AC-010, AC-022
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_page_test.dart`

```gherkin
Feature: Optional onboarding
  Scenario: Member completes onboarding without supplying data
    Given the active profile has no avatar, display name, bio, or known crafts
    And Instagram is not linked
    When the member uses Next on each clean intermediate step
    Then no profile write is issued
    And progression remains enabled
    When the member activates Finish
    Then the main app opens without requiring Instagram
```

### AT-003: Bluesky Profile Is Pre-Filled And Editable
Requirement IDs: FR-002, FR-014, FR-023
Acceptance Criteria: AC-004, AC-005, AC-023, AC-037
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_profile_step_test.dart`

```gherkin
Feature: Profile identity setup
  Scenario: Existing Bluesky identity is shown
    Given GET /v1/profiles/me returns an avatar, display name, bio, and handle
    When step 1 loads
    Then the avatar, display name, bio, and read-only @handle are shown
    And no direct PDS read is attempted by Flutter
    When the member edits a field or selects a gallery image
    Then the step becomes dirty and its primary action becomes Save & next
    And camera, avatar removal, banner, account switching, and sign-out controls are absent
```

### AT-004: Invalid Or Incomplete Profile Draft Cannot Save
Requirement IDs: FR-003
Acceptance Criteria: AC-006
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_profile_step_test.dart`

```gherkin
Feature: Profile validation
  Scenario Outline: Profile save is blocked
    Given step 1 has a dirty profile draft
    And the draft has <condition>
    Then Save & next is disabled
    And relevant validation or upload feedback is visible
    Examples:
      | condition |
      | display name beyond the existing limit |
      | bio beyond the existing limit |
      | avatar upload still running |
      | avatar upload failed |
```

### AT-005: Existing Crafts Are Selectable And Optional
Requirement IDs: FR-004
Acceptance Criteria: AC-007
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_crafts_step_test.dart`

```gherkin
Feature: Craft preferences
  Scenario: Member changes craft selections
    Given the profile contains known crafts sewing and quilting
    When step 2 loads
    Then those localized chips are selected
    And every catalog craft is available in stable catalog order
    When the member toggles crafts including clearing all known crafts
    Then chip semantics and dirty state match the current selection
```

### AT-006: Save-And-Advance Is Single-Flight And Recoverable
Requirement IDs: FR-005, FR-007, FR-010, FR-022
Acceptance Criteria: AC-008, AC-011, AC-027
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_save_navigation_test.dart`

```gherkin
Feature: Step-scoped persistence
  Scenario: Dirty step saves successfully
    Given a dirty valid profile or craft step
    When the member activates Save & next
    Then exactly one profile save is submitted
    And Save, Skip, and Back are disabled while it is in flight
    And the next step opens only after success

  Scenario: Dirty step save fails
    Given a dirty valid step whose save will fail
    When the member activates Save & next
    Then the current step and draft remain visible
    And an understandable error is shown
    And Skip, Back, and retry become available again
    And onboarding is not completed
```

### AT-007: Skip Is Immediate, Isolated, And Discards Drafts
Requirement IDs: BR-002, FR-010, RULE-003
Acceptance Criteria: AC-003, AC-015
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_completion_test.dart`

```gherkin
Feature: Skip onboarding
  Scenario: Member skips from any idle step
    Given account A is on a step with an unsaved draft
    And account B has independent incomplete state
    When account A activates Skip
    Then no confirmation is shown
    And account A enters the main app immediately
    And the unsaved draft is not written
    And previously successful writes remain
    And account B is unchanged
```

### AT-008: Drafts Survive Navigation But Not Restart
Requirement IDs: FR-011, FR-017, RULE-005
Acceptance Criteria: AC-016, AC-028, AC-029, AC-039
Priority: Should
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_draft_navigation_test.dart`

```gherkin
Feature: Session-scoped onboarding drafts
  Scenario: Draft survives step navigation
    Given the member has an unsaved craft draft
    When they navigate Back and later return sequentially
    Then the draft is restored

  Scenario: Draft does not survive app reconstruction
    Given an incomplete account closed onboarding on step 2 or 3
    When a new app process is simulated
    Then onboarding starts on step 1
    And persisted profile and Instagram data are shown
    And no unsaved draft is restored
```

### AT-009: Instagram Linking And Both Import Paths Are Shared
Requirement IDs: FR-008, FR-013, FR-015
Acceptance Criteria: AC-012, AC-018, AC-024
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_instagram_step_test.dart`

```gherkin
Feature: Instagram onboarding
  Scenario: Unlinked account sees scoped Instagram tools
    Given Instagram verification is available and the active account is unlinked
    When step 3 loads
    Then linking and discoverability controls are shown with optional/privacy copy
    And export import and manual handle import are available when eligible
    And import history and revocation are absent
    And existing parser, validation, success, and error behavior is preserved
```

### AT-010: Linked And Inactive Instagram States Are Reused
Requirement IDs: FR-008
Acceptance Criteria: AC-013
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_instagram_step_test.dart`

```gherkin
Feature: Existing Instagram account
  Scenario Outline: Existing account state is shown
    Given the Instagram account is <state>
    When onboarding step 3 and Instagram settings render that state
    Then both use the shared account section
    And the applicable linked username, discoverability, or reactivation controls match
    Examples:
      | state |
      | active |
      | membership inactive |
      | reactivation required |
```

### AT-011: Suggestions Stay Inline
Requirement IDs: FR-008, FR-016
Acceptance Criteria: AC-025
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_instagram_step_test.dart`

```gherkin
Feature: Instagram suggestions
  Scenario: Member manages suggestions without leaving onboarding
    Given step 3 has suggestions and another page cursor
    When the member follows, dismisses, or loads more
    Then the existing inline action behavior runs
    And the list updates consistently
    When the member taps a suggestion row
    Then no profile route opens
```

### AT-012: Instagram Work Never Blocks Finish
Requirement IDs: FR-009
Acceptance Criteria: AC-014, AC-026
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_instagram_step_test.dart`

```gherkin
Feature: Finish optional Instagram setup
  Scenario Outline: Finish remains available during Instagram activity
    Given step 3 has <activity> in progress
    When the member activates Finish
    Then current-process onboarding state becomes complete
    And the main app opens
    And already accepted server work is not treated as cancelled
    Examples:
      | activity |
      | verification |
      | import upload |
      | suggestion action |
```

### AT-013: Server Status Controls Cold-Start Routing
Requirement IDs: FR-018, FR-020, RULE-004
Acceptance Criteria: AC-030, AC-034, AC-038
Priority: Must
Level: Acceptance
Automation Target: `app/test/router/onboarding_status_route_test.dart`

```gherkin
Feature: Account-wide onboarding status
  Scenario: Completed account bypasses onboarding
    Given the AppView reports permanent completion for the active DID
    When the app starts on a new device-shaped client state
    Then the main app opens and onboarding does not

  Scenario: Status read fails
    Given the AppView status read fails
    When active-account initialization runs
    Then a retryable initialization gate is shown
    And neither onboarding nor the main shell is selected
```

### AT-014: Layout, Localization, And Semantics Remain Usable
Requirement IDs: NFR-001, NFR-002
Acceptance Criteria: AC-019, AC-020
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_accessibility_layout_test.dart`

```gherkin
Feature: Accessible responsive onboarding
  Scenario Outline: Every step remains operable
    Given onboarding is rendered at <viewport> with <textScale>
    When keyboard, validation, loading, and long Instagram content states are exercised
    Then no Flutter overflow exception occurs
    And all content can be reached by scrolling
    And app-bar and bottom actions remain operable
    And localized controls announce progress, selection, busy, and disabled state
    Examples:
      | viewport | textScale |
      | compact phone | 1.0 |
      | compact phone | 2.0 |
      | tablet/desktop | 1.0 |
```

### AT-015: Stale Account Operations Cannot Cross Leases
Requirement IDs: NFR-003, RULE-003
Acceptance Criteria: AC-021
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_account_isolation_test.dart`

```gherkin
Feature: Multi-account safety
  Scenario Outline: Account changes during async work
    Given an operation started for account A
    When account B becomes active before <operation> completes
    Then the result does not update B's profile, Instagram UI, or onboarding state
    And any completion retry remains owned by account A's valid session only
    Examples:
      | operation |
      | profile read or save |
      | Instagram action |
      | onboarding completion write |
```

### AT-016: Bluesky Prefill Retry Is Bounded
Requirement IDs: FR-021
Acceptance Criteria: AC-035, AC-036
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_profile_prefill_test.dart`

```gherkin
Feature: Delayed Bluesky identity indexing
  Scenario: Identity appears during retry window
    Given the first successful AppView profile response has no Bluesky identity fields
    And a later response within 5 seconds contains identity fields
    When step 1 initializes
    Then the later values are pre-filled
    And the bounded retry sequence is not repeated after revisiting step 1

  Scenario: Identity remains absent
    Given successful responses remain empty through the 5-second bound
    When the bound expires
    Then editable optional empty fields are shown
    But a true profile-read error shows retry UI instead
```

### AT-017: Profile Saves Preserve Full Snapshot And Remain Last-Write-Wins
Requirement IDs: FR-005, RULE-001, RULE-006
Acceptance Criteria: AC-009, AC-040
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_profile_payload_test.dart`

```gherkin
Feature: Atomic profile preservation
  Scenario Outline: One onboarding step changes
    Given the loaded profile has identity, banner, avatar, known crafts, and an unknown craft identifier
    And another client may update the profile after this snapshot loads
    When onboarding changes only <stepFields> and saves
    Then Flutter sends the complete editable onboarding snapshot
    And the AppView read-before-write merge preserves untouched PDS fields and unknown crafts in the resulting records
    And no new client concurrency field, pre-save refresh, merge UI, or conflict prompt is introduced
    Examples:
      | stepFields |
      | display name and bio |
      | known crafts |
```

### AT-018: Optimistic Completion Retries Silently In Memory
Requirement IDs: FR-009, FR-019
Acceptance Criteria: AC-032, AC-033
Priority: Must
Level: Acceptance
Automation Target: `app/test/onboarding/onboarding_completion_test.dart`

```gherkin
Feature: Optimistic completion reconciliation
  Scenario: Initial completion write fails
    Given Skip or Finish has released the router gate
    And the first completion write fails
    Then the member remains in the main app
    And no error message is shown
    And the client retries while the owning process and session remain active

  Scenario: Process ends before reconciliation
    Given every in-memory retry failed
    When the process is reconstructed
    Then no durable local pending marker exists
    And cold-start routing follows the server's incomplete state
```

### AT-019: OAuth Eagerly Projects Available Bluesky Profile
Requirement IDs: FR-024, NFR-004, RULE-007
Acceptance Criteria: AC-041, AC-042, AC-043
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/auth/initialize_profile_test.go` and `appview/internal/app/federated_real_flow_integration_test.go`

```gherkin
Feature: OAuth-time Bluesky profile projection
  Scenario: Existing Bluesky profile is projected before handoff
    Given the login or registration callback fetches a valid app.bsky.actor.profile and CID
    When OAuth initialization completes
    Then canonical projection has stored its identity fields and CID in bluesky_profiles
    And projection occurs before Flutter handoff creation

  Scenario: Direct projection fails
    Given the Bluesky profile fetch succeeds
    But direct database projection fails
    When remaining OAuth initialization succeeds
    Then a non-sensitive warning is logged
    And sign-in and handoff creation continue
    And repository tracking and Tap/backfill remain enabled

  Scenario: Tap later delivers the projected record
    Given OAuth directly projected a DID and CID
    When Tap delivers the same record
    Then one correct row remains
    And canonical field parsing does not diverge
```

## 4. Unit Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-003, FR-006, FR-022 | AC-006, AC-010, AC-027 | Derive bottom action and navigation enabled state. | clean/dirty, valid/invalid, upload state, save state, step index | Correct `Next`, `Save & next`, or `Finish`; Save/Skip/Back gates match requirements. | `app/test/onboarding/onboarding_action_state_test.dart` |
| UT-002 | FR-012 | AC-017 | Derive progress label and fraction for each step. | indexes 0, 1, 2; localized template | Labels are Step 1/2/3 of 3 and fractions are 1/3, 2/3, 1.0; no navigation callback exists. | `app/test/onboarding/onboarding_progress_test.dart` |
| UT-003 | FR-021 | AC-035, AC-036 | Verify bounded prefill retry with fake time. | empty success, delayed populated success, read error, elapsed clock | Retries stop at populated data, error, or 5 seconds; sequence runs once per session. | `app/test/onboarding/onboarding_profile_prefill_test.dart` |
| UT-004 | FR-019 | AC-032, AC-033 | Verify optimistic completion retry ownership and lifetime. | failing/succeeding repository, account lease changes, provider disposal | Immediate local complete; silent retries stop on success, disposal, or stale lease; no persistent marker write. | `app/test/onboarding/providers/onboarding_status_provider_test.dart` |
| UT-005 | FR-011, RULE-005 | AC-016, AC-028, AC-039 | Verify draft state is retained only by one flow controller instance. | profile/craft drafts, step transitions, controller reconstruction | Navigation restores drafts; reconstruction starts at step 1 without drafts. | `app/test/onboarding/onboarding_flow_state_test.dart` |
| UT-006 | FR-020 | AC-034 | Map status provider loading/data/error into startup gate states. | loading, incomplete, complete, error | Loading/error do not choose a route; error exposes Retry; data selects the correct route. | `app/test/auth/active_account_initialization_provider_test.dart` |
| UT-007 | RULE-001 | AC-009 | Compose the complete client-editable profile payload from each step snapshot. | display name, bio, optional newly uploaded avatar, known/unknown crafts | Identity and all craft IDs are sent; omitted unchanged image fields are left for the existing AppView read-before-write merge. | `app/test/onboarding/onboarding_profile_payload_test.dart` |
| UT-008 | FR-016 | AC-025 | Ensure onboarding suggestion presentation has no route callback. | suggestion item and inline handlers | Follow/dismiss are present; row navigation is absent/disabled. | `app/test/onboarding/onboarding_instagram_step_test.dart` |
| UT-009 | FR-024, RULE-007 | AC-041, AC-042 | Verify OAuth initialization projection branching and ordering. | profile present/missing, projector success/failure, recording handoff/projector | Present record passes DID/CID/body to projector before handoff; missing skips projection; projector failure is logged and does not fail callback. | `appview/internal/auth/initialize_profile_test.go`; `handlers_test.go` |

## 5. Integration Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| IT-001 | FR-018, RULE-003, RULE-004 | AC-030, AC-031, AC-038 | Test authenticated completion handlers and store. | Isolated Postgres schema with active Alice and Bob; route middleware context | Read incomplete, complete Alice twice, read both accounts | Alice becomes permanently complete; repeated write is idempotent; Bob remains incomplete; camelCase response and error envelope comply. | `appview/internal/api/onboarding_test.go` |
| IT-002 | FR-018 | AC-030, AC-031 | Test route policy and authentication envelope. | Full route mux and completion store | Request read/write without auth, without device ID, as departed owner, and as active owner | Existing 401/400/404 conventions apply; active current member succeeds; no account selector is accepted. | `appview/internal/routes/onboarding_route_test.go` |
| IT-003 | FR-005, RULE-001 | AC-009 | Verify onboarding-triggered profile updates preserve both record domains. | Existing PDS Bluesky record with banner/avatar refs, CraftSky record with known/unknown crafts, and fake/real profile effect harness | Save identity-only and crafts-only payloads | AppView read-before-write merge and existing `ExpectedCID` preserve untouched PDS fields; desired crafts survive; partial PDS failure remains retryable. | `appview/internal/api/profile_effects_test.go` plus Flutter payload suite |
| IT-004 | FR-014 | AC-023 | Verify `GET /v1/profiles/me` returns joined indexed Bluesky identity and CraftSky crafts. | `craftsky_profiles` and `bluesky_profiles` rows for same DID | Fetch authenticated self profile | Avatar/display name/bio and crafts are returned in existing wire shape. | `appview/internal/api/profile_store_test.go` and `profile_test.go` |
| IT-005 | FR-018 | AC-030, AC-031 | Verify migration and private deletion cleanup. | Apply new migration; seed completion for Alice and Bob; initialize owner cleanup harness | Complete Alice, run Alice private cleanup, migrate down in isolated release gate | Alice completion is removed, Bob remains, down migration succeeds. | `appview/internal/accountdeletion/private_cleanup_test.go`; `just appview-check` |
| IT-006 | NFR-003, RULE-003 | AC-021 | Reject stale async results after account switch. | Deferred profile, Instagram, and completion repositories for lease A; activate lease B | Resolve A operations after switch | B state is unchanged and A completion retry does not continue under B. | `app/test/onboarding/onboarding_account_isolation_test.dart` |
| IT-007 | FR-008, FR-015, FR-016 | AC-012, AC-013, AC-024, AC-025 | Verify shared Instagram sections against existing providers. | Existing fake migration repository for unlinked, linked, inactive, import, and paginated suggestion states | Render sections in onboarding and settings hosts | Shared states/actions match; onboarding omits history/revoke/profile navigation; settings retains management content. | `app/test/instagram_migration/instagram_migration_page_test.dart`; `app/test/onboarding/onboarding_instagram_step_test.dart` |
| IT-008 | FR-019, FR-020 | AC-032, AC-033, AC-034 | Verify Flutter API repository and startup-provider contract. | Fake Dio responses for status success/failure and completion retry | Cold-start read; optimistic finish with transient write failures | Gate waits/retries correctly; optimistic state exits; retries are silent and non-durable. | `app/test/onboarding/data/api_onboarding_repository_test.dart` |
| IT-009 | FR-024, NFR-004, RULE-007 | AC-041, AC-042, AC-043 | Verify direct OAuth projection against the real canonical projection/store path. | Isolated Postgres profile tables, OAuth PDS fake returning full profile/CID, injected canonical projector, recording handoff, and Tap event for same CID | Complete callback, inspect row before handoff, replay same event; repeat with missing record and projector failure | Full fields/CID are present before handoff; replay is idempotent; missing is a no-op; projection failure continues sign-in and leaves Tap tracking enabled. | `appview/internal/app/federated_real_flow_integration_test.go`; `appview/internal/index/bluesky_profile_test.go` |

## 6. Regression Tests
| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Edit profile still seeds current data, validates, sends a complete profile, preserves unknown crafts, updates cache, and handles save failure/discard. | RULE-001, RULE-006 | Extend/retain `app/test/profile/edit_profile_dialog_test.dart` and image/facet suites after extracting shared controls. |
| REG-002 | Instagram settings retains verification, linked state, discoverability, reactivation, both import methods, import history, suggestions, revocation, refresh, and error behavior. | FR-008, FR-015, FR-016 | Keep existing `app/test/instagram_migration/instagram_migration_page_test.dart` and `instagram_suggestions_test.dart` passing; add explicit shared-section host parity cases. |
| REG-003 | Existing router redirects signed-in incomplete accounts to onboarding and complete accounts to home without affecting auth-complete/account-deletion exceptions. | FR-018, FR-020 | Update `app/test/router/router_redirect_test.dart` for async server status while retaining all non-onboarding redirect cases. |
| REG-004 | Profile writes retain the current client snapshot semantics and AppView read-before-write/`ExpectedCID` behavior. | RULE-006 | Assert `PUT /v1/profiles/me` adds no client concurrency field or pre-save refresh while existing server-side record preconditions remain intact. |
| REG-005 | Craft catalog contains the same 22 stable IDs/order and localized labels. | FR-004 | Retain picker/catalog tests and assert onboarding reuses the catalog rather than duplicating it. |
| REG-006 | Active account initialization still gates on account-critical state and provides retry UI. | FR-020, NFR-003 | Extend existing initialization gate/provider tests with onboarding status while preserving language/session behavior. |
| REG-007 | No `SharedPreferences` onboarding key remains as a second durable authority. | FR-018, FR-019 | Replace old local-provider tests with repository-backed tests and assert completion flow does not write `onboarded_<did>` or a pending marker. |
| REG-008 | Existing OAuth missing-profile behavior, callback safety, repository tracking, Tap projection, and one-shot backfill remain intact. | FR-024, NFR-004, RULE-007 | Retain `initialize_profile_test.go`, `handlers_test.go`, `bluesky_profile_test.go`, `bluesky_backfiller_test.go`, and federated flow tests while adding the eager projector seam. |

## 7. Test Data
| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Fully populated profile state | Flutter profile with DID `did:plc:alice`, handle `alice.test`, display name, bio, avatar/banner URLs, crafts `sewing`, `quilting`, and unknown `future-craft`; corresponding PDS records retain the actual image blob references | AT-003, AT-005, AT-017, IT-003, IT-004 |
| TD-002 | Empty optional profile | Valid DID/handle with null identity fields and empty crafts | AT-002, AT-016 |
| TD-003 | Multi-account isolation | Active Alice lease and Bob lease with separate tokens, completion rows, profiles, and deferred operations | AT-007, AT-015, IT-001, IT-006 |
| TD-004 | Instagram account states | Unlinked/available, unavailable, active linked, inactive, reactivation-required, pending confirmation, and busy verification | AT-009, AT-010, AT-012, IT-007 |
| TD-005 | Instagram imports | Valid manual handles, valid export parse result, invalid export/manual input, success/failure repository responses | AT-009, IT-007, REG-002 |
| TD-006 | Instagram suggestions | At least two suggestions, busy IDs, follow/dismiss success/failure, and a non-null next cursor | AT-011, IT-007 |
| TD-007 | Completion outcomes | Incomplete, complete timestamp, transient write failures followed by success, permanent failures, read failure | AT-013, AT-018, UT-004, IT-001, IT-008 |
| TD-008 | Responsive/accessibility states | Compact phone, large viewport, text scales 1.0 and 2.0, visible keyboard, long localized/privacy strings | AT-014, MAN-002 |
| TD-009 | OAuth Bluesky projection | Full `app.bsky.actor.profile` map with display name, description, avatar/banner blob references and CID; missing-record response; projection error; duplicate Tap event with same DID/CID | AT-019, UT-009, IT-009, REG-008 |

## 8. Manual Checks
| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | FR-023 | Real gallery picker on supported mobile platforms | Run onboarding on iOS and Android; open gallery; cancel once; select a valid image once; deny/restrict photo access where supported. | Cancel leaves draft clean; valid image previews/uploads; permission or picker failures are recoverable; no camera permission is requested. |
| MAN-002 | NFR-001, NFR-002 | Touch, keyboard, safe-area, and long-content usability | Exercise all steps on a small phone and tablet/desktop with large system text, software keyboard, long Instagram content, and screen reader enabled. | No obscured content or action, scrolling is natural, focus order is logical, and announced labels/states match visible behavior. |

## 9. Test Gaps And Risks
| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | True cross-device cold-start behavior is not covered by one automated Flutter process. | FR-018, RULE-004 | Current test harness is process-local and has no device farm. | Cover server authority through Go integration tests and reconstruct Flutter containers with empty local state; optionally verify on two devices before release. |
| GAP-002 | Real OS gallery permission and picker behavior cannot be proven by widget mocks. | FR-023 | `image_picker` platform channels are outside widget-test scope. | Execute MAN-001 on iOS and Android; retain media-service unit tests for validation/preparation. |
| GAP-003 | Visual quality across every locale is not exhaustively automated. | NFR-001, NFR-002 | Generated localization and arbitrary string expansion exceed practical matrix size. | Automate English large-text/viewport cases and perform MAN-002 with representative long strings. |
| GAP-004 | Last-write-wins can overwrite a concurrent external profile edit by design. | RULE-006 | Conflict control was explicitly excluded. | Keep regression coverage proving no accidental concurrency contract was introduced; retain RISK-009 in implementation review. |

## 10. Out Of Scope
- Tests for onboarding analytics, event instrumentation, or metrics.
- Tests for camera capture, avatar removal, banner editing, account switching/sign-out within onboarding, or suggestion-to-profile navigation.
- Tests for persisting drafts/current step, durable completion retry markers, versioned completion, or future onboarding revisions.
- New profile conflict detection, client-supplied preconditions, pre-save refresh/merge, or conflict UI tests.
- New Instagram backend behavior; existing backend/provider tests remain authoritative.
- Removal of Tap tracking/backfill or tests treating direct OAuth projection as the sole indexing path.
- Lexicon or public profile-record schema changes, because none are required.

## 11. Handoff To Document Review
- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-08-31-onboarding-experience/`
- Recommended first failing test for implementation: `UT-009`, proving OAuth passes the already-fetched Bluesky record/CID to a best-effort projector before handoff, followed by `IT-009` against the canonical projection path. Then begin `IT-001` for account-wide onboarding completion.
- Suggested test order for implementation: OAuth eager projection (`UT-009`, `IT-009`, `AT-019`, `REG-008`); AppView completion migration/store/handler and cleanup (`IT-001`, `IT-002`, `IT-005`); Flutter status repository/provider/startup gate (`UT-004`, `UT-006`, `IT-008`, `AT-013`, `AT-018`); flow state/action logic (`UT-001` through `UT-005`); profile and craft steps (`AT-003` through `AT-006`, `AT-016`, `AT-017`); shared Instagram sections (`AT-009` through `AT-012`, `IT-007`); routing/account isolation/accessibility and remaining regressions (`AT-001`, `AT-002`, `AT-007`, `AT-008`, `AT-014`, `AT-015`, `REG-001` through `REG-007`).
- Commands discovered: `just app-test <test-path>`, `just app-analyze`, `just appview-test-unit`, and release-equivalent `just appview-check` (isolated Postgres/MinIO and migration down-to-zero evidence).
- Blocking gaps: None.
