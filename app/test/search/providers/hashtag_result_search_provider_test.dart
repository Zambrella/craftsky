import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/hashtag_result_search_provider.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

void main() {
  setUpAll(initializeMappers);

  test('UT-010 hashtag result provider paginates independently', () async {
    var calls = 0;
    String? seenQ;
    String? seenCursor;
    final fake = FakeSearchRepository(
      onSearchHashtags: ({required q, limit, cursor}) async {
        calls++;
        seenQ = q;
        if (calls == 1) {
          return const HashtagSearchPage(
            items: [
              HashtagSearchResult(tag: 'sock', postsLast28Days: 4),
            ],
            cursor: 'opaque:hashtags',
          );
        }
        seenCursor = cursor;
        return const HashtagSearchPage(
          items: [
            HashtagSearchResult(tag: 'sock', postsLast28Days: 4),
            HashtagSearchResult(tag: 'sockkal', postsLast28Days: 3),
          ],
        );
      },
    );
    final container = ProviderContainer.test(
      overrides: [searchRepositoryProvider.overrideWithValue(fake)],
    );
    final provider = hashtagResultSearchProvider(
      const HashtagResultSearchQuery(q: 'sock'),
    );

    final initial = await container.read(provider.future);
    await container.read(provider.notifier).loadMore();
    await container.read(provider.notifier).loadMore();

    final state = container.read(provider).value!;
    expect(seenQ, 'sock');
    expect(seenCursor, 'opaque:hashtags');
    expect(initial.hasMore, isTrue);
    expect(state.items.map((item) => item.tag), ['sock', 'sockkal']);
    expect(state.hasMore, isFalse);
    expect(calls, 2);
    expect(searchResultsPageLimit, 25);
  });
}
