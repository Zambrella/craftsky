import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/settings/services/account_deletion_acceptance_coordinator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'cleans the account and activates the MRU fallback without status state',
    () async {
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
      registry = registry.activate(
        registry.leaseFor(AccountKey('did:plc:alice'))!,
      );
      final fence = AccountDeletionLeaseFence.capture(registry);
      final effects = <String>[];
      final coordinator = AccountDeletionAcceptanceCoordinator(
        readRegistry: () async => registry,
        invalidateActiveState: () async => effects.add('invalidate-active'),
        cleanProductData: (_) async => effects.add('clean-alice'),
        removeOrdinarySession: (lease) async {
          effects.add('remove-alice');
          registry = registry.remove(lease.account.did.value);
        },
        routeAfterActiveRemoval: ({required hasFallback}) async =>
            effects.add('route:$hasFallback'),
      );

      final result = await coordinator.reconcile(fence: fence);

      expect(result, DeletionAcceptanceResult.activeRemoved);
      expect(effects, [
        'invalidate-active',
        'clean-alice',
        'remove-alice',
        'route:true',
      ]);
      expect(registry.activeDid?.value, 'did:plc:bob');
    },
  );

  test('local cleanup failure still removes the accepted account', () async {
    var registry = SessionRegistry.empty().upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final fence = AccountDeletionLeaseFence.capture(registry);
    var routed = false;
    final coordinator = AccountDeletionAcceptanceCoordinator(
      readRegistry: () async => registry,
      invalidateActiveState: () async {},
      cleanProductData: (_) async => throw StateError('cache cleanup failed'),
      removeOrdinarySession: (lease) async {
        registry = registry.remove(lease.account.did.value);
      },
      routeAfterActiveRemoval: ({required hasFallback}) async => routed = true,
    );

    await expectLater(coordinator.reconcile(fence: fence), throwsStateError);
    expect(registry.sessions, isEmpty);
    expect(routed, isTrue);
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
    registry = registry.activate(
      registry.leaseFor(AccountKey('did:plc:alice'))!,
    );
    final fence = AccountDeletionLeaseFence.capture(registry);
    registry = registry.activate(
      registry.leaseFor(AccountKey('did:plc:bob'))!,
    );
    final effects = <String>[];
    final coordinator = AccountDeletionAcceptanceCoordinator(
      readRegistry: () async => registry,
      invalidateActiveState: () async => effects.add('invalidate-active'),
      cleanProductData: (_) async => effects.add('clean-alice'),
      removeOrdinarySession: (lease) async {
        effects.add('remove-alice');
        registry = registry.remove(lease.account.did.value);
      },
      routeAfterActiveRemoval: ({required hasFallback}) async =>
          effects.add('route'),
    );

    final result = await coordinator.reconcile(fence: fence);

    expect(result, DeletionAcceptanceResult.inactiveRemoved);
    expect(effects, ['clean-alice', 'remove-alice']);
    expect(registry.activeDid?.value, 'did:plc:bob');
  });
}
