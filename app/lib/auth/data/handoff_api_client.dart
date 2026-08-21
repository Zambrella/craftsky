import 'package:craftsky_app/auth/models/pending_handoff.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

/// Credential-free OAuth handoff client.
///
/// Exchange receives a pending bearer in its response. The bearer is supplied
/// only to the direct confirmation request and is never embedded in a provider
/// key, URL, or diagnostic representation.
class HandoffApiClient {
  const HandoffApiClient(this._dio);
  final Dio _dio;

  Future<PendingHandoff> exchange({required String code}) =>
      unwrapApi(() async {
        final response = await _dio.post<Map<String, dynamic>>(
          '/v1/auth/handoffs/exchange',
          data: {'code': code},
        );
        return PendingHandoff.fromMap(response.data!);
      });

  Future<void> confirm({
    required String token,
    required String receiptId,
  }) => unwrapApi(() async {
    await _dio.post<void>(
      '/v1/auth/handoffs/confirm',
      data: {'receiptId': receiptId},
      options: Options(headers: {'Authorization': 'Bearer $token'}),
    );
  });
}
