import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/account_saved_post_state_provider.dart';
import 'package:craftsky_app/saved_posts/providers/save_post_dialog_controller.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test(
    'IT-006 AT-009 account switch rejects every late saved completion',
    () async {
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final repository = _DelayedSavedRepository();
      final listKey = SavedPostListKey(
        account: alice,
        scope: const SavedPostScope.unfiled(),
        sort: SavedPostSort.newest,
      );
      final savePost = _post('save', saved: false);
      final movePost = _post('move', saved: true, folderId: 'old');
      final unsavePost = _post('unsave', saved: true, folderId: 'old');
      final createKey = SavePostDialogKey(
        account: alice,
        uri: _post('create', saved: false).uri,
      );
      final saveKey = SavePostDialogKey(account: alice, uri: savePost.uri);
      final moveKey = SavePostDialogKey(
        account: alice,
        uri: movePost.uri,
        initialFolderId: 'old',
      );
      final container = ProviderContainer.test(
        overrides: [
          accountSavedPostRepositoryProvider(
            alice,
          ).overrideWith((ref) async => repository),
        ],
      );
      addTearDown(container.dispose);
      final subscriptions = <ProviderSubscription<Object?>>[
        container.listen(savedPostsProvider(listKey), (_, _) {}),
        container.listen(savedPostFoldersProvider(alice), (_, _) {}),
        container.listen(accountSavedPostStateProvider(alice), (_, _) {}),
        container.listen(
          savePostDialogControllerProvider(createKey),
          (_, _) {},
        ),
        container.listen(savePostDialogControllerProvider(saveKey), (_, _) {}),
        container.listen(savePostDialogControllerProvider(moveKey), (_, _) {}),
      ];
      addTearDown(() {
        for (final subscription in subscriptions) {
          subscription.close();
        }
      });
      await Future.wait([
        container.read(savedPostsProvider(listKey).future),
        container.read(savedPostFoldersProvider(alice).future),
      ]);

      final state =
          container.read(accountSavedPostStateProvider(alice).notifier)
            ..seedIfAbsent(savePost)
            ..seedIfAbsent(movePost)
            ..seedIfAbsent(unsavePost);
      container.read(savePostDialogControllerProvider(createKey).notifier)
        ..beginCreatingFolder()
        ..updateCreateName('Late folder');
      container
          .read(savePostDialogControllerProvider(saveKey).notifier)
          .selectFolder('late-save');
      container
          .read(savePostDialogControllerProvider(moveKey).notifier)
          .selectFolder('late-move');

      final lateList = container
          .read(savedPostsProvider(listKey).notifier)
          .loadMore();
      final lateCreate = container
          .read(savePostDialogControllerProvider(createKey).notifier)
          .createFolder();
      final lateSave = container
          .read(savePostDialogControllerProvider(saveKey).notifier)
          .confirmSave(savePost);
      final lateMove = container
          .read(savePostDialogControllerProvider(moveKey).notifier)
          .confirmMove(
            SavedPostItem(
              post: movePost,
              savedAt: DateTime.utc(2026, 7, 21, 10),
              folderId: 'old',
            ),
          );
      final lateUnsave = state.unsave(unsavePost);
      final lateRename = container
          .read(savedPostFoldersProvider(alice).notifier)
          .rename('old', 'Late rename');
      final lateDelete = container
          .read(savedPostFoldersProvider(alice).notifier)
          .delete('old', deleteSaves: true);
      await Future<void>.delayed(Duration.zero);

      await container.read(accountStateInvalidatorProvider)();
      for (var index = 0; index < 3; index++) {
        await Future<void>.delayed(Duration.zero);
      }
      await Future.wait([
        container.read(savedPostsProvider(listKey).future),
        container.read(savedPostFoldersProvider(alice).future),
      ]);
      container
          .read(accountSavedPostStateProvider(bob).notifier)
          .seedIfAbsent(_post('bob', saved: true, folderId: 'bob-folder'));

      repository.completeLateOperations();
      await Future.wait<Object?>([
        lateList,
        lateCreate,
        lateSave,
        lateMove,
        lateUnsave,
        lateRename,
        lateDelete,
      ]);
      await Future<void>.delayed(Duration.zero);

      expect(
        container
            .read(savedPostsProvider(listKey))
            .requireValue
            .items
            .map((item) => item.post.rkey),
        ['server-alice'],
      );
      expect(
        container
            .read(savedPostFoldersProvider(alice))
            .requireValue
            .items
            .map((folder) => folder.name),
        ['Server Alice'],
      );
      expect(
        container
            .read(accountSavedPostStateProvider(alice))
            .requireValue
            .forUri(savePost.uri)
            .initialized,
        isFalse,
      );
      expect(
        container
            .read(savePostDialogControllerProvider(createKey))
            .isCreatePending,
        isFalse,
      );
      expect(
        container.read(savePostDialogControllerProvider(saveKey)).isConfirmed,
        isFalse,
      );
      expect(
        container.read(savePostDialogControllerProvider(moveKey)).isConfirmed,
        isFalse,
      );
      expect(
        container
            .read(
              savedPostPresentationProvider(
                SavedPostKey(account: bob, uri: _post('bob', saved: true).uri),
              ),
            )
            .requireValue
            .folderId,
        'bob-folder',
      );
    },
  );
}

