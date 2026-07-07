import 'dart:async';

import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/observability_bootstrap.dart';
import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:craftsky_app/shared/observability/sentry_sanitizer.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

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
        ..sendDefaultPii = false
        ..enableLogs = true
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
  const SentryErrorReporter();

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
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    final eventId = await Sentry.captureException(
      error,
      stackTrace: stackTrace,
      withScope: (scope) => applyContext(scope, context),
    );
    return _eventIdOrNull(eventId);
  }

  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {
    final attributes = attributesFor(context);
    switch (context.severity) {
      case 'fatal':
        await Sentry.logger.fatal(message, attributes: attributes);
      case 'warning':
        await Sentry.logger.warn(message, attributes: attributes);
      default:
        await Sentry.logger.error(message, attributes: attributes);
    }
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
