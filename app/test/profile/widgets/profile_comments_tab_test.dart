import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_comments_tab.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../feed/fakes/fake_post_repository.dart';
import 'profile_tab_test_harness.dart';

Post _comment(String rkey) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
    cid: 'bafy_$rkey',
    rkey: rkey,
    text: 'comment $rkey',
    tags: const [],
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: false,
    sponsored: false,
    createdAt: DateTime.now().subtract(const Duration(minutes: 3)),
    indexedAt: DateTime.now().subtract(const Duration(minutes: 2)),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
      displayName: 'Alice',
    ),
  );
}

Future<void> _pump(WidgetTester tester, {required FakePostRepository repo}) {
  return pumpProfileTab(
    tester,
    sliver: ProfileCommentsTab(
      did: Did.parse('did:plc:alice'),
      isOwnProfile: false,
    ),
    repository: repo,
  );
}

void main() {
  group('ProfileCommentsTab', () {
    testWidgets('shows comment skeletons during the initial load', (
      tester,
    ) async {
      final pending = Completer<PostPage>();
      final repo = FakePostRepository(
        onListCommentsByAuthor: (_, {cursor, limit}) => pending.future,
      );

      await _pump(tester, repo: repo);
      await tester.pump();

      expect(find.byType(CraftskySkeletonSliverList), findsOneWidget);
      expect(find.byType(CommentRowSkeleton), findsWidgets);
    });

    testWidgets('scrolling near the end appends the next page', (tester) async {
      final calls = <ProfilePageRequest>[];
      final repo = FakePostRepository(
        onListCommentsByAuthor: (_, {cursor, limit}) async {
          calls.add((cursor: cursor, limit: limit));
          if (calls.length == 1) {
            return PostPage(
              items: [for (var i = 0; i < 10; i++) _comment('a$i')],
              cursor: 'c1',
            );
          }
          expect(cursor, 'c1');
          return PostPage(items: [_comment('b')]);
        },
      );

      await expectProfileTabInfiniteScroll(
        tester,
        sliver: ProfileCommentsTab(
          did: Did.parse('did:plc:alice'),
          isOwnProfile: false,
        ),
        repository: repo,
        requests: calls,
        lastInitialItem: find.text('comment a9'),
        appendedItem: find.text('comment b'),
      );
      expect(find.text('Load more comments'), findsNothing);
    });
  });
}
