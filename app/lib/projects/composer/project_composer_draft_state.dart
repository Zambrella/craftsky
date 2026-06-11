abstract final class ProjectComposerDraftState {
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
          final cleaned = items
              .whereType<String>()
              .map((item) => item.trim())
              .where((item) => item.isNotEmpty)
              .toList(growable: false);
          if (cleaned.isNotEmpty) normalised[entry.key] = cleaned;
        case final value? when value is! String:
          normalised[entry.key] = value;
      }
    }
    return normalised;
  }
}
