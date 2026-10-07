import 'dart:async';
import 'dart:convert' show utf8;

import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/observability/diagnostic_details.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/diagnostic_outcome.dart';
import 'package:craftsky_app/shared/observability/diagnostic_text.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/observability_bootstrap.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:craftsky_app/shared/observability/sentry_sanitizer.dart';
import 'package:dio/dio.dart';
import 'package:flutter/widgets.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';
import 'package:sentry_logging/sentry_logging.dart';

final class SentryFlutterBootstrapAdapter implements SentryBootstrapAdapter {
  const SentryFlutterBootstrapAdapter({this.appRunner});
  final Future<void> Function(ErrorReporter)? appRunner;

  @override
  Future<ErrorReporter> initialize(SentryConfig config) async {
    await SentryFlutter.init(
      (options) {
        options
          ..dsn = config.dsn
          ..environment = config.environment
          ..release = config.release
          ..dist = config.dist
          ..sendDefaultPii = false
          ..enableLogs = true
          ..tracesSampleRate = null
          ..enableAutoPerformanceTracing = false
          ..captureFailedRequests = false
          ..captureNativeFailedRequests = false;
        configureDiagnosticOptions(options);
        options.replay
          ..sessionSampleRate = 0
          ..onErrorSampleRate = 0;
      },
      appRunner: appRunner == null
          ? null
          : () => appRunner!(const GuardedErrorReporter(SentryErrorReporter())),
    );
    return const SentryErrorReporter();
  }
}

final class SentryErrorReporter implements ErrorReporter {
  const SentryErrorReporter();

  @override
  bool get enabled => true;

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    if (isExpectedDiagnostic(error, context)) return null;
    final eventId = await Sentry.captureException(
      error,
      stackTrace: stackTrace,
      withScope: (scope) async {
        await applyContext(scope, context, failure: true);
      },
    );
    return _eventIdOrNull(eventId);
  }

  static Future<void> applyContext(
    Scope scope,
    ReportContext context, {
    bool failure = false,
  }) async {
    final workflow = selectedWorkflow(context, failure: failure);
    if (workflow != null) {
      await scope.setContexts(
        'diagnostic',
        _boundFields(workflow.selectedFields),
      );
    }
    final sanitized = SentrySanitizer.sanitizeContext({
      'feature': context.feature,
      'operation': context.operation,
      'classification': context.classification,
      'severity': context.severity,
      ...context.safeDiagnostics,
    });
    await scope.setContexts('operation', sanitized);
    for (final entry in sanitized.entries) {
      if (!_diagnosticTagKeys.contains(entry.key)) continue;
      unawaited(scope.setTag(entry.key, entry.value.toString()));
    }
  }

  static SentryAttribute _attribute(Object? value) => switch (value) {
    int() => SentryAttribute.int(value),
    double() => SentryAttribute.double(value),
    bool() => SentryAttribute.bool(value),
    _ => SentryAttribute.string(value.toString()),
  };

  static String? _eventIdOrNull(SentryId id) {
    final value = id.toString();
    return value == const SentryId.empty().toString() ? null : value;
  }
}

// Final SDK hooks are the privacy boundary for automatic enrichment.
const _diagnosticTagKeys = {
  'feature',
  'operation',
  'classification',
  'severity',
  'appErrorKind',
  'httpStatus',
  'appViewError',
  'authState',
  'platform',
  'environment',
  'release',
};
const _workflowKeys = {
  'actorDid',
  'targetDid',
  'recordUri',
  'handle',
  'cid',
  'nsid',
  'recordKey',
  'attemptedDid',
  'identifierValid',
  'validationReason',
  'publicExcerpt',
  'operationAccountDid',
  'workflowRef',
  'model',
  'state',
  'itemCount',
  'hasMore',
  'hasImage',
};

/// GoRouter supplies static route names/path patterns, never concrete locations.
/// Remove arguments before the SDK formats them. Transactions remain disabled.
NavigatorObserver diagnosticNavigationObserver() => SentryNavigatorObserver(
  enableAutoTransactions: false,
  routeNameExtractor: (settings) => RouteSettings(name: settings?.name),
);

