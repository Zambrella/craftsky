import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value, {this.blockWrites = false});

  SessionRegistry value;
  final bool blockWrites;
  final writeStarted = Completer<void>();
  final allowWrite = Completer<void>();
  int writes = 0;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async {
    writes++;
    if (!writeStarted.isCompleted) writeStarted.complete();
    if (blockWrites) await allowWrite.future;
    value = registry;
  }
}

SessionRegistry _registryFixture() {
  final registry = SessionRegistry.empty()
      .upsertAndActivate(
        token: 'token-bob',
        did: 'did:plc:bob',
        handle: 'bob.test',
        cachedDisplayName: 'Bob',
      )
      .upsertAndActivate(
        token: 'token-alice',
        did: 'did:plc:alice',
        handle: 'old.example',
        cachedDisplayName: 'Alice',
        cachedAvatarUrl: 'https://cdn.example/alice.jpg',
        cachedCustomisation: const ProfileCustomisation(
          colour: 'orchid',
          border: 'thick',
          background: 'x2',
        ),
      );
  return registry.saveRoutingBinding(
    registry.leaseFor(AccountKey('did:plc:alice'))!,
    'alice-routing-binding',
  );
}

Dio _whoamiDio({required String did, required String handle}) {
  final dio = Dio(BaseOptions(baseUrl: 'https://appview.invalid'));
  DioAdapter(dio: dio).onGet(
    '/v1/whoami',
    (server) => server.reply(200, {'did': did, 'handle': handle}),
  );
  return dio;
}

void main() {
  test(
    'UT-007 matching DID persists a handle-only mutation before success',
    () async {
      final initial = _registryFixture();
      final lease = initial.activeLease!.session;
      final storage = _RegistryStorage(initial, blockWrites: true);
      final publishedHandles = <String>[];
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          accountDioProvider.overrideWith(
            (ref, account) async =>
                _whoamiDio(did: 'did:plc:alice', handle: 'new.example'),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final subscription = container.listen(sessionRegistryProvider, (_, next) {
        final active = next.value?.activeLease?.session.account.did;
        if (active != null) {
          publishedHandles.add(next.value!.sessions[active]!.handle.value);
        }
      });
      addTearDown(subscription.close);
      var validationCompleted = false;

      final validation = container
          .read(sessionValidationLauncherProvider)(lease)
          .whenComplete(() => validationCompleted = true);
      await storage.writeStarted.future;

      expect(storage.value.sessions[lease.account.did]!.handle, 'old.example');
      expect(
        container
            .read(sessionRegistryProvider)
            .requireValue
            .sessions[lease.account.did]!
            .handle,
        'old.example',
      );
      expect(validationCompleted, isFalse);
      expect(publishedHandles, isEmpty);

      storage.allowWrite.complete();
      await validation;

      final current = container.read(sessionRegistryProvider).requireValue;
      final before = initial.sessions[lease.account.did]!;
      final after = current.sessions[lease.account.did]!;
      expect(after.handle, 'new.example');
      expect(after.token, before.token);
      expect(after.did, before.did);
      expect(after.sessionGeneration, before.sessionGeneration);
      expect(after.lastUsedOrdinal, before.lastUsedOrdinal);
      expect(after.cachedDisplayName, before.cachedDisplayName);
      expect(after.cachedAvatarUrl, before.cachedAvatarUrl);
      expect(after.cachedCustomisation, before.cachedCustomisation);
      expect(current.nextSessionGeneration, initial.nextSessionGeneration);
      expect(current.nextUseOrdinal, initial.nextUseOrdinal);
      expect(current.activationGeneration, initial.activationGeneration);
      expect(current.activeDid, initial.activeDid);
      expect(current.routingBindings, initial.routingBindings);
      expect(
        current.orderedSessions.map((session) => session.did).toList(),
        initial.orderedSessions.map((session) => session.did).toList(),
      );
      expect(
        identical(
          current.sessions[AccountKey('did:plc:bob').did],
          initial.sessions[AccountKey('did:plc:bob').did],
        ),
        isTrue,
      );
      expect(storage.writes, 1);
      expect(storage.value.toJson(), current.toJson());
      expect(publishedHandles, ['new.example']);
      expect(validationCompleted, isTrue);
    },
  );

  test(
    'UT-007 stale lease cannot overwrite a replacement generation',
    () async {
      final initial = _registryFixture();
      final staleLease = initial.activeLease!.session;
      final replacement = initial.upsertAndActivate(
        token: 'replacement-token',
        did: staleLease.account.did.value,
        handle: 'replacement.example',
        cachedDisplayName: 'Replacement Alice',
      );
      final storage = _RegistryStorage(replacement);
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          accountDioProvider.overrideWith(
            (ref, account) async => _whoamiDio(
              did: staleLease.account.did.value,
              handle: 'late.example',
            ),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);

      await container.read(sessionValidationLauncherProvider)(staleLease);

      final current = container.read(sessionRegistryProvider).requireValue;
      final session = current.sessions[staleLease.account.did]!;
      expect(session.handle, 'replacement.example');
      expect(session.token, 'replacement-token');
      expect(
        session.sessionGeneration,
        replacement.activeLease!.session.sessionGeneration,
      );
      expect(storage.writes, 0);
    },
  );

  test('UT-007 DID mismatch invalidates only the captured lease', () async {
    final initial = _registryFixture();
    final lease = initial.activeLease!.session;
    final storage = _RegistryStorage(initial);
    final invalidated = <AccountSessionLease>[];
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
        accountDioProvider.overrideWith(
          (ref, account) async =>
              _whoamiDio(did: 'did:plc:bob', handle: 'new.example'),
        ),
        accountSessionInvalidatorProvider.overrideWithValue((captured) async {
          invalidated.add(captured);
        }),
      ],
    );
    await container.read(sessionRegistryProvider.future);

    await container.read(sessionValidationLauncherProvider)(lease);

    expect(invalidated, [lease]);
    expect(
      container.read(sessionRegistryProvider).requireValue.toJson(),
      initial.toJson(),
    );
    expect(storage.writes, 0);
  });
}
