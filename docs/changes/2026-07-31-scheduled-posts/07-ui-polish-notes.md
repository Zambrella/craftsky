# UI Polish Notes: Scheduled Posts

## Summary

The post timing control now uses the responsive CraftSky context menu instead
of a plain dialog. The scheduled-post count has also been removed from both
composers. A clear warning and management link appear only when all three
scheduling slots are occupied, while posting immediately remains available.
The time picker now uses compact CraftSky headline typography for its hour,
minute, and separator instead of inheriting the oversized editorial display
style.

## Polish Items

| ID | Source | Change | Files | Status |
| --- | --- | --- | --- | --- |
| UIP-001 | User feedback | Replace the post timing dialog with the shared CraftSky context menu, including selected and disabled states. | `app/lib/scheduled_posts/widgets/schedule_choice_menu.dart`, `app/lib/feed/widgets/post_composer_sheet.dart`, `app/lib/projects/widgets/project_composer_sheet.dart` | Complete |
| UIP-002 | User feedback | Remove the ambiguous scheduled-post count and show a themed warning only when the user cannot schedule another post. | `app/lib/scheduled_posts/composer/schedule_capacity_state.dart`, `app/lib/scheduled_posts/widgets/scheduled_post_capacity_warning.dart`, composer files, localization files | Complete |
| UIP-003 | Traceability | Align the requirements, acceptance tests, and coding plan with the approved capacity presentation. | `01-requirements.md`, `02-acceptance-tests.md`, `04-coding-plan.md` | Complete |
| UIP-004 | User screenshot and feedback | Give the Material time picker an explicit 42px CraftSky headline style for the hour, minute, and separator so it does not inherit the 96px editorial display style. | `app/lib/theme/app_theme.dart`, `app/test/theme/app_theme_test.dart` | Complete |

## Verification

- `flutter gen-l10n`
- `flutter test test/theme/app_theme_test.dart test/scheduled_posts test/projects/widgets/project_composer_sheet_test.dart test/projects/widgets/project_composer_submit_test.dart` — 45 tests passed
- `dart analyze` — no issues found
- `git diff --check`

The first focused capacity test run exposed a missing test-settle step after
entering text. The harness was corrected, and the focused and complete test
runs then passed.

The focused time-picker theme test failed before the override because the
picker had no explicit selector typography. It passed after the compact theme
was applied to both light and dark themes.

## Scope Guardrails

- Requirement behavior changed: No. The default remains “Now,” the limit
  remains three scheduled posts, an existing scheduled post retains its slot,
  and immediate posting remains available at capacity.
- Business logic changed: No. The existing capacity rules now drive a warning
  state instead of a persistent count label.
- API, storage, and publication behavior changed: No.
- Historical review and implementation records remain unchanged; this note and
  the updated source documents record the later user-approved presentation.

## Follow-ups

- [ ] Confirm the compact time header visually on the iOS simulator after a
  hot restart.
