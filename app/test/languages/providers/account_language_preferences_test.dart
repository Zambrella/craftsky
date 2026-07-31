import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _AccountRepository implements LanguagePreferencesRepository {
  _AccountRepository(this.loaded);

  final Future<LanguagePreferences> loaded;
  Completer<LanguagePreferences>? replacement;

  @override
  Future<LanguagePreferences> initialize(
    LanguagePreferences proposal,
  ) async => proposal;

  @override
  Future<LanguagePreferences> load() => loaded;

  @override
  Future<LanguagePreferences> replace(
    LanguagePreferences preferences,
  ) {
    final completer = replacement;
    return completer?.future ?? Future.value(preferences);
  }
}

void main() {
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
  final aliceLease = ActiveAccountLease(
    session: AccountSessionLease(account: alice, sessionGeneration: 1),
    activationGeneration: 1,
  );
  final bobLease = ActiveAccountLease(
    session: AccountSessionLease(account: bob, sessionGeneration: 2),
    activationGeneration: 2,
  );

  test('UT-012 late account result cannot populate another account', () async {
    final aliceLoad = Completer<LanguagePreferences>();
    final repositories = {
      alice: _AccountRepository(aliceLoad.future),
      bob: _AccountRepository(Future.value(bobPreferences)),
    };
    final container = ProviderContainer.test(
      overrides: [
        languagePreferencesRepositoryProvider.overrideWith(
          (ref, account) async => repositories[account]!,
        ),
      ],
    );

    final aliceFuture = container.read(
      accountLanguagePreferencesProvider(aliceLease).future,
    );
    expect(
      (await container.read(
        accountLanguagePreferencesProvider(bobLease).future,
      )).preferences,
      bobPreferences,
    );

    aliceLoad.complete(alicePreferences);
    expect((await aliceFuture).preferences, alicePreferences);
    expect(
      container
          .read(accountLanguagePreferencesProvider(bobLease))
          .requireValue
          .preferences,
      bobPreferences,
    );
  });

  test('UT-012 invalidated generation discards late replacement', () async {
    final repository = _AccountRepository(Future.value(alicePreferences));
    final replacement = Completer<LanguagePreferences>();
    repository.replacement = replacement;
    final container = ProviderContainer.test(
      overrides: [
        languagePreferencesRepositoryProvider.overrideWith(
          (ref, account) async => repository,
        ),
      ],
    );
    final provider = accountLanguagePreferencesProvider(aliceLease);
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);
    await container.read(provider.future);

    final lateReplacement = container
        .read(provider.notifier)
        .replace(bobPreferences);
    container.invalidate(provider);
    await Future<void>.delayed(Duration.zero);
    replacement.complete(bobPreferences);

    await lateReplacement;
    expect(
      (await container.read(provider.future)).preferences,
      alicePreferences,
    );
  });

  test('failed replacement publishes an error with the prior value', () async {
    final repository = _AccountRepository(Future.value(alicePreferences));
    final replacement = Completer<LanguagePreferences>();
    repository.replacement = replacement;
    final container = ProviderContainer.test(
      overrides: [
        languagePreferencesRepositoryProvider.overrideWith(
          (ref, account) async => repository,
        ),
      ],
    );
    final provider = accountLanguagePreferencesProvider(aliceLease);
    final states = <AsyncValue<AccountLanguagePreferencesState>>[];
    final subscription = container.listen(provider, (_, next) {
      states.add(next);
    });
    addTearDown(subscription.close);
    await container.read(provider.future);

    final replacing = container.read(provider.notifier).replace(bobPreferences);
    replacement.completeError(StateError('offline'));
    await replacing;

    expect(
      states,
      contains(
        isA<AsyncValue<AccountLanguagePreferencesState>>().having(
          (state) => state.value?.replacement.isLoading,
          'replacement.isLoading',
          isTrue,
        ),
      ),
    );
    expect(states.last.requireValue.replacement.hasError, isTrue);
    expect(states.last.requireValue.preferences, alicePreferences);
  });
}
