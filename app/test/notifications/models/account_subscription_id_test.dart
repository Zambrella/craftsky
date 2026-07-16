import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses bounded routing IDs and redacts string output', () {
    const wireValue = 'subscription_Abc-123';
    final id = AccountSubscriptionId.parse(wireValue);

    expect(id.wireValue, wireValue);
    expect(id, AccountSubscriptionId.parse(wireValue));
    expect(id.toString(), '<redacted-account-subscription-id>');
  });

  test('rejects malformed routing IDs', () {
    for (final value in ['', 'contains spaces', 'a' * 129]) {
      expect(() => AccountSubscriptionId.parse(value), throwsFormatException);
    }
  });
}
