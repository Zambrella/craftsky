# Acceptance Test Specification: Settings Page And Account Management

## 1. Test Strategy

Risk level: **High**. The visible Settings work is straightforward, but permanent CraftSky deletion crosses fresh OAuth reauthentication, active-account fencing, local secure state, durable AppView jobs, PDS record deletion, private Postgres data, session revocation, Instagram migration data, Tap/indexer convergence, restricted status authorization, and timed audit expiry. A false success, cross-account action, or incomplete deletion would be materially worse than a presentation defect.

The automated design therefore uses complementary levels:

- Flutter widget and router tests cover identity fallbacks, exact section order, row affordances, child-page navigation, external links, cache clearing, Sign out, two-step confirmation, deletion status, responsive account switching, local cleanup, localization, and accessibility semantics.
- Dart unit/provider tests cover pure identity/row projections, handle matching, account-lease fencing, local deletion-state transitions, MRU fallback activation, status-only account rows, and fresh-rejoin routing.
- Go unit tests cover deletion phases, retry classification, terminal-success rules, Lexicon collection inventory, namespace/blob boundaries, audit minimization, cleanup planning, and redaction.
- Go integration tests use `httptest`, deterministic PDS fakes, injected clocks, and `internal/testdb.WithSchema` to cover API contracts, fresh-reauth proof and deletion-only OAuth binding, durable acceptance, job restart/idempotency, pagination, ordinary/status credential separation, complete private cleanup, Instagram deletion, Tap replay, expected-URI receipts, AppView convergence, observability, and terminal-success-anchored 30-day expiry.
- Regression tests protect existing Settings destinations, Notifications settings, account switching, cache clearing, Sign out, shell version labels, OAuth token boundaries, and existing indexer deletion effects.
- Manual checks are limited to visual/accessibility presentation and real OAuth/PDS/Tap or offline-device behavior that repository fakes cannot fully prove.

All time-based tests inject clocks and retry schedulers; they must not wait in wall-clock time. PDS, OAuth, and convergence fakes expose deterministic barriers before and after each destructive phase so app closure, worker restart, duplicate submission, partial completion, grant expiry, receipt failure, and event lag can be tested precisely. Tap fakes preserve URI, owner DID, event ID, repo revision, action, and ack ordering. Test fixtures use disposable DIDs only and must never invoke whole-PDS account deletion.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, AT-002, UT-001, IT-001, IT-003 | Acceptance / Unit / Integration | Yes |
| BR-002 | AC-003, AC-004, AC-005 | AT-001, AT-003, IT-001, IT-002, REG-001, REG-002 | Acceptance / Integration / Regression | Yes |
| BR-003 | AC-015, AC-016, AC-017 | AT-006, AT-009, AT-010, IT-007, IT-009, IT-010, IT-012, IT-027, REG-009, MAN-003 | Acceptance / Integration / Regression / Manual | Partial: real-stack confirmation is manual |
| BR-004 | AC-003, AC-010, AC-012, AC-014 | AT-001, AT-004, AT-005, AT-014, IT-001, IT-004, IT-005, REG-003, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-001 | AC-001, AC-020, AC-031 | AT-001, UT-001, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-002, AC-021 | AT-002, IT-003, REG-005 | Acceptance / Integration / Regression | Yes |
| FR-003 | AC-003, AC-004, AC-005, AC-032 | AT-001, AT-003, UT-002, IT-001, IT-002, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| FR-004 | AC-003, AC-007, AC-022, AC-033 | AT-001, AT-004, AT-015, UT-002, UT-023, IT-001, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-006, AC-021 | AT-002, AT-003, IT-002, IT-003 | Acceptance / Integration | Yes |
| FR-006 | AC-004, AC-006 | AT-003, IT-002, REG-002 | Acceptance / Integration / Regression | Yes |
| FR-007 | AC-005, AC-007 | AT-003, AT-004, IT-002, IT-004 | Acceptance / Integration | Yes |
| FR-008 | AC-008, AC-019 | AT-004, UT-020, IT-004, REG-007 | Acceptance / Unit / Integration / Regression | Yes |
| FR-009 | AC-008, AC-019 | AT-004, UT-020, IT-004, REG-007 | Acceptance / Unit / Integration / Regression | Yes |
| FR-010 | AC-007, AC-009, AC-019 | AT-004, UT-020, IT-004, REG-003 | Acceptance / Unit / Integration / Regression | Yes |
| FR-011 | AC-010, AC-020, AC-034 | AT-004, UT-003, IT-004, REG-006 | Acceptance / Unit / Integration / Regression | Yes |
| FR-012 | AC-005, AC-011 | AT-003, IT-002, IT-005 | Acceptance / Integration | Yes |
| FR-013 | AC-012, AC-013, AC-035 | AT-005, UT-004, UT-024, IT-005, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-012, AC-036 | AT-005, UT-005, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-015, AC-016, AC-017, AC-023, AC-037 | AT-006, AT-009, AT-010, UT-008, UT-009, UT-010, IT-007, IT-009, IT-010, IT-011, IT-012, IT-027 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-018, AC-038 | AT-006, AT-007, UT-012, UT-013, IT-014, IT-015 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-013, AC-018, AC-023, AC-039 | AT-005, AT-008, UT-006, UT-007, IT-016 | Acceptance / Unit / Integration | Yes |
| FR-018 | AC-014, AC-024 | AT-014, UT-021, REG-004 | Acceptance / Unit / Regression | Yes |
| FR-019 | AC-035, AC-040 | AT-005, UT-024, IT-006, REG-011 | Acceptance / Unit / Integration / Regression | Yes |
| FR-020 | AC-023, AC-037, AC-041 | AT-006, AT-008, UT-006, UT-008, UT-024, IT-006, IT-008, IT-009, IT-011, IT-027, IT-028 | Acceptance / Unit / Integration | Yes |
| FR-021 | AC-016, AC-040, AC-042 | AT-006, AT-007, AT-008, AT-011, UT-007, UT-011, UT-024, IT-006, IT-009, IT-017 | Acceptance / Unit / Integration | Yes |
| FR-022 | AC-036, AC-043 | AT-006, AT-007, UT-014, IT-014, IT-015, IT-025, MAN-004 | Acceptance / Unit / Integration / Manual | Partial: offline-device behavior has a manual complement |
| FR-023 | AC-044 | AT-012, UT-008, IT-018, IT-019, IT-027, REG-008, REG-010 | Acceptance / Unit / Integration / Regression | Yes |
| FR-024 | AC-045 | AT-013, UT-016, IT-020, REG-012 | Acceptance / Unit / Integration / Regression | Yes |
| FR-025 | AC-046 | AT-010, UT-017, UT-019, IT-021 | Acceptance / Unit / Integration | Yes |
| FR-026 | AC-047 | AT-010, UT-018, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-027 | AC-038, AC-039, AC-044 | AT-006, AT-007, AT-008, AT-012, UT-006, UT-007, UT-013, IT-016, IT-019 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-021, AC-025 | AT-002, AT-005, UT-001, UT-015, UT-024, IT-003, IT-006, IT-026 | Acceptance / Unit / Integration | Yes |
| NFR-002 | AC-017, AC-023, AC-026, AC-041 | AT-008, AT-009, UT-007, UT-010, IT-007, IT-008, IT-011, IT-028 | Acceptance / Unit / Integration | Yes |
| NFR-003 | AC-022, AC-027 | AT-015, UT-013, UT-023, IT-001, IT-004, IT-005, MAN-002 | Acceptance / Unit / Integration / Manual | Partial: assistive-technology presentation is manual |
| NFR-004 | AC-028 | AT-015, UT-022, IT-001, IT-005 | Acceptance / Unit / Integration | Yes |
| NFR-005 | AC-029 | AT-015, IT-001, IT-002, MAN-001 | Acceptance / Integration / Manual | Partial: final visual polish is manual |
| NFR-006 | AC-040, AC-042, AC-046 | AT-010, AT-011, UT-011, UT-017, UT-019, IT-008, IT-017, IT-021, IT-023 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-025 | AT-005, UT-015, UT-024, IT-006, IT-026 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-015, AC-017 | AT-009, UT-010, IT-007, IT-010, REG-009 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-003 | AC-003, AC-007, AC-010, AC-011, AC-014, AC-033 | AT-001, AT-003, AT-004, AT-014, UT-002, IT-001, IT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-014, AC-016, AC-024 | AT-006, AT-014, UT-021, IT-009, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-005 | AC-015, AC-030 | AT-009, UT-009, IT-024 | Acceptance / Unit / Integration | Yes |
| RULE-006 | AC-039 | AT-008, UT-006, IT-016 | Acceptance / Unit / Integration | Yes |
| RULE-007 | AC-037, AC-044 | AT-009, AT-012, UT-008, IT-019, IT-027 | Acceptance / Unit / Integration | Yes |
| RULE-008 | AC-044 | AT-012, IT-018, IT-019, REG-010 | Acceptance / Integration / Regression | Yes |
| RULE-009 | AC-036, AC-048 | AT-005, AT-009, UT-005, IT-022, REG-009 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-010 | AC-046, AC-047 | AT-010, UT-017, UT-018, IT-013, IT-021 | Acceptance / Unit / Integration | Yes |
| RULE-011 | AC-045 | AT-013, UT-016, IT-020, REG-012 | Acceptance / Unit / Integration / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Render The Identity-Led Sectioned Settings Hub

