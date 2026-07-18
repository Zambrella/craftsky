import 'package:craftsky_app/auth/data/handoff_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'handoff_api_client_provider.g.dart';

@immutable
final class HandoffClientKey {
  const HandoffClientKey({required this.token, required this.deviceId});

  final String token;
  final String deviceId;

  @override
  bool operator ==(Object other) =>
      other is HandoffClientKey &&
      other.token == token &&
      other.deviceId == deviceId;

  @override
  int get hashCode => Object.hash(token, deviceId);

  @override
  String toString() => 'HandoffClientKey(<redacted>)';
}

/// Family-keyed by a redacted credential wrapper: one instance per in-flight
/// handoff. Not keep-alive — auto-disposes when no one watches it, so
/// the token doesn't linger.
///
/// The server enforces X-Craftsky-Device-Id on every authenticated
/// /v1/* call, so the handoff Dio bakes both Authorization and
/// X-Craftsky-Device-Id into BaseOptions. Callers must pre-resolve
/// `deviceIdProvider.future` and pass the value explicitly — this
/// keeps the provider itself synchronous.
@riverpod
HandoffApiClient handoffApiClient(Ref ref, HandoffClientKey key) {
  final base = baseDioOptions();
  final dio = Dio(
    base.copyWith(
      headers: {
        ...base.headers,
        'Authorization': 'Bearer ${key.token}',
        'X-Craftsky-Device-Id': key.deviceId,
      },
    ),
  )..interceptors.add(const ErrorMappingInterceptor());
  return HandoffApiClient(dio);
}
