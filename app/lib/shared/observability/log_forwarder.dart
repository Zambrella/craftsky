import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:logging/logging.dart';

final class LogForwarder {
  const LogForwarder(this._reporter);

  final ErrorReporter _reporter;

  Future<void> handle(
    LogRecord record, {
    bool promotedWarning = false,
  }) async {
    final shouldForward =
        record.level.value >= Level.SEVERE.value ||
        (record.level == Level.WARNING && promotedWarning);
    if (!shouldForward) return;

    await _reporter.captureLog(
      record,
      context: ReportContext(
        feature: record.loggerName,
        operation: 'log',
        classification: promotedWarning
            ? 'log.warning.promoted'
            : 'log.${record.level.name.toLowerCase()}',
        severity: record.level.value >= Level.SEVERE.value
            ? 'error'
            : 'warning',
      ),
    );
  }
}
