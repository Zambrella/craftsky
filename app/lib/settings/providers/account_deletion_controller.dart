import 'dart:async';

import 'package:craftsky_app/auth/data/account_deletion_repository.dart';
import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/services/account_deletion_acceptance_coordinator.dart';
import 'package:craftsky_app/settings/services/account_product_data_cleaner.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'account_deletion_controller.g.dart';

final class AccountDeletionFlowException implements Exception {
  const AccountDeletionFlowException(this.reason);

  final String reason;

  @override
  String toString() => 'AccountDeletionFlowException($reason)';
}

@Riverpod(keepAlive: true)
class AccountDeletionController extends _$AccountDeletionController {
  @override
  FutureOr<void> build() => null;

  Future<String?> startReauthentication() async {
    String? jobId;
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final sessions = await ref.read(sessionRegistryProvider.future);
      final fence = AccountDeletionLeaseFence.capture(sessions);
      final dio = await ref.read(accountDioProvider(fence.account).future);
      final client = AccountDeletionApiClient(dio);
      final intent = await client.createIntent();
      try {
        await ref
            .read(sessionRegistryProvider.notifier)
            .stageAccountDeletion(
              fence.pending(
                jobId: intent.jobId,
                expiresAt: intent.expiresAt,
              ),
            );
      } on Object {
        try {
          await client.cancelIntent(jobId: intent.jobId);
        } on Object {
          // Best effort: the short-lived unaccepted intent also expires.
        }
        rethrow;
      }
      await ref.read(accountStateInvalidatorProvider)();
      final launched = await ref.read(authUrlLauncherProvider)(intent.authUrl);
      if (!launched) {
        try {
          await client.cancelIntent(jobId: intent.jobId);
        } on Object {
          // Best effort: the short-lived unaccepted intent also expires.
        }
        await ref
            .read(sessionRegistryProvider.notifier)
            .clearAccountDeletion(intent.jobId);
        throw const AccountDeletionFlowException('browserLaunchFailed');
      }
      jobId = intent.jobId;
    });
    return state.hasError ? null : jobId;
  }

  bool canComplete(String jobId) {
    final sessions = ref.read(sessionRegistryProvider).value;
    final pending = sessions?.pendingAccountDeletion;
    return pending != null &&
        pending.jobId == jobId &&
        pending.isCurrent(sessions?.activeLease);
  }

  String? requiredHandle(String jobId) {
    final pending = ref
        .read(sessionRegistryProvider)
        .value
        ?.pendingAccountDeletion;
    return pending?.jobId == jobId ? pending?.requiredHandle : null;
  }

  Future<void> cancelPendingIntent(String jobId) async {
    final pending = ref
        .read(sessionRegistryProvider)
        .value
        ?.pendingAccountDeletion;
    if (pending == null || pending.jobId != jobId) return;
    try {
      final dio = await ref.read(
        accountDioProvider(pending.lease.session.account).future,
      );
      await AccountDeletionApiClient(dio).cancelIntent(jobId: jobId);
    } on Object {
      // Best effort. The server expires and replaces abandoned intents.
    } finally {
      await ref
          .read(sessionRegistryProvider.notifier)
          .clearAccountDeletion(jobId);
    }
  }

  Future<bool> confirm({
    required String jobId,
    required String reauthProof,
    required String confirmationHandle,
  }) async {
    var accepted = false;
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final sessions = await ref.read(sessionRegistryProvider.future);
      final pending = sessions.pendingAccountDeletion;
      if (pending == null ||
          pending.jobId != jobId ||
          !pending.isCurrent(sessions.activeLease)) {
        throw const AccountDeletionFlowException('staleAccountLease');
      }
      final fence = AccountDeletionLeaseFence.fromPending(pending);
      final dio = await ref.read(accountDioProvider(fence.account).future);
      await AccountDeletionApiClient(dio).accept(
        jobId: jobId,
        reauthProof: reauthProof,
        confirmationHandle: confirmationHandle,
      );
      final coordinator = AccountDeletionAcceptanceCoordinator(
        readRegistry: () => ref.read(sessionRegistryProvider.future),
        invalidateActiveState: ref.read(accountStateInvalidatorProvider),
        cleanProductData: ref.read(accountProductDataCleanerProvider),
        removeOrdinarySession: ref
            .read(sessionRegistryProvider.notifier)
            .removeConfirmed,
        routeAfterActiveRemoval: ({required hasFallback}) async {
          if (!ref.mounted) return;
          if (hasFallback) {
            await ref.read(accountHomeResetProvider)();
          } else {
            ref.read(goRouterProvider).go('/sign-in');
          }
        },
      );
      await coordinator.reconcile(fence: fence);
      accepted = true;
    });
    return accepted;
  }
}
