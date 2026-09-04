# CraftSky iconography audit

Audited 4 September 2026. The inventory covers hand-written Dart under
`app/lib/`; generated files and tests are excluded from the production counts.

## Summary

- The app uses 129 unique Material icons across 260 source references.
- There are no `CupertinoIcons` or custom `IconData` definitions in production
  code. The declared `cupertino_icons` dependency is unused.
- Eight craft-specific SVG icons already exist under
  `app/assets/design/icons/`. Five map directly to supported craft taxonomy
  values; `skein`, `spool`, and `stitch` are generic decorative motifs.
- Most shared component APIs accept `IconData`, so another icon-font package
  can be introduced with relatively little structural change.
- Tests contain 188 direct Material-icon references across 33 files. Many use
  `find.byIcon` as an interaction locator and will need stable keys or semantic
  finders during migration.

Counts below are static source occurrences, not the number of times an icon is
rendered at runtime.

## Production inventory

| Material icon | Count | Current purpose |
| --- | ---: | --- |
| `add` | 3 | Add a language, product, or event |
| `add_a_photo_outlined` | 1 | Add an onboarding profile photo |
| `add_rounded` | 1 | Add another composer image |
| `alternate_email` | 1 | Mention notification |
| `arrow_back` | 1 | Onboarding back navigation |
| `arrow_downward` | 1 | Move a product down |
| `arrow_downward_rounded` | 1 | Move a composer image down |
| `arrow_upward` | 1 | Move a product up |
| `arrow_upward_rounded` | 1 | Move a composer image up |
| `auto_awesome_mosaic_outlined` | 1 | Project post type |
| `badge_outlined` | 2 | Event organiser or business identity |
| `block_outlined` | 4 | Block an account or view blocked accounts |
| `bookmark` | 1 | Saved state |
| `bookmark_border` | 1 | Unsaved state |
| `bookmark_remove_outlined` | 1 | Remove a saved post |
| `bookmarks` | 1 | Selected Saved destination |
| `bookmarks_outlined` | 1 | Unselected Saved destination |
| `brightness_6_outlined` | 1 | Appearance settings |
| `brightness_auto_outlined` | 1 | System theme |
| `broken_image_outlined` | 7 | Missing or failed image/media |
| `calendar_month_outlined` | 1 | Event date field |
| `calendar_today_outlined` | 3 | Joined date or event date |
| `cancel` | 1 | Clear a search query |
| `chat_bubble_outline` | 5 | Comments, replies, and reply notifications |
| `check` | 1 | Selected account |
| `check_box` | 1 | Selected context-menu item |
| `check_box_outline_blank` | 1 | Unselected sort option |
| `chevron_left` | 2 | Previous item or RTL disclosure |
| `chevron_right` | 8 | Forward navigation or disclosure |
| `cleaning_services_outlined` | 1 | Clear image cache |
| `close` | 8 | Dismiss or remove an item |
| `cloud_off_outlined` | 2 | Offline/account-loading failure |
| `confirmation_number_outlined` | 1 | Event admission information |
| `copy_outlined` | 1 | Copy Instagram verification value |
| `create_new_folder_outlined` | 1 | Create a saved-post folder |
| `dark_mode_outlined` | 1 | Dark theme |
| `delete_forever_outlined` | 1 | Permanent account deletion |
| `delete_outline` | 9 | Delete content or an import |
| `delete_outline_rounded` | 1 | Delete a composer image |
| `description_outlined` | 1 | Terms/document link |
| `drafts_outlined` | 1 | Draft without media |
| `drag_indicator_rounded` | 1 | Reorder a composer image |
| `drive_file_move_outline` | 1 | Move a saved post to a folder |
| `edit_note` | 1 | Selected Drafts destination |
| `edit_note_outlined` | 1 | Unselected Drafts destination |
| `edit_outlined` | 9 | Edit content, a profile, or a schedule |
| `error_outline` | 7 | General error feedback |
| `event_available_outlined` | 1 | Available/upcoming event |
| `event_busy_outlined` | 1 | Cancel or unpublish an event |
| `event_outlined` | 1 | Business event settings |
| `expand_less` | 1 | Collapse a select menu |
| `expand_more` | 1 | Expand a select menu |
| `favorite` | 1 | Liked state |
| `favorite_border` | 1 | Unliked state |
| `favorite_outline` | 1 | Like notification |
| `file_open_outlined` | 2 | Select an Instagram export file |
| `filter_list` | 1 | Open sort/filter menu |
| `flag_outlined` | 4 | Report content or an account |
| `folder_outlined` | 1 | Saved-post folder |
| `format_quote` | 2 | Quote action or notification |
| `fullscreen` | 1 | Enter video fullscreen |
| `fullscreen_exit` | 1 | Exit video fullscreen |
| `grid_view` | 1 | Selected Projects destination |
| `grid_view_outlined` | 1 | Unselected Projects destination |
| `group_add_outlined` | 1 | Follow imported Instagram matches |
| `group_outlined` | 1 | Followers settings |
| `history` | 1 | Past event |
| `home` | 2 | Selected Home or return home |
| `home_outlined` | 1 | Unselected Home destination |
| `image_not_supported_outlined` | 1 | Failed business image |
| `image_outlined` | 4 | Image placeholder or image action |
| `image_search_rounded` | 2 | Choose or replace a composer image |
| `info_outline` | 5 | Information and About feedback |
| `inventory_2_outlined` | 2 | Project count or imported inventory |
| `ios_share_outlined` | 2 | Share a profile |
| `language_outlined` | 1 | Language settings |
| `light_mode_outlined` | 1 | Light theme |
| `link_off` | 1 | Disconnect Instagram |
| `link_outlined` | 1 | Unverified Instagram link |
| `location_on_outlined` | 3 | Business or event location |
| `lock_outline` | 1 | Locked scheduled-post feature |
| `logout` | 2 | Sign out |
| `mail_outline` | 1 | Email a business |
| `manage_accounts_outlined` | 1 | Account settings |
| `menu` | 1 | Open the app drawer |
| `menu_book_outlined` | 1 | Project pattern/details |
| `more_horiz` | 1 | Context-menu trigger |
| `notes` | 1 | Text post type |
| `notifications` | 1 | Selected Notifications destination |
| `notifications_none` | 1 | Unknown notification category |
| `notifications_off_outlined` | 1 | Notifications disabled/empty state |
| `notifications_outlined` | 2 | Notifications destination/settings |
| `open_in_new` | 6 | External link |
| `palette_outlined` | 2 | Project colours or profile customisation |
| `people_outline` | 2 | Event audience/capacity |
| `person` | 1 | Selected Profile destination |
| `person_add_alt_1` | 1 | Add another account |
| `person_add_alt_outlined` | 2 | Follow notification/settings |
| `person_outline` | 1 | Unselected Profile destination |
| `person_search_outlined` | 2 | Instagram match/discovery |
| `photo_camera_outlined` | 2 | Change profile image or Instagram discovery |
| `play_circle_fill` | 2 | Play external/video media |
| `privacy_tip_outlined` | 1 | Privacy policy |
| `publish_outlined` | 1 | Event publication status |
| `push_pin` | 1 | Pinned state |
| `push_pin_outlined` | 2 | Pin action or unpinned state |
| `refresh` | 16 | Refresh or retry |
| `repeat` | 4 | Repost action, state, or notification |
| `schedule` | 1 | Selected Scheduled destination |
| `schedule_outlined` | 8 | Schedule/time or Scheduled destination |
| `search` | 5 | Search destination or input |
| `search_outlined` | 1 | Unselected Search destination |
| `send_outlined` | 1 | Publish immediately |
| `settings` | 1 | Selected Settings destination |
| `settings_outlined` | 5 | Settings destination/action |
| `short_text_rounded` | 2 | Image alt text |
| `show_chart` | 2 | Trending tag or growth settings |
| `storefront_outlined` | 1 | Business products settings |
| `switch_account` | 1 | Account switcher |
| `switch_account_outlined` | 2 | Switch-account action/settings |
| `translate` | 1 | Language empty state |
| `tune` | 1 | Project filters |
| `tune_outlined` | 1 | Project configuration |
| `upcoming_outlined` | 1 | Upcoming event |
| `verified_outlined` | 2 | Verified Instagram account |
| `video_file_outlined` | 1 | Video-file fallback |
| `volume_off_outlined` | 5 | Mute action/state/settings |
| `volume_up_outlined` | 2 | Unmute action/state |
| `warning_amber_rounded` | 2 | Warning feedback |

