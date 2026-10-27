import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:flutter/foundation.dart';

@immutable
final class PostLanguageSelection {
  new _(List<String> values, {this.allowsEmpty = false})
    : values = List.unmodifiable(values) {
    if ((!allowsEmpty && values.isEmpty) ||
        values.length > 3 ||
        values.toSet().length != values.length) {
      throw DiagnosticStateError(
        'A post must have one to three distinct languages',
      );
    }
  }

  factory fromPrimary(String primaryLanguage) =>
      PostLanguageSelection._([primaryLanguage]);

  factory fromValues(List<String> values) => PostLanguageSelection._(values);

  /// Published records can legitimately omit their optional language tags.
  factory forPublishedEdit(List<String> values) =>
      PostLanguageSelection._(values, allowsEmpty: true);

  final List<String> values;
  final bool allowsEmpty;

  PostLanguageSelection add(String language) {
    if (values.contains(language) || values.length == 3) {
      throw DiagnosticStateError(
        'A post must have one to three distinct languages',
      );
    }
    return PostLanguageSelection._(
      [...values, language],
      allowsEmpty: allowsEmpty,
    );
  }

  PostLanguageSelection remove(String language) {
    if (!values.contains(language) || (!allowsEmpty && values.length == 1)) {
      throw DiagnosticStateError(
        'A post must have one to three distinct languages',
      );
    }
    return PostLanguageSelection._(
      [
        for (final value in values)
          if (value != language) value,
      ],
      allowsEmpty: allowsEmpty,
    );
  }
}
