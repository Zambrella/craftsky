import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';

final class BillingAccountLease {
  const new({
    required this.activeLease,
    required this.revenueCatAppUserId,
  });

  final ActiveAccountLease activeLease;
  final String revenueCatAppUserId;

  @override
  String toString() => 'BillingAccountLease(<redacted>)';
}

final class BillingAccountGuard {
  const new(this._readRegistry);

  final SessionRegistry Function() _readRegistry;

  BillingAccountLease capture() {
    final registry = _readRegistry();
    final activeLease = registry.activeLease;
    final appUserId = activeLease == null
        ? null
        : registry.billingCustomers[activeLease.session.account.did];
    if (activeLease == null || appUserId == null) {
      throw const BillingAccountGuardException('active_account_required');
    }
    return BillingAccountLease(
      activeLease: activeLease,
      revenueCatAppUserId: appUserId,
    );
  }

  Future<T> dispatch<T>(
    BillingAccountLease lease,
    Future<T> Function() operation,
  ) async {
    _requireCurrent(lease);
    final result = await operation();
    _requireCurrent(lease);
    return result;
  }

  Future<BillingState> dispatchBillingState(
    BillingAccountLease lease,
    Future<BillingState> Function() operation,
  ) async {
    final state = await dispatch(lease, operation);
    if (state.revenueCatAppUserId != lease.revenueCatAppUserId) {
      throw const BillingAccountGuardException('billing_identity_mismatch');
    }
    return state;
  }

  Future<T> dispatchRevenueCat<T>(
    BillingAccountLease lease,
    RevenueCatIdentityGuard identityGuard,
    Future<T> Function() operation,
  ) async {
    _requireCurrent(lease);
    late final T result;
    try {
      result = await identityGuard.dispatch(
        lease.revenueCatAppUserId,
        operation,
        isCurrent: () => isCurrent(lease),
      );
    } on RevenueCatIdentityException catch (error) {
      if (error.code == 'account_session_changed') {
        throw const BillingAccountGuardException('owner_session_changed');
      }
      rethrow;
    }
    _requireCurrent(lease);
    return result;
  }

  Future<void> prepareRevenueCat(
    BillingAccountLease lease,
    RevenueCatIdentityGuard identityGuard,
  ) async {
    _requireCurrent(lease);
    await identityGuard.requireExact(lease.revenueCatAppUserId);
    _requireCurrent(lease);
  }

  bool isCurrent(BillingAccountLease lease) {
    try {
      _requireCurrent(lease);
      return true;
    } on BillingAccountGuardException {
      return false;
    }
  }

  void _requireCurrent(BillingAccountLease lease) {
    final registry = _readRegistry();
    if (!registry.isCurrent(lease.activeLease) ||
        registry.billingCustomers[lease.activeLease.session.account.did] !=
            lease.revenueCatAppUserId) {
      throw const BillingAccountGuardException('owner_session_changed');
    }
  }
}

final class BillingAccountGuardException implements Exception {
  const new(this.code);

  final String code;

  bool get ownerSessionChanged => code == 'owner_session_changed';

  @override
  String toString() => 'BillingAccountGuardException($code)';
}
