import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/misc.dart' show ProviderListenable;

typedef BlockOverlayReader = T Function<T>(ProviderListenable<T> provider);

PdsMutationScope blockProfileMutationScope(
  BlockOverlayReader read,
  Did targetDid,
) {
  final lease =
      read(sessionRegistryProvider).value?.activeLease?.session ??
      AccountSessionLease(
        account: AccountKey(targetDid.toString()),
        sessionGeneration: 0,
      );
  return blockProfileMutationScopeFor(lease, targetDid);
}

PdsMutationScope blockProfileMutationScopeFor(
  AccountSessionLease lease,
  Did targetDid,
) => PdsMutationScope(lease: lease, identity: 'block:$targetDid');

ProfileRelationship applyBlockRelationshipOverlay(
  BlockOverlayReader read,
  ProfileRelationship relationship,
  Did targetDid,
) => applyBlockRelationshipOverlayForScope(
  read(pdsRecordOperationControllerProvider),
  blockProfileMutationScope(read, targetDid),
  relationship,
);

bool isLogicallyBlocking(
  BlockOverlayReader read,
  Did targetDid, {
  required bool authoritativeBlocking,
}) => applyBlockRelationshipOverlay(
  read,
  ProfileRelationship(
    blocking: authoritativeBlocking,
    initialized: true,
  ),
  targetDid,
).blocking;

ProfileRelationship applyBlockRelationshipOverlayForScope(
  PdsRecordOperationController controller,
  PdsMutationScope scope,
  ProfileRelationship relationship,
) {
  final overlay = controller.overlayFor(scope);
  if (overlay == null) return relationship;
  if (controller.reconcile(scope, relationship.blocking)) return relationship;
  final desired = overlay.optimisticValue;
  return desired is bool && desired != relationship.blocking
      ? relationship.copyWith(blocking: desired, initialized: true)
      : relationship;
}

Profile applyBlockProfileOverlay(BlockOverlayReader read, Profile profile) {
  final relationship = applyBlockRelationshipOverlay(
    read,
    ProfileRelationship.fromProfileFlags(
      muted: profile.muted,
      blocking: profile.blocking,
      blockedBy: profile.blockedBy,
    ),
    profile.did,
  );
  return relationship.blocking == profile.blocking
      ? profile
      : profile.copyWith(blocking: relationship.blocking);
}

ProfileAccountSummary applyBlockAccountSummaryOverlay(
  BlockOverlayReader read,
  ProfileAccountSummary account,
) {
  final relationship = applyBlockRelationshipOverlay(
    read,
    ProfileRelationship.fromProfileFlags(
      muted: account.muted,
      blocking: account.blocking,
      blockedBy: account.blockedBy,
    ),
    account.did,
  );
  return relationship.blocking == account.blocking
      ? account
      : account.copyWith(blocking: relationship.blocking);
}
