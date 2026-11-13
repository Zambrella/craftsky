import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
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
import 'package:craftsky_app/subscriptions/services/subscription_diagnostics.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final FutureProvider<SubscriptionPageModel> subscriptionPageModelProvider =
    FutureProvider.autoDispose<SubscriptionPageModel>((ref) async {
      final registry = await ref.watch(sessionRegistryProvider.future);
      if (!ref.mounted) throw AccountOperationCanceled();
      final activeLease = registry.activeLease?.session;
      if (activeLease == null) {
        throw DiagnosticStateError('Active account unavailable');
      }
      final access = await ref.watch(
        subscriptionAccessProvider(activeLease).future,
      );
      if (!ref.mounted) throw AccountOperationCanceled();
      final diagnostics = SubscriptionDiagnostics(
        reporter: ref.read(errorReporterProvider),
        accountDid: activeLease.account.did.value,
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

      final ownerGuard = BillingOwnerGuard(() {
        if (!ref.mounted) {
          throw const BillingOwnerGuardException('owner_session_changed');
        }
        return ref.read(sessionRegistryProvider).requireValue;
      });
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
      } on BillingOwnerGuardException catch (error, stack) {
        if (!error.ownerSessionChanged) {
          diagnostics.report(error, stack, operation: 'billing.load');
        }
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
      } on ApiBadRequest catch (error, stack) {
        diagnostics.report(error, stack, operation: 'billing.load');
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
      } on Object catch (error, stack) {
        diagnostics.report(error, stack, operation: 'billing.load');
        return SubscriptionPageModel(
          role: SubscriptionPageRole.owner,
          access: access,
          ownerRetained: true,
          billingAvailability: availability,
          operationNotice: SubscriptionOperationNotice.providerFailure,
          assignedAccountLabels: labels,
        );
      }
      if (!ref.mounted) throw AccountOperationCanceled();
      final current = ref.read(sessionRegistryProvider).value;
      if (current?.leaseFor(activeLease.account) != activeLease) {
        throw DiagnosticStateError('Account session unavailable');
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
        } on Object catch (error, stack) {
          diagnostics.report(error, stack, operation: 'billing.identity');
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
          ? await _loadTierPrices(
              revenueCat,
              ownerGuard,
              ownerLease,
              diagnostics,
            )
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
  SubscriptionDiagnostics diagnostics,
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
    } on Object catch (error, stack) {
      diagnostics.report(error, stack, operation: 'billing.prices');
      continue;
    }
  }
  return prices;
}
