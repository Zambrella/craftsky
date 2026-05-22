import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AuthCompletePage extends ConsumerStatefulWidget {
  const AuthCompletePage({required this.token, super.key});

  final String token;

  @override
  ConsumerState<AuthCompletePage> createState() => _AuthCompletePageState();
}

class _AuthCompletePageState extends ConsumerState<AuthCompletePage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      unawaited(
        ref
            .read(authControllerProvider.notifier)
            .completeFromDeepLink(widget.token),
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(authControllerProvider);

    return Scaffold(
      body: Center(
        child: switch (state) {
          AsyncError(:final error) when error is AuthError =>
            _AuthCompleteError(error: error),
          AsyncError(:final error) => _AuthCompleteError(
            error: GenericAuthError(error),
          ),
          _ => const _AuthCompleteLoading(),
        },
      ),
    );
  }
}

class _AuthCompleteLoading extends StatelessWidget {
  const _AuthCompleteLoading();

  @override
  Widget build(BuildContext context) {
    return const Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        StitchProgressIndicator(),
        SizedBox(height: 16),
        Text('Signing in…'),
      ],
    );
  }
}

class _AuthCompleteError extends StatelessWidget {
  const _AuthCompleteError({required this.error});

  final Object error;

  @override
  Widget build(BuildContext context) {
    final message = switch (error) {
      SignInTimedOut() => 'That sign-in link expired. Please sign in again.',
      NoPendingSignIn() => 'No sign-in is in progress. Please sign in again.',
      StorageFailure() =>
        "Couldn't save your session securely. Please sign in again.",
      _ => "Couldn't complete sign-in. Please sign in again.",
    };

    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(message, textAlign: TextAlign.center),
        ],
      ),
    );
  }
}

/// Stand-in for non-AuthError failures so the switch stays exhaustive.
class GenericAuthError implements Exception {
  const GenericAuthError(this.cause);
  final Object cause;
}
