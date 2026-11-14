import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/recent_searches_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/member_browsing_context.dart';
import '../fakes/fake_search_repository.dart';

void main() {
  test(
    'save failures reach the provider error state and allow retry',
    () async {
      const failure = ApiNetworkError('offline');
      var fail = true;
      final saved = RecentSearchItem(
        id: 'recent_1',
        type: RecentSearchType.query,
        displayLabel: 'knitting',
        payload: const QueryRecentSearchPayload(q: 'knitting'),
        updatedAt: DateTime.utc(2026),
      );
      final container = ProviderContainer.test(
        overrides: [
          testMemberBrowsingOverride,
          searchRepositoryProvider.overrideWithValue(
            FakeSearchRepository(
              onSaveRecentSearch: (_) async {
                if (fail) throw failure;
                return saved;
              },
            ),
          ),
        ],
      )..listen(saveRecentSearchProvider, (_, _) {});
      await container.read(saveRecentSearchProvider.future);
      final notifier = container.read(saveRecentSearchProvider.notifier);
      const request = SaveRecentSearchRequest(
        type: RecentSearchType.query,
        displayLabel: 'knitting',
        payload: QueryRecentSearchPayload(q: 'knitting'),
      );
      await expectLater(notifier.save(request), completes);
      expect(container.read(saveRecentSearchProvider).error, same(failure));
      expect(container.read(saveRecentSearchProvider).isLoading, isFalse);
      fail = false;
      expect(await notifier.save(request), saved);
      expect(container.read(saveRecentSearchProvider).requireValue, saved);
    },
  );

  test(
    'delete failures reach the provider error state and allow retry',
    () async {
      const failure = ApiNetworkError('offline');
      var fail = true;
      final container = ProviderContainer.test(
        overrides: [
          testMemberBrowsingOverride,
          searchRepositoryProvider.overrideWithValue(
            FakeSearchRepository(
              onDeleteRecentSearch: (_) async {
                if (fail) throw failure;
              },
            ),
          ),
        ],
      )..listen(deleteRecentSearchProvider, (_, _) {});
      await container.read(deleteRecentSearchProvider.future);
      final notifier = container.read(deleteRecentSearchProvider.notifier);
      await expectLater(notifier.delete('recent_1'), completes);
      expect(container.read(deleteRecentSearchProvider).error, same(failure));
      expect(container.read(deleteRecentSearchProvider).isLoading, isFalse);
      fail = false;
      await notifier.delete('recent_1');
      expect(container.read(deleteRecentSearchProvider).hasError, isFalse);
      expect(container.read(deleteRecentSearchProvider).isLoading, isFalse);
    },
  );

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
        overrides: [
          testMemberBrowsingOverride,
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          searchRepositoryProvider.overrideWithValue(fake),
        ],
      );

      await container.read(
        hashtagSearchProvider(
          const HashtagSearchQuery(tag: 'SockKAL'),
          audience: testMemberBrowsingAudience,
        ).future,
      );
      expect(saveCalls, 0);

      expect(
        await container.read(recentSearchPageProvider.future),
        isA<RecentSearchPage>(),
      );
      final result = await container
          .read(saveRecentSearchProvider.notifier)
          .save(
            const SaveRecentSearchRequest(
              type: RecentSearchType.hashtag,
              displayLabel: '#SockKAL',
              payload: HashtagRecentSearchPayload(tag: 'sockkal'),
            ),
          );
      expect(
        (await container.read(recentSearchPageProvider.future)).items,
        [saved],
      );
      await container
          .read(deleteRecentSearchProvider.notifier)
          .delete('recent_1');

      expect(result?.id, 'recent_1');
      expect(saveCalls, 1);
      expect(deletedId, 'recent_1');
      expect(listCalls, greaterThanOrEqualTo(2));
    },
  );
}
