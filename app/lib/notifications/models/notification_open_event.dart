import 'package:at_primitives/at_uri.dart' as at_uri;
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

enum NotificationOpenSource { foregroundBanner, backgroundOpen, initialOpen }

enum NotificationFactFailureClass {
  unsupportedPayloadVersion,
  malformedType,
  missingOrMalformedRequiredFacts,
}

sealed class NotificationFactOutcome {
  const NotificationFactOutcome();
}

final class ValidNotificationFacts extends NotificationFactOutcome {
  const ValidNotificationFacts._({
    required this.category,
    this.actorDid,
    this.subjectUri,
    this.rootUri,
    this.sourceUri,
  });

  final NotificationCategory category;
  final Did? actorDid;
  final AtUri? subjectUri;
  final AtUri? rootUri;
  final AtUri? sourceUri;

  @override
  String toString() => 'ValidNotificationFacts(category: $category)';
}

final class UnknownNotificationFacts extends NotificationFactOutcome {
  const UnknownNotificationFacts();

  @override
  String toString() => 'UnknownNotificationFacts()';
}

final class InvalidNotificationFacts extends NotificationFactOutcome {
  const InvalidNotificationFacts(this.failureClass);

  final NotificationFactFailureClass failureClass;

  @override
  String toString() => 'InvalidNotificationFacts(failureClass: $failureClass)';
}

final class NotificationOpenAttempt {
  const NotificationOpenAttempt({
    required this.accountSubscriptionId,
    required this.facts,
    required this.source,
  });

  factory NotificationOpenAttempt.fromProviderData(
    Map<String, Object?> data, {
    NotificationOpenSource source = NotificationOpenSource.backgroundOpen,
  }) {
    final accountSubscriptionId = _parseAccountSubscriptionId(
      data['accountSubscriptionId'],
    );
    final payloadVersion = data['payloadVersion'];
    final type = data['type'];

    final NotificationFactOutcome facts;
    if (payloadVersion != '1') {
      facts = const InvalidNotificationFacts(
        NotificationFactFailureClass.unsupportedPayloadVersion,
      );
    } else if (type is! String || !_typePattern.hasMatch(type)) {
      facts = const InvalidNotificationFacts(
        NotificationFactFailureClass.malformedType,
      );
    } else {
      final category = NotificationCategory.fromWireValue(type);
      facts = switch (category) {
        NotificationCategory.follow => switch (_parseDid(data['actorDid'])) {
          final actorDid? => ValidNotificationFacts._(
            category: category,
            actorDid: actorDid,
          ),
          null => const InvalidNotificationFacts(
            NotificationFactFailureClass.missingOrMalformedRequiredFacts,
          ),
        },
        NotificationCategory.like || NotificationCategory.repost => switch ((
          _parsePostUri(data['subjectUri']),
          _parsePostUri(data['rootUri']),
        )) {
          (final subjectUri?, final rootUri?) => ValidNotificationFacts._(
            category: category,
            subjectUri: subjectUri,
            rootUri: rootUri,
          ),
          _ => const InvalidNotificationFacts(
            NotificationFactFailureClass.missingOrMalformedRequiredFacts,
          ),
        },
        NotificationCategory.mention || NotificationCategory.quote =>
          switch (_parsePostUri(data['sourceUri'])) {
            final sourceUri? => ValidNotificationFacts._(
              category: category,
              sourceUri: sourceUri,
            ),
            null => const InvalidNotificationFacts(
              NotificationFactFailureClass.missingOrMalformedRequiredFacts,
            ),
          },
        NotificationCategory.reply => switch ((
          _parsePostUri(data['subjectUri']),
          _parsePostUri(data['sourceUri']),
        )) {
          (final subjectUri?, final sourceUri?) => ValidNotificationFacts._(
            category: category,
            subjectUri: subjectUri,
            sourceUri: sourceUri,
          ),
          _ => const InvalidNotificationFacts(
            NotificationFactFailureClass.missingOrMalformedRequiredFacts,
          ),
        },
        NotificationCategory.everythingElse => const ValidNotificationFacts._(
          category: NotificationCategory.everythingElse,
        ),
        NotificationCategory.unknown => const UnknownNotificationFacts(),
      };
    }

    return NotificationOpenAttempt(
      accountSubscriptionId: accountSubscriptionId,
      facts: facts,
      source: source,
    );
  }

  static final _typePattern = RegExp(r'^[A-Za-z][A-Za-z0-9]{0,63}$');
  static final _recordKeyPattern = RegExp(
    r"^[A-Za-z0-9._~:@!$&'()*+,;=-]{1,512}$",
  );

  final AccountSubscriptionId? accountSubscriptionId;
  final NotificationFactOutcome facts;
  final NotificationOpenSource source;

  static AccountSubscriptionId? _parseAccountSubscriptionId(Object? value) {
    if (value is! String) return null;
    try {
      return AccountSubscriptionId.parse(value);
    } on FormatException {
      return null;
    }
  }

  static Did? _parseDid(Object? value) {
    if (value is! String || !_isBoundedAscii(value, 1024)) return null;
    try {
      return Did.parse(value);
    } on Object {
      return null;
    }
  }

  static AtUri? _parsePostUri(Object? value) {
    if (value is! String || !_isBoundedAscii(value, 1024)) return null;
    try {
      at_uri.ensureValidAtUri(value);
      final parts = value.substring('at://'.length).split('/');
      if (parts.length != 3 ||
          parts[1] != 'social.craftsky.feed.post' ||
          !_recordKeyPattern.hasMatch(parts[2]) ||
          parts[2] == '.' ||
          parts[2] == '..') {
        return null;
      }
      Did.parse(parts[0]);
      return AtUri.parse(value);
    } on Object {
      return null;
    }
  }

  static bool _isBoundedAscii(String value, int maxBytes) =>
      value.isNotEmpty &&
      value.length <= maxBytes &&
      value.codeUnits.every((unit) => unit <= 0x7f);

  @override
  String toString() =>
      'NotificationOpenAttempt(source: $source, facts: $facts, '
      'accountSubscriptionId: <redacted>)';
}
