# Requirements: Flutter Facets UI

## 1. Initial Request

Add Flutter UI support for AT Protocol rich-text facets in Craftsky post text, profile descriptions, and the post/profile composer surfaces. The key facet types are mentions, web links, and hashtags. While composing, typing `@` should show debounced Craftsky-only account suggestions from a mock repository, prioritizing followed accounts. Typing `#` should show debounced hashtag suggestions from a mock repository, including a popularity indication such as number of posts with that hashtag in the last 28 days. For this slice, autocomplete data is Flutter-only and mock-backed. The user prefers using atproto.dart ecosystem helpers where appropriate and confirmed that Flutter should pass facet payloads now, including profile `descriptionFacets` despite current AppView incompatibility.

## 2. Current Codebase Findings

- Relevant files:
  - `app/lib/feed/widgets/post_composer_sheet.dart` owns the full-screen post composer. It currently uses `BrandTextField`, stores raw `_text`, and submits `text`, optional `reply`, and optional `images`.
  - `app/lib/feed/widgets/post_card.dart` renders post body with `Text(post.text)` only.
  - `app/lib/feed/models/post.dart` already includes `facets` as `List<Map<String, dynamic>>?` with comments stating typed rich-text rendering can land later.
  - `app/lib/feed/data/post_repository.dart`, `api_post_repository.dart`, `post_api_client.dart`, and `app/test/feed/fakes/fake_post_repository.dart` define the Flutter post repository/API/fake seam.
  - `app/lib/profile/widgets/profile_bio.dart` renders profile descriptions with plain `Text(description)`.
  - `app/lib/profile/pages/edit_profile_dialog.dart` owns profile bio editing through `BrandTextField` and currently saves only `description`.
  - `app/lib/profile/models/profile.dart` currently has `description` but no profile description facet field.
  - `app/lib/profile/data/profile_repository.dart`, `api_profile_repository.dart`, `profile_api_client.dart`, and `app/test/profile/fakes/fake_profile_repository.dart` define the Flutter profile repository/API/fake seam.
  - `app/lib/theme/brand_text_field.dart` wraps `TextField`; any autocomplete/editor reuse must either compose around it or introduce a compatible richer editor surface.
  - `lexicon/social/craftsky/feed/post.json` defines `facets` on Craftsky posts as `app.bsky.richtext.facet` references.
  - `appview/internal/api/post_request.go` and `post.go` already accept and pass through `facets` for `POST /v1/posts`.
  - `appview/internal/api/profile_request.go` currently rejects unknown profile update fields; `descriptionFacets` is not allowed today.
- Existing patterns:
  - Flutter uses Dio API clients, repository interfaces, Riverpod providers, generated providers, `dart_mappable` models, localization, and widget tests with fake repositories.
  - Flutter writes go to AppView; the app must not call the PDS directly or hold PDS tokens.
  - Repository interfaces are production-bound to API implementations and test-bound to fakes.
  - Riverpod provider files use `@riverpod` and require generated `*.g.dart` updates when provider signatures change.
  - UI should use theme values and small widget classes per app-specific Flutter rules.
- Current behavior:
  - Posts and profile descriptions display as plain text.
  - Post composer and profile bio editor do not suggest mentions or hashtags.
  - Post creation does not include generated facets from Flutter, even though AppView can receive them.
  - Profile updates do not include description facets, and the current AppView would reject `descriptionFacets` if Flutter sent it.
- Constraints discovered:
  - This requirements slice is Flutter-side only: no AppView implementation, migrations, lexicon changes, or real autocomplete endpoints.
  - Autocomplete options must come from a mock repository in this slice.
  - Post facets can be sent through the existing AppView contract.
  - Profile description facet writes are an intentional compatibility risk until a follow-up AppView/API slice adds support.
  - AT Protocol facets use UTF-8 byte offsets, inclusive start/exclusive end, and must not overlap.
  - `bluesky_text` is documented by atproto.dart as supporting handle, link, and tag detection plus byte indices/facet generation, but it is not currently in `app/pubspec.yaml`.
  - `bluesky_text` can resolve handles when converting entities to facets; Craftsky Flutter must not use it in a way that bypasses AppView architecture or calls external identity services unexpectedly.
- Test/build commands discovered:
  - Focused Flutter tests run from `app/` with `flutter test <paths>`.
  - Provider/model changes may require `dart run build_runner build --delete-conflicting-outputs` from `app/`.
  - Dependency changes would require `flutter pub get` in the implementation stage.

## 3. Clarifying Questions And Decisions

### Q1: Should Flutter generate and pass facet JSON through repository calls now, or leave network payloads unchanged?

Answer: Pass facets now.

Decision / implication: Post creation shall send `facets` when generated. Profile save shall be planned to send profile description facets as well, subject to the compatibility decision in Q2.

### Q2: How should profile-description facets be handled given the current AppView rejects `descriptionFacets`?

Answer: Risk live send.

Decision / implication: Flutter shall be specified to send `descriptionFacets` with profile updates in this slice, with a clearly documented risk that live profile saves require a near-term AppView follow-up before they are usable.

### Q3: Which implementation direction should requirements assume?

Answer: Option A recommended.

Decision / implication: Requirements shall assume a shared Flutter rich-text/facet module, reusable composer/profile autocomplete behavior, mock suggestion repositories/providers, and use of `bluesky_text` or equivalent atproto.dart helper functionality for byte-safe parsing where appropriate.

### Q4: Should profile `descriptionFacets` be gated until AppView supports them?

Answer: No. Keep the intentionally risky live send.

