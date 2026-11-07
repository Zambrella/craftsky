import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart' as model;
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    as auth;
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/models/timeline_state.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/follow_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_follow_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../profile/fakes/fake_profile_repository.dart';
import '../../test_support/member_browsing_context.dart';
import '../../test_support/post_fixtures.dart';
import '../fakes/fake_post_repository.dart';

final Post _post = postFixture(did: 'did:plc:author', handle: 'author.test');
final _bridge = PostAuthor(did: 'did:plc:bridge', handle: 'bridge.test');
final _item = TimelineItem(
  itemKey: 'post:${_post.uri}',
  post: _post,
  followedBy: _bridge,
);
final _stalePage = TimelinePage(items: [_item]);

Profile _target(bool followingAuthor, {required bool indexed}) => Profile(
  did: followingAuthor ? _post.author.did.value : _bridge.did.value,
  handle: followingAuthor ? 'author.test' : 'bridge.test',
  crafts: const [],
  viewerIsFollowing: indexed ? followingAuthor : !followingAuthor,
);

ProviderContainer _container({
  required FakePostRepository posts,
  required FakeProfileRepository profiles,
  required PdsRecordOperationController controller,
}) => ProviderContainer.test(
  overrides: [
    testMemberBrowsingOverride,
    activeLanguagePreferencesProvider.overrideWith(
      (ref) => const LanguagePreferences(
        primaryLanguage: 'en',
        contentLanguages: ['en'],
      ),
    ),
    postRepositoryProvider.overrideWithValue(posts),
    profileRepositoryProvider.overrideWithValue(profiles),
    pdsRecordOperationControllerProvider.overrideWithValue(controller),
  ],
);

Future<void> _toggle(ProviderContainer container, Profile baseline) => container
    .read(toggleFollowProfileProvider.notifier)
    .toggle(cacheKey: baseline.did, profile: baseline);

final class _Registry extends auth.SessionRegistry {
  new(this.initial);

  final model.SessionRegistry initial;

  @override
  Future<model.SessionRegistry> build() async => initial;

  void replace(model.SessionRegistry value) => state = AsyncData(value);
}

Provider<void Function()> _confirmFollow(
  PdsRecordOperationController controller,
) => Provider(
  (ref) => () {
    final indexed = _target(true, indexed: true);
    final scope = followProfileMutationScope(ref, indexed.did);
    final token = controller.begin(
      scope: scope,
      operationKey: 'confirmed-follow',
      endpoint: '/v1/profiles/@${indexed.did}/follows',
      immutableBody: 'POST\n',
    );
    controller.markAccepted(
      token,
      optimisticValue: true,
      agrees: (value) => value == true,
    );
    applyFollowProfileOverlay(ref, indexed);
  },
);

