import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_error.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/models/saved_posts_overview.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_folder_dialogs.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_row.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_row_actions.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_sort_button.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:sliver_tools/sliver_tools.dart';

class SavedPostsPage extends ConsumerStatefulWidget {
  const SavedPostsPage({super.key});

  @override
  ConsumerState<SavedPostsPage> createState() => _SavedPostsPageState();
}

class _SavedPostsPageState extends ConsumerState<SavedPostsPage> {
  final _scrollController = ScrollController();
  SavedPostSort _sort = SavedPostSort.newest;

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final auth = ref.watch(authSessionProvider).value;
    final account = auth is SignedIn ? AccountKey(auth.did.toString()) : null;
    if (account == null) {
      return Scaffold(
        appBar: AppBar(
          leading: BackButton(
            onPressed: () => const ProfileRoute().go(context),
          ),
          title: Text(l10n.savedPostsTitle),
        ),
        body: const CraftskySkeletonList(
          itemBuilder: _buildSavedPostSkeleton,
        ),
      );
    }
    final folders = ref.watch(savedPostFoldersProvider(account));
    final key = SavedPostListKey(
      account: account,
      scope: const SavedPostScope.unfiled(),
      sort: _sort,
    );
    final posts = ref.watch(savedPostsProvider(key));

    return Scaffold(
      appBar: AppBar(
        leading: BackButton(
          onPressed: () => const ProfileRoute().go(context),
        ),
        title: Text(l10n.savedPostsTitle),
        actions: [
          IconButton(
            tooltip: l10n.savedPostNewFolder,
            icon: const Icon(CraftskyIconsBold.newFolder),
            onPressed: () => unawaited(
              showCreateSavedPostFolderDialog(context, account: account),
            ),
          ),
        ],
      ),
      body: switch ((folders, posts)) {
        (AsyncError(:final error), _) => _InitialError(
          failure: SavedPostFailure.from(
            error,
            operation: SavedPostOperation.loadFolders,
          ),
          onRetry: () {
            ref
              ..invalidate(savedPostFoldersProvider(account))
              ..invalidate(savedPostsProvider(key));
          },
        ),
        (_, AsyncError(:final error)) => _InitialError(
          failure: SavedPostFailure.from(
            error,
            operation: SavedPostOperation.loadPosts,
          ),
          onRetry: () {
            ref
              ..invalidate(savedPostFoldersProvider(account))
              ..invalidate(savedPostsProvider(key));
          },
        ),
        _ => _OverviewBody(
          account: account,
          folderState: folders.value,
          postState: posts.value,
          sort: _sort,
          scrollController: _scrollController,
          onSortChanged: (sort) => setState(() => _sort = sort),
          onRefresh: () async {
            await Future.wait([
              ref.read(savedPostFoldersProvider(account).notifier).refresh(),
              ref.read(savedPostsProvider(key).notifier).refresh(),
            ]);
          },
        ),
      },
    );
  }
}

class _OverviewBody extends ConsumerWidget {
  const _OverviewBody({
    required this.account,
    required this.folderState,
    required this.postState,
    required this.sort,
    required this.scrollController,
    required this.onSortChanged,
    required this.onRefresh,
  });

