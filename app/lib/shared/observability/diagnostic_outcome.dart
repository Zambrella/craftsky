import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:dio/dio.dart';

/// Known cancellation control flow, including app-owned cause wrappers.
bool isCanceledDiagnostic(Object sourceError) {
  var error = sourceError;
  final visited = Set<Object>.identity();
  for (var depth = 0; depth < 8 && visited.add(error); depth++) {
    if (error is! DiagnosticFailureCause || error.diagnosticCause == null) {
      break;
    }
    error = error.diagnosticCause!;
  }
  return _isCancellation(error);
}

bool _isCancellation(Object error) =>
    error is AccountOperationCanceled ||
    error is ApiCanceled ||
    (error is DioException &&
        (error.type == DioExceptionType.cancel || error.error is ApiCanceled));

/// Terminal retries override an ordinary offline/expiry class, without
/// overriding cancellation. Unknown failures retain their reportable default.
bool isExpectedDiagnostic(Object sourceError, ReportContext context) {
  var error = sourceError;
  final visited = Set<Object>.identity();
  AppError? appError;
  for (var depth = 0; depth < 8 && visited.add(error); depth++) {
    if (error is AppError) appError ??= error;
    if (error is! DiagnosticFailureCause || error.diagnosticCause == null) {
      break;
    }
    error = error.diagnosticCause!;
  }
  if (_isCancellation(error)) {
    return true;
  }
  if (context.outcome == DiagnosticOutcome.terminal) return false;
  if (context.outcome == DiagnosticOutcome.expected ||
      context.outcome == DiagnosticOutcome.retry) {
    return true;
  }
  if (appError != null) return !appError.reportable;
  if (error is AppError) return !error.reportable;
  if (error is ApiException ||
      (error is DioException && error.error is ApiException)) {
    return !AppErrorMapper.map(error).reportable;
  }
  return false;
}
