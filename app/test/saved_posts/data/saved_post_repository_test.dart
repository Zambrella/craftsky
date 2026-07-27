import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/data/api_saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_api_client.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('IT-002 repository forwards every typed saved operation', () async {
    final post = PostMapper.fromMap({
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
      'viewerHasSaved': false,
      'createdAt': '2026-07-21T10:00:00.000Z',
      'indexedAt': '2026-07-21T10:00:01.000Z',
      'author': {
        'did': 'did:plc:alice',
        'handle': 'alice.craftsky.social',
      },
    });
    final savedAt = DateTime.utc(2026, 7, 21, 11);
    final state = SavedPostState(savedAt: savedAt, folderId: 'folder-a');
    final page = SavedPostPage(
      items: [
        SavedPostItem(post: post, savedAt: savedAt, folderId: 'folder-a'),
      ],
      cursor: 'opaque:posts',
    );
    final folder = SavedPostFolder(
      id: 'folder-a',
      name: 'Ideas',
      createdAt: savedAt,
      updatedAt: savedAt,
    );
    final folderPage = SavedPostFolderPage(
      items: [folder],
      cursor: 'opaque:folders',
    );
    final api = _RecordingSavedPostApi(
      state: state,
      page: page,
      folder: folder,
      folderPage: folderPage,
    );
    final repository = ApiSavedPostRepository(api) as SavedPostRepository;

    expect(await repository.save(post, folderId: null), same(state));
    await repository.unsave(post);
    expect(
      await repository.list(
        scope: const SavedPostScope.folder('folder-a'),
        sort: SavedPostSort.oldest,
        cursor: 'opaque:start',
        limit: 12,
      ),
      same(page),
    );
    expect(
      await repository.listFolders(cursor: 'opaque:folders-start', limit: 8),
      same(folderPage),
    );
    expect(await repository.createFolder('Ideas'), same(folder));
    expect(await repository.renameFolder('folder-a', 'Later'), same(folder));
    await repository.deleteFolder('folder-a', deleteSaves: true);

    expect(api.savedPost, same(post));
    expect(api.savedFolderId, isNull);
    expect(api.unsavedPost, same(post));
    expect(api.listScope, const SavedPostScope.folder('folder-a'));
    expect(api.listSort, SavedPostSort.oldest);
    expect(api.listCursor, 'opaque:start');
    expect(api.listLimit, 12);
    expect(api.folderCursor, 'opaque:folders-start');
    expect(api.folderLimit, 8);
    expect(api.createdName, 'Ideas');
    expect(api.renamedId, 'folder-a');
    expect(api.renamedName, 'Later');
    expect(api.deletedId, 'folder-a');
    expect(api.deletedSaves, isTrue);

    api.error = Exception('bounded fake failure');
    await expectLater(
      () => repository.unsave(post),
      throwsA(same(api.error)),
    );
  });
}

final class _RecordingSavedPostApi implements SavedPostApi {
  _RecordingSavedPostApi({
    required this.state,
    required this.page,
    required this.folder,
    required this.folderPage,
  });

  final SavedPostState state;
  final SavedPostPage page;
  final SavedPostFolder folder;
  final SavedPostFolderPage folderPage;
  Exception? error;
  Post? savedPost;
  String? savedFolderId;
  Post? unsavedPost;
  SavedPostScope? listScope;
  SavedPostSort? listSort;
  String? listCursor;
  int? listLimit;
  String? folderCursor;
  int? folderLimit;
  String? createdName;
  String? renamedId;
  String? renamedName;
  String? deletedId;
  bool? deletedSaves;

  void _throwIfNeeded() {
    if (error case final error?) throw error;
  }

  @override
  Future<SavedPostState> savePost(
    Post post, {
    required String? folderId,
  }) async {
    _throwIfNeeded();
    savedPost = post;
    savedFolderId = folderId;
    return state;
  }

  @override
  Future<void> unsavePost(Post post) async {
    _throwIfNeeded();
    unsavedPost = post;
  }

  @override
  Future<SavedPostPage> listSavedPosts({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async {
    _throwIfNeeded();
    listScope = scope;
    listSort = sort;
    listCursor = cursor;
    listLimit = limit;
    return page;
  }

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async {
    _throwIfNeeded();
    folderCursor = cursor;
    folderLimit = limit;
    return folderPage;
  }

  @override
  Future<SavedPostFolder> createFolder(String name) async {
    _throwIfNeeded();
    createdName = name;
    return folder;
  }

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) async {
    _throwIfNeeded();
    renamedId = folderId;
    renamedName = name;
    return folder;
  }

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) async {
    _throwIfNeeded();
    deletedId = folderId;
    deletedSaves = deleteSaves;
  }
}
