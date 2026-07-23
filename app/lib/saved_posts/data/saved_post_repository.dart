import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';

/// Typed saved-post surface consumed by providers and UI orchestration.
abstract interface class SavedPostRepository {
  Future<SavedPostState> save(Post post, {required String? folderId});

  Future<void> unsave(Post post);

  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  });

  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit});

  Future<SavedPostFolder> createFolder(String name);

  Future<SavedPostFolder> renameFolder(String folderId, String name);

  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  });
}
