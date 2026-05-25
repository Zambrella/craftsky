import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'post_comment_section.mapper.dart';

@MappableEnum()
/// User-selectable ordering for top-level comments.
enum CommentSort { oldest, newest, follows }

@MappableEnum()
/// Server-assigned render group for a comment in the ordered comment list.
enum CommentPlacement { focused, viewerAuthored, normal }

@MappableEnum()
/// Backend focus-resolution status for a comment/reply deep link.
enum FocusStatus { included, notFound, mismatchedRoot }

@MappableEnum()
/// Kind of indexed item identified by a successful focus resolution.
enum FocusKind { comment, reply }

/// Action currently available for a comment branch's reply controls.
enum BranchControl { viewReplies, hideReplies }

@MappableClass()
/// Root-post comment-section response consumed by the post route.
class PostCommentSection with PostCommentSectionMappable {
  const PostCommentSection({
    required this.post,
    required this.comments,
    required this.sort,
    this.focus,
  });

  final Post post;
  final CommentPage comments;
  final CommentSort sort;
  final FocusContext? focus;

  @override
  String toString() {
    final loadedReplies = comments.items.fold<int>(
      0,
      (total, item) => total + item.replies.items.length,
    );
    return 'PostCommentSection('
        'post: ${post.uri}, '
        'comments: ${comments.items.length}, '
        'loadedReplies: $loadedReplies, '
        'sort: ${sort.name}, '
        'focus: ${focus?.status.name}'
        ')';
  }

  /// Creates the initial unfocused UI state with all reply branches collapsed.
  PostCommentSection withCollapsedReplies() {
    return copyWith(
      comments: comments.copyWith(
        items: [
          for (final item in comments.items)
            item.copyWith(
              replies: const ReplyPage(loaded: false, items: []),
            ),
        ],
      ),
    );
  }

  /// Replaces the root post, a visible comment, or a visible reply.
  PostCommentSection replacePost(Post post) {
    return copyWith(
      post: this.post.uri == post.uri || this.post.rkey == post.rkey
          ? post
          : this.post,
      comments: comments.copyWith(
        items: [
          for (final item in comments.items)
            item.copyWith(
              post: item.post.uri == post.uri || item.post.rkey == post.rkey
                  ? post
                  : item.post,
              replies: item.replies.copyWith(
                items: [
                  for (final reply in item.replies.items)
                    reply.copyWith(
                      post:
                          reply.post.uri == post.uri ||
                              reply.post.rkey == post.rkey
                          ? post
                          : reply.post,
                    ),
                ],
              ),
            ),
        ],
      ),
    );
  }

  /// Replaces a comment branch with a loaded reply page.
  PostCommentSection setCommentReplies({
    required AtUri commentUri,
    required List<ReplyItem> replies,
    String? cursor,
    bool incrementRootReplyCount = false,
  }) {
    final updated = _mapComment(commentUri, (item) {
      final replyCount = replies.length > item.post.replyCount
          ? replies.length
          : item.post.replyCount;
      return item.copyWith(
        post: item.post.copyWith(replyCount: replyCount),
        replies: ReplyPage(loaded: true, items: replies, cursor: cursor),
      );
    });
    return updated._incrementRootReplyCountIf(incrementRootReplyCount);
  }

  /// Collapses a comment branch without retaining visible reply items.
  PostCommentSection collapseCommentReplies({required AtUri commentUri}) {
    return _mapComment(commentUri, (item) {
      return item.copyWith(
        replies: const ReplyPage(loaded: false, items: []),
      );
    });
  }

  /// Inserts a newly-created reply into the nearest visible comment branch.
  PostCommentSection insertCreatedReplyIntoNearestBranch({
    required AtUri parentUri,
    required ReplyItem reply,
  }) {
    for (final comment in comments.items) {
      final directToComment = parentUri == comment.post.uri;
      final parentReply = comment.replies.items
          .where(
            (item) => item.post.uri == parentUri,
          )
          .firstOrNull;
      final withinLoadedBranch = parentReply != null;
      if (!directToComment && !withinLoadedBranch) continue;

      final item = directToComment
          ? ReplyItem(
              post: reply.post,
              flattened: false,
              replyingTo: reply.replyingTo,
            )
          : ReplyItem(
              post: reply.post,
              flattened: true,
              replyingTo:
                  reply.replyingTo ??
                  ReplyingToAuthor(
                    uri: parentReply!.post.uri,
                    did: parentReply.post.author.did,
                    handle: parentReply.post.author.handle,
                    displayName: parentReply.post.author.displayName,
                  ),
            );
      final alreadyVisible = comment.replies.items.any(
        (reply) => reply.post.uri == item.post.uri,
      );
      return _mapComment(comment.post.uri, (comment) {
        final replies =
            [
              for (final reply in comment.replies.items)
                if (reply.post.uri != item.post.uri)
                  reply.post.uri == parentUri
                      ? reply.copyWith(
                          post: reply.post.copyWith(viewerHasReplied: true),
                        )
                      : reply,
              item,
            ]..sort(
              (left, right) => left.post.createdAt.compareTo(
                right.post.createdAt,
              ),
            );
        final replyCount = alreadyVisible
            ? comment.post.replyCount
            : comment.post.replyCount + 1;
        return comment.copyWith(
          post: comment.post.copyWith(
            replyCount: replyCount,
            viewerHasReplied: directToComment || comment.post.viewerHasReplied,
          ),
          replies: comment.replies.copyWith(
            loaded: true,
            items: replies,
          ),
        );
      })._incrementRootReplyCountIf(!alreadyVisible);
    }
    return this;
  }

