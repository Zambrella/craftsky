import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/models/subscription_page_model.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/paywall_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final FutureProvider<SubscriptionPageModel> subscriptionPageModelProvider =
    FutureProvider.autoDispose<SubscriptionPageModel>((ref) async {
      final registry = await ref.watch(sessionRegistryProvider.future);
      final activeLease = registry.activeLease?.session;
      if (activeLease == null) throw StateError('Active account unavailable');
      final access = await ref.watch(
        subscriptionAccessProvider(activeLease).future,
      );
      final owner = registry.billingOwner;
      final ownerRetained =
          owner != null && registry.sessions.containsKey(owner.did);
      final revenueCat = ref.watch(revenueCatServiceProvider);
      final availability = revenueCat.availability;
      final labels = {
        for (final session in registry.sessions.values)
          session.did: '@${session.handle.value}',
      };

      if (owner == null) {
        return SubscriptionPageModel(
          role: SubscriptionPageRole.neverReserved,
          access: access,
          billingAvailability: availability,
          assignedAccountLabels: labels,
        );
      }
      if (!ownerRetained) {
        return SubscriptionPageModel(
          role: SubscriptionPageRole.inactiveOwner,
          access: access,
          billingAvailability: availability,
          assignedAccountLabels: labels,
        );
      }
      if (owner.did != activeLease.account.did) {
        return SubscriptionPageModel(
          role: SubscriptionPageRole.beneficiary,
          access: access,
          ownerRetained: ownerRetained,
          billingAvailability: availability,
          assignedAccountLabels: labels,
        );
      }
      if (owner.revenueCatAppUserId == null) {
        return SubscriptionPageModel(
          role: SubscriptionPageRole.incompleteOwner,
          access: access,
          ownerRetained: true,
          billingAvailability: availability,
          assignedAccountLabels: labels,
        );
      }

      final ownerGuard = BillingOwnerGuard(
        () => ref.read(sessionRegistryProvider).requireValue,
      );
      final ownerLease = ownerGuard.capture();
      final repository = await ref.watch(
        subscriptionRepositoryProvider(activeLease.account).future,
      );
      BillingState state;
      try {
        state = await ownerGuard.dispatchBillingState(
          ownerLease,
          repository.getBillingAccount,
        );
      } on BillingOwnerGuardException catch (error) {
        if (error.ownerSessionChanged) rethrow;
        return SubscriptionPageModel(
          role: SubscriptionPageRole.owner,
          access: access,
          ownerRetained: true,
          ownerRecoveryLocked: true,
          billingAvailability: availability,
          assignedAccountLabels: labels,
        );
      } on ApiUnauthorized {
        return SubscriptionPageModel(
          role: SubscriptionPageRole.owner,
          access: access,
          ownerRetained: true,
          ownerSignInRequired: true,
          billingAvailability: availability,
          assignedAccountLabels: labels,
        );
      } on ApiBadRequest catch (error) {
        return SubscriptionPageModel(
          role: SubscriptionPageRole.owner,
          access: access,
          ownerRetained: true,
          ownerRecoveryLocked: error.details.statusCode == 404,
          billingAvailability: availability,
          operationNotice: error.details.statusCode == 404
              ? null
              : SubscriptionOperationNotice.providerFailure,
          assignedAccountLabels: labels,
        );
      } on Object {
        return SubscriptionPageModel(
          role: SubscriptionPageRole.owner,
          access: access,
          ownerRetained: true,
          billingAvailability: availability,
          operationNotice: SubscriptionOperationNotice.providerFailure,
          assignedAccountLabels: labels,
        );
      }
      final current = ref.read(sessionRegistryProvider).value;
      if (!ref.mounted ||
          current?.leaseFor(activeLease.account) != activeLease) {
        throw StateError('Account session unavailable');
      }
      if (availability == BillingAvailability.available) {
        RevenueCatIdentity identity;
        try {
          if (!ownerGuard.isCurrent(ownerLease)) {
            throw const BillingOwnerGuardException('owner_session_changed');
          }
          identity = await revenueCat.currentIdentity();
          if (!ref.mounted || !ownerGuard.isCurrent(ownerLease)) {
            throw const BillingOwnerGuardException('owner_session_changed');
          }
        } on BillingOwnerGuardException {
          rethrow;
        } on Object {
          return SubscriptionPageModel(
            role: SubscriptionPageRole.owner,
            access: access,
            ownerRetained: true,
            billingAvailability: availability,
            operationNotice: SubscriptionOperationNotice.providerFailure,
            assignedAccountLabels: labels,
          );
        }
        if (identity.kind == RevenueCatIdentityKind.anonymous) {
          return SubscriptionPageModel(
            role: SubscriptionPageRole.incompleteOwner,
            access: access,
            ownerRetained: true,
            billingAvailability: availability,
            assignedAccountLabels: labels,
          );
        }
        if (identity.appUserId != owner.revenueCatAppUserId) {
          return SubscriptionPageModel(
            role: SubscriptionPageRole.owner,
            access: access,
            ownerRetained: true,
            ownerRecoveryLocked: true,
            billingAvailability: availability,
            assignedAccountLabels: labels,
          );
        }
      }
      final tierPrices = availability == BillingAvailability.available
          ? await _loadTierPrices(revenueCat, ownerGuard, ownerLease)
          : const <SubscriptionTier, String>{};
      return SubscriptionPageModel(
        role: SubscriptionPageRole.owner,
        access: access,
        ownerRetained: true,
        billingAvailability: availability,
        billingState: state,
        assignedAccountLabels: labels,
        tierPrices: tierPrices,
      );
    });

Future<Map<SubscriptionTier, String>> _loadTierPrices(
  RevenueCatService revenueCat,
  BillingOwnerGuard ownerGuard,
  BillingOwnerLease ownerLease,
) async {
  final prices = <SubscriptionTier, String>{};
  for (final tier in const [
    SubscriptionTier.plus,
    SubscriptionTier.business,
  ]) {
    try {
      final offering = await ownerGuard.dispatchRevenueCat(
        ownerLease,
        RevenueCatIdentityGuard(revenueCat),
        () => revenueCat.getOffering(tier.name),
      );
      final price = offering?.packagePrice(
        PaywallCoordinator.requiredPackageIdentifier,
      );
      if (price != null && price.isNotEmpty) prices[tier] = price;
    } on BillingOwnerGuardException {
      rethrow;
    } on Object {
      continue;
    }
  }
  return prices;
}
