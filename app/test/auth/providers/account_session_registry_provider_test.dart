import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/pending_account_deletion.dart';
import 'package:craftsky_app/auth/models/pending_handoff.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeRegistryStorage implements SessionRegistryStorage {
  _FakeRegistryStorage(this.value);

  SessionRegistry value;
  bool failWrites = false;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async {
    if (failWrites) {
      throw const SessionRegistryStorageException('writeFailed');
    }
    value = registry;
  }
}

void main() {
  test(
    'REG-020 cancellation retains the session and accepted removal clears '
    'pending deletion',
    () async {
      final original = SessionRegistry.empty().upsertAndActivate(
        token: 'token-alice',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final storage = _FakeRegistryStorage(original);
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final pending = PendingAccountDeletion.capture(
        jobId: '10000000-0000-4000-8000-000000000001',
        lease: original.activeLease!,
        handle: 'alice.test',
        expiresAt: DateTime.utc(2027),
      );

      await container
          .read(sessionRegistryProvider.notifier)
          .stageAccountDeletion(pending);
      await container
          .read(sessionRegistryProvider.notifier)
          .clearAccountDeletion(pending.jobId);

      expect(storage.value.pendingAccountDeletion, isNull);
      expect(
        storage.value.leaseFor(pending.lease.session.account),
        pending.lease.session,
      );

      await container
          .read(sessionRegistryProvider.notifier)
          .stageAccountDeletion(pending);
      await container
          .read(sessionRegistryProvider.notifier)
          .removeConfirmed(pending.lease.session);

      expect(storage.value.pendingAccountDeletion, isNull);
      expect(storage.value.sessions, isEmpty);
    },
  );

  test(
    'persists pending receipt before publishing a confirmed session',
    () async {
      final storage = _FakeRegistryStorage(SessionRegistry.empty());
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final pending = PendingHandoff(
        token: 'pending-bearer',
        did: 'did:plc:alice',
        handle: 'alice.test',
        receiptId: '00000000-0000-4000-8000-000000000811',
        confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
      );

      await container
          .read(sessionRegistryProvider.notifier)
          .stageHandoff(pending);

      expect(storage.value.pendingHandoff?.receiptId, pending.receiptId);
      expect(storage.value.sessions, isEmpty);
      expect(
        container.read(sessionRegistryProvider).requireValue.sessions,
        isEmpty,
      );

      await container
          .read(sessionRegistryProvider.notifier)
          .confirmHandoff(pending.receiptId);

      expect(storage.value.pendingHandoff, isNull);
      expect(storage.value.activeDid, 'did:plc:alice');
      expect(storage.value.sessions['did:plc:alice']?.token, 'pending-bearer');
    },
  );

  test('restores all retained sessions and the active account', () async {
    final stored = SessionRegistry.empty()
        .upsertAndActivate(
          token: 'token-alice',
          did: 'did:plc:alice',
          handle: 'alice.test',
        )
        .upsertAndActivate(
          token: 'token-bob',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );
    final storage = _FakeRegistryStorage(stored);
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
      ],
    );

    final restored = await container.read(sessionRegistryProvider.future);

    expect(restored.activeDid, 'did:plc:bob');
    expect(restored.sessions.keys, {'did:plc:alice', 'did:plc:bob'});
  });

  test('publishes a mutation only after verified storage succeeds', () async {
    final original = SessionRegistry.empty().upsertAndActivate(
      token: 'token-alice',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final storage = _FakeRegistryStorage(original)..failWrites = true;
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
      ],
    );
    await container.read(sessionRegistryProvider.future);

    await expectLater(
      container
          .read(sessionRegistryProvider.notifier)
          .upsertAndActivate(
            token: 'token-bob',
            did: 'did:plc:bob',
            handle: 'bob.test',
          ),
      throwsA(isA<SessionRegistryStorageException>()),
    );

    final current = container.read(sessionRegistryProvider).requireValue;
    expect(current.toJson(), original.toJson());
    expect(current.sessions.keys, {'did:plc:alice'});
    expect(storage.value.sessions.keys, {'did:plc:alice'});
  });

  test('persists a binding only for the unchanged session lease', () async {
    final original = SessionRegistry.empty().upsertAndActivate(
      token: 'token-alice',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final storage = _FakeRegistryStorage(original);
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
      ],
    );
    await container.read(sessionRegistryProvider.future);
    final lease = original.leaseFor(AccountKey('did:plc:alice'))!;

    await container
        .read(sessionRegistryProvider.notifier)
        .saveRoutingBinding(
          lease,
          AccountSubscriptionId.parse('alice_binding'),
        );

    expect(
      storage.value.routingBindings[lease.account.did],
      'alice_binding',
    );
    await container
        .read(sessionRegistryProvider.notifier)
        .removeConfirmed(lease);
    expect(storage.value.routingBindings, isEmpty);
  });
}
