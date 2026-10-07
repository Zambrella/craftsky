import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_text.dart';
import 'package:craftsky_app/shared/observability/platform_log.dart';
import 'package:flutter/foundation.dart';
import 'package:logging/logging.dart';

abstract interface class ErrorReporter {
  bool get enabled;

  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  });

  Future<void> emitLog(String message, {required ReportContext context});

  Future<void> captureMessage(String message, {required ReportContext context});

  void addBreadcrumb(SafeBreadcrumb breadcrumb);
}

final class NoopErrorReporter implements ErrorReporter {
  const NoopErrorReporter();

  @override
  bool get enabled => false;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {}

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    return null;
  }

  @override
  Future<void> emitLog(
    String message, {
    required ReportContext context,
  }) async {}

  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {}
}

final class GuardedErrorReporter implements ErrorReporter {
  const GuardedErrorReporter(this._delegate, {this._fallbackSink});
  final PlatformLogSink? _fallbackSink;

  final ErrorReporter _delegate;

  @override
  bool get enabled {
    try {
      return _delegate.enabled;
    } on Object catch (error, stack) {
      _fallback(
        error,
        const ReportContext(
          feature: 'Telemetry',
          operation: 'state',
          classification: 'telemetry.failure',
        ),
        stack,
      );
      return false;
    }
  }

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {
    try {
      _delegate.addBreadcrumb(breadcrumb);
    } on Object catch (error, stack) {
      _fallback(
        error,
        const ReportContext(
          feature: 'Telemetry',
          operation: 'breadcrumb',
          classification: 'telemetry.failure',
        ),
        stack,
      );
    }
  }

  @override
  Future<String?> captureException(
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
      _fallback(error, context, stackTrace);
      return null;
    }
  }

  @override
  Future<void> emitLog(String message, {required ReportContext context}) async {
    try {
      await _delegate.emitLog(message, context: context);
    } on Object catch (error, stack) {
      _fallbackLog(error, stack, context);
    }
  }

  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {
    try {
      await _delegate.captureMessage(message, context: context);
    } on Object catch (error, stack) {
      _fallbackLog(error, stack, context);
    }
  }

  void _fallbackLog(Object error, StackTrace stack, ReportContext context) {
    // A failed success-log export must not create private activity history.
    final selected = ReportContext(
      feature: context.feature,
      operation: context.operation,
      classification: context.classification,
      safeDiagnostics: context.safeDiagnostics,
      workflow: context.cause == null ? null : context.workflow,
    );
    _fallback(context.cause ?? error, selected, context.stackTrace ?? stack);
  }

  void _fallback(Object cause, ReportContext context, StackTrace? stack) {
    emitLocalTelemetryFailure(cause, context, stack, sink: _fallbackSink);
  }
}

void emitLocalTelemetryFailure(
  Object cause,
  ReportContext context,
  StackTrace? stack, {
  PlatformLogSink? sink,
}) {
  DiagnosticEmitter(platformSink: sink).emitLocal(
    LogRecord(
      Level.WARNING,
      'Telemetry reporting failed',
      'Telemetry',
      cause,
      stack,
      null,
      DiagnosticMessage(
        'Telemetry reporting failed',
        context: context,
      ),
    ),
  );
}

enum DiagnosticOutcome { automatic, expected, retry, terminal }

final class ReportContext {
  const ReportContext({
    required this.feature,
    required this.operation,
    required this.classification,
    this.severity = 'error',
    this.outcome = DiagnosticOutcome.automatic,
    this.safeDiagnostics = const {},
    this.workflow,
    this.cause,
    this.stackTrace,
  });

