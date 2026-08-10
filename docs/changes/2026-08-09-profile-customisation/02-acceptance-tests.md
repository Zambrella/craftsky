# Acceptance Test Specification: Public Profile Customisation

## 1. Test Strategy

Risk level: Medium, carried forward from `01-requirements.md`.

The feature changes a public identity contract used across most avatar-bearing responses, adds AppView-owned durable state and an authenticated mutation, and introduces scoped theming at shared Flutter rendering seams. Verification therefore uses complementary layers:

- Go database and store integration tests prove defaults, atomic full replacement, concurrent last-write-wins behavior, membership lifecycle cleanup, account isolation, and indexed set-based hydration against real Postgres.
- Go request, handler, route, response-shape, and observability tests prove the authenticated/device-bound JSON contract, strict catalogue validation, backwards-compatible public identity enrichment, moderation policy, bounded telemetry, and the no-PDS/no-Tap/no-blob boundary.
- Flutter model, API-client, provider, router, and widget tests prove tolerant decoding, per-field fallback, active-account fencing, confirmed-versus-draft editing behavior, exact feedback, accessible controls, shared inside-border rendering, local texture use, and exact compact/full profile theme boundaries.
- Cross-surface regression matrices protect posts, threads, quotes, notifications, search, relationship/account lists, navigation/account switching, and existing profile content from presentation or contract regressions.
- Manual verification is limited to final aesthetic review of the approved colour/texture bundles and physical assistive-technology output; structural bounds, semantics, state, and contrast thresholds remain automated.

All tests below are implementation designs. No application code, test code, migration, generated output, or dependency is created or run during this stage.

All gated implementation inputs closed on 2026-08-10. The seven colour bundles, foreground-tinted textures at 18% opacity, and exact failure feedback are recorded in `01-requirements.md` Q11–Q13.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-003, AC-016 | AT-001, AT-003, IT-001, IT-002, IT-007 | Acceptance / Integration | Yes |
| BR-002 | AC-002, AC-008, AC-009, AC-010 | AT-002, AT-004, UT-004, UT-005, REG-004 | Acceptance / Unit / Regression | Yes |
| BR-003 | AC-011 | AT-005, UT-007, REG-005, MAN-002 | Acceptance / Unit / Regression / Manual | Partial |
| BR-004 | AC-003, AC-006, AC-007, AC-014 | AT-003, AT-006, AT-007, UT-008, UT-009, IT-002 | Acceptance / Unit / Integration | Yes |
| BR-005 | AC-001, AC-004, AC-016 | AT-001, IT-002, IT-007, IT-009 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-002, AC-015 | AT-001, AT-002, AT-008, IT-004, IT-006 | Acceptance / Integration | Yes |
| FR-002 | AC-003, AC-004, AC-005, AC-017 | AT-003, IT-002, IT-003, IT-008 | Acceptance / Integration | Yes |
| FR-003 | AC-003, AC-005, AC-013 | AT-003, UT-002, IT-002, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-003, AC-004, AC-014, AC-016 | AT-003, AT-007, IT-001, IT-002, IT-007 | Acceptance / Integration | Yes |
| FR-005 | AC-006, AC-007, AC-018 | AT-006, UT-008, REG-003, REG-007, MAN-001 | Acceptance / Unit / Regression / Manual | Partial |
| FR-006 | AC-007, AC-014 | AT-006, AT-007, UT-009, UT-010, REG-008 | Acceptance / Unit / Regression | Yes |
| FR-007 | AC-008, AC-009, AC-010 | AT-004, UT-004, UT-005, REG-004 | Acceptance / Unit / Regression | Yes |
| FR-008 | AC-002, AC-008, AC-010 | AT-002, AT-004, REG-004 | Acceptance / Regression | Yes |
| FR-009 | AC-011, AC-012 | AT-005, UT-006, UT-007, REG-005, MAN-002 | Acceptance / Unit / Regression / Manual | Partial |
| FR-010 | AC-012, AC-014 | AT-005, AT-007, UT-006, REG-005, REG-008 | Acceptance / Unit / Regression | Yes |
| FR-011 | AC-005, AC-011, AC-012, AC-019 | AT-005, UT-001, UT-006, UT-007, IT-010, MAN-002 | Acceptance / Unit / Integration / Manual | Partial |
| FR-012 | AC-013, AC-017 | AT-009, UT-001, UT-003, IT-004, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-015 | AT-008, IT-006, REG-006 | Acceptance / Integration / Regression | Yes |
| NFR-001 | AC-002, AC-019 | AT-002, IT-005 | Acceptance / Integration | Yes |
| NFR-002 | AC-020 | IT-001, REG-001 | Integration / Regression | Yes |
| NFR-003 | AC-018 | AT-010, UT-011, MAN-001, MAN-003 | Acceptance / Unit / Manual | Partial |
| NFR-004 | AC-013, AC-015, AC-017 | AT-008, AT-009, UT-003, IT-006, IT-008, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| NFR-005 | AC-019 | IT-009 | Integration | Yes |
| NFR-006 | AC-020 | UT-001–UT-011, IT-001–IT-010, REG-001–REG-009, MAN-001–MAN-003 | All applicable levels | Partial until manual checks and open exact values are resolved |
| RULE-001 | AC-005, AC-012 | AT-005, UT-001, UT-002, UT-006 | Acceptance / Unit | Yes; palette approved 2026-08-10 |
| RULE-002 | AC-005, AC-009 | AT-004, UT-001, UT-002, UT-004, UT-005 | Acceptance / Unit | Yes |
| RULE-003 | AC-005, AC-011, AC-019 | AT-005, UT-001, UT-002, UT-007, IT-010, MAN-002 | Acceptance / Unit / Integration / Manual | Partial pending exact tint/opacity |
| RULE-004 | AC-001, AC-004, AC-015 | AT-001, AT-008, IT-002, IT-006 | Acceptance / Integration | Yes |
| RULE-005 | AC-003, AC-016 | AT-003, IT-007, REG-009 | Acceptance / Integration / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Return One Public Effective Customisation Everywhere Identity Is Exposed

