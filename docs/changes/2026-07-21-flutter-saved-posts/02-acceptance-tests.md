# Acceptance Test Specification: Flutter Saved Posts

## 1. Test Strategy

This is a medium-risk Flutter and AppView change spanning private account state, destructive persistence, pagination, routing, and regression-sensitive UI extraction. The test design uses the repository's existing Flutter widget, Riverpod provider, Dio adapter, repository fake, router, and real-Postgres Go test patterns.

- Unit tests cover wire/model decoding, explicit nullable folder assignment, folder-name validation parity, saved-state reconciliation, pagination merging, overview composition, exact-post navigation, compact-summary adapters, error sanitization, and AppView delete-query parsing.
- Integration tests cover every consumed Flutter API/repository operation, independent folder/unfiled/folder-content state, confirmation-driven save/move, optimistic unsave rollback, account switching, typed routing, and both transactional AppView folder-delete modes.
- Acceptance widget and handler scenarios cover the user-visible bookmark, chooser, Settings collection, folder management, post-summary consumers, accessibility semantics, privacy boundaries, and destructive choices.
- Regression tests protect existing full-post actions, quote moderation/reveal behavior, notification context and destinations, profile privacy, AppView unfile-on-delete behavior, post data/counts, and account isolation.
- Two manual checks complement automation for real VoiceOver/TalkBack behavior and device-size/text-scale layout. They do not replace automated semantics and focus-order assertions.

All 32 acceptance criteria and every Must requirement have an automated verification path. Real-Postgres AppView cases require `TEST_DATABASE_URL`; a skipped database test is not sufficient release evidence. No blocking test gap was found. The risk level remains **Medium** because late cross-account completions and atomic deletion failures need deliberate concurrency/rollback coverage. Document review completed with an `Approved with notes` verdict; this revision applies its traceability, folder-mutation reconciliation, scenario-splitting, and dependency-diff notes.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-006 | AT-001, AT-002, AT-003 | Acceptance | Yes |
| BR-002 | AC-007, AC-008, AC-011 | AT-004, AT-006 | Acceptance | Yes |
| BR-003 | AC-013, AC-014 | AT-007, AT-008, IT-010, IT-011 | Acceptance / Integration | Yes |
| BR-004 | AC-020, AC-021, AC-022 | AT-010–AT-012 | Acceptance | Yes |
| BR-005 | AC-025, AC-026 | AT-008, AT-009, IT-006, IT-012 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-024 | AT-001, UT-001, REG-005 | Acceptance / Unit / Regression | Yes |
| FR-002 | AC-001, AC-023, AC-032 | AT-001, AT-010, AT-013, REG-001, REG-005 | Acceptance / Regression | Yes |
| FR-003 | AC-002, AC-006, AC-029 | AT-001–AT-003, IT-005 | Acceptance / Integration | Yes |
| FR-004 | AC-003, AC-004, AC-017, AC-030 | AT-002, UT-003, UT-013, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-005, AC-017, AC-030 | AT-002, AT-007, UT-003, UT-013, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-006, AC-018, AC-024, AC-029, AC-030 | AT-002, AT-003, AT-006, AT-009, UT-004, IT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-027 | UT-002, IT-001, IT-002, AT-013 | Unit / Integration / Acceptance | Yes |
| FR-008 | AC-007, AC-026 | AT-004, AT-009, UT-011, IT-007, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| FR-009 | AC-008, AC-009, AC-028 | AT-004, AT-005, UT-006, IT-003, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-009, AC-010, AC-019, AC-028, AC-031 | AT-004, AT-005, UT-005, IT-003, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-011, AC-012, AC-030 | AT-006, UT-007, UT-014, IT-004, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-011 | AT-006, UT-007, IT-008, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-013 | AC-005, AC-013, AC-017, AC-031 | AT-004, AT-005, AT-007, UT-003, UT-005, UT-011, IT-003, IT-007, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-013, AC-023 | AT-007, AT-013, IT-008 | Acceptance / Integration | Yes |
| FR-015 | AC-014, AC-015, AC-016, AC-025 | AT-008, UT-009, UT-012, IT-001, IT-009–IT-012, REG-006 | Acceptance / Unit / Integration / Regression | Yes |
| FR-016 | AC-014, AC-025 | AT-008, IT-011, IT-012, REG-008 | Acceptance / Integration / Regression | Yes |
| FR-017 | AC-015 | AT-008, IT-001, IT-011 | Acceptance / Integration | Yes |
| FR-018 | AC-018, AC-024, AC-026 | AT-003, AT-006, AT-009, UT-004, IT-005, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-019 | AC-026 | AT-009, IT-006, REG-009 | Acceptance / Integration / Regression | Yes |
| FR-020 | AC-020, AC-021, AC-022, AC-023, AC-032 | AT-010–AT-013, UT-008 | Acceptance / Unit | Yes |
| FR-021 | AC-020 | AT-010, UT-008, REG-003 | Acceptance / Unit / Regression | Yes |
| FR-022 | AC-021 | AT-011, UT-007, REG-004 | Acceptance / Unit / Regression | Yes |
| FR-023 | AC-022 | AT-012, UT-008, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-024 | AC-023 | AT-001, AT-002, AT-007, AT-010–AT-013, MAN-001, MAN-002 | Acceptance / Manual | Yes, plus manual |
| FR-025 | AC-019, AC-025 | AT-005, AT-013, UT-010, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-026 | AC-004, AC-017, AC-025 | AT-002, AT-005, AT-007, UT-003, UT-005, IT-001, IT-003, IT-012 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-025, AC-026 | AT-008, AT-009, AT-013, UT-010, IT-006, IT-012, REG-008, REG-009 | Acceptance / Unit / Integration / Regression | Yes |
| NFR-002 | AC-009, AC-010 | AT-005, UT-005, IT-003 | Acceptance / Unit / Integration | Yes |
| NFR-003 | AC-006, AC-018, AC-029, AC-030 | AT-002, AT-003, AT-006, UT-004, IT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| NFR-004 | AC-020, AC-021, AC-022 | AT-010–AT-012, UT-008, REG-003, REG-004 | Acceptance / Unit / Regression | Yes |
| NFR-005 | AC-027 | AT-013, REG-001, REG-007 | Acceptance / Regression | Yes |
| NFR-006 | AC-027 | AT-013 | Acceptance | Yes |
| RULE-001 | AC-003, AC-030 | AT-002, UT-013, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-002, AC-006, AC-029 | AT-001, AT-003 | Acceptance | Yes |
| RULE-003 | AC-013, AC-014 | AT-007, AT-008, IT-010, IT-011, REG-006 | Acceptance / Integration / Regression | Yes |
| RULE-004 | AC-005 | AT-002, UT-013, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-005 | AC-008, AC-012, AC-013 | AT-004, AT-006, UT-014, IT-004, IT-010 | Acceptance / Unit / Integration | Yes |
| RULE-006 | AC-007, AC-025 | AT-004, AT-008, IT-007, REG-002 | Acceptance / Integration / Regression | Yes |
| RULE-007 | AC-004, AC-017 | AT-002, AT-007, UT-003 | Acceptance / Unit | Yes |
| RULE-008 | AC-008, AC-028, AC-031 | AT-004, AT-005, UT-006, UT-014, IT-003 | Acceptance / Unit / Integration | Yes |
| RULE-009 | AC-026 | AT-009, IT-006, REG-009 | Acceptance / Integration / Regression | Yes |
| RULE-010 | AC-028 | AT-004, AT-005, UT-006, IT-008 | Acceptance / Unit / Integration | Yes |
| RULE-011 | AC-028, AC-031 | AT-004, AT-007, UT-006, IT-008 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Render And Invoke The Bookmark On Every Eligible Full Post

