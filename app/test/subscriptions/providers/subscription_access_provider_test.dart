import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  test(
    'IT-002 inactive account access uses its lease without activation',
    () async {
      final registry = _registry();
      final bob = registry.leaseFor(AccountKey('did:plc:bob'))!;
      final api = _AccessApi(
        Future.value(_access('did:plc:bob', SubscriptionTier.business)),
      );
      final container = _container(registry, api);
      addTearDown(container.dispose);

      final access = await container.read(
        subscriptionAccessProvider(bob).future,
      );

      expect(access.effectiveTier, SubscriptionTier.business);
      expect(
        container.read(sessionRegistryProvider).requireValue.activeDid,
        Did.parse('did:plc:alice'),
      );
      expect(api.accessCalls, 1);
      expect(api.ownerCalls, 0);
    },
  );

  test(
    'IT-002 replaced lease cannot publish a delayed access result',
    () async {
      final registry = _registry();
      final bob = registry.leaseFor(AccountKey('did:plc:bob'))!;
      final delayed = Completer<SubscriptionAccess>();
      final api = _AccessApi(delayed.future);
      final container = _container(registry, api);
      addTearDown(container.dispose);

      final result = container.read(subscriptionAccessProvider(bob).future);
      await Future<void>.delayed(Duration.zero);
      await container
          .read(sessionRegistryProvider.notifier)
          .upsertAndActivate(
            token: 'replacement',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      delayed.complete(_access('did:plc:bob', SubscriptionTier.plus));

      await expectLater(result, throwsStateError);
    },
  );

  test('IT-002 access response DID must match the requested lease', () async {
    final registry = _registry();
    final bob = registry.leaseFor(AccountKey('did:plc:bob'))!;
    final api = _AccessApi(
      Future.value(_access('did:plc:alice', SubscriptionTier.business)),
    );
    final container = _container(registry, api);
    addTearDown(container.dispose);

    await expectLater(
      container.read(subscriptionAccessProvider(bob).future),
      throwsStateError,
    );
    expect(api.accessCalls, 1);
    expect(api.ownerCalls, 0);
  });

  test('IT-002 self-access uses the requested account Dio family', () async {
    final registry = _registry();
    final bob = registry.leaseFor(AccountKey('did:plc:bob'))!;
    final requestedAccounts = <AccountKey>[];
    final bobDio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
    DioAdapter(dio: bobDio).onGet(
      '/v1/subscriptions/access',
      (server) => server.reply(200, {
        'did': 'did:plc:bob',
        'effectiveTier': 'business',
        'givesAccess': true,
        'assignedTier': 'business',
      }),
    );
    final container = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(registry),
        ),
        accountDioProvider.overrideWith((ref, account) async {
          requestedAccounts.add(account);
          return bobDio;
        }),
      ],
    );
    addTearDown(container.dispose);
    final listener = container.listen(
      subscriptionAccessProvider(bob),
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(listener.close);

    final access = await container.read(
      subscriptionAccessProvider(bob).future,
    );

    expect(access.did, Did.parse('did:plc:bob'));
    expect(requestedAccounts, [bob.account]);
    expect(registry.activeDid, Did.parse('did:plc:alice'));
  });
}

ProviderContainer _container(SessionRegistry registry, _AccessApi api) =>
    ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _RegistryStorage(registry),
        ),
        subscriptionRepositoryProvider.overrideWith(
          (ref, account) async => api,
        ),
      ],
    );

SessionRegistry _registry() => SessionRegistry.empty()
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

SubscriptionAccess _access(String did, SubscriptionTier tier) =>
    SubscriptionAccess(
      did: Did.parse(did),
      effectiveTier: tier,
      givesAccess: tier != SubscriptionTier.free,
    );

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.registry);
  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async =>
      this.registry = registry;
}

final class _AccessApi implements SubscriptionApi {
  _AccessApi(this.result);
  final Future<SubscriptionAccess> result;
  int accessCalls = 0;
  int ownerCalls = 0;

  @override
  Future<SubscriptionAccess> getAccess() {
    accessCalls++;
    return result;
  }

  @override
  Future<BillingState> getBillingAccount() {
    ownerCalls++;
    throw UnimplementedError();
  }

  @override
  Future<BillingAssignment> assign(String licenseId, String targetDid) =>
      throw UnimplementedError();

  @override
  Future<BillingState> ensureBillingAccount() => throw UnimplementedError();

  @override
  Future<void> requestReconciliation() => throw UnimplementedError();

  @override
  Future<void> unassign(String licenseId) => throw UnimplementedError();
}
