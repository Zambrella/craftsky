# Flutter i18n Scaffold Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire `flutter_localizations` + `gen_l10n` into the Craftsky Flutter app so every user-facing string flows through `app_en.arb` → `AppLocalizations`, with generated source files checked in and English as the only locale.

**Architecture:** Single-template ARB (`app/lib/l10n/app_en.arb`) drives code generation into `app/lib/l10n/generated/` via `flutter: generate: true` + `l10n.yaml` with `synthetic-package: false` and `nullable-getter: false`. Three `MaterialApp*` constructors in `app.dart` receive `localizationsDelegates` + `supportedLocales`; titles switch from `title:` to `onGenerateTitle:` so they resolve inside the `Localizations` scope. Four existing source files migrate their literal strings to `AppLocalizations.of(context).<key>` calls.

**Tech Stack:** Flutter `flutter_localizations` (SDK), `intl ^0.20.2` (already in deps), `gen_l10n` (built into the Flutter SDK — no extra package).

---

## Spec

The approved design: [docs/superpowers/specs/2026-04-19-flutter-i18n-scaffold-design.md](../specs/2026-04-19-flutter-i18n-scaffold-design.md). Read it first.

## Binding rules

- [.claude/rules/flutter.md](../../../.claude/rules/flutter.md) — in particular: don't reintroduce `_build*` helpers when rewriting widget bodies; preserve `Theme.of(context)` usage; no `.withOpacity`.
- [.claude/rules/riverpod.md](../../../.claude/rules/riverpod.md) — no provider changes in this plan; still applies to any file you touch.

## Working directory

All paths are relative to the **repo root** (the worktree root: `/Users/douglastodd/Projects/craftsky/.claude/worktrees/elated-jackson-20b896`). Run Flutter/Dart commands from `app/`.

Base SHA (before this plan): whatever `git rev-parse HEAD` returns. As of writing, `12b138d` (the very_good_analysis commit).

## File map

- **Modify** `app/pubspec.yaml` — add `flutter_localizations` dep, add `generate: true`.
- **Create** `app/l10n.yaml` — codegen config.
- **Modify** `app/analysis_options.yaml` — add `analyzer.exclude: [lib/l10n/generated/**]`.
- **Create** `app/lib/l10n/app_en.arb` — seven keys.
- **Generated (committed)** `app/lib/l10n/generated/app_localizations.dart` + `app_localizations_en.dart` — produced by `flutter gen-l10n`; do not hand-edit.
- **Modify** `app/lib/app.dart` — three `MaterialApp*` call sites + `InitializationErrorScreen` body.
- **Modify** `app/lib/router/home_page.dart` — swap four literals for `l10n` lookups.
- **Modify** `app/lib/router/error_screen.dart` — swap two literals.
- **Modify** `app/test/widget_test.dart` — add one affirmative assertion that exercises `AppLocalizations`.
- **Unchanged**: `app/lib/main.dart` (release `ErrorWidget.builder` fallback stays English — spec Decisions block explains why), `app/lib/router/router.dart` (`'Unknown routing error'` stays English — debug-only fallback), `app/lib/bootstrap.dart`.

## Note on TDD

This is mechanical wiring + string extraction. Per the spec and the same reasoning as the original scaffold plan, I'm not going to structure this as red-green-refactor per file. The smoke test already asserts the full `App → HomePage` render; extending it with one `AppLocalizations.of(context).appTitle == 'Craftsky'` assertion is enough affirmative proof that the delegate is wired. `flutter analyze` + `flutter test` + `flutter build web` form the verification gate.

---

## Chunk 1: Full implementation

### Task 1: Add `flutter_localizations` and enable `generate:` in `pubspec.yaml`

**Files:**
- Modify: `app/pubspec.yaml`

- [ ] **Step 1: Add `flutter_localizations` to runtime deps (alphabetical)**

