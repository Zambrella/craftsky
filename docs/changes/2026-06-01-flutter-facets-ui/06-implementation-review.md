# Implementation Review: Flutter Facets UI

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation reviewer
Date: 2026-06-01
Risk level: High

## Summary

The implementation delivers a substantial Flutter-only facets slice: shared facet generation and rendering utilities, mock-backed mention/hashtag suggestions, post/profile payload propagation, rendered facet tap actions, search tag routing, and a broad set of focused tests. The changed source remains within the Flutter app boundary; no AppView, migration, lexicon, or backend files were changed.

However, the implementation is not ready to approve because one explicit Must UI requirement is not implemented: active mention/hashtag tokens in editable composer/profile fields do not use the theme primary color. The editor still renders through a normal `BrandTextField`/`TextField` with one uniform text style. In addition, required high-risk coverage from the coding-plan review checklist is missing for the known live AppView `descriptionFacets` rejection path, and the autocomplete debounce/provider implementation deviates from the approved Riverpod auto-dispose debounce plan without equivalent integration coverage.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Tests | Editable post/profile fields do not visually style active mention or hashtag tokens with the theme primary color. `FacetAutocompleteEditor` composes `BrandTextField` with a plain `TextEditingController`, and `BrandTextField` renders a normal `TextField` with a uniform `theme.textTheme.bodyLarge` style. The implemented primary-color tests cover rendered/shared spans, not the editable field requirement. | `01-requirements.md` `NFR-005`, `AC-029`; `04-coding-plan.md` §6 “Editable primary-color token styling”; `app/lib/shared/rich_text/widgets/facet_autocomplete_editor.dart`; `app/lib/theme/brand_text_field.dart`; `app/test/shared/rich_text/faceted_text_span_builder_test.dart` | Add editable-field styling for active mention/hashtag tokens in the shared editor, preferably via a specialized `TextEditingController.buildTextSpan` or minimal `BrandTextField` extension, and add/adjust widget tests for `AC-029` in both composer/profile editor contexts or the shared editor with representative usage. |
| IR-002 | Important | Tests / Risk | The planned `descriptionFacets` compatibility-risk test is missing. The implementation sends `descriptionFacets` and comments on the known AppView incompatibility, but it does not simulate the current AppView `unexpected_field`/bad-request failure and verify the existing profile save error path remains safe. This was called out as an implementation-review checklist item because live profile saves can fail until a backend slice lands. | `01-requirements.md` `RULE-004`, `RISK-001`, `AC-012`; `02-acceptance-tests.md` `IT-006`, `MAN-003`; `04-coding-plan.md` “descriptionFacets compatibility handling”; `app/test/profile/data/profile_api_client_test.dart`; `app/test/profile/edit_profile_dialog_facets_test.dart` | Add the planned API-client and/or profile-dialog coverage that simulates the current AppView rejection for `descriptionFacets`, asserts it flows through the existing API exception/save-error path without crashing, and keeps the follow-up backend requirement explicit. |
| IR-003 | Important | Traceability / Tests | Autocomplete debounce was implemented with widget-local `Timer` logic and direct repository reads rather than the approved Riverpod auto-disposed provider-family debounce/cancel pattern. The pure `DebouncedFacetLookup` helper is tested, but it is not used by `FacetAutocompleteEditor`, and the shared editor tests override debounce to `Duration.zero`; they do not verify that the actual editor waits for the configured debounce, cancels superseded lookups, or prevents stale suggestions from applying. | `01-requirements.md` `NFR-002`, `AC-015`; `02-acceptance-tests.md` `AT-007`, `IT-005`; `04-coding-plan.md` §6 and guardrail `CPQ-008`; `app/lib/shared/rich_text/widgets/facet_autocomplete_editor.dart`; `app/lib/shared/rich_text/providers/facet_suggestion_providers.dart`; `app/test/shared/rich_text/facet_autocomplete_editor_test.dart`; `app/test/shared/rich_text/facet_autocomplete_controller_test.dart` | Either align the implementation with the Riverpod auto-dispose provider-family debounce plan, or document a justified deviation and add equivalent widget/provider tests proving the actual editor honors the injectable 300 ms debounce and suppresses stale/superseded results. |

## Requirement And Test Traceability

