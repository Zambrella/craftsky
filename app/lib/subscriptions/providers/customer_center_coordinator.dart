import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_guard.dart';
import 'package:craftsky_app/subscriptions/services/reconciliation_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';

enum CustomerCenterOutcome {
  completed,
  refreshed,
  providerError,
  storeAccountConflict,
  pending,
  anomaly,
  failed,
  cancelled,
}

final class CustomerCenterResult {
  const new({required this.outcome, this.state, this.retry});

  final CustomerCenterOutcome outcome;
  final BillingState? state;
  final Future<CustomerCenterResult> Function()? retry;
}

final class CustomerCenterCoordinator {
  const new({
    required this.ownerGuard,
    required this.api,
    required this.revenueCat,
    required this.wait,
    this.isCurrent = _alwaysCurrent,
    this.isForeground = _alwaysCurrent,
    this.onProviderMutationConfirmed,
    this.onStoreConflict,
    this.maxReconciliationAttempts = 30,
  });

  final BillingAccountGuard ownerGuard;
  final SubscriptionApi api;
  final RevenueCatService revenueCat;
  final ReconciliationWait wait;
  final bool Function() isCurrent;
  final bool Function() isForeground;
  final void Function()? onProviderMutationConfirmed;
  final int maxReconciliationAttempts;
  final void Function(StoreAccountConflictException)? onStoreConflict;

  Future<CustomerCenterResult> present() async {
    final owner = ownerGuard.capture();
    try {
      await ownerGuard.dispatchBillingState(owner, api.getBillingAccount);
    } on BillingAccountGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return const CustomerCenterResult(
        outcome: CustomerCenterOutcome.cancelled,
      );
    } on Object {
      return const CustomerCenterResult(
        outcome: CustomerCenterOutcome.providerError,
      );
    }
    if (!isCurrent()) {
      return const CustomerCenterResult(
        outcome: CustomerCenterOutcome.cancelled,
      );
    }
    var providerMutationPossible = false;
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
          await service.presentCustomerCenter((_) {
            if (providerMutationPossible) return;
            providerMutationPossible = true;
            onProviderMutationConfirmed?.call();
          });
        },
      );
    } on StoreAccountConflictException catch (error) {
      if (!ownerGuard.isCurrent(owner)) {
        return const CustomerCenterResult(
          outcome: CustomerCenterOutcome.cancelled,
        );
      }
      onStoreConflict?.call(error);
      return const CustomerCenterResult(
        outcome: CustomerCenterOutcome.storeAccountConflict,
      );
    } on BillingAccountGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      if (providerMutationPossible) {
        return _pending(reconcilePending);
      }
      return const CustomerCenterResult(
        outcome: CustomerCenterOutcome.cancelled,
      );
    } on Object {
      if (providerMutationPossible) {
        return _pending(reconcilePending);
      }
      return const CustomerCenterResult(
        outcome: CustomerCenterOutcome.providerError,
      );
    }
    if (!isCurrent() || !ownerGuard.isCurrent(owner)) {
      if (providerMutationPossible) return _pending(reconcilePending);
      return const CustomerCenterResult(
        outcome: CustomerCenterOutcome.cancelled,
      );
    }
    return providerMutationPossible
        ? await _reconcile(owner)
        : await _refreshOwner(owner);
  }

  Future<CustomerCenterResult> _refreshOwner(BillingAccountLease owner) async {
    if (!isCurrent() || !ownerGuard.isCurrent(owner)) {
      return const CustomerCenterResult(
        outcome: CustomerCenterOutcome.cancelled,
      );
    }
    if (!isForeground()) return _pending(() => _refreshOwner(owner));
    try {
      final state = await ownerGuard.dispatchBillingState(
        owner,
        api.getBillingAccount,
      );
      return CustomerCenterResult(
        outcome: CustomerCenterOutcome.refreshed,
        state: state,
      );
    } on BillingAccountGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return const CustomerCenterResult(
        outcome: CustomerCenterOutcome.cancelled,
      );
    } on Object {
      return _pending(() => _refreshOwner(owner));
    }
  }

  Future<CustomerCenterResult> _reconcile(BillingAccountLease owner) async {
    if (!isCurrent() || !ownerGuard.isCurrent(owner)) {
      return _pending(reconcilePending);
    }
    if (!isForeground()) return _pending(reconcilePending);
    late final BillingState baseline;
    try {
      baseline = await ownerGuard.dispatchBillingState(
        owner,
        api.getBillingAccount,
      );
    } on BillingAccountGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return _pending(reconcilePending);
    } on Object {
      return _pending(reconcilePending);
    }
    final result = await ReconciliationCoordinator(
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
    ).reconcile(baseline: baseline, intent: ReconciliationIntent.restore);

    return switch (result.outcome) {
      ReconciliationOutcome.completed => CustomerCenterResult(
        outcome: CustomerCenterOutcome.completed,
        state: result.state,
      ),
      ReconciliationOutcome.timedOut => _pending(reconcilePending),
      ReconciliationOutcome.failed =>
        result.state == null
            ? _pending(reconcilePending)
            : CustomerCenterResult(
                outcome: CustomerCenterOutcome.anomaly,
                state: result.state,
              ),
      ReconciliationOutcome.cancelled => _pending(reconcilePending),
    };
  }

  Future<CustomerCenterResult> reconcilePending() async {
    if (!isCurrent()) return _pending(reconcilePending);
    late final BillingAccountLease owner;
    try {
      owner = ownerGuard.capture();
    } on BillingAccountGuardException {
      return _pending(reconcilePending);
    }
    return await _reconcile(owner);
  }

  CustomerCenterResult _pending(
    Future<CustomerCenterResult> Function() retry,
  ) => CustomerCenterResult(
    outcome: CustomerCenterOutcome.pending,
    retry: retry,
  );
}

bool _alwaysCurrent() => true;
