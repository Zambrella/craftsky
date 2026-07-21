import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

void main() {
  setUpAll(initializeMappers);

  InstagramMatchNotification match({int count = 3, bool capped = false}) =>
      CraftskyNotification.fromMap({
            'id': '00000000-0000-0000-0000-000000000321',
            'kind': 'system',
            'type': 'instagramMatch',
            'createdAt': '2026-07-19T12:00:00Z',
            'indexedAt': '2026-07-19T12:04:00Z',
            'system': {
              'count': count,
              'countCapped': capped,
              'destination': 'instagramMigration',
            },
          })
          as InstagramMatchNotification;

  testWidgets('IT-017 renders generic actorless bounded match copy', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Column(
              children: [
                NotificationRow(notification: match(count: 1)),
                NotificationRow(notification: match()),
                NotificationRow(notification: match(count: 99, capped: true)),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(ProfileAvatar), findsNothing);
    expect(
      find.text('A new Instagram match is ready to review'),
      findsOneWidget,
    );
    expect(
      find.text('3 new Instagram matches are ready to review'),
      findsOneWidget,
    );
    expect(
      find.text('99+ new Instagram matches are ready to review'),
      findsOneWidget,
    );
    for (final forbidden in ['Alice', '@', 'did:', 'IGSID']) {
      expect(find.textContaining(forbidden), findsNothing);
    }
  });

  testWidgets('IT-017 opens the typed Instagram migration route', (
    tester,
  ) async {
    GoRouterState? destination;
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => Scaffold(
            body: NotificationRow(notification: match()),
          ),
        ),
        GoRoute(
          path: const InstagramMigrationRoute().location,
          builder: (_, state) {
            destination = state;
            return const Scaffold(body: Text('Instagram migration'));
          },
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('3 new Instagram matches are ready to review'));
    await tester.pumpAndSettle();

    expect(destination?.uri.path, const InstagramMigrationRoute().location);
  });
}
