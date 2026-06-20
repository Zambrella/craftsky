import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

Map<String, dynamic> _postMap(String rkey) => {
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
};

Post _post(String rkey) => PostMapper.fromMap(_postMap(rkey));

void main() {
  setUpAll(initializeMappers);

  test('IT-012 initial load fetches through SearchRepository', () async {
    String? seenTag;
    SearchSort? seenSort;
    int? seenLimit;
    final fake = FakeSearchRepository(
      onSearchHashtagPosts: (tag, {sort, limit, cursor}) async {
        seenTag = tag;
        seenSort = sort;
        seenLimit = limit;
        expect(cursor, isNull);
        return SearchPostPage(items: [_post('a')], cursor: 'opaque:next');
      },
    );
    final container = ProviderContainer.test(
      overrides: [searchRepositoryProvider.overrideWithValue(fake)],
    );

    final state = await container.read(
      hashtagSearchProvider(
        const HashtagSearchQuery(tag: 'SockKAL', sort: SearchSort.popular),
      ).future,
    );

    expect(seenTag, 'SockKAL');
    expect(seenSort, SearchSort.popular);
    expect(seenLimit, searchResultsPageLimit);
    expect(state.items.map((post) => post.rkey.toString()), ['a']);
    expect(state.hasMore, isTrue);
  });

  test(
    'IT-013 loadMore passes opaque cursor and suppresses duplicates',
    () async {
      var call = 0;
      String? seenCursor;
      final fake = FakeSearchRepository(
        onSearchHashtagPosts: (tag, {sort, limit, cursor}) async {
          call++;
          if (call == 1) {
            return SearchPostPage(
              items: [_post('a')],
              cursor: 'opaque:abc/+/=',
            );
          }
          seenCursor = cursor;
          return SearchPostPage(items: [_post('a'), _post('b')]);
        },
      );
      final container = ProviderContainer.test(
        overrides: [searchRepositoryProvider.overrideWithValue(fake)],
      );
      final provider = hashtagSearchProvider(
        const HashtagSearchQuery(tag: 'SockKAL'),
      );

      await container.read(provider.future);
      await container.read(provider.notifier).loadMore();

      final state = container.read(provider).value!;
      expect(seenCursor, 'opaque:abc/+/=');
      expect(state.items.map((post) => post.rkey.toString()), ['a', 'b']);
      expect(state.hasMore, isFalse);
    },
  );

  test('IT-013 loadMore no-ops while already loading', () async {
    var calls = 0;
    final gate = Completer<SearchPostPage>();
    final fake = FakeSearchRepository(
      onSearchHashtagPosts: (tag, {sort, limit, cursor}) async {
        calls++;
        if (calls == 1) {
          return SearchPostPage(items: [_post('a')], cursor: 'c1');
        }
        return gate.future;
      },
    );
    final container = ProviderContainer.test(
      overrides: [searchRepositoryProvider.overrideWithValue(fake)],
    );
    final provider = hashtagSearchProvider(
      const HashtagSearchQuery(tag: 'SockKAL'),
    );
    final sub = container.listen(provider, (_, _) {});
    addTearDown(sub.close);

    await container.read(provider.future);
    final first = container.read(provider.notifier).loadMore();
    await Future<void>.delayed(Duration.zero);
    await container.read(provider.notifier).loadMore();
    expect(calls, 2);
    gate.complete(SearchPostPage(items: [_post('b')]));
    await first;
  });
}
