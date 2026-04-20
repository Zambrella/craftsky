import 'dart:async';

import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
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

    testWidgets('loading state renders CircularProgressIndicator', (
      tester,
    ) async {
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

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.byType(WelcomePage), findsNothing);
      expect(find.byType(InitializationErrorScreen), findsNothing);
    });

    testWidgets('error state renders InitializationErrorScreen', (
      tester,
    ) async {
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

      expect(find.byType(InitializationErrorScreen), findsOneWidget);
      expect(find.text('Initialization Failed'), findsOneWidget);
      expect(find.text('Exception: boot failed'), findsOneWidget);
      expect(find.widgetWithText(ElevatedButton, 'Retry'), findsOneWidget);
    });

    testWidgets('retry invalidates the provider and recovers to WelcomePage', (
      tester,
    ) async {
      var attempt = 0;

      await tester.pumpWidget(
        ProviderScope(
          // Disable Riverpod 3.x auto-retry so this test only advances past
          // the error state via the explicit Retry-button tap below.
          // ProviderContainer.defaultRetry schedules a delayed rebuild on
          // AsyncError, which would race `pumpAndSettle` to AsyncData before
          // the test gets to see the error state at all.
          retry: (_, _) => null,
          overrides: [
            appDependenciesProvider.overrideWith((ref) async {
              attempt++;
              if (attempt == 1) {
                throw Exception('boot failed');
              }
              return stubDeps();
            }),
            // Pin auth to pre-authenticated so the router lands on WelcomePage
            // regardless of the dev stub's current default.
            authStatusProvider.overrideWithValue(false),
          ],
          child: const App(),
        ),
      );
      await tester.pumpAndSettle();

      // Sanity check: we're in the error state before the retry.
      expect(find.byType(InitializationErrorScreen), findsOneWidget);

      await tester.tap(find.widgetWithText(ElevatedButton, 'Retry'));
      await tester.pumpAndSettle();

      expect(find.byType(WelcomePage), findsOneWidget);
      expect(find.byType(InitializationErrorScreen), findsNothing);
      expect(attempt, 2);
    });

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
            // Disable Riverpod 3.x auto-retry — otherwise the default
            // exponential-backoff retry keeps re-running the failing
            // builder, and each new AsyncError transition refires the
            // severe log, defeating this test's whole point.
            retry: (_, _) => null,
            overrides: overrides,
            child: const App(key: ValueKey('app-1')),
          ),
        );
        await tester.pumpAndSettle();

        expect(records.where(isInitSevere), hasLength(1));

        // Force App.build to run again with a fresh ValueKey. The
        // ProviderScope (and its cached AsyncError for
        // appDependenciesProvider) should persist across this second
        // pumpWidget call because the widget type at the root position
        // matches and the overrides list instance is identical — Flutter
        // reuses the Element, and ProviderScope's Element owns the
        // ProviderContainer. Riverpod's WidgetRef.listen has no
        // fireImmediately flag (by design — see flutter_riverpod 3.x
        // consumer.dart:496), so a new registration on an already-errored
        // provider does NOT fire. If someone refactors the logging out of
        // ref.listen and into App.build proper, the second pumpAndSettle
        // below would produce a second SEVERE record and this assertion
        // would fail.
        await tester.pumpWidget(
          ProviderScope(
            retry: (_, _) => null,
            overrides: overrides,
            child: const App(key: ValueKey('app-2')),
          ),
        );
        await tester.pumpAndSettle();

        expect(records.where(isInitSevere), hasLength(1));
      },
    );
  });
}
