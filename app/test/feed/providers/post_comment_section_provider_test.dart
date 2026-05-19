import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart'
    hide PostCommentSection;
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

Post _post(String did, String rkey, int minute) => Post(
  uri: 'at://$did/social.craftsky.feed.post/$rkey',
  cid: 'bafy_$rkey',
  rkey: rkey,
  text: 'post $rkey',
  tags: const [],
  createdAt: DateTime.utc(2026, 5, 1, 12, minute),
  indexedAt: DateTime.utc(2026, 5, 1, 12, minute),
  author: PostAuthor(did: did, handle: '$rkey.craftsky.social'),
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
);

CommentItem _comment(String rkey, int minute) => CommentItem(
  post: _post('did:plc:bob', rkey, minute),
  placement: CommentPlacement.normal,
  replies: const ReplyPage(loaded: false, items: []),
);

CommentItem _expandedComment(
  String rkey,
  int minute, {
  required List<ReplyItem> replies,
  required String? cursor,
}) => CommentItem(
  post: _post('did:plc:bob', rkey, minute),
  placement: CommentPlacement.normal,
  replies: ReplyPage(loaded: true, items: replies, cursor: cursor),
);

ReplyItem _reply(String rkey, int minute) =>
    ReplyItem(post: _post('did:plc:carol', rkey, minute), flattened: false);

PostCommentSection _section({
  required List<CommentItem> comments,
  required String? cursor,
  CommentSort sort = CommentSort.oldest,
}) => PostCommentSection(
  post: _post('did:plc:alice', 'root', 0),
  sort: sort,
  comments: CommentPage(items: comments, cursor: cursor),
);

