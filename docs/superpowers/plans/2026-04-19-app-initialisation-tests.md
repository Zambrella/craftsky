# App Initialisation Tests Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add four widget tests to `app/test/app_test.dart` covering `App`'s initialisation behaviour — loading spinner, error screen wiring, retry round-trip, and the one-severe-record-per-transition logging contract.

**Architecture:** One new file. Four tests under a single `group('App initialisation', …)`. Shared `setUp` attaches a `Logger.root.onRecord` subscription into a fresh `List<LogRecord>` per test, mocks `SharedPreferences`, and resolves a `stubDeps()` helper identical to the one in `widget_test.dart`.

**Tech Stack:** `flutter_test` (widget testing), `flutter_riverpod` `ProviderScope.overrides`, `logging.Logger.root.onRecord`, Dart's `Completer` (for the never-resolving loading future).

---

## Spec

Approved design: [docs/superpowers/specs/2026-04-19-app-initialisation-tests-design.md](../specs/2026-04-19-app-initialisation-tests-design.md). Read it first.

## Binding rules

These are test files, so Flutter/Riverpod widget-design rules (one widget per concern, `ref.watch` in build, etc.) don't apply. What *does* apply:

- `.claude/rules/flutter.md` — use `logging` not `print`.
- `.claude/rules/riverpod.md` — `ProviderScope(overrides: [...])` is the idiomatic Riverpod test-override mechanism; `appDependenciesProvider.overrideWith((ref) => ...)` replaces the async body.

## Working directory

Repo root: `/Users/douglastodd/Projects/craftsky/.claude/worktrees/elated-jackson-20b896`. Run Flutter/Dart commands from `app/`. Base SHA: whatever `git rev-parse HEAD` returns before this plan (as of writing, `a451f24`).

## File map

- **Create** `app/test/app_test.dart` — the only new file.
- **Unchanged:** `app/test/widget_test.dart` (existing ready-path smoke test), `app/test/theme/theme_notifier_test.dart`, every file under `app/lib/`. This is a test-only change; no production-code modifications.

## Note on TDD

The plan is structured red-green per test because this is the one time writing the failing test first genuinely catches something: it verifies that our stub wiring (provider overrides, logger subscription, harness setup) produces the fixture state we think it does. Running each test before the assertions are complete, seeing the failure mode, then adding assertions, catches "my mock didn't actually set up the error state I thought" — a real failure mode in Riverpod override testing.

For each task: write the setup + skeleton test (no assertions or partial assertions), run it and confirm it fails *for the right reason*, add the assertions, re-run, commit.

---

## Chunk 1: Full implementation

### Task 1: File scaffold + shared `setUp`/`tearDown`

**Files:**
- Create: `app/test/app_test.dart`

- [ ] **Step 1: Write the scaffold with no tests**

Full file contents:

```dart
import 'dart:async';

import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/router/home_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:pub_semver/pub_semver.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  group('App initialisation', () {
    late SharedPreferences prefs;
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

    AppDependencies stubDeps() => AppDependencies(
      packageInfo: PackageInfo(
        appName: 'craftsky_app',
        packageName: 'social.craftsky.app',
        version: '1.0.0',
        buildNumber: '1',
      ),
      deviceInfo: CraftskyDeviceInfo(
        platform: 'Test',
        deviceId: 'test',
        model: 'test',
        brand: 'test',
        osVersion: '0',
      ),
      sharedPreferences: prefs,
      appVersion: Version.parse('1.0.0'),
    );

    // Tests go here.
  });
}
```

Notes:
- `stubDeps()` duplicates `widget_test.dart`'s helper on purpose. If a third test file needs it, extract to `test/_helpers.dart` — not before.
- `LogRecord`, `Level`, and `Logger` come from `package:logging/logging.dart`.
- The `late` fields are captured by the tests via Dart closure scope.

- [ ] **Step 2: Run the file — confirm zero tests, zero failures**

Run (from `app/`): `flutter test test/app_test.dart`

Expected: `All tests passed!` (0 tests, but `flutter test` reports success). If it fails to compile, fix before proceeding — usually a missing import.

- [ ] **Step 3: Commit the scaffold**

