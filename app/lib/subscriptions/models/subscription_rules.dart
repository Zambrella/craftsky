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
  new({
    required this.reason,
    Iterable<Did> dormantAssignedDids = const [],
  }) : dormantAssignedDids = List.unmodifiable(dormantAssignedDids);

  final PurchaseEligibilityReason reason;
  final List<Did> dormantAssignedDids;

  bool get eligible => reason == PurchaseEligibilityReason.eligible;
}

PurchaseEligibility purchaseEligibility(
  BillingState state,
  SubscriptionTier tier, {
  String? store,
  bool allowTierChanges = false,
}) {
  // An identified web checkout can still be payable before RevenueCat has a
  // subscription to reconcile. Do not start a separate native purchase then.
  if (state.webCheckoutPending) {
    return PurchaseEligibility(
      reason: PurchaseEligibilityReason.pendingPayment,
    );
  }
  if (state.reconciliationStale ||
      state.requestedGeneration > state.reconciledGeneration) {
    return PurchaseEligibility(
      reason: PurchaseEligibilityReason.reconciliationPending,
    );
  }

  if (!billingRelationshipsResolved(state)) {
    return PurchaseEligibility(reason: PurchaseEligibilityReason.anomaly);
  }

  if (state.subscriptions.any(
    (subscription) =>
        subscription.pendingTier != null ||
        subscription.autoRenewalStatus == 'will_change_product',
  )) {
    return PurchaseEligibility(
      reason: PurchaseEligibilityReason.renewalUncertain,
    );
  }

  if (state.subscriptions.any((subscription) => subscription.pendingPayment)) {
    return PurchaseEligibility(
      reason: PurchaseEligibilityReason.pendingPayment,
    );
  }
  if (state.subscriptions.any(
        (subscription) => isBillingAnomaly(subscription.anomaly),
      ) ||
      state.licenses.any((license) => isBillingAnomaly(license.anomaly))) {
    return PurchaseEligibility(reason: PurchaseEligibilityReason.anomaly);
  }
  // A plan change can only replace a subscription in the billing provider
  // currently taking payment. A second store cannot replace the first one.
  if (state.subscriptions.any(
    (subscription) =>
        (subscription.givesAccess ||
            subscription.pendingPayment ||
            subscription.autoRenewalStatus != 'will_not_renew') &&
        (store == null || subscription.store != store),
  )) {
    return PurchaseEligibility(reason: PurchaseEligibilityReason.active);
  }
  if (state.subscriptions.any(
    (subscription) =>
        !subscription.givesAccess &&
        subscription.autoRenewalStatus != 'will_not_renew',
  )) {
    return PurchaseEligibility(
      reason: PurchaseEligibilityReason.renewalUncertain,
    );
  }

  if (!allowTierChanges &&
      state.subscriptions.any((value) => value.givesAccess)) {
    return PurchaseEligibility(reason: PurchaseEligibilityReason.active);
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