void main() {
  setUpAll(initializeMappers);
  for (final followingAuthor in [true, false]) {
    final action = followingAuthor
        ? 'following the author'
        : 'unfollowing the intermediary';
    test('accepted $action suppresses a stale network response', () async {
      final baseline = _target(followingAuthor, indexed: false);
      final accepted = _target(followingAuthor, indexed: true);
      final controller = PdsRecordOperationController(schedule: (_, _) {});
      final container = _container(
        controller: controller,
        posts: FakePostRepository(
          onListExtendedNetwork: ({cursor, limit}) async => _stalePage,
        ),
        profiles: FakeProfileRepository(
          onFetch: (_) async => baseline,
          onFollow: (_) async => accepted,
          onUnfollow: (_) async => accepted,
        ),
      )..listen(extendedNetworkProvider, (_, _) {});
      expect(
        (await container.read(extendedNetworkProvider.future)).items,
        hasLength(1),
      );
      await _toggle(container, baseline);
      expect(controller.activeOverlays, hasLength(1));
      expect(
        (await container.read(extendedNetworkProvider.future)).items,
        isEmpty,
      );
    });

    test(
      'indexed $action supersedes an earlier pending feed request',
      () async {
        final baseline = _target(followingAuthor, indexed: false);
        final indexed = _target(followingAuthor, indexed: true);
        final stale = Completer<TimelinePage>();
        var calls = 0;
        final controller = PdsRecordOperationController(schedule: (_, _) {});
        final container = _container(
          controller: controller,
          posts: FakePostRepository(
            onListExtendedNetwork: ({cursor, limit}) async {
              calls++;
              if (calls == 1) return _stalePage;
              if (calls == 2) return await stale.future;
              return const TimelinePage(items: []);
            },
          ),
          profiles: FakeProfileRepository(
            onFetch: (_) async => indexed,
            onFollow: (_) async => indexed,
            onUnfollow: (_) async => indexed,
          ),
        )..listen(extendedNetworkProvider, (_, _) {});
        await container.read(extendedNetworkProvider.future);
        await _toggle(container, baseline);
        final pending = container
            .read(extendedNetworkProvider.future)
            .then<TimelineState?>(
              (value) => value,
              onError: (Object error) {
                expect(error, isA<ApiCanceled>());
                return null;
              },
            );
        await container.pump();
        expect(calls, 2);
        await container.read(userProfileProvider(baseline.did).future);
        await container.pump();
        expect(controller.activeOverlays, isEmpty);
        stale.complete(_stalePage);
        await pending;
        await container.pump();
        expect(calls, 3);
        expect(
          (await container.read(extendedNetworkProvider.future)).items,
          isEmpty,
        );
      },
    );
  }

  test(
    'a stale pagination completion cannot restore a newly followed author',
    () async {
      final baseline = _target(true, indexed: false);
      final indexed = _target(true, indexed: true);
      final retained = TimelineItem(
        itemKey: 'post:retained',
        post: postFixture(did: 'did:plc:retained', handle: 'retained.test'),
        followedBy: _bridge,
      );
      final stale = Completer<TimelinePage>();
      var calls = 0;
      final controller = PdsRecordOperationController(schedule: (_, _) {});
      final container = _container(
        controller: controller,
        posts: FakePostRepository(
          onListExtendedNetwork: ({cursor, limit}) async {
            calls++;
            if (calls == 2) return await stale.future;
            return TimelinePage(
              items: [retained],
              cursor: calls == 1 ? 'old-next' : 'fresh-next',
            );
          },
        ),
        profiles: FakeProfileRepository(
          onFetch: (_) async => indexed,
          onFollow: (_) async => indexed,
        ),
      )..listen(extendedNetworkProvider, (_, _) {});
      await container.read(extendedNetworkProvider.future);
      final loading = container
          .read(extendedNetworkProvider.notifier)
          .loadMore();
      await container.pump();
      expect(calls, 2);
      await _toggle(container, baseline);
      await container.read(extendedNetworkProvider.future);
      await container.read(userProfileProvider(baseline.did).future);
      await container.pump();
      stale.complete(_stalePage);
      await loading;
      final current = await container.read(extendedNetworkProvider.future);
      expect(current.items.map((item) => item.itemKey), [retained.itemKey]);
      expect(current.cursor, 'fresh-next');
      expect(calls, greaterThanOrEqualTo(4));
    },
  );

  test('a disposed profile read cannot publish a catch-up signal', () async {
    final controller = PdsRecordOperationController(schedule: (_, _) {});
    final confirm = _confirmFollow(controller);
    final container = ProviderContainer.test(
      overrides: [
        auth.sessionRegistryProvider.overrideWith(
          () => _Registry(model.SessionRegistry.empty()),
        ),
        pdsRecordOperationControllerProvider.overrideWithValue(controller),
      ],
    );
    await container.read(auth.sessionRegistryProvider.future);
    container.listen(followProfileProjectionRevisionProvider, (_, _) {});
    final revision = container.read(followProfileProjectionRevisionProvider);

    container.read(confirm)();
    container
      ..invalidate(confirm)
      ..read(confirm);
    await container.pump();

    expect(
      container.read(followProfileProjectionRevisionProvider),
      same(revision),
    );
  });

  test('a catch-up signal cannot cross an account activation', () async {
    final initial = model.SessionRegistry.empty().upsertAndActivate(
      token: 'alice-test-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final replacement = initial.upsertAndActivate(
      token: 'other-test-token',
      did: 'did:plc:other',
      handle: 'other.test',
    );
    final controller = PdsRecordOperationController(schedule: (_, _) {});
    final confirm = _confirmFollow(controller);
    final container = ProviderContainer.test(
      overrides: [
        auth.sessionRegistryProvider.overrideWith(() => _Registry(initial)),
        pdsRecordOperationControllerProvider.overrideWithValue(controller),
      ],
    );
    await container.read(auth.sessionRegistryProvider.future);
    container.listen(followProfileProjectionRevisionProvider, (_, _) {});
    final previous = container.read(followProfileProjectionRevisionProvider);

    container.read(confirm)();
    (container.read(auth.sessionRegistryProvider.notifier) as _Registry)
        .replace(replacement);
    final current = container.read(followProfileProjectionRevisionProvider);
    expect(current, isNot(same(previous)));
    await container.pump();

    expect(
      container.read(followProfileProjectionRevisionProvider),
      same(current),
    );
  });
}
