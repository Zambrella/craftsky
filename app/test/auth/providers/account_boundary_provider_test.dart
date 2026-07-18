import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/toggle_like_post_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../feed/fakes/fake_post_repository.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

Post _post(String rkey) => PostMapper.fromMap({
  'uri': 'at://did:plc:author/social.craftsky.feed.post/$rkey',
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
  'author': {'did': 'did:plc:author', 'handle': 'author.test'},
});

TimelinePage _timelinePage(Post post, {String? cursor}) => TimelinePage(
  items: [TimelineItem(itemKey: 'post:${post.uri}', post: post)],
  cursor: cursor,
);

void main() {
  setUpAll(initializeMappers);

  test(
    'authoritative active removal invalidates state and resets fallback Home',
    () async {
      var registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-alice',
            did: 'did:plc:alice',
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'token-bob',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      final aliceLease = registry.leaseFor(AccountKey('did:plc:alice'))!;
      final effects = <String>[];
      final coordinator = AccountSessionInvalidationCoordinator(
        readRegistry: () async => registry,
        invalidateLease: (lease) async {
          if (registry.leaseFor(lease.account) == lease) {
            registry = registry.remove(lease.account.did.value);
          }
        },
        invalidateAccountState: () async => effects.add('invalidate-account'),
        resetHome: () async => effects.add('home'),
      );

      await coordinator.invalidate(registry.activeLease!.session);

      expect(registry.activeDid?.value, aliceLease.account.did);
      expect(effects, ['invalidate-account', 'home']);
    },
  );

  test(
    'inactive and stale removals do not disturb the active UI boundary',
    () async {
      var registry = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-alice',
            did: 'did:plc:alice',
            handle: 'alice.test',
          )
          .upsertAndActivate(
            token: 'token-bob',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      final aliceLease = registry.leaseFor(AccountKey('did:plc:alice'))!;
      final effects = <String>[];
      final coordinator = AccountSessionInvalidationCoordinator(
        readRegistry: () async => registry,
        invalidateLease: (lease) async {
          if (registry.leaseFor(lease.account) == lease) {
            registry = registry.remove(lease.account.did.value);
          }
        },
        invalidateAccountState: () async => effects.add('invalidate-account'),
        resetHome: () async => effects.add('home'),
      );

      await coordinator.invalidate(aliceLease);
      await coordinator.invalidate(
        AccountSessionLease(
          account: AccountKey('did:plc:bob'),
          sessionGeneration: 999,
        ),
      );

      expect(registry.activeDid?.value, 'did:plc:bob');
      expect(effects, isEmpty);
    },
  );

  test(
    'UT-004 IT-003 late A reads errors and rollback cannot publish as B',
    () async {
      final postA = _post('account-a');
      final postALate = _post('account-a-late');
      final postB = _post('account-b');
      final lateTimeline = Completer<TimelinePage>();
      final lateUserPosts = Completer<PostPage>();
      final lateLike = Completer<InteractionWriteResponse>();
      var firstTimeline = true;
      var firstUserPosts = true;
      final repository = FakePostRepository(
        onListTimeline: ({cursor, limit}) {
          if (cursor != null) return lateTimeline.future;
          if (firstTimeline) {
            firstTimeline = false;
            return Future.value(_timelinePage(postA, cursor: 'a-next'));
          }
          return Future.value(_timelinePage(postB));
        },
        onListByAuthor: (id, {cursor, limit}) {
          if (firstUserPosts) {
            firstUserPosts = false;
            return lateUserPosts.future;
          }
          return Future.value(PostPage(items: [postB]));
        },
        onLike: (did, rkey) => lateLike.future,
      );
      final initial = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          postRepositoryProvider.overrideWithValue(repository),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final timelineSubscription = container.listen(
        timelineProvider,
        (_, _) {},
        fireImmediately: true,
      );
      final userPostsSubscription = container.listen(
        userPostsProvider('author.test'),
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(timelineSubscription.close);
      addTearDown(userPostsSubscription.close);

      await container.read(timelineProvider.future);
      final lateLoadMore = container.read(timelineProvider.notifier).loadMore();
      final lateRollback = container
          .read(toggleLikePostProvider.notifier)
          .toggle(post: postA);
      await Future<void>.delayed(Duration.zero);
      expect(
        container
            .read(timelineProvider)
            .requireValue
            .items
            .single
            .post
            .viewerHasLiked,
        isTrue,
      );

      final target = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;
      final coordinator = AccountActivationCoordinator(
        readRegistry: () =>
            container.read(sessionRegistryProvider).requireValue,
        commitActivation: container
            .read(sessionRegistryProvider.notifier)
            .activate,
        publishTransition: (_) {},
        invalidateAccountState: container.read(accountStateInvalidatorProvider),
        resetToHome: () async {},
      );
      await coordinator.activate(
        target,
        source: AccountActivationSource.manual,
      );
      for (var index = 0; index < 5; index++) {
        await Future<void>.delayed(Duration.zero);
      }
      expect(
        container
            .read(timelineProvider)
            .requireValue
            .items
            .map((item) => item.post.rkey),
        ['account-b'],
      );
      expect(
        container
            .read(userPostsProvider('author.test'))
            .requireValue
            .items
            .map((post) => post.rkey),
        ['account-b'],
      );

      lateTimeline.complete(_timelinePage(postALate));
      lateUserPosts.completeError(StateError('late A read failed'));
      lateLike.completeError(StateError('late A like failed'));
      await Future.wait([lateLoadMore, lateRollback]);
      for (var index = 0; index < 5; index++) {
        await Future<void>.delayed(Duration.zero);
      }

      expect(
        container
            .read(timelineProvider)
            .requireValue
            .items
            .map((item) => item.post.rkey),
        ['account-b'],
      );
      expect(
        container
            .read(userPostsProvider('author.test'))
            .requireValue
            .items
            .map((post) => post.rkey),
        ['account-b'],
      );
      expect(container.read(toggleLikePostProvider).hasError, isFalse);
      expect(
        container.read(sessionRegistryProvider).requireValue.activeDid?.value,
        'did:plc:bob',
      );
    },
  );
}
