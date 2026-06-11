import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

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

  group('ToggleLikePost', () {
    test('optimistically patches live user post lists', () async {
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
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('alice.craftsky.social').future);
      await container.read(userPostsProvider('did:plc:alice').future);
      await container.read(toggleLikePostProvider.notifier).toggle(post: post);

      final handleUpdated = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!
          .items
          .single;
      final didUpdated = container
          .read(userPostsProvider('did:plc:alice'))
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
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('alice.craftsky.social').future);
      await container.read(toggleLikePostProvider.notifier).toggle(post: post);

      final updated = container
          .read(userPostsProvider('alice.craftsky.social'))
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
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('alice.craftsky.social').future);
      await container.read(toggleLikePostProvider.notifier).toggle(post: post);

      final current = container
          .read(userPostsProvider('alice.craftsky.social'))
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
        onListTimeline: ({cursor, limit}) async => PostPage(items: [post]),
        onLike: (did, rkey) async {
          if (shouldFail) throw Exception('boom');
          return _interaction(post);
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(timelineProvider.future);
      await container.read(toggleLikePostProvider.notifier).toggle(post: post);

      var current = container.read(timelineProvider).value!.items.single;
      expect(current.viewerHasLiked, isTrue);
      expect(current.likeCount, 3);

      shouldFail = true;
      await container
          .read(toggleLikePostProvider.notifier)
          .toggle(post: current);

      current = container.read(timelineProvider).value!.items.single;
      expect(container.read(toggleLikePostProvider).hasError, isTrue);
      expect(current.viewerHasLiked, isTrue);
      expect(current.likeCount, 3);
    });

    test(
      'IT-009 patches and rolls back project like caches for did/handle keys',
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
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );
        await container.read(userPostsProvider('did:plc:alice').future);
        await container.read(userPostsProvider('alice.craftsky.social').future);
        await container.read(userProjectsProvider('did:plc:alice').future);
        await container.read(
          userProjectsProvider('alice.craftsky.social').future,
        );

        await container
            .read(toggleLikePostProvider.notifier)
            .toggle(post: post);

        _expectProjectLikeCaches(container, liked: true, likeCount: 3);
        _expectProfilePostCachesUnchanged(container);

        final liked = container
            .read(userProjectsProvider('alice.craftsky.social'))
            .value!
            .items
            .single;
        await container
            .read(toggleLikePostProvider.notifier)
            .toggle(post: liked);

        _expectProjectLikeCaches(container, liked: false, likeCount: 2);
        _expectProfilePostCachesUnchanged(container);

        final unliked = container
            .read(userProjectsProvider('alice.craftsky.social'))
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
    test('optimistically patches live user post lists', () async {
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
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('alice.craftsky.social').future);
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: post);

      final updated = container
          .read(userPostsProvider('alice.craftsky.social'))
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
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('alice.craftsky.social').future);
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: post);

      final updated = container
          .read(userPostsProvider('alice.craftsky.social'))
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
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('alice.craftsky.social').future);
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: post);

      final current = container
          .read(userPostsProvider('alice.craftsky.social'))
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
        onListTimeline: ({cursor, limit}) async => PostPage(items: [post]),
        onRepost: (did, rkey) async {
          if (shouldFail) throw Exception('boom');
          return _interaction(post);
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(timelineProvider.future);
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: post);

      var current = container.read(timelineProvider).value!.items.single;
      expect(current.viewerHasReposted, isTrue);
      expect(current.repostCount, 2);

      shouldFail = true;
      await container
          .read(toggleRepostPostProvider.notifier)
          .toggle(post: current);

      current = container.read(timelineProvider).value!.items.single;
      expect(container.read(toggleRepostPostProvider).hasError, isTrue);
      expect(current.viewerHasReposted, isTrue);
      expect(current.repostCount, 2);
    });

    test(
      'IT-009 patches and rolls back project repost caches for did/handle keys',
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
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );
        await container.read(userPostsProvider('did:plc:alice').future);
        await container.read(userPostsProvider('alice.craftsky.social').future);
        await container.read(userProjectsProvider('did:plc:alice').future);
        await container.read(
          userProjectsProvider('alice.craftsky.social').future,
        );

        await container
            .read(toggleRepostPostProvider.notifier)
            .toggle(post: post);

        _expectProjectRepostCaches(container, reposted: true, repostCount: 2);
        _expectProfilePostCachesUnchanged(container);

        final reposted = container
            .read(userProjectsProvider('alice.craftsky.social'))
            .value!
            .items
            .single;
        await container
            .read(toggleRepostPostProvider.notifier)
            .toggle(post: reposted);

        _expectProjectRepostCaches(container, reposted: false, repostCount: 1);
        _expectProfilePostCachesUnchanged(container);

        final unreposted = container
            .read(userProjectsProvider('alice.craftsky.social'))
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
  for (final id in const ['did:plc:alice', 'alice.craftsky.social']) {
    final project = container
        .read(userProjectsProvider(id))
        .value!
        .items
        .single;
    expect(project.viewerHasLiked, liked);
    expect(project.likeCount, likeCount);
  }
}

void _expectProfilePostCachesUnchanged(ProviderContainer container) {
  for (final id in const ['did:plc:alice', 'alice.craftsky.social']) {
    expect(
      container.read(userPostsProvider(id)).value!.items.single.rkey,
      'general',
    );
  }
}

void _expectProjectRepostCaches(
  ProviderContainer container, {
  required bool reposted,
  required int repostCount,
}) {
  for (final id in const ['did:plc:alice', 'alice.craftsky.social']) {
    final project = container
        .read(userProjectsProvider(id))
        .value!
        .items
        .single;
    expect(project.viewerHasReposted, reposted);
    expect(project.repostCount, repostCount);
  }
}
