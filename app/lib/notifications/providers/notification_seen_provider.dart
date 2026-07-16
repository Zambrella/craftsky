import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/services/notification_seen_policy.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final notificationSeenProvider = Provider<NotificationSeenCoordinator>(
  (ref) => NotificationSeenCoordinator(
    repository: ref.watch(notificationNewnessRepositoryProvider),
    refreshCount: () => ref
        .read(notificationNewCountProvider.notifier)
        .refreshFor(NotificationNewCountTrigger.markSeen),
  ),
);

final class NotificationSeenCoordinator {
  NotificationSeenCoordinator({
    required this._repository,
    required this._refreshCount,
  });

  final NotificationNewnessRepository _repository;
  final Future<void> Function() _refreshCount;
  final NotificationSeenGate _gate = NotificationSeenGate();

  Future<void> afterSuccessfulRender(int token) async {
    if (!_gate.consume(token: token, rendered: true)) return;
    try {
      await _repository.markSeen();
      await _refreshCount();
    } on Object {
      _gate.release(token);
    }
  }
}
