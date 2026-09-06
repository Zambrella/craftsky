# UI Polish Notes: App-Wide UI Polish

## Summary

Refined post media and expandable text, navigation-rail composition, refresh and scroll controls, Projects filtering, and app-wide interaction feedback without changing business behavior. Post images now lift into the app's root overlay during two-finger zoom.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User feedback | Gave post-card images the embedded-preview outline color and radius. | `app/lib/feed/widgets/post_image_carousel.dart` | Done |
| UIP-002 | User feedback | Reduced the wide navigation-rail composer action to the standard chunky-button height. | `app/lib/router/app_shell.dart`, `app/lib/theme/craftsky_floating_action_button.dart` | Done |
| UIP-003 | User feedback | Made Show more/less scale once with post text while retaining body color and bold emphasis. | `app/lib/shared/rich_text/widgets/faceted_text.dart` | Done |
| UIP-004 | User feedback | Changed the shared back-to-top control to a quieter surface treatment. | `app/lib/shared/widgets/scroll_to_top_button.dart` | Done |
| UIP-005 | User feedback | Moved Projects filtering from the app bar to an extended floating action button. | `app/lib/projects/pages/projects_page.dart` | Done |
| UIP-006 | User feedback | Standardized shared sorting controls on the `funnel-simple` glyph. | `app/lib/theme/craftsky_icons.dart` | Done |
| UIP-007 | User feedback | Applied a global paper-cutout snackbar theme while preserving severity surfaces. | `app/lib/theme/app_theme.dart` | Done |
| UIP-008 | User feedback | Made carousel height responsive and contained over-height portrait media to preserve its aspect ratio. | `app/lib/feed/widgets/post_image_carousel.dart` | Done |
| UIP-009 | User feedback | Replaced inline-clipped pinch zoom with a two-finger root-overlay zoom that clears shell navigation and snaps back on release. | `app/lib/feed/widgets/post_image_carousel.dart`, `app/lib/shared/widgets/root_overlay_scope.dart`, `app/lib/router/app_shell.dart` | Done |
| UIP-010 | User feedback | Removed the compact drawer-button inset from the collapsed profile header when the navigation rail is present. | `app/lib/profile/widgets/profile_sliver_app_bar.dart` | Done |
| UIP-011 | User feedback | Let scaled post-author identities use all available header width before truncating. | `app/lib/feed/widgets/post_card.dart` | Done |
| UIP-012 | User feedback | Removed the banner editor and centered the shadowless avatar editor on Edit profile. | `app/lib/profile/pages/edit_profile_dialog.dart`, `app/lib/profile/widgets/edit_profile_banner_avatar.dart` | Done |
| UIP-013 | User feedback | Removed the expanded profile menu surface and used the selected profile palette's audited contrasting foreground. | `app/lib/profile/widgets/profile_sliver_app_bar.dart` | Done |
| UIP-014 | User feedback | Replaced global radial splashes with primary-color pressed, focused, and hovered state layers; removed ChunkyButton's local ripple override. | `app/lib/theme/app_theme.dart`, `app/lib/theme/chunky_button.dart`, `app/test/theme/app_theme_test.dart` | Done |
| UIP-015 | User feedback | Opened app-controlled Feedback and Request more destinations directly while retaining confirmation for links sourced from user content. | `app/lib/router/app_shell.dart`, `app/lib/onboarding/pages/onboarding_page.dart`, `app/lib/profile/pages/edit_profile_dialog.dart` | Done |
| UIP-016 | User screenshot | Made ChunkyButton icons and labels share one state-aware foreground resolver so profile-colored secondary actions retain contrast when pressed, hovered, or focused. | `app/lib/theme/chunky_button.dart`, `app/test/profile/widgets/profile_customisation_controls_test.dart` | Done |

## Verification

- Commands run: focused Flutter widget tests, `just app-analyze`, `just app-test`, `git diff --check`, Flutter hot reload/restart, theme tests, sign-in page tests, and trusted-link flow tests.
- Passing evidence: all 2,003 app tests before the zoom dependency swap; final post-image tests (14), affected PostCard interaction tests, responsive profile-header tests, edit-profile avatar tests, shell-layout tests (3), theme tests (14), sign-in page tests (7), trusted-link and user-content confirmation tests (84), profile control/card tests (24), and static analysis of the latest changes.
- Skipped checks and reason: the final full-suite rerun reached 336 passing tests before host contention caused the six-minute command timeout to terminate Flutter's workers; no test assertion failed.

## Scope Guardrails

- Requirement behavior changed: No
- Business logic changed: No
- APIs, data models, migrations, or permissions changed: No
- Dependencies changed: Replaced `pinch_zoom` with `zoom_pinch_overlay` 1.4.3.
- Notes: Existing filter state, navigation, scrolling, and accessibility behavior are preserved. App-controlled links now launch without confirmation; user-content links still require it.

## Follow-ups

- [ ] None.
