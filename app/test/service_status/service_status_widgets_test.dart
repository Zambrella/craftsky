import 'dart:async';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/service_status_router_config.dart';
import 'package:craftsky_app/service_status/widgets/service_status_host.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import 'announcement_dismissal_test.dart' show FakeDismissalStore, announcement;
import 'service_status_controller_test.dart' show FakeStatusRepository;

void main() {
  // AT-002 / BR-001, FR-002, FR-005, NFR-003, RULE-001, RULE-003 / AC-003, AC-017, AC-019.
  testWidgets(
    'maintenance renders literal text, estimate and only status retry',
    (tester) async {
      final repository = FakeStatusRepository();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      final refresh = controller.refresh();
      repository.requests.single.complete(
        ServiceStatusDocument(
          mode: ServiceStatusMode.maintenance,
          revision: 'A',
          title: '<script>Maintenance</script>',
          message: 'https://example.invalid Plain public prose',
          estimatedRecoveryAt: DateTime.utc(2026, 10, 9, 14),
        ),
      );
      await tester.pump();
      await refresh;
      var operations = 0;
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => ServiceStatusHost(child: child!),
            home: Scaffold(
              body: TextButton(
                onPressed: () => operations++,
                child: const Text('Underlying operation'),
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      expect(find.text('<script>Maintenance</script>'), findsOneWidget);
      expect(
        find.text('https://example.invalid Plain public prose'),
        findsOneWidget,
      );
      expect(find.text('Try again'), findsOneWidget);
      expect(find.textContaining('Estimated recovery:'), findsOneWidget);
      expect(find.text('Dismiss'), findsNothing);
      await tester.tap(find.text('Underlying operation'), warnIfMissed: false);
      expect(operations, 0);
      await tester.tap(find.text('Try again'));
      expect(repository.requests, hasLength(2));
      await tester.pumpWidget(const SizedBox.shrink());
      container.dispose();
    },
  );
  testWidgets(
    'root back is consumed while covered and resumes after clearing',
    (tester) async {
      final repository = FakeStatusRepository();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(
            FakeDismissalStore(),
          ),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      final router = GoRouter(
        routes: [
          GoRoute(
            path: '/',
            builder: (context, state) => const Scaffold(body: Text('Home')),
          ),
          GoRoute(
            path: '/editor',
            builder: (context, state) => const Scaffold(body: Text('Editor')),
          ),
        ],
      );
      final config = serviceStatusRouterConfig(
        router,
        () => controller.maintenanceActive,
        dismissAnnouncement: () async {
          if (controller.announcement == null) return false;
          await controller.dismissAnnouncement();
          return true;
        },
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: config,
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => ServiceStatusHost(child: child!),
          ),
        ),
      );
      unawaited(router.push<void>('/editor'));
      await tester.pumpAndSettle();
      final maintenanceFetch = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          mode: ServiceStatusMode.maintenance,
          revision: 'A',
          title: 'Maintenance',
          message: 'Public message',
        ),
      );
      await tester.pump();
      await maintenanceFetch;
      expect(
        await (config.backButtonDispatcher! as RootBackButtonDispatcher)
            .didPopRoute(),
        isTrue,
      );
      expect(router.canPop(), isTrue);
      expect(find.text('Editor'), findsOneWidget);
      expect(find.text('Maintenance'), findsOneWidget);
      final clear = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          mode: ServiceStatusMode.normal,
          revision: 'B',
        ),
      );
      await tester.pump();
      await clear;
      expect(
        await (config.backButtonDispatcher! as RootBackButtonDispatcher)
            .didPopRoute(),
        isTrue,
      );
      await tester.pumpAndSettle();
      expect(router.canPop(), isFalse);
      expect(find.text('Home'), findsOneWidget);
      unawaited(router.push<void>('/editor'));
      await tester.pumpAndSettle();
      final noticeFetch = controller.refresh();
      repository.requests.last.complete(announcement('notice-back'));
      await tester.pump();
      await noticeFetch;
      expect(find.byType(CraftskyDialog), findsOneWidget);
      expect(
        await (config.backButtonDispatcher! as RootBackButtonDispatcher)
            .didPopRoute(),
        isTrue,
      );
      await tester.pump();
      expect(controller.announcement, isNull);
      expect(router.canPop(), isTrue);
      expect(find.text('Editor'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
      router.dispose();
      container.dispose();
    },
  );
  // AT-003 / BR-001, FR-002, FR-006 / AC-004.
  testWidgets(
    'announcement modal preserves the app, dismisses and is replaced',
    (tester) async {
      final repository = FakeStatusRepository();
      final container = ProviderContainer(
        overrides: [
          serviceStatusRepositoryProvider.overrideWithValue(repository),
          announcementDismissalStoreProvider.overrideWithValue(
            FakeDismissalStore(),
          ),
        ],
      );
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      var operations = 0;
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => ServiceStatusHost(child: child!),
            home: Scaffold(
              body: TextButton(
                onPressed: () => operations++,
                child: const Text('Use app'),
              ),
            ),
          ),
        ),
      );
      Future<void> accept(ServiceStatusDocument document) async {
        final refreshed = controller.refresh();
        repository.requests.last.complete(document);
        await tester.pump();
        await refreshed;
        await tester.pump();
      }

      final position = tester.getTopLeft(find.text('Use app'));
      await accept(announcement('A'));
      expect(find.byType(CraftskyDialog), findsOneWidget);
      expect(tester.getTopLeft(find.text('Use app')), position);
      expect(find.text('Notice'), findsOneWidget);
      expect(find.text('Use app').hitTestable(), findsNothing);
      expect(operations, 0);
      await tester.tap(find.text('Dismiss'));
      await tester.pump();
      expect(find.text('Notice'), findsNothing);
      await tester.tap(find.text('Use app'));
      expect(operations, 1);
      await accept(announcement('B'));
      expect(find.text('Notice'), findsOneWidget);
      await tester.pump();
      await tester.sendKeyEvent(LogicalKeyboardKey.escape);
      await tester.pump();
      expect(find.text('Notice'), findsNothing);
      await accept(announcement('C'));
      await tester.tapAt(const Offset(8, 8));
      await tester.pump();
      expect(find.text('Notice'), findsNothing);
      await accept(announcement('D'));
      await accept(
        const ServiceStatusDocument(
          mode: ServiceStatusMode.maintenance,
          revision: 'C',
          title: 'Maintenance',
          message: 'Public message',
        ),
      );
      expect(find.text('Notice'), findsNothing);
      expect(find.text('Maintenance'), findsOneWidget);
      expect(find.text('Dismiss'), findsNothing);
      await accept(
        const ServiceStatusDocument(
          mode: ServiceStatusMode.normal,
          revision: 'D',
        ),
      );
      expect(find.text('Maintenance'), findsNothing);
      await tester.tap(find.text('Use app'));
      expect(operations, 2);
      await tester.pumpWidget(const SizedBox.shrink());
      container.dispose();
    },
  );
}
