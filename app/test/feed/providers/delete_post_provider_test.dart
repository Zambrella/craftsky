import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
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

  group('DeletePost', () {
    test('idle build returns null', () async {
      final container = ProviderContainer.test(
        overrides: [
          postRepositoryProvider.overrideWithValue(FakePostRepository()),
        ],
      );

      expect(container.read(deletePostProvider).value, isNull);
    });

    test('successful delete removes from live family entries '
        '(both did and handle keys)', () async {
      final deleted = <(String, String)>[];
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(
          items: [_post(rkey: 'a'), _post(rkey: 'b')],
        ),
        onDelete: (did, rkey) async {
          deleted.add((did, rkey));
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('did:plc:alice').future);
      await container.read(
        userPostsProvider('alice.craftsky.social').future,
      );

      await container
          .read(deletePostProvider.notifier)
          .delete(post: _post(rkey: 'a'));

      expect(deleted, [('did:plc:alice', 'a')]);

      final didList = container.read(userPostsProvider('did:plc:alice')).value!;
      final handleList = container
          .read(userPostsProvider('alice.craftsky.social'))
          .value!;
      expect(didList.items.map((p) => p.rkey), ['b']);
      expect(handleList.items.map((p) => p.rkey), ['b']);
    });

    test('failure surfaces as AsyncError, cache untouched', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_post(rkey: 'a')]),
        onDelete: (did, rkey) async => throw Exception('boom'),
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container.read(userPostsProvider('did:plc:alice').future);

      await container
          .read(deletePostProvider.notifier)
          .delete(post: _post(rkey: 'a'));

      expect(container.read(deletePostProvider).hasError, isTrue);
      final list = container.read(userPostsProvider('did:plc:alice')).value!;
      expect(list.items.map((p) => p.rkey), ['a']);
    });

    test('reset() returns to AsyncData(null)', () async {
      final fake = FakePostRepository(
        onDelete: (did, rkey) async {},
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container
          .read(deletePostProvider.notifier)
          .delete(post: _post(rkey: 'a'));
      expect(container.read(deletePostProvider).value?.rkey, 'a');

      container.read(deletePostProvider.notifier).reset();
      expect(container.read(deletePostProvider).value, isNull);
    });
  });
}
