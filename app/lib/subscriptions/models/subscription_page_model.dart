import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter/material.dart';

enum SubscriptionPageRole {
  neverReserved,
  incompleteOwner,
  inactiveOwner,
  owner,
  beneficiary,
}

enum SubscriptionOperationNotice {
  offeringUnavailable,
  providerFailure,
  assignmentConflict,
  assignmentCooldown,
  licenseNotFound,
  targetIneligible,
  unauthorized,
  actionFailed,
  pending,
  supportRequired,
}

final class SubscriptionPageModel {
  const SubscriptionPageModel({
    required this.role,
    required this.access,
    required this.billingAvailability,
    this.ownerRetained = false,
    this.ownerRecoveryLocked = false,
    this.ownerSignInRequired = false,
    this.billingState,
    this.operationNotice,
    this.assignedAccountLabels = const {},
    this.tierPrices = const {},
    this.onSwitchToOwner,
    this.onConfirmOwner,
    this.onRetrySetup,
    this.onPurchase,
    this.onRestore,
    this.onManage,
    this.onRefresh,
    this.onAssign,
    this.onUnassign,
  });

  final SubscriptionPageRole role;
  final SubscriptionAccess access;
  final BillingAvailability billingAvailability;
  final bool ownerRetained;
  final bool ownerRecoveryLocked;
  final bool ownerSignInRequired;
  final BillingState? billingState;
  final SubscriptionOperationNotice? operationNotice;
  final Map<Did, String> assignedAccountLabels;
  final Map<SubscriptionTier, String> tierPrices;
  final VoidCallback? onSwitchToOwner;
  final VoidCallback? onConfirmOwner;
  final VoidCallback? onRetrySetup;
  final ValueChanged<SubscriptionTier>? onPurchase;
  final VoidCallback? onRestore;
  final VoidCallback? onManage;
  final VoidCallback? onRefresh;
  final ValueChanged<BillingLicense>? onAssign;
  final ValueChanged<BillingLicense>? onUnassign;
}
