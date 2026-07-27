import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_error.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_folder_dialogs.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_row.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_row_actions.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_sort_button.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SavedPostFolderScreen extends ConsumerStatefulWidget {
  const SavedPostFolderScreen({
    required this.account,
    required this.folder,
    super.key,
  });

  final AccountKey account;
  final SavedPostFolder folder;

  @override
  ConsumerState<SavedPostFolderScreen> createState() =>
      _SavedPostFolderScreenState();
}

class _SavedPostFolderScreenState extends ConsumerState<SavedPostFolderScreen> {
  SavedPostSort _sort = SavedPostSort.newest;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final key = SavedPostListKey(
      account: widget.account,
      scope: SavedPostScope.folder(widget.folder.id),
      sort: _sort,
    );
    final state = ref.watch(savedPostsProvider(key));
    final folderState = ref.watch(
      savedPostFoldersProvider(widget.account),
    );
    final folder =
        folderState.value?.folderById(widget.folder.id) ?? widget.folder;
    return Scaffold(
      appBar: AppBar(
        title: Text(folder.name),
        actions: [
          SavedPostSortButton(
            value: _sort,
            onChanged: (value) => setState(() => _sort = value),
          ),
          CraftskyContextMenuButton(
            tooltip: l10n.savedPostFolderActions,
            groups: [
              CraftskyContextMenuGroup(
                items: [
                  CraftskyContextMenuItem(
                    text: l10n.savedPostRenameFolder,
                    icon: Icons.edit_outlined,
                    onPressed: () => showRenameSavedPostFolderDialog(
                      context,
                      account: widget.account,
                      folder: folder,
                    ),
                  ),
                  CraftskyContextMenuItem(
                    text: l10n.savedPostDeleteFolder,
                    icon: Icons.delete_outline,
                    style: CraftskyContextMenuItemStyle.destructive,
                    onPressed: () async {
                      final deleted = await showDeleteSavedPostFolderDialog(
                        context,
                        account: widget.account,
                        folder: folder,
                      );
                      if (deleted &&
                          context.mounted &&
                          Navigator.canPop(context)) {
                        Navigator.of(context).pop();
                      }
                    },
                  ),
                ],
              ),
            ],
          ),
        ],
      ),
      body: switch (state) {
        AsyncData(:final value) => RefreshIndicator(
          onRefresh: ref.read(savedPostsProvider(key).notifier).refresh,
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            children: [
              if (value.items.isEmpty)
                Padding(
                  padding: const EdgeInsets.all(24),
                  child: Center(child: Text(l10n.savedPostsEmpty)),
                ),
              for (final item in value.items)
                SavedPostRow(
                  account: widget.account,
                  item: item,
                  onOpen: () => openSavedPost(context, item),
                  onMove: () => unawaited(
                    moveSavedPost(
                      context,
                      ref,
                      account: widget.account,
                      item: item,
                      sourceKey: key,
                    ),
                  ),
                  onUnsave: () => unawaited(
                    unsaveSavedPost(
                      context,
                      ref,
                      account: widget.account,
                      item: item,
                      sourceKey: key,
                    ),
                  ),
                ),
              if (value.incrementalError case final error?)
                _SavedPostListFailure(
                  error: error,
                  onRetry: ref.read(savedPostsProvider(key).notifier).loadMore,
                )
              else if (value.cursor != null)
                TextButton(
                  onPressed: value.isLoadingMore
                      ? null
                      : ref.read(savedPostsProvider(key).notifier).loadMore,
                  child: Text(l10n.savedPostsLoadMore),
                ),
            ],
          ),
        ),
        AsyncError(:final error) => Center(
          child: _SavedPostListFailure(
            error: error,
            onRetry: () => ref.invalidate(savedPostsProvider(key)),
          ),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

class _SavedPostListFailure extends StatelessWidget {
  const _SavedPostListFailure({required this.error, required this.onRetry});

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final failure = SavedPostFailure.from(
      error,
      operation: SavedPostOperation.loadPosts,
    );
    if (!failure.shouldPresent) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(failure.localizedMessage(l10n)),
        if (failure.canRetry)
          TextButton(onPressed: onRetry, child: Text(l10n.retryButton)),
      ],
    );
  }
}
