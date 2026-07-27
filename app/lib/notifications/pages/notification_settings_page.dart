import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/providers/notification_permission_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_preferences_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/widgets/notification_category_icon.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class NotificationSettingsPage extends ConsumerWidget {
  const NotificationSettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final account = ref
        .watch(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session
        .account;
    final preferences = account == null
        ? ref.watch(notificationPreferencesProvider)
        : ref.watch(accountNotificationPreferencesProvider(account));
    final permission = ref.watch(notificationPermissionProvider).value;
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.notificationSettingsAction)),
      body: switch (preferences) {
        AsyncData(:final value) => _SettingsContent(
          preferences: value,
          permission: permission,
          account: account,
        ),
        AsyncError() => Center(
          child: FilledButton(
            onPressed: () {
              if (account != null) {
                ref.invalidate(
                  accountNotificationPreferencesProvider(account),
                );
              } else {
                ref.invalidate(notificationPreferencesProvider);
              }
            },
            child: Text(l10n.retryButton),
          ),
        ),
        _ => const Center(child: StitchProgressIndicator()),
      },
    );
  }
}

class _SettingsContent extends ConsumerWidget {
  const _SettingsContent({
    required this.preferences,
    required this.permission,
    required this.account,
  });

  final NotificationPreferences preferences;
  final NotificationPermission? permission;
  final AccountKey? account;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return SingleChildScrollView(
      padding: EdgeInsets.fromLTRB(
        spacing.sp4,
        spacing.sp4,
        spacing.sp4,
        spacing.sp7,
      ),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 720),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                l10n.notificationSettingsIntro,
                style: theme.textTheme.bodyLarge?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              if (permission == NotificationPermission.denied) ...[
                SizedBox(height: spacing.sp4),
                _PermissionWarning(
                  onOpenSettings: ref
                      .read(notificationServiceProvider)
                      .openSystemNotificationSettings,
                ),
              ],
              SizedBox(height: spacing.sp5),
              for (final category in NotificationCategory.preferenceValues)
                if (preferences.known[category] case final preference?) ...[
                  _PreferenceSection(
                    category: category,
                    preference: preference,
                    account: account,
                  ),
                  SizedBox(height: spacing.sp4),
                ],
            ],
          ),
        ),
      ),
    );
  }
}

class _PermissionWarning extends StatelessWidget {
  const _PermissionWarning({required this.onOpenSettings});

  final VoidCallback onOpenSettings;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return CraftskyCard(
      key: const Key('notification-permission-warning'),
      padding: EdgeInsets.all(spacing.sp4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(
                Icons.notifications_off_outlined,
                color: theme.colorScheme.primary,
              ),
              SizedBox(width: spacing.sp3),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.notificationDeviceDisabled,
                      style: theme.textTheme.titleMedium,
                    ),
                    SizedBox(height: spacing.sp1),
                    Text(
                      l10n.notificationDeviceDisabledDescription,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          SizedBox(height: spacing.sp3),
          Align(
            alignment: AlignmentDirectional.centerEnd,
            child: OutlinedButton(
              onPressed: onOpenSettings,
              child: Text(l10n.notificationOpenSettings),
            ),
          ),
        ],
      ),
    );
  }
}

class _PreferenceSection extends ConsumerWidget {
  const _PreferenceSection({
    required this.category,
    required this.preference,
    required this.account,
  });

  final NotificationCategory category;
  final NotificationPreference preference;
  final AccountKey? account;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return CraftskyCard(
      key: Key('notification-${category.wireValue}-preference-card'),
      padding: EdgeInsets.all(spacing.sp4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Icon(
                notificationCategoryIcon(category),
                color: theme.colorScheme.primary,
              ),
              SizedBox(width: spacing.sp2),
              Expanded(
                child: Text(
                  _categoryLabel(l10n, category),
                  style: theme.textTheme.titleLarge,
                ),
              ),
            ],
          ),
          SizedBox(height: spacing.sp4),
          if (category == NotificationCategory.instagramMatch)
            Text(
              l10n.notificationInstagramMatchPreferenceDescription,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            )
          else
            CraftskySingleSelectInput<NotificationPreferenceScope>(
              label: l10n.notificationPreferenceFrom,
              keyPrefix: 'notification-${category.wireValue}-scope',
              value: preference.scope,
              options: [
                CraftskySelectOption(
                  value: NotificationPreferenceScope.everyone,
                  label: l10n.notificationScopeEveryone,
                ),
                CraftskySelectOption(
                  value: NotificationPreferenceScope.peopleIFollow,
                  label: l10n.notificationScopePeopleIFollow,
                ),
              ],
              onChanged: (value) async {
                if (value == null) return;
                final saved = account == null
                    ? await ref
                          .read(notificationPreferencesProvider.notifier)
                          .setScope(category, value: value)
                    : await ref
                          .read(
                            accountNotificationPreferencesProvider(
                              account!,
                            ).notifier,
                          )
                          .setScope(category, value: value);
                if (!saved && context.mounted) {
                  context.showError(
                    l10n.notificationPreferenceSaveError,
                  );
                }
              },
            ),
          SizedBox(height: spacing.sp3),
          Divider(color: theme.colorScheme.outlineVariant),
          MergeSemantics(
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    l10n.notificationPushEnabled,
                    style: theme.textTheme.titleMedium,
                  ),
                ),
                SizedBox(width: spacing.sp3),
                Switch(
                  key: Key(
                    'notification-${category.wireValue}-push-switch',
                  ),
                  value: preference.pushEnabled,
                  onChanged: (value) async {
                    final saved = account == null
                        ? await ref
                              .read(notificationPreferencesProvider.notifier)
                              .setPushEnabled(category, value: value)
                        : await ref
                              .read(
                                accountNotificationPreferencesProvider(
                                  account!,
                                ).notifier,
                              )
                              .setPushEnabled(category, value: value);
                    if (!saved && context.mounted) {
                      context.showError(l10n.notificationPreferenceSaveError);
                    }
                  },
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

String _categoryLabel(
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
  NotificationCategory.everythingElse =>
    l10n.notificationCategoryEverythingElse,
  NotificationCategory.unknown => l10n.notificationCategoryEverythingElse,
};
