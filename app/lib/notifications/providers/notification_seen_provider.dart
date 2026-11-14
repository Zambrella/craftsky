import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/shared/observability/consumed_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_seen_provider.g.dart';

@Riverpod(keepAlive: true)
Future<NotificationSeenCoordinator> accountNotificationSeen(
  Ref ref,
  AccountKey account,
) async {
  final reporter = ref.watch(errorReporterProvider);
  // The retained coordinator owns this disposable account repository by watch.
  // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
  final repository = await ref.watch(
    accountNotificationNewnessRepositoryProvider(account).future,
  );
  return NotificationSeenCoordinator(
    repository: repository,
    reporter: reporter,
    accountDid: () => account.did.value,
    refreshCount: () async {
      if (!ref.mounted) return;
      await ref
          .read(accountNotificationNewCountProvider(account).notifier)
          .refresh();
    },
  );
}

@Riverpod(keepAlive: true)
NotificationSeenCoordinator notificationSeen(Ref ref) =>
    NotificationSeenCoordinator(
      repository: ref.watch(notificationNewnessRepositoryProvider),
      reporter: ref.watch(errorReporterProvider),
      accountDid: () => ref.mounted && ref.exists(sessionRegistryProvider)
          ? ref
                .read(sessionRegistryProvider)
                .value
                ?.activeLease
                ?.session
                .account
                .did
                .value
          : null,
      refreshCount: () async {
        if (!ref.mounted) return;
        await ref.read(notificationNewCountProvider.notifier).refresh();
      },
    );

final class NotificationSeenCoordinator {
  new({
    required this._repository,
    required this._refreshCount,
    this._reporter = const NoopErrorReporter(),
    this._accountDid,
  });

  final NotificationNewnessRepository _repository;
  final Future<void> Function() _refreshCount;
  final ErrorReporter _reporter;
  final String? Function()? _accountDid;
  final _consumedRenderTokens = <int>{};

  Future<void> afterSuccessfulRender(int token) async {
    if (!_consumedRenderTokens.add(token)) return;
    final accountDid = _accountDid?.call();
    var markingSeen = true;
    try {
      await _repository.markSeen();
      markingSeen = false;
      await _refreshCount();
    } on Object catch (error, stack) {
      _consumedRenderTokens.remove(token);
      // Count-refresh failures publish AsyncError through their own provider.
      if (markingSeen) {
        reportConsumedFailure(
          error,
          stack,
          reporter: _reporter,
          feature: 'Notifications',
          operation: 'notifications.mark_seen',
          accountDid: accountDid,
        );
      }
    }
  }
}