Decision / implication: Flutter shall send `descriptionFacets` as planned. Live profile-save failures from the current AppView rejecting unknown fields remain an accepted, documented compatibility risk rather than a reason to add a temporary Flutter-side compatibility gate.

### Q5: Should manually typed handles generate mention facets if they resolve locally?

Answer: Yes.

Decision / implication: Submit/save-time facet generation shall resolve all syntactically valid `@handle` tokens in the current text against injected/mock Craftsky resolver data, not only handles selected from autocomplete. Unknown handles remain visible plain text with no mention facet and no warning.

### Q6: What should selecting a mention insert?

Answer: Handle only.

Decision / implication: Mention suggestions shall insert `@handle` exactly, not display names or hidden-only metadata.

### Q7: Should autocomplete selection insert a trailing separator?

Answer: Yes, insert a trailing space.

Decision / implication: Mention and hashtag selections shall append a trailing space so the user can continue typing naturally.

### Q8: When should `@` and `#` autocomplete activate?

Answer: Only at token boundaries.

Decision / implication: Autocomplete shall not activate inside words, email addresses, or URL fragments. It may activate at the start of text or after whitespace/opening punctuation such as `(`.

### Q9: Should hashtag facet values preserve casing?

Answer: Yes.

Decision / implication: Hashtag facet `tag` values shall preserve the user's typed casing while excluding the leading `#`. Future search/browse behavior may normalize internally.

### Q10: Which URL forms should link facets recognize?

Answer: Explicit `http://` and `https://` URLs plus bare domains.

Decision / implication: Bare domains shall be faceted and opened as HTTPS targets.

### Q11: How should bare-domain link facet URI payloads be stored?

Answer: Normalize to `https://...` during facet generation.

Decision / implication: Visible text may remain `craftsky.social`, but the link facet URI shall be `https://craftsky.social`.

### Q12: Should trailing punctuation be included in link facets?

Answer: No.

Decision / implication: Common trailing sentence punctuation and unmatched closing brackets/parentheses shall be excluded from the link facet range and URI.

### Q13: What characters are valid in hashtags for this slice?

Answer: Unicode letters, Unicode digits, and underscore only.

Decision / implication: Hashtags shall not include hyphens or emoji in this slice.

### Q14: What casing should hashtag suggestion insertion use?

Answer: Use the suggestion's display/canonical casing.

Decision / implication: Selecting a hashtag suggestion shall insert the tag exactly as returned by the mock hashtag repository, followed by a trailing space. Manually typed hashtags still preserve the user's typed casing in generated facets.

### Q15: Should mention and hashtag empty-result behavior differ?

Answer: Yes.

Decision / implication: Mention suggestions with no matches shall show a `No results` state because unresolved profile mentions matter. Hashtag suggestions with no matches shall hide the dropdown because users can freely create/use varied hashtags.

### Q16: Should unknown manually typed mentions warn before submit/save?

Answer: No.

Decision / implication: Unknown mentions shall remain plain text and shall not block or warn during submit/save.

### Q17: What minimum query length should autocomplete use?

Answer: At least one character after `@` or `#`.

Decision / implication: Bare `@` or `#` shall not query or show suggestions.

### Q18: What debounce duration should autocomplete use?

Answer: 300 ms.

Decision / implication: Mention and hashtag suggestions shall use a 300 ms default debounce delay that is injectable/testable.

### Q19: How should hashtag facet taps navigate?

Answer: Navigate to search with hashtag context.

Decision / implication: Tapping a hashtag facet shall navigate to the search route with the tag context, for example `/search?tag=SockKAL`, even though search results remain out of scope.

### Q20: How should mention facet taps resolve navigation?

Answer: Use the visible handle.

Decision / implication: Tapping a mention facet shall navigate using the visible syntactically valid `@handle` text. The DID remains facet metadata. If no usable visible handle exists, the tap shall fail safely without crashing.

### Q21: How should unknown or unusual incoming facet metadata render?

Answer: Tolerate it safely.

Decision / implication: The renderer shall ignore unsupported/unknown facet feature variants, use the first supported feature in incoming feature order when multiple supported features share a range, and drop only the invalid facet when malformed byte offsets point into a UTF-8 character.

## 4. Candidate Approaches

### Option A: Shared rich-text/facet module plus reusable autocomplete editor

Summary: Add a Flutter rich-text/facet layer that uses `bluesky_text` for byte-safe entity detection where appropriate, wraps Craftsky-compatible raw facet JSON, renders facet-aware text, and provides reusable mention/hashtag autocomplete for post composer and profile bio editing.

Pros:

- Aligns with AT Protocol byte-offset rules without hand-rolling Unicode indexing.
- Avoids duplicating parser/autocomplete behavior between post composer and profile bio editor.
- Keeps autocomplete data behind mock repository/provider seams that can later swap to AppView endpoints.
- Fits the existing repository/fake testing pattern.
- Allows post facets to work against the current AppView contract.

Cons:

- Adds a dependency in implementation.
- Requires careful use of `bluesky_text` to avoid unintended external handle-resolution calls from Flutter.
- Profile `descriptionFacets` writes are intentionally ahead of current AppView support.

Risks: Medium-high due to dependency/payload changes and known profile API incompatibility.

### Option B: Custom parser plus reusable autocomplete editor

Summary: Hand-roll mention, link, and hashtag detection/facet JSON generation in Flutter, while still sharing autocomplete and rendering components.

Pros:

- Avoids adding a new package.
- Gives full control over Craftsky-specific parsing behavior.

Cons:

