import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
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
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const StitchProgressIndicator(),
        SizedBox(height: spacing.sp4),
        Text(l10n.authCompleteSigningIn),
      ],
    );
  }
}

class _AuthCompleteError extends StatelessWidget {
  const _AuthCompleteError({required this.error});

  final Object error;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final message = switch (error) {
      SignInTimedOut() => l10n.authCompleteTimedOutError,
      NoPendingSignIn() => l10n.authCompleteNoPendingSignInError,
      StorageFailure() => l10n.authCompleteStorageError,
      _ => l10n.authCompleteGenericError,
    };

    return Padding(
      padding: EdgeInsets.all(spacing.sp5),
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
