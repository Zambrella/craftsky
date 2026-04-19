import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class OnboardingPage extends ConsumerWidget {
  const OnboardingPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO(craftsky): l10n
    return Scaffold(
      appBar: AppBar(title: const Text('Onboarding')),
      body: const Center(child: OnboardingPageBody()),
    );
  }
}

class OnboardingPageBody extends ConsumerWidget {
  const OnboardingPageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Onboarding'),
        const SizedBox(height: 24),
        ElevatedButton(
          onPressed: () => ref.read(onboardingStatusProvider.notifier).finish(),
          child: const Text('Finish'),
        ),
      ],
    );
  }
}
