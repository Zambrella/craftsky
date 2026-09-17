import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';

const englishLanguagePreferences = LanguagePreferences(
  primaryLanguage: 'en',
  contentLanguages: ['en'],
);

final dynamic englishLanguagePreferencesOverride =
    activeLanguagePreferencesProvider.overrideWithValue(
      englishLanguagePreferences,
    );
