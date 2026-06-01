# Acceptance Test Specification: Flutter Facets UI

## 1. Test Strategy

This is a high-risk Flutter-only slice because it changes client write payloads, introduces AT Protocol facet byte-range correctness concerns, and intentionally sends profile `descriptionFacets` before AppView supports them.

Testing should start with pure Dart unit tests for facet detection/generation, byte offsets, token activation, suggestion filtering/sorting, and renderer range normalization. Then add repository/API/provider integration tests for payload propagation and mock-backed seams. Widget acceptance tests should cover the user-visible post composer, profile bio editor, post/profile rendering, tap destinations, and existing flow regressions. Manual checks are limited to accessibility/visual checks that are hard to fully assert in widget tests.

Risk-based review recommendation: **High risk — document review and explicit approval are required before implementation continues.**

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-003, AC-004 | AT-001, AT-002, AT-005, UT-001, UT-002, WT via AT targets | Acceptance / Unit / Widget | Yes |
| BR-002 | AC-005, AC-006, AC-007 | AT-003, AT-004, AT-008, UT-012, UT-013, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-001 | AC-001, AC-008, AC-009 | AT-001, UT-001, UT-002, UT-003, UT-004, UT-005, IT-001, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-008, AC-010 | AT-001, IT-001, IT-002, IT-003 | Acceptance / Integration | Yes |
| FR-003 | AC-011, AC-012 | AT-002, IT-004, IT-006, MAN-003, GAP-001 | Acceptance / Integration / Manual | Partial: backend unsupported |
| FR-004 | AC-002, AC-013, AC-014 | AT-005, UT-009, UT-010, IT-008 | Acceptance / Unit / Widget | Yes |
| FR-005 | AC-003, AC-013, AC-014 | AT-005, UT-009, UT-010, IT-009 | Acceptance / Unit / Widget | Yes |
| FR-006 | AC-005, AC-015, AC-016, AC-030 | AT-003, AT-007, UT-011, UT-012, IT-005 | Acceptance / Unit / Widget | Yes |
| FR-007 | AC-005, AC-015, AC-016, AC-030 | AT-003, AT-007, UT-011, UT-012, IT-005 | Acceptance / Unit / Widget | Yes |
| FR-008 | AC-006, AC-015, AC-017, AC-030 | AT-004, AT-007, UT-011, UT-013, IT-005 | Acceptance / Unit / Widget | Yes |
| FR-009 | AC-006, AC-017 | AT-004, UT-013 | Acceptance / Unit / Widget | Yes |
| FR-010 | AC-005, AC-018, AC-025 | AT-003, UT-012, IT-005 | Acceptance / Unit / Widget | Yes |
| FR-011 | AC-007, AC-019 | AT-008, IT-005, IT-007 | Acceptance / Integration | Yes |
| FR-012 | AC-004, AC-009, AC-031 | AT-001, UT-004, UT-005, UT-006 | Acceptance / Unit | Yes |
| FR-013 | AC-020, AC-026, AC-027, AC-028 | AT-006, UT-014, IT-010 | Acceptance / Unit / Widget | Yes |
| NFR-001 | AC-009, AC-014 | UT-002, UT-003, UT-009, UT-010 | Unit | Yes |
| NFR-002 | AC-015 | AT-007, UT-015, IT-005 | Acceptance / Unit / Widget | Yes |
| NFR-003 | AC-016, AC-017 | AT-003, AT-004, MAN-001 | Acceptance / Manual | Partial: manual accessibility review recommended |
| NFR-004 | AC-021, AC-022 | REG-001, REG-002, REG-003, REG-004, REG-005 | Regression | Yes |
| NFR-005 | AC-002, AC-003, AC-029 | AT-005, UT-016, IT-008, IT-009, MAN-002 | Acceptance / Unit / Widget / Manual | Yes |
| RULE-001 | AC-007, AC-023 | AT-008, UT-017, IT-007, GAP-002 | Acceptance / Unit / Integration | Yes, with dependency-review gap |
| RULE-002 | AC-009, AC-024, AC-032 | AT-001, AT-002, UT-001, UT-007, UT-008 | Acceptance / Unit | Yes |
| RULE-003 | AC-009, AC-017, AC-033 | AT-001, AT-004, UT-003, UT-018 | Acceptance / Unit / Widget | Yes |
| RULE-004 | AC-012 | IT-006, MAN-003, GAP-001 | Integration / Manual | Partial: backend follow-up required |
| RULE-005 | AC-034 | AT-007, UT-011, UT-019 | Acceptance / Unit / Widget | Yes |
| RULE-006 | AC-016, AC-017, AC-030 | AT-003, AT-004, UT-020 | Acceptance / Unit / Widget | Yes |
| RULE-007 | AC-033 | UT-018 | Unit | Yes |
| RULE-008 | AC-031 | UT-004, UT-005, UT-006 | Unit | Yes |
| RULE-009 | AC-014, AC-035 | AT-005, UT-009, UT-010 | Acceptance / Unit / Widget | Yes |

