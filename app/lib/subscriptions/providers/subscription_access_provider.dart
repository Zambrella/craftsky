import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'subscription_access_provider.g.dart';

@riverpod
Future<SubscriptionAccess> subscriptionAccess(
  Ref ref,
  AccountSessionLease lease,
) async {
  await _requireLease(ref, lease);
  final repository = await ref.watch(
    subscriptionRepositoryProvider(lease.account).future,
  );
  await _requireLease(ref, lease);
  final access = await repository.getAccess();
  await _requireLease(ref, lease);
  if (access.did != lease.account.did) {
    throw StateError('Account session unavailable');
  }
  return access;
}

Future<void> _requireLease(Ref ref, AccountSessionLease lease) async {
  if (!ref.mounted) {
    throw StateError('Account session unavailable');
  }
  final registry = await ref.watch(sessionRegistryProvider.future);
  if (!ref.mounted || registry.leaseFor(lease.account) != lease) {
    throw StateError('Account session unavailable');
  }
}
