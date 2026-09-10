import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
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
  _RegistryStorage(this.value);

  SessionRegistry value;
  int writes = 0;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async {
    writes++;
    value = registry;
  }
}

void main() {
  test(
    'AT-002 cold start reactively persists a same-DID handle change',
    () async {
      var initial = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'other-token',
            did: 'did:plc:other',
            handle: 'other.example',
          )
          .upsertAndActivate(
            token: 'craftsky-token',
            did: 'did:plc:member',
            handle: 'old.example',
            cachedDisplayName: 'Needle Friend',
            cachedAvatarUrl: 'https://cdn.example/member.jpg',
            cachedCustomisation: const ProfileCustomisation(
              colour: 'teal',
              background: 'cubedark',
            ),
          );
      initial = initial.saveRoutingBinding(
        initial.activeLease!.session,
        'member-routing-binding',
      );
      final storage = _RegistryStorage(initial);
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.invalid'));
      DioAdapter(dio: dio).onGet(
        '/v1/whoami',
        (server) => server.reply(200, {
          'did': 'did:plc:member',
          'handle': 'new.example',
        }),
      );
      final validationFinished = Completer<void>();
      final observedHandles = <String>[];
      late ProviderContainer container;
      container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          accountDioProvider.overrideWith((ref, account) async => dio),
          sessionValidationLauncherProvider.overrideWith((ref) {
            final launcher = AppSessionValidationLauncher(ref).run;
            return (lease) async {
              await launcher(lease);
              validationFinished.complete();
            };
          }),
        ],
      );
      final subscription = container.listen(authSessionProvider, (_, next) {
        if (next case AsyncData(value: SignedIn(:final handle))) {
          observedHandles.add(handle);
        }
      });
      addTearDown(subscription.close);

      final restored = await container.read(authSessionProvider.future);

      expect(restored, isA<SignedIn>());
      expect((restored as SignedIn).handle, 'old.example');
      expect(validationFinished.isCompleted, isFalse);

      await validationFinished.future;
      final refreshed = await container.read(authSessionProvider.future);
      final current = container.read(sessionRegistryProvider).requireValue;
      final before = initial.sessions[initial.activeDid]!;
      final after = current.sessions[current.activeDid]!;

      expect((refreshed as SignedIn).handle, 'new.example');
      expect(observedHandles, ['old.example', 'new.example']);
      expect(storage.writes, 1);
      expect(storage.value.toJson(), current.toJson());
      expect(after.handle, 'new.example');
      expect(after.did, before.did);
      expect(after.token, before.token);
      expect(after.sessionGeneration, before.sessionGeneration);
      expect(after.lastUsedOrdinal, before.lastUsedOrdinal);
      expect(after.cachedDisplayName, before.cachedDisplayName);
      expect(after.cachedAvatarUrl, before.cachedAvatarUrl);
      expect(after.cachedCustomisation, before.cachedCustomisation);
      expect(current.activeDid, initial.activeDid);
      expect(current.activationGeneration, initial.activationGeneration);
      expect(current.nextSessionGeneration, initial.nextSessionGeneration);
      expect(current.nextUseOrdinal, initial.nextUseOrdinal);
      expect(current.routingBindings, initial.routingBindings);
      expect(
        current.orderedSessions.map((session) => session.did).toList(),
        initial.orderedSessions.map((session) => session.did).toList(),
      );
    },
  );
}
