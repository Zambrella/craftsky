import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/services/notification_presentation_eligibility.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-PUSH-015 permits only a current retained account binding', () async {
    var registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: 'did:plc:recipient',
      handle: 'recipient.test',
    );
    final lease = registry.activeLease!.session;
    registry = registry.saveRoutingBinding(lease, 'routing-account-one');
    final eligibility = SessionRegistryNotificationPresentationEligibility(
      _Storage(registry),
    );

    expect(
      await eligibility.allows(
        AccountSubscriptionId.parse('routing-account-one'),
      ),
      isTrue,
    );
    expect(
      await eligibility.allows(
        AccountSubscriptionId.parse('routing-account-removed'),
      ),
      isFalse,
    );
  });

  test('UT-PUSH-016 fails closed when the secure registry is empty', () async {
    final eligibility = SessionRegistryNotificationPresentationEligibility(
      _Storage(SessionRegistry.empty()),
    );
    expect(
      await eligibility.allows(
        AccountSubscriptionId.parse('routing-account-one'),
      ),
      isFalse,
    );
  });
}

final class _Storage implements SessionRegistryStorage {
  _Storage(this.registry);

  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async {
    this.registry = registry;
  }
}
