import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/feed/widgets/post_type_chooser.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/report_flow.dart';
import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/shared/widgets/scroll_to_top_button.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _autoLoadMoreThreshold = 3;

class FeedPage extends ConsumerStatefulWidget {
  const FeedPage({super.key});

  @override
  ConsumerState<FeedPage> createState() => _FeedPageState();
}

class _FeedPageState extends ConsumerState<FeedPage> {
  static const _scrollToTopThreshold = 200.0;
  final _scrollController = ScrollController();
  var _isPastScrollThreshold = false;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_handleScroll);
  }

  @override
  void dispose() {
    _scrollController
      ..removeListener(_handleScroll)
      ..dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final timelineAsync = ref.watch(timelineProvider);
    final hasItems = timelineAsync.value?.items.isNotEmpty ?? false;
    final isCompact = FormFactor.fromWidth(
      MediaQuery.sizeOf(context).width,
    ).isSmall;
    ref
      ..listen(deletePostProvider, (previous, next) {
        switch ((previous, next)) {
          case (AsyncLoading(), AsyncData(value: != null)):
            context.showInfo(l10n.postDeleteSuccess);
            ref.read(deletePostProvider.notifier).reset();
          case (AsyncLoading(), AsyncError()):
            context.showError(l10n.postDeleteError);
            ref.read(deletePostProvider.notifier).reset();
          case _:
            break;
        }
      })
      ..listen(toggleLikePostProvider, (previous, next) {
        if (next.hasError) {
          context.showError(l10n.postLikeError);
          ref.read(toggleLikePostProvider.notifier).reset();
        }
      });
    return Scaffold(
      floatingActionButton: isCompact
          ? Builder(
              builder: (buttonContext) => CraftskyFloatingActionButton.extended(
                tooltip: l10n.postComposeAction,
                onPressed: () => _openComposer(buttonContext),
                icon: const Icon(CraftskyIconsBold.add),
                label: Text(l10n.postComposeAction),
              ),
            )
          : null,
      body: Stack(
        children: [
          RefreshIndicator(
            edgeOffset: MediaQuery.paddingOf(context).top + kToolbarHeight,
            onRefresh: () async {
              final _ = await ref.refresh(timelineProvider.future);
            },
            child: CustomScrollView(
              controller: _scrollController,
              physics: const AlwaysScrollableScrollPhysics(),
              slivers: [
                SliverAppBar(
                  leading: AppShellDrawerScope.maybeOf(context) == null
                      ? null
                      : const AppShellDrawerButton(),
                  title: Text(l10n.feedTitle),
                  pinned: true,
                ),
                switch (timelineAsync) {
                  AsyncValue(:final value?) => _FeedLoadedSlivers(
                    items: value.items,
                    hasMore: value.hasMore,
                    isLoadingMore: timelineAsync.isLoading,
                    hasLoadMoreError: timelineAsync.hasError,
                  ),
                  _ when timelineAsync.hasError => _FeedErrorSliver(
                    onRetry: () => ref.invalidate(timelineProvider),
                  ),
                  _ => const SliverFillRemaining(
                    hasScrollBody: false,
                    child: Center(child: StitchProgressIndicator()),
                  ),
                },
              ],
            ),
          ),
          Positioned(
            left: 16,
            bottom: isCompact && Directionality.of(context) == TextDirection.rtl
                ? 88
                : 16,
            child: ScrollToTopButton(
              visible: hasItems && _isPastScrollThreshold,
              tooltip: l10n.scrollToTopAction,
              onPressed: _scrollToTop,
            ),
          ),
        ],
      ),
    );
  }

  void _handleScroll() {
    final isPastThreshold =
        _scrollController.hasClients &&
        _scrollController.offset >= _scrollToTopThreshold;
    if (isPastThreshold == _isPastScrollThreshold || !mounted) return;
    setState(() => _isPastScrollThreshold = isPastThreshold);
  }

  void _openComposer(BuildContext buttonContext) {
    unawaited(
      showTopLevelPostComposerChooser(
        buttonContext,
        position: craftskyContextMenuAnchorPosition(buttonContext),
      ),
    );
  }

  void _scrollToTop() {
    if (!_scrollController.hasClients) return;
    if (MediaQuery.disableAnimationsOf(context)) {
      _scrollController.jumpTo(_scrollController.position.minScrollExtent);
      return;
    }
    final durations = Theme.of(context).extension<DurationTheme>()!;
    unawaited(
      _scrollController.animateTo(
        _scrollController.position.minScrollExtent,
        duration: durations.medium,
        curve: durations.ease,
      ),
    );
  }
}