Requirement IDs: BR-001, BR-005, FR-001, RULE-004\
Acceptance Criteria: AC-001, AC-002\
Priority: Must\
Level: Acceptance\
Automation Targets: `appview/internal/api/profile_response_test.go`, `appview/internal/api/post_response_test.go`, `appview/internal/api/notifications_test.go`, `appview/internal/api/search_response_test.go`, `appview/internal/api/relationship_list_test.go`

```gherkin
Feature: Public customisation in identity response objects
  Scenario Outline: An allowed viewer receives a customised member identity
    Given member Alice has saved colour "<colour>", border "<border>", and background "<background>"
    And viewer Bob is allowed to receive Alice's identity on <response surface>
    When Bob fetches that response
    Then Alice's existing identity object contains one nested "customisation" object
    And that object contains at least the string fields "colour", "profileBorder", and "profileBackground"
    And additive future customisation fields do not invalidate the response contract
    And the values equal Alice's effective saved values
    And no secondary customisation endpoint is required to render the response

    Examples:
      | response surface            | colour | border | background   |
      | full profile                | cobalt | thick  | bayerdark    |
      | self profile                | cobalt | thick  | bayerdark    |
      | post author                 | cobalt | thick  | bayerdark    |
      | reply/comment author        | cobalt | thick  | bayerdark    |
      | quote-preview author        | cobalt | thick  | bayerdark    |
      | available notification actor | cobalt | thick | bayerdark  |
      | profile search result       | cobalt | thick  | bayerdark    |
      | relationship/account item  | cobalt | thick  | bayerdark    |

  Scenario: Two viewers receive the same public effective values
    Given Alice has one saved customisation
    When Alice and allowed viewer Bob fetch Alice's profile
    Then both responses contain the same effective customisation values
    And no viewer-specific variant is returned
```

### AT-002: Render The Same Custom Avatar On Every Avatar-Bearing Surface

Requirement IDs: BR-002, FR-001, FR-008, NFR-001\
Acceptance Criteria: AC-002, AC-008, AC-010\
Priority: Must\
Level: Acceptance\
Automation Target: a shared surface-matrix widget test, plus the existing tests listed in REG-004

```gherkin
Feature: Consistent avatar customisation
  Scenario Outline: A customised identity is rendered through the shared avatar seam
    Given Alice's effective customisation uses the selected base colour and "thick" border
    And Alice appears on <surface>
    When the surface renders Alice's avatar
    Then exactly one circular coloured stroke is drawn inside the avatar bounds
    And the stroke uses Alice's selected base colour
    And the existing hard-offset ink shadow is retained where that surface already had it
    And no follow-up profile or customisation request is made

    Examples:
      | surface                         |
      | feed post                       |
      | thread root                     |
      | thread reply/comment            |
      | quote preview                   |
      | social notification             |
      | compact profile header          |
      | full profile header             |
      | profile search result           |
      | follower/following list         |
      | account or post summary         |
      | edit/customisation preview      |
      | app navigation/account switcher |
```

### AT-003: Save A Complete AppView-Owned Customisation

Requirement IDs: BR-001, BR-004, FR-002, FR-003, FR-004, RULE-005\
Acceptance Criteria: AC-003, AC-004, AC-005, AC-016, AC-017\
Priority: Must\
Level: Acceptance\
Automation Targets: `appview/internal/api/profile_customisation_test.go`, `appview/internal/routes/routes_test.go`

