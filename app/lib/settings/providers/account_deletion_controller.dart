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
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'account_deletion_controller.g.dart';

final class AccountDeletionFlowException implements Exception {
  const new(this.reason);

  final String reason;

  @override
  String toString() => 'AccountDeletionFlowException($reason)';
}

final _log = Logger('AccountDeletionController');

@Riverpod(keepAlive: true)
class AccountDeletionController extends _$AccountDeletionController {
  @override
  FutureOr<void> build() => null;

  Future<String?> startReauthentication() async {
    String? jobId;
    String? accountDid;
    String? workflowRef;
    var stage = 'session';
    state = const AsyncLoading();
    final result = await AsyncValue.guard<void>(() async {
      final sessions = await ref.read(sessionRegistryProvider.future);
      final fence = AccountDeletionLeaseFence.capture(sessions);
      accountDid = fence.account.did.value;
      stage = 'client';
      final target = accountDioProvider(fence.account);
      final subscription = ref.listen(target, (_, _) {});
      try {
        final dio = await ref.read(target.future);
        final client = AccountDeletionApiClient(dio);
        stage = 'create_intent';
        final intent = await client.createIntent();
        workflowRef = intent.jobId;
        stage = 'stage_intent';
        try {
          await ref
              .read(sessionRegistryProvider.notifier)
              .stageAccountDeletion(
                fence.pending(
                  jobId: intent.jobId,
                  confirmationDid: intent.confirmationDid,
                  expiresAt: intent.expiresAt,
                ),
              );
        } on Object {
          try {
            await client.cancelIntent(jobId: intent.jobId);
          } on Object catch (error, stack) {
            _log.warning(
              DiagnosticMessage(
                'Account deletion intent rollback failed',
                context: _failureContext(
                  error,
                  'rollback_intent',
                  accountDid,
                  intent.jobId,
                ),
              ),
              error,
              stack,
            );
            // Best effort: the short-lived unaccepted intent also expires.
          }
          rethrow;
        }
        stage = 'invalidate_state';
        await ref.read(accountStateInvalidatorProvider)();
        stage = 'launch_browser';
        final launched = await ref.read(authUrlLauncherProvider)(
          intent.authUrl,
        );
        if (!launched) {
          try {
            await client.cancelIntent(jobId: intent.jobId);
          } on Object catch (error, stack) {
            _log.warning(
              DiagnosticMessage(
                'Account deletion intent rollback failed',
                context: _failureContext(
                  error,
                  'rollback_intent',
                  accountDid,
                  intent.jobId,
                ),
              ),
              error,
              stack,
            );
            // Best effort: the short-lived unaccepted intent also expires.
          }
          await ref
              .read(sessionRegistryProvider.notifier)
              .clearAccountDeletion(intent.jobId);
          throw const AccountDeletionFlowException('browserLaunchFailed');
        }
        jobId = intent.jobId;
      } finally {
        subscription.close();
      }
    });
    if (result.hasError) {
      final context = _failureContext(
        result.error!,
        stage,
        accountDid,
        workflowRef,
      );
      _log.warning(
        DiagnosticMessage(
          'Account deletion reauthentication failed',
          context: context,
        ),
        result.error,
        result.stackTrace,
      );
      // Riverpod remains the sole issue owner; enrich its synchronous report.
      withFailureDiagnosticContext(
        result.error!,
        context,
        () => state = result,
      );
    } else {
      state = result;
    }
    return result.hasError ? null : jobId;
  }

  bool canComplete(String jobId) {
    final sessions = ref.read(sessionRegistryProvider).value;
    final pending = sessions?.pendingAccountDeletion;
    return pending != null &&
        pending.jobId == jobId &&
        pending.isCurrent(sessions?.activeLease);
  }

  String? confirmationDid(String jobId) {
    final pending = ref
        .read(sessionRegistryProvider)
        .value
        ?.pendingAccountDeletion;
    return pending?.jobId == jobId ? pending?.confirmationDid : null;
  }

  Future<void> cancelPendingIntent(String jobId) async {
    final pending = ref
        .read(sessionRegistryProvider)
        .value
        ?.pendingAccountDeletion;
    if (pending == null || pending.jobId != jobId) return;
    final target = accountDioProvider(pending.lease.session.account);
    final subscription = ref.listen(target, (_, _) {});
    try {
      final dio = await ref.read(target.future);
      await AccountDeletionApiClient(dio).cancelIntent(jobId: jobId);
    } on Object {
      // Best effort. The server expires and replaces abandoned intents.
    } finally {
      subscription.close();
      await ref
          .read(sessionRegistryProvider.notifier)
          .clearAccountDeletion(jobId);
    }
  }

  Future<bool> confirm({
    required String jobId,
    required String reauthProof,
    required String confirmationDid,
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
      if (confirmationDid != pending.confirmationDid) {
        throw const AccountDeletionFlowException('confirmationDidMismatch');
      }
      final fence = AccountDeletionLeaseFence.fromPending(pending);
      final target = accountDioProvider(fence.account);
      final subscription = ref.listen(target, (_, _) {});
      try {
        final dio = await ref.read(target.future);
        await AccountDeletionApiClient(dio).accept(
          jobId: jobId,
          reauthProof: reauthProof,
          confirmationDid: pending.confirmationDid,
        );
      } finally {
        subscription.close();
      }
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
            // The app's router config retains the destination router.
            // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
            ref.read(goRouterProvider).go(const WelcomeRoute().location);
          }
        },
      );
      await coordinator.reconcile(fence: fence);
      accepted = true;
    });
    return accepted;
  }
}

ReportContext _failureContext(
  Object error,
  String stage,
  String? accountDid,
  String? jobId,
) {
  final mapped = AppErrorMapper.map(
    error,
    fallbackKind: AppErrorKind.backgroundLoadFailed,
    source: 'provider',
    fallbackClassification: 'account.deletion',
  );
  return ReportContext(
    feature: 'AccountDeletionController',
    operation: 'account.delete',
    classification: mapped.sentryClassification,
    severity: mapped.metadata.severity.name,
    safeDiagnostics: {...mapped.safeDiagnostics, 'failureStage': stage},
    workflow: accountDid == null
        ? null
        : PrivateOperationalFailureContext(
            accountDid: accountDid,
            workflowRef: jobId,
          ),
  );
}
