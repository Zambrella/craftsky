import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';

enum PaywallOutcome { reconcile, cancelled, offeringUnavailable, providerError }

final class PaywallCoordinator {
  const PaywallCoordinator(
    this._service,
    this._ownerGuard,
    this._identityGuard,
  );

  static const requiredPackageIdentifier = r'$rc_monthly';

  final RevenueCatService _service;
  final BillingOwnerGuard _ownerGuard;
  final RevenueCatIdentityGuard _identityGuard;

  Future<PaywallOutcome> present(
    BillingOwnerLease owner,
    SubscriptionTier tier,
  ) async {
    if (_service.availability != BillingAvailability.available) {
      return PaywallOutcome.offeringUnavailable;
    }
    try {
      final offering = await _ownerGuard.dispatchRevenueCat(
        owner,
        _identityGuard,
        () => _service.getOffering(tier.name),
      );
      if (offering == null || !offering.hasPackage(requiredPackageIdentifier)) {
        return PaywallOutcome.offeringUnavailable;
      }
      await _ownerGuard.prepareRevenueCat(owner, _identityGuard);
      final result = await _service.presentPaywall(offering);
      return switch (result) {
        DirectPaywallResult.purchased ||
        DirectPaywallResult.restored => PaywallOutcome.reconcile,
        DirectPaywallResult.cancelled => PaywallOutcome.cancelled,
        DirectPaywallResult.error => PaywallOutcome.providerError,
      };
    } on BillingOwnerGuardException catch (error) {
      return error.ownerSessionChanged
          ? PaywallOutcome.cancelled
          : PaywallOutcome.providerError;
    } on Object {
      return PaywallOutcome.providerError;
    }
  }
}
