import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/firebase_notification_service.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const chat = RemoteMessage(
    data: {
      'payloadVersion': '1',
      'type': 'chat',
      'accountSubscriptionId': 'routing-account-one',
      'openReference': '00000000-0000-4000-8000-000000000123',
    },
    notification: RemoteNotification(
      title: 'Alice',
      body: 'sent you a message',
    ),
  );
  test('chat delivery reaches foreground refresh with the server payload', () {
    final event = FirebaseNotificationService.foregroundEventFromMessage(chat);
    expect(event?.title, 'Alice');
    expect(event?.openAttempt.facts, isA<ChatNotificationFacts>());
  });
  for (final source in [
    NotificationOpenSource.initialOpen,
    NotificationOpenSource.backgroundOpen,
  ]) {
    test('chat OS open preserves private routing for $source without copy', () {
      final attempt = FirebaseNotificationService.openAttemptFromMessage(
        const RemoteMessage(
          data: {
            'payloadVersion': '1',
            'type': 'chat',
            'accountSubscriptionId': 'routing-account-one',
            'openReference': '00000000-0000-4000-8000-000000000123',
          },
        ),
        source,
      );
      expect(attempt?.source, source);
      expect(
        (attempt!.facts as ChatNotificationFacts).openReference.value,
        '00000000-0000-4000-8000-000000000123',
      );
    });
  }
  test(
    'UT-PUSH-013 notification foreground duplicates emit normally',
    () async {
      const message = RemoteMessage(
        data: {
          'payloadVersion': '1',
          'type': 'like',
          'accountSubscriptionId': 'routing-account-one',
          'notificationId': '00000000-0000-4000-8000-000000000020',
          'subjectUri': 'at://did:plc:viewer/social.craftsky.feed.post/comment',
          'rootUri': 'at://did:plc:viewer/social.craftsky.feed.post/root',
        },
        notification: RemoteNotification(
          title: 'Alice',
          body: 'liked your post',
        ),
      );

      final first = FirebaseNotificationService.foregroundEventFromMessage(
        message,
      );
      final duplicate = FirebaseNotificationService.foregroundEventFromMessage(
        message,
      );

      expect(first?.title, 'Alice');
      expect(first?.body, 'liked your post');
      expect(duplicate?.title, 'Alice');
      expect(duplicate?.body, 'liked your post');
    },
  );

  test(
    'UT-PUSH-014 APNs alert copy enters the same validated envelope',
    () async {
      const message = RemoteMessage(
        data: {
          'payloadVersion': '1',
          'type': 'everythingElse',
          'accountSubscriptionId': 'routing-account-one',
          'notificationId': '00000000-0000-4000-8000-000000000021',
        },
        notification: RemoteNotification(
          title: 'CraftSky',
          body: 'You have a new notification',
        ),
      );

      final event = FirebaseNotificationService.foregroundEventFromMessage(
        message,
      );

      expect(event?.title, 'CraftSky');
      expect(event?.body, 'You have a new notification');
    },
  );
}
