import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum PendingSubscriptionOperationKind {
  purchase,
  restore,
  customerCenter,
}

enum PendingSubscriptionOperationPhase { nativeInFlight, reconciliationPending }

final class PendingSubscriptionOperation {
  const new _({
    required this.kind,
    required this.phase,
    required this.ownerDid,
    this.beforeProvider,
    this.tier,
  });

  const new purchaseInFlight(
    Did ownerDid,
    SubscriptionTier tier,
  ) : this._(
        kind: PendingSubscriptionOperationKind.purchase,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
        tier: tier,
      );

  const new purchasePending(
    Did ownerDid,
    SubscriptionTier tier,
    BillingState beforeProvider,
  ) : this._(
        kind: PendingSubscriptionOperationKind.purchase,
        phase: PendingSubscriptionOperationPhase.reconciliationPending,
        ownerDid: ownerDid,
        tier: tier,
        beforeProvider: beforeProvider,
      );

  const new restoreInFlight(Did ownerDid)
    : this._(
        kind: PendingSubscriptionOperationKind.restore,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
      );

  const new restorePending(
    Did ownerDid,
    BillingState beforeProvider,
  ) : this._(
        kind: PendingSubscriptionOperationKind.restore,
        phase: PendingSubscriptionOperationPhase.reconciliationPending,
        ownerDid: ownerDid,
        beforeProvider: beforeProvider,
      );

  const new customerCenterInFlight(Did ownerDid)
    : this._(
        kind: PendingSubscriptionOperationKind.customerCenter,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
      );

  const new customerCenterPending(Did ownerDid)
    : this._(
        kind: PendingSubscriptionOperationKind.customerCenter,
        phase: PendingSubscriptionOperationPhase.reconciliationPending,
        ownerDid: ownerDid,
      );

  final PendingSubscriptionOperationKind kind;
  final PendingSubscriptionOperationPhase phase;
  final Did ownerDid;
  final SubscriptionTier? tier;
  final BillingState? beforeProvider;

  bool get canReconcile =>
      phase == PendingSubscriptionOperationPhase.reconciliationPending;
}

final subscriptionPendingOperationProvider =
    NotifierProvider<
      SubscriptionPendingOperationController,
      PendingSubscriptionOperation?
    >(SubscriptionPendingOperationController.new);

final class SubscriptionPendingOperationController
    extends Notifier<PendingSubscriptionOperation?> {
  @override
  PendingSubscriptionOperation? build() => null;

  bool beginPurchase(Did ownerDid, SubscriptionTier tier) {
    if (state != null) return false;
    state = PendingSubscriptionOperation.purchaseInFlight(ownerDid, tier);
    return true;
  }

  bool beginRestore(Did ownerDid) {
    if (state != null) return false;
    state = PendingSubscriptionOperation.restoreInFlight(ownerDid);
    return true;
  }

  bool beginCustomerCenter(Did ownerDid) {
    if (state != null) return false;
    state = PendingSubscriptionOperation.customerCenterInFlight(ownerDid);
    return true;
  }

  void recordPurchase(SubscriptionTier tier, BillingState beforeProvider) {
    final current = state;
    if (current == null ||
        current.kind != PendingSubscriptionOperationKind.purchase ||
        current.phase != PendingSubscriptionOperationPhase.nativeInFlight ||
        current.tier != tier) {
      return;
    }
    state = PendingSubscriptionOperation.purchasePending(
      current.ownerDid,
      tier,
      beforeProvider,
    );
  }

  void recordRestore(BillingState beforeProvider) {
    final current = state;
    if (current == null ||
        current.kind != PendingSubscriptionOperationKind.restore ||
        current.phase != PendingSubscriptionOperationPhase.nativeInFlight) {
      return;
    }
    state = PendingSubscriptionOperation.restorePending(
      current.ownerDid,
      beforeProvider,
    );
  }

  void recordCustomerCenterMutation() {
    final current = state;
    if (current == null ||
        current.kind != PendingSubscriptionOperationKind.customerCenter ||
        current.phase != PendingSubscriptionOperationPhase.nativeInFlight) {
      return;
    }
    state = PendingSubscriptionOperation.customerCenterPending(
      current.ownerDid,
    );
  }

  void clear() => state = null;
}
