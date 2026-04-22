import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
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
    return const Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        CurrentUserSummary(),
        SizedBox(height: 24),
        _ProfileActions(),
      ],
    );
  }
}

/// Shows the signed-in user's handle and DID. Hidden when the auth
/// session is still resolving or in a transient SignedOut state
/// (the router redirect should pull the user off this page in the
/// latter case — this guard is defensive).
class CurrentUserSummary extends ConsumerWidget {
  const CurrentUserSummary({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authSessionProvider);
    return switch (auth) {
      AsyncData(value: final SignedIn signedIn) => _SignedInCard(
        handle: signedIn.handle,
        did: signedIn.did,
      ),
      _ => const SizedBox.shrink(),
    };
  }
}

class _SignedInCard extends StatelessWidget {
  const _SignedInCard({required this.handle, required this.did});

  final String handle;
  final String did;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24),
      child: Column(
        children: [
          Text('@$handle', style: theme.textTheme.titleMedium),
          const SizedBox(height: 4),
          SelectableText(
            did,
            style: theme.textTheme.bodySmall,
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }
}

class _ProfileActions extends StatelessWidget {
  const _ProfileActions();

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
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
        const SizedBox(height: 8),
        OutlinedButton(
          onPressed: () => const PlaygroundRoute().go(context),
          child: const Text('Design playground'),
        ),
      ],
    );
  }
}
