import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_identity_cache_invalidator.dart';
import 'package:craftsky_app/profile/providers/profile_record_overlay.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Own refresh callbacks outside the short-lived onboarding/edit providers.
/// Poll within the shared overlay's 30-second lifetime, including when image
/// clearance has not completed at the first follow-up read.
final profileMutationPublicationProvider = Provider<ProfileMutationPublication>(
  ProfileMutationPublication.new,
);

final class ProfileMutationPublication {
  new(this._ref);

  final Ref _ref;

  void Function() refreshFor(AccountSessionLease lease) => () {
    if (!_ref.mounted ||
        _ref.read(sessionRegistryProvider).value?.activeLease?.session !=
            lease) {
      return;
    }
    _ref.invalidate(userProfileProvider);
    _ref.read(profileIdentityCacheInvalidatorProvider)();
  };

  bool acceptPersonal({
    required PdsMutationToken token,
    required Profile profile,
    UploadedBlob? avatar,
    UploadedBlob? banner,
    bool clearAvatar = false,
    bool clearBanner = false,
  }) {
    final controller = _ref.read(pdsRecordOperationControllerProvider);
    final pending = controller.overlayFor(token.scope)?.optimisticValue;
    final prior = pending is AcceptedPersonalProfile ? pending : null;
    final accepted = AcceptedPersonalProfile(
      profile,
      avatarCid: clearAvatar ? null : avatar?.ref.link ?? prior?.avatarCid,
      bannerCid: clearBanner ? null : banner?.ref.link ?? prior?.bannerCid,
      clearAvatar:
          clearAvatar || (avatar == null && prior?.clearAvatar == true),
      clearBanner:
          clearBanner || (banner == null && prior?.clearBanner == true),
    );
    final refresh = refreshFor(token.scope.lease);
    if (!controller.markAccepted(
      token,
      optimisticValue: accepted,
      agrees: accepted.agrees,
      refresh: refresh,
    )) {
      return false;
    }
    for (final seconds in const [5, 10, 20]) {
      controller.refreshAfter(token, Duration(seconds: seconds));
    }
    return true;
  }
}
