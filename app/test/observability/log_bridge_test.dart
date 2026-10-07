import 'package:craftsky_app/main.dart' as app_main;
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/log_forwarder.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  test(
    'SIM-T02 severe supporting logs never implicitly capture an issue',
    () async {
      final reporter = _RecordingReporter();
      await LogForwarder(reporter).handle(
        LogRecord(
          Level.SEVERE,
          'Failed to load posts',
          'Post',
          StateError('PRIVATE_CAUSE'),
          StackTrace.current,
        ),
      );
      expect(reporter.logs, hasLength(1));
      expect(reporter.errors, isEmpty);
    },
  );

  test(
    'IT-015 startup keeps local release diagnostics while reporter changes',
    () async {
      final local = <String>[];
      ErrorReporter current = const NoopErrorReporter();
      final reporter = _RecordingReporter();
      final subscription = app_main.configureRootLogForwarding(
        reporter: current,
        currentReporter: () => current,
        platformSink: local.add,
      );
      addTearDown(subscription.cancel);
      Logger.root.level = Level.ALL;
      void emit() => Logger('Post').warning(
        const DiagnosticMessage(
          'Operation failed',
          context: ReportContext(
            feature: 'Post',
            operation: 'read',
            classification: 'post.read',
            safeDiagnostics: {
              'appViewRequestId': '40000000-0000-4000-8000-000000000001',
            },
          ),
        ),
        const FormatException('PRIVATE_STARTUP'),
        StackTrace.fromString(
          '#0 loadPost (package:craftsky_app/post.dart:12:3)',
        ),
      );
      emit();
      await Future<void>.delayed(Duration.zero);
      current = reporter;
      emit();
      await Future<void>.delayed(Duration.zero);
      current = const NoopErrorReporter();
      emit();
      await Future<void>.delayed(Duration.zero);
      expect(local, hasLength(3));
      expect(reporter.logs, hasLength(1));
      for (final line in local) {
        expect(line, contains('FormatException'));
        expect(line, contains('loadPost'));
        expect(line, contains('40000000-0000-4000-8000-000000000001'));
        expect(line, isNot(contains('PRIVATE_STARTUP')));
      }
    },
  );
  group('LogForwarder', () {
    test(
      'UT-011 expected API failures retain Logs without unexpected issues',
      () async {
        final reporter = _RecordingReporter();
        final forwarder = LogForwarder(reporter);
        for (final error in <Object>[
          const ApiCanceled(),
          const ApiUnauthorized(),
          const ApiNetworkError('offline'),
          const ApiBadRequest('not_found'),
          const ApiBadRequest('validation'),
        ]) {
          await forwarder.handle(
            LogRecord(
              Level.SEVERE,
              'operation failed',
              'Api',
              error,
              StackTrace.current,
            ),
          );
        }
        expect(reporter.logs.length, 5);
        expect(reporter.errors, isEmpty);
        await forwarder.handle(
          LogRecord(
            Level.SEVERE,
            'operation failed',
            'Api',
            const ApiServerError('http_500'),
            StackTrace.current,
          ),
        );
        await forwarder.handle(
          LogRecord(
            Level.SEVERE,
            'operation failed',
            'Api',
            const FormatException('opaque private'),
            StackTrace.current,
          ),
        );
        expect(reporter.errors, isEmpty);
      },
    );
    test(
      'UT-008 export breadcrumb and issue selection are independent',
      () async {
        final reporter = _RecordingReporter();
        final forwarder = LogForwarder(reporter);
        await forwarder.handle(
          LogRecord(
            Level.WARNING,
            'device-id write failed; using in-memory only',
            'Storage',
          ),
        );
        await forwarder.handle(
          LogRecord(
            Level.INFO,
            'bootstrap complete',
            'Bootstrap',
            null,
            null,
            null,
            const DiagnosticMessage(
              'bootstrap complete',
              context: ReportContext(
                feature: 'Bootstrap',
                operation: 'initialize',
                classification: 'lifecycle',
              ),
              significant: true,
            ),
          ),
        );
        await forwarder.handle(
          LogRecord(Level.INFO, 'private routine update', 'Private'),
        );
        await forwarder.handle(
          LogRecord(Level.FINE, 'bootstrap starting', 'Bootstrap'),
        );
        await forwarder.handle(
          LogRecord(
            Level.SEVERE,
            'pending handoff storage failed',
            'Storage',
            StateError('opaque private failure'),
            StackTrace.current,
          ),
        );
        expect(reporter.logs.map((log) => log.$1), [
          'device-id write failed; using in-memory only',
          'bootstrap complete',
          'pending handoff storage failed',
        ]);
        expect(reporter.errors, isEmpty);
        expect(
          reporter.breadcrumbs.map((crumb) => crumb.message),
          contains('bootstrap complete'),
        );
        await LogForwarder(
          reporter,
          logsEnabled: false,
        ).handle(LogRecord(Level.WARNING, 'bootstrap starting', 'Bootstrap'));
        expect(reporter.logs.length, 3);
        await LogForwarder(
          reporter,
          debugLogsEnabled: true,
        ).handle(LogRecord(Level.FINE, 'bootstrap starting', 'Bootstrap'));
        expect(reporter.logs.last.$1, 'bootstrap starting');
        expect(reporter.logs.last.$2.severity, 'debug');
      },
    );
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

      expect(reporter.errors, isEmpty);
      expect(reporter.logs.length, 2);
      expect(
        reporter.logs.map((log) => log.$1),
        equals(['failed', 'fatal']),
      );
      expect(reporter.logs.first.$2.feature, 'AuthController');
      expect(reporter.logs.first.$2.classification, 'log.severe');
      expect(reporter.logs.last.$2.severity, 'fatal');
    });

    test(
      'exports ordinary warnings without promoting them into issues',
      () async {
        final reporter = _RecordingReporter();
        final forwarder = LogForwarder(reporter);

        await forwarder.handle(LogRecord(Level.WARNING, 'warning', 'Profile'));
        expect(reporter.logs.length, 1);
        expect(reporter.errors, isEmpty);

        await forwarder.handle(
          LogRecord(Level.WARNING, 'warning', 'Profile'),
        );

        expect(reporter.logs.length, 2);
        expect(reporter.errors, isEmpty);
        expect(reporter.logs.last.$2.classification, 'log.warning');
      },
    );

    test('startup forwarding subscription bridges severe root logs', () async {
      final reporter = _RecordingReporter();
      final subscription = app_main.configureRootLogForwarding(
        reporter: reporter,
      );
      addTearDown(subscription.cancel);

      final stack = StackTrace.current;
      Logger('Profile').severe('profile failed', StateError('boom'), stack);
      await Future<void>.delayed(Duration.zero);

      expect(reporter.errors, isEmpty);
      expect(reporter.logs.single.$2.stackTrace, same(stack));
      expect(reporter.logs.single.$2.feature, 'Profile');
      expect(reporter.logs.single.$2.classification, 'log.severe');
    });
  });
}

final class _RecordingReporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final messages = <String>[];
  final contexts = <ReportContext>[];
  final logs = <(String, ReportContext)>[];
  final breadcrumbs = <SafeBreadcrumb>[];

  @override
  bool get enabled => true;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {
    breadcrumbs.add(breadcrumb);
  }

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
  Future<void> emitLog(String message, {required ReportContext context}) async {
    logs.add((message, context));
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
