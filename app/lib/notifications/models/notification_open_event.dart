import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:flutter/foundation.dart';

enum NotificationOpenSource { foregroundBanner, backgroundOpen, initialOpen }

@immutable
final class NotificationId {
  factory NotificationId.parse(String value) {
    if (!_uuidPattern.hasMatch(value)) {
      throw const FormatException('Invalid notification ID');
    }
    return NotificationId._(value);
  }
  const NotificationId._(this.wireValue);

  static final _uuidPattern = RegExp(
    r'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$',
    caseSensitive: false,
  );

  /// Wire access is reserved for the notification API adapter.
  final String wireValue;

  @override
  bool operator ==(Object other) =>
      other is NotificationId && other.wireValue == wireValue;

  @override
  int get hashCode => wireValue.hashCode;

  @override
  String toString() => '<redacted-notification-id>';
}

@immutable
final class AccountSubscriptionId {
  factory AccountSubscriptionId.parse(String value) {
    if (!_identifierPattern.hasMatch(value)) {
      throw const FormatException('Invalid account subscription ID');
    }
    return AccountSubscriptionId._(value);
  }
  const AccountSubscriptionId._(this.wireValue);

  static final _identifierPattern = RegExp(r'^[A-Za-z0-9_-]{1,128}$');

  /// Wire access is reserved for secure routing storage and comparison.
  final String wireValue;

  @override
  bool operator ==(Object other) =>
      other is AccountSubscriptionId && other.wireValue == wireValue;

  @override
  int get hashCode => wireValue.hashCode;

  @override
  String toString() => '<redacted-account-subscription-id>';
}

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
