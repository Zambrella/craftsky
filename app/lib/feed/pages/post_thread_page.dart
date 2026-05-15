import 'package:craftsky_app/feed/models/post.dart' as craftsky_post;
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart'
    hide PostCommentSection;
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class PostThreadPage extends ConsumerStatefulWidget {
  const PostThreadPage({
    required this.did,
    required this.rkey,
    this.focus,
    super.key,
  });

  final String did;
  final String rkey;
  final String? focus;

  @override
  ConsumerState<PostThreadPage> createState() => _PostThreadPageState();
}

class _PostThreadPageState extends ConsumerState<PostThreadPage> {
  CommentSort _sort = CommentSort.oldest;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final sectionAsync = ref.watch(
      postCommentSectionProvider(
        widget.did,
        widget.rkey,
        sort: _sort,
        focus: widget.focus,
      ),
    );
    final pageLoader = postCommentPageLoaderProvider(
      widget.did,
      widget.rkey,
      sort: _sort,
      focus: widget.focus,
    );
    final pageLoaderAsync = ref.watch(pageLoader);
    final formFactor = FormFactorWidget.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.postThreadTitle)),
      body: switch (sectionAsync) {
        AsyncData(:final value) => _CommentSectionBody(
          section: value,
          did: widget.did,
          rkey: widget.rkey,
          focus: widget.focus,
          showInlineComposer: formFactor.isLarge,
          isLoadingMoreComments: pageLoaderAsync.isLoading,
          onNearEnd: () => ref.read(pageLoader.notifier).load(),
          onCollapseReplies: (commentUri) => ref
              .read(
                postCommentSectionProvider(
                  widget.did,
                  widget.rkey,
                  sort: _sort,
                  focus: widget.focus,
                ).notifier,
              )
              .collapseReplies(commentUri),
          selectedSort: _sort,
          onSortChanged: (sort) => setState(() => _sort = sort),
        ),
        AsyncError() => _ThreadError(
          onRetry: () => ref.invalidate(
            postCommentSectionProvider(
              widget.did,
              widget.rkey,
              sort: _sort,
              focus: widget.focus,
            ),
          ),
        ),
        _ => const Center(child: StitchProgressIndicator()),
      },
      bottomNavigationBar: switch (sectionAsync) {
        AsyncData(:final value) when formFactor.isSmall => _ReplyPrompt(
          key: const ValueKey('threadStickyReplyPrompt'),
          post: value.post,
        ),
        _ => null,
      },
    );
  }
}

class _CommentSectionBody extends StatefulWidget {
  const _CommentSectionBody({
    required this.section,
    required this.did,
    required this.rkey,
    required this.focus,
    required this.showInlineComposer,
    required this.isLoadingMoreComments,
    required this.onNearEnd,
    required this.onCollapseReplies,
    required this.selectedSort,
    required this.onSortChanged,
  });

  final PostCommentSection section;
  final String did;
  final String rkey;
  final String? focus;
  final bool showInlineComposer;
  final bool isLoadingMoreComments;
  final VoidCallback onNearEnd;
  final void Function(String commentUri) onCollapseReplies;
  final CommentSort selectedSort;
  final ValueChanged<CommentSort> onSortChanged;

  @override
  State<_CommentSectionBody> createState() => _CommentSectionBodyState();
}

class _CommentSectionBodyState extends State<_CommentSectionBody> {
  final _controller = ScrollController();

