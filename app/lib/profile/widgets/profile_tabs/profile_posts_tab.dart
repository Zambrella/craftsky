import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/feed/widgets/post_type_chooser.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/report_flow.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _autoLoadMoreThreshold = 3;

/// Posts tab body. Returns a [SliverList] so it slots into the page's
/// outer [CustomScrollView] without nesting another scrollable.
class ProfilePostsTab extends ConsumerWidget {
  const ProfilePostsTab({
    required this.handle,
    required this.isOwnProfile,
    super.key,
  });

  final String handle;
  final bool isOwnProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final postsAsync = ref.watch(userPostsProvider(handle));

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

    return switch (postsAsync) {
      AsyncValue(:final value?) => _ProfilePostsLoadedSlivers(
        handle: handle,
        posts: value.items,
        hasMore: value.hasMore,
        isLoadingMore: postsAsync.isLoading,
        hasLoadMoreError: postsAsync.hasError,
        isOwnProfile: isOwnProfile,
      ),
      AsyncError(:final error) => _ProfilePostsErrorSliver(
        error: error,
        onRetry: () => ref.invalidate(userPostsProvider(handle)),
      ),
      _ => const SliverFillRemaining(
        hasScrollBody: false,
        child: Center(child: StitchProgressIndicator()),
      ),
    };
  }
}

class _ProfilePostsLoadedSlivers extends ConsumerWidget {
  const _ProfilePostsLoadedSlivers({
    required this.handle,
    required this.posts,
    required this.hasMore,
    required this.isLoadingMore,
    required this.hasLoadMoreError,
    required this.isOwnProfile,
  });

  final String handle;
  final List<Post> posts;
  final bool hasMore;
  final bool isLoadingMore;
  final bool hasLoadMoreError;
  final bool isOwnProfile;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final viewerDid = switch (ref.watch(authSessionProvider).value) {
      SignedIn(:final did) => did,
      _ => null,
    };

    return SliverMainAxisGroup(
      slivers: [
        if (isOwnProfile)
          SliverToBoxAdapter(
            child: Padding(
              padding: EdgeInsets.all(spacing.sp4),
              child: Builder(
                builder: (buttonContext) {
                  return ChunkyButton(
                    onPressed: () {
                      unawaited(
                        showTopLevelPostComposerChooser(
                          buttonContext,
                          position: _contextMenuPosition(buttonContext),
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
            child: Center(child: Text(l10n.profilePostsEmpty)),
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
                    unawaited(
                      ref.read(userPostsProvider(handle).notifier).loadMore(),
                    );
                  }
                });
              }
              final post = posts[index];
              return PostCard(
                post: post,
                onTap: () => PostThreadRoute(
                  did: post.author.did,
                  rkey: post.rkey,
                ).push<void>(context),
                onReply: () => _replyAndOpenThread(context, ref, post),
                replyTooltip: l10n.postCommentAction,
                onLike: () => ref
                    .read(toggleLikePostProvider.notifier)
                    .toggle(post: post),
                onRepost: () => ref
                    .read(toggleRepostPostProvider.notifier)
                    .toggle(post: post),
                onDelete: isOwnProfile
                    ? () => _confirmDelete(context, ref, post)
                    : null,
                onReport: viewerDid != null && post.author.did != viewerDid
                    ? () => showPostReportSheet(context, ref, post)
                    : null,
              );
            },
          ),
        if (posts.isNotEmpty && (isLoadingMore || hasLoadMoreError))
          SliverToBoxAdapter(
            child: Padding(
              padding: EdgeInsets.all(spacing.sp4),
              child: Center(
                child: switch ((isLoadingMore, hasLoadMoreError)) {
                  (true, _) => const StitchProgressIndicator(),
                  (_, true) => TextButton.icon(
                    onPressed: () =>
                        ref.read(userPostsProvider(handle).notifier).loadMore(),
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

  RelativeRect _contextMenuPosition(BuildContext context) {
    final renderObject = context.findRenderObject();
    final overlayObject = Overlay.of(context).context.findRenderObject();
    if (renderObject is! RenderBox || overlayObject is! RenderBox) {
      return RelativeRect.fill;
    }
    final topLeft = renderObject.localToGlobal(
      Offset.zero,
      ancestor: overlayObject,
    );
    final bottomRight = renderObject.localToGlobal(
      renderObject.size.bottomRight(Offset.zero),
      ancestor: overlayObject,
    );
    return RelativeRect.fromRect(
      Rect.fromPoints(topLeft, bottomRight),
      Offset.zero & overlayObject.size,
    );
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

  Future<void> _replyAndOpenThread(
    BuildContext context,
    WidgetRef ref,
    Post post,
  ) async {
    final created = await showPostComposerSheet(context, replyTarget: post);
    if (created == null || !context.mounted) return;
    ref
        .read(userPostsProvider(handle).notifier)
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
}

class _ProfilePostsErrorSliver extends StatelessWidget {
  const _ProfilePostsErrorSliver({required this.error, required this.onRetry});

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
                l10n.profilePostsLoadError,
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
