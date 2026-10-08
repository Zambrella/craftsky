import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const expected = '20000000-0000-4000-8000-000000000001';

  test('IT-003 only exact AppView UUID reaches provider operation', () async {
    for (final identity in [
      const RevenueCatIdentity.anonymous(),
      const RevenueCatIdentity.identified('did:plc:alice'),
      const RevenueCatIdentity.identified(
        '20000000-0000-4000-8000-000000000002',
      ),
    ]) {
      var calls = 0;
      await expectLater(
        RevenueCatIdentityGuard(_IdentityService(identity)).dispatch(
          expected,
          () async => calls++,
        ),
        throwsA(isA<RevenueCatIdentityException>()),
      );
      expect(calls, 0);
    }

    var calls = 0;
    await const RevenueCatIdentityGuard(
      _IdentityService(RevenueCatIdentity.identified(expected)),
    ).dispatch(expected, () async => calls++);
    expect(calls, 1);
  });
}

final class _IdentityService implements RevenueCatIdentityService {
  const _IdentityService(this.identity);

  final RevenueCatIdentity identity;

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async => identity;

  @override
  Future<void> identify(String appViewUuid) async {}
}