void main() {
  setUpAll(initializeMappers);

  group('postCommentSectionProvider', () {
    test(
      'top-level load more prevents duplicate loads and tracks cursor',
      () async {
        final secondPage = Completer<PostCommentSection>();
        final calls = <({String? cursor, CommentSort? sort})>[];
        final fake = FakePostRepository(
          onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
            calls.add((cursor: cursor, sort: sort));
            if (cursor == null) {
              return _section(
                comments: [_comment('comment-1', 1)],
                cursor: 'c1',
              );
            }
            return secondPage.future;
          },
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );
        final subscription = container.listen(
          postCommentSectionProvider('did:plc:alice', 'root'),
          (_, _) {},
        );
        addTearDown(subscription.close);

        final initial = await container.read(
          postCommentSectionProvider('did:plc:alice', 'root').future,
        );
        expect(initial.comments.cursor, 'c1');

        final loaderSubscription = container.listen(
          postCommentPageLoaderProvider('did:plc:alice', 'root'),
          (_, _) {},
        );
        addTearDown(loaderSubscription.close);
        final firstLoad = container
            .read(
              postCommentPageLoaderProvider('did:plc:alice', 'root').notifier,
            )
            .load();
        final duplicateLoad = container
            .read(
              postCommentPageLoaderProvider('did:plc:alice', 'root').notifier,
            )
            .load();

        await Future<void>.delayed(Duration.zero);
        expect(calls, [
          (cursor: null, sort: CommentSort.oldest),
          (cursor: 'c1', sort: CommentSort.oldest),
        ]);
        expect(
          container
              .read(postCommentSectionProvider('did:plc:alice', 'root'))
              .hasValue,
          isTrue,
        );
        expect(
          container
              .read(postCommentPageLoaderProvider('did:plc:alice', 'root'))
              .isLoading,
          isTrue,
        );

        secondPage.complete(
          _section(comments: [_comment('comment-2', 2)], cursor: 'c2'),
        );
        await Future.wait([firstLoad, duplicateLoad]);

        final updated = container
            .read(postCommentSectionProvider('did:plc:alice', 'root'))
            .value!;
        expect(updated.comments.items.map((item) => item.post.rkey), [
          'comment-1',
          'comment-2',
        ]);
        expect(updated.comments.cursor, 'c2');
      },
    );

    test(
      'reply load more keeps branch cursors and items independent',
      () async {
        final calls = <({String rkey, String? cursor})>[];
        final commentA = _expandedComment(
          'comment-a',
          1,
          replies: [_reply('a-reply-1', 2)],
          cursor: 'a-cursor',
        );
        final commentB = _expandedComment(
          'comment-b',
          3,
          replies: [_reply('b-reply-1', 4)],
          cursor: 'b-cursor',
        );
        final fake = FakePostRepository(
          onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
              _section(comments: [commentA, commentB], cursor: null),
          onListCommentBranchReplies: (did, rkey, {cursor, limit}) async {
            calls.add((rkey: rkey, cursor: cursor));
            return ReplyPage(loaded: true, items: [_reply('a-reply-2', 5)]);
          },
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );
        final subscription = container.listen(
          postCommentSectionProvider('did:plc:alice', 'root'),
          (_, _) {},
        );
        addTearDown(subscription.close);

        await container.read(
          postCommentSectionProvider('did:plc:alice', 'root').future,
        );
        final loaderSubscription = container.listen(
          postCommentRepliesLoaderProvider(
            'did:plc:alice',
            'root',
            commentUri: commentA.post.uri,
          ),
          (_, _) {},
        );
        addTearDown(loaderSubscription.close);
        await container
            .read(
              postCommentRepliesLoaderProvider(
                'did:plc:alice',
                'root',
                commentUri: commentA.post.uri,
              ).notifier,
            )
            .load();

        final updated = container
            .read(postCommentSectionProvider('did:plc:alice', 'root'))
            .value!;
        final updatedA = updated.comments.items.firstWhere(
          (item) => item.post.uri == commentA.post.uri,
        );
        final updatedB = updated.comments.items.firstWhere(
          (item) => item.post.uri == commentB.post.uri,
        );
        expect(calls, [(rkey: 'comment-a', cursor: 'a-cursor')]);
        expect(updatedA.replies.items.map((item) => item.post.rkey), [
          'a-reply-1',
          'a-reply-2',
        ]);
        expect(updatedA.replies.cursor, isNull);
        expect(updatedB.replies.items.map((item) => item.post.rkey), [
          'b-reply-1',
        ]);
        expect(updatedB.replies.cursor, 'b-cursor');
      },
    );

    test('reply loaders expose per-branch loading state', () async {
      final pendingReplies = Completer<ReplyPage>();
      final commentA = _expandedComment(
        'comment-a',
        1,
        replies: [_reply('a-reply-1', 2)],
        cursor: 'a-cursor',
      );
      final commentB = _expandedComment(
        'comment-b',
        3,
        replies: [_reply('b-reply-1', 4)],
        cursor: 'b-cursor',
      );
      final fake = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            _section(comments: [commentA, commentB], cursor: null),
        onListCommentBranchReplies: (did, rkey, {cursor, limit}) =>
            pendingReplies.future,
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );
      final subscription = container.listen(
        postCommentSectionProvider('did:plc:alice', 'root'),
        (_, _) {},
      );
      addTearDown(subscription.close);

      await container.read(
        postCommentSectionProvider('did:plc:alice', 'root').future,
      );
      final loaderASubscription = container.listen(
        postCommentRepliesLoaderProvider(
          'did:plc:alice',
          'root',
          commentUri: commentA.post.uri,
        ),
        (_, _) {},
      );
      final loaderBSubscription = container.listen(
        postCommentRepliesLoaderProvider(
          'did:plc:alice',
          'root',
          commentUri: commentB.post.uri,
        ),
        (_, _) {},
      );
      addTearDown(loaderASubscription.close);
      addTearDown(loaderBSubscription.close);

      final load = container
          .read(
            postCommentRepliesLoaderProvider(
              'did:plc:alice',
              'root',
              commentUri: commentA.post.uri,
            ).notifier,
          )
          .load();
      await Future<void>.delayed(Duration.zero);

      expect(
        container
            .read(postCommentSectionProvider('did:plc:alice', 'root'))
            .hasValue,
        isTrue,
      );
      expect(
        container
            .read(
              postCommentRepliesLoaderProvider(
                'did:plc:alice',
                'root',
                commentUri: commentA.post.uri,
              ),
            )
            .isLoading,
        isTrue,
      );
      expect(
        container
            .read(
              postCommentRepliesLoaderProvider(
                'did:plc:alice',
                'root',
                commentUri: commentB.post.uri,
              ),
            )
            .isLoading,
        isFalse,
      );

      pendingReplies.complete(
        ReplyPage(loaded: true, items: [_reply('a-reply-2', 5)]),
      );
      await load;

      expect(
        container
            .read(
              postCommentRepliesLoaderProvider(
                'did:plc:alice',
                'root',
                commentUri: commentA.post.uri,
              ),
            )
            .hasValue,
        isTrue,
      );
      final updated = container
          .read(postCommentSectionProvider('did:plc:alice', 'root'))
          .value!;
      final updatedA = updated.comments.items.firstWhere(
        (item) => item.post.uri == commentA.post.uri,
      );
      final updatedB = updated.comments.items.firstWhere(
        (item) => item.post.uri == commentB.post.uri,
      );
      expect(updatedA.replies.items.map((item) => item.post.rkey), [
        'a-reply-1',
        'a-reply-2',
      ]);
      expect(updatedB.replies.items.map((item) => item.post.rkey), [
        'b-reply-1',
      ]);
    });
  });
}
