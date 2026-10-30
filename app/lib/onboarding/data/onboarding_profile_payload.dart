import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/profile/models/profile_link.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';

final class OnboardingProfilePayload {
  const new({
    required this.displayName,
    required this.pronouns,
    required this.description,
    required this.crafts,
    this.links,
    this.removedLinks,
    this.avatar,
    this.clearAvatar = false,
    this.clearBanner = false,
  });

  factory fromState(
    OnboardingFlowState state, {
    UploadedBlob? avatar,
  }) {
    final savesIdentity = state.step == OnboardingStep.profile;
    final savesLinks = savesIdentity && state.linksDirty;
    return OnboardingProfilePayload(
      displayName: savesIdentity
          ? state.identity.displayName.trim()
          : (state.baseline.displayName ?? '').trim(),
      pronouns: _optionalTrimmed(
        savesIdentity ? state.identity.pronouns : state.baseline.pronouns,
      ),
      description: savesIdentity
          ? state.identity.bio.trim()
          : (state.baseline.description ?? '').trim(),
      crafts: savesIdentity
          ? List.of(state.baseline.crafts)
          : [...state.selectedCraftIds, ...state.unknownCraftIds],
      links: savesLinks ? List.unmodifiable(state.identity.links) : null,
      removedLinks: savesLinks
          ? List.unmodifiable(
              state.baseline.editingLinks.where(
                (old) =>
                    old.uri != null &&
                    !state.identity.links.any((link) => link.uri == old.uri),
              ),
            )
          : null,
      avatar: savesIdentity ? avatar : null,
    );
  }

  final String displayName;
  final String? pronouns;
  final String description;
  final List<String> crafts;
  final List<ProfileLink>? links;
  final List<ProfileLink>? removedLinks;
  final UploadedBlob? avatar;
  final bool clearAvatar;
  final bool clearBanner;
}

String? _optionalTrimmed(String? value) {
  final trimmed = value?.trim() ?? '';
  return trimmed.isEmpty ? null : trimmed;
}
