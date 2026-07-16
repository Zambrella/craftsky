import 'package:flutter/foundation.dart';

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
