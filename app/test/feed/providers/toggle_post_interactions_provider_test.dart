import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

final _aliceDid = Did.parse('did:plc:alice');

Map<String, dynamic> _postMap({
  required String rkey,
  int likeCount = 0,
  int repostCount = 0,
  bool viewerHasLiked = false,
  bool viewerHasReposted = false,
  Project? project,
}) => {
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
  'tags': <String>[],
  'likeCount': likeCount,
  'repostCount': repostCount,
  'replyCount': 0,
  'viewerHasLiked': viewerHasLiked,
  'viewerHasReposted': viewerHasReposted,
  'viewerHasSaved': false,
  'sponsored': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
  if (project != null) 'project': project.toMap(),
};

Post _post({
  required String rkey,
  int likeCount = 0,
  int repostCount = 0,
  bool viewerHasLiked = false,
  bool viewerHasReposted = false,
  Project? project,
}) => PostMapper.fromMap(
  _postMap(
    rkey: rkey,
    likeCount: likeCount,
    repostCount: repostCount,
    viewerHasLiked: viewerHasLiked,
    viewerHasReposted: viewerHasReposted,
    project: project,
  ),
);

TimelinePage _timelinePage(List<Post> posts) => TimelinePage(
  items: [
    for (final post in posts)
      TimelineItem(itemKey: 'post:${post.uri}', post: post),
  ],
);

const _project = Project(
  common: ProjectCommon(craftType: 'social.craftsky.feed.defs#embroidery'),
);

InteractionWriteResponse _interaction(Post post) => InteractionWriteResponse(
  uri: 'at://did:plc:viewer/social.craftsky.feed.like/like1',
  cid: 'bafy_like',
  rkey: 'like1',
  subject: PostRef(uri: post.uri, cid: post.cid),
  createdAt: DateTime.parse('2026-05-04T18:25:00.000Z'),
);

