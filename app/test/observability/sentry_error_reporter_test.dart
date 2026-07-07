import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SentryErrorReporter', () {
    test('builds sanitized log attributes from report context', () {
      const context = ReportContext(
        feature: 'Profile',
        operation: 'load',
        classification: 'profile.failed',
        severity: 'warning',
        safeDiagnostics: {
          'endpointCategory': 'appview.profiles.detail',
          'rawUrl': 'https://example.test/profiles/alice',
        },
      );

      final attributes = SentryErrorReporter.attributesFor(context);

      expect(
        attributes.keys,
        containsAll([
          'feature',
          'operation',
          'classification',
          'severity',
          'endpointCategory',
        ]),
      );
      expect(attributes.keys, isNot(contains('rawUrl')));
    });
  });
}
