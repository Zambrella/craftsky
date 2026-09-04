import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/scheduled_posts/composer/schedule_composer_state.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';

Future<ScheduleChoice?> showScheduleChoiceMenu(
  BuildContext context, {
  required ScheduleChoice selectedChoice,
  required bool scheduleEnabled,
}) async {
  final l10n = AppLocalizations.of(context);
  ScheduleChoice? choice;

  await showCraftskyContextMenu(
    context,
    position: craftskyContextMenuAnchorPosition(context),
    groups: [
      CraftskyContextMenuGroup(
        items: [
          CraftskyContextMenuItem(
            text: l10n.scheduledPostNow,
            icon: CraftskyIconsBold.send,
            isSelected: selectedChoice == ScheduleChoice.now,
            onPressed: () => choice = ScheduleChoice.now,
          ),
          CraftskyContextMenuItem(
            text: l10n.scheduledPostLater,
            description: scheduleEnabled
                ? null
                : l10n.scheduledPostCapacityWarning,
            icon: CraftskyIconsBold.schedule,
            isSelected: selectedChoice == ScheduleChoice.later,
            onPressed: scheduleEnabled
                ? () => choice = ScheduleChoice.later
                : null,
          ),
        ],
      ),
    ],
  );

  return choice;
}
