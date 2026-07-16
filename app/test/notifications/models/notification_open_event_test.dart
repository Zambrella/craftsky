import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const notificationId = '018f47a2-4b0e-7f39-a621-9f6f6c75e312';
  const routingId = 'subscription_Abc123';

  group('UT-002 provider data allowlist', () {
    test('accepts known and bounded future types', () {
      final known = NotificationOpenEvent.tryParseProviderData({
        'notificationId': notificationId,
        'type': 'quote',
        'accountSubscriptionId': routingId,
      });
      final future = NotificationOpenEvent.tryParseProviderData({
        'notificationId': notificationId,
        'type': 'projectInvite2',
        'accountSubscriptionId': routingId,
      });

      expect(known, isNotNull);
      expect(known!.category, NotificationCategory.quote);
      expect(future, isNotNull);
      expect(future!.category, NotificationCategory.unknown);
    });

    test('ignores destination-shaped and unknown extra keys', () {
      final event = NotificationOpenEvent.tryParseProviderData({
        'notificationId': notificationId,
        'type': 'like',
        'accountSubscriptionId': routingId,
        'did': 'did:plc:SENSITIVE_DID',
        'handle': 'SENSITIVE_HANDLE.example',
        'uri': 'at://SENSITIVE_URI',
        'destination': '/SENSITIVE_DESTINATION',
        'body': 'SENSITIVE_PAYLOAD_TEXT',
      });

      expect(event, isNotNull);
      expect(event!.category, NotificationCategory.like);
      expect(event.toString(), isNot(contains('SENSITIVE')));
      expect(event.notificationId.toString(), '<redacted-notification-id>');
      expect(
        event.accountSubscriptionId.toString(),
        '<redacted-account-subscription-id>',
      );
    });

    test('rejects missing or malformed required values', () {
      final valid = <String, Object?>{
        'notificationId': notificationId,
        'type': 'follow',
        'accountSubscriptionId': routingId,
      };

      for (final invalid in <Map<String, Object?>>[
        {...valid}..remove('notificationId'),
        {...valid, 'notificationId': 'not-a-uuid'},
        {...valid}..remove('type'),
        {...valid, 'type': 'not-valid!'},
        {...valid, 'type': 'a' * 65},
        {...valid}..remove('accountSubscriptionId'),
        {...valid, 'accountSubscriptionId': ''},
        {...valid, 'accountSubscriptionId': 'contains spaces'},
        {...valid, 'accountSubscriptionId': 'a' * 129},
      ]) {
        expect(
          NotificationOpenEvent.tryParseProviderData(invalid),
          isNull,
          reason: '$invalid',
        );
      }
    });
  });
}