## 3. Acceptance Scenarios

### AT-001: Compose a post with generated mention, link, and hashtag facets

Requirement IDs: BR-001, FR-001, FR-002, FR-012, RULE-002, RULE-003  
Acceptance Criteria: AC-001, AC-004, AC-008, AC-009, AC-010, AC-024, AC-031, AC-032, AC-033  
Priority: Must  
Level: Acceptance / Widget  
Automation Target: `app/test/feed/widgets/post_composer_sheet_facets_test.dart`

```gherkin
Feature: Post composer facet generation
  Scenario: User submits a post containing supported rich-text entities
    Given the post composer has mock Craftsky resolver data for alice.craftsky.social
    And the user enters "🧶 Hi @alice.craftsky.social see craftsky.social, #SockKAL"
    When the user submits the post
    Then the submitted post request includes facets for the resolved mention, normalized link, and tag
    And each facet byte range points to the visible entity text
    And the bare-domain link URI is "https://craftsky.social"
    And the hashtag tag value is "SockKAL" without the leading "#"
```

### AT-002: Save a profile bio with description facets

Requirement IDs: FR-003, RULE-002, RULE-004  
Acceptance Criteria: AC-011, AC-012, AC-024, AC-032  
Priority: Must  
Level: Acceptance / Widget  
Automation Target: `app/test/profile/edit_profile_dialog_facets_test.dart`

```gherkin
Feature: Profile bio facet generation
  Scenario: User saves a profile bio with generated facets
    Given the profile editor is opened with mock Craftsky resolver data
    And the user changes the bio to "Knitting with @alice.craftsky.social #Lace"
    When the user saves the profile
    Then the profile repository receives descriptionFacets with resolved mention and tag facets
    And the existing display name, crafts, avatar, and banner update semantics are preserved
    And the test documents that the live AppView may reject descriptionFacets until backend support lands
```

### AT-003: Mention autocomplete works in post and profile editors

Requirement IDs: BR-002, FR-006, FR-007, FR-010, RULE-006, NFR-003  
Acceptance Criteria: AC-005, AC-016, AC-018, AC-025, AC-030  
Priority: Must  
Level: Acceptance / Widget  
Automation Target: `app/test/shared/rich_text/facet_autocomplete_editor_test.dart`, `app/test/feed/widgets/post_composer_sheet_facets_test.dart`, `app/test/profile/edit_profile_dialog_facets_test.dart`

```gherkin
Feature: Mention autocomplete
  Scenario Outline: User selects a Craftsky mention suggestion
    Given the <editor> has mock account suggestions containing followed, non-followed, and non-Craftsky accounts
    When the user types "@ali" at a valid token boundary and waits for debounce
    Then only matching Craftsky accounts are shown
    And followed accounts are listed before otherwise matching non-followed accounts
    And each visible suggestion shows avatar, display name, and handle
    When the user selects alice.craftsky.social
    Then the active token is replaced with "@alice.craftsky.social "
    And focus remains in the editor with the caret after the trailing space

    Examples:
      | editor             |
      | post composer      |
      | profile bio editor |
```

