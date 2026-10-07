import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  test(
    'IT-012 root console output remains independent of reporting',
    () async {
      final records = <String>[];
      final subscription = configureRootLogForwarding(
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
      expect(records.length, equals(1));
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
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    throw StateError('capture failed');
  }
}