```gherkin
Feature: Replacing the current member's customisation
  Scenario: A current member saves a supported complete combination
    Given Alice has a valid authenticated Craftsky session and bound device
    When Alice sends PUT /v1/profiles/me/customisation with all three supported keys
    Then the AppView atomically creates or replaces Alice's one DID-scoped record
    And the response is 200 with the complete authoritative effective customisation object
    And an identical retry succeeds without duplicate state
    And later reads return the saved combination after sign-out, another-device sign-in, and AppView restart
    And no PDS, Tap, lexicon, profile blob, or banner blob operation occurs

  Scenario Outline: An unauthorized mutation changes no state
    Given the request has <invalid context>
    When PUT /v1/profiles/me/customisation is attempted
    Then the existing authentication or current-membership policy returns <result>
    And no member customisation record changes
    And no route or request field permits a target DID other than "me"

    Examples:
      | invalid context                    | result                      |
      | no valid session                   | existing unauthenticated error |
      | no valid device binding            | existing device-bound error |
      | removed or non-current membership  | existing membership error   |
      | another member DID in an extra field | validation error          |
```

### AT-004: Preserve Exact Inside-Border Geometry In Every Image State

Requirement IDs: BR-002, FR-007, FR-008, RULE-002\
Acceptance Criteria: AC-008, AC-009, AC-010\
Priority: Must\
Level: Acceptance\
Automation Target: `app/test/profile/widgets/profile_avatar_test.dart`

```gherkin
Feature: Avatar inside-border geometry
  Scenario Outline: The selected thickness maps to the confirmed width
    Given an avatar has external size <size> px
    And its selected border is <level>
    When the avatar image, loading fallback, missing-image fallback, or error fallback renders
    Then exactly one circular inside stroke has width <width> px
    And the image or fallback remains circularly clipped inside the stroke
    And the widget's external dimensions do not change
    And the existing hard-offset ink shadow does not change
    And no borderless state, second ring, or decorative frame is drawn

    Examples:
      | size | level  | width |
      | 36   | thin   | 1.5   |
      | 36   | medium | 2.5   |
      | 36   | thick  | 4     |
      | 48   | thin   | 2     |
      | 48   | medium | 3.5   |
      | 48   | thick  | 5     |
      | 96   | thin   | 3     |
      | 96   | medium | 5     |
      | 96   | thick  | 8     |
```

### AT-005: Scope The Colour And Local Texture To The Confirmed Profile Regions

Requirement IDs: BR-003, FR-009, FR-010, FR-011, RULE-001, RULE-003\
Acceptance Criteria: AC-011, AC-012, AC-019\
Priority: Must\
Level: Acceptance\
Automation Targets: `app/test/profile/profile_page_test.dart`, `app/test/profile/widgets/profile_card_test.dart`

```gherkin
Feature: Compact and full profile appearance boundaries
  Scenario Outline: A bundled texture is applied only to both profile header regions
    Given Alice selected <background> with a supported colour bundle
    When Alice's profile opens in compact and full presentations
    Then the <background> transparent local mask tiles over the selected base colour in both presentations
    And both presentations use its committed audited tint and opacity
    And each mask is clipped to its profile header region
    And neither presentation makes a remote network request for the mask

    Examples:
      | background   |
      | bayerdark    |
      | cubedark     |
      | dotcrossdark |
      | scallopdark  |
      | skewdark     |
      | x2           |

  Scenario: No background texture is selected
    Given Alice's profileBackground is "none"
    When the compact and full profile headers render
    Then both use Alice's selected colour without a texture layer

  Scenario: Compact profile custom colour scope
    Given Alice uses one supported colour bundle
    When her compact profile renders
    Then the entire compact view, including buttons and links, uses Alice's bundle
    And app-wide navigation and unrelated dialogs retain the normal Craftsky theme

  Scenario: Full profile custom colour scope
    Given Alice uses one supported colour bundle
    When her full profile renders
    Then every element above the tab bar uses Alice's bundle
    And the tab bar and every element below it use the normal Craftsky theme
    And another profile rendered later uses that profile's own values without theme leakage
```

### AT-006: Edit, Save, Discard, And Retry A Draft

Requirement IDs: BR-004, FR-005, FR-006\
Acceptance Criteria: AC-006, AC-007\
Priority: Must\
Level: Acceptance\
Automation Targets: `app/test/settings/profile_customisation_page_test.dart`, `app/test/router/settings_routes_test.dart`

```gherkin
Feature: Profile customisation settings lifecycle
  Scenario: Open the active member's confirmed customisation
    Given Alice is signed in on /profile/settings
    When Alice activates "Customisation"
    Then /profile/settings/customisation opens
    And the fixed controls show Alice's confirmed colour, border, and background
    And a representative live preview reflects those choices
    And Save is not pending

  Scenario: Save a changed draft successfully
    Given Alice changes one or more fixed choices
    When Alice activates Save
    Then one full-replacement request is sent
    And duplicate Save activation is disabled while pending
    When the authoritative response succeeds
    Then the page remains open
    And the response values become the confirmed values
    And affected public and cached surfaces update without an app restart
    And the draft is no longer dirty
    And exact themed feedback "Profile customisation saved" is shown

  Scenario: Preserve a changed draft after failure
    Given Alice's preview differs from her last confirmed values
    When Save fails
    Then the confirmed public and cached appearance remains unchanged
    And Alice's draft and preview remain available for retry
    And Save is re-enabled
    And exact feedback "Couldn't save your profile customisation." is shown

  Scenario Outline: Back navigation respects draft state
    Given the draft is <state>
    When Alice activates Back
    Then <outcome>

    Examples:
      | state                                      | outcome                                      |
      | equal to confirmed values                  | Settings opens directly                      |
      | dirty                                      | branded discard confirmation opens first     |
      | reverted manually to confirmed values      | Settings opens directly                      |
      | confirmed by a successful save             | Settings opens directly                      |
```

