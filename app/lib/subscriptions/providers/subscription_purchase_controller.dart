import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/models/subscription_rules.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/paywall_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/reconciliation_coordinator.dart';

enum SubscriptionPurchaseOutcome {
  completed,
  notEligible,
  cancelled,
  offeringUnavailable,
  providerError,
  pending,
  anomaly,
  failed,
}

final class SubscriptionPurchaseResult {
  const SubscriptionPurchaseResult({
    required this.outcome,
    this.state,
    this.assignmentLicense,
    this.retry,
  });

  final SubscriptionPurchaseOutcome outcome;
  final BillingState? state;
  final BillingLicense? assignmentLicense;
  final Future<SubscriptionPurchaseResult> Function()? retry;

  bool get requiresAssignment => assignmentLicense != null;
}

final class SubscriptionPurchaseController {
  const SubscriptionPurchaseController({
    required this.ownerGuard,
    required this.api,
    required this.presentPaywall,
    required this.wait,
    required this.readAccess,
    this.isCurrent = _alwaysCurrent,
    this.isForeground = _alwaysCurrent,
    this.onProviderConfirmed,
    this.maxReconciliationAttempts = 30,
  });

  final BillingOwnerGuard ownerGuard;
  final SubscriptionApi api;
  final Future<PaywallOutcome> Function(
    BillingOwnerLease owner,
    SubscriptionTier tier,
  )
  presentPaywall;
  final ReconciliationWait wait;
  final Future<SubscriptionAccess> Function(Did did) readAccess;
  final bool Function() isCurrent;
  final bool Function() isForeground;
  final void Function(SubscriptionTier tier, BillingState beforeProvider)?
  onProviderConfirmed;
  final int maxReconciliationAttempts;

  Future<SubscriptionPurchaseResult> purchase(SubscriptionTier tier) async {
    final owner = ownerGuard.capture();
    late final BillingState beforeProvider;
    try {
      beforeProvider = await ownerGuard.dispatchBillingState(
        owner,
        api.getBillingAccount,
      );
    } on BillingOwnerGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return const SubscriptionPurchaseResult(
        outcome: SubscriptionPurchaseOutcome.cancelled,
      );
    }
    if (!isCurrent()) {
      return const SubscriptionPurchaseResult(
        outcome: SubscriptionPurchaseOutcome.cancelled,
      );
    }
    if (!purchaseEligibility(beforeProvider, tier).eligible) {
      return const SubscriptionPurchaseResult(
        outcome: SubscriptionPurchaseOutcome.notEligible,
      );
    }

    final paywallOutcome = await presentPaywall(owner, tier);
    if (!isCurrent() && paywallOutcome != PaywallOutcome.reconcile) {
      return const SubscriptionPurchaseResult(
        outcome: SubscriptionPurchaseOutcome.cancelled,
      );
    }
    switch (paywallOutcome) {
      case PaywallOutcome.cancelled:
        return const SubscriptionPurchaseResult(
          outcome: SubscriptionPurchaseOutcome.cancelled,
        );
      case PaywallOutcome.offeringUnavailable:
        return const SubscriptionPurchaseResult(
          outcome: SubscriptionPurchaseOutcome.offeringUnavailable,
        );
      case PaywallOutcome.providerError:
        return const SubscriptionPurchaseResult(
          outcome: SubscriptionPurchaseOutcome.providerError,
        );
      case PaywallOutcome.reconcile:
        onProviderConfirmed?.call(tier, beforeProvider);
        if (!isCurrent()) return _pendingRetry(tier, beforeProvider);
    }
    if (!isForeground()) return _pendingRetry(tier, beforeProvider);

