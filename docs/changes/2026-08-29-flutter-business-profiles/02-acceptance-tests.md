# Acceptance Test Specification: Flutter Business Profiles

## 1. Test Strategy

This is a medium-risk Flutter and AppView API change spanning profile presentation, independently versioned PDS writes, product/event forms, dynamic tabs, filtered pagination, external actions, and multi-account state. The test design uses the repository's existing Flutter widget, Riverpod provider, Dio adapter, typed-router, fake-repository, semantics, responsive-layout, and Go real-Postgres handler/store patterns.

- Unit tests cover wire decoding, tab composition, catalog fallback labels, declaration replacement payloads, combined-save reconciliation, product validation/order, event time conversion, event request serialization, filtered pagination state, diagnostics labels, locale formatting, image round-trips, stale-response protection, and outbound-action policy.
- Integration tests cover all consumed HTTP operations, AppView declaration CID/exact image projections, independently cut off Upcoming/History traversals, complete filter/cursor admission, account-type reconciliation, declaration conflicts and unknown-extension preservation, image upload, event CRUD/reporting, CID-identity cache convergence, routing guards, block/moderation boundaries, and account switching.
- Acceptance widget/handler scenarios cover account-type selection, business presentation, stable business Products and Upcoming Events tabs, profile editing, product management, two-view event management, event authoring/detail/reporting, destructive/conflict behavior, accessibility, and policy neutrality.
- Regression tests protect ordinary five-tab profiles, existing edit-profile behavior, settings/navigation shells, profile customization/scroll retention, blocked shells, unfiltered owner events, public upcoming behavior, existing business validation, and account boundaries.
- Manual checks are limited to real VoiceOver/TalkBack/keyboard usability and operating-system external-app handoff. Automated semantics, focus, layout, and launcher-adapter tests remain required.

All 41 acceptance criteria and every Must requirement have an automated verification path. Real-Postgres AppView cases require `TEST_DATABASE_URL`; skipped database tests are not sufficient release evidence. No blocking test gap was found. Risk remains **Medium**; the highest-risk paths are partial combined saves, declaration conflicts, time-zone conversion, independent filtered cursor invariants, projection lag, and late completions after account switching.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001 | AT-001, IT-004 | Acceptance / Integration | Yes |
| BR-002 | AC-002, AC-003 | AT-002, UT-003 | Acceptance / Unit | Yes |
| BR-003 | AC-004, AC-005 | AT-005, AT-006, IT-003, IT-007 | Acceptance / Integration | Yes |
| BR-004 | AC-006, AC-007 | AT-003, AT-009, IT-009 | Acceptance / Integration | Yes |
| FR-001 | AC-008 | AT-002, UT-001, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-001, AC-009 | AT-001, UT-011, IT-001, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-010 | AT-001, IT-004, REG-008 | Acceptance / Integration / Regression | Yes |
| FR-004 | AC-011 | AT-001, UT-018, IT-011 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-002, AC-012 | AT-002, UT-012 | Acceptance / Unit | Yes |
| FR-006 | AC-003, AC-013 | AT-002, UT-003, UT-013 | Acceptance / Unit | Yes |
| FR-007 | AC-014, AC-015 | AT-003, UT-002, REG-001, REG-005 | Acceptance / Unit / Regression | Yes |
| FR-008 | AC-016 | AT-004, REG-002 | Acceptance / Regression | Yes |
| FR-009 | AC-017, AC-018 | AT-004, UT-004, IT-001, IT-005, IT-014 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-019 | AT-004, UT-005, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-004, AC-020 | AT-005, UT-006, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-018, AC-021 | AT-005, UT-004, IT-006, IT-014 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-006, AC-022 | AT-003, UT-012, UT-013 | Acceptance / Unit | Yes |
| FR-014 | AC-005, AC-023, AC-041 | AT-006, UT-010, UT-016, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-024, AC-025 | AT-007, UT-008, UT-009, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-026, AC-027 | AT-008, IT-008, REG-007 | Acceptance / Integration / Regression | Yes |
| FR-017 | AC-007, AC-028 | AT-009, UT-010, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-018 | AC-029 | AT-009, UT-018, IT-009, IT-011 | Acceptance / Unit / Integration | Yes |
| FR-019 | AC-030 | AT-010, IT-009, IT-012 | Acceptance / Integration | Yes |
| FR-020 | AC-020, AC-025, AC-031 | AT-005, AT-007, UT-014, IT-007, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-021 | AC-032, AC-033 | AT-013, UT-001, UT-014, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-022 | AC-034 | AT-008, UT-015, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-023 | AC-041 | AT-006, UT-010, IT-003, REG-006 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-001 | AC-002, AC-035 | AT-002, AT-014, REG-009 | Acceptance / Regression | Yes |
| RULE-002 | AC-010 | AT-001, IT-004 | Acceptance / Integration | Yes |
| RULE-003 | AC-022 | AT-003, AT-014, UT-013 | Acceptance / Unit | Yes |
| RULE-004 | AC-012, AC-022, AC-029 | AT-002, AT-003, AT-009, AT-014, UT-012, UT-017 | Acceptance / Unit | Yes |
| RULE-005 | AC-011, AC-030 | AT-001, AT-010, IT-011, IT-012 | Acceptance / Integration | Yes |
| NFR-001 | AC-036 | AT-011, IT-010, REG-008 | Acceptance / Integration / Regression | Yes |
| NFR-002 | AC-037 | AT-012, UT-003, UT-013, UT-016 | Acceptance / Unit | Yes |
| NFR-003 | AC-038 | AT-012, REG-010, MAN-001 | Acceptance / Regression / Manual | Yes, plus manual |
| NFR-004 | AC-008, AC-030, AC-039 | AT-002, AT-010, IT-011, IT-012, REG-004 | Acceptance / Integration / Regression | Yes |
| NFR-005 | AC-040 | AT-012, UT-017, IT-002 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Select Account Type And Reveal Only Eligible Owner Settings

