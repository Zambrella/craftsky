import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_id.dart';

enum NotificationOpenSource { foregroundBanner, backgroundOpen, initialOpen }

final class NotificationOpenEvent {
  const NotificationOpenEvent({
    required this.notificationId,
    required this.category,
    required this.accountSubscriptionId,
    required this.source,
  });

  static final _typePattern = RegExp(r'^[A-Za-z][A-Za-z0-9]{0,63}$');

  static NotificationOpenEvent? tryParseProviderData(
    Map<String, Object?> data, {
    NotificationOpenSource source = NotificationOpenSource.backgroundOpen,
  }) {
    final notificationId = data['notificationId'];
    final type = data['type'];
    final accountSubscriptionId = data['accountSubscriptionId'];
    if (notificationId is! String ||
        type is! String ||
        accountSubscriptionId is! String ||
        !_typePattern.hasMatch(type)) {
      return null;
    }

    try {
      return NotificationOpenEvent(
        notificationId: NotificationId.parse(notificationId),
        category: NotificationCategory.fromWireValue(type),
        accountSubscriptionId: AccountSubscriptionId.parse(
          accountSubscriptionId,
        ),
        source: source,
      );
    } on FormatException {
      return null;
    }
  }

  final NotificationId notificationId;
  final NotificationCategory category;
  final AccountSubscriptionId accountSubscriptionId;
  final NotificationOpenSource source;

  @override
  String toString() =>
      'NotificationOpenEvent(category: $category, source: $source, '
      'notificationId: <redacted>, accountSubscriptionId: <redacted>)';
}
