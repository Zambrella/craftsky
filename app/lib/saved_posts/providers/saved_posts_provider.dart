import 'dart:async';

import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/account_saved_post_state_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'saved_posts_provider.g.dart';
part 'saved_posts_provider.mapper.dart';

@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class SavedPostListState with SavedPostListStateMappable {
  SavedPostListState({
    required List<SavedPostItem> items,
    this.cursor,
    this.isLoadingMore = false,
    this.incrementalError,
  }) : items = List.unmodifiable(items);

  factory SavedPostListState.fromPage(SavedPostPage page) =>
      SavedPostListState(items: page.items, cursor: page.cursor);

  final List<SavedPostItem> items;
  final String? cursor;
  final bool isLoadingMore;
  final Object? incrementalError;

  SavedPostListState appendPage(SavedPostPage page) {
    final knownUris = items.map((item) => item.post.uri).toSet();
    return SavedPostListState(
      items: [
        ...items,
        ...page.items.where((item) => knownUris.add(item.post.uri)),
      ],
      cursor: page.cursor,
    );
  }

  SavedPostListState withIncrementalError(Object error) =>
      copyWith(isLoadingMore: false, incrementalError: error);

  @override
  String toString() => 'SavedPostListState(<redacted>)';
}

bool isInvalidSavedPostCursorError(Object error) =>
    error is ApiBadRequest && error.code == 'invalid_cursor';

@riverpod
class SavedPosts extends _$SavedPosts {
  @override
  Future<SavedPostListState> build(SavedPostListKey key) async {
    final generation = captureSavedPostAccountBoundary(ref);
    ref.watch(savedPostAccountBoundaryProvider);
    final repository = await ref.watch(
      accountSavedPostRepositoryProvider(key.account).future,
    );
    final page = await repository.list(scope: key.scope, sort: key.sort);
    if (isSavedPostAccountBoundaryCurrent(ref, generation)) {
      _reconcilePage(page);
    }
    return SavedPostListState.fromPage(page);
  }

  Future<void> loadMore() async {
    final generation = captureSavedPostAccountBoundary(ref);
    final current = state.value;
    if (current == null || current.cursor == null || current.isLoadingMore) {
      return;
    }
    state = AsyncData(
      current.copyWith(isLoadingMore: true, incrementalError: null),
    );
    try {
      final repository = await ref.read(
        accountSavedPostRepositoryProvider(key.account).future,
      );
      final page = await repository.list(
        scope: key.scope,
        sort: key.sort,
        cursor: current.cursor,
      );
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return;
      _reconcilePage(page);
      state = AsyncData(current.appendPage(page));
    } on Object catch (error) {
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return;
      if (isInvalidSavedPostCursorError(error)) {
        await _restart(current, generation);
      } else {
        state = AsyncData(current.withIncrementalError(error));
      }
    }
  }

  Future<void> refresh() => _restart(
    state.value,
    captureSavedPostAccountBoundary(ref),
  );

  Future<void> _restart(
    SavedPostListState? previous,
    int generation,
  ) async {
    try {
      final repository = await ref.read(
        accountSavedPostRepositoryProvider(key.account).future,
      );
      final page = await repository.list(scope: key.scope, sort: key.sort);
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return;
      _reconcilePage(page);
      state = AsyncData(SavedPostListState.fromPage(page));
    } on Object catch (error, stackTrace) {
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return;
      state = previous == null
          ? AsyncError(error, stackTrace)
          : AsyncData(previous.withIncrementalError(error));
    }
  }

  void removeConfirmed(AtUri uri) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        items: current.items.where((item) => item.post.uri != uri).toList(),
      ),
    );
  }

  void upsertConfirmed(SavedPostItem item) {
    final current = state.value;
    if (current == null) return;
    final remaining = current.items
        .where((existing) => existing.post.uri != item.post.uri)
        .toList();
    final items = [...remaining, item]
      ..sort(
        (a, b) => key.sort == SavedPostSort.newest
            ? b.savedAt.compareTo(a.savedAt)
            : a.savedAt.compareTo(b.savedAt),
      );
    state = AsyncData(current.copyWith(items: items));
  }

  void _reconcilePage(SavedPostPage page) {
    final notifier = ref.read(
      accountSavedPostStateProvider(key.account).notifier,
    );
    page.items.forEach(notifier.reconcileSavedItem);
  }
}

void refreshLiveSavedPostListsAfterFolderDeletion(
  Ref ref, {
  required SavedPostListKey folderKey,
  required bool deleteSaves,
}) {
  final folderId = folderKey.scope.folderId;
  if (folderId == null) return;

  for (final sort in SavedPostSort.values) {
    final sourceKey = SavedPostListKey(
      account: folderKey.account,
      scope: SavedPostScope.folder(folderId),
      sort: sort,
    );
    if (ref.exists(savedPostsProvider(sourceKey))) {
      ref.invalidate(savedPostsProvider(sourceKey));
    }

    if (!deleteSaves) {
      final destinationKey = SavedPostListKey(
        account: folderKey.account,
        scope: const SavedPostScope.unfiled(),
        sort: sort,
      );
      if (ref.exists(savedPostsProvider(destinationKey))) {
        ref.invalidate(savedPostsProvider(destinationKey));
      }
    }
  }
}
