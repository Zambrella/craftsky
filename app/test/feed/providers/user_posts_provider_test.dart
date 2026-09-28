import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/pagination_contract.dart';
import '../fakes/fake_post_repository.dart';

final _aliceDid = Did.parse('did:plc:alice');

Map<String, dynamic> _samplePostMap({required String rkey, String? did}) => {
  'uri': 'at://${did ?? 'did:plc:alice'}/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
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
  'author': {'did': did ?? 'did:plc:alice', 'handle': 'alice.craftsky.social'},
};

Post _samplePost({required String rkey, String? did}) =>
    PostMapper.fromMap(_samplePostMap(rkey: rkey, did: did));

PaginationContractSubject<Post> _paginationSubject(
  PaginationContractFetch<Post> fetch,
) {
  final repository = FakePostRepository(
    onListByAuthor: (id, {cursor, limit}) async {
      expect(id, _aliceDid);
      expect(limit, userPostsPageLimit);
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
  final provider = userPostsProvider(_aliceDid);
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
    name: 'user posts',
    firstItem: _samplePost(rkey: 'page-a'),
    secondItem: _samplePost(rkey: 'page-b'),
    itemIdentity: (post) => post.rkey,
    createSubject: _paginationSubject,
  );

  group('userPostsProvider build', () {
    test('first build with empty page yields hasMore == false', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            const PostPage(items: []),
      );

      final container = ProviderContainer.test(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          postRepositoryProvider.overrideWithValue(fake),
        ],
      );

      final state = await container.read(
        userPostsProvider(_aliceDid).future,
      );
      expect(state.items, isEmpty);
      expect(state.cursor, isNull);
      expect(state.hasMore, isFalse);
    });
  });

  group('userPostsProvider loadMore', () {
    test(
      'UT-004 discards an invalid traversal and restarts page one once',
      () async {
        var calls = 0;
        final fake = FakePostRepository(
          onListByAuthor: (id, {cursor, limit}) async {
            calls++;
            expect(limit, userPostsPageLimit);
            return switch (calls) {
              1 => PostPage(
                items: [_samplePost(rkey: 'old-pin')],
                cursor: 'stale-cursor',
                pinnedPostUri:
                    'at://did:plc:alice/social.craftsky.feed.post/old-pin',
              ),
              2 => throw const ApiBadRequest('invalid_cursor'),
              3 => PostPage(
                items: [_samplePost(rkey: 'new-pin')],
                cursor: 'fresh-cursor',
                pinnedPostUri:
                    'at://did:plc:alice/social.craftsky.feed.post/new-pin',
              ),
              _ => throw StateError('unexpected request $calls'),
            };
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
            postRepositoryProvider.overrideWithValue(fake),
          ],
        );
        final provider = userPostsProvider(_aliceDid);
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);

        await container.read(provider.future);
        await container.read(provider.notifier).loadMore();

        final state = container.read(provider).requireValue;
        expect(calls, 3);
        expect(state.items.map((post) => post.rkey), ['new-pin']);
        expect(state.cursor, 'fresh-cursor');
        expect(
          state.pinnedPostUri,
          'at://did:plc:alice/social.craftsky.feed.post/new-pin',
        );
      },
    );
  });
}
