import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Internal control flow for work whose original account authority has ended.
final class AccountOperationDiscarded implements Exception {
  const AccountOperationDiscarded();
}

/// Captured activation authority shared by dependency acquisition, async steps,
/// retries and publication. Feature callers capture their dependencies;
/// this object never stores credentials or selects another account.
final class ActiveAccountOperation {
  ActiveAccountOperation._(this.ownership, this._isCurrent);

  factory ActiveAccountOperation.capture(Ref ref) =>
      ActiveAccountOperation.fromLease(ref, captureActiveAccountOperation(ref));

  factory ActiveAccountOperation.fromLease(
    Ref ref,
    ActiveAccountLease? ownership,
  ) => ActiveAccountOperation._(
    ownership,
    () => isActiveAccountOperationCurrent(ref, ownership),
  );

  factory ActiveAccountOperation.fromWidget(
    WidgetRef ref,
    ActiveAccountLease? ownership,
  ) => ActiveAccountOperation._(ownership, () {
    if (!ref.context.mounted) return false;
    if (ownership == null) return true;
    return ref.read(sessionRegistryProvider).value?.isCurrent(ownership) ??
        false;
  });

  /// Widgets supply their own lifetime check; pure services can use the same
  /// authority without holding a Riverpod reference.
  factory ActiveAccountOperation.withGuard({
    required ActiveAccountLease? ownership,
    required bool Function() isCurrent,
  }) => ActiveAccountOperation._(ownership, isCurrent);

  final ActiveAccountLease? ownership;
  final bool Function() _isCurrent;

  bool get isCurrent => _isCurrent();

  void requireCurrent({
    Exception discarded = const AccountOperationDiscarded(),
  }) {
    if (!isCurrent) throw discarded;
  }

  /// Guards both acquisition/dispatch and its completion. A failure from the
  /// step retains its original cause and stack for the feature's error owner.
  /// Callers still guard their continuation: an await of this helper is itself
  /// an async boundary. Use publish for synchronous completion side effects.
  Future<T> step<T>(
    Future<T> Function() action, {
    Exception discarded = const AccountOperationDiscarded(),
  }) async {
    requireCurrent(discarded: discarded);
    final result = await action();
    requireCurrent(discarded: discarded);
    return result;
  }

  /// Publish synchronously so no account change can interleave with the check.
  bool publish(void Function() action) {
    if (!isCurrent) return false;
    action();
    return true;
  }
}

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
