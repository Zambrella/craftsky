# UI Polish Notes: App-Wide UI Polish

## Summary

Refined post media and expandable text, navigation-rail composition, refresh and scroll controls, and Projects filtering without changing product behavior. Post images now lift into the app's root overlay during two-finger zoom.

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

## Verification

- Commands run: focused Flutter widget tests, `just app-analyze`, `just app-test`, `git diff --check`, and Flutter hot reload/restart.
- Passing evidence: all 2,003 app tests before the zoom dependency swap; final post-image tests (14), affected PostCard interaction tests (9), shell-layout tests (3), and full app analysis after the swap.
- Skipped checks and reason: the final full-suite rerun reached 336 passing tests before host contention caused the six-minute command timeout to terminate Flutter's workers; no test assertion failed.

## Scope Guardrails

- Requirement behavior changed: No
- Business logic changed: No
- APIs, data models, migrations, or permissions changed: No
- Dependencies changed: Replaced `pinch_zoom` with `zoom_pinch_overlay` 1.4.3.
- Notes: Existing callbacks, filter state, navigation, scrolling, and accessibility behavior are preserved.

## Follow-ups

- [ ] None.
