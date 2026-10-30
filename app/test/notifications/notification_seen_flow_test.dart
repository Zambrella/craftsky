import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/notifications/providers/notification_filter_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

import '../test_support/notification_text.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('a filtered visit acknowledges newness only on departure', (
    tester,
  ) async {
    final newness = _RecordingNewnessRepository();
    final container = ProviderContainer.test(
      overrides: [
        notificationRepositoryProvider.overrideWithValue(
          _QueuedNotificationRepository([
            Future.value(const NotificationPage(items: [])),
            Future.value(const NotificationPage(items: [])),
          ]),
        ),
        notificationNewnessRepositoryProvider.overrideWithValue(newness),
      ],
    );
    container
        .read(notificationFilterProvider.notifier)
        .select(NotificationCategory.mention);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const NotificationsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(newness.seenCalls, 0);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
    expect(newness.seenCalls, 1);
  });

  testWidgets(
    'marks seen on departure only after a successful render',
    (tester) async {
      final firstPage = Completer<NotificationPage>();
      final list = _QueuedNotificationRepository([
        firstPage.future,
        Future.value(NotificationPage(items: [_follow('one')])),
        Future.value(const NotificationPage(items: [])),
      ]);
      final newness = _RecordingNewnessRepository();
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          notificationRepositoryProvider.overrideWithValue(list),
          notificationNewnessRepositoryProvider.overrideWithValue(newness),
        ],
      );

      await container.read(notificationNewCountProvider.future);
      newness.countCalls = 0;
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const NotificationsPage(),
          ),
        ),
      );
      await tester.pump();
      expect(newness.seenCalls, 0);
      expect(newness.countCalls, 0);

      firstPage.completeError(Exception('list failed'));
      await tester.pumpAndSettle();
      expect(container.read(notificationsProvider).hasError, isTrue);
      expect(container.read(notificationsProvider).value, isNull);
      expect(newness.seenCalls, 0);
      expect(newness.countCalls, 0);

      await tester.tap(find.text('Retry'));
      await tester.pumpAndSettle();
      expect(notificationText('Alice followed you'), findsOneWidget);
      expect(newness.seenCalls, 0);
      expect(newness.countCalls, 0);

      container.invalidate(notificationsProvider);
      await tester.pumpAndSettle();
      expect(find.text('No notifications yet.'), findsOneWidget);
      expect(newness.seenCalls, 0);
      expect(newness.countCalls, 0);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      expect(newness.seenCalls, 1);
      expect(newness.countCalls, 1);
    },
  );
  testWidgets(
    'new highlights survive refresh and filters, '
    'and clear after leaving a retained tab',
    (tester) async {
      final newness = _RecordingNewnessRepository();
      final seen = Completer<void>();
      newness.seenResult = seen.future;
      final list = _VisitRepository(newness);
      final container = ProviderContainer.test(
        overrides: [
          notificationRepositoryProvider.overrideWithValue(list),
          notificationNewnessRepositoryProvider.overrideWithValue(newness),
        ],
      );
      final router = GoRouter(
        initialLocation: '/notifications',
        routes: [
          StatefulShellRoute.indexedStack(
            builder: (context, state, shell) => shell,
            branches: [
              StatefulShellBranch(
                routes: [
                  GoRoute(
                    path: '/notifications',
                    builder: (_, _) => const NotificationsPage(),
                  ),
                ],
              ),
              StatefulShellBranch(
                routes: [
                  GoRoute(
                    path: '/elsewhere',
                    builder: (_, _) => const Scaffold(body: Text('Elsewhere')),
                  ),
                ],
              ),
            ],
          ),
          GoRoute(
            path: '/detail',
            builder: (_, _) => const Scaffold(body: Text('Detail')),
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: router,
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();
      final highlight = find.byKey(
        const ValueKey('new-notification-highlight'),
      );
      expect(highlight, findsOneWidget);
      expect(newness.seenCalls, 0);
      container.invalidate(notificationsProvider);
      await tester.pumpAndSettle();
      expect(highlight, findsOneWidget);
      container
          .read(notificationFilterProvider.notifier)
          .select(NotificationCategory.follow);
      await tester.pumpAndSettle();
      expect(highlight, findsOneWidget);
      container.read(notificationFilterProvider.notifier).select(null);
      await tester.pumpAndSettle();
      router.go('/elsewhere');
      await tester.pumpAndSettle();
      expect(newness.seenCalls, 1);
      router.go('/notifications');
      await tester.pump();
      expect(highlight, findsNothing);
      seen.complete();
      await tester.pumpAndSettle();
      expect(notificationText('Alice followed you'), findsOneWidget);
      expect(highlight, findsNothing);
      // A pushed detail route also ends the visit while the tab stays mounted.
      unawaited(router.push<void>('/detail'));
      await tester.pumpAndSettle();
      expect(newness.seenCalls, 2);
      list.hasNew = true;
      router.pop();
      await tester.pumpAndSettle();
      expect(highlight, findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}

FollowNotification _follow(String rkey, {bool isNew = false}) =>
    CraftskyNotification.fromMap({
      'isNew': isNew,
      'id': '00000000-0000-0000-0000-000000000001',
      'uri': 'at://did:plc:alice/app.bsky.graph.follow/$rkey',
      'cid': 'bafy$rkey',
      'rkey': rkey,
      'type': 'follow',
      'actor': {
        'did': 'did:plc:alice',
        'handle': 'alice.craftsky.social',
        'displayName': 'Alice',
      },
      'createdAt': '2026-05-28T13:00:00Z',
      'indexedAt': '2026-05-28T13:00:01Z',
    }) as FollowNotification;

final class _QueuedNotificationRepository implements NotificationRepository {
  new(this.responses);

  final List<Future<NotificationPage>> responses;

  @override
  Future<NotificationPage> list({
    String? cursor,
    int? limit,
    NotificationCategory? category,
  }) => responses.removeAt(0);
}

final class _RecordingNewnessRepository
    implements NotificationNewnessRepository {
  int seenCalls = 0;
  int countCalls = 0;
  Future<void>? seenResult;

  @override
  Future<int> count() async {
    countCalls++;
    return 7;
  }

  @override
  Future<void> markSeen() async {
    seenCalls++;
    await seenResult;
  }
}

final class _VisitRepository implements NotificationRepository {
  new(this.newness);
  final _RecordingNewnessRepository newness;
  bool hasNew = false;
  @override
  Future<NotificationPage> list({
    String? cursor,
    int? limit,
    NotificationCategory? category,
  }) async => NotificationPage(
    items: [_follow('one', isNew: hasNew || newness.seenCalls == 0)],
  );
}
