import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Captures ownership for an active-account operation before its first await.
ActiveAccountLease? captureActiveAccountOperation(Ref ref) =>
    ref.read(sessionRegistryProvider).value?.activeLease;

/// Rejects a completion when account activation changed while it was pending.
///
/// A null ownership preserves isolated provider tests that do not construct the
/// authenticated registry. Production authenticated operations always capture
/// a non-null active lease.
bool isActiveAccountOperationCurrent(
  Ref ref,
  ActiveAccountLease? ownership,
) {
  if (!ref.mounted) return false;
  if (ownership == null) return true;
  return ref.read(sessionRegistryProvider).value?.isCurrent(ownership) ?? false;
}