- Higher likelihood of UTF-8 byte-offset, grapheme, punctuation, and overlap bugs.
- Duplicates ecosystem work already provided by atproto.dart helpers.
- More difficult for test design to cover comprehensively.

Risks: Medium-high due to protocol-format correctness risk.

### Option C: UI-first only, postpone payload changes

Summary: Add autocomplete and local rendering only, leaving repository/API payloads unchanged until AppView support is complete.

Pros:

- Lowest live compatibility risk.
- Smaller first implementation slice.

Cons:

- Does not satisfy the confirmed decision to pass facets now.
- Does not validate the post write path that AppView already supports.
- Delays meaningful end-to-end facet behavior.

Risks: Medium due to incomplete behavior relative to the user request.

## 5. Recommended Direction

Recommended approach: Option A, with explicit compatibility risk tracking for profile `descriptionFacets`.

Why: It best matches the requested user experience, uses the existing atproto.dart ecosystem for byte-safe parsing, fits Craftsky's Flutter repository/provider/test seams, and lets post facets work now while preparing profile facets for the near-term AppView/API follow-up.

## 6. Problem / Opportunity

Craftsky text currently loses the social affordances users expect from an AT Protocol social app: mentions are not structured, links are not clickable/styled from facets, and hashtags are not discoverable at compose time. Adding facets to Flutter makes posts and profile descriptions more expressive, creates a typed seam for future AppView autocomplete endpoints, and preserves AT Protocol-compatible rich-text payloads at write time.

## 7. Goals

- G-001: Let users compose Craftsky posts and profile bios with mention, web-link, and hashtag facets.
- G-002: Help users discover Craftsky accounts and popular hashtags while typing through debounced autocomplete.
- G-003: Render facet-aware post text and profile descriptions in the Flutter UI.
- G-004: Keep this slice Flutter-focused while making future AppView autocomplete and profile facet support additive.

## 8. Non-Goals

- NG-001: Do not implement AppView autocomplete endpoints.
- NG-002: Do not implement AppView support for profile `descriptionFacets` in this slice.
- NG-003: Do not change lexicon files.
- NG-004: Do not add migrations or modify AppView persistence.
- NG-005: Do not implement hashtag search/browse pages or mention notification delivery.
- NG-006: Do not implement website preview cards or external embed generation for links.
- NG-007: Do not call PDS or external atproto identity services directly from Flutter for autocomplete or handle resolution.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in Craftsky user | A user composing posts, replies, or editing their profile. | Needs fast mention/hashtag suggestions and valid facets in submitted content. |
| Feed/profile reader | A user reading post cards, threads, profile headers, and profile bios. | Needs recognizable and tappable rich-text affordances where facet metadata exists. |
| Future AppView implementer | Developer replacing mock autocomplete/profile facet gaps with real backend support. | Needs stable Flutter repository/provider seams and documented payload expectations. |
| Test designer | Workflow agent/person writing `02-acceptance-tests.md`. | Needs traceable requirements and compatibility risks. |

## 10. Current Behavior

Post text and profile descriptions are plain strings in the Flutter UI. The post composer and profile bio editor accept raw text only and provide no mention or hashtag suggestions. Post creation does not generate or send facets from Flutter. Profile updates do not send description facet metadata. The current AppView post create endpoint can accept `facets`, but the current profile update endpoint rejects unknown fields such as `descriptionFacets`.

## 11. Desired Behavior