void configureDiagnosticOptions(SentryOptions options) {
  // Install after the client chooses its HTTP/native transport, before other
  // integrations run. Scope attachments are assembled after beforeSend.
  <ExceptionCauseExtractor<dynamic>>[
    _DiagnosticCauseExtractor<AppError>(),
    _DiagnosticCauseExtractor<ApiUnauthorized>(),
    _DiagnosticCauseExtractor<ApiCanceled>(),
    _DiagnosticCauseExtractor<ApiBadRequest>(),
    _DiagnosticCauseExtractor<ApiServerError>(),
    _DiagnosticCauseExtractor<ApiNetworkError>(),
    _DiagnosticCauseExtractor<DioException>(),
  ].forEach(options.addExceptionCauseExtractor);
  options
    ..addIntegrationByIndex(0, _AttachmentPrivacyIntegration())
    ..addIntegration(
      LoggingIntegration(
        minEventLevel: Level.OFF,
      ),
    )
    ..maxBreadcrumbs = 50
    ..beforeBreadcrumb = (breadcrumb, hint) {
      if (breadcrumb == null) return null;
      final safe = _protectBreadcrumb(breadcrumb);
      final record = hint.get(TypeCheckHint.record);
      if (safe != null && record is LogRecord) {
        safe.data = {
          'logger': boundDiagnosticText(record.loggerName, 160),
          ...?safe.data,
        };
        if (record.object case final DiagnosticMessage message) {
          safe.data!.addAll(
            _breadcrumbContext(message.context, failure: record.error != null),
          );
        }
      }
      return safe;
    }
    ..beforeSendLog = (log) {
      log.body = boundDiagnosticText(log.body, 512);
      final fields = {
        for (final entry in log.attributes.entries)
          entry.key: entry.value.value,
      };
      final selected = <String, Object?>{
        ...SentrySanitizer.sanitizeContext(fields),
        ..._boundFields({
          for (final entry in fields.entries)
            if (_workflowKeys.contains(entry.key)) entry.key: entry.value,
        }),
      };
      for (final key in const [
        'loggerName',
        'sentry.origin',
        'sentry.trace_id',
        'sentry.span_id',
      ]) {
        final value = fields[key];
        if (value is String &&
            RegExp(r'^[A-Za-z0-9_.:-]{1,160}$').hasMatch(value)) {
          selected[key] = value;
        }
      }
      log.attributes = {
        for (final entry in selected.entries)
          entry.key: SentryErrorReporter._attribute(entry.value),
      };
      return log;
    }
    ..beforeSendTransaction = (transaction, hint) {
      _protectEvent(transaction);
      transaction.measurements.clear();
      transaction.contexts.trace?.description = null;
      transaction.contexts.trace?.data = null;
      transaction.transaction = _safeOperation(
        transaction.transaction ?? 'unknown',
      );
      for (final span in transaction.spans) {
        span.context.description = null;
        span.context.operation = _safeOperation(span.context.operation);
        span.data.clear();
        span.tags.clear();
      }
      return transaction;
    }
    ..beforeSend = (event, hint) {
      // SDK automatic integrations decorate the original typed throwable.
      // Explicit owners already classify with their terminal/retry context.
      if (event.throwableMechanism case ThrowableMechanism(:final throwable)) {
        if (throwable is Object &&
            isExpectedDiagnostic(
              throwable,
              const ReportContext(
                feature: 'flutter',
                operation: 'automatic_error',
                classification: 'flutter.automatic',
              ),
            )) {
          return null;
        }
      }
      hint.attachments.clear();
      hint
        ..screenshot = null
        ..viewHierarchy = null;
      return _protectEvent(event);
    };
}

// The SDK builds native cause relationships and stacks; the app only exposes
// the causes carried by its wrappers. No parallel diagnostic exception model.
final class _DiagnosticCauseExtractor<T> extends ExceptionCauseExtractor<T> {
  @override
  ExceptionCause? cause(T error) {
    final (cause, stack) = switch (error) {
      DiagnosticFailureCause(:final diagnosticCause, :final diagnosticStack) =>
        (diagnosticCause, diagnosticStack),
      ApiException(:final details) => (details.cause, details.stackTrace),
      DioException(:final error, :final stackTrace) => (error, stackTrace),
      _ => (null, null),
    };
    return cause == null ? null : ExceptionCause(cause, stack);
  }
}

final class _AttachmentPrivacyIntegration extends Integration<SentryOptions> {
  @override
  void call(Hub hub, SentryOptions options) {
    options.transport = _WithoutAttachmentsTransport(options.transport);
  }
}

final class _WithoutAttachmentsTransport implements Transport {
  const _WithoutAttachmentsTransport(this.delegate);
  final Transport delegate;

  @override
  Future<SentryId?> send(SentryEnvelope envelope) {
    // Remove items before their loaders run or the SDK caches the envelope.
    envelope.items.removeWhere((item) => item.header.type == 'attachment');
    return delegate.send(envelope);
  }
}

