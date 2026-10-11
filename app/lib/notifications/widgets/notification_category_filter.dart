import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/providers/notification_filter_provider.dart';
import 'package:craftsky_app/notifications/widgets/notification_category_icon.dart';
import 'package:craftsky_app/notifications/widgets/notification_category_label.dart';
import 'package:craftsky_app/shared/widgets/sort_menu_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class NotificationCategoryFilter extends ConsumerWidget {
  const NotificationCategoryFilter({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selected = ref.watch(notificationFilterProvider);
    final account = ref
        .watch(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session
        .account;
    final l10n = AppLocalizations.of(context);
    return SortMenuButton<NotificationCategory?>(
      selectedValue: selected,
      options: [
        SortMenuOption(
          value: null,
          label: l10n.notificationFilterAll,
          icon: CraftskyIcons.notifications,
        ),
        for (final category in NotificationCategory.preferenceValues)
          if (category != NotificationCategory.everythingElse)
            SortMenuOption(
              value: category,
              label: notificationCategoryLabel(l10n, category),
              icon: notificationCategoryIcon(category),
            ),
      ],
      onChanged: (category) {
        if (!context.mounted) return;
        if (ref
                .read(sessionRegistryProvider)
                .value
                ?.activeLease
                ?.session
                .account !=
            account) {
          return;
        }
        ref.read(notificationFilterProvider.notifier).select(category);
      },
    );
  }
}
