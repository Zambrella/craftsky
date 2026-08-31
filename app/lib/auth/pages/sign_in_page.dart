import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as model;
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/widgets/registration_action.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum SignInMode { signIn, addAccount }

class SignInPage extends ConsumerStatefulWidget {
  const SignInPage({super.key, this.mode = SignInMode.signIn});

  final SignInMode mode;

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
    final l10n = AppLocalizations.of(context);
    ref.listen(authControllerProvider, (prev, next) {
      switch ((prev, next)) {
        case (AsyncLoading(), AsyncError(:final error)):
          context.showError(_messageFor(l10n, error));
        case _:
          break;
      }
    });

    final state = ref.watch(authControllerProvider);
    final registry = widget.mode == SignInMode.addAccount
        ? ref.watch(sessionRegistryProvider).value
        : null;
    final canAddAccount =
        widget.mode != SignInMode.addAccount ||
        registry != null &&
            registry.sessions.length <
                model.SessionRegistry.maxRetainedAccounts;
    final busy = state is AsyncLoading;
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();

    return Scaffold(
      appBar: AppBar(
        title: Text(
          widget.mode == SignInMode.addAccount
              ? l10n.addAccountTitle
              : l10n.signInTitle,
        ),
      ),
      body: Padding(
        padding: EdgeInsets.all(spacing.sp5),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (widget.mode == SignInMode.addAccount) ...[
              Text(
                l10n.addAccountDescription,
                style: Theme.of(context).textTheme.bodyLarge,
              ),
              SizedBox(height: spacing.sp5),
            ],
            BrandTextField(
              label: l10n.signInHandleLabel,
              hintText: 'alice.bsky.social',
              controller: _controller,
              enabled: canAddAccount && !busy,
              onSubmitted: (_) => _submit(),
            ),
            SizedBox(height: spacing.sp5),
            ChunkyButton(
              onPressed: canAddAccount && !busy ? _submit : null,
              child: busy
                  ? const StitchProgressIndicator(size: 18)
                  : Text(l10n.signInContinueAction),
            ),
            if (widget.mode == SignInMode.addAccount) ...[
              SizedBox(height: spacing.sp5),
              RegistrationAction(
                enabled: canAddAccount,
                isLoading: busy,
                onPressed: _startRegistration,
              ),
            ],
          ],
        ),
      ),
    );
  }

  void _submit() {
    unawaited(
      ref
          .read(authControllerProvider.notifier)
          .signIn(handle: _controller.text),
    );
  }

  void _startRegistration() {
    unawaited(
      ref.read(authControllerProvider.notifier).startRegistration(),
    );
  }

  String _messageFor(AppLocalizations l10n, Object? error) => switch (error) {
    HandleRequired() => l10n.signInHandleRequiredError,
    InvalidHandle() => l10n.signInInvalidHandleError,
    ServerUnavailable() => l10n.signInServerUnavailableError,
    BrowserLaunchFailed() => l10n.signInBrowserLaunchError,
    RegistrationFailure.canceled => l10n.authRegistrationCanceledError,
    RegistrationFailure.providerUnavailable =>
      l10n.authRegistrationProviderUnavailableError,
    RegistrationFailure.registrationIncomplete =>
      l10n.authRegistrationIncompleteError,
    model.AccountLimitReached() => l10n.accountSwitcherMaximum,
    _ => l10n.signInGenericError,
  };
}
