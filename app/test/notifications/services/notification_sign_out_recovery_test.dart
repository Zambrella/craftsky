import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/notifications/services/notification_sign_out_recovery.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-013 IT-010 persists and drains offline cleanup in order', () async {
    var registry = _registry();
    final alice = registry.leaseFor(AccountKey('did:plc:alice'))!;
    final operations = <String>[];
    var transientFailures = 1;
    final recovery = NotificationSignOutRecovery(
      readRegistry: () => registry,
      quarantineAndRemove: (lease) async {
        operations.add('snapshot-quarantine');
        registry = registry.quarantineAndRemove(lease);
      },
      deleteCleanupCredential: (cleanup) async {
        operations.add('delete-cleanup-credential');
        registry = registry.removePendingCleanup(cleanup);
      },
      deleteProviderToken: () async => operations.add('delete-provider-token'),
      logoutCleanup: (cleanup) async {
        operations.add('deactivate-removed-account');
        if (transientFailures-- > 0) {
          return NotificationCleanupResult.retryable;
        }
        return NotificationCleanupResult.complete;
      },
      resumeRegistration: () async => operations.add('register-b'),
    );

    await recovery.begin(alice);

    expect(registry.sessions.keys, {'did:plc:bob'});
    expect(registry.routingBindings, isNot(contains('did:plc:alice')));
    expect(registry.pendingCleanups, hasLength(1));
    expect(
      registry.pendingCleanups.single.toString(),
      isNot(contains('alice')),
    );
    expect(
      NotificationRoutingStorage(() => registry).resolve(
        // Well-formed stale provider payload after local removal.
        AccountSubscriptionId.parse('alice_binding'),
      ),
      isA<RemovedNotificationRecipient>(),
    );
    expect(operations, ['snapshot-quarantine', 'delete-provider-token']);

    await recovery.retry();
    expect(registry.pendingCleanups, hasLength(1));
    expect(operations, isNot(contains('register-b')));

    await recovery.retry();
    expect(registry.pendingCleanups, isEmpty);
    expect(
      operations.sublist(operations.length - 3),
      [
        'deactivate-removed-account',
        'delete-cleanup-credential',
        'register-b',
      ],
    );
  });

  test('UT-013 unauthorized cleanup is terminal', () async {
    var registry = _registry();
    final alice = registry.leaseFor(AccountKey('did:plc:alice'))!;
    var registrations = 0;
    final recovery = NotificationSignOutRecovery(
      readRegistry: () => registry,
      quarantineAndRemove: (lease) async {
        registry = registry.quarantineAndRemove(lease);
      },
      deleteCleanupCredential: (cleanup) async {
        registry = registry.removePendingCleanup(cleanup);
      },
      deleteProviderToken: () async {},
      logoutCleanup: (_) async => NotificationCleanupResult.alreadyComplete,
      resumeRegistration: () async => registrations++,
    );

    await recovery.begin(alice);
    await recovery.retry();

    expect(registry.pendingCleanups, isEmpty);
    expect(registrations, 1);
  });
}

SessionRegistry _registry() {
  final base = SessionRegistry.empty()
      .upsertAndActivate(
        token: 'bob-secret',
        did: 'did:plc:bob',
        handle: 'bob.test',
      )
      .upsertAndActivate(
        token: 'alice-cleanup-secret',
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
