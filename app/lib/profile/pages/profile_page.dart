import 'package:craftsky_app/router/router.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ProfilePage extends ConsumerWidget {
  const ProfilePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO(craftsky): l10n
    return Scaffold(
      appBar: AppBar(title: const Text('Profile')),
      body: const Center(child: ProfilePageBody()),
    );
  }
}

class ProfilePageBody extends StatelessWidget {
  const ProfilePageBody({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Profile'),
        const SizedBox(height: 24),
        OutlinedButton(
          onPressed: () => const SavedRoute().go(context),
          child: const Text('Saved'),
        ),
        const SizedBox(height: 8),
        OutlinedButton(
          onPressed: () => const SettingsRoute().go(context),
          child: const Text('Settings'),
        ),
        const SizedBox(height: 8),
        OutlinedButton(
          onPressed: () =>
              const UserProfileRoute(handle: 'alice.bsky.social').go(context),
          child: const Text('Open a user profile'),
        ),
      ],
    );
  }
}
