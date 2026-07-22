# UI Polish Notes: Instagram DM Verification And Follow Discovery

## Summary

Aligned the Find people from Instagram page with Notification Settings and the
CraftSky design system without changing its server-side verification, import,
or suggestion behavior. Routed transient success and failure feedback through
the shared CraftSky in-app messenger so it uses the app-wide severity, replacement,
duration, and accessibility behavior. Clarified the pending-confirmation account
label, made the existing cancellation path available at that stage, and moved
the discovery choice into a clearer account-first decision layout.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User request: match Notification Settings | Centered the scroll content at a 720px maximum width and moved page/card spacing to `SpacingTheme` tokens | `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Done |
| UIP-002 | User request: use themed widgets | Replaced Material cards, progress spinners, direction dropdown, and manual text field with `CraftskyCard`, `StitchProgressIndicator`, `CraftskySingleSelectInput`, and `CraftskyMultilineTextInput` | `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Done |
| UIP-003 | Notification Settings visual pattern | Added primary-colour section icons, theme typography, and secondary surface text for supporting copy | `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Done |
| UIP-004 | Themed controls inside paper cards | Added a transparent Material surface inside `CraftskyCard` so list tiles, switches, checkboxes, and select overlays render ink/background behavior correctly | `app/lib/theme/craftsky_card.dart` | Done |
| UIP-005 | Regression coverage | Updated the page harness to use `AppTheme`, assert themed cards replace raw cards, and exercise the themed direction selector | `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-006 | User request: use CraftSky notifications | Replaced page-local snackbars for challenge copy, import completion, and failed actions with semantic `AppMessenger` info/error messages; added recording-messenger assertions for copy and import success | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-007 | User request: clarify the found account | Changed the candidate sentence to `Account: @…` and rendered only the handle in bold while preserving localized sentence ordering | `app/lib/l10n/app_en.arb`, generated localization output, `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Done |
| UIP-008 | User request: reject/cancel at confirmation | Added a `Cancel verification` action to the pending-confirmation state using the existing owned-attempt cancellation flow; covered cancellation and fresh retry in the widget test | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-009 | User request: account-first discovery choice | Moved the discovery selector directly below `Account: @…`, defaulted it to `Allow discovery`, required one option to remain selected, and made the paragraph beneath switch between discovery and private explanations | `app/lib/l10n/app_en.arb`, generated localization output, `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |
| UIP-010 | User request: emphasize linked Instagram account | Reused the localized handle-span rendering so only `@handle` is bold in `Linked as @…` | `app/lib/instagram_migration/pages/instagram_migration_page.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` | Done |

## Verification

- Commands run:
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart lib/theme/craftsky_card.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter test --reporter compact test/instagram_migration/instagram_migration_page_test.dart test/notifications/notification_settings_page_test.dart`
  - `flutter analyze lib/instagram_migration/pages/instagram_migration_page.dart test/instagram_migration/instagram_migration_page_test.dart`
  - `flutter test --reporter compact test/instagram_migration/instagram_migration_page_test.dart`
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
- Passing evidence:
  - Focused analyzer completed with no issues.
  - Four Instagram page widget tests and the neighboring Notification Settings widget test passed.
  - The focused messenger regression run passed all four Instagram page widget tests.
  - The focused candidate-account regression proves the localized sentence, bold handle span, cancellation action, cancelled state, and fresh retry behavior.
  - The final focused analyzer passed with no issues and all five Instagram page widget tests passed.
  - The account-first discovery regression verifies default and non-empty selection, vertical ordering, both option-specific explanations, confirmation value, and cancel/retry behavior; the final diff check is clean.
  - The linked-account regression verifies the full localized sentence and bold handle span; the focused analyzer and all six page widget tests pass.
- Skipped checks and reason:
  - Physical-device visual/accessibility review remains a documented external release gate.

## Scope Guardrails

- Requirement behavior changed: Yes — the user explicitly changed the pending-confirmation selector from requiring a manual choice to defaulting to `Allow discovery`; `01-requirements.md`, `02-acceptance-tests.md`, and implementation evidence were updated.
- Business logic changed: No
- APIs, data models, migrations, permissions, or dependencies changed: No
- Notes: The shared `CraftskyCard` adjustment only supplies the Material surface required by existing interactive descendants; its decoration and public API are unchanged. The pending-confirmation cancellation button calls the already implemented verification cancellation operation and does not alter its API or state semantics. The new default changes only the displayed client choice; AppView still requires explicit confirmation carrying a boolean value.

## Follow-ups

- [ ] Validate compact-screen layout, screen-reader flow, file picker, push open, and external Instagram DM launch on a physical device before release.
