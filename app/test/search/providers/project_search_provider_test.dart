import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/project_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

Post _post() => PostMapper.fromMap({
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/project',
  'cid': 'bafy_project',
  'rkey': 'project',
  'text': 'project',
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
          return SearchPostPage(items: [_post()]);
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
}