### AT-007: Fence In-Flight Work To The Initiating Account

Requirement IDs: BR-004, FR-004, FR-006, FR-010\
Acceptance Criteria: AC-014\
Priority: Must\
Level: Acceptance\
Automation Targets: `app/test/profile/providers/profile_customisation_provider_test.dart`, `app/test/router/app_shell_account_switcher_test.dart`

```gherkin
Feature: Account-scoped customisation state
  Scenario: The active account changes during a save
    Given Alice and Bob have different confirmed customisations
    And Alice starts a customisation save
    When the active account changes to Bob before Alice's request settles
    Then Bob's page, caches, theme, and feedback are not changed by Alice's completion
    And Alice's completion may update only Alice's account-scoped record and caches
    And selecting either account later restores that account's own effective values

  Scenario: The active account changes during initial loading
    Given Alice's customisation page is loading
    When the active account changes to Bob before Alice's load settles
    Then Alice's result cannot populate Bob's controls or preview
    And Bob loads or displays Bob's own confirmed values
```

### AT-008: Preserve Moderation And Unavailable-Identity Policy

Requirement IDs: FR-001, FR-013, RULE-004, NFR-004\
Acceptance Criteria: AC-015\
Priority: Must\
Level: Acceptance\
Automation Targets: `appview/internal/api/profile_response_test.go`, `appview/internal/api/notifications_test.go`, `appview/internal/api/relationship_list_test.go`

```gherkin
Feature: Customisation under existing visibility policy
  Scenario Outline: A policy-limited shell still exposes an avatar
    Given Alice is <policy state> relative to viewer Bob
    And the existing response policy exposes Alice's avatar shell
    When Bob receives that response
    Then the shell contains enough effective customisation to render the avatar border
    And customisation does not restore any stripped identity field or content

    Examples:
      | policy state |
      | muted        |
      | blocked      |
      | blocking     |
      | unavailable  |
      | warned       |
      | hidden       |

  Scenario: Existing policy removes the identity object
    Given the existing response policy removes Alice's identity and avatar object
    When Bob receives the response
    Then no customisation object is synthesized or disclosed

  Scenario: A system notification has no actor
    Given a system or unavailable-actor notification is actor-free today
    When the notification response is built
    Then it remains actor-free
    And no avatar or customisation object is synthesized
```

### AT-009: Apply Defaults And Per-Field Fallback Across Rolling Versions

Requirement IDs: FR-012, NFR-004\
Acceptance Criteria: AC-013, AC-017\
Priority: Must\
Level: Acceptance\
Automation Targets: `app/test/profile/models/profile_test.dart`, `app/test/profile/models/profile_customisation_test.dart`, `appview/internal/api/profile_response_test.go`

```gherkin
Feature: Effective customisation defaults and compatibility
  Scenario: A current member has no saved row
    When any supported AppView identity response is built
    Then it includes colour "cobalt", profileBorder "medium", and profileBackground "none"
    And no row must be eagerly created

  Scenario: A new Flutter client receives an older response
    Given the identity response omits "customisation"
    When Flutter decodes and renders it
    Then Flutter uses cobalt, medium, and none
    And the identity remains usable

  Scenario Outline: Only an unknown field falls back
    Given <field> has an unknown or retired key
    And the other two fields are valid
    When the effective customisation is derived
    Then only <field> uses its default
    And the other valid values are preserved
    And no remote resource is requested

    Examples:
      | field             |
      | colour            |
      | profileBorder     |
      | profileBackground |

  Scenario: An older client receives an enriched identity response
    When it decodes the response while ignoring unknown additive fields
    Then every pre-existing required field and behavior remains unchanged
```

### AT-010: Operate The Settings Experience Accessibly

Requirement IDs: FR-005, NFR-003\
Acceptance Criteria: AC-018\
Priority: Must\
Level: Acceptance\
Automation Target: `app/test/settings/profile_customisation_page_test.dart`

```gherkin
Feature: Accessible fixed customisation choices
  Scenario Outline: A member selects customisation without relying on colour alone
    Given the page is rendered with <input mode>, <theme>, and <text scale>
    When the member moves through and selects every colour, border, and background option
    Then every option exposes a readable label, group, value, and selected state
    And border levels and texture choices have non-colour labels or indicators
    And focus order reaches controls, preview, Save, Back, and discard confirmation coherently
    And supported text does not clip or hide actions
    And foreground, selected, hover, pressed, and soft-container pairs meet the project's audited contrast thresholds

    Examples:
      | input mode    | theme | text scale          |
      | touch         | light | default             |
      | keyboard      | light | maximum supported   |
      | screen reader | dark  | maximum supported   |
```

