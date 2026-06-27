import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:craftsky_app/search/providers/blank_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

void main() {
  test(
    'IT-013 blank search provider fetches recents and default craft top '
    'hashtags',
    () async {
      var recentCalls = 0;
      List<String>? seenCraftTypes;
      final fake = FakeSearchRepository(
        onListRecentSearches: () async {
          recentCalls++;
          return RecentSearchPage(
            items: [
              RecentSearchItem(
                id: 'recent_1',
                type: RecentSearchType.query,
                displayLabel: 'alpaca socks',
                payload: const QueryRecentSearchPayload(q: 'alpaca socks'),
                updatedAt: DateTime.parse('2026-06-20T10:00:00Z'),
              ),
            ],
          );
        },
        onTopHashtags: ({craftTypes, limit}) async {
          seenCraftTypes = craftTypes;
          return const TopHashtagsResponse(
            groups: [
              TopHashtagGroup(
                craftType: ProjectOptionCatalogs.knittingCraftToken,
                items: [TopHashtagItem(tag: 'sockkal', count: 12)],
              ),
              TopHashtagGroup(
                craftType: ProjectOptionCatalogs.crochetCraftToken,
                items: [],
              ),
            ],
          );
        },
      );
      final container = ProviderContainer.test(
        overrides: [searchRepositoryProvider.overrideWithValue(fake)],
      );

      final data = await container.read(blankSearchProvider.future);

      expect(recentCalls, 1);
      expect(seenCraftTypes, [
        for (final option in ProjectOptionCatalogs.craftTypes) option.value,
      ]);
      expect(data.recentSearches.items.single.displayLabel, 'alpaca socks');
      expect(
        data.topHashtags.groups.first.craftType,
        'social.craftsky.feed.defs#knitting',
      );
      expect(data.topHashtags.groups.first.items.single.tag, 'sockkal');
    },
  );
}
