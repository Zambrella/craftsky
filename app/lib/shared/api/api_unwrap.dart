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
  } on DioException catch (e) {
    final err = e.error;
    if (err is ApiException) throw err;
    throw ApiServerError(e.message ?? 'server_error');
  }
}
