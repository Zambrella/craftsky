import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/subscriptions/services/subscription_diagnostics.dart';

enum PaywallOutcome {
  reconcile,
  restored,
  cancelled,
  offeringUnavailable,
  providerError,
  storeAccountConflict,
  planChangeUnavailable,
  existingStoreSubscription,
}

final class PaywallCoordinator {
  const new(
    this._service,
    this._ownerGuard,
    this._identityGuard, {
    this.onStoreConflict,
    this.isCurrent = _alwaysCurrent,
    this.diagnostics = const SubscriptionDiagnostics(),
  });

  static const requiredPackageIdentifier = r'$rc_monthly';

  final RevenueCatService _service;
  final void Function(StoreAccountConflictException)? onStoreConflict;
  final SubscriptionDiagnostics diagnostics;
  final BillingAccountGuard _ownerGuard;
  final RevenueCatIdentityGuard _identityGuard;
  final bool Function() isCurrent;

  bool _operationIsCurrent(BillingAccountLease owner) =>
      isCurrent() && _ownerGuard.isCurrent(owner);

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
    if (!_operationIsCurrent(owner)) return PaywallOutcome.cancelled;
    if (_service.availability != BillingAvailability.available) {
      return PaywallOutcome.offeringUnavailable;
    }
    try {
      return await _identityGuard.dispatch(
        owner.revenueCatAppUserId,
        () async {
          final offering = await _service.getOffering(tier.name);
          if (!_operationIsCurrent(owner)) return PaywallOutcome.cancelled;
          if (offering == null ||
              !offering.hasPackage(requiredPackageIdentifier)) {
            return PaywallOutcome.offeringUnavailable;
          }
          final service = _service;
          if (service is NativeStoreOwnershipService) {
            await (service as NativeStoreOwnershipService).verifyStoreOwnership(
              owner.revenueCatAppUserId,
              requireNoActiveSubscription: !changePlan,
            );
          }
          if (!_operationIsCurrent(owner)) return PaywallOutcome.cancelled;
          final DirectPaywallResult result;
          if (changePlan) {
            if (service is! NativePlanChangeService) {
              return PaywallOutcome.planChangeUnavailable;
            }
            result = await (service as NativePlanChangeService).changePlan(
              offering,
              isCurrent: () => _operationIsCurrent(owner),
            );
          } else {
            result = await service.presentPaywall(offering);
          }
          return switch (result) {
            DirectPaywallResult.purchased => PaywallOutcome.reconcile,
            DirectPaywallResult.restored => PaywallOutcome.restored,
            DirectPaywallResult.cancelled => PaywallOutcome.cancelled,
            DirectPaywallResult.error => PaywallOutcome.providerError,
            DirectPaywallResult.storeAccountConflict =>
              PaywallOutcome.storeAccountConflict,
          };
        },
        isCurrent: () => _operationIsCurrent(owner),
      );
    } on NativePlanChangeUnavailableException {
      return PaywallOutcome.planChangeUnavailable;
    } on StoreSubscriptionExistsException {
      return _operationIsCurrent(owner)
          ? PaywallOutcome.existingStoreSubscription
          : PaywallOutcome.cancelled;
    } on StoreAccountConflictException catch (error) {
      if (!_operationIsCurrent(owner)) return PaywallOutcome.cancelled;
      onStoreConflict?.call(error);
      return PaywallOutcome.storeAccountConflict;
    } on BillingAccountGuardException catch (error) {
      return error.ownerSessionChanged
          ? PaywallOutcome.cancelled
          : PaywallOutcome.providerError;
    } on RevenueCatIdentityException catch (error, stack) {
      diagnostics.report(error, stack, operation: 'paywall.present');
      return error.code == 'account_session_changed'
          ? PaywallOutcome.cancelled
          : PaywallOutcome.providerError;
    } on Object catch (error, stack) {
      diagnostics.report(error, stack, operation: 'paywall.present');
      return PaywallOutcome.providerError;
    }
  }
}

bool _alwaysCurrent() => true;
