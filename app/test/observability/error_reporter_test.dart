import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('NoopErrorReporter', () {
    test('is disabled and returns a disabled capture result', () async {
      const reporter = NoopErrorReporter();

      final eventId = await reporter.captureException(
        StateError('boom'),
        stackTrace: StackTrace.current,
        context: const ReportContext(
          feature: 'startup',
          operation: 'initialize',
          classification: 'initialization.failed',
        ),
      );

      expect(reporter.enabled, isFalse);
      expect(eventId, isNull);
    });

    test('does not throw for logs or breadcrumbs', () async {
      const reporter = NoopErrorReporter();

      await reporter.captureMessage(
        'App log',
        context: const ReportContext(
          feature: 'test',
          operation: 'log',
          classification: 'log.severe',
        ),
      );

      reporter.addBreadcrumb(
        const SafeBreadcrumb(
          category: 'ui.action',
          message: 'retry',
          data: {'feature': 'startup'},
        ),
      );
    });
  });

  group('GuardedErrorReporter', () {
    test('turns reporter exceptions into failed capture results', () async {
      final reporter = GuardedErrorReporter(_ThrowingReporter());

      final eventId = await reporter.captureException(
        StateError('boom'),
        stackTrace: StackTrace.current,
        context: const ReportContext(
          feature: 'startup',
          operation: 'initialize',
          classification: 'initialization.failed',
        ),
      );

      expect(eventId, isNull);
    });

    test('swallows log and breadcrumb reporter exceptions', () async {
      final reporter = GuardedErrorReporter(_ThrowingReporter());

      await reporter.captureMessage(
        'App log',
        context: const ReportContext(
          feature: 'test',
          operation: 'log',
          classification: 'log.severe',
        ),
      );
      reporter.addBreadcrumb(
        const SafeBreadcrumb(category: 'ui.action', message: 'retry'),
      );
    });
  });
}

final class _ThrowingReporter implements ErrorReporter {
  @override
  bool get enabled => true;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {
    throw StateError('breadcrumb failed');
  }

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    throw StateError('capture failed');
  }

  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {
    throw StateError('log failed');
  }
}
