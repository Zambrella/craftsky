import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_post_stats_overlay.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('successive publications accumulate, deduplicate and expire', (
    tester,
  ) async {
    var changes = 0;
    final overlay = ProfilePostStatsOverlay(
      lease: AccountSessionLease(
        account: AccountKey('did:plc:alice'),
        sessionGeneration: 1,
      ),
      changed: () => changes++,
    );
    addTearDown(overlay.dispose);
    final baseline = Profile(
      did: 'did:plc:alice',
      handle: 'alice.test',
      crafts: const [],
      postCount: 5,
      projectCount: 3,
      postsLast7Days: 2,
    );
    final post = PostMapper.fromMap({
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/new',
      'cid': 'bafy_new',
      'rkey': 'new',
      'text': 'new',
      'tags': <String>[],
      'likeCount': 0,
      'repostCount': 0,
      'replyCount': 0,
      'viewerHasLiked': false,
      'viewerHasReposted': false,
      'viewerHasSaved': false,
      'sponsored': false,
      'createdAt': '2026-11-08T12:00:00Z',
      'indexedAt': '2026-11-08T12:00:00Z',
      'author': {'did': 'did:plc:alice', 'handle': 'alice.test'},
    });
    overlay
      ..accept(post, baseline)
      ..accept(post, baseline);
    expect(overlay.apply(baseline).postCount, 6);
    expect(changes, 1);
    overlay.accept(
      post.copyWith(
        uri: 'at://did:plc:alice/social.craftsky.feed.post/project',
        project: const Project(
          common: ProjectCommon(craftType: 'social.craftsky.feed.defs#sewing'),
        ),
      ),
      baseline,
    );
    final updated = overlay.apply(baseline);
    expect(updated.postCount, 6);
    expect(updated.projectCount, 4);
    expect(updated.postsLast7Days, 4);
    expect(overlay.apply(updated).postsLast7Days, 4);
    expect(
      overlay.apply(baseline.copyWith(did: 'did:plc:bob')).postsLast7Days,
      2,
    );
    await tester.pump(const Duration(seconds: 30));
    expect(overlay.hasPending, isFalse);
    expect(overlay.apply(baseline).postsLast7Days, 2);
    expect(changes, 3);
  });
}
