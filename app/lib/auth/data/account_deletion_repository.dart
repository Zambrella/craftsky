import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

final class AccountDeletionIntent {
  const AccountDeletionIntent({
    required this.jobId,
    required this.statusToken,
    required this.authUrl,
    required this.expiresAt,
  });

  factory AccountDeletionIntent.fromMap(Map<String, dynamic> map) =>
      AccountDeletionIntent(
        jobId: _requiredString(map, 'jobId'),
        statusToken: _requiredString(map, 'statusToken'),
        authUrl: Uri.parse(_requiredString(map, 'authUrl')),
        expiresAt: DateTime.parse(_requiredString(map, 'expiresAt')).toUtc(),
      );

  final String jobId;
  final String statusToken;
  final Uri authUrl;
  final DateTime expiresAt;

  @override
  String toString() => 'AccountDeletionIntent(<redacted>)';
}

final class AccountDeletionStatusSnapshot {
  const AccountDeletionStatusSnapshot({
    required this.jobId,
    required this.status,
    required this.phase,
    required this.canRetry,
    required this.needsReauthentication,
  });

  factory AccountDeletionStatusSnapshot.fromMap(Map<String, dynamic> map) {
    final status = AccountDeletionStatus.values.byName(
      _requiredString(map, 'status'),
    );
    final phase = switch (_requiredString(map, 'phase', allowEmpty: true)) {
      '' || 'queued' => AccountDeletionPhase.preparing,
      'removingPrivateData' => AccountDeletionPhase.removingPrivateData,
      'removingCraftskyRecords' => AccountDeletionPhase.removingCraftskyRecords,
      'waitingForIndexerConvergence' => AccountDeletionPhase.waitingForCraftsky,
      'finalizing' => AccountDeletionPhase.finalizing,
      final value => throw FormatException('Invalid phase: $value'),
    };
    return AccountDeletionStatusSnapshot(
      jobId: _requiredString(map, 'jobId'),
      status: status,
      phase: status == AccountDeletionStatus.deleted
          ? AccountDeletionPhase.deleted
          : phase,
      canRetry: _optionalBool(map, 'retryAllowed'),
      needsReauthentication: _optionalBool(
        map,
        'needsReauthentication',
      ),
    );
  }

  final String jobId;
  final AccountDeletionStatus status;
  final AccountDeletionPhase phase;
  final bool canRetry;
  final bool needsReauthentication;

  @override
  String toString() => 'AccountDeletionStatusSnapshot(<redacted>)';
}

final class AccountDeletionReauthentication {
  const AccountDeletionReauthentication({
    required this.authUrl,
    required this.expiresAt,
  });

  factory AccountDeletionReauthentication.fromMap(Map<String, dynamic> map) =>
      AccountDeletionReauthentication(
        authUrl: Uri.parse(_requiredString(map, 'authUrl')),
        expiresAt: DateTime.parse(_requiredString(map, 'expiresAt')).toUtc(),
      );

  final Uri authUrl;
  final DateTime expiresAt;
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

  Future<AccountDeletionStatusSnapshot> accept({
    required String jobId,
    required String statusToken,
    required String reauthProof,
    required String confirmationHandle,
  }) => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/v1/account-deletions/$jobId',
      data: {
        'reauthProof': reauthProof,
        'confirmationHandle': confirmationHandle,
      },
      options: Options(
        headers: {'X-Craftsky-Deletion-Status': statusToken},
      ),
    );
    return AccountDeletionStatusSnapshot.fromMap(response.data!);
  });

  Future<void> cancelIntent({
    required String jobId,
    required String statusToken,
  }) => unwrapApi(() async {
    await _dio.delete<void>(
      '/v1/account-deletion/intents/$jobId',
      options: Options(
        headers: {'X-Craftsky-Deletion-Status': statusToken},
      ),
    );
  });
}

final class AccountDeletionRecoveryResult {
  const AccountDeletionRecoveryResult({
    required this.jobId,
    required this.statusToken,
    required this.snapshot,
  });

  factory AccountDeletionRecoveryResult.fromMap(Map<String, dynamic> map) {
    final snapshot = AccountDeletionStatusSnapshot.fromMap(map);
    return AccountDeletionRecoveryResult(
      jobId: snapshot.jobId,
      statusToken: _requiredString(map, 'statusToken'),
      snapshot: snapshot,
    );
  }

  final String jobId;
  final String statusToken;
  final AccountDeletionStatusSnapshot snapshot;
}

final class AccountDeletionRecoveryClient {
  const AccountDeletionRecoveryClient(this._dio);

  final Dio _dio;

  Future<AccountDeletionRecoveryResult> recover() => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/v1/account-deletions/recover',
    );
    return AccountDeletionRecoveryResult.fromMap(response.data!);
  });
}

/// A status-only client. Its Dio must carry `DeletionStatus`, never an
/// ordinary CraftSky bearer or an auth-state interceptor.
final class DeletionStatusApiClient {
  const DeletionStatusApiClient(this._dio);

  final Dio _dio;

  Future<AccountDeletionStatusSnapshot> getStatus(String jobId) =>
      unwrapApi(() async {
        final response = await _dio.get<Map<String, dynamic>>(
          '/v1/account-deletions/$jobId',
        );
        return AccountDeletionStatusSnapshot.fromMap(response.data!);
      });

  Future<AccountDeletionStatusSnapshot> retry(String jobId) =>
      unwrapApi(() async {
        final response = await _dio.post<Map<String, dynamic>>(
          '/v1/account-deletions/$jobId/retry',
        );
        return AccountDeletionStatusSnapshot.fromMap(response.data!);
      });

  Future<AccountDeletionReauthentication> startReauthentication(
    String jobId,
  ) => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/v1/account-deletions/$jobId/reauth',
    );
    return AccountDeletionReauthentication.fromMap(response.data!);
  });
}

final class AccountDeletionRepository {
  const AccountDeletionRepository({
    required this.ordinaryClient,
    required this.statusClient,
  });

  final AccountDeletionApiClient ordinaryClient;
  final DeletionStatusApiClient statusClient;

  Future<AccountDeletionStatusSnapshot> acceptOrResolve({
    required String jobId,
    required String statusToken,
    required String reauthProof,
    required String confirmationHandle,
  }) async {
    try {
      return await ordinaryClient.accept(
        jobId: jobId,
        statusToken: statusToken,
        reauthProof: reauthProof,
        confirmationHandle: confirmationHandle,
      );
    } on ApiNetworkError {
      return statusClient.getStatus(jobId);
    } on ApiServerError {
      return statusClient.getStatus(jobId);
    }
  }
}

String _requiredString(
  Map<String, dynamic> map,
  String key, {
  bool allowEmpty = false,
}) {
  final value = map[key];
  if (value is! String || (!allowEmpty && value.isEmpty)) {
    throw FormatException('Invalid $key');
  }
  return value;
}

bool _optionalBool(Map<String, dynamic> map, String key) {
  final value = map[key];
  if (value == null) return false;
  if (value is! bool) throw FormatException('Invalid $key');
  return value;
}
