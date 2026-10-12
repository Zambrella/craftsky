import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';

final class RevenueCatIdentityGuard {
  const new(this._service);

  final RevenueCatIdentityService _service;

  Future<T> dispatch<T>(
    String expectedAppUserId,
    Future<T> Function() operation,
  ) async {
    await requireExact(expectedAppUserId);
    return await operation();
  }

  Future<void> requireExact(String expectedAppUserId) async {
    if (_service.availability != BillingAvailability.available) {
      throw const RevenueCatIdentityException('billing_unavailable');
    }
    final identity = await _service.currentIdentity();
    if (identity.kind != RevenueCatIdentityKind.identified ||
        identity.appUserId != expectedAppUserId) {
      throw const RevenueCatIdentityException('identity_mismatch');
    }
  }
}

final class RevenueCatIdentityException implements Exception {
  const new(this.code);

  final String code;

  @override
  String toString() => 'RevenueCatIdentityException($code)';
}