- Requirements implemented: Core facet generation (`FR-001`, `FR-012`, `NFR-001`, `RULE-002`, `RULE-003`, `RULE-007`, `RULE-008`), rendered post/profile facets (`FR-004`, `FR-005`, `RULE-009`), post payload propagation (`FR-002`), profile `descriptionFacets` send (`FR-003`), mock suggestion repositories (`BR-002`, `FR-010`, `FR-011`), shared autocomplete insertion flows (`FR-006` through `FR-009`, `RULE-006`), and tap destinations (`FR-013`) are substantially represented in code and tests.
- Tests implemented: The implementation added focused unit/widget/integration tests for facet generation, normalization, span styling, suggestion filtering, payload propagation, composer/profile submit paths, rendered surfaces, tap actions, and search tag routing.
- Unplanned behavior: No backend/AppView, lexicon, migration, or PDS/external identity changes were identified in the branch diff. No third-party atproto helper was added, so the external helper-resolution risk is avoided for this slice.
- Remaining gaps: `AC-029` editable primary-color token styling is not implemented/tested; `IT-006` / `AC-012` rejection-path coverage is missing; actual editor debounce/cancel behavior lacks integration coverage and deviates from the approved provider plan.

## Test Evidence

- Commands reviewed:
  - `git status --short` — clean before review.
  - `git diff --stat main..HEAD` and `git diff --name-only main..HEAD` — reviewed changed-file scope.
  - `cd app && flutter test test/shared/rich_text test/feed/widgets/post_composer_sheet_facets_test.dart test/profile/edit_profile_dialog_facets_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart` — run during review.
  - `cd app && dart analyze lib/shared/rich_text lib/feed/widgets/post_card.dart lib/feed/widgets/post_composer_sheet.dart lib/profile/models/profile.dart lib/profile/pages/edit_profile_dialog.dart lib/profile/providers/save_profile_provider.dart lib/profile/widgets/profile_bio.dart lib/profile/widgets/profile_meta_section.dart lib/router/router.dart lib/search/pages/search_page.dart test/shared/rich_text test/feed/widgets/post_card_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart` — run during review.
  - `05-implementation-plan.md` reported `cd app && flutter test` passing (`+486`) and focused command snapshots passing.
- Passing evidence:
  - Review-run focused Flutter facet tests passed (`+57`).
  - Review-run focused analyzer command passed with no issues.
  - Implementation-plan evidence reports full Flutter test suite passed (`+486`).
- Failing or skipped tests:
  - No review-run command failed.
  - I did not rerun the full `cd app && flutter test` suite during review; I reviewed the implementation-plan evidence and reran the focused facet/regression subset above.
  - Missing tests are captured in IR-001, IR-002, and IR-003.

## Risk Review

- Risk level: High.
- Risk notes:
  - Profile `descriptionFacets` live-save incompatibility remains intentional but needs stronger rejection-path coverage before approval.
  - AT Protocol byte-offset and rendering tests are broad and passed in the focused review run.
  - Flutter-only architecture boundary appears preserved: the branch diff is limited to Flutter app code/tests and workflow docs, with no AppView, lexicon, migration, PDS, or external identity lookup changes.
  - No new dependency was added; helper-related external resolution risk is avoided by the custom local parser/resolver approach.
- Approval notes: The implementation should return to TDD for the blocking findings above. After fixes, rerun the focused changed-file analyzer, focused facet tests, profile save/error tests, and the full Flutter suite.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: This change is user-facing, and a small polish pass may be useful after the required implementation fixes, especially for copy/localization of suggestion labels such as `No results` and `posts in the last 28 days`, visual spacing of dropdowns, and accessibility semantics. The missing editable primary-color styling is not polish; it is a blocking implementation requirement captured in IR-001.
- Suggested polish notes: Consider localizing hardcoded suggestion/dropdown copy, checking light/dark contrast for primary-colored facet text, and doing a quick visual pass on dropdown spacing in both post composer and profile editor.

## Handoff Back To TDD Builder

- Required fixes:
  - Implement and test active mention/hashtag token primary-color styling in editable composer/profile fields (`IR-001`).
  - Add the planned current-AppView `descriptionFacets` rejection/error-path coverage (`IR-002`).
  - Align autocomplete debounce with the Riverpod provider-family plan or add a documented deviation plus equivalent integration tests against the actual editor behavior (`IR-003`).
- Suggested next failing test:
  - Start with a shared-editor widget test for `AC-029`: type `@ali` or `#sock`, keep the token active, and assert the active token text span in the editable field uses `Theme.colorScheme.primary`.
- Verification to rerun:
  - `cd app && dart analyze lib/shared/rich_text lib/feed/widgets/post_card.dart lib/feed/widgets/post_composer_sheet.dart lib/profile/models/profile.dart lib/profile/pages/edit_profile_dialog.dart lib/profile/providers/save_profile_provider.dart lib/profile/widgets/profile_bio.dart lib/profile/widgets/profile_meta_section.dart lib/router/router.dart lib/search/pages/search_page.dart test/shared/rich_text test/feed/widgets/post_card_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart`
  - `cd app && flutter test test/shared/rich_text test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/feed/providers/create_post_provider_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/widgets/post_card_test.dart test/profile/data/profile_api_client_test.dart test/profile/edit_profile_dialog_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart`
  - `cd app && flutter test`
