import 'package:craftsky_app/feed/models/post.dart';
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
}

@MappableClass()
/// Opaque-cursor page of top-level comments in render order.
class CommentPage with CommentPageMappable {
  const CommentPage({required this.items, this.cursor});

  final List<CommentItem> items;
  final String? cursor;
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
}

@MappableClass()
/// Loaded or collapsed child-reply page for one comment branch.
class ReplyPage with ReplyPageMappable {
  const ReplyPage({required this.loaded, required this.items, this.cursor});

  final bool loaded;
  final List<ReplyItem> items;
  final String? cursor;
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

@MappableClass()
/// Author metadata for the true parent when a deeper reply is flattened.
class ReplyingToAuthor with ReplyingToAuthorMappable {
  const ReplyingToAuthor({
    required this.uri,
    required this.did,
    required this.handle,
    this.displayName,
  });

  final String uri;
  final String did;
  final String handle;
  final String? displayName;
}

@MappableClass()
/// Focus metadata returned with a comment-section response.
class FocusContext with FocusContextMappable {
  const FocusContext({
    required this.uri,
    required this.status,
    this.kind,
    this.commentUri,
  });

  final String uri;
  final FocusStatus status;
  final FocusKind? kind;
  final String? commentUri;
}

/// Reply graph edge used by state helpers to flatten backend nesting.
class ReplyTreeEdge {
  const ReplyTreeEdge({required this.item, required this.parentUri});

  final ReplyItem item;
  final String parentUri;
}

/// Sorts comments by product grouping: viewer-authored first, then sort order.
List<CommentItem> sortCommentItemsForViewer(
  Iterable<CommentItem> items, {
  required String viewerDid,
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
  required String rootUri,
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

/// Creates the initial unfocused UI state with all reply branches collapsed.
PostCommentSection initialCommentSectionState(PostCommentSection section) {
  return PostCommentSection(
    post: section.post,
    sort: section.sort,
    focus: section.focus,
    comments: CommentPage(
      cursor: section.comments.cursor,
      items: [
        for (final item in section.comments.items)
          CommentItem(
            post: item.post,
            placement: item.placement,
            replies: const ReplyPage(loaded: false, items: []),
          ),
      ],
    ),
  );
}

/// Returns the primary branch action available for a comment item.
BranchControl branchControlFor(CommentItem item) {
  return item.replies.loaded
      ? BranchControl.hideReplies
      : BranchControl.viewReplies;
}

/// Replaces a comment branch with a loaded reply page.
PostCommentSection setCommentReplies(
  PostCommentSection section, {
  required String commentUri,
  required List<ReplyItem> replies,
  String? cursor,
}) {
  return _mapComment(section, commentUri, (item) {
    return CommentItem(
      post: item.post,
      placement: item.placement,
      replies: ReplyPage(loaded: true, items: replies, cursor: cursor),
    );
  });
}

/// Collapses a comment branch without retaining visible reply items.
PostCommentSection collapseCommentReplies(
  PostCommentSection section, {
  required String commentUri,
}) {
  return _mapComment(section, commentUri, (item) {
    return CommentItem(
      post: item.post,
      placement: item.placement,
      replies: const ReplyPage(loaded: false, items: []),
    );
  });
}

/// Inserts a newly-created reply into the nearest visible comment branch.
PostCommentSection insertCreatedReplyIntoNearestBranch(
  PostCommentSection section, {
  required String parentUri,
  required ReplyItem reply,
}) {
  for (final comment in section.comments.items) {
    final directToComment = parentUri == comment.post.uri;
    final withinLoadedBranch = comment.replies.items.any(
      (item) => item.post.uri == parentUri,
    );
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
            replyingTo: reply.replyingTo,
          );
    return _mapComment(section, comment.post.uri, (comment) {
      return CommentItem(
        post: comment.post,
        placement: comment.placement,
        replies: ReplyPage(
          loaded: true,
          items: [...comment.replies.items, item],
          cursor: comment.replies.cursor,
        ),
      );
    });
  }
  return section;
}

PostCommentSection appendCommentPageDeduplicating(
  PostCommentSection section,
  CommentPage page,
) {
  final seen = <String>{};
  final items = <CommentItem>[];
  for (final item in [...section.comments.items, ...page.items]) {
    if (!seen.add(item.post.uri)) continue;
    items.add(item);
  }
  return PostCommentSection(
    post: section.post,
    sort: section.sort,
    focus: section.focus,
    comments: CommentPage(items: items, cursor: page.cursor),
  );
}

PostCommentSection prependCreatedComment(
  PostCommentSection section,
  Post post,
) {
  final created = CommentItem(
    post: post,
    placement: CommentPlacement.viewerAuthored,
    replies: const ReplyPage(loaded: false, items: []),
  );
  final existing = section.comments.items.where(
    (item) => item.post.uri != post.uri,
  );
  final focused = existing.where(
    (item) => item.placement == CommentPlacement.focused,
  );
  final rest = existing.where(
    (item) => item.placement != CommentPlacement.focused,
  );
  return PostCommentSection(
    post: section.post,
    sort: section.sort,
    focus: section.focus,
    comments: CommentPage(
      cursor: section.comments.cursor,
      items: [...focused, created, ...rest],
    ),
  );
}

PostCommentSection changeCommentSortClearingFocus(
  PostCommentSection section, {
  required String viewerDid,
  required CommentSort sort,
}) {
  final normalized = [
    for (final item in section.comments.items)
      CommentItem(
        post: item.post,
        placement: item.post.author.did == viewerDid
            ? CommentPlacement.viewerAuthored
            : CommentPlacement.normal,
        replies: item.replies,
      ),
  ];
  return PostCommentSection(
    post: section.post,
    sort: sort,
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
  PostCommentSection section,
  String commentUri,
  CommentItem Function(CommentItem item) update,
) {
  return PostCommentSection(
    post: section.post,
    sort: section.sort,
    focus: section.focus,
    comments: CommentPage(
      cursor: section.comments.cursor,
      items: [
        for (final item in section.comments.items)
          if (item.post.uri == commentUri) update(item) else item,
      ],
    ),
  );
}