Requirement IDs: BR-001, FR-001, FR-002, FR-003, FR-024, RULE-002
Acceptance Criteria: AC-001, AC-002, AC-023, AC-032
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Full-post bookmark
  Scenario Outline: An eligible full post exposes private saved state
    Given authenticated post data for an eligible <postType>
    And viewerHasSaved is <saved> with the matching nullable folder state
    When the full PostCard renders
    Then a localized <iconState> bookmark appears immediately before overflow
    And it has selected semantics matching <saved>
    And it exposes no public save count

    Examples:
      | postType | saved | iconState |
      | ordinary top-level post | false | outlined |
      | project post | true | filled |
      | quote post | false | outlined |
      | comment | true | filled |
      | nested reply | false | outlined |

  Scenario: An unsaved bookmark opens the chooser without saving
    Given an eligible unsaved full post
    When its outlined bookmark is tapped
    Then the save chooser opens
    And no save request is sent until the separate Save action is pressed

  Scenario: A saved bookmark starts optimistic unsave immediately
    Given an eligible full post saved in folder A
    When its filled bookmark is tapped
    Then the bookmark becomes outlined immediately
    And exactly one unsave request starts without confirmation or Undo

  Scenario: Ineligible compact and protected presentations omit the bookmark
    Given a protected post placeholder, a quote summary, and a notification summary
    When each renders
    Then none contains a bookmark or engagement control
```

### AT-002: Confirm A Save With No Folder, An Existing Folder, Or A New Folder

Requirement IDs: BR-001, FR-003, FR-004, FR-005, FR-006, FR-024, FR-026, NFR-003, RULE-001, RULE-004, RULE-007
Acceptance Criteria: AC-002, AC-003, AC-004, AC-005, AC-006, AC-017, AC-023, AC-030
Priority: Must
Level: Acceptance
Automation Target: `app/test/saved_posts/widgets/save_post_dialog_test.dart`

```gherkin
Feature: Save chooser
  Scenario: Save remains explicit while folders load incrementally
    Given an unsaved post and multiple folder pages including duplicate names
    When the chooser opens
    Then No folder is selected and exactly one option can be selected
    And no save request has been sent
    When the user loads another folder page
    Then the opaque cursor is forwarded and duplicate names remain distinct by ID
    When the user selects a folder and presses Save
    Then one save request uses that folder ID
    And the dialog remains open with one busy disabled confirm action
    And confirmed success closes silently without a success snackbar

  Scenario: Folder loading can fail without blocking an unfiled save
    Given the folder list request fails
    When the chooser renders
    Then an inline localized Retry is shown for folders
    And No folder remains selectable and savable
    And no client-only folder search is offered

  Scenario: Inline folder creation is independent of the post save
    Given the chooser is open
    When New folder expands and a valid name is submitted
    Then the form is busy once, the folder persists, the form collapses, and the new ID is selected in server order
    When the user cancels the chooser
    Then no post save occurs and the created folder remains
    When folder creation fails
    Then the field remains editable with a localized retryable error and the chooser stays open
```

### AT-003: Optimistically Unsave Once And Roll Back Exactly On Failure

Requirement IDs: BR-001, FR-003, FR-006, FR-018, NFR-003, RULE-002
Acceptance Criteria: AC-006, AC-018, AC-024, AC-029
Priority: Must
Level: Acceptance
Automation Target: `app/test/saved_posts/providers/saved_post_state_provider_test.dart`, `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Optimistic unsave
  Scenario: Successful unsave updates every loaded presentation once
    Given one saved URI in folder A is visible on multiple full-post surfaces and in Saved posts
    When the user taps any filled bookmark twice rapidly
    Then every surface becomes outlined immediately
    And exactly one DELETE is sent
    And no confirmation or Undo is shown
    When the server returns 204
    Then the confirmed state remains unsaved and the saved row disappears

  Scenario: Failed unsave restores the exact confirmed state
    Given the URI was saved in folder A
    When optimistic unsave fails
    Then every loaded surface restores saved=true and folder A
    And localized sanitized failure feedback is shown
    And a stale canonical post response cannot overwrite a newer confirmed result