class _FeedLoadedSlivers extends ConsumerWidget {
  const _FeedLoadedSlivers({
    required this.items,
    required this.hasMore,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
  });

  final List<TimelineItem> items;
  final bool hasMore;
  final bool isLoadingMore;
  final bool hasLoadMoreError;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final auth = ref.watch(authSessionProvider).value;
    return SliverMainAxisGroup(
      slivers: [
        if (items.isEmpty)
          SliverFillRemaining(
            hasScrollBody: false,
            child: CraftskyEmptyState(
              icon: CraftskyIcons.home,
              title: l10n.feedTitle,
              subtitle: l10n.feedEmpty,
              actionLabel: l10n.feedConnectInstagramAction,
              onAction: () =>
                  const InstagramMigrationRoute().push<void>(context),
            ),
          )
        else
          SliverList.builder(
            itemCount: items.length,
            itemBuilder: (context, index) {
              if (hasMore &&
                  !isLoadingMore &&
                  !hasLoadMoreError &&
                  index >= items.length - _autoLoadMoreThreshold) {
                WidgetsBinding.instance.addPostFrameCallback((_) {
                  if (context.mounted) {
                    unawaited(ref.read(timelineProvider.notifier).loadMore());
                  }
                });
              }
              final item = items[index];
              final post = item.post;
              return PostCard(
                post: post,
                collapseBody: true,
                imageInteractionMode: PostCardImageInteractionMode.navigate,
                hideWhenAuthorProtected: true,
                allowProfilePinAction: true,
                repostReason: item.reason,
                onTap: () => PostThreadRoute(
                  did: post.author.did,
                  rkey: post.rkey,
                ).push<void>(context),
                onLike: () => ref
                    .read(toggleLikePostProvider.notifier)
                    .toggle(post: post),
                onRepost: () => ref
                    .read(toggleRepostPostProvider.notifier)
                    .toggle(post: post),
                onQuote: () => unawaited(
                  showPostComposerSheet(context, quoteTarget: post),
                ),
                onReply: () => _replyAndOpenThread(context, ref, post),
                onDelete: auth is SignedIn && post.author.did == auth.did
                    ? () => _confirmDelete(context, ref, post)
                    : null,
                onReport: auth is SignedIn && post.author.did != auth.did
                    ? () => showPostReportSheet(context, ref, post)
                    : null,
                replyTooltip: l10n.postCommentAction,
              );
            },
          ),
        if (items.isNotEmpty && (isLoadingMore || hasLoadMoreError))
          SliverToBoxAdapter(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Center(
                child: switch ((isLoadingMore, hasLoadMoreError)) {
                  (true, _) => const StitchProgressIndicator(),
                  (_, true) => TextButton.icon(
                    onPressed: () =>
                        ref.read(timelineProvider.notifier).loadMore(),
                    icon: const Icon(CraftskyIconsBold.refresh),
                    label: Text(l10n.retryButton),
                  ),
                  _ => const SizedBox.shrink(),
                },
              ),
            ),
          ),
      ],
    );
  }

  Future<void> _replyAndOpenThread(
    BuildContext context,
    WidgetRef ref,
    Post post,
  ) async {
    final created = await showPostComposerSheet(context, replyTarget: post);
    if (created == null || !context.mounted) return;
    ref
        .read(timelineProvider.notifier)
        .replace(
          post.copyWith(
            replyCount: post.replyCount + 1,
            viewerHasReplied: true,
          ),
        );
    await PostThreadRoute(
      did: post.author.did,
      rkey: post.rkey,
      focus: created.uri,
      $extra: created,
    ).push<void>(context);
  }

  Future<void> _confirmDelete(
    BuildContext context,
    WidgetRef ref,
    Post post,
  ) async {
    final l10n = AppLocalizations.of(context);
    await showCraftskyDestructiveConfirmDialog(
      context,
      title: l10n.postDeleteTitle,
      message: l10n.postDeleteMessage,
      confirmLabel: l10n.postDeleteConfirm,
      onConfirm: () => ref.read(deletePostProvider.notifier).delete(post: post),
    );
  }
}

class _FeedErrorSliver extends StatelessWidget {
  const _FeedErrorSliver({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    return SliverFillRemaining(
      hasScrollBody: false,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(CraftskyIcons.error, color: theme.colorScheme.error),
            const SizedBox(height: 12),
            Text(l10n.feedLoadError, style: theme.textTheme.titleMedium),
            const SizedBox(height: 12),
            TextButton.icon(
              onPressed: onRetry,
              icon: const Icon(CraftskyIconsBold.refresh),
              label: Text(l10n.retryButton),
            ),
          ],
        ),
      ),
    );
  }
}
