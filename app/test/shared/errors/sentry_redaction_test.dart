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
  });
}
