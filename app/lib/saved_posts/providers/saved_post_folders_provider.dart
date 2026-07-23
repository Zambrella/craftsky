import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_error.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/account_saved_post_state_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'saved_post_folders_provider.g.dart';
part 'saved_post_folders_provider.mapper.dart';

@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class SavedPostFolderListState with SavedPostFolderListStateMappable {
  SavedPostFolderListState({
    required List<SavedPostFolder> items,
    this.cursor,
    this.isLoadingMore = false,
    this.incrementalError,
    this.mutationFailure,
    this.deletedFolderId,
    Map<String, SavedPostFolder> retainedFolders = const {},
  }) : items = List.unmodifiable(items),
       retainedFolders = Map.unmodifiable(retainedFolders);

  factory SavedPostFolderListState.fromPage(SavedPostFolderPage page) =>
      SavedPostFolderListState(items: page.items, cursor: page.cursor);

  final List<SavedPostFolder> items;
  final String? cursor;
  final bool isLoadingMore;
  final Object? incrementalError;
  final SavedPostFailure? mutationFailure;
  final String? deletedFolderId;
  final Map<String, SavedPostFolder> retainedFolders;

  List<SavedPostFolder> get displayItems => [
    ...items,
    ...retainedFolders.values.where(
      (retained) => !items.any((item) => item.id == retained.id),
    ),
  ];

  SavedPostFolder? folderById(String id) {
    for (final folder in items) {
      if (folder.id == id) return folder;
    }
    return retainedFolders[id];
  }

  SavedPostFolderListState appendPage(SavedPostFolderPage page) {
    final knownIds = items.map((folder) => folder.id).toSet();
    final nextItems = [
      ...items,
      ...page.items.where((folder) => knownIds.add(folder.id)),
    ];
    final nextRetained = {...retainedFolders}
      ..removeWhere((id, _) => knownIds.contains(id));
    return SavedPostFolderListState(
      items: nextItems,
      cursor: page.cursor,
      mutationFailure: mutationFailure,
      deletedFolderId: deletedFolderId,
      retainedFolders: nextRetained,
    );
  }

  SavedPostFolderListState afterConfirmedMutation({
    SavedPostFolder? retain,
    String? deletedFolderId,
  }) {
    final nextRetained = {...retainedFolders};
    if (retain != null) nextRetained[retain.id] = retain;
    return SavedPostFolderListState(
      items: items,
      deletedFolderId: deletedFolderId,
      retainedFolders: nextRetained,
    );
  }

  SavedPostFolderListState restartPage(SavedPostFolderPage page) {
    final pageIds = page.items.map((folder) => folder.id).toSet();
    return SavedPostFolderListState(
      items: page.items,
      cursor: page.cursor,
      deletedFolderId: deletedFolderId,
      retainedFolders: {...retainedFolders}
        ..removeWhere((id, _) => pageIds.contains(id)),
    );
  }

  SavedPostFolderListState removeFolder(String id) => SavedPostFolderListState(
    items: items.where((folder) => folder.id != id).toList(),
    cursor: cursor,
    retainedFolders: {...retainedFolders}..remove(id),
  );

  SavedPostFolderListState replaceFolder(SavedPostFolder folder) {
    final existsInItems = items.any((item) => item.id == folder.id);
    return SavedPostFolderListState(
      items: [
        for (final item in items) item.id == folder.id ? folder : item,
      ],
      retainedFolders: {
        ...retainedFolders,
        if (!existsInItems) folder.id: folder,
      },
    );
  }

  @override
  String toString() => 'SavedPostFolderListState(<redacted>)';
}

