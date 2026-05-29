import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('NotificationsPage renders its title', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          notificationRepositoryProvider.overrideWithValue(
            _FakeNotificationRepository(NotificationPage(items: [])),
          ),
        ],
        child: const MaterialApp(home: NotificationsPage()),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Notifications'), findsWidgets);
  });

  testWidgets('renders empty state and mixed rows', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          notificationRepositoryProvider.overrideWithValue(
            _FakeNotificationRepository(
              NotificationPage(items: [_follow('follow1')]),
            ),
          ),
        ],
        child: const MaterialApp(home: NotificationsPage()),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Alice followed you'), findsOneWidget);
  });
}

FollowNotification _follow(String rkey) =>
    CraftskyNotification.fromMap({
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

class _FakeNotificationRepository implements NotificationRepository {
  const _FakeNotificationRepository(this.page);

  final NotificationPage page;

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) async => page;
}