```

### AT-004: Enter The Private Saved Overview From Settings

Requirement IDs: BR-002, FR-008, FR-009, FR-010, FR-013, RULE-005, RULE-006, RULE-008, RULE-010, RULE-011
Acceptance Criteria: AC-007, AC-008, AC-028, AC-031
Priority: Must
Level: Acceptance
Automation Target: `app/test/router/saved_posts_route_test.dart`, `app/test/settings/settings_page_test.dart`, `app/test/saved_posts/pages/saved_posts_page_test.dart`

```gherkin
Feature: Saved collection entry and hierarchy
  Scenario: Settings is the sole private entry point
    Given a signed-in user is in Settings
    When they tap the localized Saved posts tile
    Then a typed full-screen /profile/settings/saved route opens at the root overview
    And Back returns to Settings
    And no Saved tab exists on either an own or visited profile
    And the former /profile/saved placeholder is not canonical

  Scenario: The overview separates folders from unfiled saves
    Given alphabetical folders, foldered saves, and unfiled saves
    When the overview renders
    Then all folder rows appear first in server order without counts
    And only unfiled saves appear beneath them
    And sort changes reorder only visible posts by savedAt
    And Add folder is available in the app bar
    And the overview never restores a previously open folder

  Scenario: Empty sections follow the collection hierarchy
    Given folders exist and Unfiled is empty
    Then the Unfiled empty section is hidden and folders remain visible
    Given neither folders nor unfiled saves exist
    Then the full Nothing saved yet state is shown
```

### AT-005: Page, Refresh, Retry, And Restart Each Saved Resource Independently

Requirement IDs: FR-009, FR-010, FR-013, FR-025, FR-026, NFR-002, RULE-008, RULE-010
Acceptance Criteria: AC-009, AC-010, AC-017, AC-019, AC-028, AC-031
Priority: Must
Level: Acceptance
Automation Target: `app/test/saved_posts/providers/saved_posts_provider_test.dart`, `app/test/saved_posts/pages/saved_posts_page_test.dart`, `app/test/saved_posts/pages/saved_post_folder_page_test.dart`

```gherkin
Feature: Bounded saved collection state
  Scenario Outline: A list has independent cursor and failure state
    Given <resource> has more than one page
    When its next page is requested
    Then its opaque cursor is round-tripped without parsing
    And returned IDs append once in server order
    And no per-item folder lookup or post request occurs
    When incremental loading fails
    Then confirmed rows remain and only that section exposes Retry
    When the server reports invalid_cursor
    Then only that scope and sort restarts safely from page one
    When refresh runs
    Then stale cursor/error state is replaced without duplicating rows

    Examples:
      | resource |
      | folder rows |
      | unfiled newest posts |
      | unfiled oldest posts |
      | one folder's newest posts |
      | one folder's oldest posts |

  Scenario: Folder pagination never pushes folders below unfiled rows
    Given additional folder pages and unfiled posts are both available
    When either resource loads
    Then every loaded folder remains above every unfiled post
    And foldered posts never appear in the overview

  Scenario Outline: A confirmed folder mutation safely reconciles partial pagination
    Given only the first folder page is loaded with a non-terminal opaque cursor
    And the affected folder is identified by opaque ID rather than display name
    When <mutation> succeeds
    Then the folder collection reconciles or restarts from page one in server order
    And the prior unsafe cursor is discarded
    And every opaque folder ID appears at most once
    And the overview keeps folders above unfiled posts and preserves scroll restoration
    And any valid selected new folder or confirmed open-folder title is retained
    And any deleted selection is cleared

    Examples:
      | mutation |
      | create whose name sorts before the cursor |
      | create whose name sorts after the cursor |
      | rename across the cursor boundary |
      | delete of the selected or open folder |
```

### AT-006: Open, Move, And Unsave The Exact Saved Item

Requirement IDs: BR-002, FR-006, FR-011, FR-012, FR-018, NFR-003, RULE-005
Acceptance Criteria: AC-011, AC-012, AC-018, AC-030
Priority: Must
Level: Acceptance
Automation Target: `app/test/saved_posts/pages/saved_posts_page_test.dart`, `app/test/saved_posts/pages/saved_post_folder_page_test.dart`

```gherkin
Feature: Saved item actions
  Scenario Outline: A summary opens the exact saved record
    Given a saved <postType> with savedAt metadata
    When its summary is tapped
    Then <destination> opens
    And the saved time is visible outside the PostSummary core

    Examples:
      | postType | destination |
      | top-level post | that post normally |
      | direct comment | its root thread focused on the saved URI |
      | nested reply | its root thread focused on the saved URI |

  Scenario: Move is confirmation-driven and preserves chronology
    Given a saved item is assigned to folder A
    When Move opens
    Then folder A is preselected
    When folder B is selected and confirm is pressed
    Then the dialog stays busy until the server confirms
    And failure keeps folder A's confirmed placement and the attempted selection for retry
    And success removes the item from folder A, assigns only folder B, and preserves savedAt
    And no optimistic assignment removal occurs before confirmation

  Scenario: Item unsave remains confirmation-free
    Given a visible saved row
    When Unsave is invoked
    Then the shared optimistic-unsave behavior runs without a dialog
```

### AT-007: Create, Rename, And Delete A Folder With Explicit Scope

Requirement IDs: BR-003, FR-005, FR-013, FR-014, FR-024, FR-026, RULE-003, RULE-007, RULE-011
Acceptance Criteria: AC-005, AC-013, AC-017, AC-023, AC-031
Priority: Must
Level: Acceptance
Automation Target: `app/test/saved_posts/pages/saved_posts_page_test.dart`, `app/test/saved_posts/pages/saved_post_folder_page_test.dart`

```gherkin
Feature: Folder management
  Scenario: Folder actions live in their confirmed locations
    Given the Saved overview
    When Add folder is used with a server-valid duplicate or case-variant name
    Then a distinct opaque-ID folder is created
    When that row is tapped
    Then a separate screen opens with its name as title
    And Rename and Delete are in the app-bar overflow
    And Back restores the overview and its scroll position
    And route diagnostics expose neither the folder ID nor name

  Scenario: Folder deletion always offers all safe and destructive choices
    Given any folder, including one with no visible posts
    When Delete is selected
    Then the localized dialog offers Cancel, Delete folder and keep saved posts, and Delete folder and remove saved posts
    And Cancel receives default keyboard focus
    And remove-saves receives the strongest destructive styling and semantics
    When keep-saves is chosen
    Then one folder DELETE without deleteSaves=true is issued
    When remove-saves is chosen
    Then one folder DELETE with deleteSaves=true is issued
    And Flutter never enumerates rows or sends per-post deletes