@riverpod
class SavedPostFolders extends _$SavedPostFolders {
  @override
  Future<SavedPostFolderListState> build(AccountKey account) async {
    ref.watch(savedPostAccountBoundaryProvider);
    final repository = await ref.watch(
      accountSavedPostRepositoryProvider(account).future,
    );
    final page = await repository.listFolders();
    return SavedPostFolderListState.fromPage(page);
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
        accountSavedPostRepositoryProvider(account).future,
      );
      final page = await repository.listFolders(cursor: current.cursor);
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return;
      state = AsyncData(current.appendPage(page));
    } on Object catch (error) {
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return;
      if (error is ApiBadRequest && error.code == 'invalid_cursor') {
        await _restart(current, generation);
      } else {
        state = AsyncData(
          current.copyWith(isLoadingMore: false, incrementalError: error),
        );
      }
    }
  }

  Future<void> retry() {
    final current = state.value;
    if (current == null || current.incrementalError == null) {
      return Future.value();
    }
    if (current.cursor != null) return loadMore();
    return _restart(current, captureSavedPostAccountBoundary(ref));
  }

  Future<SavedPostFolder?> create(String name) async {
    final generation = captureSavedPostAccountBoundary(ref);
    _clearMutationFailure();
    try {
      final repository = await ref.read(
        accountSavedPostRepositoryProvider(account).future,
      );
      final folder = await repository.createFolder(name);
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return null;
      final current = state.value;
      if (current != null) {
        final awaitingRestart = current.afterConfirmedMutation(retain: folder);
        state = AsyncData(awaitingRestart);
        await _restart(awaitingRestart, generation);
      }
      return folder;
    } on Object catch (error) {
      _recordMutationFailure(
        error,
        operation: SavedPostOperation.createFolder,
        generation: generation,
      );
      return null;
    }
  }

  Future<SavedPostFolder?> rename(String id, String name) async {
    final generation = captureSavedPostAccountBoundary(ref);
    _clearMutationFailure();
    String normalized;
    try {
      normalized = normalizeSavedPostFolderName(name);
    } on SavedPostFolderNameException {
      return null;
    }
    try {
      final repository = await ref.read(
        accountSavedPostRepositoryProvider(account).future,
      );
      final folder = await repository.renameFolder(id, normalized);
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return null;
      final current = state.value;
      if (current != null) {
        final awaitingRestart = current
            .replaceFolder(folder)
            .afterConfirmedMutation(retain: folder);
        state = AsyncData(awaitingRestart);
        await _restart(awaitingRestart, generation);
      }
      return folder;
    } on Object catch (error) {
      _recordMutationFailure(
        error,
        operation: SavedPostOperation.renameFolder,
        generation: generation,
      );
      return null;
    }
  }

  Future<bool> delete(String id, {required bool deleteSaves}) async {
    final generation = captureSavedPostAccountBoundary(ref);
    _clearMutationFailure();
    try {
      final repository = await ref.read(
        accountSavedPostRepositoryProvider(account).future,
      );
      await repository.deleteFolder(id, deleteSaves: deleteSaves);
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return false;
      ref
          .read(accountSavedPostStateProvider(account).notifier)
          .reconcileFolderDeletion(id, deleteSaves: deleteSaves);
      refreshLiveSavedPostListsAfterFolderDeletion(
        ref,
        folderKey: SavedPostListKey(
          account: account,
          scope: SavedPostScope.folder(id),
          sort: SavedPostSort.newest,
        ),
        deleteSaves: deleteSaves,
      );
      final current = state.value;
      if (current != null) {
        final awaitingRestart = current
            .removeFolder(id)
            .afterConfirmedMutation(deletedFolderId: id);
        state = AsyncData(awaitingRestart);
        await _restart(awaitingRestart, generation);
      }
      return true;
    } on Object catch (error) {
      _recordMutationFailure(
        error,
        operation: SavedPostOperation.deleteFolder,
        generation: generation,
      );
      return false;
    }
  }

  Future<void> refresh() => _restart(
    state.value,
    captureSavedPostAccountBoundary(ref),
  );

  Future<void> _restart(
    SavedPostFolderListState? previous,
    int generation,
  ) async {
    try {
      final repository = await ref.read(
        accountSavedPostRepositoryProvider(account).future,
      );
      final page = await repository.listFolders();
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return;
      state = AsyncData(
        previous == null
            ? SavedPostFolderListState.fromPage(page)
            : previous.restartPage(page),
      );
    } on Object catch (error, stackTrace) {
      if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return;
      state = previous == null
          ? AsyncError(error, stackTrace)
          : AsyncData(
              previous.copyWith(
                isLoadingMore: false,
                incrementalError: error,
              ),
            );
    }
  }

  void _clearMutationFailure() {
    final current = state.value;
    if (current == null || current.mutationFailure == null) return;
    state = AsyncData(current.copyWith(mutationFailure: null));
  }

  void _recordMutationFailure(
    Object error, {
    required SavedPostOperation operation,
    required int generation,
  }) {
    if (!isSavedPostAccountBoundaryCurrent(ref, generation)) return;
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        mutationFailure: SavedPostFailure.from(error, operation: operation),
      ),
    );
  }
}
