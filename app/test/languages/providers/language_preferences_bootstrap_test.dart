import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/device_locale_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _BootstrapRepository implements LanguagePreferencesRepository {
  _BootstrapRepository({this.stored});

  final LanguagePreferences? stored;
  LanguagePreferences? initializedWith;

  @override
  Future<LanguagePreferences> load() async {
    final value = stored;
    if (value != null) return value;
    throw const ApiBadRequest('language_preferences_not_found');
  }

  @override
  Future<LanguagePreferences> initialize(
    LanguagePreferences proposal,
  ) async {
    initializedWith = proposal;
    return proposal;
  }

  @override
  Future<LanguagePreferences> replace(
    LanguagePreferences preferences,
  ) async => preferences;
}

void main() {
  final account = AccountKey('did:plc:alice');

  test(
    'IT-019 initialises a new account from ordered device locales',
    () async {
      final repository = _BootstrapRepository();
      final container = ProviderContainer.test(
        overrides: [
          deviceLocalesProvider.overrideWithValue(const [
            Locale('fr', 'CA'),
            Locale('en', 'GB'),
            Locale('fr', 'FR'),
            Locale('zz'),
          ]),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, requestedAccount) async {
              expect(requestedAccount, account);
              return repository;
            },
          ),
        ],
      );

      final preferences = await container.read(
        accountLanguagePreferencesProvider(account).future,
      );

      expect(
        preferences,
        const LanguagePreferences(
          primaryLanguage: 'fr',
          contentLanguages: ['fr', 'en'],
        ),
      );
      expect(repository.initializedWith, preferences);
    },
  );

  test(
    'IT-019 does not reapply device locales to a returning account',
    () async {
      const stored = LanguagePreferences(
        primaryLanguage: 'cy',
        contentLanguages: ['cy'],
      );
      final repository = _BootstrapRepository(stored: stored);
      final container = ProviderContainer.test(
        overrides: [
          deviceLocalesProvider.overrideWithValue(const [Locale('fr', 'CA')]),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, requestedAccount) async => repository,
          ),
        ],
      );

      expect(
        await container.read(
          accountLanguagePreferencesProvider(account).future,
        ),
        stored,
      );
      expect(repository.initializedWith, isNull);
    },
  );
}
