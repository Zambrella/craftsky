import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
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

TimelineItem _timelinePost(Post post, {String? itemKey}) => TimelineItem(
  itemKey: itemKey ?? 'post:${post.uri}',
  post: post,
);

TimelineItem _repostItem({
  required String itemKey,
  required Post post,
  required String reposterDid,
  required String reposterHandle,
}) => TimelineItem(
  itemKey: itemKey,
  post: post,
  reason: RepostReason(
    type: RepostReasonType.repost,
    by: PostAuthor(did: reposterDid, handle: reposterHandle),
    uri:
        'at://$reposterDid/social.craftsky.feed.repost/${itemKey.split(':').last}',
    cid: 'bafy_repost_${itemKey.split(':').last}',
    createdAt: DateTime.parse('2026-05-04T18:24:00.000Z'),
    indexedAt: DateTime.parse('2026-05-04T18:24:01.000Z'),
  ),
);

void main() {
  setUpAll(initializeMappers);

  group('timelineProvider build', () {
    test('first build fetches page 1 and surfaces items + cursor', () async {
      int? seenLimit;
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          expect(cursor, isNull);
          seenLimit = limit;
          return TimelinePage(
            items: [
              _timelinePost(_samplePost(rkey: 'a')),
              _timelinePost(_samplePost(rkey: 'b')),
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
      expect(state.items.map((item) => item.post.rkey), ['a', 'b']);
      expect(state.cursor, 'next');
      expect(state.hasMore, isTrue);
    });

    test(
      'keeps duplicate repost feed items for the same post by itemKey',
      () async {
        final original = _samplePost(rkey: 'shared', did: 'did:plc:carol');
        final fake = FakePostRepository(
          onListTimeline: ({cursor, limit}) async => TimelinePage(
            items: [
              _repostItem(
                itemKey:
                    'repost:at://did:plc:bob/social.craftsky.feed.repost/r1',
                post: original,
                reposterDid: 'did:plc:bob',
                reposterHandle: 'bob.craftsky.social',
              ),
              _repostItem(
                itemKey:
                    'repost:at://did:plc:dana/social.craftsky.feed.repost/r2',
                post: original,
                reposterDid: 'did:plc:dana',
                reposterHandle: 'dana.craftsky.social',
              ),
            ],
          ),
        );

        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );

        final state = await container.read(timelineProvider.future);

        expect(state.items, hasLength(2));
        expect(state.items.map((item) => item.itemKey), [
          'repost:at://did:plc:bob/social.craftsky.feed.repost/r1',
          'repost:at://did:plc:dana/social.craftsky.feed.repost/r2',
        ]);
        expect(state.items.map((item) => item.post.uri), [
          original.uri,
          original.uri,
        ]);
      },
    );
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
            return TimelinePage(
              items: [_timelinePost(_samplePost(rkey: 'a'))],
              cursor: 'opaque:abc',
            );
          }
          nextPageCursor = cursor;
          return TimelinePage(items: [_timelinePost(_samplePost(rkey: 'b'))]);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(timelineProvider.future);
      await container.read(timelineProvider.notifier).loadMore();

      final state = container.read(timelineProvider).value!;
      expect(nextPageCursor, 'opaque:abc');
      expect(state.items.map((item) => item.post.rkey), ['a', 'b']);
      expect(state.cursor, isNull);
      expect(state.hasMore, isFalse);
    });

    test('failure preserves visible items and cursor for retry', () async {
      var call = 0;
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          call++;
          if (call == 1) {
            return TimelinePage(
              items: [_timelinePost(_samplePost(rkey: 'a'))],
              cursor: 'c1',
            );
          }
          if (call == 2) {
            throw Exception('network down');
          }
          expect(cursor, 'c1');
          return TimelinePage(items: [_timelinePost(_samplePost(rkey: 'b'))]);
        },
      );

      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(timelineProvider.future);
      await container.read(timelineProvider.notifier).loadMore();

      final mid = container.read(timelineProvider);
      expect(mid.hasError, isTrue);
      expect(mid.value?.items.map((item) => item.post.rkey), ['a']);
      expect(mid.value?.cursor, 'c1');

      await container.read(timelineProvider.notifier).loadMore();

      final after = container.read(timelineProvider).value!;
      expect(after.items.map((item) => item.post.rkey), ['a', 'b']);
    });

    test('no-op when hasMore is false', () async {
      var calls = 0;
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          calls++;
          return const TimelinePage(items: []);
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
      final gate = Completer<TimelinePage>();
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          calls++;
          if (calls == 1) {
            return TimelinePage(
              items: [_timelinePost(_samplePost(rkey: 'a'))],
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

      gate.complete(
        TimelinePage(items: [_timelinePost(_samplePost(rkey: 'b'))]),
      );
      await firstLoadMore;
    });
  });

  group('timelineProvider cache helpers', () {
    test(
      'prepend inserts top-level post at head and ignores duplicate URI',
      () async {
        final old = _samplePost(rkey: 'old');
        final fake = FakePostRepository(
          onListTimeline: ({cursor, limit}) async =>
              TimelinePage(items: [_timelinePost(old)]),
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );

        await container.read(timelineProvider.future);

        container.read(timelineProvider.notifier)
          ..prepend(_samplePost(rkey: 'new'))
          ..prepend(old);

        final state = container.read(timelineProvider).value!;
        expect(state.items.map((item) => item.post.rkey), ['new', 'old']);
      },
    );

    test('loadMore merge dedupes fetched posts by URI', () async {
      var call = 0;
      final duplicate = _samplePost(rkey: 'a');
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          call++;
          if (call == 1) {
            return TimelinePage(
              items: [_timelinePost(duplicate)],
              cursor: 'c1',
            );
          }
          return TimelinePage(
            items: [
              _timelinePost(duplicate),
              _timelinePost(_samplePost(rkey: 'b')),
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
      expect(state.items.map((item) => item.post.rkey), ['a', 'b']);
    });

    test(
      'removeByUri removes matching post and ignores missing rows',
      () async {
        final a = _samplePost(rkey: 'a');
        final b = _samplePost(rkey: 'b');
        final fake = FakePostRepository(
          onListTimeline: ({cursor, limit}) async =>
              TimelinePage(items: [_timelinePost(a), _timelinePost(b)]),
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );

        await container.read(timelineProvider.future);
        container.read(timelineProvider.notifier)
          ..removeByUri(a.uri)
          ..removeByUri(a.uri);

        final state = container.read(timelineProvider).value!;
        expect(state.items.map((item) => item.post.rkey), ['b']);
      },
    );

    test(
      'replace patches every timeline item with a matching post URI',
      () async {
        final original = _samplePost(rkey: 'shared', did: 'did:plc:carol');
        final fake = FakePostRepository(
          onListTimeline: ({cursor, limit}) async => TimelinePage(
            items: [
              _repostItem(
                itemKey:
                    'repost:at://did:plc:bob/social.craftsky.feed.repost/r1',
                post: original,
                reposterDid: 'did:plc:bob',
                reposterHandle: 'bob.craftsky.social',
              ),
              _repostItem(
                itemKey:
                    'repost:at://did:plc:dana/social.craftsky.feed.repost/r2',
                post: original,
                reposterDid: 'did:plc:dana',
                reposterHandle: 'dana.craftsky.social',
              ),
            ],
          ),
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );

        await container.read(timelineProvider.future);
        container
            .read(timelineProvider.notifier)
            .replace(
              original.copyWith(repostCount: 7, viewerHasReposted: true),
            );

        final state = container.read(timelineProvider).value!;
        expect(state.items, hasLength(2));
        expect(state.items.map((item) => item.post.repostCount), [7, 7]);
        expect(state.items.map((item) => item.post.viewerHasReposted), [
          true,
          true,
        ]);
      },
    );

    test(
      'IT-009 suppressActor removes authored and repost-attributed rows',
      () async {
        final bobPost = _samplePost(rkey: 'bob', did: 'did:plc:bob');
        final carolPost = _samplePost(rkey: 'carol', did: 'did:plc:carol');
        final fake = FakePostRepository(
          onListTimeline: ({cursor, limit}) async => TimelinePage(
            items: [
              _timelinePost(bobPost),
              _repostItem(
                itemKey:
                    'repost:at://did:plc:bob/social.craftsky.feed.repost/r1',
                post: carolPost,
                reposterDid: 'did:plc:bob',
                reposterHandle: 'bob.craftsky.social',
              ),
              _timelinePost(carolPost),
            ],
          ),
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );

        await container.read(timelineProvider.future);
        container.read(timelineProvider.notifier).suppressActor('did:plc:bob');

        expect(
          container
              .read(timelineProvider)
              .requireValue
              .items
              .map((item) => item.itemKey),
          ['post:${carolPost.uri}'],
        );
      },
    );
  });
}
