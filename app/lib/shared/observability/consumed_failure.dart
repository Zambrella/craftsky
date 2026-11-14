import 'dart:async';

import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_outcome.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_riverpod/misc.dart' show ProviderException;
import 'package:logging/logging.dart';

/// Owns a failure consumed into ordinary state or a result.
/// Capture the reporter and initiating account before awaiting the operation.
void reportConsumedFailure(
  Object error,
  StackTrace stackTrace, {
  required ErrorReporter reporter,
  required String feature,
  required String operation,
  String? accountDid,
  AppErrorKind fallbackKind = AppErrorKind.actionFailed,
  DiagnosticOutcome outcome = DiagnosticOutcome.automatic,
}) {
  // A failed provider read already has its issue owner in ProviderLogger.
  if (error is ProviderException) return;
  final mapped = AppErrorMapper.map(
    error,
    fallbackKind: fallbackKind,
    source: feature,
  );
  final context = ReportContext(
    feature: feature,
    operation: operation,
    classification: mapped.sentryClassification,
    severity: mapped.metadata.severity.name,
    outcome: outcome,
    safeDiagnostics: {
      ...mapped.safeDiagnostics,
      if (outcome != DiagnosticOutcome.automatic) 'outcome': outcome.name,
    },
    workflow: accountDid == null
        ? null
        : PrivateOperationalFailureContext(accountDid: accountDid),
  );
  final expected = isExpectedDiagnostic(error, context);
  Logger(feature).log(
    isCanceledDiagnostic(error)
        ? Level.FINE
        : expected
        ? Level.WARNING
        : Level.SEVERE,
    DiagnosticMessage('Operation failed', context: context),
    error,
    stackTrace,
  );
  if (!expected) {
    unawaited(
      GuardedErrorReporter(reporter).captureException(
        error,
        stackTrace: stackTrace,
        context: context,
      ),
    );
  }
}
