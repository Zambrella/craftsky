import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_page_model_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum PendingSubscriptionOperationKind {
  purchase,
  restore,
  customerCenter,
  assignment,
  unassignment,
}

enum PendingSubscriptionOperationPhase { nativeInFlight, reconciliationPending }

final class PendingSubscriptionOperation {
  const PendingSubscriptionOperation._({
    required this.kind,
    required this.phase,
    required this.ownerDid,
    this.beforeProvider,
    this.tier,
    this.licenseId,
  });

  const PendingSubscriptionOperation.purchaseInFlight(
    Did ownerDid,
    SubscriptionTier tier,
  ) : this._(
        kind: PendingSubscriptionOperationKind.purchase,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
        tier: tier,
      );

  const PendingSubscriptionOperation.purchasePending(
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

  const PendingSubscriptionOperation.restoreInFlight(Did ownerDid)
    : this._(
        kind: PendingSubscriptionOperationKind.restore,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
      );

  const PendingSubscriptionOperation.restorePending(
    Did ownerDid,
    BillingState beforeProvider,
  ) : this._(
        kind: PendingSubscriptionOperationKind.restore,
        phase: PendingSubscriptionOperationPhase.reconciliationPending,
        ownerDid: ownerDid,
        beforeProvider: beforeProvider,
      );

  const PendingSubscriptionOperation.customerCenterInFlight(Did ownerDid)
    : this._(
        kind: PendingSubscriptionOperationKind.customerCenter,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
      );

  const PendingSubscriptionOperation.customerCenterPending(Did ownerDid)
    : this._(
        kind: PendingSubscriptionOperationKind.customerCenter,
        phase: PendingSubscriptionOperationPhase.reconciliationPending,
        ownerDid: ownerDid,
      );

  const PendingSubscriptionOperation.assignmentInFlight(
    Did ownerDid,
    String licenseId,
  ) : this._(
        kind: PendingSubscriptionOperationKind.assignment,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
        licenseId: licenseId,
      );

  const PendingSubscriptionOperation.unassignmentInFlight(
    Did ownerDid,
    String licenseId,
  ) : this._(
        kind: PendingSubscriptionOperationKind.unassignment,
        phase: PendingSubscriptionOperationPhase.nativeInFlight,
        ownerDid: ownerDid,
        licenseId: licenseId,
      );

  final PendingSubscriptionOperationKind kind;
  final PendingSubscriptionOperationPhase phase;
  final Did ownerDid;
  final SubscriptionTier? tier;
  final BillingState? beforeProvider;
  final String? licenseId;

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

  bool beginAssignment(Did ownerDid, String licenseId) {
    if (state != null) return false;
    state = PendingSubscriptionOperation.assignmentInFlight(
      ownerDid,
      licenseId,
    );
    return true;
  }

  bool beginUnassignment(Did ownerDid, String licenseId) {
    if (state != null) return false;
    state = PendingSubscriptionOperation.unassignmentInFlight(
      ownerDid,
      licenseId,
    );
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

  void completeAssignmentMutation({
    required PendingSubscriptionOperationKind kind,
    AccountSessionLease? affectedTarget,
  }) {
    final current = state;
    if (current == null || current.kind != kind) return;
    state = null;
    ref.invalidate(subscriptionPageModelProvider);
    if (affectedTarget != null) {
      ref.invalidate(subscriptionAccessProvider(affectedTarget));
    }
  }

  void clear() => state = null;
}
