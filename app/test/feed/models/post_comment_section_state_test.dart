import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Post post(String did, String rkey, DateTime createdAt) => Post(
    uri: 'at://$did/social.craftsky.feed.post/$rkey',
    cid: 'bafy_$rkey',
    rkey: rkey,
    text: rkey,
    tags: const [],
    createdAt: createdAt,
    indexedAt: createdAt,
    author: PostAuthor(did: did, handle: '$rkey.craftsky.social'),
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
  );

  Post postWithReplyCount(
    String did,
    String rkey,
    DateTime createdAt,
    int replyCount,
  ) => post(did, rkey, createdAt).copyWith(replyCount: replyCount);

  CommentItem comment(String did, String rkey, int minute) => CommentItem(
    post: post(did, rkey, DateTime.utc(2026, 5, 1, 12, minute)),
    placement: CommentPlacement.normal,
    replies: const ReplyPage(loaded: false, items: []),
  );

  ReplyItem reply(String did, String rkey, int minute) => ReplyItem(
    post: post(did, rkey, DateTime.utc(2026, 5, 1, 12, minute)),
    flattened: false,
  );

  group('PostCommentSectionState sorting', () {
    test(
      'groups viewer-authored comments first and maps follows to oldest',
      () {
        final items = [
          comment('did:plc:other', 'other-early', 1),
          comment('did:plc:viewer', 'viewer-mid', 2),
          comment('did:plc:other', 'other-late', 3),
          comment('did:plc:viewer', 'viewer-late', 4),
        ];

        expect(
          sortCommentItemsForViewer(
            items,
            viewerDid: 'did:plc:viewer',
            sort: CommentSort.oldest,
          ).map((item) => item.post.rkey),
          ['viewer-mid', 'viewer-late', 'other-early', 'other-late'],
        );
        expect(
          sortCommentItemsForViewer(
            items,
            viewerDid: 'did:plc:viewer',
            sort: CommentSort.follows,
          ).map((item) => item.post.rkey),
          ['viewer-mid', 'viewer-late', 'other-early', 'other-late'],
        );
        expect(
          sortCommentItemsForViewer(
            items,
            viewerDid: 'did:plc:viewer',
            sort: CommentSort.newest,
          ).map((item) => item.post.rkey),
          ['viewer-late', 'viewer-mid', 'other-late', 'other-early'],
        );
      },
    );

    test('orders replies oldest-first regardless of comment sort', () {
      final replies = [
        reply('did:plc:other', 'late', 3),
        reply('did:plc:other', 'early', 1),
        reply('did:plc:other', 'middle', 2),
      ];

      for (final sort in CommentSort.values) {
        expect(
          sortReplyItems(
            replies,
            commentSort: sort,
          ).map((item) => item.post.rkey),
          ['early', 'middle', 'late'],
        );
      }
    });

    test('flattens deeper replies to nearest comment branch', () {
      final root = post('did:plc:alice', 'root', DateTime.utc(2026, 5, 1, 12));
      final topComment = comment('did:plc:bob', 'comment', 1);
      final firstReply = reply('did:plc:carol', 'reply', 2);
      final deeperReply = reply('did:plc:dave', 'deeper', 3);

      final branches = flattenRepliesToCommentBranches(
        rootUri: root.uri,
        comments: [topComment],
        replies: [
          ReplyTreeEdge(item: firstReply, parentUri: topComment.post.uri),
          ReplyTreeEdge(item: deeperReply, parentUri: firstReply.post.uri),
        ],
      );

      expect(branches, hasLength(1));
      expect(branches.single.post.uri, topComment.post.uri);
      expect(branches.single.replies.items.map((item) => item.post.rkey), [
        'reply',
        'deeper',
      ]);
      expect(branches.single.replies.items.first.flattened, isFalse);
      expect(branches.single.replies.items.last.flattened, isTrue);
    });

    test('initial comment section state keeps reply lists collapsed', () {
      final section = PostCommentSection(
        post: post('did:plc:alice', 'root', DateTime.utc(2026, 5, 1, 12)),
        sort: CommentSort.oldest,
        comments: CommentPage(
          items: [
            comment('did:plc:bob', 'comment', 1),
          ],
        ),
      );

      final initial = section.withCollapsedReplies();

      expect(initial.comments.items.single.replies.loaded, isFalse);
      expect(initial.comments.items.single.replies.items, isEmpty);
    });

    test('prepends newly created comments ahead of focused branches', () {
      final focused = CommentItem(
        post: post(
          'did:plc:other',
          'focused',
          DateTime.utc(2026, 5, 1, 12, 1),
        ),
        placement: CommentPlacement.focused,
        replies: const ReplyPage(loaded: false, items: []),
      );
      final normal = comment('did:plc:other', 'normal', 2);
      final created = post(
        'did:plc:viewer',
        'created',
        DateTime.utc(2026, 5, 1, 12, 3),
      );
      final section = PostCommentSection(
        post: post('did:plc:alice', 'root', DateTime.utc(2026, 5, 1, 12)),
        sort: CommentSort.oldest,
        focus: FocusContext(
          uri: focused.post.uri,
          status: FocusStatus.included,
          kind: FocusKind.comment,
        ),
        comments: CommentPage(items: [focused, normal]),
      );

      final updated = section.prependCreatedComment(created);

      expect(updated.comments.items.map((item) => item.post.rkey), [
        'created',
        'focused',
        'normal',
      ]);
      expect(
        updated.comments.items.first.placement,
        CommentPlacement.viewerAuthored,
      );
      expect(updated.post.replyCount, 1);
      expect(updated.post.viewerHasReplied, isTrue);
    });

    test('does not double-count a duplicate created comment', () {
      final created = post(
        'did:plc:viewer',
        'created',
        DateTime.utc(2026, 5, 1, 12, 1),
      );
      final section = PostCommentSection(
        post: postWithReplyCount(
          'did:plc:alice',
          'root',
          DateTime.utc(2026, 5, 1, 12),
          4,
        ),
        sort: CommentSort.oldest,
        comments: CommentPage(
          items: [
            CommentItem(
              post: created,
              placement: CommentPlacement.viewerAuthored,
              replies: const ReplyPage(loaded: false, items: []),
            ),
          ],
        ),
      );

      final updated = section.prependCreatedComment(created);

      expect(updated.post.replyCount, 4);
      expect(updated.comments.items, hasLength(1));
    });

    test('branch expansion and collapse state changes controls', () {
      final section = PostCommentSection(
        post: post('did:plc:alice', 'root', DateTime.utc(2026, 5, 1, 12)),
        sort: CommentSort.oldest,
        comments: CommentPage(
          items: [comment('did:plc:bob', 'comment', 1)],
        ),
      );
      final replyItem = reply('did:plc:carol', 'reply', 2);

      expect(
        section.comments.items.single.branchControl,
        BranchControl.viewReplies,
      );

      final expanded = section.setCommentReplies(
        commentUri: section.comments.items.single.post.uri,
        replies: [replyItem],
        cursor: 'more',
      );
      expect(
        expanded.comments.items.single.branchControl,
        BranchControl.hideReplies,
      );
      expect(expanded.comments.items.single.replies.cursor, 'more');

      final collapsed = expanded.collapseCommentReplies(
        commentUri: section.comments.items.single.post.uri,
      );
      expect(
        collapsed.comments.items.single.branchControl,
        BranchControl.viewReplies,
      );
      expect(collapsed.comments.items.single.replies.items, isEmpty);
    });

    test('inserts a new nested reply into the nearest comment branch', () {
      final root = post('did:plc:alice', 'root', DateTime.utc(2026, 5, 1, 12));
      final topComment = comment('did:plc:bob', 'comment', 1);
      final existingReply = reply('did:plc:carol', 'reply', 2);
      final createdReply = reply('did:plc:dave', 'created', 3);
      final section =
          PostCommentSection(
            post: root,
            sort: CommentSort.oldest,
            comments: CommentPage(items: [topComment]),
          ).setCommentReplies(
            commentUri: topComment.post.uri,
            replies: [existingReply],
          );

      final updated = section.insertCreatedReplyIntoNearestBranch(
        parentUri: existingReply.post.uri,
        reply: createdReply,
      );

      expect(
        updated.comments.items.single.replies.items.map(
          (item) => item.post.rkey,
        ),
        [
          'reply',
          'created',
        ],
      );
      expect(
        updated.comments.items.single.replies.items.last.flattened,
        isTrue,
      );
      expect(
        updated.comments.items.single.replies.items.last.replyingTo?.uri,
        existingReply.post.uri,
      );
      expect(
        updated.comments.items.single.replies.items.last.replyingTo?.handle,
        existingReply.post.author.handle,
      );
      expect(updated.comments.items.single.post.replyCount, 2);
      expect(updated.comments.items.single.post.viewerHasReplied, isFalse);
      expect(
        updated.comments.items.single.replies.items.first.post.viewerHasReplied,
        isTrue,
      );
      expect(updated.post.replyCount, 1);
    });

    test('marks a comment when inserting a direct created reply', () {
      final root = post('did:plc:alice', 'root', DateTime.utc(2026, 5, 1, 12));
      final topComment = comment('did:plc:bob', 'comment', 1);
      final createdReply = reply('did:plc:viewer', 'created', 2);
      final section = PostCommentSection(
        post: root,
        sort: CommentSort.oldest,
        comments: CommentPage(items: [topComment]),
      );

      final updated = section.insertCreatedReplyIntoNearestBranch(
        parentUri: topComment.post.uri,
        reply: createdReply,
      );

      expect(updated.comments.items.single.post.viewerHasReplied, isTrue);
      expect(
        updated.comments.items.single.replies.items.single.flattened,
        isFalse,
      );
      expect(updated.post.replyCount, 1);
    });

    test('loaded replies update comment reply count to visible total', () {
      final topComment = CommentItem(
        post: postWithReplyCount(
          'did:plc:bob',
          'comment',
          DateTime.utc(2026, 5, 1, 12, 1),
          15,
        ),
        placement: CommentPlacement.normal,
        replies: const ReplyPage(loaded: false, items: []),
      );
      final section = PostCommentSection(
        post: post('did:plc:alice', 'root', DateTime.utc(2026, 5, 1, 12)),
        sort: CommentSort.oldest,
        comments: CommentPage(items: [topComment]),
      );

      final updated = section.setCommentReplies(
        commentUri: topComment.post.uri,
        replies: [
          for (var i = 0; i < 16; i++) reply('did:plc:reply$i', 'reply-$i', i),
        ],
        incrementRootReplyCount: true,
      );

      expect(updated.comments.items.single.post.replyCount, 16);
      expect(updated.post.replyCount, 1);
    });

    test('de-duplicates viewer-authored comments from later pages', () {
      final viewerComment = CommentItem(
        post: post(
          'did:plc:viewer',
          'viewer-comment',
          DateTime.utc(2026, 5, 1, 12, 1),
        ),
        placement: CommentPlacement.viewerAuthored,
        replies: const ReplyPage(loaded: false, items: []),
      );
      final section = PostCommentSection(
        post: post('did:plc:alice', 'root', DateTime.utc(2026, 5, 1, 12)),
        sort: CommentSort.oldest,
        comments: CommentPage(
          items: [viewerComment, comment('did:plc:other', 'normal', 2)],
          cursor: 'page-2',
        ),
      );

      final updated = section.appendCommentPageDeduplicating(
        CommentPage(
          items: [
            CommentItem(
              post: viewerComment.post,
              placement: CommentPlacement.normal,
              replies: const ReplyPage(loaded: false, items: []),
            ),
            comment('did:plc:other', 'new-normal', 3),
          ],
        ),
      );

      expect(updated.comments.items.map((item) => item.post.rkey), [
        'viewer-comment',
        'normal',
        'new-normal',
      ]);
      expect(
        updated.comments.items.where(
          (item) => item.post.uri == viewerComment.post.uri,
        ),
        hasLength(1),
      );
    });

    test(
      'sort change clears focus promotion and preserves viewer grouping',
      () {
        final focused = CommentItem(
          post: post(
            'did:plc:other',
            'focused',
            DateTime.utc(2026, 5, 1, 12, 1),
          ),
          placement: CommentPlacement.focused,
          replies: const ReplyPage(loaded: false, items: []),
        );
        final viewer = comment('did:plc:viewer', 'viewer', 2);
        final normalLate = comment('did:plc:other', 'normal-late', 3);
        final section = PostCommentSection(
          post: post('did:plc:alice', 'root', DateTime.utc(2026, 5, 1, 12)),
          sort: CommentSort.oldest,
          focus: const FocusContext(
            uri: 'at://did:plc:other/social.craftsky.feed.post/focused',
            status: FocusStatus.included,
            kind: FocusKind.comment,
          ),
          comments: CommentPage(items: [focused, normalLate, viewer]),
        );

        final updated = section.changeCommentSortClearingFocus(
          viewerDid: 'did:plc:viewer',
          sort: CommentSort.newest,
        );

        expect(updated.sort, CommentSort.newest);
        expect(updated.focus, isNull);
        expect(updated.comments.items.map((item) => item.post.rkey), [
          'viewer',
          'normal-late',
          'focused',
        ]);
        expect(updated.comments.items.map((item) => item.placement), [
          CommentPlacement.viewerAuthored,
          CommentPlacement.normal,
          CommentPlacement.normal,
        ]);
      },
    );
  });
}
