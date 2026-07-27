# UI Polish Notes: Instagram DM Verification And Follow Discovery

## Summary

Aligned the Find people from Instagram page with Notification Settings and the
CraftSky design system without changing its server-side verification, import,
or suggestion behavior. Routed transient success and failure feedback through
the shared CraftSky in-app messenger so it uses the app-wide severity, replacement,
duration, and accessibility behavior. Clarified the pending-confirmation account
label, made the existing cancellation path available at that stage, and moved
the discovery choice into a clearer account-first decision layout.
The import card now follows the same selector-and-explanation pattern, uses a
primary manual-import action, and presents Notification Settings as an inline
link. Instagram settings cards retain their themed outline without clipping
their contents to the rounded card shape. The verified-account discovery
control now matches the standard Notification Settings switch row, and
Instagram Export appears first in the import selector while remaining the
default. Segmented controls now use the existing CraftSky moss swatch, its
explicit white `onMoss` foreground contract, and matching non-red interaction
overlays through the global app theme. Unselected segment labels and icons use
the theme's standard `onSurface` text colour.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User request: match Notification Settings | Centered the scroll content at a 720px maximum width and moved page/card spacing to `SpacingTheme` tokens | `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Done |
| UIP-002 | User request: use themed widgets | Replaced Material cards, progress spinners, direction dropdown, and manual text field with `CraftskyCard`, `StitchProgressIndicator`, `CraftskySingleSelectInput`, and `CraftskyMultilineTextInput` | `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Done |
| UIP-003 | Notification Settings visual pattern | Added primary-colour section icons, theme typography, and secondary surface text for supporting copy | `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Done |
| UIP-004 | Themed controls inside paper cards | Added a transparent Material surface inside `CraftskyCard` so list tiles, switches, checkboxes, and select overlays render ink/background behavior correctly | `app/lib/theme/craftsky_card.dart` | Done |
| UIP-005 | Regression coverage | Updated the page harness to use `AppTheme`, assert themed cards replace raw cards, and exercise the following-only import composer without a follower selector | `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-006 | User request: use CraftSky notifications | Replaced page-local snackbars for challenge copy, import completion, and failed actions with semantic `AppMessenger` info/error messages; added recording-messenger assertions for copy and import success | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-007 | User request: clarify the found account | Changed the candidate sentence to `Account: @…` and rendered only the handle in bold while preserving localized sentence ordering | `app/lib/l10n/app_en.arb`, generated localization output, `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Done |
| UIP-008 | User request: reject/cancel at confirmation | Added a `Cancel verification` action to the pending-confirmation state using the existing owned-attempt cancellation flow; covered cancellation and fresh retry in the widget test | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-009 | User request: account-first discovery choice | Moved the discovery selector directly below `Account: @…`, defaulted it to `Allow discovery`, required one option to remain selected, and made the paragraph beneath switch between discovery and private explanations | `app/lib/l10n/app_en.arb`, generated localization output, `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-010 | User request: emphasize linked Instagram account | Reused the localized handle-span rendering so only `@handle` is bold in `Linked as @…` | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-011 | User request: remove the rounded clip inside settings cards | Disabled clipping on the Instagram page's `CraftskyCard` instances while retaining their themed rounded decoration and border | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-012 | User request: inline Notification Settings action | Replaced the separate text button with underlined, primary-colour `Notification settings` link text inside the notification explanation | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/lib/l10n/app_en.arb`, generated localization output, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-013 | User request: primary Import handles action | Changed the manual `Import handles` action from an outlined button to the theme's primary filled button | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-014 | User request: selected import option explanation | Moved import guidance below the input selector and made it switch between manual-entry and JSON-file-specific copy | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/lib/l10n/app_en.arb`, generated localization output, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-015 | User request: full-width verification actions | Replaced the wrapping challenge action group with a stretched vertical column so copy, open-DM, and cancel actions use the card's full content width | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-016 | User request: destructive account/import actions | Styled `Revoke Instagram link` with the theme error colour and replaced the import's bottom text action with a trailing error-coloured bin icon retaining the localized tooltip and semantics | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-017 | User request: confirm link revocation | Routed `Revoke Instagram link` through the themed destructive confirmation dialog, clearly disclosed import deletion and accepted-follow retention, and verified that cancel is inert while confirm submits exactly one revoke | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/lib/l10n/app_en.arb`, generated localization output, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-018 | User request: match revoke icon colour | Explicitly applied the theme error colour to the revoke button's link icon as well as its text, preventing the button theme's icon colour from overriding the destructive treatment | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-019 | User request: simplify the discovery switch | Removed the selected-state helper paragraph beneath `Let others find me by my Instagram username` on the verified-account screen | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-020 | User clarification: use the primary colour for the switch | Initially replaced the success-colour thumb and track with the theme primary colour; superseded by UIP-022 when the user clarified that the switch should exactly match Notification Settings | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Superseded |
| UIP-021 | User request: put Instagram Export first | Reordered the import selector to show `Instagram Export` before `Enter handles` and preserved Instagram Export as the initial selection | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-022 | User request: match Notification Settings switch styling | Replaced the bespoke `SwitchListTile` and colour overrides with the same merged title-row and plain theme-driven `Switch` pattern used by Notification Settings | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-023 | User request: green selected import segment | Initially styled only the import selector with the CraftSky moss swatch; superseded by UIP-024 when the user requested a global theme fix and the explicit moss foreground contract | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Superseded |
| UIP-024 | User request: globally themed segmented controls | Added global light/dark segmented-button styling with moss selected backgrounds, an explicit white `onMoss` foreground for text and icons, and moss/white pressed, focused, and hovered overlays so the red accent cannot leak into interactions; removed the page-local style | `app/lib/theme/app_theme.dart`, `app/lib/theme/theme_extensions.dart`, `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/theme/app_theme_test.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-025 | User request: default-colour unselected segment text | Explicitly resolved unselected segmented-button labels and icons to each theme's `colorScheme.onSurface`, preventing inherited primary/accent colouring while retaining the selected `onMoss` contract | `app/lib/theme/app_theme.dart`, `app/test/theme/app_theme_test.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |

