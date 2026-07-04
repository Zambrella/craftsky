import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  group('NoopErrorReporter', () {
    test('is disabled and returns a disabled capture result', () async {
      const reporter = NoopErrorReporter();

      final result = await reporter.captureException(
        StateError('boom'),
        stackTrace: StackTrace.current,
        context: const ReportContext(
          feature: 'startup',
          operation: 'initialize',
          classification: 'initialization.failed',
        ),
      );

      expect(reporter.enabled, isFalse);
      expect(result.status, ReportStatus.disabled);
      expect(result.eventId, isNull);
      expect(result.hasEventId, isFalse);
    });

    test('does not throw for logs or breadcrumbs', () async {
      const reporter = NoopErrorReporter();

      await reporter.captureLog(
        LogRecord(Level.SEVERE, 'failed', 'test'),
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

      final result = await reporter.captureException(
        StateError('boom'),
        stackTrace: StackTrace.current,
        context: const ReportContext(
          feature: 'startup',
          operation: 'initialize',
          classification: 'initialization.failed',
        ),
      );

      expect(result.status, ReportStatus.failed);
      expect(result.eventId, isNull);
    });

    test('swallows log and breadcrumb reporter exceptions', () async {
      final reporter = GuardedErrorReporter(_ThrowingReporter());

      await reporter.captureLog(
        LogRecord(Level.SEVERE, 'failed', 'test'),
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

  test('ReportResult recognizes non-empty event IDs', () {
    expect(
      const ReportResult.captured(
        eventId: '0123456789abcdef0123456789abcdef',
      ).hasEventId,
      isTrue,
    );
    expect(const ReportResult.captured(eventId: '').hasEventId, isFalse);
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
  Future<ReportResult> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    throw StateError('capture failed');
  }

  @override
  Future<void> captureLog(
    LogRecord record, {
    required ReportContext context,
  }) async {
    throw StateError('log failed');
  }
}
