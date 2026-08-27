import 'package:craftsky_app/feed/media/youtube_consent.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  test(
    'YouTube consent is false by default and persists when allowed',
    () async {
      SharedPreferences.setMockInitialValues({});
      final preferences = await SharedPreferences.getInstance();
      final consent = SharedPreferencesYouTubeConsent(preferences);

      expect(consent.alwaysAllow, isFalse);

      await consent.setAlwaysAllow();

      expect(
        SharedPreferencesYouTubeConsent(preferences).alwaysAllow,
        isTrue,
      );
    },
  );
}
