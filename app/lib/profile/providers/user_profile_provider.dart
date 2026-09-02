import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_profile_provider.g.dart';

/// Single source of truth for a user's profile, keyed by handle or DID.
///
/// Holds both read and write logic so mutation methods can perform
/// optimistic updates against the cached `AsyncData` and roll back on
/// failure. Mutations only succeed against the authenticated user's
/// profile — the AppView rejects writes against any other DID — so
/// callers should only invoke them on the family entry that matches
/// the signed-in user.
///
/// Mixing handle and DID for the same user produces separate cache
/// entries; pick one form per call site.
@riverpod
class UserProfile extends _$UserProfile {
  @override
  Future<Profile> build(String handleOrDid) async {
    final ownership = captureActiveAccountOperation(ref);
    final lease = ownership?.session;
    final overlay = ref.read(businessProjectionOverlayProvider.notifier);
    final readFence = lease == null ? null : overlay.captureRead(lease);
    late final Profile profile;
    try {
      profile = await ref.watch(profileRepositoryProvider).fetch(handleOrDid);
    } on Object {
      if (readFence != null &&
          !overlay.isReadCurrent(readFence) &&
          state.value != null) {
        return state.value!;
      }
      rethrow;
    }
    if (!isActiveAccountOperationCurrent(ref, ownership)) {
      throw StateError('Active account changed');
    }
    if (lease == null) return profile;
    final key = BusinessProjectionKey.declaration(lease.account, profile.did);
    final reconciliation = overlay.reconcile<BusinessProfile>(
      key: key,
      fence: readFence!,
      authoritativeCid: profile.business?.cid,
      authoritativeView: profile.business,
    );
    if (reconciliation.isStale && state.value != null) {
      return state.value!;
    }
    return profile.copyWith(business: reconciliation.view);
  }

  /// Replaces the cached profile without refetching. Used by mutation
  /// flows (`SaveProfile.save`) that already have an authoritative
  /// server response in hand — pushing it straight into the cache
  /// avoids both the round-trip and any read-after-write lag on the
  /// AppView.
  void setCached(Profile profile) => state = AsyncData(profile);

  Future<void> updateDisplayName(String displayName) => _patch(
    optimistic: (p) => p.copyWith(displayName: displayName),
    apply: (repo) => repo.updateMe(displayName: displayName),
  );

  Future<void> updateDescription(String description) => _patch(
    optimistic: (p) => p.copyWith(description: description),
    apply: (repo) => repo.updateMe(description: description),
  );

  Future<void> updateCrafts(List<String> crafts) => _patch(
    optimistic: (p) => p.copyWith(crafts: crafts),
    apply: (repo) => repo.updateMe(crafts: crafts),
  );

  /// Shared optimistic-update plumbing. No-ops when the profile hasn't
  /// loaded yet — there's nothing to optimistically apply or roll back.
  /// On failure, restores the previous state then surfaces the error so
  /// listeners can react via `ref.listen`.
  Future<void> _patch({
    required Profile Function(Profile current) optimistic,
    required Future<Profile> Function(ProfileRepository repo) apply,
  }) async {
    final previous = state;
    final current = previous.value;
    if (current == null) return;
    final ownership = captureActiveAccountOperation(ref);

    state = AsyncData(optimistic(current));
    try {
      final updated = await apply(ref.read(profileRepositoryProvider));
      if (!isActiveAccountOperationCurrent(ref, ownership)) return;
      state = AsyncData(updated);
    } on Object catch (e, st) {
      if (!isActiveAccountOperationCurrent(ref, ownership)) return;
      state = previous;
      state = AsyncError(e, st);
    }
  }
}
