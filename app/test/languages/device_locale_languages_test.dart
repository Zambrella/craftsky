import 'package:craftsky_app/languages/data/device_locale_languages.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('derives ordered supported base languages from device locales', () {
    final result = deriveInitialLanguages(const [
      Locale('fr', 'CA'),
      Locale('en', 'GB'),
      Locale('fr', 'FR'),
      Locale('zz', 'ZZ'),
    ]);

    expect(result.primaryLanguage, 'fr');
    expect(result.contentLanguages, ['fr', 'en']);
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
