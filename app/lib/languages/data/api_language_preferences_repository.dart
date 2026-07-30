import 'package:craftsky_app/languages/data/language_preferences_api_client.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';

final class ApiLanguagePreferencesRepository
    implements LanguagePreferencesRepository {
  const ApiLanguagePreferencesRepository(this._api);

  final LanguagePreferencesApiClient _api;

  @override
  Future<LanguagePreferences> load() => _api.get();

  @override
  Future<LanguagePreferences> replace(LanguagePreferences preferences) =>
      _api.replace(preferences);

  @override
  Future<LanguagePreferences> initialize(LanguagePreferences proposal) =>
      _api.initialize(proposal);
}
