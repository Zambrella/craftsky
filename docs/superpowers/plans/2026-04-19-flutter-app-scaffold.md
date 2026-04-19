# Flutter App Scaffold Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `flutter create` default in `app/` with a minimal production scaffold: entry point + bootstrap + async app-dependencies provider + go_router + FlexColorScheme theming + form-factor/text-scale clampers.

**Architecture:** Five layers, each in its own directory or top-level file: (1) `main.dart` + `bootstrap.dart` for entry/platform init, (2) `app_dependencies.dart` for one-shot async init gated by a loading/error UI, (3) `router/` with go_router_builder typed routes, (4) `theme/` with FlexColorScheme + theme extensions + persisted `ThemeMode`, (5) responsive primitives (`FormFactorWidget`, `TextScaleFactorClamper`) in `theme/`. Pattern ported from `squiddies_flutter`.

**Tech Stack:** Flutter 3.x, Dart 3.x, Riverpod 3.x (`@riverpod` codegen), go_router 17 + go_router_builder, FlexColorScheme 8, dart_mappable, dynamic_color, logging, shared_preferences, package_info_plus, device_info_plus, pub_semver, intl.

## Spec

The approved design doc is [docs/superpowers/specs/2026-04-19-flutter-app-scaffold-design.md](../specs/2026-04-19-flutter-app-scaffold-design.md). Read it first — this plan assumes familiarity with it.

## Binding rules

The Flutter files in this plan must comply with:

- `.claude/rules/flutter.md` — one widget class per concern; no `_build*` helpers; `Theme.of(context)` for colors/text; no `.withOpacity()`; immutable models via `dart_mappable`/`freezed`; `logging` over `print`.
- `.claude/rules/riverpod.md` — `@riverpod` codegen; `ref.watch` in build / `ref.read` in callbacks; switch-over-`.when()`; check `ref.mounted` after awaits; keep notifier state through the `state` property only.

Also note the architectural rule from [AGENTS.md](../../../AGENTS.md): the Flutter app talks to the App View via HTTPS + session token only, never to the PDS directly. This scaffold adds no network code — just keep the constraint in mind so no PDS client sneaks in.

## Working directory

All paths below are relative to the **repo root** (the worktree root). The Flutter app lives under `app/`. Run Flutter/Dart commands from `app/` unless stated otherwise.

## Note on TDD for scaffold code

This plan is an 80%-new-files scaffold: entry point, bootstrap, provider declarations, theme data, route declarations. For code that is mostly declarative wiring, a red-green-refactor loop per file produces performative tests that assert `class exists` without verifying behavior. I've kept real tests where there's real behavior to verify — the widget-level smoke test (Chunk 3) and the `ThemeModeNotifier` persistence round-trip (Chunk 2). Everything else is verified by `dart run build_runner build`, `flutter analyze`, and the smoke test at the end of Chunk 3. If you find yourself writing a "test" that just pumps a widget tree and asserts it doesn't crash, you're duplicating the smoke test — skip it.

---

## Chunk 1: Pubspec, skeleton directories, and entry point

Gets the project off `flutter create` defaults, installs dependencies, lays down empty directories, and writes `main.dart` + `bootstrap.dart` + a stub `App` widget. At the end of this chunk, `flutter analyze` passes and `flutter run` launches into a blank placeholder screen.

### Task 1.1: Rewrite `app/pubspec.yaml` with the scaffold dependencies

**Files:**
- Modify: `app/pubspec.yaml`

- [ ] **Step 1: Rewrite pubspec.yaml**

Replace the `dependencies:` and `dev_dependencies:` sections. Keep `name: craftsky_app`, `description`, `publish_to: 'none'`, `version: 1.0.0+1`, and the `environment:` block. Keep the `flutter:` section with `uses-material-design: true` but delete the commented boilerplate.

Full intended file contents:

```yaml
name: craftsky_app
description: "Craftsky — crafting-focused social platform on AT Protocol."
publish_to: 'none'
version: 1.0.0+1

environment:
  sdk: ^3.11.0

dependencies:
  flutter:
    sdk: flutter
  cupertino_icons: ^1.0.8

  # State management
  flutter_riverpod: ^3.0.3
  riverpod_annotation: ^4.0.0

  # Routing
  go_router: ^17.0.0

  # Platform / init
  logging: ^1.3.0
  shared_preferences: ^2.5.4
  package_info_plus: ^9.0.0
  device_info_plus: ^12.3.0
  pub_semver: ^2.2.0
  intl: ^0.20.2

  # Theming
  flex_color_scheme: ^8.4.0
  dynamic_color: ^1.8.1

  # Serialization
  dart_mappable: ^4.6.1

dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^6.0.0

  # Code generation
  build_runner: ^2.10.4
  riverpod_generator: ^4.0.0+1
  riverpod_lint: ^3.0.3
  custom_lint: ^0.8.1
  go_router_builder: ^4.1.3
  dart_mappable_builder: ^4.6.3

flutter:
  uses-material-design: true
```

- [ ] **Step 2: Install dependencies**

Run (from `app/`): `flutter pub get`

Expected: resolves cleanly. If any version is unavailable on pub.dev at implementation time, bump to the latest minor that resolves and note it in the commit message.

- [ ] **Step 3: Commit**

```bash
git add app/pubspec.yaml app/pubspec.lock
git commit -m "feat(app): add scaffold dependencies to pubspec"
```

### Task 1.2: Lay down the empty directory structure

**Files:**
- Create: `app/lib/router/widgets/.gitkeep`
- Create: `app/lib/shared/widgets/.gitkeep`

