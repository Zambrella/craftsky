import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/providers/profile_customisation_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
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
    'editor retains draft while saving and adopts authoritative response',
    () async {
      final save = Completer<ProfileCustomisation>();
      var updateCalls = 0;
      final repository = FakeProfileRepository(
        onFetchMe: () async => _profile(),
        onFetch: (_) async => _profile(),
        onUpdateCustomisation: (_) {
          updateCalls += 1;
          return save.future;
        },
      );
      final container = ProviderContainer(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(
              SessionRegistry.empty().upsertAndActivate(
                token: 'token-alice',
                did: 'did:plc:alice',
                handle: 'alice.example',
              ),
            ),
          ),
        ],
      );
      addTearDown(container.dispose);
      await container.read(sessionRegistryProvider.future);
      final profileSubscription = container.listen(
        userProfileProvider('alice.example'),
        (_, _) {},
      );
      addTearDown(profileSubscription.close);
      await container.read(userProfileProvider('alice.example').future);

      final initial = await container.read(
        profileCustomisationEditorProvider.future,
      );
      expect(initial.confirmed, ProfileCustomisation.defaults);
      expect(initial.draft, ProfileCustomisation.defaults);
      expect(initial.isDirty, isFalse);

      final notifier =
          container.read(
              profileCustomisationEditorProvider.notifier,
            )
            ..selectColour('teal')
            ..selectBorder('thick');
      final draft = container.read(profileCustomisationEditorProvider).value!;
      expect(draft.isDirty, isTrue);

      final pending = notifier.save();
      await notifier.save();
      final loading = container.read(profileCustomisationEditorProvider);
      expect(loading.isLoading, isTrue);
      expect(loading.value?.draft, draft.draft);
      expect(updateCalls, 1);

      const authoritative = ProfileCustomisation(
        colour: 'teal',
        border: 'thick',
        background: 'x2',
      );
      save.complete(authoritative);
      await pending;

      final saved = container.read(profileCustomisationEditorProvider);
      expect(saved, isA<AsyncData<ProfileCustomisationEditorState>>());
      expect(saved.value?.confirmed, authoritative);
      expect(saved.value?.draft, authoritative);
      expect(saved.value?.isDirty, isFalse);
      expect(
        container
            .read(sessionRegistryProvider)
            .requireValue
            .activeLease
            ?.session
            .account
            .did,
        'did:plc:alice',
      );
      expect(
        container
            .read(sessionRegistryProvider)
            .requireValue
            .orderedSessions
            .single
            .cachedCustomisation,
        authoritative,
      );
      expect(
        container
            .read(userProfileProvider('alice.example'))
            .value
            ?.customisation,
        authoritative,
      );
    },
  );

  test(
    'editor exposes save error through AsyncValue and preserves draft',
    () async {
      final repository = FakeProfileRepository(
        onFetchMe: () async => _profile(),
        onUpdateCustomisation: (_) async => throw StateError('save failed'),
      );
      final container = ProviderContainer(
        overrides: [profileRepositoryProvider.overrideWithValue(repository)],
      );
      addTearDown(container.dispose);
      await container.read(profileCustomisationEditorProvider.future);

      final notifier = container.read(
        profileCustomisationEditorProvider.notifier,
      )..selectBackground('cubedark');
      final draft = container
          .read(profileCustomisationEditorProvider)
          .value!
          .draft;
      await notifier.save();

      final failed = container.read(profileCustomisationEditorProvider);
      expect(failed.hasError, isTrue);
      expect(failed.isLoading, isFalse);
      expect(failed.value?.draft, draft);
      expect(failed.value?.isDirty, isTrue);
    },
  );

  test('late save updates only its retained initiating account', () async {
    final save = Completer<ProfileCustomisation>();
    final storage = _RegistryStorage(
      SessionRegistry.empty().upsertAndActivate(
        token: 'token-alice',
        did: 'did:plc:alice',
        handle: 'alice.example',
      ),
    );
    final repository = FakeProfileRepository(
      onFetchMe: () async => _profile(),
      onUpdateCustomisation: (_) => save.future,
    );
    final container = ProviderContainer(
      overrides: [
        profileRepositoryProvider.overrideWithValue(repository),
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
      ],
    );
    addTearDown(container.dispose);
    await container.read(sessionRegistryProvider.future);
    await container.read(profileCustomisationEditorProvider.future);

    final notifier = container.read(
      profileCustomisationEditorProvider.notifier,
    )..selectColour('rose');
    final pending = notifier.save();
    await container
        .read(sessionRegistryProvider.notifier)
        .upsertAndActivate(
          token: 'token-bob',
          did: 'did:plc:bob',
          handle: 'bob.example',
        );
    save.complete(const ProfileCustomisation(colour: 'rose'));
    await pending;

    final registry = container.read(sessionRegistryProvider).requireValue;
    expect(registry.activeDid, 'did:plc:bob');
    expect(
      registry.sessions.values
          .singleWhere((session) => session.did.value == 'did:plc:alice')
          .cachedCustomisation
          .colour,
      'rose',
    );
    expect(
      registry.sessions.values
          .singleWhere((session) => session.did.value == 'did:plc:bob')
          .cachedCustomisation,
      ProfileCustomisation.defaults,
    );
  });
}

Profile _profile() => Profile(
  did: 'did:plc:alice',
  handle: 'alice.example',
  crafts: const [],
);
