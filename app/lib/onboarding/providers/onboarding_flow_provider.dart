import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/onboarding/data/onboarding_profile_payload.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/providers/profile_cache_publication.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'onboarding_flow_provider.g.dart';

final onboardingPrefillRetryDelaysProvider = Provider<List<Duration>>(
  (_) => const [
    Duration(milliseconds: 250),
    Duration(milliseconds: 500),
    Duration(seconds: 1),
    Duration(seconds: 1),
    Duration(seconds: 2),
  ],
);

final onboardingPrefillDeadlineProvider = Provider<Duration>(
  (_) => const Duration(seconds: 5),
);

final onboardingPrefillClockProvider = Provider<DateTime Function()>(
  (_) => DateTime.now,
);

@riverpod
class OnboardingFlow extends _$OnboardingFlow {
  @override
  Future<OnboardingFlowState> build(ActiveAccountLease lease) async {
    final registry = await ref.watch(sessionRegistryProvider.future);
    if (registry.activeLease != lease) {
      throw StateError('Active account changed');
    }
    final session = registry.sessions[lease.session.account.did];
    if (session == null) throw StateError('Active account changed');
    final initial = OnboardingFlowState.fromProfile(
      Profile(
        did: session.did.value,
        handle: session.handle.value,
        displayName: session.cachedDisplayName,
        avatar: session.cachedAvatarUrl,
        crafts: const [],
        customisation: session.cachedCustomisation,
      ),
    );
    final repository = ref.watch(
      accountProfileRepositoryProvider(lease).future,
    );
    unawaited(
      Future<void>.delayed(Duration.zero).then(
        (_) => _startPrefill(initial, repository),
      ),
    );
    return initial;
  }

  Future<void> _startPrefill(
    OnboardingFlowState initial,
    Future<ProfileRepository> repository,
  ) async {
    try {
      final resolved = await repository;
      if (!ref.mounted) return;
      await _prefill(initial, resolved);
    } on Object {
      // Prefill is best-effort; the editable draft is already available.
    }
  }

  Future<void> _prefill(
    OnboardingFlowState initial,
    ProfileRepository repository,
  ) async {
    if (!ref.mounted) return;
    final delays = ref.read(onboardingPrefillRetryDelaysProvider);
    final now = ref.read(onboardingPrefillClockProvider);
    final deadline = now().add(ref.read(onboardingPrefillDeadlineProvider));
    var profile = await repository.fetchMe().timeout(
      deadline.difference(now()),
    );
    for (final delay in delays) {
      if (_hasIdentity(profile)) break;
      var remaining = deadline.difference(now());
      if (remaining <= Duration.zero) break;
      await Future<void>.delayed(delay < remaining ? delay : remaining);
      if (!_isCurrent()) return;
      remaining = deadline.difference(now());
      if (remaining <= Duration.zero) break;
      try {
        profile = await repository.fetchMe().timeout(remaining);
      } on TimeoutException {
        break;
      }
    }
    if (!_isCurrent() || !identical(state.value, initial)) return;
    state = AsyncData(OnboardingFlowState.fromProfile(profile));
  }

  void updateIdentity({String? displayName, String? bio}) {
    final current = state.value;
    if (current == null || current.saving) return;
    state = AsyncData(
      current.copyWith(
        identity: current.identity.copyWith(
          displayName: displayName,
          bio: bio,
        ),
      ),
    );
  }

  void toggleCraft(String id) {
    final current = state.value;
    if (current == null || current.saving) return;
    final selected = current.selectedCraftIds.toSet();
    selected.contains(id) ? selected.remove(id) : selected.add(id);
    state = AsyncData(current.copyWith(selectedCraftIds: selected));
  }

  Future<void> pickAvatar() async {
    final current = state.value;
    if (current == null || current.saving || current.uploadingAvatar) return;
    try {
      final picker = await ref.read(
        accountProfileImagePickerProvider(lease).future,
      );
      final result = await picker.pickAndUpload(
        onPreviewReady: (bytes) {
          if (!_isCurrent()) return;
          state = AsyncData(
            (state.value ?? current).copyWith(
              avatarPreview: bytes,
              uploadingAvatar: true,
              avatarUploadFailed: false,
            ),
          );
        },
      );
      if (result == null || !_isCurrent()) return;
      state = AsyncData(
        (state.value ?? current).copyWith(
          avatarPreview: result.previewBytes,
          avatarBlob: result.uploaded.blob,
          uploadingAvatar: false,
          avatarUploadFailed: false,
        ),
      );
    } on Object {
      if (!_isCurrent()) return;
      state = AsyncData(
        (state.value ?? current).copyWith(
          uploadingAvatar: false,
          avatarUploadFailed: true,
        ),
      );
    }
  }

  void next() {
    final current = state.value;
    if (current == null || current.saving) return;
    final next = (current.step.index + 1).clamp(
      0,
      OnboardingStep.values.length - 1,
    );
    state = AsyncData(current.copyWith(step: OnboardingStep.values[next]));
  }

  void previous() {
    final current = state.value;
    if (current == null || current.saving) return;
    final previous = (current.step.index - 1).clamp(
      0,
      OnboardingStep.values.length - 1,
    );
    state = AsyncData(
      current.copyWith(step: OnboardingStep.values[previous]),
    );
  }

  Future<void> saveAndNext() async {
    final current = state.value;
    if (current == null || current.saving) return;
    state = AsyncData(current.copyWith(saving: true));
    try {
      final repository = await ref.read(
        accountProfileRepositoryProvider(lease).future,
      );
      final payload = OnboardingProfilePayload.fromState(
        current,
        avatar: current.avatarBlob,
      );
      final updated = await repository.updateMe(
        displayName: payload.displayName,
        description: payload.description,
        crafts: payload.crafts,
        avatar: payload.avatar,
      );
      if (!_isCurrent()) return;
      publishProfileCache(ref, updated);
      final saved = OnboardingFlowState.fromProfile(updated);
      final next = switch (current.step) {
        OnboardingStep.profile => saved.copyWith(
          step: OnboardingStep.crafts,
          selectedCraftIds: current.selectedCraftIds,
          unknownCraftIds: current.unknownCraftIds,
        ),
        OnboardingStep.crafts => saved.copyWith(
          step: OnboardingStep.instagram,
          identity: current.identity,
        ),
        OnboardingStep.instagram => saved.copyWith(
          step: OnboardingStep.instagram,
        ),
      };
      state = AsyncData(next);
    } on Object catch (error) {
      if (!_isCurrent()) return;
      state = AsyncData(current.copyWith(saveError: error));
    }
  }

  Future<void> complete() => ref
      .read(onboardingStatusProvider(lease.session).notifier)
      .completeOptimistically();

  bool _isCurrent() =>
      ref.mounted &&
      ref.read(sessionRegistryProvider).value?.activeLease == lease;

  static bool _hasIdentity(Profile profile) =>
      (profile.displayName?.isNotEmpty ?? false) ||
      (profile.description?.isNotEmpty ?? false) ||
      (profile.avatar?.isNotEmpty ?? false);
}
