import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/api/providers/session_auth_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'dio_provider.g.dart';

/// Android emulator maps the host machine to 10.0.2.2. iOS simulator
/// reaches localhost directly. Android is the more common footgun so
/// it's the default; iOS devs pass
/// --dart-define=CRAFTSKY_API_BASE_URL=http://localhost:8080.
const _devDefaultBaseUrl = 'http://10.0.2.2:8080';

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
  final dio = Dio(baseDioOptions());
  dio.interceptors.addAll([
    SessionAuthInterceptor.fromRef(ref),
    const ErrorMappingInterceptor(),
    // Task 14b adds SignOutOn401Interceptor below ErrorMapping.
  ]);
  return dio;
}
