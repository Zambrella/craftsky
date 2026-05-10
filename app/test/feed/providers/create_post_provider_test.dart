import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

Map<String, dynamic> _postMap({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
}) => {
  'uri': 'at://$did/social.craftsky.feed.post/$rkey',
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
  'author': {'did': did, 'handle': handle},
};

Post _post({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
}) => PostMapper.fromMap(_postMap(rkey: rkey, did: did, handle: handle));

void main() {
  setUpAll(initializeMappers);

  group('CreatePost', () {
    test('idle build returns null', () async {
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(FakePostRepository()),
        ],
      );

      final state = container.read(createPostProvider);
      expect(state.value, isNull);
      expect(state.isLoading, isFalse);
    });

    test('successful create transitions loading -> data(post)', () async {
      final fake = FakePostRepository(
        onCreate: ({required text}) async => _post(rkey: 'new'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      final transitions = <AsyncValue<Post?>>[];
      container.listen(createPostProvider, (_, next) => transitions.add(next));

      await container.read(createPostProvider.notifier).create(text: 'hi');

      expect(transitions.first, isA<AsyncLoading<Post?>>());
      expect(transitions.last.value?.rkey, 'new');
    });

    test('success prepends into live userPostsProvider entries '
        '(both did and handle keys)', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_post(rkey: 'old')]),
        onCreate: ({required text}) async => _post(rkey: 'new'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      // Pre-instantiate both family entries so they are "live".
      await container.read(userPostsProvider('did:plc:alice').future);
      await container.read(userPostsProvider('alice.craftsky.social').future);

      await container.read(createPostProvider.notifier).create(text: 'hi');

      final didEntry = container
          .read(userPostsProvider('did:plc:alice'))
          .value!;
      final handleEntry = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(didEntry.items.map((p) => p.rkey), ['new', 'old']);
      expect(handleEntry.items.map((p) => p.rkey), ['new', 'old']);
    });

    test('does not instantiate a non-live family entry', () async {
      final calls = <String>[];
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async {
          calls.add(id);
          return PostPage(items: [_post(rkey: 'x')]);
        },
        onCreate: ({required text}) async => _post(rkey: 'new'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(createPostProvider.notifier).create(text: 'hi');

      expect(
        calls,
        isEmpty,
        reason:
            'CreatePost must not call ref.exists() in a way that '
            'auto-instantiates the family entry',
      );
    });

    test('reset() returns to AsyncData(null)', () async {
      final fake = FakePostRepository(
        onCreate: ({required text}) async => _post(rkey: 'new'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(createPostProvider.notifier).create(text: 'hi');
      expect(container.read(createPostProvider).value?.rkey, 'new');

      container.read(createPostProvider.notifier).reset();
      expect(container.read(createPostProvider).value, isNull);
    });

    test('failure surfaces as AsyncError, no cache mutation', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_post(rkey: 'old')]),
        onCreate: ({required text}) async => throw Exception('boom'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('did:plc:alice').future);

      await container.read(createPostProvider.notifier).create(text: 'hi');

      expect(container.read(createPostProvider).hasError, isTrue);
      final list = container.read(userPostsProvider('did:plc:alice')).value!;
      expect(list.items.map((p) => p.rkey), ['old']);
    });
  });
}
