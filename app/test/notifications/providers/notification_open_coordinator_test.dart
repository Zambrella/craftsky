import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_id.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_resolution.dart';
import 'package:craftsky_app/notifications/services/notification_open_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_resolution_policy.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'AT-004 resolves a matching binding and uses only AppView target',
    () async {
      var resolveCalls = 0;
      NotificationResolutionOutcome? outcome;
      final coordinator = NotificationOpenCoordinator(
        currentDid: 'did:plc:current',
        loadBinding: (_) async => AccountSubscriptionId.parse('current'),
        resolve: (id) async {
          resolveCalls++;
          return NotificationResolution(
            id: id,
            category: NotificationCategory.like,
            state: NotificationResolutionState.active,
            target: NotificationPostTarget(
              AtUri.parse('at://did:plc:actor/social.craftsky.feed.post/123'),
            ),
          );
        },
        onOutcome: (value) => outcome = value,
        onUnavailable: () => fail('matching binding should resolve'),
      );

      await coordinator.open(_event('current'));

      expect(resolveCalls, 1);
      expect(
        outcome?.destination,
        NotificationDestination.post(
          AtUri.parse('at://did:plc:actor/social.craftsky.feed.post/123'),
        ),
      );
    },
  );

  test('AT-005 rejects stale or missing bindings without HTTP', () async {
    var resolveCalls = 0;
    var unavailableCalls = 0;
    AccountSubscriptionId? storedBinding = AccountSubscriptionId.parse(
      'current',
    );
    final coordinator = NotificationOpenCoordinator(
      currentDid: 'did:plc:current',
      loadBinding: (_) async => storedBinding,
      resolve: (_) async {
        resolveCalls++;
        throw StateError('must not resolve');
      },
      onOutcome: (_) => fail('must not navigate'),
      onUnavailable: () => unavailableCalls++,
    );

    await coordinator.open(_event('stale'));
    storedBinding = null;
    await coordinator.open(_event('current'));

    expect(resolveCalls, 0);
    expect(unavailableCalls, 2);
  });
}

NotificationOpenEvent _event(String binding) => NotificationOpenEvent(
  notificationId: NotificationId.parse(
    '00000000-0000-0000-0000-000000000001',
  ),
  category: NotificationCategory.unknown,
  accountSubscriptionId: AccountSubscriptionId.parse(binding),
  source: NotificationOpenSource.backgroundOpen,
);
