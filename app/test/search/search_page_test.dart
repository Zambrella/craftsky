import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_results_tab.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:craftsky_app/search/pages/search_page.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes/fake_search_repository.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('SearchPage renders blank discovery content', (tester) async {
    await tester.pumpWidget(_searchPageApp(home: const SearchPage()));
    await tester.pumpAndSettle();

    expect(find.text('Recent searches'), findsOneWidget);
    expect(find.text('linen dress'), findsOneWidget);
    expect(find.text('Trending hashtags'), findsOneWidget);
    expect(find.text('#memademay'), findsOneWidget);
  });

  testWidgets('SearchPage renders debounced profile and hashtag suggestions', (
    tester,
  ) async {
    await tester.pumpWidget(
      _searchPageApp(
        repository: FakeSearchRepository(
          onListRecentSearches: () async => const RecentSearchPage(items: []),
          onTopHashtags: ({craftTypes, limit}) async =>
              const TopHashtagsResponse(groups: []),
          onSearchSuggestions:
              ({required q, types, profileLimit, hashtagLimit}) async {
                expect(q, 'sock');
                return SearchSuggestions(
                  profiles: SearchSuggestionProfileSection(
                    hasMore: true,
                    items: [
                      ProfileSearchResult(
                        did: 'did:plc:alice',
                        handle: 'alice.craftsky.social',
                        displayName: 'Alice',
                        isCraftskyProfile: true,
                        viewerIsFollowing: false,
                        crafts: const [
                          ProjectOptionCatalogs.knittingCraftToken,
                        ],
                      ),
                    ],
                  ),
                  hashtags: const SearchSuggestionHashtagSection(
                    hasMore: true,
                    items: [
                      HashtagSearchResult(
                        tag: 'sockkal',
                        postsLast28Days: 12,
                      ),
                    ],
                  ),
                );
              },
        ),
        home: const SearchPage(),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const ValueKey('search-input')));
    await tester.enterText(find.byKey(const ValueKey('search-input')), 'sock');
    await tester.pump(const Duration(milliseconds: 350));
    await tester.pumpAndSettle();

    expect(find.text('Profiles'), findsWidgets);
    expect(find.text('@alice.craftsky.social'), findsOneWidget);
    expect(find.text('Alice • Knitting'), findsOneWidget);
    expect(find.text('Hashtags'), findsOneWidget);
    expect(find.text('#sockkal'), findsOneWidget);
    expect(find.text('12 posts'), findsOneWidget);
    expect(find.text('View all'), findsNWidgets(2));
  });

  testWidgets('SearchPage renders submitted result tabs and post results', (
    tester,
  ) async {
    await tester.pumpWidget(
      _searchPageApp(
        repository: FakeSearchRepository(
          onSearchPosts: ({required q, limit, cursor}) async {
            expect(q, 'alpaca');
            return SearchPostPage(items: [_post('result-a')]);
          },
          onSearchProjects: ({required q, limit, cursor}) async =>
              const SearchPostPage(items: []),
          onSearchProfiles: ({required q, limit, cursor}) async =>
              const ProfileSearchPage(items: []),
          onSearchHashtags: ({required q, limit, cursor}) async =>
              const HashtagSearchPage(items: []),
        ),
        home: const SearchPage(q: 'alpaca'),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Posts'), findsOneWidget);
    expect(find.text('Projects'), findsOneWidget);
    expect(find.text('Profiles'), findsOneWidget);
    expect(find.text('Tags'), findsOneWidget);
    expect(find.text('search result result-a'), findsOneWidget);
  });

  testWidgets('SearchPage renders submitted empty and error states', (
    tester,
  ) async {
    await tester.pumpWidget(
      _searchPageApp(
        repository: FakeSearchRepository(
          onSearchPosts: ({required q, limit, cursor}) async =>
              const SearchPostPage(items: []),
          onSearchProjects: ({required q, limit, cursor}) async =>
              const SearchPostPage(items: []),
          onSearchProfiles: ({required q, limit, cursor}) async =>
              const ProfileSearchPage(items: []),
          onSearchHashtags: ({required q, limit, cursor}) async =>
              Future<HashtagSearchPage>.error(StateError('boom')),
        ),
        home: const SearchPage(q: 'alpaca', tab: SearchResultsTab.tags),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text("Search didn't load."), findsOneWidget);
    expect(find.text('Retry'), findsOneWidget);

    await tester.pumpWidget(
      _searchPageApp(
        repository: FakeSearchRepository(
          onSearchPosts: ({required q, limit, cursor}) async =>
              const SearchPostPage(items: []),
          onSearchProjects: ({required q, limit, cursor}) async =>
              const SearchPostPage(items: []),
          onSearchProfiles: ({required q, limit, cursor}) async =>
              const ProfileSearchPage(items: []),
          onSearchHashtags: ({required q, limit, cursor}) async =>
              const HashtagSearchPage(items: []),
        ),
        home: const SearchPage(q: 'alpaca'),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('No posts found.'), findsOneWidget);
  });

  testWidgets('SearchPage triggers load-more from submitted post results', (
    tester,
  ) async {
    var postCalls = 0;
    String? seenCursor;
    await tester.pumpWidget(
      _searchPageApp(
        repository: FakeSearchRepository(
          onSearchPosts: ({required q, limit, cursor}) async {
            postCalls++;
            if (postCalls == 1) {
              return SearchPostPage(
                items: [_post('result-a'), _post('result-b')],
                cursor: 'opaque:posts',
              );
            }
            seenCursor = cursor;
            return SearchPostPage(items: [_post('result-c')]);
          },
          onSearchProjects: ({required q, limit, cursor}) async =>
              const SearchPostPage(items: []),
          onSearchProfiles: ({required q, limit, cursor}) async =>
              const ProfileSearchPage(items: []),
          onSearchHashtags: ({required q, limit, cursor}) async =>
              const HashtagSearchPage(items: []),
        ),
        home: const SearchPage(q: 'alpaca'),
      ),
    );
    await tester.pumpAndSettle();
    await tester.pumpAndSettle();

    expect(seenCursor, 'opaque:posts');
    expect(find.text('search result result-a'), findsOneWidget);
    expect(find.text('search result result-b'), findsOneWidget);
    expect(find.text('search result result-c'), findsOneWidget);
  });

  test('SearchRoute preserves submitted query context', () {
    expect(const SearchRoute(q: 'sock').location, '/search?q=sock');
  });

  test('SearchRoute preserves tab context', () {
    expect(
      const SearchRoute(q: 'sock', tab: SearchResultsTab.profiles).location,
      '/search?q=sock&tab=profiles',
    );
  });

  test('TagSearchRoute preserves tag query context', () {
    expect(
      const TagSearchRoute(tag: 'SockKAL').location,
      '/search/tags?tag=SockKAL',
    );
  });
}

Widget _searchPageApp({
  required Widget home,
  FakeSearchRepository? repository,
}) {
  return ProviderScope(
    overrides: [
      searchRepositoryProvider.overrideWithValue(
        repository ?? _blankSearchRepository(),
      ),
    ],
    child: _LocalizedApp(home: home),
  );
}

FakeSearchRepository _blankSearchRepository() {
  return FakeSearchRepository(
    onListRecentSearches: () async => RecentSearchPage(
      items: [
        RecentSearchItem(
          id: 'recent-1',
          type: RecentSearchType.query,
          displayLabel: 'linen dress',
          payload: const QueryRecentSearchPayload(q: 'linen dress'),
          updatedAt: DateTime.utc(2026),
        ),
      ],
    ),
    onTopHashtags: ({craftTypes, limit}) async => const TopHashtagsResponse(
      groups: [
        TopHashtagGroup(
          craftType: ProjectOptionCatalogs.sewingCraftToken,
          items: [TopHashtagItem(tag: 'memademay', count: 12)],
        ),
      ],
    ),
  );
}

Post _post(String rkey) => PostMapper.fromMap({
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'search result $rkey',
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasSaved': false,
  'viewerHasReplied': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
});

class _LocalizedApp extends StatelessWidget {
  const _LocalizedApp({required this.home});

  final Widget home;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: home,
    );
  }
}
