# Acceptance Test Specification: AppView Facet Endpoints And Plain Profile Bios

## 1. Test Strategy

This feature is medium risk because it crosses AppView API/routes, persistence, identity resolution, Flutter repositories, composer UX, and profile edit/render behavior. Use a test-first sequence that starts at the AppView API contract for facet endpoints, then proves persistence/query semantics, then switches Flutter from mock-backed seams to real endpoint-backed repositories, and finally locks the profile bio plain-text behavior.

- **Acceptance/widget tests:** user-visible composer autocomplete, post submit mention resolution, plain profile bio editing, and plain bio rendering/tap behavior.
- **Unit tests:** endpoint validation/ranking/response shaping, Flutter repository mapping/error handling, bio token parsing/tap dispatch, and removal of `descriptionFacets` from Flutter models/save paths.
- **Integration/store tests:** identity cache search/freshness/refresh, profile-initialization identity-cache upsert, exact Craftsky-only mention resolution, hashtag 28-day root-post counts, auth/device enforcement through routes, and bounded identity-cache backfill.
- **Regression tests:** profile save must not reintroduce `descriptionFacets`; post facet generation must continue emitting valid AT Protocol post facets.
- **Manual checks:** limited to a short end-to-end smoke pass for UX confidence after automated tests pass.

Risk-based review recommendation: **Review recommended before implementation, but not required to proceed.** The requirements carry a medium risk level and no blocking open questions.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-003 | AT-001, AT-002, AT-003, UT-006, UT-007, UT-008, IT-001, IT-003, IT-004 | Acceptance / Unit / Integration | Yes |
| BR-002 | AC-008, AC-009 | AT-004, UT-011, IT-007, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| BR-003 | AC-010, AC-011 | AT-005, UT-009, UT-010, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-001 | AC-001, AC-004, AC-005, AC-014 | AT-001, UT-001, UT-003, IT-001, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-001, AC-005, AC-015 | AT-001, UT-002, UT-003, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-002, AC-006, AC-016 | AT-002, UT-004, UT-007, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-006, EC-001 | AT-002, UT-004, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-003, AC-007, AC-014 | AT-003, UT-001, UT-008, IT-001, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-003, AC-007, EC-002 | AT-003, UT-005, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-001, AC-003, AC-012, EC-006 | AT-001, AT-003, AT-006, UT-006, UT-008 | Acceptance / Unit | Yes |
| FR-008 | AC-002, AC-006, AC-012, AC-016, EC-006 | AT-002, AT-006, UT-007, UT-012, REG-002 | Acceptance / Unit / Regression | Yes |
| FR-009 | AC-008 | AT-004, IT-007 | Acceptance / Integration | Yes |
| FR-010 | AC-009, AC-010 | AT-004, AT-005, UT-011, IT-007, REG-001, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-011 | AC-010, AC-011, EC-003, EC-004 | AT-005, UT-009, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-011, AC-017 | AT-005, UT-010, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-004, AC-013 | IT-002, REG-005 | Integration / Regression | Yes |
| FR-014 | AC-013, AC-018, EC-005 | IT-002, IT-003 | Integration | Yes |
| FR-015 | AC-019 | IT-005, MAN-002 | Integration / Manual | Yes + Manual smoke |
| FR-016 | AC-021 | IT-009, MAN-004 | Integration | Yes + Manual smoke |
| NFR-001 | AC-012 | AT-001, AT-003, UT-013, IT-001, REG-003 | Acceptance / Unit / Integration / Regression | Yes |
| NFR-002 | AC-005, AC-007, AC-014, EC-007 | UT-001, UT-003, UT-005, IT-001, IT-004 | Unit / Integration | Yes |
| NFR-003 | AC-010, AC-020, EC-003, EC-004 | AT-005, UT-009, UT-010, IT-008 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-001, AC-006 | AT-001, AT-002, UT-004, IT-001, IT-003 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-008, AC-009, AC-010 | AT-004, AT-005, UT-011, IT-007, REG-001, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-003 | AC-003, AC-007 | AT-003, UT-005, IT-004 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Composer mention autocomplete uses real Craftsky-only AppView suggestions

