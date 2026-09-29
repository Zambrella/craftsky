import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/onboarding/data/onboarding_profile_payload.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/profile/providers/profile_record_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
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
      Future<void>.delayed(
        Duration.zero,
      ).then((_) => _startPrefill(initial, repository)),
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

  void updateIdentity({String? displayName, String? pronouns, String? bio}) {
    final current = state.value;
    if (current == null || current.saving) return;
    state = AsyncData(
      current.copyWith(
        identity: current.identity.copyWith(
          displayName: displayName,
          pronouns: pronouns,
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

  // This method is passed directly as a ValueChanged<bool> callback.
  // ignore: avoid_positional_boolean_parameters
  void setMeetsMinimumAge(bool value) {
    final current = state.value;
    if (current == null || current.saving) return;
    state = AsyncData(current.copyWith(meetsMinimumAge: value));
  }

  Future<void> pickAvatar(ImageSource source) async {
    final current = state.value;
    if (current == null || current.saving || current.uploadingAvatar) return;
    try {
      final picker = await ref.read(
        accountProfileImagePickerProvider(lease).future,
      );
      final result = await picker.pickAndUpload(
        source: source,
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
    state = AsyncData(current.copyWith(step: OnboardingStep.values[previous]));
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
      const endpoint = '/v1/profiles/me';
      final immutableBody = jsonEncode(_profileMutationBody(payload));
      final controller = ref.read(pdsRecordOperationControllerProvider);
      final token = controller.beginOrRetry(
        scope: personalProfileMutationScope(
          lease.session,
          lease.session.account.did,
        ),
        endpoint: endpoint,
        immutableBody: immutableBody,
        newOperationKey: newPdsMutationOperationKey,
      );
      final operationKey = token.operationKey;
      final updated = await _updateProfileWithRetry(
        repository: repository,
        payload: payload,
        operationKey: operationKey,
        endpoint: endpoint,
        immutableBody: immutableBody,
        controller: controller,
        token: token,
      );
      if (!_isCurrent()) return;
      final projection = personalProfileProjection(updated);
      final reconciliation = PdsCompoundProfileReconciliation(
        bluesky: PdsFixedKeyReconciliation(
          uri: projection.bluesky!.uri,
          controlledContent: projection.bluesky!.content,
        ),
        craftsky: PdsFixedKeyReconciliation(
          uri: projection.craftsky!.uri,
          controlledContent: projection.craftsky!.content,
        ),
      );
      if (!controller.markAccepted(
        token,
        optimisticValue: updated,
        agrees: (value) => reconciliation.agrees(
          value is PdsCompoundProfileProjection ? value : null,
        ),
        refresh: _invalidatePersonalProfileReads,
      )) {
        return;
      }
      final saved = OnboardingFlowState.fromProfile(updated);
      final next = switch (current.step) {
        OnboardingStep.profile => saved.copyWith(
          step: OnboardingStep.crafts,
          selectedCraftIds: current.selectedCraftIds,
          unknownCraftIds: current.unknownCraftIds,
        ),
        OnboardingStep.crafts => saved.copyWith(
          step: OnboardingStep.guidelines,
          identity: current.identity,
        ),
        OnboardingStep.guidelines => saved.copyWith(
          step: OnboardingStep.guidelines,
        ),
      };
      state = AsyncData(next);
    } on Object catch (error) {
      if (!_isCurrent()) return;
      state = AsyncData(current.copyWith(saveError: error));
    }
  }

  Future<void> complete() {
    final current = state.value;
    if (current == null || !current.meetsMinimumAge) return Future.value();
    return ref
        .read(onboardingStatusProvider(lease.session).notifier)
        .completeOptimistically(meetsMinimumAge: true);
  }

  Future<Profile> _updateProfileWithRetry({
    required ProfileRepository repository,
    required OnboardingProfilePayload payload,
    required String operationKey,
    required String endpoint,
    required String immutableBody,
    required PdsRecordOperationController controller,
    required PdsMutationToken token,
  }) async {
    final startedAt = ref.read(pdsMutationNowProvider)();
    var retryIndex = 0;
    while (true) {
      try {
        return await repository.updateMe(
          operationKey: operationKey,
          displayName: payload.displayName,
          pronouns: payload.pronouns,
          description: payload.description,
          crafts: payload.crafts,
          avatar: payload.avatar,
        );
      } on PdsMutationAmbiguousException catch (error) {
        if (!_isCurrent()) throw const _OnboardingMutationSuperseded();
        controller.markAmbiguous(
          token,
          retryAfterSeconds: error.retryAfterSeconds,
        );
        final delay = const PdsMutationRetryPolicy().nextDelay(
          retryIndex: retryIndex,
          retryAfterSeconds: error.retryAfterSeconds,
          elapsed: ref.read(pdsMutationNowProvider)().difference(startedAt),
          jitterMillis: ref.read(pdsMutationJitterProvider),
        );
        if (delay == null) rethrow;
        retryIndex++;
        await ref.read(pdsMutationDelayProvider)(delay);
        if (!_isCurrent() ||
            !controller.canRetry(
              token,
              operationKey: operationKey,
              endpoint: endpoint,
              immutableBody: immutableBody,
            )) {
          throw const _OnboardingMutationSuperseded();
        }
      } on Object {
        controller.markFailed(token);
        rethrow;
      }
    }
  }

  void _invalidatePersonalProfileReads() {
    if (!ref.mounted) return;
    ref.invalidate(userProfileProvider);
  }

  bool _isCurrent() =>
      ref.mounted &&
      ref.read(sessionRegistryProvider).value?.activeLease == lease;

  static bool _hasIdentity(Profile profile) =>
      (profile.displayName?.isNotEmpty ?? false) ||
      (profile.pronouns?.isNotEmpty ?? false) ||
      (profile.description?.isNotEmpty ?? false) ||
      (profile.avatar?.isNotEmpty ?? false);
}

Map<String, dynamic> _profileMutationBody(
  OnboardingProfilePayload payload,
) => {
  'displayName': payload.displayName,
  'pronouns': payload.pronouns,
  'description': payload.description,
  'crafts': payload.crafts,
  if (payload.avatar case final avatar?)
    'avatar': {
      r'$type': avatar.type,
      'ref': {r'$link': avatar.ref.link},
      'mimeType': avatar.mimeType,
      'size': avatar.size,
    },
};

final class _OnboardingMutationSuperseded implements Exception {
  const _OnboardingMutationSuperseded();
}
