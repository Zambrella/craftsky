import 'package:craftsky_app/router/router.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AT-011 restricted accounts retain safety and account routes', () {
    expect(
      [
        const AccountEligibilityRoute().location,
        const AccountStandingRoute().location,
        const AccountRoute().location,
        const MutedAccountsRoute().location,
        const BlockedAccountsRoute().location,
        const AboutRoute().location,
      ].every(isRestrictedAccountRetainedLocation),
      isTrue,
    );
    expect(
      isRestrictedAccountRetainedLocation(const FeedRoute().location),
      isFalse,
    );
  });
}
