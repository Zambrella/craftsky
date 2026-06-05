import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

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
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': did ?? 'did:plc:alice', 'handle': 'alice.craftsky.social'},
};

Post _samplePost({required String rkey, String? did}) =>
    PostMapper.fromMap(_samplePostMap(rkey: rkey, did: did));

void main() {
  setUpAll(initializeMappers);

  group('timelineProvider build', () {
    test('first build fetches page 1 and surfaces items + cursor', () async {
      int? seenLimit;
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          expect(cursor, isNull);
          seenLimit = limit;
          return PostPage(
            items: [
              _samplePost(rkey: 'a'),
              _samplePost(rkey: 'b'),
            ],
            cursor: 'next',
          );
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final state = await container.read(timelineProvider.future);

      expect(seenLimit, timelinePageLimit);
      expect(state.items.map((p) => p.rkey), ['a', 'b']);
      expect(state.cursor, 'next');
      expect(state.hasMore, isTrue);
    });
  });

  group('timelineProvider loadMore', () {
    test('passes opaque cursor and appends next page', () async {
      var call = 0;
      String? nextPageCursor;
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          call++;
          expect(limit, timelinePageLimit);
          if (call == 1) {
            return PostPage(
              items: [_samplePost(rkey: 'a')],
              cursor: 'opaque:abc',
            );
          }
          nextPageCursor = cursor;
          return PostPage(items: [_samplePost(rkey: 'b')]);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(timelineProvider.future);
      await container.read(timelineProvider.notifier).loadMore();

      final state = container.read(timelineProvider).value!;
      expect(nextPageCursor, 'opaque:abc');
      expect(state.items.map((p) => p.rkey), ['a', 'b']);
      expect(state.cursor, isNull);
      expect(state.hasMore, isFalse);
    });

    test('failure preserves visible items and cursor for retry', () async {
      var call = 0;
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
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
          expect(cursor, 'c1');
          return PostPage(items: [_samplePost(rkey: 'b')]);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(timelineProvider.future);
      await container.read(timelineProvider.notifier).loadMore();

      final mid = container.read(timelineProvider);
      expect(mid.hasError, isTrue);
      expect(mid.value?.items.map((p) => p.rkey), ['a']);
      expect(mid.value?.cursor, 'c1');

      await container.read(timelineProvider.notifier).loadMore();

      final after = container.read(timelineProvider).value!;
      expect(after.items.map((p) => p.rkey), ['a', 'b']);
    });

    test('no-op when hasMore is false', () async {
      var calls = 0;
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          calls++;
          return const PostPage(items: []);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(timelineProvider.future);
      await container.read(timelineProvider.notifier).loadMore();

      expect(calls, 1);
    });

    test('no-op when a previous loadMore is still in flight', () async {
      var calls = 0;
      final gate = Completer<PostPage>();
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          calls++;
          if (calls == 1) {
            return PostPage(
              items: [_samplePost(rkey: 'a')],
              cursor: 'c1',
            );
          }
          return gate.future;
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );
      final sub = container.listen(timelineProvider, (_, _) {});
      addTearDown(sub.close);

      await container.read(timelineProvider.future);
      final firstLoadMore = container
          .read(timelineProvider.notifier)
          .loadMore();
      await Future<void>.delayed(Duration.zero);
      await container.read(timelineProvider.notifier).loadMore();

      expect(calls, 2);

      gate.complete(PostPage(items: [_samplePost(rkey: 'b')]));
      await firstLoadMore;
    });
  });

  group('timelineProvider cache helpers', () {
    test(
      'prepend inserts top-level post at head and ignores duplicate URI',
      () async {
        final old = _samplePost(rkey: 'old');
        final fake = FakePostRepository(
          onListTimeline: ({cursor, limit}) async => PostPage(items: [old]),
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );

        await container.read(timelineProvider.future);

        container.read(timelineProvider.notifier)
          ..prepend(_samplePost(rkey: 'new'))
          ..prepend(old);

        final state = container.read(timelineProvider).value!;
        expect(state.items.map((p) => p.rkey), ['new', 'old']);
      },
    );

    test('loadMore merge dedupes fetched posts by URI', () async {
      var call = 0;
      final duplicate = _samplePost(rkey: 'a');
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          call++;
          if (call == 1) {
            return PostPage(items: [duplicate], cursor: 'c1');
          }
          return PostPage(
            items: [
              duplicate,
              _samplePost(rkey: 'b'),
            ],
          );
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(timelineProvider.future);
      await container.read(timelineProvider.notifier).loadMore();

      final state = container.read(timelineProvider).value!;
      expect(state.items.map((p) => p.rkey), ['a', 'b']);
    });

    test(
      'removeByUri removes matching post and ignores missing rows',
      () async {
        final a = _samplePost(rkey: 'a');
        final b = _samplePost(rkey: 'b');
        final fake = FakePostRepository(
          onListTimeline: ({cursor, limit}) async => PostPage(items: [a, b]),
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );

        await container.read(timelineProvider.future);
        container.read(timelineProvider.notifier)
          ..removeByUri(a.uri)
          ..removeByUri(a.uri);

        final state = container.read(timelineProvider).value!;
        expect(state.items.map((p) => p.rkey), ['b']);
      },
    );
  });
}
