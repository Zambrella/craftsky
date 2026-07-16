import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum NotificationNewCountTrigger {
  ready,
  resume,
  foregroundEvent,
  pageRefresh,
  markSeen,
  elapsedTimer,
  unrelatedRebuild,
}

abstract final class NotificationNewCountPolicy {
  static bool shouldRefresh(NotificationNewCountTrigger trigger) =>
      switch (trigger) {
        NotificationNewCountTrigger.ready ||
        NotificationNewCountTrigger.resume ||
        NotificationNewCountTrigger.foregroundEvent ||
        NotificationNewCountTrigger.pageRefresh ||
        NotificationNewCountTrigger.markSeen => true,
        NotificationNewCountTrigger.elapsedTimer ||
        NotificationNewCountTrigger.unrelatedRebuild => false,
      };
}

final notificationNewCountProvider =
    AsyncNotifierProvider<NotificationNewCount, int>(NotificationNewCount.new);

class NotificationNewCount extends AsyncNotifier<int> {
  @override
  Future<int> build() =>
      ref.watch(notificationNewnessRepositoryProvider).count();

  Future<void> refreshFor(NotificationNewCountTrigger trigger) async {
    if (!NotificationNewCountPolicy.shouldRefresh(trigger)) return;
    final next = await AsyncValue.guard(
      ref.read(notificationNewnessRepositoryProvider).count,
    );
    if (ref.mounted) state = next;
  }
}
