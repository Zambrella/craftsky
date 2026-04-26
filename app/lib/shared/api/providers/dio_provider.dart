import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/api/providers/session_auth_interceptor.dart';
import 'package:craftsky_app/shared/api/providers/sign_out_on_401_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'dio_provider.g.dart';

/// Android emulator maps the host machine to 10.0.2.2. iOS simulator
/// reaches localhost directly. Android is the more common footgun so
/// it's the default; iOS devs pass
/// --dart-define=CRAFTSKY_API_BASE_URL=http://localhost:18080.
///
/// Port 18080 (not 8080) matches the host-side mapping in
/// docker-compose.yml — the appview container still listens on 8080
/// internally but is published on 18080 to avoid colliding with other
/// dev servers.
const _devDefaultBaseUrl = 'http://10.0.2.2:18080';

const _baseUrl = String.fromEnvironment(
  'CRAFTSKY_API_BASE_URL',
  defaultValue: kDebugMode ? _devDefaultBaseUrl : '',
);

/// Shared base options for both the session Dio (this file) and the
/// handoff Dio (api_client_provider.dart, family) so HTTP basics stay
/// in sync.
BaseOptions baseDioOptions() {
  if (_baseUrl.isEmpty) {
    throw StateError(
      'CRAFTSKY_API_BASE_URL must be set for non-debug builds. '
      'Pass it via --dart-define.',
    );
  }
  return BaseOptions(
    baseUrl: _baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 15),
    headers: {'Content-Type': 'application/json'},
  );
}

@Riverpod(keepAlive: true)
Dio dio(Ref ref) {
  // Interceptor ordering matters:
  //   1. SessionAuthInterceptor — attaches Bearer on outgoing requests.
  //   2. ErrorMappingInterceptor — converts DioException → ApiException
  //      in .error, PRESERVING response/statusCode on the replacement.
  //   3. SignOutOn401Interceptor — reads err.response?.statusCode == 401
  //      to decide whether to sign out. If (2) ever stops preserving
  //      response, (3) silently breaks in prod (tests use synthetic
  //      DioException objects with statusCode already set).
  final dio = Dio(baseDioOptions());
  dio.interceptors.addAll([
    SessionAuthInterceptor.fromRef(ref),
    const ErrorMappingInterceptor(),
    SignOutOn401Interceptor.fromRef(ref),
  ]);
  return dio;
}
