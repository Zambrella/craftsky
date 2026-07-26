import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_viewer_state.dart';
import 'package:craftsky_app/saved_posts/providers/account_saved_post_state_provider.dart';
import 'package:craftsky_app/saved_posts/providers/save_post_dialog_controller.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('IT-004 confirms save and move once through provider state', () async {
    final account = AccountKey('did:plc:alice');
    final repository = _MutationSavedPostRepository();
    final container = ProviderContainer.test(
      overrides: [
        accountSavedPostRepositoryProvider(
          account,
        ).overrideWith((ref) async => repository),
      ],
    );
    addTearDown(container.dispose);

    final unsavedPost = _post('new-save', saved: false);
    final saveKey = SavePostDialogKey(account: account, uri: unsavedPost.uri);
    final saveProvider = savePostDialogControllerProvider(saveKey);
    final saveSubscription = container.listen(
      saveProvider,
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(saveSubscription.close);
    container
        .read(accountSavedPostStateProvider(account).notifier)
        .seedIfAbsent(unsavedPost);
    container.read(saveProvider.notifier).selectFolder('folder-a');

    repository.saveCompleter = Completer<SavedPostState>();
    final firstSave = container
        .read(saveProvider.notifier)
        .confirmSave(unsavedPost);
    final duplicateSave = container
        .read(saveProvider.notifier)
        .confirmSave(unsavedPost);

    expect(container.read(saveProvider).isConfirming, isTrue);
    expect(container.read(saveProvider).selectedFolderId, 'folder-a');
    expect(container.read(saveProvider).isConfirmed, isFalse);
    await Future<void>.delayed(Duration.zero);
    expect(repository.saveCalls, 1);
    expect(repository.lastFolderId, 'folder-a');

    final savedAt = DateTime.utc(2026, 7, 21, 13);
    repository.saveCompleter!.complete(
      SavedPostState(savedAt: savedAt, folderId: 'folder-a'),
    );
    await firstSave;
    await duplicateSave;

    expect(container.read(saveProvider).isConfirming, isFalse);
    expect(container.read(saveProvider).isConfirmed, isTrue);
    expect(container.read(saveProvider).confirmError, isNull);
    final savedPresentation = container
        .read(
          savedPostPresentationProvider(
            SavedPostKey(
              account: account,
              uri: unsavedPost.uri,
            ),
          ),
        )
        .requireValue;
    expect(savedPresentation.isSaved, isTrue);
    expect(savedPresentation.folderId, 'folder-a');
    expect(savedPresentation.savedAt, savedAt);

    final savedPost = _post(
      'move-save',
      saved: true,
      folderId: 'folder-current',
    );
    final item = SavedPostItem(
      post: savedPost,
      savedAt: DateTime.utc(2026, 7, 21, 12),
      folderId: 'folder-current',
    );
    final moveKey = SavePostDialogKey(
      account: account,
      uri: savedPost.uri,
      initialFolderId: 'folder-current',
    );
    final moveProvider = savePostDialogControllerProvider(moveKey);
    final moveSubscription = container.listen(
      moveProvider,
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(moveSubscription.close);
    container
        .read(accountSavedPostStateProvider(account).notifier)
        .seedIfAbsent(savedPost);
    container.read(moveProvider.notifier).selectFolder('folder-attempted');

    repository.saveCompleter = Completer<SavedPostState>();
    final failedMove = container.read(moveProvider.notifier).confirmMove(item);
    final duplicateMove = container
        .read(moveProvider.notifier)
        .confirmMove(item);
    expect(container.read(moveProvider).isConfirming, isTrue);
    expect(container.read(moveProvider).selectedFolderId, 'folder-attempted');
    expect(
      container
          .read(
            savedPostPresentationProvider(
              SavedPostKey(
                account: account,
                uri: savedPost.uri,
              ),
            ),
          )
          .requireValue
          .folderId,
      'folder-current',
    );
    await Future<void>.delayed(Duration.zero);
    expect(repository.saveCalls, 2);

    repository.saveCompleter!.completeError(StateError('move failed'));
    await failedMove;
    await duplicateMove;

    final failedState = container.read(moveProvider);
    expect(failedState.isConfirming, isFalse);
    expect(failedState.isConfirmed, isFalse);
    expect(failedState.selectedFolderId, 'folder-attempted');
    expect(failedState.confirmError, SavePostDialogError.confirmFailed);
    final movedPresentation = container
        .read(
          savedPostPresentationProvider(
            SavedPostKey(
              account: account,
              uri: savedPost.uri,
            ),
          ),
        )
        .requireValue;
    expect(movedPresentation.isSaved, isTrue);
    expect(movedPresentation.folderId, 'folder-current');
  });

  test('IT-005 synchronizes optimistic unsave across consumers', () async {
    final account = AccountKey('did:plc:alice');
    final repository = _MutationSavedPostRepository();
    final container = ProviderContainer.test(
      overrides: [
        accountSavedPostRepositoryProvider(
          account,
        ).overrideWith((ref) async => repository),
      ],
    );
    addTearDown(container.dispose);

    final post = _post('shared-unsave', saved: true, folderId: 'folder-a');
    final key = SavedPostKey(account: account, uri: post.uri);
    final accountNotifier = container.read(
      accountSavedPostStateProvider(account).notifier,
    )..seedIfAbsent(post);
    final firstValues = <SavedPostPresentation>[];
    final secondValues = <SavedPostPresentation>[];
    final firstConsumer = container.listen(
      savedPostPresentationProvider(key),
      (_, next) => firstValues.add(next.requireValue),
      fireImmediately: true,
    );
    final secondConsumer = container.listen(
      savedPostPresentationProvider(key),
      (_, next) => secondValues.add(next.requireValue),
      fireImmediately: true,
    );
    addTearDown(firstConsumer.close);
    addTearDown(secondConsumer.close);

    repository.unsaveCompleter = Completer<void>();
    final firstUnsave = accountNotifier.unsave(post);
    final duplicateUnsave = accountNotifier.unsave(post);

    expect(
      container.read(savedPostPresentationProvider(key)).requireValue.isSaved,
      isFalse,
    );
    await Future<void>.delayed(Duration.zero);
    expect(firstValues.last.isSaved, isFalse);
    expect(secondValues.last.isSaved, isFalse);
    expect(firstValues.last.pendingMutation, SavedPostMutation.unsave);
    accountNotifier.seedIfAbsent(post);
    expect(firstValues.last.isSaved, isFalse);
    expect(repository.unsaveCalls, 1);

    repository.unsaveCompleter!.completeError(StateError('delete failed'));
    await firstUnsave;
    await duplicateUnsave;
    await Future<void>.delayed(Duration.zero);

    expect(firstValues.last.isSaved, isTrue);
    expect(secondValues.last.isSaved, isTrue);
    expect(firstValues.last.folderId, 'folder-a');
    expect(firstValues.last.pendingMutation, isNull);
    expect(firstValues.last.hasError, isTrue);

    repository.unsaveCompleter = Completer<void>();
    final successfulUnsave = accountNotifier.unsave(post);
    await Future<void>.delayed(Duration.zero);
    expect(firstValues.last.isSaved, isFalse);
    expect(secondValues.last.isSaved, isFalse);
    repository.unsaveCompleter!.complete();
    await successfulUnsave;
    await Future<void>.delayed(Duration.zero);

    expect(repository.unsaveCalls, 2);
    expect(firstValues.last.isSaved, isFalse);
    expect(secondValues.last.isSaved, isFalse);
    expect(firstValues.last.pendingMutation, isNull);
    expect(firstValues.last.hasError, isFalse);
  });

  test('UT-010 canceled move retains selection without inline error', () async {
    final account = AccountKey('did:plc:alice');
    final repository = _MutationSavedPostRepository();
    final container = ProviderContainer.test(
      overrides: [
        accountSavedPostRepositoryProvider(
          account,
        ).overrideWith((ref) async => repository),
      ],
    );
    addTearDown(container.dispose);

    final post = _post('canceled-move', saved: true, folderId: 'folder-a');
    final item = SavedPostItem(
      post: post,
      savedAt: DateTime.utc(2026, 7, 21, 12),
      folderId: 'folder-a',
    );
    final provider = savePostDialogControllerProvider(
      SavePostDialogKey(
        account: account,
        uri: post.uri,
        initialFolderId: 'folder-a',
      ),
    );
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);
    container
        .read(accountSavedPostStateProvider(account).notifier)
        .seedIfAbsent(post);
    container.read(provider.notifier).selectFolder('folder-b');

    repository.saveCompleter = Completer<SavedPostState>();
    final move = container.read(provider.notifier).confirmMove(item);
    await Future<void>.delayed(Duration.zero);
    repository.saveCompleter!.completeError(const ApiCanceled());
    await move;

    final state = container.read(provider);
    expect(state.isConfirming, isFalse);
    expect(state.isConfirmed, isFalse);
    expect(state.selectedFolderId, 'folder-b');
    expect(state.confirmError, isNull);
    expect(state.canConfirm, isTrue);
  });
}

Post _post(
  String rkey, {
  required bool saved,
  String? folderId,
}) => PostMapper.fromMap({
  'uri': 'at://did:plc:author/social.craftsky.feed.post/$rkey',
  'cid': 'bafy$rkey',
  'rkey': rkey,
  'text': 'A post worth returning to.',
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
  'author': {
    'did': 'did:plc:author',
    'handle': 'author.craftsky.social',
  },
});

final class _MutationSavedPostRepository implements SavedPostRepository {
  Completer<SavedPostState>? saveCompleter;
  Completer<void>? unsaveCompleter;
  int saveCalls = 0;
  int unsaveCalls = 0;
  String? lastFolderId;

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) {
    saveCalls++;
    lastFolderId = folderId;
    return saveCompleter!.future;
  }

  @override
  Future<void> unsave(Post post) {
    unsaveCalls++;
    return unsaveCompleter!.future;
  }

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) => throw UnimplementedError();

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) =>
      throw UnimplementedError();

  @override
  Future<SavedPostFolder> createFolder(String name) =>
      throw UnimplementedError();

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) =>
      throw UnimplementedError();

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => throw UnimplementedError();
}
