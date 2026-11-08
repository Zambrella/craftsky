import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/device_locale_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/diagnostic_outcome.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _BootstrapRepository implements LanguagePreferencesRepository {
  new({this.stored});

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
  test('logged-out language policy is expected cancellation', () async {
    final container = ProviderContainer.test(
      overrides: [
        activeAccountInitializationProvider.overrideWith((ref) async => null),
      ],
    );
    await container.read(activeAccountInitializationProvider.future);
    Object? failure;
    try {
      container.read(activeContentLanguagePolicyProvider);
    } on Object catch (error) {
      failure = error;
    }
    expect(failure, isNotNull);
    expect(
      isExpectedDiagnostic(
        failure!,
        const ReportContext(
          feature: 'language',
          operation: 'provider',
          classification: 'provider.failed',
        ),
      ),
      isTrue,
    );
  });

  final account = AccountKey('did:plc:alice');
  final lease = ActiveAccountLease(
    session: AccountSessionLease(account: account, sessionGeneration: 1),
    activationGeneration: 1,
  );

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
          languagePreferencesRepositoryProvider.overrideWith((
            ref,
            requestedAccount,
          ) async {
            expect(requestedAccount, account);
            return repository;
          }),
        ],
      );

      final state = await container.read(
        accountLanguagePreferencesProvider(lease).future,
      );
      final preferences = state.preferences;

      expect(
        preferences,
        const LanguagePreferences(
          primaryLanguage: 'en',
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
        (await container.read(accountLanguagePreferencesProvider(lease).future))
            .preferences,
        stored,
      );
      expect(repository.initializedWith, isNull);
    },
  );
}
