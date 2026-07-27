# UI Polish Notes: Flutter Saved Posts

## Summary

Aligned Saved Posts dialogs, sorting, section headers, and row actions with the
app's existing themed UI components. The overview remains one scroll surface,
with pinned Folders and Unfiled headers. A follow-up pass removed the header
dividers, aligned summary timestamps with regular post headers, and themed all
folder-name inputs. The Save/Move chooser now uses the themed, scrollable
single-select control instead of an expanding radio list, with design-system
spacing before inline folder creation.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User note: use the app's themed modal | Added a reusable themed modal presenter and moved save, move, create, rename, and delete folder dialogs onto `CraftskyDialog` with existing themed actions. | `app/lib/theme/craftsky_dialog.dart`, `app/lib/saved_posts/widgets/save_post_dialog.dart`, `app/lib/saved_posts/widgets/saved_post_folder_dialogs.dart` | Done |
| UIP-002 | User note: match post-reply sort styling | Replaced saved-post dropdowns with a `SortMenuButton`-based Saved Posts sort control, including localized option descriptions. | `app/lib/saved_posts/widgets/saved_post_sort_button.dart`, `app/lib/saved_posts/pages/saved_posts_page.dart`, `app/lib/saved_posts/pages/saved_post_folder_page.dart`, `app/lib/l10n/app_en.arb` | Done |
| UIP-003 | User note: sticky Folders and Unfiled headers | Composed each overview section with `MultiSliver` and `SliverPinnedHeader`; added an opaque, themed section-header surface so pinned content remains legible. | `app/lib/saved_posts/pages/saved_posts_page.dart`, `app/pubspec.yaml`, `app/pubspec.lock` | Done |
| UIP-004 | User note: consolidate Move and Unsave | Replaced inline actions with one accessible 48px context-menu button using the existing responsive themed context menu; Unsave keeps destructive styling and semantics. | `app/lib/saved_posts/widgets/saved_post_row.dart`, `app/lib/l10n/app_en.arb` | Done |
| UIP-005 | Regression coverage | Updated widget tests for themed dialogs, themed sort controls, pinned sliver structure, context-menu actions, and large-text touch-target behavior. | `app/test/saved_posts/` | Done |
| UIP-006 | User follow-up: remove header dividers | Kept the pinned headers opaque while removing their bottom border. | `app/lib/saved_posts/pages/saved_posts_page.dart` | Done |
| UIP-007 | User follow-up: show post time at the top right of summaries | Saved summaries now supply the post creation time, and summaries with authors render that time in the same 48px top-right header slot as regular posts. The existing saved-at time remains in the Saved Post row footer. | `app/lib/shared/widgets/post_summary.dart`, `app/test/shared/widgets/post_summary_test.dart`, `app/test/saved_posts/widgets/saved_post_row_test.dart` | Done |
| UIP-008 | User follow-up: theme the folder-name input | Replaced the standalone create/rename and inline New Folder text fields with `CraftskyTextInput`, preserving modal autofocus, disabled, and error states. | `app/lib/saved_posts/widgets/saved_post_folder_dialogs.dart`, `app/lib/saved_posts/widgets/save_post_dialog.dart` | Done |
| UIP-009 | User follow-up: use a dropdown for folder selection | Replaced the radio list with a scrollable `CraftskySingleSelectInput`. No folder, exact opaque-ID selection, duplicate names, pagination/retry, inline creation, and the prohibition on misleading partial client search remain intact. | `app/lib/saved_posts/widgets/save_post_dialog.dart`, `app/lib/theme/select_inputs/single_select_input.dart`, `app/lib/l10n/app_en.arb` | Done |
| UIP-010 | User follow-up: separate selector and inline folder input | Added `SpacingTheme.sp4` between the folder selection/pagination area and the expanded New Folder input. | `app/lib/saved_posts/widgets/save_post_dialog.dart` | Done |

## Verification

- Commands run:
  - `flutter gen-l10n`
  - `dart format` on changed Dart files
  - `flutter analyze`
  - `flutter test test/saved_posts`
  - `flutter test test/saved_posts test/shared/widgets/post_summary_test.dart`
  - `flutter test test/saved_posts/widgets/save_post_dialog_test.dart test/saved_posts/pages/saved_post_folder_page_test.dart test/saved_posts/pages/saved_posts_page_test.dart`
  - `flutter test test/saved_posts/widgets/save_post_dialog_test.dart test/saved_posts/pages/saved_post_folder_page_test.dart`
  - `flutter test test/theme/craftsky_form_builder_dropdown_test.dart test/notifications/notification_settings_page_test.dart test/saved_posts test/shared/widgets/post_summary_test.dart`
  - `flutter test test/saved_posts/widgets/save_post_dialog_test.dart`
- Passing evidence:
  - Flutter analysis completed with no issues.
  - All 56 Saved Posts tests passed.
  - The combined Saved Posts and shared Post Summary run passed all 57 tests.
  - Focused coverage asserts two pinned section headers, the themed dialog
    surface, context-menu action routing, and a 48px action target at 2x text.
  - Follow-up coverage asserts the simplified colored header surfaces and
    verifies the post creation time occupies the summary header rather than its
    content footer.
  - All 31 folder-dialog and affected Saved Posts page tests passed; coverage
    asserts the themed input wrapper and retained autofocus.
  - All 23 selector and move-dialog tests passed, including paginated
    duplicate-name selection, No folder, and a long scrollable list without
    partial search.
  - The combined themed-select, notification-select consumer, Saved Posts, and
    Post Summary regression run passed all 74 tests.
  - All 8 Save/Move dialog tests passed, including an explicit 16px
    design-system spacing assertion.
- Skipped checks and reason:
  - Manual screenshot comparison was not possible because the supplied
    simulator screenshot path no longer existed when inspected.

## Scope Guardrails

- Requirement behavior changed: No
- Business logic changed: No
- APIs, data models, migrations, permissions, or dependencies changed: Yes
- Notes: No APIs, models, migrations, or permissions changed. The only
  dependency change is the user-requested `sliver_tools: ^0.2.12` addition for
  `SliverPinnedHeader`.

## Follow-ups

- [ ] Check the pinned-header transitions and compact context-menu sheet on a
  simulator or physical device when a current build is available.
