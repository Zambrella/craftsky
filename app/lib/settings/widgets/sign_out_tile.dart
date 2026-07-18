import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
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
      onTap: () async {
        final messenger = MessengerScope.of(context);
        final l10n = AppLocalizations.of(context);
        final result = await ref
            .read(authControllerProvider.notifier)
            .signOut();
        if (result == null) return;
        final activeHandle = result.activeHandle;
        messenger.info(
          activeHandle == null
              ? l10n.signOutSuccess
              : l10n.signOutSuccessWithAccount(activeHandle),
        );
      },
    );
  }
}