```bash
git add app/test/app_test.dart
git commit -m "test(app): scaffold app_test.dart with shared logger/prefs setUp"
```

### Task 2: Test — loading state renders `CircularProgressIndicator`

**Files:**
- Modify: `app/test/app_test.dart`

- [ ] **Step 1: Write the test with failing assertions**

Insert this `testWidgets` block where the comment `// Tests go here.` is:

```dart
testWidgets('loading state renders CircularProgressIndicator', (tester) async {
  // Future never completes → appDependenciesProvider stays in AsyncLoading.
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

  // Wrong on purpose — expect these to fail to confirm the test is wired.
  expect(find.byType(CircularProgressIndicator), findsNothing);
  expect(find.byType(HomePage), findsOneWidget);
});
```

- [ ] **Step 2: Run test, confirm it fails for the right reason**

Run: `flutter test test/app_test.dart`

Expected: the test fails. The first `expect` should fail with something like `Expected: no matching candidates / Actual: _MatcherFound at …/CircularProgressIndicator`. This confirms the `ProviderScope` override is actually producing the loading state and the spinner is in the tree. If it fails for a different reason (e.g. the provider resolved), investigate before proceeding.

- [ ] **Step 3: Flip the assertions to the correct sign**

Replace the two wrong-on-purpose `expect`s with:

```dart
  expect(find.byType(CircularProgressIndicator), findsOneWidget);
  expect(find.byType(HomePage), findsNothing);
  expect(find.byType(InitializationErrorScreen), findsNothing);
});
```

Add `import 'package:craftsky_app/app.dart'` is already there; `InitializationErrorScreen` comes from `app.dart` already imported.

- [ ] **Step 4: Run test, confirm it passes**

Run: `flutter test test/app_test.dart`

Expected: `+1: All tests passed!`.

- [ ] **Step 5: Commit**

```bash
git add app/test/app_test.dart
git commit -m "test(app): assert CircularProgressIndicator renders while deps resolve"
```

### Task 3: Test — error state renders `InitializationErrorScreen`

**Files:**
- Modify: `app/test/app_test.dart`

- [ ] **Step 1: Write the test with failing assertions**

Append below the loading test (still inside `group('App initialisation', …)`):

```dart
testWidgets('error state renders InitializationErrorScreen', (tester) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        appDependenciesProvider.overrideWith(
          (ref) async => throw Exception('boot failed'),
        ),
      ],
      child: const App(),
    ),
  );
  await tester.pumpAndSettle();

  // Wrong on purpose.
  expect(find.byType(InitializationErrorScreen), findsNothing);
});
```

- [ ] **Step 2: Run, confirm fail-for-right-reason**

Run: `flutter test test/app_test.dart`

Expected: this specific test fails with `Expected: no matching candidates / Actual: Found one widget … InitializationErrorScreen`. That confirms the thrown-exception override actually produces the error state.

- [ ] **Step 3: Replace with the full assertion set**

```dart
  expect(find.byType(InitializationErrorScreen), findsOneWidget);
  expect(find.text('Initialization Failed'), findsOneWidget);
  expect(find.text('Exception: boot failed'), findsOneWidget);
  expect(find.widgetWithText(ElevatedButton, 'Retry'), findsOneWidget);
});
```

- [ ] **Step 4: Run, confirm passes**

Run: `flutter test test/app_test.dart`

Expected: `+2: All tests passed!`.

- [ ] **Step 5: Commit**

```bash
git add app/test/app_test.dart
git commit -m "test(app): assert InitializationErrorScreen with error text and retry"
```

### Task 4: Test — retry invalidates and recovers

**Files:**
- Modify: `app/test/app_test.dart`

- [ ] **Step 1: Write the test with a failing assertion**

Append:

```dart
testWidgets('retry invalidates the provider and recovers to HomePage', (tester) async {
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

  // Sanity check: we're in the error state before the retry.
  expect(find.byType(InitializationErrorScreen), findsOneWidget);

  await tester.tap(find.widgetWithText(ElevatedButton, 'Retry'));
  await tester.pumpAndSettle();

  // Wrong on purpose.
  expect(find.byType(HomePage), findsNothing);
});
```

