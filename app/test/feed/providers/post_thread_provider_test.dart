import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/post_thread.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/post_thread_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

Post _post(String rkey) => PostMapper.fromMap({
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
});

void main() {
  setUpAll(initializeMappers);

  group('directRepliesProvider', () {
    test('passes identifiers and pagination to repository', () async {
      final calls = <(String, String, String?, int?)>[];
      final fake = FakePostRepository(
        onListDirectReplies: (did, rkey, {cursor, limit}) async {
          calls.add((did, rkey, cursor, limit));
          return PostPage(items: [_post('reply')], cursor: 'next');
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final page = await container.read(
        directRepliesProvider(
          'did:plc:alice',
          'root',
          cursor: 'c1',
          limit: 20,
        ).future,
      );

      expect(calls, [('did:plc:alice', 'root', 'c1', 20)]);
      expect(page.items.single.rkey, 'reply');
      expect(page.cursor, 'next');
    });
  });

  group('postThreadProvider', () {
    test('passes identifiers to repository', () async {
      final calls = <(String, String)>[];
      final fake = FakePostRepository(
        onThread: (did, rkey) async {
          calls.add((did, rkey));
          return PostThread(
            post: _post(rkey),
            replies: const [],
            truncated: false,
          );
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final thread = await container.read(
        postThreadProvider('did:plc:alice', 'root').future,
      );

      expect(calls, [('did:plc:alice', 'root')]);
      expect(thread.post.rkey, 'root');
      expect(thread.truncated, isFalse);
    });
  });
}
