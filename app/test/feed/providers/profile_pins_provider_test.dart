import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/profile_pin_state.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/profile_pins_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

void main() {
  setUpAll(initializeMappers);

  test(
    'UT-003 keeps confirmed state while pending and isolates slots',
    () async {
      const standardA =
          'at://did:plc:alice/social.craftsky.feed.post/standard-a';
      const standardB =
          'at://did:plc:alice/social.craftsky.feed.post/standard-b';
      const projectA = 'at://did:plc:alice/social.craftsky.feed.post/project-a';
      final standardPin = Completer<ProfilePinState>();
      final projectPin = Completer<ProfilePinState>();
      var pinCalls = 0;
      var unpinAction = (Did _, RecordKey _) =>
          Future<ProfilePinState>.error(StateError('not configured'));
      final repository = FakePostRepository(
        onProfilePins: () async => const ProfilePinState(
          standardPostUri: standardA,
        ),
        onPin: (did, rkey) {
          pinCalls++;
          return rkey.value == 'project-a'
              ? projectPin.future
              : standardPin.future;
        },
        onUnpin: (did, rkey) => unpinAction(did, rkey),
      );
      final lease = ActiveAccountLease(
        session: AccountSessionLease(
          account: AccountKey('did:plc:alice'),
          sessionGeneration: 1,
        ),
        activationGeneration: 1,
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(repository)],
      );
      final provider = profilePinsProvider(lease);
      final initial = await container.read(provider.future);
      expect(initial.confirmed.standardPostUri, standardA);
      expect(initial.pendingSlots, isEmpty);

      final standardFuture = container
          .read(provider.notifier)
          .pin(
            did: Did.parse('did:plc:alice'),
            rkey: RecordKey.parse('standard-b'),
            slot: ProfilePinSlot.standard,
          );
      var current = container.read(provider).requireValue;
      expect(current.confirmed.standardPostUri, standardA);
      expect(current.confirmed.projectPostUri, isNull);
      expect(current.pendingSlots, {ProfilePinSlot.standard});

      final suppressed = await container
          .read(provider.notifier)
          .pin(
            did: Did.parse('did:plc:alice'),
            rkey: RecordKey.parse('standard-c'),
            slot: ProfilePinSlot.standard,
          );
      expect(suppressed, isNull);
      expect(pinCalls, 1);

      final projectFuture = container
          .read(provider.notifier)
          .pin(
            did: Did.parse('did:plc:alice'),
            rkey: RecordKey.parse('project-a'),
            slot: ProfilePinSlot.project,
          );
      current = container.read(provider).requireValue;
      expect(current.confirmed.standardPostUri, standardA);
      expect(current.pendingSlots, {
        ProfilePinSlot.standard,
        ProfilePinSlot.project,
      });
      expect(pinCalls, 2);

      projectPin.complete(
        const ProfilePinState(
          standardPostUri: standardA,
          projectPostUri: projectA,
        ),
      );
      final projectOutcome = await projectFuture;
      expect(projectOutcome, ProfilePinMutationOutcome.pinned);
      expect(projectOutcome?.message, 'Post pinned');
      current = container.read(provider).requireValue;
      expect(current.confirmed.standardPostUri, standardA);
      expect(current.confirmed.projectPostUri, projectA);
      expect(current.pendingSlots, {ProfilePinSlot.standard});

      standardPin.complete(
        const ProfilePinState(
          standardPostUri: standardB,
          projectPostUri: projectA,
        ),
      );
      final standardOutcome = await standardFuture;
      expect(standardOutcome, ProfilePinMutationOutcome.pinned);
      current = container.read(provider).requireValue;
      expect(current.confirmed.standardPostUri, standardB);
      expect(current.confirmed.projectPostUri, projectA);
      expect(current.pendingSlots, isEmpty);

      unpinAction = (_, _) => Future<ProfilePinState>.error(
        StateError('offline'),
      );
      final failedUnpin = await container
          .read(provider.notifier)
          .unpin(
            did: Did.parse('did:plc:alice'),
            rkey: RecordKey.parse('standard-b'),
            slot: ProfilePinSlot.standard,
          );
      expect(failedUnpin, ProfilePinMutationOutcome.unpinFailed);
      expect(failedUnpin?.message, 'Couldn’t unpin post. Try again.');
      current = container.read(provider).requireValue;
      expect(current.confirmed.standardPostUri, standardB);
      expect(current.confirmed.projectPostUri, projectA);
      expect(current.pendingSlots, isEmpty);

      unpinAction = (_, _) async => const ProfilePinState(
        projectPostUri: projectA,
      );
      final successfulUnpin = await container
          .read(provider.notifier)
          .unpin(
            did: Did.parse('did:plc:alice'),
            rkey: RecordKey.parse('standard-b'),
            slot: ProfilePinSlot.standard,
          );
      expect(successfulUnpin, ProfilePinMutationOutcome.unpinned);
      expect(successfulUnpin?.message, 'Post unpinned');
      current = container.read(provider).requireValue;
      expect(current.confirmed.standardPostUri, isNull);
      expect(current.confirmed.projectPostUri, projectA);
      expect(current.pendingSlots, isEmpty);
    },
  );

  test(
    'IT-016 refreshes only the affected live profile-list families',
    () async {
      const did = 'did:plc:alice';
      const handle = 'alice.test';
      const pinnedUri =
          'at://did:plc:alice/social.craftsky.feed.post/standard-b';
      final standardCalls = <String, int>{};
      final projectCalls = <String, int>{};
      final repository = FakePostRepository(
        onProfilePins: () async => const ProfilePinState(),
        onPin: (_, _) async => const ProfilePinState(
          standardPostUri: pinnedUri,
        ),
        onListByAuthor: (id, {cursor, limit}) async {
          standardCalls.update(id, (count) => count + 1, ifAbsent: () => 1);
          return const PostPage(items: []);
        },
        onListProjectsByAuthor: (id, {cursor, limit}) async {
          projectCalls.update(id, (count) => count + 1, ifAbsent: () => 1);
          return const PostPage(items: []);
        },
      );
      final lease = ActiveAccountLease(
        session: AccountSessionLease(
          account: AccountKey(did),
          sessionGeneration: 1,
        ),
        activationGeneration: 1,
      );
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(repository),
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
        ],
      );
      final standardDid = userPostsProvider(did);
      final standardHandle = userPostsProvider(handle);
      final projectsHandle = userProjectsProvider(handle);
      final subscriptions = [
        container.listen(standardDid, (_, _) {}),
        container.listen(standardHandle, (_, _) {}),
        container.listen(projectsHandle, (_, _) {}),
      ];
      addTearDown(() {
        for (final subscription in subscriptions) {
          subscription.close();
        }
      });
      await Future.wait([
        container.read(standardDid.future),
        container.read(standardHandle.future),
        container.read(projectsHandle.future),
      ]);
      final pins = profilePinsProvider(lease);
      await container.read(pins.future);

      final outcome = await container
          .read(pins.notifier)
          .pin(
            did: Did.parse(did),
            rkey: RecordKey.parse('standard-b'),
            slot: ProfilePinSlot.standard,
            authorCacheIds: const [did, handle],
          );
      await Future.wait([
        container.read(standardDid.future),
        container.read(standardHandle.future),
      ]);

      expect(outcome, ProfilePinMutationOutcome.pinned);
      expect(
        container.read(pins).requireValue.confirmed.standardPostUri,
        pinnedUri,
      );
      expect(standardCalls, {did: 2, handle: 2});
      expect(projectCalls, {handle: 1});
    },
  );

  test(
    'UT-006 fences a late mutation completion after account switch',
    () async {
      const accountAState = ProfilePinState(
        standardPostUri:
            'at://did:plc:alice/social.craftsky.feed.post/standard-a',
      );
      const lateAState = ProfilePinState(
        standardPostUri:
            'at://did:plc:alice/social.craftsky.feed.post/standard-b',
      );
      final pending = Completer<ProfilePinState>();
      final initialRegistry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final repository = FakePostRepository(
        onProfilePins: () async => accountAState,
        onPin: (_, _) => pending.future,
      );
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initialRegistry),
          ),
          postRepositoryProvider.overrideWithValue(repository),
        ],
      );
      final registry = await container.read(sessionRegistryProvider.future);
      final accountALease = registry.activeLease!;
      final providerA = profilePinsProvider(accountALease);
      await container.read(providerA.future);

      final mutation = container
          .read(providerA.notifier)
          .pin(
            did: Did.parse('did:plc:alice'),
            rkey: RecordKey.parse('standard-b'),
            slot: ProfilePinSlot.standard,
          );
      expect(container.read(providerA).requireValue.confirmed, accountAState);
      expect(container.read(providerA).requireValue.pendingSlots, {
        ProfilePinSlot.standard,
      });

      final bobLease = registry.leaseFor(AccountKey('did:plc:bob'))!;
      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      pending.complete(lateAState);

      expect(await mutation, ProfilePinMutationOutcome.staleCompletion);
      expect(container.read(providerA).requireValue.confirmed, accountAState);
      expect(
        container.read(sessionRegistryProvider).requireValue.activeDid?.value,
        'did:plc:bob',
      );
    },
  );
}
