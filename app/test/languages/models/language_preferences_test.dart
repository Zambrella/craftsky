import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const original = LanguagePreferences(
    primaryLanguage: 'en',
    contentLanguages: ['en', 'cy'],
  );

  test('Primary and Content updates are independent', () {
    expect(
      original.copyWith(primaryLanguage: 'fr'),
      const LanguagePreferences(
        primaryLanguage: 'fr',
        contentLanguages: ['en', 'cy'],
      ),
    );
    expect(
      original.copyWith(contentLanguages: ['fr']),
      const LanguagePreferences(
        primaryLanguage: 'en',
        contentLanguages: ['fr'],
      ),
    );
  });

  test('strict JSON mapping uses only camelCase preference fields', () {
    expect(LanguagePreferences.fromJson(original.toJson()), original);
    expect(
      () => LanguagePreferences.fromJson({
        ...original.toJson(),
        'accountDid': 'did:plc:other',
      }),
      throwsFormatException,
    );
  });
}