Requirement IDs: BR-001, FR-001, FR-002, FR-007, NFR-001, RULE-001
Acceptance Criteria: AC-001, AC-005, AC-012, AC-015
Priority: Must
Level: Acceptance
Automation Target: `app/test/shared/rich_text/facet_autocomplete_editor_test.dart`, plus repository coverage in `app/test/shared/rich_text/facet_suggestion_repository_test.dart`

```gherkin
Feature: AppView-backed composer mention autocomplete
  Scenario: Authenticated user sees only real Craftsky mention suggestions
    Given the Flutter account suggestion repository is backed by the AppView Dio provider
    And the AppView mention endpoint has Craftsky profiles matching "ali"
    And one non-Craftsky account also matches "ali"
    When the user types "@ali" in the post composer
    Then Flutter calls GET /v1/facets/mentions?q=ali&limit=10 through authenticated Dio
    And the suggestion list shows only Craftsky profile accounts
    And required DID, handle, isCraftskyProfile, and viewerIsFollowing fields are rendered
    And unknown optional display name or avatar fields do not prevent the item from appearing
```

### AT-002: Final post submit resolves manually typed mentions exactly

Requirement IDs: BR-001, FR-003, FR-004, FR-008, RULE-001
Acceptance Criteria: AC-002, AC-006, AC-016
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/providers/create_post_provider_test.dart`, `app/test/shared/rich_text/facet_generator_test.dart`

```gherkin
Feature: Exact AppView-backed post mention facets
  Scenario: Manually typed Craftsky handles become post mention facets on submit
    Given the post composer text is "Thanks @alice.craftsky.social and @unknown.example"
    And AppView exact resolve returns a Craftsky DID for "alice.craftsky.social"
    And AppView exact resolve returns 404 mention_not_found for "unknown.example"
    When the user submits the post
    Then exact resolve is called during final facet generation for each mention token
    And the created post body contains a mention facet for "@alice.craftsky.social"
    And no mention facet is emitted for "@unknown.example"
    And submit continues for the rest of the text
```

### AT-003: Composer hashtag autocomplete uses indexed AppView tag counts

Requirement IDs: BR-001, FR-005, FR-006, FR-007, NFR-001, RULE-003
Acceptance Criteria: AC-003, AC-007, AC-012, AC-014
Priority: Must
Level: Acceptance
Automation Target: `app/test/shared/rich_text/facet_autocomplete_editor_test.dart`, plus repository coverage in `app/test/shared/rich_text/facet_suggestion_repository_test.dart`

```gherkin
Feature: AppView-backed composer hashtag autocomplete
  Scenario: Authenticated user sees indexed 28-day hashtag counts
    Given indexed root posts contain recent tags matching "sock"
    When the user types "#sock" in the post composer
    Then Flutter calls GET /v1/facets/hashtags?q=sock&limit=10 through authenticated Dio
    And the suggestion list displays lowercase canonical tags without leading "#" in the response model
    And the UI inserts the selected tag with a leading "#"
    And the visible counts use postsLast28Days from AppView
```

### AT-004: Profile bio editing is plain text and saves no facets

Requirement IDs: BR-002, FR-009, FR-010, RULE-002
Acceptance Criteria: AC-008, AC-009
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/edit_profile_dialog_test.dart`, `app/test/profile/data/profile_api_client_test.dart`

```gherkin
Feature: Plain profile bio editing
  Scenario: Bio text containing rich-looking tokens is saved as plain description only
    Given the user opens the edit-profile dialog
    When they type "Knitting with @alice.craftsky.social #lace https://craftsky.social" in the bio field
    Then no mention or hashtag autocomplete appears in the bio editor
    When they save the profile
    Then Flutter sends PUT /v1/profiles/me with description text
    And the request body does not contain descriptionFacets
```

### AT-005: Plain profile bios render detected tokens as clickable ranges

