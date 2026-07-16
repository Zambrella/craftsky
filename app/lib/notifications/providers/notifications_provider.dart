import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notifications_state.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notifications_provider.g.dart';

const notificationsPageLimit = 20;
int _nextRenderToken = 0;

@Riverpod(keepAlive: true)
class Notifications extends _$Notifications {
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
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;

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
