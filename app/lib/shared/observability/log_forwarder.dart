import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_text.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:logging/logging.dart';

final class LogForwarder {
  const LogForwarder(
    this._reporter, {
    this.logsEnabled = true,
    this.debugLogsEnabled = false,
  });
  final ErrorReporter _reporter;
  final bool logsEnabled;
  final bool debugLogsEnabled;

  Future<void> handle(LogRecord record) async {
    final diagnostic = record.object is DiagnosticMessage
        ? record.object! as DiagnosticMessage
        : null;
    final selected = selectDiagnosticRecord(record);
    final message = boundDiagnosticText(record.message, 512);
    final source = diagnostic?.context;
    final context = ReportContext(
      feature: selected['feature']! as String,
      operation: source?.operation ?? 'log',
      classification:
          source?.classification ?? 'log.${record.level.name.toLowerCase()}',
      severity: record.level >= Level.SHOUT
          ? 'fatal'
          : record.level >= Level.SEVERE
          ? 'error'
          : record.level >= Level.WARNING
          ? 'warning'
          : record.level >= Level.INFO
          ? 'info'
          : 'debug',
      safeDiagnostics: {
        ...?source?.safeDiagnostics,
      },
      workflow: source?.workflow,
      cause: record.error,
      stackTrace: record.stackTrace,
      outcome: source?.outcome ?? DiagnosticOutcome.automatic,
    );
    final tasks = <Future<void>>[];
    final selectedInfo =
        record.level == Level.INFO && (diagnostic?.significant ?? false);
    if (logsEnabled &&
        (record.level >= Level.WARNING ||
            selectedInfo ||
            (debugLogsEnabled && record.level < Level.INFO))) {
      tasks.add(_reporter.emitLog(message, context: context));
    }
    if (selectedInfo) {
      _reporter.addBreadcrumb(
        SafeBreadcrumb(
          category: 'lifecycle',
          message: message,
          data: {'feature': context.feature},
        ),
      );
    }
    await Future.wait(tasks);
  }
}
