# Implementation Review: Flutter Facets UI

## Verdict
Status: Approved with notes
Reviewer: gpt-5.5 implementation reviewer
Date: 2026-06-01
Risk level: High

## Summary

The updated implementation is ready to hand off. The previous blocking review findings are addressed: editable composer/profile facet tokens now use theme primary coloring through `FacetTextEditingController`, and the known current-AppView `descriptionFacets` rejection path is covered at both API-client and profile-dialog/save-error levels. The implementation remains within the Flutter app boundary; no AppView, migration, lexicon, PDS, or external identity lookup changes were identified.

The only remaining note is the intentionally accepted debounce implementation deviation: the editor uses an injectable widget-local `Timer` instead of the coding plan's Riverpod auto-dispose provider-family debounce pattern. The implementation plan records explicit user instruction to skip that issue because the current behavior works. The actual editor does read the injectable 300 ms debounce provider and suppresses stale callbacks by token identity, so this is not blocking for this review, but a future cleanup could add direct widget coverage for the timer-backed editor debounce.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-004 | Suggestion | Tests / Maintainability | Autocomplete debounce still deviates from the approved Riverpod provider-family plan and lacks direct widget coverage that the actual `FacetAutocompleteEditor` waits for the configured delay before querying. This was explicitly skipped by user instruction in the review-fix pass, and the current widget implementation uses the injectable `facetAutocompleteDebounceProvider` plus stale-token guards, so it is not blocking. | `01-requirements.md` `NFR-002`, `AC-015`; `04-coding-plan.md` §6 and `CPQ-008`; `05-implementation-plan.md` “Skipped Review Finding: IR-003”; `app/lib/shared/rich_text/widgets/facet_autocomplete_editor.dart`; `app/test/shared/rich_text/facet_autocomplete_controller_test.dart` | Optional follow-up: either align autocomplete with Riverpod auto-dispose provider families or add widget tests proving the timer-backed editor does not query until the injected delay elapses and suppresses superseded results. |

## Requirement And Test Traceability
- Requirements implemented: Core facet generation and UTF-8 byte offsets (`FR-001`, `FR-012`, `NFR-001`, `RULE-002`, `RULE-003`, `RULE-007`, `RULE-008`), post/profile payload propagation (`FR-002`, `FR-003`, `RULE-004`), rendered and editable primary-color facet styling (`FR-004`, `FR-005`, `NFR-005`), mock-backed autocomplete (`BR-002`, `FR-006` through `FR-011`, `RULE-005`, `RULE-006`), safe tap actions (`FR-013`), and rendering resilience (`RULE-009`).
- Tests implemented: Unit/widget/integration coverage exists for facet generation, normalization, span styling, editable token styling, suggestion filtering/sorting, post/profile API and repository propagation, profile `descriptionFacets` rejection handling, composer/profile submit paths, rendered post/profile surfaces, tap actions, search tag routing, and existing regression suites.
- Unplanned behavior: None identified in the reviewed diff. The branch is Flutter-app focused plus workflow documents; no backend/AppView, lexicon, migration, PDS, OAuth, or external identity lookup code was added.
- Remaining gaps: No blocking gaps. The debounce implementation-plan deviation is retained as a non-blocking note per user instruction.

## Test Evidence
- Commands reviewed:
  - `git status --short` — clean before review.
  - `git diff --stat main..HEAD` and `git diff --name-only main..HEAD` — reviewed changed-file scope.
  - `cd app && dart analyze lib/shared/rich_text lib/feed/widgets/post_card.dart lib/feed/widgets/post_composer_sheet.dart lib/profile/models/profile.dart lib/profile/pages/edit_profile_dialog.dart lib/profile/providers/save_profile_provider.dart lib/profile/widgets/profile_bio.dart lib/profile/widgets/profile_meta_section.dart lib/router/router.dart lib/search/pages/search_page.dart test/shared/rich_text test/feed/widgets/post_card_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart` — passed with no issues.
  - `cd app && flutter test test/shared/rich_text test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/feed/providers/create_post_provider_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/widgets/post_card_test.dart test/profile/data/profile_api_client_test.dart test/profile/edit_profile_dialog_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart` — passed (`+118`).
  - `cd app && flutter test` — passed (`+489`).
- Passing evidence:
  - Focused analyzer and focused Flutter facet/regression tests passed during this review.
  - Full Flutter test suite passed during this review.
  - `05-implementation-plan.md` also records the review-fix commands that addressed `IR-001` and `IR-002`.
- Failing or skipped tests:
  - No reviewed command failed.
  - The Riverpod provider-family debounce refactor / direct actual-editor debounce widget test remains intentionally skipped per `05-implementation-plan.md`.

## Risk Review
- Risk level: High.
- Risk notes:
  - Profile `descriptionFacets` live-save incompatibility remains intentional until a backend/API follow-up; updated tests verify the current rejection maps through existing error handling without crashing.
  - Flutter-only architecture boundary appears preserved; autocomplete and mention resolution use mock/injected local data, and no third-party atproto helper or external resolver path was introduced.
  - AT Protocol byte-offset, overlap, malformed incoming facet, and tap-action paths have focused automated coverage.
  - The accepted debounce implementation deviation is a maintainability/test-depth risk, not an observed behavior blocker.
- Approval notes: Ready to merge or hand off, subject to the team accepting the documented debounce note and the known out-of-scope AppView `descriptionFacets` backend follow-up.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: The user-facing UI changes are coherent enough to approve. A small polish pass could still improve hardcoded suggestion/dropdown copy, localization readiness, spacing, and accessibility semantics, but no polish issue blocks the acceptance criteria.
- Suggested polish notes: Consider localizing `No results` and `posts in the last 28 days`, checking dropdown spacing in the full post composer and profile editor, and doing a quick light/dark contrast check for primary-colored editable/rendered facet text.

## Handoff Back To TDD Builder
- Required fixes: None.
- Suggested next failing test: None required for this workflow stage. Optional future cleanup could add an actual-editor debounce widget test for `AC-015`.
- Verification to rerun:
  - `cd app && dart analyze lib/shared/rich_text lib/feed/widgets/post_card.dart lib/feed/widgets/post_composer_sheet.dart lib/profile/models/profile.dart lib/profile/pages/edit_profile_dialog.dart lib/profile/providers/save_profile_provider.dart lib/profile/widgets/profile_bio.dart lib/profile/widgets/profile_meta_section.dart lib/router/router.dart lib/search/pages/search_page.dart test/shared/rich_text test/feed/widgets/post_card_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart`
  - `cd app && flutter test test/shared/rich_text test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/feed/providers/create_post_provider_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/widgets/post_card_test.dart test/profile/data/profile_api_client_test.dart test/profile/edit_profile_dialog_test.dart test/profile/edit_profile_dialog_facets_test.dart test/profile/widgets/profile_bio_test.dart test/search/search_page_test.dart`
  - `cd app && flutter test`
