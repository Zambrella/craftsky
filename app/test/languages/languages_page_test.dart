import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/pages/languages_page.dart';
import 'package:craftsky_app/languages/providers/content_language_invalidation.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
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

void main() {
  testWidgets('IT-001 presents and independently edits all three settings', (
    tester,
  ) async {
    final repository = _PageRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, account) async => repository,
          ),
          contentLanguageCacheInvalidatorProvider.overrideWithValue(() {}),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: LanguagesPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('App language'), findsOneWidget);
    expect(find.text('Primary language'), findsOneWidget);
    expect(find.text('Content languages'), findsOneWidget);
    expect(find.text('More app languages are coming.'), findsOneWidget);
    final appLanguage = tester.widget<DropdownButtonFormField<String>>(
      find.byType(DropdownButtonFormField<String>).first,
    );
    expect(appLanguage.initialValue, 'en');
    final appLanguageButton = tester.widget<DropdownButton<String>>(
      find
          .descendant(
            of: find.byType(DropdownButtonFormField<String>).first,
            matching: find.byType(DropdownButton<String>),
          )
          .first,
    );
    expect(appLanguageButton.items, hasLength(1));

    await tester.tap(find.widgetWithText(OutlinedButton, 'English'));
    await tester.pumpAndSettle();
    expect(find.text('Search languages'), findsOneWidget);
    await tester.enterText(find.byType(TextField), 'Spanish');
    await tester.pump();
    await tester.tap(find.text('Spanish').last);
    await tester.pumpAndSettle();
    expect(repository.value.primaryLanguage, 'es');
    expect(repository.value.contentLanguages, ['en']);

    await tester.ensureVisible(find.text('Add more languages…'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Add more languages…'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'French');
    await tester.pump();
    await tester.tap(find.widgetWithText(CheckboxListTile, 'French'));
    await tester.tap(find.text('Done'));
    await tester.pumpAndSettle();
    expect(repository.value.primaryLanguage, 'es');
    expect(repository.value.contentLanguages, ['en', 'fr']);

    tester
        .widget<InputChip>(find.widgetWithText(InputChip, 'English'))
        .onDeleted!();
    await tester.pumpAndSettle();
    tester
        .widget<InputChip>(find.widgetWithText(InputChip, 'French'))
        .onDeleted!();
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
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, account) async => repository,
          ),
          contentLanguageCacheInvalidatorProvider.overrideWithValue(() {}),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: LanguagesPage(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(OutlinedButton, 'English'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Spanish');
    await tester.pump();
    await tester.tap(find.text('Spanish').last);
    await tester.pumpAndSettle();

    expect(repository.value.primaryLanguage, 'en');
    expect(find.text('English'), findsWidgets);
    expect(messenger.calls, [
      ('error', 'That change could not be saved. Try again.', null),
    ]);
    expect(
      tester
          .widget<OutlinedButton>(
            find.widgetWithText(OutlinedButton, 'English'),
          )
          .onPressed,
      isNotNull,
    );
  });
}