### AT-004: Hashtag autocomplete works in post and profile editors

Requirement IDs: BR-002, FR-008, FR-009, RULE-003, RULE-006  
Acceptance Criteria: AC-006, AC-017, AC-030, AC-033  
Priority: Must  
Level: Acceptance / Widget  
Automation Target: `app/test/shared/rich_text/facet_autocomplete_editor_test.dart`, `app/test/feed/widgets/post_composer_sheet_facets_test.dart`, `app/test/profile/edit_profile_dialog_facets_test.dart`

```gherkin
Feature: Hashtag autocomplete
  Scenario Outline: User selects a hashtag suggestion
    Given the <editor> has mock hashtag suggestions with 28-day post counts
    When the user types "#sock" at a valid token boundary and waits for debounce
    Then matching hashtag suggestions are shown with their 28-day post counts
    When the user selects "#SockKAL"
    Then the active token is replaced with "#SockKAL "
    And focus remains in the editor with the caret after the trailing space
    And generated tag facets store "SockKAL" without "#"

    Examples:
      | editor             |
      | post composer      |
      | profile bio editor |
```

### AT-005: Faceted text renders safely in posts and profile descriptions

Requirement IDs: BR-001, FR-004, FR-005, NFR-001, NFR-005, RULE-009  
Acceptance Criteria: AC-002, AC-003, AC-013, AC-014, AC-029, AC-035  
Priority: Must  
Level: Acceptance / Widget  
Automation Target: `app/test/shared/rich_text/faceted_text_test.dart`, `app/test/feed/widgets/post_card_test.dart`, `app/test/profile/widgets/profile_bio_test.dart`

```gherkin
Feature: Rich-text rendering
  Scenario Outline: Render supported facets and tolerate malformed metadata
    Given <surface> text contains valid mention, link, and hashtag facets
    When the surface renders
    Then faceted ranges use the theme primary color
    And non-faceted ranges use the normal text style
    When the surface receives missing, unsupported, overlapping, or invalid UTF-8 byte ranges
    Then the UI falls back safely by dropping only invalid facets or rendering plain text
    And the app does not throw

    Examples:
      | surface             |
      | post card text      |
      | profile description |
      | editable editor     |
```

### AT-006: Tapping rendered facets invokes destination behavior safely

Requirement IDs: FR-013  
Acceptance Criteria: AC-020, AC-026, AC-027, AC-028  
Priority: Must  
Level: Acceptance / Widget  
Automation Target: `app/test/shared/rich_text/faceted_text_actions_test.dart`

```gherkin
Feature: Facet tap actions
  Scenario: User taps supported facets
    Given rendered text has mention, link, and hashtag facets
    When the user taps the mention facet with visible text "@alice.craftsky.social"
    Then Flutter navigates to Alice's profile using the visible handle
    When the user taps the link facet
    Then Flutter requests url_launcher to open the URL
    When the user taps the hashtag facet "#SockKAL"
    Then Flutter navigates to the search route with tag context "SockKAL"
    When any destination action fails
    Then the app does not crash
```

### AT-007: Autocomplete respects debounce and token-boundary activation

Requirement IDs: FR-006, FR-007, FR-008, NFR-002, RULE-005  
Acceptance Criteria: AC-015, AC-034  
Priority: Must  
Level: Acceptance / Widget  
Automation Target: `app/test/shared/rich_text/facet_autocomplete_editor_test.dart`

```gherkin
Feature: Autocomplete activation
  Scenario: User edits active tokens quickly
    Given the debounce duration is injected as 300 ms
    When the user types "@a" and then "@al" within 300 ms
    Then the mock account repository is not queried until the user pauses for the debounce period
    When "@" or "#" appears inside a word, email address, URL fragment, or without a query character
    Then autocomplete does not query and does not show suggestions
    When the trigger appears at text start or after whitespace or opening punctuation with at least one query character
    Then autocomplete may query after debounce
```

