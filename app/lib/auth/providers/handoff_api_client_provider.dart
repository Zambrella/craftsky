import 'package:craftsky_app/auth/data/handoff_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'handoff_api_client_provider.g.dart';

/// Family-keyed by (token, deviceId): one instance per in-flight
/// handoff. Not keep-alive — auto-disposes when no one watches it, so
/// the token doesn't linger.
///
/// The server enforces X-Craftsky-Device-Id on every authenticated
/// /v1/* call, so the handoff Dio bakes both Authorization and
/// X-Craftsky-Device-Id into BaseOptions. Callers must pre-resolve
/// `deviceIdProvider.future` and pass the value explicitly — this
/// keeps the provider itself synchronous.
@riverpod
HandoffApiClient handoffApiClient(Ref ref, String token, String deviceId) {
  final base = baseDioOptions();
  final dio = Dio(
    base.copyWith(
      headers: {
        ...base.headers,
        'Authorization': 'Bearer $token',
        'X-Craftsky-Device-Id': deviceId,
      },
    ),
  )..interceptors.add(const ErrorMappingInterceptor());
  return HandoffApiClient(dio);
}
