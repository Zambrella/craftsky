import 'package:flutter/foundation.dart';
import 'package:logging/logging.dart';

abstract interface class ErrorReporter {
  bool get enabled;

  Future<ReportResult> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  });

  Future<void> captureLog(LogRecord record, {required ReportContext context});

  void addBreadcrumb(SafeBreadcrumb breadcrumb);
}

final class NoopErrorReporter implements ErrorReporter {
  const NoopErrorReporter();

  @override
  bool get enabled => false;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {}

  @override
  Future<ReportResult> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    return const ReportResult.disabled();
  }

  @override
  Future<void> captureLog(
    LogRecord record, {
    required ReportContext context,
  }) async {}
}

final class GuardedErrorReporter implements ErrorReporter {
  const GuardedErrorReporter(this._delegate);

  final ErrorReporter _delegate;

  @override
  bool get enabled => _delegate.enabled;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {
    try {
      _delegate.addBreadcrumb(breadcrumb);
    } on Object catch (_) {
      // Reporting must never interrupt app code.
    }
  }

  @override
  Future<ReportResult> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    try {
      return await _delegate.captureException(
        error,
        context: context,
        stackTrace: stackTrace,
      );
    } on Object catch (_) {
      return const ReportResult.failed();
    }
  }

  @override
  Future<void> captureLog(
    LogRecord record, {
    required ReportContext context,
  }) async {
    try {
      await _delegate.captureLog(record, context: context);
    } on Object catch (_) {
      // Reporting must never interrupt app code.
    }
  }
}

final class ReportContext {
  const ReportContext({
    required this.feature,
    required this.operation,
    required this.classification,
    this.severity = 'error',
    this.safeDiagnostics = const {},
  });

  final String feature;
  final String operation;
  final String classification;
  final String severity;
  final Map<String, Object?> safeDiagnostics;
}

@immutable
final class SafeBreadcrumb {
  const SafeBreadcrumb({
    required this.category,
    required this.message,
    this.data = const {},
  });

  final String category;
  final String message;
  final Map<String, Object?> data;

  @override
  bool operator ==(Object other) {
    return other is SafeBreadcrumb &&
        category == other.category &&
        message == other.message &&
        _mapEquals(data, other.data);
  }

  @override
  int get hashCode => Object.hash(category, message, Object.hashAll(data.keys));
}

enum ReportStatus { captured, disabled, failed }

final class ReportResult {
  const ReportResult.captured({this.eventId}) : status = ReportStatus.captured;
  const ReportResult.disabled()
    : status = ReportStatus.disabled,
      eventId = null;
  const ReportResult.failed() : status = ReportStatus.failed, eventId = null;

  final ReportStatus status;
  final String? eventId;

  bool get hasEventId => eventId != null && eventId!.isNotEmpty;
}

bool _mapEquals(Map<String, Object?> a, Map<String, Object?> b) {
  if (a.length != b.length) return false;
  for (final entry in a.entries) {
    if (!b.containsKey(entry.key) || b[entry.key] != entry.value) {
      return false;
    }
  }
  return true;
}
