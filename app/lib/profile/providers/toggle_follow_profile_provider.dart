import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/follow_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'toggle_follow_profile_provider.g.dart';

@riverpod
class ToggleFollowProfile extends _$ToggleFollowProfile {
  @override
  FutureOr<Profile?> build() => null;

  Future<void> toggle({
    required Did cacheKey,
    required Profile profile,
  }) async {
    final ownership = captureActiveAccountOperation(ref);
    final desiredFollowing = !profile.viewerIsFollowing;
    final endpoint = '/v1/profiles/@$cacheKey/follows';
    final immutableBody = desiredFollowing ? 'POST\n' : 'DELETE\n';
    final controller = ref.read(pdsRecordOperationControllerProvider);
    final scope = followProfileMutationScope(ref, cacheKey);
    final token = controller.beginOrRetry(
      scope: scope,
      endpoint: endpoint,
      immutableBody: immutableBody,
      newOperationKey: newPdsMutationOperationKey,
    );
    final operationKey = token.operationKey;
    final startedAt = ref.read(pdsMutationNowProvider)();
    var retryIndex = 0;

    state = const AsyncLoading<Profile?>();
    while (true) {
      late final Profile updated;
      try {
        final repo = ref.read(profileRepositoryProvider);
        updated = desiredFollowing
            ? await repo.follow(cacheKey, operationKey: operationKey)
            : await repo.unfollow(cacheKey, operationKey: operationKey);
      } on PdsMutationAmbiguousException catch (error) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return;
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
        if (delay == null) {
          state = AsyncError<Profile?>(
            const PdsMutationUnresolvedException(),
            StackTrace.current,
          );
          return;
        }
        retryIndex++;
        await ref.read(pdsMutationDelayProvider)(delay);
        if (!isActiveAccountOperationCurrent(ref, ownership) ||
            !controller.canRetry(
              token,
              operationKey: operationKey,
              endpoint: endpoint,
              immutableBody: immutableBody,
            )) {
          return;
        }
        continue;
      } on Object catch (error, stackTrace) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return;
        controller.markFailed(token);
        state = AsyncError<Profile?>(error, stackTrace);
        return;
      }

      if (!isActiveAccountOperationCurrent(ref, ownership)) return;
      final reconciliation = PdsSetReconciliation(active: desiredFollowing);
      if (!controller.markAccepted(
        token,
        optimisticValue: desiredFollowing,
        agrees: (value) => value is bool && reconciliation.agrees(value),
        refresh: () => invalidateFollowProfileReads(ref, cacheKey),
      )) {
        return;
      }
      state = AsyncData<Profile?>(updated);
      return;
    }
  }

  void reset() => state = const AsyncData(null);
}

void invalidateFollowProfileReads(Ref ref, Did targetDid) {
  if (!ref.mounted) return;
  final selfDid = ref.read(sessionRegistryProvider).value?.activeDid;
  ref
    ..invalidate(userProfileProvider(targetDid))
    ..invalidate(timelineProvider);
  if (selfDid != null && selfDid != targetDid) {
    ref.invalidate(userProfileProvider(selfDid));
  }
}
