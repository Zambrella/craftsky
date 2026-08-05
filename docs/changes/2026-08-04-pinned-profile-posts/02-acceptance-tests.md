# Acceptance Test Specification: Pinned Profile Posts

## 1. Test Strategy

Risk level: Medium, carried forward from `01-requirements.md`.

The feature crosses private Postgres state, AppView authorization and pagination, Flutter account-scoped state, and shared card presentation. Verification therefore uses several focused layers rather than relying on a new full-system UI harness:

- Go database and store integration tests prove the two-slot invariant, atomic replacement, target-specific unpin, lifecycle cleanup, account isolation, and indexed query plans against real Postgres.
- Go handler and route tests prove the exact authenticated JSON contracts, the deliberate bodyless PUT/DELETE plus authoritative `200 OK` exception, status/error envelopes, privacy boundary, and route policies without calling a PDS.
- Go profile-list integration tests prove pin-first ordering, viewer policy, fixed page limits, full cursor traversal, and pin-bound cursor invalidation.
- Flutter model, API-client, provider, and widget tests prove exact page-level metadata omission/tolerant decoding, non-optimistic mutations, active-account-and-slot pending states, independent-slot availability, exact feedback, account fencing, menu scope, and the profile-only annotation.
- Regression tests protect chronological ordering and presentation outside profile pins.
- Manual checks are limited to physical screen-reader output and final visual parity at device text scales.

All listed automated tests are designs for implementation; no source code, test code, migration, or dependency is created or executed during this stage. There are no blocking requirements questions.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-004, AC-008 | AT-001, AT-002, AT-003, AT-005, IT-002, IT-006 | Acceptance / Integration | Yes |
| BR-002 | AC-010, AC-018 | AT-005, UT-007, MAN-001, MAN-002 | Acceptance / Unit / Manual | Partial |
| BR-003 | AC-007, AC-011, AC-015 | AT-010, AT-012, IT-012, REG-002, REG-005 | Acceptance / Integration / Regression | Yes |
| FR-001 | AC-001, AC-002, AC-005, AC-006 | AT-001, AT-002, AT-004, AT-006, UT-005 | Acceptance / Unit | Yes |
| FR-002 | AC-003, AC-004, AC-014 | AT-003, IT-002, IT-003 | Acceptance / Integration | Yes |
| FR-003 | AC-005, AC-014, AC-017 | AT-004, IT-003, IT-005 | Acceptance / Integration | Yes |
| FR-004 | AC-006, AC-016 | AT-006, IT-004, IT-011 | Acceptance / Integration | Yes |
| FR-005 | AC-007, AC-013 | AT-011, IT-001, IT-010, IT-012 | Acceptance / Integration | Yes |
| FR-006 | AC-008, AC-010, AC-012, AC-017 | AT-005, AT-007, UT-001, IT-006, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-009 | AT-007, UT-004, IT-007, IT-016 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-003, AC-005, AC-013, AC-018 | AT-003, AT-004, AT-008, AT-009, UT-003, UT-006, IT-016 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-010, AC-011, AC-018 | AT-005, AT-012, UT-007, REG-002, MAN-002 | Acceptance / Unit / Regression / Manual | Partial |
| FR-010 | AC-015, AC-016 | AT-010, IT-004, IT-011, REG-005 | Acceptance / Integration / Regression | Yes |
| FR-011 | AC-012 | AT-006, AT-011, IT-008, REG-003 | Acceptance / Integration / Regression | Yes |
| FR-012 | AC-013, AC-016 | AT-009, AT-011, IT-010, IT-011 | Acceptance / Integration | Yes |
| FR-013 | AC-017 | UT-001, UT-002, UT-009, IT-005, IT-009, IT-015 | Unit / Integration | Yes |
| NFR-001 | AC-004, AC-014, AC-016 | UT-006, IT-002, IT-003, IT-011 | Unit / Integration | Yes |
| NFR-002 | AC-009, AC-019 | IT-006, IT-007, IT-013 | Integration | Yes |
| NFR-003 | AC-007, AC-016, AC-019 | IT-011, IT-012, IT-014, REG-004 | Integration / Regression | Yes |
| NFR-004 | AC-018 | UT-005, UT-007, MAN-001, MAN-002 | Unit / Manual | Partial |
| NFR-005 | AC-019 | IT-014 | Integration | Yes |
| NFR-006 | AC-020 | IT-001–IT-016, UT-001–UT-009, REG-001–REG-008, MAN-001–MAN-002 | All applicable levels | Partial until manual checks run |
| RULE-001 | AC-003, AC-004, AC-005 | AT-003, AT-004, UT-008, IT-002, IT-003 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-001, AC-002, AC-006 | AT-001, AT-002, AT-006, UT-008, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-007, AC-011 | AT-012, IT-012, REG-002, REG-006 | Acceptance / Integration / Regression | Yes |
| RULE-004 | AC-010, AC-015 | AT-005, AT-010, UT-007 | Acceptance / Unit | Yes |

