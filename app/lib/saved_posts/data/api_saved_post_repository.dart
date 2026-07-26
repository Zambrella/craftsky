import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_api_client.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';

/// Production repository backed by the fixed-account saved-post API client.
final class ApiSavedPostRepository implements SavedPostRepository {
  const ApiSavedPostRepository(this._api);

  final SavedPostApi _api;

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) =>
      _api.savePost(post, folderId: folderId);

  @override
  Future<void> unsave(Post post) => _api.unsavePost(post);

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) => _api.listSavedPosts(
    scope: scope,
    sort: sort,
    cursor: cursor,
    limit: limit,
  );

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) =>
      _api.listFolders(cursor: cursor, limit: limit);

  @override
  Future<SavedPostFolder> createFolder(String name) => _api.createFolder(name);

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) =>
      _api.renameFolder(folderId, name);

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => _api.deleteFolder(folderId, deleteSaves: deleteSaves);
}
