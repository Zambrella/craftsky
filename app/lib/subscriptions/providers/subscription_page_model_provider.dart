import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/models/subscription_page_model.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_guard.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_setup_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/paywall_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final FutureProvider<SubscriptionPageModel> subscriptionPageModelProvider =
    FutureProvider.autoDispose<SubscriptionPageModel>((ref) async {
      ref.watch(
        sessionRegistryProvider.select((value) => value.value?.activeLease),
      );
      final registry = await ref.read(sessionRegistryProvider.future);
      final lease = registry.activeLease;
      if (lease == null) {
        throw DiagnosticStateError('Active account unavailable');
      }
      final session = lease.session;
      final api = await ref.watch(
        subscriptionRepositoryProvider(session.account).future,
      );
      final service = ref.watch(revenueCatServiceProvider);
      final state = await BillingAccountSetupCoordinator(
        readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
        setCustomer: ref
            .read(sessionRegistryProvider.notifier)
            .setBillingCustomer,
        api: api,
        revenueCat: service,
      ).setup(lease);
      final access = await ref.watch(
        subscriptionAccessProvider(session).future,
      );
      if (!ref.mounted ||
          !ref.read(sessionRegistryProvider).requireValue.isCurrent(lease)) {
        throw DiagnosticStateError('Account session unavailable');
      }
      final prices = <SubscriptionTier, String>{};
      if (service.availability == BillingAvailability.available) {
        final guard = BillingAccountGuard(
          () => ref.read(sessionRegistryProvider).requireValue,
        );
        final account = guard.capture();
        for (final tier in const [
          SubscriptionTier.plus,
          SubscriptionTier.business,
        ]) {
          try {
            final offering = await guard.dispatchRevenueCat(
              account,
              RevenueCatIdentityGuard(service),
              () => service.getOffering(tier.name),
            );
            final price = offering?.packagePrice(
              PaywallCoordinator.requiredPackageIdentifier,
            );
            if (price != null && price.isNotEmpty) prices[tier] = price;
          } on BillingAccountGuardException {
            rethrow;
          } on Object {
            continue;
          }
        }
      }
      return SubscriptionPageModel(
        access: access,
        billingAvailability: service.availability,
        billingState: state,
        allowNativeTierChanges:
            service is NativeStoreOwnershipService &&
            (service as NativeStoreOwnershipService).supportsPlanChanges,
        nativeStore: service is NativeStoreOwnershipService
            ? (service as NativeStoreOwnershipService).store
            : null,
        tierPrices: prices,
      );
    });
