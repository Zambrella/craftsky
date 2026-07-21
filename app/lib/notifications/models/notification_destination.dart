// These final value types contain only final fields; equality is safe without
// importing Flutter's `@immutable` annotation into this domain-only file.
// ignore_for_file: avoid_equals_and_hash_code_on_mutable_classes

import 'package:craftsky_app/shared/atproto/identifiers.dart';

enum NotificationOpenFeedback { unableToOpen }

sealed class NotificationDestination {
  const NotificationDestination();
}

final class NotificationsDestination extends NotificationDestination {
  const NotificationsDestination();

  @override
  bool operator ==(Object other) => other is NotificationsDestination;

  @override
  int get hashCode => 0;
}

final class InstagramMigrationDestination extends NotificationDestination {
  const InstagramMigrationDestination();

  @override
  bool operator ==(Object other) => other is InstagramMigrationDestination;

  @override
  int get hashCode => 1;
}

final class ProfileDestination extends NotificationDestination {
  const ProfileDestination(this.did);

  final Did did;

  @override
  bool operator ==(Object other) =>
      other is ProfileDestination && other.did == did;

  @override
  int get hashCode => Object.hash(ProfileDestination, did);
}

final class PostDestination extends NotificationDestination {
  const PostDestination(this.subjectUri, {this.focusUri});

  final AtUri subjectUri;
  final AtUri? focusUri;

  @override
  bool operator ==(Object other) =>
      other is PostDestination &&
      other.subjectUri == subjectUri &&
      other.focusUri == focusUri;

  @override
  int get hashCode => Object.hash(PostDestination, subjectUri, focusUri);
}

final class NotificationOpenOutcome {
  const NotificationOpenOutcome({required this.destination, this.feedback});

  final NotificationDestination destination;
  final NotificationOpenFeedback? feedback;
}