- [ ] **Step 2: Run, confirm fail-for-right-reason**

Run: `flutter test test/app_test.dart`

Expected: the retry test fails with `Found one widget … HomePage`. Confirms that tapping Retry really invalidates the provider, the override re-runs (this time with `attempt == 2`, returning `stubDeps()`), and the UI transitions to `HomePage`.

- [ ] **Step 3: Replace with the full assertion set**

```dart
  expect(find.byType(HomePage), findsOneWidget);
  expect(find.byType(InitializationErrorScreen), findsNothing);
  expect(attempt, 2);
});
```

- [ ] **Step 4: Run, confirm passes**

Run: `flutter test test/app_test.dart`

Expected: `+3: All tests passed!`.

- [ ] **Step 5: Commit**

```bash
git add app/test/app_test.dart
git commit -m "test(app): assert Retry button invalidates provider and recovers"
```

### Task 5: Test — logs exactly once per transition into error

**Files:**
- Modify: `app/test/app_test.dart`

- [ ] **Step 1: Write the test with a failing assertion**

Append:

```dart
testWidgets(
  'logs one severe record per transition into error (not per rebuild)',
  (tester) async {
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
        child: const App(key: ValueKey('app-1')),
      ),
    );
    await tester.pumpAndSettle();

    // First assertion intentionally wrong so we can confirm the severe record
    // really did land in `records`.
    expect(records.where(isInitSevere), hasLength(0));
  },
);
```

- [ ] **Step 2: Run, confirm fail-for-right-reason**

Run: `flutter test test/app_test.dart`

Expected: this test fails with `Expected: has length of <0> / Actual: has length of <1>`. Confirms the logger subscription is attached and the `ref.listen` in `App.build` fired exactly once.

- [ ] **Step 3: Replace with the full assertion — transition-not-rebuild**

Delete the wrong-on-purpose expect and complete the test:

```dart
    expect(records.where(isInitSevere), hasLength(1));

    // Force App.build to run again with a fresh ValueKey. The ProviderScope
    // (and its cached AsyncError for appDependenciesProvider) persists across
    // the pumpWidget call because the overrides list is identical, so only
    // the App widget identity changes. Riverpod's WidgetRef.listen has no
    // fireImmediately flag (by design — see flutter_riverpod 3.x
    // consumer.dart), so a new registration on an already-errored provider
    // does NOT fire. If someone refactors the logging out of ref.listen and
    // into App.build proper, the second pumpAndSettle below would produce a
    // second SEVERE record and this assertion would fail.
    await tester.pumpWidget(
      ProviderScope(
        overrides: overrides,
        child: const App(key: ValueKey('app-2')),
      ),
    );
    await tester.pumpAndSettle();

    expect(records.where(isInitSevere), hasLength(1));
  },
);
```

- [ ] **Step 4: Run, confirm passes**

Run: `flutter test test/app_test.dart`

Expected: `+4: All tests passed!`.

- [ ] **Step 5: Commit**

```bash
git add app/test/app_test.dart
git commit -m "test(app): assert severe-log fires once per error transition"
```

### Task 6: Full verification gate

- [ ] **Step 1: Whole-suite run**

Run (from `app/`): `flutter test`

Expected: `+9: All tests passed!` — 4 theme tests, 1 existing smoke test, 4 new init tests.

- [ ] **Step 2: Analyze**

Run: `flutter analyze`

Expected: `No issues found!`.

- [ ] **Step 3: custom_lint**

Run: `dart run custom_lint`

Expected: `No issues found!`.

- [ ] **Step 4: Format**

Run: `dart format .`

Expected: no changes (trailing-commas preserve + reasonable layout means the hand-written test file should already be canonical). If format edits the new file, fold those changes into an amend of the last commit.

---

## Done when

- `app/test/app_test.dart` exists with four tests under `group('App initialisation', …)`.
- `widget_test.dart` and `theme_notifier_test.dart` untouched.
- All production code under `app/lib/` untouched.
- `flutter test` shows 9 passing.
- `flutter analyze` and `dart run custom_lint` both clean.
- Five commits on `claude/elated-jackson-20b896` (scaffold + one per test).
