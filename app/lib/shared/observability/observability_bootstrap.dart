import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/platform_log.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';

// The adapter keeps SDK initialization testable without importing Sentry in
// bootstrap tests.
// ignore: one_member_abstracts
abstract interface class SentryBootstrapAdapter {
  Future<ErrorReporter> initialize(SentryConfig config);
}

final class ObservabilityBootstrap {
  const ObservabilityBootstrap._();

  static Future<ErrorReporter> initialize({
    required SentryConfig config,
    required SentryBootstrapAdapter adapter,
    PlatformLogSink? fallbackSink,
  }) async {
    if (!config.enabled) return const NoopErrorReporter();

    try {
      return GuardedErrorReporter(
        await adapter.initialize(config),
        fallbackSink: fallbackSink,
      );
    } on Object catch (error, stack) {
      emitLocalTelemetryFailure(
        error,
        const ReportContext(
          feature: 'Telemetry',
          operation: 'initialize',
          classification: 'telemetry.failure',
        ),
        stack,
        sink: fallbackSink,
      );
      return const NoopErrorReporter();
    }
  }
}