    return _reconcilePurchase(owner, tier, beforeProvider);
  }

  Future<SubscriptionPurchaseResult> _reconcilePurchase(
    BillingOwnerLease owner,
    SubscriptionTier tier,
    BillingState beforeProvider,
  ) async {
    if (!isCurrent() || !ownerGuard.isCurrent(owner)) {
      return _pendingRetry(tier, beforeProvider);
    }
    if (!isForeground()) return _pendingRetry(tier, beforeProvider);
    late final BillingState generationBaseline;
    try {
      generationBaseline = await ownerGuard.dispatchBillingState(
        owner,
        api.getBillingAccount,
      );
    } on BillingOwnerGuardException catch (error) {
      if (!error.ownerSessionChanged) rethrow;
      return _pendingRetry(tier, beforeProvider);
    } on Object {
      return _pendingRetry(tier, beforeProvider);
    }
    if (!isCurrent()) {
      return _pendingRetry(tier, beforeProvider);
    }
    final reconciliation = ReconciliationCoordinator(
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
    );
    final result = await reconciliation.reconcile(
      baseline: generationBaseline,
      comparisonBaseline: beforeProvider,
      intent: ReconciliationIntent.purchase,
      expectedTier: tier,
    );

    if (result.outcome == ReconciliationOutcome.completed) {
      final state = result.state!;
      final assignmentLicense = _assignmentLicense(tier, beforeProvider, state);
      if (assignmentLicense != null) {
        return SubscriptionPurchaseResult(
          outcome: SubscriptionPurchaseOutcome.completed,
          state: state,
          assignmentLicense: assignmentLicense,
        );
      }
      final assignedLicense = _accessibleAssignedLicense(tier, state);
      final assignedDid = assignedLicense?.assignedDid;
      if (assignedDid != null) {
        try {
          final access = await readAccess(assignedDid);
          if (!isCurrent()) {
            return const SubscriptionPurchaseResult(
              outcome: SubscriptionPurchaseOutcome.cancelled,
            );
          }
          if (!isForeground()) {
            return _pendingRetry(tier, beforeProvider);
          }
          if (access.did != assignedDid ||
              !access.givesAccess ||
              access.effectiveTier != tier) {
            return _pendingRetry(tier, beforeProvider);
          }
        } on Object {
          return _pendingRetry(tier, beforeProvider);
        }
      }
      return SubscriptionPurchaseResult(
        outcome: SubscriptionPurchaseOutcome.completed,
        state: state,
      );
    }

    return switch (result.outcome) {
      ReconciliationOutcome.completed => throw StateError('handled above'),
      ReconciliationOutcome.timedOut => _pendingRetry(tier, beforeProvider),
      ReconciliationOutcome.cancelled => _pendingRetry(tier, beforeProvider),
      ReconciliationOutcome.failed =>
        result.state == null
            ? _pendingRetry(tier, beforeProvider)
            : SubscriptionPurchaseResult(
                outcome: SubscriptionPurchaseOutcome.anomaly,
                state: result.state,
              ),
    };
  }

  Future<SubscriptionPurchaseResult> reconcilePending(
    SubscriptionTier tier,
    BillingState beforeProvider,
  ) async {
    if (!isCurrent()) return _pendingRetry(tier, beforeProvider);
    late final BillingOwnerLease owner;
    try {
      owner = ownerGuard.capture();
    } on BillingOwnerGuardException {
      return _pendingRetry(tier, beforeProvider);
    }
    return _reconcilePurchase(owner, tier, beforeProvider);
  }

  SubscriptionPurchaseResult _pendingRetry(
    SubscriptionTier tier,
    BillingState beforeProvider,
  ) => SubscriptionPurchaseResult(
    outcome: SubscriptionPurchaseOutcome.pending,
    retry: () => reconcilePending(tier, beforeProvider),
  );

  BillingLicense? _accessibleAssignedLicense(
    SubscriptionTier tier,
    BillingState state,
  ) {
    final subscriptions = {
      for (final value in state.subscriptions) value.id: value,
    };
    return state.licenses
        .where(
          (license) =>
              license.tier == tier &&
              license.assignedDid != null &&
              (subscriptions[license.subscriptionId]?.givesAccess ?? false),
        )
        .firstOrNull;
  }

  BillingLicense? _assignmentLicense(
    SubscriptionTier tier,
    BillingState before,
    BillingState current,
  ) {
    final previousLicenseIds = before.licenses.map((value) => value.id).toSet();
    final previousSubscriptions = {
      for (final value in before.subscriptions) value.id: value,
    };
    final currentSubscriptions = {
      for (final value in current.subscriptions) value.id: value,
    };
    return current.licenses.where((license) {
      if (license.tier != tier) return false;
      if (license.assignedDid != null) return false;
      final subscription = currentSubscriptions[license.subscriptionId];
      if (!(subscription?.givesAccess ?? false)) return false;
      if (!previousLicenseIds.contains(license.id)) return true;
      final previous = previousSubscriptions[license.subscriptionId];
      return subscription != null &&
          previous != null &&
          !previous.givesAccess &&
          subscription.givesAccess;
    }).firstOrNull;
  }
}

bool _alwaysCurrent() => true;
