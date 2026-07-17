import 'package:craftsky_app/notifications/models/notification_open_event.dart';

final class ForegroundNotificationEvent {
  const ForegroundNotificationEvent({
    required this.title,
    required this.body,
    required this.openAttempt,
  });

  final String title;
  final String body;
  final NotificationOpenAttempt openAttempt;

  @override
  String toString() =>
      'ForegroundNotificationEvent(facts: ${openAttempt.facts.runtimeType}, '
      'source: ${openAttempt.source}, copy: <redacted>)';
}