## 3. Acceptance Scenarios

### AT-001: Pin A Standard Post From The Owner's Profile

Requirement IDs: BR-001, FR-001, RULE-002
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/widgets/profile_posts_tab_test.dart`

```gherkin
Feature: Pinning a standard profile post
  Scenario: The owner pins an unpinned top-level standard post
    Given an authenticated current member is viewing their profile Posts tab
    And the post is their visible top-level standard post
    And their standard slot is empty
    When they open the post context menu and choose "Pin post"
    Then the AppView standard slot points to that post URI
    And the project slot is unchanged
    And the refreshed profile list shows that post first
```

### AT-002: Pin Qualifying Posts From Existing Owner-Action Surfaces

Requirement IDs: BR-001, FR-001, RULE-002
Acceptance Criteria: AC-001, AC-002
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/feed_page_test.dart`, `app/test/feed/pages/post_thread_page_test.dart`, `app/test/profile/widgets/profile_projects_tab_test.dart`

```gherkin
Feature: Pin actions on existing owner-action surfaces
  Scenario Outline: The owner pins a qualifying post
    Given an authenticated current member sees their <post kind> on <surface>
    And that card already supports owner actions
    When they choose "Pin post"
    Then the <slot> slot points to that post URI
    And the other slot is unchanged

    Examples:
      | post kind              | surface               | slot     |
      | top-level quote post   | timeline              | standard |
      | top-level project post | top-level thread card | project  |
      | top-level project post | profile Projects tab  | project  |
```

### AT-003: Replace An Occupied Pin After Server Confirmation

Requirement IDs: BR-001, FR-002, FR-008, RULE-001
Acceptance Criteria: AC-003, AC-004
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/providers/profile_pins_provider_test.dart`, `app/test/profile/widgets/profile_posts_tab_test.dart`

```gherkin
Feature: Replacing a profile pin
  Scenario: Pinning B replaces pinned standard post A
    Given standard post A is pinned
    And standard post B is visible in the owner's context menu
    When the owner chooses "Pin post" for B
    Then no confirmation dialog is shown
    And every standard-slot Pin/Unpin action for the active account is disabled while the request is pending
    And project-slot Pin/Unpin actions remain usable
    And the confirmed card order and menu state do not change while pending
    When the AppView confirms that B is the standard pin
    Then B appears first on the refreshed profile list
    And A offers "Pin post"
    And B offers "Unpin post"
    And the app shows "Post pinned"
    And the project slot is unchanged
```

### AT-004: Unpin The Current Target Without Clearing A Newer Replacement

Requirement IDs: FR-001, FR-003, FR-008, RULE-001
Acceptance Criteria: AC-005, AC-014
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/providers/profile_pins_provider_test.dart`, `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Removing a profile pin
  Scenario: The owner unpins the current target
    Given a post occupies the standard slot
    When the owner chooses "Unpin post"
    Then every standard-slot action is disabled while the request is pending
    And project-slot actions remain usable
    And the standard slot becomes empty after server confirmation
    And no replacement is required
    And the app shows "Post unpinned" after server confirmation

  Scenario: A stale device unpins a replaced target
    Given device one replaced standard post A with B
    And device two still shows A as pinned
    When device two sends target-specific unpin for A
    Then the request succeeds as an idempotent no-op
    And B remains pinned
    And the response returns B in the authoritative two-slot state
```

### AT-005: Show A Visible Pin First With The Profile-Only Attribution

Requirement IDs: BR-001, BR-002, FR-006, FR-009, RULE-004
Acceptance Criteria: AC-008, AC-010, AC-018
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/widgets/profile_posts_tab_test.dart`, `app/test/profile/widgets/profile_projects_tab_test.dart`, `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Pinned profile presentation
  Scenario: An allowed visitor opens a profile with an older pin
    Given the profile has a visible pinned post older than its newest posts
    And the profile response identifies it with page-level "pinnedPostUri"
    When an allowed visitor opens the corresponding profile tab with limit 10
    Then the pinned card is item one
    And at most nine chronological cards follow it in newest-first order
    And only that card shows a pin icon and exact text "Pinned post"
    And the row uses the repost attribution position and semantics
    And the row has no independent tap action
