import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SettingsPage extends ConsumerWidget {
  const SettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO(craftsky): l10n
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: const Center(child: SettingsPageBody()),
    );
  }
}

class SettingsPageBody extends ConsumerWidget {
  const SettingsPageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Settings'),
        const SizedBox(height: 24),
        OutlinedButton(
          onPressed: () => ref.read(authStatusProvider.notifier).signOut(),
          child: const Text('Sign out (dev)'),
        ),
      ],
    );
  }
}
