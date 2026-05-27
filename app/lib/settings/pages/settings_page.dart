import 'package:craftsky_app/settings/pages/follow_list_page.dart';
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
    return Column(
      children: [
        ListTile(
          leading: const Icon(Icons.group_outlined),
          title: const Text('Followers'),
          onTap: () => Navigator.of(context).push(
            MaterialPageRoute<void>(
              builder: (_) =>
                  const FollowListPage(kind: FollowListKind.followers),
            ),
          ),
        ),
        ListTile(
          leading: const Icon(Icons.person_add_alt_outlined),
          title: const Text('Following'),
          onTap: () => Navigator.of(context).push(
            MaterialPageRoute<void>(
              builder: (_) =>
                  const FollowListPage(kind: FollowListKind.following),
            ),
          ),
        ),
        const ClearImageCacheTile(),
        const SignOutTile(),
      ],
    );
  }
}