void main() {
  setUpAll(initializeMappers);

  test(
    'REG-001 toggle providers keep writes separate from interaction lists',
    () async {
      final post = _post(rkey: 'a');
      var likes = 0;
      var reposts = 0;
      var listCalls = 0;
      final fake = FakePostRepository(
        onLike: (did, rkey) async {
          likes++;
          return _interaction(post);
        },
        onRepost: (did, rkey) async {
          reposts++;
          return _interaction(post);
        },
        onListLikes: (did, rkey, {cursor, limit}) async {
          listCalls++;
          throw StateError('like list must not replace mutation');
        },
        onListReposts: (did, rkey, {cursor, limit}) async {
          listCalls++;
          throw StateError('repost list must not replace mutation');
        },
        onListQuotes: (did, rkey, {cursor, limit}) async {
          listCalls++;
          throw StateError('quote list must not replace mutation');
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(toggleLikePostProvider.notifier).toggle(post: post);
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: post);

      expect(likes, 1);
      expect(reposts, 1);
      expect(listCalls, 0);
    },
  );

  group('ToggleLikePost', () {
    test('IT-016 retries ambiguity with one canonical operation key', () async {
      final post = _post(rkey: 'retry');
      var calls = 0;
      final fake = FakePostRepository(
        onLike: (did, rkey) async {
          calls++;
          if (calls < 3) {
            throw const PdsMutationAmbiguousException(retryAfterSeconds: 2);
          }
          return _interaction(post);
        },
      );
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(fake),
          pdsMutationDelayProvider.overrideWithValue((_) async {}),
          pdsMutationJitterProvider.overrideWithValue((_) => 0),
        ],
      );

      await container.read(toggleLikePostProvider.notifier).toggle(post: post);

      expect(calls, 3);
      expect(fake.likeOperationKeys.toSet(), hasLength(1));
      expect(
        isCanonicalPdsMutationOperationKey(fake.likeOperationKeys.first),
        isTrue,
      );
      expect(
        container.read(toggleLikePostProvider).value?.viewerHasLiked,
        isTrue,
      );
    });

    test('IT-016 exhaustion remains actionable with the same key', () async {
      final post = _post(rkey: 'retry-limit');
      var calls = 0;
      final fake = FakePostRepository(
        onLike: (did, rkey) async {
          calls++;
          if (calls <= 7) {
            throw const PdsMutationAmbiguousException(retryAfterSeconds: 1);
          }
          return _interaction(post);
        },
      );
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(fake),
          pdsMutationDelayProvider.overrideWithValue((_) async {}),
          pdsMutationJitterProvider.overrideWithValue((_) => 0),
        ],
      );

      await container.read(toggleLikePostProvider.notifier).toggle(post: post);

      expect(calls, 7);
      expect(fake.likeOperationKeys.toSet(), hasLength(1));
      expect(container.read(toggleLikePostProvider).hasError, isTrue);

      await container.read(toggleLikePostProvider.notifier).toggle(post: post);

      expect(calls, 8);
      expect(fake.likeOperationKeys.toSet(), hasLength(1));
      expect(
        container.read(toggleLikePostProvider).value?.viewerHasLiked,
        isTrue,
      );
    });

    test('IT-016 masks stale reads and retires on logical agreement', () async {
      var authoritativeLiked = false;
      final original = _post(rkey: 'overlay', likeCount: 2);
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(
          items: [
            _post(
              rkey: 'overlay',
              likeCount: authoritativeLiked ? 3 : 2,
              viewerHasLiked: authoritativeLiked,
            ),
          ],
        ),
        onLike: (did, rkey) async => _interaction(original),
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

      await container.read(userPostsProvider(_aliceDid).future);
      await container
          .read(toggleLikePostProvider.notifier)
          .toggle(post: original);
      final staleRead = await container.read(
        userPostsProvider(_aliceDid).future,
      );
      expect(staleRead.items.single.viewerHasLiked, isTrue);
      expect(staleRead.items.single.likeCount, 3);

      authoritativeLiked = true;
      container.invalidate(userPostsProvider(_aliceDid));
      await container.read(userPostsProvider(_aliceDid).future);
      authoritativeLiked = false;
      container.invalidate(userPostsProvider(_aliceDid));
      final afterAgreement = await container.read(
        userPostsProvider(_aliceDid).future,
      );
      expect(afterAgreement.items.single.viewerHasLiked, isFalse);
      expect(afterAgreement.items.single.likeCount, 2);
    });

    test('accepted overlay updates user post lists', () async {
      final post = _post(rkey: 'a', likeCount: 2);
      final calls = <(String, String)>[];
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(items: [post]),
        onLike: (did, rkey) async {
          calls.add((did, rkey));
          return _interaction(post);
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

      await container.read(userPostsProvider(_aliceDid).future);
      await container.read(toggleLikePostProvider.notifier).toggle(post: post);
      await container.read(userPostsProvider(_aliceDid).future);

      final handleUpdated = container
          .read(userPostsProvider(_aliceDid))
          .value!
          .items
          .single;
      final didUpdated = container
          .read(userPostsProvider(_aliceDid))
          .value!
          .items
          .single;
      expect(calls, [('did:plc:alice', 'a')]);
      expect(handleUpdated.viewerHasLiked, isTrue);
      expect(handleUpdated.likeCount, 3);
      expect(didUpdated.viewerHasLiked, isTrue);
      expect(didUpdated.likeCount, 3);
    });

    test('unlikes without decrementing below zero', () async {
      final post = _post(rkey: 'a', viewerHasLiked: true);
      final calls = <(String, String)>[];
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(items: [post]),
        onUnlike: (did, rkey) async => calls.add((did, rkey)),
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

      await container.read(userPostsProvider(_aliceDid).future);
      await container.read(toggleLikePostProvider.notifier).toggle(post: post);
      await container.read(userPostsProvider(_aliceDid).future);

      final updated = container
          .read(userPostsProvider(_aliceDid))
          .value!
          .items
          .single;
      expect(calls, [('did:plc:alice', 'a')]);
      expect(updated.viewerHasLiked, isFalse);
      expect(updated.likeCount, 0);
    });

    test('rolls back live lists when repository call fails', () async {
      final post = _post(rkey: 'a', likeCount: 2);
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(items: [post]),
        onLike: (did, rkey) async => throw Exception('boom'),
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

      await container.read(userPostsProvider(_aliceDid).future);
      await container.read(toggleLikePostProvider.notifier).toggle(post: post);

      final current = container
          .read(userPostsProvider(_aliceDid))
          .value!
          .items
          .single;
      expect(container.read(toggleLikePostProvider).hasError, isTrue);
      expect(current.viewerHasLiked, isFalse);
      expect(current.likeCount, 2);
    });

    test('patches and rolls back live timeline entries', () async {
      final post = _post(rkey: 'a', likeCount: 2);
      var shouldFail = false;
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async => _timelinePage([post]),
        onLike: (did, rkey) async {
          if (shouldFail) throw Exception('boom');
          return _interaction(post);
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

      await container.read(timelineProvider.future);
      await container.read(toggleLikePostProvider.notifier).toggle(post: post);
      await container.read(timelineProvider.future);

      var current = container.read(timelineProvider).value!.items.single.post;
      expect(current.viewerHasLiked, isTrue);
      expect(current.likeCount, 3);

      shouldFail = true;
      await container
          .read(toggleLikePostProvider.notifier)
          .toggle(post: current);

      current = container.read(timelineProvider).value!.items.single.post;
      expect(container.read(toggleLikePostProvider).hasError, isTrue);
      expect(current.viewerHasLiked, isTrue);
      expect(current.likeCount, 3);
    });

    test(
      'IT-009 patches and rolls back the DID project like cache',
      () async {
        final post = _post(rkey: 'a', likeCount: 2, project: _project);
        var failLike = false;
        final fake = FakePostRepository(
          onListByAuthor: (id, {cursor, limit}) async =>
              PostPage(items: [_post(rkey: 'general')]),
          onListProjectsByAuthor: (id, {cursor, limit}) async =>
              PostPage(items: [post]),
          onLike: (did, rkey) async {
            if (failLike) throw Exception('boom');
            return _interaction(post);
          },
          onUnlike: (did, rkey) async {},
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
        await container.read(userPostsProvider(_aliceDid).future);
        await container.read(userProjectsProvider(_aliceDid).future);

        await container
            .read(toggleLikePostProvider.notifier)
            .toggle(post: post);
        await container.read(userProjectsProvider(_aliceDid).future);

        _expectProjectLikeCaches(container, liked: true, likeCount: 3);
        _expectProfilePostCachesUnchanged(container);

        final liked = container
            .read(userProjectsProvider(_aliceDid))
            .value!
            .items
            .single;
        await container
            .read(toggleLikePostProvider.notifier)
            .toggle(post: liked);
        await container.read(userProjectsProvider(_aliceDid).future);

        _expectProjectLikeCaches(container, liked: false, likeCount: 2);
        _expectProfilePostCachesUnchanged(container);

        final unliked = container
            .read(userProjectsProvider(_aliceDid))
            .value!
            .items
            .single;
        failLike = true;
        await container
            .read(toggleLikePostProvider.notifier)
            .toggle(post: unliked);

        expect(container.read(toggleLikePostProvider).hasError, isTrue);
        _expectProjectLikeCaches(container, liked: false, likeCount: 2);
        _expectProfilePostCachesUnchanged(container);
      },
    );
  });

  group('ToggleRepostPost', () {
    test('IT-016 retries ambiguity with one canonical operation key', () async {
      final post = _post(rkey: 'repost-retry');
      var calls = 0;
      final fake = FakePostRepository(
        onRepost: (did, rkey) async {
          calls++;
          if (calls < 3) {
            throw const PdsMutationAmbiguousException(retryAfterSeconds: 2);
          }
          return _interaction(post);
        },
      );
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(fake),
          pdsMutationDelayProvider.overrideWithValue((_) async {}),
          pdsMutationJitterProvider.overrideWithValue((_) => 0),
        ],
      );

      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: post);

      expect(calls, 3);
      expect(fake.repostOperationKeys.toSet(), hasLength(1));
      expect(
        isCanonicalPdsMutationOperationKey(fake.repostOperationKeys.first),
        isTrue,
      );
      expect(
        container.read(toggleRepostPostProvider).value?.viewerHasReposted,
        isTrue,
      );
    });

    test('IT-016 masks stale reads and retires on logical agreement', () async {
      var authoritativeReposted = false;
      final original = _post(rkey: 'repost-overlay', repostCount: 2);
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(
          items: [
            _post(
              rkey: 'repost-overlay',
              repostCount: authoritativeReposted ? 3 : 2,
              viewerHasReposted: authoritativeReposted,
            ),
          ],
        ),
        onRepost: (did, rkey) async => _interaction(original),
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

      await container.read(userPostsProvider(_aliceDid).future);
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: original);
      final staleRead = await container.read(
        userPostsProvider(_aliceDid).future,
      );
      expect(staleRead.items.single.viewerHasReposted, isTrue);
      expect(staleRead.items.single.repostCount, 3);

      authoritativeReposted = true;
      container.invalidate(userPostsProvider(_aliceDid));
      await container.read(userPostsProvider(_aliceDid).future);
      authoritativeReposted = false;
      container.invalidate(userPostsProvider(_aliceDid));
      final afterAgreement = await container.read(
        userPostsProvider(_aliceDid).future,
      );
      expect(afterAgreement.items.single.viewerHasReposted, isFalse);
      expect(afterAgreement.items.single.repostCount, 2);
    });

    test('accepted overlay updates user post lists', () async {
      final post = _post(rkey: 'a', repostCount: 1);
      final calls = <(String, String)>[];
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(items: [post]),
        onRepost: (did, rkey) async {
          calls.add((did, rkey));
          return _interaction(post);
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

      await container.read(userPostsProvider(_aliceDid).future);
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: post);
      await container.read(userPostsProvider(_aliceDid).future);

      final updated = container
          .read(userPostsProvider(_aliceDid))
          .value!
          .items
          .single;
      expect(calls, [('did:plc:alice', 'a')]);
      expect(updated.viewerHasReposted, isTrue);
      expect(updated.repostCount, 2);
    });

    test('unreposts without decrementing below zero', () async {
      final post = _post(rkey: 'a', viewerHasReposted: true);
      final calls = <(String, String)>[];
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(items: [post]),
        onUnrepost: (did, rkey) async => calls.add((did, rkey)),
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

      await container.read(userPostsProvider(_aliceDid).future);
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: post);
      await container.read(userPostsProvider(_aliceDid).future);

      final updated = container
          .read(userPostsProvider(_aliceDid))
          .value!
          .items
          .single;
      expect(calls, [('did:plc:alice', 'a')]);
      expect(updated.viewerHasReposted, isFalse);
      expect(updated.repostCount, 0);
    });

    test('rolls back live lists when repository call fails', () async {
      final post = _post(rkey: 'a', repostCount: 1);
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(items: [post]),
        onRepost: (did, rkey) async => throw Exception('boom'),
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

      await container.read(userPostsProvider(_aliceDid).future);
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: post);

      final current = container
          .read(userPostsProvider(_aliceDid))
          .value!
          .items
          .single;
      expect(container.read(toggleRepostPostProvider).hasError, isTrue);
      expect(current.viewerHasReposted, isFalse);
      expect(current.repostCount, 1);
    });

    test('patches and rolls back live timeline entries', () async {
      final post = _post(rkey: 'a', repostCount: 1);
      var shouldFail = false;
      final fake = FakePostRepository(
        onListTimeline: ({cursor, limit}) async => _timelinePage([post]),
        onRepost: (did, rkey) async {
          if (shouldFail) throw Exception('boom');
          return _interaction(post);
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

      await container.read(timelineProvider.future);
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: post);
      await container.read(timelineProvider.future);

      var current = container.read(timelineProvider).value!.items.single.post;
      expect(current.viewerHasReposted, isTrue);
      expect(current.repostCount, 2);

      shouldFail = true;
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: current);

      current = container.read(timelineProvider).value!.items.single.post;
      expect(container.read(toggleRepostPostProvider).hasError, isTrue);
      expect(current.viewerHasReposted, isTrue);
      expect(current.repostCount, 2);
    });

    test(
      'does not insert optimistic repost items into timeline cache',
      () async {
        final target = _post(rkey: 'target', repostCount: 1);
        final existing = _post(rkey: 'existing');
        final fake = FakePostRepository(
          onListTimeline: ({cursor, limit}) async => _timelinePage([existing]),
          onRepost: (did, rkey) async => _interaction(target),
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
        await container
            .read(toggleRepostPostProvider.notifier)
            .toggle(post: target);

        final timeline = container.read(timelineProvider).value!.items;
        expect(timeline.map((item) => item.post.rkey.toString()), ['existing']);
        expect(timeline, hasLength(1));
      },
    );

    test(
      'IT-009 patches and rolls back the DID project repost cache',
      () async {
        final post = _post(rkey: 'a', repostCount: 1, project: _project);
        var failRepost = false;
        final fake = FakePostRepository(
          onListByAuthor: (id, {cursor, limit}) async =>
              PostPage(items: [_post(rkey: 'general')]),
          onListProjectsByAuthor: (id, {cursor, limit}) async =>
              PostPage(items: [post]),
          onRepost: (did, rkey) async {
            if (failRepost) throw Exception('boom');
            return _interaction(post);
          },
          onUnrepost: (did, rkey) async {},
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
        await container.read(userPostsProvider(_aliceDid).future);
        await container.read(userProjectsProvider(_aliceDid).future);

        await container
            .read(toggleRepostPostProvider.notifier)
            .toggle(post: post);
        await container.read(userProjectsProvider(_aliceDid).future);

        _expectProjectRepostCaches(container, reposted: true, repostCount: 2);
        _expectProfilePostCachesUnchanged(container);

        final reposted = container
            .read(userProjectsProvider(_aliceDid))
            .value!
            .items
            .single;
        await container
            .read(toggleRepostPostProvider.notifier)
            .toggle(post: reposted);
        await container.read(userProjectsProvider(_aliceDid).future);

        _expectProjectRepostCaches(container, reposted: false, repostCount: 1);
        _expectProfilePostCachesUnchanged(container);

        final unreposted = container
            .read(userProjectsProvider(_aliceDid))
            .value!
            .items
            .single;
        failRepost = true;
        await container
            .read(toggleRepostPostProvider.notifier)
            .toggle(post: unreposted);

        expect(container.read(toggleRepostPostProvider).hasError, isTrue);
        _expectProjectRepostCaches(container, reposted: false, repostCount: 1);
        _expectProfilePostCachesUnchanged(container);
      },
    );
  });
}

void _expectProjectLikeCaches(
  ProviderContainer container, {
  required bool liked,
  required int likeCount,
}) {
  final project = container
      .read(userProjectsProvider(_aliceDid))
      .value!
      .items
      .single;
  expect(project.viewerHasLiked, liked);
  expect(project.likeCount, likeCount);
}

void _expectProfilePostCachesUnchanged(ProviderContainer container) {
  expect(
    container.read(userPostsProvider(_aliceDid)).value!.items.single.rkey,
    'general',
  );
}

void _expectProjectRepostCaches(
  ProviderContainer container, {
  required bool reposted,
  required int repostCount,
}) {
  final project = container
      .read(userProjectsProvider(_aliceDid))
      .value!
      .items
      .single;
  expect(project.viewerHasReposted, reposted);
  expect(project.repostCount, repostCount);
}