  @override
  void initState() {
    super.initState();
    _controller.addListener(_handleScroll);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _handleScroll() {
    if (!_controller.hasClients) return;
    if (_controller.position.extentAfter < 240) widget.onNearEnd();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return CustomScrollView(
      controller: _controller,
      slivers: [
        SliverToBoxAdapter(
          child: Padding(
            padding: EdgeInsets.symmetric(vertical: spacing.sp2),
            child: PostCard(
              post: widget.section.post,
              onReply: () => showPostComposerSheet(
                context,
                replyTarget: widget.section.post,
              ),
            ),
          ),
        ),
        SliverPadding(
          padding: EdgeInsets.only(bottom: spacing.sp5),
          sliver: SliverList.list(
            children: [
              Padding(
                padding: EdgeInsets.symmetric(horizontal: spacing.sp4),
                child: Align(
                  alignment: Alignment.centerRight,
                  child: DropdownButton<CommentSort>(
                    value: widget.selectedSort,
                    onChanged: (sort) {
                      if (sort != null) widget.onSortChanged(sort);
                    },
                    items: [
                      DropdownMenuItem(
                        value: CommentSort.oldest,
                        child: Text(l10n.postCommentsSortOldest),
                      ),
                      DropdownMenuItem(
                        value: CommentSort.newest,
                        child: Text(l10n.postCommentsSortNewest),
                      ),
                      DropdownMenuItem(
                        value: CommentSort.follows,
                        child: Text(l10n.postCommentsSortFollows),
                      ),
                    ],
                  ),
                ),
              ),
              if (widget.showInlineComposer)
                _ReplyPrompt(
                  key: const ValueKey('threadInlineReplyPrompt'),
                  post: widget.section.post,
                ),
              if (widget.section.comments.items.isEmpty)
                Padding(
                  padding: EdgeInsets.all(spacing.sp5),
                  child: Center(child: Text(l10n.postThreadEmptyReplies)),
                )
              else
                for (final comment in widget.section.comments.items)
                  _CommentCard(
                    item: comment,
                    did: widget.did,
                    rkey: widget.rkey,
                    sort: widget.selectedSort,
                    focus: widget.focus,
                    focusedUri: widget.section.focus?.uri,
                    onCollapseReplies: widget.onCollapseReplies,
                  ),
              if (widget.isLoadingMoreComments)
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 16),
                  child: Center(child: StitchProgressIndicator()),
                ),
            ],
          ),
        ),
      ],
    );
  }
}

class _CommentCard extends ConsumerWidget {
  const _CommentCard({
    required this.item,
    required this.did,
    required this.rkey,
    required this.sort,
    required this.focus,
    required this.focusedUri,
    required this.onCollapseReplies,
  });

  final CommentItem item;
  final String did;
  final String rkey;
  final CommentSort sort;
  final String? focus;
  final String? focusedUri;
  final void Function(String commentUri) onCollapseReplies;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final focusedComment = focusedUri == item.post.uri;
    final repliesLoader = postCommentRepliesLoaderProvider(
      did,
      rkey,
      commentUri: item.post.uri,
      sort: sort,
      focus: focus,
    );
    final repliesLoaderAsync = ref.watch(repliesLoader);
    return Column(
      key: focusedComment ? const ValueKey('focused-comment-target') : null,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        PostCard(
          post: item.post,
          onReply: () => showPostComposerSheet(context, replyTarget: item.post),
        ),
        if (item.replies.loaded)
          Padding(
            padding: EdgeInsetsDirectional.only(start: spacing.sp6),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                for (final reply in item.replies.items)
                  PostCard(
                    key: focusedUri == reply.post.uri
                        ? const ValueKey('focused-comment-target')
                        : null,
                    post: reply.post,
                    onReply: () => showPostComposerSheet(
                      context,
                      replyTarget: reply.post,
                    ),
                  ),
              ],
            ),
          ),
        Padding(
          padding: EdgeInsetsDirectional.only(
            start: spacing.sp8 + spacing.sp2,
            end: spacing.sp4,
            bottom: spacing.sp2,
          ),
          child: Align(
            alignment: Alignment.centerLeft,
            child: switch (item.replies.loaded) {
              false when item.post.replyCount > 0 => TextButton(
                onPressed: repliesLoaderAsync.isLoading
                    ? null
                    : () => ref.read(repliesLoader.notifier).load(),
                child: repliesLoaderAsync.isLoading
                    ? const SizedBox.square(
                        dimension: 20,
                        child: StitchProgressIndicator(),
                      )
                    : Text(l10n.postCommentsViewReplies),
              ),
              true => Wrap(
                spacing: spacing.sp2,
                children: [
                  TextButton(
                    onPressed: () => onCollapseReplies(item.post.uri),
                    child: Text(l10n.postCommentsHideReplies),
                  ),
                  if (item.replies.cursor != null)
                    TextButton(
                      onPressed: repliesLoaderAsync.isLoading
                          ? null
                          : () => ref.read(repliesLoader.notifier).load(),
                      child: repliesLoaderAsync.isLoading
                          ? const SizedBox.square(
                              dimension: 20,
                              child: StitchProgressIndicator(),
                            )
                          : Text(l10n.postCommentsLoadMoreReplies),
                    ),
                ],
              ),
              _ => const SizedBox.shrink(),
            },
          ),
        ),
      ],
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
