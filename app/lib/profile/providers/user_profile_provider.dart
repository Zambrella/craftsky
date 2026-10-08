import 'dart:async';

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
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
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
    final ownership = captureActiveAccountOperation(ref);
    final lease = ownership?.session;
    final readFence = lease == null
        ? null
        : captureBusinessProfileRead(ref, lease, did);
    late final Profile profile;
    try {
      final repository = ref.watch(profileRepositoryProvider);
      final activeDid = ref.watch(sessionRegistryProvider).value?.activeDid;
      profile = activeDid == did
          ? await repository.fetchMe()
          : await repository.fetch(did);
    } on Object {
      if (readFence != null &&
          !isBusinessRecordReadCurrent(ref, readFence) &&
          state.value != null) {
        return state.value!;
      }
      rethrow;
    }
    if (!isActiveAccountOperationCurrent(ref, ownership)) {
      throw DiagnosticStateError('Active account changed');
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

  /// Publishes completed AppView-private profile mutations. Public PDS record
  /// mutations use the shared operation overlay instead.
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
