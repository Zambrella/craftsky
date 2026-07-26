import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('IT-003 isolates resources and restarts folder pages safely', () async {
    final account = AccountKey('did:plc:alice');
    final repository = _CollectionRepository();
    final unfiledNewestKey = SavedPostListKey(
      account: account,
      scope: const SavedPostScope.unfiled(),
      sort: SavedPostSort.newest,
    );
    final unfiledOldestKey = SavedPostListKey(
      account: account,
      scope: const SavedPostScope.unfiled(),
      sort: SavedPostSort.oldest,
    );
    final folderKey = SavedPostListKey(
      account: account,
      scope: const SavedPostScope.folder('folder-a'),
      sort: SavedPostSort.newest,
    );
    repository
      ..enqueueList(
        unfiledNewestKey,
        null,
        SavedPostPage(items: [_item('u1')], cursor: 'u-next'),
      )
      ..enqueueList(unfiledOldestKey, null, SavedPostPage(items: [_item('o1')]))
      ..enqueueList(folderKey, null, SavedPostPage(items: [_item('f1')]))
      ..enqueueFolders(
        null,
        SavedPostFolderPage(
          items: [_folder('b', 'B'), _folder('d', 'D')],
          cursor: 'folder-next',
        ),
      );

    final container = ProviderContainer.test(
      overrides: [
        accountSavedPostRepositoryProvider(
          account,
        ).overrideWith((ref) async => repository),
      ],
    );
    addTearDown(container.dispose);
    final subscriptions = [
      container.listen(savedPostsProvider(unfiledNewestKey), (_, _) {}),
      container.listen(savedPostsProvider(unfiledOldestKey), (_, _) {}),
      container.listen(savedPostsProvider(folderKey), (_, _) {}),
      container.listen(savedPostFoldersProvider(account), (_, _) {}),
    ];
    addTearDown(() {
      for (final subscription in subscriptions) {
        subscription.close();
      }
    });
    await Future.wait([
      container.read(savedPostsProvider(unfiledNewestKey).future),
      container.read(savedPostsProvider(unfiledOldestKey).future),
      container.read(savedPostsProvider(folderKey).future),
      container.read(savedPostFoldersProvider(account).future),
    ]);

    expect(
      container
          .read(savedPostsProvider(unfiledNewestKey))
          .requireValue
          .items
          .single
          .post
          .rkey,
      'u1',
    );
    expect(
      container
          .read(savedPostsProvider(unfiledOldestKey))
          .requireValue
          .items
          .single
          .post
          .rkey,
      'o1',
    );
    expect(
      container
          .read(savedPostsProvider(folderKey))
          .requireValue
          .items
          .single
          .post
          .rkey,
      'f1',
    );

    final nextPage = Completer<SavedPostPage>();
    repository.enqueueList(unfiledNewestKey, 'u-next', nextPage.future);
    final firstLoadMore = container
        .read(savedPostsProvider(unfiledNewestKey).notifier)
        .loadMore();
    final duplicateLoadMore = container
        .read(savedPostsProvider(unfiledNewestKey).notifier)
        .loadMore();
    await Future<void>.delayed(Duration.zero);
    expect(
      repository.listCalls.where((call) => call.cursor == 'u-next'),
      hasLength(1),
    );
    nextPage.complete(SavedPostPage(items: [_item('u1'), _item('u2')]));
    await firstLoadMore;
    await duplicateLoadMore;
    expect(
      container
          .read(savedPostsProvider(unfiledNewestKey))
          .requireValue
          .items
          .map((item) => item.post.rkey),
      ['u1', 'u2'],
    );
    expect(
      container
          .read(savedPostsProvider(unfiledOldestKey))
          .requireValue
          .items
          .single
          .post
          .rkey,
      'o1',
    );

    final invalidCursor = Completer<SavedPostFolderPage>();
    repository
      ..enqueueFolders('folder-next', invalidCursor.future)
      ..enqueueFolders(
        null,
        SavedPostFolderPage(items: [_folder('a', 'A'), _folder('b', 'B')]),
      );
    final folderLoadMore = container
        .read(savedPostFoldersProvider(account).notifier)
        .loadMore();
    await Future<void>.delayed(Duration.zero);
    invalidCursor.completeError(const ApiBadRequest('invalid_cursor'));
    await folderLoadMore;
    expect(
      container
          .read(savedPostFoldersProvider(account))
          .requireValue
          .items
          .map((folder) => folder.id),
      ['a', 'b'],
    );

    repository
      ..createdFolder = _folder('z-created', 'Created')
      ..enqueueFolders(
        null,
        SavedPostFolderPage(items: [_folder('a', 'A'), _folder('b', 'B')]),
      );
    final created = await container
        .read(savedPostFoldersProvider(account).notifier)
        .create('Created');
    expect(created?.id, 'z-created');
    expect(
      container
          .read(savedPostFoldersProvider(account))
          .requireValue
          .folderById('z-created'),
      same(repository.createdFolder),
    );

    repository
      ..renamedFolder = _folder('b', 'Renamed')
      ..enqueueFolders(
        null,
        SavedPostFolderPage(
          items: [_folder('a', 'A'), repository.renamedFolder!],
        ),
      );
    await container
        .read(savedPostFoldersProvider(account).notifier)
        .rename('b', 'Renamed');
    expect(
      container
          .read(savedPostFoldersProvider(account))
          .requireValue
          .folderById('b')
          ?.name,
      'Renamed',
    );

    repository.enqueueFolders(
      null,
      SavedPostFolderPage(items: [_folder('a', 'A')]),
    );
    final deleted = await container
        .read(savedPostFoldersProvider(account).notifier)
        .delete('b', deleteSaves: false);
    expect(deleted, isTrue);
    expect(
      container
          .read(savedPostFoldersProvider(account))
          .requireValue
          .folderById('b'),
      isNull,
    );
    expect(repository.deleted, [('b', false)]);
  });

  test('IT-008 keep-mode folder deletion refreshes loaded Unfiled', () async {
    final account = AccountKey('did:plc:alice');
    final repository = _CollectionRepository();
    final unfiledKey = SavedPostListKey(
      account: account,
      scope: const SavedPostScope.unfiled(),
      sort: SavedPostSort.newest,
    );
    repository
      ..enqueueList(
        unfiledKey,
        null,
        const SavedPostPage(items: []),
      )
      ..enqueueFolders(
        null,
        SavedPostFolderPage(items: [_folder('folder-a', 'Ideas')]),
      );

    final container = ProviderContainer.test(
      overrides: [
        accountSavedPostRepositoryProvider(
          account,
        ).overrideWith((ref) async => repository),
      ],
    );
    addTearDown(container.dispose);
    final unfiledSubscription = container.listen(
      savedPostsProvider(unfiledKey),
      (_, _) {},
    );
    final folderSubscription = container.listen(
      savedPostFoldersProvider(account),
      (_, _) {},
    );
    addTearDown(unfiledSubscription.close);
    addTearDown(folderSubscription.close);
    await Future.wait([
      container.read(savedPostsProvider(unfiledKey).future),
      container.read(savedPostFoldersProvider(account).future),
    ]);

    repository
      ..enqueueList(
        unfiledKey,
        null,
        SavedPostPage(items: [_item('moved-by-folder-delete')]),
      )
      ..enqueueFolders(
        null,
        const SavedPostFolderPage(items: []),
      );

    final deleted = await container
        .read(savedPostFoldersProvider(account).notifier)
        .delete('folder-a', deleteSaves: false);
    await container.read(savedPostsProvider(unfiledKey).future);

    expect(deleted, isTrue);
    expect(
      container
          .read(savedPostsProvider(unfiledKey))
          .requireValue
          .items
          .map((item) => item.post.rkey),
      ['moved-by-folder-delete'],
    );
    expect(
      repository.listCalls.where(
        (call) => call.key.scope.kind == SavedPostScopeKind.unfiled,
      ),
      hasLength(2),
    );
  });

  test(
    'IT-003 retries a failed folder mutation restart from page one',
    () async {
      final account = AccountKey('did:plc:alice');
      final repository = _CollectionRepository()
        ..createdFolder = _folder('created', 'Created')
        ..enqueueFolders(
          null,
          SavedPostFolderPage(
            items: [_folder('folder-a', 'Ideas')],
            cursor: 'unsafe-cursor',
          ),
        );
      final container = ProviderContainer.test(
        overrides: [
          accountSavedPostRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
      );
      addTearDown(container.dispose);
      final subscription = container.listen(
        savedPostFoldersProvider(account),
        (_, _) {},
      );
      addTearDown(subscription.close);
      await container.read(savedPostFoldersProvider(account).future);

      final failedRestartRequest = Completer<SavedPostFolderPage>();
      repository.enqueueFolders(null, failedRestartRequest.future);
      final create = container
          .read(savedPostFoldersProvider(account).notifier)
          .create('Created');
      await Future<void>.delayed(Duration.zero);
      failedRestartRequest.completeError(StateError('restart failed'));
      final created = await create;
      final failedRestart = container
          .read(savedPostFoldersProvider(account))
          .requireValue;

      expect(created?.id, 'created');
      expect(
        failedRestart.folderById('created'),
        same(repository.createdFolder),
      );
      expect(failedRestart.cursor, isNull);
      expect(failedRestart.incrementalError, isA<StateError>());

      repository.enqueueFolders(
        null,
        SavedPostFolderPage(
          items: [_folder('created', 'Created'), _folder('folder-a', 'Ideas')],
        ),
      );
      await container.read(savedPostFoldersProvider(account).notifier).retry();

      final recovered = container
          .read(savedPostFoldersProvider(account))
          .requireValue;
      expect(recovered.incrementalError, isNull);
      expect(recovered.items.map((folder) => folder.id), [
        'created',
        'folder-a',
      ]);
      expect(repository.folderCursors, [null, null, null]);
    },
  );
}

