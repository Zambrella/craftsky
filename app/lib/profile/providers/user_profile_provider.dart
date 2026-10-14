import 'dart:async';

import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/business/providers/business_record_overlay.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/block_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/follow_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_record_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_profile_provider.g.dart';

/// Single source of truth for a user's profile, keyed by DID.
///
/// Accepted writes are composed through the shared record overlay while the
/// AppView projection catches up; this provider remains read-only.
@riverpod
class UserProfile extends _$UserProfile {
  @override
  Future<Profile> build(Did did) async {
    final accountOperation = ActiveAccountOperation.capture(ref);
    final ownership = accountOperation.ownership;
    final lease = ownership?.session;
    final readFence = lease == null
        ? null
        : captureBusinessProfileRead(ref, lease, did);
    late final Profile profile;
    try {
      final repository = ref.watch(profileRepositoryProvider);
      final activeDid = ref.watch(
        sessionRegistryProvider.select((state) => state.value?.activeDid),
      );
      profile = activeDid == did
          ? await repository.fetchMe()
          : await repository.fetch(did);
    } on Object {
      if (ref.mounted &&
          readFence != null &&
          !isBusinessRecordReadCurrent(ref, readFence) &&
          state.value != null) {
        return state.value!;
      }
      rethrow;
    }
    if (!accountOperation.isCurrent) {
      throw AccountOperationCanceled();
    }
    var reconciledProfile = profile;
    if (lease != null) {
      reconciledProfile = applyUserProfileRecordOverlays(ref, lease, profile);
      if (!isBusinessRecordReadCurrent(ref, readFence!) &&
          !hasBusinessProfileOverlay(ref, lease, profile.did) &&
          state.value != null) {
        return state.value!;
      }
    }
    return applyBlockProfileOverlay(
      ref.read,
      applyFollowProfileOverlay(ref, reconciledProfile),
    );
  }

  /// Refresh this read, retaining accepted fields while the request is pending.
  /// Riverpod preserves this value automatically when rebuilding.
  void refresh() {
    final lease = captureActiveAccountOperation(ref)?.session;
    final previous = state.value;
    if (lease != null && previous != null) {
      final pending = ref
          .read(pdsRecordOperationControllerProvider)
          .overlayFor(personalProfileMutationScope(lease, did))
          ?.optimisticValue;
      if (pending is AcceptedPersonalProfile) {
        // Retained data is not a server response and cannot retire an overlay.
        state = AsyncData(pending.applyTo(previous));
      }
    }
    ref.invalidateSelf();
  }

  /// Publishes completed AppView-private mutations. Public record saves are
  /// composed from the shared overlay by the read lifecycle.
  void setCached(Profile profile) => state = AsyncData(profile);
}

Profile applyUserProfileRecordOverlays(
  Ref ref,
  AccountSessionLease lease,
  Profile authoritative,
) {
  final personal = applyPersonalProfileOverlay(ref, lease, authoritative);
  return personal.copyWith(
    business: applyBusinessProfileOverlay(
      ref,
      lease,
      authoritative.did,
      authoritative.business,
    ),
  );
}