Under `dependencies:`, insert `flutter_localizations` directly after the existing `flutter: sdk: flutter` entry. Alphabetically `flutter_localizations` sorts after `flutter` and before `flutter_riverpod`, so place between them:

```yaml
  flutter:
    sdk: flutter
  flutter_localizations:
    sdk: flutter
  flutter_riverpod: ^3.0.3
```

- [ ] **Step 2: Set `generate: true`**

Under the top-level `flutter:` block at the bottom of the file, add `generate: true`:

```yaml
flutter:
  uses-material-design: true
  generate: true
```

- [ ] **Step 3: Do NOT commit yet**

`flutter pub get` fails here without a template ARB because `generate: true` triggers `gen_l10n` and it expects the ARB to exist. Defer `pub get` and the commit to the end of Task 4.

### Task 2: Create `l10n.yaml`

**Files:**
- Create: `app/l10n.yaml`

- [ ] **Step 1: Write `app/l10n.yaml`**

Exact contents:

```yaml
arb-dir: lib/l10n
template-arb-file: app_en.arb
output-localization-file: app_localizations.dart
output-class: AppLocalizations
output-dir: lib/l10n/generated
synthetic-package: false
nullable-getter: false
```

### Task 3: Exclude generated files from analyzer

**Files:**
- Modify: `app/analysis_options.yaml`

- [ ] **Step 1: Add `analyzer.exclude`**

The current file:

```yaml
analyzer:
  plugins:
    - custom_lint
```

Add `exclude:` under `analyzer:`:

```yaml
analyzer:
  plugins:
    - custom_lint
  exclude:
    - lib/l10n/generated/**
```

Leave the rest of the file (`formatter`, `linter.rules.public_member_api_docs: false`, the `include:`) untouched.

### Task 4: Create the ARB template

**Files:**
- Create: `app/lib/l10n/app_en.arb`

- [ ] **Step 1: Write the ARB file**

Exact contents:

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

- [ ] **Step 2: Run `flutter pub get` — triggers `gen_l10n`**

```bash
cd app && flutter pub get
```

Expected: resolves deps (now including `flutter_localizations`), then runs `gen_l10n`. On success, `app/lib/l10n/generated/app_localizations.dart` and `app/lib/l10n/generated/app_localizations_en.dart` exist.

If it errors:
- "Unable to find arb files" → you skipped Task 4 Step 1.
- "template-arb-file ... missing from arb-dir" → `l10n.yaml` or the filename in it is wrong.
- "Unsupported locale" → check `@@locale` in the ARB matches the filename suffix.

- [ ] **Step 3: Verify the generated files exist**

```bash
ls app/lib/l10n/generated/
```

Expected: `app_localizations.dart` and `app_localizations_en.dart`.

- [ ] **Step 4: Do NOT commit yet**

At this point `main.dart` still compiles against the pre-i18n app, but `MaterialApp` hasn't been wired to use the delegates. Commit happens at the end of Task 6 once all call sites are migrated and `flutter analyze` is clean.

### Task 5: Wire delegates into `MaterialApp` + migrate `InitializationErrorScreen`

**Files:**
- Modify: `app/lib/app.dart`

- [ ] **Step 1: Add import**

At the top of `app/lib/app.dart`, add (alphabetically — it sorts after `app_dependencies.dart`, before `router/router.dart`):

```dart
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
```

- [ ] **Step 2: Update `_ReadyApp`**

Find `_ReadyApp.build`. Replace the `MaterialApp.router(...)` invocation:

```dart
return MaterialApp.router(
  title: 'Craftsky',
  theme: AppTheme.lightThemeData,
  darkTheme: AppTheme.darkThemeData,
  themeMode: themeMode,
  debugShowCheckedModeBanner: false,
  routerConfig: router,
  builder: (context, child) {
    return TextScaleFactorClamper(
      child: FormFactorWidget(
        child: child ?? const SizedBox.shrink(),
      ),
    );
  },
);
```

with:

