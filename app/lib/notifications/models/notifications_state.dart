import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'notifications_state.mapper.dart';

@MappableClass(
  generateMethods: GenerateMethods.copy | GenerateMethods.equals,
)
final class NotificationsState with NotificationsStateMappable {
  const NotificationsState({
    required this.items,
    required this.renderToken,
    this.cursor,
  });

  final List<CraftskyNotification> items;
  final String? cursor;
  final int renderToken;

  bool get hasMore => cursor != null;
}
