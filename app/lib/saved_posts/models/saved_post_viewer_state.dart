import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'saved_post_viewer_state.mapper.dart';

enum SavedPostMutation { save, move, unsave }

/// One account's presentation state for one canonical post URI.
@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class SavedPostPresentation with SavedPostPresentationMappable {
  const SavedPostPresentation({
    required this.initialized,
    required this.isSaved,
    required this.revision,
    this.folderId,
    this.savedAt,
    this.pendingMutation,
    this.lastError,
  });

  const SavedPostPresentation.uninitialized()
    : initialized = false,
      isSaved = false,
      revision = 0,
      folderId = null,
      savedAt = null,
      pendingMutation = null,
      lastError = null;

  factory SavedPostPresentation.fromPost(Post post) => SavedPostPresentation(
    initialized: true,
    isSaved: post.viewerHasSaved,
    revision: 0,
    folderId: post.viewerSavedFolderId,
  );

  final bool initialized;
  final bool isSaved;
  final int revision;
  final String? folderId;
  final DateTime? savedAt;
  final SavedPostMutation? pendingMutation;
  final Object? lastError;

  bool get hasError => lastError != null;
  bool get isPending => pendingMutation != null;

  @override
  String toString() => 'SavedPostPresentation(<redacted>)';
}

/// Immutable URI map with a deliberately redacted diagnostic representation.
@immutable
final class AccountSavedPostStateMap {
  AccountSavedPostStateMap._(Map<AtUri, SavedPostPresentation> entries)
    : _entries = Map.unmodifiable(entries);

  factory AccountSavedPostStateMap.empty() =>
      AccountSavedPostStateMap._(const {});

  final Map<AtUri, SavedPostPresentation> _entries;

  bool contains(AtUri uri) => _entries.containsKey(uri);

  SavedPostPresentation forUri(AtUri uri) =>
      _entries[uri] ?? const SavedPostPresentation.uninitialized();

  AccountSavedPostStateMap put(
    AtUri uri,
    SavedPostPresentation presentation,
  ) => AccountSavedPostStateMap._({..._entries, uri: presentation});

  AccountSavedPostStateMap afterFolderDeletion(
    String folderId, {
    required bool deleteSaves,
  }) {
    var changed = false;
    final next = <AtUri, SavedPostPresentation>{};
    for (final MapEntry(:key, :value) in _entries.entries) {
      if (value.folderId != folderId) {
        next[key] = value;
        continue;
      }
      changed = true;
      next[key] = SavedPostPresentation(
        initialized: true,
        isSaved: !deleteSaves,
        revision: value.revision + 1,
        savedAt: deleteSaves ? null : value.savedAt,
      );
    }
    return changed ? AccountSavedPostStateMap._(next) : this;
  }

  @override
  String toString() => 'AccountSavedPostStateMap(<redacted>)';
}
