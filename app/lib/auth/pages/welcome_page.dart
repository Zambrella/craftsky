import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class WelcomePage extends ConsumerWidget {
  const WelcomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
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
        ChunkyButton(
          onPressed: () => const SignInRoute().go(context),
          child: const Text('Sign in'),
        ),
        const SizedBox(height: 8),
        TextButton(
          onPressed: () => const SignInRoute().go(context),
          child: const Text('Create account on a PDS'),
        ),
      ],
    );
  }
}
