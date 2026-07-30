# Acceptance Test Specification: Post And Content Languages

## 1. Test Strategy

This High-risk change requires layered, predominantly automated coverage across Flutter, the AppView API, Postgres-backed list queries, and firehose indexing. The central risk is not the language selector itself; it is keeping one visibility contract consistent across eight independently paginated browse/discovery endpoints while deliberately bypassing that contract for direct and contextual access.

Pure language-tag validation, device-locale normalisation, exact-match eligibility, repost and quote rules, composer defaults, catalogue lookup, JSON mapping, account-keyed state, and log redaction belong in unit tests. Flutter widget and provider tests cover the Languages page, all composer variants, authoritative preference loading, account switching, successful update invalidation, failed update retention, and stale in-flight result rejection. Go handler and Postgres integration tests cover authenticated private preferences, atomic create-if-absent initialisation, `POST /v1/posts`, indexing, response shaping, filtering before pagination, and the exception surfaces.

The AppView test suite must define one reusable language-visibility corpus and run it against:

- `GET /v1/feed/timeline`
- `GET /v1/projects`
- `GET /v1/search/posts`
- `GET /v1/search/projects`
- `GET /v1/search/hashtags/{tag}/posts`
- `GET /v1/profiles/{handleOrDid}/posts`
- `GET /v1/profiles/{handleOrDid}/projects`
- `GET /v1/profiles/{handleOrDid}/comments`

The corpus contains matching, mismatched, multilingual, untagged, viewer-authored, reposted, and quoted posts. Each adapter must prove the same language eligibility without weakening its existing moderation, membership, reply, import, ordering, or cursor rules. Direct post/thread reads, notifications, and Saved Posts require independent regression tests proving that the shared browse/discovery filter cannot be applied accidentally.

Automated tests should use the existing Flutter `flutter_test` and Riverpod test patterns, mocked API repositories, Go `httptest`, `internal/testdb.WithSchema`, and focused migration tests. They must not contact a live PDS or depend on a device's real locale. Flutter acceptance tests are composed client flows using fakes; they are not described as live AppView/PDS/Tap/Postgres end-to-end evidence. Go/Postgres integration tests and any explicitly run compose-stack smoke test are reported as separate verification layers. Physical-device checks are limited to screen-reader, keyboard, and large-text usability; deterministic accessibility semantics remain automated.

Risk level: **High** (carried forward). The user approved progression from requirements to test design on 2026-07-29. No blocking test-design gap is identified. Document review and explicit approval remain required before implementation.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-006, AC-007, AC-009 | AT-004, AT-005, AT-015, UT-006, UT-016, IT-005, IT-006, IT-027 | Acceptance / Unit / Integration | Yes |
| BR-002 | AC-001, AC-010, AC-011, AC-012, AC-024 | AT-006, AT-007, AT-008, AT-015, UT-003, IT-008–IT-017 | Acceptance / Unit / Integration | Yes |
| BR-003 | AC-001, AC-002, AC-003, AC-004, AC-005 | AT-001, AT-003, AT-015, IT-001, IT-018 | Acceptance / Integration | Yes |
| BR-004 | AC-003 | AT-001, REG-010 | Acceptance / Regression | Yes |
| FR-001 | AC-002 | AT-001, IT-001 | Acceptance / Integration | Yes |
| FR-002 | AC-003 | AT-001, REG-010 | Acceptance / Regression | Yes |
| FR-003 | AC-004, AC-006, AC-016, AC-026 | AT-003, AT-004, UT-006, UT-007, UT-017, IT-002, IT-018, IT-027 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-005, AC-011, AC-016 | AT-003, AT-007, UT-007, IT-002, IT-008–IT-015, IT-018 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-006, AC-007 | AT-004, UT-006, UT-016, IT-027 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-008, AC-009 | AT-005, UT-001, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-009 | AT-005, UT-016, IT-027 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-009, AC-010, AC-011 | AT-005, AT-006, AT-007, IT-006 | Acceptance / Integration | Yes |
| FR-009 | AC-010, AC-011, AC-012, AC-024 | AT-006, AT-007, AT-008, IT-008–IT-017 | Acceptance / Integration | Yes |
| FR-010 | AC-015 | AT-005, UT-010, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-014, AC-024 | AT-010, UT-012, UT-013, IT-018, IT-019 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-005, AC-007, AC-017, AC-027 | AT-004, AT-014, UT-008, UT-009, MAN-001 | Acceptance / Unit / Manual | Partial |
| FR-013 | AC-016, AC-019, AC-020, AC-022 | AT-002, AT-003, UT-011, IT-002, IT-003, IT-004, IT-023, IT-026 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-019 | AT-002, UT-002, IT-019 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-020 | AT-002, IT-003, IT-023 | Acceptance / Integration | Yes |
| FR-016 | AC-022 | AT-003, UT-011, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-023 | AT-011, UT-012, IT-019 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-008, AC-022, AC-027, AC-028 | AT-005, AT-013, UT-001, UT-004, UT-008, UT-009, UT-011, IT-004, IT-006 | Acceptance / Unit / Integration | Yes |
| NFR-002 | AC-012, AC-024 | AT-006, AT-008, IT-008–IT-017, REG-002 | Acceptance / Integration / Regression | Yes |
| NFR-003 | AC-016, AC-018 | AT-003, UT-014, IT-002, IT-024, REG-010 | Acceptance / Unit / Integration / Regression | Yes |
| NFR-004 | AC-017 | AT-014, MAN-001, MAN-002 | Acceptance / Manual | Partial |
| NFR-005 | AC-016, AC-021, AC-022, AC-023 | AT-003, AT-011, UT-012, IT-002, IT-019, IT-026 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-007, AC-008 | AT-004, AT-005, UT-001, UT-006, IT-005, IT-027 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-010, AC-024 | AT-006, UT-003, IT-008–IT-015 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-011 | AT-007, UT-003, IT-008–IT-015, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-004 | AC-013 | AT-009, UT-005, IT-008, REG-008 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-005 | AC-004, AC-005 | AT-003, UT-007, IT-018 | Acceptance / Unit / Integration | Yes |
| RULE-006 | AC-006 | AT-004, UT-017, IT-027 | Acceptance / Unit / Integration | Yes |
| RULE-007 | AC-025 | AT-012, IT-020, IT-021, IT-022, REG-004, REG-005, REG-006 | Acceptance / Integration / Regression | Yes |
| RULE-008 | AC-024, AC-025 | AT-006, AT-012, IT-008–IT-015, REG-007 | Acceptance / Integration / Regression | Yes |
| RULE-009 | AC-026 | AT-004, UT-006, UT-017, IT-027 | Acceptance / Unit / Integration | Yes |
| RULE-010 | AC-028 | AT-013, UT-004, IT-006 | Acceptance / Unit / Integration | Yes |

