import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/post_thread.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/post_thread_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

Map<String, dynamic> _postMap({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
  PostReply? reply,
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
  if (reply != null) 'reply': reply.toMap(),
};

Post _post({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
  PostReply? reply,
}) => PostMapper.fromMap(
  _postMap(rkey: rkey, did: did, handle: handle, reply: reply),
);

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
        onCreate: ({required text, reply}) async => _post(rkey: 'new'),
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

    test('root post reply uses target uri/cid for root and parent', () async {
      final target = _post(rkey: 'target');
      PostReply? capturedReply;
      final fake = FakePostRepository(
        onCreate: ({required text, reply}) async {
          capturedReply = reply;
          return _post(rkey: 'reply');
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container
          .read(createPostProvider.notifier)
          .create(
            text: 'hi',
            reply: PostReply(
              root: PostRef(uri: target.uri, cid: target.cid),
              parent: PostRef(uri: target.uri, cid: target.cid),
            ),
          );

      expect(capturedReply, isNotNull);
      expect(capturedReply!.root.uri, target.uri);
      expect(capturedReply!.root.cid, target.cid);
      expect(capturedReply!.parent.uri, target.uri);
      expect(capturedReply!.parent.cid, target.cid);
    });

    test('reply-to-reply preserves the target thread root', () async {
      final target = _post(
        rkey: 'target',
        reply: const PostReply(
          root: PostRef(
            uri: 'at://did:plc:root/social.craftsky.feed.post/root',
            cid: 'bafy_root',
          ),
          parent: PostRef(
            uri: 'at://did:plc:parent/social.craftsky.feed.post/parent',
            cid: 'bafy_parent',
          ),
        ),
      );
      PostReply? capturedReply;
      final fake = FakePostRepository(
        onCreate: ({required text, reply}) async {
          capturedReply = reply;
          return _post(rkey: 'reply');
        },
      );
      final container = ProviderContainer.test(
        overrides: [postRepositoryProvider.overrideWithValue(fake)],
      );

      await container
          .read(createPostProvider.notifier)
          .create(
            text: 'hi',
            reply: PostReply(
              root: target.reply!.root,
              parent: PostRef(uri: target.uri, cid: target.cid),
            ),
          );

      expect(capturedReply, isNotNull);
      expect(capturedReply!.root.uri, target.reply!.root.uri);
      expect(capturedReply!.root.cid, target.reply!.root.cid);
      expect(capturedReply!.parent.uri, target.uri);
      expect(capturedReply!.parent.cid, target.cid);
    });

    test(
      'reply success invalidates live thread and replies for the target',
      () async {
        final target = _post(rkey: 'target');
        var threadCalls = 0;
        var replyCalls = 0;
        final fake = FakePostRepository(
          onThread: (did, rkey) async {
            threadCalls += 1;
            expect(did, target.author.did);
            expect(rkey, target.rkey);
            return PostThread(
              post: target,
              replies: const [],
            );
          },
          onListDirectReplies: (did, rkey, {cursor, limit}) async {
            replyCalls += 1;
            expect(did, target.author.did);
            expect(rkey, target.rkey);
            return const PostPage(items: []);
          },
          onCreate: ({required text, reply}) async => _post(
            rkey: 'reply',
            reply: reply,
          ),
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );
        final threadProvider = postThreadProvider(
          target.author.did,
          target.rkey,
        );
        final repliesProvider = directRepliesProvider(
          target.author.did,
          target.rkey,
        );
        final threadSub = container.listen(
          threadProvider,
          (_, _) {},
          fireImmediately: true,
        );
        final repliesSub = container.listen(
          repliesProvider,
          (_, _) {},
          fireImmediately: true,
        );

        await container.read(threadProvider.future);
        await container.read(repliesProvider.future);
        expect(threadCalls, 1);
        expect(replyCalls, 1);

        await container
            .read(createPostProvider.notifier)
            .create(
              text: 'hi',
              reply: PostReply(
                root: PostRef(uri: target.uri, cid: target.cid),
                parent: PostRef(uri: target.uri, cid: target.cid),
              ),
            );

        await container.read(threadProvider.future);
        await container.read(repliesProvider.future);
        expect(threadCalls, 2);
        expect(replyCalls, 2);

        threadSub.close();
        repliesSub.close();
      },
    );

    test(
      'reply-to-reply success invalidates the immediate parent target',
      () async {
        final target = _post(
          rkey: 'target',
          reply: const PostReply(
            root: PostRef(
              uri: 'at://did:plc:root/social.craftsky.feed.post/root',
              cid: 'bafy_root',
            ),
            parent: PostRef(
              uri: 'at://did:plc:parent/social.craftsky.feed.post/parent',
              cid: 'bafy_parent',
            ),
          ),
        );
        var threadCalls = 0;
        var replyCalls = 0;
        final fake = FakePostRepository(
          onThread: (did, rkey) async {
            threadCalls += 1;
            expect(did, target.author.did);
            expect(rkey, target.rkey);
            return PostThread(
              post: target,
              replies: const [],
            );
          },
          onListDirectReplies: (did, rkey, {cursor, limit}) async {
            replyCalls += 1;
            expect(did, target.author.did);
            expect(rkey, target.rkey);
            return const PostPage(items: []);
          },
          onCreate: ({required text, reply}) async => _post(
            rkey: 'reply',
            reply: reply,
          ),
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );
        final threadProvider = postThreadProvider(
          target.author.did,
          target.rkey,
        );
        final repliesProvider = directRepliesProvider(
          target.author.did,
          target.rkey,
        );
        final threadSub = container.listen(
          threadProvider,
          (_, _) {},
          fireImmediately: true,
        );
        final repliesSub = container.listen(
          repliesProvider,
          (_, _) {},
          fireImmediately: true,
        );

        await container.read(threadProvider.future);
        await container.read(repliesProvider.future);
        expect(threadCalls, 1);
        expect(replyCalls, 1);

        await container
            .read(createPostProvider.notifier)
            .create(
              text: 'hi',
              reply: PostReply(
                root: target.reply!.root,
                parent: PostRef(uri: target.uri, cid: target.cid),
              ),
            );

        await container.read(threadProvider.future);
        await container.read(repliesProvider.future);
        expect(threadCalls, 2);
        expect(replyCalls, 2);

        threadSub.close();
        repliesSub.close();
      },
    );

    test('success prepends into live userPostsProvider entries '
        '(both did and handle keys)', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_post(rkey: 'old')]),
        onCreate: ({required text, reply}) async => _post(rkey: 'new'),
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
        onCreate: ({required text, reply}) async => _post(rkey: 'new'),
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
        onCreate: ({required text, reply}) async => _post(rkey: 'new'),
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
        onCreate: ({required text, reply}) async => throw Exception('boom'),
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
