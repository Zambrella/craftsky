import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:dio/dio.dart';

/// Thin typed wrapper around the three AppView endpoints this release
/// needs. All calls assume the attached [Dio] has the auth + error
/// interceptors installed (see dioProvider — assembled across Tasks
/// 7, 14a, 14b).
///
/// Each method unwraps the `DioException` that `dio` throws and
/// rethrows the `ApiException` carried in its `.error` field — so
/// callers only ever deal in `ApiException` subtypes.
class CraftskyApiClient {
  const CraftskyApiClient(this._dio);

  final Dio _dio;

  /// POST /v1/auth/login — starts an OAuth flow for [handle], returns
  /// the authorization URL the caller opens in the system browser.
  /// The app-level handoff is always `deep_link` (mobile-only).
  Future<LoginResponse> login({required String handle}) => _unwrap(() async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/v1/auth/login',
      data: {'handle': handle, 'handoff_mode': 'deep_link'},
    );
    return LoginResponseMapper.fromMap(res.data!);
  });

  /// GET /v1/whoami — resolves the caller's DID + handle. Requires an
  /// authenticated request (Bearer token attached by AuthInterceptor).
  Future<WhoAmI> whoami() => _unwrap(() async {
    final res = await _dio.get<Map<String, dynamic>>('/v1/whoami');
    return WhoAmIMapper.fromMap(res.data!);
  });

  /// POST /v1/auth/logout — revokes the current Craftsky session
  /// (single-device). Server responds 204.
  Future<void> logout() => _unwrap(() async {
    await _dio.post<void>('/v1/auth/logout');
  });

  /// Runs [body], translating any `DioException` whose `.error` is an
  /// `ApiException` into a direct throw of that `ApiException`. Other
  /// `DioException`s — theoretically unreachable because
  /// `ErrorMappingInterceptor` always sets `.error` — surface as
  /// `ApiServerError` with the underlying message.
  Future<T> _unwrap<T>(Future<T> Function() body) async {
    try {
      return await body();
    } on DioException catch (e) {
      final err = e.error;
      if (err is ApiException) throw err;
      throw ApiServerError(e.message ?? 'server_error');
    }
  }
}