- Requirement IDs: BR-001, BR-002, BR-004, FR-001, FR-003, FR-004, NFR-003, NFR-004, NFR-005, RULE-003
- Acceptance Criteria: AC-001, AC-003, AC-020, AC-022, AC-028, AC-029, AC-031, AC-032, AC-033
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/settings/settings_page_test.dart`

```gherkin
Feature: Settings hub
  Scenario Outline: Render the active identity and agreed information hierarchy
    Given <identity state> is active
    When Settings opens
    Then the established account avatar or its fallback is shown
    And <primary identity> is shown as the primary identity
    And <secondary identity> is shown as the secondary identity
    And no raw DID or No username text is visible
    And Switch account appears immediately after the identity
    And the titled sections and rows appear in the exact agreed order
    And Sign out is separated after General
    And in-app disclosure rows have one direction-aware chevron
    And direct actions have no trailing navigation icon

    Examples:
      | identity state | primary identity | secondary identity |
      | display name and handle available | Alice | @alice.test |
      | display name missing | @alice.test | no duplicate secondary line |
```

### AT-002: Reuse The Responsive Account Switcher

- Requirement IDs: BR-001, FR-002, FR-005, NFR-001
- Acceptance Criteria: AC-002, AC-006, AC-021, AC-025
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/router/app_shell_account_switcher_test.dart`, `app/test/router/account_switch_routing_test.dart`

```gherkin
Feature: Switch account from Settings
  Scenario Outline: Open the existing switcher and activate another account
    Given Alice is active and Bob is retained
    And Settings is open on a <layout> layout
    When Switch account is selected
    Then the existing <switcher surface> opens
    When Bob is selected and the existing unsaved-work guard permits activation
    Then Bob becomes active through the existing account lease boundary
    And the old Settings flow closes
    And Bob lands on Home
    And any late Alice identity or navigation completion cannot affect Bob

    Examples:
      | layout | switcher surface |
      | compact | modal bottom sheet |
      | large | anchored popover |
```

### AT-003: Navigate To Existing And New Settings Destinations

- Requirement IDs: BR-002, FR-003, FR-005, FR-006, FR-007, FR-012, RULE-003
- Acceptance Criteria: AC-004, AC-005, AC-006, AC-007, AC-011
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/router/settings_routes_test.dart`, `app/test/router/notification_settings_route_test.dart`

```gherkin
Feature: Settings destinations
  Scenario Outline: Open and return from a Settings destination
    Given Settings is open for Alice
    When <row> is selected
    Then <destination> opens through the authenticated shell
    And the active account remains Alice
    When Back is invoked
    Then Settings is restored with the correct compact or large shell selection

    Examples:
      | row | destination |
      | Notifications | existing notification settings page with unchanged controls |
      | Account | Account page containing only destructive Delete account |
      | About | About page containing Terms, Privacy policy, Clear image cache, and version |
      | Customisation | existing customisation destination |
      | Languages | existing language destination |
      | Followers | existing follower list |
      | Following | existing following list |
      | Muted accounts | existing muted list |
      | Blocked accounts | existing blocked list |
      | Find people from Instagram | existing Instagram migration destination |
```

### AT-004: Use About Without Changing Existing Behaviors

- Requirement IDs: BR-004, FR-004, FR-007, FR-008, FR-009, FR-010, FR-011, RULE-003
- Acceptance Criteria: AC-007, AC-008, AC-009, AC-010, AC-019, AC-020, AC-033, AC-034
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/settings/about_page_test.dart`, `app/test/settings/clear_image_cache_tile_test.dart`

```gherkin
Feature: About
  Scenario: Legal links, cache clearing, and build information use existing seams
    Given About is open with package version 1.2.3 and build 123
    Then Terms and Privacy policy show external-link icons rather than chevrons
    And Clear image cache and the build label show no trailing icon
    And the read-only build label is 1.2.3 (123), matching shell navigation
    When Terms or Privacy policy is selected
    Then the canonical URL opens through the device external-browser launcher
    When Clear image cache is selected
    Then no confirmation is shown
    And both existing cache scopes clear once while re-entry is disabled
    And existing safe success or mapped error feedback is used
```

### AT-005: Require Fresh Reauthentication And Exact Handle Confirmation

- Requirement IDs: BR-004, FR-013, FR-014, FR-017, FR-019, NFR-001, RULE-001, RULE-009
- Acceptance Criteria: AC-012, AC-013, AC-025, AC-035, AC-036, AC-040
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/settings/delete_account_flow_test.dart`, `appview/internal/auth/account_deletion_reauth_test.go`

```gherkin
Feature: Confirm permanent CraftSky deletion
  Scenario: Confirm the active account through OAuth and typed handle
    Given Alice is the active account
    When Delete account is selected
    Then fresh PDS OAuth reauthentication is required without asking CraftSky for a PDS password or emailed code
    And cancellation, expiry, failure, or a proof for another DID creates no job
    When Alice successfully reauthenticates
    Then the first confirmation names @alice.test
    And it states every required deletion, preservation, unrecoverability, blob-GC, and offline-device boundary
    And the second confirmation remains disabled until @alice.test is typed exactly
    When either confirmation is dismissed before submission
    Then no server or local mutation occurs
    When account activation changes before submission
    Then the stale proof cannot authorize a different account
```

### AT-006: Accept Deletion While Another Account Remains

- Requirement IDs: BR-003, FR-015, FR-016, FR-020, FR-021, FR-022, FR-027, RULE-004
- Acceptance Criteria: AC-016, AC-018, AC-038, AC-040, AC-041, AC-043
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/settings/account_deletion_controller_test.dart`, `app/test/router/app_shell_account_switcher_test.dart`

```gherkin
Feature: Multi-account deletion acceptance
  Scenario: Alice starts irreversible deletion while Bob remains retained
    Given Alice is active and Bob is the most recently used retained account
    When Alice submits the exact typed-handle confirmation and the job is accepted
    Then the fresh server OAuth session is durably bound to Alice's deletion job
    And every ordinary Alice session is immediately unusable
    And every other Alice OAuth session is removed
    And the bound session is unavailable to Flutter, ordinary APIs, and ordinary background writers
    And Alice's drafts, staged media, caches, ordinary session, and product state are erased locally
    And only Alice's minimal job binding and display identity remain for status
    And Bob becomes active immediately and lands on Home
    And Alice appears as a disabled Deleting… row in the switcher
    And selecting that row opens phase-level status without record counts or private details rather than ordinary Alice content
    And Alice's status row disappears only after terminal success
```

### AT-007: Show Deletion Status When No Other Account Remains

- Requirement IDs: FR-016, FR-021, FR-022, FR-027
- Acceptance Criteria: AC-018, AC-038, AC-042, AC-043
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/settings/deletion_status_page_test.dart`, `app/test/router/router_redirect_test.dart`

```gherkin
Feature: Last retained account deletion
  Scenario: The only retained account starts deletion
    Given Alice is the only retained account
    When Alice's deletion job is accepted
    Then Alice's ordinary local session and product data are erased
    And the app shows deletion status as the primary signed-out experience
    And the retained credential can read only Alice's job status, begin replacement fresh reauthentication when required, and request Retry
    And it cannot call a PDS endpoint or receive the job-bound server OAuth session
    And ordinary authenticated destinations are unavailable
    When terminal success is observed
    Then the remaining local status binding and display identity are erased
    And the ordinary signed-out welcome flow is available
