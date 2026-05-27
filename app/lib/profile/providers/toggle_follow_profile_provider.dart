import 'dart:async';

import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'toggle_follow_profile_provider.g.dart';

@riverpod
class ToggleFollowProfile extends _$ToggleFollowProfile {
  @override
  FutureOr<Profile?> build() => null;

  Future<void> toggle({
    required String cacheKey,
    required Profile profile,
  }) async {
    final optimistic = profile.copyWith(
      viewerIsFollowing: !profile.viewerIsFollowing,
      followerCount: _optimisticFollowerCount(profile),
    );

    if (ref.exists(userProfileProvider(cacheKey))) {
      ref.read(userProfileProvider(cacheKey).notifier).setCached(optimistic);
    }

    state = const AsyncLoading();
    try {
      final repo = ref.read(profileRepositoryProvider);
      final updated = profile.viewerIsFollowing
          ? await repo.unfollow(cacheKey)
          : await repo.follow(cacheKey);
      if (!ref.mounted) return;
      if (ref.exists(userProfileProvider(cacheKey))) {
        ref.read(userProfileProvider(cacheKey).notifier).setCached(updated);
      }
      state = AsyncData(updated);
    } on Object catch (error, stackTrace) {
      if (!ref.mounted) return;
      if (ref.exists(userProfileProvider(cacheKey))) {
        ref.read(userProfileProvider(cacheKey).notifier).setCached(profile);
      }
      state = AsyncError(error, stackTrace);
    }
  }

  int? _optimisticFollowerCount(Profile profile) {
    final count = profile.followerCount;
    if (count == null || !profile.isCraftskyProfile) return count;
    if (profile.viewerIsFollowing) {
      return count > 0 ? count - 1 : 0;
    }
    return count + 1;
  }

  void reset() => state = const AsyncData(null);
}