```

### AT-008: Delete A Folder Atomically Without Touching Public Posts

Requirement IDs: BR-003, BR-005, FR-015, FR-016, FR-017, NFR-001, RULE-003, RULE-006
Acceptance Criteria: AC-014, AC-015, AC-016, AC-025
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/saved_post_test.go`, `appview/internal/api/saved_post_store_test.go`, `appview/internal/api/saved_post_observability_test.go`

```gherkin
Feature: Atomic folder deletion modes
  Scenario Outline: Delete uses the requested private-data mode
    Given Alice owns a folder containing visible and policy-hidden saves
    And Bob owns unrelated saves and folders
    When Alice deletes the folder with <query>
    Then the handler returns 204
    And <ownedResult>
    And savedAt is unchanged for every preserved save
    And Bob's data and every indexed/public/PDS post remain unchanged
    And no PDS client, firehose event, notification, or public count mutation occurs

    Examples:
      | query | ownedResult |
      | absent | all folder saves become unfiled and the folder is deleted |
      | deleteSaves=false | all folder saves become unfiled and the folder is deleted |
      | deleteSaves=true | every save in the folder is deleted and the folder is deleted |

  Scenario: Failure and owner absence reveal nothing and commit nothing partial
    Given the folder is missing, Bob-owned, or the transaction fails
    When Alice requests delete-with-saves
    Then missing and Bob-owned requests are indistinguishable 204 no-ops
    And a real failure leaves the folder and all contained saves unchanged
    And private owner-target and folder values are absent from errors and diagnostics

  Scenario: Query validation follows the v1 contract
    Given an invalid deleteSaves value or an unknown query parameter
    When the endpoint parses the request
    Then it returns the standard camelCase validation error envelope
```

### AT-009: Synchronize Saved State Without Crossing Accounts

Requirement IDs: BR-005, FR-006, FR-008, FR-018, FR-019, NFR-001, RULE-009
Acceptance Criteria: AC-018, AC-024, AC-025, AC-026
Priority: Must
Level: Acceptance
Automation Target: `app/test/saved_posts/providers/saved_post_state_provider_test.dart`, `app/test/saved_posts/providers/account_saved_posts_provider_test.dart`

```gherkin
Feature: Account-scoped saved state
  Scenario: One confirmed URI state converges across loaded surfaces
    Given the same canonical URI appears in feed, profile, thread, and Saved posts for Alice
    When Alice saves, moves, unsaves, or deletes the containing folder
    Then every Alice surface reads the same confirmed saved/folder state
    And stale refetch data is reconciled without duplicating mutation rules per screen

  Scenario: A late completion cannot mutate the newly active account
    Given Alice starts each supported list, dialog, or mutation operation
    When the active account switches to Bob before completion
    Then Alice's completion changes none of Bob's bookmark, folder, cursor, list, dialog, navigation, or message state
    And no private Alice value is displayed or diagnosed under Bob
    When the app switches back and refetches
    Then Alice and Bob each show only their own server-confirmed state
```

### AT-010: Preserve Quote Preview Policy While Adopting PostSummary

Requirement IDs: BR-004, FR-002, FR-020, FR-021, FR-024, NFR-004
Acceptance Criteria: AC-020, AC-023, AC-032
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_card_test.dart`, `app/test/shared/widgets/post_summary_test.dart`

```gherkin
Feature: Compact quote summary
  Scenario Outline: Quote state and navigation survive extraction
    Given a quote preview is <state>
    When its parent PostCard renders
    Then PostSummary renders the existing bounded content or policy presentation for <state>
    And author tap, post tap, reveal action, project title, representative image, and one-level behavior remain as applicable
    And no bookmark or engagement control appears inside the summary

    Examples:
      | state |
      | visible |
      | muted and revealable |
      | blocked |
      | hidden |
      | unavailable |
```

### AT-011: Preserve Notification Context And Destination While Adopting PostSummary

Requirement IDs: BR-004, FR-020, FR-022, FR-024, NFR-004
Acceptance Criteria: AC-021, AC-023
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/notifications_page_test.dart`, `app/test/shared/widgets/post_summary_test.dart`

```gherkin
Feature: Notification subject summary
  Scenario Outline: A post-bearing notification retains its action context
    Given a <category> notification with a post subject
    When NotificationRow renders and is tapped
    Then PostSummary renders the bounded subject without engagement controls
    And actor/action title, icon/color, timestamp, follow control, filtering, and unread behavior remain unchanged
    And the existing exact post or root-plus-focus destination opens

    Examples:
      | category |
      | like |
      | repost |
      | reply |
      | mention |
      | quote |
      | post by followed account |
```

### AT-012: Keep Saved Metadata And Actions Outside PostSummary

Requirement IDs: BR-004, FR-020, FR-023, FR-024, NFR-004
Acceptance Criteria: AC-022, AC-023
Priority: Must
Level: Acceptance
Automation Target: `app/test/saved_posts/widgets/saved_post_row_test.dart`, `app/test/shared/widgets/post_summary_test.dart`

```gherkin
Feature: Saved post summary
  Scenario: Saved rows reuse compact content without coupling organization logic
    Given a saved item with author, text, project title, representative image, savedAt, and folder state
    When its row renders
    Then PostSummary owns only compact post content and its parent-provided tap behavior
    And saved time, Move, and Unsave remain parent-owned siblings
    And the summary contains no bookmark, engagement, folder, or mutation logic
```

