import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../feed/fakes/fake_post_repository.dart';

Map<String, dynamic> _postMap({
  required String rkey,
  bool withProject = true,
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
  'viewerHasReplied': false,
  'viewerHasSaved': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': did, 'handle': handle},
  if (withProject)
    'project': {
      'common': {'craftType': 'social.craftsky.feed.defs#knitting'},
    },
};

Post _post({required String rkey, bool withProject = true}) =>
    PostMapper.fromMap(_postMap(rkey: rkey, withProject: withProject));

void main() {
  setUpAll(initializeMappers);

  group('userProjectsProvider', () {
    test(
      'AT-007 builds with limit 10 and preserves null-project rows',
      () async {
        String? seenId;
        int? seenLimit;
        final fake = FakePostRepository(
          onListProjectsByAuthor: (id, {cursor, limit}) async {
            seenId = id;
            seenLimit = limit;
            return PostPage(
              items: [
                _post(rkey: 'project'),
                _post(rkey: 'unexpected', withProject: false),
              ],
              cursor: 'next',
            );
          },
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );
        final provider = userProjectsProvider('alice.craftsky.social');
        final subscription = container.listen(provider, (_, _) {});
        addTearDown(subscription.close);

        final state = await container.read(provider.future);

        expect(seenId, 'alice.craftsky.social');
        expect(seenLimit, userProjectsPageLimit);
        expect(state.items.map((post) => post.rkey), ['project', 'unexpected']);
        expect(state.items.last.project, isNull);
        expect(state.cursor, 'next');
        expect(state.hasMore, isTrue);
      },
    );

    test(
      'UT-014 loadMore appends, preserves data on failure, no-ops when loading',
      () async {
        var calls = 0;
        final gate = Completer<PostPage>();
        final fake = FakePostRepository(
          onListProjectsByAuthor: (id, {cursor, limit}) async {
            calls++;
            if (calls == 1) {
              return PostPage(
                items: [_post(rkey: 'a')],
                cursor: 'c1',
              );
            }
            if (calls == 2) throw Exception('network down');
            if (calls == 3) return gate.future;
            return PostPage(items: [_post(rkey: 'c')]);
          },
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );
        final sub = container.listen(userProjectsProvider('alice'), (_, _) {});
        addTearDown(sub.close);

        await container.read(userProjectsProvider('alice').future);
        await container.read(userProjectsProvider('alice').notifier).loadMore();
        final failed = container.read(userProjectsProvider('alice'));
        expect(failed.hasError, isTrue);
        expect(failed.value?.items.map((post) => post.rkey), ['a']);
        expect(failed.value?.cursor, 'c1');

        final inFlight = container
            .read(userProjectsProvider('alice').notifier)
            .loadMore();
        await Future<void>.delayed(Duration.zero);
        await container.read(userProjectsProvider('alice').notifier).loadMore();
        expect(calls, 3);
        gate.complete(PostPage(items: [_post(rkey: 'b')]));
        await inFlight;

        final state = container.read(userProjectsProvider('alice')).value!;
        expect(state.items.map((post) => post.rkey), ['a', 'b']);
      },
    );

    test(
      'UT-016 and UT-017 cache helpers prepend, dedupe, replace, and remove',
      () async {
        final fake = FakePostRepository(
          onListProjectsByAuthor: (id, {cursor, limit}) async =>
              PostPage(items: [_post(rkey: 'a')]),
        );
        final container = ProviderContainer.test(
          overrides: [postRepositoryProvider.overrideWithValue(fake)],
        );

        await container.read(
          userProjectsProvider('alice.craftsky.social').future,
        );
        final notifier = container.read(
          userProjectsProvider('alice.craftsky.social').notifier,
        );
        // Exercise helper calls one-by-one to assert the resulting cache state.
        // ignore: cascade_invocations
        notifier
          ..prepend(_post(rkey: 'b'))
          ..prepend(_post(rkey: 'b'))
          ..replace(_post(rkey: 'a').copyWith(text: 'updated'))
          ..removeByRkey('b');

        final state = container
            .read(userProjectsProvider('alice.craftsky.social'))
            .value!;
        expect(state.items.map((post) => post.rkey), ['a']);
        expect(state.items.single.text, 'updated');
      },
    );
  });
}
