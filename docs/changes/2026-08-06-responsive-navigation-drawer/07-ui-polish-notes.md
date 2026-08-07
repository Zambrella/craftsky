# UI Polish Notes: Responsive App Navigation Drawer And Extended Rail

## Summary

Simplified the drawer and navigation rail presentation by removing separator rules, using a standard Profile icon instead of the active account picture on those two surfaces, and giving the drawer the same rounded ink-outline treatment as the app's stylized context-menu bottom sheet. The compact bottom navigation still uses the active account avatar. The drawer and rail Profile rows now include a trailing switch-account icon, while tapping the Profile destination itself still navigates normally and the existing long-press shortcut remains available. The compact drawer's swipe activation region now extends 96 logical pixels from the start edge, making the gesture easier to initiate without claiming the whole screen. Terms and Privacy use compact 40-pixel tap targets in both navigation presentations to reduce footer height. The two personal-content menu labels are shortened to “Saved” and “Scheduled” while their destination page titles remain unchanged. A small, muted localized `version (build)` label sourced from the initialized package metadata now appears below Feedback.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User: remove dividers in the navigation rail and drawer | Removed the core/secondary and footer dividers from the drawer, the footer divider from the rail, and the vertical rule between the rail and page content. | `app/lib/router/app_shell.dart` | Done |
| UIP-002 | User: use a Profile icon instead of the profile picture in the rail/drawer only | Added an icon-only Profile presentation for the drawer and rail while preserving the avatar in the compact bottom bar and preserving account-switcher interactions. | `app/lib/router/app_shell.dart`, `app/test/router/app_shell_account_switcher_test.dart`, `app/test/router/app_shell_navigation_menu_test.dart` | Done |
| UIP-003 | User: add a black drawer border matching the stylized bottom sheet | Applied the existing `RadiusTheme.r4`, paper surface, transparent tint, clipping, and 1.5px `onSurface` ink outline used by the compact context-menu sheet. | `app/lib/router/app_shell.dart`, `app/test/router/app_shell_navigation_menu_test.dart` | Done |
| UIP-004 | User: make the drawer swipe gesture work farther from the edge | Set the compact shell's drawer activation width to 96 logical pixels and added a widget test that opens it from 80 pixels inside the screen. | `app/lib/router/app_shell.dart`, `app/test/router/app_shell_navigation_menu_test.dart` | Done |
| UIP-005 | User: make the Terms and Privacy tap targets more compact | Reduced the two legal-link rows/buttons to 40 pixels high in both the compact drawer and large rail without changing their labels or launch behavior. | `app/lib/router/app_shell.dart`, `app/test/router/app_shell_navigation_menu_test.dart` | Done |
| UIP-006 | User: add a switcher icon to the end of the Profile row | Added a localized trailing switch-account `IconButton` to Profile in both the compact drawer and extended rail. The button opens the existing compact sheet or anchored large-screen menu without selecting Profile; the Profile row retains its normal navigation behavior and long-press shortcut. | `app/lib/router/app_shell.dart`, `app/test/router/app_shell_account_switcher_test.dart` | Done |
| UIP-007 | User: shorten “Saved posts” and “Scheduled posts” | Added dedicated localized navigation labels, “Saved” and “Scheduled,” for the drawer and rail while retaining the full existing titles on the destination pages. | `app/lib/l10n/app_en.arb`, `app/lib/l10n/generated/app_localizations.dart`, `app/lib/l10n/generated/app_localizations_en.dart`, `app/lib/router/app_shell.dart`, `app/test/router/app_shell_navigation_menu_test.dart` | Done |
| UIP-008 | User: show the build version and number beneath Feedback | Reused the startup-resolved package metadata, formatted it through localization as `version (build)`, and rendered it in centered, muted `labelSmall` text directly below Feedback in both responsive navigation presentations. | `app/lib/l10n/app_en.arb`, `app/lib/l10n/generated/app_localizations.dart`, `app/lib/l10n/generated/app_localizations_en.dart`, `app/lib/router/app_shell.dart`, `app/test/router/app_shell_navigation_menu_test.dart` | Done |

## Verification

- Commands run:
  - `flutter test test/router/app_shell_navigation_menu_test.dart test/router/app_shell_account_switcher_test.dart test/notifications/app_shell_notification_badge_test.dart test/router/top_level_drawer_button_test.dart`
  - `flutter test test/router/app_shell_navigation_menu_test.dart --plain-name 'UIP-004 compact drawer opens from the wider swipe region'`
  - `flutter test test/router/app_shell_navigation_menu_test.dart --plain-name 'UIP-005 legal link tap targets are compact'`
  - `flutter test test/router/app_shell_account_switcher_test.dart --plain-name 'UIP-006 compact Profile row switch button opens account switcher'`
  - `flutter test test/router/app_shell_account_switcher_test.dart --plain-name 'UIP-006 rail Profile switch button opens an anchored menu'`
  - `flutter test test/router/app_shell_navigation_menu_test.dart --plain-name 'UIP-007 menu uses compact Saved and Scheduled labels'`
  - `flutter test test/router/app_shell_navigation_menu_test.dart --plain-name 'UIP-008 build version appears below Feedback'`
  - `dart analyze`
  - `git diff --check`
- Passing evidence:
  - 60 focused navigation, account-switcher, notification-badge, and top-level hamburger tests passed.
  - The widened-swipe regression test failed before `drawerEdgeDragWidth` was configured and passed after it was set to 96 logical pixels.
  - The compact legal-link layout test failed at the previous 48-pixel height and passed after both presentations were reduced to 40 pixels.
  - The switch-button tests failed before the explicit icons were present and pass for both compact and large account-switcher presentations, including independent Profile navigation and keyboard activation on the rail.
  - The compact-label test failed against the former menu copy and passes with “Saved” and “Scheduled” in both responsive presentations; the old copy is absent from the drawer.
  - The version-footer test failed before the label existed and passes with `1.0.0 (1)` positioned below Feedback in both the drawer and rail.
  - Static analysis completed with no issues.
  - Diff whitespace validation passed.
- Skipped checks and reason:
  - No physical-device visual pass was run; this remains suitable for the final implementation review or a later device QA pass.

## Scope Guardrails

- Requirement behavior changed: No. The user's latest visual instruction supersedes the earlier avatar presentation for drawer/rail only, and the wider gesture region refines the existing required swipe-to-open behavior; routing, selection, account switching, badges, and Back behavior are unchanged.
- Business logic changed: No.
- APIs, data models, migrations, permissions, or dependencies changed: No.
- Notes: The compact bottom bar remains intentionally unchanged and continues to display the active account avatar.

## Follow-ups

- [ ] Run the final implementation review if requested.
