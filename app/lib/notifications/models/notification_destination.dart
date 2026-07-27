import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'notification_destination.mapper.dart';

enum NotificationOpenFeedback { unableToOpen }

const int _notificationDestinationMethods =
    GenerateMethods.decode |
    GenerateMethods.encode |
    GenerateMethods.equals |
    GenerateMethods.copy;

@MappableClass(
  discriminatorKey: 'type',
  includeCustomMappers: [DidMapper(), AtUriMapper()],
  generateMethods: _notificationDestinationMethods,
)
sealed class NotificationDestination with NotificationDestinationMappable {
  const NotificationDestination();
}

@MappableClass(generateMethods: _notificationDestinationMethods)
final class NotificationsDestination extends NotificationDestination
    with NotificationsDestinationMappable {
  const NotificationsDestination();
}

@MappableClass(generateMethods: _notificationDestinationMethods)
final class InstagramMigrationDestination extends NotificationDestination
    with InstagramMigrationDestinationMappable {
  const InstagramMigrationDestination();
}

@MappableClass(generateMethods: _notificationDestinationMethods)
final class ProfileDestination extends NotificationDestination
    with ProfileDestinationMappable {
  const ProfileDestination(this.did);

  final Did did;
}

@MappableClass(generateMethods: _notificationDestinationMethods)
final class PostDestination extends NotificationDestination
    with PostDestinationMappable {
  const PostDestination(this.subjectUri, {this.focusUri});

  final AtUri subjectUri;
  final AtUri? focusUri;
}

final class NotificationOpenOutcome {
  const NotificationOpenOutcome({required this.destination, this.feedback});

  final NotificationDestination destination;
  final NotificationOpenFeedback? feedback;
}
