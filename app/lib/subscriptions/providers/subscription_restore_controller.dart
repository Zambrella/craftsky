import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_guard.dart';
import 'package:craftsky_app/subscriptions/services/reconciliation_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';

enum SubscriptionRestoreOutcome {
  completed,
  providerError,
  storeAccountConflict,
  pending,
  anomaly,
  failed,
  cancelled,
}

final class SubscriptionRestoreResult {
  const new({
    required this.outcome,
    this.state,
    this.retry,
  });

  final SubscriptionRestoreOutcome outcome;
  final BillingState? state;
  final Future<SubscriptionRestoreResult> Function()? retry;
}

final class SubscriptionRestoreController {
  const new({
    required this.ownerGuard,
    required this.api,
    required this.revenueCat,
    required this.wait,
    this.isCurrent = _alwaysCurrent,
    this.isForeground = _alwaysCurrent,
    this.onProviderConfirmed,
    this.onStoreConflict,
    this.maxReconciliationAttempts = 30,
  });

  final BillingAccountGuard ownerGuard;
  final SubscriptionApi api;
  final RevenueCatService revenueCat;
  final ReconciliationWait wait;
  final bool Function() isCurrent;
  final bool Function() isForeground;
  final void Function(BillingState beforeProvider)? onProviderConfirmed;
  final int maxReconciliationAttempts;
  final void Function(StoreAccountConflictException)? onStoreConflict;

  Future<SubscriptionRestoreResult> restore() async {
    final owner = ownerGuard.capture();
    late final BillingState beforeProvider;
    try {
      beforeProvider = await ownerGuard.dispatchBillingState(
        owner,
        api.getBillingAccount,
      );
    } on BillingAccountGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.cancelled,
      );
    }
    if (!isCurrent()) {
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.cancelled,
      );
    }
    var providerConfirmed = false;
    try {
      await ownerGuard.dispatchRevenueCat(
        owner,
        RevenueCatIdentityGuard(revenueCat),
        () async {
          final service = revenueCat;
          if (service is NativeStoreOwnershipService) {
            await (service as NativeStoreOwnershipService).verifyStoreOwnership(
              owner.revenueCatAppUserId,
            );
          }
          if (!ownerGuard.isCurrent(owner)) {
            throw const BillingAccountGuardException('owner_session_changed');
          }
          await service.restorePurchases();
          providerConfirmed = true;
          onProviderConfirmed?.call(beforeProvider);
        },
      );
      if (!isCurrent() || !ownerGuard.isCurrent(owner)) {
        return _pendingRetry(beforeProvider);
      }
    } on StoreAccountConflictException catch (error) {
      if (!ownerGuard.isCurrent(owner)) {
        return const SubscriptionRestoreResult(
          outcome: SubscriptionRestoreOutcome.cancelled,
        );
      }
      onStoreConflict?.call(error);
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.storeAccountConflict,
      );
    } on RevenueCatIdentityException {
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.providerError,
      );
    } on RevenueCatUnavailableException {
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.providerError,
      );
    } on BillingAccountGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      if (providerConfirmed) return _pendingRetry(beforeProvider);
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.cancelled,
      );
    } on Object {
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.providerError,
      );
    }
    if (!isForeground()) return _pendingRetry(beforeProvider);

    return await _reconcileRestore(owner, beforeProvider);
  }

  Future<SubscriptionRestoreResult> _reconcileRestore(
    BillingAccountLease owner,
    BillingState beforeProvider,
  ) async {
    if (!isCurrent() || !ownerGuard.isCurrent(owner)) {
      return _pendingRetry(beforeProvider);
    }
    if (!isForeground()) return _pendingRetry(beforeProvider);
    late final BillingState generationBaseline;
    try {
      generationBaseline = await ownerGuard.dispatchBillingState(
        owner,
        api.getBillingAccount,
      );
    } on BillingAccountGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return _pendingRetry(beforeProvider);
    } on Object {
      return _pendingRetry(beforeProvider);
    }
    final result =
        await ReconciliationCoordinator(
          requestReconciliation: () => ownerGuard.dispatch(
            owner,
            api.requestReconciliation,
          ),
          readBillingState: () => ownerGuard.dispatchBillingState(
            owner,
            api.getBillingAccount,
          ),
          wait: wait,
          isCurrent: () =>
              isCurrent() && isForeground() && ownerGuard.isCurrent(owner),
          maxAttempts: maxReconciliationAttempts,
        ).reconcile(
          baseline: generationBaseline,
          intent: ReconciliationIntent.restore,
        );

    return switch (result.outcome) {
      ReconciliationOutcome.completed => SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.completed,
        state: result.state,
      ),
      ReconciliationOutcome.timedOut => _pendingRetry(beforeProvider),
      ReconciliationOutcome.failed =>
        result.state == null
            ? _pendingRetry(beforeProvider)
            : SubscriptionRestoreResult(
                outcome: SubscriptionRestoreOutcome.anomaly,
                state: result.state,
              ),
      ReconciliationOutcome.cancelled => _pendingRetry(beforeProvider),
    };
  }

  Future<SubscriptionRestoreResult> reconcilePending(
    BillingState beforeProvider,
  ) async {
    if (!isCurrent()) return _pendingRetry(beforeProvider);
    late final BillingAccountLease owner;
    try {
      owner = ownerGuard.capture();
    } on BillingAccountGuardException {
      return _pendingRetry(beforeProvider);
    }
    return await _reconcileRestore(owner, beforeProvider);
  }

  SubscriptionRestoreResult _pendingRetry(
    BillingState beforeProvider,
  ) => SubscriptionRestoreResult(
    outcome: SubscriptionRestoreOutcome.pending,
    retry: () => reconcilePending(beforeProvider),
  );
}

bool _alwaysCurrent() => true;
