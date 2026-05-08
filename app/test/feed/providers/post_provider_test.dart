import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

void main() {
  setUpAll(initializeMappers);

  group('postProvider', () {
    test('returns the post fetched from the repository', () async {
      final fake = FakePostRepository(
        onFetch: (did, rkey) async => PostMapper.fromMap({
          'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
          'cid': 'bafy123',
          'rkey': rkey,
          'text': 'hello',
          'tags': <String>[],
          'createdAt': '2026-05-04T18:23:45.000Z',
          'indexedAt': '2026-05-04T18:23:47.000Z',
          'author': {'did': did, 'handle': 'alice.craftsky.social'},
        }),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final post = await container.read(
        postProvider('did:plc:alice', '3lf2abc').future,
      );
      expect(post.rkey, '3lf2abc');
      expect(post.author.did, 'did:plc:alice');
    });

    test('propagates repository errors as AsyncError', () async {
      final fake = FakePostRepository(
        onFetch: (_, _) async => throw Exception('boom'),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      // Listen to keep the provider alive while the future settles, then
      // assert on the resolved AsyncValue.
      final sub = container.listen(
        postProvider('did:plc:alice', 'missing'),
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