Requirement IDs: BR-001, FR-002, FR-003, FR-004, RULE-002, RULE-005
Acceptance Criteria: AC-001, AC-009, AC-010, AC-011
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/account_page_test.dart`, `app/test/settings/settings_page_test.dart`, `app/test/router/business_settings_routes_test.dart`

```gherkin
Feature: Business account selection
  Scenario: A regular member switches to business
    Given the active account is authoritatively regular
    When the member selects Business in Account settings
    Then one account-type PUT starts and the selector is busy
    And duplicate input cannot start another PUT
    When the request succeeds
    Then Business is selected
    And the main Settings page shows a Business section with Events and Products
    And the active account profile state reconciles without requiring sign-in again

  Scenario: A business member switches to regular without deleting data
    Given the account has a declaration, products, and events
    When the member selects Regular
    Then no confirmation is shown
    And no declaration or event delete request is sent
    And the Business settings section and owner business tabs disappear
    When the member switches back to Business and refreshes
    Then the retained eligible declaration, products, and events return

  Scenario: Account-type failure keeps the prior state
    Given the active account is regular
    When the Business mutation fails
    Then Regular remains selected
    And the selector becomes usable
    And localized established error feedback appears

  Scenario: A regular account cannot use an owner-management deep link
    Given the active account is regular
    When an Events or Products management route is opened directly
    Then no owner management controls are exposed
    And navigation returns to an appropriate settings surface
```

### AT-002: Present A Self-Declared Business Without Leaking Hidden Data

Requirement IDs: BR-002, FR-001, FR-005, FR-006, RULE-001, RULE-004, NFR-004
Acceptance Criteria: AC-002, AC-003, AC-008, AC-012, AC-013, AC-035, AC-039
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/profile_page_test.dart`, `app/test/profile/widgets/profile_about_tab_test.dart`, `app/test/profile/widgets/business_profile_summary_test.dart`

```gherkin
Feature: Business profile presentation
  Scenario: Eligible business details augment ordinary identity
    Given a visible business profile with all known declaration fields
    When the profile renders
    Then plain localized Business text appears without a verification icon
    And tagline and the localized primary action appear near existing identity information
    And About shows known business types, offerings, locality and country, service area, and hours
    And ordinary display name, bio, avatar, banner, crafts, and stats retain their existing ownership and presentation

  Scenario: Optional and unknown values degrade safely
    Given a business declaration with absent optional fields and safe unknown catalog values
    When About renders
    Then absent sections consume no placeholder space
    And unknown values have readable fallback labels
    And no search, permission, or category behavior is inferred

  Scenario: Blocked and omitted server data remains hidden
    Given a blocked profile shell omits account type and business data
    When Flutter decodes and renders it
    Then no business label, details, action, product tab, or event tab appears
    And Flutter does not reconstruct omitted destinations or content
```

### AT-003: Compose Business Tabs And Launch External Products

Requirement IDs: BR-004, FR-007, FR-013, RULE-003, RULE-004
Acceptance Criteria: AC-006, AC-014, AC-015, AC-022
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/profile_page_test.dart`, `app/test/profile/widgets/profile_tab_bar_business_test.dart`, `app/test/business/widgets/product_card_test.dart`

```gherkin
Feature: Business profile tabs
  Scenario Outline: Tab composition follows account type and visibility
    Given a <viewer> views a <profile> with <products> products and <events> upcoming events
    When profile data resolves
    Then Products is <productsTab>
    And Upcoming Events is <eventsTab>
    And all ordinary tabs remain in stable order

    Examples:
      | viewer | profile | products | events | productsTab | eventsTab |
      | owner | business | zero | zero | visible with setup state | visible with setup state |
      | visitor | business | one | zero | visible | visible with visitor empty state |
      | visitor | business | zero | one | visible with visitor empty state | visible |
      | visitor | business | zero | zero | visible with visitor empty state | visible with visitor empty state |
      | visitor | regular | one | one | absent | absent |
      | visitor | blocked | one | one | absent | absent |

  Scenario: A product card is external commerce only
    Given ordered products with normalized images and an optional authored price
    When the Products tab renders
    Then cards preserve authored order and show image, title, localized price, and external-link affordance
    When a card is activated
    Then only its hydrated HTTPS destination is handed to the external launcher
    And no native product detail, checkout, inventory, or availability claim appears
    And launcher failure leaves the profile stable with localized feedback
```

### AT-004: Save Ordinary And Business Profile Fields With Partial Reconciliation

Requirement IDs: FR-008, FR-009, FR-010
Acceptance Criteria: AC-016, AC-017, AC-018, AC-019
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/edit_profile_dialog_test.dart`, `app/test/profile/providers/save_profile_provider_test.dart`

```gherkin
Feature: Combined profile editor
  Scenario: Business fields are conditional and prefilled
    Given a regular active account
    Then Edit Profile contains only existing ordinary fields
    Given a business active account with a declaration
    Then Edit Profile additionally contains prefilled types, offerings, tagline, hours, service area, locality/country, and primary action
    And known selections and field limits match the approved server contract

  Scenario Outline: Save sends only changed records
    Given ordinary and business forms have <changes>
    When Save is pressed
    Then <ordinaryCalls> ordinary profile request is sent
    And <businessCalls> complete known declaration replacement is sent
    And unchanged products remain in the declaration body

    Examples:
      | changes | ordinaryCalls | businessCalls |
      | ordinary only | one | zero |
      | business only | zero | one |
      | both | one | one |

  Scenario Outline: One record succeeds and the other fails
    Given both records are dirty
    When the <success> save succeeds and the <failure> save fails
    Then the successful values become the new visible baseline
    And only the failed values remain dirty
    And the dialog stays open with accurate partial-failure feedback
    When Save is retried
    Then only the failed record is submitted

    Examples:
      | success | failure |
      | ordinary | business |
      | business | ordinary |
```

### AT-005: Curate Products Without Erasing Business Details

Requirement IDs: BR-003, FR-011, FR-012, FR-020
Acceptance Criteria: AC-004, AC-018, AC-020, AC-021, AC-031
Priority: Must
Level: Acceptance
Automation Target: `app/test/business/pages/products_settings_page_test.dart`, `app/test/business/widgets/product_editor_test.dart`