Flutter should provide a shared rich-text/facet experience for posts and profile descriptions. While composing post text or editing a profile bio, users should see 300 ms debounced autocomplete suggestions for active `@` and `#` tokens from mock repositories once at least one query character exists after the trigger and the token is at a valid boundary. Mention suggestions should include Craftsky-only accounts, prioritize followed accounts, show avatar/display name/handle, and show `No results` when there are no matches. Hashtag suggestions should include popularity counts for the last 28 days and hide the dropdown when there are no matches. Selecting a mention should insert `@handle `, and selecting a hashtag should insert the repository's display/canonical `#tag ` casing. Faceted text should use the theme's primary color in editable composer/profile fields and rendered post/profile surfaces. On submit/save, Flutter should generate valid AT Protocol facet JSON for recognized mentions, links, and hashtags in the current text and pass it through repository/API calls. Mention facets should be generated for syntactically valid handles that resolve through injected/mock Craftsky data, even if the user typed them manually; unknown handles remain plain text with no warning. Link facets should recognize explicit `http://`/`https://` URLs and bare domains, normalize bare-domain facet URIs to HTTPS, and trim trailing sentence punctuation. Hashtag facets should preserve typed casing, exclude the leading `#`, and use Unicode letters/digits/underscore only. Tapped mention, link, and hashtag facets should navigate to profile pages using the visible handle, open with `url_launcher`, and navigate to the search page with hashtag context respectively. Post facets should be compatible with the existing AppView post create contract. Profile `descriptionFacets` should be sent by Flutter by design, with the known live incompatibility documented for a follow-up AppView/API slice.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky users shall be able to author and view mention, web-link, and hashtag affordances in post text and profile descriptions. | Rich text is a core social affordance and the requested feature. | Prompt | AC-001, AC-002, AC-003, AC-004 |
| BR-002 | Business | Must | The Flutter slice shall use mock-backed autocomplete data rather than live AppView/PDS search. | Keeps this slice Flutter-only while preserving a backend seam. | Prompt | AC-005, AC-006, AC-007 |
| FR-001 | Functional | Must | The system shall detect and generate AT Protocol-compatible facet JSON for mentions, web links, and hashtags in post composer text. | Post writes need structured facets, and AppView already accepts `facets`. | Prompt, Codebase | AC-001, AC-008, AC-009 |
| FR-002 | Functional | Must | The post create flow shall pass generated post `facets` through the Flutter post repository/API/fake seams when submitting a post or reply. | Confirms the user's decision to pass facets now. | User answer Q1 | AC-008, AC-010 |
| FR-003 | Functional | Must | The profile edit flow shall detect and generate profile-description facets and pass them as `descriptionFacets` through the Flutter profile repository/API/fake seams when saving a bio. | Confirms the user's decision to risk live profile facet sends. | User answer Q2 | AC-011, AC-012 |
| FR-004 | Functional | Must | The system shall render post text with facet-aware styling and interactions when post facet metadata is available, while safely rendering plain text when facets are absent or invalid. Faceted post-card text ranges shall use the theme's primary color. | Existing `Post.facets` is raw JSON and post text currently renders plain. | Codebase, Review feedback | AC-002, AC-013, AC-014 |
| FR-005 | Functional | Must | The system shall render profile descriptions with facet-aware styling and interactions when profile description facet metadata is available, while safely rendering plain text when facets are absent or invalid. Faceted profile-description text ranges shall use the theme's primary color. | Profile descriptions are in scope but current profile model lacks facet metadata. | Prompt, Codebase, Review feedback | AC-003, AC-013, AC-014 |
| FR-006 | Functional | Must | The post composer shall show mention autocomplete when the caret is in an active `@` token and shall allow selecting a suggestion to replace the active token with `@handle ` using the selected account's handle. | Supports the requested post composer mention flow and confirmed insertion behavior. | Prompt, Grill Q6-Q7 | AC-005, AC-015, AC-016, AC-030 |
| FR-007 | Functional | Must | The profile bio editor shall show mention autocomplete when the caret is in an active `@` token and shall allow selecting a suggestion to replace the active token with `@handle ` using the selected account's handle. | User confirmed profile descriptions are in scope and mentioned profile mention debouncing; grill confirmed insertion behavior. | Prompt, Grill Q6-Q7 | AC-005, AC-015, AC-016, AC-030 |
| FR-008 | Functional | Must | The post composer and profile bio editor shall show hashtag autocomplete when the caret is in an active `#` token and shall allow selecting a suggestion to replace the active token with the selected hashtag's repository display/canonical casing plus a trailing space. | Supports requested hashtag discovery in text-entry surfaces and confirmed hashtag insertion behavior. | Prompt, Grill Q7, Grill Q14 | AC-006, AC-015, AC-017, AC-030 |
| FR-009 | Functional | Must | Hashtag autocomplete suggestions shall display an indication of popularity as post count over the last 28 days. | User requested popularity indication and suggested the 28-day count. | Prompt | AC-006, AC-017 |
| FR-010 | Functional | Must | Mention autocomplete suggestions shall include only mock accounts marked as Craftsky accounts, shall sort followed accounts ahead of otherwise matching accounts, and shall display each suggested profile's avatar, display name, and handle. | User requested Craftsky-only accounts with followed accounts prioritized; review feedback added visible suggestion fields. | Prompt, Review feedback | AC-005, AC-018, AC-025 |
| FR-011 | Functional | Must | Autocomplete repositories/providers shall be mock-backed in this slice and shaped so production AppView-backed repositories can replace them later without changing editor behavior. | Keeps the slice Flutter-only while avoiding throwaway UI. | Prompt, Discovery | AC-007, AC-019 |
| FR-012 | Functional | Must | Link facets shall be generated for recognized explicit `http://`/`https://` URLs and bare domains without showing an autocomplete dropdown; bare-domain facet URIs shall normalize to HTTPS. | Links are facets but do not require suggestion UI; grill confirmed bare-domain support and URI normalization. | Prompt, AT Protocol docs, Grill Q10-Q12 | AC-004, AC-009, AC-031 |
| FR-013 | Functional | Must | Tapping rendered facets shall invoke the appropriate destination behavior: mentions navigate to the profile page using the visible handle, links open with `url_launcher`, and hashtags navigate to the search page with hashtag context even though hashtag search results are not implemented yet. | Review feedback specified concrete destination behavior; grill confirmed mention and hashtag navigation details. | Review feedback, Grill Q19-Q20 | AC-020, AC-026, AC-027, AC-028 |
| NFR-001 | Non-functional | Must | Facet generation shall use UTF-8 byte offsets and avoid overlapping facet ranges. | AT Protocol facets require byte-index correctness. | AT Protocol docs | AC-009, AC-014 |
| NFR-002 | Non-functional | Must | Mention and hashtag suggestion lookups shall use a 300 ms default debounce delay, injectable/testable in tests, before querying the mock repository. | User explicitly requested debouncing and confirmed 300 ms. | Prompt, Grill Q18 | AC-015 |
| NFR-003 | Non-functional | Should | The autocomplete dropdown should be keyboard- and screen-reader-friendly enough for widget tests to identify options by visible text/semantics. | Composer/profile editors are core input flows and should remain accessible. | Discovery | AC-016, AC-017 |
| NFR-004 | Non-functional | Must | The feature shall not regress existing post composer validation, image attachment, reply, discard-confirmation, or profile-save dirty/validation behavior. | Existing composer/profile flows have tests and should remain stable. | Codebase | AC-021, AC-022 |
| NFR-005 | Non-functional | Must | Faceted text ranges in both editable composer/profile fields and rendered post/profile surfaces shall use the theme's primary color. | Review feedback specified the visual treatment. | Review feedback | AC-002, AC-003, AC-029 |
| RULE-001 | Business rule | Must | Flutter shall not call a PDS or external atproto identity service directly for autocomplete or mention resolution in this slice. | Preserves Craftsky architecture: Flutter talks to AppView and only uses mock data here. | AGENTS.md, Prompt | AC-007, AC-023 |
| RULE-002 | Business rule | Must | A mention facet shall only be generated when the mentioned handle can be mapped to a DID by the local/mock suggestion data or another injected Craftsky resolver seam in Flutter tests; submit/save-time generation shall resolve all syntactically valid `@handle` tokens in the current text, not only selected suggestions. | Mention facets require DIDs; grill confirmed manual resolvable handles should facet and unknown typed handles should remain plain text. | AT Protocol docs, Discovery, Grill Q5, Grill Q16 | AC-009, AC-024, AC-032 |
| RULE-003 | Business rule | Must | Hashtag facet `tag` values shall exclude the leading `#` while preserving the displayed text and the user's typed casing in the user-entered body. | Matches `app.bsky.richtext.facet#tag` shape and grill confirmed casing preservation. | AT Protocol docs, Grill Q9 | AC-009, AC-017, AC-033 |
| RULE-004 | Business rule | Must | If profile `descriptionFacets` causes live AppView saves to fail before backend support lands, that failure is accepted as a known compatibility risk of this slice and must remain visible in documentation and tests. | User explicitly chose to risk live send. | User answer Q2 | AC-012 |
| RULE-005 | Business rule | Must | Mention and hashtag autocomplete shall activate only at token boundaries, not inside words, email addresses, or URL fragments, and only after at least one character has been typed after `@` or `#`. | Avoids noisy or incorrect suggestions and matches confirmed activation behavior. | Grill Q8, Grill Q17 | AC-034 |
| RULE-006 | Business rule | Must | Selecting a mention suggestion shall insert only `@handle `, and selecting a hashtag suggestion shall insert the suggestion's display/canonical `#tag ` casing. | Keeps visible text transparent and supports natural continued typing. | Grill Q6-Q7, Grill Q14 | AC-016, AC-017, AC-030 |
| RULE-007 | Business rule | Must | Hashtag parsing shall treat Unicode letters, Unicode digits, and underscore as hashtag characters; hyphens and emoji shall not be part of generated hashtag facets in this slice. | Supports international tags while bounding parser scope. | Grill Q13 | AC-033 |
| RULE-008 | Business rule | Must | Link parsing shall include explicit HTTP/HTTPS URLs and bare domains, normalize bare domains to `https://` in facet URI payloads, exclude common trailing sentence punctuation and unmatched closing brackets/parentheses from link ranges/URIs, and not support markdown-style links in this slice. | Produces valid link targets while avoiding false positives and broken links. | Grill Q10-Q12 | AC-031 |
| RULE-009 | Business rule | Must | Rich-text rendering shall ignore unsupported/unknown facet feature variants, use the first supported feature in incoming feature order when multiple supported features share one range, and drop only invalid facets whose byte ranges cannot be mapped to valid UTF-8 character boundaries. | Keeps rendering resilient to malformed, future, or unusual incoming facet metadata. | Grill Q21 | AC-014, AC-035 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given a user types post text containing a selected mention, a web link, and a hashtag, when the post is prepared for submission, then generated facets include mention, link, and tag entries for the correct text ranges. |
| AC-002 | BR-001, FR-004, NFR-005 | Given a post contains valid facets, when the post card renders, then faceted text ranges are visually distinguishable from non-faceted text and use the theme's primary color. |
| AC-003 | BR-001, FR-005, NFR-005 | Given a profile description contains valid description facet metadata, when the profile bio renders, then faceted text ranges are visually distinguishable from non-faceted text and use the theme's primary color. |
| AC-004 | BR-001, FR-012 | Given text contains a recognized explicit HTTP/HTTPS URL or bare domain, when facets are generated, then a link facet is included and no link autocomplete dropdown is shown. |
| AC-005 | BR-002, FR-006, FR-007, FR-010 | Given the caret is in an active `@` token, when the debounce period completes, then the editor shows matching mock Craftsky account suggestions with followed accounts before non-followed matches. |
| AC-006 | BR-002, FR-008, FR-009 | Given the caret is in an active `#` token, when the debounce period completes, then the editor shows matching mock hashtag suggestions with each suggestion displaying a 28-day post count. |
| AC-007 | BR-002, FR-011, RULE-001 | Given autocomplete is used in this slice, when suggestions are requested, then they come from mock/injected Flutter repositories/providers and not from live AppView, PDS, or external identity services. |
| AC-008 | FR-001, FR-002 | Given generated post facets are non-empty, when Flutter calls the post create API client, then the JSON body includes `facets` alongside existing `text`, `reply`, and `images` fields as applicable. |
| AC-009 | FR-001, FR-012, NFR-001, RULE-002, RULE-003 | Given text includes multibyte characters or emoji before a mention, link, or hashtag, when facets are generated, then each facet uses valid UTF-8 byte offsets, no generated facets overlap, mention facets include DIDs only for resolved Craftsky accounts, and tag facets omit the leading `#` in the `tag` value while preserving casing. |
| AC-010 | FR-002 | Given a post repository fake captures create calls, when a post with generated facets is submitted in a widget/provider test, then the fake receives the same facet JSON that the API client would send. |
| AC-011 | FR-003 | Given generated profile description facets are non-empty, when Flutter saves the profile, then the profile repository/API/fake seam receives `descriptionFacets` alongside existing display name, description, crafts, avatar, and banner values. |
| AC-012 | FR-003, RULE-004 | Given the current AppView profile endpoint does not support `descriptionFacets`, when documenting or testing the production API client behavior, then the known live-save incompatibility is explicitly recorded and not treated as an accidental regression in this Flutter-only slice. |
| AC-013 | FR-004, FR-005 | Given facet metadata is missing, empty, malformed, unsupported, or references out-of-range text, when post text or profile bio renders, then the UI falls back to safe plain-text rendering without throwing. |
| AC-014 | FR-004, FR-005, NFR-001, RULE-009 | Given facets overlap, are unsorted, have unsupported feature variants, or include invalid ranges, when rich text renders, then the renderer sorts valid ranges and ignores/handles invalid or unsupported ranges safely. |
| AC-015 | FR-006, FR-007, FR-008, NFR-002 | Given the user types or edits an active `@` or `#` token repeatedly, when changes occur faster than 300 ms, then the mock repository is queried only after the user pauses for the configured debounce period. |
| AC-016 | FR-006, FR-007, RULE-006, NFR-003 | Given mention suggestions are visible, when the user selects a suggestion by tap or keyboard-equivalent test action, then the active token is replaced with `@handle `, the editor remains focused, and the caret is after the inserted trailing space. |
| AC-017 | FR-008, FR-009, RULE-003, RULE-006, NFR-003 | Given hashtag suggestions are visible, when the user selects a suggestion, then the active token is replaced with the suggestion's display/canonical `#tag ` casing, the popularity indicator was visible before selection, and generated tag facets store the tag without the `#`. |
| AC-018 | FR-010 | Given mock suggestions include non-Craftsky accounts and followed/non-followed Craftsky accounts, when an account query runs, then non-Craftsky accounts are excluded and followed matching accounts sort before non-followed matching accounts. |
| AC-019 | FR-011 | Given tests override the autocomplete repositories/providers, when composer/profile editors request suggestions, then the UI uses the override without changing editor widget code. |
| AC-020 | FR-013 | Given a rendered mention facet is tapped, when the visible faceted text contains a syntactically valid `@handle`, then Flutter navigates to that user's profile page using the visible handle. |
| AC-021 | NFR-004 | Given existing post composer flows for empty text, overlong text, replies, images, alt-text warning, and discard confirmation, when facet UI is added, then those behaviors continue to pass existing expectations. |
| AC-022 | NFR-004 | Given existing profile edit flows for seeded values, dirty state, validation, image drafts, and successful saves, when facet UI is added, then those behaviors continue to pass existing expectations except for the documented live `descriptionFacets` backend incompatibility. |
| AC-023 | RULE-001 | Given a test environment without network access, when facet autocomplete and facet generation are exercised with mock data, then tests can pass without external network calls. |
| AC-024 | RULE-002 | Given a user manually types an `@unknown.example` handle that is not present in mock/injected Craftsky resolver data, when facets are generated and the post/profile is submitted or saved, then no mention facet is generated for that unknown handle, the text remains unchanged, and no warning blocks the action. |
| AC-025 | FR-010 | Given mention suggestions are visible, when the dropdown renders, then each suggestion displays the suggested profile's avatar, display name, and handle. |
| AC-026 | FR-013 | Given a rendered link facet is tapped, when the URL is valid, then Flutter opens it using the existing `url_launcher` package. |
| AC-027 | FR-013 | Given a rendered hashtag facet is tapped, when the search route is available, then Flutter navigates to the search page with the hashtag context, for example `/search?tag=SockKAL`, even though hashtag search results are not implemented yet. |
| AC-028 | FR-013 | Given a rendered facet is tapped and its destination action cannot complete, when the failure occurs, then the app does not crash. |
| AC-029 | NFR-005 | Given a mention or hashtag is active in the post composer or profile bio editor, when the text is displayed in the editable field, then the faceted token uses the theme's primary color. |
| AC-030 | FR-006, FR-007, FR-008, RULE-006 | Given a mention or hashtag suggestion replaces a partial token in the middle or end of text, when the replacement completes, then only the active token is replaced, the selected value includes one trailing space, and surrounding text is preserved. |
| AC-031 | FR-012, RULE-008 | Given text contains a bare domain such as `craftsky.social` followed by punctuation, when link facets are generated, then the visible text range excludes trailing punctuation and the facet URI is normalized to `https://craftsky.social`. |
| AC-032 | RULE-002 | Given a user manually types a syntactically valid `@handle` that exists in injected/mock Craftsky resolver data without selecting it from autocomplete, when facets are generated at submit/save time, then a mention facet with that handle's DID is included. |
| AC-033 | RULE-003, RULE-007 | Given hashtags contain uppercase letters, Unicode letters/digits, underscores, hyphens, or emoji, when facets are generated, then valid Unicode letter/digit/underscore runs become tag facets preserving typed casing and hyphens/emoji are excluded from the hashtag token in this slice. |
| AC-034 | RULE-005 | Given `@` or `#` appears in a word, email address, URL fragment, or without at least one following query character, when the caret is in that text, then autocomplete does not query or display suggestions; given the trigger appears at text start or after whitespace/opening punctuation with at least one query character, autocomplete may query after debounce. |
| AC-035 | RULE-009 | Given incoming facet metadata contains multiple supported features on one range, an unknown feature variant, or a byte range that points into the middle of a multibyte character, when rich text renders, then the first supported feature in incoming order is used for multi-feature ranges, unknown features are ignored, invalid ranges are dropped, and other valid facets still render. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | User types `@` or `#` in the middle of a word, email address, URL fragment, or without a following query character. | Autocomplete should not activate unless the token boundary rules identify an active mention/hashtag token with at least one character after the trigger. | FR-006, FR-008, RULE-005 |
| EC-002 | User moves the caret away from an active token while a debounce timer is pending. | Pending suggestions should be ignored or hidden if they no longer match the active token/caret position. | FR-006, FR-008, NFR-002 |
| EC-003 | Mock account repository returns an empty suggestion list. | Mention dropdown should display `No results` without blocking typing, because unresolved profile mentions benefit from validation feedback. | FR-006, FR-011 |
| EC-011 | Mock hashtag repository returns an empty suggestion list. | Hashtag dropdown should not be shown, because users can freely use varied/new hashtags. | FR-008, FR-011 |
| EC-004 | Selected mention/hashtag replaces a partial token such as `@ali` or `#vog`. | Only the active token is replaced; surrounding text and selection are preserved; the inserted token includes a trailing space. | FR-006, FR-008, RULE-006 |
| EC-005 | Text contains emoji or non-Latin characters before a facet. | Facet byte offsets remain correct. | NFR-001 |
| EC-006 | Text contains overlapping detected entities, such as a hashtag inside a URL fragment. | Generated/rendered facets avoid overlaps; link handling should not create invalid overlapping tag facets or activate hashtag autocomplete inside the URL fragment. | NFR-001, RULE-005 |
| EC-007 | User edits text after selecting a mention, making the selected range no longer match the selected handle. | Submission-time generation should recompute facets from the current text and resolver state rather than trusting stale ranges. | FR-001, RULE-002 |
| EC-008 | Existing posts have `facets: null`, missing, or malformed raw JSON from old/future records. | Post rendering falls back safely to plain text. | FR-004 |
| EC-009 | Profile response lacks `descriptionFacets` because AppView does not yet provide it. | Profile bio renders safely as plain text. | FR-005 |
| EC-010 | Live profile save sends `descriptionFacets` before AppView support lands. | The failure is documented as expected risk; Flutter error handling should not crash. | FR-003, RULE-004 |
| EC-012 | User manually types a resolvable handle without selecting autocomplete. | Submit/save-time generation should create a mention facet if the handle resolves through injected/mock Craftsky data. | RULE-002 |
| EC-013 | User manually types an unknown handle. | Text should submit/save unchanged as plain text with no mention facet and no blocking warning. | RULE-002 |
| EC-014 | Text contains a bare domain followed by sentence punctuation or an unmatched closing parenthesis/bracket. | The link facet range and URI should exclude trailing punctuation/closers and normalize bare domains to HTTPS. | FR-012, RULE-008 |
| EC-015 | Text contains uppercase, Unicode, hyphenated, or emoji hashtags. | Unicode letters/digits/underscore are included in tag facets with casing preserved; hyphens and emoji are excluded from hashtag tokens in this slice. | RULE-003, RULE-007 |
| EC-016 | Incoming facet has unknown feature variants or multiple supported features on one range. | Unknown features are ignored, and the first supported feature in incoming order controls styling/tap behavior for that range. | RULE-009 |
| EC-017 | Incoming facet byte range points into the middle of a multibyte character. | Only the invalid facet is dropped; other valid facets still render and the app does not crash. | RULE-009 |

