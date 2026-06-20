import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/recent_searches_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

void main() {
  test(
    'IT-014 result fetches do not auto-save and save/delete refresh recents',
    () async {
      var saveCalls = 0;
      var listCalls = 0;
      String? deletedId;
      final saved = RecentSearchItem(
        id: 'recent_1',
        type: RecentSearchType.hashtag,
        displayLabel: '#SockKAL',
        payload: const HashtagRecentSearchPayload(tag: 'sockkal'),
        updatedAt: DateTime.parse('2026-06-20T10:00:00Z'),
      );
      final fake = FakeSearchRepository(
        onSearchHashtagPosts: (tag, {sort, limit, cursor}) async =>
            const SearchPostPage(items: []),
        onListRecentSearches: () async {
          listCalls++;
          return RecentSearchPage(items: listCalls == 1 ? [] : [saved]);
        },
        onSaveRecentSearch: (request) async {
          saveCalls++;
          return saved;
        },
        onDeleteRecentSearch: (id) async {
          deletedId = id;
        },
      );
      final container = ProviderContainer.test(
        overrides: [searchRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        hashtagSearchProvider(const HashtagSearchQuery(tag: 'SockKAL')).future,
      );
      expect(saveCalls, 0);

      expect(
        await container.read(recentSearchesProvider.future),
        isA<RecentSearchPage>(),
      );
      final result = await container
          .read(recentSearchesProvider.notifier)
          .save(
            const SaveRecentSearchRequest(
              type: RecentSearchType.hashtag,
              displayLabel: '#SockKAL',
              payload: HashtagRecentSearchPayload(tag: 'sockkal'),
            ),
          );
      await container.read(recentSearchesProvider.notifier).delete('recent_1');

      expect(result.id, 'recent_1');
      expect(saveCalls, 1);
      expect(deletedId, 'recent_1');
      expect(listCalls, greaterThanOrEqualTo(2));
    },
  );
}
