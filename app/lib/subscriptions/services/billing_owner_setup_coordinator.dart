import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';

class BillingOwnerSetupException implements Exception {
  const BillingOwnerSetupException(this.code);

  final String code;

  @override
  String toString() => 'BillingOwnerSetupException($code)';
}

final class BillingOwnerSetupCoordinator {
  const BillingOwnerSetupCoordinator({
    required this.readRegistry,
    required this.reserveOwner,
    required this.completeOwner,
    required this.api,
    required this.revenueCat,
  });

  final SessionRegistry Function() readRegistry;
  final Future<void> Function(String did) reserveOwner;
  final Future<void> Function(String did, String appUserId) completeOwner;
  final SubscriptionApi api;
  final RevenueCatIdentityService revenueCat;

  Future<BillingState> setup(ActiveAccountLease lease) async {
    var registry = readRegistry();
    _requireCurrentOwnerLease(registry, lease, allowUnreserved: true);
    final did = lease.session.account.did.value;

    if (registry.billingOwner == null) {
      await reserveOwner(did);
      registry = readRegistry();
      _requireCurrentOwnerLease(registry, lease);
    }

    final reservedUuid = registry.billingOwner!.revenueCatAppUserId;
    final state = reservedUuid == null
        ? await api.ensureBillingAccount()
        : await api.getBillingAccount();

    registry = readRegistry();
    _requireCurrentOwnerLease(registry, lease);
    if (reservedUuid != null && state.revenueCatAppUserId != reservedUuid) {
      throw const BillingOwnerSetupException('billing_account_mismatch');
    }

    if (reservedUuid == null) {
      await completeOwner(did, state.revenueCatAppUserId);
      registry = readRegistry();
      _requireCurrentOwnerLease(registry, lease);
    }

    final appUserId = registry.billingOwner!.revenueCatAppUserId!;
    final identity = await revenueCat.currentIdentity();
    registry = readRegistry();
    _requireCurrentOwnerLease(registry, lease);
    if (identity.kind == RevenueCatIdentityKind.anonymous) {
      await revenueCat.identify(appUserId);
      _requireCurrentOwnerLease(readRegistry(), lease);
    } else if (identity.appUserId != appUserId) {
      throw const BillingOwnerSetupException('revenuecat_identity_mismatch');
    }

    return state;
  }

  void _requireCurrentOwnerLease(
    SessionRegistry registry,
    ActiveAccountLease lease, {
    bool allowUnreserved = false,
  }) {
    if (!registry.isCurrent(lease)) {
      throw const BillingOwnerSetupException('owner_session_changed');
    }
    final owner = registry.billingOwner;
    if (owner == null) {
      if (allowUnreserved) return;
      throw const BillingOwnerSetupException('owner_reservation_missing');
    }
    if (owner.did != lease.session.account.did) {
      throw const BillingOwnerSetupException('billing_owner_mismatch');
    }
  }
}
