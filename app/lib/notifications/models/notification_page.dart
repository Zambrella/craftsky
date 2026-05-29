import 'package:craftsky_app/notifications/models/craftsky_notification.dart';

final class NotificationPage {
  const NotificationPage({required this.items, this.cursor});

  final List<CraftskyNotification> items;
  final String? cursor;

  static NotificationPage fromMap(Map<String, dynamic> map) => NotificationPage(
    items: [
      for (final item in map['items'] as List<dynamic>)
        CraftskyNotification.fromMap(item as Map<String, dynamic>),
    ],
    cursor: map['cursor'] as String?,
  );
}
