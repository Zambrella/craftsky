import 'package:craftsky_app/feed/models/post.dart' as craftsky_post;
import 'package:craftsky_app/feed/models/post_thread.dart';
import 'package:craftsky_app/feed/providers/post_thread_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class PostThreadPage extends ConsumerWidget {
  const PostThreadPage({required this.did, required this.rkey, super.key});

  final String did;
  final String rkey;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final threadAsync = ref.watch(postThreadProvider(did, rkey));
    final formFactor = FormFactorWidget.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.postThreadTitle)),
      body: switch (threadAsync) {
        AsyncData(:final value) => _ThreadBody(
          thread: value,
          showInlineComposer: formFactor.isLarge,
        ),
        AsyncError() => _ThreadError(
          onRetry: () => ref.invalidate(postThreadProvider(did, rkey)),
        ),
        _ => const Center(child: StitchProgressIndicator()),
      },
      bottomNavigationBar: switch (threadAsync) {
        AsyncData(:final value) when formFactor.isSmall => _ReplyPrompt(
          key: const ValueKey('threadStickyReplyPrompt'),
          post: value.post,
        ),
        _ => null,
      },
    );
  }
}

class _ThreadError extends StatelessWidget {
  const _ThreadError({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Center(
      child: TextButton.icon(
        onPressed: onRetry,
        icon: const Icon(Icons.refresh),
        label: Text(l10n.retryButton),
      ),
    );
  }
}

class _ThreadBody extends StatelessWidget {
  const _ThreadBody({required this.thread, required this.showInlineComposer});

  final PostThread thread;
  final bool showInlineComposer;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final anchorKey = ValueKey('selectedThreadPost-${thread.post.uri}');
    return CustomScrollView(
      center: anchorKey,
      slivers: [
        SliverList.list(
          children: [
            for (final ancestor in thread.ancestors.reversed)
              _ThreadPostCard(
                thread: PostThread(
                  post: ancestor,
                  replies: const [],
                ),
                isAncestor: true,
              ),
          ],
        ),
        SliverToBoxAdapter(
          key: anchorKey,
          child: Padding(
            padding: EdgeInsets.symmetric(vertical: spacing.sp2),
            child: _ThreadPostCard(thread: thread, isAnchor: true),
          ),
        ),
        SliverPadding(
          padding: EdgeInsets.only(bottom: spacing.sp5),
          sliver: SliverList.list(
            children: [
              if (showInlineComposer)
                _ReplyPrompt(
                  key: const ValueKey('threadInlineReplyPrompt'),
                  post: thread.post,
                ),
              if (thread.replies.isEmpty)
                Padding(
                  padding: EdgeInsets.all(spacing.sp5),
                  child: Center(child: Text(l10n.postThreadEmptyReplies)),
                )
              else
                for (final reply in thread.replies)
                  _ThreadPostCard(thread: reply),
              if (thread.truncated)
                Padding(
                  padding: EdgeInsets.all(spacing.sp4),
                  child: Center(child: Text(l10n.postThreadReadMoreReplies)),
                ),
            ],
          ),
        ),
      ],
    );
  }
}

class _ThreadPostCard extends StatelessWidget {
  const _ThreadPostCard({
    required this.thread,
    this.isAnchor = false,
    this.isAncestor = false,
  });

  final PostThread thread;
  final bool isAnchor;
  final bool isAncestor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final hasContinuation =
        thread.replies.isNotEmpty || thread.post.replyCount > 0;
    final author = _threadAuthorLabel(thread.post);
    final continuationLabel = thread.replies.length > 1
        ? l10n.postThreadShowMoreReplies
        : l10n.postThreadContinueThread;
    final continuationSemanticsLabel = thread.replies.length > 1
        ? l10n.postThreadShowMoreRepliesForAuthor(author)
        : l10n.postThreadContinueThreadFromAuthor(author);
    return DecoratedBox(
      decoration: BoxDecoration(
        border: Border(
          left: BorderSide(
            width: isAnchor ? spacing.sp1 : spacing.sp1 / 2,
            color: isAnchor ? theme.colorScheme.primary : swatches.borderHair,
          ),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          PostCard(
            post: thread.post,
            onTap: isAnchor
                ? null
                : () => PostThreadRoute(
                    did: thread.post.author.did,
                    rkey: thread.post.rkey,
                  ).push<void>(context),
            onReply: () => showPostComposerSheet(
              context,
              replyTarget: thread.post,
            ),
            replyTooltip: l10n.postThreadReplyToAuthor(author),
          ),
          if (!isAncestor && !isAnchor && hasContinuation)
            Padding(
              padding: EdgeInsetsDirectional.only(
                start: spacing.sp8 + spacing.sp2,
                end: spacing.sp4,
                bottom: spacing.sp2,
              ),
              child: Align(
                alignment: Alignment.centerLeft,
                child: TextButton(
                  onPressed: () => PostThreadRoute(
                    did: thread.post.author.did,
                    rkey: thread.post.rkey,
                  ).push<void>(context),
                  child: Text(
                    continuationLabel,
                    semanticsLabel: continuationSemanticsLabel,
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _ReplyPrompt extends StatelessWidget {
  const _ReplyPrompt({required this.post, super.key});

  final craftsky_post.Post post;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return SafeArea(
      child: Padding(
        padding: EdgeInsets.all(spacing.sp4),
        child: FilledButton.icon(
          onPressed: () => showPostComposerSheet(context, replyTarget: post),
          icon: const Icon(Icons.chat_bubble_outline),
          label: Text(
            l10n.postThreadReplyAction,
            semanticsLabel: l10n.postThreadReplyToAuthor(
              _threadAuthorLabel(post),
            ),
          ),
        ),
      ),
    );
  }
}

String _threadAuthorLabel(craftsky_post.Post post) {
  final displayName = post.author.displayName;
  if (displayName != null && displayName.trim().isNotEmpty) {
    return '$displayName (@${post.author.handle})';
  }
  return '@${post.author.handle}';
}
