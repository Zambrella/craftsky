import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';

/// Shared outlined action glyphs for notification settings and activity rows.
IconData notificationCategoryIcon(NotificationCategory category) =>
    switch (category) {
      NotificationCategory.like => CraftskyIcons.like,
      NotificationCategory.follow => CraftskyIcons.follow,
      NotificationCategory.reply => CraftskyIcons.comment,
      NotificationCategory.mention => CraftskyIcons.mention,
      NotificationCategory.quote => CraftskyIcons.quote,
      NotificationCategory.repost => CraftskyIcons.repost,
      NotificationCategory.instagramMatch => CraftskyIcons.findPeople,
      NotificationCategory.everythingElse ||
      NotificationCategory.unknown => CraftskyIcons.notification,
    };