(`app/lib/router/` and `app/lib/theme/` will be created implicitly by later tasks — no `.gitkeep` needed for those since they'll immediately contain code.)

- [ ] **Step 1: Create .gitkeep files**

Both files have empty contents. Use the `Write` tool with empty string contents, or:

```bash
mkdir -p app/lib/router/widgets app/lib/shared/widgets
touch app/lib/router/widgets/.gitkeep app/lib/shared/widgets/.gitkeep
```

- [ ] **Step 2: Commit**

```bash
git add app/lib/router/widgets/.gitkeep app/lib/shared/widgets/.gitkeep
git commit -m "chore(app): reserve router/widgets and shared/widgets dirs"
```

### Task 1.3: Write `main.dart`

**Files:**
- Modify: `app/lib/main.dart` (currently the counter demo)

- [ ] **Step 1: Replace `main.dart` with the new entry point**

Full file contents:

```dart
import 'dart:async';
import 'dart:developer' as developer;

import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:logging/logging.dart';

final _log = Logger('main');

Future<void> main() async {
  await runZonedGuarded(
    () async {
      final binding = WidgetsFlutterBinding.ensureInitialized();

      // Configure logging before anything else so error handlers and
      // bootstrap can both log through the root logger.
      Logger.root.level = Level.FINE;
      Logger.root.onRecord.listen((record) {
        if (kDebugMode) {
          // ignore: avoid_print
          print('${record.level.name} | ${record.loggerName}: ${record.message}');
          if (record.error != null) {
            // ignore: avoid_print
            print('  error: ${record.error}');
          }
          if (record.stackTrace != null) {
            // ignore: avoid_print
            print('  stack: ${record.stackTrace}');
          }
        }
      });

      registerErrorHandlers();

      await bootstrap(binding);
    },
    (Object error, StackTrace stack) {
      // Last-resort sink: use dart:developer log because logging may not be
      // fully wired yet depending on where the crash originates.
      developer.log(
        'runZonedGuarded: $error',
        name: 'main',
        error: error,
        stackTrace: stack,
        level: 1000,
      );
      _log.severe('runZonedGuarded caught error', error, stack);
    },
  );
}

void registerErrorHandlers() {
  final log = Logger('ErrorHandlers');

  FlutterError.onError = (FlutterErrorDetails details) {
    FlutterError.presentError(details);
    log.severe('FlutterError: ${details.exception}', details.exception, details.stack);
  };

  PlatformDispatcher.instance.onError = (Object error, StackTrace stack) {
    log.severe('Platform error', error, stack);
    return true;
  };

  ErrorWidget.builder = (FlutterErrorDetails details) {
    log.warning('Error building widget: ${details.exception}', details.exception, details.stack);
    if (kDebugMode) {
      return ErrorWidget(details.exception);
    }
    return const ColoredBox(
      color: Colors.red,
      child: Center(
        child: Text(
          'An error occurred rendering this element',
          style: TextStyle(color: Colors.white),
        ),
      ),
    );
  };
}
```

Notes:
- The two `// ignore: avoid_print` directives inside the debug-only log sink are the single place in the codebase where `print` is acceptable; everywhere else use `Logger`.
- `registerErrorHandlers` uses its own `Logger` instance — that's fine, all loggers funnel through `Logger.root`.

- [ ] **Step 2: Leave commit for end of chunk 1 once bootstrap + stub App exist**

`main.dart` imports `bootstrap.dart` and `app.dart` (transitively) that don't exist yet — don't try to `flutter analyze` until Task 1.5.

### Task 1.4: Write `bootstrap.dart`

**Files:**
- Create: `app/lib/bootstrap.dart`

- [ ] **Step 1: Write bootstrap.dart**

```dart
import 'dart:async';
import 'dart:ui';

import 'package:craftsky_app/app.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
// ignore: depend_on_referenced_packages
import 'package:flutter_web_plugins/url_strategy.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:logging/logging.dart';
import 'package:shared_preferences/shared_preferences.dart';

final _log = Logger('bootstrap');

/// Runs platform / Flutter init before `runApp`.
///
/// IMPORTANT: must never throw in production. Anything that *can* fail
/// belongs in `appDependenciesProvider`, which has loading/error UI.
Future<void> bootstrap(WidgetsBinding widgetsBinding) async {
  _log.fine('bootstrap starting');

  // Web: path URL strategy (no `#` in URLs).
  usePathUrlStrategy();

  if (kIsWeb) {
    _log.fine('web detected, skipping native init');
    runApp(
      const ProviderScope(child: App()),
    );
    return;
  }

  // Default locale for intl date/number formatting.
  final localeName = PlatformDispatcher.instance.locale.toString();
  Intl.defaultLocale = localeName;
  _log.fine('Intl.defaultLocale=$localeName');

  // dart_mappable mapper init — empty for now, grows as models are added.
  initializeMappers();

  // Namespace all shared_preferences keys.
  SharedPreferences.setPrefix('craftsky.');

  if (defaultTargetPlatform == TargetPlatform.android) {
    await SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    SystemChrome.setSystemUIOverlayStyle(
      const SystemUiOverlayStyle(
        systemNavigationBarColor: Colors.transparent,
        systemNavigationBarDividerColor: Colors.transparent,
        systemNavigationBarContrastEnforced: true,
      ),
    );
  }

  _log.fine('bootstrap complete');

  runApp(
    const ProviderScope(child: App()),
  );
}

