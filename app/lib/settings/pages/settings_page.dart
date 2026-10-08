import 'dart:async';

import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/account_activation_coordinator.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/auth/widgets/account_avatar.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_launcher.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/models/settings_identity.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/subscription_build_config.dart';
import 'package:craftsky_app/subscriptions/widgets/plus_feature_lock.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:craftsky_app/theme/theme_notifier.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SettingsPage extends ConsumerWidget {
  const SettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final registry = ref.watch(sessionRegistryProvider).value;
    final activeLease = registry?.activeLease?.session;
    final activeSession = activeLease == null
        ? null
        : registry?.sessions[activeLease.account.did];
    final loadedIdentity = ref.watch(activeAccountIdentityProvider).value;
    final auth = ref.watch(authSessionProvider).value;
    final themeMode = ref.watch(themeModeProvider);
    final subscriptionAccess = activeLease == null
        ? null
        : ref.watch(subscriptionAccessProvider(activeLease));
    final hasPaidAccess = switch (subscriptionAccess) {
      AsyncData(:final value)
          when !subscriptionAccess.isLoading &&
              !subscriptionAccess.hasError &&
              value.did == activeLease?.account.did =>
        value.allowsPlus,
      _ => false,
    };

    SettingsIdentity? identity;
    if (activeLease != null && activeSession != null) {
      identity = projectSettingsIdentity(
        lease: activeLease,
        session: activeSession,
        unavailableHandleLabel: l10n.handleUnavailable,
        loaded: loadedIdentity,
      );
    } else if (auth is SignedIn) {
      final profile = loadedIdentity?.profile;
      final displayName = profile?.displayName?.trim();
      final handleLabel = ProfileHandle(
        profile?.handle ?? auth.handle,
      ).currentLabel(unavailableLabel: l10n.handleUnavailable);
      identity = SettingsIdentity(
        primaryLabel: displayName == null || displayName.isEmpty
            ? handleLabel
            : displayName,
        secondaryLabel: displayName == null || displayName.isEmpty
            ? null
            : handleLabel,
        handleLabel: handleLabel,
        avatarUrl: profile?.avatar,
        avatarSeed: auth.did.value,
        customisation: profile?.customisation ?? ProfileCustomisation.defaults,
      );
    }

    final switcherState = registry == null
        ? null
        : AccountSwitcherState.fromRegistry(registry);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.settingsTitle)),
      body: ListView(
        children: [
          if (identity != null) _SettingsIdentityHeader(identity: identity),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.switchAccount,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsSwitchAccount,
            leading: CraftskyIconsBold.switchAccount,
            onTap: switcherState == null
                ? null
                : () => unawaited(_openSwitcher(context, ref, switcherState)),
          ),
          if (subscriptionsEnabled)
            _SubscriptionCallout(hasPaidAccess: hasPaidAccess),
          _SectionLabel(l10n.settingsSectionPreferences),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.appearance,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.appearanceTitle,
            leading: CraftskyIcons.appearance,
            subtitle: switch (themeMode) {
              ThemeMode.system => l10n.appearanceUseDeviceSetting,
              ThemeMode.light => l10n.appearanceLight,
              ThemeMode.dark => l10n.appearanceDark,
            },
            onTap: () => const AppearanceRoute().go(context),
          ),
          PlusFeatureLock(
            feature: l10n.profileCustomisationTitle,
            access: subscriptionAccess,
            showPlusBadge: false,
            onUnlocked: () => const ProfileCustomisationRoute().go(context),
            onRetry: activeLease == null
                ? null
                : () => ref.invalidate(subscriptionAccessProvider(activeLease)),
            child: SettingsRowTile(
              descriptor: const SettingsRowDescriptor(
                id: SettingsRowId.customisation,
                kind: SettingsRowKind.disclosure,
              ),
              label: l10n.profileCustomisationTitle,
              leading: CraftskyIcons.palette,
              locked: subscriptionAccess?.value?.allowsPlus != true,
              onTap: () => const ProfileCustomisationRoute().go(context),
            ),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.languages,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsLanguages,
            leading: CraftskyIcons.language,
            onTap: () => const LanguagesRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.notifications,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsNotifications,
            leading: CraftskyIcons.notifications,
            onTap: () => unawaited(
              const NotificationSettingsRoute().push<void>(context),
            ),
          ),
          _SectionLabel(l10n.settingsSectionConnections),
          PlusFeatureLock(
            feature: l10n.settingsGrowth,
            access: subscriptionAccess,
            showPlusBadge: false,
            onUnlocked: () => const FollowerGrowthRoute().go(context),
            onRetry: activeLease == null
                ? null
                : () => ref.invalidate(subscriptionAccessProvider(activeLease)),
            child: SettingsRowTile(
              descriptor: const SettingsRowDescriptor(
                id: SettingsRowId.growth,
                kind: SettingsRowKind.disclosure,
              ),
              label: l10n.settingsGrowth,
              leading: CraftskyIcons.trending,
              locked: subscriptionAccess?.value?.allowsPlus != true,
              onTap: () => const FollowerGrowthRoute().go(context),
            ),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.followers,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsFollowers,
            leading: CraftskyIcons.people,
            onTap: () => const FollowersRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.following,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsFollowing,
            leading: CraftskyIcons.follow,
            onTap: () => const FollowingRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.mutedAccounts,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsMutedAccounts,
            leading: CraftskyIcons.muted,
            onTap: () => const MutedAccountsRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.blockedAccounts,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsBlockedAccounts,
            leading: CraftskyIconsBold.block,
            onTap: () => const BlockedAccountsRoute().go(context),
          ),
          _SectionLabel(l10n.settingsSectionDiscovery),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.findPeopleFromInstagram,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.instagramMigrationTitle,
            leading: CraftskyIcons.camera,
            onTap: () => const InstagramMigrationRoute().go(context),
            subtitle: l10n.instagramMigrationSettingsSubtitle,
          ),
          if (subscriptionsEnabled)
            if (subscriptionAccess
                case AsyncData(
                  :final value,
                )
                when !subscriptionAccess.isLoading &&
                    !subscriptionAccess.hasError &&
                    value.did == activeLease?.account.did &&
                    value.allowsBusiness) ...[
              _SectionLabel(l10n.settingsSectionBusiness),
              SettingsRowTile(
                descriptor: const SettingsRowDescriptor(
                  id: SettingsRowId.businessEvents,
                  kind: SettingsRowKind.disclosure,
                ),
                label: l10n.settingsBusinessEvents,
                leading: CraftskyIcons.events,
                onTap: () => const BusinessEventsRoute().go(context),
              ),
              SettingsRowTile(
                descriptor: const SettingsRowDescriptor(
                  id: SettingsRowId.businessProducts,
                  kind: SettingsRowKind.disclosure,
                ),
                label: l10n.settingsBusinessProducts,
                leading: CraftskyIcons.storefront,
                onTap: () => const BusinessProductsRoute().go(context),
              ),
            ],
          _SectionLabel(l10n.settingsSectionGeneral),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.accountStanding,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.accountStandingTitle,
            leading: CraftskyIcons.privacy,
            onTap: () => const AccountStandingRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.account,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsAccount,
            leading: CraftskyIcons.accountSettings,
            onTap: () => const AccountRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.about,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsAbout,
            leading: CraftskyIcons.info,
            onTap: () => const AboutRoute().go(context),
          ),
          const Divider(),
          const SignOutTile(),
          const SizedBox(height: 24),
        ],
      ),
    );
  }

  Future<void> _openSwitcher(
    BuildContext context,
    WidgetRef ref,
    AccountSwitcherState state,
  ) async {
    final activation = AccountActivationCoordinator(
      readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
      commitActivation: ref.read(sessionRegistryProvider.notifier).activate,
      invalidateAccountState: ref.read(accountStateInvalidatorProvider),
      resetToHome: () async => const FeedRoute().go(context),
      confirmLeave: ref.read(unsavedWorkGuardProvider).confirmLeave,
    );
    await showAccountSwitcherSheet(
      context: context,
      fallbackState: state,
      onSelect: activation.activate,
      onAddAccount: () {
        Navigator.pop(context);
        unawaited(const AddAccountRoute().push<void>(context));
      },
    );
  }
}

