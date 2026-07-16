import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/services/notification_resolution_policy.dart';

sealed class NotificationEffect {
  const NotificationEffect();
}

final class NotificationBannerEffect extends NotificationEffect {
  const NotificationBannerEffect(this.event);
  final ForegroundNotificationEvent event;
}

final class NotificationNavigationEffect extends NotificationEffect {
  const NotificationNavigationEffect(this.outcome);
  final NotificationResolutionOutcome outcome;
}

final class NotificationUnavailableEffect extends NotificationEffect {
  const NotificationUnavailableEffect();
}