```

### AT-008: Resume A Durable Non-Cancelable Job And Retry Safely

- Requirement IDs: FR-017, FR-020, FR-021, FR-027, NFR-002, RULE-006
- Acceptance Criteria: AC-023, AC-039, AC-041, AC-042
- Priority: Must
- Level: Acceptance
- Automation Target: `appview/internal/accountdeletion/worker_acceptance_test.go`, `app/test/settings/deletion_status_page_test.dart`

```gherkin
Feature: Durable deletion recovery
  Scenario Outline: Resume after interruption without restoring ordinary access
    Given Alice's typed-handle submission was accepted
    And the durable job has completed some phases
    When <interruption> occurs
    Then reconnecting resolves the same owner-scoped job
    And completed work is not duplicated unsafely
    And no cancellation or ordinary-account restoration action is available
    And transient failures retry only within configured bounds
    And an exhausted or non-transient failure becomes Deletion needs attention
    And an unusable job OAuth session requires a fresh OAuth redirect from status before Retry
    And manual Retry plus support guidance are available without restoring ordinary access

    Examples:
      | interruption |
      | the app closes before receiving the acceptance response |
      | the network is lost |
      | the AppView worker restarts |
      | the acceptance request is duplicated |
      | a private cleanup phase fails after earlier phases succeed |
      | the bound PDS OAuth grant expires and is replaced through fresh reauthentication |
```

### AT-009: Delete Only CraftSky Records And Reach The Full Terminal Boundary

- Requirement IDs: BR-003, FR-015, FR-020, NFR-002, RULE-002, RULE-005, RULE-007, RULE-009
- Acceptance Criteria: AC-015, AC-017, AC-023, AC-026, AC-030, AC-037, AC-048
- Priority: Must
- Level: Acceptance
- Automation Target: `appview/internal/accountdeletion/worker_acceptance_test.go`, `appview/internal/accountdeletion/pds_deleter_test.go`

```gherkin
Feature: Owner-scoped CraftSky PDS deletion
  Scenario: Delete every CraftSky record without deleting the AT account or another namespace
    Given Alice's repo contains profile, post/project/reply/quote, like, and repost records under social.craftsky
    And it contains records under app.bsky and another namespace
    And it contains more CraftSky records than one PDS page or delete batch
    And some shared blobs remain referenced by retained records
    When Alice's deletion job runs and is safely retried
    Then every current and registered future social.craftsky record collection becomes empty
    And defs-only Lexicons are never treated as collections
    And missing records are idempotent
    And non-CraftSky records and the AT/PDS account remain
    And no whole-account or direct shared-blob deletion call occurs
    And PDS garbage collection is not a terminal-success gate
```

### AT-010: Remove Private And Instagram Data But Retain Only The Timed Audit

- Requirement IDs: BR-003, FR-015, FR-025, FR-026, NFR-006, RULE-010
- Acceptance Criteria: AC-016, AC-037, AC-046, AC-047
- Priority: Must
- Level: Acceptance
- Automation Target: `appview/internal/accountdeletion/private_cleanup_test.go`, `appview/internal/accountdeletion/audit_test.go`

```gherkin
Feature: Private deletion and minimized audit
  Scenario: Terminal deletion removes all private product state
    Given Alice owns every direct and indirect store in the maintained private-data manifest
    And Bob and shared-resource controls use the same stores
    And Alice has Instagram links, imports, suggestions, verification/private imported data, and a username claim
    When Alice's deletion reaches terminal success
    Then every Alice-owned row, object-store object, owner-derived shared artifact, and username claim is gone despite ordinary retention behavior
    And Bob rows and shared resources still used by Bob remain
    And no job, OAuth binding, status credential, expected URI, or receipt state remains
    And the only retained deletion audit fields are DID, job ID, timestamps, and coarse outcome
    And the audit contains no handle, tokens, content, relationships, preferences, imports, or settings
    And the audit neither keeps membership active nor blocks rejoining
    And the audit still exists immediately before terminalSuccessAt plus 30 days
    When the injected clock reaches terminalSuccessAt plus 30 days
    Then the audit expires without affecting a fresh membership
```

### AT-011: Restrict Deletion Status Authorization

- Requirement IDs: FR-021, NFR-006
- Acceptance Criteria: AC-040, AC-042
- Priority: Must
- Level: Acceptance
- Automation Target: `appview/internal/routes/account_deletion_status_test.go`, `app/test/settings/data/account_deletion_repository_test.dart`

```gherkin
Feature: Deletion-status authorization
  Scenario: A status credential cannot act as an ordinary session
    Given Alice's ordinary sessions have been revoked after job acceptance
    And the confirming device holds Alice's restricted status credential
    And Alice's deletion job holds a separate server-side deletion OAuth session
    When the credential reads Alice's status, begins replacement fresh OAuth reauthentication, or requests Retry
    Then the permitted operation succeeds
    When it calls an ordinary CraftSky or PDS endpoint, requests Bob's job, or asks for the server OAuth session
    Then access is denied with the standard non-leaking error envelope
    And no unrelated account or record data is returned
    When the credential expires or is revoked
    Then it can no longer read or retry the job
