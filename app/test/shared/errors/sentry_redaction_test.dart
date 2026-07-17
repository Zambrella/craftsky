import 'package:craftsky_app/shared/observability/sentry_sanitizer.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SentrySanitizer.sanitizeContext', () {
    test('keeps only allowlisted bounded diagnostics', () {
      final sanitized = SentrySanitizer.sanitizeContext({
        'appErrorKind': 'serviceUnavailable',
        'severity': 'error',
        'feature': 'feed',
        'classification': 'api.server_error',
        'appViewRequestId': 'req_123',
        'appViewError': 'internal_error',
        'httpStatus': 500,
        'endpointCategory': 'appview.feed.list',
        'authorization': 'Bearer secret',
        'cookie': 'session=secret',
        'requestBody': {'text': 'private project note'},
        'responseBody': 'database failed for did:plc:alice',
        'rawUrl': 'https://api.example.test/v1/feed?cursor=secret',
        'email': 'alice@example.test',
        'handle': 'alice.craftsky.social',
        'did': 'did:plc:alice',
        'deviceId': 'device-secret',
        'unknownPayload': {'x': 'y'},
      });

      expect(sanitized, {
        'appErrorKind': 'serviceUnavailable',
        'severity': 'error',
        'feature': 'feed',
        'classification': 'api.server_error',
        'appViewRequestId': 'req_123',
        'appViewError': 'internal_error',
        'httpStatus': 500,
        'endpointCategory': 'appview.feed.list',
      });
    });

    test('drops allowlisted string values that contain sensitive patterns', () {
      final sanitized = SentrySanitizer.sanitizeContext({
        'feature': 'did:plc:alice',
        'endpointCategory': 'https://api.example.test/v1/feed?cursor=secret',
        'classification': 'Bearer secret',
        'appViewRequestId': 'req_123',
      });

      expect(sanitized, {'appViewRequestId': 'req_123'});
    });

    test('REG-002 drops every notification and account sentinel field', () {
      const sentinels = {
        'firebaseToken': 'fcm-token-sensitive-value',
        'accountSubscriptionId': 'routing-sensitive-value',
        'notificationId': '018f47a2-4b0e-7f39-a621-9f6f6c75e312',
        'did': 'did:plc:sensitive-account',
        'handle': 'sensitive.craftsky.social',
        'atUri': 'at://did:plc:sensitive/social.craftsky.feed.post/secret',
        'focusUri':
            'at://did:plc:sensitive/social.craftsky.feed.post/secret-focus',
        'providerPayload': 'private provider copy',
        'rawPayload': '{"private":"raw notification payload"}',
        'providerError': 'provider failed for sensitive token and DID',
        'credential': 'credential-sensitive-value',
      };
      final sanitized = SentrySanitizer.sanitizeContext({
        'classification': 'notification.unavailable',
        'endpointCategory': 'appview.notifications.detail',
        ...sentinels,
      });

      expect(sanitized, {
        'classification': 'notification.unavailable',
        'endpointCategory': 'appview.notifications.detail',
      });
      final encoded = sanitized.toString();
      for (final sentinel in sentinels.values) {
        expect(encoded, isNot(contains(sentinel)));
      }
    });
  });
}
