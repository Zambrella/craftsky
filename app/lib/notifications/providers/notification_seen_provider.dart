import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_seen_provider.g.dart';

@Riverpod(keepAlive: true)
NotificationSeenCoordinator notificationSeen(Ref ref) =>
    NotificationSeenCoordinator(
      repository: ref.watch(notificationNewnessRepositoryProvider),
      refreshCount: () =>
          ref.read(notificationNewCountProvider.notifier).refresh(),
    );

final class NotificationSeenCoordinator {
  NotificationSeenCoordinator({
    required this._repository,
    required this._refreshCount,
  });

  final NotificationNewnessRepository _repository;
  final Future<void> Function() _refreshCount;
  final _consumedRenderTokens = <int>{};

  Future<void> afterSuccessfulRender(int token) async {
    if (!_consumedRenderTokens.add(token)) return;
    try {
      await _repository.markSeen();
      await _refreshCount();
    } on Object {
      _consumedRenderTokens.remove(token);
    }
  }
}