```dart
return MaterialApp.router(
  onGenerateTitle: (context) => AppLocalizations.of(context).appTitle,
  theme: AppTheme.lightThemeData,
  darkTheme: AppTheme.darkThemeData,
  themeMode: themeMode,
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  debugShowCheckedModeBanner: false,
  routerConfig: router,
  builder: (context, child) {
    return TextScaleFactorClamper(
      child: FormFactorWidget(
        child: child ?? const SizedBox.shrink(),
      ),
    );
  },
);
```

- [ ] **Step 3: Update `_LoadingApp`**

Replace `_LoadingApp.build`'s body with:

```dart
return MaterialApp(
  debugShowCheckedModeBanner: false,
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: const InitializationLoadingScreen(),
);
```

Note: `_LoadingApp` no longer qualifies for `const` on its `MaterialApp` because `AppLocalizations.localizationsDelegates` / `supportedLocales` are not const. That's fine — the widget still rebuilds at most once per deps-resolution.

- [ ] **Step 4: Update `_ErrorApp`**

Replace `_ErrorApp.build`'s body with:

```dart
return MaterialApp(
  debugShowCheckedModeBanner: false,
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: InitializationErrorScreen(
    error: error,
    onRetry: () => ref.invalidate(appDependenciesProvider),
  ),
);
```

- [ ] **Step 5: Migrate `InitializationErrorScreen`**

In `InitializationErrorScreen.build`, insert at the top of the method body (before `final theme = Theme.of(context);`):

```dart
final l10n = AppLocalizations.of(context);
```

Then:
- `Text('Initialization Failed', style: theme.textTheme.headlineSmall)` → `Text(l10n.initializationFailedTitle, style: theme.textTheme.headlineSmall)`
- `label: const Text('Retry')` → `label: Text(l10n.retryButton)` (drop `const` — the arg is no longer constant).

Leave `error.toString()` unchanged. Leave the `Icon` / sizes / paddings unchanged.

### Task 6: Migrate `HomePage` and `ErrorScreen`

**Files:**
- Modify: `app/lib/router/home_page.dart`
- Modify: `app/lib/router/error_screen.dart`

- [ ] **Step 1: Migrate `HomePage`**

Add at the top of `app/lib/router/home_page.dart` (alphabetically after `app_dependencies.dart`):

```dart
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
```

At the top of `build`, before `final theme = Theme.of(context);`:

```dart
final l10n = AppLocalizations.of(context);
```

Then replace the literal strings:
- `AppBar(title: const Text('Craftsky'))` → `AppBar(title: Text(l10n.appTitle))` (drop `const`).
- `Text('Craftsky', style: theme.textTheme.headlineMedium)` → `Text(l10n.appTitle, style: theme.textTheme.headlineMedium)`.
- `Text('Scaffold ready', style: theme.textTheme.titleMedium?.copyWith(...))` → `Text(l10n.homeSubtitle, style: theme.textTheme.titleMedium?.copyWith(...))` (keep the `.copyWith` call unchanged).
- `Text('v$version', style: theme.textTheme.bodySmall)` → `Text(l10n.homeVersionLabel(version), style: theme.textTheme.bodySmall)`.

- [ ] **Step 2: Migrate `ErrorScreen`**

Add at the top of `app/lib/router/error_screen.dart`:

```dart
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
```

At the top of `build`, before `final theme = Theme.of(context);`:

```dart
final l10n = AppLocalizations.of(context);
```

Replace:
- `Text('Something went wrong', style: theme.textTheme.headlineSmall)` → `Text(l10n.routingErrorTitle, style: theme.textTheme.headlineSmall)`.
- `label: const Text('Go home')` → `label: Text(l10n.goHomeButton)` (drop `const`).

Leave `error.toString()` unchanged.

- [ ] **Step 3: Run analyze**

```bash
cd app && flutter analyze
```

Expected: `No issues found!`. If failures — most likely unresolved imports — fix before proceeding.

