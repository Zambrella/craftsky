import 'dart:async';

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
  PostCommentSection? _lastSection;

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
    final visibleSection = sectionAsync.value ?? _lastSection;
    if (sectionAsync case AsyncData(:final value)) {
      _lastSection = value;
    }
    final isRefreshingComments =
        sectionAsync.isLoading &&
        sectionAsync.value == null &&
        visibleSection != null;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.postThreadTitle)),
      body: switch ((sectionAsync, visibleSection)) {
        (_, final section?) => _CommentSectionBody(
          section: section,
          did: widget.did,
          rkey: widget.rkey,
          focus: widget.focus,
          showInlineComposer: formFactor.isLarge,
          isLoadingMoreComments: pageLoaderAsync.isLoading,
          isRefreshingComments: isRefreshingComments,
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
        (AsyncError(), _) => _ThreadError(
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
      bottomNavigationBar: switch (visibleSection) {
        final value? when formFactor.isSmall => _ReplyPrompt(
          key: const ValueKey('threadStickyReplyPrompt'),
          post: value.post,
          isRootPrompt: true,
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
    required this.isRefreshingComments,
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
  final bool isRefreshingComments;
  final VoidCallback onNearEnd;
  final void Function(String commentUri) onCollapseReplies;
  final CommentSort selectedSort;
  final ValueChanged<CommentSort> onSortChanged;

  @override
  State<_CommentSectionBody> createState() => _CommentSectionBodyState();
}

class _CommentSectionBodyState extends State<_CommentSectionBody> {
  final _controller = ScrollController();
  final GlobalKey _focusedTargetKey = GlobalKey(
    debugLabel: 'focusedCommentTarget',
  );
  String? _scrolledFocusUri;
  String? _highlightedFocusUri;
  bool _focusRevealScheduled = false;
  Timer? _clearHighlightTimer;

  @override
  void initState() {
    super.initState();
    _controller.addListener(_handleScroll);
  }

  @override
  void dispose() {
    _clearHighlightTimer?.cancel();
    _controller.dispose();
    super.dispose();
  }

  void _handleScroll() {
    if (!_controller.hasClients) return;
    if (_controller.position.extentAfter < 240) widget.onNearEnd();
  }

  @override
  void didUpdateWidget(covariant _CommentSectionBody oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.section.focus?.uri != widget.section.focus?.uri) {
      _scrolledFocusUri = null;
      _highlightedFocusUri = null;
      _clearHighlightTimer?.cancel();
    }
  }

  void _scheduleFocusedReveal() {
    final focus = widget.section.focus;
    if (focus == null || focus.status != FocusStatus.included) return;
    if (_scrolledFocusUri == focus.uri) return;
    if (_focusRevealScheduled) return;
    _focusRevealScheduled = true;

    WidgetsBinding.instance.addPostFrameCallback((_) async {
      if (!mounted || _scrolledFocusUri == focus.uri) {
        _focusRevealScheduled = false;
        return;
      }
      final context = _focusedTargetKey.currentContext;
      if (context == null) {
        _focusRevealScheduled = false;
        setState(() {});
        return;
      }
      setState(() => _highlightedFocusUri = focus.uri);
      if (!_controller.hasClients) {
        _focusRevealScheduled = false;
        setState(() {});
        return;
      }
      if (_controller.position.maxScrollExtent <=
          _controller.position.minScrollExtent) {
        _scrolledFocusUri = focus.uri;
        _focusRevealScheduled = false;
        _scheduleClearHighlight(focus.uri);
        return;
      }
      final target = context.findRenderObject();
      final viewportContext = _controller.position.context.notificationContext;
      final viewport = viewportContext?.findRenderObject();
      if (target is RenderBox && viewport is RenderBox) {
        final targetTop = target
            .localToGlobal(Offset.zero, ancestor: viewport)
            .dy;
        final desiredOffset =
            (_controller.offset + targetTop - (viewport.size.height * 0.25))
                .clamp(
                  _controller.position.minScrollExtent,
                  _controller.position.maxScrollExtent,
                );
        _controller.jumpTo(desiredOffset);
      }
      if (!mounted) {
        _focusRevealScheduled = false;
        return;
      }
      _scrolledFocusUri = focus.uri;
      _focusRevealScheduled = false;
      _scheduleClearHighlight(focus.uri);
    });
  }

  void _scheduleClearHighlight(String focusUri) {
    _clearHighlightTimer?.cancel();
    _clearHighlightTimer = Timer(const Duration(seconds: 3), () {
      if (!mounted || _highlightedFocusUri != focusUri) return;
      setState(() => _highlightedFocusUri = null);
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    _scheduleFocusedReveal();
    return CustomScrollView(
      controller: _controller,
      slivers: [
        SliverToBoxAdapter(
          child: Padding(
            padding: EdgeInsets.symmetric(vertical: spacing.sp2),
            child: PostCard(
              post: widget.section.post,
              replyTooltip: l10n.postCommentAction,
              onReply: () => showPostComposerSheet(
                context,
                replyTarget: widget.section.post,
              ),
            ),
          ),
        ),
        SliverPadding(
          padding: EdgeInsets.only(bottom: spacing.sp5),
          sliver: SliverToBoxAdapter(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
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
                    isRootPrompt: true,
                  ),
                if (widget.isRefreshingComments)
                  const Padding(
                    padding: EdgeInsets.symmetric(vertical: 32),
                    child: Center(child: StitchProgressIndicator()),
                  )
                else if (widget.section.comments.items.isEmpty)
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
                      highlightedUri: _highlightedFocusUri,
                      focusedTargetKey: _focusedTargetKey,
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
    required this.highlightedUri,
    required this.focusedTargetKey,
    required this.onCollapseReplies,
  });

  final CommentItem item;
  final String did;
  final String rkey;
  final CommentSort sort;
  final String? focus;
  final String? focusedUri;
  final String? highlightedUri;
  final GlobalKey focusedTargetKey;
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
    final column = Column(
      key: focusedComment ? const ValueKey('focused-comment-target') : null,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        PostCard(
          post: item.post,
          isHighlighted: highlightedUri == item.post.uri,
          replyTooltip: l10n.postThreadReplyAction,
          showRepostAction: false,
          onReply: () => showPostComposerSheet(context, replyTarget: item.post),
        ),
        if (item.replies.loaded)
          _CommentReplyControls(
            commentUri: item.post.uri,
            repliesCursor: null,
            isLoading: repliesLoaderAsync.isLoading,
            onCollapseReplies: onCollapseReplies,
            onLoadMore: () => ref.read(repliesLoader.notifier).load(),
          ),
        if (item.replies.loaded)
          Padding(
            padding: EdgeInsetsDirectional.only(start: spacing.sp6),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                for (final reply in item.replies.items)
                  _FocusedTargetWrapper(
                    focusTargetKey: focusedTargetKey,
                    isFocused: focusedUri == reply.post.uri,
                    child: PostCard(
                      key: focusedUri == reply.post.uri
                          ? const ValueKey('focused-comment-target')
                          : null,
                      post: reply.post,
                      isHighlighted: highlightedUri == reply.post.uri,
                      replyTooltip: l10n.postThreadReplyAction,
                      showRepostAction: false,
                      showReplyCount: false,
                      onReply: () => showPostComposerSheet(
                        context,
                        replyTarget: reply.post,
                      ),
                    ),
                  ),
              ],
            ),
          ),
        if (!item.replies.loaded && item.post.replyCount > 0)
          _CommentReplyControls(
            commentUri: item.post.uri,
            repliesCursor: null,
            isLoading: repliesLoaderAsync.isLoading,
            showHide: false,
            onCollapseReplies: onCollapseReplies,
            onLoadMore: () => ref.read(repliesLoader.notifier).load(),
          ),
        if (item.replies.loaded && item.replies.cursor != null)
          _CommentReplyControls(
            commentUri: item.post.uri,
            repliesCursor: item.replies.cursor,
            isLoading: repliesLoaderAsync.isLoading,
            showHide: false,
            onCollapseReplies: onCollapseReplies,
            onLoadMore: () => ref.read(repliesLoader.notifier).load(),
          ),
      ],
    );
    return _FocusedTargetWrapper(
      focusTargetKey: focusedTargetKey,
      isFocused: focusedComment,
      child: column,
    );
  }
}

class _FocusedTargetWrapper extends StatelessWidget {
  const _FocusedTargetWrapper({
    required this.focusTargetKey,
    required this.isFocused,
    required this.child,
  });

  final GlobalKey focusTargetKey;
  final bool isFocused;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    if (!isFocused) return child;
    return KeyedSubtree(key: focusTargetKey, child: child);
  }
}

class _CommentReplyControls extends StatelessWidget {
  const _CommentReplyControls({
    required this.commentUri,
    required this.repliesCursor,
    required this.isLoading,
    required this.onCollapseReplies,
    required this.onLoadMore,
    this.showHide = true,
  });

  final String commentUri;
  final String? repliesCursor;
  final bool isLoading;
  final bool showHide;
  final void Function(String commentUri) onCollapseReplies;
  final VoidCallback onLoadMore;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final showViewReplies = !showHide && repliesCursor == null;
    return Padding(
      padding: EdgeInsetsDirectional.only(
        start: spacing.sp8 + spacing.sp2,
        end: spacing.sp4,
        bottom: spacing.sp2,
      ),
      child: Align(
        alignment: Alignment.centerLeft,
        child: Wrap(
          spacing: spacing.sp2,
          children: [
            if (showHide)
              TextButton(
                onPressed: () => onCollapseReplies(commentUri),
                child: Text(l10n.postCommentsHideReplies),
              )
            else if (showViewReplies)
              TextButton(
                onPressed: isLoading ? null : onLoadMore,
                child: isLoading
                    ? const SizedBox.square(
                        dimension: 20,
                        child: StitchProgressIndicator(),
                      )
                    : Text(l10n.postCommentsViewReplies),
              ),
            if (repliesCursor != null)
              TextButton(
                onPressed: isLoading ? null : onLoadMore,
                child: isLoading
                    ? const SizedBox.square(
                        dimension: 20,
                        child: StitchProgressIndicator(),
                      )
                    : Text(l10n.postCommentsLoadMoreReplies),
              ),
          ],
        ),
      ),
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
  const _ReplyPrompt({
    required this.post,
    this.isRootPrompt = false,
    super.key,
  });

  final craftsky_post.Post post;
  final bool isRootPrompt;

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
            isRootPrompt ? l10n.postCommentAction : l10n.postThreadReplyAction,
            semanticsLabel: isRootPrompt
                ? l10n.postCommentOnAuthor(_threadAuthorLabel(post))
                : l10n.postThreadReplyToAuthor(_threadAuthorLabel(post)),
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