final class _DelayedSavedRepository implements SavedPostRepository {
  final listMore = Completer<SavedPostPage>();
  final create = Completer<SavedPostFolder>();
  final rename = Completer<SavedPostFolder>();
  final delete = Completer<void>();
  final unsaveCompletion = Completer<void>();
  final saveCompletions = <String, Completer<SavedPostState>>{
    'save': Completer<SavedPostState>(),
    'move': Completer<SavedPostState>(),
  };
  int listFirstPageCalls = 0;
  int folderFirstPageCalls = 0;

  void completeLateOperations() {
    listMore.complete(SavedPostPage(items: [_item('late-list')]));
    create.complete(_folder('late-create', 'Late folder'));
    rename.complete(_folder('old', 'Late rename'));
    delete.complete();
    unsaveCompletion.complete();
    saveCompletions['save']!.complete(
      SavedPostState(
        savedAt: DateTime.utc(2026, 7, 21, 13),
        folderId: 'late-save',
      ),
    );
    saveCompletions['move']!.complete(
      SavedPostState(
        savedAt: DateTime.utc(2026, 7, 21, 10),
        folderId: 'late-move',
      ),
    );
  }

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) {
    if (cursor != null) return listMore.future;
    listFirstPageCalls++;
    return Future.value(
      SavedPostPage(
        items: [_item(listFirstPageCalls == 1 ? 'initial' : 'server-alice')],
        cursor: listFirstPageCalls == 1 ? 'next' : null,
      ),
    );
  }

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) {
    folderFirstPageCalls++;
    return Future.value(
      SavedPostFolderPage(
        items: [
          if (folderFirstPageCalls == 1)
            _folder('old', 'Initial')
          else
            _folder('server-alice', 'Server Alice'),
        ],
      ),
    );
  }

  @override
  Future<SavedPostFolder> createFolder(String name) => create.future;

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) =>
      rename.future;

  @override
  Future<void> deleteFolder(String folderId, {required bool deleteSaves}) =>
      delete.future;

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) =>
      saveCompletions[post.rkey.toString()]!.future;

  @override
  Future<void> unsave(Post post) => unsaveCompletion.future;
}

SavedPostFolder _folder(String id, String name) => SavedPostFolder(
  id: id,
  name: name,
  createdAt: DateTime.utc(2026, 7, 21),
  updatedAt: DateTime.utc(2026, 7, 21),
);

SavedPostItem _item(String rkey) => SavedPostItem(
  post: _post(rkey, saved: true),
  savedAt: DateTime.utc(2026, 7, 21, 12),
);

Post _post(String rkey, {required bool saved, String? folderId}) =>
    PostMapper.fromMap({
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
      'viewerHasSaved': saved,
      'viewerSavedFolderId': folderId,
      'createdAt': '2026-07-21T10:00:00.000Z',
      'indexedAt': '2026-07-21T10:00:01.000Z',
      'author': {'did': 'did:plc:author', 'handle': 'author.craftsky.social'},
    });
