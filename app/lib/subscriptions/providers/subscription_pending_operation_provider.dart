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
    required this.operationId,
    this.beforeProvider,
    this.tier,
  });

  const new purchaseInFlight(
    Did ownerDid,
    SubscriptionTier tier,
    Object operationId,
  ) : this._(
        kind: PendingSubscriptionOperationKind.purchase,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
        operationId: operationId,
        tier: tier,
      );

  const new purchasePending(
    Did ownerDid,
    SubscriptionTier tier,
    BillingState beforeProvider,
    Object operationId,
  ) : this._(
        kind: PendingSubscriptionOperationKind.purchase,
        phase: PendingSubscriptionOperationPhase.reconciliationPending,
        ownerDid: ownerDid,
        operationId: operationId,
        tier: tier,
        beforeProvider: beforeProvider,
      );

  const new restoreInFlight(Did ownerDid, Object operationId)
    : this._(
        kind: PendingSubscriptionOperationKind.restore,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
        operationId: operationId,
      );

  const new restorePending(
    Did ownerDid,
    BillingState beforeProvider,
    Object operationId,
  ) : this._(
        kind: PendingSubscriptionOperationKind.restore,
        phase: PendingSubscriptionOperationPhase.reconciliationPending,
        ownerDid: ownerDid,
        operationId: operationId,
        beforeProvider: beforeProvider,
      );

  const new customerCenterInFlight(Did ownerDid, Object operationId)
    : this._(
        kind: PendingSubscriptionOperationKind.customerCenter,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
        operationId: operationId,
      );

  const new customerCenterPending(Did ownerDid, Object operationId)
    : this._(
        kind: PendingSubscriptionOperationKind.customerCenter,
        phase: PendingSubscriptionOperationPhase.reconciliationPending,
        ownerDid: ownerDid,
        operationId: operationId,
      );

  final PendingSubscriptionOperationKind kind;
  final PendingSubscriptionOperationPhase phase;
  final Did ownerDid;
  final Object operationId;
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
    state = PendingSubscriptionOperation.purchaseInFlight(
      ownerDid,
      tier,
      Object(),
    );
    return true;
  }

  bool beginRestore(Did ownerDid) {
    if (state != null) return false;
    state = PendingSubscriptionOperation.restoreInFlight(ownerDid, Object());
    return true;
  }

  bool beginCustomerCenter(Did ownerDid) {
    if (state != null) return false;
    state = PendingSubscriptionOperation.customerCenterInFlight(
      ownerDid,
      Object(),
    );
    return true;
  }

  void recordPurchase(
    Object operationId,
    SubscriptionTier tier,
    BillingState beforeProvider,
  ) {
    final current = state;
    if (current == null ||
        !identical(current.operationId, operationId) ||
        current.kind != PendingSubscriptionOperationKind.purchase ||
        current.phase != PendingSubscriptionOperationPhase.nativeInFlight ||
        current.tier != tier) {
      return;
    }
    state = PendingSubscriptionOperation.purchasePending(
      current.ownerDid,
      tier,
      beforeProvider,
      operationId,
    );
  }

  void recordRestore(Object operationId, BillingState beforeProvider) {
    final current = state;
    if (current == null ||
        !identical(current.operationId, operationId) ||
        current.kind != PendingSubscriptionOperationKind.restore ||
        current.phase != PendingSubscriptionOperationPhase.nativeInFlight) {
      return;
    }
    state = PendingSubscriptionOperation.restorePending(
      current.ownerDid,
      beforeProvider,
      operationId,
    );
  }

  void recordPaywallRestore(Object operationId, BillingState beforeProvider) {
    final current = state;
    if (current == null ||
        !identical(current.operationId, operationId) ||
        current.kind != PendingSubscriptionOperationKind.purchase ||
        current.phase != PendingSubscriptionOperationPhase.nativeInFlight) {
      return;
    }
    state = PendingSubscriptionOperation.restorePending(
      current.ownerDid,
      beforeProvider,
      operationId,
    );
  }

  void recordCustomerCenterMutation(Object operationId) {
    final current = state;
    if (current == null ||
        !identical(current.operationId, operationId) ||
        current.kind != PendingSubscriptionOperationKind.customerCenter ||
        current.phase != PendingSubscriptionOperationPhase.nativeInFlight) {
      return;
    }
    state = PendingSubscriptionOperation.customerCenterPending(
      current.ownerDid,
      operationId,
    );
  }

  void clear(Object operationId) {
    if (identical(state?.operationId, operationId)) state = null;
  }

  void clearUnconfirmed(Object operationId) {
    if (state?.phase == PendingSubscriptionOperationPhase.nativeInFlight) {
      clear(operationId);
    }
  }
}