```gherkin
Feature: Featured product management
  Scenario: Add, edit, reorder, and remove valid cards
    Given a business declaration with details and ordered products
    When the owner edits product title, destination, image, alt, price, and order
    Then at most four cards can be saved
    And required title, HTTPS destination, and image are validated
    And optional canonical amount/currency is validated
    And one declaration replacement uses the current CID
    And every non-product known declaration field is preserved

  Scenario: Image work is explicit and recoverable
    Given a saved product image
    When no image change is made
    Then the existing blob metadata is round-tripped unchanged
    When a replacement upload fails or is cancelled
    Then the saved image is not replaced
    And retry remains available

  Scenario: A stale declaration is not overwritten
    Given another editor has changed the declaration CID
    When the Products page saves with the stale CID
    Then conflict feedback is shown
    And no blind overwrite occurs
    And Reload presents the current complete details and products before retry
```

### AT-006: Manage Events In Upcoming And History Views

Requirement IDs: BR-003, FR-014, FR-023
Acceptance Criteria: AC-005, AC-023, AC-041
Priority: Must
Level: Acceptance
Automation Target: `app/test/business/pages/events_settings_page_test.dart`, `appview/internal/api/business_event_management_acceptance_test.go`, `appview/internal/api/business_event_pagination_test.go`

```gherkin
Feature: Owner event management views
  Scenario: Upcoming contains active events nearest-first
    Given scheduled future, ongoing, publicly suppressed future, and historical owner events
    When the owner opens Upcoming
    Then requests use filter=upcoming
    And scheduled future and ongoing events appear by startsAt ascending then URI ascending
    And publicly suppressed active events remain visible with diagnostics
    And ended, cancelled, postponed, and unknown-status events are absent

  Scenario: History contains non-active events most-recent-first at its own cutoff
    Given the same retained owner events
    When the owner opens History
    Then requests use filter=history
    And ended, cancelled, postponed, unknown-status, and otherwise non-active events appear by startsAt descending then URI descending
    And each History traversal classifies every record once relative to its own cutoff

  Scenario: Each view paginates with its own frozen partition
    Given each view has multiple pages
    When later pages are loaded
    Then the opaque cursor is forwarded without client parsing
    And that view's first-page cutoff, filter, and ordering remain bound
    And a different positive limit may be used
    And rows append once without client-side repartitioning
    When an Upcoming cursor is sent with filter=history or without filter
    Then 400 invalid_cursor is returned in the standard envelope
    And only the requested view restarts safely from page one

  Scenario: Independent cutoffs do not promise a cross-view snapshot
    Given Upcoming and History first pages are requested at different server times
    When an event ends between those requests
    Then each individual traversal remains correct for its own cutoff
    And temporary cross-view overlap or omission is permitted until refresh
    When the user refreshes a view or a lifecycle mutation succeeds
    Then only affected accumulated pages and cursors are discarded and restarted

  Scenario Outline: Invalid filter and cursor combinations are rejected exactly
    When the owner requests <query>
    Then the response is <status> with error <error> in the standard envelope

    Examples:
      | query | status | error |
      | filter=unknown | 400 | invalid_filter |
      | filter= | 400 | invalid_filter |
      | filter=upcoming&filter=history | 400 | invalid_filter |
      | cursor=malformed | 400 | invalid_cursor |
      | filter=upcoming with unfiltered cursor | 400 | invalid_cursor |
      | no filter with filtered cursor | 400 | invalid_cursor |
      | filter=history with Upcoming cursor | 400 | invalid_cursor |
    And omitting filter preserves the approved all-events ordering and response
    And unknown query parameters remain ignored
```

### AT-007: Author Valid Timed And All-Day Events

Requirement IDs: FR-015, FR-020
Acceptance Criteria: AC-024, AC-025, AC-031
Priority: Must
Level: Acceptance
Automation Target: `app/test/business/widgets/event_editor_test.dart`, `app/test/business/models/event_draft_test.dart`

```gherkin
Feature: Event authoring
  Scenario: Create a timed event
    Given a business owner enters every required event field and optional details
    When Create is pressed
    Then local date/time and IANA timezone become canonical whole-second UTC startsAt and endsAt
    And no createdAt field is sent
    And valid optional image, summary, venue, event URI, and registration URI are included

  Scenario: Edit preserves server-owned and untouched data
    Given an existing event with CID, createdAt, image, and optional fields
    When the owner edits status and dates without editing the image
    Then the PUT uses the current CID
    And omits createdAt
    And preserves the existing image payload
    And cleared optional fields are absent from the full replacement as required

  Scenario: All-day DST boundaries remain local midnights
    Given an all-day event crosses a daylight-saving transition
    When local midnight boundaries are converted in the selected IANA timezone
    Then the UTC instants reflect the offset transition
    And the exclusive end remains after the start
```

### AT-008: Resolve Event Conflicts, Deletion, And Partition Movement

Requirement IDs: FR-016, FR-022
Acceptance Criteria: AC-026, AC-027, AC-034
Priority: Must
Level: Acceptance
Automation Target: `app/test/business/providers/event_mutation_provider_test.dart`, `app/test/business/pages/events_settings_page_test.dart`

```gherkin
Feature: Event mutation reconciliation
  Scenario: Stale update and delete never overwrite
    Given the event CID has changed on the server
    When an update or confirmed delete uses the stale CID
    Then conflict feedback offers Reload and Retry
    And no newer event state is overwritten

  Scenario: Delete is destructive while lifecycle changes are edits
    Given an existing event
    When Delete is selected
    Then a destructive confirmation is required before one DELETE
    When Cancelled or Postponed is selected
    Then one full event update is used instead of deletion

  Scenario: Successful changes reconcile every loaded surface
    Given an event is loaded in management, public upcoming, and detail state
    When a mutation succeeds
    Then accepted values remain visible while reads return the exact pre-write CID
    And a read with the accepted CID clears the accepted overlay
    And a read with any different CID is adopted as concurrent authoritative divergence
    And no CID is compared for chronological order
    And an event moving between Upcoming and History appears in only its new view
    And a deleted event remains absent while its deleted CID is projected, settles on not-found, and adopts a different-CID recreation
```

