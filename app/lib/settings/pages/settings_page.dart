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
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/account_type_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/models/settings_identity.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
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
    final accountType =
        ref.watch(accountTypeControllerProvider).value ??
        loadedIdentity?.profile.accountType;
    final auth = ref.watch(authSessionProvider).value;
    final themeMode = ref.watch(themeModeProvider);

    SettingsIdentity? identity;
    if (activeLease != null && activeSession != null) {
      identity = projectSettingsIdentity(
        lease: activeLease,
        session: activeSession,
        loaded: loadedIdentity,
      );
    } else if (auth is SignedIn) {
      final profile = loadedIdentity?.profile;
      final displayName = profile?.displayName?.trim();
      final handleLabel = '@${auth.handle.value}';
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
            leading: Icons.switch_account_outlined,
            onTap: switcherState == null
                ? null
                : () => unawaited(_openSwitcher(context, ref, switcherState)),
          ),
          _SectionLabel(l10n.settingsSectionPreferences),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.appearance,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.appearanceTitle,
            leading: Icons.brightness_6_outlined,
            subtitle: switch (themeMode) {
              ThemeMode.system => l10n.appearanceUseDeviceSetting,
              ThemeMode.light => l10n.appearanceLight,
              ThemeMode.dark => l10n.appearanceDark,
            },
            onTap: () => const AppearanceRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.customisation,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.profileCustomisationTitle,
            leading: Icons.palette_outlined,
            onTap: () => const ProfileCustomisationRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.languages,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsLanguages,
            leading: Icons.language_outlined,
            onTap: () => const LanguagesRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.notifications,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsNotifications,
            leading: Icons.notifications_outlined,
            onTap: () => unawaited(
              const NotificationSettingsRoute().push<void>(context),
            ),
          ),
          _SectionLabel(l10n.settingsSectionConnections),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.growth,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsGrowth,
            leading: Icons.show_chart,
            onTap: () => const FollowerGrowthRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.followers,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsFollowers,
            leading: Icons.group_outlined,
            onTap: () => const FollowersRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.following,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsFollowing,
            leading: Icons.person_add_alt_outlined,
            onTap: () => const FollowingRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.mutedAccounts,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsMutedAccounts,
            leading: Icons.volume_off_outlined,
            onTap: () => const MutedAccountsRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.blockedAccounts,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsBlockedAccounts,
            leading: Icons.block_outlined,
            onTap: () => const BlockedAccountsRoute().go(context),
          ),
          _SectionLabel(l10n.settingsSectionDiscovery),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.findPeopleFromInstagram,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.instagramMigrationTitle,
            leading: Icons.photo_camera_outlined,
            onTap: () => const InstagramMigrationRoute().go(context),
            subtitle: l10n.instagramMigrationSettingsSubtitle,
          ),
          if (accountType == AccountType.business) ...[
            _SectionLabel(l10n.settingsSectionBusiness),
            SettingsRowTile(
              descriptor: const SettingsRowDescriptor(
                id: SettingsRowId.businessEvents,
                kind: SettingsRowKind.disclosure,
              ),
              label: l10n.settingsBusinessEvents,
              leading: Icons.event_outlined,
              onTap: () => const BusinessEventsRoute().go(context),
            ),
            SettingsRowTile(
              descriptor: const SettingsRowDescriptor(
                id: SettingsRowId.businessProducts,
                kind: SettingsRowKind.disclosure,
              ),
              label: l10n.settingsBusinessProducts,
              leading: Icons.storefront_outlined,
              onTap: () => const BusinessProductsRoute().go(context),
            ),
          ],
          _SectionLabel(l10n.settingsSectionGeneral),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.account,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsAccount,
            leading: Icons.manage_accounts_outlined,
            onTap: () => const AccountRoute().go(context),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.about,
              kind: SettingsRowKind.disclosure,
            ),
            label: l10n.settingsAbout,
            leading: Icons.info_outline,
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
