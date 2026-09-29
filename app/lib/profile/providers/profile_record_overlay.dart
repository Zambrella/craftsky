import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

PdsMutationScope personalProfileMutationScope(
  AccountSessionLease lease,
  Did owner,
) => PdsMutationScope(
  lease: lease,
  identity: 'personal-profile:at://$owner/profile/self',
);

PdsCompoundProfileProjection personalProfileProjection(Profile profile) =>
    PdsCompoundProfileProjection(
      bluesky: PdsRecordProjection(
        uri: 'at://${profile.did}/app.bsky.actor.profile/self',
        cid: '',
        content: personalBlueskyProfileControlledContent(profile),
      ),
      craftsky: PdsRecordProjection(
        uri: 'at://${profile.did}/social.craftsky.actor.profile/self',
        cid: '',
        content: personalCraftskyProfileControlledContent(profile),
      ),
    );

Map<String, Object?> personalBlueskyProfileControlledContent(Profile profile) =>
    {
      'displayName': profile.displayName,
      'pronouns': profile.pronouns,
      'description': profile.description,
      'avatar': profile.avatar,
      'banner': profile.banner,
    };

Map<String, Object?> personalCraftskyProfileControlledContent(
  Profile profile,
) => {'crafts': profile.crafts};

Profile applyPersonalProfileOverlay(
  Ref ref,
  AccountSessionLease lease,
  Profile authoritative,
) {
  final controller = ref.read(pdsRecordOperationControllerProvider);
  final scope = personalProfileMutationScope(lease, authoritative.did);
  final overlay = controller.overlayFor(scope);
  if (overlay == null) return authoritative;
  if (controller.reconcile(scope, personalProfileProjection(authoritative))) {
    return authoritative;
  }
  final optimistic = overlay.optimisticValue;
  return optimistic is Profile ? optimistic : authoritative;
}
