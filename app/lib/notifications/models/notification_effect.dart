import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';

sealed class NotificationEffect {
  const NotificationEffect();
}

final class NotificationBannerEffect extends NotificationEffect {
  const NotificationBannerEffect(
    this.event, {
    required this.resolution,
    this.recipient,
  });
  final ForegroundNotificationEvent event;
  final NotificationRecipientResolution resolution;
  final NotificationRecipientIdentity? recipient;

  @override
  String toString() => 'NotificationBannerEffect(<redacted>)';
}

final class NotificationRecipientIdentity {
  const NotificationRecipientIdentity({
    required this.lease,
    required this.handle,
    this.avatarUrl,
  });

  final AccountSessionLease lease;
  final String handle;
  final String? avatarUrl;

  @override
  String toString() => 'NotificationRecipientIdentity(<redacted>)';
}

final class NotificationNavigationEffect extends NotificationEffect {
  const NotificationNavigationEffect(this.outcome);
  final NotificationOpenOutcome outcome;
}

final class NotificationUnavailableEffect extends NotificationEffect {
  const NotificationUnavailableEffect();
}

final class NotificationRemovedAccountEffect extends NotificationEffect {
  const NotificationRemovedAccountEffect();
}
