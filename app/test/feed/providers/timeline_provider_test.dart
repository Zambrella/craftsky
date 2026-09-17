import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/pagination_contract.dart';
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
  'viewerHasSaved': false,
  'sponsored': false,
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

PaginationContractSubject<TimelineItem> _paginationSubject(
  PaginationContractFetch<TimelineItem> fetch,
) {
  final repository = FakePostRepository(
    onListTimeline: ({cursor, limit}) async {
      expect(limit, timelinePageLimit);
      final page = await fetch(cursor);
      return TimelinePage(items: page.items, cursor: page.cursor);
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
  final subscription = container.listen(timelineProvider, (_, _) {});

  return PaginationContractSubject(
    initialize: () async => container.read(timelineProvider.future),
    loadMore: () => container.read(timelineProvider.notifier).loadMore(),
    snapshot: () {
      final asyncState = container.read(timelineProvider);
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
    name: 'timeline',
    firstItem: _timelinePost(_samplePost(rkey: 'page-a')),
    secondItem: _timelinePost(_samplePost(rkey: 'page-b')),
    itemIdentity: (item) => item.itemKey,
    createSubject: _paginationSubject,
    deduplicatesAcrossPages: true,
  );

  group('timelineProvider build', () {
    test(
      'IT-019 reads the established Content policy before fetching',
      () async {
        var calls = 0;
        final fake = FakePostRepository(
          onListTimeline: ({cursor, limit}) async {
            calls++;
            return const TimelinePage(items: []);
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
        final subscription = container.listen(timelineProvider, (_, _) {});
        addTearDown(subscription.close);

        await container.read(timelineProvider.future);
        expect(calls, 1);
      },
    );

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

        await container.read(timelineProvider.future);

        container.read(timelineProvider.notifier)
          ..prepend(_samplePost(rkey: 'new'))
          ..prepend(old);

        final state = container.read(timelineProvider).value!;
        expect(state.items.map((item) => item.post.rkey), ['new', 'old']);
      },
    );

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
