import 'dart:collection';
import 'dart:convert';

import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:uuid/uuid.dart';

/// Validated app-owned presentation data for an Android unique-event message.
///
/// The notification ID is presentation/deduplication identity only. Navigation
/// continues to be derived from [NotificationOpenAttempt]'s validated routing
/// facts and account binding.
final class NotificationDeliveryEnvelope {
  NotificationDeliveryEnvelope._({
    required this.notificationId,
    required this.title,
    required this.body,
    required this.openAttempt,
    required Map<String, String> providerData,
  }) : providerData = UnmodifiableMapView(providerData);

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
  final Map<String, String> providerData;

  /// Android uses the complete canonical UUID as its native string tag.
  String get androidTag => notificationId;

  String get localOpenPayload => jsonEncode(providerData);

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
      providerData: bounded,
    );
  }

  static NotificationDeliveryEnvelope? tryParseLocalPayload(
    String? payload, {
    NotificationOpenSource source = NotificationOpenSource.backgroundOpen,
  }) {
    if (payload == null || payload.length > 8192) return null;
    try {
      final decoded = jsonDecode(payload);
      if (decoded is! Map<String, dynamic>) return null;
      return tryParse(decoded, source: source);
    } on FormatException {
      return null;
    }
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
