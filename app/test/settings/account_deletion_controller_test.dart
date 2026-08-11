import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/settings/services/account_deletion_acceptance_coordinator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final entry =
      DeletionStatusEntry.pending(
        jobId: '10000000-0000-0000-0000-000000000001',
        did: 'did:plc:alice',
        handle: 'alice.test',
        statusToken: 'status-token',
      ).withStatus(
        status: AccountDeletionStatus.active,
        phase: AccountDeletionPhase.preparing,
      );

  test('persists status before cleanup and activates MRU fallback', () async {
    var registry = SessionRegistry.empty()
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
    final alice = registry.leaseFor(AccountKey('did:plc:alice'))!;
    registry = registry.activate(alice);
    final fence = AccountDeletionLeaseFence.capture(registry);
    final effects = <String>[];
    final coordinator = AccountDeletionAcceptanceCoordinator(
      readRegistry: () async => registry,
      persistStatus: (_) async => effects.add('persist-status'),
      invalidateActiveState: () async => effects.add('invalidate-active'),
      cleanProductData: (_) async => effects.add('clean-alice'),
      removeOrdinarySession: (lease) async {
        effects.add('remove-alice');
        registry = registry.remove(lease.account.did.value);
      },
      routeAfterActiveRemoval: ({required hasFallback}) async =>
          effects.add('route:$hasFallback'),
    );

    final result = await coordinator.reconcile(
      fence: fence,
      entry: entry,
    );

    expect(result, DeletionAcceptanceResult.activeRemoved);
    expect(effects, [
      'persist-status',
      'invalidate-active',
      'clean-alice',
      'remove-alice',
      'route:true',
    ]);
    expect(registry.activeDid?.value, 'did:plc:bob');
  });

  test('storage failure preserves all ordinary state', () async {
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final fence = AccountDeletionLeaseFence.capture(registry);
    var cleaned = false;
    var removed = false;
    final coordinator = AccountDeletionAcceptanceCoordinator(
      readRegistry: () async => registry,
      persistStatus: (_) async => throw StateError('secure write failed'),
      invalidateActiveState: () async {},
      cleanProductData: (_) async => cleaned = true,
      removeOrdinarySession: (_) async => removed = true,
      routeAfterActiveRemoval: ({required hasFallback}) async {},
    );

    await expectLater(
      coordinator.reconcile(fence: fence, entry: entry),
      throwsStateError,
    );

    expect(cleaned, isFalse);
    expect(removed, isFalse);
    expect(registry.sessions, isNotEmpty);
  });

  test('late Alice acceptance cannot disturb active Bob UI', () async {
    var registry = SessionRegistry.empty()
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
    final alice = registry.leaseFor(AccountKey('did:plc:alice'))!;
    registry = registry.activate(alice);
    final fence = AccountDeletionLeaseFence.capture(registry);
    registry = registry.activate(
      registry.leaseFor(AccountKey('did:plc:bob'))!,
    );
    final effects = <String>[];
    final coordinator = AccountDeletionAcceptanceCoordinator(
      readRegistry: () async => registry,
      persistStatus: (_) async => effects.add('persist-status'),
      invalidateActiveState: () async => effects.add('invalidate-active'),
      cleanProductData: (_) async => effects.add('clean-alice'),
      removeOrdinarySession: (lease) async {
        effects.add('remove-alice');
        registry = registry.remove(lease.account.did.value);
      },
      routeAfterActiveRemoval: ({required hasFallback}) async =>
          effects.add('route'),
    );

    final result = await coordinator.reconcile(
      fence: fence,
      entry: entry,
    );

    expect(result, DeletionAcceptanceResult.inactiveRemoved);
    expect(effects, ['persist-status', 'clean-alice', 'remove-alice']);
    expect(registry.activeDid?.value, 'did:plc:bob');
  });
}
