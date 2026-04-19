# Flutter App Scaffold — Design

**Date:** 2026-04-19
**Scope:** `app/` (Flutter client)
**Status:** Design

## Context

The Flutter app under `app/` is currently the `flutter create` default — a counter demo. This spec lays down a minimal, opinionated scaffold so feature work (feed, profile, auth, compose, etc.) can proceed against a clean skeleton.

The scaffold is ported from `/Users/douglastodd/Projects/squiddies/squiddies_flutter`, a working production app the user is happy with. We take its patterns for entry/bootstrap, async dependency initialization, go_router + Riverpod wiring, theming, and responsive form-factor handling.

We deliberately **do not** include the following in this pass:

- **Auth** — the OAuth BFF is still being specced ([2026-04-18-appview-oauth-bff-design.md](2026-04-18-appview-oauth-bff-design.md)). Adding auth now would bake in assumptions we can't yet verify.
- **Tab structure / `AppShell` / `StatefulShellRoute`** — the set of tabs is a product decision that deserves its own brainstorm.
- **Firebase, native splash, custom fonts, Google Sign-In, display-mode tweaks** — all optional; easy to add when the feature that needs them lands.
- **Feature directories** (`feed/`, `profile/`, …) — each feature gets its own directory when it arrives, following the squiddies convention of `pages/ + providers/ + widgets/`.

## Constraints

- **Riverpod 3.x with code generation** (`@riverpod` annotation). See `.claude/rules/riverpod.md`.
- **Flutter widget and theming rules** from `.claude/rules/flutter.md`: one widget class per concern, no `_build*` helpers, `Theme.of(context)` for colors/text styles, no `.withOpacity()`, immutable models via `dart_mappable` or `freezed`, `logging` over `print`.
- **Architectural rule from `AGENTS.md`:** the Flutter app talks to the App View only (HTTPS + session token). It must never read/write PDS directly in the happy path. The atproto SDK is `atproto.dart`.

## Architecture

Five concerns, each in its own layer:

1. **Entry point & bootstrap** — Flutter/platform init that must not throw in production.
2. **App dependencies provider** — one-shot async init that can throw, gated by a loading/error UI.
3. **Router** — `go_router` + `go_router_builder` typed routes, exposed via a Riverpod provider.
4. **Theme** — `FlexColorScheme` + theme extensions + a persisted `ThemeMode` provider.
5. **Responsive primitives** — `FormFactor` + text-scale clamping, available app-wide.

## Section 1 — Entry point & bootstrap

### `lib/main.dart`

Wraps `runApp` in `runZonedGuarded`. Ensures widgets binding, calls `registerErrorHandlers()`, then `await bootstrap(binding)`. Top-level uncaught errors are logged via `dart:developer` `log` (bootstrap hasn't necessarily completed, so we can't assume the app logger is configured yet).

### `lib/bootstrap.dart`

Platform and Flutter init that must **never** throw in production. In order:

- Configure the `logging` package: `Logger.root.level = Level.FINE`, `onRecord` listener that prints formatted records when `kDebugMode`.
- `usePathUrlStrategy()` for web (removes `#` from URLs).
- Web short-circuit: on `kIsWeb`, skip native-only init and call `runApp` immediately.
- Set `Intl.defaultLocale` from `PlatformDispatcher.instance.locale`.
- Initialize `dart_mappable` mappers (`initializeMappers()` — empty for now, grows with models).
- `SharedPreferences.setPrefix('craftsky.')` — all stored keys namespaced.
- On Android: `SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge)` + transparent system nav bar overlay.
- Finally `runApp(ProviderScope(child: const App()))`.

`registerErrorHandlers()` is a top-level function in the same file that wires:

- `FlutterError.onError` → present + log via `logging`.
- `PlatformDispatcher.instance.onError` → log and return `true`.
- `ErrorWidget.builder` → in debug, default `ErrorWidget`; in release, a minimal red fallback with "An error occurred rendering this element".

