# UI Polish Notes: Post And Content Languages

## Summary

The Languages settings page and post-composer language picker now use CraftSky's established card, modal, text-input, select-input, spacing, loading, and responsive-width patterns instead of a mixture of raw Material controls and bespoke dialogs. Primary and Content preference persistence, composer language selection, save serialization, failure rollback, and filtering behavior are unchanged.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User review: page theming is inconsistent | Rebuilt the three settings as `CraftskyCard` sections using `SpacingTheme`, the shared stitch loader, and the same 720px content constraint as Notification settings. | `app/lib/languages/pages/languages_page.dart` | Done |
| UIP-002 | User review: reuse widgets that are part of the theme | Replaced the raw App-language dropdown, Primary button/dialog, Content chips/button/dialog, and hard-coded dividers with `CraftskySingleSelectInput` and `CraftskySearchableMultiSelectInput`. | `app/lib/languages/pages/languages_page.dart` | Done |
| UIP-003 | Preserve existing catalogue usability while adopting shared inputs | Made themed select search include option descriptions as well as labels, and display option descriptions in the multi-select. Language codes therefore remain visible and searchable alongside friendly names. | `app/lib/theme/select_inputs/select_helpers.dart`; `app/lib/theme/select_inputs/single_select_input.dart`; `app/lib/theme/select_inputs/searchable_multi_select_input.dart` | Done |
| UIP-004 | Regression coverage | Updated the Languages page widget test to use the real CraftSky theme and assert branded cards/selects, no raw Material dropdown, name/code search, independent Primary/Content changes, empty Content behavior, and failed-save recovery. | `app/test/languages/languages_page_test.dart` | Done |
| UIP-005 | Simulator review: unnecessary rounded-rectangle crop clips card content | Disabled child clipping on every Languages-page `CraftskyCard` while retaining its rounded border, paper surface, and drop shadow. Added a widget assertion covering the three rendered cards. | `app/lib/languages/pages/languages_page.dart`; `app/test/languages/languages_page_test.dart` | Done |
| UIP-006 | User review: the composer's Add language modal and search field are not CraftSky themed | Replaced the raw `AlertDialog` and `TextField` with `showCraftskyModal`, `CraftskyDialog`, and `CraftskyTextInput`. Preserved search autofocus and made the result-list height respond to the keyboard and available viewport. Added regression assertions for the branded controls. | `app/lib/languages/widgets/post_language_selector.dart`; `app/test/languages/post_language_selector_test.dart` | Done |
| UIP-007 | Simulator review: an apparent empty first option appears above Abkhazian | Removed the result list's inherited top safe-area inset by setting explicit zero padding. Added a regression assertion for the list padding. | `app/lib/languages/widgets/post_language_selector.dart`; `app/test/languages/post_language_selector_test.dart` | Done |

## Verification

- Commands run:
  - `flutter test test/languages/languages_page_test.dart --reporter expanded`
  - `flutter test test/languages test/theme/craftsky_form_builder_dropdown_test.dart test/theme/craftsky_form_builder_multi_select_test.dart --reporter compact`
  - `flutter test test/languages/post_language_selector_test.dart --reporter expanded`
  - `flutter test test/languages/post_language_selector_test.dart test/languages/language_accessibility_test.dart test/feed/widgets/post_composer_languages_test.dart test/projects/widgets/project_composer_languages_test.dart --reporter compact`
  - `dart analyze`
  - `git diff --check`
- Passing evidence:
  - The focused Languages page suite passed both interaction tests.
  - The combined language and themed-select suite passed all 52 tests.
  - The rendered Languages cards are covered as non-clipping surfaces.
  - The composer selector suite passed both tests and now asserts that the picker renders a `CraftskyDialog` and `CraftskyTextInput`, with no raw `AlertDialog`.
  - The selector regression also verifies that the language list has no inherited safe-area padding before its first result.
  - The combined selector, accessibility, post-composer, and project-composer language regression run passed all 7 tests.
  - Static analysis passed with no issues.
  - Whitespace validation passed.
- Skipped checks and reason:
  - No broad AppView or complete Flutter rerun was needed because this pass changes only Flutter presentation and backward-compatible themed-select search/display behavior.
  - Physical-device visual and enlarged-text review remains part of the existing manual UI release gate.

## Scope Guardrails

- Requirement behavior changed: No
- Business logic changed: No
- APIs, data models, migrations, permissions, or dependencies changed: No
- Notes:
  - Preference reads and complete replacement writes still use the existing provider and repository flow.
  - Content-language cache invalidation and server-authoritative filtering are untouched.
  - The shared select enhancement is backward compatible: existing label search remains, while optional descriptions also become searchable and visible.

## Follow-ups

- [ ] Confirm spacing, keyboard behavior, screen-reader announcements, and enlarged text on a physical phone during the existing manual release check.
