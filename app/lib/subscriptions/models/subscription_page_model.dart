import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:material_ui/material_ui.dart';

enum SubscriptionOperationNotice {
  offeringUnavailable,
  providerFailure,
  storeAccountConflict,
  planChangeUnavailable,
  planChangeScheduled,
  unauthorized,
  actionFailed,
  pending,
  supportRequired,
}

final class SubscriptionPageModel {
  const new({
    required this.access,
    required this.billingAvailability,
    this.billingState,
    this.nativeStore,
    this.allowNativeTierChanges = false,
    this.operationNotice,
    this.conflictingAccountLabel,
    this.tierPrices = const {},
    this.onPurchase,
    this.onRestore,
    this.onManage,
    this.onRefresh,
  });

  final SubscriptionAccess access;
  final BillingAvailability billingAvailability;
  final BillingState? billingState;
  final String? nativeStore;
  final bool allowNativeTierChanges;
  final SubscriptionOperationNotice? operationNotice;
  final String? conflictingAccountLabel;
  final Map<SubscriptionTier, String> tierPrices;
  final ValueChanged<SubscriptionTier>? onPurchase;
  final VoidCallback? onRestore;
  final VoidCallback? onManage;
  final VoidCallback? onRefresh;
}
