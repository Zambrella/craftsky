import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/report_flow.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/widgets/auto_paginated_list_view.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class PostQuotesPage extends ConsumerWidget {
  const PostQuotesPage({required this.did, required this.rkey, super.key});

  final Did did;
  final RecordKey rkey;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final provider = postQuotesProvider(did, rkey);
    final quotes = ref.watch(provider);
    final data = quotes.value;
    final auth = ref.watch(authSessionProvider).value;
    final l10n = AppLocalizations.of(context);

    return Scaffold(
      appBar: AppBar(title: Text(l10n.postInteractionQuotesTitle)),
      body: switch ((data, quotes.error)) {
        (null, null) => CraftskySkeletonList(
          itemCount: 3,
          itemBuilder: (context, index) => PostCardSkeleton(
            showMedia: index == 0,
          ),
        ),
        (null, final error?) => _InitialError(
          error: error,
          onRetry: () => unawaited(ref.read(provider.notifier).refresh()),
        ),
        (final state?, _) => AutoPaginatedListView(
          itemCount: state.items.length,
          emptyState: Semantics(
            label: l10n.postInteractionQuotesEmpty,
            liveRegion: true,
            container: true,
            excludeSemantics: true,
            child: CraftskyEmptyState(
              icon: CraftskyIcons.quote,
              title: l10n.postInteractionQuotesTitle,
              subtitle: l10n.postInteractionQuotesEmpty,
            ),
          ),
          isLoadingMore: quotes.isLoading && state.hasMore,
          hasLoadMoreError: quotes.hasError && state.hasMore,
          onNearEnd: () => unawaited(ref.read(provider.notifier).loadMore()),
          itemBuilder: (_, index) {
            final post = state.items[index];
            final isOwner = auth is SignedIn && auth.did == post.author.did;
            return PostCard(
              post: post,
              onRevealPost:
                  post.availability == 'muted' &&
                      post.relationship?.revealable == true
                  ? () => unawaited(_reveal(context, ref, provider, post))
                  : null,
              collapseBody: true,
              imageInteractionMode: PostCardImageInteractionMode.navigate,
              hideWhenAuthorProtected: true,
              onTap: () => PostThreadRoute(
                did: post.author.did,
                rkey: post.rkey,
              ).push<void>(context),
              onReply: () => unawaited(_reply(context, ref, provider, post)),
              replyTooltip: l10n.postCommentAction,
              onLike: () =>
                  unawaited(_toggleLike(context, ref, provider, post)),
              onRepost: () =>
                  unawaited(_toggleRepost(context, ref, provider, post)),
              onQuote: () => unawaited(
                showPostComposerSheet(context, quoteTarget: post),
              ),
              onDelete: isOwner
                  ? () =>
                        unawaited(_confirmDelete(context, ref, provider, post))
                  : null,
              onReport: auth is SignedIn && !isOwner
                  ? () => showPostReportSheet(context, ref, post)
                  : null,
            );
          },
        ),
      },
    );
  }

  Future<void> _toggleLike(
    BuildContext context,
    WidgetRef ref,
    PostQuotesProvider provider,
    Post post,
  ) async {
    await ref.read(toggleLikePostProvider.notifier).toggle(post: post);
    if (!context.mounted) return;
    final result = ref.read(toggleLikePostProvider);
    if (result.hasError) {
      context.showError(AppLocalizations.of(context).postLikeError);
      ref.read(toggleLikePostProvider.notifier).reset();
      return;
    }
    if (result.value case final updated?) {
      ref.read(provider.notifier).replace(updated);
    }
  }

  Future<void> _reveal(
    BuildContext context,
    WidgetRef ref,
    PostQuotesProvider provider,
    Post post,
  ) async {
    try {
      final revealed = await ref
          .read(postRepositoryProvider)
          .fetch(post.author.did, post.rkey);
      ref.read(provider.notifier).replace(revealed);
    } on Object {
      if (context.mounted) {
        context.showError(AppLocalizations.of(context).postRevealError);
      }
    }
  }

  Future<void> _toggleRepost(
    BuildContext context,
    WidgetRef ref,
    PostQuotesProvider provider,
    Post post,
  ) async {
    final notifier = ref.read(toggleRepostPostProvider.notifier);
    try {
      await notifier.toggle(post: post);
      final result = ref.read(toggleRepostPostProvider);
      if (result.hasError) {
        if (context.mounted) {
          context.showError(AppLocalizations.of(context).postRepostError);
        }
        return;
      }
      if (result.value case final updated?) {
        ref.read(provider.notifier).replace(updated);
      }
    } finally {
      notifier.reset();
    }
  }

  Future<void> _reply(
    BuildContext context,
    WidgetRef ref,
    PostQuotesProvider provider,
    Post post,
  ) async {
    final created = await showPostComposerSheet(context, replyTarget: post);
    if (created == null || !context.mounted) return;
    ref
        .read(provider.notifier)
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
    PostQuotesProvider provider,
    Post post,
  ) async {
    final l10n = AppLocalizations.of(context);
    await showCraftskyDestructiveConfirmDialog(
      context,
      title: l10n.postDeleteTitle,
      message: l10n.postDeleteMessage,
      confirmLabel: l10n.postDeleteConfirm,
      onConfirm: () async {
        await ref.read(deletePostProvider.notifier).delete(post: post);
        final result = ref.read(deletePostProvider);
        if (result.value?.uri == post.uri) {
          ref.read(provider.notifier).remove(post.uri);
          if (context.mounted) context.showInfo(l10n.postDeleteSuccess);
        } else if (result.hasError && context.mounted) {
          context.showError(l10n.postDeleteError);
        }
        ref.read(deletePostProvider.notifier).reset();
      },
    );
  }
}

class _InitialError extends StatelessWidget {
  const _InitialError({required this.error, required this.onRetry});

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final unavailable = switch (error) {
      ApiBadRequest(:final code, :final details) =>
        code == 'post_not_found' && details.statusCode == 404,
      _ => false,
    };
    return Center(
      child: Padding(
        padding: EdgeInsets.all(spacing.sp5),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Semantics(
              liveRegion: true,
              child: Text(
                unavailable
                    ? l10n.postUnavailablePlaceholder
                    : l10n.errorBackgroundLoadFailed,
                style: Theme.of(context).textTheme.titleLarge,
                textAlign: TextAlign.center,
              ),
            ),
            SizedBox(height: spacing.sp3),
            if (unavailable)
              TextButton(
                onPressed: () {
                  final navigator = Navigator.of(context);
                  if (navigator.canPop()) {
                    navigator.pop();
                  } else {
                    const FeedRoute().go(context);
                  }
                },
                child: Text(l10n.backButton),
              )
            else
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
