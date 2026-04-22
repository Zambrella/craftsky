import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SignOutTile extends ConsumerWidget {
  const SignOutTile({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(authControllerProvider);
    return ListTile(
      leading: const Icon(Icons.logout),
      title: const Text('Sign out'),
      enabled: state is! AsyncLoading,
      onTap: () => ref.read(authControllerProvider.notifier).signOut(),
    );
  }
}
