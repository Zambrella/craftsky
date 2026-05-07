import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

Map<String, dynamic> _samplePostMap({required String rkey, String? did}) => {
  'uri':
      'at://${did ?? 'did:plc:alice'}/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
  'tags': <String>[],
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {
    'did': did ?? 'did:plc:alice',
    'handle': 'alice.craftsky.social',
  },
};

Post _samplePost({required String rkey, String? did}) =>
    PostMapper.fromMap(_samplePostMap(rkey: rkey, did: did));

void main() {
  setUpAll(initializeMappers);

  group('userPostsProvider build', () {
    test('first build fetches page 1 and surfaces items + cursor', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(
          items: [_samplePost(rkey: 'a'), _samplePost(rkey: 'b')],
          cursor: 'next',
        ),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final state = await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );
      expect(state.items.map((p) => p.rkey), ['a', 'b']);
      expect(state.cursor, 'next');
      expect(state.hasMore, isTrue);
    });

    test('first build with empty page yields hasMore == false', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            const PostPage(items: []),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final state = await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );
      expect(state.items, isEmpty);
      expect(state.cursor, isNull);
      expect(state.hasMore, isFalse);
    });
  });
}
