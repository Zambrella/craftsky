import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
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
  includeCustomMappers: [
    DidMapper(),
    AtUriMapper(),
    ModerationCaseReferenceMapper(),
  ],
  generateMethods: _notificationDestinationMethods,
)
sealed class NotificationDestination with NotificationDestinationMappable {
  const new();
}

@MappableClass(generateMethods: _notificationDestinationMethods)
final class NotificationsDestination extends NotificationDestination
    with NotificationsDestinationMappable {
  const new();
}

@MappableClass(generateMethods: _notificationDestinationMethods)
final class InstagramMigrationDestination extends NotificationDestination
    with InstagramMigrationDestinationMappable {
  const new();
}

@MappableClass(generateMethods: _notificationDestinationMethods)
final class ProfileDestination extends NotificationDestination
    with ProfileDestinationMappable {
  const new(this.did);

  final Did did;
}

@MappableClass(generateMethods: _notificationDestinationMethods)
final class PostDestination extends NotificationDestination
    with PostDestinationMappable {
  const new(this.subjectUri, {this.focusUri});

  final AtUri subjectUri;
  final AtUri? focusUri;
}

@MappableClass(generateMethods: _notificationDestinationMethods)
final class ModerationHistoryDestination extends NotificationDestination
    with ModerationHistoryDestinationMappable {
  const new(this.caseReference);

  final ModerationCaseReference caseReference;
}

@MappableClass(generateMethods: _notificationDestinationMethods)
final class ChatInboxDestination extends NotificationDestination
    with ChatInboxDestinationMappable {
  const new();
}

@MappableClass(
  generateMethods: _notificationDestinationMethods,
  includeCustomMappers: [ChatConversationIdMapper()],
)
final class ChatConversationDestination extends NotificationDestination
    with ChatConversationDestinationMappable {
  const new(this.conversationId);
  final ChatConversationId conversationId;
  @override
  String toString() => 'ChatConversationDestination(<redacted>)';
}

class ChatConversationIdMapper extends SimpleMapper<ChatConversationId> {
  const new();
  @override
  ChatConversationId decode(Object value) =>
      ChatConversationId.parse(value as String);
  @override
  Object encode(ChatConversationId self) => self.value;
}

class ModerationCaseReferenceMapper
    extends SimpleMapper<ModerationCaseReference> {
  const new();

  @override
  ModerationCaseReference decode(Object value) =>
      ModerationCaseReference.parse(value as String);

  @override
  Object encode(ModerationCaseReference self) => self.value;
}

final class NotificationOpenOutcome {
  const new({required this.destination, this.feedback, this.isCurrent});

  final NotificationDestination destination;
  final NotificationOpenFeedback? feedback;

  /// Rechecked when queued private navigation is finally delivered.
  final bool Function()? isCurrent;
}
