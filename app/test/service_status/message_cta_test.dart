import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/service_status/data/status_actions.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/providers/service_status_controller.dart';
import 'package:craftsky_app/service_status/widgets/service_status_host.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import 'announcement_dismissal_test.dart' show FakeDismissalStore;
import 'service_status_controller_test.dart' show FakeStatusRepository;
import 'support/status_test_fixtures.dart';

void main() {
  // AT-008 / FR-008, FR-010 / AC-014, AC-022
  for (final external in [false, true]) {
    testWidgets(
      'message CTA $external retries failure then dismisses after activation',
      (tester) async {
        final repository = FakeStatusRepository();
        final store = FakeDismissalStore();
        final launches = <String>[];
        var success = false;
        final container = ProviderContainer(
          overrides: [
            ...statusRuntimeOverrides,
            serviceStatusRepositoryProvider.overrideWithValue(repository),
            announcementDismissalStoreProvider.overrideWithValue(store),
            statusExternalLauncherProvider.overrideWithValue((uri) async {
              launches.add(uri.toString());
              return success;
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
              builder: (_, child) => ServiceStatusHost(
                onDestination: (name) async {
                  launches.add(name);
                  return success;
                },
                child: child!,
              ),
              home: const Scaffold(body: Text('App')),
            ),
          ),
        );
        await tester.pump();
        final refresh = controller.refresh();
        repository.requests.last.complete(
          ServiceStatusDocument(
            revision: 'D',
            message: StatusMessage(
              revision: 'A',
              minVersion: '1.8.0',
              title: 'News',
              description: 'Body',
              cta: StatusMessageCta(
                text: 'Read more',
                url: external ? 'https://example.test/news' : null,
                destination: external ? null : 'settings',
              ),
            ),
          ),
        );
        await tester.pump();
        await refresh;
        await tester.tap(find.text('Read more'));
        await tester.pump();
        expect(find.text('News'), findsOneWidget);
        expect(store.revision, isNull);
        expect(
          find.text('Could not open this action. Please try again.'),
          findsOneWidget,
        );
        success = true;
        await tester.tap(find.text('Read more'));
        await tester.pump();
        expect(
          launches,
          List.filled(2, external ? 'https://example.test/news' : 'settings'),
        );
        expect(find.text('News'), findsNothing);
        expect(store.revision, 'A');
        await tester.pumpWidget(const SizedBox.shrink());
        container.dispose();
      },
    );
  }
}
