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
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/models/settings_identity.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
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
          _row(
            context,
            SettingsRowId.customisation,
            l10n.profileCustomisationTitle,
            Icons.palette_outlined,
            () => const ProfileCustomisationRoute().go(context),
          ),
          _row(
            context,
            SettingsRowId.languages,
            l10n.settingsLanguages,
            Icons.language_outlined,
            () => const LanguagesRoute().go(context),
          ),
          _row(
            context,
            SettingsRowId.notifications,
            l10n.settingsNotifications,
            Icons.notifications_outlined,
            () => unawaited(
              const NotificationSettingsRoute().push<void>(context),
            ),
          ),
          _SectionLabel(l10n.settingsSectionConnections),
          _row(
            context,
            SettingsRowId.followers,
            l10n.settingsFollowers,
            Icons.group_outlined,
            () => const FollowersRoute().go(context),
          ),
          _row(
            context,
            SettingsRowId.following,
            l10n.settingsFollowing,
            Icons.person_add_alt_outlined,
            () => const FollowingRoute().go(context),
          ),
          _row(
            context,
            SettingsRowId.mutedAccounts,
            l10n.settingsMutedAccounts,
            Icons.volume_off_outlined,
            () => const MutedAccountsRoute().go(context),
          ),
          _row(
            context,
            SettingsRowId.blockedAccounts,
            l10n.settingsBlockedAccounts,
            Icons.block_outlined,
            () => const BlockedAccountsRoute().go(context),
          ),
          _SectionLabel(l10n.settingsSectionDiscovery),
          _row(
            context,
            SettingsRowId.findPeopleFromInstagram,
            l10n.instagramMigrationTitle,
            Icons.photo_camera_outlined,
            () => const InstagramMigrationRoute().go(context),
            subtitle: l10n.instagramMigrationSettingsSubtitle,
          ),
          _SectionLabel(l10n.settingsSectionGeneral),
          _row(
            context,
            SettingsRowId.account,
            l10n.settingsAccount,
            Icons.manage_accounts_outlined,
            () => const AccountRoute().go(context),
          ),
          _row(
            context,
            SettingsRowId.about,
            l10n.settingsAbout,
            Icons.info_outline,
            () => const AboutRoute().go(context),
          ),
          const Divider(),
          const SignOutTile(),
          const SizedBox(height: 24),
        ],
      ),
    );
  }

  SettingsRowTile _row(
    BuildContext context,
    SettingsRowId id,
    String label,
    IconData icon,
    VoidCallback onTap, {
    String? subtitle,
  }) => SettingsRowTile(
    descriptor: SettingsRowDescriptor(
      id: id,
      kind: SettingsRowKind.disclosure,
    ),
    label: label,
    subtitle: subtitle,
    leading: icon,
    onTap: onTap,
  );

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
