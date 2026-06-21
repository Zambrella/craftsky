import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/search/providers/search_suggestions_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

void main() {
  test(
    'IT-013 suggestions provider short-circuits blank input and delegates '
    'non-blank queries',
    () async {
      var calls = 0;
      String? seenQ;
      List<SearchSuggestionType>? seenTypes;
      int? seenProfileLimit;
      int? seenHashtagLimit;
      final fake = FakeSearchRepository(
        onSearchSuggestions:
            ({required q, types, profileLimit, hashtagLimit}) async {
              calls++;
              seenQ = q;
              seenTypes = types;
              seenProfileLimit = profileLimit;
              seenHashtagLimit = hashtagLimit;
              return SearchSuggestions(
                profiles: SearchSuggestionProfileSection(
                  items: [
                    ProfileSearchResult(
                      did: 'did:plc:alice',
                      handle: 'alice.craftsky.social',
                      isCraftskyProfile: true,
                      viewerIsFollowing: true,
                      crafts: const ['social.craftsky.feed.defs#knitting'],
                    ),
                  ],
                  hasMore: true,
                ),
                hashtags: const SearchSuggestionHashtagSection(
                  items: [
                    HashtagSearchResult(tag: 'sockkal', postsLast28Days: 7),
                  ],
                  hasMore: false,
                ),
              );
            },
      );
      final container = ProviderContainer.test(
        overrides: [searchRepositoryProvider.overrideWithValue(fake)],
      );

      final blank = await container.read(
        searchSuggestionsProvider(const SearchSuggestionQuery(q: '   ')).future,
      );
      final suggestions = await container.read(
        searchSuggestionsProvider(
          const SearchSuggestionQuery(
            q: '  sock  ',
            types: [
              SearchSuggestionType.profiles,
              SearchSuggestionType.hashtags,
            ],
            profileLimit: 2,
            hashtagLimit: 3,
          ),
        ).future,
      );

      expect(blank.profiles.items, isEmpty);
      expect(blank.hashtags.items, isEmpty);
      expect(calls, 1);
      expect(seenQ, 'sock');
      expect(seenTypes, [
        SearchSuggestionType.profiles,
        SearchSuggestionType.hashtags,
      ]);
      expect(seenProfileLimit, 2);
      expect(seenHashtagLimit, 3);
      expect(suggestions.profiles.items.single.crafts, [
        'social.craftsky.feed.defs#knitting',
      ]);
      expect(suggestions.hashtags.items.single.tag, 'sockkal');
    },
  );
}
