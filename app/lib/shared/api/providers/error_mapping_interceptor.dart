import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:dio/dio.dart';

/// Stateless, side-effect-free. Both the session Dio and the handoff
/// Dio attach this interceptor; the session Dio additionally installs
/// `_SignOutOn401Interceptor` (see Task 14b).
class ErrorMappingInterceptor extends Interceptor {
  const ErrorMappingInterceptor();

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    handler.next(
      DioException(
        requestOptions: err.requestOptions,
        response: err.response,
        type: err.type,
        error: _mapDioError(err),
        stackTrace: err.stackTrace,
      ),
    );
  }

  ApiException _mapDioError(DioException err) {
    final details = _detailsFor(err);
    return switch (err.type) {
      DioExceptionType.connectionTimeout ||
      DioExceptionType.sendTimeout ||
      DioExceptionType.receiveTimeout ||
      DioExceptionType.connectionError => ApiNetworkError(
        err.message ?? err.type.name,
        details: details,
      ),
      DioExceptionType.badResponse => _mapBadResponse(err),
      DioExceptionType.cancel => ApiCanceled(details: details),
      DioExceptionType.badCertificate || DioExceptionType.unknown =>
        err.error is Exception
            ? ApiNetworkError(err.message ?? 'network_error', details: details)
            : ApiServerError(err.message ?? 'server_error', details: details),
    };
  }

  ApiException _mapBadResponse(DioException err) {
    final status = err.response?.statusCode ?? 0;
    final details = _detailsFor(err);
    if (status == 401) return ApiUnauthorized(details: details);
    if (status >= 400 && status < 500) {
      return ApiBadRequest(details.appViewError, details: details);
    }
    return ApiServerError('http_$status', details: details);
  }

  ApiFailureDetails _detailsFor(DioException err) {
    final data = err.response?.data;
    final statusCode = err.response?.statusCode;
    String? field(String key, RegExp pattern) {
      final value = data is Map ? data[key] : null;
      return value is String && pattern.hasMatch(value) ? value : null;
    }

    // Server prose and resource paths stay out of diagnostics.
    // Validation fields are retained separately for UI validation.
    // Codes are a bounded API contract, not a catalogue of server wording.
    return ApiFailureDetails(
      statusCode: err.response?.statusCode,
      appViewError: field('error', RegExp(r'^[a-z][a-z0-9_]{0,79}$')),
      requestId: field('requestId', RegExp(r'^[A-Za-z0-9_-]{1,160}$')),
      method: err.requestOptions.method,
      cause: err.error ?? err,
      stackTrace: err.stackTrace,
      fields:
          statusCode != null &&
              statusCode >= 400 &&
              statusCode < 500 &&
              data is Map &&
              data['fields'] is Map &&
              err.requestOptions.uri.path != '/v1/auth/registrations'
          ? Map.unmodifiable({
              for (final entry in (data['fields'] as Map).entries)
                if (entry.key is String && entry.value is String)
                  entry.key as String: entry.value as String,
            })
          : const {},
    );
  }
}