### `lib/app.dart`

`App extends ConsumerWidget`. It watches `appDependenciesProvider` (see Section 2) and returns:

- `AsyncData()` → `MaterialApp.router` with `routerConfig: ref.watch(goRouterProvider)` and the theme from `ref.watch(themeProvider)`, wrapped in `TextScaleFactorClamper` and `FormFactorWidget`.
- `AsyncError(:final error)` → `InitializationErrorScreen` with a retry button that calls `ref.invalidate(appDependenciesProvider)`.
- default → `InitializationLoadingScreen` (centered `CircularProgressIndicator`).

Both error and loading screens are standalone widget classes in `app.dart` for now (simple enough that a separate file per screen would be overkill — move out if they grow).

## Section 2 — App dependencies provider

`lib/app_dependencies.dart` plus generated `.g.dart` and `.mapper.dart`.

```dart
@MappableClass()
class CraftskyDeviceInfo with CraftskyDeviceInfoMappable {
  CraftskyDeviceInfo({
    required this.platform,  // 'Android' | 'iOS' | 'Web'
    required this.deviceId,
    required this.model,
    required this.brand,
    required this.osVersion,
  });
  // fields...
}

@MappableClass()
class AppDependencies with AppDependenciesMappable {
  AppDependencies({
    required this.packageInfo,
    required this.deviceInfo,
    required this.sharedPreferences,
    required this.appVersion,
  });
  // fields...
}

@Riverpod(keepAlive: true)
Future<AppDependencies> appDependencies(Ref ref) async {
  final packageInfo = await PackageInfo.fromPlatform();
  final deviceInfo  = await _resolveDeviceInfo();  // Android/iOS/Web branches
  final prefs       = await SharedPreferences.getInstance();
  final appVersion  = Version.parse(packageInfo.version);

  final deviceLocale = PlatformDispatcher.instance.locale.toString();
  await initializeDateFormatting(deviceLocale, null);

  return AppDependencies(
    packageInfo: packageInfo,
    deviceInfo: deviceInfo,
    sharedPreferences: prefs,
    appVersion: appVersion,
  );
}
```

Accessor providers, each `@Riverpod(keepAlive: true)`, using `.select` on `appDependenciesProvider.requireValue` so downstream code never unwraps `AsyncValue`:

- `sharedPreferences(Ref ref)` → `SharedPreferences`
- `packageInfo(Ref ref)` → `PackageInfo`
- `deviceInfo(Ref ref)` → `CraftskyDeviceInfo`
- `appVersion(Ref ref)` → `Version` (from `pub_semver`)

`_resolveDeviceInfo()` is a private top-level function that branches on `kIsWeb` + `defaultTargetPlatform` and throws `UnsupportedError` for unsupported platforms. Only Android, iOS, and Web are supported in this scaffold.

**`requireValue` is safe here** because `App` does not build the router subtree until `appDependenciesProvider` resolves to `AsyncData`.

Excluded deliberately (will be added with the features that need them):

- App View HTTP client (comes with OAuth BFF).
- atproto.dart session / OAuth wiring.
- Notification handler.
- Any user/identity provider.

## Section 3 — Router

### `lib/router/router.dart`

Singleton `NavigatorKey` pattern (squiddies style) to survive hot reload:

```dart
class _NavigatorKeys {
  static GlobalKey<NavigatorState>? _rootKey;
  static GlobalKey<NavigatorState> get rootNavigatorKey =>
      _rootKey ??= GlobalKey<NavigatorState>(debugLabel: 'rootNavigator');
}
```

Router provider:

```dart
@riverpod
GoRouter goRouter(Ref ref) {
  return GoRouter(
    initialLocation: '/',
    navigatorKey: _NavigatorKeys.rootNavigatorKey,
    debugLogDiagnostics: true,
    routes: $appRoutes,
    errorBuilder: (context, state) => ErrorRoute(error: state.error!).build(context, state),
  );
}
```

Routes (this scaffold has exactly one):

