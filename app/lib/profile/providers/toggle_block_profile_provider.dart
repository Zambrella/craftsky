import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/profile/providers/block_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/search/providers/blank_search_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'toggle_block_profile_provider.g.dart';

@riverpod
class ToggleBlockProfile extends _$ToggleBlockProfile {
  @override
  FutureOr<bool?> build() => null;

  Future<void> toggle({
    required Did targetDid,
    required bool isBlocking,
  }) async {
    final ownership = captureActiveAccountOperation(ref);
    final desiredBlocking = !isBlocking;
    final endpoint = '/v1/profiles/@$targetDid/blocks';
    final immutableBody = desiredBlocking ? 'POST\n' : 'DELETE\n';
    final controller = ref.read(pdsRecordOperationControllerProvider);
    final scope = blockProfileMutationScope(ref.read, targetDid);
    final token = controller.beginOrRetry(
      scope: scope,
      endpoint: endpoint,
      immutableBody: immutableBody,
      newOperationKey: newPdsMutationOperationKey,
    );
    final operationKey = token.operationKey;
    final startedAt = ref.read(pdsMutationNowProvider)();
    var retryIndex = 0;

    state = const AsyncLoading<bool?>();
    while (true) {
      try {
        final repository = ref.read(profileRepositoryProvider);
        if (desiredBlocking) {
          await repository.block(targetDid, operationKey: operationKey);
        } else {
          await repository.unblock(targetDid, operationKey: operationKey);
        }
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
          state = AsyncError<bool?>(
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
        state = AsyncError<bool?>(error, stackTrace);
        return;
      }

      if (!isActiveAccountOperationCurrent(ref, ownership)) return;
      final reconciliation = PdsSetReconciliation(active: desiredBlocking);
      if (!controller.markAccepted(
        token,
        optimisticValue: desiredBlocking,
        agrees: (value) => value is bool && reconciliation.agrees(value),
        refresh: () => _invalidateBlockReads(
          ref,
          targetDid,
          ownership?.session.account,
        ),
      )) {
        return;
      }
      state = AsyncData<bool?>(desiredBlocking);
      return;
    }
  }

  void reset() => state = const AsyncData(null);
}

void _invalidateBlockReads(Ref ref, Did targetDid, AccountKey? account) {
  if (!ref.mounted) return;
  ref
    ..invalidate(userProfileProvider(targetDid))
    ..invalidate(timelineProvider)
    ..invalidate(blankSearchProvider)
    ..invalidate(notificationsProvider);
  if (account != null) {
    ref.invalidate(accountNotificationsProvider(account));
  }
}