## 4. Unit Test Cases

| Test ID | Requirement / AC IDs | Component | Test design | Expected result | Proposed target |
|---|---|---|---|---|---|
| UT-001 | FR-011, FR-012, RULE-001, RULE-002, RULE-003; AC-005, AC-011, AC-012, AC-013 | Shared server/client catalogue fixtures | Assert seven committed colour keys including Ink, exactly `thin`/`medium`/`thick`, and exactly `none` plus the six named texture keys; assert cobalt/default constants and uniqueness/stability of keys. | Catalogues are closed, deterministic, versioned, and server/client fixtures agree. | Go catalogue test and `app/test/profile/models/profile_customisation_test.dart` |
| UT-002 | FR-003, RULE-001, RULE-002, RULE-003; AC-005 | Go request decoder and validator | Table-test valid body plus missing fields, unknown fields, non-strings, arbitrary colour values, unsupported keys, nested resource data, URLs, malformed JSON, duplicate JSON keys if the shared strict decoder rejects them, and oversized bodies. | Only exact complete supported objects pass; errors use standard codes and field-specific `422 validation_failed` details where required. | `appview/internal/api/profile_customisation_request_test.go` |
| UT-003 | FR-012, NFR-004; AC-013, AC-017 | Flutter model/JSON decoding | Decode complete new, absent, null/malformed if tolerated by existing model policy, and one-unknown-field fixtures. Encode/decode repository response fixtures as applicable. | Absence uses all defaults; each unknown key falls back independently without discarding valid siblings or crashing. | `app/test/profile/models/profile_customisation_test.dart`, extend `profile_test.dart` and affected response-model tests |
| UT-004 | FR-007, RULE-002; AC-009 | Border width policy | Table-test all nine size/level combinations and reject/no-map unsupported sizes or levels according to implementation policy. | Widths are exactly 1.5/2.5/4, 2/3.5/5, and 3/5/8 px. | `app/test/profile/widgets/profile_avatar_test.dart` or pure style-policy test |
| UT-005 | FR-007; AC-008, AC-009, AC-010 | Shared avatar renderer | Inspect decoration/painter and layout for image success, loading, missing URL, and error at every supported size/level. | One inside stroke uses selected base colour; clipping, semantics, outer size, and current shadow remain unchanged. | `app/test/profile/widgets/profile_avatar_test.dart` |
| UT-006 | FR-009, FR-010, FR-011, RULE-001; AC-012 | Profile theme resolver and boundary widgets | Resolve every colour key to base/readable foreground/hover/pressed/soft container, test contrast calculations, and assert compact/full subtree theme ownership. | Stable audited bundles are used; compact scope is complete; full scope stops before tab bar; themes do not leak. | new profile theme-policy test plus `profile_page_test.dart`/`profile_card_test.dart` |
| UT-007 | FR-009, FR-011, RULE-003; AC-011, AC-019 | Background catalogue and renderer | Assert exact key/name/asset mapping, local bundled asset resolution, tile/clipping behavior, `none` behavior, transparent layering, and no network-image/resource loader. | Six textures and `none` render only from checked-in assets within header bounds. | new background-policy test plus profile widget tests |
| UT-008 | FR-005; AC-006, AC-007 | Settings draft state | Exercise confirmed load, live edits, revert-to-confirmed, pending state, successful reconciliation, failed save, and dirty-state comparison across all three fields. | Save/prompt state follows value equality; failed draft survives; success adopts authoritative response and becomes clean. | `app/test/settings/profile_customisation_page_test.dart` or provider test |
| UT-009 | FR-006; AC-007, AC-014 | Save provider/controller | Test one request while pending, authoritative-response reconciliation, targeted invalidation/update, failure retention, exact success feedback, approved failure feedback, and disposal. | No optimistic public mutation or duplicate request; correct account-scoped outcome and feedback. | `app/test/profile/providers/profile_customisation_provider_test.dart` |
| UT-010 | FR-006, FR-010; AC-014 | Active-account lease/fencing | Complete old-account load/save after switch, logout, or lease invalidation. | Stale continuation cannot mutate active UI, provider state, caches, theme, or feedback. | provider test using existing account-lease harness |
| UT-011 | NFR-003; AC-018 | Settings and preview semantics | Inspect semantics tree, labels, roles, selection state, traversal order, actions, focus restoration, and supported text-scale layouts in both themes. | Every choice and action is perceivable and operable without colour alone; no clipping at supported scales. | `app/test/settings/profile_customisation_page_test.dart` |

## 5. Integration Test Cases