Requirement IDs: BR-003, FR-010, FR-011, FR-012, NFR-003, RULE-002
Acceptance Criteria: AC-010, AC-011, AC-017, AC-020
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/widgets/profile_bio_test.dart`, `app/test/shared/rich_text/faceted_text_actions_test.dart`

```gherkin
Feature: Plain profile bio rendering
  Scenario: Plain bio text becomes clickable without stored facets
    Given a profile response contains description "Visit craftsky.social @alice.craftsky.social #lace"
    And the response contains no descriptionFacets
    When the profile bio renders
    Then the bare domain is styled as a clickable HTTPS link
    And the mention is styled as a clickable profile range
    And the hashtag is styled as a clickable tag range
    When the user taps the mention
    Then the app navigates by the visible handle without pre-resolving it
```

### AT-006: Autocomplete failures fail closed without blocking typing

Requirement IDs: FR-007, FR-008
Acceptance Criteria: AC-012; Edge Case: EC-006
Priority: Must
Level: Acceptance
Automation Target: `app/test/shared/rich_text/facet_autocomplete_editor_test.dart`, `app/test/feed/providers/create_post_provider_test.dart`

```gherkin
Feature: Autocomplete resilience
  Scenario: Suggestion endpoint failure does not block composer text entry
    Given the mention or hashtag suggestion endpoint returns an AppView error envelope
    When the user types in the post composer
    Then the editor remains usable
    And suggestions fail closed or disappear
    And final post submission still performs exact mention resolution for post facets
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-005, NFR-002 | AC-014; EC-007 | Validate suggestion query and limit bounds for both endpoints. | Empty query, whitespace query, 65-character query, missing limit, limit 10, limit 25, limit 26. | Empty/whitespace returns `{items:[]}`; over-64 query returns standard validation error; `limit > 25` returns standard `400 validation_error`; endpoint never returns more than 25 items. | `appview/internal/api/facet_request_test.go` |
| UT-002 | FR-002 | AC-015 | Serialize mention suggestion items with required fields and omitted unknown optional fields. | Suggestible Craftsky row with DID, handle, no display/avatar; viewer follow false. | JSON has camelCase `isCraftskyProfile`, `viewerIsFollowing`; omits unknown optional fields. | `appview/internal/api/facet_response_test.go` |
| UT-003 | FR-001, FR-002, NFR-002 | AC-005 | Rank mention suggestions deterministically. | Followed prefix match, unfollowed prefix match, followed substring match, handle tie. | Followed accounts first, stronger prefix matches before substring matches, handle ascending final tie-breaker. | `appview/internal/api/facet_suggestion_test.go` |
| UT-004 | FR-003, FR-004, RULE-001 | AC-006; EC-001 | Map AppView exact mention resolution outcomes to success or `mention_not_found`. | Valid Craftsky handle, valid non-Craftsky handle, invalid/missing handle. | Craftsky handle returns minimal resolve object; non-Craftsky/missing returns `404 mention_not_found` in the standard AppView error envelope. | `appview/internal/api/facet_test.go` |
| UT-005 | FR-006, NFR-002, RULE-003 | AC-007; EC-002 | Normalize and sort hashtag suggestion rows. | Tags with mixed casing, duplicates, zero/empty tags, equal counts. | Lowercase canonical tags, empty tags excluded, counts non-negative, count desc then tag asc. | `appview/internal/api/facet_suggestion_test.go` |
| UT-006 | FR-007, BR-001 | AC-001, AC-012 | Flutter mention repository maps to AppView suggestion endpoint and decodes items. | Dio mock for `/v1/facets/mentions?q=ali&limit=10` with wrapped items. | Repository returns `AccountSuggestion` values and uses existing Dio base/auth stack. | `app/test/shared/rich_text/facet_suggestion_repository_test.dart` |
| UT-007 | FR-003, FR-008, BR-001 | AC-002, AC-006, AC-016 | Flutter exact mention resolver maps AppView success and `mention_not_found` to facet generation behavior. | Dio success for Alice; 404 `mention_not_found` envelope for unknown. | Alice returns DID; unknown maps to null/no mention facet; resolver is used during final generation only; post submission continues for the rest of the text. | `app/test/shared/rich_text/facet_generator_test.dart`, `app/test/shared/rich_text/facet_suggestion_repository_test.dart` |
| UT-008 | FR-005, FR-007, BR-001 | AC-003, AC-012 | Flutter hashtag repository maps to AppView hashtag endpoint and decodes counts. | Dio mock for `/v1/facets/hashtags?q=sock&limit=10`. | Repository returns `HashtagSuggestion(tag, postsLast28Days)` and maps error envelopes to existing API exceptions. | `app/test/shared/rich_text/facet_suggestion_repository_test.dart` |
| UT-009 | FR-011, NFR-003, BR-003 | AC-010, AC-020; EC-003, EC-004 | Parse plain bio text by centralizing or clearly mirroring post facet generator token rules, without targeting full Bluesky parser parity. | Explicit fixtures for dotted `@handle`, Unicode hashtags, HTTP/S URLs, bare domains, `mailto:`, malformed handles, malformed URL-like text, and URL fragment with `#tag`. | Supported tokens become deterministic non-overlapping ranges; bare domains normalize to HTTPS; non-HTTP/S, malformed, or skipped overlapping tokens remain plain text and do not throw. | `app/test/profile/widgets/profile_bio_test.dart` or a shared parser test |
| UT-010 | FR-012, BR-003 | AC-011, AC-017 | Dispatch bio token taps to existing destinations. | Tap link, hashtag, valid visible handle. | Link launches URL; hashtag routes to tag/search destination; mention routes by visible handle without pre-resolve. | `app/test/shared/rich_text/faceted_text_actions_test.dart`, `app/test/profile/widgets/profile_bio_test.dart` |
| UT-011 | FR-010, BR-002, RULE-002 | AC-009, AC-010 | Remove `descriptionFacets` from Flutter profile model/update/save/bio APIs. | Profile JSON with description only; update request with description; generated mapper/build artifacts. | Profile model has no `descriptionFacets`; PUT body includes `description` only; widget API does not accept facets. | `app/test/profile/models/profile_test.dart`, `app/test/profile/data/profile_api_client_test.dart` |
| UT-012 | FR-008 | AC-002, AC-016 | Preserve post facet generator semantics for links, tags, and exact mention resolution. | Text containing emoji, links, tags, manually typed mentions. | Valid AT Protocol facets with correct UTF-8 byte offsets; no hidden selected-mention state required. | `app/test/shared/rich_text/facet_generator_test.dart` |
| UT-013 | NFR-001 | AC-012 | Validate route-level error envelope helpers for facet validation failures. | Invalid long query or invalid handle. | Error body is `{error, message, requestId}` with camelCase keys and no token leakage. | `appview/internal/api/facet_test.go`, `appview/internal/api/envelope/envelope_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-001, FR-002, FR-005, NFR-001, RULE-001 | AC-001, AC-003, AC-012, AC-014, AC-015 | AppView routes enforce auth/device and return wrapped camelCase facet responses. | Register `/v1/facets/*` routes with auth and device middleware; seed fake session/viewer. | Request mention and hashtag suggestions with and without session/device headers. | Authenticated requests succeed with `{items:[...]}`; missing auth/device fails through standard envelope; optional mention fields omitted when unknown. | `appview/internal/routes/routes_test.go`, `appview/internal/api/facet_test.go` |
| IT-002 | FR-001, FR-013, FR-014 | AC-004, AC-013, AC-018 | Identity cache search and freshness use a separate table. | Test DB schema with `craftsky_profiles`, `bluesky_profiles`, `atproto_follows`, and identity cache rows fresh/stale by timestamp. | Search mention suggestions and inspect update/read behavior. | Search uses cached handles/display names without per-candidate network calls; no handle column is required on `bluesky_profiles`; <=24h entries are fresh. | `appview/internal/api/identity_cache_store_test.go` or `appview/internal/api/facet_store_test.go` |
| IT-003 | FR-003, FR-004, FR-014, RULE-001 | AC-002, AC-006, AC-013, AC-018; EC-001, EC-005 | AppView exact mention resolve refreshes missing/stale cache and filters to Craftsky profiles. | Seed Craftsky profile for Alice, no Craftsky profile for Mallory, stale cache for Alice, fake directory resolver. | Resolve Alice, Mallory, and unknown handles through the AppView endpoint/store. | Alice returns `{did, handle, isCraftskyProfile:true}` and refreshes cache; Mallory/unknown produce `404 mention_not_found`; stale entries are refreshed on successful resolution. | `appview/internal/api/facet_store_test.go`, `appview/internal/api/facet_test.go` |
| IT-004 | FR-005, FR-006, RULE-003 | AC-003, AC-007; EC-002 | Hashtag suggestions count recent root posts only. | Seed `craftsky_posts.tags` with recent roots, old roots, replies/comments, duplicate/lowercase tags, empty tags. | Request `/v1/facets/hashtags?q=sock&limit=10`. | Counts include matching root posts from the last 28 days only, exclude replies/comments/old posts/empty tags, de-dupe lowercase canonical tags, sort count desc then tag asc; no matches returns empty items. | `appview/internal/api/facet_store_test.go` |
| IT-005 | FR-015 | AC-019 | `cli identity-cache backfill` populates handles for existing Craftsky profiles without SQL migration network work. | Test DB has existing Craftsky profile rows and empty identity cache; fake bounded resolver records calls. | Run `cli identity-cache backfill` with default limit 100 and with explicit `--limit <n>`. | Cache rows are inserted/updated for existing profiles; command respects default and explicit limits; migration DDL contains no network resolution. | `appview/cmd/cli/*_test.go`, `appview/internal/api/identity_cache_store_test.go` |
| IT-006 | FR-007, BR-001 | AC-001, AC-003 | Composer autocomplete keeps current insertion behavior with repository overrides. | Widget test with provider overrides for real repository interface/fake data and zero debounce. | Type `@ali`, select account; type `#sock`, select tag. | Suggestions display, order is honored by repository result, selected mention/tag inserts token and preserves focus/selection behavior. | `app/test/shared/rich_text/facet_autocomplete_editor_test.dart` |
| IT-007 | BR-002, FR-009, FR-010, RULE-002 | AC-008, AC-009 | Edit-profile dialog and API client save plain bio only. | Widget/repository fake captures profile update args; Dio adapter captures `PUT /v1/profiles/me`. | Type bio with `@handle`, `#tag`, URL; save. | No autocomplete overlay appears; captured update has description only; request body has no `descriptionFacets`. | `app/test/profile/edit_profile_dialog_test.dart`, `app/test/profile/data/profile_api_client_test.dart` |
| IT-008 | BR-003, FR-011, FR-012, NFR-003 | AC-010, AC-011, AC-017, AC-020 | Plain profile bio rendering/taps work without facets. | Pump profile bio with plain description and fake navigation/launcher hooks. | Inspect spans and tap detected link/mention/hashtag. | Tokens are styled and clickable; bare domain uses HTTPS; unsupported schemes plain; mention navigation uses visible handle without pre-resolve. | `app/test/profile/widgets/profile_bio_test.dart`, `app/test/shared/rich_text/faceted_text_actions_test.dart` |
| IT-009 | FR-016 | AC-021 | Profile initialization upserts the new Craftsky user's identity cache row. | OAuth/profile-initialization handler test with fake authenticated DID, fake handle resolver returning a canonical handle, and identity cache writer/store spy. | Complete the profile initialization / callback path that creates or ensures the Craftsky profile. | The user's DID and canonical handle are upserted into the separate identity cache; no handle column on `bluesky_profiles` is required; failed handle resolution does not create a partial cache row. | `appview/internal/auth/*_test.go`, `appview/internal/api/identity_cache_store_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | AppView `PUT /v1/profiles/me` already does not accept `descriptionFacets`; Flutter must stop sending it instead of expanding the server contract. | BR-002, FR-010, RULE-002 | Replace current `descriptionFacets`-sending tests with assertions that profile update bodies omit `descriptionFacets` even when bio text contains `@handle`, `#tag`, and links. |
| REG-002 | Post text still uses real `app.bsky.richtext.facet` JSON for mentions, tags, and links. | FR-008, NG-005 | Existing and new `FacetGenerator` tests continue to assert UTF-8 byte offsets and valid post facet feature objects for mentions, tags, and links. |
| REG-003 | `/v1/*` authenticated endpoints require Craftsky session auth and `X-Craftsky-Device-Id`. | NFR-001 | Route/middleware tests prove new `/v1/facets/*` endpoints are not accidentally public and return standard errors when auth/device is missing. |
| REG-004 | Flutter profile model/render APIs should not silently preserve generated mapper fields for removed bio facets. | FR-010, RULE-002 | Profile model/mapper tests fail if `descriptionFacets` remains accepted, copied, serialized, or passed into `ProfileBio`. |
| REG-005 | Handles remain identity metadata rather than Bluesky profile record metadata. | FR-013 | Schema/store tests fail if mention autocomplete requires a handle column on `bluesky_profiles` instead of the separate identity cache. |
| REG-006 | New Craftsky users should not be absent from mention autocomplete until an operator backfill runs. | FR-016 | Profile initialization/callback tests fail if successful new-user initialization does not upsert the identity cache. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Mention suggestion ranking and Craftsky-only filtering | Viewer `did:plc:viewer`; Alice `did:plc:alice` handle `alice.craftsky.social`, followed; Alicia `did:plc:alicia`, unfollowed; Mallory `did:plc:mallory`, handle `alice.elsewhere.example`, no Craftsky profile. | AT-001, UT-003, IT-001, IT-002 |
| TD-002 | Optional mention metadata omission | Craftsky profile with cached handle and no `displayName`/avatar. | AT-001, UT-002, IT-001 |
| TD-003 | Exact mention resolution | Alice Craftsky DID; Mallory resolvable non-Craftsky DID; unknown handle returning directory/error not found. | AT-002, UT-004, UT-007, IT-003 |
| TD-004 | Hashtag 28-day counts | Recent root posts tagged `sockkal`, `sockmending`; duplicate roots; old post older than 28 days; reply/comment with matching tag; empty tag. | AT-003, UT-005, IT-004 |
| TD-005 | Profile bio plain text | `Knitting with @alice.craftsky.social #lace craftsky.social https://example.com mailto:x@y.example https://craftsky.social/#lace`; additional fixtures for Unicode hashtags, malformed handles, malformed URL-like text, unsupported schemes, and overlapping URL fragment/hashtag cases. | AT-004, AT-005, UT-009, UT-010, IT-007, IT-008 |
| TD-006 | Query and limit bounds | `q=""`, `q="   "`, 64-char query, 65-char query, `limit=10`, `limit=25`, `limit=26`. | UT-001, IT-001 |
| TD-007 | Identity cache freshness | Cache row resolved at now minus 23h59m, now minus exactly 24h, now minus 24h01m, and missing row. | IT-002, IT-003 |
| TD-008 | Profile save body | Description containing rich-looking tokens with display name/crafts/avatar/banner fields preserved. | AT-004, UT-011, IT-007, REG-001 |
| TD-009 | New-user identity-cache upsert | New authenticated DID `did:plc:new`, canonical handle `new.craftsky.social`, empty/missing identity cache before profile initialization, and fake resolver failure variant. | IT-009, REG-006, MAN-004 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | BR-001, FR-007, FR-008 | Composer UX smoke with real AppView-backed suggestions. | Start dev stack, sign in, type `@` and `#` queries in composer, select suggestions, manually type a valid handle, submit. | Suggestions feel like the existing mock UX; selected and manually typed tokens submit successfully with valid post facets. |
| MAN-002 | FR-015 | Identity-cache backfill operator smoke. | Run `docker compose exec appview /app/cli identity-cache backfill` and one explicit bounded run such as `--limit 10` in a dev database with pre-existing Craftsky profiles. | Command reports bounded progress, respects default limit 100 and explicit limits, does not run from SQL migration, and autocomplete can find backfilled handles. |
| MAN-003 | BR-002, BR-003, FR-009, FR-011, FR-012 | Profile bio edit/render smoke. | Edit a bio containing `@handle`, `#tag`, and links; save; view profile; tap each detected token. | Bio editor has no autocomplete; saved bio is plain text; rendered bio tokens are clickable and route/launch correctly. |
| MAN-004 | FR-016 | New-user identity-cache smoke. | In a dev database with the identity-cache migration applied, sign in or initialize a Craftsky user whose cache row does not exist, then type that handle in mention autocomplete. | The new user's handle has an identity-cache row without running `cli identity-cache backfill`, and autocomplete can find it after normal profile initialization/indexing delay. |

## 9. Resolved Review Decisions, Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| RES-001 | Resolved: over-limit requests reject. | NFR-002 | Post-review decision: `limit > 25` returns standard `400 validation_error`; empty/whitespace query still returns `{items:[]}`. | Coding plan should assert this exact behavior in UT-001 and route/integration tests. |
| RES-002 | Resolved: identity-cache backfill command is named. | FR-015 | Post-review decision: use `cli identity-cache backfill`, default batch limit 100, configurable `--limit <n>`. | Coding plan should wire IT-005/MAN-002 to this command and avoid network work in SQL migrations. |
| RES-003 | Resolved: new Craftsky users populate identity cache immediately. | FR-016 | Manual post-plan feedback identified that backfill alone leaves newly created Craftsky users out of autocomplete until an operator action. | Coding plan should add an identity-cache upsert to the AppView profile creation/initialization path and cover it with IT-009/REG-006. |
| GAP-003 | Plaintext bio parsing intentionally targets supported token rules, not full Bluesky parity. | FR-011, NFR-003 | Requirements ask to mirror existing post facet generator rules and safely treat malformed/ambiguous tokens as plain text. | Document parser fixtures in UT-009; defer any broader Bluesky parser parity to future work. |
| GAP-004 | No automated live network identity freshness test. | FR-014 | Tests should fake the identity resolver to keep the suite deterministic and avoid live atproto dependencies. | Use fake resolver integration tests; rely on manual/dev smoke for operational confidence. |

Blocking gaps: **None identified.**

## 10. Out Of Scope

- Non-Craftsky account autocomplete or exact mention facets.
- General account search, profile discovery, or broad hashtag search pages.
- Lexicon changes or storing facets on `app.bsky.actor.profile.description`.
- Background identity refresh jobs beyond profile-initialization cache upsert, exact resolve refresh, and bounded backfill.
- Live atproto network tests in automated CI; fake resolvers should cover deterministic behavior.
- Direct device storage or exposure of PDS tokens in Flutter.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this agent: `docs/changes/2026-06-02-appview-facet-endpoints-profile-bios/`
- Recommended first failing test for implementation: `IT-001` for authenticated `GET /v1/facets/mentions` returning `{items:[...]}` and enforcing session/device/error-envelope conventions.
- Suggested test order for implementation:
  1. `IT-001`, `UT-001`, `UT-002`, `UT-003`, `UT-004` for AppView handler contracts and validation.
  2. `IT-002`, `IT-003`, `IT-004`, `IT-005`, `IT-009` for identity cache persistence, exact resolve refresh, hashtag counts, backfill, and new-user profile-initialization cache upsert.
  3. `UT-006`, `UT-007`, `UT-008`, `IT-006`, `AT-001`, `AT-002`, `AT-003` for Flutter facet repositories/composer behavior.
  4. `UT-011`, `IT-007`, `REG-001`, `REG-004`, `AT-004` for removing profile `descriptionFacets`.
  5. `UT-009`, `UT-010`, `IT-008`, `AT-005`, `REG-002` for plain bio parsing/taps and post facet regression.
  6. Manual checks `MAN-001` through `MAN-004` after automated tests pass.
- Commands discovered:
  - Go/AppView tests: `just test` from repo root after compose Postgres is running via `just dev-d` or equivalent.
  - Go format/vet: `just fmt`.
  - Focused Flutter tests from `app/`: `flutter test test/shared/rich_text test/profile test/feed/providers/create_post_provider_test.dart`.
  - Focused Flutter API-client tests from `app/`: `flutter test test/profile/data/profile_api_client_test.dart test/shared/rich_text/facet_generator_test.dart`.
- Blocking gaps: None identified.
- Review decisions applied: over-limit requests reject with `400 validation_error`; identity-cache backfill command is `cli identity-cache backfill`; newly created/initialized Craftsky users upsert identity-cache rows without waiting for backfill; profile bio parsing targets Craftsky-supported token fixtures rather than full Bluesky parity; AppView exact-resolve tests remain separate from Flutter no-facet fallback tests.
- Review recommendation: Medium-risk change; document review is recommended before implementation, though the user may explicitly skip it.
