import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

void main() {
  setUpAll(initializeMappers);

  group('postProvider', () {
    test('returns the post fetched from the repository', () async {
      final aliceDid = Did.parse('did:plc:alice');
      final rkey = RecordKey.parse('3lf2abc');
      final fake = FakePostRepository(
        onFetch: (did, rkey) async => PostMapper.fromMap({
          'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
          'cid': 'bafy123',
          'rkey': rkey,
          'text': 'hello',
          'tags': <String>[],
          'likeCount': 0,
          'repostCount': 0,
          'replyCount': 0,
          'viewerHasLiked': false,
          'viewerHasReposted': false,
          'createdAt': '2026-05-04T18:23:45.000Z',
          'indexedAt': '2026-05-04T18:23:47.000Z',
          'author': {'did': did, 'handle': 'alice.craftsky.social'},
        }),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final post = await container.read(
        postProvider(aliceDid, rkey).future,
      );
      expect(post.rkey, '3lf2abc');
      expect(post.author.did, 'did:plc:alice');
    });

    test('propagates repository errors as AsyncError', () async {
      final aliceDid = Did.parse('did:plc:alice');
      final rkey = RecordKey.parse('missing');
      final fake = FakePostRepository(
        onFetch: (_, _) async => throw Exception('boom'),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      // Listen to keep the provider alive while the future settles, then
      // assert on the resolved AsyncValue.
      final sub = container.listen(
        postProvider(aliceDid, rkey),
        (_, $) {},
        fireImmediately: true,
      );

      // Pump the event loop so the async error completes and propagates.
      for (var i = 0; i < 5; i++) {
        await Future<void>.delayed(Duration.zero);
      }

      final state = sub.read();
      expect(state.hasError, isTrue);
      expect(state.error, isA<Exception>());
    });
  });
}