## 15. Data / Persistence Impact

- New fields:
  - Flutter post create request path shall include optional `facets` when generated.
  - Flutter profile update request path shall include optional `descriptionFacets` when generated.
  - Flutter profile model should be prepared to carry optional profile description facet metadata if/when AppView returns it.
  - Flutter-only models are expected for mention suggestions, hashtag suggestions, and raw/typed facet handling.
  - Search route state should be prepared to carry optional hashtag context, such as a `tag` query parameter, for hashtag facet taps.
- Changed fields:
  - Existing post/profile text fields remain plain text; facets annotate ranges rather than changing text content.
- Migration required:
  - None in this Flutter-only slice.
- Backwards compatibility:
  - Post `facets` are compatible with the current AppView post create contract.
  - Profile `descriptionFacets` are intentionally not compatible with the current AppView profile update contract and require a follow-up backend/API slice.

## 16. UI / API / CLI Impact

- UI:
  - Post composer gains mention and hashtag autocomplete dropdowns.
  - Profile bio editor gains mention and hashtag autocomplete dropdowns.
  - Mention suggestions display avatar, display name, and handle.
  - Mention no-match autocomplete displays `No results`; hashtag no-match autocomplete hides the dropdown.
  - Mention and hashtag suggestion selection inserts a trailing space after the selected token.
  - Post cards render facet-aware text when facet metadata is available.
  - Profile bio renders facet-aware text when facet metadata is available.
  - Faceted text uses the theme's primary color in composer/profile editors and rendered post/profile text.
  - Rendered mention facets navigate to profile pages using visible handles, link facets open through `url_launcher`, and hashtag facets navigate to the search page with tag context.
