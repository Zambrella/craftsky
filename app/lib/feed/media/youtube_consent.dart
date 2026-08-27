import 'package:craftsky_app/app_dependencies.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _alwaysAllowYouTubeKey = 'external_players.youtube.always_allow';

abstract interface class YouTubeConsentPreferences {
  bool get alwaysAllow;

  Future<void> setAlwaysAllow();
}

final class SharedPreferencesYouTubeConsent
    implements YouTubeConsentPreferences {
  const SharedPreferencesYouTubeConsent(this._preferences);

  final SharedPreferences _preferences;

  @override
  bool get alwaysAllow => _preferences.getBool(_alwaysAllowYouTubeKey) ?? false;

  @override
  Future<void> setAlwaysAllow() =>
      _preferences.setBool(_alwaysAllowYouTubeKey, true);
}

final youtubeConsentPreferencesProvider = Provider<YouTubeConsentPreferences>(
  (ref) => SharedPreferencesYouTubeConsent(
    ref.watch(sharedPreferencesProvider),
  ),
);
