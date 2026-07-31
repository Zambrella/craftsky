import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _PreferencesRepository implements LanguagePreferencesRepository {
  _PreferencesRepository(this.loaded);

  final Future<LanguagePreferences> loaded;

  @override
  Future<LanguagePreferences> initialize(
    LanguagePreferences proposal,
  ) => loaded;

  @override
  Future<LanguagePreferences> load() => loaded;

  @override
  Future<LanguagePreferences> replace(
    LanguagePreferences preferences,
  ) async => preferences;
}

final class _SequencedPreferencesRepository
    implements LanguagePreferencesRepository {
  _SequencedPreferencesRepository(this.loads);

  final List<Completer<LanguagePreferences>> loads;
  var _nextLoad = 0;

  @override
  Future<LanguagePreferences> initialize(
    LanguagePreferences proposal,
  ) => load();

  @override
  Future<LanguagePreferences> load() => loads[_nextLoad++].future;

  @override
  Future<LanguagePreferences> replace(
    LanguagePreferences preferences,
  ) async => preferences;
}

void main() {
  test(
    'signed-out initialization resolves to null without loading preferences',
    () async {
      var repositoryBuilds = 0;
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(SessionRegistry.empty()),
          ),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, account) {
              repositoryBuilds++;
              throw StateError('signed-out initialization requested $account');
            },
          ),
        ],
      );
      final subscription = container.listen(
        activeAccountInitializationProvider,
        (_, _) {},
      );
      addTearDown(subscription.close);

      expect(
        await container.read(activeAccountInitializationProvider.future),
        isNull,
      );
      expect(repositoryBuilds, 0);
    },
  );

  test(
    'new login and switch reject the prior active lease completion',
    () async {
      const alicePreferences = LanguagePreferences(
        primaryLanguage: 'en',
        contentLanguages: ['en'],
      );
      const bobPreferences = LanguagePreferences(
        primaryLanguage: 'cy',
        contentLanguages: ['cy'],
      );
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final aliceLoad = Completer<LanguagePreferences>();
      final bobLoad = Completer<LanguagePreferences>();
      final storage = _RegistryStorage(
        SessionRegistry.empty().upsertAndActivate(
          token: 'token-a',
          did: alice.did.value,
          handle: 'alice.test',
        ),
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, account) async => _PreferencesRepository(
              account == alice ? aliceLoad.future : bobLoad.future,
            ),
          ),
        ],
      );
      final subscription = container.listen(
        activeAccountInitializationProvider,
        (_, _) {},
      );
      addTearDown(subscription.close);

      await container.read(sessionRegistryProvider.future);
      await container
          .read(sessionRegistryProvider.notifier)
          .upsertAndActivate(
            token: 'token-b',
            did: bob.did.value,
            handle: 'bob.test',
          );
      await Future<void>.delayed(Duration.zero);

      aliceLoad.complete(alicePreferences);
      await Future<void>.delayed(Duration.zero);
      expect(
        container.read(activeAccountInitializationProvider).hasValue,
        false,
      );

      bobLoad.complete(bobPreferences);
      final initialized = await container.read(
        activeAccountInitializationProvider.future,
      );
      expect(initialized?.lease, storage.value.activeLease);
      expect(initialized?.lease.session.account, bob);
      expect(initialized?.languagePreferences, bobPreferences);
      expect(container.read(activeLanguagePreferencesProvider), bobPreferences);
      expect(container.read(activeContentLanguagePolicyProvider), const ['cy']);
    },
  );

  test(
    'restored, switched, reactivated, and fallback leases initialize exactly',
    () async {
      const alicePreferences = LanguagePreferences(
        primaryLanguage: 'en',
        contentLanguages: ['en'],
      );
      const bobPreferences = LanguagePreferences(
        primaryLanguage: 'cy',
        contentLanguages: ['cy'],
      );
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final storage = _RegistryStorage(
        SessionRegistry.empty()
            .upsertAndActivate(
              token: 'token-a',
              did: alice.did.value,
              handle: 'alice.test',
            )
            .upsertAndActivate(
              token: 'token-b',
              did: bob.did.value,
              handle: 'bob.test',
            ),
      );
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, account) async => _PreferencesRepository(
              Future.value(
                account == alice ? alicePreferences : bobPreferences,
              ),
            ),
          ),
        ],
      );
      final subscription = container.listen(
        activeAccountInitializationProvider,
        (_, _) {},
      );
      addTearDown(subscription.close);

      Future<ActiveAccountInitialization> expectCurrent(
        AccountKey account,
        LanguagePreferences preferences,
      ) async {
        final initialized = await container.read(
          activeAccountInitializationProvider.future,
        );
        expect(initialized, isNotNull);
        expect(
          initialized!.lease,
          container.read(sessionRegistryProvider).requireValue.activeLease,
        );
        expect(initialized.lease.session.account, account);
        expect(initialized.languagePreferences, preferences);
        return initialized;
      }

      await expectCurrent(bob, bobPreferences);

      await container
          .read(sessionRegistryProvider.notifier)
          .activate(storage.value.leaseFor(alice)!);
      await expectCurrent(alice, alicePreferences);

      await container
          .read(sessionRegistryProvider.notifier)
          .activate(storage.value.leaseFor(bob)!);
      final reactivated = await expectCurrent(bob, bobPreferences);

      await container
          .read(sessionRegistryProvider.notifier)
          .removeConfirmed(reactivated.lease.session);
      await expectCurrent(alice, alicePreferences);
    },
  );

  test('same-account session replacement rejects the old completion', () async {
    const oldPreferences = LanguagePreferences(
      primaryLanguage: 'en',
      contentLanguages: ['en'],
    );
    const newPreferences = LanguagePreferences(
      primaryLanguage: 'fr',
      contentLanguages: ['fr'],
    );
    final oldLoad = Completer<LanguagePreferences>();
    final newLoad = Completer<LanguagePreferences>();
    final repository = _SequencedPreferencesRepository([oldLoad, newLoad]);
    final storage = _RegistryStorage(
      SessionRegistry.empty().upsertAndActivate(
        token: 'old-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
      ),
    );
    final container = ProviderContainer.test(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
        languagePreferencesRepositoryProvider.overrideWith(
          (ref, account) async => repository,
        ),
      ],
    );
    final subscription = container.listen(
      activeAccountInitializationProvider,
      (_, _) {},
    );
    addTearDown(subscription.close);

    await container.read(sessionRegistryProvider.future);
    while (repository._nextLoad == 0) {
      await Future<void>.delayed(Duration.zero);
    }
    final oldLease = storage.value.activeLease;
    await container
        .read(sessionRegistryProvider.notifier)
        .upsertAndActivate(
          token: 'new-token',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
    await Future<void>.delayed(Duration.zero);
    final newLease = storage.value.activeLease;
    expect(newLease, isNot(oldLease));

    oldLoad.complete(oldPreferences);
    await Future<void>.delayed(Duration.zero);
    expect(
      container.read(activeAccountInitializationProvider).value?.lease,
      isNot(newLease),
    );

    newLoad.complete(newPreferences);
    final initialized = await container.read(
      activeAccountInitializationProvider.future,
    );
    expect(initialized?.lease, newLease);
    expect(initialized?.languagePreferences, newPreferences);
  });

  test('synchronous projections fail outside the ready invariant', () {
    final preferencesContainer = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        activeAccountInitializationProvider.overrideWith(
          (ref) => null,
        ),
      ],
    );
    final policyContainer = ProviderContainer.test(
      retry: (_, _) => null,
      overrides: [
        activeAccountInitializationProvider.overrideWith(
          (ref) => null,
        ),
      ],
    );
    expect(
      () => preferencesContainer.read(activeLanguagePreferencesProvider),
      throwsA(anything),
    );
    expect(
      () => policyContainer.read(activeContentLanguagePolicyProvider),
      throwsA(anything),
    );
  });
}
