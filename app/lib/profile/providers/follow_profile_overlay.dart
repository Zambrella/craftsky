import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

PdsMutationScope followProfileMutationScope(Ref ref, Did targetDid) {
  final lease =
      captureActiveAccountOperation(ref)?.session ??
      AccountSessionLease(
        account: AccountKey(targetDid.toString()),
        sessionGeneration: 0,
      );
  return PdsMutationScope(lease: lease, identity: 'follow:$targetDid');
}

Profile applyFollowProfileOverlay(Ref ref, Profile profile) {
  final controller = ref.read(pdsRecordOperationControllerProvider);
  final scope = followProfileMutationScope(ref, profile.did);
  final overlay = controller.overlayFor(scope);
  if (overlay == null) return profile;
  if (controller.reconcile(scope, profile.viewerIsFollowing)) return profile;
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
