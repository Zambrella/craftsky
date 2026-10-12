import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'notification_preferences.mapper.dart';

@MappableEnum()
enum NotificationPreferenceScope {
  everyone,
  peopleIFollow;

  String get wireValue => toValue();
}

@MappableClass()
final class NotificationPreference with NotificationPreferenceMappable {
  const new({
    required this.scope,
    required this.pushEnabled,
  });

  final NotificationPreferenceScope scope;
  final bool pushEnabled;
}

final class NotificationPreferences {
  const new({required this.known, required this.unknown});

  factory fromMap(Map<String, dynamic> map) {
    final raw = map['preferences'] as Map<String, dynamic>;
    final known = <NotificationCategory, NotificationPreference>{};
    final unknown = <String, Object?>{};
    for (final MapEntry(:key, :value) in raw.entries) {
      final category = NotificationCategory.tryParsePreference(key);
      if (category == null) {
        unknown[key] = value;
      } else {
        known[category] = NotificationPreferenceMapper.fromMap(
          value as Map<String, dynamic>,
        );
      }
    }
    return NotificationPreferences(known: known, unknown: unknown);
  }

  final Map<NotificationCategory, NotificationPreference> known;
  final Map<String, Object?> unknown;

  NotificationPreferences replace(
    NotificationCategory category,
    NotificationPreference value,
  ) => NotificationPreferences(
    known: {...known, category: value},
    unknown: unknown,
  );
}

enum NotificationPreferenceField { scope, pushEnabled }

final class NotificationPreferencePatch {
  const new _({
    required this.category,
    required this.field,
    this.scopeValue,
    this.pushEnabledValue,
  });

  factory scope(
    NotificationCategory category, {
    required NotificationPreferenceScope value,
  }) => NotificationPreferencePatch._(
    category: category,
    field: NotificationPreferenceField.scope,
    scopeValue: value,
  );

  factory pushEnabled(
    NotificationCategory category, {
    required bool value,
  }) => NotificationPreferencePatch._(
    category: category,
    field: NotificationPreferenceField.pushEnabled,
    pushEnabledValue: value,
  );

  final NotificationCategory category;
  final NotificationPreferenceField field;
  final NotificationPreferenceScope? scopeValue;
  final bool? pushEnabledValue;

  Map<String, Object?> toMap() => {
    'preferences': {
      category.wireValue: switch (field) {
        NotificationPreferenceField.scope => {
          'scope': scopeValue!.wireValue,
        },
        NotificationPreferenceField.pushEnabled => {
          'pushEnabled': pushEnabledValue,
        },
      },
    },
  };
}
