import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/post_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

Post _post(String rkey) => PostMapper.fromMap({
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': rkey,
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

  test(
    'IT-012 post provider fetches initial state through repository',
    () async {
      String? seenQ;
      final fake = FakeSearchRepository(
        onSearchPosts: ({required q, sort, limit, cursor}) async {
          seenQ = q;
          return SearchPostPage(items: [_post('a')]);
        },
      );
      final container = ProviderContainer.test(
        overrides: [searchRepositoryProvider.overrideWithValue(fake)],
      );

      final state = await container.read(
        postSearchProvider(const PostSearchQuery(q: 'alpaca')).future,
      );

      expect(seenQ, 'alpaca');
      expect(state.items.single.rkey.toString(), 'a');
    },
  );

  test(
    'IT-013 post loadMore passes cursor, appends, de-dupes, and no-ops at end',
    () async {
      var calls = 0;
      String? seenCursor;
      final fake = FakeSearchRepository(
        onSearchPosts: ({required q, sort, limit, cursor}) async {
          calls++;
          if (calls == 1) {
            return SearchPostPage(items: [_post('a')], cursor: 'opaque:posts');
          }
          seenCursor = cursor;
          return SearchPostPage(items: [_post('a'), _post('b')]);
        },
      );
      final container = ProviderContainer.test(
        overrides: [searchRepositoryProvider.overrideWithValue(fake)],
      );
      final provider = postSearchProvider(const PostSearchQuery(q: 'alpaca'));

      await container.read(provider.future);
      await container.read(provider.notifier).loadMore();
      await container.read(provider.notifier).loadMore();

      final state = container.read(provider).value!;
      expect(seenCursor, 'opaque:posts');
      expect(state.items.map((post) => post.rkey.toString()), ['a', 'b']);
      expect(state.hasMore, isFalse);
      expect(calls, 2);
    },
  );
}
