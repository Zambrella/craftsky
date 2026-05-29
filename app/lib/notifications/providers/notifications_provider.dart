import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notifications_state.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const notificationsPageLimit = 20;

final notificationsProvider =
    AsyncNotifierProvider<Notifications, NotificationsState>(Notifications.new);

class Notifications extends AsyncNotifier<NotificationsState> {
  @override
  Future<NotificationsState> build() async {
    final repo = ref.watch(notificationRepositoryProvider);
    final page = await repo.list(limit: notificationsPageLimit);
    return NotificationsState(items: _dedupe(page.items), cursor: page.cursor);
  }

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || !current.hasMore || state.isLoading) return;

    // ignore: invalid_use_of_internal_member
    state = const AsyncLoading<NotificationsState>().copyWithPrevious(state);

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(notificationRepositoryProvider);
      final page = await repo.list(
        cursor: current.cursor,
        limit: notificationsPageLimit,
      );
      return NotificationsState(
        items: _appendDeduped(current.items, page.items),
        cursor: page.cursor,
      );
    });

    if (!ref.mounted) return;
    // ignore: invalid_use_of_internal_member
    state = next.copyWithPrevious(state);
  }
}

List<CraftskyNotification> _dedupe(List<CraftskyNotification> items) {
  final seen = <String>{};
  return [
    for (final item in items)
      if (seen.add(item.uri.toString())) item,
  ];
}

List<CraftskyNotification> _appendDeduped(
  List<CraftskyNotification> current,
  List<CraftskyNotification> next,
) {
  final seen = current.map((item) => item.uri.toString()).toSet();
  return [
    ...current,
    for (final item in next)
      if (seen.add(item.uri.toString())) item,
  ];
}
