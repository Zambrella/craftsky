import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/feed/widgets/post_type_chooser.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/report_flow.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _autoLoadMoreThreshold = 3;

class FeedPage extends ConsumerWidget {
  const FeedPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final timelineAsync = ref.watch(timelineProvider);
    ref.listen(deletePostProvider, (previous, next) {
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
    });
    ref.listen(toggleLikePostProvider, (previous, next) {
      if (next.hasError) {
        context.showError(l10n.postLikeError);
        ref.read(toggleLikePostProvider.notifier).reset();
      }
    });
    return Scaffold(
      appBar: AppBar(title: Text(l10n.feedTitle)),
      body: CustomScrollView(
        slivers: [
          switch (timelineAsync) {
            AsyncValue(:final value?) => _FeedLoadedSlivers(
              posts: value.items,
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
    );
  }
}

class _FeedLoadedSlivers extends ConsumerWidget {
  const _FeedLoadedSlivers({
    required this.posts,
    required this.hasMore,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
  });

  final List<Post> posts;
  final bool hasMore;
  final bool isLoadingMore;
  final bool hasLoadMoreError;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final auth = ref.watch(authSessionProvider).value;
    return SliverMainAxisGroup(
      slivers: [
        SliverToBoxAdapter(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Builder(
              builder: (buttonContext) {
                return ChunkyButton(
                  onPressed: () {
                    unawaited(
                      showTopLevelPostComposerChooser(
                        buttonContext,
                        position: craftskyContextMenuAnchorPosition(
                          buttonContext,
                        ),
                      ),
                    );
                  },
                  child: Text(l10n.postComposeAction),
                );
              },
            ),
          ),
        ),
        if (posts.isEmpty)
          SliverFillRemaining(
            hasScrollBody: false,
            child: Center(child: Text(l10n.feedEmpty)),
          )
        else
          SliverList.builder(
            itemCount: posts.length,
            itemBuilder: (context, index) {
              if (hasMore &&
                  !isLoadingMore &&
                  !hasLoadMoreError &&
                  index >= posts.length - _autoLoadMoreThreshold) {
                WidgetsBinding.instance.addPostFrameCallback((_) {
                  if (context.mounted) {
                    unawaited(ref.read(timelineProvider.notifier).loadMore());
                  }
                });
              }
              return PostCard(
                post: posts[index],
                onTap: () => PostThreadRoute(
                  did: posts[index].author.did,
                  rkey: posts[index].rkey,
                ).push<void>(context),
                onLike: () => ref
                    .read(toggleLikePostProvider.notifier)
                    .toggle(post: posts[index]),
                onRepost: () => ref
                    .read(toggleRepostPostProvider.notifier)
                    .toggle(post: posts[index]),
                onReply: () => _replyAndOpenThread(context, ref, posts[index]),
                onDelete:
                    auth is SignedIn && posts[index].author.did == auth.did
                    ? () => _confirmDelete(context, ref, posts[index])
                    : null,
                onReport:
                    auth is SignedIn && posts[index].author.did != auth.did
                    ? () => showPostReportSheet(context, ref, posts[index])
                    : null,
                replyTooltip: l10n.postCommentAction,
              );
            },
          ),
        if (posts.isNotEmpty && (isLoadingMore || hasLoadMoreError))
          SliverToBoxAdapter(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Center(
                child: switch ((isLoadingMore, hasLoadMoreError)) {
                  (true, _) => const StitchProgressIndicator(),
                  (_, true) => TextButton.icon(
                    onPressed: () =>
                        ref.read(timelineProvider.notifier).loadMore(),
                    icon: const Icon(Icons.refresh),
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
            Icon(Icons.error_outline, color: theme.colorScheme.error),
            const SizedBox(height: 12),
            Text(l10n.feedLoadError, style: theme.textTheme.titleMedium),
            const SizedBox(height: 12),
            TextButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh),
              label: Text(l10n.retryButton),
            ),
          ],
        ),
      ),
    );
  }
}