  /// Appends a comment page while preserving the first item for each URI.
  PostCommentSection appendCommentPageDeduplicating(CommentPage page) {
    final seen = <String>{};
    final items = <CommentItem>[];
    for (final item in [...comments.items, ...page.items]) {
      if (!seen.add(item.post.uri)) continue;
      items.add(item);
    }
    return copyWith(
      comments: CommentPage(items: items, cursor: page.cursor),
    );
  }

  /// Prepends a newly-created comment ahead of server-provided ordering.
  PostCommentSection prependCreatedComment(Post post) {
    final created = CommentItem(
      post: post,
      placement: CommentPlacement.viewerAuthored,
      replies: const ReplyPage(loaded: false, items: []),
    );
    final alreadyVisible = comments.items.any(
      (item) => item.post.uri == post.uri,
    );
    final existing = comments.items.where((item) => item.post.uri != post.uri);
    return copyWith(
      post: alreadyVisible
          ? this.post
          : this.post.copyWith(
              replyCount: this.post.replyCount + 1,
              viewerHasReplied: true,
            ),
      comments: comments.copyWith(items: [created, ...existing]),
    );
  }

  PostCommentSection _incrementRootReplyCountIf(bool shouldIncrement) {
    if (!shouldIncrement) return this;
    return copyWith(post: post.copyWith(replyCount: post.replyCount + 1));
  }

  /// Clears focus promotion and applies viewer grouping under a new sort.
  PostCommentSection changeCommentSortClearingFocus({
    required Did viewerDid,
    required CommentSort sort,
  }) {
    final normalized = [
      for (final item in comments.items)
        item.copyWith(
          placement: item.post.author.did == viewerDid
              ? CommentPlacement.viewerAuthored
              : CommentPlacement.normal,
        ),
    ];
    return copyWith(
      sort: sort,
      focus: null,
      comments: CommentPage(
        items: sortCommentItemsForViewer(
          normalized,
          viewerDid: viewerDid,
          sort: sort,
        ),
      ),
    );
  }

  PostCommentSection _mapComment(
    AtUri commentUri,
    CommentItem Function(CommentItem item) update,
  ) {
    return copyWith(
      comments: comments.copyWith(
        items: [
          for (final item in comments.items)
            if (item.post.uri == commentUri) update(item) else item,
        ],
      ),
    );
  }
}

@MappableClass()
/// Opaque-cursor page of top-level comments in render order.
class CommentPage with CommentPageMappable {
  const CommentPage({required this.items, this.cursor});

  final List<CommentItem> items;
  final String? cursor;

  bool get hasMore => cursor != null;

  @override
  String toString() {
    return 'CommentPage(items: ${items.length}, hasMore: $hasMore)';
  }
}

@MappableClass()
/// A top-level comment plus placement and child-reply loaded state.
class CommentItem with CommentItemMappable {
  const CommentItem({
    required this.post,
    required this.placement,
    required this.replies,
  });

  final Post post;
  final CommentPlacement placement;
  final ReplyPage replies;

  /// Primary branch action available for this comment item.
  BranchControl get branchControl {
    return replies.loaded
        ? BranchControl.hideReplies
        : BranchControl.viewReplies;
  }
}

@MappableClass()
/// Loaded or collapsed child-reply page for one comment branch.
class ReplyPage with ReplyPageMappable {
  const ReplyPage({required this.loaded, required this.items, this.cursor});

  final bool loaded;
  final List<ReplyItem> items;
  final String? cursor;

  bool get hasMore => cursor != null;

