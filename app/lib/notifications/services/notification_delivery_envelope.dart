import 'dart:convert';

import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:uuid/uuid.dart';

/// Validated copy and routing facts from a standard notification message.
///
/// Navigation continues to be derived from [NotificationOpenAttempt]'s
/// validated routing facts and account binding.
final class NotificationDeliveryEnvelope {
  NotificationDeliveryEnvelope._({
    required this.notificationId,
    required this.title,
    required this.body,
    required this.openAttempt,
  });

  static const int maxDisplayBytes = 256;
  static const int maxRoutingValueBytes = 1024;
  static const _allowedKeys = <String>{
    'payloadVersion',
    'type',
    'accountSubscriptionId',
    'notificationId',
    'displayTitle',
    'displayBody',
    'actorDid',
    'subjectUri',
    'rootUri',
    'sourceUri',
  };

  final String notificationId;
  final String title;
  final String body;
  final NotificationOpenAttempt openAttempt;

  static NotificationDeliveryEnvelope? tryParse(
    Map<String, Object?> data, {
    NotificationOpenSource source = NotificationOpenSource.backgroundOpen,
  }) {
    final notificationId = data['notificationId'];
    final title = data['displayTitle'];
    final body = data['displayBody'];
    if (notificationId is! String ||
        !Uuid.isValidUUID(fromString: notificationId) ||
        title is! String ||
        !_isSafeDisplayText(title) ||
        body is! String ||
        !_isSafeDisplayText(body)) {
      return null;
    }

    final bounded = <String, String>{};
    for (final key in _allowedKeys) {
      final value = data[key];
      if (value == null) continue;
      if (value is! String ||
          utf8.encode(value).length > maxRoutingValueBytes) {
        return null;
      }
      bounded[key] = value;
    }
    bounded['notificationId'] = UuidValue.fromString(notificationId).uuid;

    final openAttempt = NotificationOpenAttempt.fromProviderData(
      bounded,
      source: source,
    );
    if (openAttempt.accountSubscriptionId == null ||
        openAttempt.facts is! ValidNotificationFacts) {
      return null;
    }

    return NotificationDeliveryEnvelope._(
      notificationId: bounded['notificationId']!,
      title: title,
      body: body,
      openAttempt: openAttempt,
    );
  }

  static bool _isSafeDisplayText(String value) {
    if (value.isEmpty || utf8.encode(value).length > maxDisplayBytes) {
      return false;
    }
    return value.runes.every(
      (codePoint) =>
          codePoint > 0x1f && !(codePoint >= 0x7f && codePoint <= 0x9f),
    );
  }

  @override
  String toString() =>
      'NotificationDeliveryEnvelope(copy: <redacted>, '
      'notificationId: <redacted>, openAttempt: $openAttempt)';
}
