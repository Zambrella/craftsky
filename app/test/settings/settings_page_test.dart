import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/settings/widgets/clear_image_cache_tile.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

void main() {
  testWidgets(
    'SettingsPage renders title, ClearImageCacheTile, and SignOutTile',
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
      expect(find.text('Followers'), findsOneWidget);
      expect(find.text('Following'), findsOneWidget);
      expect(find.text('Find people from Instagram'), findsOneWidget);
      expect(find.textContaining(RegExp(r'\d+ followers')), findsNothing);
      expect(find.textContaining(RegExp(r'\d+ following')), findsNothing);
      expect(find.byType(ClearImageCacheTile), findsOneWidget);
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

    await tester.tap(find.text('Find people from Instagram'));
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/profile/settings/instagram');
    expect(find.text('Instagram route'), findsOneWidget);
  });
}
