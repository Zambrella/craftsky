import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/observability_bootstrap.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:flutter_test/flutter_test.dart';

import '../test_support/diagnostic_evidence.dart';

void main() {
  group('ObservabilityBootstrap', () {
    test(
      'initializes adapter for enabled staging or production config',
      () async {
        final adapter = _FakeSentryBootstrapAdapter();
        final config = SentryConfig.fromValues(
          dsn: 'https://public@example.sentry.io/1',
          environment: 'staging',
          release: 'craftsky@1.0.0+1',
          dist: '1',
        );

        final reporter = await ObservabilityBootstrap.initialize(
          config: config,
          adapter: adapter,
        );

        expect(reporter.enabled, isTrue);
        expect(adapter.calls, [config]);
      },
    );

    test('returns no-op reporter and skips adapter without DSN', () async {
      final adapter = _FakeSentryBootstrapAdapter();

      final reporter = await ObservabilityBootstrap.initialize(
        config: SentryConfig.fromValues(environment: 'production'),
        adapter: adapter,
      );

      expect(reporter, isA<NoopErrorReporter>());
      expect(adapter.calls, isEmpty);
    });

    test(
      'IT-012 initialization failure is locally diagnosable and returns no-op',
      () async {
        final records = <String>[];
        final adapter = _FakeSentryBootstrapAdapter(throwOnInitialize: true);

        final reporter = await ObservabilityBootstrap.initialize(
          config: SentryConfig.fromValues(
            dsn: 'https://public@example.sentry.io/1',
            environment: 'production',
          ),
          adapter: adapter,
          fallbackSink: records.add,
        );

        expect(reporter, isA<NoopErrorReporter>());
        expect(records, hasLength(1));
        expect(records.single, contains('StateError'));
        expect(records.single, contains('initialize'));
        expect(records.single, isNot(contains('Sentry failed')));
        expect(records.single, isNot(contains('example.sentry.io')));
        writeDiagnosticEvidence('flutter-initialization', local: records);
      },
    );
  });
}

final class _FakeSentryBootstrapAdapter implements SentryBootstrapAdapter {
  _FakeSentryBootstrapAdapter({this.throwOnInitialize = false});

  final bool throwOnInitialize;
  final calls = <SentryConfig>[];

  @override
  Future<ErrorReporter> initialize(SentryConfig config) async {
    calls.add(config);
    if (throwOnInitialize) throw StateError('Sentry failed');
    return _EnabledReporter();
  }
}

final class _EnabledReporter implements ErrorReporter {
  @override
  bool get enabled => true;

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    return '0123456789abcdef0123456789abcdef';
  }
}
