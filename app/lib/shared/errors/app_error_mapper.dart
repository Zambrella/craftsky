import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';

enum AppErrorSource {
  api,
  storage,
  initialization,
  routing,
  action,
  backgroundLoad,
  provider,
  unknown,
}

final class AppErrorMapper {
  const AppErrorMapper._();

  static AppError map(
    Object error, {
    AppErrorSource source = AppErrorSource.unknown,
  }) {
    return switch (error) {
      ApiException() => _mapApiException(error),
      FormatException() => _fallbackForSource(
        source,
        classification: 'parse.failed',
      ),
      _ when source == AppErrorSource.storage => const AppError(
        AppErrorKind.storageUnavailable,
        reportableOverride: true,
        sentryClassificationOverride: 'storage.failed',
        safeDiagnostics: {'source': 'storage'},
      ),
      _ => _fallbackForSource(source),
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
    AppErrorSource source, {
    String? classification,
  }) {
    return switch (source) {
      AppErrorSource.initialization => AppError(
        AppErrorKind.initializationFailed,
        sentryClassificationOverride: classification,
        safeDiagnostics: const {'source': 'initialization'},
      ),
      AppErrorSource.routing => AppError(
        AppErrorKind.navigationFailed,
        sentryClassificationOverride: classification,
        safeDiagnostics: const {'source': 'routing'},
      ),
      AppErrorSource.action => AppError(
        AppErrorKind.actionFailed,
        sentryClassificationOverride: classification,
        safeDiagnostics: const {'source': 'action'},
      ),
      AppErrorSource.backgroundLoad => AppError(
        AppErrorKind.backgroundLoadFailed,
        sentryClassificationOverride: classification,
        safeDiagnostics: const {'source': 'background_load'},
      ),
      AppErrorSource.provider => AppError(
        AppErrorKind.backgroundLoadFailed,
        sentryClassificationOverride: classification ?? 'provider.failed',
        safeDiagnostics: const {'source': 'provider'},
      ),
      AppErrorSource.storage => AppError(
        AppErrorKind.storageUnavailable,
        sentryClassificationOverride: classification ?? 'storage.failed',
        safeDiagnostics: const {'source': 'storage'},
      ),
      AppErrorSource.api || AppErrorSource.unknown => AppError(
        AppErrorKind.unexpected,
        sentryClassificationOverride: classification,
        safeDiagnostics: const {'source': 'unknown'},
      ),
    };
  }
}