## Verification

- Commands run:
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart lib/theme/craftsky_card.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter test --reporter compact test/instagram_migration/instagram_migration_page_test.dart test/notifications/notification_settings_page_test.dart`
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter test --reporter compact test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter gen-l10n`
  - `flutter test --reporter compact test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `git diff --check`
  - `flutter test --reporter compact test/theme/app_theme_test.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter analyze lib/theme/app_theme.dart lib/theme/theme_extensions.dart lib/instagram_migration/pages/instagram_migration_page.dart test/theme/app_theme_test.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `dart format lib/instagram_migration/pages/instagram_migration_page.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter test test/instagram_migration/instagram_migration_page_test.dart --plain-name 'FR-024 candidate defaults to discovery and explains choices'`
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter test --reporter compact test/instagram_migration/instagram_migration_page_test.dart`
  - `git diff --check`
  - `flutter test test/instagram_migration/instagram_migration_page_test.dart --plain-name 'FR-024 linked Instagram handle is bold'`
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter test --reporter compact test/instagram_migration/instagram_migration_page_test.dart`
  - `git diff --check`
  - `flutter gen-l10n`
  - `flutter test test/instagram_migration/instagram_migration_page_test.dart --plain-name 'FR-024 candidate requires an explicit discovery choice'`
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter test --reporter compact test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter test test/instagram_migration/instagram_migration_page_test.dart --plain-name 'FR-024 challenge can be copied opened and cancelled'`
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `git diff --check`
  - `flutter test test/instagram_migration/instagram_migration_page_test.dart --plain-name 'FR-024 linked account and import destructive actions use error styling'`
  - `flutter gen-l10n`
  - `flutter test test/instagram_migration/instagram_migration_page_test.dart --plain-name 'FR-024 revoking an Instagram link requires confirmation'`
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `git diff --check`
  - `flutter test --reporter compact test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `git diff --check`
- Passing evidence:
  - Focused analyzer completed with no issues.
  - Four Instagram page widget tests and the neighboring Notification Settings widget test passed.
  - The focused messenger regression run passed all four Instagram page widget tests.
  - The focused candidate-account regression proves the localized sentence, bold handle span, cancellation action, cancelled state, and fresh retry behavior.
  - The final focused analyzer passed with no issues and all five Instagram page widget tests passed.
  - The account-first discovery regression verifies default and non-empty selection, vertical ordering, both option-specific explanations, confirmation value, and cancel/retry behavior; the final diff check is clean.
  - The linked-account regression verifies the full localized sentence and bold handle span; the focused analyzer and all six page widget tests pass.
  - The import-card regression verifies unclipped page cards, the selector-specific description and its vertical position, the primary manual import action, and the inline clickable Notification Settings text; all six page widget tests pass.
  - The challenge-action regression verifies copy, open-DM, and cancellation buttons share one full-width vertical layout while retaining their existing actions.
  - The destructive-action regression verifies the revoke text, revoke link icon, and trailing import bin use the theme error colour, the old delete text button is absent, and the bin icon retains its localized tooltip.
  - The revoke-confirmation regression verifies the dialog copy, inert cancellation, retained linked-account UI after cancellation, exactly one confirmed repository call, and linked-account removal after success.
  - The final Instagram page suite passes all 11 widget tests, including assertions that the verified-account control uses the same unoverridden `Switch` styling as Notification Settings and that the import selector has no page-local style.
  - Dedicated light- and dark-theme tests verify moss selected backgrounds, white `onMoss` text/icon foregrounds, moss interaction overlays for unselected segments, and white interaction overlays for selected segments.
  - Light, dark, and Instagram assertions verify that unselected segment text and icons resolve to the theme's standard `onSurface` colour rather than a primary or accent colour.
  - The focused analyzer reports no issues across the shared theme, Instagram page, and their tests; the final diff check is clean.
- Skipped checks and reason:
  - Physical-device visual/accessibility review remains a documented external release gate.

## Scope Guardrails

- Requirement behavior changed: Yes — the user explicitly changed the pending-confirmation selector from requiring a manual choice to defaulting to `Allow discovery` and required destructive confirmation before revocation; `01-requirements.md`, `02-acceptance-tests.md`, and implementation evidence were updated.
- Business logic changed: No
- APIs, data models, migrations, permissions, or dependencies changed: No
- Notes: The shared `CraftskyCard` adjustment only supplies the Material surface required by existing interactive descendants; its decoration and public API are unchanged. The latest clipping change is page-local and does not change the shared card default. The pending-confirmation cancellation button calls the already implemented verification cancellation operation and does not alter its API or state semantics. The new default changes only the displayed client choice; AppView still requires explicit confirmation carrying a boolean value. Revocation still uses the existing AppView operation; the new dialog only adds explicit client confirmation before invoking it. The segmented-button styling is intentionally global per the user request and affects visual and interaction states only.

## Follow-ups

- [ ] Validate compact-screen layout, screen-reader flow, file picker, push open, and external Instagram DM launch on a physical device before release.
