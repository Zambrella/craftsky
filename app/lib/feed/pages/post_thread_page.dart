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
import 'package:craftsky_app/moderation/widgets/report_flow.dart';
import 'package:craftsky_app/projects/widgets/project_card.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/errors/notification_destination_error.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/widgets/notification_destination_error_state.dart';
import 'package:craftsky_app/shared/widgets/sort_menu_button.dart';
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
    this.initialCreatedPost,
    super.key,
  });

  final Did did;
  final RecordKey rkey;
  final AtUri? focus;
  final craftsky_post.Post? initialCreatedPost;

  @override
  ConsumerState<PostThreadPage> createState() => _PostThreadPageState();
}

class _PostThreadPageState extends ConsumerState<PostThreadPage> {
  CommentSort _sort = CommentSort.oldest;
  PostCommentSection? _lastSection;
  AtUri? _createdTargetUri;
  AtUri? _consumedInitialCreatedPostUri;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final viewerDid = switch (ref.watch(authSessionProvider).value) {
      SignedIn(:final did) => did,
      _ => null,
    };
    final sectionAsync = ref.watch(
      postCommentSectionProvider(
        widget.did,
        widget.rkey,
        sort: _sort,
        focus: widget.focus,
      ),
    );
    final pageLoaderAsync = ref.watch(
      postCommentPageLoaderProvider(
        widget.did,
        widget.rkey,
        sort: _sort,
        focus: widget.focus,
      ),
    );
    final formFactor = FormFactorWidget.of(context);
    final destinationError = sectionAsync.error;
    final isPermanentError =
        destinationError != null &&
        classifyNotificationDestinationError(destinationError) ==
            NotificationDestinationErrorKind.permanentUnavailable;
    final visibleSection = isPermanentError
        ? null
        : sectionAsync.value ?? _lastSection;
    if (sectionAsync case AsyncData(:final value)) {
      _lastSection = value;
    }
    _scheduleInitialCreatedPostSeed(visibleSection);
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
        if (next.hasError) {
          context.showError(l10n.postLikeError);
          ref.read(toggleLikePostProvider.notifier).reset();
          return;
        }
        final post = next.value;
        if (post == null) return;
        ref
            .read(
              postCommentSectionProvider(
                widget.did,
                widget.rkey,
                sort: _sort,
                focus: widget.focus,
              ).notifier,
            )
            .replacePost(post);
      })
      ..listen(toggleRepostPostProvider, (previous, next) {
        final post = next.value;
        if (post == null) return;
        ref
            .read(
              postCommentSectionProvider(
                widget.did,
                widget.rkey,
                sort: _sort,
                focus: widget.focus,
              ).notifier,
            )
            .replacePost(post);
      });
    return Scaffold(
      appBar: AppBar(title: Text(l10n.postThreadTitle)),
      body: isPermanentError
          ? _ThreadError(
              error: destinationError,
              onRetry: () => ref.invalidate(
                postCommentSectionProvider(
                  widget.did,
                  widget.rkey,
                  sort: _sort,
                  focus: widget.focus,
                ),
              ),
            )
          : switch ((sectionAsync, visibleSection)) {
              (_, final section?) => Column(
                children: [
                  if (destinationError != null)
                    _ThreadError(
                      error: destinationError,
                      onRetry: () => ref.invalidate(
                        postCommentSectionProvider(
                          widget.did,
                          widget.rkey,
                          sort: _sort,
                          focus: widget.focus,
                        ),
                      ),
                    ),
                  Expanded(
                    child: _CommentSectionBody(
                      section: section,
                      did: widget.did,
                      rkey: widget.rkey,
                      focus: widget.focus,
                      createdTargetUri: _createdTargetUri,
                      viewerDid: viewerDid,
                      showInlineComposer: formFactor.isLarge,
                      isLoadingMoreComments: pageLoaderAsync.isLoading,
                      isRefreshingComments: isRefreshingComments,
                      onNearEnd: () => ref
                          .read(
                            postCommentPageLoaderProvider(
                              widget.did,
                              widget.rkey,
                              sort: _sort,
                              focus: widget.focus,
                            ).notifier,
                          )
                          .load(),
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
                  ),
                ],
              ),
              (AsyncError(:final error), _) => _ThreadError(
                error: error,
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

  void _scheduleInitialCreatedPostSeed(PostCommentSection? section) {
    final post = widget.initialCreatedPost;
    if (post == null || section == null) return;
    if (_consumedInitialCreatedPostUri == post.uri) return;
    _consumedInitialCreatedPostUri = post.uri;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      unawaited(_handleCreatedThreadPost(post));
    });
  }

  Future<void> _handleCreatedThreadPost(craftsky_post.Post post) async {
    final reply = post.reply;
    if (reply == null) return;

    final section = ref
        .read(
          postCommentSectionProvider(
            widget.did,
            widget.rkey,
            sort: _sort,
            focus: widget.focus,
          ),
        )
        .value;
    if (section == null || reply.root.uri != section.post.uri) return;

    final notifier = ref.read(
      postCommentSectionProvider(
        widget.did,
        widget.rkey,
        sort: _sort,
        focus: widget.focus,
      ).notifier,
    );
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
      final replies = sortReplyItems([
        for (final item in pages)
          if (item.post.uri != post.uri) item,
        ReplyItem(post: post, flattened: false),
      ], commentSort: _sort);
      ref
          .read(
            postCommentSectionProvider(
              widget.did,
              widget.rkey,
              sort: _sort,
              focus: widget.focus,
            ).notifier,
          )
          .setRepliesForComment(
            commentUri: parentComment.post.uri,
            replies: replies,
            incrementRootReplyCount: true,
          );
      if (mounted) setState(() => _createdTargetUri = post.uri);
    } on Object {
      if (!mounted) return;
      ref
          .read(
            postCommentSectionProvider(
              widget.did,
              widget.rkey,
              sort: _sort,
              focus: widget.focus,
            ).notifier,
          )
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
  final Did did;
  final RecordKey rkey;
  final AtUri? focus;
  final AtUri? createdTargetUri;
  final Did? viewerDid;
  final bool showInlineComposer;
  final bool isLoadingMoreComments;
  final bool isRefreshingComments;
  final VoidCallback onNearEnd;
  final void Function(AtUri commentUri) onCollapseReplies;
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
  AtUri? _scrolledTargetUri;
  AtUri? _highlightedTargetUri;
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

  AtUri? _targetUri(_CommentSectionBody widget) {
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

  void _scheduleClearHighlight(AtUri targetUri) {
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
              projectVariant: ProjectCardVariant.detail,
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
              onReport: _reportIfViewerNotOwner(widget.section.post),
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
                    child: SortMenuButton<CommentSort>(
                      selectedValue: widget.selectedSort,
                      options: _commentSortOptions(l10n),
                      onChanged: widget.onSortChanged,
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

  VoidCallback? _reportIfViewerNotOwner(craftsky_post.Post post) {
    if (widget.viewerDid == null || widget.viewerDid == post.author.did) {
      return null;
    }
    return () => showPostReportSheet(context, ref, post);
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
  final Did did;
  final RecordKey rkey;
  final CommentSort sort;
  final AtUri? focus;
  final AtUri? targetUri;
  final AtUri? highlightedUri;
  final GlobalKey focusedTargetKey;
  final Did? viewerDid;
  final void Function(AtUri commentUri) onCollapseReplies;

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
          onReport: _reportIfViewerNotOwner(context, ref, item.post),
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
        if (item.replies.loaded && item.replies.items.isNotEmpty)
          _ReplyBranchBox(
            key: ValueKey('comment-reply-branch-${item.post.uri}'),
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
                    onReply: () =>
                        showPostComposerSheet(context, replyTarget: reply.post),
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
                    onReport: _reportIfViewerNotOwner(context, ref, reply.post),
                  ),
                ),
            ],
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
    return Padding(
      padding: EdgeInsets.symmetric(horizontal: spacing.sp4),
      child: _FocusedTargetWrapper(
        focusTargetKey: focusedTargetKey,
        isFocused: targetComment,
        child: column,
      ),
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
    return () =>
        _confirmDelete(context, ref, post, title: title, message: message);
  }

  VoidCallback? _reportIfViewerNotOwner(
    BuildContext context,
    WidgetRef ref,
    craftsky_post.Post post,
  ) {
    if (viewerDid == null || viewerDid == post.author.did) return null;
    return () => showPostReportSheet(context, ref, post);
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

class _ReplyBranchBox extends StatelessWidget {
  const _ReplyBranchBox({required this.children, super.key});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;

    return Padding(
      padding: EdgeInsetsDirectional.only(
        start: spacing.sp6,
        end: spacing.sp4,
        bottom: spacing.sp2,
      ),
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: swatches.paper2,
          borderRadius: BorderRadius.circular(radii.r3),
          border: Border.all(color: theme.colorScheme.outlineVariant),
        ),
        child: Padding(
          padding: EdgeInsets.symmetric(vertical: spacing.sp2),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: children,
          ),
        ),
      ),
    );
  }
}

List<SortMenuOption<CommentSort>> _commentSortOptions(AppLocalizations l10n) =>
    [
      SortMenuOption(
        value: CommentSort.newest,
        label: l10n.postCommentsSortNewest,
        description: l10n.postCommentsSortNewestDescription,
      ),
      SortMenuOption(
        value: CommentSort.oldest,
        label: l10n.postCommentsSortOldest,
        description: l10n.postCommentsSortOldestDescription,
      ),
      SortMenuOption(
        value: CommentSort.follows,
        label: l10n.postCommentsSortFollows,
        description: l10n.postCommentsSortFollowsDescription,
      ),
    ];

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

  final AtUri commentUri;
  final String? repliesCursor;
  final int replyCount;
  final bool isLoading;
  final bool showHide;
  final void Function(AtUri commentUri) onCollapseReplies;
  final VoidCallback onLoadMore;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final semanticColors = theme.extension<SemanticColorsTheme>()!;
    final showViewReplies = !showHide && repliesCursor == null;
    final compactButtonStyle = _compactButtonStyle(
      spacing,
      foregroundColor: semanticColors.success,
    );
    return Padding(
      padding: EdgeInsetsDirectional.only(
        start: spacing.sp8 + spacing.sp2,
        end: spacing.sp4,
        bottom: showViewReplies ? spacing.sp4 : spacing.sp2,
      ),
      child: Align(
        alignment: Alignment.centerLeft,
        child: Wrap(
          spacing: spacing.sp2,
          children: [
            if (showHide)
              TextButton(
                onPressed: () => onCollapseReplies(commentUri),
                style: compactButtonStyle,
                child: Text(l10n.postCommentsHideReplies),
              )
            else if (showViewReplies)
              TextButton(
                onPressed: isLoading ? null : onLoadMore,
                style: compactButtonStyle,
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
                style: compactButtonStyle,
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

  ButtonStyle _compactButtonStyle(
    SpacingTheme spacing, {
    required Color foregroundColor,
  }) {
    return TextButton.styleFrom(
      foregroundColor: foregroundColor,
      disabledForegroundColor: foregroundColor.withValues(alpha: 0.55),
      minimumSize: Size.zero,
      padding: EdgeInsets.fromLTRB(spacing.sp2, 0, spacing.sp2, spacing.sp2),
      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
    ).copyWith(
      overlayColor: const WidgetStatePropertyAll(Colors.transparent),
      splashFactory: NoSplash.splashFactory,
    );
  }
}

class _ThreadError extends StatelessWidget {
  const _ThreadError({required this.error, required this.onRetry});

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return NotificationDestinationErrorState(
      error: error,
      onRetry: onRetry,
      onBack: () {
        final navigator = Navigator.of(context);
        if (navigator.canPop()) {
          navigator.pop();
        } else {
          const FeedRoute().go(context);
        }
      },
      onViewNotifications: () => const NotificationsRoute().go(context),
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