  @override
  String toString() {
    return 'ReplyPage(loaded: $loaded, items: ${items.length}, '
        'hasMore: $hasMore)';
  }
}

@MappableClass()
/// Visual reply item rendered under a top-level comment.
class ReplyItem with ReplyItemMappable {
  const ReplyItem({
    required this.post,
    required this.flattened,
    this.replyingTo,
  });

  final Post post;
  final bool flattened;
  final ReplyingToAuthor? replyingTo;
}

@MappableClass(
  includeCustomMappers: [AtUriMapper(), DidMapper(), HandleMapper()],
)
/// Author metadata for the true parent when a deeper reply is flattened.
class ReplyingToAuthor with ReplyingToAuthorMappable {
  ReplyingToAuthor({
    required String uri,
    required String did,
    required String handle,
    this.displayName,
  }) : uri = AtUri.parse(uri),
       did = Did.parse(did),
       handle = Handle.parse(handle);

  final AtUri uri;
  final Did did;
  final Handle handle;
  final String? displayName;
}

@MappableClass(includeCustomMappers: [AtUriMapper()])
/// Focus metadata returned with a comment-section response.
class FocusContext with FocusContextMappable {
  FocusContext({
    required String uri,
    required this.status,
    this.kind,
    String? commentUri,
  }) : uri = AtUri.parse(uri),
       commentUri = commentUri == null ? null : AtUri.parse(commentUri);

  final AtUri uri;
  final FocusStatus status;
  final FocusKind? kind;
  final AtUri? commentUri;
}

/// Reply graph edge used by state helpers to flatten backend nesting.
class ReplyTreeEdge {
  const ReplyTreeEdge({required this.item, required this.parentUri});

  final ReplyItem item;
  final AtUri parentUri;
}

/// Sorts comments by product grouping: viewer-authored first, then sort order.
List<CommentItem> sortCommentItemsForViewer(
  Iterable<CommentItem> items, {
  required Did viewerDid,
  required CommentSort sort,
}) {
  final sorted = items.toList();
  final direction = sort == CommentSort.newest ? -1 : 1;
  sorted.sort((left, right) {
    final leftViewer = left.post.author.did == viewerDid;
    final rightViewer = right.post.author.did == viewerDid;
    if (leftViewer != rightViewer) return leftViewer ? -1 : 1;
    final created = left.post.createdAt.compareTo(right.post.createdAt);
    if (created != 0) return created * direction;
    return left.post.uri.compareTo(right.post.uri) * direction;
  });
  return sorted;
}

/// Sorts visual replies oldest-first regardless of selected comment sort.
List<ReplyItem> sortReplyItems(
  Iterable<ReplyItem> items, {
  required CommentSort commentSort,
}) {
  final sorted = items.toList()
    ..sort((left, right) {
      final created = left.post.createdAt.compareTo(right.post.createdAt);
      if (created != 0) return created;
      return left.post.uri.compareTo(right.post.uri);
    });
  return sorted;
}

/// Maps deeper backend replies into their nearest loaded comment branch.
List<CommentItem> flattenRepliesToCommentBranches({
  required AtUri rootUri,
  required Iterable<CommentItem> comments,
  required Iterable<ReplyTreeEdge> replies,
}) {
  final commentByUri = {
    for (final comment in comments) comment.post.uri: comment,
  };
  final parentByReplyUri = {
    for (final edge in replies) edge.item.post.uri: edge.parentUri,
  };
  final branchReplies = <String, List<ReplyItem>>{
    for (final comment in comments) comment.post.uri: <ReplyItem>[],
  };

  String? branchFor(ReplyTreeEdge edge) {
    var parentUri = edge.parentUri;
    while (parentUri != rootUri) {
      if (commentByUri.containsKey(parentUri)) return parentUri;
      final next = parentByReplyUri[parentUri];
      if (next == null) return null;
      parentUri = next;
    }
    return null;
  }

  for (final edge in replies) {
    final branchUri = branchFor(edge);
    if (branchUri == null) continue;
    final direct = edge.parentUri == branchUri;
    final item = direct
        ? edge.item
        : ReplyItem(
            post: edge.item.post,
            flattened: true,
            replyingTo: edge.item.replyingTo,
          );
    branchReplies[branchUri]!.add(item);
  }

  return [
    for (final comment in comments)
      CommentItem(
        post: comment.post,
        placement: comment.placement,
        replies: ReplyPage(
          loaded: true,
          items: sortReplyItems(
            branchReplies[comment.post.uri] ?? const [],
            commentSort: CommentSort.oldest,
          ),
          cursor: comment.replies.cursor,
        ),
      ),
  ];
}