| Test ID | Requirement / AC IDs | Boundary | Test design | Expected result | Proposed target |
|---|---|---|---|---|---|
| IT-001 | BR-001, FR-004, NFR-002; AC-003, AC-016, AC-020 | Postgres migration | Run migration up/down/up against real Postgres; inspect table, canonical-DID key/FK, constraints/indexes, timestamps as designed, cascade cleanup, and preservation of existing tables/data. Determine migration number from repository head during implementation. | Reversible schema; existing data survives; one row per current member; deletion cleanup works. | `appview/internal/db/profile_customisation_migration_test.go`, `migration_files_test.go` |
| IT-002 | BR-004, BR-005, FR-002, FR-003, FR-004, RULE-004; AC-003, AC-004, AC-014 | Store + authenticated handler | Test default read, create, complete replacement, identical retry, account isolation, removed membership, persistence across fresh store/session, and two-device access. | One atomic DID-scoped effective record is authoritative and owner-writable only. | `appview/internal/api/profile_customisation_store_test.go`, `profile_customisation_test.go` |
| IT-003 | FR-002, FR-003; AC-003, AC-005, AC-017 | HTTP route/decoder/store | Send valid, malformed, partial, extra-key, unsupported, unauthorized, wrong-method, oversized, and concurrent complete requests through the route. | Contract uses camelCase/full replacement/standard envelope; invalid writes are no-ops; identical retry succeeds; last committed concurrent write is complete, never field-mixed. | `appview/internal/routes/routes_test.go`, `appview/internal/api/profile_customisation_test.go` |
| IT-004 | FR-001, FR-012; AC-001, AC-002, AC-013, AC-017 | Public response builders | Seed no-row/default, saved, and retired-key records; exercise profile, post/reply/quote, notifications, search, and relationship/account endpoints. | Every otherwise-visible identity has one complete effective nested object; per-field server fallback is consistent and additive. | extend existing response tests named in AT-001 |
| IT-005 | NFR-001; AC-002, AC-019 | Paginated database reads | Seed pages with many identities and repeated authors; assert statement count remains bounded as page size grows and inspect the relevant `EXPLAIN` plan for indexed/set-based hydration. | No per-avatar DB/HTTP lookup; query plan uses intended indexes and one member's repeated appearances agree. | `appview/internal/api/profile_customisation_query_plan_test.go` plus affected store-query tests |
| IT-006 | FR-001, FR-013, NFR-004, RULE-004; AC-015 | Moderation and availability response policy | Exercise muted, blocked, blocking, unavailable, warned, hidden, stripped-identity, and actor-free system-notification fixtures. | Customisation accompanies only retained avatars and never bypasses or expands visibility. | existing profile/notification/relationship response tests |
| IT-007 | BR-005, FR-004, RULE-005; AC-003, AC-016 | AppView/PDS/Tap/blob boundary | Run save/read/session/device/delete lifecycle with recording PDS client/factory, Tap publisher/indexer hooks, and blob/profile writers. | Only AppView table/caches change; all forbidden outbound/write collaborators receive zero calls; no indexing wait exists. | `appview/internal/api/profile_customisation_boundary_test.go` or existing observability/PDS-spy harness |
| IT-008 | FR-002, FR-012, NFR-004; AC-013, AC-017 | Old/new wire compatibility | Test old response fixtures against new Flutter models, enriched fixtures against a retained old-decoder compatibility fixture where practical, and existing endpoint statuses/error envelopes alongside the new route. | Rolling client/server versions remain usable; only the new authenticated route and additive nested field change the wire surface. | Go route/response tests and Flutter API/model fixture tests |
| IT-009 | BR-005, NFR-005; AC-016, AC-019 | Logs and telemetry | Capture success, validation, auth, database, and retired-key diagnostics. Inspect metric label sets and structured logs. | Bounded operation/result/error-class signals only; no DID, preference key, or asset filename appears as a metric label; logs remain actionable and follow existing privacy patterns. | `appview/internal/api/profile_customisation_observability_test.go` |
| IT-010 | FR-011, RULE-003; AC-011, AC-019 | Flutter asset bundle/render path | Load every approved background from the test asset bundle while recording HTTP/image-provider calls and checking provenance/attribution metadata. | Every texture resolves locally; no URL/custom CSS/arbitrary resource is used; provenance records cover all six sources. | Flutter background catalogue/asset test |

## 6. Regression Test Cases

