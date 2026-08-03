import 'package:flutter/foundation.dart';

@immutable
final class PostLanguageSelection {
  PostLanguageSelection._(List<String> values)
    : values = List.unmodifiable(values) {
    if (values.isEmpty ||
        values.length > 3 ||
        values.toSet().length != values.length) {
      throw StateError('A post must have one to three distinct languages');
    }
  }

  factory PostLanguageSelection.fromPrimary(String primaryLanguage) =>
      PostLanguageSelection._([primaryLanguage]);

  factory PostLanguageSelection.fromValues(List<String> values) =>
      PostLanguageSelection._(values);

  final List<String> values;

  PostLanguageSelection add(String language) {
    if (values.contains(language) || values.length == 3) {
      throw StateError('A post must have one to three distinct languages');
    }
    return PostLanguageSelection._([...values, language]);
  }

  PostLanguageSelection remove(String language) {
    if (!values.contains(language) || values.length == 1) {
      throw StateError('A post must have one to three distinct languages');
    }
    return PostLanguageSelection._([
      for (final value in values)
        if (value != language) value,
    ]);
  }
}
