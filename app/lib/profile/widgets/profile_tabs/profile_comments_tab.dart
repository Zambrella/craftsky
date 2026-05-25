import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _autoLoadMoreThreshold = 3;

class ProfileCommentsTab extends ConsumerWidget {
  const ProfileCommentsTab({
    required this.handle,
    required this.isOwnProfile,
    super.key,
  });

  final String handle;
  final bool isOwnProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final commentsAsync = ref.watch(userCommentsProvider(handle));

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

    return switch (commentsAsync) {
      AsyncValue(:final value?) => _ProfileCommentsLoadedSlivers(
        handle: handle,
        comments: value.items,
        hasMore: value.hasMore,
        isLoadingMore: commentsAsync.isLoading,
        hasLoadMoreError: commentsAsync.hasError,
        isOwnProfile: isOwnProfile,
      ),
      AsyncError(:final error) => _ProfileCommentsErrorSliver(
        error: error,
        onRetry: () => ref.invalidate(userCommentsProvider(handle)),
      ),
      _ => const SliverFillRemaining(
        hasScrollBody: false,
        child: Center(child: StitchProgressIndicator()),
      ),
    };
  }
}

class _ProfileCommentsLoadedSlivers extends ConsumerWidget {
  const _ProfileCommentsLoadedSlivers({
    required this.handle,
    required this.comments,
    required this.hasMore,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
    required this.isOwnProfile,
  });

  final String handle;
  final List<Post> comments;
  final bool hasMore;
  final bool isLoadingMore;
  final bool hasLoadMoreError;
  final bool isOwnProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;

    return SliverMainAxisGroup(
      slivers: [
        if (comments.isEmpty)
          SliverFillRemaining(
            hasScrollBody: false,
            child: Center(child: Text(l10n.profileCommentsEmpty)),
          )
        else
          SliverList.builder(
            itemCount: comments.length,
            itemBuilder: (context, index) {
              if (hasMore &&
                  !isLoadingMore &&
                  !hasLoadMoreError &&
                  index >= comments.length - _autoLoadMoreThreshold) {
                WidgetsBinding.instance.addPostFrameCallback((_) {
                  if (context.mounted) {
                    unawaited(
                      ref
                          .read(userCommentsProvider(handle).notifier)
                          .loadMore(),
                    );
                  }
                });
              }
              final post = comments[index];
              final root = post.reply == null
                  ? null
                  : parseCraftskyPostUri(post.reply!.root.uri);
              final isReply =
                  post.reply != null &&
                  post.reply!.root.uri != post.reply!.parent.uri;
              return PostCard(
                post: post,
                style: PostCardStyle.flat,
                replyTooltip: l10n.postThreadReplyAction,
                showRepostAction: false,
                showReplyCount: false,
                showReplyLabel: true,
                deleteLabel: isReply
                    ? l10n.replyDeleteAction
                    : l10n.commentDeleteAction,
                onTap: root == null
                    ? null
                    : () => PostThreadRoute(
                        did: root.did,
                        rkey: root.rkey,
                        focus: post.uri,
                      ).push<void>(context),
                onReply: () => _replyAndMarkJoined(ref, context, post),
                onLike: () => ref
                    .read(toggleLikePostProvider.notifier)
                    .toggle(post: post),
                onDelete: isOwnProfile
                    ? () => _confirmDelete(context, ref, post, isReply: isReply)
                    : null,
              );
            },
          ),
        if (comments.isNotEmpty && (isLoadingMore || hasLoadMoreError))
          SliverToBoxAdapter(
            child: Padding(
              padding: EdgeInsets.all(spacing.sp4),
              child: Center(
                child: switch ((isLoadingMore, hasLoadMoreError)) {
                  (true, _) => const StitchProgressIndicator(),
                  (_, true) => TextButton.icon(
                    onPressed: () => ref
                        .read(userCommentsProvider(handle).notifier)
                        .loadMore(),
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

  Future<void> _confirmDelete(
    BuildContext context,
    WidgetRef ref,
    Post post, {
    required bool isReply,
  }) async {
    final l10n = AppLocalizations.of(context);
    await showCraftskyDestructiveConfirmDialog(
      context,
      title: isReply ? l10n.replyDeleteTitle : l10n.commentDeleteTitle,
      message: isReply ? l10n.replyDeleteMessage : l10n.commentDeleteMessage,
      confirmLabel: l10n.postDeleteConfirm,
      onConfirm: () => ref.read(deletePostProvider.notifier).delete(post: post),
    );
  }

  Future<void> _replyAndMarkJoined(
    WidgetRef ref,
    BuildContext context,
    Post post,
  ) async {
    final created = await showPostComposerSheet(context, replyTarget: post);
    if (created == null) return;
    ref
        .read(userCommentsProvider(handle).notifier)
        .replace(
          post.copyWith(
            replyCount: post.replyCount + 1,
            viewerHasReplied: true,
          ),
        );
  }
}

class _ProfileCommentsErrorSliver extends StatelessWidget {
  const _ProfileCommentsErrorSliver({
    required this.error,
    required this.onRetry,
  });

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return SliverFillRemaining(
      hasScrollBody: false,
      child: Center(
        child: Padding(
          padding: EdgeInsets.all(spacing.sp5),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.error_outline, color: theme.colorScheme.error),
              SizedBox(height: spacing.sp3),
              Text(
                l10n.profileCommentsLoadError,
                style: theme.textTheme.titleMedium,
              ),
              SizedBox(height: spacing.sp3),
              TextButton.icon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh),
                label: Text(l10n.retryButton),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