Every Must requirement and AC-001 through AC-028 has an automated verification path. “Partial” means deterministic catalogue and accessibility behavior is automated, while assistive-technology and extreme text-scale usability also receive a physical-device check. Both Should requirements are covered because the language catalogue and accessibility are material to the requested UI.

## 3. Acceptance Scenarios

### AT-001: Languages settings expose three distinct concepts
Requirement IDs: BR-003, BR-004, FR-001, FR-002
Acceptance Criteria: AC-002, AC-003
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/languages_page_test.dart`, `app/test/router/settings_routes_test.dart`

```gherkin
Feature: Language settings
  Scenario: English-only CraftSky presents the complete language model
    Given a signed-in user is on Settings
    When they open Languages
    Then the page contains App language, Primary language, and Content languages sections
    And App language shows English as the only available UI language
    And the page visibly says that more App languages are coming
    And the posting and reading selectors remain separately operable
```

### AT-002: A new account is initialised once from ordered device locales
Requirement IDs: FR-013, FR-014, FR-015
Acceptance Criteria: AC-019, AC-020
Priority: Must
Level: Acceptance
Automation Target: `app/test/languages/providers/language_preferences_bootstrap_test.dart`, `appview/internal/api/language_preferences_test.go`

```gherkin
Scenario: First-use locale proposal becomes authoritative once
  Given a new account has no stored language preferences
  And the device locale order is fr-CA, en-GB, fr-FR, then an unsupported locale
  When GET /v1/languages/preferences returns language_preferences_not_found
  And Flutter proposes first-use preferences to the initialize route
  Then Primary is fr
  And Content languages are fr then en
  And AppView atomically stores or returns one authoritative preference row
  And a later initialisation attempt returns that row without overwriting it
```

### AT-003: Private account preferences remain independent and portable
Requirement IDs: FR-003, FR-004, FR-013, FR-016, NFR-003, NFR-005, RULE-005
Acceptance Criteria: AC-004, AC-005, AC-016, AC-022
Priority: Must
Level: Acceptance
Automation Target: `app/test/languages/languages_page_test.dart`, `appview/internal/api/language_preferences_test.go`

```gherkin
Scenario: Primary and Content updates affect only the authenticated account
  Given Alice has English Primary and English Content languages
  When Alice changes Primary to Spanish and Content languages to Spanish and French
  Then both values persist privately for Alice in AppView
  And the same account loads them on another device
  And Alice's device-local App language is not uploaded
  And Bob's preferences are unchanged
  And any invalid replacement leaves Alice's complete stored preference unchanged
```

### AT-004: Every composer starts from Primary and permits one to three languages
Requirement IDs: BR-001, FR-003, FR-005, FR-012, RULE-001, RULE-006, RULE-009
Acceptance Criteria: AC-006, AC-007, AC-026, AC-027
Priority: Must
Level: Acceptance
Automation Target: `app/test/languages/post_language_selector_test.dart`, `app/test/feed/widgets/post_composer_languages_test.dart`, `app/test/projects/widgets/project_composer_languages_test.dart`

```gherkin
Scenario Outline: Shared composer language behavior
  Given Primary is English
  And the previous post used French and Welsh
  When the author opens a new <composer> composer
  Then only English is initially selected
  And a reply does not inherit the parent post languages
  And the full v1 base-language catalogue is searchable
  And the author may select one, two, or three distinct languages
  And a fourth selection and submission with none are prevented
  And changing Primary elsewhere does not replace this open composer's selection

  Examples:
    | composer |
    | general |
    | project |
    | reply |
    | quote |
```

### AT-005: Valid language tags survive create, index, and response paths
Requirement IDs: BR-001, FR-006, FR-007, FR-008, FR-010, NFR-001, RULE-001
Acceptance Criteria: AC-008, AC-009, AC-015
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/post_test.go`, `appview/internal/index/craftsky_post_test.go`, `app/test/feed/data/post_api_client_test.dart`

