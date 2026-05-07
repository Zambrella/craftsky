import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

Map<String, dynamic> _samplePostMap({required String rkey, String? did}) => {
  'uri': 'at://${did ?? 'did:plc:alice'}/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
  'tags': <String>[],
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {
    'did': did ?? 'did:plc:alice',
    'handle': 'alice.craftsky.social',
  },
};

Post _samplePost({required String rkey, String? did}) =>
    PostMapper.fromMap(_samplePostMap(rkey: rkey, did: did));

void main() {
  setUpAll(initializeMappers);

  group('userPostsProvider build', () {
    test('first build fetches page 1 and surfaces items + cursor', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(
          items: [
            _samplePost(rkey: 'a'),
            _samplePost(rkey: 'b'),
          ],
          cursor: 'next',
        ),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final state = await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );
      expect(state.items.map((p) => p.rkey), ['a', 'b']);
      expect(state.cursor, 'next');
      expect(state.hasMore, isTrue);
    });

    test('first build with empty page yields hasMore == false', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            const PostPage(items: []),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final state = await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );
      expect(state.items, isEmpty);
      expect(state.cursor, isNull);
      expect(state.hasMore, isFalse);
    });
  });

  group('userPostsProvider loadMore', () {
    test('appends next page and advances cursor', () async {
      var call = 0;
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async {
          call++;
          if (call == 1) {
            return PostPage(
              items: [_samplePost(rkey: 'a')],
              cursor: 'c1',
            );
          }
          expect(cursor, 'c1');
          return PostPage(items: [_samplePost(rkey: 'b')]);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      // First build to populate the state.
      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      await container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .loadMore();

      final state = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(state.items.map((p) => p.rkey), ['a', 'b']);
      expect(state.cursor, isNull);
      expect(state.hasMore, isFalse);
    });

    test('no-op when hasMore is false', () async {
      var calls = 0;
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async {
          calls++;
          return const PostPage(items: []);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );
      expect(calls, 1);

      await container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .loadMore();
      expect(calls, 1, reason: 'loadMore must not call repo when !hasMore');
    });

    test('failure preserves visible items and cursor for retry', () async {
      var call = 0;
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async {
          call++;
          if (call == 1) {
            return PostPage(
              items: [_samplePost(rkey: 'a')],
              cursor: 'c1',
            );
          }
          if (call == 2) {
            throw Exception('network down');
          }
          // Retry succeeds with the same cursor.
          expect(cursor, 'c1');
          return PostPage(items: [_samplePost(rkey: 'b')]);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      // First loadMore fails.
      await container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .loadMore();

      final mid = container.read(userPostsProvider('alice.craftsky.social'));
      expect(mid.hasError, isTrue, reason: 'state is AsyncError after failure');
      expect(
        mid.value?.items.map((p) => p.rkey),
        ['a'],
        reason: 'previous data preserved via copyWithPrevious',
      );
      expect(mid.value?.cursor, 'c1', reason: 'cursor unchanged on failure');

      // Retry uses the same cursor and succeeds.
      await container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .loadMore();

      final after = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(after.items.map((p) => p.rkey), ['a', 'b']);
    });
  });

  group('userPostsProvider prepend', () {
    test('inserts a new post at the head', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_samplePost(rkey: 'a')]),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .prepend(_samplePost(rkey: 'new'));

      final state = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(state.items.map((p) => p.rkey), ['new', 'a']);
    });

    test('dedupes by uri', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_samplePost(rkey: 'a')]),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      // Same uri as 'a' — must not double-insert.
      container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .prepend(_samplePost(rkey: 'a'));

      final state = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(state.items.map((p) => p.rkey), ['a']);
    });
  });

  group('userPostsProvider removeByRkey', () {
    test('filters the matching post out of the list', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(
          items: [
            _samplePost(rkey: 'a'),
            _samplePost(rkey: 'b'),
            _samplePost(rkey: 'c'),
          ],
        ),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .removeByRkey('b');

      final state = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(state.items.map((p) => p.rkey), ['a', 'c']);
    });

    test('no-op when rkey not present', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_samplePost(rkey: 'a')]),
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      container
          .read(userPostsProvider('alice.craftsky.social').notifier)
          .removeByRkey('not-here');

      final state = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(state.items.map((p) => p.rkey), ['a']);
    });
  });
}
