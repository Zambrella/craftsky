import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';

final class AppErrorMapper {
  const AppErrorMapper._();

  static AppError map(
    Object error, {
    AppErrorKind fallbackKind = AppErrorKind.unexpected,
    String source = 'unknown',
    String? fallbackClassification,
  }) {
    return switch (error) {
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
    return switch (error) {
      ApiUnauthorized() => AppError(
        AppErrorKind.sessionExpired,
        reportableOverride: false,
        sentryClassificationOverride: 'api.unauthorized',
        safeDiagnostics: _apiDiagnostics(error),
      ),
      ApiBadRequest(:final code) when code == 'not_found' => AppError(
        AppErrorKind.contentUnavailable,
        reportableOverride: false,
        sentryClassificationOverride: 'api.not_found',
        safeDiagnostics: _apiDiagnostics(error),
      ),
      ApiBadRequest(:final code) => AppError(
        AppErrorKind.actionFailed,
        reportableOverride: false,
        sentryClassificationOverride: 'api.bad_request',
        safeDiagnostics: _apiDiagnostics(error, appViewError: code),
      ),
      ApiServerError() => AppError(
        AppErrorKind.serviceUnavailable,
        reportableOverride: true,
        sentryClassificationOverride: 'api.server_error',
        safeDiagnostics: _apiDiagnostics(error),
      ),
      ApiNetworkError() => AppError(
        AppErrorKind.networkUnavailable,
        reportableOverride: false,
        sentryClassificationOverride: 'api.network',
        safeDiagnostics: _apiDiagnostics(error),
      ),
      ApiCanceled() => AppError(
        AppErrorKind.actionFailed,
        reportableOverride: false,
        sentryClassificationOverride: 'api.canceled',
        safeDiagnostics: _apiDiagnostics(error),
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
      if (details.statusCode != null) 'httpStatus': details.statusCode,
      if ((appViewError ?? details.appViewError) != null)
        'appViewError': appViewError ?? details.appViewError,
      if (details.requestId != null) 'appViewRequestId': details.requestId,
      if (details.endpointCategory != null)
        'endpointCategory': details.endpointCategory,
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
