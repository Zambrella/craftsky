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
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('UT-004 reduces and projects account URI saved state', () async {
    final alice = AccountKey('did:plc:alice');
    final bob = AccountKey('did:plc:bob');
    final aliceRepository = _ControlledSavedPostRepository();
    final bobRepository = _ControlledSavedPostRepository();
    final container = ProviderContainer.test(
      overrides: [
        accountSavedPostRepositoryProvider(
          alice,
        ).overrideWith((ref) async => aliceRepository),
        accountSavedPostRepositoryProvider(
          bob,
        ).overrideWith((ref) async => bobRepository),
      ],
    );
    addTearDown(container.dispose);

    final alicePost = _post(saved: true, folderId: 'folder-a');
    final bobPost = _post(saved: false);
    final aliceKey = SavedPostKey(account: alice, uri: alicePost.uri);
    final bobKey = SavedPostKey(account: bob, uri: bobPost.uri);
    final aliceNotifier = container.read(
      accountSavedPostStateProvider(alice).notifier,
    );
    final bobNotifier = container.read(
      accountSavedPostStateProvider(bob).notifier,
    );

    aliceNotifier.seedIfAbsent(alicePost);
    bobNotifier.seedIfAbsent(bobPost);

    expect(
      container.read(savedPostPresentationProvider(aliceKey)).requireValue,
      isA<SavedPostPresentation>()
          .having((state) => state.isSaved, 'isSaved', isTrue)
          .having((state) => state.folderId, 'folderId', 'folder-a'),
    );
    expect(
      container.read(savedPostPresentationProvider(bobKey)).requireValue,
      isA<SavedPostPresentation>()
          .having((state) => state.isSaved, 'isSaved', isFalse)
          .having((state) => state.folderId, 'folderId', isNull),
    );
    expect(aliceKey.toString(), isNot(contains('did:plc:alice')));
    expect(aliceKey.toString(), isNot(contains(alicePost.uri.toString())));
    expect(
      container
          .read(accountSavedPostStateProvider(alice))
          .requireValue
          .toString(),
      isNot(contains(alicePost.uri.toString())),
    );

    final accountMap = container
        .read(accountSavedPostStateProvider(alice))
        .requireValue;
    final dataProjection = projectSavedPostPresentation(
      AsyncData(accountMap),
      alicePost.uri,
    );
    final loadingProjection = projectSavedPostPresentation(
      const AsyncLoading<AccountSavedPostStateMap>(),
      alicePost.uri,
    );
    final outerError = StateError('outer failure');
    final errorProjection = projectSavedPostPresentation(
      AsyncError<AccountSavedPostStateMap>(
        outerError,
        StackTrace.empty,
      ),
      alicePost.uri,
    );
    expect(dataProjection, isA<AsyncData<SavedPostPresentation>>());
    expect(dataProjection.requireValue.folderId, 'folder-a');
    expect(loadingProjection, isA<AsyncLoading<SavedPostPresentation>>());
    expect(errorProjection, isA<AsyncError<SavedPostPresentation>>());
    expect(errorProjection.error, same(outerError));

    aliceRepository.unsaveCompleter = Completer<void>();
    final firstUnsave = aliceNotifier.unsave(alicePost);
    final duplicateUnsave = aliceNotifier.unsave(alicePost);

    expect(
      container.read(savedPostPresentationProvider(aliceKey)).requireValue,
      isA<SavedPostPresentation>()
          .having((state) => state.isSaved, 'optimistic isSaved', isFalse)
          .having(
            (state) => state.pendingMutation,
            'pendingMutation',
            SavedPostMutation.unsave,
          ),
    );
    expect(
      container
          .read(savedPostPresentationProvider(bobKey))
          .requireValue
          .isSaved,
      isFalse,
    );
    await Future<void>.delayed(Duration.zero);
    expect(aliceRepository.unsaveCalls, 1);

    aliceRepository.unsaveCompleter!.completeError(StateError('delete failed'));
    await firstUnsave;
    await duplicateUnsave;

    final rolledBack = container
        .read(savedPostPresentationProvider(aliceKey))
        .requireValue;
    expect(rolledBack.isSaved, isTrue);
    expect(rolledBack.folderId, 'folder-a');
    expect(rolledBack.pendingMutation, isNull);
    expect(rolledBack.hasError, isTrue);

    final confirmedAt = DateTime.utc(2026, 7, 21, 12);
    aliceRepository.saveResult = SavedPostState(
      savedAt: confirmedAt,
      folderId: 'folder-b',
    );
    final save = aliceNotifier.save(alicePost, 'folder-b');
    final pendingSave = container
        .read(savedPostPresentationProvider(aliceKey))
        .requireValue;
    expect(pendingSave.isSaved, isTrue);
    expect(pendingSave.folderId, 'folder-a');
    expect(pendingSave.pendingMutation, SavedPostMutation.save);
    await save;

    final confirmed = container
        .read(savedPostPresentationProvider(aliceKey))
        .requireValue;
    expect(confirmed.isSaved, isTrue);
    expect(confirmed.folderId, 'folder-b');
    expect(confirmed.savedAt, confirmedAt);
    expect(confirmed.pendingMutation, isNull);
    expect(confirmed.hasError, isFalse);

    aliceNotifier.seedIfAbsent(alicePost);
    expect(
      container
          .read(savedPostPresentationProvider(aliceKey))
          .requireValue
          .folderId,
      'folder-b',
    );
    expect(
      container
          .read(savedPostPresentationProvider(bobKey))
          .requireValue
          .isSaved,
      isFalse,
    );
  });

  test(
    'UT-014 applies confirmed folder changes without changing chronology',
    () {
      final account = AccountKey('did:plc:alice');
      final container = ProviderContainer.test();
      addTearDown(container.dispose);

      final post = _post(saved: true, folderId: 'folder-a');
      final savedAt = DateTime.utc(2026, 7, 21, 12);
      final notifier = container.read(
        accountSavedPostStateProvider(account).notifier,
      );
      final key = SavedPostKey(account: account, uri: post.uri);

      notifier
        ..reconcileSavedItem(
          SavedPostItem(post: post, savedAt: savedAt, folderId: 'folder-a'),
        )
        ..reconcileSavedItem(
          SavedPostItem(post: post, savedAt: savedAt, folderId: 'folder-b'),
        );

      final moved = container
          .read(savedPostPresentationProvider(key))
          .requireValue;
      expect(moved.folderId, 'folder-b');
      expect(moved.savedAt, savedAt);

      notifier.reconcileFolderDeletion('folder-b', deleteSaves: false);

      final unfiled = container
          .read(savedPostPresentationProvider(key))
          .requireValue;
      expect(unfiled.isSaved, isTrue);
      expect(unfiled.folderId, isNull);
      expect(unfiled.savedAt, savedAt);

      notifier
        ..reconcileSavedItem(
          SavedPostItem(post: post, savedAt: savedAt, folderId: 'folder-b'),
        )
        ..reconcileFolderDeletion('folder-b', deleteSaves: true);

      final removed = container
          .read(savedPostPresentationProvider(key))
          .requireValue;
      expect(removed.isSaved, isFalse);
      expect(removed.folderId, isNull);
      expect(removed.savedAt, isNull);
    },
  );
}

Post _post({required bool saved, String? folderId}) => PostMapper.fromMap({
  'uri': 'at://did:plc:author/social.craftsky.feed.post/3lsaved',
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
  'viewerHasSaved': saved,
  'viewerSavedFolderId': folderId,
  'createdAt': '2026-07-21T10:00:00.000Z',
  'indexedAt': '2026-07-21T10:00:01.000Z',
  'author': {
    'did': 'did:plc:author',
    'handle': 'author.craftsky.social',
  },
});

final class _ControlledSavedPostRepository implements SavedPostRepository {
  SavedPostState saveResult = SavedPostState(
    savedAt: DateTime.utc(2026, 7, 21, 11),
  );
  Completer<void>? unsaveCompleter;
  int unsaveCalls = 0;

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) async =>
      saveResult;

  @override
  Future<void> unsave(Post post) {
    unsaveCalls++;
    return unsaveCompleter?.future ?? Future<void>.value();
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
