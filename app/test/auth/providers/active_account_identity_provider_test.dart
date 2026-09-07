import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../profile/fakes/fake_profile_repository.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

void main() {
  test('active profile loads by DID when the stored handle is stale', () async {
    var fetchMeCalls = 0;
    final repository = FakeProfileRepository(
      onFetchMe: () async {
        fetchMeCalls++;
        return Profile(
          did: 'did:plc:alice',
          handle: 'alice.new',
          crafts: const [],
        );
      },
      onFetch: (identity) async => throw StateError(
        'stale stored handle must not select own profile: $identity',
      ),
    );
    final storage = _RegistryStorage(
      SessionRegistry.empty().upsertAndActivate(
        token: 'token-alice',
        did: 'did:plc:alice',
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

    final identity = await container.read(activeAccountIdentityProvider.future);

    expect(fetchMeCalls, 1);
    expect(identity?.profile.did, 'did:plc:alice');
    expect(identity?.profile.handle, 'alice.new');
    expect(storage.value.sessions.values.single.handle, 'alice.old');
    expect(storage.value.sessions.values.single.cachedDisplayName, isNull);
  });
}