```gherkin
Scenario: A bilingual post keeps one canonical language list
  Given an author selects English and French
  When the app submits the post
  Then POST /v1/posts writes langs ["en", "fr"] to the PDS record
  And the synthetic create response exposes the same values without waiting for indexing
  And the firehose indexer materialises the same values
  And later post-shaped responses and Flutter models expose the same values
  And invalid, duplicate, or more than three request values receive the standard 400 validation envelope
```

### AT-006: Every browse and discovery surface uses the same active filter
Requirement IDs: BR-002, FR-009, NFR-002, RULE-002, RULE-008
Acceptance Criteria: AC-010, AC-024
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/language_visibility_test.go`

```gherkin
Scenario Outline: English filtering hides other people's mismatched and untagged posts
  Given the authenticated viewer selected English
  And the <surface> contains otherwise-eligible English, Welsh, English-plus-French, French-only, and untagged posts
  And viewer-authored French posts exist wherever that surface can contain them
  When the viewer requests the surface
  Then English and English-plus-French posts are included
  And other people's Welsh, French-only, and untagged posts are excluded
  And viewer-authored French posts are included wherever applicable
  And the surface's existing order is preserved

  Examples:
    | surface |
    | home timeline |
    | Projects browse |
    | post search |
    | project search |
    | hashtag post results |
    | another user's profile posts |
    | another user's profile projects |
    | another user's profile comments |
```

### AT-007: Empty Content languages show all otherwise-eligible posts
Requirement IDs: FR-004, FR-008, FR-009, RULE-003
Acceptance Criteria: AC-011
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/language_visibility_test.go`, `app/test/languages/languages_page_test.dart`

```gherkin
Scenario: Clearing the last Content language disables language filtering
  Given the viewer is warned that selecting no Content languages shows all
  When they clear the final selection and the AppView update succeeds
  Then every in-scope surface includes tagged and untagged otherwise-eligible posts
  And existing non-language eligibility rules still apply
  And each surface keeps its normal order
```

### AT-008: Language filtering occurs before cursor pagination
Requirement IDs: BR-002, FR-009, NFR-002
Acceptance Criteria: AC-012
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/language_visibility_pagination_test.go`

```gherkin
Scenario Outline: Mismatched rows between eligible rows do not create cursor gaps
  Given <surface> has interleaved matching and mismatched posts across more than one page
  When the viewer follows every returned cursor
  Then each non-terminal page contains up to the requested number of eligible posts
  And no eligible post is skipped or duplicated
  And mismatched posts never appear
  And existing chronological or ranked order is unchanged among eligible posts

  Examples:
    | surface |
    | timeline |
    | Projects browse |
    | post search |
    | project search |
    | hashtag results |
    | profile posts |
    | profile projects |
    | profile comments |
```

### AT-009: Reposts and quotes use their agreed language subjects
Requirement IDs: RULE-004
Acceptance Criteria: AC-013
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/language_visibility_test.go`

```gherkin
Scenario: English readers encounter French repost and quote subjects
  Given the viewer selected English
  And a followed actor reposted a French post
  And an English outer quote post embeds a French post
  When the viewer browses discovery results
  Then the repost is excluded because its subject is French
  And the English outer quote is included
  And its quoted French content is displayed as-is
```

### AT-010: A successful Content update replaces every affected cache generation
Requirement IDs: FR-011
Acceptance Criteria: AC-014
Priority: Must
Level: Acceptance
Automation Target: `app/test/languages/providers/content_language_invalidation_test.dart`

```gherkin
Scenario Outline: Preference update outcome controls visible state
  Given every filtered surface has cached English items and an active cursor
  When changing Content languages to French <outcome>
  Then <visibleState>
  And <cursorState>

  Examples:
    | outcome | visibleState | cursorState |
    | succeeds | incompatible English items are discarded and French results load | old cursors are not reused |
    | fails | English items and effective English preference remain | existing cursors remain valid |
```

### AT-011: Account activation waits for the correct authoritative preference
Requirement IDs: FR-017, NFR-005
Acceptance Criteria: AC-021, AC-023
Priority: Must
Level: Acceptance
Automation Target: `app/test/languages/providers/account_language_preferences_test.dart`

```gherkin
Scenario: A stale account request finishes after account switching
  Given Alice's preference or filtered-list request is in flight
  When Bob becomes the active account
  Then composers and filtered surfaces show an explicit loading or retry state for Bob
  And Alice's late preference and list results are discarded
  And Bob cannot submit or browse filtered results until Bob's authoritative preferences are available
```

### AT-012: Deliberate and contextual access always shows the post as-is
Requirement IDs: RULE-007, RULE-008
Acceptance Criteria: AC-025
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/language_visibility_exceptions_test.go`, `app/test/languages/language_visibility_exceptions_test.dart`

```gherkin
Scenario Outline: A mismatched or untagged post bypasses discovery filtering
  Given the viewer selected English
  And the target content is French or untagged when the context contains post content
  When the viewer encounters it through <context>
  Then the complete post is visible as-is
  And no language-hidden placeholder is shown

  Examples:
    | context |
    | direct post view |
    | complete thread context |
    | quoted context of an eligible outer quote |
    | any notification |
    | Saved Posts or a saved folder |
    | the viewer's own content on any surface |
```

### AT-013: External region and script tags remain lossless but do not base-match
Requirement IDs: NFR-001, RULE-010
Acceptance Criteria: AC-028
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/index/craftsky_post_test.go`, `appview/internal/api/language_visibility_test.go`

