import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:flutter/material.dart';

/// Shared outlined action glyphs for notification settings and activity rows.
IconData notificationCategoryIcon(NotificationCategory category) =>
    switch (category) {
      NotificationCategory.like => Icons.favorite_outline,
      NotificationCategory.follow => Icons.person_add_alt_outlined,
      NotificationCategory.reply => Icons.chat_bubble_outline,
      NotificationCategory.mention => Icons.alternate_email,
      NotificationCategory.quote => Icons.format_quote,
      NotificationCategory.repost => Icons.repeat,
      NotificationCategory.instagramMatch => Icons.people_outline,
      NotificationCategory.everythingElse ||
      NotificationCategory.unknown => Icons.notifications_none,
    };
