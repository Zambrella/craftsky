import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_sanitizer.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SentrySanitizer.sanitizeBreadcrumb', () {
    test('keeps coarse navigation and recovery-action data', () {
      final breadcrumb = SentrySanitizer.sanitizeBreadcrumb(
        const SafeBreadcrumb(
          category: 'navigation',
          message: 'route changed',
          data: {
            'routeName': 'feed',
            'feature': 'feed',
            'action': 'retry',
            'rawPath': '/profile/alice.craftsky.social?cursor=secret',
            'handle': 'alice.craftsky.social',
            'searchQuery': 'private knitting note',
            'cursor': 'cursor-secret',
          },
        ),
      );

      expect(
        breadcrumb,
        const SafeBreadcrumb(
          category: 'navigation',
          message: 'route changed',
          data: {
            'routeName': 'feed',
            'feature': 'feed',
            'action': 'retry',
          },
        ),
      );
    });

    test('drops breadcrumbs with unsafe categories or messages', () {
      expect(
        SentrySanitizer.sanitizeBreadcrumb(
          const SafeBreadcrumb(
            category: 'http',
            message: 'GET https://api.example.test/v1/feed?cursor=secret',
          ),
        ),
        isNull,
      );
    });
  });
}
