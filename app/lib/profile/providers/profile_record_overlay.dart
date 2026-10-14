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
  if (optimistic is AcceptedPersonalProfile) {
    return optimistic.applyTo(authoritative);
  }
  return optimistic is Profile ? optimistic : authoritative;
}

/// Saved text fields plus explicit image changes still awaiting clearance.
/// The command response omits images; omission must never clear a cached image.
final class AcceptedPersonalProfile {
  const new(
    this.profile, {
    this.avatarCid,
    this.bannerCid,
    this.clearAvatar = false,
    this.clearBanner = false,
  });

  final Profile profile;
  final String? avatarCid;
  final String? bannerCid;
  final bool clearAvatar;
  final bool clearBanner;

  Profile applyTo(Profile authoritative) => authoritative.copyWith(
    displayName: profile.displayName,
    pronouns: profile.pronouns,
    description: profile.description,
    crafts: profile.crafts,
    avatar: clearAvatar ? null : authoritative.avatar,
    banner: clearBanner ? null : authoritative.banner,
  );

  bool agrees(Object? value) {
    if (value is! PdsCompoundProfileProjection) return false;
    final expected = personalProfileProjection(profile);
    final actualBluesky = value.bluesky;
    if (actualBluesky == null) return false;
    final expectedContent = Map<String, Object?>.of(expected.bluesky!.content);
    final actualContent = Map<String, Object?>.of(actualBluesky.content);
    for (final image in {'avatar': avatarCid, 'banner': bannerCid}.entries) {
      final clearing = image.key == 'avatar' ? clearAvatar : clearBanner;
      if (clearing && actualContent[image.key] != null) return false;
      if (image.value != null &&
          !_imageHasCid(actualContent[image.key], image.value!)) {
        return false;
      }
      // Unchanged images are read-owned. Explicit changes are confirmed above,
      // separately from the image-free save response's text fields.
      expectedContent.remove(image.key);
      actualContent.remove(image.key);
    }
    return PdsCompoundProfileReconciliation(
      bluesky: PdsFixedKeyReconciliation(
        uri: expected.bluesky!.uri,
        controlledContent: expectedContent,
      ),
      craftsky: PdsFixedKeyReconciliation(
        uri: expected.craftsky!.uri,
        controlledContent: expected.craftsky!.content,
      ),
    ).agrees(
      PdsCompoundProfileProjection(
        bluesky: PdsRecordProjection(
          uri: actualBluesky.uri,
          cid: actualBluesky.cid,
          content: actualContent,
        ),
        craftsky: value.craftsky,
      ),
    );
  }
}

bool _imageHasCid(Object? value, String cid) {
  if (value is! String) return false;
  final uri = Uri.tryParse(value);
  return uri != null &&
      uri.pathSegments.isNotEmpty &&
      uri.pathSegments.last.startsWith('$cid@');
}
