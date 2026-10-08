import 'dart:convert';
import 'package:craftsky_app/shared/observability/diagnostic_details.dart';
import 'package:craftsky_app/shared/observability/diagnostic_text.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/platform_log.dart';
import 'package:craftsky_app/shared/observability/sentry_sanitizer.dart';
import 'package:flutter/foundation.dart';
import 'package:logging/logging.dart';

/// Static developer message plus a few operational fields; never a payload.
final class DiagnosticMessage {
  const DiagnosticMessage(
    this.message, {
    required this.context,
    this.significant = false,
  });
  final String message;
  final ReportContext context;
  final bool significant;
  @override
  String toString() => boundDiagnosticText(message, 256);
}

/// One brief console record. Sentry retains the richer exception and stack.
final class DiagnosticEmitter {
  DiagnosticEmitter({
    PlatformLogSink? platformSink,
    this.debugLogs = const bool.fromEnvironment(
      'CRAFTSKY_DEBUG_LOGS',
    ),
    this.environment = const String.fromEnvironment(
      'SENTRY_ENVIRONMENT',
      defaultValue: 'development',
    ),
    this.release = const String.fromEnvironment('SENTRY_RELEASE'),
  }) : _platform = platformSink ?? writePlatformDiagnostic;
  final PlatformLogSink _platform;
  final bool debugLogs;
  final String environment;
  final String release;
  void emitLocal(LogRecord record) {
    final minimumLevel = debugLogs && kDebugMode ? Level.FINE : Level.INFO;
    if (record.level < minimumLevel) return;
    final selected = selectDiagnosticRecord(record);
    selected['environment'] = _technical(environment);
    if (RegExp(r'^[A-Za-z0-9_.@+\-]{1,160}$').hasMatch(release)) {
      selected['release'] = release;
    }
    try {
      _platform(jsonEncode(selected));
    } on Object {
      /* Logging cannot change the operation. */
    }
  }
}

// Fixed brief schema: <=12 operational scalars, <=6 references, <=3 causes,
// <=2 frames and a 256-byte message. No raw bodies, nested arbitrary values or
// byte-cut JSON. These independent small limits fit one 8 KiB console record.
const maxPlatformDiagnosticBytes = 8192;
Map<String, Object?> selectDiagnosticRecord(LogRecord record) {
  final context = record.object is DiagnosticMessage
      ? (record.object! as DiagnosticMessage).context
      : null;
  final fields = <String, Object?>{
    'severity': _technical(record.level.name),
    'feature': _technical(context?.feature ?? record.loggerName),
    'operation': _technical(context?.operation ?? 'log'),
    'message': boundDiagnosticText(record.message, 256),
    ...SentrySanitizer.sanitizeContext(context?.safeDiagnostics ?? const {})
        .entries
        .where((e) => e.value is String || e.value is num || e.value is bool)
        .where(
          (e) => !const {
            'incomingPath',
            'routePattern',
            'appViewMessage',
          }.contains(e.key),
        )
        .take(12)
        .fold<Map<String, Object?>>({}, (map, e) => map..[e.key] = e.value),
  };
  if (record.error != null) {
    final causes = selectedCauses(record.error!)
        .take(3)
        .map(
          (cause) => <String, Object?>{
            for (final e in cause.entries)
              e.key: e.value is String
                  ? boundDiagnosticText(e.value! as String, 128)
                  : e.value,
          },
        )
        .toList();
    fields['cause'] = {
      ...causes.first,
      if (causes.length > 1) 'causes': causes.skip(1).toList(),
    };
  }
  if (record.stackTrace != null) {
    fields['stack'] = selectedStack(record.stackTrace!).take(2).toList();
  }
  final workflow = context == null
      ? null
      : selectedWorkflow(context, failure: record.error != null);
  if (workflow != null) {
    fields['diagnostic'] = {
      for (final e
          in workflow.selectedFields.entries
              .where((e) => e.key != 'publicExcerpt')
              .take(6))
        e.key: e.value is String
            ? (RegExp(
                    r'^[A-Za-z0-9_.:/@!~%+\-]{1,160}$',
                  ).hasMatch(e.value! as String)
                  ? sanitizeKnownDiagnosticText(e.value! as String)
                  : '[OMITTED: console identifier]')
            : e.value is num || e.value is bool
            ? e.value
            : '[OMITTED]',
    };
  }
  return fields;
}

String _technical(String value) =>
    RegExp(r'^[A-Za-z0-9_.]{1,160}$').hasMatch(value) ? value : '[OMITTED]';

// Selected source context exists only while a synchronous provider transition
// publishes the same failure. This is neither stored history nor deduplication;
// reusing an error in another occurrence receives a fresh scope.
final _activeFailureContexts = Map<Object, ReportContext>.identity();
ReportContext? failureDiagnosticContext(Object error) =>
    _activeFailureContexts[error];
void withFailureDiagnosticContext(
  Object error,
  ReportContext context,
  void Function() publish,
) {
  if (_activeFailureContexts.length >= 64) {
    publish();
    return;
  }
  final previous = _activeFailureContexts[error];
  _activeFailureContexts[error] = context;
  try {
    publish();
  } finally {
    if (previous == null) {
      _activeFailureContexts.remove(error);
    } else {
      _activeFailureContexts[error] = previous;
    }
  }
}