/// Initialize all `dart_mappable` mappers here as models are added.
void initializeMappers() {
  // Empty until app_dependencies adds its mappers (Chunk 2).
}
```

Note the `depend_on_referenced_packages` ignore: `flutter_web_plugins` ships with Flutter but isn't in pubspec, and `usePathUrlStrategy()` lives there.

### Task 1.5: Write the stub `App` widget

**Files:**
- Create: `app/lib/app.dart`

For this chunk, `App` is a stub that just shows a placeholder. Chunks 2 and 3 replace its body.

- [ ] **Step 1: Write stub `app.dart`**

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class App extends ConsumerWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp(
      title: 'Craftsky',
      debugShowCheckedModeBanner: false,
      home: Scaffold(
        body: Center(
          child: Text(
            'Craftsky scaffold (chunk 1)',
            style: Theme.of(context).textTheme.titleLarge,
          ),
        ),
      ),
    );
  }
}
```

- [ ] **Step 2: Run analyze**

Run (from `app/`): `flutter analyze`

Expected: `No issues found!` (or zero errors; warnings from the default lint set are acceptable only if they also appear in a fresh `flutter create`. New warnings from our code must be fixed.)

- [ ] **Step 3: Run the app on a single target to smoke-check**

Run (from `app/`): one of
- `flutter run -d chrome` (fastest)
- `flutter run -d ios` (if simulator available)
- `flutter run -d android` (if emulator available)

Expected: app launches, shows "Craftsky scaffold (chunk 1)" centered. Check console — no uncaught exceptions, no red screen.

Stop the app (`q` in the terminal) once verified.

- [ ] **Step 4: Commit chunk 1**

```bash
git add app/lib/main.dart app/lib/bootstrap.dart app/lib/app.dart
git commit -m "feat(app): entry point, bootstrap, and stub App widget

main.dart wraps runApp in runZonedGuarded, configures logging, and
delegates platform init to bootstrap(). bootstrap() sets up
url-path strategy, locale, shared_preferences prefix, and Android
edge-to-edge system UI. Stub App renders a placeholder — theme,
router, and app dependencies land in chunks 2 and 3."
```

---

## Chunk 2: Theme layer and app-dependencies provider

Adds the theme (colors, extensions, responsive primitives, persisted theme-mode notifier) and the async `appDependenciesProvider`. At the end of this chunk, `App` resolves dependencies asynchronously and shows a themed placeholder page.

### Task 2.1: Theme extensions

**Files:**
- Create: `app/lib/theme/theme_extensions.dart`

- [ ] **Step 1: Write theme_extensions.dart**

```dart
import 'package:flutter/material.dart';

class SpacingTheme extends ThemeExtension<SpacingTheme> {
  const SpacingTheme({
    this.xs = 4,
    this.s = 8,
    this.m = 16,
    this.l = 24,
    this.xl = 32,
  });

  final double xs;
  final double s;
  final double m;
  final double l;
  final double xl;

  @override
  SpacingTheme copyWith({double? xs, double? s, double? m, double? l, double? xl}) {
    return SpacingTheme(
      xs: xs ?? this.xs,
      s: s ?? this.s,
      m: m ?? this.m,
      l: l ?? this.l,
      xl: xl ?? this.xl,
    );
  }

  @override
  SpacingTheme lerp(ThemeExtension<SpacingTheme>? other, double t) {
    if (other is! SpacingTheme) return this;
    return SpacingTheme(
      xs: lerpDouble(xs, other.xs, t)!,
      s: lerpDouble(s, other.s, t)!,
      m: lerpDouble(m, other.m, t)!,
      l: lerpDouble(l, other.l, t)!,
      xl: lerpDouble(xl, other.xl, t)!,
    );
  }
}

class RadiusTheme extends ThemeExtension<RadiusTheme> {
  const RadiusTheme({
    this.small = 4,
    this.medium = 8,
    this.large = 16,
  });

  final double small;
  final double medium;
  final double large;

  @override
  RadiusTheme copyWith({double? small, double? medium, double? large}) {
    return RadiusTheme(
      small: small ?? this.small,
      medium: medium ?? this.medium,
      large: large ?? this.large,
    );
  }

  @override
  RadiusTheme lerp(ThemeExtension<RadiusTheme>? other, double t) {
    if (other is! RadiusTheme) return this;
    return RadiusTheme(
      small: lerpDouble(small, other.small, t)!,
      medium: lerpDouble(medium, other.medium, t)!,
      large: lerpDouble(large, other.large, t)!,
    );
  }
}

class DurationTheme extends ThemeExtension<DurationTheme> {
  const DurationTheme({
    this.fast = const Duration(milliseconds: 150),
    this.medium = const Duration(milliseconds: 300),
    this.slow = const Duration(milliseconds: 500),
  });

  final Duration fast;
  final Duration medium;
  final Duration slow;

  @override
  DurationTheme copyWith({Duration? fast, Duration? medium, Duration? slow}) {
    return DurationTheme(
      fast: fast ?? this.fast,
      medium: medium ?? this.medium,
      slow: slow ?? this.slow,
    );
  }

  @override
  DurationTheme lerp(ThemeExtension<DurationTheme>? other, double t) {
    // Durations don't interpolate meaningfully; snap at midpoint.
    if (other is! DurationTheme) return this;
    return t < 0.5 ? this : other;
  }
}

class SemanticColorsTheme extends ThemeExtension<SemanticColorsTheme> {
  const SemanticColorsTheme({
    required this.error,
    required this.warning,
    required this.success,
    required this.info,
  });

  final Color error;
  final Color warning;
  final Color success;
  final Color info;

  @override
  SemanticColorsTheme copyWith({Color? error, Color? warning, Color? success, Color? info}) {
    return SemanticColorsTheme(
      error: error ?? this.error,
      warning: warning ?? this.warning,
      success: success ?? this.success,
      info: info ?? this.info,
    );
  }

  @override
  SemanticColorsTheme lerp(ThemeExtension<SemanticColorsTheme>? other, double t) {
    if (other is! SemanticColorsTheme) return this;
    return SemanticColorsTheme(
      error: Color.lerp(error, other.error, t)!,
      warning: Color.lerp(warning, other.warning, t)!,
      success: Color.lerp(success, other.success, t)!,
      info: Color.lerp(info, other.info, t)!,
    );
  }
}
```

