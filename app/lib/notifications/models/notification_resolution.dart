import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

enum NotificationResolutionState { active, retracted, unknown }

sealed class NotificationResolutionTarget {
  const NotificationResolutionTarget();

  factory NotificationResolutionTarget.fromMap(Map<String, dynamic> map) {
    try {
      return switch (map['kind']) {
        'post' => NotificationPostTarget(
          AtUri.parse(map['uri'] as String),
        ),
        'actorProfile' => NotificationProfileTarget(
          Did.parse(map['did'] as String),
        ),
        'notifications' => const NotificationListTarget(),
        _ => const UnknownNotificationTarget(),
      };
    } on Object {
      return const UnknownNotificationTarget();
    }
  }
}

final class NotificationPostTarget extends NotificationResolutionTarget {
  const NotificationPostTarget(this.uri);
  final AtUri uri;
}

final class NotificationProfileTarget extends NotificationResolutionTarget {
  const NotificationProfileTarget(this.did);
  final Did did;
}

final class NotificationListTarget extends NotificationResolutionTarget {
  const NotificationListTarget();
}

final class UnknownNotificationTarget extends NotificationResolutionTarget {
  const UnknownNotificationTarget();
}

final class NotificationResolution {
  const NotificationResolution({
    required this.id,
    required this.category,
    required this.state,
    required this.target,
  });

  factory NotificationResolution.fromMap(Map<String, Object?> map) {
    final rawState = map['state'];
    final rawTarget = map['target'];
    return NotificationResolution(
      id: NotificationId.parse(map['id']! as String),
      category: NotificationCategory.fromWireValue(map['type']! as String),
      state: switch (rawState) {
        'active' => NotificationResolutionState.active,
        'retracted' => NotificationResolutionState.retracted,
        _ => NotificationResolutionState.unknown,
      },
      target: rawTarget is Map<String, dynamic>
          ? NotificationResolutionTarget.fromMap(rawTarget)
          : const UnknownNotificationTarget(),
    );
  }

  final NotificationId id;
  final NotificationCategory category;
  final NotificationResolutionState state;
  final NotificationResolutionTarget target;
}
