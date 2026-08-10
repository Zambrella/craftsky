# UI Polish Notes: Public Profile Customisation

## Summary

Applied the user-requested visual corrections from the 2026-08-10 iPhone simulator reviews: profile app-bar and compact-close icons now have transparent surfaces, the shared avatar renders exactly one selected-colour inside border without exposing its fallback colour as a second ring, the customisation preview uses the loaded member's profile image and identity seed when available without an avatar shadow, and the Amber, Lime, and Teal bundles use darker accessible accents.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User simulator screenshot: profile app-bar icon buttons should not have a background colour | Added profile-app-bar-only transparent button styles for the drawer button and collapsed trailing action. Other profile action buttons retain their selected bundle surfaces. | `app/lib/profile/widgets/profile_sliver_app_bar.dart`, `app/lib/router/app_shell_drawer.dart`, `app/test/profile/profile_page_test.dart` | Done |
| UIP-002 | User simulator screenshots: avatar borders appeared as two differently coloured rings | Separated the fallback/shadow, clipped image, and border into ordered layers. The image or fallback now fills the avatar, and one selected-colour border is painted over it inside the unchanged outer bounds. | `app/lib/profile/widgets/profile_avatar.dart`, `app/test/profile/widgets/profile_avatar_test.dart`, `app/test/profile/widgets/profile_card_test.dart` | Done |
| UIP-003 | User simulator screenshot: customisation preview should use the member's actual profile image when available | Retained the already-loaded self-profile avatar URL and display-name/handle seed in the editor view state and supplied them to the existing preview `ProfileAvatar`. No additional request or optimistic public-state update was added. | `app/lib/profile/providers/profile_customisation_provider.dart`, `app/lib/settings/pages/profile_customisation_page.dart`, `app/test/settings/profile_customisation_page_test.dart` | Done |
| UIP-004 | User follow-up simulator screenshot: the profile hamburger and compact-profile close button still had coloured backgrounds | Replaced the inherited-theme override with explicit transparent `ButtonStyle`s on the profile hamburger and collapsed trailing action, and removed both the themed icon fill and coloured `Material` surface from the compact close control. | `app/lib/router/app_shell_drawer.dart`, `app/lib/profile/widgets/profile_sliver_app_bar.dart`, `app/lib/profile/widgets/profile_card.dart`, `app/test/profile/profile_page_test.dart`, `app/test/profile/widgets/profile_card_test.dart` | Done |
| UIP-005 | User follow-up: remove the shadow from the profile preview on the customisation page | Disabled the shared avatar shadow only for the customisation preview; profile-page and other avatar shadows are unchanged. | `app/lib/settings/pages/profile_customisation_page.dart`, `app/test/settings/profile_customisation_page_test.dart` | Done |
| UIP-006 | User follow-up: Amber, Lime, and Teal were too light for readable hyperlink and accent text | Darkened the three high-saturation base and interaction tones, changed their button/texture foreground to white, and added AA contrast coverage for normal, hover, and pressed accent text on the cream-paper and white profile surfaces. Updated the approved palette table to match. | `app/lib/profile/models/profile_customisation.dart`, `app/test/profile/models/profile_customisation_test.dart`, `app/test/profile/widgets/profile_header_background_test.dart`, `app/test/settings/profile_customisation_page_test.dart`, `docs/changes/2026-08-09-profile-customisation/01-requirements.md` | Done |

## Verification

- Commands run:
  - `flutter test test/profile/widgets/profile_avatar_test.dart test/profile/profile_page_test.dart test/settings/profile_customisation_page_test.dart`
  - `flutter test test/profile/widgets/profile_card_test.dart`
  - `flutter test test/profile/profile_page_test.dart`
  - `flutter test test/profile/widgets/profile_avatar_test.dart test/profile/profile_page_test.dart test/profile/widgets/profile_card_test.dart test/feed/widgets/post_card_test.dart test/notifications/notifications_page_test.dart test/search/search_page_test.dart test/shared/widgets/post_summary_test.dart test/profile/widgets/edit_profile_banner_avatar_test.dart test/router/app_shell_account_switcher_test.dart test/settings/profile_customisation_page_test.dart test/profile/providers/profile_customisation_provider_test.dart`
  - `flutter test test/profile/profile_page_test.dart test/profile/widgets/profile_card_test.dart test/settings/profile_customisation_page_test.dart`
  - `flutter test test/router/app_shell_navigation_menu_test.dart`
  - `flutter test test/profile/models/profile_customisation_test.dart test/profile/widgets/profile_customisation_controls_test.dart test/profile/widgets/profile_header_background_test.dart test/settings/profile_customisation_page_test.dart`
  - `flutter test test/profile/profile_page_test.dart test/profile/widgets/profile_card_test.dart test/feed/widgets/post_card_test.dart test/notifications/notifications_page_test.dart test/search/search_page_test.dart test/shared/widgets/post_summary_test.dart test/profile/widgets/edit_profile_banner_avatar_test.dart test/router/app_shell_account_switcher_test.dart`
  - `dart analyze`
  - `git diff --check`
- Passing evidence: The initial focused group passes 46 tests. The compact profile suite passes 15 tests after updating its old combined-layer shadow assertion. The final shared-avatar surface matrix passes 162 tests. The strengthened profile app-bar suite passes 29 tests. The follow-up profile/customisation group passes 54 tests, and the shared compact-navigation suite passes 48 tests. The palette/contrast group passes 20 tests and the affected-surface matrix passes 143 tests. Static analysis reports no issues and the follow-up diffs pass whitespace checks.
- Skipped checks and reason: No full Flutter suite was run because the wider affected-surface matrices cover the shared renderer, palette, profile presentations, and touched UI seams; the existing two unrelated auth router-harness failures remain documented in `06-implementation-review.md`. Simulator visual recheck remains a manual confirmation.

## Scope Guardrails

- Requirement behavior changed: Yes, visual constants only. The user revised the Amber, Lime, and Teal palette values after simulator review; the approved Q11 table now records those darker values. The border correction restores FR-007/AC-008–AC-010's already-approved single inside stroke, and the other changes refine existing presentation.
- Business logic changed: No.
- APIs, persisted/domain data models, migrations, permissions, or dependencies changed: No.
- Notes: Internal editor view state now carries the avatar URL and identity seed already returned by its existing self-profile load. The follow-up button-style, preview-shadow, and fixed-palette revisions are presentation-only. Save, persistence, catalogue keys, validation, account fencing, and public cache behavior are unchanged.

## Follow-ups

- [ ] Confirm the corrected single-ring border, transparent profile-header controls, shadowless customisation preview, and darker Amber/Lime/Teal balance on the simulator/device at the three supported avatar sizes.
