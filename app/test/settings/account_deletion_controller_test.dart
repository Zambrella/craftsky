import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/settings/providers/account_deletion_controller.dart';
import 'package:craftsky_app/settings/services/account_deletion_acceptance_coordinator.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

final class _DeletionRegistryStorage implements SessionRegistryStorage {
  _DeletionRegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

void main() {
  test(
    'REG-019 controller restores exact-handle confirmation from the durable '
    'pending intent',
    () async {
      final initial = SessionRegistry.empty().upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final storage = _DeletionRegistryStorage(initial);
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.invalid'));
      DioAdapter(dio: dio)
        ..onPost(
          '/v1/account-deletion/intents',
          (server) => server.reply(201, {
            'jobId': '10000000-0000-4000-8000-000000000001',
            'authUrl': 'https://pds.invalid/oauth',
            'expiresAt': DateTime.now()
                .toUtc()
                .add(const Duration(minutes: 10))
                .toIso8601String(),
          }),
        )
        ..onDelete(
          '/v1/account-deletion/intents/'
          '10000000-0000-4000-8000-000000000001',
          (server) => server.reply(204, null),
        );
      final launched = <Uri>[];
      var invalidations = 0;
      ProviderContainer container() => ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          accountDioProvider.overrideWith((ref, account) async => dio),
          authUrlLauncherProvider.overrideWithValue((uri) async {
            launched.add(uri);
            return true;
          }),
          accountStateInvalidatorProvider.overrideWithValue(() async {
            invalidations++;
          }),
        ],
      );
      final first = container();
      await first.read(sessionRegistryProvider.future);

      final jobId = await first
          .read(accountDeletionControllerProvider.notifier)
          .startReauthentication();

      expect(jobId, '10000000-0000-4000-8000-000000000001');
      expect(storage.value.pendingAccountDeletion?.jobId, jobId);
      expect(launched.single, Uri.parse('https://pds.invalid/oauth'));
      expect(invalidations, 1);

      final restored = container();
      await restored.read(sessionRegistryProvider.future);
      final controller = restored.read(
        accountDeletionControllerProvider.notifier,
      );
      expect(controller.canComplete(jobId!), isTrue);
      expect(controller.requiredHandle(jobId), '@alice.test');

      await controller.cancelPendingIntent(jobId);

      expect(storage.value.pendingAccountDeletion, isNull);
      expect(storage.value.sessions.values.single.token, 'alice-token');
    },
  );

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
