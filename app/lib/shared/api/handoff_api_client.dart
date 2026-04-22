import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:dio/dio.dart';

/// Minimal, token-scoped API client used for exactly one `whoami` call
/// during the OAuth handoff. Unlike `CraftskyApiClient`, this doesn't
/// participate in the session Dio's auth/state wiring — the Bearer
/// token is baked into `BaseOptions.headers` at construction.
class HandoffApiClient {
  const HandoffApiClient(this._dio);
  final Dio _dio;

  Future<WhoAmI> whoami() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/v1/whoami');
      return WhoAmIMapper.fromMap(res.data!);
    } on DioException catch (e) {
      final err = e.error;
      if (err is ApiException) throw err;
      throw ApiServerError(e.message ?? 'server_error');
    }
  }
}