## Existing branded icons

The repository already contains these 24 px, rounded-line SVGs at a 1.5 px
stroke width:

- `crochet.svg`
- `embroidery.svg`
- `knitting.svg`
- `quilting.svg`
- `sewing.svg`
- `skein.svg`
- `spool.svg`
- `stitch.svg`

These are better candidates for craft taxonomy, onboarding, empty states, and
project-type accents than generic library glyphs. They should complement, not
replace, the interface icon set.

## Library options

Package information is current as of the audit date.

| Option | Strengths | Trade-offs | Fit |
| --- | --- | --- | --- |
| [`phosphor_icons` 3.0.1](https://pub.dev/packages/phosphor_icons) | 1,500+ icons; thin, light, regular, bold, fill, and duotone families; MIT; Flutter 3.43+; regular/fill variants remain compatible with `IconData` APIs | Community-maintained fork; much smaller Flutter-package adoption than Lucide; duotone rendering is package-specific | **Best visual fit.** Friendly rounded geometry and multiple weights suit the warm, tactile paper-cutout design better than Material's product-neutral forms |
| [`lucide_icons_flutter` 3.1.17](https://pub.dev/packages/lucide_icons_flutter) | Mature Flutter package; 167k downloads; MIT; consistent outline system; stroke variants and explicit RTL variants; plain `IconData` usage | Mostly outline-led, so selected states are less expressive; can feel technical and sparse beside CraftSky's chunky type and controls | **Best low-risk alternative.** Choose this if package adoption and a restrained visual system matter more than filled/duotone personality |
| [`tabler_icons_plus` 3.46.0](https://pub.dev/packages/tabler_icons_plus) | 6,250 icons; MIT; plain typed `IconData`; broad coverage makes one-to-one migration easiest | Very new Flutter wrapper with low adoption; enormous catalogue makes consistency harder; style is closer to utility/dashboard UI | **Best coverage, weakest brand fit.** Useful if exact semantic coverage is the deciding factor |

## Recommendation

Use **Phosphor** as the interface family and retain the existing custom SVGs as
the CraftSky-specific family.

Suggested style rules:

- Use `PhosphorIconsRegular` for passive metadata and unselected navigation.
- Use `PhosphorIconsBold` for icons inside buttons and tappable controls.
- Use `PhosphorIconsFill` for selected navigation, toggled states, and strong
  status signals. This preserves the app's current outline/filled state model.
- Reserve `PhosphorIconsDuotone` for larger empty states and feature moments,
  not dense toolbars or 16-20 px metadata icons.
- Use the custom craft SVGs only where the craft itself is the meaning. Do not
  mix them into generic actions such as close, delete, or settings.
- Keep destructive actions stylistically regular and communicate severity with
  colour and labels, rather than changing icon weight unpredictably.
- Use one semantic icon for each concept. In particular, consolidate the current
  `favorite_border`/`favorite_outline`, rounded/unrounded delete and arrow
  variants, and `tune`/`filter_list` overlap.

The package choice should be proven with a small visual spike before migrating
all 260 references. The highest-value comparison screen is the app shell plus a
post card: together they exercise navigation selection, social actions, menus,
settings, and compact metadata.

## Migration notes

1. Introduce a project-owned semantic icon catalogue, for example
   `CraftskyIcons.home`, `CraftskyIcons.like`, and `CraftskyIcons.delete`, rather
   than importing a third-party package throughout feature code.
2. Keep that first catalogue typed as `IconData`. This lets the existing
   `_DestinationSpec`, context-menu models, settings rows, profile stats, event
   rows, and icon buttons migrate without becoming widget factories.
3. Add stable keys or semantic labels before changing tests that currently use
   `find.byIcon` to locate controls. `more_horiz`, like, reply, repost, close,
   and bookmark are the most heavily coupled test icons.
4. SVG rendering is provided by `CraftIcon` and `CraftIconLabel`. Keep these
   widgets separate from generic `IconData` APIs and only map assets that have
   an unambiguous craft taxonomy value.
5. Migrate in slices: semantic catalogue and shell, social actions, common
   actions/feedback, settings and business features, then low-frequency states.
6. Remove `cupertino_icons` after confirming no near-term platform-specific use.

## Primary migration seams

- Shell selected/unselected pairs: `app/lib/router/app_shell.dart`
- Notification category mapping:
  `app/lib/notifications/widgets/notification_category_icon.dart`
- Snackbar severity mapping:
  `app/lib/shared/messaging/widgets/craftsky_snack_bar.dart`
- Settings trailing-icon mapping: `app/lib/settings/models/settings_row.dart`
- Business action mapping: `app/lib/business/models/business_action.dart`
- Shared `IconData` consumers: `app/lib/theme/chunky_icon_button.dart` and
  `app/lib/theme/craftsky_context_menu.dart`

Material's `uses-material-design: true` setting should remain during and after
the migration unless all Material-provided glyph dependencies, including any
inside dependencies, are deliberately removed and verified.
