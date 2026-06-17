import 'package:craftsky_app/projects/models/project.dart';

/// Detects whether the project composer has unsaved user-entered content.
abstract final class ProjectComposerDraftState {
  /// Returns true when the body text, images, or normalized metadata values
  /// differ from the initial composer state.
  static bool hasDraft({
    required String bodyText,
    required String initialBodyText,
    required int imageCount,
    required Map<String, dynamic> formValues,
    Map<String, dynamic> initialFormValues = const {},
  }) {
    if (bodyText != initialBodyText) return true;
    if (imageCount > 0) return true;
    return _normalised(formValues).toString() !=
        _normalised(initialFormValues).toString();
  }

  static Map<String, dynamic> _normalised(Map<String, dynamic> values) {
    final normalised = <String, dynamic>{};
    for (final entry in values.entries) {
      final value = entry.value;
      switch (value) {
        case final String text when text.trim().isNotEmpty:
          normalised[entry.key] = text.trim();
        case final Iterable<Object?> items:
          final cleaned = <Object>[];
          for (final item in items) {
            final normalisedItem = _normaliseListItem(item);
            if (normalisedItem != null) cleaned.add(normalisedItem);
          }
          if (cleaned.isNotEmpty) normalised[entry.key] = cleaned;
        case final value? when value is! String:
          normalised[entry.key] = value;
      }
    }
    return normalised;
  }

  static Object? _normaliseListItem(Object? value) {
    return switch (value) {
      final String text when text.trim().isNotEmpty => text.trim(),
      final ProjectMaterial material when material.text.trim().isNotEmpty =>
        material.copyWith(text: material.text.trim()).toMap(),
      _ => null,
    };
  }
}
