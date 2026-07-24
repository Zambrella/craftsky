import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_error.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/save_post_dialog_controller.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

Future<bool?> showSavePostDialog(
  BuildContext context, {
  required AccountKey account,
  required Post post,
  String? initialFolderId,
}) => showCraftskyModal<bool>(
  context,
  builder: (dialogContext) => SavePostDialog(
    account: account,
    post: post,
    initialFolderId: initialFolderId,
  ),
);

Future<bool?> showMoveSavedPostDialog(
  BuildContext context, {
  required AccountKey account,
  required SavedPostItem item,
}) => showCraftskyModal<bool>(
  context,
  builder: (dialogContext) => SavePostDialog(
    account: account,
    post: item.post,
    initialFolderId: item.folderId,
    savedItem: item,
  ),
);

class SavePostDialog extends ConsumerWidget {
  const SavePostDialog({
    required this.account,
    required this.post,
    this.initialFolderId,
    this.savedItem,
    super.key,
  });

  final AccountKey account;
  final Post post;
  final String? initialFolderId;
  final SavedPostItem? savedItem;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final provider = savePostDialogControllerProvider(
      SavePostDialogKey(
        account: account,
        uri: post.uri,
        initialFolderId: initialFolderId,
      ),
    );
    final state = ref.watch(provider);
    final folders = ref.watch(savedPostFoldersProvider(account));

    ref.listen(provider, (_, next) {
      if (next.isConfirmed && context.mounted) {
        Navigator.of(context).pop(true);
      }
    });

    return PopScope(
      canPop: !state.isConfirming,
      child: CraftskyDialog(
        title: savedItem == null
            ? l10n.savedPostSaveAction
            : l10n.savedPostMoveTitle,
        body: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ConstrainedBox(
              constraints: const BoxConstraints(maxHeight: 360),
              child: SingleChildScrollView(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    CraftskySingleSelectInput<String?>(
                      label: l10n.savedPostFolderSelectionLabel,
                      value: state.selectedFolderId,
                      enabled: !state.isConfirming,
                      keyPrefix: 'saved-folder',
                      searchThreshold: null,
                      options: [
                        CraftskySelectOption<String?>(
                          value: null,
                          label: l10n.savedPostNoFolder,
                        ),
                        if (folders case AsyncData(:final value))
                          for (final folder in value.displayItems)
                            CraftskySelectOption<String?>(
                              value: folder.id,
                              label: folder.name,
                            ),
                      ],
                      onChanged: ref.read(provider.notifier).selectFolder,
                    ),
                    switch (folders) {
                      AsyncData(:final value) => _FolderPaginationControls(
                        account: account,
                        state: value,
                      ),
                      AsyncLoading() => const Center(
                        child: CircularProgressIndicator(),
                      ),
                      AsyncError(:final error) => _FolderFailure(
                        failure: SavedPostFailure.from(
                          error,
                          operation: SavedPostOperation.loadFolders,
                        ),
                        onRetry: () => ref.invalidate(
                          savedPostFoldersProvider(account),
                        ),
                      ),
                    },
                    if (state.isCreatingFolder) ...[
                      SizedBox(
                        key: const Key('saved-folder-create-spacing'),
                        height: spacing.sp4,
                      ),
                      CraftskyTextInput(
                        label: l10n.savedPostFolderNameHint,
                        enabled: !state.isCreatePending,
                        onChanged: ref.read(provider.notifier).updateCreateName,
                        errorText: state.createError == null
                            ? null
                            : l10n.savedPostCreateFolderError,
                      ),
                      const SizedBox(height: 8),
                      ChunkyButton(
                        onPressed: state.isCreatePending
                            ? null
                            : ref.read(provider.notifier).createFolder,
                        child: state.isCreatePending
                            ? const SizedBox.square(
                                dimension: 20,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                ),
                              )
                            : Text(l10n.savedPostCreateFolderAction),
                      ),
                    ] else
                      TextButton(
                        onPressed: state.isConfirming
                            ? null
                            : ref.read(provider.notifier).beginCreatingFolder,
                        child: Text(l10n.savedPostNewFolder),
                      ),
                  ],
                ),
              ),
            ),
            if (state.confirmError != null)
              Text(
                l10n.savedPostConfirmError,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: state.isConfirming
                ? null
                : () {
                    ref.read(provider.notifier).cancel();
                    Navigator.of(context).pop(false);
                  },
            child: Text(l10n.actionCancel),
          ),
          ChunkyButton(
            onPressed: state.canConfirm
                ? () => savedItem == null
                      ? ref.read(provider.notifier).confirmSave(post)
                      : ref.read(provider.notifier).confirmMove(savedItem!)
                : null,
            child: state.isConfirming
                ? const SizedBox.square(
                    dimension: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : Text(
                    savedItem == null
                        ? l10n.savedPostSaveAction
                        : l10n.savedPostMoveAction,
                  ),
          ),
        ],
      ),
    );
  }
}

class _FolderPaginationControls extends ConsumerWidget {
  const _FolderPaginationControls({
    required this.account,
    required this.state,
  });

  final AccountKey account;
  final SavedPostFolderListState state;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (state.incrementalError case final error?)
          _FolderFailure(
            failure: SavedPostFailure.from(
              error,
              operation: SavedPostOperation.loadFolders,
            ),
            onRetry: () =>
                ref.read(savedPostFoldersProvider(account).notifier).retry(),
          )
        else if (state.isLoadingMore)
          const Center(child: CircularProgressIndicator())
        else if (state.cursor != null)
          TextButton(
            onPressed: () =>
                ref.read(savedPostFoldersProvider(account).notifier).loadMore(),
            child: Text(l10n.savedPostLoadMoreFolders),
          ),
      ],
    );
  }
}

class _FolderFailure extends StatelessWidget {
  const _FolderFailure({required this.failure, required this.onRetry});

  final SavedPostFailure failure;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    if (!failure.shouldPresent) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(failure.localizedMessage(l10n)),
        if (failure.canRetry)
          TextButton(onPressed: onRetry, child: Text(l10n.retryButton)),
      ],
    );
  }
}
