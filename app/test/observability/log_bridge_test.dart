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

      expect(reporter.errors.single, isA<StateError>());
      expect(reporter.messages, ['App log']);
      expect(reporter.contexts.first.feature, 'AuthController');
      expect(reporter.contexts.first.classification, 'log.severe');
      expect(reporter.contexts.last.severity, 'fatal');
    });

    test('keeps ordinary warnings local unless promoted', () async {
      final reporter = _RecordingReporter();
      final forwarder = LogForwarder(reporter);

      await forwarder.handle(LogRecord(Level.WARNING, 'warning', 'Profile'));
      expect(reporter.messages, isEmpty);

      await forwarder.handle(
        LogRecord(Level.WARNING, 'warning', 'Profile'),
        promotedWarning: true,
      );

      expect(reporter.messages, ['App log']);
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

      expect(reporter.errors.single, isA<StateError>());
      expect(reporter.stacks.single, same(stack));
      expect(reporter.contexts.single.feature, 'Profile');
      expect(reporter.contexts.single.classification, 'log.severe');
    });
  });
}

final class _RecordingReporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final messages = <String>[];
  final contexts = <ReportContext>[];

  @override
  bool get enabled => true;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {}

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    stacks.add(stackTrace);
    contexts.add(context);
    return '0123456789abcdef0123456789abcdef';
  }

  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {
    messages.add(message);
    contexts.add(context);
  }
}
