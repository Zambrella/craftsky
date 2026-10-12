import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';

String notificationCategoryLabel(
  AppLocalizations l10n,
  NotificationCategory category,
) => switch (category) {
  NotificationCategory.like => l10n.notificationCategoryLikes,
  NotificationCategory.follow => l10n.notificationCategoryFollows,
  NotificationCategory.reply => l10n.notificationCategoryReplies,
  NotificationCategory.mention => l10n.notificationCategoryMentions,
  NotificationCategory.quote => l10n.notificationCategoryQuotes,
  NotificationCategory.repost => l10n.notificationCategoryReposts,
  NotificationCategory.instagramMatch =>
    l10n.notificationCategoryInstagramMatches,
  NotificationCategory.moderation => l10n.notificationCategoryModeration,
  NotificationCategory.everythingElse =>
    l10n.notificationCategoryEverythingElse,
  NotificationCategory.unknown => l10n.notificationCategoryEverythingElse,
};
