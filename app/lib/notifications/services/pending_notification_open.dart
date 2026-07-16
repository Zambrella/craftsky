import 'package:craftsky_app/notifications/models/notification_open_event.dart';

enum NotificationOpenReadiness { transient, ready, requiresSignIn }

final class PendingNotificationOpen {
  NotificationOpenEvent? _pending;

  NotificationOpenEvent? receive(
    NotificationOpenEvent event, {
    required NotificationOpenReadiness readiness,
  }) {
    switch (readiness) {
      case NotificationOpenReadiness.ready:
        return event;
      case NotificationOpenReadiness.transient:
        _pending = event;
        return null;
      case NotificationOpenReadiness.requiresSignIn:
        _pending = null;
        return null;
    }
  }

  NotificationOpenEvent? updateReadiness(NotificationOpenReadiness readiness) {
    switch (readiness) {
      case NotificationOpenReadiness.transient:
        return null;
      case NotificationOpenReadiness.ready:
        final event = _pending;
        _pending = null;
        return event;
      case NotificationOpenReadiness.requiresSignIn:
        _pending = null;
        return null;
    }
  }
}