  final AccountKey account;
  final SavedPostFolderListState? folderState;
  final SavedPostListState? postState;
  final SavedPostSort sort;
  final ScrollController scrollController;
  final ValueChanged<SavedPostSort> onSortChanged;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    if (folderState == null && postState == null) {
      return RefreshIndicator(
        onRefresh: onRefresh,
        child: const CustomScrollView(
          physics: AlwaysScrollableScrollPhysics(),
          slivers: [
            CraftskySkeletonSliverList(
              itemBuilder: _buildSavedPostSkeleton,
            ),
          ],
        ),
      );
    }
    final overview = postState != null
        ? SavedPostsOverview.project(
            folders: folderState?.displayItems ?? const [],
            items: postState!.items,
            sort: sort,
          )
        : null;
    final isEmpty = folderState != null && (overview?.isEmpty ?? false);
    return RefreshIndicator(
      onRefresh: onRefresh,
      child: CustomScrollView(
        key: const PageStorageKey('saved-posts-overview'),
        controller: scrollController,
        physics: const AlwaysScrollableScrollPhysics(),
        slivers: [
          if (isEmpty)
            SliverFillRemaining(
              hasScrollBody: false,
              child: CraftskyEmptyState(
                icon: CraftskyIcons.savedSection,
                title: l10n.savedPostsTitle,
                subtitle: l10n.savedPostsEmpty,
              ),
            )
          else ...[
            MultiSliver(
              pushPinnedChildren: true,
              children: [
                SliverPinnedHeader(
                  child: _SavedPostsSectionHeader(
                    title: l10n.savedPostsFoldersHeading,
                  ),
                ),
                SliverList.builder(
                  itemCount: folderState?.displayItems.length ?? 0,
                  itemBuilder: (context, index) {
                    final folder = folderState!.displayItems[index];
                    return ListTile(
                      key: ValueKey('saved-overview-folder-${folder.id}'),
                      leading: const Icon(CraftskyIcons.folder),
                      title: Text(folder.name),
                      trailing: const Icon(CraftskyIconsBold.next),
                      onTap: () => SavedPostFolderRoute(
                        $extra: SavedPostFolderRouteData(folder: folder),
                      ).go(context),
                    );
                  },
                ),
                if (folderState == null)
                  const CraftskySkeletonSliverList(
                    itemBuilder: _buildSavedPostSkeleton,
                    itemCount: 3,
                  )
                else if (folderState!.incrementalError case final error?)
                  SliverToBoxAdapter(
                    child: _SavedPostFailureControl(
                      failure: SavedPostFailure.from(
                        error,
                        operation: SavedPostOperation.loadFolders,
                      ),
                      onPressed: ref
                          .read(savedPostFoldersProvider(account).notifier)
                          .retry,
                    ),
                  )
                else if (folderState!.cursor != null)
                  SliverToBoxAdapter(
                    child: TextButton(
                      onPressed: folderState!.isLoadingMore
                          ? null
                          : ref
                                .read(
                                  savedPostFoldersProvider(account).notifier,
                                )
                                .loadMore,
                      child: Text(l10n.savedPostLoadMoreFolders),
                    ),
                  ),
              ],
            ),
            if (postState == null || overview!.showUnfiled)
              MultiSliver(
                pushPinnedChildren: true,
                children: [
                  SliverPinnedHeader(
                    child: _SavedPostsSectionHeader(
                      title: l10n.savedPostsUnfiledHeading,
                      trailing: SavedPostSortButton(
                        value: sort,
                        onChanged: onSortChanged,
                      ),
                    ),
                  ),
                  SliverList.builder(
                    itemCount: overview?.unfiledItems.length ?? 0,
                    itemBuilder: (context, index) {
                      final item = overview!.unfiledItems[index];
                      return SavedPostRow(
                        account: account,
                        item: item,
                        onOpen: () => openSavedPost(context, item),
                        onMove: () => unawaited(
                          moveSavedPost(
                            context,
                            ref,
                            account: account,
                            item: item,
                            sourceKey: SavedPostListKey(
                              account: account,
                              scope: const SavedPostScope.unfiled(),
                              sort: sort,
                            ),
                          ),
                        ),
                        onUnsave: () => unawaited(
                          unsaveSavedPost(
                            context,
                            ref,
                            account: account,
                            item: item,
                            sourceKey: SavedPostListKey(
                              account: account,
                              scope: const SavedPostScope.unfiled(),
                              sort: sort,
                            ),
                          ),
                        ),
                      );
                    },
                  ),
                  if (postState == null)
                    const CraftskySkeletonSliverList(
                      itemBuilder: _buildSavedPostSkeleton,
                    )
                  else if (postState!.incrementalError case final error?)
                    SliverToBoxAdapter(
                      child: _SavedPostFailureControl(
                        failure: SavedPostFailure.from(
                          error,
                          operation: SavedPostOperation.loadPosts,
                        ),
                        onPressed: ref
                            .read(
                              savedPostsProvider(
                                SavedPostListKey(
                                  account: account,
                                  scope: const SavedPostScope.unfiled(),
                                  sort: sort,
                                ),
                              ).notifier,
                            )
                            .loadMore,
                      ),
                    )
                  else if (postState!.cursor != null)
                    SliverToBoxAdapter(
                      child: TextButton(
                        onPressed: postState!.isLoadingMore
                            ? null
                            : ref
                                  .read(
                                    savedPostsProvider(
                                      SavedPostListKey(
                                        account: account,
                                        scope: const SavedPostScope.unfiled(),
                                        sort: sort,
                                      ),
                                    ).notifier,
                                  )
                                  .loadMore,
                        child: Text(l10n.savedPostsLoadMore),
                      ),
                    ),
                ],
              ),
          ],
        ],
      ),
    );
  }
}

Widget _buildSavedPostSkeleton(BuildContext context, int index) =>
    const ManagementRowSkeleton();

class _SavedPostsSectionHeader extends StatelessWidget {
  const _SavedPostsSectionHeader({required this.title, this.trailing});

  final String title;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return ColoredBox(
      color: theme.scaffoldBackgroundColor,
      child: Padding(
        padding: EdgeInsets.fromLTRB(
          spacing.sp4,
          spacing.sp3,
          spacing.sp4,
          spacing.sp2,
        ),
        child: Row(
          children: [
            Expanded(
              child: Semantics(
                header: true,
                child: Text(title, style: theme.textTheme.titleMedium),
              ),
            ),
            ?trailing,
          ],
        ),
      ),
    );
  }
}

class _InitialError extends StatelessWidget {
  const _InitialError({required this.failure, required this.onRetry});

  final SavedPostFailure failure;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Center(
    child: _SavedPostFailureControl(
      failure: failure,
      onPressed: onRetry,
    ),
  );
}

class _SavedPostFailureControl extends StatelessWidget {
  const _SavedPostFailureControl({
    required this.failure,
    required this.onPressed,
  });

  final SavedPostFailure failure;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    if (!failure.shouldPresent) return const SizedBox.shrink();
    final l10n = AppLocalizations.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(failure.localizedMessage(l10n)),
        if (failure.canRetry)
          TextButton(onPressed: onPressed, child: Text(l10n.retryButton)),
      ],
    );
  }
}
