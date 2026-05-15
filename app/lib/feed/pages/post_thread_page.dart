import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart' as craftsky_post;
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart'
    hide PostCommentSection;
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
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
  String? _createdTargetUri;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final viewerDid = switch (ref.watch(authSessionProvider).value) {
      SignedIn(:final did) => did,
      _ => null,
    };
    final sectionProvider = postCommentSectionProvider(
      widget.did,
      widget.rkey,
      sort: _sort,
      focus: widget.focus,
    );
    final sectionAsync = ref.watch(sectionProvider);
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
      ..listen(createPostProvider, (previous, next) {
        final post = next.value;
        if (post == null) return;
        unawaited(_handleCreatedThreadPost(post));
      })
      ..listen(toggleLikePostProvider, (previous, next) {
        final post = next.value;
        if (post == null) return;
        ref.read(sectionProvider.notifier).replacePost(post);
      })
      ..listen(toggleRepostPostProvider, (previous, next) {
        final post = next.value;
        if (post == null) return;
        ref.read(sectionProvider.notifier).replacePost(post);
      });
    return Scaffold(
      appBar: AppBar(title: Text(l10n.postThreadTitle)),
      body: switch ((sectionAsync, visibleSection)) {
        (_, final section?) => _CommentSectionBody(
          section: section,
          did: widget.did,
          rkey: widget.rkey,
          focus: widget.focus,
          createdTargetUri: _createdTargetUri,
          viewerDid: viewerDid,
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

  Future<void> _handleCreatedThreadPost(craftsky_post.Post post) async {
    final reply = post.reply;
    if (reply == null) return;

    final sectionProvider = postCommentSectionProvider(
      widget.did,
      widget.rkey,
      sort: _sort,
      focus: widget.focus,
    );
    final section = ref.read(sectionProvider).value;
    if (section == null || reply.root.uri != section.post.uri) return;

    final notifier = ref.read(sectionProvider.notifier);
    if (reply.parent.uri == reply.root.uri) {
      notifier.prependCreatedComment(post);
      if (mounted) setState(() => _createdTargetUri = post.uri);
      return;
    }

    final parentComment = section.comments.items
        .where((item) => item.post.uri == reply.parent.uri)
        .firstOrNull;
    if (parentComment == null || parentComment.replies.loaded) {
      notifier.insertCreatedReply(parentUri: reply.parent.uri, post: post);
      if (mounted) setState(() => _createdTargetUri = post.uri);
      return;
    }

    if (parentComment.post.replyCount == 0) {
      notifier.insertCreatedReply(parentUri: reply.parent.uri, post: post);
      if (mounted) setState(() => _createdTargetUri = post.uri);
      return;
    }

    try {
      final pages = <ReplyItem>[];
      String? cursor;
      do {
        final page = await ref
            .read(postRepositoryProvider)
            .listCommentBranchReplies(
              parentComment.post.author.did,
              parentComment.post.rkey,
              cursor: cursor,
              limit: 10,
            );
        pages.addAll(page.items);
        cursor = page.cursor;
      } while (cursor != null);
      if (!mounted) return;
      final replies = sortReplyItems(
        [
          for (final item in pages)
            if (item.post.uri != post.uri) item,
          ReplyItem(post: post, flattened: false),
        ],
        commentSort: _sort,
      );
      ref
          .read(sectionProvider.notifier)
          .setRepliesForComment(
            commentUri: parentComment.post.uri,
            replies: replies,
            incrementRootReplyCount: true,
          );
      if (mounted) setState(() => _createdTargetUri = post.uri);
    } on Object {
      if (!mounted) return;
      ref
          .read(sectionProvider.notifier)
          .insertCreatedReply(parentUri: reply.parent.uri, post: post);
      if (mounted) setState(() => _createdTargetUri = post.uri);
    }
  }
}

class _CommentSectionBody extends ConsumerStatefulWidget {
  const _CommentSectionBody({
    required this.section,
    required this.did,
    required this.rkey,
    required this.focus,
    required this.createdTargetUri,
    required this.viewerDid,
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
  final String? createdTargetUri;
  final String? viewerDid;
  final bool showInlineComposer;
  final bool isLoadingMoreComments;
  final bool isRefreshingComments;
  final VoidCallback onNearEnd;
  final void Function(String commentUri) onCollapseReplies;
  final CommentSort selectedSort;
  final ValueChanged<CommentSort> onSortChanged;

  @override
  ConsumerState<_CommentSectionBody> createState() =>
      _CommentSectionBodyState();
}

class _CommentSectionBodyState extends ConsumerState<_CommentSectionBody> {
  final _controller = ScrollController();
  final GlobalKey _focusedTargetKey = GlobalKey(
    debugLabel: 'focusedCommentTarget',
  );
  String? _scrolledTargetUri;
  String? _highlightedTargetUri;
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
    if (_targetUri(oldWidget) != _targetUri(widget)) {
      _scrolledTargetUri = null;
      _highlightedTargetUri = null;
      _clearHighlightTimer?.cancel();
    }
  }

  String? _targetUri(_CommentSectionBody widget) {
    if (widget.createdTargetUri != null) return widget.createdTargetUri;
    final focus = widget.section.focus;
    if (focus != null && focus.status == FocusStatus.included) {
      return focus.uri;
    }
    return null;
  }

  void _scheduleFocusedReveal() {
    final targetUri = _targetUri(widget);
    if (targetUri == null) return;
    if (_scrolledTargetUri == targetUri) return;
    if (_focusRevealScheduled) return;
    _focusRevealScheduled = true;

    WidgetsBinding.instance.addPostFrameCallback((_) async {
      if (!mounted || _scrolledTargetUri == targetUri) {
        _focusRevealScheduled = false;
        return;
      }
      final context = _focusedTargetKey.currentContext;
      if (context == null) {
        _focusRevealScheduled = false;
        setState(() {});
        return;
      }
      setState(() => _highlightedTargetUri = targetUri);
      if (!_controller.hasClients) {
        _focusRevealScheduled = false;
        setState(() {});
        return;
      }
      if (_controller.position.maxScrollExtent <=
          _controller.position.minScrollExtent) {
        _scrolledTargetUri = targetUri;
        _focusRevealScheduled = false;
        _scheduleClearHighlight(targetUri);
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
      _scrolledTargetUri = targetUri;
      _focusRevealScheduled = false;
      _scheduleClearHighlight(targetUri);
    });
  }

  void _scheduleClearHighlight(String targetUri) {
    _clearHighlightTimer?.cancel();
    _clearHighlightTimer = Timer(const Duration(seconds: 3), () {
      if (!mounted || _highlightedTargetUri != targetUri) return;
      setState(() => _highlightedTargetUri = null);
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
              onLike: () => ref
                  .read(toggleLikePostProvider.notifier)
                  .toggle(post: widget.section.post),
              onRepost: () => ref
                  .read(toggleRepostPostProvider.notifier)
                  .toggle(post: widget.section.post),
              onDelete: _deleteIfViewerOwned(widget.section.post),
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
                    child: _CommentSortButton(
                      selectedSort: widget.selectedSort,
                      onSortChanged: widget.onSortChanged,
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
                      targetUri: _targetUri(widget),
                      highlightedUri: _highlightedTargetUri,
                      focusedTargetKey: _focusedTargetKey,
                      viewerDid: widget.viewerDid,
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

  VoidCallback? _deleteIfViewerOwned(craftsky_post.Post post) {
    if (widget.viewerDid != post.author.did) return null;
    return () => _confirmDelete(context, post);
  }

  Future<void> _confirmDelete(
    BuildContext context,
    craftsky_post.Post post,
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

class _CommentCard extends ConsumerWidget {
  const _CommentCard({
    required this.item,
    required this.did,
    required this.rkey,
    required this.sort,
    required this.focus,
    required this.targetUri,
    required this.highlightedUri,
    required this.focusedTargetKey,
    required this.viewerDid,
    required this.onCollapseReplies,
  });

  final CommentItem item;
  final String did;
  final String rkey;
  final CommentSort sort;
  final String? focus;
  final String? targetUri;
  final String? highlightedUri;
  final GlobalKey focusedTargetKey;
  final String? viewerDid;
  final void Function(String commentUri) onCollapseReplies;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final targetComment = targetUri == item.post.uri;
    final repliesLoader = postCommentRepliesLoaderProvider(
      did,
      rkey,
      commentUri: item.post.uri,
      sort: sort,
      focus: focus,
    );
    final repliesLoaderAsync = ref.watch(repliesLoader);
    final column = Column(
      key: targetComment ? const ValueKey('focused-comment-target') : null,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        PostCard(
          post: item.post,
          style: PostCardStyle.flat,
          isHighlighted: highlightedUri == item.post.uri,
          replyTooltip: l10n.postThreadReplyAction,
          showRepostAction: false,
          showReplyCount: false,
          showReplyLabel: true,
          onReply: () => showPostComposerSheet(context, replyTarget: item.post),
          onLike: () =>
              ref.read(toggleLikePostProvider.notifier).toggle(post: item.post),
          deleteLabel: l10n.commentDeleteAction,
          onDelete: _deleteIfViewerOwned(
            context,
            ref,
            item.post,
            title: l10n.commentDeleteTitle,
            message: l10n.commentDeleteMessage,
          ),
        ),
        if (item.replies.loaded)
          _CommentReplyControls(
            commentUri: item.post.uri,
            repliesCursor: null,
            replyCount: item.post.replyCount,
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
                    isFocused: targetUri == reply.post.uri,
                    child: PostCard(
                      key: targetUri == reply.post.uri
                          ? const ValueKey('focused-comment-target')
                          : null,
                      post: reply.post,
                      style: PostCardStyle.flat,
                      isHighlighted: highlightedUri == reply.post.uri,
                      replyTooltip: l10n.postThreadReplyAction,
                      showRepostAction: false,
                      showReplyCount: false,
                      showReplyLabel: true,
                      onReply: () => showPostComposerSheet(
                        context,
                        replyTarget: reply.post,
                      ),
                      onLike: () => ref
                          .read(toggleLikePostProvider.notifier)
                          .toggle(post: reply.post),
                      deleteLabel: l10n.replyDeleteAction,
                      onDelete: _deleteIfViewerOwned(
                        context,
                        ref,
                        reply.post,
                        title: l10n.replyDeleteTitle,
                        message: l10n.replyDeleteMessage,
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
            replyCount: item.post.replyCount,
            isLoading: repliesLoaderAsync.isLoading,
            showHide: false,
            onCollapseReplies: onCollapseReplies,
            onLoadMore: () => ref.read(repliesLoader.notifier).load(),
          ),
        if (item.replies.loaded && item.replies.cursor != null)
          _CommentReplyControls(
            commentUri: item.post.uri,
            repliesCursor: item.replies.cursor,
            replyCount: item.post.replyCount,
            isLoading: repliesLoaderAsync.isLoading,
            showHide: false,
            onCollapseReplies: onCollapseReplies,
            onLoadMore: () => ref.read(repliesLoader.notifier).load(),
          ),
      ],
    );
    return _FocusedTargetWrapper(
      focusTargetKey: focusedTargetKey,
      isFocused: targetComment,
      child: column,
    );
  }

  VoidCallback? _deleteIfViewerOwned(
    BuildContext context,
    WidgetRef ref,
    craftsky_post.Post post, {
    required String title,
    required String message,
  }) {
    if (viewerDid != post.author.did) return null;
    return () => _confirmDelete(
      context,
      ref,
      post,
      title: title,
      message: message,
    );
  }

  Future<void> _confirmDelete(
    BuildContext context,
    WidgetRef ref,
    craftsky_post.Post post, {
    required String title,
    required String message,
  }) async {
    final l10n = AppLocalizations.of(context);
    await showCraftskyDestructiveConfirmDialog(
      context,
      title: title,
      message: message,
      confirmLabel: l10n.postDeleteConfirm,
      onConfirm: () => ref.read(deletePostProvider.notifier).delete(post: post),
    );
  }
}

class _CommentSortButton extends StatelessWidget {
  const _CommentSortButton({
    required this.selectedSort,
    required this.onSortChanged,
  });

  final CommentSort selectedSort;
  final ValueChanged<CommentSort> onSortChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final label = _sortLabel(l10n, selectedSort);
    return OutlinedButton.icon(
      onPressed: () => _showMenu(context),
      icon: const Icon(Icons.filter_list, size: 18),
      label: Text(label),
      style: OutlinedButton.styleFrom(
        foregroundColor: theme.colorScheme.onSurface,
        side: BorderSide(color: theme.colorScheme.onSurface, width: 1.5),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(999)),
        padding: EdgeInsets.symmetric(
          horizontal: spacing.sp3,
          vertical: spacing.sp2,
        ),
      ),
    );
  }

  void _showMenu(BuildContext context) {
    final button = context.findRenderObject()! as RenderBox;
    final overlay =
        Navigator.of(context).overlay!.context.findRenderObject()! as RenderBox;
    final offset = button.localToGlobal(Offset.zero, ancestor: overlay);
    final position = RelativeRect.fromRect(
      offset & button.size,
      Offset.zero & overlay.size,
    );
    final l10n = AppLocalizations.of(context);

    showCraftskyContextMenu(
      context,
      position: position,
      groups: [
        CraftskyContextMenuGroup(
          items: [
            _sortItem(
              sort: CommentSort.newest,
              text: l10n.postCommentsSortNewest,
              description: l10n.postCommentsSortNewestDescription,
            ),
            _sortItem(
              sort: CommentSort.oldest,
              text: l10n.postCommentsSortOldest,
              description: l10n.postCommentsSortOldestDescription,
            ),
            _sortItem(
              sort: CommentSort.follows,
              text: l10n.postCommentsSortFollows,
              description: l10n.postCommentsSortFollowsDescription,
            ),
          ],
        ),
      ],
    );
  }

  CraftskyContextMenuItem _sortItem({
    required CommentSort sort,
    required String text,
    required String description,
  }) {
    return CraftskyContextMenuItem(
      text: text,
      description: description,
      icon: Icons.check_box_outline_blank,
      isSelected: selectedSort == sort,
      onPressed: selectedSort == sort ? () {} : () => onSortChanged(sort),
    );
  }

  String _sortLabel(AppLocalizations l10n, CommentSort sort) => switch (sort) {
    CommentSort.oldest => l10n.postCommentsSortOldest,
    CommentSort.newest => l10n.postCommentsSortNewest,
    CommentSort.follows => l10n.postCommentsSortFollows,
  };
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
    required this.replyCount,
    required this.isLoading,
    required this.onCollapseReplies,
    required this.onLoadMore,
    this.showHide = true,
  });

  final String commentUri;
  final String? repliesCursor;
  final int replyCount;
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
                    : Text(l10n.postCommentsViewReplyCount(replyCount)),
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
