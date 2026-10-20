import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:dio/dio.dart';

/// Runs [body], translating any `DioException` whose `.error` is an
/// `ApiException` (set by `ErrorMappingInterceptor`) into a direct
/// throw of that `ApiException`. Other `DioException`s — theoretically
/// unreachable, but defended against — surface as `ApiServerError`.
///
/// Every per-feature API client wraps its calls in this so callers
/// only ever see sealed `ApiException` subtypes.
Future<T> unwrapApi<T>(Future<T> Function() body) async {
  try {
    return await body();
  } on DioException catch (e, stack) {
    final original = e.stackTrace == StackTrace.empty ? stack : e.stackTrace;
    final err = e.error;
    if (err is ApiException) Error.throwWithStackTrace(err, original);
    final details = ApiFailureDetails(
      cause: err ?? e,
      stackTrace: original,
      statusCode: e.response?.statusCode,
      method: e.requestOptions.method,
    );
    Error.throwWithStackTrace(
      ApiServerError('server_error', details: details),
      original,
    );
  }
}