```gherkin
Scenario: A French Canadian external post is read by a base-French filter
  Given an external record contains the valid tag fr-CA
  And the viewer selected the v1 base tag fr
  When the post is indexed and queried
  Then the exact fr-CA tag is preserved in storage and responses
  And the post is excluded from filtered browse and discovery
  And it remains visible through direct, contextual, saved, notification, and ownership exceptions
```

### AT-014: Language controls remain accessible
Requirement IDs: FR-012, NFR-004
Acceptance Criteria: AC-017
Priority: Should
Level: Acceptance
Automation Target: `app/test/languages/language_accessibility_test.dart`

```gherkin
Scenario: Selection state is available without colour
  Given enlarged text and accessibility semantics are enabled
  When a keyboard or screen-reader user operates the Languages page and post-language selector
  Then every control is reachable in a logical order
  And labels, limits, errors, and selected state are exposed semantically
  And essential controls remain visible without unusable clipping
```

### AT-015: The composed client flow connects the account-scoped feature contracts
Requirement IDs: BR-001, BR-002, BR-003
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `app/test/languages/post_languages_flow_test.dart`

```gherkin
Feature: Post and Content languages in the Flutter client
  Scenario: A user configures, publishes, and browses language-tagged content through fake repositories
    Given a fresh account is initialised from the device locales
    When the user inspects all three settings, changes Primary and Content languages, and publishes a tagged post
    Then the saved account preferences reload correctly
    And the created post contains the chosen public language tags
    And affected browse and discovery surfaces use the stored Content languages
    And direct and contextual exception surfaces still show complete posts
    And this client test does not claim a live PDS, Tap, AppView, or Postgres round trip
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-006, NFR-001, RULE-001 | AC-008 | Validate post language arrays at the AppView boundary. | Omitted; `[]`; 1–3 valid unique tags; duplicate; invalid; 4 tags. | Omitted, empty, and valid arrays pass; duplicate, invalid, and over-limit values return a `langs` validation error without a PDS call. | `appview/internal/api/post_request_test.go` |
| UT-002 | FR-014 | AC-019 | Normalise ordered device locales into supported unique base tags. | `fr-CA`, `en-GB`, `fr-FR`, unsupported, repeats, empty. | First-seen supported bases are `fr`, `en`; Primary is first; unsupported values are discarded; empty result falls back to `en`/`["en"]`. | `app/test/languages/device_locale_languages_test.dart` |
| UT-003 | RULE-002, RULE-003 | AC-010, AC-011, AC-024 | Define the test-data truth table for base-tag any-match eligibility and untagged behavior without requiring a duplicate standalone production policy helper. | Active `["en","cy"]` and empty filters against tagged, multilingual, and untagged posts. | Any exact match passes; mismatched/untagged fail only while active; empty filter passes all; the same cases seed the authoritative database-backed corpus. | Test-only assertions in `appview/internal/api/language_visibility_test.go` |
| UT-004 | NFR-001, RULE-010 | AC-028 | Keep v1 matching exact rather than language-range based. | Selected `fr`; post tags `fr`, `fr-CA`, `fr-Latn`, `en`. | Only exact `fr` matches; every valid original tag is preserved. | `appview/internal/api/language_eligibility_test.go` |
| UT-005 | RULE-004 | AC-013 | Resolve the language-bearing record for repost and quote eligibility. | Repost actor/subject tags; outer quote/quoted tags. | Repost uses subject; quote uses outer record; quoted record is context only. | `appview/internal/api/language_eligibility_test.go` |
| UT-006 | FR-003, FR-005, RULE-001, RULE-009 | AC-006, AC-007, AC-026 | Model a composer's one-to-three distinct language selection. | Primary plus add/remove attempts through zero and four items. | Opens with Primary only, prevents zero/four/duplicates, and does not persist one-off selections. | `app/test/languages/post_language_selection_test.dart` |
| UT-007 | FR-003, FR-004, RULE-005 | AC-004, AC-005 | Keep Primary and Content mutations independent. | Change either setting while observing the other. | Only the explicitly changed field changes. | `app/test/languages/models/language_preferences_test.dart` |
| UT-008 | FR-012, NFR-001 | AC-027 | Validate the versioned v1 catalogue. | Catalogue snapshot. | Values are unique supported base tags, labels are searchable, and the set is not limited to App localisations. | `app/test/languages/language_catalogue_test.dart` |
| UT-009 | FR-012, NFR-001 | AC-017, AC-027 | Render safe labels for stored valid tags absent from the current catalogue. | Known `fr`; external valid `fr-CA`; unknown future valid tag. | Known values use friendly names; preserved valid unknown values use a non-empty fallback label and are not dropped. | `app/test/languages/language_label_test.dart` |
| UT-010 | FR-010 | AC-015 | Map `langs` consistently in Go and Dart response models. | Tagged array, missing/null legacy value, empty array. | Tagged values round-trip; missing/null normalises to `[]`; encoding uses camelCase `langs`. | `appview/internal/api/post_response_test.go`, `app/test/feed/models/post_test.dart` |
| UT-011 | FR-013, FR-016, NFR-001 | AC-022 | Validate a complete private preference replacement or initialisation body with strict JSON decoding. | Invalid Primary, unsupported Content, duplicate Content, valid empty Content, unknown `accountDid`. | Invalid or unknown input is rejected atomically; empty Content is valid; no partial value or client-selected DID is produced. | `appview/internal/api/language_preferences_request_test.go` |
| UT-012 | FR-011, FR-017, NFR-005 | AC-014, AC-021, AC-023 | Reject stale preference and list completions by account and generation. | Account switch and preference update while futures are pending. | Only the current DID and generation may publish state. | `app/test/languages/providers/account_language_preferences_test.dart` |
| UT-013 | FR-011 | AC-014, AC-024 | Enumerate every affected Flutter cache for invalidation. | Successful Content update. | Timeline, Projects, post/project/hashtag search, and profile post/project/comment state all reset; unrelated direct, notification, and saved state is retained. | `app/test/languages/providers/content_language_invalidation_test.dart` |
| UT-014 | NFR-003 | AC-018 | Redact language values from logs and errors. | Preference and list operations containing distinctive language arrays. | Operation, surface, active flag, and counts may appear; full Primary, Content, and raw locale lists do not. | `appview/internal/observability/language_redaction_test.go`, `app/test/shared/errors/language_redaction_test.dart` |
| UT-015 | FR-006 | AC-008 | Build the standard validation envelope for `langs`. | Each invalid request category. | Response is `400`, camelCase standard envelope, field identifies `langs`, and no complete submitted array is echoed. | `appview/internal/api/post_request_test.go` |
| UT-016 | FR-005, FR-007 | AC-006, AC-009 | Map each composer variant into the shared create-post language field. | General, project, reply, and quote payloads. | Every payload contains the current one-to-three selections in order. | `app/test/feed/providers/create_post_provider_test.dart`, `app/test/projects/project_composer_payload_test.dart` |
| UT-017 | FR-003, RULE-006, RULE-009 | AC-006, AC-026 | Preserve an open composer while changing the future default. | Open with English, explicitly select French, then change Primary to Welsh. | Open composer remains French; next composer opens with Welsh only. | `app/test/languages/post_language_selection_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, FR-002 | AC-002, AC-003 | Settings route opens the English-only Languages page. | Flutter app harness with signed-in account. | Tap Languages in Settings. | Dedicated page and three sections render; only App language is limited to English. | `app/test/settings/languages_page_test.dart`, `app/test/router/settings_routes_test.dart` |
| IT-002 | FR-003, FR-004, FR-013, NFR-003, NFR-005 | AC-004, AC-005, AC-016 | Read and completely replace private preferences by authenticated DID. | Two authenticated DIDs and distinct stored rows. | Call `GET /v1/languages/preferences` and `PUT /v1/languages/preferences` without any account selector. | Each caller sees/changes only its row; PUT atomically replaces both values; responses are camelCase; App language is absent. | `appview/internal/api/language_preferences_test.go` |
| IT-003 | FR-015 | AC-020 | Initialise one new account concurrently. | No row; two concurrent transactions with different proposals. | Execute create-if-absent simultaneously. | Exactly one row exists and both responses equal the authoritative stored row. | `appview/internal/api/language_preferences_store_test.go` |
| IT-004 | FR-013, FR-016, NFR-001 | AC-022 | Strict preference validation and persistence are atomic. | Existing valid row. | Submit invalid Primary, unsupported Content, duplicate Content, unknown `accountDid`, and unexpected query parameters to PUT or initialize. | Every call returns the standard `400` envelope; no supplied account is selected; stored Primary and Content remain unchanged. | `appview/internal/api/language_preferences_test.go` |
| IT-005 | FR-006, FR-007, RULE-001 | AC-008, AC-009 | Post handler writes valid languages and rejects invalid values before PDS. | Fake PDS writer and authenticated handler. | Create valid bilingual and invalid posts. | Valid record has exact `langs`; synthetic response matches; invalid calls never reach fake PDS. | `appview/internal/api/post_test.go` |
| IT-006 | FR-008, NFR-001, RULE-010 | AC-009, AC-028 | Index create/update/untagged/external language values. | Postgres schema and generated Lexicon records. | Handle tagged create, tag update, absent `langs`, and valid `fr-CA`. | Materialised array tracks latest record, untagged remains empty, exact external tag is preserved, malformed records fail safely. | `appview/internal/index/craftsky_post_test.go` |
| IT-007 | FR-010 | AC-015 | All post response paths expose canonical `langs`. | Indexed tagged/untagged rows and synthetic create row. | Build direct, list, search, saved, notification-context, and create responses. | Tagged arrays stay exact; untagged values are always `[]`; Dart decoding matches. | `appview/internal/api/post_response_test.go`, `app/test/feed/data/post_api_client_test.dart` |
| IT-008 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-010, AC-011, AC-012, AC-024 | Run the shared corpus against home timeline. | `internal/testdb.WithSchema` visibility corpus and stored viewer preferences. | List filtered and show-all pages across cursors. | Contract, ownership exception, order, and cursor behavior all hold. | `appview/internal/api/language_visibility_test.go` |
| IT-009 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Run the shared corpus against Projects browse. | Same corpus with project records. | List filtered and show-all pages. | Same eligibility and pagination contract holds. | `appview/internal/api/language_visibility_test.go` |
| IT-010 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Run the shared corpus against post search. | Same corpus with a shared matching query. | Search filtered and show-all pages. | Language filtering composes with existing search order and cursors. | `appview/internal/api/language_visibility_test.go` |
| IT-011 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Run the shared corpus against project search. | Same corpus with project search values. | Search filtered and show-all pages. | Language filtering composes with existing project search rules. | `appview/internal/api/language_visibility_test.go` |
| IT-012 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Run the shared corpus against hashtag post results. | Same corpus sharing one hashtag. | List hashtag pages. | Eligibility is applied before limit/cursor without changing ranking/order. | `appview/internal/api/language_visibility_test.go` |
| IT-013 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Run the shared corpus against another user's profile posts. | Same corpus scoped to the target author; separate own-profile control. | List other-user and own-profile post pages. | Other users' mismatched/untagged rows are hidden; the viewer's own rows remain eligible on their profile. | `appview/internal/api/language_visibility_test.go` |
| IT-014 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Run the shared corpus against profile projects. | Same corpus with project rows for another user and the viewer. | List other-user and own-profile project pages. | Shared contract and profile chronology hold; viewer-authored projects remain visible on the viewer's profile. | `appview/internal/api/language_visibility_test.go` |
| IT-015 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Run the shared corpus against profile comments. | Same corpus with reply/comment rows for another user and the viewer. | List other-user and own-profile comment pages. | Shared contract and comment chronology hold; viewer-authored comments remain visible on the viewer's profile. | `appview/internal/api/language_visibility_test.go` |
| IT-016 | FR-009, NFR-002 | AC-012, AC-024 | Exercise pagination ties and mismatches across all eight adapters. | Page size two; matching and mismatched rows interleaved around equal sort keys. | Traverse every cursor to exhaustion. | Eligible rows appear exactly once in the pre-existing stable order. | `appview/internal/api/language_visibility_pagination_test.go` |
| IT-017 | FR-009 | AC-024 | Compose language eligibility with existing safety and surface policies. | Matching posts that are muted, blocked, moderated, replies where excluded, or excluded imports. | Query each affected surface. | Existing exclusions still win; language never restores an otherwise-ineligible row. | `appview/internal/api/language_visibility_test.go` |
| IT-018 | FR-003, FR-004, FR-011, RULE-005 | AC-004, AC-005, AC-014, AC-016 | Flutter preference update reconciles authoritative values and caches. | Fake repository with success and failure completions; populated surface providers. | Change Primary or Content. | Primary changes only future composer defaults; successful Content update resets all affected state; failure retains old effective state. | `app/test/languages/providers/language_preferences_provider_test.dart` |
| IT-019 | FR-014, FR-017, NFR-005 | AC-019, AC-021, AC-023 | Bootstrap, retry, and account switch remain account-scoped. | Two accounts, controllable locale source, delayed fake repository. | Initialise, fail/retry, and switch accounts mid-request. | No stale/cache fallback becomes authoritative; correct loading/error state gates composer and filtered surfaces. | `app/test/languages/providers/account_language_preferences_test.dart` |
| IT-020 | RULE-007 | AC-025 | Direct post and complete thread endpoints bypass language filtering. | Mismatched and untagged root/reply chain with active English filter. | Read direct post and complete thread. | Every contextual row is returned as-is without placeholder behavior. | `appview/internal/api/language_visibility_exceptions_test.go` |
| IT-021 | RULE-007 | AC-025 | Every notification category remains visible and opens context. | Reply, mention, quote, like, and repost notifications involving mismatched content, plus a follow notification control. | List and resolve notifications. | Every notification remains present, and content-based destinations return the needed post context. | `appview/internal/api/language_visibility_exceptions_test.go`, `app/test/notifications/notification_open_flow_test.dart` |
| IT-022 | RULE-007 | AC-025 | Saved Posts and folders bypass language filtering. | Save matching, mismatched, and untagged posts; then change Content languages. | List overview/folder and open each row. | All saved rows remain and open normally. | `appview/internal/api/language_visibility_exceptions_test.go`, `app/test/saved_posts/providers/saved_posts_provider_test.dart` |
| IT-023 | FR-013, FR-015 | AC-020 | Returning-account initialisation returns existing data. | Existing row and different device proposal. | Call initialise. | Existing row is returned unchanged and timestamps/data are not overwritten unnecessarily. | `appview/internal/api/language_preferences_test.go` |
| IT-024 | NFR-003 | AC-016, AC-018 | Account deletion removes private preferences and observability stays value-free. | Stored preference row and captured logs. | Delete account data and exercise preference/list operations. | Row is deleted; complete values never occur in captured logs. | `appview/internal/api/account_data_test.go`, `appview/internal/observability/language_redaction_test.go` |
| IT-025 | NFR-002 | AC-012, AC-024 | Representative query plans use suitable filtering/index paths. | Representative tagged/untagged dataset and migrations. | Run `EXPLAIN` checks for affected query families. | Plans avoid accidental post-hydration filtering and unbounded regressions; exact thresholds are set in the coding plan. | `appview/internal/api/language_query_plan_test.go` |
| IT-026 | FR-013, FR-016, NFR-005 | AC-016, AC-019, AC-020, AC-022 | Exact preference routes require authentication, define absent-row behavior, and reject account selectors. | Requests without auth; initialised and uninitialised accounts; bodies containing another DID; any query parameter. | Exercise `GET /v1/languages/preferences`, `PUT /v1/languages/preferences`, and `POST /v1/languages/preferences/initialize`. | Unauthenticated calls return `401`; absent GET/PUT return `404 language_preferences_not_found`; initialize returns `200` with the authoritative row; valid calls key only by the authenticated DID; unknown `did`/`accountDid` fields or any query parameter return `400 invalid_request` and do not read or mutate another row. | `appview/internal/routes/language_preferences_test.go` |
| IT-027 | FR-003, FR-005, FR-007, RULE-006, RULE-009 | AC-006, AC-009, AC-026 | General, project, reply, and quote submissions use one shared language behavior. | Widget/provider harness with Primary and one-off selections. | Submit each composer type, then reopen it. | Each request carries selected tags; every new composer resets to current Primary only. | `app/test/feed/widgets/post_composer_languages_test.dart`, `app/test/projects/widgets/project_composer_languages_test.dart` |
| IT-028 | FR-008, FR-013, FR-015, NFR-003 | AC-009, AC-016, AC-020 | Migrations add post languages and one private row per DID, support deletion, and roll back. | Apply migrations to the repository test database. | Inspect constraints/defaults, insert/delete rows, and run down migration. | Untagged defaults safely, preference DID is unique, account deletion removes preferences, and rollback restores the prior schema. | `appview/internal/db/languages_migration_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Existing Lexicon already allows zero to three language tags. | BR-001, RULE-001 | AC-008 | Assert `lexicon/social/craftsky/feed/post.json` is unchanged by this slice and generated `FeedPost.Langs` remains compatible; no `just lexgen` requirement is introduced. |
| REG-002 | Empty Content languages preserve each surface's existing show-all order and pagination. | FR-009, NFR-002, RULE-003 | AC-011, AC-012, AC-024 | Run existing timeline, Projects, search, hashtag, and profile-list fixtures with `[]` and compare ordered URIs/cursors to their pre-filter expectations. |
| REG-003 | Mutes, blocks, moderation, membership, reply policy, and import exclusions remain authoritative. | FR-009 | AC-024 | Add matching language tags to existing policy fixtures and prove hidden rows remain hidden. |
| REG-004 | Direct post and thread endpoints return complete context. | RULE-007 | AC-025 | Activate a restrictive filter and rerun direct/thread response tests without language-based row loss. |
| REG-005 | Notification categories and destinations are not discovery-filtered. | RULE-007 | AC-025 | Rerun notification list/open fixtures with mismatched post tags and prove unchanged notification visibility. |
| REG-006 | Saved Posts and folders retain saved rows across preference changes. | RULE-007 | AC-025 | Rerun saved pagination/folder tests with mismatched and untagged records. |
| REG-007 | Authors can always find their own posts. | RULE-008 | AC-024, AC-025 | On every applicable surface, activate a nonmatching filter and assert viewer-authored rows remain visible. |
| REG-008 | Eligible quote posts include complete quoted context. | RULE-004, RULE-007 | AC-013, AC-025 | English outer quote remains intact when the embedded post is French or untagged. |
| REG-009 | Existing clients may omit `langs`. | FR-006, FR-008, FR-010 | AC-015 | Create/index a post without `langs`; it remains a valid untagged record and returns `langs: []`. |
| REG-010 | App language remains device-local and English-only. | FR-002, NFR-003 | AC-003, AC-016 | Account preference reads/updates never alter SharedPreferences App language, and signing into another account does not change it. |
| REG-011 | The client cannot override authoritative server filtering. | FR-009, FR-013 | AC-010, AC-024 | Supplying `Accept-Language`, Content languages, or another DID on an affected list request does not change the authenticated account's stored policy. |
| REG-012 | Language filtering remains eligibility-only. | NFR-002 | AC-012, AC-024 | Search relevance and chronological tie-break tests retain the same relative order after nonmatching rows are removed. |
| REG-013 | Settings, composers, and post models remain backward-compatible. | FR-001, FR-005, FR-010 | AC-002, AC-006, AC-007, AC-015 | Existing settings routes, composer discard/facet/image, project composer, and legacy post JSON tests continue to pass after language fields are added. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Shared language catalogue | Versioned v1 base-language tags and English labels, including `en`, `fr`, `es`, `cy`; external-only `fr-CA` and `zh-Hant`. | AT-004, AT-013, AT-014, UT-001, UT-004, UT-008, UT-009 |
| TD-002 | Core visibility corpus | Otherwise-identical English, Welsh, French, English-plus-French, untagged, and viewer-authored French posts. | AT-006, AT-007, IT-008–IT-015 |
| TD-003 | Preference rows | Alice: Primary `en`, Content `["en"]`; Bob: Primary `fr`, Content `["fr","cy"]`; show-all account: Content `[]`. | AT-003, AT-011, IT-002–IT-004, IT-018, IT-019 |
| TD-004 | Device locale sets | Ordered `fr-CA`, `en-GB`, `fr-FR`, unsupported; repeated variants; only unsupported; empty. | AT-002, UT-002, IT-019 |
| TD-005 | Pagination corpus | Matching and mismatched rows interleaved across page boundaries, including equal sort timestamps/scores and terminal pages. | AT-008, IT-008–IT-016, REG-002, REG-012 |
| TD-006 | Repost and quote corpus | French subject reposted by English actor; English outer quote embedding French/untagged content; inverted-language controls. | AT-009, UT-005, REG-008 |
| TD-007 | External record corpus | Valid `fr-CA`, `zh-Hant`, absent/empty `langs`, malformed language JSON, and update from `["en"]` to `["fr"]`. | AT-013, IT-006, REG-009 |
| TD-008 | Account race corpus | Two DIDs, two devices, delayed reads/updates, concurrent conflicting first-use proposals, stale cache entries keyed by DID. | AT-002, AT-003, AT-011, UT-012, IT-002, IT-003, IT-019 |
| TD-009 | Exception corpus | Mismatched root plus replies, all notification categories, saved folders, direct link, viewer-owned post, and eligible quote context. | AT-012, IT-020–IT-022, REG-004–REG-008 |
| TD-010 | Invalid request corpus | Duplicate tags, 4 tags, malformed BCP-47, unsupported preference base, duplicate Content, missing/invalid auth, unknown `did`/`accountDid` JSON fields, and unexpected preference-route query parameters. | AT-003, AT-005, UT-001, UT-011, UT-015, IT-004, IT-005, IT-026 |
| TD-011 | Existing-policy corpus | Matching posts that are muted, blocked in either direction, moderated, excluded replies, nonmember content, or excluded Instagram imports. | IT-017, REG-003 |
| TD-012 | Representative query-plan corpus | Sufficient tagged/untagged rows, common and rare languages, empty filters, and selective existing predicates. | IT-025, MAN-003 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-012, NFR-004 | AC-017, AC-027 | Screen reader and keyboard operation on a physical iOS and Android device. | Enable VoiceOver/TalkBack, attach a hardware keyboard where supported, open Languages and each composer selector, search, add/remove selections, hit the three-language limit, and attempt zero selections. | Focus order is logical; control names, selected state, limits, and errors are announced; every action is available without touch-only or colour-only cues. |
| MAN-002 | NFR-004 | AC-017 | Large-text and narrow-screen usability against the supplied Bluesky reference. | Test maximum supported text scaling on representative narrow iOS and Android screens; inspect App, Primary, and Content sections plus selector sheets. | Text may wrap but essential content and actions are not clipped or overlapped; future-language and show-all explanations remain readable. |
| MAN-003 | NFR-002 | AC-012, AC-024 | Representative database performance sanity check. | With a generated tagged/untagged dataset, run affected endpoints with active and empty filters and inspect latency plus `EXPLAIN (ANALYZE, BUFFERS)` in the development database. | Filtering occurs in SQL before pagination, uses the planned indexes, and has no obvious full-scan or hydration regression beyond thresholds fixed in the coding plan. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Assistive-technology behavior cannot be proven completely by widget semantics tests. | NFR-004 | VoiceOver, TalkBack, platform keyboards, and maximum text scaling have native behavior outside `flutter_test`. | Keep MAN-001 and MAN-002 as release gates; automate deterministic semantics and layout cases in AT-014. |
| GAP-002 | Repository tests cannot establish production-scale query latency. | FR-009, NFR-002 | There is no production dataset or active-user traffic, and final index shape belongs in the coding plan. | Require IT-025, define a representative data volume and thresholds during coding plan, and complete MAN-003 before release. |
| GAP-003 | No live PDS/firehose round trip is part of normal automated acceptance. | FR-006, FR-008 | Deterministic tests use a fake PDS writer and synthetic Tap events. | Cover both boundaries independently in IT-005 and IT-006; use the normal compose-stack smoke path during implementation verification if needed. |

None of these gaps blocks document review or coding-plan design. GAP-001 and GAP-002 remain explicit release considerations.

## 10. Out Of Scope

- Translation UI, translation target behavior, automatic language detection, and mis-tag warnings.
- Selectable region/script variants or BCP-47 language-range matching.
- Algorithmic ranking changes; tests assert only eligibility within each surface's existing order.
- Filtering direct post/thread reads, notifications, Saved Posts, quoted context, or viewer-authored posts.
- Language-based moderation, trust scoring, or verification of an author's chosen tags.
- Lexicon changes, generated Lexicon regeneration, PDS record rewrites, or a production backfill.
- Real provider/device locale dependence in automated tests.
- Browser or non-Flutter clients not present in this repository.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- Requirements-to-test-design approval: user confirmed on 2026-07-29.
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-29-post-languages/`
- Risk level: High.
- Recommended first failing test for implementation: `IT-028`, proving the migration and private preference constraints before API or query code depends on them.
- First authoritative filtering test: `IT-008`, running the exact-match/show-all/untagged corpus against the executed Postgres timeline query before limit/cursor. `UT-003` is a test-data truth table, not a substitute production seam.
- Suggested test order for implementation:
  1. `IT-028`, then `IT-002`–`IT-004`, `IT-023`, and `IT-026`: migrations, exact preference routes, strict request behavior, and private account persistence.
  2. `UT-001`–`UT-005`: tag validation and semantic truth tables, without treating a duplicate Go eligibility helper as authoritative over SQL.
  3. `IT-005`–`IT-007`: create, index, and response propagation.
  4. `IT-008` first, then `IT-009`–`IT-017`: establish one authoritative database-backed timeline slice before expanding the shared corpus across every other surface.
  5. `IT-020`–`IT-022` plus REG-004–REG-008: deliberate/contextual exceptions.
  6. `UT-002`, `UT-006`–`UT-013`, `UT-016`, `UT-017`, and Flutter integrations: bootstrap, settings, composers, invalidation, and account switching.
  7. AT-001–AT-015, remaining regressions, accessibility, query plans, and manual checks.
- Commands discovered:
  - From `app/`: `flutter test <test paths>`
  - From `app/`: `dart analyze`
  - From `appview/`: `go test ./internal/api ./internal/index ./internal/db ./internal/routes ./internal/observability`
  - From the repository root with compose Postgres available: `just test`
- Blocking gaps: None.
- Verification reporting: AT-015 is a composed Flutter flow with fakes; Go/Postgres integration and any compose-stack smoke test are reported separately.
- Review gate: Formal document review is approved. Because risk is High, explicit approval is still required before implementation.
