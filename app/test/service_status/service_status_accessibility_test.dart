import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/widgets/service_status_host.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'announcement_dismissal_test.dart' show FakeDismissalStore;
import 'service_status_controller_test.dart' show FakeStatusRepository;

void main() {
  // AT-009 / NFR-003 / AC-017.
  testWidgets(
    'maintenance traps focus and restores the retained editor focus',
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
      final editor = FocusNode(debugLabel: 'retained-editor');
      final controller = container.read(
        serviceStatusControllerProvider.notifier,
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (_, child) => ServiceStatusHost(child: child!),
            home: Scaffold(body: TextField(focusNode: editor)),
          ),
        ),
      );
      editor.requestFocus();
      await tester.pump();
      expect(editor.hasFocus, isTrue);
      final fetch = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          mode: ServiceStatusMode.maintenance,
          revision: 'A',
          title: 'Maintenance',
          message: 'Public',
        ),
      );
      await tester.pump();
      await fetch;
      await tester.pump();
      expect(editor.hasFocus, isFalse);
      expect(FocusManager.instance.primaryFocus?.context?.widget, isNotNull);
      final clear = controller.refresh();
      repository.requests.last.complete(
        const ServiceStatusDocument(
          mode: ServiceStatusMode.normal,
          revision: 'B',
        ),
      );
      await tester.pump();
      await clear;
      await tester.pump();
      expect(editor.hasFocus, isTrue);
      await tester.pumpWidget(const SizedBox.shrink());
      container.dispose();
      editor.dispose();
    },
  );
  for (final dark in [false, true]) {
    for (final width in [320.0, 1280.0]) {
      for (final mode in [
        ServiceStatusMode.maintenance,
        ServiceStatusMode.announcement,
      ]) {
        testWidgets(
          '$mode dark=$dark width=$width long scaled text is reachable',
          (tester) async {
            await tester.binding.setSurfaceSize(Size(width, 640));
            addTearDown(() => tester.binding.setSurfaceSize(null));
            final semantics = tester.ensureSemantics();
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
            final fetch = controller.refresh();
            repository.requests.last.complete(
              ServiceStatusDocument(
                mode: mode,
                revision: 'A',
                title: 'Notice ${'x' * 153}',
                message: 'Public ${'long text\n' * 239}',
                estimatedRecoveryAt: DateTime.utc(2026),
              ),
            );
            await tester.pump();
            await fetch;
            await tester.pumpWidget(
              UncontrolledProviderScope(
                container: container,
                child: MaterialApp(
                  theme: dark
                      ? AppTheme.darkThemeData
                      : AppTheme.lightThemeData,
                  localizationsDelegates:
                      AppLocalizations.localizationsDelegates,
                  supportedLocales: AppLocalizations.supportedLocales,
                  builder: (context, child) => MediaQuery(
                    data: MediaQuery.of(
                      context,
                    ).copyWith(textScaler: const TextScaler.linear(1.5)),
                    child: ServiceStatusHost(child: child!),
                  ),
                  home: Scaffold(
                    body: TextButton(
                      onPressed: () {},
                      child: const Text('Covered action'),
                    ),
                  ),
                ),
              ),
            );
            await tester.pump();
            expect(tester.takeException(), isNull);
            final action = find.text(
              mode == ServiceStatusMode.maintenance ? 'Try again' : 'Dismiss',
            );
            await tester.ensureVisible(action);
            await tester.pump();
            expect(action.hitTestable(), findsOneWidget);
            expect(
              tester.getSemantics(action).label,
              mode == ServiceStatusMode.maintenance ? 'Try again' : 'Dismiss',
            );
            expect(find.bySemanticsLabel('Covered action'), findsNothing);
            expect(tester.takeException(), isNull);
            await tester.pumpWidget(const SizedBox.shrink());
            container.dispose();
            semantics.dispose();
          },
        );
      }
    }
  }
}
