import 'package:craftsky_app/notifications/models/craftsky_notification.dart';

final class NotificationPage {
  const new({required this.items, this.cursor});

  factory fromMap(Map<String, dynamic> map) => NotificationPage(
    items: [
      for (final item in map['items'] as List<dynamic>)
        CraftskyNotification.fromMap(item as Map<String, dynamic>),
    ],
    cursor: map['cursor'] as String?,
  );

  final List<CraftskyNotification> items;
  final String? cursor;
}
