import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';

enum SubscriptionTierPresentation {
  empty,
  active,
  canceledAccessible,
  dormant,
  stale,
  pending,
  anomaly,
  statusUnavailable,
}

SubscriptionTierPresentation projectTierPresentation(
  BillingState state, {
  SubscriptionTier tier = SubscriptionTier.plus,
}) {
  final subscriptions = {
    for (final value in state.subscriptions) value.id: value,
  };
  final pairs = state.licenses
      .where((value) => value.tier == tier)
      .map((license) => (license, subscriptions[license.subscriptionId]))
      .toList();
  if (pairs.isEmpty) return SubscriptionTierPresentation.empty;
  if (pairs.any(
    (pair) =>
        pair.$2 == null ||
        isBillingAnomaly(pair.$1.anomaly) ||
        isBillingAnomaly(pair.$2!.anomaly),
  )) {
    return SubscriptionTierPresentation.anomaly;
  }
  if (state.reconciliationStale ||
      state.requestedGeneration > state.reconciledGeneration) {
    return SubscriptionTierPresentation.stale;
  }
  if (pairs.any((pair) => pair.$2!.pendingPayment)) {
    return SubscriptionTierPresentation.pending;
  }
  const knownStatuses = {'active', 'cancelled', 'expired'};
  if (pairs.any((pair) => !knownStatuses.contains(pair.$2!.status))) {
    return SubscriptionTierPresentation.statusUnavailable;
  }
  if (pairs.any(
    (pair) => pair.$2!.status == 'cancelled' && pair.$2!.givesAccess,
  )) {
    return SubscriptionTierPresentation.canceledAccessible;
  }
  if (pairs.any((pair) => pair.$2!.givesAccess)) {
    return SubscriptionTierPresentation.active;
  }
  return SubscriptionTierPresentation.dormant;
}

SubscriptionTierPresentation projectLicensePresentation(
  BillingState state,
  BillingLicense license,
) {
  final subscription = state.subscriptions
      .where((value) => value.id == license.subscriptionId)
      .firstOrNull;
  if (subscription == null ||
      isBillingAnomaly(license.anomaly) ||
      isBillingAnomaly(subscription.anomaly)) {
    return SubscriptionTierPresentation.anomaly;
  }
  if (state.reconciliationStale ||
      state.requestedGeneration > state.reconciledGeneration) {
    return SubscriptionTierPresentation.stale;
  }
  if (subscription.pendingPayment) {
    return SubscriptionTierPresentation.pending;
  }
  if (!const {'active', 'cancelled', 'expired'}.contains(subscription.status)) {
    return SubscriptionTierPresentation.statusUnavailable;
  }
  if (subscription.status == 'cancelled' && subscription.givesAccess) {
    return SubscriptionTierPresentation.canceledAccessible;
  }
  if (subscription.givesAccess) return SubscriptionTierPresentation.active;
  return SubscriptionTierPresentation.dormant;
}

SubscriptionTier effectiveTierForPresentation(
  BillingState _,
  SubscriptionAccess access,
) => access.effectiveTier;
