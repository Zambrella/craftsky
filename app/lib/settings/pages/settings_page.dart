import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/widgets/clear_image_cache_tile.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SettingsPage extends ConsumerWidget {
  const SettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: const _SettingsPageBody(),
    );
  }
}

class _SettingsPageBody extends ConsumerWidget {
  const _SettingsPageBody();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return Column(
      children: [
        ListTile(
          leading: const Icon(Icons.bookmarks_outlined),
          title: Text(l10n.savedPostsTitle),
          onTap: () => const SavedPostsRoute().go(context),
        ),
        ListTile(
          leading: const Icon(Icons.group_outlined),
          title: const Text('Followers'),
          onTap: () => const FollowersRoute().go(context),
        ),
        ListTile(
          leading: const Icon(Icons.volume_off_outlined),
          title: Text(l10n.settingsMutedAccounts),
          onTap: () => const MutedAccountsRoute().go(context),
        ),
        ListTile(
          leading: const Icon(Icons.block_outlined),
          title: Text(l10n.settingsBlockedAccounts),
          onTap: () => const BlockedAccountsRoute().go(context),
        ),
        ListTile(
          leading: const Icon(Icons.person_add_alt_outlined),
          title: const Text('Following'),
          onTap: () => const FollowingRoute().go(context),
        ),
        const ClearImageCacheTile(),
        const SignOutTile(),
      ],
    );
  }
}
