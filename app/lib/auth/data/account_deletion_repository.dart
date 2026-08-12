import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

final class AccountDeletionIntent {
  const AccountDeletionIntent({
    required this.jobId,
    required this.authUrl,
    required this.expiresAt,
  });

  factory AccountDeletionIntent.fromMap(Map<String, dynamic> map) =>
      AccountDeletionIntent(
        jobId: _requiredString(map, 'jobId'),
        authUrl: Uri.parse(_requiredString(map, 'authUrl')),
        expiresAt: DateTime.parse(_requiredString(map, 'expiresAt')).toUtc(),
      );

  final String jobId;
  final Uri authUrl;
  final DateTime expiresAt;

  @override
  String toString() => 'AccountDeletionIntent(<redacted>)';
}

final class AccountDeletionApiClient {
  const AccountDeletionApiClient(this._dio);

  final Dio _dio;

  Future<AccountDeletionIntent> createIntent() => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/v1/account-deletion/intents',
    );
    return AccountDeletionIntent.fromMap(response.data!);
  });

  Future<void> accept({
    required String jobId,
    required String reauthProof,
    required String confirmationHandle,
  }) => unwrapApi(() async {
    await _dio.post<void>(
      '/v1/account-deletions/$jobId',
      data: {
        'reauthProof': reauthProof,
        'confirmationHandle': confirmationHandle,
      },
    );
  });

  Future<void> cancelIntent({required String jobId}) => unwrapApi(() async {
    await _dio.delete<void>('/v1/account-deletion/intents/$jobId');
  });
}

String _requiredString(Map<String, dynamic> map, String key) {
  final value = map[key];
  if (value is! String || value.isEmpty) {
    throw FormatException('Invalid $key');
  }
  return value;
}
