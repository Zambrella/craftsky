import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:material_ui/material_ui.dart';

/// Outlined settings glyphs, with filled variants for avatar event badges.
IconData notificationCategoryIcon(
  NotificationCategory category, {
  bool filled = false,
}) => switch (category) {
  NotificationCategory.like =>
    filled ? CraftskyIconsFill.like : CraftskyIcons.like,
  NotificationCategory.follow =>
    filled ? CraftskyIconsFill.follow : CraftskyIcons.follow,
  NotificationCategory.reply =>
    filled ? CraftskyIconsFill.comment : CraftskyIcons.comment,
  NotificationCategory.mention =>
    filled ? CraftskyIconsFill.mention : CraftskyIcons.mention,
  NotificationCategory.quote =>
    filled ? CraftskyIconsFill.quote : CraftskyIcons.quote,
  NotificationCategory.repost =>
    filled ? CraftskyIconsFill.repost : CraftskyIcons.repost,
  NotificationCategory.instagramMatch =>
    filled ? CraftskyIconsFill.findPeople : CraftskyIcons.findPeople,
  NotificationCategory.moderation =>
    filled ? CraftskyIconsFill.privacy : CraftskyIcons.privacy,
  NotificationCategory.everythingElse || NotificationCategory.unknown =>
    filled ? CraftskyIconsFill.notification : CraftskyIcons.notification,
};
