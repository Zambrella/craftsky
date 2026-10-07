import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  test(
    'IT-012 root forwarding survives a throwing injected reporter',
    () async {
      final records = <String>[];
      final subscription = configureRootLogForwarding(
        reporter: _ThrowingReporter(),
        platformSink: records.add,
      );
      addTearDown(subscription.cancel);
      Logger.root.level = Level.ALL;
      Logger('Post').severe(
        const DiagnosticMessage(
          'Operation failed',
          context: ReportContext(
            feature: 'Post',
            operation: 'read',
            classification: 'post.read',
          ),
        ),
        const FormatException('PRIVATE_ROOT'),
      );
      await Future<void>.delayed(Duration.zero);
      expect(records.length, greaterThanOrEqualTo(2));
      expect(records.join(), contains('FormatException'));
      expect(records.join(), isNot(contains('PRIVATE_ROOT')));
    },
  );
  test(
    'IT-012 throwing reporter state preserves caller and local diagnostic',
    () {
      final records = <String>[];
      final reporter = GuardedErrorReporter(
        _ThrowingReporter(throwOnEnabled: true),
        fallbackSink: records.add,
      );
      expect(reporter.enabled, isFalse);
      expect(records, hasLength(1));
      expect(records.single, contains('StateError'));
      expect(records.single, isNot(contains('PRIVATE_STATE')));
    },
  );
  test(
    'IT-012 throwing reporter preserves original cause in local fallback',
    () async {
      final records = <String>[];
      final reporter = GuardedErrorReporter(
        _ThrowingReporter(),
        fallbackSink: records.add,
      );
      const cause = FormatException('PRIVATE_SOURCE', 'PRIVATE_BODY');
      final result = await reporter.captureException(
        cause,
        stackTrace: StackTrace.fromString(
          '#0 decodePublishedPost (package:craftsky_app/post.dart:12:3)',
        ),
        context: const ReportContext(
          feature: 'Post',
          operation: 'read',
          classification: 'post.read',
          safeDiagnostics: {
            'appViewRequestId': '40000000-0000-4000-8000-000000000001',
          },
        ),
      );
      expect(result, isNull);
      expect(records, hasLength(1));
      expect(records.single, contains('FormatException'));
      expect(records.single, contains('decodePublishedPost'));
      expect(records.single, contains('40000000-0000-4000-8000-000000000001'));
      expect(records.single, contains('Telemetry reporting failed'));
      for (final private in [
        'PRIVATE_SOURCE',
        'PRIVATE_BODY',
        'capture failed',
      ]) {
        expect(records.single, isNot(contains(private)));
      }
    },
  );

  test(
    'IT-012 log and breadcrumb failures use bounded direct fallback',
    () async {
      final records = <String>[];
      final reporter = GuardedErrorReporter(
        _ThrowingReporter(),
        fallbackSink: records.add,
      );
      final context = ReportContext(
        feature: 'Post',
        operation: 'read',
        classification: 'post.read',
        cause: const FormatException('PRIVATE_ORIGINAL'),
        stackTrace: StackTrace.fromString(
          '#0 decodePost (package:craftsky_app/post.dart:12:3)',
        ),
      );
      await reporter.emitLog('Operation failed', context: context);
      await reporter.captureMessage('Operation failed', context: context);
      reporter.addBreadcrumb(
        const SafeBreadcrumb(
          category: 'lifecycle',
          message: 'PRIVATE_BREADCRUMB',
          data: {'private': 'PRIVATE_BODY'},
        ),
      );
      expect(records, hasLength(3));
      expect(records.take(2), everyElement(contains('FormatException')));
      expect(records.take(2), everyElement(contains('decodePost')));
      expect(records.last, contains('StateError'));
      expect(records.join(), isNot(contains('PRIVATE_')));
    },
  );

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
  _ThrowingReporter({this.throwOnEnabled = false});
  final bool throwOnEnabled;
  @override
  bool get enabled {
    if (throwOnEnabled) throw StateError('PRIVATE_STATE');
    return true;
  }

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
  Future<void> emitLog(String message, {required ReportContext context}) async {
    throw StateError('PRIVATE_LOG_FAILURE');
  }

  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {
    throw StateError('log failed');
  }
}