| Test ID | Requirement / AC IDs | Existing behavior protected | Regression assertion | Existing/proposed target |
|---|---|---|---|---|
| REG-001 | NFR-002, NFR-006; AC-020 | Existing migration chain and data | Full migration suite still runs up/down as applicable and preserves profiles, posts, notifications, search, relationships, moderation, accounts, and prior feature tables. | `appview/internal/db/*_migration_test.go` |
| REG-002 | NFR-004; AC-017 | Existing API fields/status/error envelopes | Snapshot/contract assertions change only by the additive nested object and new route; existing required fields and errors remain intact. | current AppView route/response tests |
| REG-003 | FR-005; AC-006 | Settings hierarchy and Back behavior | Existing Settings entries/routes still work; new route is nested under `/profile/settings`; clean Back returns to Settings and app-shell selection remains correct. | `app/test/settings/settings_page_test.dart`, `app/test/router/settings_routes_test.dart`, navigation tests |
| REG-004 | BR-002, FR-007, FR-008; AC-008, AC-009, AC-010 | Avatar dimensions, shadows, fallbacks, semantics, and surfaces | Extend `profile_avatar_test.dart`, `post_card_test.dart`, `notifications_page_test.dart`, `profile_card_test.dart`, search/profile result tests, relationship list tests, and account-switcher/navigation tests to assert the shared customisation input without layout drift. | existing Flutter surface tests |
| REG-005 | BR-003, FR-009, FR-010; AC-011, AC-012 | Compact/full profile transitions, tabs, and post content | Header background/theme updates do not recolour full-profile tab bar or tab content, alter scrolling/layout, or leak across profiles; compact mode remains entirely scoped. | `profile_page_test.dart`, `profile_card_test.dart`, profile tab tests |
| REG-006 | FR-013, NFR-004; AC-015 | Blocked/moderated/unavailable shells | Existing stripped fields, warnings, blocks, and actor-free notifications remain exactly policy-equivalent after identity enrichment. | existing Go response and Flutter presentation tests |
| REG-007 | FR-005, NFR-003; AC-006, AC-018 | Router focus and discard patterns | Direct/deep-link navigation, Back, keyboard traversal, dialogs, and route restoration follow existing settings conventions. | `settings_routes_test.dart`, new settings widget test |
| REG-008 | FR-006, FR-010; AC-007, AC-014 | Multi-account cache and presentation isolation | Switching accounts during/after load or save does not reuse another account's confirmed state, preview, feedback, or header theme. | `app_shell_account_switcher_test.dart`, provider tests |
| REG-009 | RULE-005; AC-016 | Existing PDS-backed profile editing and lexicons | Existing profile edit mutation still owns avatar/banner/display-name/bio writes; customisation save does not change it; no lexicon/generated lexicon file changes are required. | `profile_request_test.go`, Flutter profile API/edit tests, scoped git-diff review |

## 7. Test Data And Fixtures

| Fixture ID | Data | Purpose |
|---|---|---|
| TD-001 | Alice with no customisation row; Bob with a complete non-default row; Carol removed; Dana blocked/unavailable | Defaults, persistence, account isolation, lifecycle, and policy cases |
| TD-002 | Canonical catalogue fixture shared or equivalently asserted across Go and Dart: seven colour keys, three border keys, `none` plus six background keys | Prevent server/client catalogue drift and arbitrary values |
| TD-003 | Valid complete request and invalid corpus: omitted/extra fields, non-string values, arbitrary hex/RGB, bad keys, URLs/resource data, malformed/oversized JSON | Strict request and no-resource-loading validation |
| TD-004 | Old identity JSON without `customisation`; new default/customized JSON; each single retired/unknown field; existing blocked-shell fixtures | Compatibility and independent fallback |
| TD-005 | Avatar matrix of 36/48/96 px × thin/medium/thick × success/loading/null/error × light/dark | Exact geometry, clipping, fallback, semantics, and shadow coverage |
| TD-006 | Surface matrix covering feed, thread root/reply, quote, notification, compact/full profile, search, relationship/account list, summary, preview, navigation/switcher | Global shared-avatar propagation |
| TD-007 | Local assets and display names: `bayerdark`/Dither, `cubedark`/Grid, `dotcrossdark`/Cross stitch, `scallopdark`/Scallops, `skewdark`/Diagonal weave, `x2`/Crosshatch, plus provenance/attribution metadata | Exact background catalogue and responsible local use |
| TD-008 | Two valid complete save payloads submitted concurrently from separate device contexts | Atomic last-committed replacement without mixed fields |
| TD-009 | Compact/full profiles at supported viewport widths, text scales, light/dark themes, keyboard/touch/semantics modes | Scope, responsive behavior, and accessibility |
| TD-010 | Paginated post/notification/search/relationship datasets with repeated authors and increasing page sizes | Bounded query-count and indexed-plan assertions |
| TD-011 | Recording/spying PDS client, Tap/indexer hook, profile/blob writer, HTTP image/resource loader, metrics sink, and log sink | Negative boundaries and observability |
| TD-012 | Final seven audited colour bundles and their approved per-colour texture tint/opacity pairs | Exact contrast, theme-state, and visual assertions after GAP-001/GAP-002 close |

## 8. Manual Verification

| Test ID | Requirement / AC IDs | Check | Expected result | Why manual remains |
|---|---|---|---|---|
| MAN-001 | FR-005, NFR-003; AC-006, AC-007, AC-018 | Use the completed settings flow with VoiceOver/TalkBack and hardware keyboard on representative iOS/Android targets in light/dark themes and maximum supported text scale. | Spoken labels, grouping, selection changes, preview, focus order, Save state, Back/discard dialog, and feedback are clear and coherent. | Widget semantics tests cannot fully reproduce platform speech and focus presentation. |
| MAN-002 | BR-003, FR-009, FR-011, RULE-003; AC-011, AC-012, AC-019 | Review all seven approved colour bundles with all six textures plus `none` in compact/full headers at representative phone/tablet widths. | Texture density/tint is legible but subordinate, header clipping is exact, buttons/links are readable, and the full-profile boundary ends before the tab bar. | Final aesthetic balance requires human review after exact colour/tint values are approved; bounds and contrast remain automated. |
| MAN-003 | NFR-003; AC-018 | Review 36 px thick-border fallbacks and all palette colours under common colour-vision simulations. | Initial/image remains legible, thickness labels distinguish options, and no choice depends on hue alone. | Simulation and perceptual review complement automated semantics/contrast checks. |

