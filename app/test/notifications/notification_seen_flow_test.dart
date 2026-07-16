import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets(
    'IT-005 / REG-008 marks seen only after successful first-page renders',
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
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: NotificationsPage(),
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
      expect(find.text('Alice followed you'), findsOneWidget);
      expect(newness.seenCalls, 1);
      expect(newness.countCalls, 1);

      container.invalidate(notificationsProvider);
      await tester.pumpAndSettle();
      expect(find.text('No notifications yet.'), findsOneWidget);
      expect(newness.seenCalls, 2);
      expect(newness.countCalls, 2);
    },
  );
}

FollowNotification _follow(String rkey) =>
    CraftskyNotification.fromMap({
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
        })
        as FollowNotification;

final class _QueuedNotificationRepository implements NotificationRepository {
  _QueuedNotificationRepository(this.responses);

  final List<Future<NotificationPage>> responses;

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) =>
      responses.removeAt(0);
}

final class _RecordingNewnessRepository
    implements NotificationNewnessRepository {
  int seenCalls = 0;
  int countCalls = 0;

  @override
  Future<int> count() async {
    countCalls++;
    return 7;
  }

  @override
  Future<void> markSeen() async => seenCalls++;
}