### AT-009: Browse Upcoming Events And Open Complete Detail

Requirement IDs: BR-004, FR-017, FR-018, RULE-004
Acceptance Criteria: AC-007, AC-028, AC-029
Priority: Must
Level: Acceptance
Automation Target: `app/test/business/widgets/upcoming_events_tab_test.dart`, `app/test/business/pages/event_detail_page_test.dart`, `app/test/router/business_event_routes_test.dart`

```gherkin
Feature: Public upcoming event experience
  Scenario: Upcoming pages remain in server order
    Given a visible business has multiple upcoming pages
    When the tab loads and requests more
    Then initial loading, retryable error, incremental loading, and end-of-list states are distinct
    And items remain startsAt ascending then URI ascending without duplicates
    And the opaque cursor is forwarded unchanged

  Scenario: Owner and visitor empty behavior differs without hiding the tab
    Given no upcoming events
    Then the owner tab shows a localized setup action to Events settings
    And a visitor keeps the Upcoming Events tab with localized visitor-empty text

  Scenario: Visitor failure remains recoverable in the stable tab
    Given a normally visible business profile
    When its initial upcoming request fails
    Then Upcoming Events remains visible with localized Retry and no false empty state
    Given confirmed upcoming items are already visible
    When refresh or incremental loading fails
    Then confirmed items remain visible with retry feedback
    And Flutter never removes an AppView-returned item by applying its own status or lifecycle filter

  Scenario: A card opens exact in-app detail
    Given a visible event card identified by owner DID and rkey
    When it is activated
    Then the typed event detail route opens that exact record
    And all hydrated date, role, mode, summary, venue, image, and lifecycle fields render when present
    And absent optional external actions render no empty controls
    And available event and registration actions launch only their hydrated URIs
```

### AT-010: Report Visible Events And Hide Unavailable Ones

Requirement IDs: FR-019, RULE-005, NFR-004
Acceptance Criteria: AC-030, AC-039
Priority: Must
Level: Acceptance
Automation Target: `app/test/business/pages/event_detail_page_test.dart`, `app/test/moderation/widgets/report_flow_test.dart`

```gherkin
Feature: Event detail moderation boundary
  Scenario: A visitor reports a visible event
    Given a non-owner views an eligible event detail
    When they submit an existing report reason
    Then one event report request uses the owner DID and rkey
    And established pending, success, validation, and failure feedback applies

  Scenario: Owner detail does not offer self-reporting
    Given the event owner opens their event detail
    Then owner edit/management actions may appear
    And the visitor report action is absent

  Scenario: Not-found replaces stale detail safely
    Given an event becomes absent, blocked, or moderated
    When detail refresh or an action receives event_not_found
    Then event content and external/report actions are removed
    And a localized unavailable state appears
    And no suppressed reason or hidden content leaks
```

### AT-011: Keep Business State Inside The Active Account Boundary

Requirement IDs: NFR-001
Acceptance Criteria: AC-036
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/providers/account_boundary_provider_test.dart`, `app/test/business/providers/business_account_boundary_test.dart`

```gherkin
Feature: Account-scoped business state
  Scenario: Late account A work cannot publish into account B
    Given Alice starts each supported profile, product, event-list, detail, upload, and mutation operation
    And Alice has unsaved drafts, declaration CID, cursors, and accepted-state overlays
    When the active account switches to Bob before completion
    Then Alice's late values, errors, messages, and rollback affect no Bob provider or screen
    And Bob cannot submit Alice's CID, draft, cursor, or event identity
    When switching back after the unsaved-work decision
    Then each account loads only its own authoritative state
```

### AT-012: Enforce Localization, Accessibility, Layout, And Telemetry Boundaries

Requirement IDs: NFR-002, NFR-003, NFR-005
Acceptance Criteria: AC-037, AC-038, AC-040
Priority: Must
Level: Acceptance
Automation Target: business/profile/settings widget tests; `app/test/l10n/business_profile_l10n_test.dart`; observability architecture tests

```gherkin
Feature: Business UI quality boundary
  Scenario: Visible values and controls are localized and semantic
    Given Account settings, business summary/About, Edit Profile, Products manager/editor, Events manager/editor, and event detail/report
    And the closed diagnostic catalog owner-not-business, invalid-time-range, duration-exceeds-limit, record-moderated, ended, cancelled, and postponed
    When rendered in the app's supported English locale with semantics enabled
    Then visible copy and semantic names come from generated localization
    And date, time, country, known catalogs, diagnostics, and currency display follow the active locale
    And wire values remain canonical
    And selected, busy, validation, external, and destructive states are announced

  Scenario: Supported constraints remain operable
    Given 320x568 and 800x600 logical-pixel viewports at text scales 1.0 and 2.0
    And keyboard traversal plus semantics actions are enabled
    When forms, tabs, cards, dialogs, and pages render
    Then no overflow or unreachable control occurs
    And focus order and touch targets follow existing conventions

  Scenario: Authored values do not enter background network or telemetry
    Given distinct sentinel destination, email, free-text, title, price, location, alt, DID, and rkey values
    When success and failure paths execute
    Then the recording Dio/HTTP adapter and fake launcher record no background fetch, preview, DNS-resolution adapter call, or PDS read for destinations
    And captured application logs, error-reporter/Sentry events and breadcrumbs, traces, metrics labels, and route diagnostics contain no sentinel value
```

### AT-013: Return Conflict-Safe Business And Image Projections

Requirement IDs: FR-021
Acceptance Criteria: AC-032, AC-033
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_profile_acceptance_test.go`, `appview/internal/api/business_event_acceptance_test.go`, Flutter model tests

```gherkin
Feature: Client-enabling AppView projection
  Scenario: Eligible business data carries its declaration version
    Given a normally visible eligible declaration
    When the profile response is built
    Then business.cid equals the projected current declaration CID
    And Flutter can use it as If-Match for replacement
    Given a missing declaration or blocked shell
    Then no business CID is exposed

  Scenario: Product and event images use normalized image views
    Given supported source blobs with alt and aspect ratio
    When product and event responses are hydrated
    Then each image JSON has required cid, supported mime, nonnegative integer size, and alt
    And optional aspectRatio has positive integer width and height
    And JPEG, PNG, and WebP have nonempty thumb and fullsize
    And profile products, owner event lists, public event lists, and event detail share that exact object
    And cid, mime, size, alt, and aspectRatio reconstruct the blob ref, mimeType, size, alt, and aspectRatio of an unchanged mutation without display URLs
    Given an unsupported, malformed, missing, or otherwise unsafe source image
    Then the image object is omitted and the containing item remains safe
```

