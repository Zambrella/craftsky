import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';

final class BillingAccountSetupException implements Exception {
  const new(this.code);
  final String code;

  @override
  String toString() => 'BillingAccountSetupException($code)';
}

/// Obtains the active DID's customer identity from AppView. No local account
/// receives authority to purchase or manage another account's subscription.
final class BillingAccountSetupCoordinator {
  const new({
    required this.readRegistry,
    required this.setCustomer,
    required this.api,
    required this.revenueCat,
  });

  final SessionRegistry Function() readRegistry;
  final Future<void> Function(String did, String appUserId) setCustomer;
  final SubscriptionApi api;
  final RevenueCatIdentityService revenueCat;

  Future<BillingState> setup(ActiveAccountLease lease) async {
    void requireCurrent() {
      if (!readRegistry().isCurrent(lease)) {
        throw const BillingAccountSetupException('account_session_changed');
      }
    }

    requireCurrent();
    final state = await api.ensureBillingAccount();
    requireCurrent();
    await setCustomer(
      lease.session.account.did.value,
      state.revenueCatAppUserId,
    );
    requireCurrent();
    if (revenueCat.availability == BillingAvailability.available) {
      try {
        await RevenueCatIdentityGuard(revenueCat).dispatch(
          state.revenueCatAppUserId,
          () async {},
          isCurrent: () => readRegistry().isCurrent(lease),
        );
      } on Object {
        // AppView remains authoritative even when this device's store belongs
        // to another account or its SDK is unavailable. Native actions perform
        // the same ownership/identity checks before any provider operation.
      }
      requireCurrent();
    }
    return state;
  }
}