- [ ] **Step 2: Analyze**

Run: `flutter analyze lib/theme/theme_extensions.dart`

Expected: no errors. If `lerpDouble` is unresolved, add `import 'dart:ui' show lerpDouble;`.

### Task 2.2: `AppTheme`

**Files:**
- Create: `app/lib/theme/app_theme.dart`

- [ ] **Step 1: Write app_theme.dart**

```dart
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:dynamic_color/dynamic_color.dart';
import 'package:flex_color_scheme/flex_color_scheme.dart';
import 'package:flutter/material.dart';

class AppTheme {
  AppTheme._();

  static final ThemeData lightThemeData = _buildLight();
  static final ThemeData darkThemeData = _buildDark();

  static ThemeData _buildLight() {
    final base = FlexThemeData.light(
      scheme: FlexScheme.material,
      subThemesData: const FlexSubThemesData(
        interactionEffects: true,
        tintedDisabledControls: true,
        defaultRadius: 8,
        inputDecoratorIsFilled: true,
        inputDecoratorBorderType: FlexInputBorderType.outline,
      ),
      visualDensity: FlexColorScheme.comfortablePlatformDensity,
    );
    return base.copyWith(extensions: _extensions(base.colorScheme));
  }

  static ThemeData _buildDark() {
    final base = FlexThemeData.dark(
      scheme: FlexScheme.material,
      subThemesData: const FlexSubThemesData(
        interactionEffects: true,
        tintedDisabledControls: true,
        defaultRadius: 8,
        inputDecoratorIsFilled: true,
        inputDecoratorBorderType: FlexInputBorderType.outline,
      ),
      visualDensity: FlexColorScheme.comfortablePlatformDensity,
    );
    return base.copyWith(extensions: _extensions(base.colorScheme));
  }

  static List<ThemeExtension<dynamic>> _extensions(ColorScheme scheme) {
    return <ThemeExtension<dynamic>>[
      const SpacingTheme(),
      const RadiusTheme(),
      const DurationTheme(),
      SemanticColorsTheme(
        error: Colors.red.harmonizeWith(scheme.error),
        warning: Colors.orange.harmonizeWith(scheme.primary),
        success: Colors.green.harmonizeWith(scheme.primary),
        info: Colors.blue.harmonizeWith(scheme.primary),
      ),
    ];
  }
}
```

- [ ] **Step 2: Analyze**

Run: `flutter analyze lib/theme/`

Expected: no errors.

### Task 2.3: `FormFactor` and `FormFactorWidget`

**Files:**
- Create: `app/lib/theme/form_factor.dart`

- [ ] **Step 1: Write form_factor.dart**

```dart
import 'package:flutter/material.dart';

enum FormFactor {
  mobile(breakpoint: 600),
  tablet(breakpoint: 900),
  laptop(breakpoint: 1200),
  desktop(breakpoint: double.infinity);

  const FormFactor({required this.breakpoint});

  final double breakpoint;

  bool get isSmall => this == FormFactor.mobile || this == FormFactor.tablet;
  bool get isLarge => this == FormFactor.laptop || this == FormFactor.desktop;

  static FormFactor fromWidth(double width) {
    if (width <= FormFactor.mobile.breakpoint) return FormFactor.mobile;
    if (width <= FormFactor.tablet.breakpoint) return FormFactor.tablet;
    if (width <= FormFactor.laptop.breakpoint) return FormFactor.laptop;
    return FormFactor.desktop;
  }
}

/// Provides the current [FormFactor] to the subtree.
///
/// Recomputes on every build so orientation and resize changes propagate.
class FormFactorWidget extends StatelessWidget {
  const FormFactorWidget({required this.child, super.key});

  final Widget child;

  static FormFactor of(BuildContext context) {
    final scope = context.dependOnInheritedWidgetOfExactType<_FormFactorScope>();
    assert(scope != null, 'No FormFactorWidget found in context');
    return scope!.formFactor;
  }

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.sizeOf(context).width;
    final formFactor = FormFactor.fromWidth(width);
    return _FormFactorScope(formFactor: formFactor, child: child);
  }
}

class _FormFactorScope extends InheritedWidget {
  const _FormFactorScope({required this.formFactor, required super.child});

  final FormFactor formFactor;

  @override
  bool updateShouldNotify(_FormFactorScope oldWidget) => formFactor != oldWidget.formFactor;
}
```

- [ ] **Step 2: Analyze**

Run: `flutter analyze lib/theme/form_factor.dart`

Expected: no errors.

### Task 2.4: `TextScaleFactorClamper`

**Files:**
- Create: `app/lib/theme/text_scale_factor_clamper.dart`

- [ ] **Step 1: Write text_scale_factor_clamper.dart**

```dart
import 'package:flutter/material.dart';

/// Wraps [child] in a [MediaQuery] that clamps the text scaler to
/// `[minTextScaleFactor, maxTextScaleFactor]`.
class TextScaleFactorClamper extends StatelessWidget {
  const TextScaleFactorClamper({
    required this.child,
    this.minTextScaleFactor = 1.0,
    this.maxTextScaleFactor = 1.5,
    super.key,
  });

  final Widget child;
  final double minTextScaleFactor;
  final double maxTextScaleFactor;

  @override
  Widget build(BuildContext context) {
    final mediaQuery = MediaQuery.of(context);
    final scaler = mediaQuery.textScaler.clamp(
      minScaleFactor: minTextScaleFactor,
      maxScaleFactor: maxTextScaleFactor,
    );
    return MediaQuery(
      data: mediaQuery.copyWith(textScaler: scaler),
      child: child,
    );
  }
}
```

