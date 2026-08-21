import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AuthCompletePage extends ConsumerStatefulWidget {
  const AuthCompletePage({this.code, this.error, super.key});

  final String? code;
  final String? error;

  @override
  ConsumerState<AuthCompletePage> createState() => _AuthCompletePageState();
}

class _AuthCompletePageState extends ConsumerState<AuthCompletePage> {
  @override
  void initState() {
    super.initState();
    if (widget.error != null) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      unawaited(_complete());
    });
  }

  Future<void> _complete() async {
    final code = widget.code;
    if (code == null || code.isEmpty) return;
    await ref.read(authControllerProvider.notifier).completeFromDeepLink(code);
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(authControllerProvider, (previous, next) {
      if (previous case AsyncLoading() when next is AsyncData<void>) {
        const FeedRoute().go(context);
      }
    });
    final state = ref.watch(authControllerProvider);

    if (widget.error == 'account_deletion_pending') {
      return Scaffold(
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Text(
              AppLocalizations.of(context).accountDeletionAlreadyInProgress,
              textAlign: TextAlign.center,
            ),
          ),
        ),
      );
    }

    if (widget.code == null || widget.code!.isEmpty) {
      return const Scaffold(
        body: Center(child: _AuthCompleteError(error: SignInTimedOut())),
      );
    }

    return Scaffold(
      body: Center(
        child: switch (state) {
          AsyncError(:final error) when error is AuthError =>
            _AuthCompleteError(
              error: error,
              onRetry: error is ServerUnavailable || error is StorageFailure
                  ? _complete
                  : null,
            ),
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
  const _AuthCompleteError({required this.error, this.onRetry});

  final Object error;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final message = switch (error) {
      SignInTimedOut() => l10n.authCompleteTimedOutError,
      StorageFailure() => l10n.authCompleteStorageError,
      _ => l10n.authCompleteGenericError,
    };

    return Padding(
      padding: EdgeInsets.all(spacing.sp5),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(message, textAlign: TextAlign.center),
          if (onRetry case final retry?) ...[
            SizedBox(height: spacing.sp4),
            TextButton(onPressed: retry, child: Text(l10n.retryButton)),
          ],
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
