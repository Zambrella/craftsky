import 'dart:typed_data';

import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';

enum OnboardingStep { profile, crafts, instagram }

extension OnboardingStepProgress on OnboardingStep {
  int get number => index + 1;
  double get progress => number / OnboardingStep.values.length;
}

final class OnboardingIdentityDraft {
  const OnboardingIdentityDraft({
    required this.displayName,
    required this.bio,
  });

  final String displayName;
  final String bio;

  OnboardingIdentityDraft copyWith({String? displayName, String? bio}) =>
      OnboardingIdentityDraft(
        displayName: displayName ?? this.displayName,
        bio: bio ?? this.bio,
      );
}

final class OnboardingFlowState {
  OnboardingFlowState({
    required this.step,
    required this.baseline,
    required this.identity,
    required Set<String> selectedCraftIds,
    required List<String> unknownCraftIds,
    this.saving = false,
    this.uploadingAvatar = false,
    this.avatarUploadFailed = false,
    this.saveError,
    this.avatarPreview,
    this.avatarBlob,
  }) : selectedCraftIds = Set.unmodifiable(selectedCraftIds),
       unknownCraftIds = List.unmodifiable(unknownCraftIds);

  factory OnboardingFlowState.fromProfile(Profile profile) {
    final known = <String>{};
    final unknown = <String>[];
    for (final id in profile.crafts) {
      if (Craft.fromId(id) == null) {
        unknown.add(id);
      } else {
        known.add(id);
      }
    }
    return OnboardingFlowState(
      step: OnboardingStep.profile,
      baseline: profile,
      identity: OnboardingIdentityDraft(
        displayName: profile.displayName ?? '',
        bio: profile.description ?? '',
      ),
      selectedCraftIds: known,
      unknownCraftIds: unknown,
    );
  }

  final OnboardingStep step;
  final Profile baseline;
  final OnboardingIdentityDraft identity;
  final Set<String> selectedCraftIds;
  final List<String> unknownCraftIds;
  final bool saving;
  final bool uploadingAvatar;
  final bool avatarUploadFailed;
  final Object? saveError;
  final Uint8List? avatarPreview;
  final UploadedBlob? avatarBlob;

  bool get identityDirty =>
      identity.displayName != (baseline.displayName ?? '') ||
      identity.bio != (baseline.description ?? '') ||
      avatarBlob != null;

  bool get craftsDirty {
    final baselineKnown = baseline.crafts.where(
      (id) => Craft.fromId(id) != null,
    );
    return baselineKnown.toSet().length != selectedCraftIds.length ||
        !selectedCraftIds.containsAll(baselineKnown);
  }

  OnboardingFlowState copyWith({
    OnboardingStep? step,
    Profile? baseline,
    OnboardingIdentityDraft? identity,
    Set<String>? selectedCraftIds,
    List<String>? unknownCraftIds,
    bool? saving,
    bool? uploadingAvatar,
    bool? avatarUploadFailed,
    Object? saveError,
    Uint8List? avatarPreview,
    UploadedBlob? avatarBlob,
  }) => OnboardingFlowState(
    step: step ?? this.step,
    baseline: baseline ?? this.baseline,
    identity: identity ?? this.identity,
    selectedCraftIds: selectedCraftIds ?? this.selectedCraftIds,
    unknownCraftIds: unknownCraftIds ?? this.unknownCraftIds,
    saving: saving ?? this.saving,
    uploadingAvatar: uploadingAvatar ?? this.uploadingAvatar,
    avatarUploadFailed: avatarUploadFailed ?? this.avatarUploadFailed,
    saveError: saveError,
    avatarPreview: avatarPreview ?? this.avatarPreview,
    avatarBlob: avatarBlob ?? this.avatarBlob,
  );
}