- [ ] **Step 2: Analyze**

Run: `flutter analyze lib/theme/`

Expected: no errors.

### Task 2.5: `app_dependencies.dart` (model + provider + accessors)

**Files:**
- Create: `app/lib/app_dependencies.dart`
- Generated: `app/lib/app_dependencies.g.dart`, `app/lib/app_dependencies.mapper.dart` (produced by build_runner)

- [ ] **Step 1: Write app_dependencies.dart**

```dart
import 'dart:async';
import 'dart:ui';

import 'package:dart_mappable/dart_mappable.dart';
import 'package:device_info_plus/device_info_plus.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/date_symbol_data_local.dart';
import 'package:logging/logging.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:pub_semver/pub_semver.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:shared_preferences/shared_preferences.dart';

part 'app_dependencies.mapper.dart';
part 'app_dependencies.g.dart';

final _log = Logger('AppDependencies');

@MappableClass()
class CraftskyDeviceInfo with CraftskyDeviceInfoMappable {
  CraftskyDeviceInfo({
    required this.platform,
    required this.deviceId,
    required this.model,
    required this.brand,
    required this.osVersion,
  });

  /// 'Android' | 'iOS' | 'Web'
  final String platform;
  final String deviceId;
  final String model;
  final String brand;
  final String osVersion;
}

@MappableClass()
class AppDependencies with AppDependenciesMappable {
  AppDependencies({
    required this.packageInfo,
    required this.deviceInfo,
    required this.sharedPreferences,
    required this.appVersion,
  });

  final PackageInfo packageInfo;
  final CraftskyDeviceInfo deviceInfo;
  final SharedPreferences sharedPreferences;
  final Version appVersion;
}

@Riverpod(keepAlive: true)
Future<AppDependencies> appDependencies(Ref ref) async {
  _log.info('initializing app dependencies');

  final packageInfo = await PackageInfo.fromPlatform();
  final deviceInfo = await _resolveDeviceInfo();
  final prefs = await SharedPreferences.getInstance();
  final appVersion = Version.parse(packageInfo.version);

  final deviceLocale = PlatformDispatcher.instance.locale.toString();
  await initializeDateFormatting(deviceLocale, null);
  _log.fine('initialized date formatting for $deviceLocale');

  _log.info('app dependencies ready (version=$appVersion, platform=${deviceInfo.platform})');

  return AppDependencies(
    packageInfo: packageInfo,
    deviceInfo: deviceInfo,
    sharedPreferences: prefs,
    appVersion: appVersion,
  );
}

Future<CraftskyDeviceInfo> _resolveDeviceInfo() async {
  final plugin = DeviceInfoPlugin();

  if (kIsWeb) {
    final web = await plugin.webBrowserInfo;
    return CraftskyDeviceInfo(
      platform: 'Web',
      deviceId: '',
      brand: web.browserName.name,
      model: web.userAgent ?? '',
      osVersion: web.platform ?? '',
    );
  }

  switch (defaultTargetPlatform) {
    case TargetPlatform.android:
      final android = await plugin.androidInfo;
      return CraftskyDeviceInfo(
        platform: 'Android',
        deviceId: android.id,
        brand: android.brand,
        model: android.model,
        osVersion: android.version.release,
      );
    case TargetPlatform.iOS:
      final ios = await plugin.iosInfo;
      return CraftskyDeviceInfo(
        platform: 'iOS',
        deviceId: ios.identifierForVendor ?? '',
        brand: ios.utsname.machine,
        model: ios.utsname.machine,
        osVersion: ios.systemVersion,
      );
    case TargetPlatform.fuchsia:
    case TargetPlatform.linux:
    case TargetPlatform.macOS:
    case TargetPlatform.windows:
      throw UnsupportedError('Platform not supported by this scaffold: $defaultTargetPlatform');
  }
}

@Riverpod(keepAlive: true)
SharedPreferences sharedPreferences(Ref ref) =>
    ref.watch(appDependenciesProvider.select((a) => a.requireValue.sharedPreferences));

@Riverpod(keepAlive: true)
PackageInfo packageInfo(Ref ref) =>
    ref.watch(appDependenciesProvider.select((a) => a.requireValue.packageInfo));

@Riverpod(keepAlive: true)
CraftskyDeviceInfo deviceInfo(Ref ref) =>
    ref.watch(appDependenciesProvider.select((a) => a.requireValue.deviceInfo));

@Riverpod(keepAlive: true)
Version appVersion(Ref ref) =>
    ref.watch(appDependenciesProvider.select((a) => a.requireValue.appVersion));
```

- [ ] **Step 2: Run build_runner**

Run (from `app/`): `dart run build_runner build --delete-conflicting-outputs`

Expected: generates `app_dependencies.g.dart` and `app_dependencies.mapper.dart`, exits 0. If it reports conflicts or errors, fix and re-run.

- [ ] **Step 3: Wire mappers into `bootstrap.dart`**

Replace the `initializeMappers()` function in `app/lib/bootstrap.dart` to register the new mappers. The generated file exposes initializer functions named `MapperContainer.globals.useAll([...])` or per-class `SomeMappableClass.ensureInitialized()`. The exact entry point depends on `dart_mappable_builder` — check the generated `app_dependencies.mapper.dart`. Typical shape:

```dart
// In bootstrap.dart, replace the empty initializeMappers body with:
void initializeMappers() {
  AppDependenciesMapper.ensureInitialized();
  CraftskyDeviceInfoMapper.ensureInitialized();
}
```

You'll need to `import 'package:craftsky_app/app_dependencies.dart';` at the top of `bootstrap.dart`.

