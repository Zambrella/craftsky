import 'package:craftsky_app/languages/data/device_locale_languages.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('prefers English while preserving supported device language order', () {
    final result = deriveInitialLanguages(const [
      Locale('fr', 'CA'),
      Locale('en', 'GB'),
      Locale('fr', 'FR'),
      Locale('en', 'US'),
      Locale('nl', 'NL'),
      Locale('zz', 'ZZ'),
    ]);

    expect(result.primaryLanguage, 'en');
    expect(result.contentLanguages, ['fr', 'en', 'nl']);
  });

  test('adds English content without replacing a non-English primary', () {
    final result = deriveInitialLanguages(const [
      Locale('fr', 'CA'),
      Locale('nl', 'NL'),
      Locale('fr', 'FR'),
      Locale('zz'),
    ]);

    expect(result.primaryLanguage, 'fr');
    expect(result.contentLanguages, ['fr', 'nl', 'en']);
  });

  test('falls back to English when no supported locale remains', () {
    for (final locales in <List<Locale>>[
      const [],
      const [Locale('zz'), Locale('xx')],
    ]) {
      final result = deriveInitialLanguages(locales);
      expect(result.primaryLanguage, 'en');
      expect(result.contentLanguages, ['en']);
    }
  });
}
