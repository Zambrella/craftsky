import 'dart:async';

import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';

/// Serializes customer changes and the entire native operation. A paywall or
/// restore must finish before another account can identify the shared SDK.
final class RevenueCatIdentityGuard {
  const new(this._service);

  static final _tails = Expando<Future<void>>();
  final RevenueCatIdentityService _service;

  Future<T> dispatch<T>(
    String expectedAppUserId,
    Future<T> Function() operation, {
    bool Function()? isCurrent,
  }) async {
    final previous = _tails[_service] ?? Future<void>.value();
    final completed = Completer<void>();
    _tails[_service] = completed.future;
    await previous;
    try {
      void requireCurrent() {
        if (isCurrent != null && !isCurrent()) {
          throw const RevenueCatIdentityException('account_session_changed');
        }
      }

      requireCurrent();
      if (_service.availability != BillingAvailability.available) {
        throw const RevenueCatIdentityException('billing_unavailable');
      }
      final identity = await _service.currentIdentity();
      requireCurrent();
      if (identity.kind != RevenueCatIdentityKind.identified ||
          identity.appUserId != expectedAppUserId) {
        // Identified-to-identified login does not merge independent customers.
        await _service.identify(expectedAppUserId);
        requireCurrent();
      }
      await requireExact(expectedAppUserId);
      requireCurrent();
      return await operation();
    } finally {
      completed.complete();
    }
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