- [ ] **Step 4: Analyze**

Run: `flutter analyze`

Expected: no errors.

### Task 2.6: `ThemeModeNotifier`

**Files:**
- Create: `app/lib/theme/theme_notifier.dart`
- Generated: `app/lib/theme/theme_notifier.g.dart`

- [ ] **Step 1: Write theme_notifier.dart**

```dart
import 'package:craftsky_app/app_dependencies.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'theme_notifier.g.dart';

@Riverpod(keepAlive: true)
class ThemeModeNotifier extends _$ThemeModeNotifier {
  static const _prefsKey = 'theme_mode';

  @override
  ThemeMode build() {
    final prefs = ref.watch(sharedPreferencesProvider);
    return _parse(prefs.getString(_prefsKey));
  }

  Future<void> setMode(ThemeMode mode) async {
    final prefs = ref.read(sharedPreferencesProvider);
    await prefs.setString(_prefsKey, mode.name);
    if (!ref.mounted) return;
    state = mode;
  }

  static ThemeMode _parse(String? raw) {
    return switch (raw) {
      'light' => ThemeMode.light,
      'dark' => ThemeMode.dark,
      _ => ThemeMode.system,
    };
  }
}
```

- [ ] **Step 2: Run build_runner**

Run: `dart run build_runner build --delete-conflicting-outputs`

Expected: `theme_notifier.g.dart` generated.

### Task 2.7: Test `ThemeModeNotifier` persistence

This is real behavior worth testing: write → state updates → prefs round-trips.

**Files:**
- Create: `app/test/theme/theme_notifier_test.dart`

- [ ] **Step 1: Write the test**

```dart
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/theme/theme_notifier.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  group('ThemeModeNotifier', () {
    late SharedPreferences prefs;

    setUp(() async {
      SharedPreferences.setMockInitialValues({});
      prefs = await SharedPreferences.getInstance();
    });

    ProviderContainer makeContainer() {
      return ProviderContainer(
        overrides: [
          sharedPreferencesProvider.overrideWithValue(prefs),
        ],
      );
    }

    test('defaults to ThemeMode.system when no preference stored', () {
      final container = makeContainer();
      addTearDown(container.dispose);

      expect(container.read(themeModeNotifierProvider), ThemeMode.system);
    });

    test('reads persisted ThemeMode.dark', () async {
      await prefs.setString('theme_mode', 'dark');
      final container = makeContainer();
      addTearDown(container.dispose);

      expect(container.read(themeModeNotifierProvider), ThemeMode.dark);
    });

    test('setMode updates state and persists to SharedPreferences', () async {
      final container = makeContainer();
      addTearDown(container.dispose);

      await container.read(themeModeNotifierProvider.notifier).setMode(ThemeMode.light);

      expect(container.read(themeModeNotifierProvider), ThemeMode.light);
      expect(prefs.getString('theme_mode'), 'light');
    });

    test('unknown persisted value falls back to system', () async {
      await prefs.setString('theme_mode', 'garbage');
      final container = makeContainer();
      addTearDown(container.dispose);

      expect(container.read(themeModeNotifierProvider), ThemeMode.system);
    });
  });
}
```

- [ ] **Step 2: Run test, verify pass**

Run (from `app/`): `flutter test test/theme/theme_notifier_test.dart`

Expected: `All tests passed!` (4 passing).

If a test fails, fix the notifier — don't weaken the test.

### Task 2.8: Update `App` to resolve dependencies and apply theme

**Files:**
- Modify: `app/lib/app.dart`

- [ ] **Step 1: Replace app.dart**

```dart
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:craftsky_app/theme/text_scale_factor_clamper.dart';
import 'package:craftsky_app/theme/theme_notifier.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';

final _log = Logger('App');

class App extends ConsumerWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final depsAsync = ref.watch(appDependenciesProvider);

    return switch (depsAsync) {
      AsyncData() => const _ReadyApp(),
      AsyncError(:final error, :final stackTrace) => _ErrorApp(error: error, stackTrace: stackTrace),
      _ => const _LoadingApp(),
    };
  }
}

class _ReadyApp extends ConsumerWidget {
  const _ReadyApp();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final themeMode = ref.watch(themeModeNotifierProvider);

    return MaterialApp(
      title: 'Craftsky',
      theme: AppTheme.lightThemeData,
      darkTheme: AppTheme.darkThemeData,
      themeMode: themeMode,
      debugShowCheckedModeBanner: false,
      builder: (context, child) {
        return TextScaleFactorClamper(
          child: FormFactorWidget(
            child: child ?? const SizedBox.shrink(),
          ),
        );
      },
      home: const _PlaceholderHome(),
    );
  }
}

class _PlaceholderHome extends StatelessWidget {
  const _PlaceholderHome();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: Center(
        child: Text('Craftsky scaffold (chunk 2)', style: theme.textTheme.titleLarge),
      ),
    );
  }
}

class _LoadingApp extends StatelessWidget {
  const _LoadingApp();

  @override
  Widget build(BuildContext context) {
    return const MaterialApp(
      debugShowCheckedModeBanner: false,
      home: InitializationLoadingScreen(),
    );
  }
}

class _ErrorApp extends ConsumerWidget {
  const _ErrorApp({required this.error, required this.stackTrace});

  final Object error;
  final StackTrace stackTrace;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    _log.severe('App dependencies failed to initialize', error, stackTrace);
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      home: InitializationErrorScreen(
        error: error,
        onRetry: () => ref.invalidate(appDependenciesProvider),
      ),
    );
  }
}

class InitializationLoadingScreen extends StatelessWidget {
  const InitializationLoadingScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(child: CircularProgressIndicator()),
    );
  }
}

class InitializationErrorScreen extends StatelessWidget {
  const InitializationErrorScreen({
    required this.error,
    required this.onRetry,
    super.key,
  });

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(Icons.error_outline, color: theme.colorScheme.error, size: 64),
              const SizedBox(height: 16),
              Text(
                'Initialization Failed',
                style: theme.textTheme.headlineSmall,
              ),
              const SizedBox(height: 8),
              Text(
                error.toString(),
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyMedium,
              ),
              const SizedBox(height: 24),
              ElevatedButton.icon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh),
                label: const Text('Retry'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
```

