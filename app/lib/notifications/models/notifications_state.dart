import 'package:craftsky_app/notifications/models/craftsky_notification.dart';

final class NotificationsState {
  const NotificationsState({required this.items, this.cursor});

  final List<CraftskyNotification> items;
  final String? cursor;

  bool get hasMore => cursor != null;

  NotificationsState copyWith({
    List<CraftskyNotification>? items,
    String? cursor,
  }) => NotificationsState(
    items: items ?? this.items,
    cursor: cursor,
  );
}
