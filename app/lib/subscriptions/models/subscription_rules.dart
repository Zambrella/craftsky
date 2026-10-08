import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';

enum PurchaseEligibilityReason {
  eligible,
  reconciliationPending,
  active,
  pendingPayment,
  renewalUncertain,
  anomaly,
}

final class PurchaseEligibility {
  PurchaseEligibility({
    required this.reason,
    Iterable<Did> dormantAssignedDids = const [],
  }) : dormantAssignedDids = List.unmodifiable(dormantAssignedDids);

  final PurchaseEligibilityReason reason;
  final List<Did> dormantAssignedDids;

  bool get eligible => reason == PurchaseEligibilityReason.eligible;
}

PurchaseEligibility purchaseEligibility(
  BillingState state,
  SubscriptionTier tier,
) {
  if (state.reconciliationStale ||
      state.requestedGeneration > state.reconciledGeneration) {
    return PurchaseEligibility(
      reason: PurchaseEligibilityReason.reconciliationPending,
    );
  }

  if (!billingRelationshipsResolved(state)) {
    return PurchaseEligibility(reason: PurchaseEligibilityReason.anomaly);
  }

  final subscriptionsById = {
    for (final subscription in state.subscriptions)
      subscription.id: subscription,
  };
  final licenses = state.licenses
      .where((license) => license.tier == tier)
      .toList();
  if (licenses.isEmpty) {
    return PurchaseEligibility(reason: PurchaseEligibilityReason.eligible);
  }

  final subscriptions = <BillingSubscription>[];
  for (final license in licenses) {
    final subscription = subscriptionsById[license.subscriptionId];
    if (subscription == null ||
        isBillingAnomaly(license.anomaly) ||
        isBillingAnomaly(subscription.anomaly)) {
      return PurchaseEligibility(reason: PurchaseEligibilityReason.anomaly);
    }
    subscriptions.add(subscription);
  }
  if (subscriptions.any((subscription) => subscription.givesAccess)) {
    return PurchaseEligibility(reason: PurchaseEligibilityReason.active);
  }
  if (subscriptions.any((subscription) => subscription.pendingPayment)) {
    return PurchaseEligibility(
      reason: PurchaseEligibilityReason.pendingPayment,
    );
  }
  if (subscriptions.any(
    (subscription) => subscription.autoRenewalStatus != 'will_not_renew',
  )) {
    return PurchaseEligibility(
      reason: PurchaseEligibilityReason.renewalUncertain,
    );
  }

  return PurchaseEligibility(
    reason: PurchaseEligibilityReason.eligible,
    dormantAssignedDids: licenses
        .map((license) => license.assignedDid)
        .nonNulls,
  );
}

List<AccountSessionLease> assignmentCandidates({
  required SessionRegistry registry,
  required Iterable<AccountSessionLease> retainedLeases,
  required BillingState state,
}) {
  final assignedDids = state.licenses
      .map((value) => value.assignedDid)
      .nonNulls
      .toSet();
  final seen = <Did>{};
  return List.unmodifiable(
    retainedLeases.where((lease) {
      final did = lease.account.did;
      return seen.add(did) &&
          registry.leaseFor(lease.account) == lease &&
          !assignedDids.contains(did);
    }),
  );
}
