import 'dart:async';

import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_follow_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_profile_repository.dart';

void main() {
  group('ToggleFollowProfile', () {
    test(
      'sets loading and optimistic cache while request is in flight',
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

        await container.read(userProfileProvider('bob.craftsky.social').future);

        final toggle = container.read(toggleFollowProfileProvider.notifier);
        final pending = toggle.toggle(
          cacheKey: 'bob.craftsky.social',
          profile: seed,
        );

        expect(container.read(toggleFollowProfileProvider).isLoading, isTrue);
        expect(
          container.read(userProfileProvider('bob.craftsky.social')).value,
          isNotNull,
        );
        final optimistic = container
            .read(userProfileProvider('bob.craftsky.social'))
            .value!;
        expect(optimistic.viewerIsFollowing, isTrue);
        expect(optimistic.followerCount, 5);

        completer.complete(
          seed.copyWith(viewerIsFollowing: true, followerCount: 9),
        );
        await pending;

        final confirmed = container
            .read(userProfileProvider('bob.craftsky.social'))
            .value!;
        expect(confirmed.viewerIsFollowing, isTrue);
        expect(confirmed.followerCount, 9);
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

      await container.read(userProfileProvider('bob.craftsky.social').future);
      await container
          .read(toggleFollowProfileProvider.notifier)
          .toggle(
            cacheKey: 'bob.craftsky.social',
            profile: seed,
          );

      final current = container
          .read(userProfileProvider('bob.craftsky.social'))
          .value!;
      expect(current.viewerIsFollowing, isFalse);
      expect(current.followerCount, 4);
      expect(container.read(toggleFollowProfileProvider).hasError, isTrue);
    });

    test(
      'unfollow updates optimistic state then confirms server response',
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

        await container.read(userProfileProvider('bob.craftsky.social').future);
        final pending = container
            .read(toggleFollowProfileProvider.notifier)
            .toggle(
              cacheKey: 'bob.craftsky.social',
              profile: seed,
            );

        final optimistic = container
            .read(userProfileProvider('bob.craftsky.social'))
            .value!;
        expect(optimistic.viewerIsFollowing, isFalse);
        expect(optimistic.followerCount, 3);

        completer.complete(
          seed.copyWith(viewerIsFollowing: false, followerCount: 1),
        );
        await pending;

        final confirmed = container
            .read(userProfileProvider('bob.craftsky.social'))
            .value!;
        expect(confirmed.viewerIsFollowing, isFalse);
        expect(confirmed.followerCount, 1);
      },
    );
  });
}
