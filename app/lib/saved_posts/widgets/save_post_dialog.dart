import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_error.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/save_post_dialog_controller.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/subscription_build_config.dart';
import 'package:craftsky_app/subscriptions/widgets/plus_action_icon.dart';
import 'package:craftsky_app/subscriptions/widgets/plus_feature_lock.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

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
  const new({
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
    final session = ref
        .watch(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session;
    final access = subscriptionsEnabled && session?.account == account
        ? ref.watch(subscriptionAccessProvider(session!))
        : null;
    final hasPlus = switch (access) {
      AsyncData(:final value, isLoading: false, hasError: false) =>
        value.allowsPlus,
      _ => false,
    };
    final folders = hasPlus
        ? ref.watch(savedPostFoldersProvider(account))
        : null;
    if (hasPlus) {
      ref.listen(savedPostFoldersProvider(account), (_, next) {
        ref.read(provider.notifier).folderDeleted(next.value?.deletedFolderId);
      });
    }

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
                    if (hasPlus)
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
                        null => const SizedBox.shrink(),
                      },
                    if (!hasPlus)
                      PlusFeatureLock(
                        feature: l10n.savedPostNewFolder,
                        access: access,
                        onUnlocked: ref
                            .read(provider.notifier)
                            .beginCreatingFolder,
                        onRetry: session == null
                            ? null
                            : () => ref.invalidate(
                                subscriptionAccessProvider(session),
                              ),
                        showPlusBadge: false,
                        child: TextButton.icon(
                          onPressed: () {},
                          icon: const PlusActionIcon(
                            icon: CraftskyIconsBold.newFolder,
                          ),
                          label: Text(l10n.savedPostNewFolder),
                        ),
                      )
                    else if (state.isCreatingFolder) ...[
                      SizedBox(
                        key: const Key('saved-folder-create-spacing'),
                        height: spacing.sp4,
                      ),
                      CraftskyTextInput(
                        label: l10n.savedPostFolderNameHint,
                        textCapitalization: TextCapitalization.words,
                        enabled: !state.isCreatePending,
                        onChanged: ref.read(provider.notifier).updateCreateName,
                        errorText: state.createError == null
                            ? null
                            : l10n.savedPostCreateFolderError,
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
        primaryAction: CraftskyDialogPrimaryAction(
          label: state.isCreatingFolder
              ? l10n.savedPostCreateFolderAction
              : savedItem == null
              ? l10n.savedPostSaveAction
              : l10n.savedPostMoveAction,
          onPressed: state.isCreatingFolder
              ? state.isCreatePending
                    ? null
                    : ref.read(provider.notifier).createFolder
              : state.canConfirm
              ? () => savedItem == null
                    ? ref.read(provider.notifier).confirmSave(post)
                    : ref.read(provider.notifier).confirmMove(savedItem!)
              : null,
          child: state.isConfirming || state.isCreatePending
              ? const SizedBox.square(
                  dimension: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : null,
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
        ],
      ),
    );
  }
}

class _FolderPaginationControls extends ConsumerWidget {
  const new({
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
  const new({required this.failure, required this.onRetry});

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
