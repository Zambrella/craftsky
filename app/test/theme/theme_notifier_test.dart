import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/theme/theme_notifier.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  group('ThemeModeNotifier', () {
    late SharedPreferences prefs;

    setUp(() async {
      SharedPreferences.setMockInitialValues({});
      prefs = await SharedPreferences.getInstance();
    });

    ProviderContainer makeContainer() {
      return ProviderContainer(
        overrides: [
          sharedPreferencesProvider.overrideWithValue(prefs),
        ],
      );
    }

    test('defaults to ThemeMode.system when no preference stored', () {
      final container = makeContainer();
      addTearDown(container.dispose);

      expect(container.read(themeModeProvider), ThemeMode.system);
    });

    test('reads persisted ThemeMode.dark', () async {
      await prefs.setString('theme_mode', 'dark');
      final container = makeContainer();
      addTearDown(container.dispose);

      expect(container.read(themeModeProvider), ThemeMode.dark);
    });

    test('setMode updates state and persists to SharedPreferences', () async {
      final container = makeContainer();
      addTearDown(container.dispose);

      await container.read(themeModeProvider.notifier).setMode(ThemeMode.light);

      expect(container.read(themeModeProvider), ThemeMode.light);
      expect(prefs.getString('theme_mode'), 'light');
    });

    test('unknown persisted value falls back to system', () async {
      await prefs.setString('theme_mode', 'garbage');
      final container = makeContainer();
      addTearDown(container.dispose);

      expect(container.read(themeModeProvider), ThemeMode.system);
    });
  });
}
