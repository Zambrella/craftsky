import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'saved_post_folder.mapper.dart';

/// Private flat folder returned by the saved-post folder endpoints.
@MappableClass(
  generateMethods:
      GenerateMethods.decode | GenerateMethods.encode | GenerateMethods.copy,
)
@immutable
final class SavedPostFolder with SavedPostFolderMappable {
  const SavedPostFolder({
    required this.id,
    required this.name,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String name;
  final DateTime createdAt;
  final DateTime updatedAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) || other is SavedPostFolder && id == other.id;

  @override
  int get hashCode => id.hashCode;
}

/// Opaque-cursor page returned by `GET /v1/saved-post-folders`.
@MappableClass(
  ignoreNull: true,
  generateMethods:
      GenerateMethods.decode |
      GenerateMethods.encode |
      GenerateMethods.copy |
      GenerateMethods.equals,
)
final class SavedPostFolderPage with SavedPostFolderPageMappable {
  const SavedPostFolderPage({required this.items, this.cursor});

  final List<SavedPostFolder> items;
  final String? cursor;
}

enum SavedPostFolderNameError { empty, tooLong, slash, control }

final class SavedPostFolderNameException implements Exception {
  const SavedPostFolderNameException(this.error);

  final SavedPostFolderNameError error;

  @override
  String toString() => 'SavedPostFolderNameException(${error.name})';
}

/// Mirrors AppView's trim, Unicode-scalar length, slash, and control checks.
String normalizeSavedPostFolderName(String name) {
  final normalized = name.trim();
  final runeCount = normalized.runes.length;
  if (runeCount == 0) {
    throw const SavedPostFolderNameException(SavedPostFolderNameError.empty);
  }
  if (runeCount > 100) {
    throw const SavedPostFolderNameException(SavedPostFolderNameError.tooLong);
  }
  if (normalized.contains('/') || normalized.contains(r'\')) {
    throw const SavedPostFolderNameException(SavedPostFolderNameError.slash);
  }
  if (normalized.runes.any(_isUnicodeControl)) {
    throw const SavedPostFolderNameException(SavedPostFolderNameError.control);
  }
  return normalized;
}

bool _isUnicodeControl(int rune) =>
    rune <= 0x1f || (rune >= 0x7f && rune <= 0x9f);