Notes:
- Three top-level `_ReadyApp`, `_LoadingApp`, `_ErrorApp` internal widget classes plus public `InitializationLoadingScreen` / `InitializationErrorScreen` — matches the spec's "top-level classes in `app.dart`" rule.
- Router integration comes in chunk 3. For now `_ReadyApp` uses `MaterialApp` with `home:` — chunk 3 replaces it with `MaterialApp.router`.

- [ ] **Step 2: Analyze**

Run (from `app/`): `flutter analyze`

Expected: no errors.

- [ ] **Step 3: Run the app on one target**

Run: `flutter run -d chrome` (or iOS/Android).

Expected: app launches. You briefly see the loading spinner, then the "Craftsky scaffold (chunk 2)" themed placeholder. No red screen, no exceptions in the console.

- [ ] **Step 4: Commit chunk 2**

```bash
git add app/lib/theme/ app/lib/app_dependencies.dart app/lib/app_dependencies.g.dart app/lib/app_dependencies.mapper.dart app/lib/app.dart app/lib/bootstrap.dart app/test/
git commit -m "feat(app): theme layer and app-dependencies provider

Adds FlexColorScheme-based light/dark themes with spacing, radius,
duration, and semantic-color extensions. Adds FormFactorWidget and
TextScaleFactorClamper. appDependenciesProvider resolves
SharedPreferences, PackageInfo, device info, and app version
asynchronously; App renders loading/error/data states. ThemeModeNotifier
persists ThemeMode through SharedPreferences with round-trip tests."
```

---

## Chunk 3: Router, HomePage, and smoke test

Adds `go_router` wired through a Riverpod provider, the `HomePage` landing page, the `ErrorScreen`, and the widget-level smoke test. Final verification runs `flutter analyze`, `flutter test`, and `flutter run`.

### Task 3.1: `HomePage`

**Files:**
- Create: `app/lib/router/home_page.dart`

(We write the page before the router because the router imports it.)

- [ ] **Step 1: Write home_page.dart**

```dart
import 'package:craftsky_app/app_dependencies.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class HomePage extends ConsumerWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final version = ref.watch(packageInfoProvider).version;

    return Scaffold(
      appBar: AppBar(title: const Text('Craftsky')),
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(Icons.palette_outlined, size: 64, color: theme.colorScheme.primary),
              const SizedBox(height: 16),
              Text('Craftsky', style: theme.textTheme.headlineMedium),
              const SizedBox(height: 8),
              Text(
                'Scaffold ready',
                style: theme.textTheme.titleMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 8),
              Text('v$version', style: theme.textTheme.bodySmall),
            ],
          ),
        ),
      ),
    );
  }
}
```

- [ ] **Step 2: Analyze**

Run: `flutter analyze lib/router/home_page.dart`

Expected: no errors.

### Task 3.2: `ErrorScreen`

**Files:**
- Create: `app/lib/router/error_screen.dart`

- [ ] **Step 1: Write error_screen.dart**

```dart
import 'package:craftsky_app/router/router.dart';
import 'package:flutter/material.dart';

class ErrorScreen extends StatelessWidget {
  const ErrorScreen({required this.error, super.key});

  final Exception error;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(Icons.broken_image_outlined, size: 64, color: theme.colorScheme.error),
              const SizedBox(height: 16),
              Text('Something went wrong', style: theme.textTheme.headlineSmall),
              const SizedBox(height: 8),
              Text(
                error.toString(),
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyMedium,
              ),
              const SizedBox(height: 24),
              ElevatedButton.icon(
                onPressed: () => const HomeRoute().go(context),
                icon: const Icon(Icons.home),
                label: const Text('Go home'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
```

This imports `router.dart` for `HomeRoute` — fine even though router.dart imports this file back; Dart handles mutual top-level references.

### Task 3.3: `router.dart`

**Files:**
- Create: `app/lib/router/router.dart`
- Generated: `app/lib/router/router.g.dart` (produced by build_runner)

- [ ] **Step 1: Write router.dart**

```dart
import 'package:craftsky_app/router/error_screen.dart';
import 'package:craftsky_app/router/home_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'router.g.dart';

class _NavigatorKeys {
  _NavigatorKeys._();

  static GlobalKey<NavigatorState>? _rootKey;

  static GlobalKey<NavigatorState> get rootNavigatorKey =>
      _rootKey ??= GlobalKey<NavigatorState>(debugLabel: 'rootNavigator');
}

@riverpod
GoRouter goRouter(Ref ref) {
  return GoRouter(
    initialLocation: '/',
    navigatorKey: _NavigatorKeys.rootNavigatorKey,
    debugLogDiagnostics: true,
    routes: $appRoutes,
    errorBuilder: (context, state) {
      final error = state.error ?? Exception('Unknown routing error');
      return ErrorScreen(error: error);
    },
  );
}

@TypedGoRoute<HomeRoute>(path: '/', name: 'home')
class HomeRoute extends GoRouteData with $HomeRoute {
  const HomeRoute();

  @override
  Widget build(BuildContext context, GoRouterState state) => const HomePage();
}

extension GoRouterExtension on GoRouter {
  /// Pops any existing stack and replaces the current location.
  void clearStackAndNavigate(String location) {
    while (canPop()) {
      pop();
    }
    pushReplacement(location);
  }
}
```