### AT-008: Autocomplete and mention resolution use only mock/injected Flutter data

Requirement IDs: BR-002, FR-011, RULE-001  
Acceptance Criteria: AC-007, AC-019, AC-023  
Priority: Must  
Level: Acceptance / Integration  
Automation Target: `app/test/shared/rich_text/facet_suggestion_repository_test.dart`, `app/test/shared/rich_text/facet_providers_test.dart`

```gherkin
Feature: Mock-backed suggestions
  Scenario: Tests override suggestion repositories
    Given the editor is wrapped in a ProviderScope with mock suggestion and resolver overrides
    When mention and hashtag suggestions are requested
    Then the editor uses the injected Flutter repositories
    And the test passes without AppView, PDS, or external identity network access
```

### AT-009: Existing composer and profile edit flows remain stable

Requirement IDs: NFR-004  
Acceptance Criteria: AC-021, AC-022  
Priority: Must  
Level: Acceptance / Regression  
Automation Target: existing `app/test/feed/widgets/post_composer_sheet_*_test.dart`, `app/test/profile/edit_profile_dialog_test.dart`

```gherkin
Feature: Existing flow stability
  Scenario: Facet UI does not regress existing validation and save behavior
    Given existing post composer and profile editor tests cover validation, dirty state, images, replies, and discard confirmation
    When facet UI is added
    Then those existing tests continue to pass
    And any live profile save failure caused by descriptionFacets remains documented as the known compatibility risk
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, RULE-002 | AC-001, AC-009, AC-024, AC-032 | Generates mention facets only for syntactically valid handles resolved by injected Craftsky resolver data, including manually typed handles. | `Hi @alice.craftsky.social @unknown.example`; resolver maps Alice only. | One mention facet with Alice DID; unknown remains plain text; no warning flag. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-002 | FR-001, NFR-001 | AC-009 | Computes UTF-8 byte offsets with emoji/multibyte text before facets. | `🧶 café @alice.craftsky.social #Mending` | Byte start/end values match UTF-8 encoded ranges and do not split characters. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-003 | FR-001, NFR-001, RULE-003 | AC-009, AC-033 | Avoids overlapping generated facets when potential hashtag text appears inside a URL fragment. | `https://craftsky.social/tags#SockKAL #SockKAL` | Link facet covers URL per link rules; no overlapping tag in URL fragment; separate trailing hashtag facet generated. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-004 | FR-012, RULE-008 | AC-004, AC-031 | Recognizes explicit HTTP/HTTPS URLs and bare domains. | `http://a.example https://b.example craftsky.social` | Three link facets; explicit URLs preserved; bare domain URI normalized to HTTPS. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-005 | FR-012, RULE-008 | AC-031 | Trims trailing sentence punctuation from link ranges and URIs. | `See craftsky.social, (https://example.com/path).` | Link ranges exclude comma, period, and unmatched closing punctuation. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-006 | FR-012, RULE-008 | AC-031 | Does not generate markdown-style link semantics. | `[Craftsky](https://craftsky.social)` | Only the visible URL text, if recognized by parser rules, may become a link; no hidden markdown target behavior. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-007 | RULE-002 | AC-024 | Unknown manually typed mentions do not block submit/save. | `@unknown.example` with empty resolver. | No mention facet; output text unchanged; no validation error. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-008 | RULE-002 | AC-032 | Manually typed resolvable mentions create facets without autocomplete selection state. | `Thanks @alice.craftsky.social` plus resolver data. | Mention facet includes Alice DID. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-009 | FR-004, FR-005, RULE-009 | AC-013, AC-014, AC-035 | Normalizes incoming facet ranges for rendering. | Unsorted, overlapping, unsupported, multi-feature, and out-of-range raw facets. | Valid facets are sorted; unsupported variants ignored; first supported feature per range wins; invalid ranges dropped safely. | `app/test/shared/rich_text/faceted_text_model_test.dart` |
| UT-010 | FR-004, FR-005, NFR-001, RULE-009 | AC-014, AC-035 | Drops only facets whose byte ranges split multibyte characters. | Text with emoji and one bad range into emoji plus one good range. | Bad facet dropped; good facet remains; no exception. | `app/test/shared/rich_text/faceted_text_model_test.dart` |
| UT-011 | FR-006, FR-007, FR-008, RULE-005 | AC-034 | Detects active autocomplete tokens only at valid boundaries and with minimum query length. | Start-of-text, whitespace, `(`, inside word, email, URL fragment, bare `@`, bare `#`. | Active token only for valid boundary plus at least one query character. | `app/test/shared/rich_text/facet_autocomplete_controller_test.dart` |
| UT-012 | BR-002, FR-010 | AC-005, AC-018, AC-025 | Filters and sorts account suggestions. | Mock accounts: followed Craftsky, non-followed Craftsky, non-Craftsky. | Non-Craftsky excluded; followed matches first; display data retained. | `app/test/shared/rich_text/mock_account_suggestion_repository_test.dart` |
| UT-013 | BR-002, FR-009 | AC-006, AC-017 | Filters hashtag suggestions and exposes 28-day count labels. | Tags `SockKAL`, `sockmending` with counts. | Matching tags returned with count values available to UI. | `app/test/shared/rich_text/mock_hashtag_suggestion_repository_test.dart` |
| UT-014 | FR-013 | AC-020, AC-026, AC-027, AC-028 | Maps supported facet taps to destination intents and fails safely. | Mention with valid/invalid visible handle, link, hashtag, failing launcher callback. | Correct intent emitted; invalid/failing action does not throw. | `app/test/shared/rich_text/facet_action_handler_test.dart` |
| UT-015 | NFR-002 | AC-015 | Debounce is injectable and delays lookups until the configured duration. | Fake clock; multiple edits within 300 ms. | One query after pause; no query for superseded token. | `app/test/shared/rich_text/facet_autocomplete_controller_test.dart` |
| UT-016 | NFR-005 | AC-002, AC-003, AC-029 | Produces themed text spans for editable and rendered facet ranges. | Theme primary color and sample spans. | Facet spans use `Theme.colorScheme.primary`; plain spans do not. | `app/test/shared/rich_text/faceted_text_span_builder_test.dart` |
| UT-017 | RULE-001 | AC-007, AC-023 | Facet generator/resolver interfaces are local and injectable. | Mock resolver implementation without Dio/url clients. | Generation completes without network dependencies. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-018 | RULE-003, RULE-007 | AC-033 | Parses hashtag characters per slice rules. | `#SockKAL #café_2026 #sock-knit #🧶craft` | Unicode letters/digits/underscore included; hyphen/emoji excluded; casing preserved. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-019 | RULE-005 | AC-034 | Prevents autocomplete in email addresses and URL fragments. | `hi@craftsky.social`, `https://example.com/#tag`, `abc#tag`. | No active autocomplete token. | `app/test/shared/rich_text/facet_autocomplete_controller_test.dart` |
| UT-020 | RULE-006 | AC-016, AC-017, AC-030 | Replaces only the active partial token and inserts exactly one trailing space. | `Meet (@ali) and #soc`; selection inside token. | Surrounding text preserved; selected value and one trailing space inserted; caret after space. | `app/test/shared/rich_text/facet_autocomplete_controller_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-002 | AC-008 | Post API client includes facets in create body. | DioAdapter expecting `/v1/posts` with `text`, `reply`, `images`, and `facets`. | Call `PostApiClient.createPost(...)` with facets. | Body includes facet JSON unchanged and preserves existing fields. | `app/test/feed/data/post_api_client_test.dart` |
| IT-002 | FR-002 | AC-008, AC-010 | Post repository passes facets through to API client. | Fake or mock API client behind `ApiPostRepository`. | Call repository `create` with facets. | API client receives same facet list. | `app/test/feed/data/post_repository_test.dart` |
| IT-003 | FR-002 | AC-010 | `createPostProvider` passes generated facets to fake repository. | ProviderContainer override with `FakePostRepository` capturing `facets`. | Call notifier create from composer/provider path. | Fake receives same facets that API body should send. | `app/test/feed/providers/create_post_provider_test.dart` |
| IT-004 | FR-003 | AC-011 | Profile API client includes `descriptionFacets`. | DioAdapter expecting `/v1/profiles/me` body with existing fields and `descriptionFacets`. | Call `ProfileApiClient.updateMyProfile(...)`. | Body includes descriptionFacets unchanged. | `app/test/profile/data/profile_api_client_test.dart` |
| IT-005 | FR-006, FR-007, FR-008, FR-010, FR-011, NFR-002 | AC-005, AC-006, AC-015, AC-018, AC-019 | Editors use provider-overridden mock repositories with debounce. | ProviderScope overrides account/hashtag suggestion repositories and short test debounce. | Type active `@` and `#` tokens in shared editor. | Override repositories queried after debounce; UI reflects returned data. | `app/test/shared/rich_text/facet_autocomplete_editor_test.dart` |
| IT-006 | FR-003, RULE-004 | AC-012 | Known live profile `descriptionFacets` incompatibility is explicit. | API-client test simulating AppView `unexpected_field` error for `descriptionFacets`. | Save profile with descriptionFacets. | Error surfaces through existing API exception/error UI path and test name/documentation marks as expected current backend gap. | `app/test/profile/data/profile_api_client_test.dart`, `app/test/profile/edit_profile_dialog_facets_test.dart` |
| IT-007 | FR-011, RULE-001 | AC-007, AC-019, AC-023 | Autocomplete and resolver tests pass without network clients. | Test environment with only mock/injected repositories; no AppView/PDS setup. | Run suggestion and generation tests. | No Dio/AppView/PDS dependency is required. | `app/test/shared/rich_text/facet_providers_test.dart` |
| IT-008 | FR-004, NFR-005 | AC-002, AC-013, AC-014 | Post card uses shared faceted renderer when `Post.facets` exists and plain text otherwise. | Post fixture with valid facets, null facets, malformed facets. | Pump `PostCard`. | Valid ranges styled primary; invalid/null cases render safely. | `app/test/feed/widgets/post_card_test.dart` |
| IT-009 | FR-005, NFR-005 | AC-003, AC-013, AC-014 | Profile bio supports optional description facet metadata. | ProfileBio/profile model fixture with facets, absent facets, malformed facets. | Pump profile bio surface. | Valid ranges styled primary; absent/malformed facets render safely. | `app/test/profile/widgets/profile_bio_test.dart`, `app/test/profile/models/profile_test.dart` |
| IT-010 | FR-013 | AC-020, AC-026, AC-027, AC-028 | Rendered facet actions integrate with router and launcher seams. | Test router/launcher fakes. | Tap mention, link, hashtag, and failing destination. | Profile route, launcher URL, and `/search?tag=...` intents captured; failures do not crash. | `app/test/shared/rich_text/faceted_text_actions_test.dart`, `app/test/router/router_redirect_test.dart`, `app/test/search/search_page_test.dart` |
| IT-011 | FR-003 | AC-011 | Profile model/repository/fake surfaces carry optional description facet metadata. | Updated `Profile`, `ProfileRepository`, `FakeProfileRepository` signatures. | Decode profile JSON and update profile through fake. | Optional `descriptionFacets` is retained and capturable without breaking existing tests. | `app/test/profile/models/profile_test.dart`, `app/test/profile/fakes/fake_profile_repository.dart` dependent tests |
| IT-012 | FR-001, FR-002, FR-003 | AC-008, AC-011 | Generated facets are recomputed from current text at submit/save time. | User selects a mention then edits text before submit/save. | Submit post and save profile. | Payload facets reflect final current text, not stale selected range state. | `app/test/feed/widgets/post_composer_sheet_facets_test.dart`, `app/test/profile/edit_profile_dialog_facets_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Post composer empty/overlong text validation and disabled submit state. | NFR-004 | Keep/extend existing post composer widget tests; verify adding facet editor does not allow empty or overlong submission. |
| REG-002 | Post composer image attachment, image order, alt-text warning, and reply image restrictions. | NFR-004 | Run existing `post_composer_sheet_*` and composer image tests; add one facet-containing post with image to ensure `images` plus `facets` serialize together. |
| REG-003 | Post composer reply behavior, reply references, success pop, error snackbar, and discard confirmation. | NFR-004 | Run existing reply/create/discard tests after editor replacement. |
| REG-004 | Profile edit seeded values, dirty state, validation, successful save, failed save snackbar, and cache update. | NFR-004 | Run existing `app/test/profile/edit_profile_dialog_test.dart` unchanged where possible; add facet payload assertions without weakening existing assertions. |
| REG-005 | Profile image draft, clear avatar/banner, and full atomic profile update semantics. | NFR-004 | Run existing profile widget/API tests and add `descriptionFacets` cases that preserve current null/clear semantics. |
| REG-006 | Post cards with no facets render the exact existing body text and engagement layout. | FR-004, NFR-004 | Keep existing `PostCard` tests; add null/empty facets cases that still find the original body text. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | UTF-8 byte offset and mixed entity generation. | Text: `🧶 café @alice.craftsky.social see craftsky.social, #SockKAL`; Alice DID `did:plc:alice`. | AT-001, UT-001, UT-002, UT-004, UT-005 |
| TD-002 | Account suggestion filtering/sorting. | Accounts: Alice followed Craftsky with avatar/display name/handle/DID; Alicia non-followed Craftsky; Mallory non-Craftsky. | AT-003, UT-012, IT-005 |
| TD-003 | Hashtag suggestions and parsing. | Tags: `#SockKAL` count 128, `#sockmending` count 12, `#café_2026`; hyphen/emoji examples. | AT-004, UT-013, UT-018 |
| TD-004 | Renderer resilience. | Valid facets plus unsupported feature, multi-feature range, overlapping ranges, out-of-range range, byte range splitting an emoji. | AT-005, UT-009, UT-010, IT-008, IT-009 |
| TD-005 | Post create payload. | `text`, optional `reply`, optional `images`, and generated `facets` list using app.bsky.richtext.facet-compatible JSON. | AT-001, IT-001, IT-002, IT-003 |
| TD-006 | Profile update payload. | Existing displayName/description/crafts/avatar/banner values plus generated `descriptionFacets`. | AT-002, IT-004, IT-006, IT-011 |
| TD-007 | Destination action fakes. | Test router capturing profile/search routes; fake launcher capturing URLs and optionally failing. | AT-006, UT-014, IT-010 |
| TD-008 | Existing regression fixtures. | Existing fake post/profile repository fixtures, composer image drafts, reply refs, and profile image blobs. | REG-001 through REG-006 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | NFR-003 | Basic keyboard and screen-reader affordance review for autocomplete. | Run the app locally, focus the post composer/profile bio editor, type `@ali` and `#sock`, navigate suggestions with keyboard/screen reader where available. | Suggestions are reachable/identifiable by visible text or semantics; focus does not get trapped. |
| MAN-002 | NFR-005 | Visual primary-color styling review across surfaces. | Compare post card, profile bio, and editable editor facet colors against the active theme. | Faceted ranges use the theme primary color and remain legible in light/dark/dynamic themes supported by the app. |
| MAN-003 | FR-003, RULE-004 | Confirm live profile-save compatibility risk remains visible. | If pointed at the current AppView, attempt to save a bio with generated descriptionFacets. | Failure is handled without crash and remains documented as expected until backend support lands. Do not treat as Flutter implementation failure for this slice. |
| MAN-004 | FR-013 | Real-device link launch smoke check. | On a simulator/device, tap a rendered link facet. | The platform URL launcher opens or fails gracefully; no crash. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Live AppView profile saves with `descriptionFacets` are expected to fail until backend support lands. | FR-003, RULE-004 | Requirements explicitly keep the risky live send in Flutter while AppView currently rejects unknown profile update fields. | Add AppView/API support in a follow-up slice; keep Flutter tests at repository/API-client/error-handling level for now. |
| GAP-002 | Tests can prove injected/mock data paths but cannot fully prove a third-party helper never performs external resolution unless implementation avoids that API path. | RULE-001 | `bluesky_text` usage details are implementation-dependent. | In implementation review, inspect imports/API calls and require tests to use local resolver seams only; avoid helper methods that perform network identity resolution. |
| GAP-003 | Hashtag search results are out of scope. | FR-013 | Requirements only demand navigation with tag context. | Assert route/query context now; implement search results in a separate slice. |
| GAP-004 | Full accessibility certification is out of scope for widget tests. | NFR-003 | Widget tests can verify visible labels/semantics but not all assistive-tech behavior. | Run MAN-001 and capture follow-up accessibility issues separately. |
| GAP-005 | Exact future AppView autocomplete response shape is unknown. | FR-011 | This slice uses mock repositories shaped for future replacement. | Keep provider/repository interfaces minimal and document any backend contract assumptions during implementation planning. |

