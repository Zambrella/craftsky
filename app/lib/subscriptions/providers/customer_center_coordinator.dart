import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/reconciliation_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';

enum CustomerCenterOutcome {
  completed,
  refreshed,
  providerError,
  pending,
  anomaly,
  failed,
  cancelled,
}

final class CustomerCenterResult {
  const CustomerCenterResult({required this.outcome, this.state, this.retry});

  final CustomerCenterOutcome outcome;
  final BillingState? state;
  final Future<CustomerCenterResult> Function()? retry;
}

final class CustomerCenterCoordinator {
  const CustomerCenterCoordinator({
    required this.ownerGuard,
    required this.api,
    required this.revenueCat,
    required this.wait,
    this.isCurrent = _alwaysCurrent,
    this.isForeground = _alwaysCurrent,
    this.onProviderMutationConfirmed,
    this.maxReconciliationAttempts = 30,
  });

  final BillingOwnerGuard ownerGuard;
  final SubscriptionApi api;
  final RevenueCatService revenueCat;
  final ReconciliationWait wait;
  final bool Function() isCurrent;
  final bool Function() isForeground;
  final void Function()? onProviderMutationConfirmed;
  final int maxReconciliationAttempts;

  Future<CustomerCenterResult> present() async {
    final owner = ownerGuard.capture();
    try {
      await ownerGuard.dispatchBillingState(owner, api.getBillingAccount);
    } on BillingOwnerGuardException catch (error) {
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
        () => revenueCat.presentCustomerCenter((_) {
          if (providerMutationPossible) return;
          providerMutationPossible = true;
          onProviderMutationConfirmed?.call();
        }),
      );
    } on BillingOwnerGuardException catch (error) {
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
    return providerMutationPossible ? _reconcile(owner) : _refreshOwner(owner);
  }

  Future<CustomerCenterResult> _refreshOwner(BillingOwnerLease owner) async {
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
    } on BillingOwnerGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return const CustomerCenterResult(
        outcome: CustomerCenterOutcome.cancelled,
      );
    } on Object {
      return _pending(() => _refreshOwner(owner));
    }
  }

  Future<CustomerCenterResult> _reconcile(BillingOwnerLease owner) async {
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
    } on BillingOwnerGuardException catch (error) {
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
    late final BillingOwnerLease owner;
    try {
      owner = ownerGuard.capture();
    } on BillingOwnerGuardException {
      return _pending(reconcilePending);
    }
    return _reconcile(owner);
  }

  CustomerCenterResult _pending(
    Future<CustomerCenterResult> Function() retry,
  ) => CustomerCenterResult(
    outcome: CustomerCenterOutcome.pending,
    retry: retry,
  );
}

bool _alwaysCurrent() => true;
