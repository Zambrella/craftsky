import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:dio/dio.dart';

/// Auth-related AppView endpoints. Assumes the attached [Dio] has the
/// auth + error interceptors installed (see `dioProvider`); each call
/// is wrapped in `unwrapApi` so consumers see sealed `ApiException`
/// subtypes instead of raw `DioException`s.
class AuthApiClient {
  const AuthApiClient(this._dio);

  final Dio _dio;

  /// POST /v1/auth/login — starts an OAuth flow for [handle], returns
  /// the authorization URL the caller opens in the system browser.
  /// The app-level handoff is always `deep_link` (mobile-only).
  Future<LoginResponse> login({required String handle}) => unwrapApi(() async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/v1/auth/login',
      data: {'handle': handle, 'handoffMode': 'deep_link'},
    );
    return LoginResponseMapper.fromMap(res.data!);
  });

  /// GET /v1/whoami — resolves the caller's DID + handle. Requires an
  /// authenticated request (Bearer token attached by AuthInterceptor).
  Future<WhoAmI> whoami() => unwrapApi(() async {
    final res = await _dio.get<Map<String, dynamic>>('/v1/whoami');
    return WhoAmIMapper.fromMap(res.data!);
  });

  /// POST /v1/auth/logout — revokes the current CraftSky session
  /// (single-device). Server responds 204.
  Future<void> logout() => unwrapApi(() async {
    await _dio.post<void>('/v1/auth/logout');
  });
}
