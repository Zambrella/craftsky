import 'package:craftsky_app/notifications/models/notification_open_event.dart';

abstract final class NotificationRoutingPolicy {
  static bool canResolve({
    required AccountSubscriptionId? storedBinding,
    required AccountSubscriptionId eventBinding,
  }) => storedBinding != null && storedBinding == eventBinding;
}
