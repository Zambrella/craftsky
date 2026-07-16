// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'notification_category.dart';

class NotificationCategoryMapper extends EnumMapper<NotificationCategory> {
  NotificationCategoryMapper._();

  static NotificationCategoryMapper? _instance;
  static NotificationCategoryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = NotificationCategoryMapper._());
    }
    return _instance!;
  }

  static NotificationCategory fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  NotificationCategory decode(dynamic value) {
    switch (value) {
      case r'like':
        return NotificationCategory.like;
      case r'follow':
        return NotificationCategory.follow;
      case r'reply':
        return NotificationCategory.reply;
      case r'mention':
        return NotificationCategory.mention;
      case r'quote':
        return NotificationCategory.quote;
      case r'repost':
        return NotificationCategory.repost;
      case r'everythingElse':
        return NotificationCategory.everythingElse;
      case r'unknown':
        return NotificationCategory.unknown;
      default:
        return NotificationCategory.values[7];
    }
  }

  @override
  dynamic encode(NotificationCategory self) {
    switch (self) {
      case NotificationCategory.like:
        return r'like';
      case NotificationCategory.follow:
        return r'follow';
      case NotificationCategory.reply:
        return r'reply';
      case NotificationCategory.mention:
        return r'mention';
      case NotificationCategory.quote:
        return r'quote';
      case NotificationCategory.repost:
        return r'repost';
      case NotificationCategory.everythingElse:
        return r'everythingElse';
      case NotificationCategory.unknown:
        return r'unknown';
    }
  }
}

extension NotificationCategoryMapperExtension on NotificationCategory {
  String toValue() {
    NotificationCategoryMapper.ensureInitialized();
    return MapperContainer.globals.toValue<NotificationCategory>(this)
        as String;
  }
}

