import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class WelcomePage extends ConsumerWidget {
  const WelcomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO(craftsky): l10n
    return Scaffold(
      appBar: AppBar(title: const Text('Welcome')),
      body: const Center(child: WelcomePageBody()),
    );
  }
}

class WelcomePageBody extends ConsumerWidget {
  const WelcomePageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Welcome'),
        const SizedBox(height: 24),
        ElevatedButton(
          onPressed: () => context.go(RouteLocations.signIn),
          child: const Text('Sign in'),
        ),
        const SizedBox(height: 8),
        TextButton(
          onPressed: () => context.go(RouteLocations.signIn),
          child: const Text('Create account on a PDS'),
        ),
        const SizedBox(height: 32),
        OutlinedButton(
          onPressed: () => ref.read(authStatusProvider.notifier).signIn(),
          child: const Text('Dev: toggle auth'),
        ),
      ],
    );
  }
}