class _SubscriptionCallout extends StatelessWidget {
  const _SubscriptionCallout({required this.hasPaidAccess});

  final bool hasPaidAccess;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final title = hasPaidAccess
        ? l10n.settingsSubscriptionCalloutActiveTitle
        : l10n.settingsSubscriptionCalloutTitle;
    final description = hasPaidAccess
        ? l10n.settingsSubscriptionCalloutActiveDescription
        : l10n.settingsSubscriptionCalloutDescription;
    final action = hasPaidAccess
        ? l10n.settingsSubscriptionCalloutView
        : l10n.settingsSubscriptionCalloutExplore;
    final radius = theme.extension<RadiusTheme>()?.r3 ?? const RadiusTheme().r3;
    final shadow =
        theme.extension<BrandShadowTheme>()?.dropSm ??
        const BrandShadowTheme().dropSm;
    return Semantics(
      button: true,
      label: '$title. $description. $action',
      excludeSemantics: true,
      child: Container(
        key: const Key('settings-subscription-callout'),
        margin: const EdgeInsetsDirectional.fromSTEB(16, 12, 16, 4),
        decoration: BoxDecoration(
          color: Color.alphaBlend(
            colors.primary.withValues(alpha: 0.1),
            colors.surface,
          ),
          border: Border.all(color: colors.onSurface, width: 1.5),
          borderRadius: BorderRadius.circular(radius),
          boxShadow: shadow,
        ),
        clipBehavior: Clip.antiAlias,
        child: Material(
          type: MaterialType.transparency,
          child: InkWell(
            onTap: () => const SubscriptionsRoute().go(context),
            excludeFromSemantics: true,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Icon(
                        CraftskyIcons.plusTier,
                        color: colors.primary,
                        size: 28,
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Text(title, style: theme.textTheme.titleMedium),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Text(description, style: theme.textTheme.bodyMedium),
                  const SizedBox(height: 12),
                  Container(
                    width: double.infinity,
                    constraints: const BoxConstraints(minHeight: 48),
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: colors.primary,
                      borderRadius: BorderRadius.circular(radius),
                    ),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 12,
                        vertical: 8,
                      ),
                      child: Text(
                        action,
                        textAlign: TextAlign.center,
                        style: theme.textTheme.labelLarge?.copyWith(
                          color: colors.onPrimary,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _SettingsIdentityHeader extends StatelessWidget {
  const _SettingsIdentityHeader({required this.identity});

  final SettingsIdentity identity;

  @override
  Widget build(BuildContext context) => ListTile(
    contentPadding: const EdgeInsetsDirectional.fromSTEB(16, 12, 16, 8),
    leading: AccountAvatar(
      avatarUrl: identity.avatarUrl,
      seed: identity.avatarSeed,
      customisation: identity.customisation,
    ),
    title: Text(
      identity.primaryLabel,
      style: Theme.of(context).textTheme.titleMedium,
    ),
    subtitle: identity.secondaryLabel == null
        ? null
        : Text(identity.secondaryLabel!),
  );
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.label);

  final String label;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsetsDirectional.fromSTEB(16, 20, 16, 4),
    child: Text(
      label,
      style: Theme.of(context).textTheme.titleSmall?.copyWith(
        color: Theme.of(context).colorScheme.primary,
      ),
    ),
  );
}
