import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';

enum DeletionAcceptanceResult { activeRemoved, inactiveRemoved, stale }

/// Applies the irreversible client-side boundary only after AppView has
/// durably accepted deletion. There is no client-side job/status state.
final class AccountDeletionAcceptanceCoordinator {
  const AccountDeletionAcceptanceCoordinator({
    required this.readRegistry,
    required this.invalidateActiveState,
    required this.cleanProductData,
    required this.removeOrdinarySession,
    required this.routeAfterActiveRemoval,
  });

  final Future<SessionRegistry> Function() readRegistry;
  final Future<void> Function() invalidateActiveState;
  final Future<void> Function(AccountSessionLease lease) cleanProductData;
  final Future<void> Function(AccountSessionLease lease) removeOrdinarySession;
  final Future<void> Function({required bool hasFallback})
  routeAfterActiveRemoval;

  Future<DeletionAcceptanceResult> reconcile({
    required AccountDeletionLeaseFence fence,
  }) async {
    final before = await readRegistry();
    final deletingLease = fence.lease.session;
    if (before.leaseFor(deletingLease.account) != deletingLease) {
      return DeletionAcceptanceResult.stale;
    }

    final removesActive = fence.isCurrent(before);
    if (removesActive) await invalidateActiveState();
    Object? cleanupError;
    StackTrace? cleanupStackTrace;
    try {
      await cleanProductData(deletingLease);
    } on Object catch (error, stackTrace) {
      cleanupError = error;
      cleanupStackTrace = stackTrace;
    }
    await removeOrdinarySession(deletingLease);

    if (!removesActive) {
      if (cleanupError != null) {
        Error.throwWithStackTrace(cleanupError, cleanupStackTrace!);
      }
      return DeletionAcceptanceResult.inactiveRemoved;
    }
    final after = await readRegistry();
    await routeAfterActiveRemoval(hasFallback: after.activeDid != null);
    if (cleanupError != null) {
      Error.throwWithStackTrace(cleanupError, cleanupStackTrace!);
    }
    return DeletionAcceptanceResult.activeRemoved;
  }
}
