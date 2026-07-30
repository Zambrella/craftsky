import 'package:craftsky_app/languages/models/language_preferences.dart';

abstract interface class LanguagePreferencesRepository {
  Future<LanguagePreferences> load();

  Future<LanguagePreferences> replace(LanguagePreferences preferences);

  Future<LanguagePreferences> initialize(LanguagePreferences proposal);
}