## 10. Out Of Scope

- AppView autocomplete endpoints, AppView profile `descriptionFacets` support, migrations, and lexicon changes.
- Real hashtag search/browse results; only search route navigation with tag context is tested.
- Mention notifications, website preview cards, external embeds, or PDS/external identity lookups from Flutter.
- Comprehensive domain parser conformance beyond the explicit bare-domain/HTTP/HTTPS and punctuation rules in requirements.
- Creating actual Dart/Flutter test files in this test-design stage.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-06-01-flutter-facets-ui/01-requirements.md`
- Test specification: `docs/changes/2026-06-01-flutter-facets-ui/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this agent: `docs/changes/2026-06-01-flutter-facets-ui/`
- Recommended first failing test for implementation: `UT-002` in `app/test/shared/rich_text/facet_generator_test.dart` for UTF-8 byte offsets with emoji/multibyte text before a mention/link/hashtag.
- Suggested test order for implementation:
  1. `UT-002`, `UT-001`, `UT-004`, `UT-005`, `UT-018`, `UT-003` for byte-safe facet generation.
  2. `UT-009`, `UT-010`, `UT-016` for renderer normalization and primary-color spans.
  3. `UT-011`, `UT-015`, `UT-020`, `UT-012`, `UT-013` for autocomplete activation, debounce, replacement, and suggestions.
  4. `IT-001` through `IT-004` for post/profile payload propagation.
  5. `AT-003`, `AT-004`, `AT-001`, `AT-002` for editor workflows.
  6. `AT-005`, `AT-006`, `IT-008`, `IT-009`, `IT-010` for rendering and tap actions.
  7. `REG-001` through `REG-006` for existing behavior stability.
- Commands discovered:
  - From repo root: `just test` for Go/AppView tests, requiring the compose Postgres stack.
  - From `app/`: `flutter test <paths>` for focused Flutter tests.
  - From `app/`: `dart run build_runner build --delete-conflicting-outputs` if provider/model generated files change.
  - From `app/`: `flutter pub get` if implementation adds `bluesky_text` or another dependency.
- Blocking gaps:
  - `GAP-001`: live profile `descriptionFacets` backend support remains blocked outside this Flutter-only slice.
  - `GAP-002`: implementation must avoid external identity/PDS calls when using atproto.dart ecosystem helpers.
- Risk-based review recommendation: **High risk; require document review and explicit approval before implementation.**
