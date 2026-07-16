import 'package:craftsky_app/notifications/models/notification_resolution.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';

enum NotificationResolutionFailure { notFound, network, timeout, unavailable }

enum NotificationOpenFeedback { unableToOpen }

sealed class NotificationDestination {
  const NotificationDestination();

  const factory NotificationDestination.notifications() =
      NotificationsDestination;
  const factory NotificationDestination.post(AtUri uri) = PostDestination;
  const factory NotificationDestination.profile(Did did) = ProfileDestination;
}

@immutable
final class NotificationsDestination extends NotificationDestination {
  const NotificationsDestination();

  @override
  bool operator ==(Object other) => other is NotificationsDestination;

  @override
  int get hashCode => 0;
}

@immutable
final class PostDestination extends NotificationDestination {
  const PostDestination(this.uri);
  final AtUri uri;

  @override
  bool operator ==(Object other) =>
      other is PostDestination && other.uri == uri;

  @override
  int get hashCode => Object.hash(PostDestination, uri);
}

@immutable
final class ProfileDestination extends NotificationDestination {
  const ProfileDestination(this.did);
  final Did did;

  @override
  bool operator ==(Object other) =>
      other is ProfileDestination && other.did == did;

  @override
  int get hashCode => Object.hash(ProfileDestination, did);
}

final class NotificationResolutionOutcome {
  const NotificationResolutionOutcome({
    required this.destination,
    this.feedback,
  });

  final NotificationDestination destination;
  final NotificationOpenFeedback? feedback;

  bool get shouldRetry => false;
}

abstract final class NotificationResolutionPolicy {
  static NotificationResolutionOutcome forResolution(
    NotificationResolution resolution,
  ) {
    if (resolution.state != NotificationResolutionState.active) {
      return _notifications;
    }
    final destination = switch (resolution.target) {
      NotificationPostTarget(:final uri) => NotificationDestination.post(uri),
      NotificationProfileTarget(:final did) => NotificationDestination.profile(
        did,
      ),
      NotificationListTarget() || UnknownNotificationTarget() =>
        const NotificationDestination.notifications(),
    };
    return NotificationResolutionOutcome(destination: destination);
  }

  static NotificationResolutionOutcome forFailure(
    NotificationResolutionFailure failure,
  ) => switch (failure) {
    NotificationResolutionFailure.network ||
    NotificationResolutionFailure.timeout =>
      const NotificationResolutionOutcome(
        destination: NotificationDestination.notifications(),
        feedback: NotificationOpenFeedback.unableToOpen,
      ),
    NotificationResolutionFailure.notFound ||
    NotificationResolutionFailure.unavailable => _notifications,
  };

  static NotificationResolutionOutcome forException(Object error) => forFailure(
    switch (error) {
      ApiBadRequest(:final details) when details.statusCode == 404 =>
        NotificationResolutionFailure.notFound,
      ApiNetworkError() ||
      ApiServerError() => NotificationResolutionFailure.network,
      _ => NotificationResolutionFailure.unavailable,
    },
  );

  static const _notifications = NotificationResolutionOutcome(
    destination: NotificationDestination.notifications(),
  );
}
