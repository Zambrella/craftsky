import 'package:dart_mappable/dart_mappable.dart';

part 'notification_category.mapper.dart';

@MappableEnum(defaultValue: NotificationCategory.unknown)
enum NotificationCategory {
  like,
  follow,
  reply,
  mention,
  quote,
  repost,
  everythingElse,
  unknown;

  static const List<NotificationCategory> preferenceValues = [
    like,
    follow,
    reply,
    mention,
    quote,
    repost,
    everythingElse,
  ];

  static NotificationCategory fromWireValue(String value) =>
      NotificationCategoryMapper.fromValue(value);

  static NotificationCategory? tryParsePreference(String value) {
    final category = fromWireValue(value);
    return category == unknown ? null : category;
  }

  String get wireValue => toValue();
}
