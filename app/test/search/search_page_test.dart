import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_results_tab.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:craftsky_app/search/pages/search_page.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes/fake_search_repository.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('SearchPage renders blank discovery content', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          searchRepositoryProvider.overrideWithValue(
            FakeSearchRepository(
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
              onTopHashtags: ({craftTypes, limit}) async =>
                  const TopHashtagsResponse(
                    groups: [
                      TopHashtagGroup(
                        craftType: ProjectOptionCatalogs.sewingCraftToken,
                        items: [TopHashtagItem(tag: 'memademay', count: 12)],
                      ),
                    ],
                  ),
            ),
          ),
        ],
        child: const _LocalizedApp(home: SearchPage()),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Recent searches'), findsOneWidget);
    expect(find.text('linen dress'), findsOneWidget);
    expect(find.text('Trending hashtags'), findsOneWidget);
    expect(find.text('#memademay'), findsOneWidget);
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

class _LocalizedApp extends StatelessWidget {
  const _LocalizedApp({required this.home});

  final Widget home;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: home,
    );
  }
}
