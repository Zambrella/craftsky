import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_api_client.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  test('IT-001 sends and decodes every saved-post API contract', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
    final adapter = DioAdapter(dio: dio);
    final postJson = <String, dynamic>{
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lsaved',
      'cid': 'bafysaved',
      'rkey': '3lsaved',
      'text': 'A post worth returning to.',
      'tags': <String>[],
      'likeCount': 0,
      'repostCount': 0,
      'quoteCount': 0,
      'replyCount': 0,
      'viewerHasLiked': false,
      'viewerHasReposted': false,
      'viewerHasReplied': false,
      'viewerHasSaved': true,
      'viewerSavedFolderId': null,
      'createdAt': '2026-07-21T10:00:00.000Z',
      'indexedAt': '2026-07-21T10:00:01.000Z',
      'author': {
        'did': 'did:plc:alice',
        'handle': 'alice.craftsky.social',
      },
    };
    final stateJson = <String, dynamic>{
      'savedAt': '2026-07-21T11:00:00.000Z',
      'folderId': null,
    };
    final folderJson = <String, dynamic>{
      'id': 'folder-a',
      'name': 'Ideas',
      'createdAt': '2026-07-21T09:00:00.000Z',
      'updatedAt': '2026-07-21T09:30:00.000Z',
    };

    adapter
      ..onPost(
        '/v1/posts/did:plc:alice/3lsaved/saves',
        (server) => server.reply(201, stateJson),
        data: {'folderId': null},
      )
      ..onPost(
        '/v1/posts/did:plc:alice/3lsaved/saves',
        (server) => server.reply(200, {
          ...stateJson,
          'folderId': 'folder-a',
        }),
        data: {'folderId': 'folder-a'},
      )
      ..onDelete(
        '/v1/posts/did:plc:alice/3lsaved/saves',
        (server) => server.reply(204, null),
      )
      ..onGet(
        '/v1/saved-posts',
        (server) => server.reply(200, {
          'items': [
            {'post': postJson, ...stateJson},
          ],
          'cursor': 'opaque:next-posts',
        }),
        queryParameters: {
          'unfiled': 'true',
          'sort': 'newest',
          'limit': '25',
          'cursor': 'opaque:posts',
        },
      )
      ..onGet(
        '/v1/saved-posts',
        (server) => server.reply(200, {
          'items': [
            {
              'post': postJson,
              ...stateJson,
              'folderId': 'folder-a',
            },
          ],
        }),
        queryParameters: {'folderId': 'folder-a', 'sort': 'oldest'},
      )
      ..onGet(
        '/v1/saved-post-folders',
        (server) => server.reply(200, {
          'items': [folderJson],
          'cursor': 'opaque:next-folders',
        }),
        queryParameters: {'limit': '50', 'cursor': 'opaque:folders'},
      )
      ..onPost(
        '/v1/saved-post-folders',
        (server) => server.reply(201, folderJson),
        data: {'name': 'Ideas'},
      )
      ..onPatch(
        '/v1/saved-post-folders/folder-a',
        (server) => server.reply(200, {...folderJson, 'name': 'Later'}),
        data: {'name': 'Later'},
      )
      ..onDelete(
        '/v1/saved-post-folders/folder-keep',
        (server) => server.reply(204, null),
      )
      ..onDelete(
        '/v1/saved-post-folders/folder-remove',
        (server) => server.reply(204, null),
        queryParameters: {'deleteSaves': 'true'},
      );

    final api = SavedPostApiClient(dio);
    final post = PostMapper.fromMap(postJson);
    final unfiledState = await api.savePost(post, folderId: null);
    final folderedState = await api.savePost(post, folderId: 'folder-a');
    await api.unsavePost(post);
    final unfiledPage = await api.listSavedPosts(
      scope: const SavedPostScope.unfiled(),
      sort: SavedPostSort.newest,
      limit: 25,
      cursor: 'opaque:posts',
    );
    final folderPage = await api.listSavedPosts(
      scope: const SavedPostScope.folder('folder-a'),
      sort: SavedPostSort.oldest,
    );
    final folders = await api.listFolders(
      limit: 50,
      cursor: 'opaque:folders',
    );
    final created = await api.createFolder('Ideas');
    final renamed = await api.renameFolder('folder-a', 'Later');
    await api.deleteFolder('folder-keep', deleteSaves: false);
    await api.deleteFolder('folder-remove', deleteSaves: true);

    expect(unfiledState.folderId, isNull);
    expect(folderedState.folderId, 'folder-a');
    expect(unfiledPage.cursor, 'opaque:next-posts');
    expect(unfiledPage.items.single, isA<SavedPostItem>());
    expect(folderPage.cursor, isNull);
    expect(folderPage.items.single.folderId, 'folder-a');
    expect(folders.cursor, 'opaque:next-folders');
    expect(created.id, 'folder-a');
    expect(renamed.name, 'Later');
  });
}
