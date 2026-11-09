import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/safety/safety_guide_page.dart';
import 'package:craftsky_app/safety/search_support_notice.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/auth_session_fakes.dart';

void main() {
  Widget host(Widget child) => MaterialApp(
    theme: AppTheme.lightThemeData,
    localizationsDelegates: appLocalizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: child,
  );
  testWidgets(
    'public route stays available without an account or initialization',
    (tester) async {
      final container = ProviderContainer(
        overrides: [
          authSessionProvider.overrideWith(SignedOutAuthSession.new),
          sessionRegistryProvider.overrideWith(_EmptyRegistry.new),
          activeAccountInitializationProvider.overrideWith((ref) => null),
        ],
      );
      addTearDown(container.dispose);
      final router = container.read(goRouterProvider)..go('/safety');
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            theme: AppTheme.lightThemeData,
            routerConfig: router,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(SafetyGuidePage), findsOneWidget);
      expect(router.routeInformationProvider.value.uri.path, '/safety');
      expect(tester.takeException(), isNull);
    },
  );
  testWidgets(
    'dismiss is local and a different submitted query restores support',
    (tester) async {
      Future<void> show(String q) async {
        await tester.pumpWidget(
          host(
            Scaffold(
              body: Column(
                children: [
                  SearchSupportNotice(query: q),
                  const Text('Search results remain available'),
                ],
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
      }

      await show('self-harm');
      expect(find.text('Find support'), findsOneWidget);
      await tester.tap(find.text('Dismiss'));
      await tester.pump();
      expect(find.text('Find support'), findsNothing);
      expect(find.text('Search results remain available'), findsOneWidget);
      await show('knitting');
      expect(find.text('Find support'), findsNothing);
      await show('suicide');
      expect(find.text('Find support'), findsOneWidget);
    },
  );
  testWidgets('guide is public and remains readable at large text', (
    tester,
  ) async {
    await tester.pumpWidget(
      host(
        const MediaQuery(
          data: MediaQueryData(textScaler: TextScaler.linear(2)),
          child: SafetyGuidePage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Safety and support'), findsOneWidget);
    expect(find.text('Find support'), findsOneWidget);
    await tester.scrollUntilVisible(
      find.text('For parents and carers'),
      600,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();
    expect(find.text('For parents and carers'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}

class _EmptyRegistry extends SessionRegistry {
  @override
  Future<registry.SessionRegistry> build() async =>
      registry.SessionRegistry.empty();
}
