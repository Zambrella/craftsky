import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_reposts_tab.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../feed/fakes/fake_post_repository.dart';
import 'profile_tab_test_harness.dart';

void main() {
  testWidgets('renders posts reposted by the profile', (tester) async {
    final post = Post(
      uri: 'at://did:plc:bob/social.craftsky.feed.post/reposted',
      cid: 'bafy-reposted',
      rkey: 'reposted',
      text: 'A reposted pattern',
      tags: const [],
      likeCount: 0,
      repostCount: 1,
      replyCount: 0,
      viewerHasLiked: false,
      viewerHasReposted: false,
      viewerHasSaved: false,
      sponsored: false,
      createdAt: DateTime.utc(2026, 9, 27),
      indexedAt: DateTime.utc(2026, 9, 27),
      author: PostAuthor(
        did: 'did:plc:bob',
        handle: 'bob.craftsky.social',
        displayName: 'Bob',
      ),
    );
    final repository = FakePostRepository(
      onListRepostsByAuthor: (_, {cursor, limit}) async => PostPage(
        items: [post],
      ),
    );

    await pumpProfileTab(
      tester,
      sliver: ProfileRepostsTab(did: Did.parse('did:plc:alice')),
      repository: repository,
    );
    await tester.pumpAndSettle();

    expect(find.byType(PostCard), findsOneWidget);
    expect(find.text('A reposted pattern'), findsOneWidget);
    expect(find.text('Bob'), findsOneWidget);
  });
}
