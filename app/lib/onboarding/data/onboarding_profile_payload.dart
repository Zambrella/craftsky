import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';

final class OnboardingProfilePayload {
  const OnboardingProfilePayload({
    required this.displayName,
    required this.description,
    required this.crafts,
    this.avatar,
    this.clearAvatar = false,
    this.clearBanner = false,
  });

  factory OnboardingProfilePayload.fromState(
    OnboardingFlowState state, {
    UploadedBlob? avatar,
  }) {
    final savesIdentity = state.step == OnboardingStep.profile;
    final selectedCraftIds = savesIdentity
        ? state.baseline.crafts.toSet()
        : state.selectedCraftIds;
    return OnboardingProfilePayload(
      displayName: savesIdentity
          ? state.identity.displayName.trim()
          : (state.baseline.displayName ?? '').trim(),
      description: savesIdentity
          ? state.identity.bio.trim()
          : (state.baseline.description ?? '').trim(),
      crafts: [
        for (final craft in Craft.values)
          if (selectedCraftIds.contains(craft.id)) craft.id,
        ...state.unknownCraftIds,
      ],
      avatar: savesIdentity ? avatar : null,
    );
  }

  final String displayName;
  final String description;
  final List<String> crafts;
  final UploadedBlob? avatar;
  final bool clearAvatar;
  final bool clearBanner;
}
