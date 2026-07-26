import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_viewer_state.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'account_saved_post_state_provider.g.dart';

@Riverpod(keepAlive: true)
class AccountSavedPostState extends _$AccountSavedPostState {
  @override
  FutureOr<AccountSavedPostStateMap> build(AccountKey account) {
    ref.watch(savedPostAccountBoundaryProvider);
    return AccountSavedPostStateMap.empty();
  }

  void seedIfAbsent(Post post) {
    final current = state.requireValue;
    if (current.contains(post.uri)) return;
    state = AsyncData(
      current.put(post.uri, SavedPostPresentation.fromPost(post)),
    );
  }

  void reconcileSavedItem(SavedPostItem item) {
    final current = state.requireValue;
    final previous = current.forUri(item.post.uri);
    if (previous.isPending ||
        (previous.initialized &&
            previous.isSaved &&
            previous.folderId == item.folderId &&
            previous.savedAt == item.savedAt)) {
      return;
    }
    state = AsyncData(
      current.put(
        item.post.uri,
        SavedPostPresentation(
          initialized: true,
          isSaved: true,
          revision: previous.revision + 1,
          folderId: item.folderId,
          savedAt: item.savedAt,
        ),
      ),
    );
  }

  void reconcileFolderDeletion(
    String folderId, {
    required bool deleteSaves,
  }) {
    final current = state.requireValue;
    final next = current.afterFolderDeletion(
      folderId,
      deleteSaves: deleteSaves,
    );
    if (identical(current, next)) return;
    state = AsyncData(next);
  }

  Future<void> save(Post post, String? folderId) =>
      _saveOrMove(post, folderId, SavedPostMutation.save);

  Future<void> move(SavedPostItem item, String? folderId) =>
      _saveOrMove(item.post, folderId, SavedPostMutation.move);

  Future<void> _saveOrMove(
    Post post,
    String? folderId,
    SavedPostMutation mutation,
  ) async {
    final generation = captureSavedPostAccountBoundary(ref);
    seedIfAbsent(post);
    final currentMap = state.requireValue;
    final previous = currentMap.forUri(post.uri);
    if (previous.isPending) return;
    state = AsyncData(
      currentMap.put(
        post.uri,
        previous.copyWith(pendingMutation: mutation, lastError: null),
      ),
    );

    try {
      final repository = await ref.read(
        accountSavedPostRepositoryProvider(account).future,
      );
      final result = await repository.save(post, folderId: folderId);
      if (!_isCurrent(post.uri, previous.revision, mutation, generation)) {
        return;
      }
      _put(
        post.uri,
        SavedPostPresentation(
          initialized: true,
          isSaved: true,
          revision: previous.revision + 1,
          folderId: result.folderId,
          savedAt: result.savedAt,
        ),
      );
    } on Object catch (error) {
      if (!_isCurrent(post.uri, previous.revision, mutation, generation)) {
        return;
      }
      _put(
        post.uri,
        previous.copyWith(pendingMutation: null, lastError: error),
      );
    }
  }

  Future<void> unsave(Post post) async {
    final generation = captureSavedPostAccountBoundary(ref);
    seedIfAbsent(post);
    final currentMap = state.requireValue;
    final previous = currentMap.forUri(post.uri);
    if (previous.isPending) return;
    state = AsyncData(
      currentMap.put(
        post.uri,
        previous.copyWith(
          isSaved: false,
          folderId: null,
          savedAt: null,
          pendingMutation: SavedPostMutation.unsave,
          lastError: null,
        ),
      ),
    );

    try {
      final repository = await ref.read(
        accountSavedPostRepositoryProvider(account).future,
      );
      await repository.unsave(post);
      if (!_isCurrent(
        post.uri,
        previous.revision,
        SavedPostMutation.unsave,
        generation,
      )) {
        return;
      }
      _put(
        post.uri,
        SavedPostPresentation(
          initialized: true,
          isSaved: false,
          revision: previous.revision + 1,
        ),
      );
    } on Object catch (error) {
      if (!_isCurrent(
        post.uri,
        previous.revision,
        SavedPostMutation.unsave,
        generation,
      )) {
        return;
      }
      _put(
        post.uri,
        previous.copyWith(pendingMutation: null, lastError: error),
      );
    }
  }

  bool _isCurrent(
    AtUri uri,
    int revision,
    SavedPostMutation mutation,
    int generation,
  ) {
    if (!isSavedPostAccountBoundaryCurrent(ref, generation) ||
        !state.hasValue) {
      return false;
    }
    final current = state.requireValue.forUri(uri);
    return current.revision == revision && current.pendingMutation == mutation;
  }

  void _put(AtUri uri, SavedPostPresentation presentation) {
    if (!ref.mounted || !state.hasValue) return;
    state = AsyncData(state.requireValue.put(uri, presentation));
  }
}

@riverpod
AsyncValue<SavedPostPresentation> savedPostPresentation(
  Ref ref,
  SavedPostKey key,
) => projectSavedPostPresentation(
  ref.watch(accountSavedPostStateProvider(key.account)),
  key.uri,
);

AsyncValue<SavedPostPresentation> projectSavedPostPresentation(
  AsyncValue<AccountSavedPostStateMap> source,
  AtUri uri,
) => source.whenData((value) => value.forUri(uri));
