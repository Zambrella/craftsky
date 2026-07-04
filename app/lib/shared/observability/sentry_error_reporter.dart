import 'dart:async';

import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/observability_bootstrap.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:craftsky_app/shared/observability/sentry_sanitizer.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

abstract interface class SentryLogSink {
  Future<void> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  });

  Future<void> captureMessage(
    Level level, {
    required ReportContext context,
  });
}

final class SentrySdkLogSink implements SentryLogSink {
  const SentrySdkLogSink();

  @override
  Future<void> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    await Sentry.captureException(
      error,
      stackTrace: stackTrace,
      withScope: (scope) => SentryErrorReporter.applyContext(scope, context),
    );
  }

  @override
  Future<void> captureMessage(
    Level level, {
    required ReportContext context,
  }) async {
    final attributes = SentryErrorReporter.attributesFor(context);
    if (level.value >= Level.SHOUT.value) {
      await Sentry.logger.fatal('App log', attributes: attributes);
    } else {
      await Sentry.logger.error('App log', attributes: attributes);
    }
  }
}

final class SentryFlutterBootstrapAdapter implements SentryBootstrapAdapter {
  const SentryFlutterBootstrapAdapter();

  @override
  Future<ErrorReporter> initialize(SentryConfig config) async {
    await SentryFlutter.init((options) {
      options
        ..dsn = config.dsn
        ..environment = config.environment
        ..release = config.release
        ..dist = config.dist
        ..sendDefaultPii = config.options.sendDefaultPii
        ..enableLogs = config.options.enableLogs
        ..tracesSampleRate = null
        ..enableAutoPerformanceTracing = false
        ..captureFailedRequests = false
        ..captureNativeFailedRequests = false;
      options.replay
        ..sessionSampleRate = 0
        ..onErrorSampleRate = 0;
      options.beforeBreadcrumb = (breadcrumb, hint) {
        if (breadcrumb == null) return null;
        final safe = SentrySanitizer.sanitizeBreadcrumb(
          SafeBreadcrumb(
            category: breadcrumb.category ?? '',
            message: breadcrumb.message ?? '',
            data: Map<String, Object?>.from(breadcrumb.data ?? const {}),
          ),
        );
        if (safe == null) return null;
        return Breadcrumb(
          category: safe.category,
          message: safe.message,
          data: safe.data,
          level: SentryLevel.info,
        );
      };
    });
    return const SentryErrorReporter();
  }
}

final class SentryErrorReporter implements ErrorReporter {
  const SentryErrorReporter({SentryLogSink logSink = const SentrySdkLogSink()})
    : this._(logSink);

  const SentryErrorReporter._(this._logSink);

  final SentryLogSink _logSink;

  @override
  bool get enabled => true;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {
    final safe = SentrySanitizer.sanitizeBreadcrumb(breadcrumb);
    if (safe == null) return;
    unawaited(
      Sentry.addBreadcrumb(
        Breadcrumb(
          category: safe.category,
          message: safe.message,
          data: safe.data,
          level: SentryLevel.info,
        ),
      ),
    );
  }

  @override
  Future<ReportResult> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    final eventId = await Sentry.captureException(
      error,
      stackTrace: stackTrace,
      withScope: (scope) => applyContext(scope, context),
    );
    return ReportResult.captured(eventId: _eventIdOrNull(eventId));
  }

  @override
  Future<void> captureLog(
    LogRecord record, {
    required ReportContext context,
  }) async {
    if (record.level.value >= Level.SEVERE.value &&
        (record.error != null || record.stackTrace != null)) {
      await _logSink.captureException(
        record.error ?? _LogRecordException(record.message),
        context: context,
        stackTrace: record.stackTrace,
      );
      return;
    }

    await _logSink.captureMessage(record.level, context: context);
  }

  static void applyContext(Scope scope, ReportContext context) {
    final sanitized = SentrySanitizer.sanitizeContext({
      'feature': context.feature,
      'operation': context.operation,
      'classification': context.classification,
      'severity': context.severity,
      ...context.safeDiagnostics,
    });
    for (final entry in sanitized.entries) {
      unawaited(scope.setTag(entry.key, entry.value.toString()));
    }
  }

  static Map<String, SentryAttribute> attributesFor(ReportContext context) {
    final sanitized = SentrySanitizer.sanitizeContext({
      'feature': context.feature,
      'operation': context.operation,
      'classification': context.classification,
      'severity': context.severity,
      ...context.safeDiagnostics,
    });
    return {
      for (final entry in sanitized.entries) entry.key: _attribute(entry.value),
    };
  }

  static SentryAttribute _attribute(Object? value) {
    return switch (value) {
      int() => SentryAttribute.int(value),
      double() => SentryAttribute.double(value),
      bool() => SentryAttribute.bool(value),
      _ => SentryAttribute.string(value.toString()),
    };
  }

  static String? _eventIdOrNull(SentryId id) {
    final value = id.toString();
    return value == const SentryId.empty().toString() ? null : value;
  }
}

final class _LogRecordException implements Exception {
  const _LogRecordException(this.message);

  final String message;

  @override
  String toString() => 'LogRecordException: $message';
}
