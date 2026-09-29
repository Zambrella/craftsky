import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/post_search_provider.dart';
import 'package:craftsky_app/search/providers/project_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/pagination_contract.dart';
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
  'viewerHasSaved': false,
  'sponsored': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
});

List<dynamic> _overrides(FakeSearchRepository repository) => [
  activeLanguagePreferencesProvider.overrideWith(
    (ref) => const LanguagePreferences(
      primaryLanguage: 'en',
      contentLanguages: ['en'],
    ),
  ),
  searchRepositoryProvider.overrideWithValue(repository),
];

PaginationContractSubject<Post> _postSearchSubject(
  PaginationContractFetch<Post> fetch,
) {
  final repository = FakeSearchRepository(
    onSearchPosts: ({required q, limit, cursor}) async {
      expect(q, 'alpaca');
      expect(limit, searchResultsPageLimit);
      final page = await fetch(cursor);
      return SearchPostPage(items: page.items, cursor: page.cursor);
    },
  );
  final container = ProviderContainer.test(
    overrides: List.from(_overrides(repository)),
  );
  final provider = postSearchProvider(const PostSearchQuery(q: 'alpaca'));
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

PaginationContractSubject<Post> _projectSearchSubject(
  PaginationContractFetch<Post> fetch,
) {
  final repository = FakeSearchRepository(
    onSearchProjects: ({required q, limit, cursor}) async {
      expect(q, 'cardigan');
      expect(limit, searchResultsPageLimit);
      final page = await fetch(cursor);
      return SearchPostPage(items: page.items, cursor: page.cursor);
    },
  );
  final container = ProviderContainer.test(
    overrides: List.from(_overrides(repository)),
  );
  final provider = projectSearchProvider(
    const ProjectSearchQuery(q: 'cardigan'),
  );
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
    name: 'post search',
    firstItem: _post('post-a'),
    secondItem: _post('post-b'),
    itemIdentity: (post) => post.rkey,
    createSubject: _postSearchSubject,
    deduplicatesAcrossPages: true,
  );

  paginationContract(
    name: 'project search',
    firstItem: _post('project-a'),
    secondItem: _post('project-b'),
    itemIdentity: (post) => post.rkey,
    createSubject: _projectSearchSubject,
    deduplicatesAcrossPages: true,
  );
}
