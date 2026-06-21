import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/project_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

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
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
});

void main() {
  setUpAll(initializeMappers);

  test(
    'IT-012 project provider fetches initial state through repository',
    () async {
      String? seenQ;
      final fake = FakeSearchRepository(
        onSearchProjects: ({q, sort, filters, limit, cursor}) async {
          seenQ = q;
          return SearchPostPage(items: [_post('project')]);
        },
      );
      final container = ProviderContainer.test(
        overrides: [searchRepositoryProvider.overrideWithValue(fake)],
      );

      final state = await container.read(
        projectSearchProvider(const ProjectSearchQuery(q: 'cardigan')).future,
      );

      expect(seenQ, 'cardigan');
      expect(state.items.single.rkey.toString(), 'project');
    },
  );

  test(
    'IT-013 project loadMore passes cursor, appends, de-dupes, '
    'and no-ops at end',
    () async {
      var calls = 0;
      String? seenCursor;
      final fake = FakeSearchRepository(
        onSearchProjects: ({q, sort, filters, limit, cursor}) async {
          calls++;
          if (calls == 1) {
            return SearchPostPage(
              items: [_post('project')],
              cursor: 'opaque:projects',
            );
          }
          seenCursor = cursor;
          return SearchPostPage(
            items: [_post('project'), _post('project-next')],
          );
        },
      );
      final container = ProviderContainer.test(
        overrides: [searchRepositoryProvider.overrideWithValue(fake)],
      );
      final provider = projectSearchProvider(
        const ProjectSearchQuery(q: 'cardigan'),
      );

      await container.read(provider.future);
      await container.read(provider.notifier).loadMore();
      await container.read(provider.notifier).loadMore();

      final state = container.read(provider).value!;
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
