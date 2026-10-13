import 'package:craftsky_app/feed/models/post.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'saved_post.mapper.dart';

enum SavedPostSort { newest, oldest }

enum SavedPostScopeKind { all, unfiled, folder }

/// List scope with a redacted string representation for private folder IDs.
@immutable
final class SavedPostScope {
  const new all() : kind = SavedPostScopeKind.all, folderId = null;
  const new unfiled() : kind = SavedPostScopeKind.unfiled, folderId = null;

  const new folder(this.folderId) : kind = SavedPostScopeKind.folder;

  final SavedPostScopeKind kind;
  final String? folderId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SavedPostScope &&
          kind == other.kind &&
          folderId == other.folderId;

  @override
  int get hashCode => Object.hash(kind, folderId);

  @override
  String toString() => 'SavedPostScope(${kind.name})';
}

/// Server-confirmed private saved state for one post.
@MappableClass(
  ignoreNull: true,
  generateMethods:
      GenerateMethods.decode |
      GenerateMethods.encode |
      GenerateMethods.copy |
      GenerateMethods.equals,
)
final class SavedPostState with SavedPostStateMappable {
  const new({required this.savedAt, this.folderId});

  final DateTime savedAt;
  final String? folderId;
}

/// One hydrated item returned by a saved-post list endpoint.
@MappableClass(
  ignoreNull: true,
  generateMethods:
      GenerateMethods.decode |
      GenerateMethods.encode |
      GenerateMethods.copy |
      GenerateMethods.equals,
)
final class SavedPostItem with SavedPostItemMappable {
  const new({
    required this.post,
    required this.savedAt,
    this.folderId,
  });

  final Post post;
  final DateTime savedAt;
  final String? folderId;
}

/// Opaque-cursor page returned by `GET /v1/saved-posts`.
@MappableClass(
  ignoreNull: true,
  generateMethods:
      GenerateMethods.decode |
      GenerateMethods.encode |
      GenerateMethods.copy |
      GenerateMethods.equals,
)
final class SavedPostPage with SavedPostPageMappable {
  const new({required this.items, this.cursor});

  final List<SavedPostItem> items;
  final String? cursor;
}