```dart
@TypedGoRoute<HomeRoute>(path: '/', name: 'home')
class HomeRoute extends GoRouteData with $HomeRoute {
  const HomeRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) => const HomePage();
}
```

`HomePage` is a `ConsumerWidget` in `router/home_page.dart` showing a placeholder `Scaffold`: app name, app version (from `packageInfoProvider`), and a muted subtitle. No tabs, no shell. It's a landing page that proves the scaffold works.

`clearStackAndNavigate` extension on `GoRouter` is ported from squiddies — kept because it's tiny and useful for the eventual login/logout transitions.

**No `redirect` is wired.** Auth gating lands with the OAuth BFF feature.

### `lib/router/error_screen.dart`

`ErrorScreen({required Exception error})` — centered icon, text, and a "Go home" `ElevatedButton` that calls `const HomeRoute().go(context)`. `ErrorRoute extends GoRouteData` — not `@TypedGoRoute`-registered; reached only via `GoRouter.errorBuilder`.

### `lib/router/widgets/`

Empty directory committed with a `.gitkeep` — reserved for cross-route widgets.

## Section 4 — Theme

### `lib/theme/app_theme.dart`

Two static `ThemeData`: `AppTheme.lightThemeData` and `AppTheme.darkThemeData`, built via `FlexThemeData.light(...)` / `FlexThemeData.dark(...)`.

- Scheme: `FlexScheme.material` (neutral default — the crafting-oriented palette is a later design pass).
- `subThemesData`: `defaultRadius: 8.0`, `inputDecoratorBorderType: FlexInputBorderType.outline`, `inputDecoratorIsFilled: true`, `interactionEffects: true`, `tintedDisabledControls: true`.
- `visualDensity: FlexColorScheme.comfortablePlatformDensity`.
- `extensions`: the four theme extensions from `theme_extensions.dart` (below).
- No custom `textTheme` / font family — platform default. Custom fonts are added later.

### `lib/theme/theme_extensions.dart`

Four `ThemeExtension` subclasses:

- `SpacingTheme` — `xs`, `s`, `m`, `l`, `xl` doubles (4, 8, 16, 24, 32).
- `RadiusTheme` — `small`, `medium`, `large` doubles.
- `DurationTheme` — `fast`, `medium`, `slow` `Duration`s.
- `SemanticColorsTheme` — `error`, `warning`, `success`, `info` `Color`s. Built with `Colors.red/orange/green/blue.harmonizeWith(colorScheme.primary)` from `dynamic_color` so semantic colors fit the current palette.

Each implements `copyWith` and `lerp` per the `ThemeExtension` contract.

### `lib/theme/theme_notifier.dart`

```dart
@Riverpod(keepAlive: true)
class Theme extends _$Theme {
  static const _prefsKey = 'theme_mode';

  @override
  Future<ThemeMode> build() async {
    final prefs = ref.watch(sharedPreferencesProvider);
    return _parse(prefs.getString(_prefsKey));
  }

  Future<void> setTheme(ThemeMode mode) async {
    final prefs = ref.read(sharedPreferencesProvider);
    await prefs.setString(_prefsKey, mode.name);
    if (!ref.mounted) return;
    state = AsyncData(mode);
  }
}
```

`_parse` returns `ThemeMode.system` for `null` / unknown strings. `App` watches this and passes `themeMode:` to `MaterialApp.router`, defaulting to `system` on `AsyncLoading`/`AsyncError` via a switch expression (per riverpod rule: no `.when()`).

### `lib/theme/form_factor.dart`

```dart
enum FormFactor {
  mobile(breakpoint: 600),
  tablet(breakpoint: 900),
  laptop(breakpoint: 1200),
  desktop(breakpoint: double.infinity);

  const FormFactor({required this.breakpoint});
  final double breakpoint;

  bool get isSmall => this == mobile || this == tablet;
  bool get isLarge => this == laptop || this == desktop;
}
```

