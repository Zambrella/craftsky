import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';

final class BillingOwnerLease {
  const BillingOwnerLease({
    required this.activeLease,
    required this.revenueCatAppUserId,
  });

  final ActiveAccountLease activeLease;
  final String revenueCatAppUserId;

  @override
  String toString() => 'BillingOwnerLease(<redacted>)';
}

final class BillingOwnerGuard {
  const BillingOwnerGuard(this._readRegistry);

  final SessionRegistry Function() _readRegistry;

  BillingOwnerLease capture() {
    final registry = _readRegistry();
    final activeLease = registry.activeLease;
    final owner = registry.billingOwner;
    final appUserId = owner?.revenueCatAppUserId;
    if (activeLease == null ||
        owner == null ||
        owner.did != activeLease.session.account.did ||
        appUserId == null) {
      throw const BillingOwnerGuardException('active_owner_required');
    }
    return BillingOwnerLease(
      activeLease: activeLease,
      revenueCatAppUserId: appUserId,
    );
  }

  Future<T> dispatch<T>(
    BillingOwnerLease lease,
    Future<T> Function() operation,
  ) async {
    _requireCurrent(lease);
    final result = await operation();
    _requireCurrent(lease);
    return result;
  }

  Future<BillingState> dispatchBillingState(
    BillingOwnerLease lease,
    Future<BillingState> Function() operation,
  ) async {
    final state = await dispatch(lease, operation);
    if (state.revenueCatAppUserId != lease.revenueCatAppUserId) {
      throw const BillingOwnerGuardException('billing_identity_mismatch');
    }
    return state;
  }

  Future<T> dispatchRevenueCat<T>(
    BillingOwnerLease lease,
    RevenueCatIdentityGuard identityGuard,
    Future<T> Function() operation,
  ) async {
    await prepareRevenueCat(lease, identityGuard);
    final result = await operation();
    _requireCurrent(lease);
    return result;
  }

  Future<void> prepareRevenueCat(
    BillingOwnerLease lease,
    RevenueCatIdentityGuard identityGuard,
  ) async {
    _requireCurrent(lease);
    await identityGuard.requireExact(lease.revenueCatAppUserId);
    _requireCurrent(lease);
  }

  bool isCurrent(BillingOwnerLease lease) {
    try {
      _requireCurrent(lease);
      return true;
    } on BillingOwnerGuardException {
      return false;
    }
  }

  void _requireCurrent(BillingOwnerLease lease) {
    final registry = _readRegistry();
    final owner = registry.billingOwner;
    if (!registry.isCurrent(lease.activeLease) ||
        owner?.did != lease.activeLease.session.account.did ||
        owner?.revenueCatAppUserId != lease.revenueCatAppUserId) {
      throw const BillingOwnerGuardException('owner_session_changed');
    }
  }
}

final class BillingOwnerGuardException implements Exception {
  const BillingOwnerGuardException(this.code);

  final String code;

  bool get ownerSessionChanged => code == 'owner_session_changed';

  @override
  String toString() => 'BillingOwnerGuardException($code)';
}