- API:
  - Flutter `POST /v1/posts` request body gains optional `facets` from the client.
  - Flutter `PUT /v1/profiles/me` request body gains optional `descriptionFacets` from the client, ahead of current AppView support.
  - Flutter search routing gains optional hashtag context for navigation, for example `/search?tag=SockKAL`; search result implementation remains out of scope.
  - No AppView route implementation changes are in scope for this slice.
- CLI:
  - None.
- Background jobs:
  - None.

## 17. Security / Privacy / Permissions

- Authentication:
  - Existing authenticated post/profile write requirements remain unchanged.
- Authorization:
  - No new authorization rules in Flutter.
- Sensitive data:
  - Mock autocomplete data must not imply access to private account data; it should represent public Craftsky account/hashtag suggestions only.
- Abuse cases:
  - Autocomplete should not include non-Craftsky accounts in this slice.
  - Flutter must not directly query external identity services or PDSes for mention resolution.
  - Unknown manually typed handles should not produce invalid mention facets.

## 18. Observability

- Events:
  - None required for this slice.
- Logs:
  - No diagnostic logs are required. If implementation adds logs, they must not include full draft text or tokens.
- Metrics:
  - None required for this slice.
- Alerts:
  - None.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Profile `descriptionFacets` are sent before AppView supports them. | Live profile saves can fail with `unexpected_field` until backend support lands. | Document as intentional; require follow-up AppView/API slice; ensure Flutter handles save errors without crashing. |
