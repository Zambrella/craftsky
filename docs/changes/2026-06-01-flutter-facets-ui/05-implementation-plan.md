# TDD Implementation Plan: Flutter Facets UI

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`, high risk)
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Keep this slice Flutter-only: no AppView, migrations, lexicon, PDS, or external identity lookup changes.
- Keep live profile `descriptionFacets` failure visible as a known compatibility risk until a future AppView/API slice lands.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-002 | FR-001, NFR-001 | AC-009 | Fails until byte-safe facet generation exists |
| 2 | UT-001 | FR-001, RULE-002 | AC-001, AC-009, AC-024, AC-032 | Fails until mention resolution/filtering exists |
| 3 | UT-004 | FR-012, RULE-008 | AC-004, AC-031 | Fails until link detection and bare-domain normalization exist |
| 4 | UT-005 | FR-012, RULE-008 | AC-031 | Fails until link punctuation trimming exists |
| 5 | UT-018 | RULE-003, RULE-007 | AC-033 | Fails until hashtag parser matches slice rules |
| 6 | UT-003 | FR-001, NFR-001, RULE-003 | AC-009, AC-033 | Fails until overlap handling exists |
| 7 | UT-009 | FR-004, FR-005, RULE-009 | AC-013, AC-014, AC-035 | Fails until renderer normalizer exists |
| 8 | UT-010 | FR-004, FR-005, NFR-001, RULE-009 | AC-014, AC-035 | Fails until split multibyte ranges are dropped safely |
| 9 | UT-016 | NFR-005 | AC-002, AC-003, AC-029 | Fails until themed span builder exists |
| 10 | UT-011 | FR-006, FR-007, FR-008, RULE-005 | AC-034 | Fails until token detection exists |
| 11 | UT-015 | NFR-002 | AC-015 | Fails until injectable debounce path exists |
| 12 | UT-020 | RULE-006 | AC-016, AC-017, AC-030 | Fails until active-token replacement exists |
| 13 | UT-012 | BR-002, FR-010 | AC-005, AC-018, AC-025 | Fails until account suggestion repository exists |
| 14 | UT-013 | BR-002, FR-009 | AC-006, AC-017 | Fails until hashtag suggestion repository exists |
| 15 | IT-001 | FR-002 | AC-008 | Fails until post API client accepts `facets` |
| 16 | IT-002 | FR-002 | AC-008, AC-010 | Fails until post repository forwards `facets` |
| 17 | IT-003 | FR-002 | AC-010 | Fails until create-post provider forwards `facets` |
| 18 | IT-004 | FR-003 | AC-011 | Fails until profile API client accepts `descriptionFacets` |
| 19 | AT-003 | BR-002, FR-006, FR-007, FR-010, RULE-006, NFR-003 | AC-005, AC-016, AC-018, AC-025, AC-030 | Fails until mention autocomplete UI exists |
| 20 | AT-004 | BR-002, FR-008, FR-009, RULE-003, RULE-006 | AC-006, AC-017, AC-030, AC-033 | Fails until hashtag autocomplete UI exists |
| 21 | AT-001 | BR-001, FR-001, FR-002, FR-012, RULE-002, RULE-003 | AC-001, AC-004, AC-008, AC-009, AC-010, AC-024, AC-031, AC-032, AC-033 | Fails until post composer submits generated facets |
| 22 | AT-002 | FR-003, RULE-002, RULE-004 | AC-011, AC-012, AC-024, AC-032 | Fails until profile save submits `descriptionFacets` |
| 23 | AT-005 | BR-001, FR-004, FR-005, NFR-001, NFR-005, RULE-009 | AC-002, AC-003, AC-013, AC-014, AC-029, AC-035 | Fails until safe faceted rendering exists |
| 24 | AT-006 / UT-014 / IT-010 | FR-013 | AC-020, AC-026, AC-027, AC-028 | Fails until facet tap actions exist |
| 25 | REG-001..REG-006 | NFR-004 | AC-021, AC-022 | Existing tests must stay green after UI wiring |

## Implementation Steps

### Step 1: UT-002
- Write failing test: added `app/test/shared/rich_text/facet_generator_test.dart` with UTF-8 byte-offset assertions for `🧶 café @alice.craftsky.social #Mending`.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart`
- Confirmed failure: compilation failed because `lib/shared/rich_text/facet_generator.dart`, `MentionResolver`, and `FacetGenerator` did not exist.
- Implement: added a Flutter-only `FacetGenerator` with an injected local `MentionResolver`, generated raw mention/tag facet JSON, and computed UTF-8 byte offsets from the final text.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart` — passed (`+1`).
- Refactor: formatted the new source/test files and re-ran the focused command — passed (`+1`).
- Notes: Covers FR-001/NFR-001 and AC-009 for byte-safe generated facet offsets after emoji/multibyte text. No AppView, PDS, external identity, lexicon, migration, or dependency changes were made.

### Later Steps
- Execute the remaining test order above test-by-test from the approved acceptance-test/coding-plan order.
- Update this file after each red/green/refactor loop with failure and green results.

### Step 2: UT-001
- Write failing test: added a focused test in `app/test/shared/rich_text/facet_generator_test.dart` for `Hi @alice.craftsky.social @unknown.example` with only Alice resolvable.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart`
- Confirmed failure: no red failure; the minimal UT-002 implementation already included local mention resolution and unknown-handle filtering needed for UT-001.
- Implement: no additional source changes required.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart` — passed (`+2`).
- Refactor: pending after subsequent generator changes.
- Notes: Covers FR-001/RULE-002 and AC-001/AC-009/AC-024/AC-032 for locally resolved manually typed mentions and unknown mention omission without network calls.

### Step 3: UT-004
- Write failing test: added HTTP, HTTPS, and bare-domain link expectations in `facet_generator_test.dart`.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart`
- Confirmed failure: link facets list was empty (`Expected length <3>, Actual length <0>`).
- Implement: added local link detection for explicit `http://`/`https://` and bare domains, with bare-domain URI normalization to `https://...`.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart` — passed through UT-004 (`+3`).
- Refactor: pending after subsequent generator changes.
- Notes: Covers FR-012/RULE-008 and AC-004/AC-031 for basic link facet generation without link autocomplete or external calls.

### Step 4: UT-005
- Write failing test: added link punctuation trimming expectations for `See craftsky.social, (https://example.com/path).`.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart`
- Confirmed failure: explicit URL byte range included trailing `).` (`byteEnd` 48 instead of 46).
- Implement: trimmed common trailing sentence punctuation and unmatched closing `)`, `]`, `}` from generated link ranges and URI payloads.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart` — passed (`+4`).
- Refactor: pending after subsequent generator changes.
- Notes: Covers FR-012/RULE-008 and AC-031 for punctuation-safe link ranges and URIs.

### Step 5: UT-018
- Write failing test: added hashtag character/casing expectations for uppercase, Unicode letters/digits, underscore, hyphen, and emoji examples.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart`
- Confirmed failure: no red failure; the prior hashtag parser already matched the UT-018 rules.
- Implement: no additional source changes required.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart` — passed (`+5`).
- Refactor: pending after subsequent generator changes.
- Notes: Covers RULE-003/RULE-007 and AC-033 for preserving typed tag casing and excluding hyphen/emoji from hashtag tokens.

### Step 6: UT-003
- Write failing test: added URL-fragment overlap coverage for `https://craftsky.social/#SockKAL #SockKAL`.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart`
- Confirmed failure: two tag facets were generated, including one inside the URL fragment.
- Implement: sorted generated ranges deterministically and dropped later overlapping generated facets, preserving the earlier/wider URL facet.
- Run command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart` — passed (`+6`).
- Refactor: formatted `facet_generator.dart` and `facet_generator_test.dart`, then re-ran the focused command — passed (`+6`).
- Notes: Covers FR-001/NFR-001/RULE-003 and AC-009/AC-033 for non-overlapping generated facets.

### Step 7: UT-009
- Write failing test: added `app/test/shared/rich_text/faceted_text_model_test.dart` with unsorted, unsupported, multi-feature, overlapping, and out-of-range raw facets.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_model_test.dart`
- Confirmed failure: compilation failed because `faceted_text_model.dart`, `FacetedTextModel`, and `FacetFeatureKind` did not exist.
- Implement: added incoming raw-facet normalization with byte-boundary conversion, first-supported-feature selection, sorting, and safe dropping of malformed/unsupported/overlapping facets.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_model_test.dart` — passed (`+1`).
- Refactor: pending after UT-010.
- Notes: Covers FR-004/FR-005/RULE-009 and AC-013/AC-014/AC-035 for defensive rendering normalization.

### Step 8: UT-010
- Write failing test: added split-multibyte byte range coverage with an invalid range inside an emoji plus one valid tag range.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_model_test.dart`
- Confirmed failure: no red failure; UT-009 implementation already mapped only valid UTF-8 byte boundaries and dropped split ranges.
- Implement: no additional source changes required.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_model_test.dart` — passed (`+2`).
- Refactor: formatted faceted model files and re-ran with span builder after UT-016.
- Notes: Covers FR-004/FR-005/NFR-001/RULE-009 and AC-014/AC-035.

### Step 9: UT-016
- Write failing test: added `app/test/shared/rich_text/faceted_text_span_builder_test.dart` asserting facet spans use the theme primary color while plain spans keep the base color.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_span_builder_test.dart`
- Confirmed failure: compilation failed because `faceted_text_span_builder.dart` and `FacetedTextSpanBuilder` did not exist.
- Implement: added a span builder that splits normalized ranges and applies `facetColor` only to facet spans.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_span_builder_test.dart` — passed (`+1`).
- Refactor: formatted model/span files and ran `cd app && flutter test test/shared/rich_text/faceted_text_model_test.dart test/shared/rich_text/faceted_text_span_builder_test.dart` — passed (`+3`).
- Notes: Covers NFR-005 and AC-002/AC-003/AC-029 for primary-color rich text spans.

### Step 10: UT-011
- Write failing test: added `app/test/shared/rich_text/facet_autocomplete_controller_test.dart` with valid start/whitespace/opening-punctuation tokens and invalid word/email/URL-fragment/bare-trigger cases.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_controller_test.dart`
- Confirmed failure: compilation failed because `facet_autocomplete_controller.dart`, `ActiveFacetToken`, and `ActiveFacetTokenKind` did not exist.
- Implement: added pure token detection for active mention/hashtag tokens with minimum query and token-boundary validation.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_controller_test.dart` — passed (`+1`).
- Notes: Covers FR-006/FR-007/FR-008/RULE-005 and AC-034.

### Step 11: UT-015
- Write failing test: added injectable debounce coverage to the autocomplete controller test.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_controller_test.dart`
- Confirmed failure: `DebouncedFacetLookup` was missing.
- Implement: added `DebouncedFacetLookup<T>` that waits for an injected duration and returns `null` for superseded scheduled lookups.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_controller_test.dart` — passed (`+2`).
- Notes: Covers NFR-002 and AC-015 at the pure helper level; Riverpod provider-family debounce remains for later widget/provider wiring.

### Step 12: UT-020
- Write failing test: added token replacement coverage for a middle mention token and end hashtag token.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_controller_test.dart`
- Confirmed failure: `replaceActiveToken` was missing.
- Implement: added replacement helper that preserves surrounding text, normalizes exactly one trailing space, and moves the caret after the inserted token.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_controller_test.dart` — passed (`+3`).
- Notes: Covers RULE-006 and AC-016/AC-017/AC-030.

### Step 13: UT-012
- Write failing test: added `mock_account_suggestion_repository_test.dart` for Craftsky-only filtering, followed-first sorting, visible fields, and local DID resolution.
- Run command: `cd app && flutter test test/shared/rich_text/mock_account_suggestion_repository_test.dart`
- Confirmed failure: suggestion repository interfaces/models and mock implementation were missing.
- Implement: added account/hashtag suggestion interfaces and mock account repository with local `didForHandle` resolver behavior.
- Run command: `cd app && flutter test test/shared/rich_text/mock_account_suggestion_repository_test.dart` — passed (`+1`).
- Notes: Covers BR-002/FR-010 and AC-005/AC-018/AC-025 without network/AppView/PDS dependencies.

### Step 14: UT-013
- Write failing test: added `mock_hashtag_suggestion_repository_test.dart` for hashtag filtering and 28-day counts.
- Run command: `cd app && flutter test test/shared/rich_text/mock_hashtag_suggestion_repository_test.dart`
- Confirmed failure: no red failure; the UT-012 repository implementation already included the hashtag mock repository.
- Implement: no additional source changes required.
- Run command: `cd app && flutter test test/shared/rich_text/mock_hashtag_suggestion_repository_test.dart` — passed (`+1`).
- Refactor: formatted rich-text source/tests and ran all implemented rich-text unit tests — passed (`+14`).
- Notes: Covers BR-002/FR-009 and AC-006/AC-017.

### Step 15: IT-001
- Write failing test: added `PostApiClient.createPost` coverage for including raw `facets` in the `/v1/posts` body.
- Run command: `cd app && flutter test test/feed/data/post_api_client_test.dart`
- Confirmed failure: `createPost` did not accept a `facets` named parameter.
- Implement: added optional `facets` to `PostApiClient.createPost` and included it in the JSON body when provided.
- Run command: `cd app && flutter test test/feed/data/post_api_client_test.dart` — passed (`+27`).
- Notes: Covers FR-002 and AC-008.

### Step 16: IT-002
- Write failing test: added `ApiPostRepository.create` coverage for forwarding facets through to `PostApiClient`.
- Run command: `cd app && flutter test test/feed/data/post_repository_test.dart`
- Confirmed failure: `PostRepository.create`/`ApiPostRepository.create` did not accept `facets`.
- Implement: added optional `facets` to the post repository interface/implementation and adjusted `FakePostRepository` with a facets-aware callback while preserving existing callback compatibility.
- Run command: `cd app && flutter test test/feed/data/post_repository_test.dart` — passed (`+2`).
- Notes: Covers FR-002 and AC-008/AC-010.

### Step 17: IT-003
- Write failing test: added `CreatePost` provider coverage that captures facets in a fake repository.
- Run command: `cd app && flutter test test/feed/providers/create_post_provider_test.dart`
- Confirmed failure: `CreatePost.create` did not accept `facets`.
- Implement: added optional `facets` to `CreatePost.create` and forwarded them to the repository without changing existing cache mutation behavior.
- Run command: `cd app && flutter test test/feed/providers/create_post_provider_test.dart` — passed (`+11`).
- Notes: Covers FR-002 and AC-010.

### Step 18: IT-004
- Write failing test: added `ProfileApiClient.updateMyProfile` coverage for sending `descriptionFacets` in `/v1/profiles/me`.
- Run command: `cd app && flutter test test/profile/data/profile_api_client_test.dart`
- Confirmed failure: `updateMyProfile` did not accept `descriptionFacets`.
- Implement: added optional `descriptionFacets` through profile API/repository signatures and request body, and updated `FakeProfileRepository` with a facets-aware callback while preserving existing callback compatibility.
- Run command: `cd app && flutter test test/profile/data/profile_api_client_test.dart` — passed (`+8`).
- Notes: Covers FR-003 and AC-011. The live AppView incompatibility remains a known risk; this slice intentionally sends the field.

### Step 19: AT-003
- Write failing test: added `app/test/shared/rich_text/facet_autocomplete_editor_test.dart` for mention suggestions in the shared editor, using provider-overridden mock accounts containing followed/non-followed Craftsky accounts and a non-Craftsky account.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart`
- Confirmed failure: compilation failed because `facet_suggestion_providers.dart` and `FacetAutocompleteEditor` did not exist.
- Implement: added mock-backed suggestion providers and a reusable `FacetAutocompleteEditor` around `BrandTextField`; mention autocomplete debounces lookups, shows display name/handle/avatar semantics, filters via the injected repository, preserves followed-first sort, and inserts `@handle ` on selection while keeping focus.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart` — passed the mention workflow after fixing the avatar semantics assertion/setup (`+1`).
- Notes: Covers BR-002/FR-006/FR-007/FR-010/RULE-006/NFR-003 and AC-005/AC-016/AC-018/AC-025/AC-030 at the shared editor level.

### Step 20: AT-004
- Write failing test: extended `facet_autocomplete_editor_test.dart` for hashtag suggestions with 28-day post counts and canonical-casing insertion.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart`
- Confirmed failure: no hashtag suggestions were shown for `#sock`.
- Implement: added hashtag suggestion state/rendering to `FacetAutocompleteEditor`; hashtag suggestions display `#tag` plus “posts in the last 28 days” counts, hide when empty, and insert `#CanonicalTag ` with exactly one trailing space.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart` — passed (`+2`).
- Notes: Covers BR-002/FR-008/FR-009/RULE-003/RULE-006 and AC-006/AC-017/AC-030/AC-033 at the shared editor level.

### Step 21: AT-001
- Write failing test: added `app/test/feed/widgets/post_composer_sheet_facets_test.dart` for submitting `🧶 Hi @alice.craftsky.social see craftsky.social, #SockKAL` through the post composer with mock resolver data.
- Run command: `cd app && flutter test test/feed/widgets/post_composer_sheet_facets_test.dart`
- Confirmed failure: the composer did not generate or pass raw facet JSON to the fake post repository.
- Implement: wired `PostComposerSheet` to use `FacetAutocompleteEditor`, recompute facets from the final trimmed text via `facetGeneratorProvider`, and pass non-empty facets through `CreatePost.create`.
- Run command: `cd app && flutter test test/feed/widgets/post_composer_sheet_facets_test.dart` — passed (`+1`).
- Notes: Covers BR-001/FR-001/FR-002/FR-012/RULE-002/RULE-003 and AC-001/AC-004/AC-008/AC-009/AC-010/AC-024/AC-031/AC-032/AC-033 for post submit facet generation.

### Step 22: AT-002
- Write failing test: added `app/test/profile/edit_profile_dialog_facets_test.dart` for saving `Knitting with @alice.craftsky.social #Lace` with mock resolver data while preserving full profile save fields.
- Run command: `cd app && flutter test test/profile/edit_profile_dialog_facets_test.dart`
- Confirmed failure: `descriptionFacets` was `null` on the fake profile repository save call.
- Implement: wired the profile bio field to `FacetAutocompleteEditor`, generated `descriptionFacets` from the final trimmed bio via `facetGeneratorProvider`, and passed non-empty facets through `SaveProfile.save` to the profile repository seam.
- Run command: `cd app && flutter test test/profile/edit_profile_dialog_facets_test.dart` — passed (`+1`).
- Nearby command: `cd app && flutter test test/profile/edit_profile_dialog_test.dart test/profile/edit_profile_dialog_facets_test.dart` — passed (`+11`).
- Notes: Covers FR-003/RULE-002/RULE-004 and AC-011/AC-012/AC-024/AC-032. The test comment keeps the known live AppView `descriptionFacets` rejection visible as an accepted Flutter-only compatibility risk.

### Step 23: AT-005
- Write failing test: added `app/test/shared/rich_text/faceted_text_test.dart` for primary-color faceted rendering and invalid-facet fallback.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_test.dart`
- Confirmed failure: compilation failed because `FacetedText` did not exist.
- Implement: added `FacetedText` using the existing render-safe normalizer and span builder.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_test.dart` — passed (`+2`).
- Write failing surface tests: extended `app/test/feed/widgets/post_card_test.dart` and added `app/test/profile/widgets/profile_bio_test.dart` for post-card/profile-bio primary-color facet styling and invalid-facet safety.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_bio_test.dart`
- Confirmed failure: `ProfileBio` did not accept `descriptionFacets`, and `PostCard` still rendered plain `Text` without styled facet spans.
- Implement: rendered `PostCard` body with `FacetedText`, added optional `Profile.descriptionFacets`, passed it through `ProfileMetaSection` to `ProfileBio`, rendered profile bios with `FacetedText`, and regenerated Dart mappers/providers.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_bio_test.dart` — passed (`+32`).
- Nearby command: `cd app && flutter test test/shared/rich_text/faceted_text_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_bio_test.dart test/profile/edit_profile_dialog_facets_test.dart` — passed (`+33`).
- Notes: Covers BR-001/FR-004/FR-005/NFR-001/NFR-005/RULE-009 and AC-002/AC-003/AC-013/AC-014/AC-029/AC-035 for safe themed rendering across shared, post, and profile surfaces.

### Step 24: AT-006 / UT-014 / IT-010
- Write failing test: added `app/test/shared/rich_text/faceted_text_actions_test.dart` for rendered mention, link, hashtag, and failing-link taps.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_actions_test.dart`
- Confirmed failure: compilation failed because `facet_action_providers.dart` and the link-launcher/action seam did not exist.
- Implement: added `FacetActionHandler`, `facetUrlLauncherProvider`, `facetActionHandlerProvider`, recognizer support in `FacetedTextSpanBuilder`, and tap recognizers in `FacetedText`. Mention taps navigate to `/profile/<visible-handle>`, link taps call the injected launcher, hashtag taps navigate to `/search?tag=<tag>`, and destination failures are swallowed safely.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_actions_test.dart` — passed (`+4`).
- Write failing route/search test: extended `app/test/search/search_page_test.dart` for `SearchPage(tag: ...)` and `SearchRoute(tag: ...).location`.
- Run command: `cd app && flutter test test/search/search_page_test.dart`
- Confirmed failure: `SearchPage` and `SearchRoute` did not accept `tag`.
- Implement: added optional `tag` query state to `SearchRoute`, passed it into `SearchPage`, displayed hashtag context without implementing search results, and regenerated router output.
- Run command: `cd app && flutter test test/shared/rich_text/faceted_text_actions_test.dart test/search/search_page_test.dart test/shared/rich_text/faceted_text_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_bio_test.dart` — passed (`+39`).
- Notes: Covers FR-013 and AC-020/AC-026/AC-027/AC-028. Search result implementation remains out of scope; this step only preserves destination context and safe failures.

### Step 25: REG-001..REG-006
- Write failing test: no new failing test was required; regression coverage uses the existing composer/profile/post suites plus the new facet-focused cases added above.
- Run command: `cd app && flutter test test/shared/rich_text test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/feed/providers/create_post_provider_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/widgets/post_card_test.dart test/profile/data/profile_api_client_test.dart test/profile/edit_profile_dialog_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart`
- Confirmed result: passed (`+119`).
- Implement/refactor: no additional behavior changes required after the regression run.
- Final broader command: `cd app && flutter test` — passed (`+486`).
- Notes: Covers NFR-004 / AC-021 / AC-022 and REG-001..REG-006 for existing composer validation/discard behavior, profile edit dirty/save behavior, and post-card rendering stability.

## Current Verification Snapshot
- `cd app && dart analyze lib/shared/rich_text lib/feed/widgets/post_card.dart lib/feed/widgets/post_composer_sheet.dart lib/profile/models/profile.dart lib/profile/pages/edit_profile_dialog.dart lib/profile/providers/save_profile_provider.dart lib/profile/widgets/profile_bio.dart lib/profile/widgets/profile_meta_section.dart lib/router/router.dart lib/search/pages/search_page.dart test/shared/rich_text test/feed/widgets/post_card_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart` — passed with no issues.
- `cd app && dart analyze` — reported 50 existing repository analyzer issues outside the facet change set (unused import in an existing profile model test plus existing lint/info findings in feed/notifications/profile tests and providers); no errors were reported in changed facet files.
- `cd app && flutter test test/shared/rich_text/facet_generator_test.dart test/shared/rich_text/faceted_text_model_test.dart test/shared/rich_text/faceted_text_span_builder_test.dart test/shared/rich_text/facet_autocomplete_controller_test.dart test/shared/rich_text/mock_account_suggestion_repository_test.dart test/shared/rich_text/mock_hashtag_suggestion_repository_test.dart test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/feed/providers/create_post_provider_test.dart test/profile/data/profile_api_client_test.dart` — passed (`+62`).
- `cd app && flutter test test/profile/edit_profile_dialog_test.dart test/profile/edit_profile_dialog_facets_test.dart` — passed (`+11`).
- `cd app && dart run build_runner build --delete-conflicting-outputs` — completed; build_runner warned that `--delete-conflicting-outputs` was ignored by the installed version and that dependency analyzer support lags SDK language version, but generated files were updated successfully.
- `cd app && flutter test test/shared/rich_text/faceted_text_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_bio_test.dart test/profile/edit_profile_dialog_facets_test.dart` — passed (`+33`).
- `cd app && flutter test test/shared/rich_text/faceted_text_actions_test.dart test/search/search_page_test.dart test/shared/rich_text/faceted_text_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_bio_test.dart` — passed (`+39`).
- `cd app && flutter test test/shared/rich_text test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/feed/providers/create_post_provider_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/widgets/post_card_test.dart test/profile/data/profile_api_client_test.dart test/profile/edit_profile_dialog_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart` — passed (`+119`).
- `cd app && flutter test` — passed (`+486`).
- Review-fix command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart` — first failed to compile because `FacetTextEditingController` did not exist; passed after implementation (`+3`).
- Review-fix command: `cd app && flutter test test/profile/data/profile_api_client_test.dart` — passed (`+9`) with current-AppView `unexpected_field` rejection mapped as `ApiBadRequest`.
- Review-fix command: `cd app && flutter test test/profile/edit_profile_dialog_facets_test.dart` — passed (`+2`) with the `descriptionFacets` rejection flowing through the existing save-error snackbar path.
- Review-fix command: `cd app && dart analyze lib/shared/rich_text lib/feed/widgets/post_card.dart lib/feed/widgets/post_composer_sheet.dart lib/profile/models/profile.dart lib/profile/pages/edit_profile_dialog.dart lib/profile/providers/save_profile_provider.dart lib/profile/widgets/profile_bio.dart lib/profile/widgets/profile_meta_section.dart lib/router/router.dart lib/search/pages/search_page.dart test/shared/rich_text test/feed/widgets/post_card_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart` — passed with no issues after fixing two lint/info findings introduced during the review-fix pass.
- Review-fix command: `cd app && flutter test test/shared/rich_text test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/feed/providers/create_post_provider_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/widgets/post_card_test.dart test/profile/data/profile_api_client_test.dart test/profile/edit_profile_dialog_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart` — passed (`+118`).
- Review-fix command: `cd app && flutter test` — passed (`+489`).
- Follow-up styling command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart` — first failed because `@ali` was no longer primary-colored after the caret moved to the end of `Hello @ali and #sock done`; passed after persistent editable token range styling (`+3`).
- Follow-up styling command: `cd app && dart analyze lib/shared/rich_text/widgets/facet_autocomplete_editor.dart test/shared/rich_text/facet_autocomplete_editor_test.dart` — passed with no issues.
- Follow-up styling command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/profile/edit_profile_dialog_facets_test.dart` — passed (`+6`).

## Remaining Planned Work
- None for the requested review-fix pass. `IR-001` and `IR-002` are addressed with passing tests. `IR-003` remains intentionally skipped per explicit user instruction because the current debounce implementation works for this slice.

## Implementation Review Fixes

### Step 26: IR-001 / UT-016 editable primary-color token styling
- Write failing test: added `app/test/shared/rich_text/facet_autocomplete_editor_test.dart` coverage that keeps `@ali` and `#sock` active in the editable field and inspects the editable controller text span color.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart`
- Confirmed failure: compilation failed because `FacetTextEditingController` did not exist.
- Implement: added `FacetTextEditingController` overriding `buildTextSpan` to color the detected active mention/hashtag token with `Theme.colorScheme.primary`; required `FacetAutocompleteEditor` callers to use it; updated post composer/profile bio controllers and shared editor tests.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart` — passed (`+3`).
- Notes: Covers NFR-005 and AC-029 for editable composer/profile token styling through the shared editor and production composer/profile controllers.

### Step 27: IR-002 / IT-006 descriptionFacets rejection path
- Write failing test: added `app/test/profile/data/profile_api_client_test.dart` coverage simulating the current AppView `unexpected_field` 400 response when `descriptionFacets` is present, and added `app/test/profile/edit_profile_dialog_facets_test.dart` coverage where the fake profile repository throws `ApiBadRequest('unexpected_field')` after receiving generated `descriptionFacets`.
- Run command: `cd app && flutter test test/profile/data/profile_api_client_test.dart`
- Confirmed failure: no red failure; existing error mapping already converted the simulated 400 response to `ApiBadRequest('unexpected_field')`.
- Implement: no production code change required for the API-client error mapping.
- Run command: `cd app && flutter test test/profile/data/profile_api_client_test.dart` — passed (`+9`).
- Additional run command: `cd app && flutter test test/profile/edit_profile_dialog_facets_test.dart`
- Confirmed result: passed (`+2`); the existing save-error listener surfaced `Couldn't save your profile.`, stayed on the edit page, and did not add a Flutter compatibility gate that strips `descriptionFacets`.
- Notes: Covers FR-003/RULE-004 and AC-012 for the known current-AppView `descriptionFacets` incompatibility without adding a Flutter compatibility gate. The follow-up backend/API requirement remains visible in the test name/comment.

### Skipped Review Finding: IR-003 debounce/provider deviation
- Status: skipped by explicit user instruction: “apart from the debounce issue as the current implementation works.”
- Linked requirements/tests: NFR-002 / AC-015 / AT-007.
- Notes: No debounce implementation changes are planned in this review-fix pass.

### Step 28: Follow-up / persistent editable facet styling
- User feedback: composer facet text only uses the primary theme color while the caret is inside the token; it should remain primary-colored throughout editing after the caret leaves the token.
- Write failing test: extended `app/test/shared/rich_text/facet_autocomplete_editor_test.dart` to type `Hello @ali and #sock done`, leaving the caret outside both facet tokens, and assert both `@ali` and `#sock` spans still use `Theme.colorScheme.primary`.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart`
- Confirmed failure: `@ali` span color was `null` instead of the theme primary color because `FacetTextEditingController` only styled the active autocomplete token under the caret.
- Implement: changed `FacetTextEditingController.buildTextSpan` to scan the whole editable text for boundary-valid mention and hashtag token ranges, then build primary-colored spans for all detected ranges while preserving normal style for surrounding text.
- Run command: `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart` — passed (`+3`).
- Nearby verification: `cd app && dart analyze lib/shared/rich_text/widgets/facet_autocomplete_editor.dart test/shared/rich_text/facet_autocomplete_editor_test.dart` — passed with no issues; `cd app && flutter test test/shared/rich_text/facet_autocomplete_editor_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/profile/edit_profile_dialog_facets_test.dart` — passed (`+6`).
- Notes: Extends the `IR-001` / `UT-016` / `NFR-005` / `AC-029` fix so editable mention/hashtag facet tokens are styled persistently, not only when they are the active autocomplete token.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [ ] Implementation review pending in the next workflow stage

## Known Gaps / Risks During Implementation
- `descriptionFacets` live profile saves are expected to fail against the current AppView until a future backend/API slice adds support; Flutter must still send them for this slice.
- If `bluesky_text` is introduced, implementation review must verify no external identity/PDS resolution APIs are used. Current plan permits a custom parser if that is smaller and safer for local-only behavior.
