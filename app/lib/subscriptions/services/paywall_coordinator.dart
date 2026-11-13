import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';

enum PaywallOutcome {
  reconcile,
  cancelled,
  offeringUnavailable,
  providerError,
  storeAccountConflict,
  planChangeUnavailable,
}

final class PaywallCoordinator {
  const new(
    this._service,
    this._ownerGuard,
    this._identityGuard, {
    this.onStoreConflict,
  });

  static const requiredPackageIdentifier = r'$rc_monthly';

  final RevenueCatService _service;
  final void Function(StoreAccountConflictException)? onStoreConflict;
  final BillingAccountGuard _ownerGuard;
  final RevenueCatIdentityGuard _identityGuard;

  Future<PaywallOutcome> present(
    BillingAccountLease owner,
    SubscriptionTier tier,
  ) => _present(owner, tier, changePlan: false);

  Future<PaywallOutcome> changePlan(
    BillingAccountLease owner,
    SubscriptionTier tier,
  ) => _present(owner, tier, changePlan: true);

  Future<PaywallOutcome> _present(
    BillingAccountLease owner,
    SubscriptionTier tier, {
    required bool changePlan,
  }) async {
    if (_service.availability != BillingAvailability.available) {
      return PaywallOutcome.offeringUnavailable;
    }
    try {
      return await _identityGuard.dispatch(
        owner.revenueCatAppUserId,
        () async {
          final offering = await _service.getOffering(tier.name);
          if (offering == null ||
              !offering.hasPackage(requiredPackageIdentifier)) {
            return PaywallOutcome.offeringUnavailable;
          }
          final service = _service;
          if (service is NativeStoreOwnershipService) {
            await (service as NativeStoreOwnershipService).verifyStoreOwnership(
              owner.revenueCatAppUserId,
            );
          }
          if (!_ownerGuard.isCurrent(owner)) return PaywallOutcome.cancelled;
          final DirectPaywallResult result;
          if (changePlan) {
            if (service is! NativePlanChangeService) {
              return PaywallOutcome.planChangeUnavailable;
            }
            result = await (service as NativePlanChangeService).changePlan(
              offering,
              isCurrent: () => _ownerGuard.isCurrent(owner),
            );
          } else {
            result = await service.presentPaywall(offering);
          }
          return switch (result) {
            DirectPaywallResult.purchased ||
            DirectPaywallResult.restored => PaywallOutcome.reconcile,
            DirectPaywallResult.cancelled => PaywallOutcome.cancelled,
            DirectPaywallResult.error => PaywallOutcome.providerError,
            DirectPaywallResult.storeAccountConflict =>
              PaywallOutcome.storeAccountConflict,
          };
        },
        isCurrent: () => _ownerGuard.isCurrent(owner),
      );
    } on NativePlanChangeUnavailableException {
      return PaywallOutcome.planChangeUnavailable;
    } on StoreAccountConflictException catch (error) {
      if (!_ownerGuard.isCurrent(owner)) return PaywallOutcome.cancelled;
      onStoreConflict?.call(error);
      return PaywallOutcome.storeAccountConflict;
    } on BillingAccountGuardException catch (error) {
      return error.ownerSessionChanged
          ? PaywallOutcome.cancelled
          : PaywallOutcome.providerError;
    } on Object {
      return PaywallOutcome.providerError;
    }
  }
}
