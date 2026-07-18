import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final aliceBinding = AccountSubscriptionId.parse('alice_binding');

  group('UT-006 registry routing resolution', () {
    test('classifies exact, ambiguous, invalid, and removed bindings', () {
      final exactRegistry = _registry(
        bindings: const {
          'did:plc:alice': 'alice_binding',
          'did:plc:bob': 'bob_binding',
        },
      );
      final exact = NotificationRoutingStorage(() => exactRegistry).resolve(
        aliceBinding,
      );

      expect(exact, isA<ExactNotificationRecipient>());
      expect(
        (exact as ExactNotificationRecipient).lease,
        exactRegistry.leaseFor(AccountKey('did:plc:alice')),
      );

      final ambiguous = NotificationRoutingStorage(
        () => _registry(
          bindings: const {
            'did:plc:alice': 'alice_binding',
            'did:plc:bob': 'alice_binding',
          },
        ),
      );
      expect(
        ambiguous.resolve(aliceBinding),
        isA<InvalidNotificationRecipient>(),
      );
      expect(ambiguous.resolve(null), isA<InvalidNotificationRecipient>());

      final removed = NotificationRoutingStorage(() => exactRegistry).resolve(
        AccountSubscriptionId.parse('removed_binding'),
      );
      expect(removed, isA<RemovedNotificationRecipient>());

      final nonRetainedBinding = NotificationRoutingStorage(
        () => _registry(
          bindings: const {
            'did:plc:alice': 'alice_binding',
            'did:plc:removed': 'removed_binding',
          },
        ),
      );
      expect(
        nonRetainedBinding.resolve(
          AccountSubscriptionId.parse('removed_binding'),
        ),
        isA<RemovedNotificationRecipient>(),
      );
    });
  });
}

SessionRegistry _registry({required Map<String, String> bindings}) {
  final base = SessionRegistry.empty()
      .upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      )
      .upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
      );
  return _withBindings(base, bindings);
}

SessionRegistry _withBindings(
  SessionRegistry registry,
  Map<String, String> bindings,
) => SessionRegistry(
  nextSessionGeneration: registry.nextSessionGeneration,
  nextUseOrdinal: registry.nextUseOrdinal,
  activationGeneration: registry.activationGeneration,
  activeDid: registry.activeDid?.value,
  sessions: {
    for (final entry in registry.sessions.entries) entry.key.value: entry.value,
  },
  routingBindings: bindings,
);
