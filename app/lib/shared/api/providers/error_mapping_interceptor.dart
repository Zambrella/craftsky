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
    return switch (err.type) {
      DioExceptionType.connectionTimeout ||
      DioExceptionType.sendTimeout ||
      DioExceptionType.receiveTimeout ||
      DioExceptionType.connectionError => ApiNetworkError(
        err.message ?? err.type.name,
      ),
      DioExceptionType.badResponse => _mapBadResponse(err),
      DioExceptionType.cancel => const ApiCanceled(),
      DioExceptionType.badCertificate || DioExceptionType.unknown =>
        err.error is Exception
            ? ApiNetworkError(err.message ?? 'network_error')
            : ApiServerError(err.message ?? 'server_error'),
    };
  }

  ApiException _mapBadResponse(DioException err) {
    final status = err.response?.statusCode ?? 0;
    if (status == 401) return const ApiUnauthorized();
    if (status >= 400 && status < 500) {
      final data = err.response?.data;
      final code = data is Map && data['error'] is String
          ? data['error'] as String
          : null;
      return ApiBadRequest(code);
    }
    return ApiServerError('http_$status');
  }
}
