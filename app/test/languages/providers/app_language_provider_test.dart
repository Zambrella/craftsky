import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/languages/providers/app_language_provider.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  test(
    'App language is device-local and constrained to English in v1',
    () async {
      SharedPreferences.setMockInitialValues({'app_language': 'fr'});
      final preferences = await SharedPreferences.getInstance();
      final container = ProviderContainer.test(
        overrides: [
          sharedPreferencesProvider.overrideWithValue(preferences),
        ],
      );

      expect(container.read(appLanguageProvider), const Locale('en'));
      await container
          .read(appLanguageProvider.notifier)
          .select(const Locale('en'));
      expect(preferences.getString('app_language'), 'en');
    },
  );
}
