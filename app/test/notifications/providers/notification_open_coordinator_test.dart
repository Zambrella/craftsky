import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_open_coordinator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-006 and AT-002 gate every routing outcome by binding', () async {
    var routingOutcomes = 0;
    var unavailableCalls = 0;
    AccountSubscriptionId? storedBinding = AccountSubscriptionId.parse(
      'current',
    );
    final coordinator = NotificationOpenCoordinator(
      currentDid: 'did:plc:current',
      loadBinding: (_) async => storedBinding,
      onOutcome: (_) => routingOutcomes++,
      onUnavailable: () => unavailableCalls++,
    );

    await coordinator.open(_attempt(null));
    storedBinding = null;
    await coordinator.open(_attempt('current'));
    storedBinding = AccountSubscriptionId.parse('current');
    await coordinator.open(_attempt('stale'));

    expect(routingOutcomes, 0);
    expect(unavailableCalls, 3);

    await coordinator.open(_attempt('current'));

    expect(routingOutcomes, 1);
    expect(unavailableCalls, 3);
  });
}

NotificationOpenAttempt _attempt(String? binding) =>
    NotificationOpenAttempt.fromProviderData({
      'payloadVersion': '1',
      'type': 'everythingElse',
      'accountSubscriptionId': ?binding,
    });
