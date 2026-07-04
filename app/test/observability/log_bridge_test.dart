import 'package:craftsky_app/main.dart' as app_main;
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/log_forwarder.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  group('LogForwarder', () {
    test('forwards severe and shout records with safe context', () async {
      final reporter = _RecordingReporter();
      final forwarder = LogForwarder(reporter);

      await forwarder.handle(
        LogRecord(
          Level.SEVERE,
          'failed',
          'AuthController',
          StateError('boom'),
          StackTrace.current,
        ),
      );
      await forwarder.handle(LogRecord(Level.SHOUT, 'fatal', 'main'));

      expect(reporter.records.map((r) => r.level), [Level.SEVERE, Level.SHOUT]);
      expect(reporter.contexts.first.feature, 'AuthController');
      expect(reporter.contexts.first.classification, 'log.severe');
    });

    test('keeps ordinary warnings local unless promoted', () async {
      final reporter = _RecordingReporter();
      final forwarder = LogForwarder(reporter);

      await forwarder.handle(LogRecord(Level.WARNING, 'warning', 'Profile'));
      expect(reporter.records, isEmpty);

      await forwarder.handle(
        LogRecord(Level.WARNING, 'warning', 'Profile'),
        promotedWarning: true,
      );

      expect(reporter.records, hasLength(1));
      expect(reporter.contexts.single.classification, 'log.warning.promoted');
    });

    test('startup forwarding subscription bridges severe root logs', () async {
      final reporter = _RecordingReporter();
      final subscription = app_main.configureRootLogForwarding(
        reporter: reporter,
      );
      addTearDown(subscription.cancel);

      final stack = StackTrace.current;
      Logger('Profile').severe('profile failed', StateError('boom'), stack);
      await Future<void>.delayed(Duration.zero);

      expect(reporter.records, hasLength(1));
      expect(reporter.records.single.error, isA<StateError>());
      expect(reporter.records.single.stackTrace, same(stack));
      expect(reporter.contexts.single.feature, 'Profile');
      expect(reporter.contexts.single.classification, 'log.severe');
    });
  });
}

final class _RecordingReporter implements ErrorReporter {
  final records = <LogRecord>[];
  final contexts = <ReportContext>[];

  @override
  bool get enabled => true;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {}

  @override
  Future<ReportResult> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    return const ReportResult.captured();
  }

  @override
  Future<void> captureLog(
    LogRecord record, {
    required ReportContext context,
  }) async {
    records.add(record);
    contexts.add(context);
  }
}
