import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/pages/languages_page.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';

final class _PageRepository implements LanguagePreferencesRepository {
  _PageRepository({this.failReplacement = false});

  final bool failReplacement;
  LanguagePreferences value = const LanguagePreferences(
    primaryLanguage: 'en',
    contentLanguages: ['en'],
  );

  @override
  Future<LanguagePreferences> initialize(
    LanguagePreferences proposal,
  ) async => value;

  @override
  Future<LanguagePreferences> load() async => value;

  @override
  Future<LanguagePreferences> replace(
    LanguagePreferences preferences,
  ) async {
    if (failReplacement) throw StateError('offline');
    return value = preferences;
  }
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final SessionRegistry _signedInRegistry = SessionRegistry.empty()
    .upsertAndActivate(
      token: 'token',
      did: 'did:plc:test',
      handle: 'test.bsky.social',
    );

void main() {
  testWidgets('IT-001 presents and independently edits all three settings', (
    tester,
  ) async {
    final repository = _PageRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(_signedInRegistry),
          ),
          activeAccountInitializationProvider.overrideWith(
            (ref) => ActiveAccountInitialization(
              lease: _signedInRegistry.activeLease!,
              languagePreferences: repository.value,
            ),
          ),
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, account) async => repository,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const LanguagesPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('App language'), findsOneWidget);
    expect(find.text('Primary language'), findsOneWidget);
    expect(find.text('Content languages'), findsOneWidget);
    expect(find.text('More app languages are coming.'), findsOneWidget);
    expect(find.byType(CraftskyCard), findsNWidgets(3));
    for (final card in tester.widgetList<CraftskyCard>(
      find.byType(CraftskyCard),
    )) {
      expect(card.clipBehavior, Clip.none);
    }
    expect(
      find.byType(CraftskySingleSelectInput<String>),
      findsNWidgets(2),
    );
    expect(
      find.byType(CraftskySearchableMultiSelectInput<String>),
      findsOneWidget,
    );
    expect(find.byType(DropdownButtonFormField<String>), findsNothing);
    final appLanguage = tester.widget<CraftskySingleSelectInput<String>>(
      find.byKey(const Key('app-language-input')),
    );
    expect(appLanguage.value, 'en');
    expect(appLanguage.options, hasLength(1));

    await tester.tap(
      find.byKey(const Key('primary-language-search-input')),
    );
    await tester.pumpAndSettle();
    expect(
      find.byKey(const Key('primary-language-options-panel')),
      findsOneWidget,
    );
    await tester.enterText(
      find.byKey(const Key('primary-language-search-input')),
      'cy',
    );
    await tester.pumpAndSettle();
    expect(
      find.byKey(const Key('primary-language-option-cy')),
      findsOneWidget,
    );
    await tester.enterText(
      find.byKey(const Key('primary-language-search-input')),
      'Spanish',
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('primary-language-option-es')));
    await tester.pumpAndSettle();
    expect(repository.value.primaryLanguage, 'es');
    expect(repository.value.contentLanguages, ['en']);

    await tester.ensureVisible(
      find.byKey(const Key('content-languages-search-input')),
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const Key('content-languages-search-input')),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('content-languages-search-input')),
      'fr',
    );
    await tester.pumpAndSettle();
    expect(
      find.descendant(
        of: find.byKey(const Key('content-languages-option-fr')),
        matching: find.text('fr'),
      ),
      findsOneWidget,
    );
    await tester.tap(find.byKey(const Key('content-languages-option-fr')));
    await tester.pumpAndSettle();
    expect(repository.value.primaryLanguage, 'es');
    expect(repository.value.contentLanguages, ['en', 'fr']);

    await tester.tap(
      find.byKey(const Key('content-languages-remove-en')),
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const Key('content-languages-remove-fr')),
    );
    await tester.pumpAndSettle();
    expect(repository.value.contentLanguages, isEmpty);
    expect(find.textContaining('If none are selected'), findsOneWidget);
  });

  testWidgets('IT-018 retains values and reports a failed replacement', (
    tester,
  ) async {
    final repository = _PageRepository(failReplacement: true);
    final messenger = RecordingMessenger();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(_signedInRegistry),
          ),
          activeAccountInitializationProvider.overrideWith(
            (ref) => ActiveAccountInitialization(
              lease: _signedInRegistry.activeLease!,
              languagePreferences: repository.value,
            ),
          ),
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, account) async => repository,
          ),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const LanguagesPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(
      find.byKey(const Key('primary-language-search-input')),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('primary-language-search-input')),
      'Spanish',
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('primary-language-option-es')));
    await tester.pumpAndSettle();

    expect(repository.value.primaryLanguage, 'en');
    expect(find.text('English'), findsWidgets);
    expect(messenger.calls, [
      ('error', 'That change could not be saved. Try again.', null),
    ]);
    expect(
      tester
          .widget<CraftskySingleSelectInput<String>>(
            find.byKey(const Key('primary-language-input')),
          )
          .enabled,
      isTrue,
    );
  });
}
