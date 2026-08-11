import 'package:craftsky_app/auth/data/account_deletion_repository.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

@immutable
final class DeletionStatusClientKey {
  const DeletionStatusClientKey({
    required this.jobId,
    required this.statusToken,
    required this.deviceId,
  });

  final String jobId;
  final String statusToken;
  final String deviceId;

  @override
  bool operator ==(Object other) =>
      other is DeletionStatusClientKey &&
      other.jobId == jobId &&
      other.statusToken == statusToken &&
      other.deviceId == deviceId;

  @override
  int get hashCode => Object.hash(jobId, statusToken, deviceId);

  @override
  String toString() => 'DeletionStatusClientKey(<redacted>)';
}

// The builder's concrete family type is intentionally inferred by Riverpod.
// ignore: specify_nonobvious_property_types
final deletionStatusApiClientProvider = Provider.autoDispose
    .family<DeletionStatusApiClient, DeletionStatusClientKey>((ref, key) {
      final base = baseDioOptions();
      final dio = Dio(
        base.copyWith(
          headers: {
            ...base.headers,
            'Authorization': 'DeletionStatus ${key.statusToken}',
            'X-Craftsky-Device-Id': key.deviceId,
          },
        ),
      )..interceptors.add(const ErrorMappingInterceptor());
      ref.onDispose(() => dio.close(force: true));
      return DeletionStatusApiClient(dio);
    });

@immutable
final class AccountDeletionRepositoryKey {
  const AccountDeletionRepositoryKey({
    required this.account,
    required this.status,
  });

  final AccountKey account;
  final DeletionStatusClientKey status;

  @override
  bool operator ==(Object other) =>
      other is AccountDeletionRepositoryKey &&
      other.account == account &&
      other.status == status;

  @override
  int get hashCode => Object.hash(account, status);

  @override
  String toString() => 'AccountDeletionRepositoryKey(<redacted>)';
}

// The builder's concrete family type is intentionally inferred by Riverpod.
// ignore: specify_nonobvious_property_types
final accountDeletionRepositoryProvider = FutureProvider.autoDispose
    .family<AccountDeletionRepository, AccountDeletionRepositoryKey>(
      (ref, key) async => AccountDeletionRepository(
        ordinaryClient: AccountDeletionApiClient(
          await ref.watch(accountDioProvider(key.account).future),
        ),
        statusClient: ref.watch(deletionStatusApiClientProvider(key.status)),
      ),
    );
