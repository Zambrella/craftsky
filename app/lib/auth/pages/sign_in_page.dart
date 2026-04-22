import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SignInPage extends ConsumerStatefulWidget {
  const SignInPage({super.key});

  @override
  ConsumerState<SignInPage> createState() => _SignInPageState();
}

class _SignInPageState extends ConsumerState<SignInPage> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(authControllerProvider, (prev, next) {
      switch ((prev, next)) {
        case (AsyncLoading(), AsyncError(:final error)):
          final message = _messageFor(error);
          ScaffoldMessenger.of(context)
            ..clearSnackBars()
            ..showSnackBar(SnackBar(content: Text(message)));
        case _:
          break;
      }
    });

    final state = ref.watch(authControllerProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Sign in')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            BrandTextField(
              label: 'Handle',
              hintText: 'alice.bsky.social',
              controller: _controller,
              onSubmitted: (_) => _submit(),
            ),
            const SizedBox(height: 24),
            ChunkyButton(
              onPressed: state is AsyncLoading ? null : _submit,
              child: state is AsyncLoading
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Continue'),
            ),
          ],
        ),
      ),
    );
  }

  void _submit() {
    ref.read(authControllerProvider.notifier).signIn(handle: _controller.text);
  }

  String _messageFor(Object? error) => switch (error) {
        HandleRequired() => 'Please enter a handle.',
        InvalidHandle() => "We couldn't recognise that handle.",
        ServerUnavailable() => "Couldn't reach the server. Please try again.",
        BrowserLaunchFailed() =>
          "Couldn't open the browser. Check that you have one installed.",
        _ => 'Something went wrong. Please try again.',
      };
}
