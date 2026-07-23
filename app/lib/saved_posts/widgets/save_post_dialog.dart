import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_error.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/save_post_dialog_controller.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

Future<bool?> showSavePostDialog(
  BuildContext context, {
  required AccountKey account,
  required Post post,
  String? initialFolderId,
}) => showDialog<bool>(
  context: context,
  builder: (_) => SavePostDialog(
    account: account,
    post: post,
    initialFolderId: initialFolderId,
  ),
);

Future<bool?> showMoveSavedPostDialog(
  BuildContext context, {
  required AccountKey account,
  required SavedPostItem item,
}) => showDialog<bool>(
  context: context,
  builder: (_) => SavePostDialog(
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

    return AlertDialog(
      title: Text(
        savedItem == null ? l10n.savedPostSaveAction : l10n.savedPostMoveTitle,
      ),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 360),
            child: SingleChildScrollView(
              child: RadioGroup<String?>(
                groupValue: state.selectedFolderId,
                onChanged: ref.read(provider.notifier).selectFolder,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    RadioListTile<String?>(
                      value: null,
                      enabled: !state.isConfirming,
                      title: Text(l10n.savedPostNoFolder),
                    ),
                    switch (folders) {
                      AsyncData(:final value) => _FolderOptions(
                        account: account,
                        state: value,
                        enabled: !state.isConfirming,
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
                      TextField(
                        enabled: !state.isCreatePending,
                        onChanged: ref.read(provider.notifier).updateCreateName,
                        decoration: InputDecoration(
                          labelText: l10n.savedPostFolderNameHint,
                          errorText: state.createError == null
                              ? null
                              : l10n.savedPostCreateFolderError,
                        ),
                      ),
                      const SizedBox(height: 8),
                      FilledButton(
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
        FilledButton(
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
    );
  }
}

class _FolderOptions extends ConsumerWidget {
  const _FolderOptions({
    required this.account,
    required this.state,
    required this.enabled,
  });

  final AccountKey account;
  final SavedPostFolderListState state;
  final bool enabled;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        for (final folder in state.displayItems)
          RadioListTile<String?>(
            key: ValueKey('saved-folder-${folder.id}'),
            value: folder.id,
            enabled: enabled,
            title: Text(folder.name),
          ),
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
