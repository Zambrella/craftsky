import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_error.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/account_saved_post_state_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'save_post_dialog_controller.g.dart';
part 'save_post_dialog_controller.mapper.dart';

enum SavePostDialogError { invalidFolderName, createFailed, confirmFailed }

@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class SavePostDialogState with SavePostDialogStateMappable {
  const SavePostDialogState({
    required this.selectedFolderId,
    this.isCreatingFolder = false,
    this.createName = '',
    this.isCreatePending = false,
    this.isConfirming = false,
    this.isConfirmed = false,
    this.isCancelled = false,
    this.createError,
    this.confirmError,
  });

  final String? selectedFolderId;
  final bool isCreatingFolder;
  final String createName;
  final bool isCreatePending;
  final bool isConfirming;
  final bool isConfirmed;
  final bool isCancelled;
  final SavePostDialogError? createError;
  final SavePostDialogError? confirmError;

  bool get canConfirm => !isConfirming && !isConfirmed && !isCancelled;

  @override
  String toString() => 'SavePostDialogState(<redacted>)';
}

@riverpod
class SavePostDialogController extends _$SavePostDialogController {
  @override
  SavePostDialogState build(SavePostDialogKey key) {
    ref
      ..watch(savedPostAccountBoundaryProvider)
      ..listen(savedPostFoldersProvider(key.account), (_, next) {
        final deletedFolderId = next.value?.deletedFolderId;
        if (deletedFolderId != null &&
            deletedFolderId == state.selectedFolderId &&
            !state.isConfirming &&
            !state.isCancelled) {
          state = state.copyWith(selectedFolderId: null);
        }
      });
    return SavePostDialogState(selectedFolderId: key.initialFolderId);
  }

  void selectFolder(String? folderId) {
    if (state.isConfirming || state.isCancelled) return;
    state = state.copyWith(selectedFolderId: folderId, confirmError: null);
  }

  void beginCreatingFolder() {
    if (state.isCreatePending || state.isCancelled) return;
    state = state.copyWith(isCreatingFolder: true, createError: null);
  }

  void updateCreateName(String value) {
    if (state.isCreatePending || state.isCancelled) return;
    state = state.copyWith(createName: value, createError: null);
  }

  Future<SavedPostFolder?> createFolder() async {
    final generation = captureSavedPostAccountBoundary(ref);
    if (state.isCreatePending || state.isCancelled) return null;
    final name = _validatedName(state.createName);
    if (name == null) return null;

    state = state.copyWith(isCreatePending: true, createError: null);
    try {
      final folder = await ref
          .read(savedPostFoldersProvider(key.account).notifier)
          .create(name);
      if (!isSavedPostAccountBoundaryCurrent(ref, generation) ||
          state.isCancelled) {
        return null;
      }
      if (folder == null) {
        final failure = ref
            .read(savedPostFoldersProvider(key.account))
            .value
            ?.mutationFailure;
        if (failure != null && !failure.shouldPresent) {
          state = state.copyWith(isCreatePending: false, createError: null);
          return null;
        }
        throw const _FolderCreationFailed();
      }
      state = state.copyWith(
        selectedFolderId: folder.id,
        isCreatingFolder: false,
        createName: '',
        isCreatePending: false,
        createError: null,
      );
      return folder;
    } on Object {
      if (!isSavedPostAccountBoundaryCurrent(ref, generation) ||
          state.isCancelled) {
        return null;
      }
      state = state.copyWith(
        isCreatePending: false,
        createError: SavePostDialogError.createFailed,
      );
      return null;
    }
  }

  Future<void> confirmSave(Post post) => _confirm(
    () => ref
        .read(accountSavedPostStateProvider(key.account).notifier)
        .save(post, state.selectedFolderId),
  );

  Future<void> confirmMove(SavedPostItem item) => _confirm(
    () => ref
        .read(accountSavedPostStateProvider(key.account).notifier)
        .move(item, state.selectedFolderId),
  );

  Future<void> _confirm(Future<void> Function() mutation) async {
    final generation = captureSavedPostAccountBoundary(ref);
    if (!state.canConfirm) return;
    final selectedFolderId = state.selectedFolderId;
    state = state.copyWith(isConfirming: true, confirmError: null);

    await mutation();
    if (!isSavedPostAccountBoundaryCurrent(ref, generation) ||
        state.isCancelled) {
      return;
    }

    final presentation = ref
        .read(
          savedPostPresentationProvider(
            SavedPostKey(account: key.account, uri: key.uri),
          ),
        )
        .value;
    final confirmed =
        presentation != null &&
        !presentation.isPending &&
        !presentation.hasError &&
        presentation.isSaved &&
        presentation.folderId == selectedFolderId;
    final failure = presentation?.lastError == null
        ? null
        : SavedPostFailure.from(
            presentation!.lastError!,
            operation: SavedPostOperation.saveOrMove,
          );
    state = state.copyWith(
      isConfirming: false,
      isConfirmed: confirmed,
      confirmError: confirmed || !(failure?.shouldPresent ?? true)
          ? null
          : SavePostDialogError.confirmFailed,
    );
  }

  String? _validatedName(String value) {
    try {
      return normalizeSavedPostFolderName(value);
    } on SavedPostFolderNameException {
      state = state.copyWith(
        isCreatingFolder: true,
        createError: SavePostDialogError.invalidFolderName,
      );
      return null;
    }
  }

  void cancel() {
    if (state.isCreatePending || state.isConfirming) return;
    state = state.copyWith(isCancelled: true);
  }
}

final class _FolderCreationFailed implements Exception {
  const _FolderCreationFailed();
}
