import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

void main() {
  testWidgets(
    'SettingsPage renders the expanded hierarchy and SignOutTile',
    (tester) async {
      await tester.pumpWidget(
        const ProviderScope(
          child: MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: SettingsPage(),
          ),
        ),
      );
      expect(find.text('Settings'), findsWidgets);
      expect(find.text('Languages'), findsOneWidget);
      expect(find.text('Customisation'), findsOneWidget);
      expect(find.text('Followers'), findsOneWidget);
      expect(find.text('Following'), findsOneWidget);
      await tester.scrollUntilVisible(
        find.text('Find people from Instagram'),
        300,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.text('Find people from Instagram'), findsOneWidget);
      expect(find.text('Saved posts'), findsNothing);
      expect(find.text('Scheduled posts'), findsNothing);
      expect(find.text('Drafts'), findsNothing);
      expect(find.textContaining(RegExp(r'\d+ followers')), findsNothing);
      expect(find.textContaining(RegExp(r'\d+ following')), findsNothing);
      expect(find.text('Clear image cache'), findsNothing);
      await tester.scrollUntilVisible(
        find.byType(SignOutTile),
        300,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.byType(SignOutTile), findsOneWidget);
    },
  );

  testWidgets('Instagram settings entry opens the typed migration location', (
    tester,
  ) async {
    final router = GoRouter(
      initialLocation: '/settings',
      routes: [
        GoRoute(
          path: '/settings',
          builder: (_, _) => const SettingsPage(),
        ),
        GoRoute(
          path: '/profile/settings/instagram',
          builder: (_, _) => const Scaffold(body: Text('Instagram route')),
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          routerConfig: router,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(
      find.text('Find people from Instagram'),
      300,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Find people from Instagram'));
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/profile/settings/instagram');
    expect(find.text('Instagram route'), findsOneWidget);
  });
}