### AT-014: Add No Verification, Commerce, Ranking, Or Destination Privilege

Requirement IDs: RULE-001, RULE-003, RULE-004
Acceptance Criteria: AC-022, AC-035
Priority: Must
Level: Acceptance
Automation Target: business/profile widget contract tests and existing AppView business policy-neutrality tests

```gherkin
Feature: Business policy neutrality
  Scenario: Business state changes presentation only
    Given otherwise identical regular and business actors
    When Flutter renders and invokes supported business surfaces
    Then Business is plain self-declared presentation
    And no verification, subscription, ranking, reach, moderation-priority, or permission behavior differs
    And products offer no native commerce or seller-guarantee claim
    And only AppView-hydrated destinations can be launched
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-021 | AC-008, AC-032, AC-033 | Decode full-profile account type, partial/blocked business data, and exact image wire shapes. | Regular/business full profiles; blocked omission; partial business; unknown open values; exact supported/unsupported images; compact identity JSON containing unknown `accountType`. | Full profiles retain the field; blocked omission is accepted; existing compact identity mappers ignore it; unsupported URLs remain absent; malformed required image values fail safely. | `app/test/business/models/business_wire_models_test.dart`, `app/test/profile/models/profile_test.dart`, representative existing compact-model tests |
| UT-002 | FR-007 | AC-014, AC-015 | Derive one stable tab list from account type and block state. | Regular, normally visible business, and blocked compositions plus selected tab; empty/nonempty product and event transitions. | Both business tabs remain present through every normal business transition; regular/blocked profiles retain ordinary tabs only; selection, keys, and scroll identities remain stable. | `app/test/profile/models/profile_tabs_test.dart` |
| UT-003 | BR-002, FR-006, NFR-002 | AC-003, AC-013, AC-037 | Map known catalogs/location to generated localized labels and unknown safe values to bounded fallback text. | All known types/offerings/actions; unknown values; location combinations; supported English locale variants. | Canonical known labels, safe fallback, no inferred behavior, locale-correct country presentation. | `app/test/business/models/business_labels_test.dart` |
| UT-004 | FR-009, FR-012 | AC-018 | Build complete known declaration replacements for detail-only and product-only changes. | Existing declaration/CID with details, products, unknown server extensions represented outside client model. | All known fields preserved; changed area replaced; correct `If-Match`; create uses `*`; unknown preservation remains server responsibility. | `app/test/business/models/business_declaration_draft_test.dart` |
| UT-005 | FR-010 | AC-019 | Reduce per-record combined-save outcomes into baseline, dirty fields, retry plan, and feedback. | Both success; ordinary success/business failure; inverse; both failure. | Successful portion commits once; failed portion remains dirty; retry includes only failures; no successful rollback. | `app/test/profile/models/profile_save_result_test.dart` |
| UT-006 | FR-011 | AC-020 | Validate product count, required values, exact URI uniqueness, canonical price input, and authored ordering. | Zero to five products; duplicate/similar URIs; valid/invalid title, HTTPS, image, currency/amount. | Matches approved first-party limits; four accepted/five rejected; order retained; no aggressive URI normalization. | `app/test/business/models/product_draft_test.dart` |
| UT-007 | FR-014, FR-023 | AC-023, AC-041 | Classify an owner-event traversal into Upcoming or History from effective status/end and that traversal's cutoff independently of public suppression. | Scheduled future/ongoing/ended; cancelled; postponed; unknown; suppressed future; equal cutoff. | Exactly one classification per event for a supplied cutoff; suppressed active remains Upcoming; every other retained event is History; no cross-cutoff union assertion. | `appview/internal/business/event_management_filter_test.go` |
| UT-008 | FR-015 | AC-024, AC-025 | Convert timed/all-day local boundaries and IANA timezone into canonical whole-second UTC. | UTC, positive/negative offsets, DST spring/fall, local midnight, fractional seconds, invalid range. | Correct instants/exclusive end; invalid boundaries/ranges rejected before mutation. | `app/test/business/models/event_draft_test.dart` |
| UT-009 | FR-015 | AC-024, AC-025 | Serialize create/update event bodies. | Complete draft; create; update with stored createdAt; cleared optionals; unchanged image. | Create/update omit `createdAt`; required values always present; update preserves unchanged image; cleared optionals omitted correctly. | `app/test/business/models/event_mutation_test.dart` |
| UT-010 | FR-014, FR-017, FR-023 | AC-023, AC-028, AC-041 | Maintain independent page/cursor/error state for owner Upcoming, owner History, and visitor upcoming. | Initial/next pages, duplicates, invalid cursor, refresh, view switch, initial/incremental/refresh errors, AppView-returned unusual status. | Opaque cursor forwarded; rows append once in server order; visitor tab remains stable; confirmed rows survive errors; invalid cursor restarts only its scope; no client status filter or repartition. | `app/test/business/providers/event_list_provider_test.dart` |
| UT-011 | FR-002 | AC-001, AC-009 | Serialize account type and reduce pending/success/failure state. | Regular/business, rapid duplicate selection, API error. | Exact body; one in-flight request; prior confirmed type restored on error. | `app/test/business/providers/account_type_controller_test.dart` |
| UT-012 | FR-005, FR-013, RULE-004 | AC-012, AC-022, AC-029 | Map action type to localized label/icon and pass only hydrated destination to launcher abstraction. | Every known action; absent destination; HTTPS/mailto; launcher false/throw. | Correct label; omitted control when absent; exact destination handoff; sanitized failure. | `app/test/business/models/business_action_test.dart` |
| UT-013 | FR-006, FR-013, RULE-003, NFR-002 | AC-003, AC-022, AC-037 | Format seller-authored money, event dates, all-day ranges, and location for locale. | USD/JPY and representative locales; timed/all-day events; locality/country. | Locale-correct display without changing canonical source values or adding guarantees. | `app/test/business/models/business_formatters_test.dart` |
| UT-014 | FR-020, FR-021 | AC-020, AC-025, AC-031, AC-033 | Convert normalized image view to unchanged mutation image and model replace/remove/upload states. | Supported image metadata; unsupported URL; replacement success/failure/cancel; remove. | Exact blob/alt/aspect round-trip; no accidental replacement; display fallback safe. | `app/test/business/models/business_image_draft_test.dart` |
| UT-015 | FR-022 | AC-034 | Reconcile accepted local mutations by CID identity and request/account generation. | Create/update/delete; pre-write CID/absence; accepted CID; same pre-write read; accepted read; third CID before accepted; absence; refresh failure; explicit reload; stale generation/account. | Accepted overlay persists only for exact pre-write/absence lag, settles on accepted/absence, adopts third-CID divergence, survives failure, and ignores stale completions without ordering CIDs. | `app/test/business/providers/business_reconciliation_test.dart` |
| UT-016 | FR-014, NFR-002 | AC-023, AC-037 | Map the closed suppression/exclusion catalog to localized bounded owner copy. | `owner-not-business`, `invalid-time-range`, `duration-exceeds-limit`, `record-moderated`, `ended`, `cancelled`, `postponed`; empty and duplicate input. | Stable canonical labels; empty diagnostics omitted visually; no raw diagnostics treated as fields. | `app/test/business/models/event_diagnostics_test.dart` |
| UT-017 | RULE-004, NFR-005 | AC-040 | Verify outbound and observability boundaries cannot fetch or record authored sentinel values. | Distinct URI/email/text/title/price/location/alt/DID/rkey sentinels through recording Dio/HTTP, launcher, logger, Sentry/error reporter, trace, metric, and route-diagnostic adapters. | Only user activation calls the launcher; no destination fetch; every captured sink is free of sentinels and contains bounded operation/result fields only. | `app/test/business/business_privacy_architecture_test.dart`, AppView observability tests |
| UT-018 | FR-004, FR-018 | AC-011, AC-029 | Build typed owner/detail route locations and apply owner-management guard. | Business/regular account; owner DID/rkey; malformed identifiers. | Canonical routes; regular guard returns settings; detail identifies exact valid record. | `app/test/router/business_event_routes_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-002, FR-009 | AC-001, AC-017, AC-018 | Flutter API clients use exact account/declaration routes, bodies, and precondition headers. | Dio with `DioAdapter`; regular/business response; existing/new declaration. | Invoke account type and declaration mutations. | Exact camelCase payloads; `If-Match` CID or `*`; standard mapped errors; no PDS token/direct URL. | `app/test/business/data/business_api_client_test.dart` |
| IT-002 | FR-021, NFR-005 | AC-032, AC-033, AC-040 | AppView emits declaration CID and exact reusable product/event image JSON without fetching authored destinations. | Go handler/store fixtures with JPEG/PNG/WebP/unsupported/malformed/missing blobs, aspect ratio, blocked shell, destination fetch canary. | GET profile, owner event list, public event list, and event detail responses. | CID only on eligible declaration; exact required/optional `PostImageView` keys/types and both URLs on every safe-image surface; unsafe images and blocked data omitted; zero destination fetches. | `appview/internal/api/business_profile_acceptance_test.go`, `business_event_acceptance_test.go`, `business_security_acceptance_test.go` |
| IT-003 | BR-003, FR-014, FR-023 | AC-005, AC-023, AC-041 | Real-Postgres owner event filters order and paginate with independent frozen cutoffs and exact admission. | Events across all statuses/time boundaries/seven reason codes/equal starts/multiple pages; deterministic advancing clock; every cursor/filter combination. | Traverse unfiltered, Upcoming, and History at same and different cutoffs; vary later-page limit; cross-use/malformed cursors; send unknown/empty/repeated filters and unknown params. | Each traversal is complete once for its own cutoff/order; independent views make no cross-snapshot promise; exact `400 invalid_filter`/`invalid_cursor` envelopes; unknown params ignored; unfiltered unchanged. | `appview/internal/api/business_event_management_acceptance_test.go`, `business_event_pagination_test.go` |
| IT-004 | BR-001, FR-002, FR-003, RULE-002 | AC-001, AC-009, AC-010 | Account-type controller reconciles settings/profile and issues no record deletes. | Provider container with recording repository and retained data. | Switch regular/business; inject success/failure. | One PUT; immediate account-scoped state; failure rollback; zero declaration/event DELETE; restored data on re-fetch. | `app/test/business/providers/account_type_controller_test.dart` |
| IT-005 | FR-009, FR-010 | AC-017, AC-018, AC-019 | Combined save orchestrator handles every per-record outcome. | Recording ordinary/business repositories with independently controlled futures. | Save ordinary only, business only, both; complete in either order with failures. | Changed requests only; complete declaration; successful baseline retained; failed-only retry; one close on full success. | `app/test/profile/providers/save_profile_provider_test.dart` |
| IT-006 | FR-012 | AC-021 | Declaration conflict reloads complete details/products and prevents overwrite. | Existing CID then server advances CID/state. | Save stale detail/product mutation; reload; retry. | 409 mapped to conflict UI state; no blind retry; complete new declaration shown; retry uses new CID. | `app/test/business/providers/business_profile_controller_test.dart`, AppView conflict tests |
| IT-007 | BR-003, FR-011, FR-020 | AC-004, AC-020, AC-031 | Product manager composes image upload and declaration replacement. | Recording blob client/business repository; existing details/products. | Add/replace/remove/reorder; upload success/failure/cancel. | Valid full replacement, preserved details/images, correct order/CID, bounded progress/error, no fifth card. | `app/test/business/providers/products_controller_test.dart` |
| IT-008 | FR-015, FR-016, FR-020 | AC-024, AC-025, AC-026, AC-027, AC-031 | Event API/repository performs exact CRUD with immutable createdAt and CID conflicts. | Dio adapter plus widget/provider controller; current event/CID/image. | Create, update, cancel, postpone, confirmed delete; stale CID. | Exact routes/body/headers; no client createdAt; image preserved; confirmation before DELETE; conflict reload path. | `app/test/business/data/event_api_client_test.dart`, `app/test/business/providers/event_mutation_provider_test.dart` |
| IT-009 | BR-004, FR-017, FR-018, FR-019 | AC-007, AC-028, AC-029, AC-030 | Public upcoming, detail, external actions, and reports use exact APIs/routes. | Business repository fake/Dio adapter; paginated events; unavailable/moderated responses. | Load pages, open DID/rkey detail, launch actions, report. | Server order/cursor; exact detail/report path; optional controls; mapped unavailable/report feedback. | `app/test/business/data/event_api_client_test.dart`, page/router widget tests |
| IT-010 | NFR-001 | AC-036 | Late account-A reads/uploads/mutations cannot publish into account B. | Two account leases; controlled completers for every business provider/controller. | Switch to B before A completions and errors. | No B state/message/navigation/request contains A data/CID/cursor/draft; switch-back refetch is isolated. | `app/test/business/providers/business_account_boundary_test.dart`, existing account boundary suite |
| IT-011 | FR-004, FR-018, RULE-005 | AC-011, AC-029 | Typed settings/detail routes preserve compact/wide shell and guard regular owner routes. | Real `goRouterProvider` at compact/wide sizes; regular/business profiles. | Navigate settings rows, deep links, detail, Back. | Canonical locations, appropriate shell/back stack, owner guard, exact detail identity. | `app/test/router/business_settings_routes_test.dart`, `business_event_routes_test.dart` |
| IT-012 | FR-001, FR-019, RULE-005, NFR-004 | AC-008, AC-030, AC-039 | Block/moderation/auth boundaries omit or remove business/event data. | Profile blocked shells; empty lists; event 404; unauthenticated/non-current errors. | Decode/render/refresh/report. | No hidden business/event data or owner controls; safe unavailable state; standard error mapping. | `app/test/profile/profile_page_test.dart`, `app/test/business/pages/event_detail_page_test.dart`, AppView redaction tests |
| IT-013 | FR-022 | AC-034 | Accepted mutation overlays survive projection lag and converge by CID identity. | Controlled create/update/delete responses followed by exact pre-write CID, accepted CID, third CID, absence, failure, account switch, and superseded-generation reads. | Complete mutations and emit each sequence. | Exact pre-write lag cannot revert accepted state; accepted/absence settles; third CID wins as divergence; failures retain retryable overlay; stale scope/generation cannot publish. | `app/test/business/providers/business_reconciliation_test.dart` |
| IT-014 | FR-009, FR-012 | AC-018 | AppView preserves unknown top-level declaration extensions for both Flutter replacement modes. | Authoritative declaration with complete known fields and nested unknown extension containing a large integer. | Send one detail-only complete-known replacement and one product-only complete-known replacement through `PutBusinessProfileHandler`. | Changed known area replaces, every other submitted known field survives, omitted known fields follow replacement semantics, and the unknown extension remains byte/value-equivalent without numeric precision loss. | `appview/internal/api/business_profile_test.go`, `appview/internal/business/profile_merge_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Regular and non-business profiles keep the existing five tabs and order. | FR-007 | Existing profile tests assert Projects, Posts, Comments, Reposts, About with no Products/Upcoming Events. |
| REG-002 | Regular Edit Profile fields, dirty tracking, images, full ordinary payload, unsaved-work confirmation, and close-on-success remain unchanged. | FR-008 | Extend `edit_profile_dialog_test.dart` without weakening existing assertions. |
| REG-003 | Existing Settings identity, Preferences, Connections, Discovery, General, sign-out, and Account deletion remain reachable. | FR-004 | Settings tests assert old rows/order and conditional Business insertion only for business. |
| REG-004 | Blocked profile shells expose only their existing reduced identity/actions. | NFR-004 | Existing block profile tests additionally assert all business surfaces absent. |
| REG-005 | Dynamic business tabs do not regress profile customisation boundary, header collapse, selected tab, or per-tab scroll restoration. | FR-007, NFR-003 | Existing profile page/customisation/tab tests run for regular and business compositions. |
| REG-006 | Unfiltered `GET /v1/events` retains default 20/max 50, `startsAt DESC, URI DESC`, diagnostics, and opaque seek behavior. | FR-023 | Existing owner-management acceptance traversal remains unchanged with no filter. |
| REG-007 | Existing event CRUD, validation, moderation, direct reads, and public upcoming eligibility remain unchanged. | FR-016 | Existing AppView business event suites pass alongside new filter tests. |
| REG-008 | Existing account switch unsaved-work guard and account invalidation still protect all other account-scoped features. | FR-003, NFR-001 | Existing account switch/boundary suites pass with business invalidators added. |
| REG-009 | Account type remains policy-neutral across AppView summaries and Flutter non-profile surfaces. | RULE-001 | Existing `business_policy_neutrality_test.go` passes; Flutter adds no feed/search ranking or entitlement branch. |
| REG-010 | Existing responsive and accessibility baselines remain clean. | NFR-003 | `flutter analyze` and all Flutter tests pass; the AT-012 screen catalog runs at 320x568 and 800x600 with text scales 1.0/2.0 plus semantics and keyboard assertions. |
| REG-011 | Business declaration replacement preserves independently authored unknown top-level extensions. | FR-009, FR-012 | Existing `TestBusinessProfileLifecycleUsesConditionalPDSWrites` and `TestProfileReplacementMergeAndSafeHydration` remain green; IT-014 expands both detail-only and product-only cases. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Account/profile visibility matrix | Regular owner, business owner without declaration, full business owner, visitor, blocked/blocking profiles. | AT-001–AT-004, AT-011, IT-004, IT-012 |
| TD-002 | Full business declaration | Current CID; all known types/offerings; tagline; hours; service area; locality/country; every action; four ordered products; unknown safe values. | AT-002, AT-004, AT-005, UT-001–UT-006 |
| TD-003 | Product validation matrix | Zero/four/five cards; duplicate and merely similar URIs; missing title/image; valid/invalid HTTPS; USD/JPY canonical/noncanonical values; supported/unsupported image MIME. | AT-003, AT-005, UT-006, UT-014, IT-007 |
| TD-004 | Event partition matrix | Scheduled future, ongoing, equal cutoff, ended, cancelled, postponed, unknown status, invalid range, over-duration, moderated/suppressed future, equal starts/different URI. | AT-006–AT-010, UT-007–UT-010, IT-003, IT-008, IT-009 |
| TD-005 | Time-zone matrix | UTC, Europe/London spring/fall DST, America/New_York, Asia/Tokyo; timed and all-day exclusive-end ranges. | AT-007, UT-008, UT-013 |
| TD-006 | Conflict/projection sequence | Pre-write CID v1/absence, 409 after concurrent CID v2, accepted CID v3, delayed exact-v1/absence read, accepted-v3 read, opaque third CID, delete/not-found, failure, stale generation/account. | AT-004, AT-005, AT-008, UT-005, UT-015, IT-005, IT-006, IT-013 |
| TD-007 | Multi-account concurrency | Alice business with drafts/CIDs/cursors and controlled futures; Bob regular/business with distinct identifiers/data. | AT-011, IT-010, REG-008 |
| TD-008 | Localization/accessibility | Supported English locale and representative English locale variants for formatters; 320x568 and 800x600 logical pixels; text scales 1.0/2.0; keyboard/semantics enabled; all screens named in AT-012. | AT-012, UT-003, UT-013, UT-016, MAN-001 |
| TD-009 | Privacy canaries | Distinct URI, mailto, title, summary, locality, price, alt, owner DID/rkey sentinels captured through recording network, launcher, logger, Sentry/error-reporter, trace, metric, and route-diagnostic adapters. | AT-012, UT-017, IT-002, IT-012 |
| TD-010 | Image round-trip | JPEG/PNG/WebP blobs with CID/MIME/size/alt/aspect and expected thumb/full-size URLs; unsupported MIME without URLs. | AT-005, AT-007, AT-013, UT-014, IT-002 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | NFR-003 | Real assistive-technology and keyboard pass | On one mobile device with VoiceOver or TalkBack and one desktop target with keyboard, traverse account selector, business summary/actions, tabs, product/event forms, two-view event manager, pagination/retry, detail/report, and delete confirmation at large text. | Reading/focus order is logical; state and errors are announced; no trap, clipped control, inaccessible reorder action, or unlabeled image/action exists. |
| MAN-002 | FR-005, FR-013, FR-018, NFR-003 | Operating-system external handoff and responsive visual check | On supported mobile platforms, activate HTTPS and mailto primary actions, product links, event links, and registration links; repeat with no handler where feasible; inspect compact and wide presentation. | Correct external app chooser/handler opens only after activation; failure returns localized feedback; Craftsky remains stable; layouts retain clear self-declared/external affordances. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Real external app behavior varies by OS and installed handlers. | FR-005, FR-013, FR-018 | Unit/widget tests can verify exact launcher-adapter calls but not every operating-system chooser. | Keep launcher tests mandatory and run MAN-002 on supported release platforms. |
| GAP-002 | Screen-reader announcements and reorder ergonomics vary by platform. | FR-011, NFR-003 | Flutter semantics tests cannot fully reproduce VoiceOver/TalkBack interaction. | Keep automated semantics/focus tests mandatory and run MAN-001 before release. |
| GAP-003 | Independent view cutoffs and concurrent event mutations can produce normal cross-view and seek-pagination changes. | FR-014, FR-022, FR-023 | Each cutoff freezes only its traversal's time classification, not a shared snapshot or inserts, deletes, and ordering-key edits. | Test each unchanged traversal's completeness, permit temporary cross-view overlap/omission, and restart affected views after refresh/mutation. |

No blocking gap is identified.

## 10. Out Of Scope

- Lexicon/schema, migration, persistence, authentication, eligibility, taxonomy, and moderation-policy tests beyond regression of the approved AppView work.
- Native commerce, inventory, checkout, global event discovery, recurrence, attendees, ticketing, reminders, maps, and calendar export.
- Editing business records from regular-account first-party UI.
- Visible business badges across feed, search, notification, and relationship summaries.
- Pixel-perfect golden tests; responsive widget constraints and a focused manual visual pass provide more maintainable coverage for this slice.
- Live third-party destination content, DNS, or availability tests; Craftsky must not fetch those destinations.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-08-29-flutter-business-profiles/01-requirements.md`
- Test specification: `docs/changes/2026-08-29-flutter-business-profiles/02-acceptance-tests.md`
- Next review artifact: `docs/changes/2026-08-29-flutter-business-profiles/03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-08-29-flutter-business-profiles/`
- Recommended first failing test for implementation: `IT-002` AppView eligible business projection returns declaration CID and normalized product/event images while blocked/missing shapes omit CID.
- Suggested test order for implementation:
  1. IT-002 and AT-013: AppView CID/image prerequisite.
  2. IT-003, UT-007, and AT-006: owner Upcoming/History filters and cursor contract.
  3. UT-001, IT-001, and AT-001/AT-002: Flutter models, clients, account type, and presentation.
  4. UT-002 and AT-003: dynamic profile tabs and product presentation.
  5. UT-004–UT-006, IT-005–IT-007, and AT-004/AT-005: declaration editing, partial saves, products, images, conflicts.
  6. UT-008–UT-010, IT-008/IT-009, and AT-007–AT-010: event forms, management, public detail, reporting.
  7. UT-015, IT-010/IT-013, and AT-008/AT-011: projection lag and account isolation.
  8. AT-012/AT-014, regressions, full analysis/tests, then manual checks.
- Commands discovered:
  - `just app-test`
  - `just app-analyze`
  - `just test`
  - `cd app && dart run build_runner build --delete-conflicting-outputs`
  - Focused Flutter: `cd app && flutter test test/profile test/settings test/router test/business`
  - Focused AppView: `cd appview && go test ./internal/api/... ./internal/business/...`
- Blocking gaps: None.
