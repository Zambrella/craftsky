import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SignInPage extends ConsumerWidget {
  const SignInPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO: l10n
    return Scaffold(
      appBar: AppBar(title: const Text('Sign in')),
      body: const Padding(
        padding: EdgeInsets.all(24),
        child: SignInPageBody(),
      ),
    );
  }
}

class SignInPageBody extends ConsumerWidget {
  const SignInPageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const TextField(
          decoration: InputDecoration(
            labelText: 'Handle',
            hintText: 'alice.bsky.social',
          ),
        ),
        const SizedBox(height: 24),
        ElevatedButton(
          onPressed: () => ref.read(authStatusProvider.notifier).signIn(),
          child: const Text('Continue'),
        ),
      ],
    );
  }
}
