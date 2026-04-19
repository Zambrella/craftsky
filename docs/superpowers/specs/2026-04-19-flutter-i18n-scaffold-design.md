# Flutter i18n Scaffold — Design

**Date:** 2026-04-19
**Scope:** `app/` (Flutter client)
**Status:** Design

## Context

The Flutter app currently has no localization pipeline. All user-facing strings are hard-coded in Dart source:

- `HomePage` — "Craftsky", "Scaffold ready", "v1.0.0".
- `ErrorScreen` (from `GoRouter.errorBuilder`) — "Something went wrong", "Go home".
- `InitializationErrorScreen` (in `app.dart`) — "Initialization Failed", "Retry".
- `MaterialApp.title` — "Craftsky" in three places.

`pubspec.yaml` has `intl: ^0.20.2` (used by `appDependenciesProvider` for date formatting) but not `flutter_localizations`. `bootstrap.dart` sets `Intl.defaultLocale` from `PlatformDispatcher.instance.locale`. `MaterialApp.router` has no `localizationsDelegates` or `supportedLocales`.

This spec lays the Flutter [i18n](https://docs.flutter.dev/ui/internationalization) pipeline so every future user-facing string flows through `.arb` → `gen_l10n` → `AppLocalizations`. It ships with English only; additional locales land as sibling `.arb` files when translators arrive, with no consumer-side code changes.

## Decisions at a glance

- **English only.** Single template `app_en.arb`, language-code `en` (no country variant). Regional variants like `en_GB` / `en_US` become sibling files later.
- **Device locale only.** `MaterialApp` picks from `supportedLocales` using the OS locale. No user-facing picker, no persistence, no `LocaleNotifier`. That plumbing lands with the first translator.
- **`gen_l10n` with real (non-synthetic) output files.** `synthetic-package: false`, output committed to `lib/l10n/generated/`. Consistent with the rest of the scaffold's codegen (`*.g.dart`, `*.mapper.dart`, `router.g.dart`) — all checked in. Gives IDE "go to definition," avoids the magic `package:flutter_gen/` path.
- **`nullable-getter: false`.** `AppLocalizations.of(context)` returns non-nullable. Safe because the delegate is registered on every `MaterialApp` in `app.dart`.
- **Diagnostic strings stay raw English.** Log messages, `bootstrap.dart` output, exception `toString()`, and the `errorBuilder` fallback (`'Unknown routing error'`, shown only when `GoRouterState.error` is null — a bug indicator, not user-facing copy) are not localized.

## Out of scope

- Second locale.
- `LocaleNotifier` + settings UI for user-selectable locale.
- RTL-specific verification (no RTL locale in this pass).
- iOS Xcode `Runner` localizations registration (App Store metadata; not needed for render correctness).
- `use-deferred-loading: true` for web (useful with many locales; premature with one).
- Localizing `error.toString()` output from exceptions.

## Section 1 — Dependencies & configuration

### `app/pubspec.yaml`

Add to runtime dependencies (alphabetical):

```yaml
  flutter_localizations:
    sdk: flutter
```

`intl: ^0.20.2` is already present. Add under the `flutter:` section:

```yaml
flutter:
  uses-material-design: true
  generate: true
```

`generate: true` tells `flutter pub get` and `flutter run` to invoke `gen_l10n` automatically.

### `app/l10n.yaml` (new)

Placed at the `app/` root (sibling of `pubspec.yaml`):

```yaml
arb-dir: lib/l10n
template-arb-file: app_en.arb
output-localization-file: app_localizations.dart
output-class: AppLocalizations
output-dir: lib/l10n/generated
synthetic-package: false
nullable-getter: false
```

### `app/analysis_options.yaml`

Add an analyzer exclude so very_good_analysis doesn't lint generated localization files:

```yaml
analyzer:
  plugins:
    - custom_lint
  exclude:
    - lib/l10n/generated/**
```

## Section 2 — ARB template

### `app/lib/l10n/app_en.arb`

Seven keys, covering every user-facing English string in `lib/`:

```json
{
  "@@locale": "en",

  "appTitle": "Craftsky",
  "@appTitle": { "description": "The app's title, used in MaterialApp.title and as an AppBar title." },

  "homeSubtitle": "Scaffold ready",
  "@homeSubtitle": { "description": "Muted subtitle on the placeholder HomePage." },

  "homeVersionLabel": "v{version}",
  "@homeVersionLabel": {
    "description": "Renders the running app version below the subtitle on HomePage.",
    "placeholders": { "version": { "type": "String", "example": "1.0.0" } }
  },

  "initializationFailedTitle": "Initialization Failed",
  "@initializationFailedTitle": { "description": "Headline on InitializationErrorScreen when appDependenciesProvider fails." },

  "retryButton": "Retry",
  "@retryButton": { "description": "Retry-action button label on InitializationErrorScreen." },

  "routingErrorTitle": "Something went wrong",
  "@routingErrorTitle": { "description": "Headline on ErrorScreen (from GoRouter.errorBuilder)." },

  "goHomeButton": "Go home",
  "@goHomeButton": { "description": "Button label on routing ErrorScreen returning to HomeRoute." }
}
```

Nothing in `main.dart` or `bootstrap.dart` is user-facing (log messages, platform init output, and runtime errors). Localizing diagnostic logs is an anti-pattern and would actively hinder debugging.

## Section 3 — Wiring & call-site migrations

### `app/lib/app.dart`

Add:

```dart
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
```

On all three `MaterialApp*` constructors (`_ReadyApp`, `_LoadingApp`, `_ErrorApp`):

- Add `localizationsDelegates: AppLocalizations.localizationsDelegates`.
- Add `supportedLocales: AppLocalizations.supportedLocales`.
- Replace `title: 'Craftsky'` with `onGenerateTitle: (context) => AppLocalizations.of(context).appTitle`. `title:` runs outside the `Localizations` scope so `AppLocalizations.of` isn't yet available; `onGenerateTitle:` runs inside it.

In `InitializationErrorScreen.build`:

- `final l10n = AppLocalizations.of(context);` at the top.
- `'Initialization Failed'` → `l10n.initializationFailedTitle`.
- `'Retry'` → `l10n.retryButton`.

`error.toString()` stays as-is.

### `app/lib/router/home_page.dart`

- Add `import 'package:craftsky_app/l10n/generated/app_localizations.dart';`.
- `final l10n = AppLocalizations.of(context);` at the top of `build`.
- `AppBar(title: const Text('Craftsky'))` → `AppBar(title: Text(l10n.appTitle))`.
- `Text('Craftsky', style: …)` → `Text(l10n.appTitle, style: …)`.
- `Text('Scaffold ready', …)` → `Text(l10n.homeSubtitle, …)`.
- `Text('v$version', …)` → `Text(l10n.homeVersionLabel(version), …)`.

### `app/lib/router/error_screen.dart`

- Add the import and `final l10n = AppLocalizations.of(context);`.
- `'Something went wrong'` → `l10n.routingErrorTitle`.
- `'Go home'` → `l10n.goHomeButton`.

`error.toString()` stays as-is.

### `app/lib/router/router.dart`

The `errorBuilder` fallback stays unchanged:

```dart
errorBuilder: (context, state) =>
    ErrorScreen(error: state.error ?? 'Unknown routing error'),
```

`'Unknown routing error'` is a debug-only fallback — it renders only when `GoRouterState.error` is null, which should never happen in a correct app. Treating it as user copy and localizing it would wire through `context` and add a key nobody should ever read. Leave as-is.

### `app/lib/bootstrap.dart`

No changes. `Intl.defaultLocale` setup remains (used by `intl` for date/number formatting), and `MaterialApp` will still receive the device locale through `PlatformDispatcher` automatically.

## Section 4 — Verification & testing

### Codegen & generated files

`flutter pub get` triggers `gen_l10n`. One-off: `flutter gen-l10n`. Generated files, committed to git:

- `app/lib/l10n/generated/app_localizations.dart`
- `app/lib/l10n/generated/app_localizations_en.dart`

### Smoke test update

`app/test/widget_test.dart`'s existing assertion `expect(find.text('Craftsky'), findsWidgets)` keeps passing because English resolves to the same literal. Add an affirmative assertion that `Localizations` is wired:

```dart
expect(
  Localizations.localeOf(tester.element(find.byType(HomePage))),
  isNotNull,
);
```

This is one line in the existing test — not a new test file.

### Final verification gate

After implementation, in `app/`:

1. `flutter pub get` — generates `lib/l10n/generated/*.dart` without error.
2. `flutter analyze` — clean.
3. `dart run custom_lint` — clean.
4. `flutter test` — 5/5 passing (4 theme + 1 smoke, now with the l10n assertion).
5. `flutter build web` — succeeds.
6. `flutter run` smoke-check (by user): `HomePage` renders identical to before.

## Done when

- `flutter_localizations` in pubspec; `generate: true` set.
- `l10n.yaml` present with documented values.
- `app/lib/l10n/app_en.arb` contains the seven keys listed in Section 2.
- Every user-facing string listed in Context is served from `AppLocalizations`.
- `MaterialApp` instances wire `localizationsDelegates`, `supportedLocales`, and `onGenerateTitle`.
- `lib/l10n/generated/**` excluded from analyzer lints.
- Verification commands above all pass.
