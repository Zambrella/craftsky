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

    final context = ReportContext(
      feature: record.loggerName,
      operation: 'log',
      classification: promotedWarning
          ? 'log.warning.promoted'
          : 'log.${record.level.name.toLowerCase()}',
      severity: record.level.value >= Level.SHOUT.value
          ? 'fatal'
          : record.level.value >= Level.SEVERE.value
          ? 'error'
          : 'warning',
    );

    if (record.level.value >= Level.SEVERE.value &&
        (record.error != null || record.stackTrace != null)) {
      await _reporter.captureException(
        record.error ?? _LogRecordException(record.message),
        stackTrace: record.stackTrace,
        context: context,
      );
      return;
    }

    await _reporter.captureMessage('App log', context: context);
  }
}

final class _LogRecordException implements Exception {
  const _LogRecordException(this.message);

  final String message;

  @override
  String toString() => 'LogRecordException: $message';
}
