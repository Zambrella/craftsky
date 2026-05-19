import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class OnboardingPage extends ConsumerWidget {
  const OnboardingPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Onboarding')),
      body: const Center(child: _OnboardingPageBody()),
    );
  }
}

class _OnboardingPageBody extends ConsumerWidget {
  const _OnboardingPageBody();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authSessionProvider).value;
    final did = switch (auth) {
      SignedIn(:final did) => did,
      _ => null,
    };

    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Onboarding'),
        const SizedBox(height: 24),
        ChunkyButton(
          onPressed: did == null
              ? null
              : () => ref.read(onboardingStatusProvider(did).notifier).finish(),
          child: const Text('Finish'),
        ),
      ],
    );
  }
}