  final String feature;
  final String operation;
  final String classification;
  final String severity;
  final DiagnosticOutcome outcome;
  final Map<String, Object?> safeDiagnostics;
  final DiagnosticWorkflow? workflow;
  final Object? cause;
  final StackTrace? stackTrace;
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

bool _mapEquals(Map<String, Object?> a, Map<String, Object?> b) {
  if (a.length != b.length) return false;
  for (final entry in a.entries) {
    if (!b.containsKey(entry.key) || b[entry.key] != entry.value) {
      return false;
    }
  }
  return true;
}

/// Workflow visibility never grants permission to format arbitrary errors.
sealed class DiagnosticWorkflow {
  const DiagnosticWorkflow();
  Map<String, Object?> get selectedFields;
}

final class PublicRecordContext extends DiagnosticWorkflow {
  const PublicRecordContext({
    this.actorDid,
    this.targetDid,
    this.recordUri,
    this.handle,
    this.cid,
    this.nsid,
    this.recordKey,
  });
  final String? actorDid;
  final String? targetDid;
  final String? recordUri;
  final String? handle;
  final String? cid;
  final String? nsid;
  final String? recordKey;
  @override
  Map<String, Object?> get selectedFields => {
    if (actorDid != null) 'actorDid': actorDid,
    if (targetDid != null) 'targetDid': targetDid,
    if (recordUri != null) 'recordUri': recordUri,
    if (handle != null) 'handle': handle,
    if (cid != null) 'cid': cid,
    if (nsid != null) 'nsid': nsid,
    if (recordKey != null) 'recordKey': recordKey,
  };
}

final class AttemptedPublicDidContext extends DiagnosticWorkflow {
  const AttemptedPublicDidContext(this.value);
  static final _attemptedDid = RegExp(r'^did:[a-z0-9]+:[A-Za-z0-9._:!-]+$');
  final String value;
  @override
  Map<String, Object?> get selectedFields => {
    'attemptedDid': value.length <= 2048 && _attemptedDid.hasMatch(value)
        ? value
        : '[REDACTED]',
    'identifierValid': false,
    'validationReason': 'invalid DID',
  };
}

/// Construct only from an already published record failing decode/indexing.
final class PublishedRecordParseFailureContext extends DiagnosticWorkflow {
  const PublishedRecordParseFailureContext({
    required this.record,
    required this.text,
  });
  final PublicRecordContext record;
  final String text;
  @override
  Map<String, Object?> get selectedFields => {
    ...record.selectedFields,
    if (text.isNotEmpty) 'publicExcerpt': sanitizeKnownDiagnosticText(text),
  };
}

/// Explicit summary adapter for selected public model references and bounded
/// coarse state. Callers must declare a public workflow before passing record.
final class ModelDiagnosticSummary extends DiagnosticWorkflow {
  const ModelDiagnosticSummary({
    required this.model,
    required this.state,
    this.record,
    this.itemCount,
    this.hasMore,
    this.hasImage,
  });
  final DiagnosticModel model;
  final DiagnosticModelState state;
  final PublicRecordContext? record;
  final int? itemCount;
  final bool? hasMore;
  final bool? hasImage;
  @override
  Map<String, Object?> get selectedFields => {
    'model': model.name,
    'state': state.name,
    ...?record?.selectedFields,
    if (itemCount != null) 'itemCount': itemCount!.clamp(0, 1000000),
    if (hasMore != null) 'hasMore': hasMore,
    if (hasImage != null) 'hasImage': hasImage,
  };
}

enum DiagnosticModel {
  auth,
  settingsIdentity,
  activeIdentity,
  post,
  businessEvent,
  businessEventList,
}

enum DiagnosticModelState {
  signedIn,
  signedOut,
  available,
  unavailable,
  visible,
  restricted,
  scheduled,
  canceled,
  completed,
  unknown,
}

/// Construct at the operational failure boundary using its initiating account
/// and an existing non-capability workflow ID, never a target or receipt.
final class PrivateOperationalFailureContext extends DiagnosticWorkflow {
  const PrivateOperationalFailureContext({
    required this.accountDid,
    this.workflowRef,
  });
  final String accountDid;
  final String? workflowRef;
  static final _workflowUUID = RegExp(
    '^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-'
    r'[89ab][0-9a-f]{3}-[0-9a-f]{12}$',
  );
  @override
  Map<String, Object?> get selectedFields => {
    if (accountDid.length <= 2048 &&
        RegExp(r'^did:[a-z0-9]+:[A-Za-z0-9._:!-]+$').hasMatch(accountDid))
      'operationAccountDid': accountDid,
    if (workflowRef != null)
      'workflowRef': _workflowUUID.hasMatch(workflowRef!)
          ? workflowRef
          : '[OMITTED: invalid workflow reference]',
  };
}

DiagnosticWorkflow? selectedWorkflow(
  ReportContext context, {
  required bool failure,
}) {
  final workflow = context.workflow;
  if (workflow is PrivateOperationalFailureContext) {
    return failure ? workflow : null;
  }
  return workflow;
}
