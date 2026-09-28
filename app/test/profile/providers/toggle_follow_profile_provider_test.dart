import 'dart:async';

import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_follow_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_profile_repository.dart';

void main() {
  final bobDid = Did.parse('did:plc:bob');
  group('ToggleFollowProfile', () {
    test(
      'sets loading without speculative visible state while request is '
      'in flight',
      () async {
        final seed = Profile(
          did: 'did:plc:bob',
          handle: 'bob.craftsky.social',
          displayName: 'Bob',
          crafts: const [],
          followerCount: 4,
          followingCount: 2,
        );
        final completer = Completer<Profile>();
        final repo = FakeProfileRepository(
          onFetch: (_) async => seed,
          onFollow: (_) => completer.future,
        );
        final container = ProviderContainer.test(
          overrides: [profileRepositoryProvider.overrideWithValue(repo)],
        );
        final subscription = container.listen(
          userProfileProvider(bobDid),
          (_, _) {},
        );
        addTearDown(subscription.close);

        await container.read(userProfileProvider(bobDid).future);

        final toggle = container.read(toggleFollowProfileProvider.notifier);
        final pending = toggle.toggle(
          cacheKey: bobDid,
          profile: seed,
        );

        expect(container.read(toggleFollowProfileProvider).isLoading, isTrue);
        expect(
          container.read(userProfileProvider(bobDid)).value,
          isNotNull,
        );
        final optimistic = container.read(userProfileProvider(bobDid)).value!;
        expect(optimistic.viewerIsFollowing, isFalse);
        expect(optimistic.followerCount, 4);

        completer.complete(
          seed.copyWith(viewerIsFollowing: true, followerCount: 9),
        );
        await pending;
        await container.read(userProfileProvider(bobDid).future);

        final confirmed = container.read(userProfileProvider(bobDid)).value!;
        expect(confirmed.viewerIsFollowing, isTrue);
        expect(confirmed.followerCount, 5);
      },
    );

    test('retries ambiguity with one canonical operation key', () async {
      final seed = Profile(
        did: 'did:plc:bob',
        handle: 'bob.craftsky.social',
        crafts: const [],
      );
      var calls = 0;
      final repo = FakeProfileRepository(
        onFollow: (_) async {
          calls++;
          if (calls < 3) {
            throw const PdsMutationAmbiguousException(retryAfterSeconds: 2);
          }
          return seed.copyWith(viewerIsFollowing: true);
        },
      );
      final container = ProviderContainer.test(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repo),
          pdsMutationDelayProvider.overrideWithValue((_) async {}),
          pdsMutationJitterProvider.overrideWithValue((_) => 0),
        ],
      );

      await container
          .read(toggleFollowProfileProvider.notifier)
          .toggle(cacheKey: bobDid, profile: seed);

      expect(calls, 3);
      expect(repo.followOperationKeys.toSet(), hasLength(1));
      expect(
        isCanonicalPdsMutationOperationKey(repo.followOperationKeys.first),
        isTrue,
      );
      expect(
        container.read(toggleFollowProfileProvider).value?.viewerIsFollowing,
        isTrue,
      );
    });

    test('ignores a completion superseded by a newer sequence', () async {
      final seed = Profile(
        did: 'did:plc:bob',
        handle: 'bob.craftsky.social',
        crafts: const [],
      );
      final older = Completer<Profile>();
      final newer = Completer<Profile>();
      var calls = 0;
      final repo = FakeProfileRepository(
        onFollow: (_) {
          calls++;
          return calls == 1 ? older.future : newer.future;
        },
      );
      final container = ProviderContainer.test(
        overrides: [profileRepositoryProvider.overrideWithValue(repo)],
      );

      final first = container
          .read(toggleFollowProfileProvider.notifier)
          .toggle(cacheKey: bobDid, profile: seed);
      final second = container
          .read(toggleFollowProfileProvider.notifier)
          .toggle(cacheKey: bobDid, profile: seed);
      newer.complete(
        seed.copyWith(viewerIsFollowing: true, followerCount: 2),
      );
      await second;
      older.complete(
        seed.copyWith(viewerIsFollowing: true, followerCount: 1),
      );
      await first;

      expect(calls, 2);
      expect(
        container.read(toggleFollowProfileProvider).value?.followerCount,
        2,
      );
    });

    test(
      'masks stale reads, adjusts count once, and retires on agreement',
      () async {
        var authoritativeFollowing = false;
        Profile authoritative() => Profile(
          did: 'did:plc:bob',
          handle: 'bob.craftsky.social',
          crafts: const [],
          viewerIsFollowing: authoritativeFollowing,
          followerCount: authoritativeFollowing ? 5 : 4,
        );
        final repo = FakeProfileRepository(
          onFetch: (_) async => authoritative(),
          onFollow: (_) async => authoritative().copyWith(
            viewerIsFollowing: true,
            followerCount: 5,
          ),
        );
        final container = ProviderContainer.test(
          overrides: [profileRepositoryProvider.overrideWithValue(repo)],
        );
        final subscription = container.listen(
          userProfileProvider(bobDid),
          (_, _) {},
        );
        addTearDown(subscription.close);
        final seed = await container.read(userProfileProvider(bobDid).future);

        await container
            .read(toggleFollowProfileProvider.notifier)
            .toggle(cacheKey: bobDid, profile: seed);
        container.invalidate(userProfileProvider(bobDid));
        var stale = await container.read(userProfileProvider(bobDid).future);
        expect(stale.viewerIsFollowing, isTrue);
        expect(stale.followerCount, 5);

        container.invalidate(userProfileProvider(bobDid));
        stale = await container.read(userProfileProvider(bobDid).future);
        expect(stale.followerCount, 5);

        authoritativeFollowing = true;
        container.invalidate(userProfileProvider(bobDid));
        await container.read(userProfileProvider(bobDid).future);
        authoritativeFollowing = false;
        container.invalidate(userProfileProvider(bobDid));
        final afterAgreement = await container.read(
          userProfileProvider(bobDid).future,
        );
        expect(afterAgreement.viewerIsFollowing, isFalse);
        expect(afterAgreement.followerCount, 4);
      },
    );

    test(
      'stale unfollow count is floored at zero until overlay expiry',
      () async {
        var now = DateTime.utc(2026);
        final stale = Profile(
          did: 'did:plc:bob',
          handle: 'bob.craftsky.social',
          crafts: const [],
          viewerIsFollowing: true,
          followerCount: 0,
        );
        final controller = PdsRecordOperationController(now: () => now);
        final repo = FakeProfileRepository(
          onFetch: (_) async => stale,
          onUnfollow: (_) async => stale.copyWith(viewerIsFollowing: false),
        );
        final container = ProviderContainer.test(
          overrides: [
            profileRepositoryProvider.overrideWithValue(repo),
            pdsRecordOperationControllerProvider.overrideWithValue(controller),
          ],
        );
        final subscription = container.listen(
          userProfileProvider(bobDid),
          (_, _) {},
        );
        addTearDown(subscription.close);
        await container.read(userProfileProvider(bobDid).future);

        await container
            .read(toggleFollowProfileProvider.notifier)
            .toggle(cacheKey: bobDid, profile: stale);
        container.invalidate(userProfileProvider(bobDid));
        final masked = await container.read(userProfileProvider(bobDid).future);
        expect(masked.viewerIsFollowing, isFalse);
        expect(masked.followerCount, 0);

        now = now.add(const Duration(seconds: 31));
        controller.advanceTime();
        container.invalidate(userProfileProvider(bobDid));
        final expired = await container.read(
          userProfileProvider(bobDid).future,
        );
        expect(expired.viewerIsFollowing, isTrue);
        expect(expired.followerCount, 0);
      },
    );

    test('rolls back cache and surfaces error when follow fails', () async {
      final seed = Profile(
        did: 'did:plc:bob',
        handle: 'bob.craftsky.social',
        displayName: 'Bob',
        crafts: const [],
        followerCount: 4,
        followingCount: 2,
      );
      final repo = FakeProfileRepository(
        onFetch: (_) async => seed,
        onFollow: (_) async => throw Exception('boom'),
      );
      final container = ProviderContainer.test(
        overrides: [profileRepositoryProvider.overrideWithValue(repo)],
      );
      final subscription = container.listen(
        userProfileProvider(bobDid),
        (_, _) {},
      );
      addTearDown(subscription.close);

      await container.read(userProfileProvider(bobDid).future);
      await container
          .read(toggleFollowProfileProvider.notifier)
          .toggle(
            cacheKey: bobDid,
            profile: seed,
          );

      final current = container.read(userProfileProvider(bobDid)).value!;
      expect(current.viewerIsFollowing, isFalse);
      expect(current.followerCount, 4);
      expect(container.read(toggleFollowProfileProvider).hasError, isTrue);
    });

    test(
      'unfollow retains logical state while projection catches up',
      () async {
        final seed = Profile(
          did: 'did:plc:bob',
          handle: 'bob.craftsky.social',
          displayName: 'Bob',
          crafts: const [],
          viewerIsFollowing: true,
          followerCount: 4,
          followingCount: 2,
        );
        final completer = Completer<Profile>();
        final repo = FakeProfileRepository(
          onFetch: (_) async => seed,
          onUnfollow: (_) => completer.future,
        );
        final container = ProviderContainer.test(
          overrides: [profileRepositoryProvider.overrideWithValue(repo)],
        );
        final subscription = container.listen(
          userProfileProvider(bobDid),
          (_, _) {},
        );
        addTearDown(subscription.close);

        await container.read(userProfileProvider(bobDid).future);
        final pending = container
            .read(toggleFollowProfileProvider.notifier)
            .toggle(
              cacheKey: bobDid,
              profile: seed,
            );

        final pendingProfile = container
            .read(userProfileProvider(bobDid))
            .value!;
        expect(pendingProfile.viewerIsFollowing, isTrue);
        expect(pendingProfile.followerCount, 4);

        completer.complete(
          seed.copyWith(viewerIsFollowing: false, followerCount: 1),
        );
        await pending;
        await container.read(userProfileProvider(bobDid).future);

        final confirmed = container.read(userProfileProvider(bobDid)).value!;
        expect(confirmed.viewerIsFollowing, isFalse);
        expect(confirmed.followerCount, 3);
      },
    );
  });
}
