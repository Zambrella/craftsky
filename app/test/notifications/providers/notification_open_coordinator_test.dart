import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_open_coordinator.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-006 exact recipient activates before destination inference',
    () async {
      var registry = _registry();
      final bobLease = registry.leaseFor(AccountKey('did:plc:bob'))!;
      final operations = <String>[];
      var unavailable = 0;
      var removed = 0;
      final coordinator = NotificationOpenCoordinator(
        resolveRecipient: NotificationRoutingStorage(() => registry).resolve,
        isCurrentLease: (lease) => registry.leaseFor(lease.account) == lease,
        activate: (lease) async {
          expect(lease, bobLease);
          operations.add('activate');
          registry = registry.activate(lease);
          return AccountActivationResult.activated;
        },
        onOutcome: (_) => operations.add('navigate'),
        onUnavailable: () => unavailable++,
        onRemovedAccount: () => removed++,
      );

      await coordinator.open(_attempt('bob_binding'));
      expect(operations, ['activate', 'navigate']);
      expect(registry.activeDid?.value, 'did:plc:bob');

      await coordinator.open(_attempt(null));
      await coordinator.open(_attempt('removed_binding'));
      expect(unavailable, 1);
      expect(removed, 1);
    },
  );

  test('UT-007 reauthentication fences an in-flight recipient lease', () async {
    var registry = _registry();
    final activationStarted = Completer<void>();
    final releaseActivation = Completer<void>();
    var navigations = 0;
    var removed = 0;
    final coordinator = NotificationOpenCoordinator(
      resolveRecipient: NotificationRoutingStorage(() => registry).resolve,
      isCurrentLease: (lease) => registry.leaseFor(lease.account) == lease,
      activate: (lease) async {
        activationStarted.complete();
        await releaseActivation.future;
        return AccountActivationResult.activated;
      },
      onOutcome: (_) => navigations++,
      onUnavailable: () {},
      onRemovedAccount: () => removed++,
    );

    final open = coordinator.open(_attempt('bob_binding'));
    await activationStarted.future;
    registry = registry.upsertAndActivate(
      token: 'bob-reauthed-token',
      did: 'did:plc:bob',
      handle: 'bob.test',
    );
    releaseActivation.complete();
    await open;

    expect(navigations, 0);
    expect(removed, 1);
  });
}

SessionRegistry _registry() {
  final base = SessionRegistry.empty()
      .upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
      )
      .upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
  return SessionRegistry(
    revision: base.revision,
    nextSessionGeneration: base.nextSessionGeneration,
    nextUseOrdinal: base.nextUseOrdinal,
    activationGeneration: base.activationGeneration,
    activeDid: base.activeDid?.value,
    sessions: {
      for (final entry in base.sessions.entries) entry.key.value: entry.value,
    },
    routingBindings: const {
      'did:plc:alice': 'alice_binding',
      'did:plc:bob': 'bob_binding',
    },
  );
}

NotificationOpenAttempt _attempt(String? binding) =>
    NotificationOpenAttempt.fromProviderData({
      'payloadVersion': '1',
      'type': 'everythingElse',
      'accountSubscriptionId': ?binding,
    });
