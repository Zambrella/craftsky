import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../search/fakes/fake_search_repository.dart';
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
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
});

void main() {
  setUpAll(initializeMappers);

  test(
    'IT-014 project feed provider stays in project repository boundary',
    () async {
      var calls = 0;
      ProjectBrowseQuery? seenQuery;
      String? seenCursor;
      final fakeProjectRepository = FakeProjectRepository(
        onListProjects: ({required query, limit, cursor}) async {
          calls++;
          seenQuery = query;
          expect(limit, projectFeedPageLimit);
          if (calls == 1) {
            return PostPage(
              items: [_post('project')],
              cursor: 'opaque:projects',
            );
          }
          seenCursor = cursor;
          return PostPage(items: [_post('project'), _post('project-next')]);
        },
      );
      final container = ProviderContainer.test(
        overrides: [
          projectRepositoryProvider.overrideWithValue(fakeProjectRepository),
          searchRepositoryProvider.overrideWithValue(FakeSearchRepository()),
        ],
      );
      const query = ProjectBrowseQuery(
        craftTypes: [ProjectOptionCatalogs.knittingCraftToken],
        filters: ProjectBrowseFilters(
          material: ['alpaca'],
          projectTag: ['gift'],
        ),
        sort: SearchSort.popular,
      );
      final provider = projectFeedProvider(query);

      await container.read(provider.future);
      await container.read(provider.notifier).loadMore();
      await container.read(provider.notifier).loadMore();

      final state = container.read(provider).value!;
      expect(seenQuery, query);
      expect(seenCursor, 'opaque:projects');
      expect(state.items.map((post) => post.rkey.toString()), [
        'project',
        'project-next',
      ]);
      expect(state.hasMore, isFalse);
      expect(calls, 2);
    },
  );
}
