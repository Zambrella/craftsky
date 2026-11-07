import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// An indexed profile can catch up after a concurrent feed snapshot was read.
/// Dependent feeds must take a new read after this confirmation, even when the
/// profile confirmation retires the accepted-write overlay and its timers.
final followProfileProjectionRevisionProvider = Provider<Object>((ref) {
  ref.watch(
    sessionRegistryProvider.select((state) => state.value?.activeLease),
  );
  return Object();
});

PdsMutationScope followProfileMutationScope(Ref ref, Did targetDid) {
  final lease =
      captureActiveAccountOperation(ref)?.session ??
      AccountSessionLease(
        account: AccountKey(targetDid.toString()),
        sessionGeneration: 0,
      );
  return PdsMutationScope(lease: lease, identity: 'follow:$targetDid');
}

/// Accepted owner intent only; rendering a list cannot reconcile a follow.
bool? acceptedFollowProfileIntent(Ref ref, Did targetDid) {
  final scope = followProfileMutationScope(ref, targetDid);
  final desired = ref
      .read(pdsRecordOperationControllerProvider)
      .overlayFor(scope)
      ?.optimisticValue;
  return desired is bool ? desired : null;
}

Profile applyFollowProfileOverlay(Ref ref, Profile profile) {
  final controller = ref.read(pdsRecordOperationControllerProvider);
  final scope = followProfileMutationScope(ref, profile.did);
  final overlay = controller.overlayFor(scope);
  if (overlay == null) return profile;
  if (controller.reconcile(scope, profile.viewerIsFollowing)) {
    final operation = ActiveAccountOperation.capture(ref);
    // Profile reads can complete during provider construction. Publish after
    // construction, retaining the originating account/lifetime authority.
    scheduleMicrotask(() {
      operation.publish(
        () => ref.invalidate(followProfileProjectionRevisionProvider),
      );
    });
    return profile;
  }
  final desired = overlay.optimisticValue;
  if (desired is! bool || desired == profile.viewerIsFollowing) return profile;
  final count = profile.followerCount;
  return profile.copyWith(
    viewerIsFollowing: desired,
    followerCount: count == null
        ? null
        : desired
        ? count + 1
        : (count > 0 ? count - 1 : 0),
  );
}
