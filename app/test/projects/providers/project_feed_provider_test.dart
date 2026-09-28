import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../search/fakes/fake_search_repository.dart';
import '../../test_support/pagination_contract.dart';
import '../fakes/fake_project_repository.dart';

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
  'viewerHasSaved': false,
  'sponsored': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
});

const _query = ProjectBrowseQuery(
  craftTypes: [ProjectOptionCatalogs.knittingCraftToken],
  filters: ProjectBrowseFilters(
    yarnWeight: ['social.craftsky.project.defs#fingering'],
    selfDrafted: true,
  ),
  sort: SearchSort.popular,
);

PaginationContractSubject<Post> _paginationSubject(
  PaginationContractFetch<Post> fetch,
) {
  final repository = FakeProjectRepository(
    onListProjects: ({required query, limit, cursor}) async {
      expect(query, _query);
      expect(limit, projectFeedPageLimit);
      final page = await fetch(cursor);
      return PostPage(items: page.items, cursor: page.cursor);
    },
  );
  final container = ProviderContainer.test(
    overrides: [
      activeLanguagePreferencesProvider.overrideWith(
        (ref) => const LanguagePreferences(
          primaryLanguage: 'en',
          contentLanguages: ['en'],
        ),
      ),
      projectRepositoryProvider.overrideWithValue(repository),
      searchRepositoryProvider.overrideWithValue(FakeSearchRepository()),
    ],
  );
  final provider = projectFeedProvider(_query);
  final subscription = container.listen(provider, (_, _) {});

  return PaginationContractSubject(
    initialize: () async => container.read(provider.future),
    loadMore: () => container.read(provider.notifier).loadMore(),
    snapshot: () {
      final asyncState = container.read(provider);
      final state = asyncState.value!;
      return PaginationContractSnapshot(
        items: state.items,
        cursor: state.cursor,
        hasMore: state.hasMore,
        hasError: asyncState.hasError,
      );
    },
    dispose: () {
      subscription.close();
      container.dispose();
    },
  );
}

void main() {
  setUpAll(initializeMappers);

  paginationContract(
    name: 'project feed',
    firstItem: _post('project-a'),
    secondItem: _post('project-b'),
    itemIdentity: (post) => post.rkey,
    createSubject: _paginationSubject,
    deduplicatesAcrossPages: true,
  );

  test(
    'IT-014 project feed provider stays in project repository boundary',
    () async {
      ProjectBrowseQuery? seenQuery;
      final fakeProjectRepository = FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) async {
          seenQuery = query;
          expect(limit, projectFeedPageLimit);
          return PostPage(items: [_post('project')]);
        },
      );
      final container = ProviderContainer.test(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          projectRepositoryProvider.overrideWithValue(fakeProjectRepository),
          searchRepositoryProvider.overrideWithValue(FakeSearchRepository()),
        ],
      );
      final provider = projectFeedProvider(_query);
      final subscription = container.listen(provider, (_, _) {});
      addTearDown(subscription.close);

      await container.read(provider.future);

      expect(seenQuery, _query);
    },
  );
}
