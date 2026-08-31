import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/widgets/registration_action.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class WelcomePage extends ConsumerWidget {
  const WelcomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.welcomeTitle)),
      body: const Center(child: _WelcomePageBody()),
    );
  }
}

class _WelcomePageBody extends ConsumerWidget {
  const _WelcomePageBody();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    ref.listen(authControllerProvider, (previous, next) {
      if (previous case AsyncLoading()) {
        if (next case AsyncError(:final error)) {
          context.showError(_messageFor(l10n, error));
        }
      }
    });
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Text(l10n.welcomeTitle),
        SizedBox(height: spacing.sp5),
        ChunkyButton(
          onPressed: () => const SignInRoute().go(context),
          child: Text(l10n.welcomeSignInAction),
        ),
        SizedBox(height: spacing.sp2),
        RegistrationAction(
          isLoading: ref.watch(authControllerProvider) is AsyncLoading,
          onPressed: () => unawaited(
            ref.read(authControllerProvider.notifier).startRegistration(),
          ),
        ),
      ],
    );
  }

  String _messageFor(AppLocalizations l10n, Object? error) => switch (error) {
    RegistrationFailure.canceled => l10n.authRegistrationCanceledError,
    RegistrationFailure.providerUnavailable =>
      l10n.authRegistrationProviderUnavailableError,
    RegistrationFailure.registrationIncomplete =>
      l10n.authRegistrationIncompleteError,
    AccountLimitReached() => l10n.accountSwitcherMaximum,
    _ => l10n.signInGenericError,
  };
}