| RISK-002 | Incorrect UTF-8 byte offsets or overlapping facets. | PDS/AppView may reject records or render incorrect rich text. | Use `bluesky_text`/byte-safe helpers where appropriate; add tests with emoji/multibyte text and overlap cases. |
| RISK-003 | `bluesky_text` facet conversion may resolve handles externally if used naively. | Flutter could bypass Craftsky architecture and make unexpected network calls. | Use entity detection/byte indices with Craftsky mock resolver data; prohibit direct external identity calls in requirements/tests. |
| RISK-004 | Rich editor/autocomplete refactor could regress existing composer/profile behavior. | Users may lose draft/discard, image, reply, validation, or profile dirty-state behavior. | Keep existing behaviors under regression tests; prefer reusable wrapper components over broad rewrites. |
| RISK-005 | Mock autocomplete data may shape UI assumptions that do not match future AppView endpoints. | Later backend integration may require UI changes. | Define repository/provider seams with minimal stable fields: handle, DID, display name/avatar/following/Craftsky flag for accounts; tag and 28-day count for hashtags. |
| RISK-006 | Adding a dependency may affect lockfile/build/codegen workflows. | Implementation may need dependency review and `flutter pub get`. | Use the documented atproto.dart package intentionally; keep dependency impact explicit in implementation planning. |
| RISK-007 | Bare-domain link detection may produce false positives or miss uncommon valid domains. | Users may see unintended link styling/tap targets or missing link facets. | Keep parsing rules explicit, normalize only recognized domains to HTTPS, trim punctuation, and cover examples in tests. |
| RISK-008 | Search route hashtag context is ahead of real hashtag search functionality. | Taps may navigate to a placeholder search page that cannot show results yet. | Treat route/context preservation as this slice's scope; keep hashtag search results as a non-goal. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | `bluesky_text` or equivalent atproto.dart helper functionality can be used for byte-safe entity detection without requiring direct external handle resolution. | Implementation may need a lower-level/custom byte-index helper for Craftsky mention resolution. |
| ASM-002 | Future AppView support for profile `descriptionFacets` will use the `descriptionFacets` field name aligned with `app.bsky.actor.profile`. | Flutter payload shape may need adjustment in a follow-up. |
| ASM-003 | Mock hashtag popularity represents number of posts containing the hashtag in the last 28 days. | UI labels/tests may need wording changes if product chooses a different popularity metric. |
| ASM-004 | Account suggestions can be represented with DID, handle, optional display name/avatar, `isCraftskyProfile`, and `viewerIsFollowing`. | Future AppView autocomplete may need additional fields. |
| ASM-005 | The existing Flutter search route can be extended to accept optional hashtag context as query state without implementing search results in this slice. | Hashtag tap acceptance tests may need to assert navigation to `/search` without query context until a route follow-up lands. |
| ASM-006 | Bare-domain detection can be constrained enough in Flutter to avoid most false positives while still recognizing common domains. | Link parsing rules may need tightening or additional tests if false positives appear. |

