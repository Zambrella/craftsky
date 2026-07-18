import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';

enum NotificationOpenReadiness { transient, ready, requiresSignIn }

final class PendingNotificationOpenWork {
  const PendingNotificationOpenWork({
    required this.attempt,
    required this.resolution,
    this.sequence = 0,
    this.latestOnly = false,
  });

  final NotificationOpenAttempt attempt;
  final NotificationRecipientResolution resolution;
  final int sequence;
  final bool latestOnly;

  @override
  String toString() => 'PendingNotificationOpenWork(<redacted>)';
}

final class PendingNotificationOpen {
  PendingNotificationOpenWork? _pending;

  PendingNotificationOpenWork? receive(
    PendingNotificationOpenWork event, {
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

  PendingNotificationOpenWork? updateReadiness(
    NotificationOpenReadiness readiness,
  ) {
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
