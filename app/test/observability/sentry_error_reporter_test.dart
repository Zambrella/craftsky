import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  group('SentryErrorReporter', () {
    test('captures severe log errors with their stack trace', () async {
      final sink = _RecordingSentryLogSink();
      final reporter = SentryErrorReporter(logSink: sink);
      final error = StateError('failed');
      final stack = StackTrace.current;
      const context = ReportContext(
        feature: 'Profile',
        operation: 'log',
        classification: 'log.severe',
      );

      await reporter.captureLog(
        LogRecord(Level.SEVERE, 'profile failed', 'Profile', error, stack),
        context: context,
      );

      expect(sink.exceptions, [error]);
      expect(sink.stacks, [stack]);
      expect(sink.contexts, [context]);
      expect(sink.messageLevels, isEmpty);
    });

    test('keeps warning log capture on the Sentry log API', () async {
      final sink = _RecordingSentryLogSink();
      final reporter = SentryErrorReporter(logSink: sink);
      const context = ReportContext(
        feature: 'Profile',
        operation: 'log',
        classification: 'log.warning.promoted',
      );

      await reporter.captureLog(
        LogRecord(Level.WARNING, 'retry warning', 'Profile'),
        context: context,
      );

      expect(sink.exceptions, isEmpty);
      expect(sink.messageLevels, [Level.WARNING]);
      expect(sink.contexts, [context]);
    });
  });
}

final class _RecordingSentryLogSink implements SentryLogSink {
  final exceptions = <Object>[];
  final stacks = <StackTrace?>[];
  final contexts = <ReportContext>[];
  final messageLevels = <Level>[];

  @override
  Future<void> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    exceptions.add(error);
    stacks.add(stackTrace);
    contexts.add(context);
  }

  @override
  Future<void> captureMessage(
    Level level, {
    required ReportContext context,
  }) async {
    messageLevels.add(level);
    contexts.add(context);
  }
}