### AT-013: Keep Errors, Accessibility, Privacy, And Quality Gates Enforceable

Requirement IDs: FR-002, FR-007, FR-014, FR-020, FR-024, FR-025, NFR-001, NFR-005, NFR-006
Acceptance Criteria: AC-019, AC-023, AC-025, AC-027
Priority: Must
Level: Acceptance
Automation Target: relevant Flutter widget/provider tests; `appview/internal/api/saved_post_observability_test.go`; repository test gates

```gherkin
Feature: Saved-post quality boundary
  Scenario: Interactive controls expose localized operable semantics
    Given bookmark, chooser, folder, summary, retry, and destructive controls
    When tested with semantics enabled, keyboard traversal, large text, and minimum tap-target assertions
    Then every control has a localized name and applicable selected, busy, or destructive state
    And destructive focus begins on Cancel
    And bounded summary content and controls remain reachable without overflow

  Scenario: Recoverable failures stay sanitized
    Given API failures containing sentinel folder names, IDs, post URIs, owner-target pairs, and raw server messages
    When Flutter and AppView handle them
    Then confirmed content remains usable where possible
    And localized retry or failure feedback contains no sentinel private value
    And logs, Sentry payloads, traces, metrics labels, route diagnostics, and analytics dimensions contain no sentinel private value

  Scenario: Required gates pass without a new unapproved dependency
    When focused Flutter and Go tests, Flutter analysis, Go formatting, and full Flutter/AppView suites run
    And Flutter and Go dependency manifests and lockfiles are reviewed in the source diff
    Then all gates pass and the diff contains no new unapproved runtime dependency
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001 | AC-001, AC-024 | Decode and copy canonical saved viewer state, including protected-placeholder defaults. | JSON with true/folder ID, false/null, omitted fields; `copyWith` updates and explicit clear. | `viewerHasSaved` and nullable folder survive mapping/copy; legitimate omission is false/null; invalid canonical types fail normally. | `app/test/feed/models/post_test.dart` |
| UT-002 | FR-007 | AC-027 | Decode typed folder, saved item, state, and page models. | CamelCase payloads, server timestamps, nullable folder IDs, post payload, opaque cursor. | Models retain exact values and generated copy can explicitly clear cursor/folder ID. | `app/test/saved_posts/models/saved_post_test.dart` |
| UT-003 | FR-004, FR-005, FR-013, FR-026, RULE-007 | AC-004, AC-005, AC-017 | Match server folder-name validation and use folder ID as equality/selection identity. | Empty/whitespace, Unicode boundaries, slash, controls, duplicate and case-variant names/IDs. | Accepted/rejected values match AppView; duplicates remain distinct; no uniqueness rule is added. | `app/test/saved_posts/models/saved_post_folder_test.dart` |
| UT-004 | FR-006, FR-018, NFR-003 | AC-006, AC-018, AC-024, AC-029 | Reduce confirmed, pending, optimistic, rollback, and server-reconciliation events for one account/URI. | Save/move success/failure, unsave start/204/failure, stale/new server snapshots, duplicate action. | Exact prior state is restorable; newer confirmed state wins; one mutation is pending; other accounts are untouched. | `app/test/saved_posts/providers/saved_post_state_provider_test.dart` |
| UT-005 | FR-010, FR-013, FR-026, NFR-002 | AC-009, AC-010, AC-017, AC-031 | Merge cursor pages, restart invalid cursors, and reconcile folder mutations without parsing cursor contents. | Initial/next pages, duplicate IDs, terminal/null cursor, load failure, `invalid_cursor`, scope/sort change, and create/rename/delete across a partial folder cursor. | Stable server order; no duplicate opaque ID; confirmed rows retained on failure; unsafe cursor restarts from page one; selected new folder/open title remains valid; deleted selection clears. | `app/test/saved_posts/providers/saved_posts_pagination_test.dart` |
| UT-006 | FR-009, RULE-008, RULE-010, RULE-011 | AC-008, AC-028, AC-031 | Project overview sections and empty states. | Folder/unfiled combinations; foldered items; newest/oldest savedAt values. | Folders first/no counts; foldered items excluded; empty Unfiled hidden; full empty only when both empty; post sort does not reorder folders. | `app/test/saved_posts/models/saved_posts_overview_test.dart` |
| UT-007 | FR-011, FR-012, FR-022 | AC-011, AC-021 | Infer exact saved/notification destination from top-level and reply references. | Top-level post, direct comment, nested reply with root/parent/exact URI. | Top-level opens normally; comments/replies open root thread focused on exact subject URI. | `app/test/saved_posts/navigation/saved_post_destination_test.dart`, `app/test/notifications/services/notification_destination_inference_test.dart` |
| UT-008 | FR-020, FR-021, FR-023, NFR-004 | AC-020, AC-021, AC-022 | Adapt `Post` and `QuotePreviewPost` to the compact representation and enforce surface-agnostic output. | Text, author, time/metadata, project title, images, visible/protected/unavailable shapes. | Bounded common fields map correctly; first image only; no engagement/bookmark/folder logic enters the core model/widget. | `app/test/shared/widgets/post_summary_test.dart` |
| UT-009 | FR-015 | AC-016 | Parse optional `deleteSaves` strictly. | Absent, `false`, `true`, empty, mixed case, repeated, invalid, and unknown query parameters. | Absent/false select unfile; true selects delete; every invalid/unknown shape returns the standard validation classification. | `appview/internal/api/saved_post_folder_request_test.go` |
| UT-010 | FR-025, NFR-001 | AC-019, AC-025 | Map failures to localized safe UI state without rendering raw server/private data. | Standard API errors containing sentinel messages/IDs/names/URIs. | Stable localized copy and retry policy; sentinels absent from user-facing state. | `app/test/saved_posts/models/saved_post_error_test.dart`, `app/test/shared/errors/sentry_redaction_test.dart` |
| UT-011 | FR-008, FR-013 | AC-007, AC-031 | Define canonical Saved and generic folder-screen route locations/names. | Route path construction and router diagnostic metadata. | Canonical overview is `/profile/settings/saved`; folder navigation is typed; generic route name contains no private ID/name. | `app/test/router/saved_posts_route_test.dart` |
| UT-012 | FR-015 | AC-015, AC-016 | Map folder-delete validation, absence, owner mismatch, and store failures. | Missing/cross-owner, invalid query, transaction error. | Missing/cross-owner are indistinguishable 204; validation uses v1 envelope; real failures use bounded error mapping. | `appview/internal/api/saved_post_error_test.go` |
| UT-013 | FR-004, FR-005, RULE-001, RULE-004 | AC-003, AC-005, AC-030 | Manage chooser selection and independent inline-create state. | Default/current assignment, duplicate IDs, create success/failure, list failure, cancel. | Exactly one selection; create persists/selects independently; no save before confirm; No folder survives list failure. | `app/test/saved_posts/providers/save_post_dialog_controller_test.dart` |
| UT-014 | FR-011, RULE-005, RULE-008 | AC-008, AC-012 | Apply server-returned move/delete results without changing chronology. | Same URI moved/unfiled, containing folder deleted with keep, savedAt and post createdAt. | Assignment changes use server state; savedAt remains unchanged and remains the only list-sort timestamp. | `app/test/saved_posts/providers/saved_post_state_provider_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-004, FR-007, FR-015, FR-017, FR-026 | AC-003, AC-004, AC-014, AC-015, AC-016, AC-017, AC-027 | Verify every Flutter API request/response contract, including no-body DELETE and `deleteSaves` serialization. | Dio + `http_mock_adapter` with typed sample payloads/errors. | List/save/move/unsave posts; list/create/rename/delete folders; list unfiled/folder scopes and both sorts. | Exact authenticated `/v1/` method/path/body/query/camelCase contract; explicit nullable `folderId`; opaque values untouched; delete-with-saves is one request, never per-post recursion. | `app/test/saved_posts/data/saved_post_api_client_test.dart` |
| IT-002 | FR-007 | AC-027 | Verify repository interface and implementation expose all typed operations without UI JSON/network knowledge. | Fake API client/repository and typed models. | Invoke every repository method through its interface. | Arguments/results/errors forward exactly; nullable folder assignment and server-returned state are preserved. | `app/test/saved_posts/data/saved_post_repository_test.dart` |
| IT-003 | FR-009, FR-010, FR-013, FR-026, NFR-002, RULE-008 | AC-008, AC-009, AC-010, AC-017, AC-028, AC-031 | Keep folder, unfiled, and per-folder pagination/sort state independent, including after folder mutations. | ProviderContainer + controllable repository pages/completers with a partially loaded alphabetical folder collection. | Initial load, parallel load-more, refresh, sort, failure, retry, invalid cursor, then confirmed create/rename/delete on either side of the cursor. | Correct cursor per resource; concurrency guarded; folders remain above unfiled; unrelated resources stay intact; the affected folder list restarts/reconciles in server order, deduplicates by opaque ID, retains valid selection/title state, and clears deleted selection. | `app/test/saved_posts/providers/saved_posts_provider_test.dart` |
| IT-004 | FR-005, FR-006, FR-011, NFR-003, RULE-001, RULE-004, RULE-005 | AC-005, AC-006, AC-012, AC-018, AC-030 | Exercise confirmation-driven save/move and independent folder creation through provider/repository boundaries. | ProviderContainer + completers + duplicate folder fixtures. | Confirm save/move, rapid repeat, failure/retry, create/cancel. | One request; busy state; silent confirmed close; inline failure retains selection; no optimistic move; created folder survives cancel; savedAt preserved. | `app/test/saved_posts/providers/saved_post_mutation_provider_test.dart` |
| IT-005 | FR-003, FR-006, FR-018, NFR-003 | AC-006, AC-018, AC-024, AC-029 | Exercise optimistic unsave and exact rollback across loaded state. | Same URI seeded into multiple provider surfaces; controlled DELETE. | Start duplicate unsaves, succeed/fail, then reconcile a canonical post. | One request; all consumers update immediately; 204 confirms; failure restores exact folder; stale server data cannot cross confirmed ordering. | `app/test/saved_posts/providers/saved_post_mutation_provider_test.dart` |
| IT-006 | BR-005, FR-018, FR-019, NFR-001, RULE-009 | AC-018, AC-024, AC-025, AC-026 | Prove account switch invalidates or ignores every late saved/folder completion. | Alice/Bob AccountKeys, provider container, delayed repository futures, recording messenger/router. | Switch during list, dialog load/create, save, move, unsave, rename, and delete. | No Alice result mutates Bob's state/message/navigation; switching back/refetching yields per-account server state. | `app/test/saved_posts/providers/account_saved_posts_provider_test.dart` |
| IT-007 | FR-008, FR-013, RULE-006 | AC-007, AC-026, AC-031 | Verify typed Settings overview/folder routes and private diagnostic naming. | Signed-in router harness and fake repository. | Tap Settings tile, open folder, pop twice, inspect route names/locations. | Correct full-screen stack and restoration; no profile Saved tab/canonical placeholder; folder diagnostic name is generic and identifier-free. | `app/test/router/saved_posts_route_test.dart` |
| IT-008 | FR-009–FR-014, FR-023, RULE-010, RULE-011 | AC-008–AC-013, AC-017, AC-019, AC-022, AC-023, AC-028, AC-030, AC-031 | Exercise overview and folder-screen rendering, pagination, mutation reconciliation, actions, errors, empty states, focus, and scroll restoration. | Widget harness + fake repository/provider overrides + semantics. | Load/refresh/page/sort/open/move/unsave/create/rename/delete/back across success and failure fixtures, including partial folder pagination. | Confirmed hierarchy, separate folder screen, parent-owned row actions, safe three-choice delete UI, usable errors, stable overview scroll, server-ordered deduplicated folder rows, confirmed renamed title, and no dangling deleted selection. | `app/test/saved_posts/pages/saved_posts_page_test.dart`, `app/test/saved_posts/pages/saved_post_folder_page_test.dart` |
| IT-009 | FR-015 | AC-014, AC-015, AC-016 | Verify handler and route contract for folder delete query modes. | `httptest`, authenticated owner context, fake recording store. | DELETE with absent/false/true/invalid/unknown query forms. | Correct store mode and 204; standard validation envelope; no body expected; route retains auth/device/write policy. | `appview/internal/api/saved_post_test.go`, `appview/internal/routes/routes_test.go` |
| IT-010 | BR-003, FR-015, RULE-003, RULE-005 | AC-013, AC-014, AC-015 | Preserve saves atomically when folder deletion omits/false `deleteSaves`. | Isolated real-Postgres schema with multiple owners, folder saves, timestamps. | Delete once/concurrently/repeatedly, including missing/cross-owner IDs. | Owned folder removed; all its saves unfiled with identical savedAt; Bob unchanged; absence remains idempotent. | `appview/internal/api/saved_post_store_test.go` |
| IT-011 | BR-003, FR-015–FR-017, RULE-003 | AC-014, AC-015, AC-025 | Remove all owned folder saves atomically, including policy-hidden rows, and roll back injected failures. | Isolated real-Postgres schema with visible/hidden saves; transaction-failure trigger or canceled context. | Delete with `deleteSaves=true`, concurrently/repeatedly, then run failure case. | One transaction deletes folder+all owned saves; no hydration/enumeration; failure leaves every row intact; other owners/public posts unchanged. | `appview/internal/api/saved_post_store_test.go` |
| IT-012 | BR-005, FR-015, FR-016, FR-025, FR-026, NFR-001 | AC-014, AC-015, AC-019, AC-025 | Verify owner privacy, non-disclosure, no public/PDS side effects, and diagnostic redaction. | Alice/Bob/store fixtures, forbidden PDS/event fakes, sentinel private values, test observer/log capture. | Exercise success, cross-owner, invalid, and failure paths for both delete modes and Flutter error mapping. | Only authenticated private rows change; author gets no signal; forbidden collaborators untouched; sentinels absent from responses/logs/traces/metrics/Sentry. | `appview/internal/api/saved_post_observability_test.go`, `appview/internal/api/saved_post_store_test.go`, Flutter redaction tests |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Full `PostCard` still renders reply/like/repost/overflow counts, colors, menus, taps, and layout after bookmark insertion. | FR-002, NFR-005 | AC-001, AC-023, AC-027, AC-032 | Extend `app/test/feed/widgets/post_card_test.dart` to assert existing actions and the bookmark-before-overflow order across post types. |
| REG-002 | Saved content is not presented as own/visitor profile content, while normal profile tabs remain. | FR-008, RULE-006 | AC-007, AC-025 | Update `app/test/profile/profile_page_test.dart` and tab tests to assert `ProfileTab.saved`/Saved labels are absent and Settings owns the entry. |
| REG-003 | Quote previews retain visible, muted/revealable, blocked, hidden, unavailable, author/post tap, image, project, and one-level behavior. | FR-021, NFR-004 | AC-020 | Keep/extend quote cases in `app/test/feed/widgets/post_card_test.dart` while asserting `PostSummary` adoption and no controls. |
| REG-004 | Every post-bearing notification keeps actor/action context, filtering, category treatment, follow control, timestamp, and exact destination/focus. | FR-012, FR-022, NFR-004 | AC-011, AC-021 | Extend `app/test/notifications/notifications_page_test.dart` and destination inference tests for each category. |
| REG-005 | Protected placeholders that omit viewer fields still decode/render safely without exposing actions or hidden content. | FR-001, FR-002 | AC-001, AC-032 | Add omitted-field model fixture and protected `PostCard` widget assertion. |
| REG-006 | Existing folder deletion without the new query remains a non-destructive atomic unfile operation. | FR-015, RULE-003 | AC-014, AC-016 | Retain and extend existing `appview/internal/api/saved_post_test.go` / `saved_post_store_test.go` absent-query cases. |
| REG-007 | Existing saved-list response, opaque pagination, policy hydration, and exact reply context are unchanged by the delete extension. | NFR-005 | AC-009, AC-010, AC-011, AC-027 | Run existing `saved_post_*_test.go` suites unchanged plus focused response/cursor checks. |
| REG-008 | Folder delete modes never change indexed post rows, engagement counts, notifications, or introduce a PDS dependency. | FR-016, NFR-001 | AC-014, AC-025 | Snapshot relevant rows/fakes before/after both modes and assert byte/value equality and zero collaborator calls. |
| REG-009 | Existing session/account boundary behavior remains account-specific during cancellation and switching. | FR-019, NFR-001, RULE-009 | AC-026 | Add saved providers to existing account-boundary invalidation tests and retain router/account-switch suites. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Canonical saved viewer states | Same post JSON with `viewerHasSaved` false/null, true/folder ID, true/null, omitted protected fields, and malformed types. | AT-001, UT-001, IT-001, REG-005 |
| TD-002 | Eligible full-post variants | Ordinary, project, quote, direct comment, and nested reply with stable canonical URIs; protected placeholder. | AT-001, AT-006, REG-001, REG-005 |
| TD-003 | Folder identity/validation set | Distinct opaque IDs with names `Ideas`, `IDEAS`, duplicate `Ideas`, whitespace, Unicode boundary, slash, and control characters across two pages. | AT-002, AT-007, UT-003, IT-001 |
| TD-004 | Independent page and folder-mutation set | Two folder pages, two unfiled pages, two pages for two folders, newest/oldest cursors, duplicate overlap, terminal/invalid cursors, plus create/rename/delete fixtures whose alphabetical positions fall before and after a partially loaded folder cursor. | AT-004, AT-005, UT-005, IT-003, IT-008 |
| TD-005 | Chronology set | Posts whose post `createdAt` order differs from server `savedAt`, including equal timestamps and a moved/unfiled save. | AT-004, AT-006, UT-006, UT-014 |
| TD-006 | Quote policy set | Visible text/image/project, muted-revealable, blocked, hidden, unavailable, and nested quote inputs. | AT-010, UT-008, REG-003 |
| TD-007 | Notification category set | Like, repost, reply, mention, quote, post-by-followed, follow-only control, unavailable/generic/unknown rows with expected destinations. | AT-011, UT-007, REG-004 |
| TD-008 | Atomic delete database set | Alice folder with visible and policy-hidden saves; Alice unfiled/other-folder saves; Bob folder/saves; indexed public posts; original timestamps. | AT-008, IT-010–IT-012, REG-006–REG-008 |
| TD-009 | Account race set | Alice and Bob AccountKeys, same canonical URI with different saved/folder state, delayed futures for each list/dialog/mutation operation. | AT-009, UT-004, IT-006, REG-009 |
| TD-010 | Private redaction sentinels | Unique folder name, folder ID, post URI, owner-target pair, raw server message, and cursor token recognizable in captured output. | AT-008, AT-013, UT-010, IT-012 |
| TD-011 | Accessibility/layout set | Localized long strings, long Unicode folder/text, text scale 2.0+, narrow viewport, keyboard traversal, semantics-enabled widgets. | AT-007, AT-010–AT-013, MAN-001, MAN-002 |
| TD-012 | Transaction failure set | Trigger/cancelable store path that fails after save-row work but before folder deletion commits. | AT-008, IT-011 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-024 | AC-023 | Real assistive-technology and keyboard focus behavior. | On one iOS/VoiceOver or Android/TalkBack target, traverse bookmark, save/move chooser, inline folder create/error, folder overflow, and delete dialog; repeat destructive dialog with hardware keyboard where available. | Announcements include localized role/name and selected/busy state; traversal is logical; Cancel receives initial destructive focus; no hidden/duplicate control is announced. |
| MAN-002 | FR-024 | AC-023 | Narrow-device and large-text visual usability. | At maximum supported text scaling on a narrow phone, inspect a long Unicode folder name and long post text in overview, folder screen, chooser, delete dialog, quote, notification, and saved row. | Text wraps/truncates intentionally, actions/tap targets remain reachable, no overflow occurs, and compact summaries remain bounded. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Flutter widget semantics do not perfectly reproduce VoiceOver/TalkBack announcement order. | FR-024 | Framework semantics assertions cover labels/state/focus nodes but not every platform announcement. | Run MAN-001 before stage completion; record any platform-specific defect as an implementation blocker. |
| GAP-002 | Real-Postgres atomicity/privacy cases depend on `TEST_DATABASE_URL`. | FR-015, FR-016, NFR-001 | The existing Go suite can skip integration cases when the database is unavailable. | Run through `just test` against compose Postgres and preserve explicit pass evidence; a skipped run does not satisfy IT-010–IT-012. |
| GAP-003 | Device-size and text-rendering behavior varies by platform/font. | FR-024 | Widget tests cover representative constraints, not every native font rasterization. | Run MAN-002 on at least one narrow device/simulator after automated layout tests pass. |