`FormFactorWidget extends InheritedWidget` with a static `of(BuildContext)` that returns the current `FormFactor`. Resolved once in `App`'s `didChangeDependencies` based on `MediaQuery.of(context).size.width`.

### `lib/theme/text_scale_factor_clamper.dart`

Ported verbatim from squiddies — wraps its child in a `MediaQuery` with a clamped `TextScaler` in the range `[minTextScaleFactor, maxTextScaleFactor]` (defaults 1.0 and 1.5).

## Section 5 — Directory layout & dependencies

### Directory layout

Only what this scaffold needs. Feature directories come later.

```
app/
  lib/
    main.dart
    bootstrap.dart
    app.dart
    app_dependencies.dart
    app_dependencies.g.dart          (generated)
    app_dependencies.mapper.dart     (generated)
    router/
      router.dart
      router.g.dart                  (generated)
      home_page.dart
      error_screen.dart
      widgets/.gitkeep
    theme/
      app_theme.dart
      theme_extensions.dart
      theme_notifier.dart
      theme_notifier.g.dart          (generated)
      form_factor.dart
      text_scale_factor_clamper.dart
    shared/
      widgets/.gitkeep
  test/
    (unchanged for this scaffold — widget test will fail to compile against the new entry point and should be replaced with a minimal smoke test that pumps App inside a ProviderScope)
```

### `pubspec.yaml` additions

Runtime dependencies:

- `flutter_riverpod: ^3.0.3`
- `riverpod_annotation: ^4.0.0`
- `go_router: ^17.0.0`
- `logging: ^1.3.0`
- `shared_preferences: ^2.5.4`
- `package_info_plus: ^9.0.0`
- `device_info_plus: ^12.3.0`
- `pub_semver: ^2.2.0`
- `intl: ^0.20.2`
- `flex_color_scheme: ^8.4.0`
- `dynamic_color: ^1.8.1`
- `dart_mappable: ^4.6.1`
- `atproto: ^0.16.0` (latest stable; imported as a placeholder / to lock the SDK version. Actual calls land with the OAuth BFF feature.)

Dev dependencies:

- `build_runner: ^2.10.4`
- `riverpod_generator: ^4.0.0+1`
- `riverpod_lint: ^3.0.3`
- `custom_lint: ^0.8.1`
- `go_router_builder: ^4.1.3`
- `dart_mappable_builder: ^4.6.3`
- `flutter_lints: ^6.0.0`

Version pins match squiddies where possible. The agent executing the plan should verify the `atproto.dart` version against pub.dev at implementation time and bump if a newer stable exists.

### Smoke test

`test/widget_test.dart` replaced with a minimal test that:

- Wraps `const App()` in `ProviderScope` with any overrides needed to bypass native platform channels (or skips via `testWidgets('scaffold boots', ...)` tagged `@Tags(['integration'])` if wrapping is too painful).
- Pumps and verifies the placeholder `HomePage` renders, or at minimum that `App` constructs without throwing.

This keeps CI green and gives future contributors a working test file to extend.

## Verification

Before claiming the scaffold is complete, the implementer must run and report output from all of these:

- `dart run build_runner build --delete-conflicting-outputs` — no errors; `.g.dart` and `.mapper.dart` files generated.
- `flutter analyze` — no errors, no new warnings beyond baseline.
- `flutter test` — the smoke test passes.
- `flutter run` on at least one target (iOS simulator, Android emulator, or Chrome) — app launches, `HomePage` renders, no runtime errors in the console.

## Out of Scope (future work, not this spec)

- OAuth BFF session wiring.
- atproto.dart client usage.
- Feed / profile / compose / notification features.
- `AppShell` with `StatefulShellRoute` + tab navigation.
- Firebase (messaging / crashlytics / analytics).
- `flutter_native_splash` config.
- Custom fonts.
- `GoogleSignIn` or any auth provider beyond atproto OAuth.
- Deep linking beyond what `go_router` gives for free.
- i18n beyond `intl` date formatting.