```

### AT-006: Reject Invalid New Pins And Respect Viewer Policy

Requirement IDs: FR-001, FR-004, FR-011, RULE-002
Acceptance Criteria: AC-006, AC-012
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/profile_pin_test.go`, `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Pin authorization and visibility
  Scenario Outline: An invalid target cannot be newly pinned
    Given an authenticated current member attempts to pin <target>
    When the AppView validates the target
    Then it returns <status> with error code <error>
    And neither pin slot changes

    Examples:
      | target                                  | status | error          |
      | another author's post                   | 403    | forbidden      |
      | a missing post                          | 404    | post_not_found |
      | a deleted or currently hidden post      | 404    | post_not_found |
      | a comment or nested reply                | 422    | pin_not_allowed |
      | a post structurally mismatched to a slot | 422    | pin_not_allowed |

  Scenario: A stored pin is hidden from one viewer
    Given a stored pin is excluded for a viewer by block, moderation, membership, or language policy
    When that viewer opens the profile list
    Then the post is not promoted or labelled
    And page one omits "pinnedPostUri"
    And the page fills from allowed chronological posts
    And the private pin remains stored
```

### AT-007: Traverse Pin-Aware Pages And Restart After A Pin Change

Requirement IDs: FR-006, FR-007, NFR-002
Acceptance Criteria: AC-008, AC-009
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/profile_pin_pagination_test.go`, `app/test/feed/providers/user_posts_provider_test.dart`, `app/test/projects/providers/user_projects_provider_test.dart`

```gherkin
Feature: Pin-aware profile pagination
  Scenario: A fixed pin remains unique across a complete traversal
    Given an older visible pin and more posts than one requested page
    When every cursor page is requested with the same limit
    Then the pin appears only as item one on page one
    And every later page omits "pinnedPostUri"
    And every visible chronological post appears exactly once
    And no response exceeds the requested limit

  Scenario: The pin changes during traversal
    Given page one was loaded with pin A and returned a continuation cursor
    When the slot changes to pin B before the cursor is used
    Then the AppView returns 400 with error "invalid_cursor"
    And Flutter discards the traversal and restarts from page one
    And the restarted list is based on pin B
```

### AT-008: Preserve Confirmed UI State When A Mutation Fails

Requirement IDs: FR-008, NFR-004
Acceptance Criteria: AC-018
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/providers/profile_pins_provider_test.dart`, `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Pin mutation failure recovery
  Scenario Outline: A failed action restores interaction without changing confirmed state
    Given the owner starts <action> for a qualifying post
    And every action for that active-account slot is disabled while pending
    And the independent slot remains usable
    And the confirmed profile order and menu state are visible
    When the AppView request fails
    Then the confirmed order and menu state remain unchanged
    And every action for the affected slot becomes enabled again
    And the app announces <message>

    Examples:
      | action | message                              |
      | pin    | Couldn’t pin post. Try again.        |
      | unpin  | Couldn’t unpin post. Try again.      |
```

### AT-009: Fence In-Flight Results To The Initiating Account

Requirement IDs: FR-008, FR-012, NFR-001
Acceptance Criteria: AC-013, AC-016, AC-018
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/providers/profile_pins_provider_test.dart`, `app/test/auth/providers/account_boundary_provider_test.dart`

```gherkin
Feature: Account-scoped profile pin state
  Scenario: The active account changes while a pin request is pending
    Given account A starts a pin request
    When the app switches to account B before the response arrives
    And account A's response later completes
    Then account B's pin state, profile cache, and feedback are unchanged
    And account A's persisted pin remains available when A becomes active again
```

### AT-010: Allow Every Authenticated Current Member To Pin

Requirement IDs: BR-003, FR-010, RULE-004
Acceptance Criteria: AC-015, AC-016
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/profile_pin_test.go`, `app/test/feed/providers/profile_pins_provider_test.dart`

```gherkin
Feature: Universal member pinning
  Scenario: An ordinary current member uses both pin slots
    Given an authenticated current member owns one valid standard post and one valid project post
    When they pin, replace, and unpin valid targets
    Then each operation is authorized from membership, ownership, target validity, and policy only
    And both slots behave identically for every current member
    And allowed profile visitors see the resulting visible pin
```

### AT-011: Apply Permanent And Temporary Pin Lifecycle Rules

Requirement IDs: FR-005, FR-011, FR-012
Acceptance Criteria: AC-012, AC-013, AC-016
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/profile_pin_lifecycle_test.go`, `appview/internal/index/craftsky_post_test.go`

```gherkin
Feature: Profile pin lifecycle
  Scenario Outline: Permanent invalidity clears only the affected pin
    Given both pin slots are occupied
    When <change> permanently invalidates one pinned target
    Then that pin is deleted
    And no placeholder is shown
    And the pin is not moved to another slot
    And the other slot is unchanged

    Examples:
      | change                                      |
      | the indexed target is deleted               |
      | the owner's Craftsky membership is removed  |
      | the target becomes a reply                   |
      | the target changes between standard/project |

  Scenario: Temporary policy hiding retains a pin
    Given a stored pin becomes temporarily hidden or language-filtered
    When it later becomes returnable again
    Then the same retained pin reappears first
