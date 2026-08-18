import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/pending_handoff.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeRegistryStorage implements SessionRegistryStorage {
  _FakeRegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

ProviderContainer _container(SessionRegistry initial) => ProviderContainer.test(
  overrides: [
    secureSessionRegistryStorageProvider.overrideWithValue(
      _FakeRegistryStorage(initial),
    ),
    sessionValidationLauncherProvider.overrideWithValue((_) async {}),
  ],
);

void main() {
  test('cold start launches recovery for a durable pending receipt', () async {
    final pending = PendingHandoff(
      token: 'pending-token',
      did: 'did:plc:pending',
      handle: 'pending.test',
      receiptId: '00000000-0000-4000-8000-000000000817',
      confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
    );
    var launches = 0;
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _FakeRegistryStorage(SessionRegistry.empty().stageHandoff(pending)),
        ),
        sessionValidationLauncherProvider.overrideWithValue((_) async {}),
        pendingHandoffRecoveryLauncherProvider.overrideWithValue(() async {
          launches++;
        }),
      ],
    );

    expect(await container.read(authSessionProvider.future), isA<SignedOut>());
    await Future<void>.delayed(Duration.zero);

    expect(launches, 1);
  });

  test('projects SignedOut from an empty registry', () async {
    final container = _container(SessionRegistry.empty());
    expect(await container.read(authSessionProvider.future), isA<SignedOut>());
  });

  test(
    'restores the cached active account without network validation',
    () async {
      final registry = SessionRegistry.empty()
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
      final container = _container(registry);

      final auth = await container.read(authSessionProvider.future);

      expect(auth, isA<SignedIn>());
      expect((auth as SignedIn).did, 'did:plc:bob');
      expect(auth.handle, 'bob.test');
    },
  );

  test(
    'registry invalidation rebuilds the projection to MRU fallback',
    () async {
      final registry = SessionRegistry.empty()
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
      final container = _container(registry);
      await container.read(authSessionProvider.future);
      final bobLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(
            // AccountKey is deliberately constructed through the active lease.
            container
                .read(sessionRegistryProvider)
                .requireValue
                .activeLease!
                .session
                .account,
          )!;

      await container
          .read(sessionRegistryProvider.notifier)
          .invalidate(bobLease);
      final auth = await container.read(authSessionProvider.future);

      expect((auth as SignedIn).did, 'did:plc:alice');
    },
  );

  test(
    'SIM-UT-003 validates retained accounts only when they become active',
    () async {
      final registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-bob',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-alice',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final launches = <AccountKey>[];
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _FakeRegistryStorage(registry),
          ),
          sessionValidationLauncherProvider.overrideWithValue((lease) async {
            launches.add(lease.account);
          }),
        ],
      );
      await container.read(authSessionProvider.future);
      await Future<void>.delayed(Duration.zero);
      expect(launches, [AccountKey('did:plc:alice')]);
      final bobLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;

      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      await container.read(authSessionProvider.future);
      await Future<void>.delayed(Duration.zero);

      expect(launches, [
        AccountKey('did:plc:alice'),
        AccountKey('did:plc:bob'),
      ]);
    },
  );
}
