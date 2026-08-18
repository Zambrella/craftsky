import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_envelope.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const notificationId = '00000000-0000-4000-8000-000000000321';
  const data = <String, Object?>{
    'payloadVersion': '1',
    'type': 'like',
    'accountSubscriptionId': '30000000-0000-4000-8000-000000000321',
    'notificationId': notificationId,
    'displayTitle': 'Alice',
    'displayBody': 'liked your post',
    'subjectUri': 'at://did:plc:viewer/social.craftsky.feed.post/comment',
    'rootUri': 'at://did:plc:viewer/social.craftsky.feed.post/root',
  };

  test(
    'UT-PUSH-001 validates and round-trips the Android display envelope',
    () {
      final envelope = NotificationDeliveryEnvelope.tryParse(data);

      expect(envelope, isNotNull);
      expect(envelope!.notificationId, notificationId);
      expect(envelope.androidTag, notificationId);
      expect(envelope.title, 'Alice');
      expect(envelope.body, 'liked your post');
      expect(envelope.openAttempt.facts, isA<ValidNotificationFacts>());
      expect(envelope.toString(), isNot(contains(notificationId)));

      final roundTrip = NotificationDeliveryEnvelope.tryParseLocalPayload(
        envelope.localOpenPayload,
      );
      expect(roundTrip?.notificationId, notificationId);
      expect(roundTrip?.openAttempt.facts, isA<ValidNotificationFacts>());
    },
  );

  test('UT-PUSH-002 rejects malformed identity, copy, and routing facts', () {
    for (final invalid in <Map<String, Object?>>[
      {...data, 'notificationId': 'not-a-uuid'},
      {...data, 'displayTitle': ''},
      {...data, 'displayBody': 'unsafe\ncopy'},
      {...data, 'displayTitle': List.filled(257, 'a').join()},
      {...data, 'accountSubscriptionId': 'contains spaces'},
      {...data, 'subjectUri': 'https://example.com/not-an-at-uri'},
      {...data, 'payloadVersion': '2'},
      {...data, 'type': 'futureCategory'},
    ]) {
      expect(
        NotificationDeliveryEnvelope.tryParse(invalid),
        isNull,
        reason: 'accepted $invalid',
      );
    }
    expect(
      NotificationDeliveryEnvelope.tryParseLocalPayload('{broken'),
      isNull,
    );
  });
}
