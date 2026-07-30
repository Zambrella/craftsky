import 'package:craftsky_app/languages/data/language_catalogue.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:flutter/widgets.dart';

LanguagePreferences deriveInitialLanguages(List<Locale> locales) {
  final content = <String>[];
  for (final locale in locales) {
    final tag = locale.languageCode.toLowerCase();
    if (supportedLanguageTags.contains(tag) && !content.contains(tag)) {
      content.add(tag);
    }
  }
  if (content.isEmpty) content.add('en');
  return LanguagePreferences(
    primaryLanguage: content.first,
    contentLanguages: List.unmodifiable(content),
  );
}