## 9. Known Test Gaps

| Gap ID | Open requirement detail | Tests affected | Impact and closure condition |
|---|---|---|---|
| GAP-001 | Closed 2026-08-10: `cobalt`, `orchid`, `rose`, `amber`, `lime`, `teal`, and `ink` keys plus audited bundle constants are recorded in `01-requirements.md` Q11. | UT-001, UT-006, AT-005, AT-010, MAN-002, TD-002, TD-012 | Closed before each palette-sensitive test was written. |
| GAP-002 | Closed 2026-08-10: each colour bundle uses its foreground as texture tint at 18% opacity. | UT-007, AT-005, MAN-002, TD-012 | Closed before exact painter/style assertions. |
| GAP-003 | Closed 2026-08-10: exact failure feedback is `Couldn't save your profile customisation.` | AT-006, UT-009 | Closed before the exact copy assertion. |

There are no other known coverage gaps. Manual checks supplement rather than replace automated behavioral assertions.

## 10. Proposed Test Targets And Commands

### AppView

Create focused tests where no suitable seam exists:

- `appview/internal/db/profile_customisation_migration_test.go`
- `appview/internal/api/profile_customisation_request_test.go`
- `appview/internal/api/profile_customisation_store_test.go`
- `appview/internal/api/profile_customisation_test.go`
- `appview/internal/api/profile_customisation_query_plan_test.go`
- `appview/internal/api/profile_customisation_observability_test.go`

Extend existing response/route suites rather than duplicating endpoint fixtures:

- `appview/internal/api/profile_response_test.go`
- `appview/internal/api/post_response_test.go`
- `appview/internal/api/notifications_test.go`
- `appview/internal/api/search_response_test.go`
- `appview/internal/api/relationship_list_test.go`
- `appview/internal/api/profile_relationship_test.go`
- `appview/internal/routes/routes_test.go`

Run the project-standard Go suite from repository root with `just test` while the required Compose Postgres is available. Use narrow `go test` packages during red-green loops, then the full command before review.

### Flutter

Likely focused additions:

- `app/test/profile/models/profile_customisation_test.dart`
- `app/test/profile/data/profile_customisation_api_client_test.dart`
- `app/test/profile/providers/profile_customisation_provider_test.dart`
- `app/test/settings/profile_customisation_page_test.dart`

Extend shared and affected-surface suites:

- `app/test/profile/widgets/profile_avatar_test.dart`
- `app/test/profile/profile_page_test.dart`
- `app/test/profile/widgets/profile_card_test.dart`
- `app/test/feed/widgets/post_card_test.dart`
- `app/test/notifications/notifications_page_test.dart`
- profile-search provider/page tests
- `app/test/settings/relationship_list_page_test.dart` and account-list equivalents
- `app/test/router/settings_routes_test.dart`
- `app/test/router/app_shell_account_switcher_test.dart`

Use narrow `flutter test <path>` commands from `app/` during red-green loops, then the repository's full Flutter test recipe and static analysis recipe before review.

## 11. TDD Handoff

Recommended first failing tests, in order:

1. `UT-001`: define the effective customisation value object, defaults, and closed catalogue contract on both Go and Dart sides.
2. `UT-002`: lock the strict full-replacement request validation before adding persistence or a route.
3. `IT-001`: add the reversible AppView migration and lifecycle constraints.
4. `IT-002`: implement default reads and atomic DID-scoped replacement at the store boundary.
5. `IT-003`: expose the authenticated/device-bound mutation with the established API contract.
6. `IT-004` and `IT-005`: enrich all public identity shapes with set-based hydration before Flutter depends on the field.
7. `UT-003`, `UT-009`, and `AT-006`: add tolerant Flutter decoding and the confirmed/draft save lifecycle.
8. `UT-004`/`UT-005` and REG-004: replace the avatar ink stroke once at the shared renderer and drive every surface through it.
9. `UT-006`/`UT-007` and AT-005: add the bounded profile theme/background presentation after GAP-001 and GAP-002 are resolved.

At every step, keep the smallest relevant test red, implement only enough behavior to make it green, refactor behind the verified contract, then run the affected regression group. Before implementation review, run the full AppView and Flutter suites and record only commands that actually completed.

## 12. Test Design Status

Status: Complete with three explicit gated implementation inputs.

All Must requirements and AC-001 through AC-020 have planned coverage. GAP-001 through GAP-003 closed before their dependent exact assertions.
