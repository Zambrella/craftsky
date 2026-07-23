import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

/// Private saved-post and folder AppView endpoints for one fixed-account Dio.
abstract interface class SavedPostApi {
  Future<SavedPostState> savePost(Post post, {required String? folderId});

  Future<void> unsavePost(Post post);

  Future<SavedPostPage> listSavedPosts({
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

final class SavedPostApiClient implements SavedPostApi {
  const SavedPostApiClient(this._dio);

  final Dio _dio;

  @override
  Future<SavedPostState> savePost(
    Post post, {
    required String? folderId,
  }) => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      _postSavePath(post),
      data: {'folderId': folderId},
    );
    return SavedPostStateMapper.fromMap(response.data!);
  });

  @override
  Future<void> unsavePost(Post post) => unwrapApi(() async {
    await _dio.delete<void>(_postSavePath(post));
  });

  @override
  Future<SavedPostPage> listSavedPosts({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/saved-posts',
      queryParameters: {
        if (scope.kind == SavedPostScopeKind.unfiled) 'unfiled': 'true',
        if (scope.kind == SavedPostScopeKind.folder) 'folderId': scope.folderId,
        'sort': sort.name,
        'limit': ?limit?.toString(),
        'cursor': ?cursor,
      },
    );
    return SavedPostPageMapper.fromMap(response.data!);
  });

  @override
  Future<SavedPostFolderPage> listFolders({
    String? cursor,
    int? limit,
  }) => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/saved-post-folders',
      queryParameters: {
        'limit': ?limit?.toString(),
        'cursor': ?cursor,
      },
    );
    return SavedPostFolderPageMapper.fromMap(response.data!);
  });

  @override
  Future<SavedPostFolder> createFolder(String name) => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/v1/saved-post-folders',
      data: {'name': name},
    );
    return SavedPostFolderMapper.fromMap(response.data!);
  });

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) =>
      unwrapApi(() async {
        final response = await _dio.patch<Map<String, dynamic>>(
          '/v1/saved-post-folders/${Uri.encodeComponent(folderId)}',
          data: {'name': name},
        );
        return SavedPostFolderMapper.fromMap(response.data!);
      });

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => unwrapApi(() async {
    await _dio.delete<void>(
      '/v1/saved-post-folders/${Uri.encodeComponent(folderId)}',
      queryParameters: deleteSaves ? {'deleteSaves': 'true'} : null,
    );
  });

  String _postSavePath(Post post) =>
      '/v1/posts/${post.author.did}/${post.rkey}/saves';
}
