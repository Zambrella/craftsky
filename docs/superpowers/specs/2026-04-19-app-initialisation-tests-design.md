# App Initialisation Tests — Design

**Date:** 2026-04-19
**Scope:** `app/test/` (Flutter client)
**Status:** Design

## Context

`app/lib/app.dart` is the root `App` widget: it watches `appDependenciesProvider` and switches between three user-visible states (loading → error → ready) via a switch expression, plus a `ref.listen` that logs severely on transition into error exactly once.

Today's coverage:
- `app/test/widget_test.dart` — one smoke test overriding `appDependenciesProvider` with stub deps, asserting `HomePage` renders and that `l10n` keys resolve. Only exercises the **ready** path.
- `app/test/theme/theme_notifier_test.dart` — unrelated (theme persistence).

The **loading** state, the **error** state, the **retry** interaction that invalidates the provider, and the **one-severe-record-per-error-transition** logging behaviour are currently unverified. These are the behaviours this spec covers.

## Decisions at a glance

- **Scope:** `app.dart` only. Not `bootstrap.dart` (static platform init — hard to test without heavy channel mocking, low regression risk), not `main.dart`'s `runZonedGuarded` (collides with test harness's own zone), not `_resolveDeviceInfo` platform branches, not `initializeMappers`, not goldens.
- **One new test file:** `app/test/app_test.dart`. Four tests under a single `group('App initialisation', …)`.
- **Existing files untouched.** `widget_test.dart`'s ready-path smoke stays as-is — it covers `HomePage` render + l10n wiring, which this file does not.
- **Logger capture via `Logger.root.onRecord`.** `setUp` attaches a subscription collecting `LogRecord`s into a list; `tearDown` detaches. Cheap, direct, verifies the `ref.listen` contract the existing `app.dart` comment claims.

## Out of scope

- `_resolveDeviceInfo` platform branches (Android/iOS/Web/desktop `UnsupportedError`).
- `initializeMappers` round-trip.
- `FlutterError.onError` / `ErrorWidget.builder` from `main.dart`.
- `bootstrap()` end-to-end.
- Integration tests. All tests run under `flutter test` (widget-level), not `flutter test integration_test`.
- Visual regression / goldens. YAGNI until a real regression motivates them.

## Architecture

One file: `app/test/app_test.dart`.

Shared fixtures in the file's `main()`:

- `late List<LogRecord> records;` populated by a `Logger.root.onRecord.listen` subscription.
- `late StreamSubscription<LogRecord> _logSub;` detached in `tearDown`.
- `late SharedPreferences prefs;` mocked via `SharedPreferences.setMockInitialValues({})`.
- `AppDependencies stubDeps()` — identical shape to the one in `widget_test.dart`. Accept the small duplication now; if a third test file needs it, extract to `test/_helpers.dart`.

`TestWidgetsFlutterBinding.ensureInitialized()` runs first in `setUp` so the prefs mock has a binding to attach to.

## Section 1 — Test: loading state

Override `appDependenciesProvider` with a `Future` that never completes:

```dart
final completer = Completer<AppDependencies>();
await tester.pumpWidget(
  ProviderScope(
    overrides: [
      appDependenciesProvider.overrideWith((ref) => completer.future),
    ],
    child: const App(),
  ),
);
// CircularProgressIndicator spins forever (AnimationController.repeat);
// pumpAndSettle would time out. A single pump is enough — the initial
// build is synchronous in pumpWidget.
await tester.pump();
```

Assertions:
- `expect(find.byType(CircularProgressIndicator), findsOneWidget);`
- `expect(find.byType(HomePage), findsNothing);`
- `expect(find.byType(InitializationErrorScreen), findsNothing);`

This proves: loading state renders the `InitializationLoadingScreen` spinner, not the error screen, not the home page.

## Section 2 — Test: error state renders `InitializationErrorScreen`

Override with a throwing provider:

```dart
appDependenciesProvider.overrideWith(
  (ref) async => throw Exception('boot failed'),
),
```

`await tester.pumpAndSettle();`

Assertions:
- `expect(find.byType(InitializationErrorScreen), findsOneWidget);`
- `expect(find.text('Initialization Failed'), findsOneWidget);` — localized `initializationFailedTitle`.
- `expect(find.text('Exception: boot failed'), findsOneWidget);` — `error.toString()` rendered verbatim.
- `expect(find.widgetWithText(ElevatedButton, 'Retry'), findsOneWidget);` — localized `retryButton`.

