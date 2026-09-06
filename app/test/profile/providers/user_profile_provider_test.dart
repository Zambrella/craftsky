import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_profile_repository.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

void main() {
  test(
    'profiles with the same historical handle retain distinct DID state',
    () async {
      final aliceDid = Did.parse('did:plc:alice');
      final bobDid = Did.parse('did:plc:bob');
      final repository = FakeProfileRepository(
        onFetch: (did) async => switch (did) {
          'did:plc:alice' => _profile(did: did, handle: 'shared.example'),
          'did:plc:bob' => _profile(did: did, handle: 'shared.example'),
          _ => throw StateError('unexpected identity: $did'),
        },
      );
      final container = ProviderContainer.test(
        overrides: [profileRepositoryProvider.overrideWithValue(repository)],
      );

      final aliceSubscription = container.listen(
        userProfileProvider(aliceDid),
        (_, _) {},
      );
      final bobSubscription = container.listen(
        userProfileProvider(bobDid),
        (_, _) {},
      );
      addTearDown(aliceSubscription.close);
      addTearDown(bobSubscription.close);

      await Future.wait([
        container.read(userProfileProvider(aliceDid).future),
        container.read(userProfileProvider(bobDid).future),
      ]);
      container
          .read(userProfileProvider(aliceDid).notifier)
          .setCached(_profile(did: aliceDid, handle: 'alice.new'));

      expect(
        container.read(userProfileProvider(aliceDid)).requireValue.handle,
        'alice.new',
      );
      expect(
        container.read(userProfileProvider(bobDid)).requireValue.handle,
        'shared.example',
      );
    },
  );

  test(
    'active DID uses fetchMe and a handle update keeps one profile state',
    () async {
      final did = Did.parse('did:plc:alice');
      var fetchMeCalls = 0;
      final repository = FakeProfileRepository(
        onFetchMe: () async {
          fetchMeCalls++;
          return _profile(did: did, handle: 'alice.new');
        },
        onFetch: (identity) async => throw StateError(
          'active profile must not use generic fetch: $identity',
        ),
      );
      final storage = _RegistryStorage(
        SessionRegistry.empty().upsertAndActivate(
          token: 'token-alice',
          did: did,
          handle: 'alice.old',
        ),
      );
      final container = ProviderContainer.test(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final subscription = container.listen(
        userProfileProvider(did),
        (_, _) {},
      );
      addTearDown(subscription.close);

      final loaded = await container.read(userProfileProvider(did).future);
      container
          .read(userProfileProvider(did).notifier)
          .setCached(loaded.copyWith(handle: 'alice.latest'));

      expect(fetchMeCalls, 1);
      expect(
        container.read(userProfileProvider(did)).requireValue.handle,
        'alice.latest',
      );
    },
  );
}

Profile _profile({required String did, required String handle}) =>
    Profile(did: did, handle: handle, crafts: const []);
