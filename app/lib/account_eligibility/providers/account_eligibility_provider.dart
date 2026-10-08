import 'package:craftsky_app/account_eligibility/data/account_eligibility_repository.dart';
import 'package:craftsky_app/account_eligibility/data/api_account_eligibility_repository.dart';
import 'package:craftsky_app/account_eligibility/models/account_eligibility_status.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'account_eligibility_provider.g.dart';

@riverpod
Future<AccountEligibilityRepository> accountEligibilityRepository(
  Ref ref,
  ActiveAccountLease lease,
) async {
  final dio = await ref.watch(accountDioProvider(lease.session.account).future);
  return ApiAccountEligibilityRepository(dio);
}

@riverpod
Future<AccountEligibilityStatus> accountEligibility(
  Ref ref,
  ActiveAccountLease lease,
) async {
  final registry = await ref.watch(sessionRegistryProvider.future);
  if (registry.activeLease != lease) {
    throw StateError('Active account changed');
  }
  final repository = await ref.watch(
    accountEligibilityRepositoryProvider(lease).future,
  );
  final status = await repository.readStatus();
  if (ref.read(sessionRegistryProvider).value?.activeLease != lease) {
    throw StateError('Active account changed');
  }
  return status;
}