## 21. Open Questions

- [ ] Blocking for live profile usability, but not for this Flutter-only requirements stage: AppView must add profile `descriptionFacets` request/response support before live profile saves with facets can succeed.

## 22. Review Status

Status: Draft
Risk level: High
Review recommended: Required
Reviewer: TBD
Date: 2026-06-01
Notes: Review is required before implementation because the confirmed direction intentionally changes Flutter write payloads and sends profile `descriptionFacets` ahead of current AppView support, creating a known live compatibility break until a backend follow-up lands.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-06-01-flutter-facets-ui/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`, `BR-002`
  - `FR-001` through `FR-013`
  - `NFR-001`, `NFR-002`, `NFR-004`, `NFR-005`
  - `RULE-001` through `RULE-009`
- Suggested test levels:
  - Unit tests for facet parsing/generation, UTF-8 byte offsets, overlap handling, unknown mention handling, manually typed resolvable mention handling, bare-domain HTTPS normalization, link punctuation trimming, hashtag casing/character rules, and suggestion filtering/sorting.
  - Provider/repository tests for mock autocomplete seams and post/profile facet payload propagation.
  - Widget tests for post composer autocomplete, profile bio autocomplete, suggestion display fields, intentional mention-vs-hashtag empty-state behavior, token-boundary/minimum-query activation, selection/caret/trailing-space behavior, 300 ms debounce behavior with fake async where practical, primary-color facet styling, destination actions including search tag context, rich text rendering fallback/unknown feature handling, and existing composer/profile regression flows.
  - API-client tests for `facets` and `descriptionFacets` JSON body inclusion.
- Blocking open questions:
  - None for test design of this Flutter slice.
  - Live profile usability remains blocked on a future AppView/API change for `descriptionFacets`.