No blocking design gap is identified. These gaps supplement, rather than replace, automated coverage.

## 10. Out Of Scope

- Lexicon, PDS-record, firehose, public engagement-count, and source-post deletion tests, except negative regression assertions proving this feature does not touch them.
- Nested folders, tags, sharing, notes, pinning, manual order, folder/search counts, save search, bulk selection, drag-and-drop, offline queues, analytics events, and unrelated composer summaries.
- Full device end-to-end automation or a new integration-test dependency. Existing Flutter test primitives, Dio adapters, provider containers, router harnesses, and Go/Postgres tests are sufficient.
- Visual redesign of full `PostCard`, notifications, or quotes beyond the bookmark and common compact-summary extraction.
- Production load testing. Bounded cursor behavior, query plans, no N+1 requests, and duplicate-request guards are covered at unit/integration level.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- Review result: `03-document-review.md` is complete with `Approved with notes`; DR-001–DR-004 are reflected in the revised requirements and test design, and coding planning may proceed.
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-21-flutter-saved-posts/`
- Recommended first failing test for implementation: `UT-001` in `app/test/feed/models/post_test.dart` for decoding/copying `viewerHasSaved` and nullable `viewerSavedFolderId`, because every Flutter bookmark and synchronization path depends on trustworthy server state.
- Suggested test order for implementation: `UT-001` → `UT-002`/`IT-001`/`IT-002` typed contracts → `UT-004`/`IT-004`/`IT-005` mutation seam → `AT-001`–`AT-003` bookmark/chooser → `UT-005`/`UT-006`/`IT-003` collection and folder-mutation cursor reconciliation → `AT-004`–`AT-007` routes/pages/folders → `UT-009`/`IT-009`–`IT-012` AppView delete mode → `UT-008`/`AT-010`–`AT-012` summary extraction → `AT-009` account races → `AT-013`/regression/full gates and dependency-file diff review → manual checks.
- Commands discovered: `cd app && flutter test test/feed/models/post_test.dart`; `cd app && flutter test test/saved_posts`; `cd app && flutter test test/feed/widgets/post_card_test.dart test/notifications/notifications_page_test.dart test/router/saved_posts_route_test.dart`; `just app-analyze`; `just app-test`; `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./internal/api ./internal/routes`; `just fmt`; `just test`.
- Dependency review gate: inspect `app/pubspec.yaml`, Flutter lockfiles, `appview/go.mod`, and `appview/go.sum`; any new runtime dependency requires explicit approval rather than inference from passing tests.
- Blocking gaps: None.
- Risk level: Medium. Document review is complete; private account isolation, destructive transaction semantics, folder-mutation pagination reconciliation, and extraction across three existing surfaces remain high-cost regression areas for the coding plan to preserve.
