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
  final Map<String, AccountDeletionLeaseFence> _pendingFences = {};

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
      _pendingFences[intent.jobId] = fence;
      final launched = await ref.read(authUrlLauncherProvider)(intent.authUrl);
      if (!launched) {
        _pendingFences.remove(intent.jobId);
        try {
          await client.cancelIntent(jobId: intent.jobId);
        } on Object {
          // Best effort: the short-lived unaccepted intent also expires.
        }
        throw const AccountDeletionFlowException('browserLaunchFailed');
      }
      jobId = intent.jobId;
    });
    return state.hasError ? null : jobId;
  }

  bool canComplete(String jobId) {
    final fence = _pendingFences[jobId];
    final sessions = ref.read(sessionRegistryProvider).value;
    return fence != null && sessions != null && fence.isCurrent(sessions);
  }

  String? requiredHandle(String jobId) => _pendingFences[jobId]?.requiredHandle;

  Future<void> cancelPendingIntent(String jobId) async {
    final fence = _pendingFences.remove(jobId);
    if (fence == null) return;
    try {
      final dio = await ref.read(accountDioProvider(fence.account).future);
      await AccountDeletionApiClient(dio).cancelIntent(jobId: jobId);
    } on Object {
      // Best effort. The server expires and replaces abandoned intents.
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
      final fence = _pendingFences[jobId];
      final sessions = await ref.read(sessionRegistryProvider.future);
      if (fence == null || !fence.isCurrent(sessions)) {
        throw const AccountDeletionFlowException('staleAccountLease');
      }
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
      _pendingFences.remove(jobId);
      accepted = true;
    });
    return accepted;
  }
}
