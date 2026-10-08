import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/reconciliation_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';

enum SubscriptionRestoreOutcome {
  completed,
  providerError,
  pending,
  anomaly,
  failed,
  cancelled,
}

final class SubscriptionRestoreResult {
  const SubscriptionRestoreResult({
    required this.outcome,
    this.state,
    this.assignmentLicense,
    this.retry,
  });

  final SubscriptionRestoreOutcome outcome;
  final BillingState? state;
  final BillingLicense? assignmentLicense;
  final Future<SubscriptionRestoreResult> Function()? retry;

  bool get requiresAssignment => assignmentLicense != null;
}

final class SubscriptionRestoreController {
  const SubscriptionRestoreController({
    required this.ownerGuard,
    required this.api,
    required this.revenueCat,
    required this.wait,
    this.isCurrent = _alwaysCurrent,
    this.isForeground = _alwaysCurrent,
    this.onProviderConfirmed,
    this.maxReconciliationAttempts = 30,
  });

  final BillingOwnerGuard ownerGuard;
  final SubscriptionApi api;
  final RevenueCatService revenueCat;
  final ReconciliationWait wait;
  final bool Function() isCurrent;
  final bool Function() isForeground;
  final void Function(BillingState beforeProvider)? onProviderConfirmed;
  final int maxReconciliationAttempts;

  Future<SubscriptionRestoreResult> restore() async {
    final owner = ownerGuard.capture();
    late final BillingState beforeProvider;
    try {
      beforeProvider = await ownerGuard.dispatchBillingState(
        owner,
        api.getBillingAccount,
      );
    } on BillingOwnerGuardException catch (error) {
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
    try {
      await ownerGuard.prepareRevenueCat(
        owner,
        RevenueCatIdentityGuard(revenueCat),
      );
      await revenueCat.restorePurchases();
      onProviderConfirmed?.call(beforeProvider);
      if (!isCurrent() || !ownerGuard.isCurrent(owner)) {
        return _pendingRetry(beforeProvider);
      }
    } on RevenueCatIdentityException {
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.providerError,
      );
    } on RevenueCatUnavailableException {
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.providerError,
      );
    } on BillingOwnerGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.cancelled,
      );
    } on Object {
      return const SubscriptionRestoreResult(
        outcome: SubscriptionRestoreOutcome.providerError,
      );
    }
    if (!isForeground()) return _pendingRetry(beforeProvider);

    return _reconcileRestore(owner, beforeProvider);
  }

  Future<SubscriptionRestoreResult> _reconcileRestore(
    BillingOwnerLease owner,
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
    } on BillingOwnerGuardException catch (error) {
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
        assignmentLicense: _newUnassignedLicense(
          beforeProvider,
          result.state!,
        ),
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
    late final BillingOwnerLease owner;
    try {
      owner = ownerGuard.capture();
    } on BillingOwnerGuardException {
      return _pendingRetry(beforeProvider);
    }
    return _reconcileRestore(owner, beforeProvider);
  }

  SubscriptionRestoreResult _pendingRetry(
    BillingState beforeProvider,
  ) => SubscriptionRestoreResult(
    outcome: SubscriptionRestoreOutcome.pending,
    retry: () => reconcilePending(beforeProvider),
  );

  BillingLicense? _newUnassignedLicense(
    BillingState before,
    BillingState current,
  ) {
    final previousIds = before.licenses.map((value) => value.id).toSet();
    final subscriptions = {
      for (final value in current.subscriptions) value.id: value,
    };
    return current.licenses
        .where(
          (value) =>
              value.assignedDid == null &&
              !previousIds.contains(value.id) &&
              (subscriptions[value.subscriptionId]?.givesAccess ?? false),
        )
        .firstOrNull;
  }
}

bool _alwaysCurrent() => true;