```

### AT-012: Keep Pin Prominence Out Of Other Surfaces

Requirement IDs: BR-003, FR-009, NFR-003, RULE-003
Acceptance Criteria: AC-007, AC-011
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_card_test.dart`, `app/test/feed/feed_page_test.dart`, `app/test/feed/pages/post_thread_page_test.dart`, `app/test/search/search_page_test.dart`

```gherkin
Feature: Profile-only pin presentation
  Scenario Outline: A pinned post appears normally outside its profile list
    Given a post is pinned on its author's profile
    When the same post appears on <surface>
    Then it has the surface's normal order
    And it does not show "Pinned post"

    Examples:
      | surface                   |
      | timeline                  |
      | post thread               |
      | search results            |
      | notifications             |
      | saved posts               |
      | general project discovery |
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-006, FR-013 | AC-008, AC-010, AC-017 | Decode and encode profile pages with page-level pin metadata. | Page one with a valid `pinnedPostUri`; pages with the key absent; compatibility input with explicit null; cursor and ordinary post items. | Visible `pinnedPostUri` round-trips at page level; absent and explicit-null input both decode as no pin; encoding no pin omits the key; item `Post` JSON has no `isPinned`. | `app/test/feed/models/post_page_test.dart` |
| UT-002 | FR-013 | AC-017 | Decode authoritative two-slot pin state. | Both URIs present, one null, and both null. | `standardPostUri` and `projectPostUri` map exactly; timestamps or unknown commercial/access fields are not part of the model. | `app/test/feed/models/profile_pin_state_test.dart` |
| UT-003 | FR-008 | AC-003, AC-005, AC-018 | Keep mutations server-confirmed with account-and-slot pending state and exact feedback. | Deferred standard/project pin and unpin success/failure futures under one or more active accounts. | No optimistic order/menu mutation; every action for the affected active-account slot is disabled while pending; the independent slot remains usable; success/error copy is exact; confirmed state survives failure. | `app/test/feed/providers/profile_pins_provider_test.dart` |
| UT-004 | FR-007 | AC-009 | Restart profile pagination after `invalid_cursor`. | Existing items/cursor followed by mapped `400 invalid_cursor`. | Provider discards the stale traversal, requests page one once, and publishes only the restarted page without duplicates. | `app/test/feed/providers/user_posts_provider_test.dart`, `app/test/projects/providers/user_projects_provider_test.dart` |
| UT-005 | FR-001, NFR-004 | AC-001, AC-002, AC-005, AC-006, AC-018 | Derive context-menu visibility, account-and-slot pending state, and accessible labels. | Owner/non-owner; standard/project/quote/reply; current/not-current pin; eligible/ineligible surface; standard/project pending/idle for the active account. | Only qualifying owner cards on profile, timeline, and top-level thread expose the action; current pin says `Unpin post`; every action in the affected slot is disabled and labelled while the independent slot remains enabled; search/discovery do not gain it. | `app/test/feed/widgets/post_card_test.dart` |
| UT-006 | FR-008, NFR-001 | AC-013, AC-014, AC-016 | Fence mutation completion by active-account lease. | Request initiated under account A, then switch to B before completion. | Completion cannot mutate B's pin state, list cache, or messenger; A can reload its authoritative state later. | `app/test/feed/providers/profile_pins_provider_test.dart` |
| UT-007 | BR-002, FR-009, NFR-004, RULE-004 | AC-010, AC-018 | Render accessible, non-interactive pinned attribution. | Pinned/non-pinned presentation, repost attribution baseline, narrow width, supported large text scale. | Pin icon plus exact `Pinned post` uses the attribution slot/style, exposes text semantics, has no gesture, and does not clip. | `app/test/feed/widgets/post_card_test.dart` |
| UT-008 | RULE-001, RULE-002 | AC-001, AC-002, AC-003, AC-004, AC-005, AC-006 | Classify target slot and capacity rules. | Top-level standard, top-level quote, project, comment, nested reply, type change, and two independent slots. | Standard/quote infer `standard`; project infers `project`; replies are rejected; one slot operation never changes the other. | `appview/internal/api/profile_pin_policy_test.go` |
| UT-009 | FR-013 | AC-006, AC-017 | Map AppView pin errors through the Flutter API layer. | Error envelopes for `forbidden`, `post_not_found`, `pin_not_allowed`, `invalid_cursor`, and internal failure. | Status/code/message are preserved through the established `ApiException` mapping for provider handling. | `app/test/feed/data/post_api_client_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-005, NFR-006 | AC-007, AC-013, AC-020 | Apply the reversible profile-pin migration. | Pre-migration schema with memberships, posts, and unrelated rows. | Run up, down, then up. | Private pin table, slot constraint, uniqueness, foreign keys, timestamps, and indexes are correct; unrelated rows survive; final schema is usable. | `appview/internal/db/profile_pins_migration_test.go` |
| IT-002 | BR-001, FR-002, NFR-001, RULE-001 | AC-003, AC-004, AC-014 | Persist empty, independent, idempotent, and replacement states. | Current owner with two standard and two project targets. | Pin same target repeatedly; occupy both slots; replace one slot. | At most one row per owner/slot; same-target calls succeed; replacement is atomic; other slot is unchanged; returned state is authoritative. | `appview/internal/api/profile_pin_store_test.go` |
| IT-003 | FR-002, FR-003, NFR-001, RULE-001 | AC-004, AC-005, AC-014 | Serialize concurrent replacement and target-specific unpin. | Occupied slot plus controlled concurrent transactions for targets A and B. | Commit replacements in a known order and race a stale delete for A. | Last committed replacement wins; stale unpin cannot clear B; no intermediate/two-row state is committed; responses report commit-time authoritative state. | `appview/internal/api/profile_pin_store_test.go` |
| IT-004 | FR-004, FR-010, RULE-002 | AC-006, AC-015, AC-016 | Enforce membership, ownership, returnability, and structure for new pins without blocking explicit removal. | Ordinary member, non-member, other author, missing/deleted/hidden target, standard, quote, project, comment, nested reply, and a retained pin that later becomes hidden. | Call pin for every target class, then target-specific unpin for the retained hidden target. | Valid owned targets infer the correct slot; all ordinary current members succeed; invalid new pins return the exact agreed status/code without mutation; retained hidden pin can still be explicitly removed. | `appview/internal/api/profile_pin_test.go` |
| IT-005 | FR-003, FR-013 | AC-005, AC-006, AC-017 | Verify the deliberate bodyless mutation and authoritative-response wire contracts. | Fake/store-backed handlers with occupied, empty, same-target, stale-target, invalid, and unexpected-body cases. | Call GET, bodyless PUT, and bodyless DELETE routes; repeat PUT/DELETE with a body. | Success is `200 OK` with nullable camelCase `standardPostUri`/`projectPostUri`, never `204`, and has no timestamps; mutation bodies are rejected by route policy; errors use `{error, message, requestId}`; stale delete returns newer state. | `appview/internal/api/profile_pin_test.go` |
| IT-006 | BR-001, FR-006, NFR-002 | AC-008, AC-009, AC-010, AC-012 | Return a visible older pin first within the page limit and enforce exact page metadata omission. | Visible pin outside the original first chronological page plus at least 12 newer posts; no-pin and viewer-hidden-pin cases; limits 1, 2, and 10. | List profile standard and project page one without a cursor. | A visible pin is item one and matches present `pinnedPostUri`; total items never exceed limit; remainder is newest-first; pinned row is excluded from chronological selection; no-pin and hidden-pin responses fill chronologically and omit `pinnedPostUri` rather than returning null. | `appview/internal/api/profile_pin_pagination_test.go` |
| IT-007 | FR-007, NFR-002 | AC-009 | Traverse every page, omit pin metadata after page one, and invalidate cursors after pin changes. | Dataset with tied sort timestamps, pin older than page one, and enough rows for several pages. | Traverse at fixed limits; then change/clear the pin before reusing a prior cursor. | Fixed-state traversal returns every allowed post once; every later page omits `pinnedPostUri`; cursor is deterministic; changed pin state returns `400 invalid_cursor`; no duplicate, omission, or oversized page occurs. | `appview/internal/api/profile_pin_pagination_test.go` |
| IT-008 | FR-006, FR-011 | AC-012 | Apply viewer-specific block, moderation, membership, and language policy before promotion/limit. | One stored pin and viewers for allowed, blocked, moderated, non-member-policy, and language-filtered cases. | List the same profile for each viewer; reverse temporary exclusion. | Only allowed viewers receive promotion/metadata; excluded viewers get a filled chronological page; stored pin is retained and reappears after reversal. | `appview/internal/api/profile_pin_policy_test.go` |
| IT-009 | FR-013 | AC-017 | Register the pin routes with explicit no-body policies and existing middleware. | App mux with test dependencies and route-policy registry. | Exercise GET/PUT/DELETE without auth, without device ID, with any request body, and through rate classification. | Routes use `/v1/`, require authenticated current-member/device context as specified, classify all three routes as no-body, reject mutation bodies before handler work, apply the intended rate class, and return standard envelopes. | `appview/internal/routes/routes_test.go` |
| IT-010 | FR-005, FR-012 | AC-013, AC-016 | Clear pins on permanent deletion and structural invalidation. | Both slots occupied; indexed target and membership lifecycle fixtures. | Delete target, remove membership, or update target into a reply/mismatched type. | Only affected pin is deleted; no move/placeholder occurs; the other slot remains; read-time guard never surfaces stale incompatible state. | `appview/internal/api/profile_pin_lifecycle_test.go`, `appview/internal/index/craftsky_post_test.go` |
| IT-011 | FR-004, FR-010, FR-012, NFR-001, NFR-003 | AC-015, AC-016 | Keep private state isolated across owners and account lifecycle. | Two current members on one device with distinct pins plus session/sign-out/device/reinstall-equivalent reloads. | Read/mutate under each authenticated DID and reload state after session changes. | Each owner sees/mutates only their slots; pins survive client/session changes; membership removal clears only that owner; no additional account classification is required. | `appview/internal/api/profile_pin_store_test.go`, `app/test/feed/providers/profile_pins_provider_test.dart` |
| IT-012 | BR-003, FR-005, NFR-003, RULE-003 | AC-007, AC-011, AC-016, AC-019 | Enforce the AppView-only side-effect boundary. | PDS client/Tap/notification/ranking spies or privacy sentinels around successful mutations. | Pin, replace, and unpin. | Only AppView Postgres state changes; no PDS call, lexicon/Tap event, notification, interaction count, or timeline/search/discovery signal is produced; another owner's response never exposes the selection. | `appview/internal/api/profile_pin_privacy_test.go` |
| IT-013 | NFR-002 | AC-009, AC-019 | Prove bounded indexed profile-pin queries. | Real Postgres with representative profile size and installed pin indexes. | Inspect `EXPLAIN` plans and handler/store call counts for first and later pages. | Owner/slot and post lookup use intended indexes; list hydration is set-based; query count does not grow per returned card. | `appview/internal/api/profile_pin_query_plan_test.go` |
| IT-014 | NFR-003, NFR-005 | AC-019 | Emit bounded, redacted pin observability. | Recording logger/metrics sink and success/rejection/internal-error cases. | Execute pin, replace, and unpin. | Only bounded operation/slot/result/error-class dimensions are emitted; no DID, post URI, text, timestamp, or private two-slot value appears; no feature-specific alert is required. | `appview/internal/api/profile_pin_observability_test.go` |
| IT-015 | FR-013 | AC-006, AC-017 | Call and decode the exact feature-specific Flutter pin API contracts. | Mock HTTP adapter with empty, occupied, same-target, stale-delete, and error responses. | Invoke private-state GET and target-specific PUT/DELETE. | Exact encoded paths/methods are used; PUT/DELETE send no body; `200` authoritative bodies decode; `204` is not expected; standard errors map through the shared API path. | `app/test/feed/data/post_api_client_test.dart` |
| IT-016 | FR-007, FR-008 | AC-003, AC-005, AC-009, AC-013, AC-018 | Coordinate pin state and active-account-and-slot pending ownership with standard/project profile providers. | Fake repository with deferred standard/project mutations under multiple accounts, authoritative slot responses, profile pages, and `invalid_cursor`. | Pin/replace/unpin, attempt competing same-slot and independent-slot actions, refresh the affected list, and load more across a pin change. | No optimistic reorder; competing same-account same-slot requests are suppressed; the independent slot remains usable; another account is isolated; both menu states reconcile; only the affected list refreshes; invalid cursor restarts page one; failures preserve confirmed data. | `app/test/feed/providers/profile_pins_provider_test.dart`, `app/test/feed/providers/user_posts_provider_test.dart`, `app/test/projects/providers/user_projects_provider_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Profiles with no visible pin remain chronological and retain existing cursor behavior. | FR-006, FR-007, RULE-003 | AC-008, AC-009 | Run current standard/project list ordering and tied-timestamp cursor tests with both slots empty; page one and every later page omit `pinnedPostUri` and retain unchanged item order. |
| REG-002 | Shared `PostCard` does not leak profile attribution or ranking. | BR-003, FR-009, RULE-003 | AC-011 | Render the same pinned post in timeline, thread, search, notification, saved-post, and general-project contexts; assert no `Pinned post` text/icon and normal surface order. |
| REG-003 | Existing moderation, block, membership, and language filtering still happens before pagination. | FR-011 | AC-012 | Re-run current profile-list visibility tests with a stored pin and assert excluded rows do not consume page capacity or bypass policy. |
| REG-004 | Canonical post JSON remains shared and free of profile pin state. | FR-006, NFR-003 | AC-007, AC-017 | Assert `PostResponse` and non-profile JSON contain no `isPinned`, pin timestamps, or two-slot state; only profile page metadata identifies promotion. |
| REG-005 | All ordinary authenticated current members follow the same pin authorization path. | BR-003, FR-010 | AC-015, AC-016 | Exercise valid pin operations for multiple ordinary member fixtures without any additional user-category fixture or dependency. |
| REG-006 | Pinning does not change notifications, engagement counts, or feed/search/project ranking. | RULE-003 | AC-007, AC-011 | Compare these responses before and after a pin mutation; only the relevant profile list order/metadata changes. |
| REG-007 | Standard/project profile membership remains unchanged. | RULE-002 | AC-001, AC-002, AC-006 | Re-run existing tests proving standard lists include top-level quotes and exclude projects/replies while project lists include only top-level projects. |
| REG-008 | Migration reversal does not damage public post chronology or unrelated private state. | FR-005, NFR-006 | AC-007, AC-020 | Apply up/down around seeded posts and saved-post data; verify `craftsky_posts.profile_sort_at` and saved-post rows are unchanged. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Owner and account isolation | Current members Alice and Bob, allowed visitor Carol, non-member Dave; distinct authenticated DID/device contexts. | AT-001–AT-012, IT-002–IT-012, REG-005 |
| TD-002 | Standard-slot qualification | Alice top-level standard A/B, top-level quote Q, comment C, nested reply R, plus Bob's standard X. | AT-001–AT-006, UT-005, UT-008, IT-002–IT-004, REG-007 |
| TD-003 | Project-slot qualification | Alice top-level projects P1/P2 and a structurally mismatched/updated project target. | AT-002, AT-003, AT-011, UT-008, IT-002, IT-004, IT-010, REG-007 |
| TD-004 | Pagination ordering | At least 23 visible posts with deterministic `profile_sort_at`, two tied timestamps with URI tie-breakers, and an older pin originally outside page one. | AT-005, AT-007, IT-006, IT-007, REG-001 |
| TD-005 | Viewer-policy matrix | Allowed, blocked, moderation-hidden, membership-excluded, language-filtered, and later-restored views of the same stored pin. | AT-006, AT-011, IT-008, REG-003 |
| TD-006 | Authoritative wire states | `{standardPostUri, projectPostUri}` with both set, either set, and both null; profile page one with present visible `pinnedPostUri`; no-pin, hidden-pin, and all later pages with the key omitted; explicit-null compatibility input for Flutter only; no timestamps. | UT-001, UT-002, UT-009, IT-005–IT-007, IT-015, IT-016, REG-001 |
| TD-007 | Concurrency control | Targets A/B, transaction barriers, deterministic commit ordering, same-target retries, and stale unpin for A after B wins. | AT-003, AT-004, IT-002, IT-003 |
| TD-008 | Lifecycle matrix | Permanent target deletion, membership removal, standard-to-project/project-to-reply structural updates, temporary moderation/language hiding, sign-out, device replacement, reinstall-equivalent reload, and account switching. | AT-009, AT-011, IT-008, IT-010, IT-011 |
| TD-009 | Error envelopes | `403 forbidden`, `404 post_not_found`, `422 pin_not_allowed`, `400 invalid_cursor`, and internal failure, each with `message` and `requestId`. | AT-006, AT-007, AT-008, UT-009, IT-004, IT-005, IT-009, IT-015 |
| TD-010 | Presentation constraints | Repost attribution baseline, pin icon/text, narrow phone width, supported maximum text scale, light/dark themes, touch/keyboard/semantics access. | AT-005, AT-008, UT-005, UT-007, MAN-001, MAN-002 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-002, NFR-004 | AC-018 | Physical screen-reader and focus behavior. | On supported iOS and Android devices, enable VoiceOver/TalkBack; navigate eligible standard/project owner menus in idle and one-slot-pending states plus a pinned profile card; repeat with keyboard/focus navigation where supported. | Pin/Unpin actions, affected-slot disabled state, independent-slot availability, feedback, and `Pinned post` are announced clearly; icon is not the sole signal; the attribution is not exposed as a separate action. |
| MAN-002 | BR-002, FR-009, NFR-004 | AC-010, AC-018 | Visual parity with repost attribution and large-text layout. | Compare pinned and reposted cards in light/dark themes, narrow width, and supported maximum text scale. | Pin row occupies the same slot with matching typography, spacing, and subdued colour; content remains readable without clipping or obscuring the menu. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | There is no existing full-stack Flutter device E2E suite that drives a real AppView/Postgres stack. | BR-001, FR-001, FR-008 | Cross-layer behavior is instead covered by real-Postgres Go integration tests and Flutter API/provider/widget tests. | Keep the split automation for this slice; add a system E2E only if the repository adopts that harness independently. Non-blocking. |
| GAP-002 | Physical assistive-technology announcements and exact visual parity cannot be completely proven by widget semantics assertions. | BR-002, FR-009, NFR-004 | Platform screen readers, font rendering, and device layout differ from the Flutter test environment. | Run MAN-001 and MAN-002 before considering UI verification complete. Non-blocking for document review. |
| GAP-003 | Last-committed-wins can be flaky if concurrency tests rely on goroutine scheduling. | FR-002, FR-003, NFR-001 | Wall-clock timing does not guarantee transaction commit order. | Use explicit transaction barriers/locks and assert the controlled commit order; do not use sleeps. Non-blocking. |
| GAP-004 | Query-count and privacy regressions can hide behind otherwise correct responses. | FR-005, NFR-002, NFR-003 | Output-only tests cannot prove bounded access or absence of external side effects. | Add query-plan/call-count tests and PDS/Tap/notification/ranking sentinels in IT-012–IT-014. Non-blocking. |

## 10. Out Of Scope

- Payment, free/paid-user, account-plan, tier, entitlement, and feature-access behavior or fixtures. Pinning is tested for every authenticated current member with no such integration.
- PDS pin records, lexicon changes, Tap/firehose propagation, or portability across AppViews.
- Timeline, search, discovery, or notification promotion; tests only protect their unchanged behavior.
- More than one pin per slot, custom cross-slot ordering, folders, collections, scheduling, or a separate pin-management page.
- Pin analytics beyond the bounded operational telemetry in NFR-005.
- Browser-driven or production-environment tests; the app and AppView are pre-production and the repository has no applicable full-system harness.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-08-04-pinned-profile-posts/`
- Risk level: Medium; document review is complete with an `Approved` verdict.
- Resolved document-review contracts: DR-001 fixes PUT/DELETE as path-complete no-body mutations returning authoritative `200 OK`; DR-002 fixes absent `pinnedPostUri` as omitted on no-pin/hidden-pin/later pages with tolerant Flutter decoding; DR-003 fixes pending ownership per active account and slot while preserving independent-slot actions.
- Recommended first failing test for implementation: `TestProfilePinsMigration` in proposed `appview/internal/db/profile_pins_migration_test.go`, proving the owner/slot uniqueness, slot constraint, foreign-key cleanup, indexes, and reversible up/down/up lifecycle before store behavior is written.
- Suggested test order for implementation:
  1. IT-001 migration and schema invariants.
  2. UT-008 plus IT-002–IT-004 store policy, atomic replacement, idempotency, and concurrency.
  3. IT-005, IT-009, and IT-015 exact bodyless PUT/DELETE, authoritative `200`, AppView/Flutter API contracts, and no-body route policies.
  4. IT-006–IT-008 and IT-013 profile ordering, exact `pinnedPostUri` omission, visibility, cursor traversal, and query plans.
  5. IT-010–IT-012 and IT-014 lifecycle, account isolation, privacy, and observability.
  6. UT-001–UT-006, UT-009, and IT-016 Flutter models, omission/tolerant decoding, API mapping, per-account-and-slot pending providers, independent-slot behavior, feedback, restart, and account fencing.
  7. AT-001–AT-012, UT-007, and REG-001–REG-008 widget/workflow/accessibility/regression coverage.
  8. MAN-001–MAN-002 release-oriented manual checks.
- Commands discovered:
  - Full AppView gate with compose dependencies running: `just test`
  - Focused AppView packages from `appview/` with the compose `TEST_DATABASE_URL` configured: `go test ./internal/db ./internal/api ./internal/index ./internal/routes`
  - Focused future pin tests from `appview/`: `go test ./internal/db ./internal/api ./internal/index ./internal/routes -run 'ProfilePin|ProfilePins|Pinned'`
  - Full Flutter tests: `just app-test`
  - Focused Flutter tests from the repository root: `just app-test test/feed/models/post_page_test.dart test/feed/data/post_api_client_test.dart test/feed/providers/profile_pins_provider_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_posts_tab_test.dart test/profile/widgets/profile_projects_tab_test.dart`
  - Flutter analysis: `just app-analyze`
- Blocking gaps: None.
