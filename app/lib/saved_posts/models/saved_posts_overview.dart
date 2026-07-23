import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:flutter/foundation.dart';

@immutable
final class SavedPostsOverview {
  SavedPostsOverview._({
    required List<SavedPostFolder> folders,
    required List<SavedPostItem> unfiledItems,
  }) : folders = List.unmodifiable(folders),
       unfiledItems = List.unmodifiable(unfiledItems);

  factory SavedPostsOverview.project({
    required List<SavedPostFolder> folders,
    required List<SavedPostItem> items,
    required SavedPostSort sort,
  }) {
    final unfiled = items.where((item) => item.folderId == null).toList()
      ..sort(
        (a, b) => sort == SavedPostSort.newest
            ? b.savedAt.compareTo(a.savedAt)
            : a.savedAt.compareTo(b.savedAt),
      );
    return SavedPostsOverview._(
      folders: folders,
      unfiledItems: unfiled,
    );
  }

  final List<SavedPostFolder> folders;
  final List<SavedPostItem> unfiledItems;

  bool get showUnfiled => unfiledItems.isNotEmpty;
  bool get isEmpty => folders.isEmpty && unfiledItems.isEmpty;

  @override
  String toString() => 'SavedPostsOverview(<redacted>)';
}