- [ ] **Step 4: Run format**

```bash
cd app && dart format .
```

Expected: 0 or small number of files reformatted (trailing-comma preservation may shuffle things). Inspect the diff; no semantic changes.

### Task 7: Update smoke test

**Files:**
- Modify: `app/test/widget_test.dart`

- [ ] **Step 1: Add import**

Add to the top of the test file (alphabetically):

```dart
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
```

- [ ] **Step 2: Add assertion**

Inside the existing `testWidgets('App boots and renders HomePage', ...)` body, after the existing `expect(...)` calls, append:

```dart
expect(
  AppLocalizations.of(tester.element(find.byType(HomePage))).appTitle,
  'Craftsky',
);
```

This fails if the `AppLocalizations.delegate` is missing from `localizationsDelegates`, which is the actual wiring risk we're testing for.

- [ ] **Step 3: Run the test**

```bash
cd app && flutter test test/widget_test.dart
```

Expected: `+1: All tests passed!` (one test).

### Task 8: Full verification gate

From `app/`:

- [ ] **Step 1: Clean codegen re-run**

```bash
cd app && flutter gen-l10n
```

Expected: exits 0, no unexpected changes to `lib/l10n/generated/*`.

- [ ] **Step 2: Analyze**

```bash
cd app && flutter analyze
```

Expected: `No issues found!`.

- [ ] **Step 3: custom_lint**

```bash
cd app && dart run custom_lint
```

Expected: `No issues found!`. If `custom_lint` flags something in `lib/l10n/generated/`, the `analyzer.exclude` from Task 3 isn't propagating — fix by adding the path to `custom_lint`'s own config (at the plugin's recommended location) and report as a deviation.

- [ ] **Step 4: Test**

```bash
cd app && flutter test
```

Expected: `+5: All tests passed!` (4 theme tests + 1 widget test).

- [ ] **Step 5: Web build**

```bash
cd app && flutter build web
```

Expected: builds successfully (may take ~30s). This is the strongest compile-time check — it catches any i18n wiring regression web-only.

### Task 9: Single commit

- [ ] **Step 1: Stage everything and commit**

```bash
cd /Users/douglastodd/Projects/craftsky/.claude/worktrees/elated-jackson-20b896
git add app/pubspec.yaml app/pubspec.lock app/l10n.yaml app/analysis_options.yaml app/lib/l10n/ app/lib/app.dart app/lib/router/home_page.dart app/lib/router/error_screen.dart app/test/widget_test.dart
git commit -m "feat(app): wire flutter_localizations + gen_l10n, English ARB template

Adds flutter_localizations to runtime deps and enables flutter:
generate: true. l10n.yaml points gen_l10n at lib/l10n/app_en.arb,
outputting non-synthetic sources under lib/l10n/generated/ (committed).
Seven ARB keys cover every user-facing English string currently in
lib/; main.dart's release ErrorWidget fallback and router.dart's
unknown-routing-error fallback stay raw English (spec decisions).

MaterialApp.title becomes onGenerateTitle so it resolves inside the
Localizations scope; all three MaterialApp instances gain
localizationsDelegates + supportedLocales.

Smoke test now asserts AppLocalizations.of(...).appTitle, which
affirmatively exercises the delegate lookup."
```

---

## Done when

- `flutter_localizations` in `app/pubspec.yaml`, `generate: true` set.
- `app/l10n.yaml` present with documented values.
- `app/lib/l10n/app_en.arb` contains the seven keys from the spec.
- `app/lib/l10n/generated/app_localizations.dart` + `app_localizations_en.dart` committed.
- `app/analysis_options.yaml` excludes `lib/l10n/generated/**`.
- Every literal string named in the spec's "Context" section (excluding the three decisively-unlocalized cases) served from `AppLocalizations`.
- `flutter analyze` + `dart run custom_lint` + `flutter test` + `flutter build web` all clean.
- One commit on `claude/elated-jackson-20b896`.