```

### AT-012: Wait For Existing Indexers To Converge

- Requirement IDs: FR-023, FR-027, RULE-007, RULE-008
- Acceptance Criteria: AC-037, AC-044
- Priority: Must
- Level: Acceptance
- Automation Target: `appview/internal/accountdeletion/convergence_test.go`, existing `appview/internal/index/*_test.go` suites

```gherkin
Feature: AppView convergence
  Scenario: Tap deletion events lag behind completed PDS writes
    Given Alice's job durably registered each expected URI before its PDS delete
    And all social.craftsky records are gone from Alice's PDS
    But corresponding post, like, repost, and profile delete events have not all been successfully indexed and receipted
    Then deletion status reports a waiting-for-CraftSky phase rather than success
    And temporary direct-read visibility is tolerated
    And no parallel eager-hide or eager-purge path runs
    When existing idempotent indexers process duplicate and reordered delete events
    Then each expected event is acknowledged only after indexer handling and its idempotent job/URI/event-ID/repo-revision receipt succeed
    And a receipt-write failure leaves the event unacknowledged for Tap retry
    And indexed content and derived notifications are absent or retracted for every expected URI
    And a final PDS rescan remains empty
    And terminal success becomes eligible only after the job-bound OAuth session is removed
```

### AT-013: Reopen Pending Status Or Join Fresh After Success

- Requirement IDs: FR-024, RULE-011
- Acceptance Criteria: AC-045
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/auth/providers/auth_controller_test.dart`, `appview/internal/accountdeletion/rejoin_test.go`

```gherkin
Feature: Authenticate the same AT identity after deletion starts
  Scenario Outline: Authentication follows deletion state
    Given Alice's AT/PDS account still exists
    And CraftSky deletion is <state>
    When Alice authenticates again
    Then <outcome>

    Examples:
      | state | outcome |
      | pending or needs attention | only the existing deletion-status experience opens |
      | terminally successful | normal onboarding may create a fresh membership with no restored data |
```

### AT-014: Keep Sign Out Immediate And Non-Destructive

- Requirement IDs: BR-004, FR-018, RULE-003, RULE-004
- Acceptance Criteria: AC-014, AC-024
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/settings/sign_out_tile_test.dart`

```gherkin
Feature: Sign out remains distinct from Delete account
  Scenario: Sign out the active retained account
    Given Settings is open
    Then the Sign out icon and label use the theme error colour
    And the row has no chevron
    When Sign out is selected
    Then no confirmation is shown
    And the existing active-account sign-out lifecycle runs immediately
    And membership, CraftSky PDS records, and private CraftSky data remain unchanged
```

### AT-015: Support Responsive, Localized, Accessible Operation

- Requirement IDs: FR-004, NFR-003, NFR-004, NFR-005
- Acceptance Criteria: AC-022, AC-027, AC-028, AC-029
- Priority: Must
- Level: Acceptance
- Automation Target: `app/test/settings/settings_accessibility_test.dart`, `app/test/settings/settings_localizations_test.dart`

```gherkin
Feature: Accessible settings and deletion controls
  Scenario Outline: Operate Settings on a supported layout and text direction
    Given a supported locale with <direction> text direction
    And a <layout> viewport
    When Settings, About, Account, confirmation, and deletion status are exercised
    Then all new copy resolves through localization resources
    And disclosure chevrons point forward for the text direction
    And controls expose correct link, button, destructive, disabled, and selected semantics
    And every interactive target is at least 48 by 48 logical pixels
    And content remains scrollable, unclipped, and within the established width and theme system

    Examples:
      | direction | layout |
      | left-to-right | compact mobile |
      | left-to-right | large tablet or desktop |
      | right-to-left | compact mobile |
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, NFR-001 | AC-020, AC-031 | Project the active identity and discard stale identity completions. | Display name present/blank, handle, avatar present/missing, account lease changes before completion. | Correct avatar/fallback and non-duplicated handle render for the current lease only; no DID, token, `No username`, or prior-account identity leaks. | `app/test/settings/settings_identity_test.dart` |
| UT-002 | FR-003, FR-004, RULE-003 | AC-003, AC-032, AC-033 | Build the canonical Settings/About row descriptors and icon semantics. | Agreed section inventory, LTR/RTL direction, disclosure/external/action/read-only row kinds. | Exact order is stable; in-app disclosures get forward chevrons, legal links get external-link icons, and actions/version get neither. | `app/test/settings/settings_row_model_test.dart` |
| UT-003 | FR-011 | AC-010, AC-020, AC-034 | Reuse the shell build-label formatter, including incomplete metadata. | `1.2.3`/`123`, empty build number, missing package metadata. | About equals shell formatting; empty data does not produce malformed punctuation or an interactive row. | `app/test/settings/about_version_test.dart` |
| UT-004 | FR-013 | AC-012, AC-035 | Validate the typed handle exactly. | Active `@alice.test`; exact value, display name, DID, alias, whitespace/case/character variants, Bob's handle. | Submission enables only for the exact required active-handle value and never mutates on mismatch. | `app/test/settings/delete_account_confirmation_test.dart` |
| UT-005 | FR-014, RULE-009 | AC-036 | Verify confirmation content carries every required fact without over-promising. | Localized confirmation model/copy. | Copy names the handle and covers CraftSky/private/PDS-record deletion, all-device sign-out, unrecoverability, AT-account/non-CraftSky preservation, PDS GC limitation, and next-contact offline cleanup. | `app/test/settings/delete_account_copy_test.dart` |
| UT-006 | FR-017, FR-020, FR-027, RULE-006 | AC-023, AC-039, AC-041 | Enforce the deletion state machine and point of no return. | Pre-confirmation, accepted, phase-running, retrying, attention-required, successful states; cancel/pause/activate events. | Cancel is legal only before acceptance; accepted jobs cannot pause/cancel/reactivate and transition idempotently through phase-level states. | `appview/internal/accountdeletion/state_test.go` |
| UT-007 | FR-017, FR-021, FR-027, NFR-002 | AC-023, AC-039, AC-042 | Classify retryable failures and bound automatic retry. | Transient PDS/network/Tap/receipt failures, unusable OAuth grant, permanent validation failures, attempt counts, injected clock. | Transient failures schedule only configured bounded retries; an unusable grant requires replacement fresh reauthentication; exhaustion/permanent failure requires attention and manual Retry without restoring ordinary access. | `appview/internal/accountdeletion/retry_test.go` |
| UT-008 | FR-015, FR-020, FR-023, RULE-007 | AC-037, AC-044 | Evaluate terminal-success eligibility. | Final PDS rescan, private cleanup, ordinary-session removal, complete expected-URI receipts, absent/retracted indexed effects, deletion OAuth removal, blob-GC state. | Success is true only when every required gate passes; PDS blob-GC state does not participate. | `appview/internal/accountdeletion/terminal_test.go` |
| UT-009 | FR-015, RULE-005 | AC-015, AC-030 | Derive or validate the complete CraftSky record-collection inventory. | Current profile/post/like/repost Lexicons, defs-only schema, synthetic future record Lexicon. | Every primary `record` under `social.craftsky.*` is included exactly once; defs-only schemas are excluded. | `appview/internal/accountdeletion/collections_test.go` |
| UT-010 | FR-015, NFR-002, RULE-002 | AC-015, AC-017, AC-023 | Plan owner-scoped paginated deletion without namespace expansion. | Authenticated DID, paginated record pages, already-missing records, non-CraftSky collections. | Only the authenticated repo's registered CraftSky collections are deleted; missing records converge; whole-account and other-namespace operations are impossible. | `appview/internal/accountdeletion/pds_deleter_test.go` |
| UT-011 | FR-021, NFR-006 | AC-040, AC-042 | Separate status credentials, ordinary sessions, and deletion-only OAuth authority. | Matching/different job and owner; status/read, reauth start, Retry, ordinary/PDS API; expired/revoked status credential; bound/unbound OAuth session. | Only matching status/read, reauth start, and Retry are permitted to the device; ordinary/PDS/cross-job/cross-owner access and OAuth-session disclosure are denied; only the worker may resume the matching bound OAuth session. | `appview/internal/accountdeletion/status_authorization_test.go`, `oauth_binding_test.go` |
| UT-012 | FR-016, FR-022 | AC-018, AC-038, AC-043 | Transition the local registry at job acceptance. | Active Alice, remaining accounts with MRU order, no remaining account, terminal success. | MRU fallback activates immediately or status becomes primary; Alice retains only minimal status state and disappears after success. | `app/test/auth/models/session_registry_deletion_test.dart` |
| UT-013 | FR-016, FR-027, NFR-003 | AC-018, AC-027, AC-038 | Project deletion-state account-switcher rows. | Pending, convergence, retrying, attention-required, successful jobs. | Pending rows are disabled for activation, expose correct phase semantics/labels without record counts or private details, open status, offer Retry only when permitted, and vanish after success. | `app/test/auth/models/account_switcher_state_test.dart` |
| UT-014 | FR-022, NFR-006 | AC-043 | Compute device-local cleanup while preserving only status necessities. | Drafts, staged media, caches, ordinary token, cached product state, job ID, status credential, display identity. | Product data and ordinary credentials are selected for erasure; only minimal pending-job binding/identity remain until success. | `app/test/settings/account_deletion_local_cleanup_test.dart` |
| UT-015 | NFR-001, RULE-001 | AC-025 | Fence reauthentication, confirmation, and deletion completion to the captured active-account lease. | Alice lease/proof/request followed by Bob activation and late completions. | Alice's flow stays bound to Alice; no Bob deletion, removal, navigation, or success feedback occurs. | `app/test/settings/account_deletion_lease_test.dart` |
| UT-016 | FR-024, RULE-011 | AC-045 | Select the post-authentication destination from deletion state. | Same DID with pending, attention-required, successful, absent job; prior product data present/absent. | Pending states route only to status; success permits fresh onboarding; no deleted product state is restored. | `app/test/auth/models/deletion_login_policy_test.dart` |
| UT-017 | FR-025, NFR-006, RULE-010 | AC-046 | Minimize and expire deletion audit records. | Full job object containing prohibited fields; terminal-success timestamp; clocks immediately before, at, and after `terminalSuccessAt + 30 days`. | Projection retains only DID/job ID/timestamps/coarse outcome; it exists immediately before the boundary, expires at the boundary, and does not block rejoin. | `appview/internal/accountdeletion/audit_test.go` |
| UT-018 | FR-026, RULE-010 | AC-047 | Build the explicit-deletion Instagram cleanup plan. | Links, imports, suggestions, verification, private imported data, username claim, ordinary retention state. | Every listed category and claim is hard-deleted regardless of ordinary inactivation/retention behavior. | `appview/internal/accountdeletion/instagram_cleanup_test.go` |
| UT-019 | FR-025, NFR-006, RULE-010 | AC-046 | Minimize operational deletion state and redact logs/metrics. | DID, handle, OAuth/status tokens, expected URIs, Tap receipts, record content, relationships, settings, URLs, phase/outcome values before and after terminal success. | Non-terminal state contains only approved operational fields; terminal transition removes OAuth/status/expected-URI/receipt state; only coarse approved audit/log fields remain and prohibited values never appear. | `appview/internal/accountdeletion/observability_test.go` |
| UT-020 | FR-008, FR-009, FR-010 | AC-019 | Map link/cache failures to existing safe user feedback. | Launcher false/exception, cache exception containing sensitive detail. | Navigation/account state is unchanged and only the established localized safe error is shown. | `app/test/settings/about_action_error_test.dart` |
| UT-021 | FR-018, RULE-004 | AC-014, AC-024 | Keep Sign out and Delete account as distinct action policies. | Sign-out tap and deletion tap/confirmation events. | Sign out has no confirm and invokes only existing account-session removal; Delete account cannot share that direct callback. | `app/test/settings/settings_action_policy_test.dart` |
| UT-022 | NFR-004 | AC-028 | Verify localization resource completeness for every new user-visible and semantic string. | Base ARB keys and every supported locale. | Required Settings/About/Account/confirmation/status/retry/support keys resolve through generated localizations with no production literal fallback. | `app/test/settings/settings_localizations_test.dart` |
| UT-023 | FR-004, NFR-003 | AC-022, AC-027 | Verify row semantics, forward direction, and minimum targets. | Row kinds under LTR/RTL, enabled/disabled/destructive states, constrained layouts. | Correct roles/states/labels are exposed, chevrons point forward, and interactive hit targets are at least 48x48 logical pixels. | `app/test/settings/settings_accessibility_test.dart` |
| UT-024 | FR-013, FR-019, FR-020, FR-021, NFR-001, RULE-001 | AC-025, AC-035, AC-040, AC-041 | Bind fresh reauthentication, expected handle, and deletion-only OAuth session to one DID/lease/job. | Matching/stale/canceled/expired proof, different DID, active-handle mismatch, replay, duplicate acceptance, replacement reauth. | Only a fresh matching proof advances; acceptance binds its exact OAuth session once before bearer revocation; duplicate acceptance reuses the binding; replacement requires another fresh matching proof and never restores ordinary access. | `appview/internal/auth/account_deletion_reauth_test.go`, `appview/internal/accountdeletion/oauth_binding_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, BR-002, BR-004, FR-001, FR-003, FR-004, NFR-003, NFR-004, NFR-005 | AC-001, AC-003, AC-020, AC-022, AC-028, AC-029, AC-031, AC-032, AC-033 | Render the production Settings widget with real providers/theme/localizations. | Active profiles with and without display name/avatar; compact/large and LTR/RTL harnesses. | Pump and scroll Settings. | Identity, exact sections/order, icons, error-coloured Sign out, semantics, localization, width, and fallbacks match the requirements. | `app/test/settings/settings_page_test.dart` |
| IT-002 | BR-002, FR-003, FR-005, FR-006, FR-007, FR-012, NFR-005 | AC-004, AC-005, AC-006, AC-007, AC-011, AC-029 | Exercise production Settings child routes and Back behavior. | Real `goRouterProvider` under compact and large shells. | Open every existing/new destination and go Back. | Correct page opens, active account remains, Notifications behavior is unchanged, and shell selection returns correctly. | `app/test/router/settings_routes_test.dart`, `app/test/router/notification_settings_route_test.dart` |
| IT-003 | BR-001, FR-002, FR-005, NFR-001 | AC-002, AC-006, AC-021, AC-025 | Open and use the existing switcher from Settings. | Alice/Bob registry, compact sheet and large popover, pending identity future, unsaved-work guard. | Tap Switch account and activate Bob. | Existing surface opens, guard/lease apply, route resets to Bob Home, and late Alice work is fenced. | `app/test/router/app_shell_account_switcher_test.dart`, `app/test/router/account_switch_routing_test.dart` |
| IT-004 | BR-004, FR-004, FR-007, FR-008, FR-009, FR-010, FR-011, NFR-003 | AC-007, AC-008, AC-009, AC-010, AC-019, AC-020, AC-033, AC-034 | Exercise About with shared launcher, cache fakes, package metadata, and semantics. | Recording URL launcher, two cache managers, recording messenger, package metadata variants. | Tap legal links and cache action, including repeated tap/failures. | Canonical URLs launch externally; one cache operation runs with existing feedback; icon/version/read-only semantics match shell formatting. | `app/test/settings/about_page_test.dart`, `app/test/settings/clear_image_cache_tile_test.dart` |
| IT-005 | BR-004, FR-012, FR-013, FR-014, NFR-003, NFR-004, RULE-003 | AC-005, AC-011, AC-012, AC-013, AC-027, AC-028, AC-036 | Exercise Account page and both destructive confirmation steps. | Fake successful reauth, active Alice identity, recording deletion repository. | Open Account, start deletion, inspect copy, cancel at each step, type mismatches/exact handle. | Delete account is sole destructive option without chevron; no request occurs before exact completion; copy/semantics/localization are complete. | `app/test/settings/account_page_test.dart`, `app/test/settings/delete_account_flow_test.dart` |
| IT-006 | FR-013, FR-019, FR-020, FR-021, NFR-001, RULE-001 | AC-025, AC-035, AC-040, AC-041 | Verify fresh OAuth reauthentication and deletion-only authority are server-bound to the active DID/lease/job. | `httptest` OAuth callback/proof store with matching, expired, canceled, replayed, cross-DID, duplicate acceptance, and replacement cases. | Complete or cancel reauth, switch active account, accept/duplicate the job, expire and replace the grant. | Only a fresh Alice proof permits Alice confirmation; acceptance durably binds the exact OAuth session before bearer revocation; only the worker can resume it; replacement requires fresh Alice reauth; no PDS credential reaches Flutter or ordinary APIs. | `appview/internal/auth/account_deletion_reauth_test.go`, `appview/internal/accountdeletion/oauth_binding_test.go`, `app/test/settings/delete_account_flow_test.dart` |
| IT-007 | BR-003, FR-015, NFR-002, RULE-002 | AC-015, AC-017, AC-026 | Enforce deletion-acceptance API contract, authentication, owner scope, and standard errors. | `httptest` routes with bearer/device middleware and standard envelope; Alice/Bob sessions. | Send missing/malformed/unauthorized/stale/mismatched/valid and duplicate requests. | Invalid requests mutate nothing and return camelCase `{error,message,requestId}`; valid acceptance derives Alice from auth and never accepts a client-selected Bob target. | `appview/internal/routes/account_deletion_test.go` |
| IT-008 | FR-020, NFR-002, NFR-006, RULE-010 | AC-023, AC-041, AC-046 | Persist one minimized durable job across duplicate acceptance and worker restart. | Postgres schema with fixed operation identity, bound OAuth session ID, expected-URI/receipt metadata, privacy canaries, injected worker clock, and processor barriers. | Accept twice, stop worker between phases, construct a new worker, process again, then reach terminal success. | One minimal job/binding exists without token material, record content, or unrelated data; completed phases resume safely; no destructive side effect duplicates; terminal transition removes operational state and projects only the audit. | `appview/internal/accountdeletion/store_test.go`, `appview/internal/accountdeletion/worker_test.go` |
| IT-009 | BR-003, FR-015, FR-020, FR-021, RULE-004 | AC-016, AC-040, AC-041, AC-042 | Atomically accept the job, bind deletion OAuth, revoke/remove ordinary sessions/subscriptions, and issue restricted status access. | Alice has the fresh OAuth session plus older OAuth/bearer sessions across devices/subscriptions; Bob has independent state. | Accept Alice deletion and call worker, ordinary, status, reauth-start, and PDS APIs. | The job binds the fresh OAuth session before all Alice bearer sessions and other OAuth sessions are removed; Bob is untouched; only the worker can resume the bound session; the device can use only same-job status/reauth-start/Retry. | `appview/internal/accountdeletion/acceptance_test.go`, `appview/internal/accountdeletion/oauth_binding_test.go`, `appview/internal/auth/handlers_test.go` |
| IT-010 | FR-015, FR-023, RULE-002 | AC-015, AC-017, AC-044, AC-048 | Register expected URIs and delete paginated CraftSky PDS records while preserving other namespaces/account. | Deterministic PDS fake with current CraftSky records, non-CraftSky records, shared blobs, multi-page results, and a barrier before each delete. | Run PDS deletion through completion and repeat. | Each target URI is durable before its delete call; CraftSky collections empty idempotently; non-CraftSky records, account, and shared blobs remain; no whole-account/direct-blob call is recorded. | `appview/internal/accountdeletion/pds_deleter_test.go` |
| IT-011 | FR-015, FR-020, FR-021, FR-023, NFR-002 | AC-023, AC-037, AC-041, AC-044 | Resume after partial PDS/private cleanup, already-missing records, and uncertain event receipt. | Job/PDS/private-store/Tap fakes with bound OAuth and barriers after expected-URI persistence, PDS side effect, indexer handling, receipt persistence, and phase persistence. | Crash/restart and retry every boundary. | Reprocessing converges without cross-owner effects, duplicate unsafe cleanup, lost expected deletes, or false success; only the bound worker authority is reused. | `appview/internal/accountdeletion/worker_failure_test.go` |
| IT-012 | BR-003, FR-015, FR-026, RULE-010 | AC-016, AC-037, AC-047 | Hard-delete the maintained private AppView coverage manifest. | `internal/testdb.WithSchema` seeded with Alice/Bob rows for every direct and indirect store in TD-008, shared installations/import handles, media objects/cleanup jobs, and schema manifest canaries. | Run Alice private-cleanup phase and compare schema ownership surfaces with the manifest. | Every Alice-owned row/object and owner-derived shared artifact is gone, Bob/shared controls remain, orphan-only rules are respected, and the test fails if a new private owner store has no explicit delete/retain policy. | `appview/internal/accountdeletion/private_cleanup_test.go`, `private_manifest_test.go` |
| IT-013 | FR-026, RULE-010 | AC-047 | Override ordinary Instagram retention and release the username claim. | Seed all Instagram link/import/suggestion/verification/private-import tables and claim for Alice and Bob. | Run explicit Alice deletion cleanup. | Alice's complete migration state and claim are hard-deleted; Bob and ordinary non-deletion lifecycle behavior are unchanged. | `appview/internal/accountdeletion/instagram_cleanup_test.go`, `appview/internal/instagram/retention_test.go` |
| IT-014 | FR-016, FR-022 | AC-018, AC-038, AC-043 | Apply confirming-device cleanup and MRU fallback activation. | Flutter secure registry with Alice active, Bob/Carol MRU order, Alice local product fixtures, accepted-job response. | Reconcile acceptance. | Alice product data/token clear, Bob activates, minimal Alice status row remains disabled, and completion removes only that row/status state. | `app/test/settings/account_deletion_controller_test.dart` |
| IT-015 | FR-016, FR-021, FR-022 | AC-018, AC-038, AC-042, AC-043 | Apply acceptance when no other local account remains. | Alice-only registry and local data with restricted credential response. | Reconcile acceptance and later terminal success. | Status becomes primary, ordinary routes stay unavailable, only status/Retry work, and success clears the remaining binding/identity. | `app/test/settings/account_deletion_controller_test.dart`, `app/test/router/router_redirect_test.dart` |
| IT-016 | FR-017, FR-027, RULE-006 | AC-023, AC-039 | Drive bounded automatic retry into attention-required and manual Retry. | Deterministic scheduler, retryable/permanent failures, recording status API/UI. | Exhaust automatic attempts, inspect UI, trigger Retry. | No cancel/ordinary activation exists; status becomes `Deletion needs attention` with support guidance; one authorized Retry resumes the same job. | `appview/internal/accountdeletion/worker_retry_test.go`, `app/test/settings/deletion_status_page_test.dart` |
| IT-017 | FR-021, NFR-006 | AC-040, AC-042 | Enforce separation of status, ordinary, PDS, and deletion-worker authorization. | Alice/Bob jobs, restricted credentials, bound/unbound OAuth sessions, and route/worker middleware. | Call status, reauth-start, Retry, ordinary/PDS endpoints, and worker resume with each variant. | Only same-job status/reauth-start/Retry and the matching worker-bound OAuth resume succeed; cross-job/owner, ordinary/PDS client, token-disclosure, expired, and revoked uses fail without leakage. | `appview/internal/routes/account_deletion_status_test.go`, `appview/internal/accountdeletion/oauth_binding_test.go` |
| IT-018 | FR-023, RULE-008 | AC-044 | Replay existing delete handlers and record receipts only after successful indexing. | Seed indexed content/derived notifications, expected URI rows, duplicate/reordered Tap events with IDs/revisions, and injected indexer/receipt failures. | Dispatch events through the indexer plus receipt observer. | Existing handlers converge idempotently; a receipt is stored only after successful handling and before ack; either failure leaves the event unacknowledged for replay; receipts never mutate public state or invoke eager purge. | Existing `appview/internal/index/craftsky_*_test.go`, `notification_lifecycle_test.go`, and `appview/internal/accountdeletion/convergence_observer_test.go` |
| IT-019 | FR-023, FR-027, RULE-007, RULE-008 | AC-037, AC-044 | Keep the job non-terminal until receipt-backed AppView convergence. | Private/session phases complete; expected URI set; indexed effects and receipts independently missing/present; final PDS rescan empty/non-empty; deletion OAuth present/removed. | Process the job across missing, duplicate, reordered, receipt-failure, Tap-outage, newly discovered record, and complete states. | The job waits until every expected URI is receipted, its indexed/derived effects are absent/retracted, the final rescan is empty, and deletion OAuth is removed; unavailable convergence eventually requires attention; no eager-hide mutation occurs. | `appview/internal/accountdeletion/convergence_test.go` |
| IT-020 | FR-024, RULE-011 | AC-045 | Route authentication to pending status or fresh onboarding after success. | Login/OAuth completion for a DID with pending/attention/successful job and deleted product fixtures. | Authenticate before and after terminal success. | Pending/attention yields status-only access; success permits a new membership and restores nothing. | `appview/internal/accountdeletion/rejoin_test.go`, `app/test/auth/providers/auth_controller_test.dart` |
| IT-021 | FR-025, NFR-006, RULE-010 | AC-046 | Persist the minimized audit and expire it from terminal success. | Postgres job containing prohibited source fields, stored `terminalSuccessAt`, injected clock immediately before/at/after `terminalSuccessAt + 30 days`, optional fresh membership. | Transition to terminal success, verify operational state removal, and run expiry at each boundary. | Only allowed audit columns persist after success; job/OAuth/status/expected-URI/receipt state is gone; audit exists before and is absent at/after the exact boundary independently of a fresh membership. | `appview/internal/accountdeletion/audit_test.go` |
| IT-022 | RULE-009 | AC-036, AC-048 | Prove blob handling is reference-only and not a success gate. | PDS fake records a shared blob and exposes a delayed/absent GC signal. | Delete CraftSky references and evaluate terminal status. | No direct blob delete is called; shared reference survives; job may succeed without waiting for GC. | `appview/internal/accountdeletion/blob_boundary_test.go` |
| IT-023 | NFR-006 | AC-046 | Verify deletion observability is coarse and redacted. | Recording logger/metrics with sensitive fake values and every job outcome. | Run success, retry, attention, convergence delay, and audit expiry. | Required coarse phases/counters appear; DIDs, handles, tokens, URLs, record content, relationships, imports, and settings do not. | `appview/internal/accountdeletion/observability_test.go` |
| IT-024 | RULE-005 | AC-030 | Fail when a new CraftSky record Lexicon is missing from deletion inventory. | Walk `lexicon/social/craftsky/` plus deletion registry. | Compare every primary `record` NSID to registered deleters. | Sets match exactly; defs-only schemas are ignored; a synthetic/unregistered record makes the test fail. | `appview/internal/accountdeletion/collections_test.go` |
| IT-025 | FR-022 | AC-043 | Clean an offline secondary device at next contact. | Persist local Alice product data plus a server response indicating accepted deletion; start app offline then reconnect/launch. | Initialize active-account/session state. | Ordinary Alice access is blocked and local product data clears on first learned deletion state, preserving only permitted status data. | `app/test/settings/account_deletion_device_cleanup_test.dart` |
| IT-026 | NFR-001, RULE-001 | AC-025 | Reject stale client completion after another account activates. | Alice deletion request completer, Bob retained and activated before response. | Complete Alice acceptance/status/error after Bob activation. | Alice state updates only its status binding; Bob is not deleted/removed/navigated away and receives no Alice success/error. | `app/test/settings/account_deletion_lease_test.dart` |
| IT-027 | BR-003, FR-015, FR-020, FR-021, FR-023, RULE-007 | AC-015, AC-016, AC-037, AC-040, AC-044 | Run the complete job against Postgres, deterministic PDS/OAuth/session, and Tap/indexer/receipt fakes. | Alice has every current CraftSky record/private-manifest category/session plus Bob/non-CraftSky controls, a fresh OAuth session, event IDs/revisions, and failure barriers. | Accept and process through terminal success, including one duplicate/reordered Tap event and one restart. | OAuth binds before bearer revocation; only the worker uses it; expected deletes are receipted after indexer success; final rescan/effects/private gates pass; OAuth/operational state is removed; only allowed audit remains; Bob, Alice's AT account, and non-CraftSky records survive. | `appview/internal/accountdeletion/worker_acceptance_test.go` |
| IT-028 | FR-020, NFR-002 | AC-041 | Reconcile app-close/network-loss/duplicate-acceptance uncertainty from the Flutter client. | Acceptance response withheld after server commits, local retry/restart, job lookup API. | Close/restart app and resubmit/recover. | Client resolves the same job, obtains status-only state, and never creates duplicate jobs or restores an ordinary session. | `app/test/settings/account_deletion_repository_test.dart`, `appview/internal/routes/account_deletion_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Existing Settings destinations remain available, including adjacent Customisation work. | BR-002, FR-003 | AC-003, AC-004, AC-005, AC-032 | Extend `app/test/settings/settings_page_test.dart` and `settings_routes_test.dart` to assert every preserved row appears exactly once and reaches its existing typed destination. |
| REG-002 | Existing Notifications settings controls, account scope, permission guidance, retry, and errors are unchanged. | BR-002, FR-006 | AC-004, AC-006 | Run/extend `app/test/notifications/notification_settings_page_test.dart` and `app/test/router/notification_settings_route_test.dart` after adding the Settings entry. |
| REG-003 | Clear image cache still clears both scopes once and uses established feedback. | BR-004, FR-010 | AC-007, AC-009, AC-019 | Keep `app/test/settings/clear_image_cache_tile_test.dart` passing after moving the tile into About; add repeated-tap/no-confirm coverage. |
| REG-004 | Sign out removes only the active retained account and preserves membership/data. | BR-004, FR-018, RULE-004 | AC-014, AC-024 | Keep `app/test/settings/sign_out_tile_test.dart` and active-account fallback tests passing; assert the new error colour/no-confirm presentation does not call deletion APIs. |
| REG-005 | Existing account switcher preserves one-account Add account, five-account limit, unsaved-work guard, MRU ordering, and form-factor presentation. | FR-002 | AC-002, AC-021 | Extend `app/test/auth/models/account_switcher_state_test.dart` and `app/test/router/app_shell_account_switcher_test.dart` without replacing existing behaviors. |
| REG-006 | Drawer and navigation-rail build labels retain their existing localized value. | FR-011 | AC-010, AC-034 | Assert adding About does not change shell `navigationBuildVersion` rendering and About reads the same provider/formatter. |
| REG-007 | External legal link failures stay on the current surface and show safe feedback. | FR-008, FR-009 | AC-008, AC-019 | Keep shell external-link tests and add equivalent About failure cases without navigation/account changes or raw exception text. |
| REG-008 | Existing post, like, repost, and profile indexers remain idempotent and preserve their current notification/lifecycle side effects. | FR-023 | AC-044 | Run existing `appview/internal/index/craftsky_*_test.go`, `notification_lifecycle_test.go`, and profile membership lifecycle suites with duplicate/reordered delete events. |
| REG-009 | CraftSky deletion never invokes whole-PDS deletion or deletes non-CraftSky records/shared blobs. | BR-003, RULE-002, RULE-009 | AC-015, AC-017, AC-036, AC-048 | Use a recording PDS fake/canary account and fail on any account-delete, other-namespace delete, or direct shared-blob delete call. |
| REG-010 | Public AppView deletion remains indexer-driven, with no new eager-hide/read filter. | FR-023, RULE-008 | AC-044 | Query a deliberately lagging direct-read fixture before Tap convergence, verify allowed temporary visibility, and assert only indexer events remove it. |
| REG-011 | Flutter never receives or stores PDS OAuth access/refresh tokens, passwords, or emailed confirmation codes. | FR-019 | AC-035, AC-040 | Extend auth architecture/storage tests to inspect client models, secure storage, request DTOs, and logs for forbidden PDS credentials. |
| REG-012 | Rejoining after terminal success creates fresh product state rather than restoring deleted membership data. | FR-024, RULE-011 | AC-045 | Seed prior content/preferences/imports, complete deletion, rejoin the DID, and assert default fresh membership state while the AT account remains unchanged. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Active identity and multi-account fixtures. | Alice `did:plc:alice` / `alice.test` with avatar/display name; Alice without display name/avatar; Bob and Carol with deterministic MRU order; no raw DID shown in UI assertions. | AT-001, AT-002, AT-006, AT-007, UT-001, UT-012, IT-001, IT-003, IT-014, IT-015 |
| TD-002 | Responsive, text-direction, and semantics matrix. | Compact 500x800, large 1200x800, LTR and RTL locales, keyboard/touch/semantics harnesses. | AT-001, AT-002, AT-015, UT-002, UT-023, IT-001, IT-002, IT-003, MAN-001, MAN-002 |
| TD-003 | Package metadata variants. | `1.2.3`/`123`, empty build number, missing/unavailable metadata, shared shell/About provider. | AT-004, UT-003, IT-004, REG-006 |
| TD-004 | About action fakes. | Recording external launcher for canonical Terms/Privacy URLs, launcher false/throw, two recording cache managers, delayed cache completion, sanitized exception. | AT-004, UT-020, IT-004, REG-003, REG-007 |
| TD-005 | Reauthentication, OAuth binding, and confirmation cases. | Fresh/stale/expired/canceled/replayed proof for Alice; proof for Bob; exact and mismatched handles; active-lease barrier; fresh/older/bound/unbound/expired OAuth session IDs; replacement reauth. | AT-005, AT-006, AT-008, AT-011, UT-004, UT-007, UT-011, UT-015, UT-024, IT-005, IT-006, IT-009, IT-017, IT-026, IT-027 |
| TD-006 | PDS deletion boundary. | Current CraftSky profile/post/like/repost records including standard/project/reply/quote posts; multi-page data; already-missing records; defs-only and synthetic future record Lexicons; `app.bsky.*` and third-party records; shared/unreferenced blobs; recording whole-account/blob methods. | AT-009, UT-009, UT-010, IT-010, IT-011, IT-022, IT-024, IT-027, REG-009 |
| TD-007 | Durable job and failure barriers. | Fixed job/operation IDs, every phase, injected clocks, retry counts, transient/permanent errors, barriers after side effect and before persistence, duplicate acceptance, worker restart, OAuth expiry/replacement, receipt failure, and Tap outage. | AT-008, UT-006, UT-007, UT-008, IT-008, IT-011, IT-016, IT-018, IT-019, IT-027, IT-028 |
| TD-008 | Complete private-data coverage manifest. | Alice/Bob fixtures for auth/session rows; recent searches; mutes; saved folders/posts; language preferences; pins; recipient notification events/preferences/seen state/subscriptions/deliveries with shared-installation controls; scheduled posts/media/tombstones, object-store media, and object-key cleanup jobs; reporter/source/subject moderation rows; current/adjacent customisation stores; every Instagram store in TD-009; plus schema ownership canaries that require an explicit delete/retain rule for any new owner-private table/service. Public/indexer-owned and shared non-owner controls are seeded to prove preservation. | AT-010, IT-009, IT-012, IT-027 |
| TD-009 | Instagram and username-claim cleanup. | Alice/Bob verification attempts, links, identity claims, reachable conflicts/webhook work, graph imports, shared/orphan handles, suggestions/sources, reconciliation jobs, PDS follow operations, owner-derived rate-limit buckets, audit events, notification-suggestion joins/effects, ordinary retention state, and username claims. | AT-010, UT-018, IT-012, IT-013, IT-027 |
| TD-010 | Session and restricted-status authorization. | Several Alice ordinary bearer/OAuth sessions, the bound deletion OAuth session, devices/subscriptions, Bob controls, matching/cross-job/cross-owner/expired/revoked status credentials, worker identity, and ordinary background selector. | AT-006, AT-007, AT-011, UT-011, UT-024, IT-006, IT-009, IT-015, IT-017, IT-027 |
| TD-011 | Tap/indexer convergence. | Expected URI rows registered before deletion; seeded profile/post/like/repost and derived notification state; Tap delete events with owner/URI/action/event-ID/revision; missing, duplicate, reordered, delayed, handler-failing, and receipt-failing events; empty/non-empty final PDS rescans. | AT-012, UT-008, IT-018, IT-019, IT-027, REG-008, REG-010 |
| TD-012 | Audit expiry and fresh rejoin. | Allowed/prohibited audit fields; stored terminal-success timestamp; clocks immediately before/at/after `terminalSuccessAt + 30 days`; non-terminal operational/OAuth/receipt state; fresh membership created before old audit expiry. | AT-010, AT-013, UT-016, UT-017, UT-019, IT-020, IT-021, REG-012 |
| TD-013 | Device-local deletion state. | Drafts, staged media, caches, secure ordinary tokens, cached identity/product state, job binding/status credential, confirming and offline secondary devices. | AT-006, AT-007, UT-014, IT-014, IT-015, IT-025, IT-028, MAN-004 |
| TD-014 | Privacy canaries. | Unique fake DID, handle, OAuth/status tokens, expected URI, Tap receipt, content text, relationship, preference, import, setting, and full URL values for schema/log/response searches before and after terminal success. | UT-017, UT-019, IT-008, IT-017, IT-021, IT-023 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-005 | AC-029 | Final compact/large visual hierarchy and responsive behavior. | Run the app on compact iOS/Android and large tablet/desktop sizes; inspect Settings, Account, About, both confirmation steps, switcher status rows, and deletion status in light/dark themes with text scaling. | Identity/sections/actions are visually coherent, error colours are correct, content scrolls without clipping, maximum width/spacing match CraftSky, and no surface looks like an accidental generic list. |
| MAN-002 | NFR-003 | AC-022, AC-027 | Real assistive-technology and keyboard operation. | Use VoiceOver/TalkBack and desktop keyboard focus to traverse disclosure rows, external links, direct actions, destructive controls, typed confirmation, disabled deleting row, status, Retry, and support guidance in LTR/RTL. | Spoken roles/states/labels are accurate, focus order follows visual order, disabled/attention/destructive meaning is clear, controls operate without touch-only gestures, and targets are comfortably tappable. |
| MAN-003 | BR-003, FR-021, FR-023, RULE-002, RULE-007, RULE-009 | AC-015, AC-016, AC-017, AC-037, AC-040, AC-044, AC-048 | Disposable real OAuth/PDS/Tap end-to-end deletion. | Using a disposable development AT account only, seed current CraftSky and non-CraftSky records, complete fresh OAuth/typed confirmation, stop/restart app or worker once, observe ordinary-session rejection plus PDS/AppView/receipt state through terminal status, then rejoin. Never use a personal/production account. | Only the worker retains PDS authority; CraftSky records/private state disappear; non-CraftSky records and AT account survive; ordinary sessions revoke; status persists; receipt-backed Tap convergence gates success; deletion OAuth is removed; and rejoin is fresh. No whole-account or direct shared-blob deletion occurs. |
| MAN-004 | FR-022 | AC-036, AC-043 | Offline secondary-device next-contact cleanup. | Sign the same disposable account into two devices, store drafts/cache on both, keep one offline, accept deletion on the other, then launch/reconnect the offline device. | Confirming device cleans immediately; offline device cannot use the account and clears local product state on first contact while showing only permitted deletion status until success. Confirmation copy did not promise earlier erasure. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Exact REST paths, payloads, success statuses, fresh-reauth proof/window, status-credential format, schema layout, and retry counts/delays are not yet designed. | FR-013, FR-017, FR-019, FR-020, FR-021, FR-027, NFR-002 | The approved behavior now fixes the credential lifecycle, but wire and storage literals remain coding-design details. | Resolve in `04-coding-plan.md`; update automation targets/contracts without changing accepted product semantics or IDs. |
| GAP-002 | Receipt-backed AppView convergence requires new observation/storage wiring around the existing indexer dispatcher. | FR-023, RULE-007, RULE-008 | The contract is now defined, but exact schema/interface placement remains a coding-design choice. | In `04-coding-plan.md`, design expected-URI and receipt stores plus a post-handler/pre-ack observer; preserve the fixed event-ID/revision, failure/replay, final-rescan, and no-eager-purge semantics in AT-012, IT-018, and IT-019. |
| GAP-003 | Exact support destination and final localized status/deletion wording remain open. | FR-014, FR-017, FR-027, NFR-004 | Tests can assert required facts and presence of guidance, but not a final URL/label until content design chooses it. | Resolve during coding planning or manual comment; then pin exact localization keys/copy where product-visible wording is required. |
| GAP-004 | The private owner-DID manifest can drift as adjacent features land. | FR-015, FR-026, RULE-010 | The current direct/indirect store inventory is explicit in TD-008/TD-009, but future migrations can add ownership surfaces. | In `04-coding-plan.md`, design one maintained coverage manifest and the IT-012 schema/completeness test that fails when a new owner-private store lacks an explicit delete/retain rule. |
| GAP-005 | PDS behavior and OAuth reauthentication vary across real providers and cannot be proven solely with deterministic fakes. | FR-015, FR-019, RULE-002, RULE-009 | Repository automation cannot safely destructively exercise every provider. | Keep fake/contract tests exhaustive and require MAN-003 against disposable supported development PDSes before release. |
| GAP-006 | Offline-device physical storage may persist in OS backups or an unreachable device beyond CraftSky's control. | FR-014, FR-022 | The requirement promises cleanup at next app contact, not remote physical erasure or backup deletion. | Test next-contact behavior and preserve the explicit confirmation limitation; do not expand the promise. |
| GAP-007 | Customisation appears in the supplied screenshot but is absent from this checkout. | FR-003, FR-004 | The final test cannot exercise a production route that has not yet merged here. | Keep the required row/route case in REG-001 and resolve the adjacent-work merge order during document review/coding planning. |
| GAP-008 | User-authorized CraftSky PDS record deletion is an approved narrow exception that is not yet reflected in repository guidance. | BR-003, RULE-002, RULE-005, RULE-009 | The product decision and safety boundary are now explicit, but the checked-in general guidance still says not to delete from a user's PDS. | Coding plan must place the guidance/reference amendment before destructive route enablement; REG-009 and MAN-003 remain release gates. |

## 10. Out Of Scope

- Tests for deleting, deactivating, or modifying the user's DID, PDS account, general AT Protocol identity, or any non-`social.craftsky.*` record; those operations are explicitly forbidden.
- Tests for direct blob deletion, immediate PDS garbage collection, or remote deletion from a device that never reconnects.
- Tests for deletion-job cancellation, temporary CraftSky deactivation, membership restoration, or permanent re-enrolment bans.
- Tests for redesigning Notifications settings, Customisation, Languages, relationship/moderation lists, ordinary Instagram migration behavior, or the existing account-switcher presentation.
- Tests for account editing, handle/password management, data export, active-session management, sign-out-all UI, update checking, or an in-app legal browser.
- Production destructive tests against personal accounts or uncontrolled PDSes.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-08-10-settings-page/`
- Risk level: High; formal document re-review completed with `Approved with notes`, so coding planning may proceed while preserving the listed plan/release gates.
- Recommended first failing test for implementation: `UT-009`, which defines the complete `social.craftsky.*` record-collection inventory and excludes defs-only schemas before any destructive worker code exists.
- Suggested test order for implementation:
  1. Destructive namespace boundary (`UT-009`), then pure confirmation/lease and non-destructive Settings tests (`UT-001` through `UT-005`, `UT-015`, `UT-020` through `UT-023`).
  2. Deletion state, OAuth/status authorization, terminal gates, audit, and cleanup policies (`UT-006` through `UT-008`, `UT-010` through `UT-014`, `UT-016` through `UT-019`, `UT-024`).
  3. Fresh OAuth binding, API acceptance, durable schema/store, atomic bearer revocation, restricted authorization, audit/observability, and collection completeness (`IT-006` through `IT-009`, `IT-017`, `IT-021`, `IT-023`, `IT-024`).
  4. Expected-URI PDS deletion, private/Instagram manifests, worker failure/retry, and blob boundary (`IT-010` through `IT-013`, `IT-016`, `IT-022`).
  5. Existing indexer replay, post-handler/pre-ack receipts, convergence, and full worker acceptance (`IT-018`, `IT-019`, `IT-027`).
  6. Flutter Settings/About/Account, routing, local cleanup, multi-account status, pending-login, stale-lease, and uncertain-acceptance recovery (`IT-001` through `IT-005`, `IT-014`, `IT-015`, `IT-020`, `IT-025`, `IT-026`, `IT-028`).
  7. Regression suites, then manual release checks.
- Commands discovered:
  - From the repository root with the compose stack running: `just test`.
  - Focused Flutter tests: `just app-test test/settings test/router/notification_settings_route_test.dart test/router/settings_routes_test.dart test/router/app_shell_account_switcher_test.dart test/auth/models/account_switcher_state_test.dart`.
  - Flutter analysis: `just app-analyze`.
  - Flutter generation when routes/providers/localizations change: `cd app && dart run build_runner build`, plus the repository localization-generation path.
- Blocking gaps: None for coding planning. `GAP-001`, `GAP-002`, `GAP-003`, `GAP-004`, `GAP-007`, and the guidance amendment in `GAP-008` are concrete coding-plan inputs rather than unresolved product decisions. `GAP-005` and `GAP-006` remain controlled release/manual boundaries.
