import 'dart:async';

import 'package:craftsky_app/auth/data/account_deletion_repository.dart';
import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/account_deletion_repository_provider.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/deletion_status_registry_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/services/account_deletion_acceptance_coordinator.dart';
import 'package:craftsky_app/settings/services/account_product_data_cleaner.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
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
      final stored = sessions.sessions[fence.account.did]!;
      final dio = await ref.read(accountDioProvider(fence.account).future);
      final intent = await AccountDeletionApiClient(dio).createIntent();
      final entry = DeletionStatusEntry.pending(
        jobId: intent.jobId,
        did: stored.did.value,
        handle: stored.handle.value,
        statusToken: intent.statusToken,
        displayName: stored.cachedDisplayName,
        avatarUrl: stored.cachedAvatarUrl,
      );

      // This durable status-only binding must exist before the external OAuth
      // app can suspend or terminate CraftSky.
      await ref.read(deletionStatusRegistryProvider.notifier).upsert(entry);
      _pendingFences[intent.jobId] = fence;
      final launched = await ref.read(authUrlLauncherProvider)(intent.authUrl);
      if (!launched) {
        _pendingFences.remove(intent.jobId);
        try {
          await AccountDeletionApiClient(dio).cancelIntent(
            jobId: intent.jobId,
            statusToken: intent.statusToken,
          );
        } on Object {
          // Best effort: the short-lived unaccepted intent also expires.
        }
        await ref
            .read(deletionStatusRegistryProvider.notifier)
            .remove(intent.jobId);
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

  Future<void> cancelPendingIntent(String jobId) async {
    final registry = await ref.read(deletionStatusRegistryProvider.future);
    final entry = registry[jobId];
    if (entry == null || entry.status != AccountDeletionStatus.intent) return;
    final fence = _pendingFences.remove(jobId);
    if (fence != null) {
      try {
        final dio = await ref.read(accountDioProvider(fence.account).future);
        await AccountDeletionApiClient(dio).cancelIntent(
          jobId: jobId,
          statusToken: entry.statusToken,
        );
      } on Object {
        // Best effort. The server expires and replaces abandoned intents.
      }
    }
    await ref.read(deletionStatusRegistryProvider.notifier).remove(jobId);
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
      final statuses = await ref.read(deletionStatusRegistryProvider.future);
      final entry = statuses[jobId];
      final sessions = await ref.read(sessionRegistryProvider.future);
      if (fence == null || entry == null || !fence.isCurrent(sessions)) {
        throw const AccountDeletionFlowException('staleAccountLease');
      }
      final deviceId = await ref.read(deviceIdProvider.future);
      final repository = await ref.read(
        accountDeletionRepositoryProvider(
          AccountDeletionRepositoryKey(
            account: fence.account,
            status: DeletionStatusClientKey(
              jobId: jobId,
              statusToken: entry.statusToken,
              deviceId: deviceId,
            ),
          ),
        ).future,
      );
      final snapshot = await repository.acceptOrResolve(
        jobId: jobId,
        statusToken: entry.statusToken,
        reauthProof: reauthProof,
        confirmationHandle: confirmationHandle,
      );
      final acceptedEntry = entry.withStatus(
        status: snapshot.status,
        phase: snapshot.phase,
        canRetry: snapshot.canRetry,
        needsReauthentication: snapshot.needsReauthentication,
      );
      final coordinator = AccountDeletionAcceptanceCoordinator(
        readRegistry: () => ref.read(sessionRegistryProvider.future),
        persistStatus: ref.read(deletionStatusRegistryProvider.notifier).upsert,
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
            ref.read(goRouterProvider).go('/account-deletion/$jobId');
          }
        },
      );
      await coordinator.reconcile(fence: fence, entry: acceptedEntry);
      _pendingFences.remove(jobId);
      accepted = true;
    });
    return accepted;
  }

  Future<AccountDeletionStatusSnapshot?> refresh(String jobId) async {
    final registry = await ref.read(deletionStatusRegistryProvider.future);
    final entry = registry[jobId];
    if (entry == null) return null;
    final deviceId = await ref.read(deviceIdProvider.future);
    final client = ref.read(
      deletionStatusApiClientProvider(
        DeletionStatusClientKey(
          jobId: jobId,
          statusToken: entry.statusToken,
          deviceId: deviceId,
        ),
      ),
    );
    final snapshot = await client.getStatus(jobId);
    if (snapshot.status == AccountDeletionStatus.deleted) {
      await ref.read(deletionStatusRegistryProvider.notifier).remove(jobId);
    } else {
      await ref
          .read(deletionStatusRegistryProvider.notifier)
          .upsert(
            entry.withStatus(
              status: snapshot.status,
              phase: snapshot.phase,
              canRetry: snapshot.canRetry,
              needsReauthentication: snapshot.needsReauthentication,
            ),
          );
    }
    return snapshot;
  }

  Future<void> retry(String jobId) async {
    final registry = await ref.read(deletionStatusRegistryProvider.future);
    final entry = registry[jobId];
    if (entry == null || !entry.canRetry) return;
    final deviceId = await ref.read(deviceIdProvider.future);
    final client = ref.read(
      deletionStatusApiClientProvider(
        DeletionStatusClientKey(
          jobId: jobId,
          statusToken: entry.statusToken,
          deviceId: deviceId,
        ),
      ),
    );
    final snapshot = await client.retry(jobId);
    await ref
        .read(deletionStatusRegistryProvider.notifier)
        .upsert(
          entry.withStatus(
            status: snapshot.status,
            phase: snapshot.phase,
            canRetry: snapshot.canRetry,
            needsReauthentication: snapshot.needsReauthentication,
          ),
        );
  }

  Future<void> startReplacementReauthentication(String jobId) async {
    final registry = await ref.read(deletionStatusRegistryProvider.future);
    final entry = registry[jobId];
    if (entry == null || !entry.needsReauthentication) return;
    final deviceId = await ref.read(deviceIdProvider.future);
    final client = ref.read(
      deletionStatusApiClientProvider(
        DeletionStatusClientKey(
          jobId: jobId,
          statusToken: entry.statusToken,
          deviceId: deviceId,
        ),
      ),
    );
    final reauthentication = await client.startReauthentication(jobId);
    final launched = await ref.read(authUrlLauncherProvider)(
      reauthentication.authUrl,
    );
    if (!launched) {
      throw const AccountDeletionFlowException('browserLaunchFailed');
    }
  }
}
