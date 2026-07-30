import 'package:flutter/foundation.dart';

@immutable
final class LanguagePreferences {
  const LanguagePreferences({
    required this.primaryLanguage,
    required this.contentLanguages,
  });

  factory LanguagePreferences.fromJson(Map<String, dynamic> json) {
    if (!json.containsKey('primaryLanguage') ||
        !json.containsKey('contentLanguages') ||
        json.length != 2) {
      throw const FormatException('Invalid language preferences response');
    }
    final primary = json['primaryLanguage'];
    final content = json['contentLanguages'];
    if (primary is! String || content is! List) {
      throw const FormatException('Invalid language preferences response');
    }
    final languages = <String>[];
    for (final value in content) {
      if (value is! String) {
        throw const FormatException('Invalid language preferences response');
      }
      languages.add(value);
    }
    return LanguagePreferences(
      primaryLanguage: primary,
      contentLanguages: List.unmodifiable(languages),
    );
  }

  final String primaryLanguage;
  final List<String> contentLanguages;

  LanguagePreferences copyWith({
    String? primaryLanguage,
    List<String>? contentLanguages,
  }) => LanguagePreferences(
    primaryLanguage: primaryLanguage ?? this.primaryLanguage,
    contentLanguages: List.unmodifiable(
      contentLanguages ?? this.contentLanguages,
    ),
  );

  Map<String, dynamic> toJson() => {
    'primaryLanguage': primaryLanguage,
    'contentLanguages': contentLanguages,
  };

  @override
  bool operator ==(Object other) =>
      other is LanguagePreferences &&
      other.primaryLanguage == primaryLanguage &&
      listEquals(other.contentLanguages, contentLanguages);

  @override
  int get hashCode => Object.hash(
    primaryLanguage,
    Object.hashAll(contentLanguages),
  );
}
