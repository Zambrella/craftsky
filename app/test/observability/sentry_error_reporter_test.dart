import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

void main() {
  test(
    'SIM-T04 final hook preserves SDK stacks and standard metadata',
    () async {
      final options = SentryOptions();
      configureDiagnosticOptions(options);
      final stack = SentryStackTrace(
        frames: [
          SentryStackFrame(
            function: 'loadPost',
            fileName: 'post.dart',
            lineNo: 12,
          ),
        ],
      );
      final event = SentryEvent(
        exceptions: [
          SentryException(
            type: 'StateError',
            value: 'PRIVATE_DRAFT',
            stackTrace: stack,
          ),
        ],
        contexts: Contexts()..['app'] = {'app_version': '1.2.3'},
        request: SentryRequest(url: 'https://private.test/?token=SECRET'),
        user: SentryUser(email: 'private@example.test'),
      );
      final safe = await options.beforeSend!(event, Hint());
      expect(identical(safe, event), isTrue);
      expect(identical(safe!.exceptions!.single.stackTrace, stack), isTrue);
      expect(safe.contexts['app'], {'app_version': '1.2.3'});
      expect(safe.exceptions!.single.value, 'Operation failed');
      expect(safe.request, isNull);
      expect(safe.user, isNull);
    },
  );
  group('SentryErrorReporter', () {
    test('builds sanitized log attributes from report context', () {
      const context = ReportContext(
        feature: 'Profile',
        operation: 'load',
        classification: 'profile.failed',
        severity: 'warning',
        safeDiagnostics: {
          'httpMethod': 'GET',
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
          'httpMethod',
        ]),
      );
      expect(attributes.keys, isNot(contains('rawUrl')));
    });
  });
}
