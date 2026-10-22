import 'dart:async';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/service_status/data/status_actions.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/providers/status_runtime_provider.dart';
import 'package:craftsky_app/service_status/service_status_router_config.dart';
import 'package:craftsky_app/service_status/widgets/service_status_host.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

import 'announcement_dismissal_test.dart' show FakeDismissalStore;
import 'service_status_controller_test.dart' show FakeStatusRepository;
import 'support/status_test_fixtures.dart';

void main() {
  // AT-004 / FR-004, NFR-003 / AC-007, AC-020
  testWidgets(
    'late store failure after gate removal is handled safely',
    (tester) async {
      final repository = FakeStatusRepository();
      final launch = Completer<bool>();
      final container = ProviderContainer(
        overrides: [
          ...statusRuntimeOverrides,
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          statusExternalLauncherProvider.overrideWithValue(
            (uri) => launch.future,
          ),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (_, child) => ServiceStatusHost(child: child!),
            home: const Scaffold(body: Text('Retained app')),
          ),
        ),
      );
      await tester.pump();
      final policy = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          revision: 'U',
          minimumVersions: StatusMinimumVersions(ios: '1.9.0'),
        ),
      );
      await tester.pump();
      await policy;
      await tester.tap(find.text('Update'));
      await tester.pump();
      final clear = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(revision: 'D2'),
      );
      await tester.pump();
      await clear;
      launch.completeError(StateError('late external launch failure'));
      await tester.pump();
      expect(tester.takeException(), isNull);
      expect(find.text('Retained app'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
      container.dispose();
    },
  );

  // AT-005 / FR-005, NFR-001 / AC-008
  testWidgets('web reload is requested once only on explicit user action', (
    tester,
  ) async {
    final repository = FakeStatusRepository();
    var reloads = 0;
    final container = ProviderContainer(
      overrides: [
        statusRuntimeOverrides.last,
        statusPlatformProvider.overrideWithValue(StatusPlatform.web),
        statusReloadProvider.overrideWithValue(() => reloads++),
        serviceStatusRepositoryProvider.overrideWithValue(repository),
      ],
    );
    final controller = container.read(serviceStatusControllerProvider.notifier);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (_, child) => ServiceStatusHost(child: child!),
          home: const Scaffold(body: Text('App')),
        ),
      ),
    );
    await tester.pump();
    for (var i = 0; i < 2; i++) {
      final refresh = controller.refresh();
      repository.requests.last.complete(
        ServiceStatusDocument(
          revision: 'D$i',
          minimumVersions: const StatusMinimumVersions(web: '1.9.0'),
        ),
      );
      await tester.pump();
      await refresh;
    }
    expect(reloads, 0);
    await tester.tap(find.text('Reload'));
    await tester.pump();
    expect(reloads, 1);
    await tester.tap(find.text('Reload'));
    await tester.pump();
    expect(
      reloads,
      1,
      reason: 'A pending document navigation fences repeated taps',
    );
    expect(find.text('Update required'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    container.dispose();
  });

  // AT-004 / FR-004 / AC-007
  for (final platform in [StatusPlatform.ios, StatusPlatform.android]) {
    testWidgets(
      'mobile update opens official $platform listing, '
      'fences taps and retries failures',
      (tester) async {
        final repository = FakeStatusRepository();
        final requests = <Uri>[];
        final completion = Completer<bool>();
        var fail = true;
        var throwLaunch = false;
        final container = ProviderContainer(
          overrides: [
            statusRuntimeOverrides.last,
            statusPlatformProvider.overrideWithValue(platform),
            serviceStatusRepositoryProvider.overrideWithValue(repository),
            statusExternalLauncherProvider.overrideWithValue((uri) {
              requests.add(uri);
              if (throwLaunch) throw StateError('Launcher unavailable');
              return fail ? completion.future : Future.value(true);
            }),
          ],
        );
        final controller = container.read(
          serviceStatusControllerProvider.notifier,
        );
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (_, child) => ServiceStatusHost(child: child!),
              home: const Scaffold(body: Text('App')),
            ),
          ),
        );
        await tester.pump();
        final refresh = controller.refresh();
        repository.requests.last.complete(
          const ServiceStatusDocument(
            revision: 'D',
            minimumVersions: StatusMinimumVersions(
              ios: '1.9.0',
              android: '1.9.0',
            ),
          ),
        );
        await tester.pump();
        await refresh;
        await tester.tap(find.text('Update'));
        await tester.pump();
        await tester.tap(find.text('Update'));
        await tester.pump();
        expect(requests, [
          Uri.parse(
            platform == StatusPlatform.ios
                ? 'https://apps.apple.com/app/id6812428632'
                : 'https://play.google.com/store/apps/details?id=social.craftsky.app',
          ),
        ]);
        completion.complete(false);
        await tester.pump();
        expect(
          find.text('Could not open this action. Please try again.'),
          findsOneWidget,
        );
        fail = false;
        await tester.tap(find.text('Update'));
        await tester.pump();
        expect(requests, hasLength(2));
        expect(find.text('Update required'), findsOneWidget);
        expect(
          find.text('Could not open this action. Please try again.'),
          findsNothing,
        );
        throwLaunch = true;
        await tester.tap(find.text('Update'));
        await tester.pump();
        expect(
          find.text('Could not open this action. Please try again.'),
          findsOneWidget,
        );
        await tester.pumpWidget(const SizedBox.shrink());
        container.dispose();
      },
    );
  }

  // AT-003 / FR-003, NFR-001 / AC-004, AC-006
  testWidgets(
    'update covers retained editor, consumes back '
    'and cannot be escaped by navigation',
    (tester) async {
      final repository = FakeStatusRepository();
      final container = ProviderContainer(
        overrides: [
          ...statusRuntimeOverrides,
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(
            FakeDismissalStore(),
          ),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      final text = TextEditingController(text: 'Unsaved knitting notes');
      final router = GoRouter(
        routes: [
          GoRoute(
            path: '/',
            builder: (_, _) => Scaffold(body: TextField(controller: text)),
          ),
          GoRoute(
            path: '/other',
            builder: (_, _) => const Scaffold(body: Text('Other route')),
          ),
        ],
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            routerConfig: serviceStatusRouterConfig(
              router,
              () => controller.blockingActive,
            ),
            builder: (_, child) => ServiceStatusHost(child: child!),
          ),
        ),
      );
      await tester.pump();
      final refresh = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          revision: 'D1',
          minimumVersions: StatusMinimumVersions(ios: '1.9.0'),
        ),
      );
      await tester.pump();
      await refresh;
      expect(find.text('Update required'), findsOneWidget);
      expect(find.text('Dismiss'), findsNothing);
      expect(await tester.binding.handlePopRoute(), isTrue);
      await tester.sendKeyEvent(LogicalKeyboardKey.escape);
      router.go('/other');
      await tester.pumpAndSettle();
      expect(find.text('Update required'), findsOneWidget);
      expect(text.text, 'Unsaved knitting notes');
      final clear = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(revision: 'D2'),
      );
      await tester.pump();
      await clear;
      expect(find.text('Update required'), findsNothing);
      expect(find.text('Other route'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
      router.dispose();
      container.dispose();
      text.dispose();
    },
  );
}
