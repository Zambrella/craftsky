import 'package:craftsky_app/router/route_locations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class ProfilePage extends ConsumerWidget {
  const ProfilePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO: l10n
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
          onPressed: () => context.go(
            '${RouteLocations.profile}/${RouteLocations.savedChild}',
          ),
          child: const Text('Saved'),
        ),
        const SizedBox(height: 8),
        OutlinedButton(
          onPressed: () => context.go(RouteLocations.settings),
          child: const Text('Settings'),
        ),
        const SizedBox(height: 8),
        OutlinedButton(
          onPressed: () =>
              context.go('${RouteLocations.profile}/alice.bsky.social'),
          child: const Text('Open a user profile'),
        ),
      ],
    );
  }
}