- [ ] **Step 2: Run build_runner**

Run: `dart run build_runner build --delete-conflicting-outputs`

Expected: generates `router.g.dart` with `$appRoutes` and `$HomeRoute`. If go_router_builder complains that `HomeRoute` is missing `$HomeRoute`, confirm the `with $HomeRoute` mixin and the `@TypedGoRoute` annotation are spelled exactly as above.

- [ ] **Step 3: Analyze**

Run: `flutter analyze`

Expected: no errors.

### Task 3.4: Switch `App` to `MaterialApp.router`

**Files:**
- Modify: `app/lib/app.dart`

- [ ] **Step 1: Update `_ReadyApp` to use the router**

In `app/lib/app.dart`, change `_ReadyApp.build` to use `MaterialApp.router` + the `goRouterProvider`, and delete `_PlaceholderHome`.

Replace the existing `_ReadyApp` class and remove `_PlaceholderHome`:

```dart
class _ReadyApp extends ConsumerWidget {
  const _ReadyApp();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final themeMode = ref.watch(themeModeNotifierProvider);
    final router = ref.watch(goRouterProvider);

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
  }
}
```

Add at top of file:

```dart
import 'package:craftsky_app/router/router.dart';
```

- [ ] **Step 2: Analyze**

Run: `flutter analyze`

Expected: no errors. Any remaining lint about unused imports should be cleaned up.

### Task 3.5: Smoke test

**Files:**
- Modify: `app/test/widget_test.dart` (replace the `flutter create` default)

- [ ] **Step 1: Replace widget_test.dart**

```dart
import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/router/home_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    PackageInfo.setMockInitialValues(
      appName: 'craftsky_app',
      packageName: 'social.craftsky.app',
      version: '1.0.0',
      buildNumber: '1',
      buildSignature: '',
    );
  });

  testWidgets('App boots and renders HomePage', (tester) async {
    await tester.pumpWidget(const ProviderScope(child: App()));

    // Let the async appDependenciesProvider resolve.
    await tester.pumpAndSettle();

    expect(find.byType(HomePage), findsOneWidget);
    expect(find.text('Craftsky'), findsWidgets);
  });
}
```

Notes:
- `device_info_plus` has no `setMockInitialValues`-style helper. On the Flutter test platform, `defaultTargetPlatform` returns `android` and the plugin's method channel calls normally throw `MissingPluginException` in unit tests. If the test fails because of a `MissingPluginException` from `device_info_plus`, register a mock handler as a second step (Step 2 below). If it passes without, delete this step — don't carry dead mocking code.

- [ ] **Step 2: If needed, mock the `device_info_plus` channel**

Only if Step 1's test fails with a `MissingPluginException`. Add at the top of `setUp`:

```dart
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

// ...
TestDefaultBinaryMessengerBinding.ensureInitialized()
    .defaultBinaryMessenger
    .setMockMethodCallHandler(
  const MethodChannel('dev.fluttercommunity.plus/device_info'),
  (call) async {
    if (call.method == 'getAndroidDeviceInfo') {
      return <String, dynamic>{
        'id': 'test',
        'brand': 'test',
        'model': 'test',
        'version': {'release': '14', 'sdkInt': 34, /* other fields */},
        // … device_info_plus expects a larger map; consult the package source
        // if the handler returns a cast error and fill missing keys.
      };
    }
    return null;
  },
);
```

If the map schema becomes painful, route around it instead: override `appDependenciesProvider` in the `ProviderScope` to a `Future.value(...)` with a hand-built `AppDependencies` using a stub `CraftskyDeviceInfo`. This is cleaner; reach for it if the channel mock takes more than ~20 lines.

- [ ] **Step 3: Run the test**

Run (from `app/`): `flutter test test/widget_test.dart`

Expected: `All tests passed!`.

### Task 3.6: Full verification

- [ ] **Step 1: Run build_runner one more time to ensure generated files are fresh**

Run: `dart run build_runner build --delete-conflicting-outputs`

Expected: exits 0, no unexpected changes to committed files.

- [ ] **Step 2: `flutter analyze` clean**

Run: `flutter analyze`

Expected: `No issues found!` (or zero errors + no warnings that weren't in `flutter create` baseline).

- [ ] **Step 3: `flutter test` all green**

Run: `flutter test`

Expected: all tests pass (the theme-notifier suite from Chunk 2 and the smoke test from Task 3.5).

- [ ] **Step 4: Smoke-run on at least one device**

Run one of:
- `flutter run -d chrome`
- `flutter run -d ios`
- `flutter run -d android`

Expected: app launches into `HomePage` showing "Craftsky", "Scaffold ready", and the version `v1.0.0`. No red screen, no exceptions in the console. Stop with `q`.

- [ ] **Step 5: Commit chunk 3**

```bash
git add app/lib/router/ app/lib/app.dart app/test/widget_test.dart
git commit -m "feat(app): router, HomePage, and smoke test

Adds go_router via a Riverpod provider with a single typed HomeRoute,
plus an ErrorScreen constructed from GoRouter.errorBuilder. App uses
MaterialApp.router; HomePage shows the app version from
packageInfoProvider. Adds a widget smoke test that mocks
SharedPreferences and PackageInfo, pumps App, and asserts HomePage
renders. flutter analyze / flutter test / flutter run all pass."
```

---

## Done when

- `flutter analyze` clean.
- `flutter test` passes (theme-notifier + smoke test).
- `flutter run` on at least one target launches into `HomePage`.
- All three chunk commits on the branch.
- No stray files beyond what the spec's directory-layout section lists.
