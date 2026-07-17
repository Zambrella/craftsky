import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';

abstract final class NotificationDestinationInference {
  static NotificationOpenOutcome forFacts(
    NotificationFactOutcome facts,
  ) => switch (facts) {
    ValidNotificationFacts(:final category) => NotificationOpenOutcome(
      destination: switch (category) {
        NotificationCategory.follow => ProfileDestination(facts.actorDid!),
        NotificationCategory.like ||
        NotificationCategory.repost => PostDestination(
          facts.rootUri!,
          focusUri: facts.subjectUri == facts.rootUri ? null : facts.subjectUri,
        ),
        NotificationCategory.mention ||
        NotificationCategory.quote => PostDestination(facts.sourceUri!),
        NotificationCategory.reply => PostDestination(
          facts.subjectUri!,
          focusUri: facts.sourceUri,
        ),
        NotificationCategory.everythingElse => const NotificationsDestination(),
        NotificationCategory.unknown => throw StateError(
          'Unknown categories cannot be valid notification facts',
        ),
      },
    ),
    UnknownNotificationFacts() => const NotificationOpenOutcome(
      destination: NotificationsDestination(),
    ),
    InvalidNotificationFacts() => const NotificationOpenOutcome(
      destination: NotificationsDestination(),
      feedback: NotificationOpenFeedback.unableToOpen,
    ),
  };
}
