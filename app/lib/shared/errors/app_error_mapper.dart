import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:dio/dio.dart';

final class AppErrorMapper {
  const AppErrorMapper._();

  static AppError map(
    Object sourceError, {
    AppErrorKind fallbackKind = AppErrorKind.unexpected,
    String source = 'unknown',
    String? fallbackClassification,
  }) {
    var error = sourceError;
    final visited = Set<Object>.identity();
    for (var depth = 0; depth < 8 && visited.add(error); depth++) {
      if (error is AppError) return error;
      if (error is! DiagnosticFailureCause || error.diagnosticCause == null) {
        break;
      }
      error = error.diagnosticCause!;
    }
    return switch (error) {
      DioException(error: final ApiException mapped) => _mapApiException(
        mapped,
      ),
      ApiException() => _mapApiException(error),
      FormatException() => _fallbackForSource(
        fallbackKind,
        source,
        classification: 'parse.failed',
      ),
      _ => _fallbackForSource(
        fallbackKind,
        source,
        classification: fallbackClassification,
      ),
    };
  }

  static AppError _mapApiException(ApiException error) {
    if (error.details.cause is FormatException) {
      return AppError(
        AppErrorKind.unexpected,
        reportableOverride: true,
        sentryClassificationOverride: 'parse.failed',
        safeDiagnostics: _apiDiagnostics(error),
        diagnosticCause: error.details.cause,
        diagnosticStack: error.details.stackTrace,
      );
    }
    return switch (error) {
      ApiUnauthorized() => AppError(
        AppErrorKind.sessionExpired,
        reportableOverride: false,
        sentryClassificationOverride: 'api.unauthorized',
        safeDiagnostics: _apiDiagnostics(error),
        diagnosticCause: error.details.cause ?? error,
        diagnosticStack: error.details.stackTrace,
      ),
      ApiBadRequest(:final code) when code == 'not_found' => AppError(
        AppErrorKind.contentUnavailable,
        reportableOverride: false,
        sentryClassificationOverride: 'api.not_found',
        safeDiagnostics: _apiDiagnostics(error),
        diagnosticCause: error.details.cause ?? error,
        diagnosticStack: error.details.stackTrace,
      ),
      ApiBadRequest(:final code) => AppError(
        AppErrorKind.actionFailed,
        reportableOverride: false,
        sentryClassificationOverride: 'api.bad_request',
        safeDiagnostics: _apiDiagnostics(error, appViewError: code),
        diagnosticCause: error.details.cause ?? error,
        diagnosticStack: error.details.stackTrace,
      ),
      ApiServerError() => AppError(
        AppErrorKind.serviceUnavailable,
        reportableOverride: true,
        sentryClassificationOverride: 'api.server_error',
        safeDiagnostics: _apiDiagnostics(error),
        diagnosticCause: error.details.cause ?? error,
        diagnosticStack: error.details.stackTrace,
      ),
      ApiNetworkError() => AppError(
        AppErrorKind.networkUnavailable,
        reportableOverride: false,
        sentryClassificationOverride: 'api.network',
        safeDiagnostics: _apiDiagnostics(error),
        diagnosticCause: error.details.cause ?? error,
        diagnosticStack: error.details.stackTrace,
      ),
      ApiCanceled() => AppError(
        AppErrorKind.actionFailed,
        reportableOverride: false,
        sentryClassificationOverride: 'api.canceled',
        safeDiagnostics: _apiDiagnostics(error),
        diagnosticCause: error.details.cause ?? error,
        diagnosticStack: error.details.stackTrace,
      ),
    };
  }

  static Map<String, Object?> _apiDiagnostics(
    ApiException error, {
    String? appViewError,
  }) {
    final details = error.details;
    return {
      'source': 'api',
      if (details.method != null) 'httpMethod': details.method,
      if (details.statusCode != null) 'httpStatus': details.statusCode,
      if ((appViewError ?? details.appViewError) != null)
        'appViewError': appViewError ?? details.appViewError,
      if (details.requestId != null) 'appViewRequestId': details.requestId,
    };
  }

  static AppError _fallbackForSource(
    AppErrorKind kind,
    String source, {
    String? classification,
  }) => AppError(
    kind,
    sentryClassificationOverride: classification,
    safeDiagnostics: {'source': source},
  );
}
