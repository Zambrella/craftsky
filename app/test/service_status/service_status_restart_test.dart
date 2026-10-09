import 'dart:io';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/widgets/service_status_host.dart';
import 'package:craftsky_app/service_status/widgets/service_status_lifecycle_host.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'announcement_dismissal_test.dart' show FakeDismissalStore, announcement;
import 'service_status_controller_test.dart'
    show FakeStatusRepository, maintenance;

void main() {
  // AT-007 / FR-006, FR-007, FR-008, NFR-001 / AC-009, AC-010, AC-011.
  testWidgets(
    'restart shares only dismissal; new fetch fails '
    'without restoring maintenance',
    (tester) async {
      final storage = FakeDismissalStore();
      ProviderContainer fresh(FakeStatusRepository repository) =>
          ProviderContainer(
            overrides: [
              serviceStatusRepositoryProvider.overrideWithValue(repository),
              announcementDismissalStoreProvider.overrideWithValue(storage),
            ],
          );
      Future<void> mount(ProviderContainer container) => tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: ServiceStatusLifecycleHost(
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => ServiceStatusHost(child: child!),
              home: const Scaffold(body: Text('Usable app')),
            ),
          ),
        ),
      );
      final oldRepository = FakeStatusRepository();
      final old = fresh(oldRepository);
      await mount(old);
      await tester.pump();
      oldRepository.requests.single.complete(announcement('A'));
      await tester.pump();
      await tester.pump();
      expect(find.text('Notice'), findsOneWidget);
      await tester.tap(find.text('Dismiss'));
      await tester.pump();
      expect(storage.revision, 'A');
      final oldController = old.read(serviceStatusControllerProvider.notifier);
      final maintenanceFetch = oldController.refresh();
      oldRepository.requests.last.complete(maintenance);
      await tester.pump();
      await maintenanceFetch;
      expect(find.text('Maintenance'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
      old.dispose();

      final repository = FakeStatusRepository();
      final restarted = fresh(repository);
      await mount(restarted);
      await tester.pump();
      expect(repository.requests, hasLength(1));
      repository.requests.single.completeError(
        const SocketException('Offline'),
      );
      await tester.pump();
      expect(find.text('Maintenance'), findsNothing);
      expect(find.text('Usable app'), findsOneWidget);
      final controller = restarted.read(
        serviceStatusControllerProvider.notifier,
      );
      final same = controller.refresh();
      repository.requests.last.complete(announcement('A'));
      await tester.pump();
      await same;
      expect(find.text('Notice'), findsNothing);
      final newer = controller.refresh();
      repository.requests.last.complete(announcement('B'));
      await tester.pump();
      await newer;
      expect(find.text('Notice'), findsOneWidget);
      final maintenanceAgain = controller.refresh();
      repository.requests.last.complete(maintenance);
      await tester.pump();
      await maintenanceAgain;
      expect(find.text('Maintenance'), findsOneWidget);
      expect(find.text('Notice'), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
      restarted.dispose();
    },
  );
}
