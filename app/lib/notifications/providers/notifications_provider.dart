import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notifications_state.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const notificationsPageLimit = 20;
int _nextRenderToken = 0;

final notificationsProvider =
    AsyncNotifierProvider<Notifications, NotificationsState>(Notifications.new);

class Notifications extends AsyncNotifier<NotificationsState> {
  @override
  Future<NotificationsState> build() async {
    final repo = ref.watch(notificationRepositoryProvider);
    final page = await repo.list(limit: notificationsPageLimit);
    return NotificationsState(
      items: _dedupe(page.items),
      cursor: page.cursor,
      renderToken: ++_nextRenderToken,
    );
  }

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || !current.hasMore || state.isLoading) return;

    state = const AsyncLoading<NotificationsState>();

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(notificationRepositoryProvider);
      final page = await repo.list(
        cursor: current.cursor,
        limit: notificationsPageLimit,
      );
      return NotificationsState(
        items: _appendDeduped(current.items, page.items),
        cursor: page.cursor,
        renderToken: current.renderToken,
      );
    });

    if (!ref.mounted) return;
    state = next;
  }
}

List<CraftskyNotification> _dedupe(List<CraftskyNotification> items) {
  final seen = <String>{};
  return [
    for (final item in items)
      if (seen.add(item.id)) item,
  ];
}

List<CraftskyNotification> _appendDeduped(
  List<CraftskyNotification> current,
  List<CraftskyNotification> next,
) {
  final seen = current.map((item) => item.id).toSet();
  return [
    ...current,
    for (final item in next)
      if (seen.add(item.id)) item,
  ];
}
