import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_repost_post_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

Map<String, dynamic> _postMap({
  required String rkey,
  int likeCount = 0,
  int repostCount = 0,
  bool viewerHasLiked = false,
  bool viewerHasReposted = false,
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
};

Post _post({
  required String rkey,
  int likeCount = 0,
  int repostCount = 0,
  bool viewerHasLiked = false,
  bool viewerHasReposted = false,
}) => PostMapper.fromMap(
  _postMap(
    rkey: rkey,
    likeCount: likeCount,
    repostCount: repostCount,
    viewerHasLiked: viewerHasLiked,
    viewerHasReposted: viewerHasReposted,
  ),
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
  });
}