This proves: the error path renders the right screen with the error text and a retry button. Does not assert on icons or styling — those are not behaviour.

## Section 3 — Test: retry invalidates the provider and recovers

This needs an override whose behaviour changes between the first and second calls, because `ref.invalidate(appDependenciesProvider)` re-executes the provider. Simplest approach: a mutable counter in the override closure.

```dart
var attempt = 0;
await tester.pumpWidget(
  ProviderScope(
    overrides: [
      appDependenciesProvider.overrideWith((ref) async {
        attempt++;
        if (attempt == 1) {
          throw Exception('boot failed');
        }
        return stubDeps();
      }),
    ],
    child: const App(),
  ),
);

await tester.pumpAndSettle();
expect(find.byType(InitializationErrorScreen), findsOneWidget);

await tester.tap(find.widgetWithText(ElevatedButton, 'Retry'));
await tester.pumpAndSettle();

expect(find.byType(HomePage), findsOneWidget);
expect(find.byType(InitializationErrorScreen), findsNothing);
expect(attempt, 2);
```

This proves: the retry button's `onPressed` really invalidates the provider, the provider really re-executes, and the UI really transitions from error to ready.

## Section 4 — Test: logs one severe record per transition into error

Before the test block, `setUp` attaches:

```dart
late List<LogRecord> records;
late StreamSubscription<LogRecord> logSub;

setUp(() async {
  TestWidgetsFlutterBinding.ensureInitialized();
  SharedPreferences.setMockInitialValues({});
  prefs = await SharedPreferences.getInstance();
  records = <LogRecord>[];
  logSub = Logger.root.onRecord.listen(records.add);
});

tearDown(() async {
  await logSub.cancel();
});
```

Test body:

```dart
bool isInitSevere(LogRecord r) =>
    r.level == Level.SEVERE &&
    r.message == 'App dependencies failed to initialize';

final overrides = [
  appDependenciesProvider.overrideWith(
    (ref) async => throw Exception('boot failed'),
  ),
];

await tester.pumpWidget(
  ProviderScope(
    overrides: overrides,
    // A keyed App lets us swap the widget identity on a second pumpWidget
    // call below, forcing App.build to re-run without re-transitioning the
    // provider. This is what distinguishes "log on transition" from "log on
    // every build". The same ProviderScope stays mounted across both pumps
    // so the error AsyncValue is preserved.
    child: const App(key: ValueKey('app-1')),
  ),
);
await tester.pumpAndSettle();

expect(records.where(isInitSevere), hasLength(1));

// Force App.build to run again with a fresh key. The provider state is
// unchanged (still AsyncError) so ref.listen MUST NOT refire.
await tester.pumpWidget(
  ProviderScope(
    overrides: overrides,
    child: const App(key: ValueKey('app-2')),
  ),
);
await tester.pumpAndSettle();

expect(records.where(isInitSevere), hasLength(1));
```

This proves: the `ref.listen` in `App.build` fires on the `AsyncLoading → AsyncError` transition exactly once, and does not re-fire when `App.build` runs again with the provider already in `AsyncError`. If someone refactors the logging out of `ref.listen` and into the body of `App.build`, the second `pumpAndSettle` produces a second log record and this test fails.

Note: simply pumping empty frames would not actually cause `App.build` to re-run (Riverpod only rebuilds on state change), so that approach wouldn't genuinely test the claim. Swapping the `App` widget's `key` forces Flutter to rebuild the subtree including `App.build` while leaving the `ProviderScope` (and therefore the cached error state) intact.

## Section 5 — Verification

From `app/`:

1. `flutter analyze` — clean.
2. `flutter test` — count goes from 5 to 9. All passing.
3. `dart run custom_lint` — clean.

## Done when

- `app/test/app_test.dart` exists with four tests under `group('App initialisation', …)`.
- Existing `widget_test.dart` and `theme_notifier_test.dart` untouched.
- `flutter analyze` + `flutter test` (9 passing) + `dart run custom_lint` all clean.
- No new production-code changes — `app.dart`, `app_dependencies.dart`, and friends are untouched by this spec.
