import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';

enum DeletionAcceptanceResult { activeRemoved, inactiveRemoved, stale }

/// Applies the irreversible client-side boundary after the server has accepted
/// deletion. The status capability is durably stored first; no ordinary
/// credential or product data is removed if that secure write fails.
final class AccountDeletionAcceptanceCoordinator {
  const AccountDeletionAcceptanceCoordinator({
    required this.readRegistry,
    required this.persistStatus,
    required this.invalidateActiveState,
    required this.cleanProductData,
    required this.removeOrdinarySession,
    required this.routeAfterActiveRemoval,
  });

  final Future<SessionRegistry> Function() readRegistry;
  final Future<void> Function(DeletionStatusEntry entry) persistStatus;
  final Future<void> Function() invalidateActiveState;
  final Future<void> Function(AccountSessionLease lease) cleanProductData;
  final Future<void> Function(AccountSessionLease lease) removeOrdinarySession;
  final Future<void> Function({required bool hasFallback})
  routeAfterActiveRemoval;

  Future<DeletionAcceptanceResult> reconcile({
    required AccountDeletionLeaseFence fence,
    required DeletionStatusEntry entry,
  }) async {
    final before = await readRegistry();
    final deletingLease = fence.lease.session;
    if (before.leaseFor(deletingLease.account) != deletingLease ||
        deletingLease.account.did != entry.did) {
      return DeletionAcceptanceResult.stale;
    }

    await persistStatus(entry);

    final current = await readRegistry();
    final removesActive = fence.isCurrent(current);
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
