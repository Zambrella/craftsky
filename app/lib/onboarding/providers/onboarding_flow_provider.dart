import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/onboarding/data/onboarding_profile_payload.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_image_failure.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/profile/providers/profile_mutation_publication.dart';
import 'package:craftsky_app/profile/providers/profile_record_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/services/profile_image_error_reporting.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
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
    // Cached identity changes must not reset the in-progress onboarding draft.
    await ref.read(sessionRegistryProvider.future);
    if (!ref.mounted) throw AccountOperationCanceled();
    ref.watch(
      sessionRegistryProvider.select((state) => state.value?.activeLease),
    );
    final registry = ref.read(sessionRegistryProvider).requireValue;
    if (registry.activeLease != lease) {
      throw DiagnosticStateError('Active account changed');
    }
    final session = registry.sessions[lease.session.account.did];
    if (session == null) throw DiagnosticStateError('Active account changed');
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
      Future<void>.delayed(Duration.zero)
          .then((_) => _startPrefill(initial, repository)),
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
    final reporter = ref.read(errorReporterProvider);
    final initiatingAccountDid = lease.session.account.did.value;
    // Selection can span many frames. Retain the provider that owns the upload
    // client until the operation finishes; ref disposal still cancels it.
    state = AsyncData(
      current.copyWith(uploadingAvatar: true, avatarUploadFailed: false),
    );
    final pickerProvider = accountProfileImagePickerProvider(lease);
    final pickerSubscription = ref.listen(pickerProvider, (_, _) {});
    try {
      final picker = await ref.read(pickerProvider.future);
      if (!_isCurrent()) return;
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
      if (!_isCurrent()) return;
      if (result == null) {
        state = AsyncData(
          (state.value ?? current).copyWith(uploadingAvatar: false),
        );
        return;
      }
      state = AsyncData(
        (state.value ?? current).copyWith(
          avatarPreview: result.previewBytes,
          avatarBlob: result.uploaded.blob,
          uploadingAvatar: false,
          avatarUploadFailed: false,
        ),
      );
    } on Object catch (error, stack) {
      unawaited(
        reportProfileImageFailure(
          reporter: reporter,
          error: error,
          stackTrace: stack,
          operation: ProfileImageOperation.onboarding,
          accountDid: initiatingAccountDid,
        ),
      );
      if (!_isCurrent()) return;
      state = AsyncData(
        (state.value ?? current).copyWith(
          uploadingAvatar: false,
          avatarUploadFailed: true,
          avatarFailure: error is ProfileImagePickException
              ? error.reason
              : ProfileImageFailure.unknown,
        ),
      );
    } finally {
      pickerSubscription.close();
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
      if (!_isCurrent()) return;
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
      if (!ref
          .read(profileMutationPublicationProvider)
          .acceptPersonal(
            token: token,
            profile: updated,
            avatar: payload.avatar,
          )) {
        return;
      }
      final saved = OnboardingFlowState.fromProfile(
        AcceptedPersonalProfile(updated).applyTo(current.baseline),
      );
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
    final retry = PdsMutationRetry(
      ref: ref,
      controller: controller,
      token: token,
      isCurrent: _isCurrent,
      requireCurrentToken: false,
    );
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
        final decision = await retry.wait(error);
        if (!_isCurrent()) throw const _OnboardingMutationSuperseded();
        switch (decision) {
          case PdsMutationRetryDecision.obsolete:
            throw const _OnboardingMutationSuperseded();
          case PdsMutationRetryDecision.exhausted:
            rethrow;
          case PdsMutationRetryDecision.retry:
            continue;
        }
      } on Object {
        controller.markFailed(token);
        rethrow;
      }
    }
  }

  bool _isCurrent() => ActiveAccountOperation.fromLease(ref, lease).isCurrent;

  static bool _hasIdentity(Profile profile) =>
      (profile.displayName?.isNotEmpty ?? false) ||
      (profile.pronouns?.isNotEmpty ?? false) ||
      (profile.description?.isNotEmpty ?? false) ||
      (profile.avatar?.isNotEmpty ?? false);
}

Map<String, dynamic> _profileMutationBody(OnboardingProfilePayload payload) => {
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
  const new();
}