SentryEvent _protectEvent(SentryEvent event) {
  event
    ..request = null
    ..user = null
    // Remove legacy SDK enrichment as well as supported request/user fields.
    // ignore: deprecated_member_use
    ..extra = null
    ..message = null
    ..serverName = null
    ..threads = null;
  // Filter tags separately from the private-field clearing cascade above.
  // ignore: cascade_invocations
  event.tags = {
    for (final entry in SentrySanitizer.sanitizeContext(
      Map<String, Object?>.from(event.tags ?? {}),
    ).entries)
      if (_diagnosticTagKeys.contains(entry.key))
        entry.key: entry.value.toString(),
  };
  if (event.contexts['trace'] case final SentryTraceContext trace) {
    trace
      ..data = null
      ..description = null;
  }
  if (event.contexts['device'] case final SentryDevice device) {
    device
      ..name = null
      ..deviceUniqueIdentifier = null;
  }
  if (event.contexts['app'] case final SentryApp app) {
    app.deviceAppHash = null;
  }
  // Keep standard SDK metadata, and only our small operational custom contexts.
  for (final key in event.contexts.keys.toList()) {
    final value = event.contexts[key];
    if (key == 'diagnostic' && value is Map) {
      event.contexts[key] = _boundFields({
        for (final entry in value.entries)
          if (_workflowKeys.contains(entry.key))
            entry.key as String: entry.value,
      });
    } else if (key == 'operation' && value is Map) {
      event.contexts[key] = SentrySanitizer.sanitizeContext(
        Map<String, Object?>.from(value),
      );
    } else if (!const {
      'app',
      'device',
      'os',
      'runtime',
      'trace',
      'browser',
      'gpu',
    }.contains(key)) {
      event.contexts.remove(key);
    }
  }
  for (final exception in event.exceptions ?? <SentryException>[]) {
    final throwable = exception.throwable;
    exception.value = throwable is Object
        ? selectedCause(throwable)['message']! as String
        : 'Operation failed';
    final stack = exception.stackTrace;
    if (stack != null &&
        stack.frames.any(
          (frame) =>
              frame.vars.isNotEmpty ||
              frame.contextLine != null ||
              frame.preContext.isNotEmpty ||
              frame.postContext.isNotEmpty,
        )) {
      // Only frames carrying private locals/source need replacement. Native
      // frames without those fields keep the SDK structure and symbolication.
      exception.stackTrace = SentryStackTrace(
        snapshot: stack.snapshot,
        lang: stack.lang,
        frames: [
          for (final frame in stack.frames)
            SentryStackFrame(
              function: frame.function,
              module: frame.module,
              fileName: frame.fileName,
              absPath: frame.absPath,
              lineNo: frame.lineNo,
              colNo: frame.colNo,
              inApp: frame.inApp,
              instructionAddr: frame.instructionAddr,
              imageAddr: frame.imageAddr,
              symbolAddr: frame.symbolAddr,
              rawFunction: frame.rawFunction,
              package: frame.package,
              native: frame.native,
              platform: frame.platform,
              stackStart: frame.stackStart,
              symbol: frame.symbol,
              framesOmitted: frame.framesOmitted,
            ),
        ],
      );
    }
    for (final frame in exception.stackTrace?.frames ?? <SentryStackFrame>[]) {
      frame.absPath = null;
      if (frame.fileName != null) {
        frame.fileName = boundDiagnosticText(
          frame.fileName!.replaceAll(r'\', '/').split('/').last,
          160,
        );
      }
      if (frame.function != null) {
        frame.function = boundDiagnosticText(frame.function!, 256);
      }
    }
  }
  event.breadcrumbs = [
    for (final breadcrumb in event.breadcrumbs ?? <Breadcrumb>[])
      if (_protectBreadcrumb(breadcrumb) case final Breadcrumb safe) safe,
  ];
  return event;
}

Map<String, Object?> _boundFields(Map<String, Object?> fields) => {
  for (final entry in fields.entries.take(20))
    entry.key: switch (entry.value) {
      final String value =>
        utf8.encode(value).length > maxDiagnosticTextBytes &&
                entry.key != 'publicExcerpt'
            ? '[OMITTED: oversized identifier]'
            : sanitizeKnownDiagnosticText(value),
      bool() || num() => entry.value,
      _ => '[OMITTED]',
    },
};
String _safeOperation(String value) =>
    RegExp(r'^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$').hasMatch(value)
    ? value
    : 'unknown';
Breadcrumb? _protectBreadcrumb(Breadcrumb breadcrumb) {
  final safe = SentrySanitizer.sanitizeBreadcrumb(
    SafeBreadcrumb(
      category: breadcrumb.category ?? '',
      message: breadcrumb.message ?? '',
      data: Map<String, Object?>.from(breadcrumb.data ?? {}),
    ),
  );
  if (safe == null) return null;
  return Breadcrumb(
    timestamp: breadcrumb.timestamp,
    category: safe.category,
    message: safe.message,
    data: {
      ...safe.data,
      ...SentrySanitizer.sanitizeContext(
        Map<String, Object?>.from(breadcrumb.data ?? {}),
      ),
      ..._boundFields({
        for (final entry in (breadcrumb.data ?? {}).entries)
          if (_workflowKeys.contains(entry.key)) entry.key: entry.value,
      }),
    },
    level: breadcrumb.level,
  );
}

Map<String, Object?> _breadcrumbContext(
  ReportContext context, {
  required bool failure,
}) => {
  ...SentrySanitizer.sanitizeContext({
    'feature': context.feature,
    'operation': context.operation,
    ...context.safeDiagnostics,
  }),
  ...?_workflowFields(context, failure: failure),
};
Map<String, Object?>? _workflowFields(
  ReportContext context, {
  required bool failure,
}) {
  final workflow = selectedWorkflow(context, failure: failure);
  return workflow == null ? null : _boundFields(workflow.selectedFields);
}
