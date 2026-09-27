import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_reposts_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/pagination_contract.dart';
import '../fakes/fake_post_repository.dart';

final _aliceDid = Did.parse('did:plc:alice');

Post _post(String rkey) => PostMapper.fromMap({
  'uri': 'at://did:plc:bob/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 1,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasSaved': false,
  'sponsored': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:bob', 'handle': 'bob.craftsky.social'},
});

void main() {
  setUpAll(initializeMappers);

  paginationContract<Post>(
    name: 'user reposts',
    firstItem: _post('page-a'),
    secondItem: _post('page-b'),
    itemIdentity: (post) => post.rkey,
    createSubject: (fetch) {
      final repository = FakePostRepository(
        onListRepostsByAuthor: (id, {cursor, limit}) async {
          expect(id, _aliceDid);
          expect(limit, userRepostsPageLimit);
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
          postRepositoryProvider.overrideWithValue(repository),
        ],
      );
      final provider = userRepostsProvider(_aliceDid);
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
    },
  );
}