final class _ListCall {
  const _ListCall(this.key, this.cursor);
  final SavedPostListKey key;
  final String? cursor;
}

final class _CollectionRepository implements SavedPostRepository {
  final Map<String, List<Future<SavedPostPage>>> _listResponses = {};
  final Map<String, List<Future<SavedPostFolderPage>>> _folderResponses = {};
  final List<_ListCall> listCalls = [];
  final List<String?> folderCursors = [];
  final List<(String, bool)> deleted = [];
  SavedPostFolder? createdFolder;
  SavedPostFolder? renamedFolder;

  void enqueueList(
    SavedPostListKey key,
    String? cursor,
    FutureOr<SavedPostPage> response,
  ) => _listResponses
      .putIfAbsent(_listKey(key, cursor), () => [])
      .add(
        response is Future<SavedPostPage> ? response : Future.value(response),
      );

  void enqueueFolders(
    String? cursor,
    FutureOr<Object> response,
  ) => _folderResponses
      .putIfAbsent(cursor ?? '<first>', () => [])
      .add(
        response is SavedPostFolderPage
            ? Future.value(response)
            : response is Future<SavedPostFolderPage>
            ? response
            : Future.error(response),
      );

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) {
    final key = SavedPostListKey(
      account: AccountKey('did:plc:alice'),
      scope: scope,
      sort: sort,
    );
    listCalls.add(_ListCall(key, cursor));
    return _listResponses[_listKey(key, cursor)]!.removeAt(0);
  }

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) {
    folderCursors.add(cursor);
    return _folderResponses[cursor ?? '<first>']!.removeAt(0);
  }

  String _listKey(SavedPostListKey key, String? cursor) =>
      '${key.scope.kind.name}:'
      '${key.scope.folderId}:${key.sort.name}:${cursor ?? '<first>'}';

  @override
  Future<SavedPostFolder> createFolder(String name) async => createdFolder!;

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) async =>
      renamedFolder!;

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) async {
    deleted.add((folderId, deleteSaves));
  }

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) =>
      throw UnimplementedError();

  @override
  Future<void> unsave(Post post) => throw UnimplementedError();
}

SavedPostFolder _folder(String id, String name) => SavedPostFolder(
  id: id,
  name: name,
  createdAt: DateTime.utc(2026, 7, 21),
  updatedAt: DateTime.utc(2026, 7, 21),
);

SavedPostItem _item(String rkey) => SavedPostItemMapper.fromMap({
  'post': {
    'uri': 'at://did:plc:author/social.craftsky.feed.post/$rkey',
    'cid': 'bafy$rkey',
    'rkey': rkey,
    'text': rkey,
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
      'did': 'did:plc:author',
      'handle': 'author.craftsky.social',
    },
  },
  'savedAt': '2026-07-21T12:00:00.000Z',
});
