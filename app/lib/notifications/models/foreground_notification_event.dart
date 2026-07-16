import 'package:craftsky_app/notifications/models/notification_open_event.dart';

final class ForegroundNotificationEvent {
  const ForegroundNotificationEvent({
    required this.title,
    required this.body,
    required this.openEvent,
  });

  final String title;
  final String body;
  final NotificationOpenEvent openEvent;

  @override
  String toString() =>
      'ForegroundNotificationEvent(category: ${openEvent.category}, '
      'source: ${openEvent.source}, copy: <redacted>)';
}
